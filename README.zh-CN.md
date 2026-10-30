[English](./README.md) | **中文**

# channelbuild — VasDolly 多渠道打包工具

- **使用语言**：Go（编译为单文件可执行，零第三方依赖，仅标准库）
- **工具类别**：Android 多渠道打包工具（Gradle 打 release 基础包 → VasDolly 生成多渠道包 → 逐包校验渠道）
- **版本**：v1.0.0
- **界面语言**：默认英文；加 `--lzh` 全部切换为中文
- **目标平台**：macOS arm64 / amd64、Linux amd64 / arm64（交叉编译，静态链接，无 CGO）
- **只读**目标项目：不修改任何源码 / gradle / 文件。

---

## 使用说明

### 安装 / 运行

```bash
chmod +x dist/channelbuild-darwin-amd64   # 或 -arm64；Linux 选 linux-*
./dist/channelbuild-darwin-amd64 --help
```

首次从浏览器下载可能被 macOS Gatekeeper 拦截，解除方式：

```bash
xattr -d com.apple.quarantine dist/channelbuild-darwin-amd64
```

前置条件：`java`、`git` 在 `PATH` 中；目标项目根目录含可执行的 `gradlew`。

### 参数

```bash
channelbuild -p <项目路径> [选项]
```

| 参数 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `-p, --project <path>` | ✅ | — | 项目根路径（含 `gradlew` 的目录） |
| `-c, --channels-file <path>` | 否 | 内置默认 | 渠道清单文件（纯文本，每行一个渠道，`#` 注释）；传入则**整体覆盖**内置清单 |
| `-b, --branch <name>` | 否 | 当前分支 | 传入则校验当前分支是否匹配；不匹配 → 终止（exit 7） |
| `-o, --output-dir <dir>` | 否 | `~/multi_channel_build` | 渠道包输出目录（支持 `~`） |
| `--jobs <n>` | 否 | `min(4, 渠道数)` | 并行路数 |
| `--prefix <name>` | 否 | `app` | 渠道包文件名前缀 |
| `--yes` | 否 | 关 | 非交互模式，跳过全部确认（缺 `--project` 时直接失败，不挂起） |
| `--json` | 否 | 关 | 在 **stdout** 输出单行 JSON 结果摘要（所有退出路径均输出）。默认关闭——不传时 **stdout 无输出** |
| `--lzh` | 否 | 关 | 以中文显示所有面向用户的文本（help / 提示 / 错误 / 汇总）；默认英文。JSON 输出始终为英文 |
| `-h, --help` | 否 | — | 显示帮助并退出（跟随当前语言：默认英文，`--lzh` 中文） |

渠道包命名：`{prefix}_{渠道}_{versionName 的 . 替换为 _}({versionCode}).apk`，例：`app_Xiaomi_1_3_2(20260101).apk`。

```bash
# 非交互完整流程
channelbuild -p /path/to/android-project -o ~/multi_channel_build --yes
```

### 退出码

`0` 成功 · `1` 前置检查失败 · `2` 用户取消 · `3` 基础包构建失败 · `4` 渠道包生成失败 · `5` 渠道校验失败 · `6` 内部错误 · `7` 分支不匹配。

### 输出

默认 **stdout 无任何输出**——**退出码**即机器信号。

传入 `--json` 时，**stdout** 在**所有**退出路径打印**单行 JSON** 摘要（失败含 `error`）。进度与交互提示始终打印到 **stderr**（TTY 下带莫奈配色，非 TTY / `NO_COLOR` 自动禁用）。

```bash
channelbuild -p /path/to/android-project --yes --json
```
