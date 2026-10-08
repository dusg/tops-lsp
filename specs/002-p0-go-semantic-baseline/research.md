# 研究记录：P0 Go 服务器语义基线

**日期**：2026-10-07
**范围**：为 P0 Go server 语义基线确认事实来源、候选 target profile、CompilationContext 规则、Go 层次边界和 P1 验证入口。不实现 Go/TypeScript 代码，不修改 clangd，不修改 `llvm-project`。

## 研究边界

- 研究工作在 `/home/carl.du/work/llvm-project` 读取当前 checkout 的 driver、TargetInfo、attribute、活动 headers 和测试。
- 规格工程在 `/home/carl.du/work/tops-lsp` 只生成文档和 Spec Kit 元数据。
- 当前结论区分三种状态：源码/测试事实已核对；仅有源码或目标条件证据的 `candidate`/`target-dependent`；尚未由 Go server 验证的 P1 计划项。
- Clang 是预处理、语法、Sema、AST/CodeGen 和目标参数的差分 oracle，不是 Go server 的运行时组件。

## Decision 1：以活动 LLVM/Tops 源码和测试为证据优先级

**Decision**：能力矩阵首先使用 driver、TargetInfo、attribute、`clang/lib/Headers/tops`、`clang/lib/Headers/tcle.h`、Clang tests、Tops integration tests 和现有状态记录；历史文档只作索引。

**Verified evidence**：

- `clang/include/clang/Driver/Types.def` 注册 `.tops`、`TOPS`、`TOPS_DEVICE` 和 `tops-cpp-output` 输入类型。
- `clang/include/clang/Driver/Options.td` 注册 `-Tops`、`--tops-sp`、`--tops-simt` 等选项。
- `clang/include/clang/Basic/LangStandards.def` 定义 `tops` language，并继承 C++、C++11、C++14 和 Digraphs 基础。
- `clang/lib/Basic/Cuda.cpp`/`Cuda.h` 注册 GCU300、GCU400、GCU410、GCU450、GCU500、EFGCU500 和部分 AGCU 组合。
- `clang/lib/Basic/Targets/DTU.h` 将 CPU 名称映射为 `ArchVersion`；`DTU.cpp` 在对应条件下定义 `__GCU_ARCH__`。
- `clang/lib/Basic/Targets/EFGCU.cpp` 定义 `__EFGCU_ARCH__=500`。
- `clang/lib/Driver/ToolChains/Clang.cpp` 和 `ToolChains/TOPS.cpp` 处理 offload arch、target macro 和 `efgcu500` toolchain 名称。
- `clang/lib/Headers/tops/__tops_defines.h` 定义 execution/memory/launch/alignment/DTE 相关宏；其条件分支依赖已确定的 `__GCU_ARCH__`。
- `clang/lib/Headers/tops/__tops_builtins.h` 与 `__tops_efgcu_builtin_vars.h` 使用两套不同的 builtin 类型和目标宏。
- `clang/include/clang/Basic/Attr.td` 定义 `TOPSLaunchBounds` 和 `TOPSMaxOACC` 的参数、主体和 Tops language 条件。

**Rejected alternative**：只以 `clang/docs/Tops_C_Language_Extenstion.md` 或用户样例为事实来源。原因是旧文档无法覆盖当前 target 分支、Sema 限制、EFGCU500 分离和 negative tests。

## Decision 2：Go server 自有语义，Clang 只作 oracle

**Decision**：Go server 自己拥有 tokenizer、parser、AST、symbol index、semantic 和 LSP 行为；TypeScript client 只拥有 VS Code 生命周期和用户交互；Clang/Tops compiler 只提供事实和差分验证。

**Rationale**：用户规格明确要求不扩展 clangd，项目宪法要求 Go/TypeScript 分层。当前 `tops-lsp` feature 目录只有文档，没有可复用 Go parser 或 LSP runtime，因此不能把实现责任隐含转移给 clangd。

**Required boundary**：

- tokenizer/parser 不选择 target 语义或 LSP severity。
- AST/index 保存 source range、宏/条件来源和 context/header version。
- semantic 读取 `CompilationContext`/`TargetProfile`，生成诊断、completion、hover 和导航事实。
- LSP 负责编码、取消、stale 和协议错误，不重新实现 semantic rule。
- TypeScript 只发送配置/文档通知和展示结果。
- Clang oracle 不在 server runtime 调用，也不作为 fallback。

## Decision 3：六个 candidate profile 必须分离 GCU 与 EFGCU

**Decision**：六个 profile 分别记录 candidate 命令、target triple/CPU/offload arch、宏、active headers、pass kind、状态、限制和验证动作。

| Profile | 源码已观察到的标识 | 宏事实 | 当前状态 | 主要核对动作 |
| --- | --- | --- | --- | --- |
| GCU300 | `gcu300` | `__GCU_ARCH__=300` | `candidate` | `-###`、device `-dM -E`、`.tops`/`-x tops` syntax |
| GCU400 | `gcu400` | `__GCU_ARCH__=400` | `candidate` | GCU400 parser/vector/attribute tests和header分支 |
| GCU410 | `gcu410` | `__GCU_ARCH__=410` | `candidate` | 410 宏、include、兼容性和独立 target matrix |
| GCU450 | `gcu450` | `__GCU_ARCH__=450` | `candidate` | GCU450 vector/TCLE negative tests和完整分支 |
| GCU500 | `gcu500` | `__GCU_ARCH__=500` | `candidate` | Draco TCLE、GCU500 header/API 和宏差分 |
| EFGCU500 | `efgcu500`/`efgcu-enflame-tops` | `__EFGCU_ARCH__=500` | `candidate` | host/device 宏、EFGCU headers、offload/target syntax |

**Important constraint**：不能由 `__EFGCU_ARCH__=500` 推断 `__GCU_ARCH__=500`，也不能由 GCU400/410 的共用 header 分支推断所有语义等价。实际命令没有生成预期宏时，context 必须是 `missing`/`invalid`/`partial`。

## Decision 4：CompilationContext 使用文件级、数据库优先规则

**Decision**：优先选择有效的文件级 `compile_commands.json` 条目；没有条目时才使用显式 workspace settings；driver 派生字段只从 raw arguments 和实际输出得到。

**Precedence**：

1. 规范化文件 URI/路径命中的有效 compile command。
2. 从 raw arguments 派生的 language、target triple、pass kind、宏和 include 顺序。
3. 无条目时的显式 workspace fallback。
4. 没有 target/include/standard 时不使用隐藏默认，返回 `missing`/`partial`。
5. 同一来源内部冲突为 `invalid`，不得静默合并。
6. 文本、配置、数据库、profile 或 header 变化递增 context version，旧 LSP 结果失效。

**Rejected alternative**：使用全局 GCU300 默认、系统 include 或 clangd fallback。原因是同一 workspace 可有多个 target，且 header 条件在 target 宏缺失时不能安全解释。

## Decision 5：P1 以 8 × 4 场景矩阵为最小验收面

规格把八个能力域分别拆成 valid、invalid、incomplete、target-boundary 四类，共 32 个场景。已有证据包括：

- `tops/integration_test/cases/language/STATUS.md` 的 C++11/14/17 状态和 4 个 device exception。
- `exec_spec/exec_space_specifiers/main.cc` 的执行空间、constant/shared 和 inline 组合。
- `launch_config/launch_bounds_maxnreg/main.cc`、`clang/test/CodeGenEFGCU/launch-bounds-*` 和 `maxnreg-error.cc` 的属性路径。
- `builtin_vars/grid_info/main.cc`、`lane_id/main.cc` 的 builtin/lane 使用。
- `vector_types/builtin_add/main.cc`、`clang/test/DTU_test/tcle/vector_op_parser.cc`、`maskbit_parser.cc` 的 vector 正/负路径。
- `clang/test/DTU_test/tcle/countl_zero_parser.cc` 和 `tcle/Draco/**` 的 target-specific diagnostics/API 入口。
- `clang/test/CodeGenEFGCU/**` 的 EFGCU500 target、vector、launch、barrier、DTE/async API 入口。

无现有证据的 invalid/incomplete/target-boundary 场景必须在 P1 新增 fixture，不能在 P0 标记为已通过。

## Decision 6：验证只在开发机执行文档和工具链核对

**Decision**：P0 只使用开发机源码读取、文档静态检查和本地 Clang/llvm-lit 核对；不构建或运行新的 Go/TypeScript 代码，不在测试机执行 workload。

**Validation commands**：

- 文档：`jq`、`rg`、`git diff --check`、VS Code diagnostics。
- driver/profile：`clang -###`、`clang -dM -E`、`clang -H` 或等价 include trace。
- syntax/oracle：`clang -fsyntax-only`、适用 `llvm-lit`/FileCheck fixture。
- P0 交付检查：确认 docs 存在、没有模板占位符、计数为 8 domains/6 profiles/32 scenarios、没有源码实现文件变更。

命令的实际工具路径、版本和输出必须在执行后写入对应 EvidenceRecord；本研究记录中的命令族不是对尚未执行结果的断言。

## 未决项与进入 P1 的条件

| 未决项 | 当前状态 | 进入 P1 前必须完成 |
| --- | --- | --- |
| 六个 profile 的 canonical driver 命令和 host/device 宏输出 | `candidate` | 对每个 profile 保存 `-###`、`-dM -E`、include roots 和 syntax 结果 |
| Go parser 依赖和源码布局 | `out-of-scope` | 在独立实现计划中选择并记录 Go 版本、依赖和测试入口 |
| 完整 Tops C++ grammar 覆盖 | `candidate` | 以 P1 32 场景和标准 C++ baseline 定义首期范围，不声称全量 ISO coverage |
| EFGCU500 与 regular GCU API 的重叠/差异 | `target-dependent` | 读取 active headers 和 target tests，分开建 index/semantic entries |
| P1 new fixtures 的文件布局和命名 | `planned` | `/speckit.tasks` 为每个场景生成可独立运行的 fixture/task |
| LSP library、client packaging 和 server executable | `out-of-scope` | 后续实现阶段记录版本、依赖、启动方式和兼容策略 |

## 结论

P0 的可交付判断是“事实、边界、状态和验证方法齐全”，不是“Go server 已经支持”。完成计划阶段后可以进入 `/speckit.tasks`；实现阶段仍必须保留 Clang differential、Go 自有测试、LSP contract test 和 target-specific fallback。
