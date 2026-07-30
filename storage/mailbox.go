package storage

import (
	"context"
	"path"

	"github.com/emersion/go-imap/v2"
	"nouveauprintemps.org/atmail/storage/index"
)

const MailboxSeparator = '/'

const (
	InboxMailbox int64 = iota + 1
	JunkMailbox
)

func LoadMailbox(ctx context.Context, user string) ([]index.ListMailboxRow, error) {
	return get(ctx, user, func(in *DB) ([]index.ListMailboxRow, error) {
		return in.ListMailbox(ctx)
	})
}

func DescribeMailbox(ctx context.Context, user string, mailbox string) (*imap.SelectData, error) {
	return get(ctx, user, func(in *DB) (*imap.SelectData, error) {
		box, err := in.GetMailbox(ctx, mailbox)
		if err != nil {
			return nil, err
		}
		n, err := in.CountMailboxEmails(ctx, box.ID)
		if err != nil {
			return nil, err
		}
		mailboxFlags, err := in.ListMailboxFlags(ctx, box.ID)
		if err != nil {
			return nil, err
		}
		flags := make([]imap.Flag, 0, len(mailboxFlags))
		var permanentFlags []imap.Flag
		for _, f := range mailboxFlags {
			if f.UserAdded {
				permanentFlags = append(permanentFlags, imap.Flag(f.Name))
			} else {
				flags = append(flags, imap.Flag(f.Name))
			}
		}
		mails, err := in.GetLatestMailboxEmails(ctx, box.ID, 1, 0)
		if err != nil {
			return nil, err
		}
		var next uint32
		if len(mails) == 0 {
			next = 1
		} else {
			next = uint32(mails[0].ID + 1)
		}
		return &imap.SelectData{
			Flags:          flags,
			PermanentFlags: permanentFlags,
			NumMessages:    uint32(n),
			UIDNext:        imap.UID(next),
			UIDValidity:    uint32(box.ID),
		}, nil
	})
}

func CreateMailbox(ctx context.Context, user, name string) error {
	return exec(ctx, user, func(in *DB) error {
		_, err := in.NewMailbox(ctx, name)
		return err
	})

}

func DeleteMailbox(ctx context.Context, user string, id int64) error {
	return exec(ctx, user, func(in *DB) error {
		return in.DeleteMailbox(ctx, id)
	})
}

func RenameMailbox(ctx context.Context, user string, id int64, rename string) error {
	return exec(ctx, user, func(in *DB) error {
		return in.RenameMailbox(ctx, rename, id)
	})
}

func ListMailbox(ctx context.Context, user string) ([]index.ListMailboxRow, error) {
	return get(ctx, user, func(in *DB) ([]index.ListMailboxRow, error) {
		return in.ListMailbox(ctx)
	})
}

func StatusMailbox(ctx context.Context, user, mailbox string, opt *imap.StatusOptions) (*imap.StatusData, error) {
	return get(ctx, user, func(in *DB) (*imap.StatusData, error) {
		box, err := in.GetMailbox(ctx, mailbox)
		if err != nil {
			return nil, err
		}
		var status imap.StatusData
		if opt.NumMessages {
			n, err := in.CountMailboxEmails(ctx, box.ID)
			if err != nil {
				return nil, err
			}
			status.NumMessages = new(uint32(n))
		}
		if opt.UIDValidity {
			status.UIDValidity = uint32(box.ID)
		}
		if opt.UIDNext {
			mails, err := in.GetLatestMailboxEmails(ctx, box.ID, 1, 0)
			if err != nil {
				return nil, err
			}
			status.UIDNext = imap.UID(mails[0].ID + 1)
		}
		return &status, nil
	})
}

func ListMailboxEmails(ctx context.Context, user string, mailbox int64, set imap.NumSet) ([]index.Email, error) {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	var ids []int64
	switch v := set.(type) {
	case imap.SeqSet:
		seq, _ := v.Nums()
		ids = make([]int64, 0, len(seq))
		for _, id := range seq {
			ids = append(ids, int64(id))
		}
	case imap.UIDSet:
		seq, _ := v.Nums()
		ids = make([]int64, 0, len(seq))
		for _, id := range seq {
			ids = append(ids, int64(id))
		}
	}
	return index.New(meta.db).GetMailboxEmails(ctx, mailbox, ids)
}

func AddMailboxEmails(ctx context.Context, user string, mailbox int64, emails []int64) error {
	return execTx(ctx, user, func(in *DB) error {
		for _, id := range emails {
			err := in.AddMailboxEmail(ctx, mailbox, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func DeleteEmail(user string, email index.Email) error {
	return email.Delete(path.Join(Cache.Path, user))
}

func RemoveMailboxEmails(ctx context.Context, user string, mailbox int64, emails []int64) error {
	err := execTx(ctx, user, func(in *DB) error {
		for _, id := range emails {
			err := in.RemoveMailboxEmail(ctx, mailbox, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return exec(ctx, user, func(in *DB) error {
		td, err := in.ListEmailsNoMailbox(ctx)
		if err != nil {
			return err
		}
		for _, email := range td {
			err = DeleteEmail(user, email)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
