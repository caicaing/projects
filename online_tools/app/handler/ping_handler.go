package handler

import (
	"github.com/gin-gonic/gin"
)

// PingHandler 处理ping请求
type PingHandler struct{}

// NewPingHandler 创建PingHandler实例
func NewPingHandler() *PingHandler {
	return &PingHandler{}
}

// Ping 处理ping接口
func (h *PingHandler) Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

// Health 健康检查接口
func (h *PingHandler) Health(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
	})
}
