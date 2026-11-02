//go:build windows

package claudecli

import (
	"errors"
	"os"
	"os/exec"
)

// setProcessGroup is a no-op on Windows; children are not tracked there (spike runs on macOS/Linux).
// setProcessGroup không làm gì trên Windows; process con không được theo dõi (spike chạy trên macOS/Linux).
func setProcessGroup(*exec.Cmd) {}

// terminate kills the process: Windows has no SIGTERM. / terminate kill process: Windows không có SIGTERM.
func (r *Runner) terminate() error { return r.killGroup() }

// killGroup kills the process only. / killGroup chỉ kill process chính.
func (r *Runner) killGroup() error {
	if err := r.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}
