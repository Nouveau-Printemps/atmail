package relay

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-message/textproto"
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

	From      [2]string
	To        []Rcpt
	FromLocal bool
}

func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *Session) Auth(mech string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(identity, username, password string) error {
		ip := s.conn.Conn().RemoteAddr().(*net.TCPAddr).IP
		for k, cfg := range s.backend.Domains {
			ok := cfg.VerifyUser(ip, k, username, password)
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
	s.From = ParseAddress(from)
	_, s.FromLocal = s.backend.Domains[s.From[1]]
	if s.FromLocal && s.username != from {
		return smtp.ErrAuthRequired
	}
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	too := ParseAddress(to)
	rcpt := Rcpt{
		User:    too[0],
		Domain:  too[1],
		Address: to,
	}
	var cfg auth.Config
	cfg, rcpt.Local = s.backend.Domains[rcpt.Domain]
	// from local to outside
	if s.FromLocal && !rcpt.Local {
		s.To = append(s.To, rcpt)
		return nil
	}
	// not to local
	if !rcpt.Local {
		return &smtp.SMTPError{
			Code:         551,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Forwarding to remote hosts is disabled",
		}
	}
	// to local
	data := cfg.Exists(rcpt.User)
	if data == nil {
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: [3]int{5, 1, 1},
			Message:      "Address doesn't exist",
		}
	}
	if slices.Contains(auth.AdminEmails, data.Username) {
		rcpt.User = cfg.Admin.User
		rcpt.Folder = cfg.Admin.Folder
		rcpt.Key = cfg.Admin.Crypto.GetKey()
		rcpt.Address = cfg.Admin.User + "@" + rcpt.Domain
	} else {
		rcpt.User = data.Username
		rcpt.Folder = data.Folder
		rcpt.Key = data.Key
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
		"from", strings.Join(s.From[:], "@"),
		"to", strings.Join(
			utils.Map(s.To, func(r Rcpt) string { return r.Address }),
			",",
		),
	)
	b, err := io.ReadAll(r)
	if err != nil {
		l.Error("reading data", "error", err)
		return errInternal
	}
	buf := bufio.NewReader(bytes.NewBuffer(b))
	h, err := textproto.ReadHeader(buf)
	if err != nil {
		l.Error("parsing mail", "error", err)
		return errInternal
	}
	body, _ := io.ReadAll(buf)
	var spam *RspamdResponse
	var wait time.Duration
	err = func() error {
		if s.backend.Rspamd == nil {
			return nil
		}
		metadata := RspamdMetadata{
			IP:    s.conn.Conn().RemoteAddr().(*net.TCPAddr).IP.String(),
			From:  strings.Join(s.From[:], "@"),
			Rcpt:  utils.Map(s.To, func(rcpt Rcpt) string { return rcpt.Address }),
			Flags: []string{"body_block"},
		}
		if s.FromLocal {
			metadata.User = s.username
		}
		spam, err = s.backend.Rspamd.Verify(s.context, &metadata, b)
		if err != nil {
			l.Error("rspamd check", "error", err)
			return errInternal
		}
		if spam.Skipped {
			return nil
		}
		if spam.Messages != nil {
			v, ok := spam.Messages["smtp_message"]
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
				Message:      "Your message is temporarily unwanted, retry later",
			}
		case AddHeaderResponse:
			h.Add("X-Spam-Score", strconv.FormatFloat(spam.Score, 'f', 2, 64))
		case RewriteSubjectResponse:
			h.Add("Subject", spam.Subject)
		case GreylistResponse:
			wait = 15 * time.Minute
		case NoActionResponse:
		default:
			panic("not implemented")
		}
		if spam.Body != nil {
			body, _ = io.ReadAll(spam.Body)
		}
		return nil
	}()
	if err != nil {
		return err
	}
	go func() {
		if wait != 0 {
			time.Sleep(wait)
		}
		for d, groups := range utils.GroupBy(s.To, func(to Rcpt) string {
			return to.Domain
		}) {
			if !groups[0].Local {
				s.backend.Queue.Enqueue(
					strings.Join(s.From[:], "@"),
					utils.Map(groups, func(rcpt Rcpt) string { return rcpt.Address }),
					d,
					body,
				)
				continue
			}
			for _, rcpt := range groups {
				s.relayInside(s.context, rcpt, body, h, spam)
			}
		}
	}()
	return nil
}

func (s *Session) Reset() {
	s.To = nil
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
