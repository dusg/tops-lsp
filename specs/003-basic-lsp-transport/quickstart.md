# Quickstart：P0 基础服务和 LSP 传输

本指南用于验证已实现的 Go server 基础 LSP 传输、生命周期、文档同步、取消、错误和日志行为。

## 前置条件

- 工作区：`/home/carl.du/work/tops-lsp`
- Linux amd64
- Go：`/home/carl.du/sdk/go1.26.8/bin/go`，新 bash/fish 会话中的 `go` 已解析到该路径
- 不需要 GCU 设备、`topscc`、直接 Clang、clangd、网络或测试机 Docker；`topscc` 参数解析由后续 P2.0 功能验证。

当前 shell 已打开时可以重新加载配置：

```bash
source /home/carl.du/.bashrc
```

fish 请启动新 shell，或执行：

```fish
source /home/carl.du/.config/fish/config.fish
```

确认工具链：

```bash
cd /home/carl.du/work/tops-lsp
command -v go
go version
go env GOROOT GOPATH GOTOOLCHAIN
```

预期 `go version` 为 `go1.26.8 linux/amd64`，`GOROOT` 为 `/home/carl.du/sdk/go1.26.8`。官方 release 版本在网络恢复后仍需单独复核。

## 1. 检查 Go module 和实现入口

当前 module path 为 `tops-lsp`，进程入口为 `cmd/tops-lsp/main.go`。在工作区根目录执行：

```bash
cd /home/carl.du/work/tops-lsp
go mod tidy
go list ./...
```

第一版只使用标准库时，`go.sum` 可以不存在；如果出现第三方依赖，必须在计划或变更说明中记录原因。

## 2. 运行单元和组件测试

```bash
cd /home/carl.du/work/tops-lsp
go test ./...
go test -race ./...
go vet ./...
```

重点包测试可以单独执行：

```bash
go test ./internal/transport -run Test
go test ./internal/document -run Test
go test ./internal/server -run Test
```

实际重复执行时优先加入 workspace task，避免手写路径漂移。

## 3. 运行进程级 LSP 测试

```bash
cd /home/carl.du/work/tops-lsp
go test ./integration -run TestLSPProcess -v
```

进程级测试必须通过管道启动 `cmd/tops-lsp`，并检查：

1. 分片 header/body 和连续多帧能够正确解析。
2. `initialize` 返回最小 capabilities，且不声明语义 provider。
3. didOpen/didChange/didClose 保存正确文本和版本。
4. shutdown 后 exit 正常退出；未 shutdown 的 exit 使用异常退出状态。
5. stdout 只有协议帧，日志在 stderr。
6. 非法消息、未知方法、非法通知、EOF 和取消不会破坏后续会话。

## 4. 规格场景覆盖

实现时将 `spec.md` 中的场景映射到测试：

- `T-TRANSPORT-*`：`internal/transport` 和进程级 framing 测试。
- `T-LIFE-*`：`internal/server` 状态机和子进程退出状态测试。
- `T-SYNC-*`：`internal/document` UTF-16、版本和变更原子性测试。
- `T-CANCEL-*`：测试专用阻塞 handler、request registry 和 `-race` 测试。
- `T-ERROR-*`：协议 code、ID、通知无 response 和内部错误隔离测试。
- `T-LOG-*`：结构化字段、stderr、脱敏和错误关联测试。
- `T-BOUNDARY-001`：未知语义方法返回 `MethodNotFound`，代码范围检查排除 Tops 语义和 clangd。

## 5. 完成检查

```bash
cd /home/carl.du/work/tops-lsp
go test ./...
go test -race ./...
go vet ./...
git diff --check
git diff --name-only
```

`git diff --name-only` 只能包含 `go.mod`、Go server 基础目录、测试、测试数据和本 feature 计划相关文件；不得出现 `llvm-project`、clangd 扩展、Tops semantic/parser 或 TypeScript 实现文件。

当前实现已验证 Go `1.26.8`、`go build ./...`、`go test ./...`、`go test -race ./...` 和 `go vet ./...`。这里的 quickstart 不要求构建或运行 clangd；进入后续 Tops 语义计划前，先完成本 feature 的 Convergence 任务。
