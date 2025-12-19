package formatter

import (
	"log/slog"
	"strconv"
	"strings"
)

// FormatSource 格式化 Source 为 "file:line" 字符串，并应用路径裁剪。
func FormatSource(source *slog.Source, opts *Options) string {
	if source == nil {
		return ""
	}
	path := clipPath(source.File, opts)
	return path + ":" + strconv.Itoa(source.Line)
}

// clipPath 裁剪路径。
//
// 裁剪规则：
//  1. 优先移除指定前缀（如 /workspace/xxx/）
//  2. 如果路径深度超过目标深度，保留最后 N 层
func clipPath(path string, opts *Options) string {
	if opts == nil {
		return path
	}

	originalPath := path

	// 1. 处理指定前缀（如 /workspace/）
	if opts.SourceClip != "" {
		path = clipPrefix(path, opts.SourceClip)
	}

	// 2. 计算目标深度
	targetDepth := opts.SourceDepth
	if targetDepth <= 0 {
		targetDepth = 3 // 默认保留 3 层
	}

	// 3. 如果前缀裁剪成功（路径变短），检查深度
	if path != originalPath {
		depth := countDepth(path)
		if depth <= targetDepth {
			return path
		}
		return clipToDepth(path, targetDepth)
	}

	// 4. 默认裁剪：对绝对路径保留最后 N 层
	if len(path) > 0 && path[0] == '/' {
		return clipToDepth(path, targetDepth)
	}

	return path
}

// clipPrefix 裁剪指定前缀。
//
// 支持 /workspace/ 类型前缀，会同时移除前缀后的第一层目录。
// 例如：clipPrefix("/workspace/myproject/pkg/main.go", "/workspace/") -> "pkg/main.go"
func clipPrefix(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return path
	}

	path = path[len(prefix):]

	// 如果前缀以 / 结尾，再跳过项目名目录
	if strings.HasSuffix(prefix, "/") {
		if idx := strings.Index(path, "/"); idx > 0 {
			path = path[idx+1:]
		}
	}

	return path
}

// countDepth 计算路径深度。
func countDepth(path string) int {
	if path == "" {
		return 0
	}
	return strings.Count(path, "/") + 1
}

// clipToDepth 保留路径的最后 n 层。
//
// 例如：clipToDepth("/a/b/c/d.go", 3) -> "b/c/d.go"
func clipToDepth(path string, depth int) string {
	if depth <= 0 {
		return path
	}

	count := 0
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			count++
			if count == depth {
				if i+1 < len(path) {
					return path[i+1:]
				}
				return path[i:]
			}
		}
	}

	// 深度不足，返回原路径
	return path
}
