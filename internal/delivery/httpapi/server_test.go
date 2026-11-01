package httpapi_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chjpcojit/agents-management/internal/delivery/httpapi"
)

func TestHealthz(t *testing.T) {
	h := httpapi.NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestReadyz_FailingCheckHidesDetail(t *testing.T) {
	checks := map[string]httpapi.ReadinessCheck{
		"db": func(context.Context) error { return errors.New("dial tcp 10.0.0.5:5432: secret-host") },
	}
	h := httpapi.NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), checks)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"db"`) || strings.Contains(body, "secret-host") {
		t.Fatalf("body must name the check without internal detail: %s", body)
	}
}
