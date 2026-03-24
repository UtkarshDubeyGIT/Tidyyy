# Architecture: Unified Process + Managed Daemon Lifecycle

## Overview

Tidyyy now runs as a single process that can host:

- daemon lifecycle manager
- settings window
- tray runtime

The daemon is managed in-process and controlled through explicit lifecycle APIs.

## Core Runtime Structure

### 1. Process Guard

- `internal/lock/lock.go`
- Single-instance lock acquired at startup

### 2. Daemon Orchestration

- `internal/manager/manager.go`
- Owns runtime state and lifecycle transitions

Key state:

- running / stopped
- paused / resumed
- throttled (resource gate)

Key transitions:

- `Start(ctx)`
- `Stop()`
- `Restart(ctx)`
- `SetPaused(bool)`

### 3. Runtime Components Managed by Daemon

- Watcher (`internal/watcher`)
- Triage queue (`internal/triage`)
- Extractor (`internal/extractor`)
- Namer (`internal/namer`)
- Renamer (`internal/renamer`)
- Monitor (`internal/monitor/monitor.go`)
- RAM logger (`internal/monitor/ram_logger.go`)

Data flow:

1. watcher event
2. dedupe + pause checks
3. triage accept
4. job processing loop
5. extract -> name -> rename

## UI and Lifecycle Control

### Settings Window

- `internal/ui/settings.go`
- Save operation supports lifecycle hooks:
  - daemon running + relevant config changed -> restart prompt
  - daemon stopped + valid dirs -> auto-start

### Tray Runtime

- `internal/ui/tray.go`
- Settings window close is intercepted to hide window (not quit)
- Tray menu provides daemon controls:
  - Pause/Resume
  - Restart Daemon
  - Open Settings
  - Quit Tidyyy

## Main Entry Wiring

- `cmd/tidyyy/main.go`

Startup flow:

1. acquire single-instance lock
2. load config + resolve watch dirs
3. initialize dependencies
4. create daemon manager
5. optional daemon auto-start (if valid dirs)
6. if `--settings`: run tray runtime with hooks
7. otherwise run normal daemon mode until signal

Shutdown flow:

- signal or tray quit
- daemon stop
- lock release
- process exit

## Non-Goals in Current Build

1. Full history submenu + undo tray UX completion (F-07 polish)
2. Advanced batch throttle controls beyond current gate logic
3. Full UI integration test automation

## Verification Snapshot

- Unit tests pass for manager/lock/monitor
- Full test suite passes (`go test ./...`)
- Build passes (`go build -o dist/tidyyy ./cmd/tidyyy`)
