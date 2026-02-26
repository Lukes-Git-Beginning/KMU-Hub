# GSD: Verify Work

Du fuehrst eine **finale Verifikation** der aktuellen Phase durch.

## Voraussetzung

- Alle Plans der Phase muessen SUMMARY-Files haben
- Falls nicht: Erst `/gsd-execute-phase` fuer offene Plans ausfuehren

## Deine Aufgabe

1. **Sammle alle Plan-Summaries** der Phase aus `.planning/phases/{PHASE_NAME}/`

2. **Pruefe Vollstaendigkeit:**
   - Alle geplanten Plans ausgefuehrt?
   - Alle `must_haves` aus jedem Plan erfuellt?
   - Alle `requirements-completed` aus REQUIREMENTS.md abgedeckt?
   - Alle Commits vorhanden? (`git log` gegen Summary-Commits pruefen)

3. **Pruefe Code-Qualitaet:**
   - Structured Logging (kein fmt.Println/console.log)?
   - Thick Services, Thin Handlers?
   - Keine hardcoded Secrets?
   - Test-Coverage wo moeglich?

4. **Pruefe Integration:**
   - Neue Services korrekt registriert?
   - Gateway-Routes vorhanden?
   - Frontend-Routes vorhanden?
   - Docker-Compose aktualisiert (falls neuer Service)?
   - Proto-Codegen ausgefuehrt?

5. **Erstelle VERIFICATION-File** (optional, bei komplexen Phasen):

```markdown
# Phase {N}: {Name} - Verification

**Verified:** {YYYY-MM-DD}

## Plans Verified
- Plan {NN}: {PASSED/FAILED} - {Kommentar}

## Requirements Coverage
| Requirement | Status | Covered by |
|------------|--------|------------|
| {REQ-ID}   | Done   | Plan {NN}  |

## Integration Check
- [ ] Service registered in binary
- [ ] gRPC server handlers wired
- [ ] Gateway HTTP routes added
- [ ] Frontend types/hooks/pages created
- [ ] Docker-Compose updated (if needed)
- [ ] Proto codegen current

## Issues Found
{Issues oder "None"}

## Phase Status: COMPLETE
```

6. **Update STATE.md:**
   - Phase als COMPLETE markieren
   - Naechste Phase als Current Focus setzen
   - Velocity-Metrics finalisieren

7. **Move Phase to Milestones** (bei Milestone-Abschluss):
   - Wenn die Phase einen Milestone abschliesst, Dateien nach `.planning/milestones/{VERSION}/` verschieben

## Wichtig

- Sei gruendlich — diese Verifikation ist der letzte Check vor der naechsten Phase
- Melde Probleme klar und mit Loesungsvorschlag
- Aktualisiere MEMORY.md wenn neue stabile Patterns entdeckt wurden
