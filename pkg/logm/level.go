package logm

import (
	"log/slog"
	"strings"
)

// globalLevelVar 全局日志级别变量
var globalLevelVar = &slog.LevelVar{}

func init() {
	globalLevelVar.Set(slog.LevelInfo)
}

// SetLevel 动态设置全局日志级别。
//
// 该函数是线程安全的，修改会立即对所有使用全局 logger 的调用生效。
// 支持: DEBUG, INFO, WARN, ERROR（大小写不敏感）
//
// 示例:
//
//	logm.SetLevel("DEBUG")  // 开启调试日志
//	logm.SetLevel("ERROR")  // 仅显示错误
func SetLevel(level string) {
	globalLevelVar.Set(ParseLevel(level))
}

// GetLevel 获取当前全局日志级别。
func GetLevel() string {
	return globalLevelVar.Level().String()
}

// GetLevelVar 返回底层的 slog.LevelVar。
//
// 高级用法：可用于自定义 Handler 的 Level 配置。
func GetLevelVar() *slog.LevelVar {
	return globalLevelVar
}

// ParseLevel 解析日志级别字符串。
//
// 支持: DEBUG, INFO, WARN, WARNING, ERROR（大小写不敏感）
// 无法识别的级别默认返回 INFO。
func ParseLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LevelString 将 slog.Level 转换为字符串。
func LevelString(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARN"
	default:
		return "ERROR"
	}
}
