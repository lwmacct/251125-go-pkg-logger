package logm

import (
	"fmt"
	"log/slog"
	"strings"
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
//	return logm.LogError(ctx, "数据库查询失败", err, "table", "users")
func LogError(ctx any, msg string, err error, attrs ...any) error {
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
//	return logm.LogAndWrap("保存配置失败", err, "path", configPath)
//	// 返回错误: "保存配置失败: original error"
func LogAndWrap(msg string, err error, attrs ...any) error {
	allAttrs := append([]any{"error", err}, attrs...)
	slog.Error(msg, allAttrs...)
	return fmt.Errorf("%s: %w", msg, err)
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
