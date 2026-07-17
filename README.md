# GitSketch

**An interactive terminal Git branch and merge visualizer.**

Built with Go + [Bubbletea v2](https://github.com/charmbracelet/bubbletea) / [Lipgloss v2](https://github.com/charmbracelet/lipgloss).

## Features

- 🌳 **ASCII Branch Graph** — Dynamic column-lane assignment visualizes branches, merges, and octopus merges
- 🎨 **Premium Dark-Mode TUI** — Split-pane layout with vibrant branch colors and ref badges
- ⚡ **Fast** — Single `git log` parse, only renders visible rows, lazy file loading
- 🔀 **Interactive Checkout** — Press `c` to checkout any commit with confirmation prompt
- 📜 **Scrollable** — Navigate 10,000+ commit histories without lag

## Installation

```bash
go install gitsketch@latest
```

Or build from source:

```bash
git clone https://github.com/youruser/gitsketch.git
cd gitsketch
go build -o gitsketch .
```

## Usage

Run inside any Git repository:

```bash
cd your-repo
gitsketch
```

## Key Bindings

| Key | Action |
|---|---|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `g` / `Home` | Jump to top |
| `G` / `End` | Jump to bottom |
| `PgUp` / `PgDn` | Page up/down |
| `c` | Checkout selected commit |
| `q` / `Ctrl+C` | Quit |

## Architecture

```
main.go                  → Entry point
internal/git/parser.go   → Git DAG parser (spawns git-plumbing commands)
internal/git/commands.go → Git checkout & file change loading
internal/graph/renderer.go → DAG → ASCII graph layout engine
internal/tui/model.go    → Bubbletea Model (state management)
internal/tui/styles.go   → Lipgloss style definitions
internal/tui/keys.go     → Key binding definitions
```

## License

MIT
