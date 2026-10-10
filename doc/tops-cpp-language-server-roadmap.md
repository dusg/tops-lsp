# Tops C++ Language Server Roadmap

**状态**：Published

**发布日期**：2026-10-07

**最后验证**：2026-10-08

**最终发布路径**：`doc/tops-cpp-language-server-roadmap.md`

## 文档定位

本文档说明 Tops C++ 语言服务器和 VS Code 客户端如何分阶段建设。服务器用 Go 从零实现，负责 Tops C++ 的解析、语义、索引、诊断和 LSP；TypeScript 客户端负责 VS Code 的启动、配置和用户操作。用户侧编译器首选 `topscc`，也支持直接使用 `clang`/`clang++`；Clang 只用于离线对照和事实核对，不修改 clangd，也不把 clangd 放进运行链路。

## 来源工件

- [功能规格说明](../specs/001-tops-lsp-roadmap/spec.md)
- [实施计划](../specs/001-tops-lsp-roadmap/plan.md)
- [研究记录](../specs/001-tops-lsp-roadmap/research.md)
- [数据模型](../specs/001-tops-lsp-roadmap/data-model.md)
- [语法能力契约](../specs/001-tops-lsp-roadmap/contracts/capability-matrix.md)
- [LSP 边界契约](../specs/001-tops-lsp-roadmap/contracts/lsp-boundary.md)
- [编译器参数契约](../specs/001-tops-lsp-roadmap/contracts/compiler-driver.md)
- [阶段门禁契约](../specs/001-tops-lsp-roadmap/contracts/roadmap-gates.md)
- [P1.2 tokenizer/parser 规格](../specs/004-tops-cpp-tokenizer-parser/spec.md)

## 规划原则

- 分开记录标准 C++ 和 Tops 扩展。
- 语法信息以当前 Tops headers、Clang attribute/TargetInfo、driver 和测试为准；旧汇总文档只用来查目录。
- Go server 从零负责 tokenizer/parser、AST、符号索引、目标语义、诊断和 LSP。
- 文中的 `Tops C++ language server` 指服务 Tops C++ 源代码的语言服务器，实现语言为 Go，不是面向 Go 语言的 language server。
- 语言能力按标准版本和实现层次分阶段交付；基础语法、完整 C++ grammar、预处理器/宏展开和完整语义不是同一个完成条件。
- TypeScript 只负责 VS Code 扩展和 LSP 客户端，包括启动、编辑器集成、配置、状态、日志和用户命令；不负责 Tops 语义、索引或 API 分析。
- 用户编译器优先使用 `topscc`；直接 `clang`/`clang++` 是显式兼容入口。Clang 只用于事实核对和离线对照，不修改或运行 clangd。
- LSP 静态解析 `compile_commands.json` 或工作区配置中的 argv，不在运行时启动 `topscc`；`topscc --dryrun` 只用于离线检查参数展开。
- 没有经过目标条件、当前 header 和测试确认的能力，不能直接标为 `supported`。

## 来源到章节追踪

| Roadmap 章节 | 主要来源工件 | LLVM/Tops 证据入口 | 验证结果 |
| --- | --- | --- | --- |
| 语法能力地图 | `spec.md`、`research.md`、`contracts/capability-matrix.md` | `clang/lib/Headers/tops/`、`clang/include/clang/Basic/Attr.td` | 每项都写明状态、目标和验证方法 |
| Go 核心语言服务 | `data-model.md`、`contracts/lsp-boundary.md` | `clang/include/clang/Driver/Types.def`、`Options.td`、语言 tests | 先测 Go server，再用 Clang 做对照比较 |
| 编译器参数与上下文 | `data-model.md`、`contracts/compiler-driver.md` | `/opt/tops/bin/topscc`、`llvm-project/replace_topscc.sh`、Clang driver | 先验证 argv 归一化，再建立 CompilationContext |
| 目标感知语义 | `data-model.md`、`research.md`、`contracts/roadmap-gates.md` | `clang/lib/Basic/Targets/DTU.cpp`、`EFGCU.cpp`、`clang/test/DTU_test/topscc/` | 按 TargetProfile 分别验证 |
| VS Code 客户端工作流 | `contracts/lsp-boundary.md`、`spec.md` | 当前 `tops-lsp/.vscode/` 只作配置参考 | 每个编辑器操作都要有 LSP 路径和失败处理 |
| P0-P4 交付路线 | `plan.md`、`contracts/roadmap-gates.md` | `tops/integration_test/cases/language/STATUS.md` | 每阶段都写明开始、完成和降级条件 |

## 语法能力地图

本节按八个能力域列出范围。“当前证据状态”只表示当前 checkout 中有源码或测试依据，不表示 Go server 已经实现。

| 能力域 | 代表语法/符号 | 当前证据状态 | Go server roadmap |
| --- | --- | --- | --- |
| 标准 C++ | C++11、C++14、C++17 language cases、模板、lambda、constexpr、结构化绑定 | `tops/integration_test/cases/language/STATUS.md` 记录了 60 个 case 和 4 个 device coverage exception | P1 交付基础语法；P1.6 完成 C++11/14/17 grammar；C++20/23 单独评估 |
| 执行空间 | `__global__`、`__device__`、`__host__`、`__host__ __device__`、`__cooperative__`、`__sp__`、`__scalar_only__`、`__forceinline__`、`__force_noinline__` | `clang/lib/Headers/tops/__tops_defines.h`、`clang/include/clang/Basic/Attr.td`、`tops/integration_test/cases/language/exec_spec/exec_space_specifiers/main.cc` | 解析声明，检查调用边界，报告 host/device 错配，并提供悬停和导航 |
| 存储/地址空间 | `__device__`、`__constant__`、`__shared__`、`__local__`、`__private__`、`__cluster_shared__`、`__local_stack__` | Tops defines、Clang Sema/CodeGen 测试和 `topscc/addrspace/` | 在 Go 语义中记录声明空间、指针空间和访问限制 |
| 启动/资源属性 | `__thread_dims__`、`__cluster_dims__`、`__launch_bounds__`、`__maxnreg__`、`__block_tile__` | `__tops_defines.h`、`Attr.td`、`topscc/attribute/`、`launch_config/launch_bounds_maxnreg` | 检查参数和目标限制，不把资源提示当成普通 C++ 属性 |
| 向量/数值类型 | `__vector`、`__vector2`、`__vector4`、`__vector8`、`__fp16`、`__bf16`、FP8、`__valigned__` | `clang/lib/Headers/tops/vector_types.h`、`__tops_vector.h`、`CodeGenEFGCU/vector_types/` | 解析类型，提供成员补全，检查对齐和运算限制 |
| 内建变量 | `threadIdx`、`blockIdx`、`blockDim`、`gridDim`、`threadDim`、`subThreadIdx`、`warpSize` | `clang/lib/Headers/tops/__tops_builtins.h`、`__tops_efgcu_builtin_vars.h`、`builtin_vars/` | 提供字段补全、定义/引用和目标条件说明 |
| TCLE/API | `tcle.h`、Tops vector/DTE/数学和转换 headers | `clang/lib/Headers/tcle.h`、Tops headers、parser/codegen tests | 从配置的 headers 建立符号索引；没有证据的 API 保持待核验 |
| 目标条件 | `__GCU_ARCH__`、`__EFGCU_ARCH__`、`__TOPS_DEVICE_COMPILE__`、`-Tops`、`-x tops`、`topscc -arch`、`--simd/--simt*` | `clang/lib/Basic/Targets/`、`Types.def`、`Options.td`、topscc wrapper 和 driver tests | 用 CompilationContext 和 TargetProfile 统一控制解析、补全和诊断 |

### 语法与预处理器的完成边界

roadmap 中的“完整”按语言版本和输入上下文定义，不把 parser 能识别的语法误写成完整语义：

| 层次 | 计划阶段 | 完成含义 | 不包含内容 |
| --- | --- | --- | --- |
| 基础 C++ 语法 | P1.2、P1.5 | 可恢复地解析常见 C++11/14/17 声明、类型、函数、模板、表达式、语句和 Tops spelling | 完整 grammar 覆盖、完整宏展开、完整类型语义 |
| 完整 C++ grammar | P1.6 | 在明确的 C++11/14/17 language standard 下覆盖规格列出的标准语法，并为每类语法提供正向、负向和不完整输入测试 | 模板实例化、重载决议、常量求值、ODR、链接和代码生成 |
| 预处理器基础 | P1.2 | 解析 `#include`、宏定义和条件指令，保留原始 token，计算 `active`、`inactive`、`unknown` | 跨文件完整展开、token paste 结果、stringification 结果和宏递归等价性 |
| 配置上下文内的完整宏展开 | P2.2 | 在有效 `CompilationContext`、include roots 和宏集合内处理 include graph、对象宏、函数宏、variadic macro、递归展开、`#`、`##`、条件表达式和 source mapping | 超出配置上下文的外部文件；缺少输入时不得猜测或静默标记为完整 |
| C++20/23 语法 | P4.2 后续评估 | 以独立标准版本和测试材料（固定的测试输入和预期结果）评估 modules、concepts/requires、coroutines 等语法 | 不因 C++11/14/17 完成而自动宣称支持 |

P1.6 是首个“完整 C++ grammar”交付点，范围限定为 C++11/14/17；P2.2 是首个“配置上下文内的完整宏展开”交付点。两者都不等同于完整 C++ 语义。

### 状态规则

| 状态 | 含义 | 用户可见行为 |
| --- | --- | --- |
| `candidate` | 找到源码入口，但证据还不完整 | 只记录调查结果，不承诺支持 |
| `verified` | 已核对源码和适用测试/编译结果 | 可以进入 Go 语义设计 |
| `target-dependent` | 依赖目标或宏条件 | 悬停和诊断必须说明 profile |
| `supported` | Go 测试、契约测试和阶段门禁都通过 | 可以纳入稳定能力 |
| `blocked` | 有已记录的工具链或语义问题 | 显示限制和恢复条件 |
| `unsupported` | 当前 roadmap 不覆盖 | 不给出看似完整的结果 |

## 编译器 driver 与参数解析

`topscc` 是用户侧首选入口，负责把 TOPS 专用参数和普通 Clang 参数组合成 host/device 编译命令。LSP 必须理解这层 wrapper，不能把 `-arch`、`-dbin` 或 `--simd` 当成普通 Clang 参数丢给下游。

- `driver_kind=topscc` 时，解析 wrapper 参数，并保留底层 Clang 转发参数。
- `-arch <name>`、`--simd`、`--simt`、`--simt32`、`--simt128`、`-dbin`、`-dbc`、`-dllvm`、`-dlink-path`、`-dlink`、`-rdc`、`-host-only`、`-notopslib`、`-topsrt`、`-ltops` 和 `--dryrun` 属于 topscc 参数层；`-Tops`、`-x tops`、`-std`、`-I`、`-D` 等继续进入语言上下文解析。
- 已安装版本的 wrapper 默认注入 `-std=c++11`、`-Tops`、`--include tops.h`、`-fno-jump-tables` 和 Tops include 根；这些默认值必须记录为 `driver_default`，不能变成所有编译器共用的隐藏默认。
- `-arch` 组合可能展开为多个 offload target。LSP 不得静默选择其中一个目标；应保存多个候选 profile，或返回 `partial`/`invalid` 并提示用户选择分析目标。
- 未知参数、response file 无法读取、wrapper 版本无法确认或同一命令存在冲突参数时，保留原始 argv 并返回可定位的上下文状态；不伪装为完整分析。

## Go 核心语言服务

Go server 负责最终语义。Clang 只做对照比较，不直接提供 LSP 结果。

| Go 组件 | 提供的能力 | 需要记录的信息 | 验证方式 |
| --- | --- | --- | --- |
| CompilationContext loader | 读取 compile database 或工作区设置，解析 `topscc`/直接 Clang argv | `resolved`、`partial`、`missing`、`invalid` | driver 参数、优先级、多架构和缺失字段测试材料 |
| Tokenizer/parser | P1.2 解析基础标准 C++ 和 Tops 扩展；P1.6 完成 C++11/14/17 grammar；声明或函数尚未写完时仍保留已识别结构 | 文档版本、错误恢复范围和标准版本 | 有效/无效/不完整 Tops source 测试材料 |
| Preprocessor/macro expansion | P1.2 保存条件和宏原文；P2.2 在配置上下文内完成 include、宏展开和来源映射 | `CompilationContext`、include graph、宏定义和展开来源 | 宏展开 golden、Clang `-E -dD` 对照和缺失上下文测试 |
| AST and symbol index | 记录函数、类型、属性、内建变量、header 符号和宏来源 | 定义位置、引用关系、条件区域和展开来源 | document symbols、definition、references 测试 |
| Target semantic layer | 检查执行空间、地址空间、向量和目标属性 | `TargetProfile`、`target-dependent`、`unsupported` | Go 语义单测和 Clang `-fsyntax-only` 对照比较 |
| Diagnostics | 提供位置、级别、code、原因和下一步 | 新上下文不能被 `stale` 结果覆盖 | 负向、缺失上下文和目标不匹配测试材料 |
| LSP request layer | 处理 `initialize`、文档同步、diagnostics、completion、hover、definition、references、symbols | 取消、超时、重连和错误 code | LSP contract test 和端到端会话 |

### 核心 LSP 流程

1. VS Code 客户端发送 `initialize` 和工作区能力。
2. Go server 读取 `CompilationContext` 和 `TargetProfile`；缺少或冲突的信息生成诊断。
3. `didOpen`/`didChange` 更新文档版本；按已完成阶段运行 tokenizer/parser 和预处理器，生成可恢复的 AST、条件区域和符号索引。
4. Go semantic layer 根据 host/device、地址空间、向量和目标条件计算诊断和候选项。
5. server 通过标准 LSP 返回 `publishDiagnostics`、completion、hover、definition、references 和 symbols。
6. 客户端只展示结果、更新状态或转发命令，不重复判断语义。

### 失败行为

- 没有编译上下文：返回 `missing-compilation-context`，说明缺什么以及如何配置。
- 参数冲突或工具拒绝：返回 `invalid-compilation-context`，保留错误摘要和日志位置。
- 当前目标不支持语法：返回 `unsupported-target-feature`，不能静默按标准 C++ 接受。
- 头文件缺失、宏定义冲突或宏展开上下文不完整：保留原始 token 和来源，返回 `missing-compilation-context` 或 `analysis-degraded`，不能把部分展开标记为完整结果。
- 声明或函数尚未写完时（例如缺少右括号、参数或函数体）：根据已识别内容返回可恢复结果，标记 `analysis-degraded`，继续提供诊断或补全，不阻塞继续编辑。
- server 重启或上下文变化：旧结果标记 `stale-result`，不能覆盖新诊断。

## 目标感知语义

所有目标相关判断都使用文件级 `CompilationContext`，不能跨文件使用隐式默认架构。

### TargetProfile 路线

| Profile 候选 | 当前状态 | 必须核对的内容 | 失败状态 |
| --- | --- | --- | --- |
| GCU300 | candidate | `__GCU_ARCH__` 分支、向量对齐、地址空间和现有 Tops tests | `target-dependent` 或 `blocked` |
| GCU400 | candidate | DTE/地址空间、向量、launch/resource 属性和 `topscc` tests | `target-dependent` 或 `blocked` |
| GCU410 | candidate | headers、target info 和 profile-specific tests | `target-dependent` 或 `blocked` |
| GCU450 | candidate | active headers、目标宏和目标测试 | `target-dependent` 或 `blocked` |
| GCU500/EFGCU500 | candidate | `__EFGCU_ARCH__`、EFGCU builtins、SIMT/device mode | `target-dependent` 或 `blocked` |

候选 profile 要核对完整条件分支、当前 header、实际参数和 Go 测试材料，之后才能进入 `verified`。

### CompilationContext 优先级

1. 先用有效的 `compile_commands.json` 条目，并识别 argv[0] 是 `topscc` 还是直接 `clang`/`clang++`。
2. 对 `topscc` 条目先解析 wrapper 参数，再从归一化参数得到 language、target、pass、include 和宏；直接 Clang 条目按 Clang driver 规则解析。
3. 没有条目时使用 workspace fallback，只补充缺失信息，不覆盖编译数据库中的明确目标或 driver 默认来源。
4. 信息不够、目标有多个且没有选择策略、或参数无法解释时进入 `partial`/`missing`/`invalid`，提示用户配置，不猜默认架构。

### 对照验证

- 用本地 `build/bin/clang -fsyntax-only` 核对语言模式、目标参数和基础诊断。
- 用 `clang/lib/Headers/tops`、Clang TargetInfo/attribute 和现有 lit/integration tests 做事实对照。
- Go server 的正向、负向、目标边界和不完整输入测试才是最终门禁；Clang 差异不能替代 Go 测试。
- 每个差异都记录为 Go parser/semantic 修正任务，或记录为 `blocked`/`unsupported`。

## VS Code 客户端工作流

TypeScript 客户端只处理 VS Code 生命周期和用户操作。

| 编辑器动作 | TypeScript 客户端 | LSP/Go server | 用户看到的结果 |
| --- | --- | --- | --- |
| 打开 Tops 文件 | 启动 server，发送 `didOpen` | 读取上下文并发布诊断 | 显示启动状态和首批诊断 |
| 编辑文件 | 发送增量 `didChange` | 增量解析并更新索引/诊断 | 刷新诊断、补全和悬停 |
| 修改设置 | 发送 `didChangeConfiguration` | 让上下文失效并重新分析 | 显示新目标或配置错误 |
| 请求补全/悬停/导航 | 转发标准 LSP 请求 | 返回 Go 语义结果 | 显示候选、说明和位置 |
| server 错误 | 显示状态和日志位置，提供重启 | 返回稳定错误 code，不静默失败 | 用户知道如何修复或重启 |
| 关闭/重连 | 发送 shutdown/exit 或重建会话 | 清理旧文档版本和 stale 结果 | 不留下过期诊断 |

### 配置字段

- compile database 目录和文件条目选择规则。
- Go server 可执行文件或启动方式。
- `topscc` 用户编译器路径和版本；直接 `clang`/`clang++` 兼容入口及其版本。
- Clang 离线对照工具路径，仅用于离线对照，不参与 Go server 运行时语义。
- C++ 标准、Tops 输入模式、target profile/target triple。
- Tops/Clang include roots、预定义宏和必要的 device flags。
- 日志级别和日志位置。

配置来源必须可见；编译数据库中的明确参数优先于 workspace fallback。日志不得记录密钥、完整源码或无关用户数据。公开设置、命令和扩展消息必须有版本，并提供迁移或拒绝说明。

### 阶段交付物

| 阶段 | 交付物 | 类型 | 说明 |
| --- | --- | --- | --- |
| P0 | 能力清单、目标配置说明、Go server 责任边界、实现待办清单 | 文档 | 明确后续要做什么；P0 不交付生产代码 |
| P0 | 有效/无效/不完整场景清单、Clang 离线对照记录 | 测试准备和验证材料 | 为 P1 准备测试输入和对照结果 |
| P1 | tokenizer/parser、AST、符号索引、基础语义和 LSP 核心实现 | Go 代码 | 交付 Go server 的第一批可运行能力 |
| P1 | parser/semantic 单测、LSP 契约测试、核心测试材料 | 测试代码和测试数据 | 验证诊断、补全、悬停和导航 |
| P1 | LSP 方法表、诊断 code 和错误处理说明 | 文档 | 说明 Go server 对外提供什么 |
| P2.0 | `topscc`/Clang 参数解析、driver 默认值和 CompilationContext 归一化 | Go 代码、配置数据和参数测试材料 | 让同一条用户编译命令得到可审计的 language、target、pass、include 和宏上下文 |
| P2 | TargetProfile、CompilationContext 和目标相关语义规则 | Go 代码和配置数据 | 支持目标选择、上下文切换和目标诊断；依赖 P2.0 |
| P2 | 多目标正向/负向/边界测试和 Clang 对照结果 | 测试代码和测试数据 | 验证不同 GCU profile 的结果 |
| P2 | 目标支持矩阵和已知限制 | 文档 | 记录哪些目标已支持、待确认或不支持 |
| P3 | VS Code 激活、server 生命周期、配置、状态和命令 | TypeScript 代码 | 交付客户端工作流 |
| P3 | LSP 集成测试、单根/多根工作区测试、重连测试 | 测试代码和测试数据 | 验证客户端和 Go server 的协作 |
| P3 | 配置字段、日志、隐私和兼容性说明 | 文档和配置定义 | 说明用户怎么配置和恢复服务 |
| P4 | TCLE/API 索引和持续演进能力 | Go 代码 | 由 Go server 负责 API 索引和语义分析 |
| P4 | TCLE/API 的 VS Code 展示和 LSP client 集成 | TypeScript 代码 | 只负责把 Go server 的结果展示在 VS Code 中 |
| P4 | 回归、性能、日志隐私和兼容性检查 | 测试代码和测试工具 | 支持后续工具链升级后的重复验证 |
| P4 | 版本矩阵、发布检查和最终 roadmap | 文档和发布材料 | 维护长期路线并发布 `doc/tops-cpp-language-server-roadmap.md` |

## 阶段子里程碑与 specify 提示词

下面的子里程碑是后续实现的最小规划单元。P0 只产出文档，不再拆成多个子阶段；P1-P4 的每个子里程碑都应单独建立规格、验收场景和任务。提示词可以直接作为后续 `/speckit.specify` 的输入，后续 Agent 不要扩大任务范围。

以下提示词直接描述要实现的功能、范围和验收要求，供后续 `/speckit.specify` 根据需求生成规格；提示词本身不是交付物，实际完成以对应阶段的实现、测试和完成条件为准。

### P0：Go 服务器语义基线（单一文档阶段）

本阶段只写文档，不写 Go、TypeScript 或测试代码。

- **目标**：一次性确定 Tops C++ 的能力范围、目标配置、Go server 的责任边界和 P1 测试准备内容。
- **交付物**：文档、能力清单、证据表、TargetProfile/CompilationContext 定义、Go 实现待办清单和 P1 测试材料清单。
- **依赖**：无。
- **specify 提示词**：

```text
完成 Tops C++ 语言服务器的“P0 Go 服务器语义基线”。P0 只输出文档，不实现 Go/TypeScript 代码，也不扩展 clangd。文档必须一次性完成八个能力域的语法清单、源码和测试证据、GCU300/GCU400/GCU410/GCU450/GCU500/EFGCU500 候选 profile、CompilationContext 字段和优先级、Go tokenizer/parser/AST/symbol index/semantic/LSP 的责任边界，以及 P1 所需的有效、无效、不完整和目标边界测试场景。每项都要写明目标前提、状态、限制和验证方法。
```

### P1：核心语言服务

#### P1.1 Go server 基础和 LSP 传输

- **目标**：建立 Go module、server 进程、标准 LSP 生命周期和文档同步骨架。
- **交付物**：Go 代码、LSP 契约测试和启动说明。
- **依赖**：P0。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“基础服务和 LSP 传输”功能。范围只包括 Go 项目骨架、server 启停、initialize/shutdown/exit、textDocument didOpen/didChange/didClose、请求取消和基础错误返回。必须写明文件边界、LSP 契约、测试场景和日志要求，不实现具体 Tops 语义，不扩展 clangd。
```

#### P1.2 Tops C++ tokenizer/parser

- **目标**：让 Go server 能解析基础标准 C++ 和 Tops 扩展；声明或函数尚未写完时（例如缺少右括号、参数或函数体），也能继续分析并支持编辑。完整 C++11/14/17 grammar 由 P1.6 交付，完整宏展开由 P2.2 交付。
- **交付物**：Go parser 代码、parser 单测和语法测试材料。
- **依赖**：P0、P1.1。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“tokenizer/parser”功能。范围包括基础标准 C++ 语法、Tops 关键字和属性、宏条件，以及声明或函数尚未写完时（例如缺少右括号、参数或函数体）的部分解析和错误恢复。本功能只交付这些基础解析能力，不包含完整 C++11/14/17 grammar 和完整宏展开；必须列出支持和不支持的语法、正向/负向/不完整测试、诊断位置规则和与 Clang 的对照方法。语义所有权属于 Go server。
```

#### P1.3 AST 和符号索引

- **目标**：为定义跳转、引用查找、文档符号和后续语义检查建立 AST 与索引。
- **交付物**：Go 代码、索引单测、definition/references/documentSymbol 测试。
- **依赖**：P1.2。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“AST 和符号索引”功能。定义函数、类型、属性、内建变量、header 符号、宏条件、定义位置和引用关系的记录方式。必须包含文档变化后的索引更新、声明或函数尚未写完时的部分索引、缺少 header 和多文件导航场景，以及与 LSP definition/references/documentSymbol 的契约。
```

#### P1.4 核心语义和诊断

- **目标**：检查执行空间、地址空间、基本向量和 host/device 调用边界。
- **交付物**：Go 语义代码、诊断 code 表、正向/负向测试。
- **依赖**：P0、P1.2、P1.3。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“核心语义和诊断”功能。范围包括 __global__、__device__、__host__、__shared__、__local__、__constant__、__private__、基础向量和 host/device 调用边界。定义诊断 code、位置、严重级别、错误信息、受限分析和 stale result 行为，并给出有效、无效、不完整和缺少上下文的验收场景。
```

#### P1.5 基础语言功能

- **目标**：提供诊断、completion、hover、definition、references 和 document symbols 的第一个可用闭环。
- **交付物**：Go LSP 代码、LSP 契约测试、端到端测试材料和用户验证文档。
- **依赖**：P1.1、P1.3、P1.4。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“基础语言功能闭环”。覆盖 diagnostics、completion、hover、definition、references 和 document symbols，明确每个 LSP 请求的输入、结果、空结果、错误、取消和 stale 行为。验收必须覆盖一个可编辑的 Tops kernel 文件和一个存在语义错误的文件。
```

#### P1.6 C++11/14/17 完整语法

- **目标**：在 P1 基础解析之上，完成 C++11、C++14 和 C++17 语言标准中 roadmap 已列出的标准 grammar，明确语法支持与完整语义支持的边界。
- **交付物**：Go parser 代码、按 language standard 分类的语法测试材料、AST/recovery 测试、Clang 对照记录和不支持语法清单。
- **依赖**：P1.2、P1.3、P1.5。
- **完成条件**：标准声明、类型、函数、模板、表达式、语句、lambda、constexpr、attributes 和结构化绑定等已列语法均有正向、负向或不完整测试；基础 C++11/14/17 grammar 不再以 `unsupported-syntax` 作为默认处理；错误恢复、source range 和宏来源不回退。
- **失败处理**：C++20/23 构造单独返回 `unsupported-syntax` 或 `target-dependent`，不能因为 P1.6 完成而静默接受；模板实例化、重载决议、常量求值等完整语义继续由后续 semantic 阶段负责。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“C++11/14/17 完整语法”。以现有 P1 tokenizer/parser 为基础，覆盖声明、类型、函数、模板、表达式、语句、lambda、constexpr、attributes、结构化绑定和 Tops source 中需要的标准语法。必须按 C++11、C++14、C++17 分组列出支持、不支持和不完整场景，建立正向/负向/不完整测试材料、AST/source range 检查和 Clang 离线对照。不要把完整 grammar 等同于模板实例化、重载决议、常量求值或其他完整语义；C++20/23 单独处理。
```

### P2：目标感知语义

#### P2.0 topscc driver 参数解析与上下文归一化

- **目标**：让 Go server 能读取用户实际使用的 `topscc` 编译命令，并得到可审计的 `CompilerInvocation` 和 `CompilationContext`。
- **交付物**：Go driver adapter、参数分类和归一化数据、`topscc` 参数测试材料、错误/多架构测试、配置和诊断说明。
- **范围**：识别 `topscc`、直接 `clang`/`clang++` 和未知 wrapper；解析 `-arch`、device output/link、SIMD/SIMT、`-x`、`-std`、include、macro、response file 和 wrapper 默认值；保留 raw/normalized/forwarded argv。
- **不包含**：parser grammar、Clang runtime backend、设备编译执行和 TypeScript 侧参数解析。
- **依赖**：P0、P1.1；P2.1 及后续目标语义必须等待 P2.0。
- **完成条件**：单目标、组合目标、缺失 compiler、未知参数、response file、冲突参数和 wrapper 版本未知场景都有状态、诊断和可复现测试材料；LSP runtime 不启动 `topscc`。
- **失败处理**：无法安全归一化时保持 `partial`/`invalid`，不把 topscc 默认值套给直接 Clang，不静默选择多架构中的第一个目标。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“topscc driver 参数解析与编译上下文归一化”功能。支持从 compile_commands.json 的 arguments/command 和 workspace fallback 读取用户命令，区分 topscc、直接 clang/clang++ 和未知 wrapper；解析 -arch、device output/link、SIMD/SIMT、-x、-std、include、macro、response file 和 wrapper 默认值，保存 raw/normalized/forwarded 参数、来源、target candidates 和错误状态。LSP runtime 不启动 topscc，topscc --dryrun 只用于离线核对。覆盖单目标、多架构、冲突参数、未知参数、缺少 response file 和 wrapper 版本未知场景；parser 只消费归一化后的 ParseContext。
```

#### P2.1 TargetProfile 和上下文加载

- **目标**：让 Go server 根据文件选择正确的目标和编译参数。
- **交付物**：Go 代码、配置数据、上下文加载测试。
- **依赖**：P2.0、P1.1、P1.4。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“TargetProfile 和上下文加载”功能。支持文件级 target profile 选择、compile_commands.json 优先级、workspace fallback、宏和 include roots 合并，以及配置变化后的失效和重新分析。必须覆盖多目标工作区、缺少条目、冲突参数和 stale context。
```

#### P2.2 预处理器和宏展开

- **目标**：在有效 `CompilationContext` 下建立可重复的 include 和宏展开结果，为完整 C++ grammar、目标条件和符号索引提供原始来源映射。
- **交付物**：Go preprocessor/macro expansion 代码、include graph 和 source mapping 数据、宏展开测试材料、Clang `-E -dD` 对照工具或记录、缺失上下文诊断测试。
- **依赖**：P1.2、P1.6、P2.1。
- **范围**：处理配置 include roots 内的跨文件 `#include`、include guard、对象宏、函数宏、variadic macro、反斜杠续行、宏递归展开、token paste `##`、stringification `#`、条件表达式中的宏替换，以及展开 token 到原始文档的来源映射。
- **完成条件**：同一 `CompilationContext` 下，宏定义、条件分支、展开结果和 source range 可重复；每种宏形式都有正向、负向、不完整和跨文件测试材料；缺少 header、循环 include、冲突宏或无法安全展开时保留 `partial`/`analysis-degraded` 状态并提供原因。
- **失败处理**：不得使用 profile 名称猜测宏值，不得把系统中未加载的 header 当作空文件，不得把部分展开静默标为完整展开；宏相关诊断仍以原始文档位置为主。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“预处理器和完整宏展开”功能。范围包括有效 CompilationContext 和 include roots 内的跨文件 include、include guard、对象宏、函数宏、variadic macro、递归展开、反斜杠续行、token paste ##、stringification #、条件表达式宏替换和原始 source mapping。必须定义宏展开状态、未知/冲突/缺失上下文的降级行为、正向/负向/不完整/跨文件测试材料，以及与 clang -E -dD 的对照验证。不得把无法加载的外部 header 或猜测的宏值标记为完整结果。
```

#### P2.3 架构属性和类型规则

- **目标**：支持架构相关的启动/资源属性、向量对齐、内建变量、DTE/同步声明。
- **交付物**：Go 语义代码、目标规则数据、目标正向/负向测试。
- **依赖**：P2.1、P2.2、P1.4。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“架构属性和类型规则”功能。范围包括 __thread_dims__、__cluster_dims__、__launch_bounds__、__maxnreg__、__valigned__、内建变量以及 DTE/同步相关声明。每项必须说明适用 profile、参数限制、悬停内容、诊断和 unsupported/target-dependent 行为。
```

#### P2.4 目标诊断和 Clang 离线对照

- **目标**：让目标不匹配和 Go/Clang 结果差异都能被发现和解释。
- **交付物**：Go 诊断代码、对照工具/记录、目标错误测试。
- **依赖**：P2.1、P2.2、P2.3。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“目标诊断和 Clang 离线对照”功能。定义 Go server 与本地 clang -fsyntax-only 的比较范围、差异记录格式、目标不支持、目标参数冲突、缺失 header 和 stale 结果的错误行为。明确 Clang 只作离线对照，不进入 Go server 运行时路径。
```

#### P2.5 多目标检查

- **目标**：确认目标切换不会让客户端和 Go server 给出矛盾结果。
- **交付物**：多 profile 测试矩阵、回归测试代码、已知限制文档。
- **依赖**：P2.1、P2.2、P2.3、P2.4。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“多目标检查”功能。使用 GCU300、GCU400、GCU410、GCU450、GCU500/EFGCU500 候选 profile，定义同一源文件在不同 profile 下的正向、负向、边界和 stale 结果检查。每个 profile 都必须有 verified、target-dependent、blocked 或 unsupported 结论。
```

### P3：VS Code 工作流

#### P3.1 TypeScript 客户端基础

- **目标**：建立 VS Code extension、激活入口和 Go server 启动骨架。
- **交付物**：TypeScript 代码、扩展配置和启动测试。
- **依赖**：P2、P1.1、P1.5。
- **specify 提示词**：

```text
实现 Tops C++ VS Code 客户端的“TypeScript 客户端基础”功能。范围包括 extension 激活、Tops C++ language server 启动、标准 LSP client 建立、关闭和启动失败提示。明确不在客户端实现 Tops 语义，给出单根工作区的验收场景和日志要求。
```

#### P3.2 配置和生命周期

- **目标**：支持 compile database、Go server、topscc 用户编译器、直接 Clang 兼容入口、Clang 离线对照工具、目标 profile、include roots 和宏配置。
- **交付物**：TypeScript 代码、配置定义、配置变化测试和用户配置文档。
- **依赖**：P2.1、P3.1。
- **specify 提示词**：

```text
实现 Tops C++ VS Code 客户端的“配置和生命周期”功能。定义 compile database、Tops C++ language server 路径、topscc 用户编译器路径、直接 Clang 兼容入口、Clang 离线对照工具路径、C++ 标准、Tops 输入模式、target profile、include roots、宏和 device flags 的配置来源与优先级。覆盖配置修改、重启、缺失配置和冲突配置。
```

#### P3.3 状态、日志和用户命令

- **目标**：让用户知道 server 是否可用、出了什么问题以及如何恢复。
- **交付物**：TypeScript 代码、状态/日志/重启命令、隐私和兼容性文档。
- **依赖**：P3.1、P3.2。
- **specify 提示词**：

```text
实现 Tops C++ VS Code 客户端的“状态、日志和恢复命令”功能。覆盖 server unavailable、missing-compilation-context、invalid-compilation-context、request-cancelled 和 stale-result 的用户提示、日志字段、重启命令和恢复流程。日志不得记录密钥、完整源码或无关用户数据。
```

#### P3.4 客户端集成

- **目标**：验证单根、多根、重连、旧诊断清理和完整编辑流程。
- **交付物**：TypeScript 集成测试、LSP 端到端测试、VS Code 使用说明。
- **依赖**：P3.1、P3.2、P3.3。
- **specify 提示词**：

```text
实现 Tops C++ VS Code 客户端的“客户端集成验证”功能。覆盖单根和多根工作区、打开/编辑文件、补全/悬停/导航、配置修改、server 重启、断线重连和旧诊断清理。每个场景都要写清客户端动作、LSP 消息、Go server 结果和用户看到的结果。
```

### P4：TCLE 覆盖与持续演进

#### P4.1 TCLE/API 索引和客户端展示

- **目标**：由 Go server 扩展高价值 TCLE、vector、DTE 和数学 API 的索引和语义；由 TypeScript LSP client 展示补全、悬停和导航结果。
- **交付物**：Go API 索引和语义代码、TypeScript VS Code/LSP client 集成代码、API 测试材料和文档。TypeScript 不实现 API 语义。
- **依赖**：P1.3、P2.2、P2.3、P3.4。
- **specify 提示词**：

```text
实现 Tops C++ 语言工具的“TCLE/API 索引和客户端展示”功能。Tops C++ language server 负责配置的 tcle.h、vector、DTE、数学和转换 headers 的符号索引、API 语义和支持状态；TypeScript 只负责 VS Code LSP client 对 completion、hover、definition 和 references 结果的展示。每个 API 必须有来源、目标范围、支持状态、限制和测试方式；没有证据的 API 不得标记为 supported。
```

#### P4.2 回归、性能和兼容性

- **目标**：让每次工具链或语言能力变化都能重复验证，并在 C++11/14/17 grammar 完成后单独评估 C++20/23 语法。
- **交付物**：Go server 回归代码、TypeScript VS Code/LSP client 回归代码、按 C++ 标准版本分类的语法测试材料、性能测试工具、兼容性矩阵和日志隐私检查。
- **依赖**：P1.6、P2.2、P2.5、P3.4、P4.1。
- **specify 提示词**：

```text
实现 Tops C++ 语言工具的“回归、性能和兼容性”功能。定义 C++11/14/17 完整 grammar、C++20/23 语法评估、预处理器/宏展开、Tops C++ language server、LSP、VS Code、TCLE/API 的回归范围，补全/悬停/导航的性能指标，日志隐私检查，以及公开能力、设置和命令的版本兼容策略。C++20/23 必须按独立标准版本建立证据和状态，不得从 C++11/14/17 的结果推断支持。必须包含失败、降级和迁移场景。
```

#### P4.3 发布和持续维护

- **目标**：把已验证的能力、限制和下一阶段计划发布给项目成员。
- **交付物**：文档、发布检查表、版本/目标矩阵和最终 roadmap 更新。
- **依赖**：P4.1、P4.2。
- **specify 提示词**：

```text
实现 Tops C++ 语言工具的“发布和持续维护”功能。定义能力矩阵更新、版本/目标矩阵、发布前检查、已知限制、兼容性说明、迁移说明和 roadmap 更新流程。最终文档必须写入 doc/tops-cpp-language-server-roadmap.md，并保留 specs/ 下的审计来源。
```

## P0-P4 交付路线

| 阶段 | 目标 | 主要交付 | 完成条件 | 失败处理 | owner |
| --- | --- | --- | --- | --- | --- |
| P0 Go 语义基线 | 明确 Go server 的责任和事实边界 | 能力矩阵、parser/AST/index 边界、CompilationContext、Clang 对照工具，以及完整 grammar/宏展开的后续边界 | 每个 P1 条目有证据、正反例、Go 验证和对照动作 | `candidate`/`blocked`，不引入 clangd 运行依赖 | Go server + 文档维护者 |
| P1 核心语言服务 | 用户获得稳定基础反馈，并完成 C++11/14/17 grammar | 基础标准 C++、Tops 扩展、P1.6 完整 grammar、执行/存储空间、内建变量、基础向量的诊断、补全、悬停、导航 | Go unit、测试材料、LSP contract、完整 grammar 覆盖和错误路径通过 | 延后目标相关能力和完整宏展开，保持受限模式 | Go server |
| P2 目标感知语义 | 多 profile 结果一致且说得清楚 | CompilationContext、预处理器/宏展开、架构条件、启动/资源、向量对齐、DTE/同步边界 | 宏展开、跨文件来源、多个 profile 的正反例和 stale 诊断通过 | `target-dependent`/`unsupported`/`analysis-degraded` | Go server + 文档维护者 |
| P3 VS Code 工作流 | 用户能配置、查看和恢复服务 | 激活、配置、状态、日志、重启、单根/多根工作区 | 初始化、配置变更、重连和旧诊断清理通过 | 保留标准 LSP，延期自定义命令 | TypeScript client + Go server |
| P4 TCLE 与持续演进 | 能随工具链扩展能力，并按标准版本继续演进 | headers/API 导航补全、C++20/23 语法评估、版本矩阵、性能和回归治理 | 新能力有证据、测试、契约版本和迁移记录；未完成的标准版本保持独立状态 | `documentation-only`/`blocked`/`unsupported` | 共享契约 + 文档维护者 |

阶段顺序是：P0 → P1.1-P1.6 → P2.0 → P2.1-P2.5 → P3 → P4。P2.1 必须等待 P2.0；P2.2 必须等待 P1.6、P2.0 和 P2.1 完成；P3 必须等待 P2 的目标语义和配置规则完成；P4 必须等待 P2、P3 的语义和客户端边界完成。每个阶段完成后再进入下一阶段。

### 阶段到实现的交接

| 阶段 | 后续实现可以直接使用的材料 |
| --- | --- |
| P0 | 能力矩阵、TargetProfile 清单、CompilerInvocation/CompilationContext 字段、parser/semantic 待办清单、Clang 离线对照测试材料 |
| P1 | Go 模块边界、LSP 方法表、诊断 code、C++11/14/17 完整 grammar 和核心正/负/不完整测试材料 |
| P2.0 | topscc/Clang 参数分类、归一化 argv、wrapper 默认值来源、多架构和 context 错误测试材料 |
| P2 | 目标条件规则、预处理器/宏展开 source mapping、profile matrix、上下文失效规则、目标边界回归集 |
| P3 | TypeScript client workflow、配置字段、生命周期命令、日志和集成测试场景 |
| P4 | TCLE/API 扩展队列、契约版本策略、性能/回归套件和发布 checklist |

### 验证方式

| 验证层 | 覆盖内容 | 结果要求 |
| --- | --- | --- |
| 事实核对 | headers、attributes、driver、TargetInfo、测试状态 | 每条能力有路径、anchor、范围和日期 |
| Go parser/semantic | 有效、无效、不完整、完整 C++11/14/17 grammar、目标边界和缺失上下文 | Go 结果可重复，差异有记录 |
| Preprocessor/macro | include graph、宏展开、条件分支、source mapping 和缺失上下文 | 在有效 CompilationContext 内结果可重复；部分输入明确降级 |
| LSP contract | lifecycle、document sync、diagnostics、completion、hover、navigation | 请求、响应、错误和 stale 行为一致 |
| VS Code integration | 初始化、配置、重启、重连、单根/多根 | 用户看到的结果与 LSP 路径一致 |
| 性能/观测 | 补全、悬停、导航延迟，错误日志和隐私 | 95% 常规结果小于 1 秒；失败可定位且不泄露源码 |

### 目标对应内容

| 规格标准 | 文档中的对应内容 |
| --- | --- |
| SC-001 | 八个能力域表和状态规则 |
| SC-002 | Go 核心语言服务表、失败行为和 P1 测试材料要求 |
| SC-003 | 目标感知语义的 Clang 离线对照和 Go 测试材料检查 |
| SC-004 | 验证方式中的性能/观测层和 P4 回归治理 |
| SC-005 | VS Code 配置字段、缺失上下文诊断和 P3 退出条件 |
| SC-006 | LSP 流程、契约来源和各阶段行为测试要求 |
| SC-007 | TargetProfile、CompilationContext 和目标切换规则 |
| SC-008 | VS Code 配置契约、版本化消息和发布前检查 |
| SC-009 | 固定发布路径和发布前检查 |

## 发布前检查

最终文档固定为 `doc/tops-cpp-language-server-roadmap.md`。只有满足以下条件，文档才能保持 Published：

1. `specs/001-tops-lsp-roadmap/spec.md`、`plan.md`、`research.md`、`data-model.md`、`contracts/` 和 `quickstart.md` 通过检查。
2. 能力地图、基础语法与完整 C++11/14/17 grammar 的边界、预处理器/宏展开阶段、Go 架构、目标 profile、LSP 边界和 P0-P4 表没有矛盾。
3. 所有事实引用都能回到当前 LLVM/Tops checkout；未验证项保留状态和下一步动作；C++20/23 和未完成的宏上下文不能标记为 `supported`。
4. 完成 quickstart 中的文档、JSON、源码路径、Clang 对照和最终文件检查。
5. 通过 `git diff --check`，没有模板占位符、未解释的 `NEEDS CLARIFICATION` 或 unsupported claim。

## 质量与发布

文档状态为 Published。验证日期、来源工件和已知限制已经记录。本文档不替代 `specs/` 下的审计来源，也不把基础语法、Clang 对照结果或 parser 阶段完成说成完整 C++ grammar、完整宏展开或 Go server 已实现的完整语义。
