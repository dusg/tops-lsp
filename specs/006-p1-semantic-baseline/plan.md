# 实施计划：P1.4 通用 semantic baseline

**分支**：`006-p1-semantic-baseline` | **日期**：2026-10-11 | **规格说明**：[spec.md](spec.md)

**交付边界**：本计划只实现 P1.4 的通用 semantic baseline。P1.4 消费 P1.2 parser、P1.3 index/navigation 和 `ContextSeed`；P2.0/P2.3/P2.4 负责完整 driver context、profile-specific 规则和目标诊断。Clang 不进入 Go server runtime。

## 依赖

- P1.1 提供文档版本、取消、错误和日志边界。
- P1.2 提供可恢复 AST、条件区域、原始范围和 parser diagnostics。
- P1.3 提供声明、作用域、引用、header 来源和 index snapshot。
- `ContextSeed` 提供已知 language standard、include roots、宏状态和 partial/resolved 状态。

## 实施阶段

### 阶段 0：规则和测试材料

- 从 `spec.md` 建立 `SemanticRule` 清单、输入节点、适用 context、diagnostic code 和不承诺项。
- 为声明/作用域、名称、基础类型、host/device、地址空间和向量规则准备有效、无效、不完整和缺少 context 输入。
- 为每条规则记录是否需要 Clang 离线对照；不把对照命令作为 Go 测试前置条件。

### 阶段 1：semantic 数据模型

- 在 `internal/semantic/` 定义 rule、diagnostic、snapshot、version 和 degraded state。
- 复用 `internal/position`、`internal/protocol` 和现有 `LanguageDiagnostic` 编码规则。
- 保持 parser/index 不依赖 semantic package，server 负责组装分析输入和提交最新 snapshot。

### 阶段 2：通用声明与类型规则

- 实现作用域和名称查找结果消费、重复声明、未解析名称和 scope mismatch。
- 实现基础标量、指针、引用、函数参数/返回值和已确认向量的明显不兼容检查。
- 对不完整 AST 保留已确认结果，对无法安全判断的节点生成 `semantic-unsupported` 或 `analysis-degraded`。

### 阶段 3：通用 Tops 规则

- 实现已确认 host/device 调用边界。
- 实现已确认地址空间声明和明显指针空间不匹配。
- 实现基础向量类型/成员/对齐规则；目标资源、硬件布局和跨代语义留给 P2。

### 阶段 4：server 与 LSP diagnostics 接入

- 在分析队列中使用同一 document/context/index version 构造 semantic snapshot。
- 发布前检查 generation、document version、context version 和 index snapshot version。
- 通过既有 diagnostics publisher 发布稳定 code/range/severity；不增加私有 response 字段。

### 阶段 5：对照与收敛

- 运行 Go semantic、server 和 integration tests。
- 在工具和完整 context 可用时运行 Clang `-fsyntax-only` 对照，保存 known difference。
- 复核 Out of Scope 项没有被测试结果或 capability 声明误升级为 supported。

## 交付物

- Go semantic package 和 server 接入代码。
- semantic 单元测试、diagnostics contract 测试和测试材料。
- 稳定 diagnostic code 表、对照记录格式和缺失 context 说明。

## 完成条件

- 每个 P1.4 规则域有正向、负向和不完整测试。
- partial/missing context 不产生未经确认的确定性目标结论。
- 旧 semantic snapshot 不覆盖新文档或 context 结果。
- `go test ./...`、`go test -race ./...` 和 `go vet ./...` 通过；Clang 对照差异均有记录。
