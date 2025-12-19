package logm

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter"
	"github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_Default(t *testing.T) {
	err := Init()
	assert.NoError(t, err)
	defer func() { _ = Close() }()

	// 验证可以正常记录日志
	slog.Info("test message")
}

func TestInit_WithLevel(t *testing.T) {
	err := Init(WithLevel("DEBUG"))
	assert.NoError(t, err)
	defer func() { _ = Close() }()
}

func TestInit_Development(t *testing.T) {
	err := Init(Development()...)
	assert.NoError(t, err)
	defer func() { _ = Close() }()
}

func TestInit_Production(t *testing.T) {
	err := Init(Production()...)
	assert.NoError(t, err)
	defer func() { _ = Close() }()
}

func TestMustInit_Success(t *testing.T) {
	// MustInit 成功时不应 panic
	assert.NotPanics(t, func() {
		MustInit(WithLevel("INFO"))
	})
	defer func() { _ = Close() }()
}

func TestNew_ReturnsLogger(t *testing.T) {
	log := New(WithLevel("INFO"))
	assert.NotNil(t, log)
}

func TestNew_WithFormatter(t *testing.T) {
	log := New(
		WithFormatter(formatter.JSON()),
		WithLevel("DEBUG"),
	)
	assert.NotNil(t, log)
}

func TestNew_WithWriter(t *testing.T) {
	w := writer.Stdout()
	log := New(WithWriter(w))
	assert.NotNil(t, log)
}

func TestSetLevel(t *testing.T) {
	err := Init(WithLevel("INFO"))
	require.NoError(t, err)
	defer func() { _ = Close() }()

	// 动态调整级别
	SetLevel("DEBUG")
	// 验证可以正常记录 DEBUG 日志
	slog.Debug("debug message")

	SetLevel("ERROR")
	// 验证 INFO 级别被过滤
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"UNKNOWN", slog.LevelInfo}, // default
		// 小写支持
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		// 混合大小写
		{"Debug", slog.LevelDebug},
		{"Info", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseLevel(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()

	// 测试从空 context 获取 logger（应该返回默认 logger）
	logger := FromContext(ctx)
	assert.NotNil(t, logger, "FromContext should not return nil for empty context")

	// 测试从带有 logger 的 context 获取
	customLogger := slog.Default().With("custom", "value")
	ctxWithLogger := WithLogger(ctx, customLogger)
	retrievedLogger := FromContext(ctxWithLogger)
	assert.Equal(t, customLogger, retrievedLogger)
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-123"

	ctxWithReqID := WithRequestID(ctx, requestID)
	logger := FromContext(ctxWithReqID)
	assert.NotNil(t, logger)
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatBytes(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	err := LogError(context.Background(), "operation failed",
		context.DeadlineExceeded, "user_id", "123")

	assert.Equal(t, context.DeadlineExceeded, err, "LogError should return original error")

	output := buf.String()
	assert.Contains(t, output, "operation failed")
	assert.Contains(t, output, "user_id=123")
}

func TestLogAndWrap(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	originalErr := context.DeadlineExceeded
	wrappedErr := LogAndWrap("fetch failed", originalErr, "url", "http://example.com")

	assert.Contains(t, wrappedErr.Error(), "fetch failed", "error should be wrapped")

	output := buf.String()
	assert.Contains(t, output, "fetch failed")
}

func TestClipWorkspacePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "workspace path with project dir",
			input:    "/workspace/251127-ai-agent-hatch/main.go:146",
			expected: "main.go:146",
		},
		{
			name:     "workspace path with nested dirs",
			input:    "/workspace/my-project/pkg/logm/logm.go:42",
			expected: "pkg/logm/logm.go:42",
		},
		{
			name:     "no workspace prefix",
			input:    "/home/user/project/main.go:10",
			expected: "/home/user/project/main.go:10",
		},
		{
			name:     "workspace in middle of path",
			input:    "/apps/data/workspace/251219-go-pkg-logm/pkg/logm/logm.go:100",
			expected: "pkg/logm/logm.go:100",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clipWorkspacePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	h := NewHandler(&HandlerConfig{
		Formatter: formatter.Text(),
		Writers:   []Writer{stdoutWriter},
	})

	logger := slog.New(h)
	logger.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
}

func TestHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	h := NewHandler(&HandlerConfig{
		Formatter: formatter.Text(),
		Writers:   []Writer{stdoutWriter},
	})

	logger := slog.New(h).With("service", "api")
	logger.Info("started")

	output := buf.String()
	assert.Contains(t, output, "service=api")
}

func TestHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	h := NewHandler(&HandlerConfig{
		Formatter: formatter.Text(),
		Writers:   []Writer{stdoutWriter},
	})

	logger := slog.New(h).WithGroup("request")
	logger.Info("received", "method", "POST")

	output := buf.String()
	assert.Contains(t, output, "request.")
	assert.Contains(t, output, "method=POST")
}

func TestHandler_LevelFilter(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	levelVar := &slog.LevelVar{}
	levelVar.Set(slog.LevelWarn)

	h := NewHandler(&HandlerConfig{
		LevelVar:  levelVar,
		Formatter: formatter.Text(),
		Writers:   []Writer{stdoutWriter},
	})

	logger := slog.New(h)
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	output := buf.String()
	assert.NotContains(t, output, "debug")
	assert.NotContains(t, output, `msg=info`)
	assert.Contains(t, output, "warn")
	assert.Contains(t, output, "error")
}

func TestHandler_Interceptor(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	// 拦截器：添加 trace_id
	interceptor := func(ctx context.Context, r *Record) *Record {
		r.Attrs = append(r.Attrs, slog.String("trace_id", "abc123"))
		return r
	}

	h := NewHandler(&HandlerConfig{
		Formatter:    formatter.Text(),
		Writers:      []Writer{stdoutWriter},
		Interceptors: []Interceptor{interceptor},
	})

	logger := slog.New(h)
	logger.Info("test")

	output := buf.String()
	assert.Contains(t, output, "trace_id=abc123")
}

func TestHandler_InterceptorFilter(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	// 拦截器：过滤包含 "secret" 的日志
	interceptor := func(ctx context.Context, r *Record) *Record {
		if strings.Contains(r.Message, "secret") {
			return nil // 丢弃
		}
		return r
	}

	h := NewHandler(&HandlerConfig{
		Formatter:    formatter.Text(),
		Writers:      []Writer{stdoutWriter},
		Interceptors: []Interceptor{interceptor},
	})

	logger := slog.New(h)
	logger.Info("normal message")
	logger.Info("secret data")

	output := buf.String()
	assert.Contains(t, output, "normal message")
	assert.NotContains(t, output, "secret")
}

func TestHandler_AddSource(t *testing.T) {
	var buf bytes.Buffer
	stdoutWriter := &testWriter{buf: &buf}

	h := NewHandler(&HandlerConfig{
		Formatter: formatter.Text(),
		Writers:   []Writer{stdoutWriter},
		AddSource: true,
	})

	logger := slog.New(h)
	logger.Info("test")

	output := buf.String()
	assert.Contains(t, output, "source=")
	assert.Contains(t, output, ".go:")
}

func TestDebugInfoWarnError(t *testing.T) {
	err := Init(WithLevel("DEBUG"))
	require.NoError(t, err)
	defer func() { _ = Close() }()

	// 验证这些函数可以正常调用
	Debug("debug message", "key", "value")
	Info("info message", "key", "value")
	Warn("warn message", "key", "value")
	Error("error message", "key", "value")
}

func TestWith(t *testing.T) {
	err := Init(WithLevel("INFO"))
	require.NoError(t, err)
	defer func() { _ = Close() }()

	log := With("module", "test")
	assert.NotNil(t, log)
}

func TestWithGroup(t *testing.T) {
	err := Init(WithLevel("INFO"))
	require.NoError(t, err)
	defer func() { _ = Close() }()

	log := WithGroup("request")
	assert.NotNil(t, log)
}

func TestClose_Multiple(t *testing.T) {
	err := Init()
	require.NoError(t, err)

	// Close 应该不报错
	err = Close()
	assert.NoError(t, err)

	// 重复 Close 也不应该报错
	err = Close()
	assert.NoError(t, err)
}

// testWriter 是一个简单的 Writer 实现用于测试
type testWriter struct {
	buf *bytes.Buffer
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *testWriter) Close() error {
	return nil
}

func (w *testWriter) Sync() error {
	return nil
}
