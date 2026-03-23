package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents one append-only pre-rename record.
type Entry struct {
	OldPath   string `json:"old_path"`
	NewPath   string `json:"new_path"`
	Timestamp string `json:"timestamp"`
}

// FileRecorder persists pre-rename records as JSONL.
// This keeps F-05 guarantees (write before rename) without coupling to DB code.
type FileRecorder struct {
	path string
	mu   sync.Mutex
}

func NewFileRecorder(path string) *FileRecorder {
	if path == "" {
		path = "./logs/rename_history.jsonl"
	}
	return &FileRecorder{path: path}
}

func (r *FileRecorder) RecordPreRename(oldPath, newPath string) error {
	if oldPath == "" || newPath == "" {
		return fmt.Errorf("invalid history record: empty path")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("create history directory: %w", err)
	}

	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open history file: %w", err)
	}
	defer f.Close()

	entry := Entry{
		OldPath:   oldPath,
		NewPath:   newPath,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal history record: %w", err)
	}

	if _, err := f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write history record: %w", err)
	}
	return nil
}
