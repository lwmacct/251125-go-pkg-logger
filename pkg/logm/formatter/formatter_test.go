package formatter

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试用的固定时间
var testTime = time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

func newTestRecord(msg string, attrs ...slog.Attr) *Record {
	return &Record{
		Time:    testTime,
		Level:   slog.LevelInfo,
		Message: msg,
		Attrs:   attrs,
	}
}

// ============ JSON Formatter Tests ============

func TestJSONFormatter_BasicOutput(t *testing.T) {
	f := JSON()
	r := newTestRecord("test message")

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"level":"INFO"`)
	assert.Contains(t, output, `"msg":"test message"`)
	assert.Contains(t, output, `"time":"`)
	assert.True(t, strings.HasSuffix(output, "\n"))
}

func TestJSONFormatter_WithAttrs(t *testing.T) {
	f := JSON()
	r := newTestRecord("test",
		slog.String("key", "value"),
		slog.Int("count", 42),
		slog.Bool("enabled", true),
	)

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"key":"value"`)
	assert.Contains(t, output, `"count":42`)
	assert.Contains(t, output, `"enabled":true`)
}

func TestJSONFormatter_WithGroups(t *testing.T) {
	f := JSON()
	r := &Record{
		Time:    testTime,
		Level:   slog.LevelInfo,
		Message: "test",
		Groups:  []string{"request"},
		Attrs:   []slog.Attr{slog.String("method", "GET")},
	}

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"request":{`)
	assert.Contains(t, output, `"method":"GET"`)
}

func TestJSONFormatter_WithSource(t *testing.T) {
	f := JSON()
	r := newTestRecord("test")
	r.Source = &slog.Source{
		Function: "main.handler",
		File:     "/app/main.go",
		Line:     42,
	}

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"source":"/app/main.go:42"`)
}

func TestJSONFormatter_TimeFormat(t *testing.T) {
	f := JSON(WithTimeFormat("rfc3339"), WithTimezone("UTC"))
	r := newTestRecord("test")

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"time":"2024-01-15T10:30:45Z"`)
}

func TestJSONFormatter_EscapesSpecialChars(t *testing.T) {
	f := JSON()
	r := newTestRecord(`message with "quotes" and \backslash`)

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `\"quotes\"`)
	assert.Contains(t, output, `\\backslash`)
}

func TestJSONFormatter_Levels(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected string
	}{
		{slog.LevelDebug, `"level":"DEBUG"`},
		{slog.LevelInfo, `"level":"INFO"`},
		{slog.LevelWarn, `"level":"WARN"`},
		{slog.LevelError, `"level":"ERROR"`},
	}

	f := JSON()
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			r := &Record{Time: testTime, Level: tt.level, Message: "test"}
			data, err := f.Format(r)
			require.NoError(t, err)
			assert.Contains(t, string(data), tt.expected)
		})
	}
}

// ============ Text Formatter Tests ============

func TestTextFormatter_BasicOutput(t *testing.T) {
	f := Text()
	r := newTestRecord("test message")

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, "time=")
	assert.Contains(t, output, "level=INFO")
	assert.Contains(t, output, "msg=")
	assert.True(t, strings.HasSuffix(output, "\n"))
}

func TestTextFormatter_WithAttrs(t *testing.T) {
	f := Text()
	r := newTestRecord("test",
		slog.String("key", "value"),
		slog.Int("count", 42),
	)

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, "key=value")
	assert.Contains(t, output, "count=42")
}

func TestTextFormatter_QuotesSpaces(t *testing.T) {
	f := Text()
	r := newTestRecord("message with spaces")

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `msg="message with spaces"`)
}

func TestTextFormatter_EscapesNewlines(t *testing.T) {
	f := Text()
	r := newTestRecord("line1\nline2")

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `\n`)
}

func TestTextFormatter_WithSource(t *testing.T) {
	f := Text()
	r := newTestRecord("test")
	r.Source = &slog.Source{File: "/app/main.go", Line: 42}

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, "source=/app/main.go:42")
}

// ============ ColorText Formatter Tests ============

func TestColorTextFormatter_BasicOutput(t *testing.T) {
	f := ColorText()
	r := newTestRecord("test message")

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "test message")
	assert.True(t, strings.HasSuffix(output, "\n"))
}

func TestColorTextFormatter_WithAttrs(t *testing.T) {
	f := ColorText()
	r := newTestRecord("test",
		slog.String("key", "value"),
	)

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	// Color formatter adds ANSI codes, so check for key and value separately
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
}

func TestColorTextFormatter_JSONFlatten(t *testing.T) {
	f := ColorText()
	r := newTestRecord("test",
		slog.String("data", `{"user":"alice","age":30}`),
	)

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	// JSON 字符串应该被展开
	assert.Contains(t, output, "data.user")
	assert.Contains(t, output, "alice")
}

func TestColorTextFormatter_LevelColors(t *testing.T) {
	tests := []struct {
		level slog.Level
		text  string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
	}

	f := ColorText()
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			r := &Record{Time: testTime, Level: tt.level, Message: "test"}
			data, err := f.Format(r)
			require.NoError(t, err)
			assert.Contains(t, string(data), tt.text)
		})
	}
}

// ============ formatTime Tests ============

func TestFormatTime(t *testing.T) {
	tm := time.Date(2024, 1, 15, 10, 30, 45, 123000000, time.UTC)

	tests := []struct {
		format   string
		expected string
	}{
		{"time", "10:30:45"},
		{"timems", "10:30:45.123"},
		{"datetime", "2024-01-15 10:30:45"},
		{"rfc3339", "2024-01-15T10:30:45Z"},
		{"", "2024-01-15 10:30:45"},  // default
		{"2006/01/02", "2024/01/15"}, // custom format
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := formatTime(tm, tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============ loadTimezone Tests ============

func TestLoadTimezone(t *testing.T) {
	tests := []struct {
		tz       string
		expected string
	}{
		{"UTC", "UTC"},
		{"Asia/Shanghai", "Asia/Shanghai"},
		{"", "Local"},             // empty returns Local
		{"Invalid/Zone", "Local"}, // invalid returns Local
	}

	for _, tt := range tests {
		t.Run(tt.tz, func(t *testing.T) {
			loc := loadTimezone(tt.tz)
			if tt.expected == "Local" {
				assert.Equal(t, time.Local, loc)
			} else {
				assert.Equal(t, tt.expected, loc.String())
			}
		})
	}
}

// ============ Options Tests ============

func TestWithTimeFormat(t *testing.T) {
	opts := defaultOptions()
	WithTimeFormat("rfc3339")(opts)
	assert.Equal(t, "rfc3339", opts.TimeFormat)
}

func TestWithTimezone(t *testing.T) {
	opts := defaultOptions()
	WithTimezone("UTC")(opts)
	assert.Equal(t, "UTC", opts.Location.String())
}

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()
	assert.Equal(t, "datetime", opts.TimeFormat)
	assert.Equal(t, time.Local, opts.Location)
}

// ============ Source Clip Tests ============

func TestSourceClip_ClipToDepth(t *testing.T) {
	tests := []struct {
		path     string
		depth    int
		expected string
	}{
		{"/a/b/c/d.go", 3, "b/c/d.go"},
		{"/a/b/c/d.go", 2, "c/d.go"},
		{"/a/b/c/d.go", 1, "d.go"},
		{"/a/b/c/d.go", 5, "/a/b/c/d.go"}, // 深度不足
		{"a/b/c.go", 2, "b/c.go"},
		{"file.go", 3, "file.go"}, // 无斜杠
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := clipToDepth(tt.path, tt.depth)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSourceClip_ClipPrefix(t *testing.T) {
	tests := []struct {
		path     string
		prefix   string
		expected string
	}{
		{"/workspace/myproject/pkg/main.go", "/workspace/", "pkg/main.go"},
		{"/apps/data/workspace/project/src/main.go", "/apps/data/workspace/", "src/main.go"},
		{"/other/path/file.go", "/workspace/", "/other/path/file.go"}, // 无匹配
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := clipPrefix(tt.path, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSourceClip_FormatSource(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		line     int
		opts     *Options
		expected string
	}{
		{
			name:     "no clip",
			file:     "/a/b/c/d.go",
			line:     42,
			opts:     &Options{},
			expected: "b/c/d.go:42", // 默认裁剪到 3 层
		},
		{
			name:     "with prefix clip",
			file:     "/workspace/project/internal/server/handler.go",
			line:     100,
			opts:     &Options{SourceClip: "/workspace/"},
			expected: "internal/server/handler.go:100",
		},
		{
			name:     "with depth",
			file:     "/a/b/c/d/e/f.go",
			line:     1,
			opts:     &Options{SourceDepth: 2},
			expected: "e/f.go:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &slog.Source{File: tt.file, Line: tt.line}
			result := FormatSource(source, tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONFormatter_WithSourceClip(t *testing.T) {
	f := JSON(WithSourceClip("/workspace/"), WithSourceDepth(3))
	r := newTestRecord("test")
	r.Source = &slog.Source{
		File: "/workspace/project/internal/handler.go",
		Line: 42,
	}

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"source":"internal/handler.go:42"`)
	assert.NotContains(t, output, "/workspace/")
}

// ============ ColorJSON Formatter Tests ============

func TestColorJSONFormatter_BasicOutput(t *testing.T) {
	f := ColorJSON()
	r := newTestRecord("test message")

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"level":"`)
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, `"msg":"`)
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, `"time":"`)
	assert.True(t, strings.HasPrefix(output, "{"))
	assert.True(t, strings.HasSuffix(output, "}\n"))
}

func TestColorJSONFormatter_WithAttrs(t *testing.T) {
	f := ColorJSON()
	r := newTestRecord("test",
		slog.String("key", "value"),
		slog.Int("count", 42),
		slog.Bool("enabled", true),
	)

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"key":"`)
	assert.Contains(t, output, "value")
	assert.Contains(t, output, `"count":`)
	assert.Contains(t, output, "42")
	assert.Contains(t, output, `"enabled":`)
	assert.Contains(t, output, "true")
}

func TestColorJSONFormatter_WithSource(t *testing.T) {
	f := ColorJSON(WithSourceClip("/workspace/"))
	r := newTestRecord("test")
	r.Source = &slog.Source{
		File: "/workspace/project/pkg/main.go",
		Line: 42,
	}

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"source":"`)
	assert.Contains(t, output, "pkg/main.go:42")
}

func TestColorJSONFormatter_LevelColors(t *testing.T) {
	tests := []struct {
		level slog.Level
		text  string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
	}

	f := ColorJSON()
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			r := &Record{Time: testTime, Level: tt.level, Message: "test"}
			data, err := f.Format(r)
			require.NoError(t, err)
			assert.Contains(t, string(data), tt.text)
		})
	}
}

func TestColorJSONFormatter_DisableColor(t *testing.T) {
	f := ColorJSON().DisableColor()
	r := newTestRecord("test")

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	// 不应包含 ANSI 转义序列
	assert.NotContains(t, output, "\033[")
}

func TestColorJSONFormatter_WithGroups(t *testing.T) {
	f := ColorJSON()
	r := &Record{
		Time:    testTime,
		Level:   slog.LevelInfo,
		Message: "test",
		Groups:  []string{"request"},
		Attrs:   []slog.Attr{slog.String("method", "GET")},
	}

	data, err := f.Format(r)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"request":{`)
	assert.Contains(t, output, `"method":"`)
}

// ============ WithSourceClip/WithSourceDepth Tests ============

func TestWithSourceClip(t *testing.T) {
	opts := defaultOptions()
	WithSourceClip("/workspace/")(opts)
	assert.Equal(t, "/workspace/", opts.SourceClip)
}

func TestWithSourceDepth(t *testing.T) {
	opts := defaultOptions()
	WithSourceDepth(2)(opts)
	assert.Equal(t, 2, opts.SourceDepth)
}
