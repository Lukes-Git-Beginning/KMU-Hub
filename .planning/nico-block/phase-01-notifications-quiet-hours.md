# Phase 01 (Pilot) — Notifications: Ruhezeiten & „Nicht stören"-UI

> **Modul:** notifications · **Risiko:** niedrig · **Backend:** vollständig vorhanden (OpenAPI + MSW-Handler + Hooks) — reine FE-Arbeit, kein neuer Handler nötig.
> Dies ist eine **Pilot-Phase**: bewusst gut abgesteckt, damit der Workflow einmal sauber durchläuft.

## Ziel
Im Benachrichtigungs-Center (`/notifications`) einen neuen Einstellungs-Abschnitt **„Ruhezeiten & Nicht stören"** ergänzen:
- **Nicht stören (DND)**: ein Toggle, der DND an/aus schaltet (zeigt aktuellen Status).
- **Ruhezeiten (Quiet Hours)**: Start-/End-Uhrzeit (z.B. 22:00–07:00) aktivierbar, in der keine Benachrichtigungen stören.

Alles ist FE-seitig zu verdrahten — die Hooks + der Demo-Handler liefern schon Daten.

## Ist-Stand (was schon da ist)
- Seite: `desktop/src/renderer/src/modules/notifications/NotificationCenter.tsx`. Darin die Komponente **`PreferencesPanel`** (ca. Zeile 255–350) mit den bestehenden Event-Typ-Checkboxen (in_app / desktop_push je Modul).
- Fertige Hooks in `desktop/src/renderer/src/api/hooks/useNotifications.ts` (ca. Z. 220–266):
  - `useQuietHours()` / `useUpdateQuietHours()`
  - `useDNDStatus()` / `useEnableDND()` / `useDisableDND()`
- Demo-Handler liefert dafür Daten: `desktop/src/renderer/src/mocks/handlers/notifications.ts` (quiet-hours + dnd-Endpunkte vorhanden). **Kein** neuer Handler nötig.

## Muster-Vorlage (im gleichen Stil bauen)
Die bestehenden Preference-Checkboxen in **`PreferencesPanel`** (`NotificationCenter.tsx` ~Z. 292–342) zeigen exakt das Pattern: Hook lesen → Wert anzeigen → bei Änderung Mutation aufrufen. Bau den neuen Abschnitt genauso. Für den Toggle das vorhandene Switch-/Toggle-Muster aus `components/ui/` verwenden (oder den Toggle-Stil, der im Repo schon für Schalter genutzt wird — z.B. in `KommunikationSettingsPanel.tsx` der `role="switch"`-Button).

## Schritte
1. `git pull`. App läuft (`npm run dev`), öffne `/notifications` → „Einstellungen" sichtbar.
2. In `PreferencesPanel` einen neuen Abschnitt unter den Event-Typ-Checkboxen einfügen: Überschrift „Ruhezeiten & Nicht stören".
3. **DND-Toggle**: `useDNDStatus()` lesen → Toggle-Zustand. onChange → `useEnableDND()` bzw. `useDisableDND()`.
4. **Quiet Hours**: `useQuietHours()` lesen → ein Aktiv-Toggle + zwei `<input type="time">` (Start/Ende). Änderungen → `useUpdateQuietHours()` (mit den Feldern, die der Hook erwartet — schau in `useNotifications.ts` + `api/notification-client.ts` nach den genauen Feldnamen).
5. Lade-/Leerzustand sauber behandeln (während Hook lädt: nichts kaputt anzeigen).
6. i18n: neue Texte als Keys unter `notifications.*` in alle 4 Sprachen (de/en/fr/it).
7. Verifizieren (siehe unten), commit, push, „fertig" melden.

## i18n-Keys (neu, Präfix `notifications.`)
Lege passende Keys an, z.B.:
- `notifications.quietHours.title` = „Ruhezeiten & Nicht stören"
- `notifications.quietHours.dnd` = „Nicht stören"
- `notifications.quietHours.dndDesc` = „Alle Benachrichtigungen vorübergehend stummschalten"
- `notifications.quietHours.enable` = „Ruhezeiten aktiv"
- `notifications.quietHours.from` = „Von" · `notifications.quietHours.to` = „Bis"
(EN/FR/IT entsprechend.) Interpolation `{var}`, **nie** `{{var}}`.

## Demo-Handler
**Keiner nötig** — `mocks/handlers/notifications.ts` bedient quiet-hours + dnd bereits.

## Definition-of-Done
- [ ] Neuer Abschnitt „Ruhezeiten & Nicht stören" im PreferencesPanel sichtbar.
- [ ] DND-Toggle schaltet (Status wird gelesen + geändert, keine Konsolenfehler).
- [ ] Quiet-Hours: Aktiv-Toggle + Start/Ende-Zeit, Änderung wird gespeichert (Mutation feuert).
- [ ] Alle Texte i18n in 4 Sprachen, keine Raw-Keys sichtbar.
- [ ] Gescopter Typecheck grün, QA-Script grün, Screenshot @1440px sauber, keine pageErrors.

## QA-Hinweis
Schreib ein `desktop/scripts/qa-notif-quiet-hours.mjs` (kopiere ein bestehendes `qa-*.mjs` als Vorlage): App auf `/#/notifications` öffnen → „Einstellungen"-Tab → prüfen, dass „Ruhezeiten" sichtbar ist, DND-Toggle klickbar ist, Zeit-Inputs vorhanden sind, und der Raw-Key-Scan + pageErrors leer sind.
