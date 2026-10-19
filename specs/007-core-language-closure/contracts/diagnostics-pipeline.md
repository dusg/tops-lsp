# 契约：诊断合并、去重、排序与清理

这份文档固定诊断从各来源到最终发布的行为。

## 1. 来源

| 来源 | 产生位置 | 编号示例 |
|------|----------|----------|
| 语法 | `internal/parser` | 语法类编号 |
| 索引降级 | `internal/index` | `analysis-degraded` |
| 标准 C++ 语义 | `internal/semantic` 检查器 | `tops-semantic-unknown-name`、`tops-semantic-type-mismatch` |
| Tops 扩展语义 | `internal/semantic` 检查器 | `tops-semantic-host-device-boundary`、`tops-semantic-address-space`、`tops-semantic-vector-constraint` |
| 上下文状态 | `internal/semantic` 检查器 | `tops-semantic-context-degraded`、`tops-semantic-context-missing` |

上下文状态提示归入 Tops 扩展来源，也是 `tops-semantic-*` 编号，但语义上是“提醒用户当前只能做部分检查”，不是确定错误。

## 2. 编号与级别

| 编号 | 级别 | 说明 |
|------|------|------|
| `tops-semantic-unknown-name` | Error | 名称找不到声明 |
| `tops-semantic-ambiguous-name` | Warning | 有不只一个候选 |
| `tops-semantic-type-mismatch` | Error | 类型不匹配 |
| `tops-semantic-invalid-call` | Error | 调用不成立 |
| `tops-semantic-host-device-boundary` | Error | host/device 调用边界违规 |
| `tops-semantic-address-space` | Error | 内存空间使用违规 |
| `tops-semantic-vector-constraint` | Error | 向量约束违规 |
| `tops-semantic-context-missing` | Warning | 没有目标上下文 |
| `tops-semantic-context-degraded` | Warning | 上下文不完整 |
| `tops-semantic-incomplete-input` | Hint | 输入还没写完 |
| `analysis-degraded` | Information | 索引或依赖降级 |

**变更说明**：`tops-semantic-context-missing` 与 `tops-semantic-context-degraded` 在本功能中由 Information 调整为 Warning。这是对外可见变化，测试数据与本文档同步更新。

## 3. 合并

顺序如下：

1. 收集索引诊断，按 `SourceURI` 分组；缺失时归到当前文档。
2. 把语法诊断追加到当前文档。
3. 追加语义诊断（含标准 C++、Tops 扩展、上下文状态）到当前文档。
4. 对每个 URI 分别执行去重与排序。

## 4. 去重

去重键：`code | range.start | range.end | severity | message`。

- 键相同只保留第一条。
- 键不同但编号与位置相同（消息不同）的保留全部，因为可能代表不同原因。
- 去重对所有来源一视同仁，不按来源优先级覆盖。

## 5. 排序

排序键（全部升序）：`range.start`、`range.end`、`severity`、`code`、`message`。

- `severity` 用标准数值：1 Error、2 Warning、3 Information、4 Hint。
- 排序结果对相同输入必须完全一致；不依赖 map 遍历顺序。

## 6. 发布

| 项 | 规则 |
|----|------|
| 通知方法 | `textDocument/publishDiagnostics` |
| `uri` | 诊断所属文档 |
| `version` | 该文档的诊断版本；没有时按当前文档版本 |
| `contextVersion` | 本次分析的上下文版本 |
| `diagnostics` | 去重排序后的数组；无问题时为空数组 |

发布顺序按 URI 字符串升序，保证多次运行一致。

## 7. 清理

| 触发 | 行为 |
|------|------|
| 新版本分析完成且无问题 | 为该 URI 发送空数组，版本为当前文档版本 |
| 文档关闭 | 为该 URI 发送空数组 |
| 上下文整体失效 | 为所有受影响的 URI 发送空数组 |
| 分析被丢弃（过期） | 不发送任何通知 |

清理通知同样携带 `version` 和 `contextVersion`，便于客户端判断新旧。

**注意**：清理通知与真实分析结果可能具有相同的 `uri` 和 `version`（诊断为空数组）。客户端和测试不能只凭版本号判断“分析完成”，必须同时看 `diagnostics` 内容；这是已有测试踩过的问题，本功能保持该行为并在此明确记录。

## 8. 版本与过期

- 只有文档版本、上下文版本、分析代号、来源指纹全部匹配时才发布。
- 旧分析结果只能在过期日志里出现，不得进入发布负载。
- 一次发布不混合两个版本的诊断。

## 9. 可观测性

| 事件 | 记录字段 |
|------|----------|
| 分析完成并发布 | 文档标识、文档版本、上下文版本、诊断数量、编号列表 |
| 发布失败 | 文档标识、错误类别 |
| 过期丢弃 | 文档标识、来源、原因、耗时 |
| 清理 | 文档标识、版本、原因 |

禁止记录完整源码、凭据、用户主目录路径。
