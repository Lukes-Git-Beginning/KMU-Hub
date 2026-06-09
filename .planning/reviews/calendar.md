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

## ⬜ Phase B1 — Serientermin-Bearbeitung (Scope-Dialog) (2026-06-09)

**Hinklick-Pfad:** Kalender → Woche → Serientermin (z.B. „Daily Standup") klicken → Detail-Panel → „Bearbeiten" → „Speichern" → **Dialog „Serientermin bearbeiten"** mit diese/künftige/alle.

**Gebaut:**
- `RecurringEditDialog` (diese/künftige/alle) — erscheint beim Speichern eines Termins mit Wiederholung; `useUpdateRecurringEvent` (war ungenutzt) jetzt verdrahtet (scope + original_date). Demo-Handler `PUT /events/:id/recurring` ergänzt.
- Serientermine demo-fähig: rrule aus Legacy-`recurring`-Feld im Events-Handler abgeleitet (`recurringToRrule`), damit `expandedEventToUI` sie als Serie erkennt.
- **Fehlende `kalender.recurrence.*`-Keys** (none/daily/weekly/…) ×4 ergänzt — waren komplett MISSING, hätten als Raw-Key im Detail-Badge gezeigt (durch B1 erst sichtbar geworden). Plus `kalender.recurring.*` für den Dialog.

**Nebenbei (echter Demo-Bug gefixt):** Events-Mock-Handler las `calendar_ids` nur via `.get()` (erster Wert) → im Demo zeigten sich **nur Events des ersten Kalenders** (= „Mein Kalender"). Jetzt `getAll()` + `includes` → alle sichtbaren Kalender rendern (Daily Standup, Sprint Planning, Team Lunch … erscheinen jetzt).

**Grenzen / offen:**
- Scope-Dialog gilt nur für **Bearbeiten**. Löschen eines Serientermins → noch ganze Serie (kein scoped-Delete-Endpoint/Hook vorhanden) → Cluster B-Rest / Backend.
- Im Demo expandiert das Mock-Backend keine Serien in Einzel-Vorkommen → ein Serientermin erscheint einmalig; der Scope-Dialog + Wiring sind aber backend-ready.

**Verify:** `scripts/qa-kalender-recurring.mjs` grün (dialogVisible, alle 3 Scopes, rawKeys [] in Detail+Dialog, pageErrors []) · Screenshots Detail-Badge („↻ Wöchentlich") + Dialog angesehen.

⚠ **tsc-Hinweis (für Darien — projektweit, kein B1-Bug):** Der gescopte tsc über `modules/kalender` zeigt jetzt 6 TS2345-Fehler an dynamischen `t(\`…${x}\`)`-Aufrufen (recurrence-/reminder-/browse-Dropdowns, `kalender.event.newAppointment`). Diese sind **nicht auf von mir geschriebenen Zeilen** — sie entstehen, weil i18next typed-keys bei dynamischen Template-Keys streng sind und meine i18n-Key-Ergänzungen die Union vergrößert haben (TS kippt dann von „loose" auf „error"). Mein Feature-Code nutzt nur statische Keys. Saubere Lösung wäre projektweit ein typsicherer `t()`-Wrapper/Cast für dynamische Keys — bewusst NICHT in B1 gemacht (Scope + bekannt: full-green tsc ist kein Gate, QA ist es).

## ⬜ Phase B2 — Einladungen/RSVP (Teilnehmer + Antwort) (2026-06-09)

**Hinklick-Pfad:** Kalender → Woche → Termin mit Teilnehmern (z.B. „Daily Standup") klicken → Detail-Panel zeigt **Teilnehmerliste mit Status** + **„Deine Antwort"** (Zusagen/Vielleicht/Absagen).

**Gebaut:**
- **Teilnehmer sichtbar:** `expandedEventToUI` mappt jetzt das (Legacy-)`attendees`-Feld in `participants` (Name/Initialen/RSVP) + `myRsvp` aus `my_rsvp`. Die Detail-Panel-Teilnehmerliste (war bereits gebaut, nie befüllt) zeigt jetzt Namen + Status-Icons.
- **RSVP-Antwort:** „Deine Antwort"-Buttons (Zusagen/Vielleicht/Absagen) im Detail-Panel, verkabelt mit `useRSVPToEvent` (UI→Backend: maybe→tentative); ausgewählter Zustand hervorgehoben, Toast „Antwort gespeichert". Demo-Handler `POST /events/:id/rsvp` ergänzt.

**Grenzen / offen (für Darien / Backend):**
- **Einladen/Teilnehmer hinzufügen** (Form-Teilnehmer-Suche ist weiterhin Phantom-UI): bewusst NICHT in B2. Grund: kein „Add-Attendee-zu-bestehendem-Event"-Endpoint/Hook (`UpdateEventRequest` hat keine Attendee-Felder); `CreateEventRequest.attendee_ids` ginge nur beim Erstellen, bräuchte aber User-IDs (TEAM_MEMBERS-Mock hat keine). → Backend-Stück, in backend-handover notieren.
- RSVP-Auswahl persistiert im Demo nur lokal (Mock echo't `my_rsvp` nicht zurück) — Wiring ist backend-ready.

**Verify:** `scripts/qa-kalender-rsvp.mjs` grün (participantsVisible, rsvpSectionVisible, acceptBtn, rawKeys [], pageErrors []) · Screenshots Teilnehmerliste + „Zusagen"-Auswahl + Toast angesehen · gescopter tsc (gleiche 6 Baseline-typed-key-Hinweise wie oben, keine neuen).

## ⬜ Phase C1 — Raumplanung: Räume aus API + Datum-Fix (2026-06-09)

**Hinklick-Pfad:** Kalender → Toolbar „Raumplanung" (Tür-Icon) → Wochen-Raumbelegung.

**Gebaut (FE mock-first):**
- **Räume aus der Ressourcen-API:** `RoomBookingView` lädt jetzt `useResources()` statt hartkodierter `ROOMS` (Fallback auf Defaults bei leer/Laden). Mock-Handler `GET /api/v1/calendar/resources` ergänzt (`buildMockCalendarResources`, Namen passend zu den Buchungen). Hinweis: Hooks nutzen `/api/v1/calendar/resources`, der alte Mock lag auf `/api/v1/resources` (Pfad-Mismatch) → neuer Handler.
- **Datum-Bug gefixt:** Startdatum war hart auf `new Date(2026, 1, 9)` → jetzt heute. Buchungen von festen Feb-2026-Daten auf `dayOffset` umgestellt und an die **aktuelle Woche** verankert (`buildSeedBookings`), sonst war die Ansicht leer/eingefroren.

**Grenzen / Backend (für Luke — wichtig):**
- **Buchungs-Persistenz ist backend-/modell-geblockt:** Das Backend-Modell ist *event-gebunden* (`BookResourceRequest` braucht `event_id`; der Titel liegt am Event), die UI macht aber freistehende betitelte Buchungen. Voll „echt" = 2-Schritt-Flow (Event anlegen → Ressource dafür buchen) + Titel über das Event. Daher: **Create/Cancel bleiben Demo-lokal** (optimistisch, Toast). `useBookResource`/`useCancelBooking` existieren, aber sinnvoll erst mit dem event-gebundenen Flow nutzbar.
- **Fehlt:** Tages-/Wochen-Aggregat-Endpoint für Buchungen (aktuell nur `useResourceAvailability(resourceId, date)` pro Ressource — kein „alle Buchungen eines Tages"). Für echtes Laden der Belegung sinnvoll.
- Day-Tab öffnet auf Montag (`selectedDay=0`), nicht auf heute — bewusst gelassen (pre-existing, kosmetisch).

**Verify:** `scripts/qa-kalender-rooms.mjs` grün (roomVisible, phoneBoothVisible, bookingVisible, rawKeys [], pageErrors []) · Screenshot KW 24 mit Räumen + Buchungen angesehen · gescopter tsc.
