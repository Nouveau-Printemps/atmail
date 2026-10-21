package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

const SpamHeader = `X-Spam-Score`

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

func (spam *RspamdClient) do(
	ctx context.Context,
	metadata *RspamdMetadata,
	email *message.Entity,
	uri string,
) (*multipart.Reader, error) {
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
	url := strings.TrimSuffix(spam.URL, "/")
	url += "/" + strings.TrimPrefix(uri, "/")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
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
	if rawResp.StatusCode > 299 {
		return nil, fmt.Errorf("invalid status code: %d", rawResp.StatusCode)
	}
	_, params, err := mime.ParseMediaType(rawResp.Header.Get("Content-Type"))
	if err != nil {
		log.Fatal(err)
	}
	email.Body = bytes.NewBuffer(body)
	return multipart.NewReader(rawResp.Body, params["boundary"]), nil
}

func (spam *RspamdClient) Verify(ctx context.Context, metadata *RspamdMetadata, email *message.Entity) (*RspamdResponse, error) {
	r, err := spam.do(ctx, metadata, email, "checkv3")
	if err != nil {
		return nil, err
	}
	p, err := r.NextPart()
	if err != nil {
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
	}
	if !errors.Is(err, io.EOF) {
		return nil, err
	}
	return &resp, nil
}

func (spam *RspamdClient) Analyze(s *Session, email *message.Entity) (float64, time.Duration, error) {
	metadata := RspamdMetadata{
		IP:    s.conn.Conn().RemoteAddr().(*net.TCPAddr).IP.String(),
		From:  s.From.Address,
		Rcpt:  utils.Map(s.To, func(rcpt Rcpt) string { return rcpt.Address }),
		Flags: []string{"body_block", "milter"},
	}
	l := utils.Logger(s.context).With("module", "rspamd")
	ctx := utils.WithLogger(s.context, l)
	if s.From.Local {
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
	email.Header.Del(SpamHeader)
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
		email.Header.Add(SpamHeader, strconv.FormatFloat(resp.Score, 'f', 2, 64))
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

func (spam *RspamdClient) LearnSpam(s *Session, email *message.Entity) error {
	body, err := io.ReadAll(email.Body)
	if err != nil {
		return err
	}
	email.Body = bytes.NewBuffer(body)
	req, err := http.NewRequestWithContext(
		s.context,
		http.MethodPost,
		spam.URL+"/learnspam",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	resp, err := spam.Client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}
	return nil
}

func (spam *RspamdClient) LearnHam(s *Session, email *message.Entity) error {
	body, err := io.ReadAll(email.Body)
	if err != nil {
		return err
	}
	email.Body = bytes.NewBuffer(body)
	req, err := http.NewRequestWithContext(
		s.context,
		http.MethodPost,
		spam.URL+"/learnham",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	resp, err := spam.Client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}
	return nil
}
