package relay

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/utils"
)

type emailEnqueued struct {
	From     string
	To       string
	Domain   string
	Body     []byte
	MustWait time.Duration
}

const MaxDuration = 15 * time.Minute

type Queue struct {
	sender chan emailEnqueued
}

func NewQueue() *Queue {
	return &Queue{sender: make(chan emailEnqueued, 2)}
}

func (q *Queue) Enqueue(from, to, domain string, body []byte) {
	go func() {
		q.sender <- emailEnqueued{
			From:   from,
			To:     to,
			Domain: domain,
			Body:   body,
		}
	}()
}

func (q *Queue) Loop(ctx context.Context, b *Backend) {
	l := utils.Logger(ctx)
	for {
		select {
		case email := <-q.sender:
			l := l.With("email", email)
			err := b.relayOutside(
				email.From,
				email.To,
				email.Domain,
				email.Body,
			)
			if err == nil {
				continue
			}
			l.Debug("sending mail", "error", err)
			if e, ok := errors.AsType[*net.DNSError](err); ok {
				if e.IsNotFound {
					l.Debug("invalid DNS records", "missing record", e.Name, "DNS server", e.Server)
					//TODO: send fail
					continue
				}
			} else if e, ok := errors.AsType[*smtp.SMTPError](err); ok {
				if e.Code >= 500 {
					l.Warn("permanent SMTP error", "code", e.Code, "message", e.Message)
					//TODO: send fail
					continue
				}
			} else {
				l.Error("unknown reason")
				//TODO: send fail
				continue
			}
			go func() {
				if email.MustWait-MaxDuration > 0 {
					l.Warn("sending failed")
					//TODO: send fail
					return
				}
				email.MustWait = (email.MustWait + 2*time.Second) * (email.MustWait + 500*time.Millisecond)
				l.Debug("retrying later", "wait", email.MustWait)
				time.Sleep(email.MustWait)
				q.sender <- email
			}()
		case <-ctx.Done():
			return
		}
	}
}
