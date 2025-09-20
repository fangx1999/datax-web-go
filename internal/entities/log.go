package entities

import (
	"time"
)

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
	Message       string     `json:"message,omitempty"`
	LogContent    string     `json:"log_content,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TaskExecutionLog 表示统一的任务执行日志（支持独立任务和任务流步骤）
type TaskExecutionLog struct {
	ID              int        `json:"id"`
	TaskID          int        `json:"task_id"`
	TaskName        string     `json:"task_name"`
	FlowExecutionID *int       `json:"flow_execution_id,omitempty"`
	StepID          *int       `json:"step_id,omitempty"`
	StepOrder       *int       `json:"step_order,omitempty"`
	Status          string     `json:"status"`
	ExecutionType   string     `json:"execution_type"` // scheduled, manual
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Duration        string     `json:"duration,omitempty"`
	Message         string     `json:"message,omitempty"`
	LogContent      string     `json:"log_content"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
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
	Log FlowExecutionLog `json:"log"`
}
