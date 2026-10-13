# Tasks: P1.3 Tops C++ AST 和符号索引

**Input**: Design documents from [specs/005-ast-symbol-index/](.)

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: Go parser/index/server/protocol 行为、LSP 契约、文档同步和导航变化都必须有自动化测试；Clang 对照只作为独立验证，不是 Go 测试前置条件。

**Organization**: 任务按规格中的四个用户故事组织。Phase 1/2 是共享基础；用户故事阶段按 P1 顺序交付；最后阶段负责跨故事验证和文档收敛。

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 建立实现目录、测试材料入口和无新增依赖的基线。

- [X] T001 [P] 在 `go.mod`、`specs/005-ast-symbol-index/plan.md` 和 `specs/005-ast-symbol-index/quickstart.md` 中确认 Go 版本、标准库依赖和开发机验证命令，不引入第三方依赖。
- [X] T002 [P] 创建 `internal/index/`、`testdata/index/{single-file,multi-file,headers,conditions,incomplete,missing-context,stale,expected}/` 目录和带 `case_id`/计数字段的 `testdata/index/manifest.json` 最小结构。

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 提供所有用户故事依赖的 AST 元数据、索引模型、版本快照、协议模型和 server 提交边界。

**⚠️ CRITICAL**: 本阶段完成前不得开始用户故事的实现任务。

- [X] T003 [P] 为 `internal/parser/ast_index_metadata_test.go` 编写名称范围、声明角色、body 范围和不完整声明的失败测试。
- [X] T004 在 `internal/parser/ast.go`、`internal/parser/parser.go` 和 `internal/parser/construct_helpers.go` 增加 `NameRange`、`DeclarationRole`、`BodyRange` 元数据并保持现有 parser/recovery 契约。
- [X] T005 [P] 为 `internal/index/model_test.go` 编写 `SymbolRecord`、`ReferenceRecord`、`ScopeRecord`、条件状态、来源状态和 `IndexSnapshot` 模型测试。
- [X] T006 在 `internal/index/model.go` 实现数据模型、状态枚举、声明/定义位置、条件 provenance、header dependency 和索引快照字段。
- [X] T007 [P] 为 `internal/index/store_test.go` 编写版本匹配替换、旧快照拒绝、删除文档、stale 标记和并发读写测试。
- [X] T008 在 `internal/index/store.go` 实现线程安全的快照 Store、版本校验、原子替换、删除和依赖失效接口。
- [X] T009 [P] 为 `internal/protocol/navigation_test.go` 编写导航参数、`DocumentSymbol`、`SymbolKind` 和 capability JSON 的失败测试。
- [X] T010 在 `internal/protocol/navigation.go` 和 `internal/protocol/messages.go` 增加标准导航参数/结果模型及 `definitionProvider`、`referencesProvider`、`documentSymbolProvider` 能力字段。
- [X] T011 [P] 为 `internal/server/index_analysis_test.go` 编写 parser 结果版本、generation、epoch、context version 和 SourceProvider include roots 不匹配时不提交 index 的失败测试。
- [X] T012 在 `internal/server/server.go`、`internal/server/analysis.go`、`internal/server/parse_context.go` 和新建 `internal/server/index_analysis.go` 接入 index Store、SourceProvider options/constructor 与版本检查边界，令 `partialParseContextAt` 复制 roots/available 状态，但暂不实现具体符号查询。

**Checkpoint**: parser 能输出导航所需元数据，index Store 能拒绝过期快照，protocol 能编码三项标准导航模型，server 有可复用的版本提交边界。

---

## Phase 3: User Story 1 - 在当前文件中导航声明和引用 (Priority: P1) 🎯 MVP

**Goal**: 支持单文件中的函数、类型、属性、内建变量、声明/定义、引用、作用域和 document symbols。

**Independent Test**: 使用单文件 fixture，通过 index/server 测试在函数调用、类型使用、属性和 builtin 使用处验证 definition、references、documentSymbol、空结果和作用域筛选，不依赖 header 或设备。

### Tests for User Story 1 (REQUIRED for behavior changes)

- [X] T013 [P] [US1] 在 `testdata/index/single-file/basic.tops`、`testdata/index/expected/single-file-basic.json` 和 `testdata/index/manifest.json` 增加 namespace、函数声明/定义、参数/返回类型、执行空间、alias、属性/builtin 来源和 unresolved 名称材料。
- [X] T014 [P] [US1] 在 `internal/index/builder_test.go` 编写单文件 scope traversal、函数 signature/parameter/return/qualifier 摘要、属性参数/opaque 来源、builtin 字段访问/条件和 declaration/definition/NameRange 断言。
- [X] T015 [P] [US1] 在 `internal/index/query_test.go` 编写 definition、references、`includeDeclaration`、documentSymbol 嵌套、fresh 空数组、内部 stale/no-snapshot 状态、同名不同 scope 和 alias -> Object 测试；LSP `null` 由 server/protocol 契约测试覆盖。
- [X] T016 [P] [US1] 在 `internal/server/navigation_test.go` 编写单文件请求参数、位置转换、声明回退、未解析名称和当前 snapshot 查询测试。

### Implementation for User Story 1

- [X] T017 [US1] 在 `internal/index/builder.go` 实现 AST scope traversal、qualified name、声明/定义分类、函数 signature/parameter/return/qualifier 摘要、attribute/builtin/macro 记录、引用发现和稳定 `symbol_id` anchor。
- [X] T018 [US1] 在 `internal/index/query.go` 作为 active/unknown/inactive 候选规则的唯一核心实现，按 scope、qualified name、condition 和 source version 筛选 definition/references/documentSymbol；fresh 无匹配返回空数组，stale/no snapshot 由 handler 映射为 `null`。
- [X] T019 [US1] 在 `internal/server/index_analysis.go` 将当前 ParseResult 交给 index builder 并提交单文件快照，在 `internal/server/navigation.go` 实现三项导航 handler 的本地查询。
- [X] T020 [US1] 在 `internal/server/server.go` 注册 `textDocument/definition`、`textDocument/references` 和 `textDocument/documentSymbol`，并在 `internal/protocol/navigation.go` 完成 SymbolKind（含 alias -> Object）到标准 LSP 结果的映射；不重复实现候选筛选规则。
- [X] T021 [US1] 使用 `internal/index/builder_test.go`、`internal/index/query_test.go` 和 `internal/server/navigation_test.go` 验证 User Story 1 的独立验收场景并记录失败路径。

**Checkpoint**: 单文件打开后，definition、references 和 document symbols 可通过 Go server 查询；不存在的目标返回空结果，不创建虚假位置。

---

## Phase 4: User Story 2 - 查看文件、header 和多文件的符号结构 (Priority: P1)

**Goal**: 按需索引已打开文件和 include roots 中实际被引用的 header，支持跨文件导航、缺失 header、include cycle 和依赖失效。

**Independent Test**: 使用 `main.tops`、可加载 header 和缺失 header 两组 fixture，验证跨文件 definition/references/documentSymbol、本地降级结果、unresolved dependency 和 header 变化后的依赖刷新。

### Tests for User Story 2 (REQUIRED for behavior changes)

- [X] T022 [P] [US2] 在 `testdata/index/multi-file/`、`testdata/index/headers/`、`testdata/index/missing-context/` 和 `testdata/index/manifest.json` 增加跨文件、同一 header 多 requester、header URI 变化、缺失 header/include root、include cycle 和 missing-context 材料。
- [X] T023 [P] [US2] 在 `internal/index/source_test.go` 编写已打开文档优先、`SourceProvider.IncludeRoots()` 注入、include roots 按需解析、unresolved/missing/unreadable/cycle、source version 和不扫描 workspace 的失败测试。
- [X] T024 [P] [US2] 在 `internal/server/header_index_test.go` 编写 header didChange、header URI/version 变化、context invalidation、直接/间接 dependent stale、缺失 header degraded 和 include-cycle reason/非循环符号可查询测试。

### Implementation for User Story 2

- [X] T025 [US2] 在 `internal/index/source.go` 实现可注入 `SourceProvider`/`HeaderResolver`，包括 `IncludeRoots() (roots, available)`，支持已打开文档、include roots 解析、source version 和 unresolved/missing/unreadable 状态。
- [X] T026 [US2] 在 `internal/index/builder.go` 和 `internal/index/store.go` 实现 header SymbolRecord、include graph、反向依赖、循环检测、同一 header version 复用和 unresolved dependency。
- [X] T027 [US2] 在 `internal/server/index_analysis.go`、`internal/server/server.go` 和 `internal/server/parse_context.go` 接入 provider，令 header 变化使直接/间接依赖重新分析，但不启动 workspace 全量扫描。
- [X] T028 [US2] 使用 `internal/server/header_index_test.go` 和 `internal/index/source_test.go` 验证跨文件导航、缺失 header 和依赖失效的独立验收场景。

**Checkpoint**: 可读取的被引用 header 能提供可验证位置；缺失 header 不被当作空文件；依赖文件变化不会继续发布旧 header 定义。

---

## Phase 5: User Story 3 - 在编辑和不完整输入中保持部分索引 (Priority: P1)

**Goal**: 在逐字编辑、parser recovery、文档版本竞争、header/context 失效和关闭文档后保持当前版本的部分索引并丢弃旧结果。

**Independent Test**: 对同一 URI 提交递增 `didChange` 版本和不完整函数/属性/条件 fixture，验证当前符号保留、删除符号移除、旧任务丢弃、关闭后不发布和 UTF-16 range 稳定。

### Tests for User Story 3 (REQUIRED for behavior changes)

- [X] T029 [P] [US3] 在 `testdata/index/incomplete/`、`testdata/index/stale/` 和 `testdata/index/missing-context/` 增加截断函数参数/函数体/属性参数、六类条件指令及 parent-child 关系、删除声明、header URI/version 变化和版本乱序材料。
- [X] T030 [P] [US3] 在 `internal/index/partial_test.go` 编写 recovered/partial/degraded 快照、状态转换、局部 symbol 保留、删除声明清理、stable symbol ID、注释/字符串过滤、错误节点查询和 UTF-16 selection range 测试。
- [X] T031 [P] [US3] 在 `internal/server/stale_index_test.go` 编写 generation/epoch/context/header version 竞争、header URI 变化、旧任务晚到、close 后结果丢弃和取消测试。
- [X] T032 [P] [US3] 在 `internal/server/index_analysis_test.go` 增加 parser/index/diagnostic 同一 snapshot 提交、旧 index 不可查询和 partial query 状态测试。

### Implementation for User Story 3

- [X] T033 [US3] 在 `internal/index/builder.go` 和 `internal/index/store.go` 实现 partial/degraded 状态、当前快照整体替换、旧声明清理和 stale/discarded 状态转换。
- [X] T034 [US3] 在 `internal/server/index_analysis.go` 和 `internal/server/analysis.go` 将 index 构建纳入现有 latest-only 队列，提交前复用 document snapshot、generation、epoch 和 context version 校验。
- [X] T035 [US3] 在 `internal/server/server.go`、`internal/server/parse_context.go` 和 `internal/server/index_analysis.go` 处理 didClose、context invalidation、header dependent invalidation 和重新排队，不发布旧 AST/index/query result；其中 header dependent 部分显式依赖 US2 的 T025-T027。
- [X] T036 [US3] 使用 `internal/server/stale_index_test.go`、`internal/index/partial_test.go` 和 `testdata/index/incomplete/` 验证 User Story 3 的独立验收场景及 `go test -race` 下的结果一致性。

**Checkpoint**: 任意新文档版本都只查询当前快照；不完整输入保留可确认符号；旧版本、关闭文档和过期 header 结果不会覆盖或污染新结果。

---

## Phase 6: User Story 4 - 通过标准 LSP 方法使用导航结果 (Priority: P1)

**Goal**: 通过标准 LSP capability、请求参数、结果类型、空结果、错误、取消和过期行为交付可被现有 client 消费的导航接口。

**Independent Test**: 启动真实 stdio server，发送 initialize、didOpen、三项导航请求、非法参数和取消；检查 JSON 形状、capabilities、UTF-16 ranges、空/`null`结果和旧请求丢弃。

### Tests for User Story 4 (REQUIRED for behavior changes)

- [X] T037 [P] [US4] 在 `internal/protocol/navigation_contract_test.go` 编写 capability、`TextDocumentPositionParams`、`ReferenceParams`、`DocumentSymbolParams`、alias -> Object、fresh `[]` 和 stale `null` JSON 契约测试。
- [X] T038 [P] [US4] 在 `integration/lsp_process_test.go` 使用标准 LSP JSON decoder proxy 编写真实 stdio initialize capabilities、definition、references、documentSymbol、空数组/`null`和跨版本请求测试。
- [X] T039 [P] [US4] 在 `internal/server/navigation_contract_test.go` 编写 InvalidParams、未打开 URI、越界位置、RequestCancelled、stale `null`、`analysis-degraded`/`stale_result` code/event 和无私有字段测试。

### Implementation for User Story 4

- [X] T040 [US4] 在 `internal/server/server.go` 和 `internal/server/navigation.go` 完成标准请求参数校验、未打开文档错误、取消传播、过期结果处理和三项 capability 声明；不重复实现 T018 的候选规则。
- [X] T041 [US4] 在 `internal/server/navigation.go` 和 `internal/protocol/navigation.go` 将 T018 的候选结果编码为标准 LSP，固化 fresh 空数组/stale `null`、includeDeclaration、declaration fallback、alias -> Object 和 documentSymbol 可见性契约；不重新实现 index 筛选。
- [X] T042 [US4] 在 `integration/lsp_process_test.go` 和 `internal/server/server_test.go` 更新原有未知方法/生命周期断言，确认新增导航能力不破坏 initialize、shutdown、exit、didOpen/didChange/didClose、cancel 和 stdio framing。

**Checkpoint**: 现有标准 LSP client 可以直接消费三项导航结果；客户端不需要私有语义消息或文本扫描；错误、取消和过期行为可观察且不返回旧位置。

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: 完成全量 fixture、隐私检查、quickstart 验证和规格追踪。

- [X] T043 在 `internal/index/manifest.go`、`cmd/tops-index-check/main.go`、`testdata/index/manifest.json`、`testdata/index/expected/`、`testdata/index/missing-context/` 和 `specs/005-ast-symbol-index/spec.md` 实现并运行 manifest validator，输出分类总数/通过数/失败 case ID，并对齐所有 symbol kind、condition、header URI、partial、stale、out-of-scope negative 和 LSP 验收材料。
- [X] T044 在 `internal/server/observability.go` 和 `internal/server/index_observability_test.go` 实现并验证 index/query 日志与 diagnostics 映射：包含脱敏 URI、operation、symbol kind、版本、condition、状态、结果数量、耗时、diagnostic code 和 error kind；验证 `analysis_degraded`/`stale_result` 事件及 include-cycle reason，不记录完整源码或 LSP payload。
- [X] T045 在 `specs/005-ast-symbol-index/quickstart.md` 补充实现后的实际测试入口、跳过 Clang 对照的记录格式和失败恢复说明。
- [X] T046 在 `specs/005-ast-symbol-index/quickstart.md` 规定的开发机仓库根目录执行 `go run ./cmd/tops-index-check --manifest testdata/index/manifest.json`、`go test ./...`、`go test -race ./...`、`go vet ./...` 和 `git diff --check`，记录失败项及其归属文件。
- [X] T047 对照 `specs/005-ast-symbol-index/spec.md` 的 FR-001 至 FR-036、SC-001 至 SC-006、`specs/005-ast-symbol-index/data-model.md`、两个 contract 文件和 `.specify/memory/constitution.md` 完成最终覆盖/五项原则检查，修复遗漏的任务路径或验收材料。

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**：无依赖；T001/T002 可并行。
- **Foundational (Phase 2)**：依赖 Setup；T003/T005/T007/T009/T011 的测试可并行准备，T004/T006/T008/T010/T012 按对应测试和模型依赖完成；阻塞全部用户故事。
- **User Story 1 (Phase 3)**：依赖 Foundational；完成后得到 MVP 单文件导航。
- **User Story 2 (Phase 4)**：依赖 Foundational 和 US1 的 index builder/query 接口；header provider/fixture 可并行，server dependent invalidation 依赖 provider。
- **User Story 3 (Phase 5)**：依赖 Foundational 和 US1 的 snapshot 查询；partial/stale 的文档部分可先开始，T035 的 header dependent 部分必须等待 US2 的 T025-T027。
- **User Story 4 (Phase 6)**：依赖 Foundational、US1 的 handler 和 US2/US3 的最终查询/失效规则；protocol tests 和 stdio tests 可先并行编写。
- **Polish (Phase 7)**：依赖所有目标用户故事完成，且按 T043 -> T044 -> T045 -> T046 -> T047 顺序执行。

### User Story Dependencies

- **US1**：Foundational 完成后即可开始；是 MVP，不依赖 header 或完整 context。
- **US2**：依赖 US1 的 SymbolRecord/query 基础；新增 header 来源和跨文件关系，不改变单文件行为。
- **US3**：依赖 US1 的 snapshot 查询和 Foundational 的 generation/version seam；扩展 partial/stale 生命周期。
- **US4**：依赖 US1 的导航 handler，并吸收 US2/US3 的跨文件和版本结果规则；主要交付 LSP 边界和兼容性。

### Parallel Opportunities

- T001/T002 可并行；T003/T005/T007/T009/T011 可并行准备失败测试。
- US1 的 T013/T014/T015/T016 可并行准备，之后 T017 -> T018 -> T019/T020。
- US2 的 T022/T023/T024 可并行准备，T025 -> T026 -> T027。
- US3 的 T029/T030/T031 可并行准备，T033/T034/T035 按 Store/analysis 依赖执行。
- US4 的 T037/T038/T039 可并行准备，T040/T041/T042 按 handler/contract 依赖执行。
- 不同用户故事不能在 Foundational 完成前合并到 server 共享文件；共享 `internal/server/index_analysis.go` 的任务应按阶段顺序执行，避免同文件并行冲突。

### Parallel Example: User Story 1

```text
T013 testdata/index/single-file fixtures
T014 internal/index/builder_test.go
T015 internal/index/query_test.go
T016 internal/server/navigation_test.go
        -> T017 internal/index/builder.go
        -> T018 internal/index/query.go
        -> T019/T020 server navigation integration
```

### Parallel Example: User Story 2

```text
T022 multi-file/header fixtures
T023 internal/index/source_test.go
T024 internal/server/header_index_test.go
        -> T025 internal/index/source.go
        -> T026 internal/index/builder.go + store.go
        -> T027 internal/server/index_analysis.go
```

### Parallel Example: User Story 3

```text
T029 incomplete/stale fixtures
T030 internal/index/partial_test.go
T031 internal/server/stale_index_test.go
        -> T033 internal/index partial snapshot
        -> T034/T035 server version and invalidation integration
```

### Parallel Example: User Story 4

```text
T037 protocol JSON contract tests
T038 integration/lsp_process_test.go
T039 server error/cancel contract tests
        -> T040/T041 navigation contract implementation
        -> T042 lifecycle compatibility regression tests
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1 Setup 和 Phase 2 Foundational。
2. 完成 Phase 3 US1：单文件函数/类型/属性/builtin 的 index、definition、references 和 document symbols。
3. 执行 T021 的独立验收，确认空结果、作用域、声明回退和 LSP range 正确。
4. MVP 不包含外部 header 全链路，但必须保留后续 SourceProvider 和版本字段的扩展边界。

### Incremental Delivery

1. Setup + Foundational：parser metadata、index snapshot、protocol model 和 server commit seam。
2. US1：单文件导航 MVP。
3. US2：按需 header、多文件导航和依赖失效。
4. US3：partial AST/index、文档变化、stale 和 close/cancel 保护。
5. US4：标准 LSP 契约、真实 stdio、错误/取消/兼容性回归。
6. Polish：完整 fixture、隐私、quickstart、race/vet/full regression。

每个 checkpoint 都必须先运行该用户故事的定向测试，再进入下一个阶段。

## Notes

- `[P]` 只表示任务可以与同阶段的其他任务并行；共享文件或依赖未完成时不要并行实现。
- 用户故事阶段的每个行为变更都有先行测试任务；文档任务只出现在 Polish 阶段。
- Clang 对照、`topsop` corpus 和 `llvm-project` 都是开发机只读输入；不在测试机上编译、配置或运行 workload。
- 任务完成后应保持 `specs/005-ast-symbol-index/quickstart.md`、两个 contract 和 `data-model.md` 与实现一致。

## Phase 8: Convergence

- [X] T048 [HIGH] 补齐 attribute、builtin、field、macro 和 header 来源的 SymbolRecord，完善函数属性来源、类型成员关系及对应索引测试 per FR-003/FR-008/FR-009/FR-010/FR-011 (missing)
- [X] T049 [HIGH] 合并同一声明实体的 declaration/definition 位置，使用完整规范化声明 anchor 保持 symbol_id 稳定并区分重复或重载候选 per FR-004/FR-005/FR-006/FR-009 (partial)
- [X] T050 [HIGH] 按 lexical scope、qualified name、条件状态和 source version 建立引用候选及跨文件引用并集，排除 inactive 引用 per FR-007/FR-019/FR-022 (partial)
- [X] T051 [HIGH] 保存条件 region、branch、parent 和 expression provenance，并统一 unknown、ambiguous、incomplete 的 active/unknown/inactive 查询与 degraded 状态 per FR-013/FR-021/FR-028 (partial)
- [X] T052 [HIGH] 完善按需 header snapshot、相对 include 解析、版本复用、直接和间接依赖失效以及 missing/unreadable/unresolved 诊断 per FR-012/FR-016/FR-017/FR-019/FR-020 (partial)
- [X] T053 [HIGH] 将 index 更新与 diagnostics publisher 解耦，并在 context/header 变化后为仍打开的文档重新排队分析 per FR-016/FR-017/FR-018 (partial)
- [X] T054 [HIGH] 严格校验导航必填参数，增加查询前后 freshness/cancel 检查，并按每个 Location 的 source URI 编码 UTF-16/CRLF range per FR-023/FR-024/FR-025/FR-027/FR-029/FR-030 (partial)
- [X] T055 [HIGH] 统一 analysis-degraded 与 stale_result 的诊断和日志字段，补充 query、partial、unknown、ambiguous、结果数量、耗时、error kind 及可定位 range per FR-031 (partial)
- [X] T056 [HIGH] 扩展 manifest schema 和 validator，使其实际加载 fixture、构建索引并校验 symbol/location/status/diagnostic/version/header 期望 per SC-001/SC-002/SC-003/FR-034/FR-035 (missing)
- [X] T057 [HIGH] 补齐属性/builtin/macro/field、作用域歧义、条件矩阵、跨文件引用、UTF-16/CRLF、取消、stale/null、嵌套 range 和错误路径的 fixture、index/server/protocol/stdio 行为测试 per SC-004/SC-005/FR-034/FR-035 (partial)

## Phase 9: Convergence

- [X] T058 [CRITICAL] 补齐真实 stdio 导航契约矩阵，覆盖跨文件 header、嵌套 document symbols、UTF-16/CRLF、非法参数、取消、stale/null、版本竞争和 header change per SC-004/SC-005/FR-034/FR-035/Constitution IV (partial)
- [X] T059 [HIGH] 补齐 attribute、builtin、field 和 header source 的生产路径，记录参数 token、目标条件、来源位置和可定位 selection range per FR-003/FR-008/FR-010/FR-011 (missing)
- [X] T060 [HIGH] 规范化 declaration anchor，忽略不影响实体身份的参数名并验证声明/定义合并、无关文本移动、函数体变化和重复候选 per FR-004/FR-005/FR-006/FR-009 (partial)
- [X] T061 [HIGH] 建立 lexical block scope 并取消最近作用域对多个 active 候选的静默裁剪，按规则返回全部 active definition/references 候选 per FR-022 (contradicts)
- [X] T062 [HIGH] 保留 unresolved、unknown 和 inactive 引用，过滤 inactive 文本绑定并按 call、type-use、member-use、macro-use 等类别记录引用 per FR-007/FR-021/FR-022 (partial)
- [X] T063 [HIGH] 按每个已合并 source URI 重建引用，保留 header 内部引用及其 URI、范围、候选和版本，形成跨文件引用并集 per FR-019/FR-022 (partial)
- [X] T064 [HIGH] 修正条件分支 provenance，保留嵌套 branch 的父 scope 和独立 expression range，并阻止 inactive include 进入 header 加载 per FR-013/FR-021 (contradicts)
- [X] T065 [HIGH] 完善按需 header 解析和依赖 provenance，解析完整 quoted/angle include、使用 requester 相对路径、规范化 URI 并记录 include range、search roots 和依赖状态 per FR-012/FR-019/FR-020 (partial)
- [X] T066 [HIGH] 统一 header open/change/close、provider availability 和 context invalidation 的 dependent stale/requeue 流程，并解除 index 更新对 diagnostics publisher 的依赖 per FR-016/FR-017/FR-018 (partial)
- [X] T067 [HIGH] 严格校验导航请求的嵌套必填字段并保留类型类别到标准 SymbolKind 映射，正确区分 enum、class、struct、union 和 alias per FR-023/FR-025/FR-027 (partial)
- [X] T068 [HIGH] 为 index diagnostic 保存 source URI 并按来源文本编码范围，统一 partial/degraded/stale 的 analysis_degraded 与 stale_result 字段、版本、耗时和 error kind per FR-031 (partial)
- [X] T069 [HIGH] 扩展 manifest schema 和 validator，实际加载多 source fixture、执行 LSP operation 与版本/依赖变化，并逐项校验 symbol、reference、URI/range、condition、status 和 diagnostic 期望 per SC-001/SC-002/SC-003/FR-034/FR-035 (missing)
- [X] T070 [MEDIUM] 区分 #define、macro use 与普通预处理指令，避免 #if/#ifdef/#include 等 NodePreprocessorDirective 污染 macro SymbolRecord per FR-003 (contradicts)

## Phase 10: Convergence

- [X] T071 [HIGH] 对缺少 `source_files` 的可执行 manifest case 返回明确失败，或为允许的 out-of-scope case 增加显式 skipped 状态，不再把未执行的 case 计入 Passed per FR-034/FR-035/T069 (contradicts)
- [X] T072 [HIGH] 让 manifest validator 校验全部 source URI 和独立 expected oracle，执行标准 LSP operation、精确 URI/range/空结果/null，并用不同文档/header 内容验证版本变化 per SC-001/SC-002/SC-003/T069 (missing)
- [X] T073 [HIGH] 区分 attribute usage、可验证 attribute source 和 unresolved/opaque 状态，禁止把当前文档的 attribute 使用位置登记为来源定义 per FR-010/T059 (contradicts)
- [X] T074 [HIGH] 增加 builtin 的 target/pass/macro/source provenance 和 availability，避免仅按 spelling 分类或在缺少可验证 header 时创建不完整来源 per FR-004/FR-011/T059 (partial)
- [X] T075 [HIGH] 统一 server header 的 requester-relative、include-root 和同名 header ambiguity 解析规则，补充多个 opened header 的 server 集成测试 per FR-012/FR-019/T065 (contradicts)
- [X] T076 [HIGH] 让 references 查询复用 definition 的 active/unknown/inactive 候选过滤，禁止在 inactive symbol name 或 reference 位置返回当前导航结果 per FR-021/T062/T064 (contradicts)
- [X] T077 [HIGH] 补齐 stdio 的精确 range/selectionRange、嵌套 symbol range、三项 malformed 参数、stale null、RequestCancelled 错误码和 includeDeclaration=false 断言 per SC-004/SC-005/FR-023/FR-024/FR-025/FR-027/FR-029/FR-030/FR-034/FR-035/Constitution IV (partial)

## Phase 11: Convergence

- [X] T078 [HIGH] 建立 `source_files` 到 canonical `source_uris` 的逐项映射并校验顺序、重复和错误 URI，补充 source URI mismatch fixture per SC-001/SC-003/T072 (partial)
- [X] T079 [MEDIUM] 让无 root 的 manifest schema 校验也拒绝 execute case 中缺失、null 或空数组的 `source_files`/`source_uris`，保留带 `skip_reason` 的 schema-only 例外 per FR-034/FR-035/T071 (partial)
- [X] T080 [HIGH] 让 manifest 构造并验证标准 LSP request/result oracle，逐项覆盖 params、range、空数组、null、error code 和 `includeDeclaration=false`，禁止无 name/oracle 的 operation 通过 per FR-023/FR-024/FR-025/SC-001/SC-002/T072 (missing)
- [X] T081 [HIGH] 让 header-change 只选择实际 dependency target，验证该 target 的 source version、dependent stale/requeue 和新符号结果，并增加无关 opened header fixture per FR-017/SC-002/T072/T075 (partial)
- [X] T082 [HIGH] 增加 provider-backed attribute source 解析，区分 attribute usage、verified source 和 opaque/unresolved 状态，并验证 usage 不进入 definition per FR-010/T073 (partial)
- [X] T083 [HIGH] 将 builtin source/availability 绑定到可验证 header 与 target/pass/macro context，普通同名变量或未验证 header 保持 unresolved，并补充负例 fixture per FR-011/T074 (contradicts)
- [X] T084 [MEDIUM] 在 query 层显式过滤 inactive ReferenceRecord，补充 inactive reference 位置、同名 active symbol 和 cross-file 场景测试 per FR-021/FR-022/T076 (partial)
- [X] T085 [HIGH] 补齐 stdio 的精确 UTF-16/CRLF range、selectionRange、nested children、fresh empty array、stale null 和 references 全部位置断言 per SC-004/SC-005/FR-023/FR-024/FR-025/FR-029/FR-030/T077/Constitution IV (partial)
- [X] T086 [MEDIUM] 增加 requester-relative、include-root 和多个 opened 同名 header 的组合验证，明确 ambiguity 与 include-root 的优先规则 per FR-012/FR-019/T075 (partial)

## Phase 12: Convergence

- [X] T087 [HIGH] 扩展 ManifestQuery 的 malformed params、expected error code、精确 LSP range 和实际 stale/no-snapshot null oracle，禁止只序列化 nil 结果而不执行对应状态 per T080/FR-023/FR-024/FR-025/FR-027/SC-004 (partial)
- [X] T088 [HIGH] 让 header-change validator 查询实际 dependency target 的新符号 definition/references，断言 header version、dependent requeue 和无关 opened header 不影响结果 per T081/FR-017/SC-002 (partial)
- [X] T089 [HIGH] 让 serverSourceProvider 转发 ResolveAttribute/ResolveBuiltin，并增加 server 导航链路的 verified source、unverified header 和同名普通变量负例 per T082/T083/FR-010/FR-011 (missing)
- [X] T090 [MEDIUM] 在 ReferencesAt 中过滤 ReferenceStatusInactive 和 inactive provenance，增加 active 同名符号、inactive 位置及跨文件 references 输出测试 per T084/FR-021/FR-022 (partial)
- [X] T091 [HIGH] 精确断言 stdio 的全部 references 位置、selectionRange、nested children、definition/references/documentSymbol 空数组和 stale null per T085/SC-004/SC-005/FR-023/FR-024/FR-025/FR-029/FR-030/Constitution IV (partial)

## Phase 13: Convergence

- [X] T092 [HIGH] 按 receiver 类型建立 member-use 到 field 的候选、定义和引用关系，并补充 class/struct field 导航测试 per FR-009/FR-022/T057/T062 (partial)
- [X] T093 [HIGH] 将 attribute usage 建立为可查询的 ReferenceRecord，区分 usage、verified source 和 unresolved/opaque 状态并验证 includeDeclaration 语义 per FR-010/US1/AC3/T073/T082 (partial)
- [X] T094 [HIGH] 增加可注入的 CompilationContext 更新路径，保存宏、target/pass 和 context 版本并使用新 context 重新分析打开文档 per FR-013/FR-016/FR-021/FR-028/plan: context lifecycle (partial)
- [X] T095 [HIGH] 为 partial、incomplete、unknown 和 degraded 快照发布并清理稳定的 analysis-degraded 诊断，保留 source URI/range 并避免 parser diagnostics 覆盖 index diagnostics per FR-020/FR-031/Constitution V (partial)
- [X] T096 [HIGH] 让 manifest LSP queries 通过 server/protocol 标准请求路径执行，并校验 malformed error、fresh/stale null、精确 URI/range/selectionRange 及版本和依赖变化 per SC-001/SC-002/SC-003/SC-004/SC-005/FR-023/FR-030/T069/T072/T080/T087 (partial)
- [X] T097 [HIGH] 扩展 manifest 与独立 expected fixtures，覆盖 attribute、builtin、field、macro、作用域歧义、条件矩阵、include cycle/unreadable/URI update、UTF-16/CRLF、取消、stale/null 和 missing-context，并执行全部必需 oracle per FR-034/FR-035/SC-001/SC-003/SC-006/Constitution IV (missing)
- [X] T098 [HIGH] 使用标准 LSP JSON decoder proxy 完成 stdio 导航契约矩阵，精确断言嵌套 range/selectionRange、全部 references、malformed params、未打开 URI、越界位置、取消、stale null 和 header change per SC-004/SC-005/FR-023/FR-024/FR-025/FR-027/FR-029/FR-030/T058/T077/T085/T091 (partial)
- [X] T099 [HIGH] 在响应发布前原子化重新校验或取消导航请求，防止 didChange/context/header 变化发生在最终 freshness 检查后仍发布旧快照，并增加该竞态测试 per FR-018/FR-029/T054/T077/T091 (partial)

## Phase 14: Convergence

- [X] T100 [HIGH] 让 manifest validator 通过真实 server 的 didOpen/didChange、header change、dependent stale/requeue 和 context version 路径执行 version-race/header-change case，并断言最终版本、fresh/stale null 和新结果 per SC-002/SC-003/FR-016/FR-017/FR-018/T069 (partial)
- [X] T101 [HIGH] 让 definition/references 查询携带 source URI 并只匹配该 URI 的声明、定义和引用，修正 header symbol 在 root 文本上生成查询位置的 manifest oracle per FR-022/FR-023/T072 (partial)
- [X] T102 [HIGH] 让 verified builtin source 进入 definition 和 includeDeclaration=true 的 references 结果，补充 builtin、普通同名变量和 unresolved source 的 server/index 测试 per FR-011/FR-023/T059 (partial)
- [X] T103 [HIGH] 保留 quoted/angle include 形式并统一 requester-relative、include-root、opened header 和同名 header ambiguity 的解析优先级，补充无关 opened header 与多个 root 的测试 per FR-012/FR-019/plan: header provider (partial)
- [X] T104 [HIGH] 建立可复用的 header version snapshot/cache，保存 header scope 和 URI 作用域的 ConditionalRegions，在多个 requester 间复用同一 header 版本并正确失效 per FR-012/FR-013/FR-017/plan: 阶段 3 (missing)
- [X] T105 [HIGH] 将 header dependent stale 与导航最终 freshness 检查放入同一原子边界，覆盖 header 文档已更新但 dependent stale 尚未完成时不得发布旧结果的确定性竞态 per FR-018/FR-029/T099 (partial)
- [X] T106 [HIGH] 让每个 declaration、definition 和 reference location 都可作为 definition/references 查询入口，并补充从声明、定义和调用位置分别导航的测试 per FR-006/FR-023/FR-024 (partial)
- [X] T107 [HIGH] 保证 header、condition 和 partial degraded snapshot 都发布当前文档的稳定 analysis-degraded 诊断，并补齐 source/header version、reason 和 error kind 的日志字段 per FR-031/Constitution V (partial)
- [X] T108 [HIGH] 扩展 stdio 与独立 fixture oracle，精确覆盖 builtin、condition、nested range/selectionRange、全部 references、header change、malformed params、stale/null 和阻塞中的 cancel per SC-004/SC-005/FR-034/FR-035/Constitution IV (partial)

## Phase 15: Convergence

- [X] T109 [HIGH] 让 manifest validator 通过真实 server 生命周期重放 document/context/header 版本序列，执行 didOpen/didChange、dependent stale/requeue 并断言 version-race/header-change 的最终新结果与旧结果丢弃 per SC-002/FR-016/FR-017/FR-018/T100/T108 (partial)
- [X] T110 [HIGH] 为每个执行型 manifest LSP query 补齐独立的 URI、range、selectionRange、嵌套 children、空数组/null 和错误码 oracle，并禁止仅凭数量或范围合法性通过 per SC-001/SC-004/SC-005/FR-034/FR-035/T080/T096 (partial)
- [X] T111 [HIGH] 修正 builtin source 解析，只接受带 header、target/pass 和 macro provenance 的 verified source，普通同名变量或 usage 不得成为 definition/references 来源 per FR-011/T074/T083/T102 (contradicts)
- [X] T112 [HIGH] 接入带 header/context provenance 的 attribute source resolver，区分 attribute usage、verified source 和 opaque/unresolved 状态，并覆盖生产与 fixture 导航路径 per FR-010/T082/T089 (missing)
- [X] T113 [HIGH] 为可执行 tops-lsp server 接通 configured include-root SourceProvider，并增加仅打开主文件即可按需加载可读 header 的 stdio 跨文件导航 case per FR-012/FR-019/FR-032/SC-003/SC-005 (missing)
- [X] T114 [HIGH] 扩展真实 stdio 导航契约矩阵，覆盖 unknown-only、active+unknown、verified builtin、includeDeclaration 和 header 更新后的新定义/引用结果 per FR-021/FR-034/FR-035/SC-004/SC-005/T108 (partial)
- [X] T115 [MEDIUM] 让 serverSourceProvider 返回 opened document 的 document version、content digest 和 URI，并验证 verified attribute/builtin source location 保留完整 SourceVersion per FR-004/FR-010/FR-011/T089/T107 (partial)
- [X] T116 [MEDIUM] 补齐 navigation_query 的 symbol_kind、diagnostic_code 和 error_kind 稳定日志字段，并覆盖 degraded、unknown、ambiguous、空结果和错误查询 per FR-031/T044/T055/T107 (partial)
- [X] T117 [MEDIUM] 为真实 navigation handler 增加可控阻塞取消测试，确认取消只返回 RequestCancelled 且不会发布正常或过期结果 per FR-029/SC-004/T108 (partial)

## Phase 16: Convergence

- [X] T118 [HIGH] 合并 header include 的 parent conditional provenance 到 header symbols、references 和 diagnostics，确保 unknown include 遵循 active/unknown/inactive 导航规则并发布 analysis-degraded per FR-013/FR-021/FR-028/SC-003 (partial)
- [X] T119 [HIGH] 修正候选筛选的 scope 与 condition 顺序，避免 active 候选位于不可见 scope 时错误屏蔽当前 scope 的 unknown 候选，并补 definition/references 回退测试 per FR-022/US1-AC2 (partial)
- [X] T120 [HIGH] 在 header didClose 后清除 server delegate 的 opened source overlay，按 provider 或 include-root 内容重新解析依赖并验证旧 header 内容不会继续发布 per FR-016/FR-017/FR-018/plan: header lifecycle (contradicts)
- [X] T121 [MEDIUM] 对 legacy attribute/builtin resolver 强制校验 trusted header、target/pass、macro 和 context provenance，缺失任一来源证据时保持 unresolved 并补 server 负例 per FR-010/FR-011/FR-031 (partial)
- [X] T122 [MEDIUM] 为 manifest validator 增加真实 definition/references/documentSymbol in-flight 版本竞态，控制旧 parse/query 晚于新版本完成并断言旧响应不会发布 per SC-002/FR-018/T109 (partial)

## Phase 17: Convergence

- [X] T123 [HIGH] 修正嵌套 conditional branch 的 AST/index 重建和状态合并，保留 parent/child provenance 并阻止 unknown 父分支下的 inactive 符号被 active 记录覆盖 per FR-013/FR-021/FR-028/T118 (contradicts)
- [X] T124 [HIGH] 递归失效 header 间接依赖的 snapshot cache 和反向依赖关系，验证 header close/change 后 parent header 与 root 文档不会继续发布旧符号 per FR-016/FR-017/FR-018/T120 (partial)
- [X] T125 [MEDIUM] 让 index diagnostic 保存并发布完整 ConditionalProvenance，包括 parent region、expression range 和 ParentConditions，不只保留 ConditionalState per FR-013/FR-031/T118 (partial)
- [X] T126 [MEDIUM] 统一统计 parse、Store.Replace 和 navigation query 的旧版本结果覆盖或抑制次数，使 old_version_overwrite_count 覆盖全部 version-race 路径 per SC-002/FR-018/T122 (partial)
- [X] T127 [MEDIUM] 为无 fresh snapshot、查询 freshness 失效和 stale/null 响应统一记录稳定 stale_result 事件，并补 observability contract tests per FR-031/T044/T107 (partial)

## Phase 18: Convergence

- [X] T128 [MEDIUM] 统一统计 parse、Store.Replace 和 navigation query 的旧版本结果覆盖或抑制次数，使 old_version_overwrite_count 覆盖所有 version-race 失败路径而不只统计 query stale response per SC-002/FR-018/T126 (partial)
- [X] T129 [MEDIUM] 让无 fresh snapshot、最终 freshness guard 替换 null 和 stale navigation query 统一记录 stale_result 事件，并补齐对应 observability contract tests per FR-031/T127 (partial)

## Phase 19: Convergence

- [X] T130 [HIGH] 为 `FixtureCase` 增加 `context` 字段，支持 language standard、宏、target/pass 和 context version，并贯通 parser/index/server validator，移除固定的 `c++17`、`gcu400` 和 `host` per data-model: FixtureCase/FR-011/FR-013/FR-034/SC-001/SC-003 (missing)
- [X] T131 [HIGH] 取消 manifest validator 和 executor 在 `include_roots` 为空时自动注入 workspace root，保留 `IncludeRoots` unavailable 状态并增加缺少 include root 的 fixture per FR-012/FR-019/FR-020/SC-003/T075/T079 (contradicts)
- [X] T132 [HIGH] 为每个 `ConditionalProvenance` 节点保存实际 `SourceURI`，在 diagnostics 编码时按对应 source text 转换 `expressionRange`，并增加跨文件嵌套条件测试 per FR-013/FR-031/T118/T125 (partial)
- [X] T133 [HIGH] 让 executor-backed manifest 的 definition/references query 强制校验 `source_uri`、`expected_count` 以及每个结果的 URI/range oracle，禁止缺少精确 oracle 的 query 通过 per SC-001/SC-002/SC-003/SC-004/SC-005/FR-034/FR-035/T069/T110 (partial)
- [X] T134 [HIGH] 将 `expected_diagnostics` 扩展为包含 code、source URI、range 和 conditional provenance 的独立 oracle，并在 manifest validator 中校验 diagnostic 范围和来源 per FR-031/FR-034/FR-035/SC-003/T069/T097 (partial)
- [X] T135 [HIGH] 让 `executeManifestVersionRaceQuery` 通过真实 server 重放全部 `document_versions` 和对应文本，覆盖中间旧版本的抑制并报告所有 version-race 路径的旧结果覆盖次数 per SC-002/FR-016/FR-018/T109/T122/T128 (partial)
- [X] T136 [MEDIUM] 统一无 fresh snapshot、navigation stale query 和最终 freshness guard 的 `stale_result` 事件，补充 old/new document、context、header version 与 observability contract tests per FR-031/T127/T129 (partial)

## Phase 20: Convergence

- [X] T137 [HIGH] 分离 stale 结果抑制与旧版本结果覆盖的统计，避免 `observeStaleResult` 将正确丢弃的结果计入 `old_version_overwrite_count`；让 version-race manifest 校验对真实覆盖失败并补充 metrics assertions per SC-002/FR-018/T135/T128 (partial)
- [X] T138 [HIGH] 让最终 request freshness guard 的 `stale_result` 事件携带请求对应 URI、旧/新 document/context/header version，并覆盖 no-fresh、in-flight query 和 final guard 的 observability tests per FR-031/T136/T129 (partial)
- [X] T139 [HIGH] 修正 manifest executor lifecycle regression test 的硬编码 passed 数量，改为与当前可执行 case 统计一致并保持 `go test ./...` 与 `go test -race ./...` 通过 per plan: Phase 7/T046/Constitution IV (partial)

## Phase 21: Convergence

- [X] T140 [MEDIUM] 让 manifest executor lifecycle 回归测试按 manifest 中 schema-only case 动态计算 skipped 数量，同时保持 executable passed/failed 和 overwrite metrics 断言 per data-model: FixtureCase/T139/Constitution IV (partial)
- [X] T141 [MEDIUM] 保留 header invalidation 后 provider/source 的最新版本，并让 in-flight navigation `stale_result` 记录请求 URI、old/new document/context/header version，补充 header-change 竞态观测测试 per FR-031/T136/T138 (partial)
- [X] T142 [MEDIUM] 让 late parse/index `stale_result` 路径复用统一版本字段，不再固定写入 `old_context_version=0` 或空 header 版本，并补 generation/epoch/header invalidation observability test per FR-031/T136/plan: index lifecycle (partial)

## Phase 22: Convergence

- [X] T143 [MEDIUM] 为 parser 返回 document/context version 不匹配的 late result 发布统一 `stale_result` 事件，携带请求 URI、old/new document/context/generation/epoch/header version，并保持 overwrite 与 suppression metrics 分离 per FR-018/FR-031/T142 (partial)
- [X] T144 [MEDIUM] 将分析任务创建时的 header dependency/version provenance 传入 analysis queue stale 观测，避免 queue-drop 路径记录空的 old/new header version，并补齐 generation/epoch/header invalidation contract test per FR-017/FR-018/FR-031/T142/plan: index lifecycle (partial)