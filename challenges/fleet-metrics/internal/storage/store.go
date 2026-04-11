// Package storage defines the persistence interface for device metrics.
package storage

import (
	"errors"
	"time"
)

// ErrDeviceNotFound is returned when an operation targets an unknown device ID.
var ErrDeviceNotFound = errors.New("device not found")

// DeviceStats holds the raw accumulated stats for a single device.
type DeviceStats struct {
	// HeartbeatCount is the total number of heartbeats received.
	HeartbeatCount int64

	// FirstHeartbeatAt is the timestamp of the very first heartbeat.
	// The zero value indicates no heartbeat has been received yet.
	FirstHeartbeatAt time.Time

	// LastHeartbeatAt is the timestamp of the most recent heartbeat.
	// The zero value indicates no heartbeat has been received yet.
	LastHeartbeatAt time.Time

	// TotalUploadTimeNs is the sum of all upload_time values in nanoseconds.
	TotalUploadTimeNs int64

	// UploadCount is the number of upload stat records received.
	UploadCount int64
}

// DeviceStore is the persistence abstraction for device metrics.
type DeviceStore interface {
	// RecordHeartbeat records a heartbeat for the given device.
	// Returns ErrDeviceNotFound if the device ID is unknown.
	RecordHeartbeat(deviceID string, sentAt time.Time) error

	// RecordUploadStat records an upload duration for the given device.
	// Returns ErrDeviceNotFound if the device ID is unknown.
	RecordUploadStat(deviceID string, sentAt time.Time, uploadTimeNs int64) error

	// GetDeviceStats returns the raw stats for the given device.
	// Returns ErrDeviceNotFound if the device ID is unknown.
	GetDeviceStats(deviceID string) (*DeviceStats, error)

	// DeviceExists returns true if the device ID is registered.
	DeviceExists(deviceID string) bool
}
