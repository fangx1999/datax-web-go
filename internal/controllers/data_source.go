package controllers

import (
	"com.duole/datax-web-go/internal/models"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"time"
)

const (
	DSTypeMySQL = "mysql"
)

// DSFields 表示不同类型数据源的字段
type DSFields struct {
	DBURL        string
	DBUser       string
	DBPassword   string
	DBDatabase   string
	DefaultFS    string
	HadoopConfig string
}

// getDSFields extracts and validates data source fields from form data
func (ct *Controller) getDSFields(c *gin.Context, dsType string) DSFields {
	fields := DSFields{}

	if dsType == DSTypeMySQL {
		fields.DBURL = strings.TrimSpace(c.PostForm("db_url"))
		fields.DBUser = strings.TrimSpace(c.PostForm("db_user"))
		fields.DBPassword = c.PostForm("db_password")
		fields.DBDatabase = strings.TrimSpace(c.PostForm("db_database"))
	} else {
		fields.DefaultFS = strings.TrimSpace(c.PostForm("defaultfs"))
		fields.HadoopConfig = strings.TrimSpace(c.PostForm("hadoopconfig"))
	}

	return fields
}

// DSList 显示所有配置的数据源
func (ct *Controller) DSList(c *gin.Context) {
	query := `
		SELECT ds.id, ds.name, ds.type, 
		       COALESCE(uc.username, '系统') as created_by_name,
		       COALESCE(uu.username, '系统') as updated_by_name,
		       ds.created_at
		FROM data_sources ds
		LEFT JOIN users uc ON ds.created_by = uc.id
		LEFT JOIN users uu ON ds.updated_by = uu.id
		ORDER BY ds.id DESC
	`
	rows, _ := ct.db.Query(query)
	defer rows.Close()

	var list []models.DataSource
	for rows.Next() {
		var d models.DataSource
		rows.Scan(&d.ID, &d.Name, &d.Type, &d.CreatedByName, &d.UpdatedByName, &d.CreatedAt)
		list = append(list, d)
	}
	c.HTML(200, "data_source/list.tmpl", gin.H{"DataSources": list})
}

// DSGetOneJSON 返回单个数据源作为 JSON 用于内联编辑器
func (ct *Controller) DSGetOneJSON(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var ds models.DataSource
	query := `SELECT id,name,type,db_url,db_user,db_database,defaultfs,hadoopconfig FROM data_sources WHERE id=?`
	err := ct.db.QueryRow(query, id).
		Scan(&ds.ID, &ds.Name, &ds.Type, &ds.DBURL, &ds.DBUser, &ds.DBDatabase, &ds.DefaultFS, &ds.HadoopConfig)

	if err != nil {
		c.JSON(404, gin.H{"error": "数据源不存在"})
		return
	}

	c.JSON(200, ds)
}

// DSCreate 插入新数据源
func (ct *Controller) DSCreate(c *gin.Context) {
	typ := strings.TrimSpace(c.PostForm("type"))
	name := strings.TrimSpace(c.PostForm("name"))
	uid := ct.GetCurrentUserID(c)
	fields := ct.getDSFields(c, typ)

	var err error
	if typ == DSTypeMySQL {
		query := `INSERT INTO data_sources(name,type,db_url,db_user,db_password,db_database,created_by,updated_by) VALUES(?,?,?,?,?,?,?,?)`
		_, err = ct.db.Exec(query, name, typ, fields.DBURL, fields.DBUser, fields.DBPassword, fields.DBDatabase, uid, uid)
	} else {
		query := `INSERT INTO data_sources(name,type,defaultfs,hadoopconfig,created_by,updated_by) VALUES(?,?,?,?,?,?)`
		_, err = ct.db.Exec(query, name, typ, fields.DefaultFS, fields.HadoopConfig, uid, uid)
	}

	if err != nil {
		c.String(500, "创建数据源失败: "+err.Error())
		return
	}

	c.Redirect(302, "/data-sources")
}

// DSUpdate 更新现有数据源
func (ct *Controller) DSUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	typ := strings.TrimSpace(c.PostForm("type"))
	name := strings.TrimSpace(c.PostForm("name"))
	uid := ct.GetCurrentUserID(c)
	fields := ct.getDSFields(c, typ)

	if typ == DSTypeMySQL {
		query := `UPDATE data_sources SET name=?,db_url=?,db_user=?,db_password=?,db_database=?,updated_by=? WHERE id=?`
		ct.db.Exec(query, name, fields.DBURL, fields.DBUser, fields.DBPassword, fields.DBDatabase, uid, id)
	} else {
		query := `UPDATE data_sources SET name=?,defaultfs=?,hadoopconfig=?,updated_by=? WHERE id=?`
		ct.db.Exec(query, name, fields.DefaultFS, fields.HadoopConfig, uid, id)
	}

	c.Redirect(302, "/data-sources")
}

// DSDelete 删除数据源
func (ct *Controller) DSDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	query := "DELETE FROM data_sources WHERE id=?"
	result, err := ct.db.Exec(query, id)
	if err != nil {
		c.JSON(500, gin.H{"error": "删除数据源失败: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(404, gin.H{"error": "数据源不存在"})
		return
	}

	c.JSON(200, gin.H{"message": "删除成功", "redirect": "/data-sources"})
}

// ConnTestRequest 表示连接测试的请求结构
type ConnTestRequest struct {
	ID         string `json:"id"`
	DBURL      string `json:"db_url"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBDatabase string `json:"db_database"`
}

// DSConnTest 测试数据源连接
func (ct *Controller) DSConnTest(c *gin.Context) {
	var request ConnTestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(200, gin.H{"success": false, "error": "请求数据格式错误"})
		return
	}

	var url, user, pass, dbname string

	// 尝试从ID获取数据源信息（用于编辑时的测试）
	if request.ID != "" {
		if id, err := strconv.Atoi(request.ID); err == nil {
			query := `SELECT db_url,db_user,db_password,db_database FROM data_sources WHERE id=? AND type=?`
			ct.db.QueryRow(query, id, DSTypeMySQL).Scan(&url, &user, &pass, &dbname)
		}
	}

	// 如果从数据库没有获取到数据，则从请求数据获取（用于新建时的测试）
	if url == "" {
		url = strings.TrimSpace(request.DBURL)
		user = strings.TrimSpace(request.DBUser)
		pass = request.DBPassword
		dbname = strings.TrimSpace(request.DBDatabase)
	}

	// 验证必要字段
	if url == "" || user == "" || dbname == "" {
		c.JSON(200, gin.H{"success": false, "error": "缺少必要的连接参数"})
		return
	}

	// 测试连接
	if err := ct.pingMySQL(url, user, pass, dbname); err != nil {
		c.JSON(200, gin.H{"success": false, "error": "连接失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": "连接成功"})
}

// pingMySQL tests MySQL connection with proper DSN formatting
func (ct *Controller) pingMySQL(url, user, pass, dbname string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&timeout=10s", user, pass, url, dbname)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// 设置连接超时
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db.Ping()
}
