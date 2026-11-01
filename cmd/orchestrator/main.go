// Command orchestrator is the composition root: it loads config, wires adapters into use cases
// and runs the HTTP server until SIGINT or SIGTERM.
//
// Command orchestrator là composition root: đọc config, nối adapter vào use case
// và chạy HTTP server cho tới khi nhận SIGINT hoặc SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chjpcojit/agents-management/internal/adapter/repository/memory"
	"github.com/chjpcojit/agents-management/internal/adapter/system"
	"github.com/chjpcojit/agents-management/internal/delivery/httpapi"
	"github.com/chjpcojit/agents-management/internal/platform/config"
	"github.com/chjpcojit/agents-management/internal/usecase/approval"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	approvals, err := approval.NewService(
		memory.NewApprovalRepository(),
		system.Clock{},
		system.UUIDGenerator{},
		cfg.ApprovalTimeout,
		log.With(slog.String("component", "approval")),
	)
	if err != nil {
		return fmt.Errorf("wire approval service: %w", err)
	}

	// The sweeper enforces default-deny on approvals nobody answered.
	// Sweeper đảm bảo default-deny cho các approval không ai trả lời.
	sweeperDone := make(chan struct{})
	go func() {
		defer close(sweeperDone)
		runSweeper(ctx, approvals, log)
	}()

	srv := httpapi.NewServer(cfg.HTTPAddr, httpapi.NewHandler(log, nil), log)
	serveErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serveErr:
		if err != nil {
			stop()
			<-sweeperDone
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	stop()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	<-sweeperDone
	log.Info("stopped")
	return nil
}

func runSweeper(ctx context.Context, svc *approval.Service, log *slog.Logger) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := svc.ExpireDue(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Error("expire due approvals", slog.Any("error", err))
			}
		}
	}
}
