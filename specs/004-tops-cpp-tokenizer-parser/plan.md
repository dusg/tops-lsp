# 实施计划：Tops C++ Tokenizer 与 Parser

**分支**：`004-tops-cpp-tokenizer-parser` | **日期**：2026-10-08 | **规格说明**：[spec.md](spec.md)

**输入**：来自 `specs/004-tops-cpp-tokenizer-parser/spec.md` 的功能规格说明。

## 摘要

本功能在现有 Go LSP server 上增加自有 tokenizer/parser，使 server 能解析 C++11/14/17 基础语法、Tops qualifier/attribute、kernel launch、宏条件和未完成输入，并产生可恢复语法树与稳定 parser diagnostics。用户命令由上游 `topscc`/Clang context resolver 归一化；parser 不解析 raw compiler argv，不调用 Clang/clangd，不把 grammar 或诊断所有权交给 TypeScript client，也不修改 `llvm-project` 或 `topsop`。

研究后的主要决定：

- `internal/parser/` 使用 Go 标准库实现 tokenizer、递归下降声明/语句 parser、Pratt 表达式 parser、宏条件三态和错误恢复。
- `internal/document` 继续只保存文本/版本；新增共享位置映射模块，统一 document 增量和 parser diagnostics 的 UTF-16/CRLF 规则。
- 当前 server 没有 `CompilationContext` resolver。`didOpen`/`didChange` 的第一版使用显式 `partial` context，不默认选择 C++ 标准、GCU target 或宏值。
- `Server.Run` 将现有 `transport.Writer` 作为 diagnostics publisher；`Server.Handle` 保留无 publisher 的同步测试行为。
- parser 测试材料不依赖外部工具；`topscc` 参数 provenance 和 Clang/llvm-lit 离线对照只作为可选验证，真实 `cc_kernel` 目录只读引用或使用缩减片段。

详细研究、数据模型、契约和验证入口分别见 [research.md](research.md)、[data-model.md](data-model.md)、[contracts/](contracts/)、[quickstart.md](quickstart.md)。

## 技术上下文

**语言/版本**：Go `1.26.8`，当前 `go.mod` 已验证为 `module tops-lsp` 和 `go 1.26.8`；基线命令 `go version` 与 `go test ./...` 已通过。

**主要依赖**：Go 标准库，包括现有 `encoding/json`、`context`、`io`、`sync`、`log/slog` 和测试库；本功能不新增第三方 parser、LSP、日志或 Clang runtime 依赖。`ParseContext` 接收 `topscc`/直接 Clang 的归一化 provenance，不接收 raw argv。

**存储**：进程内 `internal/document.Store` 文档状态；parser 输入使用 `DocumentState` 值快照；测试材料、manifest 和 golden 摘要存放在 `testdata/parser/`。

**测试**：`go test ./...`、`go test -race ./...`、`go vet ./...`；parser 单元测试、server/LSP 集成测试和可选 Clang 对照验证 测试。当前基线 `go test ./...` 已通过。

**目标平台**：Linux amd64 Go server；标准输入输出使用现有 LSP over stdio transport。parser 单测不需要 GCU 设备、测试机 Docker、clangd 或网络。

**项目类型**：Go command-line language server 与内部 parser library。

**性能目标**：本功能不新增公开延迟门槛；第一版同步解析，要求测试材料和文档通知不会无限等待、parser 不 panic、100 个连续逐字版本不会发布旧结果。后续若引入异步解析，必须继续满足 generation/version 丢弃规则。

**约束**：

- stdout 只能输出完整 LSP frame；diagnostics 通过现有 writer 发布，日志继续写 stderr/受控日志目标。
- parser 不启动 Clang/clangd，不访问设备，不修改外部 checkout。
- parser 不解析 `topscc` wrapper 参数；driver 默认值、response file 和多目标展开由上游 `CompilationContext` 处理。
- 没有 `CompilationContext` 时只能使用 `partial` context；不隐含 C++17、GCU300 或其他 target。
- 诊断 range 使用原始 source 和 LSP UTF-16；已有 document UTF-16/CRLF 行为不能回归。
- 现有 `Server.Handle` 调用者和基础生命周期/LSP 错误契约保持兼容。

**规模/范围**：1 个内部 parser package、1 个共享位置映射 package、protocol diagnostics model、server diagnostics publisher、四类以上测试材料集合、至少 20 个 range case、至少 10 个真实 `cc_kernel` 片段/manifest 条目和标准/失败/不完整/目标边界测试。

## 宪法检查

*门禁：已在阶段 0 研究前检查，并将在阶段 1 设计完成后再次检查。*

- [x] **LSP 契约优先**：`contracts/lsp-diagnostics.md` 固定 `publishDiagnostics` payload、发布时机、版本保护、错误行为和兼容性；parser 不直接写 transport。
- [x] **Go/TypeScript 分层边界**：Go server 负责 parser、诊断和 LSP；TypeScript client 只负责生命周期和展示；本功能不创建客户端语义代码。
- [x] **Tops C/C++ 语义保真**：标准 C++、Tops spelling、宏状态和 target/pass context 作为独立输入记录；不从相邻架构推断宏值；真实 corpus 和 Clang tests 只读核对。
- [x] **行为优先的验证**：parser、位置、恢复、宏、LSP diagnostics、stale result、stdout 和 Clang 对照均有对应测试入口，覆盖正常、失败和边界路径。
- [x] **可观测且兼容地演进**：诊断 code/range/version 稳定；发布失败有结构化日志；日志不写完整源文本；现有生命周期和未知语义 request 行为保持兼容。
- [x] **没有未解释的违规项**：不新增第三方依赖、不引入运行时 Clang、不改变外部 checkout；复杂度记录为空。

## 研究与设计产物

阶段 0/1 已生成：

- [research.md](research.md)：实现路线、parser/server 边界、partial context、宏三态、位置映射、publisher、测试材料、topscc context provenance 和 Clang 离线对照决策。
- [data-model.md](data-model.md)：`ParseContext`、Token、SyntaxNode、ConditionalRegion、ParserDiagnostic、ParseResult 和版本状态。
- [contracts/parser.md](contracts/parser.md)：parser 输入/输出、grammar slice、recovery、diagnostic 和 privacy 契约。
- [contracts/lsp-diagnostics.md](contracts/lsp-diagnostics.md)：server、protocol、transport、client 与 diagnostics notification 契约。
- [contracts/clang-comparison.md](contracts/clang-comparison.md)：manifest、命令族、比较规则和执行隔离。
- [quickstart.md](quickstart.md)：Go 回归、parser 测试材料、LSP diagnostics、Clang 对照和 corpus 验证入口。

## 项目结构

### 文档（本功能）

```text
specs/004-tops-cpp-tokenizer-parser/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── parser.md
│   ├── lsp-diagnostics.md
│   └── clang-comparison.md
├── checklists/requirements.md
└── tasks.md                 # 由 /speckit.tasks 生成
```

### 源代码（实现阶段）

```text
internal/position/
├── mapper.go                 # UTF-8 byte、行、UTF-16 position/range
└── mapper_test.go

internal/document/
├── store.go                  # 复用 position mapper，保留现有文档契约
└── *_test.go

internal/parser/
├── token.go                  # TokenKind、Token、token range
├── tokenizer.go              # C++/Tops token、literal、comment、operator
├── ast.go                    # SyntaxNode、RecoveryNode、ParseResult
├── conditions.go             # directive、宏三态和 ConditionalRegion
├── parser.go                 # declaration/statement/Pratt expression parser
├── diagnostics.go            # parser diagnostic code、去重和恢复位置
├── tokenizer_test.go
├── parser_test.go
├── conditions_test.go
├── recovery_test.go
├── diagnostics_test.go
└── fixture_test.go

internal/protocol/
├── messages.go               # 复用 Range，必要时扩展通知模型
└── diagnostics.go            # Diagnostic、related info、publish params/编码

internal/server/
├── server.go                 # 文档通知接入 parser 和 publisher 边界
├── diagnostics.go            # ParseResult -> LSP notification
└── *_test.go                 # parser/LSP/stale/close 集成覆盖

testdata/parser/
├── positive/
├── negative/
├── incomplete/
├── macros/
├── topsop/
├── expected/
└── manifest.json

integration/
└── lsp_process_test.go       # 读取 publishDiagnostics frame 的进程级测试
```

**结构决策**：parser、位置映射、protocol 和 server 按所有权拆分。`document` 只提供文本快照；`parser` 不依赖 transport；`server` 是唯一把 parser 结果编码成 LSP notification 的层。`cmd/tops-lsp/main.go` 保持只组装 logger、server、stdin/stdout，不增加 grammar 逻辑。`go.mod` 不增加依赖。

## 实施阶段

### 阶段 1：位置与 protocol 基础

1. 新增共享位置 mapper，覆盖 UTF-8 byte offset、行边界、CRLF、BMP/non-BMP UTF-16 character、EOF zero-width range。
2. 让 `internal/document` 复用该 mapper，保留现有 `Store.Change` 的原子性、版本和错误行为。
3. 为 `internal/protocol` 增加 `Diagnostic`、severity、related information、`PublishDiagnosticsParams` 和标准 notification 编码；不改变现有 request/response/error code。
4. 先运行 document/protocol 既有测试和新增位置/JSON 测试，再进入 parser，避免 parser 范围问题与 LSP 编码问题混在一起。

**阶段验收**：`go test ./internal/document ./internal/protocol` 通过；既有 UTF-16/CRLF 测试结果不变；diagnostics JSON 可以独立反序列化。

### 阶段 2：Tokenizer

1. 定义 `TokenKind`、`Token`、`SourceRange` 和 EOF 不变量。
2. 实现空白/注释、标识符/关键字、数字/字符/字符串/raw literal、标准运算符、预处理行和 Tops 特有 `<<<`/`>>>` token。
3. 保留原文 spelling、trivia、conditional state 和 unterminated 标记；非法字符/字面量生成 lexical diagnostic 输入。
4. 为 UTF-8、CRLF、注释/字符串 EOF、Tops spelling、模板/launch 相邻 `>` 和宏续行添加 token snapshot 测试。

**阶段验收**：tokenizer 对所有 positive/incomplete token 测试材料返回确定序列；非法输入返回 token、EOF 和可定位错误，不 panic。

### 阶段 3：预处理与宏条件

1. 识别 `#include/#define/#undef/#if/#ifdef/#ifndef/#elif/#else/#endif/#pragma` 和未知 directive 的行范围。
2. 实现对象宏/函数宏的原始 token 保留、反斜杠续行和 nested conditional region。
3. 实现 `active`/`inactive`/`unknown` 三态及显式 macro context；不实现跨文件完整宏展开、`##` 结果或 `#` stringification。
4. 在 inactive branch 中保留 token/region 但过滤普通 syntax diagnostics；directive stack 错误始终报告。
5. 测试空 context、定义宏、undefined、冲突/unknown、数值条件、嵌套分支、未闭合 `#if` 和多余 `#endif`。

**阶段验收**：宏测试材料的分支状态稳定；unknown context 不选择隐含 target；inactive 普通错误不泄漏到公开 diagnostics。

### 阶段 4：C++/Tops parser 与 recovery

1. 实现 translation unit、namespace/using、声明符、基础类型、函数、class/struct/enum/union、语句和常用表达式的递归下降 parser。
2. 实现 Pratt expression parser，处理调用、成员、下标、cast、模板参数、运算符优先级和不完整表达式。
3. 实现 Tops qualifier/attribute、向量 spelling、内建变量 spelling、`__attribute__`/`[[...]]` 平衡参数和 `<<<config>>>(args)` 独立 launch node。
4. 为 template/attribute/parameter/statement/conditional 选择同步点，建立 `ErrorNode`、`MissingToken`、opaque 和 incomplete 节点。
5. 按 diagnostic position contract 生成稳定 code、severity、原文 range、related location、recoverable/incomplete 标记；同一恢复节点去重级联错误。
6. 添加 standard C++、Tops、negative、incomplete、unsupported 和 recovery 单测；显式 context 测试 C++11/14/17 与 target macro 边界。

**阶段验收**：parser 测试材料全部返回非空 `ParseResult`；后续独立声明在前置错误后仍可定位；每个支持域至少有正向/负向/不完整测试。

### 阶段 5：测试材料、cc_kernel corpus 与 Clang 对照验证

1. 创建 `testdata/parser/` 分类目录和 `manifest.json`，golden 文件只保存 token/node/diagnostic 摘要，不复制无关完整源码。
2. 从 `range`、`mhc_pre`、`topp_renorm_probs` 等只读 `cc_kernel` 文件提取缩减片段或 manifest 引用，覆盖 host/device、模板、`#if`、Tops qualifier、vector、attribute、DTE spelling 和 launch。
3. 增加 parser 测试材料测试，输出测试材料 ID、context status、diagnostic code/range，不输出完整源代码。
4. 先记录 manifest 中的 `topscc`/Clang driver provenance；再增加可选 Clang 离线对照 test/runner，以同一 source/context 执行 `-###`、`-E -dD`/`-dM -E`、`-fsyntax-only`，适用时执行 AST dump；工具缺失时 skip 对照，不 skip parser tests。
5. 将 GCU400/GCU450 vector parser、GCU400/GCU450 attribute、EFGCU500 launch bounds 的现有 Clang test command 作为 comparison record 样例；不从其他 target 推断宏或限制。

**阶段验收**：测试材料清单可单独运行；Clang/Go 差异都有 context 和 known-difference 记录；`topsop`、`llvm-project` 没有工作区修改。

### 阶段 6：Go server diagnostics 接入

1. 增加 server 内部 publisher，将 `ParserDiagnostic` 转为 `protocol.Diagnostic` 并通过 `transport.Writer` 写标准 notification。
2. 在 `Run` 的 notification 路径传入 writer；保留 `Server.Handle` 现有签名和无 publisher 行为，避免破坏已有同步单测。
3. `didOpen`/成功 `didChange` 后读取 `DocumentState` 值快照，构造 `partial ParseContext` 并同步调用 parser；解析失败不回滚已成功的文本变更。
4. 发布前检查 document version/context version/generation；旧结果丢弃，publisher failure 记录结构化日志并遵循现有 transport fatal 处理。
5. `didClose` 后停止旧结果发布；不新增 client-side keyword scan、compile database resolver、target default 或语义 provider。
6. 更新 protocol/server/integration 测试：现有进程测试在 didOpen/didChange 后读取 diagnostics frame；新增 valid/invalid/incomplete/stale/close/publisher failure 场景。

**阶段验收**：`go test ./...`、进程级 LSP diagnostics 测试和 stdout 纯净性测试通过；现有 lifecycle/document/cancel/error 测试继续通过。

### 阶段 7：基础实现验证

1. 在开发机执行 `go test ./...`、`go test -race ./...`、`go vet ./...`。
2. 执行 parser 测试材料汇总和 range 计数，确认至少 20 个位置案例、四类核心场景和真实 corpus 条目。
3. 在 Clang 工具可用时执行 对照验证 subset，并保存 comparison record；工具不可用时明确报告 skipped，不改变 parser 结果。
4. 检查 `git diff --check`、`git diff --name-only` 和 external checkout 状态，确认无完整源代码/凭据进入日志或测试材料产物。
5. 根据 spec FR/SC、三个契约和 quickstart 建立基础验收记录；未覆盖的完整 grammar、恢复、corpus、对照记录和 diagnostics 元数据继续由 Phase 9/10 处理。

### 阶段 9：基础收敛切片（已完成）

T061-T070 已完成基础收敛：建立核心 C++/Tops parser slice、宏三态、partial context、LSP diagnostics、generation guard、cc_kernel manifest、Clang 对照入口、日志摘要和测试材料基础一致性检查。这些任务提供可运行基线，但不代表完整 grammar 或全部质量门槛已经通过。

### 阶段 10：剩余质量门槛（已完成）

T071-T078 已完成，覆盖：

- C++11/14/17 未覆盖语法和 language-standard 边界；
- ErrorNode/MissingToken/opaque/incomplete 的统一构造与 related location；
- inactive/unknown 宏诊断过滤和嵌套 parent 状态；
- context version 更新和 ParseResult version/generation 发布校验；
- 真实 cc_kernel 结构断言；
- manifest 驱动的 Clang preprocess/syntax/AST/range 对照记录；
- 全量测试材料、golden 和契约一致性；
- LSP diagnostic 元数据映射和日志隐私验证。

**验收结果**：T057-T060、T071-T078 均已完成，并通过 parser/server 窄测试和全量回归。

### 阶段 11：可执行收敛子任务（已完成）

T079-T086 将剩余质量门槛细化为静态 parser golden、恢复诊断 golden、context invalidation、真实 `cc_kernel` AST/diagnostic 断言、comparison record schema、全量测试材料一致性和 LSP wire metadata。全部任务已完成。

Clang 对照记录保存 source/context、阶段状态、Go/Clang AST 与 diagnostic 字段。未配置 Clang 或缺少 Tops 编译上下文时，记录使用 `not-run` 或 known difference；这不阻塞 Go parser/server 测试。配置本地 LLVM fork Clang 后，标准测试材料对照通过，Tops 测试材料的缺失 `.bc` 上下文按 known difference 跳过。

**最终验收**：Phase 1-11 的任务已完成，并通过 `go test ./...`、`go test -race ./...`、`go vet ./...`、LSP integration 和 `git diff --check`。

### 阶段 12：追加收敛任务（已完成）

T087-T093 已完成：补齐 C++11/14/17 测试材料与 `LanguageStandard` gating，扩展 recovery contract，验证真实 context invalidation，接入独立测试材料/corpus expected，生成含 source hash、command provenance、AST/range/diagnostic 和 differences 的 Clang comparison records，并覆盖 `publishDiagnostics` wire metadata。

本地 LLVM fork Clang 已用于生成真实标准测试材料 comparison records；Tops 测试材料因缺少 `gcu300_*.bc` 编译上下文记录为 `context-invalid` 和 known difference。该边界不影响 Go parser/server 的基础测试。

## 依赖关系与并行边界

依赖顺序：

```text
position mapper
  -> protocol diagnostics
  -> tokenizer
  -> macro conditions
  -> parser/recovery
  -> 测试材料/对照验证
  -> server publish integration
  -> baseline validation T057-T060
  -> baseline convergence T061-T070
  -> remaining quality gates T071-T078
  -> actionable convergence subtasks T079-T086
  -> appended convergence tasks T087-T093
  -> final process/race/full validation
```

可并行工作：

- 阶段 2 tokenizer 单测与阶段 1 protocol JSON model 在接口约定确定后可并行；
- 阶段 3 macro condition 单测与阶段 5 测试材料文件整理可并行；
- Clang 对照验证清单和 Go parser unit 测试材料可并行，但不能让对照验证成为 parser test prerequisite；
- server integration 必须等待 `ParseResult`、diagnostic mapping 和 publisher contract 固定。
- T071-T078 必须等待 T061-T070 的基础收敛结果；T074 依赖 US5，T075 依赖 corpus manifest，T076/T077 依赖测试材料格式，T078 依赖 protocol/server diagnostics model。
- T079-T086 已在 T071-T078 的契约基础上完成；T080 依赖恢复节点契约，T082 依赖 context version 契约，T084/T085 依赖 comparison/测试材料格式。
- T087-T093 已在 T079-T086 的 golden、context、comparison 和 LSP 契约基础上完成；T092 依赖本地 Clang 或明确的 `context-invalid` known difference。

不可并行或需先验收：

- position mapper 未通过前不接入 parser diagnostics；
- parser recovery 未通过前不接入 `didOpen`/`didChange`，避免 server 发布不稳定结果；
- `publishDiagnostics` 集成测试必须更新现有进程测试的 frame 读取顺序后再运行全套回归。

## 错误、可观测性和外部数据流

- tokenizer/parser 错误作为结构化 `ParserDiagnostic` 返回，不 panic，不把完整 source 写日志。
- server 只记录脱敏 document ID、version、context version、parse status、diagnostic count、diagnostic code 摘要和 publisher error；不记录 token spelling、完整 message、完整 payload 或凭据。
- parser 只接收 server 传入的文本/context；不读文件、不执行外部命令、不发送网络请求。
- `cc_kernel` 和 `llvm-project` 是开发机只读输入；Clang 对照验证 由测试流程显式执行，结果不进入 server runtime。
- publisher 写失败属于 transport/可观测性失败，不伪造成功诊断；文档文本状态仍按 document store 的已提交结果保留。

## 兼容性与迁移

- 现有 `initialize`、生命周期、文档同步、取消、基础错误和 stdout framing 保持兼容。
- `Server.Handle` 继续返回 request response/exit flag；没有 publisher 时通知不返回 response，已有单测无需依赖 diagnostics frame。
- `initialize` 不必新增 client-specific capability；`textDocument/publishDiagnostics` 是标准 LSP notification。
- 旧客户端仍可完成基础同步；支持 diagnostics 的客户端获得 Go server parser 结果，不支持的客户端不会改变 server 行为。
- 本 feature 不引入 compile database、workspace setting 或 target profile resolver；后续 context 功能只扩展 `ParseContext` 构造，不改变 parser contract。

## 验证命令

开发机、仓库根目录执行：

```sh
go test ./...
go test -race ./...
go vet ./...
go test ./internal/parser/... -run 'Test.*(Token|Parse|Recovery|Diagnostic|Condition|Fixture)'
```

可选 Clang 对照验证：

```sh
/home/carl.du/work/llvm-project/build/bin/clang --version
/home/carl.du/work/llvm-project/build/bin/llvm-lit --version
/home/carl.du/work/llvm-project/build/bin/clang -### <same-source-and-context>
/home/carl.du/work/llvm-project/build/bin/clang -E -dD <same-source-and-context>
/home/carl.du/work/llvm-project/build/bin/clang -dM -E <same-source-and-context>
/home/carl.du/work/llvm-project/build/bin/clang -fsyntax-only <same-source-and-context>
```

这些命令只用于验证，不是 server 运行依赖；`<same-source-and-context>` 必须由测试材料清单提供完整参数。

## 设计完成后的宪法复查

- [x] LSP、server、protocol、transport、parser 和 TypeScript client 的所有权已在 [contracts/lsp-diagnostics.md](contracts/lsp-diagnostics.md) 固定；publisher 是唯一跨边界输出。
- [x] 标准 C++ 与 Tops syntax 分开列出；宏值只来自显式 context；target/pass 不透明传递，不从相邻架构推断。
- [x] parser unit、server integration、位置、恢复、宏、stale、失败变更和 Clang 对照验证 覆盖正常/失败/边界路径。
- [x] 错误 code、severity、range、日志字段、publisher failure 和外部只读数据流已记录。
- [x] 没有新增公开破坏性能力；diagnostics 使用标准 notification，旧 `Server.Handle` 和基础生命周期保持兼容。
- [x] 不存在需要复杂度例外的宪法违规。

## 复杂度记录

无宪法违规项。共享位置 mapper 是为满足既有 document 与新 parser 的同一 UTF-16/CRLF 契约而增加的最小公共模块；parser、protocol、server 分层分别拥有 grammar、消息和发布职责。未选择第三方 parser runtime、运行时 Clang 或客户端重复解析。
