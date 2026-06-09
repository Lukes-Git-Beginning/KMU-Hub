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
