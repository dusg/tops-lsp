# 研究记录：Tops C++ 通用 Semantic Baseline

**日期**：2026-10-13
**范围**：为 `006-semantic-baseline` 解决实现边界、上下文来源、LSP 诊断和验证入口问题。

## 决策 1：复用现有 parser/index/server pipeline

- **结论**：semantic 层消费现有 parser `ParseResult`、`SyntaxNode`、条件 provenance、`IndexSnapshot`、`ScopeRecord`、`SymbolRecord` 和 `ReferenceRecord`；不先重写 tokenizer、parser 或 symbol index。
- **依据**：`internal/parser/ast.go` 已保存 source range、声明元数据、类型/表达式 token 和版本；`internal/index/model.go` 已保存声明、定义、引用、scope、参数摘要、返回类型摘要、执行限定符和 source version；`internal/server/parse_context.go` 已串联 parse、index、stale 检查。
- **替代方案**：重建独立语法树或引入 Clang runtime。拒绝，因为会重复现有 AST/index 并违反 Go server 与 Clang runtime 边界。

## 决策 2：SemanticContext 与 IndexSnapshot 使用相同版本门控

- **结论**：新增 semantic snapshot/store，但提交前后都校验 document version、semantic context version、header/source versions、analysis generation 和 epoch；semantic 结果不能在旧 index 仍可见时单独发布。
- **依据**：`internal/server/parse_context.go`、`internal/server/index_analysis.go` 和 `internal/server/navigation.go` 已有 generation/epoch/context/document freshness 检查；`internal/index/store.go` 已有 header dependent stale 失效。
- **替代方案**：把 semantic 结果放在独立、无版本校验的全局缓存中。拒绝，因为会重新引入旧结果覆盖和 header 失效不同步。

## 决策 3：输入命令缺失或无有效编译命令时生成 GCU400 默认命令

- **结论**：仅对 `.tops`、`languageId=tops` 或明确 Tops 模式命中的 Tops-signaled document，在输入编译命令缺失或当前文档规范化 URI 对应的命令解析后没有任何可用 candidate 时生成默认 `CompilationCommand`：`arch=gcu400`、`driver=topscc`、`language=TOPS`、`language_standard=c++17`、`pass_kind=host+device`。无关 URI 不参与当前文档 candidate 计数；存在两个或以上非等价 candidate，或同一当前文档 candidate 的明确 language 冲突时保留 candidate 并标为 `invalid`，不回退到默认命令。标准库/Tops headers 和工具链内建宏来自 topscc 默认来源，工程 headers/宏来自 workspace 来源；canonical JSON、四类 instance 和字段级 conflict schema 由 `canonical.go` 与 resolver 固定实现；不得返回空 command 或 no-profile。
- **状态**：默认命令字段及所需 toolchain/workspace 来源完整时生成 `profile_id=gcu400-default` 和 `SemanticContext=resolved`；来源缺失时保留 `gcu400-default` 但状态为 `partial`；真实 target 冲突仍为 `invalid`。
- **依据**：规格澄清已确定上述字段、文件范围、GCU400 fallback、双 pass 和 provenance 规则；当前代码的 `CompilationContext`、`ParseContext` 和测试 fixture 已使用 `gcu400`、`topscc`、`c++17`、`device` 等字段。
- **替代方案**：没有命令时返回空 profile、默认普通 C++ 或静默选择其他 GCU 架构。拒绝，因为规格要求 profile 永远存在且默认固定 GCU400。

## 决策 4：host/device 使用组合 pass、按 pass 诊断

- **结论**：`pass_kind=host+device` 表示两个独立视图。semantic checker 分别计算 host/device；只有违规视图发布诊断，JSON `Diagnostic.data.passKind` 为 `host` 或 `device`，内部去重键包含 `pass_kind`。
- **依据**：现有 `CompilationContext.PassKind` 是单值字段，但规格已明确组合 pass 语义；现有 protocol `Diagnostic.data` 是可向后兼容的扩展点。
- **替代方案**：把 host/device 合并成一个规则集，或两侧都违规才报错。拒绝，因为会丢失调用边界归属或隐藏单侧错误。

## 决策 5：SemanticContext 状态与默认 fallback

- **结论**：真实可审计上下文按 `resolved/partial/missing/invalid` 进入 SemanticContext。可识别 Tops 但输入命令缺失时先使用默认命令；默认命令字段及所需来源完整时标为 `resolved`，来源缺失时标为 `partial`，无法形成语言/命令上下文时才标为 `missing`；冲突 target 使用 `gcu400-default` 但标为 `invalid`。fallback provenance 必须保留。
- **依据**：用户澄清明确选择默认命令可产生 resolved GCU400 规则；现有 parser/index 已使用 context version 和 stale 状态字段。
- **替代方案**：默认命令始终标 `missing` 或只作 profile 标签。拒绝，因为无法满足用户要求的默认 GCU400 语义检查。

## 决策 6：诊断由 Go server 自有契约生成

- **结论**：新增 `tops-semantic-*` 稳定 code；severity、主 range、related information、`profileId`、`contextStatus`、`passKind` 和版本由 Go server 生成。标准 LSP `publishDiagnostics` 是唯一 v1 用户可见出口。
- **依据**：`internal/protocol/diagnostics.go` 已有标准 `Diagnostic`、`DiagnosticData` 和条件 provenance；`internal/server/diagnostics.go` 已有 UTF-16 range 映射；`observability.go` 当前将 index diagnostics 映射为 Information，需要改为按 semantic severity 映射。
- **替代方案**：直接转发 Clang message 或新增私有 semantic LSP。拒绝，因为违反 LSP 契约优先和 Clang 离线边界。

## 决策 7：Clang 只作为离线对照

- **结论**：复用 `internal/parser/comparison_test.go` 的 runner 机制，但 semantic fixture 使用 `testdata/semantic/manifest.json`，comparison 输入固定为 `testdata/semantic/comparison_manifest.json`，输出固定为 `testdata/semantic/expected/semantic_comparison.json`；manifest 提供当前文档的 `document_uri`，`root_source_uri` 必须等于其 `NormalizeSourceURI` 结果并出现在 `source_files` 中，source/header entries 使用 T010 的 `AggregateSourceVersion`；要求 source hash、raw/normalized arguments、target、pass、include roots、宏和工具版本 provenance 一致；Go server runtime 不启动 Clang/clangd。
- **依据**：`specs/004-tops-cpp-tokenizer-parser/contracts/clang-comparison.md`、`internal/parser/comparison_test.go` 和 `comparison_contract_test.go` 已定义对照记录与 golden 机制。
- **未验证项**：本轮没有重新执行 Clang 对照，也不在计划中假设固定的 Clang 可执行文件路径；实现应沿用 `TOPS_LSP_CLANG` 或现有测试配置。

## 决策 8：验证优先使用现有 Go 入口

- **结论**：实现后使用 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`、`go test ./integration` 和已有 manifest checker；仓库没有 `.vscode/tasks.json` 或 Makefile，因此 quickstart 使用直接命令。
- **依据**：`go.mod` 声明 Go `1.26.8`；现有 `specs/005-ast-symbol-index/quickstart.md` 已记录这些入口。
- **替代方案**：新增任务系统或要求 GCU/GCUSIM。拒绝，因为本功能的 unit/contract/integration tests 应在开发机无设备环境运行。
