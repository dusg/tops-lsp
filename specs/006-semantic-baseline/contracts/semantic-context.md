# 契约：SemanticContext 与默认 CompilationCommand

## 目的

`SemanticContext` 是 Go semantic 层的输入快照。它记录分析使用的编译命令、profile、规则、来源和版本。它不是 TypeScript client 的私有消息，也不是 Clang runtime 的命令缓存。

## 默认命令

对 `.tops`、`languageId=tops` 或明确 Tops 模式命中的 Tops-signaled document，且输入编译命令缺失或当前文档规范化 URI 对应的 candidate 解析后没有任何可用 candidate 时，resolver 必须创建默认 `CompilationCommand`；无关 URI 的 candidate 不参与当前文档计数，存在两个或以上非等价 candidate 或同一当前文档候选的明确 language 冲突时保留 candidate 并将整体状态设为 `invalid`，不得合成默认命令：

```text
arch=gcu400
driver=topscc
language=TOPS
language_standard=c++17
pass_kind=host+device
profile_id=gcu400-default
```

- 标准库/Tops headers：工具链默认 include roots；
- 工程 headers：workspace include roots；
- 工具链宏：topscc 内建宏；
- 工程宏：workspace 宏；
- fallback provenance：必须保留；
- 不得返回空 command、no-profile 或把缺失命令本身标记为 `missing`；默认命令所需 roots/macros 或已引用 workspace 依赖缺失时整体状态为 `partial`，只有无法形成语言/命令上下文时整体状态才为 `missing`；单个 pass view 仍可独立使用 `availability=missing`；
- 默认命令不启动 shell、topscc 或 Clang。

默认 Tops 命令的字段来源固定为：`arch`、`driver`、`language`、`language_standard` 和 `pass_kind` 使用 `field_source=fallback_default`；工具链标准库/Tops header roots 和 topscc 内建宏使用 `field_source=toolchain_default`；workspace 工程 header roots 和工程宏使用 `field_source=workspace`。完整 workspace candidate 与 workspace fallback field 的两阶段合并规则、固定 field key 和 `null` 默认规则见 data-model；fallback field 不得补 target/arch、driver/language/standard/pass 或 toolchain roots/macros。`FieldValue` 是字段归一化后的 JSON value，object key 按 UTF-8 字节序排序、array 保持顺序，合法 JSON number 解析为有限精确十进制，去除小数尾零、`-0` 归一化为 `0`、指数形式转换为等价的无指数十进制表示，非有限数字和重复 object key 拒绝，字符串使用 UTF-8 和 JSON escape，相等判断使用递归规范化 JSON 序列化结果。按当前文档规范化 source URI 分组后，等价 `compile_commands` 的判定字段为 `working_directory` 和 `normalized_arguments`；重复等价记录只保留字典序最小的命令指纹。四类 instance canonical object 固定为：`compile_commands={kind,source_uri,working_directory,normalized_arguments}`、`workspace={kind,source_uri,version,working_directory,normalized_arguments}`、`toolchain_default={kind,toolchain_identity,toolchain_version,normalized_roots,macro_version}`、`fallback_default` 使用固定 instance `gcu400-default`；其中 `source_uri` 是规范化 URI，前三类 instance 为对应 canonical object 的 canonical JSON UTF-8 bytes 的 SHA-256 小写 hex，canonical JSON 不带尾部换行。`working_directory` 使用 workspace root 解析为绝对清理路径，不解析符号链接；response file 由 T008 的 expander 以 candidate working directory 为基准深度优先展开，最多 16 层，循环、读取失败、非法 UTF-8、未闭合引号或超深嵌套的 candidate 不可用。字段已获得可用值时 `field_resolution=resolved`；预期工具链或 workspace 来源缺失时仍保留对应的 `field_source` 并使用 `field_resolution=unresolved`，来源类型本身未知时才使用 `field_source=unresolved`；无冲突时 `field_conflict=null`，明确来源出现不同值时使用 `conflict_sort_order=compile_commands < workspace < toolchain_default < fallback_default` 和 `instance` UTF-8 字节序记录 `{sources: []ConflictSource, values: []FieldValue}`，来源选择另按 `source_precedence=compile_commands > workspace > toolchain_default > fallback_default`，要求 `len(sources)=len(values)>=2`，并将 `field_source=invalid`、`field_resolution=invalid`；任意字段冲突都使整体状态为 `invalid`。

`workspace candidate` 是 source provider 提供的完整 command record，可与 `compile_commands candidate` 对同一字段形成冲突；`workspace fallback field` 只为已选择的 command 或合成默认 command 补充缺失的工程 roots/macros，不与已有明确值形成冲突。普通 `.cpp`/`.cc` 不得通过此规则静默转换为 TOPS；它们使用目标中立的 `cpp-default` profile，`pass_kind=host`、host view `availability=available`，device view `availability=not-applicable`，不执行 Tops/GCU400 目标规则。

## Resolver API 不变量

resolver 返回非指针 `SemanticContext` value，不返回 nil。输入命令缺失时按默认命令规则合成 `CompilationCommand`；已有命令但 profile 字段为空时只在 resolver 边界替换适用的 profile，不重新合成或覆盖命令字段来源。Tops language 使用 `gcu400-default`，普通 C++ language 使用目标中立的 `cpp-default`，未知 language 使用 `cpp-default` 并标记 `missing`；后续 semantic API 只接受带非空 `profile_id` 的 SemanticContext。

命令与 profile 的组合规则固定如下：

| 输入命令 | profile 输入 | 处理 |
| --- | --- | --- |
| 缺失，且语言为 Tops | 任意 | 合成默认 `CompilationCommand`，选择 `gcu400-default` |
| 缺失，且语言为普通/未知 C++ | 任意 | 不合成 Tops 命令，选择 `cpp-default`；未知 language 标记 `missing` |
| 已有可用 command | 非空 | 保留 command 和全部字段来源，使用该 profile |
| 已有可用 command | 空 | 只在 resolver 边界替换适用 profile，不改变 command、字段来源或 command status |
| candidate 冲突 | 任意 | 保留 candidate 和 `field_conflict`，整体为 `invalid`，不合成默认命令 |

## CompilationContext adapter and pass views

语言输入按以下优先级转换：compile command 中明确的 `language` 最高；其次是明确的 Tops/C++ 模式，再其次是 `languageId`，最后才使用文件扩展名；明确普通 C++ language 使用 `cpp-default`，明确 Tops language 使用 Tops profile；只有无法确定语言时才使用 `cpp-default` 并标记 `missing`。因此 `.tops` 扩展名遇到明确普通 C++ compile command 时仍使用 `cpp-default`，`.cpp` 遇到明确 Tops 模式或 `languageId=tops` 时使用 Tops 默认命令。

语言信号固定为：compile command 的 `language` 取 `TOPS` 或 `C++`；明确模式取 `tops` 或 `cpp`；`languageId` 取 `tops` 或 `cpp`；文件扩展名取 `.tops`、`.cpp` 或 `.cc`。按优先级选取第一个非空且可识别的信号；空值和未知值视为没有该信号；同一来源的多个明确值冲突时为 `invalid`，低优先级信号不覆盖高优先级信号。典型结果为：`language=TOPS` 选 Tops，`language=C++` 选 C++，无 command 时 `mode=tops` 覆盖 `languageId=cpp` 和扩展名，`mode=cpp` 覆盖 `languageId=tops` 和 `.tops`，无 mode 时 `languageId=tops` 覆盖 `.cpp/.cc`，仅 `.tops` 选 Tops，仅 `.cpp/.cc` 选普通 C++，全部为空或未知时选 `cpp-default` 并标记 `missing`。

命令 candidate 处理顺序固定为：1) 读取 mode、`languageId` 和扩展名等文档级语言信号，但暂不读取无关 candidate 的 language；2) 用 `internal/index.NormalizeSourceURI` 规范化 candidate source URI，只保留与当前文档规范化 URI 相同的 candidate，无关 URI 只保留在 resolver input/provider provenance 中，不写入当前 `SemanticContext`、command、status 或 `field_conflict`；3) 丢弃 source URI、working directory、arguments 或 response file 不可解析的 candidate；4) 只对当前文档候选读取明确 `language`，其优先于文档级信号，同一当前文档候选的明确 language 不同直接形成语言字段冲突，低优先级信号不覆盖或制造冲突；5) 对 `working_directory` 和 `normalized_arguments` 的 canonical JSON 相等 candidate 去重，跨 `compile_commands`/workspace 来源等价时保留全部 provenance，按 `source_precedence=compile_commands > workspace` 选择字段来源但不置 `invalid`；6) candidate 数量为 0 时按语言规则合成默认命令；7) candidate 数量为 1 时选择该 candidate；8) candidate 数量大于 1 且非等价时保留所有 candidate、生成 `ConflictSource` 并将整体状态设为 `invalid`，不得合成默认命令。

candidate 的 canonical object 固定为 `{source_uri,working_directory,normalized_arguments}`，其中 `source_uri` 是 `NormalizeSourceURI` 结果；`working_directory` 使用 workspace root 解析为绝对、清理 `.`/`..` 后的 `/` 分隔路径，不解析符号链接，缺少基准或解析失败时 candidate 不可用；`normalized_arguments` 是有序参数字符串数组，response file 展开由 T008 的 expander 负责，不能只比较 response file 路径。四类 instance 使用以下 canonical object：`compile_commands` 为 `{kind,source_uri,working_directory,normalized_arguments}`；`workspace` 为 `{kind,source_uri,version,working_directory,normalized_arguments}`；`toolchain_default` 为 `{kind,toolchain_identity,toolchain_version,normalized_roots,macro_version}`；`fallback_default` 使用固定 instance `gcu400-default`。前三类 instance 是对应 canonical object 的 canonical JSON UTF-8 bytes 的 SHA-256 小写 hex，canonical JSON 不带尾部换行；`workspace.version`、toolchain 三个 metadata token 和 `normalized_roots` 的来源/缺失规则见 data-model。

现有 `server.CompilationContext.PassKind` 的单值输入按以下规则转换：

| 输入 | SemanticContext `pass_kind` | pass views |
| --- | --- | --- |
| `host` | `host` | `host` |
| `device` | `device` | `device` |
| 默认命令或明确组合命令 | `host+device` | 独立的 `host` 与 `device` 视图 |

`host+device` 不是一个可用于调用判断的第三种执行空间。SemanticContext 必须分别保存：

- `pass_views.host`：host 可调用性和 host 诊断规则；
- `pass_views.device`：device 可调用性、address-space/vector 规则和 device 诊断规则。

发布 diagnostics 时，组合命令拆成 `Diagnostic.data.passKind=host` 或 `device`；不依赖具体 pass 的 context/incomplete diagnostic 使用 `passKind=not-applicable`；内部不得把两侧 facts 合并成一个 pass。

每个 pass view 必须包含：

| 字段 | 说明 |
| --- | --- |
| `pass_kind` | `host` 或 `device` |
| `availability` | `available`、`not-applicable` 或 `missing` |
| `status` | `resolved`、`partial`、`missing` 或 `invalid` |
| `rules_available` | 当前 view 可执行的规则集合 |
| `provenance` | pass、profile、toolchain/workspace 来源 |
| `facts` | 当前 view 的 callable、address-space、vector 和诊断事实 |

host-only、device-only、两侧均可用和单侧缺失都必须保留独立状态；`not-applicable` 只用于普通 C++ 等不需要该 pass 的文档，Tops pass 来源缺失使用 `missing`；合并 snapshot 时不得用一侧状态覆盖另一侧。

组合结果表：

| 文档/输入 | host availability | device availability | 总体状态 |
| --- | --- | --- | --- |
| 普通 C++ + `cpp-default` | `available` | `not-applicable` | `resolved` 或来源缺失时 `partial` |
| Tops `host` | `available` | `not-applicable` | 按 host 来源状态 |
| Tops `device` | `not-applicable` | `available` | 按 device 来源状态 |
| Tops `host+device` | `available` 或 `missing` | `available` 或 `missing` | 任一所需 view 缺失时 `partial`，冲突时 `invalid` |

## 状态

| 状态 | 进入条件 | 规则能力 |
| --- | --- | --- |
| `invalid` | target/参数来源冲突 | 不选择冲突一方，不发布冲突上下文下的确定性目标 Error |
| `missing` | 无法形成语言/命令上下文；可识别 Tops 但输入命令缺失不属于此状态，因为会先创建默认命令 | 只执行目标无关基础规则 |
| `partial` | 部分来源、header、宏或 pass 信息缺失 | 只执行事实完整的规则 |
| `resolved` | 真实命令完整，或默认 Tops 命令字段及所需来源完整 | 使用 profile 提供的基础和目标规则 |

对 Tops 文档，`gcu400-default` 在无 target 时可以与 `resolved` 或来源缺失时的 `partial` 同时存在；在 target 冲突时与 `invalid` 同时存在。普通 C++ 和未知 language 使用 `cpp-default`。

## 状态优先级规则表

| 条件 | 最终状态 | 目标规则 |
| --- | --- | --- |
| target 或参数来源冲突 | `invalid` | 不发布冲突上下文下的确定性目标 Error |
| 无法形成语言/命令上下文 | `missing` | 只执行目标无关基础规则 |
| 默认核心字段可生成，但 toolchain roots/macros 或已引用 workspace 依赖缺失 | `partial` | 只执行事实完整的规则 |
| 真实命令完整，toolchain roots/macros 和已引用 workspace 依赖可用 | `resolved` | 使用具体 profile |
| 无命令的 Tops 文档，但默认核心字段、toolchain roots/macros 可用，未引用缺失 workspace 依赖 | `resolved` | 使用 `gcu400-default` |

workspace 中未被当前文档引用的缺失 header/macro 不自动降低状态；缺失来源必须在被引用时才影响当前 SemanticContext。

判断顺序固定为：target/参数冲突优先于其他状态；其次判断能否形成语言/命令上下文；再判断工具链默认 roots/macros 和已引用 workspace 依赖；只有全部所需来源可用时才使用 `resolved`。

## 版本

SemanticContext 必须携带 document version、context version、source/header versions、analysis generation 和 epoch。版本不匹配时丢弃 semantic snapshot，不发布旧诊断。

## 兼容性

新增 SemanticContext 字段属于 Go server 内部模型；LSP 只通过标准 diagnostics 的 `Diagnostic.data` 暴露必要的 `profileId`、`contextStatus`、`passKind` 和版本字段。旧 client 可以忽略新增 data 字段。
