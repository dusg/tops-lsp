# LSP 导航契约

**Roadmap phase**: `P1.3`

**功能**：[spec.md](../spec.md)
**相关模型**：[data-model.md](../data-model.md)

## 所有权

- Go server 负责 AST/index 构建、候选查询、版本检查、错误处理和 LSP 编码。
- TypeScript client 只负责发送标准请求、展示结果和展示 diagnostics/log 状态，不扫描源文本，不重建符号树。
- parser 不依赖 LSP 类型；index 不直接写 transport。
- Clang/clangd 不进入 server runtime；对照验证不改变 LSP 结果。

## initialize capabilities

当实现完成且 handler 已注册，`initialize` 返回的 `ServerCapabilities` 增加：

| 字段 | 值 | 说明 |
|---|---|---|
| `definitionProvider` | `true` | 支持 `textDocument/definition` |
| `referencesProvider` | `true` | 支持 `textDocument/references` |
| `documentSymbolProvider` | `true` | 支持 `textDocument/documentSymbol` |

这些是可选能力的新增，不改变既有 `textDocumentSync.openClose=true` 和 incremental change。旧 client 可以忽略新增字段。

## `textDocument/definition`

### 请求

使用标准 `TextDocumentPositionParams`：

```json
{
  "textDocument": { "uri": "file:///workspace/main.tops" },
  "position": { "line": 12, "character": 8 }
}
```

server 将 position 转为当前文档的 source range，查找 token/AST 节点，再按 index scope、qualified name、condition 和版本查询。

### 结果

返回标准 `Location[]` 或 `null`。结果形状固定为：当前新鲜快照的合法请求始终返回 `Location[]`（无匹配时为空数组）；没有新鲜快照或快照在查询期间失效时返回标准 `null`。

- 有一个或多个 active 候选时，返回全部 active 定义位置；不按最近作用域选择一个。
- 没有定义但有声明时，返回可验证的声明位置。
- 没有 active 候选时，允许返回 unknown 候选，并在 index/diagnostic/log 中保留 `analysis-degraded`。
- inactive 候选不返回。
- 没有可验证目标时返回空数组。
- 当前没有新鲜快照或查询期间快照失效时，丢弃结果并返回 `null`，不能返回旧快照位置。

## `textDocument/references`

### 请求

使用标准 `ReferenceParams`：

```json
{
  "textDocument": { "uri": "file:///workspace/main.tops" },
  "position": { "line": 12, "character": 8 },
  "context": { "includeDeclaration": true }
}
```

### 结果

返回标准 `Location[]` 或 `null`。当前新鲜快照无匹配时使用空数组；没有新鲜快照或查询期间快照失效时使用 `null`。

- `includeDeclaration=true` 时包含目标声明/来源位置；为 `false` 时排除声明位置。
- 有一个或多个 active 候选时，返回这些候选引用的并集，并去重相同 URI/range。
- 没有 active 候选时允许使用 unknown 候选，并保留 `analysis-degraded` 状态。
- inactive 候选不参与当前引用结果。
- 没有可验证引用时返回空数组。
- 快照过期或请求取消时不得返回旧引用集合；过期返回 `null`，取消返回标准取消错误。

## `textDocument/documentSymbol`

### 请求

使用标准 `DocumentSymbolParams`：

```json
{
  "textDocument": { "uri": "file:///workspace/main.tops" }
}
```

### 结果

返回嵌套 `DocumentSymbol[]` 或 `null`。每项至少包含：

- `name`：短名称；
- `kind`：标准 `SymbolKind`；
- `range`：完整声明范围；
- `selectionRange`：名称范围；
- `detail`：可选的非语义摘要；
- `children`：嵌套 scope 中的可见符号。

可见性规则：

- 展示 `active` 和 `unknown` 分支中的符号；
- 隐藏 `inactive` 分支中的符号；
- 不完整 AST 中已经识别的符号仍可展示；
- 缺少 header 时不创建虚假 document symbol；当前文档和已加载 header 的可验证符号仍可展示。

标准 `DocumentSymbol` 没有条件状态字段。条件状态保留在内部 index，并通过必要的 `analysis-degraded` diagnostics/log 说明；不增加私有 response 字段。

标准 LSP client 兼容性由 stdio integration 中的标准 JSON decoder proxy 验证；本功能不要求启动 VS Code 或新增 TypeScript 语义实现。

## SymbolKind 映射

| Index kind | 标准 SymbolKind |
|---|---|
| namespace | Namespace |
| class/struct/union/type | Class 或 Struct |
| alias | Object |
| enum | Enum |
| function | Function 或 Method |
| field | Field |
| attribute | Property |
| builtin | Variable |
| macro | Constant |
| 无法映射的内部类别 | 保留在 index，不进入 document symbols |

映射必须集中在 protocol/navigation 编码层，index 不依赖 protocol 包。

## 空结果、错误和取消

| 情况 | 行为 |
|---|---|
| 当前符号没有目标/引用 | 合法请求返回空数组 |
| 文档 URI 未打开 | 返回现有标准 `InvalidParams` 错误，不猜测文件内容 |
| position 越界或参数结构错误 | 返回 `InvalidParams`，server 不退出 |
| 当前 index 尚未准备或已失效 | 返回 `null`；不得返回旧快照；必要状态通过 diagnostics/log 说明 |
| `$/cancelRequest` 已取消请求 | 返回现有 `RequestCancelled`，不写入或发布该请求结果 |
| handler panic 或编码失败 | 沿用现有 `InternalError`/transport 可观测行为，不生成伪造结果 |

## Degraded and stale observability

| 状态 | 触发条件 | 当前文档诊断 | 结构化日志事件和字段 | client 解释 |
|---|---|---|---|---|
| `analysis-degraded` | missing/unreadable/unresolved header、unknown condition、ambiguous candidate 或 incomplete index | 发布 code 为 `analysis-degraded` 的当前文档诊断，范围指向依赖/条件/候选位置 | `analysis_degraded`；脱敏 URI、operation、symbol kind、document/context/header version、condition、candidate count、diagnostic code、duration | 导航结果可能是部分结果；空数组不代表所有 context 都没有目标 |
| `stale-result` | 文档、header、context、generation 或 epoch 变化导致查询快照失效 | 不发布 stale 诊断 | `stale_result`；脱敏 URI、operation、old/new version、reason、duration | 请求返回 `null` 或标准取消错误，client 不展示旧结果 |

`analysis-degraded` 和 `stale-result` 不作为私有导航 response 字段；diagnostics/log 是唯一的状态观测边界。

## Range 与版本

- 所有 LSP `Position` 使用 0-based line 和 UTF-16 code unit。
- 所有 `Range` 为半开区间 `[start, end)`。
- source byte range 由 `internal/position.Mapper` 转换；不在导航 handler 中重复实现 UTF-16 算法。
- 查询使用与当前文档、context、header 相符的 index snapshot；结果没有版本字段时也不得跨版本发布。
- 版本和 stale 状态通过 server 内部检查、诊断和脱敏日志保持可观测，不增加私有导航 payload。
