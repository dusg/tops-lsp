# 研究记录：Tops C++ Tokenizer 与 Parser

**Feature**：`004-tops-cpp-tokenizer-parser`

**研究日期**：2026-10-08

## 研究范围

本阶段只确定 Go parser 的实现边界、数据流、LSP 诊断发布方式、测试材料组织、topscc context provenance 和 Clang 离线对照入口。研究不修改 `/home/carl.du/work/llvm-project` 或 `/home/carl.du/work/topsop`，不运行 GCU workload，也不把 topscc/Clang/clangd 放入 server 运行路径。

## 已核对事实

- 当前 `tops-lsp` 使用 `go1.26.8 linux/amd64`，`go test ./...` 已通过。
- 当前 server 在 `internal/server/server.go` 中处理生命周期和文档通知；`handleDidOpen`、`handleDidChange` 只更新 `document.Store` 并记录日志，尚无 parser 或 `publishDiagnostics`。
- 当前 `Server.Handle` 被 server 单测直接调用；`Server.Run` 持有 `transport.Writer`，请求响应由 writer 写出，通知处理目前没有输出参数。
- 当前 `internal/protocol/messages.go` 已有 LSP `Range`、文档同步参数和 `ServerCapabilities`，但没有 `Diagnostic` 或 `PublishDiagnosticsParams`。
- 当前 `internal/document/store.go` 的 `Get` 返回 `DocumentState` 值副本；现有文档测试已经覆盖 UTF-16、CRLF、多变更和版本失败路径。位置转换函数目前是 `document` 包私有实现。
- 本地 `/home/carl.du/work/llvm-project/build/bin/clang` 输出版本 `18.1.8`，对应 `llvm-lit` 输出 `18.1.8dev`。
- 当前环境的 `/usr/bin/topscc` 解析到 `/opt/tops/bin/topscc`；`topscc --version` 输出 `v4.0.20260828` 和 Enflame compiler `5.7.8`。
- `topscc --dryrun -arch gcu400 -x tops -fsyntax-only /dev/null` 已观察到 wrapper 默认参数、Tops include 根、host/device 命令、`--cuda-gpu-arch=gcu400` 和 `__GCU_ARCH__=400`。
- `clang/test/DTU_test/tcle/vector_op_parser.cc` 使用 GCU400/GCU450 的 `clang_cc1 -triple gcu -target-cpu ... -fsyntax-only` 和 `FileCheck` 负向诊断。
- `clang/test/CodeGenEFGCU/launch-bounds-e2e.cc` 使用 `-x tops -triple efgcu-enflame-tops -target-cpu efgcu500 -fcuda-is-device`，检查 `__launch_bounds__`、`__maxnreg__` 的 IR/ASM 结果；该测试还展示了宏定义、kernel 属性和 builtin 变量的组合。
- `clang/test/DTU_test/topscc/attribute/launch_bounds.tops` 使用 GCU400/GCU450 的 C++17 device 编译参数检查属性输出。
- `/home/carl.du/work/topsop/topsop/lib/kernel/cc_kernel/range/range_kernel.tops` 包含 include、模板、namespace、`#if`、`__global__`、`__local__`、`__valigned__`、DTE 类型和模板实例化；对应 `range_host.tops` 包含 `__host__`、switch、cast 和 `<<<...>>>` kernel launch。
- `/home/carl.du/work/topsop/topsop/lib/kernel/cc_kernel/mhc_pre/mhc_pre_n512_kernel_gcu400.tops` 包含 `__vector`、`__vector2`、`__bf16`、`__device__`、`__forceinline__`、`__attribute__((noinline))`、模板和 `__restrict__`。

## Decision 1：使用 Go 自有 parser，不引入运行时 Clang

**决定**：在 `internal/parser/` 中实现 Go tokenizer、递归下降声明/语句 parser、Pratt 表达式 parser、Tops 扩展解析和错误恢复。parser 只接收文本和显式 `ParseContext`，不启动 Clang、clangd 或设备工具。

**理由**：

- 规格明确要求语义所有权属于 Go server。
- Go 标准库足以提供字符串扫描、JSON、测试和并发工具，不需要把第三方 parser runtime 变成 server 的新依赖。
- 递归下降适合按 C++ 声明、预处理分支和 Tops qualifier 设置同步点；Pratt parser 适合表达式优先级和未完成表达式恢复。
- 自有 token 和 source range 可以直接保留 Tops spelling、宏分支和原文位置，避免把 Clang AST/诊断格式变成公开契约。

**替代方案**：

- Tree-sitter C++ grammar：提供增量解析能力，但 Tops `<<<...>>>`、属性和目标条件仍需要外部扩展；本 feature 还需要稳定的自有诊断和宏三态，增加 grammar/runtime 依赖不能直接解决所有边界。
- ANTLR 或其他生成式 grammar：可生成完整 grammar，但需要额外 runtime、生成流程和错误恢复适配；当前 Go 工程没有该依赖，且 Tops 目标扩展仍需自定义。
- 运行时调用 Clang/clangd：可以复用部分语法，但违反 Go server 语义所有权、离线单测和无 clangd 运行约束，因此只保留为离线 对照验证。

## Decision 2：parser 与文档状态分层

**决定**：`internal/document` 继续只保存 URI、language ID、文本和版本；`internal/parser` 不读取 `document.Store`，而由 server 在文档成功更新后取得值快照并构造 `ParseContext`。parser 返回 `ParseResult`，server 负责将结果转换成 LSP diagnostics。

**理由**：当前 `document.Store.Get` 已返回值副本，足以形成不可变输入；不把 C++ grammar、宏和诊断写入 document store，可以保持 `003-basic-lsp-transport` 的边界。

**实现约束**：

- 解析输入包含 document URI、文本版本、language standard、宏集合、target/pass 不透明标识和 context version。
- parser 不从文件扩展名推断 target，不把 `__GCU_ARCH__` 或 `__EFGCU_ARCH__` 的值写死在 grammar 中。
- server 在发布前检查文档仍存在且版本/context version 与 parse job 相同；旧结果丢弃。
- 第一版同步解析，避免在现有 stdin 消费循环中引入额外结果排序；保留 version guard，为后续异步解析留下边界。

## Decision 3：宏条件使用三态求值

**决定**：条件区域状态使用 `active`、`inactive`、`unknown`。只对显式传入的宏值和安全的整数/布尔条件求值；无法安全确定的条件保留两侧结构并产生一个 `tops-syntax-unknown-condition` 信息级诊断。

**理由**：真实 `cc_kernel` 使用目标宏分支，而 parser 不拥有完整编译器预处理器。三态可以避免无 context 时把某个目标分支误当作活动代码，同时保留后续语法分析所需结构。

**范围**：支持 `defined`、整数常量、标识符、`!`、`&&`、`||`、`==`、`!=` 和括号的条件结构；宏 replacement token 只保留原文，不实现跨文件、递归、token paste 和 stringification 的完整展开。

## Decision 4：集中管理 source range 与 LSP UTF-16 转换

**决定**：新增内部位置映射模块，统一维护 byte offset、行边界、UTF-16 character 和 CRLF 规则；`document.Store` 的现有位置转换逻辑迁移或复用该模块，parser diagnostics 通过同一实现编码为 LSP range。

**理由**：规格要求 parser 诊断和文档增量使用相同的 UTF-16/CRLF 规则。复制两套转换代码会让 parser range 与已有 document range 在补充平面字符或 CRLF 下产生漂移。

**验证**：保留现有 `internal/document` 的 UTF-16/CRLF 测试，并新增 parser token、diagnostic、EOF 零宽范围的交叉测试。

## Decision 5：通过 publisher 边界发布 diagnostics

**决定**：在 `internal/protocol` 增加 `Diagnostic`、`DiagnosticRelatedInformation` 和 `PublishDiagnosticsParams`；在 server 内定义可写的诊断发布边界。`Run` 把现有 `transport.Writer` 作为 publisher 传给通知处理；`Server.Handle` 保留现有调用方式，使用空 publisher 供同步单测使用。

**理由**：当前 `Run` 已拥有唯一的 stdio writer，`transport.Writer` 自带互斥写保护，可以串行化 request response 和 diagnostics notification。把 writer 直接放入 `document.Store` 会越过分层；把诊断写到 stdout 的其他路径会破坏协议纯净性。

**契约行为**：

- 成功 `didOpen` 和 `didChange` 解析后发布一条 `textDocument/publishDiagnostics`。
- 诊断携带 URI、当前 document version、稳定 code、severity、LSP range、source=`tops-lsp` 和可选 related information。
- `Server.Handle` 的旧单测仍验证 notification 没有 response；需要验证诊断发布的测试使用可注入 publisher 或进程级 LSP 测试。
- `didClose` 不再发布旧版本诊断；关闭前的结果若已被客户端收到，由客户端按文档生命周期清理。
- 发布失败记录结构化日志并报告 transport fatal，不把完整源文本写入日志。

## Decision 6：测试材料分层，真实 corpus 只读引用

**决定**：新增 `testdata/parser/`，按 `positive`、`negative`、`incomplete`、`macros`、`topsop` 和 `expected` 分类；使用 manifest 记录标准、宏、target/pass、不完整状态、预期节点/诊断和 Clang 对照验证。

**理由**：parser 单测需要无外部工具即可运行，Clang 对照和真实 corpus 需要额外 context。缩减测试材料能稳定覆盖错误恢复，manifest 引用真实 `cc_kernel` 能保留工程语法代表性而不复制或修改 topsop 源码。

**验证分层**：

1. Go unit tests：token、AST、macro state、recovery、range，不能依赖 Clang。
2. Go server integration tests：didOpen/didChange/close、publishDiagnostics、旧版本丢弃和 stdout 帧。
3. Optional 对照验证 checks：同一测试材料运行 local Clang 的 `-###`、`-E -dD`、`-dM -E`、`-fsyntax-only` 和适用的 AST dump/FileCheck。
4. Corpus checks：优先 parser-only；缺 header/target 时记录 partial/context-invalid，不以设备 workload 作为 parser 通过条件。

## Decision 7：Clang 只作为离线对照

**决定**：对照记录保存源文件、`topscc`/Clang driver provenance、归一化参数、工具版本、宏/target/pass、Clang 分类/位置、Go 分类/位置和差异原因。比较接受/拒绝、主要 AST 结构和诊断类别/range，不比较完整 message 文案。

**理由**：本地 Clang/llvm-lit 已验证可用，现有 Clang tests 提供 GCU400/GCU450 vector 和 EFGCU500 launch attribute 的命令样例；但不同目标、header 和 driver 形式会改变结果，必须保存完整 context，不能把某个命令输出写成跨目标事实。

## Decision 8：没有 CompilationContext 时使用显式 partial context

**决定**：当前 server 没有 `compile_commands.json` 或 workspace context resolver，因此 `didOpen`/`didChange` 的第一版 parser 接入构造 `partial` `ParseContext`：language standard 为 `unknown`，driver kind 和 argument provenance 为 unknown，预定义宏为空，target/pass 为 unknown，context version 使用 server 的当前值。parser 仍解析标准基础语法和 Tops spelling，但不根据缺失 context 选择 C++ 标准或 GCU target。

**理由**：使用隐含 C++17、GCU300 或其他默认值会把编辑器结果伪装成完整语义，并违反规格中“不猜测 profile”的约束。显式 partial 状态可以让用户看到可用的语法结果，同时把标准/目标相关判断留给后续 `CompilationContext` 功能。

**影响**：直接 parser 测试材料必须显式提供 C++ standard、宏和 target/pass；server integration 测试材料需要覆盖 partial context。后续 context resolver 接入时只替换 context 构造，不改变 parser 输入/输出契约。

## 未决项处理

本阶段没有遗留 `NEEDS CLARIFICATION`。具体 Go type 名称、测试材料的 golden JSON 细节和 Clang 对照脚本是否用 Go 或 shell 实现，在 tasks 阶段按本研究的边界确定，不影响架构、协议和验收路径。
