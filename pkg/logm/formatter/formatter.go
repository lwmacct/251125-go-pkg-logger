// Package formatter 提供日志格式化器实现。
//
// 格式化器决定日志的输出格式，内置三种格式：
//   - JSON: 结构化 JSON 输出，适合生产环境日志采集
//   - Text: 键值对文本输出，兼容传统日志分析工具
//   - Color: 彩色终端输出，适合开发环境
package formatter

import (
	"log/slog"
	"time"
)

// Record 日志记录，Formatter 的输入。
type Record struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Attrs   []slog.Attr
	Source  *slog.Source
	Groups  []string
}

// Formatter 格式化接口。
type Formatter interface {
	Format(r *Record) ([]byte, error)
}

// Options 格式化器通用选项
type Options struct {
	TimeFormat  string
	Location    *time.Location
	SourceClip  string          // Source 路径裁剪前缀 (如 "/workspace/")
	SourceDepth int             // Source 路径保留层数 (默认 3)
	ColorScheme *ColorScheme    // 颜色配置方案
	EnableColor bool            // 启用颜色输出
	RawFields   map[string]bool // 不加引号直接输出的字段名集合
}

// Option 选项函数
type Option func(*Options)

// defaultOptions 返回默认选项
func defaultOptions() *Options {
	return &Options{
		TimeFormat:  "datetime",
		Location:    time.Local,
		ColorScheme: DefaultScheme(),
		EnableColor: true,
	}
}

// WithTimeFormat 设置时间格式
func WithTimeFormat(format string) Option {
	return func(o *Options) {
		o.TimeFormat = format
	}
}

// WithTimezone 设置时区
func WithTimezone(tz string) Option {
	return func(o *Options) {
		o.Location = loadTimezone(tz)
	}
}

// WithSourceClip 设置 Source 路径裁剪前缀
func WithSourceClip(prefix string) Option {
	return func(o *Options) {
		o.SourceClip = prefix
	}
}

// WithSourceDepth 设置 Source 路径保留层数
func WithSourceDepth(depth int) Option {
	return func(o *Options) {
		o.SourceDepth = depth
	}
}

// WithColor 启用/禁用颜色输出
func WithColor(enable bool) Option {
	return func(o *Options) {
		o.EnableColor = enable
	}
}

// WithColorScheme 设置颜色配置方案
func WithColorScheme(scheme *ColorScheme) Option {
	return func(o *Options) {
		o.ColorScheme = scheme
	}
}

// WithRawFields 设置不加引号直接输出的字段。
//
// 指定的字段值将直接输出，不进行引号包裹和转义。
// 适用于 SQL 语句等包含特殊字符但需要原样显示的场景。
//
// 示例：
//
//	formatter.Text(formatter.WithRawFields("sql", "query"))
//	// sql=SELECT * FROM "users"  而不是 sql="SELECT * FROM \"users\""
func WithRawFields(fields ...string) Option {
	return func(o *Options) {
		if o.RawFields == nil {
			o.RawFields = make(map[string]bool)
		}
		for _, f := range fields {
			o.RawFields[f] = true
		}
	}
}

// formatTime 根据格式字符串格式化时间
func formatTime(t time.Time, format string) string {
	switch format {
	case "time":
		return t.Format("15:04:05")
	case "timems":
		return t.Format("15:04:05.000")
	case "datetime":
		return t.Format("2006-01-02 15:04:05")
	case "rfc3339":
		return t.Format(time.RFC3339)
	case "rfc3339ms":
		return t.Format("2006-01-02T15:04:05.000Z07:00")
	default:
		if format == "" {
			return t.Format("2006-01-02 15:04:05")
		}
		return t.Format(format)
	}
}

// loadTimezone 加载时区
func loadTimezone(tz string) *time.Location {
	if tz == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Local
	}
	return loc
}

// 确保所有格式化器实现 Formatter 接口
var (
	_ Formatter = (*JSONFormatter)(nil)
	_ Formatter = (*TextFormatter)(nil)
	_ Formatter = (*ColorTextFormatter)(nil)
	_ Formatter = (*ColorJSONFormatter)(nil)
)
