package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "json format",
			config: &Config{
				Level:  "INFO",
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "debug level",
			config: &Config{
				Level:  "DEBUG",
				Format: "text",
				Output: "stdout",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	cfg := &Config{
		Level:  "INFO",
		Format: "text",
		Output: "stdout",
	}

	logger, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, logger)
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
			got := parseLevel(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	loggerWithAttrs := WithAttrs("key", "value")
	loggerWithAttrs.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "key=value")
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

	// 验证可以正常记录日志
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	ctxWithReqID = WithRequestID(context.Background(), requestID)
	logger = FromContext(ctxWithReqID)
	logger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "test message")
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

func TestJSONHandlerAddSource(t *testing.T) {
	var buf bytes.Buffer
	handler := newJSONHandler(&buf, &slog.HandlerOptions{
		AddSource: true,
	}, "rfc3339ms", "")

	logger := slog.New(handler)
	logger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, `"source":`, "AddSource enabled but source field not found")
	assert.NotContains(t, output, `"source":"enabled"`, "AddSource should contain file:line, not 'enabled'")
	assert.Contains(t, output, ".go:", "source should contain file.go:line format")
}

func TestJSONHandlerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := newJSONHandler(&buf, nil, "rfc3339ms", "")

	logger := slog.New(handler).WithGroup("request")
	logger.Info("test", "method", "GET", "path", "/api")

	output := buf.String()
	assert.Contains(t, output, `"request":{`, "WithGroup should create nested structure")
	assert.Contains(t, output, `"method":"GET"`)
}

func TestJSONHandlerNestedGroups(t *testing.T) {
	var buf bytes.Buffer
	handler := newJSONHandler(&buf, nil, "rfc3339ms", "")

	logger := slog.New(handler).
		WithGroup("request").
		With("id", "123").
		WithGroup("headers")

	logger.Info("test", "content-type", "application/json")

	output := buf.String()
	assert.Contains(t, output, `"request":{`, "first group should exist")
	assert.Contains(t, output, `"id":"123"`, "id should be under request group")
	assert.Contains(t, output, `"headers":{`, "nested headers group should exist")
}

func TestColoredHandlerJSONFlatten(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:        slog.LevelInfo,
		EnableColor:  false,
		PriorityKeys: []string{"time", "level", "msg"},
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	logger.Info("request", "body", `{"user":"alice","age":30}`)

	output := buf.String()
	t.Logf("Output: %s", output)

	assert.Contains(t, output, `"body.user":"alice"`, "JSON should be flattened")
	assert.Contains(t, output, `"body.age":"30"`, "JSON should be flattened")
	assert.NotContains(t, output, `"body":"{`, "should not contain escaped JSON string")
}

func TestColoredHandlerNestedJSONFlatten(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:        slog.LevelInfo,
		EnableColor:  false,
		PriorityKeys: []string{"time", "level", "msg"},
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	logger.Info("data", "payload", `{"user":{"name":"bob","id":123},"tags":["go","rust"]}`)

	output := buf.String()
	t.Logf("Output: %s", output)

	assert.Contains(t, output, `"payload.user.name":"bob"`, "nested JSON should be flattened")
	assert.Contains(t, output, `"payload.user.id":"123"`, "nested JSON number should be flattened")
	assert.Contains(t, output, `"payload.tags[0]":"go"`, "array should be flattened with index")
	assert.Contains(t, output, `"payload.tags[1]":"rust"`, "array should be flattened with index")
}

func TestColoredHandlerBasic(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:        slog.LevelDebug,
		EnableColor:  false,
		PriorityKeys: []string{"time", "level", "msg"},
		TrailingKeys: []string{"source"},
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.Contains(t, output, `"level":"DEBUG"`)
	assert.Contains(t, output, `"level":"INFO"`)
	assert.Contains(t, output, `"level":"WARN"`)
	assert.Contains(t, output, `"level":"ERROR"`)
}

func TestColoredHandlerLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelWarn, // 只显示 WARN 及以上
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	output := buf.String()
	assert.NotContains(t, output, "debug", "DEBUG should be filtered")
	assert.NotContains(t, output, `"msg":"info"`, "INFO should be filtered")
	assert.Contains(t, output, "warn", "WARN should be present")
	assert.Contains(t, output, "error", "ERROR should be present")
}

func TestColoredHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler).With("service", "api", "version", "1.0")

	logger.Info("started")

	output := buf.String()
	assert.Contains(t, output, `"service":"api"`)
	assert.Contains(t, output, `"version":"1.0"`)
}

func TestColoredHandlerWithGroupBasic(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler).WithGroup("request")

	logger.Info("received", "method", "POST")

	output := buf.String()
	t.Logf("Output: %s", output)
	assert.Contains(t, output, `"request.method":"POST"`, "WithGroup should add prefix")
}

func TestJSONHandlerTimeFormats(t *testing.T) {
	tests := []struct {
		format   string
		contains string
	}{
		{"rfc3339", "T"},
		{"rfc3339ms", "."},
		{"datetime", " "},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var buf bytes.Buffer
			handler := newJSONHandler(&buf, nil, tt.format, "")
			logger := slog.New(handler)

			logger.Info("test")

			output := buf.String()
			assert.Contains(t, output, `"time":`)
		})
	}
}

func TestNewWithCloser(t *testing.T) {
	cfg := &Config{
		Level:  "INFO",
		Format: "text",
		Output: "stdout",
	}

	logger, closer, err := NewWithCloser(cfg)
	require.NoError(t, err)
	assert.NotNil(t, logger)
	assert.Nil(t, closer, "closer should be nil for stdout")
}

func TestClose(t *testing.T) {
	err := Init(&Config{
		Level:  "INFO",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	// Close 应该不报错
	err = Close()
	assert.NoError(t, err)

	// 重复 Close 也不应该报错
	err = Close()
	assert.NoError(t, err)
}

func TestEscapeString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{`say "hi"`, `say \"hi\"`},
		{"path\\to\\file", `path\\to\\file`},
		{"line1\nline2", `line1\nline2`},
		{"tab\there", `tab\there`},
		{"normal text", "normal text"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeString(tt.input)
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

func TestColoredHandlerNonJSONString(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	logger.Info("test", "data", "just a plain string")
	logger.Info("test", "invalid", "{not valid json")

	output := buf.String()
	assert.Contains(t, output, `"data":"just a plain string"`, "plain string should not be flattened")
	assert.Contains(t, output, `"invalid":"{not valid json"`, "invalid JSON should not be flattened")
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "INFO", cfg.Level)
	assert.Equal(t, "text", cfg.Format)
	assert.Equal(t, "stdout", cfg.Output)
	assert.False(t, cfg.AddSource)
	assert.Equal(t, "datetime", cfg.TimeFormat)
	assert.Equal(t, "Asia/Shanghai", cfg.Timezone)
}

func TestDefaultColoredConfig(t *testing.T) {
	cfg := DefaultColoredConfig()

	assert.Equal(t, slog.LevelInfo, cfg.Level)
	assert.True(t, cfg.EnableColor)
	assert.True(t, cfg.AddSource)
}

func TestColoredHandlerMapFlatten(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	logger.Info("test", "data", map[string]any{"user": "alice", "age": 30})

	output := buf.String()
	t.Logf("Output: %s", output)
	assert.Contains(t, output, `"data.user":"alice"`, "map should be flattened")
	assert.Contains(t, output, `"data.age":"30"`, "map should be flattened")
}

func TestColoredHandlerStructFlatten(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	logger.Info("test", "user", User{Name: "bob", Age: 25})

	output := buf.String()
	t.Logf("Output: %s", output)
	assert.Contains(t, output, `"user.name":"bob"`, "struct should be flattened")
	assert.Contains(t, output, `"user.age":"25"`, "struct should be flattened")
}

func TestColoredHandlerSlogGroup(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	logger.Info("test", slog.Group("request", "method", "GET", "path", "/api"))

	output := buf.String()
	t.Logf("Output: %s", output)
	assert.Contains(t, output, `"request.method":"GET"`, "slog.Group should be flattened")
	assert.Contains(t, output, `"request.path":"/api"`, "slog.Group should be flattened")
}

func TestColoredHandlerWithGroupPrefix(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)

	logger := slog.New(handler).WithGroup("request")
	logger.Info("received", "method", "POST", "path", "/users")

	output := buf.String()
	t.Logf("Output: %s", output)
	assert.Contains(t, output, `"request.method":"POST"`, "WithGroup should add prefix")
	assert.Contains(t, output, `"request.path":"/users"`, "WithGroup should add prefix")
}

func TestColoredHandlerNestedWithGroup(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)

	logger := slog.New(handler).
		WithGroup("http").
		With("version", "1.1").
		WithGroup("request")

	logger.Info("received", "method", "GET")

	output := buf.String()
	t.Logf("Output: %s", output)
	assert.Contains(t, output, `"http.version":"1.1"`, "nested group should work")
	assert.Contains(t, output, `"http.request.method":"GET"`, "nested group should work")
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "valid json format",
			config: &Config{
				Level:  "INFO",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "valid color format",
			config: &Config{
				Level:  "DEBUG",
				Format: "color",
			},
			wantErr: false,
		},
		{
			name: "valid colored format alias",
			config: &Config{
				Level:  "WARN",
				Format: "colored",
			},
			wantErr: false,
		},
		{
			name: "case insensitive level",
			config: &Config{
				Level:  "debug",
				Format: "text",
			},
			wantErr: false,
		},
		{
			name: "invalid format",
			config: &Config{
				Level:  "INFO",
				Format: "yaml",
			},
			wantErr: true,
			errMsg:  "invalid log format",
		},
		{
			name: "invalid level",
			config: &Config{
				Level:  "TRACE",
				Format: "json",
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "empty format is valid",
			config: &Config{
				Level:  "INFO",
				Format: "",
			},
			wantErr: false,
		},
		{
			name: "empty level is valid",
			config: &Config{
				Level:  "",
				Format: "json",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitWithInvalidConfig(t *testing.T) {
	err := Init(&Config{
		Level:  "INFO",
		Format: "invalid_format",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log format")
}

func TestTextHandlerTimeFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := newTextHandler(&buf, nil, "datetime", "Asia/Shanghai")
	logger := slog.New(handler)

	logger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, `time="20`, "should contain time field")

	// datetime 格式不应包含 RFC3339 的 T 分隔符
	idx := strings.Index(output, `time="`)
	if idx >= 0 {
		timeField := output[idx : idx+30]
		assert.NotContains(t, timeField, "T", "datetime format should not contain T separator")
	}
}

func TestTextHandlerTimeFormats(t *testing.T) {
	tests := []struct {
		format      string
		contains    string
		notContains string
	}{
		{"datetime", "2025-", "T"},
		{"rfc3339", "T", ""},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var buf bytes.Buffer
			handler := newTextHandler(&buf, nil, tt.format, "")
			logger := slog.New(handler)

			logger.Info("test")

			output := buf.String()
			t.Logf("Output: %s", output)

			assert.Contains(t, output, "time=", "should contain time field")

			if tt.contains != "" {
				assert.Contains(t, output, tt.contains)
			}
			if tt.notContains != "" {
				idx := strings.Index(output, "time=")
				endIdx := strings.Index(output[idx:], " level=")
				if endIdx < 0 {
					endIdx = len(output) - idx
				}
				timeField := output[idx : idx+endIdx]
				assert.NotContains(t, timeField, tt.notContains, "time field should not contain %s", tt.notContains)
			}
		})
	}
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
			input:    "/workspace/my-project/pkg/logger/logger.go:42",
			expected: "pkg/logger/logger.go:42",
		},
		{
			name:     "workspace path without trailing slash",
			input:    "/workspace/project",
			expected: "/workspace/project",
		},
		{
			name:     "no workspace prefix",
			input:    "/home/user/project/main.go:10",
			expected: "/home/user/project/main.go:10",
		},
		{
			name:     "workspace in middle of path",
			input:    "/apps/data/workspace/251125-go-mod-logger/pkg/logger/logger.go:100",
			expected: "pkg/logger/logger.go:100",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just workspace",
			input:    "/workspace/",
			expected: "/workspace/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clipWorkspacePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
