// Package api wires together HTTP handlers and middleware for the fleet-metrics service.
package api

import (
	"log/slog"
	"net/http"

	"github.com/domherve/fleet-metrics/internal/api/handler"
	"github.com/domherve/fleet-metrics/internal/api/middleware"
	"github.com/domherve/fleet-metrics/internal/service"
)

// NewRouter constructs and returns the HTTP handler for the fleet-metrics API.
// Routes are registered using Go 1.22's method-prefixed ServeMux pattern syntax.
func NewRouter(svc *service.MetricsService, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/devices/{device_id}/heartbeat", handler.Heartbeat(svc))
	mux.HandleFunc("POST /api/v1/devices/{device_id}/stats", handler.StatsWrite(svc))
	mux.HandleFunc("GET /api/v1/devices/{device_id}/stats", handler.StatsRead(svc))

	return middleware.Logging(logger, mux)
}
