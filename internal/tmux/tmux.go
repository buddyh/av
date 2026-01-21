// Package tmux provides tmux session integration
package tmux

import (
	"fmt"
	"os/exec"
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

// RestartSession sends exit to a tmux session, waits, then runs claude --continue
func RestartSession(sessionName string, agent string) error {
	// Send Ctrl+C first to interrupt any running operation
	if err := sendKeys(sessionName, "C-c"); err != nil {
		return fmt.Errorf("failed to send Ctrl+C: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Send exit command
	if err := sendKeys(sessionName, "exit"); err != nil {
		return fmt.Errorf("failed to send exit: %w", err)
	}
	if err := sendKeys(sessionName, "Enter"); err != nil {
		return fmt.Errorf("failed to send Enter: %w", err)
	}

	// Wait for process to exit
	time.Sleep(2 * time.Second)

	// Start new session with --continue
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
