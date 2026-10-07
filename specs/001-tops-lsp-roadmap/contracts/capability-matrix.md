# 契约：Tops C++ 语法能力矩阵

## 目的

统一 roadmap 对 Tops C++ 语法、标准 C++ 能力和目标条件的记录方式。矩阵是事实与计划之间的边界，不是语言服务器运行时 API。

## 条目格式

每个条目必须包含以下字段：

| 字段 | 说明 |
| --- | --- |
| `id` | 来自 `SyntaxCapability.id` 的稳定标识符 |
| `category` | 八个能力域之一：标准 C++、执行空间、存储空间、启动/资源、向量/数值、内建变量、TCLE/API、目标条件 |
| `spelling` | 用户源码中出现的关键字、属性、类型、变量或 API |
| `semantic` | 用户可观察的语义和限制 |
| `targets` | 适用架构、`-Tops`/`-x tops` 模式、宏或工具链条件 |
| `source_evidence` | 当前活动 header、Clang attribute/TargetInfo、driver、测试或状态文件 |
| `test_evidence` | 正向、负向、目标边界和不完整输入的测试入口 |
| `status` | `candidate`、`verified`、`target-dependent`、`supported`、`unsupported`、`blocked`、`deprecated-source` |
| `phase` | `P0` 至 `P4` |
| `lsp_surface` | 诊断、补全、悬停、定义、引用、语义 token 或仅文档 |
| `limitations` | 已知失败、部分覆盖、未默认运行或待验证项 |
| `verification` | 可复现的最小验证动作 |

## 状态门禁

- 没有源码/头文件或 Clang 定义证据的条目不能超过 `candidate`。
- 只有源码证据但没有适用测试或编译验证的条目不能标记 `supported`。
- 目标条件不同的条目必须拆成独立 profile 记录，或在 `targets` 中逐项说明。
- `blocked` 和 `unsupported` 必须出现在用户可见 roadmap 中，并给出恢复条件或拒绝理由。
- 旧文档中的条目必须标记 `deprecated-source`，除非当前 header、attribute 和测试重新核对通过。

## 证据优先级

1. 活动目标条件、Clang TargetInfo 和 attribute/Sema 实现。
2. `clang/lib/Headers/tops` 和 `tcle.h`。
3. Clang/DTU/EFGCU lit 测试及 `tops/integration_test/cases/language`。
4. 测试状态记录。
5. 维护文档和已标注不再维护的历史文档。

## 更新规则

新增或修改条目时，必须同时更新 `status`、`phase`、`verification`、`limitations` 和相关测试证据。不得只增加关键字列表而不更新目标前提和失败路径。
