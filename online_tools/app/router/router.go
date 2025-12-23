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
	authHandler *handler.AuthHandler
}

// NewRouter 创建路由管理器
func NewRouter(engine *gin.Engine) *Router {
	return &Router{
		engine:      engine,
		pingHandler: handler.NewPingHandler(),
		authHandler: handler.NewAuthHandler(),
	}
}

// SetupMiddlewares 设置全局中间件
func (r *Router) SetupMiddlewares(authToken string) {
	// 添加CORS中间件和耗时统计中间件（全局生效）
	r.engine.Use(
		filter.CORSMiddleware(),
		filter.CostTimeMiddleware(),
	)
}

// SetupRoutes 设置所有路由
func (r *Router) SetupRoutes(authToken string) {
	// API v1 路由组
	v1 := r.engine.Group("/api/v1")
	{
		// 公开路由（不需要token验证）
		v1.POST("/login", r.authHandler.Login)
	}

	// 需要认证的路由组
	v1Auth := r.engine.Group("/api/v1")
	v1Auth.Use(filter.AuthMiddleware(authToken))
	{
		// Ping相关路由
		v1Auth.GET("/ping", r.pingHandler.Ping)
		v1Auth.GET("/health", r.pingHandler.Health)
	}
}

// Setup 一键设置中间件和路由
func (r *Router) Setup(authToken string) {
	r.SetupMiddlewares(authToken)
	r.SetupRoutes(authToken)
}
