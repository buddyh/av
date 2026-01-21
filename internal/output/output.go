// Package output handles terminal output formatting
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/buddyh/av/internal/process"
)

// Colors
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// Output handles formatted output
type Output struct {
	stdout  io.Writer
	stderr  io.Writer
	json    bool
	plain   bool
	noColor bool
}

// New creates a new Output
func New(stdout, stderr io.Writer) *Output {
	return &Output{
		stdout: stdout,
		stderr: stderr,
	}
}

// Configure sets output options
func (o *Output) Configure(jsonOut, plain, noColor bool) {
	o.json = jsonOut
	o.plain = plain
	o.noColor = noColor || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
}

func (o *Output) color(c, s string) string {
	if o.noColor || o.plain {
		return s
	}
	return c + s + colorReset
}

// JSON outputs data as JSON
func (o *Output) JSON(v any) error {
	enc := json.NewEncoder(o.stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Error prints an error message
func (o *Output) Error(err error) {
	prefix := o.color(colorRed, "error:")
	if o.plain {
		prefix = "[error]"
	}
	fmt.Fprintf(o.stderr, "%s %v\n", prefix, err)
}

// Warn prints a warning message
func (o *Output) Warn(msg string) {
	prefix := o.color(colorYellow, "warn:")
	if o.plain {
		prefix = "[warn]"
	}
	fmt.Fprintf(o.stderr, "%s %s\n", prefix, msg)
}

// Info prints an info message
func (o *Output) Info(msg string) {
	prefix := o.color(colorBlue, "info:")
	if o.plain {
		prefix = "[info]"
	}
	fmt.Fprintf(o.stdout, "%s %s\n", prefix, msg)
}

// Success prints a success message
func (o *Output) Success(msg string) {
	prefix := o.color(colorGreen, "ok:")
	if o.plain {
		prefix = "[ok]"
	}
	fmt.Fprintf(o.stdout, "%s %s\n", prefix, msg)
}

// PrintHeader prints a section header
func (o *Output) PrintHeader(title string) {
	if o.plain {
		fmt.Fprintf(o.stdout, "=== %s ===\n", title)
	} else {
		fmt.Fprintf(o.stdout, "%s\n", o.color(colorBold, title))
	}
}

// PrintVersion prints version info with update status
func (o *Output) PrintVersion(name, installed, latest string) {
	if installed == "" {
		installed = "not installed"
	}

	var status string
	if latest == "" {
		status = o.color(colorGray, "(couldn't fetch latest)")
	} else if installed == latest {
		if o.plain {
			status = "[current]"
		} else {
			status = o.color(colorGreen, "current")
		}
	} else if installed == "not installed" {
		status = ""
	} else {
		if o.plain {
			status = fmt.Sprintf("[update: %s]", latest)
		} else {
			status = o.color(colorYellow, fmt.Sprintf("update available: %s", latest))
		}
	}

	fmt.Fprintf(o.stdout, "  %-14s %s  %s\n", name, installed, status)
}

// PrintSessions prints the sessions table and returns count needing restart
func (o *Output) PrintSessions(sessions []*process.Session, claudeInstalled, codexInstalled string) int {
	if len(sessions) == 0 {
		fmt.Fprintln(o.stdout, "  No agent sessions running")
		return 0
	}

	// Count by agent
	claudeCount := 0
	codexCount := 0
	for _, s := range sessions {
		if s.Agent == "claude" {
			claudeCount++
		} else {
			codexCount++
		}
	}

	fmt.Fprintf(o.stdout, "  Found %d claude, %d codex session(s)\n\n", claudeCount, codexCount)

	// Header
	if o.plain {
		fmt.Fprintf(o.stdout, "  %-22s %-40s %-10s %s\n", "SESSION", "PATH", "VERSION", "STATUS")
	} else {
		fmt.Fprintf(o.stdout, "  %s\n", o.color(colorGray, fmt.Sprintf("%-22s %-40s %-10s %s", "SESSION", "PATH", "VERSION", "STATUS")))
	}

	needsRestart := 0

	for _, s := range sessions {
		session := s.TmuxSession
		if session == "" {
			session = fmt.Sprintf("pid:%d", s.PID)
		}

		path := shortenPath(s.WorkingDir)
		if path == "" {
			path = "-"
		}

		version := s.RunningVersion
		if version == "" {
			version = "?"
		}

		// Determine status
		currentVersion := claudeInstalled
		if s.Agent == "codex" {
			currentVersion = codexInstalled
		}

		var status string
		if version == currentVersion {
			if o.plain {
				status = "[current]"
			} else {
				status = o.color(colorGreen, "current")
			}
		} else if version == "?" {
			if o.plain {
				status = "[unknown]"
			} else {
				status = o.color(colorGray, "unknown")
			}
		} else {
			needsRestart++
			if s.TmuxSession == "" {
				if o.plain {
					status = "[outdated, no tmux]"
				} else {
					status = o.color(colorYellow, "outdated") + o.color(colorGray, " (no tmux)")
				}
			} else {
				if o.plain {
					status = "[restart needed]"
				} else {
					status = o.color(colorYellow, "restart needed")
				}
			}
		}

		// Truncate path if too long
		if len(path) > 38 {
			path = "..." + path[len(path)-35:]
		}

		fmt.Fprintf(o.stdout, "  %-22s %-40s %-10s %s\n", session, path, version, status)
	}

	return needsRestart
}

func shortenPath(path string) string {
	if path == "" {
		return ""
	}
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
