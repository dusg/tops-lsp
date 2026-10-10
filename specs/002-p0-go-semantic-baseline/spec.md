# Feature Specification: P0 Go 服务器语义基线

**Feature Branch**: `002-p0-go-semantic-baseline`

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "请为 Tops C++ 语言服务器编写“P0 Go 服务器语义基线”功能规格。P0 只输出文档，不实现 Go/TypeScript 代码，也不扩展 clangd。文档必须一次性完成八个能力域的语法清单、源码和测试证据、GCU300/GCU400/GCU410/GCU450/GCU500/EFGCU500 候选 profile、CompilationContext 字段和优先级、Go tokenizer/parser/AST/symbol index/semantic/LSP 的责任边界，以及 P1 所需的有效、无效、不完整和目标边界测试场景。每项都要写明目标前提、状态、限制和验证方法。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 审查八个能力域的语义基线 (Priority: P1)

作为 Tops C++ 语言工具维护者，我希望在一份规格中看到完整的八个能力域、源码证据、测试证据、目标前提、状态、限制和验证方法，从而可以判断一个语法条目是否足以进入 P1。

**Why this priority**: 没有统一的证据和状态，Go 服务器会把标准 C++、Tops 扩展、TCLE/API 和目标相关分支混成无条件支持；这会直接污染诊断、补全和跨目标行为。

**Independent Test**: 只检查本规格中的能力矩阵和引用路径；每个条目都必须能定位到当前 checkout 中的源码或测试，或者明确写为 `candidate`、`blocked`、`unsupported` 并给出恢复/验证动作。

**Acceptance Scenarios**:

1. **Given** 维护者打开能力矩阵，**When** 选择任意八个能力域中的一项，**Then** 可以同时看到语法清单、目标前提、源码证据、测试证据、状态、限制和验证方法。
2. **Given** 某个能力只在部分架构或编译阶段可见，**When** 维护者查看该能力，**Then** 规格不会将它写成所有 profile 的无条件能力。
3. **Given** 某个能力只在旧文档中出现而当前源码或测试没有证据，**When** 维护者查看该能力，**Then** 它不能超过 `candidate`，并且必须注明待核对的活动 header、TargetInfo 或测试入口。

### User Story 2 - 按 CompilationContext 实现 Go 语义层 (Priority: P1)

作为 Go 服务器实现者，我希望知道 `CompilationContext` 的字段、来源优先级、冲突行为和失效规则，并能区分 tokenizer、parser、AST、symbol index、semantic 和 LSP 的所有权，从而按稳定边界实现 P1，而不把语义决策转移给 clangd 或 TypeScript 客户端。

**Why this priority**: Tops 语义由输入模式、C++ 标准、目标架构、宏和 include 根共同决定；上下文或层次边界不明确时，任何局部实现都可能产生互相矛盾的结果。

**Independent Test**: 对每个字段构造一个有编译数据库、无编译数据库、字段冲突和字段变更的最小文档输入，检查是否得到确定的 `resolved`、`partial`、`missing` 或 `invalid` 状态，以及可操作的诊断。

**Acceptance Scenarios**:

1. **Given** 文件存在有效的 `compile_commands.json` 条目，**When** 工作区设置与该条目的 target 或 standard 不同，**Then** 文件使用编译数据库条目，设置不能静默覆盖明确参数。
2. **Given** 文件没有编译数据库条目但工作区明确提供了编译上下文，**When** Go 服务器分析文件，**Then** 使用工作区 fallback，并在结果中保留来源。
3. **Given** target、输入模式或 include 根缺失或冲突，**When** 服务器分析文件，**Then** 返回 `missing-compilation-context`、`invalid-compilation-context` 或 `analysis-degraded`，不猜测 profile。

### User Story 3 - 为 P1 建立可回归测试入口 (Priority: P2)

作为测试维护者，我希望每个能力域都有有效、无效、不完整和目标边界场景，并且场景明确指出现有证据与待新增测试材料，从而能在 P1 实现前固定验收范围。

**Why this priority**: P1 的风险不只在正向解析；宏不完整、目标切换、非法属性和 host/device 边界更容易暴露错误恢复与语义分层问题。

**Independent Test**: 逐项执行本规格的 P1 场景矩阵；已有测试材料用当前测试入口复核，规划测试材料在 P1 创建后以同样的 profile 和对照验证命令验证。

**Acceptance Scenarios**:

1. **Given** 一个有效 Tops 源文件，**When** 使用匹配的 profile 和上下文分析，**Then** 语法、符号和目标相关结果符合场景预期。
2. **Given** 一个无效或不完整源文件，**When** 用户在编辑过程中触发分析，**Then** 服务器保留可恢复结构并输出稳定范围的诊断，而不是崩溃或清空全部符号。
3. **Given** 同一源文件在两个 profile 下分析，**When** 条件宏或目标限制不同，**Then** 只有受影响能力的结果变化，且诊断说明 profile 和限制。

### Edge Cases

| 场景 | 目标前提 | 状态 | 限制 | 验证方法 |
| --- | --- | --- | --- | --- |
| 没有编译数据库、编译器或 Tops include 根 | 目标未解析 | `missing` | 不允许使用隐含的 GCU300 默认语义 | 检查 `CompilationContext.resolution_status` 和 `missing-compilation-context` |
| `.tops`、`.cpp + -Tops`、`-x tops` 的输入模式不一致 | `tops` language 必须已确定 | `candidate` | 当前规格只规定统一语义目标，不预先规定 driver 归一化实现 | 对三种输入执行 `-dM -E` 和 `-fsyntax-only` 差分 |
| `compile_commands.json` 使用 `topscc` argv | wrapper 版本和工作目录可确认 | `required` | 必须先解析 `-arch`、device flags 和 wrapper 默认值，再派生 Clang 参数 | 保存 raw/normalized/forwarded arguments，并用 `topscc --dryrun` 离线核对 |
| `topscc -arch` 展开多个目标 | 多架构组合或 AGCU 目标 | `partial` | 不允许静默选择第一个目标；需要显式 target selection | 检查 `target_profile_ids`、`target_selection` 和 `ambiguous-target-context` |
| `topscc` 参数与直接 Clang 参数冲突 | 同一 compile command 同时出现 wrapper 和下游目标设置 | `invalid` | 保留参数来源和冲突位置，不自行合并 | 构造冲突 argv，检查 `invalid-topscc-arguments` |
| response file 缺失或无法解析 | compile command 引用 `@file` | `partial` | 不猜测其中的 include、宏或 target | 按 working directory 展开并记录 `unresolved-response-file` |
| `compile_commands` 与工作区设置提供冲突 target、standard、include 或宏 | 文件有或无数据库条目 | `invalid` | 明确编译参数不能被设置静默覆盖 | 构造冲突条目，检查稳定错误码和来源字段 |
| `__GCU_ARCH__` 未在 header 条件分支前确定 | GCU device 编译上下文不完整 | `partial` | 不把 header 后面的兜底定义当作目标选择 | 预处理 `__tops_defines.h`，检查宏和可见声明 |
| EFGCU500 只有 `__EFGCU_ARCH__=500` | `efgcu-enflame-tops` 或 `--offload-arch=efgcu500` | `target-dependent` | 不把 EFGCU500 当成 `__GCU_ARCH__=500` | 对 host/device 两个 pass 分别执行 `-dM -E` |
| 输入处于未闭合注释、字符串、模板、属性参数或预处理条件 | 任意已解析 profile | `analysis-degraded` | 诊断范围必须可定位，既有 AST 不得全部丢失 | 逐字输入测试材料，比较增量结果和 stale 结果 |
| 当前 header 与编译器来自不同 checkout 或版本 | profile 字段不一致 | `invalid` | 规格不承诺跨版本 header 的语义等价 | 记录 compiler、resource dir、include 顺序并复核预处理输出 |
| C++11/14/17 回归通过但 device coverage exception 仍存在 | 现有 `STATUS.md` 记录 | `target-dependent` | 回归通过不等于全部 device feature 通过 | 单独保留 BUG-2/4/5/6 的负向和 blocked 结果 |
| Clang 对照验证 与 Go 服务器结果不同 | P0/P1 差分验证 | `candidate` | Clang 只作差分参考，不能成为 Go server 运行依赖 | 保留两侧诊断、profile、上下文版本和差异原因 |

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 本功能 MUST 只交付文档；不得创建或修改 Go、TypeScript、C++ parser、clangd 扩展或运行时语义代码。
- **FR-002**: 本规格 MUST 完整列出八个能力域：标准 C++、执行空间、存储空间、启动/资源、向量/数值、内建变量、TCLE/API、目标条件。
- **FR-003**: 八个能力域中的每个语法条目 MUST 记录目标前提、源码证据、测试证据、状态、限制和验证方法；缺少证据时状态不得超过 `candidate`。
- **FR-004**: 源码证据 MUST 优先来自当前 `llvm-project` checkout 的 driver、TargetInfo、Clang attribute、活动 Tops headers 和当前测试；已标记不再维护的旧文档只能作为索引。
- **FR-005**: 测试证据 MUST 区分编译/预处理、Clang Sema/CodeGen、Tops integration test、状态记录和未来 P1 测试材料；不得把“源码出现关键字”写成语义验证。
- **FR-006**: 本规格 MUST 定义 GCU300、GCU400、GCU410、GCU450、GCU500 和 EFGCU500 六个候选 `TargetProfile`，并为每个 profile 记录输入模式、候选 CPU/target、活动宏、状态、限制和验证方法。
- **FR-007**: `TargetProfile` MUST 区分 `__GCU_ARCH__` 和 `__EFGCU_ARCH__`，不得从相邻代际或同数值宏推断未核验能力；profile 尚未通过实际 driver 命令时状态必须保持 `candidate` 或 `target-dependent`。
- **FR-008**: 本规格 MUST 定义 `CompilationContext` 的字段、字段来源、字段优先级、冲突行为、缺失行为和 stale/重新解析规则。
- **FR-009**: 同一文件的有效 `compile_commands.json` 条目 MUST 优先于工作区 fallback；工作区设置不得覆盖数据库中的明确 target、standard、输入模式、include、宏或 device flags；无条目时才允许 fallback。
- **FR-010**: Go 服务器的 tokenizer、parser、AST、symbol index、semantic 和 LSP 层 MUST 各自有明确的输入、输出、拥有的判断、明确不拥有的判断、目标前提、状态、限制和验证方法。
- **FR-011**: `topscc`/直接 Clang、活动 headers 和测试 MUST 分别记录用户编译入口、事实来源和离线对照用途；clangd 不得成为 Go server 的运行时语义后端、sidecar 或隐藏 fallback。
- **FR-012**: TypeScript VS Code 客户端 MUST 只负责扩展激活、LSP 进程生命周期、配置传递、状态展示、日志、重启和用户命令；客户端不得复制目标或语义判断。
- **FR-013**: 本规格 MUST 为 P1 的八个能力域分别定义有效、无效、不完整和目标边界场景；每个场景 MUST 记录目标前提、状态、限制和验证方法，并注明现有或待新增测试证据。
- **FR-014**: P1 的诊断、补全、悬停、定义、引用和文档符号测试 MUST 使用同一 `CompilationContext` 和 profile；上下文版本变化后的旧结果 MUST 标记为 stale 或被丢弃。
- **FR-015**: 所有未实现的 Go 服务器能力 MUST 使用 `candidate`、`target-dependent`、`blocked` 或 `unsupported` 等状态表达，不得在 P0 文档中声称语言服务器已经 `supported`。
- **FR-016**: 文档 MUST 记录 P0 的验证命令族和验收产物，但不得要求 P0 构建、部署或运行新的 Go/TypeScript 代码。
- **FR-017**: P0 MUST 定义 `CompilerInvocation`，区分 `topscc`、直接 `clang`/`clang++` 和未知 wrapper，并保留原始 argv、归一化 argv、转发参数、默认值来源和未知参数。
- **FR-018**: P0 MUST 记录已安装 `topscc` wrapper 的版本范围和已验证参数类别，包括 `-arch`、device output/link 参数、SIMD/SIMT 参数、`-x`、`-std`、include、macro 和 `--dryrun`；版本相关或未验证参数必须标记待确认。
- **FR-019**: P0 MUST 规定 `topscc` 默认值只在 driver kind 和 wrapper version 已确认时生效；多架构展开、冲突参数、缺失 response file 和未知参数不得静默降级为单一完整 context。
- **FR-020**: LSP runtime MUST 只静态解析 compile command argv，不启动 `topscc`；`topscc --dryrun` 只能作为开发机离线核对，不能成为 Go server 运行时 fallback。

### P0 证据基线

证据状态说明：下表中的 `verified` 表示当前 checkout 的源码或测试事实已核对，不表示 Go 服务器已经实现。P0 不重新定义 LLVM 或 topscc 的语义，只规定 Go server 必须以这些可复核事实为输入，并在差异处保留状态。

#### 源码证据

| 证据 | 直接事实 | 目标前提 | 状态 | 限制 | 验证方法 |
| --- | --- | --- | --- | --- | --- |
| `/opt/tops/bin/topscc` | 已安装 wrapper 处理 `-arch`、device output/link、SIMD/SIMT 和 `--dryrun`，并向底层 Clang 注入 Tops language/include/default flags | `topscc` v4.0.20260828；实际工作目录和 PATH | `observed` | wrapper 参数集合和默认值可能随版本变化；只对当前安装版本作事实记录 | `topscc --version`、`readlink -f $(command -v topscc)`、`topscc --dryrun -arch gcu400 -x tops -fsyntax-only /dev/null` |
| `llvm-project/clang/include/clang/Driver/Types.def` | 注册 `tops-cpp-output`、`.tops` 对应的 `TOPS`/`TOPS_DEVICE` 输入类型 | driver 输入类型表 | `verified` | 输入类型注册不等于语义支持 | `rg -n 'TYPE\\("tops'` 并执行 `clang -###` |
| `llvm-project/clang/include/clang/Driver/Options.td` | 注册 `-Tops`，以及 `--tops-sp`、`--tops-simt` 等 Tops 选项 | Clang driver | `verified` | 具体选项组合仍需 profile 验证 | 检查 option 定义并以 `clang -###` 核对展开参数 |
| `llvm-project/clang/include/clang/Basic/LangStandards.def` | `tops` language 继承 C++、C++11、C++14 和 Digraphs 基础 | `-Tops` 或 `-x tops` | `verified` | 不决定具体 `-std` 选择和 device coverage | 预处理/语法检查三种 Tops 输入形式 |
| `llvm-project/clang/lib/Basic/Cuda.cpp`、`clang/include/clang/Basic/Cuda.h` | 枚举和名称映射包含 GCU300/400/410/450/500、EFGCU500 以及部分 AGCU 组合 | `-mcpu`/`--offload-arch`/driver target | `verified` | 名称注册不等于所有组合都可用 | 对每个候选名执行 `-###` 和 `-fsyntax-only` |
| `llvm-project/clang/lib/Basic/Targets/DTU.h`、`DTU.cpp` | `opts.CPU` 的完整分支把 gcu300/400/410/450/500 映射为 `ArchVersion`，device 条件下定义对应 `__GCU_ARCH__` | GCU device pass；`Opts.TOPS` 或 `CUDAIsDevice` 影响宏注入 | `verified` | 缺少 device/context 时不能假设宏存在 | `-dM -E`，逐 profile 检查 `__GCU_ARCH__` |
| `llvm-project/clang/lib/Basic/Targets/EFGCU.cpp` | EFGCU TargetInfo 定义 `__EFGCU_ARCH__=500` | EFGCU target | `verified` | 不产生 `__GCU_ARCH__=500` 的等价结论 | `-dM -E` 分别检查 host/device |
| `llvm-project/clang/lib/Driver/ToolChains/Clang.cpp`、`ToolChains/TOPS.cpp` | offload arch `efgcu500` 和 GCU/AGCU 名称会被转换为 target 宏或 `efgcu500` toolchain 名称 | `-Tops`、`--cuda-gpu-arch` 或 `--offload-arch` | `verified` | 直接 target 与 offload target 仍需分别记录 | `clang -###` 对比实际 cc1 参数 |
| `llvm-project/clang/lib/Headers/tops/__tops_defines.h` | 定义执行、存储、启动/资源、对齐和 DTE 相关宏；`__thread_dims__`、`__maxnreg__`、`__private_dte__`、`__local_dte__` 按 `__GCU_ARCH__` 分支 | 活动 Tops header，且目标宏必须在条件分支前确定 | `verified` | 文件末尾的默认 `__GCU_ARCH__` 不能替代缺失的编译上下文 | 对每个 profile 预处理 header 并检查可见宏 |
| `llvm-project/clang/lib/Headers/tops/__tops_builtins.h`、`__tops_efgcu_builtin_vars.h` | regular GCU 提供 `threadIdx`、`blockIdx`、`blockDim`、`gridDim`、`threadDim`、`subThreadIdx`；EFGCU header 提供前四者和 `warpSize` | regular GCU 的 `__GCU_ARCH__ >= 300`；EFGCU500 的 `__EFGCU_ARCH__=500` | `target-dependent` | 两套 builtin 类型、转换和可用变量不能合并 | 预处理、AST/符号检查和目标差分 |
| `llvm-project/clang/include/clang/Basic/Attr.td` | `TOPSLaunchBounds` 和 `TOPSMaxOACC` 声明 `tops_launch_bounds`、`__maxnreg__` 的参数和适用主体 | `LangOpts.TOPS`；EFGCU 还需目标限制验证 | `verified` | attribute 声明不覆盖全部 Sema/CodeGen 限制 | 使用正向和超限 negative tests |
| `llvm-project/clang/lib/Headers/tops/vector_types.h`、`vector_functions.h` | 提供 `char1` 至 `float4` 等 builtin vector struct、成员和 make 函数 | Tops headers 可解析 | `verified` | 向量宽度/对齐和 device 运算随目标变化 | AST、语法和 integration vector add |
| `llvm-project/clang/lib/Headers/tops/ef_fp16.h`、`ef_bf16.h`、`ef_fp4.h`、`ef_fp6.h`、`ef_fp8.h`、`tops_fp*.h` | 提供半精度、BF16、FP4/FP6/FP8 相关类型和 API | GCU/EFGCU 目标及 active include 根 | `target-dependent` | 类型名称、实现路径和可用运算不能跨 profile 直接复制 | profile 预处理、Clang syntax/codegen tests |
| `llvm-project/clang/lib/Headers/tcle.h`、`tops/__tops_dte.h`、`__tops_dte_ext.h`、`topscc_types.h`、`__tops_pipeline.h` | 提供 TCLE 向量/转换常量、DTE/mdspan、pipeline、barrier 和不同架构条件分支 | Tops device compile；部分 header 明确按 `__GCU_ARCH__ < 500` 或 400/410/450 分支 | `target-dependent` | P0 只建符号和语义边界，不承诺 DTE/硬件正确性 | header symbol index、Clang 对照验证、目标分支差分 |

#### 测试证据

| 测试证据 | 已观察范围 | 目标前提 | 状态 | 限制 | 验证方法 |
| --- | --- | --- | --- | --- | --- |
| `llvm-project/tops/integration_test/cases/language/STATUS.md` | 记录 C++11/14/17 共 60 个编译 case、默认 GTest 回归和 4 个 device coverage exception | 文档记录的 TopsPlatform/topscc/gcusim 版本 | `verified` | 本规格不把 status 记录当作当前重新执行结果 | 按记录的 case 子集重跑，并分别报告 exception |
| `llvm-project/tops/integration_test/cases/language/exec_spec/exec_space_specifiers/main.cc` | `__device__`、inline/noinline、`__host__ __device__`、`__constant__`、`__shared__` 的端到端测试材料 | EFGCU integration test | `verified` | 未覆盖的 `__cooperative__`、`__sp__` 仍为候选 | 编译、launch、同步和 host/device 结果核对 |
| `llvm-project/tops/integration_test/cases/language/launch_config/launch_bounds_maxnreg/main.cc` | `__launch_bounds__`、`__maxnreg__` 正向 kernel 和运行时结果 | EFGCU integration test | `verified` | 注释中的架构描述不替代实际 target 参数 | 读取实际 build command，重复正向 tests |
| `llvm-project/tops/integration_test/cases/language/builtin_vars/grid_info/main.cc`、`llvm-project/tops/integration_test/cases/language/builtin_vars/lane_id/main.cc` | grid/block/thread 内建变量和 lane-id fallback 的使用 | integration 测试材料的实际 target | `verified` | `warpSize` 需按 EFGCU/regular GCU 分开 | 编译和结果校验，补 profile matrix |
| `llvm-project/tops/integration_test/cases/language/vector_types/builtin_add/main.cc` | builtin vector types、成员访问、make 函数、dim3 launch 的端到端使用 | integration 测试材料的实际 target | `verified` | 文件注释中的未覆盖类型不能自动升级状态 | 编译、运行、AST 和目标差分 |
| `llvm-project/tops/integration_test/cases/language/memory_space/restrict_qualifier/main.cc`、`llvm-project/tops/integration_test/cases/language/exec_spec/exec_space_specifiers/main.cc` | `__restrict__`、`__shared__`、`__constant__` 的现有使用 | integration 测试材料 | `verified` | `__local__`、`__private__`、`__cluster_shared__` 需补负向/目标边界 | 语法、地址空间和诊断测试材料 |
| `llvm-project/clang/test/DTU_test/tcle/vector_op_parser.cc`、`llvm-project/clang/test/DTU_test/tcle/maskbit_parser.cc`、`llvm-project/clang/test/DTU_test/tcle/countl_zero_parser.cc` | GCU400/GCU450 的 vector、maskbit、overload negative diagnostics | `--target=gcu -mcpu=gcu400/gcu450` | `verified` | 这些是 Clang 对照验证，不是 Go server 测试 | 保存 expected diagnostic 并执行 Go 差分 |
| `llvm-project/clang/test/CodeGenEFGCU/launch-bounds-e2e.cc`、`llvm-project/clang/test/CodeGenEFGCU/quanlifier/maxnreg-error.cc`、`llvm-project/clang/test/CodeGenEFGCU/cluster_dims.cc` | EFGCU500 启动/资源属性正向、边界和超限 negative tests | `efgcu-enflame-tops -mcpu=efgcu500` 或等价 offload arch | `verified` | CodeGen 结果不替代 P1 LSP 行为 | `FileCheck`、Go parser/semantic 测试材料 |
| `llvm-project/clang/test/DTU_test/topscc/attribute/launch_bounds.tops`、`llvm-project/clang/test/DTU_test/topscc/attribute/maxnreg.tops` | Tops attribute 解析和属性输出检查 | `.tops`/Tops device | `verified` | 覆盖的是属性 对照验证，不是编辑器增量错误恢复 | `llvm-lit` 和 P1 incremental tests |
| `llvm-project/clang/test/CodeGenEFGCU/vector_types/builtin_vector_types.cc`、`llvm-project/clang/test/DTU_test/tcle/Draco/*` | EFGCU500 vector types、DTE、barrier、async group 等目标相关测试入口 | EFGCU500 | `target-dependent` | P0 仅纳入符号/目标边界，不把硬件执行能力纳入 P1 | 目标矩阵和 `-fsyntax-only`/CodeGen 差分 |

### 八个能力域语法清单

以下每行是一个可独立进入 `SyntaxCapability` 的清单项。状态是证据状态，不是 Go 服务器实现状态。

| ID / 能力域 | 语法清单 | 目标前提 | 源码证据 | 测试证据 | 状态 | 限制 | 验证方法 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `TOPS-STD-001` 标准 C++ | C++11：`alignas`、`alignof`、alias templates、atomics、attributes、`auto`、C99 features、char16/32、`constexpr`、`decltype`、default template args、defaulted/deleted members、delegating/inheriting constructors、enum class、explicit conversion、initializer list、inline namespace、lambda、local type template、move semantics、NSDMI、`nullptr`、`override/final`、range-for、right-angle `>>`、rvalue refs、`sizeof...`、`static_assert`、trailing return、unrestricted union、variadic templates；C++14：binary literals、contextual conversions、deprecated attributes、digit separators、generalized/generic lambdas、member-init aggregates、relaxed constexpr、return type deduction、sized deallocation、variable templates；C++17：aggregate base init、attributes、auto NTTP、constexpr lambda、CTAD、fold expressions、guaranteed copy elision、hex float、`if constexpr`、init statements、inline variables、nested namespaces、`std::byte`、structured bindings | `tops` language 加 C++ 基础；实际 `-std` 和 device/host pass 来自上下文 | `tops/integration_test/cases/language/{cxx11,cxx14,cxx17}`、Clang C++ parser | `language/STATUS.md` 记录 60/60 compile 和 4 个 exception | `verified` | BUG-2 recursive variadic、BUG-4 extern/explicit template、BUG-5 device TLS、BUG-6 device sized delete；C++11/14 status 使用兼容的更高 GTest 编译标准，不能隐藏真实标准来源 | 逐 case 读取 compile flags，运行 host/device 分离结果，P1 再做 Go AST/semantic 差分 |
| `TOPS-EXEC-001` 执行空间 | `__global__`、`__device__`、`__host__`、`__host__ __device__`、`__cooperative__`、`__sp__`、`__scalar_only__`、`__noinline__`、`__forceinline__`、`__alwaysinline__`、`__force_noinline__`、`__inline_hint__` | `TOPS` language；device 语义需 device pass；`__sp__` 需验证 `--tops-sp` 与未启用两种模式 | `llvm-project/clang/lib/Headers/tops/__tops_defines.h`、`llvm-project/clang/lib/Headers/tops/trt/host_defines.h`、`llvm-project/clang/include/clang/Basic/Attr.td` | `llvm-project/tops/integration_test/cases/language/exec_spec/exec_space_specifiers/main.cc` 覆盖主要执行和 inline 组合；Clang Sema CUDA tests 可作调用边界 对照验证 | `verified` for covered subset; `candidate` for `__cooperative__`/`__sp__`/`__scalar_only__` | 现有测试材料没有证明所有目标和调用边界；客户端不得自行判断 host/device | 有效、非法调用、未闭合 attribute、`--tops-sp`/target 切换四类测试材料；与 Clang expected diagnostics 差分 |
| `TOPS-MEM-001` 存储空间 | `__device__` global storage、`__constant__`、`__shared__`、`__cluster_shared__`、`__local__`、`__local_stack__`、`__private__`、`__mmu_pointer__`、`__restrict__`、`__shared_dte__`、`__local_dte__`、`__private_dte__` | `__GCU_ARCH__` 完整分支：`__private_dte__` 仅 `<400`，`__local_dte__`/相关资源仅 `>=400`；EFGCU shared/cluster 分支需单独 profile | `llvm-project/clang/lib/Headers/tops/__tops_defines.h`、`llvm-project/clang/lib/Headers/tops/topscc_types.h`、`llvm-project/clang/lib/Headers/tops/__tops_dte.h` | `llvm-project/tops/integration_test/cases/language/exec_spec/exec_space_specifiers/main.cc`、`llvm-project/tops/integration_test/cases/language/memory_space/restrict_qualifier/main.cc`、`llvm-project/clang/test/CodeGenEFGCU/quanlifier/shared.cc`、`llvm-project/clang/test/CodeGenEFGCU/quanlifier/cluster_shared.cc`、device ABI address-space cases | `target-dependent` | storage qualifier 可同时涉及声明位置、指针类型、初始化和硬件资源；P1 不证明 DTE 运行正确性 | 对每个 profile 做声明、指针转换、非法初始化和缺失 target 的 `-fsyntax-only`/LSP 差分；保留 address space 来源 |
| `TOPS-LAUNCH-001` 启动/资源 | `__thread_dims__(...)`、`__cluster_dims__(...)`、`__launch_bounds__(...)`、`__maxnreg__(N)`、`__block_tile__` | `__thread_dims__` 的 header 分支为 `__GCU_ARCH__ >= 300`；`__maxnreg__` 为 `>=400`；`__cluster_dims__`/EFGCU 需目标属性验证 | `llvm-project/clang/lib/Headers/tops/__tops_defines.h`、`llvm-project/clang/include/clang/Basic/Attr.td`、`llvm-project/clang/test/DTU_test/topscc/attribute/launch_bounds.tops`、`llvm-project/clang/test/DTU_test/topscc/attribute/maxnreg.tops` | `llvm-project/tops/integration_test/cases/language/launch_config/launch_bounds_maxnreg/main.cc`；`llvm-project/clang/test/CodeGenEFGCU/launch-bounds-e2e.cc`、`launch-bounds-edge-cases.cc`、`cluster_dims.cc`、`quanlifier/maxnreg-error.cc`；`llvm-project/clang/test/DTU_test/clang/topscc_sema/block_tile.tops` | `target-dependent` | 参数范围、资源限制和 CodeGen metadata 不能仅由 tokenizer 判断；测试材料的注释不替代实际 target | 正向、参数超限、属性不完整、GCU300/400/500/EFGCU500 profile 边界；检查诊断 code、range 和 profile |
| `TOPS-VEC-001` 向量/数值 | `__vector`、`__vector2`、`__vector4`、`__vector8`、`__vector bool`；builtin `char1`/`char2`/`char3`/`char4`、`short*`、`int*`、`long*`、`longlong*`、`float*`、`dim3`、`uint3`；`__fp16`、`__bf16`、FP8 E4M3/E5M2/E8M0、FP4、FP6、`float8_base`；`__valigned__` | 向量宽度、对齐和数值类型受 `__GCU_ARCH__`、`__EFGCU_ARCH__`、active headers 与 device pass 影响；`__valigned__` 在活动 `__tops_defines.h` 分支中为 GCU `<400` 与 `>=400` 两种值 | `llvm-project/clang/lib/Headers/tops/vector_types.h`、`llvm-project/clang/lib/Headers/tops/vector_functions.h`、`llvm-project/clang/lib/Headers/tops/__tops_defines.h`、`llvm-project/clang/lib/Headers/tops/{ef_,tops_}fp*.h`、`llvm-project/clang/lib/Headers/tcle.h` | `llvm-project/tops/integration_test/cases/language/vector_types/builtin_add/main.cc`；`llvm-project/clang/test/DTU_test/tcle/vector_op_parser.cc`、`maskbit_parser.cc`；EFGCU vector tests | `target-dependent` | vector width/element count、对齐和可用 operator 不能从 GCU400 推到 GCU450/GCU500/EFGCU500；只做静态语义，不做性能或 CodeGen 等价承诺 | AST 类型宽度、成员补全、非法转换、对齐属性和 profile 差分；Clang 只作 对照验证 |
| `TOPS-BUILTIN-001` 内建变量 | regular GCU：`threadIdx.{x,y,z}`、`blockIdx.{x,y,z}`、`blockDim.{x,y,z}`、`gridDim.{x,y,z}`、`threadDim.{x,y,z}`、`subThreadIdx.{x,y,z}`；EFGCU：前四者、转换为 `dim3/uint3`、`warpSize`；lane id：`__builtin_efvm_lane_id`、`__builtin_efgcu_lane_id`、`__lane_id` 候选 | regular GCU 的 `threadDim/subThreadIdx` 由 `__GCU_ARCH__ >=300` 分支；EFGCU 由 `__EFGCU_ARCH__=500` header；lane builtin 由 `__has_builtin` 条件决定 | `llvm-project/clang/lib/Headers/tops/__tops_builtins.h`、`llvm-project/clang/lib/Headers/tops/__tops_efgcu_builtin_vars.h` | `llvm-project/tops/integration_test/cases/language/builtin_vars/grid_info/main.cc`、`llvm-project/tops/integration_test/cases/language/builtin_vars/lane_id/main.cc`、DTE/topscc_types 中使用示例 | `target-dependent` | `warpSize` header 明确记录当前 builtin 限制，regular GCU 与 EFGCU 不能合并；内建变量禁止构造、赋值和取地址 | 符号/成员补全、host 侧非法访问、未完成成员访问和目标可见性矩阵；执行结果只作辅助 |
| `TOPS-TCLE-001` TCLE/API | `tcle` namespace 和 `__tcle_ai`/`__tcle_ti`/`__target`；vector make functions；`tops::shared_dte`、`local_dte`、`private_dte`、`mdspan`；`tops::pipeline`、queue、barrier、`__syncthreads`；DTE trigger/wait/event；TCLE conversion、vector、math、GEMM/FP API 名称 | headers、`__TOPS__`、device pass 和 `__GCU_ARCH__` 分支；`topscc_types.h` 明确只在 `__GCU_ARCH__ <500` 入口，GCU500/EFGCU 使用其他 API 集合 | `llvm-project/clang/lib/Headers/tcle.h`、`llvm-project/clang/lib/Headers/tops/__tops_dte*.h`、`llvm-project/clang/lib/Headers/tops/topscc_types.h`、`llvm-project/clang/lib/Headers/tops/__tops_pipeline.h`、`llvm-project/clang/lib/Headers/tops/__tops_sync.h` | `llvm-project/clang/test/DTU_test/tcle/**`、`llvm-project/clang/test/DTU_test/tcle/Draco/**`、现有 vector/DTE integration cases | `target-dependent` | P0 只要求可解析符号、目标条件和基础调用边界；不承诺 DTE、barrier、GEMM 的硬件时序、资源分配或性能 | header index、API overload/target diagnostics、有效/非法/不完整调用和 profile 差分；硬件 workload 不属于 P0 |
| `TOPS-TARGET-001` 目标条件 | 文件/模式：`.tops`、`.cpp + -Tops`、`-x tops`；参数：`-mcpu=gcu300/400/410/450/500`、`-mcpu=efgcu500`、`--cuda-gpu-arch`、`--offload-arch`、`--cuda-device-only`、`--offload-device-only`、`--tops-sp`、`--tops-simt`；宏：`__GCU_ARCH__`、`__AGCU_ARCH__`、`__EFGCU_ARCH__`、`__TOPS_DEVICE_COMPILE__`、`__TOPS_ARCH__`、`__TOPS_SIMT_MODE__` | 必须先确定 target triple、CPU、host/device pass、active include roots，再选择完整 `#if/#elif/#else` 分支 | `llvm-project/clang/include/clang/Driver/Types.def`、`Options.td`、`LangStandards.def`、`llvm-project/clang/lib/Basic/Targets/DTU.cpp`、`EFGCU.cpp`、`llvm-project/clang/lib/Driver/ToolChains/Clang.cpp`、headers 和 target tests | `.tops`/`-Tops`/`-x tops` tests、`llvm-project/clang/test/CodeGenEFGCU/**`、`llvm-project/clang/test/DTU_test/**` 多 profile RUN lines | `target-dependent` | 宏存在、宏值和实际可见声明必须以命令输出为准；不允许用相邻架构数值补全当前 profile | 每个候选 profile 执行 `-###`、`-dM -E`、`-H` 或等价 include 审计、`-fsyntax-only`，并与 Go context 记录比对 |

### 六个候选 TargetProfile

profile 是语言分析输入候选，不是硬件能力认证。所有 profile 在 P0 完成前保持 `candidate`；只有命令、active headers、宏输出和适用测试全部对齐后，才可提升为 `validated`（该状态属于实现计划，不在本规格中提前宣称）。

| Profile ID | 候选输入与 target | 已核对宏/语言 | 目标前提 | 状态 | 限制 | 验证方法 |
| --- | --- | --- | --- | --- | --- | --- |
| `gcu300-default` | `--target=gcu-enflame-tops -mcpu=gcu300`；同时核对 `-Tops --cuda-gpu-arch=gcu300 --cuda-device-only` | `__GCU_ARCH__=300`；language=`tops` | GCU300 device pass，Tops headers 在 target compiler resource dir | `candidate` | regular GCU 的 `<400` DTE/对齐/内建分支不能套用到新代；实际 canonical driver 形式待确认 | `clang -###`、`-dM -E`、`-fsyntax-only`，检查 `.tops` 和 `-x tops` |
| `gcu400-default` | `--target=gcu-enflame-tops -mcpu=gcu400`；或 `-Tops --cuda-gpu-arch=gcu400 --cuda-device-only` | `__GCU_ARCH__=400`；language=`tops` | GCU400 device pass；DTE/pipeline header 分支为 400 | `candidate` | VLIW/dual-VPT、DTE 和 vector width 不能推断为 GCU450/500 行为 | `clang/test/DTU_test` 的 gcu400 RUN lines、宏和 syntax 差分 |
| `gcu410-default` | `--target=gcu-enflame-tops -mcpu=gcu410`；或 `-Tops --cuda-gpu-arch=gcu410 --cuda-device-only` | `__GCU_ARCH__=410`；language=`tops` | GCU410 device pass；必须保留 410 的完整 header 分支 | `candidate` | 不能因为 headers 在 400/410 共用部分分支就宣称完全等价 | `-dM -E`、include trace、410-specific/兼容测试和 Go target matrix |
| `gcu450-default` | `--target=gcu-enflame-tops -mcpu=gcu450`；或 `-Tops --cuda-gpu-arch=gcu450 --cuda-device-only` | `__GCU_ARCH__=450`；language=`tops` | GCU450 device pass；vector/GEMM/TCLE target branch | `candidate` | GCU450 的 target-dependent vector/TCLE 诊断不能使用 GCU400 默认值；DTE/同步 API 需逐项确认 | `tcle/countl_zero_parser.cc`、LIBRA gcu450 tests、宏和 syntax 差分 |
| `gcu500-default` | `--target=gcu-enflame-tops -mcpu=gcu500`；或 `-Tops --cuda-gpu-arch=gcu500 --cuda-device-only` | `__GCU_ARCH__=500`；language=`tops` | GCU500 device pass；`__GCU_ARCH__ >=500` 分支和 target-specific TCLE | `candidate` | `topscc_types.h` 的 `<500` 入口不能当作 GCU500 API；需区分 GCU500 与 EFGCU500 | `clang/test/DTU_test/tcle/Draco/**`、宏、include 和 syntax 差分 |
| `efgcu500-default` | `--target=efgcu-enflame-tops -mcpu=efgcu500`；或 `-Tops --cuda-gpu-arch=efgcu500 --cuda-device-only`/`--offload-arch=efgcu500` | `__EFGCU_ARCH__=500`；EFGCU TargetInfo；language=`tops` | EFGCU500 device pass；EFGCU headers、SIMT/offload 选项 | `candidate` | 不把 `__EFGCU_ARCH__` 替换为 `__GCU_ARCH__`；regular GCU builtins、DTE 和 EFGCU builtins/API 必须分开 | `clang/test/CodeGenEFGCU/**` 的实际 RUN lines、`-###`、host/device `-dM -E`、syntax 差分 |

#### Profile 共同规则

- profile 的 `compiler`、resource dir、include roots 和版本必须来自实际命令，不允许使用系统 LLVM 代替当前 checkout 的工具链。
- `-Tops`、`.tops` 和 `-x tops` 只确定输入语言路径；target CPU、device pass、include roots 和宏仍必须记录。
- 对同一 profile，host pass 与 device pass 生成的宏集合可以不同；`CompilationContext` 必须保留 pass 类型。
- `__GCU_ARCH__` 的完整分支为 300、400、410、450、500；EFGCU 分支单独定义 `__EFGCU_ARCH__=500`。若命令没有产出预期宏，状态为 `missing` 或 `invalid`，不回退到默认 profile。

### CompilationContext 字段与优先级

`CompilationContext` 是某个 `SourceDocument` 的文件级语义上下文。它不是用户可见的 LSP 扩展消息，也不是 Clang 命令的字符串缓存；它必须同时保留原始证据、归一化结果和解析状态。

| 字段 | 内容与来源优先级 | 目标前提 | P0 状态 | 限制 | 验证方法 |
| --- | --- | --- | --- | --- | --- |
| `document_uri` | LSP 文档 URI；用于选择编译数据库条目和结果归属 | URI 可规范化 | `required` | URI 别名不能导致两个 context 互相覆盖 | 同一文件的相对/绝对 URI 选择测试 |
| `workspace_root` | 当前 workspace root；多根工作区必须带 root 标识 | client 初始化已完成 | `required` | 不能用全局 root 解释多根工作区 | 多根同名文件测试材料 |
| `compile_commands_path` | 查找并审计 `compile_commands.json` 的路径 | 文件可读 | `required` | 不存在时转 fallback，不伪造条目 | 缺失/多个数据库路径测试 |
| `compile_command_entry` | 按规范化文件路径选择的完整条目；同文件多条冲突则 `invalid` | database 条目有效 | `required` | 不能只取 `command` 的片段而丢失 flags | 多条 entry 和 response-file 测试 |
| `source` | `compile_commands` 或 `workspace_settings`；有效数据库条目优先 | 至少有一个来源 | `required` | 无来源时为 `missing` | 来源字段和诊断消息检查 |
| `raw_arguments` | 原始参数列表/拆分后的 command，保留次序、引号语义和审计值 | entry 或显式设置可读 | `required` | 不允许只保留标准和 CPU 而丢 include/宏/device flags | raw vs normalized snapshot |
| `normalized_arguments` | 路径归一化、重复项保留规则、response file 展开后的分析参数 | parser 能识别参数 | `required` | 归一化不能改变明确参数含义 | 与 `clang -###`/driver 展开结果对比 |
| `input_kind` | `.tops`、普通 C++、`-Tops`、`-x tops` 等输入形式 | driver 能确定 language | `required` | 扩展名不能覆盖明确 `-x`/`-Tops` | 三种输入形式的预处理和语法差分 |
| `language_standard` | driver/参数解析出的 `tops` language 与 C++ standard；不使用隐藏默认 | `tops`/`-std` 已解析 | `required` | `STATUS.md` 的 C++11/14/17 label 与 GTest compile flag 必须分别记录 | `-dM -E`、AST feature 测试材料 |
| `target_profile_id` | 引用六个候选 profile 之一；未知值为 `invalid` | CPU/offload arch 已确认 | `required` | 不允许从文件名、机器型号或相邻 profile 推断 | profile table 与 context snapshot 对照 |
| `target_triple` | driver/cc1 实际 target triple，例如 GCU/EFGCU target | target 已解析 | `required` | `gcu` 与 `gcu-enflame-tops` 的差异必须保留 | `clang -###` 和日志字段 |
| `cpu_or_offload_arch` | `-mcpu`、`--cuda-gpu-arch` 或 `--offload-arch` 的原始值 | target profile candidate | `required` | `efgcu500` 不可归一成 `gcu500` | 六 profile 逐项参数解析 |
| `pass_kind` | host、device 或 offload host/device pass | Tops/EFGCU driver pipeline | `required` | host/device 宏不可合并 | 分 pass `-dM -E` |
| `compiler` | compiler executable、版本、resource dir、driver mode | 本地工具可执行 | `required` | 不得默认为系统 `clang`，也不得在 Go 运行时调用 clangd | `--version`、`-print-resource-dir`、日志 |
| `include_roots` | Tops headers、Clang headers、项目 include 和系统 include，保留搜索顺序 | include 路径存在 | `required` | header checkout 不一致必须 `invalid` 或 `partial` | `-H`/等价 include trace、文件存在检查 |
| `predefined_macros` | driver 和命令注入的宏，至少审计 `__GCU_ARCH__`、`__AGCU_ARCH__`、`__EFGCU_ARCH__`、`__TOPS_DEVICE_COMPILE__`、`__TOPS_ARCH__`、`__TOPS_SIMT_MODE__` | pass/target 已确定 | `required` | 宏值必须来自预处理输出，不能由 profile 名称代填 | `-dM -E` 与 context 比对 |
| `device_flags` | `--cuda-device-only`、`--offload-device-only`、`-fcuda-is-device`、`--tops-sp`、`--tops-simt` 等 | 具体 pipeline 已确定 | `required` | 缺失 device flag 时不能提供完整 device 语义 | `clang -###` 和 profile 测试材料 |
| `working_directory` | compile command 的工作目录及 response/include 相对路径基准 | entry 有效 | `required` | 不得用 workspace root 覆盖明确目录 | 相对路径 include 测试材料 |
| `comparison_config` | P0/P1 离线对照使用的 Clang 路径、命令模板和版本；不属于 Go runtime dependency | 只在验证任务中使用 | `documentation-only` | 不能作为 server fallback 或语义后端 | 离线对照命令可复现性检查 |
| `resolution_status` | `resolved`、`partial`、`missing`、`invalid`；诊断必须与状态同时保存 | 必填 | `required` | `partial` 结果必须标为 `analysis-degraded` | 缺失、冲突、成功四类 context 测试 |
| `context_version` | 每次文本、配置、profile 或 header 根变化递增 | 服务器管理状态 | `required` | 旧 LSP 结果不能覆盖新版本 | stale diagnostics/completion test |
| `last_verified` | context 最后一次完成 driver/header/宏核对的时间 | `resolved` 或 `partial` | `required` | 时间不是语义证据本身 | 修改数据库后检查失效和重新核对 |

#### CompilationContext 优先级

1. **文件级有效 `compile_commands` 条目**：按规范化 URI/路径匹配；条目中的明确 language、target、standard、include、宏和 device flags 是最高优先级。
2. **driver 派生字段**：从第 1 层的 raw arguments 得到 target triple、pass kind、宏和 active include 顺序。它们是可审计的派生事实，不是工作区设置可以覆盖的值。
3. **工作区设置 fallback**：仅当文件没有有效编译数据库条目时使用；设置必须显式提供需要的字段，来源标为 `workspace_settings`。
4. **无隐藏语义默认值**：没有 target、include 根或 language standard 时，不使用 GCU300、系统 include 或 clangd 作为默认；状态为 `missing`/`partial`，用户可见结果必须说明缺少什么。
5. **冲突处理**：数据库和 fallback 有冲突时数据库优先；同一数据库内部的明确参数互相冲突，或 workspace fallback 内部冲突时为 `invalid`，不得静默合并。
6. **失效处理**：文件变更、配置变更、数据库变更、profile 变更或活动 header 变化使 context 和相关分析结果变为 stale；旧诊断不能覆盖新版本。

### Go 服务器责任边界

P0 规定的是责任和可验证行为，不规定 Go package 名称、第三方 parser/LSP 库或具体数据结构。数据流为：源文本与 `CompilationContext` 进入 Go server，自有 tokenizer/parser/AST/index/semantic 产生结果，再由 LSP 层编码为标准 LSP 响应；Clang 对照验证 位于验证流程之外。

| 层 | 负责 | 不负责 | 目标前提 | P0 状态 | 限制 | 验证方法 |
| --- | --- | --- | --- | --- | --- | --- |
| `tokenizer` | 识别标准 C++/Tops 关键字、标识符、字面量、注释、预处理 token、attribute 拼写和稳定 source range；对未闭合 token 保留可恢复结果 | 不决定 host/device 调用合法性、目标资源限制、诊断等级或 LSP payload | 可选 context；token 级恢复不依赖完整 target | `planned` | 宏展开/条件代码必须保留足够来源信息，不能把 inactive text 当活动声明 | token snapshot、错误输入和增量输入测试材料 |
| `parser` | 将 token 组织为声明、类型、函数、attribute、模板、表达式、预处理分支和错误恢复节点；保留未完成结构 | 不执行目标语义、CodeGen、DTE 资源分配或 LSP 请求 | target-neutral grammar；Tops attribute 词法可被识别 | `planned` | P1 只承诺覆盖矩阵中的语法，不承诺完整 ISO C++ 或汇编 grammar | valid/invalid/incomplete AST 测试材料，与 Clang syntax 对照验证 差分 |
| `AST` | 记录声明、类型、scope、attribute、调用、模板和 source range；保留 Tops 扩展 spelling 与目标条件来源 | 不生成诊断文本、不选 profile、不管理文档生命周期 | parser 产生可恢复 tree | `planned` | 不能丢失宏来源、原始 token 和不完整节点，否则无法给出可解释诊断 | AST serialization、range 稳定性和编辑后局部更新 |
| `symbol index` | 建立当前文档、workspace 文件和配置 header 中的声明、定义、引用、scope、成员和 overload 索引；按 context/header 版本失效 | 不决定语义合法性、目标 feature gate、诊断 severity 或 LSP 展示 | include roots 和宏条件已解析；缺失时标记不完整 | `planned` | header 条件分支不可在无 profile 时合并；索引不等于可调用 | definition/references/documentSymbol、header 变更和 stale index |
| `semantic` | 使用 AST、index、`CompilationContext` 和 `TargetProfile` 判断标准/Tops 语义、执行空间调用边界、存储空间、属性参数、向量/数值类型、builtin 可见性、目标条件；生成诊断、completion、hover 和导航事实 | 不 tokenize、不维护 LSP transport、不由客户端重判；不调用 clangd 取得运行时答案 | `resolved` context；`partial` 只能产生受限分析 | `planned` | DTE/GEMM/同步只做声明和目标边界，不证明硬件时序、CodeGen 或性能 | Go semantic tests、Clang syntax/diagnostic differential、32 类 P1 scenarios |
| `LSP` | 管理 initialize/shutdown、文档同步、context 选择、请求取消、结果版本、diagnostics/completion/hover/definition/references/documentSymbol 编解码和错误 code | 不复制 tokenizer/semantic 规则，不把 client setting 变成隐式语义 | server state 和 context resolver 可用 | `planned` | stale/timeout/cancel 必须有一致行为；扩展消息若新增必须版本化 | LSP contract tests、stale/cancel/error-path integration |
| `Clang 对照验证` | 在 P0/P1 验证中提供预处理、Sema、AST/CodeGen 或 diagnostic 差分参考 | 不进入 Go server runtime，不作为 clangd sidecar、fallback 或最终语义所有者 | 本地 Clang/Tops checkout 和精确 target 参数 | `documentation-only` | 对照验证 与 Go 结果差异需要记录原因，不允许直接把 对照验证 输出转发给用户 | `-###`、`-dM -E`、`-fsyntax-only`、适用 `llvm-lit` |
| `TypeScript VS Code client` | 激活、启动/停止/重启 Go server、发送 workspace settings 和文档通知、展示状态/日志/LSP 结果、执行用户命令 | 不判断 target feature、host/device 合法性、类型可用性或诊断 severity | LSP contract 和 server executable 配置 | `out-of-scope for P0 implementation` | 本 P0 不创建 TypeScript；客户端边界只作为后续契约 | P1/P3 客户端契约测试，不在 P0 构建 |

### P1 测试场景矩阵

下表中的 `existing` 是当前 checkout 已存在的源码/测试证据；`new P1 测试材料` 表示 P1 必须新增的最小场景，不把它误报为当前已经存在。每个场景均明确目标前提、状态、限制和验证方法。

#### 标准 C++

- **P1-STD-V（有效）**：`language/cxx11`、`cxx14`、`cxx17` 中现有 case 在对应 `-std`/Tops device context 下解析，保留 `STATUS.md` 的 4 个 exception。目标前提：标准和 device pass 已知。状态：`required`。限制：回归通过不等于 full device pass。验证：Go AST/semantic 结果与现有 status、Clang syntax 结果逐 case 对照。
- **P1-STD-I（无效）**：new P1 测试材料含缺失分号、错误模板参数、类型不匹配和 host/device 函数错误调用。目标前提：至少 gcu400 与 efgcu500。状态：`required`。限制：诊断必须区分 parser error 与 semantic error。验证：稳定 code/range/message 与 Clang expected diagnostic 对照。
- **P1-STD-P（不完整）**：new P1 测试材料在注释、字符串、模板参数、函数体和 initializer 中逐字符截断。目标前提：任一 resolved profile。状态：`required`。限制：不因恢复失败清空已有 index。验证：增量 `didChange` 序列、AST 可恢复节点和 stale 诊断检查。
- **P1-STD-T（目标边界）**：同一 C++11/14/17 测试材料分别用 status 记录的标准 flags、host/device pass 和六个 candidate profiles。目标前提：profile 宏和 include 根已记录。状态：`required`。限制：BUG-2/4/5/6 必须单列。验证：profile matrix、`-dM -E`、Clang syntax 和 Go 结果差分。

#### 执行空间

- **P1-EXEC-V（有效）**：现有 `exec_spec/exec_space_specifiers/main.cc` 的 `__device__`、inline/noinline、`__host__ __device__`、kernel 调用和同步。目标前提：EFGCU integration profile。状态：`required`。限制：只证明测试材料覆盖的组合。验证：parser AST、调用关系、completion/hover 和 Clang syntax。
- **P1-EXEC-I（无效）**：new 测试材料从 host 调 device、从 device 调 host、把 kernel 当普通 device 函数调用、把执行属性放到非法声明。目标前提：gcu400、gcu450、efgcu500。状态：`required`。限制：诊断不应只显示未知 attribute。验证：Sema 对照验证、semantic diagnostics 和 definition context。
- **P1-EXEC-P（不完整）**：new 测试材料截断 `__global__`/`__device__` 声明、attribute 参数和 `__host__ __device__` 函数体。目标前提：任何可解析 Tops context。状态：`required`。限制：保留函数名和参数符号。验证：逐字符 `didChange`、completion 和 parser recovery。
- **P1-EXEC-T（目标边界）**：比较 `__sp__` 在 `--tops-sp`/未启用、`__cooperative__` 与不同 target、`__scalar_only__` 在 host/device 的结果。目标前提：对应 driver flags 已由 P0 命令核对。状态：`required`。限制：若源码/测试不足，结果为 `candidate` 或 `unsupported`，不得猜测。验证：宏/driver 输出、Clang 对照验证、profile-specific LSP diagnostics。

#### 存储空间

- **P1-MEM-V（有效）**：现有 `__constant__`、`__shared__`、`__restrict__` 测试材料，以及 new 测试材料的 `__local__`/`__private__` 声明和合法指针使用。目标前提：匹配 regular GCU/EFGCU header。状态：`required`。限制：不验证真实内存性能。验证：AST address-space、符号 hover 和 `-fsyntax-only`。
- **P1-MEM-I（无效）**：非法 storage qualifier 位置、shared/private 指针混用、动态初始化限制、非匹配 address-space 转换。目标前提：gcu300、gcu400、efgcu500。状态：`required`。限制：DTE resource failure 不应伪装成普通类型错误。验证：Clang Sema/address-space 对照验证 与稳定诊断 code。
- **P1-MEM-P（不完整）**：截断 qualifier、数组声明、模板 pointer type 和 `__shared__` initializer。目标前提：已解析 language，target 可缺失。状态：`required`。限制：target 缺失时只能给 `analysis-degraded`。验证：增量 AST、completion 和 context diagnostic。
- **P1-MEM-T（目标边界）**：对比 gcu300 的 `__private_dte__` 与 gcu400/410/450 的 `__local_dte__`/shared DTE 分支，并单独比较 EFGCU `__cluster_shared__`。目标前提：完整 `__GCU_ARCH__`/`__EFGCU_ARCH__` 分支。状态：`required`。限制：不能以 GCU500 `<500` header 入口推断 EFGCU。验证：预处理、include trace、目标 negative tests。

#### 启动/资源

- **P1-LAUNCH-V（有效）**：现有 `launch_bounds_maxnreg/main.cc` 与 `CodeGenEFGCU/launch-bounds-e2e.cc` 的 `__launch_bounds__`、`__maxnreg__`、组合属性。目标前提：EFGCU500 或对应 GCU target。状态：`required`。限制：运行结果不等于 LSP 语义已实现。验证：属性 AST、参数 hover、Clang/FileCheck 对照验证。
- **P1-LAUNCH-I（无效）**：使用 `maxnreg-error.cc` 的超限值、非法参数数量、非 kernel 属性和 `__block_tile__` 非法主体。目标前提：EFGCU500/GCU target 各自的限制。状态：`required`。限制：限制值必须来自实际 target，不能跨代复制。验证：expected diagnostics、profile-specific code/range。
- **P1-LAUNCH-P（不完整）**：截断 `__thread_dims__(`、`__cluster_dims__(1,`、`__launch_bounds__(128,` 和 `__maxnreg__(`。目标前提：parser 可恢复；target 可暂缺。状态：`required`。限制：不把不完整参数诊断成目标不支持。验证：逐字符编辑、AST recovery、completion。
- **P1-LAUNCH-T（目标边界）**：比较 `__thread_dims__` 的 GCU300/400 可见性、`__maxnreg__` 的 `<400`/`>=400` 分支、GCU450/500 和 EFGCU500 的 launch/cluster 限制。目标前提：宏已由 `-dM -E` 确认。状态：`required`。限制：target 缺失时输出 `missing-compilation-context`。验证：六 profile 预处理、Clang negative/CodeGen 和 Go semantic matrix。

#### 向量/数值

- **P1-VEC-V（有效）**：现有 `vector_types/builtin_add/main.cc` 的 builtin vector、成员访问、make functions、`dim3`，并以 `tcle` parser 测试材料补充 `__vector`/`__bf16`。目标前提：GCU400/450 和 EFGCU500 分别配置 headers。状态：`required`。限制：现有 integration 注释中的未覆盖类型仍需保持候选。验证：AST 类型、completion、hover 和 syntax 对照验证。
- **P1-VEC-I（无效）**：使用 `vector_op_parser.cc`、`maskbit_parser.cc` 的错误向量宽度、标量/向量非法转换、bool vector 初始化和不匹配 operator。目标前提：gcu400/gcu450。状态：`required`。限制：诊断需要包含实际元素数和 profile。验证：Clang expected diagnostics 与 Go type checker 差分。
- **P1-VEC-P（不完整）**：截断 `__vector float`、模板 vector type、成员 `.x`/`.y` 和 FP8/BF16 初始化。目标前提：include 根可用；target 可暂缺。状态：`required`。限制：类型未闭合时不产生级联错误海啸。验证：incremental parse、completion 和诊断去重。
- **P1-VEC-T（目标边界）**：比较 `__valigned__` 的 GCU `<400` 与 `>=400` 分支、GCU400/450 vector width、GCU500/EFGCU500 FP4/FP6/FP8 可见性。目标前提：完整 header branch 和 active include。状态：`required`。限制：不得把历史文档的宽度表当成当前事实。验证：预处理、AST layout、target tests 和 profile diff。

#### 内建变量

- **P1-BUILTIN-V（有效）**：现有 `builtin_vars/grid_info/main.cc` 的 `threadIdx`/`blockIdx`/`blockDim`/`gridDim`，以及 `lane_id/main.cc` 的 lane builtin fallback。目标前提：对应 device header。状态：`required`。限制：运行值只作辅助，LSP 重点是符号/member 语义。验证：symbol index、member completion、hover 和 syntax。
- **P1-BUILTIN-I（无效）**：new 测试材料在 host 函数读取 device builtin、构造/赋值/取地址 builtin、访问不存在的 `threadDim`/`subThreadIdx`/`warpSize`。目标前提：regular GCU 与 EFGCU 各一个 profile。状态：`required`。限制：要区分 unknown symbol、target unavailable 和 illegal access。验证：header 定义、Clang 对照验证 和 semantic code。
- **P1-BUILTIN-P（不完整）**：截断 `threadIdx.`、`blockDim.`、成员访问和 `__has_builtin` 条件。目标前提：文档文本可恢复。状态：`required`。限制：completion 不能依赖完整表达式结尾。验证：`textDocument/completion` 的增量序列和 AST recovery。
- **P1-BUILTIN-T（目标边界）**：regular GCU 比较 `threadDim/subThreadIdx` 的 `__GCU_ARCH__ >=300` 分支，EFGCU 比较 `warpSize` 和四个 dim3-compatible builtins。目标前提：分别使用 `__GCU_ARCH__` 与 `__EFGCU_ARCH__`。状态：`required`。限制：不得把 EFGCU `warpSize` 当作 regular GCU builtin。验证：宏/header matrix、definition/references 和 target diagnostics。

#### TCLE/API

- **P1-TCLE-V（有效）**：解析 `tcle.h` 的 vector/convert symbols、`tops/vector_functions.h` 的 make functions、`__tops_dte_ext.h` 的 `mdspan` 和 `__tops_pipeline.h` 的 pipeline declarations。目标前提：header roots、device pass 和目标宏已解析。状态：`required`。限制：P1 不验证 DTE/同步执行结果。验证：header index、overload resolution、hover/definition。
- **P1-TCLE-I（无效）**：使用 `countl_zero_parser.cc`、`vector_op_parser.cc` 和 Draco invalid API cases 的 wrong overload、wrong target API、非法参数。目标前提：gcu400/gcu450/gcu500/efgcu500 分开运行。状态：`required`。限制：对照验证 失败不能转成 server runtime dependency。验证：target-specific expected diagnostics 和 Go semantic diff。
- **P1-TCLE-P（不完整）**：截断 namespace、template API、DTE context 初始化、pipeline template 参数和函数调用参数。目标前提：parser 可恢复，target 可缺失。状态：`required`。限制：缺失 header 与不完整调用要区分。验证：AST/index recovery、completion、`analysis-degraded`。
- **P1-TCLE-T（目标边界）**：比较 `topscc_types.h` 的 `__GCU_ARCH__ <500`、GCU450/LIBRA target、Draco GCU500 和 EFGCU500 的 API/header 可见性。目标前提：active header 和宏已确认。状态：`required`。限制：不把 DTE、barrier、GEMM 的硬件行为纳入 P1 通过条件。验证：profile include/index matrix、Clang syntax/CodeGen 对照验证。

#### 目标条件

- **P1-TARGET-V（有效）**：用 `.tops`、`.cpp + -Tops`、`-x tops` 三种入口解析同一个最小 kernel，并保留 target/standard/include/macros。目标前提：六个候选 profile 至少各有一条可复现命令。状态：`required`。限制：输入入口相同不等于宏集合相同。验证：`-###`、`-dM -E`、`-fsyntax-only`、Go context snapshot。
- **P1-TARGET-I（无效）**：未知 `-mcpu`/`--offload-arch`、冲突 GCU/EFGCU target、缺失 Tops headers、数据库与 workspace settings 冲突。目标前提：context resolver 可报告错误。状态：`required`。限制：不得回退到普通 C++ 完整成功。验证：`invalid-compilation-context`/`unsupported-target-feature` 和日志字段。
- **P1-TARGET-P（不完整）**：截断预处理 `#if`、宏定义、response-file 参数、未保存文件和正在编辑的 header。目标前提：context 部分可解析。状态：`required`。限制：保留旧 index 但标记受限，不发布过时 diagnostics。验证：增量 `didChange`、context version/stale 和 parser recovery。
- **P1-TARGET-T（目标边界）**：六个 profile 分别核对 `__GCU_ARCH__`/`__EFGCU_ARCH__`、`__TOPS_DEVICE_COMPILE__`、`__TOPS_ARCH__`、`__TOPS_SIMT_MODE__` 和 active include branches。目标前提：实际 driver 输出可保存。状态：`required`。限制：任何未核对的宏值写为 `candidate`，不作事实断言。验证：host/device `-dM -E`、include trace、Clang 对照验证 和 Go semantic/LSP contract。

### P0 交付与非目标

P0 的交付只有本规格和质量清单。P0 不创建 Go module、Go source、TypeScript extension、LSP server executable、parser dependency、clangd patch、clangd sidecar、运行时 workload 或硬件测试。P1 才根据本规格创建实现计划、任务和测试代码。

P0 允许读取当前 LLVM 源码、测试状态和本地工具输出；Clang 的使用限于核对 language/target/header/diagnostic/AST/CodeGen 事实和差分验证。任何 Clang 结果都不能替代 Go server 自有测试，也不能作为 Go server 运行时 fallback。

### Key Entities *(include if feature involves data)*

- **SyntaxCapability**：八个能力域中的一个稳定语法/语义条目，包含 ID、spelling、用户语义、目标前提、源码证据、测试证据、状态、限制、验证方法和 P1 phase。
- **EvidenceRecord**：可复核的源码、header、attribute、TargetInfo、driver、test、status 或 documentation 证据，包含 path、anchor、事实、范围和验证动作。
- **TargetProfile**：GCU300/GCU400/GCU410/GCU450/GCU500/EFGCU500 的候选编译解释 profile，包含 target、CPU/offload arch、宏、standard、include roots、compiler 和限制。
- **CompilationContext**：某个文档的文件级输入上下文，包含原始/归一化参数、profile、pass、include、宏、来源、解析状态和 context version。
- **SourceDocument**：用户正在编辑的 Tops C++ 文档及其文本版本、URI、编译上下文和 AST/index 版本。
- **LanguageDiagnostic**：带稳定 code、severity、range、message、source、related information 和 stale 状态的用户可见诊断。
- **SymbolIndex**：跨当前文档、工作区源文件和配置 header 的声明、定义、引用、scope、成员和 overload 索引。
- **LSPCapabilityContract**：服务器和 VS Code 客户端之间关于请求、响应、错误、取消、日志和兼容性的标准 LSP 契约。
- **P1Fixture**：按能力域和场景类型组织的有效、无效、不完整或目标边界测试源文件，附带 context、对照验证 和预期结果。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 本规格包含八个独立能力域；每个域的语法清单至少包含一个独立条目，并且每个条目都有目标前提、源码证据、测试证据、状态、限制和验证方法。
- **SC-002**: 本规格列出六个候选 TargetProfile；每个 profile 都有候选输入命令、宏/语言、目标前提、状态、限制和验证方法，且 EFGCU500 单独使用 `__EFGCU_ARCH__`。
- **SC-003**: `CompilationContext` 至少定义文档 URI、workspace/database 来源、raw/normalized arguments、输入模式、standard、target、pass、compiler、include、宏、device flags、状态、版本和验证时间字段，并给出不少于四级的优先级规则和冲突行为。
- **SC-004**: Go 服务器责任表覆盖 tokenizer、parser、AST、symbol index、semantic、LSP，并单独记录 Clang 对照验证 与 TypeScript client 的非所有权；每层均有目标前提、状态、限制和验证方法。
- **SC-005**: P1 场景矩阵覆盖 8 个能力域 × 4 种场景类型，共 32 个场景；每个场景均注明 existing 或 new 测试材料、目标前提、状态、限制和验证方法。
- **SC-006**: 规格中的当前源码/测试事实可回溯到 `llvm-project` 当前 checkout 的 driver、TargetInfo、headers、Clang tests、integration tests 或 `STATUS.md`；旧文档不作为唯一证据。
- **SC-007**: P0 文档明确“不实现 Go/TypeScript 代码、不扩展 clangd”，且没有把任何未实现的 Go server 能力标为 `supported`。
- **SC-008**: 对缺失 context、target 冲突、header 不一致、未完成输入、stale 结果和 对照验证 差异都定义了可观察的状态或错误行为；不得要求用户通过猜测 profile 恢复分析。
- **SC-009**: 质量清单中所有必选项在文档写入后通过；规格不包含未解决的 clarification marker 或模板占位符。
- **SC-010**: 文档校验只需要检查 Markdown 结构、路径存在性、JSON 配置和证据命令可审查性，不需要构建或运行任何新的语言服务器代码。

## Assumptions

- 当前参考 checkout 是 `/home/carl.du/work/llvm-project`，规格工程根目录是 `/home/carl.du/work/tops-lsp`；证据路径以这两个 workspace 根为基准。
- 当前 `STATUS.md` 的 TopsPlatform、topscc、gcusim 版本和 2026-07-14 状态是已有记录；本 P0 规格不把读取记录冒充为本轮重新执行。
- 六个 profile 是候选分析 profile，不是硬件认证矩阵；P0 只要求形成可复现的命令、宏、header 和测试核对入口。
- `compile_commands.json` 是文件级 context 的优先来源；没有条目时允许显式 workspace fallback，但不允许隐式 GCU300、系统 include 或 clangd fallback。
- Go parser 的完整 ISO C++ 覆盖不属于 P0/P1 的承诺；P1 以八个能力域和标准测试基线中的核心 slice 为范围。
- DTE、SDTE、barrier、GEMM、SIMT、VLIW、设备链接和运行时执行只在需要识别目标条件和 API 边界时作为证据或限制出现；P0 不承诺这些硬件行为。
- 源码文本、宏、诊断和日志只在本地编译上下文及 LSP 会话中处理；本功能不引入外部上传或远程语义服务。
- 后续实现必须遵守项目宪法的 LSP 契约优先、Go/TypeScript 分层、Tops 语义保真、行为优先验证和可观测兼容演进原则。
