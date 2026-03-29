# Chambre — Moteur de Resonance Semantique (Go)

> **Strate : S2 Operationnel** | **Version : 0.2.0** | **Depot : github.com/Virgil-LIBRIA/chambre (prive)**

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
