package router

import (
	"github.com/caicaing/projects/online_tools/app/filter"
	"github.com/caicaing/projects/online_tools/app/handler"
	"github.com/gin-gonic/gin"
)

// Router 路由管理器
type Router struct {
	engine      *gin.Engine
	pingHandler *handler.PingHandler
}

// NewRouter 创建路由管理器
func NewRouter(engine *gin.Engine) *Router {
	return &Router{
		engine:      engine,
		pingHandler: handler.NewPingHandler(),
	}
}

// SetupMiddlewares 设置全局中间件
func (r *Router) SetupMiddlewares(authToken string) {
	r.engine.Use(
		filter.AuthMiddleware(authToken),
		filter.CostTimeMiddleware(),
	)
}

// SetupRoutes 设置所有路由
func (r *Router) SetupRoutes() {
	// API v1 路由组
	v1 := r.engine.Group("/api/v1")
	{
		// Ping相关路由
		v1.GET("/ping", r.pingHandler.Ping)
		v1.GET("/health", r.pingHandler.Health)
	}
}

// Setup 一键设置中间件和路由
func (r *Router) Setup(authToken string) {
	r.SetupMiddlewares(authToken)
	r.SetupRoutes()
}
