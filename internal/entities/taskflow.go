package entities

import (
	"time"
)

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
	ID             int    `json:"id"`
	StepOrder      int    `json:"step_order"`
	TimeoutMinutes *int   `json:"timeout_minutes,omitempty"`
	TaskName       string `json:"task_name"`
	TaskID         int    `json:"task_id"`
}
