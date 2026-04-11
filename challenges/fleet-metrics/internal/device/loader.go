// Package device provides utilities for loading registered device identifiers.
package device

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// LoadFromCSV reads a single-column CSV file and returns the list of device IDs.
// The file must have a header row (device_id) which is skipped.
// Whitespace is trimmed from each value.
// Returns a descriptive error if the file cannot be opened or is malformed.
func LoadFromCSV(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("device.LoadFromCSV: open %q: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = 1
	r.TrimLeadingSpace = true

	// Skip header row.
	if _, err := r.Read(); err != nil {
		if err == io.EOF {
			// File is empty — no header, treat as empty.
			return []string{}, nil
		}
		return nil, fmt.Errorf("device.LoadFromCSV: read header: %w", err)
	}

	var ids []string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("device.LoadFromCSV: read record: %w", err)
		}
		id := strings.TrimSpace(record[0])
		if id != "" {
			ids = append(ids, id)
		}
	}

	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}
