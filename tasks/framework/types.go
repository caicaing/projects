package taskflow

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/caicaing/projects/tasks/framework/internal/model"
)

var (
	ErrNotFound   = errors.New("记录不存在")
	ErrBusy       = errors.New("任务正在执行或工作池已满")
	ErrClosed     = errors.New("调度器已关闭")
	ErrNotStarted = errors.New("调度器尚未启动")
	ErrStarted    = errors.New("调度器已经启动")
	ErrStorage    = errors.New("数据存储失败")
)

// Public aliases share model definitions with the scheduling package.
type Spec = model.Spec
type Schedule = model.Schedule
type Task = model.Task
type Channel = model.Channel
type Delivery = model.Delivery
type Run = model.Run
type Notification = model.Notification
type State = model.State
type Catalog = model.Catalog

// Plugin defines metadata and validation shared by all extensions.
type Plugin interface {
	Spec() Spec
	Validate(json.RawMessage) error
}

// TaskHandler implementations must honor context cancellation.
type TaskHandler interface {
	Plugin
	Run(context.Context, json.RawMessage) (string, error)
}

type Notifier interface {
	Plugin
	Send(context.Context, json.RawMessage, Notification) error
}

// Store persists one engine's complete state atomically. Load must return an
// owned snapshot (Version 1 for a new store). Save must not retain caller-owned
// slices or mutate committed data on failure. The host owns store cleanup.
// One store belongs to one engine; this contract does not provide leader
// election or concurrent multi-engine writes.
type Store interface {
	Load(context.Context) (State, error)
	Save(context.Context, State) error
}

// Logger allows a host to adapt its logging system without a global logger.
type Logger interface {
	ErrorContext(context.Context, string, ...any)
}

// Options configures the core; storage is required, other zero values use
// defaults. New initializes but does not start execution.
type Options struct {
	Store               Store
	Logger              Logger
	Workers             int
	HistoryLimit        int
	PollInterval        time.Duration
	StorageTimeout      time.Duration
	NotificationTimeout time.Duration
	DefaultTaskTimeout  time.Duration
	DefaultRetryDelay   time.Duration
}
