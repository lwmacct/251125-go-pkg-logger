package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// customJSONHandler 实现 [slog.Handler] 接口，输出 JSON 格式日志。
//
// 相比标准 [slog.JSONHandler]，该实现支持：
//   - 灵活的时间格式配置（unix/rfc3339/datetime 等）
//   - 自动裁剪 /workspace/ 路径前缀
//   - 正确的 Group 嵌套输出
type customJSONHandler struct {
	opts       *slog.HandlerOptions
	writer     io.Writer
	timeFormat string
	mu         sync.Mutex
	groups     []string       // 当前 group 路径
	preAttrs   map[string]any // 预计算的属性（已考虑 group 嵌套）
	location   *time.Location // 缓存的时区
}

// newJSONHandler 创建并返回一个 JSON 格式的 [slog.Handler]。
//
// timeFormat 支持: unix, unixms, rfc3339, rfc3339ms, datetime（默认）。
// timezone 为 IANA 时区名或固定偏移（如 "+08:00"），默认 Asia/Shanghai。
func newJSONHandler(w io.Writer, opts *slog.HandlerOptions, timeFormat string, timezone string) *customJSONHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	if timeFormat == "" {
		timeFormat = "datetime"
	}

	// 加载时区，默认使用上海时区 (UTC+8)
	loc := loadTimezone(timezone)

	return &customJSONHandler{
		opts:       opts,
		writer:     w,
		timeFormat: timeFormat,
		preAttrs:   make(map[string]any),
		location:   loc,
	}
}

// Enabled 实现 [slog.Handler] 接口，判断指定级别的日志是否应该被记录。
func (h *customJSONHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

// Handle 实现 [slog.Handler] 接口，将日志记录序列化为 JSON 并写入输出。
//
// 该方法是线程安全的，使用互斥锁保护写入操作。
func (h *customJSONHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 构建 JSON 对象
	m := make(map[string]any)

	// 添加时间字段
	m["time"] = h.formatTime(r.Time)

	// 添加级别字段
	m["level"] = r.Level.String()

	// 添加消息字段
	m["msg"] = r.Message

	// 添加源代码位置（如果启用）
	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			source := fmt.Sprintf("%s:%d", f.File, f.Line)
			m["source"] = clipWorkspacePath(source)
		}
	}

	// 合并预计算的属性（已包含 group 嵌套）
	for k, v := range h.preAttrs {
		m[k] = deepCopyValue(v)
	}

	// 添加记录中的属性（需要考虑当前 group 路径）
	r.Attrs(func(a slog.Attr) bool {
		h.setNestedAttr(m, h.groups, a.Key, a.Value.Any())
		return true
	})

	// 序列化为 JSON
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	// 写入
	_, err = h.writer.Write(append(data, '\n'))
	return err
}

// WithAttrs 实现 [slog.Handler] 接口，返回携带额外属性的新 handler。
func (h *customJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 深拷贝现有属性
	newPreAttrs := make(map[string]any)
	for k, v := range h.preAttrs {
		newPreAttrs[k] = deepCopyValue(v)
	}

	// 将新属性添加到当前 group 路径下
	for _, attr := range attrs {
		h.setNestedAttr(newPreAttrs, h.groups, attr.Key, attr.Value.Any())
	}

	return &customJSONHandler{
		opts:       h.opts,
		writer:     h.writer,
		timeFormat: h.timeFormat,
		groups:     h.groups,
		preAttrs:   newPreAttrs,
		location:   h.location,
	}
}

// WithGroup 实现 [slog.Handler] 接口，返回在指定分组下记录属性的新 handler。
func (h *customJSONHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)

	// 深拷贝现有属性
	newPreAttrs := make(map[string]any)
	for k, v := range h.preAttrs {
		newPreAttrs[k] = deepCopyValue(v)
	}

	return &customJSONHandler{
		opts:       h.opts,
		writer:     h.writer,
		timeFormat: h.timeFormat,
		groups:     newGroups,
		preAttrs:   newPreAttrs,
		location:   h.location,
	}
}

// formatTime 根据配置将时间格式化为字符串或数字。
//
// 返回值类型取决于格式：unix* 返回 int64/string，其他返回 string。
func (h *customJSONHandler) formatTime(t time.Time) interface{} {
	// 转换到指定时区
	if h.location != nil {
		t = t.In(h.location)
	}

	switch h.timeFormat {
	case "unix":
		// Unix 时间戳（秒）
		return t.Unix()
	case "unixms":
		// Unix 时间戳（毫秒）
		return t.UnixMilli()
	case "unixnano":
		// Unix 时间戳（纳秒）
		return t.UnixNano()
	case "rfc3339":
		// RFC3339 格式（秒精度）
		return t.Format(time.RFC3339)
	case "rfc3339ms":
		// RFC3339 格式（毫秒精度）
		return t.Format("2006-01-02T15:04:05.000Z07:00")
	case "datetime", "":
		// 默认：简单的日期时间格式（秒精度）
		return t.Format("2006-01-02 15:04:05")
	case "unixfloat":
		// Unix 时间戳（浮点数，秒+小数）
		unixSec := t.Unix()
		nanoRemainder := t.Nanosecond()
		return strconv.FormatFloat(float64(unixSec)+float64(nanoRemainder)/1e9, 'f', 3, 64)
	default:
		// 默认使用日期时间格式
		return t.Format("2006-01-02 15:04:05")
	}
}

// setNestedAttr 在嵌套的 map 结构中设置属性值。
//
// groups 指定嵌套路径，例如 ["request", "headers"] 会将 key 设置在
// m["request"]["headers"][key]。如果中间层不存在，会自动创建。
func (h *customJSONHandler) setNestedAttr(m map[string]any, groups []string, key string, value any) {
	if len(groups) == 0 {
		m[key] = value
		return
	}

	// 遍历 group 路径，逐层创建或获取嵌套 map
	current := m
	for _, group := range groups {
		if existing, ok := current[group]; ok {
			if nested, ok := existing.(map[string]any); ok {
				current = nested
			} else {
				// 已存在但不是 map，创建新的 map 覆盖
				nested := make(map[string]any)
				current[group] = nested
				current = nested
			}
		} else {
			nested := make(map[string]any)
			current[group] = nested
			current = nested
		}
	}
	current[key] = value
}

// deepCopyValue 对值进行深拷贝，防止共享状态被意外修改。
//
// 主要处理 map[string]any 类型的递归拷贝，其他类型直接返回原值。
func deepCopyValue(v any) any {
	if m, ok := v.(map[string]any); ok {
		copied := make(map[string]any, len(m))
		for k, val := range m {
			copied[k] = deepCopyValue(val)
		}
		return copied
	}
	return v
}
