package controllers

import (
	"database/sql"
	"net/http"

	"com.duole/datax-web-go/internal/services/datax"
	"github.com/gin-gonic/gin"
)

// DataXController 处理 DataX 相关的 HTTP 请求
type DataXController struct {
	dataxService *datax.Service
}

// NewDataXController 创建新的 DataX 控制器
func NewDataXController(db *sql.DB) *DataXController {
	return &DataXController{
		dataxService: datax.NewService(db),
	}
}

func (ct *DataXController) DataXConfPreview(c *gin.Context) {
	var req datax.ConfigRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求体无效",
		})
		return
	}

	// 使用 DataX 配置服务处理请求
	response := ct.dataxService.GenerateConfig(req)

	// 根据验证错误返回适当的HTTP状态码
	if !response.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   response.Error,
		})
		return
	}

	// 返回成功结果
	c.JSON(http.StatusOK, response.Data)
}
