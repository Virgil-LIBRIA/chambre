// chambre — moteur de résonance sémantique pour le corpus Point Zéro.
//
// Usage:
//   chambre              TUI interactive (défaut)
//   chambre search "q"   recherche one-shot
//   chambre serve        serveur HTTP (API REST)
//   chambre build        reconstruit le kernel
//   chambre pulse        maintenance (GC + sédimentation)
//   chambre version      affiche la version
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Virgil-LIBRIA/chambre/data"
	"github.com/Virgil-LIBRIA/chambre/search"
	"github.com/Virgil-LIBRIA/chambre/server"
	"github.com/Virgil-LIBRIA/chambre/tui"
	"github.com/Virgil-LIBRIA/chambre/vm"
)

const version = "0.1.0"

func main() {
	cmd := "tui"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "version", "--version", "-v":
		fmt.Printf("chambre %s\n", version)
	case "search", "s":
		runSearch()
	case "serve":
		runServe()
	case "pulse":
		runPulse()
	case "tui":
		runTUI()
	default:
		// Si pas de sous-commande connue, traiter comme recherche
		runSearchDirect(strings.Join(os.Args[1:], " "))
	}
}

func loadCorpus() (*data.Corpus, error) {
	// Cherche les données dans le dossier chambre_reverberante existant
	base := filepath.Join(filepath.Dir(os.Args[0]), "..", "chambre_reverberante")
	// Fallback : chemin absolu
	if _, err := os.Stat(filepath.Join(base, "kernel_pz.json")); err != nil {
		base = `C:\Users\VISION\Documents\Projets\chambre_reverberante`
	}
	return data.Load(base)
}

func runSearch() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: chambre search <query> [--mode default]")
		os.Exit(1)
	}
	query := os.Args[2]
	mode := "default"
	for i, arg := range os.Args[3:] {
		if arg == "--mode" && i+3+1 < len(os.Args) {
			mode = os.Args[i+3+1]
		}
	}
	runSearchDirect(query, mode)
}

func runSearchDirect(query string, modeOpt ...string) {
	mode := "default"
	if len(modeOpt) > 0 {
		mode = modeOpt[0]
	}

	corpus, err := loadCorpus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur chargement: %v\n", err)
		os.Exit(1)
	}

	engine := search.New(corpus)
	kvm := vm.New(corpus)

	start := time.Now()
	rev := engine.Reverberate(query, mode, 8, kvm)
	rev.Duree = data.Duration(time.Since(start))

	// Affichage CLI
	fmt.Printf("\n  %s  mode=%s  %s\n\n", query, mode, time.Duration(rev.Duree))

	// Glossaire
	if len(rev.Glossaire) > 0 {
		fmt.Print("  tau_0: ")
		for i, g := range rev.Glossaire {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(g.Terme)
		}
		fmt.Println()
	}

	// Résultats
	for i, r := range rev.Resultats {
		pilier := r.Fichier.Pilier
		score := fmt.Sprintf("%.3f", r.Score)
		fmt.Printf("  %d. [%s] %-40s %s  %s\n", i+1, pilier, r.Fichier.Nom, score, r.Source)
		if len(r.Concepts) > 0 {
			fmt.Printf("     concepts: %s\n", strings.Join(r.Concepts, ", "))
		}
		if r.Extrait != "" {
			ext := r.Extrait
			if len(ext) > 120 {
				ext = ext[:120] + "..."
			}
			fmt.Printf("     %s\n", ext)
		}
	}

	// Cross-domain
	if len(rev.CrossDomain) > 0 {
		fmt.Printf("\n  cross-domain: ")
		for i, c := range rev.CrossDomain {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s<->%s", c.De, c.Vers)
		}
		fmt.Println()
	}

	// Suggestions
	if len(rev.Suggestions) > 0 {
		fmt.Printf("  suggestions: %s\n", strings.Join(rev.Suggestions, ", "))
	}

	fmt.Println()
}

func runServe() {
	port := 5002
	if len(os.Args) > 2 {
		if p, err := fmt.Sscanf(os.Args[2], "%d", &port); p == 0 || err != nil {
			port = 5002
		}
	}
	corpus, err := loadCorpus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur chargement: %v\n", err)
		os.Exit(1)
	}
	srv := server.New(corpus, port)
	if err := srv.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "erreur serveur: %v\n", err)
		os.Exit(1)
	}
}

func runTUI() {
	corpus, err := loadCorpus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "erreur chargement: %v\n", err)
		os.Exit(1)
	}
	if err := tui.Run(corpus); err != nil {
		fmt.Fprintf(os.Stderr, "erreur TUI: %v\n", err)
		os.Exit(1)
	}
}

func runPulse() {
	fmt.Println("chambre pulse — pas encore implémenté")
}
