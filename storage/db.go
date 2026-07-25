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
	Index      *Index
	LastFile   string
	LastFinish uint32
	Folder     string
}

func New(i *Index, folder string) *Storage {
	return &Storage{
		Index:      i,
		LastFile:   NewFileName(),
		LastFinish: 0,
		Folder:     folder,
	}
}

func NewFileName() string {
	return strconv.Itoa(int(time.Now().Unix()))
}

func (s *Storage) StoreEmail(ctx context.Context, from, to [2]string, spamScore sql.NullFloat64, b []byte) error {
	lastFile := s.LastFile
	if len(b) >= MaxEmailSizeInFile {
		lastFile = NewFileName()
	}
	f, err := os.OpenFile(path.Join(s.Folder, lastFile), os.O_RDWR|os.O_CREATE, 0o660)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Seek(int64(s.LastFinish), io.SeekStart)
	if err != nil {
		return err
	}
	n, err := WriteEmail(f, b)
	if err != nil {
		return err
	}
	_, err = s.Index.NewEmail(ctx, index.NewEmailParams{
		MailFrom:  strings.Join(from[:], "@"),
		RcptTo:    strings.Join(to[:], "@"),
		SpamScore: spamScore,
		Filename:  lastFile,
		Offset:    int64(s.LastFinish),
	})
	if err != nil {
		return err
	}
	s.LastFinish += n
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
