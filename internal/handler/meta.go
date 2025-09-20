package handler

import (
	"com.duole/datax-web-go/internal/database"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// MetaHandler 元数据处理器
type MetaHandler struct{}

// NewMetaHandler 创建元数据处理器
func NewMetaHandler() *MetaHandler {
	return &MetaHandler{}
}

// Columns 列出 MySQL 数据源表的列
func (h *MetaHandler) Columns(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的数据源ID"})
		return
	}

	table := c.Param("table")
	if table == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "表名不能为空"})
		return
	}

	// 获取数据源信息
	ds, err := database.GetDB().DataSource.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}

	// 只支持MySQL数据源
	if !ds.IsMySQL() || ds.DBURL == nil || ds.DBUser == nil || ds.DBPassword == nil || ds.DBDatabase == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 MySQL 元数据或配置缺失"})
		return
	}

	// 构建连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true",
		*ds.DBUser, *ds.DBPassword, *ds.DBURL, *ds.DBDatabase)

	// 连接数据库
	dbc, err := sql.Open("mysql", dsn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "连接源库失败"})
		return
	}
	defer dbc.Close()

	// 查询表结构
	rows, err := dbc.Query(`
		SELECT column_name, data_type, column_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema=? AND table_name=?
		ORDER BY ordinal_position`, *ds.DBDatabase, table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询字段失败"})
		return
	}
	defer rows.Close()

	type Column struct {
		Name       string `json:"name"`
		DataType   string `json:"data_type"`
		ColumnType string `json:"column_type"`
		Nullable   string `json:"nullable"`
	}

	var columns []Column
	for rows.Next() {
		var name, dataType, columnType, nullable string
		if err := rows.Scan(&name, &dataType, &columnType, &nullable); err == nil {
			columns = append(columns, Column{
				Name:       name,
				DataType:   dataType,
				ColumnType: columnType,
				Nullable:   nullable,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"columns": columns})
}
