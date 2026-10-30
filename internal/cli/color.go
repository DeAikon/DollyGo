package cli

import (
	"fmt"
	"os"
)

// Category 是「按信息归属着色」的类别（莫奈风配色，见 design §19）。
type Category int

const (
	CatTitle     Category = iota // 标题 / 分隔线
	CatParam                     // 参数名/值
	CatInfo                      // 采集信息（版本/分支/提交）
	CatStep                      // 步骤进行中（⏳/spinner）
	CatSuccess                   // 成功（✅）
	CatWarn                      // 警告 / 分支（未指定 -b）
	CatError                     // 失败（❌）
	CatSummary                   // 汇总（🎉）
	CatMuted                     // 次要 / 弱化
	CatHighlight                 // 重点：versionName / versionCode / 当前分支
)

// monetCode 是各类别对应的 ANSI 256 色码（莫奈风柔和色，见 design §19）。
//
// 整体调亮：原色在纯黑背景偏暗，改用中等亮度色，兼顾白/黑背景可读性。
//
//	Title   75 天蓝     Param   80 青       Info   109 灰青
//	Step    80 青       Success 77 亮绿     Warn   215 橙（额外加粗）
//	Error  210 亮玫瑰   Summary 179 暖金    Muted  247 灰
//	Highlight 220 莫奈黄（无加粗）
var monetCode = map[Category]int{
	CatTitle:     75,
	CatParam:     80,
	CatInfo:      109,
	CatStep:      80,
	CatSuccess:   77,
	CatWarn:      215,
	CatError:     210,
	CatSummary:   179,
	CatMuted:     247,
	CatHighlight: 220,
}

// colorEnabled 判断是否应对 stderr 上色。
// 非 TTY（被重定向/管道）或设置了 NO_COLOR 时不上色，避免污染日志。
func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// TTYEnabled 报告 stderr 是否为交互式终端且未设置 NO_COLOR。
// 供进度组件（internal/progress）复用同一判定：仅此时才启用动画/清行。
func TTYEnabled() bool { return colorEnabled() }

// Paint 按类别将文本包裹为对应莫奈色（Warn 额外加粗）。
// 非 TTY 或设置了 NO_COLOR 时原样返回，保证管道/CI 输出为纯文本。
//
// 约束：传入的 s 不应包含换行——每行/每段单独着色，避免转义跨行。
// 颜色只应用于 stderr 文本；stdout 的单行 JSON 契约不受影响。
func Paint(cat Category, s string) string {
	if !colorEnabled() {
		return s
	}
	code, ok := monetCode[cat]
	if !ok {
		return s
	}
	if cat == CatWarn {
		return fmt.Sprintf("\033[1;38;5;%dm%s\033[0m", code, s)
	}
	return fmt.Sprintf("\033[38;5;%dm%s\033[0m", code, s)
}
