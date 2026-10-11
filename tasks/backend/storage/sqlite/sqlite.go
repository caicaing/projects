// Package sqlite provides the optional single-host SQLite Store adapter.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	taskflow "git.woa.com/mooyliu/mooy_test/framework"

	_ "modernc.org/sqlite"
)

// Store owns its database connection and file lock. The host must close it
// after the engine has finished shutting down. Do not share it across engines.
type Store struct {
	mu       sync.Mutex
	db       *sql.DB
	current  map[string]map[string][]any
	unlock   func()
	closed   bool
	closeErr error
}

func emptyState() taskflow.State {
	return taskflow.State{Version: 1, Tasks: []taskflow.Task{},
		Channels: []taskflow.Channel{}, Runs: []taskflow.Run{}}
}

// Open opens or initializes a database; it does not start an engine. It keeps
// the existing Version 1 schema and never converts paths or imports JSON.
func Open(ctx context.Context, path string) (*Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("SQLite 路径不能为空")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	unlock, err := lockFile(path)
	if err != nil {
		return nil, err
	}
	store := &Store{unlock: unlock}
	fail := func(err error) (*Store, error) {
		_ = store.Close()
		return nil, err
	}
	// Configurations can contain secrets; newly created files are private.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fail(err)
	}
	if err := file.Close(); err != nil {
		return fail(err)
	}
	u := url.URL{Scheme: "file", Path: path}
	store.db, err = sql.Open("sqlite", u.String())
	if err != nil {
		return fail(err)
	}
	store.db.SetMaxOpenConns(1)
	if err := store.initialize(ctx); err != nil {
		return fail(err)
	}
	return store, nil
}

func (store *Store) initialize(ctx context.Context) error {
	pragmas := "PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;"
	if _, err := store.db.ExecContext(ctx, pragmas); err != nil {
		return err
	}
	var version int
	if err := store.db.QueryRowContext(ctx,
		"PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version != 0 && version != 1 {
		return fmt.Errorf("不支持的数据库版本: %d", version)
	}
	if _, err := store.db.ExecContext(ctx, storeSchema); err != nil {
		return err
	}
	s := emptyState()
	if err := store.load(ctx, &s); err != nil {
		return err
	}
	var err error
	store.current, err = stateRecords(s)
	if err != nil {
		return err
	}
	_, err = store.db.ExecContext(ctx, "PRAGMA user_version = 1")
	return err
}

// Load returns an independent state snapshot and refreshes the diff baseline.
func (store *Store) Load(ctx context.Context) (taskflow.State, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	s := emptyState()
	if store.closed {
		return s, fmt.Errorf("SQLite 存储已关闭")
	}
	if err := store.load(ctx, &s); err != nil {
		return s, err
	}
	current, err := stateRecords(s)
	if err != nil {
		return s, err
	}
	store.current = current
	return s, nil
}

// Close is idempotent and releases the file lock, including on open failure.
func (store *Store) Close() error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.closed {
		return store.closeErr
	}
	store.closed = true
	if store.db != nil {
		store.closeErr = store.db.Close()
	}
	store.unlock()
	return store.closeErr
}
