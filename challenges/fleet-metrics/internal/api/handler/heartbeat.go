// Package handler contains HTTP handler functions for the fleet-metrics API.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/domherve/fleet-metrics/internal/service"
	"github.com/domherve/fleet-metrics/internal/storage"
)

type heartbeatRequest struct {
	SentAt *time.Time `json:"sent_at"`
}

// Heartbeat handles POST /api/v1/devices/{device_id}/heartbeat.
// It records a heartbeat event for the given device.
func Heartbeat(svc *service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.PathValue("device_id")

		var req heartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.SentAt == nil {
			writeError(w, http.StatusBadRequest, "sent_at is required")
			return
		}

		if err := svc.RecordHeartbeat(deviceID, *req.SentAt); err != nil {
			if errors.Is(err, storage.ErrDeviceNotFound) {
				writeError(w, http.StatusNotFound, "device not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
