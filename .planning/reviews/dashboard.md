# Review-Fäden — dashboard

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `dashboard` · **Strom:** L · **Reviewer (zugeteilt):** offen

---

## Phase 1 — Modul-Einstellungen (DashboardSettingsPanel)  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route(n): `/` (Übersicht) → Sidebar unten „Modul-Einstellungen" → Eintrag „Dashboard" (kontext-vorausgewählt, nur bei exakt `/`)
- Schritte: Persönlich: „Begrüßung anzeigen" abwählen → Overlay schließen → Dashboard neu laden → Begrüßungszeile weg, Layout intakt. „Kompakt" → Widget-Raster wird enger (Zeilenhöhe 64 statt 80, Abstände 12 statt 16). Für alle: Widget bei „Erlaubte Widgets" abwählen → fliegt aus „Widget hinzufügen"-Dialog UND aus dem Team-Standard.

**Worauf achten (Feinschliff):**
- [ ] Layout/Hierarchie bei voller Breite + schmal (760 geprüft, Screenshot)
- [ ] Keine Raw-i18n-Keys, keine Emojis (QA: 0 Raw-Keys, 4 Sprachen)
- [ ] Interaktionen echt: Begrüßung + Dichte greifen real; „Erlaubte Widgets" filtert den Picker real
- [ ] Widget-Chip-Labels korrekt übersetzt (bewusst OHNE WidgetRegistry-Import — siehe technische Notiz unten)
- [ ] Kontext-Preselect: `/` öffnet Dashboard-Eintrag, andere Routen NICHT (Resolver-Sonderfall exact-match für `/`)

**Screenshots:** `desktop/.qa-screenshots/dashboard-settings/` (panel-top, panel-tenant, dashboard-no-greeting, panel-760) — QA `desktop/scripts/qa-dashboard-settings.mjs`

**Bekannte offene Punkte / Backend-Bedarf:**
- Tenant-Settings mock-first (`stores/dashboardSettings.ts`) — Persistenz via `tenant_settings` (module_id='dashboard') = Luke. „Team-Standard-Widgets" wird erst beim User-Anlegen serverseitig angewendet (Backend).
- Spec nannte „Standard-Zeitraum der KPI-Widgets" — bewusst NICHT gebaut: KPI-Widgets haben kein Zeitraum-Konzept (fix monatlich, `KpiRevenue.tsx`); ein Pref wäre ein No-op. Stattdessen Begrüßung + Dichte (greifen real).
- **Technische Notiz:** `WidgetRegistry` evaluiert Widget-Namen via `i18next.t()` zur Modul-Ladezeit — ein Import aus der boot-geladenen Settings-Registry zieht das VOR die i18n-Init und leert die Namen app-weit. Panel übersetzt deshalb zur Render-Zeit (statische Key-Map). Latente Fragilität der Registry für Darien notiert.
- Typ-Erweiterung: `SettingsModuleId = ModuleId | 'dashboard'` (lib/module-settings.ts) — dashboard ist nicht preis-/leitbar, tenant-Sektionen damit admin-only.
- Vorbestehend (nicht diese Phase): `KpiRevenue.tsx:31` hardcodet deutsche Monatslabels mit ASCII-Umlaut („Maer") statt i18n.

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

---

## Technische Notiz — P1/P2 bereits implementiert in Phase 2

react-grid-layout in `WidgetContainer.tsx` liefert 2D-Grid-Drag+Resize via `isDraggable`/`isResizable` im Edit-Mode — das ist P1+P2 in einem. dnd-kit wäre das falsche Werkzeug (List-Sortierung vs. freies 2D-Grid). Layout-Persistenz: `stores/dashboard.ts` speichert in localStorage und synct debounced via PUT /api/v1/dashboard/layout. Kein separater P2-Build nötig.

---

---

## Phase 5 — Widget-Gating nach Modul-Flags  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Was gebaut:**

- `WidgetDefinition` in `WidgetRegistry.tsx` um statisches Feld `module?: ModuleId` erweitert
- Alle 19 Widgets mit Modul-Mapping versehen (siehe Tabelle unten)
- `WidgetContainer.tsx`: Picker + Grid filtern zusätzlich nach `modules.<id>` Feature-Flags
- `useFeatureFlags.ts`: Fail-closed (app-weit) — bei Error/Loading/keinen Daten gibt `isEnabled` `false` zurück; DEV-QA-Override (`__cosmi_qa_flags__`) bleibt erhalten
- `WidgetContainer.tsx`: Dashboard-lokales fail-open — wenn Flags nicht verfügbar (error/loading/kein data), werden alle module-gemappten Widgets trotzdem angezeigt (kein Flash-of-empty-Dashboard); `isWidgetAllowed()` prüft DEV-Override zuerst, dann fail-open-Fallback, dann `flags[modules.<id>]`
- `App.tsx`: `window.__cosmi_queryClient__` in DEV exposiert (QA-Helfer, tree-shaken in Prod)
- `useFeatureFlags.ts`: `window.__cosmi_qa_flags__` als pre-IS_DEMO-Override in DEV (ermöglicht Playwright-Gating-Tests ohne Backend)
- Layout-Einträge deaktivierter Widgets bleiben im Store — kein Datenverlust beim Re-Aktivieren

**Hinklicken (Pfad in der App):**
- Route `/` → Widgets-Abschnitt → mit aktivem Backend und `modules.crm=false` verschwinden CRM-Widgets aus Grid + Picker
- In Demo/Dev-Mode (kein Backend) → alle Widgets sichtbar (fail-open)

**Widget → Modul-Mapping:**

| Widget-ID | Widget-Name | Modul |
|---|---|---|
| `recent-contacts` | Letzte Kontakte | `crm` |
| `deal-pipeline` | Deal Pipeline | `crm` |
| `kpi-deals` | Deal-Überblick | `crm` |
| `unread-messages` | Ungelesene Nachrichten | `chat` |
| `team-chat` | Team-Chat | `chat` |
| `kpi-revenue` | Umsatz | `finance` |
| `kpi-tasks` | Aufgaben | `tasks` |
| `my-tasks` | Meine Aufgaben | `tasks` |
| `calendar-upcoming` | Termine heute | `calendar` |
| `my-calendar` | Mein Kalender | `calendar` |
| `team-status` | Team-Status | `team` |
| `absences` | Abwesenheiten | `team` |
| `birthdays` | Geburtstage | `team` |
| `time-clock` | Stempeluhr | `zeiterfassung` |
| `activity-feed` | Aktivitäten | (kein Modul — immer verfügbar) |
| `quick-actions` | Schnellaktionen | (kein Modul — immer verfügbar) |
| `notification-summary` | Benachrichtigungen | (kein Modul — immer verfügbar) |
| `notification-feed` | Benachrichtigungs-Feed | (kein Modul — immer verfügbar) |
| `mini-chart` | Umsatz-Chart | (kein Modul — immer verfügbar) |

**Fallback-Verhalten:**
- `useFeatureFlags` / `FeatureGate`: **fail-closed** (app-weit) — bei Error/Loading/keiner Antwort → `false`. User-Entscheidung Luke 2026-06-10.
- `WidgetContainer` (Dashboard-lokal): **fail-open** — Flags nicht verfügbar → alle Widgets sichtbar (kein Flash-of-empty-Dashboard). Betrifft ausschließlich Widget-Sichtbarkeit.
- Demo-Mode (`--mode demo`) → IS_DEMO=true → alle Flags `true`, kein Fetch (in `useFeatureFlags`)

**Offene Darien-Fragen:**

1. **Dashboard-lokales fail-open:** Entschieden — fail-open ist auf `WidgetContainer` beschränkt, `useFeatureFlags`/`FeatureGate` bleiben app-weit fail-closed (User-Entscheidung Luke 2026-06-10). Offene Detailfrage: Soll das Dashboard-Widget-Gating auch fail-closed mit Cache (z.B. 24h localStorage der zuletzt bekannten Flags) werden, oder reicht das aktuelle fail-open im Pilot-Betrieb aus?

2. **Registry-Ladezeit-Fragilität (carry-over Phase 1):** `WidgetRegistry` evaluiert `i18next.t()` zur Modul-Ladezeit — Import zieht das VOR i18n-Init und leert Widget-Namen app-weit. Neue `module?` Feld ist rein statisch (keine t()-Aufrufe) und umgeht die Falle korrekt. Latente Fragilität für zukünftige Registry-Erweiterungen bleibt dokumentiert.

3. **QA-Override-Mechanismus (`__cosmi_qa_flags__`):** DEV-only, tree-shaken in Prod. Soll nur für Playwright-Tests verwendet werden. Falls das Infrastruktur-Team ein dedizierteres Test-Setup aufbaut (z.B. MSW auch im Dev-Server), kann der Mechanismus entfernt werden.

**Screenshots:** `desktop/.qa-screenshots/dashboard-gating/` — QA `desktop/scripts/qa-dashboard-gating.mjs`
- `a-normal-load.png` — alle 5 Seeded-Widgets sichtbar (fail-open)
- `c-crm-disabled-grid.png` — CRM-Widgets korrekt ausgeblendet
- `c-crm-disabled-picker.png` — Picker ohne CRM-Einträge

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
