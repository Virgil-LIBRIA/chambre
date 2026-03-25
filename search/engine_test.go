package search

import (
	"fmt"
	"testing"

	"github.com/Virgil-LIBRIA/chambre/data"
	"github.com/Virgil-LIBRIA/chambre/vm"
)

func TestReverberate(t *testing.T) {
	corpus, err := data.Load(`C:\Users\VISION\Documents\Projets\chambre_reverberante`)
	if err != nil {
		t.Fatalf("chargement corpus: %v", err)
	}

	t.Logf("Glossaire: %d termes", len(corpus.Glossaire))
	t.Logf("Fichiers: %d", len(corpus.Fichiers))
	t.Logf("SearchCache: %d", len(corpus.SearchCache))
	t.Logf("Embeddings: %d", len(corpus.Embeddings))

	engine := New(corpus)
	kvm := vm.New(corpus)

	queries := []struct {
		q    string
		mode string
	}{
		{"intrajection", "default"},
		{"torsion vectorielle", "default"},
		{"conscience nodale", "daisy"},
		{"organisation systemique spontanee", "default"},
		{"machine temporelle fatalisation", "dayz"},
	}

	for _, q := range queries {
		t.Run(q.q, func(t *testing.T) {
			rev := engine.Reverberate(q.q, q.mode, 5, kvm)

			fmt.Printf("\n=== %s (mode=%s) ===\n", q.q, q.mode)

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

			for i, r := range rev.Resultats {
				fmt.Printf("  %d. [%s] %s  %.3f  %s\n", i+1, r.Fichier.Pilier, r.Fichier.Nom, r.Score, r.Source)
			}

			if len(rev.Resultats) == 0 {
				t.Errorf("aucun résultat pour %q", q.q)
			}
			if len(rev.Glossaire) == 0 {
				t.Logf("WARN: pas de glossaire match pour %q", q.q)
			}

			fmt.Printf("  cross-domain: %d | suggestions: %d\n", len(rev.CrossDomain), len(rev.Suggestions))
		})
	}

	t.Logf("VM ticks: %d", kvm.Ticks())
}
