# 实施计划：P1.3 Tops C++ AST 和符号索引

**分支**：`005-ast-symbol-index` | **日期**：2026-10-10 | **规格说明**：[spec.md](spec.md)

**输入**：来自 `specs/005-ast-symbol-index/spec.md` 的功能规格说明。

## 摘要

本功能作为 P1.3 在现有可恢复 parser、文档版本管理和 latest-only 分析队列之上，增加进程内 AST/符号索引。索引覆盖函数、类型、属性、内建变量、namespace、field、macro、可加载 header、条件分支、声明/定义和引用关系；由 Go server 保存最新快照并通过标准 LSP 提供 definition、references 和 document symbols。P1.3 只提供索引和导航基线，不拥有完整 C++ semantic；目标合法性由后续 semantic/context 阶段负责。

主要技术决定：

- 新建 `internal/index`，使用版本化、可替换的 `IndexSnapshot`，不引入数据库或第三方依赖。
- `internal/parser` 只补充名称范围和声明角色等最小 AST 元数据；scope、qualified name、symbol ID、引用候选和 header 关系由 index 计算。
- header 通过可注入的 `SourceProvider` 按需读取：优先已打开文档，其次是 `ParseContext.IncludeRoots` 中被引用且可读的 header；不扫描整个 workspace。
- 复用现有 `analysisQueue`、document snapshot、generation、epoch 和 context version；构建在锁外执行，版本匹配后原子提交，旧结果直接丢弃。
- `active` 候选优先；多个 active 候选全部返回；没有 active 候选时才使用 `unknown`；`inactive` 不进入 definition/references，documentSymbol 展示 active 和 unknown。
- protocol 只增加标准 LSP 参数、结果和 capability，不增加私有语义 response 字段；降级状态通过 index 状态、diagnostics 和脱敏日志表达。

研究、数据模型、契约和验证入口见：[research.md](research.md)、[data-model.md](data-model.md)、[contracts/](contracts/)、[quickstart.md](quickstart.md)。

## 技术上下文

**语言/版本**：Go `1.26.8`，由当前 `go.mod` 的 `go 1.26.8` 确定。

**主要依赖**：Go 标准库和仓库现有 `internal/document`、`internal/parser`、`internal/position`、`internal/protocol`、`internal/server`、`internal/transport`；不新增第三方 parser、LSP、数据库或 Clang runtime 依赖。

**存储**：进程内 `internal/index.Store` 保存当前 document/header 关系的不可变快照；`internal/document.Store` 继续只保存文档文本和版本；不使用持久化数据库。

**测试**：`go test ./...`、`go test -race ./...`、`go vet ./...`；新增 index unit、protocol JSON contract、server navigation/stale 和真实 stdio integration 测试。Clang 对照如运行，只是独立验证，不是测试前置或 server runtime 依赖。

**目标平台**：Linux amd64 Go language server，使用现有 LSP over stdio transport；测试不需要 GCU 设备、GCUSIM、测试机 Docker、网络或 clangd。

**项目类型**：Go command-line language server 与内部 parser/index library。

**性能目标**：本功能不承诺未经测量的延迟或吞吐数字。工程门槛是：不扫描整个 workspace；header 只按需加载；每个 URI 仍由 latest-only 队列处理；查询不使用过期快照；构建、查询和失败路径不能无限等待或 panic。后续若需要公开性能指标，另建验证任务并以实测结果为准。

**约束**：

- parser 输入必须是文本和 `ParseContext`，不得解析 raw compiler argv、response file 或启动 `topscc`/Clang/clangd。
- Go server 拥有 AST/index、候选查询、版本失效和 LSP 行为；TypeScript client 不参与语义判断。
- header 来源只包括已打开文档和配置 include roots 中实际被引用且可读取的文件；`SourceProvider.IncludeRoots()` 是 server 构造 `ParseContext` roots/available 状态的唯一输入；缺失 header 必须保留 unresolved/degraded 状态。
- `active`/`inactive`/`unknown` 条件状态不得被目标名称或相邻架构默认值替代。
- `internal/position` 是 byte range 到 LSP UTF-16 range 的统一转换入口；所有导航范围使用半开区间。
- 不实现完整类型检查、重载决议、模板实例化、完整宏展开、链接或设备执行语义。
- 日志只记录脱敏 URI、operation、symbol kind、版本、状态、condition、结果数量、耗时、diagnostic code 和 error kind，不记录完整源码、凭据或完整 LSP payload；`analysis_degraded` 和 `stale_result` 的字段由 contract 固定。

**规模/范围**：当前打开文档、已打开 workspace 文件和按需加载的被引用 header；不全量索引未打开、未配置或未被引用的 workspace 文件。索引范围以版本正确性和导航契约为主，不在本阶段设定固定文件数或源码行数上限。

## 宪法检查

*门禁：阶段 0 研究前检查，阶段 1 设计完成后复查。以下两次检查均通过。*

- [x] **LSP 契约优先**： [contracts/lsp-navigation.md](contracts/lsp-navigation.md) 固定 capability、请求参数、标准结果、空结果、错误、取消、过期和兼容性；[contracts/index-lifecycle.md](contracts/index-lifecycle.md) 固定 server/index/parser 边界。
- [x] **Go/TypeScript 分层边界**：Go server 负责 parser/index、版本和 LSP 编码；TypeScript client 只负责请求转发和结果展示；没有客户端语义实现。
- [x] **Tops C/C++ 语义保真**：标准 C++、Tops spelling、条件状态和 target/pass context 分开记录；不从相邻架构或 profile 名称推断宏值；完整 semantic 不在本功能内冒充完成。
- [x] **行为优先的验证**：计划包含 parser metadata、index、header、protocol、server、stdio integration 和 stale/错误路径测试，覆盖正常、不完整、缺失 header、条件分支、歧义和版本竞争。
- [x] **可观测且兼容地演进**：降级、缺失、歧义、取消和 stale 行为有明确记录；日志字段脱敏；新增能力使用标准可选 capability，不改变既有生命周期和 framing。
- [x] **没有未解释的违规项**：不新增第三方依赖、持久化数据库、runtime Clang、私有 LSP payload 或 workspace 全量扫描；复杂度记录说明新增 `internal/index` 和 `SourceProvider` 的边界理由。

## 项目结构

### 文档（本功能）

```text
specs/005-ast-symbol-index/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── lsp-navigation.md
│   └── index-lifecycle.md
├── checklists/requirements.md
└── tasks.md                 # 由 /speckit.tasks 生成
```

### 源代码（实现阶段）

```text
internal/parser/
├── ast.go                    # NameRange、DeclarationRole、BodyRange 等最小元数据
├── parser.go                 # 生成声明名称和角色元数据
├── construct_helpers.go       # 声明/函数结构辅助信息
└── *_test.go                  # AST 元数据和 incomplete/recovery 回归

internal/index/                # 新增
├── model.go                   # Snapshot、Symbol、Reference、Scope、Dependency
├── builder.go                 # ParseResult -> IndexSnapshot
├── query.go                   # definition/references/document symbols 候选查询
├── store.go                   # 版本检查、替换、删除和反向依赖失效
├── source.go                  # SourceProvider/HeaderResolver 边界
├── manifest.go                # fixture manifest 读取、计数和验收报告
└── *_test.go                  # scope、ID、条件、header、版本和并发测试

cmd/tops-index-check/
└── main.go                    # 执行 manifest validator 并输出分类统计

internal/protocol/
├── messages.go                # ServerCapabilities 增加导航 capability
├── navigation.go              # 标准导航 request/result/SymbolKind 模型
└── navigation_test.go         # JSON 参数、结果和能力声明

internal/server/
├── analysis.go                # 分析任务生成 index snapshot 并提交
├── parse_context.go           # 复用 document/context/generation/epoch 校验
├── navigation.go              # 三项 LSP request handler
├── index_analysis.go          # parser/index 构建和当前快照提交
├── observability.go            # analysis_degraded/stale_result 映射和日志字段
└── *_test.go                  # handler、stale、cancel、close、degraded

testdata/index/                # 新增
├── single-file/
├── multi-file/
├── headers/
├── conditions/
├── incomplete/
├── missing-context/
├── stale/
├── expected/
└── manifest.json

integration/
└── lsp_process_test.go        # stdio initialize、capabilities 和三项导航请求
```

**结构决策**：`internal/index` 隔离符号模型、header 图和查询规则；`internal/server` 只拥有分析生命周期、快照提交和 LSP handler；`internal/protocol` 只拥有标准 JSON 模型；`internal/document` 和 `internal/position` 继续提供已有文本版本及位置转换能力。`cmd/tops-lsp/main.go` 只负责组装 server，不增加索引或 grammar 逻辑。

## 设计与实施阶段

### 阶段 0：研究与设计基线

**交付物**：本计划、[research.md](research.md)、[data-model.md](data-model.md)、[contracts/lsp-navigation.md](contracts/lsp-navigation.md)、[contracts/index-lifecycle.md](contracts/index-lifecycle.md)、[quickstart.md](quickstart.md)。

**决定**：技术上下文未知项已全部消除。研究记录了现有 parser/server/protocol 边界、index owner、AST 元数据、header provider、条件候选、稳定 ID、LSP 和测试替代方案。

**验收**：计划中的 Constitution Check 通过；所有设计文件引用同一套 active/unknown/inactive、header 来源和版本失效规则。

### 阶段 1：补充 parser AST 元数据

1. 在 `internal/parser/ast.go` 增加名称精确范围、声明角色和可确认 body 范围；保留现有节点、token、条件状态、parse status 和版本字段。
2. 在 `nodeFromRange`、声明/函数 token 分割和相关构造路径中填充 `NameRange`、`DeclarationRole`；不把名称范围扩展到属性、注释或 trailing body。
3. 对 class/struct/enum/union、函数声明/定义、alias、namespace、attribute 和不完整函数添加 AST metadata 测试。
4. 确保 metadata 缺失时 index 能得到 `unknown`/`incomplete`，而不是猜测名称或跳过整份文档。

**阶段验收**：既有 parser 测试通过；新增测试验证 name range、declaration/definition、incomplete 和 UTF-8 byte range；不新增 LSP 或文件系统依赖。

### 阶段 2：实现 index 核心模型与构建器

1. 新建 `internal/index/model.go`，实现 [data-model.md](data-model.md) 定义的 `SymbolRecord`、`ReferenceRecord`、`ScopeRecord`、`ConditionalProvenance`、`HeaderDependency` 和 `IndexSnapshot`。
2. 实现 AST scope traversal：translation unit、namespace、class/struct/union、function 和 block 形成 scope；同名不同 scope 不合并。
3. 实现声明/定义分类、属性/builtin/macro 来源记录、qualified name、selection range 和 declaration/definition location。
4. 实现 normalized declaration anchor 和稳定 `symbol_id`；覆盖无关文本移动、函数体修改、同名 scope 和重复声明。
5. 遍历 AST expression/tokens 建立 `ReferenceRecord`；按 lexical scope、qualified name 和条件状态收集 candidate，保留 `resolved`/`ambiguous`/`unresolved`。
6. 实现 active/unknown/inactive 过滤、多个 active 候选的 definition/references 规则和 active/unknown document symbol 树。
7. 记录函数参数摘要、返回类型摘要、执行空间限定符、属性来源和 alias 的 `Object` document symbol 映射；添加完整类型/重载/宏展开不被误报为完成的负向状态。

**阶段验收**：index unit tests 能从最小 ParseResult 得到完整/partial/degraded snapshot；测试不要求启动 server、Clang 或设备。

### 阶段 3：header provider 与依赖失效

1. 新建 `SourceProvider` 边界，支持已打开文档查询、`IncludeRoots() (roots, available)`、include roots 按需解析和 source version 返回；为现有 `New(logger)` 保留无 roots 默认 provider，并增加可注入 options/constructor。
2. 在 index builder 中解析 include dependency，建立 requester -> target 和 target -> dependents 关系；检测 missing、unreadable 和 cycle。
3. 对可读取 header 建立独立 source/index snapshot；多个 requester 复用同一 header version，不复制不同定义。
4. 对缺失 header 保留 unresolved dependency，当前文档局部 symbols 仍可建立，并生成 degraded 状态。
5. 对已打开 header 的 `didChange`、context invalidation 或 provider version 变化，使直接/间接 dependents stale；不实现 workspace file watcher 和全量扫描。

**阶段验收**：multi-file、missing-header、include-cycle、same-header-reuse 和 dependency-invalidation 测试通过；provider 可用内存 fixture 替换。

### 阶段 4：server 保存快照和版本提交

1. 在 `Server` 中持有 index store 和可注入 SourceProvider；保持 `document.Store` 只管理文本/版本，`partialParseContextAt` 从 provider 复制 include roots/available 状态。
2. 在现有 `analysisQueue` 任务中，使用同一 `analysisSnapshot` 调用 parser 和 index builder；构建在锁外执行。
3. 提交前复用 `analysisCurrentSnapshot` 检查 document value、generation、epoch、context version 和解析结果版本；不匹配时丢弃 AST/index/diagnostic 结果。
4. 文档成功 `didOpen`/`didChange` 后替换当前 document snapshot；删除声明不能从旧 index 继续出现；`didClose` 清理 index 和依赖边。
5. header/context 失效时标记 dependents stale 并重新排队；未准备好或已过期的查询不读取旧快照。
6. 记录脱敏 index status、candidate count、source/header version、condition、symbol kind、duration、diagnostic code 和 error kind；对 `analysis-degraded` 记录 `analysis_degraded`，对 stale 只记录 `stale_result`，不记录完整 source/index payload。

**阶段验收**：server 测试覆盖新版本覆盖旧版本、任务乱序、删除声明、关闭文档、context invalidation、header dependent invalidation 和 publisher/handler failure。

### 阶段 5：protocol 模型和 LSP 导航 handler

1. 在 `internal/protocol/navigation.go` 增加 `TextDocumentPositionParams`、`ReferenceContext`、`ReferenceParams`、`DocumentSymbolParams`、`DocumentSymbol` 和标准 `SymbolKind`。
2. 在 `ServerCapabilities` 增加三个可选 capability；不改变已有生命周期、文档同步、取消和 framing。
3. 在 `internal/server/navigation.go` 注册并分发 `textDocument/definition`、`textDocument/references` 和 `textDocument/documentSymbol`。
4. handler 将 LSP position 转换为 source byte range，查询当前新鲜 index，再使用 `internal/position.Mapper` 编码 URI/range。
5. 实现 declaration fallback、includeDeclaration、多个 active 候选并集、unknown fallback、inactive 过滤、active/unknown document symbols、alias -> Object 映射和标准空结果。
6. fresh snapshot 无匹配返回空数组；无 fresh snapshot 或查询期间失效返回 `null`；malformed params、未打开 URI、位置越界使用现有 `InvalidParams`；取消沿用 `RequestCancelled`。
7. 保持 `analysis-degraded` 只在 index/diagnostics/log 中表达，不增加私有导航 response 字段。

**阶段验收**：protocol JSON tests、server handler tests 和 LSP contract tests 覆盖成功、空结果、歧义、条件、错误、取消、过期和 UTF-16 range。

### 阶段 6：fixture 与 stdio 集成验证

1. 创建 `testdata/index/manifest.json` 和 single-file、multi-file、headers、conditions、incomplete、missing-context、stale、expected fixtures；每条记录 `case_id`、category、include roots/available、versions、expected symbol/location counts、status、LSP operations 和 diagnostic codes。
2. 复用现有 parser fixtures，增加 index-specific expected 摘要、header URI 变化、注释/字符串过滤、错误节点查询和 out-of-scope negative case；真实 `topsop` corpus 只读引用或使用缩减 fixture，不修改外部 checkout。
3. 扩展 `internal/protocol` JSON 测试，确认 capability 和请求/结果符合标准 LSP 形状。
4. 扩展 `internal/server` 测试，验证分析队列、snapshot store、header invalidation、query stale 和 diagnostics/log 状态。
5. 扩展 `integration/lsp_process_test.go`，通过真实 stdio frame 和标准 LSP JSON decoder proxy 检查 initialize capabilities、打开文档、三项请求、空数组/`null`、参数错误、取消和版本竞争。

**阶段验收**：所有验收场景至少有一个可复现 fixture 或 contract test；测试不依赖设备和 Clang runtime。

### 阶段 7：全量验证和发布前复查

1. 在 `internal/index/manifest.go` 和 `cmd/tops-index-check/main.go` 实现 manifest validator，输出 case/category 总数、通过数、失败 ID，并让 SC-001 至 SC-003 使用该报告。
2. 在 `internal/server/observability.go` 实现 `analysis-degraded`/`stale-result` 的 diagnostics/log 映射和固定字段，再执行定向 observability tests。
3. 按 [quickstart.md](quickstart.md) 执行 manifest validator、定向测试、`go test ./...`、`go test -race ./...`、`go vet ./...`。
4. 检查 definition/references/documentSymbol 的 URI/range、active/unknown/inactive 规则、header missing/cycle/dependency invalidation 和 stale 结果。
5. 在 Clang 工具和完整 context 可用时执行独立对照；缺少工具或 context 时记录 skipped/known difference；执行 `git diff --check`，检查日志隐私、外部 checkout、依赖和完整源码产物。
6. 按 T043 -> T044 -> T045 -> T046 -> T047 顺序完成 fixture/实现/quickstart/全量命令/覆盖审计；最终复查同时读取 `.specify/memory/constitution.md`，逐项记录五项原则结果。

## 依赖关系与并行边界

```text
parser AST metadata
  |
  v
index model/builder/store <---- SourceProvider/header dependency
  |
  v
server snapshot commit/invalidation
  |
  v
protocol models -> navigation handlers -> stdio integration
```

可并行：

- 阶段 2 的 index model/scope 单测与阶段 5 的 protocol JSON model 可在模型命名固定后并行；
- 阶段 3 的 in-memory header provider 与阶段 6 的 fixture 整理可并行；
- parser metadata golden 和 header dependency fixture 可并行准备，但 server 集成必须等待 builder 输入输出固定；
- protocol handler 的空结果/错误测试可先写，真正查询测试必须等待 index store commit contract 固定。

不可跳过的依赖：

- AST 名称范围和声明角色未稳定前，不实现 index 的 definition/selection range；
- index snapshot/version commit 未稳定前，不让 navigation handler 读取旧 parser 结果；
- SourceProvider 和 reverse dependency 未稳定前，不声称 header 变化能刷新跨文件结果；
- LSP contract tests 必须在 protocol model 和 server handler 同时可用后运行；
- 全量验证必须在 parser、index、server、protocol 和 integration 定向测试通过后执行。

## 错误、可观测性和外部数据流

- parser/index 错误使用结构化 status/diagnostic；单个错误不清空当前文本中可确认的局部索引。
- missing header、unreadable header、unknown condition、ambiguous candidate 和 incomplete AST 产生可定位的 degraded 状态；不静默当作完整成功。
- query 不等待无限期分析，不读取旧快照；取消和 stale 结果不发布。
- server 日志只记录脱敏 document ID、operation、symbol kind、version、condition、status、candidate count、duration、diagnostic code 和 error kind；`analysis_degraded`/`stale_result` 的 old/new version 规则见 contracts。
- parser/index 只读取 server 提供的文本和 `SourceProvider` 的本地 source；不访问网络、不执行外部命令、不访问设备。
- `topsop` 和 `llvm-project` 只作为开发机只读 fixture/对照来源；Clang 结果不进入 server runtime。

## 兼容性与迁移

- 新增三个标准可选 capability；旧 client 可以忽略，不影响 initialize、shutdown、exit、didOpen/didChange/didClose、cancel 和 stdio framing。标准 LSP JSON decoder proxy 作为 client 兼容性验收，不引入 TypeScript 语义代码。
- 原来返回 `MethodNotFound` 的三个导航方法在功能启用后变为标准成功/空结果/错误响应；对应 integration 断言必须更新，这是本功能的预期能力增加。
- 不新增 TypeScript 语义代码、私有 LSP message 或持久化格式，不需要迁移数据库。
- `analysis-degraded`、stale 和 context/header version 继续通过内部状态、diagnostics/log 表达；如果未来要加入公开 payload，必须重新记录 owner、兼容性和迁移/拒绝方案。

## 验证命令

在开发机、仓库根目录执行：

```sh
go test ./internal/parser ./internal/document ./internal/position ./internal/protocol ./internal/server ./internal/index
go test ./integration
go test ./...
go test -race ./...
go vet ./...
git diff --check
```

可选 Clang 对照验证只使用测试材料提供的同一 source/context 命令；它不能成为 server runtime 或 Go parser/index 测试的前置条件。完整预期和失败处理见 [quickstart.md](quickstart.md)。

## 设计完成后的宪法复查

- [x] LSP、server、protocol、transport、parser、index 和 TypeScript client 的所有权已在两个 contract 文件中固定；跨边界 payload、错误、取消和兼容性已记录。
- [x] 标准 C++ 与 Tops syntax、条件状态和 header provenance 分开记录；没有从 profile 名称推断 target 宏值。
- [x] parser metadata、index unit、header/dependency、protocol JSON、server stale 和 stdio integration 覆盖正常、失败、不完整和边界路径。
- [x] missing header、unknown/ambiguous、取消、旧版本、关闭文档和 provider failure 都有固定 diagnostics/log code、字段或标准错误行为；fresh 空数组和 stale `null` 已区分。
- [x] 新增 capability 是标准可选字段；没有第三方依赖、runtime Clang、私有 navigation payload、数据库迁移或 workspace 全量扫描。
- [x] 不存在需要复杂度例外的宪法违规；`internal/index` 和 `SourceProvider` 的增加属于清晰职责边界内的必要复杂度。

## 复杂度记录

无宪法违规项。新增 `internal/index` 是为了隔离符号模型、查询和依赖图；新增 `SourceProvider` 是为了隔离 header 来源、支持内存 fixture 并避免 index builder 直接读取文件。拒绝把这些逻辑塞入 server 或 parser，因为会混合生命周期、文件来源和语法所有权，并增加测试耦合。
