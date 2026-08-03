package storage

import (
	"context"
	"database/sql"
	"errors"
	"path"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"nouveauprintemps.org/atmail/storage/index"
)

type DB = index.Queries

var Cache = &Meta{subs: make(map[string]*meta)}

type Meta struct {
	mu         sync.Mutex
	Path       string
	Migrations string
	subs       map[string]*meta
}

func (m *Meta) Close(ctx context.Context) error {
	m.mu.Lock()
	var err error
	for _, v := range m.subs {
		e := v.Save(ctx)
		if e != nil {
			if err == nil {
				e = err
			} else {
				err = errors.Join(err, e)
			}
		}
	}
	m.subs = nil
	return err
}

func (m *Meta) DB(ctx context.Context, user string) (*meta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.subs[user]; ok {
		return v, nil
	}
	p := path.Join(m.Path, user) + "/"
	db, err := sql.Open(
		"sqlite3",
		"file:"+p+"database.db?_journal=WAL&_foreign_keys=1&_synch=normal&_timeout=5000",
	)
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(ctx, `PRAGMA wal_checkpoint(PASSIVE)`)
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(ctx, m.Migrations)
	if err != nil {
		return nil, err
	}
	mm := &meta{db: db, path: p}
	err = mm.Sync(ctx)
	if err != nil {
		return nil, err
	}
	m.subs[user] = mm
	return mm, nil
}

type meta struct {
	mu       sync.RWMutex
	path     string
	lastFile string
	offset   uint32
	db       *sql.DB
}

func (m *meta) LastFile() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastFile
}

func (m *meta) Offset() uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.offset
}

func (m *meta) Update(lastFile string, offset uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastFile = lastFile
	m.offset = offset
}

func (m *meta) Sync(ctx context.Context) error {
	meta, err := index.New(m.db).GetMeta(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastFile = meta.LastFile.String
	m.offset = uint32(meta.Offset.Int64)
	return nil
}

func (m *meta) Save(ctx context.Context) error {
	return index.New(m.db).SetMeta(
		ctx,
		0,
		sql.NullString{String: m.lastFile, Valid: true},
		sql.NullInt64{Int64: int64(m.offset), Valid: true},
	)
}

func exec(ctx context.Context, user string, fn func(*DB) error) error {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return err
	}
	return fn(index.New(meta.db))
}

func execTx(ctx context.Context, user string, fn func(*DB) error) error {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return err
	}
	tx, err := meta.db.BeginTx(ctx, nil)
	err = fn(index.New(tx))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func get[T any](ctx context.Context, user string, fn func(*DB) (T, error)) (T, error) {
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		var v T
		return v, err
	}
	return fn(index.New(meta.db))
}
