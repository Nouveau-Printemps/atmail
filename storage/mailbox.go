package storage

import (
	"context"
	"path"

	"github.com/emersion/go-imap/v2"
	"nouveauprintemps.org/atmail/storage/index"
	"nouveauprintemps.org/atmail/utils"
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
			if f.UserAdded == 1 {
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
		if opt.NumDeleted {
			nbr, err := in.CountEmailsWithFlagInMailbox(ctx, DeletedFlag, box.ID)
			if err != nil {
				return nil, err
			}
			status.NumMessages = new(*status.NumMessages - uint32(nbr))
		}
		if opt.NumRecent {
			status.NumRecent = new(uint32(0))
		}
		if opt.NumUnseen {
			nbr, err := in.CountEmailsWithFlagInMailbox(ctx, SeenFlag, box.ID)
			if err != nil {
				return nil, err
			}
			status.NumUnseen = new(*status.NumMessages - uint32(nbr))
		}
		if opt.UIDNext {
			mails, err := in.GetLatestMailboxEmails(ctx, box.ID, 1, 0)
			if err != nil {
				return nil, err
			}
			if len(mails) > 0 {
				status.UIDNext = imap.UID(mails[0].ID + 1)
			} else {
				status.UIDNext = 1
			}
		}
		return &status, nil
	})
}

func GetIds(ctx context.Context, user string, mailbox int64, set imap.NumSet) []int64 {
	if set.Dynamic() {
		panic("cannot extract seq from a dynamic")
	}
	switch v := set.(type) {
	case imap.SeqSet:
		seq, _ := v.Nums()
		return utils.Map(seq, func(id uint32) int64 {
			res, err := FromSequence(ctx, user, mailbox, id)
			if err != nil {
				panic(err)
			}
			return res
		})
	case imap.UIDSet:
		seq, _ := v.Nums()
		return utils.Map(seq, func(id imap.UID) int64 { return int64(id) })
	}
	panic("set not handled")
}

func ListMailboxEmails(
	ctx context.Context,
	user string,
	mailbox int64,
	set imap.NumSet,
) ([]index.Email, error) {
	return get(ctx, user, func(in *DB) ([]index.Email, error) {
		if !set.Dynamic() {
			return in.GetMailboxEmails(ctx, mailbox, GetIds(ctx, user, mailbox, set))
		}
		emails, err := in.ListMailboxEmails(ctx, mailbox)
		if err != nil {
			return nil, err
		}
		res := make([]index.Email, 0, 10)
		switch v := set.(type) {
		case imap.UIDSet:
			for _, email := range emails {
				if v.Contains(imap.UID(email.ID)) {
					res = append(res, email)
				}
			}
		case imap.SeqSet:
			for _, email := range emails {
				id, err := ToSequence(ctx, user, mailbox, email.ID)
				if err != nil {
					return nil, err
				}
				if v.Contains(id) {
					res = append(res, email)
				}
			}
		}
		return res, nil
	})
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

func ToSequence(ctx context.Context, user string, mailbox, id int64) (uint32, error) {
	return get(ctx, user, func(in *DB) (uint32, error) {
		seq, err := in.GetSequence(ctx, mailbox, id)
		if err != nil {
			return 0, err
		}
		return uint32(seq), nil
	})
}

func FromSequence(ctx context.Context, user string, mailbox int64, seq uint32) (int64, error) {
	return get(ctx, user, func(in *DB) (int64, error) {
		return in.FromSequence(ctx, mailbox, int64(seq))
	})
}
