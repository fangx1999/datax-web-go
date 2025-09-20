package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/robfig/cron/v3"

	"com.duole/datax-web-go/internal/config"
	"com.duole/datax-web-go/internal/database"
	"com.duole/datax-web-go/internal/handler"
	"com.duole/datax-web-go/internal/service"
)

// setupDatabase 使用 cfg 中的配置打开到 MySQL 的连接
func setupDatabase(cfg *config.Config) *sql.DB {
	dsn := cfg.GetDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}

	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(3)
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	return db
}

// setupRouter 配置应用程序的所有路由，保持接口地址不变
func setupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default() // 使用默认中间件（包含日志和恢复中间件）

	// 加载模板
	r.LoadHTMLGlob("templates/**/*")

	// 初始化数据库单例
	database.Init(db)

	// 初始化handler单例
	handler.Init()
	h := handler.Get()

	// 认证路由
	r.GET("/login", h.Auth.ShowLogin)
	r.POST("/login", h.Auth.DoLogin)
	r.GET("/logout", h.Auth.Logout)

	// 根路径重定向
	r.GET("/", h.Auth.MustLogin(), func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/tasks")
	})

	// 任务管理
	tasks := r.Group("/tasks", h.Auth.MustLogin())
	{
		tasks.GET("", h.Task.List)
		tasks.GET("/new", h.Task.NewForm)
		tasks.POST("", h.Task.Create)
		tasks.GET("/:id", h.Task.Manage)
		tasks.POST("/:id", h.Task.UpdateJSON)
		tasks.DELETE("/:id", h.Task.Delete)
		tasks.POST("/:id/run", h.Task.RunNow)
	}

	// 任务流管理
	taskFlows := r.Group("/task-flows", h.Auth.MustLogin())
	{
		taskFlows.GET("", h.TaskFlow.List)
		taskFlows.GET("/new", h.TaskFlow.NewForm)
		taskFlows.POST("", h.TaskFlow.Create)
		taskFlows.GET("/:id", h.TaskFlow.Properties)
		taskFlows.GET("/:id/flow", h.TaskFlow.Flow)
		taskFlows.POST("/:id", h.TaskFlow.Update)
		taskFlows.DELETE("/:id", h.TaskFlow.Delete)
		taskFlows.POST("/:id/run", h.TaskFlow.RunNow)
		taskFlows.POST("/:id/toggle", h.TaskFlow.Toggle)
		taskFlows.POST("/:id/kill", h.TaskFlow.Kill)
		taskFlows.POST("/:id/steps", h.TaskFlow.AddStep)
		taskFlows.DELETE("/:id/steps/:step_id", h.TaskFlow.RemoveStep)
		taskFlows.PUT("/:id/steps/reorder", h.TaskFlow.ReorderSteps)
	}

	// 数据源管理
	dataSources := r.Group("/data-sources", h.Auth.MustLogin())
	{
		dataSources.GET("", h.DataSource.List)
		dataSources.POST("", h.DataSource.Create)
		dataSources.GET("/:id", h.DataSource.GetOneJSON)
		dataSources.POST("/:id", h.DataSource.Update)
		dataSources.DELETE("/:id", h.DataSource.Delete)
		dataSources.POST("/test", h.DataSource.TestConnection)
	}

	// 用户管理（仅管理员）
	admin := r.Group("/admin", h.Auth.MustLogin(), h.Auth.MustAdmin())
	{
		users := admin.Group("/users")
		{
			users.GET("", h.User.List)
			users.GET("/new", h.User.NewForm)
			users.POST("", h.User.Create)
			users.POST("/:id/toggle", h.User.Toggle)
		}
	}

	// 日志查看
	flowLogs := r.Group("/flow-logs", h.Auth.MustLogin())
	{
		flowLogs.GET("", h.Log.FlowLogList)
	}

	taskLogs := r.Group("/task-logs", h.Auth.MustLogin())
	{
		taskLogs.GET("", h.Log.TaskLogList)
		taskLogs.GET("/:id", h.Log.GetTaskLogDetail)
	}

	// API路由
	api := r.Group("/api", h.Auth.MustLogin())
	{
		api.GET("/meta/mysql/:id/columns/:table", h.Meta.Columns)
		api.POST("/datax/preview", h.DataX.ConfigPreview)
		api.GET("/task-logs", h.Log.GetTaskLogs)
		api.GET("/task-logs/:id", h.Log.GetTaskLogDetail)
		api.POST("/task-logs/:id/kill", h.Log.KillTaskByLog)
		api.GET("/flow-logs", h.Log.GetFlowLogs)
		api.GET("/flow-logs/:id", h.Log.GetFlowLogDetail)
		api.POST("/flow-logs/:id/kill", h.Log.KillFlowByLog)
	}

	// 工具页面
	r.GET("/tools/json-format", h.Auth.MustLogin(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "tools/json-format.tmpl", gin.H{})
	})

	// 系统路由
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204)
	})

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
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	db := setupDatabase(cfg)

	// 设置会话存储
	store := sessions.NewCookieStore([]byte(cfg.Session.Key))

	// 创建服务
	auth := service.NewAuthService(store)
	c := cron.New(cron.WithSeconds())

	// 初始化服务
	service.Init(db, c, cfg.DataX.Home, cfg.DataX.TempDir, auth)

	// 初始化调度器
	service.Get().Scheduler.LoadAndStart()

	// 设置路由
	router := setupRouter(db)

	// 运行服务器
	addr := cfg.GetServerAddr()
	if err := router.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
