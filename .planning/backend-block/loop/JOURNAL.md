# Backend-Nachtloop — Journal Lauf 17

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done) · `archive/lauf-12/` (99, davon 41 done) ·
`archive/lauf-13/` (120, davon 21 done — abgestürzt) · `archive/lauf-14/` (110, alle done —
Backlog leergelaufen, gemergt als `cadf4bf4`) · `archive/lauf-15/` (74, alle done — Backlog
leergelaufen, PR #27) · `archive/lauf-16/` (82, alle done — Backlog leergelaufen, PR #28).
Alle inkl. Laufbilanz.

---

## Laufkontext

- **Ausgangspunkt:** Lauf 16 gemergt als `b88d7eac` (PR #28) und deployt (CD 2026-10-25 13:27,
  Kopf 345 clean, 32 Container healthy). `main` = `6001c1a9`. `backend-loop` = `main` plus
  Staging-Commits (Archiv, Entscheidungsrunde, Treiber-Pflege, lokaler E2E-Stack, dieser Lauf).
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **345**. Nummer immer zur
  Laufzeit ermitteln: `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- **Coverage-Start:** **74,5 %** gesamt (Run 40196109073 auf `b88d7eac`). Paketwerte je Unit.
- **Queue:** 107 Units, roter Faden „Rechte gegen den echten Seed — und jedes Modul einmal
  Ende-zu-Ende“. Rechte-Scans (2), Lecks und Grenzen (15), Reste (5), E2E (12), entschiedene
  Features (34), Coverage (17 + 16 Füllung), Reviews (6). Begründung, Regeln und Sperren im Kopf
  von `BACKLOG.yml`.
- **Neu im Gate:** lokaler E2E-Stack `hooks/e2e-local.sh` (Regel 1 im Backlog-Kopf). Der Treiber
  setzt `GOFLAGS=-p=2` und `GOMAXPROCS=2`. Die Kopfzeilen-Zeit ist die Startzeit aus dem
  Laufkontext, wörtlich.

## Entscheidungsrunde 2026-10-25 (Luke, sieben Runden) — Ergebnis

Je Unit steht die Entscheidung im Feld `decision` von `BACKLOG.yml`. Kurzfassung:

- **Inbox:** sichtbar = Eigentümer, zugewiesen oder Mitglied der Team-Inbox. Danach bekommt member
  die Team-Inbox.
- **Abwesenheiten:** member sieht eine neutrale Projektion, der manager sieht die Art für seine
  Berichtslinie.
- **E-Mail:** Self-Setup mit Dial-Filter und Allowlist per Env. Regeln pro Nutzer, Labels geteilt,
  eigene Vorlagen änderbar, im Kontakt-Mailverlauf sieht member nur die eigenen Mails.
- **Rollenmatrix:**
  - member: Fuhrpark-Verfügbarkeitsroute, Schäden lesen, Inventar-Umbuchung, eigene Schichten,
    zugewiesene Helpdesk-Tickets, eigene Aufgaben löschen, Deals (Team), eigene Wiki-Seiten
    teilen/löschen.
  - manager: Finanz-Kachel, Personalakte und Vollfelder der Berichtslinie.
  - NICHT: Tags anlegen, CRM-Merge, Dokument-Freigaben, Kontakte importieren/löschen.
- **Datenschutz:**
  - E-Mail-Absendername beim Anonymisieren scrubben, Body bleibt als Restrisiko (HGB).
  - Meeting-Notizen/-Chat: DSAR-Export und 2 Jahre Frist.
  - Inspektionsfotos werden mit gelöscht, Lieferanten gelten als HGB-Daten.
  - Fristen: dialer_agent_status_log 12 Monate, call_sessions 24 Monate, public_bookings 24 Monate.
- **Finanzen:** Verzugszinsen streichen, Eingangsrechnung prüfen und warnen, `finance_payments.currency`
  als eigene Spalte, Ad-hoc-Rechnung nur in Tenant-Währung.
- **Schichten/Fuhrpark/Work/HR:**
  - Überlappende Schichten werden mit 409 abgelehnt, ein Tausch ist eine Übergabe.
  - Führerschein-Löschen mit Audit-Event, Privatfahrten maskiert (Fahrtenbuch bleibt lückenlos).
  - Aufgabe ohne Projekt bekommt `completed_at`.
  - Deaktivieren läuft über das Offboarding, dazu ein neuer Reaktivieren-RPC.
- **Sonstiges:**
  - Gruppen-DMs bauen. Egress startet erst nach dem Initiator-Stempel (Prod-Egress ist heute
    nicht konfiguriert).
  - Tokens: Share-Tokens als Hash, Share-Links höchstens 365 Tage, Join-Token nur für Teilnehmer,
    Offboarding widerruft CalDAV.
  - Alarm: jetzt nur die Regel auf `audit_events_dropped_total`, der Backup-Wächter folgt tagsüber.
  - Geister-Routen: team-worktime wird gebaut, der Rest wird im FE ausgeblendet.
  - Rahmenvertrag weich: buchen und warnen.

## Was aus Lauf 16 gelernt wurde — drei Dinge, die dich direkt betreffen

1. **Die Rollenmatrix bildet den Ist-Seed ab, nicht die Absicht.** 000068 gab member
   `dialer:agent:manage`, und die Matrix nannte das „ok“. Prüfe Grants gegen die fachliche
   Absicht (Block S).
2. **Handgebaute Tokens sehen keine Seed-Fehler.** `hr:self` (`resource = 'hr'`) fiel durch
   `resource LIKE '%:%'`, der Scope kam nie ins JWT, und ein fehlender Key gilt als `all`. Rechte
   nur mit echtem Token prüfen (Regel 2), neue Keys dreiteilig.
3. **Der Morgen-PR ist nicht automatisch grün.** Fünf E2E-Flows fielen erst in CI. Seit diesem
   Lauf läuft E2E lokal (Regel 1): Ein roter Flow ist ein Fund für diese Nacht, nicht für morgen.

---
