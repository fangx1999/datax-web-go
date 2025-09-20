package handler

import (
	"com.duole/datax-web-go/internal/database"
	"com.duole/datax-web-go/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// ShowLogin 显示登录页面
func (h *AuthHandler) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.tmpl", gin.H{})
}

// DoLogin 处理登录
func (h *AuthHandler) DoLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if _, err := h.auth.Login(c.Writer, c.Request, username, password); err != nil {
		c.HTML(http.StatusUnauthorized, "login.tmpl", gin.H{"Error": "用户名或密码错误"})
		return
	}

	c.Redirect(http.StatusFound, "/tasks")
}

// Logout 处理登出
func (h *AuthHandler) Logout(c *gin.Context) {
	h.auth.Logout(c.Writer, c.Request)
	c.Redirect(http.StatusFound, "/login")
}

// MustLogin 确保用户已登录
func (h *AuthHandler) MustLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := h.auth.CurrentUser(c.Request)
		if user == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 从数据库获取用户ID
		userEntity, err := database.GetDB().User.GetByUsername(user)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("user_id", userEntity.ID)
		c.Next()
	}
}

// MustAdmin 确保用户是管理员
func (h *AuthHandler) MustAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, role := h.auth.CurrentUser(c.Request)
		if role != "admin" {
			c.String(http.StatusForbidden, "Forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}
