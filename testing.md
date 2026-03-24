# Tidyyy User Testing Guide

## Purpose

This guide covers manual user testing for the unified daemon lifecycle, settings integration, and tray runtime.

## Preconditions

1. Build succeeds.
2. You have at least one valid folder for watch testing.
3. You can create/copy files into watched folders.

## Build and Sanity Commands

Run from project root:

```bash
go test ./...
go build -o dist/tidyyy ./cmd/tidyyy
```

Expected:

- All tests pass.
- Binary builds at `dist/tidyyy`.

## Runtime Test Commands

### 1) Start in settings + tray runtime mode

```bash
./dist/tidyyy --settings
```

Expected:

- Settings window opens.
- Tray icon/menu is available.
- Closing settings window hides it (app stays alive in tray).

### 2) Start in normal daemon mode

```bash
./dist/tidyyy
```

Expected:

- Daemon auto-starts if configured watch dirs are valid.
- Process runs until interrupted.

### 3) Single-instance lock validation

Terminal A:

```bash
./dist/tidyyy
```

Terminal B:

```bash
./dist/tidyyy
```

Expected:

- Second instance exits with already-running lock message.

### 4) Watcher/rename pipeline smoke test

1. Ensure a watch folder is configured in settings.
2. Drop/copy a supported file (`.png`, `.jpg`, `.jpeg`, `.webp`, `.pdf`) into the folder.

Expected:

- Event is triaged and processed.
- Rename history/log updates.

## Functional User Test Checklist

### A. Settings Save + Lifecycle Hooks

1. Open settings from tray.
2. Change `Max Name Words` or `Cloud Enabled`.
3. Save.

Expected:

- If daemon is running: restart confirmation appears.
- Choosing restart applies changes immediately.
- Choosing keep current leaves daemon running with old runtime config.

4. Stop daemon (or start with invalid/empty watch dirs), then save valid watch dirs.

Expected:

- Daemon auto-starts after save.

### B. Tray Behavior

1. Close settings window.

Expected:

- Window hides, process continues.

2. Use tray `Open Settings`.

Expected:

- Settings window reopens.

3. Use tray `Pause/Resume`.

Expected:

- Daemon pause state toggles.

4. Use tray `Restart Daemon`.

Expected:

- Daemon restarts without app exit.

5. Use tray `Quit Tidyyy`.

Expected:

- Daemon stops cleanly and process exits.

### C. Shutdown Integrity

1. Start app and trigger file processing.
2. Quit from tray (or Ctrl+C in non-tray mode).

Expected:

- Clean shutdown.
- No stuck process.

## Helpful Observability Commands

```bash
# RAM logger output
tail -f logs/ram_usage.txt

# rename history
tail -f logs/rename_history.jsonl

# check running process
ps aux | grep tidyyy | grep -v grep
```

## Exit Criteria for User Testing

You can move forward if all are true:

1. Build and tests pass.
2. Settings save hooks behave correctly (restart prompt / auto-start).
3. Tray close-to-hide and reopen path works.
4. Pause/Resume and Restart actions work from tray.
5. Quit always stops daemon and exits cleanly.
6. Single-instance lock blocks duplicate process.

## Known Limitations During Testing

1. Battery-based throttle is currently stubbed.
2. Tray history/undo UX is not fully completed yet.
3. UI integration is primarily validated manually (limited UI automation).
