# Feature Specification: P1.3 Tops C++ AST 和符号索引

**Feature Branch**: `005-ast-symbol-index`

**Created**: 2026-10-10

**Status**: Draft

**Input**: User description: "目标：为定义跳转、引用查找、文档符号和后续语义检查建立 AST 与索引。实现 Tops C++ language server 的“AST 和符号索引”功能。定义函数、类型、属性、内建变量、header 符号、宏条件、定义位置和引用关系的记录方式。必须包含文档变化后的索引更新、声明或函数尚未写完时的部分索引、缺少 header 和多文件导航场景，以及与 LSP definition/references/documentSymbol 的契约。"

## Clarifications

### Session 2026-10-10

- Q: 宏条件为 `unknown` 时，definition/references 的导航结果如何处理？ → A: 优先返回 `active` 分支；存在一个或多个 active 候选时只返回 active 候选并按多候选规则处理；只有不存在 active 候选时才返回 unknown 分支候选，并标记 `analysis-degraded`；`inactive` 分支不作为当前导航结果。
- Q: header 的索引来源是什么？ → A: 索引已打开文档，以及配置 include roots 中被引用且可读取的 header；不进行 workspace 全量扫描，无法读取的 header 保留为 `unresolved`。
- Q: 存在多个 active 候选时如何导航？ → A: definition 返回全部 active 候选，references 返回这些候选引用的并集，并通过 `analysis-degraded` 说明歧义；不按最近作用域静默选择一个候选。
- Q: documentSymbol 是否展示条件分支中的符号？ → A: 展示 active 和 unknown 分支，隐藏 inactive 分支；条件状态继续保留在索引和降级状态中。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 在当前文件中导航声明和引用 (Priority: P1)

作为 Tops C++ 开发者，我希望语言服务器记录函数、类型、属性和内建变量的声明、定义及引用，从而可以从编辑器中的名称跳转到对应位置，并查找该名称在当前文件中的使用位置。

**Why this priority**: 定义跳转和引用查找是 AST 与符号索引直接提供的核心价值，也是后续语义检查、补全和悬停结果的共同基础。

**Independent Test**: 打开一个只依赖当前文件的 Tops C++ 测试材料，在函数、类型、属性和内建变量的声明、定义及引用位置分别发起导航请求；检查返回位置、作用域和空结果行为，不需要多文件或设备运行时。

**Acceptance Scenarios**:

1. **Given** 一个文件包含 namespace、函数声明和函数定义，**When** 用户在函数调用处请求 definition，**Then** 返回该函数定义位置；如果只有声明，则返回声明位置并保留“只有声明”的索引状态。
2. **Given** 一个文件包含 class/struct/enum、类型别名及其使用，**When** 用户在类型名称处请求 definition，**Then** 返回与当前条件状态匹配的类型声明或定义位置，不返回同名但作用域不匹配的符号。
3. **Given** 一个文件包含 Tops 属性、标准属性或内建变量的多个使用，**When** 用户请求 references 且 `includeDeclaration` 为 `true`，**Then** 返回声明/来源位置和所有当前可确认的引用位置；当 `includeDeclaration` 为 `false` 时不返回声明位置。
4. **Given** 一个名称没有可确认的定义，**When** 用户请求 definition 或 references，**Then** 返回空结果，不创建虚假的文件、行列或 header 位置。

### User Story 2 - 查看文件、header 和多文件的符号结构 (Priority: P1)

作为维护多文件 kernel 的开发者，我希望索引能关联当前文档、已打开的 workspace 文件和按需加载的 header 中的符号，从而可以跨文件跳转、查找引用并查看包含嵌套作用域的 document symbols。

**Why this priority**: Tops C++ 代码通常把类型、属性、内建声明和 API 放在 header 中；没有跨文件索引，导航只能在单个源文件内工作，无法支持真实工程。

**Independent Test**: 使用至少一个源文件、一个可加载 header 和一个从 header 引用类型/函数的文件组成测试材料；再分别移除 header，检查可用结果、降级状态和缺失来源诊断。

**Acceptance Scenarios**:

1. **Given** `main.tops` include 一个可加载 header，header 中定义类型和函数，**When** 用户在 `main.tops` 中请求 definition，**Then** 返回 header 的 URI 和准确范围；请求 references 时包含跨文件引用。
2. **Given** 两个文件定义相同短名称但属于不同 namespace，**When** 用户在限定名称或对应引用处请求导航，**Then** 只返回限定作用域内的候选，不把同名符号合并成一个定义。
3. **Given** include 指向的 header 无法解析、目标不存在或目标不可读取，**When** server 建立当前文件索引，**Then** 当前文件中已解析的符号仍可用于 documentSymbol 和本地导航；分别记录 `unresolved`、`missing` 或 `unreadable` dependency，并产生可定位的 `analysis-degraded` 结果，不把缺失内容当作空 header。
4. **Given** header 内容发生变化，**When** server 接收该 header 的新文档版本，**Then** header 索引和依赖它的源文件索引进入 stale，并按新版本重新生成；旧 header 定义不得继续覆盖新结果。

### User Story 3 - 在编辑和不完整输入中保持部分索引 (Priority: P1)

作为正在逐字编辑 kernel 的开发者，我希望文档缺少右括号、函数体或部分参数时，已经识别的声明和引用仍能被索引，同时旧版本结果不会污染新版本。

**Why this priority**: 编辑器持续发送中间文档状态；如果一次语法错误就清空索引，导航和后续诊断会在最需要反馈时失效。

**Independent Test**: 对同一文件提交一组递增的 `didChange` 文档版本，依次截断函数参数、属性参数、函数体和条件分支；每个版本检查 `documentSymbol` 和可确认的 definition/references，并检查旧版本结果是否被丢弃。

**Acceptance Scenarios**:

1. **Given** 文件前部已有完整类型和函数声明，末尾函数尚未写完，**When** server 解析当前版本，**Then** 返回前部完整符号和已识别的部分函数符号；部分符号带 `incomplete`/`partial` 状态，不阻止后续查询。
2. **Given** 用户连续提交版本 `v1`、`v2`、`v3`，**When** `v1` 的索引任务晚于 `v3` 完成，**Then** server 丢弃 `v1` 结果，不发布或返回覆盖 `v3` 的定义、引用或 document symbols。
3. **Given** 新版本删除一个声明，**When** 新版本索引完成，**Then** 旧版本中的该符号不再出现在新版本查询结果中；未受影响且在新快照中仍存在的符号继续可用。
4. **Given** 文件只剩下未完成的声明或条件区域，**When** server 建立索引，**Then** 返回可描述当前文本的部分快照和原因明确的降级状态，不因恢复失败而崩溃或复用整份旧索引。

### User Story 4 - 通过标准 LSP 方法使用导航结果 (Priority: P1)

作为 VS Code 用户，我希望导航能力使用标准 LSP 请求和结果类型，从而现有客户端可以直接消费 definition、references 和 document symbols，而不需要在 TypeScript 客户端重复实现语义判断。

**Why this priority**: LSP 契约决定 Go server 与客户端的边界；契约不清会导致能力声明、空结果、取消和过期结果在两侧出现不一致。

**Independent Test**: 通过现有 stdio LSP 会话发送 `initialize`、文档同步和三项导航请求，验证能力声明、标准参数、结果 JSON、空结果、非法参数、取消和文档版本竞争行为。

**Acceptance Scenarios**:

1. **Given** server 完成初始化且导航能力可用，**When** client 查看 `initialize` 返回值，**Then** server 声明 `definitionProvider`、`referencesProvider` 和 `documentSymbolProvider`，且不要求客户端使用私有语义消息。
2. **Given** 请求参数合法且索引已准备好，**When** client 发送 `textDocument/definition`、`textDocument/references` 或 `textDocument/documentSymbol`，**Then** server 返回对应标准 LSP 结果；没有匹配项时返回空结果而不是任意位置。
3. **Given** 请求参数缺失、URI 未打开或位置无效，**When** server 处理请求，**Then** 返回可识别的标准 JSON-RPC/LSP 参数错误，不崩溃、不调用客户端语义 fallback。
4. **Given** 请求执行期间文档或上下文版本发生变化，**When** 旧请求完成，**Then** server 丢弃旧索引结果；请求以空/无结果或标准取消结果结束，不能让旧结果覆盖更新后的结果。

### Edge Cases

| 场景 | 预期行为 | 验证方式 |
| --- | --- | --- |
| 函数只有前置声明，没有函数体 | definition 返回声明位置；符号状态记录为 declaration-only | 单文件 declaration-only fixture |
| 一个函数有多个声明或定义冲突 | 记录所有声明和候选定义；definition 返回全部 active 候选，references 返回候选引用的并集，并报告可解释的 ambiguity/degraded 状态 | 多声明与冲突定义 fixture |
| 同名符号位于不同 namespace/class scope | 每条记录保留 scope 和 qualified name；导航只返回作用域匹配的候选 | qualified lookup fixture |
| 属性使用来自标准或 Tops header | 记录属性 spelling、使用位置和来源 header；缺少来源时保留 unresolved，不伪造定义 | attribute source fixture |
| 内建变量由目标 header 提供 | 记录 builtin kind、目标条件和 header 来源；没有可用 header 时只返回受限索引结果 | builtin/header fixture |
| `#if`、`#ifdef`、`#ifndef`、`#elif`、`#else`、`#endif` 组成 active/inactive/unknown 分支 | 符号保留条件区域、父子关系和状态；definition/references 按 active/unknown 候选规则返回，documentSymbol 展示 active 和 unknown、隐藏 inactive | condition matrix fixture |
| include 循环 | 保留已访问 header 和循环关系，避免重复遍历或无限递归；已有符号仍可查询 | include cycle fixture |
| header 重命名或 URI 变化 | 旧 URI 的依赖关系失效，按新 URI 建立记录；不把旧位置返回为当前定义 | URI update test |
| header 缺少 include root | 当前文档本地索引继续建立；缺失依赖记录原因和请求路径，不创建空 header 符号 | missing include-root fixture |
| 文件在 UTF-8 多字节字符和 CRLF 下导航 | 内部位置转换后，LSP range 使用 0-based UTF-16 code unit，且与文档存储位置规则一致 | UTF-16/CRLF position tests |
| 名称出现在注释、字符串或 inactive 文本中 | 不把这些文本当作普通引用；预处理指令和条件元数据仍保留 | tokenizer/index filtering tests |
| 函数、属性或模板参数尚未闭合 | 保留已识别名称和父级范围，标记 incomplete；不把整个文件标记为无索引 | incomplete index tests |
| 旧结果晚于新结果完成 | 按 document version、context version 和 header version 丢弃旧结果 | stale-result integration test |
| 查询位置落在未解析 token 或错误节点 | 返回空结果或当前可确认候选，并保留 parser/index 的降级状态；不得猜测同名目标 | recovery lookup test |

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 为每个已解析文档生成与当前 `document version` 和 `context version` 关联的 AST/index 快照；快照必须能被后续语义检查和 LSP 导航查询使用。
- **FR-002**: AST 节点 MUST 保留节点类型、原始 spelling、源范围、父子关系、声明/调用关系、条件区域来源、header 来源和 `incomplete`/恢复状态；索引不得用脱离原文的名称列表替代 AST 关系。
- **FR-003**: 符号记录 MUST 至少支持 `function`、`type`、`attribute`、`builtin`、`namespace`、`field`、`macro` 和可解析的 `header` 来源；类型至少区分 class、struct、enum、union 和 alias 的来源类别。
- **FR-004**: 每个符号记录 MUST 包含稳定 `symbol_id`、`name`、`qualified_name`、`kind`、作用域、声明位置、定义位置集合、条件状态、来源 URI、解析状态以及所属 document/context/header 版本。
- **FR-005**: `symbol_id` MUST 在无关文本移动或同一声明所在行号变化时保持可复用；声明身份、作用域、名称或来源 URI 发生实质变化时允许生成新的 ID。实现必须能通过版本序列测试证明该规则。
- **FR-006**: 声明位置与定义位置 MUST 分开记录；只有声明时，定义位置集合为空且状态为 `declaration-only`，definition 查询可以返回该声明位置作为可导航目标。
- **FR-007**: 引用记录 MUST 包含引用 URI、源范围、引用类别、条件状态和一个或多个候选 `symbol_id`；无法唯一解析时必须保留 `unresolved`/`ambiguous` 状态，不得按短名称强行绑定。
- **FR-008**: 函数索引 MUST 记录函数名、限定名、声明/定义位置、参数摘要、返回类型摘要、执行空间限定符、属性来源以及引用关系；这些摘要必须能回溯到声明头 token；本功能不要求完成重载决议或类型兼容检查。
- **FR-009**: 类型索引 MUST 记录 class、struct、enum、union、alias 等类型声明的名称、限定作用域、成员/嵌套符号关系、声明/定义位置和引用关系；不把同名不同作用域的类型合并。
- **FR-010**: 属性索引 MUST 记录标准属性和 Tops 属性的 spelling、参数 token、使用位置、可确认的定义/header 来源、适用条件和解析状态；未知属性可以保留为 opaque/unresolved，但不得阻断同一声明的其他索引。
- **FR-011**: 内建变量索引 MUST 记录名称、字段访问关系、builtin 类别、目标/宏条件、声明或 header 来源、引用位置和可用状态；没有可验证来源时不得创建虚假的定义 URI 或范围。
- **FR-012**: header 索引 MUST 只按需加载被当前文档或已打开 workspace 文件引用、且能从配置 include roots 解析并读取的 header；记录 header URI、include 关系、来源状态、内容版本或等价版本标识以及其中可解析的符号。同一 header 被多个源文件引用时应复用同一版本索引，而不是为每个调用方复制不同定义。
- **FR-013**: 对 `#if`、`#ifdef`、`#ifndef`、`#elif`、`#else` 和 `#endif`，系统 MUST 保存条件表达式、分支范围、父子条件关系和 `active`、`inactive` 或 `unknown` 状态；每个受条件影响的符号和引用都必须关联对应条件记录。
- **FR-014**: document symbols MUST 展示当前文档中可识别且处于 `active` 或 `unknown` 分支的 namespace、类型、函数、字段、属性和其他适合标准 `DocumentSymbol` 的嵌套结构；`inactive` 分支符号不得进入当前 `DocumentSymbol[]`。符号的 `range` 覆盖声明结构，`selectionRange` 只覆盖名称或最小可选位置，条件状态继续保留在索引和降级状态中。
- **FR-015**: document symbols 对不完整输入 MUST 保留已经识别的符号，并以稳定范围返回；不能因为一个恢复节点而清空其他顶层或嵌套符号。
- **FR-016**: 索引更新 MUST 在 `didOpen`、成功的 `didChange`、header 内容变化、include/context 变化后触发；更新以当前版本快照为准，删除或修改的声明不得从旧快照继续泄漏。
- **FR-017**: header 变化 MUST 使直接或间接依赖该 header 的索引进入 stale，并重新计算受影响文件；无法确定依赖范围时可以扩大失效范围，但不得继续发布已知过期的 header 符号。
- **FR-018**: 文档、header 或 context 的新版本到达后，任何旧版本 AST/index/query result MUST 被丢弃或明确阻止发布；旧任务不得覆盖新版本的索引、诊断或 LSP 响应。
- **FR-019**: 索引 MUST 支持当前打开文档、已打开 workspace 文件和配置 include roots 中按需解析的可读取 header 之间的多文件关系；不得因为本功能启动而扫描整个 workspace，URI、相对 include 路径和规范化后的文件身份必须保持一致。
- **FR-020**: 缺少 header、include root 或宏上下文时，系统 MUST 保留当前文档中已确认的局部符号，记录缺失依赖和原因，并产生 `analysis-degraded` 或等价可定位状态；不得把缺少内容静默视为空文件或完整成功。
- **FR-021**: 对 inactive 分支，系统 MUST 保留其 AST/index provenance 以支持 document symbols 和后续重新选择上下文；definition/references 不得把 inactive 记录作为当前导航结果返回。对 unknown 分支，definition/references MUST 先检查 active 候选；存在一个或多个 active 候选时只返回 active 候选，只有不存在 active 候选时才可以返回 unknown 候选，并将结果标记为 `analysis-degraded`，不得猜测目标宏值。
- **FR-022**: 索引查询 MUST 按作用域、限定名、当前条件状态和来源版本筛选候选；当存在多个 active 候选时，definition MUST 返回全部 active 候选位置，references MUST 返回这些候选引用的并集，并通过 `analysis-degraded` 说明歧义；不得按最近作用域静默选择唯一关系。没有 active 候选时才按 FR-021 处理 unknown 候选，未解析候选不得伪装成已确认关系。
- **FR-023**: 系统 MUST 实现标准 LSP `textDocument/definition` 契约：输入使用标准 `TextDocumentPositionParams`；对当前新鲜快照的合法请求返回 `Location[]`，没有匹配项时返回空数组；定义不可确认时返回空数组或可验证的声明位置，不返回不存在的 URI/range；没有新鲜快照或快照在查询期间失效时返回标准 `null`，不得返回旧快照位置。
- **FR-024**: 系统 MUST 实现标准 LSP `textDocument/references` 契约：输入使用标准 `ReferenceParams`，尊重 `context.includeDeclaration`；对当前新鲜快照的合法请求返回 `Location[]`，没有匹配项时返回空数组；没有新鲜快照或快照在查询期间失效时返回标准 `null`，不得返回旧引用集合，不使用私有语义负载替代标准位置。
- **FR-025**: 系统 MUST 实现标准 LSP `textDocument/documentSymbol` 契约：输入使用标准 `DocumentSymbolParams`；对当前新鲜快照返回嵌套 `DocumentSymbol[]`，空文档或无可见符号时返回空数组；没有新鲜快照或快照在查询期间失效时返回标准 `null`，结果包含 `name`、标准 `kind`、`range` 和 `selectionRange`。
- **FR-026**: `initialize` 返回的 capabilities MUST 在对应方法可用时声明 `definitionProvider`、`referencesProvider` 和 `documentSymbolProvider`；TypeScript client 不得复制索引或语义判断。
- **FR-027**: 导航请求参数缺失、URI 未打开、位置越界或请求结构不合法时，server MUST 返回标准 JSON-RPC/LSP 参数错误；请求处理不得导致 server 进程退出。
- **FR-028**: 导航请求遇到缺失 header、unknown 条件或不完整 AST 时，server MUST 尽可能返回当前版本可确认的部分结果，并通过既有诊断/日志状态说明 `analysis-degraded`；不得把部分结果包装成完整语义结论。
- **FR-029**: 导航请求在取消后 MUST 停止使用该请求的结果；如果请求对应文档或 context 已过期，server MUST 丢弃旧结果并以空/无结果或标准取消错误结束，不得让旧结果覆盖新请求结果。
- **FR-030**: definition、references 和 documentSymbol 的范围 MUST 使用 LSP 0-based line 与 UTF-16 code-unit character，采用半开区间；内部 byte/rune 位置必须经过统一转换后才能发布。
- **FR-031**: 索引和查询日志 MUST 只记录脱敏 URI、operation、symbol kind、版本、条件状态、结果数量、耗时、稳定 diagnostic code 和 error kind 等必要信息，不得记录完整源代码、凭据或无关用户数据。当前 `analysis-degraded` 状态必须发布稳定 code 为 `analysis-degraded` 的当前文档诊断并记录 `analysis_degraded` 事件；`stale-result` 只记录 `stale_result` 事件及旧/新版本，不得发布过期诊断。
- **FR-032**: Go server MUST 拥有 AST/index 构建、查询、版本失效和 LSP 结果编码；TypeScript client 只负责生命周期、请求转发和结果展示；Clang/clangd 不得成为运行时索引后端。
- **FR-033**: 本功能 MUST 不把完整模板实例化、重载决议、完整类型检查、完整宏展开、链接语义或设备执行结果当作索引完成条件；这些能力的缺口必须在索引状态或后续规格中保持可见。
- **FR-034**: 测试 MUST 覆盖函数、类型、属性、内建变量、header 符号、宏条件、声明/定义、引用、document symbols、文档变化、部分索引、缺少 header、跨文件、作用域歧义、UTF-16 位置、取消和 stale 结果。
- **FR-035**: 测试 MUST 同时包含 parser/index 单元测试、索引快照与版本测试以及 stdio LSP 契约/集成测试；正常、错误、缺失上下文和不完整输入都必须有可复现材料。
- **FR-036**: 索引单元测试和 LSP 契约测试 MUST 可以在没有 GCU 设备、GCUSIM、clangd 或设备 workload 的开发机环境中运行；外部 header 只作为显式测试数据或配置输入。

## Scope and Boundaries

### In Scope

- 在现有可恢复 AST 和条件区域基础上建立文档级、按需 header 级和跨打开文件的符号索引。
- 记录函数、类型、属性、内建变量、namespace、field、macro 及其声明、定义、引用、作用域和来源状态。
- 记录 `active`、`inactive`、`unknown` 宏条件及其与 AST、符号和引用的关联。
- 建立当前文档、可读取 header 和多文件之间的 include/dependency 关系。
- 在 `didOpen`、`didChange`、header/context 变化后更新索引，并按 document/context/header 版本丢弃过期结果。
- 在不完整 AST、缺少 header、未知条件和局部恢复状态下提供部分索引和可解释的降级状态。
- 通过标准 LSP `textDocument/definition`、`textDocument/references` 和 `textDocument/documentSymbol` 提供导航结果。
- Go 单元测试、索引快照测试、版本/并发测试、LSP 契约测试和多文件测试材料。

### Out of Scope

- 完整 C++ 类型检查、模板实例化、重载决议、隐式转换、常量求值、ODR、链接和代码生成语义。
- 完整宏展开、token paste/stringification 的等价展开、宏递归语义和超出当前 parser/context 的预处理器模拟。
- 根据目标名称猜测宏值、架构限制或 builtin 可用性；目标相关事实由有效 `CompilationContext`、header 和后续 semantic 层提供。
- completion、hover、rename、格式化、重构和 semantic tokens 的完整用户功能；它们可以消费本功能的索引，但不属于本次 LSP 交付。
- 修改 clangd、调用 Clang 作为 server runtime fallback、运行设备 workload 或把 GCU 运行结果作为索引通过条件。
- TypeScript client 中的语义解析、索引缓存或自定义导航协议。

## Key Entities *(include if feature involves data)*

### ASTSnapshot

表示某个文档和上下文版本的可恢复语法结构。至少包含文档 URI、document/context version、根节点、节点 ID、source range、原始 token/spelling、条件区域引用、header provenance、解析状态和恢复标记。快照不拥有 LSP 生命周期或目标合法性判断。

### SymbolRecord

表示可以被导航或被后续语义使用的声明实体。至少包含：

| 字段 | 说明 |
| --- | --- |
| `symbol_id` | 在声明身份稳定时可复用的索引标识 |
| `name` / `qualified_name` | 源码名称和完整作用域名称 |
| `kind` | `function`、`type`、`attribute`、`builtin`、`namespace`、`field`、`macro` 或其他已定义类别 |
| `scope` | namespace、class、函数或 header 作用域 |
| `declarations` | 一个或多个声明位置 |
| `definitions` | 一个或多个定义位置；没有定义时为空 |
| `references` | 引用记录或可查询的引用集合 |
| `condition` | 所属宏条件、分支状态和条件范围 |
| `origin` | 当前文档、workspace 文件、header、builtin source 或 unresolved 来源 |
| `status` | active、conditional、declaration-only、incomplete、unresolved、ambiguous 或 stale |
| `version` | document/context/header 版本关联 |

### ReferenceRecord

表示名称在源文本中的一次引用。至少包含 URI、source range、引用类别、词法/限定作用域、候选 `symbol_id` 集合、条件状态和解析状态。引用范围不能来自宏展开后不存在的虚拟文本。

### ConditionalRegion

表示一组 `#if` 及其分支。至少包含条件表达式范围、分支范围、父级区域、分支状态和受影响节点/符号关系。未知状态不能被隐式改写为 active 或 inactive。

### HeaderDependency

表示源文件或 header 对另一个 header 的依赖。至少包含 requester URI、target URI 或 unresolved include spelling、搜索来源、include range、内容版本、依赖状态和受影响索引列表。缺失 target 时保留请求信息，不生成空 header。

### IndexSnapshot

表示一次可查询的索引结果。至少包含文档集合、header 集合、符号表、引用表、作用域关系、include graph、条件状态、版本集合、完整/部分状态和失效原因。新快照发布前必须通过版本检查。

## LSP Contract

### Capability and ownership

| 方法/能力 | 方向 | 本功能契约 |
| --- | --- | --- |
| `initialize` capabilities | server -> client | 在可用时声明 `definitionProvider`、`referencesProvider`、`documentSymbolProvider` |
| `textDocument/definition` | client -> server | Go server 查询索引并返回标准位置；client 只发起请求和展示结果 |
| `textDocument/references` | client -> server | Go server 按 `includeDeclaration` 查询引用；client 不自行扫描文本 |
| `textDocument/documentSymbol` | client -> server | Go server 返回当前文档的嵌套标准符号；client 不重建层级 |
| 文档同步和 context 变化 | client -> server | 变化使相关 AST/index/query result 失效并触发重新分析 |

### Result and error behavior

| 请求 | 成功结果 | 无匹配或受限结果 | 错误/过期行为 |
| --- | --- | --- | --- |
| `textDocument/definition` | 当前新鲜快照返回 `Location[]`，优先返回定义；只有声明时返回声明 | 存在一个或多个 active 候选时返回全部 active 候选；没有 active 候选时才返回 unknown 候选并保留降级状态；当前快照无匹配时为空数组；无新鲜快照时为 `null` | 参数错误使用标准 JSON-RPC/LSP 错误；取消使用标准取消错误；过期结果返回 `null`且不得发布旧位置 |
| `textDocument/references` | 当前新鲜快照按 `includeDeclaration` 返回 `Location[]` | 存在一个或多个 active 候选时返回这些候选引用的并集；没有 active 候选时才返回 unknown 候选并保留降级状态；当前快照无匹配时为空数组；无新鲜快照时为 `null` | 参数错误使用标准错误；取消使用标准取消错误；旧版本结果丢弃 |
| `textDocument/documentSymbol` | 当前新鲜快照返回 active 和 unknown 分支的嵌套 `DocumentSymbol[]` | 隐藏 inactive 分支；空文档或无可见符号时为空数组；不完整文档仍返回已识别符号；无新鲜快照时为 `null` | 参数错误使用标准错误；旧快照不得覆盖新快照 |

- 三项请求使用标准 LSP 参数和结果类型，不增加依赖 TypeScript 的私有字段。
- `DocumentSymbol.kind` 使用标准 `SymbolKind`：函数映射为 function/method，class/struct/union 映射为 class/struct，enum 映射为 enum，alias 映射为 object，namespace 映射为 namespace，field 映射为 field，属性映射为 property，内建变量映射为 variable，宏映射为 constant；无法映射的内部类别仍保留在索引，但不强行伪装成不匹配的 document symbol。
- definition/references 结果中的每个 `Location` 必须来自当前索引快照和可验证的 URI/range；不可验证的来源只能保留为 unresolved 状态。
- 缺少 header 或 unknown 条件不是把 server 变成失败状态的理由；存在 active 候选时只返回 active 候选，只有不存在 active 候选时才返回 unknown 候选，并通过 code 为 `analysis-degraded` 的当前文档诊断和 `analysis_degraded` 日志事件说明限制。多个 active 候选不静默选择单个目标，而是返回候选集合。
- 返回形状必须区分：当前新鲜快照无匹配使用空数组；没有新鲜快照或查询期间版本失效使用 `null`；取消使用标准 `RequestCancelled`，不发布 stale 诊断或旧导航结果。
- client 不得把空结果解释为“名称不存在于所有上下文”；上下文缺失和索引不完整必须由 server 的诊断或日志状态表达。

## Test Plan

### Index and AST tests

- 验证每个 SymbolRecord 的 kind、qualified name、scope、声明/定义/引用位置、origin、condition 和 version。
- 验证同一文档版本中行号移动、无关文本插入和局部修改不会错误合并或遗失符号。
- 验证 declaration-only、incomplete、unresolved、ambiguous、inactive、unknown 和 stale 状态。
- 验证 namespace/class scope、header include、include cycle、缺少 header、header 更新和依赖失效。
- 验证函数、类型、属性、内建变量和宏条件的正向、负向和不完整材料。
- 验证六类条件指令、parent-child condition 关系、header URI 变化、注释/字符串过滤、错误节点查询以及完整模板/重载/宏展开不被误报为已完成。
- 验证参数摘要、返回类型摘要、执行空间/属性来源、builtin 字段访问和缺失来源状态。
- 验证 documentSymbol 对 active/unknown/inactive 条件分支分别展示或隐藏的规则。

### LSP contract tests

- 验证 `initialize` capabilities、三项请求的标准参数与 JSON 结果、空结果和嵌套 `DocumentSymbol`。
- 验证 `includeDeclaration`、声明优先级、跨文件 URI、UTF-16 range 和 CRLF。
- 验证 URI 未打开、位置越界、缺少参数、取消、文档版本竞争、context/header 变化和过期结果。
- 验证 client 不需要 clangd、Clang runtime 或私有语义消息即可消费结果。

### Test materials

测试材料文件按 `single-file`、`multi-file`、`headers`、`conditions`、`incomplete`、`missing-context` 和 `stale` 目录分类；manifest 的 `category` 固定为 `core-symbol`、`multi-file`、`missing-context`、`unknown-condition`、`incomplete`、`version-race`、`header-change` 或 `out-of-scope`。`testdata/index/manifest.json` 的每条记录至少包含 `case_id`、`category`、源文件 URI、依赖文件、language standard、宏 context、目标/pass 标识、输入版本序列、`expected_symbol_count`、`expected_location_count`、预期符号、预期引用、预期 LSP range、预期状态、LSP 操作和验证命令；验证器必须输出每类的总数、通过数和失败 case ID。真实 Tops corpus 只读使用，不修改来源文件。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `manifest.json` 中标记为 `core-symbol` 的函数、类型、属性、内建变量和 header case 必须全部通过；验证器按 `case_id` 统计总数和通过数，100% 的记录返回可验证 URI/LSP range，或符合记录的空/降级结果。
- **SC-002**: `manifest.json` 中所有 `version-race` 和 `header-change` case 的最终查询必须 100% 使用最新 document/context/header version；验证器必须报告旧版本结果覆盖次数为 0。
- **SC-003**: `manifest.json` 中所有 `multi-file`、`missing-context`、`unknown-condition` 和 `incomplete` case 的局部符号通过数必须等于 `expected_symbol_count`，且虚假 header URI/source range 计数为 0。
- **SC-004**: `textDocument/definition`、`textDocument/references` 和 `textDocument/documentSymbol` 的标准契约测试全部通过，包含成功、空结果、参数错误、取消、UTF-16 位置和过期结果场景。
- **SC-005**: stdio integration 中的标准 LSP JSON decoder proxy 必须成功解析每个 documentSymbol 结果的嵌套范围，并完成文件内和跨文件导航；不需要索引内部字段或私有扩展消息。真实 VS Code client UI 不属于本功能测试依赖。
- **SC-006**: 在没有 GCU 设备、GCUSIM、clangd 或设备 workload 的开发机环境中，索引单元测试和 LSP 契约测试均可独立运行并给出可定位失败信息。

## Assumptions

- 上游 parser 已提供可恢复 AST、原始 token/source range、条件区域和文档/context 版本；本功能可以扩展 AST 字段，但不重新定义 tokenizer 的位置规则。
- `CompilationContext`、include roots、宏值和目标/pass 信息由 Go server 的上下文层提供；索引不根据目标名称推断缺失值。
- header 只有在当前文档或已打开 workspace 文件引用、且能从配置 include roots 解析并读取时才建立内容索引；不可读取的 header 只建立 unresolved dependency。
- 本功能按需使用已打开文件和被引用的可加载 header 快照，不扫描工作区中未被打开、配置或引用的所有文件。
- 索引以词法名称、作用域、限定名和现有 AST 关系为基础；完整重载决议和模板实例化由后续语义功能负责。
- `analysis-degraded`、`stale-result` 和相关版本字段沿用项目已有 LSP/诊断边界；如需要新增公开 code，必须在后续计划中记录兼容性影响。
- 本功能新增标准 LSP 能力时，旧 client 可以忽略新增可选 capabilities；不新增必须由 client 实现的私有导航协议。
