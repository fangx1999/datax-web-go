package database

import (
	"database/sql"
	"errors"
	"fmt"

	"com.duole/datax-web-go/internal/entities"
)

// TaskDB 任务数据库操作（空结构体）
type TaskDB struct{}

// List 获取任务列表
func (d *TaskDB) List() ([]entities.Task, error) {
	query := `
		SELECT 
		    t.id,
		    t.name,
		    tf.name,
		    tf.id, 
		    COALESCE(uc.username, '系统') as created_by_name,
		    COALESCE(uu.username, '系统') as updated_by_name,
		    t.created_at,
		    t.updated_at
		FROM tasks t 
		LEFT JOIN task_flow_steps tfs ON t.id=tfs.task_id 
		LEFT JOIN task_flows tf ON tfs.flow_id=tf.id
		LEFT JOIN users uc ON t.created_by = uc.id
		LEFT JOIN users uu ON t.updated_by = uu.id
		ORDER BY t.id DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询任务列表失败: %w", err)
	}
	defer rows.Close()

	var tasks []entities.Task
	for rows.Next() {
		var task entities.Task
		err := rows.Scan(&task.ID, &task.Name, &task.FlowName, &task.FlowID, &task.CreatedByName,
			&task.UpdatedByName, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描任务数据失败: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetByID 根据ID获取任务
func (d *TaskDB) GetByID(id int) (*entities.Task, error) {
	query := `
			SELECT 
    				t.id, t.name, t.source_id, t.target_id, COALESCE(t.json_config,''), t.created_at, t.updated_at,
					(SELECT name FROM data_sources WHERE id=t.source_id),
					(SELECT name FROM data_sources WHERE id=t.target_id) 
			FROM tasks t WHERE t.id=?`

	var task entities.Task
	err := db.QueryRow(query, id).Scan(&task.ID, &task.Name, &task.SourceID, &task.TargetID, &task.JsonConfig,
		&task.CreatedAt, &task.UpdatedAt, &task.Source, &task.Target)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	return &task, nil
}

// Create 创建任务
func (d *TaskDB) Create(task *entities.Task) error {
	query := `INSERT INTO tasks(name,source_id,target_id,json_config,created_by,updated_by) VALUES(?,?,?,?,?,?)`

	result, err := db.Exec(query, task.Name, task.SourceID, task.TargetID, task.JsonConfig, task.CreatedBy, task.UpdatedBy)
	if err != nil {
		return fmt.Errorf("创建任务失败: %w", err)
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取任务ID失败: %w", err)
	}
	task.ID = int(taskID)

	return nil
}

// Update 更新任务
func (d *TaskDB) Update(task *entities.Task) error {
	query := `UPDATE tasks SET json_config=?, updated_by=? WHERE id=?`

	result, err := db.Exec(query, task.JsonConfig, task.UpdatedBy, task.ID)
	if err != nil {
		return fmt.Errorf("更新任务失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("检查更新结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("任务不存在")
	}

	return nil
}

// Delete 删除任务
func (d *TaskDB) Delete(id int) error {
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 先删除任务流步骤关联
	_, err = tx.Exec("DELETE FROM task_flow_steps WHERE task_id=?", id)
	if err != nil {
		return fmt.Errorf("删除任务流步骤关联失败: %w", err)
	}

	// 再删除任务
	result, err := tx.Exec("DELETE FROM tasks WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("检查删除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("任务不存在")
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// GetAvailableTasks 获取可用任务列表（未分配给任务流的任务）
func (d *TaskDB) GetAvailableTasks() ([]entities.Task, error) {
	query := `
		SELECT t.id, t.name, t.source_id, t.target_id, t.json_config,
		       ds.name as source, dt.name as target,
		       t.created_at, t.updated_at
		FROM tasks t
		LEFT JOIN data_sources ds ON t.source_id = ds.id
		LEFT JOIN data_sources dt ON t.target_id = dt.id
		LEFT JOIN task_flow_steps tfs ON t.id = tfs.task_id
		WHERE tfs.task_id IS NULL
		ORDER BY t.id DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询可用任务失败: %w", err)
	}
	defer rows.Close()

	var tasks []entities.Task
	for rows.Next() {
		var task entities.Task
		err := rows.Scan(&task.ID, &task.Name, &task.SourceID, &task.TargetID, &task.JsonConfig,
			&task.Source, &task.Target, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描任务失败: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
