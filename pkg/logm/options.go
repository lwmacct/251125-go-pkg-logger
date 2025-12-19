package logm

import (
	"io"
	"log/slog"
	"os"
	"time"
)

// Option 配置选项函数
type Option func(*options)

// options 内部配置结构
type options struct {
	level      string
	levelVar   *slog.LevelVar
	formatter  Formatter
	writers    []Writer
	addSource  bool
	timeFormat string
	timezone   string
	location   *time.Location

	interceptors []Interceptor
}

// defaultOptions 返回默认配置
func defaultOptions() *options {
	return &options{
		level:      "INFO",
		addSource:  false,
		timeFormat: "datetime",
		timezone:   "Asia/Shanghai",
	}
}

// apply 应用所有选项
func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithLevel 设置日志级别。
//
// 支持: DEBUG, INFO, WARN, ERROR（大小写不敏感）
func WithLevel(level string) Option {
	return func(o *options) {
		o.level = level
	}
}

// WithLevelVar 使用 slog.LevelVar 设置动态日志级别。
//
// 允许运行时修改日志级别。
func WithLevelVar(lv *slog.LevelVar) Option {
	return func(o *options) {
		o.levelVar = lv
	}
}

// WithFormatter 设置日志格式化器。
//
// 使用 formatter 子包中的预定义格式化器：
//   - formatter.JSON()
//   - formatter.Text()
//   - formatter.Color()
func WithFormatter(f Formatter) Option {
	return func(o *options) {
		o.formatter = f
	}
}

// WithWriter 添加日志输出目标。
//
// 使用 writer 子包中的预定义 Writer：
//   - writer.Stdout()
//   - writer.File(path, opts...)
//   - writer.Async(w, bufferSize)
func WithWriter(w Writer) Option {
	return func(o *options) {
		o.writers = append(o.writers, w)
	}
}

// WithOutput 添加输出目标（简化版本）。
//
// 支持: "stdout", "stderr", 或文件路径
func WithOutput(output string) Option {
	return func(o *options) {
		var w Writer
		switch output {
		case "stdout", "":
			w = &stdWriter{os.Stdout}
		case "stderr":
			w = &stdWriter{os.Stderr}
		default:
			// 文件路径 - 使用简单文件写入（无轮转）
			f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return // 忽略错误，后续会处理
			}
			w = &fileWriter{f}
		}
		o.writers = append(o.writers, w)
	}
}

// WithAddSource 启用源代码位置记录。
func WithAddSource(enable bool) Option {
	return func(o *options) {
		o.addSource = enable
	}
}

// WithTimeFormat 设置时间格式。
//
// 预定义格式:
//   - "time": 15:04:05
//   - "timems": 15:04:05.000
//   - "datetime": 2006-01-02 15:04:05
//   - "rfc3339": RFC3339 格式
//   - "rfc3339ms": RFC3339 带毫秒
//   - 自定义: Go time 格式字符串
func WithTimeFormat(format string) Option {
	return func(o *options) {
		o.timeFormat = format
	}
}

// WithTimezone 设置时区。
//
// 支持 IANA 时区名称（如 "Asia/Shanghai"）或固定偏移（如 "+08:00"）
func WithTimezone(tz string) Option {
	return func(o *options) {
		o.timezone = tz
	}
}

// WithInterceptor 添加日志拦截器。
//
// 拦截器按添加顺序执行，可用于添加通用字段或过滤日志。
func WithInterceptor(i Interceptor) Option {
	return func(o *options) {
		o.interceptors = append(o.interceptors, i)
	}
}

// stdWriter 包装标准输出
type stdWriter struct {
	w io.Writer
}

func (s *stdWriter) Write(p []byte) (n int, err error) { return s.w.Write(p) }
func (s *stdWriter) Close() error                      { return nil }
func (s *stdWriter) Sync() error                       { return nil }

// fileWriter 简单文件写入器（无轮转）
type fileWriter struct {
	f *os.File
}

func (f *fileWriter) Write(p []byte) (n int, err error) { return f.f.Write(p) }
func (f *fileWriter) Close() error                      { return f.f.Close() }
func (f *fileWriter) Sync() error                       { return f.f.Sync() }
