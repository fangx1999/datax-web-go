package datax

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ConfigBuilder DataX 配置构建器
type ConfigBuilder struct {
	db *sql.DB
}

// NewConfigBuilder 创建配置构建器
func NewConfigBuilder(db *sql.DB) *ConfigBuilder {
	return &ConfigBuilder{db: db}
}

// mapMySQLToDataX MySQL 类型映射到 DataX 类型
func mapMySQLToDataX(dataType string) string {
	dataType = strings.ToLower(dataType)
	switch {
	case strings.Contains(dataType, "int"):
		return "long"
	case strings.Contains(dataType, "double") || strings.Contains(dataType, "float") || strings.Contains(dataType, "decimal"):
		return "double"
	case strings.Contains(dataType, "bool") || strings.Contains(dataType, "bit"):
		return "boolean"
	case dataType == "date":
		return "date"
	case strings.Contains(dataType, "time"): // datetime/timestamp
		return "timestamp"
	default:
		return "string"
	}
}

// BuildConfig 构建 DataX 配置
func (b *ConfigBuilder) BuildConfig(req ConfigRequest) (map[string]any, error) {
	if req.SpeedChannel <= 0 {
		req.SpeedChannel = 1
	}

	// 验证列定义
	if len(req.Columns) == 0 {
		return nil, errors.New("缺少基准列定义")
	}

	// 提取列名
	columnNames := make([]string, 0, len(req.Columns))
	for _, col := range req.Columns {
		columnNames = append(columnNames, col.Name)
	}

	// 构建 reader
	reader, err := b.buildReader(req, columnNames)
	if err != nil {
		return nil, err
	}

	// 构建 writer
	writer, err := b.buildWriter(req, columnNames)
	if err != nil {
		return nil, err
	}

	// 组装完整的 DataX Job
	job := map[string]any{
		"job": map[string]any{
			"content": []map[string]any{{"reader": reader, "writer": writer}},
			"setting": map[string]any{"speed": map[string]any{"channel": req.SpeedChannel}},
		},
	}

	return job, nil
}

// buildReader 构建 reader 配置
func (b *ConfigBuilder) buildReader(req ConfigRequest, columnNames []string) (map[string]any, error) {
	switch req.InputType {
	case DataSourceMySQL:
		return b.buildMySQLReader(req, columnNames)
	case DataSourceOFS, DataSourceHDFS, DataSourceCOSN:
		return b.buildFSReader(req, columnNames)
	default:
		return nil, errors.New("不支持的输入类型")
	}
}

// buildWriter 构建 writer 配置
func (b *ConfigBuilder) buildWriter(req ConfigRequest, columnNames []string) (map[string]any, error) {
	switch req.OutputType {
	case DataSourceMySQL:
		return b.buildMySQLWriter(req, columnNames)
	case DataSourceOFS, DataSourceHDFS, DataSourceCOSN:
		return b.buildFSWriter(req, columnNames)
	default:
		return nil, errors.New("不支持的输出类型")
	}
}

// buildMySQLReader 构建 MySQL Reader
func (b *ConfigBuilder) buildMySQLReader(req ConfigRequest, columnNames []string) (map[string]any, error) {
	if req.Input.MySQL == nil {
		return nil, errors.New("缺少输入 MySQL 配置")
	}

	conn, err := GetMySQLConnection(b.db, req.Input.MySQL.SourceID)
	if err != nil {
		return nil, err
	}

	param := map[string]any{
		"username": conn.User,
		"password": conn.Pass,
		"column":   columnNames,
		"connection": []map[string]any{{
			"table": []string{req.Input.MySQL.Table},
			"jdbcUrl": []string{
				fmt.Sprintf("jdbc:mysql://%s/%s?useUnicode=true&characterEncoding=utf8", conn.Host, conn.DB),
			},
		}},
	}

	if strings.TrimSpace(req.MySQLWhere) != "" {
		param["where"] = req.MySQLWhere
	}

	return map[string]any{"name": "mysqlreader", "parameter": param}, nil
}

// buildFSReader 构建文件系统 Reader
func (b *ConfigBuilder) buildFSReader(req ConfigRequest, columnNames []string) (map[string]any, error) {
	if req.Input.FS == nil {
		return nil, errors.New("缺少输入文件系统配置")
	}

	conn, err := GetFSConnection(b.db, req.Input.FS.FSID)
	if err != nil {
		return nil, err
	}

	fileType := req.Input.FS.FileType
	if fileType == "" {
		fileType = FileFormatORC
	}

	param := map[string]any{
		"defaultFS": conn.DefaultFS,
		"path":      req.Input.FS.Path,
		"fileType":  fileType,
	}

	// 只有当hadoopConfig不为空时才添加
	if conn.HadoopConfig != nil && len(conn.HadoopConfig) > 0 {
		param["hadoopConfig"] = conn.HadoopConfig
	}

	// 添加filename字段（如果指定）
	if req.Input.FS.Filename != nil && *req.Input.FS.Filename != "" {
		param["fileName"] = *req.Input.FS.Filename
	}

	// 根据文件类型设置列配置
	switch fileType {
	case FileFormatText:
		delimiter, err := b.getFieldDelimiter(req.Input.FS.FieldDelimiter)
		if err != nil {
			return nil, err
		}
		param["fieldDelimiter"] = delimiter
		param["column"] = b.buildTextColumns(req.Input.FS.Indexes, req.Columns)
	case FileFormatORC, FileFormatParquet:
		param["column"] = b.buildIndexColumns(req.Input.FS.Indexes, req.Columns)
	default:
		return nil, errors.New("不支持的文件类型")
	}

	return map[string]any{"name": "hdfsreader", "parameter": param}, nil
}

// buildMySQLWriter 构建 MySQL Writer
func (b *ConfigBuilder) buildMySQLWriter(req ConfigRequest, columnNames []string) (map[string]any, error) {
	if req.Output.MySQL == nil {
		return nil, errors.New("缺少输出 MySQL 配置")
	}

	conn, err := GetMySQLConnection(b.db, req.Output.MySQL.TargetID)
	if err != nil {
		return nil, err
	}

	param := map[string]any{
		"username":  conn.User,
		"password":  conn.Pass,
		"column":    columnNames,
		"writeMode": "insert",
		"connection": []map[string]any{{
			"table":   []string{req.Output.MySQL.Table},
			"jdbcUrl": fmt.Sprintf("jdbc:mysql://%s/%s?useUnicode=true&characterEncoding=utf8", conn.Host, conn.DB),
		}},
	}

	return map[string]any{"name": "mysqlwriter", "parameter": param}, nil
}

// buildFSWriter 构建文件系统 Writer
func (b *ConfigBuilder) buildFSWriter(req ConfigRequest, columnNames []string) (map[string]any, error) {
	if req.Output.FS == nil {
		return nil, errors.New("缺少输出文件系统配置")
	}

	conn, err := GetFSConnection(b.db, req.Output.FS.FSID)
	if err != nil {
		return nil, err
	}

	fileType := req.Output.FS.FileType
	if fileType == "" {
		fileType = FileFormatORC
	}

	// 设置默认写入模式
	writeMode := req.Output.FS.WriteMode
	if writeMode == "" {
		writeMode = WriteModeNonConflict // 默认为nonConflict
	}

	param := map[string]any{
		"defaultFS": conn.DefaultFS,
		"path":      req.Output.FS.Path,
		"fileType":  fileType,
		"writeMode": string(writeMode),
		"column":    b.buildOutputColumns(req.Columns),
	}

	// 只有当hadoopConfig不为空时才添加
	if conn.HadoopConfig != nil && len(conn.HadoopConfig) > 0 {
		param["hadoopConfig"] = conn.HadoopConfig
	}

	// 添加filename字段（如果指定）
	if req.Output.FS.Filename != nil && *req.Output.FS.Filename != "" {
		param["fileName"] = *req.Output.FS.Filename
	}

	// 文件系统类型时fieldDelimiter是必填的
	delimiter, err := b.getFieldDelimiter(req.Output.FS.FieldDelimiter)
	if err != nil {
		return nil, err
	}
	param["fieldDelimiter"] = delimiter

	return map[string]any{"name": "hdfswriter", "parameter": param}, nil
}

// buildTextColumns 构建文本文件列配置
func (b *ConfigBuilder) buildTextColumns(indexes []int, columns []Column) []map[string]any {
	if len(indexes) == 0 {
		// 自动生成索引
		for i := range columns {
			indexes = append(indexes, i)
		}
	}

	if len(indexes) != len(columns) {
		return nil
	}

	result := make([]map[string]any, 0, len(columns))
	for i, col := range columns {
		result = append(result, map[string]any{
			"index": indexes[i],
			"type":  mapMySQLToDataX(col.DataType),
		})
	}
	return result
}

// buildIndexColumns 构建索引列配置
func (b *ConfigBuilder) buildIndexColumns(indexes []int, columns []Column) []map[string]int {
	if len(indexes) == 0 {
		// 自动生成索引
		for i := range columns {
			indexes = append(indexes, i)
		}
	}

	result := make([]map[string]int, 0, len(indexes))
	for _, idx := range indexes {
		result = append(result, map[string]int{"index": idx})
	}
	return result
}

// buildOutputColumns 构建输出列配置
func (b *ConfigBuilder) buildOutputColumns(columns []Column) []map[string]string {
	result := make([]map[string]string, 0, len(columns))
	for _, col := range columns {
		result = append(result, map[string]string{
			"name": col.Name,
			"type": mapMySQLToDataX(col.DataType),
		})
	}
	return result
}

// getFieldDelimiter 获取字段分隔符
// 文件系统类型时fieldDelimiter是必填的
func (b *ConfigBuilder) getFieldDelimiter(delimiter *string) (string, error) {
	if delimiter != nil && *delimiter != "" {
		return *delimiter, nil
	}
	return "", errors.New("fieldDelimiter is required for file system")
}

// MarshalJSON 格式化 JSON
func (b *ConfigBuilder) MarshalJSON(job map[string]any) (string, error) {
	bs, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bs), nil
}
