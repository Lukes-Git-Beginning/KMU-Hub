# i18n Migration — Komplette Roadmap

## Context
i18n Migration (react-intl → i18next) in Cosmi Desktop. Ziel: Alle ~440 .tsx/.ts-Dateien instrumentieren, damit die App vollständig DE/EN/FR/IT übersetzbar ist. Agents dürfen max ~5-7 Dateien pro Task bearbeiten, um unter 80k Tokens zu bleiben.

## Aktueller Stand (2026-04-06)

### Committed (8 Commits auf main):
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
| ❌ | Components | — | ~85 Dateien offen |

### i18n.ts Loader
Importiert: crm, dashboard, finanzen, kommunikation, profil, settings, team, work (8/8).
Noch NICHT importiert: kontakte, kontakte-2, chat, wiki, automatisierung, dokumente, admin, meetings, mails, helpdesk, buchhaltung, kalender, rapporte, notifications (14 JSONs).

### Additions-Keys gesamt: ~3.600+

---

## ~~Schritt 1: JSON-Extraktion + Lücken füllen~~ ✅ ERLEDIGT

## ~~Schritt 2: Verbleibende Module (13 Module)~~ ✅ ERLEDIGT

Alle 13 Module vollständig instrumentiert in Session vom 2026-04-06:
- Batch A (vorherige Session): kontakte (partial), chat (partial)
- Batch B: chat complete, wiki
- Batch C: automatisierung, dokumente (partial)
- Batch D: dokumente complete, admin
- Batch E: meetings, mails
- Batch F: helpdesk, buchhaltung, kalender, rapporte, notifications

---

## Schritt 2.5: Loader Update ← NÄCHSTER SCHRITT

1. Alle 14 fehlenden additions/*.json in i18n.ts importieren
2. `npx tsc --noEmit` — muss sauber sein
3. Commit: `feat(i18n): update i18n loader with all module additions`

---

## Schritt 3: Component-Verzeichnisse (~85 Dateien, skip illustrations)

Basispfad: `desktop/src/renderer/src/components/`

### Batch G (3 parallel)

**Agent: layout-1** (7 Dateien)
- AppShell.tsx, BackgroundPattern.tsx, classic/ClassicLayout.tsx, classic/ClassicSidebar.tsx, DeskEnvironment.tsx, DeskFrame.tsx, Header.tsx
- Prefix: `layout.` → Erstelle additions/layout.json

**Agent: layout-2** (7 Dateien)
- ModuleShell.tsx, OfflineBanner.tsx, PageTransitionOutlet.tsx, sidebar/Sidebar.tsx, sidebar/SidebarBadge.tsx, sidebar/SidebarBranding.tsx, sidebar/SidebarModulePanel.tsx
- Prefix: `layout.` → ERWEITERE additions/layout.json

**Agent: layout-3** (8 Dateien)
- sidebar/SidebarNav.tsx, sidebar/SidebarUser.tsx, sidebar/nav-items.ts, dock/DockBar.tsx, dock/DockLayout.tsx, topnav/ModuleOverviewPanel.tsx, topnav/TopNavBar.tsx, topnav/TopNavLayout.tsx
- Prefix: `layout.` → ERWEITERE additions/layout.json

### Batch H (3 parallel)

**Agent: shared-1** (7 Dateien)
- ConfirmDialog.tsx, DetailPanel.tsx, EmptyState.tsx, FormField.tsx, ItemActions.tsx, PageHeader.tsx, StatCard.tsx
- Prefix: `shared.` → Erstelle additions/shared.json

**Agent: shared-2** (9 Dateien)
- LayoutSwitcher.tsx, LoadingSpinner.tsx, PaletteSwitcher.tsx, PasswordExpiryDialog.tsx, TourOverlay.tsx, AnimatedCheckmark.tsx, AnimatedList.tsx, ConfettiBurst.tsx, TextReveal.tsx
- Prefix: `shared.` → ERWEITERE additions/shared.json
- HINWEIS: Viele haben wahrscheinlich KEINE deutschen Strings — nur instrumentieren wenn nötig

**Agent: shared-3** (6 Dateien)
- GlobalSearch/GlobalSearchDialog.tsx, GlobalSearch/QuickActions.tsx, GlobalSearch/RecentSearches.tsx, GlobalSearch/SearchInput.tsx, GlobalSearch/SearchResultGroup.tsx, GlobalSearch/SearchResultItem.tsx
- Prefix: `shared.globalSearch.` → ERWEITERE additions/shared.json

### Batch I (3 parallel)

**Agent: shared-4** (5 Dateien)
- RichTextEditor/RichTextEditor.tsx, RichTextEditor/EditorBubbleMenu.tsx, RichTextEditor/EditorFooter.tsx, RichTextEditor/EditorToolbar.tsx, RichTextEditor/ToolbarButton.tsx
- Prefix: `shared.editor.` → ERWEITERE additions/shared.json

**Agent: header-1** (7 Dateien)
- ClockInButton.tsx, ConnectionStatusIndicator.tsx, DailyPlannerWidget.tsx, HeaderClock.tsx, LanguageSwitcher.tsx, ProfileMenu.tsx, ProfileSwitcher.tsx
- Prefix: `header.` → Erstelle additions/header.json

**Agent: header-2** (8 Dateien)
- SearchBar.tsx, TimeTrackerWidget.tsx, HeaderWidgetSlots.tsx, header-widgets/NextMeetingWidget.tsx, header-widgets/PomodoroWidget.tsx, header-widgets/QuickNoteWidget.tsx, header-widgets/UnreadCountWidget.tsx, header-widgets/WeatherWidget.tsx
- Prefix: `header.` → ERWEITERE additions/header.json

### Batch J (1 Agent)

**Agent: components-misc** (~16 Dateien)
- settings/BexioFieldMappingEditor.tsx, settings/BexioIntegrationCard.tsx, settings/BexioSetupWizard.tsx, settings/BexioSyncDashboard.tsx
- widgets/HelpWidget.tsx, widgets/WidgetContainer.tsx, widgets/WidgetRegistry.tsx, widgets/WidgetWrapper.tsx
- desk/decorations/DeskCalendar.tsx, desk/decorations/DeskClock.tsx, desk/DeskDecorations.tsx
- chat/ChannelMemberList.tsx, chat/ReactionBar.tsx, chat/ReactionPicker.tsx
- onboarding/OnboardingWizard.tsx, dev/ProfileSwitcher.tsx
- Prefixes: je nach Verzeichnis → Erstelle additions/components-misc.json

---

## Schritt 4: Merge & Loader (1 Session-Task, kein Agent)

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
| Schritt 2.5: Loader Update | 1 (manuell) | 1 | TODO |
| Schritt 3: Components Batch G-J | ~10 | ~85 | TODO |
| Schritt 4: Merge & Loader | 1 (manuell) | — | TODO |
| Schritt 5: Übersetzungen | 1-3 | 3 JSON | TODO |
| Schritt 6: Cleanup | 1 (manuell) | — | TODO |
| **Total verbleibend** | **~12 Agents** | **~85 Dateien** | |

## Lesson Learned
- Agents verbrauchen bis zu 250k Tokens bei >10 Dateien. Max 5-7 Dateien pro Agent!
- JSON-Extraktion separat von Instrumentierung machen — sauberer und verifizierbar
- Paired Agents (z.B. wiki-1/wiki-2) besser in separate JSONs schreiben lassen und danach mergen
- Einige Module waren bereits instrumentiert aber hatten keine JSONs — Extraktion-only Agents funktionieren gut
