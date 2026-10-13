# Requirements Checklist: P1.4 通用 semantic baseline

## 范围

- [ ] 规格明确列出 P1.4 交付和不交付内容。
- [ ] parser、index、semantic、server 和 TypeScript client 的所有权没有重叠。
- [ ] ContextSeed、resolved CompilationContext 和 partial/missing 行为已定义。

## 诊断

- [ ] 每条规则有稳定 diagnostic code、severity、range、版本和降级行为。
- [ ] 不完整输入不会清空 parser/index/semantic 的可确认结果。
- [ ] stale 结果不会覆盖新版本，取消和关闭文档后不再发布旧结果。

## 验证

- [ ] 每个规则域都有有效、无效和不完整测试。
- [ ] Go 测试不依赖 GCU 设备、clangd 或 Clang runtime。
- [ ] Clang 只用于开发机离线对照，并保存上下文和差异原因。
- [ ] 未实现的完整语义保持 candidate、blocked 或 unsupported。
