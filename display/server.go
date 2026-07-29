package display

import (
	"fmt"
	"log/slog"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"nouveauprintemps.org/atmail/auth"
)

type logger struct {
	*slog.Logger
}

func (l *logger) Printf(format string, args ...any) {
	l.Logger.Error(fmt.Sprintf(format, args...))
}

type Backend struct {
	Domains     map[string]auth.Config
	MaxMailSize uint32
	Mailboxes   map[string]map[string]MailboxView
}

func (bck *Backend) Options(log *slog.Logger) *imapserver.Options {
	return &imapserver.Options{
		NewSession: bck.NewSession,
		Caps: imap.CapSet{
			imap.CapIMAP4rev2: struct{}{},
		},
		Logger:       &logger{log},
		InsecureAuth: true,
	}
}

func (bck *Backend) NewSession(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
	return &Session{backend: bck, subscribed: map[string]struct{}{}}, &imapserver.GreetingData{PreAuth: false}, nil
}
