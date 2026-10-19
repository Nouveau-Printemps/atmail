package relay

import (
	"bytes"
	"context"
	"io"

	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/filter"
	"nouveauprintemps.org/atmail/utils"
)

type Backend struct {
	Domains   map[string]auth.Config
	Rspamd    *RspamdClient
	Queue     *Queue
	LocalName string

	Context context.Context

	OnReceive   func(user, mailbox string, id int64)
	RateLimiter *auth.RateLimiter

	Filters []*filter.Tree
}

func (bck *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	l := utils.Logger(bck.Context).With("addr", c.Conn().RemoteAddr())
	l.Debug("new session")
	return &Session{backend: bck, conn: c, context: utils.WithLogger(bck.Context, l)}, nil
}

func (bck *Backend) NewContext(from, to Rcpt, mail *message.Entity) (*filter.EvaluationContext, error) {
	b, err := io.ReadAll(mail.Body)
	if err != nil {
		return nil, err
	}
	mail.Body = bytes.NewBuffer(b)
	return filter.InitEvalulationContext(
		"",
		from.Rcpt(),
		to.Rcpt(),
		mail.Header,
		string(b),
	), nil
}

type Rcpt struct {
	User    string
	Domain  string
	Folder  string
	Address string
	Key     *string
	Local   bool
	Group   *auth.Group
}

func (rcpt Rcpt) Rcpt() filter.Rcpt {
	return filter.Rcpt{
		User:    rcpt.User,
		Domain:  rcpt.Domain,
		Folder:  rcpt.Folder,
		Address: rcpt.Address,
	}
}
