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
	To       []string
	Domain   string
	Body     []byte
	MustWait time.Duration
}

const MaxDuration = 15 * 60

type Queue struct {
	sender chan emailEnqueued
	ok     chan struct{}
}

func NewQueue(concurrent uint8) *Queue {
	ok := make(chan struct{}, concurrent)
	for range concurrent {
		ok <- struct{}{}
	}
	return &Queue{sender: make(chan emailEnqueued, 2), ok: ok}
}

func (q *Queue) Enqueue(from string, to []string, domain string, body []byte) {
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
		case <-q.ok:
			l.Debug("new sender available")
			select {
			case email := <-q.sender:
				go q.send(ctx, b, email)
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (q *Queue) send(ctx context.Context, b *Backend, email emailEnqueued) {
	defer func() {
		q.ok <- struct{}{}
	}()
	l := utils.Logger(ctx).With("email", email)
	err := b.relayOutside(
		utils.WithLogger(ctx, l),
		email.From,
		email.To,
		email.Domain,
		email.Body,
	)
	if err == nil {
		return
	}
	l.Debug("cannot send email")
	if e, ok := errors.AsType[*net.DNSError](err); ok {
		if e.IsNotFound {
			l.Debug("invalid DNS records", "missing record", e.Name, "DNS server", e.Server)
			//TODO: send fail
			return
		}
	} else if e, ok := errors.AsType[*smtp.SMTPError](err); ok {
		if e.Code >= 500 {
			l.Warn("permanent SMTP error", "code", e.Code, "message", e.Message)
			//TODO: send fail
			return
		}
	} else {
		l.Error("unknown reason", "error", err)
		//TODO: send fail
		return
	}
	go func() {
		if email.MustWait-MaxDuration > 0 {
			l.Warn("sending failed")
			//TODO: send fail
			return
		}
		email.MustWait = max(email.MustWait, 3)
		email.MustWait += email.MustWait / 2
		l.Debug("retrying later", "wait", email.MustWait*time.Second)
		time.Sleep(email.MustWait * time.Second)
		q.sender <- email
	}()
}
