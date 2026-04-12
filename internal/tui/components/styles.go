package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// btop++-inspired color palette (muted, layered)
var (
	ColorFg        = lipgloss.Color("#c9d1d9")
	ColorDim       = lipgloss.Color("#484f58")
	ColorBorder    = lipgloss.Color("#30363d")
	ColorMeterBg   = lipgloss.Color("#21262d")
	ColorAccent    = lipgloss.Color("#388bfd")
	ColorGreen     = lipgloss.Color("#2ea043")
	ColorYellow    = lipgloss.Color("#d29922")
	ColorOrange    = lipgloss.Color("#cf6e2a")
	ColorRed       = lipgloss.Color("#da3633")
	ColorSelection = lipgloss.Color("#e6edf3")
)

// Gradient defines a 3-color interpolation (start → mid → end).
type Gradient struct {
	Start [3]int
	Mid   [3]int
	End   [3]int
}

var (
	UtilGradient = Gradient{
		Start: [3]int{0x2e, 0xa0, 0x43}, // green
		Mid:   [3]int{0xd2, 0x99, 0x22}, // amber
		End:   [3]int{0xda, 0x36, 0x33}, // red
	}
	MemGradient = Gradient{
		Start: [3]int{0x38, 0x8b, 0xfd}, // blue
		Mid:   [3]int{0xd2, 0x99, 0x22}, // amber
		End:   [3]int{0xda, 0x36, 0x33}, // red
	}
)

// ColorAt returns an interpolated hex color for a value 0-100.
func (g Gradient) ColorAt(pct int) lipgloss.Color {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	var r, gn, b int
	if pct <= 50 {
		t := float64(pct) / 50.0
		r = g.Start[0] + int(t*float64(g.Mid[0]-g.Start[0]))
		gn = g.Start[1] + int(t*float64(g.Mid[1]-g.Start[1]))
		b = g.Start[2] + int(t*float64(g.Mid[2]-g.Start[2]))
	} else {
		t := float64(pct-50) / 50.0
		r = g.Mid[0] + int(t*float64(g.End[0]-g.Mid[0]))
		gn = g.Mid[1] + int(t*float64(g.End[1]-g.Mid[1]))
		b = g.Mid[2] + int(t*float64(g.End[2]-g.Mid[2]))
	}

	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, gn, b))
}

// NodeStatusIcon returns an icon based on average utilization.
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

// NodeBorderColor returns border color based on utilization.
func NodeBorderColor(avgUtil float64) lipgloss.Color {
	switch {
	case avgUtil > 80:
		return ColorRed
	case avgUtil > 50:
		return ColorYellow
	default:
		return ColorGreen
	}
}

// UtilColor returns a single color for a utilization percentage.
func UtilColor(percent float64) lipgloss.Color {
	return UtilGradient.ColorAt(int(percent))
}

// MemColor returns a single color for a memory percentage.
func MemColor(percent float64) lipgloss.Color {
	return MemGradient.ColorAt(int(percent))
}
