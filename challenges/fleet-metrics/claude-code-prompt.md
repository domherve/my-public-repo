# Fleet Management Metrics Service — Implementation Brief

## Context

You are implementing a Go microservice called **fleet-metrics** for a coding assessment.
The service collects statistics from edge devices: heartbeats (for uptime tracking) and
upload timing stats (for average upload time reporting).

Use **standard library only**. No ORMs, no config-file parsers, no third-party HTTP
frameworks. Routing uses Go 1.22's enhanced `net/http` `ServeMux`, which supports
method-prefixed patterns and `{path_param}` placeholders natively.

---

## Project layout

Create this exact structure:

```
fleet-metrics/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── device/
│   │   ├── loader.go
│   │   └── loader_test.go
│   ├── storage/
│   │   ├── store.go          ← interface only
│   │   └── memory/
│   │       ├── memory.go
│   │       └── memory_test.go
│   ├── service/
│   │   ├── metrics.go
│   │   └── metrics_test.go
│   └── api/
│       ├── router.go
│       ├── handler/
│       │   ├── heartbeat.go
│       │   ├── stats_write.go
│       │   ├── stats_read.go
│       │   └── handler_test.go
│       └── middleware/
│           └── logging.go
├── testdata/
│   └── devices.csv
├── go.mod
└── README.md
```

---

## Configuration (`internal/config/config.go`)

Populated from CLI flags with `os.Getenv` fallbacks. No config file.

```go
type Config struct {
    Port              int           // --port / PORT (default: 6733)
    DevicesCSVPath    string        // --devices / DEVICES_CSV (default: "devices.csv")
    HeartbeatInterval time.Duration // --heartbeat-interval / HEARTBEAT_INTERVAL (default: 60s)
}
```

Parse with `flag` package. Env vars are checked first; flags override them.

---

## Device loader (`internal/device/`)

### CSV format

Single-column CSV with a header row:

```
device_id
60-6b-44-84-dc-64
b4-45-52-a2-f1-3c
26-9a-66-01-33-83
18-b8-87-e7-1f-06
38-4e-73-e0-33-59
```

Values are MAC addresses (strings). The loader must:
- Use `encoding/csv` from stdlib.
- Skip the header row.
- Trim whitespace from each value.
- Return `[]string` (device IDs).
- Return a descriptive error if the file cannot be opened or is malformed.

### Test (`loader_test.go`)

Table-driven tests using files in `testdata/`:
- Valid CSV → correct slice.
- Missing file → error.
- Empty file (header only) → empty slice, no error.
- CSV with extra whitespace → trimmed correctly.

---

## Storage layer (`internal/storage/`)

### Interface (`store.go`)

```go
package storage

import (
    "errors"
    "time"
)

// ErrDeviceNotFound is returned when an operation targets an unknown device ID.
var ErrDeviceNotFound = errors.New("device not found")

type DeviceStats struct {
    HeartbeatCount    int64
    FirstHeartbeatAt  time.Time   // zero value means no heartbeat received yet
    TotalUploadTimeNs int64       // sum of all upload_time values in nanoseconds
    UploadCount       int64
}

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
```

### In-memory implementation (`storage/memory/memory.go`)

- Holds a `map[string]*deviceRecord` seeded from `[]string` of device IDs at construction.
- All exported methods are safe for concurrent use (`sync.RWMutex`).
- `RecordHeartbeat`: increments `HeartbeatCount`; sets `FirstHeartbeatAt` only if it is the zero value.
- `RecordUploadStat`: adds `uploadTimeNs` to `TotalUploadTimeNs`, increments `UploadCount`.
- `GetDeviceStats`: returns a **copy** of the stats struct (not a pointer to the live map value).

### Test (`memory_test.go`)

Table-driven tests covering:
- Unknown device → `ErrDeviceNotFound`.
- Heartbeat count increments correctly.
- `FirstHeartbeatAt` is set on first heartbeat and not overwritten on subsequent ones.
- Upload stat accumulates correctly.
- Concurrent reads/writes (use `t.Parallel()` and `sync.WaitGroup` to race-detect).

---

## Service layer (`internal/service/metrics.go`)

The service layer owns all business logic. It depends on `storage.DeviceStore` (the interface, not the concrete type).

```go
type MetricsService struct {
    store             storage.DeviceStore
    heartbeatInterval time.Duration
    now               func() time.Time  // injectable for testing; defaults to time.Now
}

func New(store storage.DeviceStore, heartbeatInterval time.Duration) *MetricsService

// RecordHeartbeat delegates to store.
func (s *MetricsService) RecordHeartbeat(deviceID string, sentAt time.Time) error

// RecordUploadStat delegates to store.
func (s *MetricsService) RecordUploadStat(deviceID string, sentAt time.Time, uploadTimeNs int64) error

// GetStats computes and returns the derived statistics for a device.
func (s *MetricsService) GetStats(deviceID string) (*StatsResult, error)
```

```go
type StatsResult struct {
    AvgUploadTime string  // Go duration string, e.g. "5m10s". "0s" if no uploads recorded.
    Uptime        float64 // percentage, e.g. 98.5. 0.0 if no heartbeats received.
}
```

### Uptime formula

```
expected = floor((now - firstHeartbeatAt) / heartbeatInterval) + 1
uptime%  = min(100.0, float64(heartbeatCount) / float64(expected) * 100)
```

If `firstHeartbeatAt` is zero (no heartbeats received), return `0.0`.

Cap at `100.0` to guard against bursts.

### Avg upload time formula

```
avg = TotalUploadTimeNs / UploadCount   (integer division)
```

Format using `time.Duration(avg).String()`. Return `"0s"` if `UploadCount == 0`.

### Test (`metrics_test.go`)

Table-driven tests. Inject a fake `now` function so time is deterministic:
- No heartbeats → uptime = 0.0, avg = "0s".
- Exactly 1 heartbeat, interval elapsed once → uptime = 100.0.
- 3 heartbeats received, 6 expected → uptime = 50.0.
- Heartbeat burst (more received than expected) → uptime capped at 100.0.
- Single upload stat → avg equals that duration formatted.
- Multiple upload stats → correct average.

Use a mock/stub implementation of `storage.DeviceStore` in this test file (do not import the memory package).

---

## API layer (`internal/api/`)

### Router (`router.go`)

```go
func NewRouter(svc *service.MetricsService, logger *slog.Logger) http.Handler
```

Register routes with `net/http.NewServeMux()` using Go 1.22 method+pattern syntax:

```go
mux := http.NewServeMux()
mux.HandleFunc("POST /api/v1/devices/{device_id}/heartbeat", h.Heartbeat)
mux.HandleFunc("POST /api/v1/devices/{device_id}/stats",     h.StatsWrite)
mux.HandleFunc("GET /api/v1/devices/{device_id}/stats",      h.StatsRead)
```

Extract path parameters in handlers with `r.PathValue("device_id")`.

Wrap with the logging middleware.

### Handlers

Each handler is a struct with a `*service.MetricsService` field and implements `http.Handler` (or is a plain `func(http.ResponseWriter, *http.Request)`). Keep handlers thin:
1. Extract and validate the request (path param + body).
2. Call the service method.
3. Map the result to an HTTP response.

#### `POST /devices/{device_id}/heartbeat` (`handler/heartbeat.go`)

Request body:
```json
{ "sent_at": "2024-01-15T10:30:00Z" }
```

- `sent_at` is required; return `400` with `{"msg": "..."}` if missing or unparseable.
- Call `svc.RecordHeartbeat(deviceID, sentAt)`.
- `ErrDeviceNotFound` → `404 {"msg": "device not found"}`.
- Success → `204 No Content`.

#### `POST /devices/{device_id}/stats` (`handler/stats_write.go`)

Request body:
```json
{ "sent_at": "2024-01-15T10:30:00Z", "upload_time": 5000000000 }
```

- Both fields required; `upload_time` must be a positive integer (nanoseconds).
- Same error mapping as heartbeat handler.
- Success → `204 No Content`.

#### `GET /devices/{device_id}/stats` (`handler/stats_read.go`)

Response (200):
```json
{ "avg_upload_time": "5m10s", "uptime": 98.999 }
```

- `ErrDeviceNotFound` → `404`.
- If device exists but has no data yet → `204 No Content` (as per spec).
- Otherwise → `200` with the JSON body.

The "no data" condition is: `HeartbeatCount == 0 && UploadCount == 0`.

#### Error helper

Add a shared `writeError(w, status, msg)` function (unexported, in a `handler/util.go` file) that writes `Content-Type: application/json` and the error body.

### Handler tests (`handler/handler_test.go`)

Use `net/http/httptest` (recorder + test server). Test the full stack:
heartbeat handler, stats write, stats read. Cover:
- Valid request → expected status code and body.
- Unknown device → 404.
- Malformed JSON → 400.
- Missing required field → 400.
- GET stats with no data → 204.
- GET stats with data → 200 and correct JSON fields present.

### Middleware (`middleware/logging.go`)

Simple `slog`-based request logger that wraps any `http.Handler`. Log: method, path, status code, duration. Use a `responseWriter` wrapper to capture the status code written.

---

## Entry point (`cmd/server/main.go`)

1. Parse `Config`.
2. Load devices from CSV → `[]string`.
3. Construct `memory.NewStore(deviceIDs)`.
4. Construct `service.New(store, cfg.HeartbeatInterval)`.
5. Construct router.
6. Start `http.Server` on `cfg.Port` with reasonable timeouts:
   - `ReadTimeout:  10s`
   - `WriteTimeout: 10s`
   - `IdleTimeout:  120s`
7. Listen for `SIGINT`/`SIGTERM` and perform a graceful shutdown (5 s context).
8. Log startup and shutdown events with `slog`.

---

## Testdata (`testdata/devices.csv`)

```
device_id
60-6b-44-84-dc-64
b4-45-52-a2-f1-3c
26-9a-66-01-33-83
18-b8-87-e7-1f-06
38-4e-73-e0-33-59
```

---

## `go.mod`

Module name: `github.com/YOUR_USERNAME/fleet-metrics`
Go version: **1.22** or later (required for `log/slog` and enhanced `ServeMux` pattern matching).
**Zero external dependencies.**

---

## README.md

Include:
- Brief description.
- Prerequisites (Go 1.22+).
- How to run: `go run ./cmd/server --devices testdata/devices.csv`
- How to test: `go test ./...`
- Configuration reference (all flags and env vars with defaults).
- Example curl commands for all three endpoints.
- Uptime formula explanation.

---

## Coding standards to follow

- All exported identifiers must have godoc comments.
- No `panic` in production paths; always return errors.
- Errors are wrapped with `fmt.Errorf("context: %w", err)` for stack traceability.
- Use `context.Context` as the first parameter on functions that could block (http handlers already receive it via `r.Context()`).
- `go vet ./...` and `go test ./...` must pass with zero warnings.
- Table-driven tests use `t.Run(tc.name, ...)` so failures are identifiable.
- No `time.Sleep` in tests; use the injectable `now` function pattern instead.

---

## Sequence diagram (for reference)

```
Device                   fleet-metrics service
  |                              |
  |--- POST /heartbeat --------->|
  |                         RecordHeartbeat()
  |                         store.RecordHeartbeat()
  |<-- 204 No Content -----------|
  |                              |
  |--- POST /stats ------------->|
  |    { upload_time: 5e9 }  RecordUploadStat()
  |                         store.RecordUploadStat()
  |<-- 204 No Content -----------|
  |                              |
  |--- GET /stats -------------->|
  |                         GetStats()
  |                         store.GetDeviceStats()
  |                         compute uptime + avg
  |<-- 200 { uptime, avg } ------|
```
