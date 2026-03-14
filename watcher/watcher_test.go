package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	if w.Events == nil {
		t.Fatal("Events channel is nil")
	}
	if w.Errors == nil {
		t.Fatal("Errors channel is nil")
	}
}

func TestWatcherDetectsCreatedFile(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	defer w.Stop()

	dir := t.TempDir()
	if err := w.AddFolder(dir); err != nil {
		t.Fatalf("AddFolder() error: %v", err)
	}

	w.Start()

	target := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(target, []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	select {
	case got := <-w.Events:
		if got == "" {
			t.Fatal("received empty path")
		}
	case err := <-w.Errors:
		t.Fatalf("watcher error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for create event")
	}
}
