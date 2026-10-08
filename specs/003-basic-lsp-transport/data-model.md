# 数据模型：P0 基础服务和 LSP 传输

本模型描述基础 server 在内存中的状态和消息关系。它不包含 AST、符号、Tops target 或语义结果。

## 1. ServerSession

表示一个 Go server 进程从启动到退出的状态。

| 字段 | 类型 | 规则 |
| --- | --- | --- |
| `session_id` | opaque string | 进程启动时生成；日志中使用，不暴露源代码 |
| `state` | `starting` / `initialized` / `shutdown_pending` / `exited` | 只能按生命周期表迁移 |
| `client_info` | optional summary | 只保存 initialize 中的必要摘要，不记录完整 payload |
| `workspace_summary` | optional summary | 只保存 workspace 数量或脱敏摘要 |
| `documents` | map by normalized URI | 只包含活动文档 |
| `requests` | map by JSON-RPC ID | 只包含未完成请求 |
| `exit_reason` | optional enum | 记录正常 exit、EOF、protocol failure、internal failure 等原因 |

### 状态迁移

```text
starting --initialize success--> initialized
starting --exit---------------> exited (abnormal)
initialized --shutdown success-> shutdown_pending
initialized --exit--------------> exited (abnormal)
shutdown_pending --exit---------> exited (normal)
any active state --stdin EOF---> exited (client disconnected)
```

`exited` 是终态。`shutdown_pending` 不接受新的文档状态变化；已登记请求可以被取消或完成，但不能产生违反生命周期的额外公开能力。

## 2. TransportFrame

表示 stdin/stdout 上的一条完整 LSP 帧。

| 字段 | 类型 | 规则 |
| --- | --- | --- |
| `headers` | map string -> string | header 名称大小写不敏感；`Content-Length` 必须唯一有效 |
| `content_length` | non-negative integer | body 的 UTF-8 字节长度，不是字符数 |
| `body` | byte slice | 只有读取到声明长度后才交给 JSON 解码 |
| `direction` | stdin/stdout | stdout 只能写 response 或允许的 server notification |

不完整 header/body 不能执行。可选 `Content-Type` 可以被忽略；格式错误、冲突或无法确定长度的 header 必须产生传输错误。

## 3. LSPMessage

JSON-RPC 2.0 消息分为三类：

- **Request**：包含 `jsonrpc`、非空 request ID、`method` 和可选 `params`，必须产生一次 response。
- **Notification**：包含 `jsonrpc`、`method` 和可选 `params`，不得产生 response。
- **Response**：包含 `jsonrpc`、原 request ID，以及 `result` 或 `error` 二选一。

基础方法：`initialize`、`shutdown`、`exit`、`textDocument/didOpen`、`textDocument/didChange`、`textDocument/didClose` 和 `$/cancelRequest`。其他 request 返回 `MethodNotFound`，其他 notification 记录并忽略。

## 4. DocumentState

表示一个打开文档的最近一次有效输入。

| 字段 | 类型 | 规则 |
| --- | --- | --- |
| `uri` | URI string | 规范化后作为 map key，日志使用脱敏 ID |
| `language_id` | string | 保存客户端提供值，不据此推断 Tops 语义 |
| `version` | integer | `didOpen` 建立基线；后续 `didChange` 必须严格递增 |
| `text` | UTF-8 string | 只保存在内存；日志不得完整输出 |
| `opened` | boolean | 活动文档为 true；close 后删除状态 |
| `last_change_count` | integer | 仅用于日志和测试摘要 |

### 文档不变量

1. `didChange` 只能应用于已打开 URI。
2. 一条通知的 `contentChanges` 按数组顺序应用；中途失败时不提交部分结果。
3. 位置使用 UTF-16 code unit 解释，行从 0 开始。
4. 版本重复、倒退、缺失、范围越界、`rangeLength` 不一致或 full change 均不改变最近一次有效状态。
5. `didClose` 后的 change 不会隐式重新创建文档；再次 open 才建立新基线。

## 5. TextDocumentContentChange

表示一个增量变更。

| 字段 | 类型 | 规则 |
| --- | --- | --- |
| `range.start` / `range.end` | Position | UTF-16 行/字符位置；start 不得晚于 end |
| `range_length` | optional integer | 若存在，必须为非负整数，并等于 range 覆盖文本的 UTF-16 code unit 数；不一致时拒绝整个通知 |
| `text` | string | 要插入或替换的 UTF-8 文本 |

本功能的 capabilities 只声明 incremental sync，因此没有 `range` 的 full-text change 是 `InvalidParams` 类通知错误，不能静默当作增量变更。

## 6. RequestState

保证取消和响应只有一个终态。

| 字段 | 类型 | 规则 |
| --- | --- | --- |
| `request_id` | string or number | 与 JSON-RPC request ID 完整相等比较 |
| `method` | string | 只记录方法名，不记录完整 params |
| `state` | `running` / `cancelled` / `responded` / `failed` | 使用单向终态转换 |
| `cancel_signal` | cancellation signal | `$/cancelRequest` 命中时触发 |
| `started_at` | timestamp | 日志耗时起点 |
| `finished_at` | optional timestamp | 终态时写入 |

允许的终态迁移：`running -> cancelled`、`running -> responded`、`running -> failed`。已进入终态的请求不再发送任何额外响应。

## 7. LSPError

| 字段 | 类型 | 规则 |
| --- | --- | --- |
| `code` | integer | 使用规格规定的 JSON-RPC/LSP code |
| `name` | string | `ParseError`、`InvalidRequest` 等稳定名称 |
| `message` | string | 面向客户端的短摘要，不含堆栈和源代码 |
| `data` | optional object | 只包含脱敏的阶段、字段或关联 ID |
| `request_id` | optional ID | 可确定时保留原 ID；无法确定时按 JSON-RPC 规则处理 |

## 8. LogRecord

| 字段 | 必填 | 规则 |
| --- | --- | --- |
| `timestamp` | 是 | 结构化时间 |
| `level` | 是 | debug/info/warn/error |
| `event` | 是 | 固定事件名，如 `server_started`、`document_change_rejected` |
| `session_id` | 是 | 关联同一进程会话 |
| `request_id` | 条件 | request/cancel/error 事件需要 |
| `method` | 条件 | 消息相关事件需要 |
| `document_id` | 条件 | 只使用脱敏 URI 标识 |
| `document_version` | 条件 | 文档事件需要 |
| `duration_ms` | 条件 | 完成或失败的 request 需要 |
| `error_code` / `result` | 条件 | 失败或结果事件需要 |

不允许字段：完整源代码、完整 JSON payload、密码、token、密钥和无关用户数据。

## 9. 关系和所有权

- 一个 `ServerSession` 拥有多个 `DocumentState` 和 `RequestState`。
- 一个 `TransportFrame` 解码为一个 `LSPMessage`；一个 request 最多生成一个 response frame。
- `RequestState` 的取消只影响自身，不修改 `DocumentState` 或 `ServerSession.state`。
- `DocumentState` 只由 `document` 层更新；`server` 负责决定何时调用 open/change/close。
- `LogRecord` 可引用会话、请求和文档摘要，但不拥有这些对象，也不能改变协议状态。
