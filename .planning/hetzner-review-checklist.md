# Team-Review-Agenda (Cosmi-exe, app.zentria.tech)

> **Zweck:** EIN großer **Team-Review (Darien + Luke + Nico)** über den kompletten Bau-Stand — hands-on auf der über Hetzner laufenden Cosmi-exe.
> **Ablauf:** Alles geht auf `main`. Hier sammeln sich konkrete „klick hier → erwarte das"-Items. Abgehakt = vom zuständigen Reviewer geprüft + ok.
> **⚡ Darien-Entscheid 2026-07-19: SAMMEL-REVIEW AM ENDE** — keine Zwischen-Reviews; diese Liste sammelt ALLE Review-Items (RBAC-Batches, Modul-Reviews, Rest) und wird nach Abschluss der Bau-Strecke in einem Rutsch abgearbeitet.
> **⚡ Darien-Entscheide 2026-07-21 (Team-Review-Rahmen, siehe §0):** ① **Scope = ALLES** (alle Module, alle Bereiche inkl. settings, kompletter RBAC-Block R-1…R-6). ② **Split nach Modulen/Bereichen** auf die drei — nicht nach Kompetenz-Achse. ③ **Umgebung = Hetzner-Live**, erst NACH Lukes RBAC-Backend-Deploy. ④ **Self-Service-Customization-Tool ist NICHT Teil dieses Reviews** (noch nicht gebaut — eigener Bau-Block, spätere Runde).
> **Stand:** 2026-07-21.

## 0 · Team-Review — Rahmen (Darien 2026-07-21)

### 🚧 GATE (Vorbedingung — Review startet NICHT vorher)
Der komplette **RBAC-Block ist auf Hetzner-Live derzeit unsichtbar**: das Prod-Backend hat den **`permissions`-Endpoint noch nicht**, FE fällt dann auf Preset-Auflösung zurück. Vor dem Team-Review muss **Luke** liefern:
1. **RBAC-Backend deployen** — `permissions`-Endpoint + Enforcement (backend-gaps §RBAC R-3…R-6), sonst zeigen alle Rollen/Overrides/Vendor-Flows nichts Echtes.
2. **Prod-Demo-Seed** einspielen — sonst sind FE-mock-Module im Live-Mode leer (treffen echtes BE ohne Daten).
3. **Backend-Fixes deployen** — `9dfcf89e` (dialer recent-calls) + `b7242926` (hr correction_reason), s. Abschnitt C.
> ⏳ **Folge:** Solange dieser Gate offen ist, ist das Team-Review **blockiert**. Der aktive nächste Bau-Schritt (Darien) ist daher das **Self-Service-Customization-Tool** (frisches Terminal); das Review läuft, sobald Luke deployt hat.

### 👥 Reviewer-Zuständigkeiten (Vorschlag — beim Kickoff final zuweisen)
Jeder reviewt seine Module/Bereiche komplett durch (alle Personas, tote Buttons, leere Zustände, Raw-Keys, `{{var}}`, Detail-Modals, Sortierung, Moduleinstellungen). Streitfälle → kurzer Sync.

| Reviewer | Module / Bereiche |
|----------|-------------------|
| **Darien** (UX/Design/Daily-Use) | kontakte · dashboard · work · dokumente · kalender · profil · mails · kommunikation · wiki · **Moduleinstellungen (alle Module, Querschnitt)** |
| **Luke** (Backend/Daten/Security) | finanzen/Buchhaltung · team/HR (R-4) · Verwaltung/admin (R-6 Overrides) · security/DSGVO (R-5 Audit/Vendor/View-as/Templates) · zeiterfassung · dialer · **RBAC-Enforcement-Gesamtlogik + alle „Luke-Paket"-Nebenbefunde** |
| **Nico** (Polish/Branchen-Module) | inventar · einkauf · produktion · vertraege · helpdesk · schichten · fuhrpark · vermietung · rapporte · formulare · automatisierung · berichte |

> RBAC greift quer über alle Module — jeder prüft die RBAC-Personas (Elena/Max/Markus/Sarah/Nina/Thomas/admin) **in seinen** Modulen. Die RBAC-Batch-Blöcke (Abschnitt B) sind die dazu passenden „klick hier"-Szenarien.

## ✅ Voraussetzungen geklärt (Darien, 2026-06-24)
1. **Deploy:** macht **Luke** (irgendwann). Push auf `main` ≠ sofort live → Änderungen erscheinen auf Hetzner erst nach Lukes Deploy.
2. **Mode: LIVE** (echtes Prod-Backend, kein MSW).
   - **Folge A:** Backend-Echt-Schaltungen (dialer/dashboard/HR) sind nach Lukes Deploy prüfbar — zeigen **echte Prod-Daten** (ohne Prod-Seed ggf. leer, nicht kaputt).
   - **Folge B (wichtig):** **FE-mock-Module zeigen im Live-Mode NICHT ihre MSW-Demodaten** — sie treffen das echte Backend. Module mit fehlendem Backend (🔒, z. B. **security/DSGVO** und der **gesamte RBAC-Block**) sind auf Hetzner-Live **erst prüfbar, wenn Luke das Backend gebaut hat** (→ Gate §0). Bis dahin: reine FE/UX-Reviews dieser Module besser lokal.

---

## A · Jetzt prüfbar (FE/UX, Demo-Mode) — review-reife Module
> Die 15 FE-fertigen Module sind „review-reif". Pro Modul achten auf: tote Buttons, leere Zustände, Raw-i18n-Keys, `{{var}}`, Umlaut-Fehler, Detail-Modals (ganze Zeile klickbar, sticky Close), Sortierung.
- [ ] kontakte · [ ] calendar · [ ] dokumente · [ ] finanzen/Buchhaltung · [ ] work · [ ] team
- [ ] dashboard · [ ] vertraege · [ ] helpdesk · [ ] automatisierung · [ ] profil · [ ] mails · [ ] kommunikation · [ ] berichte · [ ] wiki

## B · Bereit zum Review (gemergt auf main)
- [ ] **RBAC R-3 Batch 1 — Aktions-Gating work/dokumente/kontakte/finanzen/wiki** (Session #17). **Demo-Mode prüfen** (Rechte kommen aus MSW; Live hat den permissions-Endpoint noch nicht → FE-Fallback löst Presets auf). Schnellster Weg: Verwaltung → Rollen → Rolle öffnen → „Als Rolle anzeigen"; echte Sessions über den ProfileSwitcher unten rechts. Achten auf:
  - Als **Elena (Nur Lesen)**: „Nur Ansicht"-Chip im Modul-Header · KEINE Erstellen/Bearbeiten/Löschen-Controls · finanzen-Tabs reduziert (kein Export/Mahnwesen/Ausgaben/Banking) · Zeilen-Menüs nur Details/PDF, **Versenden ausgegraut mit Hover-Hinweis** · dokumente-Kontextmenü: **Herunterladen ausgegraut mit Hinweis**, Edit-Einträge weg · kontakte: **Import-Button ausgegraut mit Hinweis**, kein vCard-Export im Menü.
  - Als **Max (Aushilfe/Extern)**: „Eingeschränkt"-Chip in Aufgaben · eigene zugewiesene Tasks abhakbar + kommentierbar, sonst nichts · Deep-Link `/#/finanzen` oder `/#/kontakte` → **„Kein Zugriff"-Seite** (nicht leer/Redirect).
  - Beträge-Maskierung `•••`: Rolle „Lager & Logistik" im Editor öffnen → Buchhaltung sichtbar + Rechnungen-Ansehen AN (amounts AUS lassen) → „Als Rolle anzeigen" → Buchhaltung: alle €-Werte als Punkte, Anzahl/Prozent normal.
  - Als **admin**: ALLES unverändert da (Regression).
  - ⚠ Bekannter Vorbestand (NICHT dieser Batch): Rechnungen-Liste zeigt 0,00 € je Zeile (Listen-API ohne Positionen) — separater Fix folgt.
- [ ] **RBAC R-3 Batch 2 — team-Aktionen · dashboard Ebene-2 · Verwaltungs-Tabs** (Session #18). **Demo-Mode prüfen**, Werkzeuge wie Batch 1 (Editor-Preview + ProfileSwitcher). Achten auf:
  - Als **Max (Extern)**: Dashboard zeigt NUR Projekt/Aufgaben/Dokumente-Karten, QuickActions nur „Neues Projekt/Dokument", keine Alert-Banner, kein Umsatz-Chart-Widget; Schnellaktionen-Widget ohne CRM-Buttons.
  - Als **Sarah (Teamleiter)**: Team ohne „Mitarbeiter erstellen" + ohne Lohnvorbereitung-Tab, Anfragen-Tab MIT Genehmigen; Karten-Menü ohne „Deaktivieren"; Dashboard ohne Buchhaltungs-Karte.
  - Als **Elena (Nur Lesen)**: fremdes Mitglieder-Profil = nur Übersicht-Tab mit Hinweis „Weitere Details sind für deine Rolle nicht sichtbar" (kein leeres weißes Modal, kein Dokumente-Tab).
  - Als **Nina (HR-Admin)**: Verwaltung zeigt NUR Benutzer+Rollen (kein Lizenz/Branding/IT/Sicherheit/Abrechnung/Integrationen); Direkt-URL `/#/admin/license` springt auf Benutzer.
  - Als **Thomas (IT-Admin)**: Verwaltung ohne Lizenz/Branding/Abrechnung; Sicherheit-Tab da, dort KEIN DSGVO/Auskunft-Sub-Tab (kein gdpr:execute); landet auf Audit-Log.
  - Als **admin**: alle 8 Verwaltungs-Tabs + alle 10 Security-Sub-Tabs (Start = Audit-Log), Team-Payroll „Prüfen & freigeben" da (Regression).
  - ⚠ Bekannte Nebenbefunde (NICHT dieser Batch): Benachrichtigungs-Widget zeigt extern CRM-Demo-Events (Seeds empfänger-agnostisch, → Luke-Paket) · Dashboard-Moduleinstellungen: 4 Widgets ohne Namens-Label (Vorbestand DashboardSettingsPanel).
- [ ] **RBAC R-3 Batch 3 — Aktions-Gating inventar/einkauf/produktion/vertraege/helpdesk** (Session #19). **Demo-Mode prüfen**, Werkzeuge wie Batch 1 (Editor-Preview + ProfileSwitcher unten rechts). Achten auf:
  - Als **Elena (Nur Lesen)**: „Nur Ansicht"-Chip in allen 5 Modulen · inventar ohne „Artikel hinzufügen"/CSV-Export/„Neue Inventur starten" · einkauf: Draft-Bestellung öffnen → **„An Lieferant senden" sichtbar aber ausgegraut mit Hover-Hinweis** (einzige disabled-Ausnahme), Bearbeiten/PDF/Stornieren weg · produktion ohne „Neuer Auftrag", Maschinen-Detail: Status als Text statt Dropdown · verträge ohne „Vertrag anlegen"/Zeilen-Menü-Mutationen.
  - Als **Markus (Mitarbeiter)**: helpdesk = **Requester-Modell**: nur EIGENE Tickets (5 Stück: 3 als Melder, 2 als Bearbeiter), „Neues Ticket" DA, Vorlagen-Button + Statistik-Tab WEG, kein Agent-Aktions-Block im Ticket-Detail · produktion: kein „Neuer Auftrag", aber **CSV/Laufkarte-Export bleibt** (Werker-Fall) + Schritt-Abhaken im Auftrags-Detail geht · inventar: Bestandsbewegung erfassen geht (Zeilen-Menü), aber kein Artikel-Anlegen/Export, im Bewegungs-Dialog KEINE „Korrektur"-Option · einkauf: Bestellung als Entwurf anlegen geht, Senden ausgegraut.
  - Als **Max (Extern)**: Deep-Links `/#/helpdesk`, `/#/produktion`, `/#/inventar` → „Kein Zugriff"-Seite.
  - Als **admin**: alle Create-Buttons/Tabs/Exporte unverändert (Regression); helpdesk zeigt ALLE Tickets, „Zugewiesen an" jetzt mit echten Namen statt `usr-…`-Ids (Adapter-Fix).
  - ⚠ Bekannte Nebenbefunde: helpdesk-Zuweisen-Dropdown führt weiterhin die zwei Freitext-Agenten „Marco Hartmann/Sandra Bürki" (Vorbestand, → Luke-Paket assignee-Referenzen) · KB-Artikel-Ansicht: `savedBody`-Crash-Vorbestand beim gespeicherten Artikel wurde als Drive-by gefixt.
- [ ] **RBAC R-3 Batch 4 — Aktions-Gating schichten/fuhrpark/vermietung/rapporte/dialer** (Session #20). **Demo-Mode prüfen**, Werkzeuge wie Batch 1 (Editor-Preview + ProfileSwitcher unten rechts). Achten auf:
  - Als **Elena (Nur Lesen)**: „Nur Ansicht"-Chip in allen 5 Modulen · schichten ohne CSV/PDF/„Woche veröffentlichen"/„Schicht zuweisen", Vorlagen-Tab bleibt (read) · fuhrpark: **Tracking-Tab WEG (GPS-Privacy — nur admin/manager)**, kein „Fahrzeug hinzufügen", keine Exporte · rapporte: sieht ALLE Berichte, Detail OHNE Genehmigen/Löschen/PDF/Unterschrift-Sektion · dialer: Nav nur Kampagnen+Dashboard, Cards ohne Start/Pause, `/#/dialer/workspace` leitet um.
  - Als **Markus (Mitarbeiter)**: schichten = **Swap-own-Modell**: Tausch-Anfragen-Tab zeigt NUR die eigene Anfrage (Felix↔Markus), keine Genehmigen-Buttons, Vorlagen-Tab weg, Chip „Eingeschränkt" · rapporte = **Autoren-Modell**: nur die EIGENEN 3 Berichte (Eingereicht/Abgelehnt/Entwurf), „Neuer Tagesbericht" + „Exportieren" (Werker-PDF) bleiben · fuhrpark: „Fahrt eintragen"/„Tanken eintragen"/„Schaden melden" DA, kein Tracking/Fahrzeug-anlegen/Export, kein „Wartung eintragen" · vermietung: Reservierung anlegen + Ausgeben/Zurücknehmen + Zustandsprotokoll DA, kein Objekt-CRUD/Stornieren/Export · dialer: Workspace + Dashboard + Kampagnen (read), `/#/dialer/supervisor` leitet um.
  - Als **admin**: **NEU — Rapport-Genehmigung**: eingereichten Bericht öffnen → Genehmigen/Ablehnen (Ablehnen verlangt Grund, Ersteller kann danach wieder einreichen) · schichten-Anfragen zeigen echte Namen + Initialen statt `usr-…`-Ids (Drive-by-Fix) · alle 5 dialer-Nav-Tabs + Kampagnen-Verwaltung unverändert (Regression).
  - Als **Max (Extern)**: Deep-Links `/#/schichten`, `/#/dialer` → „Kein Zugriff"-Seite.
  - ⚠ Bekannte Nebenbefunde: dialer „Bearbeiten" im Karten-Dropdown legt real eine NEUE Kampagne an statt zu editieren (Vorbestand, Edit-Pfad nie verdrahtet — Luke-Paket) · hr_admin hat jetzt schichten VOLL als HR-Domäne (Personio-Muster, spiegelt it_admin↔helpdesk).
- [ ] **RBAC R-3 Batch 5 — Aktions-Gating berichte/formulare/automatisierung + Mini-Kataloge kommunikation/kalender/zeiterfassung/infrastruktur** (Session #21, R-3-ABSCHLUSS). **Demo-Mode prüfen**, Werkzeuge wie Batch 1 (Editor-Preview + ProfileSwitcher unten rechts). Achten auf:
  - Als **Elena (Nur Lesen)**: „Nur Ansicht"-Chip in berichte/formulare/automatisierung · berichte: **DATEV-Tab DA** (Steuerberater-Fall! BWA/SuSa sichtbar) aber „DATEV Export"-Button WEG, kein „Neuer Bericht", Geplant-Tab weg · formulare: alle Listen lesbar, kein „Neues Formular", Karten-Menü ohne Mutationen · automatisierung: Liste + Protokoll lesbar, Status als grüner Punkt statt Toggle, Vorlagen-Tab weg · zeiterfassung: Team-View + Export-Button WEG · kalender: Terminbuchung-Tab WEG.
  - Als **Thomas (IT-Admin)**: berichte OHNE DATEV-Tab + OHNE Umsatz-KPI/Umsatzverlauf (**Finance-Privacy** — it_admin hat kein finance) · automatisierung + Infrastruktur VOLL (IT-Domäne) · kommunikation-Moduleinstellungen: nur Webhooks-Sektion (routing/channels/canned fehlen — keine leeren Hüllen).
  - Als **Markus (Mitarbeiter)**: berichte: „Neuer Bericht" DA, Dashboard OHNE Umsatz-KPIs/Umsatzverlauf (kein finance view) aber MIT Helpdesk/CRM/Lager-KPIs, DATEV+Geplant weg · **Editor-Scope-own**: eigener Entwurf „Helpdesk-Auslastung KW 24" MIT Bearbeiten-Toggle (ohne „Als fertig markieren"/Teilen), fremder „Monatsbericht Juni 2026" NUR lesbar (Titel nicht editierbar) · **automatisierung KOMPLETT WEG** (Nav-Eintrag + Deep-Link `/#/automatisierung` → „Kein Zugriff") · formulare: Eingänge einsehen + Status ändern (Außendienst), kein Designer/Lifecycle · zeiterfassung: keine fremden Korrekturen genehmigen, kein Export.
  - Als **admin**: berichte alle 4 Tabs + Editor-Lifecycle (Als fertig markieren → Freigeben → Archivieren) + Zeitpläne + Teilen · formulare Lifecycle/Share/Export · automatisierung Toggles + „Neue Automatisierung" · zeiterfassung Team-View + Export · Infrastruktur-Aktionen (Regression).
  - Als **Max (Extern)**: Deep-Links `/#/berichte`, `/#/formulare` → „Kein Zugriff"-Seite.
  - ⚠ Bekannte Nebenbefunde: berichte-Untertitel nennt Zeitplan-Anzahl auch ohne schedule:manage (Kosmetik) · Kanal-Anlegen im Chat jetzt hinter channel:manage (admin/manager/member — readonly/extern sehen das „+" nicht mehr) · Report-Builder (ReportBuilderShell) ist nirgends gemountet (Vorbestand, toter Code).
- [ ] **RBAC R-4 — HR-Datenkategorien-Tiefe (5 Schubladen × Zugriffsebene × Scope)** (Session #22). **Demo-Mode prüfen** (Editor-Preview + ProfileSwitcher unten rechts). Das ist die HR-Feinsteuerung: Personalakte in 5 Schubladen (Persönlich/Job/Gehalt/Dokumente/Abwesenheiten), Akten-Scope, Self-Service-Vorschläge, Offboarding-Kaskade. Achten auf:
  - Als **Sarah (Teamleiter/manager)**: sieht **Tims Akte OHNE Gehalts-Schublade** (nur Persönlich/Job/Dokumente/Abwesenheiten) · **Stefans Akte ist ganz zu** (nicht in ihrer Linie) · **Jonas tauct gar nicht in ihrer Personen-Liste auf** (Scope = eigene Linie 2 rauf/2 runter + Geschwister, nicht die ganze Firma) · Profil-Doku-Sektion zeigt KEINE Gehaltsabrechnungs-PDFs (hinter salary:view).
  - Als **Thomas (IT-Admin)**: sieht die **volle Mitarbeiter-Liste** (team:directory:full), aber beim Öffnen einer Akte **„Weitere Details sind für deine Rolle nicht sichtbar"** statt HR-Feldern (Marktlücken-USP — Verzeichnis ohne HR-Inhalt).
  - Als **Markus/Stefan (member)**: **Self-Service** — eigene Stammdaten ändern geht als **Vorschlag mit „Änderung ausstehend"-Lock** (nicht direkt), eigene **echte Gehaltsabrechnungen (9.800 €)** sichtbar; Aushilfe/extern haben KEIN Self-Service (`team:self:propose` fehlt).
  - Als **Nina (HR-Admin)**: **Anfragen-Inbox** zeigt Self-Service-Vorschläge als **Alt→Neu-Karte** (z. B. Markus) zum Genehmigen/Ablehnen.
  - Als **admin**: **Offboarding-Dialog** — Mitarbeiter-Profil → Austritt = **2-Schritt-Bestätigung** (kein One-Click), bei Führungskraft **Warnung + Pflicht-Nachfolger** (z. B. Jonas-Warnung „X unterstellte Mitarbeiter"), Kaskade sichtbar. Profil-**Bearbeiten** funktioniert (Sektions-Edit, war vorher tot verdrahtet).
  - Als **Max (Extern)**: `/#/team`-HR-Tiefe = NoAccess.
  - ⚠ Bekannte Nebenbefunde (Luke-Paket): Personalakte-Doku-Seeds sind für ALLE identisch (Tim zeigt Stefans Arbeitsvertrag) · Anfragen-Tab zeigt teils „Unbekannt"-Requester · Seed-Kollisionen (usr-e6/e9/e14/e16) · Trainings-Tab-Template-Keys treffen camelCase nicht (defaultValue deckt nur DE).
- [ ] **RBAC R-5 — Audit-Log-Live · Zentria-Setup-Zugang (GDAP-light) · Branchen-Templates · View-as** (Session #24, Teil 1). **Demo-Mode prüfen**. Vier Bausteine, alle unter Verwaltung → Sicherheit bzw. Rollen. Achten auf:
  - **Audit-Log-Live** (Sicherheit → Audit): Rolle im Editor ändern (z. B. „Team Lead" → Read Only) → im Audit-Log erscheint **live** ein Eintrag mit **Alt/Neu-Delta-Panel** („VORHER Team Lead → NACHHER Team Lead, Read Only"), Retention-Zeile „24 Monate, unveränderlich". Im **Rollen-Modul** gibt es einen vorgefilterten **„Protokoll"-Sub-Tab** (nur role.*/permission.*-Events, gleiche Quelle).
  - **Zentria-Setup-Zugang** (Sicherheit → Anbieter-Zugriff / GDAP): Zentria stellt eine Zugriffs-Anfrage (Kunde wählt KEINE Laufzeit — Zentria setzt Dauer, **Hard-Cap 30 Tage**) mit **Anlass + Ticket-Ref (#4711) + Scope als Bereichs-Auswahl** und **auto-generierten „Zugriff auf / KEIN Zugriff auf"-Listen** (Default-Preset „Setup-Standard" = alles außer HR/Lohn). Bei **sensiblen Bereichen (HR/Lohn) Pflicht-Checkbox** — Bestätigen-Button bleibt disabled ohne Haken. Kunde kann **Terminvorschlag** zurücksenden (Banner). **Header-Badge** „noch X Tage" mit Deep-Link zur Verwaltung; **Entzug** landet im Verlauf + im Audit-Log.
  - **Branchen-Templates** (Rollen → „Aus Vorlage"): Galerie mit 3 Sets (Handwerk/Dienstleister/Handel, 12 Rollen). Vorlage wählen (z. B. „Monteur/Geselle") → **Pre-Fill in den NORMALEN Erstell-Dialog**, alles editierbar, wird eine **Custom-Rolle** (System-Rollen unangetastet).
  - **View-as** (Benutzer-Detail → „Als Benutzer anzeigen"): admin sieht die App aus Sicht eines Users, **z-71-Banner + Verlassen**, Audit-Event; **Guardrail**: nicht auf sich selbst, nicht auf Admins.
  - Als **Elena (readonly) / Max (extern)**: **kein** Anbieter-Zugriff-Badge, kein View-as, NoAccess auf die Verwaltungs-Flächen.
  - ⚠ Backend offen (Luke-Paket): Audit-Write als Middleware an allen Mutations-Routen + old/new-JSONB · vendor_access-Tabellen + Status-Maschine v3 + 30-Tage-Cap + Expiry-Job + 422-sensitive_ack · Impersonation-Token mit Guardrails · Partner-Portal/Ticket-Kopplung = späteres Deliverable.
- [ ] **RBAC R-6 — Per-User-Overrides (echter Entzug pro Benutzer, allow + deny)** (Session #24, Teil 2). **Demo-Mode prüfen**. **Alleinstellungs-Feature** — kein Wettbewerber am Markt kann echten per-User-Entzug (Odoo/Google rein additiv, Salesforce nur innerhalb einer Gruppe). Einstieg: Benutzer-Detail → „Berechtigungen anpassen" (Route `/admin/users/:id/overrides`). Achten auf:
  - **Seed-Fall Markus** hat einen Allow-Override `work:project:edit`, den seine Rolle NICHT gibt → in der Benutzerliste **„Angepasst"-Badge**, Filter **„Nur angepasste"** zeigt nur Markus.
  - **Editor** (dediziert, Zwei-Pane wie Rollen-Editor): jede Zeile **TRI-STATE** — Geerbt (grau, folgt Rolle) / Erlaubt (grün) / Entzogen (rot, durchgestrichen). Read-only bis Schalter **„Benutzerdefiniert"** an. Toggle-Zyklus Geerbt↔Override, Reset pro Zeile + „Alle zurücksetzen", Staged-Footer.
  - **Speichern** → in der **Effektiv-Ansicht** (eigenes Profil + Admin-Modal) erscheinen entzogene Rechte **durchgestrichen mit „Persönlich entzogen"**, gesetzte mit **„Persönlich"-Chip**.
  - **Rollen-Wechsel bei vorhandenen Overrides** → **Bestätigungs-Dialog** „trotz Anpassungen" (Overrides bleiben erhalten, werden nie stumm verworfen).
  - **„Alle zurücksetzen"** → Badge/Filter verschwinden (hasOverrides = false).
  - Rechte-Prüfung: Override-Editor nur für **admin** (`admin:user_override:manage`); **Nina (hr_admin)** darf Rollen zuweisen, aber **NICHT** Overrides bearbeiten.
  - ⚠ Backend offen (Luke-Paket): user_permission_overrides-Tabelle · Resolution Rollen-Union → Overrides (deny/allow, Override gewinnt pro Key) · hasOverrides + deniedByOverride + `?base=1`-Param · Audit override_set/removed · Overrides dürfen auch die Modul-Sichtbarkeit (Ebene 1) schalten.
- [ ] **security / DSGVO** ✅ gemergt (`43fecf37`, S-1…S-5) — **Demo-Mode prüfbar** (reine FE/MSW-Arbeit). Hub `/admin/security` (10 Sub-Tabs). Achten auf: alle Seiten crashfrei, keine Raw-Keys (DE+EN), DSGVO-Flows durchklickbar — Audit (Filter/Export), DSAR Art.15 (Cross-Modul-Suche + Export), Export Art.15/20 (Genehmigen/Download + Frist), Erasure Art.17 (Preview/Execute + Legal-Hold-Hinweis), Retention (DACH-Fristen + Auto-Löschung-Toggle), Sessions (beenden), Vault, PW-Policy, IP-Access, 2FA. Sub-Bericht: `.planning/parallel-batch/qa-security.md`.
- [ ] **zeiterfassung** (Main, echt-geschaltet) — siehe C.

## C · Backend-Echt-Schaltung (lokal verifiziert — Hetzner erst nach Deploy + Prod-Seed)
> Diese sind gegen das **lokale** Backend + lokale Demo-Seeds live verifiziert (Screenshots lokal). Auf Hetzner brauchen sie (a) Deploy der Backend-Fixes, (b) Prod-Demo-Daten.
- [x] **dialer-Supervisor** — lokal verifiziert (2 BE-Bugs gefixt: recent-calls-SQL + protojson-Null). Hetzner: nach Deploy + Dialer-Seed.
- [x] **dashboard-Layout** — Persistenz-Roundtrip lokal verifiziert (war schon verkabelt).
- [x] **zeiterfassung/HR** — BE-Bug gefixt (NULL `correction_reason` brach die Einträge-Liste). Hetzner: nach Deploy + HR-Seed.
- **Backend-Fixes, die deployt werden sollten (Luke):** `9dfcf89e` dialer recent-calls · `b7242926` hr correction_reason. Beide brechen sonst echte Daten still.

## D · Erledigt-Bestätigung (von Darien)
_(hier hakt Darien ab, was er auf Hetzner geprüft + ok befunden hat)_
