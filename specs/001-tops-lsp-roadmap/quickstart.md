# Quickstart：验证 Tops C++ Language Server Roadmap 文档

本指南验证的是 roadmap 文档、源码事实和决策门，不启动尚未实现的 Go 服务器或 VS Code 扩展。

## 前置条件

- 工作区根目录：`/home/carl.du/work/tops-lsp`
- Tops/LLVM 参考 checkout：`/home/carl.du/work/llvm-project`
- 用户编译器优先使用 PATH 中的 `topscc`；本地工具使用 `/home/carl.du/work/llvm-project/build/bin/` 中的 `clang` 和 `llvm-lit` 做语言对照和 Tops 测试验证。
- 本地可用 `rg`、`jq` 和 Git。
- 不需要 Go module、npm package 或测试机设备；当前 feature 是文档交付。

## 1. 检查文档产物

```bash
cd /home/carl.du/work/tops-lsp
test -f specs/001-tops-lsp-roadmap/spec.md
test -f specs/001-tops-lsp-roadmap/plan.md
test -f specs/001-tops-lsp-roadmap/research.md
test -f specs/001-tops-lsp-roadmap/data-model.md
test -f specs/001-tops-lsp-roadmap/quickstart.md
test -f specs/001-tops-lsp-roadmap/contracts/capability-matrix.md
test -f specs/001-tops-lsp-roadmap/contracts/lsp-boundary.md
test -f specs/001-tops-lsp-roadmap/contracts/compiler-driver.md
test -f specs/001-tops-lsp-roadmap/contracts/roadmap-gates.md
jq -e . .specify/feature.json >/dev/null
git diff --check
```

预期结果：所有 `test`、`jq` 和 `git diff --check` 命令返回成功。

## 1A. 核对 topscc driver

```bash
cd /home/carl.du/work/tops-lsp
command -v topscc
readlink -f "$(command -v topscc)"
topscc --version
topscc --dryrun -arch gcu400 -x tops -fsyntax-only /dev/null
```

预期：确认 topscc wrapper 路径和版本，并记录默认 `-std=c++11`、`-Tops`、`--include tops.h`、Tops include 根、`--cuda-gpu-arch=gcu400` 和 host/device 命令。LSP runtime 不运行该命令；它只用于开发机离线核对。

## 2. 检查规格与计划没有残留占位符

```bash
cd /home/carl.du/work/tops-lsp
! rg -n '\[FEATURE NAME\]|\[NEEDS CLARIFICATION|TODO\(|\[未使用时删除\]|\[例如' \
  specs/001-tops-lsp-roadmap/spec.md \
  specs/001-tops-lsp-roadmap/plan.md \
  specs/001-tops-lsp-roadmap/research.md \
  specs/001-tops-lsp-roadmap/data-model.md \
  specs/001-tops-lsp-roadmap/contracts
```

预期结果：命令没有匹配输出，表示本 feature 的计划和设计文档没有未解决占位符。引用其他文档中的 `TODO` 事实时，必须在 roadmap 中改写为明确的“待验证”状态，不能保留模板占位符。

## 3. 核对 Tops 语言入口

```bash
cd /home/carl.du/work/llvm-project
rg -n 'TYPE\("tops"|TYPE\("tops-cpp-output"' clang/include/clang/Driver/Types.def
rg -n 'def Tops|LANGSTANDARD\(tops' clang/include/clang/Driver/Options.td clang/include/clang/Basic/LangStandards.def
rg -n '__global__|__device__|__shared__|__local__|__constant__|__thread_dims__|__valigned__' \
  clang/lib/Headers/tops/__tops_defines.h
rg -n 'threadIdx|blockIdx|blockDim|gridDim|threadDim|subThreadIdx' \
  clang/lib/Headers/tops/__tops_builtins.h
```

预期结果：能找到 `.tops` 输入类型、`-Tops`/`tops` language 入口、Tops 属性宏和内建变量声明。这一步只证明源码事实存在，不证明语言服务器已经支持它们。

## 4. 检查本地验证工具

```bash
test -x /home/carl.du/work/llvm-project/build/bin/clang
test -x /home/carl.du/work/llvm-project/build/bin/llvm-lit
/home/carl.du/work/llvm-project/build/bin/clang --version
```

预期结果：`clang` 和 `llvm-lit` 存在并输出版本。版本、headers 和目标参数必须作为 TargetProfile 证据记录；不能把系统 LLVM 版本替代本地工具。Go server 不依赖 `clangd`。

## 5. P0 工具链闭环验证

使用当前 checkout 的实际 `compile_commands.json` 或对应测试配置，分别验证以下输入形式：

1. `.tops` 文件直接作为输入。
2. `.cpp` 文件配合 `-Tops`。
3. 任意源文件配合 `-x tops`。
4. 目标 profile、Tops include 根、device 编译参数和预定义宏同时存在。

每种形式都记录，作为 Go server 的输入和离线对照基线：

- driver 最终采用的语言模式和 target triple。
- `-fsyntax-only` 的诊断结果。
- Go server LSP `didOpen` 的诊断、补全、悬停和定义跳转结果。
- 本地直接 `clang -fsyntax-only` 的对应诊断，用于离线对照，不作为 Go server 的运行结果。
- 缺少编译数据库、目标参数或头文件时的错误行为。

预期结果不是预先假定全部成功，而是为每种输入形式形成 `verified`、`target-dependent`、`blocked` 或 `unsupported` 证据。Go server 必须拥有最终语义结果；Clang 只提供差分参考。任何差异都进入 Go parser/semantic 的修正任务，不通过扩展 clangd 解决。

## 6. 评审顺序

1. 先检查 [research.md](research.md) 的事实来源和方案决策。
2. 再检查 [data-model.md](data-model.md) 是否能表达能力、目标、上下文、诊断和里程碑状态。
3. 再检查 [contracts/capability-matrix.md](contracts/capability-matrix.md)、[contracts/lsp-boundary.md](contracts/lsp-boundary.md) 和 [contracts/roadmap-gates.md](contracts/roadmap-gates.md) 是否覆盖章程要求。
4. 最后对照 [plan.md](plan.md) 的 Constitution Check、阶段依赖和验证命令。

## 7. 发布最终 roadmap 文档

完成 roadmap 撰写和评审后，在工作区根目录执行：

```bash
test -f doc/tops-cpp-language-server-roadmap.md
rg -n 'Go server|P0|P1|P2|P3|P4|clangd|Tops' doc/tops-cpp-language-server-roadmap.md
git diff --check
```

预期结果：最终文档位于 `doc/tops-cpp-language-server-roadmap.md`，包含 Go 从零开发、不扩展或依赖 clangd、P0-P4 阶段和 Tops 语法范围；其事实和契约引用可回溯到 `specs/001-tops-lsp-roadmap/`。当前计划阶段尚未生成该最终发布文件，因此本节只在 roadmap 撰写完成后执行。

任何无法从当前 checkout 或实际工具输出核验的结论，都必须保留为待 P0 验证项，不得在 roadmap 中写成已支持能力。
