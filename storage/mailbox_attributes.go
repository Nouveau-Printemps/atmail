package storage

import (
	"context"

	"github.com/emersion/go-imap/v2"
)

func GetMailboxAttributes(ctx context.Context, user, mailbox string) ([]imap.MailboxAttr, error) {
	return get(ctx, user, func(db *DB) ([]imap.MailboxAttr, error) {
		var attrs []imap.MailboxAttr
		ok, err := db.HasMailboxChildren(ctx, mailbox+string(MailboxSeparator)+"%")
		if err != nil {
			return nil, err
		}
		if ok {
			attrs = append(attrs, imap.MailboxAttrHasChildren)
		} else {
			attrs = append(attrs, imap.MailboxAttrHasNoChildren)
		}
		box, err := db.GetOrCreateMailbox(ctx, mailbox)
		if err != nil {
			return nil, err
		}
		switch box.ID {
		case JunkMailbox:
			attrs = append(attrs, imap.MailboxAttrJunk)
		case SentMailbox:
			attrs = append(attrs, imap.MailboxAttrSent)
		}
		return attrs, nil
	})
}
