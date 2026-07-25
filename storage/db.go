package storage

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"nouveauprintemps.org/atmail/storage/index"
)

type Index = index.Queries

type Storage struct {
	Index  *Index
	Folder string
}

func New(i *Index, folder string) *Storage {
	return &Storage{
		Index:  i,
		Folder: folder,
	}
}

func NewFileName() string {
	return strconv.Itoa(int(time.Now().Unix()))
}

func (s *Storage) StoreEmail(ctx context.Context, from, to [2]string, spamScore sql.NullFloat64, b []byte) error {
	addr := strings.Join(to[:], "@")
	meta, err := s.Index.GetMeta(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	lastFile := meta.LastFile.String
	var p string
	if !meta.LastFile.Valid {
		lastFile = NewFileName()
		p = lastFile
	} else if len(b) < MaxEmailSizeInFile {
		info, err := os.Stat(path.Join(s.Folder, lastFile))
		if err != nil {
			return err
		}
		if info.Size() >= MaxFileSize {
			lastFile = NewFileName()
		}
		p = lastFile
	} else {
		p = NewFileName()
	}
	f, err := os.OpenFile(path.Join(s.Folder, p), os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return err
	}
	defer f.Close()
	offset := meta.Offset.Int64
	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	n, err := WriteEmail(f, b)
	if err != nil {
		return err
	}
	_, err = s.Index.NewEmail(ctx, index.NewEmailParams{
		MailFrom:  strings.Join(from[:], "@"),
		RcptTo:    addr,
		SpamScore: spamScore,
		Filename:  lastFile,
		Offset:    offset,
	})
	if err != nil {
		return err
	}
	return s.Index.SetMeta(
		ctx,
		0,
		sql.NullString{String: lastFile, Valid: true},
		sql.NullInt64{Int64: offset + int64(n), Valid: true},
	)
}

func (s *Storage) Read(ctx context.Context, id int64) ([]byte, error) {
	email, err := s.Index.GetEmail(ctx, id)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path.Join(s.Folder, email.Filename))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadEmailAt(f, uint32(email.Offset))
}
