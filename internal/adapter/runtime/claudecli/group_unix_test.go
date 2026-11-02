//go:build !windows

package claudecli_test

import (
	"context"
	"errors"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// spike T12: claude exited on SIGINT while its tool's child kept running. Stop must take the child down too.
// spike T12: claude thoát khi nhận SIGINT trong khi process con của tool vẫn chạy. Stop phải dừng cả process con.
func TestStopKillsChildProcesses(t *testing.T) {
	r := start(t, nil)
	if err := r.Send("child"); err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(resultText(nextResult(t, r)))
	if err != nil {
		t.Fatalf("fake did not report a child pid: %v", err)
	}
	go drain(r)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("child %d still alive after Stop (kill 0: %v)", pid, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// A child left behind when claude exits by itself must be reaped as well.
// Process con còn sót khi claude tự thoát cũng phải được dọn.
func TestExitReapsLeftoverChildren(t *testing.T) {
	r := start(t, nil)
	_ = r.Send("child")
	pid, err := strconv.Atoi(resultText(nextResult(t, r)))
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Send("crash")
	drain(r)
	deadline := time.Now().Add(3 * time.Second)
	for !errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
		if time.Now().After(deadline) {
			t.Fatalf("child %d survived its agent", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
