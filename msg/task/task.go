package task

// TaskStatus 任务状态枚举
type TaskStatus int

// Task 定义任务的基本接口
type Task interface {
	// Execute 执行任务
	Execute() error

	// GetID 获取任务ID
	GetID() string

	// GetName 获取任务名称
	GetName() string
}
