# Wettbewerbsanalyse: Software-Tools fuer DACH-KMUs (5-200 MA)

> **Recherche:** 2026-02-16
> **Autor:** Darien (Design Research)
> **Confidence:** MEDIUM — Basiert auf Training-Data (Stand Mai 2025). Preise und Features koennen sich seit Mai 2025 geaendert haben. Keine Live-Verifizierung moeglich (WebSearch/WebFetch blocked).
> **Zweck:** Funktions-Tiefenanalyse der Konkurrenz, um KMU Hub Features priorisieren zu koennen

---

## Inhaltsverzeichnis

1. [Projektmanagement](#1-projektmanagement)
2. [Zeiterfassung](#2-zeiterfassung)
3. [ERP-Systeme](#3-erp-systeme)
4. [Schichtplanung](#4-schichtplanung)
5. [Fuhrpark](#5-fuhrpark)
6. [Bau/Handwerk](#6-bauhandwerk)
7. [Kommunikation/Video](#7-kommunikationvideo)
8. [E-Signatur](#8-e-signatur)
9. [Vermietung/Reservierung](#9-vermietungreservierung)
10. [Funktions-Tiefenanalyse pro Kategorie](#10-funktions-tiefenanalyse)
11. [Branchenspezifische Analyse](#11-branchenspezifische-analyse)
12. [Build vs. Integrate](#12-build-vs-integrate)
13. [TOP 10 Kategorien die KMUs vereinen wollen](#13-top-10-vereinen)

---

## 1. Projektmanagement

### 1.1 Tool-Uebersicht

#### Jira (Atlassian, Australien)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~15-20% bei Tech-KMUs, <5% bei Non-Tech-KMUs |
| **Preise** | Free (10 User), Standard 8.15 USD/User/Mo, Premium 16 USD/User/Mo |
| **DSGVO/DSG** | Cloud-Daten in EU moeglich (Frankfurt/Dublin), Standard Contractual Clauses, US-Mutterkonzern = Risiko fuer strenge CH-DSG-Auslegung |

**Kernfunktionen im Detail:**
- Kanban-Board mit Swimlanes (nach Assignee, Priority, Epic, Sprint), WIP-Limits pro Spalte, Board-Filter
- Scrum-Board mit Velocity Chart, Burndown, Sprint-Planung, Story Points
- Backlog-Verwaltung mit Ranking (Drag), Epics, Labels, Versionen
- Roadmap-View (Timeline) mit Epics als Balken, Abhaengigkeiten als Pfeile (Premium)
- JQL (Jira Query Language) fuer komplexe Filter — extrem maechtig, aber lernintensiv
- Automation Rules (>100 vorgefertigte Trigger, z.B. "Auto-assign bei Label-Aenderung")
- Custom Fields (Text, Number, Select, Cascading Select, Date, User, etc.)
- Workflow-Editor: Beliebige Statusuebergaenge mit Bedingungen, Validatoren, Post-Functions
- Zeiterfassung: eingebaut (Log Work), aber rudimentaer — kein Timer, manuell
- Confluence-Integration fuer Wiki/Docs
- 3000+ Marketplace Apps

**Bloat fuer KMUs:**
- JQL ist Overkill — 90% der KMU-Nutzer brauchen keine Query-Sprache
- Workflow-Editor zu komplex — KMUs brauchen max. 4-5 Status
- Scrum-Artefakte (Velocity, Sprint Planning) irrelevant fuer Handwerker/Gastro
- Advanced Roadmap (Premium) zu teuer fuer Features die KMUs nie nutzen
- Berechtigungsschema extrem granular — Admin-Albtraum fuer kleine Teams

---

#### Monday.com (Israel)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~10-15%, stark wachsend durch Marketing |
| **Preise** | Free (2 User), Basic 9 EUR/User/Mo, Standard 12 EUR/Mo, Pro 19 EUR/Mo, Enterprise: Anfrage |
| **DSGVO/DSG** | EU-Rechenzentren verfuegbar, israelisches Unternehmen, Auftragsverarbeitungsvertrag vorhanden |

**Kernfunktionen im Detail:**
- "Boards" statt Projekte — flexibel als Tabelle, Kanban, Gantt, Kalender, Chart darstellbar
- 30+ Spaltentypen (Status, People, Date, Timeline, Numbers, Formula, Mirror, Connect Boards)
- Automationen mit visueller Wenn-Dann-Logik (z.B. "Wenn Status = Fertig, benachrichtige Manager")
- Gantt-Chart (Standard-Plan): Abhaengigkeiten, kritischer Pfad, Baseline-Vergleich
- Dashboards mit 15+ Widget-Typen (Chart, Number, Battery, Timeline)
- Gaeste-Zugang: Externe mit eingeschraenkten Rechten einladen (1 Board sichtbar)
- Formulare: Oeffentliche Formulare die direkt Items in Boards erstellen
- Docs: Echtzeit-Dokumente eingebettet, verknuepft mit Board-Items
- Zeiterfassung (Standard): Timer pro Item, Berichte
- Workload-View: Kapazitaet pro Mitarbeiter visuell
- 200+ Integrations (Slack, Gmail, HubSpot, etc.)

**Bloat fuer KMUs:**
- Dashboard-Widgets z.T. nur mit Pro-Plan nutzbar
- "Connect Boards" + "Mirror Columns" = Enterprise-Komplexitaet die verwirrt
- CRM-Modul ist Aufpreis (Monday CRM = eigenes Produkt, 10+ EUR/User/Mo extra)
- WorkDocs ist ein halbes Google Docs — weder Fisch noch Fleisch

---

#### Asana (USA)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~8-12%, besonders bei Agenturen/Dienstleistern |
| **Preise** | Free (15 User), Premium 10.99 EUR/User/Mo, Business 24.99 EUR/Mo |
| **DSGVO/DSG** | EU-Datenresidenz verfuegbar, US-Unternehmen |

**Kernfunktionen im Detail:**
- Listview, Board (Kanban), Timeline (Gantt), Kalender als Ansichten pro Projekt
- Sections (Gruppen innerhalb eines Projekts) fuer Strukturierung
- Subtasks (multi-level), Abhaengigkeiten (blockiert/blockiert von)
- Custom Fields (Number, Text, Select, Multi-Select, Date, People)
- Rules/Automations (Premium): Trigger-Action-basiert, z.B. "Task abgeschlossen → verschiebe in Spalte"
- Portfolios (Business): Ueberblick ueber mehrere Projekte, Statusampeln
- Goals (Business): OKR-artiges Zielsystem mit Fortschritt aus Projekten
- Workload (Business): Kapazitaetsplanung pro Person
- Formulare: Intake-Formulare die Tasks erstellen
- Berichterstattung: Dashboards mit Charts, Status-Updates
- Gaeste: Einladung mit eingeschraenkten Rechten

**Bloat fuer KMUs:**
- Portfolios + Goals + Workload = Business-Plan Pflicht (24.99 EUR) fuer Features die nur Agenturen brauchen
- "My Tasks" als Hauptnavigation verwirrt — KMU-Mitarbeiter wollen Projekte sehen, nicht "ihre Tasks"
- Kein eingebauter Timer / Zeiterfassung

---

#### Meistertask (MeisterLabs, Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-8%, stark bei deutschen KMUs durch DSGVO-Fokus |
| **Preise** | Free (3 Projekte), Pro 13.50 EUR/User/Mo, Business 27.50 EUR/Mo |
| **DSGVO/DSG** | Server in Frankfurt, deutsches Unternehmen, ISO 27001, **staerkstes DSGVO-Argument in der Kategorie** |

**Kernfunktionen im Detail:**
- Kanban-Board (Hauptansicht) — schlicht, intuitiv, wenige Optionen
- Timeline (Business): Gantt-artig aber simpel
- Agenda: Persoenliche Tagesansicht (aehnlich "My Tasks")
- Tags, Farbcodes, Beobachter, Checklisten, Datei-Anhaenge
- Automationen (Business): Einfache Regeln (weniger als Monday/Asana)
- MindMeister-Integration (Mindmaps → Tasks)
- Zeiterfassung eingebaut (Pro): Timer-Button pro Task, Zeitberichte
- Gaeste-Zugang (Business) — aber nur lesend

**Bloat fuer KMUs:**
- Sehr wenig Bloat — das ist die Staerke. Aber auch limitiert: Keine Custom Fields, keine echten Abhaengigkeiten (nur "Beziehungen"), kein Portfolio
- Business-Plan fuer Timeline + Gaeste = teuer fuer das Gebotene

---

#### Stackfield (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~3-5%, Nische: datenschutzbewusste KMUs |
| **Preise** | Starter 11 EUR/User/Mo, Business 16 EUR/Mo, Premium 27 EUR/Mo |
| **DSGVO/DSG** | Server in Deutschland, Ende-zu-Ende-Verschluesselung, deutsches Unternehmen, **"Made in Germany"-Siegel** |

**Kernfunktionen im Detail:**
- Projektmanagement + Kommunikation in einem Tool (aehnlich wie KMU Hub!)
- Task-Module: Listen, Kanban, Gantt, Kalender
- Echtzeit-Chat + Gruppen-Chat
- Videocalls (eingebaut, E2E-verschluesselt)
- Dateiablage mit Versionierung
- Globale Suche ueber alle Module
- Custom Fields (Business)
- Automatisierungen (Premium)
- Gaeste (externe Partner)
- Audit-Log, 2FA, SSO (Premium)

**Bloat fuer KMUs:**
- UI wirkt "deutsch-funktional" — nicht so polished wie Monday/Asana
- E2E-Verschluesselung macht Suche langsam (serverside search eingeschraenkt)
- Kein CRM-Modul — reines PM + Kommunikation

---

#### ClickUp (USA)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-8%, wachsend durch Pricing |
| **Preise** | Free (100MB), Unlimited 7 USD/User/Mo, Business 12 USD/Mo |
| **DSGVO/DSG** | US-Unternehmen, EU-Rechenzentrum angekuendigt aber Umsetzung unklar |

**Kernfunktionen im Detail:**
- "Everything App" — versucht ALLES zu sein: PM, Docs, Whiteboards, Chat, Goals, Time Tracking
- 15+ Ansichten: List, Board, Gantt, Calendar, Timeline, Workload, Table, Map, Mind Map, Embed, Form, Doc
- Spaces → Folders → Lists → Tasks → Subtasks (5-Level-Hierarchie)
- Custom Fields: 15+ Typen inkl. Formula, Rollup, Relationships
- Automationen: 100+ Templates, Custom Trigger/Action
- ClickUp Docs: Echtzeit-Dokumente mit Rich-Text, eingebettete Tasks
- ClickUp Whiteboards: Freeform + Tasks einbettbar
- Zeiterfassung: Eingebaut, Timer, manuell, Berichte
- Goals/OKRs: Targets mit Auto-Progress aus Tasks
- Dashboards: 50+ Widget-Typen
- ClickUp AI (Aufpreis): Zusammenfassungen, Texte generieren

**Bloat fuer KMUs:**
- **Massiver Bloat** — 5-Level-Hierarchie verwirrt 10-Mann-Teams total
- Performance-Probleme bei vielen Ansichten/Automationen (bekannt)
- Feature-Ueberladung: Es gibt zu viel, Onboarding dauert Wochen
- DSGVO-Status unklar — problematisch fuer Schweizer KMUs

---

#### Wrike (USA, seit 2021: Citrix)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~3-5%, eher bei Mittelstand (50+ MA) |
| **Preise** | Free (5 User), Team 9.80 USD/User/Mo, Business 24.80 USD/Mo |
| **DSGVO/DSG** | EU-Rechenzentrum (Amsterdam), US-Mutterkonzern |

**Kernfunktionen im Detail:**
- Task-Listen, Board (Kanban), Gantt-Chart, Tabellenansicht
- Custom Workflows (Statusreihenfolge definieren)
- Cross-Tagging: Tasks in mehreren Projekten gleichzeitig (Ordner-System)
- Request Forms: Intake-Formulare fuer externe Anfragen
- Proofing: Visuelles Feedback auf Bildern/PDFs/Videos (fuer Agenturen)
- Dashboards: Anpassbare Widgets
- Time Tracking (Business): Timer + manuell
- Resource Management (Business): Kapazitaetsplanung
- Blueprints: Projekt-Templates mit Automatisierungen

**Bloat fuer KMUs:**
- Cross-Tagging (Folder-System statt Projekte) ist konzeptionell verwirrend
- Proofing ist Agentur-spezifisch — 90% der KMUs brauchen das nicht
- Mindest-Paketgroessen (Team = 2-15 User) — unflexibel

---

#### Basecamp (USA)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | <3%, eher in US beliebt |
| **Preise** | 15 USD/User/Mo (Basecamp) oder 299 USD/Mo flat (Basecamp Pro, unlimited User) |
| **DSGVO/DSG** | US-Unternehmen, keine EU-Rechenzentren |

**Kernfunktionen im Detail:**
- Message Board (Announcements), To-Dos (einfache Listen mit Zuweisungen)
- Schedule (einfacher Kalender), Campfire (Gruppenchat), Automatic Check-ins
- Docs & Files: Einfache Dateiablage
- Hill Charts: Fortschrittsvisualisierung (einzigartig — zeigt "am Berg oben" vs. "den Huegel runter")
- Card Table: Einfaches Kanban (neueres Feature)

**Bloat fuer KMUs:**
- Fast nichts ist Bloat — Basecamp ist bewusst minimalistisch
- Problem: ZU minimalistisch fuer viele KMUs. Kein Gantt, keine Custom Fields, keine Automationen, keine Zeiterfassung
- Flat-Fee-Modell (299/Mo) interessant fuer groessere KMUs, aber kein Einzelpreis unter 15 USD

---

### 1.2 PM-Zusammenfassung: Was KMU Hub besser machen kann

| Was Konkurrenz gut macht | Was Konkurrenz schlecht macht | KMU Hub Chance |
|--------------------------|-------------------------------|----------------|
| Monday: Flexible Ansichten (Board/Gantt/Kalender in einem) | Alle: CRM ist separates Tool oder Aufpreis | CRM + PM in einem! |
| Meistertask: DSGVO-konform, simple UI | ClickUp/Jira: Feature-Ueberladung | Einfachheit + Tiefe je nach Branche |
| Stackfield: PM + Chat + Video | Stackfield: Kein CRM, UI altmodisch | Das gleiche Konzept, aber schoener + CRM |
| Asana: Elegante UX, gutes Onboarding | Basecamp: Zu wenig Features | Goldener Mittelweg |
| ClickUp: Alles-in-einem inkl. Docs | ClickUp: Performance, DSGVO, Komplexitaet | Alles-in-einem ABER mit Branchenprofilen die Komplexitaet filtern |

---

## 2. Zeiterfassung

### 2.1 Tool-Uebersicht

#### Clockodo (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~15-20%, Marktfuehrer in DE-KMU-Segment |
| **Preise** | Zeiterfassung 8 EUR/User/Mo, Zeiterfassung Plus 12 EUR/Mo (mit Projekten + Budgets) |
| **DSGVO/DSG** | Server in Deutschland, deutsches Unternehmen, AV-Vertrag sofort verfuegbar |

**Kernfunktionen im Detail:**
- Stoppuhr-Timer (Desktop, Web, Mobile): Ein-Klick-Start, Projekt/Aufgabe zuweisen
- Manuelle Eintraege (Nachtrag)
- Kunden + Projekte + Aufgaben als 3-Level-Hierarchie
- Stundensaetze pro Projekt, Kunde oder Mitarbeiter
- Abwesenheiten: Urlaub, Krankheit, Feiertage, Ueberstunden
- Berichte: Stundenauswertung nach Kunde/Projekt/Mitarbeiter/Zeitraum
- Excel/CSV-Export, PDF-Stundenzettel
- Soll/Ist-Vergleich: Arbeitszeitkonto mit Ueber/Unterstunden
- API fuer Integrationen
- DATEV-Export (direkt ins Steuerbuero)
- Genehmigungsworkflow fuer Abwesenheiten

**Bloat fuer KMUs:**
- Wenig Bloat — Clockodo ist fokussiert
- "Plus"-Plan fuer Projekt-Budgets ist Aufpreis der nervt

---

#### Toggl Track (Estland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~10-15%, beliebt bei Freelancern + kleinen Teams |
| **Preise** | Free (5 User), Starter 10 EUR/User/Mo, Premium 20 EUR/Mo |
| **DSGVO/DSG** | EU-Unternehmen (Estland), GDPR-konform, Server in EU |

**Kernfunktionen im Detail:**
- Timer mit einem Klick — minimalistisch, schnell
- Autotracker (Desktop): Erkennt automatisch genutzte Programme/Websites
- Pomodoro-Timer eingebaut
- Projekte, Kunden, Tags, Billable/Non-Billable
- Berichte: Summary, Detailed, Weekly — filterbar, exportierbar
- Rounding (Zeiten auf 15min runden)
- Erinnerungen ("Du hast heute noch nicht getrackt")
- Kalender-Integration (Google Calendar Eintraege als Zeitvorschlaege)
- Team Dashboard (wer arbeitet gerade woran)
- Toggl Plan (separates Tool): Gantt/Timeline

**Bloat fuer KMUs:**
- Autotracker = Mitarbeiter-Ueberwachung — kontrovers in DACH
- Toggl Plan ist ein EXTRA Tool mit extra Kosten (nicht integriert!)
- Keine Abwesenheitsverwaltung — braucht Drittool

---

#### TimeCamp (Polen)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~3-5%, weniger bekannt in DACH |
| **Preise** | Free (1 User), Starter 3.99 USD/User/Mo, Premium 6.99 USD/Mo |
| **DSGVO/DSG** | EU-Unternehmen (Polen), GDPR-konform |

**Kernfunktionen im Detail:**
- Timer + automatische Zeiterkennung (erkennt Projekte anhand genutzter Apps)
- Budgets + Kostenvoranschlaege pro Projekt
- Abrechenbare vs. nicht-abrechenbare Stunden
- Rechnungserstellung direkt aus Zeiteintraegen (!) — USP
- GPS-Tracking (Mobile) fuer Aussendienst
- Abwesenheiten (einfach)
- Berichte + Export (CSV, PDF, Excel)
- Integrationen: Jira, Asana, Monday, Trello, etc.

**Bloat fuer KMUs:**
- Automatische App-Erkennung = Datenschutz-Bedenken
- GPS-Tracking in DE/CH arbeitsrechtlich heikel

---

#### Harvest (USA)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~3-5%, eher bei Agenturen |
| **Preise** | Free (1 User, 2 Projekte), Pro 12 USD/User/Mo |
| **DSGVO/DSG** | US-Unternehmen, EU-Datenspeicherung moeglich, DPA verfuegbar |

**Kernfunktionen im Detail:**
- Timer + manuell, Wochenansicht (Timesheet-Stil)
- Projekte mit Budget (Stunden oder Geld)
- Rechnungen direkt aus Stunden generieren
- Ausgaben-Tracking (Belege fotografieren)
- Team-Kapazitaetsplanung (Forecast — separates Tool!)
- Berichte: Projekt-Profitabilitaet, Team-Auslastung
- QuickBooks/Xero Integration (US-fokussiert)

**Bloat fuer KMUs:**
- Forecast als Extra-Tool nervt
- US-Buchhaltungsintegrationen in DACH unbrauchbar
- Keine DATEV-Schnittstelle

---

#### Kimai (Open Source, Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-8% bei Self-Hosted KMUs |
| **Preise** | Kostenlos (Self-Hosted), Cloud: 5-15 EUR/User/Mo |
| **DSGVO/DSG** | Open Source, Self-Hosted = volle Kontrolle, deutsches Projekt |

**Kernfunktionen im Detail:**
- Timer + manuelle Eintraege
- Projekte, Kunden, Aktivitaeten (3-Level)
- Stundensaetze, Budget-Tracking
- Berichte + Export
- Plugin-System fuer Erweiterungen (Rechnungen, Abwesenheiten, etc.)
- API fuer Integrationen
- Multi-Mandanten-faehig

**Bloat fuer KMUs:**
- Self-Hosted braucht IT-Kompetenz — KMUs haben das oft nicht
- UI ist funktional aber nicht modern
- Plugins z.T. kostenpflichtig

---

#### ZEP (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~8-12%, etabliert bei Beratungen/IT-Dienstleistern |
| **Preise** | ZEP compact 6 EUR/User/Mo, Professional 15 EUR/Mo, Enterprise: Anfrage |
| **DSGVO/DSG** | Server in Deutschland, deutsches Unternehmen, ISO 27001 |

**Kernfunktionen im Detail:**
- Zeiterfassung: Timer + Wochenstundenzettel
- Projekte mit Budgets, Phasen, Meilensteinen
- Reisekostenabrechnung (!) — DE/AT/CH-spezifische Pauschalen
- Urlaubsverwaltung mit Genehmigungsworkflow
- Personalplanung: Wer wird wo eingesetzt?
- Controlling-Berichte (Deckungsbeitragsrechnung!)
- DATEV-Export
- JIRA-Integration

**Bloat fuer KMUs:**
- UI altbacken
- Enterprise-Features (Controlling, Deckungsbeitrag) irrelevant fuer Handwerker
- Onboarding komplex

---

### 2.2 Zeiterfassungs-Zusammenfassung

| Feature | Clockodo | Toggl | Kimai | ZEP | KMU Hub Status |
|---------|----------|-------|-------|-----|----------------|
| Timer (Start/Stop) | Ja | Ja | Ja | Ja | **Gebaut** |
| Manuelle Eintraege | Ja | Ja | Ja | Ja | **Gebaut** |
| Projekte zuordnen | Ja | Ja | Ja | Ja | **Gebaut** |
| Soll/Ist-Stunden | Ja | Nein | Plugin | Ja | **Gebaut** |
| Abwesenheiten | Ja | Nein | Plugin | Ja | **Gebaut** |
| DATEV-Export | Ja | Nein | Nein | Ja | **Fehlt** |
| Reisekosten | Nein | Nein | Nein | Ja | **Fehlt** |
| Rechnungen aus Stunden | Nein | Nein | Plugin | Nein | **Fehlt** (Bruecke zu Buchhaltung) |
| Team-Dashboard | Ja | Ja | Ja | Ja | **Gebaut** |
| Reports/Export | Ja | Ja | Ja | Ja | **Gebaut** |
| Mobile Timer | Ja | Ja | Ja | Ja | **Fehlt** (erst mit React Native) |

---

## 3. ERP-Systeme

### 3.1 Tool-Uebersicht

#### SAP Business One (SAP, Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~20-25% bei groesseren KMUs (50+ MA), <5% bei Kleinst-KMUs |
| **Preise** | Lizenz: 1.700-3.200 EUR/User einmalig + Wartung 20%/Jahr, Cloud: 72-126 EUR/User/Mo |
| **DSGVO/DSG** | Deutsches Unternehmen, eigene Rechenzentren DE |

**Kernfunktionen im Detail:**
- Finanzwesen: Hauptbuch, Kreditoren, Debitoren, Kostenrechnung, Budgetierung
- Einkauf: Bestellungen, Wareneingaenge, Lieferanten, Preisvergleich
- Vertrieb: Angebote → Auftraege → Lieferscheine → Rechnungen (Belegkette)
- Lagerverwaltung: Bestaende, Chargen, Seriennummern, MRP
- Produktion: Stuecklisten, Fertigungsauftraege
- CRM (eingebaut): Leads, Opportunities, Kontakte, Aktivitaeten
- Business Intelligence: Crystal Reports, Dashboards
- Customizing: SDK + Formulareditor

**Bloat fuer KMUs:**
- **ALLES** ist Bloat fuer KMUs unter 50 MA — die Komplexitaet ist enorm
- Implementierung dauert 3-12 Monate, kostet 30.000-100.000 EUR
- Braucht Partner fuer Setup/Customizing (kein Self-Service)
- UI ist 2000er-Aera, Desktop-Client Pflicht

---

#### Odoo (Belgien)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-10%, wachsend, besonders bei technikaffinen KMUs |
| **Preise** | Community Edition: Kostenlos (Self-Hosted), Online: 24.90 EUR/User/Mo (1 App) + 7.40 EUR/User je weitere App |
| **DSGVO/DSG** | EU-Unternehmen (Belgien), Self-Hosted moeglich |

**Kernfunktionen im Detail:**
- Modularer Aufbau: 40+ offizielle Module (CRM, Sales, Purchase, Inventory, MRP, HR, Helpdesk, Website, E-Commerce, ...)
- CRM: Pipeline, Leads, Angebote direkt als PDF generieren
- Verkauf: Angebotsvorlagen, Auftragsbestaetigung, Unterschrift
- Einkauf: Bestellungen, Automatische Nachbestellungen, Lieferantenbewertung
- Lager: Barcode-Scanner, Multi-Warehouse, Routen, Chargenverfolgung
- Fertigung: BOMs, Arbeitsplaene, Qualitaetskontrolle
- Buchhaltung: Hauptbuch, Rechnungen, Zahlungsabgleich, Steuermeldung
- HR: Recruiting, Abwesenheiten, Bewertungen, Fuhrpark
- Helpdesk: Tickets, SLA, Wissensdatenbank
- Website-Builder + E-Commerce
- Studio (Low-Code): Apps/Module visuell anpassen

**Bloat fuer KMUs:**
- Preismodell ist Falle: "1 App = 24.90" klingt guenstig, aber wer CRM + Sales + Lager + HR braucht zahlt 50+ EUR/User
- Community Edition fehlen kritische Features (keine Studionanpassung, kein Multi-Company)
- Customizing braucht Odoo-Partner (aehnliches Problem wie SAP, nur guenstiger)
- 40+ Module = Ueberforderung ohne Partner

---

#### weclapp (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~3-5%, stark bei Online-Haendlern |
| **Preise** | CRM 15 EUR/User/Mo, ERP 59 EUR/User/Mo, Warenwirtschaft 59 EUR/User/Mo |
| **DSGVO/DSG** | Server in Deutschland, deutsches Unternehmen, DSGVO-konform |

**Kernfunktionen im Detail:**
- CRM: Kontakte, Firmen, Opportunities, Pipeline
- Warenwirtschaft: Artikel, Bestellungen, Lieferscheine, Rechnungen (Belegkette)
- Lagerverwaltung: Bestaende, Seriennummern, Chargen, Kommissionierung
- Buchhaltung: FiBu, DATEV-Export, Mahnwesen
- E-Commerce-Anbindung: Shopify, WooCommerce, Amazon, eBay
- POS (Kassensystem)
- REST-API

**Bloat fuer KMUs:**
- E-Commerce-Fokus — fuer Dienstleister/Handwerker wenig relevant
- 59 EUR/User/Mo fuer ERP ist teuer bei 20+ Usern
- Keine PM-Features (Projekte, Tasks)

---

#### myfactory (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~3-5%, eher bei Fertigungsbetrieben |
| **Preise** | Ab 55 EUR/User/Mo (abhaengig von Modulen) |
| **DSGVO/DSG** | Server in Deutschland, deutsches Unternehmen |

**Kernfunktionen im Detail:**
- ERP + CRM + Webshop in einer Suite
- Finanzwesen: FiBu, Kostenrechnung, Zahlungsverkehr
- Warenwirtschaft: Bestellwesen, Bestandsfuehrung, Kommissionierung
- PPS (Produktionsplanung): Stuecklisten, Arbeitsgaenge, Auftragsfertigung
- CRM: Kontakte, Aktivitaeten, Kampagnen
- DMS: Dokumentenablage
- Webshop (B2B/B2C)

**Bloat fuer KMUs:**
- PPS ist nur fuer Fertigungsbetriebe relevant
- UI datiert
- Kein Chat/Video/PM

---

#### Haufe X360 (Deutschland, basiert auf Acumatica)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~2-3%, relativ neu am Markt |
| **Preise** | Auf Anfrage (geschaetzt 70-120 EUR/User/Mo) |
| **DSGVO/DSG** | Haufe-Gruppe = deutscher Konzern, EU-Rechenzentren |

**Kernfunktionen im Detail:**
- Cloud-ERP auf Basis von Acumatica (US-Plattform, DACH-Lokalisierung durch Haufe)
- Finanzwesen: Hauptbuch, AP/AR, Anlagenbuchhaltung
- Vertrieb + CRM: Pipeline, Angebote, Auftraege
- Einkauf + Lager: Bestellwesen, Inventar
- Projektmanagement: Projekte, Budgets, Zeiterfassung
- Service Management: Field Service, Helpdesk
- Sehr anpassbar (Low-Code Plattform)

**Bloat fuer KMUs:**
- Preislich eher obere KMU-Klasse
- Implementierung braucht Partner
- Relativ unbekannt in der Breite

---

### 3.2 ERP-Zusammenfassung: Relevanz fuer KMU Hub

| ERP-Kernfunktion | KMU-Relevanz | In KMU Hub? |
|------------------|-------------|-------------|
| Rechnungen/Angebote | HOCH | **Gebaut** (Buchhaltung) |
| Warenwirtschaft/Lager | MITTEL | **Gebaut** (Inventar) |
| Einkauf/Bestellungen | MITTEL | **Gebaut** (Einkauf) |
| FiBu/Hauptbuch | HOCH (aber komplex) | **Nicht geplant** — Integration mit Bexio/Abacus |
| Produktion/BOM | NIEDRIG | **Gebaut** (Produktion) |
| DATEV-Export | HOCH fuer DE | **Gebaut** (Buchhaltung) |
| Belegkette (Angebot→Rechnung) | HOCH | **Teilweise** (kein Angebot→Auftrag→Rechnung Flow) |
| E-Commerce-Anbindung | MITTEL | **Nicht geplant** |

**Erkenntnis:** KMU Hub sollte KEIN ERP werden. Die Buchhaltungs-Integrationen (Bexio, Abacus) sind der richtige Weg. Was fehlt: Eine saubere Belegkette (Angebot → Auftrag → Lieferschein → Rechnung) fuer KMUs die kein Bexio haben.

---

## 4. Schichtplanung

### 4.1 Tool-Uebersicht

#### Papershift (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~20-25%, Marktfuehrer in DE fuer Schichtplanung |
| **Preise** | Starter 3 EUR/User/Mo, Professional 5 EUR/Mo (Zeiterfassung), Enterprise 6 EUR/Mo (HR) |
| **DSGVO/DSG** | Server in Deutschland, deutsches Unternehmen, TUeV-zertifiziert |

**Kernfunktionen im Detail:**
- Drag & Drop Schichtplan: Woche/Monat, Mitarbeiter x Tage Raster
- Schichtvorlagen: Wiederkehrende Muster speichern und anwenden
- Verfuegbarkeitsabfrage: Mitarbeiter geben Wuensche ab
- Tauschboerse: Mitarbeiter tauschen Schichten untereinander (mit Genehmigung)
- Zuschlaege: Nacht/Wochenende/Feiertag automatisch berechnet
- Qualifikationsfilter: Nur qualifizierte Mitarbeiter fuer bestimmte Schichten
- Stempeluhr (App oder Terminal): Check-in/Check-out mit GPS-Verifizierung
- Pausenverwaltung (automatische Berechnung nach Arbeitszeit)
- Abwesenheiten: Urlaub, Krankheit, Mutterschutz
- Ueberstundenverwaltung mit Arbeitszeitkonto
- Berichte + DATEV-Export
- Mitarbeiter-App (Self-Service: Schichten sehen, tauschen, Abwesenheiten beantragen)

**Bloat fuer KMUs:**
- Terminal-Hardware (Stempeluhr) = Extra-Kosten + Setup
- Enterprise-Plan fuer HR-Features (Urlaubsplanung) ist Aufpreis
- Kein PM, kein CRM, kein Chat — NUR Schichten

---

#### Shiftbase (Niederlande)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-10% in DACH |
| **Preise** | Free (1 Standort, 75 MA), Premium 4.50 EUR/User/Mo |
| **DSGVO/DSG** | EU-Unternehmen (NL), GDPR-konform, AV-Vertrag |

**Kernfunktionen im Detail:**
- Schichtplanung (Drag & Drop)
- Zeiterfassung (Timer/Terminal)
- Abwesenheitsverwaltung
- Verfuegbarkeit + Tauschboerse
- Open-Shift-Vergabe (Mitarbeiter melden sich fuer offene Schichten)
- Lohnexport (verschiedene Formate)
- Standort-basierte Stempeluhr

**Bloat fuer KMUs:**
- Free-Tier ist grosszuegig — aber Premium fuer Export/Berichte noetig
- Weniger Features als Papershift

---

#### gastromatic (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~10-15% in Gastro/Hotellerie, sehr stark in der Nische |
| **Preise** | Ab 2.50 EUR/User/Mo (Basis) bis 5 EUR (Pro) |
| **DSGVO/DSG** | Deutsches Unternehmen, Server in Deutschland |

**Kernfunktionen im Detail:**
- Gastro-spezifisch: Oeffnungszeiten als Planungsbasis
- Umsatzbasierte Personalplanung (!)  — Schichtbedarf aus Umsatzprognose
- Trinkgeldverwaltung
- MiLoG-konforme Zeiterfassung (Mindestlohn-Dokumentation)
- Jugendarbeitsschutz automatisch beruecksichtigt
- Schichtplanung mit Qualifikationen (Koch, Service, Bar)
- Digitale Personalakte
- Lohnvorbereitung fuer DATEV/Agenda/Addison

**Bloat fuer KMUs (ausserhalb Gastro):**
- Komplett Gastro-spezifisch — fuer andere Branchen nutzlos

---

#### Planday (Daenemark)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5% in DACH, staerker in Nordeuropa |
| **Preise** | Starter 2.99 EUR/User/Mo, Plus 4.99 EUR/Mo, Pro 7.99 EUR/Mo |
| **DSGVO/DSG** | EU-Unternehmen (DK), GDPR-konform |

**Kernfunktionen im Detail:**
- Schichtplanung mit Vorlagen
- Stempeluhr (App + Tablet-Kiosk)
- Payroll-Integration (DACH: limitiert)
- Revenue-Tracking (Umsatz vs. Personalkosten)
- Kommunikation: Integrierter Chat + Announcements
- Open Shifts + Swap
- Compliance: Arbeitszeitgesetz-Regeln hinterlegen

**Bloat fuer KMUs:**
- Revenue-Tracking nur relevant fuer Gastro/Retail
- DACH-Payroll-Integrationen schwaecher als Papershift

---

### 4.2 Schichtplanungs-Zusammenfassung

| Feature | Papershift | gastromatic | KMU Hub Status |
|---------|-----------|-------------|----------------|
| Wochenraster (MA x Tage) | Ja | Ja | **Gebaut** |
| Schichtvorlagen | Ja | Ja | **Gebaut** |
| Tauschboerse | Ja | Ja | **Gebaut** |
| Verfuegbarkeit MA | Ja | Ja | **Gebaut** |
| Zuschlaege berechnen | Ja | Ja | **Fehlt** |
| Qualifikationsfilter | Ja | Ja | **Fehlt** |
| Stempeluhr/Terminal | Ja | Ja | **Fehlt** (nur Timer) |
| DATEV-Export | Ja | Ja | **Fehlt** (fuer Schichten) |
| MiLoG-Konformitaet | Ja | Ja | **Fehlt** |
| App fuer Mitarbeiter | Ja | Ja | **Fehlt** (erst React Native) |

---

## 5. Fuhrpark

### 5.1 Tool-Uebersicht

#### Vimcar (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~25-30%, Marktfuehrer |
| **Preise** | Fahrtenbuch 13.90 EUR/Fahrzeug/Mo, Flotte 19.90 EUR/Mo, Fuhrpark 29.90 EUR/Mo |
| **DSGVO/DSG** | Deutsches Unternehmen, Server in Deutschland |

**Kernfunktionen im Detail:**
- Elektronisches Fahrtenbuch (finanzamtkonform!): OBD-Dongle im Auto, automatische Aufzeichnung
- GPS-Tracking: Live-Positionen, Routenverlauf
- Kilometerstand automatisch
- Fahrtenkategorisierung: Privat/Dienstlich (1%-Regel vs. Fahrtenbuch)
- Tankkarten-Integration
- Schadenmeldung (Fotos + Beschreibung)
- Fuehrerscheinkontrolle: Erinnerungen
- Wartungsplanung: TUeV, Service, HU/AU Termine
- Kostenuebersicht pro Fahrzeug (Total Cost of Ownership)
- Fahrerausweis-Management
- REST-API

**Bloat fuer KMUs:**
- OBD-Dongle = Hardware-Kosten pro Fahrzeug (einmalig ~70-100 EUR)
- GPS-Tracking = Datenschutz-Diskussion mit Betriebsrat
- Fahrtenbuch-Funktion irrelevant fuer Firmen die nur 1%-Regel nutzen

---

#### Fleetize (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-10% |
| **Preise** | Ab 14.90 EUR/Fahrzeug/Mo |
| **DSGVO/DSG** | Deutsches Unternehmen |

**Kernfunktionen im Detail:**
- Fahrzeugverwaltung: Stammdaten, Dokumente, Versicherungen
- Tankverwaltung: Tankbelege, Verbrauchsanalyse
- Wartungsmanagement: Terminplanung, Werkstatthistorie
- Reifenmanagement
- Schadenverwaltung
- Fuehrerscheinkontrolle
- TCO-Analyse (Total Cost of Ownership)

**Bloat fuer KMUs:**
- Kein GPS-Tracking (bewusst)
- Kein Fahrtenbuch
- Reifenmanagement = Nische

---

#### Avrios (Schweiz, jetzt: Fleetcor/Corpay)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-8%, staerker bei CH-Unternehmen |
| **Preise** | Auf Anfrage (geschaetzt 15-30 EUR/Fahrzeug/Mo) |
| **DSGVO/DSG** | Schweizer Unternehmen (jetzt US-Mutterkonzern), CH-Server |

**Kernfunktionen im Detail:**
- Fahrzeugverwaltung mit Dokumenten-Upload
- Kostenanalyse: TCO, Benchmark mit Marktdaten
- Tankkarten-Integration (automatischer Import)
- Automatische Belegeerfassung (KI-basiert)
- CO2-Reporting
- Fahrerportal (Self-Service)
- Versicherungsmanagement

**Bloat fuer KMUs:**
- US-Mutterkonzern = DSGVO/DSG-Fragezeichen
- Benchmark-Features fuer Kleinstflotten irrelevant
- CO2-Reporting = Nice-to-have, kein Must-have

---

### 5.2 Fuhrpark-Zusammenfassung

| Feature | Vimcar | KMU Hub Status |
|---------|--------|----------------|
| Fahrzeug-Stammdaten | Ja | **Gebaut** |
| Wartungsplanung | Ja | **Gebaut** |
| Tankbuch | Ja | **Gebaut** |
| GPS-Tracking | Ja | **Gebaut** (UI, braucht Device-Integration) |
| Kostenanalyse | Ja | **Gebaut** |
| Fahrtenbuch (finanzamtkonform) | Ja | **Fehlt** (kritisch fuer DE-Markt!) |
| OBD-Dongle | Ja | **Fehlt** (Hardware) |
| Fuehrerscheinkontrolle | Ja | **Fehlt** |
| Schadenmeldung | Ja | **Fehlt** |

**Erkenntnis:** Ein finanzamtkonformes Fahrtenbuch ist DAS Killerfeature im Fuhrpark. Ohne OBD-Dongle oder GPS-Device aber nicht automatisiert loesbar. KMU Hub koennte ein manuelles Fahrtenbuch anbieten (mit den richtigen Pflichtfeldern: Datum, Abfahrt, Ankunft, Zweck, km-Stand) — das reicht vielen KMUs.

---

## 6. Bau/Handwerk

### 6.1 Tool-Uebersicht

#### PlanRadar (Oesterreich)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~15-20% im Bausegment |
| **Preise** | Starter 29 EUR/User/Mo, Business 79 EUR/Mo, Enterprise: Anfrage |
| **DSGVO/DSG** | EU-Unternehmen (Wien), GDPR-konform |

**Kernfunktionen im Detail:**
- Maengelmanagement: Mangel auf Grundriss markieren (Pin auf Plan), Foto, Beschreibung, Zustaendigkeit
- Bautagebuch: Digitale Tagesberichte mit Wetter, Personal, Geraete, Aktivitaeten
- Planverwaltung: PDF-Plaene hochladen, Versionen verwalten
- Aufgaben auf Plaenen: Tasks direkt auf dem Gebaeudegrundriss
- Protokolle: Baustellenbegehungs-Protokolle generieren (PDF)
- Offline-Faehigkeit (Mobile App)
- Checklisten / Formulare
- Analytics: Maengelstatistik, Bearbeitungszeiten

**Bloat fuer KMUs:**
- Enterprise-Preis fuer groessere Teams
- Fokus auf Generalunternehmer/Bauherren — kleiner Handwerker braucht weniger

---

#### 123erfasst (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~10-15% bei Bau-KMUs in DE |
| **Preise** | Ab 49 EUR/Mo (Basis) + 5-10 EUR/Mitarbeiter |
| **DSGVO/DSG** | Deutsches Unternehmen, Server in Deutschland |

**Kernfunktionen im Detail:**
- Digitale Zeiterfassung auf der Baustelle: GPS-Check-in, QR-Code
- Bautagebuch: Tagesbericht mit Wetter, Personal, Maschinen, Materialverbrauch
- Rapportwesen: Aufmass (Laenge, Flaeche, Volumen)
- Regieleistungen dokumentieren
- Fotodokumentation: Bilder mit GPS-Tag + Timestamp
- Lohnanbindung: DATEV, Sage, Addison
- Geraeteverwaltung: Standorte, Einsatzzeiten

**Bloat fuer KMUs:**
- Rein Bau-fokussiert — kein PM, kein CRM
- Aufpreis-Modell (Basis + pro MA) wird teuer

---

#### Craftnote (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-10% bei kleinen Handwerksbetrieben |
| **Preise** | Free (3 Projekte), Pro 9.99 EUR/User/Mo, Business 29.99 EUR/Mo |
| **DSGVO/DSG** | Deutsches Unternehmen, Server in DE |

**Kernfunktionen im Detail:**
- Projektmanagement fuer Handwerker: Projekte mit Phasen, Aufgaben, Fotos
- Baustellendokumentation: Fotos + Zeitstempel
- Zeiterfassung pro Projekt/Baustelle
- Team-Chat pro Projekt
- Kunden-Sharing: Kunden sehen Projektfortschritt
- Rechnungserstellung (einfach)
- Notizen + Checklisten

**Bloat fuer KMUs:**
- Limitierte Free-Version (nur 3 Projekte)
- Keine Warenwirtschaft, kein Aufmass

---

#### Sorba (Schweiz)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~15-20% bei CH-Bau-KMUs, ~0% ausserhalb CH |
| **Preise** | Auf Anfrage (typisch 50-100 CHF/User/Mo) |
| **DSGVO/DSG** | Schweizer Unternehmen, CH-Server, DSG-konform |

**Kernfunktionen im Detail:**
- ERP speziell fuer Schweizer Bau: Kalkulation, Offerten, Ausmass (NPK-Positionen!)
- NPK (Normpositionen-Katalog): Schweizerische Baupositionen-Standard
- Devisierung nach SIA 451 (Schweizer Bau-Normen)
- Rapportwesen: Tagesrapporte, Regieberichte
- Lohnbuchhaltung (CH: AHV/IV/EO/ALV)
- Anlagenbuchhaltung
- Baukostenplanung nach BKP (Baukostenplan)

**Bloat fuer KMUs (ausserhalb CH-Bau):**
- 100% Schweiz-spezifisch — nicht uebertragbar
- Teuer + komplexe Implementierung

---

### 6.2 Bau/Handwerk-Zusammenfassung

| Feature | PlanRadar | 123erfasst | KMU Hub Status |
|---------|-----------|------------|----------------|
| Bautagebuch/Rapporte | Ja | Ja | **Gebaut** (Rapporte-Modul) |
| Aufmass (Flaeche/Volumen) | Nein | Ja | **Gebaut** (Rapporte-Modul) |
| Fotodokumentation | Ja | Ja | **Gebaut** (Rapporte: Photos) |
| Maengelmanagement | Ja | Nein | **Fehlt** |
| Plan-auf-Grundriss | Ja | Nein | **Fehlt** (komplex) |
| GPS-Zeiterfassung | Nein | Ja | **Fehlt** (erst Mobile) |
| NPK/SIA-Normen (CH) | Nein | Nein | **Fehlt** (Nische!) |
| DATEV-Lohn | Nein | Ja | **Fehlt** |

---

## 7. Kommunikation/Video

### 7.1 Tool-Uebersicht

#### Microsoft Teams (USA)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~50-60% (!!), dominant durch Microsoft 365 |
| **Preise** | In Microsoft 365 Business Basic (6 EUR/User/Mo) enthalten |
| **DSGVO/DSG** | US-Unternehmen, EU-Rechenzentren (DE/NL), umstrittener Datenschutz-Status, EU-Kommission 2024: Microsoft hat Datenschutz-Zusagen gemacht |

**Kernfunktionen im Detail:**
- Chat: 1:1, Gruppen, Channels (Teams), Threads, @Mentions, Reactions
- Video/Audio: Meetings (bis 300 Teilnehmer), Screensharing, Background Blur, Together Mode
- Kalender: Integriert mit Outlook/Exchange
- Dateien: OneDrive/SharePoint Integration, Co-Authoring
- Apps/Tabs: Einbetten von Drittapps (Planner, Power BI, etc.)
- Planner: Einfaches Kanban (in Teams eingebettet)
- Whiteboard: Microsoft Whiteboard eingebettet
- Recording + Transkription (Business-Plan)
- Breakout Rooms
- Teams Phone: Telefonie (Aufpreis)

**Bloat fuer KMUs:**
- SharePoint ist ein Labyrinth — Dateien "verschwinden" fuer nicht-technische User
- Teams-Channels vs. Teams-Chats vs. Outlook verwirrt
- Admin-Center extrem komplex
- Microsoft 365 als Voraussetzung = Vendor Lock-In

---

#### Slack (USA, Salesforce)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~10-15%, eher bei Tech/Startup/Agentur |
| **Preise** | Free (90 Tage Historie), Pro 7.25 EUR/User/Mo, Business+ 12.50 EUR/Mo |
| **DSGVO/DSG** | US-Unternehmen, EU-Rechenzentren verfuegbar (seit GovSlack/Enterprise) |

**Kernfunktionen im Detail:**
- Channels: Oeffentlich/Privat, organisiert nach Thema/Projekt/Team
- Threads: Diskussionen in Threads (verhindert Channel-Chaos)
- Huddles: Spontane Audio/Video-Anrufe (Slack-eigenes Feature)
- Slack Connect: Channels mit externen Firmen teilen
- Workflow Builder: Einfache Automationen ohne Code
- Canvas: Kollaborative Docs innerhalb Channels
- 2400+ App-Integrationen (Jira, GitHub, Salesforce, etc.)
- Emoji Reactions, Custom Emoji
- Durchsuchbare Historie (Pro: unbegrenzt)
- Slack AI: Zusammenfassungen, Suche

**Bloat fuer KMUs:**
- 90 Tage im Free-Plan = Datenverlust-Angst
- Slack Connect = Enterprise-Feature
- Preis pro User laeuft bei Teams weg (3 vs. 7 EUR/User)
- Kein Kalender, keine Dateiverwaltung (nur als Anhaenge)

---

#### Zoom (USA)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~20-25% fuer Video, weniger fuer Chat |
| **Preise** | Free (40min Limit), Pro 13.33 EUR/User/Mo, Business 18.33 EUR/Mo |
| **DSGVO/DSG** | US-Unternehmen, EU-Rechenzentren (Frankfurt), nach 2020-Skandal verbessert |

**Kernfunktionen im Detail:**
- Video-Meetings: Bis 1000 Teilnehmer, Breakout Rooms, Polls, Q&A
- Zoom Chat: Persistenter Chat mit Channels
- Zoom Whiteboard: Zeichenflaeche
- Zoom Phone: Cloud-Telefonie
- Zoom Docs: Neueres Feature — AI-gestuetztes Docs
- Zoom AI Companion: Meeting-Zusammenfassungen, Action Items
- Virtual Backgrounds, Noise Cancellation

**Bloat fuer KMUs:**
- Zoom versucht "alles zu sein" (Chat + Phone + Docs) — aber niemand nutzt Zoom Chat als Hauptchat
- Zoom Phone teuer + komplex
- Zoom Docs = halbes Produkt

---

#### Jitsi (Open Source, 8x8/Community)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~3-5%, beliebt bei datenschutzbewussten KMUs |
| **Preise** | Kostenlos (Self-Hosted), 8x8 JaaS: ab 0.003 USD/min |
| **DSGVO/DSG** | Self-Hosted = volle Kontrolle, Open Source |

**Kernfunktionen im Detail:**
- Video/Audio Meetings (WebRTC)
- Screensharing
- Chat waehrend Meeting
- Aufzeichnung (Self-Hosted: Server-seitig)
- E2E-Encryption (experimental)
- Lobby / Warteraum
- Polls, Hand-Raising
- SIP/H.323 Gateway (fuer Telefon-Einwahl)

**Bloat fuer KMUs:**
- Self-Hosted braucht IT-Kompetenz
- Kein persistenter Chat
- Kein Kalender, keine Dateiablage

---

#### BigBlueButton (Open Source)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | <2%, eher Bildungssektor |
| **Preise** | Kostenlos (Self-Hosted) |
| **DSGVO/DSG** | Self-Hosted = volle Kontrolle |

**Kernfunktionen im Detail:**
- Video/Audio Meetings
- Whiteboard (eingebaut, guter Funktionsumfang)
- Praesentations-Modus (Folien hochladen)
- Breakout Rooms
- Polling, Shared Notes
- Recording

**Bloat fuer KMUs:**
- Bildungs-fokussiert (Lehrerzimmer-UX)
- Setup komplex, braucht potenten Server

---

#### Element / Matrix (UK/Open Source)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | <2%, Nische bei Behoerden/Sicherheitsbewussten |
| **Preise** | Free (Self-Hosted), Element Cloud: 5 USD/User/Mo |
| **DSGVO/DSG** | Self-Hosted moeglich, E2E-Encrypted by default, Matrix-Protokoll offen |

**Kernfunktionen im Detail:**
- Chat: Raeume (wie Channels), E2E-verschluesselt
- Video/Audio: 1:1 + Gruppen (basiert auf Jitsi/Element Call)
- Bridges: Verbindung zu Slack, Teams, IRC, etc.
- Spaces: Organisationsstruktur fuer Raeume
- Federation: Server-uebergreifende Kommunikation

**Bloat fuer KMUs:**
- Federation ist Overkill
- UX nicht business-tauglich
- Bridges instabil

---

### 7.2 Kommunikations-Zusammenfassung

| Feature | Teams | Slack | KMU Hub Status |
|---------|-------|-------|----------------|
| Chat (Channels + DM) | Ja | Ja | **Gebaut** (Luke) |
| Threads | Ja | Ja | **Geplant** |
| Video-Calls | Ja | Huddles | **Geplant** (LiveKit) |
| Screensharing | Ja | Ja | **Geplant** (LiveKit) |
| Whiteboard | Ja | Nein | **Geplant** (D8) |
| Kalender | Ja (Outlook) | Nein | **Gebaut** |
| Dateien | Ja (SharePoint) | Nein | **Gebaut** (Dokumente) |
| @Mentions | Ja | Ja | **Geplant** |
| Reactions | Ja | Ja | **Geplant** |
| Meeting-Aufzeichnung | Ja | Nein | **Geplant** (LiveKit) |
| Teams/Slack Bridge | N/A | N/A | **Geplant** (Phase 15, KILLER-USP!) |

**Erkenntnis:** KMU Hub mit Chat + Video + Kalender + Dateien in einem ist genau das was Stackfield versucht, aber KMU Hub hat zusaetzlich CRM + PM + Branchenmodule. Die Teams/Slack-Bridge (H9) ist DER Differentiator — kein Konkurrent macht das.

---

## 8. E-Signatur

### 8.1 Tool-Uebersicht

#### Skribble (Schweiz)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~10-15% in CH, ~3-5% in DE |
| **Preise** | Free (begrenzte Signaturen), Business 20 CHF/Mo Basis + 2-2.50 CHF/Signatur (EES), QES: 10-25 CHF/Signatur |
| **DSGVO/DSG** | Schweizer Unternehmen, Server in der Schweiz, DSG+DSGVO konform, **ZertES-konform (CH)** |

**Kernfunktionen im Detail:**
- 3 Signatur-Level: EES (einfach), FES (fortgeschritten), QES (qualifiziert = handschriftgleich)
- QES ueber Video-Identifikation (SwissID, ID-now)
- Batch-Signing: Mehrere Dokumente auf einmal signieren
- API fuer Integration (REST API, gut dokumentiert)
- Visual Signature: Handschrift hochladen/zeichnen
- Audit Trail: Wer hat wann signiert (revisionssicher)
- Erinnerungen: Automatische Nachfass-Mails
- Templates: Signatur-Positionen auf PDFs vordefinieren

**Bloat fuer KMUs:**
- QES (qualifiziert) braucht Video-Ident — Aufwand fuer Gelegenheitsnutzer
- Pro-Signatur-Kosten laeufen bei vielen Dokumenten weg
- EES reicht fuer 90% der KMU-Faelle, aber KMUs wissen das nicht

---

#### DocuSign (USA)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~15-20% in DACH (global Marktfuehrer) |
| **Preise** | Personal 10 EUR/Mo (5 Docs), Standard 25 EUR/User/Mo, Business Pro 40 EUR/Mo |
| **DSGVO/DSG** | US-Unternehmen, EU-Rechenzentren (Frankfurt), DPA verfuegbar. **ABER: US-Cloud-Act = Restrisiko** |

**Kernfunktionen im Detail:**
- E-Signatur: EES + AES (Advanced), QES ueber Partner
- Envelope-System: Dokument + Signatur-Felder + Empfaenger als "Umschlag"
- Workflows: Sequentielle Unterschriften (erst A, dann B)
- PowerForms: Self-Service Signatur-Links
- API (umfangreich, viele SDKs)
- CLM (Contract Lifecycle Management): Vertragsmanagement (Enterprise)
- ID Verification: Ausweis-Pruefung vor Signatur

**Bloat fuer KMUs:**
- CLM = Enterprise-Feature
- US-Cloud-Act ist Problem fuer strenge DSGVO/DSG-Auslegung
- Standard-Plan limitiert auf "Senden" — Empfangen kostenlos (verwirrendes Modell)

---

#### Adobe Sign (USA)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-10% |
| **Preise** | Acrobat Pro mit Sign: 23.79 EUR/Mo, Enterprise: Anfrage |
| **DSGVO/DSG** | US-Unternehmen, EU-Rechenzentren |

**Kernfunktionen im Detail:**
- In Adobe Acrobat integriert (= viele Nutzer haben es schon)
- Signatur-Workflows, Templates
- API (weniger Doku als DocuSign/Skribble)
- PDF-Bearbeitung + Signatur in einem Tool

**Bloat fuer KMUs:**
- Gebunden an Adobe-Oekosystem
- Teuer wenn nur fuer Signatur genutzt

---

#### signNow (USA, airSlate)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | <3% in DACH |
| **Preise** | Business 8 USD/User/Mo, Enterprise 15 USD/Mo |
| **DSGVO/DSG** | US-Unternehmen |

**Kernfunktionen im Detail:**
- E-Signatur mit Templates
- Formulare + Felder
- API
- Guenstigster Anbieter

**Bloat fuer KMUs:**
- Wenig DACH-Praesenz
- Keine QES

---

### 8.2 E-Signatur-Zusammenfassung

**Empfehlung fuer KMU Hub:** Skribble-API integrieren. Gruende:
1. Schweizer Unternehmen = perfekt fuer DACH-Datenschutz
2. ZertES-konform (Schweiz) + eIDAS-konform (EU) = rechtlich sicher
3. Gut dokumentierte REST-API
4. EES fuer Alltagsdokumente, QES fuer Vertraege
5. Branding: "Swiss Made" passt zu KMU Hub's EU-Datensouveraenitaets-Narrativ

KMU Hub Status: Rapporte-Modul hat "Digitale Unterschrift (Placeholder)" — hier Skribble anbinden.

---

## 9. Vermietung/Reservierung

### 9.1 Tool-Uebersicht

#### Rentman (Niederlande)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~15-20% bei Eventtech/Veranstaltungstechnik |
| **Preise** | Essentials 35 EUR/User/Mo, Standard 59 EUR/Mo, Pro 79 EUR/Mo |
| **DSGVO/DSG** | EU-Unternehmen (NL), GDPR-konform |

**Kernfunktionen im Detail:**
- Equipment-Verwaltung: Artikel mit Seriennummern, Zustand, Standort
- Verfuegbarkeitskalender: Visuell (Timeline), Konflikterkennung
- Angebote + Rechnungen: Aus Reservierungen generiert
- Crew-Planung: Mitarbeiter zu Projekten zuweisen
- Transport-Planung: Fahrzeuge + Routen
- QR-Code-Scanning fuer Check-in/Check-out
- Packlisten-Generierung
- Kontaktverwaltung (Kunden + Lieferanten)

**Bloat fuer KMUs:**
- Sehr Event-spezifisch
- 35+ EUR/User ist teuer fuer KMUs
- Overkill fuer einfache Werkzeugvermietung

---

#### Booqable (Niederlande)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~5-10% bei Verleih-Unternehmen |
| **Preise** | Essential 29 EUR/Mo, Growth 79 EUR/Mo, Premium 199 EUR/Mo (jeweils Basis, nicht pro User) |
| **DSGVO/DSG** | EU-Unternehmen (NL), GDPR-konform |

**Kernfunktionen im Detail:**
- Produkt-Katalog: Artikel mit Varianten, Bundles
- Online-Buchung: Embeddable Booking Widget fuer Website
- Verfuegbarkeit: Kalender + Bestandsberechnung
- Bestellungen: Erstellen, Auschecken, Einchecken
- Kunden-Verwaltung
- Preise: Stunden/Tage/Wochen-Saetze + Rabatte
- Zahlungen: Stripe/PayPal Integration
- Standort-Management

**Bloat fuer KMUs:**
- Online-Booking-Fokus — nicht jeder KMU braucht das
- Kein PM, kein CRM darueber hinaus

---

#### EasyVerein (Deutschland)

| Eigenschaft | Detail |
|-------------|--------|
| **Marktanteil DACH-KMU** | ~10-15% bei Vereinen, <1% bei KMUs |
| **Preise** | Ab 3.90 EUR/Mo (kleine Vereine), Pro 9.90 EUR/Mo |
| **DSGVO/DSG** | Deutsches Unternehmen |

**Kernfunktionen im Detail:**
- Mitgliederverwaltung
- Beitragsverwaltung + SEPA-Lastschrift
- Veranstaltungsmanagement
- Dokumentenablage
- Kommunikation (Newsletter, Serienbriefe)

**Bloat fuer KMUs:**
- Ist fuer Vereine, nicht fuer Unternehmen

---

### 9.2 Vermietungs-Zusammenfassung

| Feature | Rentman | Booqable | KMU Hub Status |
|---------|---------|----------|----------------|
| Objekt-Verwaltung | Ja | Ja | **Gebaut** (4 Typen: Geraet, Raum, Fahrzeug, Werkzeug) |
| Verfuegbarkeitskalender | Ja | Ja | **Gebaut** (Wochenraster) |
| Reservierungen CRUD | Ja | Ja | **Gebaut** |
| Konflikterkennung | Ja | Ja | **Gebaut** |
| Preisberechnung | Ja | Ja | **Gebaut** (Stunden/Tage/Wochen) |
| Online-Buchungs-Widget | Nein | Ja | **Fehlt** (aber auch nicht Prio) |
| Angebote→Rechnungen | Ja | Ja | **Fehlt** (Bruecke zu Buchhaltung) |
| QR-Code Check-in/out | Ja | Nein | **Fehlt** |

---

## 10. Funktions-Tiefenanalyse pro Kategorie

### 10.1 Projektmanagement

| Funktion | Wird gebraucht? | Warum? | KMU Hub Status |
|----------|----------------|--------|----------------|
| Kanban-Board | JA (95%) | Universell, jede Branche, jede Teamgroesse | **Gebaut** |
| Gantt-Chart | JA (60%) | Bauplanung, Projektplanung, Meilensteine | **Gebaut** (UI), Phase 6 Backend |
| Listenansicht (Tabelle) | JA (80%) | Schnelle Uebersicht, Sortieren/Filtern | **Gebaut** |
| Kalenderansicht | JA (70%) | Deadlines visualisieren | **Gebaut** (separates Kalender-Modul) |
| Abhaengigkeiten | JA (40%) | Nur bei komplexen Projekten (Bau, IT) | **Gebaut** (UI), Phase 6 Backend |
| Subtasks/Checklisten | JA (90%) | Jeder braucht verschachtelte Aufgaben | **Gebaut** |
| Custom Fields | JA (50%) | Branchenspezifische Daten an Tasks | **Geplant** (C9) |
| Automationen/Regeln | NEIN (20%) | KMUs unter 20 MA machen das manuell | **Nicht geplant** (richtig so) |
| Sprint Planning | NEIN (5%) | Nur Scrum-Teams (IT), nicht Handwerker | **Nicht geplant** (richtig so) |
| Portfolio-View | NEIN (10%) | Nur bei 10+ parallelen Projekten | **Nicht geplant** (richtig so) |
| OKRs/Goals | NEIN (5%) | Enterprise-Feature, KMUs nutzen das nicht | **Nicht geplant** (richtig so) |
| Gaeste-Zugang (Kunden) | JA (40%) | Agenturen zeigen Kunden den Status | **Nicht geplant** (sollte kommen!) |
| Vorlagen | JA (70%) | Wiederkehrende Projekte schnell anlegen | **Gebaut** |
| Zeiterfassung pro Task | JA (60%) | Fuer Abrechnung/Budgetierung | **Gebaut** (UI), Phase 6 Backend |
| Datei-Anhaenge pro Task | JA (80%) | Specs, Bilder, PDFs an Tasks haengen | **Gebaut** (UI) |

### 10.2 Zeiterfassung

| Funktion | Wird gebraucht? | Warum? | KMU Hub Status |
|----------|----------------|--------|----------------|
| Timer (Start/Stop) | JA (90%) | Kernfunktion | **Gebaut** |
| Manuelle Eintraege | JA (95%) | Nachtrag, Offline-Zeiten | **Gebaut** |
| Soll/Ist-Vergleich | JA (80%) | Arbeitszeitkonto, Ueberstunden | **Gebaut** |
| DATEV-Export | JA (70%) | Lohnabrechnung fuer Steuerberater | **Fehlt** — KRITISCH fuer DE |
| Reisekostenabrechnung | NEIN (20%) | Nur Aussendienstler | Nicht geplant |
| Automatische App-Erkennung | NEIN (5%) | Datenschutzproblematisch in DACH | Nicht geplant (richtig) |
| GPS-Stempeluhr | JA (30%) | Bauarbeiter, Aussendienstler | Erst mit Mobile App |
| Rechnungen aus Stunden | JA (40%) | Dienstleister rechnen nach Stunden ab | **Fehlt** — Bridge zu Buchhaltung |
| Genehmigungsworkflow | JA (50%) | "Ueberstunden genehmigen" | **Gebaut** |
| Stundenzettel PDF | JA (60%) | Fuer Kunden/Steuerberater | **Fehlt** |

### 10.3 Schichtplanung

| Funktion | Wird gebraucht? | Warum? | KMU Hub Status |
|----------|----------------|--------|----------------|
| Wochenraster (MA x Tage) | JA (100%) | Kernfunktion jeder Schichtplanung | **Gebaut** |
| Schichtvorlagen | JA (80%) | Wiederkehrende Muster | **Gebaut** |
| Tauschboerse | JA (60%) | Flexibilitaet, Mitarbeiterzufriedenheit | **Gebaut** |
| Zuschlaege (Nacht/WE) | JA (70%) | Gesetzliche Pflicht in vielen Faellen | **Fehlt** |
| Qualifikationsfilter | JA (50%) | Gastro: Koch darf nicht an Bar | **Fehlt** |
| Stempeluhr | JA (40%) | Zeiterfassung = Schichtbeginn/-ende | Nur Timer |
| Open Shifts | JA (30%) | MA melden sich fuer offene Schichten | **Fehlt** |
| Arbeitszeitgesetz-Regeln | JA (80%) | Max 10h/Tag, 11h Ruhezeit, etc. | **Fehlt** — KRITISCH |
| DATEV-Lohnexport | JA (60%) | Stunden→Lohn→Steuerberater | **Fehlt** |

### 10.4 Kommunikation

| Funktion | Wird gebraucht? | Warum? | KMU Hub Status |
|----------|----------------|--------|----------------|
| 1:1 Chat | JA (95%) | Kern | **Gebaut** |
| Gruppen/Channels | JA (90%) | Pro Projekt/Team | **Gebaut** |
| Threads | JA (60%) | Verhindert Chaos | **Geplant** |
| Video-Calls | JA (80%) | Remote/Hybrid | **Geplant** (LiveKit) |
| Screensharing | JA (70%) | Support, Demos | **Geplant** (LiveKit) |
| Reactions | JA (80%) | Quick-Feedback | **Geplant** |
| Datei-Sharing | JA (90%) | Bilder, PDFs | **Geplant** |
| @Mentions | JA (80%) | Aufmerksamkeit lenken | **Geplant** |
| Meeting-Aufzeichnung | JA (30%) | Nur bei wichtigen Meetings | **Geplant** |
| Transkription | NEIN (10%) | Nice-to-have, nicht must-have | Nicht geplant |
| Telefonie (PSTN) | NEIN (15%) | KMUs nutzen Mobiltelefone | Nicht geplant (richtig) |

---

## 11. Branchenspezifische Analyse

### 11.1 Bau

**Was braucht ein Bauunternehmen (10-50 MA)?**

| Kernbedarf | Typische Tools | KMU Hub Abdeckung |
|------------|---------------|-------------------|
| Rapporte/Bautagebuch | PlanRadar, 123erfasst | **Rapporte-Modul: 90%** |
| Aufmass | 123erfasst, BauProfi | **Rapporte-Modul: 70%** (Flaeche/Volumen ja, NPK nein) |
| Zeiterfassung Baustelle | 123erfasst, Papershift | **Zeiterfassung: 60%** (kein GPS-Stempeln) |
| Projektplanung (Gantt) | MS Project, Monday | **Work-Modul: 80%** |
| Angebote/Rechnungen | Bexio, Sorba | **Buchhaltung: 50%** (keine Belegkette) |
| Fotodokumentation | PlanRadar | **Rapporte: 80%** (Photo-Upload) |
| Maengelmanagement | PlanRadar | **Fehlt komplett** |
| Materialverwaltung | ERP | **Inventar-Modul: 70%** |
| Fuhrpark | Vimcar | **Fuhrpark-Modul: 60%** (kein Fahrtenbuch) |
| Schichtplanung | Papershift | **Schichten-Modul: 60%** (keine Zuschlaege) |

**Fazit Bau:** KMU Hub deckt ~70% ab. Fehlende Elemente: Maengelmanagement auf Grundriss, GPS-Zeiterfassung, finanzamtkonformes Fahrtenbuch, Zuschlagsberechnung.

---

### 11.2 Handwerk

**Was braucht ein Handwerksbetrieb (5-20 MA)?**

| Kernbedarf | Typische Tools | KMU Hub Abdeckung |
|------------|---------------|-------------------|
| Auftragsannahme + Planung | Craftnote | **Work-Modul: 70%** |
| Zeiterfassung (einfach) | Clockodo | **Zeiterfassung: 90%** |
| Angebote/Rechnungen | Bexio, lexoffice | **Buchhaltung: 50%** |
| Materialeinkauf | Handwerker-ERP | **Einkauf-Modul: 70%** |
| Kundenverwaltung | CRM | **CRM-Modul: 90%** |
| Fotodokumentation | Craftnote | **Rapporte: 80%** |
| Fahrtenbuch | Vimcar | **Fehlt** |
| Terminplanung (Kunden) | Google Calendar | **Kalender: 80%** (Terminbuchung-Tab!) |
| Team-Chat | WhatsApp (!) | **Chat: 90%** |

**Fazit Handwerk:** KMU Hub deckt ~80% ab. Hauptproblem: Handwerker wollen ein einfaches "Angebot → Auftrag → Rechnung"-Flow (Belegkette). Das fehlt.

---

### 11.3 Gastro

**Was braucht ein Restaurant/Hotel (10-50 MA)?**

| Kernbedarf | Typische Tools | KMU Hub Abdeckung |
|------------|---------------|-------------------|
| Schichtplanung | gastromatic, Papershift | **Schichten: 50%** (keine Zuschlaege, kein MiLoG) |
| Zeiterfassung | Papershift | **Zeiterfassung: 60%** (keine Stempeluhr) |
| Reservierungen | OpenTable, resmio | **Fehlt** (Vermietung-Modul ist fuer Equipment) |
| Kassensystem | orderbird, Lightspeed | **Bewusst entfernt** (Kasse-Modul removed) |
| Personalplanung | gastromatic | **Schichten: 50%** |
| Einkauf/Lieferanten | gastromatic, Choco | **Einkauf: 60%** |
| Lohnabrechnung | DATEV | **Lohn: 40%** (CH-spezifisch, DE fehlt) |
| Trinkgeldverwaltung | gastromatic | **Fehlt** |
| Gaeste-Kommunikation | WhatsApp | **Chat: 70%** (intern gut, extern fehlt) |
| HACCP-Checklisten | Kitchenscience | **Formulare: 50%** (koennte genutzt werden) |

**Fazit Gastro:** KMU Hub deckt nur ~55% ab. Gastro braucht ein Tisch-Reservierungssystem, Kasse (bewusst entfernt), und gastro-spezifische Schichtplanung (Zuschlaege, MiLoG). Gastro ist die schwaechste Branche fuer KMU Hub.

---

### 11.4 Handel (Einzel-/Grosshandel)

**Was braucht ein Haendler (5-50 MA)?**

| Kernbedarf | Typische Tools | KMU Hub Abdeckung |
|------------|---------------|-------------------|
| Warenwirtschaft | weclapp, JTL | **Inventar: 70%** |
| Bestellwesen | ERP | **Einkauf: 80%** |
| Lagerverwaltung | ERP | **Inventar: 60%** (kein Barcode-Scanning) |
| Rechnungen | Bexio, lexoffice | **Buchhaltung: 60%** |
| Kundenverwaltung | CRM | **CRM: 90%** |
| E-Commerce-Anbindung | Shopify, WooCommerce | **Fehlt** |
| Versandabwicklung | DHL/DPD-Integration | **Fehlt** |
| Kassensystem | Lightspeed | **Bewusst entfernt** |
| Lieferantenbewertung | ERP | **Einkauf: 70%** (Rating vorhanden) |
| Preislisten-Management | ERP | **Fehlt** |

**Fazit Handel:** KMU Hub deckt ~65% ab. Fehlende Elemente: E-Commerce-Anbindung, Versandabwicklung, Barcode-Scanning, Preislisten. Handel ist aber auch nicht Kernzielgruppe.

---

### 11.5 Dienstleister (Agentur/Beratung)

**Was braucht eine Agentur/Beratung (5-30 MA)?**

| Kernbedarf | Typische Tools | KMU Hub Abdeckung |
|------------|---------------|-------------------|
| Projektmanagement | Monday, Asana, ClickUp | **Work-Modul: 90%** |
| Zeiterfassung (billable!) | Clockodo, Toggl, Harvest | **Zeiterfassung: 80%** |
| Kundenverwaltung | Pipedrive, HubSpot | **CRM: 90%** |
| Deal-Pipeline | Pipedrive | **CRM: 90%** (Deal-Pipeline gebaut) |
| Rechnungen aus Stunden | Harvest, Clockodo | **Fehlt** (Bruecke Zeiterfassung→Rechnung) |
| Teamkommunikation | Slack, Teams | **Chat: 90%** |
| Video-Calls | Zoom, Teams | **Meetings: 80%** (LiveKit) |
| Dokumenten-Sharing | Google Drive, Dropbox | **Dokumente: 80%** |
| Angebotserstellung | Propose, Bexio | **Buchhaltung: 50%** |
| Reporting (Auslastung) | Toggl, ZEP | **Berichte: 60%** |
| Gaeste-Zugang fuer Kunden | Monday, Asana | **Fehlt** |

**Fazit Dienstleister:** KMU Hub deckt ~85% ab. Staerkste Branche! Fehlende Elemente: "Stunden → Rechnung" Workflow, Gaeste-Zugang fuer Kunden, bessere Auslastungsberichte.

---

## 12. Build vs. Integrate

### 12.1 Projektmanagement (Work-Modul)

**Status:** KMU Hub hat ein solides Work-Modul (Projekte, Tasks, Kanban, Gantt, Abhaengigkeiten, Sub-Tasks, Vorlagen, Zeiterfassung).

**Was fehlt gegenueber Monday/Asana/ClickUp:**

| Feature | Prioritaet | Build oder Skip? |
|---------|-----------|------------------|
| Custom Fields | HOCH | **BUILD** — Branchenspezifische Felder sind das Herzpunkt der Anpassbarkeit |
| Automationen/Regeln | NIEDRIG | **SKIP fuer v1** — KMUs unter 20 MA machen das manuell |
| Gaeste-Zugang (extern) | MITTEL | **BUILD** — Agenturen brauchen das dringend, Differentiator gegenueber internen Tools |
| Workload-View (Kapazitaet) | MITTEL | **DEFER (v2)** — Nur relevant ab 15+ MA im Projektgeschaeft |
| Portfolio/Multi-Projekt-View | NIEDRIG | **SKIP** — Dashboard-Cards reichen |
| Formulare (Intake) | GEBAUT | **Formulare-Modul existiert** |

### 12.2 Zeiterfassung

**Status:** 6 Sub-Views gebaut: Overview, Timer, Entries, Team, Reports, Absences.

**Was fehlt:**

| Feature | Prioritaet | Build oder Skip? |
|---------|-----------|------------------|
| DATEV-Export | HOCH | **BUILD** — Ohne das ist die Zeiterfassung in DE nur halb brauchbar |
| Stundenzettel als PDF | MITTEL | **BUILD** — Einfach, hoher Nutzwert |
| Stunden→Rechnung Workflow | MITTEL | **BUILD** — Bridge zu Buchhaltung: "Diese 40h fuer Kunde X → Rechnung generieren" |
| GPS-Stempeluhr | NIEDRIG | **DEFER** — Erst mit React Native Mobile App |
| Reisekosten | NIEDRIG | **SKIP** — Zu nischig, gibt Spezialtools |

### 12.3 Chat/Video (LiveKit)

**Status:** Chat gebaut (Luke), Video/Audio geplant (Phase 8, LiveKit).

**Was fehlt gegenueber Teams/Slack/Zoom:**

| Feature | Prioritaet | Build oder Skip? |
|---------|-----------|------------------|
| Threads | HOCH | **BUILD** — Ohne Threads wird Chat ab 10 MA chaotisch |
| Meeting-Aufzeichnung | MITTEL | **BUILD** — LiveKit kann das, Backend-Aufwand ueberschaubar |
| Breakout Rooms | NIEDRIG | **SKIP** — Unter 50 MA selten gebraucht |
| Teams/Slack Bridge | HOCH | **BUILD** — USP! Geplant in Phase 15 |
| Transkription | NIEDRIG | **DEFER (v2)** — Nice-to-have, nicht must-have |
| Telefonie (PSTN) | NIEDRIG | **SKIP** — Zu komplex, KMUs haben Handys |

### 12.4 ERP-Funktionen

**Status:** Buchhaltung (einfach), Inventar, Einkauf, Produktion als Module.

**Strategie:** KMU Hub ist KEIN ERP. Die Strategie ist richtig: Einfache eingebaute Buchhaltung + Integrationen fuer Spezialisten (Bexio, Abacus, Run my Accounts).

| Feature | Prioritaet | Build oder Integrate? |
|---------|-----------|----------------------|
| Belegkette (Angebot→Rechnung) | HOCH | **BUILD** — KMUs ohne Bexio brauchen das |
| FiBu/Hauptbuch | NIEDRIG | **INTEGRATE** (Bexio/Abacus) — Zu komplex zum Selberbauen |
| DATEV-Export | HOCH | **BUILD** — Schnittstelle zum Steuerberater |
| MwSt-Berechnung (DE/AT/CH) | HOCH | **BUILD** — Muss in Rechnungsmodul |
| QR-Rechnung (CH) | HOCH fuer CH | **BUILD** — Swiss QR-Code auf Rechnung ist Pflicht seit 2022 |
| Lohnabrechnung | MITTEL | **INTEGRATE** — CH-Sozialversicherungsmath ist zu komplex fuer v1 |

### 12.5 E-Signatur

**Empfehlung: INTEGRATE mit Skribble API.**

| Aspekt | Detail |
|--------|--------|
| Eigenentwicklung? | NEIN — E-Signatur braucht Zertifikate, Trust Service Provider, rechtliche Compliance. Unverhaeltnismaessiger Aufwand. |
| Skribble API? | JA — REST API, Schweizer Anbieter, passt zu KMU Hub Narrativ |
| Wo anbinden? | Vertraege-Modul ("Vertrag zur Unterschrift senden"), Rapporte-Modul ("Rapport digital unterschreiben"), Formulare-Modul ("Formular signieren lassen") |
| Aufwand | MITTEL — API-Anbindung, Signatur-Status-Tracking, Webhook fuer Completion |
| Alternativ | DocuSign als Fallback (groesserer Marktanteil, aber US-Bedenken) |

---

## 13. TOP 10 Software-Kategorien die KMUs vereinen wollen

Basierend auf der Analyse: Welche Tool-Fragmentierung nervt KMUs am meisten?

### Ranking: "Ich will nicht 10 Tools bezahlen"

| Rang | Zusammenlegung | Typischer Status Quo | Nervfaktor | KMU Hub Chance |
|------|---------------|---------------------|------------|----------------|
| **1** | **CRM + Projektmanagement** | Pipedrive + Monday.com (oder Excel + Outlook) | SEHR HOCH | **VOLL ABGEDECKT** — DAS ist der Kern-USP! |
| **2** | **Chat + Video + Kalender** | Teams + Zoom + Google Calendar | SEHR HOCH | **VOLL ABGEDECKT** — Chat + LiveKit + Kalender-Modul |
| **3** | **Zeiterfassung + Projektmanagement** | Clockodo + Asana | HOCH | **ABGEDECKT** — Timer in Work-Modul, eigenes Zeiterfassungs-Modul |
| **4** | **Rechnungen + Zeiterfassung** | Bexio/lexoffice + Clockodo | HOCH | **TEILWEISE** — Buchhaltung gebaut, aber "Stunden→Rechnung" fehlt |
| **5** | **Dateien + Chat + E-Mail** | Dropbox + Slack + Gmail | HOCH | **ABGEDECKT** — Dokumente + Chat + E-Mail Modul |
| **6** | **Schichtplanung + Zeiterfassung + Lohn** | Papershift + Excel + DATEV | HOCH | **TEILWEISE** — Schichten + Zeit gebaut, DATEV-Export fehlt |
| **7** | **CRM + E-Mail + Kalender** | HubSpot Free + Gmail + Google Calendar | MITTEL | **ABGEDECKT** — CRM + Mail + Kalender |
| **8** | **Fuhrpark + Zeiterfassung** | Vimcar + Clockodo | MITTEL | **ABGEDECKT** — Fuhrpark + Zeiterfassung Module |
| **9** | **Helpdesk + CRM + Chat** | Zendesk + Pipedrive + Slack | MITTEL | **ABGEDECKT** — Helpdesk + CRM + Chat |
| **10** | **Inventar + Einkauf + Rechnungen** | weclapp (oder 3 Excel-Tabellen) | MITTEL | **TEILWEISE** — Module gebaut, Belegkette fehlt |

### Fazit: KMU Hub's groesster Wettbewerbsvorteil

**Die Top 3 "Ich will nicht X Tools bezahlen"-Kombinationen sind ALLE abgedeckt:**
1. CRM + PM = Check
2. Chat + Video + Kalender = Check
3. Zeit + PM = Check

**Wo KMU Hub noch nachziehen muss:**
- **Rang 4: "Stunden → Rechnung" Workflow** — Die Bridge zwischen Zeiterfassung und Buchhaltung fehlt
- **Rang 6: DATEV-Export** — Ohne das ist die Zeiterfassung in Deutschland nur halb brauchbar
- **Rang 10: Belegkette** — Angebot → Auftrag → Lieferschein → Rechnung fehlt fuer KMUs ohne Bexio

---

## 14. Gesamt-Empfehlungen fuer KMU Hub

### Was SOFORT gebaut werden sollte (hoechster ROI)

| Feature | Begruendung | Geschaetzter Aufwand |
|---------|-------------|---------------------|
| DATEV-Export (Zeiterfassung + Buchhaltung) | Ohne das kauft kein deutsches KMU die Zeiterfassung | Mittel |
| Belegkette (Angebot→Rechnung) | Jeder Handwerker/Dienstleister braucht das | Mittel |
| QR-Rechnung (CH) | Pflicht in der Schweiz, ohne das kein Schweizer Kunde | Klein |
| Gaeste-Zugang (Projekte/Tasks) | Agenturen brauchen das, kein Konkurrent im All-in-One-Segment hat das gut | Mittel |
| Stunden→Rechnung Workflow | Verbindet die beiden Module die KMUs am meisten vereinen wollen | Mittel |

### Was NICHT gebaut werden sollte

| Feature | Begruendung |
|---------|-------------|
| Eigene E-Signatur | Skribble-Integration statt Eigenentwicklung |
| Volles FiBu/Hauptbuch | Zu komplex, Bexio/Abacus Integration reicht |
| Kassensystem | Richtige Entscheidung es zu entfernen |
| Sprint Planning/Scrum | Irrelevant fuer 95% der KMU-Zielgruppe |
| E-Commerce-Anbindung | Nische, dafuer gibt es weclapp/JTL |
| PSTN-Telefonie | Zu komplex, KMUs haben Handys |

### KMU Hub Positionierung im Wettbewerb

```
               Spezialisiert                    All-in-One
                    |                               |
Enterprise   SAP Business One                  Salesforce + Teams + Jira
                    |                               |
Mittelstand  Odoo    -------- KMU Hub ---------- Monday + Slack + Zoom
                    |         (hier!)               |
Klein-KMU    Clockodo + Craftnote              ClickUp (versucht's)
                    |                               |
               Einfach                          Komplex
```

**KMU Hub's Sweet Spot:** All-in-One fuer 5-200 MA, einfacher als ClickUp, mehr Features als Craftnote, europaeischer als Monday, und — dank Branchenprofilen — nie so ueberladen wie es scheint.

---

## 15. Quellen und Confidence

| Bereich | Confidence | Anmerkung |
|---------|-----------|-----------|
| Tool-Features | MEDIUM | Basiert auf Training-Data (Stand Mai 2025), Preise koennen sich geaendert haben |
| Preise | LOW-MEDIUM | Preise aendern sich regelmaessig, bitte vor Go-to-Market aktualisieren |
| Marktanteile | LOW | Keine exakten Zahlen verfuegbar, Schaetzungen basierend auf Branchenberichten |
| DSGVO/DSG Status | MEDIUM | Grundsaetzliche Aussagen stimmen, Details koennen sich geaendert haben |
| Branchenanalyse | MEDIUM-HIGH | Basiert auf realer KMU-Praxis im DACH-Raum |
| Build vs. Integrate | HIGH | Strategische Empfehlungen sind robust und nicht preisabhaengig |
| Top 10 Vereinigung | MEDIUM-HIGH | Basiert auf typischen KMU-Werkzeuglandschaften |

**WICHTIG:** WebSearch und WebFetch waren blocked. Alle Preise und Features basieren auf Training-Data (Stand Mai 2025). Vor Verwendung in Pitch-Decks oder Go-to-Market-Materialien muessen die Preise mit den aktuellen Websites der Anbieter abgeglichen werden.
