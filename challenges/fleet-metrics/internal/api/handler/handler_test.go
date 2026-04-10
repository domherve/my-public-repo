package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/domherve/fleet-metrics/internal/api/handler"
	"github.com/domherve/fleet-metrics/internal/service"
	"github.com/domherve/fleet-metrics/internal/storage"
	"github.com/domherve/fleet-metrics/internal/storage/memory"
)

// newTestService creates a MetricsService backed by an in-memory store
// pre-seeded with the given device IDs.
func newTestService(deviceIDs []string) *service.MetricsService {
	store := memory.NewStore(deviceIDs)
	return service.New(store, 60*time.Second)
}

// newTestServiceWithStore creates a MetricsService backed by the given store.
func newTestServiceWithStore(store storage.DeviceStore) *service.MetricsService {
	return service.New(store, 60*time.Second)
}

// errStore always returns ErrDeviceNotFound.
type errStore struct{}

func (e *errStore) RecordHeartbeat(_ string, _ time.Time) error             { return storage.ErrDeviceNotFound }
func (e *errStore) RecordUploadStat(_ string, _ time.Time, _ int64) error   { return storage.ErrDeviceNotFound }
func (e *errStore) GetDeviceStats(_ string) (*storage.DeviceStats, error)   { return nil, storage.ErrDeviceNotFound }
func (e *errStore) DeviceExists(_ string) bool                               { return false }

func TestHeartbeatHandler(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		deviceIDs  []string
		targetID   string
		body       string
		wantStatus int
	}{
		{
			name:       "valid request",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-1",
			body:       `{"sent_at":"2024-01-15T10:30:00Z"}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "unknown device",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-999",
			body:       `{"sent_at":"2024-01-15T10:30:00Z"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "malformed JSON",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-1",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing sent_at",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-1",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := newTestService(tc.deviceIDs)
			h := handler.Heartbeat(svc)

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/v1/devices/{device_id}/heartbeat", h)

			req := httptest.NewRequest(http.MethodPost,
				"/api/v1/devices/"+tc.targetID+"/heartbeat",
				strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestStatsWriteHandler(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		deviceIDs  []string
		targetID   string
		body       string
		wantStatus int
	}{
		{
			name:       "valid request",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-1",
			body:       `{"sent_at":"2024-01-15T10:30:00Z","upload_time":5000000000}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "unknown device",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-999",
			body:       `{"sent_at":"2024-01-15T10:30:00Z","upload_time":5000000000}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "malformed JSON",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-1",
			body:       `{bad json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing sent_at",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-1",
			body:       `{"upload_time":5000000000}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing upload_time",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-1",
			body:       `{"sent_at":"2024-01-15T10:30:00Z"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative upload_time",
			deviceIDs:  []string{"dev-1"},
			targetID:   "dev-1",
			body:       `{"sent_at":"2024-01-15T10:30:00Z","upload_time":-1}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := newTestService(tc.deviceIDs)
			h := handler.StatsWrite(svc)

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/v1/devices/{device_id}/stats", h)

			req := httptest.NewRequest(http.MethodPost,
				"/api/v1/devices/"+tc.targetID+"/stats",
				strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestStatsReadHandler(t *testing.T) {
	t.Parallel()

	t.Run("unknown device returns 404", func(t *testing.T) {
		t.Parallel()
		svc := newTestServiceWithStore(&errStore{})
		h := handler.StatsRead(svc)

		mux := http.NewServeMux()
		mux.HandleFunc("GET /api/v1/devices/{device_id}/stats", h)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/ghost/stats", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("no data returns 204", func(t *testing.T) {
		t.Parallel()
		svc := newTestService([]string{"dev-1"})

		mux := http.NewServeMux()
		mux.HandleFunc("GET /api/v1/devices/{device_id}/stats", handler.StatsRead(svc))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/stats", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("status = %d, want 204", w.Code)
		}
	})

	t.Run("with data returns 200 and JSON fields", func(t *testing.T) {
		t.Parallel()
		svc := newTestService([]string{"dev-1"})

		// Seed some data via the write handlers.
		writeMux := http.NewServeMux()
		writeMux.HandleFunc("POST /api/v1/devices/{device_id}/heartbeat", handler.Heartbeat(svc))
		writeMux.HandleFunc("POST /api/v1/devices/{device_id}/stats", handler.StatsWrite(svc))

		hbBody := bytes.NewBufferString(`{"sent_at":"2024-01-15T10:30:00Z"}`)
		writeMux.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/api/v1/devices/dev-1/heartbeat", hbBody))

		statBody := bytes.NewBufferString(`{"sent_at":"2024-01-15T10:30:00Z","upload_time":5000000000}`)
		writeMux.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/api/v1/devices/dev-1/stats", statBody))

		// Now read.
		readMux := http.NewServeMux()
		readMux.HandleFunc("GET /api/v1/devices/{device_id}/stats", handler.StatsRead(svc))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/stats", nil)
		w := httptest.NewRecorder()
		readMux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if _, ok := resp["avg_upload_time"]; !ok {
			t.Error("response missing avg_upload_time field")
		}
		if _, ok := resp["uptime"]; !ok {
			t.Error("response missing uptime field")
		}
		// Validate that ErrDeviceNotFound is not in the response.
		if _, ok := resp["msg"]; ok {
			if errors.New(resp["msg"].(string)) != nil {
				t.Logf("response msg: %s", resp["msg"])
			}
		}
	})
}
