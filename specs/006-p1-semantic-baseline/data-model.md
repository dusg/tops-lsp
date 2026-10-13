# 数据模型：P1.4 通用 semantic baseline

本模型描述 semantic baseline 的输入、规则、结果和版本关系，不表示当前仓库已经存在对应运行时对象。

## SemanticRule

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `id` | 稳定标识符 | 是 | 例如 `SEM-SCOPE-001`；发布后不复用 |
| `domain` | 枚举 | 是 | `declaration`、`scope`、`name`、`type`、`execution-space`、`memory-space`、`vector` |
| `input_nodes` | 节点类型列表 | 是 | 只列出 parser/index 已稳定提供的节点 |
| `context_predicate` | 文本 | 是 | 说明需要的 ContextSeed、header、macro 或 target 条件 |
| `diagnostic_code` | 稳定代码 | 是 | 必须来自 P1.4 code 表或明确的降级 code |
| `status` | 枚举 | 是 | `candidate`、`planned`、`implemented`、`verified`、`blocked` |
| `limitations` | 文本 | 是 | 记录不覆盖的语义和目标范围 |
| `verification` | 文本 | 是 | 正向、负向、不完整和对照检查 |

## SemanticDiagnostic

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `code` | 稳定代码 | 是 | 不依赖 message 文本判断 |
| `severity` | 枚举 | 是 | `error`、`warning`、`information`、`hint` |
| `range` | LSP Range | 是 | 经过 `internal/position` 转换，使用 UTF-16 半开区间 |
| `message` | 文本 | 是 | 说明已确认事实、限制和下一步动作 |
| `rule_id` | 引用 | 是 | 指向 SemanticRule |
| `source` | 文本 | 是 | 例如 `tops-semantic` 或 `compilation-context` |
| `related_information` | 位置列表 | 否 | 可指向 declaration、header 或 ContextSeed 来源 |
| `document_version` | 整数 | 是 | 诊断所属文档版本 |
| `context_version` | 整数 | 是 | 诊断所属上下文版本 |
| `degraded` | 布尔值 | 是 | partial/missing/unknown context 时为 true |

## SemanticSnapshot

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `document_uri` | URI | 是 | 文档身份 |
| `document_version` | 整数 | 是 | 与 document store 快照一致 |
| `context_version` | 整数 | 是 | 与 ContextSeed/CompilationContext 一致 |
| `index_snapshot_version` | 文本 | 是 | 与 P1.3 index 快照一致 |
| `rules` | SemanticRule 引用列表 | 是 | 本次实际运行的规则 |
| `diagnostics` | SemanticDiagnostic 列表 | 是 | 按稳定顺序输出 |
| `status` | 枚举 | 是 | `complete`、`partial`、`invalid`、`stale` |
| `discard_reason` | 文本 | 否 | `stale` 或 `invalid` 时说明原因 |

## 关系

- `ContextSeed` 是 P1.4 的最小上下文输入；resolved `CompilationContext` 可以补充 driver/profile provenance。
- 一个 `SemanticSnapshot` 只消费同一版本的 parser/index 快照。
- `analysis-degraded` 是用户可见 diagnostic code；`stale-result` 只作为内部事件和丢弃原因，不发布为过期诊断。
- semantic package 不拥有 LSP transport；server 负责取消、版本检查和发布。
