# Feature Specification: Tops C++ Language Server Roadmap

**Feature Branch**: `001-tops-lsp-roadmap`

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "调查Tops C++语言扩展语法，并制定Tops C++语言服务器和vscode客户端的roadmap"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 建立 Tops C++ 语法能力地图 (Priority: P1)

作为语言工具维护者，我希望看到一份有证据、有目标架构边界的 Tops C++ 语法能力地图，从而可以按用户价值和验证条件安排语言服务器工作。

**Why this priority**: 没有统一的语法范围和支持状态，服务器容易把标准 C++、Tops 扩展、TCLE API 以及架构相关行为混为一谈，后续诊断和补全也无法建立稳定预期。

**Independent Test**: 仅使用当前工具链头文件、Clang 属性声明、语言集成测试和状态记录，逐项检查能力地图；每个条目都能指向已核验依据，或被明确标记为目标相关、暂未确认或超出范围。

**Acceptance Scenarios**:

1. **Given** 当前 checkout 中存在 Tops 头文件、Clang attribute 声明和 language tests，**When** 维护者查看能力地图，**Then** 可以按执行空间、存储空间、启动/调度属性、向量与扩展数值类型、内建变量、TCLE/API 和标准 C++ 能力分类定位条目。
2. **Given** 某个语法或类型依赖 `__GCU_ARCH__`、`__EFGCU_ARCH__`、工具链版本或特定头文件，**When** 维护者查看该条目，**Then** 可以看到适用条件、证据状态和下一步验证动作，不会被表述为无条件支持。

### User Story 2 - 在 Tops 源码中获得核心语言服务 (Priority: P1)

作为 Tops C++ 开发者，我希望在编辑 kernel 和 host/device 混合代码时获得解析、诊断、补全、悬停和导航反馈，从而在编译前发现语法和明显的语义错误。

**Why this priority**: 这是语言服务器提供直接用户价值的最小闭环，优先覆盖日常 kernel 编写中频率最高且最容易因上下文错误而误判的语言扩展。

**Independent Test**: 使用一组包含有效和无效标准 C++、执行空间修饰符、存储空间修饰符、内建变量和向量类型的最小源文件；在不运行设备 workload 的情况下检查编辑器反馈是否与预期结果一致。

**Acceptance Scenarios**:

1. **Given** 一个具有有效 Tops 编译上下文的源文件，**When** 用户输入 `__global__`、`__device__`、`__shared__` 或向量类型声明，**Then** 编辑器提供相应的语法着色、补全或悬停信息，并保持标准 C++ 的正常解析。
2. **Given** 一个把 host/device 调用关系、存储空间或向量对齐写错的源文件，**When** 用户保存或编辑该文件，**Then** 编辑器显示带位置、原因和修复方向的诊断，不静默吞掉错误。
3. **Given** 用户从函数、类型、内建变量或 Tops header 中的符号发起导航，**When** 服务器能够解析该符号，**Then** 用户可以跳转到定义并查找引用；无法解析时，编辑器显示可解释的缺失上下文信息。

### User Story 3 - 按目标架构获得一致的 Tops 语义反馈 (Priority: P2)

作为需要支持多代 GCU 的开发者，我希望语言服务依据实际编译目标解释架构相关语法和限制，从而避免在一个目标上看似正确、换目标后才失败。

**Why this priority**: Tops 头文件和 Clang 属性存在架构条件；不把目标上下文纳入分析会产生错误补全、错误诊断或漏报。

**Independent Test**: 对同一组代表性源文件分别提供不同目标 profile，比较属性、向量、DTE/同步相关声明和内建变量的可用性、悬停说明以及诊断结果是否按 profile 变化。

**Acceptance Scenarios**:

1. **Given** 用户已选择或由编译数据库提供目标 profile，**When** 用户悬停查看架构相关语法，**Then** 说明中包含适用目标和已知约束。
2. **Given** 源文件使用当前 profile 不支持的扩展，**When** 服务器分析该文件，**Then** 诊断明确指出目标不匹配，并避免把该扩展当作普通标准 C++ 能力提供。
3. **Given** 目标架构、标准版本、include 路径或编译器路径缺失，**When** 用户打开 Tops 源文件，**Then** 客户端提示缺失配置或使用了受限模式，不把不完整分析结果伪装成完整语义结果。

### User Story 4 - 在 VS Code 中配置、观察和恢复语言服务 (Priority: P2)

作为 VS Code 用户，我希望通过熟悉的工作区设置和命令管理 Tops 语言服务，并能看到服务器状态和失败原因，从而不需要手动操作后台进程来恢复编辑体验。

**Why this priority**: 服务器能力只有在客户端能稳定启动、传递编译上下文并反馈错误时才可用；客户端还必须保持与服务器的职责边界。

**Independent Test**: 在单根和多根工作区分别打开 Tops 源文件，配置编译数据库或显式编译上下文，执行启动、重启、查看日志和修改设置等操作，验证用户可见结果与 LSP 消息路径一致。

**Acceptance Scenarios**:

1. **Given** 工作区存在有效的 Tops 编译上下文，**When** 用户打开匹配的 Tops 源文件，**Then** VS Code 客户端自动激活服务并展示初始化成功状态。
2. **Given** 编译器、头文件、编译数据库或目标配置不可用，**When** 客户端尝试启动或更新文档，**Then** 用户看到可执行的错误信息和相关日志位置。
3. **Given** 服务器发生可恢复的连接或配置错误，**When** 用户执行重启或修正设置，**Then** 客户端重新建立 LSP 会话，且不会遗留过期诊断覆盖新结果。

### User Story 5 - 以可回归的里程碑扩展语言覆盖 (Priority: P3)

作为项目维护者，我希望每个 roadmap 阶段都有可独立验收的能力、测试和退出条件，从而可以逐步扩展语法覆盖而不破坏标准 C++、LSP 契约或已有 VS Code 工作流。

**Why this priority**: Tops 语法横跨标准 C++、Clang 扩展、目标属性和 TCLE API；分阶段验收可以控制语义风险和跨层回归成本。

**Independent Test**: 逐阶段执行能力矩阵、服务器行为测试、LSP 契约测试和 VS Code 客户端集成测试；每一阶段都能单独判断通过、延期或降级为明确的 unsupported 状态。

**Acceptance Scenarios**:

1. **Given** 一个 roadmap 阶段已声明范围，**When** 运行其正向、负向和边界测试材料，**Then** 每个声明的能力都有可重复的通过条件，未覆盖能力不会被标记为已支持。
2. **Given** LSP 能力、设置或命令发生变化，**When** 执行跨服务器和客户端的契约验证，**Then** 可以确认请求、响应、错误行为和用户可见结果的兼容性影响。

### Edge Cases

- 用户只打开文件但没有编译数据库、编译器路径或 Tops include 路径。
- 同一个工作区同时包含不同目标架构或不同 C++ 标准的编译单元。
- 当前文件尚未保存、语法处于不完整状态，或头文件正在被编辑。
- 标准 C++ 语法有效，但在 host/device 执行空间或目标架构语境下无效。
- Tops 扩展宏未定义、被项目宏覆盖，或头文件来自与编译器不匹配的安装版本。
- 某项能力在源码和测试中存在，但测试状态明确记录为部分覆盖、已知失败或未默认运行。
- LSP 请求超时、服务器崩溃、客户端重连，或旧诊断在新结果到达前仍留在编辑器中。
- 单根与多根工作区中存在相同文件名、不同 include 根或不同编译上下文。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 建立 Tops C++ 语法能力目录，至少覆盖标准 C++、执行空间、存储空间/地址空间、启动与调度属性、向量与扩展数值类型、内建变量、TCLE/API 和目标相关宏。
- **FR-002**: 每个能力条目 MUST 记录用户可见语义、适用目标或前提、证据来源、当前状态和验证方式；状态至少区分已核验、目标相关、暂未确认和超出当前 roadmap 范围。
- **FR-003**: 系统 MUST 将标准 C++ 能力与 Tops 特有能力分开呈现，并允许按实际编译标准分析；当前仓库状态记录的 C++11、C++14、C++17 能力及已知 device coverage exception MUST 纳入覆盖基线，而不得把回归通过等同于全部 device feature 通过。
- **FR-004**: 语言服务 MUST 对 Tops 源文件提供增量语法解析，并在输入不完整、宏展开不完整或头文件缺失时返回可定位、可解释的诊断。
- **FR-005**: 语言服务 MUST 识别并解释执行空间相关声明，包括 `__global__`、`__device__`、`__host__`、`__host__ __device__`、`__cooperative__`、`__sp__`、内联控制以及同类已核验属性；错误的调用边界 MUST 能产生诊断。
- **FR-006**: 语言服务 MUST 识别并解释存储空间和地址空间相关声明，包括 `__device__`、`__constant__`、`__shared__`、`__local__`、`__private__` 以及当前目标实际提供的其他空间属性；空间不匹配 MUST 能产生诊断或明确的未完成分析提示。
- **FR-007**: 语言服务 MUST 识别启动和资源提示相关声明，包括 `__thread_dims__`、`__cluster_dims__`、`__launch_bounds__`、`__maxnreg__` 和当前目标实际提供的同类属性，并在参数格式或目标限制不满足时报告原因。
- **FR-008**: 语言服务 MUST 识别向量语法、向量布尔类型、扩展浮点类型和 `__valigned__` 等相关声明；悬停信息 MUST 说明类型可用范围、对齐或运算限制在当前目标下是否适用。
- **FR-009**: 语言服务 MUST 为 `threadIdx`、`blockIdx`、`blockDim`、`gridDim`、`threadDim`、`subThreadIdx`、`warpSize` 等当前目标提供的内建变量提供符号解析、成员补全和目标条件说明。
- **FR-010**: 语言服务 MUST 能从用户配置的 Tops/TCLE headers 建立可导航的类型、函数和常量符号；对仅有声明但缺少可用实现或文档的 API MUST 显示实际可确认的状态。
- **FR-011**: 系统 MUST 支持从编译数据库或工作区配置获得文件对应的 `topscc`/直接 Clang driver、编译标准、目标 profile、include 路径和必要的预定义宏，并明确配置优先级和缺失字段的行为。
- **FR-012**: 系统 MUST 在目标 profile 改变后重新分析受影响文档，并使补全、悬停、诊断和符号解析使用同一目标上下文；不得让客户端自行复制语义判断。
- **FR-013**: 语言服务 MUST 为常用 Tops 关键字、属性、类型、内建变量和 header 符号提供上下文补全与悬停说明；说明 MUST 标出标准 C++、Tops 扩展和目标相关能力的区别。
- **FR-014**: 语言服务 MUST 支持定义跳转、引用查找和文档符号等基础导航能力；无法完成导航时 MUST 给出缺少编译上下文、宏条件或头文件的可操作原因。
- **FR-015**: VS Code 客户端 MUST 负责扩展激活、工作区配置、语言服务器生命周期、状态展示、日志查看、重启和用户命令；语义分析结果 MUST 通过标准 LSP 消息或有文档的扩展契约传递。
- **FR-016**: 服务器 MUST 负责语言分析、工作区状态和 LSP 协议行为；客户端 MUST 不重复实现执行空间、目标属性、类型可用性或诊断等级等语义决策。
- **FR-017**: 每个新增能力 MUST 记录从编辑器动作到 LSP 请求、服务器结果、客户端处理和用户可见结果的完整路径，并说明负载、错误行为、兼容性影响和日志位置。
- **FR-018**: 系统 MUST 为服务器行为、客户端行为和跨边界契约分别提供自动化测试；语言扩展测试 MUST 覆盖有效用法、非法用法、目标边界和不完整输入。
- **FR-019**: 失败 MUST 产生可执行的诊断或日志，不得静默回退为看似完整的标准 C++ 分析；日志 MUST 不记录密钥、不必要的完整源代码或无关用户数据。
- **FR-020**: Roadmap MUST 为每个阶段定义范围、依赖、交付能力、测试入口、退出条件、延期时的降级状态和向后兼容影响。
- **FR-021**: Roadmap MUST 以 Go 从零开发的服务器作为确定架构；服务器自行负责 Tops C++ 的解析、语义、索引和 LSP 行为，不扩展或运行时依赖 clangd；`topscc` 是用户编译器入口，直接 Clang 是兼容入口，Clang 离线对照不进入 runtime。
- **FR-022**: 最终发布的 roadmap 文档 MUST 写入 `doc/tops-cpp-language-server-roadmap.md`；`specs/001-tops-lsp-roadmap/` 下的文件仅作为研究、设计、契约和验收来源。
- **FR-023**: Roadmap MUST 为 `CompilerInvocation` 定义独立的参数解析计划，区分 `topscc` wrapper 参数、直接 Clang 参数、链接/输出参数和未知参数，并保留 raw/normalized/forwarded argv。
- **FR-024**: `topscc` 参数解析计划 MUST 覆盖 `-arch`、device output/link 参数、SIMD/SIMT 参数、`-x`、`-std`、include、macro、response file、wrapper 默认值来源和多架构展开。
- **FR-025**: 多架构展开、wrapper 版本未知、response file 无法读取或参数冲突时，roadmap MUST 定义 `partial`/`invalid` 状态和用户可见诊断，不得静默选择目标或套用隐藏默认。
- **FR-026**: LSP runtime MUST 静态解析 compile command argv；`topscc --dryrun` 只能作为离线核对工具，不能作为 Go server runtime fallback。

### Syntax Survey and Roadmap Scope

当前 checkout 中已核验的能力范围如下。这里的“已核验”表示能在当前仓库的源码、头文件或测试资料中找到直接依据，不表示语言服务器已经实现该能力。

| 能力域 | 已核验内容 | 语言服务器 roadmap 关注点 |
| --- | --- | --- |
| 标准 C++ | `tops/integration_test/cases/language` 记录了 C++11、C++14、C++17 的 language cases，并记录了 4 个 device coverage exception | 保持标准 C++ 解析和 Tops 语境一致；单独展示 device 例外 |
| 执行空间 | `__global__`、`__device__`、`__host__`、`__host__ __device__`、`__cooperative__`、`__sp__`、`__scalar_only__`、`__forceinline__`、`__force_noinline__` | 调用边界、入口函数约束、host/device 双路径和补全/悬停 |
| 存储空间 | `__constant__`、`__shared__`、`__local__`、`__private__`、`__cluster_shared__`、`__local_stack__` | 地址空间、声明位置、指针和访问约束 |
| 启动与资源属性 | `__thread_dims__`、`__cluster_dims__`、`__launch_bounds__`、`__maxnreg__`、`__block_tile__` | 参数校验、目标限制和资源提示的可解释性 |
| 向量与数值类型 | `__vector`、`__vector2`、`__vector4`、`__vector8`，`__fp16`、`__bf16`、FP8 类型及 `__valigned__` | 类型解析、成员/运算补全、对齐和目标可用范围 |
| 内建变量 | `threadIdx`、`blockIdx`、`blockDim`、`gridDim`、`threadDim`、`subThreadIdx`，EFGCU 分支中的 `warpSize` | 目标 profile 下的符号、成员和悬停信息 |
| TCLE/API | `tcle.h`、Tops vector/DTE/数学等 headers 及相关 parser/codegen tests | 从用户配置的 headers 建立导航、补全和文档状态 |
| 目标条件 | `__GCU_ARCH__`、`__EFGCU_ARCH__`、`__TOPS_DEVICE_COMPILE__` 等条件会改变可见声明或语义 | profile 选择、宏上下文和避免错误的跨架构推断 |

权威性约束：`clang/lib/Headers/tops`、Clang 的 Tops attribute 声明、当前语言测试和测试状态优先于已标注为不再维护的旧汇总文档。语言服务器第一阶段不替代 topscc 的代码生成、设备链接、运行时执行、汇编语义、性能模型或跨 kernel 数据流分析；这些能力只有在后续阶段定义明确契约后才可纳入。

### Roadmap Phases

| 阶段 | 目标结果 | 主要能力 | 退出条件 |
| --- | --- | --- | --- |
| P0 Go 服务器语义基线 | 维护者拥有可审查的能力矩阵、Go 语义边界和目标 profile 定义 | 语法分类、Go tokenizer/parser/AST/index 边界、标准版本基线、编译上下文优先级、Clang 差分验证、unsupported/暂未确认规则 | 每个 P1 条目都有来源、正反例、目标前提、Go 侧验证动作和 Clang 差分动作 |
| P1 核心语言服务 | 用户能编辑 Tops kernel 并获得稳定的基础反馈 | 标准 C++、执行空间、常用存储空间、基础内建变量、基础向量类型的解析、诊断、补全、悬停和导航 | 代表性有效/无效测试材料通过；LSP 请求和错误路径有契约测试 |
| P2.0 topscc driver 上下文 | 用户的 `topscc` 编译命令可转换为可审计的 `CompilationContext` | wrapper 参数解析、直接 Clang 兼容入口、默认值来源、response file、多架构和错误降级 | `CompilerInvocation` 字段、参数类别、单/多目标和失败路径都有契约与测试材料 |
| P2 目标感知语义 | 同一源文件在不同目标 profile 下得到一致且可解释的结果 | 架构条件、启动/资源属性、向量对齐和可用范围、DTE/同步声明的识别与边界诊断 | 目标正反例矩阵通过；目标缺失或不匹配不会静默降级为完整结果 |
| P3 VS Code 工作流 | 用户能配置、观察和恢复语言服务 | 编译上下文设置、单根/多根工作区、激活、状态、日志、重启、配置错误提示和兼容迁移 | 客户端集成测试覆盖初始化、配置变化、重连、旧诊断清理和多根工作区 |
| P4 TCLE 覆盖与持续演进 | 语言工具覆盖高价值 TCLE/API，并可随工具链演进 | header 符号、数学/向量/DTE API 的补全和导航，版本/目标能力矩阵，性能和回归治理 | 每个新增能力都有证据、测试、契约变更记录和兼容性结论 |

### Scope Boundaries

- **In scope**: Tops C++ 源码的语法与静态语义反馈、目标/编译上下文、标准 LSP 能力、VS Code 工作流、可回归的语法能力地图。
- **Out of scope for the initial roadmap**: clangd 扩展或运行时依赖、设备代码生成与链接、运行时 workload 执行、汇编器/反汇编器、性能调优建议、调试器协议、跨 kernel 数据流和完整的 DTE 正确性证明。

### Key Entities *(include if feature involves data)*

- **Syntax Capability**: 一个可被语言服务识别、补全、解释或诊断的标准 C++ 能力或 Tops 扩展，包含分类、语义、证据、状态和目标前提。
- **Target Profile**: 用于解释源码的目标架构、编译标准、工具链版本、预定义宏和相关限制的集合。
- **Compilation Context**: 文件对应的编译器路径、编译数据库条目、include 路径、编译选项和工作区覆盖设置。
- **Compiler Invocation**: 一次 `topscc`/直接 Clang 用户编译命令的 driver kind、raw/normalized/forwarded argv、默认值来源、目标候选和解析状态。
- **Source Document**: 用户正在编辑的 Tops C++ 文件及其已解析的宏、头文件和符号上下文。
- **Language Diagnostic**: 与源位置关联的错误、警告、信息或受限分析提示，包含原因、严重级别、来源和可执行的下一步。
- **LSP Capability Contract**: 服务器和 VS Code 客户端之间关于请求、响应、负载、错误行为、日志和兼容性的标准或扩展契约。
- **Roadmap Milestone**: 一组有优先级、依赖、测试、退出条件和降级状态的语言或客户端能力。
- **Roadmap Document**: 面向项目维护者和开发者发布的最终 roadmap 文档，固定发布路径为 `doc/tops-cpp-language-server-roadmap.md`，内容必须引用已通过门禁的规格、能力矩阵、LSP 契约和阶段目标。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 语法能力地图覆盖上述 8 个能力域，并为每个条目提供至少一个当前仓库证据或明确的“暂未确认”状态；不得存在没有状态的 roadmap 条目。
- **SC-002**: P1 验收集为每个核心能力域至少提供 1 个有效用法、1 个无效用法和 1 个不完整输入场景；每个场景都能在服务器或客户端测试中重复执行。
- **SC-003**: 在已配置且可复现的编译上下文中，核心测试材料的预期诊断、符号解析和目标可用性结果达到至少 95% 的一致率；其余结果必须有记录的已知限制，不得静默通过。
- **SC-004**: 对代表性工作区的常规补全、悬停和导航请求，95% 的用户可见结果在 1 秒内返回；超过阈值的请求必须产生可定位的性能日志。
- **SC-005**: 用户从打开工作区到看到首个有效诊断或初始化错误提示，不超过 5 分钟；缺失编译上下文时也必须在该流程内得到明确原因。
- **SC-006**: 每次新增或修改 LSP 能力、设置或命令都通过服务器单元测试、客户端测试和跨边界契约测试中的适用集合，并保留正常、失败和边界路径结果。
- **SC-007**: 目标 profile 切换后，所有受影响文档在下一次用户可见结果中使用同一 profile；验收中不得出现客户端和服务器对同一语法给出相互矛盾结论的情况。
- **SC-008**: 现有标准 C++ 语言能力和已公开的 LSP/客户端设置默认保持兼容；不兼容变更必须在 roadmap 记录版本标记、迁移方式或拒绝理由。
- **SC-009**: 最终 roadmap 文档存在于 `doc/tops-cpp-language-server-roadmap.md`，并且其能力范围、Go 架构、阶段门禁和验证入口与 `specs/001-tops-lsp-roadmap/` 中通过检查的设计文档一致。

## Assumptions

- 语言服务器确定由 Go 从零实现，VS Code 客户端由 TypeScript 实现；这是项目章程和当前架构决策规定的职责边界，不扩展或运行时依赖 clangd。
- 服务器优先使用项目已有的编译数据库和 Tops/Clang headers；没有完整上下文时采用明确的受限模式，而不是猜测目标或静默使用默认架构。
- 初始目标 profile 以当前 Tops 工具链实际可验证的 GCU300、GCU400、GCU410、GCU450、GCU500/EFGCU500 变体为候选范围；正式支持矩阵必须在 P0 根据活动 headers、编译器和测试确认。
- `clang/docs/Tops_C_Language_Extenstion.md` 已明确标注后续不再维护，因此只作为历史分类索引；最终语义以当前 headers、Clang attribute 定义、测试和工具链验证为准。
- 当前 `tops-lsp` checkout 主要是 Spec Kit、工作区和调试配置骨架；实现阶段需要先在计划中确认现有编译、测试和依赖入口，不假设外部 `clangd` 路径已经可用。
- 源代码文本默认只在本地编译上下文和 LSP 会话中处理；roadmap 不引入将源码上传到外部服务的要求。
- 首期关注可解释、可回归的静态语言能力，不承诺替代 topscc 的完整编译、设备链接或运行时验证。
