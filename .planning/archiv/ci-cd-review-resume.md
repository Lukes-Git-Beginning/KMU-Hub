# Übergabe-Brief: CI/CD-Pipeline-Review (GitHub Actions)

**Für die nächste Session.** Self-contained — als Prompt nutzbar (oben anpinnen, dann „leg los").
Sprache: Deutsch erklären (User ist *kein* CI/CD-Experte und möchte verstehen, nicht nur Ergebnisse),
Code/Identifier Englisch. Erstellt 2026-06-25.

## Ziel

Das **komplette GitHub-Actions-Gerüst** einmal gründlich anschauen und dem User **erklären**:
1. **Was ist da?** Jeden Workflow in einem Satz: was triggert ihn, was tut er, wie lange läuft er.
2. **Was ist gut / was kann besser / was kann weg?** Konkret, priorisiert (keep / improve / remove).
3. **Ist die Lauflänge normal?** ← Kernfrage des Users. Er kann CI-Laufzeiten nicht einordnen.
   Reale Zahlen holen (s.u.) und gegen branchenübliche Werte für einen Go-Microservice-Monorepo +
   Electron-Frontend einordnen. „X min für N Services ist (un)normal, weil …".
4. **Optimierungspotenzial:** Caching, Parallelität, Path-Filter, Matrix, Job-Splitting, Runner-Größe.

**Erst Review + Erklärung. Änderungen NUR nach expliziter Freigabe** (lean — kein blindes Umschreiben
funktionierender Pipelines; CI-Rot blockiert Prod-Deploys).

## Die 7 Workflows (`.github/workflows/`)

`ci.yml` · `ci-desktop.yml` · `cd.yml` · `nightly.yml` · `scans.yml` · `claude-pr.yml` · `security-review.yml`

Alle zuerst **vollständig lesen**, bevor bewertet wird — nicht aus den Dateinamen schließen.

## Bekannter Kontext (aus MEMORY.md — verifizieren, kann veraltet sein)

- **Pipeline-Split (2026-06-09, `8f6aaa32`):** per-push `ci.yml` = nur lint / test (`-race`) / e2e /
  openapi. Smoke → `nightly.yml`. gosec/trivy/npm-audit → `scans.yml` (weekly + bei Dependency-Change).
  → Schwere Jobs wurden bewusst aus dem per-push-Pfad rausgezogen. Prüfen ob das sauber umgesetzt ist.
- **CD-Mechanik:** Push auf `main` → `ci.yml` grün → **`cd.yml` auto-deployt auf Hetzner-Prod**
  (Migration läuft auf Prod-DB). `deploy/scripts/deploy.sh` baut **seriell** (16 GB RAM, sonst OOM).
  Deploy-Job-Timeout zuletzt 20→40 min angehoben (`baf16a3a`). → Build-Dauer ist ein realer Schmerzpunkt.
- **Bekannte Pathologie „Zero-Step-Billing-Wall":** Jobs failen 0-Steps/~2s mit „log not found" wenn
  GitHub-Actions-Minuten/Billing aufgebraucht → Step-Counts via `gh run view <id> --json jobs` prüfen,
  nicht Logs. → Im Review erwähnen: laufen wir ins Minuten-Budget? (`gh api` Billing, s.u.)
- **`.knowledge`-only-Commits triggern kein CI/CD.** Prüfen ob `.planning/`-/Doc-Pfade ebenfalls
  ge-path-filtert sind (dieser Commit hier ist ein Live-Testfall — lief CI für ein `.planning`-Doc?).
- **Migrations-Drift-Fallen:** Revert-nach-Deploy + detached-HEAD-Rollback (s. MEMORY.md Infra-Sektion).
  Relevant für die Bewertung der `cd.yml`-Robustheit (Auto-Rollback-Mechanik).

## Konkrete Analyse-Schritte

1. **Lesen:** alle 7 YAMLs + `deploy/scripts/deploy.sh` + ggf. `backend/.golangci.yml`,
   `backend/Makefile` (CI ruft Make-Targets), `desktop/package.json` (FE-Scripts).
2. **Reale Lauflängen holen** (das beantwortet die Kernfrage):
   ```bash
   gh="/c/Program Files/GitHub CLI/gh.exe"
   "$gh" run list --workflow ci.yml -L 20 --json displayTitle,conclusion,createdAt,updatedAt,databaseId
   "$gh" run list --workflow cd.yml -L 20 --json displayTitle,conclusion,createdAt,updatedAt
   "$gh" run list --workflow ci-desktop.yml -L 20 --json conclusion,createdAt,updatedAt
   # Job-/Step-Dauer eines konkreten Runs:
   "$gh" run view <databaseId> --json jobs
   ```
   Aus `createdAt`→`updatedAt` die Wall-Clock je Run; Median + Ausreißer bilden. Pro Workflow.
3. **Minuten-Budget prüfen** (Billing-Wall-Verdacht):
   `"$gh" api /repos/Lukes-Git-Beginning/KMU-Hub/actions/billing/usage` bzw. user/org-Billing-Endpoint.
4. **Einordnen vs. Best Practice:** Cache-Hits (Go-Build-Cache, `actions/setup-go` cache, npm/electron
   cache)? Redundante Checkouts/Setups? Sinnvolle `concurrency`-Group (Cancel-in-progress)? Path-Filter
   korrekt? `permissions:` minimal? Pinned Action-SHAs vs. Tags? Matrix sinnvoll genutzt?
5. **Output an den User:** (a) Tabelle „Workflow → Trigger → was → Median-Laufzeit → Urteil", (b)
   priorisierte Liste keep/improve/remove mit Begründung, (c) klares Ja/Nein zur Lauflänge-Frage mit
   Vergleichszahlen, (d) Top-3-Quick-Wins. Dann fragen, ob umgesetzt werden soll.

## Tooling-Pfade

- gh: `"/c/Program Files/GitHub CLI/gh.exe"` · Go: `export PATH="$PATH:/c/Program Files/Go/bin"`
- Repo: `github.com/Lukes-Git-Beginning/KMU-Hub` (private), Branch `main`, direct-to-main Default.
- Prod-Deploy ist scharf: Push auf main = potenzieller Auto-Deploy → bei Workflow-Änderungen
  vorsichtig, vor Push rückfragen. `cd.yml`-Edits können Prod-Deploys brechen.

## Querverweise

MEMORY.md → „Infrastruktur (Hetzner Production)" + „Hetzner Production Server" (Deploy-Details,
bekannte Fallen). `.knowledge/deployment.md` (Docker, CI/CD, Hetzner). `deploy/scripts/deploy.sh`.
