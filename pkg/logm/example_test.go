package logm_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter"
)

// Example 展示 logm 包的基本使用方式。
func Example() {
	// 使用 Functional Options 初始化
	_ = logm.Init(
		logm.WithLevel("DEBUG"),
		logm.WithFormatter(formatter.Text()),
	)
	defer func() { _ = logm.Close() }()

	// 实际使用时，日志会输出到配置的目标
	// logm.Info("应用启动", "version", "1.0.0")
	// logm.Debug("调试信息", "key", "value")
	// Output:
}

// Example_slogCompatible 展示 logm 与标准库 slog 的完全兼容性。
//
// logm.Init() 之后，可以直接使用 slog.Info() 等标准库函数，
// 日志会自动路由到 logm 配置的 Handler。
// 这种设计允许业务代码仅依赖标准库 log/slog，便于替换日志实现。
func Example_slogCompatible() {
	var buf bytes.Buffer
	w := &testBufWriter{buf: &buf}

	// 使用 JSON formatter 初始化 logm
	_ = logm.Init(
		logm.WithFormatter(formatter.JSON()),
		logm.WithWriter(w),
		logm.WithLevel("INFO"),
	)
	defer func() { _ = logm.Close() }()

	// 直接使用标准库 slog，日志会经过 logm 的 JSON formatter
	slog.Info("hello from slog", "user", "alice")

	// 验证输出是 JSON 格式（由 logm 配置的 formatter 生成）
	output := buf.String()
	if strings.Contains(output, `"msg":"hello from slog"`) &&
		strings.Contains(output, `"user":"alice"`) &&
		strings.Contains(output, `"level":"INFO"`) {
		fmt.Println("slog uses logm's formatter")
	}
	// Output: slog uses logm's formatter
}

// Example_development 展示开发环境预设配置。
//
// PresetDev() 预设会自动配置彩色输出、DEBUG 级别、显示源代码位置。
func Example_development() {
	// 使用开发环境预设：彩色输出 + DEBUG + 源代码位置
	_ = logm.Init(logm.PresetDev()...)
	defer func() { _ = logm.Close() }()

	// 此处不实际输出日志，仅展示 API 用法
	// logm.Debug("调试信息", "module", "auth")
	// Output:
}

// Example_production 展示生产环境预设配置。
//
// PresetProd() 预设会自动配置 JSON 格式、INFO 级别、RFC3339 时间格式。
func Example_production() {
	// 使用生产环境预设：JSON + INFO + RFC3339
	_ = logm.Init(logm.PresetProd()...)
	defer func() { _ = logm.Close() }()

	// 此处不实际输出日志，仅展示 API 用法
	// logm.Info("服务启动", "port", 8080)
	// Output:
}

// Example_fromEnv 展示通过环境变量初始化日志的方式。
//
// PresetFromEnv 会根据 LOGM_ENV 环境变量自动选择开发或生产配置。
func Example_fromEnv() {
	// PresetFromEnv 从环境变量读取配置，适合大多数应用
	// 支持: LOGM_ENV, LOGM_LEVEL, LOGM_FORMAT, LOGM_OUTPUT, LOGM_SOURCE 等
	_ = logm.Init(logm.PresetFromEnv()...)
	defer func() { _ = logm.Close() }()

	// 此处不实际输出日志，仅展示 API 用法
	// logm.Info("日志系统已初始化")
	// Output:
}

// Example_withRequestID 展示如何在请求处理中追踪日志。
func Example_withRequestID() {
	ctx := context.Background()

	// 为请求添加追踪 ID
	ctx = logm.WithRequestID(ctx, "req-12345")

	// 从 context 获取带有 request_id 的 logger
	log := logm.FromContext(ctx)
	_ = log // log.Info("处理请求") 会自动包含 request_id 字段
	// Output:
}

// Example_new 展示如何创建独立的 logger 实例。
func Example_new() {
	// 创建 JSON 格式的独立 logger，适用于模块专用日志
	jsonLogger := logm.New(
		logm.WithLevel("INFO"),
		logm.WithFormatter(formatter.JSON()),
		logm.WithAddSource(true),
	)
	_ = jsonLogger
	// Output:
}

// ExampleFormatBytes 展示字节格式化函数的使用。
func ExampleFormatBytes() {
	fmt.Println(logm.FormatBytes(0))
	fmt.Println(logm.FormatBytes(1024))
	fmt.Println(logm.FormatBytes(1024 * 1024))
	fmt.Println(logm.FormatBytes(1536 * 1024 * 1024))
	// Output:
	// 0 B
	// 1.0 KB
	// 1.0 MB
	// 1.5 GB
}

// Example_with 展示如何创建带有固定属性的 logger。
func Example_with() {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	// 创建带有固定属性的 logger，适用于模块级日志标记
	moduleLog := logm.With("module", "worker", "version", "2.0")
	moduleLog.Info("任务完成", "count", 42)

	output := buf.String()
	if strings.Contains(output, "module=worker") &&
		strings.Contains(output, "count=42") {
		fmt.Println("with works")
	}
	// Output: with works
}

// ExampleLogError 展示错误日志的记录方式。
func ExampleLogError() {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	// LogError 记录错误并返回原始错误，适用于一行完成日志和返回
	err := context.DeadlineExceeded
	returnedErr := logm.LogError(context.Background(), "操作超时", err, "timeout", "5s")

	if returnedErr == context.DeadlineExceeded {
		fmt.Println("original error returned")
	}
	// Output: original error returned
}

// ExampleWithGroup 展示使用分组组织日志属性。
func ExampleWithGroup() {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	// 使用分组：属性会嵌套在 "request" 下
	reqLog := logm.WithGroup("request")
	reqLog.Info("处理请求", "method", "GET", "path", "/api/users")

	output := buf.String()
	if strings.Contains(output, `"request":{`) {
		fmt.Println("group works")
	}
	// Output: group works
}

// ExampleFromContext 展示从 context 获取 logger。
func ExampleFromContext() {
	ctx := context.Background()

	// 从空 context 获取时，返回全局默认 logger
	_ = logm.FromContext(ctx)

	// 可以向 context 中存入自定义 logger
	customLog := slog.Default().With("service", "api")
	ctx = logm.WithLogger(ctx, customLog)

	// 再次获取时返回自定义 logger
	log := logm.FromContext(ctx)
	_ = log
	// Output:
}

// Example_coloredOutput 展示彩色输出配置。
//
// 彩色输出适合开发环境，便于快速区分日志级别。
func Example_coloredOutput() {
	_ = logm.Init(
		logm.WithLevel("DEBUG"),
		logm.WithFormatter(formatter.ColorText()),
		logm.WithTimeFormat("time"),
		logm.WithAddSource(true),
	)
	defer func() { _ = logm.Close() }()

	// 不同级别的日志会显示不同颜色
	// logm.Debug("调试信息", "module", "auth")
	// logm.Info("用户登录", "user_id", 12345)
	// Output:
}

// Example_jsonOutput 展示 JSON 输出配置。
//
// JSON 输出适合生产环境，便于日志采集和分析。
func Example_jsonOutput() {
	_ = logm.Init(
		logm.WithLevel("INFO"),
		logm.WithFormatter(formatter.JSON(
			formatter.WithTimeFormat("rfc3339ms"),
		)),
	)
	defer func() { _ = logm.Close() }()

	// JSON 输出格式化后易于机器解析
	// logm.Info("API 请求", "method", "POST", "path", "/api/users")
	// Output:
}

// Example_dynamicLevel 展示动态调整日志级别。
func Example_dynamicLevel() {
	_ = logm.Init(logm.WithLevel("INFO"))
	defer func() { _ = logm.Close() }()

	// 运行时动态调整日志级别
	logm.SetLevel("DEBUG")

	// 现在 DEBUG 级别的日志可以输出了
	// logm.Debug("动态启用调试日志")

	// 切换回 INFO 级别
	logm.SetLevel("INFO")
	// Output:
}

// Example_slogGroup 展示使用 slog.Group 组织结构化日志。
func Example_slogGroup() {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	// slog.Group 将多个属性组织在一个命名空间下
	slog.Info("HTTP 请求",
		slog.Group("request",
			"method", "GET",
			"path", "/api/users",
		),
		slog.Group("response",
			"status", 200,
			"size", 1024,
		),
	)

	output := buf.String()
	if strings.Contains(output, `"request":{`) &&
		strings.Contains(output, `"response":{`) {
		fmt.Println("slog.Group works")
	}
	// Output: slog.Group works
}

// ExampleParseLevel 展示日志级别解析。
func ExampleParseLevel() {
	fmt.Println(logm.ParseLevel("DEBUG"))
	fmt.Println(logm.ParseLevel("INFO"))
	fmt.Println(logm.ParseLevel("WARN"))
	fmt.Println(logm.ParseLevel("ERROR"))
	fmt.Println(logm.ParseLevel("unknown")) // 默认返回 INFO
	// Output:
	// DEBUG
	// INFO
	// WARN
	// ERROR
	// INFO
}

// testBufWriter 用于测试的 Writer 实现
type testBufWriter struct {
	buf *bytes.Buffer
}

func (w *testBufWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *testBufWriter) Close() error {
	return nil
}

func (w *testBufWriter) Sync() error {
	return nil
}
