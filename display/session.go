package display

import (
	"context"
	"log/syslog"
	"net"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/nyttikord/logos"
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

	mailboxes *Boxes

	username string

	selected *mailbox.View
	readOnly bool

	subscribed map[string]struct{}
}

func (s *Session) Login(username, password string) error {
	ip := s.conn.NetConn().RemoteAddr().(*net.TCPAddr).IP
	for d, cfg := range s.backend.Domains {
		if cfg.VerifyUser(ip, d, username, password) {
			s.username = username
			break
		}
	}
	l := utils.Logger(s.context)
	if len(s.username) == 0 {
		l.InfoContext(
			logos.NewSyslogContext(s.context, syslog.LOG_AUTH),
			"login failed",
			"user", username,
		)
		if s.backend.RateLimiter.Limit(
			utils.WithLogger(s.context, l.With("module", "rate-limiter")),
			ip,
		) {
			s.conn.Bye("rate limited")
			return nil
		}
		l.Debug("auth failed", "user", username)
		return imapserver.ErrAuthFailed
	}
	s.backend.RateLimiter.Accept(s.context, ip)
	l = l.With("user", username)
	s.context = utils.WithLogger(s.context, l)
	l.InfoContext(logos.NewSyslogContext(
		s.context,
		syslog.LOG_AUTH,
	), "client connected")
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
	mp := make(map[string]*mailbox.View, len(box))
	for _, b := range box {
		mp[b.Name] = mailbox.NewView(uint32(b.ID), b.Name, uint32(b.Count))
	}
	s.mailboxes = &Boxes{mp: mp}
	s.backend.SetUserBoxes(username, s.mailboxes)
	return nil
}
