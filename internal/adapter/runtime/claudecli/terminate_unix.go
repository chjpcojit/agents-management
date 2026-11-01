//go:build !windows

package claudecli

import "syscall"

// terminate asks the process to stop with SIGTERM. / terminate yêu cầu process dừng bằng SIGTERM.
func (r *Runner) terminate() error { return r.cmd.Process.Signal(syscall.SIGTERM) }
