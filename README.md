# ██████╗ ██████╗  ██████╗      ██╗███████╗ ██████╗████████╗    ███████╗ ██████╗ █████╗ ███╗   ██╗
██╔══██╗██╔══██╗██╔═══██╗     ██║██╔════╝██╔════╝╚══██╔══╝    ██╔════╝██╔════╝██╔══██╗████╗  ██║
██████╔╝██████╔╝██║   ██║     ██║█████╗  ██║        ██║       ███████╗██║     ███████║██╔██╗ ██║
██╔═══╝ ██╔══██╗██║   ██║██   ██║██╔══╝  ██║        ██║       ╚════██║██║     ██╔══██║██║╚██╗██║
██║     ██║  ██║╚██████╔╝╚█████╔╝███████╗╚██████╗   ██║       ███████║╚██████╗██║  ██║██║ ╚████║
╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚════╝ ╚══════╝ ╚═════╝   ╚═╝       ╚══════╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝
                                                                                                

CLI and TUI project scanner for auditing Git hygiene, open-source readiness, dependencies, and exportable portfolio reports.

ProjectScans scans one project or a folder of projects and summarizes the signals that matter before sharing, maintaining, or publishing code: language mix, README quality, license presence, tests, CI, dependency manifests, Git metadata, and safety findings.

## What It Does

ProjectScans helps developers review local codebases before they become portfolio entries, internal deliverables, or public open-source repositories.

It can run as a terminal command for quick reports, or as an interactive TUI for browsing audit views and exporting reports.

```text
┌───────────────┐     ┌─────────────────┐     ┌────────────────────┐
│ Local folder  │────→│ ProjectScans     │────→│ Terminal/TUI report │
└───────────────┘     └─────────────────┘     └────────────────────┘
                             │
                             └────→ HTML / JSON / Markdown / CSV
```

## Features

- CLI summary, audit, details, JSON, Markdown, CSV, and HTML reports.
- Interactive Bubble Tea TUI for browsing project health.
- Git metadata checks for branch, remote, dirty state, and recent history.
- Open-source readiness scoring for README, tests, license, CI, deploy hints, and remote setup.
- README quality audit for title, description, installation, usage, configuration, and license sections.
- Safety audit for risky files, local machine references, and secret-like assignments without printing secret values.
- Configurable project labels and ignore rules through `.projectscan.toml` and `.projectscanignore`.

## Stack

- Go 1.26+
- Bubble Tea, Bubbles, and Lip Gloss for the TUI
- TOML config support through `github.com/BurntSushi/toml`

## Installation

Clone the repository and build the binary:

```bash
git clone https://github.com/logando-al/projectscans.git
cd projectscans
go build -o projectscan .
```

Run from source without building:

```bash
go run . --audit /path/to/projects
```

## Usage

Launch the TUI:

```bash
./projectscan
```

Scan the current folder:

```bash
./projectscan .
```

Scan a workspace folder:

```bash
./projectscan /path/to/projects
```

Show the full terminal audit:

```bash
./projectscan --audit /path/to/projects
```

Show project details:

```bash
./projectscan --details /path/to/projects
```

Export a report:

```bash
./projectscan export /path/to/projects --report open-source-readiness --format html
```

Available report types:

```text
summary
audit
details
portfolio
git
readiness
safety
readme
loc
git-hygiene
deps
open-source-readiness
external-tools
```

Available export formats:

```text
terminal
markdown
json
csv
html
```

Exports are written to `projectscan-exports/`.

## Configuration

ProjectScans looks for `.projectscan.toml` in the scanned root unless a config path is passed with `--config`.

Example:

```toml
ignore_patterns = ["archive-*", "tmp"]

[projects.my-app]
display_name = "My App"
label = "production-ready"
pinned = true
```

Use `.projectscanignore` for one ignore pattern per line:

```text
node_modules
dist
private-sandbox
```

## Running Tests

```bash
go test ./...
go vet ./...
go mod verify
```

## Repository Layout

```text
.
├── main.go
├── internal/
│   ├── audit/
│   ├── cli/
│   ├── export/
│   ├── scan/
│   └── tui/
├── README.md
└── LICENSE
```

## License

ProjectScans is released under the MIT License. See `LICENSE` for details.
