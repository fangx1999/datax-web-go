package controllers

import (
	"com.duole/datax-web-go/internal/models"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

// TaskList 显示所有任务
func (ct *Controller) TaskList(c *gin.Context) {
	rows, _ := ct.db.Query(`
		SELECT t.id, t.name, tf.name, tf.id, 
		       COALESCE(uc.username, '系统') as created_by_name,
		       COALESCE(uu.username, '系统') as updated_by_name,
		       t.created_at, t.updated_at
		FROM tasks t 
		LEFT JOIN task_flow_steps tfs ON t.id=tfs.task_id 
		LEFT JOIN task_flows tf ON tfs.flow_id=tf.id
		LEFT JOIN users uc ON t.created_by = uc.id
		LEFT JOIN users uu ON t.updated_by = uu.id
		ORDER BY t.id DESC
	`)
	defer rows.Close()
	var tasks []models.Task
	for rows.Next() {
		var r models.Task
		rows.Scan(&r.ID, &r.Name, &r.FlowName, &r.FlowID, &r.CreatedByName, &r.UpdatedByName, &r.CreatedAt, &r.UpdatedAt)
		tasks = append(tasks, r)
	}
	c.HTML(200, "task/list.tmpl", gin.H{"Tasks": tasks})
}

// TaskNewForm 显示创建新任务的表单
func (ct *Controller) TaskNewForm(c *gin.Context) {
	// 获取各种类型的数据源
	mysql, err := ct.GetDataSourcesByType("mysql")
	if err != nil {
		c.String(500, fmt.Sprintf("获取MySQL数据源失败: %v", err))
		return
	}

	ofs, err := ct.GetDataSourcesByType("ofs")
	if err != nil {
		c.String(500, fmt.Sprintf("获取OFS数据源失败: %v", err))
		return
	}

	hdfs, err := ct.GetDataSourcesByType("hdfs")
	if err != nil {
		c.String(500, fmt.Sprintf("获取HDFS数据源失败: %v", err))
		return
	}

	cosn, err := ct.GetDataSourcesByType("cosn")
	if err != nil {
		c.String(500, fmt.Sprintf("获取COSN数据源失败: %v", err))
		return
	}

	// 只获取未禁用的任务流
	rows, err := ct.db.Query("SELECT id, name FROM task_flows WHERE enabled = 1 ORDER BY name")
	if err != nil {
		c.String(500, fmt.Sprintf("获取任务流失败: %v", err))
		return
	}
	defer rows.Close()

	var taskFlows []models.TaskFlowSelection
	for rows.Next() {
		var tf models.TaskFlowSelection
		if err := rows.Scan(&tf.ID, &tf.Name); err != nil {
			c.String(500, fmt.Sprintf("扫描任务流数据失败: %v", err))
			return
		}
		taskFlows = append(taskFlows, tf)
	}

	c.HTML(200, "task/new.tmpl", gin.H{
		"MySQL":     mysql,
		"OFS":       ofs,
		"HDFS":      hdfs,
		"COSN":      cosn,
		"TaskFlows": taskFlows,
	})
}

// TaskCreate 处理新任务的创建
func (ct *Controller) TaskCreate(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	rawJSON := strings.TrimSpace(c.PostForm("datax_json"))

	// 验证必需参数
	if name == "" {
		c.String(400, "任务名称不能为空")
		return
	}

	if rawJSON == "" {
		c.String(400, "JSON配置不能为空")
		return
	}

	// 解析并验证ID参数
	srcID, err := strconv.Atoi(c.PostForm("source_id"))
	if err != nil || srcID <= 0 {
		c.String(400, "无效的源数据源ID")
		return
	}

	tgtID, err := strconv.Atoi(c.PostForm("target_id"))
	if err != nil || tgtID <= 0 {
		c.String(400, "无效的目标数据源ID")
		return
	}

	flowID, err := strconv.Atoi(c.PostForm("flow_id"))
	if err != nil || flowID <= 0 {
		c.String(400, "无效的任务流ID")
		return
	}

	// 验证JSON格式
	var job map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &job); err != nil {
		c.String(400, "JSON格式不正确")
		return
	}

	// 美化JSON
	pretty, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		c.String(500, "JSON格式化失败")
		return
	}

	// 获取当前用户ID
	userID := ct.GetCurrentUserID(c)

	// 开始事务
	tx, err := ct.db.Begin()
	if err != nil {
		c.String(500, "数据库事务开始失败")
		return
	}
	defer tx.Rollback()

	// 创建任务
	result, err := tx.Exec(`INSERT INTO tasks(name, source_id, target_id, json_config, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?)`, name, srcID, tgtID, string(pretty), userID, userID)
	if err != nil {
		c.String(500, "创建任务失败")
		return
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		c.String(500, "获取任务ID失败")
		return
	}

	// 添加到任务流 - 获取下一个步骤顺序
	var maxOrder int
	err = tx.QueryRow("SELECT COALESCE(MAX(step_order), 0) FROM task_flow_steps WHERE flow_id=?", flowID).Scan(&maxOrder)
	if err != nil {
		c.String(500, "获取步骤顺序失败")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO task_flow_steps (flow_id, task_id, step_order)
		VALUES (?, ?, ?)`, flowID, taskID, maxOrder+1)
	if err != nil {
		c.String(500, "添加任务到流程失败")
		return
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		c.String(500, "提交事务失败")
		return
	}

	c.Redirect(302, "/tasks")
}

// TaskManage 显示任务管理页面
func (ct *Controller) TaskManage(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var task models.Task
	err := ct.db.QueryRow(`SELECT t.id, t.name, t.source_id, t.target_id, COALESCE(t.json_config,''), t.created_at, t.updated_at,
        (SELECT name FROM data_sources WHERE id=t.source_id),
        (SELECT name FROM data_sources WHERE id=t.target_id) FROM tasks t WHERE t.id=?`, id).
		Scan(&task.ID, &task.Name, &task.SourceID, &task.TargetID, &task.JsonConfig, &task.CreatedAt, &task.UpdatedAt, &task.Source, &task.Target)

	if err != nil {
		c.String(404, "任务不存在")
		return
	}

	c.HTML(200, "task/manage.tmpl", gin.H{
		"Task": task,
	})
}

// TaskUpdateJson 处理任务 JSON 配置更新
func (ct *Controller) TaskUpdateJson(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	jsonConfig := strings.TrimSpace(c.PostForm("datax_json"))

	// 验证JSON格式
	var jsonData interface{}
	if err := json.Unmarshal([]byte(jsonConfig), &jsonData); err != nil {
		c.String(400, "JSON格式不正确")
		return
	}

	// 获取当前用户ID
	userID := ct.GetCurrentUserID(c)

	// 更新任务JSON配置
	_, err := ct.db.Exec(`UPDATE tasks SET json_config=?, updated_by=? WHERE id=?`, jsonConfig, userID, id)
	if err != nil {
		c.String(500, "更新任务失败")
		return
	}

	c.Redirect(302, fmt.Sprintf("/tasks/%d", id))
}

// TaskRunNow 手动触发任务执行
func (ct *Controller) TaskRunNow(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	_, err := ct.sched.RunTask(c, id)
	if err != nil {
		c.String(500, fmt.Sprintf("执行失败: %v", err))
		return
	}
	c.Redirect(302, fmt.Sprintf("/tasks/%d", id))
}

// TaskDelete 永久删除任务
func (ct *Controller) TaskDelete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(400, gin.H{"error": "无效的任务ID"})
		return
	}

	// 开始事务
	tx, err := ct.db.Begin()
	if err != nil {
		c.JSON(500, gin.H{"error": "数据库事务开始失败"})
		return
	}
	defer tx.Rollback()

	// 删除相关记录
	_, err = tx.Exec("DELETE FROM task_flow_steps WHERE task_id=?", id)
	if err != nil {
		c.JSON(500, gin.H{"error": "删除任务流步骤失败"})
		return
	}

	result, err := tx.Exec("DELETE FROM tasks WHERE id=?", id)
	if err != nil {
		c.JSON(500, gin.H{"error": "删除任务失败"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		c.JSON(404, gin.H{"error": "任务不存在"})
		return
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "提交事务失败"})
		return
	}

	c.JSON(200, gin.H{"message": "删除成功", "redirect": "/tasks"})
}
