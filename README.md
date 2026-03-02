# xssh

[中文文档](README_CN.md)

A multiplexed SSH terminal in your terminal. Connect to multiple servers side by side, broadcast commands to all of them, and scroll through history — all from a single window.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+">
  <img src="https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/License-MIT-blue" alt="License">
</p>

## Features

- **Up to 9 panes** — SSH sessions and local shells in an auto-arranged grid (1×1 → 3×3)
- **Broadcast input** — Real-time keystroke forwarding to multiple panes simultaneously, with selective targeting
- **Scroll & search** — Color-preserving scrollback buffer with regex search
- **SSH config aware** — Fuzzy-searchable host selector reads `~/.ssh/config` aliases
- **Session groups** — Save and load named multi-host configurations
- **Shared borders** — Space-efficient single-line dividers between panes (toggleable)
- **Auto-reconnect** — Dropped SSH connections are retried automatically
- **Mouse support** — Click to focus, scroll to browse history
- **Zoom** — Full-screen any single pane, then restore the grid

## Install

```bash
go install github.com/xssh/xssh@latest
```

Or build from source:

```bash
git clone https://github.com/xssh/xssh.git
cd xssh
go build -o xssh .
```

## Quick Start

```bash
# Interactive host selector (reads ~/.ssh/config)
xssh

# Two local shells side by side
xssh - -

# Connect to three servers
xssh web1 web2 db1

# Mix local and remote
xssh - user@192.168.1.10 staging

# Load a saved group
xssh -g production
```

## Keyboard Shortcuts

All shortcuts use the **Ctrl+\\** prefix key, then the action key.

| Keys | Action |
|------|--------|
| `Ctrl+\ 1-9` | Focus pane 1-9 |
| `Ctrl+\ h/j/k/l` | Focus left/down/up/right |
| `Ctrl+\ z` | Zoom / restore current pane |
| `Ctrl+\ x` | Close current pane |
| `Ctrl+\ r` | Reconnect current pane |
| `Ctrl+\ R` | Reconnect all panes |
| `Ctrl+\ b` | Broadcast input (real-time, toggle) |
| `Ctrl+\ m` | Select which panes receive broadcast |
| `Ctrl+\ [` | Enter scroll mode |
| `Ctrl+\ e` | Add a new pane |
| `Ctrl+\ s` | Save current session as group |
| `Ctrl+\ ?` | Show help overlay |
| `Ctrl+\ \` | Send literal Ctrl+\ to session |

### Scroll Mode

| Keys | Action |
|------|--------|
| `↑/k` `↓/j` | Scroll up / down |
| `PgUp` `PgDn` | Half-page scroll |
| `g` / `G` | Jump to top / bottom |
| `/` | Search |
| `n` / `N` | Next / previous match |
| `q` `Esc` | Exit scroll mode |

## CLI Flags

```
xssh [flags] [targets...]

Targets:
  -                     Local shell
  user@host             SSH connection
  alias                 SSH config alias

Flags:
  -h, --help            Show help
  -g, --group NAME      Load a saved group
  --save NAME targets…  Save targets as a named group
  --list-groups         List saved groups
  --list-hosts          List SSH config hosts
  --borders MODE        Border style: shared (default) or full
```

## Border Modes

**Shared** (default — `--borders shared`):
```
╭──────┬──────╮
│ pane1 │ pane2 │
├──────┼──────┤
│ pane3 │ pane4 │
├──────┴──────┤
│ input bar    │
╰──────────────╯
```

**Full** (`--borders full`):
```
╭──────╮╭──────╮
│ pane1 ││ pane2 │
╰──────╯╰──────╯
╭──────╮╭──────╮
│ pane3 ││ pane4 │
╰──────╯╰──────╯
╭──────────────╮
│ input bar    │
╰──────────────╯
```

## Configuration

Config file: `~/.xssh/config.yaml`

```yaml
general:
  scrollback_lines: 5000
  reconnect_attempts: 3
  reconnect_interval: 5s
  ssh_timeout: 10s

groups:
  production:
    - web1
    - web2
    - db-master
  staging:
    - staging-web
    - staging-db
```

## Layout Grid

| Panes | Grid |
|-------|------|
| 1 | 1×1 |
| 2 | 1×2 |
| 3-4 | 2×2 |
| 5-6 | 3×2 |
| 7-9 | 3×3 |

## Requirements

- Go 1.25+
- A terminal with 256-color and mouse support
- macOS or Linux

## License

MIT
