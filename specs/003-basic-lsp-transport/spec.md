# Feature Specification: P0 基础服务和 LSP 传输

**Feature Branch**: `003-basic-lsp-transport`

**Created**: 2026-10-08

**Status**: Draft

**Input**: User description: "请为 Tops C++ language server 编写“基础服务和 LSP 传输”功能。范围只包括 Go 项目骨架、server 启停、initialize/shutdown/exit、textDocument didOpen/didChange/didClose、请求取消和基础错误返回。必须写明文件边界、LSP 契约、测试场景和日志要求，不实现具体 Tops 语义，不扩展 clangd。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 启动并完成基础 LSP 会话 (Priority: P1)

作为 VS Code 客户端维护者，我希望能够启动 Go server、完成 `initialize`、正常执行 `shutdown` 和 `exit`，从而确认进程和传输层已经具备可复用的生命周期基础。

**Why this priority**: 没有稳定的进程状态和初始化顺序，后续任何语言服务都无法可靠接入。

**Independent Test**: 启动 server 子进程，通过标准输入输出发送一组合法的 JSON-RPC/LSP 消息，检查初始化响应、状态变化、正常退出状态和 stdout 中的协议内容。

**Acceptance Scenarios**:

1. **Given** server 进程已启动且尚未初始化，**When** 客户端发送合法的 `initialize` 请求，**Then** server 返回 JSON-RPC 成功响应，声明文档同步能力，并进入 initialized 状态。
2. **Given** server 已进入 initialized 状态，**When** 客户端发送 `shutdown` 请求后再发送 `exit` 通知，**Then** server 返回 `shutdown` 成功响应并以正常状态退出。
3. **Given** server 尚未完成 `initialize`，**When** 客户端发送除 `initialize` 外的请求，**Then** server 返回 `ServerNotInitialized` 错误，不退出进程。
4. **Given** server 收到没有对应 `shutdown` 的 `exit` 通知，**When** server 结束进程，**Then** 退出状态表示异常生命周期，且日志记录退出原因。

### User Story 2 - 在编辑过程中同步文档内容 (Priority: P1)

作为 Tops C++ 开发者，我希望打开、修改和关闭文档时，server 能准确保存当前文本和版本，从而为后续语言分析提供稳定的文档输入。

**Why this priority**: 文档同步是所有后续诊断、补全和导航能力的唯一输入入口；本功能只负责保存输入，不解释 Tops 语义。

**Independent Test**: 通过 LSP 通知依次发送 `textDocument/didOpen`、包含一个或多个增量变更的 `textDocument/didChange` 和 `textDocument/didClose`，在 server 测试接口或测试日志中检查文档文本、版本和生命周期状态。

**Acceptance Scenarios**:

1. **Given** 文档尚未打开，**When** server 收到合法的 `textDocument/didOpen`，**Then** server 保存 URI、languageId、文本和版本，并可接受后续变更。
2. **Given** 文档已打开且版本为 `v`，**When** server 收到版本大于 `v` 的多个增量变更，**Then** server 按消息中的顺序逐项应用变更，保存最终文本和新版本。
3. **Given** 文档已打开，**When** server 收到范围越界、版本过旧或不符合当前同步模式的变更，**Then** server 不破坏最近一次有效文本，记录可定位的拒绝原因，并继续处理后续合法消息。
4. **Given** 文档已打开，**When** server 收到 `textDocument/didClose`，**Then** server 删除该 URI 的活动文档状态；关闭后的变更不能重新创建文档。

### User Story 3 - 处理取消和基础协议错误 (Priority: P1)

作为客户端维护者，我希望取消正在处理的请求并获得稳定的基础错误码，从而避免旧请求阻塞会话，也能区分协议错误、参数错误、生命周期错误和内部错误。

**Why this priority**: 编辑器会频繁产生过期请求；如果取消和错误路径不明确，客户端会看到过期结果，维护者也无法定位传输故障。

**Independent Test**: 使用可控的测试请求处理器制造一个未完成请求，发送 `$/cancelRequest`，再分别注入解析错误、未知方法、非法参数和未初始化请求，检查错误响应、无迟到响应以及 server 是否仍能处理下一条合法消息。

**Acceptance Scenarios**:

1. **Given** 一个带请求 ID 的请求仍在处理，**When** server 收到匹配 ID 的 `$/cancelRequest` 通知，**Then** server 停止发布该请求的正常结果，最多返回一次 `RequestCancelled` 错误，并保持会话可用。
2. **Given** `$/cancelRequest` 的 ID 不存在或对应响应已经发出，**When** server 收到该通知，**Then** server 不发送额外响应，不修改文档状态，并记录调试级别事件。
3. **Given** server 收到格式错误的 JSON、非法 JSON-RPC 请求或未知请求方法，**When** server 处理消息，**Then** server 使用对应标准错误码返回错误，或对通知只记录错误而不发送通知响应。
4. **Given** 某条请求处理触发未预期的内部异常，**When** server 处理错误，**Then** 客户端只收到 `InternalError` 摘要和关联标识，详细堆栈只写入受控日志，进程不会因单个请求而静默失效。

### User Story 4 - 观察启动、同步和失败原因 (Priority: P2)

作为语言服务器维护者，我希望从日志中看到会话、消息、文档同步、取消和退出的关键事件，同时不泄露完整源代码或凭据，从而可以诊断连接问题和状态错误。

**Why this priority**: 基础传输在编辑器启动阶段失败时通常没有语义结果；可审计日志是恢复问题的必要证据。

**Independent Test**: 运行正常会话、拒绝消息、取消请求和异常退出场景，检查日志事件、关联 ID、耗时和脱敏规则，并确认 stdout 只包含 LSP 协议帧。

**Acceptance Scenarios**:

1. **Given** server 启动并完成初始化，**When** server 处理生命周期和文档通知，**Then** 日志至少能关联会话、方法、文档脱敏标识、版本、结果和耗时。
2. **Given** 输入包含源代码文本、完整文件路径或敏感配置值，**When** server 写日志，**Then** 日志不包含完整源代码、凭据或不必要的原始配置值。
3. **Given** server 发生协议错误或异常退出，**When** 客户端查看日志，**Then** 能根据错误码和关联标识定位失败阶段、退出原因和是否可重试。

### Edge Cases

| 场景 | 预期行为 | 验证方式 |
| --- | --- | --- |
| 一条消息被拆成多个 stdin 片段 | 按 `Content-Length` 等待完整帧，不提前解析 | 传输测试按字节分段写入 |
| 多条消息在一次读取中连续到达 | 逐帧处理，响应 ID 不串线 | 连续写入两个或多个完整帧 |
| `Content-Length` 超过默认 16 MiB 上限 | 产生 transport error，关闭输入并取消活动 request | 配置上限的 transport 测试和 termination 测试 |
| `Content-Length` 按字符数而非 UTF-8 字节数填写 | 拒绝当前帧并记录协议错误，后续帧按边界继续处理或按实现约定结束会话 | 含非 ASCII 文本的错误长度测试 |
| `didChange` 同时包含多个变更 | 按数组顺序使用同一文档版本应用 | 多变更、重叠范围和前后相邻范围测试 |
| 文档使用多字节字符，位置使用 LSP UTF-16 单位 | 按 LSP position encoding 解释范围，不使用字节下标 | 中英文和补充平面字符的变更测试 |
| 文档使用 CRLF 行尾 | `\r` 属于行终止符，不计入该行的 LSP character 范围，变更后保留 CRLF | CRLF 行首、行尾和跨行变更测试 |
| `contentChanges` 提供错误的 `rangeLength` | 拒绝整个通知并保留最近一次有效文本和版本 | UTF-16 code unit 长度一致/不一致测试 |
| 文档版本重复、倒退或缺失 | 拒绝变更并保留最近有效状态；通知不产生 JSON-RPC 响应 | 版本序列测试和日志检查 |
| 未打开文档就发送 `didChange` 或 `didClose` | 不创建隐含文档；记录警告并继续会话 | 通知顺序测试 |
| 已关闭文档再次发送变更 | 忽略变更，不污染文档存储 | close 后 change 测试 |
| 取消未知请求或已完成请求 | 不发送额外响应，不影响其他请求 | cancel ID 组合测试 |
| `exit` 在 `shutdown` 前到达 | 立即结束并标记非正常生命周期 | 退出状态和日志测试 |
| 客户端在响应前关闭 stdin | server 停止读取并记录 EOF；不等待无法到达的 `exit` | 子进程 EOF 测试 |
| stdout 写入非协议日志 | 视为传输违规，测试必须失败 | stdout 纯净性检查 |
| 收到不支持的语义请求 | 返回 `MethodNotFound`，不启动 Tops 语义分析 | 未知方法测试 |

## Scope and Boundaries

### In Scope

- 创建可编译的 Go 项目骨架和 server 进程入口。
- 使用标准 JSON-RPC 2.0 消息和 LSP over stdio 传输。
- 实现 server 的启动、初始化、关闭和退出状态管理。
- 实现 `initialize`、`shutdown`、`exit` 和必要的 `$/cancelRequest` 处理。
- 实现 `textDocument/didOpen`、`textDocument/didChange`、`textDocument/didClose` 文档同步。
- 返回基础 JSON-RPC/LSP 错误，并记录可诊断日志。
- 为上述行为建立 Go 单元测试、传输测试和进程级集成测试。

### Out of Scope

- 不实现 Tops C++ tokenizer、parser、AST、symbol index、semantic analysis 或任何具体 Tops 语义。
- 不实现 `publishDiagnostics`、completion、hover、definition、references、documentSymbol 等语义服务；本功能不声明这些能力。
- 不解析 `compile_commands.json`、Tops target profile、GCU 宏、Tops headers 或 clang 编译参数。
- 不调用 clangd，不修改 clangd，不将 clangd 作为 server 后端、sidecar 或 fallback。
- 不创建 TypeScript VS Code 扩展功能；客户端只作为本契约的测试对端。
- 不实现 TCP、WebSocket、HTTP 或自定义二进制传输。
- 不执行编译、设备 workload、GCU 测试或运行时语义验证。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 功能 MUST 提供一个可编译的 Go 项目骨架，具有明确的进程入口、内部模块边界、测试入口和可复现的本地测试方式；本规格不锁定第三方 LSP 库。
- **FR-002**: server MUST 使用标准 JSON-RPC 2.0 over stdio；stdout 只能写协议帧，日志和诊断输出 MUST 写入 stderr 或受控日志目标。
- **FR-003**: 每个 LSP 消息 MUST 使用 `Content-Length` 表示 JSON UTF-8 字节长度，并以完整的 header/body 帧处理；不完整帧不能被当作完整请求执行。
- **FR-004**: server MUST 维护明确的生命周期状态：启动未初始化、initialized、shutdown pending 和退出；非法状态转换 MUST 返回可识别错误或按通知规则记录并忽略。
- **FR-005**: server MUST 接受合法的 `initialize` 请求，返回可被客户端消费的 server 信息和最小 capabilities，其中必须声明 `textDocumentSync.openClose=true` 与增量同步；不得声明本功能范围外的语义能力。
- **FR-006**: `initialize` MUST 在响应中保留请求 ID；非法参数、重复初始化和不支持的初始化输入 MUST 返回标准 JSON-RPC 错误，不得启动隐含的语义配置。
- **FR-007**: server MUST 在 initialized 状态接受 `shutdown` 请求，返回 JSON-RPC 成功结果 `null`，并进入等待 `exit` 的状态；未初始化时 MUST 返回 `ServerNotInitialized`。
- **FR-008**: server MUST 接受 `exit` 通知并结束进程；正常 `shutdown` 后的 `exit` 使用正常退出状态，其他顺序使用非正常退出状态并记录原因。
- **FR-009**: server MUST 接受 `textDocument/didOpen`，保存 URI、languageId、文本和版本；保存文本只作为文档输入，不得触发具体 Tops 语义分析。
- **FR-010**: server MUST 按 LSP 增量同步规则处理 `textDocument/didChange` 的 `contentChanges`，支持同一通知中的多个变更，并按数组顺序应用。
- **FR-011**: `didChange` MUST 按 LSP position encoding 解释行和字符位置，初始基线使用 UTF-16；实现不得把字符位置直接当作 UTF-8 字节偏移。
- **FR-012**: server MUST 要求已打开文档的变更版本严格前进；版本重复、倒退、缺失、范围越界、变更模式不匹配或应用失败时，必须保留最近一次有效状态并记录拒绝原因。
- **FR-013**: server MUST 接受 `textDocument/didClose` 并移除活动文档状态；未知 URI 的 close、close 后的 change 和未 open 的 change 不得隐式创建文档。
- **FR-014**: server MUST 处理标准 `$/cancelRequest` 通知，按 JSON-RPC request ID 关联进行中的请求；取消后不得发送该请求的正常结果或迟到结果。
- **FR-015**: 取消请求 MUST 不得终止进程、回滚其他文档或影响其他请求；未知 ID、重复取消和已完成请求的取消必须是无副作用操作。
- **FR-016**: 对请求，server MUST 至少区分并返回 `ParseError`、`InvalidRequest`、`MethodNotFound`、`InvalidParams`、`InternalError`、`ServerNotInitialized` 和 `RequestCancelled` 等标准错误；错误响应必须保留对应 request ID（无法确定 ID 时使用 JSON-RPC 规定的空 ID 行为）。
- **FR-017**: 对通知，server MUST 不发送 JSON-RPC response；通知参数错误、未知通知或状态错误必须记录日志，不得因单条通知结束进程。
- **FR-018**: server MUST 对单个请求的内部异常进行边界处理，向客户端返回不包含堆栈和源代码的错误摘要，并在日志中写入关联标识和完整诊断信息。
- **FR-019**: server MUST 为启动、initialize、shutdown、exit、消息解析、文档打开/变更/关闭、取消、错误和 EOF 记录结构化日志事件；每条事件至少包含时间、级别、事件名、会话标识和结果或错误码。
- **FR-020**: 文档同步和错误日志 MUST 记录脱敏的文档标识、版本和必要的范围信息，但不得记录完整源代码、凭据、密钥或无关用户数据；默认日志级别不得输出每条消息的完整 payload。
- **FR-021**: 实现 MUST 以测试证明 stdout 协议纯净、消息边界正确、生命周期顺序正确、文档变更可复现、取消无迟到响应、错误码稳定且 server 能从可恢复错误继续运行。
- **FR-022**: 实现 MUST 不修改 `llvm-project` 中的 Clang、clangd、Tops headers、driver 或测试语义，不增加任何 clangd 扩展或运行时依赖。
- **FR-023**: 实现 MUST 保持 server 与未来 TypeScript 客户端之间的职责边界：server 负责协议和文档状态，客户端只负责进程管理、消息发送和结果展示；本功能不在客户端复制语义判断。

### 文件边界与交付物

下表是本功能的文件归属契约。具体 Go 类型名和第三方依赖在实施计划中确定，但不得突破目录职责。

| 文件或目录 | 责任 | 本功能允许的内容 | 明确禁止的内容 |
| --- | --- | --- | --- |
| `go.mod`、`go.sum` | Go 项目元数据 | module 声明、必要依赖和版本记录 | Tops 语义依赖、clangd 运行时依赖、未使用依赖 |
| `cmd/tops-lsp/main.go` | 进程入口 | 创建 server、绑定 stdin/stdout、配置日志、映射退出状态 | JSON-RPC 解析细节、文档变更算法、Tops 语义判断 |
| `internal/transport/` | LSP stdio 传输 | `Content-Length` 帧读取/写入、JSON-RPC 消息边界、EOF 和传输错误 | 文档存储、生命周期策略、Tops 语义 |
| `internal/protocol/` | 协议模型和错误 | JSON-RPC/LSP 基础消息、请求 ID、错误码、生命周期和文档同步参数校验模型 | target/profile 解析、语义诊断、clangd 调用 |
| `internal/server/` | server 状态和分发 | 生命周期状态、请求路由、取消登记、请求恢复和响应发送顺序 | tokenizer、parser、AST、symbol index、Tops 规则 |
| `internal/document/` | 文档状态 | URI、languageId、文本、版本、UTF-16 范围变更和 open/close 状态 | C++ 语法、宏、include、编译数据库和语义分析 |
| `internal/logging/` | 可观测性 | 结构化日志、级别、关联标识、脱敏和 stderr/文件输出 | 完整源代码、凭据、静默吞错 |
| `internal/*_test.go` | 单元和组件测试 | 帧、协议、状态、文档、取消、错误和日志行为测试 | 依赖真实 GCU 设备或 clangd 的测试 |
| `testdata/lsp/` | 进程级测试数据 | 合法/非法/不完整消息帧、文档同步序列和预期结果 | Tops 语义 fixture、真实用户源代码和敏感配置 |
| `tops-lsp` 工程根之外 | 其他仓库和工具链 | 不修改 | 不得修改 `/home/carl.du/work/llvm-project` 或 clangd 相关文件 |

本功能不创建 `internal/tokenizer/`、`internal/parser/`、`internal/ast/`、`internal/symbolindex/` 或 `internal/semantic/`。这些目录属于后续语义功能，当前 server 对未知语义方法返回 `MethodNotFound`。

### LSP 传输契约

#### 消息封装

- 传输使用 stdin/stdout 的字节流；每条消息由 ASCII header、空行和 UTF-8 JSON body 组成。
- `Content-Length` 是 body 的 UTF-8 字节数，不是 Unicode 字符数；header 名称大小写不影响识别。
- `Content-Type` 等兼容 header 可以出现并被忽略；格式错误、重复冲突或影响 `Content-Length` 解析的 header 必须拒绝。
- server 必须支持单条消息被分片读取以及多条消息合并读取；解析器只能在 body 达到声明长度后交给 JSON-RPC 层。
- stdout 不得混入日志、调试文本或 panic 输出；无法写入响应时记录 stderr 日志并结束会话或进入明确失败状态。
- stdin EOF 表示客户端断开。server 必须释放文档和请求状态并记录 EOF，不等待不存在的 `exit`。

#### JSON-RPC 和 LSP 方法

| 方法 | 类型 | 最小输入 | 成功行为 | 失败行为 |
| --- | --- | --- | --- | --- |
| `initialize` | request | JSON-RPC ID、LSP initialize params | 返回 server 信息和仅包含文档同步的 capabilities，进入 initialized | 参数错误返回 `InvalidParams`；重复或状态不合法返回 `InvalidRequest` |
| `shutdown` | request | JSON-RPC ID，无需业务参数 | 返回 `null`，进入 shutdown pending | 未初始化返回 `ServerNotInitialized`；重复 shutdown 返回状态错误 |
| `exit` | notification | 无需参数 | 结束进程；正常 shutdown 后为正常退出 | 未完成 shutdown 时为非正常退出；通知不返回 response |
| `textDocument/didOpen` | notification | URI、languageId、version、text | 建立或替换该 URI 的活动文档输入 | 参数错误或重复状态只记录日志，不发送 response |
| `textDocument/didChange` | notification | URI、version、按顺序排列的增量 `contentChanges` | 应用全部合法变更并保存新版本 | 任一变更不能应用时保留旧状态并记录拒绝；不发送 response |
| `textDocument/didClose` | notification | URI | 移除活动文档 | 未知 URI 只记录日志；不发送 response |
| `$/cancelRequest` | notification | 被取消的 request ID | 标记对应请求已取消 | 未知或已完成 ID 无副作用；不发送 response |

`initialize` 返回的 `ServerCapabilities` 不得声明本功能未实现的 semantic provider。`textDocumentSync` 必须表示 open/close 通知和增量 change；`save` 不属于本功能承诺。客户端发送的其他请求不进入语义 fallback，而是返回 `MethodNotFound`。

#### 生命周期状态

| 当前状态 | 允许消息 | 状态变化 | 其他消息行为 |
| --- | --- | --- | --- |
| 启动未初始化 | `initialize`、`exit` | initialize 成功后进入 initialized；exit 直接异常退出 | request 返回 `ServerNotInitialized`；notification 记录并忽略 |
| initialized | 文档通知、`shutdown`、`$/cancelRequest` | shutdown 成功后进入 shutdown pending | 未知 request 返回 `MethodNotFound` |
| shutdown pending | `exit`、已在处理的取消通知 | exit 后进程结束 | 新 request 返回状态错误；新的文档通知不再改变状态 |
| exited | 无 | 进程已结束 | 不再读取或发送消息 |

### 基础错误契约

错误响应使用 JSON-RPC `error` 对象，`code` 使用标准值，`message` 是稳定且面向用户可理解的摘要，`data` 只放不含源代码的结构化诊断信息。最小错误表如下：

| 错误名 | code | 使用场景 | 是否可重试 |
| --- | ---: | --- | --- |
| `ParseError` | `-32700` | JSON body 无法解析 | 由客户端修正消息后重试 |
| `InvalidRequest` | `-32600` | 缺少 JSON-RPC 字段、非法 ID 或生命周期顺序不合法 | 修正请求后重试 |
| `MethodNotFound` | `-32601` | 请求方法不在本功能契约内 | 否，客户端应停止发送该方法 |
| `InvalidParams` | `-32602` | 请求参数类型或必需字段错误 | 修正参数后重试 |
| `InternalError` | `-32603` | 未预期的 server 内部错误 | 由客户端决定重试，并依据日志处理 |
| `ServerNotInitialized` | `-32002` | initialize 之前收到需要初始化的请求 | 初始化后重试 |
| `RequestCancelled` | `-32800` | 请求在完成前被 `$/cancelRequest` 取消 | 否，旧请求结果不得补发 |

通知没有 JSON-RPC response。若通知 body 可解析但参数无效，server 只能记录错误、保留已有状态并继续会话；若整条消息无法解析且无法确定消息类型，按 JSON-RPC 规则发送 `id: null` 的解析错误（若传输仍可恢复），否则记录日志并结束会话。

### 请求取消规则

- server 为每个仍未完成的 request ID 维护可取消状态；request 完成、取消或出错后必须移除登记。
- `$/cancelRequest` 只表达取消意图，不保证已经开始的底层操作立即停止；但 server 不得在取消确认后发送正常结果。
- 取消与正常完成并发时只能有一个终态：成功、标准错误或 `RequestCancelled`，不能重复响应。
- 取消不改变 `DocumentState`、生命周期状态或其他 request 的结果。
- 由于本功能没有长时间语义请求，生产路径的取消测试使用传输层可控的阻塞 request handler；该 handler 只用于测试，不构成公开 LSP 方法。

### 日志要求

#### 必须记录的事件

- 进程启动、传输初始化失败和 stdin/stdout 绑定结果。
- LSP 帧读取、解析失败、未知方法和参数拒绝；默认只记录方法、ID 和长度，不记录完整 payload。
- `initialize` 成功或失败、客户端标识摘要、工作区数量摘要和 server 状态变化。
- `didOpen`、`didChange`、`didClose` 的接受或拒绝、文档脱敏标识、版本和变更数量。
- `$/cancelRequest` 的收到、命中/未命中和最终请求状态。
- `shutdown`、`exit`、EOF、正常/异常退出和未处理内部错误。

#### 字段和隐私

每条结构化日志至少包含 `timestamp`、`level`、`event`、`session_id`，请求相关事件还包含 `request_id`、`method`、`duration_ms` 和 `result` 或 `error_code`；文档事件包含脱敏后的 `document_id` 和 `document_version`。路径只能按实现约定脱敏，不能默认写入完整绝对路径。

日志不得包含完整源代码、完整 JSON payload、访问令牌、密码、密钥或无关用户数据。调试级别也只能输出必要的字段和长度、范围摘要。日志输出不得进入 stdout；默认错误和生命周期日志写入 stderr，后续文件日志配置必须在客户端契约中单独记录。

## Test Scenarios

测试分为纯 Go 单元测试、传输组件测试和 server 子进程集成测试。所有测试都使用合成文本和合成 URI，不依赖 Tops compiler、clangd、GCU 设备或网络。

| ID | 场景类型 | 输入 | 预期结果 | 验证层 |
| --- | --- | --- | --- | --- |
| `T-TRANSPORT-001` | 有效 | 单条带正确 `Content-Length` 的 `initialize` 帧 | 完整读取并返回同 ID 响应 | transport + process |
| `T-TRANSPORT-002` | 边界 | header/body 分片，多个帧合并，body 含 UTF-8 文本 | 帧边界和 body 字节数正确，无串帧 | transport |
| `T-TRANSPORT-003` | 无效 | 缺失或错误 `Content-Length`、非法 JSON、格式错误的 header | 返回/记录协议错误，不把后续数据误当 body；可选 `Content-Type` 等兼容 header 不改变帧边界 | transport + process |
| `T-TRANSPORT-004` | 隔离 | 启动、处理消息和 panic recovery 期间产生日志 | stdout 只有协议帧，日志只在 stderr/受控目标 | process + logging |
| `T-LIFE-001` | 有效 | initialize -> shutdown -> exit | 成功响应、正常退出、状态顺序完整 | server + process |
| `T-LIFE-002` | 无效 | 未 initialize 的 shutdown/未知 request、重复 initialize | 返回稳定错误，进程继续运行 | server |
| `T-LIFE-003` | 边界 | exit 在 shutdown 前、stdin EOF、响应写失败 | 正确退出状态，释放状态并记录原因 | server + process |
| `T-SYNC-001` | 有效 | didOpen 一个空文档和一个含 UTF-8 文本的文档 | 保存 URI、languageId、全文和版本 | document |
| `T-SYNC-002` | 有效 | didOpen 后发送单个和多个增量 change | 按顺序得到预期最终文本和版本 | document + LSP |
| `T-SYNC-003` | 边界 | UTF-16 位置、行尾、空范围插入、删除和替换 | 范围计算与 LSP position encoding 一致 | document |
| `T-SYNC-004` | 无效 | 旧版本、重复版本、越界范围、full change、未 open change | 拒绝当前通知，保留最近有效状态，不发送 response | document + logging |
| `T-SYNC-005` | 生命周期 | didClose 后 change，再次 open 同 URI | close 后 change 无效；重新 open 建立新基线 | document + server |
| `T-CANCEL-001` | 有效 | 可控阻塞 request 与匹配的 `$/cancelRequest` | 只产生一次 `RequestCancelled` 终态，不产生正常结果 | server |
| `T-CANCEL-002` | 边界 | 未知 ID、重复 cancel、响应已发送后 cancel | 无额外响应，不影响其他 request 或文档 | server + process |
| `T-ERROR-001` | 协议 | ParseError、InvalidRequest、MethodNotFound、InvalidParams | 错误 code、ID、message 和 data 形状稳定 | protocol + server |
| `T-ERROR-002` | 内部错误 | 测试 handler 抛出内部错误 | 客户端无堆栈；日志有错误 code 和关联 ID；会话仍可继续 | server + logging |
| `T-ERROR-003` | 通知 | 非法 didOpen/didChange/didClose 参数或未知通知 | 不返回 response，不修改有效文档，日志可定位 | server + logging |
| `T-LOG-001` | 日志 | 正常会话、拒绝消息、取消和异常退出 | 必填事件和字段存在，URI/文本/敏感值已脱敏 | logging + process |
| `T-BOUNDARY-001` | 范围 | 请求 completion、hover、definition 或任意 Tops 语义方法 | 返回 MethodNotFound，不调用 clangd，不解析 Tops 语义 | server + repository review |

测试必须覆盖每类行为的成功、失败和边界路径；`T-BOUNDARY-001` 还必须检查代码变更范围，不得出现 `llvm-project`、clangd 扩展或具体 Tops 语义修改。

## Key Entities *(include if feature involves data)*

- **ServerSession**：一次 server 进程的会话状态，包含生命周期状态、session ID、已初始化标记和活动请求登记。
- **DocumentState**：一个已打开文档的 URI、languageId、文本、版本和最近一次有效变更结果；不包含 AST 或 Tops 语义结果。
- **RequestContext**：正在处理的 JSON-RPC 请求的 ID、方法、取消状态、开始时间和终态，保证请求最多产生一个响应。
- **LSPMessage**：带 JSON-RPC 字段和 stdio 帧元数据的请求、响应或通知；帧长度按 UTF-8 字节计算。
- **LSPError**：稳定错误 code、message、可选的脱敏 data 和 request ID，用于协议错误和请求失败。
- **LogRecord**：结构化日志事件，包含时间、级别、事件、会话/请求关联信息、结果和脱敏文档信息。
- **FileBoundary**：记录文件或目录的责任、允许内容和禁止内容，防止基础传输功能越界到 Tops 语义或 clangd。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 进程级测试能够在同一会话中完成 initialize、至少一轮文档打开/变更/关闭、shutdown 和 exit；所有合法响应的 request ID 与请求一一对应。
- **SC-002**: 传输测试覆盖分片帧、合并帧、UTF-8 字节长度和 stdout 纯净性；测试中不得出现无法归属的协议字节。
- **SC-003**: 文档同步测试覆盖单变更、多变更、UTF-16 位置、版本倒退、越界范围和 close 后 change；每个失败场景都证明最近一次有效文本未被破坏。
- **SC-004**: 取消测试证明每个被取消 request 最多产生一个终态，且取消后没有正常响应、迟到响应或对其他 request/document 的副作用。
- **SC-005**: 基础错误测试覆盖至少 7 个规定错误名，并验证错误 code、request ID、message 结构和通知无响应规则。
- **SC-006**: 日志测试覆盖启动、初始化、文档同步、取消、错误、EOF 和退出事件；日志包含规定关联字段且不包含完整源代码、完整 payload 或凭据。
- **SC-007**: 文件边界检查确认本功能只新增或修改 Go 骨架、基础 LSP 传输、文档存储、日志和对应测试，不修改 Tops 语义、clangd、`llvm-project` 或 TypeScript 扩展。
- **SC-008**: 对本功能范围外的语义请求，server 一律返回 `MethodNotFound` 或按通知规则记录并忽略，不产生隐含的普通 C++ 或 Tops 语义结果。
- **SC-009**: 实施完成后，Go 项目可在无 GCU 设备、无 clangd 和无网络的环境中完成本功能的构建和测试；具体命令在实施计划中记录。

## Assumptions

- 初始传输固定为标准 LSP over stdio；不在本功能中提供 TCP、WebSocket、HTTP 或自定义传输。
- LSP 文档同步使用增量模式，位置基线使用 UTF-16；客户端遵守 LSP 的 `version` 和 `contentChanges` 约定。
- `initialize` 只建立协议会话，不解析 compile database、不选择目标架构、不读取 Tops headers，也不启动语义分析。
- server 可以接受客户端发送的非本功能通知而不返回 response；未知请求使用 `MethodNotFound`，不提供隐藏 fallback。
- 日志默认面向本地开发和诊断，源文本只保存在内存中的 `DocumentState`；本功能不引入外部上传或远程日志服务。
- 具体 Go module path、LSP 数据结构库、日志编码和测试框架由实施计划确定，但必须满足本规格的文件边界和协议行为。
- TypeScript 客户端、Go 语义层和 clangd 集成属于后续功能；当前规格完成后可以独立实现和验收基础 server。
