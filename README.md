# Chambre — Moteur de Resonance Semantique (Go)

> **Strate : S2 Operationnel** | **Version : 0.2.0** | **Depot : github.com/Virgil-LIBRIA/chambre (public)**

> **Note sur la famille "Chambre"** — ce repo contient la reimplementation
> Go generique (binaire unique portable). Une version Python active avec les
> embeddings du corpus source et le bridge MCP pour Claude Code est
> maintenue separement, en repo prive, pour des raisons de confidentialite
> des donnees d'ingestion. Les deux implementations sont compatibles sur
> les endpoints HTTP REST (`/reverberate`, `/intempt`, `/health`).
>
> Le `mcp_server.py` present dans ce repo est une archive de l'etat au
> 2026-04-04 du bridge MCP — la version active (avec guard singleton socket)
> vit avec le reste du code Python dans le repo prive.

## Role

Version Go de la Chambre Reverberante — binaire unique (~11 Mo) remplacant l'ecosysteme Python+Node.js+Flask. Objectif SaaS : moteur generique ou Point Zero est le premier client, pas le code source.

## Architecture

```
chambre/
├── main.go              Point d'entree (--workspace flag, CHAMBRE_WORKSPACE env)
├── data/
│   ├── loader.go        Chargement corpus (workspace.json ou legacy auto-detecte)
│   ├── types.go         Types + DefaultModes() (6 modes INTemple)
│   └── workspace.go     Format workspace.json generique (manifest)
├── search/
│   └── engine.go        Moteur (cosine similarity + fulltext fallback)
├── tui/
│   ├── app.go           Interface terminal (bubbletea, modes dynamiques)
│   └── styles.go        Couleurs piliers dynamiques (workspace → default → auto-hash)
├── server/
│   └── server.go        API REST HTTP (compatible chambre.py)
└── pipeline/            Pipeline Python (lourd, offline, une fois)
    ├── ingest.py        Extraction texte (python-docx, pypdf)
    └── analyze.py       Auto-glossaire spaCy (fr_core_news_md, TF-IDF, k-means)
```

## Workspace auto-genere (pz-workspace/)

```
pz-workspace/
├── workspace.json       Manifeste (nom, config modes/piliers, chemins)
├── glossaire.json       80 termes auto-detectes (spaCy noun_chunks)
├── kernel.json          1881 liens (cooccurrence inter-documents)
├── embeddings.json      151 vecteurs spaCy 300d
├── index.json           153 fichiers indexes (VAULT VISION)
└── search_cache.json    Texte extrait par fichier
```

## Commandes

```bash
# Recherche
chambre search "intrajection" --mode daisy --top 5

# Interface terminal
chambre tui

# Serveur HTTP REST
chambre serve --port 5002

# Workspace specifique
chambre --workspace ./mon-corpus/ tui

# Pipeline (preparation workspace)
python pipeline/ingest.py "chemin/corpus/" --output ./workspace/
python pipeline/analyze.py ./workspace/
```

## Plan SaaS

| # | Phase | Statut | Livrable |
|---|-------|--------|----------|
| 0 | Decouplage Go | Termine | --workspace, modes/piliers dynamiques |
| 1 | Pipeline ingestion | Termine | 153 fichiers VAULT indexes |
| 2 | Auto-glossaire | Termine | 80 termes, 1881 liens, 151 vecteurs |
| 3 | Auto-kernel | Integre P2 | Clustering k-means + cooccurrence |
| 4 | Workspace complet | Termine | PZ = premier workspace, format valide |
| 5 | Frontend web | Futur | Upload → pipeline → chambre → UI |

## Ameliorations pipeline en attente

- Deduplication Voyage en Silence (16 versions polluent les clusters)
- Normalisation des accents (concepts avec/sans traites separement)
- Noms de piliers auto-detectes a ameliorer (premier sous-dossier → concept central)
- Test de genericite sur un corpus non-PZ

## Twin links

| Ce projet | Lien | Projet distant |
|-----------|------|----------------|
| pipeline/ | genere depuis → | VAULT VISION/ (corpus source) |
| search/engine.go | compatible avec → | chambre_reverberante/chambre.py |
| data/workspace.go | lu par → | pz-workspace/workspace.json |

## Liens

- [../chambre_reverberante/](../chambre_reverberante/) — Version Python originale
- [../corpus_indexer/](../corpus_indexer/) — Boite a outils VAULT
- [../glossaire_pz/](../glossaire_pz/) — Glossaire de reference (88 termes)




README_chambre_GO_corrected.md

Edit

Preview


type: DOC_LIVRABLE projet: chambre plateforme: GitHub — Virgil-LIBRIA/chambre (EXISTANT) status: READY — repo déjà public, enrichir le README existant date: 2026-03-29 strate: S2 repo: https://github.com/Virgil-LIBRIA/chambre stack: Go + spaCy (hybride) commits: 5 (HEAD: Phases 1-2 pipeline ingestion + auto-glossaire spaCy) twin_link: Projets/chambre/CLAUDE.md correction: README précédent (README_chambre_reverberante.md) décrivait Python/Flask — incorrect
chambre — Semantic Resonance Engine
A Go-native semantic memory server for AI-assisted knowledge work.
REST API · Interactive TUI · spaCy NLP pipeline · Local-only.

Go spaCy bubbletea Local

What it is
chambre is a semantic resonance engine: given a query, it finds the most relevant fragments in a local document corpus, enriches them with glossary matches and cross-domain links, and returns a structured response — all running locally via a Go HTTP server.

It ships with an interactive TUI (bubbletea) for direct corpus exploration, and a Python/spaCy pipeline for ingestion and auto-glossary generation.

Architecture
workspace.json          ← project config (decoupled, generic since Phase 0)
│
├── server/             ← Go HTTP REST server
│   └── Compatible with original chambre.py API contract
│
├── vm/                 ← Kernel VM (3-layer memory: ECUME / CREUX / OCEAN)
│   └── 7 opcodes: ANCHOR TRAVERSE TUNNEL GHOST QUERY SEDIMENT VOLUTION
│
├── kernel/             ← Corpus backbone (concept graph, 88 terms, 247 links)
│
├── search/             ← Cosine similarity + fulltext fallback
│
├── nlp/                ← spaCy bridge (fr_core_news, stem + entity detection)
│
├── tui/                ← Interactive terminal UI (bubbletea)
│   └── Live corpus navigation, query mode, VM state display
│
├── pipeline/           ← Ingestion pipeline (Phases 1-2)
│   └── Auto-glossary generation via spaCy
│
└── pz-workspace/       ← Default workspace (Point Zéro corpus)
Commit history (current)
b277222  feat: Phases 1-2 — pipeline ingestion + auto-glossaire spaCy
0ae71fe  feat: Phase 0 — decouplage generique (workspace.json)
0af52a5  feat: serveur HTTP REST (compatible chambre.py)
d6226f9  feat: TUI interactive (bubbletea) + fix kernel loader
ffcd7d6  init: moteur de resonance semantique en Go
Install
bash
git clone https://github.com/Virgil-LIBRIA/chambre
cd chambre
go build ./...

# Python pipeline (optional, for ingestion + auto-glossary)
pip install spacy
python -m spacy download fr_core_news_md
Requirements: Go 1.22+, Python 3.10+ (pipeline only)
No cloud API. No Ollama required for the Go server itself.

Usage
Start the server
bash
./chambre serve --workspace pz-workspace
# → REST API on :5002
Interactive TUI
bash
./chambre tui --workspace pz-workspace
Navigate the corpus live, run queries, inspect VM state (ECUME/CREUX/OCEAN layers), trigger opcodes — all from the terminal.

Ingest a new corpus (Python pipeline)
bash
python pipeline/ingest.py --input /path/to/your/docs --workspace my-workspace
# Builds index, generates auto-glossary via spaCy, writes workspace.json
API
bash
# Health
GET /health

# Semantic search
POST /reverberate
{"query": "your query", "mode": "default", "top_k": 5}

# Structured INTEMPT response
POST /intempt
{"query": "...", "mode": "daisy", "G": 0.3, "n_spin": 0.8}

# Kernel navigation
GET /kernel
GET /kernel/<concept_id>

# VM state
GET /vm/status
GET /vm/hot
API contract is backward-compatible with the original chambre.py (Python/Flask).

Query modes
Mode	G	n_spin	Use when
silence_actif	1.0	0.1	Single precise answer
dayz	1.0	0.2	Strict factual lookup
default	0.5	0.5	Balanced
daisy	0.1	0.8	Creative exploration
vibratoire	0.3	0.9	Maximum serendipity
G = focal gradient (0 = wide, 1 = narrow)
n_spin = complexity tolerance (0 = crystallized, 1 = diffracted)

workspace.json
Decoupled since Phase 0. Point to any corpus by changing the workspace:

json
{
  "name": "my-corpus",
  "corpus_path": "./data/docs",
  "glossary": "./data/glossary.json",
  "embedding_model": "nomic-embed-text",
  "language": "fr"
}
Kernel VM
State-aware memory across queries in a session:

Layer	Role	Cap
ECUME	Recently visited concepts	50
CREUX	Consolidated frequencies	200
OCEAN	Permanent insights	unlimited
7 opcodes: ANCHOR TRAVERSE TUNNEL GHOST QUERY SEDIMENT VOLUTION

Using with Claude Code
markdown
## Semantic Memory (localhost:5002)
curl -s -X POST http://localhost:5002/reverberate \
  -H "Content-Type: application/json" \
  -d '{"query": "QUERY_HERE", "mode": "default", "top_k": 5}'
Roadmap
 Phase 3 — cross-workspace federation
 gRPC interface (alongside REST)
 Embedding cache persistence (SQLite)
 Web UI (optional, alongside TUI)
License
MIT — Virgil-LIBRIA
