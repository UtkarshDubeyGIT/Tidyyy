package monitor

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// RAMLogger appends periodic process memory snapshots to a text file.
type RAMLogger struct {
	path     string
	interval time.Duration
	logger   *slog.Logger
}

func NewRAMLogger(path string, interval time.Duration, logger *slog.Logger) *RAMLogger {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if path == "" {
		path = "./logs/ram_usage.txt"
	}
	return &RAMLogger{path: path, interval: interval, logger: logger}
}

func (r *RAMLogger) Start(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("create ram log directory: %w", err)
	}

	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open ram log file: %w", err)
	}

	w := bufio.NewWriter(f)
	info, err := f.Stat()
	if err == nil && info.Size() == 0 {
		if _, err := w.WriteString("timestamp,alloc_mb,sys_mb,heap_inuse_mb,heap_idle_mb,num_gc,goroutines\n"); err != nil {
			_ = f.Close()
			return fmt.Errorf("write ram log header: %w", err)
		}
		if err := w.Flush(); err != nil {
			_ = f.Close()
			return fmt.Errorf("flush ram log header: %w", err)
		}
	}

	r.logger.Info("ram logger started", "path", r.path, "interval", r.interval.String())

	go func() {
		defer func() {
			_ = w.Flush()
			_ = f.Close()
			r.logger.Info("ram logger stopped", "path", r.path)
		}()

		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		// Write an immediate sample so logs include startup memory.
		r.writeSample(w)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.writeSample(w)
			}
		}
	}()

	return nil
}

func (r *RAMLogger) writeSample(w *bufio.Writer) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	line := fmt.Sprintf(
		"%s,%.2f,%.2f,%.2f,%.2f,%d,%d\n",
		time.Now().Format(time.RFC3339),
		bytesToMB(m.Alloc),
		bytesToMB(m.Sys),
		bytesToMB(m.HeapInuse),
		bytesToMB(m.HeapIdle),
		m.NumGC,
		runtime.NumGoroutine(),
	)
	if _, err := w.WriteString(line); err != nil {
		r.logger.Error("ram logger write failed", "err", err)
		return
	}
	if err := w.Flush(); err != nil {
		r.logger.Error("ram logger flush failed", "err", err)
	}
}

func bytesToMB(v uint64) float64 {
	return float64(v) / (1024.0 * 1024.0)
}
