package storage

import (
	"context"
	"os"

	"nouveauprintemps.org/atmail/storage/index"
)

type Index = index.Queries

type Storage struct {
	Index        *Index
	LastFilePath string
	LastFinish   uint32
}

func (s *Storage) StoreEmail(ctx context.Context, b []byte, from, to [2]string) error {
	f, err := os.OpenFile(s.LastFilePath, os.O_RDWR, 0o660)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := WriteEmail(f, b)
	if err != nil {
		return err
	}
	s.LastFinish += n
	return nil
}
