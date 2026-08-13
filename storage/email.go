package storage

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"nouveauprintemps.org/atmail/storage/store"
)

func newFileName() string {
	return strconv.Itoa(int(time.Now().Unix()))
}

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
	var p string
	if lastFile == "" {
		lastFile = newFileName()
		p = lastFile
	} else if len(b) < store.MaxEmailSizeInFile {
		info, err := os.Stat(cache.path + lastFile)
		if err != nil {
			return err
		}
		if info.Size() >= store.MaxFileSize {
			lastFile = newFileName()
		}
		p = lastFile
	} else {
		p = newFileName()
	}
	f, err := os.OpenFile(
		path.Join(cache.path, p),
		os.O_RDWR|os.O_CREATE,
		0o660,
	)
	if err != nil {
		return err
	}
	defer f.Close()
	offset := int64(cache.Offset())
	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
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
			box, err := in.GetOrCreateMailbox(ctx, "INBOX"+string(MailboxSeparator)+folder)
			if err != nil {
				return err
			}
			boxID = box.ID
		}
		return in.AddMailboxEmail(ctx, boxID, id)
	})
	return
}

func Read(ctx context.Context, user string, id int64) ([]byte, error) {
	cache, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	email, err := store.New(cache.db).GetEmail(ctx, id)
	if err != nil {
		return nil, err
	}
	return email.Read(cache.path)
}

func ReadEmail(ctx context.Context, user string, email store.Email) ([]byte, error) {
	cache, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	return email.Read(cache.path)
}
