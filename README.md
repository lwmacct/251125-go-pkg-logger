# logm

基于 Go 1.21+ `log/slog` 的结构化日志库。

## 为什么选择 logm

- **零侵入设计**：初始化后可直接使用标准库 `slog.Info()` 等函数，业务代码无需依赖 logm
- **可插拔架构**：Handler + Formatter + Writer 分离，按需组合
- **生产就绪**：支持日志轮转、异步写入、动态级别调整

## 安装

```bash
go get github.com/lwmacct/251219-go-pkg-logm
```

## 快速开始

```go
package main

import (
    "log/slog"
    "github.com/lwmacct/251219-go-pkg-logm/pkg/logm"
)

func main() {
    // 一次性初始化（失败时 panic）
    logm.MustInit(logm.Development()...)
    defer logm.Close()

    // 之后直接使用标准库 slog
    slog.Info("started", "port", 8080)
}
```

## 文档

完整 API 文档和使用示例：

```bash
go doc github.com/lwmacct/251219-go-pkg-logm/pkg/logm
go doc github.com/lwmacct/251219-go-pkg-logm/pkg/logm/formatter
go doc github.com/lwmacct/251219-go-pkg-logm/pkg/logm/writer
```

或在线查看：[pkg.go.dev](https://pkg.go.dev/github.com/lwmacct/251219-go-pkg-logm/pkg/logm)
