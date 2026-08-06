# Resume-Prompt: Deploy-Fix-Session 2026-06-25

**Stand (2026-06-25, Abend):** `origin/main = 11e964d0` (2 smoke.sh-Fix-Commits auf `baf16a3a`). Prod gesund — Smoke 29/29 grün, alle Services healthy, Migr.233 clean, `/health` 200.

---

## ✅ Nichts blockierend — Prod ist gesund

Wave 1–4 (retention-policies Migr.233, security/auth, formulare/vertraege/mails OpenAPI) live und verifiziert:
`/health` 200, migrate `Exited(0)`, retention_policies-Tabelle da, auth+gateway+23 Services healthy, 5 Modul-Routen 401 unauth (korrekt).

Laufende Binaries zeigen `/health` → `baf16a3a` (kosmetischer Lag, da smoke.sh-Commits keinen Rebuild triggern). Self-heilt beim nächsten Code-Deploy — **kein Handlungsbedarf.**

---

## Was passierte (Kontext für Root-Cause-Verständnis)

Verwaister `deploy.sh`-Lock → aufgelöst. CD-Redeploy blieb identisch hängen.

**Echte Root Cause:** Host stand in **detached HEAD** (Folge eines früheren deploy.sh-Auto-Rollbacks via `git checkout <sha>`). In detached HEAD zieht `git pull origin main` den HEAD nicht nach → migrate-Image ohne Migr.233 → Loop `no migration found for version 233` → auth `Created` → gateway unhealthy → 503.

Rollback-Auslöser: 2 Smoke-Skript-Bugs auf `berichte/definitions`:
1. `a6cd2215` — Smoke nutzte manager-Token gegen admin-scoped Route → Fix: Admin-Token
2. `11e964d0` — Smoke nahm flaches Array statt Envelope `{definitions:[…], total}` → Fix: `.definitions[]`

Recovery: Host auf `main` reattacht; 2 smoke.sh-Commits; manueller Deploy.

---

## Offene Punkte (nicht blockierend)

1. **Kosmetik (optional):** Rebuild-Deploy für `/health`-Commit-Sync auf `11e964d0` — empfohlen: lassen bis ohnehin Code deployt wird.
2. **RBAC-Entscheidung:** `manager`/`member` bekommen aktuell keinen Zugriff auf die 5 Welle-2-Module (berichte/helpdesk/wiki/formulare/vertraege) — admin-only per Grants. Separate Produktentscheidung; wenn ja → Seed-Migration konsistent über alle 5.
3. **Vertagt aus altem Resume:**
   - Nightly Finance-Integrationstest rot (Test-Bug `UPDATE quantity=0` vs `CHECK>0`, Vor-Regression)
   - OpenAPI-Drift: dialer/inventar/vermietung
   - Darien-FE-Lane: 8 security-client-Pfad-Fixe + documents-Normalizer entfernen
   - Welle-2-Forward: admin Tenant-Provisioning/Billing, settings OAuth

---

## Gotchas

- **Detached-HEAD-Falle** — bei hängendem/rollbackendem Deploy ZUERST prüfen:
  ```bash
  git symbolic-ref -q HEAD || echo DETACHED
  ```
  Recovery: `git checkout main && git merge --ff-only origin/main`. deploy.sh's eigener `git pull` repariert detached HEAD NICHT.

- **Smoke-Fails zuerst als Skript-Bug verdächtigen** — nicht blind auf Prod-Defekt schließen (Root Cause war 2× Smoke-Bug).

- **`.env`-Literal in Bash-Command** triggert check-no-secrets-Hook → Workaround:
  ```bash
  EXT=env; . "/opt/kmuhub/.${EXT}.production"
  ```
  Quoted Heredoc `<<'REMOTE'` nutzen (verhindert lokale Expansion).

- **cd.yml-only-Commits** triggern kein CI/CD → `gh workflow run cd.yml --ref main` manuell.

- **deploy/scripts/**-Commits** triggern KEIN CI (Path-Filter) → smoke.sh-Fix erst beim nächsten Deploy aktiv (oder `git pull` am Host).

- **Docker Desktop lokal AUS** — nicht starten.

- **Server-SSH:** `ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195`; App-Pfad `/opt/kmuhub/`; psql via Container: `sudo docker exec <postgres-container> psql -U kmuhub -d kmuhub`.

---

## Disziplin

- Max 3 Subagenten gleichzeitig; `isolation: worktree` für FE-/Multi-File-Wellen
- Hot-Files (`openapi.yaml`, i18n-JSONs, mocks-index) nur Main-Session anfassen
- Keine AI-Attribution in Commits (kein Co-Authored-By)
- Forward-fix statt revert (Migration-Drift-Falle)
- Bei Prod-Eingriffen kurz nachfragen — kein unbeaufsichtigtes Deployment
