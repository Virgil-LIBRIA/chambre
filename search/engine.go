// Package search implémente le moteur de résonance sémantique.
package search

import (
	"math"
	"sort"
	"strings"

	"github.com/Virgil-LIBRIA/chambre/data"
	"github.com/Virgil-LIBRIA/chambre/nlp"
	"github.com/Virgil-LIBRIA/chambre/vm"
)

// Engine est le moteur de recherche.
type Engine struct {
	corpus    *data.Corpus
	stemmer   *nlp.Stemmer
	stemIndex map[string][]string // stem -> [noms de fichiers]
	docStems  map[string]map[string]int // nom -> {stem: count}
}

// New crée un nouveau moteur et pré-indexe les stems.
func New(corpus *data.Corpus) *Engine {
	e := &Engine{
		corpus:    corpus,
		stemmer:   nlp.NewStemmer(),
		stemIndex: make(map[string][]string),
		docStems:  make(map[string]map[string]int),
	}
	e.buildStemIndex()
	return e
}

// buildStemIndex pré-calcule les stems de chaque document.
func (e *Engine) buildStemIndex() {
	for nom, entry := range e.corpus.SearchCache {
		text := strings.ToLower(entry.Nom + " " + entry.Text)
		tokens := nlp.Tokenize(text)
		stems := e.stemmer.StemAll(tokens)

		counts := make(map[string]int, len(stems))
		for _, s := range stems {
			counts[s]++
		}
		e.docStems[nom] = counts

		seen := make(map[string]bool)
		for _, s := range stems {
			if !seen[s] {
				e.stemIndex[s] = append(e.stemIndex[s], nom)
				seen[s] = true
			}
		}
	}
}

// Reverberate exécute les 3 temps de la résonance.
func (e *Engine) Reverberate(query, mode string, topK int, kvm *vm.VM) data.Reverberation {
	m, ok := data.Modes[mode]
	if !ok {
		m = data.Modes["default"]
	}
	seuil := data.Seuil(m.G, m.NSpin)

	// TEMPS 1 — IMPULSION : préparer la requête
	qTokens := nlp.Tokenize(query)
	qStems := e.stemmer.StemAll(qTokens)

	// TEMPS 2 — REBOND : chercher dans le corpus
	var results []data.Resultat

	// 2a. Recherche par embeddings (si disponibles)
	if len(e.corpus.Embeddings) > 0 {
		results = e.searchEmbeddings(query, seuil)
	}

	// 2b. Recherche fulltext (toujours, complète les embeddings)
	ftResults := e.searchFulltext(qTokens, qStems, seuil)
	results = mergeResults(results, ftResults)

	// 2c. Boost VM : concepts chauds amplifient le score
	if kvm != nil {
		hot := kvm.HotConcepts(20)
		hotSet := toSet(hot)
		for i := range results {
			overlap := 0
			for _, c := range results[i].Concepts {
				if hotSet[c] {
					overlap++
				}
			}
			if overlap > 0 {
				boost := math.Min(float64(overlap)*0.05, 0.15)
				results[i].Score += boost
			}
		}
	}

	// Tri par score décroissant
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > topK {
		results = results[:topK]
	}

	// TEMPS 3 — INTERFÉRENCE : glossaire + cross-domain
	glossaire := e.findGlossaire(qTokens, qStems)
	crossDomain := e.findCrossDomain(glossaire)
	suggestions := e.findSuggestions(glossaire, kvm)

	// Annoter les résultats avec les concepts détectés
	for i := range results {
		results[i].Concepts = e.matchConcepts(results[i].Fichier.Nom, glossaire)
	}

	// Alimenter la VM
	if kvm != nil {
		conceptIDs := make([]string, 0, len(glossaire))
		for _, g := range glossaire {
			conceptIDs = append(conceptIDs, g.ID)
		}
		kvm.OnReverberate(query, conceptIDs, crossDomain)
	}

	return data.Reverberation{
		Resultats:   results,
		Glossaire:   glossaire,
		CrossDomain: crossDomain,
		Suggestions: suggestions,
		Mode:        mode,
		VMContext:    vmContext(kvm),
	}
}

// searchEmbeddings compare le vecteur de la requête aux embeddings cachés.
func (e *Engine) searchEmbeddings(query string, seuil float64) []data.Resultat {
	// Pour l'instant, pas d'appel Ollama — on utilise le cache existant
	// TODO: appel Ollama optionnel pour générer l'embedding de la query
	return nil
}

// searchFulltext cherche via l'index de stems pré-calculé.
func (e *Engine) searchFulltext(tokens, stems []string, seuil float64) []data.Resultat {
	if len(stems) == 0 {
		return nil
	}

	// Collecter les documents candidats via l'index inversé
	docScores := make(map[string]float64)
	for _, s := range stems {
		for _, nom := range e.stemIndex[s] {
			docScores[nom] += 1.0
		}
	}

	var results []data.Resultat
	nStems := float64(len(stems))

	for nom, hits := range docScores {
		entry, ok := e.corpus.SearchCache[nom]
		if !ok {
			continue
		}

		// Score = proportion de stems de la query trouvés dans le doc
		stemScore := hits / nStems

		// Bonus : correspondance directe du texte (tokens exacts)
		entryText := strings.ToLower(entry.Nom + " " + entry.Text)
		directHits := 0
		for _, t := range tokens {
			if strings.Contains(entryText, t) {
				directHits++
			}
		}
		directScore := float64(directHits) / float64(len(tokens))

		// Score TF-IDF simplifié : les stems rares valent plus
		tfBoost := 0.0
		docStemCounts := e.docStems[nom]
		totalDocs := float64(len(e.corpus.SearchCache))
		for _, s := range stems {
			if docStemCounts[s] > 0 {
				idf := math.Log(totalDocs / float64(len(e.stemIndex[s])+1))
				tfBoost += idf
			}
		}
		if nStems > 0 {
			tfBoost = tfBoost / (nStems * 5.0) // normaliser
			if tfBoost > 0.3 {
				tfBoost = 0.3
			}
		}

		score := 0.4*directScore + 0.3*stemScore + 0.3*tfBoost
		if score < seuil {
			continue
		}

		// Extrait : trouver la phrase contenant le premier token
		extrait := findExtrait(entryText, tokens[0], 200)

		results = append(results, data.Resultat{
			Fichier: data.Fichier{
				Nom:    nom,
				Pilier: entry.Pilier,
				Type:   entry.Type,
				Pages:  entry.Pages,
				Chemin: nom,
			},
			Score:   math.Round(score*1000) / 1000,
			Source:  "fulltext",
			Extrait: extrait,
		})
	}
	return results
}

// findExtrait extrait un passage autour du terme cherché.
func findExtrait(text, term string, maxLen int) string {
	idx := strings.Index(text, term)
	if idx < 0 {
		if len(text) > maxLen {
			return text[:maxLen] + "..."
		}
		return text
	}
	start := idx - 60
	if start < 0 {
		start = 0
	}
	end := idx + len(term) + 140
	if end > len(text) {
		end = len(text)
	}
	s := text[start:end]
	if start > 0 {
		s = "..." + s
	}
	if end < len(text) {
		s = s + "..."
	}
	return s
}

// findGlossaire détecte les concepts tau_0 dans la requête.
func (e *Engine) findGlossaire(tokens, stems []string) []data.Concept {
	var hits []data.Concept
	seen := make(map[string]bool)

	queryLower := strings.Join(tokens, " ")

	for id, concept := range e.corpus.Glossaire {
		if seen[id] {
			continue
		}

		// Match direct sur le terme
		termeLower := strings.ToLower(concept.Terme)
		if strings.Contains(queryLower, termeLower) {
			hits = append(hits, concept)
			seen[id] = true
			continue
		}

		// Match sur l'ID
		idNorm := strings.ReplaceAll(id, "-", " ")
		if strings.Contains(queryLower, idNorm) {
			hits = append(hits, concept)
			seen[id] = true
			continue
		}

		// Match stems
		termeStems := e.stemmer.StemAll(nlp.Tokenize(termeLower))
		overlap := stemOverlap(stems, termeStems)
		if len(termeStems) > 0 && float64(overlap)/float64(len(termeStems)) >= 0.7 {
			hits = append(hits, concept)
			seen[id] = true
		}
	}
	return hits
}

// findCrossDomain détecte les ponts inter-piliers.
func (e *Engine) findCrossDomain(glossaire []data.Concept) []data.CrossLink {
	var links []data.CrossLink
	piliers := make(map[string][]string)

	for _, g := range glossaire {
		piliers[g.Pilier] = append(piliers[g.Pilier], g.ID)
	}

	// Cross-domain = concepts de piliers différents liés entre eux
	for _, g := range glossaire {
		for _, l := range g.Liens {
			if target, ok := e.corpus.Glossaire[l.Vers]; ok {
				if target.Pilier != g.Pilier {
					links = append(links, data.CrossLink{
						De:     g.ID,
						Vers:   target.ID,
						Pilier: [2]string{g.Pilier, target.Pilier},
					})
				}
			}
		}
	}
	return links
}

// findSuggestions propose des concepts adjacents non explorés.
func (e *Engine) findSuggestions(glossaire []data.Concept, kvm *vm.VM) []string {
	seen := make(map[string]bool)
	for _, g := range glossaire {
		seen[g.ID] = true
	}

	var sugg []string
	for _, g := range glossaire {
		for _, l := range g.Liens {
			if !seen[l.Vers] {
				if _, ok := e.corpus.Glossaire[l.Vers]; ok {
					sugg = append(sugg, l.Vers)
					seen[l.Vers] = true
				}
			}
		}
	}

	if len(sugg) > 5 {
		sugg = sugg[:5]
	}
	return sugg
}

// matchConcepts retourne les concepts détectés dans un fichier.
func (e *Engine) matchConcepts(nom string, glossaire []data.Concept) []string {
	var matched []string
	for _, g := range glossaire {
		for _, f := range g.Fichiers {
			if f == nom {
				matched = append(matched, g.ID)
				break
			}
		}
	}
	return matched
}

// --- Utilitaires ---

func mergeResults(a, b []data.Resultat) []data.Resultat {
	seen := make(map[string]int)
	for i, r := range a {
		seen[r.Fichier.Nom] = i
	}
	for _, r := range b {
		if idx, ok := seen[r.Fichier.Nom]; ok {
			// Garder le meilleur score
			if r.Score > a[idx].Score {
				a[idx].Score = r.Score
				a[idx].Source = r.Source
			}
		} else {
			a = append(a, r)
		}
	}
	return a
}

func vmContext(kvm *vm.VM) data.VMContext {
	if kvm == nil {
		return data.VMContext{}
	}
	return data.VMContext{
		HotConcepts: kvm.HotConcepts(10),
		HotPairs:    kvm.HotPairs(10),
		Spirales:    kvm.Spirales(),
		Ticks:       kvm.Ticks(),
	}
}

func toSet[T comparable](items []T) map[T]bool {
	s := make(map[T]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

func stemOverlap(a, b []string) int {
	bSet := toSet(b)
	n := 0
	for _, s := range a {
		if bSet[s] {
			n++
		}
	}
	return n
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
