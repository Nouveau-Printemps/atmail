package storage

import (
	"context"

	"github.com/emersion/go-imap/v2"
	"nouveauprintemps.org/atmail/storage/index"
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

func DeleteEmailsWithFlag(ctx context.Context, user string, flag int64) ([]index.Email, error) {
	return get(ctx, user, func(in *DB) ([]index.Email, error) {
		emails, err := in.ListEmailsWithFlag(ctx, flag)
		if err != nil {
			return nil, err
		}
		for _, email := range emails {
			err = DeleteEmail(user, email)
			if err != nil {
				return nil, err
			}
		}
		return emails, in.RemoveEmailsWithFlag(ctx, flag)
	})
}

func ListEmailFlags(ctx context.Context, user string, email int64) ([]index.Flag, error) {
	return get(ctx, user, func(in *DB) ([]index.Flag, error) {
		return in.ListEmailFlags(ctx, email)
	})
}

func AddEmailFlag(ctx context.Context, user string, email, flag int64) error {
	return exec(ctx, user, func(in *DB) error {
		return in.AddEmailFlag(ctx, email, flag)
	})
}

func RemoveEmailAllFlags(ctx context.Context, user string, email int64) error {
	return exec(ctx, user, func(in *DB) error {
		return in.RemoveEmailFlags(ctx, email)
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
		return nil
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
		return nil
	})
}
