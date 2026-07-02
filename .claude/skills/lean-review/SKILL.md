---
name: lean-review
description: Prüft den aktuellen Diff auf Over-Engineering und schlägt schlankere Alternativen vor (delete/stdlib/native/yagni/shrink). Read-only, ändert nichts. Manuell via /lean-review aufrufen.
phase: review
disable-model-invocation: true
---

# Lean-Review

Lies den unten eingespielten Diff und finde, **was schlanker ginge** — ohne Korrektheit, Validierung,
Security oder Accessibility zu opfern. Komplement zur `lean-code`-Rule: dort wird schlank *geschrieben*,
hier schlank *nachgeprüft*. **Read-only: schlagen vor, nicht ändern.**

## Kontext (wird vor der Antwort eingespielt)

Uncommittete Änderungen:
!`git diff`

Gestagte Änderungen:
!`git diff --staged`

> Falls beide Injektionen leer bleiben (nichts geändert / kein Git-Repo): den Diff selbst über das
> Terminal-Tool holen (`git diff`, `git diff --staged`) oder den User nach dem Ziel-Bereich fragen.

## Vorgehen

Erst verstehen, dann bewerten: Verfolge, was der Diff tut, bevor du kürzt. Eine „Vereinfachung", die das
Verhalten ändert, ist kein Befund, sondern ein neuer Bug.

Jeden Befund mit genau einem Tag versehen:

- `delete:` — toter Code, ungenutztes Feature, Abstraktion ohne Aufrufer.
- `stdlib:` — manuell reimplementiert, was die Standardbibliothek schon kann.
- `native:` — Dependency/Eigenbau, der ein natives Plattform-Feature dupliziert (`<input type="date">`, CSS, fetch).
- `yagni:` — Abstraktion/Konfigurierbarkeit mit nur einer Implementation; spekulativ.
- `shrink:` — gleiche Logik, deutlich weniger Zeilen.

## Output-Format

Pro Befund **eine** Zeile:

```
L<line> <file>: <tag> <was>. → <Ersatz>.
```

Nach den Befunden eine Abschlusszeile:

- `netto: −<N> Zeilen möglich.` — wenn es etwas zu kürzen gibt.
- `Schon schlank. Ship.` — wenn nicht. (Nichts erfinden, nur um etwas zu sagen.)

## Regeln

- Keine Stiltyrannei: nur Befunde, die Zeilen, eine Dependency oder echte Komplexität sparen.
- `lazy, not negligent`: Validierung an Trust-Boundaries, Error-Handling, Security, A11y nie als „Over-Engineering" flaggen.
- Deutsch, Umlaute korrekt. Code/Identifier bleiben englisch.
