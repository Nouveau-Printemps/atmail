package relay

import (
	"bytes"
	"context"
	"database/sql"
	"log/slog"
	"net"
	"slices"

	"github.com/emersion/go-message/textproto"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/storage"
)

func (s *Session) relayInside(user string, body []byte, h textproto.Header, spam *RspamdResponse) {
	score := sql.NullFloat64{Float64: 0, Valid: false}
	if spam != nil {
		score.Float64 = spam.Score
		score.Valid = true
	}
	b := formatMail(h.Map(), body)
	id, err := storage.StoreEmailInbox(context.TODO(), s.From, s.To, user, score, b)
	if err != nil {
		slog.Error("cannot save email", "error", err)
		return
	}
	if s.backend.OnReceive != nil {
		s.backend.OnReceive(user, id)
	}
}

func relayOutside(from, to, domain string, body []byte) {
	mxs, err := net.LookupMX(domain)
	l := slog.With("domain", domain, "from", from, "to", to)
	if err != nil {
		l.Error("looking up MX record", "error", err)
		return
	}
	slices.SortFunc(mxs, func(a, b *net.MX) int {
		return int(b.Pref) - int(a.Pref)
	})
	for _, mx := range mxs {
		l := l.With("mx", mx.Host)
		next := true
		// wrapping in anonymous function call to use defer
		func() {
			conn, err := net.Dial("tcp", mx.Host)
			if err != nil {
				l.Error("connecting to SMTP relay", "error", err)
				return
			}
			defer conn.Close()
			client := smtp.NewClient(conn)
			err = client.Hello(domain)
			if err != nil {
				l.Error("connecting to SMTP relay", "error", err)
				return
			}
			defer client.Quit()
			err = client.SendMail(from, []string{to}, bytes.NewBuffer(body))
			if err != nil {
				l.Error("sending email", "error", err)
				return
			}
			l.Debug("mail sent")
			next = false
		}()
		if !next {
			return
		}
	}
}
