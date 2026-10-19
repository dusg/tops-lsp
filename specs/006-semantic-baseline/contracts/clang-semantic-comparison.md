# 契约：Semantic Clang 离线对照

## 作用

Clang 只在开发机测试流程中作为对照编译器。它不进入 Go server runtime，不作为 SemanticContext fallback，也不直接提供用户诊断文本。

## 可比较条件

双方必须使用相同的：

- 源文件 bytes/source hash；
- working directory；
- raw/normalized arguments；
- target/profile、pass/pass views；
- toolchain/project include roots；
- toolchain/workspace macros；
- header versions；
- compiler/tool version provenance。

任一关键字段不一致时，记录 `not-comparable` 和缺失字段，不归因于 semantic checker。

## ComparisonCase 字段

每条 `comparison_manifest.json` 记录必须包含：

| 字段 | 说明 |
| --- | --- |
| `fixture_case_id` | 必须引用 `testdata/semantic/manifest.json` 中存在的 `case_id` |
| `document_uri` | 当前文档的 raw URI，作为 root source 的唯一输入 |
| `root_source_uri` | 当前文档 root source 的规范化 URI；必须出现在 `source_files` 中 |
| `source_files` / `source_hash` | 完整输入文件集合和内容指纹 |
| `working_directory` | 双方相同的路径基准 |
| `raw_arguments` / `normalized_arguments` | 原始和归一化参数 |
| `profile_id` / `pass_kind` | profile 和 pass 视图 |
| `include_provenance` / `macro_provenance` | toolchain/workspace 来源 |
| `header_versions` / `compiler_tool_version` | 依赖和工具版本 |
| `source_version_fingerprint` | 与 semantic diagnostic `sourceVersion` 一致 |
| `compare_status` | `comparable` 或 `not-comparable` |
| `missing_fields` / `known_difference` | 不可比较原因或已知差异 |
| `neutral_diagnostics` | 顶层 pass-neutral Go diagnostics；每条 `passKind=not-applicable`，不放入 `pass_results` |

`compare_status=not-comparable` 时不得生成用户可见的 semantic diagnostic。

`root_source_uri` 必须由当前 `document_uri` 经 `NormalizeSourceURI` 得到；缺失、无法规范化、与派生值不一致或不在 `source_files` 中时，comparison record 必须为 `not-comparable`。每个 `source_files[].content_hash` 作为对应 `SourceVersionEntry.version`；`header_versions` 直接提供 header 的 `SourceVersionEntry`，由 T010 的 `AggregateSourceVersion` 统一生成 `source_version_fingerprint`。

`pass_results` 的每个实际 pass 只能使用 `host` 或 `device`；`go_status` 只能是 `complete`、`degraded`、`unsupported` 或 `invalid`，`clang_status` 只能是 `available`、`skipped`、`failed` 或 `not-comparable`；`missing_fields`、`known_difference` 和 `clang_categories` 是字符串数组，`go_diagnostics` 是带 code/severity/range/passKind 的对象数组。validator 必须校验 `fixture_case_id` 引用、字段类型和上述枚举。

执行入口固定为：

```sh
go test ./internal/semantic -run TestSemanticComparisonManifest
TOPS_LSP_CLANG=/path/to/clang TOPS_LSP_ALLOW_GOLDEN_WRITE=1 go test ./internal/semantic -run TestSemanticComparisonManifest
```

未设置 `TOPS_LSP_CLANG` 时 comparison test 报告 skipped，纯 Go semantic tests 仍必须通过；golden 写入只允许在显式设置 `TOPS_LSP_ALLOW_GOLDEN_WRITE=1` 时发生。

## 对照记录

```text
fixture
source_hash
language_standard
profile_id
pass_kind
include_provenance
macro_provenance
go_status
go_diagnostics(code/range/severity/passKind)
neutral_diagnostics(code/range/severity/passKind=not-applicable)
clang_status
clang_normalized_categories
known_difference
```

只比较类别、确定性、主范围和来源关系；不比较或转发 Clang 原始 message。

## 现有入口

复用 `internal/parser/comparison_test.go`、`comparison_contract_test.go` 和 `TOPS_LSP_CLANG` 环境变量；fixture manifest 固定为 `testdata/semantic/manifest.json`，semantic runner 固定读取 `testdata/semantic/comparison_manifest.json`，normalized golden 固定写入 `testdata/semantic/expected/semantic_comparison.json`。新增 semantic 对照测试必须在 Go server runtime 之外执行，并能在没有 Clang 可执行文件时跳过或运行纯 Go semantic tests。

纯 Go runner 可通过 `semantic.GenerateComparisonRecords` 读取两个 manifest，并通过 `semantic.WriteComparisonRecords` 写入 normalized golden；该路径不启动 Clang，也不把 comparison record 放入 server semantic cache 或 `publishDiagnostics`。
