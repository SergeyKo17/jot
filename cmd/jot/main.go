// Package main is the entry point for jot.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/SergeyKo17/jot/internal/server"
	"github.com/SergeyKo17/jot/internal/store"
	"github.com/SergeyKo17/jot/internal/tools"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	base, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home dir: %w", err)
	}

	dbDir := filepath.Join(base, ".jot")
	if err := os.MkdirAll(dbDir, 0o700); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}

	dbPath := filepath.Join(dbDir, "facts.db")
	store, err := store.New(ctx, dbPath, logger)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()

	t := tools.New(store, logger)

	srv := server.New(logger, t)
	return srv.Run(context.Background())
}
