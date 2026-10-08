# 契约：Go server 基础 LSP 传输

## 契约范围

本契约固定 Go server 与 LSP client 之间的基础消息、生命周期、文档同步、取消、错误和日志行为。它不定义 Tops C++ tokenizer、parser、AST、symbol index 或 semantic 规则，也不新增非标准扩展消息。

## Owner

| 责任方 | 拥有内容 | 不拥有内容 |
| --- | --- | --- |
| Go server | stdio framing、JSON-RPC 分发、生命周期、文档内存状态、请求取消、错误响应、结构化日志 | VS Code UI、Tops 语义、clangd |
| VS Code client | 启动/停止/重启进程、发送 LSP 消息、展示结果和日志 | 语义合法性、诊断等级、文档变更算法 |
| LSP contract | 方法、负载、错误、取消、兼容性和测试行为 | 具体 Go package、第三方库、Tops API |

## 传输

- 使用 stdin/stdout 的 JSON-RPC 2.0 字节流。
- 每条消息由 header、空行和 UTF-8 JSON body 组成。
- `Content-Length` 必须是 body 的 UTF-8 字节长度；header 名称大小写不敏感且该 header 只能有一个有效值。
- `Content-Type` 等兼容 header 可以被忽略；格式错误、冲突或影响长度解析的 header 必须拒绝。
- 默认 `Content-Length` 上限为 16 MiB；超过上限的 frame 产生 transport error，server 关闭输入并终止会话。
- server 支持分片读取和一次读取多个 frame；只有 body 完整后才进入 JSON 解码。
- stdout 只允许协议帧；日志、panic 和调试信息写 stderr 或受控日志目标。
- stdin EOF 结束会话并释放状态，不等待不存在的 `exit`。
- context cancel、frame 读取失败或 response 写失败会关闭可关闭的输入，并取消、等待所有活动 request。

## 生命周期

| 当前状态 | 消息 | 成功行为 | 失败行为 |
| --- | --- | --- | --- |
| `starting` | `initialize` request | 返回 server info 和最小 `textDocumentSync` capabilities，进入 `initialized` | `InvalidParams` 或 `InvalidRequest` |
| `starting` | 其他 request | 无 | `ServerNotInitialized` |
| `initialized` | `shutdown` request | 返回 `null`，进入 `shutdown_pending` | 重复/非法顺序返回状态错误 |
| `initialized` | 文档通知、`$/cancelRequest` | 按对应契约处理 | 通知记录并忽略错误 |
| `shutdown_pending` | `exit` notification | 进程正常退出 | 新 request 不再启动处理 |
| 任意活动状态 | `exit` notification | 进程退出 | 未完成 shutdown 时使用异常退出状态 |

`initialize` 返回的 capabilities 只声明 `openClose=true` 和 incremental change，不声明 diagnostics、completion、hover、definition、references 或 document symbols。

## 文档同步

### `textDocument/didOpen`

客户端发送 URI、`languageId`、整数 `version` 和完整 `text`。server 建立活动 `DocumentState`。本操作不触发 Tops 语义分析。

### `textDocument/didChange`

客户端发送 URI、严格大于当前版本的整数 `version` 和按顺序排列的 `contentChanges`。每个 change 必须有 UTF-16 range 和替换文本；若提供 `rangeLength`，它必须等于被替换范围的 UTF-16 code unit 数。server 要么原子应用全部变更，要么保留最近一次有效文本；通知错误不发送 response。CRLF 行尾的 `\r` 属于行终止符，不计入该行的 LSP character 范围。

### `textDocument/didClose`

server 删除活动 URI。未知 close 只记录警告；close 后的 change 不会重新创建文档。

## 取消

- 客户端用 `$/cancelRequest` notification 发送 request ID。
- server 对活动 request 维护取消信号；未知、重复或已完成 ID 无副作用。
- 被取消的 request 不得发送正常结果或迟到结果；最多产生一次 `RequestCancelled` 终态。
- 取消不改变文档和 session 状态。

## 错误

| 名称 | code | 请求行为 | 通知行为 |
| --- | ---: | --- | --- |
| `ParseError` | `-32700` | 可恢复时返回 `id: null` | 记录并按传输可恢复性处理 |
| `InvalidRequest` | `-32600` | 返回错误并保留 request ID | 记录，不发送 response |
| `MethodNotFound` | `-32601` | 未知 request 返回错误 | 未知 notification 记录并忽略 |
| `InvalidParams` | `-32602` | 返回错误 | 保留旧状态并记录 |
| `InternalError` | `-32603` | 返回无堆栈摘要和关联 ID | 记录错误 |
| `ServerNotInitialized` | `-32002` | 初始化前的受限 request 返回错误 | 记录并忽略 |
| `RequestCancelled` | `-32800` | 取消 request 的唯一终态之一 | 不适用 |

错误 message 不包含完整源代码或堆栈；详细诊断只进入受控日志。

## 日志

必须记录 `server_started`、`initialize_succeeded/failed`、`frame_rejected`、`document_opened`、`document_changed`、`document_change_rejected`、`document_closed`、`request_cancelled`、`shutdown`、`exit`、`stdin_eof` 和 `internal_error` 等事件。

每条日志至少有 `timestamp`、`level`、`event`、`session_id`；请求事件增加 `request_id`、`method`、`duration_ms` 和 `result/error_code`；文档事件增加脱敏 `document_id` 和版本。默认不记录完整 payload、源文本、凭据或密钥。

## 兼容性和测试

- 只使用标准 LSP 方法；本阶段没有版本化扩展消息。
- server 新增未知能力时不得静默返回普通 C++ 或 Tops 语义结果。
- 任何方法、错误 code、文档版本规则或日志字段变更都必须更新本契约和对应 contract tests。
- 实现验收必须包含 framing、生命周期、文档同步、取消、错误、日志和范围边界测试。
