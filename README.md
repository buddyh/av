# av - Agent Versions

Monitor and manage AI coding agent versions. Track running sessions of [Claude Code](https://github.com/anthropics/claude-code) and [OpenAI Codex](https://github.com/openai/codex), detect outdated instances, and restart them with one command.

## Why?

When Claude Code or Codex auto-updates, running sessions continue using the old binary in memory. `av` detects this version mismatch and can restart outdated sessions to pick up the new version.

## Features

- Detect installed versions of Claude Code and Codex
- Find all running sessions and their actual binary version
- Fetch latest versions from GitHub/npm
- Identify sessions running outdated versions
- Restart outdated tmux sessions with `--continue` flag
- Works without tmux (detection only, manual restart)
- JSON output for scripting
- Respects `NO_COLOR` and `--plain` for accessibility

## Installation

```bash
# From source
go install github.com/buddyspencer/av/cmd/av@latest

# Or clone and build
git clone https://github.com/buddyspencer/av.git
cd av
make install
```

## Usage

```bash
# Show all info (default)
av

# Skip fetching latest versions (faster)
av --no-fetch

# Check for updates only (no process scan)
av check

# Restart outdated sessions (tmux only)
av restart

# Restart all sessions
av restart --all

# JSON output
av --json
```

## Example Output

```
Installed Versions
  Claude Code    2.1.14  current
  Codex          0.80.0  update available: 0.88.0

Running Sessions
  Found 16 claude, 0 codex session(s)

  SESSION                PATH                                     VERSION    STATUS
  terminal-094510        ~/repos/aptus-swift                      2.1.11     restart needed
  terminal-165620        ~/repos/beatbox-storelocator             2.1.12     restart needed
  terminal-140609        ~/repos/cmc-lead-intel                   2.1.14     current
  pid:12345              -                                        2.1.11     outdated (no tmux)

3 session(s) need restart. Run `av restart` to update them.
```

## How It Works

1. **Installed version**: Reads symlink at `~/.local/bin/claude` or runs `claude --version`
2. **Running version**: Inspects child processes to find the actual binary path (e.g., `/versions/2.1.14`)
3. **Latest version**: Fetches from GitHub releases API (Claude) or npm registry (Codex)
4. **tmux integration**: Maps TTY to session name via `tmux list-panes`
5. **Restart**: Sends `Ctrl+C`, `exit`, then `claude --continue` via `tmux send-keys`

## Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--plain` | Plain text output (no colors/symbols) |
| `--no-color` | Disable colors |
| `--no-fetch` | Skip fetching latest versions |
| `--all` | (restart) Restart all sessions, even current ones |

## Requirements

- macOS or Linux
- Go 1.21+ (for building)
- tmux (optional, for session names and restart functionality)

## License

MIT
