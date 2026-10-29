package store

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	"nouveauprintemps.org/atmail/utils"
)

func (e *Email) openFile(base string) (*os.File, error) {
	return os.OpenFile(path.Join(base, e.Filename), os.O_RDWR, 0o660)
}

func (e *Email) Read(base string) (io.ReadCloser, byte, error) {
	f, err := e.openFile(base)
	if err != nil {
		return nil, 0, err
	}
	return e.Reader(f)
}

func (e *Email) Reader(f *os.File) (io.ReadCloser, byte, error) {
	return Reader(f, uint32(e.Offset))
}

func (e *Email) Delete(base string) error {
	f, err := e.openFile(base)
	if err != nil {
		return err
	}
	defer f.Close()
	return DeleteEmailAt(f, uint32(e.Offset))
}

func newFileName() string {
	return strconv.Itoa(int(time.Now().Unix()))
}

func OpenFile(ctx context.Context, base string, lastFile *string, offset *int64, b []byte) (*os.File, error) {
	var p string
	var off int64 = 0
	if len(b) < MaxEmailSizeInFile && *lastFile != "" {
		info, err := os.Stat(path.Join(base, *lastFile))
		if err != nil {
			return nil, err
		}
		if info.Size() >= MaxFileSize {
			*lastFile = newFileName()
		}
		p = *lastFile
		off = *offset
	} else {
		p = newFileName()
		off = 0
	}
	if *lastFile == "" {
		*lastFile = p
		*offset = off
	}
	f, err := os.OpenFile(
		path.Join(base, p),
		os.O_RDWR|os.O_CREATE,
		0o660,
	)
	if err != nil {
		return nil, err
	}
	_, err = f.Seek(off, io.SeekStart)
	if errors.Is(err, io.EOF) {
		f.Close()
		utils.Logger(ctx).Error("invalid file", "error", io.ErrUnexpectedEOF)
		return OpenFile(ctx, base, new(""), offset, b)
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, err
}
