// Package data gère le chargement des fichiers JSON du corpus.
package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Corpus contient toutes les donnees chargees en memoire.
type Corpus struct {
	Glossaire   map[string]Concept    // id -> concept
	Fichiers    []Fichier
	Kernel      Kernel
	Embeddings  map[string][]float64  // nom_fichier -> vecteur
	SearchCache map[string]SearchEntry
	Modes       map[string]Mode
	Piliers     map[string]PilierConfig

	// Workspace
	Name     string
	BasePath string
}

// Kernel est le graphe de concepts.
type Kernel struct {
	Noeuds map[string]Concept    `json:"noeuds"`
	Liens  []KernelLien          `json:"liens"`
	Iles   map[string]IleRaw     `json:"iles"`
	Stats  KernelStats           `json:"stats"`
}

type IleRaw struct {
	Concepts []string `json:"concepts"`
}

type KernelLien struct {
	De   string `json:"de"`
	Vers string `json:"vers"`
	Type string `json:"type"`
}

type Ile struct {
	Nom      string   `json:"nom"`
	Concepts []string `json:"concepts"`
}

type KernelStats struct {
	Termes       int `json:"termes"`
	Liens        int `json:"liens"`
	CrossDomain  int `json:"cross_domain"`
	Bidirectionnel int `json:"bidirectionnel"`
}

// SearchEntry est une entrée du cache de recherche fulltext.
type SearchEntry struct {
	Nom    string `json:"name"`
	Pilier string `json:"pilier"`
	Text   string `json:"text"`
	Chars  int    `json:"chars"`
	Words  int    `json:"words"`
	Size   int    `json:"size"`
	Type   string `json:"type,omitempty"`
	Pages  int    `json:"pages_estimees,omitempty"`
	Chemin string `json:"chemin,omitempty"`
}

// GlossaireJSON est le format du fichier glossaire_pz.json.
type GlossaireJSON struct {
	Version string   `json:"version"`
	Termes  []GTerme `json:"termes"`
}

type GTerme struct {
	ID         string   `json:"id"`
	Terme      string   `json:"terme"`
	Pilier     string   `json:"pilier"`
	Definition string   `json:"definition_courte"`
	Aliases    []string `json:"aliases,omitempty"`
	Synonymes  []string `json:"synonymes,omitempty"`
	Relations  []string `json:"relations,omitempty"`
}

// Load charge un workspace chambre.
// Si basePath contient workspace.json, on l'utilise comme manifeste.
// Sinon, on cherche les fichiers dans le layout legacy (retrocompat PZ).
func Load(basePath string) (*Corpus, error) {
	c := &Corpus{
		BasePath:    basePath,
		Glossaire:   make(map[string]Concept),
		Embeddings:  make(map[string][]float64),
		SearchCache: make(map[string]SearchEntry),
		Modes:       DefaultModes(),
	}

	wsPath := filepath.Join(basePath, "workspace.json")
	if _, err := os.Stat(wsPath); err == nil {
		return c.loadFromWorkspace(wsPath)
	}

	// Legacy : layout chambre_reverberante d'origine
	return c.loadLegacy(basePath)
}

// loadFromWorkspace charge depuis un manifeste workspace.json.
func (c *Corpus) loadFromWorkspace(wsPath string) (*Corpus, error) {
	var ws Workspace
	if err := loadJSON(wsPath, &ws); err != nil {
		return nil, fmt.Errorf("workspace.json: %w", err)
	}

	c.Name = ws.Name
	base := filepath.Dir(wsPath)

	// Glossaire (requis)
	if ws.Files.Glossaire != "" {
		if err := c.loadGlossaire(filepath.Join(base, ws.Files.Glossaire)); err != nil {
			return nil, fmt.Errorf("glossaire: %w", err)
		}
	}

	// Kernel (optionnel)
	if ws.Files.Kernel != "" {
		if err := loadJSON(filepath.Join(base, ws.Files.Kernel), &c.Kernel); err != nil {
			fmt.Fprintf(os.Stderr, "warn: kernel: %v\n", err)
		}
	}

	// Index (requis)
	if ws.Files.Index != "" {
		if err := c.loadIndex(filepath.Join(base, ws.Files.Index)); err != nil {
			return nil, fmt.Errorf("index: %w", err)
		}
	}

	// Search cache
	if ws.Files.SearchCache != "" {
		if err := c.loadSearchCache(filepath.Join(base, ws.Files.SearchCache)); err != nil {
			fmt.Fprintf(os.Stderr, "warn: search cache: %v\n", err)
		}
	}

	// Embeddings
	if ws.Files.Embeddings != "" {
		if err := c.loadEmbeddings(filepath.Join(base, ws.Files.Embeddings)); err != nil {
			fmt.Fprintf(os.Stderr, "warn: embeddings: %v\n", err)
		}
	}

	// Config modes
	if len(ws.Config.Modes) > 0 {
		c.Modes = make(map[string]Mode, len(ws.Config.Modes))
		for name, mc := range ws.Config.Modes {
			c.Modes[name] = Mode{Nom: name, G: mc.G, NSpin: mc.NSpin}
		}
	}

	// Config piliers
	if len(ws.Config.Piliers) > 0 {
		c.Piliers = ws.Config.Piliers
	}

	return c, nil
}

// loadLegacy charge depuis le layout chambre_reverberante d'origine.
func (c *Corpus) loadLegacy(basePath string) (*Corpus, error) {
	c.Name = "Legacy"

	// Glossaire
	gPath := filepath.Join(basePath, "..", "glossaire_pz", "glossaire_pz.json")
	if err := c.loadGlossaire(gPath); err != nil {
		return nil, fmt.Errorf("glossaire: %w", err)
	}

	// Kernel
	kPath := filepath.Join(basePath, "kernel_pz.json")
	if err := loadJSON(kPath, &c.Kernel); err != nil {
		fmt.Fprintf(os.Stderr, "warn: kernel non charge: %v\n", err)
	}

	// Corpus index
	iPath := filepath.Join(basePath, "..", "corpus_indexer", "corpus_index.json")
	if err := c.loadIndex(iPath); err != nil {
		return nil, fmt.Errorf("index: %w", err)
	}

	// Search cache
	sPath := filepath.Join(basePath, "..", "corpus_indexer", "_search_cache.json")
	if err := c.loadSearchCache(sPath); err != nil {
		fmt.Fprintf(os.Stderr, "warn: search cache non charge: %v\n", err)
	}

	// Embeddings cache
	ePath := filepath.Join(basePath, "_embeddings_cache.json")
	if err := c.loadEmbeddings(ePath); err != nil {
		fmt.Fprintf(os.Stderr, "warn: embeddings non charges: %v\n", err)
	}

	return c, nil
}

func (c *Corpus) loadGlossaire(path string) error {
	var gj GlossaireJSON
	if err := loadJSON(path, &gj); err != nil {
		return err
	}
	for _, t := range gj.Termes {
		liens := make([]Lien, 0, len(t.Relations))
		for _, r := range t.Relations {
			liens = append(liens, Lien{Vers: r, Type: "associatif"})
		}
		syns := t.Synonymes
		if len(syns) == 0 {
			syns = t.Aliases
		}
		c.Glossaire[t.ID] = Concept{
			ID:         t.ID,
			Terme:      t.Terme,
			Pilier:     t.Pilier,
			Definition: t.Definition,
			Synonymes:  syns,
			Liens:      liens,
		}
	}
	return nil
}

// IndexFichier est le format brut du corpus_index.json.
type IndexFichier struct {
	Nom           string `json:"nom"`
	CheminRelatif string `json:"chemin_relatif"`
	Pilier        string `json:"pilier"`
	Type          string `json:"type"`
	Extension     string `json:"extension"`
	Taille        int    `json:"taille_octets"`
}

func (c *Corpus) loadIndex(path string) error {
	var idx struct {
		Fichiers []IndexFichier `json:"fichiers"`
	}
	if err := loadJSON(path, &idx); err != nil {
		return err
	}
	c.Fichiers = make([]Fichier, 0, len(idx.Fichiers))
	for _, f := range idx.Fichiers {
		c.Fichiers = append(c.Fichiers, Fichier{
			Nom:    f.Nom,
			Pilier: f.Pilier,
			Type:   f.Type,
			Chemin: f.CheminRelatif,
		})
	}
	return nil
}

func (c *Corpus) loadSearchCache(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var wrapper struct {
		Files map[string]SearchEntry `json:"files"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return err
	}
	c.SearchCache = wrapper.Files
	return nil
}

func (c *Corpus) loadEmbeddings(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &c.Embeddings)
}

func loadJSON(path string, dest interface{}) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}
