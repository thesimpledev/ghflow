<div align="center">

# ghflow

```
   _____ _    _ ______ _      ______          __
  / ____| |  | |  ____| |    / __ \ \        / /
 | |  __| |__| | |__  | |   | |  | \ \  /\  / /
 | | |_ |  __  |  __| | |   | |  | |\ \/  \/ /
 | |__| | |  | | |    | |___| |__| | \  /\  /
  \_____|_|  |_|_|    |______\____/   \/  \/
```

**A terminal dashboard for monitoring GitHub Actions workflows across multiple repositories.**

</div>

[![Built with Claude](https://img.shields.io/badge/Built%20with-Claude%20Code-blueviolet?style=flat-square)](https://claude.ai)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Bubble Tea](https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4?style=flat-square)](https://github.com/charmbracelet/bubbletea)
[![Vibe Coded](https://img.shields.io/badge/100%25-Vibe%20Coded-ff6b6b?style=flat-square)](#-vibe-coded)

---

## Features

- **3x2 Grid Layout** - Monitor up to 6 repos at a glance
- **Live Status Updates** - Auto-refreshes every 30 seconds
- **Vim Navigation** - hjkl to move, because arrow keys are for normies
- **Slash Commands** - /add, /remove, /save, /load, /new
- **Tab Completion** - Smart completions for paths, repos, and profiles
- **Profile Support** - Save and switch between different repo sets

## Installation

### Prerequisites

ghflow requires the [GitHub CLI](https://cli.github.com) for API authentication:

```bash
# macOS
brew install gh

# Linux (Debian/Ubuntu)
sudo apt install gh

# Then authenticate
gh auth login
```

### Install with Go

```bash
go install github.com/thesimpledev/ghflow@latest
```

Make sure `~/go/bin` is in your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Install from Source

```bash
git clone https://github.com/thesimpledev/ghflow.git
cd ghflow
go build -o ghflow .
sudo mv ghflow /usr/local/bin/  # optional: install system-wide
```

### Download Binary

Pre-built binaries are available on the [Releases](https://github.com/thesimpledev/ghflow/releases) page.

## Usage

```bash
ghflow
```

### Navigation

| Key | Action |
|-----|--------|
| h | Move left |
| j | Move down |
| k | Move up |
| l | Move right |
| Enter | Focus card / Select |
| Esc | Back / Unfocus |
| / | Open command input |
| q | Quit |

### Commands

| Command | Description |
|---------|-------------|
| /add path | Add a repo by navigating to its directory |
| /remove repo | Remove a repo from the dashboard |
| /save name | Save current repos as a named profile |
| /load profile | Load a saved profile |
| /new | Clear dashboard and start fresh |
| /refresh | Manually refresh all statuses |
| /quit | Exit the application |

### Profiles

Save different sets of repos for different contexts:

```
/save work        # Save current setup as "work"
/new              # Start fresh  
/load work        # Switch back to work repos
```

Profiles are stored in ~/.config/ghflow/profiles/

## Status Icons

| Icon | Meaning |
|------|---------|
| [ok] | Success (green) |
| [X] | Failed (red) |
| [~] | In Progress (orange) |
| [?] | Pending (gray) |
| [-] | Cancelled |

---

## Vibe Coded

This entire project was **vibe coded** - built through pure conversational collaboration with [Claude Code](https://claude.ai).

No boilerplate was copy-pasted. No Stack Overflow was consulted. Just vibes.

### The Process

1. Human: "I want a GitHub workflow dashboard CLI in Go"
2. Claude scaffolds the project
3. Human: "Add a 3x2 grid with slash commands"
4. Claude implements it
5. Human: "Tab should add / for directories"
6. Claude fixes it in 30 seconds
7. Ship it

### Stats

- **Time to MVP**: ~1 hour of conversation
- **Lines of code**: ~1500
- **Stack Overflow visits**: 0
- **Mass-produced boilerplate**: None
- **Vibes**: Immaculate

---

## Tech Stack

- [Go](https://go.dev) - Because we are not animals
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [GitHub CLI](https://cli.github.com) - API authentication

## License

MIT - Do whatever you want with it.

---

<p align="center">
  <i>Built with vibes and Claude Code</i>
</p>
