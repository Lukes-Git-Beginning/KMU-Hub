# i18n Migration — Komplette Roadmap

## Context
i18n Migration (react-intl → i18next) in Cosmi Desktop. Ziel: Alle ~440 .tsx/.ts-Dateien instrumentieren, damit die App vollständig DE/EN/FR/IT übersetzbar ist. Agents dürfen max ~5-7 Dateien pro Task bearbeiten, um unter 80k Tokens zu bleiben.

## Aktueller Stand (2026-04-06)

### Committed (13 Commits auf main):
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

### i18n.ts Loader
Alle 31 additions-JSONs (22 Module + 9 Components) importiert und in mergedDE gemergt.

### Additions-Keys gesamt: ~4.000+

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

## Schritt 4: Merge & Loader (1 Session-Task, kein Agent) ← NÄCHSTER SCHRITT

1. Alle additions/*.json Keys in de.json mergen
2. i18n.ts Loader: ALLE additions importieren
3. `npx tsc --noEmit` — muss sauber sein
4. `grep -r` nach verbleibenden hardcoded deutschen Strings
5. Commit: `feat(i18n): complete Phase 3 instrumentation`

---

## Schritt 5: Phase 4 — Übersetzungen (EN/FR/IT)

1. de.json ist jetzt komplett
2. en.json generieren (alle Keys übersetzen)
3. fr.json generieren
4. it.json generieren
5. Commit: `feat(i18n): add EN/FR/IT translations`

---

## Schritt 6: Phase 5 — Cleanup

1. i18next.d.ts auf strikte Typen umstellen
2. Unused Keys entfernen
3. Final Build + Smoke Test
4. Commit: `chore(i18n): strict types and final cleanup`

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
| Schritt 4: Merge & Loader | 1 (manuell) | — | TODO |
| Schritt 5: Übersetzungen | 1-3 | 3 JSON | TODO |
| Schritt 6: Cleanup | 1 (manuell) | — | TODO |
| **Total verbleibend** | **~3** | **~3 Dateien** | |

## Lesson Learned
- Agents verbrauchen bis zu 250k Tokens bei >10 Dateien. Max 5-7 Dateien pro Agent!
- JSON-Extraktion separat von Instrumentierung machen — sauberer und verifizierbar
- Paired Agents (z.B. wiki-1/wiki-2) besser in separate JSONs schreiben lassen und danach mergen
- Einige Module waren bereits instrumentiert aber hatten keine JSONs — Extraktion-only Agents funktionieren gut
