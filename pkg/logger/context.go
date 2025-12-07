package logger

import (
	"context"
	"log/slog"
)

// contextKey 是用于 context 中存储 logger 的键类型
type contextKey struct{}

var loggerKey = contextKey{}

// WithLogger 将 logger 存入 context
//
// 用于在请求处理链路中传递带有特定上下文信息的 logger
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext 从 context 中获取 logger
//
// 如果 context 中没有 logger，则返回全局默认 logger
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// WithRequestID 创建带有请求 ID 的 logger 并存入 context
//
// 常用于 HTTP 请求处理，用于追踪单个请求的日志
func WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := FromContext(ctx).With("request_id", requestID)
	return WithLogger(ctx, logger)
}
