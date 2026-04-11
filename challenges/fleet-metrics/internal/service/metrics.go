// Package service implements business logic for the fleet-metrics service.
package service

import (
	"fmt"
	"math"
	"time"

	"github.com/domherve/fleet-metrics/internal/storage"
)

// StatsResult holds the computed, human-readable statistics for a device.
type StatsResult struct {
	// AvgUploadTime is the mean upload duration as a Go duration string (e.g. "5m10s").
	// Returns "0s" when no upload stats have been recorded.
	AvgUploadTime string `json:"avg_upload_time"`

	// Uptime is the percentage of expected heartbeats that were received (0.0–100.0).
	// Returns 0.0 when no heartbeats have been received.
	Uptime float64 `json:"uptime"`
}

// MetricsService owns all business logic for recording and querying device metrics.
type MetricsService struct {
	store             storage.DeviceStore
	heartbeatInterval time.Duration
	now               func() time.Time
}

// New constructs a MetricsService backed by the given store and heartbeat interval.
func New(store storage.DeviceStore, heartbeatInterval time.Duration) *MetricsService {
	return &MetricsService{
		store:             store,
		heartbeatInterval: heartbeatInterval,
		now:               time.Now,
	}
}

// SetNow overrides the clock function used for uptime computation.
// Intended for use in tests only.
func (s *MetricsService) SetNow(fn func() time.Time) {
	s.now = fn
}

// RecordHeartbeat delegates to the underlying store.
// Returns storage.ErrDeviceNotFound if the device ID is unknown.
func (s *MetricsService) RecordHeartbeat(deviceID string, sentAt time.Time) error {
	if err := s.store.RecordHeartbeat(deviceID, sentAt); err != nil {
		return fmt.Errorf("service.RecordHeartbeat: %w", err)
	}
	return nil
}

// RecordUploadStat delegates to the underlying store.
// Returns storage.ErrDeviceNotFound if the device ID is unknown.
func (s *MetricsService) RecordUploadStat(deviceID string, sentAt time.Time, uploadTimeNs int64) error {
	if err := s.store.RecordUploadStat(deviceID, sentAt, uploadTimeNs); err != nil {
		return fmt.Errorf("service.RecordUploadStat: %w", err)
	}
	return nil
}

// GetStats computes and returns the derived statistics for a device.
// Returns storage.ErrDeviceNotFound if the device ID is unknown.
func (s *MetricsService) GetStats(deviceID string) (*StatsResult, error) {
	stats, err := s.store.GetDeviceStats(deviceID)
	if err != nil {
		return nil, fmt.Errorf("service.GetStats: %w", err)
	}

	result := &StatsResult{
		AvgUploadTime: avgUploadTime(stats),
		Uptime:        s.uptime(stats),
	}
	return result, nil
}

// avgUploadTime computes the mean upload time as a duration string.
func avgUploadTime(stats *storage.DeviceStats) string {
	if stats.UploadCount == 0 {
		return "0s"
	}
	avg := stats.TotalUploadTimeNs / stats.UploadCount
	return time.Duration(avg).String()
}

// uptime computes the uptime percentage based on heartbeat history.
// The window is defined from the first to the last heartbeat timestamp,
// so that historical data is scored over the period the device was active.
func (s *MetricsService) uptime(stats *storage.DeviceStats) float64 {
	if stats.FirstHeartbeatAt.IsZero() {
		return 0.0
	}

	elapsed := stats.LastHeartbeatAt.Sub(stats.FirstHeartbeatAt)
	expected := math.Floor(float64(elapsed.Nanoseconds() / s.heartbeatInterval.Nanoseconds()))
	pct := float64(stats.HeartbeatCount) / expected * 100.0
	if pct > 100.0 {
		pct = 100.0
	}
	return pct
}
