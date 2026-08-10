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

	mailproto "github.com/emersion/go-message/textproto"

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
	User       string          `json:"user"`
	SettingsId string          `json:"settings_id"`
	Settings   json.RawMessage `json:"settings"`
	Mime       string          `json:"mime"`
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
	Messages map[string]string `json:"messages,omitzero"`

	// Set if body is given
	Body io.Reader `json:"-"`
}

func (spam *RspamdClient) Verify(ctx context.Context, metadata *RspamdMetadata, mail []byte) (*RspamdResponse, error) {
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
	msg, err := w.CreatePart(h)
	if err != nil {
		return nil, err
	}
	_, err = msg.Write(mail)
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
			return nil, errors.New("rspamd: no content")
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
		resp.Body = p
		p, err = r.NextPart()
	}
	if err == nil {
		return nil, errors.New("rspamd: invalid response")
	}
	if !errors.Is(err, io.EOF) {
		return nil, err
	}
	return &resp, nil
}

func (spam *RspamdClient) Analyze(s *Session, h mailproto.Header, b []byte) (float64, []byte, time.Duration, error) {
	metadata := RspamdMetadata{
		IP:    s.conn.Conn().RemoteAddr().(*net.TCPAddr).IP.String(),
		From:  strings.Join(s.From[:], "@"),
		Rcpt:  utils.Map(s.To, func(rcpt Rcpt) string { return rcpt.Address }),
		Flags: []string{"body_block"},
	}
	l := utils.Logger(s.context).With("module", "rspamd")
	ctx := utils.WithLogger(s.context, l)
	if s.FromLocal {
		metadata.User = s.username
	}
	resp, err := spam.Verify(ctx, &metadata, b)
	if err != nil {
		l.Error("rspamd check", "error", err)
		return 0, nil, 0, errInternal
	}
	if resp.Skipped {
		return 0, b, 0, nil
	}
	if resp.Messages != nil {
		v, ok := resp.Messages["smtp_message"]
		if ok {
			return 0, nil, 0, &smtp.SMTPError{
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
		return 0, nil, 0, &smtp.SMTPError{
			Code:         550,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Your message is unwanted",
		}
	case SoftRejectResponse:
		l.Debug("soft rejecting message")
		return 0, nil, 0, &smtp.SMTPError{
			Code:         450,
			EnhancedCode: [3]int{4, 7, 1},
			Message:      "Your message is temporarily unwanted, retry later",
		}
	case AddHeaderResponse:
		l.Debug("adding header")
		h.Add("X-Spam-Score", strconv.FormatFloat(resp.Score, 'f', 2, 64))
	case RewriteSubjectResponse:
		l.Debug("modifying subject")
		h.Add("Subject", resp.Subject)
	case GreylistResponse:
		l.Debug("greylisting message")
		wait = 15 * time.Minute
	case NoActionResponse:
	default:
		panic("not implemented")
	}
	if resp.Body != nil {
		b, _ = io.ReadAll(resp.Body)
	}
	return resp.Score, b, wait, nil
}
