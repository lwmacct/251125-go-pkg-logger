package writer

// MultiWriter 多目标 Writer。
//
// 将日志同时写入多个目标。
type MultiWriter struct {
	writers []Writer
}

// Multi 创建多目标 Writer。
func Multi(writers ...Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// Write 实现 io.Writer。
//
// 写入所有目标，忽略单个目标的写入错误。
func (m *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range m.writers {
		_, _ = w.Write(p)
	}
	return len(p), nil
}

// Close 实现 io.Closer。
//
// 关闭所有目标。
func (m *MultiWriter) Close() error {
	var firstErr error
	for _, w := range m.writers {
		if err := w.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Sync 实现 Writer.Sync。
//
// 刷新所有目标。
func (m *MultiWriter) Sync() error {
	var firstErr error
	for _, w := range m.writers {
		if err := w.Sync(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Add 添加 Writer。
func (m *MultiWriter) Add(w Writer) {
	m.writers = append(m.writers, w)
}
