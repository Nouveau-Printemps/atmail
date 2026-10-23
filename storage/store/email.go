package store

import (
	"io"
	"os"
	"path"
)

func (e *Email) openFile(base string) (*os.File, error) {
	return os.OpenFile(path.Join(base, e.Filename), os.O_RDWR, 0o660)
}

func (e *Email) Read(base string) (io.ReadCloser, error) {
	f, err := e.openFile(base)
	if err != nil {
		return nil, err
	}
	return e.Reader(f)
}

func (e *Email) Reader(f *os.File) (io.ReadCloser, error) {
	return Reader(f, uint32(e.Offset))
}

func (e *Email) Delete(base string) error {
	f, err := e.openFile(base)
	if err != nil {
		return err
	}
	defer f.Close()
	return deleteEmailAt(f, uint32(e.Offset))
}
