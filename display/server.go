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
)

type logger struct {
	*slog.Logger
}

func (l *logger) Printf(format string, args ...any) {
	ctx := logos.NewContext(context.Background(), 1, true, false)
	log, _, _ := strings.Cut(fmt.Sprintf(format, args...), "\n")
	l.Logger.ErrorContext(ctx, log)
}

type Backend struct {
	Domains     map[string]auth.Config
	MaxMailSize uint32
	mailboxes   map[string]map[string]*mailbox.View
	muBoxes     sync.RWMutex
}

func (bck *Backend) Options(log *slog.Logger) *imapserver.Options {
	bck.mailboxes = make(map[string]map[string]*mailbox.View, 2)
	return &imapserver.Options{
		NewSession: bck.NewSession,
		Caps: imap.CapSet{
			imap.CapIMAP4rev1: {},
			imap.CapIMAP4rev2: {},
		},
		Logger:       &logger{log},
		InsecureAuth: true,
	}
}

func (bck *Backend) NewSession(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
	return &Session{
		backend:    bck,
		conn:       conn,
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
