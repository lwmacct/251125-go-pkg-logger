package writer

import (
	"sync"
)

// AsyncWriter 异步 Writer。
//
// 使用缓冲通道异步写入，提升高并发场景下的性能。
// 调用 Close 时会等待所有缓冲数据写入完成。
type AsyncWriter struct {
	writer Writer
	ch     chan []byte
	wg     sync.WaitGroup
	closed bool
	mu     sync.Mutex
}

// Async 创建异步 Writer。
//
// bufferSize 指定缓冲通道大小，建议值 1000-10000。
func Async(w Writer, bufferSize int) *AsyncWriter {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	aw := &AsyncWriter{
		writer: w,
		ch:     make(chan []byte, bufferSize),
	}

	aw.wg.Add(1)
	go aw.run()

	return aw
}

// run 后台写入协程
func (a *AsyncWriter) run() {
	defer a.wg.Done()
	for data := range a.ch {
		_, _ = a.writer.Write(data)
	}
}

// Write 实现 io.Writer。
//
// 将数据复制后放入缓冲通道，非阻塞（除非缓冲区满）。
func (a *AsyncWriter) Write(p []byte) (n int, err error) {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return 0, nil
	}
	a.mu.Unlock()

	// 复制数据避免竞态
	data := make([]byte, len(p))
	copy(data, p)

	select {
	case a.ch <- data:
		return len(p), nil
	default:
		// 缓冲区满，丢弃日志（或可选择阻塞）
		return len(p), nil
	}
}

// Close 实现 io.Closer。
//
// 关闭通道并等待所有缓冲数据写入完成。
func (a *AsyncWriter) Close() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.closed = true
	a.mu.Unlock()

	close(a.ch)
	a.wg.Wait()
	return a.writer.Close()
}

// Sync 实现 Writer.Sync。
//
// 等待当前缓冲区数据写入完成。
func (a *AsyncWriter) Sync() error {
	// 创建一个 done 通道来同步
	done := make(chan struct{})
	a.ch <- nil // 发送一个 nil 作为同步标记

	go func() {
		for data := range a.ch {
			if data == nil {
				close(done)
				return
			}
			_, _ = a.writer.Write(data)
		}
	}()

	<-done
	return a.writer.Sync()
}
