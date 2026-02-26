# GSD: Discuss Phase

Du startest die **Discuss-Phase** fuer eine neue Entwicklungsphase im KMU Hub Projekt.

## Deine Aufgabe

1. **Lies den aktuellen Stand:**
   - `.planning/STATE.md` — aktuelle Position, Velocity, Accumulated Decisions
   - `.planning/ROADMAP.md` — Phase-Beschreibung und Success Criteria
   - `.planning/REQUIREMENTS.md` — Requirements die zur Phase gehoeren
   - `.planning/PROJECT.md` — Projekt-Kontext

2. **Recherchiere den Codebase:**
   - Welche bestehenden Module/Services sind relevant?
   - Welche Patterns aus vorherigen Phasen koennen wiederverwendet werden?
   - Welche Abhaengigkeiten zu anderen Services bestehen?
   - Schaue in `.planning/milestones/` und `.planning/phases/` nach Patterns aus aehnlichen Phasen

3. **Diskutiere mit dem User folgende Punkte:**
   - **Phase Boundary:** Was gehoert rein, was nicht?
   - **Implementation Decisions:** Architektur, Patterns, Libraries
   - **Specifics:** Konkrete Ideen und Anforderungen
   - **Deferred Ideas:** Was wird bewusst auf spaeter verschoben?

4. **Erstelle das CONTEXT-File:**
   - Pfad: `.planning/phases/{PHASE_NAME}/{PHASE}-CONTEXT.md`
   - Nutze exakt dieses Format:

```markdown
# Phase {N}: {Name} - Context

**Gathered:** {YYYY-MM-DD}
**Status:** Ready for planning

<domain>
## Phase Boundary

{Was diese Phase abdeckt, in 2-3 Saetzen}

</domain>

<decisions>
## Implementation Decisions

### {Entscheidung 1}
- {Detail}
- {Detail}

### Claude's Discretion
- {Dinge die Claude bei der Implementation entscheiden darf}

</decisions>

<specifics>
## Specific Ideas

- {Konkrete Ideen}

</specifics>

<deferred>
## Deferred Ideas

{Was bewusst verschoben wird, oder "None"}

</deferred>

---

*Phase: {phase-slug}*
*Context gathered: {YYYY-MM-DD}*
```

## Wichtig

- Stelle genuegend Fragen um den Scope klar zu definieren
- Referenziere bestehenden Code mit Dateipfaden
- Identifiziere wiederverwendbare Patterns aus vorherigen Phasen
- Halte die Discussion fokussiert auf die Phase — keine Scope-Creep-Diskussionen
- Config: `.planning/config.json` bestimmt Mode (yolo/careful) und Depth
