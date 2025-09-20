package database

import (
	"database/sql"
	"fmt"

	"com.duole/datax-web-go/internal/entities"
)

// LogDB 日志数据库操作（空结构体）
type LogDB struct{}

// GetTaskLogs 获取任务日志列表
func (d *LogDB) GetTaskLogs(page, pageSize int, taskID, status string) (*entities.TaskLogListResponse, error) {
	// 构建查询条件
	whereClause := "1=1"
	args := []interface{}{}

	if taskID != "" {
		whereClause += " AND task_id = ?"
		args = append(args, taskID)
	}
	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM task_execution_logs WHERE %s", whereClause)
	var total int
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("查询任务日志总数失败: %w", err)
	}

	// 查询数据
	query := fmt.Sprintf(`
		SELECT tel.id, tel.task_id, t.name as task_name, tel.status, tel.start_time, tel.end_time, tel.message
		FROM task_execution_logs tel
		LEFT JOIN tasks t ON tel.task_id = t.id
		WHERE %s
		ORDER BY tel.start_time DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询任务日志失败: %w", err)
	}
	defer rows.Close()

	var logs []entities.TaskExecutionLog
	for rows.Next() {
		var log entities.TaskExecutionLog
		err := rows.Scan(
			&log.ID, &log.TaskID, &log.TaskName, &log.Status,
			&log.StartTime, &log.EndTime, &log.Message)
		if err != nil {
			return nil, fmt.Errorf("扫描任务日志数据失败: %w", err)
		}
		logs = append(logs, log)
	}

	return &entities.TaskLogListResponse{
		Logs:      logs,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: (total + pageSize - 1) / pageSize,
	}, nil
}

// GetFlowLogs 获取流程日志列表
func (d *LogDB) GetFlowLogs(page, pageSize int, flowID, status string) (*entities.FlowLogListResponse, error) {
	// 构建查询条件
	whereClause := "1=1"
	args := []interface{}{}

	if flowID != "" {
		whereClause += " AND flow_id = ?"
		args = append(args, flowID)
	}
	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM flow_execution_logs WHERE %s", whereClause)
	var total int
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("查询流程日志总数失败: %w", err)
	}

	// 查询数据
	query := fmt.Sprintf(`
		SELECT fel.id, fel.flow_id, tf.name as flow_name, fel.status, fel.start_time, fel.end_time, fel.message
		FROM flow_execution_logs fel
		LEFT JOIN task_flows tf ON fel.flow_id = tf.id
		WHERE %s
		ORDER BY fel.start_time DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询流程日志失败: %w", err)
	}
	defer rows.Close()

	var logs []entities.FlowExecutionLog
	for rows.Next() {
		var log entities.FlowExecutionLog
		err := rows.Scan(
			&log.ID, &log.FlowID, &log.FlowName, &log.Status,
			&log.StartTime, &log.EndTime, &log.Message)
		if err != nil {
			return nil, fmt.Errorf("扫描流程日志数据失败: %w", err)
		}
		logs = append(logs, log)
	}

	return &entities.FlowLogListResponse{
		Logs:      logs,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: (total + pageSize - 1) / pageSize,
	}, nil
}

// GetTaskLogDetail 获取任务日志详情
func (d *LogDB) GetTaskLogDetail(id int) (*entities.TaskExecutionLog, error) {
	query := `
		SELECT tel.id, tel.task_id, t.name as task_name, tel.status, 
		       tel.start_time, tel.end_time, tel.message, tel.log_content
		FROM task_execution_logs tel
		LEFT JOIN tasks t ON tel.task_id = t.id
		WHERE tel.id = ?
	`

	var log entities.TaskExecutionLog
	err := db.QueryRow(query, id).Scan(
		&log.ID, &log.TaskID, &log.TaskName, &log.Status,
		&log.StartTime, &log.EndTime, &log.Message, &log.LogContent)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("任务日志不存在")
		}
		return nil, fmt.Errorf("查询任务日志详情失败: %w", err)
	}

	return &log, nil
}

// GetFlowLogDetail 获取流程日志详情
func (d *LogDB) GetFlowLogDetail(id int) (*entities.FlowLogDetailResponse, error) {
	query := `
		SELECT fel.id, fel.flow_id, tf.name as flow_name, fel.status,
		       fel.start_time, fel.end_time, fel.message, fel.log_content
		FROM flow_execution_logs fel
		LEFT JOIN task_flows tf ON fel.flow_id = tf.id
		WHERE fel.id = ?
	`

	var log entities.FlowExecutionLog
	err := db.QueryRow(query, id).Scan(
		&log.ID, &log.FlowID, &log.FlowName, &log.Status,
		&log.StartTime, &log.EndTime, &log.Message, &log.LogContent)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("流程日志不存在")
		}
		return nil, fmt.Errorf("查询流程日志详情失败: %w", err)
	}

	return &entities.FlowLogDetailResponse{
		Log: log,
	}, nil
}
