# Review-Paket · mails → review-reif (Paket A, Haupt-Terminal)

> 10-Phasen-Autonom-Batch abgeschlossen 2026-06-22. Modul **mails** von „gebaut" auf **review-reif → Nico** gebracht.
> Branch `main`, Commits `e2213408` … `86de8f53`. Alle Phasen: gescopter tsc grün, `eslint` grün, Playwright-Screenshot-QA verifiziert.

## Was gebaut wurde (MA-1 … MA-10)

| Phase | Commit | Inhalt | QA-Skript |
|---|---|---|---|
| **MA-1** Stateful-MSW-Fundament | `e2213408` | In-Memory-Email-Store: read/star/move/delete **persistieren** jetzt; Daten-Shape normalisiert (leere Previews, generische Icons, „NaN KB", tote Reply-Actions gefixt); echte Downloads (valides PDF) / Print / `.eml`-Export statt Toast-Stubs | `qa-mails-ma1.mjs` |
| **MA-2** Thread-Lese-Ansicht | `d74d9e74` | Konversations-Ansicht (neueste offen, ältere eingeklappt), Zitat-Verlauf hinter „···"-Toggle, **Inline-Bild** (Base64, `sanitizeMailBody`), Anhänge je Nachricht, Inbox-Liste nach Thread gruppiert + Zähler-Badge | `qa-mails-ma2.mjs` |
| **MA-3** Multi-Account + Unified | `b38b4ea8` | 2 zusätzliche Konten (info@/support@) mit eigenen Ordnern + Mails, Account-Switcher, „Alle Eingänge"-Unified-Ansicht mit Konto-Chips | `qa-mails-ma3.mjs` |
| **MA-4** Filter + Sortierung | `5a4b55ce` | Filter-Chips (Alle/Ungelesen/Markiert) + `shared/SortMenu` (Feld + Richtung) | `qa-mails-ma4.mjs` |
| **MA-5** Vorlagen-CRUD | `f4beb558` | Built-in + **eigene** Vorlagen (anlegen/bearbeiten/löschen, persistiert), Platzhalter-Ausfüllen mit Live-Vorschau beim Einfügen | `qa-mails-ma5.mjs` |
| **MA-6** Labels + Regeln | `d07551d9` | Labels (anlegen mit Farbe/löschen, Chips auf Zeilen, Zuweisung im Aktionsmenü, cross-folder Label-View), Wenn-Dann-Regeln (Dialog + „Regeln anwenden" stateful) | `qa-mails-ma6.mjs` |
| **MA-7** CRM-Integration | `c8286781` | CRM-Panel: Kontakt verknüpfen (Picker, stateful) + zum Kontakt springen, **Deal aus Mail** + **Aktivität protokollieren** (echte Erstellung via shared CRM-Hooks → erscheint in /crm) | `qa-mails-ma7.mjs` |
| **MA-8** Bulk + Shortcuts | `9e9f4e4d` | Mehrfachauswahl (Checkboxen + Bulk-Bar: gelesen/markieren/archivieren/Spam/löschen), Spam-Ordner ergänzt, Gmail-Shortcuts (j/k/e/r/u/s/#/x/Esc) | `qa-mails-ma8.mjs` |
| **MA-9** Settings-Panel | `33019d04` | `MailsSettingsPanel` auf `ModuleSettingsShell` — personal (Standard-Konto, Signatur, Benachrichtigungen, Konversationsansicht) + tenant (Server, Auto-Antwort, DSGVO-Badges, externe Bilder); altes `MailSettingsTab` abgelöst | `qa-mails-ma9.mjs` |
| **MA-10** i18n ×4 + Schluss-QA | `86de8f53` | 226 `mails.*`-Keys in de/en/fr/it (0 fehlend, 0 Doppelklammern, 0 `_one/_other`), Demo-Tiefe-Audit (keine Stubs), holistische QA (0 Raw-Keys, 0 Errors über 5 Views) | `qa-mails-ma10-holistic.mjs` |

## Gate-Entscheidungen (von Darien bestätigt)
- **Reading-Pane behalten** (3-Pane, nicht DetailModal — korrekt für Mail-Triage)
- **Mehrere Konten + Unified Inbox**
- **CRM voll** (Verknüpfen + Deal + Aktivität)
- **Echte Threads**

## Wie reviewen
Pro Phase: `cd desktop && node scripts/qa-mails-ma<N>.mjs` (Dev-Server auf 5173 muss laufen: `npm run dev`). Screenshots landen in `desktop/.qa-screenshots/mails-ma<N>/`.
Holistisch: `node scripts/qa-mails-ma10-holistic.mjs`.

## Architektur-Notizen für Nico
- **Store:** `mocks/data/email-store.ts` ist jetzt ein vollständiges stateful MSW-Backend (messages/folders/accounts/labels/rules/contactLinks, CRUD + bulk + applyRules). `mocks/data/emails.ts` = Seed.
- **Handler:** `mocks/handlers/email.ts` delegiert komplett an den Store; deckt alle Client-Routen ab.
- **Neue Komponenten:** `ThreadView`, `AccountSwitcher`, `RulesDialog`, `MailCrmPanel`, `settings/MailsSettingsPanel`, `lib/mail-export.ts`.
- **Neue Stores:** `stores/mailTemplates.ts`, `stores/mailPrefs.ts`.
- **Sanitizer:** `sanitizeMailBody` erlaubt nur Raster-`data:image` (sicher) — global bleibt strikt.
- **Scoped tsc:** `tsconfig.mailscheck.json`.

## Offen / mögliche Findings (für Darien-Review)
- Nav-Rail-Badge „E-Mail 12" ist noch ein Seed-Wert (nav-items.ts), nicht aus den Ordner-Counts abgeleitet — bewusst out-of-scope gelassen (shared Layout).
- Deal aus Mail: legt Deal mit Wert 0 an (im CRM editierbar) — kein Wert-Dialog in mails.
- Tippen-Lag im TipTap-Editor ([[project_editor_typing_lag]]) betrifft Compose ebenfalls — separat eingeplant.
