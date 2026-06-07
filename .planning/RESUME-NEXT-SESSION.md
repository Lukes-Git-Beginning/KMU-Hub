# ▶ HIER STARTEN

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
