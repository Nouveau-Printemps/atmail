package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/utils"
)

type RspamdClient struct {
	Client *http.Client
	URL    string
}

type RspamdMetadata struct {
	IP         string          `json:"ip"`
	Helo       string          `json:"helo,omitzero"`
	Hostname   string          `json:"hostname,omitzero"`
	Flags      []string        `json:"flags,omitzero"`
	From       string          `json:"from"`
	QueueId    string          `json:"queue_id,omitzero"`
	Rcpt       []string        `json:"rcpt"`
	User       string          `json:"user,omitzero"`
	SettingsId string          `json:"settings_id,omitzero"`
	Settings   json.RawMessage `json:"settings,omitzero"`
	Mime       string          `json:"mime,omitzero"`
}

type ActionResponse string

const (
	NoActionResponse       ActionResponse = "no action"
	GreylistResponse       ActionResponse = "greylist"
	AddHeaderResponse      ActionResponse = "add header"
	RewriteSubjectResponse ActionResponse = "rewrite subject"
	SoftRejectResponse     ActionResponse = "soft reject"
	RejectResponse         ActionResponse = "reject"
)

type RspamdSymbol struct {
	Name    string   `json:"name"`
	Score   float64  `json:"score"`
	Options []string `json:"options"`
}

type RspamdResponse struct {
	Skipped       bool                    `json:"is_skipped"`
	Score         float64                 `json:"score"`
	RequiredScore float64                 `json:"required_score"`
	Action        ActionResponse          `json:"action"`
	Symbols       map[string]RspamdSymbol `json:"symbols"`
	// Subject is set if [RspamdResponse.Action] is [RewriteSubjectResponse].
	Subject string `json:"subject,omitzero"`
	// URLs found in the message.
	URLs []string `json:"urls,omitzero"`
	// Emails found in the message.
	Emails    []string `json:"emails,omitzero"`
	MessageId string   `json:"message-id,omitzero"`
	// Messages returned by Rspamd (smtp_message key is intended to be returned as SMTP response)
	Messages      map[string]string `json:"messages,omitzero"`
	DkimSignature *string           `json:"dkim-signature,omitzero"`
}

func (spam *RspamdClient) Verify(ctx context.Context, metadata *RspamdMetadata, email *message.Entity) (*RspamdResponse, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if metadata != nil {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="metadata"`)
		h.Set("Content-Type", "application/json")
		meta, err := w.CreatePart(h)
		if err != nil {
			return nil, err
		}
		err = json.NewEncoder(meta).Encode(metadata)
		if err != nil {
			return nil, err
		}
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="message"`)
	h.Set("Content-Type", "application/octet-stream")
	// store a copy of the body email
	body, err := io.ReadAll(email.Body)
	if err != nil {
		return nil, err
	}
	email.Body = bytes.NewBuffer(body)
	var cp bytes.Buffer
	err = email.WriteTo(&cp) // because it formats the header + body
	if err != nil {
		return nil, err
	}
	// write the copy of the email in multipart
	msg, err := w.CreatePart(h)
	if err != nil {
		return nil, err
	}
	_, err = cp.WriteTo(msg)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		spam.URL+"/checkv3",
		&buf,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("PerformDkimSign", "Yes")
	rawResp, err := spam.Client.Do(req)
	if err != nil {
		return nil, err
	}
	_, params, err := mime.ParseMediaType(rawResp.Header.Get("Content-Type"))
	if err != nil {
		log.Fatal(err)
	}
	r := multipart.NewReader(rawResp.Body, params["boundary"])
	p, err := r.NextPart()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("no content")
		}
		return nil, err
	}
	var resp RspamdResponse
	err = json.NewDecoder(p).Decode(&resp)
	if err != nil {
		return nil, err
	}
	p, err = r.NextPart()
	if err == nil {
		email.Body = p
		_, err = r.NextPart()
		if err == nil {
			return nil, errors.New("invalid response")
		}
	} else {
		// restore email's body (see comments above)
		email.Body = bytes.NewBuffer(body)
	}
	if !errors.Is(err, io.EOF) {
		return nil, err
	}
	return &resp, nil
}

func (spam *RspamdClient) Analyze(s *Session, email *message.Entity) (float64, time.Duration, error) {
	metadata := RspamdMetadata{
		IP:    s.conn.Conn().RemoteAddr().(*net.TCPAddr).IP.String(),
		From:  strings.Join(s.From[:], "@"),
		Rcpt:  utils.Map(s.To, func(rcpt Rcpt) string { return rcpt.Address }),
		Flags: []string{"body_block", "milter"},
	}
	l := utils.Logger(s.context).With("module", "rspamd")
	ctx := utils.WithLogger(s.context, l)
	if s.FromLocal {
		metadata.User = s.username
	}
	resp, err := spam.Verify(ctx, &metadata, email)
	if err != nil {
		l.Error("rspamd check", "error", err)
		return 0, 0, errInternal
	}
	if resp.Skipped {
		return 0, 0, nil
	}
	if resp.DkimSignature != nil {
		// because go-message.Entity doesn't accept \n\t in headers
		email.Header.Add("DKIM-Signature", strings.ReplaceAll(*resp.DkimSignature, "\n\t", " "))
	}
	if resp.Messages != nil {
		v, ok := resp.Messages["smtp_message"]
		if ok {
			return 0, 0, &smtp.SMTPError{
				Code:         550,
				EnhancedCode: [3]int{5, 7, 1},
				Message:      v,
			}
		}
	}
	var wait time.Duration
	switch resp.Action {
	case RejectResponse:
		l.Debug("rejecting message as spam")
		return 0, 0, &smtp.SMTPError{
			Code:         550,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Your message is unwanted",
		}
	case SoftRejectResponse:
		l.Debug("soft rejecting message")
		return 0, 0, &smtp.SMTPError{
			Code:         450,
			EnhancedCode: [3]int{4, 7, 1},
			Message:      "Your message is temporarily unwanted, retry later",
		}
	case AddHeaderResponse:
		l.Debug("adding header")
		email.Header.Add("X-Spam", strconv.FormatFloat(resp.Score, 'f', 2, 64))
	case RewriteSubjectResponse:
		l.Debug("modifying subject")
		email.Header.Set("Subject", resp.Subject)
	case GreylistResponse:
		l.Debug("greylisting message")
		wait = 15 * time.Minute
	case NoActionResponse:
	default:
		panic("not implemented")
	}
	return resp.Score, wait, nil
}
