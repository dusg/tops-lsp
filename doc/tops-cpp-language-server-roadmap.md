# Tops C++ Language Server Roadmap

**状态**：Draft

**发布日期**：2026-10-07

**最后验证**：2026-10-11

**最终发布路径**：`doc/tops-cpp-language-server-roadmap.md`

**状态说明**：`Draft` 表示本文尚未发布，不表示已有阶段没有代码或测试，也不表示后续阶段已经实现。源工件中的任务勾选不能单独改变本文状态；只有发布前检查全部通过并记录新的验证日期后，本文状态才可更新为 `Published`。

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
- [P1.1 基础服务和 LSP 传输规格](../specs/003-basic-lsp-transport/spec.md)
- [P1.2 tokenizer/parser 规格](../specs/004-tops-cpp-tokenizer-parser/spec.md)
- [P1.3 AST 和符号索引规格](../specs/005-ast-symbol-index/spec.md)
- [P1.4 通用 semantic baseline 规格](../specs/006-p1-semantic-baseline/spec.md)

`specs/002-p0-go-semantic-baseline/` 是早期 P0 参考工件；本路线图不把它作为当前 canonical gate。当前阶段门禁以 `specs/001-tops-lsp-roadmap/` 下的 contract 为准，保留 `specs/002` 现有内容不变。

## 规划原则

- 分开记录标准 C++ 和 Tops 扩展。
- 语法信息以当前 Tops headers、Clang attribute/TargetInfo、driver 和测试为准；旧汇总文档只用来查目录。
- Go server 从零负责 tokenizer/parser、AST、符号索引、目标语义、诊断和 LSP。
- 文中的 `Tops C++ language server` 指服务 Tops C++ 源代码的语言服务器，实现语言为 Go，不是面向 Go 语言的 language server。
- 语言能力按标准版本和实现层次分阶段交付；基础语法、已列 C++ grammar、预处理器/宏展开和 semantic baseline 分别定义完成条件，完整语义能力另行评估。
- TypeScript 只负责 VS Code 扩展和 LSP 客户端，包括启动、编辑器集成、配置、状态、日志和用户命令；不负责 Tops 语义、索引或 API 分析。
- 用户编译器优先使用 `topscc`；直接 `clang`/`clang++` 是显式兼容入口。Clang 只用于事实核对和离线对照，不修改或运行 clangd。
- LSP 静态解析 `compile_commands.json` 或工作区配置中的 argv，不在运行时启动 `topscc`；`topscc --dryrun` 只用于离线检查参数展开。
- LSP 协议基线采用 LSP 3.18。P1.7 只收敛已经实现的 framing、生命周期、文档同步、push diagnostics、当前导航、取消和错误，并且只为这些能力声明 capability。pull diagnostics、workspace/file operations、progress、trace 和 telemetry 等其他协议域按依赖、owner、测试和用户价值逐项评估；在实现和契约测试通过前，不得声明为 `supported`。
- LSP 契约必须同时记录消息类型、方向、参数、结果、空结果、错误、取消、版本和 capability；`initialize`、`initialized`、已实现的文档同步、位置编码、push diagnostics 和 stdio framing 是 P1 协议基线，工作区事件、文件操作、pull diagnostics 和进度按后续阶段单独验收。
- 没有经过目标条件、当前 header 和测试确认的能力，不能直接标为 `supported`。

## 来源到章节追踪

| Roadmap 章节 | 主要来源工件 | LLVM/Tops 证据入口 | 验证结果 |
| --- | --- | --- | --- |
| 语法能力地图 | `spec.md`、`research.md`、`contracts/capability-matrix.md` | `clang/lib/Headers/tops/`、`clang/include/clang/Basic/Attr.td` | 每项都写明状态、目标和验证方法 |
| Go 核心语言服务 | `data-model.md`、`contracts/lsp-boundary.md` | `clang/include/clang/Driver/Types.def`、`Options.td`、语言 tests | 先测 Go server，再用 Clang 做对照比较 |
| 编译器参数与上下文 | `data-model.md`、`contracts/compiler-driver.md` | `/opt/tops/bin/topscc`、`llvm-project/replace_topscc.sh`、Clang driver | 先验证 argv 归一化，再建立 CompilationContext |
| 目标感知语义 | `data-model.md`、`research.md`、`contracts/roadmap-gates.md` | `clang/lib/Basic/Targets/DTU.cpp`、`EFGCU.cpp`、`clang/test/DTU_test/topscc/` | 按 TargetProfile 分别验证 |
| P1.1/P1.2/P1.3/P1.4 实现边界 | `specs/003`、`specs/004`、`specs/005`、`specs/006` | 已形成的 P1.1-P1.3 代码和测试基线位于 `tops-lsp/internal/{transport,document,parser,index,server}`，不表示这些阶段的所有 gate 已完成；P1.4 的 semantic 所有权由 `specs/006` 定义 | 每个子阶段只拥有自己的输入、输出和测试 |
| VS Code 客户端工作流 | `contracts/lsp-boundary.md`、`spec.md` | 当前 `tops-lsp/.vscode/` 只作配置参考 | 每个编辑器操作都要有 LSP 路径和失败处理 |
| 细分阶段交付路线 | `plan.md`、`contracts/roadmap-gates.md` | `tops/integration_test/cases/language/STATUS.md` | 每个子阶段都写明开始、完成和降级条件 |

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

roadmap 中的“已列 grammar”和“semantic baseline”按语言版本和输入上下文分别定义，不把 parser 能识别的语法误写成完整语义诊断：

| 层次 | 计划阶段 | 完成含义 | 不包含内容 |
| --- | --- | --- | --- |
| 基础 C++ 语法 | P1.2 | 可恢复地解析选定的 C++11/14/17 声明、类型、函数、模板、表达式、语句和 Tops spelling；P1.5 只消费这些结果提供语言请求 | 每个标准版本的完整 grammar 覆盖、完整宏展开、完整类型语义 |
| 已列 C++ grammar | P1.6 | 在明确的 C++11/14/17 language standard 下覆盖规格列出的标准语法，并为每类语法提供正向、负向和不完整输入测试；新增语法在 P1.4 规则可用后通过 semantic integration gate 接入 | 链接、代码生成和运行时行为；未覆盖的完整语义另行评估 |
| 预处理器基础 | P1.2 | 解析 `#include`、宏定义和条件指令，保留原始 token，计算 `active`、`inactive`、`unknown` | 跨文件完整展开、token paste 结果、stringification 结果和宏递归等价性 |
| 配置上下文内的完整宏展开 | P2.2 | 在有效 `CompilationContext`、include roots 和宏集合内处理 include graph、对象宏、函数宏、variadic macro、递归展开、`#`、`##`、条件表达式和 source mapping | 超出配置上下文的外部文件；缺少输入时不得猜测或静默标记为完整 |
| C++20/23 语法 | P4.2 后续评估 | 以独立标准版本和测试材料（固定的测试输入和预期结果）评估 modules、concepts/requires、coroutines 等语法 | 不因 C++11/14/17 完成而自动宣称支持 |

P1.6 是首个“已列 C++ grammar”交付点，范围限定为 C++11/14/17；P2.2 是首个“配置上下文内的完整宏展开”交付点。两者都不等同于完整 C++ 语义。

P1.2 与 P1.6 的测试材料边界如下：P1.2 只负责基础结构、Tops spelling、错误恢复和宏条件的可恢复解析；P1.6 负责按 C++11、C++14、C++17 分组验收本路线图列出的声明、类型、函数、模板、表达式、语句、lambda、constexpr、attributes 和结构化绑定等 grammar。P1.2 能识别某个结构，不等于该结构已经通过 P1.6 的标准版本 gate；同一测试材料不得同时作为两个 gate 的唯一完成证据。

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

Go server 负责最终语义。P1.4 先提供通用 semantic baseline，P2 负责 target-specific 规则；Clang 只做对照比较，不直接提供 LSP 结果。

| Go 组件 | 提供的能力 | 需要记录的信息 | 验证方式 |
| --- | --- | --- | --- |
| CompilationContext loader | 读取 compile database 或工作区设置，解析 `topscc`/直接 Clang argv | `resolved`、`partial`、`missing`、`invalid` | driver 参数、优先级、多架构和缺失字段测试材料 |
| Tokenizer/parser | P1.2 解析基础标准 C++ 和 Tops 扩展；P1.6 补齐已列 C++11/14/17 grammar；声明或函数尚未写完时仍保留已识别结构 | 文档版本、ContextSeed、错误恢复范围和标准版本 | 有效/无效/不完整 Tops source 测试材料 |
| Preprocessor/macro expansion | P1.2 保存条件和宏原文；P2.2 在配置上下文内完成 include、宏展开和来源映射 | `CompilationContext`、include graph、宏定义和展开来源 | 宏展开 golden、Clang `-E -dD` 对照和缺失上下文测试 |
| AST and symbol index | 记录函数、类型、属性、内建变量、header 符号和宏来源；在缺失 header、unknown 条件或不完整输入时保留局部结果 | 定义位置、引用关系、条件区域和展开来源；`analysis-degraded` 诊断和 `stale_result` 日志由索引状态触发 | document symbols、definition、references、降级和 stale 测试 |
| Semantic layer | P1.4 检查已核验的声明、作用域、名称、基础类型、通用 host/device、地址空间和基础向量规则；P2 接入 profile-specific 规则 | `ContextSeed` 或 `CompilationContext`、`analysis-degraded`、`target-dependent` | Go semantic 单测和 Clang `-fsyntax-only` 对照比较 |
| Diagnostics | 提供标准 C++ 和 Tops 扩展的语法/语义诊断、位置、级别、code、原因和下一步；可比输入的诊断行为与 Clang 对照结果一致 | 新上下文不能被 `stale` 结果覆盖；Clang 不进入 server runtime | 标准 C++/Tops 正向、负向、不完整、缺失上下文和目标不匹配测试材料 |
| LSP request layer | P1.1/P1.7 处理协议与文档同步；P1.2 处理 parser diagnostics；P1.3 处理导航、索引降级诊断和 stale 日志；P1.4 处理 semantic diagnostics；P1.5 处理 diagnostics 聚合、completion/hover；P1.7 收敛已实现能力的协议契约 | 取消、超时、重连、错误 code 和 capability 状态 | LSP contract test 和端到端会话 |

### LSP 3.18 协议基线与能力范围

roadmap 采用 LSP 3.18 的标准消息和数据类型。下表区分当前阶段能力和未来候选能力；“基线”表示必须先完成协议行为，候选能力只有在独立实现和契约测试通过后才能进入 `supported`。

| 协议域 | roadmap 选择 | 计划阶段 | 必须验证的内容 |
| --- | --- | --- | --- |
| Base Protocol | JSON-RPC 2.0 over stdio、`Content-Length` framing、request/response/notification、请求 ID 和标准 response error | P1.1、P1.7 | 半包、多消息、非法 framing、未知方法、EOF 和错误响应不使 server 崩溃 |
| Lifecycle | `initialize` -> `InitializeResult` -> `initialized` -> 工作阶段 -> `shutdown` -> `exit`；声明 client/server capabilities，首期使用静态注册 | P1.1、P1.7 | 顺序、重复初始化、shutdown 后请求、能力协商和启动失败行为 |
| Document synchronization | `textDocument/didOpen`、增量 `textDocument/didChange`、`textDocument/didClose`；声明 `textDocumentSync`、文档版本、`DocumentSelector` 和 position encoding | P1.1 | full/incremental 选择、版本竞争、UTF-16/UTF-8 位置、CRLF、未保存文本和 URI 归一化 |
| Diagnostics push | `textDocument/publishDiagnostics` | P1.2、P1.3、P1.4、P1.5、P1.7 | notification 方向、诊断版本、空诊断清理、旧结果丢弃、`Diagnostic.code` 与 JSON-RPC error code 分离 |
| Diagnostics pull | `textDocument/diagnostic`、`workspace/diagnostic`、`workspace/diagnostic/refresh` | P4.x candidate | request/result 方向、`resultId`/`previousResultId`、取消、空结果和快照复用 |
| Core navigation | `textDocument/definition`、`textDocument/references`、`textDocument/documentSymbol` | P1.3、P1.7 | 参数、结果类型、`null`/空数组、capability、取消和过期结果 |
| Core language (completion/hover) | `textDocument/completion`、`textDocument/hover` | P1.5、P1.7 | 触发条件、结果类型、空结果、capability、取消和过期结果 |
| Cancellation | `$/cancelRequest`、标准取消错误 | P1.1、P1.7 | 取消传播、请求完成与取消竞争、旧结果丢弃和会话可用性 |
| Progress | work-done progress、partial result 和相关 progress token | P4.x candidate | 长任务进度、部分结果合并、取消和超时策略 |
| Workspace initialization | `rootUri`/`workspaceFolders` | P1.1 | 保存初始化摘要；不声明工作区事件或配置 provider |
| Workspace configuration | `workspace/configuration`、`workspace/didChangeConfiguration` | P3.2 | 初始配置读取、配置变化、来源优先级和 context 重新分析 |
| Workspace folders and file events | `workspace/didChangeWorkspaceFolders`、`workspace/didChangeWatchedFiles`、标准 file operation 方法 | P2.1、P3.4 | P2.1 负责生成 context 变化并调用 P1.3 的失效接口，P1.3 负责快照失效与重新分析，P3.4 负责客户端单根/多根和文件事件集成 |
| Window and client observability | `window/logMessage`、`window/showMessage` 和 extension output channel 的标准映射 | P3.3 | 错误摘要、日志级别、隐私字段和 server/client 展示 |
| Trace and telemetry | `$/setTrace`、`$/logTrace`、`telemetry/event` | P4.x candidate | trace 开关、telemetry 关闭策略、隐私字段和契约测试 |
| Future language capability candidates | `textDocument/signatureHelp`、`textDocument/semanticTokens/full`/`range`/`full/delta`、`workspace/symbol`、`textDocument/declaration`、`textDocument/typeDefinition`、`textDocument/implementation`、`textDocument/rename`、`textDocument/prepareRename`、`textDocument/codeAction`/`codeAction/resolve`、`textDocument/codeLens`、`codeLens/resolve`、`textDocument/formatting`、`textDocument/rangeFormatting`、`textDocument/onTypeFormatting`、`textDocument/foldingRange`、`textDocument/selectionRange`、`textDocument/linkedEditingRange`、`textDocument/inlayHint`/`inlayHint/resolve`、call hierarchy、type hierarchy、`textDocument/documentColor`/`colorPresentation`、`textDocument/documentLink`/`documentLink/resolve`、`textDocument/inlineCompletion` 和 notebook synchronization | P4.x，逐项评估 | 每项先记录 owner、前置条件、空结果、错误、取消、stale、性能和隐私测试；只有实现和契约测试通过后才能标记为 `supported` |

`textDocument/didSave`、`textDocument/willSave` 和 `textDocument/willSaveWaitUntil` 也不属于 P1.1 的基础同步承诺；需要时按独立协议能力评估。P1.7 不提前注册 pull diagnostics、progress、workspace/file operations 或 telemetry。上表是当前 roadmap 的唯一 LSP candidate registry；P4.0 和 P4.2 只按能力类别和独立任务引用它，不重复维护方法清单。

协议基线的具体规则如下：

- `textDocument/publishDiagnostics` 是 server 到 client 的 notification；pull diagnostics 使用标准 request/result 和 `resultId`。两种模式必须共享同一诊断快照和版本规则，roadmap 中的“发布”不表示 request response。旧计算结果在 server 内丢弃，不把内部 `stale` 布尔值编码成标准 `Diagnostic` 字段。
- `missing-compilation-context` 和 `invalid-compilation-context` 按请求或文档场景分别映射到产品诊断或标准 JSON-RPC/LSP response error；`analysis-degraded` 必须映射到当前文档的稳定 `Diagnostic.code`，并记录对应日志事件；`stale-result` 只表示内部过期状态，记录 `stale_result` 日志事件，并按请求类型返回 `null` 或标准取消错误，不发布 stale 诊断。
- `Position`、`Range`、`TextDocumentContentChangeEvent` 和所有导航结果使用 initialize 阶段协商的同一 position encoding；没有协商结果时遵循 LSP 3.18 的默认规则，并在测试中覆盖非 ASCII 文本。
- server 超时属于本地策略；客户端取消使用 `$/cancelRequest`，不能定义一个与 LSP response error 冲突的自定义 timeout 协议。
- workspace 配置的初始读取方式固定为 `workspace/configuration`，并兼容 `initializationOptions` 中的启动参数；后续变化统一通过 `workspace/didChangeConfiguration`，外部 compile database/header 变化通过 watched files 和 file operation 消息传递。

### 核心 LSP 流程

1. VS Code 客户端发送 `initialize`，包含 `rootUri`/`workspaceFolders`、client capabilities、document selector 相关信息和 position encoding 候选。
2. Go server 返回 `InitializeResult` 和声明的 server capabilities；客户端发送 `initialized`，之后才进入正常请求和通知阶段。
3. Go server 读取 `CompilationContext` 和 `TargetProfile`；缺少或冲突的信息生成诊断。
4. `didOpen`/`didChange` 更新文档版本；按已完成阶段运行 tokenizer/parser 和预处理器，生成可恢复的 AST、条件区域和符号索引。
5. Go semantic layer 根据 host/device、地址空间、向量和目标条件计算诊断和候选项。
6. server 通过标准 LSP 发送 `textDocument/publishDiagnostics`，并响应 completion、hover、definition、references 和 document symbols。
7. 客户端只展示结果、更新状态或转发命令，不重复判断语义。

### 失败行为

- 没有编译上下文：返回 `missing-compilation-context`，说明缺什么以及如何配置。
- 参数冲突或工具拒绝：返回 `invalid-compilation-context`，保留错误摘要和日志位置。
- 当前目标不支持语法：返回 `unsupported-target-feature`，不能静默按标准 C++ 接受。
- 头文件缺失、宏定义冲突或宏展开上下文不完整：保留原始 token 和来源，返回 `missing-compilation-context` 或 `analysis-degraded`，不能把部分展开标记为完整结果。
- 声明或函数尚未写完时（例如缺少右括号、参数或函数体）：根据已识别内容返回可恢复结果，标记 `analysis-degraded`，继续提供诊断或补全，不阻塞继续编辑。
- server 重启或上下文变化：旧结果记录为 `stale-result`，不能覆盖新诊断；过期导航结果返回 `null` 或标准取消错误。
- LSP 请求参数错误、未初始化请求、取消、内容版本冲突和 server 取消必须使用标准协议错误语义；产品 code 只放在诊断或日志中。
- 已确认的必实现能力必须在 server/client capability 和契约测试中出现；client 缺少必需能力时返回可解释的标准兼容性错误并提供升级/恢复路径。只有未列入本路线图的其他方法才可以不声明。

## 目标感知语义

所有目标相关判断都使用文件级 `CompilationContext`，不能跨文件使用隐式默认架构。

### TargetProfile 路线

| Profile 候选 | 当前状态 | 必须核对的内容 | 失败状态 |
| --- | --- | --- | --- |
| GCU300 | candidate | `__GCU_ARCH__` 分支、向量对齐、地址空间和现有 Tops tests | `target-dependent` 或 `blocked` |
| GCU400 | candidate | DTE/地址空间、向量、launch/resource 属性和 `topscc` tests | `target-dependent` 或 `blocked` |
| GCU410 | candidate | headers、target info 和 profile-specific tests | `target-dependent` 或 `blocked` |
| GCU450 | candidate | active headers、目标宏和目标测试 | `target-dependent` 或 `blocked` |
| GCU500 | candidate | active headers、target info、device mode 和 profile-specific tests | `target-dependent` 或 `blocked` |
| EFGCU500 | candidate | `__EFGCU_ARCH__`、EFGCU builtins、SIMT/device mode 和 profile-specific tests | `target-dependent` 或 `blocked` |

候选 profile 要核对完整条件分支、当前 header、实际参数和 Go 测试材料，之后才能进入 `verified`。

### CompilationContext 优先级

1. 先用有效的 `compile_commands.json` 条目，并识别 argv[0] 是 `topscc` 还是直接 `clang`/`clang++`。
2. 对 `topscc` 条目先解析 wrapper 参数，再从归一化参数得到 language、target、pass、include 和宏；直接 Clang 条目按 Clang driver 规则解析。
3. 没有条目时使用 workspace fallback，只补充缺失信息，不覆盖编译数据库中的明确目标或 driver 默认来源。
4. 信息不够、目标有多个且没有选择策略、或参数无法解释时进入 `partial`/`missing`/`invalid`，提示用户配置，不猜默认架构。

### 对照验证

- 用本地 `build/bin/clang -fsyntax-only` 核对语言模式、目标参数以及标准 C++ 和 Tops 扩展的语法/语义诊断。
- 对可比输入比较诊断触发条件、分类、source range、严重级别和消息语义；差异记录为 Go parser/semantic 修正任务，不把 Clang message 直接转发给用户。
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
| 修改 compile database 或被引用 header | 发送 `workspace/didChangeWatchedFiles` 或版本化刷新命令 | 使受影响 context/header 失效并重新分析 | 不继续显示旧定义和旧诊断 |
| 增加或移除工作区根目录 | 发送 `workspace/didChangeWorkspaceFolders` | 更新根目录和文件级 context | 只分析当前有效 workspace |
| 请求补全/悬停/导航 | 转发标准 LSP 请求 | 返回 Go 语义结果 | 显示候选、说明和位置 |
| server 错误 | 显示状态和日志位置，提供重启 | 返回稳定错误 code，不静默失败 | 用户知道如何修复或重启 |
| 关闭/重连 | 发送 shutdown/exit 或重建会话 | 清理旧文档版本和 stale 结果 | 不留下过期诊断 |

P3.2 负责配置读取和配置变化，P2.1 负责 context 选择、合并和版本输入，P1.3 负责收到 context/header 变化后的 AST/index 失效与重新分析，P3.4 负责 workspace folder、watched files 和 file operation 的客户端集成；P1.1 只保存 `initialize` 中的 workspace 摘要。

### 配置字段

- compile database 目录和文件条目选择规则。
- Go server 可执行文件或启动方式。
- `topscc` 用户编译器路径和版本；直接 `clang`/`clang++` 兼容入口及其版本。
- Clang 离线对照工具路径，仅用于离线对照，不参与 Go server 运行时语义。
- C++ 标准、Tops 输入模式、target profile/target triple。
- Tops/Clang include roots、预定义宏和必要的 device flags。
- LSP 3.18 document selector、language id、URI 归一化、document sync mode 和 position encoding 策略。
- workspace 配置的初始读取方式、配置变化通知、compile database/header watcher 或显式刷新命令。
- 日志级别和日志位置。

配置来源必须可见；编译数据库中的明确参数优先于 workspace fallback。日志不得记录密钥、完整源码或无关用户数据。公开设置、命令和扩展消息必须有版本，并提供迁移或拒绝说明。

### 阶段交付物

阶段 gate 的入口、验证、退出条件、fallback、兼容性和 owner 以 [contracts/roadmap-gates.md](../specs/001-tops-lsp-roadmap/contracts/roadmap-gates.md) 为唯一来源；本文的阶段章节和摘要表只提供导航与范围说明，不产生第二份完成条件。P1.1、P1.2、P1.3 和 P1.4 的实现边界分别由 [003](../specs/003-basic-lsp-transport/spec.md)、[004](../specs/004-tops-cpp-tokenizer-parser/spec.md)、[005](../specs/005-ast-symbol-index/spec.md) 和 [006](../specs/006-p1-semantic-baseline/spec.md) 维护；P2、P3、P4 的提示词在对应子规格建立前只是规划输入，不表示对应阶段已经进入或完成。

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
完成 Tops C++ 语言服务器的“Go 服务器语义基线”。本阶段只输出文档，不实现 Go/TypeScript 代码，也不扩展 clangd。文档必须一次性完成八个能力域的语法清单、源码和测试证据、GCU300、GCU400、GCU410、GCU450、GCU500、EFGCU500 候选 profile、CompilationContext 字段和优先级、Go tokenizer/parser/AST/symbol index/semantic/LSP 的责任边界，以及下一阶段所需的有效、无效、不完整和目标边界测试场景。GCU500 与 EFGCU500 必须分别记录 target、宏集合、builtins、device mode 和验证材料。每项都要写明目标前提、状态、限制和验证方法。
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
- **交付物**：Go 代码、索引单测、definition/references/documentSymbol 测试，以及缺失 header、unknown 条件和不完整输入的 `analysis-degraded` 诊断与 `stale_result` 日志测试。
- **依赖**：P1.2 的 AST/source range 和 ContextSeed 接口。
- **责任边界**：P1.3 负责 versioned AST/index snapshot、已知 header/provider 依赖的失效和重新排队，以及导航结果编码；不负责选择或合并 `compile_commands.json`、生成最终 `CompilationContext`，也不实现客户端 watcher。

- **内部验收切片**：P1.3-a 验收 AST/index snapshot 和局部符号；P1.3-b 验收 header/provider 依赖、context version 变化和 stale/requeue；P1.3-c 验收 definition/references/documentSymbol、`analysis-degraded` 诊断和 `stale_result` 日志。以上切片不新增 canonical roadmap phase。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“AST 和符号索引”功能。定义函数、类型、属性、内建变量、header 符号、宏条件、定义位置和引用关系的记录方式。必须包含文档变化后的索引更新、声明或函数尚未写完时的部分索引、缺少 header 和多文件导航场景，以及与 LSP definition/references/documentSymbol 的契约。
```

#### P1.4 通用 semantic baseline

- **目标**：建立可独立验收的通用 semantic baseline，覆盖已核验的声明、作用域、名称、基础类型、通用 host/device、地址空间和基础向量规则；不把完整 C++ 语义作为本阶段完成条件。
- **交付物**：`specs/006-p1-semantic-baseline/` 规格、Go semantic 规则和诊断代码、正向/负向/不完整测试、诊断 code 表和可选的 Clang `-fsyntax-only` 对照记录。
- **依赖**：P1.2、P1.3 和 ContextSeed；P2.1 提供 resolved `CompilationContext`，P2.3/P2.4 提供目标特有规则，但不扩大本阶段规则范围。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“通用 semantic baseline”功能。范围包括已解析声明、作用域、名称查找、基础标量/指针/引用/函数参数与返回值、已核验的 host/device 调用边界、地址空间和基础向量约束。建立稳定的 semantic diagnostic code、位置、严重级别、ContextSeed 降级和版本丢弃规则；Clang 只用于相同输入和 provenance 的离线对照，不进入 Go server runtime，也不直接转发 Clang message。模板实例化、完整重载决议、完整常量求值、完整访问控制、链接和代码生成保留为后续候选能力。
```

#### P1.5 基础语言功能

- **目标**：在 P1.3 已完成导航的基础上，提供诊断、completion 和 hover 的第一个可用闭环。
- **交付物**：Go LSP 代码、LSP 契约测试、端到端测试材料和用户验证文档。
- **依赖**：P1.1、P1.3、P1.4。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“基础语言功能闭环”。新增 diagnostics、completion 和 hover，并以 P1.3 已实现的 definition、references、document symbols 为前提验证端到端闭环；明确每个请求的输入、结果、空结果、错误、取消和 stale 行为。验收必须覆盖一个可编辑的 Tops kernel 文件、一个标准 C++ 语法/语义错误文件和一个 Tops 扩展语义错误文件。
```

#### P1.6 C++11/14/17 完整语法

- **目标**：在 P1.2 parser 基线之上，完成 roadmap 已列出的 C++11、C++14 和 C++17 grammar；grammar gate 不等待 P1.4/P1.5，新增语法接入已完成的 P1.4 规则作为后续 semantic integration gate，不把完整语义作为本阶段完成条件。
- **交付物**：Go parser 代码、按 language standard 分类的语法测试材料、AST/recovery 测试、Clang 对照记录和不支持语法清单。
- **依赖**：P1.2、P1.3 的 AST/source range 契约；不等待 P1.5 的 LSP 集成。
- **完成条件**：每类已列 C++11/14/17 grammar 都有正向、负向或不完整测试，AST/source range 和错误恢复稳定；P1.4 尚未完成的规则记录为待接入项，不阻塞 grammar gate。
- **semantic integration gate**：P1.4 规则可用后，补充新增 grammar 与已实现通用语义规则的正向、负向和不完整测试；该 gate 不改变 P1.6 的 parser 交付范围。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“C++11/14/17 已列 grammar”。覆盖 roadmap 明确列出的声明、类型、函数、模板、表达式、语句、lambda、constexpr、attributes、结构化绑定和 Tops source 所需标准语法。必须按 C++11、C++14、C++17 分组列出支持、不支持和不完整场景，建立正向/负向/不完整测试材料、AST/source range 检查和 Clang 离线对照；grammar gate 先完成 parser/AST 结果，新增或补全的语法在 P1.4 规则可用后再接入 semantic integration gate，未覆盖语义保持可见状态。链接、代码生成、运行时行为和 C++20/23 单独处理。
```

#### P1.7 LSP 3.18 协议基线

- **目标**：收敛 P1.1-P1.5 已实现方法的 LSP 3.18 framing、生命周期、同步、push diagnostics、当前导航、取消、错误和 capability registry。
- **交付物**：协议/transport 代码、stdio 和生命周期测试、当前方法的 capability 表、版本/position encoding/取消/stale 契约和兼容性说明。
- **范围**：只覆盖已实现的 `initialize`、文档同步、`publishDiagnostics`、P1.3 当前导航和 P1.5 completion/hover；pull diagnostics、progress、workspace/file operations、save hooks 和 telemetry 保留候选状态，不在本阶段注册。
- **依赖**：P1.1、P1.2、P1.3、P1.4、P1.5。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“LSP 3.18 协议基线”功能。范围包括 JSON-RPC 2.0 over stdio、Content-Length framing、initialize/initialized/shutdown/exit 生命周期、已实现的文档同步、push diagnostics、当前导航、completion/hover、取消、错误、position encoding 和 capability registry。每个已实现方法都定义消息类型、方向、参数、结果、空结果、错误、取消、版本和测试入口。未实现的方法不注册 capability，使用标准 `MethodNotFound` 或保留为候选状态；不使用私有 response 字段表达 semantic 状态。覆盖生命周期、stdio framing、UTF-16/非 ASCII、CRLF、版本竞争、取消、stale 和已实现方法的契约。
```

### P2：目标感知语义

#### P2.0 topscc driver 参数解析与上下文归一化

- **目标**：让 Go server 在 P1 ContextSeed 之上读取用户实际使用的 `topscc` 编译命令，并得到可审计的 `CompilerInvocation` 和 target candidates；最终文件级 `CompilationContext` 由 P2.1 形成。
- **交付物**：Go driver adapter、参数分类和归一化数据、`topscc` 参数测试材料、错误/多架构测试、配置和诊断说明。
- **范围**：识别 `topscc`、直接 `clang`/`clang++` 和未知 wrapper；解析 `-arch`、device output/link、SIMD/SIMT、`-x`、`-std`、include、macro、response file 和 wrapper 默认值；保留 raw/normalized/forwarded argv，不负责文件级 context 合并。
- **不包含**：parser grammar、Clang runtime backend、设备编译执行和 TypeScript 侧参数解析。
- **依赖**：P0、P1.1 和 ContextSeed 边界；P2.1 及后续目标语义等待 P2.0，P1.2/P1.3/P1.4 不等待完整 driver context。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“topscc driver 参数解析与编译上下文归一化”功能。支持从 compile_commands.json 的 arguments/command 和 workspace fallback 读取用户命令，区分 topscc、直接 clang/clang++ 和未知 wrapper；解析 -arch、device output/link、SIMD/SIMT、-x、-std、include、macro、response file 和 wrapper 默认值，保存 raw/normalized/forwarded 参数、来源、target candidates 和错误状态。LSP runtime 不启动 topscc，topscc --dryrun 只用于离线核对。覆盖单目标、多架构、冲突参数、未知参数、缺少 response file 和 wrapper 版本未知场景；parser 只消费归一化后的 ParseContext。
```

#### P2.1 TargetProfile 和上下文加载

- **目标**：让 Go server 根据文件选择正确的目标和编译参数，并形成文件级 resolved `CompilationContext`；将新的 context version 交给 P1.3 的 snapshot invalidation 接口。
- **交付物**：Go 代码、配置数据、上下文加载测试。
- **依赖**：P2.0、P1.1 和 ContextSeed 边界；不依赖 P1.4 semantic 规则完成。
- **责任边界**：P2.1 负责 compile database/workspace fallback 的选择、参数合并、TargetProfile 选择和 context version；P1.3 负责收到变化后的 AST/index 失效和重新分析，P3.4 负责发送客户端事件。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“TargetProfile 和上下文加载”功能。支持文件级 target profile 选择、compile_commands.json 优先级、workspace fallback、宏和 include roots 合并，并为 P1.3 产生可追踪的 context version 和失效输入。必须覆盖多目标工作区、缺少条目、冲突参数和 stale context；不重复实现 P1.3 的 AST/index snapshot invalidation。
```

#### P2.2 预处理器和宏展开

- **目标**：在有效 `CompilationContext` 下建立可重复的 include 和宏展开结果，为已列 C++ grammar、目标条件和符号索引提供原始来源映射。
- **交付物**：Go preprocessor/macro expansion 代码、include graph 和 source mapping 数据、宏展开测试材料、Clang `-E -dD` 对照工具或记录、缺失上下文诊断测试。
- **依赖**：P1.2、P2.1；P1.6 与新增 grammar 的交互场景作为后续补充测试，不作为宏展开启动的硬依赖。
- **范围**：处理配置 include roots 内的跨文件 `#include`、include guard、对象宏、函数宏、variadic macro、反斜杠续行、宏递归展开、token paste `##`、stringification `#`、条件表达式中的宏替换，以及展开 token 到原始文档的来源映射。
- **内部验收切片**：P2.2-a 验收 include graph、include guard 和 context 缺失状态；P2.2-b 验收对象宏、函数宏、variadic macro、递归展开、`#`、`##` 和条件表达式；P2.2-c 验收 source mapping、header/context 变化后的 parser/index/semantic snapshot 失效和重新分析。以上是 P2.2 内部切片，不新增 canonical roadmap phase。
- **交接规则**：宏、header 或 `CompilationContext` 变化必须使受影响的 AST/index/semantic snapshot 进入 stale，并在新版本完成后重新提交；不得由 P2.2 单独发布脱离版本的展开结果。
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

- **目标**：复用 P1.4 的通用诊断模型和 Clang 对照框架，发现并解释 profile-specific 目标不匹配、参数冲突、上下文缺失以及这些目标规则导致的 Go/Clang 结果差异；不重新实现标准 C++ 语义诊断。
- **交付物**：目标相关诊断规则和 Go 接入代码、按 `TargetProfile`/`CompilationContext` 分类的对照记录、目标错误测试、差异归因和修复任务格式。
- **依赖**：P2.1、P2.2、P2.3。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“目标诊断和 Clang 离线对照”功能。复用通用语义阶段已定义的诊断模型、稳定 diagnostic code、source range/severity/message 比较规则和 Clang -fsyntax-only 对照入口，不重复实现声明合法性、名称查找、类型检查、重载决议、模板、表达式或初始化等通用 C++ 语义。范围只包括基于已归一化 driver、workspace、预处理器和架构规则结果的 profile-specific 目标限制、目标不支持、目标参数冲突、缺失或冲突 header/macro/include context、stale context 和多目标候选解释；比较同一 CompilationContext 和 TargetProfile 下 Go 与 Clang 的触发条件、分类、source range、严重级别和消息语义，记录差异归因及 Go 修复任务。Clang 只作离线对照，不进入 Go server 运行时路径。
```

#### P2.5 多目标检查

- **目标**：确认目标切换不会让客户端和 Go server 给出矛盾结果。
- **交付物**：多 profile 测试矩阵、回归测试代码、已知限制文档。
- **依赖**：P2.1、P2.2、P2.3、P2.4。
- **specify 提示词**：

```text
实现 Tops C++ language server 的“多目标检查”功能。使用 GCU300、GCU400、GCU410、GCU450、GCU500 和 EFGCU500 候选 profile，定义同一源文件在不同 profile 下的正向、负向、边界和 stale 结果检查。每个 profile 都必须有 verified、target-dependent、blocked 或 unsupported 结论。
```

### P3：VS Code 工作流

#### P3.1 TypeScript 客户端基础

- **目标**：建立 VS Code extension、激活入口和 Go server 启动骨架。
- **交付物**：TypeScript 代码、扩展配置和启动测试。
- **依赖**：P1.1 和最小配置契约；可以与 P1.2-P2.0 并行，不能复制语义判断。
- **specify 提示词**：

```text
实现 Tops C++ VS Code 客户端的“TypeScript 客户端基础”功能。范围包括 extension 激活、Tops C++ language server 启动、标准 LSP client 建立、关闭和启动失败提示。明确不在客户端实现 Tops 语义，给出单根工作区的验收场景和日志要求。
```

#### P3.2 配置和生命周期

- **目标**：支持 compile database、Go server、topscc 用户编译器、直接 Clang 兼容入口、Clang 离线对照工具、目标 profile、include roots 和宏配置。
- **交付物**：TypeScript 代码、配置定义、配置变化测试和用户配置文档。
- **依赖**：P3.1 和最小配置契约；基础客户端配置不等待 P2.1，完整 target/profile 配置子项在 P2.1 完成后补充。
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
- **依赖**：P3.1、P3.2、P3.3；验证 completion/hover 时还需 P1.5，验证 resolved target context 时还需 P2.1。
- **specify 提示词**：

```text
实现 Tops C++ VS Code 客户端的“客户端集成验证”功能。覆盖单根和多根工作区、打开/编辑文件、补全/悬停/导航、配置修改、server 重启、断线重连和旧诊断清理。每个场景都要写清客户端动作、LSP 消息、Go server 结果和用户看到的结果。
```

### P4：TCLE 覆盖与持续演进

#### P4.0 剩余 LSP 能力扩展

- **目标**：在 P1.7 registry 稳定后，按能力切片和用户价值扩展剩余 LSP 能力，不重复实现 P1.3 已完成的三项导航。
- **交付物**：按能力组分别提供 Go server/client 实现、标准契约、测试、性能和隐私记录。
- **范围**：按 index-backed、edit-backed、presentation-backed 和 advanced 四类能力按独立任务推进；具体方法以 LSP 3.18 candidate registry 为准，本节不重复维护方法清单。
- **依赖**：对应能力的 P1.3/P1.4/P2/P3 前置条件；不要求所有候选能力同时开始或同时完成。
- **specify 提示词**：

```text
实现 Tops C++ 语言工具的“剩余 LSP 能力扩展”。按 index-backed、edit-backed、presentation-backed 和 advanced 能力组分别建立规格、owner、标准方法、context 前提、空结果、错误、取消、stale、性能和隐私测试。每项能力独立通过后才注册 capability 并标记 `supported`；缺少目标或上下文时返回标准错误、`target-dependent` 或 `analysis-degraded`，不把候选能力静默升级为支持。
```

#### P4.1 TCLE/API 索引和客户端展示

- **目标**：由 Go server 扩展高价值 TCLE、vector、DTE 和数学 API 的索引和语义；由 TypeScript LSP client 展示补全、悬停和导航结果。
- **交付物**：Go API 索引和语义代码、TypeScript VS Code/LSP client 集成代码、API 测试材料和文档。TypeScript 不实现 API 语义。
- **依赖**：P1.3、P2.2、P2.3 和 P3.4；不等待所有 P4.0 候选能力完成。
- **specify 提示词**：

```text
实现 Tops C++ 语言工具的“TCLE/API 索引和客户端展示”功能。Tops C++ language server 负责配置的 tcle.h、vector、DTE、数学和转换 headers 的符号索引、API 语义和支持状态；TypeScript 只负责 VS Code LSP client 对 completion、hover、definition 和 references 结果的展示。每个 API 必须有来源、目标范围、支持状态、限制和测试方式；没有证据的 API 不得标记为 supported。
```

#### P4.2 回归、性能和兼容性

- **目标**：让每次工具链或语言能力变化都能重复验证，并在 C++11/14/17 grammar 完成后单独评估 C++20/23 语法。
- **交付物**：Go server 回归代码、TypeScript VS Code/LSP client 回归代码、按 C++ 标准版本分类的语法测试材料、性能测试工具和固定测量条件、兼容性矩阵和日志隐私检查。
- **依赖**：P1.6、P1.7、P2.2、P2.5、P3.4、P4.1，以及 P4.0 中已经完成并登记的能力切片；不等待 P4.0 全部候选能力完成。
- **specify 提示词**：

```text
实现 Tops C++ 语言工具的“回归、性能和兼容性”功能。定义已列 C++11/14/17 grammar、C++20/23 语法评估、预处理器/宏展开、当前已实现的 Tops C++ language server 能力、候选 LSP 能力、VS Code、TCLE/API 的回归范围，补全/悬停/导航的性能指标和固定测量条件；记录 stdio framing、position encoding、push/pull diagnostics、取消/进度、workspace folders/configuration/watched files、file operation、window/trace/telemetry、日志隐私检查，以及公开能力、设置和命令的版本兼容策略。性能验证必须固定测试输入、请求类型、冷/热状态和 percentile 计算方式。C++20/23 和每项候选 LSP 能力都必须按独立标准或方法建立证据和状态，不得从 C++11/14/17 或其他 LSP 方法的结果推断支持。必须包含失败、降级、取消、stale、迁移和旧 client 能力不足场景。
```

#### P4.3 发布和持续维护

- **目标**：把已验证的能力、限制和下一阶段计划发布给项目成员。
- **交付物**：文档、发布检查表、版本/目标矩阵和最终 roadmap 更新。
- **依赖**：P4.1、P4.2。
- **specify 提示词**：

```text
实现 Tops C++ 语言工具的“发布和持续维护”功能。定义能力矩阵更新、版本/目标矩阵、发布前检查、已知限制、兼容性说明、迁移说明和 roadmap 更新流程。最终文档必须写入 doc/tops-cpp-language-server-roadmap.md，并保留 specs/ 下的审计来源。
```

## 阶段交付摘要

本表只提供阶段导航，不产生第二份完成条件。每阶段的入口证据、范围、依赖、验证、退出条件、失败处理、兼容性和 owner 以 [contracts/roadmap-gates.md](../specs/001-tops-lsp-roadmap/contracts/roadmap-gates.md) 为准；当前 P2.1-P2.5、P3.1-P3.4 和 P4.0-P4.3 以阶段组导航，组内子阶段只有在建立独立规格、任务和 gate 后才能单独报告完成。

| 阶段 | 目标 | 主要交付 |
| --- | --- | --- |
| P0 Go 语义基线 | 明确能力、事实和责任边界 | 能力矩阵、ContextSeed/CompilationContext、P1 验收材料和候选 capability registry |
| P1.1 基础服务和 LSP 传输 | server 能建立会话并保存文档输入 | stdio、生命周期、文档同步、取消、错误和日志 |
| P1.2 tokenizer/parser | server 能提供可恢复语法结果 | 基础 C++/Tops grammar、宏条件结构、错误恢复和 parser diagnostics |
| P1.3 AST/index/navigation | server 能提供新鲜快照和基础导航，并在结果受限时说明原因 | AST 元数据、按需 header、条件状态、definition/references/documentSymbol、`analysis-degraded` 诊断和 `stale_result` 日志 |
| P1.4 semantic baseline | server 能提供范围明确的通用语义诊断 | 声明、作用域、名称、基础类型、通用 Tops 规则和稳定 diagnostics |
| P1.5 核心语言请求 | 用户获得当前已实现能力的 diagnostics、completion、hover 和导航闭环 | 消费 P1.3/P1.4 结果的标准请求 |
| P1.6 C++11/14/17 grammar | parser 按标准版本补齐已列语法 | grammar、AST/recovery 和标准版本测试 |
| P1.7 LSP 协议收敛 | server/client 的基础协议和已实现 capability 稳定 | framing、生命周期、同步、diagnostics、当前导航、取消、错误和 registry |
| P2.0-P2.5 目标感知语义 | driver、context、profile 和目标规则可复现；各子阶段按独立规格逐项验收 | CompilerInvocation、文件级 context、宏展开、架构属性、目标诊断和多目标检查 |
| P3.1-P3.4 VS Code 工作流 | client 能配置、观察和恢复当前 server；各子阶段按客户端集成结果验收 | 激活、配置、状态、日志、重启和当前方法集成 |
| P4.0-P4.3 剩余能力与持续演进 | 按用户价值独立扩展 LSP、TCLE/API 和发布治理 | 单项 LSP/API 能力、回归、性能、兼容性和发布材料 |

阶段顺序和并行关系以 [contracts/roadmap-gates.md](../specs/001-tops-lsp-roadmap/contracts/roadmap-gates.md) 为准：P0 → P1.1 → P1.2 → P1.3 → P1.4 → P1.5；P1.6 的 grammar gate 在 P1.2/P1.3 的 parser、AST 和 source range 契约稳定后即可开始，可与 P1.4/P1.5 并行；P1.4 完成后再运行 P1.6 的 semantic integration gate；P1.7 只收敛 P1.1-P1.5 已实现方法。P2.0 在 P1.1 和 ContextSeed 边界稳定后即可开始，P2.1 形成文件级 resolved context，P2.2 依赖 P2.1，P2.3-P2.5 再按目标规则推进。P3.1 可与 P1.2-P2.0 并行，P3.2 的完整 target/profile 配置等待 P2.1，P3.4 负责 workspace folder、watched files 和 file operation 的客户端集成，并按 P1.5/P2.1 的实际验证范围运行。P4.0 按能力切片推进，P4.2 消费已完成切片，不要求整批能力串行完成。

### 阶段到实现的交接

阶段交接只引用 [contracts/roadmap-gates.md](../specs/001-tops-lsp-roadmap/contracts/roadmap-gates.md)、各阶段 `spec.md/plan.md/tasks.md` 和对应测试材料。本文件的阶段提示词只用于导航，门禁字段不在此处另行定义；新增能力必须先更新 canonical gate，再更新本节引用的子规格。当前 P2.1-P2.5、P3.1-P3.4 和 P4.0-P4.3 以阶段组进行路线图导航；组内子阶段只有在建立独立规格、任务和 gate 后，才能作为独立完成项报告。

### 验证方式

| 验证层 | 覆盖内容 | 结果要求 |
| --- | --- | --- |
| 事实核对 | headers、attributes、driver、TargetInfo、测试状态 | 每条能力有路径、anchor、范围和日期 |
| Go parser/semantic | 有效、无效、不完整、P1.4 通用 semantic baseline、P1.6 已列 C++11/14/17 grammar、目标边界和缺失上下文 | Go 结果可重复；可比诊断与 Clang 离线对照一致，差异有记录；未实现完整语义保持可见状态 |
| Preprocessor/macro | include graph、宏展开、条件分支、source mapping 和缺失上下文 | 在有效 CompilationContext 内结果可重复；部分输入明确降级 |
| LSP contract | Base Protocol、stdio framing、lifecycle、`initialized`、已实现 capability、DocumentSelector、document sync、position encoding、diagnostics、当前 completion/hover/navigation、取消和 stale | 请求/通知方向、参数、响应、错误、空结果、版本、取消和 stale 行为一致；候选能力不出现在 server capability |
| Future LSP candidates | pull diagnostics、progress、workspace/file operations、trace 和 telemetry | P4.x 或对应客户端阶段单独验收 | 每个方法先定义 owner、前置条件、空结果、错误、取消、stale、性能和隐私；通过后才加入 server capability |
| VS Code integration | 初始化、`initialized`、配置、重启、重连、单根/多根、workspace folder 变化、compile database/header 变化 | 用户看到的结果与 LSP 路径一致；外部上下文变化不会遗留旧诊断和旧导航 |
| 性能/观测 | 补全、悬停、导航延迟，错误日志和隐私 | 95% 常规结果小于 1 秒；失败可定位且不泄露源码 |

### 目标对应内容

| 规格标准 | 文档中的对应内容 |
| --- | --- |
| SC-001 | 八个能力域表和状态规则 |
| SC-002 | Go 核心语言服务表、失败行为和 P1 测试材料要求 |
| SC-003 | 目标感知语义的 Clang 离线对照和 Go 测试材料检查 |
| SC-004 | 验证方式中的性能/观测层和 P4 回归治理 |
| SC-005 | VS Code 配置字段、缺失上下文诊断和 P3 退出条件 |
| SC-006 | LSP 3.18 流程、方法/capability 表、契约来源和各阶段行为测试要求 |
| SC-007 | TargetProfile、CompilationContext 和目标切换规则 |
| SC-008 | VS Code 配置契约、版本化消息和发布前检查 |
| SC-009 | 固定发布路径和发布前检查 |

## 发布前检查

最终文档固定为 `doc/tops-cpp-language-server-roadmap.md`。只有满足以下条件，文档才能从 Draft 进入 Published：

1. `specs/001-tops-lsp-roadmap/`、`specs/003-basic-lsp-transport/`、`specs/004-tops-cpp-tokenizer-parser/`、`specs/005-ast-symbol-index/` 和 `specs/006-p1-semantic-baseline/` 的 spec、plan、research、data-model、contracts、tasks 和 quickstart 通过文档结构、链接、范围和一致性检查。这里的“通过检查”不等同于对应 Go/TypeScript 实现任务全部完成。
2. 能力地图、P1.2 基础 grammar、P1.4 semantic baseline、P1.6 已列 C++11/14/17 grammar 的边界、预处理器/宏展开阶段、Go 架构、目标 profile、LSP 3.18 基线、候选 capability 和细分阶段表没有矛盾。
3. 所有事实引用都能回到当前 LLVM/Tops checkout；未验证项保留状态和下一步动作；C++20/23 和未完成的宏上下文不能标记为 `supported`。
4. 完成 quickstart 中的文档、JSON、源码路径、Clang 对照、LSP framing/生命周期/同步/诊断/工作区契约和最终文件检查。
5. 通过 `git diff --check`，没有模板占位符、未解释的 `NEEDS CLARIFICATION`、把 LSP 子集写成全量支持的表述或 unsupported claim。

发布状态由本文件的发布前检查和验证记录决定，不由任一来源工件的任务勾选单独决定。若阶段实现尚未完成，路线图仍可保持 `Draft`，但必须保留已验证能力、候选能力和未完成范围的明确状态。

## 质量与发布

文档当前保持 Draft。它不替代 `specs/` 下的审计来源，也不把基础语法、Clang 对照结果或 parser 阶段完成单独说成完整 C++ grammar、完整宏展开、完整语义诊断或完整 LSP 支持。P1.4 以通用 semantic baseline 的实现、测试和阶段门禁为完成依据；P1.7 只确认已实现的协议能力；候选 LSP/API 能力按独立工件和测试结果逐项更新状态。
