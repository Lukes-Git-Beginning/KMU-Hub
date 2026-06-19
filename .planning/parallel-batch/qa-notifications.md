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

---

## N-3 — Zeilenklick → `shared/DetailModal`  ✅ (2026-06-19)

**Geändert:** `modules/notifications/NotificationCenter.tsx` (In-Card-Expand → zentriertes
Modal), `i18n/messages/{de,en,fr,it}.json` (+10 Keys ins notifications-Cluster).

**Was:**
- Die alte In-Card-Expansion entfernt; Zeilenklick öffnet jetzt **`shared/DetailModal`**
  (zentriert, Gradient-Stripe, sticky Header-Close + intern scrollender Body — wie
  helpdesk/vertraege/automatisierung).
- Ganze Zeile klickbar: `role="button"` + `tabIndex=0` + Enter/Space-Handler +
  `focus-visible`-Ring. Innerer Quick-Mark-Read-Button mit `stopPropagation`.
- Modal-Inhalt: **Akteur + Avatar** (Initialen; System → Megaphone-Icon, getönt),
  relative Zeit, **Volltext-Body**, Meta-Grid (Modul-Badge / Priorität-Pill / Von /
  Erhalten), Header-`PriorityPill`. Footer: **Öffnen** (deep_link, schließt + navigiert),
  **Als gelesen markieren** (nur unread), **Anpinnen/Lösen**, **Ignorieren**.
- `actor_name` ist nicht im openapi-`Notification`-Typ → gezielter `NotificationWithActor`-Cast.
- Neue i18n-Keys (×4): `actions.markRead`, `priority.{low,normal,high,urgent}`,
  `detail.{from,system,module,priority,received}`. Priority-Label dynamisch
  `t('notifications.priority.'+priority)`.

**Verify (qa-notif-n3.mjs, :5174, 1440×1000, DE):**
- Klick → Dialog offen, alle 4 Footer-Buttons, Akteur „Thomas Meier", Meta „Von".
- „Öffnen" → URL `#/pipeline` (deep-link-Navigation greift).
- System-Modal (Sicherheitswarnung): „Dringend"-Pill rot, „Sicherheit"-Badge,
  System-Avatar (Megaphone).
- Pin im Modal → Button flippt auf „Lösen". ESC schließt; Enter auf fokussierter Zeile öffnet.
- **0 Raw-Keys.** (Console: nur WS, P4 out-of-scope.)

---

## N-4 — Sidebar-Badges + Modul-Einstellungen  ✅ (2026-06-19)

**Geändert:** `api/hooks/useNotifications.ts` (`useModuleUnreadCounts`),
`hooks/useFilteredNavItems.ts` (Badge-Injektion), `lib/module-settings.ts`
(`SettingsModuleId` + `'notifications'`), `modules/settings/tabs/NotificationSettingsTab.tsx`
(`embedded`-Prop), **neu** `modules/settings/panels/NotificationsSettingsPanel.tsx`,
`modules/settings/module-settings-registry.tsx` (Eintrag), `i18n/{de,en,fr,it}.json` (+3 Keys).

**Was — Sidebar-Badges:**
- `useModuleUnreadCounts()` gruppiert die ungelesenen Notifications nach `module_id`
  (reuse der Listen-Query, unread-only).
- In `useFilteredNavItems.resolve` injiziert: hat ein Nav-Item Live-Unreads, wird sein
  Badge zum Zahl-Badge (überschreibt den statischen Text-Badge). `'live'`-Indikatoren
  (meetings) bleiben unangetastet; Items ohne Unreads behalten ihren (übersetzten) Badge.
  → ein zentraler Ort, alle Layout-Varianten profitieren, `nav-items`-Config bleibt read-only.

**Was — Modul-Settings-Eintrag:**
- `SettingsModuleId` um `'notifications'` erweitert (wie `dashboard`/`automatisierung` —
  Querschnitts-Surface, nicht im Pricing-`ModuleId`).
- `NotificationSettingsTab` bekommt `embedded`-Prop (lässt eigenen h2/Subtitle + `max-w-2xl`
  weg, wenn in einer Shell-Section).
- `NotificationsSettingsPanel`: `ModuleSettingsShell` (moduleId `notifications`) mit **nur
  personal**-Section, die die **echte** `NotificationSettingsTab embedded` rendert — Präferenzen
  erreichbar gemacht, **nicht dupliziert**. Keine tenant-Section (notifications nicht leadable
  → keine echte tenant-Rolle; tote tenant-Controls vermieden, Plan-`tenant` ist optional).
- Registry-Eintrag `id:'notifications'`, Bell-Icon, `navMatch:['/notifications']` → bei
  `/notifications` automatisch vorausgewählt. ⚠ Main trägt zeitgleich `berichte` in dieselbe
  Registry — finaler Merge behält beide (Branch-Iso).

**Verify (qa-notif-n4.mjs, :5174, 1440×1000, DE):**
- Sidebar: Aufgaben **1**, Kommunikation **1**, Kontakte **2** (neu), Modul-Einstellungen **1**;
  E-Mail 12 / Buchhaltung „Neu" bleiben (keine Notification-Unreads). → „Kontakte 2, Aufgaben 1".
- Settings-Overlay listet „Benachrichtigungen" (Bell, AKTIV bei /notifications); Panel zeigt
  „Meine Benachrichtigungen" + embedded Matrix (Nachrichten/Aufgaben/…), DND, Quiet-Hours —
  kein Doppel-Header.
- **0 Raw-Keys.** (Console: nur WS, P4 out-of-scope.)

---

## N-5 — Sound-Toggle wirksam + Demo-Tiefe-Schlusscheck  ✅ (2026-06-19)

**Geändert:** **neu** `lib/notification-sound.ts`, `modules/notifications/NotificationToast.tsx`
(Chime-Trigger), `modules/settings/tabs/NotificationSettingsTab.tsx` (Ton-Sektion),
`i18n/{de,en,fr,it}.json` (+4 Keys).

**Was — Sound:**
- `lib/notification-sound.ts`: `playNotificationSound()` — dezenter Web-Audio-Chime
  (zwei Töne C6→G6, kein Asset), no-op wenn Audio blockiert/unverfügbar, wirft nie.
  `prefersReducedMotion()`-Helfer.
- `NotificationToast`: spielt **einen** Chime pro neuem Toast-Batch, wenn
  `useNotificationsStore.soundEnabled` **und** nicht `prefers-reduced-motion`.
- `NotificationSettingsTab`: neue „Ton"-Sektion — Switch (`soundEnabled`/`toggleSound`)
  + „Testen"-Button (`playNotificationSound`, disabled wenn Ton aus). Sichtbar in den
  Präferenzen (auch im Modul-Settings-Overlay, da embedded). +4 i18n-Keys ×4.

**Verify (qa-notif-n5.mjs, :5174 — AudioContext-Oszillator-Spy):**
- „Testen" → `__chimes` 0→2 (Chime spielt tatsächlich). Ton-Sektion sichtbar.
- **Sweep DE+EN × 1440/1024:**
  - DE @1440: Center sauber; Empty-Unread nach Mark-all-read = „Alle gelesen" (BellOff) —
    **und Sidebar-Badges fallen live zurück** (Aufgaben 5/Kommunikation 3 statisch, Kontakte
    ohne Badge, Bell-Badge weg) → beweist die Live-Datenbindung aus N-4.
  - EN @1440: voll übersetzt (Notifications / 6 unread / All modules / Security / Mark as read /
    Date / Modul-Badges) — Nav-Label-Wiederverwendung trägt alle 4 Sprachen. Modal: „Open/Mark as read".
  - DE @1024: collapsed Sidebar mit Live-Badge-Dots, Filter-Chips wrappen sauber, kein Layout-Bruch.
- **0 Raw-Keys / 0 `{{}}` / 0 echte Console-Errors** über alle Zustände (WS gefiltert, P4).
- Tote-Buttons-Sweep: Center/Bell/Toast/SettingsTab — alle Buttons wirken (Filter/Sort/Tabs/
  Mark-Read/Modal-Aktionen/Snooze/Sound). Kein Toast-only-Stub mehr.

---

## Definition of Done — notifications review-reif  ✅ 5/5

Alle 5 Punkte verifiziert (Screenshots angesehen). Unread/Priorität/Modul/Deep-Link lebendig,
Filter+Sort, DetailModal, Sidebar-Badges + Settings-Eintrag, Sound wirksam.
**0 Raw-Keys / 0 Doppelklammern / 0 echte Console-Errors** (nur WS = P4 🔒). Jede Phase
ein Commit+Push auf `parallel/notifications`. Bereit für den finalen Merge durchs Main-Terminal
(i18n + `module-settings-registry.tsx`: beide Key-/Eintrags-Blöcke behalten, dann `npm run build`).

**Offen für Luke (out-of-scope):** P4 echtes Realtime/WebSocket (die `ERR_CONNECTION_REFUSED`-
Konsolenmeldungen stammen daher), P5 Multi-Channel/Push, OS-Desktop-Notifications.
