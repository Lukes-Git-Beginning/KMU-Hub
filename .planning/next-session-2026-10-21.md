# Prompt für die nächste Session (vorbereitet 2026-10-20)

> Diesen Block in eine neue Session kopieren. Er setzt voraus, dass Lauf 14 in der Nacht 20./21.10.
> gelaufen ist. Falls nicht, Schritt 0 überspringen.

```
Kontext (Stand 20.10.2026, gemessen):
- Gesamtbewertung liegt als Doc vor: https://claude.ai/code/artifact/cd78f47f-b2a5-45cf-a016-ae46b822bde3
  Empfehlung dort: 90-Tage-Test (fokussierter Hybrid) bis Ende Januar 2027. Technisch heißt das:
  Pilot-Readiness, nicht Feature-Bau.
- Produktion (SSH, nur lesend, 20.10.): 1 Tenant (…-0001), 219 Nutzer davon 196 Testkonten,
  5 Kontakte, 6 Tickets. Seit dem 22.09. kein einziger Login. Backups täglich, aber NUR lokal
  in /opt/kmuhub/backups (kein Offsite). Modul-Flags an: video, helpdesk, wiki, berichte,
  formulare, vertraege — Schichten NICHT, obwohl auf der Preisliste buchbar.
- Lauf 14 (Nachtloop) lief in der Nacht 20./21.10. auf backend-loop.

Arbeite in dieser Reihenfolge, verify-first, eine Einheit = ein Commit, Review-Gate nach jeder
Einheit (kurz zeigen, was geändert wurde, bevor die nächste startet):

0. Lauf 14 nachbereiten: `git log --oneline main..backend-loop`, JOURNAL.md, blocked-Units
   (das sind Befunde, keine Ausfälle). Funde aus dem Journal, die nicht im Backlog stehen, als
   Units anlegen und nach Schwere einsortieren. CI am Branch grün bekommen (kein -race lokal,
   CI sieht Races), dann nach main mergen. Vor einem PR die beiden Claude-Review-Workflows
   deaktivieren (sie zünden auch bei Draft).

1. Offene Registrierung schließen. `POST /api/v1/auth/register` (internal/gateway/route_auth.go:62)
   ist öffentlich, ohne Captcha, und legt jeden als `member` im Standard-Tenant an
   (internal/auth/service.go:94). In Prod haben sich am 28.08. bereits zwei externe Adressen
   registriert. Haken: Smoke- und E2E-Tests registrieren selbst über diesen Endpunkt
   (backend/test/smoke/helpers_test.go:118/183, smoke_test.go:172, backend/test/e2e/*).
   Erst die Optionen mit mir abstimmen (z. B. Env-Schalter default aus + Tests schalten ihn in
   ihrer Umgebung an; oder nur noch Einladung durch Admin), DANN bauen. Regeln: Compose-
   Passthrough pro Service prüfen, keine neue config.RequireX vor Passthrough+Werten,
   openapi.yaml mitziehen + swagger-cli validate.

2. Nightly ist seit 04.10. rot: Job "Smoke Tests" scheitert im Step "Start MinIO", obwohl
   nightly.yml:86 schon quay.io/minio/minio nutzt. `gh run list --workflow nightly.yml`,
   `gh run view <id> --log-failed`, Ursache finden, fixen, per `gh workflow run` verifizieren.

3. Offsite-Backup + Restore-Drill (1.0-Kriterium 5). backup.sh hat einen age-verschlüsselten
   Offsite-Pfad (opt-in). Ich buche die Hetzner Storage Box; du sagst mir genau, welche Werte
   in /opt/kmuhub/.env.production gehören. Schreibzugriffe auf den Server NUR nach meiner
   Freigabe pro Schritt. Danach einen Restore in eine Wegwerf-Instanz mit Zeitmessung und
   das Ergebnis in docs/operations/BACKUP.md.

4. Scans rot (npm audit „shipped Electron runtime“, trivy HIGH): nur prüfen und mir sagen, ob
   es ein Ein-Commit-Fix ist. Nicht anfangen, wenn größer.

5. Schichten: buchbar laut Preisliste, in Prod abgeschaltet (COSMI_MODULE_SCHICHTEN_ENABLED fehlt
   im Compose-Passthrough und in .env.production). Nur die Entscheidungsfrage vorbereiten
   (anschalten vs. von der Preisliste nehmen), nicht bauen.

Am Ende: Memory aktualisieren (MEMORY.md ist über Budget, vorher Altes nach memory/archive/
verschieben), pushen.
```
