package storage

import (
	"context"
	"database/sql"
	"embed"
	"path"
	"slices"

	"nouveauprintemps.org/atmail/utils"
)

//go:embed store/migrations
var migrations embed.FS

func migrate(ctx context.Context, db *sql.DB) error {
	l := utils.Logger(ctx).With("module", "db")
	entries, err := migrations.ReadDir("store/migrations")
	if err != nil {
		return err
	}
	migs := make([]string, 0, len(entries))
	for _, entry := range entries {
		migs = append(migs, entry.Name())
	}
	slices.Sort(migs)
	for _, mig := range migs {
		l.Debug("migrating", "mig", mig)
		b, err := migrations.ReadFile(path.Join("store/migrations", mig))
		if err != nil {
			return err
		}
		_, err = db.ExecContext(ctx, string(b))
		if err != nil {
			return err
		}
	}
	l.Debug("migrations finished")
	return nil
}
