package storage

import (
	"context"
	"database/sql"
	"io"
	"strings"
	"time"

	"nouveauprintemps.org/atmail/storage/store"
)

func StoreEmail(
	ctx context.Context,
	from, to [2]string,
	user string,
	spamScore sql.NullFloat64,
	b []byte,
	encrypted bool,
	callback func(context.Context, *DB, int64) error,
) error {
	cache, err := Cache.DB(ctx, user)
	if err != nil {
		return err
	}
	lastFile := cache.LastFile()
	offset := int64(cache.Offset())
	f, err := store.OpenFile(ctx, cache.path, &lastFile, &offset, b)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := store.WriteEmail(f, encrypted, b)
	if err != nil {
		return err
	}
	tx, err := cache.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	in := store.New(tx)
	id, err := in.NewEmail(ctx, store.NewEmailParams{
		MailFrom:     strings.Join(from[:], "@"),
		RcptTo:       strings.Join(to[:], "@"),
		SpamScore:    spamScore,
		Filename:     lastFile,
		InternalDate: time.Now().Unix(),
		Offset:       offset,
	})
	if err != nil {
		return err
	}
	err = callback(ctx, in, id)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	cache.Update(lastFile, uint32(offset)+n)
	return nil
}

func StoreSpam(
	ctx context.Context,
	from, to [2]string,
	user string,
	spamScore sql.NullFloat64,
	b []byte,
	encrypted bool,
) (uid int64, err error) {
	err = StoreEmail(ctx, from, to, user, spamScore, b, encrypted, func(ctx context.Context, in *DB, id int64) error {
		uid = id
		err = in.AddEmailFlag(ctx, id, JunkFlag)
		if err != nil {
			return err
		}
		return in.AddMailboxEmail(ctx, JunkMailbox, id)
	})
	return
}

func StoreEmailInbox(
	ctx context.Context,
	from, to [2]string,
	user string,
	spamScore sql.NullFloat64,
	b []byte,
	folder string,
	encrypted bool,
) (uid int64, err error) {
	err = StoreEmail(ctx, from, to, user, spamScore, b, encrypted, func(ctx context.Context, in *DB, id int64) error {
		uid = id
		boxID := InboxMailbox
		if folder != "" {
			box, err := in.GetOrCreateMailbox(ctx, folder)
			if err != nil {
				return err
			}
			boxID = box.ID
		}
		return in.AddMailboxEmail(ctx, boxID, id)
	})
	return
}

func Read(ctx context.Context, user string, id int64) (io.ReadCloser, byte, error) {
	cache, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, 0, err
	}
	email, err := store.New(cache.db).GetEmail(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	return email.Read(cache.path)
}

func ReadEmail(ctx context.Context, user string, email store.Email) (io.ReadCloser, byte, error) {
	cache, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, 0, err
	}
	return email.Read(cache.path)
}

func DeleteEmail(ctx context.Context, user string, id int64) error {
	return exec(ctx, user, func(in *DB) error {
		email, err := in.GetEmail(ctx, id)
		if err != nil {
			return err
		}
		return deleteEmail(ctx, in, user, email)
	})
}

func GetEmail(ctx context.Context, user string, id int64) (store.Email, error) {
	return get(ctx, user, func(in *DB) (store.Email, error) {
		return in.GetEmail(ctx, id)
	})
}

func EditEmail(ctx context.Context, user string, email store.Email, encrypted bool, content []byte) error {
	cache, err := Cache.DB(ctx, user)
	if err != nil {
		return err
	}
	lastFile := cache.LastFile()
	offset := int64(cache.Offset())
	f, err := store.OpenFile(ctx, cache.path, &lastFile, &offset, content)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := store.WriteEmail(f, encrypted, content)
	if err != nil {
		return err
	}
	err = store.DeleteEmailAt(f, uint32(email.Offset))
	if err != nil {
		return err
	}
	err = store.New(cache.db).UpdateOffset(ctx, offset, email.ID)
	if err != nil {
		return err
	}
	cache.Update(lastFile, uint32(offset)+n)
	return nil
}

func ListEmails(ctx context.Context, user string) ([]store.Email, error) {
	return get(ctx, user, func(in *DB) ([]store.Email, error) {
		return in.ListEmails(ctx)
	})
}
