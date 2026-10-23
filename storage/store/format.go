package store

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"os"
)

type readCloser struct {
	r io.Reader
	c io.Closer
}

func (rc *readCloser) Read(b []byte) (int, error) {
	return rc.r.Read(b)
}

func (rc *readCloser) Close() error {
	return rc.c.Close()
}

type readMultiCloser struct {
	rc io.ReadCloser
	c  io.Closer
}

func (rc *readMultiCloser) Read(b []byte) (int, error) {
	return rc.rc.Read(b)
}

func (rc *readMultiCloser) Close() error {
	err := rc.rc.Close()
	if err != nil {
		return err
	}
	return rc.c.Close()
}

// Reader returns an [EmailReader] for the given email.
// Returns nil if the email was deleted.
func Reader(f *os.File, offset uint32) (io.ReadCloser, error) {
	_, err := f.Seek(int64(offset), 0)
	if err != nil {
		return nil, err
	}
	var header [5]byte
	_, err = io.ReadFull(f, header[:])
	if err != nil {
		return nil, err
	}
	if header[0]&EmailDeleted != 0 {
		return nil, nil
	}
	ln := binary.BigEndian.Uint32(header[1:])
	var rc io.ReadCloser = &readCloser{
		r: io.LimitReader(f, int64(ln)),
		c: f,
	}
	if header[0]&EmailCompressed != 0 {
		r, err := gzip.NewReader(bufio.NewReader(rc))
		if err != nil {
			return nil, err
		}
		rc = &readMultiCloser{rc: r, c: f}
	}
	return rc, nil
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
	EmailEncrypted
)

func WriteEmail(f *os.File, encrypted bool, b []byte) (uint32, error) {
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
	if encrypted {
		header[0] |= EmailEncrypted
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
