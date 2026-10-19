# 契约：Semantic Diagnostics LSP 输出

## 输出方法

v1 只使用标准 `textDocument/publishDiagnostics`。不新增 hover、completion 或私有 semantic LSP 请求。

## 稳定 code 与 severity

| Code | Severity | 条件 |
| --- | --- | --- |
| `tops-semantic-unknown-name` | Error | resolved SemanticContext 下没有可见声明 |
| `tops-semantic-ambiguous-name` | Warning | 候选不能安全消歧 |
| `tops-semantic-type-mismatch` | Error | 基础类型转换确定不兼容 |
| `tops-semantic-invalid-call` | Error | 唯一 callable 候选确定不可调用；`conversion_status=unsupported` 不适用此 Error |
| `tops-semantic-host-device-boundary` | Error | 当前 pass 的调用边界确定非法 |
| `tops-semantic-address-space` | Error | 当前 profile 的地址空间规则确定违反 |
| `tops-semantic-vector-constraint` | Error | 当前 profile 的 vector 约束确定违反 |
| `tops-semantic-context-missing` | Information | 无法形成可用上下文 |
| `tops-semantic-context-degraded` | Information | 上下文部分可用，规则被抑制 |
| `tops-semantic-incomplete-input` | Hint | parser 标记可恢复的不完整输入 |

`gcu400-default` 生成 `resolved` SemanticContext 时可以产生 GCU400 目标 Error；`invalid/partial/missing` 不产生未经证明的目标 Error。

## Diagnostic.data

保持现有字段，并追加/使用：

```json
{
  "documentVersion": 7,
  "contextVersion": 3,
  "sourceVersion": "sha256:...",
  "contextStatus": "resolved",
  "profileId": "gcu400-default",
  "passKind": "host",
  "semanticKind": "host-device-boundary"
}
```

- `passKind` 对 pass-sensitive diagnostic 必须是 `host` 或 `device`；不依赖具体 pass 的 `tops-semantic-context-missing`、`tops-semantic-context-degraded` 和 `tops-semantic-incomplete-input` 使用 `not-applicable`；组合命令不得直接作为诊断值；
- `profileId` 必须非空；
- `contextStatus` 必须与 SemanticContext 一致；
- `sourceVersion` 是按 URI 排序后的 source/header version aggregate fingerprint，不逐个暴露 header 版本；semantic diagnostic 必填，旧 parser/index diagnostic 可以省略；
- `semanticKind` 只能是 `unknown-name`、`ambiguous-name`、`type-mismatch`、`invalid-call`、`host-device-boundary`、`address-space`、`vector-constraint`、`context-missing`、`context-degraded` 或 `incomplete-input`；
- 不记录完整源文本、凭据或原始 Clang message。

## Internal-to-wire field mapping

| Semantic model field | JSON field | Required rule |
| --- | --- | --- |
| `document_version` | `documentVersion` | 当前诊断所属文档版本；当前文档诊断必填 |
| `context_version` / `semantic_context_version` | `contextVersion` | 当前 SemanticContext 版本 |
| `source_version_fingerprint` | `sourceVersion` | semantic diagnostic 必填；旧 parser/index diagnostic 可以省略 |
| `context_status` | `contextStatus` | `resolved`、`partial`、`missing` 或 `invalid` |
| `profile_id` | `profileId` | 始终非空 |
| `pass_kind` | `passKind` | pass-sensitive diagnostic 为 `host` 或 `device`；pass-neutral diagnostic 为 `not-applicable`；组合命令不得直接作为诊断值 |
| `semantic_kind` | `semanticKind` | 上述固定 semantic kind 之一 |
| `conditional_provenance` | `conditionalProvenance` | 有条件来源时提供 |
| `generation` / `epoch` | 不进入 JSON | 只用于 stale 判断和日志 |

`sourceVersion` 的确定算法唯一使用导出的 `internal/index.NormalizeSourceURI` 和 `internal/semantic.AggregateSourceVersion(documentURI string, entries []SourceVersionEntry) (string, error)`；`NormalizeSourceURI` 只负责 URI 规范化，aggregate helper 负责同一规范化 URI 的重复记录合并、版本冲突、排序和 SHA-256。root source 固定为当前 raw `document_uri` 的规范化 URI；同一规范化 URI 的版本相同只保留一条，版本冲突写入 aggregate-only 的 `status:conflict`；随后按 URI UTF-8 字节序排序，每项序列化为 `uri\0version:<value>\n` 或 `uri\0status:<value>\n`，使用 UTF-8 bytes 计算 SHA-256，输出 `sha256:<hex>`。semantic diagnostic 的 root source 必须存在，否则不发布该 diagnostic；comparison record 使用同一规则，root source 缺失时标记为 `not-comparable`；依赖缺失使用 `missing`、`unresolved`、`unreadable` 或 `stale` status token。

## Range 与 related information

- 主 range 是最小可操作的原文范围，使用 0-based UTF-16 半开区间；
- host/device/address-space/vector Error 没有非空 range 时不得发布；
- 声明、profile 或 header 来源使用标准 `relatedInformation`；
- 重复判断包含 code、range、candidate、condition 和 passKind。

## 版本与错误

- 当前文档/context/header 版本不匹配时不发布旧诊断；只记录 `stale_result`；
- 请求取消使用现有标准取消错误；
- 参数错误使用标准 JSON-RPC/LSP error；
- 诊断 message 由 Go server 生成，Clang message 仅进入离线对照记录。
