package taskflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/caicaing/projects/tasks/framework/internal/schedule"
)

func (e *Engine) SaveTask(ctx context.Context, id string,
	task Task) (Task, error) {
	if err := e.lockWrite(ctx); err != nil {
		return Task{}, err
	}
	defer e.unlockWrite()
	if err := e.availabilityLocked(); err != nil {
		return Task{}, err
	}
	task = task.Clone()
	e.normalizeTask(&task)
	schedule, err := NormalizeSchedule(task.Schedule)
	if err != nil {
		return Task{}, err
	}
	task.Schedule = schedule
	p, err := e.validateTask(task)
	if err != nil {
		return Task{}, err
	}
	s := e.state.Clone()
	index := -1
	if id != "" {
		for i, t := range s.Tasks {
			if t.ID == id {
				index = i
				task.CreatedAt = t.CreatedAt
			}
		}
		if index == -1 {
			return Task{}, ErrNotFound
		}
	} else {
		id = newID()
		task.CreatedAt = time.Now()
	}
	task.ID = id
	task.UpdatedAt = time.Now()
	task.NextRun = nil
	if task.Enabled && e.started {
		next, err := p.Next(task.UpdatedAt)
		if err != nil {
			return Task{}, err
		}
		task.NextRun = &next
	}
	if index >= 0 {
		s.Tasks[index] = task
	} else {
		s.Tasks = append(s.Tasks, task)
	}
	if err := e.commitLocked(ctx, s); err != nil {
		return Task{}, err
	}
	e.plans[id] = p
	return task.Clone(), nil
}

func (e *Engine) DeleteTask(ctx context.Context, id string) error {
	if err := e.lockWrite(ctx); err != nil {
		return err
	}
	defer e.unlockWrite()
	if err := e.availabilityLocked(); err != nil {
		return err
	}
	if e.runningTasks[id] {
		return ErrBusy
	}
	s := e.state.Clone()
	for i, t := range s.Tasks {
		if t.ID != id {
			continue
		}
		s.Tasks = append(s.Tasks[:i], s.Tasks[i+1:]...)
		if err := e.commitLocked(ctx, s); err != nil {
			return err
		}
		delete(e.plans, id)
		return nil
	}
	return ErrNotFound
}

func (e *Engine) SaveChannel(ctx context.Context, id string,
	ch Channel) (Channel, error) {
	if err := e.lockWrite(ctx); err != nil {
		return Channel{}, err
	}
	defer e.unlockWrite()
	if err := e.availabilityLocked(); err != nil {
		return Channel{}, err
	}
	ch = ch.Clone()
	ch.Name = strings.TrimSpace(ch.Name)
	if err := e.validateChannel(ch); err != nil {
		return Channel{}, err
	}
	s := e.state.Clone()
	index := -1
	if id != "" {
		for i, c := range s.Channels {
			if c.ID == id {
				index = i
				ch.CreatedAt = c.CreatedAt
			}
		}
		if index == -1 {
			return Channel{}, ErrNotFound
		}
	} else {
		id = newID()
		ch.CreatedAt = time.Now()
	}
	ch.ID, ch.UpdatedAt = id, time.Now()
	if index >= 0 {
		s.Channels[index] = ch
	} else {
		s.Channels = append(s.Channels, ch)
	}
	if err := e.commitLocked(ctx, s); err != nil {
		return Channel{}, err
	}
	return ch.Clone(), nil
}

func (e *Engine) DeleteChannel(ctx context.Context, id string) error {
	if err := e.lockWrite(ctx); err != nil {
		return err
	}
	defer e.unlockWrite()
	if err := e.availabilityLocked(); err != nil {
		return err
	}
	for _, t := range e.state.Tasks {
		for _, channelID := range t.NotifierIDs {
			if channelID == id {
				return fmt.Errorf("任务 %s 仍在使用该通知渠道", t.Name)
			}
		}
	}
	s := e.state.Clone()
	for i, c := range s.Channels {
		if c.ID == id {
			s.Channels = append(s.Channels[:i], s.Channels[i+1:]...)
			return e.commitLocked(ctx, s)
		}
	}
	return ErrNotFound
}

func (e *Engine) TestChannel(ctx context.Context, id string) error {
	e.mu.Lock()
	if err := e.executionReadyLocked(); err != nil {
		e.mu.Unlock()
		return err
	}
	engineCtx := e.ctx
	var channel *Channel
	for _, ch := range e.state.Channels {
		if ch.ID == id {
			copy := ch
			copy.Config = append(json.RawMessage(nil), ch.Config...)
			channel = &copy
		}
	}
	if channel != nil {
		e.wg.Add(1)
	}
	e.mu.Unlock()
	if channel == nil {
		return ErrNotFound
	}
	defer e.wg.Done()
	n := Notification{Title: "Taskflow 测试通知", Message: "通知渠道连接正常"}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(engineCtx, cancel)
	defer stop()
	if err := engineCtx.Err(); err != nil {
		return err
	}
	return e.sendNotification(ctx, *channel, n)
}

func (e *Engine) normalizeTask(t *Task) {
	t.Name = strings.TrimSpace(t.Name)
	t.Schedule.Expression = strings.TrimSpace(t.Schedule.Expression)
	if t.Schedule.Timezone == "" {
		t.Schedule.Timezone = "Asia/Shanghai"
	}
	if t.Timeout == "" {
		t.Timeout = e.options.DefaultTaskTimeout.String()
	}
	if t.RetryDelay == "" {
		t.RetryDelay = e.options.DefaultRetryDelay.String()
	}
	if t.NotifyOn == "" {
		t.NotifyOn = "failure"
	}
	if t.NotifierIDs == nil {
		t.NotifierIDs = []string{}
	}
}

func (e *Engine) validateTask(t Task) (schedule.Plan, error) {
	p, err := schedule.Parse(t.Schedule)
	if err != nil {
		return p, err
	}
	if len(t.Name) == 0 || len(t.Name) > 200 {
		return p, fmt.Errorf("任务名称必填且最多 200 字节")
	}
	if err := e.validateTaskPluginLocked(t); err != nil {
		return p, fmt.Errorf("任务参数: %w", err)
	}
	_, err = validDuration(t.Timeout, time.Second, 24*time.Hour)
	if err != nil {
		return p, fmt.Errorf("超时: %w", err)
	}
	_, err = validDuration(t.RetryDelay, time.Second, time.Hour)
	if err != nil {
		return p, fmt.Errorf("重试间隔: %w", err)
	}
	if t.Retries < 0 || t.Retries > 10 {
		return p, fmt.Errorf("重试次数须为 0 到 10")
	}
	if t.NotifyOn != "failure" && t.NotifyOn != "always" &&
		t.NotifyOn != "never" {
		return p, fmt.Errorf("通知策略须为 failure、always 或 never")
	}
	seen := make(map[string]bool)
	for _, id := range t.NotifierIDs {
		found := false
		for _, ch := range e.state.Channels {
			if ch.ID == id {
				found = true
			}
		}
		if !found || seen[id] {
			return p, fmt.Errorf("通知渠道不存在或重复: %s", id)
		}
		seen[id] = true
	}
	_, err = p.Next(time.Now())
	return p, err
}

func (e *Engine) validateChannel(c Channel) error {
	if len(strings.TrimSpace(c.Name)) == 0 || len(c.Name) > 200 {
		return fmt.Errorf("通知名称必填且最多 200 字节")
	}
	e.mu.Unlock()
	defer e.mu.Lock()
	n, err := e.registry.Notifiers.Create(c.Type)
	if err != nil {
		return err
	}
	return safeValidate(n, c.Config)
}

func (e *Engine) validateTaskPluginLocked(t Task) error {
	e.mu.Unlock()
	defer e.mu.Lock()
	h, err := e.registry.Tasks.Create(t.Type)
	if err != nil {
		return err
	}
	return safeValidate(h, t.Config)
}

func validDuration(raw string, min, max time.Duration) (time.Duration, error) {
	d, err := time.ParseDuration(raw)
	if err != nil || d < min || d > max {
		return 0, fmt.Errorf("时长须介于 %s 和 %s", min, max)
	}
	return d, nil
}
