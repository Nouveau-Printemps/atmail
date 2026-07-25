package storage

import (
	"encoding/binary"
	"errors"
	"os"
)

// ReadEmailAt returns the email stored at in the file with the given offset.
// Returns nil if the email was deleted.
func ReadEmailAt(f *os.File, offset uint32) ([]byte, error) {
	var header [5]byte
	_, err := f.ReadAt(header[:], int64(offset))
	if err != nil {
		return nil, err
	}
	if header[0]&EmailDeleted == 1 {
		return nil, nil
	}
	ln := binary.BigEndian.Uint32(header[1:])
	b := make([]byte, ln)
	_, err = f.ReadAt(b, int64(offset)+int64(len(header)))
	return b, err
}

const (
	// 20MB
	MaxFileSize = 20 * 1024 * 1024
	// 5MB
	MaxEmailSizeInFile = 5 * 1024 * 1024
)

var ErrFileIsFull = errors.New("file is full")

const (
	EmailDeleted = 1 << iota
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
	// if is stored in file
	if len(b) < MaxEmailSizeInFile {
		size := binary.BigEndian.AppendUint32(make([]byte, 1, 5), uint32(len(b)))
		p, err := f.Write(size)
		if err != nil {
			return 0, err
		}
		n += uint32(p)
	}
	p, err := f.Write(b)
	n += uint32(p)
	return uint32(n), err
}

func DeleteEmailAt(b []byte, offset uint32) {
	b[offset] |= EmailDeleted
}
