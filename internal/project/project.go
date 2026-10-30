// Package project 负责只读采集目标项目信息：版本、分支、提交、签名配置。
//
// 全程不修改目标项目任何文件。
package project

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"channel_build/internal/proc"
)

func gradlew(root string) string { return filepath.Join(root, "gradlew") }

func runCmd(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := proc.Run(cmd)
	return buf.String(), err
}

// ReadVersion 通过 Gradle init script 读取真实 versionName / versionCode（R4）。
// 不解析源码、不修改项目。
func ReadVersion(ctx context.Context, root, initScript string) (string, int, error) {
	out, err := runCmd(ctx, root, gradlew(root),
		"-q", "--no-configuration-cache", "--no-daemon", "-I", initScript, ":app:printAppVersion")
	if err != nil {
		return "", 0, fmt.Errorf("gradle :app:printAppVersion failed: %v: %s", err, tail(out, 800))
	}

	var (
		name string
		code int
	)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "VERSION_NAME="); ok {
			name = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "VERSION_CODE="); ok {
			code, _ = strconv.Atoi(strings.TrimSpace(v))
		}
	}
	if name == "" || code == 0 {
		return "", 0, fmt.Errorf("could not parse VERSION_NAME/VERSION_CODE from gradle output: %s", tail(out, 800))
	}
	return name, code, nil
}

// CurrentBranch 返回当前分支（git rev-parse --abbrev-ref HEAD）。
func CurrentBranch(ctx context.Context, root string) (string, error) {
	out, err := runCmd(ctx, root, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse --abbrev-ref HEAD: %v", err)
	}
	return strings.TrimSpace(out), nil
}

// RecentCommits 返回最近 n 条提交（"%h %s"）。
func RecentCommits(ctx context.Context, root string, n int) ([]string, error) {
	out, err := runCmd(ctx, root, "git", "log", fmt.Sprintf("-%d", n), "--pretty=format:%h %s")
	if err != nil {
		return nil, fmt.Errorf("git log: %v", err)
	}
	var res []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			res = append(res, l)
		}
	}
	return res, nil
}

func tail(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
