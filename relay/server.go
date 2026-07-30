package relay

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/emersion/go-message/textproto"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/auth"
	"nouveauprintemps.org/atmail/storage"
)

type Backend struct {
	Domains map[string]auth.Config
	Rspamd  *RspamdClient
}

func (bck *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{backend: bck}, nil
}

type Session struct {
	backend *Backend
	authFor []string

	From [2]string
	To   [2]string

	RedirectTo string
}

func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *Session) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		for k, cfg := range s.backend.Domains {
			if cfg.VerifyUser(k, username, password) {
				s.authFor = append(s.authFor, k)
			}
		}
		if len(s.authFor) == 0 {
			return smtp.ErrAuthFailed
		}
		return nil
	}), nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	a := ParseAddress(from)
	if _, ok := s.backend.Domains[a[1]]; ok && len(s.authFor) == 0 {
		return smtp.ErrAuthRequired
	}
	s.From = a
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	a := ParseAddress(to)
	cfg, ok := s.backend.Domains[a[1]]
	if !ok {
		return &smtp.SMTPError{
			Code:         551,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Forwarding to remote hosts is disabled",
		}
	}
	if cfg.Static != nil {
		if _, ok := cfg.Static.Users[a[0]]; !ok {
			return &smtp.SMTPError{
				Code:         550,
				EnhancedCode: [3]int{5, 1, 1},
				Message:      "Address doesn't exist",
			}
		}
	} else if cfg.CatchAll != nil {
		s.RedirectTo = cfg.CatchAll.User
	}
	s.To = a
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
	if s.RedirectTo != "" {
		s.To[0] = s.RedirectTo
	}
	go func() {
		score := sql.NullFloat64{Float64: 0, Valid: false}
		if spam != nil {
			score.Float64 = spam.Score
			score.Valid = true
		}
		b := formatMail(h.Map(), body)
		err := storage.StoreEmailInbox(context.TODO(), s.From, s.To, score, b)
		if err != nil {
			slog.Error("cannot save email", "error", err)
		}
	}()
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
