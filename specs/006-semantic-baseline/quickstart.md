# 验证指南：Tops C++ 通用 Semantic Baseline

**功能**：[spec.md](spec.md)
**计划**：[plan.md](plan.md)
**数据模型**：[data-model.md](data-model.md)
**SemanticContext 契约**：[contracts/semantic-context.md](contracts/semantic-context.md)
**诊断契约**：[contracts/lsp-semantic-diagnostics.md](contracts/lsp-semantic-diagnostics.md)
**Clang 对照契约**：[contracts/clang-semantic-comparison.md](contracts/clang-semantic-comparison.md)

本指南用于实现完成后的开发机验证。不要求 GCU 设备、GCUSIM、测试机 Docker、clangd 或 Clang runtime；Clang 只在显式离线对照测试中使用。

## 前置条件

- 工作目录为 `/home/carl.du/work/tops-lsp`。
- Go 版本与 [go.mod](../../go.mod#L1) 中的 `go 1.26.8` 一致。
- 测试材料使用 `testdata/semantic/`、现有 `testdata/index/` 和临时 workspace/source provider。
- 默认 Tops 命令只适用于 `.tops`、`languageId=tops` 或明确 Tops 模式。

## 1. 定向 semantic 测试

```sh
go test ./internal/semantic ./internal/parser ./internal/index ./internal/protocol ./internal/server
go test ./integration
go run ./cmd/tops-semantic-check --manifest testdata/semantic/manifest.json
```

预期：

- declaration/scope/name lookup、scalar/pointer/reference、function parameter/return tests 通过；
- default `CompilationCommand` 产生 `gcu400-default` 和 `pass_kind=host+device`；
- host/device diagnostics 按 JSON `Diagnostic.data.passKind` 隔离；
- address-space/vector 规则只按当前 profile/pass 执行；
- no-profile snapshot 和 no-profile diagnostic 数量为 0。
- semantic manifest 输出每类场景总数、通过数、失败数和失败 case ID；callable 四类各有四种场景。

## 2. 默认命令与文件范围

验证以下场景：

1. Tops-signaled document 的输入命令缺失，或当前文档规范化 URI 对应的 candidate 为空时，生成 `arch=gcu400`、`driver=topscc`、`language=TOPS`、`language_standard=c++17`、`pass_kind=host+device`；无关 URI 不参与计数，两个或以上非等价 candidate 或同一文档候选的明确 language 冲突时保留冲突并得到 `SemanticContext.status=invalid`，不生成默认命令。
2. `languageId=tops` 的 `.cpp`：允许使用同一默认 Tops 命令；`.tops` 遇到明确普通 C++ compile command 时使用 `cpp-default`。
3. 普通 `.cpp`/`.cc` 无 compile command：使用 `cpp-default`，来源完整时整体为 `resolved`、来源缺失时为 `partial`，不得被静默转换为 TOPS。
4. 输入命令缺失但工具链/workspace 所需来源缺失：仍先生成默认命令，`SemanticContext.status=partial`。
5. 未知 language：使用 `cpp-default`，`SemanticContext.status=missing`，不启用 Tops/GCU400 规则。
6. 多个 target 冲突：保留 `gcu400-default` profile，但 `SemanticContext.status=invalid`，不选择冲突 target。
7. 默认命令检查整体 `source=default`、五个核心字段的 `field_source=fallback_default`、toolchain/workspace 字段来源、固定 command field key 的 map shape、无冲突时 `field_conflict=null`、缺失来源的 `field_resolution=unresolved` 和冲突时按 `conflict_sort_order=compile_commands < workspace < toolchain_default < fallback_default` 再按 `instance` 字节序保存的 `{sources,values}` 记录；无冲突字段按 `source_precedence=compile_commands > workspace > toolchain_default > fallback_default` 选择来源；`sources` 与 `values` 长度相等且至少包含两个冲突来源，任意字段冲突时字段为 `field_source=invalid`/`field_resolution=invalid`、整体为 `invalid`，`1`、`1.0` 和 `1e0` 等价。
8. 同一规范化 source URI 的两条等价 `compile_commands` 只保留一条；无关 URI 不制造冲突；两条非等价命令按稳定 `ConflictSource.instance` 记录并进入冲突状态；交换输入 candidate 顺序、重复读取 workspace/toolchain 来源或重新生成 fallback 时，四类 instance 和最终 snapshot 必须保持不变。相对 working directory 必须以 workspace root 规范化，response file 展开失败的 candidate 不可用，canonical JSON hash 使用无尾部换行的 UTF-8 bytes。

## 3. Headers、宏和条件分支

- 标准库和 `topscc` Tops headers 从工具链默认 include roots 加载；工程 headers 从 workspace roots 加载。
- 工具链内建宏和 workspace 工程宏分别记录 provenance。
- 缺少工程 header 或宏时标记 unresolved/degraded；不扫描整个 workspace，不创建虚假 header/macro。
- 验证 `#if/#ifdef/#ifndef/#elif/#else/#endif` 的 active/inactive/unknown 状态和 parent condition。

## 4. Semantic diagnostics

使用有效、无效、不完整和 target 边界 fixture，检查：

- `tops-semantic-unknown-name`、`tops-semantic-type-mismatch`、`tops-semantic-invalid-call`；
- `tops-semantic-host-device-boundary`、`tops-semantic-address-space`、`tops-semantic-vector-constraint`；
- resolved default GCU400 context 下目标 Error 具有非空源码 range；
- partial/missing/invalid context 下只发布允许的降级结果；
- host/device 视图分别发布，合法 pass 不发布诊断；
- host-only、device-only、host/device 两侧可用和单侧 `availability=missing` 四种 pass-view 组合分别检查 `availability`、`status`、`rules_available` 和 `provenance`；单侧缺失时整体为 `partial`，无法形成语言/命令上下文时才为整体 `missing`；
- UTF-16、CRLF、Unicode、EOF 和零宽范围正确；
- `textDocument/publishDiagnostics` 使用标准结构，不新增私有 semantic LSP。

## 5. 版本和 stale

对同一 URI 提交文档版本、workspace/header 版本和 SemanticContext 版本变化：

- 新版本提交后旧 semantic snapshot、diagnostics 和 LSP 结果不再发布；
- stale 只记录 `stale_result`，包含旧/新版本和 reason；
- header 变化使依赖它的 semantic snapshot 失效；
- 组合 pass 的旧结果不得覆盖新 pass 或新上下文结果。

## 6. Clang 离线对照

复用 `internal/parser/comparison_test.go` 的环境变量和 manifest 入口，仅在输入 provenance 完全一致时比较：

```sh
TOPS_LSP_CLANG=/path/to/clang go test ./internal/parser -run Comparison
TOPS_LSP_CLANG=/path/to/clang go test ./internal/semantic -run TestSemanticComparisonManifest
# 仅在更新 golden 时允许写入 comparison 输出
TOPS_LSP_CLANG=/path/to/clang TOPS_LSP_ALLOW_GOLDEN_WRITE=1 go test ./internal/semantic -run TestSemanticComparisonManifest
```

纯 Go comparison runner 也可以直接生成 normalized record；它不会启动 Clang：

```go
records, err := semantic.GenerateComparisonRecords("testdata/semantic/comparison_manifest.json", "testdata/semantic/manifest.json")
if err != nil {
	panic(err)
}
_ = semantic.WriteComparisonRecords("testdata/semantic/expected/semantic_comparison.json", records)
```

对照记录至少包含 `document_uri`、`root_source_uri`、source hash、raw/normalized arguments、target、pass、include roots、宏、工具版本、Go code/range 和 Clang normalized category。`root_source_uri` 必须等于 `NormalizeSourceURI(document_uri)` 并出现在 source files 中；root 或 source/header version entry 缺失时标记 `not-comparable`。server runtime 不启动 Clang/clangd，也不转发 Clang message。

未设置 `TOPS_LSP_CLANG` 时，纯 Go semantic tests 必须通过；comparison test 显式跳过并报告 skipped，不得把跳过当作 semantic validator failure。

## 7. 全量质量检查

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
git diff --check
```

预期：现有 parser/index/navigation/stale tests 和新增 semantic tests 全部通过；没有修改 `llvm-project`、`topsop` 或外部 checkout。
