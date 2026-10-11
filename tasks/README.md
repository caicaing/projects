# Taskflow

一个独立的 Go 定时任务框架，加上可选的管理后端和中文管理台。
框架只提供注册、调度、执行、重试、取消与通知路由；任务和通知由使用方实现。

## 目录

```text
framework/          独立公共 Go 模块，无第三方依赖
backend/            管理服务、SQLite 适配器、应用插件
frontend/           管理页面、登录和 API 代理
examples/consumer/  只引入框架并自行注册插件的独立项目
backend/conf.yaml   后端实际配置，模板在同目录，已加入 Git 忽略
frontend/conf.yaml  Web 网关实际配置，模板在同目录，已加入 Git 忽略
scripts/            本地启动与公网部署脚本
```

框架不包含 HTTP 管理、登录、数据库驱动、Bark 或喝水提醒。
这些是应用代码，位于 backend；使用方也可以完全不用本仓库管理台。
接入说明见 [框架文档](framework/README.md)，可运行示例见
[consumer](examples/consumer/README.md)。

## 启动与配置

要求 Go 1.23+、Node.js 18+，使用 SQLite，无需独立数据库服务。
在根目录执行：

```sh
cp backend/conf.example.yaml backend/conf.yaml
cp frontend/conf.example.yaml frontend/conf.yaml
# 填写两份配置的 backend.token（相同值）以及前端 web.password。
make run
```

已有配置时不要重新复制模板。make run 会在缺少配置时复制对应模板，
但不会生成默认密码或 token；请先填写它们。
默认打开 <http://127.0.0.1:8080>，账号为 admin，密码使用前端配置中的值。
make run 会编译后端并安装前端 YAML 解析依赖，也可分别执行
make backend 和 make frontend。数据库默认是 backend/data/taskflow.db。
已有 SQLite 表结构及数据保持可用，旧 JSON 不自动导入。

前后端各自读取独立配置，所有值直接使用，不读取或替换环境变量：

- backend/conf.yaml：backend 为 API 地址、数据库路径、token；
  scheduler 为任务并发数、记录保留条数、任务和通知超时。
- frontend/conf.yaml：web 为 Web 地址、站点源和登录账号密码；
  backend.url 为代理目标，backend.token 为内部鉴权令牌。

两份 backend.token 必须相同，联合启动会检查地址和 token 是否一致。
前端配置不包含数据库和调度参数，后端配置不包含管理台登录密码。
`${NAME}` 等文本也按字面值使用。数据库相对路径以后端配置文件目录为准。
两份 conf.yaml 已忽略；请勿将实际密钥写回配置模板。修改配置后重启服务。

```sh
make build
./backend/bin/taskflow -config backend/conf.yaml
# 前端单独指定配置文件：
cd frontend && npm start -- --config /path/frontend.yaml
# 联合启动也可分别指定配置文件（先 make build）：
# node scripts/dev.mjs --backend-config /path/backend.yaml \
#   --frontend-config /path/frontend.yaml
```

公网设置前端 web.origin 为实际 HTTPS 域名，只暴露 Caddy HTTPS 入口，
前后端保持本机监听。登录密码至少 12 字符，token 至少 16 字符。
Web 网关验证登录及 CSRF 后再代理请求；token 只在服务器之间传递，
不会下发浏览器。公网反向代理见 [前端说明](frontend/README.md)。

## 调度和执行

管理台默认使用时间选择，不必填写表达式。API 也支持结构化计划：

```json
{"kind":"daily","time":"09:00","timezone":"Asia/Shanghai"}
{"kind":"weekly","time":"10:30","weekdays":[1,3,5]}
{"kind":"monthly","time":"08:00","day":15}
{"kind":"hourly","minute":30}
{"kind":"interval","every":5,"unit":"m"}
```

`weekdays` 使用 0–6 表示周日到周六；间隔单位支持 s/m/h/d。
结构化字段由后端统一校验并生成表达式，月度计划若选择的日期不存在，
该月跳过。编辑已有简单 Cron 时自动显示相应时间选择；复杂表达式仍可
在「高级自定义计划」编辑，不会被默默转换为不等价的固定间隔。

- 五段 Cron：`分 时 日 月 周`，支持 `*`、`1,3`、`1-5`、`*/10`。
  周日支持 `0` 或 `7`；日和周同时受限时采用 OR 规则。
  不支持秒字段、月份英文名或 `@daily` 等别名。
- Cron 默认时区为 `Asia/Shanghai`，可以配置 IANA 时区。
  管理台展示时间统一使用上海时区，并支持预览后续五次执行。
- 固定间隔使用 Go 时长格式：`30s`、`5m`、`1h30m`，最短 1 秒。
  周期以调度触发时间计算，手动运行不会重置定时计划。
- 调度器每秒检查到期任务。同一任务同时最多一次执行；任务重叠或
  全局工作池已满时，定时触发被跳过并记录 `skipped`，不堆积队列。
  此时手动触发返回 HTTP 409。
- 总超时覆盖所有重试和等待；最多重试十次，采用固定重试间隔。
  最多执行次数为 `1 + retries`。失败输出保留最后一次尝试的结果。
- 暂停只停止未来的定时触发，不取消已经执行的任务，仍可手动运行。
  配置修改只影响后续执行；正在执行的任务使用启动时的配置快照。
- 支持主动取消，任务插件必须响应 `context.Context`。
  Go 无法强制停止不响应 Context 的插件，请为 SDK/网络操作传递 Context。
- 执行状态包括运行中、成功、失败、取消、中断和跳过；详情保留输出、
  尝试次数、耗时及通知投递结果。输出限制约 64 KiB，错误约 8 KiB。
- 任务、渠道、关联关系、运行记录及通知结果保存在 SQLite 中。
  状态变化通过事务增量写入；文件锁禁止多个进程使用同一数据库。
  重启后重新计算未来执行时间，不补跑停机期间错过的任务；
  未完成的记录标记为 `interrupted`，不会自动重放。
- 收到 SIGINT/SIGTERM 后取消活动任务并等待退出，最长等待 15 秒。

## 通知

支持控制台、通用 Webhook 和 Bark 通知。任务可关联多个渠道，支持
`failure`（仅失败）、`always`（每次执行完成）、`never` 三种策略。
取消和跳过不会触发 `failure` 策略。每个渠道默认 10 秒投递超时。
通知失败不改变任务执行结果，不自动重试；详情中可查看投递错误。
每次执行会固定渠道配置快照。手动测试渠道可在暂停状态下使用。
业务完成即释放任务并发名额，通知使用独立的并发限制（与 workers 数相同）。
通知超时包含等待投递名额的时间；取消运行也会取消该运行正在等待或发送的通知，
已完成的业务结果保持原样。

Webhook 配置：

```json
{
  "url": "https://your-service.example/webhook",
  "headers": {"Authorization": "Bearer your-secret"}
}
```

接收端收到 POST JSON：

```json
{
  "title": "[succeeded] 示例任务",
  "message": "业务执行输出",
  "run": {
    "id": "execution-id",
    "task_id": "task-id",
    "task_name": "示例任务",
    "status": "succeeded"
  }
}
```

实际 `run` 还包含开始/结束时间、耗时、输出等字段。通用 Webhook
需要接收端支持上述结构；特定机器人或邮件服务的协议可通过 Notifier
插件适配。HTTP 任务和通知默认接受 2xx，不自动跟随重定向。

### Bark 通知与喝水提醒

管理台的「通知渠道」中添加渠道，类型选择「Bark 手机通知」：

```json
{
  "server_url": "https://api.day.app",
  "device_key": "你的 Bark 设备 Key",
  "group": "喝水提醒",
  "sound": "minuet"
}
```

从 Bark App 中获取设备 Key。例如 App 提供的地址为
`https://api.day.app/abc123/`，则 `device_key` 填 `abc123`，
`server_url` 填 `https://api.day.app`，不要填写含设备 Key 的完整推送地址。
`server_url` 可省略，默认使用上述服务器，也支持自建服务器与路径前缀。
`group`、`sound` 为可选参数；sound 使用 Bark 支持的铃声名。
插件向服务器的 `/push` 接口发送 JSON，HTTP 2xx 且响应 `code` 为 200
才视为成功；不会跟随重定向。可以先使用渠道的「测试」按钮验证连接。

同一渠道批量通知多个设备，使用 `device_keys` 数组替代 `device_key`：

```json
{
  "server_url": "https://api.day.app",
  "device_keys": ["设备A的Key", "设备B的Key", "设备C的Key"],
  "group": "喝水提醒",
  "sound": "minuet"
}
```

两种字段只能填写一种；旧单设备配置继续可用。设备 Key 不能为空或包含
空白字符，重复 Key 自动去重。同一批设备共用服务器、标题、正文和铃声；
不同服务器或不同参数请创建多个通知渠道，并让任务勾选这些渠道。

插件逐设备调用 `/push`，最多 4 个并发请求，兼容现有 Bark 服务端。
单个设备失败后继续向其余设备发送；全部成功才将渠道标记为 `sent`，
部分失败标记为 `failed`，错误包含成功数和失败设备序号（按去重后的顺序），
最多列出 10 项错误。渠道的「测试」也会向全部配置的设备发送。
整批共享 scheduler.notification_timeout 配置的渠道超时，超时或取消后停止继续发送，不自动重试；
重复手动测试或运行任务会再次通知已经成功的设备。

然后新建任务，类型选择「喝水提醒」，参数示例：

```json
{"message":"该喝水啦，休息一下，喝杯水吧。"}
```

参数也可填写 `{}` 使用默认文案。执行计划选择「每小时」并设置分钟，
或选择「固定间隔」每 1 小时；通知策略选择「每次执行完成」，勾选刚创建的
Bark 渠道。保存后可立即手动运行检查手机推送。
每小时和固定间隔计划会全天触发；限定工作时间可使用高级 Cron，
例如 `0 9-18 * * 1-5` 表示工作日 09:00–18:00 每整点提醒。

喝水任务只生成文案，通知由引擎统一投递，因此可以同时绑定多个渠道，
也可以切换成其他通知方式。Bark 推送失败会记录在通知投递详情中，
任务本身仍显示执行成功。实现分别位于
`backend/plugins/reminder/drink_water.go` 和
`backend/plugins/bark/bark.go`，通过 `cmd/taskflow/plugins.go` 注册。

## 添加新任务和通知方式

统一的插件接口、任务接口、通知接口：

```go
type Plugin interface {
    Spec() taskflow.Spec
    Validate(json.RawMessage) error
}

type TaskHandler interface {
    Plugin
    Run(context.Context, json.RawMessage) (string, error)
}

type Notifier interface {
    Plugin
    Send(context.Context, json.RawMessage, taskflow.Notification) error
}
```

任务与通知复用 `PluginFactory[T]` 的 Register / Create / Specs 契约。
`Registry` 组合 `TaskFactory` 和 `NotifierFactory`，调度器只通过工厂
创建具体实现。工厂在注册时检查类型、元信息及示例参数；创建 Engine
时固定注册快照。注册构造器而不是共享实例，每次任务执行和通知投递
创建独立对象；同一次执行的重试复用该执行的对象。

在 `backend/cmd/taskflow/plugins.go` 注册：

```go
func registerPlugins(r *taskflow.Registry) error {
    if err := r.Tasks.Register(func() taskflow.TaskHandler {
        return &InventorySyncTask{}
    }); err != nil {
        return err
    }
    return r.Notifiers.Register(func() taskflow.Notifier {
        return &MailNotifier{}
    })
}
```

`InventorySyncTask`、`MailNotifier` 是需要开发者实现的业务类型。
构造器应快速、无外部副作用，并返回新的实例；可以注入并发安全的
数据库或 SDK 客户端。修改任务/渠道的 type 和 config 即可切换实现。
管理台从工厂目录读取名称、说明和参数示例，无需修改前端或 API。

轻量业务可使用 `TaskFunc`、`NotifierFunc` 函数适配器，提供元信息、
参数校验函数和执行/投递函数即可。完整可编译示例在
`backend/plugins/demo/plugins.go`：求和任务与审计通知。
启用时把注册入口改为：

```go
package main

import (
    "github.com/caicaing/projects/tasks/backend/plugins/demo"
    "github.com/caicaing/projects/tasks/framework"
)

func registerPlugins(r *taskflow.Registry) error {
    return plugins.Register(r)
}
```

重新编译后管理台即可选择新增类型。通用 HTTP/日志任务和自定义任务
使用同一个调度流程，统一享有计划、超时、重试、取消、记录和通知能力。
插件必须传递 Context，不响应取消的代码无法被 Go 强制停止。
构造、参数校验、任务执行与通知投递的 panic 转换为错误；校验应无副作用。

## API

| 方法 | 地址 | 用途 |
| --- | --- | --- |
| GET | `/healthz` | 存活检查 |
| GET | `/api/state` | 任务、渠道、保留的运行记录 |
| GET | `/api/catalog` | 插件类型和参数示例 |
| POST | `/api/tasks` | 创建任务 |
| PUT / DELETE | `/api/tasks/{id}` | 替换配置 / 删除任务 |
| POST | `/api/tasks/{id}/run` | 手动运行，返回执行记录及 ID |
| POST | `/api/runs/{id}/cancel` | 取消运行中的任务 |
| POST | `/api/channels` | 创建通知渠道 |
| PUT / DELETE | `/api/channels/{id}` | 替换配置 / 删除渠道 |
| POST | `/api/channels/{id}/test` | 发送测试通知 |
| POST | `/api/schedules/preview` | 预览后续五次执行 |

PUT 是完整替换，不是部分更新；请求 JSON，错误格式为 `{"error":"…"}`。
正在执行的任务不能删除；仍被任务引用的渠道不能删除。

创建任务请求示例：

```json
{
  "name": "每天生成日报",
  "description": "调用已有报表接口",
  "type": "http",
  "config": {
    "url": "https://your-service.example/reports/daily",
    "method": "POST",
    "body": "{}",
    "expected_status": 200
  },
  "schedule": {
    "kind": "daily",
    "time": "09:00",
    "timezone": "Asia/Shanghai"
  },
  "enabled": true,
  "timeout": "30s",
  "retries": 2,
  "retry_delay": "2s",
  "notify_on": "failure",
  "notifier_ids": []
}
```

## 一键公网部署

Debian 13.2 服务器只需 systemd 正常运行，拉取项目后执行：

```sh
sudo bash scripts/deploy.sh --domain taskflow.example.com
```

首次运行输入并确认登录密码（至少 12 字符），账号默认 `admin`，
也可通过 `--user` 指定。后端 Token 自动生成，密码不会在终端显示。
脚本自动安装缺失或版本过旧的 Go、Node.js、npm 和 Caddy，支持
x86_64 和 ARM64。已有 Go 1.23+、Node.js 18+（含 npm）和 Caddy 2
可用时直接复用；新安装使用 Go 官方稳定版、Node.js 24 LTS 和
Caddy 官方稳定版，并校验下载文件的官方 SHA-256/SHA-512 散列。
运行时安装到 /opt/taskflow-runtimes，不依赖个人目录或登录 Shell。
自动安装通过 apt 安装下载工具，并从 Go、Node.js 官方站点及 GitHub
下载运行时，不会额外启动默认的 caddy.service。
自动安装检测 Debian 13 主版本，兼容 13.2 及后续小版本更新。
其他 Linux 发行版需自行准备 Go、Node.js、npm 和 Caddy；
Node.js 和 Caddy 需安装在系统级路径，服务用户能够访问。
脚本构建后端、安装前端依赖，创建专用的 taskflow 系统用户，并安装
taskflow.service 和 taskflow-caddy.service，由 systemd 管理进程与开机启动。
Caddy 是唯一的公网代理，自动申请和续期 HTTPS 证书。
脚本不修改防火墙，也不改已有的 Caddy 服务配置。

域名必须先解析到服务器公网 IP；安全组及防火墙放行 TCP 80、443，
宿主机这些端口需空闲。Caddy 安装包可能自动启动默认 caddy.service，
使用本脚本前需确保它未占用这些端口。构建依赖和签发证书均需联网。
如果已有 Caddy 管理其他站点，可按 deploy/Caddyfile 添加本站点，
自行管理前后端进程；不要同时启动两个占用相同端口的 Caddy 实例。

部署配置保存在 `.taskflow-deploy/`，不读取环境变量：

- backend.yaml：后端配置，监听 127.0.0.1:8081。
- frontend.yaml：Web 网关配置，包含 HTTPS 站点源、登录凭据及后端地址。
- Caddyfile：HTTPS 代理配置，域名必须与前端 web.origin 一致。

源配置目录权限为 700，文件为 600，包含明文密钥且已加入 Git 忽略。
请保护和备份。脚本将配置复制到 /etc/taskflow，文件权限为 640，
仅 root 和 taskflow 服务用户可读取；修改源配置后重新部署会重启服务。
修改 token 时须同步修改两份 YAML。安装目录与数据位置如下：

| 路径 | 内容 |
| --- | --- |
| /opt/taskflow | 后端可执行文件、前端页面、依赖和启动脚本 |
| /opt/taskflow-runtimes | 自动安装的 Go、Node.js、npm 和 Caddy |
| /etc/taskflow | 服务实际读取的两份 YAML 与 Caddyfile |
| /var/lib/taskflow/taskflow.db | SQLite 数据库 |
| /var/lib/taskflow/caddy | Caddy 证书和运行数据 |

前后端以 taskflow 用户运行，分别只监听 127.0.0.1:8081 和
127.0.0.1:8080；Caddy 通过受限权限绑定 80、443。
更新代码、配置或停止服务会保留数据库和证书。
旧部署的数据不会自动迁移，切换前应备份数据库并迁入上述路径，
确保数据库及其目录由 taskflow 用户读写。

脚本确认后端鉴权可用及 HTTPS 登录页可访问后才报告完成。
首次签发证书可能需要等待；120 秒内尚未就绪时，脚本返回失败但保留服务
继续运行，可检查日志、修复 DNS/端口配置后再次执行。后续无需手动续期。

```sh
# 后续部署、查看状态、日志或停止服务，复用保存的配置。
sudo bash scripts/deploy.sh
sudo bash scripts/deploy.sh --status
sudo bash scripts/deploy.sh --logs
sudo bash scripts/deploy.sh --stop
```

非交互部署时，将 backend/conf.example.yaml 和 frontend/conf.example.yaml
复制为 .taskflow-deploy/backend.yaml 和 frontend.yaml，填写账号、密码和
相同的 token；后端 database 改为 /var/lib/taskflow/taskflow.db，
前端 web.origin 改为实际 HTTPS 源。将 deploy/Caddyfile 复制到该目录并
替换域名，再执行 sudo bash scripts/deploy.sh。
数据库须使用 /var/lib/taskflow/ 下的绝对路径。
完整请求路径为：公网 HTTPS → Caddy → Node 登录网关 → Go 后端。

## 验证与边界

```sh
make check
```

该命令编译后端全部包、执行 `go vet`，并检查前端 JavaScript 语法。按当前开发约定，暂不新增单元测试，
后续统一补充，并以新增代码覆盖率至少 85% 为目标。

框架面向小规模、单实例使用，管理后端的 SQLite 在事务中保存变化的记录。
没有分布式锁、多租户权限、事务性通知 outbox、持久化执行队列或
恰好一次执行保障。重试的业务操作应支持幂等；高并发或多副本部署
需要进一步引入共享存储、队列及集群协调机制。
