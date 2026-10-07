# Tasks: Tops C++ Language Server Roadmap

**Input**: Design documents from `specs/001-tops-lsp-roadmap/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Final publication**: `doc/tops-cpp-language-server-roadmap.md`

**Scope note**: 本任务列表交付 roadmap 文档，不实现 Go server、TypeScript client、parser 或运行时。由于本 feature 是文档型交付，不新增行为测试；所有文档一致性、源码证据、契约和发布路径验证均作为明确任务执行。

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 准备最终 roadmap 文档的发布入口和事实来源。

- [X] T001 [P] Verify the feature pointer and source artifact paths in `.specify/feature.json` and `specs/001-tops-lsp-roadmap/`.
- [X] T002 [P] Verify the Clang/Tops evidence roots referenced by `specs/001-tops-lsp-roadmap/research.md`, including `llvm-project/clang/lib/Headers/tops/`, `llvm-project/clang/include/clang/Basic/`, and `llvm-project/tops/integration_test/cases/language/`.
- [X] T003 Create the final publication document skeleton with title, status, source-artifact links, and publication metadata in `doc/tops-cpp-language-server-roadmap.md`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 建立最终文档的统一结构、架构边界和证据追踪；本阶段完成前不得编写用户故事章节。

**Checkpoint**: Final document foundation ready. User story sections can be written sequentially against the same publication file.

- [X] T004 Add the roadmap purpose, audience, scope boundaries, and source-of-truth rules to `doc/tops-cpp-language-server-roadmap.md` using `specs/001-tops-lsp-roadmap/spec.md` and `specs/001-tops-lsp-roadmap/research.md`.
- [X] T005 Record the fixed architecture boundary in `doc/tops-cpp-language-server-roadmap.md`: Go server owns tokenizer/parser, AST, index, target semantics, diagnostics, and LSP; TypeScript owns VS Code lifecycle; Clang is only a reference and differential oracle; clangd is not extended or used at runtime.
- [X] T006 Add a source-to-section traceability table to `doc/tops-cpp-language-server-roadmap.md` that links each planned section to `specs/001-tops-lsp-roadmap/data-model.md`, `specs/001-tops-lsp-roadmap/contracts/`, and the relevant LLVM/Tops evidence path.
- [X] T007 Run the foundational document checks from `specs/001-tops-lsp-roadmap/quickstart.md` against `doc/tops-cpp-language-server-roadmap.md`, `specs/001-tops-lsp-roadmap/`, and `.specify/feature.json`; resolve missing references before starting user-story work.

---

## Phase 3: User Story 1 - Tops C++ Syntax Capability Map (Priority: P1) 🎯 MVP

**Goal**: 让维护者能够按能力域、目标条件、证据状态和验证动作查阅 Tops C++ 语法范围。

**Independent Test**: Review `doc/tops-cpp-language-server-roadmap.md` against `specs/001-tops-lsp-roadmap/contracts/capability-matrix.md`; every listed capability has a category, spelling, target condition, evidence, status, limitation, and verification action.

### Tests for User Story 1

- [X] T008 [US1] Verify that the syntax inventory in `doc/tops-cpp-language-server-roadmap.md` covers all eight capability domains from `specs/001-tops-lsp-roadmap/spec.md`.
- [X] T009 [US1] Verify positive, negative, target-dependent, blocked, and unsupported status rules against `specs/001-tops-lsp-roadmap/contracts/capability-matrix.md` and `specs/001-tops-lsp-roadmap/data-model.md`.

### Implementation for User Story 1

- [X] T010 [US1] Write the standard C++ baseline, C++11/C++14/C++17 coverage, and known device coverage exceptions in `doc/tops-cpp-language-server-roadmap.md` using `llvm-project/tops/integration_test/cases/language/STATUS.md`.
- [X] T011 [US1] Write the execution-space and storage/address-space capability sections in `doc/tops-cpp-language-server-roadmap.md` using `llvm-project/clang/lib/Headers/tops/__tops_defines.h`, `llvm-project/clang/include/clang/Basic/Attr.td`, and `llvm-project/tops/integration_test/cases/language/exec_spec/exec_space_specifiers/main.cc`.
- [X] T012 [US1] Write the launch/resource, vector/numeric, builtin-variable, TCLE/API, and target-condition sections in `doc/tops-cpp-language-server-roadmap.md` using `llvm-project/clang/lib/Headers/tops/__tops_builtins.h`, `llvm-project/clang/lib/Headers/tops/vector_types.h`, `llvm-project/clang/test/DTU_test/topscc/`, and `llvm-project/tops/integration_test/cases/language/vector_types/`.
- [X] T013 [US1] Add evidence status, known limitations, and minimum verification actions for every syntax section in `doc/tops-cpp-language-server-roadmap.md`; do not promote a source-only observation to `supported`.

**Checkpoint**: User Story 1 is independently reviewable as a complete syntax and evidence map.

---

## Phase 4: User Story 2 - Go Core Language Service Roadmap (Priority: P1)

**Goal**: 明确 Go server 从零实现核心解析、语义和 LSP 能力的交付顺序与验收方式。

**Independent Test**: Review the Go server section in `doc/tops-cpp-language-server-roadmap.md` against `specs/001-tops-lsp-roadmap/data-model.md` and `specs/001-tops-lsp-roadmap/contracts/lsp-boundary.md`; each core capability has an owner, input context, result, error behavior, and test path.

### Tests for User Story 2

- [X] T014 [US2] Verify the Go server capability table in `doc/tops-cpp-language-server-roadmap.md` covers tokenizer/parser, AST, symbol index, diagnostics, completion, hover, definition, references, and document symbols.
- [X] T015 [US2] Verify the planned Go server scenarios include valid input, invalid host/device or address-space usage, incomplete input, missing headers, and stale-result handling from `specs/001-tops-lsp-roadmap/spec.md`.

### Implementation for User Story 2

- [X] T016 [US2] Write the Go-native parsing and semantic architecture roadmap in `doc/tops-cpp-language-server-roadmap.md`, including tokenizer/parser, AST, symbol index, incremental document state, and target-aware semantic ownership.
- [X] T017 [US2] Write the core LSP capability flow in `doc/tops-cpp-language-server-roadmap.md` using `specs/001-tops-lsp-roadmap/contracts/lsp-boundary.md`, including `initialize`, document sync, diagnostics, completion, hover, definition, references, and symbols.
- [X] T018 [US2] Write Go server unit, parser/semantic fixture, LSP contract, integration, and Clang differential validation requirements in `doc/tops-cpp-language-server-roadmap.md`; make clear that Clang results do not replace Go behavior tests.

**Checkpoint**: User Story 2 is independently reviewable as a Go-owned core language-service roadmap.

---

## Phase 5: User Story 3 - Target-Aware Tops Semantics (Priority: P2)

**Goal**: 明确多目标 GCU profile、编译上下文和目标相关诊断的路线。

**Independent Test**: Review the target section in `doc/tops-cpp-language-server-roadmap.md` with multiple `TargetProfile` entries from `specs/001-tops-lsp-roadmap/data-model.md`; each profile has conditions, limitations, positive/negative fixtures, and a fallback state.

### Tests for User Story 3

- [X] T019 [US3] Verify target-profile fields and lifecycle states in `doc/tops-cpp-language-server-roadmap.md` match `TargetProfile` and `CompilationContext` in `specs/001-tops-lsp-roadmap/data-model.md`.
- [X] T020 [US3] Verify the target roadmap includes `.tops`, `.cpp + -Tops`, `-x tops`, compile database precedence, workspace fallback, and missing-context diagnostics from `specs/001-tops-lsp-roadmap/research.md`.

### Implementation for User Story 3

- [X] T021 [US3] Write the target-profile and compilation-context roadmap in `doc/tops-cpp-language-server-roadmap.md`, including target triple, C++ standard, macros, include roots, compiler oracle, and known limitations.
- [X] T022 [US3] Write target-aware diagnostics for architecture-gated attributes, vector alignment, builtin variables, DTE/synchronization declarations, and unsupported features in `doc/tops-cpp-language-server-roadmap.md`.
- [X] T023 [US3] Write the Clang differential oracle and Go-side positive/negative/boundary fixture strategy in `doc/tops-cpp-language-server-roadmap.md`, referencing `llvm-project/clang/test/DTU_test/topscc/` and `llvm-project/tops/integration_test/cases/language/`.

**Checkpoint**: User Story 3 is independently reviewable as a target-aware semantic roadmap without relying on clangd.

---

## Phase 6: User Story 4 - VS Code Client Workflow (Priority: P2)

**Goal**: 明确 TypeScript 客户端如何连接 Go server、传递配置、展示结果和恢复错误。

**Independent Test**: Review the client section in `doc/tops-cpp-language-server-roadmap.md` against `specs/001-tops-lsp-roadmap/contracts/lsp-boundary.md`; every editor action has a client owner, LSP message, server result, visible result, and failure path.

### Tests for User Story 4

- [X] T024 [US4] Verify initialization, document synchronization, configuration change, restart, shutdown, reconnection, stale diagnostics, and single-root/multi-root scenarios in `doc/tops-cpp-language-server-roadmap.md`.
- [X] T025 [US4] Verify client/server ownership, payload, error behavior, logging, privacy, and compatibility statements against `specs/001-tops-lsp-roadmap/contracts/lsp-boundary.md`.

### Implementation for User Story 4

- [X] T026 [US4] Write the TypeScript VS Code activation, Go server lifecycle, workspace configuration, status, restart, and command roadmap in `doc/tops-cpp-language-server-roadmap.md`.
- [X] T027 [US4] Write the compilation-context settings contract in `doc/tops-cpp-language-server-roadmap.md`, covering compile database, Go server path, Clang oracle path, target profile, include roots, macros, and precedence.
- [X] T028 [US4] Write client observability, privacy, cancellation, stale-result, compatibility, and migration requirements in `doc/tops-cpp-language-server-roadmap.md`.

**Checkpoint**: User Story 4 is independently reviewable as a complete VS Code client and LSP boundary roadmap.

---

## Phase 7: User Story 5 - Roadmap Milestones and Evolution Gates (Priority: P3)

**Goal**: 将语法能力、Go server、VS Code client、测试、风险和最终发布组织为可追踪的 P0-P4 交付路线。

**Independent Test**: Review the milestone table in `doc/tops-cpp-language-server-roadmap.md` against `specs/001-tops-lsp-roadmap/contracts/roadmap-gates.md`; every phase has entry evidence, scope, dependencies, validation, exit criteria, fallback, compatibility, and owner.

### Tests for User Story 5

- [X] T029 [US5] Verify P0-P4 phase order, dependencies, exit criteria, fallback states, and owners in `doc/tops-cpp-language-server-roadmap.md` against `specs/001-tops-lsp-roadmap/contracts/roadmap-gates.md`.
- [X] T030 [US5] Verify all measurable outcomes SC-001 through SC-009 from `specs/001-tops-lsp-roadmap/spec.md` are represented by a milestone, validation step, or publication gate in `doc/tops-cpp-language-server-roadmap.md`.

### Implementation for User Story 5

- [X] T031 [US5] Write the P0-P4 roadmap milestone table, dependency graph, and incremental delivery strategy in `doc/tops-cpp-language-server-roadmap.md`.
- [X] T032 [US5] Write the performance target, test matrix, observability requirements, compatibility policy, risks, and ownership model in `doc/tops-cpp-language-server-roadmap.md`.
- [X] T033 [US5] Add the final publication rule and source-artifact index to `doc/tops-cpp-language-server-roadmap.md`, requiring consistency with `specs/001-tops-lsp-roadmap/` before publication.

**Checkpoint**: User Story 5 is independently reviewable as a complete, gated roadmap document.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: 统一语言、链接、证据状态和最终发布质量。

- [X] T034 [P] Check all source links and evidence paths in `doc/tops-cpp-language-server-roadmap.md` against `specs/001-tops-lsp-roadmap/research.md` and the current `llvm-project` checkout.
- [X] T035 [P] Check terminology and architecture consistency across `specs/001-tops-lsp-roadmap/spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/`, and `doc/tops-cpp-language-server-roadmap.md`.
- [X] T036 Run the complete validation commands in `specs/001-tops-lsp-roadmap/quickstart.md`, including the final `doc/tops-cpp-language-server-roadmap.md` existence and content check.
- [X] T037 Verify the final document has no unresolved placeholders, unsupported claims, missing source references, or contradictions with the Go-from-scratch/no-clangd decision in `doc/tops-cpp-language-server-roadmap.md`.
- [X] T038 Mark the final `RoadmapDocument` as reviewed/published in `doc/tops-cpp-language-server-roadmap.md` only after all previous tasks pass, and record the verification date.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001-T003 establish the final document path and evidence roots.
- **Foundational (Phase 2)**: T004-T007 depends on Setup and blocks all user-story writing.
- **User Story 1 (Phase 3)**: T008-T013 depends on the foundational document structure and is the MVP content slice.
- **User Story 2 (Phase 4)**: T014-T018 depends on US1's syntax map and the LSP boundary contract.
- **User Story 3 (Phase 5)**: T019-T023 depends on US1 and US2's capability and context definitions.
- **User Story 4 (Phase 6)**: T024-T028 depends on US2's LSP flow; it can be reviewed independently after the core contract is stable.
- **User Story 5 (Phase 7)**: T029-T033 depends on US1-US4 so the phase gates cover the full roadmap.
- **Polish (Phase 8)**: T034-T038 depends on all required user stories and produces the publishable document.

### User Story Dependencies

- **User Story 1 (P1)**: Depends only on the Foundational phase; MVP.
- **User Story 2 (P1)**: Depends on US1's syntax categories and the shared LSP boundary; independently reviewable after those inputs exist.
- **User Story 3 (P2)**: Depends on US1's target evidence and US2's Go semantic ownership.
- **User Story 4 (P2)**: Depends on US2's LSP contract; independent of the detailed target matrix after compilation-context fields are defined.
- **User Story 5 (P3)**: Depends on the completion of US1-US4 and consolidates their acceptance criteria.

### Parallel Opportunities

- T001 and T002 can run in parallel because they inspect different source sets and do not edit the final document.
- T034 and T035 can run in parallel after all story sections are complete because one checks links/evidence and the other checks terminology/architecture consistency.
- Story content tasks must remain sequential when they edit `doc/tops-cpp-language-server-roadmap.md`; do not run concurrent writers against that file.

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Setup and Foundational phases.
2. Complete US1 to produce the syntax capability map with evidence and status rules.
3. Run T008-T013 independently and review the draft in `doc/tops-cpp-language-server-roadmap.md`.
4. Stop at the US1 checkpoint to validate the roadmap foundation before adding Go server and client sections.

### Incremental Delivery

1. Add US2 to define the Go-owned core language-service roadmap.
2. Add US3 to define target profiles and differential validation.
3. Add US4 to define the TypeScript client and LSP workflow.
4. Add US5 to consolidate P0-P4 gates, metrics, risks, and publication rules.
5. Run Polish and publish only after `doc/tops-cpp-language-server-roadmap.md` is consistent with all `specs/001-tops-lsp-roadmap/` sources.

### Parallel Team Strategy

1. One owner completes Setup and Foundational phases because they define the shared document structure.
2. After US1, reviewers can independently prepare US2, US3, and US4 content outlines, but only one writer should merge changes into `doc/tops-cpp-language-server-roadmap.md` at a time.
3. US5 and Polish run after the merged roadmap sections are stable.

## Notes

- Every task uses the required checklist format with a sequential ID and an exact file path.
- `[P]` appears only on tasks that can run concurrently without editing the same incomplete section.
- This feature deliberately has documentation validation tasks instead of Go/TypeScript behavior tests; actual server/client tests belong to the later implementation tasks generated after this roadmap is accepted.
- The final document path is fixed and must not be moved back under `specs/`.
