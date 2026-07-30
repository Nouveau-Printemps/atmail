package display

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"maps"
	"net/mail"
	"slices"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"nouveauprintemps.org/atmail/relay"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/utils"
)

var errNotFound = &imap.Error{
	Type: imap.StatusResponseTypeNo,
	Code: imap.ResponseCodeTryCreate,
	Text: "Mailbox not found",
}

func (s *Session) Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error) {
	m, ok := s.mailboxes[mailbox]
	if !ok {
		return nil, errNotFound
	}
	s.selected = &MailboxSelectedView{
		SessionTracker: m.NewSession(),
		Name:           mailbox,
	}
	s.readOnly = options.ReadOnly
	res, err := storage.DescribeMailbox(context.TODO(), s.username, mailbox)
	if err != nil {
		return nil, err
	}
	s.selected.ID = imap.UID(res.UIDValidity)
	return res, nil
}

func (s *Session) Unselect() error {
	s.selected.Close()
	s.selected = nil
	return nil
}

func (s *Session) Create(mailbox string, options *imap.CreateOptions) error {
	sep := string(storage.MailboxSeparator)
	if strings.HasSuffix(mailbox, sep) ||
		strings.Contains(mailbox, sep+sep) {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeCannot,
			Text: "bad mailbox name",
		}
	}
	mailbox = strings.TrimPrefix(mailbox, sep)
	return storage.CreateMailbox(context.TODO(), s.username, mailbox)
}

func (s *Session) Delete(mailbox string) error {
	box, ok := s.mailboxes[mailbox]
	if !ok {
		return errNotFound
	}
	return storage.DeleteMailbox(context.TODO(), s.username, int64(box.UIDValidity))
}

func (s *Session) Rename(mailbox, newName string, options *imap.RenameOptions) error {
	box, ok := s.mailboxes[mailbox]
	if !ok {
		return errNotFound
	}
	return storage.RenameMailbox(context.TODO(), s.username, int64(box.UIDValidity), newName)
}

func (s *Session) Subscribe(mailbox string) error {
	_, ok := s.mailboxes[mailbox]
	if !ok {
		return errNotFound
	}
	s.subscribed[mailbox] = struct{}{}
	return nil
}

func (s *Session) Unsubscribe(mailbox string) error {
	delete(s.subscribed, mailbox)
	return nil
}

func (s *Session) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {
	var boxes []string
	if options.ReturnSubscribed {
		boxes = slices.Collect(maps.Keys(s.subscribed))
	} else {
		boxes = utils.ReduceMapToSlice(s.mailboxes, func(k string, _ MailboxView) string {
			return k
		})
	}
	for _, p := range patterns {
		for _, box := range boxes {
			if imapserver.MatchList(box, storage.MailboxSeparator, ref, p) {
				var attrs []imap.MailboxAttr
				if _, ok := s.subscribed[box]; ok {
					attrs = append(attrs, imap.MailboxAttrSubscribed)
				}
				var status *imap.StatusData
				if options.ReturnStatus != nil {
					var err error
					status, err = s.Status(box, options.ReturnStatus)
					if err != nil {
						return err
					}
				}
				err := w.WriteList(&imap.ListData{
					Delim:   storage.MailboxSeparator,
					Mailbox: box,
					Attrs:   attrs,
					Status:  status,
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Session) Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error) {
	return storage.StatusMailbox(
		context.TODO(),
		s.username,
		mailbox,
		options,
	)
}

func (s *Session) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	if uint32(r.Size()) >= s.backend.MaxMailSize {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeLimit,
			Text: "mail is too big to be added",
		}
	}
	box, ok := s.mailboxes[mailbox]
	if !ok {
		return nil, errNotFound
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	m, err := mail.ReadMessage(bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	from := m.Header.Get("From")
	if from != s.username {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeAuthorizationFailed,
			Text: "from header field is invalid",
		}
	}
	to := m.Header.Get("To")
	var uid imap.UID
	err = storage.StoreEmail(
		context.TODO(),
		relay.ParseAddress(from), relay.ParseAddress(to),
		sql.NullFloat64{Float64: 0, Valid: false},
		b,
		func(ctx context.Context, in *storage.DB, id int64) error {
			uid = imap.UID(id)
			return in.AddMailboxEmail(ctx, int64(box.UIDValidity), id)
		})
	if err != nil {
		return nil, err
	}
	box.QueueNumMessages(uint32(uid))
	return &imap.AppendData{
		UID:         imap.UID(uid),
		UIDValidity: uint32(box.UIDValidity),
	}, nil
}

func (s *Session) Poll(w *imapserver.UpdateWriter, allowExpunge bool) error {
	return s.selected.Poll(w, allowExpunge)
}

func (s *Session) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	return s.selected.Idle(w, stop)
}
