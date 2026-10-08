# Tasks: P0 基础服务和 LSP 传输

**Input**: Design documents from `specs/003-basic-lsp-transport/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [quickstart.md](quickstart.md), [contracts/](contracts/)

**Implementation language**: Go `1.26.8`; use the verified user SDK at `/home/carl.du/sdk/go1.26.8`.

**Scope**: 本任务列表只实现 Go 项目骨架、LSP over stdio、server 生命周期、文档同步、请求取消、基础错误和结构化日志。不得实现 Tops C++ tokenizer/parser/AST/symbol index/semantic，不得修改 `llvm-project`，不得扩展或调用 clangd，不得创建 TypeScript 代码或其他传输。

**Tests**: 所有 server 行为、LSP 契约、文档同步、取消、错误和日志任务都包含测试任务。测试使用 Go 标准库、合成 URI/文本和 server 子进程，不依赖 GCU 设备、Tops compiler、clangd、网络或测试机。

**Task format**: `[ID] [P?] [Story] Description`

- `[P]`：可以并行执行，且不编辑同一未完成文件。
- `[Story]`：任务所属用户故事；共享基础任务不使用故事标签。
- 每个任务都包含明确的文件路径；同一文件的修改任务必须按依赖顺序执行。

## Phase 1: Setup

**Purpose**: 建立 Go module 和基础目录边界，不实现行为。

- [x] T001 确认 `/home/carl.du/sdk/go1.26.8/bin/go`、`go version` 和 `go env GOROOT`，在仓库根目录创建 `go.mod`；模块路径按仓库命名约定确认，Go 版本记录为 `1.26.8`。
- [x] T002 [P] 创建 `cmd/tops-lsp/`、`internal/transport/`、`internal/protocol/`、`internal/server/`、`internal/document/`、`internal/logging/`、`integration/` 和 `testdata/lsp/` 目录边界；不创建语义目录。
- [x] T003 [P] 将 [contracts/file-boundaries.md](contracts/file-boundaries.md) 转成实现检查项，确认 `main.go`、transport、protocol、server、document、logging 和 integration test 的责任不重叠。
- [x] T004 验证空 Go module 可以执行 `go list ./...`，并记录当前没有既有 Go 源码、module 或 `.vscode/tasks.json` 的事实；不引入第三方依赖。

**Checkpoint**: Go module 和目录边界就绪；后续任务可以创建代码和行为测试。

## Phase 2: Foundational

**Purpose**: 固定所有用户故事共同依赖的协议模型、错误模型和 stdio framing。用户故事实现不得在本阶段前开始。

### Tests first

- [x] T005 [P] 编写 `internal/protocol/errors_test.go`，覆盖 7 个基础错误名、标准 code、request ID 保留、`result`/`error` 互斥以及通知不产生 response 的断言。
- [x] T006 [P] 编写 `internal/transport/stdio_test.go`，覆盖正确 `Content-Length`、UTF-8 字节长度、分片 header/body、多帧合并、可选 `Content-Type`、非法 header、EOF 和 stdout 写帧。
- [x] T007 [P] 编写 `internal/protocol/messages_test.go`，覆盖 request、notification、response、string/number ID、initialize/shutdown/document sync 的最小参数解码和未知字段处理。

### Implementation

- [x] T008 实现 `internal/protocol/messages.go`，定义 JSON-RPC 2.0 request/notification/response、ID、initialize、shutdown、document sync 和 cancel 的最小协议模型；不得加入语义 provider。
- [x] T009 实现 `internal/protocol/errors.go`，固化 `ParseError`、`InvalidRequest`、`MethodNotFound`、`InvalidParams`、`InternalError`、`ServerNotInitialized` 和 `RequestCancelled` 的 code、message 和脱敏 data。
- [x] T010 实现 `internal/transport/stdio.go`，提供 `Content-Length` frame reader/writer、UTF-8 字节计数、分片/合并帧处理、EOF 和传输错误边界；stdout 只接受协议写入。
- [x] T011 运行 `gofmt`、`go test ./internal/protocol ./internal/transport` 和 `go vet ./...`，确认协议/传输基础组件通过后再进入用户故事。

**Checkpoint**: 协议模型、错误模型和字节传输层可独立测试，且不依赖 server 或 Tops 语义。

## Phase 3: User Story 1 - 启动并完成基础 LSP 会话 (Priority: P1) 🎯 MVP

**Goal**: server 可以启动、初始化、处理最小生命周期和文档同步能力声明，并按合法/非法顺序退出。

**Independent Test**: 通过 server 子进程发送 initialize -> shutdown -> exit 和非法顺序消息，检查 response ID、capabilities、stdout 纯净性、stderr 和退出状态。

### Tests for User Story 1

- [x] T012 [P] [US1] 编写 `internal/server/lifecycle_test.go`，覆盖 starting/initialized/shutdown_pending/exited 状态、initialize 成功、重复 initialize、未初始化 request、shutdown 返回 `null`、exit 顺序和未知 request。
- [x] T013 [P] [US1] 编写 `integration/lsp_process_test.go` 的基础子进程测试，覆盖 stdin/stdout 管道、完整/分片 LSP frame、合法 lifecycle 和正常/异常退出状态。
- [x] T014 [P] [US1] 在 `testdata/lsp/valid-session.jsonl` 和 `testdata/lsp/invalid-messages.jsonl` 写入合成 lifecycle 输入序列；不使用真实用户源码。

### Implementation for User Story 1

- [x] T015 [US1] 实现 `internal/server/server.go` 的 session 状态机、消息分发和 response 写入顺序；区分 request 与 notification，并保证每个 request 最多一次 response。
- [x] T016 [US1] 在 `internal/server/server.go` 实现 `initialize`、`shutdown`、`exit` 和未知方法处理；`initialize` 只声明 `textDocumentSync.openClose=true` 与 incremental change，不声明语义能力。
- [x] T017 [US1] 实现 `cmd/tops-lsp/main.go` 的进程入口、stdin/stdout 绑定、server 创建、stderr 日志初始化和正常/异常退出状态映射。
- [x] T018 [US1] 完成 `integration/lsp_process_test.go`，断言 stdout 不包含日志或 panic 文本，stderr 能区分启动、协议错误和退出原因。

**Checkpoint**: User Story 1 可独立演示和测试；server 能作为无语义 LSP transport 进程运行。

## Phase 4: User Story 2 - 在编辑过程中同步文档内容 (Priority: P1)

**Goal**: server 保存合法的 open/change/close 文档状态，按 UTF-16 范围和递增版本原子应用增量变更。

**Independent Test**: 直接测试 `DocumentState` 和 server notification 路由，覆盖有效文本、多个 change、UTF-16、错误版本、越界范围和 close 后 change。

### Tests for User Story 2

- [x] T019 [P] [US2] 编写 `internal/document/store_test.go`，覆盖 didOpen、单个/多个增量 change、空范围插入、删除、替换、多行文本、BMP/非 BMP UTF-16 位置和 didClose。
- [x] T020 [P] [US2] 编写 `internal/document/store_invalid_test.go`，覆盖未 open change、重复/倒退/缺失版本、越界 range、start 晚于 end、full change 和中途失败时的原子性。
- [x] T021 [US2] 编写 `internal/server/document_sync_test.go`，覆盖通知参数解码、文档路由、非法通知无 response、close 后 change 不重新创建和最近有效状态保留。
- [x] T022 [P] [US2] 在 `testdata/lsp/document-changes.jsonl` 添加合成的 UTF-8/UTF-16、多 change、错误版本和 close 顺序样例。

### Implementation for User Story 2

- [x] T023 [US2] 实现 `internal/document/store.go` 的 `DocumentState`、URI 规范化、文本版本、open/close 和增量变更事务；不得引入 AST、宏或 target context。
- [x] T024 [US2] 在 `internal/document/store.go` 实现 UTF-16 Position 到 UTF-8 字节边界的转换、行尾处理和 range 校验；错误变更不得提交部分结果。
- [x] T025 [US2] 在 `internal/server/server.go` 接入 `textDocument/didOpen`、`textDocument/didChange`、`textDocument/didClose`；通知失败只记录日志并保留旧状态。
- [x] T026 [US2] 将文档同步接入 `integration/lsp_process_test.go`，验证 server 子进程处理 open/change/close 后仍能响应 shutdown，不产生语义诊断。

**Checkpoint**: User Story 2 可独立证明文档输入可靠，且不产生任何 Tops 语义结果。

## Phase 5: User Story 3 - 处理取消和基础协议错误 (Priority: P1)

**Goal**: server 能取消活动 request、避免重复/迟到 response，并从可恢复协议和内部错误中继续服务。

**Independent Test**: 使用测试专用阻塞 handler 发送匹配/未知/重复 cancel，同时注入解析、参数、未知方法和内部异常，检查单一终态和会话可用性。

### Tests for User Story 3

- [x] T027 [P] [US3] 编写 `internal/server/cancel_test.go`，使用测试专用阻塞 handler 覆盖 cancel-before-dispatch、cancel-during-request、未知 ID、重复 cancel、已完成 request 和无迟到 response。
- [x] T028 [P] [US3] 编写 `internal/server/error_recovery_test.go`，覆盖未初始化 request、未知方法、非法参数、非法通知、内部异常、后续合法 request 和通知无 response。
- [x] T029 [P] [US3] 编写 `internal/transport/recovery_test.go`，覆盖非法 JSON、无法确定 ID 的 `ParseError`、可恢复后续 frame、EOF 和写响应失败。
- [x] T030 [US3] 在 `integration/lsp_process_test.go` 增加 cancel/error 会话，确认进程不因单条 request 或 notification 错误静默退出。

### Implementation for User Story 3

- [x] T031 [US3] 实现 `internal/server/requests.go` 的活动 request registry、request ID 比较、单一终态转换和取消信号；不得影响其他 request、document 或 session 状态。
- [x] T032 [US3] 在 `internal/server/server.go` 接入 `$/cancelRequest`，使用 `context.Context` 或等价信号传递取消，并在 response 写入前再次检查终态。
- [x] T033 [US3] 完善 `internal/server/server.go` 的请求/通知错误边界：请求返回标准错误，通知只记录并忽略；未知语义 request 固定返回 `MethodNotFound`，不得调用 clangd。
- [x] T034 [US3] 在 `internal/server/server.go` 捕获未预期内部异常，向客户端返回不含堆栈的 `InternalError`，并保留可关联日志信息；单个 request 错误后继续读取下一条消息。
- [x] T035 [US3] 运行 `go test -race ./internal/server ./internal/transport`，确认取消与 response 竞态没有数据竞争、重复 response 或 goroutine 泄漏。

**Checkpoint**: User Story 3 可独立证明取消和基础错误行为稳定，且不会把错误转成隐含语义结果。

## Phase 6: User Story 4 - 观察启动、同步和失败原因 (Priority: P2)

**Goal**: 日志能关联会话、request、document 和失败阶段，同时不泄露源代码、完整 payload 或凭据。

**Independent Test**: 运行正常、拒绝、取消、EOF 和异常退出场景，检查结构化事件、stderr 位置、脱敏字段和错误关联。

### Tests for User Story 4

- [x] T036 [P] [US4] 编写 `internal/logging/logger_test.go`，覆盖日志级别、固定事件名、session/request/document 关联字段、stderr writer 和敏感值脱敏。
- [x] T037 [P] [US4] 编写 `internal/server/logging_test.go`，覆盖启动、initialize、document open/change/close、cancel、error、shutdown、exit 和 EOF 事件。
- [x] T038 [US4] 在 `integration/lsp_process_test.go` 增加 stdout/stderr 分离、完整 payload 不出现、URI/路径脱敏和异常退出关联测试。

### Implementation for User Story 4

- [x] T039 [US4] 实现 `internal/logging/logger.go` 的结构化 `LogRecord`、日志级别、session ID、request ID、脱敏 document ID、错误 code 和 duration 字段。
- [x] T040 [US4] 由 `internal/server/server.go` 统一记录 transport 的启动绑定、frame 拒绝、EOF、读写失败和 payload 长度摘要；不记录完整 body。
- [x] T041 [US4] 将日志接入 `internal/server/server.go` 和 `internal/document/store.go`，记录生命周期、文档接受/拒绝、取消、错误和退出事件。
- [x] T042 [US4] 完成 `cmd/tops-lsp/main.go` 的 stderr logger wiring，确认 stdout writer 与 logger writer 永不共用；更新 `contracts/lsp-transport.md` 中实际字段差异（如有）。

**Checkpoint**: User Story 4 可独立证明基础 server 可观察且满足隐私边界。

## Phase 7: Polish & Cross-Cutting Validation

**Purpose**: 完成全量测试、规格追踪、范围检查和 quickstart 验收。

- [x] T043 [P] 运行 `gofmt -w` 覆盖所有 Go 源码和测试文件，并运行 `go vet ./...`。
- [x] T044 [P] 运行 `go test ./...`，记录所有包和进程级测试结果。
- [x] T045 [P] 运行 `go test -race ./...`，重点检查 transport、server request registry、document store 和 logger。
- [x] T046 对照 `spec.md` 的 FR-001 至 FR-023、SC-001 至 SC-009 和 `contracts/` 逐项检查实现；将未完成项记录为 blocked，不宣称 supported。
- [x] T047 检查 `git diff --name-only`，确认变更只包含 Go module、`cmd/tops-lsp`、`internal/`、`integration/`、`testdata/lsp/` 和必要文档；不得出现 `llvm-project`、clangd、Tops semantic/parser 或 TypeScript 文件。
- [x] T048 执行 [quickstart.md](quickstart.md) 的完整命令，补充实际 Go 版本、测试结果和已知限制；若官方 Go 版本检查仍不可用，保留 `toolchain-review-needed` 说明。
- [x] T049 重新执行 Constitution Check，确认 LSP owner、Go/TypeScript 边界、错误/日志、测试覆盖和兼容性没有因实现变更而失效。

## Phase 8: Convergence Follow-up

**Purpose**: 补齐当前实现与规格之间仍存在的测试、日志和文档缺口。本阶段任务均为新增工作，不能沿用前面 49 个已完成任务的状态。

**Evidence**: 当前 `go test ./...` 通过，但现有测试没有进程级 malformed JSON/EOF 场景；request 日志没有统一断言 `request_id`、`method`、`duration_ms` 和结果/错误字段；`quickstart.md` 仍描述为“尚未创建 Go 源码”；文档变更测试没有覆盖 CRLF 行尾和 `rangeLength` 一致性。

- [x] T050 [P] 为 `integration/lsp_process_test.go` 增加 malformed JSON frame 场景，验证 server 返回 `ParseError`、使用 `id: null`、记录解析事件，并能继续处理后续合法消息。
- [x] T051 [P] 新增 `internal/server/eof_test.go` 或等价进程测试，验证 stdin EOF 的非正常退出状态、`stdin_eof` 日志、活动 request 的取消和等待，不留下后台 goroutine。
- [x] T052 [P] 扩展 `internal/server/logging_test.go` 和进程级日志断言，验证每条结构化日志含 `timestamp`、`level`、`event`、`session_id`；request 事件含 `request_id`、`method`、`duration_ms` 以及 `result` 或 `error_code`。
- [x] T053 为 `internal/server/server.go` 增加 request 生命周期日志上下文，在 initialize、shutdown、未知方法、参数错误、内部错误、取消和 response 写失败路径补齐 `request_id`、`method`、`duration_ms`、结果/错误字段；不得记录完整 params。
- [x] T054 为 `internal/server/server.go` 的 panic recovery 增加受控完整诊断记录，客户端仍只收到无堆栈 `InternalError`；测试验证日志有关联 ID 且 stdout/response 不含堆栈或源代码。
- [x] T055 [P] 在 `internal/document/store_test.go` 和 `internal/document/store_invalid_test.go` 增加 CRLF 行尾、行尾位置和 `rangeLength` 一致性场景，覆盖有效、越界和不一致输入。
- [x] T056 在 `internal/document/store.go` 实现 CRLF 行尾的 LSP position 处理，并在 `rangeLength` 存在时校验其与被替换范围的 UTF-16 长度一致；失败时保持最近有效状态。
- [x] T057 更新 `specs/003-basic-lsp-transport/quickstart.md`，删除“尚未创建 Go 源码”和“实现阶段创建”等过时表述，记录当前 Go module、实现入口、测试命令和已验证结果。
- [x] T058 对照更新后的日志/文档/文档同步契约检查 `spec.md`、`contracts/lsp-transport.md`、`data-model.md` 和实现；若行为与契约不一致，先修实现或记录明确的兼容性决定。
- [x] T059 运行 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`、`git diff --check` 和 quickstart 检查，并把 Convergence 任务逐项标记为完成。

## Phase 9: Convergence Follow-up 2

**Purpose**: 收敛异步 request 分发下的 lifecycle 并发、初始化摘要和 transport 错误可观测性。当前全量测试和 race 已通过，但以下差异来自代码与规格的直接对照，尚未由现有测试覆盖。

- [x] T060 [P] 新增 `internal/server/lifecycle_concurrency_test.go`，并发发送两个 `initialize` 或两个 `shutdown` request，验证只能有一个合法状态转换，其他 request 返回稳定错误且每个 request 只响应一次。
- [x] T061 在 `internal/server/server.go` 将 lifecycle 状态检查与状态写入放入同一临界区，明确并发 initialize/shutdown/exit 的优先级和响应行为；补充 race 验证。
- [x] T062 [P] 扩展 `internal/server/logging_test.go` 和进程级日志断言，验证 `server_started`、transport/frame 拒绝、initialize 成功/失败和 parse error 日志包含必要的长度、client/workspace 摘要及结果字段。
- [x] T063 在 `internal/protocol/messages.go` 和 `internal/server/server.go` 保存 initialize 的 `clientInfo`、`processId` 和 workspace 数量摘要，并记录 transport ready/bind 结果；不得记录完整 initialize payload。
- [x] T064 在 `internal/server/server.go` 为 parse error response 写失败、初始化参数失败和 frame 读取失败补充结构化错误事件，包含脱敏 request/方法/长度信息；增加对应错误路径测试。
- [x] T065 统一 `spec.md`、`plan.md`、`data-model.md` 和 `contracts/lsp-transport.md` 中的耗时字段名称，采用已实现的 `duration_ms`，并检查日志事件名称和字段表一致。
- [x] T066 运行 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`、`git diff --check` 和文档诊断，完成后再标记 Phase 9 任务。

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**：无依赖；先建立 Go module 和目录边界。
- **Phase 2 Foundational**：依赖 Setup；完成前不进入用户故事实现。
- **User Story 1**：依赖 Foundational；它是 MVP，先提供可启动和可通信的 server。
- **User Story 2**：依赖 US1 的 server 分发，但文档 store 的测试和实现可在 US1 稳定后独立推进。
- **User Story 3**：依赖 US1 的 request 路由和协议错误模型；取消和错误恢复在 US2 文档路由之后接入。
- **User Story 4**：依赖 US1-US3 的生命周期、文档、取消和错误事件；logger 基础测试可以提前编写。
- **Phase 7 Polish**：依赖所有目标用户故事完成。
- **Phase 8 Convergence**：依赖 Phase 7；补齐初始实现的异常路径、日志字段、文档同步边界和 quickstart。
- **Phase 9 Convergence Follow-up 2**：依赖 Phase 8；补齐并发 lifecycle、initialize 摘要和 transport 错误可观测性。

### User Story Dependencies

- **US1 (P1)**：Foundational 完成后可开始，无语义依赖。
- **US2 (P1)**：依赖 US1 的 server 分发入口；`internal/document` 单元测试不依赖 Tops 语义。
- **US3 (P1)**：依赖 protocol errors 和 server request 路由；不依赖任何 Tops 语义。
- **US4 (P2)**：依赖前面故事产生的事件点；日志单测可与文档 store 单测并行。

### Parallel Opportunities

- T002、T003 可并行执行；它们分别处理目录和边界审查。
- T005-T007 可并行编写，因为分别修改 protocol errors、transport 和 protocol message 测试文件。
- T012-T014 可并行编写；它们分别修改 lifecycle test、integration test 和 testdata。
- T019、T020、T022 可并行编写；T021 依赖 server 路由测试结构，不与 T012 并行修改同一文件。
- T027-T029 可并行编写；T030 需要等基础进程测试结构稳定。
- T036-T038 可并行编写；日志实现从 T039 开始。
- T043-T045 可并行执行不同验证命令，但如果环境资源有限，按 T043 -> T044 -> T045 顺序执行。
- 修改 `internal/server/server.go`、`integration/lsp_process_test.go` 或同一协议文件的任务必须顺序执行。

### Within Each User Story

- 先写行为测试，再实现对应行为；测试应在缺少实现时失败，除非测试只验证已有基础组件。
- 先完成数据模型和协议边界，再接入 server 路由。
- 先完成单元/组件测试，再运行进程级集成测试。
- 故事的独立测试通过后，才进入下一个故事；失败项不得被标记为完成。

## MVP and Incremental Strategy

### MVP: User Story 1

1. 完成 Phase 1 Setup 和 Phase 2 Foundational。
2. 完成 T012-T018，得到可以初始化、声明文档同步能力并正常退出的无语义 Go server。
3. 运行 `go test ./...` 和 `go vet ./...`，确认 stdout/stderr 和生命周期行为。

### Incremental Delivery

1. US1：传输和生命周期可用。
2. US2：文档 open/change/close 可用，输入版本和 UTF-16 范围稳定。
3. US3：取消、错误隔离和未知语义方法边界可用。
4. US4：日志和隐私要求可审计。
5. Phase 7：全量 race、进程和范围验证通过后，进入 Convergence 检查。
6. Phase 8：补齐异常路径、日志字段、CRLF/rangeLength 和实现后文档。
7. Phase 9：补齐并发 lifecycle、initialize 摘要和 transport 错误日志，完成后再进入后续 Tops 语义 feature。

## Notes

- `[P]` 只表示不同文件且没有未满足依赖；同一文件的任务不标记 `[P]`。
- 每个行为变更都有对应测试任务；本任务列表没有省略行为测试。
- 测试专用阻塞 handler 只存在于测试代码，不是公开 LSP 方法。
- 不创建 `internal/tokenizer/`、`internal/parser/`、`internal/ast/`、`internal/symbolindex/` 或 `internal/semantic/`。
- 不提交、创建分支或修改 `/home/carl.du/work/llvm-project`；实现和验证只在当前开发机 workspace 进行。
