package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// globalCloser 保存全局 logger 的可关闭资源（如文件句柄）。
//
// 当使用 [InitCfg] 或 [InitEnv] 初始化时会被设置，调用 [Close] 时释放。
var globalCloser io.Closer

// InitCfg 使用指定配置初始化全局日志系统，设置 [slog.Default]。
//
// 该函数应在应用启动时调用一次。可多次调用以更换配置，
// 之前的资源（如文件）会被自动关闭。
// 如果输出到文件，应在程序退出时调用 [Close] 释放资源。
//
// 示例：
//
//	func main() {
//	    if err := logger.InitCfg(&logger.Config{
//	        Level:  "DEBUG",
//	        Format: "color",
//	    }); err != nil {
//	        log.Fatalf("初始化日志失败: %v", err)
//	    }
//	    defer logger.Close()
//	    // 使用 slog.Info() 或 logger.Info() 记录日志
//	}
func InitCfg(cfg *Config) error {
	logger, closer, err := NewWithCloser(cfg)
	if err != nil {
		return err
	}
	// 关闭之前的 closer（忽略错误，因为我们正在替换它）
	if globalCloser != nil {
		_ = globalCloser.Close()
	}
	globalCloser = closer
	slog.SetDefault(logger)
	return nil
}

// InitEnv 从环境变量读取配置并初始化全局日志系统。
//
// 这是最简单的初始化方式，适合大多数应用。根据 IS_SANDBOX 环境变量
// 自动切换开发/生产模式的默认配置。
//
// 支持的环境变量：
//   - IS_SANDBOX:      环境检测 (1/true 为开发环境，影响以下默认值)
//   - LOG_LEVEL:       日志级别 (DEBUG, INFO, WARN, ERROR)
//   - LOG_FORMAT:      输出格式 (json, text, color)
//   - LOG_OUTPUT:      输出目标 (stdout, stderr, 或文件路径)
//   - LOG_ADD_SOURCE:  是否添加源代码位置 (true, false)
//   - LOG_TIME_FORMAT: 时间格式 (datetime, time, timems, rfc3339, rfc3339ms)
//
// 默认值对照表：
//
//	| 配置项          | 开发环境 (IS_SANDBOX=1) | 生产环境  |
//	|-----------------|-------------------------|-----------|
//	| LOG_LEVEL       | DEBUG                   | INFO      |
//	| LOG_FORMAT      | color                   | json      |
//	| LOG_ADD_SOURCE  | true                    | false     |
//	| LOG_TIME_FORMAT | time (15:04:05)         | datetime  |
func InitEnv() error {
	isSandbox := isSandboxEnv()

	// 根据环境选择默认值
	defaultLevel := "INFO"
	defaultFormat := "json"
	defaultAddSource := false
	defaultTimeFormat := "datetime"

	if isSandbox {
		defaultLevel = "DEBUG"
		defaultFormat = "color"
		defaultAddSource = true
		defaultTimeFormat = "time"
	}

	cfg := &Config{
		Level:      getEnv("LOG_LEVEL", defaultLevel),
		Format:     getEnv("LOG_FORMAT", defaultFormat),
		Output:     getEnv("LOG_OUTPUT", "stdout"),
		AddSource:  getEnvBool("LOG_ADD_SOURCE", defaultAddSource),
		TimeFormat: getEnv("LOG_TIME_FORMAT", defaultTimeFormat),
		Timezone:   "Asia/Shanghai",
	}

	return InitCfg(cfg)
}

// Close 释放全局 logger 持有的资源（如打开的日志文件）。
//
// 应在程序退出前调用，通常配合 defer 使用。
// 如果没有需要关闭的资源（如输出到 stdout），该函数是空操作并返回 nil。
// 多次调用是安全的。
func Close() error {
	if globalCloser != nil {
		err := globalCloser.Close()
		globalCloser = nil
		return err
	}
	return nil
}

// isSandboxEnv 检测当前是否运行在沙盒/开发环境中。
//
// 通过检查 IS_SANDBOX 环境变量判断，值为 "1" 或 "true"（不区分大小写）时返回 true。
func isSandboxEnv() bool {
	value := os.Getenv("IS_SANDBOX")
	return value == "1" || strings.ToLower(value) == "true"
}

// getEnv 获取环境变量的值，如果未设置或为空则返回默认值。
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量，"true" 或 "1" 解析为 true，其他为 false。
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
}
