package controllers

import (
	"net/http"

	"com.duole/datax-web-go/internal/services"
	"github.com/gin-gonic/gin"
)

// AuthController 处理认证相关的 HTTP 请求
type AuthController struct {
	auth *services.AuthService
}

// NewAuthController 创建新的认证控制器
func NewAuthController(auth *services.AuthService) *AuthController {
	return &AuthController{
		auth: auth,
	}
}

// 中间件确保用户已登录。如果未登录，重定向到登录页面。
func (ac *AuthController) MustLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := ac.auth.CurrentUser(c.Request)
		if user == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}

// 中间件确保用户具有管理员角色。非管理员用户
// 被禁止访问包装的路由。
func (ac *AuthController) MustAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, role := ac.auth.CurrentUser(c.Request)
		if role != "admin" {
			c.String(403, "Forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}

// ShowLogin 渲染登录页面
func (ac *AuthController) ShowLogin(c *gin.Context) {
	c.HTML(200, "login.tmpl", gin.H{})
}

// DoLogin 处理登录表单提交
func (ac *AuthController) DoLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	if _, err := ac.auth.Login(c.Writer, c.Request, username, password); err != nil {
		c.HTML(401, "login.tmpl", gin.H{"Error": "用户名或密码错误"})
		return
	}
	c.Redirect(302, "/tasks")
}

// Logout 处理登出
func (ac *AuthController) Logout(c *gin.Context) {
	ac.auth.Logout(c.Writer, c.Request)
	c.Redirect(302, "/login")
}
