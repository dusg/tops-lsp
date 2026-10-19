# Tasks: Tops C++ 通用 Semantic Baseline

**Input**: Design documents from `specs/006-semantic-baseline/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: 所有 Go server、semantic、LSP diagnostics、配置和版本行为变更都包含测试任务；Clang 只在离线对照任务中使用。

**Organization**: 任务按 User Story 分组；每个 story 在完成后都有独立测试条件。

## Phase 1: Setup

**Purpose**: 建立 semantic 包、测试材料和 manifest 的最小结构。

- [X] T001 [P] 创建 `internal/semantic/doc.go` 和 semantic 包目录，保持 Go module 无新增第三方依赖
- [X] T002 [P] 创建 `testdata/semantic/` 的 `manifest.json`、`comparison_manifest.json`、`expected/`、`declarations/`、`types/`、`callables/`、`calls/basic/`、`calls/host-device/`、`address-space/`、`vectors/`、`out-of-scope/`、`default-context/`、`conditions/`、`incomplete/` 和 `stale/` 目录
- [X] T003 [P] 创建并登记 `testdata/semantic/manifest.json` 的 fixture case rows（允许先引用由后续 fixture 任务创建的路径）；Phase 1 只校验 manifest schema、case/condition 引用和枚举，不因后续 fixture 文件尚未创建而失败，文件存在性和可读性由 T064 在所有 fixture 任务完成后校验。创建 `cmd/tops-semantic-check/main.go` 校验入口和 `cmd/tops-semantic-check/main_test.go` 成功/失败退出测试，在 `internal/semantic/fixture_test.go` 校验唯一 case_id、condition_id/parent_conditions 循环和悬空引用、condition 的 `macros`/byte-offset 半开 `expression_range`/`expected_state`、`out_of_scope_kind`、`default_execution_space` 适用性和布尔值、非法枚举、非法 capability/scenario/expected_status/out_of_scope_kind/default_execution_space 组合、诊断 code/severity/passKind/range/source_uri；manifest 必须登记 declarations/types/callables/calls/address-space/vectors 六类 capability 的四种 scenario、六类 out-of-scope kind，以及 host/device/host-device/kernel-entry × valid/invalid/incomplete/context-boundary 的 16 个 callable case；负例必须覆盖缺字段、普通 C++ `default_execution_space=true`、非布尔值、非 default-host Tops callable 为 true、`published` 缺必需字段和 `suppressed` 缺 reason；普通六类 capability 各覆盖四种 scenario 及对应 expected_status，`out-of-scope` capability 只覆盖六类 `out_of_scope_kind` 和 `scenario=unsupported`，其 `expected_status` 为 `unsupported/degraded`；另校验 callable kind × scenario 的 4×4 共 16 条登记和每格的 expected_diagnostics

## Phase 2: Foundational

**Purpose**: 完成所有 User Story 依赖的上下文、profile、版本和 LSP data 基础设施。

**Checkpoint**: SemanticContext resolver、非空 profile、snapshot 版本门控和标准 diagnostics data 契约可独立测试后，才能开始 User Story 实现。

- [X] T004 [P] 在 `internal/semantic/context_test.go` 编写 `resolved`、`partial`、`missing`、`invalid` 和 fallback provenance 的 SemanticContext contract tests
- [X] T005 [P] 在 `internal/semantic/command_test.go` 编写 resolver 单元契约：分别测试输入命令缺失、当前文档 candidate 为空两种 Tops 默认命令触发条件；只统计 `NormalizeSourceURI` 后与当前文档 URI 相同的 candidate，并覆盖无关 URI、URI 别名、同 URI 多 candidate、`workspace_fallback_fields` 只含 workspace roots/macros 且不得补 target/driver/language/standard/pass/toolchain roots、workspace candidate 与 workspace fallback field 的明确冲突边界、response file/路径不可解析 candidate、16 层可用与 17 层/循环/非法 UTF-8/未闭合引号/嵌套相对路径失败、同一文档明确 language 冲突不得回退默认命令、低优先级语言信号不覆盖高优先级信号、两条等价 `compile_commands` 去重、两条非等价命令生成稳定 `ConflictSource.instance`、workspace/toolchain/fallback 三类 instance canonical object/golden vector、已有 command 但 profile 为空时只替换 profile、`.tops`/`.cpp`/`.cc` 与 Tops/C++ 信号矩阵、普通 C++ 隔离、整体 `source` 与固定 command field key 的 `field_source`/`field_resolution`/`field_conflict` map、`source_precedence` 与 `conflict_sort_order`、target conflict 和 roots/macros 缺失；明确断言无命令且来源完整为 `resolved`、来源缺失为 `partial`、无法形成上下文为 `missing`、任意字段冲突为 `invalid`、默认 command 五个字段和 profile 非空，以及 canonical JSON 的嵌套 object、重复 key/非有限数拒绝、字符串转义、Unicode、布尔/null、array 顺序、UTF-8 key 排序、`-0`、指数展开、相对 working directory、response file 失败和 golden vector。
- [X] T006 [P] 在 `internal/protocol/diagnostics_test.go`、`internal/index/source_test.go` 和 `internal/semantic/source_version_test.go` 增加内部字段到 JSON `documentVersion`、`contextVersion`、`sourceVersion`、`contextStatus`、`profileId`、`passKind`、`semanticKind` 的完整 wire mapping contract tests；`internal/index/source_test.go` 只断言 `NormalizeSourceURI`，semantic 包测试 `SourceVersionEntry`、`AggregateSourceVersion`、重复 URI、版本冲突、root source 缺失和 semanticKind 固定集合
- [X] T007 [P] 在 `internal/semantic/store_test.go` 编写 document/context/header/generation/epoch/source/header version maps 版本提交、`complete/degraded/stale` 生命周期、unsupported_reason、stale 丢弃、SemanticSnapshot 直接 profile_id/context_status/source_version_fingerprint 和 resolver 输出始终携带 profile 的类型/状态断言 tests
- [X] T008 [P] 实现 `internal/semantic/command.go` 和 `internal/semantic/canonical.go` 的 `CompilationCommand`、`FieldValue`、canonical JSON serializer、response-file expander、四类 `ConflictSource.instance`、固定 command field key maps 和默认命令常量模型；expander 负责 candidate working directory 基准、16 层深度优先嵌套、UTF-8/token quoting/cycle 失败，canonical JSON 使用无尾部换行的 UTF-8 bytes，instance 使用对应 canonical object 的 SHA-256 小写 hex，path/toolchain/workspace metadata 的归一化输入遵守 semantic-context contract
- [X] T009 [P] 实现 `internal/semantic/context.go`、`internal/semantic/profile.go` 的 SemanticContext/SemanticProfile 状态、fallback provenance、toolchain/workspace roots/macros 和 pass views
- [X] T010 导出 `internal/index.NormalizeSourceURI`，实现 `internal/semantic/source_version.go` 的 `SourceVersionEntry`/`AggregateSourceVersion`，并实现 `internal/semantic/resolver.go` 的当前文档 URI candidate 过滤、语言信号优先级、candidate 去重/冲突、字段合并与 `field_conflict`、输入命令缺失时的默认 Tops `CompilationCommand`、Tops-signaled document 范围、默认 GCU400/cpp-default 命令、target conflict 和 no-profile 边界替换；完整 `workspace candidate` 参与等价/冲突判断，`workspace fallback field` 只补齐缺失工程 roots/macros；非等价 candidate 或明确 language 冲突不得合成默认命令，默认/toolchain 来源只补齐缺失字段，不与明确字段形成冲突；依赖 T008/T009 的 model、canonicalization 和状态定义
- [X] T011 实现 `internal/semantic/store.go` 的按 URI/version semantic snapshot store，以及提交前后的 freshness 校验
- [X] T012 [P] 修改 `internal/protocol/diagnostics.go` 的 `DiagnosticData`，以 `omitempty` 追加 semantic/profile/context/pass 字段并保持旧 client 兼容
- [X] T013 [P] 在 `internal/server/semantic_pipeline_test.go` 增加 parse -> index -> semantic 的 stale/取消/generation/epoch contract tests
- [X] T014 接入 `internal/server/server.go`、`internal/server/parse_context.go` 和 `internal/server/analysis.go` 的 SemanticContext resolver 与版本快照入口，不改变现有 parser/index 行为

## Phase 3: User Story 1 - 声明、作用域和基础类型语义 (Priority: P1) 🎯 MVP

**Goal**: 在现有 AST/index 上提供声明、作用域、名称查找、基础 scalar/pointer/reference、函数参数和返回值 semantic facts，并发布确定性基础诊断。

**Independent Test**: 只运行 US1 semantic unit/server tests，使用单文件 declarations/types/calls fixture；不需要 GCU、Clang、hover 或 completion。

### Tests for User Story 1

- [X] T015 [P] [US1] 创建 `testdata/semantic/declarations/valid.tops`、`invalid.tops`、`incomplete.tops` 和 `context-boundary.tops`，覆盖 namespace/block/function/parameter/type/variable/field
- [X] T016 [P] [US1] 创建 `testdata/semantic/types/valid.tops`、`invalid.tops`、`incomplete.tops` 和 `context-boundary.tops`，覆盖标准标量隐式转换、指针/引用转换、user-defined conversion `conversion_status=unsupported` 且不产生 `tops-semantic-type-mismatch` 或 `tops-semantic-invalid-call` Error、以及上下文缺失
- [X] T017 [P] [US1] 创建 `testdata/semantic/calls/basic/valid.tops`、`invalid.tops`、`incomplete.tops`、`context-boundary.tops` 和 `overload-dependent.tops`，只覆盖 CallFact 的调用点、候选、参数转换和恢复节点；overload-dependent 预期为 `candidate_status=ambiguous/unresolved`、`expected_status=degraded`，多个候选只允许 `tops-semantic-ambiguous-name` Warning，不得发布确定性 Error，不重复验证 CallableFact 声明
- [X] T018 [P] [US1] 在 `internal/semantic/lookup_test.go` 增加 scope、qualified name、active/unknown/inactive candidate 和 declaration/definition tests
- [X] T019 [P] [US1] 在 `internal/semantic/type_test.go` 增加 scalar、pointer、reference、cv、conversion kind 和 unresolved type tests
- [X] T020 [P] [US1] 创建 `testdata/semantic/callables/{host,device,host-device,kernel-entry}/{valid,invalid,incomplete,context-boundary}.tops` 共 16 个 fixture，其中 `callables/host/valid.tops` 登记为未标注 Tops 函数的 `default-host` 子场景并设置 `default_execution_space=true`；另创建 `testdata/semantic/out-of-scope/{templates,constant-evaluation,access-control,linkage,code-generation,overload-resolution}.tops` 六类 fixture；在 `internal/semantic/callable_test.go` 增加 parameter order、return type、declaration-only 和四类执行空间事实，在 `internal/semantic/out_of_scope_test.go` 逐类断言 `expected_status=unsupported/degraded`；user-defined conversion 的 `unsupported` 状态以及不产生 `tops-semantic-type-mismatch`/`tops-semantic-invalid-call` Error 由 T017 的 CallFact tests 覆盖
- [X] T021 [P] [US1] 在 `internal/server/semantic_us1_test.go` 增加 semantic snapshot、diagnostic code/range/severity 和标准 `publishDiagnostics` tests

### Implementation for User Story 1

- [X] T022 [P] [US1] 在 `internal/semantic/model.go` 定义 `DeclarationFact`、`ScopeFact`、`SemanticType`、`CallableFact`、`ReferenceFact`、`CallFact` 和带 `calls` 字段的 `SemanticSnapshot`
- [X] T023 [US1] 在 `internal/semantic/lookup.go` 基于 `IndexSnapshot` 实现作用域、qualified name、条件状态和版本筛选的名称查找
- [X] T024 [US1] 在 `internal/semantic/types.go` 实现 scalar/cv/pointer/reference 类型表示、标准隐式转换和 unresolved 标记
- [X] T025 [US1] 在 `internal/semantic/callable.go` 实现参数/返回值摘要、声明定义关联、唯一 callable 判断和基础参数兼容检查
- [X] T026 [US1] 在 `internal/semantic/call.go` 构建 CallFact 的调用点、候选、逐参数 conversion status 和 `SemanticSnapshot.calls`，并在 `internal/semantic/checker.go` 组装 US1 semantic facts 与 `tops-semantic-unknown-name`、`tops-semantic-ambiguous-name`、`tops-semantic-type-mismatch`、`tops-semantic-invalid-call`
- [X] T027 [US1] 在 `internal/server/semantic_analysis.go` 接入 US1 semantic snapshot 构建，并在 `internal/server/diagnostics.go` 映射基础 semantic diagnostics
- [X] T028 [US1] 在 `internal/server/semantic_us1_test.go` 通过 `.tops`、`languageId=tops` 和普通 `.cpp` fixture 验证 US1 独立验收条件

**Checkpoint**: US1 可以独立输出声明/作用域/基础类型 facts 和标准 diagnostics，既有 navigation/index tests 不回归。

## Phase 4: User Story 2 - host/device、地址空间和向量约束 (Priority: P1)

**Goal**: 使用 profile/pass views 检查已确认或默认 GCU400 的 host/device 调用边界、地址空间和基础 vector 约束，并按 pass 发布诊断。

**Independent Test**: 只运行 US2 semantic/server/protocol tests；使用 resolved `gcu400-default`、invalid target、host/device 双视图 fixture，检查 code、range、severity 和 JSON `Diagnostic.data.passKind`。

### Tests for User Story 2

- [X] T029 [P] [US2] 创建 `testdata/semantic/calls/host-device/valid.tops`、`invalid.tops`、`incomplete.tops` 和 `context-boundary.tops`，只覆盖调用点、候选和 host/device 结果，不重复创建 callable 声明 fixture
- [X] T030 [P] [US2] 创建 `testdata/semantic/address-space/valid.tops`、`invalid.tops`、`incomplete.tops` 和 `context-boundary.tops`，覆盖地址空间合法、非法、不完整和 profile/context 边界
- [X] T031 [P] [US2] 创建 `testdata/semantic/vectors/valid.tops`、`invalid.tops`、`incomplete.tops` 和 `context-boundary.tops`，覆盖 vector element/lane/width/alignment 四类场景
- [X] T032 [P] [US2] 在 `internal/semantic/call_boundary_test.go` 读取 T020 的 16 个 callable fixture，分别断言四类 canonical callable kind；其中 `default-host` 只是 host kind 的子场景且 `default_execution_space=true`，不增加第五类，另测普通 C++ `default_execution_space=false`、host-only、device-only、host/device 两侧可用和单侧 `availability=missing` 四种 pass-view 组合，断言单侧 pass 缺失时整体为 `partial`，无法形成语言/命令上下文时才为整体 `missing`；host view 只保留 host 可调用 candidate，device view 只保留 device 可调用 candidate，`host-device` candidate 可出现在两侧，缺失或不可用 candidate 不得标为 resolved，并验证每个 candidate 的 symbol_id、pass provenance 和 candidate status
- [X] T033 [P] [US2] 在 `internal/semantic/target_rules_test.go` 增加 address-space、vector element/lane/width/alignment、resolved fallback 和 invalid/partial suppression tests
- [X] T034 [P] [US2] 在 `internal/server/semantic_pass_diagnostics_test.go` 增加 host/device 分开发布、合法 pass 不发布、pass-aware dedup、pass-neutral context/incomplete diagnostic 使用 `passKind=not-applicable`、JSON `Diagnostic.data.passKind`、non-empty range 和单侧 pass 缺失状态 tests
- [X] T035 [US2] 在 `internal/server/semantic_pass_diagnostics_test.go` 只增加 semantic severity mapping assertions；pass 发布、合法 pass 不发布和 pass-aware dedup 由 T034 覆盖，wire schema 由 T006 覆盖；依赖 T034 对同一测试文件的基础断言

### Implementation for User Story 2

- [X] T036 [US2] 在 `internal/semantic/call_boundary.go` 实现 host/device pass view、resolved Tops 未标注函数默认 host、跨空间调用和 kernel entry 规则
- [X] T037 [US2] 在 `internal/semantic/target_rules.go` 实现 profile 驱动的 address-space 和 vector constraint provider/checker，禁止跨 profile 推断
- [X] T038 [US2] 在 `internal/semantic/diagnostics.go` 实现 v1 `tops-semantic-*` code registry、severity policy、最小 range、related information 和 pass-aware dedup；调用 T010 的 `AggregateSourceVersion`，root source 固定为当前 `document_uri` 的规范化 URI，root source 缺失时抑制 semantic diagnostic
- [X] T039 [US2] 修改 `internal/server/diagnostics.go`、`internal/server/observability.go`，按 semantic severity 映射 LSP diagnostics，携带 profile/context/pass/version 并保持既有 parser/index diagnostics 行为
- [X] T040 [US2] 在 `internal/server/semantic_analysis.go` 将 host/device 两个 pass 的 checker 结果合并为同一 snapshot 但保留独立诊断来源、status、rules_available 和 provenance，覆盖 host-only/device-only/两侧可用/单侧缺失
- [X] T041 [US2] 在 `integration/lsp_process_test.go` 增加标准 `textDocument/publishDiagnostics` 的 host/device、address-space、vector、UTF-16/CRLF 和 invalid params 契约测试

**Checkpoint**: US2 在 resolved GCU400 fallback 下能报告目标错误，在 invalid/partial/missing 下按规则抑制；host/device 诊断不互相吞并。

## Phase 5: User Story 3 - SemanticContext 降级和版本丢弃 (Priority: P1)

**Goal**: 保证默认命令、工具链/workspace provenance、header/context/document 变化和 stale 结果在完整 server 生命周期中可观察且不污染新结果。

**Independent Test**: 只运行 US3 resolver/stale/header/integration tests，提交无命令、默认命令、target conflict、宏/header 变化和连续文档版本，检查最终 snapshot 与诊断版本。

### Tests for User Story 3

- [X] T042 [P] [US3] 在 `internal/semantic/default_command_test.go` 验证 server 集成路径把 resolver 输出接入最终 `SemanticContext`/snapshot：输入命令缺失或当前文档 candidate 为空时 `.tops`/`languageId=tops` 使用默认命令，明确 command/candidate 冲突不被默认命令覆盖；验证 `host+device` 双视图、`c++17`、非空 profile、`resolved`/`partial`/`missing`/`invalid` 状态传播、普通 C++ 隔离和 stale 提交边界。T005 负责 resolver 的 candidate、字段 map、canonical JSON 和 instance golden，不在此重复。
- [X] T043 [P] [US3] 在 `internal/semantic/provenance_test.go` 使用 fake source provider 调用计数和 provider adapter contract 验证 `CommandCandidateProvider` 输出的 `{document_uri,workspace_root,candidates,workspace_fallback_fields,toolchain_provenance}` 精确结构、candidate 的 `version`/字段类型/null 规则、`field_source`/`field_resolution`/`field_conflict` 传递、工具链标准库/Tops headers、`source_kind=header` 的 URI/version/context cache、`source_kind=macro` 的 workspace descriptor/config version cache、workspace 工程 headers/宏和 `workspace_fallback_fields`；来源缺失状态和 missing/unresolved 状态由 T005/T042 验证；workspace enumeration 期望为 0，唯一引用工程 URI 的读取次数必须等于唯一 URI 数量，重复引用必须命中缓存，provider 不执行 T010 的 language/candidate/default 状态决策
- [X] T044 [P] [US3] 创建并使用 `testdata/semantic/default-context/`、`testdata/semantic/stale/` fixture，在 `internal/server/semantic_stale_test.go` 验证 document/context/header/source/generation/epoch 变化、旧结果丢弃、输入命令缺失仍使用非空默认 command、resolver 输出 profile 非空/无 no-profile 类型和 `stale_result` 字段
- [X] T045 [P] [US3] 在 `internal/server/semantic_context_test.go` 验证 Tops target 缺失且默认核心字段及所需来源完整时生成 resolved default、来源缺失时保持 partial、target conflict 保持 invalid、target conflict 与 roots/macros 缺失重叠时仍优先 invalid、host-only/device-only/双侧可用/单侧 pass 缺失四种组合的整体状态，以及 workspace 变化递增 context version
- [X] T046 [P] [US3] 在 `testdata/semantic/conditions/` 增加 active/inactive/unknown macro cases，并在 `internal/semantic/condition_test.go` 验证工具链/工程宏 provenance

### Implementation for User Story 3

- [X] T047 [US3] 实现 `CommandCandidateProvider` 的 source provider 接口和 `WorkspaceSourceCache` 的工具链/workspace roots/macros 读取、`source_kind`/URI-or-macro-descriptor/version/context cache key、缺失来源和 header/macro 失效联动；输出固定 `{document_uri,workspace_root,candidates,workspace_fallback_fields,toolchain_provenance}`，其中 `workspace_fallback_fields` 只含 workspace roots/macros/version/source/resolution，不重复实现 T010 的 resolver、candidate、workspace candidate/fallback field 或 default/invalid 状态决策
- [X] T048 [US3] 完善 `internal/server/parse_context.go`、`internal/server/server.go`，将 T047 的 `CommandCandidateProvider` 结果接入 T010 resolver，完成默认 `CompilationCommand` 注入和 `SemanticContext` version 更新
- [X] T049 [US3] 完善 `internal/server/analysis.go`、`internal/server/index_analysis.go` 的 semantic/index 原子提交、header dependent invalidation、WorkspaceSourceCache 失效和 no-profile 路径删除
- [X] T050 [US3] 完善 `internal/server/observability.go` 的 semantic operation、profile/pass/context、旧新版本和 `stale_result` 日志字段，确保不记录完整源码
- [X] T051 [US3] 在 `internal/server/stale_diagnostics_test.go` 和 `internal/server/index_observability_test.go` 增加 semantic diagnostics 清理、旧版本覆盖计数为零和 fallback provenance 观测测试
- [X] T052 [US3] 在 `integration/lsp_process_test.go` 增加 didOpen/didChange/header/context 变化、取消和旧请求响应丢弃的端到端测试

**Checkpoint**: US3 可以在无真实 compile_commands 的 Tops 文档上完成默认 GCU400 semantic 分析，并证明旧结果不会覆盖当前版本。

## Phase 6: User Story 4 - Clang 离线对照验证 (Priority: P2)

**Goal**: 复用现有 parser comparison 入口验证 semantic categories/ranges/provenance，不把 Clang 变成 runtime 依赖或消息来源。

**Independent Test**: 运行有完整 provenance 的离线 comparison fixture，以及无 Clang 可执行文件的纯 Go semantic tests；输入不一致时得到 `not-comparable`。

### Tests for User Story 4

- [X] T053 [P] [US4] 在 `testdata/semantic/comparison_manifest.json` 增加 `fixture_case_id`、`document_uri`、派生的 `root_source_uri`、source files/hash、working directory、raw/normalized arguments、profile/pass、include/macro provenance、header versions、compiler/tool version、sourceVersion fingerprint 字段，并指定 `testdata/semantic/expected/semantic_comparison.json` 输出
- [X] T054 [US4] [执行依赖 T053/T057/T058] 在 `internal/semantic/comparison_test.go` 的 `TestSemanticComparisonManifest` 增加 manifest `fixture_case_id` 引用完整性、字段类型/枚举、相同输入可比较、缺字段/字段不一致 `not-comparable`、`document_uri` 缺失或无法规范化、`root_source_uri` 必须等于规范化当前文档 URI 且存在于 `source_files`、`source_files.content_hash`/`header_versions` 到 `SourceVersionEntry` 的映射、`host+device` 必须生成且仅生成 host/device 两个实际 pass result、每个 `go_diagnostics.passKind` 必须等于所属 pass、normalized category/range、共享 `AggregateSourceVersion` 的 sourceVersion aggregate/重复 URI/冲突版本、root source 缺失、顶层 `neutral_diagnostics` 的 `passKind=not-applicable` 与实际 pass 结果隔离、`not-comparable` 不生成用户可见诊断、以及无原始 message 转发 tests
- [X] T055 [US4] 在 `internal/parser/comparison_contract_test.go` 扩展 parser comparison runner adapter contract，只验证 semantic comparison manifest/golden 的读取、输出路径、跳过 Clang 和结果转交边界；T054 负责 pass_results、passKind、顶层 `neutral_diagnostics`、`not-comparable` 和 semantic category/range 规则，避免重复实现断言；T055 依赖 T058 对 runner 接口的修改
- [X] T056 [P] [US4] 在 `internal/server/runtime_boundary_test.go` 验证 server semantic path 不启动 Clang/clangd，未设置 `TOPS_LSP_CLANG` 时纯 Go tests 仍可运行

### Implementation for User Story 4

- [X] T057 [P] [US4] [执行依赖 T010] 在 `internal/semantic/comparison.go` 实现 semantic comparison record、source hash 和 `not-comparable` 分类；`root_source_uri` 必须规范化并校验属于 `source_files`，`source_version_fingerprint` 必须调用 T010 的 `AggregateSourceVersion`，不重复实现 URI/version aggregate
- [X] T058 [P] [US4] 修改 `internal/parser/comparison_test.go`、`internal/parser/comparison_contract_test.go` 的离线 runner 接口，使 parser 默认仍读取原有 parser manifest，semantic runner 显式读取 `testdata/semantic/comparison_manifest.json` 并写入 `testdata/semantic/expected/semantic_comparison.json`，且不进入 server runtime
- [X] T059 [US4] [执行依赖 T054/T057/T058] 在 `testdata/semantic/expected/semantic_comparison.json` 保存 normalized comparison expectations，不保存完整 Clang message 或凭据
- [X] T060 [US4] 将 Clang 对照结果限制为测试报告/ComparisonRecord，禁止写入 `publishDiagnostics` 或 server runtime semantic cache

**Checkpoint**: US4 可独立证明相同 provenance 的对照可复现，输入不一致可定位，Go server 不依赖 Clang。

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: 完成兼容性、文档和全量验证，不扩大 semantic scope。

- [X] T061 [P] 在 `internal/protocol/diagnostics_test.go`、`internal/server/diagnostics_test.go` 和 `integration/lsp_process_test.go` 完成旧 diagnostics JSON、UTF-16/CRLF、empty/null/cancel 和旧 client 忽略新 data 字段的回归验证
- [X] T062 [P] 在 `internal/server/observability_test.go` 增加无完整源码日志、profile/pass/context version 和 stale reason 的隐私回归测试
- [X] T063 [P] 更新 `specs/006-semantic-baseline/quickstart.md`、`contracts/*.md` 和必要的仓库开发文档，使命令、字段和实现保持一致
- [X] T064 在 T015-T020、T029-T031、T044、T046、T053、T057-T058 fixture/runner 任务完成后，运行 `specs/006-semantic-baseline/quickstart.md` 中的定向 Go tests、integration、`go test ./internal/semantic -run TestSemanticComparisonManifest`、full tests、race、vet、build、`go run ./cmd/tops-semantic-check --manifest testdata/semantic/manifest.json`，确认 manifest 的六类 capability、六类 out-of-scope 和 16 个 callable case 实际存在且路径可读，并运行 `git diff --check`
- [X] T065 检查 `internal/parser`、`internal/index`、`internal/server/navigation.go` 既有测试与默认普通 C++ 文件行为无回归，并确认 `llvm-project`、`topsop` 和外部 checkout 无修改

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**：无依赖；T001-T003 可并行。
- **Phase 2 Foundational**：依赖 Phase 1；T004-T007 的契约测试可并行，T008/T009/T012 可分文件实现，T010 依赖 T008/T009 的 model、canonicalization 和状态定义，T011/T014 依赖对应 store/context 模型；T008-T014 完成后才能开始 User Story。
- **Phase 3 US1**：依赖 Phase 2；US1 是 MVP，为 US2 的 semantic facts 提供输入。
- **Phase 4 US2**：依赖 US1 的 model/lookup/callable facts，且依赖 Phase 2 的 protocol/store。
- **Phase 5 US3**：依赖 Phase 2；默认命令与 stale pipeline 可与 US2 的 checker 实现并行，但最终集成依赖 US1/US2 snapshot 结构。
- **Phase 6 US4**：comparison schema/runner tests 依赖 Phase 2；最终 semantic 对照验收依赖 US1/US2/US3 的真实 checker、diagnostic、profile/pass 和 context 结果；不依赖设备或 Go server runtime 中的 Clang。
- **Phase 7 Polish**：依赖所有目标 User Story 完成。

### User Story Dependencies

- **US1 (P1)**：Foundation 完成后独立可交付；建议作为 MVP。
- **US2 (P1)**：依赖 US1 已交付的声明、scope、type、callable model；完成上游模型后，可独立运行 US2 测试命令验收目标规则和 pass diagnostics。
- **US3 (P1)**：依赖 Foundation 的 resolver/store；默认命令和 provenance tests 可先独立运行，完整 stale 验收再接入 US1/US2 snapshot。
- **US4 (P2)**：runner/schema tests 可在 Foundation 后独立运行；最终对照验收依赖 US1/US2/US3 的真实 semantic 输出，不依赖设备，Clang 只在离线 runner 中使用。

### Parallel Opportunities

- Phase 1：T001、T002、T003 可并行。
- Phase 2：T004、T005、T006、T007、T013 可并行编写测试；实现阶段 T008/T009/T012 可分文件并行，T010/T011/T014 依赖模型。
- US1：T015-T021 可并行准备 fixture/tests；T022、T024、T025 可分文件实现，T023 依赖 T022，T026-T028 依赖 model/checker。
- US2：T029-T034 可并行准备测试，T035 在 T034 后补充同一测试文件的 severity assertions；T036、T037 可分文件实现，T038/T039 依赖诊断模型，T040/T041 依赖 checker 和 mapping。
- US3：T042-T046 可并行准备；T047/T050 可分文件实现，T048 依赖 T047/T010 的 provider/resolver 接口，T049 依赖 store/pipeline，T051/T052 依赖集成。
- US4：T053、T056 可并行编写；T057/T058 可分文件并行实现，实际执行先完成 T057/T058，再运行 T054 和 T055，最后由 T059 生成 golden；T060 依赖 runner 和 comparison record。
- Polish：T061-T063 可并行，T064-T065 在所有实现合并后执行。

## Implementation Strategy

### MVP First

1. 完成 Phase 1-2：Context/Profile/Store/Diagnostic data 基础。
2. 完成 US1：声明、scope、基础类型和唯一 callable diagnostics。
3. 执行 T028 的独立 US1 测试和现有 parser/index/server 回归。
4. 只有 US1 稳定后再启用 US2 的目标规则和 US3 的默认命令全链路。

### Incremental Delivery

1. Foundation：非空 profile、SemanticContext、版本和标准 diagnostics data。
2. US1：普通 semantic facts 和基础 diagnostics，可作为第一版 MVP。
3. US2：GCU400 host/device/address/vector 规则和按 pass diagnostics。
4. US3：默认命令、headers/macros provenance、stale 和 observability。
5. US4：离线 Clang comparison，保持 runtime 独立。
6. Polish：全量测试、race、vet、build 和 quickstart 验证。

### 完成门槛

- 所有任务保持 checkbox、连续任务 ID、可选 `[P]`/`[USx]` 标签和明确路径的格式；
- 每个 User Story 都有独立测试入口和 checkpoint；
- 默认 Tops 命令、GCU400 fallback、组合 pass、no-profile 删除和 LSP diagnostics 契约均有测试；
- 不运行设备 workload，不把 Clang 接入 server runtime，不修改外部 checkout。

## Phase 8: Convergence

- [X] T066 [US3] 将 `CommandCandidateProvider`、`WorkspaceSourceCache`、toolchain/workspace roots 和宏 provenance 接入 server resolver，完成默认命令字段合并、已引用来源缺失状态和 context version 联动 per FR-014/FR-035/FR-038 (partial)
- [X] T067 [US3] 修正 resolver 的 target/language/status 优先级，使非等价 target conflict 保留 `gcu400-default` 且为 `invalid`，并覆盖普通 C++ 来源状态与字段 provenance per FR-013/FR-033/SC-012 (contradicts)
- [X] T068 [US1] 修正 scope lookup 和 semantic diagnostics 的 active/unknown/inactive 条件筛选，确保 active 候选优先且 unknown 不产生未经证明的确定性错误 per FR-004/FR-011 (partial)
- [X] T069 [US1] 将基础 scalar/pointer/reference conversion 接入 `CallFact` 和 snapshot diagnostics，完整记录参数转换并保持 user-defined 与 overload-dependent conversion 为 `unsupported`/`degraded` per FR-007/SC-010 (missing)
- [X] T070 [US2] 将 address-space/vector 规则接入 semantic snapshot 的独立 host/device pass view，补齐 rules/provenance、单侧缺失状态和 pass-specific facts per FR-009/FR-010/FR-036/FR-039/SC-020 (missing)
- [X] T071 [US2] 完成 semantic diagnostic registry、最小源码 range、context degraded fallback、related information、固定 `semanticKind` 和包含 candidate/condition/pass 的稳定去重 per FR-020/FR-022/FR-024/SC-002 (partial)
- [X] T072 [US3] 在 semantic snapshot 提交边界重新校验 document/context/source/header/generation/epoch 版本，统一 header 失效、旧诊断清理和 `stale_result` 观测 per FR-026/FR-027/FR-028/SC-004 (partial)
- [X] T073 [US4] 实现独立 semantic offline comparison runner，补齐 comparison manifest/golden schema、共享 source-version fingerprint、pass results、`not-comparable` 规则，并确保 Clang 不进入 server runtime per FR-029/FR-030/SC-006 (missing)
- [X] T074 [P] 补充 fixture 预期行为、UTF-16/CRLF/Unicode/zero-width、版本竞争和无外部编译器进程的可执行 contract tests，不只验证 manifest 登记 per FR-031/SC-003/SC-005/SC-007 (partial)
- [X] T075 [P] 按实现后的字段和 runner 更新 quickstart/contracts/开发文档，并执行最终 semantic manifest、full/race/vet/build/integration 与既有 parser/index/navigation 回归 per plan: Phase 7 (partial)

## Phase 9: Convergence

- [X] T076 [P1] [US3] 将 `CommandCandidateProvider`、`WorkspaceSourceCache`、toolchain/workspace roots 和宏 provenance 接入 server 默认路径，按已引用 URI/descriptor/version 读取与缓存，并联动来源失效和 context version per FR-014/FR-035/FR-038/SC-017/SC-018 (partial)
- [X] T077 [P1] [US3] 修正 resolver 的 existing command/profile、candidate 缺失字段状态和冲突优先级，使实际 target 选择对应 profile，未核验 roots/macros 保持 unresolved，target/language conflict 保留 `invalid` 和完整 field provenance per FR-013/FR-033/FR-041/FR-042/SC-012/SC-014 (contradicts)
- [X] T078 [P1] [US1] 让 semantic snapshot 消费 parser/index 的 recovered/partial/incomplete 状态，保留已确认 facts，标记 degraded，并发布带合法 range、版本和 `Hint` severity 的 `tops-semantic-incomplete-input` per FR-015/FR-019/FR-031/SC-003 (missing)
- [X] T079 [P1] [US1] 修正 qualified name 和 scope lookup 的多候选处理，按 active/unknown/inactive 条件保留候选并在无法唯一确定时发布 ambiguity，不得从 map 顺序静默选择 per FR-004/FR-011/SC-003 (partial)
- [X] T080 [P1] [US1] 将解析得到的 named/user-defined 类型和 overload-dependent call 转换接入 `CallFact`，分别记录 `unsupported`/`degraded` 状态并禁止错误的 type-mismatch 或 invalid-call Error per FR-005/FR-007/SC-010 (partial)
- [X] T081 [P1] [US2] 从 resolver/server 输入提供真实 host/device `PassAvailability`，构建独立 pass facts，接通 address-space/vector 规则和单侧缺失状态，禁止组合 pass 在缺少 pass provenance 时发布目标 Error per FR-009/FR-010/FR-036/FR-039/SC-015/SC-020 (missing)
- [X] T082 [P1] [US2] 完成 semantic diagnostic range 门控、降级替代、conditional provenance、related information、固定 `semanticKind` 和 pass-aware 稳定去重，确保 target Error 没有可验证非空 range 时不进入 snapshot 或 LSP per FR-017/FR-020/FR-022/FR-024/SC-002/SC-005 (partial)
- [X] T083 [P1] [US3] 补齐 document/header/context 失效和取消路径的旧 semantic diagnostics 清理、source/header version 观测及 stale 计数测试，确保新结果完成前旧结果不再对外可用 per FR-026/FR-027/FR-028/SC-004 (partial)
- [X] T084 [P2] [US4] 增加独立 semantic offline comparison runner，读取专用 comparison manifest，校验双方源文本、raw/normalized arguments、target/pass、include/macro/header provenance 和版本一致性，按 pass 生成 `not-comparable` 记录且不进入 server runtime per FR-029/FR-030/SC-006 (missing)
- [X] T085 [P1] 完成 semantic fixture 的实际执行断言、UTF-16/CRLF/Unicode/zero-width、版本竞争和无外部编译器进程 contract tests，随后同步 quickstart/contracts/开发文档并执行 manifest、full/race/vet/build/integration 与既有 parser/index/navigation 回归 per FR-031/SC-001/SC-003/SC-005/SC-007/plan: Phase 7 (partial)

## Phase 10: Convergence

- [X] T086 [P1] [US3] 将 resolver 生成的有效 `SemanticContext`、默认 `CompilationCommand`、toolchain/workspace roots 与 macros 以及 pass view 接入 parser/index 实际分析入口，并补充 Tops 默认命令实际生效的集成断言 per FR-013/FR-035/FR-038/SC-012/SC-014 (verified by T095)
- [X] T087 [P1] 建立可执行 semantic fixture runner，读取每个 manifest source，构建 parser/index/context/snapshot 并断言 `expected_status`、`expected_diagnostics`、code、severity、range、pass 和降级结果 per FR-031/SC-001/SC-003/SC-005/SC-007 (verified by T097)
- [X] T088 [P1] [US2] 从真实 scope/callable 输入构建 caller/callee 和独立 host/device pass facts，移除用当前 pass 合成 caller 及 `unknown` call kind 的路径，并保留各 pass 的 status/rules/provenance per FR-036/FR-039/SC-015/SC-020 (verified by pass-view/checker tests)
- [X] T089 [P1] [US2] 将声明和基础操作中的 address-space/vector facts 接入 semantic checker，填充 element/lane/width/alignment 并执行 valid、invalid、incomplete 和 context-boundary fixture per FR-009/FR-010/SC-003/SC-015 (verified by fixture/target tests)
- [X] T090 [P1] [US3] 将已引用 header、toolchain/workspace macro 和其他 source provider 的 missing/unresolved 状态传播到 `SemanticContext` 的 `partial/degraded` 结果，抑制来源不足时的目标 Error per FR-013/FR-015/FR-016/SC-017/SC-018 (verified by source-degraded tests)
- [X] T091 [P1] [US3] 完成 resolver 的 canonical candidate 去重与全部 provenance 保留，合并 provider 的 `field_source`、`field_resolution`、`field_conflict` 并覆盖输入顺序置换和等价 candidate 场景 per FR-014/FR-042/SC-014 (verified by resolver contracts)
- [X] T092 [P1] 统一 semantic diagnostic registry、fixture 与 wire 的 code/severity/pass 规则，补齐 related information、条件状态去重和 snapshot 生成前的非空 range 门控 per FR-017/FR-020/FR-022/FR-024/SC-002/SC-005 (verified by diagnostic contracts)
- [X] T093 [P1] [US3] 扩展 semantic stale 提交与观测路径，记录 semantic profile/pass/context status 及旧新 document/context/source/header 版本，并补充 header、context、取消和版本竞争的 stale 计数测试 per FR-026/FR-027/FR-028/SC-004 (verified by T100)
- [X] T094 [P2] [US4] 实现独立 semantic offline comparison runner，读取 source bytes 并校验 raw/normalized arguments、root/source hash、target/pass、include/macro/header/compiler provenance；仅在离线流程使用 `TOPS_LSP_CLANG`，输入不一致生成 `not-comparable`，不得进入 server runtime per FR-029/FR-030/SC-006 (verified by T098)

## Phase 11: Convergence

- [X] T095 [P1] [US3] 修正 server 的 `CompilationContext` 到 `SemanticContext` 适配，保留明确 compile command language/mode 的优先级，使 `.tops` 配合明确普通 C++ command 使用 `cpp-default`，并补充 server 集成断言 per FR-037/SC-014/SC-016 (verified)
- [X] T096 [P1] [US3] 统一 candidate/default/existing command 路径的 toolchain/workspace source resolution 与 host/device pass view 状态传播，在 pass 缺失时同步刷新 `status`、`rules_available` 和 `provenance`，覆盖四种 pass-view 组合 per FR-013/FR-038/SC-017/SC-020 (verified)
- [X] T097 [P1] 完善 semantic fixture runner，移除 `out-of-scope` 对实际 diagnostics/status 的绕过，逐 case 比较 parser/index/context/snapshot 的 expected status、diagnostics、降级和 unsupported 结果 per FR-031/SC-001/SC-003/SC-008 (verified)
- [X] T098 [P2] [US4] 完成真实 semantic offline comparison runner，读取 source bytes 后构建 Go parser/index/context/snapshot 的 pass results，校验 provenance/hash，不复制 manifest 结果；修复可比较 fixture 的 source hash/golden，并仅在离线流程接入 `TOPS_LSP_CLANG` per FR-029/FR-030/SC-006 (verified)
- [X] T099 [P1] 在 semantic snapshot 与 LSP diagnostic 边界校验 root source 存在于 source/header versions 并由 `AggregateSourceVersion` 生成 fingerprint，缺失 root 时抑制 diagnostic，并补充伪造 fingerprint 和缺失 root contract tests per FR-020/FR-043/SC-002 (verified)
- [X] T100 [P1] 在 document/header/context 失效时清理已发布的旧 semantic diagnostics，使异步新分析完成前旧 LSP 结果不再可见，并补充版本竞争、取消和 header invalidation 的发布断言 per FR-027/SC-004 (verified)

## Phase 12: Convergence

- [X] T101 [P1] [US3] 在正常 `cmd/tops-lsp` 路径实现并注册 concrete `CommandCandidateProvider`，按 compile_commands、workspace candidate、toolchain provenance 和已引用来源提供字段级 provenance、candidate conflict、默认命令及缓存失效的真实 server 集成测试 per FR-014/FR-035/FR-042/SC-014/SC-017/SC-018 (missing)
- [X] T102 [P1] [US3] 修正 existing command 的 target/profile 选择和 toolchain/workspace roots/macros 的 unresolved 判定，保留明确 `gcu400`/其他 target 的 profile 与 field provenance，并增加 `gcu500`、缺失 roots/macros 和 profile 冲突测试 per FR-013/FR-035/FR-038/FR-042/SC-012/SC-014 (contradicts)
- [X] T103 [P1] [US2] 从真实 scope/callable 输入构建独立 host/device pass facts，补齐 `host-device` 与 `kernel-entry` caller/callee boundary，禁止将 `host+device` 作为诊断归属或在缺少 pass provenance 时发布目标 Error per FR-008/FR-036/FR-039/SC-011/SC-015/SC-020 (partial)
- [X] T104 [P1] [US2] 将 vector element/lane/width/alignment 约束绑定到具体 `SemanticProfile` 和 pass provenance，缺失约束时返回 degraded/context 结果，并增加不同 profile 与缺失 provenance 的测试 per FR-010/FR-015/SC-003/SC-015 (partial)
- [X] T105 [P1] [US1] 扩展 semantic fixture manifest 和 runner，逐 case 校验 declaration/scope/type/conversion/address-space/vector/pass facts 及 incomplete、context-boundary、unsupported 结果，不只依赖 diagnostics 比较 per FR-001/FR-003/FR-005/FR-007/FR-031/FR-044/SC-001/SC-003/SC-010 (partial)
- [X] T106 [P2] [US4] 按 comparison case 和实际 host/device pass 构造完整离线 Clang 输入，传递 working directory、raw/normalized arguments、target、include roots、宏、tool/header provenance，并在任一字段不一致时生成 `not-comparable` per FR-029/FR-030/SC-006 (contradicts)
- [X] T107 [P1] 增加 semantic diagnostics 的 Unicode、CRLF、EOF、零宽 range 和完整 `Diagnostic.data` 端到端断言，验证标准 `publishDiagnostics` 的 UTF-16 半开范围和 pass-neutral/pass-sensitive 映射 per FR-020/FR-023/FR-031/SC-002/SC-005 (partial)
- [X] T108 [P1] 补齐 semantic manifest validator 的固定 provenance 字段、唯一 condition ID、parent condition 引用、实际 source byte range、UTF-16 可转换性及 comparison manifest schema 负例校验 per FR-044/SC-001/SC-003/SC-005/SC-010 (partial)
- [X] T109 [P2] 将 comparison manifest 的 `source_hash`、`source_version_fingerprint` 及其他关键 provenance 字段设为必需输入，缺失时稳定生成 `not-comparable`，并补充缺字段 golden/contract tests per FR-029/SC-006 (partial)

## Phase 13: Convergence

- [X] T110 [P1] [US3] 修正 concrete command provider 对 `clang`/`clang++`/普通 C++ driver 的 language 推断，使无 `-x` 的 `.tops` compile command 仍使用 `cpp-default`，并增加真实 compile database 的普通 C++ 隔离测试 per FR-037/SC-016 (contradicts)
- [X] T111 [P1] [US3] 明确并实现 existing `CompilationContext`/`ExistingCommand`/`ExistingProfile` 与 candidate provider 的优先级，覆盖同时存在时的普通 C++、target、profile、field provenance 和 candidate conflict 结果 per FR-014/FR-041/FR-042/SC-014 (contradicts)
- [X] T112 [P1] [US2] 让 concrete provider 传递 workspace/toolchain/pass provenance 和 `PassAvailability`，并在对应 pass provenance 缺失时抑制 host/device、address-space/vector 目标 Error，补充正常 server 路径测试 per FR-008/FR-036/FR-039/SC-011/SC-020 (partial)
- [X] T113 [P1] [US1] 为 semantic manifest 的 declarations/types/calls/address-space/vectors 及 user-defined、overload-dependent conversion case 登记显式 `expected_facts`，逐 case 断言 declaration/scope/type、conversion status、candidate status、CallPassFact、vector/address-space facts per FR-007/FR-044/SC-010 (partial)
- [X] T114 [P1] 扩展 server/LSP wire contract matrix，使用真实分析路径覆盖全部 v1 semantic code 的 severity、UTF-16/CRLF range、`Diagnostic.data`、pass-neutral/pass-sensitive 和 resolved/degraded/missing 行为 per FR-017/FR-020/FR-023/FR-031/SC-002/SC-005 (partial)
- [X] T115 [P2] 为 comparison manifest 的 root URI、source hash/fingerprint、raw/normalized arguments、target/pass、include/macro/header/compiler provenance 增加 mutation tests，确保任一关键字段不一致稳定生成 `not-comparable` 并验证 Clang invocation 映射 per FR-029/FR-030/SC-006 (partial)
