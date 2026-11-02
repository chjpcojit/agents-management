//go:build !windows

package claudecli

import (
	"errors"
	"os/exec"
	"syscall"
)

// setProcessGroup starts claude as the leader of its own process group, so the commands its tools
// spawn can be signalled together: spike T12 saw claude exit while a tool's child kept running.
//
// setProcessGroup chạy claude làm leader của process group riêng, để các lệnh do tool của nó sinh ra
// được gửi signal cùng lúc: spike T12 thấy claude thoát trong khi process con của tool vẫn chạy.
func setProcessGroup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// terminate sends SIGTERM to the whole group. / terminate gửi SIGTERM cho cả group.
func (r *Runner) terminate() error { return signalGroup(r.cmd.Process.Pid, syscall.SIGTERM) }

// killGroup sends SIGKILL to the whole group. ESRCH (nothing left) is not an error.
// killGroup gửi SIGKILL cho cả group. ESRCH (không còn process nào) không phải lỗi.
func (r *Runner) killGroup() error { return signalGroup(r.cmd.Process.Pid, syscall.SIGKILL) }

func signalGroup(pgid int, sig syscall.Signal) error {
	if err := syscall.Kill(-pgid, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
