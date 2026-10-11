package taskflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (e *Engine) Trigger(ctx context.Context, id string) (Run, error) {
	if err := e.lockWrite(ctx); err != nil {
		return Run{}, err
	}
	defer e.unlockWrite()
	if err := e.executionReadyLocked(); err != nil {
		return Run{}, err
	}
	for _, t := range e.state.Tasks {
		if t.ID == id {
			return e.startLocked(ctx, t, "manual", time.Now())
		}
	}
	return Run{}, ErrNotFound
}

func (e *Engine) CancelRun(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.executionReadyLocked(); err != nil {
		return err
	}
	if cancel, ok := e.active[id]; ok {
		cancel()
		return nil
	}
	return ErrNotFound
}

func (e *Engine) startLocked(storeCtx context.Context, t Task, trigger string,
	now time.Time) (Run, error) {
	busy := e.runningTasks[t.ID] ||
		len(e.runningTasks) >= e.options.Workers
	if busy && trigger == "manual" {
		return Run{}, ErrBusy
	}
	var handler TaskHandler
	if !busy {
		var err error
		handler, err = e.createTaskLocked(t.Type)
		if err != nil {
			return Run{}, err
		}
		if err := e.executionReadyLocked(); err != nil {
			return Run{}, err
		}
	}
	s := e.state.Clone()
	if trigger == "schedule" {
		next, err := e.plans[t.ID].Next(now)
		if err != nil {
			return Run{}, err
		}
		for i := range s.Tasks {
			if s.Tasks[i].ID == t.ID {
				s.Tasks[i].NextRun = &next
			}
		}
	}
	r := Run{
		ID: newID(), TaskID: t.ID, TaskName: t.Name,
		Trigger: trigger, Status: "running", StartedAt: now,
		Deliveries: []Delivery{},
	}
	var ctx context.Context
	var cancel context.CancelFunc
	if busy {
		r.Status, r.Error = "skipped", ErrBusy.Error()
		r.FinishedAt = &now
	} else {
		ctx, cancel = context.WithCancel(e.ctx)
		e.active[r.ID] = cancel
		e.runningTasks[t.ID] = true
		// Reserve before Save releases the mutex so shutdown waits for us.
		e.wg.Add(1)
	}
	s.Runs = append([]Run{r}, s.Runs...)
	if err := e.commitLocked(storeCtx, s); err != nil {
		if !busy {
			cancel()
			delete(e.active, r.ID)
			delete(e.runningTasks, t.ID)
			e.wg.Done()
		}
		return Run{}, err
	}
	if !busy {
		go e.execute(ctx, cancel, handler, t.Clone(), r, s.Channels)
	}
	return r.Clone(), nil
}

func (e *Engine) createTaskLocked(kind string) (TaskHandler, error) {
	e.mu.Unlock()
	defer e.mu.Lock()
	return e.registry.Tasks.Create(kind)
}

func (e *Engine) execute(ctx context.Context, cancel context.CancelFunc,
	handler TaskHandler, t Task, r Run, channels []Channel) {
	defer e.wg.Done()
	defer cancel()
	defer func() {
		e.mu.Lock()
		delete(e.active, r.ID)
		e.mu.Unlock()
	}()
	r = e.runTask(ctx, handler, t, r)
	e.mu.Lock()
	delete(e.runningTasks, t.ID)
	e.mu.Unlock()
	e.recordRun(r)
	if t.NotifyOn == "always" ||
		(t.NotifyOn == "failure" && r.Status == "failed") {
		r.Deliveries = e.notify(ctx, t, r, channels)
		e.recordRun(r)
	}
}

func (e *Engine) runTask(parent context.Context, handler TaskHandler,
	t Task, r Run) Run {
	timeout, _ := time.ParseDuration(t.Timeout)
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	delay, _ := time.ParseDuration(t.RetryDelay)
	var err error
	for attempt := 0; attempt <= t.Retries; attempt++ {
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
		r.Attempts++
		r.Output, err = safeRun(handler, ctx, t.Config)
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		r.Output = truncate(r.Output, 65536)
		if err == nil || attempt == t.Retries || ctx.Err() != nil {
			break
		}
		e.recordRun(r)
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
		}
	}
	now := time.Now()
	r.FinishedAt = &now
	r.DurationMS = now.Sub(r.StartedAt).Milliseconds()
	r.Status = "succeeded"
	if err != nil {
		r.Status, r.Error = "failed", truncate(err.Error(), 8192)
		if ctx.Err() == context.Canceled {
			r.Status = "canceled"
		}
	}
	return r
}

func (e *Engine) recordRun(r Run) {
	// Final records must survive runtime-context cancellation.
	ctx := context.WithoutCancel(e.ctx)
	if err := e.lockWrite(ctx); err != nil {
		return
	}
	s := e.state.Clone()
	var saveErr error
	for i := range s.Runs {
		if s.Runs[i].ID == r.ID {
			s.Runs[i] = r.Clone()
			saveErr = e.commitLocked(ctx, s)
			if saveErr != nil {
				e.state, e.dirty = s, true
			}
			break
		}
	}
	e.unlockWrite()
	if saveErr != nil {
		e.options.Logger.ErrorContext(ctx, "保存执行记录失败",
			"run", r.ID, "error", saveErr)
	}
}

func (e *Engine) notify(ctx context.Context, t Task,
	r Run, channels []Channel) []Delivery {
	results := []Delivery{}
	n := Notification{
		Title:   fmt.Sprintf("[%s] %s", r.Status, t.Name),
		Message: r.Output, Run: r.Clone(),
	}
	if r.Error != "" {
		n.Message = r.Error
	}
	// 没有可推送内容（成功但无输出，或超时/取消被清空）时保持静默。
	hasContent := strings.TrimSpace(n.Message) != ""
	for _, id := range t.NotifierIDs {
		for _, ch := range channels {
			if ch.ID != id {
				continue
			}
			d := Delivery{ChannelID: id, ChannelName: ch.Name}
			if !ch.Enabled || !hasContent {
				d.Status = "skipped"
			} else {
				d.Status = "sent"
				if err := e.sendNotification(ctx, ch, n); err != nil {
					d.Status, d.Error = "failed", truncate(err.Error(), 8192)
				}
			}
			results = append(results, d)
		}
	}
	return results
}

// Notification concurrency is independent of task workers. The channel's
// timeout covers waiting for a slot and sending, and honors run cancellation.
func (e *Engine) sendNotification(parent context.Context, ch Channel,
	n Notification) error {
	ctx, cancel := context.WithTimeout(parent, e.options.NotificationTimeout)
	defer cancel()
	select {
	case e.notifySlots <- struct{}{}:
		defer func() { <-e.notifySlots }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	notifier, err := e.registry.Notifiers.Create(ch.Type)
	if err != nil {
		return err
	}
	return safeSend(notifier, ctx, ch.Config, n)
}

func safeRun(h TaskHandler, ctx context.Context,
	config json.RawMessage) (output string, err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("任务插件 panic: %v", v)
		}
	}()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return h.Run(ctx, config)
}

func safeSend(n Notifier, ctx context.Context, config json.RawMessage,
	message Notification) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("通知插件 panic: %v", v)
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	config = append(json.RawMessage(nil), config...)
	message.Run = message.Run.Clone()
	if err := n.Send(ctx, config, message); err != nil {
		return err
	}
	return ctx.Err()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n…输出已截断"
}
