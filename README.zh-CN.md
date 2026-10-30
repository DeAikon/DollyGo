[English](./README.md) | **中文**

# DollyGo —— VasDolly 多渠道打包工具

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go&logoColor=white)
![License](https://img.shields.io/github/license/DeAikon/DollyGo?style=flat)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-333?style=flat)
![Dependencies](https://img.shields.io/badge/Dependencies-stdlib%20only-2ea44f?style=flat)
![Version](https://img.shields.io/badge/Version-1.0.0-blue?style=flat)

DollyGo 是一个单文件 CLI：构建一次 Android release APK，使用 VasDolly 为每个渠道生成 APK，并在报告成功前逐个校验产物。

- **使用语言**：Go 1.24（仅标准库；运行时会使用内置的 `VasDolly.jar`）
- **工具类别**：Android 多渠道打包工具
- **版本**：v1.0.0
- **界面语言**：默认英文；加 `--lzh` 全部切换为中文
- **目标平台**：macOS、Linux（arm64 / amd64）。不支持 Windows，因为进程组终止依赖 Unix 语义
- **对目标项目的写入范围**：DollyGo 不修改目标项目的源码、Gradle 文件或签名配置；但执行 release 构建时，Gradle 仍会在项目内写入 `app/build/`、`.gradle/` 等构建产物与缓存

---

## 工作流程

1. 前置检查：项目目录、可执行的 `gradlew`、`java`、`git` 以及已释放的内置资源。
2. 分支校验：仅在传入 `-b/--branch` 且能读取当前分支时执行。
3. 读取渠道清单（内置清单或 `-c/--channels-file`），并校验渠道名。
4. 通过只读的 Gradle init script 读取 `versionName` / `versionCode`。
5. 展示项目、提交与签名信息，并进行两次确认（`--yes` 跳过）。
6. 执行 `./gradlew --no-daemon clean :app:assembleRelease`。
7. 要求基础包为 V2/V3 签名。
8. 使用 VasDolly 并行生成渠道包。
9. 重命名为 `{prefix}_{渠道}_{versionName 的 . 替换为 _}({versionCode}).apk`，并移动到输出目录。
10. 对每个渠道包做三重校验：内嵌渠道 == 请求渠道 == 从文件名解析出的渠道。

## 前置条件

- Go 1.24+（仅构建时需要）。
- `java`（JDK）与 `git` 在 `PATH` 中。
- 目标项目根目录必须包含可执行的 `gradlew`，并且存在 `app` 模块。
- 目标项目必须具备可用的 release 签名配置，且 `assembleRelease` 能成功执行。
- 基础包预期位于 `app/build/outputs/apk/release/app-release.apk`。
- 生成渠道包要求基础包使用 V2 或 V3 签名；仅 V1 签名的 APK 会被拒绝。

## 安装 / 构建

```bash
git clone https://github.com/DeAikon/DollyGo.git
cd DollyGo
go build -o dist/dollygo .
./dist/dollygo --help
```

交叉编译（静态链接，无 CGO）：

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/dollygo-darwin-arm64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/dollygo-darwin-amd64 .
CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -o dist/dollygo-linux-amd64 .
CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -o dist/dollygo-linux-arm64 .
```

若二进制通过浏览器下载，macOS Gatekeeper 可能拦截首次运行：

```bash
xattr -d com.apple.quarantine dist/dollygo
```

## 致谢

DollyGo 使用腾讯 [VasDolly](https://github.com/Tencent/VasDolly) 完成 APK 渠道信息的写入与读取。感谢 VasDolly 项目及其贡献者的出色工作。

VasDolly 采用 **BSD 3-Clause License** 开源协议；其内置第三方组件可能使用其他协议，详见 [VasDolly LICENSE](https://github.com/Tencent/VasDolly/blob/master/LICENSE)。

## 使用说明

```bash
dollygo -p <项目路径> [选项]
```

| 参数 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `-p, --project <path>` | `--yes` 模式下必填 | — | 项目根路径（含 `gradlew` 的目录）。未传且未加 `--yes` 时会交互提示输入 |
| `-c, --channels-file <path>` | 否 | 内置清单 | 渠道清单文件（纯文本，每行一个渠道，`#` 注释）；传入则整体覆盖内置清单 |
| `-b, --branch <name>` | 否 | 当前分支 | 当前分支不匹配时终止（exit `7`）；无法读取当前分支时跳过校验 |
| `-o, --output-dir <dir>` | 否 | `~/multi_channel_build` | 渠道包输出目录（支持 `~`） |
| `--jobs <n>` | 否 | `min(4, 渠道数)` | 并行路数；必须 `>= 1`，且会被限制为不超过渠道数 |
| `--prefix <name>` | 否 | `app` | 渠道包文件名前缀；不能为空，且不能包含 `/`、`\` 或空白字符 |
| `--yes` | 否 | 关 | 非交互模式，跳过两次确认；缺 `-p` 时直接失败（exit `1`） |
| `--json` | 否 | 关 | 在 **stdout** 输出单行 JSON 摘要（除 `--help` 外的所有退出路径）。默认关闭，此时 stdout 无输出 |
| `--lzh` | 否 | 关 | 以中文显示所有面向用户的文本；JSON 输出始终为英文 |
| `-h, --help` | 否 | — | 向 **stdout** 打印帮助并退出 `0`（跟随当前语言） |

内置默认渠道：`Official`、`Xiaomi`、`Oppo`、`Vivo`、`Huawei`、`Honor`、`Tencent`。

渠道名会经过校验：去除首尾空白后，包含 `,`、`/` 或空白字符的名称会被拒绝；重复名称与空渠道清单同样以 exit `1` 拒绝。

渠道包命名：`{prefix}_{渠道}_{versionName 的 . 替换为 _}({versionCode}).apk`，例：`app_Xiaomi_1_3_2(20260101).apk`。

```bash
# 交互模式（两次确认）
dollygo -p /path/to/android-project

# 非交互完整流程
dollygo -p /path/to/android-project -o ~/multi_channel_build --yes

# 非交互 + 机器可读结果
dollygo -p /path/to/android-project -b release --yes --json

# 中文界面
dollygo -p /path/to/android-project --yes --lzh
```

## 退出码

| 退出码 | 含义 |
|---|---|
| `0` | 成功 |
| `1` | 前置检查失败（缺少 `java` / `git` / 项目目录 / `gradlew` / 渠道文件；渠道名非法；渠道清单为空；`--yes` 模式缺 `-p`） |
| `2` | 用户取消（确认时回答 N；SIGINT / SIGTERM） |
| `3` | 基础包构建失败（`assembleRelease`） |
| `4` | 渠道包生成失败（包括基础包不是 V2/V3 签名，或预期产物缺失） |
| `5` | 渠道校验失败（读回值、预期值、文件名三者不一致） |
| `6` | 内部 / 未知错误 |
| `7` | 分支不匹配 |

## 输出契约

- **stdout**：默认无任何输出；**退出码**即机器信号。
- 传入 `--json` 时，stdout 在除 `--help` 外的所有退出路径打印**单行 JSON**（失败包含 `error`）。
- **stderr**：人类可读的进度、spinner 与交互提示；不要从 stderr 解析结果。
- TTY 下进度使用莫奈风 256 色配色；非 TTY 或设置 `NO_COLOR` 时自动禁用颜色与动画。
- 基础包构建的完整 Gradle 日志写入 `<outputDir>/gradle-build.log`，每次运行覆盖。
- JSON 键名与 `error` 值始终为英文，不受 `--lzh` 影响。

JSON 字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `success` | bool | 始终存在 |
| `versionName` | string | 应用 `versionName` |
| `versionCode` | int | 应用 `versionCode` |
| `branch` | string | 当前 git 分支 |
| `baseApk` | string | 构建出的 release 基础包路径 |
| `outputDir` | string | 渠道包所在目录 |
| `prefix` | string | 本次使用的文件名前缀 |
| `channels` | string[] | 本次请求的渠道 |
| `verified` | string[] | 通过校验的渠道（成功时出现） |
| `jobs` | int | 实际并行路数 |
| `durationMs` | int | 耗时（毫秒，始终存在） |
| `error` | string | 仅在失败时出现（英文） |

除 `success` 与 `durationMs` 外，其余字段为空时会省略。

```bash
dollygo -p /path/to/android-project --yes --json
```

## 限制

- 当前 Gradle 模块与基础包路径固定为 `:app` 和 `app/build/outputs/apk/release/app-release.apk`。
- 工具会先执行 `clean` 再执行 `assembleRelease`，因此目标项目内已有的构建产物会被清除。
- 仅支持 macOS 与 Linux。
- 工具不会创建或修改签名配置；目标项目必须本身就能构建已签名的 release APK。

## 许可证

MIT，详见 [LICENSE](./LICENSE)。
