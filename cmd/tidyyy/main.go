package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tidyyy/internal/config"
	"github.com/tidyyy/internal/extractor"
	"github.com/tidyyy/internal/lock"
	"github.com/tidyyy/internal/manager"
	"github.com/tidyyy/internal/monitor"
	"github.com/tidyyy/internal/namer"
	"github.com/tidyyy/internal/renamer"
	"github.com/tidyyy/internal/triage"
	"github.com/tidyyy/internal/ui"
	"github.com/tidyyy/internal/watcher"
)

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

func hasArg(flag string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return false
}

func main() {
	godotenv.Load() // Ignore error, file might not exist

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	settingsMode := hasArg("--settings")

	// Single-instance lock to ensure only one daemon is running
	instanceLock, err := lock.NewSingleInstanceLock(os.TempDir())
	if err != nil {
		logger.Error("failed to create instance lock", "err", err)
		os.Exit(1)
	}
	if err := instanceLock.Lock(); err != nil {
		logger.Error("tidyyy is already running", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = instanceLock.Unlock()
	}()

	// Load application configuration
	appCfg, err := config.Load()
	if err != nil {
		logger.Warn("failed to load config, using defaults", "err", err)
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			home = ""
		}
		appCfg = config.Default(home)
	}

	// Resolve watch directories from config
	watchDirsRaw := getEnv("WATCH_DIRS", strings.Join(appCfg.WatchDirs, ","))
	watchDirs, err := resolveWatchDirs(watchDirsRaw)
	if err != nil {
		if settingsMode {
			logger.Warn("watch folder validation failed; opening settings without auto-start", "err", err)
			watchDirs = nil
		} else {
			logger.Error("invalid watch folders", "err", err)
			os.Exit(1)
		}
	}

	// Initialize services
	popplerBin := getEnv("POPPLER_BIN", "")
	tesseractBin := getEnv("TESSERACT_BIN", "")
	llamaCLIPath := getEnv("LLAMA_CLI_PATH", "llama-cli")
	modelPath := getEnv("MODEL_PATH", namer.DefaultModelPath)
	timeoutSec := getEnvInt("TIMEOUT_SEC", 25)
	pdfPageLimit := getEnvInt("PDF_PAGE_LIMIT", 8)
	queueDepth := getEnvInt("QUEUE_DEPTH", 64)
	cloudEnabled := getEnvBool("CLOUD_ENABLED", appCfg.CloudEnabled)
	cloudBaseURL := getEnv("CLOUD_BASE_URL", "")
	cloudAPIKey := getEnv("CLOUD_API_KEY", appCfg.CloudAPIKey)
	cloudModel := getEnv("CLOUD_MODEL", "")
	maxNameWords := getEnvInt("MAX_NAME_WORDS", appCfg.MaxNameWords)
	skipRenameOnInvalid := getEnvBool("SKIP_RENAME_ON_INVALID", false)
	ramLogFile := getEnv("RAM_LOG_FILE", "./logs/ram_usage.txt")
	ramLogIntervalSec := getEnvInt("RAM_LOG_INTERVAL_SEC", 2)
	dedupTTLSec := getEnvInt("DEDUP_TTL_SEC", 300) // 5 minutes default

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
		MaxWords:     maxNameWords,
	}, logger)

	// Note: history is not used by daemon manager directly; it's integrated into renamer
	renamerSvc := renamer.New(nil) // history recorder passed as nil; renamer handles internally

	// Initialize watcher
	w, err := watcher.New()
	if err != nil {
		logger.Error("watcher init failed", "err", err)
		os.Exit(1)
	}
	defer w.Stop()

	// Add watch directories to watcher
	for _, dir := range watchDirs {
		if err := w.AddFolder(dir); err != nil {
			logger.Error("failed to watch folder", "folder", dir, "err", err)
			os.Exit(1)
		}
	}

	// Initialize triage
	tr := triage.New(queueDepth, logger)
	defer tr.Close()

	// Initialize monitor and RAM logger
	sysMonitor := monitor.NewMonitor(logger)
	ramLogger := monitor.NewRAMLogger(ramLogFile, time.Duration(ramLogIntervalSec)*time.Second, logger)

	// Create daemon manager
	daemonMgr := manager.NewDaemonManager(
		w,
		tr,
		extractorSvc,
		namerSvc,
		renamerSvc,
		sysMonitor,
		ramLogger,
		manager.Config{
			DedupTTL:        time.Duration(dedupTTLSec) * time.Second,
			ShutdownTimeout: 5 * time.Second,
			Logger:          logger,
		},
	)

	// Application context for daemon lifecycle
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Auto-start daemon if watch directories are valid
	if len(watchDirs) > 0 {
		logger.Info("auto-starting daemon with watch folders", "count", len(watchDirs))
		if err := daemonMgr.Start(ctx); err != nil {
			logger.Error("failed to start daemon", "err", err)
			// Don't exit; user can try restarting from UI
		}
	} else {
		logger.Info("no watch folders configured; daemon not started")
	}

	if settingsMode {
		settingsHooks := &ui.SettingsLifecycleHooks{
			IsDaemonRunning: func() bool {
				return daemonMgr.IsRunning()
			},
			StartDaemon: func() error {
				if daemonMgr.IsRunning() {
					return nil
				}

				latestCfg, err := config.Load()
				if err != nil {
					return fmt.Errorf("load latest config: %w", err)
				}
				resolved, err := resolveWatchDirs(strings.Join(latestCfg.WatchDirs, ","))
				if err != nil {
					return fmt.Errorf("validate watch dirs: %w", err)
				}
				if len(resolved) == 0 {
					return fmt.Errorf("no valid watch directories configured")
				}

				for _, dir := range resolved {
					if err := w.AddFolder(dir); err != nil {
						logger.Warn("watch folder add skipped", "folder", dir, "err", err)
					}
				}
				return daemonMgr.Start(ctx)
			},
			RestartDaemon: func() error {
				latestCfg, err := config.Load()
				if err != nil {
					return fmt.Errorf("load latest config: %w", err)
				}
				resolved, err := resolveWatchDirs(strings.Join(latestCfg.WatchDirs, ","))
				if err != nil {
					return fmt.Errorf("validate watch dirs: %w", err)
				}

				for _, dir := range resolved {
					if err := w.AddFolder(dir); err != nil {
						logger.Warn("watch folder add skipped", "folder", dir, "err", err)
					}
				}
				return daemonMgr.Restart(ctx)
			},
		}

		trayHooks := &ui.TrayLifecycleHooks{
			IsDaemonRunning: func() bool {
				return daemonMgr.IsRunning()
			},
			IsPaused: func() bool {
				return daemonMgr.IsPaused()
			},
			SetPaused: func(paused bool) error {
				return daemonMgr.SetPaused(paused)
			},
			RestartDaemon: settingsHooks.RestartDaemon,
			Quit: func() error {
				if daemonMgr.IsRunning() {
					if err := daemonMgr.Stop(); err != nil {
						return err
					}
				}
				cancel()
				return nil
			},
		}

		if err := ui.RunTrayRuntime(logger, settingsHooks, trayHooks); err != nil {
			logger.Error("tray runtime failed", "err", err)
			if daemonMgr.IsRunning() {
				if stopErr := daemonMgr.Stop(); stopErr != nil {
					logger.Error("failed to stop daemon", "err", stopErr)
				}
			}
		}
		return
	}

	// Handle graceful shutdown on signal
	<-ctx.Done()
	logger.Info("shutdown signal received")

	// Stop daemon
	if daemonMgr.IsRunning() {
		if err := daemonMgr.Stop(); err != nil {
			logger.Error("failed to stop daemon", "err", err)
		}
	}

	logger.Info("tidyyy stopped")
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
