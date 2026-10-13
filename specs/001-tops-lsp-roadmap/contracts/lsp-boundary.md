# 契约：Go 服务器与 VS Code 客户端 LSP 边界

## 契约范围

本契约定义 roadmap 中服务器、客户端和用户之间的职责边界。它不锁定具体 Go LSP 库、TypeScript 扩展框架或自定义设置名；这些选择必须在后续实现计划中单独记录。

## 端到端路径

| 步骤 | 责任方 | 输入 | 输出 | 失败行为 |
| --- | --- | --- | --- | --- |
| 1. 用户打开/编辑文件 | VS Code 客户端 | 文档 URI、文本变化、工作区 | 标准 LSP 文档通知 | 无法启动服务器时显示启动错误并记录日志 |
| 2. 建立最小上下文 | 服务器 | 文档 URI、workspace 设置和可用的局部宏/include 信息 | `ContextSeed`，状态为 `resolved`、`partial`、`missing` 或 `invalid` | 缺失字段生成可定位诊断，不猜测目标；parser/index 可以在 partial 状态继续恢复 |
| 3. 归一化编译命令 | 服务器 | compile database 或工作区设置中的 `topscc`/Clang argv | `CompilerInvocation` 和 target candidates | P2.0 负责 wrapper、多目标和冲突参数；失败时保留 partial/invalid 状态 |
| 4. 形成文件级编译上下文并分析 | 服务器 | 文档、ContextSeed、`CompilationContext`、`TargetProfile` 和 Tops headers | 语法/语义结果、符号、候选项 | P2.1 负责文件级 context 选择和合并；解析失败保留可恢复结果，标记受限分析 |
| 5. 返回协议结果 | 服务器 | 标准 LSP 请求 | 当前已实现的 diagnostics、completion、hover、definition、references、symbols | 取消、超时和 stale 结果按请求生命周期处理 |
| 6. 展示/执行命令 | VS Code 客户端 | LSP 响应、用户命令 | 编辑器反馈、状态、日志、重启 | 客户端不重新判断语义，只展示服务器结果或转发命令 |

## 标准 LSP 能力

| 能力 | 方向 | 阶段/状态 | 最小要求 |
| --- | --- | --- | --- |
| `initialize` / `shutdown` / `exit` | 双向生命周期 | P1.1 `supported` | 声明服务版本和已经实现的标准能力；失败可解释 |
| `textDocument/didOpen`、`didChange`、`didClose` | 客户端到服务器 | P1.1 `supported` | 增量文本和上下文变化后触发重新分析 |
| `textDocument/publishDiagnostics` | 服务器到客户端 | P1.2/P1.4 `supported` | 诊断必须带范围、严重级别、来源、稳定 code 和 stale 处理 |
| `textDocument/completion`、`textDocument/hover` | 客户端到服务器 | P1.5 `planned` | 只有对应 handler、语义规则和契约测试完成后声明；结果说明标准/Tops/目标差异 |
| `textDocument/definition`、`references`、`documentSymbol` | 客户端到服务器 | P1.3 `supported` | 支持当前新鲜快照、按需 header 和可解析宏上下文的导航 |
| `workspace/didChangeConfiguration` | 客户端到服务器 | P3.2 `planned` | 配置变化使相关 context 失效并重新分析；未实现前不声明对应行为 |

未来 LSP 方法（例如 semantic tokens、rename、code action、hierarchy、formatting、inline completion 和 notebook synchronization）属于候选能力，不属于 P1.7 的 capability 声明。每项必须先有 owner、上下文前提、标准结果、错误/取消/stale 行为和契约测试，完成后单独标记 `supported`。

## 配置契约

roadmap 只规定语义字段，不预先锁定设置名：

- 编译数据库目录和条目选择规则；优先读取 `arguments`，没有时解析 `command`。
- Go server 可执行文件或启动方式。
- `topscc` 用户编译器路径和版本；直接 `clang`/`clang++` 兼容入口。
- Clang 离线对照工具路径，仅用于兼容性和对照比较，不参与 Go server 运行时语义。
- `topscc` wrapper 参数的解析状态、未识别参数和多架构分析策略。
- C++ 标准和 Tops 输入模式。
- target profile 或 target triple。
- Tops/Clang include 根。
- 预定义宏和必要的 device 编译参数。
- 日志级别和日志输出位置。

编译数据库中的明确参数优先于工作区 fallback；任何覆盖行为必须显示来源和原因。

## 错误与取消

- `missing-compilation-context`：缺少编译器、目标、include 或标准信息；客户端应提示配置动作。
- `invalid-compilation-context`：参数冲突或工具拒绝；客户端应显示原始工具错误的摘要和日志位置。
- `invalid-topscc-arguments`：`topscc` wrapper 参数冲突、值缺失或版本不匹配；客户端应指出参数来源和修复方向。
- `ambiguous-target-context`：命令展开出多个目标且没有选择策略；客户端不得让 server 静默选择一个目标。
- `unsupported-target-feature`：语法存在但当前 profile 不支持；不得降级成普通标准 C++ 成功结果。
- `analysis-degraded`：只能进行部分解析；结果标记受限范围，不能伪装为完整语义。
- `server-unavailable`：服务器未启动、崩溃或重连；客户端提供重启命令。
- `request-cancelled` / `stale-result`：取消或上下文版本已变化的结果不能覆盖最新诊断。

## 可观测性与隐私

日志至少能关联操作类型、文档脱敏标识、目标 profile、上下文来源、耗时、错误 code 和重试/重启原因。日志不得记录密钥、完整源代码或无关用户数据；诊断消息可以引用必要的符号和位置，但不得复制无关源码片段。

## 兼容性

- 优先使用标准 LSP 方法和数据类型。
- 扩展消息、设置和命令必须使用版本化命名空间，并记录 owner、payload、错误行为、日志和迁移方案。
- 服务器新增可选能力不得使旧客户端初始化失败。
- 删除或改变公开能力前，必须提供迁移说明或明确拒绝理由，并更新契约测试。
