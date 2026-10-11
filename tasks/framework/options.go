package taskflow

import (
	"fmt"
	"log/slog"
	"reflect"
	"time"
)

func normalizeOptions(o Options) (Options, error) {
	if nilDependency(o.Store) {
		return o, fmt.Errorf("Store 不能为空")
	}
	if o.Workers < 0 || o.HistoryLimit < 0 {
		return o, fmt.Errorf("Workers 和 HistoryLimit 不能为负数")
	}
	if o.Workers == 0 {
		o.Workers = 8
	}
	if o.HistoryLimit == 0 {
		o.HistoryLimit = 500
	}
	defaults := []struct {
		name  string
		value *time.Duration
		base  time.Duration
	}{
		{"PollInterval", &o.PollInterval, time.Second},
		{"StorageTimeout", &o.StorageTimeout, 5 * time.Second},
		{"NotificationTimeout", &o.NotificationTimeout, 10 * time.Second},
		{"DefaultTaskTimeout", &o.DefaultTaskTimeout, 30 * time.Second},
		{"DefaultRetryDelay", &o.DefaultRetryDelay, 2 * time.Second},
	}
	for _, item := range defaults {
		if *item.value < 0 {
			return o, fmt.Errorf("%s 不能为负数", item.name)
		}
		if *item.value == 0 {
			*item.value = item.base
		}
	}
	if o.DefaultTaskTimeout < time.Second ||
		o.DefaultTaskTimeout > 24*time.Hour {
		return o, fmt.Errorf("DefaultTaskTimeout 须介于 1s 和 24h")
	}
	if o.DefaultRetryDelay < time.Second || o.DefaultRetryDelay > time.Hour {
		return o, fmt.Errorf("DefaultRetryDelay 须介于 1s 和 1h")
	}
	if nilDependency(o.Logger) {
		o.Logger = slog.Default()
	}
	return o, nil
}

func nilDependency(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
