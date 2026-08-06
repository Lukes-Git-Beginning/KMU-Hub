# UI-Tiefen-Audit & Umsetzungspläne — alle Module

> **Zentrale Arbeitsdatei für den Modul-Ausbau bis Launch.** Pro Modul: Ist-Befund (UI-Tiefe) · Markt-Soll · Phasen-Umsetzungsplan · Backend-Gaps (→ Luke).
> Status 2026-06-02: Methodik + QA-Standard definiert. Audits fertig: **Kunden-Zentrale (CRM+Kontakte), kalender, dialer, formulare, dokumente**. Restliche Module folgen.
> Verwandte Dateien: `.planning/backend-gaps.md` (Luke-Liste) · `.planning/ist-abgleich.md` (Feature-Abgleich Wellen 1–3) · `.planning/pre-launch-todos.md` (Login/UX) · `.planning/cosmi-modul-marktvergleich.txt` (Markt-Grundlage).
> Beim Bauen: zuerst Lukes RLS-Welle abwarten · visueller QA-Pass (Teil G) ist Pflicht je Bau-Einheit · Architektur „Kontakte = Kunden-Zentrale inkl. CRM".

## Context
Die bisherigen Ist-Abgleich-Wellen (1–3, in `.planning/ist-abgleich.md`) prüften **Feature-Existenz** (gibt es Funktion X, hängt FE an echtem BE oder Mock). Sie prüften NICHT, ob ein Modul als **bedienbares Programm tiefgründig vollständig** ist — alle Reiter, Menüs, Buttons, Listen-Features, Detail-Tabs, Aktionen. Darien will, dass die Module tiefgründig gehen (echte Markt-Alternative für KMU bis 30–40 MA).

Dieses Dokument ist der **erste UI-Tiefen-Audit, exemplarisch am Modul CRM** (ZFA-Kernmodul). Es ist ein **reiner Befund** wie die vorherigen Wellen — keine Code-Änderung. Methodik: Soll-Katalog aus Markt-Recherche (HubSpot, Pipedrive, Zoho, CentralStation, weclapp) × Ist aus Code → Gap. Wenn das Format passt, wird es auf die übrigen ZFA-Must-Module skaliert.

Quellen der Analyse: 3 parallele Explore-Agents — (A) CRM-Frontend-Code-Inventar, (B) Markt-UI-Recherche, (C) Cosmi-Shared-Components.

---

## Teil A — Audit-Methodik (wiederverwendbar für alle Module)

Pro Modul gegen den Code geprüft, je 8 Dimensionen:
1. **Navigationsstruktur** — alle nötigen Reiter/Views/Einstieg vorhanden & erreichbar?
2. **Listen-Ansichten** — Suche · Filter · Sortierung · Spalten-Config · Bulk-Aktionen · Pagination · Import/Export?
3. **Detail-Ansichten** — alle Sub-Tabs · Felder · Inline-Edit · Verknüpfungen · Timeline?
4. **Aktions-Buttons** — jeder erwartete Button vorhanden **und** verdrahtet (nicht Mock/toast/501)?
5. **Dialoge/Drawer/Modals** — vorhanden, echte Picker statt Freitext?
6. **Spezial-View** (hier Pipeline) — DnD, Quick-Actions, Stage-Management?
7. **Modul-Settings** — eigene Konfig (Pipeline-Stufen, Custom-Fields, Scoring, Tags)?
8. **Zustände** — Empty / Loading / Error / rollenabhängige UI?

Bewertung je Soll-Punkt: **Priorität** (MUSS/SOLLTE/KANN aus Markt) × **Ist** (✅ da / ◐ teilweise / ✗ fehlt) → **Handlung** (🟡 FE-Arbeit, BE+Hooks da · 🔴 BE nötig → Luke · 🟠 beides) + Hinweis auf wiederverwendbare Cosmi-Komponente.

---

## Teil B — CRM-Befund je Dimension

### 1. Navigationsstruktur
- **Ist:** CRM hat Sub-Nav mit 5 Reitern (Kontakte, Firmen, Deals, Aktivitäten, Suche, in `CRMLayout.tsx`). **Aber** `/crm` ist **nicht in der Sidebar** registriert (`nav-items.ts`) — Zugang läuft über `/kontakte` (paralleles Modul) oder direkte URL.
- **Soll-Lücken:** kein **Reports/Analysen**-Reiter · keine **Einstellungen** · keine **Lead-Inbox** (MUSS-Befund für Finanzberatung: ungeprüfte Telefon-/Formular-Leads vor Pipeline-Entry). Globale `Cmd+K`-Suche existiert projektweit (`GlobalSearchDialog`), CRM-Objekte teilweise drin.
- **Handlung:** 🟡 CRM in Sidebar + Reports-/Settings-Reiter ergänzen; **Konsolidierung `/crm` ↔ `/kontakte` klären** (zwei parallele Kontakt-Implementierungen).

### 2. Listen-Ansichten (Kontakte/Firmen/Deals/Aktivitäten)
- **Ist:** Suche ✅, Pagination ✅, Virtualisierung ✅, Empty/Loading/Error ✅. Import/Export-Buttons nur bei Kontakten.
- **Soll-Lücken (alle MUSS/SOLLTE):**
  - ✗ **Bulk-Auswahl + Bulk-Aktionen** (keine Liste hat es) 🟡
  - ✗ **Spalten-Konfiguration** (show/hide, reorder) 🟡
  - ✗ **Sortier-UI** (Hook-Params `sort_by/sort_desc` existieren, keine UI) 🟡
  - ✗ **Gespeicherte Filter / Segmente / dynamische Listen** 🟠 (Segmente brauchen evtl. BE)
  - ✗ **Filter-Panel** (Stage-/Tag-/Owner-/Datum-Filter) — `stage_id` im Hook da, keine UI 🟡
  - ◐ Import/Export: `ImportExportDialog.tsx` ist **komplett Mock** (kein CSV-Parsing, Random-Preview, toast-only); Firmen/Deals haben gar keinen Button 🟠 (XLSX-Endpoint fehlt — Welle 1)
- **Wiederverwendung:** `TaskFilterBar` (Filter-Vorlage), `TaskListView`/`TaskListHeader` (GroupBy/Sort), `ItemActions` (Zeilen-Kontextmenü), shadcn `Table`. → Bulk/Filter/Sort sind als Muster im Projekt vorhanden.

### 3. Detail-Ansichten
- **Kontakt-Detail (Ist):** Info-Card + Tags (read-only) + 2 Sub-Tabs (Aktivitäten, Aufgaben). 
- **Soll-Lücken (MUSS/SOLLTE):**
  - ✗ **Aktivitäts-Timeline** — `ContactTimeline.tsx` + `useContactTimeline` voll implementiert, aber **nirgends eingebunden (Dead Code)** 🟡 (Quick-Win)
  - ✗ **E-Mail-Verlauf am Kontakt** — BE-Link-API + Hook (`getContactEmails`) da, FE nur `mailto:` 🟡
  - ✗ **Notiz-Sofort-Log** (ohne Modal), Notizen-Tab 🟡
  - ✗ **Dokumente/Anhang-Tab** 🟠 (cross-cutting Upload)
  - ✗ **verknüpfte Deals-Panel** am Kontakt 🟡
  - ✗ **Consent/DSGVO-Status** im Detail (Panel existiert in kontakte-Modul, nicht eingebunden) 🟡
  - ✗ **E-Mail-/Anruf-Buttons** aus Detail (im `/kontakte`-Modul vorhanden, im `/crm` nicht; Dialer-CTI-Bridge existiert) 🟡
- **Firmen-Detail:** kein **Deals-Tab**, kein Aufgaben-Tab 🟡. **Deal-Detail:** kein **Quick-Stage-Wechsel** (nur via Edit), `alert()` statt Toast bei Quote-Fehler (UI-Bug) 🟡.

### 4. Aktions-Buttons
- **Ist:** Create/Edit/Delete für Kontakt/Firma/Deal ✅ verdrahtet; Aktivitäten Complete/Delete/Edit ✅.
- **Soll-Lücken:** ✗ Merge (BE `MergeContacts` da, kein Button), ✗ Duplicate (nur im kontakte-Modul), ✗ Bulk-Aktionen, ✗ E-Mail/Anruf/Aktivität-Quick-Add aus Detail, ✗ Tag-Zuweisung aus Detail (read-only). Tag-Mutation gibt zudem **501** über HTTP (nur gRPC).

### 5. Dialoge / Picker
- **Ist:** ContactForm/CompanyForm/DealForm/ActivityForm ✅ vollständig. `ConfirmDialog` ✅.
- **Kritische Soll-Lücke:** **`DealFormDialog` nutzt Freitext-Inputs für Kontakt/Firma** statt echtem Lookup/Autocomplete auf die Datensätze 🟡 (Daten-Integritäts-Problem). 

### 6. Pipeline / Board
- **Ist:** Kanban ✅ (Stages, Summen, Wahrscheinlichkeits-Badge, Empty-State).
- **Soll-Lücken (MUSS):**
  - ✗ **Drag & Drop** — `useMoveDealToStage` voll implementiert, **nicht verdrahtet** („future enhancement"). dnd-kit ist im Projekt vorhanden (work-Kanban nutzt es) 🟡 (Quick-Win)
  - ✗ **Quick-Actions auf Karten** (Stage wechseln, Notiz, Aktivität) 🟡
  - ✗ **mehrere Pipelines**, ✗ **Rotting/Idle-Indikator** (SOLLTE), ✗ **Forecast-View** (SOLLTE; weighted_value im BE da) 🟠
  - ✗ **Stage-Verwaltung aus Board** 🟡

### 7. Modul-Settings / Konfiguration  ← größte Tiefen-Lücke
- **Ist:** **Es gibt KEINE CRM-Einstellungsseite.** Obwohl die Backend-Hooks existieren:
  - Pipeline-Stage-CRUD (`usePipelineStages` + Create/Update/Delete/Reorder) ✅ Hooks da → **keine UI** 🟡
  - Custom-Field-Builder: `CustomFieldsConfig.tsx` existiert (kontakte-Modul), aber lokaler Store, nicht verdrahtet; `/api/v1/custom-fields` BE da 🟡
  - Tag-Verwaltung: keine zentrale UI (Tags = freie Strings) 🟡
  - Lead-Scoring-Regeln: ✗ FE+BE (🔴, Welle 1)
- **Soll (MUSS):** Pipeline-Stage-Editor, Custom-Field-Builder, Rollen/Rechte, E-Mail-Integration-Setup. → Alle als CRM-`/settings`-Reiter zu bauen, BE überwiegend vorhanden.

### 8. Zustände
- **Ist:** Empty/Loading/Error überall ✅ sauber (inkl. Illustrations `EmptyContacts`/`EmptyDeals`).
- **Soll-Lücke:** ✗ **keine rollen-/berechtigungsabhängige UI** (alle Buttons für alle sichtbar) — relevant ab Team-Größe / Finanzberatung-Compliance 🟠.

---

## Teil C — Konsolidierte Gap-Liste (priorisiert)

**🟡 = Frontend-Arbeit (BE/Hooks existieren — mein Part, kein Luke) · 🔴 = Backend nötig (Luke) · 🟠 = beides**

### MUSS — Tiefe fehlt, hohe Wirkung
| Gap | Handlung | Wiederverwenden |
|---|---|---|
| Pipeline **Drag&Drop** (Hook da) | 🟡 | dnd-kit (work-Kanban) |
| **Pipeline-Stage-Editor**-UI (Hooks da) | 🟡 | Dialog + ItemActions |
| Kontakt-**Timeline** einbinden (Dead Code) | 🟡 | ContactTimeline existiert |
| Listen **Bulk-Aktionen** | 🟡 | ItemActions, Checkbox |
| Listen **Filter-Panel** + **Sortier-UI** (Params da) | 🟡 | TaskFilterBar |
| **DealForm echte Kontakt/Firma-Picker** (statt Freitext) | 🟡 | Select/Combobox |
| **E-Mail-Verlauf** am Kontakt (Hook da) | 🟡 | — |
| **Verknüpfte Deals**-Panel am Kontakt/Firma | 🟡 | DetailPanel/Tabs |
| **CRM-Reports-Reiter** (BE-Reports da) | 🟡 | StatCard, berichte-Charts |
| **Custom-Field-Builder**-UI (BE da) | 🟡 | CustomFieldsConfig anbinden |
| **Import/Export** echt (Dialog ist Mock) | 🟠 | XLSX-Endpoint fehlt (Luke) |
| **Lead-Quelle** + **Lead-Status**-Felder | 🟠 | Custom Fields |
| **Lead-Inbox** (ungeprüfte Leads) | 🟠 | — |
| **CRM in Sidebar** + `/crm`↔`/kontakte` konsolidieren | 🟡 | nav-items.ts |

### SOLLTE
| Gap | Handlung |
|---|---|
| Spalten-Konfiguration in Listen | 🟡 |
| Gespeicherte Filter / Segmente | 🟠 |
| E-Mail/Anruf/Aktivität-Quick-Add aus Detail (Dialer-CTI da) | 🟡 |
| Merge- + Duplicate-Button im CRM (BE da) | 🟡 |
| Notiz-Sofort-Log + Notizen-Tab | 🟡 |
| Dokumente-Tab am Kontakt | 🟠 (Upload cross-cutting) |
| Consent/DSGVO-Status im Detail (Panel da) | 🟡 |
| Pipeline: Rotting-Indikator, Forecast-View, mehrere Pipelines | 🟠 |
| Wiedervorlage „nächster Schritt"-Pflicht beim Stage-Wechsel | 🟡 |
| Empfohlen-von-Lookup + Empfehlungs-Report | 🟠 |
| Rollen-/berechtigungsabhängige UI | 🟠 |
| Quote-Fehler `alert()` → Toast (Bug) | 🟡 |

### KANN (später / Enterprise)
Lead-Scoring-ML, Blueprint-Pflichtprozesse, KI-Zusammenfassung, Meeting-Booking-Page (überschneidet Kalender-Terminbuchung), strukturiertes Beratungsprotokoll-Template (Finanzberatungs-Nische), Audit-Log.

---

## Teil D — Kernbefunde / Muster

1. **Tiefe ≠ Existenz:** Die Welle-1-Bewertung „crm gut" stimmt auf Feature-Ebene, aber auf **UI-Tiefen-Ebene fehlt viel** — v.a. Bulk/Filter/Sort in Listen, Detail-Tabs (Timeline/E-Mail/Deals/Dokumente), und **eine komplette Settings-Ebene**.
2. **Das Mock→Hook-Muster setzt sich fort, plus „Hook ohne UI":** Mehrere fertige Backend-Hooks haben **gar keine Oberfläche** (Pipeline-Stage-CRUD, MoveDealToStage/DnD, ContactTimeline). Das ist überwiegend **FE-Arbeit ohne Luke-Bedarf** — gute Nachricht.
3. **Doppelstruktur Kontakte:** `/crm/contacts` und `/kontakte` sind zwei parallele Implementierungen derselben Daten, inkonsistent (API vs. lokaler Store). Muss vor dem Tiefen-Ausbau konsolidiert werden, sonst doppelte Arbeit.
4. **Daten-Integrität:** DealForm-Freitext für Kontakt/Firma + Tag-501-über-HTTP sind echte Fehlerquellen.
5. **Reichlich wiederverwendbare Bausteine** (Teil C-Spalte): DetailPanel, ConfirmDialog, TaskFilterBar, ItemActions, Tabs, EmptyState+Illustrations, PageHeader, RichTextEditor, Modul-Blaupause aus `work/`. → Tiefen-Ausbau bedeutet großteils **Komposition vorhandener Teile**, nicht Neubau.

---

## Teil E — Einordnung & nächster Schritt (Befund, keine Umsetzung)

- **Aufwand grob:** Der CRM-Tiefen-Ausbau ist mehrheitlich **FE-Kompositionsarbeit** (🟡) — Pipeline-DnD, Stage-Editor, Listen-Bulk/Filter/Sort, Detail-Tabs, Settings-Reiter. Echte Luke-Punkte (🔴/🟠): XLSX-Import, Lead-Scoring, Segmente, Empfohlen-von-Feld, Upload (cross-cutting) — die meisten stehen schon in `backend-gaps.md`.
- **Voraussetzung:** `/crm`↔`/kontakte`-Konsolidierung zuerst entscheiden.
- **Verifikation des Befunds (optional vor Umsetzung):** App lokal starten und CRM durchklicken (`/run`-Skill / Electron-Dev), um die Code-Befunde visuell zu bestätigen (z.B. dass Pipeline kein DnD hat, Timeline fehlt, Settings-Reiter fehlt). Darien kann zusätzlich Referenz-Screenshots liefern, um „tiefgründig" pro Dimension visuell festzuzurren.

**Getroffene Entscheidungen (2026-06-02):**
- Fahrplan: (1) CRM-Befund besprechen → (2) CRM-Umsetzung durchplanen → (3) gleicher Prozess (Audit + Plan) für übrige ZFA-Must-Module. Bauen erst danach + nach Lukes RLS-Welle.
- **FINALE Architektur (02.06): EIN Modul „Kontakte = Kunden-Zentrale" — KEIN eigenständiges CRM-Modul.** CRM ist kein App-Modul, sondern eine Funktions-Kategorie (Vertrieb). Die vorhandenen `crm/`-Code-Funktionen (Pipeline, Deals, Aktivitäten) werden **ins Kontakte-Modul integriert + sichtbar gemacht**. Das Kontakte-Modul enthält damit: (a) Stammdaten/Adressbuch, (b) **Kunden-360°-Verknüpfungen** (Projekte/Aufgaben, Dateien, Verträge, Deals, Angebote/Rechnungen direkt am Kunden — Cosmi-USP), (c) **vollen CRM-/Vertriebsumfang** (Pipeline, Leads, Forecast, Reports). Anspruch: **voller Funktionsumfang / Feature-Parität**. Doppelstruktur crm/contacts ↔ kontakte wird dabei aufgelöst (eine Quelle, echte API). (chat+kommunikation bleibt das separate Merge.)
- **Visueller QA-Pass = Pflichtstandard** für ALLE Module (siehe Teil F): nach jeder Bau-Einheit App rendern @ 3 Fenstergrößen + Screenshot-Analyse auf Layout-Bugs.
- `video`/`meetings` → **meetings kanonisch**; `modules/video/VideoPage` = verwaiste Route → Cleanup. `features/video/` (LiveKit) bleibt.
- `buchhaltung` → langfristig **eigenes Buchhaltungs-Modul**; jetzt nicht umbauen; im Audit mit finanzen zusammen betrachten.
- Altlasten (calendar-Stub, Profil-Doppel, Security-Doppel, settings-Mock-Sessions) → **einzeln beim jeweiligen Modul-Audit** besprechen.

---

## Teil F — Umsetzungsplan: „Kontakte = Kunden-Zentrale" (inkl. CRM)

Ein Modul, **voller Funktionsumfang (Feature-Parität)**. **🟡 = FE (mein Part)** · **🔴 = Luke (BE)**. Wiederverwendung aus Teil C. Reihenfolge = Vorschlag (anpassbar).

**Phase 0 — Fundament**
- `crm/`-Funktionen ins Kontakte-Modul integrieren & in der Sidebar sichtbar machen (Kontakte = Kunden-Zentrale). Doppelstruktur crm/contacts ↔ kontakte auflösen (eine Quelle, echte API statt lokalem Store). 🟡

**Phase 1 — Stammdaten & Listen**
- Liste: Filter · Sortierung · Spalten-Config · Bulk-Aktionen · echtes Import/Export. 🟡 (+🔴 XLSX-Endpoint)
- Custom Fields · Tags/Gruppen-Verwaltung · Dubletten-Erkennung & Merge (BE da). 🟡
- Kontakt-Detail: Timeline einbinden (Dead Code) · alle Felder · Inline-Edit. 🟡

**Phase 2 — Kunden-360° (Verknüpfungen — Cosmi-USP)**
- Projekte/Aufgaben, Dateien, Verträge, Angebote/Rechnungen, Deals **direkt am Kunden** zuordnen & anzeigen. 🟡 (+🔴 Verknüpfungs-Endpoints, Datei-Upload)
- E-Mail-Verlauf am Kunden (Hook da) · E-Mail/Anruf-Quick-Actions (Dialer-CTI) · Notiz-Sofort-Log · Consent/DSGVO-Panel. 🟡

**Phase 3 — Pipeline & Deals (Vertriebs-Herz)**
- Drag&Drop verdrahten (dnd-kit) · echte Kontakt-Picker im Deal · Quick-Actions auf Karten · Rotting-Indikator · Stage-Summen · Won/Lost-Grund · mehrere Pipelines · Forecast-View · Quote-`alert()`→Toast-Bug. 🟡 (mehrere Pipelines evtl. 🔴)

**Phase 4 — Leads**
- Lead-Inbox (ungeprüft, vor Pipeline) · Lead-Quelle + Lead-Status · Formular→Lead (Formulare-Verknüpfung) · Lead-Scoring. 🟡 + 🔴 (Lead-Entität, Scoring)

**Phase 5 — Aktivitäten & Wiedervorlage**
- „Meine Aufgaben heute/überfällig" · Wiedervorlage „nächster Schritt"-Pflicht beim Stage-Wechsel · periodische Wiedervorlage (Jahresgespräch Bestandsmandanten). 🟡

**Phase 6 — Auswertungen** (BE-Reports da; StatCard/berichte-Charts)
- Pipeline-Funnel · Conversion · Aktivitäten- · Lead-Quellen-Report · Forecast. 🟡

**Phase 7 — Einstellungen**
- Pipeline-Stage-Editor (Hooks da) · Custom-Field-Builder · Lead-Scoring-Regeln · Tag-Verwaltung. 🟡

**Phase 8 — Finanzberatungs-Tiefe**
- „Empfohlen von"-Lookup + Empfehlungs-Report · Beratungsprotokoll-Tab · Mandanten-Segmente (A/B/C). 🟡 + 🔴 (Empfohlen-von-Feld, Segmente)

**Luke-Bündel (parallel → backend-gaps.md):** XLSX-Import · Lead-Entität + Inbox-Endpoint · Lead-Scoring · mehrere Pipelines · gespeicherte Segmente · Empfohlen-von-Feld · Foto/Datei-Upload (cross-cutting) · Verknüpfungs-Endpoints.

## Teil G — Visueller QA-Pass (Pflichtstandard, ALLE Module)
Nach JEDER Bau-Einheit, bevor sie als fertig gilt (statischer Code-Review erkennt Layout-Bugs NICHT). **Claude macht die Screenshots selbst, automatisiert** — kein manueller Aufwand für Darien.
1. App starten (Electron-Dev / Playwright), per Automation **jedes Menü, jeden Reiter, jeden Header, jede View** durchklicken.
2. Pro Ansicht Screenshots bei **3 Fenstergrößen** (voll ~1440 · halbiert ~720 · klein) **und in relevanten Zuständen** (Empty State · gefüllte Liste · offener Dialog/Drawer · Detail mit allen Tabs).
3. **Claude analysiert die Bilder selbst** (liest Screenshots nativ; alt. QA-Sub-Agent) auf: **abgeschnittener/überlaufender Text · Überlappungen · Elemente außerhalb des Sichtbereichs · kaputte Ausrichtung/Abstände · Scroll-/Responsive-Brüche**.
4. Befund → fixen → erneut rendern → bis sauber. Erst dann gilt die Einheit als fertig.
5. **Marken-/Geschmacks-Feinschliff** danach mit Dariens Referenz-Screenshots (separat von der Bug-Findung).
(Konkrete Render-Methode — Electron-Dev vs. Playwright — wird zu Baubeginn fixiert + kurz getestet, dass das automatisierte Durchklicken sauber läuft.)

## Verifikation (gesamt)
Je Phase: App durchklicken (Funktion stimmt) **+** visueller QA-Pass (Layout stimmt @ 3 Größen).

---

# MODUL-AUDIT 2: kalender (ZFA-kritisch)

## Ist-Befund (UI-Tiefe)
**Kern solide:** Tag/Woche/Monat-Views ✅, Drag&Drop + Resize ✅, CalDAV-App-Passwörter ✅, Kategorien anlegen/löschen ✅, geteilte Kalender + Sichtbarkeits-Toggle ✅, QuickCreate + EventFormModal ✅. (Altlast `modules/calendar/` = toter Stub, nicht im Router.)

**Große Tiefen-/Mock-Lücken:**
- **Terminbuchung = 100 % Mock** — `BOOKING_SERVICES`/`MOCK_BOOKINGS`/`BOOKING_STAFF`/`MOCK_EXTERNAL_SERVICES` hardcoded (Demo für Friseur/Massage!), Booking-URL hardcoded, kein Backend. **ZFA-kritisch.**
- Räume hardcoded (`ROOMS`-Konstante); `useResources` (voll implementiert) ungenutzt; RoomBookingView komplett Mock.
- Erinnerungen werden nicht an API übergeben (Adapter-Lücke `uiEventToCreateRequest`); `useSetEventReminders` ungenutzt → Push-Reminder-Polling läuft ins Leere.
- Attendees/RSVP nie geladen (`attendeeToUI` nicht in `expandedEventToUI` aufgerufen); `useRSVPToEvent` ungenutzt.
- Feiertage nie geladen (`useHolidays` ungenutzt → Feiertags-Kalender bleibt leer).
- Settings (Arbeitszeiten/Standard-View/Feiertags-Region) nur lokaler Store; KalenderPage ignoriert sie (Mo–Fr + 7–20 Uhr hardcoded); `useUpdateCalendarPreferences` ungenutzt.
- CalDAV: Enable/Disable/Status/Test-Hooks existieren, kein UI.

**Bugs:** Current-Time-Indikator hardcoded auf 10:30 · Kategorien-Edit ist No-Op (kein API) · „+N weitere" im Monat ohne onClick · kein Empty/Error-State · kein Cross-Day-Drag · zwei identische Settings-Icons (Usability).
**Fehlende Views:** Jahr · Agenda/Liste · 7-Tage-Woche (Sa/So) · Tastatur-Shortcuts.

## Markt-Soll (Kern = Terminbuchung, Calendly/Cal.com-Niveau)
MUSS: öffentliche Buchungsseite (Slug/Embed) · Verfügbarkeitsregeln (Arbeitszeiten, Buffer, Vorlaufzeit, Max/Tag) · Termintypen + Dauer · Bestätigungs-Mail · Erinnerungs-Mail (24–48h) · Umbuchung/Stornierung durch Gast · Formularfelder vor Buchung · Online-Meeting-Link.
SOLLTE: Round-Robin auf mehrere Berater (gleichmäßig) · Arbeitszeiten pro Berater · „Requires Confirmation" (Compliance) · SMS-Erinnerung. KANN: Routing-Forms, Weighted-RR, Payment.

## Umsetzungsplan kalender
**Phase 1 — Mock→API verdrahten (FE, Backend existiert):** Räume aus `useResources` · Erinnerungen an API + Adapter-Fix · Attendees/RSVP laden & verdrahten · Feiertage aus `useHolidays` · Settings an Preferences-API · CalDAV Enable/Status/Test-UI. 🟡
**Phase 2 — Bugs + Views:** Current-Time live · „+N weitere"-Popover · Empty/Error-States · Cross-Day-Drag · echter RRULE-Editor (BYDAY/INTERVAL/UNTIL) · 7-Tage-Woche · Agenda-View · Kategorien-Edit. 🟡 (Kategorien-Edit braucht 🔴 `useUpdateEventCategory`-Endpoint)
**Phase 3 — TERMINBUCHUNG (ZFA-kritisch, FE-Neubau + großer Luke-Block):** echte Buchungsseiten-Verwaltung · Verfügbarkeits-Engine · öffentlicher Buchungs-Flow · Bestätigungs-/Erinnerungs-Mails · Umbuchung/Storno · Round-Robin. 🟡 + 🔴 (Backend fehlt komplett)
QA-Pass (Teil G) je Phase.

## Backend-Gaps kalender (Luke → backend-gaps.md)
🔴 **Terminbuchung komplett**: `booking-pages` CRUD · öffentl. `/book/:slug` (unauth) · `/availability` · `/bookings` · Bestätigungs-/Erinnerungs-Mail-Versand · Round-Robin-Verteilung. · 🔴 `useUpdateEventCategory`-Endpoint. (Reminders/Attendees/Resources/Holidays/Preferences-Endpoints existieren bereits — nur FE-Anbindung nötig.)

---

# MODUL-AUDIT 3: dialer (ZFA-Telefonakquise)

## Ist-Befund (UI-Tiefe)
**Recht weit & verdrahtet:** Agent-Workspace mit 4-Phasen-Flow (idle→dialing→on_call→wrap_up) ✅, Call-Controls (Anrufen/Auflegen/Skip/Notizen) ✅, Outcome-Grid + Wrap-up + Callback-Picker ✅, Kampagnen-Liste/Detail/Erstellen ✅, Kontakte hinzufügen (manuell + CRM-Saved-Filter) ✅, Agent-Dashboard (KPIs + Outcome-Donut, 30s-Refresh) ✅, Settings = Wrap-up-Code-Verwaltung ✅, CTI/CRM-Bridge ✅.

**Lücken (Tiefe + DSGVO-kritisch):**
- **Skript-Engine fehlt komplett** (Gesprächsleitfaden) — Markt-MUSS für regulierte Finanzberatung: Branching, Pflichtfelder, **Einwilligungs-Capture**.
- **Recording fehlt komplett** (FE+BE; keine Aufnahme/Playback, keine Consent-Ansage).
- **DNC-Sperrliste fehlt komplett** (DSGVO-Pflicht: Widersprüche sperren) — kein UI, kein BE.
- **Consent-Status unsichtbar im UI** — BE blockt Anruf bei fehlender Einwilligung, FE zeigt keine Warnung/Badge (Fehler läuft ungefangen in toast-error). ⚠ Welle-1-Risiko: Standard-`NewService`-Konstruktor lässt `consentAsserter` nil.
- **Kampagnen-Settings nicht editierbar** (MaxAttempts/CallHoursFrom-To/TimeZone im BE-Modell, kein UI) — Anrufzeiten-Einschränkung ist DSGVO-relevant.
- **Agenten-Zuweisung** nicht per UI pflegbar (Feld da, nur Avatar-Anzeige).
- AMD fehlt · Mute/Hold/Transfer fehlen · Power/Predictive disabled (Phase 2/3) · keine Callback-Listenview („fällige Callbacks heute") · Dashboard nur Tagesdaten (kein Zeitraum/Historie).

## Markt-Soll (Outbound-Akquise KMU)
MUSS: Preview + Power-Dialer · Workspace (Kontakt-Card + Notiz + Outcome + Callback) · **Skript-Engine** (Branching, Pflichtfelder, Einwilligungs-Capture) · DSGVO (Aufnahme-Ansage, **DNC-Liste**, Anrufzeiten, echte Caller-ID, EU-Speicherung) · Auto-Anruf-Log + Click-to-Dial. SOLLTE: Multi-Stage-Kampagnen + Redial-Regeln · Screen-Pop / Voicemail-Drop / AMD · Echtzeit-Dashboard (Agenten-Status, Conversion-Rate) · Termin-Anlage aus Dialer. KANN/Overkill: Predictive (Drop-Call-Risiko bei Beratung), KI-Sentiment, BI.

## Umsetzungsplan dialer
**Phase 1 — Compliance (DSGVO, kritisch):** Consent-Status im UI (Badge/Warnung „Anruf blockiert — keine Einwilligung") + `consentAsserter`-nil-Fix · DNC-Sperrliste (UI + BE) · Anrufzeiten in Kampagnen-Settings-UI. 🟡 + 🔴
**Phase 2 — Skript-Engine:** Gesprächsleitfaden im Workspace (Branching, Pflichtfelder, Einwilligungs-Capture, dynamische Kundendaten). 🟡 + 🔴 (BE-Skript-Modell)
**Phase 3 — Kampagnen-Tiefe:** alle `CampaignSettings` im Dialog editierbar · Agenten-Zuweisung · Multi-Stage/Redial-Regeln · Callback-Listenview. 🟡 (+🔴 Multi-Stage)
**Phase 4 — Telefonie-Ausbau (Infra/Luke):** Recording + Consent-Ansage + Playback-Archiv · AMD · Voicemail-Drop · Power-Dialer aktivieren. 🔴 + 🟡
**Phase 5 — Dashboard:** Zeitraum-Filter · Historie · Conversion-Rate (Termine/Anrufe). 🟡
QA-Pass (Teil G) je Phase.

## Backend-Gaps dialer (Luke → backend-gaps.md)
🔴 Skript-Modell + Engine · DNC-Repository + Check im `InitiateCall` · Recording (Storage, recording_url, Consent-Ansage) · AMD · Power/Predictive-Logik · **consentAsserter-nil-Fix im Standard-Konstruktor** (Sicherheits-Bug). (Consent-Assert, CTI-Bridge, Outcomes, Callback existieren bereits.)

---

# MODUL-AUDIT 4: formulare (ZFA-Lead-Capture)

## Ist-Befund (UI-Tiefe)
**Builder solide:** 8 Feldtypen mit Preview ✅, bedingte Logik (Show/Hide) voll ✅, mehrseitige Formulare ✅, Vorlagen (Galerie + Duplizieren) ✅, Submissions-Liste/Detail/Status ✅, isPublic-Toggle ✅, Empty/Loading/Error ✅.

**Kritische Lücken:**
- **KEIN öffentlicher Submit-Endpoint** — alle Submission-Routes hinter Auth. **Externes Lead-Capture funktioniert gar nicht.** ZFA-kritisch (Online-Erstberatungs-Anfrage geht nicht).
- **Share-Dialog = Stub** — „Link kopieren"/„Per E-Mail" nur Toast, kein echter Link/Clipboard/Embed-Code.
- **Webhook-UI fehlt komplett** — BE + alle Hooks (CRUD, Delivery-Log, HMAC) fertig, null Frontend.
- **Export = Stub** — CSV/XLSX-Button nur Toast; Hook + BE fertig, nicht verdrahtet.
- **DnD nur Dekor** — GripVertical sichtbar, `reorderFields`-Store da, kein dnd-kit eingebunden.
- **Email-Feldtyp aus UI ausgeblendet** (BE-valide, fehlt im Builder-Menü) · kein Feldtyp-Wechsel · keine Validierungsregeln (min/max/regex).
- **DSGVO:** kein Consent-Feldtyp, kein IP-Logging-Opt-out, kein Datenschutz-Link, kein Spam-Schutz.
- Formular-Aktionen (E-Mail/Task/CRM bei Submission) = Store-Typen „Sprint 2", kein UI. · Completion-Rate hardcoded 87% (`useFormStats` ungenutzt). · kein Status-Filter/Pagination bei Submissions.

## Markt-Soll (Lead-Capture, Jotform/Typeform/Fillout/Tally)
MUSS: Pflicht-Feldtypen (inkl. E-Mail/Telefon) · DnD-Builder · Mehrseitig · bedingte Logik · **öffentlicher Link + Inline/Popup-Embed** · Submission-Liste + CSV-Export · **E-Mail-Benachrichtigung + Autoresponder** · **DSGVO-Consent-Checkbox (Pflicht) + EU-Hosting + Spam-Schutz** · Logo/Farben/Danke-Seite. SOLLTE: native **CRM-Weiterleitung (Formular→Lead)** · Webhooks · QR-Code · Partial Submissions · Drop-off-Analytics · eigene Subdomain. KANN: Berechnungen, E-Signatur, KI-Generator.

## Umsetzungsplan formulare
**Phase 1 — Lead-Capture scharf (KRITISCH):** öffentlicher unauth Submit-Endpoint (🔴) + echter Share-Link/Embed-UI + Veröffentlichungs-Flow + Danke-Seite/Redirect. 🟡 + 🔴
**Phase 2 — Benachrichtigung & CRM-Brücke:** E-Mail bei neuer Submission + Autoresponder + **Submission→CRM-Lead** (Formular-Aktionen) + Webhook-UI (BE fertig). 🟡 + 🔴 (Mail-Versand)
**Phase 3 — Builder-Tiefe:** DnD (dnd-kit) · Email-Feldtyp einblenden · Feldtyp-Wechsel · Validierungsregeln. 🟡
**Phase 4 — Submissions-Tiefe:** Export verdrahten (Hook da) · Status-Filter · Pagination · Statistik echt. 🟡
**Phase 5 — DSGVO:** Consent-Feldtyp · IP-Opt-out · Datenschutz-Link · Spam-Schutz (reCAPTCHA/Honeypot) · Submission-Löschung. 🟡 + 🔴
QA-Pass (Teil G) je Phase.

## Backend-Gaps formulare (Luke → backend-gaps.md)
🔴 **Öffentlicher unauth Submit-Endpoint** (KRITISCH) · Submission→CRM-Lead-Hook · E-Mail-Benachrichtigung + Autoresponder-Versand · Consent-Feldtyp + IP-Opt-out · Spam-Schutz · Submission-Delete (DSGVO). (Webhooks, Export, Schemas, Stats existieren — nur FE-Anbindung.)

---

# MODUL-AUDIT 5: dokumente (Beratungsdoku, Kundenfreigaben)

## Ist-Befund (UI-Tiefe)
**Solide:** Ordnerbaum (Personal/Team/Project) ✅, Breadcrumb ✅, Grid/Listen-Toggle ✅, Drag&Drop-Upload + Progress ✅, Ordner CRUD ✅, Datei umbenennen/löschen ✅, Favoriten ✅, Multi-Select (Ctrl/Shift) ✅, Kontextmenüs ✅, Versionierung (Historie/Restore/Auto-WOPI) ✅, OnlyOffice/WOPI-Editor (Co-Editing, Lock) ✅, interne Freigaben (read/write) ✅, Vorschau (Bild/PDF/Text/Video) ✅.

**Kritische Lücken:**
- **Externe Share-Links = 100% Mock** (`generateMockLink`, Ablauf/Passwort/Permission-UI ohne Backend) — kein public-Token-Endpoint. Für Kundenfreigaben (DSGVO) zentral.
- **Kommentare zu Dateien** fehlen komplett (FE+BE).
- **„Mit mir geteilt"-Ansicht rendert nichts** (`_sharedData` verworfen) · **Virtual-Folder** (Chat/Email/Task-Anhänge) rendern nichts (`_virtualXFiles` verworfen).
- **Bulk-Aktionen** fehlen (Multi-Select da, keine Delete/Move/Share-Toolbar).
- **Verschieben/Kopieren per Menü = Stub** (`moveComingSoon`/`copyComingSoon`; `useCopyFile`+BE da) · Ordner-Verschieben kein UI (BE bereit).
- **Suche** clientseitig (geladene Dateien) statt `useSearchFiles`-Backend (existiert, ungenutzt) · **keine Sortier-UI** (State eingefroren).
- Download-Button im FileDetailPanel ohne onClick (Bug) · Versions-Download nur Toast · „In Office öffnen" nur Toast · Template-Galerie erstellt keine echte Datei.
- User-Picker im ShareDialog = rohe User-ID (kein Autocomplete) · Tag-Quick-Add nur „erster Tag".

## Markt-Soll (Nextcloud/SharePoint/Drive/ecoDMS)
MUSS: Ordnerbaum + Gruppenrechte + Ordner-Freigabe · Auto-Versionierung + Restore · **File-Drop/Upload-Link (Kunde lädt hoch) + Passwort + Ablauf (erzwingbar)** · **Admin-Übersicht aller externen Links** · OCR-Volltextsuche · **Audit-Log (intern+extern, unveränderlich)** · EU/Self-Hosted + Verschlüsselung · Papierkorb + Lösch-Schutz · Bulk-Aktionen · MFA/Passwort-Policy. SOLLTE: Versionsvergleich · Secure-View (Lesen ohne Download) · Tags/Metadaten · E-Signatur-Integration · Aufbewahrungsfristen (GoBD 6/10J). KANN: Wasserzeichen · KI-Klassifizierung · Duplikat-Erkennung.

## Umsetzungsplan dokumente
**Phase 1 — Externe Freigaben (DSGVO, KRITISCH):** echte Share-Links (Token, Passwort, Ablauf, Permission) + **Upload-Link/File-Drop** + Admin-Link-Übersicht. 🟡 + 🔴 (public-Token-Backend)
**Phase 2 — Mock→API + tote Ansichten:** „Mit mir geteilt" + Virtual-Folder rendern · Suche auf `useSearchFiles` · Verschieben/Kopieren-Dialog (Hooks/BE da) · Ordner-Verschieben-UI · Sortier-UI · Download-Bug-Fix. 🟡
**Phase 3 — Bulk + Tiefe:** Bulk-Toolbar (Delete/Move/Share/Tag) · User-Picker im Share · Tag-Picker · Versions-Download/-Vergleich. 🟡
**Phase 4 — Kommentare + Compliance:** Datei-Kommentare/Annotation · Audit-Log (Zugriffe/Freigaben) · Aufbewahrungsfristen · Signatur-Verknüpfung (→ vertraege). 🟡 + 🔴
QA-Pass (Teil G) je Phase.

## Backend-Gaps dokumente (Luke → backend-gaps.md)
🔴 Externe Share-Link-Tokens (public-Resolve, Passwort-Hash, Ablauf, Download-Limit) · **Upload-Link/File-Drop-Endpoint** · Datei-Kommentar-Endpoints · Audit-Log (Zugriff/Freigabe, unveränderlich) · Aufbewahrungsfristen-Job · OCR-Volltextsuche (falls nicht vorhanden). (Versionierung, WOPI, interne Shares, Move/Copy, Suche-Endpoint, Favoriten existieren — FE-Anbindung.)

---

# MODUL-AUDIT 6: vertraege (ZFA — Mandatsverträge)

## Ist-Befund (UI-Tiefe)
**Ganze Seite auf Mock-Store** (`useVertraegeStore`) — alle 13 fertigen `useVertraege.ts`-Hooks + Client + Backend ungenutzt. UI-Gerüst aber reich: Tabs (Aktiv/Auslaufend/Archiv/Vorlagen) ✅, Liste mit Suche + Typ-Filter ✅, Detail-Panel (Laufzeit-Progress + Kündigungsfrist-Marker, Konditionen, History-Timeline) ✅, Anlegen-Dialog (6 Typen, Reminder-Checkboxen 30/60/90) ✅, ESignaturDialog (Unterzeichner, Sequenz/Parallel, Status-Flow) ✅, Stats-Row ✅.

**Alles Mock/lokal:** kein echter Persist; Dokument-Upload = `MOCK_DOCUMENTS` (Service-Stub TODO MinIO); Signatur nur lokal (Skribble = Phase-D-Placeholder); **kein Reminder-CRUD-UI** (nur Checkboxen; Worker läuft, Delivery nicht verdrahtet); **kein Parteien-UI** (Hook+BE da); **kein Export-Button** (Route da, Plain-Text); **kein Loading/Error-State**; Status-/Typ-Enums FE↔BE divergieren (draft fehlt im Store).

## Markt-Soll (ContractHero/Juro/fynk + Skribble)
KRITISCH: **Fristenüberwachung + gestaffelte Kündigungs-Erinnerungen** (90/60/30 T, Fälligkeits-Dashboard, Auto-Verlängerungs-Warnung) · **E-Signatur (eIDAS/FES)** extern + Protokoll. HOCH: Liste (Status/Typ/Gegenpartei/Laufzeit-Filter, Volltextsuche) · Detail (Metadaten, PDF-Viewer, Versionen, Parteien, Historie) · Audit-Log + DSGVO/EU. MITTEL: Freigabe-Workflow (1–2 Stufen) · QES (Skribble) · OCR + KI-Metadaten-Extraktion. NIEDRIG: KI-Risiko, AI-Draft.

## Umsetzungsplan vertraege
**Phase 1 — Mock→API (Fundament):** Seite von Store auf `useVertraege.ts`-Hooks (Liste/Detail/CRUD) + Loading/Error + Status/Typ-Mapping FE↔BE. 🟡
**Phase 2 — Fristen & Reminder (KRITISCH):** Reminder-CRUD-UI (gestaffelt) + Fälligkeits-Dashboard (alle Verträge) + Auto-Verlängerungs-Warnung. 🟡 + 🔴 (Delivery-Worker)
**Phase 3 — Dokumente & Parteien:** echter Upload (MinIO) + Viewer + Versionen + Parteien-Management. 🟡 + 🔴 (UploadDocument-Stub)
**Phase 4 — Signatur & Export:** E-Signatur echt (Skribble/eIDAS) + Protokoll + PDF-Export. 🔴 + 🟡
**Phase 5 — Compliance:** echtes Audit-Log (nicht hardcoded) + Aufbewahrungsfristen + OCR-Suche. 🟡 + 🔴
QA-Pass (Teil G) je Phase.

## Backend-Gaps vertraege (Luke → backend-gaps.md)
🔴 UploadDocument-MinIO (Stub) · Reminder-Delivery-Worker (Mail) · E-Signatur-Provider (Skribble/eIDAS + Webhook) · Audit-Log/contract_events · PDF-Export-Renderer · OCR-Volltextsuche. (Contract/Party/Reminder-CRUD + Export-Route existieren — FE-Anbindung.)

---

# MODUL-AUDIT 7: mails (Kundenkommunikation)

## Ist-Befund (UI-Tiefe)
**Client-Kern solide:** 2–3-Spalten-Layout (Ordner-Sidebar/Liste/Lese-Pane) ✅, Ordner aus API + Unread-Badges ✅, Verfassen/Reply/Reply-All/Forward ✅ (Rich-Text, To/CC/BCC mit CRM-Autocomplete, Entwürfe, Electron-Pop-out), Sync-Button + 30s-Polling ✅, Volltextsuche ✅, Empty/Loading-States ✅, DSGVO-Aufbewahrungsfristen-Anzeige ✅ (Frontend-Heuristik, differenzierend).

**Lücken/Stubs:**
- **Konto-Einrichten speichert NICHT** — `MailServerSettingsTab` Speichern = nur Toast, `useCreateEmailAccount` etc. ungenutzt; Mitarbeiter-Account-Liste = Mock (`EMPLOYEES`).
- **Multi-Account-Switcher + Unified Inbox fehlen** (1 Account/User).
- **Signaturen:** aus lokalem Store, voll Signature-CRUD-API ungenutzt; Firmen-Signatur-Speichern = Toast.
- **Vorlagen:** 6 hardcoded, kein CRUD, keine Platzhalter-Substitution, kein BE.
- **Regeln & Filter:** fehlen komplett (FE+BE) · **Suchfilter** (ungelesen/Anhang/Absender) fehlen · Sortier-UI hardcoded.
- **Thread-Rendering fehlt** (Backend-Threads + `useEmailThread` da, flache Liste im UI).
- Attachment-Upload (leerer Button) + Download (Toast) + Drucken/PDF (Toast) = Stubs.
- E-Mail↔Kontakt nur lesbar (Link-Badge); `useLinkEmailToContact` ungenutzt (kein Verknüpf-Button).
- Bulk-Aktionen + Pagination-UI fehlen · EWS/OAuth2/PGP/S/MIME fehlen · AI-Draft = Mock.

## Markt-Soll (Outlook/eM Client/Mailbird/Thunderbird)
P0: 3-Spalten + Konversations-/Thread-Ansicht · IMAP/SMTP + **OAuth2** + Exchange/EWS · 2+ Konten · Rich-Text-Compose + Anhänge + Entwürfe + Undo-Send · **Signaturen pro Konto** · Volltext + Grundfilter · **E-Mail↔Kontakt-Verknüpfung + Auto-Logging + Kontext-Panel** (Kern-Mehrwert) · TLS/OAuth2/Tracking-Block. P1: Vorlagen/Schnellbausteine · Regeln/Filter (serverseitig) · Snooze/Tags · Unified Inbox + Account-Switcher · S/MIME · Scheduled Send · Aufgabe/Termin aus E-Mail. P2: KI-Compose, PGP, Legal-Hold.

## Umsetzungsplan mails
**Phase 1 — Konto + Signatur scharf:** Konto-Setup an API (Create/Update/Test, OAuth2) · Multi-Account-Switcher · Signatur-CRUD-UI (statt Store). 🟡 + 🔴 (OAuth2, EWS)
**Phase 2 — CRM-Verknüpfung (Kern-Mehrwert):** E-Mail↔Kontakt verknüpfen/trennen-UI · Auto-Logging am Kontakt · Kontext-Panel · Aufgabe/Termin aus E-Mail. 🟡
**Phase 3 — Produktivität:** Vorlagen-CRUD + Platzhalter-Substitution (🔴 BE) · Regeln/Filter (🔴 BE) · Suchfilter · Thread-Rendering (Hook da) · Bulk + Pagination. 🟡 + 🔴
**Phase 4 — Anhänge & Compliance:** Attachment-Upload/Download verdrahten · Tracking-Block · S/MIME · serverseitige Archivierung. 🟡 + 🔴
QA-Pass (Teil G) je Phase.

## Backend-Gaps mails (Luke → backend-gaps.md)
🔴 Multi-Account (`ListEmailAccounts`) · OAuth2 + Exchange/EWS · Vorlagen-CRUD-Endpoint + Platzhalter · Regeln/Filter-Engine · Suchfilter-Params (is_unread/has_attachments/from) · S/MIME · serverseitige Archivierung. (IMAP/SMTP, Signature-CRUD, Thread-Logik, Contact-Link, Sync, Suche existieren — FE-Anbindung.)

---

# MODUL-AUDIT 8: helpdesk (Kundenservice, hinter Feature-Flag)

## Ist-Befund (UI-Tiefe)
**Komplette Mock-Wall:** ALLE `useHelpdesk.ts`-Hooks (13+) + Backend ungenutzt; FE 100% über `useHelpdeskStore` (Mock). UI-Gerüst reich: Tabs (Tickets/Wissensdatenbank/Statistik) ✅, Ticket-Liste mit Filtern (clientseitig) ✅, Detail-Panel (SLA-Badge, Status-Dropdown, Nachrichtenthread, interne Notizen, Canned-Picker, CSAT, AI-Vorschlag) ✅, Canned-Panel, BusinessHoursDialog, RoutingConfig, Statistik-Charts ✅ — aber ALLES Toast/Mock, kein API-Call.
**Kein Backend für:** Knowledge-Base, CSAT, Routing-Regeln, Multi-Channel, Zeiterfassung. **Fehlt:** Sortier-UI, Bulk, Pagination, Zuweisen-Button, Merge-UI, Einstellungen-Tab.
**Diskrepanzen:** Status-Enums FE↔BE (kaum Overlap), Prioritäts-Enums divergieren, Canned-Schema FE↔BE, Merge-URL inkonsistent, SLA hardcoded auf 2026-02-15 (friert ein).

## Markt-Soll (Zammad/Zendesk/Freshdesk)
MUSS: **E-Mail→Ticket + Multi-Postfach** · Queue-Ansichten (Meine/Offene/Überfällige) + Spalten/Filter · **öffentl. Antwort vs. interne Notiz (DSGVO)** · Status/Priorität/Zuweisung/Tags · Kunden-/Org-Zuordnung · **Audit-Trail** · Anhänge · SLA + Business-Hours + Frist-Anzeige + Eskalation · Canned Responses · Trigger · Rich-Text. SOLLTE: @Mentions, verknüpfte Tickets, Collision-Warning, Merge, **Zeiterfassung/Ticket (Honorar)**, Mandanten-Portal, CSAT, SLA-Reporting. KANN: KB öffentlich, KI-Vorschläge, CTI, Chat-Widget. OVERKILL: Social Media, Predictive.

## Umsetzungsplan helpdesk
**Phase 1 — Mock→API:** Seite von Store auf `useHelpdesk.ts`-Hooks (Liste/Detail/CRUD/Messages) + Enum-Mapping FE↔BE + Loading/Error. 🟡
**Phase 2 — Ticket-Arbeit:** Antworten (öffentlich/intern) · Status/Zuweisung/Tags · Audit-Trail · Anhänge · Merge-UI · Sortier/Bulk/Pagination. 🟡
**Phase 3 — SLA & Automatisierung:** SLA live aus BE (nicht hardcoded) · Business-Hours-API · Eskalation · Canned-CRUD · Trigger. 🟡 + 🔴
**Phase 4 — Multi-Channel & Mehrwert:** E-Mail→Ticket (mails-Verknüpfung) · Zeiterfassung/Ticket · CSAT · Knowledge-Base · Mandanten-Portal · Reporting. 🟡 + 🔴
QA-Pass (Teil G) je Phase.

## Backend-Gaps helpdesk (Luke → backend-gaps.md)
🔴 Knowledge-Base-Endpoints · CSAT-Modell+Endpoint · Routing-Regeln-Engine · Multi-Channel (E-Mail→Ticket, source_channel) · Zeiterfassung-Feld · Business-Hours-SLA-Berechnung ("not yet implemented") · Statistik-Endpoint · Enum-Angleichung FE↔BE. (Ticket/Queue/Canned/SLA-Policy/Merge-CRUD existieren — FE-Anbindung.)

---

# MODUL-AUDIT 9: finanzen (Angebote/Rechnungen — Vollersatz-Ziel)

## Ist-Befund (UI-Tiefe)
**Solider Kern verdrahtet:** Dashboard (8 Metriken) ✅, Angebote (QuoteFormDialog mit Positionen/Steuersätzen/Kleinunternehmer, Versand/Akzept/Ablehnung/Convert/PDF) ✅, Rechnungen (InvoiceFormDialog, Länder/Währung-UI, Status-Filter, Versand/Zahlung/Storno/PDF) ✅, Gutschriften ✅, Mahnwesen (DunningPanel 3-stufig, Detect/Send/Escalate/PDF/Config) ✅, DATEV-Export ✅.
**Mock/Stub/fehlt:** E-Rechnung-Indikator = Mock (ZUGFeRD-BE existiert via `?format=zugferd`, FE ruft nie auf; **XRechnung-UBL fehlt im BE**) · Banking/Belegkette/Hours-to-Invoice/expenses/transactions = Mock (kein BE) · Währung/taxCountry werden NICHT ans BE übergeben · **wiederkehrende Rechnungen fehlen** · Rabatt-Feld fehlt · Kontakt-Picker fehlt (Freitext) · GoBD-Hooks (Lock/Journal/Validate/Stats/Export) ungenutzt (kein UI) · Bexio/BMD-Export = Toast.

## Markt-Soll (sevdesk/Lexware/WISO)
Fluss: Kundenstamm (aus CRM) → Angebot → 1-Klick → Rechnung → **E-Rechnung XRechnung+ZUGFeRD (Pflicht 2025)** → Zahlungsabgleich+Mahnwesen → DATEV→Steuerberater. MUSS: Positionen+Steuer+Rabatt · Angebot→Rechnung · GoBD-Nummern+Archiv · XRechnung+ZUGFeRD versenden+validieren · Storno/Gutschrift · §19 Kleinunternehmer · DATEV-Export + Steuerberater-Zugang · **CRM-Datenbasis (kein Silo)**. SOLLTE: wiederkehrende/Teil-Rechnungen · Open-Banking-Abgleich · Reverse-Charge · Skonto · Leistungskatalog. KANN: Fremdwährung, BWA, Zahlungs-QR.

## Umsetzungsplan finanzen
**Phase 1 — E-Rechnung scharf (Pflicht 2025, KRITISCH):** EInvoice-Indikator an echten ZUGFeRD-Endpoint + **XRechnung-UBL** (BE-Neubau) + Validierung + XML-Download. 🟡 + 🔴
**Phase 2 — Stammdaten-Brücke:** Kontakt-Picker aus CRM + Leistungskatalog + Rabatt + Währung/taxCountry ans BE. 🟡 + 🔴
**Phase 3 — Wiederkehrend & Zahlung:** wiederkehrende/Teil-Rechnungen + Open-Banking-Zahlungsabgleich (ersetzt BankingWidget-Mock). 🔴 + 🟡
**Phase 4 — GoBD & Brücke:** GoBD-UI (Lock/Journal/Validate) + Belegkette (BE) + Hours→Invoice echt (zeiterfassung) + Steuerberater-Zugang. 🟡 + 🔴
QA-Pass (Teil G) je Phase.

## Backend-Gaps finanzen (Luke → backend-gaps.md)
🔴 **XRechnung-UBL-Serializer** · wiederkehrende Rechnungen (Scheduler) · Open-Banking/Zahlungsabgleich (FinAPI) · Belegkette-Aggregation · currency/taxCountry-Mapping · Hours→Invoice-Bridge. (Angebot/Rechnung/Gutschrift/Mahnwesen/DATEV/ZUGFeRD-CII/GoBD existieren — FE-Anbindung.)

---

# MODUL-AUDIT 10: berichte (BI/Reporting, hinter Feature-Flag)

## Ist-Befund (UI-Tiefe)
**Verdrahtet:** Tabs (Dashboards/Bericht-erstellen/Geplant/DATEV) ✅, KPI-Dashboard (Recharts, Modul-Filter, 4 KPIs aus CRM/Finanzen/Helpdesk/Inventar) ✅, Export (PDF/CSV/XLSX, echte Libs) ✅, ScheduleList (CRUD/Toggle/Cron-Freitext/E-Mail-Empfänger + Backend-Scheduler mit Mailer) ✅, DATEV-BWA-View ✅.
**Fehlt/Mock:** **No-Code Query-Builder fehlt** (ReportBuilder = nur Format+Datum; custom-Kind aus UI gefiltert) · Drill-Down nur Placeholder · **cross-Modul-Executor fehlt** (cross im Dropdown → Server-Fehler) · Breakouts/Pivot fehlen · Schedule-Edit-UI fehlt · **Alert-Schwellwerte fehlen** · datev_susa-Executor fehlt · DATEV-Variant-Switcher funktionslos · kein Skeleton/Error-State.

## Markt-Soll (Metabase/PowerBI/Superset/Looker)
MUSS: Drag-Dashboard + KPI-Cards + 5 Chart-Typen + Tabs · **No-Code-Builder** (Datenquelle/Felder/Aggregation/Group-by/Filter) · Drill (Klick→Detail) + Cross-Filter · Datums-+Kategorie-Filter · CSV+PDF + Link-Sharing · **zeitgesteuerter Versand + Schwellwert-Alert** · zentrale Datenquelle + Excel. SOLLTE: Pivot · Perioden-Vergleich · Zeit-Granularität · modulübergreifend · XLSX · iFrame-Embed. KANN: RLS, Echtzeit-Alert.

## Umsetzungsplan berichte
**Phase 1 — No-Code-Builder (Kern-Lücke):** visueller Query-Builder + custom-Definitionen erstellen/bearbeiten (Mutations da). 🟡 + 🔴 (query_config-Contract)
**Phase 2 — Dashboard-Tiefe:** Drill-Down echt (pro-KPI-Run) · Cross-Filter · Pivot/Breakout · Perioden-Vergleich · Zeit-Granularität. 🟡 + 🔴
**Phase 3 — Alerts & Schedule:** Schwellwert-Alerts · Schedule-Edit · Cron-Builder · Slack/Teams. 🟡 + 🔴
**Phase 4 — Cross-Modul + DATEV:** cross-Executor · datev_susa · Variant-Switcher-Fix · Skeleton/Error. 🟡 + 🔴
QA-Pass (Teil G) je Phase.

## Backend-Gaps berichte (Luke → backend-gaps.md)
🔴 query_config-Interpretation (No-Code) · cross-Modul-Executor · breakout/pivot-Schema · datev_susa-Executor · Schwellwert-Alert-Engine · produktion-KPI. (Export, Scheduler+Mailer, KPI-Aggregation, DATEV-BWA, CRUD existieren — FE-Anbindung.)

---

# MODUL-AUDIT 11: team (HR/Mitarbeiter)
## Ist-Befund
Verdrahtet ✅: Mitgliederverwaltung (CRUD+Wizard), Organigramm, Abwesenheits-Kalender, HR-Admin (Urlaubs-Genehmigung, Zeitkorrekturen), HR-Settings.
Mock/ungenutzt: **Digitale Personalakte** (Hooks+BE da, UI nutzt MOCK_DOCUMENTS) · **Onboarding-Workflows** (kein BE) · Self-Service-View (Mock) · **DATEV-HR-Lohn** (kein BE; route_datev_upload = Finance-Buchungen, NICHT Lohn) · Modul-Zuweisung (User aus MOCK_USERS, Grant/Revoke in-memory) · Schulungen (Mock). Kein Deaktivieren/Delete-Endpoint (leerer Update-Body). Org-Detail E-Mail/Anruf = Toast.
## Markt-Soll (Personio/HiBob/Factorial)
MUSS: GoBD-Personalakte + Custom-Felder + Dok-Upload · Urlaubsantrag+Genehmigung+Resturlaub+eAU · Teamkalender + Abwesenheitstypen · Onboarding/Offboarding-Checklisten · RBAC + Feldrechte (Gehalt nur HR) · Arbeitszeiterfassung (BAG-Pflicht) · **native DATEV-Lohn (LODAS) + Stammdaten/Abwesenheits-Export**. SOLLTE: ESS, Manager-Self-Service, Delegationsrechte, Audit-Log, Überstunden-Saldo, e-Signatur. KANN: LMS, Org-Export, Geo-Clock-in.
## Umsetzungsplan team
P1 Personalakte an API (Hooks da) + Deaktivieren-Endpoint 🟡+🔴 · P2 Onboarding/Offboarding (UI+BE) + Self-Service an API 🟡+🔴 · P3 RBAC-Feldrechte (Gehalt) + Modul-Zuweisung echtes BE 🟡+🔴 · P4 DATEV-HR-Lohn (LODAS) + Arbeitszeit (Zeiterfassung-Bridge) + e-Signatur 🔴+🟡. QA-Pass je Phase.
## Backend-Gaps team (Luke)
🔴 Onboarding/Offboarding-API · DATEV-HR-Lohn (LODAS, getrennt von Finance-DATEV) · Modul-Zuweisung-Endpoint · Mitarbeiter-Deaktivieren · Self-Service-Aggregat · eAU · Feldrechte. (Employee-CRUD, Absence, Leave-Approval, Korrekturen, Personalakte-Endpoint existieren.)

---

# MODUL-AUDIT 12: kommunikation (chat + inbox → EIN Modul)
## Ist-Befund
chat ✅: Channels (öffentl/privat), DMs, Threads (real-time WS), @Mentions-Autocomplete. Suche BE da, kein UI.
inbox ✅: Unified Inbox (3-spaltig), Routing-Rules-Editor (AND/OR, 4 Actions, Test).
Bugs/Mock/fehlt: **Reaktionen = Mock + `useReactions.ts` importiert falsch aus `video-client`** · **File-Upload fehlt end-to-end** (UI da, kein POST /files) · Chat-Suche-UI fehlt · Mentions-Panel fehlt (BE-Route da) · Video-Call-Integration fehlt · Bots fehlen · **ChannelSettingsDialog 100% Mock** (kein Kanal-Linking-BE — kritisch für Merge!) · WhatsApp kein eigener BE-Kanal (auf 'chat' gemappt) · beide getrennt in Nav.
## Markt-Soll (Slack/Mattermost + Crisp/Chatwoot)
Hinweis: kein Tool macht beides exzellent — Merge ist ambitioniert. Team-Chat MUSS: Channels/DMs/Threads/@Mentions/Datei-Sharing/Suche. SOLLTE: Reaktionen/Pin/Presence. Call: 1:1+Gruppen-Video aus Chat (MUSS), Screen-Share (SOLLTE). Unified Inbox MUSS: zentrale Inbox · E-Mail+Web-Widget+**WhatsApp** · Zuweisung/Status/interne Notizen/Kollisionsschutz/Canned. SOLLTE: Round-Robin/Tags/Kundenprofil. Kanal-Verwaltung MUSS: E-Mail/WhatsApp/Widget anschließen (Self-Service). Bots: Autoresponder (MUSS).
## Umsetzungsplan kommunikation
P1 Merge zu einem Modul/Nav (Team-Chat-Tab + Kunden-Inbox-Tab) 🟡 · P2 Bugs: Reaktionen an echten Chat-Endpoint (Import-Fix) + File-Upload end-to-end (🔴 Route) + Chat-Suche-UI + Mentions-Panel 🟡+🔴 · P3 Kanal-Verknüpfungen verwaltbar (🔴 Linking-API E-Mail/WhatsApp/Widget) + WhatsApp-Adapter 🔴 · P4 Video-Call-Bridge (meetings) + Kollisionsschutz + Canned + Autoresponder 🟡+🔴. QA-Pass je Phase.
## Backend-Gaps kommunikation (Luke)
🔴 Chat-Reaction-Endpoint · **File-Upload-Route POST /channels/{id}/files** · Kanal-Linking-API (externe Kanäle) · WhatsApp-Adapter · Bot/Autoresponder · Video-Call-Bridge. (Channels/DMs/Threads/Suche/Inbox/Routing existieren — FE-Anbindung + Merge.)

---

# MODUL-AUDIT 13: meetings (Video, kanonisch; `modules/video` = verwaist)
## Ist-Befund (aus Welle-2-Tiefenanalyse)
**Am ausgereiftesten:** LiveKit Video/Audio-Call (Gallery/Speaker) ✅, Screen-Sharing ✅, Recording (Egress-Webhook) ✅, Lobby/PreJoin/Moderation ✅, **Consent-Banner DSGVO vorbildlich** (nicht wegklickbar, Decline→Blur/Mute, Snapshot in DB) ✅, WebRTC browser-basiert ✅, self-hosted (LiveKit+coturn Hetzner) ✅, MeetingFormDialog + Kalender-Verknüpfung.
Fehlt/Mock: **Breakout-Räume** (LiveKit-fähig, kein UI/BE) · **Recording-Download/List-UI** (Egress speichert file_url, kein Listen-UI) · Meeting-Recurrence (Icons da, Logik unklar) · `modules/video/VideoPage` = verwaiste Route → Cleanup · externer Gäste-Link unklar.
## Markt-Soll (Zoom/Teams/Jitsi)
MUSS: Video/Audio (WebRTC) · Screen-Share · Lobby/Moderation · Consent/Aufnahme-Hinweis · self-hosted/EU · Meeting planen + Kalender · Einladung. SOLLTE: Recording + Download-Archiv · Breakout · externer Gäste-Link (ohne Account) · Transkript. KANN: virtuelle Hintergründe, Whiteboard, Live-Reactions, Streaming.
## Umsetzungsplan meetings
P1 Cleanup verwaiste `modules/video`-Route 🟡 · P2 Recording-Archiv-UI (Liste/Download, file_url da) + externer Gäste-Link 🟡+🔴 · P3 Breakout-Räume (LiveKit-fähig) + Recurrence 🔴+🟡 · P4 Transkript/Notizen. QA-Pass je Phase.
## Backend-Gaps meetings (Luke)
🔴 Breakout-Räume-RPC · Recording-List/Download-Endpoint · Recurrence-Logik · öffentl. Gäste-Join-Link. (LiveKit-Call/Screen/Recording/Consent/Lobby/Egress existieren — sehr vollständig.)

---

# MODUL-AUDIT 14: work (Projekte/Aufgaben)
## Ist-Befund (aus Welle-2)
⭐ Sehr vollständig & verdrahtet: Aufgaben (Zuweisung/Fälligkeit) ✅, Kanban (dnd-kit, optimistisch) ✅, Listen-Ansicht ✅, Gantt (kritischer Pfad, Zoom) ✅, Abhängigkeiten+Teilaufgaben ✅, Kommentare+Anhänge ✅, automatisierte Regeln (via automation) ✅, Zeiterfassung integriert (TaskTimer + route_work_time) ✅.
Fehlt: **Projekt-Portfolios** (kein BE/UI) · Gantt nur due_date, **kein start_date** im Task-Modell (Balken schätzend) · 500-Task-Hardlimit im Gantt.
## Markt-Soll (Asana/Monday/factro)
MUSS: Aufgaben+Zuweisung+Fälligkeit · Kanban · Liste · Abhängigkeiten/Teilaufgaben · Kommentare/Anhänge · Zeiterfassung. SOLLTE: Gantt mit start_date · Projekt-Portfolios · Auto-Regeln · Kapazitäts-/Auslastungsplanung · wiederkehrende Aufgaben · Vorlagen. KANN: Workload-Charts, OKR, Budget/Projekt, Kalender-Ansicht.
## Umsetzungsplan work
P1 start_date ins Task-Modell → echter Gantt 🟡+🔴 · P2 Projekt-Portfolios (Entität+UI) 🔴+🟡 · P3 Vorlagen + wiederkehrende Aufgaben + Kapazitätsplanung 🟡+🔴. QA-Pass je Phase.
## Backend-Gaps work (Luke)
🔴 start_date-Feld · Projekt-Portfolio-Entität+Aggregation · wiederkehrende Aufgaben · Kapazitätsplanung. (Task-CRUD/Kanban/Dependencies/Subtasks/Kommentare/Files/Time-Tracking/Automation existieren — eines der vollständigsten Module.)

---

# MODUL-AUDIT 15: zeiterfassung
## Ist-Befund
Hinweis: ZeiterfassungPage = Wrapper auf ZeiterfassungTab (Profil-Modul). Verdrahtet ✅: Clock-In/Out, Pause, Live-Timer, **§3-ArbZG-Compliance (Backend arbzg.go)**, Tages-/Wochensaldo, Korrekturantrag + Genehmigung. Mock (useTimeTrackingStore): **manuelle Einträge, Reports, Export (nur Toast!), Kategorien-CRUD, Arbeitsziele, Team-Übersicht, Überstundensaldo**. **Kunde/Projekt/Leistung = null** (keine Verknüpfung zu CRM/Work). §5-Ruhezeit-Verletzung + autoBreakDeducted nie im UI. Polling 5 min (zu träge für Stechuhr).
## Markt-Soll (clockodo/Papershift/Harvest)
MUSS: Timer + manuelle Einträge · §3-ArbZG · Zuordnung Kunde/Projekt/Leistung · Auswertung + Export (CSV/**DATEV-Lohn**). SOLLTE: Stundenkonto/Saldo (kumuliert) · Pausenregeln · Manager-Teamübersicht · Arbeitszeitnachweis-PDF (§17 MiLoG). KANN: GPS-Stempel, Geofencing.
## Umsetzungsplan zeiterfassung
P1 manuelle Einträge + Reports + Export an echte API (statt Mock-Store) + Kunde/Projekt/Leistung-Verknüpfung 🟡+🔴 · P2 Stundenkonto-Saldo-Endpoint + Manager-Teamübersicht (useEmployees) 🟡+🔴 · P3 §5-Anzeige + autoBreakDeducted + Polling-Fix + Arbeitszeitnachweis-PDF 🟡+🔴. QA-Pass.
## Backend-Gaps zeiterfassung (Luke)
🔴 manuelle Zeiteintrag-Endpoint (/entries) · Export-API (CSV/DATEV-Lohn) · Stundenkonto-Saldo (kumuliert) · Projekt/Kunde/Leistung im Worktime-Schema · Team-Übersicht-Aggregat. (Clock-In/Out, Pause, Summaries, Korrekturen, ArbZG existieren.)

# MODUL-AUDIT 16: wiki
## Ist-Befund
**Komplett Mock-Store** (`useWikiStore`) — gesamte `useWiki.ts` (8 Queries, 9 Mutations) + `wiki-client.ts` (16 Fns gegen /api/v1/wiki) ungenutzt. **Editor = `<textarea>` + String-Concat** (kein TipTap, obwohl Typen TipTapContent vorsehen). Versionsverlauf zeigt nur an (kein Restore-Button trotz Hook). Volltextsuche = client-seitiger Filter (FTS-Backend + Hook ungenutzt). Kategorien Mock. Baum nur 1 Ebene (parent_id ignoriert). Share-Link hardcoded (`cosmi://wiki/{slug}`, Token-Route fehlt). Anhänge: Hooks da, kein UI. Kommentare fehlen ganz. Templates Mock.
## Markt-Soll (Confluence/Notion/BookStack)
MUSS: Rich-Editor (TipTap) · Versionsverlauf + Restore · Volltextsuche · verschachtelte Struktur · Inline-Anhänge. SOLLTE: Kategorien/Labels · Share-Links (intern/extern) · Berechtigungen pro Artikel · Templates · Kommentare. KANN: @Mentions, Export PDF/Word, Diff-Vergleich.
## Umsetzungsplan wiki
P1 ALLES von Store auf `useWiki.ts`-Hooks (Liste/Detail/CRUD/Suche) 🟡 · P2 Editor → TipTap (RichTextEditor wiederverwenden) + echte verschachtelte Struktur (parent_id) 🟡 · P3 Versions-Restore-Button + Anhänge-UI + Share-Token (🔴 Route registrieren) + Berechtigungen 🟡+🔴 · P4 Kommentare 🔴+🟡. QA-Pass.
## Backend-Gaps wiki (Luke)
🔴 Share-Token-Route in route_wiki.go registrieren + öffentl. Read · Kommentar-Endpoints · Artikel-Templates-Endpoint · Berechtigungen pro Artikel. (CRUD, Versionen, FTS, Kategorien, Anhänge-Endpoints existieren — reine FE-Anbindung, größter „Hook-da-UI-fehlt"-Fall.)

# MODUL-AUDIT 17: automatisierung (Brücke/„Zapier-Light", KMU-tauglich)
## Ist-Befund
**Stark & verdrahtet:** 17 Trigger (CRM/Work/Email/Biz/HR/Calendar/Dialer) ✅, 8 Action-Typen ✅, Bedingungen (rekursive AND/OR + Expression mit Live-Test) ✅, 12 Templates in 4 Kategorien ✅, Template-Galerie ✅, **Wizard + visueller React-Flow-Editor** (beide, gemeinsamer State) ✅, Execution-Log (expandierbar) ✅, Dry-Run/Simulation ✅, Circuit-Breaker (100/h) ✅.
Fehlt/Stub: **Branching/Verzweigung** (Engine rein sequenziell) · **http_request-Action (ausgehend)** fehlt · **Webhook-Trigger (eingehend)** fehlt · `notification.send` = Log-Stub (kein echter RPC) · Cron nur hardcoded 5-Min-Poller (kein freier Cron) · Dialer fehlt in UI-Modul-Labels · 1 Template-Inkonsistenz.
## Markt-Soll (Zapier/Make/n8n)
MUSS (KMU): Trigger/Action · if/then-Conditions · Templates · Multi-Step · Execution-Log. SOLLTE: **Verzweigungen** · **Webhooks (in+out)/HTTP-Action** · zeitbasierte/Cron-Trigger · Retry-Logik. KANN: n8n-Niveau (Sub-Flows, parallele Pfade, Code-Nodes) — Overkill für KMU.
## Umsetzungsplan automatisierung
P1 http_request-Action (out) + Webhook-Trigger (in) 🔴+🟡 · P2 Branching/Verzweigung im Engine + Editor 🔴+🟡 · P3 freier Cron-Trigger + notification.send echter RPC + Dialer-Labels + Template-Fix 🟡+🔴. QA-Pass.
## Backend-Gaps automatisierung (Luke)
🔴 Branch-/Merge-Step im Engine-Modell · `http_request`-ActionExecutor · `webhook.received`-Trigger (Route) · Cron-Trigger (frei konfigurierbar) · notification.send echter RPC. (17 Trigger, 8 Actions, Conditions, Templates, Engine, Dry-Run existieren — KMU-Niveau weitgehend erreicht.)

---

# MODUL-AUDIT 18–24: BRANCHEN-MODULE (Post-Launch / Solar-Pilot)
> Basis: Welle-3-Tiefenanalyse (`.planning/ist-abgleich.md`). **Durchgängiges Muster: ALLE 7 auf Zustand-Mock-Store, fertige TanStack-Hooks + Backend ungenutzt.** Umstellung je ~1–2 Tage FE. Cross-cutting: S3-Foto-Upload, Signatur-Persistenz, Einkauf↔Inventar-Sync, Mobile/PWA+Offline (für Solar-Außendienst).

## fuhrpark (Markt: Vimcar/Mobexo/Fleetster)
Ist: FE Mock; Fahrzeugakte/Wartung-TÜV/Schaden ◐ (BE da). **Führerscheinkontrolle + Fahrzeugbuchung-Pool fehlen ganz** (FE+BE). GPS/Fahrtenbuch/Tankkarten ◐ (BE-Modell fehlt teils). Foto-Upload Mock.
Plan: P1 Mock→API (Akte/Wartung/Schaden) 🟡 · P2 Führerscheinkontrolle + Pool-Buchung + Fahrtenbuch 🔴+🟡 · P3 GPS/Telematik + Tankkarten 🔴.
BE-Gaps: 🔴 LicenseCheck-Modell · VehicleBooking + Conflict-Check · LogbookEntry (finanzamtkonform) + PDF · FuelRecord · GPS-Webhook · Foto-Upload.

## inventar (Markt: weclapp/Myfactory/Zoho Inventory)
Ist: FE Mock (nur toast); Bestands-Alarm ✅; Stammdaten/Bestandsführung ◐. **Chargen/Seriennummern + Inventur + Kommissionierung fehlen im BE.** Wareneingang↔Einkauf-Sync fehlt. Barcode = Texteingabe (kein Kamera-Scan).
Plan: P1 Mock→API (Stammdaten/Bewegung/Warnung) 🟡 · P2 Inventur + Chargen/Serien 🔴+🟡 · P3 Kommissionierung + echter Barcode-Scan + Einkauf-Sync 🔴+🟡.
BE-Gaps: 🔴 batch/serial im Item-Modell · InventurSession-Modell · Kommissionierung · Einkauf→Inventar-RecordMovement.

## vermietung (Markt: Rentman/Booqable/easyJob)
Ist: FE Mock-Store; Objekt/Buchung/Zustandsprotokoll/Übergabe ◐ (BE da). Signatur (SignatureCanvas) nicht ans BE. **Online-Buchungsportal fehlt.** Tarife nur daily_rate (keine Staffeln/Wochensätze).
Plan: P1 Mock→API 🟡 · P2 Checklist-Format + Signatur-Persistenz + Tarif-Erweiterung 🔴+🟡 · P3 Online-Buchungsportal 🔴.
BE-Gaps: 🔴 strukturiertes Checklist-Format · signature_url im Inspection-Modell · Online-Buchungs-Endpoint · Tarif-Staffeln.

## einkauf (Markt: weclapp/Myfactory/SOG ERP)
Ist: FE Mock; Lieferanten/Bestellungen ◐ (BE da). **Auto-Bestellvorschläge fehlen.** Wareneingang bucht NICHT in Inventar (Sprint-3-Item). SupplierRating/Rahmenverträge/Katalog = FE-Tabs ohne BE. 2-stufiger Freigabe-Workflow fehlt. Status-Mismatch FE↔BE.
Plan: P1 Mock→API 🟡 · P2 Wareneingang→Inventar-Sync + 2-stufige Freigabe 🔴+🟡 · P3 Auto-Bestellvorschläge + Rahmenverträge + Katalog 🔴.
BE-Gaps: 🔴 SupplierRating · FrameworkContract + Katalog · 2-stufiger Approval (approved_by) · Auto-Bestellvorschläge · Einkauf↔Inventar-Bridge.

## produktion (Brücke — MRP-Tiefe bewusst begrenzt; Markt: weclapp/abas/TaxMetall)
Ist: FE Mock; Produktionsaufträge ◐. **BOM fehlt im BE** (FE-Tab da). Maschinenbelegung ◐ (nur String-ID, kein Maschinenregister). Kalkulation fehlt. progress/work_steps/scrap-Felder fehlen BE. **MRP = Fake-Hash.** QualityCheck-UI ohne BE.
Plan: P1 Mock→API (Aufträge) 🟡 · P2 BOM + work_steps/progress + Maschinenregister 🔴+🟡 · P3 MRP (Inventar-Abgleich) + Kalkulation + QualityCheck 🔴.
BE-Gaps: 🔴 BOM-Modell+CRUD · work_steps/progress/scrap · Maschinen-Stammdaten · MRP-Inventar-Abgleich · QualityCheck · Kalkulation.

## schichten (Solar-Pilot; Markt: Papershift/shyftplan/Shiftbase)
Ist: FE Mock/inline-Konstanten; Dienstplan-Erstellung ◐ (BE da). **Auto-Planer fehlt** (ApplyTemplate ≠ Planer). ArbZG doppelt (FE lokal + BE-Endpoint ungenutzt). **Verfügbarkeiten/Qualifikationen + Schichttausch fehlen im BE.** Mobile-Ansicht fehlt. Minderjährigen-Regeln (JArbSchG) fehlen.
Plan: P1 Mock→API (Dienstplan) + ArbZG aus BE 🟡 · P2 Schichttausch + Verfügbarkeiten/Qualifikationen 🔴+🟡 · P3 Auto-Planer + JArbSchG + Mobile 🔴+🟡.
BE-Gaps: 🔴 shift_swap_requests · Availability + Qualifikations-Tabelle · regelbasierter Auto-Planer · is_minor + JArbSchG.

## rapporte (Solar-Pilot; Markt: HERO/mfr/Craftboxx)
Ist: FE Mock; Approval-Workflow ✅ (modelliert). Mobile-Erfassung/Foto/GPS/Material ◐ (BE da, FE Mock/kein geolocation). Offline-Queue existiert, client nutzt ihn nicht. **Signatur-Persistenz fehlt BE.** **Aufmaß-Tab komplett FE ohne BE.** Mobile-Zugang fehlt (Electron-Desktop).
Plan: P1 Mock→API + geolocation + Foto-Upload 🟡+🔴 · P2 Signatur-Persistenz + Offline-Anbindung 🔴+🟡 · P3 Aufmaß-BE + Mobile/PWA 🔴.
BE-Gaps: 🔴 Signatur-Persistenz · Aufmaß-Modell (Measurement) · weather-Feld · Foto-Upload (S3) · Mobile/PWA.

---

# MODUL-AUDIT 25–30: SYSTEM-MODULE
> Basis: Welle-2-Tiefenanalyse. Kein klassischer „Markt-Ersatz" — Infrastruktur. Markt-Soll = IAM/Workspace-Standards (MS365/Zoho/Frontegg).

## security ⭐ (am vollständigsten)
Ist: ✅ Session-Mgmt/Revocation, RBAC, Passwort-Policy, Login-Verlauf, **unveränderbare Audit-Logs (Hash-Chain) + Export**, DSGVO-Export (Art. 30), DSAR, Recht-auf-Vergessen, 2FA-TOTP-Wizard. Lücken: **„Passwort vergessen"-Flow fehlt** (pre-launch wichtig) · WebAuthn/Passkeys · SSO (SAML/OIDC) · Federation. → Phase: Passwort-Reset 🔴+🟡 (pre-launch), Rest später.

## admin
Ist: Benutzer+Einladungen ✅; Tenant-Verwaltung ◐ (RLS-BE da, kein UI); **Provisioning + Super-Admin-Level + Billing/License + Tenant-Monitoring fehlen** (Mock). → P1 Tenant-CRUD-UI · P2 Billing-Backend (🔴) · P3 Monitoring-API (🔴). BE-Gaps: 🔴 Provisioning-Endpoint, System-Level-Rolle, Billing/License-Service, Tenant-Monitoring-API.

## dashboard
Ist: ✅ konfigurierbare Widgets, Layout-Persistenz, Schnellzugriffe, Personalisierung; KPIs/Feed ◐ (aus Modul-Services, kein zentraler Aggregator). → meist fertig; SOLLTE: Widget-Picker-UI, Default-Layouts pro Rolle (BE da). 🟡

## notifications
Ist: ✅ In-App-Center, Pro-Nutzer-Präferenzen, DND/QuietHours/Mutes, Modul-Gruppierung; Multi-Channel ◐ (desktop_push da, **Mail/SMS im Gateway nicht exponiert**), Workflow-Builder ◐. → P1 Mail/SMS-Kanal exponieren 🔴. BE-Gaps: 🔴 E-Mail/SMS-Dispatch im Gateway.

## profil
Ist: Benachrichtigungs-Präferenzen ✅; persönl. Daten/Sprache/Theme ◐ (lokaler Store, **BE-Persistenz fehlt** → Multi-Device); Avatar-Upload ✗ (kein Endpoint); Presence ◐ (hardcoded). **Doppel mit settings/ProfileTab → konsolidieren.** → P1 an User-API + Avatar-Upload 🟡+🔴 · Doppel auflösen. BE-Gaps: 🔴 User-Preferences-Persistenz, Avatar-Upload.

## settings
Ist: Standard-Werte ✅; Integrationen ◐ (Wizards da); **Workspace-Branding nur localStorage** (kein BE); Modul-Aktivierung ◐ (Flag-Registry da, **kein Admin-Toggle-UI**); Sprache/Region client-only. SecurityTab enthält Mock-Sessions (Altlast). → P1 Branding-Persistenz (🔴) + Modul-Aktivierungs-Toggle-UI 🟡 · Mock-Sessions raus. BE-Gaps: 🔴 Branding-Persistenz-Endpoint.

**System-Cleanups (Welle-2):** Profil-Doppel (profil↔settings) · SecurityAdminPage↔SecurityAdminHubTab (zwei Einstiege) · settings-Mock-Sessions · calendar-Stub · modules/video-Route.

---

# MODUL-AUDIT 31: buchhaltung (Brücke-Modul, aktuell _DEPRECATED→finanzen)
## Ist-Befund (Welle-2)
Als `_DEPRECATED` markiert, Inhalte nach finanzen migriert, noch im Routing, FE Mock-Store. GoBD-Journal/Belegerfassung ◐ (BE JournalSummary/Belegbilder da, FE Mock); DATEV-Export ✅; Mahnwesen ✅ (in finanzen). **Automatische Kontierung fehlt** (kein Kontenplan SKR03/04); EÜR ◐ (Mock); **Steuerberater-Zugang fehlt**.
## Markt-Soll (Brücke! — KEIN DATEV-Vollersatz)
MUSS (Brücke-Scope): GoBD-Journal · DATEV-Export · Belegerfassung+Archiv · **Steuerberater-Zugang**. SOLLTE: EÜR-Auswertung. KANN (= Brücke-Grenze, an DATEV/Steuerberater übergeben): automatische Kontierung, USt-Voranmeldung/ELSTER, Jahresabschluss.
## Entscheidung & Plan
User 02.06: langfristig **eigenes Buchhaltungs-Modul**, JETZT nicht umbauen (bleibt in finanzen). → Wenn reaktiviert: P1 Journal/Belege an echte API 🟡 · P2 Steuerberater-Zugang (read-only Rolle) 🔴+🟡 · P3 EÜR-Endpoint 🔴.
## Backend-Gaps buchhaltung (Luke, später)
🔴 EÜR-Endpoint · Steuerberater-Rolle (read-only) · (optional) Kontierung/Kontenplan. (Journal-Summary, DATEV-Export, Belegbilder, GoBD existieren.)

---

# ✅ ALLE MODULE AUDITIERT (Stand 2026-06-02)
**31 Audit-Einheiten = alle 32 PDF-Module** (chat+kommunikation als 1, video=meetings, calendar=kalender als Altlast).
Reihenfolge der Audits: Kunden-Zentrale(CRM+Kontakte) · kalender · dialer · formulare · dokumente · vertraege · mails · helpdesk · finanzen · berichte · team · kommunikation · meetings · work · zeiterfassung · wiki · automatisierung · 7 Branchen · 6 System · buchhaltung.

## Durchgängige Muster (über alle Module)
1. **„BE+Hooks fertig, FE hängt an Mock/lokalem Store"** — bei weitem häufigster Fall (vertraege, helpdesk, wiki komplett; kalender/kontakte/finanzen/team teils; ALLE 7 Branchen). → Großteil ist **FE-Anbindung, kein Luke**.
2. **Echte Backend-Neubauten (🔴) konzentriert auf:** Online-Terminbuchung (kalender), öffentl. Formular-Submit, Dialer-Telefonie (Skript/DNC/Recording), E-Rechnung-XRechnung, externe Share-Links (dokumente), Lead-Inbox/Scoring (CRM), DATEV-HR-Lohn (team), Branchen-Spezial (BOM/MRP/Inventur/Schichttausch).
3. **Cross-cutting (einmal bauen → viele Module):** S3/Foto-Upload · Signatur-Persistenz · Mobile/PWA+Offline (Branchen) · „Mock-Store→Hook"-Umstellung · CRM-Datenbasis (kein Silo).
4. **⭐ Am vollständigsten:** security, meetings, work, automatisierung, dialer(Kern), finanzen(Kern). **🔴 Größte Lücken:** Branchen (alle Mock + BE-Lücken), wiki/helpdesk/vertraege (komplett Mock), Online-Terminbuchung.
5. **Cleanups:** calendar-Stub · modules/video-Route · Profil-Doppel · Security-Doppel · settings-Mock-Sessions · chat+kommunikation-Merge · buchhaltung-Status.

## Nächster Schritt (nach Lukes RLS-Welle)
Bau-Reihenfolge pro Modul: Phase-Plan oben → je Bau-Einheit visueller QA-Pass (Teil G). Backend-Gaps gehen gebündelt an Luke (siehe je Modul + `.planning/backend-gaps.md`).

---

## Persistenz / Memory-Status (erledigt 2026-06-02)
Memory-Pointer gesetzt: Architektur „Kontakte = Kunden-Zentrale inkl. CRM" · visueller QA-Pass als Standard · diese Datei als zentrale Audit/Plan-Quelle. Modul-Audits hier: CRM/Kunden-Zentrale, kalender, dialer, formulare, dokumente. **Noch offen (Audit):** mails, kommunikation(chat+inbox), meetings, work, zeiterfassung, wiki, team, helpdesk, berichte, finanzen, buchhaltung, vertraege, automatisierung, + Branchen (rapporte, schichten, fuhrpark, vermietung, inventar, einkauf, produktion), + System (admin, dashboard, profil, security, settings, notifications).
