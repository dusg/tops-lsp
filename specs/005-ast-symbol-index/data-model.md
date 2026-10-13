# 数据模型：AST 和符号索引

**功能**：[spec.md](spec.md)
**研究决策**：[research.md](research.md)

## 模型边界

`parser` 产出可恢复的 AST、tokens、条件区域和 parser diagnostics；`index` 把这些结果转换为可查询的符号快照；`server` 保存并提交最新快照；`protocol` 把查询结果编码为标准 LSP 类型。

索引是进程内数据，不写数据库，不把完整源码写入日志。所有位置都以原始文档 URI 和原始 source byte range 为来源，发布 LSP 前统一转换为 UTF-16 半开区间。

## 版本对象

### SourceVersion

| 字段 | 类型 | 规则 |
|---|---|---|
| `uri` | string | 规范化后的文档或 header URI |
| `document_version` | int | 来自 `DocumentState.Version`；必须递增 |
| `context_version` | int | 来自 server context version |
| `source_version` | string | header 可用时的内容版本、文档版本或 provider 摘要 |
| `generation` | uint64 | 来自 server per-URI analysis generation |
| `epoch` | uint64 | 来自 server context/invalidation epoch |

快照提交必须同时匹配 document version、context version、generation 和 epoch。任一项不匹配，结果进入 `stale`/discarded，不得替换当前快照。

### ParseSnapshot

由 parser 结果和输入版本组成：

- `DocumentURI`、`DocumentVersion`、`ContextVersion`；
- `Root`、`Tokens`、`ConditionalRegions`；
- `ParseStatus`：`complete`、`recovered`、`partial` 或 `invalid`；
- parser diagnostics；
- header/include context provenance。

索引只消费 parse snapshot，不调用 tokenizer、读取 compiler argv 或启动 Clang。

## AST 元数据

现有 `SyntaxNode` 保留已有字段，并新增最小导航元数据：

| 字段 | 类型 | 规则 |
|---|---|---|
| `NameRange` | `parser.SourceRange` | 只覆盖声明/使用可选中的名称；未知时为空 |
| `DeclarationRole` | enum | `none`、`declaration`、`definition`、`unknown` |
| `BodyRange` | `parser.SourceRange` | 函数/类定义 body 可确定时记录；不完整时可为空 |

`NameRange` 和 `DeclarationRole` 是语法结构信息，不表示类型合法性、重载决议或 target 可用性。作用域和 qualified name 在 index 遍历中计算，避免把 workspace 关系写入 parser。

## ASTNodeRecord

index 内部可以为每个 AST 节点建立轻量记录：

| 字段 | 说明 |
|---|---|
| `node_key` | 当前快照内的遍历键，不承诺跨版本稳定 |
| `kind` | 原始 `parser.NodeKind` |
| `range` | 节点完整范围 |
| `name_range` | 可选名称范围 |
| `parent_key` | 当前快照内父节点 |
| `scope_id` | 所属 lexical scope |
| `conditional_state` | `active`、`inactive`、`unknown` |
| `incomplete` | 是否由恢复节点或不完整输入产生 |
| `origin_uri` | 源文档 URI |

## SymbolRecord

表示可被导航或后续 semantic 消费的声明实体。

| 字段 | 类型 | 说明 |
|---|---|---|
| `symbol_id` | string | 由 origin、kind、qualified name、role 和 normalized declaration anchor 组成的可复用标识 |
| `name` | string | 源码短名称 |
| `qualified_name` | string | 按 namespace/class/函数作用域拼出的限定名 |
| `kind` | enum | `function`、`type`、`attribute`、`builtin`、`namespace`、`field`、`macro`、`header` |
| `type_kind` | enum | `class`、`struct`、`enum`、`union`、`alias` 或空 |
| `scope_id` | string | 当前快照内的 lexical scope |
| `scope_name` | string | 可读的限定作用域 |
| `name_range` | SourceRange | 用于 definition、selectionRange 和定位 |
| `declarations` | []SymbolLocation | 所有声明位置 |
| `definitions` | []SymbolLocation | 所有可确认定义位置 |
| `references` | []ReferenceID | 指向引用表 |
| `condition` | ConditionalProvenance | 条件区域和分支状态 |
| `origin` | SymbolOrigin | 当前文档、打开文件、header、builtin source 或 unresolved |
| `detail` | string | 参数/返回类型/attribute spelling 等非语义摘要 |
| `signature` | string | 规范化声明头摘要，用于函数候选区分和稳定 ID |
| `parameter_summary` | string | 函数参数原文摘要；无法确认时为空 |
| `return_type_summary` | string | 函数返回类型原文摘要；无法确认时为空 |
| `execution_qualifiers` | []string | `__global__`、`__device__` 等已识别 Tops qualifier |
| `attribute_usages` | []SymbolLocation | 当前源码中的属性使用位置；不能作为可确认 definition |
| `attribute_sources` | []SymbolLocation | 属性使用对应的可确认 header/builtin 来源；来源未知时为空 |
| `builtin` | BuiltinProvenance | builtin 的 target/pass/macro context、source URI 和 availability |
| `status` | enum | `active`、`conditional`、`declaration-only`、`incomplete`、`unresolved`、`ambiguous`、`stale` |
| `source_version` | SourceVersion | 文档、context、header 版本关系 |
| `children` | []SymbolID | document symbol 树的子节点 |

### SymbolLocation

| 字段 | 规则 |
|---|---|
| `uri` | 可验证的源文档或 header URI |
| `range` | 声明/定义/引用完整范围，LSP 转换前为 source range |
| `selection_range` | 只覆盖名称；来源缺失时为空并回退到 `range` |
| `role` | `declaration`、`definition`、`reference`、`builtin-source` |
| `source_version` | 与目标快照一致 |

### BuiltinProvenance

| 字段 | 说明 |
|---|---|
| `target_profile` / `pass_kind` | 来自 `ParseContext`，未知时保留 `unknown` |
| `macro_context` | 当前解析使用的宏值快照 |
| `source_uri` | 可验证 builtin header 来源；unresolved 使用空值 |
| `availability` | `available` 或 `unresolved` |

不允许创建不存在的 URI 或伪造 header location。只有 `origin` 可验证时，location 才能进入 LSP 结果。

### symbol_id 规则

1. 规范化 `origin URI`、`kind`、`qualified_name`、`DeclarationRole`。
2. 对声明头 token 去除 comment/newline/trivia，忽略函数体和不影响声明身份的空白。
3. 对规范化字段计算稳定摘要，拼接到 ID；重复同身份声明按稳定出现序号区分。
4. 无关文本移动、函数体修改或行号变化不能改变 ID。
5. 名称、作用域、来源 URI 或声明身份改变时允许 ID 改变；rename 不在本功能中维持跨版本别名。

## ReferenceRecord

表示源文本中的一次名称使用。

| 字段 | 说明 |
|---|---|
| `reference_id` | 当前快照内唯一键 |
| `uri` | 引用所在 URI |
| `range` | 引用名称范围 |
| `name` | 引用短名称 |
| `qualified_name` | 能从语法确定时记录 |
| `scope_id` | 引用所在 lexical scope |
| `candidates` | 候选 `symbol_id` 列表 |
| `condition` | 引用所在条件状态 |
| `kind` | `call`、`type-use`、`attribute-use`、`builtin-use`、`member-use`、`macro-use` 或 `other` |
| `status` | `resolved`、`ambiguous`、`unresolved`、`inactive`、`unknown` |
| `source_version` | 版本关系 |
| `builtin` | builtin-use 的 target/pass/macro/source provenance；无 header 时为 `unresolved` |

引用解析只依据 lexical scope、qualified name、现有 AST/tokens 和条件状态；不执行完整 overload resolution、模板实例化或类型兼容检查。

## ConditionalProvenance

| 字段 | 说明 |
|---|---|
| `region_id` | 对应 `ConditionalRegion` 的稳定快照键 |
| `branch_index` | 分支顺序 |
| `expression_range` | `#if/#elif` 条件范围 |
| `state` | `active`、`inactive`、`unknown` |
| `parent_region_id` | 嵌套条件父级 |
| `condition_tokens` | 原始条件 token 摘要，不记录无关完整源码 |

查询规则：

- active 候选存在时，definition/references 只使用 active 候选；多个 active 候选全部返回。
- 没有 active 候选时，允许使用 unknown 候选并将查询结果视为 degraded。
- inactive 候选保留在 index provenance，不进入当前 definition/references；documentSymbol 也隐藏。
- documentSymbol 展示 active 和 unknown 符号。

## ScopeRecord

| 字段 | 说明 |
|---|---|
| `scope_id` | 当前快照内作用域 ID |
| `kind` | translation-unit、namespace、class、struct、union、function、block、header |
| `name` | 当前作用域名称 |
| `qualified_name` | 完整作用域名称 |
| `parent_id` | 父作用域 |
| `range` | 作用域范围 |
| `symbols` | 直接声明的 SymbolID |

作用域树用于限制同名查找，不代表 C++ 完整 name lookup 或 overload resolution。

## HeaderDependency

| 字段 | 说明 |
|---|---|
| `requester_uri` | 发起 include 的文档/header |
| `include_spelling` | 原始 include token 摘要 |
| `include_range` | include 位置 |
| `target_uri` | 解析成功的 header URI；失败时为空 |
| `search_roots` | 使用的 include roots 来源标识，不记录无关路径到日志 |
| `source_version` | header 内容版本 |
| `status` | `resolved`、`unresolved`、`missing`、`unreadable`、`cycle`、`stale` |
| `dependents` | 反向依赖 URI 集合 |

解析顺序：已打开文档优先，其次是配置 include roots 中的被引用文件；不得扫描未被引用的 workspace 文件。无法解析 include spelling 时使用 `unresolved`，已解析但读取失败时使用 `unreadable`，找不到目标时使用 `missing`；当前文档局部符号仍可建立。检测到 include cycle 时保留 cycle dependency，已加载的非循环部分仍可查询，快照状态为 `degraded`，发布 code 为 `analysis-degraded` 且 reason 为 `include-cycle` 的诊断，不递归读取循环边。

## IndexSnapshot

| 字段 | 说明 |
|---|---|
| `root_uri` | 查询所属文档 URI |
| `source_version` | root 文档版本 |
| `context_version` | context 版本 |
| `generation` / `epoch` | server 分析版本 |
| `parse_status` | parser 状态 |
| `index_status` | `complete`、`partial`、`degraded`、`stale` |
| `ast_root` | 可选的 parser root 引用 |
| `scopes` | scope table |
| `symbols` | SymbolID -> SymbolRecord |
| `references` | ReferenceID -> ReferenceRecord |
| `dependencies` | HeaderDependency 列表 |
| `diagnostics` | 缺失 header、unknown/ambiguous 等可定位状态 |
| `created_at` | 仅内部观测，不进入 LSP payload |

快照是替换单元：新版本成功构建后整体提交；旧版本不与新版本混合。部分/降级快照也只能包含当前文本确认的符号，不能保留已被删除的旧声明。

## Store 与状态转换

```text
no snapshot
    -> building
    -> complete | partial | degraded
complete/partial/degraded
    -> stale (document/header/context/generation changed)
stale
    -> discarded (old task) | building (new task)
```

### 状态转换规则

`IndexSnapshot.index_status` 按以下优先级计算：版本失效时为 `stale`；旧任务完成或文档关闭后的 `discarded` 是 Store 的内部生命周期标记，不是 `IndexSnapshot.index_status` 值；否则存在 missing/unreadable/unresolved header、include cycle、unknown 条件、ambiguous candidate 或 provider error 时为 `degraded`；否则 parser `recovered`、`partial`、`invalid` 或存在 incomplete symbol 时为 `partial`；只有 parser `complete` 且来源、条件和候选均可确认时为 `complete`。`invalid` 但仍有可识别 root/symbol 时保留 partial，完全没有可识别结构时保留 degraded 空快照。

`SymbolRecord.status` 与快照状态独立：`incomplete`/`unresolved`/`ambiguous` 记录单个符号质量，快照通过 `index_status` 汇总；快照失效后所有符号不再可查询。

`Store` 至少提供：

- 按 URI 获取与版本匹配的当前快照；
- 以版本条件原子替换文档快照；
- 删除文档并清理依赖；
- 根据 header URI 查找直接/间接依赖并标记 stale；
- 返回 snapshot status，但不把内部状态扩展到标准 LSP response。

## FixtureCase

`testdata/index/manifest.json` 的每条 fixture 记录至少包含：

| 字段 | 规则 |
|---|---|
| `case_id` | 稳定、唯一的测试材料 ID |
| `category` | `core-symbol`、`multi-file`、`missing-context`、`unknown-condition`、`incomplete`、`version-race`、`header-change` 或 `out-of-scope` |
| `source_uris` | 与 `source_files` 一一对应、顺序一致的 canonical URI；fixture 可使用相对路径，由 validator 按 root 解析 |
| `source_files` | validator 实际加载的 fixture 文件；execute case 必须非空 |
| `include_roots` | provider 使用的 roots 和 available 状态 |
| `context` | language standard、宏、target/pass、context version |
| `document_versions` | 文档变化序列和 header source versions |
| `version_texts` | 与 `document_versions` 一一对应的文档文本；用于验证旧结果不会覆盖新文本 |
| `expected_symbol_count` | 当前 case 应进入 index 的符号数量 |
| `expected_location_count` | 各 LSP 操作应返回的可验证位置数量 |
| `expected_status` | complete/partial/degraded/stale 或 out-of-scope 状态 |
| `lsp_operations` / `queries` | 标准 definition/references/documentSymbol request/result oracle，包含 expected count、URI、`[]`/`null`、malformed params 或 error code |
| `expected_diagnostics` | 稳定 code、范围和 related information 摘要 |
| `validation_mode` / `skip_reason` | `schema-only` case 必须明确说明跳过原因，不计入 passed |

验证器必须按 `case_id` 输出总数、通过数、失败数、skipped 数和失败 ID；SC-001 至 SC-003 使用这些计数，不使用未定义的固定样本数量。

## LSP 映射

| Index 数据 | LSP 结果 |
|---|---|
| SymbolRecord definitions | definition `Location[]` |
| ReferenceRecord candidates | references `Location[]` |
| ScopeRecord + active/unknown SymbolRecord | nested `DocumentSymbol[]` |
| `name_range` | `selectionRange` |
| symbol kind | 标准 `SymbolKind` |
| missing/unknown/ambiguous | diagnostics/log 中的 degraded 状态 |

标准 LSP 结果只使用可验证 URI/range。当前新鲜快照无匹配时返回空数组；没有新鲜快照或查询期间版本失效时返回标准 `null`；过期快照不得映射为结果；取消沿用现有 JSON-RPC `RequestCancelled`。