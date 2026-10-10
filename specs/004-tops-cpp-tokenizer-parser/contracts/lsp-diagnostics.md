# 契约：Parser Diagnostics 与 LSP 边界

## 参与方

| 参与方 | 所有权 | 不负责 |
| --- | --- | --- |
| Go `parser` | token、syntax tree、macro state、recovery、parser diagnostic | JSON-RPC framing、客户端展示、Clang runtime |
| Go `server` | 文档 snapshot、ParseContext、版本保护、diagnostic 编码和发布 | TypeScript 语义判断、Clang fallback |
| Go `protocol` | LSP `Diagnostic` 和 `publishDiagnostics` 数据模型 | grammar、macro evaluation |
| Go `transport` | `Content-Length` frame 和并发安全写入 | diagnostic 内容 |
| TypeScript client | server 生命周期、接收和展示标准 LSP notification | 关键字扫描、目标判断、诊断重算 |
| Clang 对照验证 | 离线对照记录 | server runtime 结果 |

## 消息

server 在文档成功打开或成功变更后，可以发送标准 LSP notification：

```json
{
  "jsonrpc": "2.0",
  "method": "textDocument/publishDiagnostics",
  "params": {
    "uri": "file:///workspace/example.tops",
    "version": 2,
    "diagnostics": [
      {
        "range": {
          "start": { "line": 3, "character": 10 },
          "end": { "line": 3, "character": 10 }
        },
        "severity": 1,
        "code": "tops-syntax-missing-token",
        "source": "tops-lsp",
        "message": "expected ';'"
      }
    ]
  }
}
```

`Diagnostic` 字段：

- `range` 必填，使用 LSP 0-based UTF-16 half-open range；
- `severity` 映射 Error=1、Warning=2、Information=3、Hint=4；
- `code` 使用稳定字符串；
- `source` 固定为 `tops-lsp`；
- `message` 是 Go server 自有的简短说明，不转发 Clang 原文；
- `relatedInformation` 只携带 opening delimiter、宏定义或 context 来源等必要位置；
- `version` 与已打开文档的版本一致。

## 发布时机

1. `didOpen` 参数合法且文档写入成功后，使用该版本文本解析并发布一次。
2. `didChange` 所有 change 成功应用后，使用新版本文本解析并发布一次。
3. change 被拒绝时，保留旧文本/版本和旧分析结果；不得发布针对未应用文本的 diagnostics。
4. `didClose` 后停止发布该 URI 的旧结果；旧 parse job 即使结束也必须丢弃。
5. parser 没有 diagnostics 时仍发布空数组，以清除当前版本的旧诊断。
6. publisher 写失败时记录 `diagnostics_publish_failed`，不记录源文本；按现有 transport fatal 规则结束或进入明确失败状态。

## Server 接入边界

- `Server.Run` 已持有 `transport.Writer`，应把它包装为 server 内部 publisher，并与 response 共用 writer 的互斥写保护。
- `Server.Handle` 保持现有返回值和同步测试语义；没有 publisher 时，文档通知仍更新 store，但不向 nil sink 写 frame。
- parser 不直接依赖 transport；server 将 `ParserDiagnostic` 转为 `protocol.Diagnostic` 后交给 publisher。
- 诊断发布必须发生在文档成功更新之后；不能让 parser 失败回滚已成功应用的文档文本。
- server 记录脱敏文档 ID、版本、context version、结果状态、diagnostic count 和 code 摘要，不记录全文或诊断中可能重复的源代码片段。

## 版本与竞态

发布前检查：

- 文档仍然 open；
- `DocumentState.Version` 等于 ParseContext 的 `document_version`；
- 当前 context version 等于 ParseResult 的 `context_version`；
- 当前 generation 没有被取消或替代。

同步第一版也执行检查。若未来 parser 改为异步，旧结果只能被丢弃，不能以空诊断覆盖新版本。

## 初始化与兼容性

- `initialize` 可继续只声明已有 `textDocumentSync` 能力；推送 diagnostics 使用标准 notification，不新增客户端专用扩展能力。
- 现有未知语义 request 仍返回 `MethodNotFound`；本功能不实现 completion、hover、definition 或 references。
- stdout 仍只能包含 LSP frame；日志继续写 stderr/受控日志目标。
- 新增 protocol model 对旧客户端是向后兼容的；不理解 diagnostics 的客户端不会改变 server 的 parser 归属。

## 测试契约

- protocol 单测验证 JSON 字段、severity、code、version、related information 和空 diagnostics。
- server 单测验证无 publisher 的 `Handle` 兼容行为、版本保护和失败变更不发布。
- integration 测试在 `didOpen`/`didChange` 后读取并检查 diagnostics frame，再继续读取 shutdown response。
- 测试必须确认 stdout 没有 JSON-RPC response 之外的日志或 parser 文本。
