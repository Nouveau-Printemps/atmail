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
	lastFile := Cache.LastFile()
	var p string
	if lastFile == "" {
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
	offset := int64(Cache.Offset())
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
	Cache.Update(lastFile, uint32(offset)+n)
	return nil
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
