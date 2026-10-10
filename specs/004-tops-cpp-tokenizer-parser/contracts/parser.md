# 契约：Go Tokenizer 与 Parser

## 所有权

`internal/parser` 是标准 C++ 基础语法、Tops syntax spelling、宏条件结构、错误恢复和 parser diagnostics 的唯一所有者。

- `topscc`/直接 Clang 只在上游 `CompilationContext` 和离线对照中出现；parser 不解析 raw compiler argv。
- TypeScript client 只接收 LSP 结果，不扫描关键字、不计算宏条件、不重排诊断。
- `internal/document` 只提供已验证的文本/版本快照，不实现 grammar。
- parser 不读取文件、不启动进程、不访问设备、不写日志；外部数据流由 server、context resolver 和离线对照流程负责。

## 输入契约

```text
Parse(text, context) -> ParseResult

text: UTF-8 document text
context:
  document_uri
  language_id
  language_standard: c++11 | c++14 | c++17 | unknown
  driver_kind: topscc | clang | clang++ | unknown
  compiler_context_status: resolved | partial | missing | invalid
  argument_provenance: opaque normalized-context reference
  predefined_macros: map<string, MacroValue>
  target_profile: opaque string
  pass_kind: opaque string
  include_roots_available: bool/unknown
  document_version: non-negative integer
  context_version: non-negative integer
```

`ParseContext` 只携带已归一化的 compiler provenance 和语义参数；不包含 raw argv、response file 内容、工具进程句柄或 `topscc --dryrun` 输出。driver 默认值必须由上游 context resolver 写明来源，parser 不自行补全。

输入不完整、包含非法 UTF-8 或缺少 target/include context 时仍必须返回结果；不能以 panic 或 nil 结果代替恢复状态。

## 输出契约

`ParseResult` 必须包含：

- 原文 token 序列和 EOF token；
- `translation-unit` 根节点；
- conditional region 索引；
- 按稳定顺序排列的 parser diagnostics；
- `complete`、`recovered`、`partial` 或 `invalid` 状态；
- 输入的 document/context version。

结果是确定性的：相同 text、context 和版本输入必须产生相同 token kind、节点 kind、诊断 code、range 和顺序。

## Grammar slice

### Standard C++

支持 C++11/14/17 baseline 的 lexical token、声明、类型、函数、类/struct/enum/union、namespace/using、模板、表达式、语句、lambda、属性和 include directive。完整语法范围以 feature spec 的 Supported Syntax 表为准。

### Tops

parser 必须为执行空间、存储空间、向量/数值、启动/资源属性、内建变量和 kernel launch 建立独立节点或可识别属性：

- `__global__`、`__device__`、`__host__`、`__cooperative__`、`__sp__`、`__scalar_only__`；
- `__shared__`、`__constant__`、`__local__`、`__private__`、`__cluster_shared__`、`__local_stack__`、`__restrict__`；
- `__thread_dims__`、`__cluster_dims__`、`__launch_bounds__`、`__maxnreg__`、`__block_tile__`、`__valigned__`；
- `__vector`、`__vector2`、`__vector4`、`__vector8`、`__fp16`、`__bf16` 和已列出的扩展类型 spelling；
- `callee<<<config-list>>>(argument-list)`。

节点保存 spelling 和原文范围，但不判断资源上限、类型可转换性、DTE 时序、硬件性能或 CodeGen 结果。

## Macro condition contract

- 识别 `#if/#ifdef/#ifndef/#elif/#else/#endif` 的嵌套关系。
- 条件状态只有 `active`、`inactive`、`unknown`。
- 未传入的宏不能按架构名称自动赋值。
- inactive 分支保留 token 和区域节点，但过滤普通语法诊断。
- unknown 分支保留两侧节点并产生一个 `tops-syntax-unknown-condition`；不复制一组相同诊断。
- directive 嵌套错误总是诊断，即使它位于当前 inactive body。
- 不执行完整跨文件宏展开、token paste、stringification 或递归 replacement 语义。

## Recovery contract

恢复同步点按上下文选择：

1. literal/comment：消费到结束 delimiter 或 EOF；
2. attribute/parameter/template/launch：优先寻找对应 close token、逗号、分号或 EOF；
3. declaration/statement：优先寻找分号、右花括号、下一个顶层 declaration 或预处理 directive；
4. conditional region：优先寻找下一个 branch directive 或 `#endif`。

每个恢复节点最多产生一个能解释根因的主诊断；恢复后可独立解析的声明必须拥有独立节点和范围。

## Diagnostic contract

稳定 code 至少包括：

- `tops-syntax-unexpected-token`
- `tops-syntax-missing-token`
- `tops-syntax-unterminated`
- `tops-syntax-invalid-literal`
- `tops-syntax-invalid-directive`
- `tops-syntax-unknown-condition`
- `tops-syntax-unsupported`
- `tops-syntax-invalid-tops-attribute`

位置必须来自原文，统一转换为 LSP UTF-16。missing token 使用零宽 insertion range；unexpected token 覆盖完整 token；EOF 未闭合结构关联 opening delimiter；宏相关诊断不能引用展开后不存在的位置。

## Determinism and privacy

parser 不记录完整源代码或 token spelling 到日志。测试可检查完整 token/AST；生产日志只记录 server 层的文档脱敏 ID、版本、状态、diagnostic count 和 code 摘要。
