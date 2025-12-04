package filter

import (
	"fmt"
	"time"
	
	"github.com/gin-gonic/gin"
)

// AuthMiddleware 自定义鉴权中间件：校验请求头中的 Token
func AuthMiddleware(validToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 前置逻辑：校验 Token
		token := c.GetHeader("Authorization")
		if token != validToken {
			// 鉴权失败，终止请求流程，直接返回错误
			c.Abort()
			c.JSON(401, gin.H{"error": "token 无效"})
			return
		}

		// 鉴权成功，执行后续逻辑
		c.Next()
	}
}

// CostTimeMiddleware 自定义中间件：统计请求耗时 + 打印请求信息
func CostTimeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 前置逻辑：请求到达时执行
		start := time.Now()
		fmt.Printf("请求开始：%s %s | 客户端IP：%s\n", c.Request.Method, c.Request.URL.Path, c.ClientIP())

		// 执行后续的中间件/业务函数
		c.Next()

		// 后置逻辑：响应返回后执行（c.Next() 之后）
		cost := time.Since(start)
		fmt.Printf("请求结束：%s %s | 耗时：%v | 状态码：%d\n",
			c.Request.Method, c.Request.URL.Path, cost, c.Writer.Status())
	}
}
