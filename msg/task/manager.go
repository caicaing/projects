package task

import (
	"errors"
	"fmt"
)

// TaskManager 任务管理器接口
type TaskManager interface {
	// RegisterTask 注册任务
	RegisterTask(task Task) error

	// ExecuteAllTasks 执行所有任务
	ExecuteAllTasks() error
}

// DefaultTaskManager 默认任务管理器实现
type DefaultTaskManager struct {
	tasks map[string]Task
}

// NewDefaultTaskManager 创建默认任务管理器
func NewDefaultTaskManager() *DefaultTaskManager {
	return &DefaultTaskManager{
		tasks: make(map[string]Task),
	}
}

// RegisterTask 注册任务
func (m *DefaultTaskManager) RegisterTask(task Task) error {
	if task == nil {
		return errors.New("task cannot be nil")
	}

	taskID := task.GetID()
	if _, exists := m.tasks[taskID]; exists {
		return fmt.Errorf("task with ID %s already exists", taskID)
	}

	m.tasks[taskID] = task
	return nil
}

// ExecuteAllTasks 执行所有任务
func (m *DefaultTaskManager) ExecuteAllTasks() error {
	tasks := m.tasks
	// 在新的goroutine中并行执行所有任务
	for _, task := range tasks {
		go task.Execute()
	}
	return nil
}
