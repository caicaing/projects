package taskflow

import (
	"context"
	"fmt"
	"time"
)

// Start begins scheduling. It may be called once; Close or cancellation of
// ctx permanently stops the engine. CRUD is available before Start, execution
// is not. An engine is not restartable; create a new engine after shutdown.
func (e *Engine) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("Context 不能为空")
	}
	if err := e.lockWrite(ctx); err != nil {
		return err
	}
	defer e.unlockWrite()
	if err := e.availabilityLocked(); err != nil {
		return err
	}
	if e.started {
		return ErrStarted
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s := e.state.Clone()
	now := time.Now()
	for i := range s.Tasks {
		if !s.Tasks[i].Enabled {
			continue
		}
		next, err := e.plans[s.Tasks[i].ID].Next(now)
		if err != nil {
			return err
		}
		s.Tasks[i].NextRun = &next
	}
	if err := e.commitLocked(ctx, s); err != nil {
		return err
	}
	if err := e.availabilityLocked(); err != nil {
		return err
	}
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.started = true
	e.wg.Add(1)
	go e.loop()
	return nil
}

// Close cancels scheduling and active operations, then waits for their final
// state writes. Failed writes return ErrStorage and can be retried with Close.
// A timeout does not stop cleanup; Close can be called again to wait and flush.
// The host may close storage only after Close has returned nil.
func (e *Engine) Close(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("Context 不能为空")
	}
	e.requestStop()
	select {
	case <-e.done:
		return e.flush(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Engine) requestStop() {
	e.stopOnce.Do(func() {
		close(e.stopping)
		go func() {
			e.mu.Lock()
			if e.cancel != nil {
				e.cancel()
			}
			e.mu.Unlock()
			e.wg.Wait()
			// Wait for already accepted definition/storage operations too.
			e.writes <- struct{}{}
			<-e.writes
			close(e.done)
		}()
	})
}

func (e *Engine) availabilityLocked() error {
	select {
	case <-e.stopping:
		return ErrClosed
	default:
	}
	if e.started && e.ctx.Err() != nil {
		return ErrClosed
	}
	return nil
}

func (e *Engine) executionReadyLocked() error {
	if err := e.availabilityLocked(); err != nil {
		return err
	}
	if !e.started {
		return ErrNotStarted
	}
	return nil
}

func (e *Engine) loop() {
	defer e.wg.Done()
	ticker := time.NewTicker(e.options.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			e.tick(now)
		case <-e.ctx.Done():
			e.requestStop()
			return
		}
	}
}
