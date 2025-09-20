package database

import (
	"database/sql"
	"fmt"

	"com.duole/datax-web-go/internal/entities"
)

// TaskFlowDB 任务流数据库操作（空结构体）
type TaskFlowDB struct{}

// List 获取任务流列表
func (d *TaskFlowDB) List() ([]entities.TaskFlow, error) {
	query := `
		SELECT tf.id, tf.name, tf.description, tf.cron_expr, tf.enabled,
		       COALESCE(uc.username, '系统') as created_by_name,
		       COALESCE(uu.username, '系统') as updated_by_name,
		       tf.created_at
		FROM task_flows tf
		LEFT JOIN users uc ON tf.created_by = uc.id
		LEFT JOIN users uu ON tf.updated_by = uu.id
		ORDER BY tf.id DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询任务流列表失败: %w", err)
	}
	defer rows.Close()

	var flows []entities.TaskFlow
	for rows.Next() {
		var flow entities.TaskFlow
		err := rows.Scan(
			&flow.ID, &flow.Name, &flow.Description, &flow.CronExpr, &flow.Enabled,
			&flow.CreatedByName, &flow.UpdatedByName,
			&flow.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描任务流数据失败: %w", err)
		}
		flows = append(flows, flow)
	}

	return flows, nil
}

// ListEnabled 获取启用的任务流列表
func (d *TaskFlowDB) ListEnabled() ([]entities.TaskFlow, error) {
	query := `SELECT id, name FROM task_flows WHERE enabled = 1 ORDER BY name`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询启用的任务流列表失败: %w", err)
	}
	defer rows.Close()

	var flows []entities.TaskFlow
	for rows.Next() {
		var flow entities.TaskFlow
		err := rows.Scan(&flow.ID, &flow.Name)
		if err != nil {
			return nil, fmt.Errorf("扫描任务流数据失败: %w", err)
		}
		flows = append(flows, flow)
	}

	return flows, nil
}

// GetByID 根据ID获取任务流
func (d *TaskFlowDB) GetByID(id int) (*entities.TaskFlow, error) {
	query := `SELECT id,name,description,cron_expr,enabled,created_at,updated_at FROM task_flows WHERE id=?`

	var flow entities.TaskFlow
	err := db.QueryRow(query, id).Scan(&flow.ID, &flow.Name, &flow.Description, &flow.CronExpr, &flow.Enabled, &flow.CreatedAt, &flow.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("任务流不存在")
		}
		return nil, fmt.Errorf("查询任务流失败: %w", err)
	}

	return &flow, nil
}

// Create 创建任务流
func (d *TaskFlowDB) Create(flow *entities.TaskFlow) error {
	query := `INSERT INTO task_flows(name,description,cron_expr,enabled,created_by,updated_by) VALUES(?,?,?,?,?,?)`

	result, err := db.Exec(query, flow.Name, flow.Description, flow.CronExpr, flow.Enabled, flow.CreatedBy, flow.UpdatedBy)
	if err != nil {
		return fmt.Errorf("创建任务流失败: %w", err)
	}

	if flowID, err := result.LastInsertId(); err == nil {
		flow.ID = int(flowID)
	}

	return nil
}

// Update 更新任务流
func (d *TaskFlowDB) Update(flow *entities.TaskFlow) error {
	query := `UPDATE task_flows SET name=?,description=?,cron_expr=?,enabled=?,updated_by=? WHERE id=?`

	result, err := db.Exec(query, flow.Name, flow.Description, flow.CronExpr, flow.Enabled, flow.UpdatedBy, flow.ID)
	if err != nil {
		return fmt.Errorf("更新任务流失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("检查更新结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("任务流不存在")
	}

	return nil
}

// Delete 删除任务流
func (d *TaskFlowDB) Delete(id int) error {
	query := `DELETE FROM task_flows WHERE id=?`

	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除任务流失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("检查删除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("任务流不存在")
	}

	return nil
}

// GetSteps 获取任务流步骤
func (d *TaskFlowDB) GetSteps(flowID int) ([]entities.TaskFlowStep, error) {
	query := `
		SELECT tfs.id, tfs.task_id, tfs.step_order,
		       t.name as task_name
		FROM task_flow_steps tfs
		LEFT JOIN tasks t ON tfs.task_id = t.id
		WHERE tfs.flow_id = ?
		ORDER BY tfs.step_order
	`

	rows, err := db.Query(query, flowID)
	if err != nil {
		return nil, fmt.Errorf("查询任务流步骤失败: %w", err)
	}
	defer rows.Close()

	var steps []entities.TaskFlowStep
	for rows.Next() {
		var step entities.TaskFlowStep
		err := rows.Scan(&step.ID, &step.TaskID, &step.StepOrder, &step.TaskName)
		if err != nil {
			return nil, fmt.Errorf("扫描任务流步骤失败: %w", err)
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// AddStep 添加步骤到任务流
func (d *TaskFlowDB) AddStep(flowID, taskID int) error {
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 获取下一个步骤顺序（在事务中加锁）
	var maxOrder int
	err = tx.QueryRow("SELECT COALESCE(MAX(step_order), 0) FROM task_flow_steps WHERE flow_id=? FOR UPDATE", flowID).Scan(&maxOrder)
	if err != nil {
		return fmt.Errorf("获取步骤顺序失败: %w", err)
	}

	// 插入新步骤
	query := `INSERT INTO task_flow_steps(flow_id, task_id, step_order) VALUES (?, ?, ?)`
	_, err = tx.Exec(query, flowID, taskID, maxOrder+1)
	if err != nil {
		return fmt.Errorf("添加步骤失败: %w", err)
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// RemoveStep 从任务流中移除步骤
func (d *TaskFlowDB) RemoveStep(flowID, stepID int) error {
	query := `DELETE FROM task_flow_steps WHERE flow_id=? AND id=?`

	result, err := db.Exec(query, flowID, stepID)
	if err != nil {
		return fmt.Errorf("移除步骤失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("检查移除结果失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("步骤不存在")
	}

	return nil
}

// ReorderSteps 重新排序步骤
func (d *TaskFlowDB) ReorderSteps(flowID int, stepOrders []int) error {
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 更新每个步骤的顺序
	for i, stepID := range stepOrders {
		query := `UPDATE task_flow_steps SET step_order=? WHERE flow_id=? AND id=?`
		_, err = tx.Exec(query, i+1, flowID, stepID)
		if err != nil {
			return fmt.Errorf("更新步骤顺序失败: %w", err)
		}
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}
