package relay

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/emersion/go-smtp"
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
	for {
		select {
		case email := <-q.sender:
			err := b.relayOutside(
				email.From,
				email.To,
				email.Domain,
				email.Body,
			)
			if err == nil {
				continue
			}
			slog.Warn("sending mail", "error", err, "email", email)
			if e, ok := errors.AsType[*net.DNSError](err); ok {
				if e.IsNotFound {
					//TODO: send fail
					continue
				}
			} else if e, ok := errors.AsType[*smtp.SMTPError](err); ok {
				if e.Code >= 500 {
					//TODO: send fail
					continue
				}
			} else {
				//TODO: send fail
				continue
			}
			go func() {
				if email.MustWait-MaxDuration > 0 {
					//TODO: send fail
					return
				}
				email.MustWait = (email.MustWait + 2*time.Second) * (email.MustWait + 500*time.Millisecond)
				time.Sleep(email.MustWait)
				q.sender <- email
			}()
		case <-ctx.Done():
			return
		}
	}
}
