package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// globalCloser 保存全局 logger 的可关闭资源
var globalCloser io.Closer

// InitCfg 使用配置初始化全局日志系统
//
// 这个函数应该在应用启动时调用一次，用于配置全局的 slog.Default() logger
// 如果输出到文件，应在程序退出时调用 Close() 关闭文件
//
// 使用示例：
//
//	func main() {
//	    if err := logger.InitCfg(&logger.Config{
//	        Level:  "DEBUG",
//	        Format: "color",
//	    }); err != nil {
//	        log.Fatalf("初始化日志失败: %v", err)
//	    }
//	    defer logger.Close()
//	    // ...
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

// EnvConfig 返回根据环境变量生成的默认配置
//
// 通过 IS_SANDBOX 环境变量自动检测运行环境，返回对应的默认配置。
// 可用于获取配置后进行修改，再传给 InitCfg。
//
// 使用示例：
//
//	cfg := logger.EnvConfig()
//	cfg.Level = "WARN"  // 覆盖某个配置
//	logger.InitCfg(cfg)
func EnvConfig() *Config {
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

	return &Config{
		Level:      getEnv("LOG_LEVEL", defaultLevel),
		Format:     getEnv("LOG_FORMAT", defaultFormat),
		Output:     getEnv("LOG_OUTPUT", "stdout"),
		AddSource:  getEnvBool("LOG_ADD_SOURCE", defaultAddSource),
		TimeFormat: getEnv("LOG_TIME_FORMAT", defaultTimeFormat),
		Timezone:   "Asia/Shanghai",
	}
}

// InitEnv 从环境变量初始化全局日志系统
//
// 等同于 InitCfg(EnvConfig())，详见 EnvConfig 文档。
func InitEnv() error {
	return InitCfg(EnvConfig())
}

// Close 关闭全局 logger 的资源（如文件）
//
// 应在程序退出时调用，确保日志文件正确关闭
func Close() error {
	if globalCloser != nil {
		err := globalCloser.Close()
		globalCloser = nil
		return err
	}
	return nil
}

// isSandboxEnv 检测是否为沙盒/开发环境
func isSandboxEnv() bool {
	value := os.Getenv("IS_SANDBOX")
	return value == "1" || strings.ToLower(value) == "true"
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
}
