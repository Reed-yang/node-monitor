package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var helpEntries = []struct {
	key  string
	desc string
}{
	{"↑/↓, j/k", "Navigate nodes"},
	{"Enter", "Node detail view"},
	{"Esc, q", "Back / Exit"},
	{"Tab", "Toggle panel/compact"},
	{"p", "Toggle processes"},
	{"s", "Cycle sort order"},
	{"g", "Cycle node group"},
	{"/", "Search nodes"},
	{"?", "Toggle this help"},
}

// RenderHelp renders the keyboard shortcut help overlay.
func RenderHelp(width, height int) string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Keyboard Shortcuts"))
	lines = append(lines, "")

	for _, e := range helpEntries {
		key := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")).Width(12).Render(e.key)
		lines = append(lines, "  "+key+"  "+e.desc)
	}

	content := strings.Join(lines, "\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#555555")).
		Padding(1, 3).
		Width(40).
		Align(lipgloss.Left)

	rendered := style.Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, rendered)
}
