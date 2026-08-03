package display

import (
	"context"
	"log/slog"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"nouveauprintemps.org/atmail/storage"
)

type MailboxView struct {
	*imapserver.MailboxTracker
	UIDValidity imap.UID
}

type MailboxSelectedView struct {
	*imapserver.SessionTracker
	ID   imap.UID
	Name string
}

type Session struct {
	backend *Backend
	conn    *imapserver.Conn

	mailboxes map[string]MailboxView

	username string

	selected *MailboxSelectedView
	readOnly bool

	subscribed map[string]struct{}
}

func (s *Session) Login(username, password string) error {
	for d, cfg := range s.backend.Domains {
		if cfg.VerifyUser(d, username, password) {
			s.username = username
			break
		}
	}
	if len(s.username) == 0 {
		return imapserver.ErrAuthFailed
	}
	slog.Debug("client connected", "ip", s.conn.NetConn().RemoteAddr(), "user", username)
	boxes, ok := s.backend.GetUserBoxes(s.username)
	if ok {
		s.mailboxes = boxes
		return nil
	}
	box, err := storage.LoadMailbox(context.TODO(), username)
	if err != nil {
		return err
	}
	s.mailboxes = make(map[string]MailboxView, len(box))
	for _, b := range box {
		view := MailboxView{
			MailboxTracker: imapserver.NewMailboxTracker(uint32(b.Count)),
			UIDValidity:    imap.UID(b.ID),
		}
		s.mailboxes[b.Name] = view
	}
	s.backend.SetUserBoxes(username, s.mailboxes)
	return nil
}
