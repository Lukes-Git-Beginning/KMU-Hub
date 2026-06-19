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

---

## P-2 — Dokumente-Tab echt (MSW)  ✅ fertig
**Build:** `npm run build` → EXIT 0 (`✓ built in 1m 25s`, 0 TS/Rollup-Fehler).
**Geändert:**
- `mocks/handlers/hr.ts`: neue Handler (camelCase wire shape, kein Adapter im
  hr-client) — `GET …/employees/:id/documents/categories` (5 Kategorien),
  `GET …/employees/:id/documents` (7 Demo-Docs: Arbeitsvertrag,
  3× Gehaltsabrechnung, 2× Zertifikat, Bescheinigung — mit Kategorie, Dateiname,
  Größe, Datum, Uploader, Notizen), `POST …/documents` (Upload → neues Doc,
  **stateful** in der Session). `categories` vor der Liste registriert
  (Shadowing-frei). `/employees/:id/documents*` ist exklusiv hr.ts (geprüft).
- `api/hr-types.ts`: additiv `EmployeeDocument.fileSize?`, `UploadDocumentInput.fileName?/fileSize?`.
- `modules/profil/tabs/DokumenteTab.tsx`: `handleUpload`-Toast-Stub raus →
  echter Upload-Dialog (Datei + Kategorie + Notiz) über `useUploadEmployeeDocument`;
  Preview-Toast-Stub raus → zentriertes `shared/DetailModal` (Metadaten +
  Platzhalter-Vorschau + Demo-Vorschau-Badge + sticky Download-Footer);
  Download-Toast-Stub raus → echter Blob-Download (Muster team `PersonnelDocuments`);
  Doc-Zeilen ganze Zeile klickbar (`role=button`, Enter/Space, aria-label).
- i18n: +13 `profil.documents.*`-Keys ×4 (upload/preview/fileSize/uploaded/note/
  openDocument/selectFileFirst), single-brace `{name}`, via untracked
  `scripts/add-profil-i18n.mjs` ins zusammenhängende `profil.documents.`-Sub-Cluster
  einsortiert (voller `profil.*`-Cluster ist NICHT zusammenhängend — zeiterfassung verstreut).

**Verify (Screenshots angesehen, `qa-profil-docs.mjs` + `qa-profil.mjs`):**
- Liste: **7 Demo-Docs** mit PDF-Badge/Datum/Größe/Kategorie/Uploader/Notiz;
  Sidebar-Counts korrekt (Arbeitsvertrag 1 / Gehaltsabrechnungen 3 / Zertifikate 2 /
  Bescheinigungen 1 / Sonstiges 0).
- Zeilen-Klick → **DetailModal-Preview** (Demo-Vorschau-Badge, Metadaten,
  „Im Produktivbetrieb…"-Hinweis, sticky **Herunterladen**).
- Upload-Dialog → Datei (synthetisch) → submit → **8 Docs**, neues „Test-Zeugnis_2026.pdf"
  oben (Von Stefan Vogel), Erfolgs-Toast „Dokument hochgeladen", überlebt im Session-State.
- Download = client-seitiger Blob (wie team), Preview + Zeile verkabelt.
- Raw-Keys/`{{var}}`: **0** (DE+EN, beide Viewports). Page-Errors: **0**.
  Console: `documents`-Connection-Refused **weg**; nur noch app-globales feature-flags.

**Commit:** `feat(profil): wire documents tab to MSW (list/upload/preview/download)`

---

## P-3 — Avatar-Upload demo-real (MSW) + DND-Fallback  ✅ fertig
**Build:** `npm run build` → EXIT 0 (0 TS/Rollup-Fehler).
**Geändert:**
- `mocks/handlers/hr.ts`: `POST …/employees/:id/avatar` (Demo — gibt die hochgeladene
  Data-URL als gespeicherte URL zurück).
- `modules/profil/tabs/ProfilTab.tsx`:
  - **Avatar**: `handleAvatarFile` führt den Upload jetzt über eine `useMutation`
    gegen den MSW-Endpoint (statt nur lokale Data-URL); on success `updateProfile`
    + lokale Persistenz bleibt (überlebt Reload). „Upload folgt…"-Disclaimer entfernt
    (`profil.avatar.saved` = „Profilbild gespeichert."), neuer `profil.avatar.error`.
  - **DND**: kein `disabled` mehr im Demo — Backend-Pfad (MSW `/notifications/dnd`
    ist stateful gemockt) wenn erreichbar, sonst **lokaler Demo-Fallback** (`dndDemoActive`),
    der den Schalter umschaltbar hält + den Zustand sichtbar zeigt; Status-Text spiegelt
    den aktiven Zustand (statt „Backend nicht erreichbar").
- i18n: `profil.avatar.saved` aktualisiert + `profil.avatar.error` ×4 (via add-Script,
  Multi-Sub-Cluster: profil.documents. + profil.avatar.).

**Verify (Screenshots angesehen, `qa-profil-avatar-dnd.mjs`):**
- Avatar: synthetisches PNG → `img[src^="data:image"]` erscheint (0→1) **und überlebt
  Reload** (1) — Upload läuft über MSW, lokale Persistenz greift.
- DND: Switch **enabled** (nicht disabled), aria-checked false→true, Status „Deaktiviert"
  → „Aktiv", Toast „Bitte-nicht-stören aktiviert".
- Raw-Keys/`{{var}}`: 0. Page-Errors: 0. Console: nur feature-flags (Demo-Lücke).

**Commit:** `feat(profil): route avatar upload through MSW + add DND demo fallback`

---

## P-4 — Dead-Code-Cleanup (verwaister zeiterfassung-Ordner)  ✅ fertig
**Build:** `npm run build` → EXIT 0 (0 „Could not resolve"/TS/Rollup-Fehler).
**Abgesichert vor Löschung:** grep über `src/` — **kein** Import (absolut, relativ
`./`, `../`, oder via Pfad-Substring `tabs/zeiterfassung`) zeigt in den Ordner;
`ZeiterfassungTab.tsx` nutzt ausschließlich `@/modules/zeiterfassung/components/*`
(das echte Modul) + hr-hooks. → 11 verwaiste Dateien gelöscht
(`TodayView/WeekView/MonthView/ReportsView/TeamView/OverviewView/CategoriesView/
ApprovalBanner/ExportDialog/ManualEntryForm/time-utils`).
**i18n-Purge (konservativ):** keine dynamische/konkatenierte `profil.zeiterfassung.*`-
Nutzung (geprüft) → statisches Greppen zuverlässig. Pro Key Live-Usage über den
**gesamten** ts/tsx-Tree gezählt: **151 von 199** Keys mit **null** Live-Referenz
(gehörten exklusiv zu den gelöschten Views) → in allen 4 Sprachen entfernt; die
**48** aktiv genutzten (echtes zeiterfassung-Modul + hr-hooks) **behalten**.
Zeilenbasierte Entfernung + JSON-Validierung. Untracked Helper
`scripts/purge-profil-zeiterfassung-i18n.mjs` (recomputet das Dead-Set).

**Verify (Screenshots angesehen):**
- Build grün, keine Missing-Import-Fehler.
- Zeiterfassung-Tab unverändert funktional: Heute (KPIs + Einträge), Woche,
  **Auswertungen** (KPIs, Stunden-pro-Tag-Chart, Nach-Projekt, Abrechenbar-Donut),
  Team, Korrekturen — alle gerendert, **0 Raw-Keys** in allen Sub-Tabs (1440+1024).
- Da entfernte Keys 0 Live-Referenzen hatten, kann keine View einen Raw-Key zeigen.

**Commit:** `refactor(profil): remove orphaned zeiterfassung folder + 151 dead i18n keys`
