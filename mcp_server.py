"""
Chambre Réverbérante — Serveur MCP local
Bridge entre Claude Code et la Chambre (localhost:5002).

La Chambre doit tourner séparément : python chambre.py
Ce serveur MCP wrappe ses endpoints REST.
"""

import json
import urllib.request
import urllib.error
from fastmcp import FastMCP

mcp = FastMCP("chambre-reverberante")

CHAMBRE_URL = "http://localhost:5002"


def _post(endpoint: str, data: dict) -> dict:
    """POST JSON vers la Chambre."""
    req = urllib.request.Request(
        f"{CHAMBRE_URL}{endpoint}",
        data=json.dumps(data).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.URLError:
        return {"error": "Chambre non accessible sur localhost:5002. Lancer: python chambre.py"}


def _get(endpoint: str) -> dict:
    """GET vers la Chambre."""
    try:
        with urllib.request.urlopen(f"{CHAMBRE_URL}{endpoint}", timeout=10) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.URLError:
        return {"error": "Chambre non accessible sur localhost:5002. Lancer: python chambre.py"}


@mcp.tool()
def reverberate(query: str, mode: str = "default", top_k: int = 5) -> str:
    """Recherche sémantique dans le corpus Point Zéro via la Chambre Réverbérante.
    Retourne les fichiers les plus proches, les termes glossaire (tau_0), et les croisements inter-piliers.

    Args:
        query: Le concept ou la question à explorer
        mode: Mode de recherche — default, dayz (strict), translucide (passerelle), daisy (créatif), vibratoire, silence_actif (chirurgical)
        top_k: Nombre de résultats (défaut: 5)
    """
    data = {"query": query, "mode": mode, "top_k": top_k}
    result = _post("/reverberate", data)
    if "error" in result:
        return result["error"]

    lines = []
    # Résultats corpus
    for r in result.get("resultats", []):
        score = r.get("score", "?")
        fichier = r.get("fichier", "?")
        contexte = r.get("contexte", "")[:150]
        lines.append(f"[{score:.3f}] {fichier}\n  {contexte}")

    # Glossaire (tau_0)
    glossaire = result.get("glossaire", [])
    if glossaire:
        lines.append("\n--- Ancres glossaire (tau_0) ---")
        for g in glossaire:
            lines.append(f"• {g.get('terme', g)}: {g.get('definition', '')[:120]}")

    # Cross-domain
    cross = result.get("cross_domain", [])
    if cross:
        lines.append("\n--- Cross-domain (sérendipité) ---")
        for c in cross:
            lines.append(f"↔ {c}")

    # Alertes
    if result.get("loi_diffraction"):
        lines.append("\n⚠ Loi de Diffraction : complexité sans ancrage, revenir au Point Zéro")
    if result.get("tensegrite", {}).get("ok") is False:
        lines.append("\n⚠ Rupture de Tenségrité : contradiction avec un tau_0")

    return "\n".join(lines) if lines else "Aucun résultat."


@mcp.tool()
def intempt(query: str, mode: str = "default") -> str:
    """Requête INTemple structurée (Protocole v4.5).
    Retourne un bloc Markdown avec tau_0, G, n-spin évalués.

    Args:
        query: Le concept ou la question
        mode: Mode — default, dayz, translucide, daisy, vibratoire, silence_actif
    """
    result = _post("/intempt", {"query": query, "mode": mode})
    if "error" in result:
        return result["error"]
    # L'endpoint /intempt retourne du markdown directement ou un JSON structuré
    if isinstance(result, str):
        return result
    return json.dumps(result, ensure_ascii=False, indent=2)


@mcp.tool()
def kernel_concept(concept: str) -> str:
    """Naviguer le kernel du corpus par concept.
    Retourne les liens, le pilier, l'île et les connexions du concept.

    Args:
        concept: Le terme à explorer dans le kernel
    """
    result = _get(f"/kernel/concept/{concept}")
    if "error" in result:
        return result["error"]
    return json.dumps(result, ensure_ascii=False, indent=2)


@mcp.tool()
def kernel_stats() -> str:
    """Statistiques du kernel : nombre de termes, liens, piliers, îles."""
    result = _get("/kernel")
    if "error" in result:
        return result["error"]
    return json.dumps(result, ensure_ascii=False, indent=2)


@mcp.tool()
def chambre_health() -> str:
    """Vérifier si la Chambre Réverbérante est accessible."""
    result = _get("/health")
    if "error" in result:
        return f"❌ {result['error']}"
    return f"✅ Chambre opérationnelle"


if __name__ == "__main__":
    mcp.run()
