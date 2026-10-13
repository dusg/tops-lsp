# Feature Specification: P1.4 通用 semantic baseline

**Feature Branch**: `006-p1-semantic-baseline`

**Created**: 2026-10-11

**Status**: Draft

**Input**: 在 P1.2 tokenizer/parser 和 P1.3 AST/index/navigation 之上建立可独立验收的标准 C++ 与通用 Tops 语义诊断基线。

## 目的

本功能让 Go server 对已经能够恢复解析的语法结构提供一组范围明确的通用语义诊断。它消费 `ContextSeed` 和 P1.2/P1.3 的结果，不解析 raw compiler argv，不启动 `topscc` 或 Clang，不把 Clang message 转发给用户。

本功能不是“完整 C++ 语义”交付。模板实例化、完整重载决议、完整常量求值、完整访问控制和链接/代码生成语义保留为后续候选能力。

## User Stories & Testing

### User Story 1 - 检查声明、作用域和名称

作为 Tops C++ 开发者，我希望已识别的声明、作用域和名称引用得到一致的基本诊断，从而能在进入复杂模板语义前发现明显错误。

**Independent Test**: 使用单文件有效、无效和不完整输入，检查声明位置、作用域、名称查找、重复声明和未解析名称的 code、range、severity 和降级状态。

### User Story 2 - 检查基础类型关系

作为 Tops C++ 开发者，我希望基础类型、指针/引用、返回值、参数和已确认向量类型的明显不匹配得到诊断。

**Independent Test**: 使用内置类型、指针/引用、函数参数/返回值和当前测试材料中的向量类型，分别验证有效、无效和缺少类型来源的输入。

### User Story 3 - 检查通用 Tops 调用边界

作为 Tops kernel 开发者，我希望通用的 host/device 调用边界、存储空间声明和基础向量约束在有足够上下文时得到解释。

**Independent Test**: 使用已核对的 Tops spelling 和 `ContextSeed`，检查 host/device 调用、地址空间不匹配、基础向量使用和未配置目标的降级行为。

### User Story 4 - 在编辑过程中保持可解释结果

作为正在编辑不完整 kernel 的开发者，我希望 semantic 结果不会覆盖新版本，也不会把 partial context 当成完整结论。

**Independent Test**: 对同一文档提交递增版本、删除括号/参数/函数体并切换 ContextSeed，检查旧结果丢弃、保留 parser/index 结果和 `analysis-degraded` 诊断。

### User Story 5 - 与 Clang 做离线对照

作为语言工具维护者，我希望能够记录 Go semantic 与 Clang 对同一输入的差异，但不让 Clang 进入 server runtime。

**Independent Test**: 使用相同源文本、language standard、include roots、宏和 target/pass provenance 运行离线对照，保存双方上下文、诊断 code/range/severity 和差异原因。

## Scope and Boundaries

### In Scope

- 支持 AST 中已经识别的声明、定义、作用域、名称引用和基础类型摘要。
- 基础名称查找、重复声明、未解析名称和作用域不匹配诊断。
- 内置标量类型、指针、引用、函数参数/返回值和已确认向量类型的基础兼容性检查。
- 通用 Tops host/device 调用边界、已确认存储空间声明和基础向量/对齐规则；目标特有资源限制留给 P2。
- 使用 `ContextSeed` 时提供部分结果，并明确标记 `analysis-degraded`；使用 resolved context 时记录来源和 target profile。
- 标准 `LanguageDiagnostic` 模型、稳定 code、LSP range、severity、related information 和 document/context version。
- Go semantic 单测、server/LSP diagnostics 测试和 Clang 离线对照记录。

### Out of Scope

- 完整 C++11/14/17 语义覆盖的宣称。
- 模板实例化、substitution failure、完整 overload resolution、隐式转换全覆盖、完整常量求值、ODR、完整访问控制、链接和代码生成。
- P2 的 wrapper 参数解析、多目标选择、架构属性上限、DTE/barrier/GEMM/同步硬件语义和设备执行结果。
- `completion`、`hover` 的展示逻辑；P1.5 消费本功能的语义结果并提供请求处理。
- Clang/clangd 运行时后端、TypeScript 语义判断和客户端关键字扫描。

## Semantic Rules

| 规则域 | P1.4 交付 | 不承诺 |
| --- | --- | --- |
| 声明与作用域 | 已解析声明的合法性、作用域关系、重复声明和基本名称查找 | 完整 C++ lookup、ADL 和模板依赖名称 |
| 基础类型 | 标量、指针、引用、函数参数/返回值和已确认向量的明显兼容性 | 完整隐式转换、用户定义转换和所有标准库类型 |
| Tops 调用边界 | 已确认 host/device qualifier 的明显调用方向错误 | 复杂模板实例化、跨目标代码生成行为 |
| 地址空间 | 已确认存储空间声明和明显指针空间不匹配 | 目标资源分配、地址空间布局和设备执行正确性 |
| 向量与对齐 | 已有 header/test 证据支持的类型、成员和基础对齐规则 | 跨代向量运算等价性、性能和硬件指令语义 |
| 上下文状态 | `resolved` 时使用 profile；`partial`/`missing` 时保留部分结果和降级原因 | 从 profile 名称、相邻架构或缺失 header 猜测语义 |

## Requirements

- **FR-001**: semantic layer MUST 只消费 parser/index 输出和 `ContextSeed` 或 resolved `CompilationContext`，不得解析 raw compiler argv。
- **FR-002**: 每条规则 MUST 记录适用上下文、输入节点、诊断 code、severity、主 range、related information 和 recoverability。
- **FR-003**: 名称查找 MUST 按作用域和 qualified name 工作；未解析或歧义结果不得按短名称静默绑定。
- **FR-004**: 基础类型检查 MUST 区分确定不兼容、来源缺失和规则未实现，分别使用稳定诊断或 `analysis-degraded`。
- **FR-005**: host/device、地址空间和向量规则只有在 `ContextSeed` 提供足够事实时才生成确定性错误；缺少目标或 header 时不得猜测。
- **FR-006**: semantic 结果 MUST 关联 document version、context version 和 parser/index snapshot version；旧结果不得发布。
- **FR-007**: 不完整 AST 仍应保留可确认的 semantic 结果；恢复节点之后的错误必须有独立位置，不得清空整份结果。
- **FR-008**: `analysis-degraded` 必须通过稳定诊断 code 和日志表达；不得把内部状态放入私有 LSP response 字段。
- **FR-009**: Clang 对照 MUST 使用相同输入和 provenance，结果只写入对照记录，不进入 Go server runtime。
- **FR-010**: 本功能的测试 MUST 在没有 GCU 设备、clangd 和 Clang runtime 的开发机上运行；Clang 对照命令单独执行且可跳过。
- **FR-011**: 未在本规格规则表中实现的语义 MUST 保持 `candidate`、`blocked` 或 `unsupported`，不得写成 `supported`。

## Diagnostic Codes

| Code | 用途 |
| --- | --- |
| `tops-semantic-duplicate-declaration` | 同一作用域内发现不允许的重复声明 |
| `tops-semantic-unresolved-name` | 名称在当前可确认作用域中无法解析 |
| `tops-semantic-scope-mismatch` | 名称或声明与当前作用域不匹配 |
| `tops-semantic-type-mismatch` | 基础类型、参数、返回值或指针/引用明显不兼容 |
| `tops-semantic-host-device-boundary` | 已确认 host/device 调用方向不合法 |
| `tops-semantic-address-space` | 已确认地址空间声明或指针空间明显不匹配 |
| `tops-semantic-vector-constraint` | 已确认向量类型或对齐规则不满足 |
| `analysis-degraded` | 上下文、header、宏或 AST 不足，结果受限 |
| `semantic-unsupported` | 输入需要本阶段未承诺的语义规则 |

## Success Criteria

- 每个 P1.4 规则域至少有有效、无效和不完整输入，并能重复获得稳定 code、range 和 severity。
- ContextSeed 为 partial/missing 时，server 不生成未经上下文确认的确定性目标结论。
- 文档、上下文或 index 版本变化后，旧 semantic 结果不会覆盖新结果。
- Go semantic 测试、适用的 server/LSP diagnostics 测试和 `go vet` 通过；Clang 差异有记录时不阻塞 Go 基线测试。
- P1.4 完成后仍明确保留本规格 Out of Scope 中的完整语义缺口。

## Compatibility and Failure Handling

- 缺少 ContextSeed、header 或 profile 时返回部分 parser/index 结果，并发布 `analysis-degraded`。
- 参数结构错误使用既有标准 JSON-RPC/LSP 错误；semantic 规则错误使用 `LanguageDiagnostic.code`。
- 请求取消或文档关闭后停止发布旧结果。
- 任何无法安全判断的规则都保留为 `candidate`/`unsupported`，不使用默认架构或 Clang fallback。
