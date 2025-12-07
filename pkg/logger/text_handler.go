package logger

import (
	"io"
	"log/slog"
	"time"
)

// newTextHandler 创建并返回一个文本格式的 [slog.TextHandler]。
//
// 通过 ReplaceAttr 钩子实现自定义时间格式，其他行为与标准 TextHandler 一致。
// timeFormat 支持: rfc3339, rfc3339ms, time, timems, datetime（默认）或自定义 Go 时间格式。
// timezone 为 IANA 时区名或固定偏移（如 "+08:00"），默认 Asia/Shanghai。
func newTextHandler(w io.Writer, opts *slog.HandlerOptions, timeFormat string, timezone string) *slog.TextHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	if timeFormat == "" {
		timeFormat = "datetime"
	}

	// 加载时区，默认使用上海时区 (UTC+8)
	loc := loadTimezone(timezone)

	// 使用 ReplaceAttr 来自定义时间格式
	originalReplace := opts.ReplaceAttr
	opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		// 先执行原有的 ReplaceAttr（如果有）
		if originalReplace != nil {
			a = originalReplace(groups, a)
		}

		// 只处理顶级的 time 字段
		if len(groups) == 0 && a.Key == slog.TimeKey {
			if t, ok := a.Value.Any().(time.Time); ok {
				// 转换到指定时区
				if loc != nil {
					t = t.In(loc)
				}
				// 格式化时间
				formatted := formatTimeString(t, timeFormat)
				return slog.String(slog.TimeKey, formatted)
			}
		}
		return a
	}

	return slog.NewTextHandler(w, opts)
}

// formatTimeString 将时间按指定格式转换为字符串。
//
// 支持的预定义格式：
//   - "rfc3339":    RFC3339 格式（秒精度）
//   - "rfc3339ms":  RFC3339 格式（毫秒精度）
//   - "time":       仅时间 15:04:05
//   - "timems":     仅时间（毫秒）15:04:05.000
//   - "datetime":   日期时间（默认）2006-01-02 15:04:05
//
// 其他值作为自定义 Go 时间格式字符串处理。
func formatTimeString(t time.Time, timeFormat string) string {
	switch timeFormat {
	case "rfc3339":
		return t.Format(time.RFC3339)
	case "rfc3339ms":
		return t.Format("2006-01-02T15:04:05.000Z07:00")
	case "time":
		return t.Format("15:04:05")
	case "timems":
		return t.Format("15:04:05.000")
	case "datetime", "":
		// 默认格式：日期时间（秒精度）
		return t.Format("2006-01-02 15:04:05")
	default:
		// 自定义格式
		return t.Format(timeFormat)
	}
}
