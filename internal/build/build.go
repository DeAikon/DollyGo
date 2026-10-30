// Package build 封装 gradle 基础包构建。
package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/DeAikon/DollyGo/internal/proc"
)

// AssembleRelease 执行 ./gradlew --no-daemon clean :app:assembleRelease（先 clean 再 assemble），
// 成功后返回基础包绝对路径。
func AssembleRelease(ctx context.Context, root string, log io.Writer) (string, error) {
	if log == nil {
		log = os.Stderr
	}
	cmd := exec.CommandContext(ctx, filepath.Join(root, "gradlew"), "--no-daemon", "clean", ":app:assembleRelease")
	cmd.Dir = root
	cmd.Stdout = log
	cmd.Stderr = log

	// 自成进程组：context 取消（含 SIGINT）时杀掉整个进程组，避免孙进程残留。
	if err := proc.Run(cmd); err != nil {
		return "", fmt.Errorf("gradlew clean :app:assembleRelease failed: %v", err)
	}

	apk := filepath.Join(root, "app", "build", "outputs", "apk", "release", "app-release.apk")
	fi, err := os.Stat(apk)
	if err != nil {
		return "", fmt.Errorf("release apk not found after build: %s", apk)
	}
	if fi.Size() == 0 {
		return "", fmt.Errorf("release apk is empty: %s", apk)
	}
	return apk, nil
}
