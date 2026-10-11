// Package plugins demonstrates optional task and notification extensions.
package plugins

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

func Register(r *taskflow.Registry) error {
	if err := r.Tasks.Register(func() taskflow.TaskHandler {
		return taskflow.TaskFunc{
			Definition: taskflow.Spec{
				Type: "sum", Name: "求和示例",
				Description: "演示通过函数适配器添加业务任务。",
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
				Type: "audit", Name: "审计日志示例",
				Description: "演示自定义通知投递，替换 SendFn 即可对接 SDK。",
				Example:     json.RawMessage(`{}`),
			},
			ValidateFn: func(raw json.RawMessage) error {
				return taskflow.DecodeConfig(raw, &struct{}{})
			},
			SendFn: func(ctx context.Context, _ json.RawMessage,
				n taskflow.Notification) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				slog.Info("自定义通知", "title", n.Title, "run_id", n.Run.ID)
				return nil
			},
		}
	})
}

func runSum(ctx context.Context, raw json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var config sumConfig
	if err := taskflow.DecodeConfig(raw, &config); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d + %d = %d", config.A, config.B,
		config.A+config.B), nil
}
