# Quickstart：验证 P0 Go 服务器语义基线计划

本指南验证规格、计划、证据入口和契约文档，不启动 Go server，不构建 TypeScript client，不运行 clangd，不在测试机运行 workload。

## 前置条件

- 规格工作区：`/home/carl.du/work/tops-lsp`
- LLVM/Tops 参考 checkout：`/home/carl.du/work/llvm-project`
- 用户编译器优先使用 PATH 中的 `topscc`；语言对照工具使用 `/home/carl.du/work/llvm-project/build/bin/clang` 和 `/home/carl.du/work/llvm-project/build/bin/llvm-lit`；实际路径和版本必须在执行时确认。
- 开发机可用 `rg`、`jq`、`git`。
- 本 feature 不需要 Go module、npm package、硬件、gcusim 或测试机 Docker 容器。

## 0. 核对 topscc driver

```bash
cd /home/carl.du/work/tops-lsp
command -v topscc
readlink -f "$(command -v topscc)"
topscc --version
topscc --dryrun -arch gcu400 -x tops -fsyntax-only /dev/null
```

预期：能确认 wrapper 路径和版本，并在 dry-run 中看到 `-std=c++11`、`-Tops`、`--include tops.h`、Tops include 根、`--cuda-gpu-arch=gcu400` 以及 host/device 命令。该步骤只记录参数事实，不在 LSP runtime 中启动 topscc。

## 1. 检查 P0 文档产物

```bash
cd /home/carl.du/work/tops-lsp

test -f specs/002-p0-go-semantic-baseline/spec.md
test -f specs/002-p0-go-semantic-baseline/plan.md
test -f specs/002-p0-go-semantic-baseline/research.md
test -f specs/002-p0-go-semantic-baseline/data-model.md
test -f specs/002-p0-go-semantic-baseline/quickstart.md
test -f specs/002-p0-go-semantic-baseline/contracts/capability-matrix.md
test -f specs/002-p0-go-semantic-baseline/contracts/lsp-boundary.md
test -f specs/002-p0-go-semantic-baseline/contracts/roadmap-gates.md
test -f specs/002-p0-go-semantic-baseline/checklists/requirements.md
jq -e '.feature_directory == "specs/002-p0-go-semantic-baseline"' .specify/feature.json >/dev/null
```

预期：所有文件和 feature pointer 存在；没有生成源码目录。

## 2. 检查文档完整性

```bash
cd /home/carl.du/work/tops-lsp

! rg -n '\[FEATURE NAME\]|\[DATE\]|\[未使用时删除\]|\[例如|\[NEEDS CLARIFICATION' \
  specs/002-p0-go-semantic-baseline/spec.md \
  specs/002-p0-go-semantic-baseline/plan.md \
  specs/002-p0-go-semantic-baseline/research.md \
  specs/002-p0-go-semantic-baseline/data-model.md \
  specs/002-p0-go-semantic-baseline/contracts

test "$(rg -c '^\| `TOPS-[A-Z]+-[0-9]+`' specs/002-p0-go-semantic-baseline/spec.md)" -ge 8
test "$(rg -c '^\| `[^`]+-default`' specs/002-p0-go-semantic-baseline/spec.md)" -eq 6
test "$(rg -c '^\- \*\*P1-[A-Z]+-(V|I|P|T)' specs/002-p0-go-semantic-baseline/spec.md)" -eq 32
! rg -n '^\- \[ \]' specs/002-p0-go-semantic-baseline/checklists/requirements.md
git diff --check
```

预期：八个能力域、六个 candidate profile、32 个 P1 场景均存在；清单全勾选；没有模板占位符和空白字符错误。

## 3. 检查当前 LLVM 证据入口

```bash
cd /home/carl.du/work/llvm-project

rg -n 'TYPE\("tops|TYPE\("tops-cpp-output' clang/include/clang/Driver/Types.def
rg -n 'def Tops|tops_sp|tops_simt' clang/include/clang/Driver/Options.td
rg -n 'LANGSTANDARD\(tops' clang/include/clang/Basic/LangStandards.def
rg -n 'GCU300|GCU400|GCU410|GCU450|GCU500|EFGCU500' \
  clang/include/clang/Basic/Cuda.h clang/lib/Basic/Cuda.cpp
rg -n 'ArchVersion = 300|ArchVersion = 400|ArchVersion = 410|ArchVersion = 450|ArchVersion = 500' \
  clang/lib/Basic/Targets/DTUTargetInfo.h
rg -n '__GCU_ARCH__|__AGCU_ARCH__' clang/lib/Basic/Targets/DTU.cpp
rg -n '__EFGCU_ARCH__' clang/lib/Basic/Targets/EFGCU.cpp
rg -n 'TOPSLaunchBounds|TOPSMaxOACC' clang/include/clang/Basic/Attr.td
rg -n '__global__|__device__|__shared__|__local__|__valigned__' \
  clang/lib/Headers/tops/__tops_defines.h
rg -n 'threadIdx|blockIdx|blockDim|gridDim|threadDim|subThreadIdx' \
  clang/lib/Headers/tops/__tops_builtins.h
rg -n 'threadIdx|blockIdx|blockDim|gridDim|warpSize' \
  clang/lib/Headers/tops/__tops_efgcu_builtin_vars.h
```

预期：可以定位 driver、TargetInfo、属性、header 和 builtin 入口。这一步只证明源码事实存在，不证明 Go server 已实现；用户编译器入口以 topscc 核对结果为准。

## 4. 核对候选 profile

在开发机使用实际本地 compiler，逐个 profile 保存命令和输出。命令形式必须以 `clang -###` 的实际展开为准；以下是验证族，不是对所有环境的固定命令承诺：

```bash
cd /home/carl.du/work/llvm-project

build/bin/clang -### -Tops --cuda-gpu-arch=gcu300 --cuda-device-only -x c++ /path/to/minimal.tops
build/bin/clang -### -Tops --cuda-gpu-arch=gcu400 --cuda-device-only -x c++ /path/to/minimal.tops
build/bin/clang -### -Tops --cuda-gpu-arch=gcu410 --cuda-device-only -x c++ /path/to/minimal.tops
build/bin/clang -### -Tops --cuda-gpu-arch=gcu450 --cuda-device-only -x c++ /path/to/minimal.tops
build/bin/clang -### -Tops --cuda-gpu-arch=gcu500 --cuda-device-only -x c++ /path/to/minimal.tops
build/bin/clang -### -Tops --cuda-gpu-arch=efgcu500 --cuda-device-only -x c++ /path/to/minimal.tops
```

对每个 profile 记录：

- 实际 target triple、CPU/offload arch、host/device pass。
- `__GCU_ARCH__` 或 `__EFGCU_ARCH__` 的 `-dM -E` 输出。
- Tops/Clang include 根和顺序。
- `-fsyntax-only` 结果及适用 Clang/CodeGen test。
- Go server 后续应使用的 `CompilationContext` 快照。

EFGCU500 必须单独核对 `__EFGCU_ARCH__=500`，不能用 `__GCU_ARCH__=500` 替代。没有预期宏或 include 根时，profile 保持 `candidate`/`invalid`，不使用隐藏默认。

## 5. 检查现有测试证据

```bash
cd /home/carl.du/work/llvm-project

sed -n '1,80p' tops/integration_test/cases/language/STATUS.md
find tops/integration_test/cases/language -path '*exec_spec*' -o \
  -path '*launch_config*' -o -path '*builtin_vars*' -o \
  -path '*vector_types*' -o -path '*memory_space*'
find clang/test/CodeGenEFGCU -maxdepth 2 -type f | rg 'launch|cluster|vector_types|qualifier|maxnreg'
find clang/test/DTU_test/tcle -type f | rg 'parser|Draco|gcu450|vector_op|maskbit|countl_zero'
```

现有 `STATUS.md` 的回归数字和 BUG-2/4/5/6 只作为状态证据；重新运行时必须分别报告 compile、host/device runtime 和 device exception，不得将默认 regression PASS 写成全量 device pass。

## 6. 评审层次与 P1 入口

按以下顺序评审：

1. [spec.md](spec.md)：确认八域、六 profile、context、层次边界和 32 场景。
2. [research.md](research.md)：确认事实、决策、限制和未决项。
3. [data-model.md](data-model.md)：确认实体、字段、状态和失效关系。
4. [contracts/capability-matrix.md](contracts/capability-matrix.md)：确认能力条目证据门禁。
5. [contracts/lsp-boundary.md](contracts/lsp-boundary.md)：确认 Go/TypeScript/Clang owner 和 LSP 错误路径。
6. [contracts/roadmap-gates.md](contracts/roadmap-gates.md)：确认 P0/P1 entry、exit、fallback 和 owner。
7. [plan.md](plan.md)：确认宪法检查、依赖、风险和阶段顺序。

完成上述检查后，才可运行 `/speckit.tasks`。P1 任务必须为 32 个场景建立 Go 自有测试材料/contract test；Clang 结果只作为差分依据。

## 7. Done 条件

- P0 文档和契约文件均存在且没有模板占位符。
- 八个能力域、六个 profile、32 个 P1 场景的计数和字段门禁通过。
- `CompilationContext` 优先级和 stale 规则可被测试描述。
- Go server、TypeScript client 和 Clang 对照验证 的责任不重叠。
- `llvm-project` 没有被本 feature 修改。
- 不需要构建、部署或运行新的 Go/TypeScript 代码。
