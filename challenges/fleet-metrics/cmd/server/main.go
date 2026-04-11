// Command server is the entry point for the fleet-metrics HTTP service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/domherve/fleet-metrics/internal/api"
	"github.com/domherve/fleet-metrics/internal/config"
	"github.com/domherve/fleet-metrics/internal/device"
	"github.com/domherve/fleet-metrics/internal/service"
	"github.com/domherve/fleet-metrics/internal/storage/memory"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := config.Load()

	deviceIDs, err := device.LoadFromCSV(cfg.DevicesCSVPath)
	if err != nil {
		logger.Error("failed to load devices", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("devices loaded", slog.Int("count", len(deviceIDs)))

	store := memory.NewStore(deviceIDs)
	svc := service.New(store, cfg.HeartbeatInterval)
	router := api.NewRouter(svc, logger)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", slog.String("addr", addr))
		serverErr <- srv.ListenAndServe()
	}()

	// Wait for shutdown signal or server error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case sig := <-quit:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("server stopped")
}
