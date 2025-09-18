package datax

import (
	"errors"
	"net/http"
)

// ValidationError 验证错误
type ValidationError struct {
	Message    string
	StatusCode int
}

func (e ValidationError) Error() string {
	return e.Message
}

// Validator 配置请求验证器
type Validator struct{}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateConfigRequest 验证配置请求
func (v *Validator) ValidateConfigRequest(req ConfigRequest) error {
	// 基础业务规则验证
	if err := v.validateBasicRules(req); err != nil {
		return err
	}

	// 输入侧验证
	if err := v.validateInput(req); err != nil {
		return err
	}

	// 输出侧验证
	if err := v.validateOutput(req); err != nil {
		return err
	}

	return nil
}

// validateBasicRules 验证基础业务规则
func (v *Validator) validateBasicRules(req ConfigRequest) error {
	// 至少一端必须为 MySQL
	if req.InputType != DataSourceMySQL && req.OutputType != DataSourceMySQL {
		return ValidationError{
			Message:    "输入/输出至少一端必须为 MySQL",
			StatusCode: http.StatusBadRequest,
		}
	}

	// 必须有基准列
	if len(req.Columns) == 0 {
		return ValidationError{
			Message:    "请先加载并勾选基准 MySQL 列",
			StatusCode: http.StatusBadRequest,
		}
	}

	return nil
}

// validateInput 验证输入配置
func (v *Validator) validateInput(req ConfigRequest) error {
	switch req.InputType {
	case DataSourceMySQL:
		if req.Input.MySQL == nil || req.Input.MySQL.SourceID == 0 || req.Input.MySQL.Table == "" {
			return ValidationError{
				Message:    "缺少输入 MySQL 的 source_id/table",
				StatusCode: http.StatusBadRequest,
			}
		}
	case DataSourceOFS, DataSourceHDFS, DataSourceCOSN:
		if req.Input.FS == nil || req.Input.FS.FSID == 0 || req.Input.FS.Path == "" {
			return ValidationError{
				Message:    "缺少输入 FS 的 fs_id/path",
				StatusCode: http.StatusBadRequest,
			}
		}
	default:
		return ValidationError{
			Message:    "未知输入类型",
			StatusCode: http.StatusBadRequest,
		}
	}
	return nil
}

// validateOutput 验证输出配置
func (v *Validator) validateOutput(req ConfigRequest) error {
	switch req.OutputType {
	case DataSourceMySQL:
		if req.Output.MySQL == nil || req.Output.MySQL.TargetID == 0 || req.Output.MySQL.Table == "" {
			return ValidationError{
				Message:    "缺少输出 MySQL 的 target_id/table",
				StatusCode: http.StatusBadRequest,
			}
		}
	case DataSourceOFS, DataSourceHDFS, DataSourceCOSN:
		if req.Output.FS == nil || req.Output.FS.FSID == 0 || req.Output.FS.Path == "" {
			return ValidationError{
				Message:    "缺少输出 FS 的 fs_id/path",
				StatusCode: http.StatusBadRequest,
			}
		}
	default:
		return ValidationError{
			Message:    "未知输出类型",
			StatusCode: http.StatusBadRequest,
		}
	}
	return nil
}

// IsValidationError 检查是否为验证错误
func IsValidationError(err error) (ValidationError, bool) {
	var ve ValidationError
	if errors.As(err, &ve) {
		return ve, true
	}
	return ValidationError{}, false
}
