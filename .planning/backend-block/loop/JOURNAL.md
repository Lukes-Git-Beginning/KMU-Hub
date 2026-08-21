# Backend-Nachtloop — Journal Lauf 10

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37, inkl. `logs/`).

---

## Laufkontext

- **Ausgangspunkt:** `backend-loop` auf `origin/main` gemergt (nicht rebased). `main` steht auf
  dem Stand der Lauf-10-Vorbereitung; CI grün auf `1b49a1f3` (Run 32508843776), CI Desktop grün
  (32508843728), CD grün auf `390a75f5`.
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = **314**, `schema_migrations` clean. Nächste freie
  Nummer wäre 315 — aber immer zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E '^[0-9]{6}' | sort | tail -1`.
  Genau **zwei** Units dieses Laufs bringen eine Migration mit:
  `fix-gobd-belegarchiv-worm-enforcement` (REVOKE-only) und
  `feat-retention-worker-schema-and-engine` (neue Tabelle + RLS).
- **Lokale DB:** läuft und ist ab diesem Lauf **Startbedingung**. `run-loop.ps1` prüft im Vorflug
  Port 5432, die Anmeldung als `kmuhub_app` und den Migrationskopf und bricht ab, wenn eins davon
  fehlt. Grund: `testutil.SkipIfNoDB` (`backend/internal/testutil/rls.go:24`) fragt nur, **ob**
  `DATABASE_URL` gesetzt ist. Am 2026-08-22 an `internal/security/audit` gemessen — mit DB
  33 PASS / 0 SKIP, ohne die Variable 19 PASS / 14 SKIP, beide Male `ok` und Exit 0.
  Der Treiber exportiert `DATABASE_URL` für die Kindprozesse; der `export` aus `ITERATION.md`
  Schritt 5 bleibt trotzdem richtig und schadet nicht.
- **Rolle:** `kmuhub_app` (NOSUPERUSER NOBYPASSRLS), niemals `kmuhub` — der Superuser hat
  BYPASSRLS und würde jede RLS-Lücke durchwinken.
- **Coverage-Ausgangslage:** gesamt **60,2 %** bei Gate 15 %, `internal/gateway` **46,1 %** als
  schwächstes Kernpaket. Vollständige Paketliste im Kopf von `BACKLOG.yml`.
- **Umfang:** 48 vorab geschriebene Units — Block A (14, Datenschutz-Mechanik), Block B (25,
  Gateway-Härtung), Block C (9, Muster-Scans). Block C legt weitere Units zur Laufzeit an; in
  Lauf 9 haben 11 Scans 16 Zusatz-Units erzeugt.

## Was in diesem Lauf gilt

- Pin-Tests umdrehen, nicht löschen. Mutations-Probe ist Pflicht.
- Root Cause statt Symptom: vor jedem Fix alle Caller greppen.
- Eine Prämisse aus dem Backlog, die sich am Code als falsch erweist, wird **nicht trotzdem
  gebaut** — sie wird hier widerlegt und die Unit auf `blocked` gesetzt. Befund 1 im
  `BACKLOG.yml`-Kopf ist genau so entstanden: der angeblich fehlende Foreign Key auf
  `contact_tags` existiert seit Migration 000007.
- Scan-Units ändern kein Verhalten. `neue-units:` muss IDs nennen, die wirklich in
  `BACKLOG.yml` stehen.
- Gesperrt: Frontend/Desktop, CSAT und Public-Token-Routen, Dependency-Bumps, `deploy/`,
  Migrations-Umnummerierungen, Preise und Modul-Zuschnitt.

---
