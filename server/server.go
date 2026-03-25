// Package server implémente le serveur HTTP REST.
// Compatible avec l'API de chambre.py (Python).
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Virgil-LIBRIA/chambre/data"
	"github.com/Virgil-LIBRIA/chambre/search"
	"github.com/Virgil-LIBRIA/chambre/vm"
)

// Server est le serveur HTTP.
type Server struct {
	engine *search.Engine
	vm     *vm.VM
	corpus *data.Corpus
	port   int
}

// New crée un nouveau serveur.
func New(corpus *data.Corpus, port int) *Server {
	return &Server{
		engine: search.New(corpus),
		vm:     vm.New(corpus),
		corpus: corpus,
		port:   port,
	}
}

// Run lance le serveur.
func (s *Server) Run() error {
	mux := http.NewServeMux()

	// CORS middleware
	handler := corsMiddleware(mux)

	// Routes
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /resonance", s.resonance)
	mux.HandleFunc("POST /reverberate", s.reverberate)
	mux.HandleFunc("GET /kernel", s.kernel)
	mux.HandleFunc("GET /kernel/concept/{id}", s.kernelConcept)
	mux.HandleFunc("GET /vm/status", s.vmStatus)
	mux.HandleFunc("GET /vm/hot", s.vmHot)
	mux.HandleFunc("POST /vm/gc", s.vmGC)
	mux.HandleFunc("GET /vm/recall/{what}", s.vmRecall)

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	fmt.Printf("  Chambre Reverberante — http://%s\n", addr)
	fmt.Printf("  %d termes | %d fichiers | %d embeddings\n\n",
		len(s.corpus.Glossaire), len(s.corpus.SearchCache), len(s.corpus.Embeddings))

	return http.ListenAndServe(addr, handler)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

// --- Handlers ---

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"chambre": "active",
		"status":  "ok",
		"runtime": "go",
	})
}

func (s *Server) resonance(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"glossaire_termes": len(s.corpus.Glossaire),
		"fichiers_indexes": len(s.corpus.Fichiers),
		"search_cache":     len(s.corpus.SearchCache),
		"embeddings_count": len(s.corpus.Embeddings),
		"vm_ticks":         s.vm.Ticks(),
	})
}

func (s *Server) reverberate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string  `json:"query"`
		Mode  string  `json:"mode"`
		TopK  int     `json:"top_k"`
		G     float64 `json:"G"`
		NSpin float64 `json:"n_spin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}
	if req.Query == "" {
		http.Error(w, `{"error":"query required"}`, 400)
		return
	}
	if req.Mode == "" {
		req.Mode = "default"
	}
	if req.TopK == 0 {
		req.TopK = 5
	}

	start := time.Now()
	rev := s.engine.Reverberate(req.Query, req.Mode, req.TopK, s.vm)
	rev.Duree = data.Duration(time.Since(start))

	writeJSON(w, rev)
}

func (s *Server) kernel(w http.ResponseWriter, r *http.Request) {
	full := r.URL.Query().Get("full") == "1"

	resp := map[string]interface{}{
		"stats": map[string]int{
			"termes":    len(s.corpus.Glossaire),
			"liens":     len(s.corpus.Kernel.Liens),
			"iles":      len(s.corpus.Kernel.Iles),
			"fichiers":  len(s.corpus.Fichiers),
		},
	}

	if full {
		resp["noeuds"] = s.corpus.Glossaire
		resp["liens"] = s.corpus.Kernel.Liens
		resp["iles"] = s.corpus.Kernel.Iles
	}

	// Piliers
	piliers := make(map[string]int)
	for _, c := range s.corpus.Glossaire {
		piliers[c.Pilier]++
	}
	resp["piliers"] = piliers

	writeJSON(w, resp)
}

func (s *Server) kernelConcept(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	concept, ok := s.corpus.Glossaire[id]
	if !ok {
		http.Error(w, `{"error":"concept not found"}`, 404)
		return
	}
	writeJSON(w, concept)
}

func (s *Server) vmStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"ticks":        s.vm.Ticks(),
		"hot_concepts": len(s.vm.HotConcepts(100)),
		"hot_pairs":    len(s.vm.HotPairs(100)),
		"spirales":     s.vm.Spirales(),
	})
}

func (s *Server) vmHot(w http.ResponseWriter, r *http.Request) {
	n := 20
	if ns := r.URL.Query().Get("n"); ns != "" {
		if v, err := strconv.Atoi(ns); err == nil {
			n = v
		}
	}
	writeJSON(w, map[string]interface{}{
		"hot_concepts": s.vm.HotConcepts(n),
		"hot_pairs":    s.vm.HotPairs(n),
		"spirales":     s.vm.Spirales(),
	})
}

func (s *Server) vmGC(w http.ResponseWriter, r *http.Request) {
	removed := s.vm.GC()
	writeJSON(w, map[string]interface{}{
		"removed": removed,
		"ticks":   s.vm.Ticks(),
	})
}

func (s *Server) vmRecall(w http.ResponseWriter, r *http.Request) {
	what := r.PathValue("what")
	switch strings.ToLower(what) {
	case "hot_concepts":
		writeJSON(w, s.vm.HotConcepts(50))
	case "hot_pairs":
		writeJSON(w, s.vm.HotPairs(50))
	case "spirales":
		writeJSON(w, s.vm.Spirales())
	default:
		http.Error(w, `{"error":"unknown recall type"}`, 400)
	}
}
