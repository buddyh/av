// Package tui provides interactive terminal UI components
package tui

import (
	"fmt"
	"strings"

	"github.com/buddyh/av/internal/process"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // green
	unselectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))   // gray
	disabledStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))   // dark gray
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))  // yellow
	headerStyle     = lipgloss.NewStyle().Bold(true)
	versionOld      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // red
	versionNew      = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // green
	activeWorkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // red
	helpStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))   // dark gray
)

// SessionItem represents a session in the picker
type SessionItem struct {
	Session        *process.Session
	Selected       bool
	CurrentVersion string // installed version to compare against
	Disabled       bool   // can't restart (has active work)
}

// PickerModel is the bubbletea model for session picker
type PickerModel struct {
	items      []SessionItem
	cursor     int
	submitted  bool
	cancelled  bool
	newVersion string
}

// NewPicker creates a new session picker
func NewPicker(sessions []*process.Session, installedClaude, installedCodex string) PickerModel {
	var items []SessionItem
	for _, s := range sessions {
		currentVersion := installedClaude
		if s.Agent == "codex" {
			currentVersion = installedCodex
		}
		// Only include sessions that need restart
		if s.RunningVersion != "" && s.RunningVersion != currentVersion && s.TmuxSession != "" {
			disabled := s.HasActiveWork
			items = append(items, SessionItem{
				Session:        s,
				Selected:       !disabled, // default selected unless disabled
				CurrentVersion: currentVersion,
				Disabled:       disabled,
			})
		}
	}
	return PickerModel{
		items:      items,
		newVersion: installedClaude,
	}
}

// Init implements tea.Model
func (m PickerModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			m.submitted = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ", "x":
			if len(m.items) > 0 && !m.items[m.cursor].Disabled {
				m.items[m.cursor].Selected = !m.items[m.cursor].Selected
			}
		case "a":
			// Select all (except disabled)
			for i := range m.items {
				if !m.items[i].Disabled {
					m.items[i].Selected = true
				}
			}
		case "n":
			// Select none
			for i := range m.items {
				m.items[i].Selected = false
			}
		}
	}
	return m, nil
}

// View implements tea.Model
func (m PickerModel) View() string {
	if len(m.items) == 0 {
		return "No sessions need restart.\n"
	}

	var b strings.Builder

	b.WriteString(headerStyle.Render("Select sessions to restart:"))
	b.WriteString("\n\n")

	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		checkbox := "[ ]"
		style := unselectedStyle
		if item.Disabled {
			checkbox = "[-]"
			style = disabledStyle
		} else if item.Selected {
			checkbox = "[x]"
			style = selectedStyle
		}

		path := shortenPath(item.Session.WorkingDir)
		if len(path) > 35 {
			path = "..." + path[len(path)-32:]
		}

		version := fmt.Sprintf("%s -> %s",
			versionOld.Render(item.Session.RunningVersion),
			versionNew.Render(item.CurrentVersion))

		status := ""
		if item.Disabled {
			status = activeWorkStyle.Render(" (busy)")
		}

		line := fmt.Sprintf("%s %s %-20s %-38s %s%s",
			cursor,
			checkbox,
			item.Session.TmuxSession,
			path,
			version,
			status)

		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate • space toggle • a all • n none • enter confirm • q quit"))
	b.WriteString("\n")

	return b.String()
}

// Cancelled returns true if user cancelled
func (m PickerModel) Cancelled() bool {
	return m.cancelled
}

// SelectedSessions returns the sessions that were selected
func (m PickerModel) SelectedSessions() []*process.Session {
	var selected []*process.Session
	for _, item := range m.items {
		if item.Selected {
			selected = append(selected, item.Session)
		}
	}
	return selected
}

func shortenPath(path string) string {
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "/Users/"); idx != -1 {
		parts := strings.SplitN(path[idx:], "/", 4)
		if len(parts) >= 4 {
			return "~/" + parts[3]
		}
	}
	return path
}
