# 契约：Go 文件边界

## 目的

本契约把基础 LSP server 的责任分配到文件和目录，防止实现阶段把协议工作扩展为 Tops 语义实现或 clangd 集成。

## 文件归属

| 路径 | 允许内容 | 禁止内容 |
| --- | --- | --- |
| `go.mod` | Go module、Go 版本、必要依赖 | 未使用或语义相关依赖 |
| `cmd/tops-lsp/main.go` | 进程入口、stdin/stdout、日志初始化、退出状态 | JSON-RPC 解析、文档变更算法、Tops 规则 |
| `internal/transport/stdio.go` | `Content-Length` framing、EOF、读写错误 | 生命周期策略、文档状态、语义判断 |
| `internal/protocol/messages.go` | JSON-RPC/LSP 基础消息和最小参数模型 | 编译上下文、Tops AST、语义诊断 |
| `internal/protocol/errors.go` | 标准错误名、code 和错误响应 | 自定义语义错误、clangd 错误转发 |
| `internal/server/server.go` | 生命周期状态、方法路由、请求登记、取消、单响应 | tokenizer、parser、AST、symbol index、semantic |
| `internal/document/store.go` | URI、文本、版本、UTF-16 增量变更、open/close | C++ 语法、宏、include、target/profile |
| `internal/logging/logger.go` | 结构化字段、日志级别、关联 ID、脱敏、stderr | 完整 payload、源代码、凭据 |
| `internal/**/*_test.go` | 对应目录的单元和组件测试 | 真实 GCU、clangd、外部网络依赖 |
| `integration/lsp_process_test.go` | server 子进程、管道、退出状态、stdout/stderr 验证 | 语义回归和设备 workload |
| `testdata/lsp/` | 合成 LSP 帧、文档变更、错误序列 | 真实用户源代码和敏感配置 |

## 禁止新增的边界

本 feature 不创建以下目录或实现：

- `internal/tokenizer/`
- `internal/parser/`
- `internal/ast/`
- `internal/symbolindex/`
- `internal/semantic/`
- clangd adapter、clangd sidecar 或 compiler subprocess wrapper
- TypeScript VS Code extension
- TCP、WebSocket、HTTP 或自定义二进制 transport

未知 Tops 语义方法必须返回 `MethodNotFound`；不得调用 Clang 或 clangd 生成隐藏结果。

## 依赖方向

```text
cmd/tops-lsp
    -> server -> protocol
                -> document
    -> transport
    -> logging

integration tests -> cmd/tops-lsp
unit tests       -> owning internal package
```

`transport` 不依赖 `document` 或 `server`；`document` 不依赖 Tops headers；`logging` 不读取源文件内容。若未来增加 semantic 层，只能通过明确的 server 接口接入，不能反向改变基础 transport 契约。
