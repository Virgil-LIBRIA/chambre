// Package vm implémente la Kernel VM — mémoire structurelle vivante.
// 7 opcodes, 3 couches mémoire (écume/creux/océan), GC.
package vm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Virgil-LIBRIA/chambre/data"
)

const (
	maxEcume   = 50
	decayRate  = 0.95
	sedimentAt = 20
	memFile    = "kernel_memory.json"
)

// VM est la machine virtuelle du kernel.
type VM struct {
	memory  Memory
	ticks   int
	corpus  *data.Corpus
	memPath string
}

// Memory contient les 3 couches.
type Memory struct {
	Ecume Ecume `json:"ecume"`
	Creux Creux `json:"creux"`
	Ocean Ocean `json:"ocean"`
}

// Ecume : mémoire court terme.
type Ecume struct {
	Requetes []string `json:"requetes"`
	Concepts []string `json:"concepts"`
	Anchors  []string `json:"anchors"`
}

// Creux : mémoire moyen terme.
type Creux struct {
	ConceptFreq  map[string]float64 `json:"concept_freq"`
	PairesChauds map[string]float64 `json:"paires_chaudes"`
	Tunnels      map[string]int     `json:"tunnels_actifs"`
	Spirales     []string           `json:"spirales"`
}

// Ocean : mémoire long terme.
type Ocean struct {
	Volutions []Volution `json:"volutions"`
	Insights  []string   `json:"insights"`
}

type Volution struct {
	Date    string `json:"date"`
	Type    string `json:"type"`
	Details string `json:"details"`
}

// New crée une VM, charge la mémoire persistante si elle existe.
func New(corpus *data.Corpus) *VM {
	memPath := memFile
	if corpus != nil && corpus.BasePath != "" {
		memPath = filepath.Join(corpus.BasePath, memFile)
	}

	v := &VM{
		corpus:  corpus,
		memPath: memPath,
		memory: Memory{
			Creux: Creux{
				ConceptFreq:  make(map[string]float64),
				PairesChauds: make(map[string]float64),
				Tunnels:      make(map[string]int),
			},
		},
	}
	v.load()
	return v
}

// OnReverberate est appelé après chaque recherche.
func (v *VM) OnReverberate(query string, concepts []string, cross []data.CrossLink) {
	v.ticks++

	// QUERY
	v.memory.Ecume.Requetes = append(v.memory.Ecume.Requetes, query)
	if len(v.memory.Ecume.Requetes) > maxEcume {
		v.memory.Ecume.Requetes = v.memory.Ecume.Requetes[1:]
	}

	// ANCHOR
	for _, c := range concepts {
		v.memory.Ecume.Concepts = append(v.memory.Ecume.Concepts, c)
		v.memory.Ecume.Anchors = append(v.memory.Ecume.Anchors, c)
		v.memory.Creux.ConceptFreq[c] += 1.0
	}
	if len(v.memory.Ecume.Concepts) > maxEcume {
		v.memory.Ecume.Concepts = v.memory.Ecume.Concepts[len(v.memory.Ecume.Concepts)-maxEcume:]
	}

	// TRAVERSE — paires de concepts co-détectés
	for i := 0; i < len(concepts); i++ {
		for j := i + 1; j < len(concepts); j++ {
			pair := concepts[i] + "+" + concepts[j]
			v.memory.Creux.PairesChauds[pair] += 1.0
		}
	}

	// TUNNEL — cross-domain
	for _, cl := range cross {
		key := cl.De + "<>" + cl.Vers
		v.memory.Creux.Tunnels[key]++
	}

	// Détection de spirales
	v.detectSpirales()

	// SEDIMENT — tous les N ticks
	if v.ticks%sedimentAt == 0 {
		v.sediment()
	}

	v.save()
}

// HotConcepts retourne les N concepts les plus chauds.
func (v *VM) HotConcepts(n int) []string {
	type kv struct {
		k string
		v float64
	}
	var items []kv
	for k, val := range v.memory.Creux.ConceptFreq {
		items = append(items, kv{k, val})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].v > items[j].v })
	out := make([]string, 0, n)
	for i := 0; i < n && i < len(items); i++ {
		out = append(out, items[i].k)
	}
	return out
}

// HotPairs retourne les N paires les plus traversées.
func (v *VM) HotPairs(n int) []string {
	type kv struct {
		k string
		v float64
	}
	var items []kv
	for k, val := range v.memory.Creux.PairesChauds {
		items = append(items, kv{k, val})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].v > items[j].v })
	out := make([]string, 0, n)
	for i := 0; i < n && i < len(items); i++ {
		out = append(out, items[i].k)
	}
	return out
}

// Spirales retourne les patterns répétitifs détectés.
func (v *VM) Spirales() []string {
	return v.memory.Creux.Spirales
}

// Ticks retourne le nombre de ticks écoulés.
func (v *VM) Ticks() int {
	return v.ticks
}

// GC applique le decay et nettoie la mémoire.
func (v *VM) GC() (removed int) {
	for k, val := range v.memory.Creux.ConceptFreq {
		val *= decayRate
		if val < 0.1 {
			delete(v.memory.Creux.ConceptFreq, k)
			removed++
		} else {
			v.memory.Creux.ConceptFreq[k] = val
		}
	}
	for k, val := range v.memory.Creux.PairesChauds {
		val *= decayRate
		if val < 0.1 {
			delete(v.memory.Creux.PairesChauds, k)
			removed++
		} else {
			v.memory.Creux.PairesChauds[k] = val
		}
	}
	v.save()
	return
}

// --- Interne ---

func (v *VM) detectSpirales() {
	// Détection simple : si un concept apparaît >5 fois dans l'écume
	freq := make(map[string]int)
	for _, c := range v.memory.Ecume.Concepts {
		freq[c]++
	}
	v.memory.Creux.Spirales = nil
	for c, n := range freq {
		if n >= 5 {
			v.memory.Creux.Spirales = append(v.memory.Creux.Spirales, c)
		}
	}
}

func (v *VM) sediment() {
	// Cristalliser les concepts avec freq > 3 dans ocean.insights
	for c, f := range v.memory.Creux.ConceptFreq {
		if f > 3.0 {
			found := false
			for _, ins := range v.memory.Ocean.Insights {
				if ins == c {
					found = true
					break
				}
			}
			if !found {
				v.memory.Ocean.Insights = append(v.memory.Ocean.Insights, c)
			}
		}
	}
}

func (v *VM) load() {
	raw, err := os.ReadFile(v.memPath)
	if err != nil {
		return
	}
	var saved struct {
		Memory Memory `json:"memory"`
		Ticks  int    `json:"ticks"`
	}
	if json.Unmarshal(raw, &saved) == nil {
		v.memory = saved.Memory
		v.ticks = saved.Ticks
		// Assurer les maps
		if v.memory.Creux.ConceptFreq == nil {
			v.memory.Creux.ConceptFreq = make(map[string]float64)
		}
		if v.memory.Creux.PairesChauds == nil {
			v.memory.Creux.PairesChauds = make(map[string]float64)
		}
		if v.memory.Creux.Tunnels == nil {
			v.memory.Creux.Tunnels = make(map[string]int)
		}
	}
}

func (v *VM) save() {
	saved := struct {
		Memory Memory `json:"memory"`
		Ticks  int    `json:"ticks"`
		Saved  string `json:"saved_at"`
	}{
		Memory: v.memory,
		Ticks:  v.ticks,
		Saved:  time.Now().Format(time.RFC3339),
	}
	raw, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(v.memPath, raw, 0644)
}
