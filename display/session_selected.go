package display

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-message/textproto"
	"nouveauprintemps.org/atmail/storage"
	"nouveauprintemps.org/atmail/storage/store"
	"nouveauprintemps.org/atmail/utils"
)

func (s *Session) Close() error {
	if s.selected != nil {
		_, err := storage.DeleteEmailsWithFlag(context.TODO(), s.username, storage.DeletedFlag)
		if err != nil {
			return err
		}
		s.selected = nil
	}
	return nil
}

func (s *Session) Expunge(w *imapserver.ExpungeWriter, uids *imap.UIDSet) error {
	if uids == nil {
		deleted, err := storage.DeleteEmailsWithFlag(context.TODO(), s.username, storage.DeletedFlag)
		if err != nil {
			return err
		}
		for _, d := range deleted {
			err = w.WriteExpunge(uint32(d.ID))
			if err != nil {
				return err
			}
		}
		return nil
	}
	ns, _ := uids.Nums()
	keys := utils.Map(ns, func(v imap.UID) int64 { return int64(v) })
	err := storage.RemoveMailboxEmails(
		context.TODO(),
		s.username,
		int64(s.selected.ID),
		keys,
	)
	if err != nil {
		return err
	}
	for _, u := range ns {
		err = w.WriteExpunge(uint32(u))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) Search(
	kind imapserver.NumKind,
	criteria *imap.SearchCriteria,
	options *imap.SearchOptions,
) (*imap.SearchData, error) {
	return storage.Search(
		context.TODO(),
		s.username,
		int64(s.selected.ID),
		kind,
		criteria,
		options,
	)
}

func (s *Session) Fetch(wr *imapserver.FetchWriter, set imap.NumSet, options *imap.FetchOptions) error {
	emails, err := storage.ListMailboxEmails(
		context.TODO(),
		s.username,
		int64(s.selected.ID),
		set,
	)
	if err != nil {
		return err
	}
	markSeen := true
	for _, sec := range options.BinarySection {
		markSeen = markSeen && !sec.Peek
	}
	for _, sec := range options.BodySection {
		markSeen = markSeen && !sec.Peek
	}
	for _, email := range emails {
		if markSeen {
			err = storage.AddEmailFlag(context.TODO(), s.username, email.ID, storage.SeenFlag)
			if err != nil {
				return err
			}
			s.mailboxes[s.selected.Name].WriteMailboxFlags([]imap.Flag{imap.FlagSeen})
		}
		seq, err := storage.ToSequence(context.TODO(), s.username, int64(s.selected.ID), email.ID)
		if err != nil {
			return err
		}
		w := wr.CreateMessage(seq)

		if options.UID {
			w.WriteUID(imap.UID(email.ID))
		}
		if options.Flags {
			flags, err := storage.ListEmailFlags(context.TODO(), s.username, email.ID)
			if err != nil {
				return err
			}
			w.WriteFlags(utils.Map(flags, func(f store.Flag) imap.Flag { return imap.Flag(f.Name) }))
		}
		if options.InternalDate {
			w.WriteInternalDate(time.Unix(email.InternalDate, 0))
		}
		b, err := storage.ReadEmail(context.TODO(), s.username, email)
		if err != nil {
			return err
		}
		if options.RFC822Size {
			w.WriteRFC822Size(int64(len(b)))
		}
		if options.Envelope {
			h, _ := textproto.ReadHeader(bufio.NewReader(bytes.NewBuffer(b)))
			w.WriteEnvelope(imapserver.ExtractEnvelope(h))
		}
		if options.BodyStructure != nil {
			w.WriteBodyStructure(imapserver.ExtractBodyStructure(bytes.NewReader(b)))
		}

		for _, bs := range options.BodySection {
			buf := imapserver.ExtractBodySection(bytes.NewReader(b), bs)
			wc := w.WriteBodySection(bs, int64(len(buf)))
			_, writeErr := wc.Write(buf)
			closeErr := wc.Close()
			if writeErr != nil {
				return writeErr
			}
			if closeErr != nil {
				return closeErr
			}
		}

		for _, bs := range options.BinarySection {
			buf := imapserver.ExtractBinarySection(bytes.NewReader(b), bs)
			wc := w.WriteBinarySection(bs, int64(len(buf)))
			_, writeErr := wc.Write(buf)
			closeErr := wc.Close()
			if writeErr != nil {
				return writeErr
			}
			if closeErr != nil {
				return closeErr
			}
		}

		for _, bss := range options.BinarySectionSize {
			n := imapserver.ExtractBinarySectionSize(bytes.NewReader(b), bss)
			w.WriteBinarySectionSize(bss, n)
		}

		err = w.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) Store(
	w *imapserver.FetchWriter,
	set imap.NumSet,
	flags *imap.StoreFlags,
	options *imap.StoreOptions,
) error {
	emails, err := storage.ListMailboxEmails(
		context.TODO(),
		s.username,
		int64(s.selected.ID),
		set,
	)
	if err != nil {
		return err
	}
	for _, email := range emails {
		switch flags.Op {
		case imap.StoreFlagsSet:
			err = storage.RemoveEmailAllFlags(context.TODO(), s.username, email.ID)
			if err != nil {
				return err

			}
			fallthrough
		case imap.StoreFlagsAdd:
			err = storage.AddEmailFlags(context.TODO(), s.username, email.ID, flags.Flags)
		case imap.StoreFlagsDel:
			err = storage.RemoveEmailFlags(context.TODO(), s.username, email.ID, flags.Flags)
		default:
			panic(fmt.Errorf("unknown STORE flag operation: %v", flags.Op))
		}
		if err != nil {
			return err
		}
		seq, err := storage.ToSequence(context.TODO(), s.username, int64(s.selected.ID), email.ID)
		if err != nil {
			return err
		}
		s.mailboxes[s.selected.Name].WriteMessageFlags(
			seq,
			s.selected.ID,
			flags.Flags,
		)
	}
	if !flags.Silent {
		return s.Fetch(w, set, &imap.FetchOptions{Flags: true})
	}
	return nil
}

func (s *Session) Copy(set imap.NumSet, dest string) (*imap.CopyData, error) {
	if dest == s.selected.Name {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "This is the same mailbox",
		}
	}
	target, ok := s.mailboxes[dest]
	if !ok {
		return nil, errNotFound
	}

	mails, err := storage.ListMailboxEmails(
		context.TODO(),
		s.username,
		int64(target.ID),
		set,
	)
	if err != nil {
		return nil, err
	}
	ids := utils.Map(mails, func(m store.Email) int64 { return m.ID })
	err = storage.AddMailboxEmails(context.TODO(), s.username, int64(target.ID), ids)
	if err != nil {
		return nil, err
	}
	return &imap.CopyData{
		UIDValidity: uint32(target.ID),
	}, nil
}

func (s *Session) Move(w *imapserver.MoveWriter, set imap.NumSet, dest string) error {
	v, err := s.Copy(set, dest)
	if err != nil {
		return err
	}
	err = w.WriteCopyData(v)
	if err != nil {
		return err
	}
	return storage.RemoveMailboxEmails(
		context.TODO(),
		s.username,
		int64(s.selected.ID),
		storage.GetIds(context.TODO(), s.username, int64(s.selected.ID), set),
	)
}

func (s *Session) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	return s.selected.Idle(w, stop)
}
