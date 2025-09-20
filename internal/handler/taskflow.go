package handler

import (
	"com.duole/datax-web-go/internal/database"
	"com.duole/datax-web-go/internal/entities"
	"com.duole/datax-web-go/internal/service"
	"context"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type TaskFlowHandler struct{}

func NewTaskFlowHandler() *TaskFlowHandler {
	return &TaskFlowHandler{}
}

// List 显示所有任务流
func (h *TaskFlowHandler) List(c *gin.Context) {
	flows, err := database.GetDB().TaskFlow.List()
	if err != nil {
		//todo 错误页面处理
		return
	}
	c.HTML(http.StatusOK, "taskflow/list.tmpl", gin.H{"Flows": flows})
}

// NewForm 显示创建新任务流的表单
func (h *TaskFlowHandler) NewForm(c *gin.Context) {
	c.HTML(http.StatusOK, "taskflow/form.tmpl", gin.H{"IsEdit": false})
}

// Create 创建新任务流
func (h *TaskFlowHandler) Create(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	description := strings.TrimSpace(c.PostForm("description"))
	cronExpr := strings.TrimSpace(c.PostForm("cron_expr"))

	// 验证必需参数
	if name == "" {
		c.String(http.StatusBadRequest, "任务流名称不能为空")
		return
	}

	// 获取当前用户ID
	userID := c.GetInt("user_id")

	// 创建任务流实体
	taskFlow := &entities.TaskFlow{
		Name:        name,
		Description: description,
		CronExpr:    cronExpr,
		Enabled:     true,
		CreatedBy:   &userID,
		UpdatedBy:   &userID,
	}

	// 调用database层创建
	err := database.GetDB().TaskFlow.Create(taskFlow)
	if err != nil {
		c.String(http.StatusInternalServerError, "创建任务流失败: "+err.Error())
		return
	}

	if err := service.Get().Scheduler.ReloadTaskFlow(taskFlow.ID); err != nil {
		log.Printf("taskflow: failed to schedule new flow %d: %v", taskFlow.ID, err)
	}

	c.Redirect(http.StatusFound, "/task-flows")
}

// Properties 显示任务流属性页面
func (h *TaskFlowHandler) Properties(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务流ID")
		return
	}

	flow, err := database.GetDB().TaskFlow.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "任务流不存在")
		return
	}

	c.HTML(http.StatusOK, "taskflow/form.tmpl", gin.H{
		"IsEdit": true,
		"Flow":   flow,
	})
}

// Update 更新任务流
func (h *TaskFlowHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务流ID")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	description := strings.TrimSpace(c.PostForm("description"))
	cronExpr := strings.TrimSpace(c.PostForm("cron_expr"))

	// 验证必需参数
	if name == "" {
		c.String(http.StatusBadRequest, "任务流名称不能为空")
		return
	}

	// 获取当前用户ID
	userID := c.GetInt("user_id")

	// 获取现有任务流
	taskFlow, err := database.GetDB().TaskFlow.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "任务流不存在")
		return
	}

	// 更新字段
	taskFlow.Name = name
	taskFlow.Description = description
	taskFlow.CronExpr = cronExpr
	taskFlow.UpdatedBy = &userID

	// 调用database层更新
	err = database.GetDB().TaskFlow.Update(taskFlow)
	if err != nil {
		c.String(http.StatusInternalServerError, "更新任务流失败: "+err.Error())
		return
	}

	if err := service.Get().Scheduler.ReloadTaskFlow(taskFlow.ID); err != nil {
		log.Printf("taskflow: failed to reload flow %d: %v", taskFlow.ID, err)
	}

	c.Redirect(http.StatusFound, "/task-flows")
}

// Delete 删除任务流
func (h *TaskFlowHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务流ID")
		return
	}

	if err := service.Get().Scheduler.RemoveTaskFlowFromCron(id); err != nil {
		log.Printf("taskflow: failed to remove flow %d from scheduler: %v", id, err)
	}

	err = database.GetDB().TaskFlow.Delete(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "删除任务流失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功", "redirect": "/task-flows"})
}

// Toggle 切换任务流启用/禁用状态
func (h *TaskFlowHandler) Toggle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务流ID"})
		return
	}

	// 获取当前用户ID
	userID := c.GetInt("user_id")

	// 获取现有任务流
	taskFlow, err := database.GetDB().TaskFlow.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务流不存在"})
		return
	}

	// 切换状态
	taskFlow.Enabled = !taskFlow.Enabled
	taskFlow.UpdatedBy = &userID

	// 调用database层更新
	err = database.GetDB().TaskFlow.Update(taskFlow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "切换任务流状态失败: " + err.Error()})
		return
	}

	if err := service.Get().Scheduler.ReloadTaskFlow(taskFlow.ID); err != nil {
		log.Printf("taskflow: failed to reload flow %d after toggle: %v", taskFlow.ID, err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "任务流状态更新成功"})
}

// Flow 显示任务流流程图页面
func (h *TaskFlowHandler) Flow(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务流ID")
		return
	}

	flow, err := database.GetDB().TaskFlow.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "任务流不存在")
		return
	}

	// 获取任务流步骤
	steps, err := database.GetDB().TaskFlow.GetSteps(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "获取任务流步骤失败: "+err.Error())
		return
	}

	// 获取可用任务
	availableTasks, err := database.GetDB().Task.GetAvailableTasks()
	if err != nil {
		c.String(http.StatusInternalServerError, "获取可用任务失败: "+err.Error())
		return
	}

	c.HTML(http.StatusOK, "taskflow/flow.tmpl", gin.H{
		"FlowID":         id,
		"CronExpr":       flow.CronExpr,
		"Enabled":        flow.Enabled,
		"Steps":          steps,
		"AvailableTasks": availableTasks,
	})
}

// AddStep 添加步骤到任务流
func (h *TaskFlowHandler) AddStep(c *gin.Context) {
	flowID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务流ID")
		return
	}

	taskID, err := strconv.Atoi(c.PostForm("task_id"))
	if err != nil || taskID <= 0 {
		c.String(http.StatusBadRequest, "无效的任务ID")
		return
	}

	// 调用service层添加步骤
	err = database.GetDB().TaskFlow.AddStep(flowID, taskID)
	if err != nil {
		c.String(http.StatusInternalServerError, "添加步骤失败: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/task-flows/"+strconv.Itoa(flowID)+"/flow")
}

// RemoveStep 从任务流中移除步骤
func (h *TaskFlowHandler) RemoveStep(c *gin.Context) {
	flowID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务流ID")
		return
	}

	stepID, err := strconv.Atoi(c.Param("step_id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的步骤ID")
		return
	}

	// 调用service层移除步骤
	err = database.GetDB().TaskFlow.RemoveStep(flowID, stepID)
	if err != nil {
		c.String(http.StatusInternalServerError, "移除步骤失败: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/task-flows/"+strconv.Itoa(flowID)+"/flow")
}

// ReorderSteps 重新排序步骤
func (h *TaskFlowHandler) ReorderSteps(c *gin.Context) {
	flowID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务流ID")
		return
	}

	// 解析步骤顺序
	orderValues := c.PostFormArray("step_orders[]")
	stepOrders := make([]int, 0, len(orderValues))
	for _, value := range orderValues {
		stepID, convErr := strconv.Atoi(value)
		if convErr != nil {
			c.String(http.StatusBadRequest, "无效的步骤顺序")
			return
		}
		stepOrders = append(stepOrders, stepID)
	}

	if len(stepOrders) == 0 {
		c.String(http.StatusBadRequest, "没有提供步骤顺序")
		return
	}

	// 调用service层重新排序
	err = database.GetDB().TaskFlow.ReorderSteps(flowID, stepOrders)
	if err != nil {
		c.String(http.StatusInternalServerError, "重新排序失败: "+err.Error())
		return
	}

	c.String(http.StatusOK, "步骤顺序更新成功")
}

// RunNow 立即执行任务流
func (h *TaskFlowHandler) RunNow(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务流ID")
		return
	}

	svc := service.Get().Scheduler
	go func(flowID int) {
		if runErr := svc.RunTaskFlow(context.Background(), flowID); runErr != nil {
			log.Printf("taskflow: run flow %d failed: %v", flowID, runErr)
		}
	}(id)

	c.JSON(http.StatusOK, gin.H{"message": "任务流已提交执行"})
}

// Kill 终止任务流执行
func (h *TaskFlowHandler) Kill(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务流ID")
		return
	}

	if err := service.Get().Scheduler.KillTaskFlow(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "任务流已终止"})
}
