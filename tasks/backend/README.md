# Taskflow 管理后端

本目录是独立的 Go 应用模块，只负责配置读取、插件装配、SQLite 持久化
和 HTTP 管理 API。调度能力来自 [framework](../framework/README.md)。

```text
cmd/taskflow/     应用入口及插件注册
config/           conf.yaml 读取和校验
api/              管理 API 与后端 Token 校验
storage/sqlite/   本应用的 SQLite 适配器
plugins/builtin/  HTTP/日志任务、Webhook/日志通知
plugins/bark/     Bark 单设备/批量通知
plugins/reminder/ 喝水提醒任务
plugins/demo/     函数插件示例
```

## 启动和配置

将本目录 conf.example.yaml 复制为 conf.yaml，填写 backend.token 后，
从项目根目录执行：

```sh
make backend
# 或先 make build，再从项目根目录执行：
./backend/bin/taskflow -config backend/conf.yaml
```

在 backend 目录直接启动时，默认读取当前目录的 conf.yaml。
通过 -config 指定其他配置文件；数据库相对路径以配置文件目录为基准。
本配置只包含 backend 和 scheduler，不读取前端配置或环境变量，
所有 YAML 字符串均按字面值使用。
conf.yaml 已忽略，不应提交其中的真实密钥。
未知字段、重复字段、多文档 YAML 和锚点/别名会拒绝加载。

| 配置段 | 职责 |
| --- | --- |
| backend | API 监听地址、数据库路径、后端 token |
| scheduler | 并发数、历史条数、任务和通知超时 |

backend.token 至少 16 字符，必须与 frontend/conf.yaml 的 backend.token 相同。
任务和通知的业务参数由管理台保存在数据库，不写入启动配置。
SQLite 表结构不变，已有数据仍可读取；数据库文件只能由一个进程打开。

## 添加插件

实现框架的 TaskHandler / Notifier，在 cmd/taskflow/plugins.go 显式注册。
本应用默认注册 builtin、Bark 和喝水提醒；可以任意删减或替换注册列表。
框架不会自动注册这些类型，也不导入本目录任何包。

插件依赖数据库、SDK 等资源时由应用初始化，并在构造器闭包中传入。
退出时先等待 Engine.Close，再关闭 SQLite 及业务资源。

## 管理 API

路由与 JSON 协议保持原样，完整接口见 [根目录说明](../README.md#api)。
api.New(engine, token) 返回 http.Handler，可挂到业务服务。
生产管理入口通过 Caddy 和 Web 登录网关，后端只在本机监听。

使用方只需要框架时无需引入这个模块；需要此套管理台时才引用 backend/api。
宿主机部署由 systemd 管理进程，Caddy 自动签发和续期 HTTPS 证书，
配置及操作见 [根目录说明](../README.md#一键公网部署)。
