package triage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"github.com/tidyyy/internal/triage"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func newTriager(t *testing.T) *triage.Triager {
	t.Helper()
	tr := triage.New(64, nil)
	t.Cleanup(tr.Close)
	return tr
}

// writeFile creates a temp file with the given name and raw bytes.
func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return path
}

// Minimal valid PNG: 1×1 transparent pixel.
var minimalPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR length + "IHDR"
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // width=1, height=1
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // bit depth, color type…
	0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, // IDAT length + "IDAT"
	0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00, // deflate stream
	0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc, // …
	0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, // IEND length + "IEND"
	0x44, 0xae, 0x42, 0x60, 0x82, // IEND CRC
}

// Minimal PDF header (enough for MIME sniffing).
var minimalPDF = []byte("%PDF-1.4\n")

// ── tests ─────────────────────────────────────────────────────────────────────

func TestAccept_PNG_Queued(t *testing.T) {
	tr := newTriager(t)
	path := writeFile(t, "screenshot.png", minimalPNG)

	queued, err := tr.Accept(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !queued {
		t.Fatal("expected PNG to be queued")
	}

	select {
	case job := <-tr.Queue():
		if job.Path != path {
			t.Errorf("path: got %q, want %q", job.Path, path)
		}
		if job.MIMEType != "image/png" {
			t.Errorf("mime: got %q, want %q", job.MIMEType, "image/png")
		}
		if job.Label != "png" {
			t.Errorf("label: got %q, want %q", job.Label, "png")
		}
	default:
		t.Fatal("queue was empty after Accept returned true")
	}
}

func TestAccept_PDF_Queued(t *testing.T) {
	tr := newTriager(t)
	path := writeFile(t, "report.pdf", minimalPDF)

	queued, err := tr.Accept(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !queued {
		t.Fatal("expected PDF to be queued")
	}

	job := <-tr.Queue()
	if job.MIMEType != "application/pdf" {
		t.Errorf("mime: got %q, want %q", job.MIMEType, "application/pdf")
	}
}

func TestAccept_UnsupportedExtension_Skipped(t *testing.T) {
	tr := newTriager(t)
	path := writeFile(t, "video.mp4", []byte("fake-mp4-content"))

	queued, err := tr.Accept(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued {
		t.Fatal("expected .mp4 to be skipped, not queued")
	}
}

func TestAccept_PNGExtension_WrongMIME_Skipped(t *testing.T) {
	// File has .png extension but content is plain text — MIME sniff must
	// catch this and reject it.
	tr := newTriager(t)
	path := writeFile(t, "not-really.png", []byte("hello world"))

	queued, err := tr.Accept(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued {
		t.Fatal("expected file with wrong MIME to be skipped")
	}
}

func TestAccept_ZeroByte_Skipped(t *testing.T) {
	tr := newTriager(t)
	path := writeFile(t, "empty.png", []byte{})

	queued, err := tr.Accept(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queued {
		t.Fatal("expected zero-byte file to be skipped")
	}
}

func TestAccept_NonexistentFile_Error(t *testing.T) {
	tr := newTriager(t)

	_, err := tr.Accept(context.Background(), "/tmp/does-not-exist-tidyyy.png")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestAccept_ContextCancelled_Error(t *testing.T) {
	// Fill the queue so that the enqueue select hits the ctx.Done branch.
	tr := triage.New(1, nil)
	t.Cleanup(tr.Close)

	// Pre-fill the single-slot queue.
	path1 := writeFile(t, "first.png", minimalPNG)
	_, _ = tr.Accept(context.Background(), path1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	path2 := writeFile(t, "second.png", minimalPNG)
	_, err := tr.Accept(ctx, path2)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestAccept_QueueFull_DropsGracefully(t *testing.T) {
	// depth=1 so second job overflows silently (no error, no panic).
	tr := triage.New(1, nil)
	t.Cleanup(tr.Close)

	path1 := writeFile(t, "a.png", minimalPNG)
	path2 := writeFile(t, "b.png", minimalPNG)

	queued1, err1 := tr.Accept(context.Background(), path1)
	queued2, err2 := tr.Accept(context.Background(), path2)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v / %v", err1, err2)
	}
	if !queued1 {
		t.Error("first job should have been queued")
	}
	if queued2 {
		t.Error("second job should have been dropped (queue full), not queued")
	}
}

func TestAccept_ConcurrentSafety(t *testing.T) {
	tr := triage.New(256, nil)
	t.Cleanup(tr.Close)

	const workers = 20
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		go func() {
			path := writeFile(t, "concurrent.png", minimalPNG)
			_, err := tr.Accept(context.Background(), path)
			errCh <- err
		}()
	}

	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("worker error: %v", err)
		}
	}
}