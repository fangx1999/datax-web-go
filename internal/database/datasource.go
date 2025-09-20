package database

import (
	"database/sql"
	"errors"
	"fmt"

	"com.duole/datax-web-go/internal/entities"
)

// DataSourceDB 数据源数据库操作（空结构体）
type DataSourceDB struct{}

// List 获取数据源列表
func (d *DataSourceDB) List() ([]entities.DataSource, error) {
	query := `
		SELECT 
		    ds.id,
		    ds.name,
		    ds.type, 
		    COALESCE(uc.username, '系统') as created_by_name,
		    COALESCE(uu.username, '系统') as updated_by_name,
		    ds.created_at
		FROM data_sources ds
		LEFT JOIN users uc ON ds.created_by = uc.id
		LEFT JOIN users uu ON ds.updated_by = uu.id
		ORDER BY ds.id DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询数据源列表失败: %w", err)
	}
	defer rows.Close()

	var dataSources []entities.DataSource
	for rows.Next() {
		var ds entities.DataSource
		err := rows.Scan(&ds.ID, &ds.Name, &ds.Type, &ds.CreatedByName, &ds.UpdatedByName, &ds.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描数据源数据失败: %w", err)
		}
		dataSources = append(dataSources, ds)
	}

	return dataSources, nil
}

// GetByID 根据ID获取数据源
func (d *DataSourceDB) GetByID(id int) (*entities.DataSource, error) {
	query := `SELECT id,name,type,db_url,db_user,db_database,defaultfs,hadoopconfig FROM data_sources WHERE id=?`

	var ds entities.DataSource
	err := db.QueryRow(query, id).Scan(
		&ds.ID, &ds.Name, &ds.Type, &ds.DBURL, &ds.DBUser, &ds.DBDatabase, &ds.DefaultFS, &ds.HadoopConfig)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("数据源不存在")
		}
		return nil, fmt.Errorf("查询数据源失败: %w", err)
	}

	return &ds, nil
}

// GetByType 根据类型获取数据源
func (d *DataSourceDB) GetByType(dataType string) ([]entities.DataSource, error) {
	query := `SELECT id, name FROM data_sources WHERE type=? ORDER BY name`

	rows, err := db.Query(query, dataType)
	if err != nil {
		return nil, fmt.Errorf("查询数据源失败: %w", err)
	}
	defer rows.Close()

	var dataSources []entities.DataSource
	for rows.Next() {
		var ds entities.DataSource
		err := rows.Scan(&ds.ID, &ds.Name)
		if err != nil {
			return nil, fmt.Errorf("扫描数据源数据失败: %w", err)
		}
		dataSources = append(dataSources, ds)
	}

	return dataSources, nil
}

// Create 创建数据源
func (d *DataSourceDB) Create(ds *entities.DataSource) error {
	var query string
	var args []interface{}

	if ds.Type == "mysql" {
		query = `INSERT INTO data_sources(name,type,db_url,db_user,db_password,db_database,created_by,updated_by) VALUES(?,?,?,?,?,?,?,?)`
		args = []interface{}{ds.Name, ds.Type, ds.DBURL, ds.DBUser, ds.DBPassword, ds.DBDatabase, ds.CreatedBy, ds.UpdatedBy}
	} else {
		query = `INSERT INTO data_sources(name,type,defaultfs,hadoopconfig,created_by,updated_by) VALUES(?,?,?,?,?,?)`
		args = []interface{}{ds.Name, ds.Type, ds.DefaultFS, ds.HadoopConfig, ds.CreatedBy, ds.UpdatedBy}
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("创建数据源失败: %w", err)
	}

	return nil
}

// Update 更新数据源
func (d *DataSourceDB) Update(ds *entities.DataSource) error {
	var query string
	var args []interface{}

	if ds.Type == "mysql" {
		// 如果密码为空，则不更新密码字段
		if ds.DBPassword == nil || *ds.DBPassword == "" {
			query = `UPDATE data_sources SET name=?,db_url=?,db_user=?,db_database=?,updated_by=? WHERE id=?`
			args = []interface{}{ds.Name, ds.DBURL, ds.DBUser, ds.DBDatabase, ds.UpdatedBy, ds.ID}
		} else {
			query = `UPDATE data_sources SET name=?,db_url=?,db_user=?,db_password=?,db_database=?,updated_by=? WHERE id=?`
			args = []interface{}{ds.Name, ds.DBURL, ds.DBUser, ds.DBPassword, ds.DBDatabase, ds.UpdatedBy, ds.ID}
		}
	} else {
		query = `UPDATE data_sources SET name=?,defaultfs=?,hadoopconfig=?,updated_by=? WHERE id=?`
		args = []interface{}{ds.Name, ds.DefaultFS, ds.HadoopConfig, ds.UpdatedBy, ds.ID}
	}

	result, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新数据源失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("检查更新结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("数据源不存在")
	}

	return nil
}

// Delete 删除数据源
func (d *DataSourceDB) Delete(id int) error {
	query := `DELETE FROM data_sources WHERE id=?`

	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除数据源失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("检查删除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("数据源不存在")
	}

	return nil
}
