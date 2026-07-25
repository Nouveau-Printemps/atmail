package storage

import (
	"encoding/binary"
	"errors"
	"os"
)

// ReadEmail returns the email stored at the beginning of the given bytes.
// Returns nil if the email was deleted.
func ReadEmail(b []byte) []byte {
	if b[0]&EmailDeleted == 1 {
		return nil
	}
	return b[5 : 5+binary.BigEndian.Uint32(b[1:])]
}

// ReadEmailAt the given position in the given bytes.
// See [ReadEmail].
func ReadEmailAt(b []byte, offset uint32) []byte {
	return ReadEmail(b[offset:])
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
