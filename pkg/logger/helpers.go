package logger

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// FormatBytes 将字节数格式化为人类可读的字符串（如 "1.5 KB"、"2.3 MB"）。
//
// 使用 1024 为单位换算，支持 B、KB、MB、GB、TB、PB、EB。
// 常用于日志中输出文件大小、网络传输量等信息。
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

// LogError 记录错误日志并返回原始错误，适用于同时需要记录和返回错误的场景。
//
// ctx 可以是 [context.Context]（从中提取 logger）或其他类型（使用全局 logger）。
// 错误会作为 "error" 字段自动添加到日志属性中。
//
// 示例：
//
//	return logger.LogError(ctx, "数据库查询失败", err, "table", "users")
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

// LogAndWrap 记录错误日志并返回带有上下文信息的包装错误。
//
// 与 [LogError] 不同，该函数使用 fmt.Errorf 的 %w 动词包装原始错误，
// 使得错误链可以通过 [errors.Is] 和 [errors.As] 追溯。
//
// 示例：
//
//	return logger.LogAndWrap("保存配置失败", err, "path", configPath)
//	// 返回错误: "保存配置失败: original error"
func LogAndWrap(msg string, err error, attrs ...any) error {
	allAttrs := append([]any{"error", err}, attrs...)
	slog.Error(msg, allAttrs...)
	return fmt.Errorf("%s: %w", msg, err)
}

// Debug 记录调试级别的结构化日志
//
// 用于输出详细的调试信息，生产环境通常不会开启。
// attrs 支持键值对形式的结构化字段，例如：
//
//	logger.Debug("处理请求", "user_id", 123, "action", "login")
func Debug(msg string, attrs ...any) {
	slog.Debug(msg, attrs...)
}

// Info 记录信息级别的结构化日志
//
// 用于记录应用的正常运行状态和重要事件。
// 这是默认的日志级别，适合记录业务流程中的关键节点。
//
//	logger.Info("用户登录成功", "user_id", 123, "ip", "192.168.1.1")
func Info(msg string, attrs ...any) {
	slog.Info(msg, attrs...)
}

// Warn 记录警告级别的结构化日志
//
// 用于记录可能存在问题但不影响程序继续运行的情况。
// 例如：资源即将耗尽、配置不推荐、性能下降等。
//
//	logger.Warn("连接池使用率过高", "usage", 0.95, "max", 100)
func Warn(msg string, attrs ...any) {
	slog.Warn(msg, attrs...)
}

// Error 记录错误级别的结构化日志
//
// 用于记录程序运行中遇到的错误，但不一定导致程序终止。
// 建议同时记录错误对象和相关上下文信息。
//
//	logger.Error("数据库连接失败", "error", err, "host", dbHost)
func Error(msg string, attrs ...any) {
	slog.Error(msg, attrs...)
}

// shanghaiTimezone 是上海时区的固定偏移（UTC+8）。
//
// 作为 [loadTimezone] 的最终后备方案，确保在没有系统时区数据库时仍能正常工作。
var shanghaiTimezone = time.FixedZone("CST", 8*3600)

// loadTimezone 根据时区标识加载 [time.Location]。
//
// 支持的格式：
//   - IANA 时区名称: "Asia/Shanghai"、"America/New_York"
//   - 固定偏移格式: "+08:00"、"-05:00"、"+0800"
//
// 加载策略（按优先级）：
//  1. 尝试 time.LoadLocation（依赖系统时区数据库）
//  2. 解析固定偏移格式
//  3. 查找常用时区的固定偏移映射
//  4. 返回上海时区 (UTC+8) 作为最终后备
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

// parseFixedOffset 解析固定偏移格式的时区字符串，返回对应的 [time.Location]。
//
// 支持格式: "+08:00"、"-05:00"、"+0800"、"-0500"。
// 解析失败时返回 nil。
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

// knownTimezoneOffset 返回常用时区名称对应的固定偏移 [time.Location]。
//
// 当系统缺少时区数据库（如精简容器镜像）时，为常用时区提供后备方案。
// 注意：固定偏移不支持夏令时，欧美时区在夏季可能有 1 小时偏差。
// 未知时区返回 nil。
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
