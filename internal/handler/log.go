package handler

import (
	"com.duole/datax-web-go/internal/database"
	"com.duole/datax-web-go/internal/service"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
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

// GetTaskLogs 获取任务执行日志列表
func (h *LogHandler) GetTaskLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := strings.TrimSpace(c.Query("status"))
	executionType := strings.TrimSpace(c.Query("execution_type"))
	taskName := strings.TrimSpace(c.Query("task_name"))

	parseDate := func(val string) (*time.Time, error) {
		if strings.TrimSpace(val) == "" {
			return nil, nil
		}
		t, err := time.ParseInLocation("2006-01-02", val, time.Local)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}

	dateFrom, err := parseDate(c.Query("date_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期"})
		return
	}

	dateTo, err := parseDate(c.Query("date_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期"})
		return
	}

	if dateTo != nil {
		endOfDay := dateTo.Add(24*time.Hour - time.Nanosecond)
		dateTo = &endOfDay
	}

	filters := database.TaskLogFilters{
		Status:        status,
		ExecutionType: executionType,
		TaskName:      taskName,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
	}

	result, err := database.GetDB().Log.GetTaskLogs(page, pageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务日志失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":        result.Logs,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total":       result.Total,
		"total_pages": result.TotalPages,
	})
}

// GetTaskLogDetail 获取任务日志详情（支持JSON和HTML）
func (h *LogHandler) GetTaskLogDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日志ID"})
		return
	}

	logEntry, err := database.GetDB().Log.GetTaskLogDetail(id)
	if err != nil {
		if errors.Is(err, database.ErrTaskLogNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "日志不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志详情失败: " + err.Error()})
		}
		return
	}

	if c.GetHeader("Accept") == "application/json" || strings.HasPrefix(c.FullPath(), "/api/") {
		c.JSON(http.StatusOK, gin.H{
			"id":                logEntry.ID,
			"task_id":           logEntry.TaskID,
			"task_name":         logEntry.TaskName,
			"status":            logEntry.Status,
			"execution_type":    logEntry.ExecutionType,
			"start_time":        logEntry.StartTime,
			"end_time":          logEntry.EndTime,
			"duration":          logEntry.Duration,
			"flow_execution_id": logEntry.FlowExecutionID,
			"step_id":           logEntry.StepID,
			"step_order":        logEntry.StepOrder,
			"content":           logEntry.LogContent,
			"created_at":        logEntry.CreatedAt,
		})
		return
	}

	c.HTML(http.StatusOK, "task_log/detail.tmpl", gin.H{"Log": logEntry})
}

// FlowLogList 显示任务流日志列表页面
func (h *LogHandler) FlowLogList(c *gin.Context) {
	c.HTML(http.StatusOK, "flow_log/list.tmpl", gin.H{})
}

// GetFlowLogs 获取任务流日志列表
func (h *LogHandler) GetFlowLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := strings.TrimSpace(c.Query("status"))
	executionType := strings.TrimSpace(c.Query("execution_type"))
	flowName := strings.TrimSpace(c.Query("flow_name"))

	parseDate := func(val string) (*time.Time, error) {
		if strings.TrimSpace(val) == "" {
			return nil, nil
		}
		t, err := time.ParseInLocation("2006-01-02", val, time.Local)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}

	dateFrom, err := parseDate(c.Query("date_from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期"})
		return
	}

	dateTo, err := parseDate(c.Query("date_to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期"})
		return
	}

	if dateTo != nil {
		endOfDay := dateTo.Add(24*time.Hour - time.Nanosecond)
		dateTo = &endOfDay
	}

	filters := database.FlowLogFilters{
		Status:        status,
		ExecutionType: executionType,
		FlowName:      flowName,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
	}

	result, err := database.GetDB().Log.GetFlowLogs(page, pageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务流日志失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": result.Logs,
		"pagination": gin.H{
			"current_page": result.Page,
			"page_size":    result.PageSize,
			"total_pages":  result.TotalPages,
			"total":        result.Total,
		},
	})
}

// GetFlowLogDetail 获取任务流日志详情
func (h *LogHandler) GetFlowLogDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日志ID"})
		return
	}

	response, err := database.GetDB().Log.GetFlowLogDetail(id)
	if err != nil {
		if errors.Is(err, database.ErrFlowLogNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "日志不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务流日志详情失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             response.Log.ID,
		"flow_id":        response.Log.FlowID,
		"flow_name":      response.Log.FlowName,
		"status":         response.Log.Status,
		"execution_type": response.Log.ExecutionType,
		"start_time":     response.Log.StartTime,
		"end_time":       response.Log.EndTime,
		"duration":       response.Log.Duration,
		"content":        response.Log.LogContent,
	})
}

// KillTaskByLog 根据日志ID终止任务
func (h *LogHandler) KillTaskByLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日志ID"})
		return
	}

	logEntry, err := database.GetDB().Log.GetTaskLogDetail(id)
	if err != nil {
		if errors.Is(err, database.ErrTaskLogNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "日志不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询日志信息失败: " + err.Error()})
		return
	}

	if err := service.Get().Scheduler.KillTask(logEntry.TaskID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "任务已终止"})
}

// KillFlowByLog 根据日志ID终止任务流
func (h *LogHandler) KillFlowByLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日志ID"})
		return
	}

	response, err := database.GetDB().Log.GetFlowLogDetail(id)
	if err != nil {
		if errors.Is(err, database.ErrFlowLogNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "日志不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询日志信息失败: " + err.Error()})
		return
	}

	if err := service.Get().Scheduler.KillTaskFlow(response.Log.FlowID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "任务流已终止"})
}
