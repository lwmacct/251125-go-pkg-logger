package logm

import (
	"os"
	"strings"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter"
	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
)

// Development 返回开发环境预设配置。
//
// 特点：
//   - 彩色输出到 stdout
//   - DEBUG 级别
//   - 显示源代码位置
//   - 简洁时间格式 (15:04:05)
//
// 使用：
//
//	logm.Init(logm.Development()...)
func Development() []Option {
	return []Option{
		WithLevel("DEBUG"),
		WithFormatter(formatter.Color(
			formatter.WithTimeFormat("time"),
		)),
		WithWriter(writer.Stdout()),
		WithAddSource(true),
		WithTimeFormat("time"),
		WithTimezone("Asia/Shanghai"),
	}
}

// Production 返回生产环境预设配置。
//
// 特点：
//   - JSON 格式输出
//   - INFO 级别
//   - 不显示源代码位置
//   - RFC3339 时间格式
//
// 使用：
//
//	logm.Init(logm.Production()...)
func Production() []Option {
	return []Option{
		WithLevel("INFO"),
		WithFormatter(formatter.JSON(
			formatter.WithTimeFormat("rfc3339ms"),
		)),
		WithWriter(writer.Stdout()),
		WithAddSource(false),
		WithTimeFormat("rfc3339ms"),
		WithTimezone("UTC"),
	}
}

// ProductionWithFile 返回生产环境预设配置（带文件输出）。
//
// 同时输出到 stdout 和指定文件，文件启用轮转。
//
// 使用：
//
//	logm.Init(logm.ProductionWithFile("/var/log/app.log")...)
func ProductionWithFile(path string, opts ...writer.FileOption) []Option {
	return []Option{
		WithLevel("INFO"),
		WithFormatter(formatter.JSON(
			formatter.WithTimeFormat("rfc3339ms"),
		)),
		WithWriter(writer.Multi(
			writer.Stdout(),
			writer.File(path, opts...),
		)),
		WithAddSource(false),
		WithTimeFormat("rfc3339ms"),
		WithTimezone("UTC"),
	}
}

// FromEnv 根据环境变量返回配置。
//
// 支持的环境变量：
//   - IS_SANDBOX: 1/true 使用开发配置，否则使用生产配置
//   - LOG_LEVEL: DEBUG, INFO, WARN, ERROR
//   - LOG_FORMAT: json, text, color
//   - LOG_OUTPUT: stdout, stderr, 或文件路径
//   - LOG_ADD_SOURCE: true, false
//   - LOG_TIME_FORMAT: time, datetime, rfc3339, rfc3339ms
//
// 使用：
//
//	logm.Init(logm.FromEnv()...)
func FromEnv() []Option {
	isSandbox := isSandboxEnv()

	// 基础预设
	var opts []Option
	if isSandbox {
		opts = Development()
	} else {
		opts = Production()
	}

	// 环境变量覆盖
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		opts = append(opts, WithLevel(level))
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		var f Formatter
		switch strings.ToLower(format) {
		case "json":
			f = formatter.JSON()
		case "text":
			f = formatter.Text()
		case "color", "colored":
			f = formatter.Color()
		}
		if f != nil {
			opts = append(opts, WithFormatter(f))
		}
	}

	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		opts = append(opts, WithOutput(output))
	}

	if addSource := os.Getenv("LOG_ADD_SOURCE"); addSource != "" {
		enable := strings.ToLower(addSource) == "true" || addSource == "1"
		opts = append(opts, WithAddSource(enable))
	}

	if timeFormat := os.Getenv("LOG_TIME_FORMAT"); timeFormat != "" {
		opts = append(opts, WithTimeFormat(timeFormat))
	}

	return opts
}

// isSandboxEnv 检测是否为沙盒/开发环境
func isSandboxEnv() bool {
	value := os.Getenv("IS_SANDBOX")
	return value == "1" || strings.ToLower(value) == "true"
}
