# QA-Log — admin (Sub-Terminal, Branch `parallel/admin`)

> Dev: `VITE_DEV_PORT=5174 npm run dev` (Demo/MSW, kein Backend). Screenshot-QA-Scripts: `scripts/qa-admin-*.mjs`.
> Build-Gate: `npm run build > /tmp/admin-build.log 2>&1; echo "EXIT $?"` + `grep -i error`.

## A-0 — Ist-Research + Marktrecherche (Gate, beantwortet)

### Ist-Stand verifiziert
- AdminHub = 4 Tabs (IT | Sicherheit | Abrechnung | Integrationen), URL-synced, Gating `userHasRole(['admin','it_support'])`. Screenshots angesehen.
- **Overlap-Falle:** Scope existiert fragmentarisch in TABU-Modulen → admin-native distinkt bauen. team = HR (EmployeeProfile), admin = Account/Access-Layer. Branding basic in ITAdminHubTab. Lizenz read-only in settings/BillingSettingsTab. RBAC = statisches `@/config/roles`.
- Tech: i18n = i18next-**ICU** → **single-brace `{var}`** (`{{…}}` ist Bug, sichtbar als roher Text in settings/Billing-Placeholdern, TABU). Flache dotted Keys, 4 Sprachen. Kein `admin`-MSW-Handler (neu). `/api/v1/users` in Demo nicht gemockt → eigene Seed-Quelle. `electron-vite dev` akzeptiert kein `--port` → env-gateter `server.port` (Default 5173, `VITE_DEV_PORT=5174`).

### Marktrecherche — Markt vs. Cosmi-Ist (Linear, Notion, Slack, GitLab, M365, Auth0/Okta, Vanta, WorkOS)
**Benutzer** · *Funktion:* Liste (Name/Mail/Rolle/Status/letzter Login), 3 Status (active/pending/deactivated), Invite-Modal mit Rolle beim Einladen, kontextuelle Bulk-Toolbar, Seat-Limit inline im Modal. *Gestaltung:* dichte Tabelle (~36–40px, Linear), Status als Pill **+ Icon** (nie nur Farbe), Side-Drawer/Detail, Seat-Progress-Bar (Amber 80% / Rot 100%). → **Cosmi:** Editorial-Liste, ganze Zeile klickbar → `shared/DetailModal`, Rollen-Dot + Pill-Status mit Icon, kompakter Seat-Meter (Bar amber→rot).
**RBAC** (A-2) · Rollen=Spalten, Ressourcen=Zeilen, gruppiert, Checkboxen, Sticky Header+Spalte; 5 feste Rollen, keine Custom Roles für 1.0.
**Lizenz** (A-3) · Seat-Auslastung + Modul-Aktivierung als Karten-Grid (aktiv = voll, inaktiv = gedimmt); `@/lib/pricing` + `useTenant` konsumieren, Billing-Finanzview nicht duplizieren.
**Branding** (A-4) · 2-Spalten mit Live-Preview, Logo-Drag&Drop, Hex + Kontrast; Brand-Color in Cosmi-Token-Palette, kein Full-White-Label.
**Über-Engineering vermeiden:** SCIM/Directory-Sync, ABAC, Custom Roles, Full-White-Label, Real-time Seat-Charts.

### Dariens Gate-Antworten (umgesetzt)
1/2 team↔admin: **dieselbe Person, zwei Layer** — Seed aus `shared-ids.ts`/`CURRENT_USER`, keine parallele Registry. team unangetastet.
2 RBAC-Quelle: `@/config/roles` kanonisch; IT-Tab-„Berechtigungen" konsolidieren (A-2).
3 RBAC: (b) stateful mock-persist, 5 feste Rollen, keine Custom.
4 Lizenz: eigener Provisioning-Bereich neben Billing, `@/lib/pricing`+`useTenant` konsumieren.
5 Branding: aus ITAdminHubTab herauslösen + erweitern (Live-Preview), Token-Palette.
6 Port: env-gateter `server.port` ✓. 7 Stash: unangetastet. 8 sub-admin.md von main gesynct, qa-admin.md (dieses File) angelegt.

---

## A-1 — Benutzerverwaltung ✅ (1/5)

**Gebaut:** Neuer Tab **„Benutzer"** (`/admin/users`, erster Tab). Account/Access-Layer:
- `api/admin-types.ts` (Contract `AdminUser`/`AdminUserStatus`/`InviteUserInput`), `api/hooks/useAdminUsers.ts` (list/invite/update/resend).
- `mocks/handlers/admin.ts` (stateful: GET list, POST invite, PATCH role/status, POST resend) + `mocks/data/admin-users.ts` (14 Seed-Accounts aus `IDS.users`+`CURRENT_USER`, 11 aktiv / 2 eingeladen / 1 deaktiviert).
- UI: `modules/admin/users/` — `UsersAdminHubTab` (Liste, Suche, Status-Filter mit Counts, `shared/SortMenu`, Seat-Meter amber/rot), `InviteUserDialog` (E-Mail + Rolle, Seat-Hinweis inline), `UserDetailModal` (Rolle ändern, aktivieren/deaktivieren, Einladung erneut senden/zurückziehen, Self-Protection für eingeloggten Admin), `presentation.ts` (Rolle/Status-Visuals, Intl-Relativzeit).
- i18n: 52 Keys/Sprache (`admin.users.*` + `admin.hub.tabs.users`), single-brace ICU, 4 Sprachen via `scripts/add-admin-users-i18n.mjs`.

**Verify (Screenshots angesehen, :5174):**
- Build EXIT 0, ESLint EXIT 0 auf geänderten Dateien.
- DE + EN: **0 Raw-Keys, 0 `{{var}}`, 0 Console-Errors**. Intl-Relativzeit lokalisiert korrekt (DE „vor 38 Minuten" / EN „38 minutes ago" / „Never").
- Liste sortier-/filterbar; Status-Filter-Counts stimmen (Alle 14 / Aktiv 11 / Eingeladen 2 / Deaktiviert 1).
- **Stateful bestätigt:** Einladen erzeugt sichtbaren Pending-User (Name aus E-Mail abgeleitet), Count Eingeladen 2→3, Seat-Meter springt 13/14→14/14 (amber→rot bei 100%), Toast; **überlebt Navigation** (/admin/it → zurück).
- Detail-Modal: ganze Zeile klickbar, Gradient-Stripe, Rolle/Konto/Zugang-Sektionen; Eingeladene zeigen „erneut senden" + „zurückziehen", Aktive „deaktivieren", Deaktivierte „reaktivieren".

**Offen für Luke (backend-gaps.md → admin):** echter Invite-Flow (E-Mail/Token), Account-Provisioning, Status-Persistenz im Gateway.

---

## A-2 — Rollen & RBAC-Matrix ✅ (2/5)

**Gebaut:** Neuer Tab **„Rollen"** (`/admin/roles`, 2. Tab) — die **kanonische** Permission-Surface.
- `mocks/data/admin-permissions.ts` (7 Capability-Gruppen × 17 Capabilities + Default-Grants pro Rolle, Admin implizit alles), Contract in `api/admin-types.ts` (`PermissionGroup`/`PermissionMatrix`/`SetPermissionInput`), Hooks `api/hooks/useRolePermissions.ts` (GET + optimistisches PATCH).
- `mocks/handlers/admin.ts`: GET `/api/v1/admin/permissions` (groups+matrix), PATCH (cell-Toggle, Admin server-seitig gesperrt).
- UI `modules/admin/roles/RolesAdminHubTab.tsx`: Rollen-Summary-Karten (5 Rollen aus `@/config/roles` + Nutzer-Counts aus A-1), **sticky Permission-Matrix** (Rollen=Spalten, Capabilities=Zeilen gruppiert, Sticky Header+erste Spalte), **Admin-Spalte gesperrt (Lock-Icon)**, Checkbox-Toggle, Demo-Hinweis „nicht erzwungen". Rollen-Karte klickbar → `DetailModal` mit zugeordneten Nutzern.
- i18n: 32 Keys/Sprache (`admin.roles.*` + `admin.hub.tabs.roles`, ICU-Plural `userCount`), 4 Sprachen via `scripts/add-admin-roles-i18n.mjs`.

**Verify (Screenshots angesehen, :5174):** Build EXIT 0, ESLint 0. DE+EN **0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors**. Matrix lesbar bei 17 Capabilities (Sticky-Scroll), Admin-Spalte Lock. Rollen-Counts stimmen (1+1+2+1+9 = 14, konsistent mit A-1-Liste). **Toggle wirkt optimistisch + persistiert** (überlebt Navigation /admin/users → zurück). Rollen-Detail zeigt zugeordnete Nutzer (Projektleiterin → Sarah Müller + Laura Neumann). Skeleton-Loading beim ersten Laden (kein Spinner).

**Konsolidierung (Darien-Antwort 2):** Der „Berechtigungen"-Subtab in `settings/tabs/ITAdminTab.tsx` (`PermissionsSection`) ist eine **Fake**-Matrix (falsche Rollen `Admin/Manager/Mitarbeiter/Extern`, kein Persist, „Speichern"=Toast). Liegt in `settings/` (TABU) → **nicht angetastet**; die A-2-Matrix ist jetzt die kanonische. **→ Coordination: settings/Main-Lane sollte den Legacy-Subtab entfernen** (sonst zwei Permission-UIs im Hub).

**Offen für Luke:** echte RBAC-Persistenz/Enforcement im Gateway (Capability-Grants pro Rolle).

---

## A-3 — Lizenz / Modul-Aktivierung ✅ (3/5)

**Gebaut:** Neuer Tab **„Lizenz"** (`/admin/license`, 3. Tab) — Provisioning-Layer **neben** dem read-only Billing-Tab (kein Kosten-View dupliziert).
- `mocks/data/admin-license.ts` (24 Module mit Gruppe + active + assignedSeats, 17 aktiv / 7 inaktiv), Contract `TenantModule`/`LicenseResponse`/`ToggleModuleInput` in `api/admin-types.ts`, Hooks `api/hooks/useTenantModules.ts` (GET + optimistisches Toggle).
- `mocks/handlers/admin.ts`: GET `/api/v1/admin/license`, PATCH (Modul an/aus; Deaktivieren gibt Seats frei).
- UI `modules/admin/license/`: `LicenseAdminHubTab` (3 Übersichtskarten: Plan `useTenant` + Status + Verlängerung, **Seat-Bar** `useAdminUsers` amber/rot, Aktive-Module-Zähler) + gruppiertes Modul-Karten-Grid (Kern/Kommunikation/Team/Branche/Tools), aktive Karten solide + Seat-Count + grüner Switch, **inaktive gestrichelt/gedimmt + „Nicht aktiviert"**. `moduleMeta.ts` mappt Modul→`layout.navItems.*`-Label (kein team-Import). `@/lib/pricing` `MODULE_PRICES` konsumiert (Preis-Chip).
- i18n: 22 Keys/Sprache (`admin.license.*` + `admin.hub.tabs.license`, ICU-Plural `assigned`), 4 Sprachen via `scripts/add-admin-license-i18n.mjs`.

**Verify (Screenshots angesehen, :5174):** Build EXIT 0, ESLint 0 (prefer-const-Fix). DE+EN **0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors**. **Seats 13/14 konsistent mit A-1** (11 aktiv + 2 eingeladen), Aktive-Module 17/24, Verlängerungsdatum lokalisiert. **Toggle wirkt optimistisch + persistiert** (Schichtplanung aktiviert, überlebt Navigation /admin/roles → zurück). Inaktive Karten visuell klar getrennt (dashed/dimmed).

**Offen für Luke:** echte tenant-weite Modul-Lizenzierung/Provisioning + Seat-Enforcement.
