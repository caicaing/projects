# Online Tools

一个基于Gin框架的在线工具服务。

## 项目结构

```
online_tools/
├── main.go                    # 应用入口
├── app/
│   ├── config/               # 配置管理
│   │   └── config.go         # 应用配置
│   ├── filter/               # 中间件
│   │   └── filter.go         # 认证、日志等中间件
│   ├── handler/              # 业务处理器
│   │   └── ping_handler.go  # Ping接口处理器
│   └── router/               # 路由管理
│       └── router.go         # 路由注册和配置
├── go.mod
└── go.sum
```

## 架构设计

### 1. 分层架构
- **main.go**: 应用启动入口，负责初始化和启动服务
- **config**: 配置层，管理应用配置
- **router**: 路由层，统一管理路由注册和中间件配置
- **handler**: 处理器层，实现具体的业务逻辑
- **filter**: 中间件层，提供认证、日志等通用功能

### 2. 设计优势
- ✅ **职责分离**: 路由、处理器、中间件各司其职
- ✅ **易于测试**: 每个handler都可以独立测试
- ✅ **易于扩展**: 新增接口只需添加handler和注册路由
- ✅ **配置集中**: 所有配置统一管理
- ✅ **代码清晰**: main函数简洁，逻辑清晰

## 快速开始

### 运行服务
```bash
go run main.go
```

### 测试接口
```bash
# Ping接口
curl -H "Authorization: 123456" http://localhost:8801/api/v1/ping

# 健康检查
curl -H "Authorization: 123456" http://localhost:8801/api/v1/health
```

## 如何添加新接口

### 1. 创建Handler
在 `app/handler/` 目录下创建新的handler文件：

```go
package handler

import "github.com/gin-gonic/gin"

type UserHandler struct{}

func NewUserHandler() *UserHandler {
    return &UserHandler{}
}

func (h *UserHandler) GetUser(c *gin.Context) {
    c.JSON(200, gin.H{"user": "example"})
}
```

### 2. 注册路由
在 `app/router/router.go` 中添加：

```go
type Router struct {
    engine      *gin.Engine
    pingHandler *handler.PingHandler
    userHandler *handler.UserHandler  // 添加新handler
}

func NewRouter(engine *gin.Engine) *Router {
    return &Router{
        engine:      engine,
        pingHandler: handler.NewPingHandler(),
        userHandler: handler.NewUserHandler(),  // 初始化
    }
}

func (r *Router) SetupRoutes() {
    v1 := r.engine.Group("/api/v1")
    {
        v1.GET("/ping", r.pingHandler.Ping)
        v1.GET("/health", r.pingHandler.Health)
        v1.GET("/user", r.userHandler.GetUser)  // 注册路由
    }
}
```

## 配置说明

在 `app/config/config.go` 中修改配置：

```go
func NewConfig() *Config {
    return &Config{
        Server: ServerConfig{
            Port: ":8801",  // 修改端口
        },
        Auth: AuthConfig{
            Token: "123456",  // 修改认证token
        },
    }
}
```

## 中间件

当前支持的中间件：
- **AuthMiddleware**: Token认证中间件
- **CostTimeMiddleware**: 请求耗时统计中间件

可在 `app/filter/filter.go` 中添加更多中间件。
