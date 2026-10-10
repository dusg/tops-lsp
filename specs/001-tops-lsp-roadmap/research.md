# 研究记录：Tops C++ Language Server Roadmap

**日期**：2026-10-07
**范围**：为 roadmap 文档确定事实来源、语义接入路线、编译上下文和验证方式；不实现 Go 服务器、TypeScript 客户端或独立 parser。

## Decision 1：以 Clang/Tops 工具链作为语义事实来源

**Decision**：能力矩阵以当前 `llvm-project` checkout 中的 Clang 前端、Tops headers、TargetInfo/attribute 定义和已有测试为一手依据；用户编译命令的入口和默认值以已安装 `topscc` wrapper 为准；旧汇总文档只能作为索引。

**Rationale**：当前源码直接定义了 Tops 语言模式、`.tops` 输入类型、`-Tops` driver 选项、执行/存储属性、内建变量和目标条件。`tops/integration_test/cases/language` 与 `clang/test/DTU_test/topscc` 提供了正向、负向和目标相关测试入口。语义事实必须跟随活动 header、编译器和测试，避免把历史文档或其他架构的数值当成当前结论。

**Evidence**：

- `clang/include/clang/Driver/Types.def`：`.tops` 对应 `TOPS` 输入类型。
- `clang/include/clang/Driver/Options.td`：注册 `-Tops`。
- `clang/include/clang/Basic/LangStandards.def`：定义 `tops` language standard。
- `clang/lib/Headers/tops/__tops_defines.h`：Tops 属性和宏入口。
- `clang/lib/Headers/tops/__tops_builtins.h`：内建变量入口。
- `clang/include/clang/Basic/Attr.td`：Clang Tops/目标属性声明。
- `tops/integration_test/cases/language/STATUS.md`：现有 C++11/14/17 language case 状态和 device coverage exception。

**Alternatives considered**：

- 只以 `clang/docs/Tops_C_Language_Extenstion.md` 为来源：拒绝，因为该文档已明确标注后续不再维护。
- 只扫描用户样例：拒绝，因为样例不足以覆盖属性约束、目标条件和失败路径。

## Decision 2：Go server 从零实现，不扩展或依赖 clangd

**Decision**：roadmap 确定使用 Go 从零开发语言服务器。Go server 自行维护 Tops C++ 的 tokenizer/parser、AST、符号索引、目标语义、诊断和 LSP 行为；不在 clangd 上添加 Tops 支持，也不把 clangd 作为运行时语义后端。`topscc` 是用户侧编译器入口，直接 Clang 是兼容入口；两者的输出只用于编译上下文核对和离线对照。

**Rationale**：这是用户已经确定的架构选择，并符合项目章程中“Go 服务器负责语言分析、工作区状态和 LSP 协议行为”的边界。当前 `tops-lsp` 没有现成 parser、Go module 或语言服务实现，因此 roadmap 必须把 Go 侧语义模型和增量分析边界作为 P0/P1 的核心交付，而不是把实现责任转移到 clangd。

**Alternatives considered**：

- 扩展当前 clangd：拒绝，违反已确定的架构选择；clangd 仅保留为可选的外部差分参考，不进入服务器运行链路。
- Go LSP + C++/clangd sidecar：拒绝，服务器语义必须由 Go 自有实现负责，Clang 只用于核对结果。
- 独立 ANTLR parser：不作为已选依赖；是否采用生成式 grammar 只能在 Go parser 设计任务中评估，不能改变 Go server 的语义所有权。

## Decision 2A：静态解析 topscc argv，不在 server runtime 启动编译器

**Decision**：LSP 从 `compile_commands.json` 的 `arguments`/`command` 或显式 workspace settings 读取 argv。识别 argv[0] 为 `topscc` 时，先解析 wrapper 参数，再解析底层 Clang 参数；识别为直接 `clang`/`clang++` 时只使用 Clang 参数规则。server runtime 不启动 `topscc`；`topscc --dryrun` 只作为开发机离线核对命令。

**Verified evidence**：当前环境中的 `/usr/bin/topscc` 解析到 `/opt/tops/bin/topscc`，版本为 `v4.0.20260828`，版本命令同时输出 Enflame compiler `5.7.8`。`topscc --dryrun -arch gcu400 -x tops -fsyntax-only /dev/null` 输出了 `-std=c++11`、`-x tops`、`-Tops`、`--include tops.h`、Tops include 根、`--cuda-gpu-arch=gcu400`、host/device 两个 cc1 命令和 `__GCU_ARCH__=400`。

**Wrapper 参数范围**：已安装 wrapper 直接处理 `-arch`、`-dbin`、`-dbc`、`-dllvm`、`-dlink-path`、`-dlink`、`-rdc`、`-host-only`、`-notopslib`、`-topsrt`、`-ltops`、`-kdd`、`--simd`、`--simt`、`--simt32`、`--simt128` 和 `--dryrun`；`-Tops`、`-x`、`-std`、`-I`、`-D` 等参数继续进入底层 Clang 参数。参数完整集合受 wrapper 版本影响，不能写成跨版本保证。

**Rationale**：静态解析可保持 LSP 启动和文档分析的可预测性；离线 dry-run 仍能核对实际 wrapper 展开，而不把外部进程、超时和编译器副作用引入 server runtime。

## Decision 3：编译数据库优先，工作区设置作为显式 fallback

**Decision**：roadmap 要求优先从 `compile_commands.json` 获取每个文件的语言模式、目标、标准、include 路径和预定义宏；缺少条目时再使用工作区设置，并明确受限模式、优先级和缺失字段提示。

**Rationale**：Tops 语义依赖编译参数和目标宏。当前 `tops-lsp/.vscode/settings.json` 只有一个指向 workspace 外部 checkout 的 `tlsp.path`，没有已实现的 Tops compilation database 配置。把上下文定义成文件级而非全局默认值，才能支持同一工作区的多目标和多 C++ 标准。

**Validation required in P0**：

1. 检查 `.tops` 是否保留 native Tops 输入类型。
2. 检查 `.cpp + -Tops` 和 `-x tops` 是否得到相同语言模式。
3. 检查编译数据库是否保留 `-Tops`、`-x tops`、`--target`、device flags、include 路径和宏。
4. 检查 Go server 在缺少数据库时的 compiler-context fallback 是否使用正确的本地工具，而不是当前 settings 中未经确认的外部路径；clangd 不作为 fallback。

## Decision 4：roadmap 交付采用能力矩阵、LSP 契约和阶段门禁三类文档

**Decision**：除 `plan.md` 外，Phase 1 生成：

- `data-model.md`：能力条目、目标 profile、编译上下文、诊断、LSP 契约和里程碑之间的关系。
- `contracts/capability-matrix.md`：每个语法能力的记录格式和状态规则。
- `contracts/lsp-boundary.md`：编辑器动作、LSP 消息、服务器结果、客户端展示、错误与兼容性规则。
- `contracts/roadmap-gates.md`：P0-P4 的入口条件、验证证据、退出条件和降级状态。
- `quickstart.md`：维护者如何核对文档、源码证据、Go server 骨架和最小 Clang 差分验证。

**Rationale**：项目章程要求每份计划明确职责、契约、目标假设、测试、错误行为、可观测性和兼容性。把这些信息拆成稳定的文档契约，能避免 roadmap 只有叙述而无法验收。

**Alternatives considered**：

- 只在 `plan.md` 中写长篇设计：拒绝，因为能力矩阵和 LSP 边界会随阶段演进，需要可单独评审和引用。
- 立即创建 Go/TypeScript 源码目录：拒绝，本 feature 的交付物是 roadmap 文档；Go server 的源码布局和 parser 依赖应在后续任务中按已确定架构落地。

## Decision 5：验证以本地工具链和文档静态校验为主

**Decision**：本 feature 的验证不构建或运行新的语言服务器；使用仓库现有的本地 LLVM `clang` 作为差分验证工具，并执行可重复的文档检查。具体包括 `build/bin/clang`、`build/bin/llvm-lit` 的可用性检查，Tops driver/headers/test 路径核对，Go server 未来所需的源码入口检查，Markdown/JSON 错误检查和 `git diff --check`。`build/bin/clangd` 不属于 Go server 的运行依赖。

**Rationale**：当前 `tops-lsp` 没有 Go module、npm package、TypeScript 配置、CMake 或任务入口，构建服务器没有可执行基线。直接运行不存在的构建命令会把文档任务和未来实现混在一起。

**Constraints**：

- 只在开发机检查源码、构建配置和文档；本 feature 不涉及测试机 workload。
- 任何架构数值或支持结论都必须回到完整条件分支、活动 header 和实际目标参数核对。
- 未验证的 Go parser/semantic 行为写成“待 P0/P1 验证”，不能写成已支持；Clang 差分结果不能替代 Go server 自身测试。
