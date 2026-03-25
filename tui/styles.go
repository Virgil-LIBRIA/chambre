// Package tui implémente l'interface terminal interactive.
package tui

import "github.com/charmbracelet/lipgloss"

// Palette Point Zéro
var (
	colorNoir      = lipgloss.Color("#0a0a0a")
	colorEncre     = lipgloss.Color("#111111")
	colorGraphite  = lipgloss.Color("#1e1e1e")
	colorCendre    = lipgloss.Color("#2c2c2c")
	colorBrume     = lipgloss.Color("#3a3a3a")
	colorArgent    = lipgloss.Color("#888888")
	colorIvoire    = lipgloss.Color("#e8e4dc")
	colorBlanc     = lipgloss.Color("#f5f2ee")
	colorOr        = lipgloss.Color("#c9a84c")
	colorOrPale    = lipgloss.Color("#d4b87a")
	colorTurquoise = lipgloss.Color("#4a9b8e")
	colorPrune     = lipgloss.Color("#7a4a8e")
	colorCobalt    = lipgloss.Color("#3d6fa8")
	colorRouge     = lipgloss.Color("#9e4a4a")
	colorVert      = lipgloss.Color("#5a9e5a")
)

// Styles réutilisables
var (
	styleTitle = lipgloss.NewStyle().
			Foreground(colorOr).
			Bold(true)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorArgent)

	styleDim = lipgloss.NewStyle().
			Foreground(colorBrume)

	styleResult = lipgloss.NewStyle().
			Foreground(colorIvoire)

	styleScore = lipgloss.NewStyle().
			Foreground(colorArgent)

	styleExtrait = lipgloss.NewStyle().
			Foreground(colorArgent).
			Italic(true)

	styleGlossaire = lipgloss.NewStyle().
			Foreground(colorOrPale)

	styleCrossDomain = lipgloss.NewStyle().
				Foreground(colorPrune)

	styleSuggestion = lipgloss.NewStyle().
			Foreground(colorTurquoise)

	stylePrompt = lipgloss.NewStyle().
			Foreground(colorOr)

	styleError = lipgloss.NewStyle().
			Foreground(colorRouge)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorBrume)

	styleVM = lipgloss.NewStyle().
		Foreground(colorTurquoise)

	styleHeader = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(colorCendre).
			Width(80)
)

// pilierStyle retourne le style coloré pour un pilier.
func pilierStyle(pilier string) lipgloss.Style {
	var fg lipgloss.Color
	switch pilier {
	case "PZ":
		fg = colorOr
	case "ETH":
		fg = colorTurquoise
	case "GQ":
		fg = colorCobalt
	case "CONSCIENCE", "CSC":
		fg = colorPrune
	case "INT":
		fg = colorRouge
	case "OSS":
		fg = colorVert
	default:
		fg = colorArgent
	}
	return lipgloss.NewStyle().Foreground(fg)
}
