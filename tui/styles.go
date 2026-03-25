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

// pilierColors stocke les couleurs dynamiques chargees depuis le workspace.
var pilierColors map[string]lipgloss.Color

// SetPilierColors configure les couleurs depuis la config workspace.
func SetPilierColors(piliers map[string]string) {
	pilierColors = make(map[string]lipgloss.Color, len(piliers))
	for name, color := range piliers {
		pilierColors[name] = lipgloss.Color(color)
	}
}

// palette de secours si le workspace ne definit pas de couleurs
var defaultPilierColors = map[string]lipgloss.Color{
	"PZ":         colorOr,
	"ETH":        colorTurquoise,
	"GQ":         colorCobalt,
	"CONSCIENCE": colorPrune,
	"CSC":        colorPrune,
	"INT":        colorRouge,
	"OSS":        colorVert,
}

// pilierStyle retourne le style colore pour un pilier.
func pilierStyle(pilier string) lipgloss.Style {
	// 1. Couleurs dynamiques du workspace
	if pilierColors != nil {
		if fg, ok := pilierColors[pilier]; ok {
			return lipgloss.NewStyle().Foreground(fg)
		}
	}
	// 2. Couleurs par defaut
	if fg, ok := defaultPilierColors[pilier]; ok {
		return lipgloss.NewStyle().Foreground(fg)
	}
	// 3. Auto-couleur basee sur le hash du nom
	return lipgloss.NewStyle().Foreground(autoColor(pilier))
}

// autoColor genere une couleur deterministe pour un pilier inconnu.
func autoColor(name string) lipgloss.Color {
	// Palette de 8 couleurs distinctes pour les piliers auto-detectes
	palette := []string{
		"#c9a84c", "#4a9b8e", "#3d6fa8", "#7a4a8e",
		"#9e4a4a", "#5a9e5a", "#b87333", "#8b6c9e",
	}
	h := 0
	for _, c := range name {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return lipgloss.Color(palette[h%len(palette)])
}
