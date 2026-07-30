package index

import (
	"os"
	"path"
)

func (e *Email) openFile(base string, flag int) (*os.File, error) {
	return os.OpenFile(path.Join(base, e.Filename), flag|os.O_RDWR, 0o660)
}

func (e *Email) Read(base string) ([]byte, error) {
	f, err := e.openFile(base, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return e.ReadFile(f)
}

func (e *Email) ReadFile(f *os.File) ([]byte, error) {
	return readEmailAt(f, uint32(e.Offset))
}

func (e *Email) Delete(base string) error {
	f, err := e.openFile(base, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	return deleteEmailAt(f, uint32(e.Offset))
}
