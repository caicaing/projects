package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caicaing/projects/tasks/framework"
	"github.com/caicaing/projects/tasks/framework/storage/memory"
)

func main() {
	if err := run(); err != nil {
		slog.Error("示例退出", "error", err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()
	engine, err := taskflow.New(ctx, taskflow.Options{Store: memory.New()},
		registerPlugins)
	if err != nil {
		return err
	}
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(),
			5*time.Second)
		defer cancel()
		if err := engine.Close(shutdown); err != nil {
			runErr = err
		}
	}()
	channel, err := engine.SaveChannel(ctx, "", taskflow.Channel{
		Name: "业务审计", Type: "audit", Enabled: true,
		Config: json.RawMessage(`{"prefix":"计算结果"}`),
	})
	if err != nil {
		return err
	}
	task, err := engine.SaveTask(ctx, "", taskflow.Task{
		Name: "业务求和", Type: "sum", Enabled: true,
		Config:   json.RawMessage(`{"a":1,"b":2}`),
		Schedule: taskflow.Schedule{Kind: "interval", Expression: "1h"},
		NotifyOn: "always", NotifierIDs: []string{channel.ID},
	})
	if err != nil {
		return err
	}
	if err := engine.Start(ctx); err != nil {
		return err
	}
	if _, err := engine.Trigger(ctx, task.ID); err != nil {
		return err
	}
	fmt.Println("已执行求和任务，每小时再执行；Ctrl+C 退出")
	<-ctx.Done()
	return nil
}
