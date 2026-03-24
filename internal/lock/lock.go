// Package lock provides single-instance locking for daemon ownership.
package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrAlreadyLocked = errors.New("instance already locked by another process")
)

// SingleInstanceLock ensures only one process owns the daemon at a time.
type SingleInstanceLock interface {
	Lock() error
	Unlock() error
	IsLocked() bool
}

// fileLock implements SingleInstanceLock using file-based locking.
type fileLock struct {
	lockPath string
	locked   bool
}

// NewSingleInstanceLock creates a new single-instance lock.
func NewSingleInstanceLock(lockDir string) (SingleInstanceLock, error) {
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	lockPath := filepath.Join(lockDir, "tidyyy.lock")
	return &fileLock{lockPath: lockPath, locked: false}, nil
}

// Lock acquires the lock; returns error if already locked.
func (fl *fileLock) Lock() error {
	if fl.locked {
		return nil
	}

	if _, err := os.Stat(fl.lockPath); err == nil {
		return ErrAlreadyLocked
	}

	f, err := os.Create(fl.lockPath)
	if err != nil {
		return fmt.Errorf("create lock file: %w", err)
	}
	_ = f.Close()

	fl.locked = true
	return nil
}

// Unlock releases the lock; safe to call multiple times.
func (fl *fileLock) Unlock() error {
	if !fl.locked {
		return nil
	}

	if err := os.Remove(fl.lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove lock file: %w", err)
	}

	fl.locked = false
	return nil
}

// IsLocked returns true if lock is currently held.
func (fl *fileLock) IsLocked() bool {
	return fl.locked
}
