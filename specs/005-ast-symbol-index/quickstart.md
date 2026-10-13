# 验证指南：AST 和符号索引

**功能**：[spec.md](spec.md)
**数据模型**：[data-model.md](data-model.md)
**LSP 契约**：[contracts/lsp-navigation.md](contracts/lsp-navigation.md)

本指南用于实现完成后的开发机验证。它不要求 GCU 设备、GCUSIM、测试机 Docker、clangd、网络或 Clang runtime。

## 前置条件

- 在 `/home/carl.du/work/tops-lsp` 执行命令。
- Go 版本与 [go.mod](../../go.mod#L1) 中的 `go 1.26.8` 一致。
- 测试材料使用临时 workspace URI、内存 header provider 或项目内 fixture；不修改 `topsop` 和 `llvm-project`。
- `testdata/index/manifest.json` 的 execute 记录包含按顺序对应的 `source_files`/`source_uris`、`expected_symbol_count`、`expected_location_count`、`expected_status`、`lsp_operations`/`queries` 和 `expected_diagnostics`；query 可声明 `include_declaration`、`expected_count`、`expected_null`、malformed params 或 error code；schema-only 记录必须包含 `skip_reason`。验证器输出每类总数、通过数、失败数、skipped 数和失败 ID。

## 1. 基线和定向测试

```sh
go run ./cmd/tops-index-check --manifest testdata/index/manifest.json
go test ./internal/parser ./internal/document ./internal/position ./internal/protocol ./internal/server ./internal/index
go test ./integration
go test ./...
go test -race ./...
go vet ./...
```

预期：

- parser、document、position 和已有 server 测试继续通过；
- index 单元测试覆盖模型、scope、ID、条件和版本失效；
- protocol 测试覆盖导航参数、结果和 capability JSON；
- server 测试覆盖分析队列提交和旧快照丢弃；
- integration 测试通过真实 stdio frame 验证 initialize、导航请求和错误行为。
- manifest validator 输出每个 category 的 total/passed/failed/skipped 和失败 case ID；SC-001 至 SC-003 只接受 execute case 的通过结果，schema-only case 必须单独计为 skipped。

## 2. 单文件导航

测试材料至少包含：

- namespace、class/struct/enum、alias；
- 函数声明、函数定义和调用；
- 标准/Tops attribute；
- builtin 变量和字段访问；
- 一处 unresolved 名称。

检查：

1. `initialize` 声明 `definitionProvider`、`referencesProvider`、`documentSymbolProvider`。
2. definition 在调用处返回函数定义；只有声明时返回声明。
3. references 按 `includeDeclaration` 包含或排除声明。
4. documentSymbol 使用嵌套范围，`selectionRange` 只覆盖名称。
5. unresolved 名称返回空结果，不创建伪造 URI/range。
6. 当前新鲜快照无匹配返回空数组；没有新鲜快照或查询期间失效返回 `null`。
7. alias 在 documentSymbol 中使用标准 `Object` kind；函数结果包含参数/返回类型、执行空间和属性来源摘要。

## 3. header 和多文件

使用内存 source provider 或临时 fixture：

```text
main.tops -> include/api.h
main.tops -> include/types.h
```

`api.h` 定义函数，`types.h` 定义类型，`main.tops` 同时使用二者。检查：

- definition 能跳转到 header URI；
- references 包含跨文件位置并去重；
- 同名不同 namespace 不合并；
- 同一 header 被多个文件引用时复用相同 source version；
- 删除 header 或移除 include root 后，当前文档局部符号仍可查询，并产生 unresolved/degraded 状态；
- `SourceProvider.IncludeRoots()` 的 roots/available 状态能传入 `ParseContext`；无法解析 include spelling 时为 `unresolved`，已解析但找不到目标时为 `missing`；
- header URI/version 变化会刷新直接和间接依赖，不返回旧 URI；
- 不会扫描未打开、未配置或未被引用的 workspace 文件。

## 4. 条件分支和多候选

使用同一源文件分别提供：

- 一个 active 分支；
- 一个 inactive 分支；
- 一个 unknown 分支；
- 两个 active 同名候选。
- 覆盖 `#if/#ifdef/#ifndef/#elif/#else/#endif` 和至少一层嵌套 parent-child condition。

检查：

- 有 active 候选时 definition 返回全部 active 候选；references 返回候选引用并集；不静默选择最近作用域；
- 没有 active 候选时才使用 unknown 候选，并产生 `analysis-degraded` 状态；
- inactive 不进入 definition/references；
- documentSymbol 展示 active 和 unknown，隐藏 inactive；
- 条件状态仍保留在 index snapshot 中。
- 完整模板实例化、重载决议和完整宏展开材料只验证状态可见或 out-of-scope，不得标记为完整索引。

## 5. 不完整输入和版本竞争

对同一 URI 依次提交递增版本：

1. 完整声明和函数定义；
2. 删除函数右括号；
3. 删除函数体；
4. 删除一处声明；
5. 恢复完整文本。

并在分析任务执行期间提交后续版本。检查：

- 每个版本仍返回当前文本中已确认的局部 symbols；
- incomplete/partial 不会清空前面已完成的独立声明；
- 删除的声明不从旧快照泄漏；
- 晚到的旧任务不覆盖最新 index、diagnostics 或导航结果；
- 关闭文档后，迟到结果被丢弃。

## 6. LSP 错误路径

发送以下请求：

- 缺少 `textDocument` 或 `position`；
- URI 未打开；
- position 超出当前文本；
- 对正在分析的文档取消请求；
- 在请求处理期间发送 `didChange`。

预期：参数问题返回标准 `InvalidParams`；取消返回现有 `RequestCancelled`；当前快照无匹配返回空数组；过期/无新鲜快照返回 `null`且不使用旧快照；`analysis-degraded` 使用稳定 diagnostic code，stale 只记录 `stale_result` 事件；server 不崩溃。

标准 LSP JSON decoder proxy 必须能解析三项请求和结果；本验证不要求启动 VS Code 或新增 TypeScript 语义实现。

## 7. 代码质量检查

```sh
git diff --check
git status --short
```

检查变更范围只包含 `tops-lsp` 代码、测试材料和本功能文档；不应出现完整用户源代码、凭据、外部 checkout 修改或额外依赖。
