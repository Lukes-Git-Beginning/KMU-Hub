# RESUME — nächster Einstieg (Stand 2026-07-14, Session-Start #7)

> **★★★★★ WIEDEREINSTIEG NACH PAUSE — main `adfa75ff` (gepullt, sauberer FF von +19). Darien war weg; in der Pause hat FAST NUR LUKE gebaut (23 von 25 Commits seit #6). Kein neuer eigener FE-Bau in der Pause.**
>
> **Was Luke seit #6 (05.07.) gemacht hat — drei Kampagnen, alle CD-deployt:**
> 1. **Bexio Invoice-Pull Welle 3b** (`1f8475a7`→`e19a68fb`, `eae34f42`) — Read-only-Mirror: externe Bexio-Rechnungen werden gespiegelt (Migr. **000243**, `UpsertImported`). **FE via Worktree-Subagent gebaut:** `bexio-types.ts` auf flache Wire-Shape, `bexio-client.ts` Pfad-Drift gefixt (`/oauth/authorize`, `/sync/trigger`), Invoice-Pull-Toggle im Wizard + Config-Persist, **Read-only-Badge/Banner auf `source='bexio'`-Rechnungen, alle mutierenden Aktionen ausgeblendet** (Edit/Send/RecordPayment/MarkPaid/Cancel/Storno). i18n ×4, Screenshot-QA gemacht. → **DARIEN-REVIEW-KANDIDAT** (Subagent-Bau, noch nicht von Darien angesehen).
> 2. **ProtoTimestamp/protojson-Kampagne** (~13 Commits, 3 Runden) — Gateway serialisierte Proto-Timestamps als `{seconds,nanos}` (brach FE-Datumsparsing gegen echtes BE). Jetzt via **protojson** über ~alle Module: rapporte/inventar/crm · auth/berichte/automation · fuhrpark/einkauf/produktion · security/formulare/settings · vermietung/schichten/booking/integration · hr map-envelope · biz/finance. plugin/lexware/datev-Protos echt regeneriert. **crm/chat/email waren KEIN Bug** (Timestamps schon `string`).
> 3. **FE/BE-Contract-Mismatch-Kampagne** (`c13586a3`→`39f6393c`→`e8bb19df`, 6 Baustellen/3 Wellen) — Schwester-Klasse: FE-Client las falsche Wire-Shape (nested vs flach, camelCase vs snake_case, falsche URL-Pfade), MSW spiegelte die falsche Erwartung. Bereinigt: **hr/Leave** (Envelope-Unwrap + POST-Body camelCase→snake) · **Integrations** (BE-Fix `HandleGetLinkStatus` ehrt `{platform}`) · **auth/2FA+Sessions+Audit** (`security-client.ts` Pfade/Bodies) · **automation** (echte Stats) · **produktion** (Envelope-Unwrap) · **formulare** (Drilldown auf 4 echte BE-Zähler gekappt, fiktive Analytics raus, 18 tote i18n-Keys). Referenz-Clients sauber: `helpdesk-client`/`booking-client`/`hr-client`.
>
> **★ WAS DAS FÜR DARIENS FE-TRACK BEDEUTET:** Die Wire-Shape-Mismatches waren der Haupt-Blocker der Echt-Schaltung (Welle 1). Luke hat sie modulweit **vorab gefixt** → hr/security/automation/produktion/formulare/integration-Clients sind jetzt auf echte BE-Realität ausgerichtet. Diese Module sind damit **deutlich näher an sauberer Echt-Schaltung** als der Plan (Stand 28.06.) sagt. ⚠ Aber: Luke konnte **keine Electron-Screenshot-QA** fahren (GUI nicht headless erfassbar) → visuelle Verifikation der bereinigten Module steht aus.
>
> **★ DIESE SESSION #7 GEBAUT (Welle 2, dialer Demo-Tiefe — gepusht, Auto-Deploy):** dialer als erstes Welle-2-Modul komplett auf review-reif. **D-A** `AgentDetailModal` (Supervisor-Zeile klickbar → Status/Calls/letzte-5-Anrufe, gefiltert aus `recent_calls`) · **D-B** Workspace-Idle shared `EmptyState` + differenzierter Leer-Fall (CTA „Zu den Kampagnen" / „Kampagne wählen") · **D-C** war schon `ModuleSettingsShell`+registriert (nur verifiziert) · **D-D** `dialerPrefs`→personal-only backend-persistiert + neuer `dialerTenant`-Store (tenant, role-gated), beide im zentralen Hydrator (X-4-Split #1 von 5 gemischten) · **D-E** CampaignForm Mode-Erklärung + ContactQueue `SortMenu` + **filter-aware EmptyState** (fixt Fehlbeschriftung „Keine Kampagnen"→Kontakt-Copy). Bonus: `dialer-normalize.ts` Baseline-Typfehler. Gates grün (scoped tsc/eslint/i18n×4/Screenshot-QA angesehen). Screenshots `desktop/.qa-screenshots/dialer-tiefe/`, QA `scripts/qa-dialer-tiefe.mjs`+`qa-dialer-settings-panel.mjs`. ⚠ **DialerSettings-Panel-Screenshot vom CosmiLaunch-Dev-Artefakt verdeckt** (Overlay-Navigation re-triggert LaunchOverlay im Dev-Server; Panel-Content per DOM-Assertion bestätigt).
>
> **★ AUCH SESSION #7 GEBAUT (X-4-Store-Splits KOMPLETT, `d0d7dfa6`, gepusht):** die 4 restlichen gemischten Prefs-Stores gesplittet (automatisierung/mail/formulare/berichte) → je personal-Store (user, backend-persist) + neuer `*Tenant`-Store (tenant, role-gated), alle 8 im zentralen Hydrator. Consumer außerhalb der Panels mit umgehängt (FormularePage 5 Tenant-Felder + `DEFAULT_CONSENT_TEXT`/`_PRIVACY_URL` → formulareTenant; ScheduleReportModal `allowedFormats` → berichteTenant). tsc/eslint grün, Smoke (`scripts/qa-x4-splits-smoke.mjs`): keine pageerrors, alle 4 Tenant-Sektionen rendern. **→ X-4-Settings-Rollout jetzt VOLLSTÄNDIG** (alle ~18 Stores backend-persist + swap-ready). ⚠ Modul-IDs (`automatisierung`/`mail`/`formulare`/`berichte`) brauchen Backend-settings-Registry-Einträge (Luke).
>
> **★ OFFEN / NÄCHSTE UNIT:** (a) **video-tote-Buttons** (VideoPage CallHistory/Header ohne onClick + fehlendes Detail-Modal — klarer Qualitätsmangel, review-reif machen) · (b) **Bexio-Invoice-Pull-FE reviewen** (Subagent-Bau) · (c) **admin-Lücken** (Integrations-Tab-Placeholder, License-Detail-Modal) · (d) **Welle 3 Onboarding/Info-Center** (§1.2) · (e) **Echt-Schaltungs-Verifikation der Luke-bereinigten Module** (hr/security/automation/produktion/formulare visuell gegen echtes BE). **Luke-gebunden bleibt:** security-DSGVO-Echt-Schaltung, mails-IMAP, admin-Backend (Invite/RBAC/License/S3).
> **★ DOCKER-REALITÄT (aus #6, prüfen):** postgres = custom-Image (pgvector+pg_cron, Migr. jetzt **000243**) → muss gebaut werden. Bringup `--no-deps --no-build` (OOM!), Login `demo@local.test`/`Demo1234!`. Images sind nach Lukes Pull **stale** → betroffene Services neu bauen vor Echt-QA.
> **★ Git-Hygiene:** `deploy/docker/docker-compose.flags.yml` + `desktop/scripts/qa-dialer-callflow.mjs` bleiben untracked (lokal).
>
> ---
> _(Stand #6 folgt)_

# RESUME — Historie (Stand 2026-07-05, Session-Ende #6)

> **★★★★★ SESSION-ENDE 2026-07-05 #6 — main `b5e3ec55` (alles gepusht, CI+CD grün, Auto-Deploy lief). NEUES TERMINAL: HIER STARTEN (erst `git pull`).**
>
> **Diese Session (#6): Lukes 07-05-Quick-Win-Welle verifiziert + X-4-Settings-Rest fertig gebaut.**
> 1. **CRM Kontakt-CSV-Import echt-verifiziert — 2 mock-verdeckte Bugs gefunden+gefixt (`ea1748a5`):** (a) **Wire-Contract-Mismatch** — FE sendete Field-Mapping als JSON-Feld `field_mapping`, Gateway erwartet `map_<spalte>=<feld>`-Formularfelder → **jede** Zeile geskippt (`imported_count:0`). FE sendet jetzt `map_*`. (b) **Auto-Detection** `knownMappings` kannte `first_name`/`last_name` (Unterstrich) nicht → Export→Import-Round-Trip erkannte Namen nicht; ergänzt. Live end-to-end (Preview/Import/Export CSV+vCard/Visibility gegen echtes crm) verifiziert. **GAP→Luke:** company beim Import ignoriert + Export leer (Round-Trip).
> 2. **Video Incoming-Call/Decline (`44b23e77`) — Code-Review sauber, kein Bug.** Backend-Round-Trip komplett (`videoWSAdapter.NotifyCallDeclined`→EndCall+BroadcastCallEnded). Nicht live-2-Client getestet. **→Luke:** `caller_name`-Lookup im `call.incoming`-Broadcast (FE fällt auf ID zurück).
> 3. **Dunning-Mahnung E-Mail+PDF (`273f1b6b`) — live verifiziert** (create→send→PDF gegen biz/minio, SMTP graceful suppressed, Log bestätigt). **→Luke (Prod-Risiko):** Mail-Send ist **fatal** bei konfiguriertem Mailer + braucht Company-Settings → Tenant ohne Settings bekäme 500. Non-fatal machen erwägen.
> 4. **X-4 Settings-Rest FERTIG (`b5e3ec55`):** 6 Stores backend-persistiert nach crmPrefs-Muster + im zentralen `useHydrateModuleSettings` registriert. **user:** workPrefs, vertraegePrefs. **tenant** (read alle, write role-gated): financeTenant, wikiSettings, dashboardSettings, zeiterfassungSettings. Runtime-verifiziert (`scripts/qa-x4-rest-hydrator.mjs`: alle 6 hydratisieren Server-Werte nach localStorage-Default-Seed+Login). **→ X-4 self-doable KOMPLETT.** Offen X-4 = nur Welle-2-Reste: gemischte Store-Splits (dialer/automatisierung/berichte/mail/formulare-Prefs) + groß (payrollSettings/workSettings, Backend teils fehlt).
>
> **★ PUSH-MODE (Darien 07-05):** pro verifiziertem Modul auf main → **Auto-Deploy live**. Vor jedem Push CI-grün (eslint geänderte Dateien + scoped tsc + qa). backend-gaps.md 07-05-Block gepflegt.
> **★ DOCKER-REALITÄT:** postgres ist jetzt **custom-Image** (pgvector+pg_cron, `deploy/docker/postgres/Dockerfile`, Migr. 242) → muss **gebaut** werden (`--no-build` schlägt fehl mit „No such image"). Diese Session neu gebaut (waren stale nach Lukes Pull): postgres, crm, gateway, biz, migrate. Stack healthy: postgres/redis/auth/crm/gateway/minio/biz/work. Bringup: `--no-deps --no-build` (OOM!), Login `demo@local.test`/`Demo1234!`. PUT-Settings brauchen Idempotency-Key (setzt `authenticatedRequest` autom.).
> **★ NÄCHSTE UNIT (Vorschlag):** Welle 2 — **admin Demo-Tiefe** + **settings-Lücken (P2)** + **gemischte X-4 Store-Splits** + **Demo-Tiefe-Phasen** (notifications/formulare/dialer/video) · ODER **Welle 3 Onboarding/Info-Center** (reines FE, `§1.2`). Luke-gebunden bleibt: security-DSGVO-Echt-Schaltung, mails-IMAP, admin-Backend (Invite/RBAC/License/S3).
>
> ---
> _(Historie #5 folgt)_

# RESUME — Historie (Stand 2026-06-28, Session-Ende #5)

> **★★★★ SESSION-ENDE 2026-06-28 #5 — main `31330bb2` (alles gepusht, CI grün, Auto-Deploy lief). NEUES TERMINAL: HIER STARTEN (erst `git pull`).**
>
> **AUFTRAG NEUES TERMINAL: X-4-Settings-Rest fertigmachen** (Darien: „im neuen Terminal X-4-Rest, dann schauen wir weiter").
> **Rezept steht** (Referenz `stores/crmPrefs.ts` + `api/settings-persist.ts` + zentraler `hooks/useHydrateModuleSettings.ts`). Pro Store: persist behalten, `serverInitialized` + `initFromServer` (loadModuleSettings) + Write-Through (saveModuleSettings) in Settern, dann **in `useHydrateModuleSettings.ts` registrieren** (kein pro-Page-useEffect mehr — zentral in DeskEnvironment). Verify-Muster: `desktop/scripts/qa-x4-central-hydrator.mjs` (Server-Wert via API setzen mit Idempotency-Key → frischer Client hydratisiert).
> **X-4-Rest:** tenant-Settings (wikiSettings/dashboardSettings/zeiterfassungSettings/financeTenant — scope `'tenant'`, Schreiben nur Lead/Admin) · mittel (workPrefs/vertraegePrefs) · gemischt dialer/automatisierung/berichte/mail/formulare-Prefs = **Store-Split → Welle 2, NICHT jetzt**. Voller Plan: `.planning/welle1-finish-plan.md`.
>
> **Diese Session (#5) gebaut — Welle 1 self-doable praktisch durch (alles gepusht):**
> 1. **helpdesk echt-geschaltet** (`a1242d6d`) — Lukes tenant-Fix wirkt; 1 mock-Bug (KB/Routing-`undefined`-Crash → `unwrapList`); helpdesk-demo-Seed (6 Tickets, diverse UUIDs für Ticket-Nr).
> 2. **kommunikation-Inbox echt-geschaltet** (`9621ecc4`) — **3 mock-Bugs**: received_at `{seconds,nanos}`→ISO (ConversationList-Sort-Crash) · channel-Int→String · getMessage `{message}`-unwrap (Thread-Crash). inbox-demo-Seed (6 Messages). `inbox-client.ts` `normalizeMessage`.
> 3. **documents** (`50b8632a`) — K6 Share-List-URL `/shares`→`/shares/entity` (war 405) · K7 `normalizeShare` Enum-Int. READ regressionsfrei.
> 4. **zeiterfassung** (`faf4fe8c`) — live verifiziert (−16h15m), AbsenceCalendar-guard.
> 5. **security NUR verifiziert** — Backend echt (~25 Endpoints real), echt-schaltbar, NICHT geschaltet (Darien-Wunsch). Master-Plan „2/10" überholt.
> 6. **X-4: 8 personal-Prefs-Stores + zentraler Hydrator** (`07d31f3a`+`943ab109`) — crm/finance/team/dashboard/helpdesk/zeiterfassung/wiki/dokumente. Live verifiziert.
> 7. **DB-Migr. 227–234 nachgezogen** (hing auf 226 — migrate-Image war auch stale, neu gebaut).
>
> **★ DOCKER-REALITÄT (wichtig):** Die laufenden Images waren **vor Lukes Pull** (alter Code). Für Echt-Schaltung **Gateway + notification + auth + migrate neu bauen** nötig (`docker compose build <svc>` dann `up -d --no-deps --no-build <svc>`; Gateway mit `-f docker-compose.flags.yml`). Stack läuft (15 Services healthy). Bringup: `--no-deps --no-build` (OOM!), DB-User `kmuhub`/`kmuhub`, Login `demo@local.test`/`Demo1234!`. **PUT-Settings braucht Idempotency-Key-Header** (setzt `authenticatedRequest` automatisch; curl-Tests brauchen ihn manuell).
> **★ LUKE-TEXT rausgegeben** (Darien verschickt): Dank + Prod-Seed/Feature-Flags-Bitte + neue Backend-Gaps. Alles in `backend-gaps.md` (28.06.-Block): helpdesk-Namen/Kategorie/ticket_number · inbox-Thread-RPC+Canned · documents-naked-Shapes · mails-IMAP (Luke).
> **★ HETZNER-REVIEW-CAVEAT:** Code deployed ≠ auf Hetzner mit Daten sichtbar — Prod braucht Feature-Flags (`.env.production`) + Prod-Seed (Demo-Daten nur lokal). Luke-Schritt.
> **★ MASTER/BACKEND-PLAN aktualisiert** (§0+§6 / 28.06.-Block). ~18 Module echt-verkabelt, FE ~50–55 %.

---
_(Historie #4 folgt)_

>
> **⚠ NEUE REGEL (Luke): CI-grün beim Push → AUTO-DEPLOY auf Hetzner.** Jeder Push MUSS CI-grün sein. Desktop-CI (`ci-desktop.yml`) = `eslint src/` + `npx tsc --noEmit` (full, ~3,5min grün) + `vitest` + `npm run build`. Vor JEDEM Push lokal grün fahren (eslint auf geänderte Dateien reicht meist; full-tsc ist grün). [[feedback_hetzner_review_workflow]]
>
> **Diese Session gebaut (alles gepusht, CI-grün):**
> 1. **Welle 1 stark vorangetrieben — self-doable Modul-Echt-Schaltung praktisch DURCH.** Neu echt-verkabelt + live verifiziert: **documents** (READ, `5e6c14ef` — 5 Wire-Drifts FE-tolerant gefixt) · **calendar** (`a943937d`, Luke-verdrahtet, work-Service up) · **wiki** (`a6cf212b` — Crash-Bug `r.articles.map()` ohne Guard gefixt) · **automatisierung/berichte/kommunikation-chat** (`85f2d24b`, Services gebaut+verifiziert, keine FE-Fixes nötig). **~16 Module echt-verkabelt.**
> 2. **settings X-4 Referenz-Muster** (`cc5d930d`) — dokumente-Tenant-Settings store-direct → Backend (`/settings/dokumente/tenant`, Migr.138), `initFromServer`+Write-Through, end-to-end verifiziert (localStorage gewiped → hydratet aus Backend). **Rollout auf ~12 weitere Stores = self-doable Rest.**
> 3. **6 mock-verdeckte Bugs gefunden+gefixt** (FE-tolerant) + **2 Deploy-Blocker für Luke** (Feature-Flags X-7 + helpdesk-tenant).
>
> **★ FEATURE-FLAGS (X-7, deploy-kritisch):** helpdesk/wiki/berichte/formulare/vertraege/video/Branchen hängen im Gateway an `COSMI_MODULE_*_ENABLED` (default OFF). Lokal aktiviert via **`deploy/docker/docker-compose.flags.yml`** (untracked Override — beim Gateway-Start `-f docker-compose.yml -f docker-compose.flags.yml`). **Prod muss die Flags in `.env.production` setzen, sonst deployt der Auto-Deploy ohne diese Module.**
>
> **★ admin (Sub) FERTIG + GEMERGT** (`79020623`, 25.06.) — A-1…A-5 FE-mock-first: Benutzerverwaltung (Invite/Detail-Modal), RBAC-Matrix, Lizenz/Modul-Aktivierung, Branding-Tab, Settings-Overlay-Eintrag (~3346 Z., i18n ×4, QA `qa-admin.md`). Merge-Konflikt nur ITAdminHubTab (Branding-Dublette raus = Sub-Version genommen). **Full CI lokal grün** (eslint+tsc+`npm run build` 1m11s) vor Push. **Echt-Schaltung wartet auf Luke** (Auth-Invite/RBAC-Persist/License-Service/S3 — `backend-gaps.md` §Vorausschau). Offen: admin Demo-Tiefe-Schliff.
>
> **★ 2 LUKE-TEXTE rausgegeben** (Darien verschickt): (a) Welle-1-Blocker (helpdesk-tenant/security-DSGVO/mails-IMAP/inbox/documents-Wire/Feature-Flags) · (b) Vorausschau Welle 2/3 (admin-Stack/settings-OAuth/profil-S3/security + Onboarding=FE-only). Beide persistiert in `backend-gaps.md` (oben „🔭 Vorausschau" + „Echt-Schaltung-Befunde").
>
> **NÄCHSTE UNIT (Vorschlag, ~Welle 2+3 fast komplett self-doable):** Welle 2 ~80% ohne Luke (admin via Sub + **Demo-Tiefe-Phasen** notifications/formulare/dialer/video + Tiefe-Re-Checks kontakte/calendar/dokumente/zeiterfassung + **settings-Tabs P2** + **X-4-Rollout** ~12 Stores) · Welle 3 ~95–100% (Onboarding/Info-Center = reines FE). Luke-gebunden bleibt nur: settings-OAuth (P4) + security-DSGVO-Echt-Schaltung.
>
> **DOCKER-STACK läuft** (13 Services: postgres/redis/auth/crm/gateway/dialer/biz/minio/document/work/helpdesk/wiki + automation/berichte/chat). Gateway läuft mit Flags-Override. Falls weg: hochfahren + Gateway mit `-f docker-compose.flags.yml` recreaten + bei neuem Service Gateway-Restart (Service-Discovery). **Nur Main fasst Docker an (OOM).**
> **DEV-SERVER-QUIRK bleibt:** `electron-vite dev --mode localbackend` kam mehrmals flaky in MSW/Demo-Mode hoch (kein Login-Screen → QA-Timeout). Fix: sauber killen (`Get-NetTCPConnection -LocalPort 5173 | Stop-Process` + `Get-Process electron | Stop-Process`) + neu starten. **`--port`-Flag geht NICHT** (electron-vite CACError) — Sub nutzt 5173 oder env-gateten Port.
> **Git-Hygiene offen:** `deploy/docker/docker-compose.flags.yml` (untracked, LOKAL behalten — nicht committen) · `desktop/scripts/qa-dialer-callflow.mjs` (untracked, vorbestehend).
> **Master-Plan synchronisiert** (§0 Gesamtstand + §6 Bau-Status-Tabelle auf 25.06., ~16 echt-verkabelt). QA-Skripte: `qa-mock-exit-dokumente.mjs`, `qa-mock-exit-modules.mjs` (Multi-Route), `qa-settings-dokumente-persist.mjs`.

---
_(Historie #3 folgt)_

# RESUME — Historie (Stand 2026-06-23, Session-Ende #2)

> **★★ SESSION-ENDE 2026-06-24 #3 — main `156ca17a`, alles gepusht. NEUES TERMINAL: HIER STARTEN (erst `git pull`).**
>
> **Diese Session gebaut (alle live/API-verifiziert + gepusht):**
> 1. **dialer-Supervisor echt-geschaltet** — FE-Normalizer für protojson-Null-Omission (`api/dialer-normalize.ts`, `48b5daf9`) + **Backend-Bug** recent-calls-Query las nicht-existente `cc.contact_name` → crm-Join (`9dfcf89e`). Live-Screenshots `desktop/.qa-screenshots/dialer-supervisor/`.
> 2. **dashboard-Layout echt verifiziert** — war schon über `apiClient`↔gateway-nativ verkabelt, kein Code nötig. Roundtrip GET→PUT→GET live ok.
> 3. **zeiterfassung/HR echt-geschaltet** — **Backend-Bug** `correction_reason` (nullable) in `string` gescannt → 500 bei JEDEM echten Eintrag → `COALESCE` in 3 SELECTs (`b7242926`). API-verifiziert (entries 500→3). HR liegt auf **biz**-Service. FE-Screenshot offen (Dev-Server-Quirk, s.u.).
> 4. **security/DSGVO (Sub) gemergt** — `43fecf37` (S-1…S-5, review-reif): 11 Seiten crashfrei, GDPR-Flows Art.15/17/20, ein Hub `/admin/security` (10 Tabs), i18n ×4. Merge konfliktfrei, Build grün. Bericht `.planning/parallel-batch/qa-security.md`. **Offen: Art.30 RoPA** = eigener Folge-Batch.
> → **3 mock-verdeckte Backend-Bugs** gefunden+gefixt (alle brachen echte Daten still, alle deploy-relevant für Luke).
>
> **★ NEUER WORKFLOW (Darien, 24.06.):** Darien reviewt jetzt **hands-on auf der Hetzner-Cosmi-exe** (app.zentria.tech), parallel während gebaut wird. → **ALLES auf main pushen.** Prüf-Items in **`.planning/hetzner-review-checklist.md`** (lebende Datei pflegen). **2 offene Fragen dort:** (a) wer/wie deployt auf Hetzner (Auto-Deploy `cd.yml` nicht scharf → Push ≠ live), (b) Demo- oder Live-Mode der Hetzner-exe. Backend-Echt-Schaltungen sind nur lokal sichtbar (brauchen Deploy + Prod-Seed) → die weiter lokal verifizieren.
>
> **Docker-Stack läuft noch** (postgres/redis/auth/crm/gateway/dialer/biz/minio healthy) — **nur Main fasst Docker an** (OOM!). Falls weg: hochfahren (Abschnitt unten), Seeds: `backend/seeds/demo/{demo-seed,dialer-demo,hr-worktime-demo}.sql`. biz braucht minio (sonst Crash-Loop „gobd archive").
> **DEV-SERVER-QUIRK:** `electron-vite dev --mode localbackend` kam zuletzt in **MSW/Demo-Mode** hoch (Auto-Login „Stefan Vogel" statt „Demo Local") trotz Build-Flag → flakig. Beim Dialer-Lauf lief dieselbe Instanz korrekt localbackend. Check: Login-Screen = localbackend ok; Auto-Login Stefan-Vogel = Demo-Mode (neu starten). Kill: PowerShell `Get-NetTCPConnection -LocalPort 5173 | Stop-Process`.
> **NÄCHSTE UNIT (Vorschlag):** Welle 2 — **admin** (Benutzer/RBAC/Lizenz) oder **settings-Persistenz** (X-4, BE Migr 138 da); oder neues Sub für 2. Modul. Reste zeiterfassung: FE-Screenshot + `useAbsenceCalendar`-null-guard (`hr-hooks.ts:508` `select: d=>d.entries` ohne `?? []`).
> **Master-Plan:** ~47–50 % FE-Phasen, 10 echt-verkabelt, 11 FE-mock-fertig — `MASTER-PLAN.md` §6.
> **Git-Hygiene offen:** `desktop/package-lock.json` (npm-install-Churn, uncommitted), `desktop/scripts/qa-dialer-callflow.mjs` (untracked, vorbestehend) — committen oder verwerfen.


> **★ UPDATE 2026-06-24 #2 — dialer SUPERVISOR echt-geschaltet (Welle 1), live verifiziert + gepusht (`48b5daf9`).**
> Lukes neue Endpoints (`GET /dialer/supervisor` + `/dialer/contacts/{id}/calls`, `fb045f9f`) ans FE gehängt. **Zwei mock-verdeckte Bugs gefunden:**
> (1) **`recent_calls` immer leer** — Lukes Query las `cc.contact_name` (Spalte existiert nicht in `dialer_campaign_contacts`); SQL-Fehler wurde still als WARN geschluckt → Feed leer. Gefixt: crm-`contacts`+`companies`-Join in `GetRecentCallsForTenant` (`c10f8d2f`, im dialer-Service).
> (2) **protojson lässt Null-Werte weg** (`EmitUnpopulated:false`) → `totals.active_agents`/`on_call` fehlten, `recent_calls` fehlte ganz → FE wäre bei leerem Dialer abgestürzt (`recent_calls.length` auf undefined). FE-Normalizer `api/dialer-normalize.ts` füllt die Defaults (`48b5daf9`), eingehängt in `dialer-client.ts`.
> **Verifikation:** Docker-Stack hoch (postgres/redis/auth/crm/gateway/dialer healthy), Dialer-Demo-Seed `backend/seeds/demo/dialer-demo.sql` (Kampagne + 2 Agents + 3 Outcomes + 5 Call-Sessions HEUTE → 5 Calls/2 Termine). Live gegen :8080: Supervisor zeigt KPIs (Aktive 0/Im Gespräch 0/Anrufe 5/Termine 2 — die 0 rendern statt undefined = Normalizer wirkt), Team mit calls_today, Letzte Anrufe voll. Screenshots `desktop/.qa-screenshots/dialer-supervisor/`, Skript `qa-dialer-supervisor-localbackend.mjs`.
> **LEHRE (Windows):** `curl | python -m json.tool` zeigt UTF-8-Umlaute fälschlich als Mojibake (`Ã¼`) — Python 3.14 liest stdin als cp1252, NICHT UTF-8. Echte Bytes mit `xxd` prüfen (`C3 BC` = sauberes ü). Es gab KEINEN Encoding-Bug; die Namen rendern sauber.
> **Gebaut-aber-nicht-meine-Lane:** `npm run build` war rot wegen `@livekit/track-processors` — nur stale node_modules, `npm install` fixt es (Dep ist in package.json). Danach Build grün.
> **PARALLEL:** Sub-Terminal baut `security`/DSGVO auf Branch `parallel/security` (Paket `.planning/parallel-batch/sub-security.md`, S-0 done, S-1…S-5 freigegeben). Main merged den Branch am Ende.
> **Docker läuft noch** (nur Main fasst Docker an). Offen für dialer: Contact-Calls-Detail im UI screenshotten (ContactDetailModal), Supervisor-Leer-Zustand sauber live testen (Pass B nutzte Cache).
>
> **DANACH verifiziert (selbe Session):**
> - **dashboard-Layout = echt, KEIN Code-Change nötig.** Store nutzt schon den echten `apiClient` (`/api/v1/dashboard/layout`, gateway-nativ, `response.JSON` — keine protojson-Falle). Roundtrip GET→PUT→GET live bestätigt (`{layout, active_widgets, is_custom, updated_at}`, persistiert). FE ruft `initFromServer`+`ensureDefaults` on mount. Abhaken in MASTER-PLAN.
> - **zeiterfassung/HR ENTBLOCKT (Backend läuft jetzt):** HR liegt auf dem **biz**-Service (`HRRoutes.ServiceName()="biz"`, HR+Finance teilen das biz-Binary). biz crashte (`failed to connect to minio for gobd archive`) → **minio + createbucket hochgefahren** → biz healthy → Gateway-Restart → HR-Endpoints antworten. Shapes geprobt: `/hr/time/balance` flach (clean), `/hr/time/status` flach, `/hr/time/entries` = `{entries:null,total:0}` (**`entries` ist `null` nicht `[]` bei leer → jede Konsum-Hook muss `?? []`**). `normalizeWireTimestamps` (für `{seconds,nanos}`) ist in `hr-client.ts` schon drin (Luke-Welle). **DONE (selbe Session):** HR-Demo geseedet (`hr-worktime-demo.sql`, 3 Einträge Mo–Mi). **Echter Backend-Bug gefunden+gefixt** (`b7242926`): `GetByID/List/GetActiveShift` scannten `correction_reason` (nullable, bei normalen Einträgen NULL) in einen `string` → `can't scan NULL into *string` → **Einträge-Liste warf 500 bei JEDEM echten Eintrag**; Fix = `COALESCE(correction_reason,'')` in den 3 SELECTs. API-verifiziert: `/hr/time/entries` liefert jetzt 3 Einträge (vorher 500), `/balance` korrekt -975. **FE-Screenshot ausstehend** — Dev-Server-Quirk: `electron-vite dev --mode localbackend` kam in MSW/Demo-Mode hoch (Stefan-Vogel statt Demo-Local), trotz korrektem Build-Flag; beim Dialer-Lauf lief dieselbe Instanz korrekt localbackend → flakiger Start, nächste Session sauber neu starten + verifizieren. Offen außerdem: `useAbsenceCalendar`-Hook (hr-hooks.ts:508) `select: (data)=>data.entries` ohne `?? []` → Crash-Risiko bei leerem `{entries:null}`. Stack-Stand: postgres/redis/auth/crm/gateway/dialer/biz/minio healthy.


> **★ UPDATE 2026-06-24 — B-12 DONE, Buchhaltung KOMPLETT echt (gepusht `4712857a`).**
> (1) Betrag-Fix `protoTaxBreakdown()`-Fallback in `biz_grpc.go` — `toProto{Invoice,Quote,CreditNote}` lesen jetzt das `tax_breakdown`-JSONB **oder** die Einzelspalten `subtotal/total_tax/gross_total` (der Seed füllt nur die Spalten → vorher 0,00 €). (2) Zweiter mock-verdeckter Bug gefixt: Dunning-Pfade in `finance-client.ts` **und** `mocks/handlers/finance.ts` Plural→Singular (`/finance/dunning` + `/dunning/config`) — das Gateway routet Singular, Plural gab 404 und der Mahnungen-Tab degradierte still zum Empty-State. Alle finanzen-Tabs live verifiziert: `desktop/.qa-screenshots/b12-finanzen/`. QA-Skripte: `qa-b12-finanzen-amounts.mjs`, `qa-b12-dunning-settle.mjs`.
> **Recovery-Lehre:** `docker compose up` OHNE `--no-deps` zieht den ganzen gateway-Dependency-Graph (alle 23 µSvc) rein und baut sie → WSL2-vmmem-Explosion (16 GB Maschine, RAM auf 1,4 GB) → Daemon-Hänger. Immer `up -d --no-deps <nur-was-man-braucht>`. Recovery: Docker Desktop killen + `wsl --shutdown` (gibt vmmem frei) + neu starten.

> **★ MOCK-EXIT — kontakte ist KOMPLETT echt (Referenz-Modul).** READ + voller CRUD (Create/Update/Delete) durch die echte UI gegen das lokale Backend, live verifiziert (Screenshots `desktop/.qa-screenshots/crud-*.png`). Casing-Entscheidung getroffen: **Option C** (per-Modul `dual()`-Adapter, kein globaler Transform — FE ist gemischt-casing). Vollständiger Bericht + Backend-Handover + camelCase-Risiko-Set für die nächsten Module: **`.planning/kontakte-mock-exit-DONE.md`**.
>
> **Diese Session neu:** `api/casing.ts` (`dual()`-Helper), `mocks/demo-mode-flag.ts` (Leaf-Flag), kontakte-Adapter mode-branched + position↔jobTitle, useContacts PATCH→PUT, Mock-Handler PATCH→PUT, Demo-User→admin (Seed idempotent). 3 weitere Mock-verdeckte Bugs gefunden+gefixt (PUT-Methode, position-Feld, custom_fields-Array).
>
> **Voriger Durchstich (Session #1):** Login + Kontakte-Liste echt, 2 Bugs gefixt (CORS-Idempotency-Key `d4a9c1a4`, Contact-Adapter snake_case `3979b142`).

## Was diese Session fertig wurde (gepusht, `043cb372`)
- **Lokales Backend läuft** via Docker (`deploy/docker/docker-compose.yml`): **postgres + redis + auth + crm + gateway** (Minimal-Subset; voller 24-Service-Stack crasht die Maschine → nur bauen, was man braucht). Gateway auf `:8080`, Migrationen bis **000226**.
- **Demo-Seeds** (`backend/seeds/demo/demo-seed.sql`, idempotent, Tenant `…0001`): 8 companies, 12 contacts, 8 deals, 3 projects, 10 tasks. **Finance-Block auskommentiert** (line_items ist separate `finance_line_items`-Tabelle → noch fixen).
- **Mock-Exit verifiziert end-to-end:** Login (`demo@local.test` / `Demo1234!`) → Kontakte mit echten Namen/Firmen/Avataren. QA-Skripte: `desktop/scripts/qa-mock-exit-kontakte.mjs` (Token-Inject) + `qa-mock-exit-login.mjs` (echter Login-Flow).
- **2 echte Bugs gefixt (Mocks hatten sie verdeckt):**
  - `fix(gateway)` `d4a9c1a4` — **CORS allow-headers** um `Idempotency-Key` ergänzt. HardMode verlangt den Header bei jeder Mutation, CORS verbot ihn → jede Mutation (Login/Create/Update) aus jedem Browser-Client blockiert. **Betrifft Luke/Prod.**
  - `fix(kontakte)` `3979b142` — Contact-**Adapter liest snake_case** (Gateway liefert `first_name`, OpenAPI-Typen sind camelCase = **Spec-Drift X-3**). Sonst Namen/Firma leer. **Muster betrifft JEDES Modul beim Mock-Exit.**
- **Tooling** `043cb372` — `RENDERER_VITE_DEV_BYPASS_AUTH=false` erzwingt echten Login im Dev-Build (`App.tsx`); `.planning/mock-exit-readiness-matrix.md` (Modul × Backend × Wire-Shape × Auth × RLS); `SESSION-RUNBOOK.md` Markt-Recherche als Pflicht-Schritt.
- **NICHT angefasst:** Login-Animation/`AuthLayout` (läuft auf main+Hetzner korrekt; das „falsche" C-Icon war nur ein Dev-Artefakt durch wiederholte Reloads → statischer Fallback statt Animation).

## Lokal wieder hochfahren (neues Terminal)
```bash
# 1. Docker-Backend (läuft evtl. noch — prüfen):
docker ps   # postgres/redis/auth/crm/gateway healthy?
# falls weg: cd "C:/Users/darie/Documents/KMU Hub"
docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/.env up -d --no-deps postgres redis auth crm gateway
# (.env liegt unter deploy/docker/.env — gitignored; Werte: deploy/docker/README.md + MIGRATION_DATABASE_URL)
# Seed (falls DB frisch): docker exec -i docker-postgres-1 psql -U kmuhub -d kmuhub --single-transaction < backend/seeds/demo/demo-seed.sql

# 2. FE gegen echtes Backend (Mode localbackend = DEMO_MODE=false + :8080 + echter Login):
cd desktop && npx electron-vite dev --mode localbackend
# Login: demo@local.test / Demo1234!  (Tenant …0001, sieht Seed-Daten)
# Hinweis: nur kontakte/firmen/deals live (crm); andere Module 503 (Service nicht gebaut)
```

## Was als Nächstes (Reihenfolge nach Hebel)
1. **~~OpenAPI-Casing~~ GELÖST** — Entscheidung Option C (per-Modul `dual()`). Globaler Transform verworfen (FE gemischt-casing, würde Tausende snake-Leser brechen). Casing-Risiko-Set + Pattern in `kontakte-mock-exit-DONE.md`.
2. **Nächstes Modul nach kontakte-Pattern echt schalten** — Reihenfolge nach Risiko-Set: crm/companies → crm/deals+pipeline-stages (DealInfo-Casing!) → work. Pro Modul: `dual()`-Adapter falls OpenAPI-getippt, Methode/Wire-Shape/Idempotency/RBAC gegen echtes Backend prüfen (nicht nur Mock).
3. **work + biz dazuholen** → Aufgaben/Projekte/Finanzen echt (`docker compose build work biz` + `up -d --no-deps`).
4. **Finance-Seed fixen** (line_items → `finance_line_items`-Tabelle) → finanzen-Demo nicht leer.
5. **RLS-scharf testen:** `DATABASE_URL` auf `kmuhub_app:app_dev` (statt Superuser) → wie Prod. Migration 000121, einmalig `ALTER ROLE kmuhub_app WITH PASSWORD 'app_dev'`.
6. **Luke-Handover offen** (siehe `kontakte-mock-exit-DONE.md`): contact-Schema zu dünn (9 Extra-Felder), OpenAPI-Spec-Drift contacts (PATCH→PUT, title→position, custom_fields-Array), Timeline-Endpoint hängt.

## Parallel: regulärer Bau-Track (MASTER-PLAN.md)
Der Mock-Exit ist Welle 1 (Echt-Schaltung) in Aktion. `MASTER-PLAN.md` bleibt die SSOT für die übrigen Wellen. SESSION-RUNBOOK-Zyklus gilt weiter.
