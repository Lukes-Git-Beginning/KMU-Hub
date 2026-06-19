# Review — notifications

> **Status:** review-reif (N-1…N-5, Sub-Terminal `parallel/notifications`, gemergt nach `main`).
> **Lane:** FE/UX-Review (mock-first). Echtes Realtime/WebSocket (P4) + Multi-Channel/Push (P5) = Lukes Lane, **nicht** Teil dieses Reviews.
> **Screens:** `/#/notifications` (Center) · Topbar-Glocke · Sidebar-Badges · `/settings` Benachrichtigungen (Matrix/DND/Ruhezeiten).
> **Kontext:** Vor dem Batch war das Center durch einen Schema-Bug halb-tot (alle als gelesen, Deep-Links/Preferences tot). Jetzt lebendig.

## Was gebaut wurde (Definition of Done)
- [x] **N-1 — Schema-Fix + Seed-Upgrade**: MSW-Seeds auf das Center-/Hook-Schema gebracht (`is_read`, `priority`, `module_id`, `deep_link`). Folge: **Unread-Count sichtbar** (6 ungelesen), Prioritätsfarben, Modul-Badges, „Öffnen"→Ziel-Route, Preferences-Checkboxen speichern.
- [x] **N-2 — Filter + Sortierung**: Modul-Filter-Chips (Alle Module / Projekte / Aufgaben / Kommunikation / Kontakte / Team / Dokumente / Buchhaltung / Verträge / Sicherheit) + `shared/SortMenu`.
- [x] **N-3 — Detail = `shared/DetailModal`**: Zeilenklick → zentriertes Modal mit allen Feldern (Akteur, Modul, Priorität, Volltext, Zeitstempel) + Aktionen Öffnen / Als gelesen markieren / Anpinnen / Ignorieren. Ganze Zeile klickbar, sticky Close.
- [x] **N-4 — Sidebar-Badges + Einstellungen**: Unread-Counts **pro Modul** in der Sidebar (z. B. Aufgaben 1, Kontakte 2). Modul-Settings-Eintrag „Benachrichtigungen" (verweist auf die bestehenden Präferenzen: Modul×Kanal-Matrix, DND, Ruhezeiten).
- [x] **N-5 — Sound + Schlusscheck**: Sound-Toggle hörbar gemacht (`notification-sound.ts`, respektiert Stumm/`prefers-reduced-motion`). Demo-Tiefe-Sweep, keine toten Buttons.

## Worauf besonders achten
- Center: Unread-Count oben + „Ungelesen (6)"-Tab; ungelesene visuell markiert?
- Modul-Filter-Chips reduzieren die Liste korrekt; SortMenu Feld + Richtung?
- Zeilenklick → DetailModal zentriert mit allen Infos; „Öffnen" navigiert zum Datensatz; Mark-Read/Pin/Ignorieren wirken.
- Sidebar zeigt Modul-Unread-Badges; Topbar-Glocke konsistent.
- Modul-Einstellungen → „Benachrichtigungen" → Matrix/DND/Ruhezeiten erreichbar.
- EN umschalten über Center + Einstellungen (keine Raw-Keys, keine deutschen Reste).

## Out of scope (kein Mangel)
- Echtes Realtime/WebSocket (P4 🔒), Multi-Channel E-Mail-Digest/SMS/PWA-Push (P5 🔒), OS-/Desktop-Notifications.
- Notification-Inhalte sind bewusst deutsche DACH-Demo-Daten.
- Bekannte app-globale Demo-Lücke: `GET /feature-flags` 1× `ERR_CONNECTION_REFUSED` beim Start (Feature-Flag-System per ADR OFF) — überall, nicht notifications-spezifisch.

## Befunde
> Format pro Zeile: **Schweregrad** (P0 / P1 / P2) · **was** · **wo** (Datei/Screen) · **Repro**.

_(noch keine — von Nico zu füllen)_
