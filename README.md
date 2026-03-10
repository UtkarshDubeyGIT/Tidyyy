# proposed application repository structure

```
tidyyy/
├── cmd/
│   └── tidyyy/
│       └── main.go              # Entry point — starts daemon, tray
│
├── internal/
│   ├── watcher/
│   │   └── watcher.go           # fsnotify wrapper, folder monitoring
│   │
│   ├── triage/
│   │   └── triage.go            # File type detection, queue eligibility
│   │
│   ├── extractor/
│   │   ├── extractor.go         # Interface + dispatcher
│   │   ├── ocr.go               # Tesseract 5 integration (images)
│   │   └── pdf.go               # pdftotext / Poppler wrapper
│   │
│   ├── namer/
│   │   ├── namer.go             # Interface for name generation
│   │   ├── local.go             # llama.cpp on-device model
│   │   ├── cloud.go             # Optional OpenAI-compat API fallback
│   │   └── slug.go              # Slug validation, length enforcement, rules
│   │
│   ├── renamer/
│   │   └── renamer.go           # Atomic rename + conflict resolution (-2, -3)
│   │
│   ├── history/
│   │   └── history.go           # SQLite append-only log (modernc/sqlite)
│   │
│   ├── config/
│   │   └── config.go            # Load/save settings (watched folders, toggles)
│   │
│   ├── scheduler/
│   │   └── scheduler.go         # Batch throttle, CPU/battery gating (≤2 concurrent)
│   │
│   └── ui/
│       ├── tray.go              # systray — icon, Pause/History/Settings/Quit
│       └── settings.go          # fyne.io settings window
│
├── assets/
│   ├── icon-active.png
│   └── icon-paused.png
│
├── models/                      # Bundled Tesseract tiny model (gitignored if large)
│
├── db/
│   └── schema.sql               # SQLite schema for history log
│
├── go.mod
├── go.sum
└── README.md
```
## How the pipeline maps to packages
watcher uses fsnotify to emit events for new files in configured folders. It's purely event-driven — no polling (per F-01).
triage receives each event and checks: is this a PNG/JPG/PDF? Is it in the exclusion glob list? If eligible, it pushes to the processing queue.
scheduler wraps the queue with the concurrency limit (default 2) and the CPU/battery guards from F-08. This is the chokepoint that keeps Tidyyy featherweight.
extractor dispatches to either ocr.go (Tesseract for images) or pdf.go (pdftotext for PDFs) and returns raw extracted text.
namer takes that text and produces a validated 2–5 word slug. slug.go enforces the naming rules (no timestamps, no counters, lowercase-hyphen only). local.go is the default path via llama.cpp; cloud.go only activates when the user opts in.
renamer does the atomic rename — writes the undo record to history first, then renames (per C-4 atomicity constraint).
history is a simple SQLite table: (id, old_path, new_path, renamed_at). Used by the tray's undo action.
ui is intentionally thin — just systray for the tray daemon and a fyne.io window for settings. No Electron, no web views.

### Key design decisions from the PRD to keep in mind
The PRD is strict about a few things that should be baked in early rather than bolted on: the resource ceiling (CPU < 2%, RAM < 80 MB, binary < 50 MB) needs to be tracked from M0, not patched at M3. The namer/slug.go validation layer should be a hard rejection — not a soft warning — so the engine never emits a name that violates the rules. And the history log should be written before the rename, not after, because that's the only way atomicity (C-4) + reversibility (C-6) both hold.
