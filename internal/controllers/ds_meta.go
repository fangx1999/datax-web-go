package controllers

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
)

// MetaColumns 列出 MySQL 数据源表的列。
func (ct *Controller) MetaColumns(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	table := c.Param("table")
	var url, user, pass, dbname string
	err := ct.db.QueryRow(`SELECT db_url,db_user,db_password,db_database FROM data_sources WHERE id=? AND type='mysql'`, id).
		Scan(&url, &user, &pass, &dbname)
	if err != nil {
		c.JSON(400, gin.H{"error": "仅支持 MySQL 元数据或配置缺失"})
		return
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true", user, pass, url, dbname)
	dbc, openErr := sql.Open("mysql", dsn)
	if openErr != nil {
		c.JSON(500, gin.H{"error": "连接源库失败"})
		return
	}
	defer dbc.Close()
	rows, qerr := dbc.Query(`
		SELECT column_name, data_type, column_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema=? AND table_name=?
		ORDER BY ordinal_position`, dbname, table)
	if qerr != nil {
		c.JSON(500, gin.H{"error": "查询字段失败"})
		return
	}
	defer rows.Close()

	type col struct {
		Name       string `json:"name"`
		DataType   string `json:"data_type"`
		ColumnType string `json:"column_type"`
		Nullable   string `json:"nullable"`
	}
	var cols []col
	for rows.Next() {
		var cName, dType, cType, nul string
		if err := rows.Scan(&cName, &dType, &cType, &nul); err == nil {
			cols = append(cols, col{
				Name:       cName,
				DataType:   dType,
				ColumnType: cType,
				Nullable:   nul,
			})
		}
	}
	c.JSON(200, gin.H{"columns": cols})
}
