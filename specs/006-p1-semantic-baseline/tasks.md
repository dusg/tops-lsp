# Tasks: P1.4 通用 semantic baseline

**Input**: Design documents from `specs/006-p1-semantic-baseline/`

**Prerequisites**: P1.1、P1.2、P1.3；[spec.md](spec.md)、[plan.md](plan.md)、[data-model.md](data-model.md)、[contracts/semantic-boundary.md](contracts/semantic-boundary.md)

**Scope**: 只实现 P1.4 规格列出的通用 semantic baseline。不得实现完整模板语义、完整重载决议、P2 target-specific 规则、Clang runtime 或 TypeScript 语义。

## Phase 1: 规则和测试材料

- [ ] T001 在 `testdata/semantic/manifest.json` 记录规则 ID、输入类别、ContextSeed 状态、预期 diagnostic code/range/severity 和可选对照命令。
- [ ] T002 [P] 为声明、作用域、名称、基础类型、host/device、地址空间和向量规则各添加有效、无效、不完整 fixture。
- [ ] T003 [P] 在 `specs/006-p1-semantic-baseline/` 记录每条规则的 source evidence、限制和未实现状态。

## Phase 2: Semantic model

- [ ] T004 在 `internal/semantic/` 定义 SemanticRule、SemanticDiagnostic、SemanticSnapshot 和版本状态。
- [ ] T005 [P] 为 semantic model 添加状态、稳定排序、版本和降级行为测试。
- [ ] T006 复用 `internal/position` 和 `internal/protocol`，保证诊断 range 使用 UTF-16 半开区间。

## Phase 3: 通用规则

- [ ] T007 实现声明/作用域/名称查找、重复声明、未解析名称和 scope mismatch。
- [ ] T008 实现基础标量、指针、引用、函数参数/返回值和已确认向量的明显类型不匹配。
- [ ] T009 实现已确认 host/device 调用边界、地址空间和基础向量/对齐规则。
- [ ] T010 对 partial/missing/unknown ContextSeed 保留部分结果并生成 `analysis-degraded` 或 `semantic-unsupported`。

## Phase 4: Server/LSP 接入

- [ ] T011 在 `internal/server` 将 parser/index/context snapshot 传给 semantic，并在提交前检查所有版本。
- [ ] T012 在现有 diagnostics publisher 中发布 semantic diagnostics，不新增私有 LSP response 字段。
- [ ] T013 [P] 添加取消、关闭文档、版本竞争和 publisher failure 测试。

## Phase 5: 验证

- [ ] T014 运行 `go test ./internal/semantic ./internal/server ./integration -count=1`。
- [ ] T015 运行 `go test ./...`、`go test -race ./...` 和 `go vet ./...`。
- [ ] T016 在工具可用时执行 Clang 离线对照并保存 known difference；工具不可用时记录 skipped。
- [ ] T017 运行 `git diff --check`，检查日志、fixture 和文档没有完整源码、凭据或未解释占位符。
