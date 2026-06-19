# Sub-Terminal — notifications Schema-Fix + Demo-Tiefe (N-1 … N-5)

> **Du bist das Sub-Terminal im Klon `…/KMU-Hub-review`, Dev-Port 5174.** Lies zuerst `.planning/parallel-batch/README.md` (Lane-Regeln, Build-+-Verify-Standard, Gates). Du baust **nur notifications**. berichte gehört dem Main-Terminal — fass es nicht an.
>
> **Selbst-enthaltend:** Alle Klärungen sind beantwortet (siehe README „Entscheidungen"). Bau die 5 Punkte ohne Rückfragen ab. Melde Darien nach jedem Punkt „N-x fertig, n/5".

## Ausgangslage (Ist-Abgleich 2026-06-19)
notifications ist **strukturell solide, aber durch einen Schema-Bug im Demo halb-tot.** Vorhanden + fertig: Notification-Center (`/notifications` — Liste, Tab All/Unread, Mark-All-Read, Pagination, In-Card-Expand mit Pin/Dismiss/Open), Topbar-Bell (`NotificationBell.tsx`, Popover + Unread-Badge, voll verkabelt), und in `/settings` der `NotificationSettingsTab` (Modul×Kanal-Matrix, DND, Stummgeschaltete Ressourcen, Quiet-Hours — alle stateful via MSW, **fertig**).

**Zwei-Quellen-System (wichtig):**
- `useNotificationsStore` (Zustand, `stores/notifications.ts`) → nur transiente **Toasts** (`NotificationToast.tsx`), 10 Hardcode-Mocks.
- `useNotifications` (TanStack Query + MSW, `mocks/handlers/notifications.ts`) → die **Center- + Bell-Quelle**, 13 Demo-Notifications.

**Der Kern-Bug:** Der MSW-Handler liefert `read` (Boolean), das Center erwartet `is_read` → **alle Notifications werden als gelesen gerendert** (kein blauer Rand, Unread-Count in der Bell = 0). Außerdem fehlen `priority`, `module_id`, `deep_link` in den Seeds → Prioritätsfarbe immer grau, Modul-Badge leer, „Öffnen" navigiert ins Leere (`/`). Das Preferences-Panel im Center nutzt ein anderes eventType-Schema als der Hook (`event_key`/`display_name` fehlen) → Checkboxen speichern lautlos nicht.

i18n: alle `notifications.*`-Keys vorhanden, `notifications.center.unread` korrekt als ICU-Plural. Kein `{{}}`-Bug. Modul-Settings: **kein** notifications-Eintrag in `module-settings-registry.tsx`.

## Branch-Setup (einmalig, ZUERST — Sicherung gegen main-Konflikte)
Bau **NICHT** direct-to-main. Erstelle einmal deinen Isolations-Branch: `git checkout main && git pull`, dann `git checkout -b parallel/notifications`. Alle N-Punkte committest + pushst du auf **diesen** Branch (`git push -u origin parallel/notifications`). **Kein** `git pull --rebase` von main nötig — dein Branch ist isoliert. Das Main-Terminal merged `parallel/notifications` am Ende kontrolliert.

## Workflow pro Punkt
bauen → i18n ×4 (`{var}`, ICU-Plural) → MSW/Demo-Daten → Compile-Gate (`npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error`, **nie `| tail`**) → Playwright-Screenshot-QA gegen **:5174** + **Bilder ansehen** → iterieren → commit + push auf `parallel/notifications` → Eintrag in `qa-notifications.md`.

---

### N-1 — Schema-Fix + MSW-Seed-Upgrade (Kern, Blocker)  `[FOUNDATION]`
**Ist:** `mocks/handlers/notifications.ts` ~L13–173: 13 Seeds mit `read` (Boolean), **ohne** `priority`, `module_id`, `deep_link`. Preferences-eventTypes liefern `{ type, label, category }`, der Hook erwartet `{ event_key, display_name, description, default_in_app, default_desktop_push, module_id }`. `NotificationCenter.tsx` ~L41/76/265 liest `is_read`.
**Soll:**
- Seeds auf das vom Center/Hook erwartete Schema bringen: `is_read` (statt/zusätzlich zu `read`), `priority` (low/normal/high/critical gemischt), `module_id` (crm/tasks/hr/finance/helpdesk gemischt), `deep_link` (auf echte App-Routen, z. B. `/finanzen`, `/work/my-tasks`, `/helpdesk`).
- 6 davon ungelesen lassen → Unread-State + Bell-Count werden sichtbar.
- Preferences-eventTypes auf das Hook-Schema umstellen (`event_key`, `display_name`, `description`, `default_in_app`, `default_desktop_push`, `module_id`).
**Verify:** Bell zeigt Unread-Count > 0; Center zeigt ungelesene fett/markiert + Prioritätsfarbe + Modul-Badge; „Öffnen" navigiert zur Ziel-Route; Preferences-Checkboxen speichern (überleben Reload).

### N-2 — Modul-Filter + Sortierung
**Ist:** Center hat nur All/Unread-Tabs, keine Sortierung, kein Modul-Filter.
**Soll:** Filter nach Modul (crm/tasks/hr/finance/helpdesk — Dropdown oder Chips) + `shared/SortMenu` (Felder Datum / Priorität, Richtung asc/desc). Beides auf der bestehenden Center-Liste.
**Verify:** Modul-Filter reduziert die Liste korrekt; Sort nach Priorität/Datum reordert beidseitig; leerer Filter-Zustand sauber.

### N-3 — Detail-Standard: Zeilenklick → `shared/DetailModal`  `[PATTERN]`
**Ist:** Zeilenklick togglet nur die In-Card-Expansion (Karte klappt auf). Kein zentriertes Fenster, kein vollständiger Detail-View.
**Soll:** Zeilenklick öffnet `shared/DetailModal` (zentriert, sticky Close — wie helpdesk/vertraege/automatisierung) mit allen Feldern: Akteur/Avatar, Modul-Badge, Priorität, Zeitstempel, Volltext-Body, **Deep-Link-Button** („Öffnen" → Ziel-Route), Mark-Read, Pin, Dismiss. Ganze Zeile klickbar (`role=button` + Enter/Space), innere Buttons `stopPropagation`.
**Verify:** Zeile klicken → zentriertes Modal mit allen Infos + Aktionen; „Öffnen" navigiert; Mark-Read/Pin/Dismiss wirken; Close sticky beim Scrollen.

### N-4 — Sidebar-Badges + Modul-Einstellungen
**Soll:**
- **Sidebar-Badges:** Unread-Counts **pro Modul** in der Sidebar surfacen (z. B. CRM 2, Aufgaben 1). Minimal-invasiv am bestehenden Sidebar-/Nav-Render — kein Umbau. Quelle: die `module_id`-gruppierten Unread-Counts aus dem Hook.
- **Modul-Settings-Eintrag:** `id: 'notifications'` in `module-settings-registry.tsx` mit `ModuleSettingsShell`:
  - **personal:** Verweis/Verlinkung auf die vorhandenen Notification-Präferenzen (die `NotificationSettingsTab`-Felder — Matrix/DND/Quiet-Hours) — nicht duplizieren, sondern erreichbar machen.
  - **tenant** (optional, falls sinnvoll): workspace-weite Default-Kanäle.
  - ⚠ Koordination: Main trägt zeitgleich `id: 'berichte'` in **dieselbe** Registry ein — du baust auf `parallel/notifications`, finaler Merge behält beide (siehe README Regel 2).
**Verify:** Sidebar zeigt Modul-Badges mit korrekten Counts; Einstellungs-Fenster zeigt „Benachrichtigungen"-Eintrag, Präferenzen erreichbar.

### N-5 — Sound-Toggle wirksam + Demo-Tiefe-Schlusscheck
**Ist:** `stores/notifications.ts` hat `soundEnabled`, aber **kein** Audio-Playback (leeres Feature).
**Soll:**
- Sound-Toggle wirksam machen: bei neuer Toast-Notification einen dezenten Ton abspielen, wenn `soundEnabled` (Web-Audio oder kurzes Audio-Asset; bei `prefers-reduced-motion`/stumm respektieren). Toggle sichtbar in den Präferenzen.
- Sweep: keine toten Buttons/Toast-only-Stubs mehr in `modules/notifications/`; alle Zustände (leer / gemischt gelesen-ungelesen / DND aktiv / Preferences offen) Screenshot-QA @1440+1024, DE+EN.
**Verify:** Sound spielt bei neuer Notification (wenn aktiv); 0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors über alle Zustände, DE+EN.

---

## Out of scope (NICHT in diesem Batch — 🔒 Luke)
- **P4** echtes Realtime/WebSocket (der `useNotificationWebSocket`-Code existiert, verbindet aber nur gegen echtes Backend) — Demo bleibt MSW.
- **P5** Multi-Channel (E-Mail-Digest, SMS, PWA-Push).
- OS-/Desktop-Notifications (Electron-IPC) — nur wenn trivial erreichbar, sonst Luke.

## Definition of Done (notifications review-reif)
Alle 5 Punkte verifiziert (Screenshots angesehen), Unread/Priorität/Modul/Deep-Link lebendig, 0 Raw-Keys / 0 Doppelklammern / 0 Console-Errors, jede Phase ein Commit+Push auf `parallel/notifications` (rebase nicht nötig, Branch isoliert), `qa-notifications.md` gepflegt. Dann Darien Bescheid: „notifications 5/5 fertig".
