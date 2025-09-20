package handler

import (
	"com.duole/datax-web-go/internal/database"
	"com.duole/datax-web-go/internal/entities"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
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

// DataSourceHandler 数据源处理器
type DataSourceHandler struct{}

// NewDataSourceHandler 创建数据源处理器
func NewDataSourceHandler() *DataSourceHandler {
	return &DataSourceHandler{}
}

// getDSFields 提取和验证数据源字段
func getDSFields(c *gin.Context, dsType string) DSFields {
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

// List 显示所有配置的数据源
func (h *DataSourceHandler) List(c *gin.Context) {
	dataSources, err := database.GetDB().DataSource.List()
	if err != nil {
		//todo 错误页面处理
		return
	}
	c.HTML(http.StatusOK, "data_source/list.tmpl", gin.H{"DataSources": dataSources})
}

// Create 创建新数据源
func (h *DataSourceHandler) Create(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	dsType := c.PostForm("type")
	fields := getDSFields(c, dsType)

	createdBy := c.GetInt("user_id")
	// 创建数据源实体
	ds := &entities.DataSource{
		Name:      name,
		Type:      dsType,
		CreatedBy: &createdBy,
		UpdatedBy: &createdBy,
	}
	// 根据类型设置配置
	if dsType == DSTypeMySQL {
		ds.DBURL = &fields.DBURL
		ds.DBUser = &fields.DBUser
		ds.DBPassword = &fields.DBPassword
		ds.DBDatabase = &fields.DBDatabase
	} else {
		ds.DefaultFS = &fields.DefaultFS
		ds.HadoopConfig = &fields.HadoopConfig
	}

	// 调用database层创建
	err := database.GetDB().DataSource.Create(ds)
	if err != nil {
		c.String(http.StatusInternalServerError, "创建数据源失败: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/data-sources")
}

// Update 更新数据源
func (h *DataSourceHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的数据源ID")
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	dsType := c.PostForm("type")
	fields := getDSFields(c, dsType)
	updatedBy := c.GetInt("user_id")

	ds := &entities.DataSource{
		ID:        id,
		Name:      name,
		Type:      dsType,
		UpdatedBy: &updatedBy,
	}
	// 根据类型设置配置
	if dsType == DSTypeMySQL {
		ds.DBURL = &fields.DBURL
		ds.DBUser = &fields.DBUser
		ds.DBPassword = &fields.DBPassword
		ds.DBDatabase = &fields.DBDatabase
	} else {
		ds.DefaultFS = &fields.DefaultFS
		ds.HadoopConfig = &fields.HadoopConfig
	}

	// 调用database层更新
	err = database.GetDB().DataSource.Update(ds)
	if err != nil {
		c.String(http.StatusInternalServerError, "更新数据源失败: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/data-sources")
}

// Delete 删除数据源
func (h *DataSourceHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的数据源ID")
		return
	}

	err = database.GetDB().DataSource.Delete(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "删除数据源失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功", "redirect": "/data-sources"})
}

// TestConnection 测试数据源连接
func (h *DataSourceHandler) TestConnection(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "无效的数据源ID")
		return
	}

	ds, err := database.GetDB().DataSource.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "数据源不存在")
		return
	}

	// 只测试MySQL连接
	if ds.IsMySQL() && ds.DBURL != nil && ds.DBUser != nil && ds.DBPassword != nil && ds.DBDatabase != nil {
		err = testConnection(*ds.DBURL, *ds.DBUser, *ds.DBPassword, *ds.DBDatabase)
		if err != nil {
			c.String(http.StatusInternalServerError, "连接测试失败: "+err.Error())
			return
		}
		c.String(http.StatusOK, "连接测试成功")
	} else {
		c.String(http.StatusBadRequest, "只支持MySQL数据源连接测试")
	}
}

// GetOneJSON 获取单个数据源的JSON信息
func (h *DataSourceHandler) GetOneJSON(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的数据源ID"})
		return
	}

	ds, err := database.GetDB().DataSource.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}

	c.JSON(http.StatusOK, ds)
}

// testConnection 测试数据库连接
func testConnection(url, user, password, dbname string) error {
	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&timeout=10s", user, password, url, dbname)

	// 创建临时连接
	testDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer testDB.Close()

	// 设置连接超时
	testDB.SetConnMaxLifetime(30 * time.Second)
	testDB.SetMaxOpenConns(1)
	testDB.SetMaxIdleConns(1)

	// 测试连接
	if err := testDB.Ping(); err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}

	return nil
}
