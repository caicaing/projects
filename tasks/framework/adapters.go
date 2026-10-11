package taskflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// TaskFunc adapts functions to the task plugin interface.
type TaskFunc struct {
	Definition Spec
	ValidateFn func(json.RawMessage) error
	RunFn      func(context.Context, json.RawMessage) (string, error)
}

func (p TaskFunc) Spec() Spec { return p.Definition }

func (p TaskFunc) Validate(raw json.RawMessage) error {
	if p.RunFn == nil {
		return fmt.Errorf("任务插件缺少 RunFn")
	}
	if p.ValidateFn != nil {
		return p.ValidateFn(raw)
	}
	var object map[string]any
	return DecodeConfig(raw, &object)
}

func (p TaskFunc) Run(ctx context.Context,
	raw json.RawMessage) (string, error) {
	return p.RunFn(ctx, raw)
}

// NotifierFunc adapts functions to the notification plugin interface.
type NotifierFunc struct {
	Definition Spec
	ValidateFn func(json.RawMessage) error
	SendFn     func(context.Context, json.RawMessage, Notification) error
}

func (p NotifierFunc) Spec() Spec { return p.Definition }

func (p NotifierFunc) Validate(raw json.RawMessage) error {
	if p.SendFn == nil {
		return fmt.Errorf("通知插件缺少 SendFn")
	}
	if p.ValidateFn != nil {
		return p.ValidateFn(raw)
	}
	var object map[string]any
	return DecodeConfig(raw, &object)
}

func (p NotifierFunc) Send(ctx context.Context, raw json.RawMessage,
	n Notification) error {
	return p.SendFn(ctx, raw, n)
}

func DecodeConfig(raw json.RawMessage, value any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return fmt.Errorf("配置必须是 JSON 对象")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("配置只能包含一个 JSON 对象")
	}
	return nil
}
