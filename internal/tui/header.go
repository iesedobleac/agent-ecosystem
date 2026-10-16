package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/iesedobleac/iesedoble-ai/internal/installer"
)

// logo is <IESEDOBLE^AI /> in a pixel font, like a self-closing tag, with AI
// as a boxed exponent raised above the name.
var logo = []string{
	`                                                        ╭─────────╮`,
	`                                                        │ ▄▀█ ▀█▀ │`,
	`  ▄ ▀█▀ █▀▀▀▀ ▄▀▀▀▀ █▀▀▀▀ █▀▀▀▄ ▄▀▀▀▄ █▀▀▀▄ █     █▀▀▀▀ │ █▀█ ▄█▄ │   █ ▄`,
	`▄▀   █  █▄▄▄  ▀▄▄▄  █▄▄▄  █   █ █   █ █▄▄▄▀ █     █▄▄▄  ╰─────────╯  █   ▀▄`,
	` ▀▄  █  █         █ █     █   █ █   █ █   █ █     █                 ▄▀  ▄▀`,
	`    ▀▀▀ ▀▀▀▀▀ ▀▀▀▀  ▀▀▀▀▀ ▀▀▀▀   ▀▀▀  ▀▀▀▀  ▀▀▀▀▀ ▀▀▀▀▀             ▀`,
}

// The logo's columns: the tag marks in grey, "IESEDOBLE" with the gradient
// and the AI exponent in green.
const (
	nameStart = 4
	nameEnd   = 55
	aiEnd     = 67
)

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

var green = rgb{0x9E, 0xCE, 0x6A}

func paint(c rgb, r rune) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c.hex())).Render(string(r))
}

// header is the banner, sized to the terminal width (0 when not known yet).
func header(width int) string {
	version := installer.Version
	if version != "" && version[0] >= '0' && version[0] <= '9' {
		version = "v" + version
	}
	tagline := "agents, skills and MCP for your AI assistants"
	logoWidth := 0
	for _, line := range logo {
		logoWidth = max(logoWidth, utf8.RuneCountInString(line))
	}
	if width != 0 && width < logoWidth+4+margin {
		return title.Render("iesedoble-ai") + " " + subtle.Render(version) + "\n" + subtle.Render(tagline)
	}

	var b strings.Builder
	b.WriteString("\n\n") // breathing room above the logo
	for _, line := range logo {
		b.WriteString("  ")
		for col, r := range []rune(line) {
			switch {
			case r == ' ':
				b.WriteRune(r)
			case col >= nameStart && col < nameEnd:
				b.WriteString(paint(colorAt(float64(col-nameStart)/float64(nameEnd-nameStart-1)), r))
			case col >= nameEnd && col < aiEnd && strings.ContainsRune("█▀▄", r):
				b.WriteString(paint(green, r))
			case col >= nameEnd && col < aiEnd: // the box around AI
				b.WriteString(paint(mix(green, rgb{0x1A, 0x1B, 0x26}, 0.35), r))
			default: // < and />
				b.WriteString(subtle.Render(string(r)))
			}
		}
		b.WriteString("\n")
	}

	// "agents, skills … ──── v1.2.3", as wide as the logo.
	rule := max(3, logoWidth-utf8.RuneCountInString(tagline)-len(version)-2)
	b.WriteString("\n  " + subtle.Render(tagline+" "+strings.Repeat("─", rule)+" "+version))
	return b.String()
}
