package components

import "github.com/charmbracelet/lipgloss"

// Color thresholds matching the spec.
func UtilColor(percent float64) lipgloss.Color {
	switch {
	case percent < 30:
		return lipgloss.Color("#00FF00")
	case percent < 60:
		return lipgloss.Color("#FFFF00")
	case percent < 85:
		return lipgloss.Color("#FF8C00")
	default:
		return lipgloss.Color("#FF0000")
	}
}

func MemColor(percent float64) lipgloss.Color {
	switch {
	case percent < 50:
		return lipgloss.Color("#00FFFF")
	case percent < 75:
		return lipgloss.Color("#FFFF00")
	case percent < 90:
		return lipgloss.Color("#FF8C00")
	default:
		return lipgloss.Color("#FF0000")
	}
}

func NodeStatusIcon(avgUtil float64) string {
	switch {
	case avgUtil > 80:
		return "🔥"
	case avgUtil > 50:
		return "⚡"
	default:
		return "✓"
	}
}

func NodeBorderColor(avgUtil float64) lipgloss.Color {
	switch {
	case avgUtil > 80:
		return lipgloss.Color("#FF0000")
	case avgUtil > 50:
		return lipgloss.Color("#FFFF00")
	default:
		return lipgloss.Color("#00FF00")
	}
}
