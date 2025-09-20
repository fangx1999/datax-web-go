package handler

import (
	"com.duole/datax-web-go/internal/database"
	"com.duole/datax-web-go/internal/entities"
	"com.duole/datax-web-go/internal/service"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type TaskHandler struct{}

func NewTaskHandler() *TaskHandler {
	return &TaskHandler{}
}

// List 显示所有任务
func (h *TaskHandler) List(c *gin.Context) {
	tasks, err := database.GetDB().Task.List()
	if err != nil {
		//todo 错误页面处理
		return
	}
	c.HTML(http.StatusOK, "task/list.tmpl", gin.H{"Tasks": tasks})
}

// NewForm 显示创建新任务的表单
func (h *TaskHandler) NewForm(c *gin.Context) {
	// 获取各种类型的数据源
	mysql, err := database.GetDB().DataSource.GetByType("mysql")
	if err != nil {
		//todo 错误处理
		return
	}

	ofs, err := database.GetDB().DataSource.GetByType("ofs")
	if err != nil {
		//todo 错误处理
		return
	}

	hdfs, err := database.GetDB().DataSource.GetByType("hdfs")
	if err != nil {
		//todo 错误处理
		return
	}

	cosn, err := database.GetDB().DataSource.GetByType("cosn")
	if err != nil {
		//todo 错误处理
		return
	}

	// 获取启用的任务流列表
	taskFlows, err := database.GetDB().TaskFlow.ListEnabled()
	if err != nil {
		//todo 错误处理
		return
	}

	c.HTML(http.StatusOK, "task/new.tmpl", gin.H{
		"MySQL":     mysql,
		"OFS":       ofs,
		"HDFS":      hdfs,
		"COSN":      cosn,
		"TaskFlows": taskFlows,
	})
}

// Create 创建新任务
func (h *TaskHandler) Create(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	rawJSON := strings.TrimSpace(c.PostForm("datax_json"))

	if name == "" {
		c.String(http.StatusBadRequest, "任务名称不能为空")
		return
	}

	if rawJSON == "" {
		c.String(http.StatusBadRequest, "JSON配置不能为空")
		return
	}

	// 解析并验证ID参数
	srcID, err := strconv.Atoi(c.PostForm("source_id"))
	if err != nil || srcID <= 0 {
		c.String(http.StatusBadRequest, "无效的源数据源ID")
		return
	}

	tgtID, err := strconv.Atoi(c.PostForm("target_id"))
	if err != nil || tgtID <= 0 {
		c.String(http.StatusBadRequest, "无效的目标数据源ID")
		return
	}

	// 任务流ID为可选，0表示独立任务
	flowID := 0
	if flowIDStr := c.PostForm("flow_id"); flowIDStr != "" {
		flowID, err = strconv.Atoi(flowIDStr)
		if err != nil || flowID < 0 {
			c.String(http.StatusBadRequest, "无效的任务流ID")
			return
		}
	}

	// 获取当前用户ID
	userID := c.GetInt("user_id")

	// 验证JSON格式
	var job map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &job); err != nil {
		c.String(http.StatusBadRequest, "JSON格式不正确")
		return
	}

	// 美化JSON
	pretty, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		c.String(http.StatusInternalServerError, "JSON格式化失败")
		return
	}

	// 创建任务实体
	task := &entities.Task{
		Name:       name,
		SourceID:   srcID,
		TargetID:   tgtID,
		JsonConfig: string(pretty),
		CreatedBy:  &userID,
		UpdatedBy:  &userID,
	}

	err = database.GetDB().Task.Create(task)
	if err != nil {
		c.String(http.StatusInternalServerError, "创建任务失败: "+err.Error())
		return
	}

	// 如果指定了任务流，则添加到任务流中
	if flowID > 0 {
		err = database.GetDB().TaskFlow.AddStep(flowID, task.ID)
		if err != nil {
			c.String(http.StatusInternalServerError, "添加到任务流失败: "+err.Error())
			return
		}
	}

	c.Redirect(http.StatusFound, "/tasks")
}

// Manage 任务管理页面
func (h *TaskHandler) Manage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务ID")
		return
	}

	task, err := database.GetDB().Task.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "任务不存在")
		return
	}

	c.HTML(http.StatusOK, "task/manage.tmpl", gin.H{"Task": task})
}

// UpdateJSON 更新任务JSON配置
func (h *TaskHandler) UpdateJSON(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务ID")
		return
	}

	rawJSON := strings.TrimSpace(c.PostForm("datax_json"))
	if rawJSON == "" {
		c.String(http.StatusBadRequest, "JSON配置不能为空")
		return
	}

	// 验证JSON格式
	var job map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &job); err != nil {
		c.String(http.StatusBadRequest, "JSON格式不正确")
		return
	}

	updatedBy := c.GetInt("user_id")
	task := &entities.Task{
		ID:         id,
		JsonConfig: rawJSON,
		UpdatedBy:  &updatedBy,
	}

	// 调用database层更新
	err = database.GetDB().Task.Update(task)
	if err != nil {
		c.String(http.StatusInternalServerError, "更新任务失败: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/tasks/%d", id))
}

// Delete 删除任务
func (h *TaskHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务ID")
		return
	}

	err = database.GetDB().Task.Delete(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "删除任务失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功", "redirect": "/tasks"})
}

// RunNow 立即执行任务
func (h *TaskHandler) RunNow(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的任务ID")
		return
	}

	// 异步执行任务
	go func() {
		// 直接获取服务实例
		svc := service.Get()
		ctx := context.Background()
		_, err := svc.Scheduler.RunTask(ctx, id)
		if err != nil {
			log.Printf("任务 %d 执行失败: %v", id, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "任务已提交执行"})
}
