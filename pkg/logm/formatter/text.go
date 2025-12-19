package formatter

import (
	"bytes"
	"log/slog"
	"strconv"
)

// TextFormatter 文本格式化器。
//
// 输出 key=value 格式的文本日志，兼容传统日志分析工具。
type TextFormatter struct {
	opts *Options
}

// Text 创建文本格式化器。
func Text(opts ...Option) *TextFormatter {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return &TextFormatter{opts: o}
}

// Format 实现 Formatter 接口。
func (f *TextFormatter) Format(r *Record) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	// 时间
	t := r.Time
	if f.opts.Location != nil {
		t = t.In(f.opts.Location)
	}
	buf.WriteString("time=")
	buf.WriteString(formatTime(t, f.opts.TimeFormat))

	// 级别
	buf.WriteString(" level=")
	buf.WriteString(levelString(r.Level))

	// 消息
	buf.WriteString(" msg=")
	writeTextValue(buf, r.Message)

	// 源代码位置
	if r.Source != nil {
		buf.WriteString(" source=")
		buf.WriteString(FormatSource(r.Source, f.opts))
	}

	// 属性
	f.writeAttrs(buf, r.Attrs, r.Groups)

	buf.WriteByte('\n')

	return copyBytes(buf.Bytes()), nil
}

// writeAttrs 写入属性
func (f *TextFormatter) writeAttrs(buf *bytes.Buffer, attrs []slog.Attr, groups []string) {
	prefix := ""
	for _, g := range groups {
		prefix += g + "."
	}

	for _, attr := range attrs {
		if attr.Key == "" {
			continue
		}
		buf.WriteByte(' ')
		buf.WriteString(prefix)
		f.writeAttr(buf, attr, prefix)
	}
}

// writeAttr 写入单个属性
func (f *TextFormatter) writeAttr(buf *bytes.Buffer, attr slog.Attr, prefix string) {
	buf.WriteString(attr.Key)
	buf.WriteByte('=')
	f.writeValue(buf, attr.Value, prefix+attr.Key+".")
}

// writeValue 写入值
func (f *TextFormatter) writeValue(buf *bytes.Buffer, v slog.Value, prefix string) {
	v = v.Resolve()

	switch v.Kind() {
	case slog.KindString:
		writeTextValue(buf, v.String())
	case slog.KindInt64:
		buf.WriteString(strconv.FormatInt(v.Int64(), 10))
	case slog.KindUint64:
		buf.WriteString(strconv.FormatUint(v.Uint64(), 10))
	case slog.KindFloat64:
		buf.WriteString(strconv.FormatFloat(v.Float64(), 'f', -1, 64))
	case slog.KindBool:
		if v.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case slog.KindDuration:
		buf.WriteString(v.Duration().String())
	case slog.KindTime:
		t := v.Time()
		if f.opts.Location != nil {
			t = t.In(f.opts.Location)
		}
		writeTextValue(buf, formatTime(t, f.opts.TimeFormat))
	case slog.KindGroup:
		// 展开分组
		attrs := v.Group()
		for i, attr := range attrs {
			if i > 0 {
				buf.WriteByte(' ')
				buf.WriteString(prefix[:len(prefix)-1]) // 去掉末尾的点
			}
			f.writeAttr(buf, attr, prefix)
		}
	default:
		writeTextValue(buf, v.String())
	}
}

// writeTextValue 写入文本值（需要时添加引号）
func writeTextValue(buf *bytes.Buffer, s string) {
	needQuote := false
	for _, r := range s {
		if r == ' ' || r == '"' || r == '=' || r == '\n' || r == '\r' || r == '\t' {
			needQuote = true
			break
		}
	}

	if !needQuote && len(s) > 0 {
		buf.WriteString(s)
		return
	}

	buf.WriteByte('"')
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
			buf.WriteRune(r)
		}
	}
	buf.WriteByte('"')
}
