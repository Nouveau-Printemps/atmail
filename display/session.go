package display

import (
	"context"
	"net"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"nouveauprintemps.org/atmail/mailbox"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/utils"
)

var errInternal = &imap.Error{
	Type: imap.StatusResponseTypeNo,
	Code: imap.ResponseCodeServerBug,
	Text: "Internal error",
}

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
	ip := s.conn.NetConn().RemoteAddr().(*net.TCPAddr).IP
	for d, cfg := range s.backend.Domains {
		ok := cfg.VerifyUser(ip, d, username, password)
		if ok {
			s.username = username
			break
		}
	}
	l := utils.Logger(s.context)
	if len(s.username) == 0 {
		if s.backend.RateLimiter.Limit(
			utils.WithLogger(s.context, l.With("module", "rate-limiter")),
			ip,
		) {
			s.conn.Bye("rate limited")
			return nil
		}
		return imapserver.ErrAuthFailed
	}
	l = l.With("user", username)
	l.Debug("client connected", "ip", s.conn.NetConn().RemoteAddr())
	s.context = utils.WithLogger(s.context, l)
	boxes, ok := s.backend.GetUserBoxes(s.username)
	if ok {
		s.mailboxes = boxes
		return nil
	}
	box, err := storage.LoadMailbox(s.context, username)
	if err != nil {
		l.Error("loading mailbox", "error", err)
		return errInternal
	}
	s.mailboxes = make(map[string]*mailbox.View, len(box))
	for _, b := range box {
		s.mailboxes[b.Name] = mailbox.NewView(uint32(b.ID), b.Name)
	}
	s.backend.SetUserBoxes(username, s.mailboxes)
	return nil
}
