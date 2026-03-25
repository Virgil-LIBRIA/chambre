"""
chambre-pipeline ingest — extraction de texte et indexation d'un corpus.

Usage:
    python ingest.py <corpus_dir> [--output <workspace_dir>] [--name "Mon Corpus"]

Entree: un dossier contenant des .docx, .pdf, .md, .txt
Sortie: workspace/ avec index.json + search_cache.json
"""

import argparse
import hashlib
import json
import os
import re
import sys
from datetime import datetime
from pathlib import Path

# --- Extraction texte ---

def extract_docx(path: Path) -> str:
    """Extrait le texte d'un fichier .docx."""
    from docx import Document
    try:
        doc = Document(str(path))
        return "\n".join(p.text for p in doc.paragraphs if p.text.strip())
    except Exception as e:
        print(f"  warn: {path.name}: {e}", file=sys.stderr)
        return ""


def extract_pdf(path: Path) -> str:
    """Extrait le texte d'un fichier .pdf."""
    from pypdf import PdfReader
    try:
        reader = PdfReader(str(path))
        texts = []
        for page in reader.pages:
            t = page.extract_text()
            if t:
                texts.append(t)
        return "\n".join(texts)
    except Exception as e:
        print(f"  warn: {path.name}: {e}", file=sys.stderr)
        return ""


def extract_text(path: Path) -> str:
    """Extrait le texte d'un fichier .md ou .txt."""
    try:
        return path.read_text(encoding="utf-8", errors="replace")
    except Exception as e:
        print(f"  warn: {path.name}: {e}", file=sys.stderr)
        return ""


EXTRACTORS = {
    ".docx": extract_docx,
    ".pdf": extract_pdf,
    ".md": extract_text,
    ".txt": extract_text,
}

SUPPORTED_EXTENSIONS = set(EXTRACTORS.keys())


# --- Indexation ---

def count_words(text: str) -> int:
    return len(text.split())


def file_hash(path: Path) -> str:
    h = hashlib.md5()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            h.update(chunk)
    return h.hexdigest()[:12]


def detect_pilier(path: Path, corpus_root: Path) -> str:
    """Detecte le pilier a partir du chemin relatif (premier sous-dossier)."""
    try:
        rel = path.relative_to(corpus_root)
        parts = rel.parts
        if len(parts) > 1:
            # Premier dossier = pilier potentiel
            folder = parts[0]
            # Nettoyer les prefixes numeriques (0_, 1_, 2_, etc.)
            clean = re.sub(r"^\d+[_\s]*", "", folder).strip()
            if clean:
                # Prendre les premiers mots significatifs
                words = clean.split()
                if words:
                    return words[0].upper()[:10]
        return "RACINE"
    except ValueError:
        return "AUTRE"


def scan_corpus(corpus_dir: Path):
    """Parcourt le corpus et retourne les fichiers supportes."""
    files = []
    for path in sorted(corpus_dir.rglob("*")):
        if path.is_file() and path.suffix.lower() in SUPPORTED_EXTENSIONS:
            # Ignorer les fichiers caches et temporaires
            if any(p.startswith(".") or p.startswith("~") for p in path.parts):
                continue
            if path.name.startswith("~$"):
                continue
            files.append(path)
    return files


def ingest_file(path: Path, corpus_root: Path):
    """Ingere un fichier : extraction texte + metadonnees."""
    ext = path.suffix.lower()
    extractor = EXTRACTORS.get(ext)
    if not extractor:
        return None, None

    text = extractor(path)
    if not text.strip():
        return None, None

    rel_path = str(path.relative_to(corpus_root)).replace("\\", "/")
    words = count_words(text)
    pilier = detect_pilier(path, corpus_root)

    index_entry = {
        "nom": path.name,
        "chemin_relatif": rel_path,
        "pilier": pilier,
        "type": ext.lstrip("."),
        "extension": ext,
        "taille_octets": path.stat().st_size,
    }

    cache_entry = {
        "name": path.name,
        "pilier": pilier,
        "text": text[:50000],  # limiter a 50k chars par fichier
        "chars": len(text),
        "words": words,
        "size": path.stat().st_size,
        "type": ext.lstrip("."),
        "pages_estimees": max(1, words // 300),
        "chemin": rel_path,
    }

    return index_entry, cache_entry


# --- Main ---

def main():
    parser = argparse.ArgumentParser(
        description="chambre-pipeline ingest — extraction et indexation d'un corpus"
    )
    parser.add_argument("corpus_dir", help="Dossier contenant les documents")
    parser.add_argument("--output", "-o", default=None,
                        help="Dossier de sortie (defaut: <corpus_dir>_workspace/)")
    parser.add_argument("--name", "-n", default=None,
                        help="Nom du corpus (defaut: nom du dossier)")
    args = parser.parse_args()

    corpus_dir = Path(args.corpus_dir).resolve()
    if not corpus_dir.is_dir():
        print(f"Erreur: {corpus_dir} n'est pas un dossier", file=sys.stderr)
        sys.exit(1)

    output_dir = Path(args.output) if args.output else corpus_dir.parent / f"{corpus_dir.name}_workspace"
    output_dir = output_dir.resolve()
    output_dir.mkdir(parents=True, exist_ok=True)

    corpus_name = args.name or corpus_dir.name

    print(f"\n  chambre-pipeline ingest")
    print(f"  corpus: {corpus_dir}")
    print(f"  output: {output_dir}")
    print(f"  nom:    {corpus_name}\n")

    # Scanner les fichiers
    files = scan_corpus(corpus_dir)
    print(f"  {len(files)} fichiers detectes\n")

    if not files:
        print("  Aucun fichier supporte trouve.", file=sys.stderr)
        sys.exit(1)

    # Ingerer
    index_entries = []
    cache_entries = {}
    errors = 0

    for i, path in enumerate(files):
        ext = path.suffix.lower()
        safe_name = path.name[:60].encode("ascii", errors="replace").decode("ascii")
        print(f"  [{i+1}/{len(files)}] {safe_name}", end="")

        idx, cache = ingest_file(path, corpus_dir)
        if idx and cache:
            index_entries.append(idx)
            cache_entries[path.name] = cache
            print(f"  ({cache['words']} mots)")
        else:
            errors += 1
            print("  SKIP")

    print(f"\n  Resultats: {len(index_entries)} fichiers indexes, {errors} erreurs")

    # Ecrire index.json
    index_path = output_dir / "index.json"
    with open(index_path, "w", encoding="utf-8") as f:
        json.dump({"fichiers": index_entries}, f, ensure_ascii=False, indent=2)
    print(f"  -> {index_path}")

    # Ecrire search_cache.json
    cache_path = output_dir / "search_cache.json"
    with open(cache_path, "w", encoding="utf-8") as f:
        json.dump({"files": cache_entries}, f, ensure_ascii=False, indent=2)
    print(f"  -> {cache_path}")

    # Detecter les piliers
    piliers = {}
    for entry in index_entries:
        p = entry["pilier"]
        piliers[p] = piliers.get(p, 0) + 1

    # Ecrire workspace.json (squelette — sera enrichi par analyze)
    ws = {
        "name": corpus_name,
        "version": "1.0.0",
        "created": datetime.now().strftime("%Y-%m-%d"),
        "files": {
            "glossaire": "glossaire.json",
            "kernel": "kernel.json",
            "index": "index.json",
            "search_cache": "search_cache.json",
            "embeddings": "embeddings.json",
            "memory": "memory.json",
        },
        "config": {
            "modes": {
                "silence_actif": {"g": 1.0, "n_spin": 0.1},
                "dayz": {"g": 1.0, "n_spin": 0.2},
                "default": {"g": 0.5, "n_spin": 0.5},
                "translucide": {"g": 0.5, "n_spin": 0.5},
                "daisy": {"g": 0.1, "n_spin": 0.8},
                "vibratoire": {"g": 0.3, "n_spin": 0.9},
            },
            "piliers": {
                name: {"label": name}
                for name in sorted(piliers.keys())
            },
        },
    }
    ws_path = output_dir / "workspace.json"
    with open(ws_path, "w", encoding="utf-8") as f:
        json.dump(ws, f, ensure_ascii=False, indent=2)
    print(f"  -> {ws_path}")

    # Resume
    print(f"\n  Piliers detectes:")
    for p, n in sorted(piliers.items(), key=lambda x: -x[1]):
        print(f"    {p}: {n} fichiers")

    total_words = sum(c["words"] for c in cache_entries.values())
    total_chars = sum(c["chars"] for c in cache_entries.values())
    print(f"\n  Total: {total_words:,} mots, {total_chars:,} caracteres")
    print(f"  Workspace pret dans: {output_dir}\n")
    print(f"  Prochaine etape: python analyze.py {output_dir}\n")


if __name__ == "__main__":
    main()
