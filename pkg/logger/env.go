package logger

import (
	"os"
	"strings"
)

// InitFromEnv 从环境变量初始化全局日志系统
//
// 这是初始化日志的便捷方式，适用于通过环境变量配置应用的场景（如容器化部署）。
//
// 支持的环境变量：
//   - LOG_LEVEL: 日志级别 (DEBUG, INFO, WARN, ERROR)，默认 INFO
//   - LOG_FORMAT: 输出格式 (json, text, color/colored)，默认 color
//   - LOG_OUTPUT: 输出目标 (stdout, stderr, 或文件路径)，默认 stdout
//   - LOG_ADD_SOURCE: 是否添加源代码位置 (true, false)，默认 true
//   - LOG_TIME_FORMAT: 时间格式，默认 datetime
//
// 时间格式可选值：datetime, rfc3339, rfc3339ms, unix, unixms, unixfloat
//
// 使用示例：
//
//	func main() {
//	    if err := logger.InitFromEnv(); err != nil {
//	        log.Fatalf("初始化日志失败: %v", err)
//	    }
//	    defer logger.Close()
//	    // ...
//	}
func InitFromEnv() error {
	cfg := &Config{
		Level:      getEnv("LOG_LEVEL", "INFO"),
		Format:     getEnv("LOG_FORMAT", "color"), // 默认使用彩色输出
		Output:     getEnv("LOG_OUTPUT", "stdout"),
		AddSource:  getEnvBool("LOG_ADD_SOURCE", true), // 默认启用源代码位置
		TimeFormat: getEnv("LOG_TIME_FORMAT", "datetime"),
	}

	return Init(cfg)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
}
