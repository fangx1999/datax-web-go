package service

import (
	"com.duole/datax-web-go/internal/service/datax"
	"database/sql"
	"github.com/robfig/cron/v3"
	"sync"
)

// 全局服务实例
var (
	instance *Service
	once     sync.Once
)

// Service 统一的服务层接口
type Service struct {
	Auth      *AuthService
	Scheduler *Scheduler
	DataX     *datax.Service
}

// Init 初始化全局服务实例
func Init(db *sql.DB, c *cron.Cron, dataxHome, tempDir string, auth *AuthService) {
	once.Do(func() {
		// 创建DataX服务
		dataxService := datax.NewService(db)

		// 创建调度器
		scheduler := NewScheduler(db, c, dataxHome, tempDir)

		instance = &Service{
			Auth:      auth,
			Scheduler: scheduler,
			DataX:     dataxService,
		}
	})
}

// Get 获取全局服务实例
func Get() *Service {
	if instance == nil {
		panic("服务未初始化，请先调用 service.Init()")
	}
	return instance
}
