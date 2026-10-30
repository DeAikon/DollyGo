// Package progress 提供打包期的人类友好进度提示（emoji + spinner）。
//
// 约定（见 design.md §17 / prd.md R14）：
//   - 所有进度输出走 **stderr**，stdout 仍只保留单行 JSON 契约。
//   - spinner 动画**仅在 stderr 为 TTY 时**启用；非 TTY（管道 / CI）降级为静态行，
//     绝不产生 `\r`，避免污染日志。
//   - TTY 判定复用 internal/cli 的既有逻辑（字符设备 + NO_COLOR）。
package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/DeAikon/DollyGo/internal/cli"
)

// spinnerFrames 是 braille 帧序列。
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const (
	frameInterval = 100 * time.Millisecond
	clearLine     = "\r\033[K" // 回到行首并清除整行
)

// Spinner 在 stderr 上渲染「进行中 → 终态」提示，可跨多个步骤复用。
//
// TTY：每 ~100ms 以 `\r` 重绘一帧；Stop/Success/Fail 先清行再输出终态行。
// 非 TTY：Start 打印 "⏳ <msg>"，Success/Fail 打印 "✅/❌ <msg>"，不产生 `\r`。
type Spinner struct {
	w   io.Writer
	tty bool

	// writeMu 串行化对 w 的写入（spinner 帧 vs 子进程输出），避免交错。
	writeMu sync.Mutex

	mu       sync.Mutex
	msg      string
	running  bool
	stopCh   chan struct{}
	loopDone chan struct{}
}

// New 创建 Spinner。w 为 nil 时使用 os.Stderr。
func New(w io.Writer) *Spinner {
	if w == nil {
		w = os.Stderr
	}
	return &Spinner{w: w, tty: cli.TTYEnabled()}
}

// Start 开始一个步骤。若已在运行则仅更新消息。
func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	s.msg = msg
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	if !s.tty {
		s.mu.Unlock()
		s.writeLine(cli.Paint(cli.CatStep, "⏳ "+msg))
		return
	}
	stopCh := make(chan struct{})
	loopDone := make(chan struct{})
	s.stopCh = stopCh
	s.loopDone = loopDone
	s.mu.Unlock()

	go s.loop(stopCh, loopDone)
}

// Update 更新进行中消息（线程安全，供并行批次回调使用）。
func (s *Spinner) Update(msg string) {
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
}

// Stop 结束动画并清除当前行；可重复调用。
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	stopCh := s.stopCh
	loopDone := s.loopDone
	s.stopCh, s.loopDone = nil, nil
	s.mu.Unlock()

	if s.tty && stopCh != nil {
		close(stopCh)
		<-loopDone
		s.writeMu.Lock()
		fmt.Fprint(s.w, clearLine)
		s.writeMu.Unlock()
	}
}

// Success 结束当前步骤并打印成功终态行。
func (s *Spinner) Success(msg string) {
	s.Stop()
	s.writeLine(cli.Paint(cli.CatSuccess, "✅ "+msg))
}

// Fail 结束当前步骤并打印失败终态行。
func (s *Spinner) Fail(msg string) {
	s.Stop()
	s.writeLine(cli.Paint(cli.CatError, "❌ "+msg))
}

// OutputWriter 返回与 spinner 协调的 writer：写入前清除当前 spinner 行，
// 避免子进程输出（gradle / VasDolly）与动画帧互相污染。仅 TTY 且运行中才清行。
func (s *Spinner) OutputWriter() io.Writer { return &spinWriter{s: s} }

// writeLine 在持写锁下输出一行终态文本。
func (s *Spinner) writeLine(text string) {
	s.writeMu.Lock()
	fmt.Fprintln(s.w, text)
	s.writeMu.Unlock()
}

// loop 以固定间隔重绘 spinner 帧，直到 stopCh 关闭。
func (s *Spinner) loop(stopCh, loopDone chan struct{}) {
	defer close(loopDone)
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	i := 0
	s.render(i)
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			i = (i + 1) % len(spinnerFrames)
			s.render(i)
		}
	}
}

// render 绘制一帧：回到行首、写帧+消息、清除到行尾（避免短消息残留）。
func (s *Spinner) render(i int) {
	s.mu.Lock()
	msg, running := s.msg, s.running
	s.mu.Unlock()
	if !running {
		return
	}
	s.writeMu.Lock()
	fmt.Fprintf(s.w, "\r%s\033[K", cli.Paint(cli.CatStep, spinnerFrames[i]+" "+msg))
	s.writeMu.Unlock()
}

// spinWriter 是 Spinner.OutputWriter 返回的协调 writer。
type spinWriter struct{ s *Spinner }

func (sw *spinWriter) Write(p []byte) (int, error) {
	s := sw.s
	s.mu.Lock()
	clear := s.tty && s.running
	s.mu.Unlock()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if clear {
		fmt.Fprint(s.w, clearLine)
	}
	return s.w.Write(p)
}
