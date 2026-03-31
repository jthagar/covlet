# covlet

Cover letter templates with a Go **backend** (Fiber API) and a **terminal UI** (Bubble Tea). There is no web frontend; run the TUI on your machine.

## Layout

- `backend/` — HTTP API (`./backend/cmd/covlet`): template CRUD, render, PDF export via [gopdf](https://github.com/signintech/gopdf) with embedded DejaVu Sans fonts (optional custom TTFs via env).
- `frontend/` — TUI client (`./frontend/cmd/covlet-tui`).

## Environment

| Variable | Purpose |
|----------|---------|
| `COVLET_HOME` | Application data directory (default: `~/.local/share/covlet`). Templates live under `<home>/templates`. |
| `COVLET_LISTEN` / `COVLET_PORT` | HTTP listen address (default `:8080`). |
| `COVLET_PDF_FONT` | Optional path to a TTF for PDF body text (replaces embedded regular font). |
| `COVLET_PDF_FONT_BOLD` | Optional path to a TTF for PDF titles. If unset while `COVLET_PDF_FONT` is set, the same file is used for both. |
| `COVLET_API` | Default API URL for the TUI (default `http://127.0.0.1:8080`). |
| `COVLET_RESUME` | Default resume YAML path for the TUI (`-resume` overrides). |

## Run locally

1. **Backend:** `go run ./backend/cmd/covlet`
2. **TUI:** `go run ./frontend/cmd/covlet-tui` (optional `-api`, `-resume`)

`POST /api/v1/export/pdf` returns `application/pdf` bytes (Ctrl+P in the TUI; save directory defaults to `~/Downloads`).

## Docker

`docker compose up` builds and runs the API image only. Run the TUI from the repo with Go.
