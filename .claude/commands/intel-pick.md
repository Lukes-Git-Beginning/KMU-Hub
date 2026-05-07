---
description: Markiert einen Friday-Report-Insight aus zentria-intel als keep/followup/inspire/dismiss. Backup wenn Discord-Bot offline ist.
argument-hint: "<id> [keep|followup|inspire|dismiss] [tags] [note]"
---

# intel-pick

Du markierst einen Insight aus dem Wochen-Report `zentria-intel/weekly/<...>.md` als gepickt. Discord-Bot ist primaerer Channel — dieser Skill ist nur Backup wenn Bot offline ist oder Luke ohne Discord arbeitet.

## Argumente parsen

`$ARGUMENTS` hat folgende Form (Slash-Command-Style):

```
W19-T03-i07 keep modul:helpdesk,thema:auto-triage "Auto-Triage AI Pflicht 2027"
```

- `<id>`: Stable-ID-Format `W{week}-T{theme}-i{item}`
- `<action>`: `keep` | `followup` | `inspire` | `dismiss`
- `<tags>`: komma-separierte `key:value`-Tags (optional)
- `<note>`: Quoted Notiz (optional)

Bei `followup` zusaetzliche Frage: Wann faellig? (Format `YYYY-MM-DD` oder relative `+30d`)

## Workflow

1. **Source finden:** Suche Insight in `~/Documents/zentria-intel/weekly/` — neuester Friday-Report zuerst.
2. **Slug erzeugen:** Aus Title kebab-case + Modul + Wochen-Suffix, z.B. `helpdesk-auto-triage-2026-w19.md`.
3. **Frontmatter generieren:**
   ```yaml
   ---
   id: W19-T03-i07
   created: 2026-05-08
   weekday: friday
   modules: [helpdesk]                 # aus tags
   themes: [auto-triage]                # aus tags
   sources: [...]                       # aus dem Original-Insight
   n_sources: 5
   trend_score: 0.87                    # falls im Original
   decision: keep                       # aus action
   followup_due: 2026-06-01             # nur bei followup
   note: |
     {Lukes Notiz hier}
   ---
   ```
4. **Body:** Original-Snippet aus Friday-Report einbetten plus Lukes Notiz.
5. **Schreiben in:**
   - `keep` -> `keepers/<slug>.md`
   - `followup` -> `followups/<slug>.md`
   - `inspire` -> `inspiration/<slug>.md`
   - `dismiss` -> nur Eintrag in `.state/dismissed.jsonl` (kein File)
6. **Index aktualisieren:**
   - `keep` -> Eintrag in `KEEPERS.md` unter passendem Modul-Header
   - `followup` -> Eintrag in `FOLLOWUPS.md` unter passendem Faelligkeits-Bucket
   - `inspire` -> Eintrag in `INSPIRATION.md`
7. **MEMORY.md updaten:** Bei `keep` mit "wichtig"-Tag oder strategischer Notiz: Pointer-Zeile in user-`MEMORY.md` Block `## Intel-System` hinzufuegen.
8. **Git commit + push** im zentria-intel-Repo:
   - Conventional: `feat(keepers): pick W19-T03-i07 as keep`
   - Keine AI-Attribution
   - Push nach origin

## Verifikation

Vor dem Beenden: `cat ~/Documents/zentria-intel/keepers/<slug>.md` und zeige Luke das Ergebnis.

## Falls ID nicht gefunden

Liste die letzten 3 Friday-Reports und ihre IDs, frage Luke nach Korrektur.
