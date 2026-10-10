# Tasks: P0 Go 服务器语义基线

**Input**: Design documents from `specs/002-p0-go-semantic-baseline/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Scope**: 本任务列表只完成 P0 文档、证据、数据模型、契约和验收。不得创建 Go/TypeScript 源码、Go module、parser 依赖、LSP executable、clangd 扩展或运行时 workload。当前设计工件已经存在，因此任务以审查、修正和验证为主。

**Tests**: 本 feature 只修改文档和 Spec Kit 元数据，不改变服务器、客户端、解析器、语义分析或 LSP 行为，因此不新增行为测试。所有文档一致性、topscc argv 证据、配置、契约和 quickstart 检查都作为明确任务；P1 Go 行为测试在后续实现 feature 中执行。

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 确认当前 feature、设计工件和只读 LLVM 证据根可用。

- [X] T001 [P] Verify `.specify/feature.json` points to `specs/002-p0-go-semantic-baseline/` and all prerequisite documents exist.
- [X] T002 [P] Verify the document tree under `specs/002-p0-go-semantic-baseline/`, including `checklists/` and `contracts/`, contains no source-code directories or implementation files.
- [X] T003 [P] Verify the referenced topscc wrapper and LLVM/Tops evidence roots, including `/opt/tops/bin/topscc`, `clang/include/clang/Driver/`, `clang/lib/Basic/Targets/`, `clang/lib/Headers/tops/`, `clang/test/`, and `tops/integration_test/cases/language/`.
- [X] T004 Record the documentation-only boundary and the behavior-test omission rationale in `specs/002-p0-go-semantic-baseline/tasks.md` and keep `llvm-project` read-only for this feature.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 统一规格、计划、证据、数据模型和跨边界契约；本阶段完成前不得关闭任何用户故事。

- [X] T005 Reconcile the mandatory requirements and success criteria in `specs/002-p0-go-semantic-baseline/spec.md` with the phase outputs listed in `specs/002-p0-go-semantic-baseline/plan.md`.
- [X] T006 [P] Audit source and test evidence paths, statuses, limitations, and verification actions in `specs/002-p0-go-semantic-baseline/research.md` against the current `/home/carl.du/work/llvm-project/` checkout.
- [X] T007 [P] Audit the six candidate profile records and the `__GCU_ARCH__`/`__EFGCU_ARCH__` separation in `specs/002-p0-go-semantic-baseline/spec.md` and `specs/002-p0-go-semantic-baseline/research.md`.
- [X] T008 [P] Audit `CompilerInvocation` and `CompilationContext` fields, topscc wrapper defaults, precedence, conflict handling, missing handling, versioning, and stale invalidation in `specs/002-p0-go-semantic-baseline/data-model.md`.
- [X] T009 [P] Audit Go tokenizer/parser/AST/symbol index/semantic/LSP ownership and the TypeScript client/Clang 对照验证 non-ownership in `specs/002-p0-go-semantic-baseline/contracts/lsp-boundary.md`.
- [X] T010 [P] Audit the P0/P1 entry, exit, fallback, compatibility, and owner rules in `specs/002-p0-go-semantic-baseline/contracts/roadmap-gates.md`.

**Checkpoint**: 规格、计划、研究、数据模型和契约之间的范围、状态和 owner 一致，且可以进入用户故事验收。

---

## Phase 3: User Story 1 - 审查八个能力域的语义基线 (Priority: P1) 🎯 MVP

**Goal**: 维护者可以按能力域审查语法清单、源码/测试证据、目标前提、状态、限制和验证方法。

**Independent Test**: 对照 `specs/002-p0-go-semantic-baseline/contracts/capability-matrix.md` 检查 `spec.md` 的八个能力域；每个条目必须具备完整字段，且未将源码观察直接升级为 Go server `supported`。

### Tests for User Story 1

- [X] T011 [P] [US1] Run the eight-domain coverage count and placeholder checks from `specs/002-p0-go-semantic-baseline/quickstart.md` against `spec.md`, `research.md`, `data-model.md`, and `contracts/`.
- [X] T012 [P] [US1] Verify that all source and test paths cited by the eight-domain matrix in `specs/002-p0-go-semantic-baseline/spec.md` exist or are explicitly marked as planned evidence.

### Implementation for User Story 1

- [X] T013 [US1] Complete or correct the standard C++ baseline, C++11/C++14/C++17 coverage, and BUG-2/4/5/6 limitations in `specs/002-p0-go-semantic-baseline/spec.md` using `llvm-project/tops/integration_test/cases/language/STATUS.md`.
- [X] T014 [US1] Complete or correct the execution-space, memory-space, launch/resource, vector/numeric, builtin, TCLE/API, and target-condition rows in `specs/002-p0-go-semantic-baseline/spec.md` using active LLVM/Tops headers and tests.
- [X] T015 [US1] Align field names, status gates, evidence priority, and planned P1 test material rules between `specs/002-p0-go-semantic-baseline/spec.md` and `specs/002-p0-go-semantic-baseline/contracts/capability-matrix.md`.
- [X] T016 [US1] Record unresolved source-only or target-specific items as `candidate`, `target-dependent`, `blocked`, or `unsupported` with a recovery condition in `specs/002-p0-go-semantic-baseline/research.md`.

**Checkpoint**: User Story 1 is independently reviewable as a complete eight-domain evidence and status map.

---

## Phase 4: User Story 2 - 按 CompilationContext 实现 Go 语义层 (Priority: P1)

**Goal**: 后续实现者可以依据统一的 `CompilationContext` 和 Go 层次边界拆分 P1，而不依赖 clangd 或让客户端复制语义判断。

**Independent Test**: 对照 `spec.md`、`data-model.md` 和 `contracts/lsp-boundary.md`，确认每个 context 字段和 Go 层次都有来源、输入、输出、owner、失败行为和验证方法。

### Tests for User Story 2

- [X] T017 [P] [US2] Run document diagnostics and cross-reference checks for `CompilationContext`, `TargetProfile`, `LanguageDiagnostic`, and `LSPCapabilityContract` across `specs/002-p0-go-semantic-baseline/`.
- [X] T018 [P] [US2] Verify the LSP boundary table covers lifecycle, document sync, diagnostics, completion, hover, definition, references, document symbols, configuration changes, cancellation, stale results, privacy, and compatibility.

### Implementation for User Story 2

- [X] T019 [US2] Complete the `CompilerInvocation`/`CompilationContext` field tables and precedence rules in `specs/002-p0-go-semantic-baseline/data-model.md`, including topscc argv normalization, database-first selection, workspace fallback, conflict states, and context version invalidation.
- [X] T020 [US2] Align the tokenizer/parser/AST/symbol index/semantic/LSP responsibilities and non-responsibilities across `specs/002-p0-go-semantic-baseline/spec.md`, `data-model.md`, and `contracts/lsp-boundary.md`.
- [X] T021 [US2] Define or correct the minimum context and LSP error codes in `specs/002-p0-go-semantic-baseline/contracts/lsp-boundary.md` and `data-model.md`, including `missing-compilation-context`, `invalid-compilation-context`, `unsupported-target-feature`, `analysis-degraded`, `request-cancelled`, and `stale-result`.
- [X] T022 [US2] Verify that `specs/002-p0-go-semantic-baseline/plan.md` and `contracts/lsp-boundary.md` explicitly prohibit clangd runtime fallback and TypeScript semantic duplication.

**Checkpoint**: User Story 2 is independently reviewable as a Go-owned semantic and LSP boundary plan without implementation code.

---

## Phase 5: User Story 3 - 为 P1 建立可回归测试入口 (Priority: P2)

**Goal**: P1 可以为八个能力域分别建立有效、无效、不完整和目标边界测试材料，并且每个场景的 context、状态、限制和验证方式明确。

**Independent Test**: 统计 `spec.md` 中的 32 个 P1 场景，并逐项映射到现有证据或明确的 new test material；检查目标边界不会被普通正向测试材料覆盖。

### Tests for User Story 3

- [X] T023 [P] [US3] Verify the 8 × 4 P1 scenario count and IDs in `specs/002-p0-go-semantic-baseline/spec.md`.
- [X] T024 [P] [US3] Verify each P1 scenario records a target prerequisite, status, limitation, verification method, and existing/planned evidence classification.
- [X] T025 [P] [US3] Verify existing test material references and planned test material gaps in `specs/002-p0-go-semantic-baseline/research.md` and `contracts/capability-matrix.md`.

### Implementation for User Story 3

- [X] T026 [US3] Map existing valid test materials for standard C++, execution/memory space, launch/resource, vector/builtin, TCLE/API, and target-condition domains in `specs/002-p0-go-semantic-baseline/spec.md`.
- [X] T027 [US3] Define the planned P1 invalid and incomplete test material groups, including parser recovery, host/device calls, address-space misuse, invalid attributes, vector conversions, builtin access, API overloads, and missing context.
- [X] T028 [US3] Define the planned P1 target-boundary matrix for GCU300/GCU400/GCU410/GCU450/GCU500/EFGCU500, including `-dM -E`, include-root, `-fsyntax-only`, and Clang differential evidence.
- [X] T029 [US3] Align P1 scenario fallback behavior with `contracts/roadmap-gates.md`, preserving `candidate`, `target-dependent`, `blocked`, `unsupported`, `analysis-degraded`, and `stale-result` states.

**Checkpoint**: User Story 3 is independently reviewable as a complete P1 test material and validation entry plan.

---

## Phase 6: Polish & Cross-Cutting Validation

**Purpose**: 完成文档一致性、质量门禁和进入下一阶段前的最终检查。

- [X] T030 [P] Run `get_errors` on `specs/002-p0-go-semantic-baseline/` and resolve relevant diagnostics in the plan artifacts.
- [X] T031 [P] Run the artifact existence, JSON, placeholder, 8/6/32 coverage, and `git diff --check` commands from `specs/002-p0-go-semantic-baseline/quickstart.md`.
- [X] T032 [P] Recheck all referenced source/test paths against `/home/carl.du/work/llvm-project/`; record any missing or changed path as `blocked` or `candidate` instead of silently rewriting it.
- [X] T033 Verify terminology and architecture consistency across `spec.md`, `plan.md`, `research.md`, `data-model.md`, `quickstart.md`, and `contracts/`.
- [X] T034 Verify no Go, TypeScript, clangd, C++ parser, runtime, build, or test-machine workload files were introduced by this feature.
- [X] T035 Mark the task list and quality checklist only after all document gates pass; leave implementation behavior tasks for the subsequent P1 feature.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; verifies the feature pointer, paths, evidence roots, and documentation-only scope.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories until the specification, plan, evidence, data model, and contracts agree.
- **User Story 1 (Phase 3)**: Depends on Foundational; MVP evidence and capability map.
- **User Story 2 (Phase 4)**: Depends on Foundational and the shared capability/context vocabulary; it may be reviewed after the baseline vocabulary is stable.
- **User Story 3 (Phase 5)**: Depends on the capability matrix and context/LSP contracts from US1 and US2.
- **Polish (Phase 6)**: Depends on all required user-story reviews and produces the gate for `/speckit.tasks` completion or the next P1 implementation feature.

### User Story Dependencies

- **User Story 1 (P1)**: Starts after Phase 2; no dependency on P1 implementation code.
- **User Story 2 (P1)**: Uses the capability IDs and target vocabulary from US1; it remains independently reviewable as a design boundary.
- **User Story 3 (P2)**: Uses the evidence and context rules from US1/US2 to define the 32 scenario entry points.

### Parallel Opportunities

- T001-T003 can run in parallel because they inspect different paths and do not modify implementation files.
- T006-T010 can run in parallel after T005 because they review separate design artifacts.
- T011-T012, T017-T018, T023-T025, and T030-T032 can run in parallel when they only read or validate separate artifacts.
- Tasks that modify the same Markdown file, especially `spec.md`, must run sequentially to avoid conflicting edits.

### Within Each User Story

- Complete the review/test tasks before changing the corresponding document.
- Keep evidence corrections local to the owning document and update linked contracts in the same logical change.
- Re-run the narrow validation for the touched artifact before moving to the next story.
- A story is complete only when its independent review condition passes; no story task creates Go/TypeScript behavior.

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 and Phase 2.
2. Complete T011-T016 to make the eight-domain capability map independently reviewable.
3. Stop at the US1 checkpoint and run the quickstart document checks.
4. Only then proceed to the CompilationContext/LSP boundary and P1 scenario tasks.

### Incremental Delivery

1. Stabilize evidence and capability statuses in US1.
2. Stabilize context, semantic ownership, LSP errors, stale handling, and no-clangd boundary in US2.
3. Stabilize the 32 P1 scenario entry points and target matrix in US3.
4. Run Phase 6 and hand the completed task list to the next P1 implementation feature.

### Parallel Team Strategy

1. One owner completes Setup and Foundational because they define shared vocabulary and gates.
2. After Phase 2, reviewers can audit `research.md`, `data-model.md`, and contracts in parallel, but edits to `spec.md` remain serialized.
3. A documentation owner consolidates findings, then the validation owner runs Phase 6 before `/speckit.tasks` is considered complete.

## Notes

- Every task has a sequential ID, an exact path, and a story label where applicable.
- `[P]` is used only when tasks can run concurrently without editing the same incomplete artifact.
- This task list intentionally contains documentation and validation work rather than server/client behavior tests; the reason is recorded in the Tests section and `plan.md`.
- P1 implementation must create its own Go parser/semantic/LSP tests before claiming behavior support.
