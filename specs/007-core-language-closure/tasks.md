# 任务清单：Tops C++ 基础语言功能（错误提示、补全、悬停）

**输入**：来自 `specs/007-core-language-closure/` 的设计文档

**前提**：[plan.md](plan.md)、[spec.md](spec.md)、[research.md](research.md)、[data-model.md](data-model.md)、[contracts/](contracts/)

**测试说明**：规格 FR-033 到 FR-037 明确要求端到端验收，所以测试任务必须做。按规格澄清记录，本功能不为验收另写包内单元测试；现有包内测试作为回归基线继续运行，不删除。

**任务格式**：`[ID] [P?] [Story] 说明 + 文件路径`

- `[P]`：可以并行做（改的是不同文件、且不依赖尚未完成的任务）
- `[Story]`：属于哪个用户场景（US1–US4）

**路径约定**：Go 服务端代码在 `internal/`，端到端测试在 `integration/`，验收文件在 `testdata/lsp/fixtures/`。

**单一口径**：空结果的形状只在 T029 一处规定；其他任务遇到空结果时直接引用 T029，不重复描述，避免多处措辞漂移。

---

## 阶段 1：准备工作（共享基础设施）

**目的**：先把协议类型、关键字清单、验收文件准备好。这些都不含业务判断。

- [X] T001 在 `internal/protocol/errors.go` 增加错误码常量 `ContentModified ErrorCode = -32801`，放在 `RequestCancelled` 之后，注释写明“结果依赖的内容已变化，客户端可重试”。（FR-025a）
- [X] T002 [P] 新建 `internal/protocol/completion.go`，加入 `CompletionParams`（`textDocument`、`position`）、`CompletionItem`、`TextEdit` 三个结构；`CompletionItem` 字段固定为 `label`、`kind`、`detail`、`insertText`、`textEdit`、`filterText`、`sortText`、`data`；并定义补全种类常量：`CompletionItemKindText = 1`、`Method = 2`、`Function = 3`、`Field = 5`、`Variable = 6`、`Class = 7`、`Interface = 8`、`Module = 9`、`Property = 10`、`Enum = 13`、`Keyword = 14`、`Struct = 22`、`Constant = 21`。（FR-012、FR-030）
- [X] T003 [P] 新建 `internal/protocol/hover.go`，加入 `HoverParams`（`textDocument`、`position`）、`Hover`（`range`、`contents`）、`MarkupContent`（`kind`、`value`），并定义 `MarkupKindMarkdown = "markdown"`、`MarkupKindPlainText = "plaintext"`。（FR-019）
- [X] T004 [P] 修改 `internal/protocol/messages.go`，在 `InitializeParams` 增加嵌套结构解析客户端能力 `capabilities.textDocument.hover.contentFormat`（`[]string`）；同时新增 `Capabilities` 字段承载它，不改变已有字段的 JSON 名称。（FR-019）
- [X] T005 [P] 新建 `internal/parser/keywords.go`，导出两个函数 `CppKeywords() []string` 和 `TopsKeywords() []string`，返回已排序的副本；数据直接复用同包已有的 `keywordSpellings` 与 `topsKeywordSpellings`，不得复制词表内容。（FR-013）
- [X] T006 [P] 新建三个验收文件：`testdata/lsp/fixtures/valid-kernel.tops`（能正常分析的 Tops kernel，含 `__global__` 函数、`__shared__` 变量、结构体和局部变量，不超过 120 行）、`testdata/lsp/fixtures/std-cpp-error.tops`（含一处确定的语法错误和一处已核实的语义错误，例如缺分号与未声明名字）、`testdata/lsp/fixtures/tops-semantic-error.tops`（含一处已核实的 host/device 调用边界违规，不超过 80 行）。三个文件都用 UTF-8、LF 换行。（FR-033）

**检查点**：`go build ./...` 通过；三个验收文件能被现有解析器读取且不崩溃。

---

## 阶段 2：基础机制（阻塞所有用户场景）

**目的**：把六个请求共用的机制做好：分析完成通知、诊断去重排序、可见名字查询、能力声明。任何用户场景都依赖这一层。

**⚠️ 这一阶段不完成，后面的场景都做不了。**

- [X] T007 新建 `internal/server/analysis_wait.go`，实现三件事：① 按文档地址登记等待者并广播“分析结束”；② `analysisInProgress(uri string) bool`，判定条件为“该文档分析队列存在且 `pending != nil` 或 `running == true`”；③ `waitForAnalysis(ctx, uri) error`，用 2 秒上限等待，超时返回可识别为 `ContentModified` 的错误。等待方每次被唤醒后必须重新检查是否有新鲜结果，不缓存历史通知。（FR-014a、FR-023、FR-025a）
- [X] T008 在 `internal/server/analysis.go`、`internal/server/parse_context.go`、`internal/server/server.go` 的三处时机调用广播：分析提交结果后、分析被丢弃（过期/取消/失败）后、文档关闭或上下文整体失效后。广播必须发生在释放 `server.mu` 之后，避免死锁。（FR-008、FR-009、FR-025a）
- [X] T009 在 `internal/server/diagnostics.go` 增加去重与排序辅助函数：去重键为 `code + "\x00" + 起始偏移 + "\x00" + 结束偏移 + "\x00" + 严重级别 + "\x00" + 消息`；排序键依次为起始偏移、结束偏移、严重级别、编号、消息，全部升序；对同一 URI 单独处理。（FR-002、FR-003）
- [X] T010 修改 `internal/server/observability.go` 的 `publishAnalysisDiagnosticsWithSemantic`，在组装 `byURI` 之后、调用 `publishDiagnosticSets` 之前，对每个 URI 执行 T009 的去重与排序；不得改变“清理时发送空数组”的既有行为。（FR-002、FR-003）
- [X] T011 修改 `internal/semantic/diagnostics.go`、`internal/semantic/checker.go`、`internal/semantic/fixture.go`，把 `tops-semantic-context-missing` 与 `tops-semantic-context-degraded` 的严重级别由 `SeverityInformation` 改为 `SeverityWarning`，并同步 `fixture.go` 里期望级别字符串为 `Warning`；三处必须一致。（FR-005、FR-006）
- [X] T012 在 `internal/index/query.go` 新增 `VisibleSymbolsAt(uri string, offset int) []SymbolRecord`：找出包含光标的最内层作用域，沿 `ParentID` 向上收集，加上 `ScopeTranslationUnit` 与 `ScopeHeader` 的符号；排除 `ConditionalInactive` 与 `SymbolStatusStale`；结果按 `Name` 升序。不得修改已有函数的返回类型。（FR-013）
- [X] T013 修改 `internal/server/server.go`：在 `initialize` 结果里增加 `CompletionProvider`（空对象，不含 `triggerCharacters` 与 `resolveProvider`）和 `HoverProvider = true`；在 `handleRequest` 的 switch 中新增 `textDocument/completion` 与 `textDocument/hover` 两个分支，沿用 `navigationStateError` 的状态检查；不得注册 `diagnosticProvider`、`workspaceDiagnosticProvider`、`workspaceSymbolProvider`，也不实现诊断刷新（`workspace/diagnostic/refresh`）。（FR-011、FR-018、FR-024、FR-029）

**检查点**：`go build ./...`、`go vet ./...` 通过；`go test -p 128 -count=1 ./...` 中除新增用例外的既有测试全部通过（T011 影响的语义测试数据已同步）。

---

## 阶段 3：用户场景 1 — 可编辑 Tops kernel 文件的编辑闭环（P1）🎯 MVP

**目标**：打开 `valid-kernel.tops`，六种请求都能基于同一份最新结果给出正确结果，改一行后全部更新，且 2 秒内看到新提示。

**独立验证方式**：单独跑 `TestLSPProcessCoreLanguageValidKernel` 即可，不依赖其他场景的测试。

### 测试

- [X] T014 [P] [US1] 新建 `integration/lsp_core_language_test.go`，实现 `TestLSPProcessCoreLanguageValidKernel`：读取 `testdata/lsp/fixtures/valid-kernel.tops`，启动真实服务器进程，按顺序执行“打开并等诊断 → 请求补全 → 悬停 → 跳定义 → 查引用 → 列符号 → 改一行并升版本 → 再请求一次”；断言诊断无非预期 Error、补全含已声明名字、悬停含名字与种类且含 profile 与 pass、定义与引用指向正确位置、文档符号含函数与变量、第二次请求只反映新版本。**计时断言（SC-005）**：记录发送 `didChange` 的时间戳，轮询读取该 URI 的新诊断通知，断言从改完到收到新诊断不超过 2 秒。复用 `integration/lsp_process_test.go` 中已有的 `startServer`、`writeJSON`、`readTypedUntilID`、`readUntilDiagnosticsVersions`、`fileURI`、`sourceLSPPosition`、`fullTextChangeParams`。（FR-025、FR-026、FR-028、SC-002、SC-003、SC-005）

### 实现

- [X] T015 [US1] 新建 `internal/server/completion.go`，实现 `handleCompletion`：参数校验失败返回 `InvalidParams`；无新鲜结果时按 T007 判定处理（分析中返回单条提示项，否则返回空名单，形状按 T029）；有新鲜结果时调用 T012 与 T017 生成候选，按 [contracts/completion-candidates.md](contracts/completion-candidates.md) 第 5、6 节计算 `textEdit`、`filterText`、`sortText`，去重键为“名字 + 种类 + 来源地址”。（FR-012、FR-014、FR-014a、FR-017）
- [X] T016 [P] [US1] 新建 `internal/server/hover.go`，实现 `handleHover`：参数校验失败返回 `InvalidParams`；无新鲜结果且分析进行中时返回提示结果（`range` 规则见 [contracts/lsp-core-language.md](contracts/lsp-core-language.md) 4.5.1，含空文件与光标在偏移 0 的处理）；有新鲜结果时组装三段内容（名字与种类、类型或签名、来源与上下文状态），格式按初始化时协商的 `contentFormat`，缺失时用 Markdown。（FR-019、FR-021、FR-023）
- [X] T017 [US1] 新建 `internal/server/completion_rules.go`，实现 [contracts/completion-candidates.md](contracts/completion-candidates.md) 第 2、3 节的判断：光标落在注释、字符串、预处理指令行、`#include` 路径、条件状态未知区域时返回空名单；按语法规则表（含“声明开头”的前一 token 判定）决定允许的候选类别；点号或箭头之后返回空名单。**FR-016 的排除项必须逐项返回空名单且不报错**：成员访问、自动导入、代码片段、重载排序、文档渲染、跨仓符号搜索。（FR-013、FR-015、FR-016）
- [X] T018 [US1] 在 `internal/server/hover.go` 与 `internal/server/completion.go` 中保留来源信息，并严格区分载体：**补全项**可以放附加字段（LSP 的 `CompletionItem.data`），写入 `source`（`index` / `semantic` / `keyword` / `predefined-macro`）、`profileId`、`passKind`、`contextStatus`；**悬停结果**没有附加字段，来源信息必须写进可见文字（第三段），不得塞进自定义响应字段。上下文不是 `resolved` 时悬停追加降级说明行。（FR-020、FR-028、FR-030）

**检查点**：`go test -p 128 -count=1 ./integration -run TestLSPProcessCoreLanguageValidKernel -v` 通过。

---

## 阶段 4：用户场景 2 — 标准 C++ 错误的稳定提示（P1）

**目标**：打开 `std-cpp-error.tops` 看到位置准确、编号固定、级别为 Error 的提示；改对后提示消失；旧版本提示被丢弃；关闭文档后提示被清空；文件中仍有错误时补全、悬停、导航仍可用。

**独立验证方式**：单独跑 `TestLSPProcessCoreLanguageStandardCppErrors` 即可。

### 测试

- [X] T019 [P] [US2] 在 `integration/lsp_core_language_test.go` 增加 `TestLSPProcessCoreLanguageStandardCppErrors`：读取 `testdata/lsp/fixtures/std-cpp-error.tops`，断言诊断编号稳定、范围覆盖出错片段、级别为 Error、`version` 与 `contextVersion` 对应当前分析；把文件改成正确内容并升版本后，**先断言已收到该版本的非空诊断数组（证明分析结果已到），再断言收到同一版本的空数组（清理完成）**，只看到空数组不视为通过；在错误版本上请求补全、悬停、定义，断言仍能解析的名字有结果、无法解析的位置返回空结果且不报非标准错误。（FR-007、FR-025、FR-034、FR-035、FR-036、SC-002）
- [X] T020 [P] [US2] 在 `integration/lsp_core_language_test.go` 增加 `TestLSPProcessCoreLanguageStaleAndCloseCleanup`：① 连续发送两次 `didChange`，断言只发布最终版本的诊断，不出现旧版本诊断；② 发送 `didClose` 后，断言收到该 URI 的空诊断数组，且后续请求按“文档未打开”返回空结果。头文件变化引发的旧结果作废与上条使用同一套过期丢弃路径，本任务不单独构造头文件场景；现有 `TestLSPProcessNavigationConditionBuiltinAndHeaderChangeMatrix` 已覆盖头文件变化下的导航行为，本功能不改变它。（FR-008、FR-009）

### 实现

- [X] T021 [US2] 核对并补齐 `internal/server/observability.go` 的发布路径：语法诊断与语义诊断必须进入同一份诊断集合，语法诊断携带 `data.documentVersion` 与 `data.contextVersion`，语义诊断额外携带 `profileId`、`passKind`、`semanticKind`；两条路径都不得丢掉 `source = "tops-lsp"`。（FR-001、FR-002、FR-004、FR-026）
- [X] T022 [US2] 核对并补齐 `internal/server/diagnostics.go` 与 `internal/server/observability.go` 的清理与丢弃路径：新版本无问题时对当前文档发空数组并带当前文档版本；过期分析结果一律丢弃、不发布；`didClose` 与上下文整体失效时清空受影响 URI。清理通知不得被当作“分析完成”的判据（已有测试踩过这个坑），在代码注释里写明这一点。（FR-007、FR-008、FR-009、FR-026）

**检查点**：`go test -p 128 -count=1 ./integration -run 'TestLSPProcessCoreLanguageStandardCppErrors|TestLSPProcessCoreLanguageStaleAndCloseCleanup' -v` 通过。

---

## 阶段 5：用户场景 3 — Tops 扩展规则错误只报有把握的问题（P1）

**目标**：打开 `tops-semantic-error.tops`，在上下文齐全时报 `tops-semantic-*` Error；上下文不完整时只报 Warning 级降级提示。

**独立验证方式**：单独跑 `TestLSPProcessCoreLanguageTopsSemanticErrors` 即可。

### 测试

- [X] T023 [P] [US3] 在 `integration/lsp_core_language_test.go` 增加 `TestLSPProcessCoreLanguageTopsSemanticErrors`：用环境变量提供齐全的目标上下文（`TOPS_LSP_TARGET_PROFILE`、`TOPS_LSP_PASS_KIND`、`TOPS_LSP_PREDEFINED_MACROS`），打开 `tops-semantic-error.tops`，断言收到 `tops-semantic-host-device-boundary`（或该文件实际违反的 `tops-semantic-*` 编号）的 Error，且 `data.profileId`、`data.passKind`、`data.contextStatus` 正确；再用不提供目标上下文的方式启动一次，断言不出现 Error，只有 `tops-semantic-context-degraded` 且级别为 Warning。（FR-005、FR-006、FR-010、FR-034、FR-035、SC-002）

### 实现

- [X] T024 [US3] 核对 `internal/server/diagnostics.go` 的 `semanticDiagnosticsForState`：降级诊断（`tops-semantic-context-missing`、`tops-semantic-context-degraded`）在 `PassKind = "not-applicable"` 时也能发布，级别取自 T011；确定性 Error 在缺少范围或上下文不是 `resolved` 时不得发布。（FR-005、FR-010、FR-006）
- [X] T025 [US3] 在 `internal/server/hover.go` 与 `internal/server/completion.go` 中区分标准 C++ 与 Tops 扩展来源：Tops 符号写明 profile 与 pass；上下文为 `partial`、`missing`、`invalid` 时标注降级，不得给出依赖缺失事实的确定结论。（FR-020、FR-021、FR-024）

**检查点**：`go test -p 128 -count=1 ./integration -run TestLSPProcessCoreLanguageTopsSemanticErrors -v` 通过。

---

## 阶段 6：用户场景 4 — 所有请求行为规则一致（P2）

**目标**：六个方法的参数错误、未打开文档、取消、过期、未实现方法行为一致；分析进行中时补全与悬停立即给提示、导航最多等 2 秒；版本竞争与取消重复 100 次不出现旧结果。

**独立验证方式**：单独跑 `TestLSPProcessCoreLanguageRequestContract` 或 `TestLSPProcessCoreLanguageAnalyzingState` 即可。

### 测试

- [X] T026 [P] [US4] 在 `integration/lsp_core_language_test.go` 增加 `TestLSPProcessCoreLanguageRequestContract`，按方法列出实际适用的用例（不适用项显式跳过，不笼统写“六个方法各五类”）：
  - `textDocument/definition`、`textDocument/references`、`textDocument/completion`、`textDocument/hover`：参数非法（缺 `uri`、位置越界）→ `InvalidParams`；未打开文档 → 按 T029 的空结果形状；取消 → `RequestCancelled`。
  - `textDocument/documentSymbol`：没有位置参数，只测“缺 `uri` → `InvalidParams`”“未打开文档 → `[]`”“取消 → `RequestCancelled`”。
  - 六个方法共同：请求 `textDocument/diagnostic`、`workspace/symbol`、`completionItem/resolve` → `MethodNotFound`。
  - **简单客户端场景（SC-007）**：初始化参数里**不带** `capabilities.textDocument.hover.contentFormat`，断言初始化结果仍只包含约定能力，且悬停内容按 Markdown 返回，不报错。
  - 导航超时：启动时设置 `TOPS_LSP_NAVIGATION_BLOCK=1`，断言跳定义在 2 秒后返回 `ContentModified`，而不是无限等待。
  - **版本竞争与取消的重复验证（SC-004）**：对“改版本同时发请求”和“请求中途取消”各循环 100 次，统计返回旧版本结果的次数与发出旧版本诊断的次数，断言两者均为 0。（FR-025a、FR-027、FR-031、SC-001、SC-004、SC-006、SC-007）
- [X] T027 [P] [US4] 在 `integration/lsp_core_language_test.go` 增加 `TestLSPProcessCoreLanguageAnalyzingState`：打开文件后**立即**发送请求（不等待任何诊断），断言悬停返回“正在分析中”提示、补全只返回一条同类提示项且不含真实候选、两者都不报错且不阻塞；同时断言同一时刻跳定义、查引用、列符号要么在 2 秒内返回结果，要么返回 `ContentModified`。（FR-014a、FR-023、FR-025a、SC-009）

### 实现

- [X] T028 [US4] 修改 `internal/server/navigation.go`：`handleDefinition`、`handleReferences`、`handleDocumentSymbol` 在没有新鲜结果且分析进行中时调用 T007 的等待函数；等待超时返回 `ContentModified`；已有旧结果直接按空结果返回，不等待；保留现有 `navigationGate` 与取消检查的顺序，确保已有取消测试仍通过。**同时核对 FR-028**：导航结果仍携带来源记录，受目标配置影响时写明 profile 与 pass；本功能不改变该行为的语义，只在结果组装处确认未被遗漏。（FR-025、FR-025a、FR-027、FR-028）
- [X] T029 [US4] 统一空结果形状（**唯一出口**）：`internal/server/completion.go` 返回 `[]`、`internal/server/hover.go` 返回 `null`、`internal/server/navigation.go` 中定义与悬停返回 `null`、引用与文档符号返回 `[]`；未打开文档、光标落在不产生候选的区域、解析不出名字、只有已失效旧结果四种情况都按此处理，不返回错误。（FR-015、FR-017、FR-022、FR-023、FR-025、FR-027）
- [X] T030 [US4] 在 `internal/server/observability.go` 补齐日志分类：请求结果分“成功 / 空结果 / 参数错误 / 取消 / 过期丢弃 / 导航超时”，记录文档标识、版本、编号，禁止记录源码与凭据。（FR-032）

**检查点**：`go test -p 128 -count=1 ./integration -run 'TestLSPProcessCoreLanguageRequestContract|TestLSPProcessCoreLanguageAnalyzingState' -v` 通过。

---

## 阶段 7：收尾与横切事项

- [X] T031 运行完整检查并把结果记入 `specs/007-core-language-closure/quickstart.md`：`go build ./...`、`go vet ./...`、`go test -p 128 -count=1 ./...`。任何失败都要定位并修好，不得靠调整断言绕过。**另确认环境独立性（FR-037）**：在未安装 GCU 驱动、未配置 `TOPS_LSP_CLANG`、无金丝雀硬件的机器上重跑一遍集成测试并记录结果。
- [X] T032 [P] 检查日志隐私与文字语言：在 `internal/` 全量搜索日志调用与面向用户的字符串，确认没有把源码文本、用户主目录路径或凭据写入日志，且 stale 丢弃与请求失败均可通过日志分类定位；确认诊断消息、悬停内容、补全详情、日志与错误信息全部为英文、无中英混排（FR-004、FR-032、SC-008）。
- [X] T033 [P] 按 [quickstart.md](quickstart.md) 逐项走一遍，把实际结果补记在 `quickstart.md` 末尾的“验证记录”小节（若该小节不存在则新增）。
- [X] T034 [P] 检查 [contracts/](contracts/) 三份契约与实际实现是否一致（能力声明字段、诊断去重排序键、补全规则表与“声明开头”判定、悬停分析中范围规则、2 秒超时、`ContentModified`）；不一致时改实现或改契约，并在 `plan.md` 记录原因。

---

## 依赖关系与执行顺序

### 阶段依赖

- **阶段 1 准备工作**：无依赖，可立即开始。
- **阶段 2 基础机制**：依赖阶段 1 完成，阻塞所有用户场景。
- **阶段 3 到阶段 6**：都依赖阶段 2 完成；彼此之间没有硬依赖，可并行分工。
- **阶段 7 收尾**：依赖所有要交付的场景完成。

### 场景之间

- **US1（P1）**：依赖阶段 2；不依赖其他场景。
- **US2（P1）**：依赖阶段 2；与 US1 无依赖，但共用诊断发布路径（T021、T022 改动 `internal/server/observability.go`，与 T015/T016 不在同一文件）。
- **US3（P1）**：依赖阶段 2，特别是 T011（降级级别）；不依赖 US1、US2。
- **US4（P2）**：依赖阶段 2，特别是 T007、T008（等待与广播）；其测试会用到 US1、US2、US3 建立的文件，但判定逻辑独立。T028、T029 修改 `internal/server/navigation.go`，与 T015 的 `completion.go` 不冲突。

### 每个场景内部

- 测试先写，运行后应失败（红），实现后转绿。
- 协议类型（阶段 1）先于服务端处理函数。
- 查询能力（T012）先于补全实现（T015）。

---

## 并行执行示例

### 阶段 1

```text
T002 补全协议类型      internal/protocol/completion.go
T003 悬停协议类型      internal/protocol/hover.go
T005 关键字导出        internal/parser/keywords.go
T006 三个验收文件      testdata/lsp/fixtures/*.tops
```

### 阶段 3（US1）

```text
T014 端到端测试        integration/lsp_core_language_test.go
T016 悬停实现          internal/server/hover.go
```

T015 与 T017 都涉及补全，按 T017 → T015 的顺序做。

### 阶段 4 到阶段 6 的测试

```text
T019 标准 C++ 错误用例
T020 过期与关闭清理用例
T023 Tops 规则错误用例
T026 请求契约与 100 次重复用例
T027 分析进行中用例
```

五个用例都在 `integration/lsp_core_language_test.go` 中新增顶层测试函数，可以同时编写；若同时编辑同一文件，按函数分段，避免冲突。

---

## 实施策略

### 先做最小可用版本

1. 完成阶段 1、阶段 2。
2. 完成阶段 3（US1）。
3. **停下来验证**：单独跑 `TestLSPProcessCoreLanguageValidKernel`。
4. 此时已经有了可用的诊断、补全、悬停、导航闭环。

### 增量交付

1. 加 US2（含过期与关闭清理）→ 验证标准 C++ 错误 → 可用。
2. 加 US3 → 验证 Tops 规则错误 → 可用。
3. 加 US4 → 验证六种请求的失败行为与分析进行中行为 → 可用。
4. 每个阶段都不破坏前面阶段的测试。

### 分工建议

阶段 2 优先由一人完成，避免多人同时改 `internal/server/` 下的公共文件。阶段 3 到阶段 6 可以按场景分给不同人，但 `integration/lsp_core_language_test.go` 需要按测试函数分段编辑。

---

## 说明

- `[P]` 表示可以并行，但前提是改的文件不同、且不依赖尚未完成的任务。
- 每个场景都能独立完成并独立测试。
- 每个任务都写了具体文件路径、判定标准和对应需求编号；遇到需要判断的地方，以 [contracts/](contracts/) 为准。
- 每条需求（FR-001 到 FR-037，含带字母后缀的 FR-014a、FR-025a）与每条成功标准（SC-001 到 SC-009）都能在本文档找到至少一个任务。
- 提交按任务或逻辑分组进行，不要攒到最后一次提交。
- 阶段 2 的诊断级别调整（T011）会让部分既有语义测试失败，这是预期内的对外可见变化，必须同步改测试数据，不能改回 Information。

---

## 阶段 8：Convergence

- [X] T035 在 `internal/server/completion.go` 与 `internal/server/hover.go` 生成结果之后、返回响应之前，再次确认文档版本与上下文版本仍与所用快照一致；不一致时补全返回 `[]`、悬停返回 `null`，并记录一次过期丢弃日志，不得返回基于旧快照的结果。（FR-014、FR-026、contracts/lsp-core-language.md 3.4，partial）
- [X] T036 修正 `internal/server/hover.go` 的 `analyzingHover` 边界规则：光标偏移为 0 且文件非空时返回选中第一个字符的 `[0,1)` 范围；文件末尾与空文件规则保持契约 4.5.1 不变。（FR-023、contracts/lsp-core-language.md 4.5.1，contradicts）
- [X] T037 在 `internal/server/completion.go` 与 `internal/server/hover.go` 的“存在已失效旧结果”分支使用 `stale` 结果分类记录日志，并保留文档标识与版本，使过期丢弃与请求失败都能靠日志分类定位。（FR-032、SC-008、contracts/lsp-core-language.md 3.4，partial）
- [X] T038 在 `internal/server/completion.go` 中为关键字候选（标准 C++ 与 Tops）补齐 `data` 的 `source`、`profileId`、`passKind`、`contextStatus`，与符号候选、预定义宏候选保持同一来源字段约定。（FR-028、data-model.md 第 4 节、T018，partial）
- [X] T039 在 `integration/lsp_core_language_test.go` 补充验收断言：`tops-semantic-error.tops` 在 `resolved` 上下文下所有 Error 必须是预期的 `tops-semantic-*` 且无无关 Error，并验证补全、悬停、跳定义有结果；`std-cpp-error.tops` 增加查引用与列文档符号断言；按 SC-003 核对可见声明名字进入补全、可解析位置的悬停返回名字与种类。（FR-034、FR-035、SC-003，partial）

---

## 阶段 9：Convergence

- [X] T040 在 `internal/server/hover.go` 的悬停内容中，当符号的条件状态为 `ConditionalUnknown` 时追加一条明确的降级说明（说明条件编译不可判定），不得只显示 `Context: resolved`；对应的可见文字保持英文。（FR-021、data-model.md 第 7 节，partial）
- [X] T041 把 `integration/lsp_core_language_test.go` 中 `TestLSPProcessCoreLanguageValidKernel` 的改版后新诊断耗时断言从 3 秒收紧为 2 秒，与 SC-005 一致。（SC-005、T014，partial）

---

## 阶段 10：Convergence

- [X] T042 在 `internal/server/observability.go` 的 `observeNavigation` 与 `observeNavigationError` 日志中补充 `profile_id` 与 `pass_kind`（取所用的索引快照），使导航结果在受目标配置影响时能写明 profile 与 pass；快照缺失时记录空值。（FR-028、T028，partial）
- [X] T043 把 `integration/lsp_core_language_test.go` 中 `TestLSPProcessCoreLanguageRequestContract` 的取消用例改为循环 100 次：每次发送导航请求后立即取消，断言每次都返回 `RequestCancelled`，且不返回旧版本结果。（T026、SC-004，partial）

---

## 阶段 11：Convergence

- [X] T044 在 `testdata/lsp/fixtures/valid-kernel.tops` 末尾增加一行注释和一个含字符串字面量的合法声明；在 `TestLSPProcessCoreLanguageValidKernel` 中断言光标落在注释内、字符串内、以及声明之间空白处时，补全返回空名单、悬停返回空值且不报错。（FR-015、FR-022、FR-034，partial）

---

## 阶段 12：Convergence

- [X] T045 在 `TestLSPProcessCoreLanguageStandardCppErrors` 中断言初版文件的所有 Error 诊断只能属于 `tops-syntax-*` 语法类或 `tops-semantic-unknown-name`，不得混入其他 Error，补齐 FR-035 对第二个错误文件的检查。（FR-035、SC-002，partial）
