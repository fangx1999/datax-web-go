package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
	DataX    DataXConfig    `yaml:"datax"`
	Session  SessionConfig  `yaml:"session"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// DataXConfig DataX配置
type DataXConfig struct {
	Home    string `yaml:"home"`
	TempDir string `yaml:"temp_dir"`
}

// SessionConfig 会话配置
type SessionConfig struct {
	Key string `yaml:"key"`
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	setDefaults(&config)

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults(config *Config) {
	if config.Server.Port == "" {
		config.Server.Port = "8000"
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Database.Host == "" {
		config.Database.Host = "127.0.0.1"
	}
	if config.Database.Port == "" {
		config.Database.Port = "3306"
	}
	if config.Database.Name == "" {
		config.Database.Name = "datax_web"
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "INFO"
	}
	if config.DataX.Home == "" {
		config.DataX.Home = "/opt/datax"
	}
	if config.DataX.TempDir == "" {
		config.DataX.TempDir = "/tmp/datax-web"
	}
	if config.Session.Key == "" {
		config.Session.Key = "default-session-key-change-in-production"
	}
}

// GetDSN 获取数据库连接字符串
func (c *Config) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Asia%%2FShanghai",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
	)
}

// GetServerAddr 获取服务器地址
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}
