// Package cli 负责命令行参数解析与交互提示（含橙色加粗警告）。
package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/DeAikon/DollyGo/internal/i18n"
)

// Version 是工具版本号（help 头部与 README 体现，见 R21）。
const Version = "1.0.0"

// Options 是解析后的命令行参数。
type Options struct {
	Project      string // -p/--project      项目根路径（必填）
	ChannelsFile string // -c/--channels-file 渠道清单文件（覆盖内置）
	Branch       string // -b/--branch        期望分支
	OutputDir    string // -o/--output-dir    输出目录
	Jobs         int    // --jobs             并行路数（0 表示自动）
	Prefix       string // --prefix           文件名前缀（默认 app）
	Yes          bool   // --yes              非交互
	JSON         bool   // --json             在 stdout 输出单行 JSON 摘要（默认关闭）
	LZH          bool   // --lzh              以中文显示所有面向用户的文本（默认英文）
	Help         bool   // -h/--help          显示当前语言的帮助
}

// Parse 手写解析参数，保持零第三方依赖。支持 `-p v`、`--project v`、`--project=v` 形式。
func Parse(args []string) (*Options, error) {
	o := &Options{Prefix: "app"}

	next := func(flag string, i *int) (string, error) {
		*i++
		if *i >= len(args) {
			return "", fmt.Errorf("flag %s requires a value", flag)
		}
		return args[*i], nil
	}

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "" {
			continue
		}

		var name, inline string
		hasInline := false
		switch {
		case strings.HasPrefix(a, "--"):
			name = a[2:]
		case strings.HasPrefix(a, "-") && len(a) > 1:
			name = a[1:]
		default:
			return nil, fmt.Errorf("unexpected argument: %q", a)
		}
		if k := strings.IndexByte(name, '='); k >= 0 {
			inline, name = name[k+1:], name[:k]
			hasInline = true
		}

		get := func() (string, error) {
			if hasInline {
				return inline, nil
			}
			return next("-"+name, &i)
		}

		switch name {
		case "p", "project":
			v, err := get()
			if err != nil {
				return nil, err
			}
			o.Project = v
		case "c", "channels-file":
			v, err := get()
			if err != nil {
				return nil, err
			}
			o.ChannelsFile = v
		case "b", "branch":
			v, err := get()
			if err != nil {
				return nil, err
			}
			o.Branch = v
		case "o", "output-dir":
			v, err := get()
			if err != nil {
				return nil, err
			}
			o.OutputDir = v
		case "jobs":
			v, err := get()
			if err != nil {
				return nil, err
			}
			n, convErr := strconv.Atoi(strings.TrimSpace(v))
			if convErr != nil || n < 1 {
				return nil, fmt.Errorf("invalid --jobs value: %q", v)
			}
			o.Jobs = n
		case "prefix":
			v, err := get()
			if err != nil {
				return nil, err
			}
			o.Prefix = v
		case "yes":
			o.Yes = true
		case "json":
			o.JSON = true
		case "lzh":
			o.LZH = true
		case "h", "help":
			o.Help = true
		default:
			return nil, fmt.Errorf("unknown flag: %s", a)
		}
	}

	return o, nil
}

const helpTextEN = `DollyGo v` + Version + ` - VasDolly multi-channel APK build tool

Usage:
  dollygo -p <project-path> [options]

Options:
  -p, --project <path>        Project root (directory containing gradlew) [required]
  -c, --channels-file <path>  Channel list file (plain text, one channel per line,
                              # for comments); overrides the built-in default list
  -b, --branch <name>         Expected branch; abort if the current branch differs (exit 7)
  -o, --output-dir <dir>      Output directory (default ~/multi_channel_build)
      --jobs <n>              Parallelism (default min(4, channel count); auto-degrades)
      --prefix <name>         APK filename prefix (default app)
      --yes                   Non-interactive mode; skip all confirmations
                              (fails if --project is missing)
      --json                  Print the result as a single-line JSON summary on
                              stdout (default: off; stdout is empty by default)
      --lzh                   Show all user-facing text in Chinese (default: English)
  -h, --help                  Show this help and exit

APK naming:
  {prefix}_{channel}_{versionName with . replaced by _}({versionCode}).apk
  e.g. app_Xiaomi_1_3_2(20260101).apk

Exit codes:
  0  success                    1  preflight failure
  2  cancelled by user          3  base APK build failed
  4  channel generation failed  5  channel verification failed
  6  internal/unknown error     7  branch mismatch

Output:
  By default nothing is written to stdout; pass --json to print a single-line
  JSON summary on stdout (on every exit path). The exit code is always the
  machine signal. Progress and interactive prompts are printed to stderr.
`

const helpTextZH = `DollyGo v` + Version + ` - VasDolly 多渠道打包工具

用法:
  dollygo -p <项目路径> [选项]

参数:
  -p, --project <path>        项目根路径（含 gradlew 的目录）【必填】
  -c, --channels-file <path>  渠道清单文件（纯文本，每行一个渠道，# 为注释）；
                              传入则整体覆盖内置默认清单
  -b, --branch <name>         期望分支；与当前分支不符则终止（exit 7）
  -o, --output-dir <dir>      渠道包输出目录（默认 ~/multi_channel_build）
      --jobs <n>              并行路数（默认 min(4, 渠道数)；渠道数更少时自动降级）
      --prefix <name>         渠道包文件名前缀（默认 app）
      --yes                   非交互模式，跳过全部确认（缺 --project 时直接失败）
      --json                  在 stdout 输出单行 JSON 结果摘要（默认关闭；默认 stdout 无输出）
      --lzh                   以中文显示所有面向用户的文本（默认英文）
  -h, --help                  显示本帮助并退出

渠道包命名:
  {prefix}_{渠道}_{versionName 的 . 替换为 _}({versionCode}).apk
  例: app_Xiaomi_1_3_2(20260101).apk

退出码:
  0  全部成功            1  前置检查失败        2  用户取消
  3  基础包构建失败      4  渠道包生成失败      5  渠道校验失败
  6  内部/未知错误       7  分支不匹配

输出:
  默认 stdout 无任何输出；传入 --json 时在 stdout 打印单行 JSON 摘要（所有退出路径均输出）。
  退出码始终是机器信号；进度与交互提示打印到 stderr。
`

// PrintHelp 输出**当前语言**的帮助文本（-h/--help；默认英文，--lzh 中文，AC25/AC27）。
func PrintHelp(w io.Writer) {
	if i18n.Current() == i18n.ZH {
		fmt.Fprint(w, helpTextZH)
		return
	}
	fmt.Fprint(w, helpTextEN)
}
