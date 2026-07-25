package storage

import (
	"context"
	"database/sql"
	"errors"
	"sync"
)

var Cache = &meta{}

type meta struct {
	mu       sync.RWMutex
	lastFile string
	offset   uint32
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

func (m *meta) Sync(ctx context.Context, index *Index) error {
	meta, err := index.GetMeta(ctx)
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

func (m *meta) Save(ctx context.Context, index *Index) error {
	return index.SetMeta(
		ctx,
		0,
		sql.NullString{String: m.lastFile, Valid: true},
		sql.NullInt64{Int64: int64(m.offset), Valid: true},
	)
}
