# 契约：Clang 离线对照

## 目的

Clang 离线对照只用于核对 parser 测试材料的接受/拒绝分类、预处理分支、主要 AST 结构和诊断位置。它不属于 Go server 运行时，不作为 parser fallback，也不把 Clang message 转发给用户。

## 测试材料清单

每个需要对照的条目至少包含：

| 字段 | 说明 |
| --- | --- |
| `fixture` | 仓库内测试材料或只读 `cc_kernel` 路径 |
| `language_standard` | `c++11`、`c++14` 或 `c++17` |
| `arguments` | source、language、target/pass、include、macro 的完整参数来源 |
| `expected_kind` | positive、negative、incomplete、macro-boundary |
| `expected_diagnostics` | Go parser code、主 range、severity 和 recoverability |
| `compare` | preprocess、syntax、ast、diagnostics 的子集 |
| `context_status` | resolved、partial、context-invalid |
| `known_difference` | 已确认的工具链/目标差异；没有差异时为空 |

manifest 不保存凭据、完整命令中的敏感值或完整用户源代码副本。

## 验证阶段

### 1. Driver/context

使用当前 checkout 的本地 Clang/Clang-cc1 入口展开参数，优先记录：

```text
clang -### ...
clang -E -dD ...
clang -dM -E ...
```

若测试材料的 target、pass、include 或宏无法复现，标记 `context-invalid`，不能与默认 Clang 命令比较。

已核对的仓库测试命令形态包括：

- `clang_cc1 -triple gcu -target-cpu gcu400/gcu450 -O0 -fsyntax-only`，用于 GCU vector parser 负向诊断；
- `clang_cc1 -x tops -triple efgcu-enflame-tops -target-cpu efgcu500 -fcuda-is-device -emit-llvm`，用于 EFGCU launch attribute pipeline；
- `clang -nogpulib -std=c++17 --cuda-gpu-arch=gcu400/gcu450 --cuda-device-only -O3 -S -emit-llvm`，用于 GCU400/GCU450 Tops attribute 测试材料。

具体参数以测试材料的实际编译上下文为准，不从相邻 GCU 目标推断。

### 2. Preprocess

使用 `-E -dD` 或等价输出检查 directive、宏定义和区域；使用 `-dM -E` 检查宏集合。比较：

- `#if/#elif/#else/#endif` 的嵌套关系；
- 已知宏的 active/inactive 分支；
- 未知宏是否应进入 Go 的 unknown 状态；
- 源位置是否仍来自原文。

不要求 Go parser 复制 Clang 的完整宏展开文本。

### 3. Syntax

使用 `-fsyntax-only` 比较：

- positive 测试材料是否接受；
- negative 测试材料是否拒绝；
- incomplete 测试材料是否具有可解释的 EOF/恢复分类；
- 主要错误的 file/line/column 是否映射到相同原文 token。

Clang message 文案不是契约。Go code、severity、recoverability 和 LSP range 以本项目契约为准。

### 4. AST

适用时使用 `-Xclang -ast-dump=json`，比较声明、类型、函数、模板、属性和 launch 片段的节点类别、名称、嵌套关系和原文范围。对隐式节点、工具链内部节点和宏展开偏移做归一化后再比较。

### 5. Diagnostic record

对每个差异保存：

```text
`fixture`（测试材料）
source hash
language_standard
macro/target/pass context
Go status + diagnostics(code/range)
Clang status + diagnostics(kind/range)
difference category
known difference / next action
```

source hash 只用于关联输入，不替代测试材料内容，也不写入 server 日志。

## 真实 cc_kernel corpus

优先使用以下只读样例或其缩减片段：

- `topsop/topsop/lib/kernel/cc_kernel/range/range_kernel.tops`
- `topsop/topsop/lib/kernel/cc_kernel/range/range_host.tops`
- `topsop/topsop/lib/kernel/cc_kernel/mhc_pre/mhc_pre_n512_kernel_gcu400.tops`
- `topsop/topsop/lib/kernel/cc_kernel/topp_renorm_probs/topp_renorm_probs_kernel_gcu400.tops`

parser-only 检查可以在缺 header/context 时运行；Clang syntax-only 只有在完整参数可复现时执行。任何 device workload、编译部署或硬件结果都不属于本离线对照的通过条件。

## 执行隔离

- Go unit tests 不依赖 Clang/clangd。
- 离线对照检查可以由 Go test、shell task 或独立工具实现，但必须在 tasks 中固定入口和输出目录。
- server 运行时不启动 Clang/clangd，不读取离线对照结果，不改变 LSP diagnostic 所有权。