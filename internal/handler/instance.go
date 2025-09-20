package handler

import (
	"com.duole/datax-web-go/internal/service"
	"sync"
)

// 全局handler单例
var (
	instance *Handlers
	once     sync.Once
)

// Handlers 统一的handler实例
type Handlers struct {
	Auth       *AuthHandler
	Task       *TaskHandler
	TaskFlow   *TaskFlowHandler
	DataSource *DataSourceHandler
	User       *UserHandler
	Log        *LogHandler
	Meta       *MetaHandler
	DataX      *DataXHandler
}

// Init 初始化handler单例
func Init() {
	once.Do(func() {
		// 获取服务实例
		svc := service.Get()

		instance = &Handlers{
			Auth:       NewAuthHandler(svc.Auth),
			Task:       NewTaskHandler(),
			TaskFlow:   NewTaskFlowHandler(),
			DataSource: NewDataSourceHandler(),
			User:       NewUserHandler(),
			Log:        NewLogHandler(),
			Meta:       NewMetaHandler(),
			DataX:      NewDataXHandler(svc.DataX),
		}
	})
}

// Get 获取handler实例
func Get() *Handlers {
	if instance == nil {
		panic("handlers未初始化，请先调用 handler.Init()")
	}
	return instance
}
