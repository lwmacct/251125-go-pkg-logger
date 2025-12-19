package writer

import (
	"gopkg.in/natefinch/lumberjack.v2"
)

// FileWriter 文件 Writer，支持日志轮转。
//
// 基于 lumberjack 实现，支持按大小轮转、备份数量限制和压缩。
type FileWriter struct {
	lj *lumberjack.Logger
}

// FileOption 文件 Writer 选项
type FileOption func(*lumberjack.Logger)

// File 创建文件 Writer。
//
// 默认配置：100MB 轮转、保留 7 个备份、30 天过期、启用压缩。
func File(path string, opts ...FileOption) *FileWriter {
	lj := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    100, // MB
		MaxBackups: 7,
		MaxAge:     30, // days
		Compress:   true,
		LocalTime:  true,
	}

	for _, opt := range opts {
		opt(lj)
	}

	return &FileWriter{lj: lj}
}

// WithRotation 设置轮转配置。
//
// maxSize: 单个文件最大大小（MB）
// maxBackups: 保留的备份文件数量
func WithRotation(maxSize, maxBackups int) FileOption {
	return func(lj *lumberjack.Logger) {
		lj.MaxSize = maxSize
		lj.MaxBackups = maxBackups
	}
}

// WithMaxAge 设置文件保留天数。
func WithMaxAge(days int) FileOption {
	return func(lj *lumberjack.Logger) {
		lj.MaxAge = days
	}
}

// WithCompress 设置是否压缩旧日志。
func WithCompress(enable bool) FileOption {
	return func(lj *lumberjack.Logger) {
		lj.Compress = enable
	}
}

// WithLocalTime 设置文件名时间戳是否使用本地时间。
func WithLocalTime(enable bool) FileOption {
	return func(lj *lumberjack.Logger) {
		lj.LocalTime = enable
	}
}

// Write 实现 io.Writer。
func (f *FileWriter) Write(p []byte) (n int, err error) {
	return f.lj.Write(p)
}

// Close 实现 io.Closer。
func (f *FileWriter) Close() error {
	return f.lj.Close()
}

// Sync 实现 Writer.Sync。
func (f *FileWriter) Sync() error {
	// lumberjack 没有显式的 sync 方法，每次写入都会 flush
	return nil
}

// Rotate 手动触发日志轮转。
func (f *FileWriter) Rotate() error {
	return f.lj.Rotate()
}
