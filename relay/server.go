package relay

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/mail"
	"slices"
	"strconv"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/storage"
)

type Backend struct {
	Domains []string
	Rspamd  *RspamdClient
	Storage *storage.Storage
}

func (bck *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{backend: bck}, nil
}

type Session struct {
	backend *Backend
	auth    bool

	From [2]string
	To   [2]string
}

func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *Session) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		s.auth = true
		return nil
	}), nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	a := ParseAddress(from)
	if slices.Contains(s.backend.Domains, a[1]) && !s.auth {
		return smtp.ErrAuthRequired
	}
	s.From = a
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	a := ParseAddress(to)
	if !slices.Contains(s.backend.Domains, a[1]) {
		s.Reset()
		return &smtp.SMTPError{
			Code:         551,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Forwarding to remote hosts disabled",
		}
	}
	s.To = a
	return nil
}

func (s *Session) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	println("got " + string(b))
	msg, err := mail.ReadMessage(bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	body := msg.Body
	headers := map[string][]string(msg.Header)
	var spam *RspamdResponse
	if s.backend.Rspamd == nil {
		// this goto avoids an hugly condition that is almost always met
		// thus, this goto produces a faster assembly because there is no bad branch prediction in the common case
		goto valid_email
	}
	spam, err = s.backend.Rspamd.Verify(context.Background(), nil, b)
	if err != nil {
		return err
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
		headers["X-Spam-Score"] = []string{strconv.FormatFloat(spam.Score, 'f', 2, 64)}
	case RewriteSubjectResponse:
		headers["Subject"] = []string{spam.Subject}
	case GreylistResponse:
	case NoActionResponse:
	default:
		panic("not implemented")
	}
	if spam.Body != nil {
		body = spam.Body
	}
valid_email:
	go func() {
		score := sql.NullFloat64{Float64: 0, Valid: false}
		if spam != nil {
			score.Float64 = spam.Score
			score.Valid = true
		}
		b, _ := io.ReadAll(body)
		err := s.backend.Storage.StoreEmail(context.Background(), b, s.From, s.To)
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
