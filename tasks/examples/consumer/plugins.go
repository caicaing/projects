package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/caicaing/projects/tasks/framework"
)

type sumConfig struct {
	A int `json:"a"`
	B int `json:"b"`
}

type auditConfig struct {
	Prefix string `json:"prefix"`
}

// These plugins belong to this consumer, not to the framework repository.
func registerPlugins(r *taskflow.Registry) error {
	if err := r.Tasks.Register(func() taskflow.TaskHandler {
		return taskflow.TaskFunc{
			Definition: taskflow.Spec{
				Type: "sum", Name: "业务求和任务",
				Description: "独立项目定义的任务，共用框架调度与通知。",
				Example:     json.RawMessage(`{"a":1,"b":2}`),
			},
			ValidateFn: func(raw json.RawMessage) error {
				return taskflow.DecodeConfig(raw, &sumConfig{})
			},
			RunFn: runSum,
		}
	}); err != nil {
		return err
	}
	return r.Notifiers.Register(func() taskflow.Notifier {
		return taskflow.NotifierFunc{
			Definition: taskflow.Spec{
				Type: "audit", Name: "业务审计通知",
				Description: "独立项目定义的通知，可替换为业务 SDK 调用。",
				Example:     json.RawMessage(`{"prefix":"业务通知"}`),
			},
			ValidateFn: func(raw json.RawMessage) error {
				return taskflow.DecodeConfig(raw, &auditConfig{})
			},
			SendFn: sendAudit,
		}
	})
}

func runSum(ctx context.Context, raw json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var c sumConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d + %d = %d", c.A, c.B, c.A+c.B), nil
}

func sendAudit(ctx context.Context, raw json.RawMessage,
	n taskflow.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var c auditConfig
	if err := taskflow.DecodeConfig(raw, &c); err != nil {
		return err
	}
	slog.Info(c.Prefix, "title", n.Title, "message", n.Message)
	return nil
}
