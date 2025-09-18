package controllers

import (
	"com.duole/datax-web-go/internal/models"
	"github.com/gin-gonic/gin"
	"log"
)

// GetDataSourcesByType 按类型检索数据源
func (ct *Controller) GetDataSourcesByType(srcType string) ([]models.DataSource, error) {
	rows, err := ct.db.Query("SELECT id,name FROM data_sources WHERE type=?", srcType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []models.DataSource
	for rows.Next() {
		var ds models.DataSource
		if err := rows.Scan(&ds.ID, &ds.Name); err != nil {
			log.Printf("Error scanning data source: %v", err)
			continue
		}
		sources = append(sources, ds)
	}
	return sources, nil
}

// GetCurrentUserID 从请求中获取当前用户 ID
func (ct *Controller) GetCurrentUserID(c *gin.Context) int {
	user, _ := ct.auth.CurrentUser(c.Request)
	var uid int
	ct.db.QueryRow("SELECT id FROM users WHERE username=?", user).Scan(&uid)
	return uid
}
