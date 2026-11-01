//go:build windows

package claudecli

// terminate kills the process: Windows has no SIGTERM. / terminate kill process: Windows không có SIGTERM.
func (r *Runner) terminate() error { return r.cmd.Process.Kill() }
