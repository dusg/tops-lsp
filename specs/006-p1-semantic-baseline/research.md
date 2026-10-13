# 研究记录：P1.4 通用 semantic baseline

## 研究目的

确定 P1.4 的最小语义范围、输入边界、诊断模型和验证方式，避免把 parser、index、target semantic 和完整 C++ 语义混为一个阶段。

## 已确认的来源边界

- `specs/003-basic-lsp-transport/` 只负责 Go server、stdio、生命周期、文档同步、取消、基础错误和日志，不负责语义服务。
- `specs/004-tops-cpp-tokenizer-parser/spec.md` 将模板实例化、完整类型系统、重载决议和调用合法性列为 parser 阶段范围外，并将 completion、hover、definition、references 和索引留给后续功能。
- `specs/005-ast-symbol-index/spec.md` 负责 AST/index、header 依赖和 definition/references/documentSymbol；完整类型检查、模板实例化、重载决议以及 completion/hover 不属于该功能。
- `specs/001-tops-lsp-roadmap/` 原有 P1.4 曾把完整 C++ 语义写成单一里程碑，但没有独立 semantic 工件。P1.4 现改为通用 semantic baseline，完整语义能力不在本阶段一次承诺。

以上结论来自当前 checkout 中对应规格的范围和排除项；它们不表示 Go semantic 已经实现。

## 方案决定

### 1. Semantic 输入

P1.4 只接收 parser/index 输出和 `ContextSeed` 或 resolved `CompilationContext`。它不解析 raw compiler argv，不读取 profile 名称推断宏值，不启动 `topscc`、Clang 或 clangd。

### 2. ContextSeed 与完整 context

P1 阶段允许使用 `partial` ContextSeed 提供局部语义结果。缺少目标、header 或宏值时，semantic 只能保留可确认结果并发布 `analysis-degraded`。P2.0 完成完整 driver 归一化和 target 选择后，P1.4 规则可以消费 resolved context，但不因此自动扩大 P1.4 的规则范围。

### 3. 规则范围

首批规则只覆盖声明/作用域/名称、基础类型、通用 host/device 调用边界、已确认地址空间和基础向量约束。模板实例化、完整重载决议、复杂隐式转换、完整常量求值、链接和代码生成保留为后续候选能力。

### 4. 诊断来源

Go semantic 产生最终用户诊断。Clang 只作为开发机离线对照，保存输入 provenance、双方分类、range、severity 和差异原因；对照命令失败不能让 Go 基线测试变成通过或失败的隐藏条件。

### 5. 结果版本

semantic snapshot 绑定 document version、context version 和 parser/index snapshot version。发布前必须检查版本；旧结果只记录 stale 事件，不覆盖新的 diagnostics 或导航结果。

## 未解决事项

- 各 GCU profile 的具体地址空间、向量对齐和 host/device 限制需要在 P2.3/P2.4 按活动 header、完整宏分支和目标命令逐项核对。
- 完整模板语义、重载决议和常量求值的边界需要后续独立规格定义；P1.4 不用默认行为替代这些规则。
