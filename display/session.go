package display

import (
	"context"

	"github.com/emersion/go-imap/v2/imapserver"
	"nouveauprintemps.org/atmail/mailbox"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/utils"
)

type Session struct {
	backend *Backend
	conn    *imapserver.Conn

	context context.Context

	mailboxes map[string]*mailbox.View

	username string

	selected *mailbox.View
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
	l := utils.Logger(s.context).With("user", username)
	l.Debug("client connected", "ip", s.conn.NetConn().RemoteAddr())
	s.context = utils.WithLogger(s.context, l)
	boxes, ok := s.backend.GetUserBoxes(s.username)
	if ok {
		s.mailboxes = boxes
		return nil
	}
	box, err := storage.LoadMailbox(s.context, username)
	if err != nil {
		return err
	}
	s.mailboxes = make(map[string]*mailbox.View, len(box))
	for _, b := range box {
		s.mailboxes[b.Name] = mailbox.NewView(uint32(b.ID), b.Name)
	}
	s.backend.SetUserBoxes(username, s.mailboxes)
	return nil
}
