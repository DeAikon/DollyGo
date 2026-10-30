**English** | [中文](./README.zh-CN.md)

# DollyGo — VasDolly Multi-Channel APK Builder

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go&logoColor=white)
![License](https://img.shields.io/github/license/DeAikon/DollyGo?style=flat)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-333?style=flat)
![Dependencies](https://img.shields.io/badge/Dependencies-stdlib%20only-2ea44f?style=flat)
![Version](https://img.shields.io/badge/Version-1.0.0-blue?style=flat)

DollyGo is a single-binary CLI that builds one Android release APK, generates one APK per channel with VasDolly, and verifies every output before reporting success.

- **Language**: Go 1.24 (standard library only; the bundled `VasDolly.jar` is used at runtime)
- **Category**: Android multi-channel APK tool
- **Version**: v1.0.0
- **UI language**: English by default; `--lzh` switches all user-facing text to Chinese
- **Platforms**: macOS and Linux (arm64 / amd64). Windows is not supported because process-group termination is Unix-specific.
- **Target project safety**: DollyGo never edits the target project's source, Gradle files, or signing configuration. Gradle itself still writes build outputs and caches under the project (`app/build/`, `.gradle/`) because a release build is performed.

---

## How it works

1. Preflight: project directory, executable `gradlew`, `java`, `git`, and embedded assets.
2. Branch check: only when `-b/--branch` is given and the current branch can be read.
3. Load channels from the built-in list or `-c/--channels-file`, then validate names.
4. Read `versionName` / `versionCode` through a read-only Gradle init script.
5. Show project, commit, and signing information; ask for two confirmations (`--yes` skips both).
6. Run `./gradlew --no-daemon clean :app:assembleRelease`.
7. Require the base APK to be V2/V3 signed.
8. Generate channel APKs with VasDolly in parallel batches.
9. Rename each APK to `{prefix}_{channel}_{versionName with . replaced by _}({versionCode}).apk` and move it to the output directory.
10. Verify every APK three ways: embedded channel == requested channel == channel parsed from the filename.

## Requirements

- Go 1.24+ (build only).
- `java` (JDK) and `git` on `PATH`.
- The target project root must contain an executable `gradlew` and an `app` module.
- The target project must have a working release signing configuration; `assembleRelease` must succeed.
- The base APK is expected at `app/build/outputs/apk/release/app-release.apk`.
- Channel generation requires the base APK to be signed with V2 or V3; a V1-only APK is rejected.

## Install / build

```bash
git clone https://github.com/DeAikon/DollyGo.git
cd DollyGo
go build -o dist/dollygo .
./dist/dollygo --help
```

Cross-compile (static, no CGO):

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/dollygo-darwin-arm64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/dollygo-darwin-amd64 .
CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -o dist/dollygo-linux-amd64 .
CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -o dist/dollygo-linux-arm64 .
```

If the binary was downloaded through a browser, macOS Gatekeeper may block the first run:

```bash
xattr -d com.apple.quarantine dist/dollygo
```

## Acknowledgements

DollyGo uses [Tencent VasDolly](https://github.com/Tencent/VasDolly) to write and read APK channel information. Thanks to the VasDolly project and its contributors for their excellent work.

VasDolly is licensed under the **BSD 3-Clause License**; bundled third-party components may use other licenses (see the [VasDolly LICENSE](https://github.com/Tencent/VasDolly/blob/master/LICENSE) for details).

## Usage

```bash
dollygo -p <project-path> [options]
```

| Option | Required | Default | Description |
|---|---|---|---|
| `-p, --project <path>` | Required in `--yes` mode | — | Project root (directory containing `gradlew`). Without `--yes`, you are prompted when omitted |
| `-c, --channels-file <path>` | No | Built-in list | Channel list file (plain text, one channel per line, `#` comments); overrides the built-in list |
| `-b, --branch <name>` | No | Current branch | Abort with exit `7` when the current branch differs. If the current branch cannot be read, the check is skipped |
| `-o, --output-dir <dir>` | No | `~/multi_channel_build` | Output directory for channel APKs (supports `~`) |
| `--jobs <n>` | No | `min(4, channel count)` | Parallelism; must be `>= 1` and is capped at the channel count |
| `--prefix <name>` | No | `app` | APK filename prefix; must not be empty or contain `/`, `\`, or whitespace |
| `--yes` | No | Off | Non-interactive; skips both confirmations. Fails with exit `1` if `--project` is missing |
| `--json` | No | Off | Print a single-line JSON summary on **stdout** (on every exit path except `--help`). Default off: stdout is empty |
| `--lzh` | No | Off | Show all user-facing text in Chinese. JSON output is always English |
| `-h, --help` | No | — | Print help to **stdout** and exit `0` (follows the current language) |

Built-in default channels: `Official`, `Xiaomi`, `Oppo`, `Vivo`, `Huawei`, `Honor`, `Tencent`.

Channel names are validated: after trimming, names containing `,`, `/`, or whitespace are rejected, as are duplicate names and an empty channel list (exit `1`).

Channel APK naming: `{prefix}_{channel}_{versionName with . replaced by _}({versionCode}).apk`, e.g. `app_Xiaomi_1_3_2(20260101).apk`.

```bash
# Interactive run (two confirmations)
dollygo -p /path/to/android-project

# Non-interactive full run
dollygo -p /path/to/android-project -o ~/multi_channel_build --yes

# Non-interactive + machine-readable result
dollygo -p /path/to/android-project -b release --yes --json

# Chinese UI
dollygo -p /path/to/android-project --yes --lzh
```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Preflight failure (missing `java` / `git` / project / `gradlew` / channels file; invalid channel name; empty channel list; `--yes` without `-p`) |
| `2` | Cancelled by the user (answered N; SIGINT / SIGTERM) |
| `3` | Base APK build failed (`assembleRelease`) |
| `4` | Channel generation failed (including a base APK that is not V2/V3 signed, or a missing expected output) |
| `5` | Channel verification failed (read-back value, expected value, and filename do not match) |
| `6` | Internal / unknown error |
| `7` | Branch mismatch |

## Output contract

- **stdout**: empty by default; the **exit code** is the machine signal.
- With `--json`, stdout prints exactly **one line of JSON** on every exit path except `--help` (failures include `error`).
- **stderr**: human-readable progress, spinners, and prompts. Do not scrape stderr for results.
- Progress output uses a Monet-style 256-color scheme on a TTY; color and animation are disabled for non-TTY output or when `NO_COLOR` is set.
- The full Gradle output of the base build is written to `<outputDir>/gradle-build.log` and overwritten on each run.
- JSON keys and `error` values are always English, regardless of `--lzh`.

JSON fields:

| Field | Type | Notes |
|---|---|---|
| `success` | bool | Always present |
| `versionName` | string | App `versionName` |
| `versionCode` | int | App `versionCode` |
| `branch` | string | Current git branch |
| `baseApk` | string | Path to the built release base APK |
| `outputDir` | string | Directory containing the channel APKs |
| `prefix` | string | Filename prefix used |
| `channels` | string[] | Channels requested for this run |
| `verified` | string[] | Channels that passed verification (present on success) |
| `jobs` | int | Parallelism used |
| `durationMs` | int | Elapsed time in milliseconds (always present) |
| `error` | string | Present only on failure (English) |

Fields other than `success` and `durationMs` are omitted when empty.

```bash
dollygo -p /path/to/android-project --yes --json
```

## Limitations

- The Gradle module and APK path are currently fixed to `:app` and `app/build/outputs/apk/release/app-release.apk`.
- The tool runs `clean` before `assembleRelease`, so existing build outputs in the target project are removed.
- Only macOS and Linux are supported.
- The tool does not create or modify signing configuration; the target project must already be able to build a signed release APK.

## License

MIT. See [LICENSE](./LICENSE).
