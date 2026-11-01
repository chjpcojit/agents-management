// Package httpapi is the HTTP delivery layer: routing, request decoding and error mapping.
// It calls use cases only and holds no business rule.
//
// Package httpapi là delivery layer HTTP: routing, decode request và map lỗi.
// Chỉ gọi use case, không chứa business rule.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// ReadinessCheck reports whether one dependency is ready. A nil error means ready.
// ReadinessCheck cho biết một dependency đã sẵn sàng chưa. Lỗi nil nghĩa là sẵn sàng.
type ReadinessCheck func(ctx context.Context) error

// NewHandler builds the root handler with health endpoints.
// checks are named so the readiness body tells which dependency failed, without internal detail.
//
// NewHandler tạo handler gốc với các endpoint health.
// checks có tên để body readiness chỉ ra dependency nào lỗi mà không lộ chi tiết nội bộ.
func NewHandler(log *slog.Logger, checks map[string]ReadinessCheck) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, log, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		failed := make([]string, 0)
		for name, check := range checks {
			if err := check(ctx); err != nil {
				failed = append(failed, name)
				log.WarnContext(ctx, "readiness check failed", slog.String("check", name), slog.Any("error", err))
			}
		}
		if len(failed) > 0 {
			writeJSON(w, log, http.StatusServiceUnavailable, map[string]any{"status": "unavailable", "failed": failed})
			return
		}
		writeJSON(w, log, http.StatusOK, map[string]string{"status": "ready"})
	})
	return mux
}

// NewServer wraps the handler with conservative timeouts against slow clients.
// WriteTimeout stays 0 because streaming endpoints (SSE) will be long-lived.
//
// NewServer bọc handler với timeout chặt để chống client chậm.
// WriteTimeout để 0 vì các endpoint streaming (SSE) sẽ giữ kết nối lâu.
func NewServer(addr string, h http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}

func writeJSON(w http.ResponseWriter, log *slog.Logger, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Warn("write response body", slog.Any("error", err))
	}
}
