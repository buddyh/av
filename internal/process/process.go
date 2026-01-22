// Package process detects running AI agent processes
package process

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Session represents a running agent session
type Session struct {
	PID            int    `json:"pid"`
	Agent          string `json:"agent"` // "claude" or "codex"
	TTY            string `json:"tty"`
	RunningVersion string `json:"running_version"`
	Command        string `json:"command"`
	TmuxSession    string `json:"tmux_session,omitempty"`
	WorkingDir     string `json:"working_dir,omitempty"`
	HasActiveWork  bool   `json:"has_active_work,omitempty"`
}

// versionRegex extracts version from paths like /versions/2.1.14
var versionRegex = regexp.MustCompile(`/versions/(\d+\.\d+\.\d+)`)

// FindAgentSessions finds all running Claude and Codex sessions
func FindAgentSessions() []*Session {
	var sessions []*Session

	// Find claude processes
	claudeSessions := findProcesses("claude")
	sessions = append(sessions, claudeSessions...)

	// Find codex processes
	codexSessions := findProcesses("codex")
	sessions = append(sessions, codexSessions...)

	return sessions
}

func findProcesses(agent string) []*Session {
	// Use ps directly instead of pgrep (more reliable across platforms)
	// Find processes where command is exactly the agent name
	out, err := exec.Command("ps", "-eo", "pid=,tty=,command=").Output()
	if err != nil {
		return nil
	}

	var sessions []*Session
	seenTTYs := make(map[string]bool)

	// Match lines where command is exactly "claude" or "claude --flags"
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		pid := 0
		fmt.Sscanf(fields[0], "%d", &pid)
		tty := fields[1]
		command := strings.Join(fields[2:], " ")

		// Check if this is the agent we're looking for
		// Command should start with agent name (e.g., "claude" or "claude --continue")
		cmdParts := strings.Fields(command)
		if len(cmdParts) == 0 || cmdParts[0] != agent {
			continue
		}

		// Skip if no TTY or background process
		if tty == "??" || tty == "" {
			continue
		}

		// Skip duplicate TTYs (keep first/main process)
		if seenTTYs[tty] {
			continue
		}
		seenTTYs[tty] = true

		// Find running version from child process
		runningVersion := findRunningVersion(fmt.Sprintf("%d", pid), agent)

		sessions = append(sessions, &Session{
			PID:            pid,
			Agent:          agent,
			TTY:            tty,
			RunningVersion: runningVersion,
			Command:        command,
		})
	}

	return sessions
}

// claudeVersionRegex extracts version from Claude binary paths like /share/claude/versions/2.1.14
var claudeVersionRegex = regexp.MustCompile(`/share/claude/versions/(\d+\.\d+\.\d+)`)

// findRunningVersion looks at child processes to find the actual running binary version
func findRunningVersion(parentPID string, agent string) string {
	// Get child process commands
	out, err := exec.Command("pgrep", "-P", parentPID).Output()
	if err != nil {
		return ""
	}

	childPids := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, childPid := range childPids {
		if childPid == "" {
			continue
		}

		cmdOut, err := exec.Command("ps", "-o", "command=", "-p", childPid).Output()
		if err != nil {
			continue
		}

		cmd := string(cmdOut)

		// For Claude: look for /share/claude/versions/X.X.X
		if agent == "claude" {
			if matches := claudeVersionRegex.FindStringSubmatch(cmd); len(matches) > 1 {
				return matches[1]
			}
		}

		// For Codex: look for generic version pattern but only in codex paths
		if agent == "codex" && strings.Contains(cmd, "codex") {
			if matches := versionRegex.FindStringSubmatch(cmd); len(matches) > 1 {
				return matches[1]
			}
		}
	}

	return ""
}

// EnrichWithTmux adds tmux session info to sessions
func EnrichWithTmux(sessions []*Session, panes map[string]TmuxPane) {
	for _, s := range sessions {
		ttyPath := "/dev/" + s.TTY
		if pane, ok := panes[ttyPath]; ok {
			s.TmuxSession = pane.Session
			s.WorkingDir = pane.Path
		}
	}
}

// TmuxPane represents a tmux pane's info
type TmuxPane struct {
	TTY     string
	Session string
	Path    string
}

// ShortenPath converts /Users/buddy/repos/foo to ~/repos/foo
func ShortenPath(path string) string {
	home, _ := filepath.Abs(filepath.Join("~"))
	if strings.HasPrefix(path, "/Users/") {
		parts := strings.SplitN(path, "/", 4)
		if len(parts) >= 4 {
			return "~/" + parts[3]
		}
	}
	return strings.Replace(path, home, "~", 1)
}
