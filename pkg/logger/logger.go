// Package logger 提供统一的结构化日志系统
//
// 基于 Go 1.21+ 的 log/slog 包，提供：
// 1. 统一的日志配置和初始化
// 2. 支持多种输出格式（JSON、文本）
// 3. 灵活的日志级别控制
// 4. 结构化日志的最佳实践
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config 定义日志系统的配置选项。
//
// 所有字段都有合理的默认值，零值配置即可正常工作。
// 使用 [InitCfg] 或 [InitEnv] 初始化全局日志，或使用 [New] 创建独立实例。
type Config struct {
	// Level 日志级别: DEBUG, INFO, WARN, ERROR
	Level string
	// Format 输出格式: json, text, color
	Format string
	// Output 输出目标: stdout, stderr, 或文件路径
	Output string
	// AddSource 是否添加源代码位置信息
	AddSource bool
	// TimeFormat 时间格式: datetime (默认), rfc3339, rfc3339ms, unix, unixms
	TimeFormat string
	// Timezone 时区名称，例如 "Asia/Shanghai"，默认为 "Asia/Shanghai"
	Timezone string
}

// defaultConfig 返回默认配置（内部使用）
func defaultConfig() *Config {
	return &Config{
		Level:      "INFO",
		Format:     "text",
		Output:     "stdout",
		AddSource:  false,
		TimeFormat: "datetime",
		Timezone:   "Asia/Shanghai",
	}
}

// validFormats 有效的输出格式
var validFormats = map[string]bool{
	"json": true, "text": true, "color": true, "colored": true,
}

// validLevels 有效的日志级别
var validLevels = map[string]bool{
	"DEBUG": true, "INFO": true, "WARN": true, "WARNING": true, "ERROR": true,
}

// Validate 验证配置字段是否有效。
//
// 检查 Format 和 Level 字段的值是否在允许范围内。
// 返回 error 描述第一个遇到的无效字段；配置有效时返回 nil。
func (c *Config) Validate() error {
	if c.Format != "" && !validFormats[c.Format] {
		return fmt.Errorf("invalid log format: %q, valid options: json, text, color", c.Format)
	}

	level := strings.ToUpper(c.Level)
	if c.Level != "" && !validLevels[level] {
		return fmt.Errorf("invalid log level: %q, valid options: DEBUG, INFO, WARN, ERROR", c.Level)
	}

	return nil
}

// New 创建并返回一个新的 [slog.Logger] 实例。
//
// 适用于需要独立配置的场景，例如为特定模块创建专用 logger。
// 如果 cfg 为 nil，将使用默认配置（INFO 级别、text 格式、输出到 stdout）。
//
// 注意：如果输出到文件，建议使用 [NewWithCloser] 以便正确关闭文件资源。
func New(cfg *Config) (*slog.Logger, error) {
	logger, _, err := NewWithCloser(cfg)
	return logger, err
}

// NewWithCloser 创建新的 [slog.Logger] 实例，同时返回用于释放资源的 [io.Closer]。
//
// 当输出目标是文件时，closer 不为 nil，调用者应在适当时机调用 closer.Close() 关闭文件。
// 当输出目标是 stdout 或 stderr 时，closer 为 nil。
//
// 如果 cfg 为 nil，将使用默认配置。配置验证失败时返回 error。
func NewWithCloser(cfg *Config) (*slog.Logger, io.Closer, error) {
	if cfg == nil {
		cfg = defaultConfig()
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	writer, closer, err := getWriter(cfg.Output)
	if err != nil {
		return nil, nil, err
	}

	handler := createHandler(cfg, writer)
	return slog.New(handler), closer, nil
}

// createHandler 根据配置创建 slog.Handler
func createHandler(cfg *Config, writer io.Writer) slog.Handler {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	switch cfg.Format {
	case "json":
		return newJSONHandler(writer, opts, cfg.TimeFormat, cfg.Timezone)
	case "color", "colored":
		colorConfig := &ColoredHandlerConfig{
			Level:        level,
			AddSource:    cfg.AddSource,
			EnableColor:  true,
			CallerClip:   "",
			PriorityKeys: []string{"time", "level", "msg"},
			TrailingKeys: []string{"source"},
			TimeFormat:   cfg.TimeFormat,
			Timezone:     cfg.Timezone,
		}
		return NewColoredHandler(writer, colorConfig)
	default: // text
		return newTextHandler(writer, opts, cfg.TimeFormat, cfg.Timezone)
	}
}

// parseLevel 解析日志级别字符串（大小写不敏感）
func parseLevel(levelStr string) slog.Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// getWriter 获取输出写入器
// 返回 writer 和 closer（如果是文件则 closer 不为 nil）
func getWriter(output string) (io.Writer, io.Closer, error) {
	switch output {
	case "stdout", "":
		return os.Stdout, nil, nil
	case "stderr":
		return os.Stderr, nil, nil
	default:
		// 文件路径
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, nil, err
		}
		return file, file, nil
	}
}

// WithAttrs 返回一个携带额外属性的全局 logger 副本。
//
// attrs 参数为键值对形式，奇数位置为键（string），偶数位置为值。
// 返回的 logger 会在每条日志中自动包含这些属性。
//
// 示例：
//
//	log := logger.WithAttrs("module", "worker", "node_id", nodeID)
//	log.Info("任务完成", "count", 42)  // 输出包含 module、node_id、count
func WithAttrs(attrs ...any) *slog.Logger {
	return slog.Default().With(attrs...)
}

// WithGroup 返回一个带有属性分组的全局 logger 副本。
//
// 分组会将后续添加的属性嵌套在指定名称下，使 JSON 输出更有层次结构。
// 例如 WithGroup("request").With("method", "GET") 输出 {"request":{"method":"GET"}}。
func WithGroup(name string) *slog.Logger {
	return slog.Default().WithGroup(name)
}
