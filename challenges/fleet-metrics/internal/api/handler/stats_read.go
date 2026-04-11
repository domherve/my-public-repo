package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/domherve/fleet-metrics/internal/service"
	"github.com/domherve/fleet-metrics/internal/storage"
)

type statsReadResponse struct {
	AvgUploadTime string  `json:"avg_upload_time"`
	Uptime        float64 `json:"uptime"`
}

// StatsRead handles GET /api/v1/devices/{device_id}/stats.
// Returns computed uptime and average upload time for the given device.
// Returns 204 No Content when the device exists but has no recorded data.
func StatsRead(svc *service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.PathValue("device_id")

		result, err := svc.GetStats(deviceID)
		if err != nil {
			if errors.Is(err, storage.ErrDeviceNotFound) {
				writeError(w, http.StatusNotFound, "device not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		// No data recorded yet.
		if result.Uptime == 0.0 && result.AvgUploadTime == "0s" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(statsReadResponse{
			AvgUploadTime: result.AvgUploadTime,
			Uptime:        result.Uptime,
		})
	}
}
