package storage

import (
	"context"

	"github.com/emersion/go-imap/v2"
	"nouveauprintemps.org/atmail/storage/index"
)

const MailboxSeparator = '/'

func LoadMailbox(ctx context.Context, user string) ([]index.ListMailboxRow, error) {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	return index.New(meta.db).ListMailbox(ctx)
}

func DescribeMailbox(ctx context.Context, user string, mailbox string) (*imap.SelectData, error) {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	in := index.New(meta.db)
	box, err := in.GetMailbox(ctx, mailbox)
	if err != nil {
		return nil, err
	}
	n, err := in.CountMailboxEmails(ctx, box.ID)
	if err != nil {
		return nil, err
	}
	mailboxFlags, err := in.GetMailboxFlags(ctx, box.ID)
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
}

func CreateMailbox(ctx context.Context, user, name string) error {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return err
	}
	_, err = index.New(meta.db).NewMailbox(ctx, name)
	return err
}

func DeleteMailbox(ctx context.Context, user string, id int64) error {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return err
	}
	return index.New(meta.db).DeleteMailbox(ctx, id)
}

func RenameMailbox(ctx context.Context, user string, id int64, rename string) error {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return err
	}
	return index.New(meta.db).RenameMailbox(ctx, rename, id)
}

func ListMailbox(ctx context.Context, user string) ([]index.ListMailboxRow, error) {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	return index.New(meta.db).ListMailbox(ctx)
}

func StatusMailbox(ctx context.Context, user, mailbox string, opt *imap.StatusOptions) (*imap.StatusData, error) {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	in := index.New(meta.db)
	box, err := in.GetMailbox(ctx, mailbox)
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
}

func GetMailboxEmails(ctx context.Context, user string, mailbox int64, set imap.NumSet) ([]index.Email, error) {
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

func AddMailboxEmail(ctx context.Context, user string, mailbox int64, emails []int64) error {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return err
	}
	tx, err := meta.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	in := index.New(tx)
	for _, id := range emails {
		err = in.AddMailboxEmail(ctx, mailbox, id)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
