package relay

import (
	"context"

	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/auth"
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
}

func (bck *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	l := utils.Logger(bck.Context).With("addr", c.Conn().RemoteAddr())
	l.Debug("new session")
	return &Session{backend: bck, conn: c, context: utils.WithLogger(bck.Context, l)}, nil
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
