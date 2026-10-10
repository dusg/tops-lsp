# Tasks: Tops C++ Tokenizer 与 Parser

**Input**: Design documents from `specs/004-tops-cpp-tokenizer-parser/`

**Prerequisites**: [plan.md](plan.md)、[spec.md](spec.md)、[research.md](research.md)、[data-model.md](data-model.md)、[contracts/](contracts/)、[quickstart.md](quickstart.md)

**Tests**: parser、LSP diagnostics、server 文档行为和位置映射均为行为变更，必须新增或更新自动化测试。topscc/Clang 参数 provenance 和 Clang 离线对照只作为可选验证，不能阻塞 parser 基础测试。

**Organization**: 任务按 spec 中的用户故事组织；所有任务包含顺序 ID 和具体文件路径。

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 固定 Go 基线、目录结构和测试材料入口。

- [x] T001 [P] 在 `go.mod` 和仓库根目录确认 Go `1.26.8`、`module tops-lsp`、无新增第三方依赖，并记录 `go test ./...` 基线命令。
- [x] T002 [P] 按 [plan.md](plan.md) 创建 `internal/position/`、`internal/parser/`、`testdata/parser/positive/`、`testdata/parser/negative/`、`testdata/parser/incomplete/`、`testdata/parser/macros/`、`testdata/parser/topsop/` 和 `testdata/parser/expected/`。
- [x] T003 [P] 在 `testdata/parser/manifest.json` 定义测试材料 ID、源文件、`driver_kind`、raw/normalized argument provenance、`language_standard`、宏 context、target/pass context、预期 diagnostics 和 Clang 离线对照选项字段。
- [x] T004 在 `internal/parser/fixture_test.go` 建立读取 manifest、定位仓库相对测试材料、脱敏失败输出和 golden 摘要比较的测试辅助入口，依赖 T003。

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 建立所有用户故事依赖的 source range、protocol diagnostics、parser 数据模型和测试边界。

**Checkpoint**: Foundation 完成后，US1 至 US4 可以并行开发；US5 需等待 parser 输出和 protocol publisher 契约稳定。

- [x] T005 [P] 在 `internal/position/mapper_test.go` 先添加 UTF-8 byte、BMP/non-BMP UTF-16、CRLF、多行、EOF zero-width 和非法边界测试。
- [x] T006 实现在 `internal/position/mapper.go` 中统一的 byte offset、line、UTF-16 character、`SourceRange` 转换和边界校验，满足 `contracts/parser.md` 的位置规则，并使 T005 通过。
- [x] T007 在 `internal/document/store.go` 复用 `internal/position/mapper.go`，保持现有版本、原子 change、UTF-16 和 CRLF 行为；更新 `internal/document/*_test.go` 防止已有文档同步回归。
- [x] T008 [P] 在 `internal/protocol/diagnostics_test.go` 先添加 `Diagnostic`、severity、code、source、version、related information、空数组和 notification JSON 的契约测试。
- [x] T009 在 `internal/protocol/diagnostics.go` 实现 `Diagnostic`、`DiagnosticRelatedInformation`、`PublishDiagnosticsParams`、severity 常量和标准 notification 编码，满足 T008，且不改变 `internal/protocol/errors.go` 的既有错误码。
- [x] T010 [P] 在 `internal/parser/ast_test.go` 先添加 `Token`、`SourceRange`、`SyntaxNode`、`RecoveryNode`、`ConditionalRegion`、`ParserDiagnostic` 和 `ParseResult` 的不变量测试。
- [x] T011 在 `internal/parser/token.go`、`internal/parser/ast.go` 和 `internal/parser/diagnostics.go` 定义 parser 数据模型、稳定 diagnostic code、结果状态和版本字段，满足 T010 和 `data-model.md`。
- [x] T012 在 `internal/parser/diagnostics_test.go` 添加 parser diagnostic 到 LSP range 的共享断言，覆盖原文 range、UTF-16 转换、missing token 零宽范围和 related location。
- [x] T013 在 `internal/position/`、`internal/document/`、`internal/protocol/` 和 `internal/parser/` 运行 foundation 检查 `go test ./internal/position ./internal/document ./internal/protocol ./internal/parser/...`，确认公共数据模型和测试辅助 API 冻结。

---

## Phase 3: User Story 1 - 解析标准 C++ 基础语法 (Priority: P1) 🎯 MVP

**Goal**: Go parser 能稳定解析 C++11/14/17 基础 token、声明、语句、表达式和模板，并在标准语法错误后保留后续结构。

**Independent Test**: 仅运行 `internal/parser` 的标准正向/负向测试材料测试，不需要 GCU、Clang 或 server 进程；检查 AST、token、diagnostic code/range 和后续声明。

### Tests for User Story 1

- [x] T014 [P] [US1] 在 `internal/parser/tokenizer_test.go` 和 `testdata/parser/positive/standard_tokens.cpp` 添加标准 C++ 关键字、标识符、数字/字符/字符串/raw literal、注释和运算符的 token snapshot 测试。
- [x] T015 [P] [US1] 在 `internal/parser/parser_declarations_test.go` 和 `testdata/parser/positive/standard_declarations.cpp` 添加 namespace、using、typedef/alias、class/struct/enum/union、函数和变量声明测试。
- [x] T016 [P] [US1] 在 `internal/parser/parser_expressions_test.go` 和 `testdata/parser/positive/standard_expressions.cpp` 添加控制流、初始化列表、lambda、调用、成员、cast、运算符优先级和 C++11/14/17 模板表达式测试。
- [x] T017 [P] [US1] 在 `internal/parser/parser_negative_test.go` 和 `testdata/parser/negative/standard_invalid.cpp` 添加缺分号、缺右括号、非法声明符、错误模板边界和错误表达式的预期 code/range 测试。

### Implementation for User Story 1

- [x] T018 [US1] 在 `internal/parser/tokenizer.go` 实现标准 C++ token 扫描、literal/comment 状态、关键字分类、运算符 longest-match 和 EOF token，满足 T014。
- [x] T019 [US1] 在 `internal/parser/parser.go` 实现 translation unit、namespace/using、声明符、基础类型、函数、class/struct/enum/union、compound/控制流语句解析，满足 T015。
- [x] T020 [US1] 在 `internal/parser/parser.go` 实现 Pratt expression parser、调用/成员/下标/cast、模板参数、lambda、初始化列表和 C++11/14/17 基础语法，满足 T016。
- [x] T021 [US1] 运行 `go test ./internal/parser/... -run 'Test(Standard|Token|Declaration|Expression|Template|Negative)'`，确认 US1 独立测试通过，并记录不属于标准 baseline 的语法为明确 unsupported。

**Checkpoint**: 完成 T021 后，标准 C++ parser 可以作为 MVP 单独交付；不依赖 Tops 目标或 Clang runtime。

---

## Phase 4: User Story 2 - 解析 Tops 关键字、属性和 kernel launch (Priority: P1)

**Goal**: Go parser 能识别 Tops execution/storage qualifier、vector/attribute spelling 和 `<<<...>>>` launch，并保留原始范围。

**Independent Test**: 使用 Tops 正向/负向测试材料和缩减的 `cc_kernel` 片段运行 parser 单测，确认 Tops 节点、属性参数、launch 配置与普通调用参数分离。

### Tests for User Story 2

- [x] T022 [P] [US2] 在 `internal/parser/tops_qualifier_test.go` 和 `testdata/parser/positive/tops_qualifiers.tops` 添加 `__global__`、`__device__`、`__host__ __device__`、storage qualifier、`__vector`、`__fp16`/`__bf16` 和 inline attribute 的 AST/range 测试。
- [x] T023 [P] [US2] 在 `internal/parser/kernel_launch_test.go` 和 `testdata/parser/positive/kernel_launch.tops` 添加完整、模板化、带 shared/stream、尾随逗号和不完整前缀的 `<<<config>>>(args)` 结构测试。
- [x] T024 [P] [US2] 在 `internal/parser/tops_negative_test.go` 和 `testdata/parser/negative/tops_invalid.tops` 添加非法 qualifier 位置、属性空参数/多余逗号、launch 分隔符和未知完整 attribute 的测试。
- [x] T025 [P] [US2] 在 `testdata/parser/topsop/cc_kernel_manifest.json` 添加 `range/range_kernel.tops`、`range/range_host.tops`、`mhc_pre/mhc_pre_n512_kernel_gcu400.tops` 和 `topp_renorm_probs/topp_renorm_probs_kernel_gcu400.tops` 的只读引用或缩减片段条目。

### Implementation for User Story 2

- [x] T026 [US2] 在 `internal/parser/tokenizer.go` 和 `internal/parser/parser.go` 增加 Tops qualifier/attribute 名称分类、平衡参数 token、opaque attribute 和向量类型 spelling 解析，满足 T022/T024。
- [x] T027 [US2] 在 `internal/parser/parser.go` 增加 `<<<config-list>>>(argument-list)` 独立 launch node，区分模板关闭、普通 shift token 和 launch close，满足 T023。
- [x] T028 [US2] 在 `internal/parser/fixture_test.go` 增加 `cc_kernel_manifest.json` corpus 读取和 parser-only 验证，检查 host/device、模板、`#if`、Tops qualifier、vector、attribute、DTE spelling 和 launch 至少五类结构，缺 header 时标记 partial。
- [x] T029 [US2] 运行 `go test ./internal/parser/... -run 'Test(Tops|KernelLaunch|CCKernel|Qualifier|Attribute)'`，确认 Tops positive/negative/corpus 场景独立通过且未修改 `topsop`。

**Checkpoint**: 完成 T029 后，Tops 语法层可以在没有设备执行的情况下独立验收。

---

## Phase 5: User Story 3 - 处理宏条件和目标上下文 (Priority: P1)

**Goal**: parser 保存预处理区域，并依据显式宏 context 产生 `active`、`inactive` 或 `unknown`，不猜测 GCU target。

**Independent Test**: 对同一宏测试材料使用空、已知、冲突和非数字宏 context，检查分支状态、诊断过滤和嵌套恢复。

### Tests for User Story 3

- [x] T030 [P] [US3] 在 `internal/parser/conditions_test.go` 和 `testdata/parser/macros/conditions_positive.cpp` 添加 `#define/#undef`、函数宏、续行、`#if/#ifdef/#ifndef/#elif/#else/#endif`、`defined` 和嵌套条件的正向测试。
- [x] T031 [P] [US3] 在 `internal/parser/conditions_edge_test.go` 和 `testdata/parser/macros/conditions_edge.cpp` 添加空宏 context、unknown/冲突宏、数值比较、inactive 非法代码、缺失/多余 directive 和 token paste/stringification 边界测试。

### Implementation for User Story 3

- [x] T032 [US3] 在 `internal/parser/conditions.go` 实现预处理 directive 行扫描、conditional stack、branch range、父子关系和原始 replacement token 保留，满足 T030。
- [x] T033 [US3] 在 `internal/parser/conditions.go` 实现 `defined`、整数值、`!`、`&&`、`||`、`==`、`!=`、括号和 unknown propagation 的三态条件求值，满足 T031。
- [x] T034 [US3] 在 `internal/parser/tokenizer.go`、`internal/parser/parser.go` 和 `internal/parser/conditions.go` 集成 conditional state：保留 inactive token/region、过滤普通 inactive diagnostics、始终报告 directive stack 错误。
- [x] T035 [US3] 运行 `go test ./internal/parser/... -run 'Test(Condition|Macro|Directive|Inactive|Unknown)'`，确认宏 context 只使用显式值，未定义 target 不触发隐含架构选择。

**Checkpoint**: 完成 T035 后，宏条件和目标边界可以脱离完整 semantic resolver 独立验收。

---

## Phase 6: User Story 4 - 在未完成输入中保持结构并恢复 (Priority: P1)

**Goal**: parser 对 EOF、未闭合 token/结构和前置错误保持可恢复 AST、稳定 diagnostics 和后续声明。

**Independent Test**: 对完整测试材料逐步截断，运行 parser 直到 EOF，确认每个版本返回非空 `ParseResult`、不 panic、后续结构可见且 range 符合契约。

### Tests for User Story 4

- [x] T036 [P] [US4] 在 `internal/parser/incomplete_token_test.go` 和 `testdata/parser/incomplete/literals_and_delimiters.cpp` 添加未闭合 comment、string、char、raw string、括号、花括号、方括号和模板 `<` 截断测试。
- [x] T037 [P] [US4] 在 `internal/parser/recovery_test.go` 和 `testdata/parser/incomplete/recovery_after_error.cpp` 添加属性参数、函数参数、initializer、launch、宏条件错误后继续解析完整函数/宏区域的测试。
- [x] T038 [P] [US4] 在 `internal/parser/diagnostics_test.go` 和 `testdata/parser/incomplete/diagnostic_ranges.cpp` 添加 unexpected token、missing token 零宽、EOF、opening delimiter related information、UTF-16 和 CRLF range golden 测试。

### Implementation for User Story 4

- [x] T039 [US4] 在 `internal/parser/tokenizer.go` 增加未闭合 literal/comment/raw string 状态和 EOF token 标记，满足 T036，并生成 `tops-syntax-unterminated`/`tops-syntax-invalid-literal`。
- [x] T040 [US4] 在 `internal/parser/parser.go` 和 `internal/parser/ast.go` 实现 delimiter/template/attribute/statement/conditional recovery sync point、`ErrorNode`、`MissingToken`、opaque 和 incomplete node，满足 T037。
- [x] T041 [US4] 在 `internal/parser/diagnostics.go` 实现稳定 code/severity、原文 range、EOF zero-width、related location、恢复节点去重和级联错误抑制，满足 T038。
- [x] T042 [US4] 在 `internal/parser/recovery_test.go` 增加全量不完整测试材料的 panic-free、后续声明保留、diagnostic 顺序稳定和重复 code+range 去重测试。
- [x] T043 [US4] 运行 `go test ./internal/parser/... -run 'Test(Incomplete|Recovery|Diagnostic|Range|Panic|Continuation)'`，确认所有截断输入均返回可观察结果。

**Checkpoint**: 完成 T043 后，parser 能处理编辑器中间状态，错误恢复不会清空后续结构。

---

## Phase 7: User Story 5 - 以 Go server 为唯一语义所有者验证结果 (Priority: P2)

**Goal**: Go server 在文档生命周期中发布自有 parser diagnostics，使用 partial context 和版本保护；Clang 只作为可选离线 对照验证。

**Independent Test**: 启动现有 Go server，发送 `initialize`、`didOpen`、`didChange`、`didClose`，检查 `publishDiagnostics` frame、版本和 stdout；另行运行 对照验证 时确认 server 不需要 Clang/clangd。

### Tests for User Story 5

- [x] T044 [P] [US5] 在 `internal/server/diagnostics_test.go` 先添加 publisher 的 JSON frame、parser diagnostic 映射、partial context、空 diagnostics、publisher failure 和无 publisher 的 `Server.Handle` 兼容测试。
- [x] T045 [P] [US5] 在 `integration/lsp_process_test.go` 先添加 `didOpen`/`didChange` 后读取 `textDocument/publishDiagnostics`、继续读取 shutdown response、stdout 纯净性和无 Clang runtime 依赖的进程测试。
- [x] T046 [P] [US5] 在 `internal/server/stale_diagnostics_test.go` 添加旧 document version、旧 context version、generation、非法 change、close 后迟到结果和 publisher 写失败测试。
- [x] T047 [P] [US5] 在 `internal/parser/comparison_test.go` 和 `testdata/parser/expected/clang_comparison.json` 添加可选离线对照清单、skip 条件、接受/拒绝分类、宏/AST/diagnostic 差异记录测试。

### Implementation for User Story 5

- [x] T048 [US5] 在 `internal/server/diagnostics.go` 实现 `ParserDiagnostic` 到 `protocol.Diagnostic` 的映射、source/severity/code/version、related information 和 `publishDiagnostics` frame 写入。
- [x] T049 [US5] 在 `internal/server/parse_context.go` 实现 server 的 partial `ParseContext` 构造：language ID 来自文档、standard/target/pass 为 unknown、宏集合为空、context version 可追踪，不设置隐含 GCU target。
- [x] T050 [US5] 在 `internal/server/server.go` 增加 parser 注入/调用边界：`didOpen` 和成功 `didChange` 使用 `DocumentState` 值快照调用 parser，保留 `Server.Handle` 无 publisher 行为。
- [x] T051 [US5] 在 `internal/server/server.go` 调整 `Run` 的 notification 路径，将现有 `transport.Writer` 作为 publisher 传入；保持 response 与 diagnostics 共用 writer 互斥，并让 `didClose` 停止旧结果。
- [x] T052 [US5] 在 `internal/server/server.go` 和 `internal/server/stale_diagnostics_test.go` 增加 version/context/generation 发布前检查，确保迟到结果被丢弃且非法 change 不触发新解析。
- [x] T053 [US5] 在 `integration/lsp_process_test.go` 更新原有生命周期测试的 frame 读取顺序，并验证合法/非法/不完整 source 的 diagnostics 不影响 shutdown、cancel、EOF 和基础错误行为。
- [x] T054 [US5] 在 `internal/parser/comparison_test.go` 实现可选 Clang 离线对照 runner：使用 manifest 的同一 source/context 执行 `-###`、`-E -dD`、`-dM -E`、`-fsyntax-only` 和适用 AST dump；工具缺失时 skip，不让 server runtime 调用 Clang。
- [x] T055 [US5] 在 `internal/server/diagnostics_test.go` 和 `internal/server/parse_context.go` 验证 parse status、diagnostic count/code、publisher failure 日志脱敏，不记录完整 source、token spelling、payload 或凭据。
- [x] T056 [US5] 在 `internal/server/`、`internal/parser/` 和 `integration/` 运行 `go test ./... -run 'Test.*(Diagnostic|Parse|Stale|Publisher|LSPProcess|Comparison|Logging)'`，确认 Go server 是 parser diagnostics 唯一来源，topscc/Clang 只出现在 context 准备或离线对照路径。

**Checkpoint**: 完成 T056 后，parser、LSP diagnostics、版本保护和离线对照形成可独立复现的 server 功能。

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: 完成全量验证、范围审计和 quickstart 交付检查。

- [x] T057 [P] 在 `specs/004-tops-cpp-tokenizer-parser/quickstart.md` 对照实际命令更新 parser 测试材料、server diagnostics 和 对照验证 验证入口，保持路径和测试名称一致。
- [x] T058 在 `testdata/parser/manifest.json`、`testdata/parser/expected/` 和 `specs/004-tops-cpp-tokenizer-parser/contracts/` 执行测试材料和 contract 一致性检查，确认每个支持域都有正向、负向、不完整或明确不适用记录。
- [x] T059 [P] 在开发机执行 `go test ./...`、`go test -race ./...` 和 `go vet ./...`，并在 `specs/004-tops-cpp-tokenizer-parser/quickstart.md` 记录最终结果；构建/测试不转移到测试机。
- [x] T060 执行 `git diff --check`、`git diff --name-only` 和 external checkout 范围审计，确认只修改 `tops-lsp` 允许路径，不修改 `/home/carl.du/work/llvm-project`、`/home/carl.du/work/topsop` 或 clangd，并确认日志/测试材料无完整源代码和敏感数据。

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**：T001-T004 无代码依赖，可并行完成；T004 依赖 T003 的 manifest 字段。
- **Foundational (Phase 2)**：T005/T008/T010/T012 的测试可在不同文件并行；T006、T009、T011 分别使对应测试通过；T013 阻塞所有用户故事。
- **US1 (Phase 3)**：依赖 T013；T014-T017 先建立失败测试，T018-T020 实现后由 T021 验收。
- **US2 (Phase 4)**：依赖 T013 和 tokenizer/model；T022-T025 可分开准备，T026-T028 实现/接入，T029 验收。
- **US3 (Phase 5)**：依赖 T013 和 tokenizer/model；T030/T031 先建立宏测试，T032-T034 实现后由 T035 验收。
- **US4 (Phase 6)**：依赖 T013 和 tokenizer/model；T036-T038 先建立恢复/range 测试，T039-T042 实现后由 T043 验收。
- **US5 (Phase 7)**：依赖 protocol foundation T009、parser checkpoint T021/T029/T035/T043；T044-T047 先建立跨边界测试，T048-T055 实现，T056 验收。
- **Polish (Phase 8)**：依赖所有目标用户故事；T057-T060 是最终交付门槛。
- **Baseline Convergence (Phase 9)**：T061-T070 是已完成的基础收敛切片；不替代 Phase 10 的剩余质量门槛。
- **Remaining Quality Gates (Phase 10)**：T071-T078 已完成；T071-T073 可并行，T074 依赖 US5，T075 依赖 T067，T076/T077 依赖测试材料清单，T078 依赖 protocol/server diagnostics。
- **Actionable Convergence Subtasks (Phase 11)**：T079-T086 已完成；T079/T081/T083/T084/T085/T086 按文件分工并行，T080 依赖 T072，T082 依赖 T074。
- **最终完成**：Phase 8 基础交付、Phase 10 质量门槛和 Phase 11 可执行子任务均已通过；Clang 未配置或上下文缺失时，记录保留 `not-run`/known difference，不阻塞 Go parser 测试。

### User Story Dependencies

```text
Foundation T013
  ├── US1 T014-T021
  ├── US2 T022-T029
  ├── US3 T030-T035
  └── US4 T036-T043
          \   |   /
           US5 T044-T056
                |
           Polish T057-T060
             |
            Quality Gates T071-T078
              |
         Convergence Subtasks T079-T086
```

US1 是 MVP；US2、US3、US4 可以在 parser 基础模型和 tokenizer 接口稳定后由不同开发者并行，但同一 `internal/parser/parser.go` 的实现任务需要串行合并。US5 必须等待 parser 输出和 diagnostic contract 固定。

### Within Each User Story

- 每个故事先写/更新行为测试和测试材料，再实现对应代码；测试应在缺少实现时失败或明确记录已有基线。
- 数据模型和位置映射先于 parser；parser 先于 server publisher；publisher 先于进程级 LSP 断言。
- 同一文件的实现任务不标记 `[P]`；只有不同文件且无未完成依赖的任务标记 `[P]`。
- 每个 checkpoint 必须先运行该故事的窄测试，再进入下一个依赖故事。

## Parallel Opportunities

### Foundation

```text
T005 position tests       T008 protocol tests       T010 parser model tests
         |                       |                         |
       T006                    T009                      T011
             \___________________ T013 ___________________/
```

### Parser stories after T013

```text
US1: T014-T017 tests -> T018-T020 implementation
US2: T022-T025 tests/data -> T026-T028 implementation
US3: T030-T031 tests -> T032-T034 implementation
US4: T036-T038 tests -> T039-T042 implementation
```

US1 至 US4 的测试准备可以并行；实现阶段需按共享文件冲突和 checkpoint 顺序合并。

### Server/对照验证 story

```text
T044 publisher tests   T045 process tests   T046 stale tests   T047 对照验证 tests
          \_____________________ T048-T055 _____________________/
                                  |
                                 T056
```

Clang 对照验证 tests可以与 server contract tests 并行，但 对照验证 不得成为 parser unit test 的前置条件。

### Convergence Subtasks after Phase 10

```text
T079 grammar test materials  T081 inactive/directive tests
T083 corpus AST assertions  T084 Clang 对照记录
T085 test material consistency T086 LSP metadata tests
              \             |             /
               T080 recovery contract
               T082 context version
```

T079、T081、T083-T086 可在不同文件中并行准备；T080 需先明确 T072 的节点契约，T082 需先明确 T074 的 context 更新契约。每个子任务完成后先运行其所属故事的窄测试，再合并到最终回归。

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1 Setup 和 Phase 2 Foundational。
2. 完成 US1：标准 token、声明/语句、表达式/模板和负向诊断。
3. 执行 T021，只交付 parser-only C++ baseline，不接入 Tops target 语义或 Clang runtime。

### Incremental Delivery

1. 加入 US2，解析 Tops qualifier、attribute、vector spelling 和 kernel launch。
2. 加入 US3，处理宏条件和 unknown context。
3. 加入 US4，处理未完成输入、错误恢复和诊断位置。
4. 加入 US5，接入 `publishDiagnostics`、partial context、stale guard 和可选 Clang 对照验证。
5. 完成 T057-T060 的全量回归和范围审计，再完成 T071-T078 的剩余质量门槛。
6. 按依赖完成 T079-T086 的可执行子任务，并重新运行对应故事的窄测试和最终全量验证；当前已完成。

### Parallel Team Strategy

1. 一名开发者完成 T005-T013，冻结位置、protocol 和 parser model。
2. 基础模型稳定后，开发者 A 负责 US1，开发者 B 负责 US2，开发者 C 负责 US3/US4 测试材料与测试；共享 `parser.go` 的实现合并时按 checkpoint 串行。
3. parser stories 完成后由另一名开发者负责 US5 server publisher、进程测试和 对照验证，避免 parser grammar 与 LSP writer 同时修改同一 server 路径。

## Notes

- `[P]` 只表示不同文件、无未完成依赖的任务可以并行；不表示可以跳过前置 checkpoint。
- `[US1]` 至 `[US5]` 对应 `spec.md` 中的五个用户故事；Setup、Foundational 和 Polish 任务不带故事标签。
- `tasks.md` 不创建 TypeScript 代码、clangd 扩展、设备 workload 或测试机构建任务。
- Clang/llvm-lit 只在 T047/T054、T076 和 T084 的离线 对照验证 路径出现；`clang_comparison.json` 保存 source/context、阶段状态、Go/Clang AST/diagnostic 字段和 known difference。Go parser 和 server 基础测试必须无外部编译器也能运行。

## Phase 9: Baseline Convergence (Completed)

- [x] T061 [US1] 在 `internal/parser/` 建立 C++11/C++14/C++17 核心词法、声明、函数、语句、表达式、模板和标准属性解析切片，并为 `testdata/parser/positive/`、`negative/` 和 `incomplete/` 建立节点、token、诊断 code/range 基线；完整 grammar 扩展由 T071 继续负责。
- [x] T062 [US2] 在 `internal/parser/ast.go`、`internal/parser/parser.go` 和 Tops 测试材料中建立 qualifier、vector/numeric spelling、opaque attribute 及 kernel launch 的 callee/config/argument 基线；相邻关闭符号和更完整 launch grammar 由 T072 继续负责。
- [x] T063 [US3] 在 `internal/parser/conditions.go`、`tokenizer.go` 和相关测试中建立 unknown 条件的基础三态传播、函数宏标记和反斜杠续行处理；inactive lexical filtering 和坏条件诊断由 T073 继续负责。
- [x] T064 [US5] 在 `internal/parser/fixture_test.go`、`internal/server/parse_context.go` 和 `internal/server/` 测试中接入 document/language/driver/context/provenance、宏、target/pass、include 和版本字段的基础传递；context 更新语义由 T074 继续负责。
- [x] T065 [US4] 在 `internal/parser/ast.go`、`parser.go`、`diagnostics.go` 和位置/恢复测试中建立基础 recovery 节点、EOF range、related location 和 20 个范围案例；完整恢复节点统一规则由 T072 继续负责。
- [x] T066 [US5] 在 `internal/server/diagnostics.go`、`server.go` 和 stale 测试中建立同步 document version/generation/open 状态保护、空 diagnostics 和连续编辑基线；迟到结果的完整 context version 校验由 T074 继续负责。
- [x] T067 [US2] 扩展 `testdata/parser/topsop/cc_kernel_manifest.json` 和 `internal/parser/cc_kernel_test.go` 到至少 10 个只读 corpus 条目，并建立 partial context 与结构标签基线；实际 AST 结构断言由 T075 继续负责。
- [x] T068 [US5] 在 `internal/parser/comparison_test.go`、`testdata/parser/manifest.json` 和 `testdata/parser/expected/clang_comparison.json` 建立可选 Clang 语法/预处理对照和基础记录；完整 AST/range/diagnostic 记录由 T076 继续负责。
- [x] T069 [US5] 在 `internal/server/diagnostics.go`、`parse_context.go` 和日志测试中建立基础 severity、状态摘要和 source 脱敏；完整 LSP diagnostic 元数据映射由 T078 继续负责。
- [x] T070 在 `internal/parser/fixture_test.go`、`testdata/parser/manifest.json`、`testdata/parser/expected/`、contracts 和 quickstart 中建立基础一致性检查；全部测试材料登记和精确 golden 校验由 T077 继续负责。

## Phase 10: Remaining Quality Gates

- [x] T071 [US1] 在 `internal/parser/` 为尚未覆盖的 C++11/C++14/C++17 声明、类型、语句、模板和标准属性补充行为测试材料与 language-standard 边界诊断，验收实际节点/token/range，而不是重复已有核心切片。
- [x] T072 [US4] 在 `internal/parser/ast.go`、`parser.go` 和 `diagnostics.go` 统一 ErrorNode/MissingToken/opaque/incomplete 节点构造，补齐模板、属性、参数、launch 和 directive 缺失 close 的 EOF 插入点及 opening delimiter related location。
- [x] T073 [US3] 在 `internal/parser/tokenizer.go`、`conditions.go` 和测试中完成 inactive lexical/syntax diagnostics 过滤，校验未消费的 `#`/`##`、坏条件表达式和嵌套 parent 状态不会猜测结果。
- [x] T074 [US5] 在 `internal/server/parse_context.go`、`diagnostics.go`、`server.go` 和 stale 测试中定义 context 更新触发条件，发布前同时校验 ParseResult 的 document/context version、open 状态和 generation。
- [x] T075 [US2] 在 `internal/parser/cc_kernel_test.go` 中对至少 10 个只读 corpus 条目断言实际解析出的 host/device、`#if`、模板、Tops qualifier、属性、vector/DTE spelling、launch 和 diagnostics，而不是只信 manifest 标签。
- [x] T076 [US5] 在 `internal/parser/comparison_test.go` 和 `testdata/parser/expected/` 为每个需要对照的测试材料保存同一 source/context 下的 preprocess、syntax、AST、range、diagnostic code 和 known difference 记录。
- [x] T077 [US5] 在 `internal/parser/fixture_test.go`、`testdata/parser/manifest.json` 和 `expected/` 登记并校验全部 positive/negative/incomplete/macros/Tops/corpus 文件、路径唯一性、context、节点摘要和精确 diagnostic code/range。
- [x] T078 [US5] 在 `internal/protocol/diagnostics.go`、`internal/server/diagnostics.go` 和测试中补齐 conditional state、recoverability/incomplete、context version 和 related information 的 LSP 映射，并验证 unknown-condition/unsupported/recovery 结果可区分。

## Phase 11: Actionable Convergence Subtasks

**Purpose**: 将 T071-T078 的剩余质量门槛拆成可以单独失败和验收的实现任务。

- [x] T079 [US1] 在 `internal/parser/parser_declarations_test.go`、`parser_expressions_test.go` 和 `testdata/parser/positive/`、`negative/`、`incomplete/` 增加声明符、类型、控制流、模板、标准属性和 C++11/14/17 边界的 expected node/token/range golden，并让 `fixture_test.go` 校验这些结果。
- [x] T080 [US4] 在 `internal/parser/ast.go`、`parser.go`、`diagnostics.go` 和 `diagnostic_ranges_test.go` 统一创建 ErrorNode/MissingToken/opaque/incomplete 节点，逐项断言 EOF 插入点、opening delimiter related range、severity、顺序和重复诊断去重。
- [x] T081 [US3] 在 `internal/parser/conditions_test.go`、`conditions_edge_test.go` 和 `tokenizer_test.go` 增加 inactive 分支非法字符/语法、嵌套 parent、未消费 `#`/`##`、坏条件表达式和函数宏续行的失败用例，校验普通诊断被过滤且 directive 诊断保留。
- [x] T082 [US5] 在 `internal/server/parse_context.go`、`server.go` 和 `stale_diagnostics_test.go` 增加 context 更新入口及 context version 递增测试，验证 ParseResult 的 document/context version、open 状态和 generation 在迟到结果、关闭文档和连续编辑时同时生效。
- [x] T083 [US2] 在 `internal/parser/cc_kernel_test.go` 和 `testdata/parser/topsop/cc_kernel_manifest.json` 对至少 10 条真实 corpus 断言实际 SyntaxNode/diagnostic 内容和 partial 状态，不再只根据 manifest 的 structures 标签通过。
- [x] T084 [US5] 在 `internal/parser/comparison_test.go`、`testdata/parser/manifest.json` 和 `testdata/parser/expected/clang_comparison.json` 固定对照记录 schema，保存命令来源、Go/Clang 状态、AST/range/diagnostic code 和 known difference，并为每个对照测试材料生成或校验记录。
- [x] T085 [US5] 在 `internal/parser/fixture_test.go`、`testdata/parser/manifest.json` 和 `testdata/parser/expected/` 增加目录枚举、路径唯一性、expected node/token/diagnostic range、context provenance 和 compare record 的全量一致性测试。
- [x] T086 [US5] 在 `internal/protocol/diagnostics.go`、`internal/server/diagnostics.go`、`diagnostics_test.go` 和 `lsp_process_test.go` 补齐 conditional state、recoverability、incomplete、context version 和 related information 的 LSP 映射，并验证客户端收到的 unknown-condition/unsupported/recovery 结果可区分。

## Phase 12: Convergence (Completed)

- [x] T087 [US1] 在 `internal/parser/` 与 `testdata/parser/` 补充 C++11、C++14、C++17 分标准的正向/负向/不完整测试材料，覆盖规格 Supported Syntax 中尚未验证的声明、类型、语句、模板、表达式和标准属性；实现并测试 `LanguageStandard` gating，精确校验节点、token、range 以及 `tops-syntax-unsupported`。依赖 T071/T079；验收：每个标准均有可运行测试材料，低标准不静默接受高标准语法，高标准有效结构保持可解析。 per FR-004/FR-013/SC-001 (partial)
- [x] T088 [US4] 在 `internal/parser/ast.go`、`parser.go`、`diagnostics.go`、recovery tests 和 expected goldens 中补齐 ErrorNode、MissingToken、opaque、incomplete 的统一恢复 contract，覆盖 template、attribute、parameter、initializer、statement、launch、directive 的 EOF/中间截断、opening delimiter related range、后续独立声明和级联诊断抑制。依赖 T072/T080；验收：每个恢复场景返回非空 `ParseResult`，节点字段、零宽插入点、related range、诊断顺序和去重结果均有精确断言。 per FR-011/FR-012/FR-018/SC-002 (partial)
- [x] T089 [US5] 在 `internal/server/`、`stale_diagnostics_test.go` 和 `integration/lsp_process_test.go` 固定 context 更新入口与 `contextVersion` 生命周期，增加可控迟到解析结果的测试，覆盖 context invalidation、连续 document change、didClose 和 publisher 前后的 document/context version 校验。依赖 T074/T082；验收：旧 generation 或旧 context result 不得发布，新结果的 `contextVersion` 与 `publishDiagnostics` version 可观测，连续 100 个版本只保留最新结果。 per FR-008/FR-019/SC-006 (partial)
- [x] T090 [US5] 在 `internal/parser/fixture_test.go`、`testdata/parser/manifest.json` 和 `testdata/parser/expected/` 扩展测试材料预期结果格式与一致性校验，保存 conditional state/region、节点层级与关键字段、launch callee/config/argument、diagnostic severity/code/range/recoverability/incomplete/version 以及 context provenance；枚举并校验 `topsop` corpus manifest，避免只用当前实现生成的快照自洽通过。依赖 T079/T085；验收：每个测试材料的 expected 内容可独立审查并精确失败，所有 parser 测试材料和 corpus 条目均有唯一、完整的 context 与 expected 记录。 per FR-022/FR-028/SC-001/SC-009 (partial)
- [x] T091 [US2] 在 `internal/parser/cc_kernel_test.go`、`testdata/parser/topsop/cc_kernel_manifest.json` 和 `testdata/parser/expected/` 为每一条只读 `cc_kernel` corpus 建立实际 SyntaxNode、token、diagnostic 和 partial 状态断言，将 manifest 的 `structures` 映射到源码中真实出现并被 parser 识别的结构，不再以通用 declaration 或标签存在替代验证。依赖 T075/T083；验收：至少 10 条条目分别验证 host/device、`#if`、template、Tops qualifier、attribute、vector/DTE 和 launch 等结构，缺 context 的条目明确保留 partial。 per FR-023/SC-004 (partial)
- [x] T092 [US5] 在 `internal/parser/comparison_test.go`、`testdata/parser/manifest.json` 和 `testdata/parser/expected/clang_comparison.json` 完成可复现 comparison record runner，使用同一 source、language standard、宏、include、target/pass、raw/normalized provenance 执行 preprocess、syntax、AST、range 和 diagnostic 阶段，比较 Go/Clang 结果并写入记录。依赖 T076/T084；验收：本地 Clang 可用时标准测试材料产生真实状态和差异字段，缺少 Tops `.bc` 时记录完整 `context-invalid`/known difference，不得把全部记录保留为 `not-run`，并检查 SC-005 的 95% 分类门槛。 per FR-024/FR-025/SC-005 (partial)
- [x] T093 [US5] 在 `internal/protocol/diagnostics.go`、`internal/server/diagnostics.go`、`internal/server/diagnostics_test.go` 和 `integration/lsp_process_test.go` 增加 publishDiagnostics wire contract，覆盖 unknown-condition、unsupported、recovery、related information、code、severity、source、UTF-16 range、document version、context version 以及 close/stale 结果。依赖 T078/T086；验收：客户端收到的各类诊断 metadata 可区分且与 parser 原始结果一致，旧版本和关闭文档结果不会出现在 wire 输出中。 per FR-019/FR-020/SC-008/SC-009 (partial)

## Phase 13: Convergence

- [ ] T094 [US5] 在 `internal/parser/comparison_test.go` 和 `testdata/parser/manifest.json` 让 Clang 对照 runner 使用同一测试材料的 raw/normalized provenance、include roots、target/pass、language standard 和宏 context，记录实际展开命令并在无法复现时保留 `context-invalid`/known difference per FR-024/FR-025 (partial)
- [ ] T095 [US5] 在 `internal/parser/comparison_test.go` 和 `testdata/parser/expected/clang_comparison.json` 增加接受/拒绝分类一致率统计、差异原因完整性检查和 SC-005 的 95% 门槛，避免只校验 comparison record schema per SC-005/FR-025 (missing)
- [ ] T096 [US1] 在 `internal/parser/tokenizer.go`、`tokenizer_test.go` 和 parser 测试材料中补齐数字、字符、字符串、转义和 `u8R`/`LR`/`UR` 等 raw literal 前缀的识别与非法结构诊断，产生稳定的 `tops-syntax-invalid-literal` 及原文范围 per FR-003/FR-014 (partial)
- [ ] T097 [US2] 在 `internal/parser/parser.go`、`kernel_launch_test.go` 和 Tops 负向测试材料中校验 `callee<<<config-list>>>(argument-list)` 的调用括号、配置边界和不完整恢复，不能把缺少调用括号的 launch 静默接受 per FR-006/PARSER-NEG-LAUNCH (partial)
- [ ] T098 [US3] 在 `internal/parser/conditions.go`、`conditions_edge_test.go` 和宏测试材料中记录 conditional frame 的 `#else` 状态，诊断重复 `#else`、`#elif` 位于 `#else` 之后及非法 directive 参数，并保持 directive range 规则 per FR-007/FR-014 (partial)
- [ ] T099 [US3] 在 `internal/parser/conditions.go` 和条件测试中对 unknown 分支的 `#define/#undef` 做保守宏状态合并，不得把未知分支的副作用直接写入确定宏表，并验证后续条件继续保持 `unknown` per FR-009/US3-AC2 (contradicts)
- [ ] T100 [US4] 在 `internal/parser/parser.go`、`diagnostics.go`、恢复测试和 range golden 中按恢复节点抑制未终止 literal 后附带的 EOF `missing-token` 级联诊断，同时保留恢复后的独立声明和独立错误 per FR-018/US4-AC3 (contradicts)
- [ ] T101 [US1] 在 `internal/parser/parser.go`、AST 测试和标准测试材料中扩展当前分隔符扫描/`kindForRange` 方案为结构化声明、类型、语句、模板和 Pratt expression 解析，精确保存嵌套节点、运算符优先级和后续声明结构 per FR-004/US1-AC1 (partial)
- [ ] T102 [US1] 在 `internal/parser/parser.go`、`standard_gating_test.go` 和标准负向测试材料中补齐 C++11/C++14/C++17 边界及 out-of-scope 特性 gating，至少覆盖 `char8_t` 和同类高标准语法在低标准下的 `tops-syntax-unsupported` per FR-004/SC-001 (partial)
- [ ] T103 [US1] 在 `internal/parser/ast.go`、`parser.go` 和不支持语法测试材料中为可识别但超出 v1 范围的 construct 保存 `opaque` 或等价 unsupported recovery node、最小边界和稳定诊断，而不是只在普通 declaration/statement 上追加诊断 per FR-013 (partial)
