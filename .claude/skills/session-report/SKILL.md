---
name: session-report
description: Erzeugt am Sessionende einen knappen Übergabe-Report aus dem Git-Kontext (geänderte Dateien, Commits) plus offenen Punkten und vorgeschlagenen nächsten Schritten. Manuell via /session-report aufrufen.
phase: reflect
disable-model-invocation: true
---

# Session-Report

Erzeuge eine kurze, ehrliche Sessionende-Übergabe. Ziel: Die nächste Session (oder ein Kollege) versteht
in 30 Sekunden, was passiert ist und was als Nächstes ansteht. **Kein Roman, keine Selbstbeweihräucherung** —
Stichpunkte, Signal statt Theater.

## Kontext (wird vor der Antwort eingespielt)

Aktueller Branch:
!`git branch --show-current`

Arbeitsbaum-Status (uncommitted):
!`git status --short`

Commits dieser Session / der letzten Arbeit:
!`git log --oneline -15`

Umfang der uncommitteten Änderungen:
!`git diff --stat`

> Falls die Injektionen leer bleiben (kein Git-Repo o. Ä.): die obigen Befehle selbst über das
> Terminal-Tool ausführen, dann fortfahren.

## Report-Format

Gib **genau** diese Abschnitte aus (leere Abschnitte weglassen):

### Gemacht
- 3–6 Stichpunkte: was real umgesetzt wurde (verifiziert, nicht „sollte funktionieren").

### Geänderte Dateien & Commits
- Neue/geänderte Dateien thematisch gruppiert; relevante Commit-Hashes + Subjects.
- Uncommittete Änderungen explizit als solche kennzeichnen.

### Offen / Nächste Schritte
- Konkret und priorisiert. Was blockiert, was ist Quick Win, was braucht eine Entscheidung des Users.

### Risiken / Notizen *(optional)*
- Nur wenn es etwas gibt: ungetestete Pfade, bewusst ausgelassene Punkte, externe Abhängigkeiten.

## Regeln

- Faktentreu: Wenn etwas nicht getestet wurde, sag es. Keine erfundenen „erledigt"-Häkchen.
- Deutsch, Umlaute korrekt. Englische Identifier/Commits bleiben englisch.
- Wenn ein `LEARNINGS.md` im Repo existiert und diese Session ein wiederkehrendes Muster offenbart hat,
  schlage einen passenden Eintrag vor (aber schreibe ihn nicht ungefragt).
