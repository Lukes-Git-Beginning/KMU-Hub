---
description: Manueller Trigger für die Friday-Synthese aus zentria-intel, wenn die Routine ausfällt.
argument-hint: "[--week=YYYY-Wxx | leer fuer aktuelle Woche]"
---

# intel-friday

Du triggerst manuell die `intel-friday`-Routine. Standardmaessig laeuft sie automatisch Fr 05:30 Berlin (Opus). Dieser Skill ist fuer:
- Routine ausgefallen oder Pool war zu eng
- Bericht fuer eine alte Woche nachtraeglich erzeugen
- Test der Pipeline nach Aenderungen

## Argumente

`$ARGUMENTS`:
- leer -> aktuelle ISO-Woche
- `--week=2026-W19` -> spezifische Woche

## Workflow

1. **Woche bestimmen:** ISO-Woche aus Argument oder `date +%G-W%V`.
2. **Daily-Reports verifizieren:** Zaehle `~/Documents/zentria-intel/daily/` Files fuer die Ziel-Woche.
   - Erwartet: 9 Reports (Mo-Do je 2 + Fr 1 evening + ggf. Sa regulation)
   - Falls < 5: Warne den User dass Datenbasis duenn ist, frage ob trotzdem fortfahren.
3. **Routine-Prompt laden:** `~/Documents/zentria-intel/.routines/intel-friday.prompt.md`
4. **Via Skill(schedule, ...) als One-Off triggern:**
   - Modus: `run-once`
   - Modell: `claude-opus-4-7`
   - Token-Cap: 60000 Output
   - Working Dir: `~/Documents/zentria-intel/`
5. **Output erwarten:** `weekly/{year}-W{week}.md`
6. **Discord-Push triggern:** Schreibe `.state/discord_push_pending.json` mit dem Filename — Bot pollt das und postet.
7. **Verifikation:** Lies das erste Embed (Header) und gib es als Bestaetigung an Luke.

## Out-of-band-Output

Wenn der Bot offline ist, schreibe explizit als Output:

```
Friday-Report fertig: ~/Documents/zentria-intel/weekly/2026-W19.md
Bot ist offline -> Push manuell triggern oder Datei direkt oeffnen.
Pick-Mechanik per /intel-pick (Backup-Skill).
```
