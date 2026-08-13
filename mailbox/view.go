package mailbox

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

type MessageUpdate struct {
	ID    uint32
	Seq   uint32
	Flags []imap.Flag
}

type View struct {
	ID    uint32
	Name  string
	Count atomic.Uint32

	writers map[*imapserver.UpdateWriter]chan error
	mu      sync.RWMutex
}

func NewView(id uint32, name string, count uint32) *View {
	v := &View{
		ID:      id,
		Name:    name,
		writers: make(map[*imapserver.UpdateWriter]chan error),
	}
	v.Count.Store(count)
	return v
}

func (v *View) Idle(ctx context.Context, w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	errc := make(chan error)
	v.mu.Lock()
	v.writers[w] = errc
	v.mu.Unlock()
	defer func() {
		v.mu.Lock()
		delete(v.writers, w)
		v.mu.Unlock()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-stop:
		return nil
	case err := <-errc:
		return err
	}
}

// WriteNewMessages increases the [View.Count] by n and send its updated value.
func (v *View) WriteNewMessages(n uint32) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	for w, errc := range v.writers {
		err := w.WriteNumMessages(v.Count.Add(n))
		if err != nil {
			errc <- err
		}
	}
}

func (v *View) WriteExpunge(seq uint32) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	for w, errc := range v.writers {
		err := w.WriteExpunge(seq)
		if err != nil {
			errc <- err
		}
	}
}

func (v *View) WriteMailboxFlags(flags []imap.Flag) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	for w, errc := range v.writers {
		err := w.WriteMailboxFlags(flags)
		if err != nil {
			errc <- err
		}
	}
}

func (v *View) WriteMessageFlags(id, seq uint32, flags []imap.Flag) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	for w, errc := range v.writers {
		err := w.WriteMessageFlags(seq, imap.UID(id), flags)
		if err != nil {
			errc <- err
		}
	}
}
