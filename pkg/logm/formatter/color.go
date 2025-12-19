package formatter

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
)

// ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// ColorFormatter 彩色格式化器。
//
// 输出带 ANSI 颜色的日志，适合终端开发调试。
// 支持 JSON 字符串自动展开和嵌套结构平铺。
type ColorFormatter struct {
	opts         *Options
	enableColor  bool
	flattenJSON  bool
	priorityKeys []string
	trailingKeys []string
}

// ColorOption ColorFormatter 特有选项
type ColorOption func(*ColorFormatter)

// Color 创建彩色格式化器。
func Color(opts ...Option) *ColorFormatter {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return &ColorFormatter{
		opts:         o,
		enableColor:  true,
		flattenJSON:  true,
		priorityKeys: []string{"time", "level", "msg"},
		trailingKeys: []string{"source"},
	}
}

// WithColor 启用/禁用颜色
func WithColor(enable bool) ColorOption {
	return func(f *ColorFormatter) {
		f.enableColor = enable
	}
}

// WithFlattenJSON 启用/禁用 JSON 平铺
func WithFlattenJSON(enable bool) ColorOption {
	return func(f *ColorFormatter) {
		f.flattenJSON = enable
	}
}

// Format 实现 Formatter 接口。
func (f *ColorFormatter) Format(r *Record) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	// 时间
	t := r.Time
	if f.opts.Location != nil {
		t = t.In(f.opts.Location)
	}
	f.writeColored(buf, colorGray, formatTime(t, f.opts.TimeFormat))
	buf.WriteByte(' ')

	// 级别（带颜色）
	f.writeLevel(buf, r.Level)
	buf.WriteByte(' ')

	// 消息
	f.writeColored(buf, colorBlue, r.Message)

	// 属性
	f.writeAttrs(buf, r.Attrs, r.Groups)

	// 源代码位置
	if r.Source != nil {
		buf.WriteByte(' ')
		f.writeColored(buf, colorPurple, r.Source.File+":"+strconv.Itoa(r.Source.Line))
	}

	buf.WriteByte('\n')

	return copyBytes(buf.Bytes()), nil
}

// writeLevel 写入级别（带颜色）
func (f *ColorFormatter) writeLevel(buf *bytes.Buffer, level slog.Level) {
	var color, text string
	switch {
	case level < slog.LevelInfo:
		color, text = colorCyan, "DEBUG"
	case level < slog.LevelWarn:
		color, text = colorGreen, "INFO"
	case level < slog.LevelError:
		color, text = colorYellow, "WARN"
	default:
		color, text = colorRed, "ERROR"
	}

	if f.enableColor {
		buf.WriteString(color)
		buf.WriteString(colorBold)
	}
	buf.WriteString(text)
	if f.enableColor {
		buf.WriteString(colorReset)
	}
}

// writeColored 写入带颜色的文本
func (f *ColorFormatter) writeColored(buf *bytes.Buffer, color, text string) {
	if f.enableColor {
		buf.WriteString(color)
	}
	buf.WriteString(text)
	if f.enableColor {
		buf.WriteString(colorReset)
	}
}

// writeAttrs 写入属性
func (f *ColorFormatter) writeAttrs(buf *bytes.Buffer, attrs []slog.Attr, groups []string) {
	prefix := ""
	for _, g := range groups {
		prefix += g + "."
	}

	for _, attr := range attrs {
		if attr.Key == "" {
			continue
		}
		buf.WriteByte(' ')
		f.writeAttr(buf, attr, prefix)
	}
}

// writeAttr 写入单个属性
func (f *ColorFormatter) writeAttr(buf *bytes.Buffer, attr slog.Attr, prefix string) {
	key := prefix + attr.Key
	f.writeColored(buf, colorCyan, key)
	buf.WriteByte('=')
	f.writeValue(buf, attr.Value, key)
}

// writeValue 写入值
func (f *ColorFormatter) writeValue(buf *bytes.Buffer, v slog.Value, keyPath string) {
	v = v.Resolve()

	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		// 尝试展开 JSON 字符串
		if f.flattenJSON && len(s) > 0 && (s[0] == '{' || s[0] == '[') {
			if expanded := f.tryFlattenJSON(s, keyPath); expanded != "" {
				buf.WriteString(expanded)
				return
			}
		}
		f.writeColored(buf, colorGreen, strconv.Quote(s))

	case slog.KindInt64:
		f.writeColored(buf, colorYellow, strconv.FormatInt(v.Int64(), 10))

	case slog.KindUint64:
		f.writeColored(buf, colorYellow, strconv.FormatUint(v.Uint64(), 10))

	case slog.KindFloat64:
		f.writeColored(buf, colorYellow, strconv.FormatFloat(v.Float64(), 'f', -1, 64))

	case slog.KindBool:
		if v.Bool() {
			f.writeColored(buf, colorYellow, "true")
		} else {
			f.writeColored(buf, colorYellow, "false")
		}

	case slog.KindDuration:
		f.writeColored(buf, colorYellow, v.Duration().String())

	case slog.KindTime:
		t := v.Time()
		if f.opts.Location != nil {
			t = t.In(f.opts.Location)
		}
		f.writeColored(buf, colorGreen, strconv.Quote(formatTime(t, f.opts.TimeFormat)))

	case slog.KindGroup:
		// 展开分组为平铺格式
		attrs := v.Group()
		for i, attr := range attrs {
			if i > 0 {
				buf.WriteByte(' ')
			}
			f.writeAttr(buf, attr, keyPath+".")
		}

	case slog.KindAny:
		f.writeAny(buf, v.Any(), keyPath)

	default:
		f.writeColored(buf, colorGreen, strconv.Quote(v.String()))
	}
}

// writeAny 写入任意类型
func (f *ColorFormatter) writeAny(buf *bytes.Buffer, v any, keyPath string) {
	if v == nil {
		f.writeColored(buf, colorGray, "null")
		return
	}

	// 尝试 JSON 序列化后平铺
	if f.flattenJSON {
		data, err := json.Marshal(v)
		if err == nil && len(data) > 0 {
			if expanded := f.tryFlattenJSON(string(data), keyPath); expanded != "" {
				buf.WriteString(expanded)
				return
			}
		}
	}

	// 回退到简单字符串
	data, _ := json.Marshal(v)
	f.writeColored(buf, colorGreen, string(data))
}

// tryFlattenJSON 尝试展开 JSON 为平铺格式
func (f *ColorFormatter) tryFlattenJSON(s string, keyPath string) string {
	var data any
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return ""
	}

	var parts []string
	f.flattenValue(data, keyPath, &parts)
	return strings.Join(parts, " ")
}

// flattenValue 递归展开值
func (f *ColorFormatter) flattenValue(v any, path string, parts *[]string) {
	switch val := v.(type) {
	case map[string]any:
		for k, v := range val {
			f.flattenValue(v, path+"."+k, parts)
		}
	case []any:
		for i, v := range val {
			f.flattenValue(v, path+"["+strconv.Itoa(i)+"]", parts)
		}
	case string:
		*parts = append(*parts, f.coloredKV(path, strconv.Quote(val)))
	case float64:
		*parts = append(*parts, f.coloredKV(path, strconv.FormatFloat(val, 'f', -1, 64)))
	case bool:
		*parts = append(*parts, f.coloredKV(path, strconv.FormatBool(val)))
	case nil:
		*parts = append(*parts, f.coloredKV(path, "null"))
	default:
		data, _ := json.Marshal(val)
		*parts = append(*parts, f.coloredKV(path, string(data)))
	}
}

// coloredKV 生成带颜色的 key=value
func (f *ColorFormatter) coloredKV(key, value string) string {
	if f.enableColor {
		return colorCyan + key + colorReset + "=" + colorGreen + value + colorReset
	}
	return key + "=" + value
}
