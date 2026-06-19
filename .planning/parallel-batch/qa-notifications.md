# QA-Protokoll — notifications (Sub-Terminal, parallel/notifications)

Build-Gate: `npm run build` (electron-vite) im `desktop/`, echter Exit + `grep -i error`.
QA-Server: `npx vite --config vite.qa.config.mjs` → :5174 (reiner Renderer, demo-mode).
Screenshots: `desktop/.qa-screenshots/notif-*/` (gitignored, lokal angesehen).

---

## N-1 — Schema-Fix + MSW-Seed-Upgrade  ✅ (2026-06-19)

**Geändert:** `mocks/handlers/notifications.ts` (kompletter Daten-/Handler-Rewrite).

**Was:**
- Seeds auf das `components['schemas']['Notification']`-Schema gebracht: `read`→`is_read`,
  neu `priority` (low/normal/high/**urgent** — Plantext sagte „critical", echtes Enum ist
  `urgent`, Code folgt Schema), `module_id` (nav-id-konform: contacts/tasks/chat/team/
  vertraege/documents/finance/projects/settings → N-4 Sidebar-Badges können andocken),
  `deep_link` (echte Routen: `/pipeline`, `/work/my-tasks`, `/kommunikation`, `/vertraege`,
  `/admin/security`, …), `event_type_key`, `read_at`.
- **6 von 13 ungelesen** (001,002,003,006,009,v-3) → verteilt contacts ×2 / tasks / chat /
  settings / vertraege. Alle 4 Priority-Stufen im Bild (urgent=Sicherheitswarnung rot).
- List-Handler liest jetzt `is_read` + `module_id` (vom Hook gesendet) + legacy `unread` +
  `page`/`page_size`. `unread-count` über `!is_read`. mark-read / read-all **stateful**
  (mutieren `is_read`+`read_at`, überleben Refetch).
- Preferences-GET liefert jetzt ein **Array** `NotificationPreference[]` (vorher Objekt →
  `preferences.find` crashte beim Öffnen des Panels). Neuer **PUT-Handler** (Upsert by
  `event_type_key`, stateful). 2 vorgesetzte Prefs für Demo-Tiefe.
- event-types auf Hook-Schema: `event_key`/`display_name`/`description`/`default_in_app`/
  `default_desktop_push`/`module_id` (14 Typen, 9 Modul-Gruppen).

**Verify (qa-notif-n1.mjs, :5174, 1440×1000, DE):**
- Bell-Badge = **6**; Center-Subtitle „6 ungelesene Benachrichtigungen" (ICU-Plural ok, kein `{{}}`).
- 6 Cards mit `border-l-primary` (unread fett + blauer Rand); Priority-Icons farblich
  differenziert (rot/orange/blau/grau); Modul-Badges gefüllt.
- Unread-Tab: genau 6 Cards, Pagination „Seite 1 (6 gesamt)".
- Preferences-Panel öffnet **ohne Crash**; 9 Gruppen-Header, 28 Checkboxen; seeded Prefs
  korrekt reflektiert (Dokument geteilt → Desktop aus).
- Toggle „Deal erstellt → In-App" geflippt, Panel zu+auf (Refetch) → Änderung **bleibt** erhalten.
- **0 Raw-Keys, 0 `{{}}`.**

**Bekannt/akzeptiert:** Console `net::ERR_CONNECTION_REFUSED` = `useNotificationWebSocket`
gegen echtes Backend (P4 🔒 out-of-scope, MSW-Demo hat kein WS). Kein App-Fehler.

**Bewusst nach N-2 verschoben:** Modul-Badges zeigen aktuell die rohe id („contacts").
Lesbares Label („Kontakte") kommt mit der zentralen Modul-Label-Funktion in N-2 (Filter
braucht dieselbe Quelle) — dann Badge + Filter konsistent.

**Commit:** siehe `feat(notifications): fix seed schema …` auf `parallel/notifications`.

---

## N-2 — Modul-Filter + Sortierung  ✅ (2026-06-19)

**Geändert:** `modules/notifications/NotificationCenter.tsx` (Filter/Sort-Leiste,
lesbare Badges), `i18n/messages/{de,en,fr,it}.json` (+6 Keys ins notifications-Cluster).

**Was:**
- **Modul-Filter-Chips** (client-seitig, auf der geladenen Liste): „Alle Module" +
  ein Chip je vorkommendem Modul, jeweils mit Farb-Dot (aus dem Nav-Farbsystem,
  `moduleHsl`) + Count. Reihenfolge wie Sidebar. Toggle-Verhalten (zweiter Klick = aus).
- **`shared/SortMenu`** mit Feldern Datum / Priorität × Richtung asc/desc. Priorität-Rank
  urgent>high>normal>low.
- **Lesbare Modul-Badges:** rohe id („contacts") → Nav-Label („Kontakte") via `navItems`-
  Lookup (wiederverwendet vorhandene 4-Sprachen-Labels, konsistent mit der Sidebar —
  finance→„Buchhaltung"). Sonderfall `settings`→neuer Key `notifications.modules.security`
  („Sicherheit") statt „Modul-Einstellungen".
- Gefilterter Leerzustand: `EmptyState` filtered-Variante (`emptyFiltered`/…Description).
- Pagination-Count respektiert den Filter (`Seite 1 (3 gesamt)` bei Verträge-Filter).
- Neue i18n-Keys (single-brace, ×4): `center.allModules`, `center.emptyFiltered{module}`,
  `center.emptyFilteredDescription`, `sort.date`, `sort.priority`, `modules.security`.

**Verify (qa-notif-n2.mjs, :5174, 1440×1000, DE):**
- Filterbar: 10 Chips (Alle Module 13 + 9 Module mit Counts: Kontakte 2, Verträge 3, …),
  Farb-Dots, SortMenu „Datum ↓".
- Badges lesbar: „Kontakte / Aufgaben / Kommunikation / Team / Verträge" (kein raw id).
- Verträge-Chip → genau **3** Cards, „Seite 1 (3 gesamt)".
- Sort Priorität ↓ → erste Card „Sicherheitswarnung" (urgent), danach high → normal.
- **0 Raw-Keys, 0 `{{}}`.** (Console: nur WS `ERR_CONNECTION_REFUSED`, P4 out-of-scope.)
- `EmptyState` filtered-Variante gebaut; visueller Edge-Case-Shot im N-5-Sweep.
