# Taskflow 框架

独立 Go 模块 `github.com/caicaing/projects/tasks/framework`，包名 `taskflow`。
只依赖标准库，提供任务/通知注册、时间计划、调度、执行、重试、取消和结果。
不包含管理 API、登录、配置文件解析、数据库驱动或具体业务插件。

## 使用

```go
import (
    "context"

    "github.com/caicaing/projects/tasks/framework"
    "github.com/caicaing/projects/tasks/framework/storage/memory"
)

engine, err := taskflow.New(ctx, taskflow.Options{
    Store: memory.New(),
}, registerPlugins)
if err != nil {
    return err
}
if err := engine.Start(ctx); err != nil {
    _ = engine.Close(context.Background())
    return err
}
// 注册任务计划：engine.SaveTask(ctx, "", task)
// 注册渠道配置：engine.SaveChannel(ctx, "", channel)
// 手动执行：engine.Trigger(ctx, taskID)
// 关闭并等待执行退出：engine.Close(shutdownCtx)
```

完整独立项目见 [consumer](../examples/consumer/README.md)。
配置由使用方管理，可以从自己的 conf.yaml 读取参数后传入 Options。
框架不关心配置来源；通常只需 Store，其他参数使用默认值。

## 插件注册

任务实现 `TaskHandler`：Spec、Validate、Run；通知实现 `Notifier`：
Spec、Validate、Send。它们接收 JSON 业务参数和 Context。
通知策略和渠道关联保存在 Task.NotifyOn / Task.NotifierIDs。
框架根据关联调用通知，不需要任务依赖具体通知实现。

```go
func registerPlugins(r *taskflow.Registry) error {
    if err := r.Tasks.Register(func() taskflow.TaskHandler {
        return &MyTask{}
    }); err != nil {
        return err
    }
    return r.Notifiers.Register(func() taskflow.Notifier {
        return &MyNotifier{}
    })
}
```

构造器返回独立实例，可在闭包中注入业务依赖；框架不自动注册插件。
注册在 New 时固定，每次执行或通知创建实例，同一次任务重试复用实例。
轻量业务也可用 TaskFunc / NotifierFunc，参数解码可用 DecodeConfig。
Spec.Example 必须是有效配置，Validate 应快速且没有外部副作用。

## 状态与生命周期

默认内存存储由 `storage/memory.New()` 提供。需要持久化时自行实现 Store：
Load 返回独立的 State，Save 原子保存完整 State，失败不留下部分更新。
空存储的 State.Version 为 1，可用 State.Clone() 复制。
本仓库后端的 SQLite 是应用适配器，不是框架依赖。

New 只初始化，Start 开始执行；Start 的 Context 取消后停止调度和运行。
Close 等待活动任务、渠道测试和最终记录保存，支持重复等待。
关闭超时后清理继续；执行记录保存失败时保留内存结果，Close 会再次尝试保存。
如果存储仍不可用，Close 返回 ErrStorage；恢复后可再次调用 Close 重试。
调用方在 Close 返回 nil 后关闭存储及业务资源，关闭后不支持重启该 Engine。
插件和存储必须响应 Context，否则框架无法强制终止它们。

Options 零值使用默认参数：8 个 worker、500 条历史、1 秒轮询、
30 秒任务超时、2 秒重试间隔、10 秒通知超时、5 秒存储超时。
手动触发的请求 Context 仅控制记录提交，执行继承 Start 的 Context。
业务执行结束后释放任务 worker；通知并发独立限制为 Workers，
渠道超时包含等待投递名额的时间。CancelRun 同时取消业务和通知阶段，
任务自身超时不阻止失败通知；取消通知不会改变已经完成的业务结果。
通知失败不改变任务结果，不自动重试，关闭仍会等待通知退出及结果保存。
构造、执行、投递及参数校验的 panic 会转换为错误。

只支持单实例调度；同一 Store 只能给一个 Engine 使用。
插件源码在使用方仓库维护，发布框架不需要带上管理台和业务插件。
框架模块发布使用 `framework/vX.Y.Z` tag；私有仓库下载需 Git 读取权限，
并配置 GOPRIVATE=github.com/caicaing/projects。当前示例通过 replace 使用本地模块。
