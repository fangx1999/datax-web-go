package controllers

import (
	"com.duole/datax-web-go/internal/models"
	"com.duole/datax-web-go/internal/services"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

// UserList 显示所有用户
func (ct *Controller) UserList(c *gin.Context) {
	rows, _ := ct.db.Query(`
		SELECT 
			u.id, u.username, u.role, u.disabled,
		    COALESCE(uc.username, '系统') as created_by_name,
		    COALESCE(uu.username, '系统') as updated_by_name,
		    u.created_at
		FROM users u
		LEFT JOIN users uc ON u.created_by = uc.id
		LEFT JOIN users uu ON u.updated_by = uu.id
		ORDER BY u.id DESC
	`)
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		rows.Scan(&u.ID, &u.Username, &u.Role, &u.Disabled, &u.CreatedByName, &u.UpdatedByName, &u.CreatedAt)
		users = append(users, u)
	}
	c.HTML(200, "user/list.tmpl", gin.H{"Users": users})
}

// UserNewForm 显示创建新用户的表单
func (ct *Controller) UserNewForm(c *gin.Context) {
	c.HTML(200, "user/form.tmpl", gin.H{})
}

// UserCreate 创建新用户
func (ct *Controller) UserCreate(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	role := c.PostForm("role")
	password := c.PostForm("password")

	// 获取当前用户ID
	createdBy := ct.GetCurrentUserID(c)
	hashed, _ := services.HashPassword(password)
	ct.db.Exec(`INSERT INTO users(username, password, role, disabled, created_by, updated_by) 
		VALUES (?, ?, ?, 0, ?, ?)`, username, hashed, role, createdBy, createdBy)

	c.Redirect(302, "/admin/users")
}

// UserToggle 切换用户启用/禁用状态
func (ct *Controller) UserToggle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的用户ID"})
		return
	}

	// 检查用户是否存在
	var exists bool
	err = ct.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id=?)", id).Scan(&exists)
	if err != nil {
		c.JSON(500, gin.H{"error": "查询用户失败"})
		return
	}
	if !exists {
		c.JSON(404, gin.H{"error": "用户不存在"})
		return
	}

	// 获取当前用户ID
	updatedBy := ct.GetCurrentUserID(c)
	result, err := ct.db.Exec("UPDATE users SET disabled=1-disabled, updated_by=? WHERE id=?", updatedBy, id)
	if err != nil {
		c.JSON(500, gin.H{"error": "更新用户状态失败"})
		return
	}

	// 检查是否真的更新了记录
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(400, gin.H{"error": "用户状态未发生变化"})
		return
	}

	c.JSON(200, gin.H{"message": "用户状态更新成功"})
}
