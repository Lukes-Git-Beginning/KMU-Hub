# RESUME — nächster Einstieg (Stand 2026-07-15, Session #11)

> **★★★★★ SESSION #11 (2026-07-15) — main `10a32584` (gepusht, Auto-Deploy lief). NEUES TERMINAL: HIER STARTEN (erst `git pull`).**
>
> **Branchen-Block 5/7: schichten (`9cb06ab7`) + fuhrpark (`10a32584`) komplett auf Standard**, gleiches Rezept (Marktrecherche-Agent → bauen → Gates → ALL PASS + Bilder angesehen).
>
> **fuhrpark (`10a32584`, Recherche Vimcar/Fleetster/Avrios/Carano):**
> 1. **„Fahrt eintragen" echt** (war Coming-soon-Toast): `AddTripDialog` verdrahtet ungenutzte `useCreateTripLog` — finanzamts-Layout (Route, km-Stände mit Auto-Strecke, Kategorie, Zweck-Pflicht bei Geschäftsfahrt, Fahrer). Speichern-Roundtrip QA-verifiziert (4→5 Zeilen).
> 2. **Fahrzeug-Detail Slide-over→`DetailModal`** + **Zeilen-Detail-Modals** für Wartung/Tanken/Fahrtenbuch (`FuhrparkDetailModals.tsx`), Tabellenzeilen `role=button`+Tastatur, echte Kennzeichen via `plateFor`-Lookup (Adapter tragen nur vehicle_id!).
> 3. **Echte Exporte statt totem `getExportUrl`** (aus fuhrpark-client entfernt): Fahrtenbuch-**PDF** (Finanzamt-Layout, WinAnsi) + Fahrtenbuch-**CSV** + Fahrzeugliste-**CSV** (`fuhrpark-export.ts`).
> 4. **Settings-Panel** (`fuhrparkPrefs`: Standard-Tab + **Standard-Fahrtenkategorie seedet TripDialog** + Reifen-Banner-Toggle · `fuhrparkTenant`: **reminderLeadDays speist getDateStatus/KPI** [war 30 hardcoded], **currency vereinheitlicht CHF/EUR-Mix** [TCO zeigte EUR, Tabellen CHF!], defaultFuelType [war 'diesel' hardcoded ×2], **privateTripsEnabled gatet Privat-Option**) + Registry + Hydrator ×2.
> 5. **3 Mock-Bugs via Screenshot-QA gefunden+gefixt:** (a) **GET /services-Handler fehlte komplett** → Wartung-Tab war im Demo IMMER leer (nur per-vehicle + /upcoming existierten), (b) Services-Seed nutzte `scheduled_date`+deutschen Freitext statt Wire-Shape `scheduled_at`+Enums → Datum „—", alles als „Service" gemappt; Seed gefixt + repair/tire_change-Einträge ergänzt, (c) Trip-Create berechnete `km` nicht (macht das echte BE) → frische Zeile zeigte „NaN km" überall. Dazu: Raw-Key `fuhrpark.vehicleType.car` im Fahrzeug-Modal (fehlendes t()), Platzhalter-Datum 2099→„—", fuhrpark-client re-exportiert jetzt seine Wire-Typen (tsc-Vorbestand).
> QA `scripts/qa-fuhrpark-tiefe.mjs` (10 Steps ALL PASS), tsconfig `fuhrparkcheck`, i18n ×4 (tote Keys costChf/chfPerLiter/tripComingSoon entfernt).
> **★ Paritäts-Kandidaten aus fuhrpark-Recherche (NICHT gebaut):** OBD2/GPS-Dongle + Auto-Fahrterfassung, Live-GPS-Karte, Führerscheinkontrolle (§21 StVG OCR), UVV-Unterweisung, Schadens-Workflow mehrstufig, Kraftstoffkarten-Import (UTA/DKV), DATEV-Buchungsstapel, ELSTER-Export, Leasingrückgabe-Workflow, Pool-Buchung, CO₂/ESG, Arbeitsweg als 3. Fahrtkategorie (API hat nur is_private bool), Manipulationsschutz Fahrtenbuch (Einträge nicht editier-/löschbar + Änderungsprotokoll — DER Finanzamts-Blocker). ⚠ fuhrpark-Seeds sind noch statisch 2024-datiert (kein daysFromNow-Muster wie schichten) — bei Gelegenheit modernisieren.
>
> **schichten (`9cb06ab7`, Recherche Planday/Deputy/Papershift/Shiftbase):**
> 1. **Grid-Zellen-Klick (belegt) → `ShiftDetailModal`** (war Info-Toast): Meta-Grid (MA/Datum/Zeit/Pause/Netto), Status-Badge draft/published, Zuschlag, ArbZG-Hinweise des MA, **Zuweisung entfernen** + **Tausch-Formular inline** (verdrahtet die ungenutzte `useCreateSwapRequest`!). Zellen `role=button`+Tastatur.
> 2. **„Vorlage bearbeiten" echt** (`useUpdateTemplate` lag ungenutzt bereit, Seeding-beim-Öffnen statt stale state) + **„Auf Woche anwenden"-Dialog** (aktuelle/nächste KW, verdrahtet ungenutzte `useApplyTemplate`).
> 3. **PDF-Stub → echter Dienstplan-PDF** (`schichten-export.ts`, WinAnsi-Muster aus rapporte: Kopf KW/Zeitraum/Summen, pro MA die Wochen-Schichten) + **neuer Wochen-CSV** (eine Zeile pro Zuweisung, BOM für Excel).
> 4. **Settings-Panel** (`schichtenPrefs`: Standard-Tab wirksam + Zuschlag-Badges-Toggle · `schichtenTenant`: **swapEnabled gatet Tausch-Button im Modal**, **maxWeeklyHours speist computeViolations/ArbZG**, **defaultBreakMinutes speist Template-Dialog + Adapter**) + Registry + Hydrator ×2. ⚠ `stores/schichten.ts` = Daten-Store (wie vermietung) → Settings-Stores heißen schichtenPrefs/schichtenTenant.
> 5. **Alt-Bug gefunden+gefixt (via Screenshot-QA!):** Klick auf belegte Zelle feuerte Drag-Selbst-Drop → sinnloser unassign→assign-API-Roundtrip + „Schicht verschoben"-Toast bei jedem Klick. Guard in `handleDrop` (Quelle==Ziel → no-op).
> QA `scripts/qa-schichten-tiefe.mjs` (10 Steps ALL PASS), tsconfig `schichtencheck`, SortMenu Name/Rolle/Wochenstunden, i18n ×4 (ICU-Plural arbzgHinweise/applyToastHint).
> **★ Paritäts-Kandidaten aus schichten-Recherche (NICHT gebaut):** Auto-Scheduling (KI), MA-Verfügbarkeits-Selbstpflege echt (Tab ist noch localStorage-Mock), MA-zu-MA-Tausch-Marktplatz (3-Step + Notifications), Skills-/Qualifikations-Matrix, Rotations-Templates, Wochen-als-Vorlage-Bibliothek, Offene-Schichten-Bewerbungsflow, Schichttyp-Farbe/Pause im Backend-Modell (Adapter-Defaults, color/breakMinutes fehlen im BE).
> **★ VERBLEIBEN: Branchen ×2** — **einkauf** (mittel; tote toast.info Bestellung/Lieferant bearbeiten/deaktivieren/Warenkorb/Neuer Abruf) → **produktion** (groß, Statuswechsel-Endpoint ERST greppen [5/5 Modulen lag er ungenutzt bereit], sonst Luke). Rezept + Stand: `.planning/branchen-block/README.md`.
>
> ---
> _(Historie #10 folgt)_

# RESUME — Historie (Stand 2026-07-15, Session #10)

> **★★★★★ SESSION #10 (2026-07-15) — main `aafe636a` (alles gepusht, Auto-Deploy lief).**
>
> **Branchen-Block gestartet — 2 von 7 Modulen komplett auf Standard (je Marktrecherche via Web-Agent + Screenshot-QA ALL PASS + Bilder angesehen + 1 Commit + Push):**
> 1. **inventar PILOT** (`1d1fac6c`) — (a) Artikel-Detail `DetailPanel`→`shared/DetailModal` (eigene `ItemDetailModal.tsx`, Zeilen role=button+Tastatur), (b) **Lagerort-Karten klickbar** → `LocationDetailModal` (Kennzahlen + klickbare Artikel-Liste + Bestandsliste-CSV) mit **onBack-Kette** ins Artikel-Modal, (c) **„Neue Inventur" echt**: Hook/Client/stateful-MSW existierten ALLE schon (Button war nie verdrahtet!) → `NewInventurDialog` (Stichtag/Lagerort-Scope=Teilinventur/nur-mit-Bestand, myfactory-Muster) **+ fehlende Ist-Zählung nachgebaut**: editierbare Ist-Inputs, Lifecycle open→counting→review (Zoho-Muster), Zählliste-CSV je Session, (d) Settings-Panel (`inventarPrefs`: Standard-Tab/Dichte/Warnungen wirksam · `inventarTenant`: Standard-Einheit/Mindestbestand-Default/Barcode-Format/**Negativbestand-Sperre** — greift im BewegungDialog), (e) SortMenu + Artikel/Bewegungen-CSV; Bewegungen-Tab hat jetzt **eigenen Artikel-Selector** (hing vorher am offenen Slide-over = im Modal-Paradigma kaputt), (f) ArtikelDialog stale-state via Remount-key gefixt, Nachbestellen/Zur-Bestellung → navigate('/einkauf'). QA `scripts/qa-inventar-tiefe.mjs`, tsconfig `inventarcheck`.
> 2. **vermietung** (`aafe636a`) — (a) Objekt-Detail → `ObjectDetailModal` + **neue `RentalDetailModal`**: Meta/Preis+Kaution (Kaution-erhalten via updateRental)/Notizen/**Zustandsprotokoll-Liste** (inspections-API) + **Lifecycle-Aktionen „Ausgeben"/„Zurücknehmen"** (`useStartRental`/`useEndRental` existierten ungenutzt!) + **Überfällig-Badge** (Booqable late-order), (b) Reservierungs-Zeilen + **belegte Kalender-Slots** öffnen das Rental-Modal (statt Info-Toast), Objekt-Modal→Rental mit Back-Kette, (c) **toten Konfliktcheck aktiviert** (`void hasConflict` in ReservationDialog!) inkl. tenant-`bufferDays` (Booqable-Puffer), Save-Block bei Konflikt, (d) Settings-Panel (`vermietungViewPrefs`: Standard-Tab/KPI-Leiste · `vermietungTenant`: Standardwährung/Vorbereitungszeit/Kautions-Pflicht — alle wirksam), (e) SortMenu + Objekt/Reservierungs-CSV, Dialog-Remount-keys, kaputtes totes `getExportUrl` in vermietung-client entfernt + 2 Baseline-Casts gefixt. ⚠ `stores/vermietungPrefs.ts` = DATEN-Store (objectPrefs/rentalPrefs-Mock-Zusatzfelder), NICHT der Settings-Store — deshalb heißt der neue `vermietungViewPrefs`. QA `scripts/qa-vermietung-tiefe.mjs`, tsconfig `vermietungcheck`.
>
> 3. **rapporte** (`23c39644`, gleiche Session) — (a) Report-Detail → `ReportDetailModal` (Karten role=button), (b) **PDF-Export-Stub → ECHTES PDF**: dependency-freier Generator nach `mail-export.ts`-`makeDemoPdf`-Muster, erweitert um **WinAnsiEncoding + Latin-1-Bytes** (Umlaute korrekt!) — Markt-Layout Kopf/Zeiten/Arbeiter/Tätigkeiten/Material/Unterschrift (`rapporte-export.ts`), (c) Sammel-CSV + SortMenu (Datum/Projekt/Autor/Status), (d) Settings-Panel (`rapportePrefs`: Standard-Zeitraum + **Standard-Arbeitszeiten** seeden den NewReportDialog [HERO/mfr-Muster] · `rapporteTenant`: **Unterschrift-Pflicht vor Einreichen** [im Approval-Abschnitt wirksam: Button disabled + Hinweis] + Währung), (e) Materialkosten-Stat via formatCurrency(tenant) statt CHF-Hardcode. **Nebenbei-Bugs gefixt:** `formatDateShort` doppeltes `T00:00:00` (Aufmaß-Karten zeigten „Invalid Date"!), totes `getExportPDFUrl` (fehlendes API_BASE_URL, wie vermietung), SignatureCanvas bekam nie existentes `onCancel`-Prop (Abbrechen war funktionslos → echter Cancel-Button im Modal), NewReportDialog-Remount-key. Workers-Sektion bei 0 versteckt (API liefert keine worker-Rows — Mock-Lücke). QA `scripts/qa-rapporte-tiefe.mjs`, tsconfig `rapportecheck`.
>
> **★ MARKTRECHERCHE-ABLAGE:** je Modul ein Web-Research-Agent VOR dem Bauen (inventar: weclapp/Zoho/myfactory · vermietung: Rentman/Booqable/easyJob · rapporte: HERO/mfr/Craftboxx/plancraft). Ergebnisse flossen direkt in Feature-Auswahl ein. **Paritäts-Kandidaten aus rapporte-Recherche (NICHT gebaut, für Modul-Paritäts-Phase):** Direkt-zu-Rechnung nach Freigabe (plancraft 2025-Differenzierer), Nummernkreise, drei Positionstabellen (Stunden/Material/Leistungen), GPS+Zeitstempel auf Fotos, Kunden-PDF-Sichtbarkeitsregeln, Abschluss-Qualifikation beim Einreichen. Muster beibehalten für die restlichen 4.
> **★ GATES wie Pilot-Spec:** scoped tsc grün für ALLE geänderten Dateien (Baseline-Fehler nur in unberührten Dateien) · eslint --quiet grün · i18n ×4 (ICU-Plural in en/fr/it wo Formen differieren) · Screenshot-QA ALL PASS + Bilder angesehen · PUSH-MODE pro Modul.
> **★ VERBLEIBEN: Branchen ×4** — **schichten/fuhrpark/einkauf** (mittel; Audit-Fehlstellen in `.planning/branchen-block/README.md`-Tabelle; ⚠ fuhrpark-client hat vermutlich dasselbe tote `getExportUrl`) → **produktion** (groß, Statuswechsel-Endpoint mit Luke klären). Muster: die 3 fertigen Module als Referenz (inventar=Pilot). **Frisches Terminal, erst `git pull`.**
> **★ WEITER OFFEN:** 3 Bexio-Review-Punkte (#8, `.planning/bexio-review-paket.md`) · Onboarding/Info-Center O-0 (NACH Branchen).
>
> ---
> _(Historie #9 folgt)_

# RESUME — Historie (Stand 2026-07-15, Session #9)

> **★★★★★ SESSION #9 (2026-07-15) — main `ec95077b` (alles gepusht, Auto-Deploy lief). NEUES TERMINAL: HIER STARTEN (erst `git pull`).**
>
> **Sequenz Schritt 3 (video + notifications Demo-Tiefe) DURCH — 2 Batches, beide gepusht+deployed, je Screenshot-QA ALL PASS + Bilder angesehen:**
> 1. **video Demo-Tiefe** (`76e253bf`) — (a) **`CallHistoryDetailModal`** (`modules/video/CallHistoryDetailModal.tsx`, shared/DetailModal): ganze History-Zeile klickbar (role=button, innere Call-Buttons stopPropagation), Meta-Grid + Teilnehmer-Chips + Notiz + Aufzeichnung + **echter Protokoll-Download** (Blob .txt) + Rückruf. `CallHistoryEntry`-Typ um company/phone/topic/notes/hasRecording/recordingDuration/participants erweitert, Mocks ch1–ch8 angereichert. (b) **Echte Kamera-Vorschau** (`VideoDeviceSettings.tsx` `CameraPreview`): getUserMedia + `<video>`, graceful Placeholder-Fallback (Start-Button) bei fehlender Kamera → QA mit `--use-fake-device-for-media-stream` verifiziert (aktiver Stream sichtbar). (c) **video settings-komplett**: `VideoSettingsPanel` (ModuleSettingsShell, personal Geräte via neuem `videoPrefs`-Store + tenant Recording-Policy/Standard-Raum via `videoTenant`-Store, beide backend-persist + im zentralen Hydrator), Registry-Eintrag `id:'video' navMatch:['/video']`, **moduleId durchgängig `'meetings'`** (kanonisch/leadable, kein SettingsModuleId-Typ-Update nötig). Geräte-Controls (`VideoDevicePrefsControls`) shared zwischen in-Page-Tab (mit Preview) + Overlay-Panel. (d) Tote Buttons: ActiveCall-„Details" + active-Tab-Empty-State-Buttons → `navigate('/meetings')`/`openNewMeeting`. QA `scripts/qa-video-tiefe.mjs`, tsconfig `videocheck` (+videoPrefs/videoTenant).
> 2. **notifications Demo-Tiefe** (`ec95077b`) — (a) **priority-Mismatch-Fix**: `stores/notifications.ts` `NotificationPriority` +`'urgent'` (fehlte, obwohl API/MSW/OpenAPI es kannten) + überfällige-Rechnung-Seed auf `urgent`/unread. (b) **Priority-Filter-Chips** im `NotificationCenter` (2. Chip-Reihe unter Modul-Chips, per-Prio-Counts + Farbdots, nur wenn >1 Prio da). (c) **Snooze** wieder da: `snoozeApi` (notification-client) + `useSnoozeNotification` (hook) + **MSW `POST /:id/snooze`** (setzt `snoozed_until`, `isSnoozed`-Filter in list+unread-count) + DropdownMenu im Detail-Modal (30 Min/1 Std/morgen früh) + Toast. (d) Cleanup: Center-EmptyState → `shared/EmptyState`, Bell-Dropdown Prioritäts-Links-Rand (urgent/high), **Zombie `components/dashboard/NotificationsFeed.tsx` gelöscht** (hardcoded, nur in design-reference gerendert, index.ts-Export raus). QA `scripts/qa-notif-tiefe.mjs`, tsconfig `notifcheck` (+stores/notifications).
>
> **★ GATES:** scoped tsc (videocheck/notifcheck) grün für ALLE geänderten Dateien; verbleibende Fehler sind bekannte Baseline in unberührten Dateien (LiveKit `BackgroundSelector`, `config/roles.ts` avatarUrl, `useProjects`, `finance-client` DATEV-cast, Calendar-Widgets). eslint grün. i18n ×4 (`{var}`, keine `{{}}`). PUSH-MODE: pro verifiziertem Batch auf main → Auto-Deploy.
> **★ OFFEN aus #8 (NICHT angefasst) — 3 Bexio-Review-Punkte für Darien:** (a) doppelter Info-Banner bei Bexio-Rechnung, (b) PDF-Download bei Bexio-Rechnungen sichtbar — gewollt?, (c) Bexio-Auffindbarkeit für Nicht-Admins. Voll: `.planning/bexio-review-paket.md` #8-Block.
> **★ AUCH #9 — Modul-Fertigstellungs-Push gestartet** (Darien: „alle Module auf Demo-Tiefe fertig, damit Luke Backend bündeln kann + Onboarding/Tutorials alle Funktionen haben"):
> - **Verifizierter Demo-Tiefe-Stand** (2 Explore-Audits gegen echten Code — MASTER-PLAN war an Stellen veraltet): **~20 Module review-reif** inkl. video/notifications/dialer/formulare/automatisierung. Audit-Details im Chat.
> - **Nachbesserungs-Batch** (`28018a48`): **team** SortMenu (Name/Abteilung/Status, `shared/SortMenu`) + `shared/EmptyState` für leeren Schulungs-Katalog + Teilnahmen-Tabelle · **admin** `IntegrationsAdminHubTab` Platzhalter-Div → `shared/EmptyState` (+ i18n statt hardcoded DE). QA `qa-nb-team.mjs` ALL PASS. → **alle Nicht-Branchen-Module jetzt review-reif.**
> - **VERBLEIBT: Branchen ×7** (rapporte/schichten/fuhrpark/vermietung/inventar/einkauf/produktion) — Darien-Entscheid: **VOLL auf Standard**. Audit-Befund (alle 7 TEILWEISE, **produktion NUR GRUNDGERÜST**): durchgängig **`DetailPanel`(Slide-over)→`shared/DetailModal`**, **kein Settings-Panel registriert** (alle 7, kein Eintrag in `module-settings-registry.tsx`), tote Buttons/Toast-Stubs (PDF-Exports rapporte/schichten, Edit-Dialoge einkauf, **produktion-Statuswechsel**, fuhrpark-AddTripLog, inventar-„Neue Inventur"), kein `SortMenu`, keine echten Exports. Aufwand: **vermietung/inventar/rapporte klein · schichten/fuhrpark/einkauf mittel · produktion groß**.
> - **NÄCHSTER SCHRITT — Branchen-Block:** Empfehlung 1 Referenz-Modul (klein, z.B. vermietung/inventar) komplett auf Standard → Muster etablieren (DetailPanel→DetailModal [API-kompatibel, mechanisch] + Settings-Panel personal/tenant + tote Buttons + SortMenu + echte Exports) → dann die anderen 6 (2-Terminal, disjunkte Module). **Frisches Terminal empfohlen** (großer Block).
> **★ WEITER OFFEN:** 3 Bexio-Review-Punkte (#8, `.planning/bexio-review-paket.md`) · Onboarding/Info-Center O-0 (§1.2 MASTER-PLAN — NACH Branchen, braucht alle Module fertig).
>
> ---
> _(Historie #8 folgt)_

# RESUME — Historie (Stand 2026-07-15, Session #8)

> **★★★★★ SESSION #8 (2026-07-15) — main `4ff3a812` (alles gepusht, CI grün, Auto-Deploy lief).**
>
> **Diese Session #8 — Darien-Sequenz Schritt 1 (Bexio) + 2 (security) durch:**
> 1. **Bexio verdrahtet + reviewbar** (`935a92ff`) — **Kern-Befund:** der echte Invoice-Pull-Wizard + Sync-Dashboard (`modules/settings/integrations/BexioSetupWizard`+`BexioSyncDashboard`) waren **toter Code** (nur `IntegrationsPage`, nirgends geroutet); beide erreichbaren Einstiege rendern den alten `BexioConfigPanel` (localStorage-Mock, kein Invoice-Pull). → Wizard/Dashboard in `IntegrationSettingsTab` **+** `FinanzIntegrationenTab` verdrahtet (Karte: disconnected→Wizard, connected→Dashboard), alten `BexioConfigPanel` gelöscht. **2 Demo-Mocks:** Read-only-Rechnung `2026-0042` (`source:'bexio'`, mocks/data/invoices.ts) + stateful Bexio-Sync-Endpoints (mocks/handlers/settings.ts). ⚠ **`Einstellungen→Integrationen` gibt's nicht** (adminOnly immer versteckt, SettingsPage:110) — realer Weg = **Modul-Einstellungen-Overlay → Buchhaltung/cosmi → Integrationen**.
> 2. **Bexio Schritt-3-Layout-Bug** (`985c24f7`, **Darien-Fund**) — Feld-Zuordnung sprengte den Screen: (a) `grid-cols-[1fr,auto,…]` mit **Kommas** = ungültiges CSS → Grid kollabiert → Dropdowns gestapelt; Fix Underscores. (b) Wizard-Dialog ohne `max-h` → Fix `max-h-[85vh]` + nur Step-Body scrollt (Nav sticky). Screenshot-verifiziert.
> 3. **Lexware identischer Bug** (`9a4d26db`) — `LexwareFieldMappingEditor` + `LexwareSetupWizard` hatten denselben Komma-Grid+max-h-Bug → mitgefixt. Kein weiterer Komma-Grid im Code.
> 4. **security FE verifiziert + Backend an Luke** (`4ff3a812`, **Darien-Entscheid**) — `security-client.ts` ist echt-schaltungs-bereit (Lukes Contract-Kampagne: Pfade/Envelopes/protojson-ISO → **kein FE-Adapter nötig**). Geroutete Oberfläche = **`SecurityAdminHubTab`** (`/admin/security`), NICHT die alte `SecurityAdminPage` (toter Code, App.tsx:237). Demo-QA `scripts/qa-security-demo.mjs` = ALL PASS (Audit-Log/Sessions visuell sauber). **Backend-Rest (security-demo.sql + Docker-Live-QA + destruktive Erasure/Vault) bleibt bei Luke** — `.planning/security-echtschaltung-luke.md` #8-Block.
>
> **★ OFFENE BEXIO-REVIEW-PUNKTE (Darien mitgeben):** (a) doppelter Info-Banner bei Bexio-Rechnung (Readonly + „bereits versendet"), (b) PDF-Download bei Bexio-Rechnungen sichtbar — gewollt?, (c) Bexio-Auffindbarkeit für Nicht-Admins (nur tenant-Section, für Nicht-Leads disabled). Voll in `.planning/bexio-review-paket.md` #8-Block.
> **★ NÄCHSTER SCHRITT (Sequenz Schritt 3): video + notifications** (Handoff: neues Terminal). **video:** CallHistory-Zeile klickbar → Detail-Modal (`shared/DetailModal`; `CallHistoryEntry`-Typ in stores/meetings.ts ist dünn → für Tiefe evtl. um Teilnehmer/Notiz/Aufzeichnung erweitern) + Kamera-Vorschau-Placeholder. **notifications:** Demo-Tiefe + priority-Mismatch-Fix. Danach Sequenz-Schritt 4: Welle 3 Onboarding/Info-Center (O-0 Konzept mit Darien).
>
> ---
> _(Detail-Historie #7 folgt)_

# RESUME — Historie (Stand 2026-07-15, Session-ENDE #7)

> **★★★★★ SESSION-ENDE #7 (2026-07-15) — main `46836248` (alles gepusht, CI grün, Auto-Deploy lief). NEUES TERMINAL: HIER STARTEN (erst `git pull`).**
>
> **Diese Session #7 gebaut (4 Batches, alle gepusht+deployed):**
> 1. **dialer Demo-Tiefe** (`d1db43f4`) — AgentDetailModal, Workspace-Idle-EmptyState, dialerPrefs→personal+tenant-Split, CampaignForm-Mode-Erklärung, ContactQueue-SortMenu + filter-aware EmptyState (fixt „Keine Kampagnen"-Fehlbeschriftung).
> 2. **X-4-Store-Splits KOMPLETT** (`d0d7dfa6`) — automatisierung/mail/formulare/berichte je personal+tenant, backend-persist, im zentralen Hydrator. Consumer außerhalb Panels mitgezogen (FormularePage, ScheduleReportModal). **X-4-Settings-Rollout jetzt vollständig** (~18 Stores swap-ready). ⚠ Modul-IDs brauchen Backend-Registry (Luke).
> 3. **video-Buttons + Radix-Crash** (`4771f9c3`) — VideoPage Header/CallHistory-Buttons (waren onClick-los) → öffnen `MeetingFormDialog` (Rückruf mit Kontakt-Prefill via neuem `presetTitle`). **Echten Crash gefunden+gefixt:** `MeetingFormDialog` hatte `<SelectItem value="">` (Radix-verboten) → Crash beim Öffnen, betraf **auch MeetingsPage**. Sentinel `"none"`.
> 4. **work/task-UX-Fixes** (`46836248`, Darien-Live-Review) — (A1) Complete-Kreis in MyTasks 16→24px + Hover-Häkchen · (A2) TaskDetailPage-Zurück-Pfeil `navigate(-1)`→Projekt-Board · (A3) KanbanBoard-Karten-Klick → volle Task-Seite statt Mini-Panel (Panel bleibt für projektlose Standalone-Tasks).
>
> **★ QA-DURCHBRUCH:** CosmiLaunch-Splash blockierte Demo-Screenshot-QA (Overlay). Fix: **`sessionStorage['cosmi:launch-played']='1'` im Playwright-`addInitScript`** überspringt ihn → alle Demo-QA-Skripte laufen jetzt sauber. Muster in `scripts/qa-video-actions.mjs`/`qa-tasks-fixes.mjs`. Scoped-tsc-Configs: dialercheck/x4split/videocheck/workqa (KanbanBoard ergänzt).
> **★ PUSH-MODE aktiv** (Darien): pro verifiziertem Batch auf main → Auto-Deploy. Vor Push: eslint geänderte Dateien + scoped tsc + Screenshot-QA (Bilder angesehen). Full-tsc-Baseline nicht grün (bekannt) — scoped reicht.
> **★ DARIEN-SEQUENZ (festgelegt 2026-07-15) — genau diese Reihenfolge:**
> 1. **Bexio-Review** — Darien reviewt die Bexio-Invoice-Pull-Dinger hands-on. **VOLLES PAKET: `.planning/bexio-review-paket.md`** (Checkliste + Datei-Refs). ⚠ **ERST 2 Demo-Mocks bauen** (Bexio-Rechnung im Seed + Sync-Status/Logs-Handler), sonst im Demo-Modus NICHT sichtbar — Details im Paket. Danach App starten + Darien geht die Checkliste durch, dann Fixes.
> 2. **security Echt-Schaltung** — FE S-1…S-5 gegen echtes Backend (~85 % real), `security-client.ts` Pfade/Wire-Shapes abgleichen, KEIN destruktiver GDPR-Test. Kein Luke nötig.
> 3. **NEUES TERMINAL: video + notifications parallel** — Darien öffnet ein Sub-Terminal, beide Module gleichzeitig (disjunkt). video: CallHistory-Detail-Modal + Kamera-Vorschau. notifications: Demo-Tiefe + priority-Mismatch-Fix.
> 4. **NEUES TERMINAL: Welle 3 Onboarding/Info-Center** (§1.2) — O-0 Konzept zuerst (mit Darien abstimmen) → O-1…O-6.
>
> **★ PROD-BLOCKER für Luke (P0, separat klären):** X-7 Feature-Flags in `.env.production` (sonst deployt Auto-Deploy helpdesk/wiki/berichte/formulare/vertraege/video unsichtbar) · Prod-Migrationsdrift (Prod 209, lokal 243 → 210–243 nachlaufen) · mails-IMAP · admin-Backend (Invite/RBAC/License/S3) · security-Spec-Lücke (31 Endpoints fehlen in openapi.yaml). **Weitere self-doable Reste** (nach der Sequenz): admin-Demo-Tiefe, Tiefe-Re-Checks T-1…T-4, Branchen×7-Demo-Tiefe.
>
> ---
> _(Detail-Historie #7 folgt)_

# RESUME — Historie (Stand 2026-07-14, Session-Start #7)

> **★★★★★ WIEDEREINSTIEG NACH PAUSE — main `adfa75ff` (gepullt, sauberer FF von +19). Darien war weg; in der Pause hat FAST NUR LUKE gebaut (23 von 25 Commits seit #6). Kein neuer eigener FE-Bau in der Pause.**
>
> **Was Luke seit #6 (05.07.) gemacht hat — drei Kampagnen, alle CD-deployt:**
> 1. **Bexio Invoice-Pull Welle 3b** (`1f8475a7`→`e19a68fb`, `eae34f42`) — Read-only-Mirror: externe Bexio-Rechnungen werden gespiegelt (Migr. **000243**, `UpsertImported`). **FE via Worktree-Subagent gebaut:** `bexio-types.ts` auf flache Wire-Shape, `bexio-client.ts` Pfad-Drift gefixt (`/oauth/authorize`, `/sync/trigger`), Invoice-Pull-Toggle im Wizard + Config-Persist, **Read-only-Badge/Banner auf `source='bexio'`-Rechnungen, alle mutierenden Aktionen ausgeblendet** (Edit/Send/RecordPayment/MarkPaid/Cancel/Storno). i18n ×4, Screenshot-QA gemacht. → **DARIEN-REVIEW-KANDIDAT** (Subagent-Bau, noch nicht von Darien angesehen).
> 2. **ProtoTimestamp/protojson-Kampagne** (~13 Commits, 3 Runden) — Gateway serialisierte Proto-Timestamps als `{seconds,nanos}` (brach FE-Datumsparsing gegen echtes BE). Jetzt via **protojson** über ~alle Module: rapporte/inventar/crm · auth/berichte/automation · fuhrpark/einkauf/produktion · security/formulare/settings · vermietung/schichten/booking/integration · hr map-envelope · biz/finance. plugin/lexware/datev-Protos echt regeneriert. **crm/chat/email waren KEIN Bug** (Timestamps schon `string`).
> 3. **FE/BE-Contract-Mismatch-Kampagne** (`c13586a3`→`39f6393c`→`e8bb19df`, 6 Baustellen/3 Wellen) — Schwester-Klasse: FE-Client las falsche Wire-Shape (nested vs flach, camelCase vs snake_case, falsche URL-Pfade), MSW spiegelte die falsche Erwartung. Bereinigt: **hr/Leave** (Envelope-Unwrap + POST-Body camelCase→snake) · **Integrations** (BE-Fix `HandleGetLinkStatus` ehrt `{platform}`) · **auth/2FA+Sessions+Audit** (`security-client.ts` Pfade/Bodies) · **automation** (echte Stats) · **produktion** (Envelope-Unwrap) · **formulare** (Drilldown auf 4 echte BE-Zähler gekappt, fiktive Analytics raus, 18 tote i18n-Keys). Referenz-Clients sauber: `helpdesk-client`/`booking-client`/`hr-client`.
>
> **★ WAS DAS FÜR DARIENS FE-TRACK BEDEUTET:** Die Wire-Shape-Mismatches waren der Haupt-Blocker der Echt-Schaltung (Welle 1). Luke hat sie modulweit **vorab gefixt** → hr/security/automation/produktion/formulare/integration-Clients sind jetzt auf echte BE-Realität ausgerichtet. Diese Module sind damit **deutlich näher an sauberer Echt-Schaltung** als der Plan (Stand 28.06.) sagt. ⚠ Aber: Luke konnte **keine Electron-Screenshot-QA** fahren (GUI nicht headless erfassbar) → visuelle Verifikation der bereinigten Module steht aus.
>
> **★ DIESE SESSION #7 GEBAUT (Welle 2, dialer Demo-Tiefe — gepusht, Auto-Deploy):** dialer als erstes Welle-2-Modul komplett auf review-reif. **D-A** `AgentDetailModal` (Supervisor-Zeile klickbar → Status/Calls/letzte-5-Anrufe, gefiltert aus `recent_calls`) · **D-B** Workspace-Idle shared `EmptyState` + differenzierter Leer-Fall (CTA „Zu den Kampagnen" / „Kampagne wählen") · **D-C** war schon `ModuleSettingsShell`+registriert (nur verifiziert) · **D-D** `dialerPrefs`→personal-only backend-persistiert + neuer `dialerTenant`-Store (tenant, role-gated), beide im zentralen Hydrator (X-4-Split #1 von 5 gemischten) · **D-E** CampaignForm Mode-Erklärung + ContactQueue `SortMenu` + **filter-aware EmptyState** (fixt Fehlbeschriftung „Keine Kampagnen"→Kontakt-Copy). Bonus: `dialer-normalize.ts` Baseline-Typfehler. Gates grün (scoped tsc/eslint/i18n×4/Screenshot-QA angesehen). Screenshots `desktop/.qa-screenshots/dialer-tiefe/`, QA `scripts/qa-dialer-tiefe.mjs`+`qa-dialer-settings-panel.mjs`. ⚠ **DialerSettings-Panel-Screenshot vom CosmiLaunch-Dev-Artefakt verdeckt** (Overlay-Navigation re-triggert LaunchOverlay im Dev-Server; Panel-Content per DOM-Assertion bestätigt).
>
> **★ AUCH SESSION #7 GEBAUT (X-4-Store-Splits KOMPLETT, `d0d7dfa6`, gepusht):** die 4 restlichen gemischten Prefs-Stores gesplittet (automatisierung/mail/formulare/berichte) → je personal-Store (user, backend-persist) + neuer `*Tenant`-Store (tenant, role-gated), alle 8 im zentralen Hydrator. Consumer außerhalb der Panels mit umgehängt (FormularePage 5 Tenant-Felder + `DEFAULT_CONSENT_TEXT`/`_PRIVACY_URL` → formulareTenant; ScheduleReportModal `allowedFormats` → berichteTenant). tsc/eslint grün, Smoke (`scripts/qa-x4-splits-smoke.mjs`): keine pageerrors, alle 4 Tenant-Sektionen rendern. **→ X-4-Settings-Rollout jetzt VOLLSTÄNDIG** (alle ~18 Stores backend-persist + swap-ready). ⚠ Modul-IDs (`automatisierung`/`mail`/`formulare`/`berichte`) brauchen Backend-settings-Registry-Einträge (Luke).
>
> **★ AUCH SESSION #7 GEBAUT (video-tote-Buttons + Radix-Crash-Fix, gepusht):** VideoPage Header „Neuer Anruf"/„Meeting starten" + CallHistory-Rückruf-Buttons (waren onClick-los) verdrahtet → öffnen den wiederverwendeten `MeetingFormDialog`; Rückruf seedet den Titel mit dem Kontaktnamen (neuer `presetTitle`-Prop). **DABEI echten Crash gefunden+gefixt:** `MeetingFormDialog` hatte `<SelectItem value="">` (Kontakt/Deal-„Keine Verknüpfung") → Radix-verboten → Dialog crashte in Error-Boundary („A `<Select.Item />` must have a value prop that is not an empty string"). Sentinel `"none"` statt `""` + Filter auf Items mit id. **Betraf auch MeetingsPage** (gleicher Dialog). Screenshot-QA ALL PASS (Dialog öffnet, Prefill „Anna Müller"). tsc/eslint grün. **★ QA-LEHRE:** CosmiLaunch-Splash überspringt man via `sessionStorage['cosmi:launch-played']='1'` im addInitScript (löst die Overlay-Blockade in ALLEN Demo-QA-Skripten — endlich).
>
> **★ OFFEN / NÄCHSTE UNIT:** (a) **Bexio-Invoice-Pull-FE reviewen** (Subagent-Bau) · (b) **admin-Lücken** (Integrations-Tab-Placeholder, License-Detail-Modal) · (c) **video-Rest** (CallHistory-Detail-Modal, Settings-Persist, Kamera-Vorschau-Placeholder) · (d) **Welle 3 Onboarding/Info-Center** (§1.2) · (e) **Echt-Schaltungs-Verifikation der Luke-bereinigten Module** (hr/security/automation/produktion/formulare visuell gegen echtes BE). **Luke-gebunden bleibt:** security-DSGVO-Echt-Schaltung, mails-IMAP, admin-Backend (Invite/RBAC/License/S3).
> **★ DOCKER-REALITÄT (aus #6, prüfen):** postgres = custom-Image (pgvector+pg_cron, Migr. jetzt **000243**) → muss gebaut werden. Bringup `--no-deps --no-build` (OOM!), Login `demo@local.test`/`Demo1234!`. Images sind nach Lukes Pull **stale** → betroffene Services neu bauen vor Echt-QA.
> **★ Git-Hygiene:** `deploy/docker/docker-compose.flags.yml` + `desktop/scripts/qa-dialer-callflow.mjs` bleiben untracked (lokal).
>
> ---
> _(Stand #6 folgt)_

# RESUME — Historie (Stand 2026-07-05, Session-Ende #6)

> **★★★★★ SESSION-ENDE 2026-07-05 #6 — main `b5e3ec55` (alles gepusht, CI+CD grün, Auto-Deploy lief). NEUES TERMINAL: HIER STARTEN (erst `git pull`).**
>
> **Diese Session (#6): Lukes 07-05-Quick-Win-Welle verifiziert + X-4-Settings-Rest fertig gebaut.**
> 1. **CRM Kontakt-CSV-Import echt-verifiziert — 2 mock-verdeckte Bugs gefunden+gefixt (`ea1748a5`):** (a) **Wire-Contract-Mismatch** — FE sendete Field-Mapping als JSON-Feld `field_mapping`, Gateway erwartet `map_<spalte>=<feld>`-Formularfelder → **jede** Zeile geskippt (`imported_count:0`). FE sendet jetzt `map_*`. (b) **Auto-Detection** `knownMappings` kannte `first_name`/`last_name` (Unterstrich) nicht → Export→Import-Round-Trip erkannte Namen nicht; ergänzt. Live end-to-end (Preview/Import/Export CSV+vCard/Visibility gegen echtes crm) verifiziert. **GAP→Luke:** company beim Import ignoriert + Export leer (Round-Trip).
> 2. **Video Incoming-Call/Decline (`44b23e77`) — Code-Review sauber, kein Bug.** Backend-Round-Trip komplett (`videoWSAdapter.NotifyCallDeclined`→EndCall+BroadcastCallEnded). Nicht live-2-Client getestet. **→Luke:** `caller_name`-Lookup im `call.incoming`-Broadcast (FE fällt auf ID zurück).
> 3. **Dunning-Mahnung E-Mail+PDF (`273f1b6b`) — live verifiziert** (create→send→PDF gegen biz/minio, SMTP graceful suppressed, Log bestätigt). **→Luke (Prod-Risiko):** Mail-Send ist **fatal** bei konfiguriertem Mailer + braucht Company-Settings → Tenant ohne Settings bekäme 500. Non-fatal machen erwägen.
> 4. **X-4 Settings-Rest FERTIG (`b5e3ec55`):** 6 Stores backend-persistiert nach crmPrefs-Muster + im zentralen `useHydrateModuleSettings` registriert. **user:** workPrefs, vertraegePrefs. **tenant** (read alle, write role-gated): financeTenant, wikiSettings, dashboardSettings, zeiterfassungSettings. Runtime-verifiziert (`scripts/qa-x4-rest-hydrator.mjs`: alle 6 hydratisieren Server-Werte nach localStorage-Default-Seed+Login). **→ X-4 self-doable KOMPLETT.** Offen X-4 = nur Welle-2-Reste: gemischte Store-Splits (dialer/automatisierung/berichte/mail/formulare-Prefs) + groß (payrollSettings/workSettings, Backend teils fehlt).
>
> **★ PUSH-MODE (Darien 07-05):** pro verifiziertem Modul auf main → **Auto-Deploy live**. Vor jedem Push CI-grün (eslint geänderte Dateien + scoped tsc + qa). backend-gaps.md 07-05-Block gepflegt.
> **★ DOCKER-REALITÄT:** postgres ist jetzt **custom-Image** (pgvector+pg_cron, `deploy/docker/postgres/Dockerfile`, Migr. 242) → muss **gebaut** werden (`--no-build` schlägt fehl mit „No such image"). Diese Session neu gebaut (waren stale nach Lukes Pull): postgres, crm, gateway, biz, migrate. Stack healthy: postgres/redis/auth/crm/gateway/minio/biz/work. Bringup: `--no-deps --no-build` (OOM!), Login `demo@local.test`/`Demo1234!`. PUT-Settings brauchen Idempotency-Key (setzt `authenticatedRequest` autom.).
> **★ NÄCHSTE UNIT (Vorschlag):** Welle 2 — **admin Demo-Tiefe** + **settings-Lücken (P2)** + **gemischte X-4 Store-Splits** + **Demo-Tiefe-Phasen** (notifications/formulare/dialer/video) · ODER **Welle 3 Onboarding/Info-Center** (reines FE, `§1.2`). Luke-gebunden bleibt: security-DSGVO-Echt-Schaltung, mails-IMAP, admin-Backend (Invite/RBAC/License/S3).
>
> ---
> _(Historie #5 folgt)_

# RESUME — Historie (Stand 2026-06-28, Session-Ende #5)

> **★★★★ SESSION-ENDE 2026-06-28 #5 — main `31330bb2` (alles gepusht, CI grün, Auto-Deploy lief). NEUES TERMINAL: HIER STARTEN (erst `git pull`).**
>
> **AUFTRAG NEUES TERMINAL: X-4-Settings-Rest fertigmachen** (Darien: „im neuen Terminal X-4-Rest, dann schauen wir weiter").
> **Rezept steht** (Referenz `stores/crmPrefs.ts` + `api/settings-persist.ts` + zentraler `hooks/useHydrateModuleSettings.ts`). Pro Store: persist behalten, `serverInitialized` + `initFromServer` (loadModuleSettings) + Write-Through (saveModuleSettings) in Settern, dann **in `useHydrateModuleSettings.ts` registrieren** (kein pro-Page-useEffect mehr — zentral in DeskEnvironment). Verify-Muster: `desktop/scripts/qa-x4-central-hydrator.mjs` (Server-Wert via API setzen mit Idempotency-Key → frischer Client hydratisiert).
> **X-4-Rest:** tenant-Settings (wikiSettings/dashboardSettings/zeiterfassungSettings/financeTenant — scope `'tenant'`, Schreiben nur Lead/Admin) · mittel (workPrefs/vertraegePrefs) · gemischt dialer/automatisierung/berichte/mail/formulare-Prefs = **Store-Split → Welle 2, NICHT jetzt**. Voller Plan: `.planning/welle1-finish-plan.md`.
>
> **Diese Session (#5) gebaut — Welle 1 self-doable praktisch durch (alles gepusht):**
> 1. **helpdesk echt-geschaltet** (`a1242d6d`) — Lukes tenant-Fix wirkt; 1 mock-Bug (KB/Routing-`undefined`-Crash → `unwrapList`); helpdesk-demo-Seed (6 Tickets, diverse UUIDs für Ticket-Nr).
> 2. **kommunikation-Inbox echt-geschaltet** (`9621ecc4`) — **3 mock-Bugs**: received_at `{seconds,nanos}`→ISO (ConversationList-Sort-Crash) · channel-Int→String · getMessage `{message}`-unwrap (Thread-Crash). inbox-demo-Seed (6 Messages). `inbox-client.ts` `normalizeMessage`.
> 3. **documents** (`50b8632a`) — K6 Share-List-URL `/shares`→`/shares/entity` (war 405) · K7 `normalizeShare` Enum-Int. READ regressionsfrei.
> 4. **zeiterfassung** (`faf4fe8c`) — live verifiziert (−16h15m), AbsenceCalendar-guard.
> 5. **security NUR verifiziert** — Backend echt (~25 Endpoints real), echt-schaltbar, NICHT geschaltet (Darien-Wunsch). Master-Plan „2/10" überholt.
> 6. **X-4: 8 personal-Prefs-Stores + zentraler Hydrator** (`07d31f3a`+`943ab109`) — crm/finance/team/dashboard/helpdesk/zeiterfassung/wiki/dokumente. Live verifiziert.
> 7. **DB-Migr. 227–234 nachgezogen** (hing auf 226 — migrate-Image war auch stale, neu gebaut).
>
> **★ DOCKER-REALITÄT (wichtig):** Die laufenden Images waren **vor Lukes Pull** (alter Code). Für Echt-Schaltung **Gateway + notification + auth + migrate neu bauen** nötig (`docker compose build <svc>` dann `up -d --no-deps --no-build <svc>`; Gateway mit `-f docker-compose.flags.yml`). Stack läuft (15 Services healthy). Bringup: `--no-deps --no-build` (OOM!), DB-User `kmuhub`/`kmuhub`, Login `demo@local.test`/`Demo1234!`. **PUT-Settings braucht Idempotency-Key-Header** (setzt `authenticatedRequest` automatisch; curl-Tests brauchen ihn manuell).
> **★ LUKE-TEXT rausgegeben** (Darien verschickt): Dank + Prod-Seed/Feature-Flags-Bitte + neue Backend-Gaps. Alles in `backend-gaps.md` (28.06.-Block): helpdesk-Namen/Kategorie/ticket_number · inbox-Thread-RPC+Canned · documents-naked-Shapes · mails-IMAP (Luke).
> **★ HETZNER-REVIEW-CAVEAT:** Code deployed ≠ auf Hetzner mit Daten sichtbar — Prod braucht Feature-Flags (`.env.production`) + Prod-Seed (Demo-Daten nur lokal). Luke-Schritt.
> **★ MASTER/BACKEND-PLAN aktualisiert** (§0+§6 / 28.06.-Block). ~18 Module echt-verkabelt, FE ~50–55 %.

---
_(Historie #4 folgt)_

>
> **⚠ NEUE REGEL (Luke): CI-grün beim Push → AUTO-DEPLOY auf Hetzner.** Jeder Push MUSS CI-grün sein. Desktop-CI (`ci-desktop.yml`) = `eslint src/` + `npx tsc --noEmit` (full, ~3,5min grün) + `vitest` + `npm run build`. Vor JEDEM Push lokal grün fahren (eslint auf geänderte Dateien reicht meist; full-tsc ist grün). [[feedback_hetzner_review_workflow]]
>
> **Diese Session gebaut (alles gepusht, CI-grün):**
> 1. **Welle 1 stark vorangetrieben — self-doable Modul-Echt-Schaltung praktisch DURCH.** Neu echt-verkabelt + live verifiziert: **documents** (READ, `5e6c14ef` — 5 Wire-Drifts FE-tolerant gefixt) · **calendar** (`a943937d`, Luke-verdrahtet, work-Service up) · **wiki** (`a6cf212b` — Crash-Bug `r.articles.map()` ohne Guard gefixt) · **automatisierung/berichte/kommunikation-chat** (`85f2d24b`, Services gebaut+verifiziert, keine FE-Fixes nötig). **~16 Module echt-verkabelt.**
> 2. **settings X-4 Referenz-Muster** (`cc5d930d`) — dokumente-Tenant-Settings store-direct → Backend (`/settings/dokumente/tenant`, Migr.138), `initFromServer`+Write-Through, end-to-end verifiziert (localStorage gewiped → hydratet aus Backend). **Rollout auf ~12 weitere Stores = self-doable Rest.**
> 3. **6 mock-verdeckte Bugs gefunden+gefixt** (FE-tolerant) + **2 Deploy-Blocker für Luke** (Feature-Flags X-7 + helpdesk-tenant).
>
> **★ FEATURE-FLAGS (X-7, deploy-kritisch):** helpdesk/wiki/berichte/formulare/vertraege/video/Branchen hängen im Gateway an `COSMI_MODULE_*_ENABLED` (default OFF). Lokal aktiviert via **`deploy/docker/docker-compose.flags.yml`** (untracked Override — beim Gateway-Start `-f docker-compose.yml -f docker-compose.flags.yml`). **Prod muss die Flags in `.env.production` setzen, sonst deployt der Auto-Deploy ohne diese Module.**
>
> **★ admin (Sub) FERTIG + GEMERGT** (`79020623`, 25.06.) — A-1…A-5 FE-mock-first: Benutzerverwaltung (Invite/Detail-Modal), RBAC-Matrix, Lizenz/Modul-Aktivierung, Branding-Tab, Settings-Overlay-Eintrag (~3346 Z., i18n ×4, QA `qa-admin.md`). Merge-Konflikt nur ITAdminHubTab (Branding-Dublette raus = Sub-Version genommen). **Full CI lokal grün** (eslint+tsc+`npm run build` 1m11s) vor Push. **Echt-Schaltung wartet auf Luke** (Auth-Invite/RBAC-Persist/License-Service/S3 — `backend-gaps.md` §Vorausschau). Offen: admin Demo-Tiefe-Schliff.
>
> **★ 2 LUKE-TEXTE rausgegeben** (Darien verschickt): (a) Welle-1-Blocker (helpdesk-tenant/security-DSGVO/mails-IMAP/inbox/documents-Wire/Feature-Flags) · (b) Vorausschau Welle 2/3 (admin-Stack/settings-OAuth/profil-S3/security + Onboarding=FE-only). Beide persistiert in `backend-gaps.md` (oben „🔭 Vorausschau" + „Echt-Schaltung-Befunde").
>
> **NÄCHSTE UNIT (Vorschlag, ~Welle 2+3 fast komplett self-doable):** Welle 2 ~80% ohne Luke (admin via Sub + **Demo-Tiefe-Phasen** notifications/formulare/dialer/video + Tiefe-Re-Checks kontakte/calendar/dokumente/zeiterfassung + **settings-Tabs P2** + **X-4-Rollout** ~12 Stores) · Welle 3 ~95–100% (Onboarding/Info-Center = reines FE). Luke-gebunden bleibt nur: settings-OAuth (P4) + security-DSGVO-Echt-Schaltung.
>
> **DOCKER-STACK läuft** (13 Services: postgres/redis/auth/crm/gateway/dialer/biz/minio/document/work/helpdesk/wiki + automation/berichte/chat). Gateway läuft mit Flags-Override. Falls weg: hochfahren + Gateway mit `-f docker-compose.flags.yml` recreaten + bei neuem Service Gateway-Restart (Service-Discovery). **Nur Main fasst Docker an (OOM).**
> **DEV-SERVER-QUIRK bleibt:** `electron-vite dev --mode localbackend` kam mehrmals flaky in MSW/Demo-Mode hoch (kein Login-Screen → QA-Timeout). Fix: sauber killen (`Get-NetTCPConnection -LocalPort 5173 | Stop-Process` + `Get-Process electron | Stop-Process`) + neu starten. **`--port`-Flag geht NICHT** (electron-vite CACError) — Sub nutzt 5173 oder env-gateten Port.
> **Git-Hygiene offen:** `deploy/docker/docker-compose.flags.yml` (untracked, LOKAL behalten — nicht committen) · `desktop/scripts/qa-dialer-callflow.mjs` (untracked, vorbestehend).
> **Master-Plan synchronisiert** (§0 Gesamtstand + §6 Bau-Status-Tabelle auf 25.06., ~16 echt-verkabelt). QA-Skripte: `qa-mock-exit-dokumente.mjs`, `qa-mock-exit-modules.mjs` (Multi-Route), `qa-settings-dokumente-persist.mjs`.

---
_(Historie #3 folgt)_

# RESUME — Historie (Stand 2026-06-23, Session-Ende #2)

> **★★ SESSION-ENDE 2026-06-24 #3 — main `156ca17a`, alles gepusht. NEUES TERMINAL: HIER STARTEN (erst `git pull`).**
>
> **Diese Session gebaut (alle live/API-verifiziert + gepusht):**
> 1. **dialer-Supervisor echt-geschaltet** — FE-Normalizer für protojson-Null-Omission (`api/dialer-normalize.ts`, `48b5daf9`) + **Backend-Bug** recent-calls-Query las nicht-existente `cc.contact_name` → crm-Join (`9dfcf89e`). Live-Screenshots `desktop/.qa-screenshots/dialer-supervisor/`.
> 2. **dashboard-Layout echt verifiziert** — war schon über `apiClient`↔gateway-nativ verkabelt, kein Code nötig. Roundtrip GET→PUT→GET live ok.
> 3. **zeiterfassung/HR echt-geschaltet** — **Backend-Bug** `correction_reason` (nullable) in `string` gescannt → 500 bei JEDEM echten Eintrag → `COALESCE` in 3 SELECTs (`b7242926`). API-verifiziert (entries 500→3). HR liegt auf **biz**-Service. FE-Screenshot offen (Dev-Server-Quirk, s.u.).
> 4. **security/DSGVO (Sub) gemergt** — `43fecf37` (S-1…S-5, review-reif): 11 Seiten crashfrei, GDPR-Flows Art.15/17/20, ein Hub `/admin/security` (10 Tabs), i18n ×4. Merge konfliktfrei, Build grün. Bericht `.planning/parallel-batch/qa-security.md`. **Offen: Art.30 RoPA** = eigener Folge-Batch.
> → **3 mock-verdeckte Backend-Bugs** gefunden+gefixt (alle brachen echte Daten still, alle deploy-relevant für Luke).
>
> **★ NEUER WORKFLOW (Darien, 24.06.):** Darien reviewt jetzt **hands-on auf der Hetzner-Cosmi-exe** (app.zentria.tech), parallel während gebaut wird. → **ALLES auf main pushen.** Prüf-Items in **`.planning/hetzner-review-checklist.md`** (lebende Datei pflegen). **2 offene Fragen dort:** (a) wer/wie deployt auf Hetzner (Auto-Deploy `cd.yml` nicht scharf → Push ≠ live), (b) Demo- oder Live-Mode der Hetzner-exe. Backend-Echt-Schaltungen sind nur lokal sichtbar (brauchen Deploy + Prod-Seed) → die weiter lokal verifizieren.
>
> **Docker-Stack läuft noch** (postgres/redis/auth/crm/gateway/dialer/biz/minio healthy) — **nur Main fasst Docker an** (OOM!). Falls weg: hochfahren (Abschnitt unten), Seeds: `backend/seeds/demo/{demo-seed,dialer-demo,hr-worktime-demo}.sql`. biz braucht minio (sonst Crash-Loop „gobd archive").
> **DEV-SERVER-QUIRK:** `electron-vite dev --mode localbackend` kam zuletzt in **MSW/Demo-Mode** hoch (Auto-Login „Stefan Vogel" statt „Demo Local") trotz Build-Flag → flakig. Beim Dialer-Lauf lief dieselbe Instanz korrekt localbackend. Check: Login-Screen = localbackend ok; Auto-Login Stefan-Vogel = Demo-Mode (neu starten). Kill: PowerShell `Get-NetTCPConnection -LocalPort 5173 | Stop-Process`.
> **NÄCHSTE UNIT (Vorschlag):** Welle 2 — **admin** (Benutzer/RBAC/Lizenz) oder **settings-Persistenz** (X-4, BE Migr 138 da); oder neues Sub für 2. Modul. Reste zeiterfassung: FE-Screenshot + `useAbsenceCalendar`-null-guard (`hr-hooks.ts:508` `select: d=>d.entries` ohne `?? []`).
> **Master-Plan:** ~47–50 % FE-Phasen, 10 echt-verkabelt, 11 FE-mock-fertig — `MASTER-PLAN.md` §6.
> **Git-Hygiene offen:** `desktop/package-lock.json` (npm-install-Churn, uncommitted), `desktop/scripts/qa-dialer-callflow.mjs` (untracked, vorbestehend) — committen oder verwerfen.


> **★ UPDATE 2026-06-24 #2 — dialer SUPERVISOR echt-geschaltet (Welle 1), live verifiziert + gepusht (`48b5daf9`).**
> Lukes neue Endpoints (`GET /dialer/supervisor` + `/dialer/contacts/{id}/calls`, `fb045f9f`) ans FE gehängt. **Zwei mock-verdeckte Bugs gefunden:**
> (1) **`recent_calls` immer leer** — Lukes Query las `cc.contact_name` (Spalte existiert nicht in `dialer_campaign_contacts`); SQL-Fehler wurde still als WARN geschluckt → Feed leer. Gefixt: crm-`contacts`+`companies`-Join in `GetRecentCallsForTenant` (`c10f8d2f`, im dialer-Service).
> (2) **protojson lässt Null-Werte weg** (`EmitUnpopulated:false`) → `totals.active_agents`/`on_call` fehlten, `recent_calls` fehlte ganz → FE wäre bei leerem Dialer abgestürzt (`recent_calls.length` auf undefined). FE-Normalizer `api/dialer-normalize.ts` füllt die Defaults (`48b5daf9`), eingehängt in `dialer-client.ts`.
> **Verifikation:** Docker-Stack hoch (postgres/redis/auth/crm/gateway/dialer healthy), Dialer-Demo-Seed `backend/seeds/demo/dialer-demo.sql` (Kampagne + 2 Agents + 3 Outcomes + 5 Call-Sessions HEUTE → 5 Calls/2 Termine). Live gegen :8080: Supervisor zeigt KPIs (Aktive 0/Im Gespräch 0/Anrufe 5/Termine 2 — die 0 rendern statt undefined = Normalizer wirkt), Team mit calls_today, Letzte Anrufe voll. Screenshots `desktop/.qa-screenshots/dialer-supervisor/`, Skript `qa-dialer-supervisor-localbackend.mjs`.
> **LEHRE (Windows):** `curl | python -m json.tool` zeigt UTF-8-Umlaute fälschlich als Mojibake (`Ã¼`) — Python 3.14 liest stdin als cp1252, NICHT UTF-8. Echte Bytes mit `xxd` prüfen (`C3 BC` = sauberes ü). Es gab KEINEN Encoding-Bug; die Namen rendern sauber.
> **Gebaut-aber-nicht-meine-Lane:** `npm run build` war rot wegen `@livekit/track-processors` — nur stale node_modules, `npm install` fixt es (Dep ist in package.json). Danach Build grün.
> **PARALLEL:** Sub-Terminal baut `security`/DSGVO auf Branch `parallel/security` (Paket `.planning/parallel-batch/sub-security.md`, S-0 done, S-1…S-5 freigegeben). Main merged den Branch am Ende.
> **Docker läuft noch** (nur Main fasst Docker an). Offen für dialer: Contact-Calls-Detail im UI screenshotten (ContactDetailModal), Supervisor-Leer-Zustand sauber live testen (Pass B nutzte Cache).
>
> **DANACH verifiziert (selbe Session):**
> - **dashboard-Layout = echt, KEIN Code-Change nötig.** Store nutzt schon den echten `apiClient` (`/api/v1/dashboard/layout`, gateway-nativ, `response.JSON` — keine protojson-Falle). Roundtrip GET→PUT→GET live bestätigt (`{layout, active_widgets, is_custom, updated_at}`, persistiert). FE ruft `initFromServer`+`ensureDefaults` on mount. Abhaken in MASTER-PLAN.
> - **zeiterfassung/HR ENTBLOCKT (Backend läuft jetzt):** HR liegt auf dem **biz**-Service (`HRRoutes.ServiceName()="biz"`, HR+Finance teilen das biz-Binary). biz crashte (`failed to connect to minio for gobd archive`) → **minio + createbucket hochgefahren** → biz healthy → Gateway-Restart → HR-Endpoints antworten. Shapes geprobt: `/hr/time/balance` flach (clean), `/hr/time/status` flach, `/hr/time/entries` = `{entries:null,total:0}` (**`entries` ist `null` nicht `[]` bei leer → jede Konsum-Hook muss `?? []`**). `normalizeWireTimestamps` (für `{seconds,nanos}`) ist in `hr-client.ts` schon drin (Luke-Welle). **DONE (selbe Session):** HR-Demo geseedet (`hr-worktime-demo.sql`, 3 Einträge Mo–Mi). **Echter Backend-Bug gefunden+gefixt** (`b7242926`): `GetByID/List/GetActiveShift` scannten `correction_reason` (nullable, bei normalen Einträgen NULL) in einen `string` → `can't scan NULL into *string` → **Einträge-Liste warf 500 bei JEDEM echten Eintrag**; Fix = `COALESCE(correction_reason,'')` in den 3 SELECTs. API-verifiziert: `/hr/time/entries` liefert jetzt 3 Einträge (vorher 500), `/balance` korrekt -975. **FE-Screenshot ausstehend** — Dev-Server-Quirk: `electron-vite dev --mode localbackend` kam in MSW/Demo-Mode hoch (Stefan-Vogel statt Demo-Local), trotz korrektem Build-Flag; beim Dialer-Lauf lief dieselbe Instanz korrekt localbackend → flakiger Start, nächste Session sauber neu starten + verifizieren. Offen außerdem: `useAbsenceCalendar`-Hook (hr-hooks.ts:508) `select: (data)=>data.entries` ohne `?? []` → Crash-Risiko bei leerem `{entries:null}`. Stack-Stand: postgres/redis/auth/crm/gateway/dialer/biz/minio healthy.


> **★ UPDATE 2026-06-24 — B-12 DONE, Buchhaltung KOMPLETT echt (gepusht `4712857a`).**
> (1) Betrag-Fix `protoTaxBreakdown()`-Fallback in `biz_grpc.go` — `toProto{Invoice,Quote,CreditNote}` lesen jetzt das `tax_breakdown`-JSONB **oder** die Einzelspalten `subtotal/total_tax/gross_total` (der Seed füllt nur die Spalten → vorher 0,00 €). (2) Zweiter mock-verdeckter Bug gefixt: Dunning-Pfade in `finance-client.ts` **und** `mocks/handlers/finance.ts` Plural→Singular (`/finance/dunning` + `/dunning/config`) — das Gateway routet Singular, Plural gab 404 und der Mahnungen-Tab degradierte still zum Empty-State. Alle finanzen-Tabs live verifiziert: `desktop/.qa-screenshots/b12-finanzen/`. QA-Skripte: `qa-b12-finanzen-amounts.mjs`, `qa-b12-dunning-settle.mjs`.
> **Recovery-Lehre:** `docker compose up` OHNE `--no-deps` zieht den ganzen gateway-Dependency-Graph (alle 23 µSvc) rein und baut sie → WSL2-vmmem-Explosion (16 GB Maschine, RAM auf 1,4 GB) → Daemon-Hänger. Immer `up -d --no-deps <nur-was-man-braucht>`. Recovery: Docker Desktop killen + `wsl --shutdown` (gibt vmmem frei) + neu starten.

> **★ MOCK-EXIT — kontakte ist KOMPLETT echt (Referenz-Modul).** READ + voller CRUD (Create/Update/Delete) durch die echte UI gegen das lokale Backend, live verifiziert (Screenshots `desktop/.qa-screenshots/crud-*.png`). Casing-Entscheidung getroffen: **Option C** (per-Modul `dual()`-Adapter, kein globaler Transform — FE ist gemischt-casing). Vollständiger Bericht + Backend-Handover + camelCase-Risiko-Set für die nächsten Module: **`.planning/kontakte-mock-exit-DONE.md`**.
>
> **Diese Session neu:** `api/casing.ts` (`dual()`-Helper), `mocks/demo-mode-flag.ts` (Leaf-Flag), kontakte-Adapter mode-branched + position↔jobTitle, useContacts PATCH→PUT, Mock-Handler PATCH→PUT, Demo-User→admin (Seed idempotent). 3 weitere Mock-verdeckte Bugs gefunden+gefixt (PUT-Methode, position-Feld, custom_fields-Array).
>
> **Voriger Durchstich (Session #1):** Login + Kontakte-Liste echt, 2 Bugs gefixt (CORS-Idempotency-Key `d4a9c1a4`, Contact-Adapter snake_case `3979b142`).

## Was diese Session fertig wurde (gepusht, `043cb372`)
- **Lokales Backend läuft** via Docker (`deploy/docker/docker-compose.yml`): **postgres + redis + auth + crm + gateway** (Minimal-Subset; voller 24-Service-Stack crasht die Maschine → nur bauen, was man braucht). Gateway auf `:8080`, Migrationen bis **000226**.
- **Demo-Seeds** (`backend/seeds/demo/demo-seed.sql`, idempotent, Tenant `…0001`): 8 companies, 12 contacts, 8 deals, 3 projects, 10 tasks. **Finance-Block auskommentiert** (line_items ist separate `finance_line_items`-Tabelle → noch fixen).
- **Mock-Exit verifiziert end-to-end:** Login (`demo@local.test` / `Demo1234!`) → Kontakte mit echten Namen/Firmen/Avataren. QA-Skripte: `desktop/scripts/qa-mock-exit-kontakte.mjs` (Token-Inject) + `qa-mock-exit-login.mjs` (echter Login-Flow).
- **2 echte Bugs gefixt (Mocks hatten sie verdeckt):**
  - `fix(gateway)` `d4a9c1a4` — **CORS allow-headers** um `Idempotency-Key` ergänzt. HardMode verlangt den Header bei jeder Mutation, CORS verbot ihn → jede Mutation (Login/Create/Update) aus jedem Browser-Client blockiert. **Betrifft Luke/Prod.**
  - `fix(kontakte)` `3979b142` — Contact-**Adapter liest snake_case** (Gateway liefert `first_name`, OpenAPI-Typen sind camelCase = **Spec-Drift X-3**). Sonst Namen/Firma leer. **Muster betrifft JEDES Modul beim Mock-Exit.**
- **Tooling** `043cb372` — `RENDERER_VITE_DEV_BYPASS_AUTH=false` erzwingt echten Login im Dev-Build (`App.tsx`); `.planning/mock-exit-readiness-matrix.md` (Modul × Backend × Wire-Shape × Auth × RLS); `SESSION-RUNBOOK.md` Markt-Recherche als Pflicht-Schritt.
- **NICHT angefasst:** Login-Animation/`AuthLayout` (läuft auf main+Hetzner korrekt; das „falsche" C-Icon war nur ein Dev-Artefakt durch wiederholte Reloads → statischer Fallback statt Animation).

## Lokal wieder hochfahren (neues Terminal)
```bash
# 1. Docker-Backend (läuft evtl. noch — prüfen):
docker ps   # postgres/redis/auth/crm/gateway healthy?
# falls weg: cd "C:/Users/darie/Documents/KMU Hub"
docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/.env up -d --no-deps postgres redis auth crm gateway
# (.env liegt unter deploy/docker/.env — gitignored; Werte: deploy/docker/README.md + MIGRATION_DATABASE_URL)
# Seed (falls DB frisch): docker exec -i docker-postgres-1 psql -U kmuhub -d kmuhub --single-transaction < backend/seeds/demo/demo-seed.sql

# 2. FE gegen echtes Backend (Mode localbackend = DEMO_MODE=false + :8080 + echter Login):
cd desktop && npx electron-vite dev --mode localbackend
# Login: demo@local.test / Demo1234!  (Tenant …0001, sieht Seed-Daten)
# Hinweis: nur kontakte/firmen/deals live (crm); andere Module 503 (Service nicht gebaut)
```

## Was als Nächstes (Reihenfolge nach Hebel)
1. **~~OpenAPI-Casing~~ GELÖST** — Entscheidung Option C (per-Modul `dual()`). Globaler Transform verworfen (FE gemischt-casing, würde Tausende snake-Leser brechen). Casing-Risiko-Set + Pattern in `kontakte-mock-exit-DONE.md`.
2. **Nächstes Modul nach kontakte-Pattern echt schalten** — Reihenfolge nach Risiko-Set: crm/companies → crm/deals+pipeline-stages (DealInfo-Casing!) → work. Pro Modul: `dual()`-Adapter falls OpenAPI-getippt, Methode/Wire-Shape/Idempotency/RBAC gegen echtes Backend prüfen (nicht nur Mock).
3. **work + biz dazuholen** → Aufgaben/Projekte/Finanzen echt (`docker compose build work biz` + `up -d --no-deps`).
4. **Finance-Seed fixen** (line_items → `finance_line_items`-Tabelle) → finanzen-Demo nicht leer.
5. **RLS-scharf testen:** `DATABASE_URL` auf `kmuhub_app:app_dev` (statt Superuser) → wie Prod. Migration 000121, einmalig `ALTER ROLE kmuhub_app WITH PASSWORD 'app_dev'`.
6. **Luke-Handover offen** (siehe `kontakte-mock-exit-DONE.md`): contact-Schema zu dünn (9 Extra-Felder), OpenAPI-Spec-Drift contacts (PATCH→PUT, title→position, custom_fields-Array), Timeline-Endpoint hängt.

## Parallel: regulärer Bau-Track (MASTER-PLAN.md)
Der Mock-Exit ist Welle 1 (Echt-Schaltung) in Aktion. `MASTER-PLAN.md` bleibt die SSOT für die übrigen Wellen. SESSION-RUNBOOK-Zyklus gilt weiter.
