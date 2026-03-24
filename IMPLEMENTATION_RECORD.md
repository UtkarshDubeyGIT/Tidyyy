# Tidyyy Unified Daemon Lifecycle - Implementation Record

## Status

Core daemon lifecycle, settings integration, and tray runtime wiring are implemented and working.

## Implemented Components

### 1. Daemon Manager

File: `internal/manager/manager.go`

Implemented lifecycle APIs:

- `Start(ctx)`
- `Stop()`
- `Restart(ctx)`
- `SetPaused(bool)`
- `IsPaused()`
- `IsRunning()`
- `Status()`
- `SubscribeMetrics()`

Behavior:

- Idempotent start/stop guards (`ErrAlreadyRunning`, `ErrNotRunning`)
- Internal context lifecycle with graceful shutdown
- Pause state support (transient runtime state)
- Resource throttle gate driven by monitor updates
- Deduplication in watcher event flow
- Bounded concurrent processing (semaphore)

### 2. Monitor + Metrics

Files:

- `internal/monitor/monitor.go`
- `internal/monitor/ram_logger.go`

Behavior:

- Periodic metrics publication
- CPU/memory telemetry (battery currently stubbed)
- RAM logging lifecycle tied to daemon context

### 3. Single-Instance Lock

File: `internal/lock/lock.go`

Behavior:

- Prevents second process from owning daemon via lock file
- Exposes lock/unlock/isLocked semantics

### 4. Settings Save Lifecycle Hooks

File: `internal/ui/settings.go`

Behavior on save:

- Detects daemon-impacting config changes
- If daemon is running and daemon-impacting fields changed:
  - Prompts user to restart now or keep current runtime config
- If daemon is stopped and saved watch dirs are valid:
  - Auto-starts daemon via hook
- Handles invalid/empty watch dir outcomes with user-facing status

### 5. Tray Runtime and Hide-on-Close

File: `internal/ui/tray.go`

Behavior:

- Creates tray runtime around settings window
- Closing settings window hides window (does not quit app)
- Tray menu supports:
  - Daemon status label
  - Pause/Resume
  - Restart Daemon
  - Open Settings
  - Quit Tidyyy

### 6. Main Wiring

File: `cmd/tidyyy/main.go`

Behavior:

- Uses lock to ensure single instance
- Builds daemon manager dependencies
- Auto-starts daemon when watch dirs are valid
- In `--settings` mode:
  - Runs tray runtime with settings + tray hooks
  - Keeps app active in background when settings closes
- Quit flow stops daemon cleanly

## Test and Build Verification

Most recent checks:

- `go test ./...` passes
- `go build -o dist/tidyyy ./cmd/tidyyy` passes

New unit tests added:

- `internal/manager/manager_test.go`
- `internal/lock/lock_test.go`
- `internal/monitor/monitor_test.go`

## Known Gaps / Follow-Ups

1. Tray/history UX is basic and functional but not yet full F-07 history submenu + one-click undo UX.
2. CPU/battery gating currently uses simplified monitor behavior; battery remains stubbed.
3. UI package has limited automated integration coverage (manual checks recommended).

## Release Readiness

- Daemon lifecycle: ready
- Settings-driven daemon restart/autostart: ready
- Tray hide-on-close runtime: ready
- Proceed to user testing: yes
