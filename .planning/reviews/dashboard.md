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

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
