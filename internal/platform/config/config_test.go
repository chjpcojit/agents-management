package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/chjpcojit/agents-management/internal/platform/config"
)

func env(m map[string]string) config.LookupFunc {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func TestDefaults(t *testing.T) {
	cfg, err := config.Load(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != "127.0.0.1:8787" || cfg.ApprovalTimeout != 5*time.Minute || cfg.AllowNonLoopback {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestRejectsNonLoopbackByDefault(t *testing.T) {
	_, err := config.Load(env(map[string]string{config.EnvHTTPAddr: "0.0.0.0:8787"}))
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("want loopback error, got %v", err)
	}
	if _, err := config.Load(env(map[string]string{config.EnvHTTPAddr: "0.0.0.0:8787", config.EnvAllowNonLoopback: "true"})); err != nil {
		t.Fatalf("opt-in should allow, got %v", err)
	}
}

func TestReportsAllErrorsTogether(t *testing.T) {
	_, err := config.Load(env(map[string]string{
		config.EnvApprovalTimeout:  "soon",
		config.EnvShutdownTimeout:  "1ms",
		config.EnvLogLevel:         "loud",
		config.EnvAllowNonLoopback: "maybe",
	}))
	if err == nil {
		t.Fatal("want error")
	}
	for _, key := range []string{config.EnvApprovalTimeout, config.EnvShutdownTimeout, config.EnvLogLevel, config.EnvAllowNonLoopback} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error does not mention %s: %v", key, err)
		}
	}
}
