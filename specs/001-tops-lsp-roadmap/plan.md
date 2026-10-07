# 实施计划：Tops C++ Language Server Roadmap

**分支**：`001-tops-lsp-roadmap` | **日期**：2026-10-07 | **规格说明**：[spec.md](spec.md)

**输入**：来自 `specs/001-tops-lsp-roadmap/spec.md` 的功能规格说明；本次用户要求为“制定撰写 roadmap 文档的实施计划”。

**交付边界**：本计划只安排 roadmap 文档的研究、设计、评审和验证，不创建 Go 服务器、TypeScript 扩展、parser grammar 或运行时实现。实际 Go server 从零实施必须在后续 `/speckit.tasks` 和实现计划中单独立项；不扩展或依赖 clangd。

## 摘要

本 feature 交付一套可审查、可回归的 Tops C++ 语言服务器与 VS Code 客户端 roadmap 文档，最终发布物固定为 `doc/tops-cpp-language-server-roadmap.md`。文档以当前 LLVM/Tops 工具链为事实来源，覆盖 8 个能力域：标准 C++、执行空间、存储空间、启动/资源属性、向量/数值类型、内建变量、TCLE/API 和目标条件。

研究结论是：Clang 已识别 `.tops`、`-Tops` 和 `-x tops`，但当前 `tops-lsp` 没有现成 Go/TypeScript/parser 工程。架构已确定为 Go server 从零开发：Go 自己拥有 tokenizer/parser、AST、索引、目标语义和 LSP 行为；Clang/Tops 仅作为事实参考、编译兼容性 oracle 和差分验证工具，clangd 不进入运行链路。

本计划生成并维护以下设计产物：语法能力矩阵、能力与目标数据模型、Go/TypeScript LSP 边界契约、P0-P4 roadmap 门禁和文档验证 quickstart。

## 技术上下文

**语言/版本**：roadmap 交付使用中文 Markdown 和 JSON 元数据；目标实现确定为 Go 从零开发的 server + TypeScript VS Code client。当前仓库尚无 `go.mod`、`package.json` 或版本基线，Go/TypeScript 版本在后续实现任务中锁定。

**主要依赖**：Go server 自有 parser、AST、符号索引、目标语义和 LSP 模块；TypeScript VS Code client；当前 `/home/carl.du/work/llvm-project` checkout 的 Clang/Tops frontend、`clang/lib/Headers/tops`、Clang attribute/TargetInfo、`clang/test/DTU_test/topscc`、`clang/test/CodeGenEFGCU`、`tops/integration_test/cases/language` 仅用于事实和差分验证；本地 `build/bin/clang`、`llvm-lit` 用于 P0/P1 核对。

**存储**：Markdown 文档和 `.specify/feature.json`；不引入数据库或外部服务。

**测试**：文档结构使用 `get_errors`、`git diff --check`、`jq`、`rg` 校验；Go server 使用 parser/semantic/LSP unit、contract、integration 和差分 fixture；Tops 事实使用本地 `clang` 和现有 lit fixture 核对；本 feature 不运行设备 workload。

**目标平台**：Linux 开发机上的 VS Code 多根工作区；参考代码位于同一工作区的 `llvm-project` 根。

**项目类型**：技术路线文档/语言工具 roadmap；不是可执行服务器或 VS Code 扩展交付。

**性能目标**：本次文档校验不设运行时性能目标；roadmap 保留规格中的用户目标，即代表性补全、悬停和导航请求 95% 在 1 秒内返回，实际实现阶段再建立测量基线。

**约束**：事实必须能回溯到当前 checkout、活动 header、Clang 定义、测试或实际工具输出；未验证的 Go 语义写成待 P0/P1 验证。不得把 clangd 作为运行依赖，不得把 Clang 差分结果替代 Go server 自身测试；不得把旧文档、其他架构或系统 LLVM 推断为当前行为；不得上传源代码到外部服务；不得修改 `llvm-project` 参考 checkout。

**规模/范围**：8 个语法能力域、P0-P4 五个 roadmap 阶段、1 个能力矩阵、1 个数据模型、3 个契约文档、1 个 quickstart 和 1 个最终发布文档 `doc/tops-cpp-language-server-roadmap.md`；本次不创建源码目录。

## 宪法检查

*门禁：阶段 0 研究前和阶段 1 设计后均通过。*

- [x] 已明确 LSP、服务器和扩展的职责，以及所有变更的跨边界契约。职责与请求路径记录在 [contracts/lsp-boundary.md](contracts/lsp-boundary.md)，roadmap 只承诺标准 LSP 或有版本的扩展契约。
- [x] Tops C/C++ 的语法、语义和目标假设明确且可追溯。证据入口、状态规则和 P0 验证记录在 [research.md](research.md) 与 [contracts/capability-matrix.md](contracts/capability-matrix.md)。
- [x] 自动化测试覆盖变更行为，包括相关失败路径和边界路径。文档静态检查、Clang 差分 P0 检查、Go LSP 契约测试和各阶段正/负/边界 fixture 记录在 [quickstart.md](quickstart.md) 与 [contracts/roadmap-gates.md](contracts/roadmap-gates.md)。
- [x] 已说明错误行为、诊断、日志和外部数据流。LSP 错误 code、stale 结果、日志敏感数据和本地数据边界记录在 [contracts/lsp-boundary.md](contracts/lsp-boundary.md)。
- [x] 对于公开变更，已记录向后兼容性以及迁移或拒绝行为。标准 LSP 优先、扩展命名空间版本化、设置迁移和 unsupported/blocked 降级规则已记录在契约与阶段门禁中。
- [x] 没有宪法违规项。文档产物拆分是为了满足能力矩阵、契约和阶段评审，不引入运行时复杂度；不需要 Complexity Tracking 例外。

## 项目结构

### 文档（本功能）

```text
specs/001-tops-lsp-roadmap/
├── spec.md                         # 功能规格说明
├── plan.md                         # 本文件
├── research.md                     # 阶段 0 事实与方案决策
├── data-model.md                   # 阶段 1 文档数据模型
├── quickstart.md                   # 阶段 1 文档与工具链验证指南
├── contracts/
│   ├── capability-matrix.md        # 语法能力条目契约
│   ├── lsp-boundary.md             # 服务器/客户端 LSP 边界契约
│   └── roadmap-gates.md            # P0-P4 阶段门禁契约
└── checklists/requirements.md      # 规格质量清单

doc/
└── tops-cpp-language-server-roadmap.md  # 唯一最终发布文档，由后续发布任务生成
```

### 源代码（仓库根目录）

```text
tops-lsp/
├── .specify/
├── .github/prompts/ and .github/agents/
├── .vscode/
└── specs/001-tops-lsp-roadmap/

llvm-project/                         # 只读参考 checkout，不由本 feature 修改
├── clang/lib/Headers/tops/
├── clang/include/clang/Basic/
├── clang/test/DTU_test/topscc/
├── clang/test/CodeGenEFGCU/
└── tops/integration_test/cases/language/
```

**结构决策**：研究、设计和验收工件放在 feature 目录；最终发布文档固定放在 `doc/tops-cpp-language-server-roadmap.md`；`llvm-project` 只提供语义和验证证据。后续源码必须采用 Go server 从零实现的模块边界，TypeScript extension 负责 VS Code 生命周期；parser、semantic index、contract test 和 build task 的真实目录在后续任务中确定，当前不制造空目录或伪接口，也不修改 clangd。

## 阶段执行

### Phase 0：Go 服务器语义基线

1. 用 [contracts/capability-matrix.md](contracts/capability-matrix.md) 从 Tops headers、Clang attribute/TargetInfo、driver 和测试状态建立 8 个能力域的候选条目。
2. 核对 `.tops`、`.cpp + -Tops`、`-x tops`、C++ 标准、目标参数、include 路径和宏，定义 Go server 的 `CompilationContext` 输入和优先级。
3. 设计 Go tokenizer/parser、AST、符号索引、目标语义和增量分析的最小边界；每个 P1 能力都必须有 Go 侧数据结构、诊断规则和 LSP 结果归属。
4. 使用本地 `build/bin/clang` 做 `-fsyntax-only` 差分验证，确认 Clang/Tops 结果可以作为兼容性 oracle；Clang 结果不能替代 Go server 自身的行为测试。

**Phase 0 输出**：已经生成的 [research.md](research.md)、能力矩阵初稿、目标 profile 候选、Go 语义边界和差分验证规则。

### Phase 1：数据模型与跨边界设计

1. 按 [data-model.md](data-model.md) 定义 `SyntaxCapability`、`EvidenceRecord`、`TargetProfile`、`CompilationContext`、`LanguageDiagnostic`、`LSPCapabilityContract` 和 `RoadmapMilestone` 的字段、关系和状态迁移，并标注 Go server 的所有权。
2. 按 [contracts/lsp-boundary.md](contracts/lsp-boundary.md) 记录编辑器动作、LSP 请求、服务器分析、客户端展示、错误、日志和兼容性路径；不在客户端复制语义决策。
3. 按 [contracts/roadmap-gates.md](contracts/roadmap-gates.md) 为 P0-P4 写入口证据、范围、验证、退出条件、降级状态和 owner。
4. 用 [quickstart.md](quickstart.md) 固化文档检查、源码证据检查和 P0 工具链验证；将未验证结论保留在待验证状态。
5. 重新执行 Constitution Check，确认新增契约和 roadmap 产物没有引入未记录的公开变更或外部数据流。

**Phase 1 输出**：本计划引用的全部设计文档，并为 `doc/tops-cpp-language-server-roadmap.md` 准备发布内容和来源引用。完成后才能进入 `/speckit.tasks`，把文档审查结果拆为实施任务。

## 交付与验证矩阵

| 产物 | 解决的问题 | 主要验证 | 完成条件 |
| --- | --- | --- | --- |
| `research.md` | 事实源和 Go 从零实现路线 | 路径存在、源码/测试证据可定位、Go 所有权已明确 | 无未解释的 `NEEDS CLARIFICATION`；不把推测写成事实 |
| `data-model.md` | roadmap 信息如何稳定记录 | 字段覆盖 spec 的实体、关系和状态 | 能表达目标条件、证据、诊断和阶段门禁 |
| `contracts/capability-matrix.md` | 语法能力如何判定支持 | 正/负/目标边界规则审查 | 每个条目能追溯到证据和验证动作 |
| `contracts/lsp-boundary.md` | Go/TypeScript 如何协作 | 标准 LSP 方法、错误、日志、兼容性审查 | 每个公开能力有 owner、负载、错误和用户结果 |
| `contracts/roadmap-gates.md` | 每阶段何时可进入下一阶段 | P0-P4 入口/退出/降级检查 | 失败不会静默升级为 supported |
| `quickstart.md` | 维护者如何复核文档和工具链 | 文档静态检查、本地工具版本和源码入口 | 新维护者可以复现检查并知道预期结果 |
| `doc/tops-cpp-language-server-roadmap.md` | 对外发布最终 roadmap | 与 spec、research、data model 和 contracts 的一致性检查 | 路径固定、来源完整、架构和 P0-P4 门禁一致 |

## 依赖、风险与控制

| 风险 | 控制措施 | 触发的 roadmap 状态 |
| --- | --- | --- |
| Go server 与 Clang 语义可能漂移 | P0/P1 建立 Clang 差分 oracle、Go 正/负/边界 fixture 和版本化能力矩阵 | `blocked` 或 `target-dependent` |
| 旧文档与活动 header 不一致 | 能力矩阵提高活动 header/attribute/test 的证据优先级 | `deprecated-source` |
| 不同 GCU 目标共享错误语义 | 每个目标 profile 读取完整条件分支并记录适用范围 | `target-dependent` |
| tops-lsp 没有现成 Go 实现入口 | 当前只交付文档；后续任务先建立 Go module、parser、semantic index 和 LSP 骨架 | 不提前宣称实现 |
| LSP 跨层行为难以回归 | 为每个能力记录标准 LSP 流程和契约测试入口 | 未通过则阶段不升级 |
| 日志泄露源码或配置 | 契约限制日志字段，不记录密钥、完整源码和无关用户数据 | `blocked` |

## 复杂度记录

无宪法违规项，无需记录复杂度例外。拆分为研究、数据模型、三个契约和 quickstart 是为了让 roadmap 可以独立评审和复用，不是引入运行时抽象。
