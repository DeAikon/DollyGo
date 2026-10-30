// Package i18n 提供全局双语（英文 / 中文）消息层。
//
// 约定（见 design.md §24 / prd.md R22）：
//   - 默认英文（EN）；`--lzh` 切换为中文（ZH）。
//   - 所有面向用户的文本（help / 启动提示 / 展示块 / 进度步骤 / 错误 / 汇总）
//     都从 Messages 取；`Set` 在参数解析后、任何输出之前调用。
//   - **JSON 输出保持英文**（键名与 error 值属稳定机器契约），不经过本包。
//   - 零第三方依赖。
package i18n

// Lang 是界面语言。
type Lang int

const (
	// EN 英文（默认）。
	EN Lang = iota
	// ZH 中文（--lzh）。
	ZH
)

// Messages 覆盖所有面向用户的字符串。
//
// 技术标识符（如 gradle 字段名 keystore / keyAlias / storePassword / keyPassword）
// 两种语言取值相同，但仍显式列出，以保证覆盖完整、双语目录同步。
type Messages struct {
	// --- 参数 / 前置错误（stderr 文案；JSON 的 error 值保持英文） ---
	ArgErrorPrefix string // 参数错误前缀
	ArgErrorHint   string // 用法提示

	// --- 项目路径交互 ---
	PromptProject   string
	ProjectSetRerun string
	ErrYesNoProject string
	ErrNoProject    string
	PreflightFailed string // 前置检查失败前缀

	// --- 启动期步骤 ---
	StepCheckingEnv string
	EnvCheckFailed  string
	EnvCheckPassed  string
	StepReadingVer  string
	VerReadFailed   string
	VerReadDone     string

	// --- 分支 ---
	BranchMismatch     string
	BranchMismatchLine string // fmt: 期望, 当前

	// --- 渠道清单 / 前缀 ---
	ChannelsFailed           string
	ChannelsFailedLine       string
	ChannelsSourceBuiltin    string // 内置默认清单（来源描述）
	ChannelsSourceBuiltinFmt string // fmt: count
	PrefixInvalid            string
	PrefixInvalidLine        string
	ReadVerFailedLine        string

	// --- 确认 ---
	ConfirmInfo  string
	ConfirmStart string
	Cancelled    string

	// --- 输出目录 ---
	MkdirOutputFailed string

	// --- 构建 ---
	StepBuilding        string
	CreateLogFailed     string
	CreateLogFailedLine string
	BuildFailedFmt      string // fmt: 日志路径
	BuildFailedLine     string
	BuildLogLine        string
	BuildDoneFmt        string // fmt: 日志路径

	// --- 签名 ---
	StepCheckingSig string
	SigAbnormalFmt  string // fmt: 模式
	SigAbnormalLine string // fmt: 模式
	SigModePrefix   string

	// --- 渠道生成 ---
	StepGeneratingFmt string // fmt: jobs, batches
	GenProgressFmt    string // fmt: done, total
	BatchPutFmt       string // fmt: batch, count, channels
	BatchDoneFmt      string // fmt: batch, count, ms
	GenFailed         string
	GenFailedLine     string
	GenDoneFmt        string // fmt: count
	MoveFailedLine    string

	// --- 校验 ---
	StepVerifying     string
	VerifyFailFmt     string // fmt: expected, got
	VerifyFailLineFmt string // fmt: file, expected, got, filename
	VerifyPassed      string

	// --- 汇总 ---
	SummaryTitle    string
	SummaryVersion  string // fmt: versionName, versionCode
	SummaryBranch   string // fmt: branch
	SummaryChannels string // fmt: count
	SummaryOutput   string // fmt: dir
	SummaryDuration string // fmt: duration

	// --- 展示块 ---
	DisplayTitle        string
	SecProject          string
	SecSource           string
	SecSigning          string
	LblProjectPath      string
	LblChannelsSource   string
	LblChannelsCount    string
	LblChannels         string
	LblExpectedBranch   string
	BranchWarnFmt       string // fmt: 当前分支
	LblOutputDir        string
	LblJobs             string
	LblPrefix           string
	LblVersionName      string
	LblVersionCode      string
	LblCurrentBranch    string
	LblRecentCommits    string
	NoCommits           string
	LblKeystore         string
	LblKeyAlias         string
	LblStorePassword    string
	LblKeyPassword      string
	WarnKeystoreMissing string
	ParseFailedSkipped  string
}

var en = &Messages{
	ArgErrorPrefix: "argument error: ",
	ArgErrorHint:   "Run --help for usage.",

	PromptProject:   "Enter project root (directory containing gradlew): ",
	ProjectSetRerun: "Project path set; showing full info again.",
	ErrYesNoProject: "Error: --project is required in --yes (non-interactive) mode.",
	ErrNoProject:    "Error: no project path provided.",
	PreflightFailed: "Preflight failed: ",

	StepCheckingEnv: "Checking runtime environment (java / git / gradlew)…",
	EnvCheckFailed:  "Runtime environment check failed",
	EnvCheckPassed:  "Runtime environment check passed",
	StepReadingVer:  "Reading version and source info (Gradle configuring, ~30–60s, please wait)…",
	VerReadFailed:   "Failed to read version info",
	VerReadDone:     "Version / branch / commits read",

	BranchMismatch:     "Branch mismatch",
	BranchMismatchLine: "Branch mismatch: expected %s, current %s\n",

	ChannelsFailed:           "Invalid channel list",
	ChannelsFailedLine:       "Invalid channel list: ",
	ChannelsSourceBuiltin:    "built-in default",
	ChannelsSourceBuiltinFmt: "built-in default (%d)",
	PrefixInvalid:            "Invalid prefix",
	PrefixInvalidLine:        "Prefix error: ",
	ReadVerFailedLine:        "Failed to read version info: ",

	ConfirmInfo:  "Is the information above correct? Continue?",
	ConfirmStart: "Start the build?",
	Cancelled:    "Cancelled.",

	MkdirOutputFailed: "Failed to create output directory: ",

	StepBuilding:        "🔨 Cleaning and building base APK (clean + assembleRelease)…",
	CreateLogFailed:     "Failed to create Gradle log file",
	CreateLogFailedLine: "Failed to create Gradle log file: ",
	BuildFailedFmt:      "Base APK build failed (log: %s)",
	BuildFailedLine:     "Base APK build failed: ",
	BuildLogLine:        "Gradle log: ",
	BuildDoneFmt:        "Base APK built (log: %s)",

	StepCheckingSig: "🔍 Checking base APK signature scheme…",
	SigAbnormalFmt:  "Signature scheme abnormal: %q (want V2/V3)",
	SigAbnormalLine: "Base APK signature scheme abnormal: %q (want V2/V3)\n",
	SigModePrefix:   "Signature scheme: ",

	StepGeneratingFmt: "📦 Generating channel APKs (%d parallel · %d batches)…",
	GenProgressFmt:    "📦 Generating channel APKs … %d/%d batches done",
	BatchPutFmt:       "[batch %d] put %d channels: %s\n",
	BatchDoneFmt:      "[batch %d] done %d channels, %dms\n",
	GenFailed:         "Channel APK generation failed",
	GenFailedLine:     "Channel APK generation failed: ",
	GenDoneFmt:        "Channel APKs generated (%d)",
	MoveFailedLine:    "Failed to move channel APK: ",

	StepVerifying:     "🔎 Verifying channels per APK…",
	VerifyFailFmt:     "Verification failed: expected=%s got=%s",
	VerifyFailLineFmt: "Verification failed: file=%s expected=%s got=%s filename=%s\n",
	VerifyPassed:      "Verification passed",

	SummaryTitle:    "🎉 Build complete",
	SummaryVersion:  "   Version     : %s (%d)",
	SummaryBranch:   "   Branch      : %s",
	SummaryChannels: "   Channels    : %d",
	SummaryOutput:   "   Output dir  : %s",
	SummaryDuration: "   Duration    : %s",

	DisplayTitle:        "==================== Multi-Channel Build ====================",
	SecProject:          "[Project]",
	SecSource:           "[Source]",
	SecSigning:          "[Signing] (passwords masked)",
	LblProjectPath:      "  Project path     : ",
	LblChannelsSource:   "  Channels source  : ",
	LblChannelsCount:    "  Channel count    : ",
	LblChannels:         "  Channels         : ",
	LblExpectedBranch:   "  Expected branch  : ",
	BranchWarnFmt:       "  Branch           : %s  (no -b given, using current branch)",
	LblOutputDir:        "  Output dir       : ",
	LblJobs:             "  Parallelism      : ",
	LblPrefix:           "  Filename prefix  : ",
	LblVersionName:      "  versionName      : ",
	LblVersionCode:      "  versionCode      : ",
	LblCurrentBranch:    "  Current branch   : ",
	LblRecentCommits:    "  Recent 5 commits:",
	NoCommits:           "    (none)",
	LblKeystore:         "  keystore         : ",
	LblKeyAlias:         "  keyAlias         : ",
	LblStorePassword:    "  storePassword    : ",
	LblKeyPassword:      "  keyPassword      : ",
	WarnKeystoreMissing: "  Warning: keystore file not found: ",
	ParseFailedSkipped:  "  (parse failed, skipped) ",
}

var zh = &Messages{
	ArgErrorPrefix: "参数错误: ",
	ArgErrorHint:   "使用 --help 查看用法。",

	PromptProject:   "请输入项目根路径（含 gradlew 的目录）: ",
	ProjectSetRerun: "已设置项目路径，将重新展示完整信息。",
	ErrYesNoProject: "错误: --yes（非交互）模式下必须提供 --project。",
	ErrNoProject:    "错误: 未提供项目路径。",
	PreflightFailed: "前置检查失败: ",

	StepCheckingEnv: "正在检查运行环境（java / git / gradlew）…",
	EnvCheckFailed:  "运行环境检查失败",
	EnvCheckPassed:  "运行环境检查通过",
	StepReadingVer:  "正在读取版本与源码信息（Gradle 配置中，约 30–60s，请稍候）…",
	VerReadFailed:   "读取版本信息失败",
	VerReadDone:     "版本/分支/提交读取完成",

	BranchMismatch:     "分支不匹配",
	BranchMismatchLine: "分支不匹配: 期望 %s，当前 %s\n",

	ChannelsFailed:           "渠道清单错误",
	ChannelsFailedLine:       "渠道清单错误: ",
	ChannelsSourceBuiltin:    "内置默认清单",
	ChannelsSourceBuiltinFmt: "内置默认清单 (%d 个)",
	PrefixInvalid:            "前缀非法",
	PrefixInvalidLine:        "前缀错误: ",
	ReadVerFailedLine:        "读取版本信息失败: ",

	ConfirmInfo:  "以上信息是否正确？是否继续？",
	ConfirmStart: "是否开始打包？",
	Cancelled:    "已取消。",

	MkdirOutputFailed: "创建输出目录失败: ",

	StepBuilding:        "🔨 正在清理并构建基础包 (clean + assembleRelease)…",
	CreateLogFailed:     "创建 Gradle 日志文件失败",
	CreateLogFailedLine: "创建 Gradle 日志文件失败: ",
	BuildFailedFmt:      "基础包构建失败（日志：%s）",
	BuildFailedLine:     "基础包构建失败: ",
	BuildLogLine:        "Gradle 日志: ",
	BuildDoneFmt:        "基础包构建完成（日志：%s）",

	StepCheckingSig: "🔍 正在校验基础包签名方式…",
	SigAbnormalFmt:  "签名方式异常: %q（期望 V2/V3）",
	SigAbnormalLine: "基础包签名方式异常: %q（期望 V2/V3）\n",
	SigModePrefix:   "签名方式: ",

	StepGeneratingFmt: "📦 正在生成渠道包（%d 路并行 · %d 批）…",
	GenProgressFmt:    "📦 生成渠道包 … %d/%d 批完成",
	BatchPutFmt:       "[batch %d] put %d 渠道: %s\n",
	BatchDoneFmt:      "[batch %d] 完成 %d 渠道，耗时 %dms\n",
	GenFailed:         "渠道包生成失败",
	GenFailedLine:     "渠道包生成失败: ",
	GenDoneFmt:        "渠道包生成完成（%d 个）",
	MoveFailedLine:    "移动渠道包失败: ",

	StepVerifying:     "🔎 正在逐包校验渠道…",
	VerifyFailFmt:     "校验失败: 期望=%s 实际=%s",
	VerifyFailLineFmt: "校验失败: 文件=%s 期望=%s 实际=%s 文件名内嵌=%s\n",
	VerifyPassed:      "校验通过",

	SummaryTitle:    "🎉 打包完成",
	SummaryVersion:  "   版本     : %s (%d)",
	SummaryBranch:   "   分支     : %s",
	SummaryChannels: "   渠道数   : %d",
	SummaryOutput:   "   输出目录 : %s",
	SummaryDuration: "   耗时     : %s",

	DisplayTitle:        "==================== 多渠道打包 ====================",
	SecProject:          "【项目信息】",
	SecSource:           "【源码信息】",
	SecSigning:          "【签名信息】（密码已屏蔽）",
	LblProjectPath:      "  项目路径     : ",
	LblChannelsSource:   "  渠道文件来源 : ",
	LblChannelsCount:    "  渠道数       : ",
	LblChannels:         "  渠道         : ",
	LblExpectedBranch:   "  期望分支     : ",
	BranchWarnFmt:       "  分支         : %s  （未指定 -b，使用当前分支）",
	LblOutputDir:        "  输出目录     : ",
	LblJobs:             "  并行路数     : ",
	LblPrefix:           "  文件名前缀   : ",
	LblVersionName:      "  versionName  : ",
	LblVersionCode:      "  versionCode  : ",
	LblCurrentBranch:    "  当前分支     : ",
	LblRecentCommits:    "  最近 5 条提交:",
	NoCommits:           "    (无)",
	LblKeystore:         "  keystore      : ",
	LblKeyAlias:         "  keyAlias      : ",
	LblStorePassword:    "  storePassword : ",
	LblKeyPassword:      "  keyPassword   : ",
	WarnKeystoreMissing: "  警告: keystore 文件不存在: ",
	ParseFailedSkipped:  "  (解析失败，跳过) ",
}

var (
	cur Lang = EN
	// M 是当前语言的完整消息集，由 Set 切换。
	M *Messages = en
)

// Set 切换当前语言（默认 EN）。
func Set(l Lang) {
	if l == ZH {
		cur, M = ZH, zh
		return
	}
	cur, M = EN, en
}

// Current 返回当前语言。
func Current() Lang { return cur }
