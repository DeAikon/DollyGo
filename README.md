**English** | [中文](./README.zh-CN.md)

# channelbuild — VasDolly Multi-Channel APK Builder

- **Language**: Go (compiled to a single executable; no third-party dependencies, stdlib only)
- **Category**: Android multi-channel APK tool (Gradle builds a release base APK → VasDolly generates per-channel APKs → each channel is verified)
- **Version**: v1.0.0
- **UI language**: English by default; add `--lzh` to switch everything to Chinese
- **Targets**: macOS arm64 / amd64, Linux amd64 / arm64 (cross-compiled, statically linked, no CGO)
- **Read-only** on the target project: it never modifies any source / gradle / file.

---

## Usage

### Install / run

```bash
chmod +x dist/channelbuild-darwin-amd64   # or -arm64; on Linux pick linux-*
./dist/channelbuild-darwin-amd64 --help
```

If downloaded via a browser, macOS Gatekeeper may block the first run; clear it with:

```bash
xattr -d com.apple.quarantine dist/channelbuild-darwin-amd64
```

Requirements: `java` and `git` on `PATH`; the target project root must contain an executable `gradlew`.

### Options

```bash
channelbuild -p <project-path> [options]
```

| Option | Required | Default | Description |
|---|---|---|---|
| `-p, --project <path>` | ✅ | — | Project root (directory containing `gradlew`) |
| `-c, --channels-file <path>` | no | built-in | Channel list file (plain text, one channel per line, `#` comments); overrides the built-in list |
| `-b, --branch <name>` | no | current branch | If given, verify the current branch matches; mismatch → abort (exit 7) |
| `-o, --output-dir <dir>` | no | `~/multi_channel_build` | Output directory for channel APKs (supports `~`) |
| `--jobs <n>` | no | `min(4, channel count)` | Parallelism |
| `--prefix <name>` | no | `app` | APK filename prefix |
| `--yes` | no | off | Non-interactive; skip all confirmations (fails if `--project` is missing) |
| `--json` | no | off | Print the result as a single-line JSON summary on **stdout** (on every exit path). Default off — **stdout is empty** unless passed |
| `--lzh` | no | off | Show all user-facing text in Chinese (help / prompts / errors / summary); default is English. JSON output is always English |
| `-h, --help` | no | — | Show help and exit (follows the current language) |

Channel APK naming: `{prefix}_{channel}_{versionName with . replaced by _}({versionCode}).apk`, e.g. `app_Xiaomi_1_3_2(20260101).apk`.

```bash
# Non-interactive full run
channelbuild -p /path/to/android-project -o ~/multi_channel_build --yes
```

### Exit codes

`0` success · `1` preflight failure · `2` cancelled by user · `3` base build failed · `4` channel generation failed · `5` channel verification failed · `6` internal error · `7` branch mismatch.

### Output

By default **nothing is written to stdout** — the **exit code** is the machine signal.

With `--json`, **stdout** prints a **single-line JSON** summary on **every** exit path (failures include `error`). Progress and interactive prompts always go to **stderr** (Monet color scheme on a TTY; auto-disabled for non-TTY / `NO_COLOR`).

```bash
channelbuild -p /path/to/android-project --yes --json
```
