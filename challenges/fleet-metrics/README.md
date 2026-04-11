# fleet-metrics

A Go microservice that collects and reports statistics from edge devices: heartbeat events (for uptime tracking) and upload timing stats (for average upload time reporting).

Built with Go 1.22 standard library only - no external dependencies.

---

## TL;DR

Running the project in a development environment (with Go 1.22+ installed) from this directory, using the sample device data provided:

```sh
go run ./cmd/server --devices testdata/devices.csv
```

Then run the device simulator and check the results reported.

More discussion about this submission can be found [here](submission/DISCUSSION.md), and more extensive build and run instructions can be found in this README down below.


---

## Prerequisites

- **Go 1.22 or later** (required for `log/slog` and enhanced `net/http` ServeMux pattern matching)

Verify your version:

```sh
go version
```

---

## Build

```sh
go build -o fleet-metrics ./cmd/server
```

Or run directly without building a binary:

```sh
go run ./cmd/server --devices testdata/devices.csv
```

---

## Run

```sh
go run ./cmd/server --devices testdata/devices.csv
```

The server logs to stdout in JSON format and listens on port `6733` by default.

---

## Test

```sh
go test ./...
```

To include the race detector (recommended):

```sh
go test -race ./...
```

---

## Configuration

All configuration has defaults and can be set via CLI flags or environment variables. **CLI flags take precedence over env vars.**

| Flag                   | Env var              | Default        | Description                                      |
|------------------------|----------------------|----------------|--------------------------------------------------|
| `--port`               | `PORT`               | `6733`         | TCP port the HTTP server listens on              |
| `--devices`            | `DEVICES_CSV`        | `devices.csv`  | Path to the CSV file containing device IDs       |
| `--heartbeat-interval` | `HEARTBEAT_INTERVAL` | `60s`          | Expected interval between consecutive heartbeats |

### Examples

Using flags:

```sh
go run ./cmd/server --port 8080 --devices ./testdata/devices.csv --heartbeat-interval 30s
```

Using environment variables:

```sh
PORT=8080 DEVICES_CSV=./testdata/devices.csv HEARTBEAT_INTERVAL=30s go run ./cmd/server
```

---

## API

Base path: `/api/v1/devices/{device_id}`

### POST `/api/v1/devices/{device_id}/heartbeat`

Record a heartbeat from a device.

**Request body:**
```json
{ "sent_at": "2024-01-15T10:30:00Z" }
```

**Responses:**
- `204 No Content` — success
- `400 Bad Request` — missing or invalid `sent_at`
- `404 Not Found` — device ID not registered

**Example:**
```sh
curl -s -o /dev/null -w "%{http_code}" \
  -X POST http://localhost:6733/api/v1/devices/60-6b-44-84-dc-64/heartbeat \
  -H 'Content-Type: application/json' \
  -d '{"sent_at":"2024-01-15T10:30:00Z"}'
```

---

### POST `/api/v1/devices/{device_id}/stats`

Record an upload timing stat from a device.

**Request body:**
```json
{ "sent_at": "2024-01-15T10:30:00Z", "upload_time": 5000000000 }
```

`upload_time` is in **nanoseconds** and must be a positive integer.

**Responses:**
- `204 No Content` — success
- `400 Bad Request` — missing/invalid fields or non-positive `upload_time`
- `404 Not Found` — device ID not registered

**Example:**
```sh
curl -s -o /dev/null -w "%{http_code}" \
  -X POST http://localhost:6733/api/v1/devices/60-6b-44-84-dc-64/stats \
  -H 'Content-Type: application/json' \
  -d '{"sent_at":"2024-01-15T10:30:00Z","upload_time":5000000000}'
```

---

### GET `/api/v1/devices/{device_id}/stats`

Retrieve computed statistics for a device.

**Responses:**
- `200 OK` — device has data
- `204 No Content` — device exists but has no recorded data yet
- `404 Not Found` — device ID not registered

**Response body (200):**
```json
{ "avg_upload_time": "5s", "uptime": 98.5 }
```

- `avg_upload_time` — mean upload duration as a Go duration string (e.g. `"5s"`, `"1m30s"`)
- `uptime` — percentage of expected heartbeats received (0.0–100.0)

**Example:**
```sh
curl -s http://localhost:6733/api/v1/devices/60-6b-44-84-dc-64/stats | jq .
```

---

## Uptime formula

Uptime is the percentage of expected heartbeats that were received:

```
expected = floor((now - first_heartbeat_at) / heartbeat_interval) + 1
uptime%  = min(100.0, heartbeat_count / expected × 100)
```

- `now` is the server's current time at the moment of the query.
- `first_heartbeat_at` is the timestamp of the very first heartbeat received for the device.
- `heartbeat_interval` is configurable (default: 60 seconds).
- The result is capped at `100.0` to handle burst scenarios where a device sends more heartbeats than expected.
- Returns `0.0` if no heartbeats have been received.

---

## Project layout

```
fleet-metrics/
├── cmd/
│   └── server/
│       └── main.go             # Entry point: config, wiring, HTTP server lifecycle
├── internal/
│   ├── config/
│   │   └── config.go           # CLI flag + env var configuration
│   ├── device/
│   │   ├── loader.go           # CSV device ID loader
│   │   └── loader_test.go
│   ├── storage/
│   │   ├── store.go            # DeviceStore interface + DeviceStats type
│   │   └── memory/
│   │       ├── memory.go       # Thread-safe in-memory implementation
│   │       └── memory_test.go
│   ├── service/
│   │   ├── metrics.go          # Business logic: uptime + avg upload time computation
│   │   └── metrics_test.go
│   └── api/
│       ├── router.go           # Route registration
│       ├── handler/
│       │   ├── heartbeat.go    # POST /heartbeat
│       │   ├── stats_write.go  # POST /stats
│       │   ├── stats_read.go   # GET /stats
│       │   ├── util.go         # Shared writeError helper
│       │   └── handler_test.go
│       └── middleware/
│           └── logging.go      # slog request logger
├── testdata/
│   └── devices.csv             # Sample device list used in tests and local runs
└── go.mod
```


### Packages

```
  ┌─────────────────────────┬─────────────────────────────────────────┬─────────────────────────────────────────────┐
  │         Package         │                 File(s)                 │                   Purpose                   │
  ├─────────────────────────┼─────────────────────────────────────────┼─────────────────────────────────────────────┤
  │ internal/config         │ config.go                               │ CLI flags + env var parsing (--port,        │
  │                         │                                         │ --devices, --heartbeat-interval)            │
  ├─────────────────────────┼─────────────────────────────────────────┼─────────────────────────────────────────────┤
  │ internal/device         │ loader.go, loader_test.go               │ CSV device ID loader with table-driven      │
  │                         │                                         │ tests                                       │
  ├─────────────────────────┼─────────────────────────────────────────┼─────────────────────────────────────────────┤
  │ internal/storage        │ store.go                                │ DeviceStore interface + ErrDeviceNotFound   │
  ├─────────────────────────┼─────────────────────────────────────────┼─────────────────────────────────────────────┤
  │ internal/storage/memory │ memory.go, memory_test.go               │ Thread-safe in-memory store with            │
  │                         │                                         │ sync.RWMutex                                │
  ├─────────────────────────┼─────────────────────────────────────────┼─────────────────────────────────────────────┤
  │ internal/service        │ metrics.go, metrics_test.go             │ Business logic: uptime % and avg upload     │
  │                         │                                         │ time; injectable clock for tests            │
  ├─────────────────────────┼─────────────────────────────────────────┼─────────────────────────────────────────────┤
  │ internal/api            │ router.go                               │ Route wiring with Go 1.22 method+pattern    │
  │                         │                                         │ ServeMux                                    │
  ├─────────────────────────┼─────────────────────────────────────────┼─────────────────────────────────────────────┤
  │ internal/api/handler    │ heartbeat.go, stats_write.go,           │ Thin HTTP handlers + httptest-based tests   │
  │                         │ stats_read.go, util.go, handler_test.go │                                             │
  ├─────────────────────────┼─────────────────────────────────────────┼─────────────────────────────────────────────┤
  │ internal/api/middleware │ logging.go                              │ slog-based request logger wrapping any      │
  │                         │                                         │ http.Handler                                │
  ├─────────────────────────┼─────────────────────────────────────────┼─────────────────────────────────────────────┤
  │ cmd/server              │ main.go                                 │ Startup, wiring, graceful shutdown on       │
  │                         │                                         │ SIGINT/SIGTERM                              │
  └─────────────────────────┴─────────────────────────────────────────┴─────────────────────────────────────────────┘

  Key design decisions

  - SetNow exposed on MetricsService for deterministic time injection in service tests — no time.Sleep anywhere
  - GetDeviceStats returns a copy of the stats struct to prevent data races
  - stats_read returns 204 when a device exists but has no recorded data yet (both HeartbeatCount == 0 and UploadCount
  == 0)
  - Zero external dependencies — stdlib only

```