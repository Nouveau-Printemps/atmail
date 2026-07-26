package storage

import (
	"context"
	"database/sql"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"nouveauprintemps.org/atmail/storage/index"
)

type Index = index.Queries

func newFileName() string {
	return strconv.Itoa(int(time.Now().Unix()))
}

func StoreEmail(ctx context.Context, from, to [2]string, spamScore sql.NullFloat64, b []byte) error {
	addr := strings.Join(to[:], "@")
	cache, err := Cache.DB(ctx, addr)
	if err != nil {
		return err
	}
	lastFile := cache.LastFile()
	var p string
	if lastFile == "" {
		lastFile = newFileName()
		p = lastFile
	} else if len(b) < MaxEmailSizeInFile {
		info, err := os.Stat(Cache.PathOf(addr, lastFile))
		if err != nil {
			return err
		}
		if info.Size() >= MaxFileSize {
			lastFile = newFileName()
		}
		p = lastFile
	} else {
		p = newFileName()
	}
	f, err := os.OpenFile(
		Cache.PathOf(addr, p),
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
	n, err := WriteEmail(f, b)
	if err != nil {
		return err
	}
	_, err = cache.index.NewEmail(ctx, index.NewEmailParams{
		MailFrom:  strings.Join(from[:], "@"),
		RcptTo:    addr,
		SpamScore: spamScore,
		Filename:  lastFile,
		Offset:    offset,
	})
	if err != nil {
		return err
	}
	cache.Update(lastFile, uint32(offset)+n)
	return nil
}

func Read(ctx context.Context, user string, id int64) ([]byte, error) {
	cache, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	email, err := cache.index.GetEmail(ctx, id)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(Cache.PathOf(user, email.Filename))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadEmailAt(f, uint32(email.Offset))
}
