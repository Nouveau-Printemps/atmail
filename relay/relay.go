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
	"time"

	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/utils"
)

func (s *Session) relayInside(ctx context.Context, rcpt Rcpt, b []byte, h message.Header, spam *float64) {
	l := utils.Logger(ctx).With("to", rcpt.Address)
	var score sql.NullFloat64
	if spam != nil {
		score.Float64 = *spam
		score.Valid = true
	}
	var encrypted bool
	if rcpt.Key != nil {
		// decode before encrypting
		switch h.Get("Content-Transfer-Encoding") {
		case "quoted-printable":
			b, _ = io.ReadAll(quotedprintable.NewReader(bytes.NewBuffer(b)))
			h.Set("Content-Transfer-Encoding", "8bit")
		case "base64":
			b, _ = io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(b)))
			h.Set("Content-Transfer-Encoding", "8bit")
		}
		body, err := auth.EncryptEmail(*rcpt.Key, b)
		if err != nil {
			l.Error("cannot encrypt email", "error", err)
		} else {
			b = body
			encrypted = true
		}
	}
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
	l.Debug("email saved")
	if s.backend.OnReceive != nil {
		s.backend.OnReceive(rcpt.Address, rcpt.Folder, id)
	}
}

func (b *Backend) relayOutside(ctx context.Context, from string, to []string, domain string, body []byte) error {
	l := utils.Logger(ctx).With("domain", domain, "from", from, "to", to)
	relays, err := relaysOf(domain)
	if err != nil {
		return err
	}
	var accErr error
	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}
	tlsDialer := tls.Dialer{NetDialer: &dialer}
	for host, err := range relays {
		if err == nil {
			l := l.With("host", host)
			// wrapping in anonymous function call to use defer
			err = func() error {
				var conn net.Conn
				if host.withTLS {
					conn, err = tlsDialer.DialContext(ctx, "tcp", host.address)
				} else {
					conn, err = dialer.DialContext(ctx, "tcp", host.address)
				}
				if err != nil {
					return err
				}
				defer conn.Close()
				client := smtp.NewClient(conn)
				if ok, _ := client.Extension("STARTTLS"); ok {
					l.Debug("using STARTTLS")
					client, err = smtp.NewClientStartTLS(conn, tlsDialer.Config)
					if err != nil {
						return err
					}
				}
				defer client.Close()
				err = client.Hello(b.LocalName)
				if err != nil {
					return err
				}
				return client.SendMail(from, to, bytes.NewBuffer(body))
			}()
			if err == nil {
				l.Debug("email sent")
				return nil
			}
		}
		if accErr != nil {
			accErr = errors.Join(accErr, err)
		} else {
			accErr = err
		}
	}
	return accErr
}

type relay struct {
	address string
	withTLS bool
}

// Single-use iterator.
func relaysOf(domain string) (iter.Seq2[relay, error], error) {
	mxs, err := net.LookupMX(domain)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(mxs, func(a, b *net.MX) int {
		return int(b.Pref) - int(a.Pref)
	})
	return func(yield func(relay, error) bool) {
		for _, mx := range mxs {
			_, srvs, err := net.LookupSRV("submission", "tcp", mx.Host)
			if err != nil {
				if e, ok := errors.AsType[*net.DNSError](err); !ok || !e.IsNotFound {
					if !yield(relay{}, err) {
						return
					}
					continue
				}
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
				slices.SortFunc(srvs, func(a, b *net.SRV) int {
					res := int(b.Priority) - int(a.Priority)
					if res != 0 {
						return res
					}
					return int(b.Weight) - int(a.Weight)
				})
			}
			for _, srv := range srvs {
				ips, err := net.LookupIP(srv.Target)
				if err != nil {
					if !yield(relay{}, err) {
						return
					}
					continue
				}
				for _, ip := range ips {
					l := slog.With("domain", domain, "target", srv.Target)
					ip, err := netip.ParseAddr(ip.String())
					if err != nil {
						l.Warn("invalid srv record", "error", err)
						continue
					}
					if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() {
						l.Warn("invalid srv record", "error", "invalid IP")
						continue
					}
					addr := netip.AddrPortFrom(ip, srv.Port)
					if !yield(relay{address: addr.String(), withTLS: srv.Port == 465}, nil) {
						return
					}
				}
			}
		}
	}, nil
}
