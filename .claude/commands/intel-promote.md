---
description: Promotet einen zentria-intel-Cluster als dauerhafte Note in den .knowledge/-Vault.
argument-hint: "<modul-oder-thema> [--dry-run]"
---

# intel-promote

Du verschiebst angesammeltes Markt-Wissen aus dem ephemeren Intel-System in den langlebigen `.knowledge/`-Vault des KMU-Hub-Repos. Promotion ist semi-automatisch: Routine schlaegt vor, dieser Skill fuehrt aus, Luke bestaetigt.

## Promotion-Kriterien (vor Promotion checken)

1. Mind. **5 Keepers** mit `modul:<modul>` oder `thema:<thema>` Tag
2. Spannweite mind. **3 Wochen** (nicht alles aus einer Woche)
3. Mind. **3 verschiedene Quellen** (n_sources verteilt)

Falls nicht erfuellt: `--dry-run` automatisch + Bericht warum nicht.

## Workflow

1. **Keepers sammeln:** Lies alle `~/Documents/zentria-intel/keepers/*.md` mit passendem Tag.
2. **Synthese-Sub-Agent starten:** Spawn Plan-Agent (Opus) mit Auftrag:
   > "Synthetisiere folgende {N} Markt-Insights zu Modul {modul} in eine kohaerente Note nach Cosmi-`.knowledge/`-Stil (YAML-Frontmatter, `[[wikilinks]]`, ~500-1000 Worte). Gruppiere thematisch. Verlinke auf bestehende `.knowledge/`-Notes wo passend (architektur, integrationen, etc.)"
3. **Output-Pfad:** `~/Documents/KMU Hub/.knowledge/intel-<modul>.md`
4. **Frontmatter-Schema:**
   ```yaml
   ---
   tags: [intel, modul-<modul>]
   updated: 2026-05-08
   source: zentria-intel
   keepers_included: [W19-T03-i07, W18-T01-i03, ...]   # Stable-IDs zur Rueckverfolgung
   promotion_window: 2026-W17 .. 2026-W19
   ---
   ```
5. **`.knowledge/_index.md` updaten:** Neuer Tabellen-Eintrag mit Pointer auf neue Note.
6. **Quellen-Keepers markieren** (NICHT loeschen, nur markieren): Frontmatter um `promoted_to: intel-<modul>` ergaenzen.
7. **Git commit + push** im KMU-Hub-Repo:
   - `feat(knowledge): promote intel-<modul> from {N} keepers (W{x}-W{y})`
   - Direct-to-main (User-Memory-Praeferenz)

## Sub-Modi

### `--dry-run`

Zeige nur den Plan was promoted werden wuerde, ohne zu schreiben. Nutzbar fuer Vorschau.

## Wichtig

- `intel-promote` ist **manuell triggert**, nicht automatisch. Auto-Promotion-Vorschlaege landen in `~/Documents/zentria-intel/promotions/<modul>-suggestion.md`, Luke entscheidet.
- Nach erfolgreicher Promotion: User soll `Skill(intel-recall, "<modul>")` testen — der Recall sollte jetzt die promoted-Note plus Keepers zusammenfuehren.
- KEINE Promotion ohne User-Bestaetigung im Voll-Modus (nicht --dry-run).
