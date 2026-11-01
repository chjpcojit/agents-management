// Package config loads and validates process configuration from environment variables.
// Package config đọc và kiểm tra cấu hình process từ biến môi trường.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"
)

// Config is the validated runtime configuration of the orchestrator.
// Config là cấu hình runtime đã được kiểm tra của orchestrator.
type Config struct {
	HTTPAddr         string
	AllowNonLoopback bool
	LogLevel         slog.Level
	ApprovalTimeout  time.Duration
	ShutdownTimeout  time.Duration
}

// Environment variable names. Documented in README.md.
// Tên các biến môi trường. Có mô tả trong README.md.
const (
	EnvHTTPAddr         = "ORCH_HTTP_ADDR"
	EnvAllowNonLoopback = "ORCH_ALLOW_NON_LOOPBACK"
	EnvLogLevel         = "ORCH_LOG_LEVEL"
	EnvApprovalTimeout  = "ORCH_APPROVAL_TIMEOUT"
	EnvShutdownTimeout  = "ORCH_SHUTDOWN_TIMEOUT"
)

// LookupFunc matches os.LookupEnv so tests can supply a fake environment.
// LookupFunc có cùng chữ ký với os.LookupEnv để test truyền môi trường giả.
type LookupFunc func(key string) (string, bool)

// FromEnv loads the configuration from the process environment.
// FromEnv đọc cấu hình từ môi trường của process.
func FromEnv() (Config, error) { return Load(os.LookupEnv) }

// Load reads every variable, applies defaults and reports all problems together.
// Load đọc mọi biến, áp default và báo tất cả lỗi cùng một lúc.
func Load(lookup LookupFunc) (Config, error) {
	var errs []error
	get := func(key, def string) string {
		if v, ok := lookup(key); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		return def
	}
	dur := func(key, def string, minimum time.Duration) time.Duration {
		raw := get(key, def)
		d, err := time.ParseDuration(raw)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %q is not a duration", key, raw))
			return 0
		}
		if d < minimum {
			errs = append(errs, fmt.Errorf("%s: must be at least %s, got %s", key, minimum, d))
		}
		return d
	}

	cfg := Config{
		HTTPAddr:        get(EnvHTTPAddr, "127.0.0.1:8787"),
		ApprovalTimeout: dur(EnvApprovalTimeout, "5m", 10*time.Second),
		ShutdownTimeout: dur(EnvShutdownTimeout, "15s", time.Second),
	}

	switch v := strings.ToLower(get(EnvAllowNonLoopback, "false")); v {
	case "true", "1":
		cfg.AllowNonLoopback = true
	case "false", "0":
	default:
		errs = append(errs, fmt.Errorf("%s: %q is not a boolean", EnvAllowNonLoopback, v))
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(get(EnvLogLevel, "info"))); err != nil {
		errs = append(errs, fmt.Errorf("%s: %w", EnvLogLevel, err))
	}

	if err := checkAddr(cfg.HTTPAddr, cfg.AllowNonLoopback); err != nil {
		errs = append(errs, fmt.Errorf("%s: %w", EnvHTTPAddr, err))
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

// checkAddr enforces a loopback bind by default: the orchestrator controls agents with shell
// access, so exposing it beyond the machine needs an explicit opt-in (e.g. behind Tailscale).
//
// checkAddr mặc định bắt buộc bind loopback: orchestrator điều khiển agent có quyền shell,
// nên mở ra ngoài máy phải bật rõ ràng (ví dụ khi đứng sau Tailscale).
func checkAddr(addr string, allowNonLoopback bool) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%q is not host:port: %w", addr, err)
	}
	if port == "" {
		return fmt.Errorf("%q has no port", addr)
	}
	if allowNonLoopback {
		return nil
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("%q is not a loopback address; set %s=true to allow", addr, EnvAllowNonLoopback)
	}
	return nil
}
