# Logger 包

基于 Go 1.21+ `log/slog` 的统一结构化日志系统。

## 特性

- 支持多种输出格式：JSON、Text、Colored（彩色终端）
- 灵活的日志级别控制（DEBUG、INFO、WARN、ERROR）
- 环境自动检测（开发/生产环境智能默认值）
- 多种时间格式配置
- Context 集成，支持请求链路追踪
- 彩色 Handler 支持 JSON/map/struct 自动平铺
- WithGroup 分组支持
- 配置验证
- 文件输出支持

## 初始化 API

| 函数 | 说明 |
|------|------|
| `InitEnv()` | 从环境变量初始化（推荐），自动检测 `IS_SANDBOX` 选择开发/生产配置 |
| `InitCfg(cfg)` | 手动配置初始化 |
| `EnvConfig()` | 返回根据环境变量生成的默认 `*Config`，可修改后传给 `InitCfg` |
| `Close()` | 关闭资源（文件输出时必须调用） |

## 环境变量

| 变量 | 说明 | 开发环境默认 | 生产环境默认 |
|------|------|-------------|-------------|
| `IS_SANDBOX` | 环境检测 (1/true 为开发) | - | - |
| `LOG_LEVEL` | DEBUG, INFO, WARN, ERROR | DEBUG | INFO |
| `LOG_FORMAT` | json, text, color | color | json |
| `LOG_OUTPUT` | stdout, stderr, 文件路径 | stdout | stdout |
| `LOG_ADD_SOURCE` | true, false | true | false |
| `LOG_TIME_FORMAT` | 见时间格式表 | time | datetime |

## 时间格式

| 格式 | 示例 |
|------|------|
| `time` | `10:30:00` |
| `timems` | `10:30:00.123` |
| `datetime` | `2024-01-15 10:30:00` |
| `rfc3339` | `2024-01-15T10:30:00+08:00` |
| `rfc3339ms` | `2024-01-15T10:30:00.123+08:00` |
| 自定义 | Go 时间格式字符串 |

## Config 配置项

| 字段 | 类型 | 说明 |
|------|------|------|
| `Level` | string | 日志级别 |
| `Format` | string | 输出格式 |
| `Output` | string | 输出目标 |
| `AddSource` | bool | 显示源码位置 |
| `TimeFormat` | string | 时间格式 |
| `Timezone` | string | 时区（默认 Asia/Shanghai） |

## 示例

完整示例请参考 [main.go](../../main.go)。

## 测试

```bash
go test ./pkg/logger/... -v
```
