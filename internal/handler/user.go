package handler

import (
	"com.duole/datax-web-go/internal/database"
	"com.duole/datax-web-go/internal/entities"
	"com.duole/datax-web-go/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// List 显示所有用户
func (h *UserHandler) List(c *gin.Context) {
	users, err := database.GetDB().User.List()
	if err != nil {
		//todo 错误页面返回
		return
	}
	c.HTML(http.StatusOK, "user/list.tmpl", gin.H{"Users": users})
}

// NewForm 显示创建新用户的表单
func (h *UserHandler) NewForm(c *gin.Context) {
	c.HTML(http.StatusOK, "user/form.tmpl", gin.H{})
}

// Create 创建新用户
func (h *UserHandler) Create(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	role := c.PostForm("role")
	password := c.PostForm("password")

	hashed, err := service.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	createdBy := c.GetInt("user_id")
	user := &entities.User{
		Username:  username,
		Role:      role,
		Password:  hashed,
		CreatedBy: &createdBy,
		UpdatedBy: &createdBy,
	}

	err = database.GetDB().User.Create(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	c.Redirect(http.StatusFound, "/admin/users")
}

// Toggle 切换用户启用/禁用状态
func (h *UserHandler) Toggle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	updatedBy := c.GetInt("user_id")
	err = database.GetDB().User.Toggle(id, updatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户状态更新成功"})
}
