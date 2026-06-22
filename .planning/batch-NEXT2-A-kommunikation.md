# Batch-NEXT2 · Paket A (HAUPT-Terminal) — kommunikation/chat → review-reif

> **10-Phasen-Autonom-Batch** ([[feedback_ten_phase_autonomous_batch]]). Fährt das **Haupt-Terminal** (`KMU Hub`, Port 5173).
> **Disjunkt zu Paket B** (formulare, Sub-Terminal). Berührt NUR `modules/kommunikation/**`, `mocks/handlers/chat.ts`, `mocks/data/chat*.ts`, neu `modules/kommunikation/settings/**`, i18n-Keys `kommunikation.*` / `chat.*`.

## Ziel
Das **kommunikation/chat**-Modul (Team-Messaging, 3-Panel da) von „gebaut" auf **review-reif → Nico** bringen: Demo-Tiefe-Standard + DMs + Suche + Edit/Löschen/Lesezeichen + Unified Inbox + Settings-Panel.

## Ist-Stand (verifiziert 2026-06-22, grob)
- `modules/kommunikation/` ~28 Dateien / ~4853 Zeilen, 3-Panel-Chat (Kanal-Liste │ Nachrichten │ Detail/Thread).
- MSW-Handler: `mocks/handlers/chat.ts`.
- **Fehlt:** `modules/kommunikation/settings/` (kein Modul-Settings-Panel — [[feedback_module_settings_per_module]]).
- Tracker P2 (DMs+Suche) / P3 (Unified+Edit/Löschen/Lesezeichen) sind die offenen FE-Stränge; P4 LiveKit + P5 Bots/Webhooks sind 🔒 (Luke) → NICHT in diesem Paket.

## Recherche-Auftrag (VOR dem Bauen — Gate)
1. Wie tief ist chat wirklich? (Kanäle? Threads/Antworten? Reaktionen? send/edit/delete — wirken sie, oder Toast-Stub? MSW stateful oder statisch?)
2. Welche der 10 Phasen unten sind schon (teil-)erledigt → streichen/zusammenfassen.
3. Demo-Tiefe-Audit: ganze Zeile/Nachricht klickbar? tote Buttons? Downloads echt? `chat.ts`-Vertragsumfang?
4. Gibt es schon DMs/Unread-Tracking/Suche im Ansatz, oder Neubau?

## Gate-Fragen an Darien (gebündelt stellen, dann bauen)
- **DM-Umfang:** nur 1:1, oder auch Gruppen-DMs?
- **Unified Inbox:** ein zusammengeführter „Alle ungelesenen"-Eingang über Kanäle + DMs — als Default-Fokus?
- **Threads:** Antworten als Inline-Thread (wie Slack) oder flach mit Zitat?
- **Settings-Scope:** personal (Status, Benachrichtigungen, Lese-Bestätigungen) vs. tenant (Kanal-Erstellungs-Policy, Nachrichten-Aufbewahrung)?

## Phasen (FINALISIERT nach Gate 2026-06-22)

> **Gate-Ergebnis:** Modul ist weiter als gedacht — Settings-Panel, 1:1-DMs, Suche, Reaktionen, Inline-Threads, @Mentions, Datei-Upload existieren bereits funktional. **Kern-Mangel = MSW NICHT stateful.** Darien-Entscheidungen: (1) Fokus = Demo-Tiefe/Stateful statt Neubau; (2) beide Bereiche (Team-Chat `modules/chat` + Posteingang `modules/kommunikation`); (3) Gruppenchats im **internen** Team-Chat bauen (extern = Anbindung); (4) `/poll`+`/reminder` echt machen, `/giphy` bleibt Stub.

- [ ] **KO-1** **Fundament: `chat-store.ts`** (stateful, analog `email-store.ts`) — send/edit/delete/read/reactions persistieren In-Memory; `chat.ts`-Handler komplett auf Store umstellen (GET liest aus Store). Kern-Demo-Tiefe.
- [ ] **KO-2** **Gruppen-DMs (intern):** Datenmodell `participants[]`, Multi-Personen-Picker, Gruppen-DM-Header/Avatare, stateful get-or-create.
- [ ] **KO-3** **Threads wirksam + Seeds:** Thread-Replies im Store (senden persistiert), Seed-Replies für Demo-Tiefe (Panel nicht mehr leer).
- [ ] **KO-4** **Bookmarks/Lesezeichen:** stateful Toggle an Nachricht, Bookmark-Panel (gespeicherte Nachrichten), klickbar→Jump.
- [ ] **KO-5** **Datei-Download + Anhang-Tiefe:** echter Blob-Download aus `FileAttachmentCard`, Anhang-Seeds breiter, Bild-Vorschau.
- [ ] **KO-6** **Suche klickbar (Jump-to-Result, scroll+highlight) + Unread persistiert** (read im Store, Badge bleibt weg).
- [ ] **KO-7** **Unified Inbox „Alle ungelesenen"** über Kanäle + DMs (fokussiertes Abarbeiten im Team-Chat).
- [ ] **KO-8** **Posteingang-Tiefe:** `/poll`+`/reminder` echt (stateful, im Thread sichtbar), `/giphy` sauber als Stub markiert, ganze Zeile klickbar, Stars persistieren, tote Buttons fixen.
- [ ] **KO-9** **Kanal-Verwaltung wirksam:** create/rename/leave/join stateful + sichtbar, privat/öffentlich, `ChannelSettingsDialog` wirkt; Settings-Panel-Feinschliff.
- [ ] **KO-10** **i18n ×4** (`kommunikation.*`/`chat.*`, `{var}`, ICU-Plural) + Demo-Tiefe-Schlusscheck + Playwright-QA + Screenshots WIRKLICH ansehen.

## Disjunktheit / Konflikt-Vermeidung
- **NUR** `modules/kommunikation/**`, `mocks/handlers/chat.ts`, `mocks/data/chat*.ts`, neu `modules/kommunikation/settings/**`.
- **Finger weg von** `modules/formulare/**`, `mocks/handlers/formulare.ts` (= Paket B).
- i18n (`de/en/fr/it.json`): nur `kommunikation.*`/`chat.*`-Keys → rebased additiv mit Paket B.

## Build-+-Verify-Standard (pro Phase)
bauen → i18n ×4 → gescopter tsc (foreground, echter Exit, NIE `| tail`) → `eslint src/ --quiet` ([[feedback_lint_before_push]]) → Playwright-Screenshot-QA + PNGs ansehen → ein Commit (explizite Pfade, `mocks/data/` braucht ggf. `git add -f`) → push (`pull --rebase`). Scoped tsconfig anlegen (`tsconfig.kommunikationcheck.json`). Latenz: vorbestehende latente tsc-Fehler ignorieren.
