---
tags: [tooling, deferred, knowledge-graph]
updated: 2026-04-19
---
# Graphify (vertagt auf Sprint 2/3)

## Was ist Graphify

Open-Source Python-Tool + Claude-Code-Skill ([graphify.net](https://graphify.net/), [GitHub](https://github.com/safishamsi/graphify)), das einen Ordner (Code/Docs/Markdown/PDFs/Bilder/Video) in einen queryablen **Knowledge Graph** komprimiert.

- **Pipeline:** Tree-sitter AST-Parsing (deterministisch, 0 Token) + optionale LLM-Subagents fuer Semantic-Extraction → NetworkX-Graph mit Leiden-Clustering
- **Output:** `graph.json` exponiert als **MCP Server** mit Tools wie `query_graph`, `get_node`, `get_neighbors`, `shortest_path`
- **Compression:** ~71.5x bei ~50 Files laut Self-Benchmark, ~5–15x realistisch bei 22 Files, ~1x bei <10 Files

## Warum jetzt nicht (Stand 2026-04-19)

`.knowledge/` hat aktuell ~22 Markdown-Notes (~5k Zeilen) — zu klein fuer nennenswerte Token-Compression. MCP-Filesystem-Tools (`mcp__knowledge__*`) erfuellen den Zweck heute ausreichend, ohne Setup-Overhead. Die echten Vorteile von Graphify (Relationship-Queries, Cluster-Erkennung, Cross-File-Traversal) zaehlen erst ab ~50+ Knoten.

## Wann re-evaluieren

**Sprint 2/3 (ab 2026-05-12)**, wenn:
- Backend-Code wegen Option-B-Full-Retrofit (~50 Tabellen) deutlich waechst
- `.knowledge/` ueber 40+ Notes hat
- Token-Budget pro Session wieder spuerbar steigt

Dann **Graphify auf `backend/` anwenden** (Go-Code mit vielen Packages, Services, Migrations — dort lohnt sich Compression). Vault selbst bleibt als kuratierter Memory-Layer.

## Empfohlenes Setup (fuer spaeter)

Klassischer Stack: [lucasrosati/claude-code-memory-setup](https://github.com/lucasrosati/claude-code-memory-setup) — Obsidian-Vault als Kontext-Memory + Graphify auf Codebase als MCP-Server. Beides koexistiert.

```bash
pip install graphifyy
graphify install
graphify build /path/to/backend
# graph.json als MCP Server konfigurieren
```

## Quellen

- [Graphify GitHub](https://github.com/safishamsi/graphify)
- [graphify.net](https://graphify.net/)
- [claude-code-memory-setup (Obsidian + Graphify)](https://github.com/lucasrosati/claude-code-memory-setup)
- [Supercharging Claude Code DX (Medium)](https://renjithvr11.medium.com/supercharging-claude-code-dx-solving-context-bloat-and-ai-amnesia-with-graphify-and-ogham-mcp-6f8d5b414081)

## Verwandte Notes
- [[_index]] — Vault-Master-Index
- [[architektur]] — Backend-Struktur (Ziel-Codebase fuer Graphify)
