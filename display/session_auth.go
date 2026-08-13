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
	"nouveauprintemps.org/atmail/mailbox"
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
	s.selected = m
	s.readOnly = options.ReadOnly
	res, err := storage.DescribeMailbox(s.context, s.username, mailbox)
	if err != nil {
		utils.Logger(s.context).Error("describing mailbox", "error", err)
		return nil, errInternal
	}
	s.selected.Count.Store(res.NumMessages)
	return res, nil
}

func (s *Session) Unselect() error {
	s.selected = nil
	return nil
}

func (s *Session) Create(box string, options *imap.CreateOptions) error {
	if _, ok := s.mailboxes[box]; ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeAlreadyExists,
			Text: "already exists",
		}
	}
	sep := string(storage.MailboxSeparator)
	if strings.HasSuffix(box, sep) ||
		strings.Contains(box, sep+sep) {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeCannot,
			Text: "bad mailbox name",
		}
	}
	box = strings.TrimPrefix(box, sep)
	id, err := storage.CreateMailbox(s.context, s.username, box)
	if err != nil {
		utils.Logger(s.context).Error("creating mailbox", "error", err)
		return err
	}
	s.mailboxes[box] = mailbox.NewView(uint32(id), box, 0)
	return nil
}

func (s *Session) Delete(mailbox string) error {
	box, ok := s.mailboxes[mailbox]
	if !ok {
		return errNotFound
	}
	err := storage.DeleteMailbox(s.context, s.username, int64(box.ID))
	if err != nil {
		utils.Logger(s.context).Error("deleting mailbox", "error", err)
		return errInternal
	}
	return nil
}

func (s *Session) Rename(mailbox, newName string, options *imap.RenameOptions) error {
	box, ok := s.mailboxes[mailbox]
	if !ok {
		return errNotFound
	}
	err := storage.RenameMailbox(s.context, s.username, int64(box.ID), newName)
	if err != nil {
		utils.Logger(s.context).Error("renaming mailbox", "error", err)
		return errInternal
	}
	return nil
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
		boxes = utils.ReduceMapToSlice(s.mailboxes, func(k string, _ *mailbox.View) string {
			return k
		})
	}
	for p, box := range utils.Zip(slices.Values(patterns), slices.Values(boxes)) {
		if !imapserver.MatchList(box, storage.MailboxSeparator, ref, p) {
			continue
		}
		l := utils.Logger(s.context).With("box", box)
		var attrs []imap.MailboxAttr
		if _, ok := s.subscribed[box]; ok {
			attrs = append(attrs, imap.MailboxAttrSubscribed)
		}
		a, err := storage.GetMailboxAttributes(s.context, s.username, box)
		if err != nil {
			l.Error("getting mailbox attributes", "error", err)
			return errInternal
		}
		attrs = append(attrs, a...)
		var status *imap.StatusData
		if options.ReturnStatus != nil {
			var err error
			status, err = s.Status(box, options.ReturnStatus)
			if err != nil {
				l.Error("fetching status", "error", err)
				return errInternal
			}
		}
		err = w.WriteList(&imap.ListData{
			Delim:   storage.MailboxSeparator,
			Mailbox: box,
			Attrs:   attrs,
			Status:  status,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error) {
	_, ok := s.mailboxes[mailbox]
	if !ok {
		return nil, errNotFound
	}
	st, err := storage.StatusMailbox(
		s.context,
		s.username,
		mailbox,
		options,
	)
	if err != nil {
		utils.Logger(s.context).Error("fetchin status", "error", err)
		return nil, errInternal
	}
	return st, nil
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
	from, err := mail.ParseAddress(m.Header.Get("From"))
	if err != nil {
		return nil, err
	}
	if from.Address != s.username {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeAuthorizationFailed,
			Text: "from header field is invalid",
		}
	}
	to := m.Header.Get("To")
	var uid imap.UID
	err = storage.StoreEmail(
		s.context,
		relay.ParseAddress(from.Address), relay.ParseAddress(to),
		s.username,
		sql.NullFloat64{Float64: 0, Valid: false},
		b,
		false,
		func(ctx context.Context, in *storage.DB, id int64) error {
			uid = imap.UID(id)
			return in.AddMailboxEmail(ctx, int64(box.ID), id)
		})
	if err != nil {
		utils.Logger(s.context).Error("storing email", "error", err)
		return nil, errInternal
	}
	box.WriteNewMessages(1)
	return &imap.AppendData{
		UID:         imap.UID(uid),
		UIDValidity: box.ID,
	}, nil
}

func (s *Session) Poll(w *imapserver.UpdateWriter, allowExpunge bool) error {
	return nil
}

func (s *Session) Namespace() (*imap.NamespaceData, error) {
	return &imap.NamespaceData{
		Personal: []imap.NamespaceDescriptor{{
			Prefix: "",
			Delim:  '/',
		}},
	}, nil
}
