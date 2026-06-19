# QA-Protokoll — profil (Sub-Terminal, Branch `parallel/profil`, Dev-Port 5174)

> QA-Harness: `desktop/scripts/qa-profil.mjs` (Playwright) gegen den QA-Vite-Server
> (`desktop/vite.qa.config.mjs`, demo-mode MSW, :5174). Seedet Demo-Auth-Tokens →
> Auth-Store lädt den echten Demo-User (Stefan Vogel / usr-e1 via `/auth/me`),
> stubbt WebSocket, überspringt Onboarding, erzwingt Locale. Screenshots aller 4
> Tabs @ 1440 + 1024, Raw-Key-/`{{ }}`-Scan, Console-/Page-Error-Sammlung.
> Lauf: `node scripts/qa-profil.mjs [de|en|fr|it]`.

## Bekannte Demo-Lücken (nicht profil-Scope, nicht durch diesen Batch verursacht)
- `GET /api/v1/feature-flags` ist app-global **nicht** gemockt → 1× `ERR_CONNECTION_REFUSED`
  beim Start (Feature-Flag-System ist OFF, ADR). Tritt überall auf, nicht nur in profil.
- `GET /api/v1/hr/employees/:id/documents` war vor **P-2** ungemockt → `ERR_CONNECTION_REFUSED`
  im Dokumente-Tab. Wird in P-2 geschlossen.

---

## P-1 — current-user-Single-Source + Account-Info  ✅ fertig
**Build:** `npm run build` → EXIT 0 (`✓ built in 50.76s`, 0 TS/Rollup-Fehler).
**Geändert:**
- `stores/settings.ts`: Profil-Seed **Darien Morales → Stefan Vogel** (Name/E-Mail/
  Telefon/Position=Geschäftsführer/Bio), neues Feld `joinedAt: '2021-03-01'`. Mail-Seed
  (username, signature, imap/smtp-Host) auf `techvision.de`/Stefan Vogel angeglichen.
  Kommentar verweist auf `CURRENT_USER` (shared-ids) als Single Source of Truth.
- `modules/profil/tabs/ProfilTab.tsx`: Account-Info zeigt wieder **„Mitglied seit"**
  (toter Key `profil.info.memberSince` reaktiviert), Wert aus `profile.joinedAt`,
  Monat+Jahr in aktiver Locale (`März 2021` / `March 2021`).
- `modules/profil/tabs/AbwesenheitenTab.tsx`: **Crash gefixt** — der Tab erwartete
  camelCase-`LeaveRequest`, der Demo-Handler (team.ts, registriert vor hr.ts, gewinnt)
  liefert snake_case und ignoriert `employee_id`. Folge: `b.createdAt.localeCompare`
  auf `undefined` → harter Crash via ModuleErrorBoundary (reproduzierbar im echten
  Demo, sobald ein Auth-User geladen ist). Normalizer am Komponenten-Rand
  (snake/camel tolerant) + clientseitiger Filter auf den aktuellen User; Balance-Reads
  (`entitlement/taken` ↔ `totalEntitlement/used`) ebenfalls toleriert.

**Verify (Screenshots angesehen):**
- Profil-Tab @1440/1024 (DE+EN): Header **Stefan Vogel / Geschäftsführer /
  Administrator / Online**, Form = Stefan/Vogel/stefan.vogel@techvision.de/
  +49 151 2345 6789/Geschäftsführer/Bio. **Sidebar unten + Topbar = Stefan Vogel**
  (konsistent, Auth-Store-Single-Source).
- Account-Info: Rolle=Administrator · E-Mail=stefan.vogel@techvision.de ·
  **Mitglied seit = März 2021** (EN: „Member since"). Kein „Darien"/„Morales" mehr.
- Abwesenheiten-Tab: rendert ohne Crash, zeigt **nur** Stefans 3 eigene Anträge
  (Urlaub ausstehend, Homeoffice + Sonderurlaub genehmigt), Filter-Counts korrekt
  (Alle 3 / Ausstehend 1 / Genehmigt 2), Resturlaub **17 / 30** konsistent.
- Raw-Keys / `{{var}}`: **0** über alle 4 Tabs, beide Viewports, DE+EN.
- Page-Errors: **0**. Console-Errors: nur die o.g. Demo-Lücken (feature-flags, documents).

**Commit:** `feat(profil): seed current-user as Stefan Vogel + member-since, fix absences data contract`
