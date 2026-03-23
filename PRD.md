# 📋 Tidyyy PRD v0.1

> **Version:** 0.1 — Initial Draft | **Status:** In Review | **Date:** March 2026 | **Owner:** Product Team
> 

---

# 1. Executive Summary

Tidyyy is a lightweight, background file-renaming utility that lives on the user's local machine. It watches designated folders and uses AI-driven content and metadata analysis to replace cryptic auto-generated filenames — screenshots, exported PDFs, phone photos — with concise, human-readable names in two to four words.

The core promise is zero friction: install once, forget it exists, and always find your files instantly.

## The Problem vs. The Solution

| ❌ Problem | ✅ Solution |
| --- | --- |
| Files like `Screenshot2026-02-12at7.21.34AM.png` or `IMG_4938.JPG` are unsearchable noise. Users waste minutes hunting for the right file, rename manually, or give up entirely. | Tidyyy silently analyses each file, derives intent from its content, and renames it to something like `power-bi-export-guide.png` — searchable, descriptive, and tidy. |

---

# 2. Product Vision & Goals

Tidyyy should feel like a thoughtful assistant that silently keeps your file system in order — not a heavy application demanding attention or resources.

## Design Pillars

| Pillar | Description |
| --- | --- |
| 🪶 Featherweight | Minimal CPU and RAM footprint. Never interrupts the user. Degrades gracefully on low-resource machines. |
| 🎯 Precision Names | 2–4 words derived from actual content. Always lowercase, hyphen-separated slugs. |
| 🔒 Local-first | No cloud dependency for processing. AI calls are optional and explicitly opt-in. Files never leave the device without consent. |
| 📦 Tiny Binary | Single installable under 50 MB target. No heavy runtimes bundled unless strictly necessary. |
| 👻 Background-only | Runs as a system tray daemon. No mandatory onboarding, no nagging notifications. |

---

# 3. Scope

## In Scope — v1

- PNG / JPG / JPEG / WEBP screenshots and photos
- PDF documents — exported reports, forms, receipts
- Watched folder list configurable by user
- Rename-on-detect (< 30 s lag)
- Undo / rename history log
- macOS and Windows tray daemon
- Optional on-device OCR for image text

## Out of Scope — v1

- Video files
- Audio files
- Cloud drive integration *(v2 candidate)*
- Bulk-rename of existing files at install
- Linux support *(v2 candidate)*
- Enterprise MDM deployment

---

# 4. Naming Convention Rules

These rules are non-negotiable constraints baked into the renaming engine, not soft guidelines.

## Structural Rules

| Rule | Detail | Example |
| --- | --- | --- |
| Format | lowercase-hyphen-separated-slug | `power-bi-dashboard.png` ✓ |
| Length (target) | 2–3 words for most files | `sales-report.pdf` ✓ |
| Length (max) | 4–5 words hard ceiling | `q1-2026-budget-forecast.xlsx` ✓ |
| No timestamps | Never include dates or times | `screenshot-2026.png` ✗ |
| No counters | No IMG_001-style suffixes | `document-1.pdf` ✗ |
| Extension | Preserve original unchanged | `.jpeg` stays `.jpeg` |
| Conflict | Append -2, -3 if name already exists | `report-2.pdf` |

## Derivation Priority (in order)

1. **File content** — OCR text, embedded PDF metadata, EXIF description fields
2. **Visual content summary** — AI scene / object detection for images
3. **Source application metadata** — e.g. app name in screenshot EXIF
4. **Folder context** — parent folder name may hint at domain
5. **Filename pattern heuristics** — last resort, extract any meaningful tokens

---

# 5. Functional Requirements

## 5.1 Core Engine

| ID | Requirement | Description | Priority |
| --- | --- | --- | --- |
| F-01 | Folder Watcher | Monitor user-configured folders using native OS file-system events (FSEvents / ReadDirectoryChangesW) with no polling. | Must Have |
| F-02 | File Triage | On new-file detect: determine type, skip non-target types silently, queue eligible files for analysis. | Must Have |
| F-03 | Content Extraction | Extract text from PDFs (pdftotext / pdfmium), run on-device OCR on images to surface readable content. | Must Have |
| F-04 | Name Generation | Produce a 2–5 word slug using extracted content. Primary: lightweight on-device model. Secondary: optional cloud LLM call. | Must Have |
| F-05 | Atomic Rename | Rename the file atomically. Write undo record to history log before renaming. | Must Have |
| F-06 | Conflict Resolution | Detect name collisions in the same directory; append -2, -3, etc. automatically. | Must Have |
| F-07 | Undo / History | Maintain an append-only SQLite log of (old_name, new_name, timestamp, folder). Expose single-click undo via tray. | Must Have |
| F-08 | Batch Throttle | Process at most N files concurrently (default 2). Pause queue when CPU > 60% or battery < 15%. | Must Have |

## 5.2 Configuration & UX

| ID | Requirement | Description | Priority |
| --- | --- | --- | --- |
| F-09 | Tray Icon & Menu | System tray icon shows active/paused state. Menu: Pause, History, Settings, Quit. | Must Have |
| F-10 | Settings UI | Minimal native-style settings window: add/remove watched folders, toggle AI cloud mode, set max name length. | Must Have |
| F-11 | Notifications | Optional toast on successful rename showing old → new name. Off by default. | Should Have |
| F-12 | Exclusion Rules | Allow glob patterns to exclude files (e.g. `*.tmp`, `node_modules/**`). | Should Have |
| F-13 | Auto-start | Register as login item on install. Toggle in Settings. | Should Have |
| F-14 | Manual Trigger | Right-click any file in Finder / Explorer → 'Rename with Tidyyy'. | Nice to Have |

---

# 6. Non-Functional Requirements

| Category | Target | Notes |
| --- | --- | --- |
| CPU usage | < 2% average | Burst up to 15% during active rename; return to idle within 3 s |
| RAM footprint | < 80 MB resident | Excluding OS-level file watch buffers |
| Disk (binary) | < 50 MB installed | Ship with Tesseract tiny model; larger models are optional download |
| Rename latency | < 30 s from file land | Acceptable for background daemon; real-time is a v2 target |
| Accuracy target | > 85% acceptance | Measured via opt-in feedback; below threshold triggers model update |
| Startup time | < 1 s to tray idle | Must not delay system login perceptibly |
| Data privacy | No telemetry default | All analysis local by default; cloud opt-in is explicit and auditable |

---

# 7. Technical Architecture

## Proposed Stack

| Layer | Technology | Rationale |
| --- | --- | --- |
| Runtime | `Go 1.22+` | Single static binary ~10 MB, near-zero idle CPU, native threads |
| File watching | `fsnotify (Go)` | Wraps FSEvents / inotify / ReadDirectoryChanges natively |
| PDF text | `pdftotext (Poppler CLI)` | Ships as a tiny sidecar; fast, accurate, no heavy library |
| OCR | `Tesseract 5 (tiny model)` | ~8 MB model; good enough for screenshot captions and UI text |
| Name generation | `llama.cpp (3B) / API` | On-device default; optional OpenAI-compat API for power users |
| Persistence | `SQLite via modernc/sqlite` | Pure Go, no CGO; stores history log and config |
| UI / tray | `systray + fyne.io` | Minimal native tray icon + settings window; < 5 MB addition |

## Processing Pipeline

```
File Lands → Triage → Content Extract → Name Generate → Atomic Rename → Log
```

---

# 8. Design Constraints

| ID | Constraint |
| --- | --- |
| C-1 | **Resource ceiling** — CPU < 2% idle, RAM < 80 MB, binary < 50 MB. Hard gates for any PR merge. |
| C-2 | **Name length** — 2–3 words target, 4–5 words max. The engine must reject longer outputs. |
| C-3 | **Relevance** — name must be derivable from content or verified metadata. Heuristic guesses from filename patterns are a last resort. |
| C-4 | **Atomicity** — rename must succeed fully or not at all. Partial renames leave the file under the original name. |
| C-5 | **Privacy** — no file content is transmitted off-device unless cloud mode is explicitly enabled by the user. |
| C-6 | **Reversibility** — every rename must be undoable from the history log without requiring the original filename to be remembered. |

---

# 9. Open Questions

| ID | Question |
| --- | --- |
| OQ-1 | Model size vs. accuracy trade-off: Is a 3B parameter model sufficient for ≥ 85% accuracy, or do we need an 8B model at the cost of RAM? |
| OQ-2 | Windows support timeline: Can the Go + fsnotify + fyne stack ship Windows and macOS simultaneously at v1? |
| OQ-3 | EXIF write-back: Should Tidyyy update the EXIF title/description field to match the new slug, or is rename-only cleaner? |
| OQ-4 | Opt-in feedback loop: How do we collect accuracy feedback without compromising privacy? Anonymous content hash + thumbs-up/down? |
| OQ-5 | App bundle identity: Signed `.app` (macOS notarization) on day one, or developer-ID signing sufficient for v1? |

---

# 10. Milestones

| Milestone | Target | Deliverable |
| --- | --- | --- |
| M0 — Scaffold | Week 2 | Repo, CI, folder watcher, trivial rename smoke test |
| M1 — MVP Engine | Week 5 | PDF + OCR extraction + slug generation + undo log working on macOS |
| M2 — Tray UI | Week 7 | Tray icon, settings window, exclusion rules |
| M3 — Beta | Week 10 | Windows port, accuracy ≥ 80% on test corpus, installer |
| M4 — v1.0 | Week 14 | Code-signed binaries, notarized macOS app, public release |

---

*Tidyyy PRD v0.1 — Confidential Draft — March 2026*

# Code Repository Structure

```sql
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