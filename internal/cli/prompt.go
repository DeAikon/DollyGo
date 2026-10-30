package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"channel_build/internal/i18n"
)

// SigningView 是展示用的签名信息视图（**不含密码**，从设计上保证屏蔽）。
type SigningView struct {
	Found          bool
	KeystorePath   string
	KeyAlias       string
	KeystoreExists bool
	Warn           string
}

// DisplayInfo 汇总确认页需要展示的全部信息。
type DisplayInfo struct {
	ProjectRoot    string
	ChannelsSource string
	ChannelsCount  int
	Channels       []string
	ExpectedBranch string
	CurrentBranch  string
	OutputDir      string
	Jobs           int
	Prefix         string
	VersionName    string
	VersionCode    int
	Commits        []string
	Signing        SigningView
}

// Display 打印完整信息（参数 + 采集信息）。未指定 -b 时，分支名以「警告」类别
// （琥珀 214 + 加粗）显示（AC4 / R16）；其余按信息归属着色（莫奈配色，design §19）。
// 文案取自 i18n 当前语言（默认英文，--lzh 中文）。
func Display(w io.Writer, d DisplayInfo) {
	m := i18n.M
	fmt.Fprintln(w)
	fmt.Fprintln(w, Paint(CatTitle, m.DisplayTitle))
	fmt.Fprintln(w, Paint(CatTitle, m.SecProject))
	fmt.Fprintln(w, Paint(CatParam, m.LblProjectPath+d.ProjectRoot))
	fmt.Fprintln(w, Paint(CatParam, m.LblChannelsSource+d.ChannelsSource))
	fmt.Fprintln(w, Paint(CatParam, fmt.Sprintf("%s%d", m.LblChannelsCount, d.ChannelsCount)))
	fmt.Fprintln(w, Paint(CatParam, m.LblChannels+strings.Join(d.Channels, ", ")))
	if d.ExpectedBranch != "" {
		fmt.Fprintln(w, Paint(CatParam, m.LblExpectedBranch+d.ExpectedBranch))
	} else {
		fmt.Fprintln(w, Paint(CatWarn, fmt.Sprintf(m.BranchWarnFmt, d.CurrentBranch)))
	}
	fmt.Fprintln(w, Paint(CatParam, m.LblOutputDir+d.OutputDir))
	fmt.Fprintln(w, Paint(CatParam, fmt.Sprintf("%s%d", m.LblJobs, d.Jobs)))
	fmt.Fprintln(w, Paint(CatParam, m.LblPrefix+d.Prefix))
	fmt.Fprintln(w)

	fmt.Fprintln(w, Paint(CatTitle, m.SecSource))
	fmt.Fprintln(w, Paint(CatHighlight, m.LblVersionName+d.VersionName))
	fmt.Fprintln(w, Paint(CatHighlight, fmt.Sprintf("%s%d", m.LblVersionCode, d.VersionCode)))
	fmt.Fprintln(w, Paint(CatHighlight, m.LblCurrentBranch+d.CurrentBranch))
	fmt.Fprintln(w, Paint(CatInfo, m.LblRecentCommits))
	if len(d.Commits) == 0 {
		fmt.Fprintln(w, Paint(CatMuted, m.NoCommits))
	}
	for _, c := range d.Commits {
		fmt.Fprintln(w, Paint(CatInfo, "    "+c))
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, Paint(CatTitle, m.SecSigning))
	if d.Signing.Found {
		fmt.Fprintln(w, Paint(CatInfo, m.LblKeystore+d.Signing.KeystorePath))
		fmt.Fprintln(w, Paint(CatInfo, m.LblKeyAlias+d.Signing.KeyAlias))
		fmt.Fprintln(w, Paint(CatInfo, m.LblStorePassword+"***"))
		fmt.Fprintln(w, Paint(CatInfo, m.LblKeyPassword+"***"))
		if !d.Signing.KeystoreExists {
			fmt.Fprintln(w, Paint(CatWarn, m.WarnKeystoreMissing+d.Signing.KeystorePath))
		}
	} else {
		fmt.Fprintln(w, Paint(CatMuted, m.ParseFailedSkipped+d.Signing.Warn))
	}
	fmt.Fprintln(w, Paint(CatTitle, "==================================================="))
}

// PromptProject 交互要求输入项目路径（R3.1）。
func PromptProject(r *bufio.Reader, w io.Writer) (string, error) {
	fmt.Fprint(w, i18n.M.PromptProject)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Confirm 请求 y/N 确认；空输入/其它输入视为否。
func Confirm(r *bufio.Reader, w io.Writer, question string) (bool, error) {
	fmt.Fprintf(w, "%s [y/N]: ", question)
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
