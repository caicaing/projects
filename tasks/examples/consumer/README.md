# 独立项目接入示例

这个独立 Go 模块只依赖 framework，不依赖 backend、SQLite、管理 API
或任何框架仓库里的业务插件。

```sh
cd examples/consumer
go run .
```

示例自行注册 sum 任务和 audit 通知，以内存存储创建计划及渠道关联。
启动时手动运行一次求和，此后每小时执行；Ctrl+C 取消并等待引擎退出。
关闭进程后内存任务和记录不保留，不发送真实外部通知。

复制到其他仓库时调整 go.mod 中 replace 为 framework 实际目录。
正式发布后可以移除 replace，通过仓库访问凭据下载：

```sh
GOPRIVATE=git.woa.com go get git.woa.com/mooyliu/mooy_test/framework@latest
```

业务插件放在本项目 plugins.go，配置和数据库也由使用方自行管理。
框架能力和协议见 [公共包文档](../../framework/README.md)。
