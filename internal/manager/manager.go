// Package manager orchestrates daemon lifecycle, resource gating, and pause state.
package manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tidyyy/internal/extractor"
	"github.com/tidyyy/internal/monitor"
	"github.com/tidyyy/internal/namer"
	"github.com/tidyyy/internal/renamer"
	"github.com/tidyyy/internal/triage"
	"github.com/tidyyy/internal/watcher"
	"golang.org/x/sync/semaphore"
)

// DaemonManager orchestrates daemon lifecycle, resource gating, and pause state.
type DaemonManager interface {
	Start(ctx context.Context) error
	Stop() error
	Restart(ctx context.Context) error
	SetPaused(paused bool) error
	IsPaused() bool
	IsRunning() bool
	SubscribeMetrics() <-chan MetricsUpdate
	Status() string
}

// MetricsUpdate is re-exported from monitor package for convenience.
type MetricsUpdate = monitor.MetricsUpdate

// Errors returned by DaemonManager.
var (
	ErrAlreadyRunning = errors.New("daemon already running")
	ErrNotRunning     = errors.New("daemon not running")
	ErrPaused         = errors.New("daemon paused; file not queued")
	ErrInvalidContext = errors.New("context already canceled")
)

// Config holds configuration for daemon manager initialization.
type Config struct {
	DedupTTL        time.Duration
	ShutdownTimeout time.Duration
	Logger          *slog.Logger
}

// daemonManager implements DaemonManager interface.
type daemonManager struct {
	mu              sync.RWMutex
	running         bool
	ctx             context.Context
	cancel          context.CancelFunc
	parentCtx       context.Context
	shutdownTimeout time.Duration
	paused          bool
	throttleGate    atomic.Bool

	watcher       *watcher.Watcher
	triager       *triage.Triager
	extractorSvc  *extractor.Service
	namerSvc      *namer.Service
	renamerSvc    *renamer.Service
	monitor       monitor.Monitor
	ramLogger     *monitor.RAMLogger
	logger        *slog.Logger

	metricsC      chan MetricsUpdate
	processingSem *semaphore.Weighted
	deduper       *deduper
}

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

// NewDaemonManager creates a new daemon manager with dependencies.
func NewDaemonManager(
	w *watcher.Watcher,
	tr *triage.Triager,
	es *extractor.Service,
	ns *namer.Service,
	rs *renamer.Service,
	mon monitor.Monitor,
	rl *monitor.RAMLogger,
	cfg Config,
) DaemonManager {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.DedupTTL == 0 {
		cfg.DedupTTL = 5 * time.Minute
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 5 * time.Second
	}

	return &daemonManager{
		watcher:         w,
		triager:         tr,
		extractorSvc:    es,
		namerSvc:        ns,
		renamerSvc:      rs,
		monitor:         mon,
		ramLogger:       rl,
		logger:          cfg.Logger,
		metricsC:        make(chan MetricsUpdate, 1),
		processingSem:   semaphore.NewWeighted(2),
		deduper:         newDeduper(cfg.DedupTTL),
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

// Start initializes and launches all daemon components.
func (dm *daemonManager) Start(ctx context.Context) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.running {
		return ErrAlreadyRunning
	}

	if ctx.Err() != nil {
		return ErrInvalidContext
	}

	dm.parentCtx = ctx
	dm.ctx, dm.cancel = context.WithCancel(ctx)

	if dm.monitor != nil {
		if err := dm.monitor.Start(dm.ctx); err != nil {
			dm.cancel()
			return fmt.Errorf("monitor start: %w", err)
		}
		go dm.metricSubscriber()
	}

	if dm.ramLogger != nil {
		if err := dm.ramLogger.Start(dm.ctx); err != nil {
			dm.cancel()
			if dm.monitor != nil {
				dm.monitor.Stop()
			}
			return fmt.Errorf("ram logger start: %w", err)
		}
	}

	if dm.watcher != nil {
		dm.watcher.Start()
		dm.logger.Info("watcher started")
		go dm.watcherEventLoop()
	}

	go dm.jobProcessingLoop()

	dm.running = true
	dm.logger.Info("daemon started")
	return nil
}

// metricSubscriber reads from monitor and updates throttle gate.
func (dm *daemonManager) metricSubscriber() {
	if dm.monitor == nil {
		return
	}

	monitorC := dm.monitor.SubscribeMetrics()
	for {
		select {
		case <-dm.ctx.Done():
			return
		case update, ok := <-monitorC:
			if !ok {
				return
			}
			newThrottle := update.CPUPercent > 60 || update.BatteryLevel < 15
			dm.throttleGate.Store(newThrottle)

			dm.mu.RLock()
			paused := dm.paused
			dm.mu.RUnlock()

			update.PauseState = paused

			select {
			case dm.metricsC <- update:
			default:
			}
		}
	}
}

// watcherEventLoop: watcher.Events → triage.Accept()
func (dm *daemonManager) watcherEventLoop() {
	if dm.watcher == nil || dm.triager == nil {
		return
	}

	for {
		select {
		case <-dm.ctx.Done():
			dm.logger.Info("watcher event loop stopped")
			return
		case path, ok := <-dm.watcher.Events:
			if !ok {
				dm.logger.Info("watcher events channel closed")
				return
			}

			dm.logger.Info("file event received", "path", path)

			if dm.IsPaused() {
				dm.logger.Debug("event skipped: daemon paused", "path", path)
				continue
			}

			if !dm.deduper.shouldProcess(path) {
				dm.logger.Info("file deduplicated or already processed recently", "path", path)
				continue
			}

			dm.logger.Info("file accepted for triage", "path", path)
			if _, err := dm.triager.Accept(dm.ctx, path); err != nil {
				dm.logger.Error("triage accept failed", "path", path, "err", err)
			}
		case err, ok := <-dm.watcher.Errors:
			if !ok {
				dm.logger.Info("watcher errors channel closed")
				return
			}
			dm.logger.Error("watcher error", "err", err)
		}
	}
}

// jobProcessingLoop: triage.Queue() → processFile
func (dm *daemonManager) jobProcessingLoop() {
	if dm.triager == nil {
		return
	}

	for job := range dm.triager.Queue() {
		if dm.throttleGate.Load() {
			dm.logger.Info("throttled: backing off job", "path", job.Path)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if err := dm.processingSem.Acquire(dm.ctx, 1); err != nil {
			dm.logger.Info("job processing loop stopped")
			return
		}

		go func(job triage.FileJob) {
			defer dm.processingSem.Release(1)
			dm.processFile(dm.ctx, job)
		}(job)
	}
	dm.logger.Info("triage queue drained")
}

// processFile implements the file processing pipeline.
func (dm *daemonManager) processFile(ctx context.Context, job triage.FileJob) {
	fileCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	dm.logger.Info("processing file from queue", "path", job.Path)

	content, err := dm.extractorSvc.ExtractPath(fileCtx, job.Path)
	if err != nil {
		if errors.Is(err, extractor.ErrUnsupported) || errors.Is(err, extractor.ErrTooShort) || errors.Is(err, extractor.ErrTooLarge) {
			dm.logger.Info("file skipped", "path", job.Path, "reason", err.Error())
			return
		}
		dm.logger.Error("extraction failed", "path", job.Path, "err", err)
		return
	}

	slug, source, err := dm.namerSvc.GenerateSlug(fileCtx, content.CleanText)
	if err != nil {
		dm.logger.Error("name generation failed", "path", job.Path, "err", err)
		return
	}

	dm.logger.Info("name generated",
		"path", job.Path,
		"source", content.Source,
		"namer", source,
		"slug", slug,
	)

	ext := filepath.Ext(job.Path)
	if filepath.Base(job.Path) == slug+ext {
		dm.logger.Info("file already canonical", "path", job.Path, "slug", slug)
		return
	}

	renamedPath, err := dm.renamerSvc.RenameWithConflict(job.Path, slug)
	if err != nil {
		dm.logger.Error("rename failed", "path", job.Path, "slug", slug, "err", err)
		return
	}

	dm.logger.Info("file renamed", "old_path", job.Path, "new_path", renamedPath)
	fmt.Printf("%s -> %s\n", filepath.Base(job.Path), filepath.Base(renamedPath))
}

// Stop gracefully shuts down all daemon components with timeout.
func (dm *daemonManager) Stop() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !dm.running {
		return ErrNotRunning
	}

	dm.cancel()

	if dm.watcher != nil {
		dm.watcher.Stop()
	}

	if dm.triager != nil {
		dm.triager.Close()
	}

	drainCtx, cancel := context.WithTimeout(context.Background(), dm.shutdownTimeout)
	defer cancel()

	drainTicker := time.NewTicker(100 * time.Millisecond)
	defer drainTicker.Stop()

	for {
		select {
		case <-drainCtx.Done():
			dm.logger.Warn("shutdown timeout: processFile jobs did not drain in time", "timeout", dm.shutdownTimeout)
			goto done
		case <-drainTicker.C:
			goto done
		}
	}

done:
	if dm.monitor != nil {
		dm.monitor.Stop()
	}

	// RAM logger stops when context is canceled; no explicit Stop() needed

	dm.running = false
	dm.paused = false

	dm.logger.Info("daemon stopped")
	return nil
}

// SetPaused pauses/resumes file processing queue.
func (dm *daemonManager) SetPaused(paused bool) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !dm.running {
		return ErrNotRunning
	}

	dm.paused = paused
	state := "paused"
	if !paused {
		state = "resumed"
	}
	dm.logger.Info("daemon " + state)
	return nil
}

// IsPaused returns current pause state.
func (dm *daemonManager) IsPaused() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.paused
}

// IsRunning returns true if daemon is currently running.
func (dm *daemonManager) IsRunning() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.running
}

// Status returns human-readable daemon state.
func (dm *daemonManager) Status() string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if !dm.running {
		return "Stopped"
	}
	if dm.paused {
		return "Paused"
	}
	if dm.throttleGate.Load() {
		return "Throttled"
	}
	return "Running"
}

// Restart stops then starts daemon atomically.
func (dm *daemonManager) Restart(ctx context.Context) error {
	if err := dm.Stop(); err != nil && err != ErrNotRunning {
		return err
	}
	return dm.Start(ctx)
}

// SubscribeMetrics returns a channel for MetricsUpdate at ~1s intervals.
func (dm *daemonManager) SubscribeMetrics() <-chan MetricsUpdate {
	return dm.metricsC
}
