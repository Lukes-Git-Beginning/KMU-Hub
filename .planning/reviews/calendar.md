# Review-Fäden — calendar

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `calendar` · **Strom:** D · **Reviewer (zugeteilt):** offen

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->

## ⬜ Cleanup — totes Kalender-Shell-Modul entfernt (2026-06-09)

**Befund (Spec-Korrektur):** Die Pilot-Spec `dein-pc-block/calendar-p1-views.md` ging davon aus, dass die Tag/Woche/Monat-Views im Modul `modules/calendar/CalendarLayout.tsx` nur Platzhalter sind und gebaut werden müssen. **Das stimmt nicht:**
- `modules/calendar/` (nur die eine Datei `CalendarLayout.tsx`) wird **nirgends importiert/gemountet** — totes Modul, ein abgebrochener früherer Versuch.
- Die **live** Kalender-UI ist `modules/kalender/KalenderPage.tsx` (Route `kalender`, ~3063 Zeilen). Sie hat `WeekView`/`DayView`/`MonthView` **bereits voll implementiert** (View-Switcher, Navigation, Monats-Grid, Monatstag→Tag-Drilldown) plus Event-CRUD, Kategorien, Raumbuchung, geteilte Kalender (Browse), Settings-Tab. → **Calendar P1 ist faktisch erledigt.**

**Gemacht:** Tote Dateien gelöscht (`modules/calendar/CalendarLayout.tsx` + ungenutzter `stores/calendar.ts` — `useCalendarStore` wurde nur von der toten Datei konsumiert, keine Tests). Kein Funktionsverlust (war nicht erreichbar).

**Offene Punkte / an andere Lanes:**
- 🟡 **Bug für Luke (Dashboard-Lane):** `components/dashboard/QuickActionsBar.tsx` Aktion „Neuer Termin" navigiert nach `/calendar` — diese Route existiert nicht (App.tsx mountet nur `kalender`). Muss `/kalender` sein. Trivialer 1-Zeilen-Fix, aber Dashboard-Lane → nicht von Strom D angefasst.
- **Folge für den Lane-Plan:** calendar ist eines der *fertigeren* Module, nicht der Shell aus der Spec. Nächster Strom-D-Schritt: echter Gap-Audit von `modules/kalender` gegen den Markt-/Phasenplan (statt der falschen Spec).

## ⬜ Phase A — Calendar Correctness-Sweep (2026-06-09)

**Kontext:** Nach Gap-Audit von `modules/kalender/KalenderPage.tsx` (Cluster A: „sieht fertig aus, tut aber nichts"-Bugs). Hinklick-Pfad: Sidebar → Kalender → Views Tag/Woche/Monat + Sidebar „Feiertage DE".

**Gefixt (4, alle per Screenshot @1440 verifiziert):**
1. **Feiertage geladen** — `useHolidays` war nie aufgerufen; jetzt verdrahtet (Region aus Settings `holidayRegion` → country/subdivision, Jahr aus `currentDate`), in `events`-useMemo gemerged wenn „Feiertage" sichtbar. Demo-Mock-Handler `/api/v1/calendar/holidays` + `buildMockHolidays` ergänzt. ✓ „Tag der Arbeit" erscheint am 1. Mai (Screenshot).
2. **Reminder-Send** — `uiEventToCreateRequest` ließ `reminder` fallen; jetzt `reminder_minutes` via `reminderToMinutes`. ⚠ **Nur Create-Pfad** — voller Round-Trip (Anzeige/Edit) braucht Backend (`ExpandedEvent` hat kein `reminders`-Feld) + `useSetEventReminders`-Wiring → siehe backend-handover. Nicht screenshot-sichtbar.
3. **Settings angewandt** — `defaultView` ist jetzt Initial-View; Arbeitszeiten `workStartHour/workEndHour` ersetzen die hartkodierten 7/20 (per `useWorkHours()`-Hook, lokales Shadowing in Week/DayView → keine 25 Call-Sites angefasst). ✓ Grid zeigt 8–17 (Screenshot).
4. **Now-Indicator** — war in WeekView auf 10:30 hartkodiert; jetzt echte Zeit via `useNowMinutes()` (Minuten-Tick) + Range-Guard. **DayView hatte gar keinen** → ergänzt. ✓ rote Linie bei 15:08 in Woche+Tag (Screenshot).

**Nebenbei:** ASCII-Umlaut in Mock-Daten „Besprechungsraeume" → „Besprechungsräume".

**Verifiziert als NICHT-Probleme (Capability-Map war ungenau):**
- Monats-„+N mehr": klickt sehr wohl — bubblet zum Zell-`onClick` → Drilldown in Tagesansicht. Kein Fix nötig.
- i18n-Key `recurrence.Benutzerdefiniert...` mit Punkten: `keySeparator:false` → flache Keys lösen sauber auf. Kein Raw-Key.

**Verify:** gescopter tsc exit 0 · `scripts/qa-kalender-fixes.mjs` grün (rawKeys [] in Woche+Monat, pageErrors []) · Screenshots Woche/Tag/Monat/Monat-zurück angesehen.

**Offene Punkte / Folge-Cluster (für Darien / Backend):**
- 🟡 **weekStartsOn** + **defaultReminder** aus Settings noch nicht angewandt (kein 5/7-Tage-Setting vorhanden; Woche fix Mo-Fr). Bewusst ausgelassen — minimaler Effekt, dokumentiert.
- 🟡 **Reminder voll** (Cluster B + Backend): `ExpandedEvent.reminders` + `useSetEventReminders` auf Update.
- 🟡 **Cluster B** (Markt-Lücken): Serientermin-Bearbeitung (this/future/all — `useUpdateRecurringEvent` ungenutzt), Einladungen/RSVP (Teilnehmer-Feld ist Phantom-UI), Kalender erstellen/verwalten.
- 🟡 **Cluster C** (Backend-schwer): Raumbuchung an Ressourcen-API (`RoomBookingView` lokaler State), Terminbuchung/Calendly komplett Mock.
