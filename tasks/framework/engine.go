package taskflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"git.woa.com/mooyliu/mooy_test/framework/internal/schedule"
)

type Engine struct {
	mu           sync.Mutex
	writes       chan struct{}
	state        State
	registry     *Registry
	plans        map[string]schedule.Plan
	active       map[string]context.CancelFunc
	runningTasks map[string]bool
	options      Options
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	done         chan struct{}
	stopping     chan struct{}
	stopOnce     sync.Once
	notifySlots  chan struct{}
	dirty        bool
	started      bool
}

// New registers plugins, loads storage and restores state without starting
// goroutines or executing tasks. The host owns the supplied store even if
// initialization fails. Call Start after dependencies are ready.
func New(ctx context.Context, options Options,
	registrations ...func(*Registry) error) (*Engine, error) {
	if ctx == nil {
		return nil, fmt.Errorf("Context 不能为空")
	}
	options, err := normalizeOptions(options)
	if err != nil {
		return nil, err
	}
	r := NewRegistry()
	for _, register := range registrations {
		if register == nil {
			return nil, fmt.Errorf("插件注册函数不能为空")
		}
		if err := register(r); err != nil {
			return nil, fmt.Errorf("注册插件失败: %w", err)
		}
	}
	loadCtx, cancel := context.WithTimeout(ctx, options.StorageTimeout)
	defer cancel()
	s, err := options.Store.Load(loadCtx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStorage, err)
	}
	if s.Version != 1 {
		return nil, fmt.Errorf("不支持的存储版本: %d", s.Version)
	}
	e := &Engine{
		state: s.Clone(), registry: r.snapshot(), options: options,
		plans:        make(map[string]schedule.Plan),
		active:       make(map[string]context.CancelFunc),
		runningTasks: make(map[string]bool), done: make(chan struct{}),
		writes: make(chan struct{}, 1), stopping: make(chan struct{}),
		notifySlots: make(chan struct{}, options.Workers),
	}
	e.mu.Lock()
	err = e.restore(ctx)
	e.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) restore(ctx context.Context) error {
	now := time.Now()
	for i := range e.state.Channels {
		if err := e.validateChannel(e.state.Channels[i]); err != nil {
			return fmt.Errorf("通知渠道恢复失败: %w", err)
		}
	}
	for i := range e.state.Tasks {
		t := &e.state.Tasks[i]
		e.normalizeTask(t)
		normalized, err := NormalizeSchedule(t.Schedule)
		if err != nil {
			return fmt.Errorf("任务 %s 的计划恢复失败: %w", t.Name, err)
		}
		t.Schedule = normalized
		p, err := e.validateTask(*t)
		if err != nil {
			return fmt.Errorf("任务 %s 恢复失败: %w", t.Name, err)
		}
		e.plans[t.ID] = p
		// NextRun is calculated only when the host starts scheduling.
		t.NextRun = nil
	}
	for i := range e.state.Runs {
		r := &e.state.Runs[i]
		if r.Status == "running" {
			r.Status = "interrupted"
			r.Error = "进程退出中断了本次执行；未自动重放"
			r.FinishedAt = &now
			r.DurationMS = now.Sub(r.StartedAt).Milliseconds()
		}
	}
	return e.commitLocked(ctx, e.state)
}

func (e *Engine) Snapshot() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state.Clone()
}

func (e *Engine) Catalog() Catalog { return e.registry.Catalog() }

func (e *Engine) tick(now time.Time) {
	if err := e.lockWrite(e.ctx); err != nil {
		return
	}
	defer e.unlockWrite()
	if e.availabilityLocked() != nil {
		return
	}
	for _, task := range e.state.Tasks {
		if !task.Enabled || task.NextRun == nil || now.Before(*task.NextRun) {
			continue
		}
		if _, err := e.startLocked(e.ctx, task, "schedule", now); err != nil {
			e.options.Logger.ErrorContext(e.ctx, "调度失败",
				"task", task.ID, "error", err)
		}
	}
}

func (e *Engine) commitLocked(ctx context.Context, s State) error {
	e.trimHistory(&s)
	if err := e.persistLocked(ctx, s); err != nil {
		return err
	}
	e.state = s
	e.dirty = false
	return nil
}

func (e *Engine) trimHistory(s *State) {
	if len(s.Runs) <= e.options.HistoryLimit {
		return
	}
	keep := make([]Run, 0, e.options.HistoryLimit)
	for _, r := range s.Runs {
		if len(keep) < e.options.HistoryLimit || r.Status == "running" ||
			e.active[r.ID] != nil {
			keep = append(keep, r)
		}
	}
	s.Runs = keep
}

func newID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
