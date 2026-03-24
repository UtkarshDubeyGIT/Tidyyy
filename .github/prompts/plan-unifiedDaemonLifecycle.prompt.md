## Plan: Unified GUI + Managed Daemon Lifecycle

Convert Tidyyy from split modes into one app process where the GUI opens on launch, daemon lifecycle is managed in-process, daemon auto-starts when saved watch paths are valid, and it keeps running in background until explicit app quit. When settings change during runtime, show a restart option as requested.

**Steps**

1. Phase 1: Extract daemon orchestration from [cmd/tidyyy/main.go](cmd/tidyyy/main.go) into reusable start/stop primitives so lifecycle is callable from GUI flow instead of hardwired to CLI execution.
2. Add a daemon manager state machine with guarded APIs (start, stop, restart) and idempotent behavior to prevent duplicate starts and unsafe concurrent transitions.
3. Add single-instance locking at app boot so only one Tidyyy process can own the daemon at a time.
4. Phase 2: Make GUI-first startup the default. Load config at launch, then auto-start daemon only if at least one watch directory is valid and accessible.
5. Wire settings save in [internal/ui/settings.go](internal/ui/settings.go) to lifecycle behavior:
   - If daemon is stopped and save produces valid watch dirs, auto-start immediately.
   - If daemon is running and settings changed, prompt Restart now or Keep current daemon config.
6. Implement **F-08 (Batch Throttle)**: add queue depth and resource-gating logic to daemon manager — process at most N files concurrently (default 2), pause queue when CPU > 60% or battery < 15%. Integrate with triage queue and watcher event loop.
7. Implement **F-09 (Tray Icon & Menu)** with full UX: system tray icon displays active/paused state; context menu includes: Pause/Resume, History (F-07), Open Settings, Restart Daemon, Quit Tidyyy. Closing settings window hides UI but keeps app and daemon alive.
8. Implement **F-07 (Undo / History)**: expose single-click undo action in tray History submenu; show recent rename operations (old_name → new_name, timestamp, folder) fetched from existing SQLite history log.
9. Phase 3: Harden shutdown path so Quit always stops watcher, triage, batch throttle, and monitor cleanly before process exit.
10. Add tests for daemon manager transitions, batch throttle behavior (CPU/battery gating), tray menu state bindings, history log integration, single-instance lock behavior, and regression checks in watcher/triage flows.

**Relevant files**

- [cmd/tidyyy/main.go](cmd/tidyyy/main.go) — split startup logic and route through app-managed lifecycle.
- [internal/ui/settings.go](internal/ui/settings.go) — save-triggered daemon actions, restart prompt, hide-on-close behavior.
- [internal/ui/tray.go](internal/ui/tray.go) — **NEW**: system tray icon & menu (F-09); Pause/Resume, History submenu, Settings, Restart, Quit; state indicator (active/paused).
- [internal/config/config.go](internal/config/config.go) — reuse and potentially extend config validity helpers for startup checks.
- [internal/watcher/watcher.go](internal/watcher/watcher.go) — verify safe stop/start usage under manager-driven restarts.
- [internal/triage/triage.go](internal/triage/triage.go) — integrate queue depth constraint and pause gate from batch throttle (F-08).
- [internal/monitor/ram_logger.go](internal/monitor/ram_logger.go) — track CPU/battery for throttle gate; ensure clean stop on daemon shutdown.
- [internal/history/history.go](internal/history/history.go) — append history queries for tray History menu display (F-07); single-click undo action support.
- [internal/watcher/watcher_test.go](internal/watcher/watcher_test.go) — restart/stop safety coverage.
- [internal/triage/triage_test.go](internal/triage/triage_test.go) — clean shutdown and queue depth behavior under manager stop.
- [internal/renamer/renamer_test.go](internal/renamer/renamer_test.go) — regression safety while lifecycle control changes.

**Verification**

1. Start app with valid saved watch dirs: GUI opens and daemon auto-starts; tray icon shows active state.
2. Start app with invalid/empty dirs: daemon stays stopped; save valid dirs in settings; daemon starts immediately and tray icon updates.
3. While running, save changed settings: restart prompt appears and both choices work correctly.
4. Close settings window: app remains in tray, daemon continues; tray menu remains accessible.
5. **F-09 Tray Menu**: click Pause → daemon pauses, tray icon changes to paused state, queue stops accepting new jobs; click Resume → resumes. History menu item shows recent renames (F-07); click Undo on an entry → file reverted to original name.
6. **F-08 Batch Throttle**: generate > N file events rapidly; verify only N files process concurrently. Simulate CPU > 60% or battery < 15%; queue pauses and resumes when resource condition clears.
7. Quit from tray: daemon stops cleanly, all pending jobs drain gracefully, process exits with no dangling goroutines.
8. Launch a second instance while first runs: second instance is blocked with clear message; single-instance lock verified.
9. Run focused tests for lifecycle manager, batch throttle resource gating, tray state bindings, history log queries, single-instance lock, and package regressions.

**Decisions captured**

- Included: unified process, GUI-first launch, background tray mode, single-instance enforcement.
- Included: save-time restart option when settings change while daemon is running.
- Included: **F-07 (Undo/History)** — single-click undo via tray History menu, integrating with existing SQLite log.
- Included: **F-08 (Batch Throttle)** — resource-gated queue with max concurrent files (≤2) and CPU/battery pause logic.
- Included: **F-09 (Tray Icon & Menu)** — full system tray integration with active/paused state indicator, Pause/Resume toggle, History submenu, Settings, Restart, Quit actions.
- Excluded: hot-reload without restart (user requested live restart option instead).
- Excluded: cross-process IPC reload design (unified in-process manager replaces this need).

If this plan looks right, approve and I will hand off for implementation.
