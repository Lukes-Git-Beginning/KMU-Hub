# Phase 01 (Pilot) — Notifications-Einstellungen im Demo-Modus lauffähig machen

> **Modul:** notifications · **Risiko:** niedrig · **Art:** Demo-Mock-Handler-Fix (kein neues UI).
> **Wichtig (korrigiert 2026-06-08):** Die UI existiert bereits — `desktop/src/renderer/src/modules/settings/tabs/NotificationSettingsTab.tsx` (erreichbar via Einstellungen → „Benachrichtigungen", eingebunden in `SettingsPage.tsx`). Sie hat DND-Toggle, Quiet Hours (Aktiv-Toggle + Start/Ende + Wochentage + Save) und Muted-Resources. **Nicht** nochmal bauen, **kein** zweiter Einstiegspunkt (kein Duplikat).
> Das Problem: Im **Demo-Modus** funktioniert dieses Tab nicht — die Mock-Handler liefern ein falsches Schema und es fehlen die Schreib-Endpunkte. Genau das reparierst du.

## Ziel
Den bestehenden `NotificationSettingsTab` im **Demo-Modus** voll lauffähig machen: Quiet-Hours- + DND-Daten laden korrekt, und Speichern/Umschalten funktioniert ohne Netzwerkfehler.

## Der konkrete Bug (von Nicos Discovery, bitte selbst gegenprüfen)
- **Schema-Mismatch:**
  - Handler `mocks/handlers/notifications.ts` (~Z. 219–232) liefert `{enabled, start, end, timezone, override_urgent}` bzw. für DND `{enabled, until}`.
  - Client `api/notification-client.ts` (~Z. 13–24) erwartet aber `{quiet_hours: {start_time, end_time, is_active, days, timezone}}` bzw. für DND `{is_active, expires_at}`.
- **Fehlende Endpunkte:** Für quiet-hours und dnd gibt es nur GET — **PUT/POST/DELETE fehlen** → die Mutations (`useUpdateQuietHours`, `useEnableDND`, `useDisableDND`, mute/unmute) werfen im Demo Network-Errors.

## Aufgabe
1. **Wahrheit feststellen:** Lies `api/notification-client.ts` (die genauen Request-/Response-Typen) + die zugehörigen Hooks in `api/hooks/useNotifications.ts` + die OpenAPI-Typen in `api/types.ts` (such nach `notifications/quiet-hours`, `notifications/dnd`, `notifications/mutes`). **Die Client-/OpenAPI-Typen sind die Wahrheit** — der Handler muss sich danach richten, nicht umgekehrt.
2. **GET-Handler korrigieren:** Quiet-Hours + DND so zurückgeben, wie der Client sie erwartet (Feldnamen exakt: `start_time`/`end_time`/`is_active`/`days`/`timezone` bzw. `is_active`/`expires_at`).
3. **Schreib-Endpunkte ergänzen:** PUT/POST/DELETE (je nachdem was die Hooks aufrufen — schau in `notification-client.ts` welche Methode + Pfad) für quiet-hours, dnd (enable/disable) und mutes. **Stateful im Handler** (ein Modul-lokales Objekt im Speicher), damit ein Save sichtbar erhalten bleibt, solange die App läuft — Muster: wie andere Handler mit veränderbarem State in `mocks/handlers/` arbeiten.
4. **Keine UI-Änderung** nötig, außer es fällt ein echter Bug im Tab auf (dann minimal + dokumentieren).

## Muster-Vorlage
Andere Handler in `mocks/handlers/` zeigen den MSW-Stil (`http.get/put/post/delete`, `HttpResponse.json`). Ein Beispiel für einen Handler, der geschriebene Werte im Speicher hält + zurückgibt, findest du in den vorhandenen Handlern (z.B. inbox/chat) — gleiches Muster für quiet-hours/dnd.

## i18n
Keine neuen Keys nötig (UI existiert, Keys unter `settings.notifications.*` sind da). Falls du doch Text ergänzt → 4 Sprachen, `{var}`.

## Definition-of-Done
- [ ] Einstellungen → „Benachrichtigungen" lädt im Demo **echte** Quiet-Hours- + DND-Werte (keine leeren/undefined-Felder).
- [ ] DND an/aus schalten funktioniert ohne Konsolenfehler (Status ändert sich sichtbar).
- [ ] Quiet Hours: Zeit/Aktiv/Wochentage ändern + **Speichern** funktioniert ohne Netzwerkfehler; nach Speichern bleibt der Wert erhalten (stateful Handler).
- [ ] Muted-Resources hinzufügen/entfernen funktioniert (falls die UI das anbietet).
- [ ] Keine pageErrors, keine Raw-Keys.
- [ ] Gescopter Typecheck grün, QA-Script grün, Screenshot @1440px zeigt das gefüllte, funktionierende Tab.

## QA-Hinweis
`desktop/scripts/qa-notif-settings.mjs` (Vorlage aus bestehendem `qa-*.mjs`): App auf `/#/settings` öffnen → Tab „Benachrichtigungen" anklicken → prüfen, dass Quiet-Hours-Felder befüllt sind, DND-Toggle klickbar ist und ein Save **keinen** pageError wirft. Screenshot des Tabs. (Den genauen Settings-Routen-/Tab-Selektor im Code nachsehen.)

## Hinweis fürs Review
Dieser Fix lässt auch das **schon vorhandene** Settings-Tab im Demo funktionieren — also Mehrwert über die Phase hinaus. Backend selbst ist real vorhanden (OpenAPI + internal/notification); nur die Demo-Mock-Schicht hing hinterher.
