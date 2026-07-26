package display

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

type Session struct {
	backend *Backend

	authFor []string
}

func (s *Session) Close() error

func (s *Session) Login(username, password string) error {
	for k, cfg := range s.backend.Domains {
		if cfg.VerifyUser(username, password) {
			s.authFor = append(s.authFor, k)
		}
	}
	if len(s.authFor) == 0 {
		return imapserver.ErrAuthFailed
	}
	return nil
}

func (s *Session) Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error)

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

func (s *Session) Unselect() error

func (s *Session) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) error

func (s *Session) Search(kind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error)

func (s *Session) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error

func (s *Session) Store(w *imapserver.FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error

func (s *Session) Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error)
