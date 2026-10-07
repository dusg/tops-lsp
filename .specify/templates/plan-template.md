# 实施计划：[FEATURE]

**分支**：`[###-feature-name]` | **日期**：[DATE] | **规格说明**：[link]

**输入**：来自 `/specs/[###-feature-name]/spec.md` 的功能规格说明

**说明**：本模板由 `/speckit.plan` 命令填写；模板定义了计划执行流程。

## 摘要

[从功能规格说明中提取：主要需求和研究得出的技术方案]

## 技术上下文

<!--
  必须执行：将本节内容替换为本项目的技术细节。
  这里的结构用于指导迭代，仅作为建议。
-->

**语言/版本**：[例如 Python 3.11、Swift 5.9、Rust 1.75 或 NEEDS CLARIFICATION]

**主要依赖**：[例如 FastAPI、UIKit、LLVM 或 NEEDS CLARIFICATION]

**存储**：[如适用，例如 PostgreSQL、CoreData、文件或 N/A]

**测试**：[例如 pytest、XCTest、cargo test 或 NEEDS CLARIFICATION]

**目标平台**：[例如 Linux server、iOS 15+、WASM 或 NEEDS CLARIFICATION]

**项目类型**：[例如 library/cli/web-service/mobile-app/compiler/desktop-app 或 NEEDS CLARIFICATION]

**性能目标**：[领域相关，例如 1000 req/s、10k lines/sec、60 fps 或 NEEDS CLARIFICATION]

**约束**：[领域相关，例如 <200ms p95、<100MB memory、支持离线或 NEEDS CLARIFICATION]

**规模/范围**：[领域相关，例如 10k users、1M LOC、50 screens 或 NEEDS CLARIFICATION]

## 宪法检查

*门禁：必须在阶段 0 研究前通过，并在阶段 1 设计后重新检查。*

- [ ] 已明确 LSP、服务器和扩展的职责，以及所有变更的跨边界契约。
- [ ] Tops C/C++ 的语法、语义和目标假设明确且可追溯。
- [ ] 自动化测试覆盖变更行为，包括相关失败路径和边界路径。
- [ ] 已说明错误行为、诊断、日志和外部数据流。
- [ ] 对于公开变更，已记录向后兼容性以及迁移或拒绝行为。
- [ ] 任何违规都在复杂度记录中说明了原因、更简单的替代方案及后续行动。

## 项目结构

### 文档（本功能）

```text
specs/[###-feature]/
├── plan.md              # 本文件（/speckit.plan 命令输出）
├── research.md          # 阶段 0 输出（/speckit.plan 命令）
├── data-model.md        # 阶段 1 输出（/speckit.plan 命令）
├── quickstart.md        # 阶段 1 输出（/speckit.plan 命令）
├── contracts/           # 阶段 1 输出（/speckit.plan 命令）
└── tasks.md             # 阶段 2 输出（/speckit.tasks 命令，不由 /speckit.plan 创建）
```

### 源代码（仓库根目录）
<!--
  必须执行：将下面的占位目录树替换为本功能的具体布局。
  删除未使用的选项，并使用真实路径扩展选定的结构（例如 apps/admin、packages/something）。
  交付的计划不得包含选项标签。
-->

```text
# [未使用时删除] 选项 1：单项目（默认）
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [未使用时删除] 选项 2：Web 应用（检测到 "frontend" + "backend" 时）
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [未使用时删除] 选项 3：移动端 + API（检测到 "iOS/Android" 时）
api/
└── [same as backend above]

ios/ or android/
└── [平台相关结构：功能模块、UI 流程、平台测试]
```

**结构决策**：[记录选定的结构，并引用上面列出的真实目录]

## 复杂度记录

> **仅在宪法检查存在必须说明理由的违规项时填写**

| 违规项 | 必要原因 | 拒绝更简单替代方案的原因 |
|-----------|------------|-------------------------------------|
| [例如第 4 个项目] | [当前需求] | [为什么 3 个项目不足] |
| [例如 Repository pattern] | [具体问题] | [为什么直接访问数据库不足] |
