package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/robfig/cron/v3"

	"com.duole/datax-web-go/internal/controllers"
	"com.duole/datax-web-go/internal/services"
	"com.duole/datax-web-go/internal/util"
)

// setupDatabase 使用 cfg 中的配置打开到 MySQL 的连接。
// 它设置合理的连接池大小并 ping 数据库以确保
// 连接性。如果无法建立连接，程序将终止。
func setupDatabase(cfg *util.Config) *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Asia%%2FShanghai",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}

	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxOpenConns(10) // 减少最大连接数
	db.SetMaxIdleConns(3)  // 减少空闲连接数
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	return db
}

// setupRouter 配置应用程序的所有路由。它接受一个
// 控制器实例并注册认证、仪表板、任务、数据源、
// 用户、工具和监控的处理器。在必要时应用
// 认证和授权的中间件。
func setupRouter(ct *controllers.Controller) *gin.Engine {
	r := gin.Default()
	// 加载模板
	r.LoadHTMLGlob("templates/**/*")
	// 提供静态文件
	r.Static("/static", "./static")
	// 认证路由
	r.GET("/login", ct.ShowLogin)
	r.POST("/login", ct.DoLogin)
	r.GET("/logout", ct.Logout)
	// 根路径重定向到任务
	r.GET("/", ct.MustLogin(), func(c *gin.Context) {
		c.Redirect(302, "/tasks")
	})
	// 任务管理
	r.GET("/tasks", ct.MustLogin(), ct.TaskList)
	r.GET("/tasks/new", ct.MustLogin(), ct.TaskNewForm)
	r.POST("/tasks", ct.MustLogin(), ct.TaskCreate)
	r.GET("/tasks/:id", ct.MustLogin(), ct.TaskManage)
	r.POST("/tasks/:id", ct.MustLogin(), ct.TaskUpdateJson)
	r.DELETE("/tasks/:id", ct.MustLogin(), ct.TaskDelete)
	r.POST("/tasks/:id/run", ct.MustLogin(), ct.TaskRunNow)
	// 任务流管理（带调度）
	r.GET("/task-flows", ct.MustLogin(), ct.TaskFlowList)
	r.GET("/task-flows/new", ct.MustLogin(), ct.TaskFlowNewForm)
	r.POST("/task-flows", ct.MustLogin(), ct.TaskFlowCreate)
	r.GET("/task-flows/:id", ct.MustLogin(), ct.TaskFlowProperties) // 直接到属性页
	r.GET("/task-flows/:id/flow", ct.MustLogin(), ct.TaskFlowFlow)
	r.POST("/task-flows/:id", ct.MustLogin(), ct.TaskFlowUpdate)
	r.DELETE("/task-flows/:id", ct.MustLogin(), ct.TaskFlowDelete)
	r.POST("/task-flows/:id/run", ct.MustLogin(), ct.TaskFlowRunNow)
	r.POST("/task-flows/:id/toggle", ct.MustLogin(), ct.TaskFlowToggle)
	r.POST("/task-flows/:id/kill", ct.MustLogin(), ct.TaskFlowKill)
	r.POST("/task-flows/:id/steps", ct.MustLogin(), ct.TaskFlowAddStep)
	r.DELETE("/task-flows/:id/steps/:step_id", ct.MustLogin(), ct.TaskFlowRemoveStep)
	r.PUT("/task-flows/:id/steps/reorder", ct.MustLogin(), ct.TaskFlowReorderSteps)
	// 数据源管理
	r.GET("/data-sources", ct.MustLogin(), ct.DSList)
	r.POST("/data-sources", ct.MustLogin(), ct.DSCreate)
	// 支持内联编辑的 JSON 获取：/data-sources/:id?format=json
	r.GET("/data-sources/:id", ct.MustLogin(), func(c *gin.Context) {
		if c.Query("format") == "json" {
			ct.DSGetOneJSON(c)
			return
		}
		// 默认重定向到列表页
		c.Redirect(302, "/data-sources")
	})
	r.POST("/data-sources/:id", ct.MustLogin(), ct.DSUpdate)
	r.DELETE("/data-sources/:id", ct.MustLogin(), ct.DSDelete)
	r.POST("/data-sources/test", ct.MustLogin(), ct.DSConnTest)
	// 元数据 API
	r.GET("/api/meta/mysql/:id/columns/:table", ct.MustLogin(), ct.MetaColumns)
	// 用户管理（仅管理员）
	r.GET("/admin/users", ct.MustLogin(), ct.MustAdmin(), ct.UserList)
	r.GET("/admin/users/new", ct.MustLogin(), ct.MustAdmin(), ct.UserNewForm)
	r.POST("/admin/users", ct.MustLogin(), ct.MustAdmin(), ct.UserCreate)
	r.POST("/admin/users/:id/toggle", ct.MustLogin(), ct.MustAdmin(), ct.UserToggle)
	// 工具：JSON 格式化页面
	r.GET("/tools/json-format", ct.MustLogin(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "tools/json-format.tmpl", gin.H{})
	})
	// 流程日志
	r.GET("/flow-logs", ct.MustLogin(), ct.FlowLogList)
	r.GET("/api/flow-logs", ct.MustLogin(), ct.GetFlowLogs)
	r.GET("/api/flow-logs/:id", ct.MustLogin(), ct.GetFlowLogDetail)
	// 任务日志
	r.GET("/task-logs", ct.MustLogin(), ct.TaskLogList)
	r.GET("/task-logs/:id", ct.MustLogin(), ct.TaskLogDetail)
	r.GET("/api/task-logs", ct.MustLogin(), ct.GetTaskLogs)
	r.GET("/api/task-logs/:id", ct.MustLogin(), ct.GetTaskLogDetail)
	// DataX 预览
	r.POST("/api/datax/preview", ct.MustLogin(), ct.DataXPreview)
	return r
}

func main() {
	// 默认配置文件路径
	configPath := "./config.yaml"

	// 如果命令行参数指定了配置文件路径，则使用指定的路径
	if len(os.Args) > 1 && os.Args[1] != "" {
		configPath = os.Args[1]
	}

	// 从YAML文件加载配置
	cfg := util.LoadConfigFromYaml(configPath)
	// Connect to DB
	db := setupDatabase(cfg)
	// Set up session store using a secret key from config
	store := sessions.NewCookieStore([]byte(cfg.SessionKey))
	// Create services
	auth := services.NewAuthService(db, store)
	c := cron.New(cron.WithSeconds())
	sched := services.NewScheduler(db, c, cfg.DataxHome, cfg.TempDir)
	// Initialize scheduler (handles both task execution and task flow scheduling)
	sched.LoadAndStart()
	// Create controller
	ct := controllers.NewController(db, auth, cfg, sched)
	// Set up router
	router := setupRouter(ct)
	// Run server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
