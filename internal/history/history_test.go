package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileRecorder_RecordPreRename_AppendsJSONL(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "rename_history.jsonl")

	r := NewFileRecorder(logPath)
	if err := r.RecordPreRename("/tmp/a.png", "/tmp/sales-report.png"); err != nil {
		t.Fatalf("RecordPreRename error: %v", err)
	}

	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	if !s.Scan() {
		t.Fatal("expected one history line")
	}
	line := s.Bytes()
	var e Entry
	if err := json.Unmarshal(line, &e); err != nil {
		t.Fatalf("invalid json line: %v", err)
	}
	if e.OldPath != "/tmp/a.png" || e.NewPath != "/tmp/sales-report.png" {
		t.Fatalf("unexpected history entry: %+v", e)
	}
	if e.Timestamp == "" {
		t.Fatal("expected non-empty timestamp")
	}
}
