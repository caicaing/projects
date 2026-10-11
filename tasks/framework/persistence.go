package taskflow

import (
	"context"
	"fmt"
)

// Writers are serialized separately from reads, cancellation and shutdown.
// Successful lockWrite holds both the write token and the state mutex.
func (e *Engine) lockWrite(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("Context 不能为空")
	}
	select {
	case e.writes <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	e.mu.Lock()
	if err := ctx.Err(); err != nil {
		e.unlockWrite()
		return err
	}
	return nil
}

func (e *Engine) unlockWrite() {
	e.mu.Unlock()
	<-e.writes
}

// The write token stays held while Save runs, but the state mutex does not.
func (e *Engine) persistLocked(ctx context.Context, s State) error {
	e.mu.Unlock()
	defer e.mu.Lock()
	storeCtx, cancel := context.WithTimeout(ctx, e.options.StorageTimeout)
	defer cancel()
	if err := e.options.Store.Save(storeCtx, s); err != nil {
		return fmt.Errorf("%w: %w", ErrStorage, err)
	}
	return nil
}

// Failed runtime writes remain in memory until a later commit or Close retry.
func (e *Engine) flush(ctx context.Context) error {
	if err := e.lockWrite(ctx); err != nil {
		return err
	}
	defer e.unlockWrite()
	if !e.dirty {
		return nil
	}
	return e.commitLocked(ctx, e.state.Clone())
}
