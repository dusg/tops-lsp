# 实施计划：Tops C++ 基础语言功能（错误提示、补全、悬停）

**分支**：`007-core-language-closure` | **日期**：2026-10-19 | **规格说明**：[spec.md](spec.md)

**输入**：来自 `specs/007-core-language-closure/spec.md` 的功能规格说明。

## 摘要

在已经能跳定义、查引用、列文档符号的基础上，补上三件事：

1. **错误提示合并**：把语法、标准 C++ 语义、索引降级、Tops 扩展语义四类问题合成一份清单，去掉重复项、固定排序、按当前文档版本和上下文版本发布；改好代码后发空清单清掉旧提示。
2. **补全**：按光标位置的语法规则和语义可见性给出候选；分析还没完成时只给一条“正在分析中”的提示项。
3. **悬停**：显示名字、种类、已知类型或签名、声明位置、来源和上下文状态；分析还没完成时显示“正在分析中”。

同时把六类请求的行为规则统一起来：没打开文件、参数不对、取消、结果过期、还没有结果。导航类请求（跳定义、查引用、列符号）在还没有结果时最多等 2 秒，超时返回标准 `ContentModified` 错误。

验收只用三个真实文件走端到端服务器会话：一个能正常分析的 Tops kernel、一个标准 C++ 写错的文件、一个 Tops 规则写错的文件。

## 技术上下文

**语言/版本**：Go `1.26.8`，沿用当前 module；不新增语言运行时。

**主要依赖**：现有 `internal/parser`、`internal/index`、`internal/semantic`、`internal/server`、`internal/protocol`、`internal/position`、`internal/transport`、`internal/document`。不新增第三方依赖。

**存储**：沿用内存结构：`document.Store`（打开的文档）、`index.IndexSnapshot`（符号与引用）、`semantic.SemanticSnapshot`（语义事实与诊断）、`semantic.SemanticStore`。本功能新增“分析完成通知”内存信号，用于等待和判断“分析进行中”，不引入数据库或磁盘缓存。

**测试**：`go build ./...`、`go vet ./...`、`go test -p 128 -count=1 ./...`、`go test -p 128 -count=1 ./integration`。验收测试复用 `integration/lsp_process_test.go` 已有的进程启动、读写响应、等待诊断等辅助函数，新增用例放在 `integration/lsp_core_language_test.go`。

**目标平台**：Linux 开发机；不需要 GCU 硬件、GCUSIM、Clang 运行时或测试机。

**项目类型**：Go 语言服务器（LSP over stdio）。

**性能目标**：文件不超过 500 行时，改完代码 2 秒内能看到新诊断（规格 SC-005）；导航等待上限固定 2 秒；补全和悬停不等待分析，立即返回。

**约束**：

- 只用标准推送诊断，不注册 `textDocument/diagnostic`、`workspace/diagnostic` 或诊断刷新。
- 语义状态不写进自定义响应字段；附加信息只放标准 `Diagnostic.data` 或补全项附加字段。
- 用户可见文字统一英文。
- 日志不记完整源码，不记凭据。
- 补全不做成员访问、自动导入、代码片段、重载排序、文档渲染、跨仓搜索。
- 悬停不显示注释文档、外部网页或推测性说明。

**规模/范围**：当前打开的文档、已索引的依赖或头文件、工具链内建宏与工程宏；不扫描整个工作区，不访问网络。v1 不做模板实例化、完整重载决议、完整常量求值、完整访问控制、链接或代码生成。

## 宪法检查

*门禁：阶段 0 研究前检查一次，阶段 1 设计后再次检查。*

- [x] **原则一 LSP 契约优先**：`contracts/lsp-core-language.md` 固定补全、悬停、定义、引用、文档符号的请求参数、成功结果、空结果、错误、取消、过期和“分析进行中”行为；`contracts/diagnostics-pipeline.md` 固定诊断的合并、去重、排序、级别和版本字段；`contracts/completion-candidates.md` 固定候选来源与语法规则表。能力声明只增加 `completionProvider` 和 `hoverProvider`，不声明补全二次解析。
- [x] **原则二 Go/TypeScript 分层边界**：所有语言判断留在 Go 服务器；TypeScript 侧不重复计算可见性、类型或诊断。新增协议类型只描述传输结构，不含语义判断。
- [x] **原则三 Tops C/C++ 语义保真**：候选和悬停区分标准 C++ 事实与 Tops 扩展事实；Tops 内建变量和关键字来自 `internal/parser` 已有的 `topsKeywordSpellings` 与索引中的 `SymbolKindBuiltin`；上下文为 `partial`/`missing`/`invalid` 时不出确定结论。
- [x] **原则四 行为优先的验证**：新增端到端用例覆盖六个功能、五类行为（正常、空结果、参数错误、取消、过期）、分析进行中、2 秒超时，以及三个测试文件；已存在的包内测试继续运行，不因本功能删除。
- [x] **原则五 可观测且兼容地演进**：新增日志分类（请求结果、空结果、参数错误、取消、过期丢弃、导航超时），不记源码；现有公开能力和环境变量保持兼容；`tops-semantic-context-degraded` 的严重级别由 Information 调整为 Warning，属于已记录的对外可见变化，需同时更新语义测试数据和契约文档。
- [x] **无未记录例外**：见下方「复杂度与偏差记录」。

## 项目结构

### 文档（本功能）

```text
specs/007-core-language-closure/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── lsp-core-language.md
    ├── diagnostics-pipeline.md
    └── completion-candidates.md
```

### 源代码（仓库根目录）

```text
internal/parser/
└── keywords.go                 # 新增：导出关键字与 Tops 关键字清单（复用现有表）

internal/index/
├── model.go                    # 读用：宏已经作为 SymbolKindMacro 符号存在，不改结构
└── query.go                    # 修改：新增按位置取可见名字的查询

internal/protocol/
├── completion.go               # 新增：补全请求、补全项、补全列表类型
├── hover.go                    # 新增：悬停请求、悬停结果、标记内容类型
├── messages.go                 # 修改：InitializeParams 增加客户端能力（悬停标记类型）
└── errors.go                   # 修改：补充 ContentModified 常量（如尚不存在）

internal/server/
├── completion.go               # 新增：补全请求处理与候选生成
├── completion_rules.go         # 新增：光标位置判断与语法规则表
├── hover.go                    # 新增：悬停请求处理与内容渲染
├── navigation.go               # 修改：导航等待上限与超时错误
├── analysis_wait.go            # 新增：分析完成通知、进行中判断、带超时等待
├── diagnostics.go              # 修改：诊断去重辅助
├── observability.go            # 修改：聚合发布时应用去重与固定排序
└── server.go                   # 修改：能力声明、补全与悬停分派

internal/semantic/
├── diagnostics.go              # 修改：降级诊断级别改为 Warning
├── checker.go                  # 修改：降级诊断严重级别改为 Warning
└── fixture.go                  # 修改：预期级别字符串同步

integration/
├── lsp_core_language_test.go   # 新增：三个测试文件驱动的端到端用例
└── lsp_process_test.go         # 复用：进程启动与响应读取辅助

testdata/lsp/fixtures/
├── valid-kernel.tops           # 新增：可编辑的 Tops kernel
├── std-cpp-error.tops          # 新增：标准 C++ 语法/语义错误
└── tops-semantic-error.tops    # 新增：Tops 扩展规则错误
```

**结构决定**：沿用仓库现有单模块分层（`internal/*` 按职责分包，`integration/` 放真实进程测试）。不新增服务、不新增构建目标。

## 阶段 0：研究结论

详细内容见 [research.md](research.md)。要点：

1. 导航等待用“分析完成通知”实现，不用轮询；超时用 `context.WithTimeout` 返回 `ContentModified`。
2. “分析进行中”通过分析队列状态和完成通知判断，补全与悬停据此返回提示。
3. 悬停标记类型从初始化参数读取；客户端没声明时按 Markdown 处理（项目显式选择）。
4. 诊断去重键为“编号 + 起止位置 + 级别 + 消息”，排序键为“起位置、止位置、级别、编号、消息”。
5. 宏已经是 `SymbolKindMacro` 符号（`internal/index/builder.go` 的 `symbolInfo`），补全直接复用符号索引和 `PredefinedMacros`，不新建宏表。
6. 降级诊断级别改为 Warning，需要同步三处：诊断规格表、检查器、测试数据期望。

## 阶段 1：设计与契约

- [data-model.md](data-model.md)：请求、分析结果、诊断、补全项、悬停结果、上下文、宏（复用符号）、分析完成通知、测试文件。
- 契约：
  - [contracts/lsp-core-language.md](contracts/lsp-core-language.md)
  - [contracts/diagnostics-pipeline.md](contracts/diagnostics-pipeline.md)
  - [contracts/completion-candidates.md](contracts/completion-candidates.md)
- [quickstart.md](quickstart.md)：如何用三个测试文件跑通验收。

### 设计后复查

*阶段 1 完成后重新对照宪法五项原则。*

- 原则一：三个契约文档覆盖了能力声明、请求参数、结果、空结果、错误、取消、过期、分析进行中、2 秒超时，以及初始化时读取的客户端能力；没有新增私有响应字段。
- 原则二：新增的 `internal/protocol` 类型只是字段容器；候选可见性、诊断级别、悬停内容全部在 `internal/server`、`internal/index`、`internal/semantic` 内决定。
- 原则三：宏复用已索引的 `SymbolKindMacro`，内置变量复用 `SymbolKindBuiltin` 与可用性字段；契约里明确了“条件未知时不出确定结论”。
- 原则四：端到端用例覆盖三个测试文件与六类行为；计划里列出了需要的用例类型，实现阶段按此编写。
- 原则五：`ContentModified` 常量会新增到 `internal/protocol/errors.go`；降级诊断级别变化同时影响已有测试数据，已列入实现任务。
- 结论：设计未引入新的宪法例外。

## 复杂度与偏差记录

| 偏差 | 为什么需要 | 更简单的替代方案为何不够 |
|------|------------|--------------------------|
| 验收只做端到端进程测试，不为本功能新增包内单元测试 | 规格澄清记录已确认（用户选择 B）；三个测试文件要在真实协议路径上验证完整行为 | 只做包内测试无法覆盖协议序列化、进程生命周期和真实版本竞争；只做端到端会减慢定位——因此保留现有包内测试作为回归基线 |
| 调整 `tops-semantic-context-degraded` 的严重级别 | 规格澄清记录要求统一 Warning | 保持 Information 会让用户按“提示”忽略它，与规格的成功标准冲突 |

## 一致性修复记录（2026-10-19）

`/speckit-analyze` 报告的问题已逐项修复，以下是修复位置。

| 编号 | 问题 | 修复位置 |
|------|------|----------|
| C1 | SC-004 的 100 次版本竞争与取消无任务 | `tasks.md` T026 增加 100 次循环统计 |
| C2 | “收到空诊断数组”可能被清理通知满足 | `tasks.md` T019 改为“先收到非空数组，再收到同版本空数组” |
| C3 | FR-008、FR-009 无任务 | `tasks.md` 新增 T020（stale 与关闭清理用例）、T022（丢弃与清理实现核对） |
| C4 | SC-005 的 2 秒计时无断言 | `tasks.md` T014 增加时间戳与轮询计时断言 |
| C5 | SC-009 的分析进行中行为无测试 | `tasks.md` 新增 T027 `TestLSPProcessCoreLanguageAnalyzingState` |
| A1 | 空文件时悬停“正在分析”的 `range` 未定义 | `contracts/lsp-core-language.md` 4.5.1 增加边界规则表 |
| A2 | “声明开头”缺少可判定的 token 依据 | `contracts/completion-candidates.md` 第 3 节改为按前一非空白 token 判定 |
| A3 | “六个方法各跑五类用例”表述笼统 | `tasks.md` T026 改为按方法列出适用用例 |
| A4 | 用户可见文字必须英文但无检查 | `tasks.md` T032 增加英文与中英混排检查 |
| C6 | FR-028 无任务 | `tasks.md` T028 增加来源记录核对 |
| C7 | FR-037 环境独立性无任务 | `tasks.md` T031 增加无驱动、无 Clang 环境的复跑要求 |
| C8 | FR-016 的排除项未逐项验证 | `tasks.md` T017 要求逐项返回空名单且不报错 |
| I1 | 数据模型把 `type` 一律映射为 `Class`，与任务定义的 `Struct`/`Enum` 冲突 | `data-model.md` 种类映射改为按 `TypeKind` 细分，与现有 `protocolSymbolKind` 一致 |
| I2 | 带字母后缀的需求编号未在任务中显式引用 | `tasks.md` 中所有任务补齐需求编号与成功标准编号引用 |
| I3 | 空结果形状在多个任务重复描述 | `tasks.md` 改为只在 T029 规定，其他任务引用 T029 |

修复后覆盖情况：FR 39 条、SC 9 条全部有对应任务；任务总数 34（T001–T034）。

## 第二轮一致性修复（2026-10-19）

第二轮复查又发现 12 处跨文档不一致，已全部修复。

| 编号 | 问题 | 修复位置 |
|------|------|----------|
| D1 | FR-006 只写 `tops-semantic-context-degraded`，但实现另有 `tops-semantic-context-missing` | `spec.md` FR-006 明确两个编号各自的适用状态 |
| D2 | FR-002 说“四类来源”，诊断契约实际列了五行（含上下文状态） | `spec.md` FR-002 改为“全部来源”，`contracts/diagnostics-pipeline.md` 第 1 节改为“来源”并说明归属 |
| D3 | FR-005 未说明候选歧义的级别 | `spec.md` FR-005、`data-model.md` 级别表补充“歧义用 Warning” |
| D4 | T018 把悬停来源写进 `data` 字段，但 LSP 悬停结果没有该字段 | `tasks.md` T018 改为“补全用 `data`，悬停写进可见文字” |
| D5 | T014 未引用 SC-002 | `tasks.md` T014 补齐 |
| D6 | 计划的项目结构缺 `internal/server/completion_rules.go` | `plan.md` 项目结构补齐 |
| D7 | `quickstart.md` 仍写“三个子测试” | 改为六个子测试的表格 |
| D8 | 契约未规定“已有失效旧结果”与“分析进行中”的优先级 | `contracts/lsp-core-language.md` 3.2 增加判定顺序表 |
| D9 | T020 要求“修改相关头文件”，但没有对应 fixture | `tasks.md` T020 改为说明既有测试覆盖，不新增 fixture |
| D10 | `data-model.md` 缺 `tops-semantic-ambiguous-name` 级别；`documentVersion` 要求与契约不一致 | `data-model.md` 级别表与字段约束同步 |
| D11 | `quickstart.md` 缺新增功能的核对项与排查项 | 补充排序、旧诊断丢弃、关闭清理、排除项、版本竞争、文字语言等 |
| D12 | 契约 4.4“`[]` 中只含一条提示项”表述不清 | 改为“返回只含一条提示项的列表” |

第二轮修复后未再发现 CRITICAL 或 HIGH 级问题。

## 第三轮一致性修复（2026-10-19）

| 编号 | 问题 | 修复位置 |
|------|------|----------|
| D13 | 提示项 `sortText` 写 `0`，与补全契约“三位数字前缀”冲突 | `contracts/lsp-core-language.md` 改为 `000`；`contracts/completion-candidates.md` 明确 `000` 保留给提示项、真实候选从 `001` 起 |
| D14 | 契约 3.2 把“文档未打开”单独列成一张单行表，结构冗余 | 合并为一张判定顺序表（5 行） |
| D15 | `research.md` 仍写 `sortText` 以 `0` 开头 | 改为 `000` |
| D16 | 规格正文用带字母的示例编号做说明，干扰需求编号统计 | 改为“编号（例如 `FR-001`）” |

第三轮复跑检查结果：FR 与 SC 的规格/任务引用集合完全一致（diff 为空），任务 34 条格式全部合规，无占位符、无未解释的 `MUST`、无残留 `待确认`。未发现新的 CRITICAL、HIGH 或 MEDIUM 级问题。
