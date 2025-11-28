package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/lwmacct/251125-go-mod-logger/pkg/logger"
)

func main() {
	// 示例 1: InitEnv - 从环境变量初始化（推荐）
	demoInitEnv()

	// 示例 2: InitCfg - 手动配置
	demoInitCfg()

	// 示例 3: 彩色输出
	demoColoredOutput()

	// 示例 4: JSON 输出
	demoJSONOutput()

	// 示例 5: 时间格式
	demoTimeFormats()

	// 示例 6: Context 集成
	demoContextIntegration()

	// 示例 7: 结构化日志与数据平铺
	demoStructuredLogging()
}

// demoInitEnv 演示从环境变量初始化
func demoInitEnv() {
	printSection("InitEnv - 从环境变量初始化（推荐）")

	// 自动检测 IS_SANDBOX 环境变量：
	// - IS_SANDBOX=1: 开发环境 (color + DEBUG + source + time)
	// - 其他: 生产环境 (json + INFO + no source + datetime)
	//
	// 支持的环境变量覆盖：
	// - LOG_LEVEL: DEBUG, INFO, WARN, ERROR
	// - LOG_FORMAT: json, text, color
	// - LOG_OUTPUT: stdout, stderr, 文件路径
	// - LOG_ADD_SOURCE: true, false
	// - LOG_TIME_FORMAT: datetime, time, rfc3339, rfc3339ms
	if err := logger.InitEnv(); err != nil {
		panic(err)
	}
	defer logger.Close()

	slog.Info("环境检测", "is_sandbox", os.Getenv("IS_SANDBOX"))
	slog.Debug("调试信息（开发环境可见）")
}

// demoInitCfg 演示手动配置初始化
func demoInitCfg() {
	printSection("InitCfg - 手动配置")

	cfg := &logger.Config{
		Level:      "DEBUG",
		Format:     "color",
		Output:     "stdout",
		AddSource:  true,
		TimeFormat: "datetime",
		Timezone:   "Asia/Shanghai",
	}
	if err := logger.InitCfg(cfg); err != nil {
		panic(err)
	}

	slog.Info("手动配置初始化", "level", cfg.Level, "format", cfg.Format)
}

// demoColoredOutput 演示彩色输出
func demoColoredOutput() {
	printSection("彩色输出")

	cfg := &logger.Config{
		Level:      "DEBUG",
		Format:     "color",
		TimeFormat: "time", // 简洁时间格式 15:04:05
	}
	if err := logger.InitCfg(cfg); err != nil {
		panic(err)
	}

	slog.Debug("调试信息", "module", "auth")
	slog.Info("用户登录", "user_id", 12345, "ip", "192.168.1.100")
	slog.Warn("连接池即将耗尽", "active", 95, "max", 100)
	slog.Error("数据库连接失败", "error", "connection refused")
}

// demoJSONOutput 演示 JSON 输出
func demoJSONOutput() {
	printSection("JSON 输出")

	cfg := &logger.Config{
		Level:      "INFO",
		Format:     "json",
		TimeFormat: "datetime",
	}
	if err := logger.InitCfg(cfg); err != nil {
		panic(err)
	}

	slog.Info("API 请求", "method", "POST", "path", "/api/users", "duration_ms", 42)
}

// demoTimeFormats 演示时间格式
func demoTimeFormats() {
	printSection("时间格式")

	formats := []struct {
		name   string
		format string
	}{
		{"time", "time"},           // 15:04:05
		{"timems", "timems"},       // 15:04:05.000
		{"datetime", "datetime"},   // 2006-01-02 15:04:05
		{"rfc3339", "rfc3339"},     // 2006-01-02T15:04:05+08:00
		{"rfc3339ms", "rfc3339ms"}, // 2006-01-02T15:04:05.000+08:00
		{"custom", "01-02 15:04"},  // 自定义格式
	}

	for _, f := range formats {
		cfg := &logger.Config{
			Level:      "INFO",
			Format:     "color",
			TimeFormat: f.format,
		}
		if err := logger.InitCfg(cfg); err != nil {
			panic(err)
		}
		slog.Info("时间格式示例", "format", f.name)
	}
}

// demoContextIntegration 演示 Context 集成
func demoContextIntegration() {
	printSection("Context 集成")

	cfg := &logger.Config{
		Level:  "INFO",
		Format: "color",
	}
	if err := logger.InitCfg(cfg); err != nil {
		panic(err)
	}

	// 创建带 request_id 的 context
	ctx := context.Background()
	ctx = logger.WithRequestID(ctx, "req-abc123")

	// 从 context 获取 logger
	log := logger.FromContext(ctx)
	log.Info("处理请求", "action", "create_user")
}

// demoStructuredLogging 演示结构化日志与数据平铺
func demoStructuredLogging() {
	printSection("结构化日志与数据平铺")

	cfg := &logger.Config{
		Level:  "INFO",
		Format: "color",
	}
	if err := logger.InitCfg(cfg); err != nil {
		panic(err)
	}

	// JSON 字符串自动平铺
	slog.Info("收到请求体", "body", `{"user":"alice","age":30}`)

	// map 自动平铺
	slog.Info("配置信息", "config", map[string]any{
		"debug":   true,
		"timeout": 30,
	})

	// slog.Group 分组
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

	// 带属性的 logger
	serviceLogger := logger.WithAttrs("service", "payment", "version", "2.0")
	serviceLogger.Info("支付成功", "order_id", "ORD-001", "amount", 99.99)
}

// printSection 打印分隔标题
func printSection(title string) {
	os.Stdout.WriteString("\n")
	os.Stdout.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	os.Stdout.WriteString("  " + title + "\n")
	os.Stdout.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}
