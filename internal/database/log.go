package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"com.duole/datax-web-go/internal/entities"
)

// 统一的日志错误定义，方便调用方通过 errors.Is 判断
var (
	ErrTaskLogNotFound = errors.New("任务日志不存在")
	ErrFlowLogNotFound = errors.New("流程日志不存在")
)

// TaskLogFilters 用于构建任务日志查询条件
type TaskLogFilters struct {
	Status        string
	ExecutionType string
	TaskName      string
	DateFrom      *time.Time
	DateTo        *time.Time
}

// FlowLogFilters 用于构建任务流日志查询条件
type FlowLogFilters struct {
	Status        string
	ExecutionType string
	FlowName      string
	DateFrom      *time.Time
	DateTo        *time.Time
}

// LogDB 日志数据库操作（空结构体）
type LogDB struct{}

// GetTaskLogs 获取任务日志列表
func (d *LogDB) GetTaskLogs(page, pageSize int, filters TaskLogFilters) (*entities.TaskLogListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	conditions := []string{"1=1"}
	args := make([]interface{}, 0)

	if filters.Status != "" {
		conditions = append(conditions, "tl.status = ?")
		args = append(args, filters.Status)
	}
	if filters.ExecutionType != "" {
		conditions = append(conditions, "tl.execution_type = ?")
		args = append(args, filters.ExecutionType)
	}
	if filters.TaskName != "" {
		conditions = append(conditions, "t.name LIKE ?")
		args = append(args, "%"+filters.TaskName+"%")
	}
	if filters.DateFrom != nil {
		conditions = append(conditions, "tl.start_time >= ?")
		args = append(args, filters.DateFrom.UTC())
	}
	if filters.DateTo != nil {
		conditions = append(conditions, "tl.start_time <= ?")
		args = append(args, filters.DateTo.UTC())
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`
                SELECT COUNT(*)
                FROM task_logs tl
                LEFT JOIN tasks t ON tl.task_id = t.id
                WHERE %s
        `, whereClause)

	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("查询任务日志总数失败: %w", err)
	}

	offset := (page - 1) * pageSize
	queryArgs := append(append([]interface{}{}, args...), pageSize, offset)

	query := fmt.Sprintf(`
                SELECT
                        tl.id,
                        tl.task_id,
                        COALESCE(t.name, ''),
                        tl.flow_execution_id,
                        tl.step_id,
                        tl.step_order,
                        tl.status,
                        tl.execution_type,
                        tl.start_time,
                        tl.end_time,
                        tl.log,
                        tl.created_at
                FROM task_logs tl
                LEFT JOIN tasks t ON tl.task_id = t.id
                WHERE %s
                ORDER BY tl.start_time DESC
                LIMIT ? OFFSET ?
        `, whereClause)

	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("查询任务日志失败: %w", err)
	}
	defer rows.Close()

	logs := make([]entities.TaskExecutionLog, 0)
	for rows.Next() {
		var (
			logEntry   entities.TaskExecutionLog
			flowExecID sql.NullInt64
			stepID     sql.NullInt64
			stepOrder  sql.NullInt64
			endTime    sql.NullTime
			logContent string
			createdAt  time.Time
		)

		if err := rows.Scan(
			&logEntry.ID,
			&logEntry.TaskID,
			&logEntry.TaskName,
			&flowExecID,
			&stepID,
			&stepOrder,
			&logEntry.Status,
			&logEntry.ExecutionType,
			&logEntry.StartTime,
			&endTime,
			&logContent,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("扫描任务日志数据失败: %w", err)
		}

		if flowExecID.Valid {
			v := int(flowExecID.Int64)
			logEntry.FlowExecutionID = &v
		}
		if stepID.Valid {
			v := int(stepID.Int64)
			logEntry.StepID = &v
		}
		if stepOrder.Valid {
			v := int(stepOrder.Int64)
			logEntry.StepOrder = &v
		}
		if endTime.Valid {
			end := endTime.Time
			logEntry.EndTime = &end
			logEntry.Duration = formatDuration(logEntry.StartTime, &end)
		} else {
			logEntry.Duration = ""
		}

		logEntry.Message = truncateLog(logContent)
		logEntry.LogContent = logContent
		logEntry.CreatedAt = createdAt

		logs = append(logs, logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历任务日志失败: %w", err)
	}

	return &entities.TaskLogListResponse{
		Logs:       logs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: calcTotalPages(total, pageSize),
	}, nil
}

// GetFlowLogs 获取流程日志列表
func (d *LogDB) GetFlowLogs(page, pageSize int, filters FlowLogFilters) (*entities.FlowLogListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	conditions := []string{"1=1"}
	args := make([]interface{}, 0)

	if filters.Status != "" {
		conditions = append(conditions, "tfe.status = ?")
		args = append(args, filters.Status)
	}
	if filters.ExecutionType != "" {
		conditions = append(conditions, "tfe.execution_type = ?")
		args = append(args, filters.ExecutionType)
	}
	if filters.FlowName != "" {
		conditions = append(conditions, "tf.name LIKE ?")
		args = append(args, "%"+filters.FlowName+"%")
	}
	if filters.DateFrom != nil {
		conditions = append(conditions, "tfe.start_time >= ?")
		args = append(args, filters.DateFrom.UTC())
	}
	if filters.DateTo != nil {
		conditions = append(conditions, "tfe.start_time <= ?")
		args = append(args, filters.DateTo.UTC())
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`
                SELECT COUNT(*)
                FROM task_flow_executions tfe
                LEFT JOIN task_flows tf ON tfe.flow_id = tf.id
                WHERE %s
        `, whereClause)

	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("查询任务流日志总数失败: %w", err)
	}

	offset := (page - 1) * pageSize
	queryArgs := append(append([]interface{}{}, args...), pageSize, offset)

	query := fmt.Sprintf(`
                SELECT
                        tfe.id,
                        tfe.flow_id,
                        COALESCE(tf.name, ''),
                        tfe.status,
                        tfe.execution_type,
                        tfe.start_time,
                        tfe.end_time,
                        tfe.created_at
                FROM task_flow_executions tfe
                LEFT JOIN task_flows tf ON tfe.flow_id = tf.id
                WHERE %s
                ORDER BY tfe.start_time DESC
                LIMIT ? OFFSET ?
        `, whereClause)

	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("查询任务流日志失败: %w", err)
	}
	defer rows.Close()

	logs := make([]entities.FlowExecutionLog, 0)
	for rows.Next() {
		var (
			logEntry  entities.FlowExecutionLog
			endTime   sql.NullTime
			createdAt time.Time
		)

		if err := rows.Scan(
			&logEntry.ID,
			&logEntry.FlowID,
			&logEntry.FlowName,
			&logEntry.Status,
			&logEntry.ExecutionType,
			&logEntry.StartTime,
			&endTime,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("扫描任务流日志失败: %w", err)
		}

		if endTime.Valid {
			end := endTime.Time
			logEntry.EndTime = &end
			logEntry.Duration = formatDuration(logEntry.StartTime, &end)
		}

		logEntry.CreatedAt = createdAt
		logs = append(logs, logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历任务流日志失败: %w", err)
	}

	return &entities.FlowLogListResponse{
		Logs:       logs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: calcTotalPages(total, pageSize),
	}, nil
}

// GetTaskLogDetail 获取任务日志详情
func (d *LogDB) GetTaskLogDetail(id int) (*entities.TaskExecutionLog, error) {
	query := `
                SELECT
                        tl.id,
                        tl.task_id,
                        COALESCE(t.name, ''),
                        tl.flow_execution_id,
                        tl.step_id,
                        tl.step_order,
                        tl.status,
                        tl.execution_type,
                        tl.start_time,
                        tl.end_time,
                        tl.log,
                        tl.created_at
                FROM task_logs tl
                LEFT JOIN tasks t ON tl.task_id = t.id
                WHERE tl.id = ?
        `

	var (
		logEntry   entities.TaskExecutionLog
		flowExecID sql.NullInt64
		stepID     sql.NullInt64
		stepOrder  sql.NullInt64
		endTime    sql.NullTime
		content    string
		createdAt  time.Time
	)

	err := db.QueryRow(query, id).Scan(
		&logEntry.ID,
		&logEntry.TaskID,
		&logEntry.TaskName,
		&flowExecID,
		&stepID,
		&stepOrder,
		&logEntry.Status,
		&logEntry.ExecutionType,
		&logEntry.StartTime,
		&endTime,
		&content,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskLogNotFound
		}
		return nil, fmt.Errorf("查询任务日志详情失败: %w", err)
	}

	if flowExecID.Valid {
		v := int(flowExecID.Int64)
		logEntry.FlowExecutionID = &v
	}
	if stepID.Valid {
		v := int(stepID.Int64)
		logEntry.StepID = &v
	}
	if stepOrder.Valid {
		v := int(stepOrder.Int64)
		logEntry.StepOrder = &v
	}
	if endTime.Valid {
		end := endTime.Time
		logEntry.EndTime = &end
		logEntry.Duration = formatDuration(logEntry.StartTime, &end)
	}

	logEntry.LogContent = content
	logEntry.Message = truncateLog(content)
	logEntry.CreatedAt = createdAt

	return &logEntry, nil
}

// GetFlowLogDetail 获取流程日志详情
func (d *LogDB) GetFlowLogDetail(id int) (*entities.FlowLogDetailResponse, error) {
	query := `
                SELECT
                        tfe.id,
                        tfe.flow_id,
                        COALESCE(tf.name, ''),
                        tfe.status,
                        tfe.execution_type,
                        tfe.start_time,
                        tfe.end_time,
                        tfe.created_at
                FROM task_flow_executions tfe
                LEFT JOIN task_flows tf ON tfe.flow_id = tf.id
                WHERE tfe.id = ?
        `

	var (
		logEntry  entities.FlowExecutionLog
		endTime   sql.NullTime
		createdAt time.Time
	)

	err := db.QueryRow(query, id).Scan(
		&logEntry.ID,
		&logEntry.FlowID,
		&logEntry.FlowName,
		&logEntry.Status,
		&logEntry.ExecutionType,
		&logEntry.StartTime,
		&endTime,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFlowLogNotFound
		}
		return nil, fmt.Errorf("查询流程日志详情失败: %w", err)
	}

	if endTime.Valid {
		end := endTime.Time
		logEntry.EndTime = &end
		logEntry.Duration = formatDuration(logEntry.StartTime, &end)
	}
	logEntry.CreatedAt = createdAt

	// 汇总子任务日志内容
	builder := &strings.Builder{}
	rows, err := db.Query(`
                SELECT COALESCE(t.name, ''), tl.status, tl.start_time, tl.end_time, tl.log
                FROM task_logs tl
                LEFT JOIN tasks t ON tl.task_id = t.id
                WHERE tl.flow_execution_id = ?
                ORDER BY tl.start_time
        `, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var (
				taskName sql.NullString
				status   string
				start    time.Time
				end      sql.NullTime
				content  string
			)

			if scanErr := rows.Scan(&taskName, &status, &start, &end, &content); scanErr != nil {
				err = fmt.Errorf("扫描子任务日志失败: %w", scanErr)
				break
			}

			fmt.Fprintf(builder, "任务: %s\n", taskName.String)
			fmt.Fprintf(builder, "状态: %s\n", status)
			fmt.Fprintf(builder, "开始时间: %s\n", start.Format(time.RFC3339))
			if end.Valid {
				fmt.Fprintf(builder, "结束时间: %s\n", end.Time.Format(time.RFC3339))
			}
			builder.WriteString("日志内容:\n")
			builder.WriteString(content)
			builder.WriteString("\n\n")
		}

		if err == nil {
			if rowsErr := rows.Err(); rowsErr != nil {
				err = fmt.Errorf("遍历子任务日志失败: %w", rowsErr)
			}
		}
	}

	if err != nil {
		return nil, err
	}

	logEntry.LogContent = builder.String()

	return &entities.FlowLogDetailResponse{Log: logEntry}, nil
}

func truncateLog(content string) string {
	const maxLen = 200
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

func formatDuration(start time.Time, end *time.Time) string {
	if end == nil {
		return ""
	}
	duration := end.Sub(start)
	if duration < 0 {
		duration = 0
	}
	return duration.Truncate(time.Second).String()
}

func calcTotalPages(total, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}
