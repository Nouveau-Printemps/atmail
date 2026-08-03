package mailbox

import (
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

	listening atomic.Bool

	messages     chan uint32
	expunge      chan uint32
	mailboxFlags chan []imap.Flag
	messageFlags chan MessageUpdate
}

func NewView(id uint32, name string) *View {
	return &View{
		ID:           id,
		Name:         name,
		messages:     make(chan uint32, 1),
		expunge:      make(chan uint32, 1),
		mailboxFlags: make(chan []imap.Flag, 1),
		messageFlags: make(chan MessageUpdate, 1),
	}
}

func (v *View) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	if v.listening.Swap(true) {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "already listening",
		}
	}
	defer v.listening.Store(false)
	for {
		var err error
		select {
		case <-stop:
			return nil
		case n := <-v.messages:
			err = w.WriteNumMessages(n)
		case seq := <-v.expunge:
			err = w.WriteExpunge(seq)
		case flags := <-v.mailboxFlags:
			err = w.WriteMailboxFlags(flags)
		case up := <-v.messageFlags:
			err = w.WriteMessageFlags(up.Seq, imap.UID(up.ID), up.Flags)
		}
		if err != nil {
			return err
		}
	}
}

// WriteNewMessages increases the [View.Count] by n and send its updated value.
func (v *View) WriteNewMessages(n uint32) {
	if v.listening.Load() {
		v.messages <- v.Count.Add(n)
	}
}

func (v *View) WriteExpunge(seq uint32) {
	if v.listening.Load() {
		v.expunge <- seq
	}
}

func (v *View) WriteMailboxFlags(flags []imap.Flag) {
	if v.listening.Load() {
		v.mailboxFlags <- flags
	}
}

func (v *View) WriteMessageFlags(id, seq uint32, flags []imap.Flag) {
	if v.listening.Load() {
		v.messageFlags <- MessageUpdate{
			ID:    id,
			Seq:   seq,
			Flags: flags,
		}
	}
}
