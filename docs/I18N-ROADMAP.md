# i18n Migration — Komplette Roadmap

## Context
i18n Migration (react-intl → i18next) in Cosmi Desktop. Ziel: Alle ~440 .tsx/.ts-Dateien instrumentieren, damit die App vollständig DE/EN/FR/IT übersetzbar ist. Agents dürfen max ~5-7 Dateien pro Task bearbeiten, um unter 80k Tokens zu bleiben.

## Aktueller Stand (2026-04-06)

### Committed (4 Commits auf main):
- `1b7ee22` — i18next Infrastruktur + react-intl entfernt (20 Dateien)
- `cf825a7` — Dashboard, CRM instrumentiert (78 Dateien)
- `10cffa5` — Work, Profil, Settings, Kommunikation, Finanzen, Team (107 Dateien)
- `88d9f1f` — Schritt 1B: 4 additions JSONs extrahiert + Loader aktualisiert

### Uncommitted (Batch A teilweise):
- **kontakte-2** ✅ — 6 Dateien instrumentiert, kontakte-2.json erstellt (64 Keys)
- **chat-1** ✅ — 7 Dateien instrumentiert, chat.json erstellt (37 Keys)
- **kontakte-1** ⚠️ FEHLER — Agent abgebrochen (Escape). 6/7 .tsx-Dateien modifiziert, ABER:
  - `kontakte.json` wurde NICHT erstellt (Keys fehlen!)
  - `DuplicateMatchCard.tsx` wurde NICHT bearbeitet
  - Fix nötig: kontakte.json aus den t()-Calls in den 6 modifizierten Dateien extrahieren + DuplicateMatchCard.tsx instrumentieren

### Modul-Status Phase 3

| Status | Modul | Dateien | JSON | Keys |
|--------|-------|---------|------|------|
| ✅ | auth | 2/4 | — | in de.json |
| ✅ | security | 8/11 | — | in de.json |
| ✅ | dashboard | 20/21 | dashboard.json ✅ | 187 |
| ✅ | crm | 17/20 | crm.json ✅ | 302 |
| ✅ | work | 37/37 | work.json ✅ | 323 |
| ✅ | profil | 15/16 | profil.json ✅ | 275 |
| ✅ | settings | 33/34 | settings.json ✅ | 233 |
| ✅ | kommunikation | 22/22 | kommunikation.json ✅ | 167 |
| ✅ | finanzen | 15/16 | finanzen.json ✅ | 244 |
| ✅ | team | 14/14 | team.json ✅ | 233 |
| ⚠️ | kontakte | 14/15 | kontakte-2.json ✅, kontakte.json ❌ | 64 (Teil) |
| ✅ | chat | 7/13 | chat.json ✅ | 37 (Teil) |
| ❌ | 11 Module | 0 | ❌ | Nicht begonnen |
| ❌ | Components | 0 | ❌ | Nicht begonnen |

### i18n.ts Loader
Importiert: crm, dashboard, finanzen, kommunikation, profil, settings, team, work (8/8 fertig).
Noch NICHT importiert: kontakte, chat (+ 11 weitere Module aus Schritt 2).

### Additions-Keys gesamt: ~2.065 (1.964 + 64 kontakte-2 + 37 chat)

---

## ~~Schritt 1: JSON-Extraktion + Lücken füllen~~ ✅ ERLEDIGT

4 Agents haben JSONs extrahiert für settings (233), kommunikation (167), finanzen (244), team (233). Loader aktualisiert. TypeScript kompiliert sauber.

---

## Schritt 2: Verbleibende Module (13 Module, ~95 Dateien) ← IN PROGRESS

Jeder Agent bekommt max 7 Dateien + erstellt/erweitert die JSON.
Basispfad: `desktop/src/renderer/src/modules/`

### ~~Batch A~~ (teilweise erledigt)

**Agent: kontakte-1** ⚠️ ABGEBROCHEN — 6/7 Dateien instrumentiert, JSON NICHT erstellt
- ✅ KontaktePage.tsx, ContactFormDialog.tsx, ContactDetailPanel.tsx, FirmaDetailPanel.tsx, ConsentPanel.tsx, DuplicateDetectionDialog.tsx
- ❌ DuplicateMatchCard.tsx — nicht bearbeitet
- ❌ kontakte.json — nicht erstellt
- **FIX NÖTIG:** kontakte.json aus t()-Calls extrahieren + DuplicateMatchCard.tsx instrumentieren

**Agent: kontakte-2** ✅ (6/8 Dateien instrumentiert, 2 übersprungen)
- ✅ CustomFieldRow, CustomFieldsConfig, GroupManagerDialog, ImportContactsDialog, MergeFieldSelector, NewsletterPanel
- ⏭️ CustomFieldPreview (bereits instrumentiert), adapters.ts (keine UI-Strings)
- Erstellt: additions/kontakte-2.json (64 Keys)

**Agent: chat-1** ✅ (7 Dateien instrumentiert)
- ✅ ChatLayout, ChannelHeader, ChannelList, CreateChannelDialog, MessageBubble, MessageInput, MessageList
- Erstellt: additions/chat.json (37 Keys)

### Batch B (3 parallel)

**Agent: chat-2** (6 Dateien)
- messages/FileAttachmentCard.tsx, FileDropZone.tsx, MentionAutocomplete.tsx, ReactionBar.tsx, ReactionPicker.tsx, threads/ThreadPanel.tsx
- Prefix: `chat.` → ERWEITERE additions/chat.json

**Agent: wiki-1** (6 Dateien)
- WikiPage.tsx, WikiArticle.tsx, WikiArticleHeader.tsx, WikiEditor.tsx, WikiSidebar.tsx, WikiSearch.tsx
- Prefix: `wiki.` → Erstelle additions/wiki.json

**Agent: wiki-2** (6 Dateien)
- WikiCategoryDialog.tsx, WikiShareDialog.tsx, WikiTemplateDialog.tsx, WikiTreeNode.tsx, WikiVersionHistory.tsx, WikiVersionItem.tsx
- Prefix: `wiki.` → ERWEITERE additions/wiki.json

### Batch C (3 parallel)

**Agent: automatisierung-1** (6 Dateien)
- AutomatisierungPage.tsx, AutomationEditor.tsx, AutomationWizard.tsx, ActionConfigurator.tsx, ConditionBuilder.tsx, TriggerSelector.tsx
- Prefix: `automatisierung.` → Erstelle additions/automatisierung.json

**Agent: automatisierung-2** (6 Dateien)
- ExecutionLogViewer.tsx, TemplateGallery.tsx, nodes/ActionNode.tsx, nodes/ConditionNode.tsx, nodes/TriggerNode.tsx, nodes/nodeTypes.ts
- Prefix: `automatisierung.` → ERWEITERE additions/automatisierung.json

**Agent: dokumente-1** (6 Dateien)
- DokumentePage.tsx, FileContextMenu.tsx, FileDetailPanel.tsx, FilePreviewModal.tsx, FolderCreateDialog.tsx, OnlyOfficeEditor.tsx
- Prefix: `dokumente.` → Erstelle additions/dokumente.json

### Batch D (3 parallel)

**Agent: dokumente-2** (5 Dateien)
- RenameDialog.tsx, ShareDialog.tsx, ShareLinkDialog.tsx, TemplateGalleryDialog.tsx, VersionHistoryPanel.tsx
- Prefix: `dokumente.` → ERWEITERE additions/dokumente.json

**Agent: admin-1** (5 Dateien)
- CalDAVAdminPage.tsx, InfrastrukturPage.tsx, plugins/ExecutionLogViewer.tsx, plugins/IndustryTemplateGallery.tsx, plugins/PermissionApprovalDialog.tsx
- Prefix: `admin.` → Erstelle additions/admin.json

**Agent: admin-2** (5 Dateien)
- plugins/PluginDetailDialog.tsx, plugins/PluginListPage.tsx, plugins/PluginSettingsEditor.tsx, plugins/ValidationRulesEditor.tsx, plugins/WorkflowRulesEditor.tsx
- Prefix: `admin.` → ERWEITERE additions/admin.json

### Batch E (3 parallel)

**Agent: meetings-1** (7 Dateien, skip index.ts)
- MeetingsPage.tsx, MeetingFormDialog.tsx, MeetingDetailPanel.tsx, MeetingLobby.tsx, MeetingRoomView.tsx, CallOverlay.tsx, MeetingActionItems.tsx
- Prefix: `meetings.` → Erstelle additions/meetings.json

**Agent: meetings-2** (2 Dateien)
- MeetingNotesPanel.tsx, MeetingSummaryView.tsx
- Prefix: `meetings.` → ERWEITERE additions/meetings.json

**Agent: mails** (7 Dateien, skip compose-shared.tsx wenn nur Types)
- MailsPage.tsx, ComposeInline.tsx, ComposeModal.tsx, compose-shared.tsx, ComposeWindowPage.tsx, EmailTemplateDialog.tsx, ExportDialog.tsx, ImportWizard.tsx
- Prefix: `mails.` → Erstelle additions/mails.json

### Batch F (3 parallel)

**Agent: helpdesk** (7 Dateien)
- HelpdeskPage.tsx, BusinessHoursDialog.tsx, CannedResponsePicker.tsx, CannedResponsesPanel.tsx, CSATWidget.tsx, SLABadge.tsx, TicketRoutingConfig.tsx
- Prefix: `helpdesk.` → Erstelle additions/helpdesk.json

**Agent: buchhaltung** (6 Dateien)
- BuchhaltungPage.tsx, ExpenseFormDialog.tsx, ExportDialog.tsx, InvoiceDetailPanel.tsx, InvoiceFormDialog.tsx, PaymentRecordDialog.tsx
- Prefix: `buchhaltung.` → Erstelle additions/buchhaltung.json

**Agent: kalender+rapporte+notifications** (11 Dateien kombiniert)
- kalender/: KalenderPage.tsx, CalendarBrowseDialog.tsx, CategoryManagerDialog.tsx, RoomBookingView.tsx, adapters.ts
- rapporte/: RapportePage.tsx, SignatureCanvas.tsx, SketchCanvas.tsx
- notifications/: NotificationBell.tsx, NotificationCenter.tsx, NotificationToast.tsx
- Prefixes: `kalender.`, `rapporte.`, `notifications.`
- Erstelle additions/kalender.json, rapporte.json, notifications.json

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
| Schritt 2: Module Batch A-F | ~18 | ~86 | TODO |
| Schritt 3: Components Batch G-J | ~10 | ~85 | TODO |
| Schritt 4: Merge & Loader | 1 (manuell) | — | TODO |
| Schritt 5: Übersetzungen | 1-3 | 3 JSON | TODO |
| Schritt 6: Cleanup | 1 (manuell) | — | TODO |
| **Total verbleibend** | **~30 Agents** | **~171 Dateien** | |

Pro Session (3 parallel, ~40k tokens/Agent): ~4-5 Batches = ~12-15 Agents pro Session.
Geschätzt 2-3 Sessions für Schritt 2+3, dann 1 Session für Schritt 4-6.
