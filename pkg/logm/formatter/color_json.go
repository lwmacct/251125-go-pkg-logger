package formatter

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strconv"
)

// ColorJSONFormatter 彩色 JSON 格式化器。
//
// 输出带 ANSI 颜色的 JSON 格式日志，适合终端查看。
// JSON 结构便于日志解析，颜色便于人工阅读。
type ColorJSONFormatter struct {
	opts        *Options
	enableColor bool
}

// ColorJSON 创建彩色 JSON 格式化器。
func ColorJSON(opts ...Option) *ColorJSONFormatter {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return &ColorJSONFormatter{
		opts:        o,
		enableColor: true,
	}
}

// DisableColor 禁用颜色输出。
func (f *ColorJSONFormatter) DisableColor() *ColorJSONFormatter {
	f.enableColor = false
	return f
}

// Format 实现 Formatter 接口。
func (f *ColorJSONFormatter) Format(r *Record) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	buf.WriteByte('{')

	// time
	t := r.Time
	if f.opts.Location != nil {
		t = t.In(f.opts.Location)
	}
	f.writeKey(buf, "time", false)
	f.writeColoredString(buf, colorGray, formatTime(t, f.opts.TimeFormat))

	// level
	f.writeKey(buf, "level", true)
	f.writeLevel(buf, r.Level)

	// msg
	f.writeKey(buf, "msg", true)
	f.writeColoredString(buf, colorBlue, r.Message)

	// source
	if r.Source != nil {
		f.writeKey(buf, "source", true)
		f.writeColoredString(buf, colorPurple, FormatSource(r.Source, f.opts))
	}

	// 其他属性
	f.writeAttrs(buf, r.Attrs, r.Groups)

	buf.WriteByte('}')
	buf.WriteByte('\n')

	return copyBytes(buf.Bytes()), nil
}

// writeKey 写入 JSON key
func (f *ColorJSONFormatter) writeKey(buf *bytes.Buffer, key string, comma bool) {
	if comma {
		buf.WriteByte(',')
	}
	buf.WriteByte('"')
	buf.WriteString(key)
	buf.WriteString(`":`)
}

// writeLevel 写入带颜色的级别值
func (f *ColorJSONFormatter) writeLevel(buf *bytes.Buffer, level slog.Level) {
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
	f.writeColoredString(buf, color, text)
}

// writeColoredString 写入带颜色的 JSON 字符串值
func (f *ColorJSONFormatter) writeColoredString(buf *bytes.Buffer, color, s string) {
	buf.WriteByte('"')
	if f.enableColor {
		buf.WriteString(color)
	}
	// 转义 JSON 特殊字符
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				buf.WriteString(`\u00`)
				buf.WriteByte("0123456789abcdef"[r>>4])
				buf.WriteByte("0123456789abcdef"[r&0xf])
			} else {
				buf.WriteRune(r)
			}
		}
	}
	if f.enableColor {
		buf.WriteString(colorReset)
	}
	buf.WriteByte('"')
}

// writeColoredValue 写入带颜色的值（非字符串类型）
func (f *ColorJSONFormatter) writeColoredValue(buf *bytes.Buffer, color, value string) {
	if f.enableColor {
		buf.WriteString(color)
	}
	buf.WriteString(value)
	if f.enableColor {
		buf.WriteString(colorReset)
	}
}

// writeAttrs 写入属性
func (f *ColorJSONFormatter) writeAttrs(buf *bytes.Buffer, attrs []slog.Attr, groups []string) {
	// 处理分组
	openGroups := 0
	for _, g := range groups {
		buf.WriteString(`,"`)
		buf.WriteString(g)
		buf.WriteString(`":{`)
		openGroups++
	}

	for _, attr := range attrs {
		if attr.Key == "" {
			continue
		}
		buf.WriteByte(',')
		f.writeAttr(buf, attr)
	}

	// 关闭分组
	for i := 0; i < openGroups; i++ {
		buf.WriteByte('}')
	}
}

// writeAttr 写入单个属性
func (f *ColorJSONFormatter) writeAttr(buf *bytes.Buffer, attr slog.Attr) {
	buf.WriteByte('"')
	buf.WriteString(attr.Key)
	buf.WriteString(`":`)
	f.writeValue(buf, attr.Value)
}

// writeValue 写入值
func (f *ColorJSONFormatter) writeValue(buf *bytes.Buffer, v slog.Value) {
	v = v.Resolve()

	switch v.Kind() {
	case slog.KindString:
		f.writeColoredString(buf, colorGreen, v.String())

	case slog.KindInt64:
		f.writeColoredValue(buf, colorYellow, strconv.FormatInt(v.Int64(), 10))

	case slog.KindUint64:
		f.writeColoredValue(buf, colorYellow, strconv.FormatUint(v.Uint64(), 10))

	case slog.KindFloat64:
		f.writeColoredValue(buf, colorYellow, strconv.FormatFloat(v.Float64(), 'f', -1, 64))

	case slog.KindBool:
		if v.Bool() {
			f.writeColoredValue(buf, colorYellow, "true")
		} else {
			f.writeColoredValue(buf, colorYellow, "false")
		}

	case slog.KindDuration:
		f.writeColoredString(buf, colorYellow, v.Duration().String())

	case slog.KindTime:
		t := v.Time()
		if f.opts.Location != nil {
			t = t.In(f.opts.Location)
		}
		f.writeColoredString(buf, colorGreen, formatTime(t, f.opts.TimeFormat))

	case slog.KindGroup:
		buf.WriteByte('{')
		attrs := v.Group()
		for i, attr := range attrs {
			if i > 0 {
				buf.WriteByte(',')
			}
			f.writeAttr(buf, attr)
		}
		buf.WriteByte('}')

	case slog.KindAny:
		f.writeAny(buf, v.Any())

	default:
		f.writeColoredString(buf, colorGreen, v.String())
	}
}

// writeAny 写入任意类型
func (f *ColorJSONFormatter) writeAny(buf *bytes.Buffer, v any) {
	if v == nil {
		f.writeColoredValue(buf, colorGray, "null")
		return
	}

	data, err := json.Marshal(v)
	if err != nil {
		f.writeColoredString(buf, colorRed, "<error>")
		return
	}

	// 复杂类型直接输出 JSON（不带颜色）
	buf.Write(data)
}
