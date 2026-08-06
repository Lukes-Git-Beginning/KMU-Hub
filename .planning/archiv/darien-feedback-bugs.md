# Darien-Screenshot-Feedback — offene Bugs (erfasst 2026-06-06, fixen NACH Pause)

> Aus 3 Screenshots. NICHT sofort fixen (Pause). Beim nächsten Terminal als erstes abarbeiten — sind großteils trivial.

## 1. Raw i18n-Keys im Kontakte-View-Toggle (Screenshot 2) — KRITISCH, mein Fehler
- **Symptom:** Toolbar-Buttons zeigen `kontakte.view.list` / `kontakte.view.grid` als Rohtext statt „Liste" / „Raster".
- **Ursache:** Keys werden in `modules/kontakte/KontaktePage.tsx` via `t()` benutzt, fehlen aber komplett in den i18n-Dateien (0 Treffer in de/en/fr/it).
- **Verwendete Keys (alle fehlen):** `kontakte.view.list`, `kontakte.view.grid`, `kontakte.view.toggleAriaLabel`.
- **Fix:** 3 Keys × 4 Locales additiv ergänzen (de: Liste/Raster/„Ansicht umschalten").
- **QA-Lehre:** Bug entstand in Block A (Liste↔Raster-Toggle), von meiner QA übersehen — Labels sind bei schmaler Breite `hidden sm:inline`, daher in den small-Screenshots unsichtbar; bei voller Breite sichtbar. → Raw-Key-Check IMMER bei voller Breite + ALLE `t()`-Keys des Views prüfen, nicht nur neue Namespaces. [[feedback_qa_thoroughness]]

## 2. Raw i18n-Keys im Aktivitäten-Sort (Screenshot 3) — KRITISCH (vorbestehend)
- **Symptom:** Sort-Steuerung zeigt `crm.activities.sort.created_at`, `crm.activities.sort.due_date`, `crm.activities.sort.subject` als Rohtext.
- **Ursache:** `ActivitiesListPage.tsx` `SORT_OPTIONS` nutzt diese labelKeys; fehlen in i18n (0 Treffer). Vorbestehend (nicht von mir eingeführt), aber sichtbar.
- **Fix:** 3 Keys × 4 Locales (de: „Erstellt" / „Fällig" / „Betreff").

## 3. ASCII-Substitution in Aktivitäts-Mock (Screenshot 3)
- **Symptom:** „Regelmaessiges Kundentreffen" statt „Regelmäßiges".
- **Ursache:** `mocks/data/contacts.ts` (1× „Regelmaessig"). Evtl. weitere im selben File gegenchecken.
- **Fix:** → „Regelmäßiges".

## 4. Consent-Grant-Flow: „Erteilen" + „Bestätigen" gleichzeitig (Screenshot 1) — UX, mit Darien klären
- **Symptom:** Bei „Telefonische Kontaktaufnahme" ist der Quelle-Picker (Dropdown „Vertrag" + grüner „Bestätigen") offen, ABER der grüne `Erteilen`-Button rechts bleibt zusätzlich sichtbar → redundant/verwirrend.
- **Vermutung:** Beim Start des Grant-Flows sollte der `Erteilen`-Button ausgeblendet werden (nur Picker + Bestätigen + evtl. Abbrechen zeigen). In `ConsentPanel.tsx` prüfen (`grantingId`-State steuert vermutlich den Picker, blendet aber den Erteilen-Button nicht aus).
- **Status:** Detail-Verhalten mit Darien kurz abstimmen, dann fixen.
