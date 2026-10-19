# 契约：六个语言请求的行为

这份文档固定补全、悬停、跳定义、查引用、列文档符号、诊断的行为。客户端、服务器和测试都以这份文档为准。

词汇约定：

- **空值结果**：`result` 为 `null`。
- **空名单**：`result` 为 `[]`。
- **新鲜结果**：文档版本、上下文版本、分析代号、来源指纹都与当前一致的分析结果。
- **分析进行中**：文档已打开、还没有新鲜结果、并且分析尚未结束。

## 1. 能力声明

`initialize` 返回的 `capabilities` 只包含这些字段：

| 字段 | 值 |
|------|-----|
| `textDocumentSync.openClose` | `true` |
| `textDocumentSync.change` | `2`（增量） |
| `definitionProvider` | `true` |
| `referencesProvider` | `true` |
| `documentSymbolProvider` | `true` |
| `completionProvider` | `{}`（不声明触发字符、不声明 `resolveProvider`） |
| `hoverProvider` | `true` |

**不得出现**：`diagnosticProvider`、`workspaceSymbolProvider`、`completionProvider.triggerCharacters`、`completionProvider.resolveProvider`、`completionProvider.allCommitCharacters`。

## 2. 初始化时读取的客户端能力

| 字段 | 用途 | 缺失时 |
|------|------|--------|
| `capabilities.textDocument.hover.contentFormat` | 悬停内容格式 | 按 Markdown 处理 |

判定规则：列表中出现 `markdown` → 用 Markdown；否则 → 用 `plaintext`。

## 3. 六种请求的共同规则

### 3.1 参数校验

| 情况 | 结果 |
|------|------|
| 请求体不是合法 JSON 对象 | `InvalidParams` |
| 缺 `textDocument.uri` 或为空 | `InvalidParams` |
| 位置类请求缺 `position` 或行列非法 | `InvalidParams` |
| 位置超出当前文档范围 | `InvalidParams` |

参数错误时不返回部分结果。

### 3.2 文档状态

**判定顺序**（从上到下，第一条命中即停止，不得跳着用后面的规则）：

| 顺序 | 情况 | 结果 |
|------|------|------|
| 1 | 文档未打开 | 按第 4 节各方法的“空结果”列 |
| 2 | 有新鲜结果 | 按新鲜结果处理；光标处无对应内容时按“空结果”列 |
| 3 | 无新鲜结果，但存在已失效的旧结果 | 按“空结果”列，不返回旧数据，不等待 |
| 4 | 无新鲜结果也无旧结果，且分析进行中 | 按第 4 节“分析进行中”列 |
| 5 | 无新鲜结果也无旧结果，且分析已结束（失败或无产出） | 按“空结果”列 |

### 3.3 取消

请求处理过程中收到取消：

- 结果尚未生成 → 返回 `RequestCancelled`，不发送结果。
- 结果已经生成且校验通过 → 允许返回成功结果。

同一个请求只响应一次。

### 3.4 过期

计算过程中文档版本、上下文版本或来源指纹变化，且没有新鲜结果可用：

- 按“空结果”列返回。
- 记录一条过期丢弃日志（分类 + 文档标识 + 版本），不记源码。

### 3.5 未实现的方法

`textDocument/diagnostic`、`workspace/diagnostic`、`workspace/symbol`、`completionItem/resolve` 及其他未注册方法 → `MethodNotFound`。

服务器尚未初始化或正在关闭时，沿用现有生命周期错误行为。

## 4. 各方法

### 4.1 `textDocument/definition`

| 项 | 内容 |
|----|------|
| 请求 | `textDocument.uri`、`position` |
| 成功 | 一个或多个 `Location` |
| 空结果 | `null` |
| 分析进行中 | 最多等 2 秒；超时 → `ContentModified` |
| 参数错误 | `InvalidParams` |
| 取消 | `RequestCancelled` |

### 4.2 `textDocument/references`

| 项 | 内容 |
|----|------|
| 请求 | `textDocument.uri`、`position`、`context.includeDeclaration` |
| 成功 | `Location` 列表（可能为空） |
| 空结果 | `[]` |
| 分析进行中 | 最多等 2 秒；超时 → `ContentModified` |
| 参数错误 | `InvalidParams` |
| 取消 | `RequestCancelled` |

### 4.3 `textDocument/documentSymbol`

| 项 | 内容 |
|----|------|
| 请求 | `textDocument.uri` |
| 成功 | `DocumentSymbol` 列表（可能为空） |
| 空结果 | `[]` |
| 分析进行中 | 最多等 2 秒；超时 → `ContentModified` |
| 参数错误 | `InvalidParams` |
| 取消 | `RequestCancelled` |

### 4.4 `textDocument/completion`

| 项 | 内容 |
|----|------|
| 请求 | `textDocument.uri`、`position` |
| 成功 | `CompletionItem[]`（本版不使用 `CompletionList`） |
| 正常 | 当前光标位置语法允许、语义可见的候选；每个候选有 `label`、`kind`，已知时带 `detail`、`textEdit`、`filterText`、`sortText` |
| 空结果 | `[]` |
| 分析进行中 | 返回只含一条提示项的列表（见 4.4.1），不等待 |
| 注释、字符串、预处理指令内的位置 | `[]` |
| 参数错误 | `InvalidParams` |
| 取消 | `RequestCancelled` |

#### 4.4.1 “正在分析”提示项

| 字段 | 值 |
|------|-----|
| `label` | `Analyzing…` |
| `kind` | `1`（Text） |
| `detail` | `Analysis in progress; completion is not available yet` |
| `insertText` | 不填 |
| `textEdit.range` | 光标处的空范围 |
| `textEdit.newText` | 空字符串 |
| `filterText` | 当前正在输入的前缀（可能为空） |
| `sortText` | `000`（按 [completion-candidates.md](completion-candidates.md) 第 6 节的三位数字前缀规则，排在真实候选之前） |

该场景下不得同时返回真实候选。

### 4.5 `textDocument/hover`

| 项 | 内容 |
|----|------|
| 请求 | `textDocument.uri`、`position` |
| 成功 | `{ range, contents: { kind, value } }` |
| 正常 | 名字 + 种类；已知时类型或签名；声明位置；profile 与 pass；上下文状态 |
| 空结果 | `null` |
| 分析进行中 | 返回提示结果（见 4.5.1），不等待 |
| 注释、字符串、空白、解析不出名字 | `null` |
| 参数错误 | `InvalidParams` |
| 取消 | `RequestCancelled` |

#### 4.5.1 “正在分析”提示结果

| 字段 | 值 |
|------|-----|
| `range` | 光标处单字符范围；光标在文件末尾时取前一个字符 |
| `contents.kind` | 按第 2 节协商 |
| `contents.value` | `Analyzing… analysis is still in progress` |

**`range` 的边界规则**（按顺序判断，前一条命中就不再往后看）：

| 情况 | `range` |
|------|---------|
| 光标偏移大于 0 且不在文件末尾 | 选中光标处一个字符：`[offset, offset+1)` |
| 光标在文件末尾，且文件非空 | 选中前一个字符：`[len-1, len)` |
| 文件为空（`len == 0`） | `0:0 - 0:0`（空范围） |
| 光标在偏移 0 且文件非空 | 选中第一个字符：`[0, 1)` |

空范围是合法的 LSP 位置，客户端会把它渲染成光标处的提示框；不得因文件为空而返回 `null`，否则违反 FR-023 的“分析进行中必须给提示”。

### 4.6 诊断（`textDocument/publishDiagnostics`）

诊断是服务器主动推送的通知，不是请求。

| 项 | 内容 |
|----|------|
| 时机 | 每次分析提交完结果后；文档关闭或上下文失效时清空 |
| 参数 | `uri`、`version`、`contextVersion`、`diagnostics` |
| 无问题时 | 仍然发送一次，`diagnostics` 为空数组 |
| 旧结果 | 不发送 |

详细规则见 [diagnostics-pipeline.md](diagnostics-pipeline.md)。

## 5. 一致性要求

- 一次请求只能基于一份新鲜结果；不得把不同文档版本、不同上下文版本、不同 pass 的结果拼接。
- 补全和悬停绝不等待分析；导航类最多等 2 秒。
- 所有结果都保留来源记录：profile、pass、来源类别。
- 六个方法的空结果、错误、取消、过期规则与本节一致；导航与分析进行中的唯一差异是“等待 + 超时错误”。
