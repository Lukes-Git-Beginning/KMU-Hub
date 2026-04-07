# i18n Migration — Komplette Roadmap

## Context
i18n Migration (react-intl → i18next) in Cosmi Desktop. Ziel: Alle ~440 .tsx/.ts-Dateien instrumentieren, damit die App vollständig DE/EN/FR/IT übersetzbar ist. Agents dürfen max ~5-7 Dateien pro Task bearbeiten, um unter 80k Tokens zu bleiben.

## ✅ MIGRATION ABGESCHLOSSEN (2026-04-08)

**Ergebnis:** 7.221 Keys × 4 Sprachen (DE/EN/FR/IT), strict TypeScript types, 21 Commits.

## Commit-Historie

### Committed (18 Commits auf main):
- `1b7ee22` — i18next Infrastruktur + react-intl entfernt (20 Dateien)
- `cf825a7` — Dashboard, CRM instrumentiert (78 Dateien)
- `10cffa5` — Work, Profil, Settings, Kommunikation, Finanzen, Team (107 Dateien)
- `88d9f1f` — Schritt 1B: 4 additions JSONs extrahiert + Loader aktualisiert
- `549fb82` — Kontakte (partial) + Chat (partial)
- `304ccbe` — kontakte.json Extraktion (148 Keys)
- `e517ec2` — Chat (complete) + Wiki (66 Keys)
- `60119f8` — Automatisierung (156 Keys) + Dokumente partial (78 Keys)
- `99a97bd` — Dokumente complete (163 Keys) + Admin (270 Keys)
- `fafc4a8` — Meetings (201 Keys) + Mails (115 Keys)
- `99b36e0` — Helpdesk (120), Buchhaltung (158), Kalender (33), Rapporte (130), Notifications (30)
- `44aa791` — i18n Loader mit allen Modul-Additions aktualisiert
- `50c1b15` — Component-Verzeichnisse instrumentiert (412 Keys, 9 JSONs)
- `c2ab5c7` — Wave 2+3: Security, Features, Config, API Hooks, Remaining Pages (315 Keys)
- `1a8a218` — Settings Tabs instrumentiert (12 Dateien, +404 Keys)
- `3779465` — Integration Panels instrumentiert (11 Dateien, +261 Keys)
- `7175e44` — Module Pages: Finanzen, Security, Team, Kalender, Mails (+196 Keys)
- `052a98e` — API Hooks, Types, Remaining Components (+108 Keys)

### Modul-Status Phase 3

| Status | Modul | JSON | Keys |
|--------|-------|------|------|
| ✅ | auth | — (in de.json) | — |
| ✅ | security | — (in de.json) | — |
| ✅ | dashboard | dashboard.json | 187 |
| ✅ | crm | crm.json | 302 |
| ✅ | work | work.json | 323 |
| ✅ | profil | profil.json | 275 |
| ✅ | settings | settings.json | 233 |
| ✅ | kommunikation | kommunikation.json | 167 |
| ✅ | finanzen | finanzen.json | 244 |
| ✅ | team | team.json | 233 |
| ✅ | kontakte | kontakte.json + kontakte-2.json | 148 + 64 |
| ✅ | chat | chat.json | ~46 |
| ✅ | wiki | wiki.json | 66 |
| ✅ | automatisierung | automatisierung.json | 156 |
| ✅ | dokumente | dokumente.json | 163 |
| ✅ | admin | admin.json | 270 |
| ✅ | meetings | meetings.json | 201 |
| ✅ | mails | mails.json | 115 |
| ✅ | helpdesk | helpdesk.json | 120 |
| ✅ | buchhaltung | buchhaltung.json | 158 |
| ✅ | kalender | kalender.json | 33 |
| ✅ | rapporte | rapporte.json | 130 |
| ✅ | notifications | notifications.json | 30 |
| ✅ | Components | 9 JSONs | 412 |
| ✅ | fuhrpark | fuhrpark.json | 169 |
| ✅ | einkauf | einkauf.json | 139 |
| ✅ | inventar | inventar.json | — |
| ✅ | vermietung | vermietung.json | — |
| ✅ | vertraege | vertraege.json | — |
| ✅ | produktion | produktion.json | — |
| ✅ | formulare | formulare.json | — |
| ✅ | schichten | schichten.json | — |
| ✅ | berichte | berichte.json | — |
| ✅ | video | video.json | — |

### i18n.ts Loader
Alle 41 additions-JSONs (32 Module + 9 Components) importiert und in mergedDE gemergt.

### de.json Keys gesamt: 7.221 (nach Merge von 46 additions + base)

---

## ~~Schritt 1: JSON-Extraktion + Lücken füllen~~ ✅ ERLEDIGT

## ~~Schritt 2: Verbleibende Module (13 Module)~~ ✅ ERLEDIGT

## ~~Schritt 2.5: Loader Update~~ ✅ ERLEDIGT (Commit `44aa791`)

## ~~Schritt 3: Component-Verzeichnisse~~ ✅ ERLEDIGT (Commit `50c1b15`)

9 Agents in 3 Batches (je 3 parallel), 46 Dateien instrumentiert, 412 Keys in 9 JSONs:
- `components-layout-core.json` (13 Keys) — ModuleShell, OfflineBanner, Sidebar*
- `components-layout-nav.json` (35 Keys) — nav-items, DockLayout, TopNav*
- `components-header-widgets.json` (30 Keys) — ClockIn, ConnectionStatus, header-widgets/*
- `components-header-menus.json` (35 Keys) — ProfileMenu, ProfileSwitcher, LanguageSwitcher, SearchBar
- `components-header-time.json` (62 Keys) — DailyPlanner, HeaderClock, TimeTracker
- `components-shared-search-editor.json` (45 Keys) — GlobalSearch/*, RichTextEditor/*
- `components-shared-misc.json` (43 Keys) — ConfirmDialog, DetailPanel, PasswordExpiry, Tour, Layout/PaletteSwitcher
- `components-widgets.json` (75 Keys) — WidgetWrapper, HelpWidget, WidgetContainer, WidgetRegistry
- `components-chat-desk-onboarding.json` (74 Keys) — Chat/*, Desk/*, OnboardingWizard, dev/ProfileSwitcher

---

## ~~Schritt 3.5: Straggler-Module (Wave 1)~~ ✅ ERLEDIGT

3 Agents parallel, 14 Dateien instrumentiert, 10 neue JSONs:
- `fuhrpark.json` (169 Keys) — FuhrparkPage, SchadensmeldungDialog
- `einkauf.json` (139 Keys) — EinkaufPage
- `inventar.json` — InventarPage
- `vermietung.json` — VermietungPage, ZustandsprotokollDialog
- `vertraege.json` — VertraegePage, ESignaturDialog
- `produktion.json` — ProduktionPage, MaschinenbelegungChart
- `formulare.json` — FormularePage
- `schichten.json` — SchichtenPage
- `berichte.json` — BerichtePage
- `video.json` — VideoPage

### ~~Verbleibend (Schritt 3.75)~~ ✅ ERLEDIGT (2026-04-07)

4 Waves, 11 Agents, 42 Dateien, +969 Keys:
- Wave 1: Settings Tabs (12 Dateien, +404 Keys)
- Wave 2: Integration Panels (11 Dateien, +261 Keys)
- Wave 3: Module Pages — Finanzen, Security, Team, Kalender, Mails (13 Dateien, +196 Keys)
- Wave 4: API Hooks, Types, Components (6 Dateien, +108 Keys)

Mock-Daten in Stores bewusst NICHT instrumentiert (werden durch Backend-Daten ersetzt).

---

## ~~Schritt 4: Merge & Loader~~ ✅ ERLEDIGT

- 46 additions/*.json (6.972 Keys) in de.json gemergt → **7.221 Keys total**
- i18n.ts von 153 auf 44 Zeilen vereinfacht (keine separaten Imports mehr)
- additions/ Verzeichnis entfernt
- `tsc --noEmit` + `npm run build` sauber
- Commit: `feat(i18n): merge additions into de.json and simplify loader`

---

## ~~Schritt 5: Phase 4 — Übersetzungen (EN/FR/IT)~~ ✅ ERLEDIGT

- Alle 7.221 Keys in EN, FR, IT übersetzt (3 Chunks × 3 Sprachen parallel)
- Volle Key-Parität: DE = EN = FR = IT = 7.221 Keys
- Production Build verifiziert
- Commit: `feat(i18n): add EN/FR/IT translations for all 7,221 keys`

---

## ~~Schritt 6: Phase 5 — Cleanup~~ ✅ ERLEDIGT

- i18next.d.ts auf strikte Typen umgestellt (de.json als resources Source of Truth)
- Unused-Key-Analyse: 119 Kandidaten, alle False Positives (dynamische Keys, ICU Plurals)
- `tsc --noEmit` + `npm run build` sauber
- Commit: `chore(i18n): enable strict type-safe translation keys`

---

## Agent-Prompt-Template

Jeder Agent bekommt dieses Template, ausgefüllt mit seinen spezifischen Werten:

```
## Task: i18n — {MODULE_NAME}

Working directory: C:/Users/Luke/Documents/KMU Hub
Basispfad: desktop/src/renderer/src/{BASE_PATH}/

### Dateien (max 7):
{FILE_LIST}

### Aufgabe
1. Lies jede Datei
2. Füge `import { useTranslation } from 'react-i18next'` hinzu (falls nicht vorhanden)
3. Füge `const { t } = useTranslation()` im Component-Body hinzu (vor dem ersten return)
4. Ersetze ALLE hardcoded deutschen Strings durch t()-Calls
5. {CREATE_OR_EXTEND} `desktop/src/renderer/src/i18n/additions/{JSON_FILE}`

### Pattern
IMPORT: import { useTranslation } from 'react-i18next'
HOOK:   const { t } = useTranslation()
JSX:    <p>Text</p>  →  <p>{t('{PREFIX}.section.element')}</p>
TOAST:  toast.success('OK')  →  toast.success(t('common.success'))
ATTR:   placeholder="..."  →  placeholder={t('{PREFIX}.placeholder')}

### Key-Prefix: `{PREFIX}.`
### Bestehende common.* Keys (NICHT in JSON aufnehmen):
save, cancel, delete, search, loading, error, success, noResults, edit, create,
close, confirm, back, next, filter, download, upload, yes, no, retry, details,
status, actions, required, optional, ok, copy, copied, resetFilters

### Regeln
- NUR Dateien in {BASE_PATH}/ anfassen
- NUR additions/{JSON_FILE} erstellen/erweitern, NICHT de.json
- Brand-Namen hardcoded lassen: Cosmi, Zentria, Bexio, DATEV, Lexware, FinAPI
- Technische Strings hardcoded: CSS-Klassen, URLs, IDs, Enum-Werte
- Dateien OHNE deutsche Strings überspringen
- Für .ts Utility-Files ohne React: import i18next from 'i18next' + i18next.t('key')
- JSON-Format: flach, {"prefix.section.element": "Deutscher Text"}
```

---

## Statistik-Übersicht

| Phase | Agents | Dateien | Status |
|-------|--------|---------|--------|
| ~~Schritt 1B: JSON-Extraktion~~ | ~~4~~ | ~~scan only~~ | ✅ DONE |
| ~~Schritt 2: Module Batch A-F~~ | ~18 | ~86 | ✅ DONE |
| ~~Schritt 2.5: Loader Update~~ | 1 | 1 | ✅ DONE |
| ~~Schritt 3: Components Batch G-J~~ | 9 | 46 | ✅ DONE (412 Keys) |
| ~~Schritt 3.5 Wave 1: Straggler-Module~~ | 3 | 14 | ✅ DONE (10 neue JSONs) |
| ~~Schritt 3.5 Wave 2+3~~ | ~7 | ~36 | ✅ DONE (315 Keys) |
| ~~Schritt 3.75: Remaining Strings~~ | 11 | 42 | ✅ DONE (+969 Keys) |
| ~~Schritt 4: Merge & Loader~~ | 1 (Script) | 46→1 JSON | ✅ DONE (7.221 Keys) |
| ~~Schritt 5: Uebersetzungen~~ | 9 (3×3) | 3 JSON | ✅ DONE (7.221×4 Sprachen) |
| ~~Schritt 6: Cleanup~~ | 1 | 1 (d.ts) | ✅ DONE (strict types) |
| **Total verbleibend** | **~5** | **~3 Dateien** | |

## Lesson Learned
- Agents verbrauchen bis zu 250k Tokens bei >10 Dateien. Max 5-7 Dateien pro Agent!
- JSON-Extraktion separat von Instrumentierung machen — sauberer und verifizierbar
- Paired Agents (z.B. wiki-1/wiki-2) besser in separate JSONs schreiben lassen und danach mergen
- Einige Module waren bereits instrumentiert aber hatten keine JSONs — Extraktion-only Agents funktionieren gut
- Bei shared JSON (settings.json): sequentielle Agents, nicht parallel — sonst Merge-Konflikte
- Session-Limits beachten: nach jeder Wave pausieren wenn noetig
- Wenn Agent JSON-Write verpasst (Limit): Keys manuell per Node-Script nachtragen
- Mock-Daten in Stores bewusst nicht instrumentieren — werden durch Backend-Daten ersetzt
