package formatter

import (
	"bytes"
	"sync"
)

// 缓冲池，减少内存分配
var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// getBuffer 从池中获取缓冲区
func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// putBuffer 将缓冲区放回池中
func putBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 {
		// 太大的缓冲区不回收
		return
	}
	bufferPool.Put(buf)
}

// copyBytes 复制字节切片
func copyBytes(b []byte) []byte {
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}
