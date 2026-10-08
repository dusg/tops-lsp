# 数据模型：P0 Go 服务器语义基线

本模型用于描述 P0 文档和后续 P1 测试需要记录的对象，不表示当前 workspace 已经存在 Go runtime、数据库或持久化服务。所有状态必须能回溯到 source/test evidence 和 context version。

## 1. EvidenceRecord

表示一条可以由维护者复核的事实证据。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `kind` | `header` / `attribute` / `target-info` / `driver` / `test` / `status` / `documentation` | 是 | `documentation` 不能作为唯一的当前语义证据 |
| `path` | workspace-relative path | 是 | `llvm-project` 路径必须指向当前 checkout；外部路径需标明环境 |
| `anchor` | symbol/test name/命令片段 | 是 | 使维护者能快速定位，不依赖易变行号 |
| `fact` | 直接观察到的事实 | 是 | 不写未经验证的根因或跨架构推断 |
| `scope` | checkout、target、pass、版本或测试范围 | 是 | 明确 host/device、GCU/EFGCU 和标准版本 |
| `status` | `observed` / `candidate` / `blocked` | 是 | 证据本身也可标为待核对 |
| `verified_on` | date | 是 | 记录最近一次实际阅读/命令核对日期 |
| `verification_command` | command text | 否 | 已执行才填写输出；计划命令标为 planned |
| `limitations` | text | 是 | 记录只证明 syntax、compile、CodeGen 或 runtime 的边界 |

## 2. SyntaxCapability

表示八个能力域中一个稳定的语法或语义条目。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `id` | `TOPS-<DOMAIN>-<number>` | 是 | 发布后不复用 |
| `domain` | `standard-cpp` / `execution-space` / `memory-space` / `launch-resource` / `vector-numeric` / `builtin` / `tcle-api` / `target-condition` | 是 | 必须属于八个域之一 |
| `spelling` | string/list | 是 | 用户源码中的关键字、attribute、type、builtin 或 API |
| `user_semantics` | text | 是 | 面向用户的语义；不把 CodeGen 或硬件实现写成用户保证 |
| `target_precondition` | text | 是 | profile、macro、pass、language、header 或标准前提 |
| `source_evidence` | EvidenceRecord[] | 是 | 至少一条当前源码/TargetInfo/header/driver 证据 |
| `test_evidence` | EvidenceRecord[] | 是 | 没有现有测试时必须记录 `planned` fixture 和缺口 |
| `status` | `candidate` / `verified` / `target-dependent` / `supported` / `unsupported` / `blocked` / `deprecated-source` | 是 | P0 未实现 Go server，不能把 Go 能力标为 `supported` |
| `limitations` | text | 是 | 记录目标、版本、测试 coverage 或运行时边界 |
| `verification` | text | 是 | 至少包含 valid、invalid、incomplete 或 target-boundary 中适用的动作 |
| `planned_phase` | `P0` / `P1` / `P2` / `P3` / `P4` | 是 | 语义 baseline 与后续用户能力分开 |
| `lsp_surface` | enum list | 是 | `diagnostics`、`completion`、`hover`、`definition`、`references`、`document-symbol`、`documentation-only` |

**状态门禁**：没有源码证据不能超过 `candidate`；只有源码证据没有测试/编译核对不能标 `supported`；目标不同必须拆 profile 或明确分支；`blocked`/`unsupported` 必须保留恢复条件或拒绝理由。

## 3. TargetProfile

表示一个语言分析候选目标，不表示硬件认证。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `id` | stable string | 是 | `gcu300-default`、`gcu400-default`、`gcu410-default`、`gcu450-default`、`gcu500-default`、`efgcu500-default` |
| `target_triple` | string | 是 | 保留 direct target 与 offload target 的区别 |
| `cpu_or_offload_arch` | string | 是 | `gcu300`...`gcu500` 或 `efgcu500` |
| `input_modes` | list | 是 | `.tops`、`-Tops`、`-x tops` 及适用 device flags |
| `arch_macros` | map | 是 | GCU 使用 `__GCU_ARCH__`；EFGCU 使用 `__EFGCU_ARCH__`；未命中时为空且为 invalid/missing |
| `language` | string | 是 | `tops`，并记录 C++ standard 来源 |
| `pass_kind` | enum | 是 | `host`、`device`、`offload-host`、`offload-device` |
| `compiler` | object | 是 | executable、version、resource dir、driver mode |
| `include_roots` | ordered list | 是 | active Tops/Clang/project/system roots，顺序可审计 |
| `known_limitations` | list | 是 | 不同 profile 的 header/API/语义缺口 |
| `evidence` | EvidenceRecord[] | 是 | 记录 target map、macro output、headers 和 tests |
| `state` | `candidate` / `configured` / `validated` / `stale` / `invalid` | 是 | P0 计划产物初始为 `candidate` |
| `verification` | text | 是 | `-###`、`-dM -E`、include trace、syntax 和适用 tests |

## 4. CompilationContext

表示一个 SourceDocument 的文件级编译解释上下文。

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `document_uri` | URI | 是 | 归属文档和 context key |
| `workspace_root` | URI/path | 是 | 多根工作区必须参与 key |
| `compile_commands_path` | path | 否 | 找不到则转 workspace fallback，不伪造条目 |
| `compile_command_entry` | object | 否 | 同文件多条冲突为 `invalid` |
| `source` | `compile_commands` / `workspace_settings` | 是 | 有效数据库条目优先 |
| `raw_arguments` | string[] | 是 | 保留原始次序和 flags |
| `normalized_arguments` | string[] | 是 | 路径/response file/重复规则归一化，不改变明确语义 |
| `input_kind` | enum | 是 | `.tops`、`.cpp + -Tops`、`-x tops` 等 |
| `language_standard` | object | 是 | `tops` language、`-std`、driver default 的实际来源 |
| `target_profile_id` | reference | 是 | 六个 profile 之一；未知值为 invalid |
| `target_triple` | string | 是 | driver/cc1 的实际 triple |
| `cpu_or_offload_arch` | string | 是 | 保留原始 `-mcpu`/offload arch |
| `pass_kind` | enum | 是 | host/device/offload pass 不可合并 |
| `compiler` | object | 是 | executable、version、resource dir、mode |
| `include_roots` | ordered list | 是 | active include 搜索顺序 |
| `predefined_macros` | map | 是 | 至少记录 GCU/EFGCU/TOPS device/SIMT 宏的实际输出 |
| `device_flags` | string[] | 是 | device-only、Tops SP/SIMT 等 flags |
| `working_directory` | path | 是 | 相对 include/response file 的基准 |
| `oracle_config` | object | 否 | 只用于验证，不属于 Go runtime dependency |
| `resolution_status` | `resolved` / `partial` / `missing` / `invalid` | 是 | 缺失/冲突必须有 diagnostic |
| `diagnostic` | LanguageDiagnostic reference | 条件 | `partial`/`missing`/`invalid` 时必填 |
| `context_version` | monotonic integer | 是 | 文本、配置、profile、数据库或 header 变化递增 |
| `last_verified` | date/time | 是 | 最近一次完成 driver/header/macro 核对 |

### CompilationContext 优先级

```text
有效文件级 compile_commands entry
    -> driver 派生的 language/target/pass/macros/include
    -> 无 entry 时的显式 workspace_settings fallback
    -> 没有隐藏 target/include/standard 默认
```

数据库中的明确 target、standard、输入模式、include、宏和 device flags 不被工作区设置覆盖。来源内部冲突为 `invalid`。发生变化时，context 和关联的 AST/index/diagnostic/LSP result 进入 stale，重新解析后才可恢复为 resolved。

## 5. SourceDocument、AST 和 SymbolIndex

### SourceDocument

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `uri` | 是 | LSP 文档标识 |
| `text_version` | 是 | 增量文本版本 |
| `text` | 是 | 当前编辑文本；日志不得完整记录 |
| `compilation_context` | 是 | 文件级 context 引用 |
| `token_version` / `ast_version` / `index_version` | 是 | 判断增量结果是否过期 |
| `parse_state` | 是 | `complete`、`recoverable`、`degraded`、`invalid` |

### ASTNode

AST 必须保存：kind、source range、原始 spelling、声明/类型/调用关系、attribute 参数、宏/条件来源、模板信息、错误恢复标记和 context/header version。AST 不产生诊断文本、不选择 profile、不管理 LSP lifecycle。

### SymbolIndexEntry

| 字段 | 说明 |
| --- | --- |
| `symbol_id` | 稳定符号标识，受 workspace/context/header version 约束 |
| `name` / `qualified_name` | 源码名称和限定名 |
| `kind` | function/type/variable/field/macro/builtin/API |
| `declaration` / `definition` / `references` | 可导航位置集合 |
| `scope` | namespace/class/function/header scope |
| `target_visibility` | profile、macro/pass 条件 |
| `type_summary` | 可用于 hover/completion 的类型摘要 |
| `state` | active、conditional、unresolved、stale |

Index 不将“存在声明”解释成“当前 target 可调用”；semantic 层必须再次判断可见性和合法性。

## 6. LanguageDiagnostic

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `code` | stable string | 是 | 与 context/capability 关联，不能只靠 message 匹配 |
| `severity` | error/warning/information/hint | 是 | 由 semantic/LSP contract 统一定义 |
| `range` | document range | 是 | 即使输入不完整也必须稳定可定位 |
| `message` | text | 是 | 事实、目标条件、限制和下一步动作 |
| `source` | `tops-lsp`/`compilation-context`/`oracle-diff` | 是 | 标明责任方 |
| `related_information` | locations | 否 | 可指向 header、compile command 或 evidence |
| `context_version` | integer | 是 | 旧版本不能覆盖新结果 |
| `stale` | boolean | 是 | stale 结果不发布或显式标记 |

P0 最小 context/error code 集合：`missing-compilation-context`、`invalid-compilation-context`、`unsupported-target-feature`、`analysis-degraded`、`server-unavailable`、`request-cancelled`、`stale-result`。

## 7. Go 层次边界

| 层 | 输入 | 输出 | 拥有的判断 | 不拥有的判断 |
| --- | --- | --- | --- | --- |
| tokenizer | text、document version | tokens、ranges、recovery markers | token 分类和范围 | target legality、diagnostic severity、LSP payload |
| parser | tokens、grammar mode | recoverable AST | 语法结构和错误恢复 | target feature gate、CodeGen、LSP lifecycle |
| AST | parser nodes | normalized declarations/types/calls | 结构和 provenance 保存 | 用户诊断 wording、profile 选择 |
| symbol index | AST、headers、context/header versions | declarations/definitions/references/scope | 索引和失效 | semantic legality、severity、client display |
| semantic | AST、index、CompilationContext、TargetProfile | diagnostics/completion/hover/navigation facts | Tops/C++ 目标语义 | tokenization、transport、client re-judgment、clangd runtime |
| LSP | server state、semantic results、client messages | standard LSP responses/notifications | 编解码、cancel/stale/error lifecycle | 重新实现 semantic rules |

## 8. LSPCapabilityContract

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `operation` | 是 | `initialize`、document sync、diagnostics、completion、hover、definition、references、documentSymbol、configuration |
| `owner` | 是 | server/client/shared-contract |
| `request` | 是 | 参数、context version、触发和取消 |
| `response` | 是 | 结果、空结果、version |
| `error_behavior` | 是 | missing/invalid/degraded/cancelled/stale/server unavailable |
| `observability` | 是 | 脱敏 URI、profile、context version、duration、error code |
| `compatibility` | 是 | 标准 LSP、可选能力、版本化扩展、迁移/拒绝 |
| `acceptance` | 是 | P1/P3 fixture 或 contract test 引用 |

## 9. P1Fixture

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `id` | 是 | `P1-<DOMAIN>-<V/I/P/T>` |
| `domain` | 是 | 八域之一 |
| `scenario_type` | 是 | valid、invalid、incomplete、target-boundary |
| `source_path` | 是 | existing path 或明确的 new fixture path |
| `context` | 是 | profile、standard、pass、include、macros |
| `expected_parser_state` | 是 | complete/recoverable/degraded/invalid |
| `expected_semantic_result` | 是 | diagnostics/symbols/completion/hover/navigation |
| `oracle_evidence` | 是 | Clang/test/status evidence 或 planned gap |
| `limitations` | 是 | 不覆盖的 runtime/CodeGen/hardware 行为 |
| `verification` | 是 | Go test、LSP contract、Clang differential 或文档检查 |

## 10. 生命周期和关系

- `SourceDocument` 解析为一个带 `CompilationContext` 引用的 AST；context 变化使 AST/index/diagnostics 进入 stale。
- 一个 `SyntaxCapability` 可引用多个 `EvidenceRecord`，并由多个 `P1Fixture` 验证。
- `TargetProfile` 决定条件宏、可见 header 和 semantic feature gate，但不决定 LSP transport。
- `SymbolIndex` 为当前文档、workspace header 和 conditional declarations 提供导航数据；semantic 决定是否可用。
- `LanguageDiagnostic` 必须关联 context version 和 capability/error code。
- `LSPCapabilityContract` 把 semantic 结果映射为用户可见结果；TypeScript client 只展示或转发。
- `RoadmapGate` 只有在 entry evidence、设计契约、测试和 fallback 完整时才可进入下一状态。
