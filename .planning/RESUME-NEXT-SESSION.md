# ▶ HIER STARTEN

## 👥 DELEGATION AN NICO (Stand 2026-06-08) — läuft parallel
Nico (sie/ihr, kein erfahrener Coder, hat Claude Code + VS Code) übernimmt den Block **Content & Self-Service** (wiki/formulare/berichte/notifications). Komplettes Paket: **`.planning/nico-block/`** (README → RUNBOOK → WORKFLOW → KICKOFF → phase-01/02 → BACKLOG). Sie startet mit **Pilot phase-01 (notifications Ruhezeiten/DND)** + **phase-02 (berichte Sparkline)**.
- **Build-+-Verify-Standard ist jetzt im Repo-`CLAUDE.md` verankert** (gilt auto für jede Session/Phase, alle).
- **▶ Review-Auftrag Haupt-Team:** Wenn Nico „fertig" meldet (Commit-SHA + Verify-Checkliste + Screenshots) → gegen die Definition-of-Done der jeweiligen Spec prüfen. Pilot grün → Block freigeben + Backlog-Specs schreiben. Hohe Fehlerrate → Scope enger fassen.
- **Realismus:** 3–5 Phasen/Tag/Person, nicht 10. Engpass = Review-Gate.

## ⏯ AKTIVER BATCH (Stand 2026-06-08, Pause → weiter 2026-06-09) — „team-Lohn + Kommunikation-Merge"
Feinplan: `.planning/batch-kommunikation-feinplan.md`. Fünfer-Batch, direct-to-main, origin synchron (HEAD `42b2752`).

**▶ MORGEN HIER STARTEN: Phase 4 (Posteingang scharfschalten).** ZUERST 15 Min echten Inbox-Backend-Stand verifizieren (proto/gateway/service-impl) — Lektion aus Phase 3: Notizen waren veraltet, 3/4 Items waren backend-fertig → Phase halbierte sich. Schätzung Darien gegeben: Phase 4 ~1 Session, Phase 5 ~1,5–2 Sessions, gesamt ~2,5–3 (kann auf ~2 schrumpfen wenn Inbox-Backend so komplett ist wie der Feinplan sagt). Verifikations-Setup unten (`tsconfig.pNcheck.json` + `scripts/qa-*.mjs` gegen `npm run dev` :5173).
- ✅ **Phase 1** Lohn-Stammdaten am Mitarbeiterprofil (`520b77c`)
- ✅ **Phase 2** Modul-Merge chat+kommunikation → ein Modul, Umschalter Team|Posteingang (`b242dd0`)
- ✅ **Phase 3** Team-Chat scharf — KOMPLETT. mark-read/Member-Panel/Inline-Edit/Raw-Key-Fix (`f3d2364`) + Volltextsuche (`8b6a3a0`) + **Join/Leave-UI** (`3c05079`, echt) + **Mentions-Inbox** (`dcfa12b`, echt — Backend war doch da) + **File-Upload** (`7c0c7ff`, echt — `POST /api/v1/files/upload` mit `message_id`) + **Reactions** (`2ac5893`, session-persistenter Store, Backend wirklich nicht verdrahtet → backend-gaps). Befund: nur Reactions ist echter Backend-Gap; Mentions+File-Upload waren backend-fertig.
- ⏳ **Phase 4** Posteingang scharf (Snooze/Claim/Assign/Bulk/Routing/Team-Inbox verdrahten + Demo-Handler; Status/Threading/Tags mock-first). Demo-Handler fehlen für snooze/claim/teams/rules.
- ⏳ **Phase 5** Synergie (interne Notizen+@Kollegen im Kunden-Thread, Collision), Call-Bridge ins video-Modul, Slash-Command-Mock-Shell, KommunikationSettingsPanel tenant-Sektionen (Kanäle/Routing/Team-Inbox/Canned/Retention), backend-gaps final.
- **Entscheidungen Darien:** Merge-UI = Umschalter · Audio/Video = echte video-Bridge · Bots/Slash = Mock-Shell+Doku · **Demo-Handler bauen wo nötig** (Mock-first verdrahtbar) · Phasen-Position bei Auflistung mit angeben (~140-Phasen-Plan).
- **Verifikations-Setup:** gescopter `tsconfig.pNcheck.json` (extends web, inkl. global.d.ts/vite-env.d.ts/i18next.d.ts + geänderte Dateien) → `node_modules/.bin/tsc --noEmit -p ...` (NIE busy-wait-Loop danebenlaufen lassen — bremst tsc aus!). QA: `scripts/qa-*.mjs` gegen `npm run dev` :5173.

---


> Autonomer Bau-Lauf. Ziel: alle ~25 Module „settings-komplett" durcharbeiten (gründlich > schnell). Reihenfolge in `module-phase-plans.md` (Entscheidungs-Log oben).

## Stand (2026-06-07, autonomer Lauf)
Branch `main`, direct-to-main, origin synchron. Commits dieser Session:
- `1f9e487` Settings-Fundament (scope personal/tenant, Modul-Leiter, Lock)
- `348a057` Modul-Einstellungs-Fenster (Overlay statt Route, MODULE + COSMI, aktives Modul markiert)
- `abef77d` CRM-Panel B1 (Pipeline + Eigene Felder)
- `2c27163` CRM Tags + Persönlich/Für-alle-Bereiche + Button-Umbau (Admin-Btn raus, → „Modul-Einstellungen")

## ✅ Fertig
- **Settings-Fundament** + **Modul-Einstellungs-Fenster** (App-Shell). Öffnet links unten „Modul-Einstellungen" als Overlay. Links Modul-Liste (Gruppe MODULE + COSMI/Allgemein), rechts Inhalt, aktives Modul vorausgewählt. Persönliche Settings bleiben oben rechts (Profil-Menü → /settings).
- **Kontakte Phase 7 = KOMPLETT.** CRM-Eintrag im Fenster mit zwei Bereichen: **Persönlich** (Standard-Ansicht/Dichte/Avatare, real verdrahtet) + **Für alle** (Pipeline-Editor, Eigene Felder, Tag-Verwaltung).

## ⚙ Architektur-Bausteine (für JEDES weitere Modul nutzen)
- `components/shared/ModuleSettingsShell` — Sektionen mit `scope: 'personal'|'tenant'`, rendert automatisch Bereich-Header „Persönlich"/„Für alle" + Lock für Nicht-Leiter.
- `modules/settings/module-settings-registry.tsx` — hier jeden Modul-Eintrag registrieren (Gruppe `module`, `navMatch`, `roles`).
- `components/shared/ColorSwatchPicker` + `lib/swatch-colors` · `stores/moduleLeads` + `hooks/useModuleSettings` (useIsModuleLead) · `stores/<modul>Prefs.ts` für persönliche Prefs.
- **i18n: i18next-ICU → einfache Klammern `{var}`** (NICHT `{{var}}`). Audit: `scripts/i18n-audit.mjs`. QA: `scripts/qa-*.mjs` (Playwright gegen :5173, `npm run dev`).

## ⭐ NÄCHSTE SESSION STARTET HIER (neues Terminal, 2026-06-07)

**NEUER STANDARD-WORKFLOW: Fünfer-Phasen-Batches** (Memory `feedback_five_phase_batches`). Zyklus pro Batch:
1. Nächste **5 Phasen** identifizieren (aus `module-phase-plans.md`, >140 Phasen gesamt)
2. Gründliche **Recherche + Ist-Abgleich**
3. **Besprechen** mit Darien (VOR dem Bauen)
4. Alle **5 Phasen ausführen** (bauen + QA + commit/push)
5. Gemeinsames **Review** + Rest anpassen

**Konkret Schritt 1 für den nächsten Batch — Phase 1 steht schon fest:**
- **Phase 1 = Lohn-Stammdaten am Mitarbeiterprofil** (Spec fertig: `.planning/team-lohn-stammdaten-spec.md`, recherchiert, NICHT gebaut). Mock-first: `stores/payrollMasterData.ts` + `lib/payroll-enums.ts` + `EmployeePayrollData.tsx` (hr_only) im MemberDetailPanel, speist PayrollPrepPanel. Lohnvorbereitung bleibt in team (entschieden).
- **Phasen 2–5:** aus `module-phase-plans.md` identifizieren (Modul **kommunikation** ist als Nächstes dran; mails davor zurückgestellt). Beim Start: Phasenplan lesen, die 5 zusammenstellen, recherchieren, dann mit Darien besprechen.

→ Beim Wiedereinstieg: erst `git fetch` + Stand prüfen, dann diesen Batch-Zyklus starten.

## ▶ NÄCHSTE SCHRITTE (in Reihenfolge)
Pro Modul: Features + **Moduleinstellungen mitbauen** (Persönlich + Für-alle) + Screenshot-QA @ Breiten + i18n ×4 + commit/push.
1. ✅ **Kontakte Phase 8 = KOMPLETT** (autonomer Lauf 2026-06-07). Hybrid-Architektur (Darien): Tab „Beratungsprotokolle" im Detail-Modal → Historie-Liste; „Neues Protokoll" öffnet Vollbild-Editor-Route `/kontakte/protokoll/:contactId/:protocolId` (8 Abschnitte nach Spec, finalize=immutable, 10-J.-Retention-Hinweis). „Empfohlen von" = Kontakt-Picker (`ReferredByField`) + Empfehler-Report in Auswertungen. Mandanten-Segmente A/B/C regelbasiert (`lib/segments.ts`, Schwellen in CRM-Settings „Für alle", Badge im Header). Neue Dateien: `lib/advisory.ts`+`lib/segments.ts`, `stores/advisoryProtocols.ts`+`referrals.ts`+`segmentSettings.ts`, `modules/kontakte/advisory/*`, `ReferredByField.tsx`, `SegmentSettings.tsx`. i18n: 162 Keys ×4. QA: `scripts/qa-advisory.mjs`/`qa-referral.mjs`/`qa-segment.mjs` (alle grün, keine Raw-Keys, keine pageErrors). ⚠ **tsc auf diesem Projekt ist brutal langsam (~20 Min cold)** — Verifikation primär via Vite+Playwright-QA, tsc 1× am Schluss.
2. ✅ **Buchhaltung (finanzen) = Settings-Konsolidierung KOMPLETT** (2026-06-07, Commit `585eb89`). Modul war schon ~90% gebaut (Faktura-Kette/Mahnwesen/Belegkette/Banking/DATEV-Bexio-BMD-Export echt). Gemacht: **FinanceSettingsPanel** (ModuleSettingsShell) konsolidiert die früheren In-Page-Admin-Tabs → Persönlich (Start-Ansicht) + Für-alle (Stammdaten, Rechnungen/Steuer/Mahnwesen, Integrationen) via `embedded`-Prop; In-Page-Admin-Tabs aus FinanzenPage raus; **Ausgaben/Transaktionen → mock-first TanStack** (`useFinanceLedger`, Store bleibt Backing wegen totem buchhaltung-Ordner); Label → „Buchhaltung"; IntegrationCard-Raw-Key-Bug gefixt. **Wir nennen das Modul „Buchhaltung"** (Code-Ordner `finanzen/`). E-Rechnung/DATEV/Bexio scharf = Luke-Backend (Launch-Blocker). `modules/buchhaltung/` (dead) weiter NICHT anfassen. ⚠ FinanzenPage hat vorbestehende `total_gross`-Baseline-Typefehler (nicht von uns).
3. ✅ **team = KOMPLETT** (2026-06-07, Commit `d1680f1`). Modul war schon reich (Mitarbeiter/Anträge TanStack). Gemacht: **TeamSettingsPanel** (Persönlich: Start-Tab+Ansicht · Für-alle: HR-Config embedded + **PayrollSettings** = DATEV-Verbindung/Lohnarten-/Abwesenheits-Mapping/Abrechnungsgruppen); In-Page-Einstellungen+Integrationen-Tabs konsolidiert; neuer **Lohnvorbereitung-Tab** (`PayrollPrepPanel`: monatl. Lohnlauf Zeitraum→Änderungsliste→freigeben→Export→Historie, Symbiose wie Buchhaltung) + Lohnabzugs-Vorschau extrahiert; HRIntegrationPanel gelöscht. Recherche-Spec `team-datev-lohn-spec.md`. Mock-first; DATEV-Datei-Gen + payroll_runs = Luke (`backend-gaps.md`). ⚠ **Demo-Daten:** team-Modul zeigt überall „Unbekannt" (userName leer im Demo-Mode, modulweit/vorbestehend). QA-Lock-Bug gefunden+gefixt (Zustand-Selektor auf Funktion statt State).
4. **kommunikation** ← NÄCHSTES (mails davor zurückgestellt, Backend-Arch mit Luke). Dann work → … → security ans Ende. → **kommunikation** → **work** → … (Reihenfolge `module-phase-plans.md`). **mails zurückgestellt** (Backend-Arch mit Luke). **security ans Ende** (DSGVO mit Luke).

## Offene Domänen-Fragen (Darien, wenn Zeit)
- Kontakte P8: welche Felder braucht ein Beratungsprotokoll fachlich? (DSGVO/Doku-Pflicht Finanzberatung) — aktuell Claude-Defaults.

## Arbeitsregeln
- Keine sichtbaren Scrollbars · Zurück-Buttons in Detail-Views · keine ASCII-Umlaute · keine Emojis in UI · wiederverwendbar in `shared/` bauen.
- Pro Bau-Einheit: typecheck (`npx tsc --noEmit -p tsconfig.web.json`) + lint + Screenshot-QA (Raw-Keys @ voller Breite!) + commit/push.
- Backend-Bedarf → `.planning/backend-gaps.md` für Luke.
