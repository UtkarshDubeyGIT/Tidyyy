package lock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSingleInstanceLockLifecycle(t *testing.T) {
	dir := t.TempDir()

	l, err := NewSingleInstanceLock(dir)
	if err != nil {
		t.Fatalf("new lock failed: %v", err)
	}

	if l.IsLocked() {
		t.Fatalf("expected unlocked initially")
	}

	if err := l.Lock(); err != nil {
		t.Fatalf("lock failed: %v", err)
	}
	if !l.IsLocked() {
		t.Fatalf("expected locked after Lock")
	}

	if err := l.Unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}
	if l.IsLocked() {
		t.Fatalf("expected unlocked after Unlock")
	}
}

func TestSingleInstanceLockBlocksSecondProcess(t *testing.T) {
	dir := t.TempDir()

	l1, err := NewSingleInstanceLock(dir)
	if err != nil {
		t.Fatalf("new lock 1 failed: %v", err)
	}
	l2, err := NewSingleInstanceLock(dir)
	if err != nil {
		t.Fatalf("new lock 2 failed: %v", err)
	}

	if err := l1.Lock(); err != nil {
		t.Fatalf("first lock failed: %v", err)
	}
	defer l1.Unlock()

	if err := l2.Lock(); !errors.Is(err, ErrAlreadyLocked) {
		t.Fatalf("expected ErrAlreadyLocked, got: %v", err)
	}
}

func TestSingleInstanceLockCreatesFile(t *testing.T) {
	dir := t.TempDir()

	l, err := NewSingleInstanceLock(dir)
	if err != nil {
		t.Fatalf("new lock failed: %v", err)
	}

	if err := l.Lock(); err != nil {
		t.Fatalf("lock failed: %v", err)
	}
	defer l.Unlock()

	p := filepath.Join(dir, "tidyyy.lock")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected lock file to exist: %v", err)
	}
}
