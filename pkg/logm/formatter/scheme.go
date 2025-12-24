package formatter

import "log/slog"

// ANSI 颜色代码
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

// ColorScheme 颜色配置方案
type ColorScheme struct {
	Time   string // 时间颜色
	Debug  string // DEBUG 级别
	Info   string // INFO 级别
	Warn   string // WARN 级别
	Error  string // ERROR 级别
	Key    string // 属性键
	String string // 字符串值
	Number string // 数字值
	Source string // 源代码位置
	Null   string // null 值
}

// DefaultScheme 默认配色方案
func DefaultScheme() *ColorScheme {
	return &ColorScheme{
		Time:   ColorGray,
		Debug:  ColorCyan,
		Info:   ColorGreen,
		Warn:   ColorYellow,
		Error:  ColorRed,
		Key:    ColorCyan,
		String: ColorGreen,
		Number: ColorYellow,
		Source: ColorGray,
		Null:   ColorGray,
	}
}

// LevelColor 返回级别对应颜色
func (s *ColorScheme) LevelColor(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return s.Debug
	case level < slog.LevelWarn:
		return s.Info
	case level < slog.LevelError:
		return s.Warn
	default:
		return s.Error
	}
}
