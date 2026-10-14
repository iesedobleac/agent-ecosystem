package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/iesedobleac/agent-ecosystem/internal/installer"
)

// logo is "AGENT" in block letters; the subtitle below spells out the rest.
var logo = []string{
	` █████╗  ██████╗ ███████╗███╗   ██╗████████╗`,
	`██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝`,
	`███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║   `,
	`██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║   `,
	`██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║   `,
	`╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝   `,
}

// gradient runs left to right across the logo: blue, purple, cyan.
var gradient = []rgb{{0x7A, 0xA2, 0xF7}, {0xBB, 0x9A, 0xF7}, {0x7D, 0xCF, 0xFF}}

type rgb struct{ r, g, b float64 }

func (c rgb) hex() string { return fmt.Sprintf("#%02X%02X%02X", int(c.r), int(c.g), int(c.b)) }

func mix(a, b rgb, t float64) rgb {
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

// colorAt is the gradient color at position t in [0, 1].
func colorAt(t float64) rgb {
	seg := t * float64(len(gradient)-1)
	i := min(int(seg), len(gradient)-2)
	return mix(gradient[i], gradient[i+1], seg-float64(i))
}

// header is the banner, sized to the terminal width (0 when not known yet).
func header(width int) string {
	version := installer.Version
	if version != "" && version[0] >= '0' && version[0] <= '9' {
		version = "v" + version
	}
	tagline := subtle.Render("agents, skills and MCP for your AI assistants")
	logoWidth := utf8.RuneCountInString(logo[0])
	if width != 0 && width < logoWidth+4 {
		return title.Render("agent-ecosystem") + " " + subtle.Render(version) + "\n" + tagline
	}

	var b strings.Builder
	for _, line := range logo {
		b.WriteString("  ")
		col := 0
		for _, r := range line {
			c := colorAt(float64(col) / float64(logoWidth-1))
			switch {
			case r == ' ':
				b.WriteRune(r)
			case r == '█':
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c.hex())).Render(string(r)))
			default: // the box-drawing shadow, darker for depth
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(mix(c, rgb{0x1A, 0x1B, 0x26}, 0.55).hex())).Render(string(r)))
			}
			col++
		}
		b.WriteString("\n")
	}

	// " E C O S Y S T E M  ──── v1.2.3", as wide as the logo.
	name := " E C O S Y S T E M "
	rule := max(3, logoWidth-utf8.RuneCountInString(name)-len(version)-1)
	var sub strings.Builder
	for i, r := range name {
		c := colorAt(float64(i) / float64(logoWidth-1))
		sub.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(c.hex())).Render(string(r)))
	}
	b.WriteString("  " + sub.String() + subtle.Render(strings.Repeat("─", rule)+" "+version) + "\n")
	b.WriteString("  " + tagline)
	return b.String()
}
