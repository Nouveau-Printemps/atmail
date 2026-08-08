package relay

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"iter"
	"log/slog"
	"mime/quotedprintable"
	"net"
	"net/netip"
	"slices"

	"github.com/emersion/go-message/textproto"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/utils"
)

func (s *Session) relayInside(ctx context.Context, rcpt Rcpt, body []byte, h textproto.Header, spam *RspamdResponse) {
	l := utils.Logger(ctx).With("to", rcpt.Address)
	score := sql.NullFloat64{Float64: 0, Valid: false}
	if spam != nil {
		score.Float64 = spam.Score
		score.Valid = true
	}
	var encrypted bool
	if rcpt.Key != nil {
		// decode before encrypting
		switch h.Get("Content-Transfer-Encoding") {
		case "quoted-printable":
			body, _ = io.ReadAll(quotedprintable.NewReader(bytes.NewBuffer(body)))
			h.Set("Content-Transfer-Encoding", "8bit")
		case "base64":
			body, _ = io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(body)))
			h.Set("Content-Transfer-Encoding", "8bit")
		}
		b, err := auth.EncryptEmail(*rcpt.Key, body)
		if err != nil {
			l.Error("cannot encrypt email", "error", err)
		} else {
			body = b
			encrypted = true
		}
	}
	b := formatMail(h.Map(), body)
	id, err := storage.StoreEmailInbox(
		ctx,
		s.From, [2]string{rcpt.User, rcpt.Domain},
		rcpt.Address,
		score,
		b,
		rcpt.Folder,
		encrypted,
	)
	if err != nil {
		l.Error("cannot save email", "error", err)
		return
	}
	if s.backend.OnReceive != nil {
		s.backend.OnReceive(rcpt.Address, rcpt.Folder, id)
	}
}

func (b *Backend) relayOutside(from, to, domain string, body []byte) error {
	l := slog.With("domain", domain, "from", from, "to", to)
	relays, err := relaysOf(domain)
	if err != nil {
		return err
	}
	var accErr error
	for host, err := range relays {
		if err == nil {
			l := l.With("host", host)
			// wrapping in anonymous function call to use defer
			err = func() error {
				var conn net.Conn
				var client *smtp.Client
				conn, err = tls.Dial("tcp", host, nil)
				if err != nil {
					l.Warn("dialing with tls to relay")
					conn, err = net.Dial("tcp", host)
				}
				defer conn.Close()
				client = smtp.NewClient(conn)
				if ok, _ := client.Extension("STARTTLS"); ok {
					l.Debug("using STARTTLS")
					client, err = smtp.NewClientStartTLS(conn, nil)
					if err != nil {
						return err
					}
				}
				defer client.Close()
				err = client.Hello(b.LocalName)
				if err != nil {
					return err
				}
				return client.SendMail(from, []string{to}, bytes.NewBuffer(body))
			}()
		}
		if err == nil {
			return nil
		}
		if accErr != nil {
			accErr = errors.Join(accErr, err)
		} else {
			accErr = err
		}
	}
	return accErr
}

// Single-use iterator.
func relaysOf(domain string) (iter.Seq2[string, error], error) {
	mxs, err := net.LookupMX(domain)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(mxs, func(a, b *net.MX) int {
		return int(b.Pref) - int(a.Pref)
	})
	return func(yield func(string, error) bool) {
		for _, mx := range mxs {
			_, srvs, err := net.LookupSRV("submission", "tcp", mx.Host)
			if err != nil {
				if e, ok := errors.AsType[*net.DNSError](err); ok && e.IsNotFound {
					slog.Debug("no srv record, fallback to standard ports", "mx", mx.Host)
					srvs = []*net.SRV{
						{
							Target: mx.Host,
							Port:   465,
						},
						{
							Target: mx.Host,
							Port:   587,
						},
						{
							Target: mx.Host,
							Port:   25,
						},
						// not standard but common
						{
							Target: mx.Host,
							Port:   2525,
						},
					}
				} else {
					if !yield("", err) {
						return
					}
					continue
				}
			} else {
				slices.SortFunc(srvs, func(a, b *net.SRV) int {
					res := int(b.Priority) - int(a.Priority)
					if res != 0 {
						return res
					}
					return int(b.Weight) - int(a.Weight)
				})
			}
			for _, srv := range srvs {
				ip := netip.MustParseAddr(srv.Target)
				if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() {
					slog.Warn("invalid srv record", "domain", domain, "target", srv.Target)
					continue
				}
				addr := netip.AddrPortFrom(ip, srv.Port)
				if !yield(addr.String(), nil) {
					return
				}
			}
		}
	}, nil
}
