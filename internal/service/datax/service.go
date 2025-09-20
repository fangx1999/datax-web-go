package datax

import (
	"database/sql"
	"github.com/gin-gonic/gin"
)

// Service DataX 配置服务
type Service struct {
	validator *Validator
	builder   *ConfigBuilder
}

// NewService 创建 DataX 配置服务
func NewService(db *sql.DB) *Service {
	return &Service{
		validator: NewValidator(),
		builder:   NewConfigBuilder(db),
	}
}

// GenerateConfig 生成 DataX 配置
func (s *Service) GenerateConfig(req ConfigRequest) ConfigResponse {
	// 验证请求
	if err := s.validator.ValidateConfigRequest(req); err != nil {
		if ve, ok := IsValidationError(err); ok {
			return ConfigResponse{
				Success: false,
				Error:   ve.Message,
			}
		}
		return ConfigResponse{
			Success: false,
			Error:   "验证失败: " + err.Error(),
		}
	}

	// 构建配置
	job, err := s.builder.BuildConfig(req)
	if err != nil {
		return ConfigResponse{
			Success: false,
			Error:   "构建失败: " + err.Error(),
		}
	}

	// 返回成功结果
	return ConfigResponse{
		Success: true,
		Data: gin.H{
			"json":    job,
			"message": "配置生成成功",
		},
		Message: "配置生成成功",
	}
}
