# 数据模型：Tops C++ 通用 Semantic Baseline

## CompilationCommand

表示用于生成 `SemanticContext` 的真实或合成编译命令。

| 字段 | 说明 | 约束 |
| --- | --- | --- |
| `source` | `compile_commands`、workspace 或 `default` | 粗粒度来源摘要；`default` 表示合成命令，具体字段来源以 `field_source` 为准 |
| `raw_arguments` | 原始参数或等价结构 | 不丢失参数顺序和 provenance |
| `normalized_arguments` | 归一化参数 | 不能改变明确参数含义 |
| `arch` | target arch | 默认 Tops 命令为 `gcu400` |
| `driver` | 编译入口 | 默认 `topscc` |
| `language` | 输入语言 | 默认 Tops 命令为 `TOPS` |
| `language_standard` | C++ standard | 默认 `c++17` |
| `pass_kind` | 执行 pass | `host`、`device` 或 `host+device` |
| `toolchain_include_roots` | 标准库和 Tops headers | 来自 topscc/toolchain 默认路径 |
| `workspace_include_roots` | 工程 headers | 来自 workspace/source provider |
| `toolchain_macros` | 工具链内建宏 | 单独记录 provenance |
| `workspace_macros` | 工程宏 | 来自 workspace 编译配置 |
| `field_source` | 每个字段的来源 map | 固定字段 key 的值为 `compile_commands`、`workspace`、`toolchain_default`、`fallback_default`、`unresolved` 或 `invalid` |
| `field_conflict` | 每个字段的冲突 map | 固定字段 key 的值为 `null` 或 `{sources: []ConflictSource, values: []FieldValue}`，保留所有来源，不选择一方 |
| `field_resolution` | 每个字段的解析状态 map | 固定字段 key 的值为 `resolved`、`unresolved` 或 `invalid` |
| `fallback` | 是否合成命令 | 默认命令为 true |
| `version` | 命令/上下文版本 | 变化使下游 snapshot stale |

`source` 表示命令记录的整体来源，不替代字段级来源：真实 `compile_commands` 记录使用 `source=compile_commands`，workspace 提供的等价命令使用 `source=workspace`，输入命令缺失时合成的命令使用 `source=default`；无论整体来源为何，补齐字段都必须在 `field_source` 中记录 `toolchain_default`、`workspace` 或 `fallback_default`。`field_source`、`field_resolution`、`field_conflict` 都是对象，固定 key 集合为 `source_uri`、`working_directory`、`raw_arguments`、`normalized_arguments`、`arch`、`driver`、`language`、`language_standard`、`pass_kind`、`toolchain_include_roots`、`workspace_include_roots`、`toolchain_macros`、`workspace_macros`；每个 key 必须出现，`field_conflict` 无冲突时使用 `null`。`workspace candidate` 是 source provider 提供的完整 command record，可与 `compile_commands candidate` 形成明确字段冲突；`workspace fallback field` 只是为已选择的 command 或合成默认 command 补充缺失的工程 roots/macros，不与已有明确值形成冲突。workspace command 的输入形状至少包含 `source_uri`、`working_directory`、`normalized_arguments` 和字段来源；没有 `compile_commands` candidate 时才可选择 `source=workspace`，与 compile command 同时存在且字段冲突时整体为 `invalid`。字段冲突只保留参与冲突的来源实例，并通过对应 key 的 `field_conflict={sources,values}`、`field_source=invalid`、`field_resolution=invalid` 表示，不把整体来源改写成另一枚举。

`FieldValue` 表示字段归一化后的 JSON value：允许 JSON `string`、`number`、`boolean`、`null`、array 或 object；object key 按 UTF-8 字节序规范化排序，array 保持原顺序，合法 JSON number 解析为有限精确十进制，去除小数尾零，`-0` 归一化为 `0`，指数形式转换为等价的无指数十进制表示；非有限数字和重复 object key 拒绝。字符串使用 UTF-8，控制字符、引号和反斜杠使用 JSON escape，其他 Unicode code point 保持 UTF-8。相等判断使用该规范化 JSON 序列化结果；`canonical_json(object)` 对任意 object 递归应用上述规则，输出为不带尾部换行的 UTF-8 JSON bytes。命令 candidate object 固定为 `{source_uri,working_directory,normalized_arguments}`，其中 `source_uri` 必须是 `NormalizeSourceURI` 的结果；`working_directory` 为以 workspace root 解析后的绝对、清理 `.`/`..` 的 `/` 分隔路径，不解析符号链接，缺少解析基准或解析失败的 candidate 不可用；response file 必须由 T008 的 expander 以 candidate working directory 为基准展开为有序参数数组，展开失败的 candidate 不可用。`ConflictSource` 的结构固定为 `{kind,instance}`，其中 `kind` 为 `compile_commands`、`workspace`、`toolchain_default` 或 `fallback_default`。按 `internal/index.NormalizeSourceURI` 规范化 source URI 分组后，等价 `compile_commands` 的判定字段为 `working_directory` 和 `normalized_arguments`；重复等价记录只保留字典序最小的命令指纹。`compile_commands` 的 instance canonical object 为 `{kind,source_uri,working_directory,normalized_arguments}`；workspace 的 instance canonical object 为 `{kind,source_uri,version,working_directory,normalized_arguments}`；toolchain 的 instance canonical object 为 `{kind,toolchain_identity,toolchain_version,normalized_roots,macro_version}`；前三类 instance 都是对应 canonical object 的 SHA-256 小写 hex，hash 输入是 canonical JSON 的 UTF-8 bytes，fallback 的 instance 固定为 `gcu400-default`。同一 `kind` 的不同 instance 不合并。`field_conflict` 的结构固定为 `{sources: []ConflictSource, values: []FieldValue}`；`kind` 的 `conflict_sort_order` 固定为 `compile_commands` < `workspace` < `toolchain_default` < `fallback_default`，同 kind 再按 `instance` UTF-8 字节序排序，`{kind,instance}` 不重复，`values` 与 `sources` 一一对应，`len(sources)=len(values)>=2`，只记录参与冲突的来源和值。`source_precedence` 独立固定为 `compile_commands` > `workspace` > `toolchain_default` > `fallback_default`，只用于无冲突字段的来源选择；任意字段的 `field_conflict` 非空都会使整体 `SemanticContext.status=invalid`；仅来源缺失且没有冲突时使用 `field_resolution=unresolved` 并保持整体 `partial`。

字段合并分两阶段：第一阶段只合并完整 command candidate，`compile_commands candidate` 与 `workspace candidate` 都可以提供全部 command 字段；对同一字段出现不同明确值时记录全部 `field_conflict` 并将整体状态设为 `invalid`，无冲突时按 `source_precedence` 选择来源。第二阶段才应用 fallback field：`workspace fallback field` 只补齐 workspace roots/macros 的缺失项，toolchain/default 只补齐各自允许的缺失字段，不覆盖已有明确值，也不与明确值形成冲突。字段来源必须逐项保存，不能只保存 `source` 一个整体值。

`CommandCandidateProvider` 是 T047 提供给 T010/T048 的输入边界：输入为当前文档 raw `document_uri`、workspace root 和可选 toolchain descriptor；输出为 `{document_uri,workspace_root,candidates,workspace_fallback_fields,toolchain_provenance}`。`workspace_fallback_fields` 固定为 `{workspace_include_roots,workspace_macros,version,field_source,field_resolution}`，只承载 workspace roots/macros 的补缺值和来源版本；不得承载 target、arch、driver、language、standard、pass 或 toolchain roots/macros。`toolchain_provenance` 固定为 `{toolchain_identity,toolchain_version,normalized_roots,macro_version,field_source,field_resolution}`，缺失 token 使用 `null` 和 `field_resolution=unresolved`。`candidates` 中每项固定为 `{source_kind,source_uri,version,working_directory,raw_arguments,normalized_arguments,field_source,field_resolution,field_conflict,parse_error}`：`source_kind` 为 `compile_commands` 或 `workspace`；`source_uri`、`working_directory` 为 raw string；`version` 对 workspace candidate 是 source provider 的 UTF-8 version token、对 compile_commands candidate 为 `null`；`raw_arguments` 为保持输入顺序的 string array；`normalized_arguments` 为 string array，展开失败时为 `null`；`field_source`/`field_resolution`/`field_conflict` 使用 CompilationCommand 的固定 field key map；`parse_error` 为 string 或 `null`。解析失败 candidate 可以保留在 provider 输出中，但不得进入 resolver 的可用 candidate 集合；T010 只消费该结果执行 URI 过滤、语言选择、去重、冲突和 fallback 决策，T047 不重复实现这些语义规则，T048 负责把 provider 结果接入 server resolver。

来源元数据规则：`workspace.version` 使用 source provider 提供的 UTF-8 版本 token，缺失时写 JSON `null` 并将该字段标记为 `field_resolution=unresolved`；`toolchain_identity`、`toolchain_version` 和 `macro_version` 使用 toolchain provenance descriptor 的 UTF-8 token，缺失时同样写 `null` 并标记 unresolved；`normalized_roots` 是使用 `NormalizeSourceURI` 规范化后按 URI UTF-8 字节序排序的数组，来源不可用时写 `null`。response file 以 candidate 的 `working_directory` 为相对路径基准，嵌套展开采用深度优先，最多 16 层；文件必须是 UTF-8，token 使用空白分隔、单/双引号保留空白、反斜杠转义下一个字符；循环、超过深度、读取失败、非法 UTF-8 或未闭合引号都使 candidate 不可用。默认/toolchain 来源只补齐缺失字段，不与已有明确字段形成冲突；只有两个或以上明确来源对同一字段给出不同 `FieldValue` 时才写入该字段的 `field_conflict`。

默认命令只对 `.tops`、`languageId=tops` 或明确 Tops 模式生成。普通 C++ 文件不得被改成 TOPS。

## SemanticProfile

表示语义规则集。

- `profile_id`：非空；真实 Tops target 使用具体 profile，Tops 无明确 target 时引用 `gcu400-default`，普通或未解析为 Tops 的 C++ 文档引用 `cpp-default`。
- `target_arch`：GCU400 或其他已解析 target；fallback profile 记录 GCU400。
- `rules`：host/device、address-space、vector 和基础类型规则。
- `fallback`：标记是否来自默认命令。
- `target_provenance`：真实 compile command 或 fallback 来源。

普通 C++ 的 `cpp-default` 不包含 Tops/GCU400 target rules；它只提供标准 C++ 基础规则。

## WorkspaceSourceCache

按 `source_kind`、规范化 URI 或 workspace macro descriptor、source version 和 SemanticContext version 缓存 workspace header/macro 来源：

- 相同 URI/version/context 的重复引用只读取一次；
- 不同 source version 必须重新读取；
- header 失效或 context version 变化时删除受影响缓存；
- workspace macro 配置版本变化时删除对应 macro entries；
- workspace enumeration 永远不作为读取入口。

## SemanticContext

表示一次 semantic 分析输入快照。

- `document_uri`
- `CompilationCommand`
- `SemanticProfile`
- `status`：`resolved`、`partial`、`missing`、`invalid`
- `document_version`
- `semantic_context_version`
- `source_versions` / `header_versions`
- `analysis_generation` / `epoch`
- `pass_views`：host、device 的独立事实
- `conditional_macros`
- `missing_fields` / `conflicts`
- `provenance`
- `source_version_fingerprint`：按 URI 排序后的 source/header version aggregate fingerprint，对外映射为 `sourceVersion`

状态规则：

1. 输入命令缺失或无有效命令的 Tops 文档先合成默认 `CompilationCommand`；只有在默认核心字段、toolchain roots/macros 和已引用 workspace 依赖完整时才生成 `resolved`，来源缺失时为 `partial`，但始终保留 fallback profile。
2. 可识别 Tops 但输入命令缺失不进入整体 `missing`；只有无法形成语言/命令上下文时才进入 `missing`，单个 pass view 的 `availability=missing` 不改变该整体状态规则。
3. target 冲突时仍有 `gcu400-default` profile，状态为 `invalid`，不选择冲突的一方。
4. `partial/missing/invalid` 只执行事实完整的基础规则；`resolved` 的默认 GCU400 profile 可以执行 GCU400 规则。
5. semantic snapshot 必须始终有非空 `profile_id`；resolver 之后不允许 no-profile 分支。

`pass_views` 使用固定结构：

- `pass_views.host`：`pass_kind=host`、规则可用性、pass provenance、状态和 host facts；
- `pass_views.device`：`pass_kind=device`、规则可用性、pass provenance、状态和 device facts。

每个 pass view 还必须记录 `availability`：`available`、`not-applicable` 或 `missing`。普通 C++ 的 device view 为 `not-applicable`；Tops `host+device` 中一侧来源缺失为 `missing`，不得伪装成 `not-applicable`。

`pass_kind=host+device` 只表示同时存在两个视图，不是第三种执行空间。

状态优先级：

| 条件 | 最终状态 | 规则能力 |
| --- | --- | --- |
| target/参数来源冲突 | `invalid` | 不选择冲突一方，不发布冲突上下文下的确定性目标 Error |
| 无法形成语言/命令上下文 | `missing` | 只执行目标无关基础规则 |
| 默认核心字段可生成，但 toolchain roots/macros 或已引用 workspace 依赖缺失 | `partial` | 只执行事实完整规则 |
| 真实命令完整，toolchain roots/macros 和已引用 workspace 依赖可用 | `resolved` | 使用具体 profile |
| 无命令，但默认核心字段、toolchain roots/macros 可用，未引用缺失 workspace 依赖 | `resolved` | 使用 `gcu400-default` |

未被当前文档引用的 workspace header/macro 缺失不自动降低状态；缺失来源在被引用时才影响当前 SemanticContext。

## SemanticSnapshot

表示可提交和发布的 semantic 结果。

- `context`
- `profile_id`：直接存储且非空；与 `context.profile_id` 一致
- `context_status`：直接存储当前 snapshot 状态
- `source_version_fingerprint`：直接存储用于诊断和提交校验的 aggregate fingerprint
- `document_version`、`semantic_context_version`、`source_versions`、`header_versions`、`generation`、`epoch`
- `declarations`
- `scopes`
- `types`
- `callables`
- `calls`
- `references`
- `diagnostics`
- `status`
- 版本集合和失效原因

`SemanticSnapshot.status` 只允许 `complete`、`degraded` 或 `stale`：`complete` 表示当前 SemanticContext 和必需来源可用；`degraded` 表示存在 incomplete/partial/missing/invalid 或 unsupported 事实；fixture 的 `expected_status=unsupported` 或 `invalid` 都映射为 snapshot `degraded`，分别记录 `unsupported_reason` 或 `invalid_reason`；`stale` 只用于失效过程，不能被查询、提交或发布。

只有所有 document/context/source/header/generation/epoch 版本匹配时才能提交。旧 snapshot 只产生 stale 观测，不发布。

## SemanticType

基础类型表示，支持：

- scalar kind 和 cv qualifiers；
- pointer depth、pointee type、address space；
- reference kind 和 referred type；
- 标准标量隐式转换和可直接判断的指针/引用转换类别；
- `conversion_status`：`supported`、`unsupported` 或 `unresolved`；user-defined conversion 为 `unsupported` 且不产生 type mismatch Error；
- unresolved/unsupported 标记。

用户自定义转换、模板相关转换和完整 overload resolution 不属于 v1。

## CallableFact

记录函数名称、限定名、scope、参数顺序和类型、返回类型、声明/定义、执行空间属性、`default_execution_space`、执行空间 provenance、pass view 和调用边界。普通 C++ 的 `default_execution_space=false`；resolved Tops 未标注函数才可以为 true。

## CallFact

记录一个调用点，而不是被调用函数声明：

- 调用表达式 range 和 callee spelling；
- `caller_kind` 和 `callee_kind`：`host`、`device`、`host-device` 或 `kernel-entry`；
- candidate `symbol_id` 列表及 `resolved`/`ambiguous`/`unresolved` 状态；
- caller/callee pass view；
- 每个参数的 conversion status，包括 `supported`、`unsupported`、`unresolved`；
- 条件 provenance 和诊断来源。

`callables` fixture 只验证 CallableFact 的声明、参数和返回值；`calls/basic` fixture 只验证 CallFact 的调用点、候选和转换结果；`calls/host-device` fixture 验证 pass 边界。

overload-dependent conversion 使用 `candidate_status=ambiguous` 或 `unresolved`、`expected_status=degraded`；多个候选时可发布 `tops-semantic-ambiguous-name` Warning，但不得发布确定性 Error；完整 overload-resolution fixture 使用 `capability=out-of-scope` 和 `expected_status=unsupported`。

## SemanticDiagnostic

- `code`：稳定 `tops-semantic-*` code；
- `severity`：Error、Warning、Information 或 Hint；
- `range`：最小可操作源码范围，LSP 使用 UTF-16 半开区间；
- `message`：Go server 自有文本；
- `pass_kind`：pass-sensitive diagnostic 使用 host 或 device，pass-neutral 的 context/incomplete diagnostic 使用 `not-applicable`（组合命令拆成独立诊断）；
- `semantic_kind`：固定 v1 semantic diagnostic kind；
- `profile_id`、`context_status`、document/context/source version；
- `source_version_fingerprint`：由 T010 实现的 `AggregateSourceVersion` 生成，对外 JSON `sourceVersion`；
- `conditional_provenance`；
- `related_information`；
- `recoverable` / `incomplete`。

目标特定 Error 必须有非空源码范围；没有范围时只能降级或不发布。

`source_version_fingerprint` 算法唯一使用 `internal/index/source.go` 导出的 `NormalizeSourceURI` 和 `internal/semantic.AggregateSourceVersion(documentURI string, entries []SourceVersionEntry) (string, error)`；`NormalizeSourceURI` 只负责 URI 规范化，aggregate helper 负责去重、冲突状态、排序和 SHA-256。root source 固定为当前 `document_uri` 的规范化 URI。同一规范化 URI 的相同版本只保留一条，不同版本写入 aggregate-only 的 `status:conflict`；再按 URI 字节序排序，逐行写入 `uri\0version:<value>\n` 或 `uri\0status:<value>\n`，使用 UTF-8 bytes 计算 SHA-256，输出 `sha256:<hex>`。semantic diagnostic 总有 root source；root source 缺失或 `document_uri` 无法规范化时不发布，comparison record 标记为 `not-comparable`；依赖缺失使用 `missing`、`unresolved`、`unreadable` 或 `stale` 等稳定 status token。

## SourceVersionEntry

`SourceVersionEntry` 是 `AggregateSourceVersion` 的输入项，固定为 `{uri,version,status}`：`uri` 是 source/header URI，`version` 是 UTF-8 version token，`status` 只能是 `missing`、`unresolved`、`unreadable` 或 `stale`；`version` 与 `status` 必须恰有一个非空。helper 接收当前文档的 raw `document_uri`，先用 `NormalizeSourceURI` 规范化 root 和每个 entry 的 URI；root 必须存在于 entries 且使用 `version`，否则返回 error。相同规范化 URI 的重复 entry 按相同 version 去重，不同 version 或 status 产生 aggregate-only 的 `status:conflict` 序列化 token；`conflict` 不是 `SourceVersionEntry.status` 的合法输入值。

## SemanticFixtureCase

fixture manifest 的每条记录至少包含：

- `case_id`：当前 fixture manifest 内唯一；comparison manifest 通过此字段引用；
- `scenario`：普通 capability 使用 `valid`、`invalid`、`incomplete` 或 `context-boundary`；`capability=out-of-scope` 只使用 `unsupported`；
- `capability` 与用户验收能力的映射：`declarations`=声明/作用域/名称，`types`=scalar/pointer/reference，`callables`=函数参数/返回值，`calls`=host/device 调用，`address-space`=地址空间，`vectors`=向量约束，`out-of-scope`=后续能力边界；
- `out_of_scope_kind`：`templates`、`overload-resolution`、`constant-evaluation`、`access-control`、`linkage` 或 `code-generation`，仅 `capability=out-of-scope` 时必填；
- `callable_kind`：callables/call capability 必须是 `host`、`device`、`host-device` 或 `kernel-entry`；其他 capability 写 `not-applicable`；`host-device` 对应 `__host__ __device__`，`default-host` 通过 `callable_kind=host` 和 `default_execution_space=true` 表示；
- `default_execution_space`：callables capability 必须是布尔值；只有明确登记的 resolved Tops `default-host` 子场景为 `true`，普通 C++ 和其他 callable 场景为 `false`；其他 capability 写 `not-applicable`；
- `caller_kind`/`callee_kind`：calls capability 必填，其他 capability 写 `not-applicable`；`callee_kind=kernel-entry` 时必须使用明确的 kernel entry 调用场景；
- `source_files`、`profile_id`、宏/include provenance；`pass_kind` 对需要执行视图的 capability 必填，否则写 `not-applicable`；
- `condition`：条件 fixture 必须包含 `condition_id`、`macros`（name 到 macro value 的对象）、`parent_conditions`（当前 fixture 内已声明 condition ID 的整数数组，必须无循环）、`expression_range`（`source_uri` 及 source byte offset 的半开 `start`/`end`）和 `expected_state`（`active`、`inactive` 或 `unknown`）；非条件 fixture 可省略；diagnostic range 必须带属于 `source_files` 的 `source_uri`，另按 LSP UTF-16 半开规则验证；
- `expected_status`：`complete`、`degraded`、`unsupported` 或 `invalid`；普通 capability 的 `valid`/`invalid`/`incomplete`/`context-boundary` 分别使用 `complete`/`invalid`/`degraded`/`degraded`，out-of-scope 的 `unsupported` 使用 `unsupported` 或 `degraded`；
- `expected_diagnostics`：code、severity、passKind、range、`status`（`published`/`suppressed`）和 reason。

字段形状固定如下：`source_files` 是字符串数组；`callable_kind`、`caller_kind`、`callee_kind` 和 `pass_kind` 是字符串，`pass_kind` 允许 `host`、`device`、`host+device` 或 `not-applicable`；`default_execution_space` 在 callables capability 中是布尔值，其他 capability 为字符串 `not-applicable`；`condition` 是对象，`condition_id` 是 fixture 内唯一整数，`condition.macros` 是对象、`condition.parent_conditions` 是当前 fixture 内的 condition ID 数组、`condition.expression_range` 是 `{source_uri,start,end}` 的 source byte offset 半开范围、`condition.expected_state` 是 `active`/`inactive`/`unknown`；`expected_diagnostics` 是对象数组，range 使用带 `source_uri` 的 source byte offset 半开范围，发布到 LSP 时再转换为 UTF-16。

validator 必须拒绝缺少条件必填字段、重复或不存在的 `case_id`/`condition_id`、parent condition 循环或悬空引用、未知 `capability/scenario/out_of_scope_kind/callable_kind/caller_kind/callee_kind/pass_kind/expected_status`、非法 capability/scenario/expected_status/out_of_scope_kind/default_execution_space 组合、非法 diagnostic code/severity/passKind/range/source_uri；`declarations/types/address-space/vectors/out-of-scope` 的 callable/caller/callee/pass/default_execution_space 字段按适用性规则使用 `not-applicable`，`callables` 的 `default_execution_space` 必须为布尔值并遵守 default-host 规则，`calls` 的 caller/callee 字段必须遵守 CallFact 规则。

`cmd/tops-semantic-check` 读取 `testdata/semantic/manifest.json`，输出每类总数、通过数、失败数和失败 case ID；任一失败返回非零退出码。

## SemanticComparisonCase

`testdata/semantic/comparison_manifest.json` 单独记录：

- 输入：`fixture_case_id`（必须引用 fixture manifest 中存在的 `case_id`）、`document_uri`、`root_source_uri`、source files/hash、working directory、raw/normalized arguments、profile/pass、include/macro provenance、header versions、compiler/tool version、`source_version_fingerprint`；
- 输出预期：`compare_status`（`comparable`/`not-comparable`）、`missing_fields`、`known_difference`、Go normalized diagnostics 和 Clang normalized categories；
- `neutral_diagnostics`：不依赖具体 pass 的 Go semantic diagnostics，放在 comparison 顶层；每条必须使用 `passKind=not-applicable`，不得伪装成 `pass_results` 中的实际 host/device pass；
- golden：固定写入 `testdata/semantic/expected/semantic_comparison.json`。

字段形状固定如下：`document_uri` 是当前文档的 raw URI；`root_source_uri` 必须等于 `NormalizeSourceURI(document_uri)` 并出现在 `source_files` 中，任一缺失、无法规范化或不一致时 `compare_status=not-comparable`。`source_files` 是按规范化 URI 排序的 `{uri,content_hash}` 数组；每个 source file 生成一个 `SourceVersionEntry{uri,version=content_hash}`；`header_versions` 是 `{uri,version,status}` 数组，直接生成对应 `SourceVersionEntry`，`version` 与 `status` 遵守 SourceVersionEntry 互斥规则；root entry 的 version 使用 root source file 的 `content_hash`。`source_hash` 是对每行 `uri\0content_hash\n` 的 UTF-8 bytes 计算 SHA-256；`pass_kind` 可为 `host`、`device` 或 `host+device`；`pass_results` 是每个实际 pass 的 `{pass_kind,go_status,go_diagnostics,clang_status,clang_categories}` 数组，实际 pass 的 `pass_kind` 只能为 `host` 或 `device`；`go_status` 只能为 `complete`、`degraded`、`unsupported` 或 `invalid`，`clang_status` 只能为 `available`、`skipped`、`failed` 或 `not-comparable`；`go_diagnostics` 和 `clang_categories` 是数组，`missing_fields` 与 `known_difference` 是字符串数组；`neutral_diagnostics` 是顶层对象数组，至少包含 code、severity、range 和 `passKind=not-applicable`；`pass_results.go_diagnostics` 至少包含 code、severity、range 和 passKind，且只记录对应实际 pass 的 pass-sensitive diagnostic。

## 状态关系

```text
真实 compile_commands ----+
                         |
输入命令缺失的 Tops -----+--> 默认 GCU400 命令 -> SemanticProfile(gcu400-default)
普通/未知 language ------+--> cpp-default
                         |
                         v
                   SemanticContext
                 /      |        |        \
             resolved  partial  missing  invalid
                 |        |        |        |
          来源完整   来源缺失  无法形成   冲突目标
                              上下文     不报确定错误
                 |
         SemanticSnapshot -> publishDiagnostics
```
