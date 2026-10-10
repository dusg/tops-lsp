# 数据模型：Tops C++ Tokenizer 与 Parser

## 模型边界

本数据模型描述 Go parser 和 server 之间的输入/输出，不定义完整 C++ semantic、symbol index 或代码生成模型。所有 source range 以原始文档为基准；所有 LSP position 使用 0-based line 和 UTF-16 code unit。

## Parser 输入

### ParseContext

| 字段 | 类型/取值 | 必填 | 约束 |
| --- | --- | --- | --- |
| `document_uri` | string | 是 | 只用于结果归属和日志脱敏，不参与语法推断 |
| `language_id` | string | 是 | `cpp`、`c` 或 `tops` 等输入标签；不能单独决定完整 Tops 语义 |
| `language_standard` | enum | 是 | `c++11`、`c++14`、`c++17`；未知或缺失为受限状态 |
| `driver_kind` | enum | 是 | `topscc`、`clang`、`clang++` 或 `unknown`；由上游 context resolver 提供 |
| `compiler_context_status` | enum | 是 | `resolved`、`partial`、`missing`、`invalid` |
| `argument_provenance` | opaque reference | 是 | 指向已归一化参数来源；parser 不读取 raw argv |
| `predefined_macros` | map<string, MacroValue> | 否 | 只使用显式传入的值；未出现的宏为 unknown，不默认为某个 GCU target |
| `target_profile` | opaque string | 否 | 保存调用方 context；parser 不根据 profile 名称推断宏值 |
| `pass_kind` | enum/opaque string | 否 | host、device、offload host 或 offload device；只作为结果元数据 |
| `include_roots_available` | bool/enum | 否 | 表示外部 header context 是否完整；不触发 parser 自己加载 header |
| `document_version` | integer | 是 | 与 LSP 文档版本对应，必须非负 |
| `context_version` | integer | 是 | 编译上下文变化时递增，旧结果不得覆盖新结果 |

`ParseContext` 不包含完整源代码副本、凭据、raw compiler argv、response file 内容或工具进程句柄。源文本作为 parser 的独立输入参数传入；Clang 离线对照配置不进入 parser runtime context。

### MacroValue

| 状态 | 内容 | 条件求值行为 |
| --- | --- | --- |
| `undefined` | 宏明确未定义 | `defined(NAME)` 为 false；数值使用保持 unknown，除非规则能安全确定 |
| `defined` | 只表示存在但无数值 | `defined(NAME)` 为 true；直接数值比较为 unknown |
| `integer` | 有符号整数值 | 可参与受支持的 `!`、`&&`、`||`、`==`、`!=` 和括号表达式 |
| `unknown` | 来源缺失、冲突或无法安全展开 | 条件区域为 unknown，不选择活动分支 |

### SourcePosition 与 SourceRange

```text
SourcePosition:
  byte_offset: UTF-8 byte offset, inclusive
  line: zero-based source line
  utf16_character: zero-based LSP character

SourceRange:
  start: SourcePosition, inclusive
  end: SourcePosition, exclusive
```

实现可以额外保存 rune offset，但 byte offset 和 LSP UTF-16 映射必须由同一位置模块生成。CRLF 中的 `\r` 不计入行内可见 character；EOF 可以是零宽 range。

## Token

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `kind` | TokenKind | identifier、keyword、literal、operator、punctuation、comment、directive、newline、unknown、eof 等 |
| `spelling` | string | 原文 token，不保存宏展开替代文本 |
| `range` | SourceRange | 原文范围 |
| `line_start`/`line_end` | integer | 便于预处理行和错误恢复；由 range 派生 |
| `leading_trivia` | trivia list | 可选注释/空白信息；不得改变主 token range |
| `conditional_state` | active/inactive/unknown | token 所属条件区域状态 |
| `unterminated` | bool | 字符串、注释、raw literal 等是否在 EOF 截断 |

tokenizer 必须保证 token range 单调不回退；EOF token 的 start/end 相同。inactive 区域仍保留 token，以支持条件区域结构和后续 context 变化，但普通语法诊断需由 parser 按状态抑制。

## SyntaxNode

`SyntaxNode` 是带 `kind`、`range`、`children` 和可选属性的统一树节点。实现可以为高频节点使用 Go typed struct，但必须保留以下可观察字段。

| 节点类别 | 必需内容 |
| --- | --- |
| `translation-unit` | 顶层声明、预处理区域和保留 token |
| `namespace`/`using` | 名称、限定名、子声明 |
| `declaration` | declaration specifier、declarator、initializer、attributes |
| `function` | qualifier、返回类型、名称、参数、模板参数、body |
| `class`/`struct`/`enum`/`union` | 名称、基类/枚举项/成员和 body range |
| `template` | 参数、主体声明、显式参数/特化 spelling |
| `statement`/`expression` | 运算符、操作数、调用、控制结构、子节点 |
| `tops-qualifier` | Tops spelling、分类、参数 token、声明归属 |
| `tops-launch` | callee、配置表达式列表、调用参数列表、完整/不完整状态 |
| `attribute` | 标准/Tops/opaque 属性名称、参数 token、所属声明 |
| `preprocessor-directive` | directive 名称、原始参数、行范围 |
| `conditional-region` | 条件表达式、分支、状态、父级和范围 |
| `opaque` | 能识别边界但暂不解析内部语义的结构 |
| `error`/`missing-token` | 错误位置、期望 token 类别、恢复同步点和是否 EOF |

节点不得用宏展开后的虚拟 range 替换原文 range。对 unknown 条件，两个分支都保留在 `conditional-region` 下，但普通 semantic ownership 不在本 feature 内。

## ConditionalRegion

| 字段 | 类型 | 约束 |
| --- | --- | --- |
| `condition_range` | SourceRange | 指向 `#if`/`#elif` 条件表达式，不覆盖整个 body |
| `branches` | []ConditionalBranch | 按源码顺序排列，至少包含起始 branch |
| `state` | enum | active、inactive、unknown |
| `parent_id` | optional ID | 支持嵌套条件 |
| `directive_diagnostics` | []ParserDiagnostic | 只保存预处理结构错误 |

`ConditionalBranch` 保存 `directive_range`、`body_range`、条件 token 和计算结果。多分支中如果前一分支状态为 true，后续分支为 inactive；如果前置条件未知且不能排除，则保留 unknown。

## ParserDiagnostic

| 字段 | 类型 | 约束 |
| --- | --- | --- |
| `code` | string | 稳定 code，如 `tops-syntax-missing-token` |
| `severity` | enum | Error、Warning、Information、Hint |
| `message` | string | 面向用户的简短说明；不依赖 Clang 原文 |
| `range` | SourceRange | 主位置，半开范围 |
| `related` | []RelatedLocation | opening delimiter、宏定义等辅助位置 |
| `recoverable` | bool | 是否 parser 可继续产生结构 |
| `incomplete` | bool | 是否由 EOF/编辑中间状态触发 |
| `conditional_state` | active/unknown | inactive 普通 syntax error 不进入公开结果 |
| `document_version` | integer | 结果来源版本 |
| `context_version` | integer | 结果来源 context |

稳定 code 最小集合：

- `tops-syntax-unexpected-token`
- `tops-syntax-missing-token`
- `tops-syntax-unterminated`
- `tops-syntax-invalid-literal`
- `tops-syntax-invalid-directive`
- `tops-syntax-unknown-condition`
- `tops-syntax-unsupported`
- `tops-syntax-invalid-tops-attribute`

## ParseResult

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `tokens` | []Token | 包含可保留的 inactive token 和 EOF |
| `root` | *SyntaxNode | translation-unit；解析失败也必须返回 error/recovery 节点 |
| `conditional_regions` | []ConditionalRegion | 顶层索引，节点树中仍保留嵌套关系 |
| `diagnostics` | []ParserDiagnostic | 已按原文顺序和稳定去重规则排列 |
| `status` | enum | complete、recovered、partial、invalid |
| `document_version` | integer | 与输入 context 相同 |
| `context_version` | integer | 与输入 context 相同 |
| `consumed_bytes` | integer | tokenizer/parser 实际消费到的位置，调试/测试使用 |

结果状态规则：

- `complete`：输入完成且没有 parser Error。
- `recovered`：存在可恢复诊断或 incomplete 结构，但根节点和后续结构可用。
- `partial`：宏 context、include context 或 target context 不完整，语法树仍可用。
- `invalid`：输入包含无法建立可靠边界的词法/结构错误；仍必须返回部分 token、diagnostics 和 root。

## Server 分析状态

server 不把 `ParseResult` 写回 `DocumentState`，而保存可选的最新分析摘要或直接发布结果。需要异步保护时使用以下逻辑字段：

| 字段 | 说明 |
| --- | --- |
| `document_version` | 文档文本版本 |
| `context_version` | 编译上下文版本 |
| `analysis_generation` | server 内部单调序列，用于丢弃迟到任务 |
| `published_generation` | 最近一次成功写入 publisher 的序列 |

发布前必须满足：文档仍打开、文档版本相等、context version 相等、generation 未被取消。同步解析也执行同一检查，保证将来改成异步时不改变契约。

## LSP 映射

`ParserDiagnostic` 映射到标准 LSP `Diagnostic`：

- `range`：通过统一位置模块转换为 LSP UTF-16 range。
- `severity`：Error=1、Warning=2、Information=3、Hint=4。
- `code`：保留稳定字符串。
- `source`：固定为 `tops-lsp`。
- `message`：使用 Go server 自有摘要。
- `relatedInformation`：仅包含有助于定位 opening delimiter、宏定义或 context 来源的范围。
- `version`：由 `PublishDiagnosticsParams.version` 携带；没有合法文档版本时不发布文档诊断。

## 状态转换

```text
Document unopened
  -> didOpen accepted
Document open + no result
  -> parse snapshot
Document open + current result
  -> didChange accepted -> new snapshot -> parse snapshot
Document open + rejected change
  -> keep previous text/version/result
Document open
  -> didClose accepted -> stop publishing
```

诊断发布不是文档状态转换的成功条件：文档更新成功但 publisher 写失败时，server 保留文档状态并记录 fatal transport error；不会把失败结果伪装成空诊断。

## 校验规则

- `document_version` 不得小于零；server 仍沿用现有严格递增规则。
- `context_version` 变化必须使旧分析结果失效。
- Token range 不得越过源文本长度；LSP range 经过 UTF-16 边界检查。
- Missing token 的 range 必须零宽；unexpected token 的 range 不得为空，除非 token 本身为 EOF。
- inactive 区域普通诊断必须被过滤；预处理嵌套错误不受该过滤影响。
- 同一个恢复节点不得发布重复的 code + range 诊断。
