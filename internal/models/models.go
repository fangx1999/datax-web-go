package models

import (
	"time"
)

// ========== 数据源模型 ==========

// DataSource 表示数据源记录
type DataSource struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	DBURL         *string   `json:"db_url,omitempty"`
	DBUser        *string   `json:"db_user,omitempty"`
	DBPassword    *string   `json:"db_password,omitempty"`
	DBDatabase    *string   `json:"db_database,omitempty"`
	DefaultFS     *string   `json:"defaultfs,omitempty"`
	HadoopConfig  *string   `json:"hadoopconfig,omitempty"`
	CreatedBy     *int      `json:"created_by,omitempty"`
	UpdatedBy     *int      `json:"updated_by,omitempty"`
	CreatedByName *string   `json:"created_by_name,omitempty"`
	UpdatedByName *string   `json:"updated_by_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ========== 任务模型 ==========

// Task 表示任务记录
type Task struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	SourceID   int    `json:"source_id"`
	TargetID   int    `json:"target_id"`
	JsonConfig string `json:"json_config"`
	// Additional fields for display
	Source        string    `json:"source,omitempty"`
	Target        string    `json:"target,omitempty"`
	FlowName      string    `json:"flow_name,omitempty"`
	FlowID        int       `json:"flow_id,omitempty"`
	CreatedBy     *int      `json:"created_by,omitempty"`
	UpdatedBy     *int      `json:"updated_by,omitempty"`
	CreatedByName *string   `json:"created_by_name,omitempty"`
	UpdatedByName *string   `json:"updated_by_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// LogItem 表示日志条目
type LogItem struct {
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
	Status  string    `json:"status"`
	Content string    `json:"content"`
}

// TaskFlowSelection 表示用于选择的任务流（简化版本）
type TaskFlowSelection struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ========== User Models ==========

// User 表示用户记录
type User struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	Role          string    `json:"role"`
	Disabled      bool      `json:"disabled"`
	CreatedBy     *int      `json:"created_by,omitempty"`
	UpdatedBy     *int      `json:"updated_by,omitempty"`
	CreatedByName *string   `json:"created_by_name,omitempty"`
	UpdatedByName *string   `json:"updated_by_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ========== TaskFlow Models ==========

// TaskFlow 表示包含所有详情的任务流
type TaskFlow struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	CronExpr      string    `json:"cron_expr"`
	Enabled       bool      `json:"enabled"`
	CreatedBy     *int      `json:"created_by,omitempty"`
	UpdatedBy     *int      `json:"updated_by,omitempty"`
	CreatedByName *string   `json:"created_by_name,omitempty"`
	UpdatedByName *string   `json:"updated_by_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TaskFlowStep 表示任务流中的步骤
type TaskFlowStep struct {
	ID             int       `json:"id"`
	StepOrder      int       `json:"step_order"`
	TimeoutMinutes *int      `json:"timeout_minutes,omitempty"`
	TaskName       string    `json:"task_name"`
	TaskID         int       `json:"task_id"`
	CreatedBy      *int      `json:"created_by,omitempty"`
	UpdatedBy      *int      `json:"updated_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ========== Log Models ==========

// FlowExecutionLog 表示任务流执行日志
type FlowExecutionLog struct {
	ID            int        `json:"id"`
	FlowID        int        `json:"flow_id"`
	FlowName      string     `json:"flow_name"`
	Status        string     `json:"status"`
	ExecutionType string     `json:"execution_type"` // scheduled, manual
	StartTime     time.Time  `json:"start_time"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Duration      string     `json:"duration,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TaskExecutionLog 表示统一的任务执行日志（支持独立任务和任务流步骤）
type TaskExecutionLog struct {
	ID               int        `json:"id"`
	TaskID           int        `json:"task_id"`
	TaskName         string     `json:"task_name"`
	ExecutionContext string     `json:"execution_context"` // 'standalone' or 'flow_step'
	FlowExecutionID  *int       `json:"flow_execution_id,omitempty"`
	StepID           *int       `json:"step_id,omitempty"`
	StepOrder        *int       `json:"step_order,omitempty"`
	Status           string     `json:"status"`
	ExecutionType    string     `json:"execution_type"` // scheduled, manual
	StartTime        time.Time  `json:"start_time"`
	EndTime          *time.Time `json:"end_time,omitempty"`
	Duration         string     `json:"duration,omitempty"`
	LogContent       string     `json:"log_content"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// FlowLogListResponse 表示流程日志列表响应
type FlowLogListResponse struct {
	Logs       []FlowExecutionLog `json:"logs"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// TaskLogListResponse 表示任务日志列表响应
type TaskLogListResponse struct {
	Logs       []TaskExecutionLog `json:"logs"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// FlowLogDetailResponse 表示流程日志详情响应
type FlowLogDetailResponse struct {
	Steps     []TaskExecutionLog `json:"steps"`
	Execution FlowExecutionLog   `json:"execution"`
}
