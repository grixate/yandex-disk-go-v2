package yadisk

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDiskInfoLargeNumbers(t *testing.T) {
	body := []byte(`{"max_file_size":53687091200,"total_space":319975063552,"used_space":26157681270,"revision":1649182091142479}`)
	var disk DiskInfo
	if err := json.Unmarshal(body, &disk); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if disk.MaxFileSize != 53687091200 {
		t.Fatalf("max_file_size = %d", disk.MaxFileSize)
	}
	if disk.Revision != 1649182091142479 {
		t.Fatalf("revision = %d", disk.Revision)
	}
}

func TestTimestampParseAndRaw(t *testing.T) {
	var ts Timestamp
	if err := json.Unmarshal([]byte(`"2024-03-01T12:01:02Z"`), &ts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !ts.Valid || ts.Time.IsZero() {
		t.Fatalf("expected parsed timestamp")
	}

	var unknown Timestamp
	if err := json.Unmarshal([]byte(`"not-a-time"`), &unknown); err != nil {
		t.Fatalf("unmarshal unknown: %v", err)
	}
	if unknown.Valid {
		t.Fatalf("unexpected valid timestamp")
	}
	if unknown.Raw != "not-a-time" {
		t.Fatalf("raw = %q", unknown.Raw)
	}

	unknown.Time = time.Unix(0, 0)
	unknown.Valid = true
	out, err := json.Marshal(unknown)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) == "\"\"" {
		t.Fatalf("unexpected empty marshal")
	}
}
