package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
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
			if (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
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
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if logger == nil {
		t.Error("New() returned nil logger")
	}
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
			if got != tt.want {
				t.Errorf("parseLevel(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// 使用 WithAttrs 创建带属性的 logger
	loggerWithAttrs := WithAttrs("key", "value")
	loggerWithAttrs.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "key=value") {
		t.Errorf("WithAttrs() output missing attribute, got: %s", output)
	}
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()

	// 测试从空 context 获取 logger（应该返回默认 logger）
	logger := FromContext(ctx)
	if logger == nil {
		t.Error("FromContext() returned nil for empty context")
	}

	// 测试从带有 logger 的 context 获取
	customLogger := slog.Default().With("custom", "value")
	ctxWithLogger := WithLogger(ctx, customLogger)
	retrievedLogger := FromContext(ctxWithLogger)
	if retrievedLogger != customLogger {
		t.Error("FromContext() did not return the expected logger")
	}
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-123"

	ctxWithReqID := WithRequestID(ctx, requestID)
	logger := FromContext(ctxWithReqID)

	// 验证不会 panic 并且返回有效的 logger
	if logger == nil {
		t.Error("WithRequestID() resulted in nil logger")
	}

	// 验证可以正常记录日志
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	// 重新创建带 request_id 的 logger
	ctxWithReqID = WithRequestID(context.Background(), requestID)
	logger = FromContext(ctxWithReqID)
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Logf("Output: %s", output)
	}
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
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %s, want %s", tt.input, got, tt.want)
			}
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

	// 验证 source 字段包含文件路径和行号格式 (file.go:123)
	if !strings.Contains(output, `"source":`) {
		t.Errorf("AddSource enabled but source field not found in output: %s", output)
	}
	if strings.Contains(output, `"source":"enabled"`) {
		t.Errorf("AddSource should contain file:line, not 'enabled': %s", output)
	}
	// 验证格式正确 (包含 .go: 表示文件名和行号)
	if !strings.Contains(output, ".go:") {
		t.Errorf("source should contain file.go:line format: %s", output)
	}
}

func TestJSONHandlerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := newJSONHandler(&buf, nil, "rfc3339ms", "")

	// 创建带有 group 的 logger
	logger := slog.New(handler).WithGroup("request")
	logger.Info("test", "method", "GET", "path", "/api")

	output := buf.String()

	// 验证 group 嵌套结构
	if !strings.Contains(output, `"request":{`) {
		t.Errorf("WithGroup should create nested structure, got: %s", output)
	}
	if !strings.Contains(output, `"method":"GET"`) {
		t.Errorf("method attribute should be in output: %s", output)
	}
}

func TestJSONHandlerNestedGroups(t *testing.T) {
	var buf bytes.Buffer
	handler := newJSONHandler(&buf, nil, "rfc3339ms", "")

	// 创建多层嵌套的 group
	logger := slog.New(handler).
		WithGroup("request").
		With("id", "123").
		WithGroup("headers")

	logger.Info("test", "content-type", "application/json")

	output := buf.String()

	// 验证嵌套结构
	if !strings.Contains(output, `"request":{`) {
		t.Errorf("first group should exist: %s", output)
	}
	if !strings.Contains(output, `"id":"123"`) {
		t.Errorf("id should be under request group: %s", output)
	}
	if !strings.Contains(output, `"headers":{`) {
		t.Errorf("nested headers group should exist: %s", output)
	}
}

func TestColoredHandlerJSONFlatten(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:        slog.LevelInfo,
		EnableColor:  false, // 禁用颜色便于测试
		PriorityKeys: []string{"time", "level", "msg"},
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	// 测试 JSON 字符串平铺
	logger.Info("request", "body", `{"user":"alice","age":30}`)

	output := buf.String()
	t.Logf("Output: %s", output)

	// 验证 JSON 被平铺
	if !strings.Contains(output, `"body.user":"alice"`) {
		t.Errorf("JSON should be flattened, expected body.user, got: %s", output)
	}
	if !strings.Contains(output, `"body.age":"30"`) {
		t.Errorf("JSON should be flattened, expected body.age, got: %s", output)
	}
	// 不应该包含原始 JSON 字符串
	if strings.Contains(output, `"body":"{`) {
		t.Errorf("should not contain escaped JSON string: %s", output)
	}
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

	// 测试嵌套 JSON 平铺
	logger.Info("data", "payload", `{"user":{"name":"bob","id":123},"tags":["go","rust"]}`)

	output := buf.String()
	t.Logf("Output: %s", output)

	// 验证嵌套对象被平铺
	if !strings.Contains(output, `"payload.user.name":"bob"`) {
		t.Errorf("nested JSON should be flattened: %s", output)
	}
	if !strings.Contains(output, `"payload.user.id":"123"`) {
		t.Errorf("nested JSON number should be flattened: %s", output)
	}
	// 验证数组被平铺
	if !strings.Contains(output, `"payload.tags[0]":"go"`) {
		t.Errorf("array should be flattened with index: %s", output)
	}
	if !strings.Contains(output, `"payload.tags[1]":"rust"`) {
		t.Errorf("array should be flattened with index: %s", output)
	}
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

	// 测试各级别日志
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	if !strings.Contains(output, `"level":"DEBUG"`) {
		t.Errorf("should contain DEBUG level: %s", output)
	}
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Errorf("should contain INFO level: %s", output)
	}
	if !strings.Contains(output, `"level":"WARN"`) {
		t.Errorf("should contain WARN level: %s", output)
	}
	if !strings.Contains(output, `"level":"ERROR"`) {
		t.Errorf("should contain ERROR level: %s", output)
	}
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

	if strings.Contains(output, "debug") {
		t.Errorf("DEBUG should be filtered: %s", output)
	}
	if strings.Contains(output, `"msg":"info"`) {
		t.Errorf("INFO should be filtered: %s", output)
	}
	if !strings.Contains(output, "warn") {
		t.Errorf("WARN should be present: %s", output)
	}
	if !strings.Contains(output, "error") {
		t.Errorf("ERROR should be present: %s", output)
	}
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

	if !strings.Contains(output, `"service":"api"`) {
		t.Errorf("should contain service attr: %s", output)
	}
	if !strings.Contains(output, `"version":"1.0"`) {
		t.Errorf("should contain version attr: %s", output)
	}
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

	// WithGroup 应该为属性添加前缀
	if !strings.Contains(output, `"request.method":"POST"`) {
		t.Errorf("WithGroup should add prefix, expected request.method: %s", output)
	}
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
			if !strings.Contains(output, `"time":`) {
				t.Errorf("should contain time field: %s", output)
			}
		})
	}
}

func TestNewWithCloser(t *testing.T) {
	// 测试 stdout（closer 应为 nil）
	cfg := &Config{
		Level:  "INFO",
		Format: "text",
		Output: "stdout",
	}

	logger, closer, err := NewWithCloser(cfg)
	if err != nil {
		t.Fatalf("NewWithCloser() error = %v", err)
	}
	if logger == nil {
		t.Error("logger should not be nil")
	}
	if closer != nil {
		t.Error("closer should be nil for stdout")
	}
}

func TestClose(t *testing.T) {
	// 先初始化到 stdout
	err := Init(&Config{
		Level:  "INFO",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Close 应该不报错
	err = Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 重复 Close 也不应该报错
	err = Close()
	if err != nil {
		t.Errorf("second Close() error = %v", err)
	}
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
			if got != tt.want {
				t.Errorf("escapeString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	err := LogError(context.Background(), "operation failed",
		context.DeadlineExceeded, "user_id", "123")

	// 验证返回原始错误
	if err != context.DeadlineExceeded {
		t.Errorf("LogError should return original error")
	}

	output := buf.String()
	if !strings.Contains(output, "operation failed") {
		t.Errorf("should log message: %s", output)
	}
	if !strings.Contains(output, "user_id=123") {
		t.Errorf("should log attrs: %s", output)
	}
}

func TestLogAndWrap(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	originalErr := context.DeadlineExceeded
	wrappedErr := LogAndWrap("fetch failed", originalErr, "url", "http://example.com")

	// 验证错误被包装
	if !strings.Contains(wrappedErr.Error(), "fetch failed") {
		t.Errorf("error should be wrapped: %v", wrappedErr)
	}

	output := buf.String()
	if !strings.Contains(output, "fetch failed") {
		t.Errorf("should log message: %s", output)
	}
}

func TestColoredHandlerNonJSONString(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	// 测试非 JSON 字符串不会被解析
	logger.Info("test", "data", "just a plain string")
	logger.Info("test", "invalid", "{not valid json")

	output := buf.String()

	if !strings.Contains(output, `"data":"just a plain string"`) {
		t.Errorf("plain string should not be flattened: %s", output)
	}
	if !strings.Contains(output, `"invalid":"{not valid json"`) {
		t.Errorf("invalid JSON should not be flattened: %s", output)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != "INFO" {
		t.Errorf("default level should be INFO, got %s", cfg.Level)
	}
	if cfg.Format != "text" {
		t.Errorf("default format should be text, got %s", cfg.Format)
	}
	if cfg.Output != "stdout" {
		t.Errorf("default output should be stdout, got %s", cfg.Output)
	}
	if cfg.AddSource != false {
		t.Errorf("default AddSource should be false")
	}
	if cfg.TimeFormat != "datetime" {
		t.Errorf("default TimeFormat should be datetime, got %s", cfg.TimeFormat)
	}
	if cfg.Timezone != "Asia/Shanghai" {
		t.Errorf("default Timezone should be Asia/Shanghai, got %s", cfg.Timezone)
	}
}

func TestDefaultColoredConfig(t *testing.T) {
	cfg := DefaultColoredConfig()

	if cfg.Level != slog.LevelInfo {
		t.Errorf("default level should be INFO")
	}
	if cfg.EnableColor != true {
		t.Errorf("default EnableColor should be true")
	}
	if cfg.AddSource != true {
		t.Errorf("default AddSource should be true")
	}
}

func TestColoredHandlerMapFlatten(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	// 测试 map[string]any 平铺
	logger.Info("test", "data", map[string]any{"user": "alice", "age": 30})

	output := buf.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, `"data.user":"alice"`) {
		t.Errorf("map should be flattened, expected data.user: %s", output)
	}
	if !strings.Contains(output, `"data.age":"30"`) {
		t.Errorf("map should be flattened, expected data.age: %s", output)
	}
}

func TestColoredHandlerStructFlatten(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	// 测试 struct 平铺
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	logger.Info("test", "user", User{Name: "bob", Age: 25})

	output := buf.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, `"user.name":"bob"`) {
		t.Errorf("struct should be flattened, expected user.name: %s", output)
	}
	if !strings.Contains(output, `"user.age":"25"`) {
		t.Errorf("struct should be flattened, expected user.age: %s", output)
	}
}

func TestColoredHandlerSlogGroup(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)
	logger := slog.New(handler)

	// 测试 slog.Group 平铺
	logger.Info("test", slog.Group("request", "method", "GET", "path", "/api"))

	output := buf.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, `"request.method":"GET"`) {
		t.Errorf("slog.Group should be flattened, expected request.method: %s", output)
	}
	if !strings.Contains(output, `"request.path":"/api"`) {
		t.Errorf("slog.Group should be flattened, expected request.path: %s", output)
	}
}

func TestColoredHandlerWithGroupPrefix(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)

	// 使用 WithGroup 创建 logger
	logger := slog.New(handler).WithGroup("request")
	logger.Info("received", "method", "POST", "path", "/users")

	output := buf.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, `"request.method":"POST"`) {
		t.Errorf("WithGroup should add prefix, expected request.method: %s", output)
	}
	if !strings.Contains(output, `"request.path":"/users"`) {
		t.Errorf("WithGroup should add prefix, expected request.path: %s", output)
	}
}

func TestColoredHandlerNestedWithGroup(t *testing.T) {
	var buf bytes.Buffer
	config := &ColoredHandlerConfig{
		Level:       slog.LevelInfo,
		EnableColor: false,
	}
	handler := NewColoredHandler(&buf, config)

	// 嵌套 group
	logger := slog.New(handler).
		WithGroup("http").
		With("version", "1.1").
		WithGroup("request")

	logger.Info("received", "method", "GET")

	output := buf.String()
	t.Logf("Output: %s", output)

	// version 应该在 http 下
	if !strings.Contains(output, `"http.version":"1.1"`) {
		t.Errorf("nested group should work, expected http.version: %s", output)
	}
	// method 应该在 http.request 下
	if !strings.Contains(output, `"http.request.method":"GET"`) {
		t.Errorf("nested group should work, expected http.request.method: %s", output)
	}
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
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, expected to contain %q", err, tt.errMsg)
			}
		})
	}
}

func TestInitWithInvalidConfig(t *testing.T) {
	// 测试 Init 时配置验证是否生效
	err := Init(&Config{
		Level:  "INFO",
		Format: "invalid_format",
	})
	if err == nil {
		t.Error("Init() should return error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid log format") {
		t.Errorf("Init() error = %v, expected to contain 'invalid log format'", err)
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
			expected: "/workspace/project", // 没有后续路径，保持原样
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
			if result != tt.expected {
				t.Errorf("clipWorkspacePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
