// Package memory provides an in-memory implementation of storage.DeviceStore.
package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/domherve/fleet-metrics/internal/storage"
)

type deviceRecord struct {
	mu                sync.RWMutex
	heartbeatCount    int64
	firstHeartbeatAt  time.Time
	lastHeartbeatAt   time.Time
	totalUploadTimeNs int64
	uploadCount       int64
}

// Store is a thread-safe, in-memory implementation of storage.DeviceStore.
type Store struct {
	mu      sync.RWMutex
	devices map[string]*deviceRecord
}

// NewStore constructs a Store pre-seeded with the given device IDs.
func NewStore(deviceIDs []string) *Store {
	devices := make(map[string]*deviceRecord, len(deviceIDs))
	for _, id := range deviceIDs {
		devices[id] = &deviceRecord{}
	}
	return &Store{devices: devices}
}

// DeviceExists reports whether the given device ID is registered.
func (s *Store) DeviceExists(deviceID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.devices[deviceID]
	return ok
}

// RecordHeartbeat increments the heartbeat counter for the device.
// Sets FirstHeartbeatAt only on the first call.
// Returns storage.ErrDeviceNotFound for unknown device IDs.
func (s *Store) RecordHeartbeat(deviceID string, sentAt time.Time) error {
	rec, err := s.getRecord(deviceID)
	if err != nil {
		return err
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.heartbeatCount++
	if rec.firstHeartbeatAt.IsZero() {
		rec.firstHeartbeatAt = sentAt
	}
	rec.lastHeartbeatAt = sentAt
	return nil
}

// RecordUploadStat adds the upload duration to the device's running totals.
// Returns storage.ErrDeviceNotFound for unknown device IDs.
func (s *Store) RecordUploadStat(deviceID string, _ time.Time, uploadTimeNs int64) error {
	rec, err := s.getRecord(deviceID)
	if err != nil {
		return err
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.totalUploadTimeNs += uploadTimeNs
	rec.uploadCount++
	return nil
}

// GetDeviceStats returns a snapshot copy of the device's statistics.
// Returns storage.ErrDeviceNotFound for unknown device IDs.
func (s *Store) GetDeviceStats(deviceID string) (*storage.DeviceStats, error) {
	rec, err := s.getRecord(deviceID)
	if err != nil {
		return nil, err
	}

	rec.mu.RLock()
	defer rec.mu.RUnlock()
	return &storage.DeviceStats{
		HeartbeatCount:    rec.heartbeatCount,
		FirstHeartbeatAt:  rec.firstHeartbeatAt,
		LastHeartbeatAt:   rec.lastHeartbeatAt,
		TotalUploadTimeNs: rec.totalUploadTimeNs,
		UploadCount:       rec.uploadCount,
	}, nil
}

func (s *Store) getRecord(deviceID string) (*deviceRecord, error) {
	s.mu.RLock()
	rec, ok := s.devices[deviceID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", storage.ErrDeviceNotFound, deviceID)
	}
	return rec, nil
}
