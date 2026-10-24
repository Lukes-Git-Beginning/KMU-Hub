# Backend-Nachtloop — Journal Lauf 16

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done) · `archive/lauf-12/` (99, davon 41 done) ·
`archive/lauf-13/` (120, davon 21 done — abgestürzt) · `archive/lauf-14/` (110, alle done —
Backlog leergelaufen, gemergt als `cadf4bf4`) · `archive/lauf-15/` (74, alle done — Backlog
leergelaufen, PR #27). Alle inkl. Laufbilanz.

---

## Laufkontext

- **Ausgangspunkt:** Lauf 15 gemergt als `da9fc602` (PR #27) und deployt (CD 2026-10-24
  10:44 UTC, Smoke 27/27). `backend-loop` = `main` plus Staging-Commits (Entscheidungsrunde in
  NEXT, E2E-Stack-Erweiterung, dieser Lauf). CI grün auf `da9fc602` (Run 40085600528, alle
  fünf Jobs inkl. E2E, plus CI Desktop).
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **332**, `dirty = f`. Lauf 16
  bringt mehrere Seed-Migrationen. Nummer immer zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- **Coverage-Start:** **73,7 %** gesamt bei Gate 15 % (Run 40085600528). Bezugswerte je Paket im
  Kopf von `BACKLOG.yml`.
- **Queue:** 52 Units, roter Faden „Echte Nutzer, Teil 2: der Mitarbeiter, nicht der Admin —
  und jede Eingabe kommt an“. Sicherheit (1), Totalausfälle für member/manager (16),
  entschiedene Einzelfixes (11), Geister-Routen und Scans (14), E2E als member (8),
  Coverage (2). Begründung, sechs neue Regeln und Sperren im Kopf von `BACKLOG.yml`.
- **CI-E2E erweitert:** der E2E-Job startet seit dem Staging zusätzlich `inventar` (:9110),
  `einkauf` (:9111) und `schichten` (:9115) mit `COSMI_MODULE_*_ENABLED=true` (nur auf
  `backend-loop`). Lokal geprüft bis zum Port-Bind (Config und DB ok, `bind` verbietet die
  Sandbox) — abgenommen erst im Morgen-PR.
- **Nicht im Lauf:** sieben Desktop-Hooks mit rohem `fetch` ohne Auth-Header
  (`fe-raw-fetch-without-auth-header` in NEXT). Ein Gateway-Fix hilft dort nicht.

## Was aus Lauf 15 gelernt wurde — vier Dinge, die dich direkt betreffen

1. **Admin überdeckt alles.** Jede Prüfung in Lauf 15 lief als Admin (462 von 463 Keys). Die
   Vorbereitung fand danach ganze Module, die für member 403 liefern. Teste als member
   (Regel 6), und prüfe vor jedem Grant die Datengrenze (Regel 1).
2. **Neue E2E-Flows finden Altbugs.** Der Kalender-Flow fand `include_inactive`, das schon
   vor Lauf 15 kaputt war. Scheitert ein Flow-Schritt, ist das ein Fund — Schritt nicht
   weglassen, Fix-Unit an den Kopf.
3. **Read-Modify-Write ist Datenverlust.** `UpdateItem` schrieb den Bestand aus einem
   ungesperrten Read zurück und löschte parallele Wareneingänge (`8cc079ce`). Zähler, Salden,
   Summen nur in der gesperrten Transaktion ändern.
4. **CI läuft erst im Morgen-PR.** Lokal fehlen `-race` und der E2E-Stack. Tests mit
   Goroutinen und jede E2E-Iteration in `offen:` vermerken.

---
