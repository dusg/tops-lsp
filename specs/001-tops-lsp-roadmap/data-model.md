# 数据模型：Tops C++ Language Server Roadmap

本模型描述 roadmap 文档需要维护的信息，不表示当前仓库已经存在对应运行时对象或数据库。文档以 Markdown 为主，字段定义用于保证不同阶段的记录可以互相引用和审查。

## SyntaxCapability

表示一个标准 C++ 能力或 Tops C++ 扩展。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `id` | 稳定标识符 | 是 | 采用 `TOPS-<domain>-<number>`，一旦发布不复用 |
| `category` | 枚举 | 是 | `standard-cpp`、`execution-space`、`memory-space`、`launch-resource`、`vector-numeric`、`builtin`、`tcle-api`、`target-condition` |
| `spelling` | 文本 | 是 | 记录关键字、属性、类型、内建变量或 API 名称 |
| `user_semantics` | 文本 | 是 | 说明用户可观察的语言行为，不写猜测性实现细节 |
| `target_predicate` | 文本 | 是 | 写明架构、编译模式、宏或工具链前提；无条件能力写 `all-configured-targets` |
| `source_evidence` | EvidenceRecord 列表 | 是 | 至少一个当前仓库源码、header、attribute 定义或文档入口 |
| `test_evidence` | EvidenceRecord 列表 | 否 | 若尚无测试，必须写明缺口和计划验证动作 |
| `status` | 枚举 | 是 | 见“状态规则” |
| `planned_phase` | 枚举 | 是 | `P0` 至 `P4` |
| `user_features` | 枚举列表 | 是 | `diagnostics`、`completion`、`hover`、`definition`、`references`、`semantic-tokens` 或 `documentation-only` |
| `verification` | 文本 | 是 | 描述最小正向、负向、边界检查 |
| `limitations` | 文本 | 是 | 记录已知失败、部分覆盖、版本限制或 `none-recorded` |

### 状态规则

- `candidate`：在源码或文档中发现，但证据尚未完成核对。
- `verified`：源码/头文件和至少一个适用的测试或编译验证已核对。
- `target-dependent`：语义已核对，但适用目标或配置不同。
- `supported`：已满足对应阶段的证据、契约和回归门禁。
- `unsupported`：当前工具链明确拒绝或 roadmap 明确不覆盖。
- `blocked`：存在已记录的编译、链接、运行或工具链缺陷，不能承诺用户能力。
- `deprecated-source`：仅存在于不再维护的旧资料，不能作为当前支持依据。

状态不得从 `candidate` 直接跳到 `supported`。`target-dependent`、`blocked` 和 `unsupported` 必须在用户可见说明中保留，不能静默降级。

## EvidenceRecord

表示一条可以被维护者复核的事实依据。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `kind` | 枚举 | 是 | `header`、`attribute`、`target-info`、`driver`、`test`、`status`、`documentation` |
| `path` | workspace-relative 或绝对路径 | 是 | 指向实际存在的文件；外部路径标记来源环境 |
| `anchor` | 符号、测试名或行附近文本 | 是 | 使复核者能快速定位，不依赖易变的行号 |
| `fact` | 文本 | 是 | 只写直接观察到的事实 |
| `verified_on` | 日期 | 是 | 记录核对日期 |
| `verification_command` | 文本 | 否 | 记录使用过的读取、搜索或测试命令 |
| `scope` | 文本 | 是 | 说明 checkout、工具链版本或目标范围 |

## TargetProfile

表示语言分析所使用的目标上下文。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `id` | 稳定标识符 | 是 | 例如 `gcu400-default`，名称必须表达已确认的目标范围 |
| `triple_or_driver_mode` | 文本 | 是 | 记录 target triple、`-Tops` 或 `-x tops` 等输入模式 |
| `arch_macros` | 字符串映射 | 是 | 记录实际注入或配置的架构宏；未经验证时为空并标记待核对 |
| `language_standard` | 文本 | 是 | 例如 `tops`、`c++17`；必须来自编译上下文 |
| `include_roots` | 路径列表 | 是 | 按优先级记录 Tops/Clang headers |
| `compiler` | 路径与版本 | 是 | 记录本地工具路径和版本，禁止默认使用系统 LLVM |
| `known_limitations` | 文本列表 | 是 | 记录目标相关缺陷、部分覆盖和不支持项 |
| `state` | 枚举 | 是 | `unresolved`、`configured`、`validated`、`stale`、`invalid` |

## CompilerInvocation

表示从 `compile_commands.json` 或 workspace settings 得到的一次用户编译命令。它只描述参数解析结果，不表示 LSP 运行时已经启动编译器。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `driver_kind` | 枚举 | 是 | `topscc`、`clang`、`clang++`、`unknown` |
| `executable` | 路径 | 是 | 原始 argv[0]；必须保留用户配置值 |
| `resolved_executable` | 路径 | 否 | 经过 PATH/符号链接解析后的实际路径 |
| `driver_version` | 文本 | 否 | 已核对才填写；未知时标记 `unverified` |
| `wrapper_version` | 文本 | 否 | `topscc` wrapper 版本；直接 Clang 为空 |
| `underlying_driver` | 文本 | 否 | `topscc` 展开的底层 `clang`/`clang++`/`syclcc` |
| `raw_arguments` | 字符串列表 | 是 | 保留原始 argv 顺序和参数边界 |
| `normalized_arguments` | 字符串列表 | 是 | 展开 response file、归一化路径并记录 wrapper 展开结果 |
| `forwarded_arguments` | 字符串列表 | 是 | 传给底层 Clang 的参数；不丢弃未知参数 |
| `driver_defaults` | 字段映射 | 是 | 记录默认值及来源，例如 `-std=c++11`、`-Tops`、`gcu300` |
| `semantic_arguments` | 字符串列表 | 是 | 影响 language、target、pass、include、macro 的参数 |
| `non_semantic_arguments` | 字符串列表 | 是 | 输出、链接或设备打包参数；保留但不参与 parser context |
| `unknown_arguments` | 字符串列表 | 是 | 无法分类的参数；不得静默丢弃 |
| `target_candidates` | 字符串列表 | 是 | `-arch` 组合或多架构展开后的候选目标 |
| `status` | 枚举 | 是 | `resolved`、`partial`、`missing`、`invalid` |
| `diagnostic` | 文本 | 条件 | 参数缺失、冲突、response file 无法读取或版本未知时必填 |

`topscc --dryrun` 的输出只作为离线证据，不写入 Go server runtime context；server 只使用静态参数解析结果。

## CompilationContext

表示某个 SourceDocument 的实际编译解释上下文。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `document_uri` | URI | 是 | 对应用户文档 |
| `source` | 枚举 | 是 | `compile_commands` 或 `workspace_settings` |
| `driver_invocation` | `CompilerInvocation` 引用 | 是 | 保存用户 driver、wrapper 展开和参数来源 |
| `raw_arguments` | 字符串列表 | 是 | 保留原始参数，供差异检查和审计 |
| `normalized_arguments` | 字符串列表 | 是 | 记录去重、路径归一化后的分析参数 |
| `target_profile_id` | 引用 | 是 | 必须引用已定义的 TargetProfile |
| `target_profile_ids` | 引用列表 | 是 | 多架构命令的候选 profile；单目标时只有一个元素 |
| `target_selection` | 枚举 | 是 | `single`、`multi`、`unresolved` |
| `resolution_status` | 枚举 | 是 | `resolved`、`partial`、`missing`、`invalid` |
| `diagnostic` | 文本 | 否 | `partial`、`missing` 或 `invalid` 时必填 |
| `last_verified` | 日期 | 是 | 编译上下文最后一次成功核对日期 |

优先级规则：同一文件优先使用有效的 `compile_commands` 条目；条目中的 driver kind、wrapper 参数、明确目标和语言参数优先。没有条目时才使用 workspace settings；workspace settings 不得把直接 Clang 改写成 topscc，也不得覆盖数据库中的明确参数。`topscc` 默认值必须带 `driver_default` 来源。

## LanguageDiagnostic

表示服务器给客户端的用户可见问题或受限分析提示。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `code` | 稳定代码 | 是 | 与能力或上下文错误关联，不能只依赖文本匹配 |
| `severity` | 枚举 | 是 | `error`、`warning`、`information`、`hint` |
| `range` | 文档范围 | 是 | 必须落在当前文档或明确的关联文件中 |
| `message` | 文本 | 是 | 说明事实、目标条件和下一步动作 |
| `source` | 文本 | 是 | 例如 `tops-lsp` 或 `compilation-context` |
| `related_information` | 位置列表 | 否 | 指向 header、编译命令或能力证据 |
| `stale` | 布尔值 | 是 | 旧结果在新上下文切换时不得继续覆盖新结果 |

## LSPCapabilityContract

表示一个服务器与 VS Code 客户端共享的协议能力。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `operation` | 方法名 | 是 | 标准 LSP 方法或命名空间明确的扩展方法 |
| `owner` | 枚举 | 是 | `server`、`client`、`shared-contract` |
| `request` | 结构描述 | 是 | 参数、触发条件和取消行为 |
| `response` | 结构描述 | 是 | 返回值、空结果和版本行为 |
| `error_behavior` | 文本 | 是 | 服务器不可用、上下文缺失、超时和 stale 结果行为 |
| `observability` | 文本 | 是 | 日志字段、级别、敏感数据约束 |
| `compatibility` | 文本 | 是 | 向后兼容、版本化和迁移/拒绝行为 |
| `acceptance` | 测试引用 | 是 | 指向契约测试和用户可见验收场景 |

## RoadmapMilestone

表示一个阶段性 roadmap 交付门。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `id` | 枚举 | 是 | `P0`、`P1`、`P2`、`P3`、`P4` |
| `objective` | 文本 | 是 | 用户可见结果 |
| `dependencies` | 引用列表 | 是 | 能力、profile、契约或工具链依赖 |
| `deliverables` | 文件/能力列表 | 是 | 必须能在仓库中定位 |
| `entry_evidence` | EvidenceRecord 列表 | 是 | 进入阶段前必须具备 |
| `exit_criteria` | 可测条件列表 | 是 | 与 spec 的 success criteria 对齐 |
| `fallback_state` | 文本 | 是 | 无法完成时的 `blocked`、`unsupported` 或延期说明 |
| `status` | 枚举 | 是 | `planned`、`research-passed`、`design-passed`、`ready-for-tasks`、`implemented`、`verified` |

## 关系与生命周期

- 一个 `SourceDocument` 解析为一个当前的 `CompilationContext`，并引用一个 `TargetProfile`。
- 一个 `SyntaxCapability` 可以引用多个 `EvidenceRecord`，并被多个 `RoadmapMilestone` 交付。
- 一个 `LSPCapabilityContract` 引用一个或多个 `SyntaxCapability`，并关联服务器、客户端和契约测试。
- 一个 `LanguageDiagnostic` 必须引用触发它的 `CompilationContext` 或 `SyntaxCapability`，以便解释目标条件和错误来源。
- `CompilationContext` 在编译数据库或工作区设置变化后从 `resolved` 变为 `stale`，重新核对后才能回到 `resolved`。
- `RoadmapMilestone` 只能从 `research-passed` 进入 `design-passed`，再进入 `ready-for-tasks`；没有契约或测试证据不能进入 `verified`。

## RoadmapDocument

表示最终对外发布的路线图文档。它与 `specs/001-tops-lsp-roadmap/` 下的规划工件分离。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `publication_path` | 固定路径 | 是 | 必须为 `doc/tops-cpp-language-server-roadmap.md` |
| `source_artifacts` | 路径列表 | 是 | 至少引用 spec、research、data model、LSP boundary、compiler-driver 和 roadmap gates |
| `architecture_decision` | 文本 | 是 | 明确 Go server 从零实现、不扩展或依赖 clangd |
| `phase_summary` | RoadmapMilestone 列表 | 是 | 与 P0-P4 门禁保持一致 |
| `verification_status` | 枚举 | 是 | `draft`、`reviewed`、`published` |
| `last_verified` | 日期 | 是 | 最近一次与来源工件对齐的日期 |

发布规则：只有当所有来源工件通过文档错误检查、门禁字段完整且发布文档路径固定时，`verification_status` 才能从 `reviewed` 进入 `published`。发布文档不得复制未经确认的目标语义或省略 Go/clangd 架构边界。
