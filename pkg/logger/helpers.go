package logger

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// FormatBytes 格式化字节数为人类可读格式
//
// 用于日志中输出文件大小、传输速率等信息
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// LogError 记录错误日志并返回错误
//
// 这是一个便捷函数，用于在需要同时记录日志和返回错误的场景：
//   return logger.LogError(ctx, "操作失败", err, "user_id", userID)
func LogError(ctx interface{}, msg string, err error, attrs ...any) error {
	var logger *slog.Logger

	// 尝试从 context 获取 logger
	if c, ok := ctx.(interface{ Value(key any) any }); ok {
		if l, ok := c.Value(loggerKey).(*slog.Logger); ok {
			logger = l
		}
	}

	// 如果没有从 context 获取到，使用默认 logger
	if logger == nil {
		logger = slog.Default()
	}

	// 合并错误到属性中
	allAttrs := append([]any{"error", err}, attrs...)
	logger.Error(msg, allAttrs...)

	return err
}

// LogAndWrap 记录错误日志并包装错误信息
//
// 用于在错误传播链中添加上下文信息
func LogAndWrap(msg string, err error, attrs ...any) error {
	allAttrs := append([]any{"error", err}, attrs...)
	slog.Error(msg, allAttrs...)
	return fmt.Errorf("%s: %w", msg, err)
}

// Debug 结构化调试日志的快捷方式
func Debug(msg string, attrs ...any) {
	slog.Debug(msg, attrs...)
}

// Info 结构化信息日志的快捷方式
func Info(msg string, attrs ...any) {
	slog.Info(msg, attrs...)
}

// Warn 结构化警告日志的快捷方式
func Warn(msg string, attrs ...any) {
	slog.Warn(msg, attrs...)
}

// Error 结构化错误日志的快捷方式
func Error(msg string, attrs ...any) {
	slog.Error(msg, attrs...)
}

// 上海时区固定偏移（UTC+8），用于 time.LoadLocation 失败时的后备方案
var shanghaiTimezone = time.FixedZone("CST", 8*3600)

// loadTimezone 加载时区，支持以下格式：
//   - IANA 时区名称: "Asia/Shanghai", "America/New_York"
//   - 固定偏移: "+08:00", "-05:00", "+0800"
//
// 如果加载失败或为空，默认返回上海时区 (UTC+8)
func loadTimezone(timezone string) *time.Location {
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}

	// 首先尝试 time.LoadLocation（依赖系统时区数据）
	if loc, err := time.LoadLocation(timezone); err == nil {
		return loc
	}

	// 尝试解析固定偏移格式（如 "+08:00", "-05:00", "+0800"）
	if loc := parseFixedOffset(timezone); loc != nil {
		return loc
	}

	// 对于已知的时区名称，使用固定偏移作为后备
	if loc := knownTimezoneOffset(timezone); loc != nil {
		return loc
	}

	// 最终后备：上海时区
	return shanghaiTimezone
}

// parseFixedOffset 解析固定偏移格式的时区字符串
// 支持格式: "+08:00", "-05:00", "+0800", "-0500"
func parseFixedOffset(s string) *time.Location {
	if len(s) < 5 {
		return nil
	}

	sign := 1
	if s[0] == '-' {
		sign = -1
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	} else {
		return nil
	}

	var hours, minutes int

	// 解析 "08:00" 或 "0800" 格式
	if len(s) == 5 && s[2] == ':' {
		// "08:00" 格式
		hours = int(s[0]-'0')*10 + int(s[1]-'0')
		minutes = int(s[3]-'0')*10 + int(s[4]-'0')
	} else if len(s) == 4 {
		// "0800" 格式
		hours = int(s[0]-'0')*10 + int(s[1]-'0')
		minutes = int(s[2]-'0')*10 + int(s[3]-'0')
	} else {
		return nil
	}

	if hours > 14 || minutes > 59 {
		return nil
	}

	offset := sign * (hours*3600 + minutes*60)
	name := fmt.Sprintf("UTC%+03d:%02d", sign*hours, minutes)
	return time.FixedZone(name, offset)
}

// knownTimezoneOffset 返回已知时区的固定偏移
// 当系统没有时区数据库时，提供常用时区的后备方案
func knownTimezoneOffset(timezone string) *time.Location {
	offsets := map[string]int{
		// 亚洲
		"Asia/Shanghai":    8 * 3600,
		"Asia/Hong_Kong":   8 * 3600,
		"Asia/Taipei":      8 * 3600,
		"Asia/Singapore":   8 * 3600,
		"Asia/Tokyo":       9 * 3600,
		"Asia/Seoul":       9 * 3600,
		// 欧洲（标准时间，不考虑夏令时）
		"Europe/London":    0,
		"Europe/Paris":     1 * 3600,
		"Europe/Berlin":    1 * 3600,
		// 美洲（标准时间）
		"America/New_York":    -5 * 3600,
		"America/Los_Angeles": -8 * 3600,
		// UTC
		"UTC": 0,
	}

	if offset, ok := offsets[timezone]; ok {
		return time.FixedZone(timezone, offset)
	}
	return nil
}

// clipWorkspacePath 裁剪 /workspace/xxx/ 前缀
//
// 当路径包含 /workspace/ 时，去掉 /workspace/ 及其后一级目录
// 例如：/workspace/251127-ai-agent-hatch/main.go:146 -> main.go:146
//
// 这在容器化或沙盒环境中很有用，可以使日志中的源代码位置更简洁
func clipWorkspacePath(path string) string {
	const workspacePrefix = "/workspace/"
	idx := strings.Index(path, workspacePrefix)
	if idx == -1 {
		return path
	}

	// 找到 /workspace/ 后面的部分
	rest := path[idx+len(workspacePrefix):]

	// 找到下一个 /，跳过项目目录名
	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		return path
	}

	// 返回项目目录后面的部分
	return rest[slashIdx+1:]
}
