package display

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/nyttikord/logos"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/mailbox"
	"nouveauprintemps.org/atmail/utils"
)

type logger struct {
	*slog.Logger
}

func (l *logger) Printf(format string, args ...any) {
	content := fmt.Sprintf(format, args...)
	if strings.Contains(content, "i/o timeout") {
		return
	}
	ctx := logos.NewContext(context.Background(), 1, false, false)
	log, _, _ := strings.Cut(content, "\n")
	l.Logger.ErrorContext(ctx, log)
}

type Backend struct {
	Context      context.Context
	Domains      map[string]auth.Config
	MaxEmailSize uint32
	RateLimiter  *auth.RateLimiter
	mailboxes    map[string]map[string]*mailbox.View
	muBoxes      sync.RWMutex
}

func (bck *Backend) Options(insecureAuth bool) *imapserver.Options {
	bck.mailboxes = make(map[string]map[string]*mailbox.View, 2)
	return &imapserver.Options{
		NewSession: bck.NewSession,
		Caps: imap.CapSet{
			imap.CapIMAP4rev1: {},
			imap.CapIMAP4rev2: {},
		},
		Logger:       &logger{utils.Logger(bck.Context)},
		InsecureAuth: insecureAuth,
	}
}

func (bck *Backend) NewSession(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
	l := utils.Logger(bck.Context).With("addr", conn.NetConn().RemoteAddr())
	l.Debug("new session")
	return &Session{
		backend:    bck,
		conn:       conn,
		context:    utils.WithLogger(bck.Context, l),
		subscribed: map[string]struct{}{},
	}, &imapserver.GreetingData{PreAuth: false}, nil
}

func (bck *Backend) GetUserBoxes(user string) (map[string]*mailbox.View, bool) {
	bck.muBoxes.RLock()
	defer bck.muBoxes.RUnlock()
	bxs, ok := bck.mailboxes[user]
	return bxs, ok
}

func (bck *Backend) SetUserBoxes(user string, boxes map[string]*mailbox.View) {
	bck.muBoxes.Lock()
	defer bck.muBoxes.Unlock()
	bck.mailboxes[user] = boxes
}
