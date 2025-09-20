package handler

import (
	"com.duole/datax-web-go/internal/service/datax"
	"github.com/gin-gonic/gin"
	"net/http"
)

// DataXHandler DataX处理器
type DataXHandler struct {
	dataxService *datax.Service
}

// NewDataXHandler 创建DataX处理器
func NewDataXHandler(dataxService *datax.Service) *DataXHandler {
	return &DataXHandler{dataxService: dataxService}
}

// ConfigPreview DataX配置预览
func (h *DataXHandler) ConfigPreview(c *gin.Context) {
	var req datax.ConfigRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求体无效",
		})
		return
	}

	// 使用 DataX 配置服务处理请求
	response := h.dataxService.GenerateConfig(req)

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
