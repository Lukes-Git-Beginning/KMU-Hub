# Luke-Runbook — Marathon-Tag (Backend-P0 vormittags + FE-Lane nachmittags)

> Dein Handbuch für den Tag. Der **FE-Build-+-Verify-Prozess ist identisch zu Nico** — nicht duplizieren, sondern `../nico-block/RUNBOOK.md` + `../nico-block/WORKFLOW.md` lesen. Dieses Doc nennt nur die Luke-spezifischen Teile.

## 0. Einmal-Setup (falls noch nicht da)
1. Repo klonen (eigener Klon, nicht der von Darien/Nico): `git clone <KMU-Hub-Repo> KMU-Hub-luke && cd KMU-Hub-luke`.
2. Claude Code installieren + im Repo-Root starten (`claude`).
3. FE-Umgebung (`desktop/`): `npm install` · `npm run dev` (Vite :5173, Fenster offen lassen) · Demo-Modus läuft ohne Backend.
4. Playwright (für QA, einmalig, in `desktop/`): `npm install -D playwright && npx playwright install chromium`.
5. Backend-Umgebung: dein üblicher Go-Stack (protoc, Go 1.25, golangci-lint) — wie bisher.

## 1. Tagesstruktur
### Vormittag — Backend-P0
- Quelle: `.planning/backend-handover-luke.md` (nach Launch-Impact sortiert). **P0 zuerst.**
- Deine Domäne: du führst, Claude unterstützt nach `CLAUDE.md`-Architektur-Regeln (Thick Services/Thin Handlers, golang-migrate `make migrate-create`, slog, Prepared Statements, tenant_id, Idempotency).
- Bei 🟢-Punkten (FE wartet nur auf Endpoint): Endpoint bauen → die anderen FE-Ströme ziehen später nach (Mock-Store → Hook).
- Backend-Commits auf deinem üblichen Weg (eigenes Repo/Branch), **nicht** im FE-Strom-Branch.

### Nachmittag — FE-Lane (vertraege → dashboard → profil)
- **Genau wie Nico:** eine Phase = eine Spec, volle 6-Schritte-Schleife (`../nico-block/WORKFLOW.md`).
- Branch: `marathon/luke-fe` (einmal `git checkout -B`, pro Phase Commit + `git push -u origin marathon/luke-fe`).
- Pilot: `phase-01-vertraege-settings.md`. Danach `BACKLOG.md`.

## 2. Die harten Regeln (FE — identisch zu Nico)
Siehe `../nico-block/RUNBOOK.md §1` (i18n ×4, einfache Klammern, ICU-Plural, keine Emojis, keine ASCII-Umlaute, `shared/` wiederverwenden, Motion nur transform/opacity) und `../collision-map.md` (Branch-Modell, nur deine Lane, Hot Files additiv).

## 3. Verify-Checkliste vor „fertig" (FE)
Die **Mandatory-Verify-Checkliste** aus `../nico-block/RUNBOOK.md` ist Pflicht: gescopter tsc grün · QA-Script grün · **Screenshots wirklich angesehen** (Roh-Keys/Emojis/Layout/leere Zustände) · i18n ×4 vollständig · Review-Faden geschrieben.

## 4. Fallen (Luke-spezifisch)
- **Nicht** Backend-Arbeit in den FE-Strom-Branch mischen — getrennt halten.
- **Nicht** `main` anfassen/mergen (Branch-Modell, Darien merged beim Review).
- Bei FE: nur vertraege/dashboard/profil. Wenn du Backend-Bedarf in der FE-Lane siehst → `backend-handover-luke.md` ergänzen, FE mock-first weiterbauen.
