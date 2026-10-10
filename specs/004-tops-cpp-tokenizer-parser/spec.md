# Feature Specification: Tops C++ Tokenizer 与 Parser

**Feature Branch**: `004-tops-cpp-tokenizer-parser`

**Created**: 2026-10-08

**Status**: Draft

**Input**: User description: "请为 Tops C++ language server 实现“tokenizer/parser”功能。范围包括标准 C++ 基础语法、Tops 关键字和属性、宏条件、未完成输入以及错误恢复。必须列出支持和不支持的语法、正向/负向/不完整测试、诊断位置规则和与 Clang 的对照方法。语义所有权属于 Go server。真实的 Tops kernel 代码在 topsop 的 topsop/lib/kernel/cc_kernel 目录中，可以用来参考测试编写，或直接用来功能验证。目标：让 Go server 能解析标准 C++ 和 Tops 扩展，并能处理未完成输入。交付物：Go parser 代码、parser 单测和语法测试材料。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 解析标准 C++ 基础语法 (Priority: P1)

作为 Tops C++ 开发者，我希望 Go server 能解析常见的标准 C++ 源码结构，从而在还没有加入 Tops 扩展时也能持续编辑普通的 C++ 声明、函数和模板。

**Why this priority**: 标准 C++ 是 Tops 源文件的基础。标准语法不能被误判为 Tops 错误，否则后续所有扩展诊断都会失去可靠输入。

**Independent Test**: 只使用不包含 Tops 扩展的 C++ 测试材料，运行 tokenizer/parser 单测，检查 token、语法树、错误恢复和源位置；该测试不需要 GCU 设备或 Clang 运行时依赖。

**Acceptance Scenarios**:

1. **Given** 一个包含命名空间、类型、函数、控制流、模板和 C++11/14/17 基础表达式的源文件，**When** Go server 解析它，**Then** 返回完整的翻译单元结构，不产生 Tops 专属诊断。
2. **Given** 一个标准 C++ 文件只缺少一个分号或右括号，**When** Go server 解析它，**Then** 产生一个带稳定范围的语法诊断，并保留后续声明和函数结构。
3. **Given** 同一个测试材料在 C++11、C++14 和 C++17 上下文中解析，**When** 语法属于对应标准，**Then** 结果使用上下文中的标准；不属于当前标准的受限语法必须给出明确诊断或 `unsupported-syntax`，不得静默接受。

### User Story 2 - 解析 Tops 关键字、属性和 kernel launch (Priority: P1)

作为 Tops kernel 开发者，我希望 Go server 能识别执行空间、内存空间、向量和启动属性，并能解析 `<<<...>>>` kernel launch，从而真实的 `.tops` kernel 在编辑时不会被当作普通 C++ 的非法 token 流。

**Why this priority**: Tops 修饰符和 kernel launch 是 Tops 源文件区别于普通 C++ 的直接语法入口，也是 `topsop/topsop/lib/kernel/cc_kernel` 中最常见的结构之一。

**Independent Test**: 使用最小 Tops 测试材料和从 `topsop` corpus 选取的缩减测试材料，分别检查声明、属性参数、launch 配置、模板和 host/device 文件中的语法树节点。

**Acceptance Scenarios**:

1. **Given** 一个包含 `__global__`、`__device__`、`__host__`、`__shared__`、`__local__`、`__valigned__` 或 `__vector` 的源文件，**When** Go server 解析它，**Then** 每个修饰符都作为带原始 spelling 和范围的 Tops 节点保存。
2. **Given** 一个 host 文件使用 `kernel<<<grid, block, shared, stream>>>(args)`，**When** Go server 解析它，**Then** 将配置列表、被调用名称和普通参数分开保存，不把 `<<<` 或 `>>>` 报告为普通比较运算符错误。
3. **Given** 一个 Tops 属性参数不完整或括号不配对，**When** 用户继续输入，**Then** parser 返回可恢复节点和局部诊断，不清空同一文件中已经识别的函数、模板和宏分支。

### User Story 3 - 处理宏条件和目标上下文 (Priority: P1)

作为需要在多个 GCU 目标上维护 kernel 的开发者，我希望 Go server 能保留和解释 `#if`、`#elif`、`#else`、`#endif` 条件区域，从而在编辑 `__GCU_ARCH__`、`__EFGCU_ARCH__` 等宏控制的代码时不会把非活动分支当作当前代码报错。

**Why this priority**: 真实 `cc_kernel` 文件使用宏条件选择目标相关实现。错误处理非活动分支或把未知宏当成已知值，会同时造成漏报和误报。

**Independent Test**: 使用包含嵌套条件、未定义宏、宏定义、分支切换和未知条件的测试材料；分别提供已知宏集合、空宏集合和冲突宏集合，比较活动状态和诊断。

**Acceptance Scenarios**:

1. **Given** `CompilationContext` 提供某个条件宏的值，**When** parser 解析条件区域，**Then** 为每个区域记录 `active`、`inactive` 或 `unknown`，且 inactive 分支中的普通语法错误不产生当前文档诊断。
2. **Given** 条件表达式引用未提供的宏，**When** parser 继续解析，**Then** 保留两个可能分支，生成一个定位到条件表达式的受限分析诊断，不擅自选择 GCU target 或宏值。
3. **Given** `#if`、`#elif`、`#else` 和 `#endif` 嵌套不平衡，**When** parser 到达错误指令或文件结尾，**Then** 只报告对应的预处理结构错误，并继续解析后面的独立声明。

### User Story 4 - 在未完成输入中保持结构并恢复 (Priority: P1)

作为在编辑器中逐字输入 kernel 的开发者，我希望文件在括号、字符串、模板、属性、注释或预处理条件尚未闭合时仍能获得稳定的解析结果，从而可以继续输入而不触发 server 崩溃或整文件诊断丢失。

**Why this priority**: LSP 的主要输入是中间编辑状态，不完整输入是常态。错误恢复质量直接决定诊断是否能用于编辑，而不是只能用于编译完成的文件。

**Independent Test**: 对同一有效测试材料依次删除末尾 token、截断任意嵌套结构并通过文档变更序列提交；检查每个中间版本都返回 `ParseResult`，并且后续有效声明仍可被找到。

**Acceptance Scenarios**:

1. **Given** 输入在字符串、字符、原始字符串或块注释中结束，**When** parser 到达 EOF，**Then** 保留已消费文本，生成一个 `unterminated` 诊断，并把位置指向对应起始分隔符到 EOF 的原文范围。
2. **Given** 输入在模板参数、函数参数、initializer、属性参数或 kernel launch 配置中结束，**When** parser 到达 EOF，**Then** 插入缺失节点并在 EOF 产生零宽诊断，同时保留外层声明名称。
3. **Given** 一个错误结构后紧接着另一个完整函数或宏区域，**When** parser 恢复，**Then** 后续结构可独立解析，级联错误不会覆盖后续节点的 source range。

### User Story 5 - 以 Go server 为唯一语义所有者验证结果 (Priority: P2)

作为语言服务器维护者，我希望 tokenizer、parser、诊断和宏分支状态由 Go server 自己产生，而不是运行时依赖 Clang 或由客户端重复判断，从而能维护稳定的 LSP 行为并独立演进 Tops 语法覆盖。

**Why this priority**: 语义所有权是本功能的架构约束。Clang 可以帮助核对事实和兼容性，但不能变成 Go server 的隐藏后端、fallback 或诊断转发器。

**Independent Test**: 在没有启动 Clang、clangd 或 GCU 设备时运行 Go parser 单测和测试材料检查；另行运行 Clang 对照命令，比较分类和位置，并确认运行时结果不依赖对照命令。

**Acceptance Scenarios**:

1. **Given** Go parser 可用但 Clang 不在 PATH，**When** server 解析标准 C++ 和已支持 Tops 测试材料，**Then** parser 单测和本地 server 分析仍能完成。
2. **Given** Go parser 与 Clang 对同一测试材料的分类不同，**When** 运行对照检查，**Then** 测试报告差异和编译上下文，不直接把 Clang 输出作为用户诊断或运行时 fallback。
3. **Given** 客户端收到 parser 诊断，**When** 用户修改文档，**Then** 下一次诊断由 Go server 根据新文本和新 context version 生成；客户端只展示结果，不重新解释语义。

### Edge Cases

| 场景 | 预期行为 | 验证方式 |
| --- | --- | --- |
| 文件以 UTF-8 多字节字符或补充平面字符开头 | token byte offset 可内部使用字节，但公开诊断转换为 LSP UTF-16 code unit | 位置测试材料同时检查 UTF-8 byte、rune 和 UTF-16 character |
| 文件使用 CRLF | `\r` 不计入行内可见字符范围，行尾位置仍按 LSP 规则计算 | CRLF token/range 单测 |
| `#if` 条件未知 | 保留可能分支，报告一个 `unknown-condition` 或 `analysis-degraded` 诊断，不选择默认架构 | 空宏 context 与部分宏 context 对照 |
| inactive 分支包含明显非法 token | 不报告该分支的普通语法错误，但仍检查预处理指令嵌套结构 | active/inactive 双分支测试材料 |
| `#endif` 缺失或多余 | 诊断定位到指令或 EOF，后续顶层声明仍尝试解析 | nested preprocessor 测试材料 |
| `<<<` 只输入一部分 | 创建不完整 launch 节点，不按移位运算表达式强行恢复 | 逐字符输入测试 |
| `>>>` 出现在模板关闭和 launch 关闭的相邻位置 | 根据当前 parser 状态区分模板关闭、launch 关闭和普通 token | template/launch 混合测试材料 |
| 宏定义使用反斜杠换行 | 将连续物理行作为一个宏定义处理，保留每个物理行的原文范围 | multiline macro 测试材料 |
| 宏条件含 token paste 或 stringification | 词法层保留 `##`/`#`，条件求值不展开其结果；给出受限状态而不是猜值 | macro expansion boundary 测试材料 |
| 未知 Tops 属性但括号完整 | 保留为 opaque attribute，不破坏后续声明；不冒充已支持语义 | unknown attribute 测试材料 |
| 已识别 Tops 属性参数数量错误 | 产生属性参数诊断，但保留属性节点和声明节点 | negative attribute 测试材料 |
| 真实 `cc_kernel` 缺少完整 include 根 | 仍解析当前文本和可识别的局部结构；把 header 解析缺口标为受限上下文 | corpus manifest + missing include test |
| 同一文档的旧 parse 结果晚于新版本返回 | 旧结果不得发布，或必须被 server 依据 `context_version` 丢弃 | document change/stale result test |
| 一处错误导致多个后续 expected token 缺失 | 只保留有区分价值的主诊断，避免同一恢复节点产生重复级联错误 | diagnostic de-duplication test |

## Scope and Boundaries

### In Scope

- Go server 自有的 C++ tokenizer 和 parser 代码，以及 parser 所需的 AST/恢复节点和诊断数据。
- C++11、C++14、C++17 基础语法的可回归解析覆盖。
- Tops 执行空间、存储空间、向量/数值 spelling、启动/资源属性、内建变量 spelling 和 kernel launch 语法。
- `#include`、宏定义和条件指令的词法/结构解析，以及 `active`、`inactive`、`unknown` 分支状态。
- 经过 `didOpen` 或成功 `didChange` 的文档解析，并由 Go server 发布 parser diagnostics；不新增客户端语义判断。
- 未完成输入、EOF 恢复、错误节点、缺失 token 和诊断位置规则。
- Go parser 单元测试、正向/负向/不完整/宏条件测试、诊断范围测试和语法测试材料。
- 从 `/home/carl.du/work/topsop/topsop/lib/kernel/cc_kernel` 选取并缩减真实 Tops kernel 结构作为 corpus 参考或 parser 验证输入。
- 使用同一 source、编译标准、宏、include 和 target 参数与 Clang 做离线对照验证。

### Out of Scope

- 完整 ISO C++20/C++23 语法，包括 modules、concepts/requires、coroutines、C++20 ranges 语法和后续标准新增语法。
- 完整预处理器语义，包括跨文件 include 加载、完整宏展开、token pasting `##`、stringification `#`、宏递归展开和条件之外的宏替换等价性。
- 模板实例化、完整类型检查、重载决议、常量求值、ODR、链接和代码生成语义。
- Tops DTE、barrier、GEMM、同步、资源占用、地址空间布局和设备执行正确性；本功能只解析其可见声明、调用和属性 spelling。
- 补全、悬停、定义跳转、引用查找、索引和性能分析；这些能力只在后续功能消费 parser 输出。
- 调用 clangd、修改 clangd、在 Go server 运行时启动 Clang，或把 Clang 诊断转发为用户结果。
- 修改 `/home/carl.du/work/llvm-project` 或 `/home/carl.du/work/topsop` 中的源码、header、编译配置和测试；它们只作为只读证据或验证 corpus。
- GCU 设备、GCUSIM、测试机 Docker 和 workload 执行；本功能的 parser 单测在开发机本地 Go 环境完成。

## Supported Syntax

本表定义 v1 parser 的支持边界。“支持”表示 tokenizer/parser 能形成稳定节点并按本规格产生诊断，不表示已经完成全部 C++ 语义或保证目标代码可生成。

### Standard C++ Baseline

| 类别 | 支持内容 | 约束 |
| --- | --- | --- |
| 词法 | 空白、换行、行注释、块注释、标识符、关键字、整数字面量、浮点字面量、字符/字符串/原始字符串、转义、常用 C++ 运算符和标点 | token range 必须来自原文；未知字符必须产生稳定 lexical diagnostic |
| 翻译单元 | 顶层声明、声明序列、空声明、带分号声明、`#include` 和属性序列 | 不要求加载被 include 文件的完整语义 |
| 名称和作用域 | namespace、nested namespace、匿名 namespace、using declaration/directive、typedef、alias declaration、qualified-id、依赖名称 | 保留原始名称和限定符，不做完整符号解析 |
| 类型声明 | 基础类型、cv/ref、指针、数组、函数类型、`auto`、`decltype`、enum、struct、class、union、成员声明、构造/析构声明 | 解析声明形状，不做完整类型兼容检查 |
| 函数 | 参数、返回类型、函数体、默认参数、重载 spelling、lambda、尾置返回类型、函数指示符和基本声明属性 | 函数调用合法性不在本功能范围 |
| 语句 | compound、表达式语句、声明语句、`if/else`、`switch/case/default`、`for`、range-for、`while`、`do`、`return`、`break`、`continue`、`goto`、label、`try/catch` | 恢复时优先保留块边界和后续顶层声明 |
| 表达式 | literal、name、qualified-id、调用、下标、成员访问、指针成员、前后缀运算、二元/三元/赋值/逗号表达式、cast、`sizeof`、`alignof`、`decltype`、`new/delete`、initializer list | 运算符优先级和括号结构必须稳定；不保证类型推导 |
| 模板 | template parameter list、函数/类模板声明、显式模板参数、模板特化和实例化 spelling、dependent qualified-id、fold expression 的 C++17 基础形状 | 不实例化模板，不计算 substitution failure |
| C++11/14/17 | `constexpr`、`nullptr`、range-for、lambda、variadic template、`decltype`、`auto`、generic lambda、return type deduction、variable template、`if constexpr`、structured binding、fold expression、nested namespace 等基础语法 | 实际可用标准由 `CompilationContext.language_standard` 决定；不把高标准语法静默降级为低标准语法 |
| 标准属性 | `[[...]]` 属性序列、属性参数中的平衡 token、声明前后属性位置 | 未知属性作为 opaque attribute；不判断实现定义语义 |

### Tops Syntax

| 类别 | 支持 spelling/结构 | 解析行为 |
| --- | --- | --- |
| 执行空间 | `__global__`、`__device__`、`__host__`、`__host__ __device__`、`__cooperative__`、`__sp__`、`__scalar_only__` | 形成 function qualifier 节点；组合顺序和重复项由 parser 记录并可诊断 |
| 内联和函数属性 | `__noinline__`、`__forceinline__`、`__alwaysinline__`、`__force_noinline__`、`__inline_hint__` | 形成 attribute/qualifier 节点；不执行优化语义 |
| 存储空间 | `__device__`、`__constant__`、`__shared__`、`extern __shared__`、`__local__`、`__private__`、`__cluster_shared__`、`__local_stack__`、`__mmu_pointer__`、`__restrict__` | 保存 qualifier、声明主体和 pointer/declarator 结构；目标限制交给后续 semantic 功能 |
| 启动和资源属性 | `__thread_dims__(...)`、`__cluster_dims__(...)`、`__launch_bounds__(...)`、`__maxnreg__(...)`、`__block_tile__(...)` | 解析名称、平衡参数 token 和逗号分隔表达式；不计算资源合法上限 |
| 对齐和向量 | `__valigned__`、`__vector T`、`__vector2 T`、`__vector4 T`、`__vector8 T`、`__vector bool` | 形成 type/attribute 节点；保留元素类型和 spelling，不推断跨架构向量宽度 |
| 扩展数值类型 | `__fp16`、`__bf16`、活动 Tops headers 中出现的 FP8/FP6/FP4 类型 spelling | 作为可解析类型名称或扩展类型节点；实际可用目标由后续 context/semantic 处理 |
| 内建变量 spelling | `threadIdx`、`blockIdx`、`blockDim`、`gridDim`、`threadDim`、`subThreadIdx`、`warpSize` | 普通名称和成员访问先形成 AST；parser 不在无 context 时硬编码其可用目标 |
| kernel launch | `callee<<<config-list>>>(argument-list)`，包括一到多个配置表达式、尾随逗号和不完整配置 | 形成独立 launch expression，区分配置列表和调用参数列表 |
| 宏化 Tops 类型/属性 | 例如 `__TCLE_DEVICE_TYPE__` 等由 header 或工程定义的宏名称 | tokenizer 按宏标识符处理，条件区域保留其原文；不把任意宏自动升级为 Tops 关键字 |
| Tops 属性包裹形式 | 由 `__attribute__((...))` 或等价 Tops spelling 形成的平衡属性 token | 解析属性名称和参数 token；未知属性保持 opaque，不静默删除 |

### Macro Conditions

支持以下预处理结构和条件表达式的结构解析：

- 指令：`#include`、`#define`、`#undef`、`#if`、`#ifdef`、`#ifndef`、`#elif`、`#else`、`#endif`、`#pragma` 和未知指令的行级保留。
- 宏定义：对象宏、函数宏、参数列表、可变参数 spelling、反斜杠换行和原始 replacement token 保留。
- 条件表达式：标识符、整数常量、`defined(NAME)`、`defined NAME`、括号、`!`、`&&`、`||`、`==`、`!=` 以及可被 tokenizer 识别的剩余 token。
- 分支状态：在 `ParseContext` 提供值时计算 `active`/`inactive`；无法确定时为 `unknown`，不得使用隐含 GCU300 或其他架构默认值。
- 条件区域：保留每个分支的原始范围、指令范围、父子嵌套关系和条件表达式 token；inactive 文本仍保留 token/region，但不参与当前普通语法诊断。

## Unsupported Syntax

以下内容在 v1 不承诺语义支持。parser 对能够识别边界的 unsupported construct 必须生成 `opaque` 或 `unsupported` 节点和稳定诊断，不能静默当作完整支持；单纯未知标识符仍按普通 C++ 标识符处理。

| 类别 | 不支持内容 | 处理方式 |
| --- | --- | --- |
| 新标准语法 | C++20/23 modules、`import`/module declaration、concepts/requires、coroutines、C++20 ranges 专属语法和未列入 baseline 的新语法 | 在可定位构造范围产生 `unsupported-syntax`，恢复到下一个声明/分号/块边界 |
| 完整宏展开 | 跨文件宏展开、递归展开的完整等价性、token paste `##` 的结果、stringification `#` 的结果、复杂 variadic replacement 语义 | 保留原始 token 和宏定义；条件求值不猜测展开结果，必要时报告受限分析 |
| Include 语义 | 递归加载所有 header、include guard 的全局求值、系统 header 的完整符号索引 | 解析指令并保留路径 token；缺少外部内容时不把缺口当作 parser 崩溃 |
| 完整 C++ 语义 | 模板实例化、overload resolution、隐式转换、常量求值、ODR、链接、完整类型系统和调用可达性 | 不在 parser 阶段生成语义结论；由后续 Go semantic 层负责 |
| Tops 硬件语义 | DTE/SDTE、barrier、GEMM、同步、资源上限、地址空间布局、线程调度、性能和设备执行结果 | 只解析声明/调用/属性结构；不声称硬件行为正确 |
| 编译器后端语法 | 汇编指令、机器编码、设备链接脚本、CodeGen metadata 的完整约束 | 作为 opaque body 或 unsupported construct 保留边界 |
| 工具功能 | completion、hover、definition、references、semantic tokens、格式化和重构 | 本功能只提供 parser 输出和 parser diagnostics；后续功能消费这些结果 |

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 提供 Go tokenizer/parser 实现，输入为当前文档文本和可选 `ParseContext`，输出为可恢复语法树、token 结果、条件区域和 parser diagnostics。
- **FR-002**: Go server MUST 负责本功能的语法分类、Tops 扩展识别、宏条件状态、错误恢复和 parser diagnostics；TypeScript client、clangd 和运行时 Clang 不得拥有这些判断。
- **FR-003**: tokenizer MUST 识别标准 C++11/14/17 baseline 所需 token、注释、字面量、Tops spelling、预处理指令和 source range；无法识别的字符必须产生可定位 lexical diagnostic，不得静默丢弃。
- **FR-004**: parser MUST 解析本规格 Supported Syntax 中列出的翻译单元、声明、类型、函数、模板、表达式、语句、属性、宏条件和 Tops kernel launch 结构。
- **FR-005**: parser MUST 将 Tops 执行空间、存储空间、向量/数值、启动/资源属性和内建变量 spelling 保存为可区分节点，并保留原始 spelling、参数 token 和 source range。
- **FR-006**: parser MUST 能区分普通 C++ 的 `<`/`>` 运算结构、模板关闭和 Tops `<<<...>>>` launch；遇到不完整 launch 时必须使用恢复节点，不得把整个剩余文件当作移位表达式错误。
- **FR-007**: parser MUST 解析 `#if`、`#ifdef`、`#ifndef`、`#elif`、`#else`、`#endif` 的嵌套结构，并保存每个分支的原始范围、父级关系和条件表达式。
- **FR-008**: `ParseContext` MUST 支持传入预定义宏名称和值、C++ language standard、document URI、document version 和 context version；target profile 可作为不透明标识传入，但 parser 不得根据名称猜测宏值。
- **FR-009**: 宏条件求值 MUST 使用 `active`、`inactive`、`unknown` 三态；缺少宏、冲突宏或无法安全求值时必须保持 `unknown`，并产生一个可定位的受限分析结果。
- **FR-010**: inactive 分支中的普通 C++/Tops 语法错误 MUST 不发布为当前活动文档的 parser diagnostic；预处理指令自身的嵌套错误仍必须诊断。
- **FR-011**: parser MUST 支持未闭合注释、字符串、字符、原始字符串、括号、花括号、方括号、模板参数、属性参数、宏条件和 launch 配置的恢复，并返回非空 `ParseResult`。
- **FR-012**: 错误恢复 MUST 生成可识别的 `ErrorNode`、`MissingToken` 或等价恢复节点，保留已解析的前缀、后续独立声明、条件区域和原始 token；单个错误不得使 server 崩溃或清空整棵树。
- **FR-013**: 对于可识别但 v1 不支持的语法，parser MUST 产生 `unsupported-syntax` 诊断和恢复边界；对未知但语法上合法的标识符或完整 opaque attribute，不得误报为 unsupported。
- **FR-014**: parser diagnostics MUST 使用稳定 code、severity、message、source range 和 recoverability 标记；至少包含 `unexpected-token`、`missing-token`、`unterminated`、`invalid-directive`、`unknown-condition`、`unsupported-syntax` 和 `invalid-tops-attribute` 类别。
- **FR-015**: 所有公开位置 MUST 使用 LSP 0-based line 和 UTF-16 code-unit character，范围为半开区间 `[start, end)`；内部 byte offset 可以存在，但不得直接作为 LSP character 输出。
- **FR-016**: 对 unexpected token，主范围 MUST 覆盖完整 offending token；对 missing token，主范围 MUST 是预期插入点的零宽范围；对 EOF 未闭合结构，主范围 MUST 覆盖起始分隔符到 EOF，并关联起始分隔符位置。
- **FR-017**: 诊断位置 MUST 基于原始文档文本，不基于宏展开后的虚拟文本；宏相关结果应在条件指令或调用位置给出主范围，必要时保存宏定义位置作为 related information。
- **FR-018**: parser MUST 对一个恢复节点抑制重复级联诊断，但不得抑制恢复节点之后能独立定位的错误；同一文档每个 diagnostic 的 range、code 和 version 必须可稳定复现。
- **FR-019**: Go server 在 `didOpen` 和成功 `didChange` 后 MUST 使用当前文档版本和 context version 调用 parser，并通过标准 `textDocument/publishDiagnostics` 发布 parser diagnostics；旧版本结果不得覆盖新版本。
- **FR-020**: `publishDiagnostics` 只允许发送 Go server 生成的诊断；客户端不得根据关键字、目标名、Clang 输出或测试材料自行补发语义诊断。
- **FR-021**: parser 单测 MUST 在没有 GCU 设备、clangd 和运行时 Clang 的环境中覆盖 token、AST、宏状态、恢复和位置规则；对照测试可以单独依赖配置好的 Clang。
- **FR-022**: 测试 MUST 包含正向、负向、不完整、宏条件、恢复、目标边界和真实 `cc_kernel` corpus 场景；每个测试材料必须记录输入标准、宏 context、预期诊断和是否需要 Clang 对照验证。
- **FR-023**: 真实 corpus 验证 MUST 只读 `/home/carl.du/work/topsop/topsop/lib/kernel/cc_kernel`；可以使用缩减副本或 manifest 引用，不得修改 topsop 源文件，不得将设备运行结果当作 parser 通过条件。
- **FR-024**: Clang 对照 MUST 使用与 Go parser 相同的源文本、language standard、include roots、predefined macros、target/pass 参数，并比较接受/拒绝分类、宏分支、AST 结构和诊断位置；Clang 只作为离线 对照验证。
- **FR-025**: 当 Clang 与 Go parser 结果不一致时，测试 MUST 保存两侧编译上下文、输入版本、分类、range、诊断 code 和差异原因；不得通过调用 Clang 运行时补齐 Go parser 结果。
- **FR-026**: 本功能 MUST 不修改 `llvm-project`、`topsop` 或 clangd，不引入 clangd/Clang runtime dependency，不创建 TypeScript 语义实现；任何新增 Go 依赖必须在实施计划中说明理由和验证命令。
- **FR-027**: parser MUST 只接收上游 `CompilationContext` 归一化后的 `ParseContext`，其中可以包含 `topscc`/直接 Clang 的 driver provenance、language、target/pass、include 和宏状态；parser 不得解析 raw compiler argv、response file 或启动 `topscc`。
- **FR-028**: 测试材料 manifest MUST 能记录 `driver_kind`、raw/normalized argument provenance、context status 和离线对照命令来源；这些记录用于复现输入，不改变 Go parser 的语义所有权。

## Diagnostic Position Rules

诊断位置规则是 parser 和 LSP 层的共同契约，测试材料必须逐项验证。

1. **坐标系统**：源位置使用 0-based line 和 UTF-16 code-unit character；range 为半开区间。内部 token 可以保存 UTF-8 byte offset 和 rune offset，但外部只能发布转换后的 LSP 位置。
2. **词法错误**：非法字符或非法数字/转义 token 的 range 覆盖最小完整 offending token；如果无法形成 token，覆盖单个原始 Unicode scalar value 的 UTF-16 范围。
3. **unexpected token**：range 覆盖完整 unexpected token，不向前扩展到前一个空格或注释；message 中指出期望类别而不是依赖完整 Clang 文案。
4. **missing token**：range 为零宽插入点。插入点位于 parser 首次确认缺失的位置；若在 EOF 确认，插入点为文档末尾。opening delimiter 作为 related information 保留。
5. **未闭合结构**：字符串、字符、原始字符串和块注释的主 range 从 opening delimiter 起始到 EOF；括号、模板、属性和 launch 的 missing close 使用 EOF 零宽主 range，并关联 opening delimiter。
6. **预处理指令**：directive 名称拼写错误、分支顺序错误和多余 `#endif` 的主 range 覆盖指令关键字到该行末尾；缺失 `#endif` 的主 range 为 EOF 零宽，related information 指向对应 `#if`。
7. **未知条件**：`unknown-condition` 主 range 覆盖条件表达式，不覆盖整个分支；分支内只抑制依赖活动状态的普通诊断，结构性预处理错误仍可诊断。
8. **不支持语法**：主 range 覆盖可识别 unsupported construct 的最小完整边界；若边界未闭合，使用当前已消费范围并标记 recoverable，不伪造文件末尾的大范围错误。
9. **Tops 属性**：属性名称或参数错误的 range 只覆盖属性或具体参数；声明主体的 range 仍保持独立，便于后续解析和导航。
10. **错误恢复**：同一恢复节点只保留最早且能解释根因的诊断；恢复节点后的独立 token 若再次违反语法，必须拥有自己的 range 和 diagnostic。
11. **宏来源**：诊断 range 指向用户当前打开的原始文本；宏定义和宏调用同时相关时，主位置指向触发处，宏定义位置放入 related information，不能输出展开后不存在的行列。
12. **版本与发布**：每条诊断带 document version/context version 归属。server 发布新版本结果前必须丢弃旧版本结果，关闭文档后不得继续发布该文档诊断。

建议的稳定 code 集合如下；实现可以增加子 code，但不得改变已有 code 的含义：

| Code | 用途 | 默认严重级别 |
| --- | --- | --- |
| `tops-syntax-unexpected-token` | token 出现在当前 grammar 位置之外 | Error |
| `tops-syntax-missing-token` | 能确定需要插入的 token | Error |
| `tops-syntax-unterminated` | 字符串、注释、delimiter 或属性未闭合 | Incomplete/Hint |
| `tops-syntax-invalid-literal` | 数字、字符或字符串 literal 结构非法 | Error |
| `tops-syntax-invalid-directive` | 预处理指令或条件嵌套非法 | Error |
| `tops-syntax-unknown-condition` | 条件无法依据当前宏 context 求值 | Information |
| `tops-syntax-unsupported` | 可识别但超出 v1 范围的语法 | Information |
| `tops-syntax-invalid-tops-attribute` | Tops 属性名称、参数或位置结构非法 | Error |

## 测试计划和语法测试材料

### 测试材料目录

实现阶段在 `tops-lsp` 内新增 parser 测试材料根目录；现有 `testdata/lsp/` 继续只服务传输和文档同步，不混入 C++ 语法样例。

```text
internal/parser/
  tokenizer.go
  parser.go
  ast.go
  conditions.go
  diagnostics.go
  tokenizer_test.go
  parser_test.go
  recovery_test.go
  diagnostics_test.go

testdata/parser/
  positive/
  negative/
  incomplete/
  macros/
  topsop/
  expected/
  manifest.json
```

每个测试材料条目至少包含：源文件路径、`language_standard`、宏名称和值、target/pass 的不透明标识、是否允许 unknown branch、预期 token/节点摘要、预期 diagnostics code/range、是否需要 Clang 对照。测试材料可以用 `.cpp`、`.tops` 或 `.h` 扩展名；扩展名不能替代显式 context。

### Positive Tests

| ID | 内容 | 最小预期 |
| --- | --- | --- |
| `PARSER-POS-STD` | 命名空间、using、class/struct、函数、控制流、模板、lambda、初始化列表和 C++11/14/17 基础语法 | 无 parser error；顶层声明、函数和模板节点数量稳定 |
| `PARSER-POS-TOPS-QUALIFIER` | `__global__`、`__device__`、`__host__ __device__`、`__shared__`、`__local__`、`__private__`、`__valigned__`、`__vector` | 每个 Tops spelling 有独立节点和原文范围 |
| `PARSER-POS-ATTRIBUTE` | `__thread_dims__(...)`、`__cluster_dims__(...)`、`__launch_bounds__(...)`、`__maxnreg__(...)`、标准 `[[...]]`、`__attribute__((...))` | 属性名称和参数 token 平衡，声明主体可继续解析 |
| `PARSER-POS-LAUNCH` | `kernel<<<grid, block>>>`、带 shared/stream 的配置、模板 kernel launch 和普通参数 | 生成独立 launch node，配置和调用参数不混淆 |
| `PARSER-POS-MACRO` | `#define`、函数宏、续行、`#if/#elif/#else/#endif`、`defined` 和嵌套条件 | 分支状态和区域范围与预定义宏 context 一致 |
| `PARSER-POS-CC-KERNEL` | 从 `range/range_kernel.tops`、`range/range_host.tops` 选取或缩减模板、namespace、`#if`、DTE 类型、属性和 launch 片段 | 在包含足够 context 时解析局部结构；缺 header 时只报告受限 context |
| `PARSER-POS-CC-KERNEL-VECTOR` | 从 `mhc_pre/mhc_pre_n512_kernel_gcu400.tops` 选取 `__vector`、`__device__` 和类型别名片段 | 类型声明和函数结构保持可恢复 |
| `PARSER-POS-CC-KERNEL-DIMS` | 从 `topp_renorm_probs/topp_renorm_probs_kernel_gcu400.tops` 选取 `__thread_dims__` 与 `__global__` 片段 | 属性参数和 kernel 声明节点稳定 |

### Negative Tests

| ID | 内容 | 最小预期 |
| --- | --- | --- |
| `PARSER-NEG-STD` | 缺分号、缺右括号、错误模板参数边界、非法声明符和错误表达式 | 产生对应稳定 code/range，不崩溃，后续声明仍解析 |
| `PARSER-NEG-TOPS` | Tops qualifier 放在无法解析的位置、属性参数为空或多余逗号、非法 launch 分隔符 | 产生 `tops-syntax-invalid-tops-attribute` 或 `tops-syntax-unexpected-token` |
| `PARSER-NEG-LAUNCH` | `kernel<<<>>>`、缺少调用括号、配置括号嵌套错误、模板关闭与 launch 关闭冲突 | 不把整个文件解析成 shift expression；错误范围局部可定位 |
| `PARSER-NEG-MACRO` | 多余 `#endif`、`#elif` 位于错误位置、缺 directive 名、坏的条件表达式 | 产生 `tops-syntax-invalid-directive`，普通后续声明仍保留 |
| `PARSER-NEG-UNSUPPORTED` | C++20 concepts/module/coroutine 或其他明确超出范围的构造 | 产生 `tops-syntax-unsupported`，保留 opaque/recovery node |
| `PARSER-NEG-UTF16` | UTF-8 多字节内容前后的非法 token、CRLF 和诊断跨行 | 诊断 byte offset 与 LSP UTF-16 range 转换一致 |

### Incomplete Tests

| ID | 截断位置 | 最小预期 |
| --- | --- | --- |
| `PARSER-INC-COMMENT-LITERAL` | `/*`、`"`、`'`、raw string 开头 | `unterminated`，保留前置 token，范围符合规则 |
| `PARSER-INC-DELIMITER` | `(`、`{`、`[`、模板 `<`、属性 `[[` | EOF missing token 或 unterminated，后续已完成节点保留 |
| `PARSER-INC-TOPS-ATTRIBUTE` | `__thread_dims__(1,`、`__maxnreg__(`、`__vector` | 恢复属性/type node，诊断指向 EOF 或当前 token |
| `PARSER-INC-LAUNCH` | `kernel<<<`、`kernel<<<grid,`、`kernel<<<grid>>` | 生成 incomplete launch node，不走普通 shift/relation 分支 |
| `PARSER-INC-MACRO` | `#if defined(`、续行末尾、未闭合 `#if` | 保留 conditional region，给出 unknown/incomplete context 结果 |
| `PARSER-INC-TEMPLATE` | `template <typename T`、`foo<Bar,`、dependent qualified-id | 保留模板头/调用名称，并在 EOF 产生局部恢复诊断 |
| `PARSER-INC-RECOVERY` | 有效函数后追加半个声明，再追加有效函数 | 后一个完整函数仍能被识别，诊断不跨越其范围 |

### Macro and Target Boundary Tests

- 使用空宏 context、只提供 `__GCU_ARCH__`、只提供 `__EFGCU_ARCH__`、提供冲突值和提供非数字宏的 context 分别运行同一测试材料；parser 只使用实际传入值，不按 profile 名称推断。
- 对 `#if defined(__GCU_ARCH__)`、`#if defined(__EFGCU_ARCH__)`、数值比较、嵌套分支和未知宏分别检查 `active`、`inactive`、`unknown`。
- 同一 `cc_kernel` 片段在 `.tops`、`.cpp` 加 Tops language context 和 header 缺失 context 下运行，检查源 range 不变，受限状态只改变诊断和分支状态。
- target boundary 测试材料只验证宏条件和 parser 可见结构，不把 GCU300/GCU400/GCU450/GCU500/EFGCU500 的硬件资源或指令语义写成 parser 通过条件。

### Clang Comparison Method

Clang 对照是离线验证流程，不是 Go server 的运行路径。每个需要对照的测试材料必须保存完整命令、工具版本、工作目录和输出摘要。

1. **统一输入**：使用与 Go `ParseContext` 相同的源文件 bytes、`language_standard`、宏定义、include 顺序、target/pass 参数和工作目录。不要用不同的默认标准或系统 include 代替。
2. **驱动核对**：先用配置好的本地 LLVM/Tops `clang -###` 展开实际 cc1 参数，记录 language、target、CPU、device/pass、include 和宏来源；如果命令无法复现，测试材料状态为 `context-invalid`，不把 Go 结果与默认 Clang 结果直接比较。
3. **宏/预处理对照**：使用 `-E -dD` 或等价预处理输出核对 directive、宏定义和活动分支；使用 `-dM -E` 核对预定义宏集合。比较分支状态和源区域，不要求 Go 复制 Clang 的完整宏展开文本。
4. **语法接受/拒绝对照**：使用 `-fsyntax-only`。正向测试材料比较是否接受和主要声明结构；负向测试材料比较错误类别和原文位置。Clang 的完整 message 文案不作为稳定契约。
5. **AST 对照**：对支持的声明/类型/函数/属性/模板/launch 片段，在适用时使用 `-Xclang -ast-dump=json`。去除隐式节点、工具链特有节点和宏展开偏移后，比较节点类别、嵌套关系、名称和原文范围。
6. **诊断对照**：在 Clang 支持 JSON 诊断时保存结构化输出；否则保存稳定的 file/line/column 和分类摘要。转换到 LSP UTF-16 后比较主 range、行列和类别，不要求 Go message 与 Clang 逐字相同。
7. **Tops 扩展差异**：如果 Clang 因 command/context 不同而拒绝某个 Tops spelling，先修正命令和 include；仍有差异时记录为测试材料例外，说明 Go parser 的目标、Clang 的目标和语法所有权，不把 Clang 输出转发给用户。
8. **真实 corpus**：对 `cc_kernel` 选定文件可先做本地 tokenizer/parser 解析，再用同一编译上下文执行 Clang syntax-only；缺少 header 或 target 时保留 `partial/context-invalid` 结果，不用设备 workload 替代语法对照。
9. **运行时隔离**：Go parser 测试必须在没有 Clang/clangd 时可运行；Clang 对照测试单独运行并输出差异报告，不能由 server 在运行时按需启动 Clang。

## Delivery and File Boundaries

### Required Deliverables

- Go tokenizer/parser 代码，归属 `tops-lsp` 的内部 Go package；具体公共 API 在 plan 中确定，但必须能从文档文本和 `ParseContext` 得到 `ParseResult`。
- parser 单测，至少覆盖 tokenizer、标准语法、Tops 语法、宏条件、未完成输入、错误恢复、诊断位置和 stale version 行为。
- syntax 测试材料，按 positive、negative、incomplete、macros 和 `topsop` corpus 分类，并带预期 token/节点/diagnostic 元数据。
- parser 与现有 Go server 文档生命周期的最小接入：open/change 后解析，发布 Go server 自有 parser diagnostics，关闭或旧版本结果不得继续发布。
- Clang 对照命令和差异记录格式；对照脚本或测试入口可以在 plan 中确定，但不得成为 server runtime dependency。

### File Boundaries

| 路径 | 允许内容 | 禁止内容 |
| --- | --- | --- |
| `internal/parser/` | tokenizer、parser、AST/recovery node、macro condition、diagnostic model 和 Go unit tests | 调用 clangd/Clang 运行时、设备执行、客户端语义判断 |
| `internal/server/` | 在文档 open/change 生命周期中触发 parser、管理 version/context version、发布 `publishDiagnostics` | 复制 grammar、按关键字自行生成诊断、改变既有 transport 生命周期规则 |
| `internal/protocol/` | 必要的 `publishDiagnostics` 参数模型和 parser diagnostic 到 LSP diagnostic 的编码 | parser grammar、target 语义和 Clang fallback |
| `internal/document/` | 继续保存文档文本和版本；如需提供不可变 snapshot，只增加与现有 store 契约一致的能力 | C++ token、AST、宏或 Tops 语义 |
| `testdata/parser/` | 独立语法测试材料、manifest、golden token/AST/diagnostic 摘要 | 密钥、完整用户源码、会修改的 topsop 源文件 |
| `testdata/lsp/` | 保持现有传输/文档同步测试数据 | C++ 语法测试材料 |
| `specs/004-tops-cpp-tokenizer-parser/` | 本规格、后续 plan/tasks/checklists/contracts | 不把未实现代码写成已支持事实 |
| `/home/carl.du/work/topsop/topsop/lib/kernel/cc_kernel/` | 只读 corpus 参考和 parser 验证输入 | 修改源码、重写 kernel 或把 workload 结果当 parser 证据 |
| `/home/carl.du/work/llvm-project/` | 只读 Clang/Tops 对照工具、headers 和测试 | 修改 Clang、clangd、headers、driver 或测试 |

## Key Entities *(include if feature involves data)*

- **Token**：原始文档中的词法单元，包含 kind、spelling、byte range、LSP range、行尾/注释信息和是否处于 inactive conditional region。
- **SourceRange**：原始 source 的半开范围，能够转换为 LSP UTF-16 range；不允许只保存展开文本位置。
- **SyntaxNode**：翻译单元、声明、类型、表达式、语句、属性、Tops qualifier、kernel launch、预处理指令或条件区域节点。
- **RecoveryNode**：`ErrorNode`、`MissingToken`、opaque 或 incomplete 节点，用于保留 parser 无法完整识别但仍有结构价值的输入。
- **ConditionalRegion**：`#if` 到匹配分支结束的区域，包含条件 token、分支列表、父级关系、原始范围和 `active`/`inactive`/`unknown` 状态。
- **ParseContext**：文档 URI、版本、C++ standard、预定义宏、target/pass 不透明标识、include 可用性和 context version；parser 只使用显式提供的值。
- **ParseResult**：token 序列、根语法树、条件区域、恢复信息、parser diagnostics、完成/受限状态和输入版本。
- **ParserDiagnostic**：稳定 code、severity、message、原文 source range、related information、recoverable/incomplete 标记和版本归属。
- **SyntaxFixture**：源文件、ParseContext、预期节点/诊断和可选 Clang 对照验证元数据组成的可重复测试输入。
- **ClangComparisonRecord**：同一测试材料的 Clang 命令、工具版本、接受/拒绝结果、宏/AST/诊断摘要、Go 结果和差异原因；只存在于离线验证。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**：每个 Supported Syntax 类别至少有 1 个正向测试材料、1 个负向测试材料或明确的“不适用”说明；标准 C++、Tops 扩展、宏条件和错误恢复四个核心域均有正向、负向和不完整测试材料。
- **SC-002**：所有 parser 测试材料在输入完整、截断、包含未知宏和包含 UTF-8/CRLF 的情况下都返回结构化 `ParseResult`；测试中不得出现 parser panic 或因单个错误导致的空结果。
- **SC-003**：诊断位置单测覆盖至少 20 个范围案例，包括 UTF-16、CRLF、EOF 零宽插入点、非法 token、未闭合结构、宏指令和 Tops 属性；所有案例的预期 range 与实际 range 完全一致。
- **SC-004**：至少 10 个来自 `/home/carl.du/work/topsop/topsop/lib/kernel/cc_kernel` 的代表性片段或 manifest 条目通过本地 parser 解析；其中至少覆盖 host/device 文件、`#if`、模板、Tops qualifier、属性、向量或 DTE spelling、kernel launch 中的 5 类结构。缺失外部 context 的条目必须明确标为 partial，而不是静默通过。
- **SC-005**：正向和负向参考编译器对照测试材料的接受/拒绝分类达到 95% 以上一致；剩余差异必须全部有完整编译上下文和已知原因，不得隐藏差异。
- **SC-006**：同一文档连续提交至少 100 个逐字编辑版本时，用户看到的结果只对应最近一次有效版本；旧结果不得覆盖新结果，且服务不崩溃。
- **SC-007**：在没有外部编译器或设备的环境中，基础语法检查仍可独立完成；参考对照不可用时不阻塞 tokenizer/parser 的基本验收。
- **SC-008**：文档打开、合法修改、非法或不完整输入以及关闭后的诊断都来自同一个权威分析服务；客户端无需自行扫描关键字或调用外部工具即可展示结果。
- **SC-009**：用户可从测试材料的 code、severity、range 和 recoverability 判断错误属于标准语法、Tops 属性、宏条件、未完成输入或 unsupported syntax；不得依赖 Clang 的完整文案才能理解结果。
- **SC-010**：参考源码和测试数据保持只读；新增依赖、命令、测试材料 manifest 和对照入口均在后续 plan/tasks 中有明确记录。

## Assumptions

- 本功能以当前 `tops-lsp` 的 Go server 为实现主体；现有 `003-basic-lsp-transport` 的生命周期、文档同步、请求取消和日志契约继续有效。
- 用户描述中的 `tospop` 按当前 workspace 的实际路径 `/home/carl.du/work/topsop/topsop/lib/kernel/cc_kernel` 解释。
- C++11/14/17 是本 feature 的标准基线；C++20/23 新语法在未来单独扩展，不因 tokenizer 能读到相关字符就视为已支持。
- parser 不默认选择 GCU300、GCU400 或其他 target；`__GCU_ARCH__`、`__EFGCU_ARCH__` 和其他宏的值必须来自显式 `ParseContext` 或编译数据库派生结果。
- 目标架构相关的硬件约束、header 可见性和 API 语义由后续 Go semantic 功能逐项验证；本 feature 只保证 parser 能保留相关 spelling 和条件结构。
- `cc_kernel` 真实文件可能需要 Tops headers、项目 include 和 target 参数才能完整分析；缺少这些内容时，parser 仍应提供局部语法结果并标记受限状态。
- Clang/Tops toolchain 仅用于离线事实核对、语法/预处理/AST/诊断对照；Go server 运行时不启动 Clang 或 clangd。
- parser 单测和本地测试材料不需要 GCU 设备或测试机；需要运行 Tops 编译器或设备 workload 的验证属于后续独立任务。
- 具体 Go package API、是否将 tokenizer 与 parser 放在同一 package、测试材料 golden 文件格式和对照脚本入口由 `/speckit.plan` 确定，但不得突破本规格的责任边界和交付物。
