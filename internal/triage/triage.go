// Package triage implements F-02: File Triage.
//
// On a new-file detection event from the folder watcher (F-01), Triage:
//  1. Determines the file type by MIME sniffing (not just extension).
//  2. Silently skips files that are not in the target set.
//  3. Enqueues eligible files onto a bounded, concurrent-safe channel for
//     the Content Extraction stage (F-03) to consume.
package triage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// TargetExtensions is the canonical allow-list from the PRD (Section 3, In Scope v1).
// Extension check is a fast pre-filter; MIME sniffing is the authoritative gate.
var TargetExtensions = map[string]struct{}{
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".webp": {},
	".pdf":  {},
}

// These are checked after the extension pre-filter to avoid unnecessary disk reads.
var TargetMIMETypes = map[string]string{
	"image/png":       "png",
	"image/jpeg":      "jpeg",
	"image/jpg":       "jpg",
	"image/webp":      "webp",
	"application/pdf": "pdf",
}

// FileJob is a unit of work placed on the queue for F-03 (Content Extraction).
type FileJob struct {
	Path     string // Absolute path to the file.
	MIMEType string // Detected MIME type (e.g. "image/png").
	Label    string // Human-readable label (e.g. "png").
}

// Triager filters incoming file-system events and enqueues eligible files.
type Triager struct {
	queue  chan FileJob
	logger *slog.Logger
	once   sync.Once
	cancel context.CancelFunc
}

// New creates a Triager with a bounded internal queue.
// queueDepth controls the maximum number of pending FileJobs before Accept blocks.
// A depth of 64 is a sensible default for a background daemon.
func New(queueDepth int, logger *slog.Logger) *Triager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Triager{
		queue:  make(chan FileJob, queueDepth),
		logger: logger,
	}
}

// Queue returns the read-only channel that downstream stages (F-03) consume from.
func (t *Triager) Queue() <-chan FileJob {
	return t.queue
}

// Accept is the entry-point called by the folder watcher (F-01) for each
// CREATE / WRITE event. It is safe to call from multiple goroutines.
//
// The call is non-blocking: if the internal queue is full it logs a warning
// and drops the job rather than stalling the watcher goroutine.
//
// Returns (true, nil) when the file was successfully queued.
// Returns (false, nil) when the file was silently skipped (non-target type).
// Returns (false, err) when something unexpected prevented triage.
func (t *Triager) Accept(ctx context.Context, path string) (queued bool, err error) {
	// ── 1. Extension pre-filter (cheap, no disk I/O beyond stat) ──────────────
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := TargetExtensions[ext]; !ok {
		t.logger.Debug("triage: skipped — extension not in target set",
			slog.String("path", path),
			slog.String("ext", ext),
		)
		return false, nil
	}

	// ── 2. Confirm the file exists and is a regular file ─────────────────────
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("triage: stat %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		t.logger.Debug("triage: skipped — not a regular file", slog.String("path", path))
		return false, nil
	}
	if info.Size() == 0 {
		t.logger.Debug("triage: skipped — zero-byte file", slog.String("path", path))
		return false, nil
	}

	// ── 3. MIME sniff (reads first 512 bytes, as per net/http.DetectContentType) ──
	mimeType, err := detectMIME(path)
	if err != nil {
		return false, fmt.Errorf("triage: mime detect %q: %w", path, err)
	}

	label, ok := TargetMIMETypes[mimeType]
	if !ok {
		t.logger.Debug("triage: skipped — MIME not in target set",
			slog.String("path", path),
			slog.String("mime", mimeType),
		)
		return false, nil
	}

	job := FileJob{
		Path:     path,
		MIMEType: mimeType,
		Label:    label,
	}

	// ── 4. Non-blocking enqueue ───────────────────────────────────────────────
	select {
	case t.queue <- job:
		t.logger.Info("triage: queued",
			slog.String("path", path),
			slog.String("mime", mimeType),
		)
		return true, nil
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		// Queue full — drop and warn rather than block the watcher goroutine.
		t.logger.Warn("triage: queue full, dropping job — consider increasing queue depth",
			slog.String("path", path),
		)
		return false, nil
	}
}

// Close drains in-flight jobs and shuts down the queue channel.
// It is safe to call Close multiple times; subsequent calls are no-ops.
func (t *Triager) Close() {
	t.once.Do(func() {
		close(t.queue)
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// detectMIME reads the first 512 bytes of a file and returns its MIME type.
// Using net/http.DetectContentType mirrors the standard library approach and
// avoids any CGO or external dependency.
func detectMIME(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	if n == 0 {
		return "application/octet-stream", nil
	}

	return http.DetectContentType(buf[:n]), nil
}
