package renamer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// PreRenameRecorder stores old->new records before the filesystem rename.
type PreRenameRecorder interface {
	RecordPreRename(oldPath, newPath string) error
}

// Service performs in-directory atomic renames and conflict resolution.
//
// F-05 Atomic Rename:
//   - We only rename within the same directory, so os.Rename is atomic on
//     supported local filesystems.
//   - We persist a pre-rename history record before renaming.
//
// F-06 Conflict Resolution:
//   - If slug.ext already exists, append -2, -3, ... up to -999.
type Service struct {
	recorder PreRenameRecorder
}

func New(recorder PreRenameRecorder) *Service {
	return &Service{recorder: recorder}
}

func (s *Service) RenameWithConflict(path string, slug string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if slug == "" {
		return "", fmt.Errorf("empty slug")
	}

	dir := filepath.Dir(path)
	ext := filepath.Ext(path)

	baseName := slug + ext
	candidate := filepath.Join(dir, baseName)
	if candidate == path {
		return path, nil
	}

	if exists, err := fileExists(candidate); err != nil {
		return "", err
	} else if !exists {
		if err := s.recordBeforeRename(path, candidate); err != nil {
			return "", err
		}
		if err := os.Rename(path, candidate); err != nil {
			return "", err
		}
		return candidate, nil
	}

	for i := 2; i < 1000; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s-%d%s", slug, i, ext))
		exists, err := fileExists(candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			if err := s.recordBeforeRename(path, candidate); err != nil {
				return "", err
			}
			if err := os.Rename(path, candidate); err != nil {
				return "", err
			}
			return candidate, nil
		}
	}

	return "", fmt.Errorf("unable to find free filename for slug %q", slug)
}

func (s *Service) recordBeforeRename(oldPath, newPath string) error {
	if s.recorder == nil {
		return nil
	}
	if err := s.recorder.RecordPreRename(oldPath, newPath); err != nil {
		return fmt.Errorf("record pre-rename history: %w", err)
	}
	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
