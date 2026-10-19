# 实施计划：Tops C++ 通用 Semantic Baseline

**分支**：`006-semantic-baseline` | **日期**：2026-10-13 | **规格说明**：[spec.md](spec.md)

**输入**：来自 `specs/006-semantic-baseline/spec.md` 的功能规格说明。

## 摘要

在现有 parser、symbol index 和 LSP diagnostics pipeline 上增加 Go server semantic 层。semantic 层消费已解析 AST/index 和 `SemanticContext`，提供声明/作用域/名称查找、基础 scalar/pointer/reference 类型、函数参数/返回值、GCU400 host/device、address-space、vector 约束和稳定 `tops-semantic-*` diagnostics。

输入编译命令缺失或没有有效 Tops 编译命令时，对 Tops 文档生成合成 `CompilationCommand`：`arch=gcu400`、`driver=topscc`、`language=TOPS`、`language_standard=c++17`、`pass_kind=host+device`，使用工具链默认 headers/内建宏和 workspace 工程 headers/宏，并生成带 fallback provenance 的 `gcu400-default`；不得返回空 command 或 no-profile。完整默认命令进入 `resolved`，所需来源缺失进入 `partial`，真实 target 冲突进入 `invalid`。

实现不增加 hover、completion 或私有 semantic LSP，不在 Go server runtime 启动 Clang/clangd。Clang 只复用现有离线对照测试入口。

## 技术上下文

**语言/版本**：Go `1.26.8`，沿用当前 module；不新增语言运行时。

**主要依赖**：现有 `internal/parser`、`internal/index`、`internal/server`、`internal/protocol`、`internal/position`；无新增第三方依赖。Clang 仅作为离线测试工具。

**存储**：现有内存 `document.Store`、`index.Store`；新增按 URI/version 管理的 semantic snapshot/store，不引入数据库或 workspace 全量索引。

**测试**：`go test ./...`、定向 Go package tests、`go test -race ./...`、`go vet ./...`、`go build ./...`、`go test ./integration`、现有 `tops-index-check`、新增 `tops-semantic-check` manifest validator，以及可选的 `TOPS_LSP_CLANG` 离线对照。

**目标平台**：Linux development machine；unit/contract/integration tests 不需要 GCU、GCUSIM、clangd 或测试机。

**项目类型**：Go language server/compiler-adjacent service。

**性能目标**：本功能不新增未经规格确认的吞吐数字；保持现有单文档增量分析模型，禁止 workspace 全量扫描。实现阶段记录定向测试耗时，性能优化不阻塞 semantic correctness。

**约束**：标准 LSP diagnostics；`Diagnostic.data` 向后兼容；默认 profile 永远为非空；no-profile 分支删除；target 规则按 `SemanticContext` 状态和 pass 视图门控；日志不得记录完整源码；Clang 不进入 runtime。

**规模/范围**：当前打开文档、workspace 来源提供的工程 headers、工具链默认标准库/Tops headers；按需加载，不扫描未引用文件。v1 不实现模板实例化、完整 overload resolution、完整 constant evaluation、完整 access control、链接或 code generation。

## 宪法检查

*门禁：阶段 0 研究前通过，阶段 1 设计后再次通过。*

- [x] **LSP 契约优先**：`contracts/lsp-semantic-diagnostics.md` 固定标准 `publishDiagnostics`、`Diagnostic.data`、range、severity、pass 和 stale 行为；不增加私有 semantic LSP。
- [x] **Go/TypeScript 分层边界**：Go server 负责 SemanticContext、profile、semantic checker、diagnostics 和版本；TypeScript 不复制语义判断。
- [x] **Tops C/C++ 语义保真**：默认 Tops 命令、GCU400 profile、host/device 双 pass、headers/macros provenance 和 target 冲突状态均写入 `research.md`、`data-model.md` 和契约。
- [x] **行为优先的验证**：计划包含 semantic unit、protocol、server stale、组合 pass、默认命令、header/macro provenance、integration 和离线对照测试。
- [x] **可观测且兼容地演进**：稳定 `tops-semantic-*` code、标准 LSP 输出、`stale_result` 观测、pass/profile/context data 和无源码日志约束已定义。
- [x] **无未记录例外**：没有需要复杂度例外的宪法违规项。

## 项目结构

### 文档（本功能）

```text
specs/006-semantic-baseline/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── semantic-context.md
    ├── lsp-semantic-diagnostics.md
    └── clang-semantic-comparison.md
```

### 源代码（仓库根目录）

```text
cmd/tops-semantic-check/
├── main.go                 # semantic fixture manifest validator
└── main_test.go            # validator success/failure exit contract

internal/semantic/
├── command.go              # CompilationCommand、默认命令常量和 command field schema
├── canonical.go            # FieldValue、canonical JSON、ConflictSource.instance 和字段合并辅助函数
├── source_version.go       # SourceVersionEntry 与 AggregateSourceVersion 共享 helper
├── context.go              # SemanticContext、profile、状态和版本
├── profile.go              # SemanticProfile 与 profile rules
├── resolver.go             # 当前文档 command candidate、语言、workspace candidate/fallback、profile/context 来源解析和字段冲突
├── source_cache.go         # WorkspaceSourceCache 与 header/macro 失效
├── store.go                # semantic snapshot 版本存储
├── model.go                # semantic facts、types、callables、diagnostics
├── lookup.go               # scope/index candidate lookup
├── types.go                # scalar/pointer/reference 与转换
├── callable.go             # 参数/返回值和 callable facts
├── call.go                 # CallFact 与调用点候选/转换
├── checker.go              # 基础 semantic checks
├── call_boundary.go        # host/device pass checks
├── target_rules.go         # address-space/vector profile rules
├── diagnostics.go          # stable code、severity、range 和去重
├── comparison.go           # semantic offline comparison records
└── *_test.go

internal/server/
├── server.go               # CompilationContext 到 SemanticContext 的适配字段
├── parse_context.go         # 默认命令和 context version 生成入口
├── analysis.go              # semantic 分析排队、freshness 和提交
├── index_analysis.go        # index/semantic snapshot 的版本联动
├── semantic_analysis.go     # semantic snapshot 构建和提交
├── diagnostics.go           # semantic diagnostic 到 LSP 的映射
├── observability.go         # semantic/stale 事件和必要字段
└── *_test.go

internal/protocol/
├── diagnostics.go           # Diagnostic.data 的 profile/context/pass 字段
└── diagnostics_test.go

testdata/semantic/
├── manifest.json
├── comparison_manifest.json
├── declarations/
├── types/
├── callables/
├── calls/
│   ├── basic/
│   └── host-device/
├── address-space/
├── vectors/
├── out-of-scope/
├── default-context/
├── conditions/
├── incomplete/
├── stale/
└── expected/
    └── semantic_comparison.json

internal/parser/
├── comparison_test.go       # 复用现有 Clang offline runner
└── comparison_contract_test.go
integration/
└── lsp_process_test.go      # 标准 publishDiagnostics/stale/取消契约

internal/index/
├── source.go                # 导出 NormalizeSourceURI
└── source_test.go           # URI 规范化固定向量
```

**结构决策**：semantic 规则和数据模型放在新 `internal/semantic`，不把语义判断塞入 parser/index；server 负责生命周期、版本和发布；protocol 只扩展标准 diagnostics data；测试材料单独放在 `testdata/semantic`，复用现有 parser/index fixture 作为输入。User Story 的“独立测试”表示在 foundational 及该 story 明确列出的上游模型完成后可单独运行，不要求完全脱离共享模型。

## 实施阶段

### Phase 0：研究与基线（已完成，非实现阶段）

1. 固定 `CompilationCommand`、`SemanticProfile`、`SemanticContext` 状态和默认 GCU400 命令字段。
2. 核对工具链默认标准库/Tops header roots、内建宏和 workspace source provider 的来源边界；不存在来源时保留 unresolved/degraded。
3. 保存现有 parser/index/server/protocol 测试作为回归基线；不改变既有 definition/references/documentSymbol 结果。
4. 固定离线 Clang 对照入口为 `TOPS_LSP_CLANG`；comparison 输入为 `testdata/semantic/comparison_manifest.json`，输出为 `testdata/semantic/expected/semantic_comparison.json`，fixture 注册仍使用 `testdata/semantic/manifest.json`，不假设固定可执行文件路径。

本阶段的交付物是本目录中的规格、研究记录、数据模型、contracts 和 quickstart；它不新增实现任务 ID。后续 T005、T043、T058 和 T065 分别对相关边界执行测试或最终回归，不改变 Phase 0 已确定的规则。

### Phase 1：模型与上下文

1. 在 `internal/semantic` 定义 `CompilationCommand`、`SemanticProfile`、`SemanticContext`、状态、pass views、provenance 和版本字段。
2. 实现 Tops 文档默认命令 resolver：`gcu400`/`topscc`/`TOPS`/`c++17`/`host+device`，工具链默认 roots/macros 加 workspace roots/macros。
3. 实现真实 command、默认 command、target conflict 的状态转换；resolver 边界保证非空 profile，删除后续 no-profile 分支。
4. 增加 resolver unit tests：Tops/普通 C++ 文件范围、输入命令缺失时使用默认 command、header/macro 来源、冲突 target、版本递增和 fallback provenance。

### Phase 2：Semantic facts 与 checker

1. 从 AST/index 构建 declaration、scope、type、callable、reference 和 conditional facts；复用已有 source ranges 和 candidate statuses。
2. 实现基础 scalar/pointer/reference 类型表示、标准标量隐式转换和可直接判断的指针/引用转换；明确模板/用户转换/完整 overload 的 degraded 状态。
3. 实现 host/device 两个 pass view 的调用边界检查；未标注 Tops 函数在 resolved context 下按 host；组合 pass 不合并两侧事实。
4. 实现 profile 驱动的 address-space 和 vector element/lane/width/alignment 规则；target conflict/partial/missing 按状态抑制确定性 Error。
5. 实现 stable diagnostic code、severity、最小源码 range、UTF-16 映射、related information、pass/profile/context data 和 pass-aware dedup。
6. 增加 semantic unit/fixture tests，覆盖 valid/invalid/incomplete/unknown-condition/default-GCU400/host-device/address/vector/standard conversion。

### Phase 3：Server pipeline、LSP 和 stale

1. 在 `parseSnapshot` 的 parse -> index 之后接入 semantic analysis，提交前后复用 document/context/header/generation/epoch freshness checks。
2. 让 semantic snapshot 与 index snapshot 使用同一提交/失效边界；header/context/document 变化时 semantic 结果一并 stale。
3. 扩展 `internal/protocol/diagnostics.go` 的 `DiagnosticData`，使用 `profileId`、`contextStatus`、`passKind`、semantic kind 和版本字段，保持 `omitempty` 兼容。
4. 修改 server diagnostics mapping，不再把 semantic diagnostics 全部固定为 Information；parser/index diagnostics 既有行为保持不变。
5. 记录 `stale_result`、semantic operation、旧/新版本、profile/pass/context status，不记录完整 source text。
6. 扩展 server/protocol/integration tests：publishDiagnostics JSON、组合 pass、取消、参数错误、旧结果丢弃、Unicode/CRLF/zero-width range。

### Phase 4：离线对照与全量验证

本阶段的 comparison schema/runner 单元测试可以提前准备；使用真实 semantic code/severity/range 的最终对照验收必须在 US1、US2、US3 完成后执行。

1. 复用现有 parser comparison runner，但 semantic runner 只读取 `testdata/semantic/comparison_manifest.json`，输出 `testdata/semantic/expected/semantic_comparison.json`，并携带相同 `document_uri`、派生的 `root_source_uri`、source hash、arguments、target/pass、include/macro provenance；`root_source_uri` 必须等于 `NormalizeSourceURI(document_uri)` 且出现在 source files 中。
2. 输入 provenance 不一致时生成 `not-comparable`，不把 Clang message 或 code 进入用户诊断。
3. 执行 quickstart 中的定向测试、全量测试、race、vet、build、integration 和 manifest validator。
4. 复核旧 navigation/index/parser tests、默认普通 C++ 文件行为、no-profile 分支计数和无源码日志约束。

## 关键技术决策

### SemanticContext 提交边界

semantic worker 在开始时复制完整版本集合。只有 document、SemanticContext、source/header、generation 和 epoch 全部匹配时，才把 snapshot 写入 store 并发布 diagnostics。任一不匹配只记录 stale，不发布旧结果。

### 默认 GCU400 语义

无有效 Tops compile command 时，完整默认命令产生 `resolved` `SemanticContext`，允许按 GCU400 profile 检查；`fallback provenance` 仍在 context、diagnostics data 和 comparison record 中保留。多个 target 冲突时使用结构化 `gcu400-default`，但保持 `invalid`，不选择任一冲突目标。

### 诊断与 LSP 兼容

所有 v1 用户可见输出使用标准 `textDocument/publishDiagnostics`。新增 data 字段使用 `omitempty`，旧 client 可忽略；不新增 hover/completion/private semantic message。host/device 诊断按 pass 分开，range/related information 均来自原始 source。

### 外部数据流

工具链默认 headers/macros 从配置或可审计的 topscc resource/include roots 读取；工程 headers/macros 从 workspace source provider/settings 按需读取；不扫描整个 workspace。Clang 只在离线测试进程中启动，server runtime 不读取其结果。

## 设计后宪法复核

- [x] LSP contract 已写入 `contracts/lsp-semantic-diagnostics.md`，无私有 semantic LSP。
- [x] Go server 拥有 SemanticContext、checker、diagnostics 和 stale；TypeScript 不拥有语义决策。
- [x] GCU400 fallback、host+device、headers/macros provenance 和 target conflict 均有 data model、contract 和测试计划。
- [x] 正常、错误、不完整、冲突、缺失来源、取消、Unicode/CRLF 和版本竞争均有测试阶段。
- [x] 诊断 code/severity/range/pass/profile/context、stale logging 和 Clang data flow 已定义。
- [x] 默认 `Diagnostic.data` 扩展保持向后兼容；没有需要复杂度例外的宪法违规。

## 复杂度记录

无需要记录的宪法违规项。新增 `internal/semantic` 是为了隔离语义所有权，属于本功能范围内的必要边界；更简单的做法是把规则加入 parser/index，但会违反分层和测试隔离要求。
