// Package proc 统一子进程的进程组与取消语义。
//
// 目的：exec.CommandContext 默认只杀**直接子进程**，孙进程（gradle daemon 派生、
// mock 脚本里的 sleep 等）在 SIGINT 后会残留。这里让每个子进程自成进程组，并在
// context 取消时 SIGKILL **整个进程组**；同时登记活跃进程组，供信号处理器在
// os.Exit 之前**同步**杀掉所有子进程组（os.Exit 会跳过 defer，且不等待 os/exec
// 的 watchCtx goroutine 被调度）。
package proc

import (
	"os/exec"
	"sync"
	"syscall"
)

var (
	mu    sync.Mutex
	pgids = map[int]struct{}{} // 活跃进程组组长 pid（== pgid）
)

// Configure 让 cmd 自成进程组，并在 context 取消时杀掉整个进程组。
// cmd 必须由 exec.CommandContext 创建（Cancel 才会被触发）。
func Configure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
}

// Run 配置进程组、启动、登记进程组、等待并注销。
// 登记使 KillAll 能同步终止所有存活进程组。
func Run(cmd *exec.Cmd) error {
	Configure(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	register(pid)
	err := cmd.Wait()
	unregister(pid)
	return err
}

// KillAll 同步 SIGKILL 所有已登记的进程组。并发安全，可重复调用。
func KillAll() {
	mu.Lock()
	defer mu.Unlock()
	for pid := range pgids {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
}

func register(pid int) {
	mu.Lock()
	pgids[pid] = struct{}{}
	mu.Unlock()
}

func unregister(pid int) {
	mu.Lock()
	delete(pgids, pid)
	mu.Unlock()
}
