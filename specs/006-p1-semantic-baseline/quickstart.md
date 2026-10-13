# Quickstart：P1.4 semantic baseline

本指南用于验证 P1.4 语义规则和文档契约，不启动 clangd，不把 Clang 作为 Go server runtime。

## 前置条件

- 工作区根目录：`/home/carl.du/work/tops-lsp`
- 已完成 P1.1 transport、P1.2 parser 和 P1.3 index 的 Go 基线。
- 可选：本地 Clang 和 Tops headers，用于离线对照；缺少时只跳过对照，不跳过 Go 测试。

## 文档检查

```bash
cd /home/carl.du/work/tops-lsp
test -f specs/006-p1-semantic-baseline/spec.md
test -f specs/006-p1-semantic-baseline/plan.md
test -f specs/006-p1-semantic-baseline/research.md
test -f specs/006-p1-semantic-baseline/data-model.md
test -f specs/006-p1-semantic-baseline/contracts/semantic-boundary.md
rg -n 'ContextSeed|analysis-degraded|semantic-unsupported|Out of Scope' specs/006-p1-semantic-baseline
git diff --check
```

## Go 测试

实现完成后在开发机执行：

```bash
cd /home/carl.du/work/tops-lsp
go test ./internal/semantic
go test ./internal/server ./integration -count=1
go test ./...
go test -race ./...
go vet ./...
```

## 规则检查

每个规则域至少检查：

1. 有效声明/名称/类型或 Tops 调用；
2. 一个确定错误；
3. 一个不完整或缺少 ContextSeed 的输入；
4. document/context/index 版本变化后的旧结果丢弃。

## 对照记录

在配置完整的开发机上，使用与 Go 输入相同的 source、language standard、include roots、宏和 target/pass 运行 Clang `-fsyntax-only`。保存分类、range、severity、上下文和 known difference；不把 Clang 输出转换为 Go server 诊断。

## 通过条件

- P1.4 规则测试和适用 LSP diagnostics 测试通过。
- partial/missing context 没有产生未经确认的目标结论。
- 旧 semantic snapshot 不覆盖新结果。
- 未实现的完整语义仍保持可见状态，没有出现在 `supported` capability 中。
