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
	// Get PIDs of main agent processes (not child processes like MCP servers)
	out, err := exec.Command("pgrep", "-f", fmt.Sprintf("^%s", agent)).Output()
	if err != nil {
		// Also try without anchor for symlinked binaries
		out, err = exec.Command("pgrep", "-x", agent).Output()
		if err != nil {
			return nil
		}
	}

	var sessions []*Session
	seenTTYs := make(map[string]bool)

	pids := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, pidStr := range pids {
		if pidStr == "" {
			continue
		}

		// Get process info
		info, err := exec.Command("ps", "-o", "pid=,tty=,command=", "-p", pidStr).Output()
		if err != nil {
			continue
		}

		fields := strings.Fields(strings.TrimSpace(string(info)))
		if len(fields) < 3 {
			continue
		}

		pid := 0
		fmt.Sscanf(fields[0], "%d", &pid)
		tty := fields[1]
		command := strings.Join(fields[2:], " ")

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
		runningVersion := findRunningVersion(pidStr, agent)

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
