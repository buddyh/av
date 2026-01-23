// Package tmux provides tmux session integration
package tmux

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/buddyh/av/internal/process"
)

// IsAvailable checks if tmux is running
func IsAvailable() bool {
	_, err := exec.Command("tmux", "list-sessions").Output()
	return err == nil
}

// GetPanes returns a map of TTY -> TmuxPane for all tmux panes
func GetPanes() map[string]process.TmuxPane {
	panes := make(map[string]process.TmuxPane)

	out, err := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_tty}:#{session_name}:#{pane_current_path}").Output()
	if err != nil {
		return panes
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		panes[parts[0]] = process.TmuxPane{
			TTY:     parts[0],
			Session: parts[1],
			Path:    parts[2],
		}
	}

	return panes
}

// CapturePane captures the last N lines from a tmux pane
func CapturePane(sessionName string, lines int) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-S", fmt.Sprintf("-%d", lines)).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Patterns for detecting active work
var (
	// Active operation - shows "ctrl+c to interrupt"
	ctrlCPattern = regexp.MustCompile(`ctrl\+c to interrupt`)
	// Running indicator with ellipsis (… or ...)
	runningPattern = regexp.MustCompile(`Running[….]+`)
	// Active spinner patterns
	spinnerPattern = regexp.MustCompile(`[⏺✻].*(?:Thinking|Reading|Writing|Manifesting|Editing)[….]*`)
)

// HasActiveWork checks if the session has background tasks running
func HasActiveWork(sessionName string) bool {
	content, err := CapturePane(sessionName, 20)
	if err != nil {
		return false
	}

	// Check for active work indicators - only the last 20 lines
	// The "ctrl+c to interrupt" is the clearest signal of active work
	if ctrlCPattern.MatchString(content) {
		return true
	}
	if runningPattern.MatchString(content) {
		return true
	}
	if spinnerPattern.MatchString(content) {
		return true
	}
	return false
}

// RestartSession sends exit to a tmux session, waits, then resumes with --continue
func RestartSession(sessionName string, agent string) error {
	// Send Ctrl+C multiple times to:
	// 1. Interrupt any running operation
	// 2. Clear any suggested text in the prompt
	for i := 0; i < 3; i++ {
		if err := sendKeys(sessionName, "C-c"); err != nil {
			return fmt.Errorf("failed to send Ctrl+C: %w", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Clear the input line (Ctrl+U) to remove any partial text
	if err := sendKeys(sessionName, "C-u"); err != nil {
		return fmt.Errorf("failed to send Ctrl+U: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Send exit command
	if err := sendKeys(sessionName, "exit"); err != nil {
		return fmt.Errorf("failed to send exit: %w", err)
	}
	if err := sendKeys(sessionName, "Enter"); err != nil {
		return fmt.Errorf("failed to send Enter: %w", err)
	}

	// Wait for process to exit
	time.Sleep(2 * time.Second)

	// Build resume command - use --continue which resumes the most recent
	// session in the current directory (handled correctly by Claude)
	var cmd string
	switch agent {
	case "claude":
		cmd = "claude --continue"
	case "codex":
		cmd = "codex --continue"
	default:
		return fmt.Errorf("unknown agent: %s", agent)
	}

	if err := sendKeys(sessionName, cmd); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}
	if err := sendKeys(sessionName, "Enter"); err != nil {
		return fmt.Errorf("failed to send Enter: %w", err)
	}

	return nil
}

func sendKeys(sessionName string, keys string) error {
	_, err := exec.Command("tmux", "send-keys", "-t", sessionName, keys).Output()
	return err
}
