package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message structures for state notifications
type logMsg string
type statusMsg string
type doneMsg struct{}

type model struct {
	tasks      []TrackTask
	currentIdx int
	logs       []string
	status     string
	done       bool
	logChan    chan string
	errChan    chan error
}

func initialModel(tasks []TrackTask, logChan chan string, errChan chan error) model {
	return model{
		tasks:      tasks,
		currentIdx: 0,
		logs:       []string{"Initializing Medley engine..."},
		status:     "Ready",
		logChan:    logChan,
		errChan:    errChan,
	}
}

func (m model) Init() tea.Cmd {
	// Spin off asynchronous checking steps for log and error loops
	return tea.Batch(waitForLog(m.logChan), waitForErr(m.errChan))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case logMsg:
		timestamp := time.Now().Format("15:04:05")
		formattedLine := fmt.Sprintf("[%s] %s", timestamp, string(msg))

		m.logs = append(m.logs, formattedLine)
		if len(m.logs) > 10 {
			m.logs = m.logs[1:]
		}
		return m, waitForLog(m.logChan)

	case statusMsg:
		parts := strings.SplitN(string(msg), "|", 2)
		if len(parts) == 2 {
			var idx int
			if _, err := fmt.Sscanf(parts[0], "%d", &idx); err == nil {
				m.currentIdx = idx
			}
			m.status = parts[1]
		} else {
			m.status = string(msg)
		}
		return m, nil

	case doneMsg:
		m.done = true
		m.status = "Complete"
		return m, tea.Quit
	}

	return m, nil
}

func (m model) View() string {
	// Use lipgloss to construct clean visual terminal components
	var builder strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render
	statusStyle := lipgloss.NewStyle().Background(lipgloss.Color("5")).Foreground(lipgloss.Color("15")).Padding(0, 1).Render

	builder.WriteString(headerStyle(" AUDLER ── High-Fidelity Audio Provisioning Pipeline\n\n"))

	// Create a clear progression layout
	builder.WriteString(fmt.Sprintf(" Progress: [%d/%d] tasks evaluated\n", m.currentIdx, len(m.tasks)))
	builder.WriteString(fmt.Sprintf(" Engine Status: %s\n\n", statusStyle(m.status)))

	builder.WriteString(lipgloss.NewStyle().Underline(true).Render("Execution Terminal Streams:") + "\n")
	for _, l := range m.logs {
		builder.WriteString("  " + l + "\n")
	}

	builder.WriteString("\n Press [q] or [ctrl+c] to exit cleanly.\n")
	return builder.String()
}

// Asynchronous wrapper channels converting runtime activities into thread-safe TUI updates
func waitForLog(ch chan string) tea.Cmd {
	return func() tea.Msg {
		return logMsg(<-ch)
	}
}

func waitForErr(ch chan error) tea.Cmd {
	return func() tea.Msg {
		if err := <-ch; err != nil {
			return logMsg("Error encountered: " + err.Error())
		}
		return nil
	}
}
