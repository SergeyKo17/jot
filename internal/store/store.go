// Package store provides SQLite storage for memories.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/SergeyKo17/jot/migrations"
	"github.com/pressly/goose/v3"
)

// Store manages SQLite database connections and queries.
type Store struct {
	db     *sql.DB
	logger *slog.Logger
}

// New opens a SQLite database at path and runs migrations.
func New(ctx context.Context, path string, logger *slog.Logger) (_ *Store, err error) {
	if path == "" {
		return nil, fmt.Errorf("db path is empty")
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, db.Close())
		}
	}()

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		migrations.FS,
	)
	if err != nil {
		return nil, fmt.Errorf("goose provider: %w", err)
	}

	_, err = provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrations up: %w", err)
	}

	return &Store{db: db, logger: logger}, nil
}

// Close closes db connection.
func (s *Store) Close() {
	if err := s.db.Close(); err != nil {
		s.logger.Error("close db", "err", err)
	}
}

// Memory represents a fact to store.
type Memory struct {
	Text    string
	Project string
	Source  string
	Tags    []string
}

const saveQuery = `
INSERT INTO memories (text, project, tags, source)
VALUES (?, ?, ?, ?)
RETURNING id`

// Save inserts a new memory and returns its ID.
func (s *Store) Save(ctx context.Context, mem Memory) (int64, error) {
	if mem.Tags == nil {
		mem.Tags = []string{}
	}
	tagsJSON, err := json.Marshal(mem.Tags)
	if err != nil {
		return 0, fmt.Errorf("marshal tags: %w", err)
	}

	var id int64
	err = s.db.QueryRowContext(ctx, saveQuery,
		mem.Text,
		mem.Project,
		tagsJSON,
		mem.Source).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert memory: %w", err)
	}

	return id, nil
}
