// Package config provides configuration parsing for the fleet-metrics server.
package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the server.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	// Controlled by --port flag or PORT env var (default: 6733).
	Port int

	// DevicesCSVPath is the path to the CSV file containing registered device IDs.
	// Controlled by --devices flag or DEVICES_CSV env var (default: "devices.csv").
	DevicesCSVPath string

	// HeartbeatInterval is the expected interval between consecutive heartbeats.
	// Used to compute uptime percentage.
	// Controlled by --heartbeat-interval flag or HEARTBEAT_INTERVAL env var (default: 60s).
	HeartbeatInterval time.Duration
}

// Load parses configuration from environment variables and CLI flags.
// Env vars are the baseline; flags override them when explicitly provided.
func Load() *Config {
	cfg := &Config{
		Port:              envInt("PORT", 6733),
		DevicesCSVPath:    envString("DEVICES_CSV", "devices.csv"),
		HeartbeatInterval: envDuration("HEARTBEAT_INTERVAL", 60*time.Second),
	}

	flag.IntVar(&cfg.Port, "port", cfg.Port, "TCP port to listen on (env: PORT)")
	flag.StringVar(&cfg.DevicesCSVPath, "devices", cfg.DevicesCSVPath, "path to devices CSV file (env: DEVICES_CSV)")
	flag.DurationVar(&cfg.HeartbeatInterval, "heartbeat-interval", cfg.HeartbeatInterval, "expected heartbeat interval (env: HEARTBEAT_INTERVAL)")

	flag.Parse()
	return cfg
}

func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
