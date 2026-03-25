"""
chambre-pipeline analyze — extraction automatique de concepts (glossaire + kernel).

Usage:
    python analyze.py <workspace_dir> [--max-concepts 100] [--min-freq 3]

Entree: workspace/ avec search_cache.json (produit par ingest.py)
Sortie: glossaire.json, kernel.json, embeddings.json dans le workspace

Utilise spaCy fr_core_news_md pour :
- noun_chunks → candidats concepts
- TF-IDF → importance
- vecteurs 300d → similarite / clustering
"""

import argparse
import json
import math
import re
import sys
from collections import Counter, defaultdict
from datetime import datetime
from pathlib import Path

import spacy
from spacy.tokens import Span
import numpy as np


# --- Extraction de concepts ---

def load_search_cache(workspace: Path) -> dict:
    """Charge le search_cache.json."""
    path = workspace / "search_cache.json"
    with open(path, "r", encoding="utf-8") as f:
        data = json.load(f)
    return data.get("files", data)


def clean_chunk(text: str) -> str:
    """Nettoie un noun chunk."""
    # Retirer determinants et prepositions en debut
    text = re.sub(r"^(l[ea]s?\s+|un[es]?\s+|d[eu]s?\s+|des\s+|du\s+|au[x]?\s+|ce[st]?\s+|sa\s+|son\s+|ses\s+|leur[s]?\s+|notre\s+|nos\s+|votre\s+|vos\s+|ma\s+|mon\s+|mes\s+|ta\s+|ton\s+|tes\s+)", "", text, flags=re.IGNORECASE)
    # Retirer ponctuation finale
    text = text.strip(" \t\n\r.,;:!?-–—")
    return text


def is_valid_concept(text: str) -> bool:
    """Filtre les candidats non pertinents."""
    if len(text) < 3:
        return False
    if len(text.split()) > 5:
        return False
    # Rejeter les nombres seuls, les mots tres courts
    if re.match(r"^\d+$", text):
        return False
    # Rejeter les pronoms et mots-outils
    stopwords = {
        "chose", "choses", "fait", "faits", "cas", "facon", "maniere",
        "partie", "parties", "niveau", "sens", "point", "terme", "termes",
        "type", "types", "sorte", "sortes", "forme", "formes", "mode",
        "idee", "idees", "question", "questions", "exemple", "exemples",
        "rapport", "rapport", "lieu", "effet", "cadre", "base",
        "page", "pages", "chapitre", "chapitres", "section", "sections",
        "fichier", "fichiers", "document", "documents", "texte", "textes",
        "version", "versions", "note", "notes", "remarque", "remarques",
        # Pronoms et mots grammaticaux
        "il", "ils", "elle", "elles", "on", "nous", "vous", "je", "tu",
        "qui", "que", "quoi", "dont", "ou", "lui", "leur", "eux",
        "cela", "ceci", "ce", "ca", "celui", "celle", "ceux", "celles",
        "tout", "tous", "toute", "toutes", "rien", "autre", "autres",
        "meme", "quelque chose", "quelques", "chaque", "aucun", "aucune",
        "peu", "beaucoup", "plus", "moins", "tres", "bien", "mal",
        "fois", "moment", "temps", "jour", "jours", "annee", "mois",
        "monde", "homme", "femme", "gens", "vie", "mort", "main", "yeux",
        "tete", "corps", "place", "fin", "debut", "suite", "reste",
        "mot", "mots", "points", "chose",
    }
    if text.lower() in stopwords:
        return False
    # Rejeter les mots d'un seul caractere apres nettoyage
    words = text.split()
    if all(len(w) <= 2 for w in words):
        return False
    return True


def extract_concepts_from_doc(nlp, text: str, max_chars: int = 30000):
    """Extrait les noun chunks d'un texte avec spaCy."""
    # Limiter la taille pour la performance
    if len(text) > max_chars:
        text = text[:max_chars]

    doc = nlp(text)
    chunks = []
    for chunk in doc.noun_chunks:
        # Filtrer : le head du chunk doit etre un nom (pas un pronom)
        if chunk.root.pos_ not in ("NOUN", "PROPN"):
            continue
        clean = clean_chunk(chunk.text.strip())
        if is_valid_concept(clean):
            chunks.append(clean.lower())
    return chunks


# --- TF-IDF ---

def compute_tfidf(doc_concepts: dict[str, list[str]], min_freq: int = 3):
    """
    Calcule le TF-IDF des concepts.
    doc_concepts: {nom_fichier: [liste de concepts]}
    Retourne: {concept: score_tfidf}
    """
    # Document frequency
    df = Counter()
    # Total frequency
    tf_global = Counter()
    n_docs = len(doc_concepts)

    for doc_name, concepts in doc_concepts.items():
        tf_global.update(concepts)
        unique = set(concepts)
        df.update(unique)

    # Filtrer par frequence minimale
    tfidf = {}
    for concept, freq in tf_global.items():
        if freq < min_freq:
            continue
        idf = math.log(n_docs / (df[concept] + 1))
        # TF normalise par le max
        tf = freq / tf_global.most_common(1)[0][1]
        tfidf[concept] = tf * idf

    return tfidf


# --- Definition courte ---

def find_definition(text: str, term: str, max_len: int = 200) -> str:
    """Trouve la premiere phrase contenant le terme."""
    text_lower = text.lower()
    term_lower = term.lower()

    # Chercher la position du terme
    idx = text_lower.find(term_lower)
    if idx < 0:
        return ""

    # Trouver le debut de la phrase (remonter jusqu'au point precedent)
    start = max(0, text_lower.rfind(".", 0, idx) + 1)
    # Trouver la fin de la phrase
    end_candidates = [
        text_lower.find(".", idx),
        text_lower.find("!", idx),
        text_lower.find("?", idx),
    ]
    end = min((e for e in end_candidates if e > 0), default=len(text))
    end = min(end + 1, len(text))

    sentence = text[start:end].strip()
    if len(sentence) > max_len:
        sentence = sentence[:max_len] + "..."
    return sentence


# --- Relations (cooccurrence) ---

def build_cooccurrence(doc_concepts: dict[str, list[str]], top_concepts: set[str]):
    """
    Construit les relations par cooccurrence dans les memes documents.
    Retourne: {(concept_a, concept_b): count}
    """
    cooc = Counter()
    for doc_name, concepts in doc_concepts.items():
        # Concepts de ce document qui sont dans le top
        present = list(set(c for c in concepts if c in top_concepts))
        for i in range(len(present)):
            for j in range(i + 1, len(present)):
                pair = tuple(sorted([present[i], present[j]]))
                cooc[pair] += 1
    return cooc


# --- Vecteurs et clustering ---

def compute_concept_vectors(nlp, concepts: list[str]) -> dict[str, list[float]]:
    """Calcule le vecteur spaCy moyen pour chaque concept."""
    vectors = {}
    for concept in concepts:
        doc = nlp(concept)
        if doc.has_vector and doc.vector_norm > 0:
            vectors[concept] = doc.vector.tolist()
    return vectors


def cluster_concepts(vectors: dict[str, list[float]], n_clusters: int = 6):
    """
    Clustering k-means simple des concepts par leurs vecteurs.
    Retourne: {concept: cluster_id}, {cluster_id: centroid_concept}
    """
    if len(vectors) < n_clusters:
        # Pas assez de concepts pour clusterer
        assignments = {c: 0 for c in vectors}
        return assignments, {0: list(vectors.keys())[0] if vectors else ""}

    concepts = list(vectors.keys())
    matrix = np.array([vectors[c] for c in concepts])

    # K-means simple (sans sklearn)
    # Initialiser avec k concepts aleatoires
    rng = np.random.default_rng(42)
    idx = rng.choice(len(concepts), size=n_clusters, replace=False)
    centroids = matrix[idx].copy()

    for _ in range(50):
        # Assigner chaque concept au centroid le plus proche
        dists = np.linalg.norm(matrix[:, None] - centroids[None, :], axis=2)
        labels = np.argmin(dists, axis=1)

        # Recalculer les centroids
        new_centroids = np.zeros_like(centroids)
        for k in range(n_clusters):
            members = matrix[labels == k]
            if len(members) > 0:
                new_centroids[k] = members.mean(axis=0)
            else:
                new_centroids[k] = centroids[k]

        if np.allclose(centroids, new_centroids, atol=1e-6):
            break
        centroids = new_centroids

    # Trouver le concept le plus central de chaque cluster
    assignments = {}
    cluster_names = {}
    for k in range(n_clusters):
        member_indices = np.where(labels == k)[0]
        if len(member_indices) == 0:
            continue
        members = matrix[member_indices]
        centroid = centroids[k]
        dists_to_center = np.linalg.norm(members - centroid, axis=1)
        central_idx = member_indices[np.argmin(dists_to_center)]
        cluster_names[k] = concepts[central_idx]

        for mi in member_indices:
            assignments[concepts[mi]] = k

    return assignments, cluster_names


# --- Export ---

def concept_to_id(text: str) -> str:
    """Genere un ID normalise pour un concept."""
    # Retirer accents simples
    replacements = {
        "e": "eeeee", "a": "aaaa", "i": "iii", "o": "ooo", "u": "uuu",
    }
    result = text.lower().strip()
    result = re.sub(r"[''`]", "", result)
    result = re.sub(r"[^a-z0-9\s-]", "", result)
    result = re.sub(r"\s+", "-", result)
    result = result.strip("-")
    return result[:50]


# --- Main ---

def main():
    parser = argparse.ArgumentParser(
        description="chambre-pipeline analyze — auto-glossaire et auto-kernel"
    )
    parser.add_argument("workspace_dir", help="Dossier workspace (produit par ingest)")
    parser.add_argument("--max-concepts", type=int, default=100,
                        help="Nombre max de concepts a extraire (defaut: 100)")
    parser.add_argument("--min-freq", type=int, default=3,
                        help="Frequence minimale d'un concept (defaut: 3)")
    parser.add_argument("--n-clusters", type=int, default=6,
                        help="Nombre de piliers/clusters (defaut: 6)")
    args = parser.parse_args()

    workspace = Path(args.workspace_dir).resolve()
    if not workspace.is_dir():
        print(f"Erreur: {workspace} n'est pas un dossier", file=sys.stderr)
        sys.exit(1)

    print(f"\n  chambre-pipeline analyze")
    print(f"  workspace: {workspace}")

    # Charger les textes
    cache = load_search_cache(workspace)
    print(f"  {len(cache)} fichiers dans le cache\n")

    # Charger spaCy
    print("  Chargement spaCy fr_core_news_md...", end=" ", flush=True)
    nlp = spacy.load("fr_core_news_md", disable=["ner"])
    print("OK\n")

    # Phase 1 : Extraction des concepts par document
    print("  Extraction des concepts...")
    doc_concepts = {}
    all_texts = {}

    for i, (name, entry) in enumerate(cache.items()):
        text = entry.get("text", "")
        if not text:
            continue

        safe = name[:50].encode("ascii", errors="replace").decode("ascii")
        print(f"    [{i+1}/{len(cache)}] {safe}", end="")

        concepts = extract_concepts_from_doc(nlp, text)
        doc_concepts[name] = concepts
        all_texts[name] = text
        print(f"  ({len(concepts)} chunks)")

    # Phase 2 : TF-IDF scoring
    print(f"\n  Calcul TF-IDF (min_freq={args.min_freq})...")
    tfidf = compute_tfidf(doc_concepts, min_freq=args.min_freq)
    print(f"  {len(tfidf)} concepts au-dessus du seuil")

    # Top N concepts
    sorted_concepts = sorted(tfidf.items(), key=lambda x: -x[1])[:args.max_concepts]
    top_concepts = set(c for c, _ in sorted_concepts)
    print(f"  Top {len(sorted_concepts)} concepts retenus\n")

    # Phase 3 : Vecteurs et clustering
    print("  Calcul des vecteurs spaCy...")
    concept_list = [c for c, _ in sorted_concepts]
    vectors = compute_concept_vectors(nlp, concept_list)
    print(f"  {len(vectors)} concepts avec vecteurs\n")

    print(f"  Clustering en {args.n_clusters} piliers...")
    assignments, cluster_names = cluster_concepts(vectors, n_clusters=args.n_clusters)

    # Afficher les clusters
    clusters = defaultdict(list)
    for concept, cluster_id in assignments.items():
        clusters[cluster_id].append(concept)

    pilier_names = {}
    for cluster_id, members in sorted(clusters.items()):
        central = cluster_names.get(cluster_id, members[0])
        pilier_name = central.upper().replace(" ", "_")[:15]
        pilier_names[cluster_id] = pilier_name
        print(f"    Pilier {pilier_name}: {len(members)} concepts")
        for m in members[:5]:
            score = tfidf.get(m, 0)
            print(f"      - {m} ({score:.4f})")
        if len(members) > 5:
            print(f"      ... +{len(members) - 5}")
    print()

    # Phase 4 : Relations (cooccurrence)
    print("  Construction des relations...")
    cooc = build_cooccurrence(doc_concepts, top_concepts)
    # Garder les paires significatives (>= 2 docs communs)
    significant_pairs = {pair: count for pair, count in cooc.items() if count >= 2}
    print(f"  {len(significant_pairs)} relations significatives\n")

    # Phase 5 : Definitions
    print("  Extraction des definitions...")
    definitions = {}
    for concept in concept_list:
        for name, text in all_texts.items():
            defn = find_definition(text, concept)
            if defn and len(defn) > 30:
                definitions[concept] = defn
                break

    # Phase 6 : Export glossaire.json
    print("  Export glossaire.json...")
    termes = []
    for concept, score in sorted_concepts:
        cid = concept_to_id(concept)
        if not cid:
            continue

        cluster_id = assignments.get(concept, 0)
        pilier = pilier_names.get(cluster_id, "AUTRE")

        # Relations de ce concept
        relations = []
        for (a, b), count in significant_pairs.items():
            if a == concept:
                relations.append(concept_to_id(b))
            elif b == concept:
                relations.append(concept_to_id(a))

        termes.append({
            "id": cid,
            "terme": concept,
            "pilier": pilier,
            "definition_courte": definitions.get(concept, ""),
            "synonymes": [],
            "relations": relations[:10],  # max 10 relations
        })

    glossaire = {
        "version": "auto-" + datetime.now().strftime("%Y%m%d"),
        "termes": termes,
    }
    glossaire_path = workspace / "glossaire.json"
    with open(glossaire_path, "w", encoding="utf-8") as f:
        json.dump(glossaire, f, ensure_ascii=False, indent=2)
    print(f"  -> {glossaire_path} ({len(termes)} termes)")

    # Phase 7 : Export kernel.json
    print("  Export kernel.json...")
    liens = []
    for (a, b), count in significant_pairs.items():
        a_id = concept_to_id(a)
        b_id = concept_to_id(b)
        a_cluster = assignments.get(a, -1)
        b_cluster = assignments.get(b, -1)
        lien_type = "cross-domain" if a_cluster != b_cluster else "associatif"
        liens.append({
            "de": a_id,
            "vers": b_id,
            "type": lien_type,
        })

    iles = {}
    for cluster_id, members in clusters.items():
        name = pilier_names.get(cluster_id, f"cluster_{cluster_id}")
        iles[name] = {"concepts": [concept_to_id(m) for m in members]}

    kernel = {
        "noeuds": {concept_to_id(c): {"terme": c, "pilier": pilier_names.get(assignments.get(c, 0), "AUTRE")}
                    for c, _ in sorted_concepts if concept_to_id(c)},
        "liens": liens,
        "iles": iles,
        "stats": {
            "termes": len(termes),
            "liens": len(liens),
            "cross_domain": sum(1 for l in liens if l["type"] == "cross-domain"),
            "bidirectionnel": 0,
        },
    }
    kernel_path = workspace / "kernel.json"
    with open(kernel_path, "w", encoding="utf-8") as f:
        json.dump(kernel, f, ensure_ascii=False, indent=2)
    print(f"  -> {kernel_path} ({len(liens)} liens)")

    # Phase 8 : Export embeddings.json (vecteurs spaCy par fichier)
    print("  Export embeddings.json...")
    file_embeddings = {}
    for name, text in all_texts.items():
        # Vecteur moyen du document
        doc = nlp(text[:10000])
        if doc.has_vector and doc.vector_norm > 0:
            file_embeddings[name] = doc.vector.tolist()

    emb_path = workspace / "embeddings.json"
    with open(emb_path, "w", encoding="utf-8") as f:
        json.dump(file_embeddings, f)
    print(f"  -> {emb_path} ({len(file_embeddings)} vecteurs)")

    # Mettre a jour workspace.json avec les piliers detectes
    ws_path = workspace / "workspace.json"
    with open(ws_path, "r", encoding="utf-8") as f:
        ws = json.load(f)

    ws["config"]["piliers"] = {
        name: {"label": name}
        for name in pilier_names.values()
    }
    with open(ws_path, "w", encoding="utf-8") as f:
        json.dump(ws, f, ensure_ascii=False, indent=2)
    print(f"  -> {ws_path} (piliers mis a jour)")

    # Resume
    print(f"\n  === Resume ===")
    print(f"  Concepts: {len(termes)}")
    print(f"  Piliers:  {len(pilier_names)}")
    print(f"  Liens:    {len(liens)} ({sum(1 for l in liens if l['type'] == 'cross-domain')} cross-domain)")
    print(f"  Vecteurs: {len(file_embeddings)}")
    print(f"\n  Workspace complet dans: {workspace}")
    print(f"  Tester: chambre --workspace {workspace} search \"concept\"\n")


if __name__ == "__main__":
    main()
