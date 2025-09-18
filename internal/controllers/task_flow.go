package controllers

import (
	"com.duole/datax-web-go/internal/models"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"strconv"
	"strings"
)

// ========== 任务流处理器 ==========

// TaskFlowList 显示所有任务流，包含运行、切换、编辑和删除操作
func (ct *Controller) TaskFlowList(c *gin.Context) {
	rows, _ := ct.db.Query(`
		SELECT tf.id, tf.name, tf.description, tf.cron_expr, tf.enabled,
		       COALESCE(uc.username, '系统') as created_by_name,
		       COALESCE(uu.username, '系统') as updated_by_name,
		       tf.created_at
		FROM task_flows tf
		LEFT JOIN users uc ON tf.created_by = uc.id
		LEFT JOIN users uu ON tf.updated_by = uu.id
		ORDER BY tf.id DESC
	`)
	defer rows.Close()

	var flows []models.TaskFlow
	for rows.Next() {
		var r models.TaskFlow
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CronExpr, &r.Enabled, &r.CreatedByName, &r.UpdatedByName, &r.CreatedAt); err != nil {
			c.String(500, "扫描任务流数据失败: %v", err)
			return
		}
		flows = append(flows, r)
	}
	c.HTML(200, "taskflow/list.tmpl", gin.H{"Flows": flows})
}

// TaskFlowNewForm 显示创建新任务流的表单
func (ct *Controller) TaskFlowNewForm(c *gin.Context) {
	c.HTML(200, "taskflow/form.tmpl", gin.H{"IsEdit": false})
}

// TaskFlowCreate 处理新任务流的创建
func (ct *Controller) TaskFlowCreate(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	description := strings.TrimSpace(c.PostForm("description"))
	cronExpr := strings.TrimSpace(c.PostForm("cron"))

	// 创建人
	uid := ct.GetCurrentUserID(c)

	// 入库
	result, err := ct.db.Exec(`INSERT INTO task_flows(name, description, cron_expr, enabled, created_by, updated_by)
		VALUES(?, ?, ?, 1, ?, ?)`, name, description, cronExpr, uid, uid)
	if err != nil {
		c.String(500, "创建任务流失败: "+err.Error())
		return
	}

	// 获取新创建的任务流ID
	flowID, err := result.LastInsertId()
	if err != nil {
		c.String(500, "获取任务流ID失败: "+err.Error())
		return
	}

	// 将新任务流加入调度器
	if err := ct.sched.ReloadTaskFlow(int(flowID)); err != nil {
		log.Printf("scheduler: failed to add new task flow %d: %v", flowID, err)
		// 不返回错误，因为任务流已经创建成功，只是调度失败
	}

	c.Redirect(302, "/task-flows")
}

// TaskFlowProperties 显示任务流属性编辑页面
func (ct *Controller) TaskFlowProperties(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	// 获取任务流详情
	var name, description, cronExpr string
	err := ct.db.QueryRow("SELECT name, description, cron_expr FROM task_flows WHERE id=?", id).
		Scan(&name, &description, &cronExpr)
	if err != nil {
		c.String(404, "任务流不存在")
		return
	}

	c.HTML(200, "taskflow/form.tmpl", gin.H{
		"FlowID": id, "Name": name, "Description": description, "Cron": cronExpr, "IsEdit": true,
	})
}

// TaskFlowFlow 显示任务流图表编辑页面
func (ct *Controller) TaskFlowFlow(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	// 获取任务流详情
	var name string
	err := ct.db.QueryRow("SELECT name FROM task_flows WHERE id=?", id).Scan(&name)
	if err != nil {
		c.String(404, "任务流不存在")
		return
	}

	// 获取任务流步骤
	stepRows, _ := ct.db.Query(`
		SELECT s.id, s.step_order, s.timeout_minutes, 
		       t.name as task_name, t.id as task_id
		FROM task_flow_steps s
		JOIN tasks t ON s.task_id = t.id
		WHERE s.flow_id = ?
		ORDER BY s.step_order`, id)
	defer stepRows.Close()

	var steps []models.TaskFlowStep
	for stepRows.Next() {
		var s models.TaskFlowStep
		stepRows.Scan(&s.ID, &s.StepOrder, &s.TimeoutMinutes, &s.TaskName, &s.TaskID)
		steps = append(steps, s)
	}

	// 获取可用于添加步骤的任务（仅限未在任何流程中的任务）
	taskRows, _ := ct.db.Query(`
		SELECT t.id, t.name 
		FROM tasks t 
		LEFT JOIN task_flow_steps tfs ON t.id = tfs.task_id 
		WHERE tfs.task_id IS NULL 
		ORDER BY t.name
	`)
	defer taskRows.Close()
	var availableTasks []models.TaskFlowSelection
	for taskRows.Next() {
		var t models.TaskFlowSelection
		taskRows.Scan(&t.ID, &t.Name)
		availableTasks = append(availableTasks, t)
	}

	c.HTML(200, "taskflow/flow.tmpl", gin.H{
		"FlowID": id, "Name": name, "Steps": steps, "AvailableTasks": availableTasks,
	})
}

// TaskFlowRunNow 手动触发任务流执行（异步）
func (ct *Controller) TaskFlowRunNow(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// 检查任务流是否存在
	var exists bool
	err := ct.db.QueryRow("SELECT EXISTS(SELECT 1 FROM task_flows WHERE id=?)", id).Scan(&exists)
	if err != nil {
		c.String(500, "查询任务流失败: "+err.Error())
		return
	}
	if !exists {
		c.String(404, "任务流不存在")
		return
	}

	// 检查任务流是否已经在运行
	if ct.sched.IsTaskFlowRunning(id) {
		c.String(400, "任务流正在运行中，请稍后再试")
		return
	}

	// 异步执行任务流
	go func() {
		if err := ct.sched.RunTaskFlow(context.Background(), id); err != nil {
			log.Printf("Task flow %d execution failed: %v", id, err)
		}
	}()

	// 立即返回成功响应
	c.JSON(200, gin.H{
		"message":  "任务流已开始执行",
		"flow_id":  id,
		"redirect": fmt.Sprintf("/task-flows/%d", id),
	})
}

// TaskFlowToggle 切换任务流的启用状态
func (ct *Controller) TaskFlowToggle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	ct.db.Exec("UPDATE task_flows SET enabled=1-enabled WHERE id=?", id)

	// 在调度器中重新加载任务流以应用启用/禁用更改
	if err := ct.sched.ReloadTaskFlow(id); err != nil {
		log.Printf("Failed to reload task flow %d in scheduler: %v", id, err)
	}

	c.Redirect(302, "/task-flows")
}

// TaskFlowKill 取消正在运行的任务流
func (ct *Controller) TaskFlowKill(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := ct.sched.KillTaskFlow(id); err != nil {
		c.String(400, fmt.Sprintf("无法终止: %v", err))
		return
	}
	c.Redirect(302, fmt.Sprintf("/task-flows/%d", id))
}

// TaskFlowUpdate 处理任务流字段更新
func (ct *Controller) TaskFlowUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	description := strings.TrimSpace(c.PostForm("description"))
	cronExpr := strings.TrimSpace(c.PostForm("cron"))

	// 获取当前用户ID
	uid := ct.GetCurrentUserID(c)

	// 先查询当前的cron表达式
	var currentCronExpr string
	err := ct.db.QueryRow(`SELECT cron_expr FROM task_flows WHERE id=?`, id).Scan(&currentCronExpr)
	if err != nil {
		c.String(500, "查询失败: "+err.Error())
		return
	}

	// 更新数据库
	_, err = ct.db.Exec(`UPDATE task_flows SET description=?, cron_expr=?, updated_by=? WHERE id=?`,
		description, cronExpr, uid, id)
	if err != nil {
		c.String(500, "更新失败: "+err.Error())
		return
	}

	// 只有当cron表达式发生变化时才重新加载任务流
	if currentCronExpr != cronExpr {
		if err := ct.sched.ReloadTaskFlow(id); err != nil {
			log.Printf("Failed to reload task flow %d in scheduler: %v", id, err)
			// 向用户显示错误信息
			c.String(500, fmt.Sprintf("更新失败: cron表达式无效 - %v", err))
			return
		}
		log.Printf("Successfully reloaded task flow %d with new cron expression: %s", id, cronExpr)
	}

	c.Redirect(302, "/task-flows")
}

// TaskFlowDelete 永久删除任务流及其步骤
func (ct *Controller) TaskFlowDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// 先从cron调度中移除任务流
	if err := ct.sched.RemoveTaskFlowFromCron(id); err != nil {
		log.Printf("Failed to remove task flow %d from cron scheduler: %v", id, err)
	}

	// 先删除步骤，再删除任务流（避免外键约束问题）
	_, err := ct.db.Exec("DELETE FROM task_flow_steps WHERE flow_id=?", id)
	if err != nil {
		c.JSON(500, gin.H{"error": "删除任务流步骤失败: " + err.Error()})
		return
	}

	// 删除任务流
	result, err := ct.db.Exec("DELETE FROM task_flows WHERE id=?", id)
	if err != nil {
		c.JSON(500, gin.H{"error": "删除任务流失败: " + err.Error()})
		return
	}

	// 检查是否真的删除了
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(404, gin.H{"error": "任务流不存在"})
		return
	}

	c.JSON(200, gin.H{"message": "删除成功", "redirect": "/task-flows"})
}

// TaskFlowAddStep 向流程添加任务
func (ct *Controller) TaskFlowAddStep(c *gin.Context) {
	flowID, _ := strconv.Atoi(c.Param("id"))
	taskID, _ := strconv.Atoi(c.PostForm("task_id"))
	timeoutStr := strings.TrimSpace(c.PostForm("timeout_minutes"))

	// Get next step order
	var maxOrder int
	ct.db.QueryRow("SELECT COALESCE(MAX(step_order), 0) FROM task_flow_steps WHERE flow_id=?", flowID).Scan(&maxOrder)

	// Parse timeout
	var timeout *int
	if timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil && t > 0 {
			timeout = &t
		}
	}

	// 获取当前用户ID
	uid := ct.GetCurrentUserID(c)

	// 插入新步骤
	ct.db.Exec("INSERT INTO task_flow_steps (flow_id, task_id, step_order, timeout_minutes, created_by, updated_by) VALUES (?, ?, ?, ?, ?, ?)", flowID, taskID, maxOrder+1, timeout, uid, uid)

	c.Redirect(302, fmt.Sprintf("/task-flows/%d/flow", flowID))
}

// TaskFlowRemoveStep 从流程中移除步骤
func (ct *Controller) TaskFlowRemoveStep(c *gin.Context) {
	flowID, _ := strconv.Atoi(c.Param("id"))
	stepID, _ := strconv.Atoi(c.Param("step_id"))

	// 开始事务
	tx, err := ct.db.Begin()
	if err != nil {
		c.String(500, fmt.Sprintf("开始事务失败: %v", err))
		return
	}
	defer tx.Rollback()

	// 获取要删除的步骤的order
	var deletedOrder int
	err = tx.QueryRow("SELECT step_order FROM task_flow_steps WHERE id=? AND flow_id=?", stepID, flowID).Scan(&deletedOrder)
	if err != nil {
		c.String(404, "步骤不存在")
		return
	}

	// 删除步骤
	_, err = tx.Exec("DELETE FROM task_flow_steps WHERE id=? AND flow_id=?", stepID, flowID)
	if err != nil {
		c.String(500, fmt.Sprintf("删除步骤失败: %v", err))
		return
	}

	// 重新排序：将order大于被删除步骤的步骤order都减1
	_, err = tx.Exec("UPDATE task_flow_steps SET step_order = step_order - 1 WHERE flow_id = ? AND step_order > ?", flowID, deletedOrder)
	if err != nil {
		c.String(500, fmt.Sprintf("重新排序失败: %v", err))
		return
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		c.String(500, fmt.Sprintf("提交事务失败: %v", err))
		return
	}

	c.JSON(200, gin.H{"message": "步骤删除成功"})
}

// TaskFlowReorderSteps 更新步骤顺序
func (ct *Controller) TaskFlowReorderSteps(c *gin.Context) {
	flowID, _ := strconv.Atoi(c.Param("id"))
	stepOrders := c.PostFormArray("step_order")

	if len(stepOrders) == 0 {
		c.String(400, "没有提供步骤顺序")
		return
	}

	// 验证任务流是否存在
	var flowExists bool
	err := ct.db.QueryRow("SELECT EXISTS(SELECT 1 FROM task_flows WHERE id=?)", flowID).Scan(&flowExists)
	if err != nil {
		c.String(500, fmt.Sprintf("验证任务流失败: %v", err))
		return
	}
	if !flowExists {
		c.String(404, "任务流不存在")
		return
	}

	// 开始事务
	tx, err := ct.db.Begin()
	if err != nil {
		c.String(500, fmt.Sprintf("开始事务失败: %v", err))
		return
	}
	defer tx.Rollback()

	// 第一步：将所有步骤的step_order设置为临时值（避免唯一键冲突）
	_, err = tx.Exec("UPDATE task_flow_steps SET step_order = step_order + 10000 WHERE flow_id = ?", flowID)
	if err != nil {
		c.String(500, fmt.Sprintf("设置临时顺序失败: %v", err))
		return
	}

	// 第二步：按照新顺序更新step_order
	for i, orderStr := range stepOrders {
		stepID, err := strconv.Atoi(orderStr)
		if err != nil {
			c.String(400, fmt.Sprintf("无效的步骤ID: %s", orderStr))
			return
		}

		// 验证步骤是否属于该流程
		var stepExists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM task_flow_steps WHERE id=? AND flow_id=?)", stepID, flowID).Scan(&stepExists)
		if err != nil {
			c.String(500, fmt.Sprintf("验证步骤失败: %v", err))
			return
		}
		if !stepExists {
			c.String(400, fmt.Sprintf("步骤 %d 不属于该流程", stepID))
			return
		}

		_, err = tx.Exec("UPDATE task_flow_steps SET step_order=? WHERE id=? AND flow_id=?", i+1, stepID, flowID)
		if err != nil {
			c.String(500, fmt.Sprintf("更新步骤顺序失败: %v", err))
			return
		}
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		c.String(500, fmt.Sprintf("提交事务失败: %v", err))
		return
	}
	c.JSON(200, gin.H{"message": "步骤顺序更新成功"})
}
