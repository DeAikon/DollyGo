// Package report 定义退出码常量与「单行 JSON」结果输出。
//
// 约定（见 prd.md R23）：JSON 是**可选**输出——仅当调用方显式开启（CLI 的 `--json`）时，
// 才在 stdout 打印单行 JSON，且在**所有退出路径**一致输出（含失败）；
// 默认（未开启）stdout 无任何输出，调用方（shell / CI）以**退出码**作为机器信号。
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// 退出码契约（见 prd.md R10）。
const (
	OK             = 0 // 全部成功
	Preflight      = 1 // 前置检查失败
	Cancelled      = 2 // 用户取消
	BuildFailed    = 3 // 基础包构建失败
	GenFailed      = 4 // 渠道包生成失败
	VerifyFailed   = 5 // 渠道校验失败
	Internal       = 6 // 内部/未知错误
	BranchMismatch = 7 // 分支不匹配
)

// Result 是输出到 stdout 的单行 JSON 摘要。
type Result struct {
	Success     bool     `json:"success"`
	VersionName string   `json:"versionName,omitempty"`
	VersionCode int      `json:"versionCode,omitempty"`
	Branch      string   `json:"branch,omitempty"`
	BaseApk     string   `json:"baseApk,omitempty"`
	OutputDir   string   `json:"outputDir,omitempty"`
	Prefix      string   `json:"prefix,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	Verified    []string `json:"verified,omitempty"`
	Jobs        int      `json:"jobs,omitempty"`
	DurationMs  int64    `json:"durationMs"`
	Error       string   `json:"error,omitempty"`
}

var emitOnce sync.Once

// Emit 将结果以单行 JSON 打印到 stdout（仅当 enabled 为 true）。
// 仅生效一次，避免信号路径与主流程重复输出。
func Emit(r Result, enabled bool) {
	if !enabled {
		return
	}
	emitOnce.Do(func() {
		b, err := json.Marshal(r)
		if err != nil {
			// 极端兜底：marshal 失败也要保证输出可解析的 JSON。
			fmt.Fprintf(os.Stdout, "{\"success\":false,\"error\":%q,\"durationMs\":0}\n", "json marshal failed: "+err.Error())
			return
		}
		fmt.Fprintln(os.Stdout, string(b))
	})
}
