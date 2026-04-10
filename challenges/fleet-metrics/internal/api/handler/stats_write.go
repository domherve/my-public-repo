package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/domherve/fleet-metrics/internal/service"
	"github.com/domherve/fleet-metrics/internal/storage"
)

type statsWriteRequest struct {
	SentAt     *time.Time `json:"sent_at"`
	UploadTime *int64     `json:"upload_time"`
}

// StatsWrite handles POST /api/v1/devices/{device_id}/stats.
// It records an upload timing stat for the given device.
func StatsWrite(svc *service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.PathValue("device_id")

		var req statsWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.SentAt == nil {
			writeError(w, http.StatusBadRequest, "sent_at is required")
			return
		}
		if req.UploadTime == nil {
			writeError(w, http.StatusBadRequest, "upload_time is required")
			return
		}
		if *req.UploadTime <= 0 {
			writeError(w, http.StatusBadRequest, "upload_time must be a positive integer")
			return
		}

		if err := svc.RecordUploadStat(deviceID, *req.SentAt, *req.UploadTime); err != nil {
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
