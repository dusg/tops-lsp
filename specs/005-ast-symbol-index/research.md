# 研究记录：AST 和符号索引

**功能**：[spec.md](spec.md)
**日期**：2026-10-10

## 结论摘要

本功能采用进程内、版本化的符号索引，不引入持久化数据库、第三方 parser 或 Clang runtime。`internal/parser` 继续负责可恢复 AST 和条件区域；新的 `internal/index` 负责从 `parser.ParseResult` 建立符号、引用、作用域和 header 依赖；`internal/server` 保存最新快照并负责版本提交和 LSP 请求；`internal/protocol` 只增加标准导航参数、结果和能力字段。

索引只处理当前打开文档、已打开 workspace 文件，以及从 `ParseContext.IncludeRoots` 按需解析并读取的被引用 header。它不扫描整个 workspace。definition、references 和 documentSymbol 使用标准 LSP 类型；条件状态和降级原因通过索引状态、既有 diagnostics/log 表达，不增加私有 response 字段。

## 已核对的当前基线

- `ParseResult` 已包含 tokens、AST root、conditional regions、parser diagnostics、parse status、document/context version 和 `ConsumedBytes`。[ast.go](../../internal/parser/ast.go#L102)
- `SyntaxNode` 已包含节点类型、范围、名称、声明/类型 token、子节点、条件状态和 incomplete 标记，但没有名称精确范围、作用域、声明角色、稳定节点 ID 或引用关系。[ast.go](../../internal/parser/ast.go#L28)
- `ParseContext` 已包含文档 URI、宏、target/pass、include roots、文档版本和上下文版本。[ast.go](../../internal/parser/ast.go#L77)
- server 已有每 URI 一个 pending/running 分析队列，并按 document snapshot、generation、epoch 和 context version 丢弃过期解析结果。[analysis.go](../../internal/server/analysis.go#L1) [parse_context.go](../../internal/server/parse_context.go#L104)
- server 当前只发布 parser diagnostics，不保存最新 `ParseResult` 或 index snapshot。[parse_context.go](../../internal/server/parse_context.go#L38)
- protocol 当前只有基础生命周期、文档同步、取消和 diagnostics 模型；没有 definition、references、documentSymbol 的参数、结果或 capability 字段。[messages.go](../../internal/protocol/messages.go#L9) [diagnostics.go](../../internal/protocol/diagnostics.go#L14)
- document store 严格要求版本递增，`Get` 返回文档值快照；position mapper 已处理 UTF-16、CRLF、EOF 和 byte range 转换。[store.go](../../internal/document/store.go#L39) [mapper.go](../../internal/position/mapper.go#L32)
- `go.mod` 声明 Go `1.26.8`，当前模块没有新增依赖需求。[go.mod](../../go.mod#L1)

## 决策 1：索引所有权和存储

**Decision**：新建 `internal/index`，提供不可变的 `IndexSnapshot` 和线程安全的 `Store`；server 在现有分析队列中构建快照，经过 generation/context/version 检查后替换当前快照。

**Rationale**：

- parser 不应拥有 workspace 生命周期、header 依赖或 LSP 查询。
- server 已经拥有文档快照和过期检查，复用它可以避免第二套竞态控制。
- 进程内快照足以满足当前功能，且不会引入数据库 schema、持久化失效和额外外部依赖。
- builder 在锁外工作，commit 在版本检查后完成，避免解析或 header 读取持有 server 全局锁。

**Alternatives considered**：

- 把索引字段直接塞入 `internal/server.Server`：拒绝，因为会把符号模型、header 图和查询规则与 LSP 生命周期混在一起。
- 使用持久化数据库：拒绝，当前需求只覆盖打开文档和按需 header，持久化会增加失效、隐私和迁移成本。
- 让 parser 直接返回 LSP `DocumentSymbol`：拒绝，parser 不应拥有 LSP 类型，也无法表达跨文件引用和索引版本。

## 决策 2：AST 与索引的边界

**Decision**：在 parser AST 上补充最小结构元数据：声明名称的 `NameRange` 和声明角色 `declaration`/`definition`/`unknown`；必要时保留函数/类型的 body 范围。作用域、限定名、symbol ID 和引用候选由 index builder 计算。

**Rationale**：

- `SyntaxNode.Range` 覆盖整个声明，不能稳定提供 `DocumentSymbol.selectionRange` 或声明名称位置。
- 声明/定义区别属于可由语法确认的结构信息，不需要引入完整类型语义。
- 作用域和限定名依赖父节点遍历；放在 index 可以同时处理 header、条件状态和 workspace 关系。
- 不增加 parser 对 LSP、文件系统或 symbol store 的依赖。

**Alternatives considered**：

- 由 index 从所有 token 重新猜测名称范围：拒绝，容易与 parser 对声明、属性、模板和 launch 的判断分叉。
- 在 parser 阶段完成符号解析和重载决议：拒绝，超出 P1.3 范围，也会把 target/header 语义提前耦合。
- 为每个 AST 节点分配基于遍历序号的稳定 ID：拒绝，插入前置节点会改变序号，不能满足无关文本移动的 ID 复用要求。

## 决策 3：稳定 symbol ID

**Decision**：使用规范化声明身份生成可复用 ID：`origin URI + kind + qualified name + declaration role + normalized declaration anchor`。anchor 由去除 trivia、函数体内容和不影响声明身份的 token 后的声明头生成；同名同签名重复声明使用确定的声明序号处理。

**Rationale**：

- URI、作用域和名称可以区分大部分符号；声明头可区分同名函数候选，而不依赖行号。
- 行移动、无关文本插入和函数体修改不会改变声明身份。
- ID 不是跨 rename 或跨 URI 的永久数据库主键；声明名称、作用域、来源 URI 或签名发生实质变化时允许变化。
- 使用 Go 标准库完成规范化和摘要，不增加第三方依赖。

**Alternatives considered**：

- 直接使用 byte offset：拒绝，行移动和前置插入会使 ID 全部变化。
- 只使用短名称：拒绝，无法区分 namespace/class scope、重载和 header 来源。
- 使用完整源码 hash：拒绝，函数体或无关空白变化会导致声明 ID 不稳定。

## 决策 4：header 来源和依赖图

**Decision**：定义可注入的 `SourceProvider`/`HeaderResolver` 边界，并要求 `IncludeRoots() (roots, available)` 作为 server 构造 `ParseContext.IncludeRoots`/`IncludeRootsAvailable` 的唯一来源。优先复用已打开文档；对 include roots 中被引用的 header 按需解析和读取；无法解析 spelling 时记录 `unresolved`，找不到目标时记录 `missing`，不可读取时记录 `unreadable`，不创建空 header。保留现有 `New(logger)` 的无 roots 默认行为，并增加内部 options/constructor 供 provider 注入。建立反向依赖图，打开 header 的 `didChange` 或 source version 变化时失效依赖它的快照。

**Rationale**：

- 满足规格对多文件导航和缺少 header 的要求，同时明确排除 workspace 全量扫描。
- 现有 server 没有完整 CompilationContext resolver，也没有默认 include roots；将来源抽象为 provider，并在 `partialParseContextAt` 复制 provider roots/available 状态，可以让 unit test 使用内存文件，让未来 context 层提供真实 roots，同时保持 `New(logger)` 兼容。
- 外部文件 watcher 不属于本功能；当前可验证的变化入口是已打开 header 的文档版本、context invalidation 和 provider 返回的新版本。

**Alternatives considered**：

- 启动时扫描整个 workspace：拒绝，成本不可控，也违背规格澄清。
- 缺 header 时当作空文件：拒绝，会产生虚假的“无定义”结果并隐藏配置问题。
- server 直接在 index builder 中调用 `os.ReadFile`：拒绝，难以测试、无法复用打开文档，且把文件来源策略耦合进索引算法。

## 决策 5：条件分支与候选查询

**Decision**：以 `ParseResult.ConditionalRegions` 的分支范围和状态作为条件 provenance 的来源，符号和引用都关联条件状态。查询规则为：

1. 有一个或多个 `active` 候选时，definition 返回全部 active 候选；references 返回这些候选引用的并集。
2. 没有 active 候选时，才允许使用 `unknown` 候选，并通过 `analysis-degraded` 说明限制。
3. `inactive` 候选不进入当前 definition/references；documentSymbol 展示 active 和 unknown，隐藏 inactive。
4. 不按最近作用域静默选择单个候选，不把 unknown 改写成某个 GCU target。

**Rationale**：这同时保留编辑器中的结构信息和导航的可解释性，且不要求 P1.3 实现重载决议或完整宏展开。条件区域比单个 branch node 的条件字段更适合作为 source of truth，因为 parser 可能为分支重新建立局部节点。

**Alternatives considered**：

- 只返回 active：拒绝，context 不完整时会使 documentSymbol 和导航突然为空。
- 返回全部 active/inactive/unknown：拒绝，会把明确无效代码混入当前导航。
- 按名称选择第一个候选：拒绝，隐藏歧义并可能跳到错误定义。

## 决策 6：引用解析边界

**Decision**：index 采用 lexical scope、qualified name 和现有 AST/tokens 建立候选集合；保存 `resolved`、`unresolved`、`ambiguous` 状态，但不做 overload resolution、模板实例化或完整类型兼容检查。多个 active 候选的 LSP 结果按规格返回集合。

**Rationale**：这是可测试且符合 P1.3 边界的最小导航语义。后续 semantic 层可以消费同一 `ReferenceRecord.candidates`，不需要重做文本扫描。

**Alternatives considered**：

- 只做同名文本匹配：拒绝，会跨越作用域并产生大量误关联。
- 直接调用 Clang/clangd 解析引用：拒绝，违背 Go server 唯一语义所有权和 runtime 隔离。
- 在本功能中实现完整重载决议：拒绝，扩大范围并要求尚未确定的类型系统。

## 决策 7：LSP 与过期结果

**Decision**：新增标准 `TextDocumentPositionParams`、`ReferenceParams`、`DocumentSymbolParams`、`DocumentSymbol` 和 `SymbolKind` 模型；三项请求只返回标准 `Location[]` 或 `DocumentSymbol[]`。无当前新鲜快照时返回标准空/`null`结果；取消返回现有 `RequestCancelled`；版本过期结果在 server 内丢弃，不发布私有 stale payload。

**Rationale**：现有 protocol 已有 Position、Range、Location、错误码和 request cancellation 基础；扩展这些模型比引入自定义 JSON 更兼容。`analysis-degraded` 通过 index 状态、parser diagnostics 和脱敏日志表达，不污染标准导航结果。

**Alternatives considered**：

- 在 LSP response 中增加 `status`/`contextVersion` 私有字段：拒绝，旧 client 可能无法消费，且规格要求标准结果。
- 让 client 根据空结果重新扫描文本：拒绝，违反 Go/TypeScript 分层边界。
- 过期时继续返回旧快照：拒绝，会把已删除或已修改的定义展示给用户。

## 决策 8：测试和性能边界

**Decision**：新增 index 单测、protocol JSON 契约测试、server navigation/stale 测试和真实 stdio integration 测试；复用现有 parser/document/position 测试。第一版不增加公开延迟或吞吐目标，以 latest-only、无全 workspace 扫描、无无限递归和不阻塞请求为工程约束。

**Rationale**：当前规格的主要风险是索引正确性、版本竞态和 LSP 契约，不是已知的性能 SLA。先固定可重复的正确性门禁，避免无证据地承诺行/秒或 p95。

**Alternatives considered**：

- 现在设定未经测量的延迟/吞吐数字：拒绝，属于未验证承诺。
- 只做 index unit test：拒绝，无法验证标准 LSP 编码、能力声明和真实 stdio 行为。
- 把 Clang 对照作为所有测试前置：拒绝，parser/index/LSP 测试必须离线独立运行。

## 未决项处理结果

计划模板中的技术上下文未知项已全部通过上述决策消除：语言和版本使用现有 `go.mod`，依赖保持标准库，存储使用进程内快照，header 使用 provider 按需读取，LSP 使用标准模型，性能不设未经验证的公开数字，workspace 范围由已打开文档和被引用 header 明确限定。