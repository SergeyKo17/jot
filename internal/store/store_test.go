package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func testStore(t *testing.T) *Store {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	t.Helper()
	dir := t.TempDir()
	s, err := New(ctx, filepath.Join(dir, "test.db"), slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "empty path", path: "", wantErr: true},
		{name: "invalid path", path: filepath.Join(t.TempDir(), "no", "such", "dir", "test.db"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := New(ctx, tt.path, slog.Default())
			if (err != nil) != tt.wantErr {
				t.Errorf("New(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestSave(t *testing.T) {
	s := testStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name    string
		mem     Memory
		wantErr error
	}{
		{name: "basic fact", mem: Memory{Text: "Go uses gofmt"}, wantErr: nil},
		{name: "with project", mem: Memory{Text: "uses pgx", Project: "backend"}, wantErr: nil},
		{name: "with tags", mem: Memory{Text: "prefers tags", Tags: []string{"preference"}}, wantErr: nil},
		{name: "all fields", mem: Memory{Text: "test", Project: "backend", Source: "claude code", Tags: []string{"preference"}}, wantErr: nil},
		{name: "empty text", mem: Memory{Text: ""}, wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := s.Save(ctx, tt.mem)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && id == 0 {
				t.Errorf("Save() id = 0, want > 0")
			}
		})
	}

	t.Run("incrementing ids", func(t *testing.T) {
		id1, err := s.Save(ctx, Memory{Text: "first"})
		if err != nil {
			t.Fatal(err)
		}
		id2, err := s.Save(ctx, Memory{Text: "second"})
		if err != nil {
			t.Fatal(err)
		}
		if id2 <= id1 {
			t.Errorf("Save() ids not incrementing: id1=%d, id2=%d", id1, id2)
		}
	})
}

func TestSearch(t *testing.T) {
	s := testStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	seeds := []Memory{
		{Text: "Go uses gofmt for formatting", Project: "backend", Tags: []string{"tooling"}},
		{Text: "gofmt runs on save in IDE", Project: "frontend", Tags: []string{"workflow"}},
		{Text: "PostgreSQL preferred over MySQL", Project: "backend", Tags: []string{"database"}},
		{Text: "PostgreSQL supports JSONB", Project: "backend", Tags: []string{"database"}},
	}
	for _, m := range seeds {
		if _, err := s.Save(ctx, m); err != nil {
			t.Fatalf("seed Save(%q): %v", m.Text, err)
		}
	}

	tests := []struct {
		name     string
		input    SearchParams
		wantRows []MemoryRow
		wantErr  error
	}{
		{
			name:  "text only",
			input: SearchParams{Text: "gofmt"},
			wantRows: []MemoryRow{
				{Text: "Go uses gofmt for formatting", Project: "backend"},
				{Text: "gofmt runs on save in IDE", Project: "frontend"},
			},
		},
		{
			name:  "with project",
			input: SearchParams{Text: "gofmt", Project: "backend"},
			wantRows: []MemoryRow{
				{Text: "Go uses gofmt for formatting", Project: "backend"},
			},
		},
		{
			name:  "with tag",
			input: SearchParams{Text: "gofmt", Tag: "tooling"},
			wantRows: []MemoryRow{
				{Text: "Go uses gofmt for formatting", Project: "backend"},
			},
		},
		{
			name:  "project and tag",
			input: SearchParams{Text: "PostgreSQL", Project: "backend", Tag: "database"},
			wantRows: []MemoryRow{
				{Text: "PostgreSQL supports JSONB", Project: "backend"},
				{Text: "PostgreSQL preferred over MySQL", Project: "backend"},
			},
		},
		{
			name:     "no results",
			input:    SearchParams{Text: "Python"},
			wantRows: nil,
		},
		{
			name:     "project no match",
			input:    SearchParams{Text: "gofmt", Project: "mobile"},
			wantRows: nil,
		},
		{
			name:    "empty text",
			input:   SearchParams{Text: ""},
			wantErr: ErrEmptyQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.Search(ctx, tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Search() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(result) != len(tt.wantRows) {
				t.Fatalf("Search() got %d rows, want %d", len(result), len(tt.wantRows))
			}

			for i := range result {
				if result[i].Text != tt.wantRows[i].Text {
					t.Errorf("row[%d].Text = %q, want %q", i, result[i].Text, tt.wantRows[i].Text)
				}
				if result[i].Project != tt.wantRows[i].Project {
					t.Errorf("row[%d].Project = %q, want %q", i, result[i].Project, tt.wantRows[i].Project)
				}
			}
		})
	}
}

func TestSearchLimit(t *testing.T) {
	s := testStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := range 15 {
		_, err := s.Save(ctx, Memory{Text: fmt.Sprintf("fact number %d about Go", i)})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	result, err := s.Search(ctx, SearchParams{Text: "Go"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(result) != 10 {
		t.Errorf("Search() got %d rows, want 10", len(result))
	}
}

func TestDelete(t *testing.T) {
	s := testStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("existing", func(t *testing.T) {
		id, err := s.Save(ctx, Memory{Text: "to be deleted"})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Delete(ctx, id); err != nil {
			t.Fatalf("Delete(%d) = %v", id, err)
		}
		rows, err := s.Search(ctx, SearchParams{Text: "deleted"})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 0 {
			t.Errorf("Search after delete got %d rows, want 0", len(rows))
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := s.Delete(ctx, 999)
		if !errors.Is(err, ErrMemoryNotFound) {
			t.Errorf("Delete(999) = %v, want ErrMemoryNotFound", err)
		}
	})

	t.Run("zero id", func(t *testing.T) {
		err := s.Delete(ctx, 0)
		if !errors.Is(err, ErrInvalidID) {
			t.Errorf("Delete(0) = %v, want ErrDeleteEmptyID", err)
		}
	})

	t.Run("double delete", func(t *testing.T) {
		id, err := s.Save(ctx, Memory{Text: "double delete test"})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Delete(ctx, id); err != nil {
			t.Fatalf("first Delete(%d) = %v", id, err)
		}
		err = s.Delete(ctx, id)
		if !errors.Is(err, ErrMemoryNotFound) {
			t.Errorf("second Delete(%d) = %v, want ErrMemoryNotFound", id, err)
		}
	})

	t.Run("delete preserves others", func(t *testing.T) {
		id1, err := s.Save(ctx, Memory{Text: "keep this memory"})
		if err != nil {
			t.Fatal(err)
		}
		id2, err := s.Save(ctx, Memory{Text: "remove this memory"})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Delete(ctx, id2); err != nil {
			t.Fatalf("Delete(%d) = %v", id2, err)
		}
		rows, err := s.Search(ctx, SearchParams{Text: "memory"})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatalf("Search got %d rows, want 1", len(rows))
		}
		if rows[0].ID != id1 {
			t.Errorf("remaining row ID = %d, want %d", rows[0].ID, id1)
		}
	})
}

func ptr[T any](v T) *T { return &v }

func TestUpdate(t *testing.T) {
	s := testStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("update text", func(t *testing.T) {
		id, err := s.Save(ctx, Memory{Text: "old text", Project: "backend"})
		if err != nil {
			t.Fatal(err)
		}
		row, err := s.Update(ctx, UpdateParams{ID: id, Text: ptr("new text")})
		if err != nil {
			t.Fatalf("Update() = %v", err)
		}
		if row.Text != "new text" {
			t.Errorf("Text = %q, want %q", row.Text, "new text")
		}
		if row.Project != "backend" {
			t.Errorf("Project = %q, want %q", row.Project, "backend")
		}
	})

	t.Run("update project", func(t *testing.T) {
		id, err := s.Save(ctx, Memory{Text: "keep this", Project: "old"})
		if err != nil {
			t.Fatal(err)
		}
		row, err := s.Update(ctx, UpdateParams{ID: id, Project: ptr("new")})
		if err != nil {
			t.Fatalf("Update() = %v", err)
		}
		if row.Project != "new" {
			t.Errorf("Project = %q, want %q", row.Project, "new")
		}
		if row.Text != "keep this" {
			t.Errorf("Text = %q, want %q", row.Text, "keep this")
		}
	})

	t.Run("update tags", func(t *testing.T) {
		id, err := s.Save(ctx, Memory{Text: "tagged fact", Tags: []string{"old-tag"}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.Update(ctx, UpdateParams{ID: id, Tags: ptr([]string{"new-tag"})})
		if err != nil {
			t.Fatalf("Update() = %v", err)
		}
		rows, err := s.Search(ctx, SearchParams{Text: "tagged", Tag: "new-tag"})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Errorf("Search by new tag got %d rows, want 1", len(rows))
		}
		rows, err = s.Search(ctx, SearchParams{Text: "tagged", Tag: "old-tag"})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 0 {
			t.Errorf("Search by old tag got %d rows, want 0", len(rows))
		}
	})

	t.Run("update all fields", func(t *testing.T) {
		id, err := s.Save(ctx, Memory{Text: "original", Project: "old", Tags: []string{"v1"}})
		if err != nil {
			t.Fatal(err)
		}
		row, err := s.Update(ctx, UpdateParams{
			ID:      id,
			Text:    ptr("updated"),
			Project: ptr("new"),
			Tags:    ptr([]string{"v2"}),
		})
		if err != nil {
			t.Fatalf("Update() = %v", err)
		}
		if row.Text != "updated" {
			t.Errorf("Text = %q, want %q", row.Text, "updated")
		}
		if row.Project != "new" {
			t.Errorf("Project = %q, want %q", row.Project, "new")
		}
		rows, err := s.Search(ctx, SearchParams{Text: "updated", Tag: "v2"})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Errorf("Search by new tag got %d rows, want 1", len(rows))
		}
	})

	t.Run("empty text", func(t *testing.T) {
		id, err := s.Save(ctx, Memory{Text: "will not clear"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.Update(ctx, UpdateParams{ID: id, Text: ptr("")})
		if !errors.Is(err, ErrEmptyText) {
			t.Errorf("Update() = %v, want ErrEmptyText", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := s.Update(ctx, UpdateParams{ID: 999, Text: ptr("x")})
		if !errors.Is(err, ErrMemoryNotFound) {
			t.Errorf("Update(999) = %v, want ErrMemoryNotFound", err)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		_, err := s.Update(ctx, UpdateParams{ID: 0, Text: ptr("x")})
		if !errors.Is(err, ErrInvalidID) {
			t.Errorf("Update(0) = %v, want ErrInvalidID", err)
		}
	})

	t.Run("no fields", func(t *testing.T) {
		id, err := s.Save(ctx, Memory{Text: "no change"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.Update(ctx, UpdateParams{ID: id})
		if !errors.Is(err, ErrNoFieldsToUpdate) {
			t.Errorf("Update() = %v, want ErrNoFieldsToUpdate", err)
		}
	})
}
