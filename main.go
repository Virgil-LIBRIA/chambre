// chambre — moteur de resonance semantique generique.
//
// Usage:
//   chambre [--workspace <path>]              TUI interactive (defaut)
//   chambre [--workspace <path>] search "q"   recherche one-shot
//   chambre [--workspace <path>] serve        serveur HTTP (API REST)
//   chambre [--workspace <path>] pulse        maintenance (GC + sedimentation)
//   chambre version                           affiche la version
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

const version = "0.2.0"

var workspacePath string

func main() {
	args := os.Args[1:]

	// Extraire --workspace avant la sous-commande
	var filtered []string
	for i := 0; i < len(args); i++ {
		if (args[i] == "--workspace" || args[i] == "-w") && i+1 < len(args) {
			workspacePath = args[i+1]
			i++ // skip value
		} else {
			filtered = append(filtered, args[i])
		}
	}

	cmd := "tui"
	if len(filtered) > 0 {
		cmd = filtered[0]
	}

	switch cmd {
	case "version", "--version", "-v":
		fmt.Printf("chambre %s\n", version)
	case "search", "s":
		runSearch(filtered)
	case "serve":
		runServe(filtered)
	case "pulse":
		runPulse()
	case "tui":
		runTUI()
	default:
		// Si pas de sous-commande connue, traiter comme recherche
		runSearchDirect(strings.Join(filtered, " "))
	}
}

func loadCorpus() (*data.Corpus, error) {
	// 1. Flag --workspace explicite
	if workspacePath != "" {
		return data.Load(workspacePath)
	}

	// 2. Variable d'environnement CHAMBRE_WORKSPACE
	if env := os.Getenv("CHAMBRE_WORKSPACE"); env != "" {
		return data.Load(env)
	}

	// 3. workspace.json dans le repertoire courant
	if _, err := os.Stat("workspace.json"); err == nil {
		cwd, _ := os.Getwd()
		return data.Load(cwd)
	}

	// 4. Dossier frere chambre_reverberante (legacy)
	base := filepath.Join(filepath.Dir(os.Args[0]), "..", "chambre_reverberante")
	if _, err := os.Stat(filepath.Join(base, "kernel_pz.json")); err == nil {
		return data.Load(base)
	}

	// 5. Fallback : chemin legacy absolu
	base = `C:\Users\VISION\Documents\Projets\chambre_reverberante`
	return data.Load(base)
}

func runSearch(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: chambre search <query> [--mode default]")
		os.Exit(1)
	}
	query := args[1]
	mode := "default"
	for i, arg := range args[2:] {
		if arg == "--mode" && i+2+1 < len(args) {
			mode = args[i+2+1]
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

func runServe(args []string) {
	port := 5002
	if len(args) > 1 {
		if p, err := fmt.Sscanf(args[1], "%d", &port); p == 0 || err != nil {
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
