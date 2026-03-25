// Package data définit les types partagés du système.
package data

import "time"

// Concept représente un terme du glossaire/kernel.
type Concept struct {
	ID         string   `json:"id"`
	Terme      string   `json:"terme"`
	Pilier     string   `json:"pilier"`
	Definition string   `json:"definition_courte"`
	Synonymes  []string `json:"synonymes,omitempty"`
	Liens      []Lien   `json:"liens,omitempty"`
	Fichiers   []string `json:"fichiers,omitempty"`
}

// Lien relie deux concepts.
type Lien struct {
	Vers  string  `json:"vers"`
	Type  string  `json:"type"` // constitutif, associatif, cross-domain, hierarchique
	Poids float64 `json:"poids,omitempty"`
}

// Fichier représente un document du corpus.
type Fichier struct {
	Nom    string `json:"nom"`
	Pilier string `json:"pilier"`
	Type   string `json:"type"`
	Pages  int    `json:"pages_estimees"`
	Chemin string `json:"chemin"`
}

// Resultat est un fichier scoré par le moteur de recherche.
type Resultat struct {
	Fichier  Fichier  `json:"fichier"`
	Score    float64  `json:"score"`
	Source   string   `json:"source"` // embedding, fulltext, nlp
	Concepts []string `json:"concepts_glossaire,omitempty"`
	Extrait  string   `json:"extrait,omitempty"`
}

// CrossLink décrit un pont inter-piliers.
type CrossLink struct {
	De     string `json:"de"`
	Vers   string `json:"vers"`
	Pilier [2]string `json:"piliers"`
}

// Reverberation est la réponse complète d'une recherche.
type Reverberation struct {
	Resultats   []Resultat  `json:"resultats"`
	Glossaire   []Concept   `json:"glossaire"`
	CrossDomain []CrossLink `json:"cross_domain"`
	VMContext   VMContext    `json:"vm_context"`
	Suggestions []string    `json:"suggestions,omitempty"`
	Mode        string      `json:"mode"`
	Duree       Duration    `json:"duree"`
}

// Duration wraps time.Duration pour un JSON lisible.
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
}

// VMContext contient le contexte chaud de la VM.
type VMContext struct {
	HotConcepts []string `json:"hot_concepts,omitempty"`
	HotPairs    []string `json:"hot_pairs,omitempty"`
	Spirales    []string `json:"spirales,omitempty"`
	Ticks       int      `json:"ticks"`
}

// Mode INTemple avec ses paramètres.
type Mode struct {
	Nom   string
	G     float64
	NSpin float64
}

// Modes prédéfinis du protocole INTemple v4.5.
var Modes = map[string]Mode{
	"silence_actif": {Nom: "silence_actif", G: 1.0, NSpin: 0.1},
	"dayz":          {Nom: "dayz", G: 1.0, NSpin: 0.2},
	"default":       {Nom: "default", G: 0.5, NSpin: 0.5},
	"translucide":   {Nom: "translucide", G: 0.5, NSpin: 0.5},
	"daisy":         {Nom: "daisy", G: 0.1, NSpin: 0.8},
	"vibratoire":    {Nom: "vibratoire", G: 0.3, NSpin: 0.9},
}

// Seuil calcule le seuil de pertinence selon G et n_spin.
func Seuil(g, nspin float64) float64 {
	s := 0.35 + 0.30*g - 0.15*nspin
	if s < 0.20 {
		return 0.20
	}
	if s > 0.75 {
		return 0.75
	}
	return s
}
