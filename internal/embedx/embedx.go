// Package embedx 负责把内嵌资源释放到系统通用临时目录。
//
// 内嵌资源（VasDolly.jar / version-init.gradle / channels.default.txt）由 main 包通过
// //go:embed 声明（Go 的 embed 路径不能使用 ".."，故声明位于模块根的 main.go）。
package embedx

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Resources 描述释放后的资源路径与内存中的默认渠道清单。
type Resources struct {
	JarPath         string
	InitScriptPath  string
	DefaultChannels []byte
}

// Prepare 将内嵌资源释放到 tmpDir，并返回其路径。
// 调用方负责在结束（含信号中断）时删除 tmpDir。
func Prepare(assetsFS fs.FS, tmpDir string) (*Resources, error) {
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return nil, fmt.Errorf("sub fs: %w", err)
	}

	jar := filepath.Join(tmpDir, "VasDolly.jar")
	if err := writeEmbedded(sub, "VasDolly.jar", jar, 0o644); err != nil {
		return nil, err
	}

	initScript := filepath.Join(tmpDir, "version-init.gradle")
	if err := writeEmbedded(sub, "version-init.gradle", initScript, 0o644); err != nil {
		return nil, err
	}

	dc, err := fs.ReadFile(sub, "channels.default.txt")
	if err != nil {
		return nil, fmt.Errorf("read embedded channels.default.txt: %w", err)
	}

	return &Resources{
		JarPath:         jar,
		InitScriptPath:  initScript,
		DefaultChannels: dc,
	}, nil
}

func writeEmbedded(fsys fs.FS, name, dst string, perm os.FileMode) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("read embedded %s: %w", name, err)
	}
	if err := os.WriteFile(dst, b, perm); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}
