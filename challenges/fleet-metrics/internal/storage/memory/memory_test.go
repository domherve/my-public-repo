package memory_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/domherve/fleet-metrics/internal/storage"
	"github.com/domherve/fleet-metrics/internal/storage/memory"
)

func newTestStore() *memory.Store {
	return memory.NewStore([]string{"dev-1", "dev-2"})
}

func TestUnknownDevice(t *testing.T) {
	t.Parallel()
	s := newTestStore()

	t.Run("RecordHeartbeat", func(t *testing.T) {
		t.Parallel()
		err := s.RecordHeartbeat("unknown", time.Now())
		if !errors.Is(err, storage.ErrDeviceNotFound) {
			t.Errorf("got %v, want ErrDeviceNotFound", err)
		}
	})

	t.Run("RecordUploadStat", func(t *testing.T) {
		t.Parallel()
		err := s.RecordUploadStat("unknown", time.Now(), 1000)
		if !errors.Is(err, storage.ErrDeviceNotFound) {
			t.Errorf("got %v, want ErrDeviceNotFound", err)
		}
	})

	t.Run("GetDeviceStats", func(t *testing.T) {
		t.Parallel()
		_, err := s.GetDeviceStats("unknown")
		if !errors.Is(err, storage.ErrDeviceNotFound) {
			t.Errorf("got %v, want ErrDeviceNotFound", err)
		}
	})
}

func TestHeartbeatCount(t *testing.T) {
	t.Parallel()
	s := newTestStore()
	now := time.Now()

	for i := 0; i < 5; i++ {
		if err := s.RecordHeartbeat("dev-1", now); err != nil {
			t.Fatalf("RecordHeartbeat: %v", err)
		}
	}

	stats, err := s.GetDeviceStats("dev-1")
	if err != nil {
		t.Fatalf("GetDeviceStats: %v", err)
	}
	if stats.HeartbeatCount != 5 {
		t.Errorf("HeartbeatCount = %d, want 5", stats.HeartbeatCount)
	}
}

func TestFirstHeartbeatAtNotOverwritten(t *testing.T) {
	t.Parallel()
	s := newTestStore()

	first := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	second := first.Add(time.Minute)

	if err := s.RecordHeartbeat("dev-1", first); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordHeartbeat("dev-1", second); err != nil {
		t.Fatal(err)
	}

	stats, err := s.GetDeviceStats("dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if !stats.FirstHeartbeatAt.Equal(first) {
		t.Errorf("FirstHeartbeatAt = %v, want %v", stats.FirstHeartbeatAt, first)
	}
}

func TestUploadStatAccumulates(t *testing.T) {
	t.Parallel()
	s := newTestStore()
	now := time.Now()

	uploads := []int64{1_000_000, 2_000_000, 3_000_000}
	var total int64
	for _, u := range uploads {
		total += u
		if err := s.RecordUploadStat("dev-1", now, u); err != nil {
			t.Fatalf("RecordUploadStat: %v", err)
		}
	}

	stats, err := s.GetDeviceStats("dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalUploadTimeNs != total {
		t.Errorf("TotalUploadTimeNs = %d, want %d", stats.TotalUploadTimeNs, total)
	}
	if stats.UploadCount != int64(len(uploads)) {
		t.Errorf("UploadCount = %d, want %d", stats.UploadCount, len(uploads))
	}
}

func TestGetDeviceStatsReturnsCopy(t *testing.T) {
	t.Parallel()
	s := newTestStore()
	now := time.Now()

	if err := s.RecordHeartbeat("dev-1", now); err != nil {
		t.Fatal(err)
	}
	stats, err := s.GetDeviceStats("dev-1")
	if err != nil {
		t.Fatal(err)
	}
	// Mutate the returned copy.
	stats.HeartbeatCount = 9999

	// Verify the store is unaffected.
	stats2, err := s.GetDeviceStats("dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if stats2.HeartbeatCount == 9999 {
		t.Error("GetDeviceStats returned a pointer to live map value instead of a copy")
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	t.Parallel()
	s := newTestStore()
	now := time.Now()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = s.RecordHeartbeat("dev-1", now)
		}()
		go func() {
			defer wg.Done()
			_, _ = s.GetDeviceStats("dev-1")
		}()
	}

	wg.Wait()

	stats, err := s.GetDeviceStats("dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if stats.HeartbeatCount != goroutines {
		t.Errorf("HeartbeatCount = %d, want %d", stats.HeartbeatCount, goroutines)
	}
}
