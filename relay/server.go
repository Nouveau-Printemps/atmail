package relay

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/filter"
	"nouveauprintemps.org/atmail/utils"
)

type Backend struct {
	Domains   map[string]auth.Config
	rspamd    *RspamdClient
	Queue     *Queue
	LocalName string

	// because go-smtp doesn't use context in functions
	context context.Context

	OnReceive   func(user, mailbox string, id int64)
	rateLimiter *auth.RateLimiter

	Filters []*filter.Tree

	relayCache map[string]relay
	muCache    sync.RWMutex
}

func NewBackend(
	ctx context.Context,
	domains map[string]auth.Config,
	rspamd *string,
	localName string,
	rl *auth.RateLimiter,
	filters []*filter.Tree,
	concurrentSenders uint8,
) *Backend {
	bck := &Backend{
		Domains:     domains,
		Queue:       NewQueue(concurrentSenders),
		LocalName:   localName,
		context:     ctx,
		rateLimiter: rl,
		Filters:     filters,
		relayCache:  make(map[string]relay),
	}
	if rspamd != nil {
		bck.rspamd = &RspamdClient{
			Client: http.DefaultClient,
			URL:    *rspamd,
		}
	}
	return bck
}

func (bck *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	l := utils.Logger(bck.context).With("addr", c.Conn().RemoteAddr())
	l.Debug("new session")
	return &Session{backend: bck, conn: c, context: utils.WithLogger(bck.context, l)}, nil
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
