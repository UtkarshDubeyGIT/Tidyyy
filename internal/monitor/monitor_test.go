package monitor

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestMonitorPublishesMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := NewMonitor(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	select {
	case update := <-m.SubscribeMetrics():
		if update.Timestamp.IsZero() {
			t.Fatalf("expected non-zero timestamp")
		}
		if update.CPUPercent < 0 || update.CPUPercent > 100 {
			t.Fatalf("cpu percent out of range: %v", update.CPUPercent)
		}
		if update.MemPercent < 0 || update.MemPercent > 100 {
			t.Fatalf("mem percent out of range: %v", update.MemPercent)
		}
		if update.BatteryLevel != -1 {
			t.Fatalf("expected battery level -1 for stub, got: %d", update.BatteryLevel)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for metrics")
	}

	cancel()
}
