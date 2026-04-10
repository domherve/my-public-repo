package device_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/domherve/fleet-metrics/internal/device"
)

func TestLoadFromCSV(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		content string // written to a temp file; empty string means missing file
		want    []string
		wantErr bool
	}{
		{
			name: "valid CSV",
			content: "device_id\n60-6b-44-84-dc-64\nb4-45-52-a2-f1-3c\n26-9a-66-01-33-83\n",
			want:    []string{"60-6b-44-84-dc-64", "b4-45-52-a2-f1-3c", "26-9a-66-01-33-83"},
		},
		{
			name:    "missing file",
			wantErr: true,
		},
		{
			name:    "header only",
			content: "device_id\n",
			want:    []string{},
		},
		{
			name:    "extra whitespace",
			content: "device_id\n  60-6b-44-84-dc-64  \n  b4-45-52-a2-f1-3c  \n",
			want:    []string{"60-6b-44-84-dc-64", "b4-45-52-a2-f1-3c"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var path string
			if tc.content != "" {
				dir := t.TempDir()
				path = filepath.Join(dir, "devices.csv")
				if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
					t.Fatalf("setup: write temp file: %v", err)
				}
			} else {
				path = filepath.Join(t.TempDir(), "nonexistent.csv")
			}

			got, err := device.LoadFromCSV(path)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: got %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
