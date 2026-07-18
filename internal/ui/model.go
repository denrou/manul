// Package ui implements the terminal interface: a scrollable viewport
// rendering markdown through glamour with the user's theme.
package ui

import (
	_ "embed"
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

//go:embed welcome.md
var WelcomeMarkdown string

// maxContentWidth keeps long lines readable on wide terminals.
const maxContentWidth = 100

var statusBarStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "252"}).
	Background(lipgloss.AdaptiveColor{Light: "252", Dark: "236"}).
	Padding(0, 1)

// Model is the root bubbletea model: one document in a viewport.
type Model struct {
	title    string
	source   string
	viewport viewport.Model
	ready    bool
	err      error
}

func New(title, markdown string) Model {
	return Model{title: title, source: markdown}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		statusBarHeight := 1
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-statusBarHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - statusBarHeight
		}
		m.err = m.renderContent(msg.Width)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// renderContent re-renders the markdown for the given terminal width.
func (m *Model) renderContent(width int) error {
	wrap := min(width, maxContentWidth)
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrap),
	)
	if err != nil {
		return err
	}
	rendered, err := renderer.Render(m.source)
	if err != nil {
		return err
	}
	m.viewport.SetContent(rendered)
	return nil
}

func (m Model) View() string {
	if !m.ready {
		return "loading..."
	}
	if m.err != nil {
		return fmt.Sprintf("render error: %v\n\npress q to quit", m.err)
	}
	return m.viewport.View() + "\n" + m.statusBar()
}

func (m Model) statusBar() string {
	scroll := fmt.Sprintf("%3.0f%%", m.viewport.ScrollPercent()*100)
	left := statusBarStyle.Render("manul · " + m.title)
	right := statusBarStyle.Render(scroll + " · q quit")
	gap := m.viewport.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return left + lipgloss.NewStyle().Width(gap).Render("") + right
}
