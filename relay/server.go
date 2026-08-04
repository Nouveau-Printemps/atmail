package relay

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/emersion/go-message/textproto"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/auth"
)

type Backend struct {
	Domains   map[string]auth.Config
	Rspamd    *RspamdClient
	Queue     *Queue
	LocalName string

	OnReceive func(user string, id int64)
}

func (bck *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{backend: bck, conn: c}, nil
}

type Session struct {
	backend *Backend
	conn    *smtp.Conn
	authAs  string

	From      [2]string
	To        [2]string
	FromLocal bool
	ToLocal   bool

	RedirectTo string
}

func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *Session) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		for k, cfg := range s.backend.Domains {
			if cfg.VerifyUser(k, username, password) {
				s.authAs = username
			}
		}
		if len(s.authAs) == 0 {
			return smtp.ErrAuthFailed
		}
		slog.Debug("client connected", "ip", s.conn.Conn().RemoteAddr(), "user", username)
		return nil
	}), nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.From = ParseAddress(from)
	_, s.FromLocal = s.backend.Domains[s.From[1]]
	if s.FromLocal && s.authAs != from {
		return smtp.ErrAuthRequired
	}
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.To = ParseAddress(to)
	var cfg auth.Config
	cfg, s.ToLocal = s.backend.Domains[s.To[1]]
	// from local to outside
	if s.FromLocal && !s.ToLocal {
		return nil
	}
	// not to local
	if !s.ToLocal {
		return &smtp.SMTPError{
			Code:         551,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Forwarding to remote hosts is disabled",
		}
	}
	// to local
	if cfg.Static != nil {
		if _, ok := cfg.Static.Users[s.To[0]]; !ok {
			return &smtp.SMTPError{
				Code:         550,
				EnhancedCode: [3]int{5, 1, 1},
				Message:      "Address doesn't exist",
			}
		}
	} else if cfg.CatchAll != nil {
		s.RedirectTo = cfg.CatchAll.User
	}
	return nil
}

var errInternal = &smtp.SMTPError{
	Code:    451,
	Message: "Internal error",
}

func (s *Session) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		slog.Error(
			"reading data",
			"error", err,
			"from", strings.Join(s.From[:], "@"),
			"to", strings.Join(s.To[:], "@"),
		)
		return errInternal
	}
	buf := bufio.NewReader(bytes.NewBuffer(b))
	h, err := textproto.ReadHeader(buf)
	if err != nil {
		slog.Error(
			"parsing mail",
			"error", err,
			"from", strings.Join(s.From[:], "@"),
			"to", strings.Join(s.To[:], "@"),
		)
		return errInternal
	}
	body, _ := io.ReadAll(buf)
	var spam *RspamdResponse
	if s.backend.Rspamd == nil {
		// this goto avoids an hugly condition that is almost always met
		// thus, this goto produces a faster assembly because there is no bad branch prediction in the common case
		goto valid_email
	}
	spam, err = s.backend.Rspamd.Verify(context.TODO(), nil, b)
	if err != nil {
		slog.Error(
			"rspamd check",
			"error", err,
			"from", strings.Join(s.From[:], "@"),
			"to", strings.Join(s.To[:], "@"),
		)
		return errInternal
	}
	if spam.Skipped {
		// see above
		goto valid_email
	}
	if spam.Messages != nil {
		v, ok := spam.Messages["spam_message"]
		if ok {
			return &smtp.SMTPError{
				Code:         550,
				EnhancedCode: [3]int{5, 7, 1},
				Message:      v,
			}
		}
	}
	switch spam.Action {
	case RejectResponse:
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Your message is unwanted",
		}
	case SoftRejectResponse:
		return &smtp.SMTPError{
			Code:         450,
			EnhancedCode: [3]int{4, 7, 1},
			Message:      "Your message is temporary unwanted, retry later",
		}
	case AddHeaderResponse:
		h.Add("X-Spam-Score", strconv.FormatFloat(spam.Score, 'f', 2, 64))
	case RewriteSubjectResponse:
		h.Add("Subject", spam.Subject)
	case GreylistResponse:
	case NoActionResponse:
	default:
		panic("not implemented")
	}
	if spam.Body != nil {
		body, _ = io.ReadAll(spam.Body)
	}
valid_email:
	user := s.RedirectTo + "@" + s.To[1]
	if s.RedirectTo == "" {
		user = strings.Join(s.To[:], "@")
	}
	if s.ToLocal {
		go s.relayInside(user, body, h, spam)
	} else {
		s.backend.Queue.Enqueue(
			strings.Join(s.From[:], "@"),
			strings.Join(s.To[:], "@"),
			s.To[1],
			body,
		)
	}
	return nil
}

func (s *Session) Reset() {
}

func (s *Session) Logout() error {
	*s = Session{backend: s.backend}
	return nil
}

func ParseAddress(address string) [2]string {
	mailbox, domain, _ := strings.Cut(address, "@")
	return [2]string{mailbox, domain}
}

func formatMail(headers map[string][]string, body []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(headers) + len(body))
	for k, arr := range headers {
		for _, v := range arr {
			buf.WriteString("\r\n")
			buf.WriteString(k)
			buf.WriteString(": ")
			buf.WriteString(v)
		}
	}
	if len(headers) > 0 {
		buf.WriteString("\r\n\r\n")
	}
	buf.Grow(len(body))
	buf.Write(body)
	return buf.Bytes()[2:]
}
