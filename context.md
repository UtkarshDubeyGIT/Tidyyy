# Tidyyy User Context

## What Tidyyy Is

Tidyyy is a local-first background desktop utility that watches selected folders and automatically renames newly added files into clean, searchable names.

Primary file types currently handled:

- PNG
- JPG and JPEG
- WEBP
- PDF

Core value:

- Less manual cleanup of generated file names
- Better searchability in Finder and Explorer
- Consistent slug-style naming format

## Product Promise

- Lightweight background behavior with a single process model
- Local naming path by default using on-device model inference
- Optional cloud naming fallback when explicitly enabled
- Conflict-safe renames with suffix strategy like -2, -3

## High-Level Pipeline

1. Watcher receives file-system events in configured folders.
2. Triage validates file type with extension and MIME checks.
3. Extractor reads text from PDF or OCRs image content.
4. Namer generates a slug from extracted text.
5. Renamer atomically renames file in-place and resolves collisions.

## Naming Rules

Tidyyy slug validation enforces practical naming constraints:

- Lowercase words joined by hyphens
- 2 to 5 words total
- No file extension in generated slug
- No timestamp-style output
- No numeric counter suffix from model output
- Original extension is preserved during rename

If a generated name already exists in the same folder, Tidyyy appends suffixes automatically:

- example-name.pdf
- example-name-2.pdf
- example-name-3.pdf

## Runtime Behavior Users Should Expect

- File settle delay before processing: about 2 seconds after file write activity
- Dedupe protection to avoid repeated processing of the same path over a TTL window
- Default concurrent processing cap: 2 jobs
- Throttle gate when monitor reports high resource conditions
- Graceful shutdown on quit and OS termination signals

## Tray and Settings UX

Tidyyy can run in settings-plus-tray mode.

Tray capabilities:

- Daemon status indicator
- Pause and resume daemon
- Restart daemon
- Open settings window
- Quit Tidyyy

Settings behavior:

- Save may prompt daemon restart when daemon-affecting options changed
- If daemon is stopped and valid watch folders are saved, daemon auto-start is attempted
- Closing settings window hides window in tray runtime, does not exit the app

## Single Instance Model

Tidyyy uses a lock file in the system temp directory so only one daemon instance owns processing at a time.

If another instance is started while one is active, startup is rejected.

## Configuration Sources

Tidyyy reads configuration from two places:

1. User config file in the OS user config directory
2. Environment variables from .env or shell environment

Environment values override config file values at runtime.

## User-Visible Configuration Fields

Common runtime environment settings:

- WATCH_DIRS: comma-separated absolute or relative folder list
- POPPLER_BIN: optional Poppler bin path
- TESSERACT_BIN: optional tesseract executable or bin path
- LLAMA_CLI_PATH: local llama-cli executable path
- MODEL_PATH: gguf model path
- TIMEOUT_SEC: command timeout for extraction/naming subprocesses
- PDF_PAGE_LIMIT: max PDF pages for text extraction pass
- QUEUE_DEPTH: triage queue size
- CLOUD_ENABLED: enables cloud naming path when true
- CLOUD_BASE_URL: OpenAI-compatible base URL
- CLOUD_API_KEY: cloud API token
- CLOUD_MODEL: cloud model identifier
- MAX_NAME_WORDS: max slug word count, clamped to 2..5
- SKIP_RENAME_ON_INVALID: when true, disables heuristic fallback naming
- RAM_LOG_FILE: RAM log output path
- RAM_LOG_INTERVAL_SEC: RAM sampling interval
- DEDUP_TTL_SEC: dedupe interval for repeated file paths

## Extraction Details Relevant to Users

PDF flow:

- First attempts pdftotext extraction for configured page range
- Falls back to rendering first page and OCR when PDF text is empty

Image flow:

- Uses tesseract OCR
- Enforces maximum OCR input size threshold

Content cleanup:

- Drops stop words and noisy OCR artifacts
- Deduplicates repeated tokens
- Produces cleaned text input for naming

## Naming Source Order

At runtime Tidyyy attempts naming in this order:

1. Local model via llama-cli
2. Cloud model if enabled and configured
3. Heuristic fallback slug, unless fallback is disabled

If fallback is disabled and both local and cloud naming fail validation, rename is skipped.

## Logs and Observability

Operational files commonly useful during testing:

- logs/ram_usage.txt for periodic memory logging
- logs/rename_history.jsonl for pre-rename history entries

Useful checks:

- tail logs while dropping sample files in watched folders
- verify process is alive in system process list
- verify single-instance lock behavior by launching a second binary

## Build and Run

Project-level scripts:

- build.sh builds dist/tidyyy
- run.sh builds if needed, then runs dist/tidyyy with passthrough args

Typical usage patterns:

- Run daemon mode for background operation
- Run with --settings for tray plus settings workflow

## Current Known Gaps

Based on current implementation notes:

- Battery metric is stubbed in monitor layer
- Tray history and one-click undo UX are not fully completed
- UI integration testing is mostly manual at this stage

## Security and Privacy Notes

- Do not commit real API keys or tokens to repository files.
- Keep cloud naming opt-in and explicit for users.
- Treat extracted content as potentially sensitive user data.

## Landing Page Push Context

If this file is used to brief landing page updates, use these positioning points:

- Headline idea: Stop losing files to bad names.
- Core proof: automated rename of screenshots, photos, and PDFs.
- Trust message: local-first path with optional cloud fallback.
- CTA direction: Download Tidyyy and See How It Works.

Recommended landing sections:

1. Hero value proposition with clear CTA
2. Before/after filename examples
3. How it works pipeline
4. Local-first and privacy trust block
5. Supported files and platform status
6. Final CTA and FAQ
