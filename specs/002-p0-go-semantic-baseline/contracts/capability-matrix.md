# 契约：P0 八域语法能力矩阵

## 目的

本契约规定如何把 Tops C++ 语法、标准 C++、TCLE/API 和目标条件记录为可审查的 `SyntaxCapability`。它不是 Go server runtime API，也不把源码出现关键字当作实现支持。

## 条目字段

每个条目必须包含：

| 字段 | 要求 |
| --- | --- |
| `id` | `TOPS-<DOMAIN>-<number>`，稳定且不复用 |
| `domain` | `standard-cpp`、`execution-space`、`memory-space`、`launch-resource`、`vector-numeric`、`builtin`、`tcle-api`、`target-condition` 之一 |
| `spelling` | 用户源码中的关键字、attribute、类型、builtin、宏或 API |
| `user_semantics` | 用户可观察的解析/语义行为和限制 |
| `target_precondition` | target profile、triple/CPU、pass、language standard、macro、header 或 include 条件 |
| `source_evidence` | 当前 driver、TargetInfo、attribute、header 或 compiler source 的 EvidenceRecord |
| `test_evidence` | 现有 test/status 或明确标为 planned 的 P1 fixture |
| `status` | `candidate`、`verified`、`target-dependent`、`supported`、`unsupported`、`blocked`、`deprecated-source` |
| `limitations` | 版本、目标、coverage、runtime/CodeGen 或未知项 |
| `verification` | 可复现的 valid、invalid、incomplete、target-boundary 检查 |
| `planned_phase` | P0/P1/P2/P3/P4 |
| `lsp_surface` | diagnostics/completion/hover/definition/references/document-symbol/documentation-only |

## 证据优先级

1. 当前活动 target 条件、Clang TargetInfo、attribute 和 Sema/driver 实现。
2. `clang/lib/Headers/tops`、`clang/lib/Headers/tcle.h` 和当前 active include roots。
3. Clang/DTU/EFGCU lit、FileCheck 和 `tops/integration_test/cases/language`。
4. `STATUS.md` 等测试状态记录。
5. 历史维护文档，必须标记 `deprecated-source`，不能作为唯一依据。

## 状态门禁

- 没有源码证据的条目不能超过 `candidate`。
- 只有源码证据、没有适用 test 或 syntax/compiler verification 的条目不能标为 `supported`。
- 目标条件不同的条目必须拆成 profile 记录或在 `target_precondition` 明确分支。
- GCU 与 EFGCU 的宏、builtin、header 和 API 不因数值相同而合并。
- P0 中 Go server 尚未实现，因此任何语言服务 surface 只能写为 planned/candidate；`supported` 只允许在后续实现和回归门禁通过后使用。
- `blocked`/`unsupported` 必须有恢复条件、拒绝理由和用户可见行为。

## 八域覆盖门禁

| 域 | P0 必须覆盖 | 最小现有/计划证据 |
| --- | --- | --- |
| standard-cpp | C++11/14/17 baseline、device exception | `tops/integration_test/cases/language/STATUS.md` 和各标准目录 |
| execution-space | global/device/host/host-device、inline/noinline、candidate 的 cooperative/sp/scalar-only | `exec_spec/exec_space_specifiers/main.cc`、Clang Sema oracle、P1 negative/incomplete |
| memory-space | constant/shared/local/private/cluster、restrict、DTE context qualifiers | `__tops_defines.h`、memory/exec fixtures、EFGCU qualifier tests |
| launch-resource | thread/cluster dims、launch bounds、maxnreg、block tile | `launch_config`、`CodeGenEFGCU`、`topscc/attribute` |
| vector-numeric | vector spellings/builtin structs、half/BF16/FP4/FP6/FP8、valigned | vector headers、vector integration、tcle parser tests |
| builtin | regular GCU builtins、EFGCU builtins、lane/warp candidates | builtin headers、grid/lane fixtures、target matrix |
| tcle-api | tcle/vector/DTE/pipeline/barrier/queue/API symbols | current headers、`clang/test/DTU_test/tcle`、planned P1 fixtures |
| target-condition | input forms、CPU/offload arch、macro/pass/header branches | driver/TargetInfo/toolchain、`-###`/`-dM -E` |

## 测试记录格式

每个 P1 fixture 必须按以下字段记录：

```text
id: P1-<DOMAIN>-V|I|P|T
source_path: existing path or planned new fixture
profile: one of six candidate profiles
context: standard, pass, include roots, raw/normalized args, macros
parser_state: complete|recoverable|degraded|invalid
expected: diagnostics/symbols/completion/hover/navigation
oracle: Clang/status/test evidence or explicit gap
limitations: runtime/CodeGen/hardware behavior excluded from P1
verification: Go test + LSP contract + differential check as applicable
```

有效、无效、不完整和目标边界场景必须分别可执行；不能用一个 happy-path fixture 代表整个能力域。
