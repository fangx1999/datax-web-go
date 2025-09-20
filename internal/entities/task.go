package entities

import (
	"time"
)

// Task 表示任务记录
type Task struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	SourceID   int    `json:"source_id"`
	TargetID   int    `json:"target_id"`
	JsonConfig string `json:"datax_json"`
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
