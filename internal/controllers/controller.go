package controllers

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"net/http"

	"com.duole/datax-web-go/internal/services"
	"com.duole/datax-web-go/internal/util"
)

// Controller 聚合处理 HTTP 请求的依赖项。它持有
// 数据库、认证服务、配置和调度器的引用。
// 每个处理器使用这些引用来执行其任务。
type Controller struct {
	db              *sql.DB
	auth            *services.AuthService
	cfg             *util.Config
	sched           *services.Scheduler
	logController   *LogController
	dataxController *DataXController
	authController  *AuthController
}

// NewController 使用提供的依赖项构造 Controller。
func NewController(db *sql.DB, auth *services.AuthService, cfg *util.Config, sched *services.Scheduler) *Controller {
	return &Controller{
		db:              db,
		auth:            auth,
		cfg:             cfg,
		sched:           sched,
		logController:   NewLogController(db),
		dataxController: NewDataXController(db),
		authController:  NewAuthController(auth),
	}
}

// 委托给专门的控制器
func (ct *Controller) MustLogin() gin.HandlerFunc {
	return ct.authController.MustLogin()
}

func (ct *Controller) MustAdmin() gin.HandlerFunc {
	return ct.authController.MustAdmin()
}

func (ct *Controller) ShowLogin(c *gin.Context) {
	ct.authController.ShowLogin(c)
}

func (ct *Controller) DoLogin(c *gin.Context) {
	ct.authController.DoLogin(c)
}

func (ct *Controller) Logout(c *gin.Context) {
	ct.authController.Logout(c)
}

func (ct *Controller) DataXPreview(c *gin.Context) {
	ct.dataxController.DataXConfPreview(c)
}

// ========== Unified Log Handlers ==========

// FlowLogList 显示任务流日志列表页面
func (ct *Controller) FlowLogList(c *gin.Context) {
	c.HTML(http.StatusOK, "flow_log/list.tmpl", gin.H{})
}

// GetFlowLogs 获取任务流日志列表 (API)
func (ct *Controller) GetFlowLogs(c *gin.Context) {
	ct.logController.GetFlowLogs(c)
}

// GetFlowLogDetail 获取任务流日志详情 (API)
func (ct *Controller) GetFlowLogDetail(c *gin.Context) {
	ct.logController.GetFlowLogDetail(c)
}

// GetStepLogDetail 获取步骤日志详情 (API) - 已废弃，使用GetTaskLogDetail
func (ct *Controller) GetStepLogDetail(c *gin.Context) {
	ct.logController.GetTaskLogDetail(c)
}

// TaskLogList 显示任务日志列表页面
func (ct *Controller) TaskLogList(c *gin.Context) {
	c.HTML(http.StatusOK, "task_log/list.tmpl", gin.H{})
}

// TaskLogDetail 显示任务日志详情页面
func (ct *Controller) TaskLogDetail(c *gin.Context) {
	c.HTML(http.StatusOK, "task_log/detail.tmpl", gin.H{})
}

// GetTaskLogs 获取任务日志列表 (API)
func (ct *Controller) GetTaskLogs(c *gin.Context) {
	ct.logController.GetTaskLogs(c)
}

// GetTaskLogDetail 获取任务日志详情 (API)
func (ct *Controller) GetTaskLogDetail(c *gin.Context) {
	ct.logController.GetTaskLogDetail(c)
}
