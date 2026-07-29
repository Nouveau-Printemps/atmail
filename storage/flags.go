package storage

import (
	"context"

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
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	in := index.New(meta.db)
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
}

func ListEmailFlags(ctx context.Context, user string, email int64) ([]index.Flag, error) {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	return index.New(meta.db).ListEmailFlags(ctx, email)
}

func AddEmailFlag(ctx context.Context, user string, email, flag int64) error {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return err
	}
	return index.New(meta.db).AddEmailFlag(ctx, email, flag)
}
