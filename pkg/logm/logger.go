package logm

import (
	"log/slog"
	"sync"
	"time"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter"
	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

var (
	// globalHandler 全局 Handler
	globalHandler *Handler
	// globalMu 保护全局状态
	globalMu sync.RWMutex
)

// Init 初始化全局日志系统。
//
// 使用 Functional Options 模式配置：
//
//	logm.Init(
//	    logm.WithLevel("DEBUG"),
//	    logm.WithFormatter(formatter.Color()),
//	    logm.WithWriter(writer.Stdout()),
//	)
//
// 也可使用预设配置：
//
//	logm.Init(logm.Development()...)
//	logm.Init(logm.Production()...)
func Init(opts ...Option) error {
	o := defaultOptions()
	o.apply(opts...)

	// 解析时区
	if o.timezone != "" {
		o.location = mustLoadTimezone(o.timezone)
	}
	if o.location == nil {
		o.location = time.Local
	}

	// 默认 formatter
	if o.formatter == nil {
		o.formatter = formatter.Text(
			formatter.WithTimeFormat(o.timeFormat),
			formatter.WithTimezone(o.timezone),
		)
	}

	// 默认 writer
	if len(o.writers) == 0 {
		o.writers = append(o.writers, writer.Stdout())
	}

	// 创建 LevelVar
	levelVar := o.levelVar
	if levelVar == nil {
		levelVar = globalLevelVar
	}
	levelVar.Set(ParseLevel(o.level))

	// 创建 Handler
	h := NewHandler(&HandlerConfig{
		LevelVar:     levelVar,
		Formatter:    o.formatter,
		Writers:      o.writers,
		Interceptors: o.interceptors,
		AddSource:    o.addSource,
		TimeFormat:   o.timeFormat,
		Location:     o.location,
	})

	// 设置全局
	globalMu.Lock()
	if globalHandler != nil {
		_ = globalHandler.Close()
	}
	globalHandler = h
	globalMu.Unlock()

	slog.SetDefault(slog.New(h))
	return nil
}

// MustInit 初始化全局日志系统，失败时 panic。
//
// 适用于程序启动阶段，日志系统初始化失败通常意味着程序无法正常运行：
//
//	func main() {
//	    logm.MustInit(logm.Development()...)
//	    defer logm.Close()
//	    // ...
//	}
func MustInit(opts ...Option) {
	if err := Init(opts...); err != nil {
		panic("logm: init failed: " + err.Error())
	}
}

// New 创建独立的 logger 实例。
//
// 返回的 logger 独立于全局配置，适用于模块专用日志。
func New(opts ...Option) *slog.Logger {
	o := defaultOptions()
	o.apply(opts...)

	// 解析时区
	if o.timezone != "" {
		o.location = mustLoadTimezone(o.timezone)
	}
	if o.location == nil {
		o.location = time.Local
	}

	// 默认 formatter
	if o.formatter == nil {
		o.formatter = formatter.Text(
			formatter.WithTimeFormat(o.timeFormat),
			formatter.WithTimezone(o.timezone),
		)
	}

	// 默认 writer
	if len(o.writers) == 0 {
		o.writers = append(o.writers, writer.Stdout())
	}

	// 创建独立的 LevelVar
	levelVar := &slog.LevelVar{}
	levelVar.Set(ParseLevel(o.level))

	h := NewHandler(&HandlerConfig{
		LevelVar:     levelVar,
		Formatter:    o.formatter,
		Writers:      o.writers,
		Interceptors: o.interceptors,
		AddSource:    o.addSource,
		TimeFormat:   o.timeFormat,
		Location:     o.location,
	})

	return slog.New(h)
}

// Close 关闭全局日志系统，释放资源。
func Close() error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalHandler != nil {
		err := globalHandler.Close()
		globalHandler = nil
		return err
	}
	return nil
}

// Sync 刷新全局日志缓冲区。
func Sync() error {
	globalMu.RLock()
	h := globalHandler
	globalMu.RUnlock()

	if h != nil {
		return h.Sync()
	}
	return nil
}

// Default 返回全局默认 logger。
func Default() *slog.Logger {
	return slog.Default()
}

// 便捷日志函数

// Debug 记录调试级别日志。
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Info 记录信息级别日志。
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn 记录警告级别日志。
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error 记录错误级别日志。
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// With 返回带有额外属性的 logger。
func With(args ...any) *slog.Logger {
	return slog.Default().With(args...)
}

// WithGroup 返回带有分组的 logger。
func WithGroup(name string) *slog.Logger {
	return slog.Default().WithGroup(name)
}

// mustLoadTimezone 加载时区，失败返回 nil
func mustLoadTimezone(tz string) *time.Location {
	if tz == "" {
		return nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil
	}
	return loc
}
