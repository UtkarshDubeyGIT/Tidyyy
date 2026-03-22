package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tidyyy/internal/extractor"
	"github.com/tidyyy/internal/monitor"
	"github.com/tidyyy/internal/namer"
	"github.com/tidyyy/internal/triage"
	"github.com/tidyyy/internal/watcher"
)

type deduper struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

func newDeduper(ttl time.Duration) *deduper {
	return &deduper{seen: map[string]time.Time{}, ttl: ttl}
}

func (d *deduper) shouldProcess(path string) bool {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()

	// Clean up old entries periodically to prevent memory leaks
	if len(d.seen) > 1000 {
		for k, v := range d.seen {
			if now.Sub(v) > d.ttl*10 {
				delete(d.seen, k)
			}
		}
	}

	if t, ok := d.seen[path]; ok {
		if now.Sub(t) < d.ttl {
			return false
		}
	}
	d.seen[path] = now
	return true
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}

func main() {
	godotenv.Load() // Ignore error, file might not exist

	watchDirsRaw := getEnv("WATCH_DIRS", "")
	popplerBin := getEnv("POPPLER_BIN", "")
	tesseractBin := getEnv("TESSERACT_BIN", "")
	llamaCLIPath := getEnv("LLAMA_CLI_PATH", "llama-cli")
	modelPath := getEnv("MODEL_PATH", namer.DefaultModelPath)
	timeoutSec := getEnvInt("TIMEOUT_SEC", 25)
	pdfPageLimit := getEnvInt("PDF_PAGE_LIMIT", 8)
	queueDepth := getEnvInt("QUEUE_DEPTH", 64)
	cloudEnabled := getEnvBool("CLOUD_ENABLED", false)
	cloudBaseURL := getEnv("CLOUD_BASE_URL", "")
	cloudAPIKey := getEnv("CLOUD_API_KEY", "")
	cloudModel := getEnv("CLOUD_MODEL", "")
	skipRenameOnInvalid := getEnvBool("SKIP_RENAME_ON_INVALID", false)
	ramLogFile := getEnv("RAM_LOG_FILE", "./log/ram_usage.txt")
	ramLogIntervalSec := getEnvInt("RAM_LOG_INTERVAL_SEC", 2)
	dedupTTLSec := getEnvInt("DEDUP_TTL_SEC", 300)  // 5 minutes default

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	watchDirs, err := resolveWatchDirs(watchDirsRaw)
	if err != nil {
		logger.Error("invalid watch folders", "err", err)
		os.Exit(1)
	}
	if len(watchDirs) == 0 {
		logger.Error("no watch folder configured", "hint", "use --watch '/path/a,/path/b'")
		os.Exit(1)
	}

	extractorSvc := extractor.New(extractor.Config{
		PDFPageLimit:   pdfPageLimit,
		PopplerBin:     popplerBin,
		TesseractBin:   tesseractBin,
		CommandTimeout: time.Duration(timeoutSec) * time.Second,
	}, logger)

	namerSvc := namer.New(namer.Config{
		ModelPath:    modelPath,
		LlamaCLIPath: llamaCLIPath,
		EnableCloud:  cloudEnabled,
		CloudBaseURL: cloudBaseURL,
		CloudAPIKey:  cloudAPIKey,
		CloudModel:   cloudModel,
		UseFallback:  !skipRenameOnInvalid,
	}, logger)

	w, err := watcher.New()
	if err != nil {
		logger.Error("watcher init failed", "err", err)
		os.Exit(1)
	}
	defer w.Stop()

	for _, dir := range watchDirs {
		if err := w.AddFolder(dir); err != nil {
			logger.Error("failed to watch folder", "folder", dir, "err", err)
			os.Exit(1)
		}
	}

	tr := triage.New(queueDepth, logger)
	defer tr.Close()
	d := newDeduper(time.Duration(dedupTTLSec) * time.Second)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ramLogger := monitor.NewRAMLogger(ramLogFile, time.Duration(ramLogIntervalSec)*time.Second, logger)
	if err := ramLogger.Start(ctx); err != nil {
		logger.Error("failed to start ram logger", "err", err)
		os.Exit(1)
	}

	w.Start()
	logger.Info("tidyyy started", "watch_folders", strings.Join(watchDirs, ", "))

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case path, ok := <-w.Events:
				if !ok {
					return
				}
				if !d.shouldProcess(path) {
					continue
				}
				if _, err := tr.Accept(ctx, path); err != nil {
					logger.Error("triage accept failed", "path", path, "err", err)
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				logger.Error("watcher error", "err", err)
			}
		}
	}()

	go func() {
		for job := range tr.Queue() {
			processFile(ctx, logger, extractorSvc, namerSvc, job.Path)
		}
	}()

	<-ctx.Done()
	logger.Info("tidyyy stopped")
}

func processFile(ctx context.Context, logger *slog.Logger, extractorSvc *extractor.Service, namerSvc *namer.Service, path string) {
	fileCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	content, err := extractorSvc.ExtractPath(fileCtx, path)
	if err != nil {
		if errors.Is(err, extractor.ErrUnsupported) || errors.Is(err, extractor.ErrTooShort) || errors.Is(err, extractor.ErrTooLarge) {
			logger.Info("file skipped", "path", path, "reason", err.Error())
			return
		}
		logger.Error("extraction failed", "path", path, "err", err)
		return
	}

	slug, source, err := namerSvc.GenerateSlug(fileCtx, content.CleanText)
	if err != nil {
		logger.Error("name generation failed", "path", path, "err", err)
		return
	}

	logger.Info("name generated",
		"path", path,
		"source", content.Source,
		"namer", source,
		"slug", slug,
	)

	ext := filepath.Ext(path)
	if filepath.Base(path) == slug+ext {
		logger.Info("file already canonical", "path", path, "slug", slug)
		return
	}

	renamedPath, err := renameWithConflict(path, slug)
	if err != nil {
		logger.Error("rename failed", "path", path, "slug", slug, "err", err)
		return
	}

	logger.Info("file renamed", "old_path", path, "new_path", renamedPath)
	fmt.Printf("%s -> %s\n", filepath.Base(path), filepath.Base(renamedPath))
}

func renameWithConflict(path string, slug string) (string, error) {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)

	baseName := slug + ext
	candidate := filepath.Join(dir, baseName)
	if candidate == path {
		return path, nil
	}

	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(path, candidate); err != nil {
			return "", err
		}
		return candidate, nil
	}

	for i := 2; i < 1000; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s-%d%s", slug, i, ext))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			if err := os.Rename(path, candidate); err != nil {
				return "", err
			}
			return candidate, nil
		}
	}

	return "", fmt.Errorf("unable to find free filename for slug %q", slug)
}

func resolveWatchDirs(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		raw = filepath.Join(home, "Downloads")
	}

	parts := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", abs, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", abs)
		}
		if _, exists := seen[abs]; exists {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	return out, nil
}
