# Parallel-Batch — Koordination (Main + Sub) — dialer + security

> **Stand 2026-06-24 (Batch 5).** Zwei Terminals bauen parallel, damit sie sich **nicht in die Quere kommen**.
> Vorheriger Batch (berichte + notifications, Batch 4) ist durch → notifications/berichte review-reif/echt-geschaltet.
> **Main = dialer** (Welle-1 **Echt-Schaltung** gegen lokales Docker-Backend — Supervisor-Overview + protojson-Enum-Mapping). **Sub = security/DSGVO** (Welle-2 FE-Tiefe, **MSW-only, kein Backend**, P0-Launch-Blocker).

## Rollen & Klone

| | **Main-Terminal** | **Sub-Terminal** |
|---|---|---|
| Klon | `…/KMU Hub` (Hauptklon) | `…/KMU-Hub-review` (zweiter Klon) |
| Dev-Port | **5173** | **5174** |
| Modul | **dialer** (echt-Schaltung) | **security** (S-0…S-6) |
| Mode | `--mode localbackend` (DEMO_MODE=false, :8080) | Demo-Mode (MSW, **kein Backend**) |
| Docker | **Main fährt Docker** (dialer-Service `up -d --no-deps`) | **NIE Docker** (OOM-Schutz) |
| Plan-Datei | (Main-Lane, ad-hoc) | `sub-security.md` |
| QA-Protokoll | (Main) | `qa-security.md` (selbst anlegen) |
| Branch | **`main`** (Hauptklon) | **`parallel/security`** (Iso, NICHT direct-to-main) |
| Zusatzrolle | plant Sub-Paket, beantwortet S-0-Klärungen, merged am Ende `parallel/security` | erst S-0 Research+Gate → STOPP → dann S-1…S-6 autonom, meldet `S-x fertig, n/6` |

## Lane-Trennung — was kollidiert NICHT
- **Modul-Code disjunkt:** `modules/dialer/` vs. `modules/security/`. Kein Overlap.
- **MSW-Handler disjunkt:** `mocks/handlers/dialer.ts` (Main) vs. `mocks/handlers/security.ts` (Sub) — beide bereits in `index.ts` registriert → **kein `index.ts`-Touch** von beiden Seiten.
- **Docker:** nur Main. Sub ist MSW-only → kein zweiter Stack, kein OOM.
- **shared/ einfrieren:** beide **konsumieren nur** `shared/DetailModal` + `shared/SortMenu`. Niemand baut neue `shared/`-Komponenten.

## Die echten Reibungspunkte (Regeln verbindlich)
1. **i18n — an Objektgrenze, durch Branch-Iso entschärft.** dialer-Keys unter `dialer.*`; security unter `security/gdpr/audit/ipAccess/vault/password/session/dsar.*`. **Disjunkte Prefix-Cluster.** `{var}` single-brace ×4 Sprachen, ICU-Plural. Beim finalen Merge beide Key-Blöcke behalten, danach `npm run build`.
2. **`module-settings-registry.tsx`** — beide fügen je **einen** Eintrag (Main `dialer`, Sub `security`). Anderer `id` → beim Merge beide behalten.
3. **Branch-Isolation:** Sub baut auf `parallel/security`, Main auf `main` → null Live-Konflikt. Main merged am Ende einmal kontrolliert.
4. **App.tsx-Routen:** dialer-Route existiert (Main fasst sie nicht an). security nur minimal-invasiv falls nötig → niedrige Kollision.

## Build-+-Verify-Standard — pro Phase IMMER (CLAUDE.md)
bauen → i18n ×4 (`{var}`, nie `{{var}}`; Plural als ICU) → MSW/Demo-Daten → Compile-Gate (`npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error`, **nie `| tail`**) → **Playwright-Screenshot-QA + Bilder WIRKLICH ansehen** → iterieren bis grün → **ein Commit + Push** → Eintrag in `qa-<modul>.md`.

## Gates & Fallen (bewährt)
- **Build-Gate IMMER mit echtem Exit. NIE `npm run build | tail`** — `$?` wäre tails Exit (immer 0).
- **Dev-Server killen (Windows):** PowerShell `Get-NetTCPConnection -LocalPort 5174` + `Stop-Process` (`pkill -f vite` greift in Git Bash NICHT). Nur 1 Dev-Server pro QA-Runde.
- **Playwright-Klick auf 20px-Icons timeout't** → JS-Klick-Fallback `locator.evaluate(el => el.click())`.
- Neue Dateien unter `mocks/data/` brauchen `git add -f` (`.gitignore` ignoriert `data/`).
