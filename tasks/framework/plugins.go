package taskflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"sync"
)

// PluginFactory is the common registration and creation contract.
type PluginFactory[T Plugin] interface {
	Register(func() T) error
	Create(string) (T, error)
	Specs() []Spec
}

type TaskFactory = Factory[TaskHandler]
type NotifierFactory = Factory[Notifier]

func NewTaskFactory() *TaskFactory { return NewFactory[TaskHandler]() }

func NewNotifierFactory() *NotifierFactory { return NewFactory[Notifier]() }

type factoryEntry[T Plugin] struct {
	spec        Spec
	constructor func() T
}

// Factory creates a fresh plugin instance for every Create call.
type Factory[T Plugin] struct {
	mu      sync.RWMutex
	entries map[string]factoryEntry[T]
}

func NewFactory[T Plugin]() *Factory[T] {
	return &Factory[T]{entries: make(map[string]factoryEntry[T])}
}

func (f *Factory[T]) Register(constructor func() T) error {
	instance, spec, err := construct(constructor)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}
	if err := safeValidate(instance, spec.Example); err != nil {
		return fmt.Errorf("插件 %s 的示例配置无效: %w", spec.Type, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.entries[spec.Type]; exists {
		return fmt.Errorf("插件类型重复: %s", spec.Type)
	}
	f.entries[spec.Type] = factoryEntry[T]{
		spec: cloneSpec(spec), constructor: constructor,
	}
	return nil
}

func (f *Factory[T]) Create(kind string) (T, error) {
	f.mu.RLock()
	entry, ok := f.entries[kind]
	f.mu.RUnlock()
	if !ok {
		var zero T
		return zero, fmt.Errorf("未注册的插件类型: %s", kind)
	}
	instance, spec, err := construct(entry.constructor)
	if err == nil && spec.Type != kind {
		err = fmt.Errorf("插件构造器返回了不一致的类型: %s", kind)
	}
	return instance, err
}

func (f *Factory[T]) Specs() []Spec {
	f.mu.RLock()
	defer f.mu.RUnlock()
	list := make([]Spec, 0, len(f.entries))
	for _, entry := range f.entries {
		list = append(list, cloneSpec(entry.spec))
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Type < list[j].Type
	})
	return list
}

func (f *Factory[T]) snapshot() *Factory[T] {
	f.mu.RLock()
	defer f.mu.RUnlock()
	copy := NewFactory[T]()
	for key, entry := range f.entries {
		entry.spec = cloneSpec(entry.spec)
		copy.entries[key] = entry
	}
	return copy
}

func construct[T Plugin](constructor func() T) (instance T,
	spec Spec, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("插件构造器 panic: %v", value)
		}
	}()
	if constructor == nil {
		return instance, spec, fmt.Errorf("插件构造器不能为空")
	}
	instance = constructor()
	if nilDependency(instance) {
		return instance, spec, fmt.Errorf("插件构造器返回了 nil")
	}
	return instance, instance.Spec(), nil
}

func safeValidate(plugin Plugin, raw json.RawMessage) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("插件参数校验 panic: %v", value)
		}
	}()
	return plugin.Validate(append(json.RawMessage(nil), raw...))
}

var typePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

func validateSpec(s Spec) error {
	if !typePattern.MatchString(s.Type) || s.Name == "" {
		return fmt.Errorf("插件需要名称及小写字母开头的 type，最多 64 字符")
	}
	var example map[string]json.RawMessage
	if err := DecodeConfig(s.Example, &example); err != nil {
		return fmt.Errorf("插件 %s 的 example 须为 JSON 对象: %w", s.Type, err)
	}
	return nil
}

func cloneSpec(s Spec) Spec {
	s.Example = append(json.RawMessage(nil), s.Example...)
	return s
}

// Registry composes the two factories used by the engine.
type Registry struct {
	Tasks     *TaskFactory
	Notifiers *NotifierFactory
}

func NewRegistry() *Registry {
	return &Registry{
		Tasks:     NewTaskFactory(),
		Notifiers: NewNotifierFactory(),
	}
}

func (r *Registry) Catalog() Catalog {
	return Catalog{Tasks: r.Tasks.Specs(), Notifiers: r.Notifiers.Specs()}
}

func (r *Registry) snapshot() *Registry {
	return &Registry{
		Tasks: r.Tasks.snapshot(), Notifiers: r.Notifiers.snapshot(),
	}
}
