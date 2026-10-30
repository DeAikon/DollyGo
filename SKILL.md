---
name: channelbuild
description: Build Android multi-channel APKs with VasDolly (Gradle release base APK + per-channel APKs + verification). Use when asked to build/release channel APKs, produce 渠道包, run channelbuild, or ship an Android release to multiple app stores (Xiaomi/Oppo/Vivo/Huawei/Honor/Tencent, etc.).
---

# channelbuild — Android multi-channel APK builder

Drives the `channelbuild` CLI: it builds one Gradle **release** base APK, generates one
channel APK per channel with VasDolly, and verifies every output. It is **read-only** on the
target project (it never edits source, Gradle, or any file in it).

The tool is shell-only — no MCP, no server. Pass `--json` to get a parseable JSON
result on stdout, then parse **stdout JSON + exit code**. By default stdout is empty and
the **exit code** is the only machine signal.

## When to use / when not to

**Use when:**
- You need to build channel APKs for an Android project, or a full signed release.
- You want per-channel APKs plus a verification result in one command.

**Do not use when:**
- The project has no release signing config, or `assembleRelease` is not expected to work.
- You only need a single APK with no channels — use `./gradlew :app:assembleRelease` directly.
- `java` or `git` is not on `PATH` (see Prerequisites).

## Prerequisites

- `java` on `PATH` (runs `gradlew` and VasDolly; JDK 17 works).
- `git` on `PATH`.
- Target project root: the directory that contains an **executable** `gradlew`.
- The `channelbuild` binary, executable (`chmod +x`).
- macOS: if the binary was downloaded via a browser, Gatekeeper may block the first run —
  clear it with `xattr -d com.apple.quarantine <binary>`.

## Invocation

```bash
channelbuild -p <project-path> [options] --yes --json
```

Agents should always pass `--yes` for a non-interactive run. With `--yes`, `-p/--project`
is mandatory — omitting it fails immediately with exit `1` (it will not hang on a prompt).

Pass `--json` to receive a machine-readable result on stdout; without it stdout is empty
and only the exit code is emitted.

## Options

| Option | Required | Default | Description |
|---|---|---|---|
| `-p, --project <path>` | ✅ | — | Project root (directory containing `gradlew`) |
| `-c, --channels-file <path>` | no | built-in list | Channel list file (plain text, one channel per line, `#` comments); **overrides** the built-in list |
| `-b, --branch <name>` | no | current branch | If given, abort when the current branch differs (exit `7`) |
| `-o, --output-dir <dir>` | no | `~/multi_channel_build` | Output directory for channel APKs (supports `~`) |
| `--jobs <n>` | no | `min(4, channel count)` | Parallelism (auto-degrades when fewer channels) |
| `--prefix <name>` | no | `app` | APK filename prefix |
| `--yes` | no | off | Non-interactive; skip all confirmations (fails if `--project` is missing) |
| `--json` | no | off | Print the result as a single-line JSON summary on **stdout** (on every exit path). **Default off — stdout is empty**; pass it to get parseable JSON |
| `--lzh` | no | off | Show all user-facing text in Chinese (default: English) |
| `-h, --help` | no | — | Show help and exit |

Built-in default channels: `Official`, `Xiaomi`, `Oppo`, `Vivo`, `Huawei`, `Honor`, `Tencent`.

Output APK naming: `{prefix}_{channel}_{versionName with . replaced by _}({versionCode}).apk`
— e.g. `app_Xiaomi_1_3_2(20260101).apk`.

## Output contract

- **stdout**: **empty by default.** Pass `--json` to print exactly **one line of JSON** on
  **every** exit path (success and failure). The exit code is always the machine signal.
- **stderr**: human-readable progress, spinners, and prompts. **Never scrape stderr** for
  results.
- **JSON keys and `error` values are always English**, regardless of `--lzh`.

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
| `durationMs` | int | Elapsed time in ms (always present) |
| `error` | string | Present only on failure (English) |

Fields other than `success` and `durationMs` are omitted when empty.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Preflight failure (missing java/git/project/channels file; invalid channel name; empty channels file; `--yes` without `-p`) |
| `2` | Cancelled by user (answered N; SIGINT/SIGTERM) |
| `3` | Base APK build failed (`assembleRelease`) |
| `4` | Channel generation failed (incl. base APK not V2/V3 signed, or expected output missing) |
| `5` | Channel verification failed (read-back ≠ expected ≠ filename) |
| `6` | Internal/unknown error |
| `7` | Branch mismatch |

## Examples

**Success (exit 0)** — all channels built and verified:

```bash
channelbuild -p /path/to/android-project -b release --yes --json
# exit 0
```

```json
{"success":true,"versionName":"1.3.2","versionCode":20260101,"branch":"release","baseApk":"/path/to/android-project/app/build/outputs/apk/release/app-release.apk","outputDir":"/Users/me/multi_channel_build","prefix":"app","channels":["Official","Xiaomi","Oppo","Vivo","Huawei","Honor","Tencent"],"verified":["Official","Xiaomi","Oppo","Vivo","Huawei","Honor","Tencent"],"jobs":4,"durationMs":654321}
```

**Branch mismatch (exit 7)** — no build happens:

```bash
channelbuild -p /path/to/android-project -b release --yes --json   # current branch is "main"
# exit 7
```

```json
{"success":false,"branch":"main","durationMs":1234,"error":"branch mismatch: expected \"release\", current \"main\""}
```

**Verification failure (exit 5)** — a channel APK did not read back as expected:

```json
{"success":false,"versionName":"1.3.2","versionCode":20260101,"branch":"release","baseApk":"/path/to/android-project/app/build/outputs/apk/release/app-release.apk","outputDir":"/Users/me/multi_channel_build","prefix":"app","channels":["Official","Xiaomi"],"jobs":2,"durationMs":234567,"error":"verify: file=/Users/me/multi_channel_build/app_Xiaomi_1_3_2(20260101).apk expected=Xiaomi got= filename=Xiaomi"}
```

## Guardrails

- **Read-only** on the target project — the tool never modifies its source or Gradle files.
- Requires **JDK** and **git**; do not run without them.
- **Parse the exit code.** Pass `--json` and parse stdout JSON for details; without `--json`
  stdout is empty. Do not scrape stderr for results or progress.
- On failure, read **`<outputDir>/gradle-build.log`** for the full Gradle output of the base
  build (overwritten each run).
- Default UI language is English; `--lzh` switches stderr/help to Chinese. **JSON is
  unaffected and always English.**
- Channel names are validated: after trimming, names containing `,`, `/`, spaces, empty
  values, or duplicates are rejected (exit `1`).

## Install

Ship `SKILL.md` alongside the binary. To make it discoverable by an agent, copy the folder
to either location:

```bash
# user-level
cp -r <channel_build-dir> ~/.config/opencode/skills/channelbuild/

# project-level
cp -r <channel_build-dir> <project>/.opencode/skills/channelbuild/
```
