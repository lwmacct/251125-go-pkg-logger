package formatter

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"
)

// JSONFormatter JSON 格式化器。
//
// 输出紧凑的 JSON 格式，适合生产环境日志采集和分析。
type JSONFormatter struct {
	opts *Options
}

// JSON 创建 JSON 格式化器。
func JSON(opts ...Option) *JSONFormatter {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return &JSONFormatter{opts: o}
}

// Format 实现 Formatter 接口。
func (f *JSONFormatter) Format(r *Record) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	buf.WriteByte('{')

	// 时间
	t := r.Time
	if f.opts.Location != nil {
		t = t.In(f.opts.Location)
	}
	buf.WriteString(`"time":"`)
	buf.WriteString(formatTime(t, f.opts.TimeFormat))
	buf.WriteByte('"')

	// 级别
	buf.WriteString(`,"level":"`)
	buf.WriteString(LevelName(r.Level))
	buf.WriteByte('"')

	// 消息
	buf.WriteString(`,"msg":`)
	writeJSONString(buf, r.Message)

	// 源代码位置
	if r.Source != nil {
		buf.WriteString(`,"source":"`)
		buf.WriteString(FormatSource(r.Source, f.opts))
		buf.WriteByte('"')
	}

	// 属性
	f.writeAttrs(buf, r.Attrs, r.Groups)

	buf.WriteByte('}')
	buf.WriteByte('\n')

	return copyBytes(buf.Bytes()), nil
}

// writeAttrs 写入属性
func (f *JSONFormatter) writeAttrs(buf *bytes.Buffer, attrs []slog.Attr, groups []string) {
	// 处理分组
	openGroups := 0
	for _, g := range groups {
		buf.WriteString(`,"`)
		buf.WriteString(g)
		buf.WriteString(`":{`)
		openGroups++
	}

	// 写入属性
	first := len(groups) == 0
	for _, attr := range attrs {
		if attr.Key == "" {
			continue
		}
		switch {
		case !first:
			buf.WriteByte(',')
		case openGroups > 0:
			// 分组内的第一个属性不需要逗号
		default:
			buf.WriteByte(',')
		}
		first = false
		f.writeAttr(buf, attr)
	}

	// 关闭分组
	for range openGroups {
		buf.WriteByte('}')
	}
}

// writeAttr 写入单个属性
func (f *JSONFormatter) writeAttr(buf *bytes.Buffer, attr slog.Attr) {
	buf.WriteByte('"')
	buf.WriteString(attr.Key)
	buf.WriteString(`":`)
	f.writeValue(buf, attr.Value)
}

// writeValue 写入值
func (f *JSONFormatter) writeValue(buf *bytes.Buffer, v slog.Value) {
	v = v.Resolve()

	switch v.Kind() {
	case slog.KindString:
		writeJSONString(buf, v.String())
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
		writeJSONString(buf, v.Duration().String())
	case slog.KindTime:
		t := v.Time()
		if f.opts.Location != nil {
			t = t.In(f.opts.Location)
		}
		writeJSONString(buf, t.Format(time.RFC3339Nano))
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
		writeJSONString(buf, v.String())
	}
}

// writeAny 写入任意类型
func (f *JSONFormatter) writeAny(buf *bytes.Buffer, v any) {
	if v == nil {
		buf.WriteString("null")
		return
	}

	// 特殊处理 error 接口
	// json.Marshal(error) 返回 {} 因为 error 接口没有导出字段
	if err, ok := v.(error); ok {
		writeJSONString(buf, err.Error())
		return
	}

	// 尝试 JSON 序列化
	data, err := json.Marshal(v)
	if err != nil {
		writeJSONString(buf, "<error>")
		return
	}
	buf.Write(data)
}

// writeJSONString 写入 JSON 字符串（带转义）
func writeJSONString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	EscapeJSON(buf, s)
	buf.WriteByte('"')
}
