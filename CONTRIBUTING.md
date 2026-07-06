# Contributing to Lymuru

Thank you for your interest in contributing to Lymuru! This guide covers how to set up your local environment, make changes, and submit contributions.

---

## Getting Started

### Prerequisites

- **Go 1.25+** — [download](https://go.dev/dl/)
- **Bun** — [install](https://bun.sh)
- **Wails CLI** — `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Python 3.11+** *(optional)* — only needed for the Deezer sidecar
- **Windows 10/11** — the primary development target

### First-time setup

```bash
# Clone the repository
git clone https://github.com/rioBMO/Lymuru.git
cd Lymuru

# Install frontend dependencies
cd frontend
bun install
cd ..

# Verify everything builds
wails build
```

### Development workflow

```bash
# Start the dev server (hot reload for Go + React)
wails dev

# Build the native executable
wails build
# Output: build/bin/Lymuru.exe

# Run Go tests
go test ./backend/...
```

---

## How to Contribute

### Reporting issues

Found a bug or have a suggestion? Use one of our issue templates:

- **[Bug Report](https://github.com/rioBMO/Lymuru/issues/new?template=bug_report.md)** — something isn't working
- **[Feature Request](https://github.com/rioBMO/Lymuru/issues/new?template=feature_request.md)** — idea for a new feature

Include as much detail as possible: steps to reproduce, expected vs actual behavior, environment details, and any relevant logs from `data/logs/lymuru.log`.

### Submitting pull requests

1. **Create a branch** from `main` (or the relevant base branch):
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Branch naming** — use a descriptive prefix:
   - `fix/` — bug fixes
   - `feat/` — new features
   - `docs/` — documentation changes
   - `refactor/` — code restructuring (no behavioral changes)

3. **Make your changes** — keep them focused and scoped to a single concern.

4. **Commit your changes** using clear, descriptive messages:
   ```
   feat: add resample quality selector
   fix: handle empty search results gracefully
   docs: update contributing guide
   ```

5. **Build and test** before submitting:
   ```bash
   go build ./...
   wails build
   go test ./backend/...
   ```

6. **Push** and open a pull request using the [PR template](.github/pull_request_template.md).

---

## Code Style

- **Go**: Follow standard Go conventions. Handle errors explicitly. Keep packages focused on a single responsibility.
- **React / TypeScript**: Follow the patterns established in `frontend/src/`. Use shadcn/ui primitives for all UI elements. Keep components focused and readable.
- **Commenting**: Explain *why* something is done, not *what* is being done. The code should be self-documenting for the "what".

---

## Project Structure

```
Lymuru/
├── main.go                  # Wails entrypoint
├── app.go / app_downloads.go  # Wails bindings (backend → frontend API)
├── backend/                 # Go packages (providers, lyrics, FFmpeg, SQLite, etc.)
├── frontend/                # React + TypeScript + Vite + Bun
│   └── src/
│       ├── components/      # Pages, dialogs, UI primitives
│       ├── hooks/           # React hooks
│       └── lib/             # Utilities and settings
├── sidecar/
│   └── deezload.py          # Python Deezer sidecar
└── data/                    # Runtime data (created at first launch)
```

---

## License

All contributions fall under the [MIT License](LICENSE).
