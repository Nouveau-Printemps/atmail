package storage

import (
	"context"

	"github.com/emersion/go-imap/v2"
	"nouveauprintemps.org/atmail/storage/store"
)

const (
	SeenFlag int64 = iota + 1
	AnsweredFlag
	FlaggedFlag
	DeletedFlag
	DraftFlag
	ForwardedFlag
	MDNSentFlag
	JunkFlag
	NotJunkFlag
	PhishingFlag
)

func DeleteEmailsWithFlag(ctx context.Context, user string, flag int64) ([]store.Email, error) {
	return get(ctx, user, func(in *DB) ([]store.Email, error) {
		emails, err := in.ListEmailsWithFlag(ctx, flag)
		if err != nil {
			return nil, err
		}
		return emails, in.RemoveEmailsWithFlag(ctx, flag)
	})
}

func ListEmailFlags(ctx context.Context, user string, email int64) ([]store.Flag, error) {
	return get(ctx, user, func(in *DB) ([]store.Flag, error) {
		return in.ListEmailFlags(ctx, email)
	})
}

func AddEmailFlag(ctx context.Context, user string, email, flag int64) error {
	return execTx(ctx, user, func(in *DB) error {
		err := in.AddEmailFlag(ctx, email, flag)
		if err != nil {
			return err
		}
		return in.IncrementEmailsModseq(ctx, email)
	})
}

func RemoveEmailAllFlags(ctx context.Context, user string, email int64) error {
	return execTx(ctx, user, func(in *DB) error {
		err := in.RemoveEmailFlags(ctx, email)
		if err != nil {
			return err
		}
		return in.IncrementEmailsModseq(ctx, email)
	})
}

func AddEmailFlags(ctx context.Context, user string, email int64, flags []imap.Flag) error {
	return execTx(ctx, user, func(in *DB) error {
		for _, flag := range flags {
			err := in.AddEmailFlagName(ctx, email, string(flag))
			if err != nil {
				return err
			}
		}
		return in.IncrementEmailsModseq(ctx, email)
	})
}

func RemoveEmailFlags(ctx context.Context, user string, email int64, flags []imap.Flag) error {
	return execTx(ctx, user, func(in *DB) error {
		for _, flag := range flags {
			err := in.RemoveEmailFlagName(ctx, email, string(flag))
			if err != nil {
				return err
			}
		}
		return in.IncrementEmailsModseq(ctx, email)
	})
}
