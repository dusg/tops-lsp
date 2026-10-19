# Feature Specification: Tops C++ 通用 Semantic Baseline

**Feature Branch**: `006-semantic-baseline`

**Created**: 2026-10-13

**Status**: Draft

**Input**: User description: "实现 Tops C++ language server 的“通用 semantic baseline”功能。范围包括已解析声明、作用域、名称查找、基础标量/指针/引用/函数参数与返回值、已核验的 host/device 调用边界、地址空间和基础向量约束。建立稳定的 semantic diagnostic code、位置、严重级别、SemanticContext 降级和版本丢弃规则；Clang 只用于相同输入和 provenance 的离线对照，不进入 Go server runtime，也不直接转发 Clang message。模板实例化、完整重载决议、完整常量求值、完整访问控制、链接和代码生成保留为后续候选能力。"

## Clarifications

### Session 2026-10-13

- Q: v1 semantic baseline 的对外范围是什么？ → A: v1 只发布标准 diagnostics；semantic facts 仅供现有 server 能力和后续语义能力内部消费，不新增 hover、completion 或私有 semantic LSP。
- Q: 基础标量、指针和引用的兼容检查采用哪种规则？ → A: v1 支持标准标量隐式转换及可直接判断的常见指针/引用转换；用户自定义转换、模板相关转换和依赖完整 overload resolution 的转换仍不在范围内。
- Q: 未写执行空间属性的函数如何参与 host/device 检查？ → A: 在 `language=TOPS` 且 `SemanticContext=resolved` 时按 host 处理；普通 C++、语言未解析或上下文不完整时不推断默认 host，并按降级规则处理。
- Q: host/device、地址空间和向量规则判定为错误时是否必须有源码位置？ → A: 必须有；没有可验证源码范围时不得发布确定性 Error，只能发布带降级范围的 context 信息或不发布。
- Q: 没有或不确定上下文时如何创建 profile？ → A: profile resolver 始终保留现有的 `gcu400-default` fallback profile；Tops 没有 target 时通过完整默认命令选择它并生成 `resolved` SemanticContext，普通 C++ 选择 `cpp-default`，target 冲突时保留 `invalid`，相关结果都保留 provenance。
- Q: 输入编译命令缺失或没有有效编译命令时如何处理？ → A: 对 `.tops`、`languageId=tops` 或明确 Tops 模式，resolver 必须使用一条默认 `CompilationCommand`，设置 `arch=gcu400`、`driver=topscc`、`language=TOPS`、`language_standard=c++17` 和 `pass_kind=host+device`，并由它生成 `gcu400-default`；不得返回空 command 或 no-profile 结果。默认核心字段及所需来源完整时为 `resolved`，来源缺失时为 `partial`；include、宏和其他未核验参数不能被静默伪造。
- Q: `SemanticContext` 名称是否继续使用？ → A: 统一改名为 `SemanticContext`，强调它是 semantic 分析上下文，而不是待填充的 seed。
- Q: 默认编译命令除 `arch=gcu400` 外是否自动补齐字段？ → A: 自动补齐 `driver=topscc`、`language=TOPS`、`language_standard=c++17` 和默认 pass；include、宏和其他未核验参数仍不能静默填充。
- Q: 默认编译命令的 C++ standard 使用哪个值？ → A: `language_standard=c++17`。
- Q: 默认编译命令的 pass 使用哪个值？ → A: 使用组合 pass `host+device`，保留 host 和 device 两个分析视图，不把两侧规则合并成一个 pass。
- Q: `language=TOPS` 的默认命令适用于哪些文件？ → A: 仅适用于 `.tops`、`languageId=tops` 或明确 Tops 编译模式；普通 C++ 文件不静默套用。
- Q: 默认命令的标准库、Tops 头文件和工程头文件从哪里加载？ → A: 标准库和 `topscc` 提供的 Tops 头文件使用工具链默认路径；工程头文件使用 workspace 来源，缺失时降级，不扫描整个 workspace。
- Q: 默认 Tops 命令的宏从哪里加载？ → A: 使用 `topscc` 工具链内建宏和 workspace 工程宏，两类宏必须分别记录 provenance。
- Q: 合成的 GCU400 默认命令生成的 SemanticContext 使用什么状态？ → A: 使用 `resolved`；默认字段完整时按 GCU400 规则检查，同时保留 fallback provenance，表示它来自合成命令而非真实 compile_commands。
- Q: 同一位置在 host 和 device 视图结果不同，诊断如何发布？ → A: 按 pass 分开发布；只有违规 pass 发布诊断，并在 JSON `Diagnostic.data.passKind` 标明来源。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 查看声明、作用域和基础类型语义 (Priority: P1)

作为 Tops C++ 开发者，我希望语言服务器能根据当前文件和可用 header 识别声明、作用域、名称和基础类型，从而在编辑时得到稳定、可解释的语义反馈。

**Why this priority**: 声明和名称查找是函数调用、类型检查、导航和后续补全的共同基础。没有这一层，诊断只能停留在词法或语法错误。

**Independent Test**: 使用只包含单文件声明、嵌套作用域、基础标量、指针、引用和函数参数/返回值的 fixture，分别检查有效、无效和不完整输入，不依赖设备运行或 Clang runtime。

**Acceptance Scenarios**:

1. **Given** 文件包含 namespace、局部作用域、变量声明和函数声明，**When** 用户编辑或查询一个名称，**Then** server 按最近的可见作用域查找声明，并保留声明来源、范围和作用域信息。
2. **Given** 同名名称位于不同作用域，**When** 当前作用域存在唯一可见声明，**Then** server 只使用该声明；存在多个无法在本功能内消除的候选时，**Then** 返回候选集合并产生 `tops-semantic-ambiguous-name`，不得静默选择一个。
3. **Given** 函数具有基础标量、指针或引用参数以及返回类型，**When** server 建立 semantic snapshot，**Then** 参数顺序、类型摘要、返回类型和声明/定义关系可被后续 LSP 能力使用。
4. **Given** 用户正在输入未闭合的声明或函数签名，**When** server 分析当前版本，**Then** 已识别的声明和作用域继续可用，不完整部分标记为降级或 incomplete，不清空整份旧索引。

### User Story 2 - 检查 host/device、地址空间和向量约束 (Priority: P1)

作为 Tops kernel 开发者，我希望语言服务器只对已经由目标上下文和 provenance 核验过的 host/device 调用边界、地址空间关系和基础向量约束给出错误，从而避免把相邻架构或普通 C++ 规则误套到当前文件。

**Why this priority**: 这些规则直接影响 Tops C++ kernel 的可调用性和类型安全，也是普通 C++ semantic baseline 与 Tops 语义的分界。

**Independent Test**: 为至少一个 host pass 和一个 device pass 准备相同输入的有效、非法和目标边界 fixture；分别提供完整、部分和缺失的 `SemanticContext`，检查错误、降级和无诊断行为。

**Acceptance Scenarios**:

1. **Given** 当前 `SemanticContext` 明确给出函数的 host/device 属性和调用方向，**When** host 或 device 函数调用另一个函数，**Then** server 对已核验的合法边界不报错，对已核验的非法边界报告 `tops-semantic-host-device-boundary`，并把调用点作为主范围。
2. **Given** 指针或引用携带已核验的地址空间，**When** 发生明确不允许的地址空间使用或转换，**Then** server 报告 `tops-semantic-address-space`；如果转换规则缺少目标事实，**Then** 只报告 context 降级，不猜测转换是否合法。
3. **Given** 向量元素类型、lane 数、宽度或对齐约束来自当前目标 provenance，**When** 声明或基础操作违反确定约束，**Then** server 报告 `tops-semantic-vector-constraint`；缺少约束事实时不把候选限制当成确定错误。
4. **Given** 同一源文件在两个已解析 pass 或 profile 下具有不同的可见规则，**When** 用户切换 `SemanticContext`，**Then** 只有受影响的 semantic facts 和 diagnostics 变化，结果保留对应 provenance。

### User Story 3 - 在 SemanticContext 不完整和版本竞争时保持可信反馈 (Priority: P1)

作为编辑器用户，我希望上下文不完整时仍能得到标准 C++ 层面的有用结果，并且不会看到旧文档或旧上下文产生的诊断覆盖当前文件。

**Why this priority**: 编译数据库、目标宏和 header 在编辑器会话中可能晚到或发生变化；错误回退和旧结果污染会比缺少一条诊断更难排查。

**Independent Test**: 对同一文档提交连续版本和上下文版本，覆盖 resolved、partial、missing、invalid 四种 `SemanticContext`；检查发布的诊断、状态、版本字段和 stale 计数。

**Acceptance Scenarios**:

1. **Given** `SemanticContext` 只有语言标准和部分来源信息，**When** server 分析声明、作用域和基础类型，**Then** 继续执行不依赖目标的检查，并发布最多一组当前版本的 context 降级信息。
2. **Given** Tops 文件的输入编译命令缺失且工具链默认 roots/macros 和已引用 workspace 依赖可用，**When** server 创建完整的 `arch=gcu400` 默认编译命令，**Then** 使用 `profile_id=gcu400-default` 和 `SemanticContext=resolved` 按 GCU400 规则检查；不得返回空 command 或 no-profile 结果；所需来源缺失时为 `partial`，字段冲突时为 `invalid`。
3. **Given** `v1` 或旧 `SemanticContext` 的分析晚于 `v2` 完成，**When** server 准备发布结果，**Then** 丢弃旧结果，不发布旧 diagnostics、semantic snapshot 或 LSP 结果，也不覆盖 `v2`。
4. **Given** 文档、相关 header 或 context version 发生变化，**When** 新分析尚未完成，**Then** 旧 semantic 结果标记为 stale 并停止对外提供；新结果只在所有版本匹配后发布。

### User Story 4 - 使用可审查的离线 Clang 对照验证 (Priority: P2)

作为语言服务器维护者，我希望能用相同源文本和 provenance 运行离线 Clang 对照验证，从而发现语义差异，但不让 Go server 依赖 Clang 或把 Clang message 直接显示给用户。

**Why this priority**: 对照验证能提高语义规则的可审查性，但运行时依赖会破坏 Go server 的独立部署和错误归属。

**Independent Test**: 记录一次包含源文本 digest、原始/归一化参数、工作目录、target、pass、include roots、宏和工具版本的离线对照；验证 Go server 在没有 Clang/clangd 可执行文件时仍能运行同一 semantic tests。

**Acceptance Scenarios**:

1. **Given** Go semantic 输入和离线 Clang 输入拥有相同的源文本与 provenance，**When** 执行对照验证，**Then** 比较诊断类别、位置和确定程度，并保留差异记录。
2. **Given** provenance 缺字段或两侧输入不一致，**When** 执行对照验证，**Then** 记录为不可比较，不把差异归因于 semantic 实现。
3. **Given** server 运行在没有 Clang/clangd 的环境中，**When** 处理文档和 LSP 请求，**Then** 不启动外部编译器，不等待 Clang，不使用 Clang 作为 fallback。
4. **Given** Clang 返回一条 message，**When** server 发布 diagnostics，**Then** 发布 Go server 自己的稳定 code、severity、range 和 message，不直接转发原始 Clang message。

### Edge Cases

| 场景 | 预期行为 | 验证方式 |
| --- | --- | --- |
| 同名声明处于嵌套 namespace、block 或函数参数作用域 | 按作用域和可见性筛选；无法唯一确定时保留候选并报告 ambiguity | scope/name lookup fixture |
| 声明只有 prototype、没有定义 | 保留 declaration-only semantic fact；不伪造定义位置 | declaration-only fixture |
| 基础类型、指针或引用的层级不匹配 | 先检查标准隐式转换和可直接判断的常见指针/引用转换；没有可用转换序列时报告 `tops-semantic-type-mismatch` | scalar/pointer/reference fixture |
| 函数调用存在多个候选或需要完整 overload resolution | 不静默选候选；报告 `tops-semantic-ambiguous-name` 或降级状态 | overloaded-call fixture |
| overload-dependent conversion | `CallFact.candidate_status=ambiguous/unresolved`、`expected_status=degraded`；多个候选最多报告 `tops-semantic-ambiguous-name` Warning，不报告确定性 Error | overload-dependent fixture |
| host/device 属性缺失、冲突或只出现在未知条件分支 | `language=TOPS` 且 context resolved 时未标注函数按 host；其他情况不推断默认 host，目标检查降级 | call-boundary fixture |
| 地址空间规则只在某个 profile 或 pass 可用 | 只在对应 `SemanticContext` 生效；其他 profile 不继承该规则 | address-space profile fixture |
| 向量宽度、lane 数或对齐事实缺失 | 不使用相邻架构的数值；只报告 context 降级 | vector-context fixture |
| `#if` 分支为 inactive 或 unknown | inactive 分支不产生当前语义错误；unknown 分支不产生确定目标错误，并保留 provenance | condition fixture |
| 缺少 header 或 include root | 当前文档中可确认的 semantic facts 继续提供；外部依赖标为 unresolved/degraded | missing-header fixture |
| Tops-signaled document 没有编译命令或 target 缺失，且默认核心字段及所需 roots/macros 和已引用依赖完整 | 创建完整的 `arch=gcu400` 默认编译命令，返回 `profile_id=gcu400-default`，SemanticContext 标记为 `resolved` 并保留 fallback provenance；来源缺失时标记为 `partial` | default-profile fixture |
| 多个 target 冲突 | 不选择冲突的一方；使用 `gcu400-default` 作为结构化 fallback，SemanticContext 标记为 `invalid`，不发布冲突上下文下的确定性目标错误 | conflicting-target fixture |
| 普通 `.cpp`/`.cc` 没有编译命令 | 使用 `cpp-default`、`pass_kind=host`、host `availability=available`、device `availability=not-applicable`；所需普通 C++ 来源完整时整体为 `resolved`，来源缺失时为 `partial`，不套用 Tops/GCU400 目标规则 | non-tops-default fixture |
| 文档含 CRLF、UTF-8 多字节字符或零宽诊断 | diagnostics 使用统一的 0-based UTF-16 半开范围，范围不得越界 | UTF-16/range tests |
| 旧文档版本、context version、header version 或 analysis generation 晚到 | 丢弃旧结果，只记录 stale 观测，不发布旧诊断 | version-race integration test |
| 模板实例化、完整常量求值、访问控制、链接或代码生成影响结论 | 不把未实现能力报告为已验证错误；保留 unsupported/degraded 状态供后续规格使用 | out-of-scope fixture |

## Requirements *(mandatory)*

### Functional Requirements

#### Semantic Facts and Scope

- **FR-001**: Go server MUST 以当前 AST、symbol index、条件 provenance 和 `SemanticContext` 为输入生成 semantic snapshot；TypeScript client 不得复制 semantic 判断。
- **FR-002**: Semantic snapshot MUST 记录文档 URI、document version、semantic context version、相关 source/header version、analysis generation/epoch、完整或降级状态，并能被诊断和后续 LSP 查询引用。
- **FR-003**: 系统 MUST 识别已解析的 namespace、block、function、parameter、type、variable、field、声明和定义，并保留每个事实的名称、限定名、作用域、来源 URI、source range 和条件状态。
- **FR-004**: 名称查找 MUST 按当前作用域、限定名、条件状态和新鲜版本筛选候选；active 候选优先，inactive 候选不得作为当前结果返回，unknown 候选只能在没有 active 候选时作为降级结果保留。
- **FR-005**: 系统 MUST 支持基础标量、指针和引用的类型记录，至少区分类型名称、const/volatile 等已解析限定符、指针层级、引用类别和可用的地址空间信息；对标准标量隐式转换及可直接判断的常见指针/引用转换，MUST 记录转换类别；无法由当前语法和上下文确定的部分必须标为 unresolved。
- **FR-006**: 对唯一可确定的函数声明或定义，系统 MUST 记录参数顺序、参数类型摘要、返回类型摘要、执行空间属性和声明/定义关系；对参数个数或返回类型的确定性错误可以诊断，但不得依赖完整模板实例化或重载决议。
- **FR-007**: 对唯一可确定的函数调用，系统 MUST 检查标准标量隐式转换及可直接判断的常见指针/引用转换；用户自定义转换 MUST 记录 `conversion_status=unsupported`，不得产生 `tops-semantic-type-mismatch` 或 `tops-semantic-invalid-call` Error，并返回可解释的 unsupported 状态；overload-dependent conversion MUST 记录 `candidate_status=ambiguous` 或 `unresolved`，expected status 为 `degraded`，多个候选可发布 `tops-semantic-ambiguous-name` Warning；完整 overload-resolution fixture 的 expected status 为 `unsupported`；模板相关转换、存在多个候选或需要完整 overload resolution 时不得静默选一个。
- **FR-008**: 系统 MUST 只执行当前 `SemanticContext` profile 和 pass 视图提供的 host/device 调用边界，包括 host、device、`__host__ __device__` 和 kernel entry 的适用规则；当 `language=TOPS` 且 `SemanticContext=resolved` 时，未写执行空间属性的函数 MUST 按 host 参与检查；普通 C++ 的 host view 只表示标准 C++ host 分析，`default_execution_space=false`，不得推断 Tops default-host 或发布 Tops host/device Error。
- **FR-009**: 地址空间信息 MUST 随对象、指针和引用类型传播；系统只可按当前 `SemanticContext` profile/pass 规则判断兼容使用和转换，不得从其他 profile 推断规则。
- **FR-010**: 基础向量 semantic fact MUST 记录 element type、lane/width、alignment 和来源约束；系统只对当前 `SemanticContext` profile 明确提供的 vector constraint 进行确定性检查，不承诺完整 vector overload 或 code generation 语义。
- **FR-011**: 语义规则 MUST 遵守条件 provenance：inactive 分支不产生当前诊断，unknown 分支只能产生降级信息或非确定性结果，且结果必须保留条件来源。

#### SemanticContext and Degradation

- **FR-012**: `SemanticContext` MUST 至少包含 language standard、非空 `profile_id`、target profile 状态、`pass_kind`、宏 context、include roots、host/device rules、address-space rules、vector constraints、来源类型、provenance 标识和 semantic context version；`pass_kind` MUST 支持 `host`、`device` 和组合值 `host+device`。
- **FR-013**: `SemanticContext` MUST 区分 `resolved`、`partial`、`missing` 和 `invalid`；每个状态必须记录缺失字段或冲突来源。对 Tops 文档，profile resolver MUST 始终提供非空 `profile_id`：target 明确且无冲突时使用对应 profile，target 缺失时选择完整的 `gcu400-default` fallback，只有默认核心字段及所需来源完整时才标为 `resolved`，来源缺失时标为 `partial`，target 冲突时使用 `gcu400-default` 但标为 `invalid`；普通 C++ 文档即使 target 缺失也使用 `cpp-default`，未知 language 使用 `cpp-default` 并标为 `missing`。fallback provenance 必须保留，不得伪造真实 compile_commands 证据。
- **FR-014**: SemanticContext 来源优先级 MUST 按固定顺序处理：先读取 mode、`languageId` 和扩展名等文档级语言信号，再将 candidate 的 `source_uri` 用 `internal/index.NormalizeSourceURI` 规范化，只保留与当前文档规范化 URI 相同的 candidate，其他 URI 及其 candidate language 不参与当前文档的计数或冲突；过滤不可解析 candidate，对等价 candidate 去重并保留全部 provenance；只对当前文档候选评估明确 `language`，candidate language 优先于文档级信号，同一当前文档候选的明确 language 不同属于语言字段冲突，保留候选并标记 `invalid`，不得静默过滤或合成默认命令。candidate 数量为 0 时才允许 Tops 使用 `gcu400-default` 默认字段，数量为 1 时选择该 candidate，数量大于 1 且非等价时始终为 `invalid`，不得回退默认命令；低优先级语言信号不会覆盖或制造冲突。字段缺失时只允许 `workspace fallback field` 补充缺失的工程 header roots/macros；完整 `workspace candidate` 仍参与 candidate 等价和字段冲突判断，compile command 中明确提供的字段优先，字段冲突不得被 fallback 覆盖。Clang/clangd 不得出现在运行时 fallback 链路中。
- **FR-015**: `resolved` SemanticContext 可以执行 profile 提供的所有本功能范围内检查，包括由 `gcu400-default` fallback 提供的 GCU400 规则；`partial` 只能执行事实完整的规则；`missing` 或 `invalid` 只能提供不依赖缺失或冲突事实的声明、作用域和基础类型结果，并发布稳定的 context 状态。
- **FR-016**: 当 SemanticContext 处于 `partial`、`missing` 或 `invalid`，或适用的 pass/profile 规则缺失时，系统 MUST 抑制相应的确定性目标错误，并以 `tops-semantic-context-missing` 或 `tops-semantic-context-degraded` 表示原因；`resolved` 的 `gcu400-default` fallback 可以按 GCU400 规则产生目标诊断。

### SemanticContext 状态判定顺序

状态枚举的展示顺序不表示优先级；实际判定优先级固定为 `invalid` > `missing` > `partial` > `resolved`，前一条命中后不再被后一条覆盖。

状态按以下顺序判定，前一条命中后不再被后一条覆盖：

1. target 或参数来源冲突时为 `invalid`，即使默认 profile 或部分来源可用，也不得选择冲突的一方。
2. 无法形成语言或命令上下文时为 `missing`。
3. 默认核心字段可以形成，但工具链默认 roots/macros 或已引用的 workspace 依赖缺失时为 `partial`。
4. 真实命令完整，或默认命令核心字段、工具链 roots/macros 和已引用 workspace 依赖均可用时为 `resolved`。

未被当前文档引用的 workspace header/macro 缺失不改变状态；只有被引用的缺失来源才进入第 3 条。

### SemanticContext 字段来源规则表

| 字段类别 | compile command 或完整 workspace candidate 明确提供 | workspace fallback field | 无有效 Tops command 时的默认值 | 冲突行为 |
| --- | --- | --- | --- | --- |
| target/arch | 最高优先级 | 不适用 | `gcu400` | `invalid`，不选择一方 |
| driver/language/standard/pass | 最高优先级 | 不适用 | `topscc`、`TOPS`、`c++17`、`host+device` | `invalid` |
| toolchain roots/macros | 使用命令指定或工具链默认来源 | 不适用 | topscc 默认 roots/macros | 缺失时 `partial` |
| workspace roots/macros | 使用命令指定或 workspace 来源 | 只补缺失项 | 无默认工程来源 | 被引用来源缺失时 `partial` |

完整 `workspace candidate` 的明确字段参与 candidate 等价和冲突判断；`workspace fallback field` 只允许补充 workspace roots/macros，不能提供 target、arch、driver、language、standard、pass 或 toolchain roots/macros。

同一字段出现多个不同明确值时，冲突优先于所有补全规则。

#### Diagnostic Contract

- **FR-017**: Semantic diagnostic code MUST 使用 `tops-semantic-` 前缀；code、severity、主范围规则和降级行为构成稳定契约，message 文本可以改进但不得替代 code。
- **FR-018**: v1 semantic diagnostic code MUST 至少包含以下集合：`tops-semantic-unknown-name`、`tops-semantic-ambiguous-name`、`tops-semantic-type-mismatch`、`tops-semantic-invalid-call`、`tops-semantic-host-device-boundary`、`tops-semantic-address-space`、`tops-semantic-vector-constraint`、`tops-semantic-context-missing`、`tops-semantic-context-degraded`、`tops-semantic-incomplete-input`。
- **FR-019**: 确定性错误的 severity MUST 为 `Error`；候选歧义 MUST 为 `Warning`；context 缺失或降级 MUST 为 `Information`；可恢复的不完整输入 MUST 为 `Hint`。未达到确定性条件的规则不得使用 `Error`。
- **FR-020**: 每条 semantic diagnostic MUST 包含 code、severity、server-owned message、source、主 range、document version、semantic context version、context status、适用的 `pass_kind` 和必要的 related information；pass-sensitive diagnostic 的 `pass_kind` 必须是 `host` 或 `device`，pass-neutral 的 context/incomplete diagnostic 使用 `not-applicable`；内部字段 MUST 按契约映射为 JSON `documentVersion`、`contextVersion`、`sourceVersion`、`contextStatus`、`profileId`、`passKind` 和 `semanticKind`；generation/epoch 只进入 stale 日志，不进入 LSP data；不得把完整源文本、凭据或原始 Clang message 放入诊断。
- **FR-021**: 主 range MUST 指向最小可操作的源范围：未知名称指向名称 token，调用边界指向调用表达式或被调用名称，类型/地址空间/向量错误指向冲突的类型或属性 token，context 状态指向第一个受影响构造或文档起始位置，不得使用虚拟展开文本位置。
- **FR-022**: `tops-semantic-host-device-boundary`、`tops-semantic-address-space` 和 `tops-semantic-vector-constraint` 只有在对应源码构造存在可验证非空 range 时才能发布 `Error`；如果无法取得该 range，系统 MUST 抑制确定性 Error，并改为 context degraded 信息或不发布。
- **FR-023**: 所有发布给 LSP 的 range MUST 使用 0-based UTF-16 code unit 和半开区间；CRLF、UTF-8 多字节字符、EOF 和零宽范围必须通过位置转换验证。
- **FR-024**: 同一 semantic snapshot 中相同 code、主范围、候选来源、条件状态和 `pass_kind` 的重复诊断 MUST 去重；host/device 两个 pass 的诊断不得互相去重，诊断顺序 MUST 稳定，便于增量编辑和离线对照。
- **FR-025**: v1 semantic baseline MUST 只通过标准 `textDocument/publishDiagnostics` 发布用户可见结果；semantic facts 仅供现有 server 能力和后续语义能力内部消费，不新增 hover、completion 或私有 semantic LSP。没有当前新鲜结果时不得发布旧诊断。

#### Version, Observability and Offline Comparison

- **FR-026**: Semantic result 只有在 document version、semantic context version、相关 source/header version、analysis generation 和 epoch 与当前 server 状态一致时才可提交或发布。
- **FR-027**: 任何版本不匹配、文档关闭、context 失效、header 失效或请求取消 MUST 丢弃 semantic snapshot 和 diagnostics；旧结果只允许产生 `stale_result` 观测，不得覆盖新版本或发布 stale diagnostic。
- **FR-028**: stale 观测 MUST 至少记录 operation、旧/新 document version、旧/新 context version、相关 source/header version、generation/epoch 和 reason；日志不得记录完整源代码。
- **FR-029**: Clang 对照验证 MUST 使用与 Go semantic 完全相同的源文本、工作目录、原始/归一化参数、target、pass、include roots、宏、compiler/header provenance 和输入版本；任一关键字段不同都必须标记为不可比较。
- **FR-030**: Clang 只能作为开发机或测试流程中的离线对照编译器；Go server runtime 不得启动 Clang/clangd、链接其库、等待其输出或把其 message 直接转发给用户。
- **FR-031**: 测试 MUST 覆盖正常、错误、不完整、missing/partial SemanticContext、条件分支、header 缺失、Unicode/CRLF、取消和版本竞争；测试必须能在没有 GCU 设备、GCUSIM、Clang runtime 或 clangd server 的环境中运行。
- **FR-032**: 依赖模板实例化、完整 overload resolution、完整常量求值、完整访问控制、链接或代码生成的结论 MUST 保持 out-of-scope、unsupported 或 degraded，不得计入本功能的 semantic correctness 通过条件。
- **FR-033**: profile resolver MUST NOT 返回 nil、空字符串或无 profile 的 semantic snapshot；`gcu400-default` MUST 始终存在，并明确标记为 GCU400 fallback provenance。对 Tops 文档没有 target 时可以使用它生成 `resolved` SemanticContext；存在 target 冲突时不得把 `invalid` SemanticContext 变成 `resolved`。普通 C++ 文档不得因 target 缺失选择 `gcu400-default`，而应使用 `cpp-default`。
- **FR-034**: 实现 MUST 删除 profile resolver 之后针对 nil、空字符串或无 profile snapshot 的降级处理路径；profile 缺失只能在 resolver 边界被替换为适用的默认 profile，Tops 使用 `gcu400-default`，普通或未知 language 使用 `cpp-default`，不得继续生成 no-profile semantic snapshot、no-profile diagnostic 或依赖“无 profile”状态的分支。
- **FR-035**: 当 Tops-signaled document 的输入编译命令缺失，或当前文档规范化 URI 对应的 candidate 解析后没有任何可用 candidate 时，resolver MUST 创建一条默认编译命令记录，设置 `arch=gcu400`、`driver=topscc`、`language=TOPS`、`language_standard=c++17` 和 `pass_kind=host+device`，并以该记录生成 `profile_id=gcu400-default`；无关 URI 的 candidate 不计入当前文档。存在两个或以上非等价 command candidate，或同一当前文档候选的明确 language 冲突时，不得合成默认命令，必须保留 candidate 并将整体状态设为 `invalid`。不得返回空 command 或 no-profile semantic snapshot。标准库和 `topscc` Tops headers 使用工具链默认 include roots，工程 include roots 使用 workspace 来源，工具链内建宏和 workspace 工程宏分别加载，其他未核验参数必须保持未解析或降级，不得静默填充。
- **FR-036**: `pass_kind=host+device` MUST 表示同一默认命令包含 host 和 device 两个分析视图；SemanticContext MUST 分别保留两侧的执行空间事实，semantic checker 不得将组合 pass 当作单一执行空间，也不得在缺少对应 pass provenance 时发布目标特定 Error。
- **FR-037**: 本规格将 `.tops`、`languageId=tops` 或明确 Tops 模式命中的文档称为 Tops-signaled document。语言判定 MUST 按 compile command 明确 `language`、明确 Tops/C++ 模式、`languageId`、文件扩展名的顺序处理；`language=TOPS` 的默认命令只能用于 Tops-signaled document，`.tops` 遇到明确普通 C++ compile command 时必须使用 `cpp-default`；普通 `.cpp`、`.cc` 或其他非 Tops 文档 MUST NOT 被静默转换为 TOPS 或接收 Tops/GCU400 目标规则。
- **FR-038**: 默认命令的 include 和 macro provenance MUST 区分工具链默认标准库/Tops header roots、工具链内建宏、workspace 工程 header roots 和 workspace 工程宏；workspace 工程 roots/macros 不得替代工具链来源，缺失的任一来源必须标记 unresolved/degraded，不得扫描整个 workspace 或创建虚假 header/macro。
- **FR-039**: 对 `pass_kind=host+device`，semantic checker MUST 分别计算 host 和 device 视图；同一位置只有违规 pass 发布对应诊断，合法 pass 不发布，且每条 pass-sensitive JSON diagnostic 的 `Diagnostic.data.passKind` MUST 标明 `host` 或 `device`。不依赖具体 pass 的 `tops-semantic-context-missing`、`tops-semantic-context-degraded` 和 `tops-semantic-incomplete-input` 使用 `passKind=not-applicable`。
- **FR-040**: `semanticKind` 的 JSON 值 MUST 只来自 v1 diagnostic code 去掉 `tops-semantic-` 前缀后的固定集合；`sourceVersion` MUST 是按 URI 排序后的 source/header version aggregate fingerprint，generation/epoch 不进入 JSON。
- **FR-041**: profile resolver MUST 返回非指针的 `SemanticContext` value；普通 C++ 使用目标中立的 `cpp-default`，空 profile 输入必须在 resolver 边界转换为适用的默认 profile，但该替换不触发默认 `CompilationCommand` 合成，也不得覆盖已有命令的字段来源。`SemanticSnapshot.profile_id` 的非空不变量、Tops fallback、fallback provenance、target conflict 和 no-profile 边界由 FR-033/FR-034 统一定义，后续 semantic API 不得接收或分支处理 no-profile 状态。
- **FR-042**: `CompilationCommand` MUST 记录整体 `source` 摘要和每个字段的 `field_source`、`field_conflict`、`field_resolution`；整体 `source` 允许为 `compile_commands`、`workspace` 或 `default`，其中 `default` 表示合成默认命令，具体字段来源以 `field_source` 为准；`field_source` 允许为 `compile_commands`、`workspace`、`toolchain_default`、`fallback_default`、`unresolved` 或 `invalid`，`field_resolution` 允许为 `resolved`、`unresolved` 或 `invalid`；无冲突时 `field_conflict=null`，冲突时必须保留按固定来源顺序排列的 `{sources,values}`；任意字段冲突都使整体状态为 `invalid`，不得只记录一个整体 command source 来掩盖字段混合来源。
- **FR-043**: `sourceVersion` MUST 使用导出的 `internal/index.NormalizeSourceURI` 作为唯一 URI 规范化函数和唯一共享 aggregate helper；root source 固定为当前 `document_uri` 的规范化 URI，先按规范化 URI 去重，相同版本只保留一条，不同版本写入 `status:conflict`，再按 URI UTF-8 字节序排序 root source 和依赖 source/header；每项序列化为 `uri\0version:<value>\n` 或 `uri\0status:<value>\n`，对 UTF-8 bytes 计算 SHA-256，并输出 `sha256:<hex>`。semantic diagnostic 必须包含 root source；root source 缺失或 `document_uri` 无法规范化时不得发布该 diagnostic，comparison record 必须标记为 `not-comparable`；缺失依赖使用 `missing`、`unresolved`、`unreadable` 或 `stale` status token，generation/epoch 不参与计算。
- **FR-044**: semantic fixture manifest MUST 固定 `capability`、`scenario`、条件适用的 `callable_kind`/`caller_kind`/`callee_kind`/`pass_kind`、条件适用的 `default_execution_space`、`source_files`、`expected_status`、`expected_diagnostics`、`profile_id` 和 provenance 字段；普通 capability 的 `scenario` 只能是 `valid`、`invalid`、`incomplete` 或 `context-boundary`，对应的 `expected_status` 分别只能是 `complete`、`invalid`、`degraded`、`degraded`；`capability=out-of-scope` 只能使用 `scenario=unsupported`，`expected_status` 只能是 `unsupported` 或 `degraded`，并填写六类 `out_of_scope_kind`；不适用的字段必须写 `not-applicable`。`callables` capability 必须提供布尔 `default_execution_space`，只有明确登记的 resolved Tops `default-host` 子场景为 `true`，普通 C++ 和其他 callable 场景为 `false`；其他 capability 写 `not-applicable`。`expected_diagnostics` 的 `status` 只能是 `published` 或 `suppressed`，`published` 必须有 code/severity/passKind/range，pass-sensitive diagnostic 的 passKind 为 `host`/`device`，pass-neutral diagnostic 的 passKind 为 `not-applicable`，`suppressed` 必须有 code/reason；validator MUST 校验缺字段、非法枚举、capability/scenario/expected_status/out_of_scope_kind/default_execution_space 的非法组合、诊断 code/severity/passKind/range，并输出各 capability/scenario 的总数、通过数、失败数和失败 case ID，存在失败时返回非零退出码。

### Stable Diagnostic Table

| Code | Severity | 确定性条件 | 主范围 | 降级规则 |
| --- | --- | --- | --- | --- |
| `tops-semantic-unknown-name` | Error | resolved context 下当前作用域没有可见声明 | 名称 token | 缺少 header 或条件未知时改为 context degraded，不报确定错误 |
| `tops-semantic-ambiguous-name` | Warning | 存在多个候选且本功能不能安全消除歧义 | 名称或调用名称 | 不进行完整 overload resolution，保留候选来源 |
| `tops-semantic-type-mismatch` | Error | 基础标量/指针/引用关系确定不兼容 | 冲突类型 token 或操作符 | 复杂隐式转换、模板或未知类型时降级 |
| `tops-semantic-invalid-call` | Error | 唯一 callable 候选确定不可调用或参数基础关系确定非法 | 调用表达式 | 多候选或缺少函数声明时降级 |
| `tops-semantic-host-device-boundary` | Error | host/device 边界及调用方向均有核验 provenance | 被调用名称或调用表达式 | 缺失 pass/属性/目标规则时不报 Error |
| `tops-semantic-address-space` | Error | 地址空间和转换规则均已核验且确定不兼容 | 指针、引用或转换 token | 规则缺失时只报 context code |
| `tops-semantic-vector-constraint` | Error | element/lane/width/alignment 约束已核验且确定违反 | vector 类型或属性 token | 不继承相邻 profile 的约束 |
| `tops-semantic-context-missing` | Information | 没有可用的 SemanticContext 字段支撑目标检查 | 受影响构造或文档起始位置 | 继续提供不依赖目标的基础结果 |
| `tops-semantic-context-degraded` | Information | SemanticContext 部分可用，至少一类规则被抑制 | 第一个受影响构造 | 同一版本相同原因的 context 信息去重 |
| `tops-semantic-incomplete-input` | Hint | AST/parser 标记为可恢复且输入尚未闭合 | EOF、缺失 token 或恢复节点 | 不把恢复状态升级为语义 Error |

## Scope and Boundaries

### In Scope

- 已解析声明、声明/定义关系、namespace/block/function/parameter/type/variable/field 作用域和名称查找。
- 基础标量、指针、引用、函数参数和返回值的类型表示与确定性检查，包括标准标量隐式转换及可直接判断的常见指针/引用转换。
- 有 provenance 的 host/device 调用边界、地址空间兼容性和基础向量 element/lane/width/alignment 约束。
- `SemanticContext` 的来源、状态、降级规则、版本和 provenance 传播。
- 稳定 semantic diagnostic code、severity、主范围、related information、UTF-16 位置和标准 LSP 发布。
- v1 只对外提供标准 diagnostics；semantic facts 不通过新增的 hover、completion 或私有 semantic LSP 暴露。
- 文档、context、header 变化后的 semantic snapshot 更新和旧结果丢弃。
- 相同输入和 provenance 的离线 Clang 对照验证记录。

### Out of Scope

- 模板实例化、完整重载决议、完整常量求值、完整访问控制、链接和代码生成。
- 依赖未核验 target、宏、header 或 pass 的语义推断。
- 将 Clang/clangd 作为 Go server runtime、sidecar、fallback 或诊断消息来源。
- 把 Clang 原始 message、虚拟展开位置或 compiler-private code 直接暴露给用户。
- 设备执行、性能、DTE 时序、GEMM 结果和真实硬件行为验证。
- TypeScript client 中的语义解析、候选选择或诊断重写。
- 本期新增的 semantic hover、completion 或私有 semantic LSP 请求。
- 用户自定义转换，以及依赖模板实例化或完整 overload resolution 的转换序列。

## Key Entities *(include if feature involves data)*

- **SemanticContext**：semantic 分析使用的文件级上下文快照，包含 target/pass、宏、include roots、规则能力、来源、provenance、状态和版本。
- **CompilationCommand**：用于生成 SemanticContext 的原始或默认命令记录；默认记录至少包含 `arch=gcu400`、`driver=topscc`、`language=TOPS`、`language_standard=c++17`、`pass_kind=host+device`、工具链默认标准库/Tops header roots、工具链内建宏、workspace 工程 header roots、workspace 工程宏、fallback provenance 和未解析字段状态。
- **SemanticProfile**：供 SemanticContext 引用的语义 profile，包含非空 ID、target 身份、规则能力、provenance 和 fallback 标记；Tops 无明确 target 时引用 `gcu400-default`，普通或未解析为 Tops 的 C++ 文档引用 `cpp-default`。
- **DeclarationFact**：从 AST/index 得到的声明语义事实，包含名称、限定名、kind、作用域、声明/定义位置、条件状态和来源版本。
- **ScopeFact**：namespace、block、function、class 或 translation unit 的可见性边界，记录父子关系、声明集合和 source range。
- **SemanticType**：基础标量、指针或引用的类型表示，包含限定符、指针层级、pointee/reference 类型、地址空间、可用标准转换类别和 unresolved 部分。
- **CallableFact**：函数参数、返回值、执行空间、属性、声明/定义关系和可调用边界。
- **CallFact**：调用点的表达式范围、callee spelling、candidate symbol IDs、candidate 状态、caller/callee pass view 和每个参数的 conversion status。
- **VectorConstraint**：由当前 target/pass provenance 提供的 element type、lane/width、alignment 和允许操作约束。
- **SemanticDiagnostic**：稳定 code、severity、message、主 range、related information、context status 和版本字段组成的用户可见诊断。
- **SemanticSnapshot**：一次可查询的 semantic facts、CallFact 和 diagnostics 集合，直接保存非空 `profile_id`、`context_status`、`source_version_fingerprint`，并绑定 document/context/source/header 版本以及 `complete`、`degraded`、`stale` 状态；`stale` 只能表示失效过程，不能被查询或发布。
- **ComparisonRecord**：离线对照验证记录，包含双方输入 digest、参数/provenance、版本、归一化类别、位置和差异原因，不包含运行时 Clang message 转发内容。

## Semantic Evaluation Contract

### SemanticContext 状态

| 状态 | Profile 选择 | 允许执行的检查 | 必须抑制的结论 | 对外状态 |
| --- | --- | --- | --- | --- |
| `resolved` | 已核验的具体 profile | 所有已核验的基础声明、类型、调用边界、地址空间和向量规则 | 仍超出本规格范围的能力 | 正常 semantic diagnostics |
| `partial` | target 已明确时保留具体 profile；Tops target 未明确时使用 `gcu400-default`，普通或未解析为 Tops 的 C++ 使用 `cpp-default` | 已具备完整事实的标准和 Tops 规则 | 依赖缺失字段的 target-specific Error | `tops-semantic-context-degraded` |
| `missing` | Tops 使用 `gcu400-default`，普通或未解析为 Tops 的 C++ 使用 `cpp-default` | 已知语言和 AST/index 支撑的声明、作用域、基础类型 | 所有需要 target/pass/provenance 的确定结论 | `tops-semantic-context-missing` |
| `invalid` | Tops 使用 `gcu400-default`，普通 C++ 使用 `cpp-default` | 可独立确认的局部事实和错误来源记录 | 冲突上下文下的目标语义 Error，禁止选择一侧 | context 信息和 degraded snapshot |

### Default Profile

- canonical fallback profile ID 为 `gcu400-default`，对应 GCU400 的 profile 定义，但 fallback 标记表示当前 target provenance 尚未核验。
- 没有有效编译命令时，profile resolver 先创建包含 GCU400/Tops 默认字段、工具链标准库/Tops header roots、工具链内建宏和 workspace 工程 header roots/macros 的 `CompilationCommand`，再生成 `gcu400-default`；默认命令不是对外执行的 shell fallback。
- `gcu400-default` 保证 profile 选择不为空；完整的合成默认命令可以生成 `resolved` SemanticContext，但 fallback provenance 必须保留，不能伪装成真实 GCU400 compile_commands 证据。
- SemanticContext 状态仍控制目标特定规则：`resolved` 的 `gcu400-default` 可以按 GCU400 profile 执行 host/device、地址空间和向量规则并产生确定性 `Error`；`partial`、`missing` 或 `invalid` 仍只能执行事实完整的受限规则。
- 收到有效 target context 后，server 创建对应具体 profile 和更高版本的 SemanticContext；旧的 `gcu400-default` semantic snapshot 按版本规则失效。

### 查找和候选规则

- 查询先按当前 URI、document/context/header version 和条件状态筛选，再按作用域和限定名匹配。
- active 候选优先；一个 active 候选可以执行本功能范围内的基础检查。
- 多个 active 候选必须保留候选集合并报告 ambiguity；不得借用完整 overload resolution 的结论。
- 没有 active 候选时，unknown 候选只能作为降级事实；inactive 候选不属于当前语义结果。
- 不完整节点可以贡献已经确认的声明和类型 token，但不得制造虚拟定义、虚拟参数或虚拟 target 规则。

### 结果提交规则

- semantic worker 开始时固定 `document version + SemanticContext version + dependency versions + generation/epoch`。
- 计算完成后重新检查所有版本和取消状态；任一不匹配都丢弃结果。
- 当前版本没有新鲜 semantic snapshot 时，诊断不发布；依赖 semantic 查询的 LSP 请求使用标准空/`null`/取消行为，不返回旧位置。
- context 状态、diagnostic code 和版本可以记录在协议允许的 `Diagnostic.data` 中，但 TypeScript client 不得据此重做语义判断。

## Offline Clang Comparison Contract

- 对照输入必须使用同一源文本字节、工作目录、原始 argv、归一化 argv、target/profile、host/device pass、include roots、宏集合、活动 header 版本和 provenance 标识。
- 对照结果只比较归一化的类别、确定性、主范围、相关来源和版本；Go server 自己生成 `tops-semantic-*` code、severity 和 message。
- 对照运行只属于开发机或测试流程；server 进程不执行外部命令，不依赖 clangd，不把 Clang 结果缓存为 runtime semantic source。
- 发现输入不一致、工具版本不一致或 active header 不一致时，ComparisonRecord 必须标记不可比较，并记录具体缺失或冲突字段。

## Test Plan

### Semantic Unit Tests

- 声明、定义、namespace/block/function scope、参数和返回值的正向、负向、不完整和 shadowing 场景。
- 基础标量、指针、引用、指针层级、限定符和唯一函数调用的确定性类型场景。
- 标准标量隐式转换、可直接判断的常见指针/引用转换、禁止的转换、用户自定义转换和依赖 overload resolution 的转换场景。
- host/device 合法调用、非法调用、属性缺失、属性冲突、unknown condition 和多候选调用。
- `language=TOPS` 下未标注函数的默认 host、非 Tops/不完整 context 下的降级和显式属性覆盖场景。
- address-space 兼容性、明确非法转换、缺少规则和 profile 切换场景。
- vector element/lane/width/alignment 的确定性限制、缺失约束和相邻 profile 隔离场景。
- `SemanticContext` 的 resolved/partial/missing/invalid 规则，禁止隐藏默认值和降级 code 去重。
- `gcu400-default` 的 fallback 创建、profile 非空约束，以及 resolver 之后 no-profile snapshot/diagnostic 分支为零的回归场景。
- 没有有效编译命令时默认 `CompilationCommand` 的创建、GCU400/Tops 默认字段和未核验 include/宏字段保持未解析的回归场景。
- 默认 `pass_kind=host+device` 的双视图创建、host/device 规则隔离和组合 pass 不产生误报的回归场景。
- 同一源码位置在 host/device 视图产生不同结果时的按 pass 诊断、JSON `Diagnostic.data.passKind` 和去重隔离场景。
- 工具链默认标准库/Tops headers 与 workspace 工程 headers 的 include provenance、缺失来源和不扫描 workspace 的回归场景。
- 工具链内建宏与 workspace 工程宏的 provenance、条件分支状态和缺失宏降级的回归场景。
- Tops 与普通 C++ 文件的默认命令隔离，验证普通 `.cpp`/`.cc` 不会获得 `language=TOPS` 或 Tops/GCU400 规则。

### Diagnostic and LSP Tests

- 每个稳定 code 的 severity、主 range、related information、message 来源、`Diagnostic.data` 版本和条件状态。
- UTF-16、CRLF、多字节字符、EOF、零宽范围、重复诊断和稳定排序。
- 当前版本诊断发布、文档变化、context/header 变化、取消、stale-result 丢弃和无旧结果覆盖。
- 标准 `textDocument/publishDiagnostics` 契约；TypeScript client 不需要私有 semantic message。

### Offline Comparison Tests

- 相同源文本和 provenance 的可比较记录。
- 缺少 target、pass、include、宏、header 或版本的不可比较记录。
- server 在没有 Clang/clangd 可执行文件时运行全部 semantic unit tests，并验证没有外部编译器调用。
- 比较差异只进入 ComparisonRecord 和测试报告，不进入用户可见的 Clang 原始 message。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: semantic fixture manifest 显式覆盖声明/作用域/名称查找、基础标量/指针/引用、函数参数/返回值、host/device、地址空间和向量约束六类能力；每类都必须登记 valid、invalid、incomplete 和 SemanticContext boundary 四种场景。
- **SC-002**: 每个 v1 `tops-semantic-*` code 都有自动化断言，覆盖 code、severity、主 range、降级规则和 UTF-16 编码；未定义 code 不得出现在用户可见 semantic diagnostics 中。
- **SC-003**: 在 `resolved` SemanticContext 下，所有确定性 fixture 的预期错误和无错误结果 100% 匹配；在 `partial`/`missing`/`invalid` SemanticContext 下，目标依赖规则不得产生未经证明的 Error。
- **SC-004**: 连续 document/context/header version 测试中，旧 semantic snapshot、旧诊断和旧 LSP 结果覆盖当前版本的次数为 0；stale 观测包含旧/新版本和丢弃原因。
- **SC-005**: 所有 diagnostic range 在 UTF-16、CRLF、Unicode、多行和零宽边界测试中均可被标准 LSP 客户端解析，不出现越界或负位置。
- **SC-006**: 相同输入和 provenance 的离线 Clang 对照记录可重复生成；fixture manifest 和 comparison manifest 的用途分开，输入不一致时 100% 被标记为不可比较，且用户可见结果不包含原始 Clang message。
- **SC-007**: semantic unit、diagnostic、version-race 和 LSP contract tests 在没有 GCU 设备、GCUSIM、Clang runtime 或 clangd server 的开发机环境中可独立运行。
- **SC-008**: `capability=out-of-scope` 的模板实例化、完整 overload resolution、完整常量求值、完整访问控制、链接和代码生成各至少有一个 fixture；这些 fixture 的 `expected_status` 必须为 `unsupported` 或 `degraded`，不得标记为本功能已完成。
- **SC-009**: v1 semantic baseline 的用户可见输出只包含标准 diagnostics；测试不得要求新增 hover、completion 或私有 semantic LSP 请求才能通过。
- **SC-010**: 标量、指针和引用 fixture 必须分别覆盖标准隐式转换成功、无可用转换失败、用户自定义转换 `unsupported` 且不产生 type-mismatch/invalid-call Error、以及 overload-dependent 转换降级；每种结果都必须匹配预期 code 或状态。
- **SC-011**: callable manifest 必须登记 host、device、`__host__ __device__` 和 kernel entry 四类，每类各有 valid、invalid、incomplete 和 SemanticContext boundary 一项，共 16 个明确预期；未标注 Tops 函数的 default-host 是 host 类的子场景，不增加第五类；host/device 规则还必须验证单侧缺失 provenance 的状态。
- **SC-012**: 每个 semantic snapshot 和 `SemanticContext` 都必须包含非空 `profile_id`；Tops 文档在 target 缺失且默认核心字段及所需 toolchain/workspace 来源完整时，100% 通过完整默认命令使用 `gcu400-default` 并保持 `resolved`，来源缺失时保持 `partial`；Tops target 冲突时使用 `gcu400-default` 并保持 `invalid`，普通 C++ 文档无论是否有 target 都使用 `cpp-default`。
- **SC-013**: profile resolver 的回归测试中，Tops 文档的 nil、空字符串或无 profile 输入 100% 在 resolver 边界转换为 `gcu400-default`，普通或未解析为 Tops 的 C++ 文档转换为 `cpp-default`；resolver 输出类型和 semantic snapshot 都不得表示 no-profile 状态，后续代码路径不得读取或处理 no-profile 分支。
- **SC-014**: Tops fixture 的输入编译命令缺失或命令解析后没有可用 candidate 时，resolver 100% 创建默认 `CompilationCommand` 并设置 `arch=gcu400`、`driver=topscc`、`language=TOPS`、`language_standard=c++17` 和 `pass_kind=host+device`；两个或以上非等价 command candidate 必须稳定地产生相同的冲突记录和 `invalid` 状态，不得回退到默认命令；输入顺序置换不得改变结果；未核验的 include 和宏字段不得被填成事实。
- **SC-015**: `pass_kind=host+device` 的测试必须分别验证 host/device semantic facts 和调用边界；组合 pass 不得产生无法归属到具体 pass 的目标特定 Error。
- **SC-016**: 普通 `.cpp`/`.cc` 没有编译命令时，SemanticContext 使用 `cpp-default`，`language=TOPS` 的错误应用次数为 0，Tops/GCU400 目标诊断数量为 0。
- **SC-017**: 默认 Tops 命令的 include roots 测试必须证明标准库/Tops headers 来自工具链默认路径、工程 headers 来自 workspace 来源；fake source provider 的 `workspace.enumerate` 调用次数必须为 0，引用工程 header 的唯一 URI 读取次数必须等于唯一引用 URI 数量，重复引用必须命中缓存，缺失来源不产生虚假 header。
- **SC-018**: 默认 Tops 命令的宏测试必须证明工具链内建宏和 workspace 工程宏分别进入 `SemanticContext`；宏来源不明时条件状态为 `unknown`，不得伪造 `active` 或 `inactive`。
- **SC-019**: `pass_kind=host+device` 的诊断测试中，host/device 结果分别计算；只有违规 pass 发布诊断，且每条 JSON diagnostic 都带正确的 `Diagnostic.data.passKind`，两侧重复诊断不得被错误合并。
- **SC-020**: 每个 pass view 都必须记录 `status`、`rules_available` 和 `provenance`；host-only、device-only、两侧均可用和单侧缺失四种组合都必须有测试预期。

## Assumptions

- 现有 parser/index 能提供可恢复 AST、声明/引用、作用域、条件 provenance、source range 和 document/context version；semantic 层消费这些结果，不重新定义词法位置规则。
- `SemanticContext` 是 semantic 层的新增输入对象，由可审计的编译上下文和 workspace fallback 组合而成；没有有效编译命令时先创建包含 GCU400/Tops 默认字段和 `pass_kind=host+device` 的命令，再使用 `gcu400-default`，允许按 GCU400 规则检查但必须保留 fallback provenance。
- 地址空间名称、向量宽度和 host/device 规则只有在当前工具链、活动 header、测试或其他明确 provenance 核验后才进入 `SemanticContext`。
- v1 支持标准标量隐式转换及可直接判断的常见指针/引用转换；用户自定义转换、模板相关转换和依赖完整 overload resolution 的转换留给后续能力。
- 在 `language=TOPS` 且 context resolved 时，未标注执行空间的函数按 host 处理；其他语言或不完整 context 不使用该默认值。
- 标准 LSP diagnostics 和现有版本/stale 观测机制是对外边界；新增 semantic 字段必须保持向后兼容。
- Clang 对照编译器只在开发机离线验证中可用，不能成为用户运行语言服务器的前置依赖。
- 本功能的测试不需要真实 GCU、GCUSIM、设备 workload 或代码生成产物。
