package store

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"os"
)

// readEmailAt returns the email stored at in the file with the given offset.
// Returns nil if the email was deleted.
func readEmailAt(f *os.File, offset uint32) ([]byte, error) {
	var header [5]byte
	_, err := f.ReadAt(header[:], int64(offset))
	if err != nil {
		return nil, err
	}
	if header[0]&EmailDeleted != 0 {
		return nil, nil
	}
	ln := binary.BigEndian.Uint32(header[1:])
	b := make([]byte, ln)
	_, err = f.ReadAt(b, int64(offset)+int64(len(header)))
	if header[0]&EmailCompressed != 0 {
		slog.Debug("email compressed, decompressing")
		buf := bytes.NewBuffer(b)
		r, err := gzip.NewReader(buf)
		if err != nil {
			return nil, err
		}
		b, err = io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		r.Close()
	}
	return b, err
}

const (
	// 20MiB
	MaxFileSize = 20 * 1024 * 1024
	// 5MiB
	MaxEmailSizeInFile = 5 * 1024 * 1024
	// 1KiB
	CompressEmailBiggerThan = 1024
)

var ErrFileIsFull = errors.New("file is full")

const (
	EmailDeleted = 1 << iota
	EmailCompressed
)

func WriteEmail(f *os.File, b []byte) (uint32, error) {
	s, err := f.Stat()
	if err != nil {
		return 0, err
	}
	if s.Size() >= MaxFileSize {
		return 0, ErrFileIsFull
	}
	var n uint32
	header := make([]byte, 1, 5)
	if len(b) >= CompressEmailBiggerThan {
		slog.Debug("compressing email")
		header[0] |= EmailCompressed
		var buf bytes.Buffer
		w, _ := gzip.NewWriterLevel(&buf, 5)
		_, err := w.Write(b)
		if err != nil {
			return 0, err
		}
		w.Close()
		b = buf.Bytes()
	}
	p, err := f.Write(binary.BigEndian.AppendUint32(header, uint32(len(b))))
	if err != nil {
		return 0, err
	}
	n += uint32(p)
	p, err = f.Write(b)
	n += uint32(p)
	return uint32(n), err
}

func deleteEmailAt(f *os.File, offset uint32) error {
	var data [1]byte
	_, err := f.ReadAt(data[:], int64(offset))
	if err != nil {
		return err
	}
	data[0] |= EmailDeleted
	_, err = f.WriteAt(data[:], int64(offset))
	return err
}
