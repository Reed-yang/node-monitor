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
	{"Enter", "Open node detail"},
	{"Click", "Select / open node"},
	{"Esc", "Close detail / exit"},
	{"q", "Quit"},
	{"Tab", "Focus grid / panel"},
	{"p", "Toggle processes"},
	{"s", "Cycle sort order"},
	{"g", "Cycle node group"},
	{"/", "Search nodes"},
	{"?", "Toggle this help"},
}

// RenderHelp renders the keyboard shortcut help overlay.
func RenderHelp(width, height int) string {
	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorFg).Render("Keyboard Shortcuts"))
	lines = append(lines, "")

	for _, e := range helpEntries {
		key := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent).Width(12).Render(e.key)
		desc := lipgloss.NewStyle().Foreground(ColorFg).Render(e.desc)
		lines = append(lines, "  "+key+"  "+desc)
	}

	content := strings.Join(lines, "\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 3).
		Width(42).
		Align(lipgloss.Left)

	rendered := style.Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, rendered,
		lipgloss.WithWhitespaceBackground(ColorBg))
}
