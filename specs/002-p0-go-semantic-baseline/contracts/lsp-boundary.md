# 契约：P0 Go 服务器、VS Code 客户端与 Clang oracle 边界

## 契约范围

本契约定义 P0 计划中 Go server、TypeScript VS Code client、Clang oracle、CompilationContext 和用户之间的责任边界。它不选择 Go LSP library、parser library、package 名称、client setting 名称或扩展消息格式；这些选择必须在后续实现计划中单独记录。

## 端到端路径

| 步骤 | Owner | 输入 | 输出 | 失败行为 |
| --- | --- | --- | --- | --- |
| 用户打开/编辑文件 | VS Code client | URI、文本变化、workspace、配置 | 标准 LSP 文档通知和配置通知 | server 无法启动时展示错误并记录脱敏日志 |
| 选择 CompilationContext | Go server | compile database、workspace fallback、文件 URI | context、TargetProfile、来源、version | 缺失/冲突字段生成 context diagnostic；不猜 profile |
| tokenizer/parser | Go server | 文本、document version、language mode | tokens、recoverable AST、ranges | 未完成输入保留结构，标记 `analysis-degraded` |
| symbol index | Go server | AST、headers、include roots、宏条件 | declaration/definition/reference/member index | header/context 变化使 index stale；不把未知当作无条件可用 |
| semantic | Go server | AST、index、context、profile | diagnostics、completion/hover/navigation facts | unsupported target feature、semantic error 由稳定 code 表达 |
| LSP encode/serve | Go server | semantic result、request id、context version | standard LSP response/notification | cancel/stale 结果不得覆盖新结果；超时可观测 |
| 展示和命令 | TypeScript client | LSP responses、用户命令 | 编辑器反馈、状态、日志、restart | client 不重新判断语义，只展示/转发 |
| differential check | P0/P1 validation only | Go result、Clang command/output、同一 fixture/context | 差异记录和修复任务 | Clang 失败不转为 server runtime fallback |

## Go server ownership

- Go server 拥有 tokenizer、parser、AST、symbol index、CompilationContext、TargetProfile、semantic diagnostics、completion、hover、definition、references、documentSymbol 和 LSP protocol behavior。
- semantic 是唯一拥有 Tops/C++ target legality、host/device call boundary、address space、attribute argument、vector type、builtin visibility 和 target feature gate 判断的层。
- LSP 层只能管理协议生命周期、request/cancel/version/stale 和结果编码；它不能复制 semantic rule。
- Go server 不调用 clangd，也不把 Clang AST/diagnostic 作为隐藏运行时后端。

## TypeScript client ownership

- 激活扩展、启动/停止/重启 Go server、发送 workspace configuration、处理文档同步、显示 diagnostics/completion/hover/navigation、展示状态和日志、执行用户命令。
- 不拥有 target profile 选择规则、CompilationContext precedence、语义合法性、诊断 severity、类型可用性或 unsupported 判断。
- 对 server 返回的 `missing-compilation-context`、`invalid-compilation-context`、`unsupported-target-feature`、`analysis-degraded`、`server-unavailable`、`request-cancelled`、`stale-result` 只负责展示、引导配置或触发重试。

## 最小标准 LSP surface

| Operation | Direction | P0/P1 requirement |
| --- | --- | --- |
| `initialize` / `shutdown` / `exit` | 双向生命周期 | 声明 server version、可选能力和 client capability；失败可解释 |
| `textDocument/didOpen` / `didChange` / `didClose` | client -> server | 文本版本和 context 变化触发增量解析/失效 |
| `textDocument/publishDiagnostics` | server -> client | 每个诊断有 range、severity、source、稳定 code、context version；stale 不覆盖最新 |
| `textDocument/completion` | client -> server | 返回关键字、attribute、类型、builtin、header/API 候选；结果携带 context/version 关联 |
| `textDocument/hover` | client -> server | 说明标准/Tops/target-dependent 差异、证据状态和限制 |
| `textDocument/definition` / `references` / `documentSymbol` | client -> server | 支持当前文档、可解析 headers、宏/条件索引；不完整时返回受限结果和原因 |
| `workspace/didChangeConfiguration` | client -> server | 使受影响 context/AST/index/diagnostic stale，重新分析并清理旧结果 |

P0 不新增非标准消息。若后续需要扩展消息，必须使用版本化命名空间，记录 owner、payload、错误、日志、兼容性和迁移/拒绝方案。

## CompilationContext 边界

- 有效文件级 `compile_commands` 条目优先于 workspace settings。
- workspace fallback 不能覆盖数据库中的明确 target、standard、input mode、include、macro 或 device flag。
- `__GCU_ARCH__` 和 `__EFGCU_ARCH__` 必须从实际 target/pass/header 证据得到；没有预期宏时只能是 missing/partial/invalid。
- context version 变化使旧 AST/index/diagnostic/LSP result 失效；client 不得缓存并展示旧语义结论。

## 错误、取消与 stale

| Code | Owner | 用户可见行为 | 日志字段 |
| --- | --- | --- | --- |
| `missing-compilation-context` | context resolver | 提示缺少 compiler/target/standard/include，并进入受限模式 | 脱敏 URI、workspace、来源、缺失字段 |
| `invalid-compilation-context` | context resolver | 显示冲突参数摘要和修复方向 | 参数来源、冲突字段、target、duration |
| `unsupported-target-feature` | semantic | 说明当前 profile 不支持的能力，不降级成普通 C++ 成功 | profile、capability ID、range、context version |
| `analysis-degraded` | parser/index/semantic | 保留可恢复结果并说明受限范围 | parse state、header/context version、duration |
| `server-unavailable` | LSP/client lifecycle | 显示启动/崩溃/重启动作和日志位置 | operation、restart reason、server version |
| `request-cancelled` | LSP | 丢弃取消请求结果，不显示旧结果替代新结果 | request id、operation、context version |
| `stale-result` | LSP/context | 不发布或显式标记旧结果，最新结果优先 | request id、old/new version、operation |

日志不得记录密钥、完整源代码或无关用户数据。diagnostic message 可引用必要的 symbol、profile 和位置，但不复制无关源码片段。

## 兼容性

- 优先标准 LSP 方法和类型；新增可选能力不得使旧 client 初始化失败。
- public setting/command/message 变更必须记录版本、owner、payload、错误和迁移/拒绝行为。
- 删除或改变 diagnostics code、profile 字段或 context precedence 必须更新 contract tests 和迁移说明。
- Clang oracle 版本变化可能改变差分输出，但不改变 Go server 的 runtime ownership；差异必须记录为证据/修复任务。
