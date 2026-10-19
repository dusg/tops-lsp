# 数据模型

这份文档说明本功能涉及的数据结构：哪些已经存在、哪些需要新增、字段含义和约束。字段名与代码里实际使用的一致，方便实现时直接对照。

## 1. 一次请求

客户端发来的六种请求之一。

| 字段 | 含义 | 来源 |
|------|------|------|
| 方法名 | `textDocument/definition`、`references`、`documentSymbol`、`completion`、`hover` | 请求消息 |
| 文档地址 | `textDocument.uri` | 请求参数 |
| 光标位置 | `position`（行、列，UTF-16 列） | 请求参数，定位类请求才有 |
| 引用上下文 | `context.includeDeclaration` | 仅 `references` |

**约束**：

- 文档地址为空、位置格式不对、位置超出文件范围 → `InvalidParams`。
- 位置换算统一走 `internal/position` 的 `Mapper`，不自己算 UTF-16。

## 2. 一次分析的结果

| 字段 | 含义 |
|------|------|
| 根文档地址 | 这次分析针对的文档 |
| 文档版本 | 编辑器的文档版本号 |
| 上下文版本 | 目标上下文（profile、pass、宏、include 根）的版本号 |
| 分析代号 | 同一文档的递增编号，用于判断新旧 |
| 纪元 | 上下文整体失效时递增，用于一次性作废所有结果 |
| 来源指纹 | 文档与依赖内容摘要，用于比较内容是否变化 |
| 状态 | `complete` / `partial` / `degraded` / `stale` |

对应类型：`index.IndexSnapshot`（符号、引用、作用域、诊断、源码）与 `semantic.SemanticSnapshot`（声明、作用域、类型、可调用、调用、诊断）。

**约束**：

- 六个请求的结果都必须来自同一份“当前”结果：文档版本、上下文版本、代号、来源指纹全部匹配。
- 旧结果只能被丢弃，不能参与返回。

## 3. 诊断

| 字段 | 含义 | 约束 |
|------|------|------|
| `range` | 代码起止位置 | 必须能换算成合法的行、列 |
| `severity` | 1 Error / 2 Warning / 3 Information / 4 Hint | 见下面的级别规则 |
| `code` | 诊断编号 | 稳定、不含中文 |
| `source` | 固定为 `tops-lsp` | 固定 |
| `message` | 英文说明 | 不得中英混排 |
| `relatedInformation` | 关联位置与说明 | 可选 |
| `data.documentVersion` | 文档版本 | 语法诊断与语义诊断必填 |
| `data.contextVersion` | 上下文版本 | 必填 |
| `data.contextStatus` | `resolved` / `partial` / `missing` / `invalid` | 语义诊断必填 |
| `data.profileId` | 目标配置 | 语义诊断必填 |
| `data.passKind` | `host` / `device` / `not-applicable` | 语义诊断必填 |
| `data.semanticKind` | 编号去掉 `tops-semantic-` 前缀 | 语义诊断必填 |
| `data.reason` | 降级或失败原因 | 索引诊断使用 |

**级别规则**：

| 类别 | 级别 |
|------|------|
| 语法错误 | Error |
| 已核实且确定不成立的标准 C++ 语义错误 | Error |
| 已核实的 Tops 扩展违规 | Error |
| 名称候选歧义（`tops-semantic-ambiguous-name`） | Warning |
| 上下文缺失或不完整（`tops-semantic-context-missing`、`tops-semantic-context-degraded`） | Warning |
| 输入不完整（`tops-semantic-incomplete-input`） | Hint |
| 索引降级（`analysis-degraded`） | Information |

**去重键**：`code | range.start | range.end | severity | message`。

**排序键**：`range.start`、`range.end`、`severity`、`code`、`message`，全部升序。

## 4. 补全项

新增协议类型，字段与 LSP 标准一致。

| 字段 | 含义 | 约束 |
|------|------|------|
| `label` | 显示的名字 | 必填 |
| `kind` | 种类（标准枚举数值） | 必填 |
| `detail` | 补充说明，例如类型或签名 | 已知时填写 |
| `insertText` | 插入的文本 | 提示项不填 |
| `textEdit` | 替换范围与新文本 | 真实候选填；提示项为空范围、空文本 |
| `filterText` | 过滤用的文本 | 提示项使用当前前缀 |
| `sortText` | 排序文本 | 保证顺序稳定 |
| `data` | 附加信息（来源、profile、pass、降级状态） | 只放能公开的信息 |

**种类映射**（索引符号 → 补全种类）。`type` 按 `SymbolRecord.TypeKind` 细分，与现有 `protocolSymbolKind` 的判断保持一致：

| 索引种类 | `TypeKind` | 补全种类 |
|----------|------------|----------|
| `function` | — | `Function` |
| `type` | `enum` | `Enum` |
| `type` | `struct`、`union` | `Struct` |
| `type` | 其他（含 `class`、别名） | `Class` |
| `namespace` | — | `Module` |
| `field` | — | `Field` |
| `variable` / `builtin` | — | `Variable` |
| `macro` | — | `Constant` |
| `attribute` | — | `Property` |
| 关键字 | — | `Keyword` |
| 提示项 | — | `Text` |

## 5. 悬停结果

| 字段 | 含义 | 约束 |
|------|------|------|
| `range` | 光标处名字的范围 | 必须落在文档内 |
| `contents.kind` | `markdown` 或 `plaintext` | 由客户端能力决定 |
| `contents.value` | 说明文字 | 只含已核实事实 |

**内容结构**（三段）：

1. 名字与种类，例如 `foo — function`。
2. 类型或签名（已知时）。
3. 来源：声明位置、profile、pass、上下文状态；降级时追加说明。

## 6. 上下文

决定 Tops 规则是否适用的信息。

| 状态 | 含义 | 对外表现 |
|------|------|----------|
| `resolved` | 信息齐全 | 可以发布 Error |
| `partial` | 只有一部分 | 不发 Error，最多 Warning 降级提示 |
| `missing` | 没有 | 同上 |
| `invalid` | 互相矛盾 | 同上 |

沿用 `semantic.SemanticContext`：`Profile.ProfileID`、`Command.PassKind`、`Status`、来源记录。

## 7. 宏

不新增结构。宏已经是 `index.SymbolRecord`，`Kind` 为 `SymbolKindMacro`，来自 `#define` 或 `PredefinedMacros`。

**补全约束**：条件状态为“无效”（`ConditionalInactive`）的宏不进入候选；条件未知时进入候选，但悬停要标明降级。

## 8. 分析完成通知（新增）

| 字段 | 含义 |
|------|------|
| 文档地址 | 通知按文档地址分组 |
| 通知通道 | 每个文档一组等待者，广播时全部唤醒 |

**触发时机**：

- 一次分析结束（成功提交结果）。
- 分析被丢弃（过期、取消、失败），避免等待方死等。
- 文档关闭、上下文整体失效。

**约束**：不保存历史通知；等待方每次被唤醒后重新检查是否有新鲜结果。

## 9. 验收测试文件

| 文件 | 用途 | 预期 |
|------|------|------|
| `testdata/lsp/fixtures/valid-kernel.tops` | 能正常分析 | 无非预期 Error；六种请求都有正常结果 |
| `testdata/lsp/fixtures/std-cpp-error.tops` | 标准 C++ 写错 | 预期语法或语义 Error；修正后清空 |
| `testdata/lsp/fixtures/tops-semantic-error.tops` | Tops 规则写错 | 预期 `tops-semantic-*` Error 与来源字段 |

**约束**：

- 三个文件都以真实文件存在，测试读取内容，不把源码内联到测试里。
- 每个文件不超过 500 行。
- 运行不需要 GCU 硬件、GCUSIM 或 Clang。
