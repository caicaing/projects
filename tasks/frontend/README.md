# Taskflow 管理台

独立浏览器 ES Module 页面和 Node Web 网关，要求 Node.js 18+。
仅服务端读取 YAML 配置，浏览器不读取配置或后端 token。

## 启动

将本目录 conf.example.yaml 复制为 conf.yaml，填写 backend.token 和 web.password。

```sh
make frontend
# 单独启动时：
cd frontend
npm ci
npm start
```

默认读取 frontend/conf.yaml，通过命令行指定其他文件：

```sh
npm start -- --config /path/frontend.yaml
```

backend.token 用于服务器之间鉴权，须与后端配置中的 token 相同；
backend.url 为代理目标。web.username/password 用于登录，
web.host/port 是网关监听地址，web.origin 是浏览器访问的完整站点源，
如 https://taskflow.example.com。配置修改后重启。
默认本机地址为 http://127.0.0.1:8080，后端为 http://127.0.0.1:8081。

所有值直接取自配置，不读取环境变量，也不展开 ${NAME} 等文本。
本配置不包含后端数据库或调度设置。任务和通知参数由管理 API 保存到 SQLite。

## 公网部署

配置 web.origin 为公网 HTTPS 域名，HTTPS 代理保留浏览器 Host，
将所有请求代理到 Node 网关。前后端只在本机监听。
使用 Caddy 自动管理 HTTPS，配置示例见 ../deploy/Caddyfile，
将域名替换为 web.origin 中的域名，代理目标默认为 127.0.0.1:8080。
部署脚本使用 systemd 管理前后端和 Caddy，配置修改后重新部署会重启服务。
一键部署及 Caddy 自动证书见 [根目录说明](../README.md#一键公网部署)。

## 访问保护

- 页面、API 和健康检查均验证服务端会话；未登录只能访问登录页及样式。
- 密码在 Node 服务端验证，后端 token 由代理添加，不传给浏览器。
- HttpOnly / SameSite=Strict Cookie，HTTPS 使用 Secure 和 __Host- 前缀。
- 写请求要求同源 Origin 和 CSRF token；登录有限流和并发限制。
- 会话最长 8 小时，闲置 30 分钟过期；退出或网关重启会撤销会话。

## 目录

public/ 为登录页、管理页和资源，src/ 为页面模块和时间选择组件。
config.mjs 读取 YAML，auth.mjs 处理登录与会话，proxy.mjs 代理后端请求，
dev-server.mjs 是启动入口。

插件类型和参数示例来自后端工厂，无需在前端新增类型分支。
此管理台依赖本仓库管理 API；只使用 framework 的项目无需引入前端。
