package writer

import (
	"io"
	"os"
)

// StdWriter 标准输出/错误输出 Writer。
type StdWriter struct {
	w io.Writer
}

// Stdout 创建标准输出 Writer。
func Stdout() *StdWriter {
	return &StdWriter{w: os.Stdout}
}

// Stderr 创建标准错误输出 Writer。
func Stderr() *StdWriter {
	return &StdWriter{w: os.Stderr}
}

// Write 实现 io.Writer。
func (s *StdWriter) Write(p []byte) (n int, err error) {
	return s.w.Write(p)
}

// Close 实现 io.Closer（无操作）。
func (s *StdWriter) Close() error {
	return nil
}

// Sync 实现 Writer.Sync（无操作）。
func (s *StdWriter) Sync() error {
	return nil
}
