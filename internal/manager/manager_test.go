package manager

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/tidyyy/internal/monitor"
)

type fakeMonitor struct {
	startErr error
	started  bool
	stopped  bool
	metrics  chan monitor.MetricsUpdate
}

func newFakeMonitor() *fakeMonitor {
	return &fakeMonitor{metrics: make(chan monitor.MetricsUpdate, 8)}
}

func (f *fakeMonitor) Start(ctx context.Context) error {
	if f.startErr != nil {
		return f.startErr
	}
	f.started = true
	return nil
}

func (f *fakeMonitor) Stop() {
	f.stopped = true
}

func (f *fakeMonitor) SubscribeMetrics() <-chan monitor.MetricsUpdate {
	return f.metrics
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDaemonStartStopIdempotent(t *testing.T) {
	m := newFakeMonitor()
	dm := NewDaemonManager(nil, nil, nil, nil, nil, m, nil, Config{Logger: testLogger()})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := dm.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !dm.IsRunning() {
		t.Fatalf("daemon should be running after Start")
	}

	if err := dm.Start(ctx); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got: %v", err)
	}

	if err := dm.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if dm.IsRunning() {
		t.Fatalf("daemon should not be running after Stop")
	}
	if !m.stopped {
		t.Fatalf("monitor Stop should be called")
	}

	if err := dm.Stop(); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("expected ErrNotRunning, got: %v", err)
	}
}

func TestDaemonSetPausedRequiresRunning(t *testing.T) {
	dm := NewDaemonManager(nil, nil, nil, nil, nil, nil, nil, Config{Logger: testLogger()})

	if err := dm.SetPaused(true); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("expected ErrNotRunning when paused before start, got: %v", err)
	}
}

func TestDaemonPauseResumeStatus(t *testing.T) {
	dm := NewDaemonManager(nil, nil, nil, nil, nil, nil, nil, Config{Logger: testLogger()})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := dm.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer func() { _ = dm.Stop() }()

	if got := dm.Status(); got != "Running" {
		t.Fatalf("expected Running, got %q", got)
	}

	if err := dm.SetPaused(true); err != nil {
		t.Fatalf("set paused failed: %v", err)
	}
	if !dm.IsPaused() {
		t.Fatalf("expected paused state true")
	}
	if got := dm.Status(); got != "Paused" {
		t.Fatalf("expected Paused, got %q", got)
	}

	if err := dm.SetPaused(false); err != nil {
		t.Fatalf("set resumed failed: %v", err)
	}
	if dm.IsPaused() {
		t.Fatalf("expected paused state false")
	}
	if got := dm.Status(); got != "Running" {
		t.Fatalf("expected Running, got %q", got)
	}
}

func TestDaemonRestart(t *testing.T) {
	dm := NewDaemonManager(nil, nil, nil, nil, nil, nil, nil, Config{Logger: testLogger()})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := dm.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if err := dm.Restart(ctx); err != nil {
		t.Fatalf("restart failed: %v", err)
	}
	if !dm.IsRunning() {
		t.Fatalf("daemon should be running after restart")
	}

	if err := dm.Stop(); err != nil {
		t.Fatalf("stop after restart failed: %v", err)
	}
}

func TestDaemonStartWithCanceledContext(t *testing.T) {
	dm := NewDaemonManager(nil, nil, nil, nil, nil, nil, nil, Config{Logger: testLogger()})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := dm.Start(ctx); !errors.Is(err, ErrInvalidContext) {
		t.Fatalf("expected ErrInvalidContext, got: %v", err)
	}
}

func TestDaemonThrottleStateViaMetrics(t *testing.T) {
	m := newFakeMonitor()
	dm := NewDaemonManager(nil, nil, nil, nil, nil, m, nil, Config{Logger: testLogger()})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := dm.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer func() { _ = dm.Stop() }()

	m.metrics <- monitor.MetricsUpdate{CPUPercent: 70, BatteryLevel: 100, Timestamp: time.Now()}

	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case <-deadline:
			t.Fatalf("expected status to become Throttled")
		default:
			if dm.Status() == "Throttled" {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}
