package handler

import (
	"com.duole/datax-web-go/internal/database"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// LogHandler 日志处理器
type LogHandler struct{}

// NewLogHandler 创建日志处理器
func NewLogHandler() *LogHandler {
	return &LogHandler{}
}

// TaskLogList 显示任务日志列表页面
func (h *LogHandler) TaskLogList(c *gin.Context) {
	c.HTML(http.StatusOK, "task_log/list.tmpl", gin.H{})
}

// GetTaskLogs 获取任务执行日志列表（支持独立任务和任务流步骤）
func (h *LogHandler) GetTaskLogs(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	taskID := c.Query("task_id")

	// 调用database层获取日志
	response, err := database.GetDB().Log.GetTaskLogs(page, pageSize, taskID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务日志失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetTaskLogDetail 获取任务日志详情（支持JSON和HTML）
func (h *LogHandler) GetTaskLogDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日志ID"})
		return
	}

	// 调用database层获取日志详情
	log, err := database.GetDB().Log.GetTaskLogDetail(id)
	if err != nil {
		if err.Error() == "日志不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": "日志不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志详情失败: " + err.Error()})
		}
		return
	}

	// 根据Accept头决定返回格式
	if c.GetHeader("Accept") == "application/json" {
		c.JSON(http.StatusOK, log)
	} else {
		c.HTML(http.StatusOK, "task_log/detail.tmpl", gin.H{"Log": log})
	}
}

// FlowLogList 显示任务流日志列表页面
func (h *LogHandler) FlowLogList(c *gin.Context) {
	c.HTML(http.StatusOK, "flow_log/list.tmpl", gin.H{})
}

// GetFlowLogs 获取任务流日志列表
func (h *LogHandler) GetFlowLogs(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	flowID := c.Query("flow_id")

	// 调用database层获取日志
	response, err := database.GetDB().Log.GetFlowLogs(page, pageSize, flowID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务流日志失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetFlowLogDetail 获取任务流日志详情
func (h *LogHandler) GetFlowLogDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日志ID"})
		return
	}

	// 调用database层获取日志详情
	response, err := database.GetDB().Log.GetFlowLogDetail(id)
	if err != nil {
		if err.Error() == "日志不存在" {
			c.JSON(http.StatusNotFound, gin.H{"error": "日志不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志详情失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}
