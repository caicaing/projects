// Package memory provides a non-persistent Store for embedded deployments.
package memory

import (
	"context"
	"fmt"
	"sync"

	"git.woa.com/mooyliu/mooy_test/framework"
)

// Store retains state in memory; there are no resources to close.
type Store struct {
	mu    sync.Mutex
	state taskflow.State
}

func New() *Store { return &Store{state: taskflow.State{Version: 1}} }

func (store *Store) Load(ctx context.Context) (taskflow.State, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return taskflow.State{}, err
	}
	return store.state.Clone(), nil
}

func (store *Store) Save(ctx context.Context, state taskflow.State) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if state.Version != 1 {
		return fmt.Errorf("不支持的存储版本: %d", state.Version)
	}
	store.state = state.Clone()
	return nil
}
