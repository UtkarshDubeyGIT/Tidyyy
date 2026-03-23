package renamer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameWithConflict_RenamesWithoutConflict(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "IMG_1234.png")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	svc := New(nil)
	got, err := svc.RenameWithConflict(src, "sales-report")
	if err != nil {
		t.Fatalf("RenameWithConflict error: %v", err)
	}

	want := filepath.Join(dir, "sales-report.png")
	if got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected target to exist: %v", err)
	}
}

func TestRenameWithConflict_AppendsCounterOnCollision(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "IMG_1234.png")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	// Occupy the primary target name.
	if err := os.WriteFile(filepath.Join(dir, "sales-report.png"), []byte("y"), 0o644); err != nil {
		t.Fatalf("write colliding file: %v", err)
	}

	svc := New(nil)
	got, err := svc.RenameWithConflict(src, "sales-report")
	if err != nil {
		t.Fatalf("RenameWithConflict error: %v", err)
	}

	want := filepath.Join(dir, "sales-report-2.png")
	if got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected suffixed target to exist: %v", err)
	}
}

func TestRenameWithConflict_NoOpWhenAlreadyCanonical(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "sales-report.png")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	svc := New(nil)
	got, err := svc.RenameWithConflict(src, "sales-report")
	if err != nil {
		t.Fatalf("RenameWithConflict error: %v", err)
	}

	if got != src {
		t.Fatalf("expected no-op rename, got %q want %q", got, src)
	}
}

type failRecorder struct{}

func (f *failRecorder) RecordPreRename(oldPath, newPath string) error {
	return errors.New("history down")
}

func TestRenameWithConflict_AbortIfHistoryWriteFails(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "IMG_1234.png")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	svc := New(&failRecorder{})
	_, err := svc.RenameWithConflict(src, "sales-report")
	if err == nil {
		t.Fatal("expected error when history write fails")
	}

	// Ensure file was not renamed when pre-rename record failed.
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("expected source to still exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sales-report.png")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected target to not exist, got err=%v", err)
	}
}
