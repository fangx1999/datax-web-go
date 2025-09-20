package database

import (
	"database/sql"
	"sync"
)

// 全局数据库连接单例
var (
	db   *sql.DB
	once sync.Once
)

// Init 初始化数据库连接（在main.go中调用）
func Init(database *sql.DB) {
	once.Do(func() {
		db = database
	})
}

// DB 统一的数据库操作层（单例）
type DB struct {
	DataSource DataSourceDB
	Task       TaskDB
	TaskFlow   TaskFlowDB
	User       UserDB
	Log        LogDB
}

var (
	instance *DB
	onceDB   sync.Once
)

// GetDB 获取数据库操作单例实例
func GetDB() *DB {
	onceDB.Do(func() {
		if db == nil {
			panic("数据库未初始化，请先调用 database.Init()")
		}
		instance = &DB{}
	})
	return instance
}
