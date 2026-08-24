---
description: Aktualisiert den .knowledge/-Vault mit allen Änderungen seit dem letzten Update. Nach Commits, Feature-Abschlüssen oder Architektur-Änderungen.
argument-hint: "[bereich oder leer fuer auto-detect]"
---

# Knowledge Base Update

Du aktualisierst die Obsidian Knowledge Base in `.knowledge/`. Arbeite gruendlich und vollstaendig.

## Schritt 1: Aenderungen erkennen

Fuehre diese Befehle parallel aus um zu verstehen was sich geaendert hat:

```bash
# Commits seit letztem .knowledge/ Update
git log --oneline --since="$(git log -1 --format=%ci -- .knowledge/)" -- . ':!.knowledge/'

# Was hat sich geaendert? (Dateien gruppiert)
git diff --stat $(git log -1 --format=%H -- .knowledge/)..HEAD -- . ':!.knowledge/'

# Aktuelle .knowledge/ Frontmatter-Daten
grep -r "^updated:" .knowledge/*.md
```

Falls $ARGUMENTS angegeben wurde, fokussiere dich NUR auf diesen Bereich (z.B. "i18n", "deployment", "api").

## Schritt 2: Relevanz-Mapping

Ordne die Aenderungen den .knowledge/ Notes zu:

| Aenderung betrifft | Note aktualisieren |
|--------------------|--------------------|
| `backend/internal/gateway/route_*.go` | `architektur.md`, `api.md` |
| `backend/migrations/` | `datenbank.md` |
| `backend/internal/*/` (neue Packages) | `architektur.md`, `integrationen.md` |
| `desktop/src/renderer/src/i18n/` | `i18n.md` |
| `desktop/src/renderer/src/modules/` | `architektur.md` |
| `desktop/src/renderer/src/stores/` | `architektur.md` |
| `desktop/src/renderer/src/components/` | `design.md` |
| `desktop/package.json` (neue Deps) | `stack.md` |
| `deploy/`, `.github/workflows/` | `deployment.md` |
| `backend/internal/middleware/`, Security | `security.md` |
| `backend/internal/biz/bexio|lexware|datev/` | `integrationen.md` |
| `backend/internal/plugin/` | `integrationen.md` |
| Tests, E2E, Smoke | `testing.md` |
| Preisaenderungen | `pricing.md` |
| Bug-Fixes, Workarounds | `troubleshooting.md` |
| Grosse Features abgeschlossen | `milestones.md` |

Wenn KEINE relevanten Aenderungen fuer eine Note gefunden werden, ueberspringe sie.

## Schritt 3: Notes aktualisieren

Fuer jede betroffene Note:

1. **Lesen** — Aktuelle Note mit Read-Tool lesen
2. **Vergleichen** — Was ist neu/geaendert im Code vs. was steht in der Note?
3. **Aktualisieren** — Nur die Abschnitte aendern die betroffen sind. NICHT die ganze Datei neu schreiben wenn nur ein Abschnitt betroffen ist.
4. **Frontmatter** — `updated:` auf heutiges Datum setzen
5. **Cross-References** — Wenn eine neue Note referenziert werden muss, `[[note-name]]` Links hinzufuegen

### Regeln
- **Sprache:** Deutsch fuer Inhalte, Englisch fuer Code/Tech-Terms
- **Branding:** "Cosmi" (nicht "KMU Hub") fuer alle neuen Eintraege
- **Obsidian-Format:** YAML Frontmatter mit `tags` und `updated`, Links via `[[note-name]]`
- **Praegnant:** Knowledge Base ist Referenz, kein Tutorial. Kurz und faktisch.
- **Keine Duplikate:** Wenn Info bereits in einer Note steht, nicht nochmal schreiben
- **Neue Notes** nur erstellen wenn ein komplett neues Themengebiet entsteht (z.B. ein neues Integrationsframework). Dann auch `_index.md` aktualisieren.

## Schritt 4: Index pruefen

Wenn neue Notes erstellt wurden:
1. `_index.md` — Note in passende Kategorie einfuegen
2. Andere Notes — `[[neue-note]]` Links wo relevant

## Schritt 5: Zusammenfassung

Gib eine kurze Zusammenfassung aus:
- Welche Notes aktualisiert (mit Einzeiler was geaendert wurde)
- Welche Notes uebersprungen (keine relevanten Aenderungen)
- Ob neue Notes erstellt wurden
