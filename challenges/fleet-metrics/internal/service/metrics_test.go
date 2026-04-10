package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/domherve/fleet-metrics/internal/service"
	"github.com/domherve/fleet-metrics/internal/storage"
)

// fakeStore is a stub implementation of storage.DeviceStore used in tests.
type fakeStore struct {
	stats *storage.DeviceStats
	err   error
}

func (f *fakeStore) RecordHeartbeat(_ string, _ time.Time) error           { return f.err }
func (f *fakeStore) RecordUploadStat(_ string, _ time.Time, _ int64) error { return f.err }
func (f *fakeStore) DeviceExists(_ string) bool                            { return f.err == nil }
func (f *fakeStore) GetDeviceStats(_ string) (*storage.DeviceStats, error) {
	if f.err != nil {
		return nil, f.err
	}
	cp := *f.stats
	return &cp, nil
}

func TestGetStats_NoHeartbeats(t *testing.T) {
	t.Parallel()
	store := &fakeStore{stats: &storage.DeviceStats{}}
	svc := service.New(store, 60*time.Second)

	result, err := svc.GetStats("dev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Uptime != 0.0 {
		t.Errorf("Uptime = %f, want 0.0", result.Uptime)
	}
	if result.AvgUploadTime != "0s" {
		t.Errorf("AvgUploadTime = %q, want \"0s\"", result.AvgUploadTime)
	}
}

func TestGetStats_ExactlyOneHeartbeatIntervalElapsed(t *testing.T) {
	t.Parallel()
	interval := 60 * time.Second
	first := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	store := &fakeStore{stats: &storage.DeviceStats{
		HeartbeatCount:   1,
		FirstHeartbeatAt: first,
		LastHeartbeatAt:  first, // single heartbeat: first == last, zero elapsed → expected = 1
	}}
	svc := service.New(store, interval)

	result, err := svc.GetStats("dev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Uptime != 100.0 {
		t.Errorf("Uptime = %f, want 100.0", result.Uptime)
	}
}

func TestGetStats_HalfUptimeExpected(t *testing.T) {
	t.Parallel()
	interval := 60 * time.Second
	first := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	last := first.Add(6 * interval)

	store := &fakeStore{stats: &storage.DeviceStats{
		HeartbeatCount:   3,
		FirstHeartbeatAt: first,
		LastHeartbeatAt:  last,
	}}
	svc := service.New(store, interval)

	result, err := svc.GetStats("dev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 50.0
	if result.Uptime != want {
		t.Errorf("Uptime = %f, want %f", result.Uptime, want)
	}
}

func TestGetStats_UptimeCappedAt100(t *testing.T) {
	t.Parallel()
	interval := 60 * time.Second
	first := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Burst: 10 heartbeats received but only 1 expected (first == last → expected = 1).
	store := &fakeStore{stats: &storage.DeviceStats{
		HeartbeatCount:   10,
		FirstHeartbeatAt: first,
		LastHeartbeatAt:  first,
	}}
	svc := service.New(store, interval)

	result, err := svc.GetStats("dev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Uptime != 100.0 {
		t.Errorf("Uptime = %f, want 100.0 (capped)", result.Uptime)
	}
}

func TestGetStats_SingleUpload(t *testing.T) {
	t.Parallel()
	uploadNs := int64(5 * time.Minute)
	store := &fakeStore{stats: &storage.DeviceStats{
		TotalUploadTimeNs: uploadNs,
		UploadCount:       1,
	}}
	svc := service.New(store, 60*time.Second)

	result, err := svc.GetStats("dev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Duration(uploadNs).String()
	if result.AvgUploadTime != want {
		t.Errorf("AvgUploadTime = %q, want %q", result.AvgUploadTime, want)
	}
}

func TestGetStats_MultipleUploads(t *testing.T) {
	t.Parallel()
	// 3 uploads: 1s, 2s, 3s → avg = 2s
	store := &fakeStore{stats: &storage.DeviceStats{
		TotalUploadTimeNs: int64(6 * time.Second),
		UploadCount:       3,
	}}
	svc := service.New(store, 60*time.Second)

	result, err := svc.GetStats("dev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := (2 * time.Second).String()
	if result.AvgUploadTime != want {
		t.Errorf("AvgUploadTime = %q, want %q", result.AvgUploadTime, want)
	}
}

func TestGetStats_DeviceNotFound(t *testing.T) {
	t.Parallel()
	store := &fakeStore{err: storage.ErrDeviceNotFound}
	svc := service.New(store, 60*time.Second)

	_, err := svc.GetStats("ghost")
	if !errors.Is(err, storage.ErrDeviceNotFound) {
		t.Errorf("got %v, want ErrDeviceNotFound", err)
	}
}
