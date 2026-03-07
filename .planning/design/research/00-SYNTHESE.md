# Synthese: DACH-KMU-Marktrecherche fuer KMU Hub

**Datum:** 2026-02-16
**Quellen:** 5 Recherche-Dokumente (Office/E-Mail/Storage, CRM/Sales/Helpdesk, Finanzen/Buchhaltung/HR, Projektmanagement/ERP/Branche, DSGVO/DSG-Compliance)
**Gesamtumfang:** ~6.000 Zeilen Primaerrecherche
**Confidence:** MEDIUM (Trainingsdaten-basiert, keine Live-Verifizierung)

---

## 1. BUILD vs. INTEGRATE Matrix

### Gesamtuebersicht ueber alle Kategorien

| Kategorie | Entscheidung | Begruendung | Aufwand | Prioritaet |
|-----------|-------------|-------------|---------|-----------|
| **CRM (Kontakte, Deals, Pipeline)** | BUILD | Kernkompetenz, bereits gebaut (Luke Phase 3). Weiter ausbauen. | Bereits investiert | KERN |
| **Projektmanagement (Kanban, Gantt)** | BUILD | Kernkompetenz, Work-Modul bereits gebaut (Luke Phase 6). | Bereits investiert | KERN |
| **Chat** | BUILD | Kernkompetenz, bereits gebaut (Luke). LiveKit fuer Video geplant. | Bereits investiert | KERN |
| **Video/Audio-Konferenz** | BUILD (LiveKit) | LiveKit ist self-hostable, EU-kompatibel, WebRTC-basiert. | MITTEL | HOCH |
| **Kalender** | BUILD | Bereits gebaut (Luke Phase 7). CalDAV-Client spaeter. | Bereits investiert | KERN |
| **Zeiterfassung** | BUILD | Bereits gebaut. Timer, Eintraege, Reports, Abwesenheiten. | Bereits investiert | KERN |
| **Helpdesk/Ticketing** | BUILD | Bereits gebaut. Benchmark: Zammad. Weiter ausbauen. | Bereits investiert | KERN |
| **Rechnungsstellung** | BUILD | Bereits vorhanden (finance.ts). QR-Rechnung + ZUGFeRD ergaenzen. | MITTEL | HOCH |
| **Angebotstool** | BUILD | Differentiator! Kein CRM-Konkurrent hat das gut integriert. Angebot-zu-Rechnung Flow. | MITTEL | HOCH |
| **Wiki/Wissensmanagement** | BUILD | Vorhanden, aber TipTap-Editor fehlt. BookStack als Vorbild, nicht Notion. | MITTEL | HOCH |
| **Schichtplanung** | BUILD | Bereits gebaut. Zuschlaege + Arbeitszeitgesetz-Regeln ergaenzen. | Bereits investiert | KERN |
| **HR-Basis (Stammdaten, Urlaub)** | BUILD | Bereits gebaut. Einfache CRUD + Workflows. | Bereits investiert | KERN |
| **Fuhrpark** | BUILD | Bereits gebaut. Manuelles Fahrtenbuch ergaenzen. | KLEIN | MITTEL |
| **Inventar/Einkauf** | BUILD | Bereits gebaut. Belegkette fehlt. | KLEIN | MITTEL |
| **Rapporte/Bautagebuch** | BUILD | Bereits gebaut. Starker Differentiator fuer Bau-KMUs. | Bereits investiert | KERN |
| **Vermietung** | BUILD | Bereits gebaut. Reservierungen + Konflikterkennung. | Bereits investiert | KERN |
| **Formulare** | BUILD | Bereits gebaut. Intake-Formulare, Checklisten. | Bereits investiert | KERN |
| **Vertraege** | BUILD | Bereits gebaut. Skribble-Integration spaeter. | Bereits investiert | KERN |
| --- | --- | --- | --- | --- |
| **E-Mail (Client)** | BUILD (IMAP/SMTP) | IMAP-Client bauen, KEINEN eigenen Mailserver. UI bereits vorhanden. Go-Libraries: `emersion/go-imap`. | HOCH (6-8 Wo.) | KRITISCH |
| **Datei-Storage** | HYBRID | Eigener Basis-Storage (S3/MinIO) + optionale Nextcloud-Integration per WebDAV. | HOCH | HOCH |
| **Office-Editor (Word/Excel)** | INTEGRATE (OnlyOffice) | Selbst bauen ist unrealistisch. OnlyOffice Document Server per WOPI-Protokoll einbetten. | MITTEL (2-4 Wo.) | HOCH |
| **Wiki-Editor** | BUILD (TipTap) | TipTap/ProseMirror als Rich-Text-Editor. Open Source, React-kompatibel, erweiterbar. | MITTEL | HOCH |
| --- | --- | --- | --- | --- |
| **Buchhaltung (Doppelte FiBu)** | INTEGRATE | Zu komplex, GoBD-Testat noetig (~15-50k EUR), jaehrliche Aenderungen. DATEV-Export + Bexio-API stattdessen. | N/A | PFLICHT-INTEGRATION |
| **Lohnabrechnung** | INTEGRATE | NIEMALS selbst bauen. DE: 6 SV-Zweige x 6 Steuerklassen x 16 Bundeslaender. CH: 26 Kantone x Quellensteuer. | N/A | PFLICHT-INTEGRATION |
| **Newsletter/Marketing-E-Mail** | INTEGRATE | Deliverability ist ein eigenes Fachgebiet. Brevo (EU) als primaere API, CleverReach als DACH-Option. | MITTEL | MITTEL |
| **E-Signatur** | INTEGRATE (Skribble) | Trust Service Provider noetig. Skribble = Schweizer Firma, REST API, ZertES+eIDAS-konform. | MITTEL | MITTEL |
| **Banking/Bankabgleich** | INTEGRATE (FinAPI) | Aggregator fuer 4.000+ Banken. Eigene EBICS-Implementierung waere irrsinnig aufwaendig. | MITTEL | MITTEL |
| **Handelsregister** | INTEGRATE (Zefix) | CH: Zefix-API kostenlos. DE: Kommerzieller Anbieter spaeter. | KLEIN | NIEDRIG |
| --- | --- | --- | --- | --- |
| **Eigener Mailserver** | NICHT BAUEN | Deliverability, Spam-Reputation, ISP-Beziehungen — eigene Disziplin. | N/A | NIEMALS |
| **Volles ERP** | NICHT BAUEN | SAP/Odoo-Territorium. KMU Hub ist ein Hub, kein ERP. | N/A | NIEMALS |
| **Kassensystem (POS)** | NICHT BAUEN | Richtig entfernt. Hardware-abhaengig, Branchennische. | N/A | NIEMALS |
| **PSTN-Telefonie** | NICHT BAUEN | Zu komplex, KMUs haben Mobiltelefone. | N/A | NIEMALS |
| **Recruiting/ATS** | NICHT BAUEN | Personio, Greenhouse etc. sind spezialisiert. | N/A | NIEMALS |
| **Sprint Planning/Scrum** | NICHT BAUEN | Irrelevant fuer 95% der KMU-Zielgruppe. | N/A | NIEMALS |

---

## 2. KMU Hub Bestandsaufnahme

### Was wir SCHON haben (aus Code gelesen)

**Kern-Module (funktional, UI gebaut):**
- CRM: Kontakte, Firmen (als String), Deals, Pipeline-Kanban, Aktivitaeten, Tags, Gruppen
- Projektmanagement: Kanban, Gantt (UI), Listen, Subtasks, Vorlagen, Abhaengigkeiten (UI)
- Chat: 1:1, Gruppen/Channels (Luke)
- Kalender: Terminplanung, Terminbuchung-Tab mit 8 Services
- Zeiterfassung: Timer, manuelle Eintraege, Soll/Ist, Reports, Abwesenheiten
- Helpdesk: Tickets, SLA-Berechnung, Knowledge Base
- Buchhaltung: Rechnungen, Angebote (type: 'quote'), Mahnwesen, Zahlungen, Spesen
- Team/HR: Mitarbeiterverwaltung, Urlaub/Krankheit-Antraege, Payroll (Anzeige), Schulungen
- Dokumente: Upload/Download, Ordner, Berechtigungen, Versionierung, Vault
- Wiki: Artikel, Kategorien (hierarchisch), CRUD
- E-Mail: 3-Panel-Layout (UI), ComposeModal, Ordner, Flags, Archivierung (kein Backend!)
- Schichtplanung: Wochenraster (Mo-So x MA), farbige Bloecke, Tauschboerse
- Fuhrpark: Fahrzeuge, Wartung, Tankbuch, GPS (UI), Kosten
- Inventar: Artikel, Bestaende, Kategorien
- Einkauf: Bestellungen, Lieferanten, Rating
- Produktion: Stuecklisten, Fertigungsauftraege
- Berichte: Dashboard-Widgets, Statistiken
- Vertraege: CRUD, Status, Zuordnung
- Formulare: Builder, Checklisten
- Vermietung: Objekte (4 Typen), Reservierungen, Konflikterkennung, Preisberechnung
- Rapporte: Tagesberichte, Aufmass, Fotodokumentation

**Branchensystem (gebaut):**
- 10 Business-Profile mit Sidebar/Dashboard-Filterung
- 7 Industry-Module (Inventar, Schichten, Einkauf, Helpdesk, Fuhrpark, Produktion, Berichte)
- 4 Erweiterungsmodule (Vertraege, Formulare, Vermietung, Rapporte)
- 5 Extensions (Buchhaltung+Mahnwesen, Team+Lohn+Schulungen, etc.)
- Dev-Toggle "Alle Module anzeigen"

**Desktop-Infrastruktur:**
- Electron + React + TypeScript
- Desk-System mit 5-Layer Room Scenes, 6 Themes
- Glass/Crystal-Modus, Dark Mode
- Hash-basiertes Routing, Zustand-Stores

### Was FEHLT fuer eine echte Markt-Alternative

| Luecke | Warum kritisch | Referenz-Wettbewerber |
|--------|---------------|----------------------|
| **E-Mail-Backend** (IMAP/SMTP) | Ohne E-Mail ist KMU Hub nur die Haelfte. 100% der KMUs brauchen Mail. | Outlook, Gmail, Thunderbird |
| **Rich-Text-Editor** (Wiki + Dokumente) | Ohne WYSIWYG-Editor ist das Wiki unbrauchbar fuer Nicht-Techniker. | Notion, Confluence, BookStack |
| **Office-Dokumente inline bearbeiten** | "Dann brauche ich trotzdem Office 365" --> KMU Hub wird irrelevant. | Google Docs, OnlyOffice |
| **Custom Fields** (CRM, Helpdesk, Projekte) | Ohne Custom Fields ist es kein echtes CRM. JEDES KMU hat eigene Felder. | Salesforce, HubSpot, Pipedrive |
| **Firma als eigene Entity** (nicht String) | Firma/Person-Hierarchie ist Standard bei jedem CRM. | HubSpot, Pipedrive, cobra |
| **Belegkette** (Angebot -> Auftrag -> Rechnung) | DER Workflow den jeder Handwerker/Dienstleister braucht. | Bexio, Lexoffice, sevDesk |
| **DATEV-Export** | Ohne DATEV-Schnittstelle kauft kein deutsches KMU die Buchhaltung/Zeiterfassung. | Lexoffice, sevDesk, Clockodo |
| **QR-Rechnung (CH)** | Seit 2022 PFLICHT in der Schweiz. Ohne das kein Schweizer Kunde. | Bexio, Abacus |
| **ZUGFeRD/XRechnung** | Ab 2025 Empfang Pflicht, ab 2027/2028 Versand Pflicht (DE B2B). | Easybill, Lexoffice |
| **Akademischer Titel + Anrede-Logik** | DACH-Grunderwartung. "Herr Prof. Dr. Mueller" muss funktionieren. | cobra, CentralStation |
| **Duplikaterkennung** (CRM) | KMUs importieren Listen, Duplikate entstehen sofort. | HubSpot, Salesforce |
| **Canned Responses** (Helpdesk) | Jedes Helpdesk hat Textbausteine. Standard-Feature. | Zendesk, Zammad, Freshdesk |
| **Private Notizen** (Helpdesk) | Interne Kommentare die der Kunde nicht sieht. Basis-Feature. | Alle Helpdesks |
| **E-Mail-zu-Kontakt-Zuordnung** | Mails automatisch dem CRM-Kontakt zuordnen. DAS CRM-Feature. | HubSpot, Pipedrive |
| **Gaeste-Zugang** (Projekte) | Agenturen muessen Kunden den Projektstatus zeigen koennen. | Monday, Asana |
| **MWSt multi-country** | Nur CH-Saetze vorhanden. DE (19%/7%) und AT (20%/10%/13%) fehlen. | Bexio, Lexoffice |
| **PDF-Generierung** (Rechnungen/Angebote) | Jedes Dokument das rausgeht, muss als PDF gehen. | Alle Buchhaltungstools |
| **Externer Link-Share** (Dateien) | Dateien per Link an Kunden/Partner senden. | Dropbox, OneDrive, Nextcloud |

---

## 3. Priorisierte Feature-Luecken (Top 20)

Sortiert nach Business Impact x Machbarkeit. Aufwand in Entwicklungszeit (Backend + Frontend).

| # | Feature | Business Impact | Aufwand | Abhaengigkeiten | Begruendung |
|---|---------|----------------|---------|-----------------|-------------|
| 1 | **IMAP/SMTP E-Mail-Backend** | SEHR HOCH (100% KMUs brauchen Mail) | 6-8 Wochen | Go-Library `emersion/go-imap` | Ohne Mail-Backend bleibt die halbe App leer. UI ist fertig. |
| 2 | **DATEV-Export** (Zeiterfassung + Rechnungen) | SEHR HOCH (60-75% der DE-KMUs nutzen DATEV ueber Steuerberater) | 1-2 Wochen | Dokumentiertes CSV-Format | Ohne DATEV kauft kein deutsches KMU. Format ist dokumentiert. |
| 3 | **Custom Fields** (CRM, Helpdesk, Projekte) | SEHR HOCH (jedes KMU braucht eigene Felder) | 3-4 Wochen | JSONB oder EAV-Schema in PostgreSQL | Ohne Custom Fields kein echtes CRM. Branchenspezifische Anpassung unmoeglich. |
| 4 | **Belegkette** (Angebot -> Auftrag -> Lieferschein -> Rechnung) | SEHR HOCH (jeder Handwerker/Dienstleister) | 3-4 Wochen | Finance-Modul vorhanden | DER Workflow: 1 Klick von Angebot zu Rechnung. |
| 5 | **QR-Rechnung** (Swiss QR-Code) | SEHR HOCH fuer CH (Pflicht seit 2022) | 1-2 Wochen | Swiss QR-Code Spec (offen) | Ohne QR-Rechnung kein Schweizer Kunde. Spec ist gut dokumentiert. |
| 6 | **Rich-Text-Editor** (Wiki) | HOCH (Wiki ohne Editor = unbrauchbar) | 2-3 Wochen | TipTap/ProseMirror | Wiki wird ohne WYSIWYG nicht genutzt. |
| 7 | **Firma als eigene Entity** | HOCH (Standard bei jedem CRM) | 2-3 Wochen | DB-Migration | Firma/Person-Hierarchie: 1 Firma, N Ansprechpartner. |
| 8 | **MWSt multi-country** (DE/AT/CH) | HOCH (laenderspezifisch Pflicht) | 2-3 Tage | Konfiguration | DE: 19%/7%, CH: 8.1%/2.6%/3.8%, AT: 20%/10%/13%. |
| 9 | **PDF-Generierung** (Rechnungen/Angebote) | HOCH (jedes Dokument = PDF) | 1-2 Wochen | Go-Libraries (wkhtmltopdf, Puppeteer) | Rechnungen/Angebote als PDF versenden/archivieren. |
| 10 | **Akadem. Titel + Anrede-Logik** | HOCH fuer DACH | 2-3 Tage | Kontakt-Modell erweitern | "Herr Prof. Dr. Mueller", Sie/Du-Flag, bevorzugte Sprache. |
| 11 | **Canned Responses + Private Notizen** (Helpdesk) | HOCH (Standard bei jedem Helpdesk) | 3-5 Tage | Helpdesk-Modul vorhanden | Textbausteine + interne Kommentare. Einfach, hoher Nutzen. |
| 12 | **Duplikaterkennung** (CRM) | HOCH (bei jedem Datenimport) | 1-2 Wochen | Matching-Algorithmus (Name + E-Mail) | KMUs importieren CSV-Listen, Duplikate sind unvermeidlich. |
| 13 | **Externer Link-Share** (Dateien) | MITTEL-HOCH | 3-5 Tage | Signierte Download-URLs | "Datei per Link an Kunden senden" = Basis. |
| 14 | **OnlyOffice WOPI-Integration** | HOCH (langfristig) | 2-4 Wochen | Docker + WOPI-Endpoints in Go | .docx/.xlsx direkt in KMU Hub bearbeiten. Ohne das brauchen Kunden weiter Office 365. |
| 15 | **Bexio-API Integration** | HOCH fuer CH | 2-4 Wochen | OAuth2, REST API | DER Schweizer Buchhaltungs-Standard. 80.000+ Nutzer. |
| 16 | **ZUGFeRD/XRechnung** | MITTEL (erst ab 2027 Versand-Pflicht DE) | 2-3 Wochen | Go-Libraries (`invopop/gobl`) | PDF/A-3 mit eingebettetem XML. Wird Pflicht. |
| 17 | **Stunden-zu-Rechnung Workflow** | MITTEL-HOCH (Dienstleister) | 1-2 Wochen | Finance + Zeiterfassung vorhanden | "Diese 40h fuer Kunde X --> Rechnung generieren". |
| 18 | **Gaeste-Zugang** (Projekte) | MITTEL (Agenturen, Berater) | 2-3 Wochen | Auth-System erweitern | Kunden sehen Projektstatus. Kein Konkurrent im All-in-One hat das gut. |
| 19 | **Nextcloud WebDAV-Integration** | MITTEL | 2-3 Wochen | WebDAV-Protokoll | Dateien aus Nextcloud in KMU Hub anzeigen. Starker DSGVO-Stack. |
| 20 | **GoBD-konforme Rechnungen** | MITTEL-HOCH fuer DE | 1-2 Wochen | Unveraenderbare Records, Audit-Log | Lueckenlose Nummern, Storno statt Loeschung, Aenderungsprotokoll. |

---

## 4. Integrations-Strategie

### Pflicht-Integrationen (ohne diese kein Market Fit)

| Integration | Prioritaet | Zielmarkt | API-Typ | Aufwand | Details |
|-------------|-----------|-----------|---------|---------|---------|
| **DATEV-Export** | KRITISCH | DE (60-75% KMUs) | CSV-Format (proprietaer, aber dokumentiert) | 1-2 Wochen | Buchungsstapel-Format, Windows-1252 Encoding, Semikolon-Trennung. KEIN API-Zugang noetig, nur Format-Export. |
| **Bexio REST-API** | KRITISCH | CH (80.000+ Nutzer) | REST, OAuth2 | 2-4 Wochen | Bidirektional: Kontakte, Rechnungen, Zahlungen sync. Gut dokumentiert, stabile API. |
| **IMAP/SMTP** (Protokoll) | KRITISCH | Alle | Standard-Protokoll | 6-8 Wochen | Kein einzelner Anbieter, sondern Protokoll-Support. Deckt Exchange, Gmail, OX, Hostpoint, Infomaniak ab. |
| **Swiss QR-Code** | KRITISCH | CH | Offene Spec | 1-2 Wochen | Swiss QR-Code auf Rechnungen. Seit 2022 Pflicht in CH. |

### Empfohlene Integrationen (starker Differentiator)

| Integration | Prioritaet | Zielmarkt | API-Typ | Aufwand | Details |
|-------------|-----------|-----------|---------|---------|---------|
| **Brevo** (Newsletter/Transactional) | HOCH | DACH | REST API | 2-3 Wochen | EU-Firma (Frankreich). Transactional + Marketing in einer API. Guenstig. |
| **CleverReach** (Newsletter) | HOCH | DE | REST API v3, Webhooks | 2-3 Wochen | Deutsche Firma. DSGVO-exzellent. DACH-KMUs kennen es. |
| **Skribble** (E-Signatur) | MITTEL | DACH | REST API | 2-3 Wochen | Schweizer Firma. EES/FES/QES. ZertES + eIDAS konform. Perfekt fuer Vertraege-Modul. |
| **OnlyOffice Document Server** | HOCH | Alle | WOPI-Protokoll (REST) | 2-4 Wochen | EU-Firma (Lettland). Beste .docx-Kompatibilitaet im Open-Source-Bereich. Docker-Deployment. |
| **Zefix** (CH Handelsregister) | MITTEL | CH | REST API (kostenlos!) | 3-5 Tage | Firmensuche -> UID, Adresse, Rechtsform automatisch ausfuellen. |
| **FinAPI** (Banking-Aggregator) | MITTEL | DE/AT | REST API, PSD2 | 3-4 Wochen | 4.000+ Banken in DACH. Automatischer Bankabgleich. Ab ~500 EUR/Mo. |
| **ZUGFeRD/XRechnung** | MITTEL | DE | XML-Schema (EN 16931) | 2-3 Wochen | Ab 2027 Versand-Pflicht B2B DE. Go-Libraries verfuegbar. |

### Optionale Integrationen (Nice-to-have, v2+)

| Integration | Zielmarkt | Begruendung | Aufwand |
|-------------|-----------|-------------|---------|
| **Nextcloud** (WebDAV/CalDAV/CardDAV) | DSGVO-bewusste KMUs | Dateien, Kalender, Kontakte sync. Starker DSGVO-Stack. | 3-4 Wochen |
| **Collabora Online** (Alternative zu OnlyOffice) | Self-Hosted Kunden | LibreOffice im Browser. Fuer Kunden die OnlyOffice nicht wollen. | 2-3 Wochen |
| **Abacus AbaConnect** | CH (groessere KMUs) | Fuer Kunden mit Abacus-Buchhaltung. AbaConnect-API weniger modern als Bexio. | 3-4 Wochen |
| **Run my Accounts** | CH (Startups) | "Buchhaltung als Service". REST-API dokumentiert. | 1-2 Wochen |
| **Teams/Slack Bridge** | Alle | USP! Kein Konkurrent macht das. Chat-Nachrichten zwischen KMU Hub und Teams/Slack. | 3-4 Wochen |
| **DocuSign** (E-Signatur Fallback) | International | Fuer KMUs die bereits DocuSign nutzen. US-Firma = DSGVO-Bedenken. | 2-3 Wochen |

---

## 5. DSGVO/DSG Quick-Reference (Handlungsfaehige Checkliste)

### VOR Beta-Launch (MUSS)

- [ ] **AVV-Vorlage** erstellen lassen (Rechtsanwalt, ~2.000-4.000 EUR) -- Art. 28 DSGVO
- [ ] **Datenschutzerklaerung** erstellen lassen (~1.000-2.000 EUR)
- [ ] **TOMs dokumentieren** (Technische und Organisatorische Massnahmen) -- Art. 32 DSGVO
- [ ] **Verarbeitungsverzeichnis** anlegen -- Art. 30 DSGVO
- [ ] **Row-Level Security** (PostgreSQL RLS) implementieren -- Mandantentrennung
- [ ] **TLS 1.3** ueberall (API, DB, Redis, LiveKit)
- [ ] **Audit-Logging** (Basis) -- Wer hat wann was geaendert
- [ ] **Passwort-Policy** (min. 12 Zeichen) + **2FA** (TOTP)
- [ ] **Backup-Verschluesselung** (AES-256)
- [ ] **Sub-Processor-Liste** erstellen und veroeffentlichen (Hetzner, OVH, etc.)
- [ ] **Externen DSB** engagieren (~300-500 EUR/Monat) -- empfohlen, noch nicht Pflicht bei 3 MA

### VOR Go-Live (SOLL)

- [ ] **DSGVO-Auskunft-Tool** -- Art. 15: Globale Suche, JSON/CSV-Export aller Daten einer Person
- [ ] **DSGVO-Loeschung** -- Art. 17: Kaskadierte Anonymisierung ueber alle Module. ACHTUNG: GoBD-Rechnungen 10 Jahre behalten, nur Kontaktdaten anonymisieren!
- [ ] **Datenexport** (Portabilitaet) -- Art. 20: Strukturiertes ZIP-Paket
- [ ] **GoBD-konformes Finance-Modul** -- Unveraenderbare Rechnungen, lueckenlose Nummern, Storno statt Loeschung
- [ ] **Consent-Management** -- Einwilligungsflags pro Kontakt pro Zweck (mit Timestamp + Quelle)
- [ ] **Retention-Policy Engine** -- Aufbewahrungsfristen automatisch berechnen (DE: 6/10 Jahre, CH: 10 Jahre)
- [ ] **Incident-Response-Plan** dokumentieren -- 72h-Meldefrist (DE), "so rasch wie moeglich" (CH)
- [ ] **Penetrationstest** beauftragen (~3.000-8.000 EUR)

### Aufbewahrungsfristen (Kurzuebersicht)

| Dokumenttyp | DE | CH | AT |
|-------------|-----|-----|-----|
| Rechnungen/Buchungsbelege | 10 Jahre | 10 Jahre | 7 Jahre |
| Geschaeftsbriefe/E-Mails | 6 Jahre | 10 Jahre | 7 Jahre |
| Personalakten (nach Austritt) | 3 + 6 Jahre (Lohn) | 5 + 10 Jahre (Lohnausweise) | 3 + 7 Jahre |
| Arbeitszeitnachweise | 2 Jahre | -- | -- |
| Vertraege (buchungsrelevant) | 10 Jahre | 10 Jahre | 7 Jahre |

### Schweiz-spezifisch (nDSG seit 01.09.2023)

- Bussen treffen **natuerliche Personen** (Geschaeftsfuehrung!), bis 250.000 CHF
- Nur **natuerliche** Personen geschuetzt (nicht juristische -- anders als DSGVO)
- Kein Pflicht-DSB, aber freiwilliger "Datenschutzberater" erleichtert Meldeverfahren
- Datenfluss DE <-> CH ist **unproblematisch** (beidseitige Angemessenheitsbeschluesse)
- Manche Schweizer Kunden WOLLEN trotzdem Schweizer Datenresidenz (Verkaufsargument!)

### Hosting-Empfehlung

| Zweck | Anbieter | Begruendung |
|-------|----------|-------------|
| SaaS (DE-Kunden) | **Hetzner** (Falkenstein/Nuernberg) | ISO 27001, BSI C5, guenstig |
| SaaS (CH-Kunden) | **Exoscale** (Zuerich/Genf) | Schweizer Datenresidenz |
| Backup/DR | **OVH** (DE) | Georedundanz |
| Self-Hosted | Kundeninfrastruktur | Volle Kontrolle |

---

## 6. Wettbewerber-Positionierung

### Direkte Wettbewerber

| Wettbewerber | Typ | Staerke | Schwaeche | KMU Hub Vorteil |
|-------------|------|---------|-----------|-----------------|
| **Zoho One** (Indien) | All-in-One Suite (40+ Apps) | Einziges vergleichbares All-in-One. Guenstiger Preis (~45 EUR/User/Mo). | Indische Firma (DACH-Misstrauen), ueberladene UI, keine Onsite-Konfiguration, kein Self-Hosting. | EU-Hosting, Massanfertigung, einfachere UI, Self-Hosted |
| **Odoo** (Belgien) | Modulares ERP (40+ Module) | Open Source, EU-Firma, modular. | Pricing-Falle (pro Modul), braucht Partner fuer Customizing, Community-Edition limitiert. | Alles inkludiert, kein Modul-Aufpreis, Branchenprofile filtern Komplexitaet |
| **ClickUp** (USA) | "Everything App" | Versucht alles zu sein (PM + Docs + Chat + Goals). | Massiver Bloat, 5-Level-Hierarchie verwirrt, Performance-Probleme, DSGVO unklar. | EU-Hosting, kein Bloat dank Branchenprofile, CRM/Helpdesk integriert (ClickUp hat keins) |
| **Stackfield** (DE) | PM + Chat + Video | Deutscher Datenschutz-Champion, E2E-verschluesselt. | Kein CRM, kein Helpdesk, kein Finance, UI altmodisch. | Alles was Stackfield kann + CRM + Helpdesk + Finance + Branchenmodule |

### Indirekte Wettbewerber (Kategorie-Champions)

| Kategorie | Champion | KMU Hub Differenzierung |
|-----------|----------|------------------------|
| CRM | Pipedrive (EU), HubSpot (US) | Pipedrive hat kein PM/Helpdesk/Chat. HubSpot wird schnell teuer (Pro = 500 EUR/Mo). KMU Hub: Alles in einem. |
| Helpdesk | Zendesk (US), Zammad (DE) | Zendesk = teuer (55+ EUR/Agent). Zammad = kein CRM. KMU Hub: Helpdesk + CRM + alles andere. |
| PM | Monday.com (IL), Meistertask (DE) | Monday hat kein CRM (Aufpreis!). Meistertask hat kein CRM/Helpdesk/Finance. |
| Zeiterfassung | Clockodo (DE), Toggl (EE) | Clockodo hat kein CRM/PM. KMU Hub: Zeit + PM + CRM in einem. |
| Buchhaltung | Bexio (CH), Lexoffice (DE) | Wir konkurrieren NICHT. Wir INTEGRIEREN. KMU Hub = der Hub, Bexio/Lexoffice = die Buchhalter. |
| Chat/Video | Teams (US), Slack (US) | KMU Hub: eigener Chat/Video, EU-hosted. Kein Microsoft/Salesforce-Lock-in. |

### Wo KMU Hub BESSER ist

1. **All-in-One** -- Kein Toolwechsel, ein Login, eine UI
2. **EU-Datensouveraenitaet** -- 100% EU-Hosting, Self-Hosted Option, kein US Cloud Act
3. **Massanfertigung** -- 1-Woche-Onsite = kein anderer Anbieter macht das
4. **Branchenprofile** -- UI zeigt nur relevante Module, nie ueberladen
5. **Desktop-App** (Electron) -- Offline-faehig, schneller als Cloud-only
6. **Desk-Metapher** -- Einzigartiges UI-Konzept, personalisierbar

### Wo KMU Hub SCHLECHTER ist (noch)

1. **Kein E-Mail-Backend** -- "Dann brauche ich trotzdem Outlook"
2. **Keine Office-Editing** -- "Dann brauche ich trotzdem Microsoft 365"
3. **Keine Custom Fields** -- "Ich kann meine Daten nicht richtig abbilden"
4. **Keine DATEV-Schnittstelle** -- "Mein Steuerberater kann damit nichts anfangen"
5. **Kein Newsletter** -- "Fuer Kunden-Mails brauche ich noch ein extra Tool"
6. **Keine mobile App** -- "Unterwegs geht gar nichts"

### Preis-Kontext

Ein typisches DACH-KMU (15 MA, 3 Buero, 1 Vertrieb) zahlt heute:

| Szenario | Monatliche Kosten |
|----------|-------------------|
| Pipedrive + Freshdesk + CleverReach + Clockodo + Dropbox | ~115 + 30 + 35 + 96 + 75 = ~351 EUR |
| HubSpot Pro + Zendesk | ~665 EUR |
| Zoho One (15 User) | ~675 EUR |
| Microsoft 365 + Notion + Slack | ~738 EUR |
| **KMU Hub** (Zielpreis) | Muss unter 400 EUR/Mo fuer 15 User liegen, um attraktiv zu sein |

---

## 7. Branchen-Empfehlung

### Ranking: Welche Branchen zuerst?

Sortiert nach Abdeckungsgrad (wie viel % des Bedarfs deckt KMU Hub JETZT ab).

| Rang | Branche | Abdeckung | Staerke | Fehlende Elemente | Empfehlung |
|------|---------|-----------|---------|-------------------|------------|
| **1** | **Dienstleister/Agentur/Beratung** (5-30 MA) | **~85%** | CRM + PM + Zeit + Chat + Meetings = Perfekt. Deal-Pipeline, Kanban, Timer -- genau was Agenturen brauchen. | Stunden-zu-Rechnung, Gaeste-Zugang, Auslastungsberichte | **ERSTE ZIELGRUPPE. Kleinster Gap, hoechster Fit.** |
| **2** | **Handwerk** (5-20 MA) | **~80%** | CRM + Rapporte + Zeiterfassung + Einkauf + Fuhrpark. Starke Branchenmodule. | Belegkette (Angebot->Rechnung), Fahrtenbuch, DATEV-Export | **ZWEITE ZIELGRUPPE. Rapporte-Modul ist Differentiator.** |
| **3** | **Bau** (10-50 MA) | **~70%** | Rapporte, Aufmass, Fotodokumentation, Inventar, Schichtplanung. | Maengelmanagement auf Grundriss, GPS-Zeiterfassung, finanzamtkonformes Fahrtenbuch, Zuschlaege | **DRITTE ZIELGRUPPE. Rapporte-Modul ist stark, aber Luecken bei Schichtplanung/Fahrtenbuch.** |
| **4** | **Handel** (5-50 MA) | **~65%** | Inventar, Einkauf, CRM, Berichte. | E-Commerce-Anbindung, Versandabwicklung, Barcode-Scanning, Preislisten | Spaetere Zielgruppe. Handel braucht E-Commerce-Features die nicht Kernkompetenz sind. |
| **5** | **Gastro/Hotel** (10-50 MA) | **~55%** | Schichtplanung (Basis), Einkauf, Chat. | Tisch-Reservierung, Kasse (entfernt), gastro-spezifische Schichtplanung (Zuschlaege, MiLoG, Trinkgeld) | **SCHWAECHSTE BRANCHE. Kasse entfernt = richtige Entscheidung, aber Gastro ist ohne Kasse und Tischreservierung nicht bedienbar.** |

### Empfohlene Go-to-Market Reihenfolge

```
Phase 1 (Beta):     Dienstleister/Agenturen (85% Abdeckung)
Phase 2 (Launch):   + Handwerk (80% Abdeckung)
Phase 3 (6 Mo):     + Bau (70% Abdeckung)
Phase 4 (12 Mo):    + Handel (nach E-Commerce-Integration)
Phase 5 (18+ Mo):   + Gastro (nach spezifischen Erweiterungen)
```

**Begr undung:** Dienstleister und Handwerker haben den hoechsten Feature-Fit, die geringste Luecke, und -- entscheidend -- sie sind die Branchen die am meisten unter Tool-Fragmentierung leiden ("Ich habe 5 verschiedene Tools und keines spricht mit dem anderen").

---

## 8. Naechste Schritte

### Sofort (vor naechstem Sprint)

| Was | Aufwand | Wer | Warum |
|-----|---------|-----|-------|
| MWSt-Saetze multi-country (DE/AT/CH) in finance.ts | 2-3 Tage | Luke | Blockiert Rechnungsmodul fuer DE/AT |
| Akadem. Titel + Anrede-Logik + bevorzugte Sprache | 2-3 Tage | Luke | DACH-Grunderwartung, kleine Aenderung |
| Canned Responses + Private Notizen (Helpdesk) | 3-5 Tage | Luke | Standard-Feature, fehlt komplett |
| Ticket-Kategorien + Geschaeftszeiten-Kalender | 3-5 Tage | Luke | SLA-Berechnung ist sonst falsch |

### Kurzfristig (naechste 4-6 Wochen)

| Was | Aufwand | Abhaengigkeit |
|-----|---------|---------------|
| DATEV-Export (Zeiterfassung + Rechnungen) | 1-2 Wochen | DATEV-Format-Spec |
| QR-Rechnung (Swiss QR-Code) | 1-2 Wochen | Swiss QR-Code Spec |
| PDF-Generierung (Rechnungen/Angebote) | 1-2 Wochen | Go-Library (wkhtmltopdf) |
| Belegkette (Angebot -> Auftrag -> Rechnung) | 3-4 Wochen | Finance-Modul |
| Custom Fields (JSONB-Schema) | 3-4 Wochen | DB-Migration |

### Mittelfristig (2-4 Monate)

| Was | Aufwand | Abhaengigkeit |
|-----|---------|---------------|
| IMAP/SMTP E-Mail-Backend | 6-8 Wochen | `emersion/go-imap` |
| Firma als eigene Entity (nicht String) | 2-3 Wochen | DB-Migration, CRM-Refactoring |
| Duplikaterkennung (CRM) | 1-2 Wochen | Matching-Algorithmus |
| TipTap Rich-Text-Editor (Wiki) | 2-3 Wochen | TipTap npm Package |
| Bexio-API Integration (CH) | 2-4 Wochen | OAuth2, API-Dokumentation |
| GoBD-konforme Rechnungen | 1-2 Wochen | Audit-Log vorhanden? |

### Langfristig (4-8 Monate)

| Was | Aufwand | Abhaengigkeit |
|-----|---------|---------------|
| OnlyOffice WOPI-Integration | 2-4 Wochen | Docker + Go WOPI-Endpoints |
| Nextcloud WebDAV-Integration | 2-3 Wochen | WebDAV-Protokoll |
| Brevo/CleverReach Newsletter-Integration | 2-3 Wochen | REST APIs |
| Skribble E-Signatur-Integration | 2-3 Wochen | Skribble REST API |
| ZUGFeRD/XRechnung | 2-3 Wochen | Go-Libraries (`invopop/gobl`) |
| FinAPI Banking-Integration | 3-4 Wochen | FinAPI Account + API |
| Teams/Slack Bridge | 3-4 Wochen | Teams Bot API, Slack Events API |
| Gaeste-Zugang (Projekte) | 2-3 Wochen | Auth-System erweitern |
| Mobile App (React Native) | 3-6 Monate | React Native Setup |

### Compliance-Roadmap (parallel)

| Wann | Was | Kosten |
|------|-----|--------|
| Vor Beta | AVV + Datenschutzerklaerung (Rechtsanwalt) | 3.000-6.000 EUR |
| Vor Beta | TOMs + Verarbeitungsverzeichnis + RLS | Interne Arbeit |
| Vor Beta | Externer DSB | ~300-500 EUR/Monat |
| Vor Launch | DSGVO-Tools (Auskunft, Loeschung, Export) | 4-6 Wochen Entwicklung |
| Vor Launch | Penetrationstest | 3.000-8.000 EUR |
| Nach Launch | ISO 27001 Vorbereitung | 30.000-70.000 EUR (ab 12 Mo) |
| Nach Launch | Swiss Cluster (Exoscale) | Laufende Hosting-Kosten |

### Geschaetzte Gesamtkosten Compliance bis Launch

**~12.000-24.000 EUR** (+ Entwicklungszeit, die in der Roadmap inkludiert ist)

---

## Anhang: Positionierungs-Statement

> **KMU Hub ist das Betriebssystem fuer dein KMU.**
>
> Buchhaltung und Lohn machst du weiterhin mit deinem Steuerberater / Bexio / DATEV -- aber alles andere (CRM, Projekte, Chat, Meetings, HR-Basics, Rechnungen, Zeiterfassung, Helpdesk, Dokumente) laeuft in EINER App. Die spricht mit deinen bestehenden Tools, liegt auf europaeischen Servern, und zeigt dir genau die Module die deine Branche braucht.
>
> Nicht noch ein Tool. Die Klammer um alles.

---

*Hinweis: Alle Preise und Marktanteile basieren auf Training-Daten (Stand Mai 2025). Vor Verwendung in kundengerichteten Materialien muessen aktuelle Preise verifiziert werden. Rechtliche Angaben stellen keine Rechtsberatung dar.*
