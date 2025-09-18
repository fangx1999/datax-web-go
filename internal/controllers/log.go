package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"com.duole/datax-web-go/internal/models"
	"github.com/gin-gonic/gin"
)

// LogController 统一的日志控制器
type LogController struct {
	db *sql.DB
}

// NewLogController 创建日志控制器
func NewLogController(db *sql.DB) *LogController {
	return &LogController{db: db}
}

// GetTaskLogs 获取任务执行日志列表（支持独立任务和任务流步骤）
func (lc *LogController) GetTaskLogs(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	taskName := c.Query("task_name")
	executionType := c.Query("execution_type")       // scheduled, manual
	executionContext := c.Query("execution_context") // standalone, flow_step
	flowExecutionID := c.Query("flow_execution_id")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if status != "" {
		whereClause += " AND tl.status = ?"
		args = append(args, status)
	}

	if taskName != "" {
		whereClause += " AND t.name LIKE ?"
		args = append(args, "%"+taskName+"%")
	}

	if executionType != "" {
		whereClause += " AND tl.execution_type = ?"
		args = append(args, executionType)
	}

	if executionContext != "" {
		whereClause += " AND tl.execution_context = ?"
		args = append(args, executionContext)
	}

	if flowExecutionID != "" {
		whereClause += " AND tl.flow_execution_id = ?"
		args = append(args, flowExecutionID)
	}

	if dateFrom != "" {
		whereClause += " AND DATE(tl.start_time) >= ?"
		args = append(args, dateFrom)
	}

	if dateTo != "" {
		whereClause += " AND DATE(tl.start_time) <= ?"
		args = append(args, dateTo)
	}

	// 查询总数
	countQuery := `
		SELECT COUNT(*) 
		FROM task_logs tl
		LEFT JOIN tasks t ON tl.task_id = t.id
		` + whereClause

	var total int
	err := lc.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询任务日志总数失败: " + err.Error(),
		})
		return
	}

	// 计算分页
	totalPages := (total + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize

	// 查询任务执行日志列表
	query := `
		SELECT tl.id, tl.task_id, t.name as task_name, tl.execution_context,
		       tl.flow_execution_id, tl.step_id, tl.step_order,
		       tl.status, tl.execution_type, tl.start_time, tl.end_time, tl.log, tl.created_at
		FROM task_logs tl
		LEFT JOIN tasks t ON tl.task_id = t.id
		` + whereClause + `
		ORDER BY tl.start_time DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)
	rows, err := lc.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询任务日志失败: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var logs []models.TaskExecutionLog
	for rows.Next() {
		var log models.TaskExecutionLog
		var endTime sql.NullTime
		var flowExecutionID, stepID, stepOrder sql.NullInt64

		err := rows.Scan(&log.ID, &log.TaskID, &log.TaskName, &log.ExecutionContext,
			&flowExecutionID, &stepID, &stepOrder, &log.Status, &log.ExecutionType,
			&log.StartTime, &endTime, &log.LogContent, &log.CreatedAt)
		if err != nil {
			continue
		}

		// 处理可空字段
		if endTime.Valid {
			log.EndTime = &endTime.Time
			log.Duration = lc.calculateDuration(log.StartTime, endTime.Time)
		}
		if flowExecutionID.Valid {
			log.FlowExecutionID = &[]int{int(flowExecutionID.Int64)}[0]
		}
		if stepID.Valid {
			log.StepID = &[]int{int(stepID.Int64)}[0]
		}
		if stepOrder.Valid {
			log.StepOrder = &[]int{int(stepOrder.Int64)}[0]
		}

		logs = append(logs, log)
	}

	response := models.TaskLogListResponse{
		Logs:       logs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetTaskLogDetail 获取任务执行详情
func (lc *LogController) GetTaskLogDetail(c *gin.Context) {
	logID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的日志ID",
		})
		return
	}

	// 查询任务执行详情
	query := `
		SELECT tl.id, tl.task_id, t.name as task_name, tl.execution_context,
		       tl.flow_execution_id, tl.step_id, tl.step_order,
		       tl.status, tl.execution_type, tl.start_time, tl.end_time, tl.log, tl.created_at
		FROM task_logs tl
		LEFT JOIN tasks t ON tl.task_id = t.id
		WHERE tl.id = ?
	`

	var log models.TaskExecutionLog
	var endTime sql.NullTime
	var flowExecutionID, stepID, stepOrder sql.NullInt64

	err = lc.db.QueryRow(query, logID).Scan(
		&log.ID, &log.TaskID, &log.TaskName, &log.ExecutionContext,
		&flowExecutionID, &stepID, &stepOrder, &log.Status, &log.ExecutionType,
		&log.StartTime, &endTime, &log.LogContent, &log.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "日志记录不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "查询日志详情失败: " + err.Error(),
			})
		}
		return
	}

	if endTime.Valid {
		log.EndTime = &endTime.Time
		log.Duration = lc.calculateDuration(log.StartTime, endTime.Time)
	}

	if flowExecutionID.Valid {
		log.FlowExecutionID = &[]int{int(flowExecutionID.Int64)}[0]
	}

	if stepID.Valid {
		log.StepID = &[]int{int(stepID.Int64)}[0]
	}

	if stepOrder.Valid {
		log.StepOrder = &[]int{int(stepOrder.Int64)}[0]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    log,
	})
}

// GetFlowLogs 获取任务流执行日志列表
func (lc *LogController) GetFlowLogs(c *gin.Context) {
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	flowName := c.Query("flow_name")
	executionType := c.Query("execution_type") // scheduled, manual
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if status != "" {
		whereClause += " AND tfe.status = ?"
		args = append(args, status)
	}

	if flowName != "" {
		whereClause += " AND tf.name LIKE ?"
		args = append(args, "%"+flowName+"%")
	}

	if executionType != "" {
		whereClause += " AND tfe.execution_type = ?"
		args = append(args, executionType)
	}

	if dateFrom != "" {
		whereClause += " AND DATE(tfe.start_time) >= ?"
		args = append(args, dateFrom)
	}

	if dateTo != "" {
		whereClause += " AND DATE(tfe.start_time) <= ?"
		args = append(args, dateTo)
	}

	// 查询总数
	countQuery := `
		SELECT COUNT(*) 
		FROM task_flow_executions tfe
		LEFT JOIN task_flows tf ON tfe.flow_id = tf.id
		` + whereClause

	var total int
	err := lc.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询任务流日志总数失败: " + err.Error(),
		})
		return
	}

	// 计算分页
	totalPages := (total + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize

	// 查询任务流执行日志列表
	query := `
		SELECT tfe.id, tfe.flow_id, tf.name as flow_name, tfe.status, 
		       tfe.execution_type, tfe.start_time, tfe.end_time, tfe.created_at
		FROM task_flow_executions tfe
		LEFT JOIN task_flows tf ON tfe.flow_id = tf.id
		` + whereClause + `
		ORDER BY tfe.start_time DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)
	rows, err := lc.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询任务流日志失败: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var logs []models.FlowExecutionLog
	for rows.Next() {
		var log models.FlowExecutionLog
		var endTime sql.NullTime

		err := rows.Scan(&log.ID, &log.FlowID, &log.FlowName, &log.Status,
			&log.ExecutionType, &log.StartTime, &endTime, &log.CreatedAt)
		if err != nil {
			continue
		}

		if endTime.Valid {
			log.EndTime = &endTime.Time
			log.Duration = lc.calculateDuration(log.StartTime, endTime.Time)
		}

		logs = append(logs, log)
	}

	response := models.FlowLogListResponse{
		Logs:       logs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetFlowLogDetail 获取任务流执行详情
func (lc *LogController) GetFlowLogDetail(c *gin.Context) {
	executionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的执行ID",
		})
		return
	}

	// 查询执行详情
	executionQuery := `
		SELECT tfe.id, tfe.flow_id, tf.name as flow_name, tfe.status, 
		       tfe.execution_type, tfe.start_time, tfe.end_time, tfe.created_at
		FROM task_flow_executions tfe
		LEFT JOIN task_flows tf ON tfe.flow_id = tf.id
		WHERE tfe.id = ?
	`

	var execution models.FlowExecutionLog
	var endTime sql.NullTime

	err = lc.db.QueryRow(executionQuery, executionID).Scan(
		&execution.ID, &execution.FlowID, &execution.FlowName, &execution.Status,
		&execution.ExecutionType, &execution.StartTime, &endTime, &execution.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "执行记录不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "查询执行详情失败: " + err.Error(),
			})
		}
		return
	}

	if endTime.Valid {
		execution.EndTime = &endTime.Time
		execution.Duration = lc.calculateDuration(execution.StartTime, endTime.Time)
	}

	// 查询步骤日志（从统一的task_logs表查询）
	stepsQuery := `
		SELECT tl.id, tl.task_id, t.name as task_name, tl.execution_context,
		       tl.flow_execution_id, tl.step_id, tl.step_order,
		       tl.status, tl.execution_type, tl.start_time, tl.end_time, tl.log, tl.created_at
		FROM task_logs tl
		LEFT JOIN tasks t ON tl.task_id = t.id
		WHERE tl.flow_execution_id = ? AND tl.execution_context = 'flow_step'
		ORDER BY tl.step_order
	`

	rows, err := lc.db.Query(stepsQuery, executionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询步骤日志失败: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var steps []models.TaskExecutionLog
	for rows.Next() {
		var step models.TaskExecutionLog
		var endTime sql.NullTime
		var flowExecutionID, stepID, stepOrder sql.NullInt64

		err := rows.Scan(&step.ID, &step.TaskID, &step.TaskName, &step.ExecutionContext,
			&flowExecutionID, &stepID, &stepOrder, &step.Status, &step.ExecutionType,
			&step.StartTime, &endTime, &step.LogContent, &step.CreatedAt)
		if err != nil {
			continue
		}

		if endTime.Valid {
			step.EndTime = &endTime.Time
			step.Duration = lc.calculateDuration(step.StartTime, endTime.Time)
		}

		if flowExecutionID.Valid {
			step.FlowExecutionID = &[]int{int(flowExecutionID.Int64)}[0]
		}

		if stepID.Valid {
			step.StepID = &[]int{int(stepID.Int64)}[0]
		}

		if stepOrder.Valid {
			step.StepOrder = &[]int{int(stepOrder.Int64)}[0]
		}

		steps = append(steps, step)
	}

	response := models.FlowLogDetailResponse{
		Steps:     steps,
		Execution: execution,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// calculateDuration 计算持续时间
func (lc *LogController) calculateDuration(start, end time.Time) string {
	duration := end.Sub(start)

	if duration.Hours() >= 1 {
		return fmt.Sprintf("%.1f小时", duration.Hours())
	} else if duration.Minutes() >= 1 {
		return fmt.Sprintf("%.0f分钟", duration.Minutes())
	} else {
		return fmt.Sprintf("%.0f秒", duration.Seconds())
	}
}
