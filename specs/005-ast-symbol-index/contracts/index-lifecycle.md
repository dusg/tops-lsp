# 索引生命周期与内部边界契约

**Roadmap phase**: `P1.3`

**功能**：[spec.md](../spec.md)
**相关模型**：[data-model.md](../data-model.md)

## 组件边界

```text
DocumentState + ParseContext
          |
          v
      parser.Parse
          |
          v
 ParseSnapshot + ConditionalRegions
          |
          v
   index.Builder + SourceProvider
          |
          v
   IndexSnapshot / HeaderDependency
          |
          v
 index.Store (version-checked commit)
          |
          v
 server navigation handlers -> standard LSP results
```

| 组件 | 输入 | 输出 | 不负责 |
|---|---|---|---|
| `internal/parser` | 文本、`ParseContext` | tokens、AST、条件区域、parser diagnostics | symbol ID、header 读取、LSP |
| `internal/index` builder | parse snapshot、文本、source provider | scope、symbols、references、dependencies、index status | transport、target legality、完整 overload resolution |
| `internal/index` store | versioned index snapshot | 当前快照、依赖失效、查询输入 | JSON 编码、客户端展示 |
| `internal/server` | document lifecycle、analysis queue、index store | commit、导航查询、diagnostics/log、LSP response | client-side semantic scan |
| `internal/protocol` | index/query DTO | 标准 LSP JSON | parser/index 规则 |

## 输入契约

Builder 输入必须包含：

- 当前 `document.DocumentState` 值快照；
- `parser.ParseResult` 及对应的 `ParseContext`；
- document/context/generation/epoch 版本；
- `SourceProvider`，用于已打开文档和按需 header；
- 不可变的 include roots 和宏/target provenance。

Builder 不读取 raw compiler argv、response file，不启动 `topscc`、Clang 或 clangd，不访问设备，不发送网络请求。

## SourceProvider

这是一个内部接口，具体命名可在实现任务中确定，必须表达以下行为：

1. 根据 URI 获取已打开文档的文本和版本。
2. 提供 `IncludeRoots() (roots []string, available bool)`，作为 server 构造 `ParseContext.IncludeRoots` 和 `IncludeRootsAvailable` 的唯一来源。
3. 根据 requester URI、include spelling 和 include roots 解析被引用 header。
4. 返回 header 文本、URI、source version 或明确的 `unresolved`/`missing`/`unreadable` 状态。
5. 记录 include range 和搜索来源，供 `HeaderDependency` 和 diagnostics 使用。
6. 不进行 workspace 全量扫描；只有被当前文档或已打开 workspace 文件引用的 header 才能被加载。

测试使用内存 provider；生产边界可以由 server 适配 document store 和文件系统 provider。server 保留现有 `New(logger)`，使用无 roots 的默认 provider；增加带 `SourceProvider` 的 options/constructor 供生产和测试注入，`partialParseContextAt` 从 provider 复制 roots/available 状态，不猜测默认路径。

## 构建规则

- AST 递归遍历建立 scope tree；namespace/class/function/block 进入或退出 lexical scope。
- declaration/definition 的 name range 使用 parser 提供的 `NameRange`；无法确认时保留 incomplete/unknown 状态。
- 作用域和 qualified name 在 index 中计算；同名不同作用域不合并。
- 通过 normalized declaration anchor 生成 symbol ID；不使用绝对 line/byte offset 作为唯一身份。
- 遍历声明、attribute、builtin/header source 和表达式/调用 token，建立 `SymbolRecord` 与 `ReferenceRecord`。
- 对 unresolved/ambiguous 绑定保留候选集合，不产生虚假唯一目标；alias 作为 `Object` document symbol kind 编码。
- 由 `ConditionalRegions` 给节点、符号和引用附加 active/inactive/unknown provenance。
- 建立 requester -> header 和 header -> dependent 的双向关系，检测 include cycle；循环边记录 `cycle`，已加载的非循环部分继续建立索引，禁止递归读取循环边。

## 提交规则

1. server 在现有分析队列中取得 `DocumentState` 快照。
2. builder 在 server 锁外构建 index snapshot；header provider 读取也在锁外执行。
3. 构建完成后检查 document version、context version、generation、epoch、文档是否仍打开。
4. 只有全部匹配时才在 `index.Store` 中原子替换当前 URI 快照。
5. 替换时删除当前文档旧快照中的已删除符号，不能增量保留过期声明。
6. header 版本变化使直接/间接 dependent snapshot 标记 stale，并排队重新分析；对未知依赖范围可以扩大失效，但不能发布旧 header 符号。
7. 文档关闭后删除文档快照和依赖边，迟到 builder 结果被丢弃。

## 查询规则

- query 只读取与当前 document/context/header 版本匹配的快照。
- 有 active 候选时，definition 返回全部 active 候选，references 返回其引用并集。
- 没有 active 候选时才使用 unknown 候选，并通过 diagnostics/log 表达 degraded。
- inactive 候选留在 provenance，不进入 definition/references；documentSymbol 只构建 active/unknown 节点。
- 当前新鲜快照无匹配时返回空数组；没有新鲜快照或查询期间版本失效时返回标准 `null`；不回退到旧快照。
- `includeDeclaration` 只影响 references 的输出位置，不改变 index 中的 declaration/definition 记录。

## 状态和失效

| 状态 | 进入条件 | 可查询行为 |
|---|---|---|
| `complete` | parse 和所有需要的来源都可确认 | 正常查询 |
| `partial` | parser recovered/incomplete，但有局部符号 | 返回局部结果，保留状态 |
| `degraded` | missing header、unknown condition、ambiguous candidate 等 | 返回可确认结果并通过 diagnostics/log 说明 |
| `stale` | 文档/header/context/generation/epoch 变化 | 不可发布；等待新快照 |
| `discarded` | 旧任务完成或文档关闭 | Store 内部生命周期标记，不是 `IndexSnapshot.index_status`，不可查询 |

parser 到 snapshot 的状态映射必须固定：`complete` 且来源/条件/候选可确认才是 `complete`；`recovered`/`partial`/`invalid` 但仍有 root/symbol 是 `partial`；missing/unreadable/unresolved header、include cycle、unknown condition、ambiguous candidate 或 provider error 优先是 `degraded`；任何版本变化后的快照是 `stale`，旧任务或关闭文档结果进入 Store 的 `discarded` 生命周期标记。

## 错误和可观测性

- 缺 header、不可读取文件和 unknown condition 是 index status/diagnostic，不把当前文档文本回滚。
- 索引构建失败必须有稳定 error kind 和脱敏 URI；不得记录完整源代码或凭据。
- `analysis-degraded` 必须发布稳定 diagnostic code `analysis-degraded` 并记录 `analysis_degraded` 事件；stale 不发布诊断，只记录 `stale_result` 事件并让查询返回 `null`。
- include cycle 的 `analysis-degraded` 诊断必须带 reason `include-cycle`，并明确已加载的非循环符号仍可查询。
- 日志至少包含 operation、脱敏 document ID、symbol kind、index status、condition、document/context/header version、candidate count、duration、diagnostic code 和 error kind。
- parser/index 的诊断位置来自原始文档；header related information 只使用 provider 已验证的 URI/range。

## 兼容性

- `internal/document` 仍只保存文档文本和版本；不把 AST/index 放入 document store。
- `internal/position` 继续是唯一的 UTF-16/CRLF range 转换入口。
- server 原有 analysis queue、cancel、shutdown、exit 和 transport framing 行为保持不变。
- protocol 只增加标准导航类型和 capability，旧客户端可以忽略新增可选能力。
