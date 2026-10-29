# Backend-Nachtloop — Journal Lauf 18

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done) · `archive/lauf-12/` (99, davon 41 done) ·
`archive/lauf-13/` (120, davon 21 done — abgestürzt) · `archive/lauf-14/` (110, alle done —
Backlog leergelaufen, gemergt als `cadf4bf4`) · `archive/lauf-15/` (74, alle done — Backlog
leergelaufen, PR #27) · `archive/lauf-16/` (82, alle done — Backlog leergelaufen, PR #28) ·
`archive/lauf-17/` (132 Einträge, 116 done in 117 Iterationen — Deadline 10:00, PR #29).
Alle inkl. Laufbilanz.

---

## Laufkontext

- **Ausgangspunkt:** Lauf 17 gemergt als `90b03133` (PR #29) und deployt (CD 2026-10-26 09:49,
  `/health` meldet `90b03133`). `backend-loop` = `main` plus Staging-Commits (Archiv Lauf 17,
  dieser Lauf) plus die FE-Commits vom 2026-10-29.
- **Migrationen:** Repo-Kopf **364**. Nummer immer zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- **Coverage-Start:** **76,8 %** gesamt (Run 40300605217 auf `90b03133`). Paketwerte je Unit.
- **Queue:** 44 Units, roter Faden „Fehlerklassen-Zwillinge“. Pilot-Pfad und Entschiedenes (10),
  Einzelfälle aus Lauf 17 (14), Zwillings-Scans (11 plus 2 Reviews), Coverage-Auffüller (7).
  Begründung, Regeln und Sperren im Kopf von `BACKLOG.yml`.
- **Neu:** Scans legen ihre Fix-Units direkt hinter sich (Regel 1). `IST:`-Subtests aus Lauf 17
  werden zu Erwartungen, sobald ihr Fix gebaut ist (Regel 3).

## Entscheidungsrunde 2026-10-29 (Luke, vier Runden) — Ergebnis

Je Unit steht die Entscheidung im Feld `decision` von `BACKLOG.yml`. Kurzfassung:

- **Einladung:** Mail über den vorhandenen System-Mailer, Annahmeseite `/invite` im Gateway nach dem
  Muster `/reset-password`. Kein Deep-Link in die App, kein neues Env.
- **`notification.send`:** Zustellung über den Event-Bus an den vorhandenen `InboxConsumer`, keine
  neue RPC. Automation darf auf das eigene Event nicht wieder triggern.
- **Aufnahme:** Teilnehmer willigen in die `pending`-Aufnahme ein, Egress erst nach allen Ja und dem
  Initiator. Der Status geht als String über den Wire.
- **Abo-Rechnung:** „Jetzt erzeugen“ vor Fälligkeit ergibt 409. Vorziehen gibt es nicht.
- **Rechte:** manager liest Fuhrpark-Schäden mit Scope all. Geteilte E-Mail-Vorlage anlegen nur mit
  dem groben Schreib-Key, member bekommt 403.
- **GPS bei Privatfahrten:** so lassen, `lean:`-Marker gesetzt (Staging-Commit), Trigger „erster
  Kunde mit Tracker“.
- **Retention:** Fristen bleiben bis zur anwaltlichen Beratung liegen. Der Scan prüft nur, ob
  vorhandene Löschwege die Kindtabellen mitnehmen.
- **Produktion-Hülle:** Der Desktop entpackt (FE, tagsüber erledigt).
- **Tagsüber, nicht im Lauf:** vier FE-Units (Berichte-Link einmalig zeigen, Einladungsdialog,
  Gutschrift-Storno über `/cancel`, Produktion entpacken), DSGVO-Bereinigung der Prod-Testkonten,
  Darien-Briefing zu Schichten.

## Was aus Lauf 17 gelernt wurde — drei Dinge, die dich direkt betreffen

1. **Die teuersten Funde kamen nebenbei.** `work.create_task` ohne `created_by` (Totalausfall) und
   der `notification.send`-Stub fielen erst in der vorletzten Coverage-Unit auf. Deshalb scannt
   dieser Lauf die Fehlerklassen gezielt, statt auf Zufallsfunde zu warten.
2. **RLS verdeckt fehlende Tenant-Prädikate.** In sechs Coverage-Units überlebte die Mutationsprobe
   auf das Tenant-Prädikat zuerst, weil RLS die Zeile schon verbarg. Erst ein Subtest unter
   System-Kontext machte sie rot (Regel h).
3. **CRLF-Altdateien verfälschen Proben.** Mehrzeilige Probenmuster trafen in CRLF-Dateien nicht
   und galten zuerst als „überlebt“. Zeilenende vor der Probe prüfen.

---
