package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthHandler 处理认证相关请求
type AuthHandler struct{}

// NewAuthHandler 创建AuthHandler实例
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// Login 处理登录接口
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	// 绑定JSON请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"details": err.Error(),
		})
		return
	}

	// 简单的用户验证逻辑（实际项目中应该查询数据库）
	// 这里使用示例账号：admin/admin123
	if req.Username == "admin" && req.Password == "admin123" {
		// 生成token（实际项目中应该使用JWT）
		token := generateToken(req.Username)

		c.JSON(http.StatusOK, LoginResponse{
			Token:    token,
			Username: req.Username,
			Message:  "登录成功",
		})
		return
	}

	// 登录失败
	c.JSON(http.StatusUnauthorized, gin.H{
		"error": "用户名或密码错误",
	})
}

// generateToken 生成简单的token（实际项目中应该使用JWT）
func generateToken(username string) string {
	// 这里只是示例，实际应该使用JWT或其他安全的token生成方式
	timestamp := time.Now().Unix()
	return username + "_token_" + string(rune(timestamp))
}
