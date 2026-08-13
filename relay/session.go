package relay

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/emersion/go-message"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/utils"
)

type Session struct {
	backend  *Backend
	conn     *smtp.Conn
	username string

	context context.Context

	From Rcpt
	To   []Rcpt
}

func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *Session) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		ip := s.conn.Conn().RemoteAddr().(*net.TCPAddr).IP
		for d, cfg := range s.backend.Domains {
			if cfg.VerifyUser(ip, d, username, password) {
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
				s.conn.Reject()
				return nil
			}
			return smtp.ErrAuthFailed
		}
		l = l.With("user", username)
		s.context = utils.WithLogger(s.context, l)
		l.Debug("client connected")
		return nil
	}), nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	parsed := ParseAddress(from)
	s.From = Rcpt{
		User:    parsed[0],
		Domain:  parsed[1],
		Address: from,
	}
	_, s.From.Local = s.backend.Domains[s.From.Domain]
	if s.From.Local && s.username != from {
		return smtp.ErrAuthRequired
	}
	return nil
}

func (s *Session) rcpt(to string, opts *smtp.RcptOptions) (Rcpt, error) {
	too := ParseAddress(to)
	rcpt := Rcpt{
		User:    too[0],
		Domain:  too[1],
		Address: to,
	}
	var cfg auth.Config
	cfg, rcpt.Local = s.backend.Domains[rcpt.Domain]
	// from local to outside
	if s.From.Local && !rcpt.Local {
		return rcpt, nil
	}
	// not to local
	if !rcpt.Local {
		return rcpt, &smtp.SMTPError{
			Code:         551,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Forwarding to remote hosts is disabled",
		}
	}
	// to local
	data := cfg.Exists(s.From.Address, rcpt.User)
	if data == nil {
		return rcpt, &smtp.SMTPError{
			Code:         550,
			EnhancedCode: [3]int{5, 1, 1},
			Message:      "Address doesn't exist",
		}
	}
	switch {
	case slices.Contains(auth.AdminEmails, data.Username):
		rcpt.User = cfg.Admin.User
		rcpt.Folder = cfg.Admin.Folder
		rcpt.Key = cfg.Admin.Crypto.GetKey()
		rcpt.Address = cfg.Admin.User + "@" + rcpt.Domain
	case data.Group != nil:
		rcpt.Group = data.Group
	default:
		rcpt.User = data.Username
		rcpt.Folder = data.Folder
		rcpt.Key = data.Key
	}
	return rcpt, nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	rcpt, err := s.rcpt(to, opts)
	if err != nil {
		return err
	}
	s.To = append(s.To, rcpt)
	return nil
}

var errInternal = &smtp.SMTPError{
	Code:    451,
	Message: "Internal error",
}

func (s *Session) Data(r io.Reader) error {
	l := utils.Logger(s.context).With(
		"from", s.From.Address,
		"to", strings.Join(
			utils.Map(s.To, func(r Rcpt) string { return r.Address }),
			",",
		),
	)
	msg, err := message.Read(r)
	if err != nil {
		l.Error("reading data", "error", err)
		return errInternal
	}
	var score *float64
	var wait time.Duration
	if s.backend.Rspamd != nil {
		var sc float64
		sc, wait, err = s.backend.Rspamd.Analyze(s, msg)
		if err != nil {
			if _, ok := errors.AsType[*smtp.SMTPError](err); ok {
				return err
			}
			l.Error("analyzing email with rspamd", "error", err)
			return errInternal
		}
		score = &sc
	}
	var buf bytes.Buffer
	err = msg.WriteTo(&buf)
	if err != nil {
		l.Error("rendering email", "error", err)
		return errInternal
	}
	go func(ctx context.Context, to []Rcpt) {
		if wait != 0 {
			time.Sleep(wait)
		}
		s.send(ctx, s.From.Address, to, buf.Bytes(), msg.Header, score)
	}(s.context, s.To)
	return nil
}

func (s *Session) Reset() {
	s.To = nil
}

func (s *Session) Logout() error {
	return nil
}

func ParseAddress(address string) [2]string {
	mailbox, domain, _ := strings.Cut(address, "@")
	return [2]string{mailbox, domain}
}
