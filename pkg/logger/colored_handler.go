package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// 日志级别颜色映射
var levelColors = map[slog.Level]string{
	slog.LevelDebug: "\033[94m", // 蓝色
	slog.LevelInfo:  "\033[92m", // 绿色
	slog.LevelWarn:  "\033[93m", // 黄色
	slog.LevelError: "\033[91m", // 红色
}

// 字段颜色映射
var keyColors = map[string]string{
	"time":   "\033[90m", // 灰色
	"level":  "",         // 使用级别颜色
	"msg":    "\033[34m", // 深蓝色
	"error":  "\033[31m", // 红色
	"warn":   "\033[33m", // 黄色
	"source": "\033[35m", // 紫色（调用位置）
	"data":   "\033[32m", // 绿色
	"other":  "\033[36m", // 青色（其他字段）
}

const colorReset = "\033[0m"

// ColoredHandlerConfig 彩色 handler 配置
type ColoredHandlerConfig struct {
	// Level 最小日志级别
	Level slog.Level
	// AddSource 是否添加源代码位置
	AddSource bool
	// EnableColor 是否启用颜色
	EnableColor bool
	// CallerClip 调用路径裁剪前缀
	CallerClip string
	// PriorityKeys 优先显示的字段（按顺序）
	PriorityKeys []string
	// TrailingKeys 固定在尾部的字段（按顺序）
	TrailingKeys []string
	// TimeFormat 时间格式: datetime (默认), rfc3339, rfc3339ms, time, timems
	TimeFormat string
	// Timezone 时区名称，例如 "Asia/Shanghai"
	Timezone string
}

// DefaultColoredConfig 返回默认配置
func DefaultColoredConfig() *ColoredHandlerConfig {
	return &ColoredHandlerConfig{
		Level:        slog.LevelInfo,
		AddSource:    true,
		EnableColor:  true,
		CallerClip:   "",
		PriorityKeys: []string{"time", "level", "msg"},
		TrailingKeys: []string{"source"},
	}
}

// coloredHandler 支持彩色输出和字段排序的 handler
type coloredHandler struct {
	writer   io.Writer
	config   *ColoredHandlerConfig
	groups   []string
	attrs    []slog.Attr
	mu       sync.Mutex
	location *time.Location // 缓存的时区
}

// NewColoredHandler 创建彩色 handler
func NewColoredHandler(w io.Writer, config *ColoredHandlerConfig) *coloredHandler {
	if config == nil {
		config = DefaultColoredConfig()
	}

	// 加载时区，默认使用上海时区 (UTC+8)
	loc := loadTimezone(config.Timezone)

	return &coloredHandler{
		writer:   w,
		config:   config,
		location: loc,
	}
}

// Enabled 实现 slog.Handler 接口
func (h *coloredHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.config.Level
}

// Handle 实现 slog.Handler 接口
func (h *coloredHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 收集所有字段
	fields := make(map[string]string)

	// 添加时间
	fields["time"] = h.getFormattedTime(r.Time)

	// 添加级别
	fields["level"] = r.Level.String()

	// 添加消息
	if r.Message != "" {
		fields["msg"] = r.Message
	}

	// 添加源代码位置
	if h.config.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			source := fmt.Sprintf("%s:%d", f.File, f.Line)
			fields["source"] = h.clipPath(source)
		}
	}

	// 添加 handler 自身的属性（支持 JSON 平铺，已包含 group 前缀）
	for _, attr := range h.attrs {
		h.flattenAttr(fields, attr.Key, attr.Value)
	}

	// 添加记录中的属性（需要加上当前 group 前缀）
	r.Attrs(func(a slog.Attr) bool {
		key := h.buildKey(a.Key)
		h.flattenAttr(fields, key, a.Value)
		return true
	})

	// 格式化输出
	output := h.formatFields(fields)

	// 写入
	_, err := h.writer.Write([]byte(output + "\n"))
	return err
}

// WithAttrs 实现 slog.Handler 接口
func (h *coloredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 为新属性添加当前 group 前缀
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	for _, attr := range attrs {
		key := h.buildKey(attr.Key)
		newAttrs = append(newAttrs, slog.Attr{Key: key, Value: attr.Value})
	}

	return &coloredHandler{
		writer:   h.writer,
		config:   h.config,
		groups:   h.groups,
		attrs:    newAttrs,
		location: h.location,
	}
}

// WithGroup 实现 slog.Handler 接口
func (h *coloredHandler) WithGroup(name string) slog.Handler {
	return &coloredHandler{
		writer:   h.writer,
		config:   h.config,
		groups:   append(slices.Clip(h.groups), name),
		attrs:    h.attrs,
		location: h.location,
	}
}

// getFormattedTime 根据配置格式化时间
func (h *coloredHandler) getFormattedTime(t time.Time) string {
	// 转换到指定时区
	if h.location != nil {
		t = t.In(h.location)
	}

	switch h.config.TimeFormat {
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
		return t.Format(h.config.TimeFormat)
	}
}

// buildKey 根据当前 groups 构建完整的 key
func (h *coloredHandler) buildKey(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	return strings.Join(h.groups, ".") + "." + key
}

// clipPath 裁剪路径
func (h *coloredHandler) clipPath(path string) string {
	originalPath := path

	// 优先处理 /workspace/xxx/ 前缀
	path = clipWorkspacePath(path)

	// 如果 workspace 切割成功（路径发生了变化），检查深度
	// 深度 <= 2 时直接返回，不再进一步裁剪
	if path != originalPath {
		depth := strings.Count(path, "/")
		if depth <= 2 {
			return path
		}
	}

	if h.config.CallerClip != "" {
		return strings.Replace(path, h.config.CallerClip, "", 1)
	}

	// 默认裁剪：只保留最后 3 层路径
	if len(path) > 0 && path[0] == '/' {
		count := 0
		pos := len(path) - 1

		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '/' {
				count++
				if count == 3 {
					pos = i
					break
				}
			}
		}

		if count >= 3 {
			return path[pos:]
		}
	}

	return filepath.Base(path)
}

// flattenAttr 将属性平铺到 fields 中，支持 JSON 字符串、map、struct 的递归展开
func (h *coloredHandler) flattenAttr(fields map[string]string, prefix string, v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		// 字符串：尝试解析为 JSON
		s := v.String()
		if len(s) > 1 && (s[0] == '{' || s[0] == '[') {
			var data any
			if err := json.Unmarshal([]byte(s), &data); err == nil {
				h.flattenJSON(fields, prefix, data)
				return
			}
		}
		fields[prefix] = escapeString(s)

	case slog.KindGroup:
		// slog.Group：递归处理每个属性
		attrs := v.Group()
		for _, attr := range attrs {
			newKey := attr.Key
			if prefix != "" {
				newKey = prefix + "." + attr.Key
			}
			h.flattenAttr(fields, newKey, attr.Value)
		}

	case slog.KindAny:
		// any 类型：尝试平铺 map/struct
		h.flattenAny(fields, prefix, v.Any())

	default:
		// 其他基本类型
		fields[prefix] = h.formatValue(v)
	}
}

// flattenAny 处理 any 类型的平铺（map、struct 等）
func (h *coloredHandler) flattenAny(fields map[string]string, prefix string, v any) {
	if v == nil {
		fields[prefix] = "null"
		return
	}

	// 尝试直接类型断言为 map[string]any
	if m, ok := v.(map[string]any); ok {
		h.flattenJSON(fields, prefix, m)
		return
	}

	// 尝试 json.Marshal 转换（支持 struct 和其他 map 类型）
	data, err := json.Marshal(v)
	if err != nil {
		// 无法序列化，使用默认格式
		fields[prefix] = escapeString(fmt.Sprintf("%v", v))
		return
	}

	// 解析 JSON 并平铺
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		fields[prefix] = escapeString(string(data))
		return
	}

	h.flattenJSON(fields, prefix, parsed)
}

// flattenJSON 递归平铺 JSON 数据
func (h *coloredHandler) flattenJSON(fields map[string]string, prefix string, data any) {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			newKey := key
			if prefix != "" {
				newKey = prefix + "." + key
			}
			h.flattenJSON(fields, newKey, val)
		}
	case []any:
		for i, val := range v {
			newKey := fmt.Sprintf("%s[%d]", prefix, i)
			h.flattenJSON(fields, newKey, val)
		}
	case string:
		fields[prefix] = escapeString(v)
	case float64:
		// JSON 数字默认是 float64
		if v == float64(int64(v)) {
			fields[prefix] = fmt.Sprintf("%d", int64(v))
		} else {
			fields[prefix] = fmt.Sprintf("%g", v)
		}
	case bool:
		fields[prefix] = fmt.Sprintf("%t", v)
	case nil:
		fields[prefix] = "null"
	default:
		fields[prefix] = escapeString(fmt.Sprintf("%v", v))
	}
}

// formatValue 格式化值（用于非 JSON 场景）
func (h *coloredHandler) formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return escapeString(v.String())
	case slog.KindInt64:
		return fmt.Sprintf("%d", v.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", v.Uint64())
	case slog.KindFloat64:
		return fmt.Sprintf("%g", v.Float64())
	case slog.KindBool:
		return fmt.Sprintf("%t", v.Bool())
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	default:
		return escapeString(fmt.Sprintf("%v", v.Any()))
	}
}

// escapeString 转义字符串中的特殊字符（用于 JSON 风格输出）
func escapeString(s string) string {
	// 快速路径：如果没有需要转义的字符，直接返回
	needsEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '"' || s[i] == '\\' || s[i] < 0x20 {
			needsEscape = true
			break
		}
	}
	if !needsEscape {
		return s
	}

	// 需要转义
	var builder strings.Builder
	builder.Grow(len(s) + 10)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			if c < 0x20 {
				builder.WriteString(fmt.Sprintf(`\u%04x`, c))
			} else {
				builder.WriteByte(c)
			}
		}
	}
	return builder.String()
}

// formatFields 格式化字段为彩色输出
func (h *coloredHandler) formatFields(fields map[string]string) string {
	// 分类字段
	priorityFields := make([]string, 0, len(h.config.PriorityKeys))
	trailingFields := make([]string, 0, len(h.config.TrailingKeys))
	otherFields := make([]string, 0)

	// 按优先级和尾部字段分类
	prioritySet := make(map[string]bool)
	trailingSet := make(map[string]bool)

	for _, key := range h.config.PriorityKeys {
		prioritySet[key] = true
		if _, exists := fields[key]; exists {
			priorityFields = append(priorityFields, key)
		}
	}

	for _, key := range h.config.TrailingKeys {
		trailingSet[key] = true
		if _, exists := fields[key]; exists {
			trailingFields = append(trailingFields, key)
		}
	}

	// 收集其他字段并排序
	for key := range fields {
		if !prioritySet[key] && !trailingSet[key] {
			otherFields = append(otherFields, key)
		}
	}
	sort.Strings(otherFields)

	// 合并所有字段
	allKeys := make([]string, 0, len(fields))
	allKeys = append(allKeys, priorityFields...)
	allKeys = append(allKeys, otherFields...)
	allKeys = append(allKeys, trailingFields...)

	// 构建输出
	var builder strings.Builder
	builder.Grow(len(allKeys) * 50) // 预估容量

	builder.WriteByte('{')

	first := true
	for _, key := range allKeys {
		value, exists := fields[key]
		if !exists {
			continue
		}

		if !first {
			builder.WriteByte(',')
		}
		first = false

		// 写入键
		builder.WriteByte('"')
		builder.WriteString(key)
		builder.WriteString(`":"`)

		// 写入值（带颜色）
		if h.config.EnableColor {
			h.writeColoredValue(&builder, key, value)
		} else {
			builder.WriteString(value)
		}

		builder.WriteByte('"')
	}

	builder.WriteByte('}')
	return builder.String()
}

// writeColoredValue 写入带颜色的值
func (h *coloredHandler) writeColoredValue(builder *strings.Builder, key, value string) {
	// 特殊处理 level 字段
	if key == "level" {
		level := parseLevel(value) // 使用包级别函数
		if color, ok := levelColors[level]; ok {
			builder.WriteString(color)
			builder.WriteString(value)
			builder.WriteString(colorReset)
			return
		}
	}

	// 使用字段颜色
	color, ok := keyColors[key]
	if !ok {
		color = keyColors["other"]
	}

	if color != "" {
		builder.WriteString(color)
		builder.WriteString(value)
		builder.WriteString(colorReset)
	} else {
		builder.WriteString(value)
	}
}
