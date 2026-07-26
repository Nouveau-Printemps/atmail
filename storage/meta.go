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

func (m *Meta) PathOf(user string, file string) string {
	return path.Join(m.Path, user, file)
}

func (m *Meta) DB(ctx context.Context, user string) (*meta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.subs[user]; ok {
		return v, nil
	}

	db, err := sql.Open(
		"sqlite3",
		"file:"+m.PathOf(user, "database.db")+"?_journal=WAL",
	)
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(ctx, m.Migrations)
	if err != nil {
		return nil, err
	}
	mm := &meta{index: index.New(db)}
	err = mm.Sync(ctx)
	if err != nil {
		return nil, err
	}
	m.subs[user] = mm
	return mm, nil
}

type meta struct {
	mu       sync.RWMutex
	lastFile string
	offset   uint32
	index    *Index
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
	meta, err := m.index.GetMeta(ctx)
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
	return m.index.SetMeta(
		ctx,
		0,
		sql.NullString{String: m.lastFile, Valid: true},
		sql.NullInt64{Int64: int64(m.offset), Valid: true},
	)
}
