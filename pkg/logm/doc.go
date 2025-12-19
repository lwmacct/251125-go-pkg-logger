// Package logm 提供统一的结构化日志系统。
//
// 基于 Go 1.21+ 的 log/slog 包构建，采用 Functional Options 模式配置，
// 支持多种输出格式、日志轮转、异步写入和动态级别调整。
//
// # Architecture
//
// logm 包采用 Handler + Formatter + Writer 架构：
//   - Handler: 统一的 slog.Handler 实现，处理日志记录
//   - Formatter: 格式化器接口，决定日志输出格式（JSON/Text/Color）
//   - Writer: 输出目标接口，支持多种输出（Stdout/File/Async/Multi）
//
// # Quick Start
//
// 最简单的使用方式是使用预设配置：
//
//	func main() {
//	    // 开发环境：彩色输出 + DEBUG + 源代码位置
//	    logm.MustInit(logm.PresetDev()...)
//	    defer logm.Close()
//
//	    slog.Info("应用启动", "version", "1.0.0")
//	}
//
// 生产环境：
//
//	logm.MustInit(logm.PresetProd()...)
//
// 从环境变量读取配置：
//
//	logm.MustInit(logm.PresetFromEnv()...)
//
// # Functional Options
//
// 使用 Functional Options 进行精确配置：
//
//	logm.Init(
//	    logm.WithLevel("DEBUG"),
//	    logm.WithFormatter(formatter.ColorText()),
//	    logm.WithWriter(writer.Multi(
//	        writer.Stdout(),
//	        writer.File("/var/log/app.log", writer.WithRotation(100, 7)),
//	    )),
//	    logm.WithAddSource(true),
//	)
//
// # Sub-packages
//
// formatter 子包提供格式化器实现：
//
//	import "github.com/.../logm/formatter"
//
//	formatter.JSON()       // JSON 格式，适合生产环境
//	formatter.Text()       // 键值对格式，兼容传统工具
//	formatter.ColorText()  // 彩色文本，适合开发环境
//	formatter.ColorJSON()  // 彩色 JSON，适合终端调试
//
// writer 子包提供输出目标实现：
//
//	import "github.com/.../logm/writer"
//
//	writer.Stdout()                          // 标准输出
//	writer.File(path, writer.WithRotation(100, 7))  // 带轮转的文件
//	writer.Async(w, 1000)                    // 异步写入
//	writer.Multi(w1, w2)                     // 多目标输出
//
// # Dynamic Level
//
// 支持运行时动态调整日志级别：
//
//	logm.SetLevel("DEBUG")  // 开启调试日志
//	logm.SetLevel("ERROR")  // 只显示错误
//
// # Interceptors
//
// 使用拦截器添加通用字段或过滤日志：
//
//	logm.Init(
//	    logm.WithInterceptor(func(ctx context.Context, r *logm.Record) *logm.Record {
//	        r.Attrs = append(r.Attrs, slog.String("trace_id", getTraceID(ctx)))
//	        return r
//	    }),
//	)
//
// # Context Integration
//
// 在 HTTP 请求等场景中，可将 logger 存入 context 实现请求追踪：
//
//	func Handler(w http.ResponseWriter, r *http.Request) {
//	    ctx := logm.WithRequestID(r.Context(), r.Header.Get("X-Request-ID"))
//	    log := logm.FromContext(ctx)
//	    log.Info("处理请求", "path", r.URL.Path)
//	}
//
// # Thread Safety
//
// 本包所有导出函数都是并发安全的。全局 logger 可在多个 goroutine 中安全使用。
// [slog.Logger] 实例也是并发安全的，可以在 context 中自由传递。
// 动态级别调整（SetLevel）也是线程安全的。
package logm
