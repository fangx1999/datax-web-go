package entities

import (
	"time"
)

// User 表示用户记录
type User struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	Password      string    `json:"-"` // 密码不序列化到JSON
	Role          string    `json:"role"`
	Disabled      bool      `json:"disabled"`
	CreatedBy     *int      `json:"created_by,omitempty"`
	UpdatedBy     *int      `json:"updated_by,omitempty"`
	CreatedByName *string   `json:"created_by_name,omitempty"`
	UpdatedByName *string   `json:"updated_by_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
