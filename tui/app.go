package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Virgil-LIBRIA/chambre/data"
	"github.com/Virgil-LIBRIA/chambre/search"
	"github.com/Virgil-LIBRIA/chambre/vm"
)

// view identifie le panneau actif.
type view int

const (
	viewSearch view = iota
	viewConcept
	viewVM
)

// Model est le modèle principal de la TUI.
type Model struct {
	engine   *search.Engine
	vm       *vm.VM
	corpus   *data.Corpus
	input    textinput.Model
	viewport viewport.Model

	// État
	currentView view
	mode        string
	results     *data.Reverberation
	query       string
	searching   bool
	err         error
	history     []string

	// Modes dynamiques
	modeNames []string

	// Concept explorer
	concept     *data.Concept
	conceptPath []string

	// Dimensions
	width  int
	height int
	ready  bool
}

type searchDoneMsg struct {
	rev   data.Reverberation
	query string
	dur   time.Duration
}

type errMsg struct{ err error }

// New cree le modele TUI.
func New(corpus *data.Corpus) Model {
	ti := textinput.New()
	ti.Placeholder = "Interroger le corpus..."
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60
	ti.PromptStyle = stylePrompt
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorIvoire)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorBrume)
	ti.Prompt = "  > "

	// Configurer les couleurs piliers depuis le workspace
	if len(corpus.Piliers) > 0 {
		colors := make(map[string]string, len(corpus.Piliers))
		for name, cfg := range corpus.Piliers {
			if cfg.Color != "" {
				colors[name] = cfg.Color
			}
		}
		if len(colors) > 0 {
			SetPilierColors(colors)
		}
	}

	// Charger les noms de modes depuis le corpus
	modeNames := make([]string, 0, len(corpus.Modes))
	for name := range corpus.Modes {
		modeNames = append(modeNames, name)
	}
	// Trier pour un ordre stable
	sort.Strings(modeNames)

	return Model{
		engine:    search.New(corpus),
		vm:        vm.New(corpus),
		corpus:    corpus,
		input:     ti,
		mode:      "default",
		modeNames: modeNames,
	}
}

// Run lance la TUI.
func Run(corpus *data.Corpus) error {
	m := New(corpus)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.currentView != viewSearch {
				m.currentView = viewSearch
				m.concept = nil
				return m, nil
			}
			if m.input.Value() != "" {
				m.input.SetValue("")
				return m, nil
			}
			return m, tea.Quit

		case "enter":
			if m.currentView == viewSearch && m.input.Value() != "" {
				return m, m.doSearch(m.input.Value())
			}

		case "tab":
			// Cycle modes (dynamiques depuis le workspace)
			for i, mode := range m.modeNames {
				if mode == m.mode {
					m.mode = m.modeNames[(i+1)%len(m.modeNames)]
					break
				}
			}
			return m, nil

		case "ctrl+k":
			m.currentView = viewVM
			return m, nil

		case "1", "2", "3", "4", "5", "6", "7", "8":
			if m.results != nil && m.currentView == viewSearch {
				idx := int(msg.String()[0] - '1')
				if idx < len(m.results.Resultats) {
					// Rechercher le fichier
					nom := m.results.Resultats[idx].Fichier.Nom
					return m, m.doSearch(strings.TrimSuffix(nom, ".docx"))
				}
			}

		case "ctrl+g":
			// Naviguer vers le premier concept glossaire
			if m.results != nil && len(m.results.Glossaire) > 0 {
				c := m.results.Glossaire[0]
				m.concept = &c
				m.currentView = viewConcept
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-6)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
		}

	case searchDoneMsg:
		m.searching = false
		m.results = &msg.rev
		m.query = msg.query
		m.history = append(m.history, msg.query)
		content := m.renderResults(msg.rev, msg.dur)
		m.viewport.SetContent(content)
		m.viewport.GotoTop()
		return m, nil

	case errMsg:
		m.searching = false
		m.err = msg.err
		return m, nil
	}

	// Mettre à jour les sous-composants
	if m.currentView == viewSearch {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) doSearch(query string) tea.Cmd {
	m.searching = true
	return func() tea.Msg {
		start := time.Now()
		rev := m.engine.Reverberate(query, m.mode, 8, m.vm)
		dur := time.Since(start)
		return searchDoneMsg{rev: rev, query: query, dur: dur}
	}
}

func (m Model) View() string {
	if !m.ready {
		return "  Chargement..."
	}

	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Input
	b.WriteString(m.input.View())
	b.WriteString("\n")

	// Mode bar
	b.WriteString(m.renderModeBar())
	b.WriteString("\n")

	// Content
	switch m.currentView {
	case viewSearch:
		if m.searching {
			b.WriteString(styleDim.Render("\n  Reverberation..."))
		} else if m.results != nil {
			b.WriteString(m.viewport.View())
		} else {
			b.WriteString(m.renderWelcome())
		}
	case viewConcept:
		b.WriteString(m.renderConceptView())
	case viewVM:
		b.WriteString(m.renderVMView())
	}

	// Footer help
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

// --- Renderers ---

func (m Model) renderHeader() string {
	name := "Chambre Reverberante"
	if m.corpus.Name != "" && m.corpus.Name != "Legacy" {
		name = m.corpus.Name
	}
	title := styleTitle.Render(name)
	stats := styleSubtitle.Render(fmt.Sprintf(
		"%d termes | %d fichiers | %d embeddings",
		len(m.corpus.Glossaire),
		len(m.corpus.SearchCache),
		len(m.corpus.Embeddings),
	))
	return fmt.Sprintf("  %s  %s", title, stats)
}

func (m Model) renderModeBar() string {
	var parts []string
	for _, mode := range m.modeNames {
		label := mode
		if mode == m.mode {
			parts = append(parts, styleGlossaire.Render("["+label+"]"))
		} else {
			parts = append(parts, styleDim.Render(" "+label+" "))
		}
	}
	return "  " + strings.Join(parts, " ")
}

func (m Model) renderResults(rev data.Reverberation, dur time.Duration) string {
	var b strings.Builder

	// Glossaire
	if len(rev.Glossaire) > 0 {
		b.WriteString("\n  ")
		b.WriteString(styleGlossaire.Render("tau_0: "))
		var terms []string
		for _, g := range rev.Glossaire {
			terms = append(terms, g.Terme)
		}
		b.WriteString(styleGlossaire.Render(strings.Join(terms, ", ")))
		b.WriteString(styleDim.Render(fmt.Sprintf("  %s", dur.Round(time.Millisecond))))
		b.WriteString("\n")
	}

	// Résultats
	for i, r := range rev.Resultats {
		b.WriteString("\n")
		pilier := pilierStyle(r.Fichier.Pilier).Render(fmt.Sprintf("[%s]", r.Fichier.Pilier))
		score := styleScore.Render(fmt.Sprintf("%.3f", r.Score))
		num := styleDim.Render(fmt.Sprintf("  %d.", i+1))

		// Nom court
		nom := r.Fichier.Nom
		if len(nom) > 55 {
			nom = nom[:52] + "..."
		}
		name := styleResult.Render(nom)

		b.WriteString(fmt.Sprintf("%s %s %s  %s\n", num, pilier, name, score))

		// Extrait
		if r.Extrait != "" {
			ext := r.Extrait
			if len(ext) > 120 {
				ext = ext[:117] + "..."
			}
			b.WriteString(styleExtrait.Render("     " + ext))
			b.WriteString("\n")
		}
	}

	// Cross-domain
	if len(rev.CrossDomain) > 0 {
		b.WriteString("\n")
		b.WriteString(styleCrossDomain.Render("  cross-domain: "))
		var links []string
		for _, c := range rev.CrossDomain {
			links = append(links, c.De+"<->"+c.Vers)
		}
		if len(links) > 5 {
			links = links[:5]
			links = append(links, "...")
		}
		b.WriteString(styleCrossDomain.Render(strings.Join(links, ", ")))
		b.WriteString("\n")
	}

	// Suggestions
	if len(rev.Suggestions) > 0 {
		b.WriteString("\n")
		b.WriteString(styleSuggestion.Render("  suggestions: " + strings.Join(rev.Suggestions, ", ")))
		b.WriteString("\n")
	}

	// VM context
	hot := m.vm.HotConcepts(5)
	if len(hot) > 0 {
		b.WriteString("\n")
		b.WriteString(styleVM.Render("  vm chauds: " + strings.Join(hot, ", ")))
		b.WriteString(styleDim.Render(fmt.Sprintf("  [%d ticks]", m.vm.Ticks())))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderWelcome() string {
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(styleDim.Render("                        ◯\n\n"))
	b.WriteString(styleSubtitle.Render("          L'echo attend l'impulsion.\n\n"))
	b.WriteString(styleDim.Render("     Tapez un concept, une question, un mot-cle\n"))
	b.WriteString(styleDim.Render("     du corpus Point Zero.\n"))
	return b.String()
}

func (m Model) renderConceptView() string {
	if m.concept == nil {
		return ""
	}
	c := m.concept
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styleTitle.Render("  " + c.Terme))
	b.WriteString("\n")
	b.WriteString(pilierStyle(c.Pilier).Render("  " + c.Pilier))
	b.WriteString("\n\n")

	if c.Definition != "" {
		b.WriteString(styleResult.Render("  " + c.Definition))
		b.WriteString("\n\n")
	}

	if len(c.Liens) > 0 {
		b.WriteString(styleSubtitle.Render("  Liens:\n"))
		for i, l := range c.Liens {
			if i >= 15 {
				b.WriteString(styleDim.Render(fmt.Sprintf("  ... +%d\n", len(c.Liens)-15)))
				break
			}
			b.WriteString(fmt.Sprintf("  %s %s\n",
				styleDim.Render(l.Type),
				styleResult.Render(l.Vers),
			))
		}
	}

	return b.String()
}

func (m Model) renderVMView() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styleTitle.Render("  Kernel VM"))
	b.WriteString(styleDim.Render(fmt.Sprintf("  [%d ticks]", m.vm.Ticks())))
	b.WriteString("\n\n")

	hot := m.vm.HotConcepts(20)
	if len(hot) > 0 {
		b.WriteString(styleSubtitle.Render("  Concepts chauds:\n"))
		for _, c := range hot {
			b.WriteString(styleVM.Render("    " + c + "\n"))
		}
	}

	pairs := m.vm.HotPairs(10)
	if len(pairs) > 0 {
		b.WriteString("\n")
		b.WriteString(styleSubtitle.Render("  Paires:\n"))
		for _, p := range pairs {
			b.WriteString(styleCrossDomain.Render("    " + p + "\n"))
		}
	}

	spirales := m.vm.Spirales()
	if len(spirales) > 0 {
		b.WriteString("\n")
		b.WriteString(styleError.Render("  Spirales detectees:\n"))
		for _, s := range spirales {
			b.WriteString(styleError.Render("    " + s + "\n"))
		}
	}

	return b.String()
}

func (m Model) renderHelp() string {
	return styleHelp.Render("  enter:chercher  tab:mode  ctrl+g:glossaire  ctrl+k:vm  1-8:resultat  esc:retour  ctrl+c:quitter")
}
