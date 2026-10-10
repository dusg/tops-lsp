# 契约：topscc/Clang 编译参数解析

## 目的

本契约定义用户编译命令如何转换为 `CompilerInvocation` 和 `CompilationContext`。`topscc` 是用户侧首选编译器；直接 `clang`/`clang++` 是兼容入口；Clang 离线对照只用于验证，不是 server runtime 后端。

## Driver 类型

| `driver_kind` | 识别方式 | 处理规则 |
| --- | --- | --- |
| `topscc` | argv[0] basename 为 `topscc`，或用户显式配置为 topscc | 先解析 wrapper 参数，再解析转发给底层 Clang 的参数；记录 wrapper 版本和默认值来源 |
| `clang`/`clang++` | argv[0] 为直接 Clang driver | 按 Clang 参数规则解析，不套用 topscc 默认值 |
| `unknown` | 无法识别或 wrapper 路径不可确认 | 保留 raw argv，进入 `partial` 或 `invalid`，不猜测 Tops target |

## 输入与参数边界

1. `compile_commands.json` 优先读取 `arguments` 数组；只有没有 `arguments` 时才解析 `command` 字符串。
2. 保留 argv 顺序、空参数、工作目录和 response file 路径；response file 以命令工作目录为基准展开。
3. 不使用字符串拼接重新构造参数；归一化结果必须能回指原始参数和来源。
4. LSP runtime 只做静态解析，不启动 `topscc`。`topscc --dryrun` 只能在开发机离线核对时使用。

## topscc 参数类别

| 类别 | 已观察参数 | 解析结果 |
| --- | --- | --- |
| 目标与模式 | `-arch <name>`、`--simd`、`--simt`、`--simt32`、`--simt128` | target candidates、pass、SIMD/SIMT 宏和冲突状态 |
| 设备输出 | `-dbin`、`-dbc`、`-dllvm`、`-dlink-path <path>`、`-dlink <file>`、`-rdc` | device mode、link metadata；不把输出格式当作 parser 语义 |
| host/device | `-host-only`、`-fsycl` | pass kind 或特殊 driver mode；组合不确定时标记 `target-dependent` |
| 库和链接 | `-notopslib`、`-topsrt <mode>`、`-ltops`、`-kdd` | 保留为 non-semantic arguments；需要影响宏/头文件时另行记录 |
| wrapper 控制 | `--dryrun`、`-o <file>`、`-x <lang>`、`-Xclang <arg>` | `--dryrun` 只用于离线检查；其余按参数语义和转发关系记录 |

已安装 wrapper 在未显式指定时注入 `-std=c++11`、`-Tops`、`--include tops.h`、`-fno-jump-tables` 和 Tops include 根；这些值必须标记 `driver_default`，并绑定 wrapper 版本。直接 Clang 不继承这些默认值。

## Clang 参数类别

`-Tops`、`-x tops`、`-std=*`、`-I`、`-isystem`、`-include`、`-D`、`-U`、`--target`、`-mcpu`、`--cuda-gpu-arch`、`--offload-arch`、device-only/pass 参数和 `-Xclang` 中会影响语言分析的参数属于 semantic arguments。`-o`、库、链接和设备打包参数只保留审计信息，不得影响 parser 的语法所有权。

`--driver-mode` 是底层 Clang 参数，不是 topscc 专属参数；LSP 应保留并记录其转发关系，不应自行扩展 driver mode 语义。

## 目标与默认值

- 单个 `-arch` 生成一个 target candidate，并保留原始 spelling 和 wrapper 展开后的参数。
- `agcu*` 或组合架构可能生成多个 target candidate；没有明确的分析目标选择策略时，`target_selection=multi`，不能静默使用第一个目标。
- wrapper 默认架构、标准和 include 必须与 `resolved_executable`、wrapper version 绑定。无法确认 wrapper 版本时，使用 `partial`，不把默认值写成全局事实。
- `gcu300` 的底层 compiler 选择属于 wrapper 行为；LSP 记录 compiler provenance，不把它改写为直接 Clang 的用户命令。

## 错误状态

| 状态/code | 条件 | 行为 |
| --- | --- | --- |
| `missing` / `missing-compiler-context` | 没有 compiler argv、路径或工作目录 | 保留文档 parser 的受限结果，提示配置入口 |
| `invalid` / `invalid-topscc-arguments` | wrapper 参数值缺失、冲突或明确被 wrapper 拒绝 | 不生成完整 TargetProfile，记录原始参数和位置 |
| `partial` / `ambiguous-target-context` | 多架构、未知参数或版本信息不足 | 保留候选目标和来源，禁止静默选择 |
| `partial` / `unresolved-response-file` | response file 不存在或无法解析 | 不猜测其宏、include 或 target 内容 |

## 验证方式

在开发机使用已安装的 `topscc --version` 和 `topscc --dryrun` 核对 wrapper 事实；使用直接 Clang 的 `-###`、`-dM -E`、`-fsyntax-only` 做语言结果对照。验证记录必须保存 driver kind、wrapper/Clang 版本、raw/normalized argv、target/pass、include 和宏来源。验证失败时保留 `partial`/`invalid`，不得把对照结果当成 Go server 结果。