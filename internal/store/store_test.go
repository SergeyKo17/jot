package store

import (
	"context"
	"errors"
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
			id, err := s.Save(context.Background(), tt.mem)
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
		id1, err := s.Save(context.Background(), Memory{Text: "first"})
		if err != nil {
			t.Fatal(err)
		}
		id2, err := s.Save(context.Background(), Memory{Text: "second"})
		if err != nil {
			t.Fatal(err)
		}
		if id2 <= id1 {
			t.Errorf("Save() ids not incrementing: id1=%d, id2=%d", id1, id2)
		}
	})
}
