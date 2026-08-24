---
name: lean-debt
description: Erntet alle lean:-Marker im Repo zu einem Schulden-Ledger und flaggt Marker ohne Upgrade-Trigger. Read-only, manuell via /lean-debt.
disable-model-invocation: true
context: fork
metadata:
  phase: review
---

# Lean-Debt

Sammle alle `lean:`-Marker im Repo und mach die bewusst eingegangenen Vereinfachungen sichtbar. Diese
Marker setzt die `lean-code`-Rule, wenn etwas absichtlich klein gehalten wird — mit Deckel und
Upgrade-Pfad. **Read-only: nur berichten, nichts ändern.**

## Kontext (wird vor der Antwort eingespielt)

Gefundene Marker:
!`git grep -nE "(#|//|/\*|<!--|--)[[:space:]]*lean:" -- . ":(exclude)node_modules" ":(exclude)*lock*"`

> Falls leer (keine Marker / kein Git-Repo): mit den Such-Tools nach `lean:` in Kommentaren suchen
> (Wortgrenze beachten, damit „clean:" nicht matcht). Wenn es wirklich keine Marker gibt, sag das knapp.

## Vorgehen

Jeden Treffer in eine Ledger-Zeile übersetzen. **Eigene Beispiele/Doku ignorieren** — Treffer in
`SKILL.md`, `rules/`, `README`, der Validierungs-Doku u.ä. sind Beschreibung der Konvention, keine echte
Schuld.

Pro echtem Marker:

```
<file>:<line> — <was vereinfacht wurde>. Deckel: <das genannte Limit>. Upgrade: <der Trigger zum Revisit>.
```

Nennt der Kommentar keinen Upgrade-Trigger, die Zeile zusätzlich mit `[kein-trigger]` markieren — das ist
das höchste Risiko (eine Vereinfachung, die niemand je wieder anfasst).

## Output-Format

- Ledger-Zeilen, gruppiert nach Datei.
- Abschluss: `N Marker, davon M ohne Trigger.`
- Wenn `M > 0`: die `[kein-trigger]`-Einträge separat auflisten — sie brauchen zuerst einen Trigger.

## Regeln

- Faktentreu: nur reportieren, was wirklich im Code steht. Keine Deckel/Trigger erfinden.
- Deutsch, Umlaute korrekt. Pfade/Identifier bleiben original.
