package display

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/nyttikord/logos"
	"nouveauprintemps.org/atmail/auth"
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
	Mailboxes   map[string]map[string]MailboxView
}

func (bck *Backend) Options(log *slog.Logger) *imapserver.Options {
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
