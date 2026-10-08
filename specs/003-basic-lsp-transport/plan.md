# 实施计划：P0 基础服务和 LSP 传输

**分支**：`003-basic-lsp-transport` | **日期**：2026-10-08 | **规格说明**：[spec.md](spec.md)

**输入**：来自 `specs/003-basic-lsp-transport/spec.md` 的功能规格说明。

**交付边界**：本计划安排一个 Go server 基础实现。实现只覆盖 Go 项目骨架、LSP over stdio、生命周期、文档同步、请求取消、基础错误和结构化日志；不实现 Tops C++ 语义，不修改 `llvm-project`，不扩展或调用 clangd，不创建 TypeScript 代码。

## 摘要

本功能从当前只有规格文档的 `tops-lsp` 工程开始，建立一个可独立测试的 Go 命令行 server。实现顺序按传输边界、协议模型、server 状态、文档存储、取消与错误、日志和进程级测试展开。所有结果都保持内存状态，后续语义层可以在不改变基础 LSP 契约的情况下接入。

主要交付物：

- Go module 和 `cmd/tops-lsp` 进程入口。
- 标准 JSON-RPC 2.0 over stdio 帧读取/写入。
- `initialize`、`shutdown`、`exit` 和 server 生命周期状态机。
- `textDocument/didOpen`、`textDocument/didChange`、`textDocument/didClose` 的增量文档存储。
- `$/cancelRequest`、基础 JSON-RPC/LSP 错误和单响应保证。
- 结构化日志、stdout 协议纯净性和敏感数据脱敏。
- 单元、组件和子进程集成测试；不依赖 Tops compiler、clangd、GCU 设备或网络。

## 技术上下文

**语言/版本**：Go `1.26.8`。开发机已验证 `/home/carl.du/sdk/go1.26.8/bin/go` 可执行；bash 和 fish 的新会话已将其置于 `PATH` 首位。`/usr/local/go/bin/go` 为 `1.22.5`，用户目录旧安装为 `1.24.2`。Go 官方版本端点在本次检查中超时，因此 `1.26.8` 是当前本机已验证的最高版本，不把它表述为已完成网络复核的官方最新版本。

**主要依赖**：首版使用 Go 标准库：`encoding/json`、`bufio`、`bytes`、`io`、`sync`、`context`、`time`、`os`、`log/slog` 和测试相关标准库。初始实现不引入第三方 LSP 或日志库；如后续确需依赖，必须在计划变更中说明原因和兼容性。

**存储**：进程内存中的文档表、活动请求表和会话状态；不引入数据库、磁盘文档缓存或外部服务。

**测试**：`go test ./...`、`go test -race ./...`、`go vet ./...`；传输和 server 测试使用 `io.Reader`/`io.Writer` 与子进程，不依赖真实编辑器或 GCU 设备。

**目标平台**：Linux amd64 开发机；server 通过 stdin/stdout 与 VS Code LSP client 通信，日志默认写入 stderr。

**项目类型**：Go 命令行语言服务器基础设施。

**性能目标**：本阶段不建立语义请求的延迟目标。传输读取不能因半帧阻塞错误地执行请求；文档变更、错误和取消处理必须在测试中保持确定性，不得出现 goroutine 泄漏或请求无限等待。

**约束**：

- stdout 只能输出完整 LSP 协议帧；任何日志、panic 和调试输出必须离开 stdout。
- 文档同步使用增量模式，位置使用 UTF-16，版本必须严格递增。
- 请求最多产生一个终态响应；取消不能覆盖其他请求或文档状态。
- 通知错误不返回 JSON-RPC response；请求错误使用稳定标准 code。
- 本功能不创建 tokenizer、parser、AST、symbol index 或 semantic 目录。
- 不解析 `compile_commands.json`，不选择 GCU target，不读取 Tops headers，不使用 clangd。
- 所有测试使用合成 URI 和文本，不写入真实用户源代码或敏感配置。

**规模/范围**：1 个 Go server 进程、1 个内存文档表、1 个活动请求表、7 个基础错误名、7 个核心 LSP 方法/通知和至少 18 个规格测试场景。

## 宪法检查

*门禁：阶段 0 研究前通过，阶段 1 设计后再次检查。*

- [x] **LSP 契约优先**：`contracts/lsp-transport.md` 记录 stdio framing、方法、响应、错误、取消、日志和兼容性；没有新增未记录的扩展消息。
- [x] **Go/TypeScript 分层边界**：Go server 拥有协议、生命周期和文档状态；本阶段不创建 TypeScript 代码，客户端只作为 LSP 对端和进程管理者。
- [x] **Tops C/C++ 语义保真**：本阶段不判断 Tops 语义，不伪造普通 C++ fallback，也不修改 Tops 工具链；未知语义请求返回 `MethodNotFound`。
- [x] **行为优先的验证**：测试计划覆盖有效、非法、不完整、取消、EOF、错误和 stdout 污染路径。
- [x] **可观测且兼容地演进**：日志字段、脱敏规则、错误 code、退出状态和旧客户端行为已记录。
- [x] **没有违规项**：没有复杂度例外；标准库优先和按边界拆分是本功能的最小实现方案。

## 项目结构

### 文档（本功能）

```text
specs/003-basic-lsp-transport/
├── spec.md                         # 功能规格
├── plan.md                         # 本文件
├── research.md                     # 阶段 0 的事实和设计决策
├── data-model.md                   # 会话、文档、请求和错误状态
├── quickstart.md                   # Go 环境、构建和验证入口
├── contracts/
│   ├── lsp-transport.md            # LSP stdio 和 server/client 契约
│   └── file-boundaries.md          # Go 文件归属和禁止越界范围
└── checklists/requirements.md      # 规格质量清单
```

### 源代码（实现阶段创建）

```text
go.mod
cmd/tops-lsp/main.go
internal/
├── transport/stdio.go              # Content-Length 帧读取和写入
├── protocol/messages.go            # JSON-RPC/LSP 基础消息模型
├── protocol/errors.go              # 标准错误 code 和错误响应
├── server/server.go                # 生命周期、分发、取消和单响应
├── document/store.go               # URI、文本、版本和 UTF-16 增量变更
└── logging/logger.go               # 结构化日志、关联 ID 和脱敏

testdata/lsp/
├── valid-session.jsonl              # 合成会话输入/输出样例
├── invalid-messages.jsonl          # 协议和参数错误样例
└── document-changes.jsonl          # 增量文本变更样例

internal/transport/stdio_test.go
internal/protocol/errors_test.go
internal/server/server_test.go
internal/document/store_test.go
internal/logging/logger_test.go
integration/lsp_process_test.go      # 子进程 stdout/stderr/退出状态测试
```

**结构决策**：使用单一 Go module 和 `cmd` + `internal` 结构。`transport` 只处理字节帧，`protocol` 只处理协议模型，`server` 只处理生命周期和分发，`document` 只处理文本状态，`logging` 只处理可观测性。实现阶段可拆分同一目录内的文件，但不能让 `main.go` 直接实现协议或文档算法。

## 阶段执行

### 阶段 0：工程准备和研究

1. 在 `/home/carl.du/work/tops-lsp` 确认 Go 版本为 `go1.26.8`，确认 `go env GOROOT` 指向 `/home/carl.du/sdk/go1.26.8`。
2. 创建 `go.mod`，模块路径使用仓库约定；当前没有既有 Go module，因此在任务实施前确认最终 module path，并保持内部包不依赖该路径。
3. 运行 `go list ./...` 或等价最小命令，确认空骨架的包发现行为；此时不添加语义包。
4. 阅读 `research.md`、`data-model.md` 和两个契约文件，将实现范围与规格逐项对照。

**阶段 0 输出**：Go module 初始化、依赖决策记录和研究结论；不修改 `llvm-project`。

### 阶段 1：传输和协议模型

1. 在 `internal/transport/stdio.go` 实现 header 读取、`Content-Length` UTF-8 字节计算、分片读取、多帧读取、EOF 和写帧。
2. 在 `internal/protocol/messages.go` 定义 request、response、notification、ID、initialize、shutdown、exit 和文档同步所需的最小 JSON 模型。
3. 在 `internal/protocol/errors.go` 固化 `ParseError`、`InvalidRequest`、`MethodNotFound`、`InvalidParams`、`InternalError`、`ServerNotInitialized` 和 `RequestCancelled` 的 code 与响应结构。
4. 为传输和协议层添加不依赖 server 的测试，覆盖非法 header、非 ASCII body、分片/合并帧、ID 保留和通知无响应。

**阶段 1 输出**：可独立测试的协议/传输组件；不解析文档语义。

### 阶段 2：生命周期和文档同步

1. 在 `internal/server/server.go` 实现启动未初始化、initialized、shutdown pending、exited 状态及合法/非法消息路由。
2. 实现 `initialize` 的最小 capabilities，只声明 open/close 和 incremental document sync，不声明语义 provider。
3. 在 `internal/document/store.go` 实现 `didOpen`、按数组顺序应用 `didChange`、严格版本校验、UTF-16 range 转换和 `didClose`。
4. 明确通知错误的处理：保留最近一次有效 `DocumentState`，记录拒绝事件，不发送 JSON-RPC response。
5. 实现 `shutdown` 返回 `null`、`exit` 正常/异常退出状态和 stdin EOF 清理。

**阶段 2 输出**：不依赖 Tops 语义的可运行 server 核心；单元测试覆盖文档和状态机。

### 阶段 3：取消、错误隔离和日志

1. 在 server 中登记活动 request，使用 `context.Context` 或等价取消信号处理 `$/cancelRequest`。
2. 保证取消、成功、内部错误三种终态最多只能选一个；取消不改变文档和会话状态。
3. 对请求分派边界捕获内部异常，向客户端只返回 `InternalError` 摘要，日志保留关联 ID 和受控诊断。
4. 在 `internal/logging/logger.go` 定义结构化字段、日志级别、文档 ID 脱敏和 stderr 输出；禁止完整 payload、源文本、凭据和密钥。
5. 增加日志测试，检查必需事件、字段、stderr 位置和敏感值不出现。

**阶段 3 输出**：可取消、可恢复、可审计的基础 server。

### 阶段 4：进程级集成验证

1. 用 `os/exec` 启动 `cmd/tops-lsp`，通过管道发送完整和分片 LSP 帧。
2. 验证合法 initialize -> didOpen -> didChange -> didClose -> shutdown -> exit 会话及正常退出。
3. 验证非法顺序、未知方法、非法 JSON、非法通知、EOF、未完成 shutdown 的 exit 和 stdout 污染。
4. 使用测试专用阻塞 handler 验证取消和无迟到响应；该 handler 不作为公开 LSP 方法。
5. 执行 `go test ./...`、`go test -race ./...` 和 `go vet ./...`，记录失败场景和修复结果。

**阶段 4 输出**：规格中的 LSP 基础场景全通过，且没有 Tops 语义或 clangd 依赖。

### 阶段 5：计划验收

1. 对照 `spec.md` 的 FR-001 至 FR-023 和 SC-001 至 SC-009 检查实现范围。
2. 检查 `git diff --name-only`，确认变更只在 Go module、基础 server、测试和测试数据范围内。
3. 重新执行 Constitution Check，确认没有新增公开消息、设置、语义能力或日志隐私违规。
4. 将实际验证命令和已知缺口补入实现变更说明；只有基础能力通过后才进入 Tops 语义计划。

## 交付与验证

| 产物 | 责任 | 主要验证 | 完成条件 |
| --- | --- | --- | --- |
| `go.mod` 和 `cmd/tops-lsp/main.go` | Go 工程入口 | `go test ./...`、进程启动 | 无 GCU/clangd/网络依赖即可启动并退出 |
| `internal/transport/` | stdio framing | 分片、合并、UTF-8 字节长度、stdout 纯净性 | 每帧边界正确，非法帧有明确错误 |
| `internal/protocol/` | JSON-RPC/LSP 模型 | 错误码和 ID 测试 | 请求响应结构稳定，通知无 response |
| `internal/server/` | 生命周期和分发 | 状态机、单响应、取消、EOF | 正常/异常生命周期行为符合契约 |
| `internal/document/` | 文档同步 | UTF-16、多个 change、版本和 close | 失败变更不破坏最近有效状态 |
| `internal/logging/` | 日志和隐私 | 字段、事件、stderr、脱敏测试 | 不记录完整源代码、payload 或凭据 |
| `integration/lsp_process_test.go` | 跨组件行为 | 子进程、退出状态、stdout/stderr | 规格场景可重复执行 |
| `contracts/*.md` | 设计契约 | 代码评审和实现对照 | 文件责任、协议、错误和兼容性无歧义 |

## 依赖、风险与控制

| 风险 | 控制措施 | 触发状态 |
| --- | --- | --- |
| 官方最新 Go 版本无法联网确认 | 以本机已验证 `go1.26.8` 为实现基线；恢复网络后复核官方 release，若版本变化再更新 `go.mod` 和计划 | `toolchain-review-needed` |
| 当前仓库没有 Go module | 先建立最小 module 和标准库测试；module path 在实现前记录 | `blocked` 直到确认 |
| stdout 混入日志导致 VS Code 断连 | 集中封装 writer，进程级测试逐字节检查 stdout | `transport-invalid` |
| UTF-16 位置和 UTF-8 字节混用 | `document` 层单独测试 BMP、非 BMP、多行、插入、删除和替换 | `document-sync-invalid` |
| 取消与响应竞态产生重复响应 | request state 使用单一终态转换和并发测试，运行 `-race` | `request-state-invalid` |
| 非法通知无法返回错误 | 统一日志事件和状态保留规则；不伪造 notification response | `notification-rejected` |
| 日志泄露源文本或凭据 | 默认不记录 payload，脱敏函数单测覆盖 URI、路径和值 | `privacy-blocked` |
| 后续实现误加语义 fallback | 未知语义方法测试固定为 `MethodNotFound`，变更范围检查排除 clangd/Tops 目录 | `scope-violation` |

## 复杂度记录

无宪法违规项。单一 Go module、标准库优先和按责任目录拆分已经是满足协议、取消、日志和测试要求的最小结构；不引入 transport abstraction、插件机制或语义接口。
