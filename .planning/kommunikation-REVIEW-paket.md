# Review-Paket — kommunikation/chat (Paket A, 2026-06-22)

> **Status: review-reif.** 10-Phasen-Autonom-Batch (KO-1…KO-10) im Haupt-Terminal. Disjunkt zu Paket B (formulare, Sub-Terminal). Alle Phasen gescopt-typecheck + voller `eslint src/` grün + Playwright-Screenshot-verifiziert. **Findings bitte als Fix-Phasen in `MASTER-TRACKER.md` → dann Nico.**

## Was das Modul jetzt kann (vorher → nachher)

Das Modul hat **zwei Bereiche** (Umschalter oben, `?bereich=team|posteingang`):
- **Team-Chat** (`modules/chat/`) — internes Slack/Teams
- **Posteingang** (`modules/kommunikation/`) — Omnichannel-Kundeninbox

**Kern-Fund beim Gate:** Das Modul war funktional gebaut, aber der MSW-Mock war **nicht stateful** (alles sprang nach Kanalwechsel zurück) und alle Nachrichten zeigten **„Unbekannt"** als Absender (Seeds nutzten falsches Feld-Schema). Beides gefixt in KO-1.

| Phase | Was | Wie prüfen |
|---|---|---|
| **KO-1** | **Stateful chat-store** (`mocks/data/chat-store.ts`, analog email-store): senden/bearbeiten/löschen/gelesen/Reaktionen persistieren. + Seed-Schema auf `created_by`/`sender_first_name` normalisiert → echte Namen + Edit/Löschen für eigene Nachrichten. | Nachricht senden → Kanal wechseln → zurück: bleibt. Eigene Nachricht hovern → Stift/Papierkorb da. |
| **KO-2** | **Gruppen-DMs (intern):** „Neue Nachricht"-Button (Stift-Icon) in DM-Sektion → Multi-Personen-Picker. 1 Person = 1:1, mehrere = Gruppe (Users-Icon). | DM-Sektion → Stift → 2 Personen wählen → „Gruppe starten". |
| **KO-3** | **Threads echt:** jede Nachricht mit Antwort-Zähler hat jetzt echte Seed-Antworten (vorher leer). Antworten persistieren. | Nachricht mit „N Antworten" klicken → Thread gefüllt → eigene Antwort senden. |
| **KO-4** | **Lesezeichen:** Bookmark-Button an jeder Nachricht + Lesezeichen-Panel (Sidebar) mit Sprung zum Kanal. | Nachricht hovern → Bookmark → Sidebar „Lesezeichen". |
| **KO-5** | **Datei-Download + Bild-Vorschau:** echter Blob-Download (Demo-Inhalt) aus Anhang-Karten; Server-Bilder bekommen Platzhalter-Vorschau. design-Kanal hat Bild+PDF-Anhang. | #design erste Nachricht → Download-Icon klicken (Datei landet im Download-Ordner). |
| **KO-6** | **Suche klickbar:** Suchergebnisse springen zur Nachricht (scroll + gelber Flash). Unread-Badge bleibt nach Lesen weg. | Such-Icon → „Meeting" → Ergebnis klicken. |
| **KO-7** | **Unified Inbox „Alle ungelesenen":** ein Eingang über Kanäle + DMs mit Vorschau, Sprung, „Alle gelesen". | Sidebar „Alle ungelesenen" (Badge 8). |
| **KO-8** | **Posteingang-Slash-Commands echt:** `/umfrage` → abstimmbare Umfrage im Verlauf, `/erinnerung` → Erinnerung mit Fälligkeit. `/giphy` = „kommt bald"-Stub. | Posteingang → Konversation → `/` tippen → /umfrage. |
| **KO-9** | **Kanal-Verwaltung:** „Kanal bearbeiten" (owner/admin) im Kanal-Menü → Name/Beschreibung/Sichtbarkeit. Erstellen/Verlassen wirken (stateful). | #allgemein → ⋮ → „Kanal bearbeiten". |
| **KO-10** | i18n ×4 vollständig, Demo-Tiefe-Schlusscheck, holistische QA (0 Raw-Keys über alle Ansichten). | — |

## QA-Skripte (alle grün)
`desktop/scripts/qa-komm-ko{1..9}.mjs` + `qa-komm-ko10-holistic.mjs`. Lauf: Dev-Server auf 5173 (`npm run dev`), dann `node scripts/qa-komm-ko<n>.mjs`.

## Scoped typecheck
`tsconfig.kommunikationcheck.json` (deckt modules/chat, modules/kommunikation, chat-Hooks, chat-Mocks ab; `__tests__` exkludiert wegen vorbestehender jest-dom-Typdefs).

## Bekannte Grenzen / bewusst offen
- **Kein echtes Backend** — alles Mock-first (stateful in-memory chat-store + persistierter inbox-thread-Store). FE→BE-Wiring kommt gebündelt nach allen Batches ([[project_fe_be_wiring_phase]]).
- **`/giphy`** bewusst Stub (braucht externe API).
- **LiveKit-Calls / Bots / Webhooks** = 🔒 Luke, NICHT in diesem Paket (P4/P5).
- Reaktions-Button-Hover war im QA schwer zu selektieren (opacity-Hover-Bar) — Funktion ist intakt, nur QA-Selektor-Artefakt.

## Disjunktheit
Berührt: `modules/chat/**`, `modules/kommunikation/**`, `mocks/data/chat-store.ts`+`chat-data.ts`, `mocks/handlers/chat.ts`, chat-Hooks (`useChannels/useMessages/useReactions/useBookmarks/useUnreadInbox`), `stores/inboxThread.ts`, `types/communication.ts`, i18n `chat.*`/`kommunikation.*`. Keine Überschneidung mit formulare (Paket B).
