# 契约：Roadmap 阶段门禁

## 门禁记录格式

每个阶段必须记录：

- `phase`：`P0` 至 `P4`。
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
| P0 Go 服务器语义基线 | 当前 headers、Clang 属性、driver、tests 和状态记录可定位 | Go tokenizer/parser、AST、符号索引、目标语义边界、compile database 优先级和 Clang 差分 oracle | `.tops`、`.cpp + -Tops`、`-x tops`，目标参数，Go parser fixtures，`clang -fsyntax-only` 差分 | 每个 P1 条目有证据、正反例、目标前提、Go 侧验证动作和差分动作；Go 所有权明确 | 未验证能力标记 `candidate`/`blocked`；不引入 clangd 运行依赖 |
| P1 核心语言服务 | P0 Go 语义模型和 LSP 边界稳定 | 标准 C++、执行/常用存储空间、基础内建变量/向量服务 | Go server unit、LSP contract、有效/无效/不完整 fixture、Clang 差分 | 核心 fixture 诊断、补全、悬停、导航和失败路径可重复 | 延后目标相关能力，保持受限模式 |
| P2 目标感知语义 | P1 上下文和 LSP 边界稳定 | 架构条件、启动/资源、向量对齐、DTE/同步边界 | 多 profile 正反例矩阵、目标切换和 stale 诊断 | 目标不匹配可解释；客户端与服务器不矛盾 | 目标能力标记 `target-dependent` 或 `unsupported` |
| P3 VS Code 工作流 | P1 LSP 生命周期契约稳定 | 激活、配置、状态、日志、重启、单根/多根工作区 | 客户端集成、配置变更、重连、旧诊断清理 | 用户能从配置到结果完成闭环；错误可恢复 | 保留标准 LSP 能力，延期自定义命令 |
| P4 TCLE 覆盖与持续演进 | P2 和 P3 的语义、配置和客户端边界都已完成 | header/API 导航补全、版本/目标矩阵、性能和回归治理 | TCLE corpus、契约回归、性能观测和兼容性评审 | 新增能力具有证据、测试、契约版本和迁移记录 | 维持 `documentation-only` 或 `blocked` 状态 |

## 门禁评审规则

- 不能用“源码中出现关键字”替代语义或目标验证。
- 不能用“默认回归通过”替代 device feature 全覆盖。
- 不能让客户端复制服务器的目标或语义判断。
- 不能在没有错误日志、诊断或降级状态的情况下宣称阶段完成。
- 最终发布前必须检查 `doc/tops-cpp-language-server-roadmap.md` 存在，并与 `specs/001-tops-lsp-roadmap/` 下的来源工件一致。
- 只有规格、研究、数据模型、契约和 quickstart 都能互相引用时，阶段才可进入 `/speckit.tasks`。
