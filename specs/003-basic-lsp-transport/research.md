# 研究记录：P0 基础服务和 LSP 传输

## 研究范围

本阶段只确认当前工程状态、Go 工具链、已有契约和实现边界。没有读取或修改 `llvm-project` 的 Tops 语义实现，因为本功能明确排除具体语义和 clangd。

## 已确认事实

| 主题 | 事实 | 证据/检查 | 对计划的影响 |
| --- | --- | --- | --- |
| 当前 feature | `.specify/feature.json` 指向 `specs/003-basic-lsp-transport` | 读取 `.specify/feature.json` | 所有计划产物写入当前目录 |
| Go 源码状态 | `tops-lsp` 当前没有 `.go` 文件和 `go.mod` | 文件搜索 | 需要从 Go module 和最小进程入口开始 |
| 任务入口 | 当前没有 `.vscode/tasks.json` | 文件搜索 | 计划记录 Go 命令；实现后再补匹配 task |
| 旧设计边界 | `001` 的 LSP 契约规定 Go server 拥有协议行为，客户端拥有生命周期和展示；`002` 计划要求 Go 语义不依赖 clangd | 读取现有规格和契约 | 本 feature 只细化基础传输，不改变既有边界 |
| 系统 Go | `/usr/local/go/bin/go` 为 `go1.22.5 linux/amd64` | 直接执行 `go version` | 不将系统旧版作为本 feature 的默认工具链 |
| 用户旧 Go | `/home/carl.du/sdk/go1.24.2/bin/go` 为 `go1.24.2 linux/amd64` | 直接执行 `go version` | 保留，不覆盖 |
| 选定 Go | `/home/carl.du/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.8.linux-amd64/bin/go` 可执行；复制后 `/home/carl.du/sdk/go1.26.8/bin/go` 为 `go1.26.8 linux/amd64` | 直接执行 `go version` | 计划使用 Go 1.26.8；GOROOT 已验证为 `/home/carl.du/sdk/go1.26.8` |
| PATH | bash 和 fish 配置已将 `/home/carl.du/sdk/go1.26.8/bin` 放在 PATH 前面 | 读取并修改 `/home/carl.du/.bashrc`、`/home/carl.du/.config/fish/config.fish`，新 shell 验证 | 新终端默认使用 Go 1.26.8 |
| 官方版本复核 | `https://go.dev/dl/` 内容抓取失败，`https://go.dev/VERSION?m=text` 请求超时 | 网络检查结果 | 不能把本机最高版本写成已完成网络确认的“官方最新”；恢复网络后复核 |

## 设计决策

### 使用标准库完成第一版

当前仓库没有 Go module 或 LSP 依赖。基础范围只需要 JSON 编解码、字节流 framing、并发状态、取消、时间和日志；Go 标准库已覆盖这些能力。因此第一版使用 `encoding/json`、`bufio`、`io`、`sync`、`context`、`time` 和 `log/slog`，不引入第三方 LSP library。

这不是对未来语义层的长期依赖承诺。后续若引入协议库，必须先比较其 framing、错误、取消、版本和日志行为，不能让库默认行为改变本规格。

### 使用 stdio，不提供备用传输

规格明确要求 LSP over stdio。进程入口只绑定 stdin/stdout，日志写 stderr。TCP、WebSocket、HTTP 和自定义二进制传输不进入本阶段，这可以让进程级测试直接检查字节流和退出状态。

### 文档状态只保存文本和版本

`DocumentState` 不保存 AST、符号、诊断或 target context。这样可以证明文档同步本身正确，也避免在未实现 Tops 语义时产生隐含 fallback。

### 取消使用内部测试 handler 验证

本功能公开请求很短，单靠 `initialize`/`shutdown` 无法可靠覆盖运行中取消。因此测试提供可控阻塞的内部 handler，验证 request registry、终态竞争和无迟到响应；该 handler 不注册为公开 LSP 方法。

## 未决项和处理方式

| 项目 | 当前状态 | 处理方式 |
| --- | --- | --- |
| Go module path | 仓库没有既有 module，尚未有正式命名约定 | 在实现任务开始时确认仓库命名；内部包不导出，因此不影响契约 |
| 官方 Go 最新版本 | 网络不可达，未完成官方发布页复核 | 暂以已验证的 `go1.26.8` 构建；网络恢复后在实现前再次检查 |
| 第三方 LSP library | 当前没有依赖 | 首版不引入；只有需求扩大时重新评估 |
| 日志文件配置 | 规格只要求 stderr 或受控目标 | 第一版固定 stderr，文件日志配置留给后续客户端契约 |
| TypeScript 启动命令 | 当前没有客户端源码 | 用子进程集成测试模拟 client；不在此 feature 创建扩展 |

## 研究结论

本功能可以在无 GCU、无 clangd、无网络的开发机上独立实现和测试。实施顺序必须先固定字节帧和协议模型，再实现 server 状态和文档变更；取消、错误和日志只能建立在这两个边界之上。Go 1.26.8 是当前本机已验证基线，但网络恢复后的官方版本检查仍是实现前的工具链复核项。
