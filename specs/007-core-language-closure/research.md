# 阶段 0 研究记录

**功能**：Tops C++ 基础语言功能（错误提示、补全、悬停）
**日期**：2026-10-19

这份记录只写已经核对过代码的事实，以及需要在这些事实上做的选择。核对过的文件在每条里写明。

---

## 1. 导航请求等不到分析时怎么办

**问题**：跳定义、查引用、列符号在“文件刚打开、还没有任何分析结果”时，现在会怎样？

**核对到的事实**：

- `internal/server/navigation.go` 的 `navigationInput` 在找不到当前快照时返回空快照，`handleDefinition` 看到 `snapshot.RootURI == ""` 就直接返回成功且结果为空。
- 真正的等待来自 `waitNavigation`，它只在 `server.navigationGate` 不为空时才停住。`cmd/tops-lsp/main.go` 只在环境变量 `TOPS_LSP_NAVIGATION_BLOCK=1` 时设置这个门，属于测试开关；正常运行不会等待。

**决定**：新增一个“分析完成通知”机制，按文档地址广播；导航请求先看有没有新鲜快照，没有就等通知，最多 2 秒；超时返回 `ContentModified`。

**理由**：
- 轮询既浪费 CPU 又难保证 2 秒精度；广播通知在分析结束、文档变更、上下文失效三个点触发即可。
- 超时返回错误而不是空结果，与规格一致：用户能知道“稍后重试”，客户端也能自动重试。

**考虑过的替代方案**：
- 继续无限等待：会让请求挂住，违反规格。
- 直接返回空结果：用户在刚打开文件时会看到“没有定义”，误判成代码没写对。

**注意**：现有 `TOPS_LSP_NAVIGATION_BLOCK` 门控保留，放在新等待逻辑之前；已有取消测试依赖它。

---

## 2. 怎么判断“分析进行中”

**问题**：补全和悬停需要区分“还没有结果但正在分析”和“分析了但没结果”。

**核对到的事实**：
- `internal/server/analysis.go` 里每个文档有一个 `analysisQueue`，字段 `pending` 和 `running` 能表示“还有排队的任务”或“正在跑”。
- `enqueueAnalysis` 在新任务到来时会丢掉旧的 `pending` 并记录过期观测。

**决定**：新增 `analysisInProgress(uri) bool`，判定条件为“该文档的队列存在且 `pending != nil` 或 `running == true`”，或者“当前没有新鲜快照且分析尚未因失败停止”。分析结束时广播通知，供等待方和判断方复用。

**理由**：复用已有队列状态，不新增生命周期；避免凭空猜测“正在分析”。

**考虑过的替代方案**：
- 用时间阈值猜（例如刚变更 500 毫秒内算进行中）：不准确，容易在慢机器上误判。

---

## 3. 悬停内容的格式

**问题**：客户端可能只支持纯文本，也可能支持 Markdown。

**核对到的事实**：
- `internal/protocol/messages.go` 的 `InitializeParams` 目前只解析 `processId`、`clientInfo`、`rootUri`、`workspaceFolders`，没有客户端能力字段。

**决定**：在 `InitializeParams` 增加 `capabilities.textDocument.hover.contentFormat`（字符串数组）。只要列表里出现 `markdown` 就用 Markdown，否则用 `plaintext`；字段缺失时按 Markdown 处理。

**理由**：
- 这是项目显式选择：现代编辑器基本都渲染 Markdown，缺失该字段通常表示客户端不关心格式。
- 该选择不影响纯文本客户端：它们会显式声明 `plaintext`。

**考虑过的替代方案**：
- 一律纯文本：丢失签名和代码块的可读性。
- 一律 Markdown：对显式只支持纯文本的客户端会显示原始符号。

---

## 4. 诊断去重与排序

**问题**：四类诊断合并后可能重复，顺序也不固定。

**核对到的事实**：
- `internal/server/observability.go` 的 `publishAnalysisDiagnosticsWithSemantic` 依次追加索引诊断、语法诊断、语义诊断，然后用 `publishDiagnosticSets` 按 URI 排序发布；单个 URI 内部的诊断顺序取决于追加顺序。
- 四个阶段可能对同一位置报同一问题，例如索引降级的 `analysis-degraded` 与语义降级。

**决定**：
- 去重键：`编号 + 起始偏移 + 结束偏移 + 严重级别 + 消息`。
- 排序：起始偏移、结束偏移、严重级别（Error < Warning < Information < Hint）、编号、消息，全部升序。
- 去重与排序在 `publishDiagnosticSets` 之前完成，对每个 URI 分别处理。

**理由**：位置优先符合用户从上往下阅读的顺序；用消息参与去重键可以避免把“同一位置同一编号但不同原因”的诊断错误合并。

**考虑过的替代方案**：
- 只按编号去重：会误删同编号不同位置的问题。
- 保留原追加顺序：相同输入在不同分析路径下顺序不稳定，测试无法断言。

---

## 5. 宏候选从哪里来

**问题**：补全要给“已核实的宏”，数据源在哪？

**核对到的事实**：
- `internal/index/builder.go` 的 `symbolInfo` 对 `NodePreprocessorDirective` 调用 `macroName`，把宏注册为 `SymbolKindMacro` 符号。
- `internal/index/query.go` 的 `documentSymbolsForIDs` 会把宏列入文档符号；`internal/server/navigation.go` 把宏映射为 `protocol.SymbolKindConstant`。
- `internal/index/model.go` 的 `IndexSnapshot.PredefinedMacros` 保存编译上下文提供的宏。

**决定**：宏候选直接复用已索引的 `SymbolKindMacro` 符号和 `PredefinedMacros`，不新增独立宏表。

**理由**：宏已经是符号体系的一部分，新增平行结构会造成两份真相，且违反“不重复实现语义判断”。

**考虑过的替代方案**：新建独立宏清单。已否决：会与现有符号索引产生不一致。

**约束**：只有条件状态为“有效”的宏进入候选；条件未知时按降级处理（可以出现在候选里，但悬停要标明降级）。

---

## 6. 降级提示的严重级别

**问题**：规格要求 `tops-semantic-context-degraded` 用 Warning，现状是什么？

**核对到的事实**：
- `internal/semantic/diagnostics.go` 的规格表把 `tops-semantic-context-degraded` 定为 `SeverityInformation`。
- `internal/semantic/checker.go` 的 `contextStatusCode` 返回 `SeverityInformation`。
- `internal/semantic/fixture.go` 的期望级别映射里写的是 `Information`。
- 该诊断的范围由 `contextStatusDiagnostic` 给出：源文件为空时是空范围，否则是 `0..1`。

**决定**：三处同步改为 Warning；范围保持现状（`0..1` 或空）。空范围时按规格不发布确定性 Error，但这条本身是 Warning，允许在无范围时发布。

**理由**：规格澄清记录已定；保持范围不变可以避免触碰已有测试的比较逻辑。

**影响**：已有语义测试数据中的期望级别字符串需要同步更新，否则测试会失败。这是计划中明确列出的改动项。

---

## 7. 补全候选怎么算

**问题**：规格要求“由语法规则和语义可见性共同决定”，需要可执行的规则。

**核对到的事实**：
- `internal/index/builder.go` 有 `scopeIsVisible` 和 `scopeAt`，但都是包内私有函数。
- `internal/index/model.go` 的 `ScopeRecord` 带 `Range`、`ParentID`、`Symbols`、`Children`；`SymbolRecord` 带 `ScopeID`、`NameRange`、`Kind`、`Status`、`Condition`。
- `internal/parser/tokenizer.go` 有 `keywordSpellings` 和 `topsKeywordSpellings`，都是包内私有变量。
- `internal/parser/ast.go` 的 `NodePreprocessorDirective` 有 `DirectiveRange`，可用于判断光标是否在预处理指令行。

**决定**：
- 在 `internal/parser` 新增导出的关键字清单函数（返回排序后的字符串列表），不复制词表。
- 在 `internal/index` 新增按位置查询可见名字的方法：找出包含光标的最内层作用域，沿 `ParentID` 向上收集可见符号，加上翻译单元级和头文件级符号。
- 在 `internal/server` 新增语法规则表，判断光标处允许哪类名字（见 `contracts/completion-candidates.md`）。

**理由**：词表和可见性规则都是已有语义事实，导出比复制安全；规则表把“语法允许”变成可测试的显式条目。

**考虑过的替代方案**：
- 在 server 里重写一份作用域遍历：会与索引的可见性规则漂移。
- 不区分位置，全量列出所有符号：违反规格，且用户会被无关候选淹没。

---

## 8. “正在分析中”这条补全提示项怎么表示

**问题**：规格要求补全名单里只放一条提示项，且不能插入代码。

**决定**：
- `label` 和 `detail` 用英文说明正在分析。
- `kind` 用标准补全种类里的 `Text`（数值 1）。
- 不提供 `insertText`；`textEdit` 使用光标处的空范围、新文本为空字符串；`filterText` 使用当前正在输入的前缀，保证用户继续输入时它仍可见。
- 该提示项的 `sortText` 用 `000`，排在真实候选之前；但该场景下不会同时返回真实候选。

**理由**：空替换范围保证不改变文件内容；`filterText` 保证它不会被编辑器悄悄过滤掉。

**考虑过的替代方案**：
- 返回空名单：用户不知道是在分析还是没结果（规格已明确要求提示）。
- 用一条假符号充当提示：可能被误插入。

---

## 9. 悬停内容怎么组织

**决定**：
- 内容分三段：第一行是名字和种类；第二段是已知类型或函数签名；第三段是来源信息（声明位置、profile、pass、上下文状态）。
- Markdown 模式下用代码围栏包住签名；纯文本模式下用普通行。
- 上下文不是 `resolved` 时追加一行降级说明。
- 光标在注释、字符串、空白或解析不出名字时返回空结果。

**理由**：三段结构在两种格式下都能读，也便于测试断言关键片段。

**考虑过的替代方案**：
- 只返回签名一行：缺少来源和降级信息，违反规格。
- 返回完整文档注释：规格明确排除。

---

## 10. 三个测试文件怎么驱动

**决定**：
- 三个 `.tops` 文件放在 `testdata/lsp/fixtures/`，测试用 `repositoryRoot` 拼绝对路径，再转成 `file://` 地址。
- 复用 `integration/lsp_process_test.go` 的 `startServer`、`writeJSON`、`readTypedUntilID`、`readUntilDiagnosticsVersions`、`fileURI`、`sourceLSPPosition`。
- 新增 `integration/lsp_core_language_test.go`，每个文件一个顶层测试，测试内按“打开 → 等诊断 → 请求六种能力 → 改版本 → 再请求”的顺序执行。
- 期望值写在测试代码里，文件名、诊断编号、位置偏移在测试中显式列出，不额外引入期望结果文件。

**理由**：端到端用例本来就要读源码计算偏移；再维护一份外部期望文件容易出现两份真相。

**考虑过的替代方案**：外部 JSON 期望文件。已否决：本次只有三个文件，内联更直接，也符合规格“测试代码读取文件内容”的要求。

---

## 11. 超时与性能怎么验证

**决定**：
- 导航超时用 2 秒固定值，测试中构造“分析被刻意拖住”的场景，断言收到 `ContentModified`。
- “改完 2 秒内看到新诊断”用轮询读取诊断通知，超过 2 秒判定失败。
- 不引入百分位统计；性能指标留到后续性能阶段。

**理由**：固定阈值可复现；百分位需要固定硬件和冷热状态，属于后续阶段范围。

---

## 12. 未决问题

无。规格中的澄清记录已覆盖本轮决策；剩余细节属于任务拆分，不阻塞设计。
