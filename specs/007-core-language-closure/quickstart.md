# 快速验证指南

这份文档说明怎么在本地验证“基础语言功能”是否真的可用：不需要 GCU 硬件，不需要 Clang 运行时，也不需要测试机。

## 1. 前置条件

| 项 | 要求 |
|----|------|
| Go | `1.26.8`（与 `go.mod` 一致） |
| 工作目录 | 仓库根目录 |
| 网络 | 不需要 |
| 硬件 | 不需要 |

## 2. 构建与基础检查

```bash
cd /home/carl.du/work/tops-lsp
go build ./...
go vet ./...
```

预期：两条命令都没有输出错误。

## 3. 全量测试

```bash
cd /home/carl.du/work/tops-lsp
go test -p 128 -count=1 ./...
```

预期：全部通过。

## 4. 端到端验收

三个验收文件在 `testdata/lsp/fixtures/`：

| 文件 | 代表场景 |
|------|----------|
| `valid-kernel.tops` | 能正常分析的 Tops kernel |
| `std-cpp-error.tops` | 标准 C++ 写错 |
| `tops-semantic-error.tops` | Tops 扩展规则写错 |

运行：

```bash
cd /home/carl.du/work/tops-lsp
go test -p 128 -count=1 ./integration -run TestLSPProcessCoreLanguage -v
```

预期输出中应能看到六个子测试分别通过：

| 子测试 | 验证内容 |
|--------|----------|
| `TestLSPProcessCoreLanguageValidKernel` | 正常 kernel 的六种请求、改版后更新、2 秒内出新诊断 |
| `TestLSPProcessCoreLanguageStandardCppErrors` | 标准 C++ 错误的编号与级别、修复后清空 |
| `TestLSPProcessCoreLanguageStaleAndCloseCleanup` | 旧版本诊断丢弃、关闭文档后清空 |
| `TestLSPProcessCoreLanguageTopsSemanticErrors` | Tops 规则错误与上下文降级 |
| `TestLSPProcessCoreLanguageRequestContract` | 参数错误、空结果、取消、未实现方法、100 次版本竞争与取消 |
| `TestLSPProcessCoreLanguageAnalyzingState` | 分析进行中时悬停与补全的提示、导航的 2 秒上限 |

## 5. 手工验证（用编辑器）

如果你想把服务器接到编辑器里看效果：

1. 构建服务器：

   ```bash
   cd /home/carl.du/work/tops-lsp
   go build -o build/tops-lsp ./cmd/tops-lsp
   ```

2. 在编辑器里把 `.tops` 文件关联到 `tops` 语言，并把语言服务器命令指向 `build/tops-lsp`（stdio 模式）。

3. 打开 `testdata/lsp/fixtures/valid-kernel.tops`，确认：
   - 问题面板没有意料之外的 Error。
   - 在函数体内输入名字前缀，出现候选。
   - 鼠标停在已声明的名字上，出现说明框。
   - `F12` 能跳到声明，`Shift+F12` 能列出引用。
   - 侧边栏的文档符号里有函数和变量。

4. 打开 `testdata/lsp/fixtures/std-cpp-error.tops`，确认问题面板出现语法或语义 Error；把错误改对后，提示消失。

5. 打开 `testdata/lsp/fixtures/tops-semantic-error.tops`，确认出现 `tops-semantic-` 开头的 Error，鼠标悬停能看到 profile 和 pass。

## 6. 逐项核对

| 检查项 | 怎么确认 |
|--------|----------|
| 诊断合并 | `std-cpp-error.tops` 中同一位置只出现一条相同问题 |
| 诊断排序 | 同一文件的问题按从上到下的位置顺序排列 |
| 诊断级别 | 语法与语义错误是 Error；上下文不完整与上下文缺失是 Warning |
| 诊断清理 | 改对代码后，问题面板清空（先看到旧提示，再看到清空才有效） |
| 旧诊断丢弃 | 连续快速修改时，不出现旧版本的提示 |
| 关闭清理 | 关闭文档后，问题面板清空 |
| 补全正常 | `valid-kernel.tops` 中能补出已声明的名字 |
| 补全不猜 | 光标在注释或字符串里时，候选为空 |
| 补全排除项 | 点号后、自动导入、代码片段均不出现候选，也不报错 |
| 补全分析中 | 刚打开文件立刻请求，只出现一条“正在分析中”提示项，没有真实候选 |
| 悬停正常 | 名字、种类、类型或签名、来源都有 |
| 悬停分析中 | 刚打开文件立刻请求，出现“正在分析中”提示；文件为空时也给提示，不返回空结果 |
| 导航超时 | 分析被拖住时，跳定义最多等 2 秒，然后收到 `ContentModified` |
| 取消 | 请求中途取消返回 `RequestCancelled` |
| 版本竞争 | 重复“改版本 + 发请求” 100 次，不出现旧版本结果 |
| 能力声明 | 初始化结果里只有约定字段，没有拉取诊断、工作区符号和补全二次解析 |
| 文字语言 | 诊断、悬停、补全、错误信息均为英文 |

详细规则见：

- [contracts/lsp-core-language.md](contracts/lsp-core-language.md)
- [contracts/diagnostics-pipeline.md](contracts/diagnostics-pipeline.md)
- [contracts/completion-candidates.md](contracts/completion-candidates.md)

## 7. 已知不覆盖

| 项 | 说明 |
|----|------|
| 性能百分位 | 本版只验证 500 行以内、2 秒内出现诊断 |
| 跨版本兼容矩阵 | 留到后续性能与兼容阶段 |
| 成员访问补全 | 本版明确不做 |
| 拉取式诊断 | 本版明确不做 |
| 无硬件环境下的设备行为 | 本功能不涉及设备执行 |

## 8. 出问题时怎么看

| 现象 | 先查什么 |
|------|----------|
| 没有诊断 | 文档版本与上下文版本是否匹配；日志里是否有过期丢弃记录 |
| 补全一直为空 | 光标是否在注释、字符串或预处理指令里；是否有新鲜结果 |
| 悬停一直是“正在分析中” | 分析是否卡住；日志里是否有发布失败或 panic 记录 |
| 导航一直超时 | 分析耗时是否超过 2 秒；文件是否过大 |
| 提示重复出现 | 去重键是否覆盖了该组合 |
| 改对后提示不消失 | 是否只收到清理通知而被误当成分析结果；两者版本号可能相同 |
| 补全候选种类不对 | 结构体、枚举的分类映射是否按 `TypeKind` 细分 |

日志只包含文档标识、版本、编号和原因分类，不包含源码内容。

## 9. 验证记录

**日期**：2026-10-19  
**环境**：Linux 开发机，Go `1.26.8`，未安装 GCU 驱动，未设置 `TOPS_LSP_CLANG`（集成测试不调用 Clang，也不需要 GCU 硬件）。

| 命令 | 结果 |
|------|------|
| `go build ./...` | 通过，无输出 |
| `go vet ./...` | 通过，无输出 |
| `go test -p 128 -count=1 ./...` | 全部包通过 |
| `go test -count=1 ./integration -run TestLSPProcessCoreLanguage -v` | 六个子测试全部通过 |

六个验收子测试的实际耗时（同一台开发机）：

| 子测试 | 耗时 |
|--------|------|
| `TestLSPProcessCoreLanguageValidKernel` | 0.18s |
| `TestLSPProcessCoreLanguageStandardCppErrors` | 0.18s |
| `TestLSPProcessCoreLanguageStaleAndCloseCleanup` | 0.18s |
| `TestLSPProcessCoreLanguageTopsSemanticErrors` | 0.34s |
| `TestLSPProcessCoreLanguageRequestContract` | 2.46s（含 2 秒导航超时断言、100 次版本竞争与 100 次取消循环） |
| `TestLSPProcessCoreLanguageAnalyzingState` | 0.39s |

运行前清空了 `TOPS_LSP_CLANG` 与 `TOPS_LSP_INCLUDE_ROOTS`，结果与设置前一致，确认环境独立。

