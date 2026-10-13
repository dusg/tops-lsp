# 契约：P1.4 semantic 边界

## 所有权

| 组件 | 负责 | 不负责 |
| --- | --- | --- |
| `internal/parser` | token、可恢复 AST、条件区域和 parser diagnostics | semantic 合法性、目标限制、LSP 请求 |
| `internal/index` | 声明、作用域、引用、header 和 index snapshot | 完整类型检查、host/device 合法性、诊断发布 |
| `internal/semantic` | P1.4 规则、semantic snapshot、稳定诊断和降级状态 | raw argv、transport、Clang runtime、客户端语义 |
| `internal/server` | 组装 context、版本检查、取消和 diagnostics 发布 | 在客户端重复规则 |
| TypeScript client | 请求转发、结果展示、配置和生命周期 | 关键字扫描、目标判断、诊断等级 |

## 输入

`internal/semantic` 接收：

- parser `ParseResult`；
- P1.3 `IndexSnapshot`；
- `ContextSeed` 或 resolved `CompilationContext`；
- document version、context version 和 index snapshot version。

缺少任何必要来源时，semantic 必须输出部分结果或 `analysis-degraded`，不能使用默认架构。

## 诊断规则

- 确定性规则错误使用稳定 `tops-semantic-*` code。
- 上下文、header、宏或 AST 不足使用 `analysis-degraded`。
- 本阶段未实现的语义使用 `semantic-unsupported`，不得伪装为成功。
- 主 range 指向原始文档；related information 指向可验证的声明或来源。
- 诊断发布前必须通过 document/context/index 版本检查。

## LSP 边界

P1.4 只提供 diagnostics 数据和 server 内部 semantic 结果。`completion`、`hover` 以及其他请求由 P1.5 或后续阶段消费这些结果；semantic package 不声明 capability、不写 transport、不增加私有 response 字段。

## 对照验证

Clang 对照必须使用同一 source、language standard、include roots、宏、target/pass 和输入版本。对照结果只写入记录，不作为 Go server 运行时输入。差异必须记录分类、range、severity、上下文和原因状态。
