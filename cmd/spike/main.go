// Command spike drives real `claude -p` processes through the runner and the MCP server to answer
// the open questions of the agent-runtime spec. It runs on the owner's machine, where the Max login lives.
//
//	spike preflight   check claude, login and env
//	spike run         interactive session with manual approvals
//	spike test        run T1–T15 and write docs/spike-report.md
//
// Command spike điều khiển process `claude -p` thật qua runner và MCP server để trả lời các câu hỏi
// mở của spec agent-runtime. Chạy trên máy owner, nơi có login Max.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

const usage = `usage: spike <command> [flags]

commands:
  preflight   check claude binary, Max login and env
  run         interactive session with manual approvals
  test        run cases T1–T15 and write a report

run "spike <command> -h" for flags
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "preflight":
		err = cmdPreflight(os.Args[2:], os.Stdout)
	case "run":
		err = cmdInteractive(os.Args[2:], os.Stdin, os.Stdout)
	case "test":
		err = cmdTest(os.Args[2:], os.Stdout)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// newLogger logs to stderr: warnings by default, everything with -v. Logs never carry prompts or tool input.
// newLogger ghi log ra stderr: mặc định từ warn, -v thì ghi hết. Log không bao giờ chứa prompt hay tool input.
func newLogger(verbose bool) *slog.Logger {
	if !verbose {
		return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// signalContext ends on Ctrl-C so a running case stops and the report still gets written.
// signalContext kết thúc khi Ctrl-C để case đang chạy dừng lại và report vẫn được ghi.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
