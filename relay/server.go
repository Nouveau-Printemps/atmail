package relay

import (
	"io"
	"log"
	"slices"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type Backend struct {
	Domains []string
}

func (bkd *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{backend: bkd}, nil
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
			Code:         550,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Relaying denied",
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
	log.Println("Data: ", string(b))
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
