package datax

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// GetMySQLConnection 根据 ID 获取 MySQL 数据源连接配置
func GetMySQLConnection(db *sql.DB, id int) (*MySQLConnection, error) {
	var url, user, pass, database string
	var typ string

	err := db.QueryRow("SELECT type, db_url, db_user, db_password, db_database FROM data_sources WHERE id = ?",
		id).Scan(&typ, &url, &user, &pass, &database)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("数据源不存在")
		}
		return nil, fmt.Errorf("查询数据源失败: %v", err)
	}

	if typ != "mysql" {
		return nil, errors.New("数据源类型不是MySQL")
	}

	return &MySQLConnection{
		Host: url,
		User: user,
		Pass: pass,
		DB:   database,
	}, nil
}

// GetFSConnection 根据 ID 获取文件系统数据源连接配置
func GetFSConnection(db *sql.DB, id int) (*FSConnection, error) {
	var defaultfs, hadoopcfg string
	var typ string

	err := db.QueryRow(
		"SELECT type, defaultfs, hadoopconfig FROM data_sources WHERE id = ?",
		id,
	).Scan(&typ, &defaultfs, &hadoopcfg)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("数据源不存在")
		}
		return nil, fmt.Errorf("查询数据源失败: %v", err)
	}

	if typ != "ofs" && typ != "hdfs" && typ != "cosn" {
		return nil, errors.New("数据源类型不是文件系统类型")
	}

	// 解析hadoopconfig字符串为map
	hadoopConfig := make(map[string]string)

	// 如果hadoopcfg不为空，解析为map
	if hadoopcfg != "" {
		// 尝试解析为JSON格式
		err := json.Unmarshal([]byte(hadoopcfg), &hadoopConfig)
		if err != nil {
			// 如果不是JSON格式，尝试解析为简单的键值对格式
			// 例如：key1=value1,key2=value2
			pairs := strings.Split(hadoopcfg, ",")
			for _, pair := range pairs {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					hadoopConfig[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
		}
	}

	return &FSConnection{
		DefaultFS:    defaultfs,
		HadoopConfig: hadoopConfig,
	}, nil
}
