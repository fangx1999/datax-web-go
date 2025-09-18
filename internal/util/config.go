package util

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// Config 保存从 YAML 文件加载的所有运行时配置。
//
// 依赖此包的应用程序可以访问数据库连接参数、
// 会话密钥和 DataX 安装目录，而无需硬编码任何值。
// 提供了合理的默认值以简化开发，但在生产环境中
// 应使用配置文件覆盖这些值。
type Config struct {
	DBHost     string `yaml:"db.host"`
	DBPort     string `yaml:"db.port"`
	DBUser     string `yaml:"db.user"`
	DBPass     string `yaml:"db.pass"`
	DBName     string `yaml:"db.name"`
	SessionKey string `yaml:"session_key"`
	Port       string `yaml:"port"`
	DataxHome  string `yaml:"datax_home"`
	TempDir    string `yaml:"temp_dir"`
}

// LoadConfigFromYaml 从 YAML 文件读取配置值。
// 缺失的值将替换为合理的默认值。
func LoadConfigFromYaml(configPath string) *Config {
	// 从YAML文件加载配置
	if configPath != "" {
		cfg, err := loadFromYamlFile(configPath)
		if err == nil {
			return cfg
		}
		fmt.Printf("警告: 无法从YAML加载配置: %v, 将使用默认值", err)
	}

	// 使用默认配置
	return nil
}

// loadFromYamlFile loads configuration from a YAML file
func loadFromYamlFile(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var yamlConfig struct {
		DB struct {
			Host string `yaml:"host"`
			Port string `yaml:"port"`
			User string `yaml:"user"`
			Pass string `yaml:"pass"`
			Name string `yaml:"name"`
		} `yaml:"db"`
		SessionKey string `yaml:"session_key"`
		Port       string `yaml:"port"`
		DataxHome  string `yaml:"datax_home"`
		TempDir    string `yaml:"temp_dir"`
	}

	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return nil, fmt.Errorf("解析YAML配置失败: %w", err)
	}

	cfg := &Config{
		DBHost:     yamlConfig.DB.Host,
		DBPort:     yamlConfig.DB.Port,
		DBUser:     yamlConfig.DB.User,
		DBPass:     yamlConfig.DB.Pass,
		DBName:     yamlConfig.DB.Name,
		SessionKey: yamlConfig.SessionKey,
		Port:       yamlConfig.Port,
		DataxHome:  yamlConfig.DataxHome,
		TempDir:    yamlConfig.TempDir,
	}

	// 使用默认值填充空字段
	if cfg.DBHost == "" {
		cfg.DBHost = "127.0.0.1"
	}
	if cfg.DBPort == "" {
		cfg.DBPort = "3306"
	}
	if cfg.DBUser == "" {
		cfg.DBUser = "root"
	}
	if cfg.DBName == "" {
		cfg.DBName = "datax_web"
	}
	if cfg.SessionKey == "" {
		cfg.SessionKey = "change-me-very-secret"
	}
	if cfg.Port == "" {
		cfg.Port = "8000"
	}
	if cfg.DataxHome == "" {
		cfg.DataxHome = "/opt/datax"
	}
	if cfg.TempDir == "" {
		cfg.TempDir = "/tmp/datax-web"
	}

	return cfg, nil
}
