package main

import (
	"github.com/caicaing/send_msg/task"
	"github.com/caicaing/send_msg/task/imp"
)

func main() {
	// 创建任务管理器
	manager := task.NewDefaultTaskManager()
	//manager.RegisterTask(imp.NewSendMessageTask())
	manager.RegisterTask(imp.NewWaterReminderTask())
	// 执行普通消息任务
	manager.ExecuteAllTasks()
	select {}
}
