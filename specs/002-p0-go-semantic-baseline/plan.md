# 实施计划：P0 Go 服务器语义基线

**分支**：`002-p0-go-semantic-baseline` | **日期**：2026-10-07 | **规格说明**：[spec.md](spec.md)

**输入**：来自 `specs/002-p0-go-semantic-baseline/spec.md` 的功能规格说明。

**交付边界**：本计划只安排 P0 语义基线的研究、设计、证据复核和文档验收。P0 不创建 Go/TypeScript 源码、Go module、parser 依赖、LSP executable、clangd 扩展、clangd sidecar 或运行时 workload；`llvm-project` 只读作为事实与差分参考。

## 摘要

本 feature 将 P0 规格转化为可执行的文档设计和验收计划，目标是让后续 Go 服务器实现者可以按同一份语义基线开始 P1。计划交付以下内容：

- 八个能力域的可审查语法能力矩阵，逐项记录目标前提、源码/测试证据、状态、限制和验证方法。
- GCU300、GCU400、GCU410、GCU450、GCU500、EFGCU500 六个候选 `TargetProfile`，明确 `__GCU_ARCH__` 与 `__EFGCU_ARCH__` 的分离。
- `CompilationContext` 字段、来源优先级、冲突/缺失/stale 处理和验证命令。
- Go tokenizer、parser、AST、symbol index、semantic、LSP 的责任边界，以及 TypeScript client 和 Clang oracle 的非所有权。
- P1 的 32 个场景入口：八个能力域分别覆盖有效、无效、不完整和目标边界。
- 文档质量门禁、源代码事实核对、LSP/语义契约和 P0/P1 阶段门禁。

本计划不会把 Clang 的结果转发为 Go server 运行时结果，也不会把现有测试状态记录升级为语言服务器已经支持。

## 技术上下文

**语言/版本**：中文 Markdown 和 JSON 元数据；目标实现架构固定为 Go server + TypeScript VS Code client，但本 feature 不锁定 Go、TypeScript 或第三方库版本。

**主要依赖**：`/home/carl.du/work/llvm-project` 当前 checkout 的 Clang driver、TargetInfo、attribute、活动 Tops headers、Clang tests 和 Tops integration tests；本地 `clang`、`llvm-lit` 仅用于事实核对和差分验证。Go server 不依赖 clangd。

**存储**：Markdown 文档、`checklists/requirements.md` 和 `.specify/feature.json`；不引入数据库、网络服务或源码上传。

**测试**：文档使用 `jq`、`rg`、`git diff --check` 和 VS Code diagnostics 检查；目标事实使用 `clang -###`、`-dM -E`、`-fsyntax-only` 和适用 `llvm-lit` fixture；P0 不构建或运行新的 Go/TypeScript 代码，也不运行设备 workload。

**目标平台**：Linux 开发机上的 VS Code 多根工作区；源码和构建配置在开发机检查。测试机不参与本 feature。

**项目类型**：compiler/language-tooling 规格与设计文档；不是可执行服务器或 VS Code 扩展交付。

**性能目标**：P0 文档本身没有运行时性能目标。P1 实现沿用规格中代表性 completion、hover、navigation 请求 95% 在 1 秒内返回的用户目标，并在实现计划中建立实际测量基线。

**约束**：

- 只采用当前 checkout、活动 header、Clang TargetInfo/attribute、driver 输出和测试作为事实证据。
- 未执行的 target 命令、未建立的 Go 语义和未覆盖的 API 保持 `candidate`、`target-dependent`、`blocked` 或 `unsupported`。
- 读取 GCU 条件编译时必须确认目标和完整宏分支；不能用相邻代际或默认宏推断结果。
- `compile_commands` 优先于 workspace fallback；无 target/include/standard 时不使用隐式 GCU300 或系统 LLVM。
- P0 文档不修改 `llvm-project`，不扩展 clangd，不把 Clang oracle 作为运行时 fallback。
- 诊断和日志设计不得记录密钥、完整源码或无关用户数据。

**规模/范围**：8 个能力域、6 个候选 profile、1 个 `CompilationContext` 模型、6 个 Go server 层次、32 个 P1 场景、3 个契约文件和 1 个 quickstart。P0 不建立源码目录。

## 宪法检查

*门禁：阶段 0 研究前通过，阶段 1 设计后再次通过。*

- [x] **LSP 契约优先**：`contracts/lsp-boundary.md` 定义 owner、输入/输出、错误、取消、日志和兼容性；P0 不新增未记录的扩展消息。
- [x] **Go/TypeScript 分层边界**：Go server 拥有语言分析、工作区状态和 LSP 行为；TypeScript client 只拥有 VS Code 生命周期、配置转发和用户命令。
- [x] **Tops C/C++ 语义保真**：能力矩阵和 `research.md` 回溯到活动 headers、driver、TargetInfo、attribute 和测试；目标条件按 profile 分离。
- [x] **行为优先的验证**：P1 场景矩阵覆盖正常、错误、不完整和目标边界；Clang 差分不替代 Go server 自有测试。
- [x] **可观测且兼容地演进**：`LanguageDiagnostic`、context 状态、stale 结果、日志隐私和 unsupported/blocked 降级均有契约。
- [x] **没有违规项**：本计划只生成文档，不引入运行时复杂度；复杂度记录保持为空。

## 项目结构

### 文档（本功能）

```text
specs/002-p0-go-semantic-baseline/
├── spec.md                              # P0 功能规格
├── plan.md                              # 本文件
├── research.md                          # 阶段 0 研究结论与证据入口
├── data-model.md                        # 阶段 1 数据模型与状态
├── quickstart.md                        # 文档和源码事实核对步骤
├── contracts/
│   ├── capability-matrix.md             # 八域能力条目契约
│   ├── lsp-boundary.md                  # Go/TypeScript/Clang oracle 边界
│   └── roadmap-gates.md                 # P0/P1 阶段门禁
└── checklists/requirements.md            # 规格质量清单
```

### 源代码（参考 checkout，不由本 feature 修改）

```text
/home/carl.du/work/llvm-project/
├── clang/include/clang/Driver/           # .tops、-Tops、tops language 输入入口
├── clang/include/clang/Basic/             # language standard、CudaArch、attribute
├── clang/lib/Basic/Targets/               # DTU/EFGCU target 和宏分支
├── clang/lib/Driver/                      # TOPS/EFGCU target command 展开
├── clang/lib/Headers/tops/                # Tops 属性、builtin、vector、DTE/API headers
├── clang/test/DTU_test/                   # GCU/Tops Sema、parser、CodeGen oracle
├── clang/test/CodeGenEFGCU/               # EFGCU500 target tests
└── tops/integration_test/cases/language/  # C++ 和 Tops integration fixtures/status
```

**结构决策**：所有 P0 设计工件放在 feature 目录；`llvm-project` 只提供证据；不创建 `src/`、`go.mod`、`package.json` 或伪 LSP 接口。后续 P1 实现的 Go/TypeScript 目录由 `/speckit.tasks` 和独立实现计划确定，不在本 feature 中提前制造。

## 阶段执行

### 阶段 0：研究与事实核对

1. 读取并审查规格中的八域清单、六 profile、`CompilationContext` 和 32 个 P1 场景。
2. 核对 `.tops`、`.cpp + -Tops`、`-x tops`、`tops` language、`-mcpu`/offload arch 和 driver 展开路径。
3. 读取 GCU/EFGCU TargetInfo 的完整宏分支，记录 `__GCU_ARCH__` 与 `__EFGCU_ARCH__` 的来源，不使用相邻代际推断。
4. 核对活动 Tops headers、Clang attribute、builtin/vector/API 入口和现有测试状态；把只存在源码证据的条目标为 `candidate` 或 `target-dependent`。
5. 固化 Clang oracle 命令族：`-###`、`-dM -E`、include trace、`-fsyntax-only` 和适用 `llvm-lit`；只记录命令和结果，不引入运行时依赖。

**阶段 0 输出**：[research.md](research.md) 和可复核的证据入口；不创建源码。

### 阶段 1：设计与契约

1. 用 [data-model.md](data-model.md) 将 `SyntaxCapability`、`EvidenceRecord`、`TargetProfile`、`CompilationContext`、`SourceDocument`、`SymbolIndex`、`LanguageDiagnostic`、`LSPCapabilityContract` 和 `P1Fixture` 规范化。
2. 用 [contracts/capability-matrix.md](contracts/capability-matrix.md) 固化八域条目字段、证据优先级和状态门禁。
3. 用 [contracts/lsp-boundary.md](contracts/lsp-boundary.md) 固化 Go server、TypeScript client、Clang oracle 的输入/输出/错误/日志/兼容性边界。
4. 用 [contracts/roadmap-gates.md](contracts/roadmap-gates.md) 定义 P0/P1 的入口证据、范围、验证、退出条件和 fallback。
5. 用 [quickstart.md](quickstart.md) 固化文档校验、源码事实核对、profile 验证和不运行实现代码的操作顺序。
6. 阶段 1 结束后重新执行 Constitution Check；只有字段、契约和门禁完整，才允许进入 `/speckit.tasks`。

**阶段 1 输出**：本计划引用的设计工件；仍不创建 Go/TypeScript/C++ 实现。

### 阶段 2：任务拆分准备

1. 将研究项拆成证据核对任务，将设计项拆成文档/契约任务，将 P1 32 场景拆成可独立验收的 fixture 任务。
2. 为每个任务标注受影响层、输入 context、目标 profile、错误行为、验证命令和是否需要 Clang oracle。
3. 明确 P0 任务完成不等于 Go server `supported`；实现任务必须在 P1 单独建立 Go 测试、LSP contract test 和差分测试。

**阶段 2 输出**：供 `/speckit.tasks` 使用的任务拆分依据；本计划不创建 `tasks.md`。

## 交付与验证矩阵

| 产物 | 责任 | 主要验证 | 完成条件 |
| --- | --- | --- | --- |
| `spec.md` | 规格基线 | 质量清单、八域/六 profile/32 场景计数 | 无未解决 clarification marker；P0 非目标明确 |
| `research.md` | 事实与方案 | 路径存在、完整宏分支、命令可审查 | 每个重要结论有当前源码或测试入口 |
| `data-model.md` | 数据和状态 | 字段、关系、状态迁移审查 | 能表达 context、profile、证据、诊断和 fixture |
| `contracts/capability-matrix.md` | 能力条目契约 | 正/负/不完整/目标边界规则审查 | 条目状态不会从源码出现直接升级为 supported |
| `contracts/lsp-boundary.md` | 跨边界契约 | owner、负载、错误、stale、隐私、兼容性审查 | 客户端不复制语义，Clang 不进入 runtime |
| `contracts/roadmap-gates.md` | 阶段门禁 | P0/P1 entry/exit/fallback 审查 | 失败时保留 blocked/target-dependent/unsupported |
| `quickstart.md` | 维护者验证入口 | `jq`、`rg`、`git diff --check`、Clang 命令审查 | 新维护者可在开发机复核文档与事实 |

## 依赖、风险与控制

| 风险 | 控制措施 | 触发状态 |
| --- | --- | --- |
| Go 语义与 Clang 行为漂移 | Clang 只做 oracle；每个 P1 场景必须有 Go 自有预期和差分记录 | `candidate`/`blocked` |
| GCU 代际条件被错误合并 | 每个 profile 执行完整宏分支核对；EFGCU500 独立处理 | `target-dependent` |
| `.tops`/`-Tops`/`-x tops` 产生不同 context | 保存 raw/normalized args、pass、宏和 include 顺序，禁止隐式归一化 | `invalid`/`partial` |
| 缺少现成 Go 工程入口 | P0 只交付文档；后续任务先建立 Go module 和最小测试骨架 | 不提前宣称实现 |
| 测试状态被误读为全量 device 支持 | 保留 `STATUS.md` 的 BUG-2/4/5/6，并区分 compile、runtime 和 LSP 验证 | `target-dependent` |
| 增量输入产生 stale 诊断 | 使用 context/document version；旧结果不能覆盖新结果 | `analysis-degraded`/`stale-result` |
| 日志暴露源代码或凭据 | 契约只允许脱敏 URI、profile、耗时、错误 code 和重启原因 | `blocked` |
| 设计文档和后续任务发生漂移 | `/speckit.tasks` 前检查 spec、plan、research、data-model、contracts、quickstart 一致 | 阻止进入下一阶段 |

## 复杂度记录

无宪法违规项，无需复杂度例外。拆分研究、模型、契约和 quickstart 是为了让后续任务可以独立评审，不引入运行时抽象或额外实现依赖。
