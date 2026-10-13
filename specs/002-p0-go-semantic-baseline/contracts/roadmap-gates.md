# 契约：P0 与 P1 子阶段门禁

## 门禁记录格式

每个阶段必须记录：

- `phase`：P0、P1.1-P1.7、P2.0-P2.5、P3.1-P3.4 或 P4.0-P4.3。
- `objective`：用户/维护者可观察的结果。
- `entry_evidence`：进入阶段前可复核的源码、测试、工具或契约。
- `scope`：本阶段承诺和明确不承诺的能力。
- `dependencies`：profile、CompilerInvocation、CompilationContext、Go layer、LSP contract 或 Clang 离线对照依赖。
- `validation`：valid、invalid、incomplete、target-boundary 和失败路径。
- `exit_criteria`：可复核的完成条件。
- `fallback`：`candidate`、`target-dependent`、`blocked`、`unsupported` 或延期处理。
- `compatibility`：对 standard C++、LSP、setting、command 和后续任务的影响。
- `owner`：文档、Go server、client 或 shared contract。
- `publication_path`：P0 规格和设计工件所在的 feature directory；本 feature 不发布最终 roadmap。

## P0 Go 服务器语义基线

| 项目 | 定义 |
| --- | --- |
| `phase` | P0 |
| `objective` | 维护者拥有可审查的八域能力矩阵、六个候选 profile、CompilationContext 规则、Go 层次边界和 P1 场景入口 |
| `entry_evidence` | P0 spec、已安装 topscc wrapper、当前 driver/TargetInfo/headers、Clang attributes、Tops language status、现有 Clang/Tops tests |
| `scope` | 文档研究、topscc 参数证据、CompilerInvocation、候选 profile、context/data model、capability/LSP/gate contracts、quickstart |
| `dependencies` | 当前 LLVM/Tops checkout、开发机本地工具路径、项目宪法和 Spec Kit feature pointer |
| `validation` | 文档静态检查；`topscc --version`、`topscc --dryrun`、直接 `-###`、`-dM -E`、include trace、`-fsyntax-only`/适用 `llvm-lit` 的验证方案；八域×四场景矩阵设计审查 |
| `exit_criteria` | 8 domains、6 profiles、CompilerInvocation、CompilationContext precedence、6 Go layers、Clang/client boundaries、32 P1 scenarios 全部有状态/限制/验证方法；无模板占位符；质量清单通过 |
| `fallback` | 未确认的 profile/语义保持 `candidate`/`target-dependent`；缺证据标 `blocked` 或 `unsupported`；不得进入 `supported` |
| `compatibility` | 不修改现有 LSP、setting、command、Clang、Tops headers 或测试；只新增文档工件 |
| `owner` | P0 文档维护者；用户命令事实由 topscc 提供；语言事实由当前 LLVM/Tops checkout 提供；Clang 仅作离线对照 |
| `publication_path` | `specs/002-p0-go-semantic-baseline/` |

## P1 子阶段入口

| 项目 | 定义 |
| --- | --- |
| `P1.1` | 基础 transport、生命周期、文档同步、取消和错误；见 `specs/003-basic-lsp-transport/` |
| `P1.2` | 基础 tokenizer/parser、宏条件结构和恢复；见 `specs/004-tops-cpp-tokenizer-parser/` |
| `P1.3` | AST/index/navigation baseline；见 `specs/005-ast-symbol-index/` |
| `P1.4` | 通用 semantic baseline；见 `specs/006-p1-semantic-baseline/` |
| `P1.5` | 消费 P1.3/P1.4 的 diagnostics、completion、hover 和当前导航闭环 |
| `P1.6` | 已列 C++11/14/17 grammar 补齐和 P1.4 已实现规则接入 |
| `P1.7` | 已实现方法的 LSP 协议、错误、取消、stale 和 capability 收敛 |
| `P2.0-P2.5` | 完整 context、target/profile、宏展开和目标诊断 |
| `P3.1-P3.4` | VS Code client 工作流，基础 client 可与 P1.2-P2.0 并行 |
| `P4.0-P4.3` | 剩余 LSP/API 能力逐项实现、回归、性能和发布 |

## 阶段评审规则

- 不能用源码中出现关键字替代语义、目标或测试验证。
- 不能用默认 regression PASS 替代 full device feature coverage。
- 不能把 GCU 和 EFGCU 的宏、builtin、header/API 直接合并。
- 不能让 client 复制 target/semantic 判断。
- 不能在缺少 error/diagnostic/log 或 fallback 状态时宣称门禁通过。
- 不能把 Clang 对照验证 变成 Go runtime dependency。
- 只有 `spec.md`、`plan.md`、`research.md`、`data-model.md`、`quickstart.md`、contracts 和 checklist 互相引用且静态检查通过，P0 才能进入 `/speckit.tasks`。

## 失败状态规则

| 状态 | 使用条件 | 后续动作 |
| --- | --- | --- |
| `candidate` | 发现 spelling/source，但缺少 target/test 核对 | 保留候选，补 EvidenceRecord 和验证任务 |
| `target-dependent` | 语义或可见性随 profile/pass/header 条件变化 | 拆 profile/分支，禁止无条件 completion/hover |
| `blocked` | 工具链、编译、测试或依赖缺口阻止核对 | 记录阻塞原因、恢复条件和 owner |
| `unsupported` | 当前 scope 明确不承诺或工具链拒绝 | 用户可见拒绝理由，不静默标准 C++ fallback |
| `analysis-degraded` | 输入/context 不完整但可保留部分结果 | 标记范围、版本和缺失字段，等待重新分析 |
| `stale-result` | context/document/header version 已变化 | 丢弃旧结果或显式标记，不覆盖最新结果 |
