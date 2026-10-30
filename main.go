// channelbuild - VasDolly 多渠道打包工具
//
// 流程：解析参数 → 前置检查 → 分支校验 → 采集信息 → 确认 → 构建基础包 →
//
//	并行生成渠道包 → 重命名/移动 → 逐包校验 → 输出 JSON。
//
// 设计文档：.trellis/tasks/10-26-vasdolly-multichannel-build/{prd,design,implement}.md
package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"channel_build/internal/build"
	"channel_build/internal/cli"
	"channel_build/internal/embedx"
	"channel_build/internal/i18n"
	"channel_build/internal/proc"
	"channel_build/internal/progress"
	"channel_build/internal/project"
	"channel_build/internal/report"
	"channel_build/internal/vasdolly"
)

// 内嵌资源（Go embed 不支持 ".."，故声明在模块根的 main 包）。
//
//go:embed assets/VasDolly.jar
//go:embed assets/channels.default.txt
//go:embed assets/version-init.gradle
var assetsFS embed.FS

// interrupted 由信号处理器置位。main 据此把退出码统一收敛为 Cancelled(2)：
// 否则主流程被中断后走「构建失败」路径可能抢先 os.Exit(3)，与信号契约冲突。
var interrupted atomic.Bool

func main() {
	code := run()
	if interrupted.Load() {
		code = report.Cancelled
	}
	os.Exit(code)
}

func run() int {
	start := time.Now()
	ms := func() int64 { return time.Since(start).Milliseconds() }

	opts, err := cli.Parse(os.Args[1:])
	if err != nil {
		// 预扫描 --lzh，使解析失败提示也能用正确语言（此时 opts 不可用）。
		i18n.Set(langFromArgs(os.Args[1:]))
		report.Emit(report.Result{Success: false, Error: "args: " + err.Error(), DurationMs: ms()}, jsonFromArgs(os.Args[1:]))
		fmt.Fprintln(os.Stderr, i18n.M.ArgErrorPrefix+err.Error())
		fmt.Fprintln(os.Stderr, i18n.M.ArgErrorHint)
		return report.Preflight
	}
	// 解析后、任何输出之前立即设定语言（默认英文，--lzh 中文）。
	i18n.Set(i18n.EN)
	if opts.LZH {
		i18n.Set(i18n.ZH)
	}
	if opts.Help {
		cli.PrintHelp(os.Stdout)
		return report.OK
	}

	// JSON 输出默认关闭，仅 --json 时写 stdout（R23）。
	jsonOut := opts.JSON

	w := os.Stderr

	// 临时目录 + 释放内嵌资源（信号清理依赖它，故尽早创建）。
	tmpDir, err := os.MkdirTemp("", "channelbuild-*")
	if err != nil {
		report.Emit(report.Result{Success: false, Error: "internal: mkdtemp: " + err.Error(), DurationMs: ms()}, jsonOut)
		return report.Internal
	}
	res, err := embedx.Prepare(assetsFS, tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		report.Emit(report.Result{Success: false, Error: "internal: release embedded assets: " + err.Error(), DurationMs: ms()}, jsonOut)
		return report.Internal
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 打包期进度全部走 stderr；所有 spinner 必须在 report.Emit 之前 Stop。
	// 提前创建，使信号处理器能 Stop 掉正在运行的 spinner（否则 JSON 会与动画混行）。
	sp := progress.New(w)

	// 信号处理：Stop spinner → 输出 JSON → 取消 context → 杀整个进程组 → 清理临时目录 → 退出。
	// os.Exit 会跳过 defer，且不等 os/exec 的 watchCtx goroutine，故必须显式清理 + 同步杀组。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-sigCh:
			// 先 Stop spinner 清行，再输出 JSON（sync.Once 保证唯一），再 cancel；
			// 否则子进程被杀会让主流程抢先输出「构建失败」，导致 JSON 与退出码不一致。
			interrupted.Store(true)
			sp.Stop()
			report.Emit(report.Result{Success: false, Error: "interrupted by signal", DurationMs: ms()}, jsonOut)
			cancel()
			proc.KillAll()
			_ = os.RemoveAll(tmpDir)
			os.Exit(report.Cancelled)
		case <-done:
		}
	}()
	defer func() {
		signal.Stop(sigCh)
		close(done)
		_ = os.RemoveAll(tmpDir)
	}()

	reader := bufio.NewReader(os.Stdin)

	// R3.1：缺 --project 时交互补全；--yes 模式下直接失败（禁止挂起，AC17）。
	if strings.TrimSpace(opts.Project) == "" {
		if opts.Yes {
			report.Emit(report.Result{Success: false, Error: "preflight: --project is required in --yes mode", DurationMs: ms()}, jsonOut)
			fmt.Fprintln(w, i18n.M.ErrYesNoProject)
			return report.Preflight
		}
		p, perr := cli.PromptProject(reader, w)
		if perr != nil || strings.TrimSpace(p) == "" {
			report.Emit(report.Result{Success: false, Error: "preflight: no project path provided", DurationMs: ms()}, jsonOut)
			fmt.Fprintln(w, i18n.M.ErrNoProject)
			return report.Preflight
		}
		opts.Project = strings.TrimSpace(p)
		fmt.Fprintln(w, i18n.M.ProjectSetRerun)
	}
	projectRoot := opts.Project
	if abs, absErr := filepath.Abs(opts.Project); absErr == nil {
		projectRoot = abs
	}

	// R18：启动期进度提示，消除「启动 → 信息展示」之间的空窗（主因：Gradle 配置读取版本）。
	sp.Start(i18n.M.StepCheckingEnv)

	// preflight（exit 1）
	if perr := preflight(projectRoot, res); perr != nil {
		sp.Fail(i18n.M.EnvCheckFailed)
		report.Emit(report.Result{Success: false, Error: "preflight: " + perr.Error(), DurationMs: ms()}, jsonOut)
		fmt.Fprintln(w, i18n.M.PreflightFailed+perr.Error())
		return report.Preflight
	}
	sp.Success(i18n.M.EnvCheckPassed)

	// R18：版本/源码采集（含分支校验）耗时主要来自 Gradle 配置（约 30–60s），持续给出进行中提示。
	sp.Start(i18n.M.StepReadingVer)

	// 分支校验（先于信息采集，fail fast，exit 7）。
	curBranch, berr := project.CurrentBranch(ctx, projectRoot)
	if berr != nil {
		curBranch = ""
	}
	if opts.Branch != "" && curBranch != "" && curBranch != opts.Branch {
		sp.Fail(i18n.M.BranchMismatch)
		report.Emit(report.Result{
			Success:    false,
			Branch:     curBranch,
			Error:      fmt.Sprintf("branch mismatch: expected %q, current %q", opts.Branch, curBranch),
			DurationMs: ms(),
		}, jsonOut)
		fmt.Fprintf(w, i18n.M.BranchMismatchLine, opts.Branch, curBranch)
		return report.BranchMismatch
	}

	// 渠道清单 + 校验（先于耗时的 gradle 版本读取，尽早失败）。
	channels, srcDesc, cerr := loadChannels(opts, res)
	if cerr != nil {
		sp.Fail(i18n.M.ChannelsFailed)
		report.Emit(report.Result{Success: false, Branch: curBranch, Error: "preflight: " + cerr.Error(), DurationMs: ms()}, jsonOut)
		fmt.Fprintln(w, i18n.M.ChannelsFailedLine+cerr.Error())
		return report.Preflight
	}
	if perr := validatePrefix(opts.Prefix); perr != nil {
		sp.Fail(i18n.M.PrefixInvalid)
		report.Emit(report.Result{Success: false, Branch: curBranch, Error: "preflight: " + perr.Error(), DurationMs: ms()}, jsonOut)
		fmt.Fprintln(w, i18n.M.PrefixInvalidLine+perr.Error())
		return report.Preflight
	}
	jobs := resolveJobs(opts.Jobs, len(channels))

	// 采集版本信息（Gradle init script，只读）。
	versionName, versionCode, verr := project.ReadVersion(ctx, projectRoot, res.InitScriptPath)
	if verr != nil {
		sp.Fail(i18n.M.VerReadFailed)
		report.Emit(report.Result{Success: false, Branch: curBranch, Error: "preflight: read version: " + verr.Error(), DurationMs: ms()}, jsonOut)
		fmt.Fprintln(w, i18n.M.ReadVerFailedLine+verr.Error())
		return report.Preflight
	}
	commits, _ := project.RecentCommits(ctx, projectRoot, 5)
	signing := project.ParseSigning(projectRoot)
	sp.Success(i18n.M.VerReadDone)

	outputDir := opts.OutputDir
	if strings.TrimSpace(outputDir) == "" {
		home, _ := os.UserHomeDir()
		outputDir = filepath.Join(home, "multi_channel_build")
	}
	outputDir = expandHome(outputDir)

	// 确认展示。
	cli.Display(w, cli.DisplayInfo{
		ProjectRoot:    projectRoot,
		ChannelsSource: srcDesc,
		ChannelsCount:  len(channels),
		Channels:       channels,
		ExpectedBranch: opts.Branch,
		CurrentBranch:  curBranch,
		OutputDir:      outputDir,
		Jobs:           jobs,
		Prefix:         opts.Prefix,
		VersionName:    versionName,
		VersionCode:    versionCode,
		Commits:        commits,
		Signing: cli.SigningView{
			Found:          signing.Found,
			KeystorePath:   signing.StoreFile,
			KeyAlias:       signing.KeyAlias,
			KeystoreExists: signing.KeystoreExists,
			Warn:           signing.Warn,
		},
	})

	base := report.Result{
		VersionName: versionName,
		VersionCode: versionCode,
		Branch:      curBranch,
		OutputDir:   outputDir,
		Prefix:      opts.Prefix,
		Channels:    channels,
		Jobs:        jobs,
	}

	// 两次确认（--yes 跳过）；选 N → exit 2 且不执行构建（AC6）。
	if !opts.Yes {
		ok, _ := cli.Confirm(reader, w, i18n.M.ConfirmInfo)
		if !ok {
			base.Error = "cancelled by user at confirm step"
			base.DurationMs = ms()
			report.Emit(base, jsonOut)
			fmt.Fprintln(w, i18n.M.Cancelled)
			return report.Cancelled
		}
		ok2, _ := cli.Confirm(reader, w, i18n.M.ConfirmStart)
		if !ok2 {
			base.Error = "cancelled by user at start step"
			base.DurationMs = ms()
			report.Emit(base, jsonOut)
			fmt.Fprintln(w, i18n.M.Cancelled)
			return report.Cancelled
		}
	}

	// outputDir 必须在构建前创建：Gradle 日志要写入 <outputDir>/gradle-build.log（R15）。
	if mkErr := os.MkdirAll(outputDir, 0o755); mkErr != nil {
		base.Error = "internal: mkdir output dir: " + mkErr.Error()
		base.DurationMs = ms()
		report.Emit(base, jsonOut)
		fmt.Fprintln(w, i18n.M.MkdirOutputFailed+mkErr.Error())
		return report.Internal
	}
	gradleLogPath := filepath.Join(outputDir, "gradle-build.log")

	// 构建基础包（exit 3）。
	// Gradle 的 stdout/stderr 全部落文件（覆盖创建），CLI 仅显示 spinner，不再被 Gradle 原始输出刷屏（R15 / AC19）。
	sp.Start(i18n.M.StepBuilding)
	gradleLog, logErr := os.Create(gradleLogPath)
	if logErr != nil {
		sp.Fail(i18n.M.CreateLogFailed)
		base.Error = "internal: create gradle log: " + logErr.Error()
		base.DurationMs = ms()
		report.Emit(base, jsonOut)
		fmt.Fprintln(w, i18n.M.CreateLogFailedLine+logErr.Error())
		return report.Internal
	}
	baseApk, err := build.AssembleRelease(ctx, projectRoot, gradleLog)
	_ = gradleLog.Close()
	if err != nil {
		sp.Fail(fmt.Sprintf(i18n.M.BuildFailedFmt, gradleLogPath))
		base.Error = "build: " + err.Error()
		base.DurationMs = ms()
		report.Emit(base, jsonOut)
		fmt.Fprintln(w, i18n.M.BuildFailedLine+err.Error())
		fmt.Fprintln(w, i18n.M.BuildLogLine+gradleLogPath)
		return report.BuildFailed
	}
	sp.Success(fmt.Sprintf(i18n.M.BuildDoneFmt, gradleLogPath))
	base.BaseApk = baseApk

	runner := vasdolly.New(res.JarPath, sp.OutputWriter())

	// put 前断言基础包签名方式为 V2/V3（避免静默回退 V1 注释模式）。
	sp.Start(i18n.M.StepCheckingSig)
	mode, err := runner.SignatureMode(ctx, baseApk)
	if err != nil || (mode != "V2" && mode != "V3") {
		sp.Fail(fmt.Sprintf(i18n.M.SigAbnormalFmt, mode))
		base.Error = fmt.Sprintf("gen: base apk signature mode is %q (want V2/V3): %v", mode, err)
		base.DurationMs = ms()
		report.Emit(base, jsonOut)
		fmt.Fprintf(w, i18n.M.SigAbnormalLine, mode)
		return report.GenFailed
	}
	sp.Success(i18n.M.SigModePrefix + mode)

	// 并行生成渠道包（exit 4）。
	batches := vasdolly.SplitBatches(channels, jobs)
	totalBatches := len(batches)
	sp.Start(fmt.Sprintf(i18n.M.StepGeneratingFmt, jobs, totalBatches))
	genRoot := filepath.Join(tmpDir, "gen")
	if err := runner.Generate(ctx, baseApk, genRoot, batches, func(done, _ int) {
		sp.Update(fmt.Sprintf(i18n.M.GenProgressFmt, done, totalBatches))
	}); err != nil {
		sp.Fail(i18n.M.GenFailed)
		base.Error = "gen: " + err.Error()
		base.DurationMs = ms()
		report.Emit(base, jsonOut)
		fmt.Fprintln(w, i18n.M.GenFailedLine+err.Error())
		return report.GenFailed
	}
	sp.Success(fmt.Sprintf(i18n.M.GenDoneFmt, len(channels)))

	// 重命名 + 移动到输出目录（outputDir 已在构建前创建）。
	vnUnderscore := strings.ReplaceAll(versionName, ".", "_")
	type outFile struct {
		channel string
		path    string
	}
	var outs []outFile
	for i, batch := range batches {
		dir := filepath.Join(genRoot, fmt.Sprintf("batch-%d", i))
		for _, ch := range batch {
			src := filepath.Join(dir, ch+"-"+vasdolly.BaseApkName)
			name := fmt.Sprintf("%s_%s_%s(%d).apk", opts.Prefix, ch, vnUnderscore, versionCode)
			dst := filepath.Join(outputDir, name)
			if err := moveFile(src, dst); err != nil {
				base.Error = "internal: move channel apk: " + err.Error()
				base.DurationMs = ms()
				report.Emit(base, jsonOut)
				fmt.Fprintln(w, i18n.M.MoveFailedLine+err.Error())
				return report.Internal
			}
			outs = append(outs, outFile{channel: ch, path: dst})
		}
	}

	// 逐包三重校验（exit 5）：读回值 == 渠道文件预期 == 文件名内嵌渠道。
	sp.Start(i18n.M.StepVerifying)
	verified := make([]string, 0, len(outs))
	for _, o := range outs {
		got, verr := runner.Channel(ctx, o.path)
		nameCh := channelFromFilename(filepath.Base(o.path), opts.Prefix, vnUnderscore, versionCode)
		if verr != nil || got != o.channel || nameCh != o.channel {
			gotStr := got
			if verr != nil {
				gotStr = "(read failed: " + verr.Error() + ")"
			}
			sp.Fail(fmt.Sprintf(i18n.M.VerifyFailFmt, o.channel, gotStr))
			base.Error = fmt.Sprintf("verify: file=%s expected=%s got=%s filename=%s", o.path, o.channel, gotStr, nameCh)
			base.DurationMs = ms()
			report.Emit(base, jsonOut)
			fmt.Fprintf(w, i18n.M.VerifyFailLineFmt, o.path, o.channel, gotStr, nameCh)
			return report.VerifyFailed
		}
		verified = append(verified, o.channel)
	}
	sp.Success(i18n.M.VerifyPassed)

	base.Success = true
	base.Verified = verified
	base.DurationMs = ms()
	report.Emit(base, jsonOut)

	// 🎉 完成汇总（stderr；JSON 已输出到 stdout）。
	fmt.Fprintln(w, cli.Paint(cli.CatSummary, i18n.M.SummaryTitle))
	fmt.Fprintln(w, cli.Paint(cli.CatSummary, fmt.Sprintf(i18n.M.SummaryVersion, versionName, versionCode)))
	fmt.Fprintln(w, cli.Paint(cli.CatSummary, fmt.Sprintf(i18n.M.SummaryBranch, curBranch)))
	fmt.Fprintln(w, cli.Paint(cli.CatSummary, fmt.Sprintf(i18n.M.SummaryChannels, len(verified))))
	fmt.Fprintln(w, cli.Paint(cli.CatSummary, fmt.Sprintf(i18n.M.SummaryOutput, outputDir)))
	fmt.Fprintln(w, cli.Paint(cli.CatSummary, fmt.Sprintf(i18n.M.SummaryDuration, (time.Duration(base.DurationMs)*time.Millisecond).String())))
	return report.OK
}

func langFromArgs(args []string) i18n.Lang {
	for _, a := range args {
		if a == "--lzh" || a == "-lzh" {
			return i18n.ZH
		}
	}
	return i18n.EN
}

// jsonFromArgs 在参数解析成功前预扫描 --json，使早期失败路径也能按开关输出 JSON。
func jsonFromArgs(args []string) bool {
	for _, a := range args {
		if a == "--json" || a == "-json" {
			return true
		}
	}
	return false
}

// preflight 校验项目目录、gradlew、java、git 与已释放的内嵌资源。
func preflight(root string, res *embedx.Resources) error {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("project dir not found: %s", root)
	}
	gradlew := filepath.Join(root, "gradlew")
	fi, statErr := os.Stat(gradlew)
	if statErr != nil || fi.IsDir() {
		return fmt.Errorf("gradlew not found: %s", gradlew)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("gradlew is not executable: %s (run: chmod +x gradlew)", gradlew)
	}
	if _, lookErr := exec.LookPath("java"); lookErr != nil {
		return fmt.Errorf("java not found in PATH")
	}
	if _, lookErr := exec.LookPath("git"); lookErr != nil {
		return fmt.Errorf("git not found in PATH")
	}
	if _, statErr := os.Stat(res.JarPath); statErr != nil {
		return fmt.Errorf("embedded VasDolly.jar not released: %v", statErr)
	}
	if _, statErr := os.Stat(res.InitScriptPath); statErr != nil {
		return fmt.Errorf("embedded version-init.gradle not released: %v", statErr)
	}
	return nil
}

// loadChannels 读取渠道清单：传入 --channels-file 则整体覆盖内置默认。
func loadChannels(opts *cli.Options, res *embedx.Resources) (channels []string, source string, err error) {
	var raw []byte
	if strings.TrimSpace(opts.ChannelsFile) != "" {
		raw, err = os.ReadFile(opts.ChannelsFile)
		if err != nil {
			return nil, opts.ChannelsFile, fmt.Errorf("read channels file %s: %w", opts.ChannelsFile, err)
		}
		source = opts.ChannelsFile
	} else {
		raw = res.DefaultChannels
		source = i18n.M.ChannelsSourceBuiltin
	}
	channels, err = parseChannels(raw)
	if err != nil {
		return nil, source, err
	}
	if strings.TrimSpace(opts.ChannelsFile) == "" {
		source = fmt.Sprintf(i18n.M.ChannelsSourceBuiltinFmt, len(channels))
	}
	return channels, source, nil
}

// parseChannels 逐行解析渠道：trim、跳过空行与 # 注释，拒绝非法/重复渠道名。
func parseChannels(raw []byte) ([]string, error) {
	var out []string
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.ContainsAny(line, ",/\t ") {
			return nil, fmt.Errorf("invalid channel name %q: must not contain ',', '/', or whitespace", line)
		}
		if seen[line] {
			return nil, fmt.Errorf("duplicate channel name %q", line)
		}
		seen[line] = true
		out = append(out, line)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("channel list is empty (empty file or only comments)")
	}
	return out, nil
}

// validatePrefix 校验 --prefix 是合法文件名片段。
func validatePrefix(p string) error {
	if strings.TrimSpace(p) == "" {
		return fmt.Errorf("--prefix must not be empty")
	}
	if strings.ContainsAny(p, "/\\ \t") {
		return fmt.Errorf("--prefix %q contains illegal filename characters", p)
	}
	return nil
}

// resolveJobs 计算实际并行路数：默认 4，且不超过渠道数（R9）。
func resolveJobs(requested, n int) int {
	if n <= 0 {
		return 1
	}
	if requested <= 0 {
		requested = 4
	}
	if requested > n {
		return n
	}
	return requested
}

// expandHome 展开路径开头的 ~。
func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
		}
	}
	return p
}

// channelFromFilename 从 {prefix}_{channel}_{vn}({vc}).apk 反解渠道名；不匹配返回 ""。
func channelFromFilename(name, prefix, vnUnderscore string, vc int) string {
	pre := prefix + "_"
	suf := fmt.Sprintf("_%s(%d).apk", vnUnderscore, vc)
	if !strings.HasPrefix(name, pre) || !strings.HasSuffix(name, suf) {
		return ""
	}
	inner := name[len(pre) : len(name)-len(suf)]
	return inner
}

// moveFile 移动文件（跨设备时退化为 copy+remove）。同名默认覆盖。
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		return err
	}
	return os.Remove(src)
}
