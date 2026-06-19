# Review — helpdesk

> **Status:** review-reif (Demo-tief H-1…H-8, gemergt, `a221278d`).
> **Lane:** FE/UX-Review (mock-first, FE auf Zustand-Store + persist). Echtes Ticket-Backend, Mail→Ticket, CRM-Kontakt-Lookup = Lukes Lane.
> **Screens:** `/#/helpdesk` — Tabs Tickets · Wissensdatenbank · Statistik.

## Was gebaut wurde (Definition of Done)
- [x] **H-1 — Store-Fundament**: Store auf persist umgestellt, sinnvolle Demo-Threads für **alle 15 Tickets**; Actions für Anlegen/Status/Zuweisen/Eskalieren/Mergen/Reply/CSAT/Canned-CRUD/Geschäftszeiten/Routing.
- [x] **H-2 — Ticket-Detail = `DetailModal`** (war Slide-over): zentriert, sticky Header (Betreff/Nr/Status/Prio), sticky Footer mit Reply-Bereich, Body scrollt intern. **Tabellenzeile voll tastatur-zugänglich** (Enter/Space, Focus-Ring). Escape schließt.
- [x] **H-3 — Aktionen verkabelt**: Neues Ticket erscheint sofort oben (Auto-Nr `HD-2026-NNNN`); Reply/interne Notiz hängt sichtbar am Thread (echter Autor); Status ändert Badge in Tabelle **und** Header; CSAT an Store gebunden. **Überlebt Reload.**
- [x] **H-4 — Zuweisen / Eskalieren / Mergen**: neue Aktionsleiste im Modal. Zuweisen → Agent-Wechsel + Thread-Notiz; Eskalieren → Priorität +1 (bis kritisch, dann disabled); Mergen → Quell-Thread ans Ziel, Quelle „Geschlossen", Modal schließt.
- [x] **H-5 — Canned Responses CRUD**: Anlegen/Bearbeiten/Löschen wirken + persistieren (war nur Toast).
- [x] **H-6 — Settings-Panel** (`ModuleSettingsShell`, Eintrag „Helpdesk"): **persönlich** (Start-Tab + Standard-Statusfilter) + **tenant** (Geschäftszeiten + Routing-Regeln, admin-gated). Header-Buttons entfernt, Tab/Filter aus Prefs initialisiert.
- [x] **H-7 — Sortierung + echte SLA**: `SortMenu` (Erstellt / Priorität / Status / SLA-Restzeit, asc/desc). SLA-Uhr auf echte Zeit umgestellt (kein eingefrorenes „122d" mehr) — Mix aus überfällig / bald fällig / komfortabel.
- [x] **H-8 — i18n SLA + KB-Speichern**: SLA-Labels über `t()` (EN sauber, keine deutschen Reste); KB-Artikel-Bearbeitung stateful + sanitisiert, überlebt Reload.

## Worauf besonders achten
- Ticket anlegen / antworten / Status ändern → Reload → bleibt alles erhalten?
- Sortierung Feld + Richtung beidseitig wirksam?
- EN-Umschalten: SLA-Badges (z. B. „4h overdue" / „1d 3h left") komplett englisch?
- Settings-Panel: persönlich vs. tenant korrekt getrennt, Routing/Geschäftszeiten speichern echt?

## Out of scope (kein Mangel)
- TanStack-/MSW-Migration des Stores, echtes Ticket-Backend, Mail→Ticket, CRM-Kontakt-Lookup. Ticket-Inhalte/Threads sind bewusst deutsche DACH-Demo-Daten.

## Befunde
> Format pro Zeile: **Schweregrad** (P0 / P1 / P2) · **was** · **wo** (Datei/Screen) · **Repro**.

_(noch keine — von Nico zu füllen)_
