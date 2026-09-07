// Package store provides SQLite storage for memories.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/SergeyKo17/jot/migrations"
	"github.com/pressly/goose/v3"
)

// ErrEmptyQuery is returned when search text is empty.
var ErrEmptyQuery = errors.New("search: text is required")

// ErrInvalidID is returned when memory ID is zero or negative.
var ErrInvalidID = errors.New("invalid memory ID")

// ErrMemoryNotFound is returned when memory doesn't exist.
var ErrMemoryNotFound = errors.New("memory not found")

// ErrNoFieldsToUpdate is returned when no fields are provided for update.
var ErrNoFieldsToUpdate = errors.New("update: no fields to update")

// ErrEmptyText is returned when text is set to empty string.
var ErrEmptyText = errors.New("text cannot be empty")

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

// SearchParams holds search parameters for querying memories.
type SearchParams struct {
	Text    string
	Project string
	Tag     string
}

// MemoryRow represents a single search result.
type MemoryRow struct {
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	Project   string `json:"project,omitempty"`
	CreatedAt string `json:"created_at"`
}

// Search queries memories by text match with optional project and tag filters.
func (s *Store) Search(ctx context.Context, params SearchParams) (_ []MemoryRow, err error) {
	if params.Text == "" {
		return nil, ErrEmptyQuery
	}

	query, args := createRecallQuery(params)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	var result []MemoryRow
	for rows.Next() {
		var r MemoryRow
		if err := rows.Scan(&r.ID, &r.Text, &r.Project, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("search scan: %w", err)
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search rows: %w", err)
	}

	return result, nil
}

// createRecallQuery builds a FTS5 search query with optional WHERE clauses for project and tag.
func createRecallQuery(params SearchParams) (string, []any) {
	query := `SELECT m.id, m.text, m.project, m.created_at
	FROM memories_fts f
	JOIN memories m ON m.rowid = f.rowid
	WHERE f.text MATCH ?`
	args := []any{params.Text}

	if params.Project != "" {
		query += " AND m.project = ?"
		args = append(args, params.Project)
	}

	if params.Tag != "" {
		query += ` AND m.id IN 
		(SELECT mm.id FROM memories mm, JSON_EACH(mm.tags) 
			WHERE JSON_EACH.value = ?)`
		args = append(args, params.Tag)
	}

	query += " ORDER BY f.rank LIMIT 10"

	return query, args
}

const deleteQuery = `
DELETE FROM memories
WHERE id = ?`

// Delete removes a memory by ID.
func (s *Store) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidID
	}
	result, err := s.db.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}

	countMems, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete rows affected: %w", err)
	}
	if countMems == 0 {
		return ErrMemoryNotFound
	}
	return nil
}

// UpdateParams holds fields to update. Nil means don't touch.
type UpdateParams struct {
	ID      int64
	Text    *string
	Project *string
	Tags    *[]string
}

// Update modifies an existing memory's fields by ID.
func (s *Store) Update(ctx context.Context, params UpdateParams) (MemoryRow, error) {
	if params.ID <= 0 {
		return MemoryRow{}, ErrInvalidID
	}

	query, args, err := createUpdateQuery(params)
	if err != nil {
		return MemoryRow{}, err
	}

	var r MemoryRow
	err = s.db.QueryRowContext(ctx, query, args...).Scan(&r.ID, &r.Text, &r.Project, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return MemoryRow{}, ErrMemoryNotFound
	}
	if err != nil {
		return MemoryRow{}, fmt.Errorf("update memory: %w", err)
	}

	return r, nil
}

// createUpdateQuery builds a dynamic UPDATE query from non-nil fields.
func createUpdateQuery(params UpdateParams) (string, []any, error) {
	sets := make([]string, 0, 3)
	args := make([]any, 0, 4)

	if params.Text != nil && *params.Text == "" {
		return "", nil, ErrEmptyText
	}

	if params.Text != nil {
		sets = append(sets, "text = ?")
		args = append(args, *params.Text)
	}
	if params.Project != nil {
		sets = append(sets, "project = ?")
		args = append(args, *params.Project)
	}
	if params.Tags != nil {
		tagsJSON, err := json.Marshal(*params.Tags)
		if err != nil {
			return "", nil, fmt.Errorf("marshal tags: %w", err)
		}
		sets = append(sets, "tags = ?")
		args = append(args, tagsJSON)
	}

	if len(sets) == 0 {
		return "", nil, ErrNoFieldsToUpdate
	}

	query := "UPDATE memories SET " + strings.Join(sets, ", ") +
		", updated_at = datetime('now') WHERE id = ? RETURNING id, text, project, created_at"
	args = append(args, params.ID)

	return query, args, nil
}
