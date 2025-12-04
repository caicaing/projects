package main

import (
	"github.com/caicaing/projects/online_tools/app/config"
	"github.com/caicaing/projects/online_tools/app/router"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.NewConfig()
	// 创建Gin引擎
	engine := gin.Default()
	// 创建路由管理器并设置路由
	r := router.NewRouter(engine)
	r.Setup(cfg.Auth.Token)
	// 启动服务
	engine.Run(cfg.Server.Port)
}
