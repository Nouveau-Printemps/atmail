package display

import (
	"context"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"nouveauprintemps.org/atmail/storage"
)

type Session struct {
	backend *Backend

	username string

	mailbox  string
	readOnly bool
}

func (s *Session) Close() error {
	*s = Session{backend: s.backend}
	return nil
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

func (s *Session) Create(mailbox string, options *imap.CreateOptions) error

func (s *Session) Delete(mailbox string) error

func (s *Session) Rename(mailbox, newName string, options *imap.RenameOptions) error

func (s *Session) Subscribe(mailbox string) error

func (s *Session) Unsubscribe(mailbox string) error

func (s *Session) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error

func (s *Session) Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error)

func (s *Session) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error)

func (s *Session) Poll(w *imapserver.UpdateWriter, allowExpunge bool) error

func (s *Session) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error

func (s *Session) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) error

func (s *Session) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error)

func (s *Session) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error

func (s *Session) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error

func (s *Session) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error)
