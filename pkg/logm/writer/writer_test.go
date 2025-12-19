package writer

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============ StdWriter Tests ============

func TestStdout(t *testing.T) {
	w := Stdout()
	assert.NotNil(t, w)
	assert.Equal(t, os.Stdout, w.w)
}

func TestStderr(t *testing.T) {
	w := Stderr()
	assert.NotNil(t, w)
	assert.Equal(t, os.Stderr, w.w)
}

func TestStdWriter_Write(t *testing.T) {
	// 使用一个 buffer 来测试 write 功能
	var buf bytes.Buffer
	w := &StdWriter{w: &buf}

	n, err := w.Write([]byte("test message"))
	require.NoError(t, err)
	assert.Equal(t, 12, n)
	assert.Equal(t, "test message", buf.String())
}

func TestStdWriter_Close(t *testing.T) {
	w := Stdout()
	err := w.Close()
	assert.NoError(t, err) // Close is a no-op for stdout
}

func TestStdWriter_Sync(t *testing.T) {
	w := Stdout()
	err := w.Sync()
	assert.NoError(t, err) // Sync is a no-op for stdout
}

// ============ FileWriter Tests ============

func TestFile_Create(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.log")

	w := File(path)
	require.NotNil(t, w)

	n, err := w.Write([]byte("test line\n"))
	require.NoError(t, err)
	assert.Equal(t, 10, n)

	err = w.Close()
	require.NoError(t, err)

	// 验证文件内容
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "test line\n", string(content))
}

func TestFile_WithRotation(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.log")

	// 创建带轮转的 writer (1MB max, 3 backups)
	w := File(path, WithRotation(1, 3))
	require.NotNil(t, w)

	n, err := w.Write([]byte("test line\n"))
	require.NoError(t, err)
	assert.Equal(t, 10, n)

	err = w.Close()
	require.NoError(t, err)
}

func TestFileWriter_Sync(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.log")

	w := File(path)
	_, err := w.Write([]byte("test\n"))
	require.NoError(t, err)

	err = w.Sync()
	assert.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)
}

// ============ AsyncWriter Tests ============

func TestAsync_Create(t *testing.T) {
	var buf bytes.Buffer
	inner := &mockWriter{buf: &buf}

	w := Async(inner, 100)
	require.NotNil(t, w)

	err := w.Close()
	require.NoError(t, err)
}

func TestAsync_Write(t *testing.T) {
	var buf bytes.Buffer
	inner := &mockWriter{buf: &buf}

	w := Async(inner, 100)
	defer func() { _ = w.Close() }()

	n, err := w.Write([]byte("test message"))
	require.NoError(t, err)
	assert.Equal(t, 12, n)

	// 等待异步写入完成
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, "test message", buf.String())
}

func TestAsync_DefaultBufferSize(t *testing.T) {
	var buf bytes.Buffer
	inner := &mockWriter{buf: &buf}

	// bufferSize <= 0 should default to 1000
	w := Async(inner, 0)
	require.NotNil(t, w)
	assert.Equal(t, 1000, cap(w.ch))

	err := w.Close()
	require.NoError(t, err)
}

func TestAsync_Close(t *testing.T) {
	var buf bytes.Buffer
	inner := &mockWriter{buf: &buf}

	w := Async(inner, 100)

	// 写入一些数据
	_, _ = w.Write([]byte("data1"))
	_, _ = w.Write([]byte("data2"))

	// Close 应该等待所有数据写入
	err := w.Close()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "data1")
	assert.Contains(t, buf.String(), "data2")

	// 重复 Close 应该是安全的
	err = w.Close()
	assert.NoError(t, err)
}

func TestAsync_WriteAfterClose(t *testing.T) {
	var buf bytes.Buffer
	inner := &mockWriter{buf: &buf}

	w := Async(inner, 100)
	err := w.Close()
	require.NoError(t, err)

	// 关闭后写入应该返回成功但不写入数据
	n, err := w.Write([]byte("after close"))
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestAsync_ConcurrentWrite(t *testing.T) {
	var buf bytes.Buffer
	inner := &mockWriter{buf: &buf, mu: &sync.Mutex{}}

	w := Async(inner, 1000)
	defer func() { _ = w.Close() }()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = w.Write([]byte("x"))
		}()
	}
	wg.Wait()

	// 等待异步写入完成
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 100, len(buf.String()))
}

// ============ MultiWriter Tests ============

func TestMulti_Create(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	w1 := &mockWriter{buf: &buf1}
	w2 := &mockWriter{buf: &buf2}

	mw := Multi(w1, w2)
	require.NotNil(t, mw)

	err := mw.Close()
	require.NoError(t, err)
}

func TestMulti_Write(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	w1 := &mockWriter{buf: &buf1}
	w2 := &mockWriter{buf: &buf2}

	mw := Multi(w1, w2)
	defer func() { _ = mw.Close() }()

	n, err := mw.Write([]byte("test message"))
	require.NoError(t, err)
	assert.Equal(t, 12, n)

	assert.Equal(t, "test message", buf1.String())
	assert.Equal(t, "test message", buf2.String())
}

func TestMulti_Sync(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	w1 := &mockWriter{buf: &buf1}
	w2 := &mockWriter{buf: &buf2}

	mw := Multi(w1, w2)
	defer func() { _ = mw.Close() }()

	_, _ = mw.Write([]byte("test"))

	err := mw.Sync()
	assert.NoError(t, err)
}

func TestMulti_Close(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	w1 := &mockWriter{buf: &buf1}
	w2 := &mockWriter{buf: &buf2}

	mw := Multi(w1, w2)

	err := mw.Close()
	require.NoError(t, err)

	assert.True(t, w1.closed)
	assert.True(t, w2.closed)
}

func TestMulti_Empty(t *testing.T) {
	mw := Multi()
	require.NotNil(t, mw)

	n, err := mw.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)

	err = mw.Close()
	assert.NoError(t, err)
}

// ============ Helper: mockWriter ============

type mockWriter struct {
	buf    *bytes.Buffer
	mu     *sync.Mutex
	closed bool
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	if m.mu != nil {
		m.mu.Lock()
		defer m.mu.Unlock()
	}
	return m.buf.Write(p)
}

func (m *mockWriter) Close() error {
	m.closed = true
	return nil
}

func (m *mockWriter) Sync() error {
	return nil
}
