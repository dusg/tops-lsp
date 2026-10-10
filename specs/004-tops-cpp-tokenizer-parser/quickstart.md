# 快速验证：Tops C++ Tokenizer 与 Parser

## 目的

本指南验证 Go parser、server diagnostics、语法测试材料、未完成输入和 Clang 离线对照。基础 parser 验证不需要 GCU 设备、测试机 Docker、clangd 或运行时 Clang。

相关契约：

- [parser 契约](contracts/parser.md)
- [LSP diagnostics 契约](contracts/lsp-diagnostics.md)
- [Clang 离线对照契约](contracts/clang-comparison.md)
- [数据模型](data-model.md)

## 前提

在开发机执行：

```sh
cd /home/carl.du/work/tops-lsp
go version
```

当前已核对的 Go 基线为 `go1.26.8 linux/amd64`。实现前不需要安装 GCU 驱动、进入测试机或启动 clangd。

用户编译器和参数来源核对：

```sh
command -v topscc
readlink -f "$(command -v topscc)"
topscc --version
topscc --dryrun -arch gcu400 -x tops -fsyntax-only /dev/null
```

`topscc --dryrun` 只用于确认 wrapper 参数展开；parser 测试材料必须把归一化后的 context 传给 parser，不能让 parser 自己启动或解析 topscc。

可选的 Clang 离线对照工具：

```sh
/home/carl.du/work/llvm-project/build/bin/clang --version
/home/carl.du/work/llvm-project/build/bin/llvm-lit --version
```

当前本地工具输出分别为 Clang `18.1.8` 和 llvm-lit `18.1.8dev`。如果工具不可用，只跳过离线对照，不跳过 Go parser 单测。

## 基础回归

```sh
go test ./...
go test -race ./...
go vet ./...
```

预期：现有 transport、protocol、document、server、logging、integration 测试和新增 parser 测试全部通过；race 检查不报告 data race；`go vet` 不报告问题。

2026-10-08 实际验证：`go test ./...`、`go test -race ./...` 和 `go vet ./...` 均通过。

## Parser 单测

实现阶段新增 `internal/parser/` 后执行：

```sh
go test ./internal/parser/...
go test ./internal/parser/... -run 'Test.*(Token|Parse|Recovery|Diagnostic|Condition)'
```

至少检查：

1. standard C++11/14/17 声明、模板、表达式和语句；
2. Tops qualifier、attribute、向量 spelling 和 `<<<...>>>` launch；
3. `#if/#elif/#else/#endif` 的 active/inactive/unknown；
4. 未闭合 literal、comment、delimiter、template、attribute、launch 和宏条件；
5. UTF-16、CRLF、EOF zero-width、unexpected token 和 related range；
6. unsupported syntax 的 opaque/recovery node；
7. parser 在任何测试材料上不 panic，并保留后续独立声明。

## Server diagnostics 验证

```sh
go test ./internal/server/... -run 'Test.*(Diagnostic|Parse|Document|Stale|Close)'
go test ./integration/... -run 'TestLSPProcess.*Diagnostic'
```

验证流程：

1. `initialize` 成功；
2. `didOpen` 一个完整或不完整 Tops 测试材料；
3. 读取一条 `textDocument/publishDiagnostics` frame；
4. 检查 URI、document version、code、severity、LSP UTF-16 range 和 source；
5. 发送合法 `didChange`，确认只收到新版本结果；
6. 发送非法 change，确认文本、版本和旧结果不被未应用输入覆盖；
7. 发送 `didClose`，确认旧 parse result 不再发布；
8. 检查 stdout 只有 LSP frame，日志不含完整源文本或诊断 token 内容。

## 测试材料检查

测试材料根目录为 `testdata/parser/`，manifest 至少覆盖：

```text
positive/      standard C++ 和 Tops 正向结构
negative/      非法 token、属性、launch、directive 和 unsupported syntax
incomplete/    EOF 截断与错误恢复
macros/        active/inactive/unknown 条件
topsop/        只读 cc_kernel 缩减片段或 manifest 引用
expected/      token/AST/diagnostic golden 摘要
manifest.json  context、预期结果和 对照验证 选项
```

执行测试材料汇总测试：

```sh
go test ./internal/parser/... -run TestFixture
```

每个失败条目必须输出测试材料 ID、context status、diagnostic code/range 和恢复状态；不要输出完整源代码。

## Clang 对照

对照只在测试材料清单标记 `compare` 时执行。先确认 manifest 中的 `driver_kind`、raw/normalized argument provenance 和 context status；`topscc --dryrun` 只在离线准备阶段执行。再确认直接 Clang 参数：

```sh
/home/carl.du/work/llvm-project/build/bin/clang -### <same-source-and-context>
/home/carl.du/work/llvm-project/build/bin/clang -E -dD <same-source-and-context>
/home/carl.du/work/llvm-project/build/bin/clang -dM -E <same-source-and-context>
/home/carl.du/work/llvm-project/build/bin/clang -fsyntax-only <same-source-and-context>
```

适用时再执行：

```sh
/home/carl.du/work/llvm-project/build/bin/clang -Xclang -ast-dump=json -fsyntax-only <same-source-and-context>
```

对照 GCU/EFGCU Tops 测试材料时，沿用源码测试中已经出现的 target/pass 形式，并把完整参数写入 comparison record。比较接受/拒绝、宏区域、主要 AST 节点和诊断 range；不比较完整 message 文案。

已核对的参考测试入口：

- `clang/test/DTU_test/tcle/vector_op_parser.cc`：GCU400/GCU450 vector parser negative syntax；
- `clang/test/CodeGenEFGCU/launch-bounds-e2e.cc`：EFGCU500 Tops launch attribute pipeline；
- `clang/test/DTU_test/topscc/attribute/launch_bounds.tops`：GCU400/GCU450 attribute syntax。

## `cc_kernel` corpus 验证

优先使用以下只读路径中的片段或 manifest 条目：

```text
topsop/topsop/lib/kernel/cc_kernel/range/range_kernel.tops
topsop/topsop/lib/kernel/cc_kernel/range/range_host.tops
topsop/topsop/lib/kernel/cc_kernel/mhc_pre/mhc_pre_n512_kernel_gcu400.tops
topsop/topsop/lib/kernel/cc_kernel/topp_renorm_probs/topp_renorm_probs_kernel_gcu400.tops
```

先运行 parser-only 测试材料；只有 include、target、pass 和宏 context 完整时才运行 Clang syntax-only。不要修改 `topsop` 源码，不使用设备 workload 代替 parser 验证。

## 失败处理

- Go parser panic：保留最小测试材料和版本信息，先修复 recovery，再扩大 corpus。
- range 不一致：先检查统一位置映射、UTF-16、CRLF 和 EOF 规则，不调整单个测试材料的预期位置绕过问题。
- Clang/Go 分类不一致：保存两侧 context 和 comparison record；确认是否为 target/header 差异后再修改 grammar。
- publisher 写失败：检查 stdout frame 和 writer 互斥，不把 parser 诊断写入日志或 stderr 协议流。
- 设备或测试机问题：停止本 feature 的 parser 验证；parser 单测和本地测试材料不应转移到测试机。
