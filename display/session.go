package display

import (
	"context"
	"log/slog"
	"maps"
	"regexp"
	"slices"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"nouveauprintemps.org/atmail/storage"
)

type Session struct {
	backend *Backend

	username string

	mailbox  string
	readOnly bool

	subscribed map[string]struct{}
}

func (s *Session) Login(username, password string) error {
	for _, cfg := range s.backend.Domains {
		if cfg.VerifyUser(username, password) {
			s.username = username
			break
		}
	}
	if len(s.username) == 0 {
		return imapserver.ErrAuthFailed
	}
	return nil
}

func (s *Session) Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error) {
	s.mailbox = mailbox
	s.readOnly = options.ReadOnly
	return storage.DescribeMailbox(context.TODO(), s.username, mailbox)
}

func (s *Session) Unselect() error {
	s.mailbox = ""
	return nil
}

func (s *Session) Create(mailbox string, options *imap.CreateOptions) error {
	return storage.CreateMailbox(context.TODO(), s.username, mailbox)
}

func (s *Session) Delete(mailbox string) error {
	return storage.DeleteMailbox(context.TODO(), s.username, mailbox)
}

func (s *Session) Rename(mailbox, newName string, options *imap.RenameOptions) error {
	return storage.RenameMailbox(context.TODO(), s.username, mailbox, newName)
}

func (s *Session) Subscribe(mailbox string) error {
	s.subscribed[mailbox] = struct{}{}
	return nil
}

func (s *Session) Unsubscribe(mailbox string) error {
	delete(s.subscribed, mailbox)
	return nil
}

func (s *Session) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {
	regs := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		r, err := parsePattern(ref, p)
		if err != nil {
			slog.Debug("error while parsing list pattern", "error", err)
		} else {
			regs = append(regs, r)
		}
	}
	var boxes []string
	if options.ReturnSubscribed {
		boxes = slices.Collect(maps.Keys(s.subscribed))
	} else {
		rawBoxes, err := storage.ListMailbox(context.TODO(), s.username)
		if err != nil {
			return err
		}
		boxes = make([]string, 0, len(rawBoxes))
		for _, v := range rawBoxes {
			boxes = append(boxes, v.Name)
		}
	}
	for _, r := range regs {
		for _, box := range boxes {
			if r.MatchString(box) {
				var attrs []imap.MailboxAttr
				if _, ok := s.subscribed[box]; ok {
					attrs = append(attrs, imap.MailboxAttrSubscribed)
				}
				var status *imap.StatusData
				if options.ReturnStatus != nil {
					var err error
					status, err = s.Status(s.mailbox, options.ReturnStatus)
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

func (s *Session) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error)

func (s *Session) Poll(w *imapserver.UpdateWriter, allowExpunge bool) error

func (s *Session) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error

func (s *Session) Close() error

func (s *Session) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) error

func (s *Session) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error)

func (s *Session) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error

func (s *Session) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error

func (s *Session) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error)
