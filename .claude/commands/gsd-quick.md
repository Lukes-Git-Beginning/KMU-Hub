# GSD: Quick Fix

Du fuehrst einen **Ad-hoc Fix** ausserhalb des strukturierten Phase-Workflows durch.

## Wann nutzen

- Bugfixes die nicht auf eine Phase warten koennen
- Kleine Verbesserungen (Typos, fehlende Imports, Logging)
- Dokumentations-Updates
- Dependency-Updates
- Config-Aenderungen

## Deine Aufgabe

1. **Verstehe das Problem** — was muss gefixt werden?
2. **Finde die betroffenen Dateien** im Codebase
3. **Implementiere den Fix** — minimal und fokussiert
4. **Committe** mit passendem Conventional Commit Prefix (fix:, docs:, chore:, refactor:)
5. **Verifiziere** dass der Fix funktioniert

## Regeln

- **Kein Scope Creep** — nur den gemeldeten Fix machen
- **Keine neuen Features** — dafuer gibt es den Phase-Workflow
- **Architektur-Regeln einhalten** (CLAUDE.md)
- **Keine AI-Attribution** in Commits
- **STATE.md nicht updaten** — Quick Fixes werden nicht im Phase-Tracking erfasst

## Beispiele

```
/gsd-quick Fix: WebSocket reconnect loop when token expires
/gsd-quick Docs: Update API endpoint documentation for /contacts
/gsd-quick Chore: Update go.mod dependencies
```
