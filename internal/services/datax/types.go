package datax

// DataX 支持的数据源类型
type DataSourceType string

const (
	DataSourceMySQL DataSourceType = "mysql"
	DataSourceOFS   DataSourceType = "ofs"
	DataSourceHDFS  DataSourceType = "hdfs"
	DataSourceCOSN  DataSourceType = "cosn"
)

// 支持的文件格式
type FileFormat string

const (
	FileFormatORC     FileFormat = "orc"
	FileFormatParquet FileFormat = "parquet"
	FileFormatText    FileFormat = "text"
)

// 列定义
type Column struct {
	Name     string `json:"name"`      // 列名
	DataType string `json:"data_type"` // 数据类型
}

// DataX 配置请求
type ConfigRequest struct {
	InputType    DataSourceType `json:"inType"`       // 输入数据源类型
	OutputType   DataSourceType `json:"outType"`      // 输出数据源类型
	MySQLBase    string         `json:"mysqlBase"`    // 基准 MySQL 端 ("in" 或 "out")
	MySQLWhere   string         `json:"mysqlWhere"`   // MySQL WHERE 条件
	Columns      []Column       `json:"columns"`      // 基准列定义
	SpeedChannel int            `json:"speedChannel"` // 并发通道数

	Input struct {
		MySQL *MySQLConfig `json:"mysql,omitempty"`
		FS    *FSConfig    `json:"fs,omitempty"`
	} `json:"in"`

	Output struct {
		MySQL *MySQLConfig `json:"mysql,omitempty"`
		FS    *FSConfig    `json:"fs,omitempty"`
	} `json:"out"`
}

// MySQL 配置
type MySQLConfig struct {
	SourceID int    `json:"source_id"`
	TargetID int    `json:"target_id"`
	Table    string `json:"table"`
}

// 文件系统配置
type FSConfig struct {
	FSID           int        `json:"fs_id"`
	FileType       FileFormat `json:"fileType"`
	Path           string     `json:"path"`
	Indexes        []int      `json:"indexes"`
	FieldDelimiter *string    `json:"fieldDelimiter,omitempty"`
}

// 数据源连接配置
type DataSourceConfig struct {
	MySQL *MySQLConnection `json:"mysql,omitempty"`
	FS    *FSConnection    `json:"fs,omitempty"`
}

// MySQL 连接配置
type MySQLConnection struct {
	Host string
	User string
	Pass string
	DB   string
}

// 文件系统连接配置
type FSConnection struct {
	DefaultFS    string
	HadoopConfig map[string]string
}

// DataX Job 元数据
type JobMetadata struct {
	ReaderType string
	WriterType string
	TableName  string
	SourcePath string
	TargetPath string
}

// 配置生成响应
type ConfigResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Message string                 `json:"message,omitempty"`
}
