package entities

import (
	"time"
)

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

// IsMySQL 检查是否为MySQL数据源
func (ds *DataSource) IsMySQL() bool {
	return ds.Type == "mysql"
}
