// Package writer 提供日志输出目标实现。
//
// Writer 决定日志的输出位置，内置多种类型：
//   - Stdout/Stderr: 标准输出
//   - File: 文件输出，支持轮转
//   - Async: 异步写入，提升性能
//   - Multi: 多目标输出
//
// # 使用示例
//
//	import "github.com/.../logger/writer"
//
//	logm.Init(logm.WithWriter(writer.Stdout()))
//	logm.Init(logm.WithWriter(writer.File("/var/log/app.log", writer.WithRotation(100, 7))))
package writer

import "io"

// Writer 日志输出目标接口。
//
// 扩展 io.Writer，增加 Close 和 Sync 方法。
type Writer interface {
	io.Writer
	io.Closer
	// Sync 刷新缓冲区
	Sync() error
}

// 确保所有 Writer 实现接口
var (
	_ Writer = (*StdWriter)(nil)
	_ Writer = (*FileWriter)(nil)
	_ Writer = (*AsyncWriter)(nil)
	_ Writer = (*MultiWriter)(nil)
)
