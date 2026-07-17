# GitSketch

**An interactive terminal Git branch and merge visualizer.**

Built with Go + [Bubbletea v2](https://charm.land/bubbletea/v2) / [Lipgloss v2](https://charm.land/lipgloss/v2).

---

## Features

- 🌳 **Curved Unicode Branch Graph** — Dynamic column-lane assignment visualizes branches, merges, and octopus merges with organic curved lines (`╭`, `╯`, `╰`, `╮`).
- 🎨 **Premium Dark-Mode TUI** — Split-pane layout with vibrant branch colors, ref badges, and metadata.
- ⚡ **Performance-First Design** — Single-pass topological parsing, lazy file loading, and viewport-only rendering to seamlessly handle 10,000+ commit histories.
- 📄 **Fullscreen Diff Viewer** — Press `Enter` to view a fullscreen, scrollable unified diff with smart line-by-line syntax coloring (additions, deletions, hunk headers).
- 🔍 **Interactive Regex Filtering** — Press `/` to filter the graph dynamically. The DAG automatically redraws on matching commits in real-time as you type.
- 🔀 **Interactive Branch Manager** — Press `b` to create or delete branches directly from any commit node.
- 🚀 **One-Click Checkout** — Press `c` to checkout the selected commit with a safe confirmation check.

---

## Installation

### 1-Command Installer (Recommended)
To download, select the package for your OS/Arch, and install GitSketch automatically:
```bash
curl -fsSL https://raw.githubusercontent.com/Wian47/GitSketch/master/install.sh | sh
```

### Via Go Package Registry
If you have Go installed, you can install the binary directly from GitHub:
```bash
go install github.com/Wian47/GitSketch@latest
```
*(Make sure your Go binary path, usually `~/go/bin`, is in your system `$PATH`)*

### Build from Source
To compile and install from a local clone:
```bash
git clone https://github.com/Wian47/GitSketch.git
cd GitSketch
go install .
```


---

## Usage

Run `gitsketch` inside **any** Git-initialized repository:
```bash
cd /path/to/any/git/repo
gitsketch
```

---

## Key Bindings

| Key | Context / Action |
|---|---|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `g` / `Home` | Jump to top of history |
| `G` / `End` | Jump to bottom of history |
| `PgUp` / `PgDn` | Page up / page down |
| `Enter` | **Normal Mode**: View fullscreen commit diff<br>**Diff Mode**: Return to DAG view |
| `/` | Open regex filter input |
| `b` | Open Branch Manager (create `c`, delete `d`, cancel `esc`) |
| `c` | Checkout selected commit (confirms `y`/`n`) |
| `Esc` / `q` / `Ctrl+C` | Cancel mode / Go back / Quit |

---

## Architecture

```
main.go                    → Entry point, repository validator
internal/git/parser.go     → Git DAG parser (custom formatting)
internal/git/commands.go   → Checkout, diff generation, branch commands
internal/graph/renderer.go → Topological DAG curve rendering engine
internal/tui/model.go      → TUI controller & viewport layout manager
internal/tui/styles.go     → Color palette & element style sheets
internal/tui/keys.go       → Hotkey declarations & templates
```

---
