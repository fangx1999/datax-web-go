package util

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PathValidator 路径校验器
type PathValidator struct {
	hadoopCmd string // hadoop命令路径，默认为"hadoop"
}

// NewPathValidator 创建路径校验器
func NewPathValidator() *PathValidator {
	return &PathValidator{
		hadoopCmd: "hadoop", // 默认使用hadoop命令
	}
}

// SetHadoopCmd 设置hadoop命令路径
func (pv *PathValidator) SetHadoopCmd(cmd string) {
	pv.hadoopCmd = cmd
}

// ValidateAndCreatePath 验证路径是否存在，不存在则创建
func (pv *PathValidator) ValidateAndCreatePath(fsType, path string) error {
	// 检查路径是否存在
	exists, err := pv.checkPathExists(path)
	if err != nil {
		return fmt.Errorf("检查路径失败: %v", err)
	}

	if !exists {
		// 路径不存在，创建目录
		if err := pv.createPath(path); err != nil {
			return fmt.Errorf("创建路径失败: %v", err)
		}
	}

	return nil
}

// checkPathExists 检查路径是否存在
func (pv *PathValidator) checkPathExists(path string) (bool, error) {
	// 使用hadoop fs -test -e命令检查路径是否存在
	cmd := exec.Command(pv.hadoopCmd, "fs", "-test", "-e", path)
	err := cmd.Run()

	if err != nil {
		// 如果命令返回非零退出码，说明路径不存在
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				return false, nil // 路径不存在
			}
		}
		return false, err
	}

	return true, nil
}

// createPath 创建路径
func (pv *PathValidator) createPath(path string) error {
	// 使用hadoop fs -mkdir -p命令创建目录
	cmd := exec.Command(pv.hadoopCmd, "fs", "-mkdir", "-p", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hadoop命令执行失败: %v, 输出: %s", err, string(output))
	}

	return nil
}

// ValidateDataXConfigPaths 验证DataX配置中的所有路径
func (pv *PathValidator) ValidateDataXConfigPaths(configJSON string) error {
	// 解析JSON配置
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("解析JSON配置失败: %v", err)
	}

	// 提取所有路径并验证
	paths := pv.extractPathsFromConfig(config)
	for _, path := range paths {
		if err := pv.ValidateAndCreatePath("", path); err != nil {
			return fmt.Errorf("验证路径失败 %s: %v", path, err)
		}
	}

	return nil
}

// extractPathsFromConfig 从配置中提取所有路径
func (pv *PathValidator) extractPathsFromConfig(config map[string]interface{}) []string {
	var paths []string

	// 获取job配置
	job, ok := config["job"].(map[string]interface{})
	if !ok {
		return paths
	}

	// 获取content数组
	content, ok := job["content"].([]interface{})
	if !ok || len(content) == 0 {
		return paths
	}

	// 遍历content中的每个配置
	for _, item := range content {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// 检查reader路径
		if reader, ok := itemMap["reader"].(map[string]interface{}); ok {
			if path := pv.extractPathFromReader(reader); path != "" {
				paths = append(paths, path)
			}
		}

		// 检查writer路径
		if writer, ok := itemMap["writer"].(map[string]interface{}); ok {
			if path := pv.extractPathFromWriter(writer); path != "" {
				paths = append(paths, path)
			}
		}
	}

	return paths
}

// extractPathFromReader 从reader配置中提取路径
func (pv *PathValidator) extractPathFromReader(reader map[string]interface{}) string {
	// 检查是否是文件系统类型（非MySQL）
	if readerType, ok := reader["name"].(string); ok && readerType != "mysqlreader" {
		if param, ok := reader["parameter"].(map[string]interface{}); ok {
			if path, exists := param["path"]; exists {
				if pathStr, ok := path.(string); ok && pathStr != "" {
					return pathStr
				}
			}
		}
	}
	return ""
}

// extractPathFromWriter 从writer配置中提取路径
func (pv *PathValidator) extractPathFromWriter(writer map[string]interface{}) string {
	// 检查是否是文件系统类型（非MySQL）
	if writerType, ok := writer["name"].(string); ok && writerType != "mysqlwriter" {
		if param, ok := writer["parameter"].(map[string]interface{}); ok {
			if path, exists := param["path"]; exists {
				if pathStr, ok := path.(string); ok && pathStr != "" {
					return pathStr
				}
			}
		}
	}
	return ""
}

// ProcessDatePlaceholders 处理配置中的日期占位符
func ProcessDatePlaceholders(config string, executionDate ...time.Time) string {
	var date time.Time
	if len(executionDate) > 0 {
		date = executionDate[0]
	} else {
		// 默认使用前一天
		date = time.Now().AddDate(0, 0, -1)
	}

	// 替换各种日期占位符
	config = strings.ReplaceAll(config, "${yyyy-mm-dd}", date.Format("2006-01-02"))
	config = strings.ReplaceAll(config, "${yyyy_mm_dd}", date.Format("2006_01_02"))
	config = strings.ReplaceAll(config, "${yyyy}", date.Format("2006"))
	config = strings.ReplaceAll(config, "${mm}", date.Format("01"))
	config = strings.ReplaceAll(config, "${dd}", date.Format("02"))
	config = strings.ReplaceAll(config, "${HH}", date.Format("15"))
	config = strings.ReplaceAll(config, "${MM}", date.Format("04"))
	config = strings.ReplaceAll(config, "${SS}", date.Format("05"))

	return config
}
