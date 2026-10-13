# 契约：Roadmap 阶段门禁

## 门禁记录格式

每个阶段必须记录：

- `phase`：`P0`、`P1.1` 至 `P1.7`、`P2.0` 至 `P2.5`、`P3.1` 至 `P3.4`、`P4.0` 至 `P4.3`。
- `objective`：用户可见目标。
- `entry_evidence`：进入阶段前已有的源码、测试、工具或契约证据。
- `scope`：本阶段承诺和明确不承诺的能力。
- `dependencies`：目标 profile、编译上下文、服务器、客户端或契约依赖。
- `validation`：正向、负向、边界和失败路径验证。
- `exit_criteria`：可复核的完成条件。
- `fallback`：验证失败时的 `blocked`、`unsupported` 或延期处理。
- `compatibility`：对已有标准 C++、LSP、设置和命令的影响。
- `owner`：服务器、客户端、共享契约或文档维护责任。
- `publication_path`：最终 roadmap 文档路径，固定为 `doc/tops-cpp-language-server-roadmap.md`。

## 阶段门禁

| 阶段 | 入口 | 范围 | 验证 | 退出条件 | 失败处理 |
| --- | --- | --- | --- | --- | --- |
| P0 Go 服务器语义基线 | 当前 headers、Clang 属性、driver、tests 和状态记录可定位 | 能力矩阵、TargetProfile、ContextSeed/CompilationContext 边界、Go 层所有权和验证材料 | 源码/测试路径、driver 命令族和 P1 场景审查 | 每个 P1 子阶段有证据、正反例、目标前提、Go 验证动作和对照动作 | 未验证能力标记 `candidate`/`blocked`；不引入 clangd 运行依赖 |
| P1.1 基础服务和 LSP 传输 | P0 文档契约稳定 | Go module、stdio framing、生命周期、文档同步、取消、基础错误和日志；不声明语义 provider | transport/server/document 单测、进程测试、`go test ./...` | initialize/shutdown/exit、增量同步、取消和 stdout 纯净行为可重复 | 协议错误按标准错误返回；语义请求保持 `MethodNotFound` |
| P1.2 tokenizer/parser | P1.1 文档输入稳定，ContextSeed 接口明确 | C++11/14/17 基础 grammar、Tops spelling、宏条件结构、错误恢复和 parser diagnostics | parser 正/负/不完整/宏条件/位置测试 | 返回可恢复 ParseResult；不完整输入不清空已识别结构 | `analysis-degraded`/`unsupported-syntax`，不猜测目标或宏值 |
| P1.3 AST/index/navigation | P1.2 AST 和 ContextSeed 可用 | AST 元数据、符号索引、按需 header、definition/references/documentSymbol | index snapshot、header、条件、版本和 LSP contract 测试 | 新鲜快照和部分索引可查询；旧结果不覆盖新结果 | 缺失 header/unknown 条件保留 `analysis-degraded` |
| P1.4 通用 semantic baseline | P1.2/P1.3 输出稳定，ContextSeed 可表达受限状态 | 声明、作用域、名称查找、基础类型、通用 Tops host/device、地址空间和基础向量规则 | Go semantic 正/负/不完整测试和 Clang 离线差异记录 | 每条规则有稳定 code/range/severity；不把未实现完整语义写成 supported | 未覆盖模板实例化、完整重载和常量求值保持 `candidate`/`blocked` |
| P1.5 核心语言请求 | P1.3 导航和 P1.4 semantic baseline 稳定 | diagnostics、completion、hover 以及已完成导航的核心用户闭环 | Go/LSP/client 适用契约测试 | 只声明当前已实现方法；空结果、取消、stale 可复现 | 缺少 context 时返回标准错误或 `analysis-degraded` |
| P1.6 C++11/14/17 grammar | P1.2 parser 和 P1.3 AST/source range 契约稳定 | 已列 C++11/14/17 grammar 补齐；新增语法接入 P1.4 已实现规则；不包含 C++20/23 | 按标准版本分类的正向、负向、不完整、AST/recovery 和 Clang 离线对照测试 | 每类已列语法都有测试；基础 grammar 不再以 `unsupported-syntax` 作为默认处理 | C++20/23 保持独立候选；未覆盖语义保持 `candidate`/`analysis-degraded` |
| P1.7 LSP 协议收敛 | P1.1-P1.5 的已实现方法可枚举 | framing、生命周期、同步、push diagnostics、当前导航、取消、错误和已实现 capability | stdio、版本、UTF-16/CRLF、取消、stale 和 capability 契约 | capability 只覆盖已实现方法；候选方法不提前注册 | 未实现方法返回标准 `MethodNotFound` 或不声明 capability |
| P2.0 topscc driver 归一化 | P1.1 生命周期和 ContextSeed 边界稳定 | topscc/直接 Clang argv、wrapper 默认值、response file、多目标候选和参数错误；不负责最终文件级 context 合并 | `topscc --version`、`--dryrun`、静态 argv、冲突/缺失/多目标材料 | CompilerInvocation 和候选 context 状态可审计；runtime 不启动 topscc | `partial`/`invalid`，不静默选择目标或套用默认 |
| P2.1-P2.5 目标感知语义 | P2.0 driver 结果可复现，P1.4 semantic baseline 可消费上下文 | P2.1 形成文件级 TargetProfile/CompilationContext；后续阶段处理 include/macro context、完整宏展开、架构属性、目标诊断和多目标矩阵 | profile 正/负/边界、header/source mapping、stale 测试 | 目标不匹配可解释；客户端与服务器不矛盾；P4.2 可按已完成切片回归 | `target-dependent`/`unsupported`/`analysis-degraded` |
| P3.1-P3.4 VS Code 工作流 | P1.1 生命周期和最小配置契约稳定；完整 target 配置等待 P2.1，语言集成按 P1.5 适用 | 激活、配置、状态、日志、重启、单根/多根和当前已实现方法集成 | client integration、配置变化、重连和旧诊断清理 | client 与当前 server capability 一致；不复制语义 | 保留标准 LSP，未实现命令延期 |
| P4.0-P4.3 剩余能力与持续演进 | P1.7 capability registry、P2/P3 相关边界稳定；回归按已完成能力切片推进 | 按能力切片扩展剩余 LSP、TCLE/API、回归、性能、兼容性和发布 | 每个方法/API 独立 owner、测试、性能和迁移记录 | 单项能力通过后才标记 `supported`；不阻塞其他独立能力 | 单项 `candidate`/`blocked`/`unsupported` 不升级为全局失败 |

## 门禁评审规则

- 不能用“源码中出现关键字”替代语义或目标验证。
- 不能用“默认回归通过”替代 device feature 全覆盖。
- 不能让客户端复制服务器的目标或语义判断。
- 不能在没有错误日志、诊断或降级状态的情况下宣称阶段完成。
- 最终发布前必须检查 `doc/tops-cpp-language-server-roadmap.md` 存在，并与 `specs/001-tops-lsp-roadmap/` 下的来源工件一致。
- 只有规格、研究、数据模型、契约和 quickstart 都能互相引用时，阶段才可进入 `/speckit.tasks`。
