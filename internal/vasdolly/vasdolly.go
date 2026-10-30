// Package vasdolly 封装 VasDolly CLI：渠道读取/写入、签名方式读取，以及并行生成。
//
// 关键事实（已核实 VasDolly 3.0.6 源码）：
//   - CLI **失败也返回退出码 0**（Main 无 System.exit，异常仅打印后 return）。
//     因此生成成功与否**不能依赖退出码**，必须以「预期输出文件是否存在」为准。
//   - 输出命名：<channel>-<baseApkName>（基础包 app-release.apk → Xiaomi-app-release.apk）。
//   - get -c 输出：`Channel: <渠道>,len=<长度>`；无渠道时实测为 `Channel: ,len=0`（**仍返回 0**）。
//   - get -s 输出：`signature mode:V2`。
package vasdolly

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"channel_build/internal/i18n"
	"channel_build/internal/proc"
)

// BaseApkName 是目标项目 release 基础包的固定文件名。
const BaseApkName = "app-release.apk"

var (
	// 渠道名允许为空（无渠道的 apk 实测输出 `Channel: ,len=0`），由校验阶段判定不符。
	reChannel = regexp.MustCompile(`Channel:\s*(.*?),len=(\d+)`)
	reSigMode = regexp.MustCompile(`signature mode:(\S+)`)
)

// Runner 执行 VasDolly CLI 调用。
type Runner struct {
	jar string
	log io.Writer
}

// New 创建 Runner。log 为 nil 时不输出进度。
func New(jar string, log io.Writer) *Runner {
	return &Runner{jar: jar, log: log}
}

func (r *Runner) exec(ctx context.Context, args ...string) (string, error) {
	full := append([]string{"-jar", r.jar}, args...)
	cmd := exec.CommandContext(ctx, "java", full...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := proc.Run(cmd)
	return buf.String(), err
}

// SignatureMode 读取 apk 的签名方式（get -s），期望 V2 或 V3。
func (r *Runner) SignatureMode(ctx context.Context, apk string) (string, error) {
	out, err := r.exec(ctx, "get", "-s", apk)
	if err != nil {
		return "", fmt.Errorf("vasdolly get -s failed: %v | output: %s", err, oneLine(out))
	}
	m := reSigMode.FindStringSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("cannot parse signature mode from output: %s", oneLine(out))
	}
	return m[1], nil
}

// Channel 读取 apk 内嵌渠道（get -c）。
func (r *Runner) Channel(ctx context.Context, apk string) (string, error) {
	out, err := r.exec(ctx, "get", "-c", apk)
	if err != nil {
		return "", fmt.Errorf("vasdolly get -c failed: %v | output: %s", err, oneLine(out))
	}
	m := reChannel.FindStringSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("cannot parse channel from output: %s", oneLine(out))
	}
	return m[1], nil
}

// SplitBatches 将渠道轮询均分到 n 批（n 会被裁剪到 [1, len(channels)]）。
func SplitBatches(channels []string, n int) [][]string {
	if len(channels) == 0 {
		return nil
	}
	if n < 1 {
		n = 1
	}
	if n > len(channels) {
		n = len(channels)
	}
	batches := make([][]string, n)
	for i, ch := range channels {
		batches[i%n] = append(batches[i%n], ch)
	}
	return batches
}

// Generate 并行执行各批 put，输出到 genRoot/batch-<i>/。
// onProgress（可空）在每批成功后回调，参数为已完成批次数与总批次数，供上层刷新进度。
// 任一批失败会取消其余批次并返回聚合错误（调用方据此 exit 4）。
func (r *Runner) Generate(ctx context.Context, baseApk, genRoot string, batches [][]string, onProgress func(done, total int)) error {
	gctx, cancel := context.WithCancel(ctx)
	defer cancel()

	total := len(batches)
	var completed atomic.Int64

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []string
	)

	for i, batch := range batches {
		wg.Add(1)
		go func(idx int, chs []string) {
			defer wg.Done()

			dir := filepath.Join(genRoot, fmt.Sprintf("batch-%d", idx))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("batch %d: mkdir: %v", idx, err))
				mu.Unlock()
				cancel()
				return
			}

			start := time.Now()
			if r.log != nil {
				fmt.Fprintf(r.log, i18n.M.BatchPutFmt, idx, len(chs), strings.Join(chs, ","))
			}
			if err := r.putBatch(gctx, baseApk, dir, chs); err != nil {
				mu.Lock()
				errs = append(errs, err.Error())
				mu.Unlock()
				cancel()
				return
			}
			if r.log != nil {
				fmt.Fprintf(r.log, i18n.M.BatchDoneFmt, idx, len(chs), time.Since(start).Milliseconds())
			}
			if onProgress != nil {
				onProgress(int(completed.Add(1)), total)
			}
		}(i, batch)
	}

	wg.Wait()
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// putBatch 执行一批 put，并以「预期输出文件存在性」为主信号判定成败（不依赖退出码）。
func (r *Runner) putBatch(ctx context.Context, baseApk, dir string, channels []string) error {
	joined := strings.Join(channels, ",")
	out, err := r.exec(ctx, "put", "-c", joined, baseApk, dir)
	if err != nil {
		return fmt.Errorf("batch [%s]: jvm non-zero exit: %v | output: %s", joined, err, oneLine(out))
	}

	var missing []string
	for _, ch := range channels {
		p := filepath.Join(dir, ch+"-"+BaseApkName)
		if fi, statErr := os.Stat(p); statErr != nil || fi.Size() == 0 {
			missing = append(missing, ch)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("batch [%s]: expected channel apk missing for %v | output: %s", joined, missing, oneLine(out))
	}
	if strings.Contains(out, "Exception") {
		return fmt.Errorf("batch [%s]: vasdolly reported exception | output: %s", joined, oneLine(out))
	}
	return nil
}

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " | ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > 600 {
		s = s[:600] + "..."
	}
	return s
}
