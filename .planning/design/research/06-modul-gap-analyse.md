# Modul-Gap-Analyse: KMU Hub vs. Markt

**Datum:** 2026-02-17
**Methode:** Jedes Frontend-Modul (`desktop/src/renderer/src/modules/`) gegen die Synthese (`00-SYNTHESE.md`) und Wettbewerber-Features abgeglichen.
**Referenz:** Top 20 Feature-Luecken aus Synthese Abschnitt 3, Build vs. Integrate Matrix Abschnitt 1.
**LOC-Angaben:** Ungefaehre Zeilenanzahl der jeweiligen `*Page.tsx` Datei.

---

## Inhalt

1. [CRM (Backend-verbunden)](#1-crm-backend-verbunden)
2. [Kontakte (Lokaler Zustand)](#2-kontakte-lokaler-zustand)
3. [E-Mail](#3-e-mail)
4. [Meetings](#4-meetings)
5. [Work / Projekte](#5-work--projekte)
6. [Kalender](#6-kalender)
7. [Zeiterfassung](#7-zeiterfassung)
8. [Helpdesk](#8-helpdesk)
9. [Buchhaltung](#9-buchhaltung)
10. [Team / HR](#10-team--hr)
11. [Dokumente](#11-dokumente)
12. [Schichtplanung](#12-schichtplanung)
13. [Inventar](#13-inventar)
14. [Einkauf](#14-einkauf)
15. [Fuhrpark](#15-fuhrpark)
16. [Produktion](#16-produktion)
17. [Berichte](#17-berichte)
18. [Formulare](#18-formulare)
19. [Vermietung](#19-vermietung)
20. [Vertraege](#20-vertraege)
21. [Rapporte](#21-rapporte)
22. [Dashboard](#22-dashboard)
23. [Chat](#23-chat)
24. [Zusammenfassung](#zusammenfassung)

---

## Gap-Level Definitionen

| Level | Bedeutung |
|-------|-----------|
| **KLEIN** | Modul ist funktional komplett fuer MVP. Nur Feinschliff/Integrationen fehlen. |
| **MITTEL** | Modul funktioniert, aber 2-5 wichtige Features fehlen fuer Markt-Wettbewerbsfaehigkeit. |
| **GROSS** | Modul hat grundlegende Luecken die den Nutzen stark einschraenken. Ohne Behebung kein Market Fit. |

---

## 1. CRM (Backend-verbunden)

**Dateien:** `modules/crm/contacts/ContactsListPage.tsx` (212), `ContactDetailPage.tsx` (472), `CompaniesListPage.tsx` (213), `CompanyDetailPage.tsx` (325), `DealsListPage.tsx` (262), `DealDetailPage.tsx` (514), `ActivitiesListPage.tsx` (231), `CRMSearchPage.tsx` (188)

**Aktueller Stand:** Backend-verbundenes CRM mit React Query API-Hooks (`useContacts`, `useProjects`). Kontakte, Firmen (eigene Entitaet!), Deals mit Pipeline-Kanban, Aktivitaeten (Call/Meeting/Note/Email/Task), Cross-Entity-Suche, Tags, Custom Fields (nur Anzeige, read-only). Create/Edit-Formulare zeigen "Kommt bald"-Placeholder.

**Gap-Level:** GROSS

**Vorhandene Features:**
- Kontaktliste mit Tabelle, Suche, Pagination, Tags
- Kontaktdetail mit Info-Grid, Custom Fields (Anzeige), Aktivitaeten-Tab, verlinkte Tasks
- Firmen als eigene Entitaet mit Detail-Seite und verlinkten Kontakten
- Deals mit Listen- und Pipeline-Ansicht, Stages, Wert-Anzeige
- Deal-Detail mit Stage, Wert, verlinkte Kontakte/Firmen, Custom Fields, Aktivitaeten
- Aktivitaetenliste mit Typ-Filter-Tabs, Abschluss-Toggle
- Cross-Entity-Suche (Kontakte, Firmen, Deals, Aktivitaeten)

**Fehlende Features (aus Recherche):**
- [ ] **Create/Edit-Formulare** (Kontakte, Firmen, Deals) -- Alles zeigt "Kommt bald" -- Aufwand: 2-3 Wochen
- [ ] **Custom Fields editierbar** -- Aktuell nur read-only Anzeige, keine Erstellung/Bearbeitung -- Aufwand: 3-4 Wochen (Synthese #3)
- [ ] **Akademischer Titel + Anrede-Logik** -- "Herr Prof. Dr. Mueller", Sie/Du-Flag, bevorzugte Sprache -- Aufwand: 2-3 Tage (Synthese #10)
- [ ] **Duplikaterkennung** -- Kein Matching bei Import/Erstellung sichtbar -- Aufwand: 1-2 Wochen (Synthese #12)
- [ ] **E-Mail-zu-Kontakt-Zuordnung** -- Mails dem CRM-Kontakt automatisch zuordnen -- Aufwand: 1-2 Wochen (abhaengig von E-Mail-Backend)
- [ ] **Kontakt-Timeline** -- Chronologische Ansicht aller Interaktionen (Mails, Calls, Meetings, Deals) -- Aufwand: 1-2 Wochen
- [ ] **Import/Export (CSV, vCard)** -- Kein Import-Dialog im CRM-Modul (nur in Kontakte-Modul) -- Aufwand: 3-5 Tage
- [ ] **Consent-Management** -- Einwilligungsflags pro Kontakt pro Zweck mit Timestamp -- Aufwand: 1 Woche (DSGVO)

**Frontend-Aenderungen noetig:**
- Vollstaendige CRUD-Formulare fuer Kontakte, Firmen, Deals bauen
- Custom Fields Editor UI (Feld-Typen: Text, Zahl, Datum, Dropdown, Checkbox, URL)
- Anrede/Titel-Dropdown in Kontakt-Formularen (Herr/Frau/Divers + akad. Titel)
- Duplikaterkennung-Hinweis-Banner bei Kontakt-Erstellung
- Timeline-Komponente fuer Kontakt/Firmen-Detail

---

## 2. Kontakte (Lokaler Zustand)

**Datei:** `modules/kontakte/KontaktePage.tsx` (500)

**Aktueller Stand:** Eigenstaendiges Kontakte-Modul mit lokalem Zustand (Zustand-Store, Mock-Daten). Volle CRUD, Gruppen, Favoriten, Import-Dialog, Duplikaterkennung-UI, Call-Overlay, vCard-Export. Firma als String gespeichert (nicht als Entitaet).

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Vollstaendiges CRUD (Erstellen, Bearbeiten, Loeschen)
- Kategorien-Sidebar, Gruppen, Favoriten
- Import-Dialog, Duplikaterkennung-UI
- Call-Overlay, E-Mail/Chat-Aktionen, vCard-Export
- Suche, Sortierung, Filter

**Fehlende Features (aus Recherche):**
- [ ] **Firma als Entitaet statt String** -- Im CRM-Modul existiert Firma als Entity, aber Kontakte-Modul speichert nur String. Zusammenfuehrung noetig -- Aufwand: 2-3 Wochen (Synthese #7)
- [ ] **Backend-Anbindung** -- Aktuell rein lokal (Zustand), keine API-Calls. Muss auf Backend-CRM migriert werden -- Aufwand: 2-3 Wochen
- [ ] **Akademischer Titel + Anrede-Logik** -- Kontaktmodell hat nur `salutation: 'Herr' | 'Frau'`, kein Titel-Feld -- Aufwand: 2-3 Tage (Synthese #10)
- [ ] **Consent-Management pro Kontakt** -- Keine Einwilligungsflags vorhanden -- Aufwand: 3-5 Tage

**Frontend-Aenderungen noetig:**
- Kontakte-Modul mit CRM-Modul zusammenfuehren oder auf gemeinsames Backend umstellen
- Firma-Feld von Freitext auf Entitaets-Auswahl umbauen (Autocomplete + Erstellung)
- Titel-Feld zum Kontaktformular hinzufuegen
- DSGVO-Tab/Sektion in Kontaktdetail (Einwilligungen, Loeschrecht)

---

## 3. E-Mail

**Datei:** `modules/mails/MailsPage.tsx` (553)

**Aktueller Stand:** Vollstaendiges 3-Panel-Layout (Ordner, Liste, Inhalt), Compose-Modal, Reply/Forward, Ordner-Verwaltung, Stern, Archiv, Drucken, Export. KEIN BACKEND -- alles Zustand-Store mit Mock-Daten.

**Gap-Level:** GROSS

**Vorhandene Features:**
- 3-Panel-Layout (Outlook-aehnlich)
- Ordner-Navigation (Inbox, Sent, Drafts, Archive, Trash)
- ComposeModal mit An/CC/BCC, Betreff, Body
- Reply/Reply All/Forward
- Stern, Archivieren, Als gelesen/ungelesen markieren
- Drucken, Export

**Fehlende Features (aus Recherche):**
- [ ] **IMAP/SMTP Backend** -- KRITISCH! Ohne E-Mail-Backend bleibt die halbe App leer. UI ist fertig. -- Aufwand: 6-8 Wochen (Synthese #1, Go-Library `emersion/go-imap`)
- [ ] **E-Mail-zu-Kontakt-Zuordnung** -- Mails automatisch dem CRM-Kontakt zuordnen -- Aufwand: 1-2 Wochen (abhaengig von IMAP-Backend)
- [ ] **Rich-Text-Editor** (fuer Compose) -- Aktuell vermutlich Plain-Text. TipTap/ProseMirror fuer HTML-Mails -- Aufwand: 1-2 Wochen
- [ ] **E-Mail-Vorlagen** -- Vorlagen fuer wiederkehrende Mails (Angebots-Versand etc.) -- Aufwand: 3-5 Tage
- [ ] **E-Mail-Signatur** -- Konfigurierbare Signatur pro Benutzer -- Aufwand: 2-3 Tage
- [ ] **Attachment-Handling** -- Upload/Download von Anhaengen -- Aufwand: 1 Woche (Backend-abhaengig)
- [ ] **DSGVO: E-Mail-Aufbewahrungsfristen** -- DE: 6 Jahre Geschaeftsbriefe, CH: 10 Jahre -- Aufwand: 1 Woche

**Frontend-Aenderungen noetig:**
- API-Layer fuer IMAP/SMTP-Anbindung (React Query Hooks)
- Rich-Text-Editor (TipTap) in ComposeModal integrieren
- Kontakt-Chip-Auswahl (aus CRM-Daten) fuer An/CC/BCC
- Signatur-Editor in Profil-Einstellungen
- Vorlagen-Auswahl im Compose-Dialog

---

## 4. Meetings

**Datei:** `modules/meetings/MeetingsPage.tsx` (617)

**Aktueller Stand:** Grid- und Timeline-Ansichten, Filter-Tabs, Meeting-Karten, MeetingRoomView, Formular-Dialog, Detail-Panel. Zustand-Store.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Grid- und Timeline-Ansichten
- Filter-Tabs (Alle, Bevorstehend, Vergangen)
- Meeting-Karten mit Status, Teilnehmer, Raum
- MeetingRoomView (Raumbelegung)
- Formular-Dialog (Titel, Datum, Uhrzeit, Raum, Teilnehmer, Beschreibung)
- Detail-Panel

**Fehlende Features (aus Recherche):**
- [ ] **LiveKit Video-Integration** -- Video/Audio-Konferenz direkt in der App. LiveKit geplant aber nicht implementiert -- Aufwand: 3-4 Wochen (Backend: LiveKit Server)
- [ ] **Kalender-Synchronisation** -- Meetings <-> Kalender bidirektional -- Aufwand: 1-2 Wochen
- [ ] **Einladungs-E-Mails** -- Automatische Mail an Teilnehmer bei Erstellung -- Aufwand: 1 Woche (abhaengig von E-Mail-Backend)
- [ ] **Wiederkehrende Meetings** -- Wiederholungsmuster (taeglich, woechentlich, monatlich) -- Aufwand: 3-5 Tage
- [ ] **Meeting-Notizen / Protokoll** -- Waehrend/nach dem Meeting Notizen erfassen -- Aufwand: 1 Woche

**Frontend-Aenderungen noetig:**
- LiveKit-Komponente fuer Video einbetten (Electron WebRTC)
- Wiederholungs-Auswahl im Formular-Dialog
- Notiz-Editor im Detail-Panel
- Kalender-Integration (gemeinsamer Event-Store)

---

## 5. Work / Projekte

**Dateien:** `modules/work/projects/ProjectsListPage.tsx` (367), `modules/work/tasks/MyTasksPage.tsx` (572)

**Aktueller Stand:** Backend-verbunden mit React Query API-Hooks (`useProjects`, `useCreateProject`, `useCreateFromTemplate`, `useMyTasks`, `useCreateTask`, `useUpdateTask`). Projekte mit Vorlagen, Tasks mit Prioritaeten, Listen-Ansicht.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Projektliste mit Erstellung und Vorlagen
- Task-Management mit Prioritaeten, Status-Aenderung
- Backend-Anbindung (React Query)
- Subtasks, Abhaengigkeiten (UI vorhanden)
- Kanban-Board (in separater Komponente)

**Fehlende Features (aus Recherche):**
- [ ] **Gaeste-Zugang** -- Kunden/Partner sehen Projektstatus. Kein Konkurrent im All-in-One hat das gut -- Aufwand: 2-3 Wochen (Synthese #18, Auth-System erweitern)
- [ ] **Stunden-zu-Rechnung Workflow** -- "Diese 40h fuer Kunde X --> Rechnung generieren" -- Aufwand: 1-2 Wochen (Synthese #17)
- [ ] **Gantt-Ansicht (interaktiv)** -- Gantt UI existiert, aber Drag-Resize fuer Balken fehlt -- Aufwand: 2-3 Wochen
- [ ] **Auslastungsberichte** -- Teamauslastung pro Projekt/Mitarbeiter visualisieren -- Aufwand: 1-2 Wochen
- [ ] **Projekt-Budget-Tracking** -- Budget vs. tatsaechliche Kosten (Stunden x Stundensatz) -- Aufwand: 1-2 Wochen

**Frontend-Aenderungen noetig:**
- "Rechnung erstellen aus Zeiteintraegen"-Button in Projekt-Detail
- Gaeste-Portal-Ansicht (read-only Projektstatus)
- Budget-Sektion in Projekt-Detail (Soll/Ist-Vergleich)
- Interaktive Gantt-Balken (Drag-to-resize/move)

---

## 6. Kalender

**Datei:** `modules/kalender/KalenderPage.tsx` (2143)

**Aktueller Stand:** Umfangreichstes Modul (2143 LOC). Zwei Top-Tabs: Kalender + Terminbuchung. Wochen-/Tages-/Monatsansichten, ueberlappende Events mit Layout-Algorithmus, Ganztages-Events, aktuelle Zeitlinie, Quick-Create-Popover, vollstaendiges Event-Formular (Titel, Datum/Zeit, Ganztag, Kategorie, Ort, Raum, Beschreibung, Wiederholung, Erinnerung, Kalenderauswahl, Teilnehmersuche, Video-Call-Toggle), Event-Detail-Panel mit RSVP-Status, Raumbuchung, Kalenderquellen. Terminbuchung-Tab mit 8 Buchungsservices, Tagesuebersicht-Timeline.

**Gap-Level:** KLEIN

**Vorhandene Features:**
- Woche/Tag/Monat-Ansichten mit interaktivem Layout
- Quick-Create bei Klick auf Zeitslot
- Vollstaendiges Event-Formular (Wiederholung, Erinnerung, Raum, Teilnehmer, Video)
- RSVP-Status-Anzeige (akzeptiert/abgelehnt/vielleicht/ausstehend)
- Raumbuchung, Kalenderquellen (eigene/geteilte/andere)
- Terminbuchung-Tab mit 8 Services, Tagesuebersicht

**Fehlende Features (aus Recherche):**
- [ ] **CalDAV-Synchronisation** -- Kalender mit externen Clients synchronisieren (Thunderbird, Apple Calendar) -- Aufwand: 2-3 Wochen (Backend)
- [ ] **Drag-and-Drop zum Verschieben/Resizen von Events** -- Aktuell nur Anzeige, kein Drag -- Aufwand: 1-2 Wochen
- [ ] **Push-Erinnerungen** -- Desktop-Benachrichtigungen (Electron Notification API) -- Aufwand: 3-5 Tage
- [ ] **Externer Buchungslink** -- Kunden koennen online Termine buchen (wie Calendly) -- Aufwand: 2-3 Wochen

**Frontend-Aenderungen noetig:**
- Drag-and-Drop-Handler fuer Events (Week/Day-View)
- Electron-Notification fuer Erinnerungen
- Oeffentliche Buchungsseite (separates Frontend oder Embedded-Widget)

---

## 7. Zeiterfassung

**Dateien:** `modules/zeiterfassung/ZeiterfassungPage.tsx` (10, Wrapper), `modules/profil/tabs/ZeiterfassungTab.tsx` (222)

**Aktueller Stand:** Timer-Toolbar mit Start/Pause/Resume/Stop, Kategorie-Auswahl, Beschreibung, aktive Timer-Anzeige mit Elapsed-Time, Tages-Zusammenfassung (Soll/Ist mit Fortschrittsbalken), 7 Ansichts-Tabs (Uebersicht/Heute/Woche/Monat/Reports/Team/Kategorien). Zustand-Store.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Live-Timer mit Start/Pause/Resume/Stop
- Kategorie-Auswahl mit Farbindikator
- Beschreibungs-Eingabe
- Tages-Fortschritt (Minuten vs. Ziel, farbkodierter Balken)
- 7 Ansichts-Tabs (UI-Rahmen vorhanden)

**Fehlende Features (aus Recherche):**
- [ ] **DATEV-Export** -- Zeiteintraege im DATEV-Format exportieren (CSV, Windows-1252, Semikolon) -- Aufwand: 1-2 Wochen (Synthese #2, KRITISCH fuer DE)
- [ ] **Stunden-zu-Rechnung Workflow** -- Gebuchte Stunden zu Rechnung konvertieren -- Aufwand: 1-2 Wochen (Synthese #17)
- [ ] **Projekt-Zuordnung** -- Timer direkt einem Projekt/Task zuordnen (nicht nur Kategorie) -- Aufwand: 3-5 Tage
- [ ] **GPS-Zeiterfassung** -- Standort bei Start/Stop erfassen (fuer Bau/Handwerk) -- Aufwand: 1-2 Wochen
- [ ] **Ueberstunden-Berechnung** -- Automatische Ueberstunden-Erkennung basierend auf Soll-Arbeitszeit -- Aufwand: 3-5 Tage
- [ ] **Abwesenheits-Integration** -- Urlaub/Krankheit aus Team-Modul in Zeiterfassung reflektieren -- Aufwand: 3-5 Tage
- [ ] **Genehmigungsworkflow** -- Vorgesetzter genehmigt Wochenrapport -- Aufwand: 1-2 Wochen

**Frontend-Aenderungen noetig:**
- Projekt/Task-Dropdown im Timer-Toolbar
- Export-Button mit Format-Auswahl (DATEV, CSV, Excel)
- "Rechnung erstellen"-Button fuer selektierte Zeiteintraege
- Ueberstunden-Anzeige im Wochen/Monats-Report
- Genehmigungs-Banner fuer Vorgesetzte

---

## 8. Helpdesk

**Datei:** `modules/helpdesk/HelpdeskPage.tsx` (~1008)

**Aktueller Stand:** Drei Tabs: Tickets, Wissensdatenbank, Statistik. Ticket-Tabelle mit Status/Prioritaet/SLA-Filtern, Ticket-Detail-Slide-over mit Nachrichten-Thread, Reply/Interne-Notiz-Toggle, Status-Aenderungs-Dropdown. Knowledge Base mit Artikel-Karten und Detail-Ansicht ("War dieser Artikel hilfreich?"). Statistiken mit KPIs, Balkendiagramm, Status/Prioritaets-Aufschluesselungen.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Ticket-Tabelle mit Multi-Filter (Status, Prioritaet, SLA)
- Ticket-Detail mit Nachrichten-Thread
- Reply / Interne Notiz Toggle
- Status-Aenderung per Dropdown
- SLA-Anzeige (Reaktionszeit)
- Wissensdatenbank mit Artikel-Karten und "Hilfreich?"-Bewertung
- Statistik-Dashboard (KPIs, Charts)

**Fehlende Features (aus Recherche):**
- [ ] **Canned Responses / Textbausteine** -- Standard bei jedem Helpdesk. Fehlt komplett -- Aufwand: 3-5 Tage (Synthese #11)
- [ ] **Ticket-Kategorien** -- Keine Kategorien-Zuordnung sichtbar -- Aufwand: 2-3 Tage (Synthese Sofort-Empfehlung)
- [ ] **Geschaeftszeiten-Kalender** -- SLA-Berechnung beruecksichtigt keine Oeffnungszeiten -- Aufwand: 3-5 Tage (Synthese Sofort-Empfehlung)
- [ ] **E-Mail-zu-Ticket Konvertierung** -- Eingehende Mails automatisch als Ticket erfassen -- Aufwand: 1-2 Wochen (abhaengig von IMAP-Backend)
- [ ] **Ticket-Zuweisung / Routing** -- Automatische Zuweisung an Team/Agent basierend auf Regeln -- Aufwand: 1-2 Wochen
- [ ] **Kundenzufriedenheit (CSAT)** -- Bewertung nach Ticket-Schliessung -- Aufwand: 3-5 Tage
- [ ] **Custom Fields fuer Tickets** -- Branchenspezifische Felder -- Aufwand: siehe CRM Custom Fields (Synthese #3)
- [ ] **Wissensdatenbank: Rich-Text-Editor** -- Artikel-Inhalt ist vermutlich Plain-Text -- Aufwand: 1-2 Wochen (TipTap, Synthese #6)

**Frontend-Aenderungen noetig:**
- Canned-Response-Sidebar oder Dropdown im Reply-Bereich
- Kategorie-Auswahl beim Ticket-Erstellen und -Bearbeiten
- Geschaeftszeiten-Konfiguration in Helpdesk-Einstellungen
- Agent-Zuweisungs-Dropdown in Ticket-Detail
- CSAT-Rating-Widget nach Ticket-Schliessung
- TipTap-Editor fuer Wissensdatenbank-Artikel

---

## 9. Buchhaltung

**Datei:** `modules/buchhaltung/BuchhaltungPage.tsx` (675)

**Aktueller Stand:** 6 Tabs: Rechnungen, Angebote, Ausgaben, Mahnungen, Transaktionen, Reports. Stats-Karten (Einnahmen, Ausgaben, Saldo, offene Rechnungen, offene Mahnungen). Rechnungstabelle mit Betrag (formatCHF), Faelligkeit, Restbetrag, Status. Angebote-Tab aehnlich. Ausgaben mit Genehmigung/Ablehnung, Kategorien, Lieferanten. Mahnwesen mit 3-Stufen-Eskalation (Stufe 1-2-3 Visualisierung), Versenden/Eskalieren/Inkasso-Aktionen. Reports mit Einnahmen-vs-Ausgaben-Balkendiagramm, Ausgaben-nach-Kategorie. Rechnungs-Detail-Panel. NUR CHF-Waehrung, NUR Schweizer MWSt-Saetze (7.7%, 8.1%).

**Gap-Level:** GROSS

**Vorhandene Features:**
- Rechnungen CRUD mit Status-Tracking
- Angebote (type: 'quote') mit separatem Tab
- Ausgaben mit Genehmigung/Ablehnung
- 3-Stufen-Mahnwesen mit Eskalation
- Transaktionen-Uebersicht
- Reports mit Charts (Einnahmen vs. Ausgaben, Ausgaben nach Kategorie)
- Zahlungs-Erfassung, Export-Dialog

**Fehlende Features (aus Recherche):**
- [ ] **MWSt multi-country (DE/AT/CH)** -- Nur CH-Saetze (7.7%, 8.1%). DE: 19%/7%, AT: 20%/10%/13%, CH: 8.1%/2.6%/3.8% -- Aufwand: 2-3 Tage (Synthese #8, `finance.ts` aendern)
- [ ] **Belegkette (Angebot -> Auftrag -> Lieferschein -> Rechnung)** -- DER Workflow fuer Handwerker/Dienstleister. "1 Klick von Angebot zu Rechnung" -- Aufwand: 3-4 Wochen (Synthese #4)
- [ ] **QR-Rechnung (Swiss QR-Code)** -- Seit 2022 PFLICHT in der Schweiz -- Aufwand: 1-2 Wochen (Synthese #5)
- [ ] **ZUGFeRD/XRechnung** -- Ab 2025 Empfang Pflicht DE, ab 2027/2028 Versand Pflicht -- Aufwand: 2-3 Wochen (Synthese #16)
- [ ] **DATEV-Export** -- Buchungsstapel-Export fuer Steuerberater (60-75% DE-KMUs) -- Aufwand: 1-2 Wochen (Synthese #2, KRITISCH)
- [ ] **PDF-Generierung** -- Rechnungen/Angebote als PDF versenden/archivieren -- Aufwand: 1-2 Wochen (Synthese #9)
- [ ] **GoBD-konforme Rechnungen** -- Lueckenlose Nummern, Storno statt Loeschung, Aenderungsprotokoll -- Aufwand: 1-2 Wochen (Synthese #20)
- [ ] **Bexio-API Integration** -- DER Schweizer Buchhaltungs-Standard. Bidirektional: Kontakte, Rechnungen, Zahlungen -- Aufwand: 2-4 Wochen (Synthese #15)
- [ ] **Banking-Integration (FinAPI)** -- Automatischer Bankabgleich -- Aufwand: 3-4 Wochen
- [ ] **Multi-Waehrung** -- Nur CHF. Mindestens EUR muss zusaetzlich unterstuetzt werden -- Aufwand: 1 Woche
- [ ] **Stunden-zu-Rechnung Workflow** -- Zeiteintraege zu Rechnung konvertieren -- Aufwand: 1-2 Wochen (Synthese #17)

**Frontend-Aenderungen noetig:**
- `finance.ts`: MWSt-Saetze-Array um DE/AT erweitern, `formatCHF` zu `formatCurrency(amount, currency)` umbauen
- "Rechnung erstellen aus Angebot"-Button (Belegkette)
- QR-Code-Vorschau auf Rechnungs-PDF
- DATEV-Export-Button in Reports-Tab
- Waehrungs-Auswahl (CHF/EUR) im Rechnungs-Formular
- PDF-Vorschau-Panel
- Bexio-Sync-Status-Anzeige
- GoBD-Compliance: Loeschen durch Storno ersetzen, Audit-Log anzeigen

---

## 10. Team / HR

**Datei:** `modules/team/TeamPage.tsx` (1213)

**Aktueller Stand:** 5 Tabs: Mitglieder, Antraege, Abwesenheiten, Lohn, Schulungen. Mitglieder mit Grid-/Listen-Ansicht, Abteilungs-Karten, Suche, Einladen/Bearbeiten/Deaktivieren, Cross-Modul-Navigation (E-Mail/Call/Chat). Antraege mit HR-Request-Cards und Genehmigungs-Workflow. Abwesenheiten mit AbsenceCalendar. Lohn mit Monats-Navigation, Lohntabelle (Brutto, Abzuege, Netto), Gehalts-Aufschluesselung (AHV/IV/EO, BVG Pensionskasse, Quellensteuer, Sonstige). Schulungen mit Katalog (Typen: Sicherheit/Technik/Soft Skills/Compliance/Zertifizierung), Pflicht/Optional-Badges, Gueltigkeits-Tracking, Teilnahme-Tabelle mit Ablauf-Warnungen.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Mitglieder-Verwaltung (Grid/Liste, CRUD, Abteilungen)
- HR-Antraege mit Genehmigungs-Workflow
- Abwesenheits-Kalender
- Lohnanzeige mit Schweizer Abzuegen (AHV, BVG, Quellensteuer)
- Schulungs-Katalog mit Gueltigkeits-Tracking
- Beschaeftigungstypen: Vollzeit/Teilzeit/Stundenlohn

**Fehlende Features (aus Recherche):**
- [ ] **Lohnabrechnung: NICHT SELBST BAUEN** -- Nur Anzeige ist OK, echte Berechnung via Integration (DATEV, Bexio). DE hat 6 SV-Zweige x 6 Steuerklassen x 16 Bundeslaender. Unser Lohn-Tab zeigt nur CH-Abzuege -- Aufwand: Integration-only
- [ ] **Multi-Country Lohn-Anzeige** -- Nur Schweizer Abzuege (AHV, BVG, Quellensteuer). DE/AT-Abzuege fehlen -- Aufwand: 1 Woche
- [ ] **Digitale Personalakte** -- Dokumente pro Mitarbeiter (Vertrag, Zeugnis, Zertifikate) -- Aufwand: 1-2 Wochen
- [ ] **Onboarding-Checkliste** -- Standardisierte Einarbeitungs-Checklisten fuer Neue -- Aufwand: 1 Woche
- [ ] **Org-Chart** -- Organisationsstruktur visuell darstellen -- Aufwand: 1-2 Wochen
- [ ] **Mitarbeiter-Self-Service-Portal** -- Eigene Daten einsehen, Antraege stellen, Gehaltsabrechnung herunterladen -- Aufwand: 2-3 Wochen

**Frontend-Aenderungen noetig:**
- Laender-Auswahl bei Mitarbeiter-Erstellung (CH/DE/AT)
- Laenderspezifische Abzugs-Anzeige (DE: Lohnsteuer/SV, AT: SV/Lohnsteuer)
- Dokumente-Sektion im Mitarbeiter-Detail
- Onboarding-Tab mit Checklisten
- Org-Chart-Visualisierung (Baumstruktur)

---

## 11. Dokumente

**Datei:** `modules/dokumente/DokumentePage.tsx` (1407)

**Aktueller Stand:** 2 Tabs: Dateien, Wiki. Dateien: Ordner-Baum-Sidebar mit verschachtelten Ordnern, Grid-/Listen-Ansichten, Drag-Drop-Upload-Overlay, Speicherplatz-Anzeige, Datei-Aktionen (Vorschau, Download, Umbenennen, Teilen, Verschieben, Favorit, Loeschen), Dateityp-Icons/Farben, Datei-Detail-Panel mit Tags, Teilen-Dialog, Verschieben-Dialog. System-Ordner (Root, Geteilt, Favoriten, Tresor). Wiki: Kategorie-Sidebar, Suche mit Tag-Filter, Artikel-Karten, Artikel-Detail-Ansicht mit Metadaten, Artikel-Formular mit Kategorie/Inhalt/Tags. Wiki-Inhalt nutzt `<textarea>` -- KEIN Rich-Text-Editor.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Dateimanager mit Ordner-Baum, Grid/Liste, Drag-Drop-Upload
- Speicherplatz-Anzeige
- Datei-Aktionen (Vorschau, Download, Umbenennen, Teilen, Verschieben, Favorit, Loeschen)
- Tresor (verschluesselte Dateien)
- Wiki mit Kategorien, Tags, CRUD
- Teilen-Dialog, Verschieben-Dialog

**Fehlende Features (aus Recherche):**
- [ ] **Rich-Text-Editor (Wiki)** -- Wiki nutzt plain `<textarea>`. TipTap/ProseMirror fuer WYSIWYG -- Aufwand: 2-3 Wochen (Synthese #6, HOCH)
- [ ] **OnlyOffice WOPI-Integration** -- .docx/.xlsx direkt in KMU Hub bearbeiten -- Aufwand: 2-4 Wochen (Synthese #14, Docker + Go WOPI-Endpoints)
- [ ] **Externer Link-Share** -- Dateien per signiertem Link an Kunden/Partner senden -- Aufwand: 3-5 Tage (Synthese #13)
- [ ] **Versionierung (Anzeige)** -- Versionsverlauf einer Datei einsehen -- Aufwand: 1 Woche
- [ ] **Nextcloud WebDAV-Integration** -- Dateien aus Nextcloud in KMU Hub anzeigen -- Aufwand: 2-3 Wochen (Synthese #19)
- [ ] **Datei-Vorschau im Browser** -- PDF/Bild-Vorschau inline (nicht nur Download) -- Aufwand: 1-2 Wochen
- [ ] **Volltextsuche in Dokumenten** -- OCR/Text-Extraktion fuer durchsuchbare Dokumente -- Aufwand: 2-3 Wochen (Backend)

**Frontend-Aenderungen noetig:**
- `<textarea>` in Wiki-Formular durch TipTap-Editor ersetzen
- OnlyOffice iFrame-Embedding fuer Office-Dokumente
- "Link teilen"-Button mit Token-basierter URL-Generierung
- Versions-Timeline im Datei-Detail-Panel
- Inline-Vorschau fuer PDF/Bilder (Modal oder Panel)
- Nextcloud-Browser-Integration (WebDAV-Ordner als externe Quelle)

---

## 12. Schichtplanung

**Datei:** `modules/schichten/SchichtenPage.tsx` (1097)

**Aktueller Stand:** 3 Tabs: Wochenplan, Vorlagen, Anfragen. KPIs: Auslastung, offene Schichten, Tausch-Anfragen, Wochenstunden. Wochenraster (7 Spalten Mo-So x Mitarbeiter-Zeilen), farbige Schicht-Bloecke mit Vorlagen-Icons, Verfuegbarkeits-Punkte (gruen/gelb/rot), Heute-Hervorhebung, Hover-to-Add. Schicht-Vorlagen: Frueh/Spaet/Nacht mit Start/End-Zeiten, Pausenminuten, Farbe, Netto-Stunden. Tausch-Anfragen mit Antragsteller<->Tauscher-Anzeige, Grund, Genehmigen/Ablehnen. Zuweisungs- und Vorlagen-Dialoge.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Visuelles Wochenraster (Mo-So x Mitarbeiter)
- Farbige Schicht-Bloecke mit Vorlagen-Farben
- Verfuegbarkeits-Indikatoren (gruen/gelb/rot)
- 3 Standard-Vorlagen (Frueh/Spaet/Nacht)
- Tausch-Anfragen mit Genehmigungs-Workflow
- KPI-Dashboard (Auslastung, offene Schichten, Wochenstunden)
- Zuweisungs-Dialog (Mitarbeiter, Vorlage, Datum, Notizen)

**Fehlende Features (aus Recherche):**
- [ ] **Zuschlaege (Nacht/Wochenende/Feiertag)** -- Nacht-/Wochenend-/Feiertagszuschlaege fehlen komplett -- Aufwand: 1-2 Wochen
- [ ] **Arbeitszeitgesetz-Regeln** -- Maximale Arbeitszeit, Ruhezeiten, Mindest-Pausen automatisch pruefen -- Aufwand: 1-2 Wochen
- [ ] **Konflikterkennung** -- Doppelbelegungen, Ueberarbeitung automatisch warnen -- Aufwand: 1 Woche
- [ ] **Feiertags-Kalender (CH/DE/AT)** -- Laenderspezifische Feiertage einbeziehen -- Aufwand: 3-5 Tage
- [ ] **Mitarbeiter-Self-Service** -- Eigene Verfuegbarkeit eintragen, Tausch-Anfragen stellen -- Aufwand: 1-2 Wochen
- [ ] **Drag-and-Drop im Wochenplan** -- Schichten per Drag verschieben -- Aufwand: 1-2 Wochen
- [ ] **PDF-Export Wochenplan** -- Ausdruckbarer Schichtplan fuer Aushang -- Aufwand: 3-5 Tage

**Frontend-Aenderungen noetig:**
- Zuschlag-Anzeige auf Schicht-Bloecken (z.B. "+25% Nacht")
- Validierungs-Warnungen (rote Umrandung bei Regelverstoessen)
- Konflikterkennung-Banner
- Feiertags-Spalten-Markierung
- Drag-and-Drop-Handler fuer Schicht-Bloecke
- Export-Button (PDF/Excel)

---

## 13. Inventar

**Datei:** `modules/inventar/InventarPage.tsx` (1039)

**Aktueller Stand:** 3 Tabs: Artikel, Lagerorte, Bewegungen. Artikel-Tabelle mit Status-Punkt (ok/warning/critical), Name, SKU, Kategorie, Bestand, Mindestbestand, Standort, Preis (CHF). Lager-Visualisierung mit Bestands-Balken und Mindestbestand-Markierung. Standort-Karten mit Typ-Icons (Lager/Laden/Fahrzeug), Artikel-Zusammenfassung, kritische Anzahl. Bewegungen-Tabelle (Ein/Aus/Transfer/Korrektur) mit Datum, Artikel, Typ, Menge, Von/Nach, Referenz, Ersteller. Detail-Panel. Artikel-Dialog (CRUD mit Kategorie/Einheit/Standort/Mindestbestand/Preis/Barcode). Bewegungs-Dialog.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Artikel-Verwaltung mit SKU, Kategorie, Bestand, Mindestbestand
- Bestands-Visualisierung (Balken mit Schwellenwert)
- Standort-Verwaltung (Lager/Laden/Fahrzeug)
- Bewegungen (Eingang/Ausgang/Transfer/Korrektur)
- Barcode-Feld vorhanden
- Detail-Panel

**Fehlende Features (aus Recherche):**
- [ ] **Belegkette-Anbindung** -- Wareneingang automatisch aus Einkaufs-Bestellung -- Aufwand: 1-2 Wochen (Synthese)
- [ ] **Barcode-Scanner** -- Barcode per Kamera/Scanner erfassen (Electron API) -- Aufwand: 1-2 Wochen
- [ ] **Automatische Nachbestellung** -- Bei Unterschreitung Mindestbestand: Bestellvorschlag generieren -- Aufwand: 1 Woche
- [ ] **Chargen-/Seriennummern-Tracking** -- Fuer Rueckverfolgbarkeit -- Aufwand: 1-2 Wochen
- [ ] **Inventur-Workflow** -- Zaehlliste generieren, Ist-vs-Soll-Vergleich -- Aufwand: 1-2 Wochen
- [ ] **Multi-Waehrung fuer Preise** -- Nur CHF, kein EUR -- Aufwand: 3-5 Tage

**Frontend-Aenderungen noetig:**
- Scanner-Integration (Electron IPC fuer Barcode-Kamera)
- "Nachbestellen"-Button bei kritischem Bestand (Link zu Einkauf)
- Inventur-Tab mit Zaehlliste und Diff-Anzeige
- Chargen-/Seriennummern-Felder im Artikel-Dialog

---

## 14. Einkauf

**Datei:** `modules/einkauf/EinkaufPage.tsx` (1210)

**Aktueller Stand:** 3 Tabs: Bestellungen, Lieferanten, Katalog. Bestellungs-Tabelle mit Bestellnummer, Lieferant, Status-Badge, Betrag (CHF), Lieferdatum. Status-Timeline-Visualisierung (Entwurf -> Gesendet -> Bestaetigt -> Teillieferung -> Erhalten, oder Storniert). Lieferanten-Karten mit Kontaktinfos, Zahlungsbedingungen, Bestellanzahl. Bestell-Detail-Panel mit Betrags-/Artikel-Zusammenfassung, Daten, Status-Timeline, Positionen mit empfangenen Mengen, Lieferanten-Link. Neuer-Bestellungs-Dialog mit Lieferanten-Auswahl, Positionen (Name/Menge/Stueckpreis) mit Hinzufuegen/Entfernen, Gesamt, Lieferdatum, Notizen. Wareneingangs-Dialog mit Artikel-spezifischer Empfangsmengen-Eingabe, Teillieferungs-Checkbox. Katalog-Tab: Platzhalter/Leer.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- Bestellungen mit vollstaendigem Lifecycle (Entwurf bis Empfangen)
- Status-Timeline-Visualisierung
- Lieferanten-Verwaltung mit Kontaktdaten
- Wareneingang mit Teillieferungs-Unterstuetzung
- Neuer-Bestellungs-Dialog mit Positionen
- Detail-Panels fuer Bestellung und Lieferant

**Fehlende Features (aus Recherche):**
- [ ] **Katalog-Funktionalitaet** -- Tab existiert als Platzhalter, keine Inhalte -- Aufwand: 2-3 Wochen
- [ ] **Belegkette: Bestellung -> Lieferschein -> Rechnung** -- Keine Verbindung zur Buchhaltung -- Aufwand: 2-3 Wochen (Teil von Synthese #4)
- [ ] **Lieferanten-Bewertung** -- Rating/Score basierend auf Lieferqualitaet/Termintreue -- Aufwand: 1 Woche
- [ ] **Rahmenvertraege** -- Abrufbestellungen aus Rahmenvertraegen -- Aufwand: 1-2 Wochen
- [ ] **Genehmigungsworkflow** -- Bestellungen ab Betrag X muessen genehmigt werden -- Aufwand: 1 Woche
- [ ] **Multi-Waehrung** -- Nur CHF, kein EUR fuer internationale Lieferanten -- Aufwand: 3-5 Tage
- [ ] **Inventar-Integration** -- Wareneingang automatisch Inventar-Bewegung erstellen -- Aufwand: 1 Woche

**Frontend-Aenderungen noetig:**
- Katalog-Tab mit Artikel-Suche und Schnellbestellung
- "Rechnung zuordnen"-Button im Bestellungs-Detail
- Bewertungs-Sterne/Score im Lieferanten-Detail
- Genehmigungs-Banner fuer Bestellungen ueber Schwellenwert
- Waehrungs-Auswahl im Bestellungs-Dialog

---

## 15. Fuhrpark

**Datei:** `modules/fuhrpark/FuhrparkPage.tsx` (1625)

**Aktueller Stand:** 4 Tabs: Fahrzeuge, Wartung, Tankprotokoll, Tracking. Fahrzeug-Karten mit Kennzeichen, Marke/Modell, Typ (PKW/Lieferwagen/LKW), Fahrer-Zuweisung, km-Stand, Pruefungs-/Versicherungsstatus mit Ampel (ueberfaellig/bald/ok). Wartungs-Tabelle mit Typ (Service/Reparatur/Pruefung/Reifen), Kosten, km-Stand. Tankprotokoll mit monatlichen Kosten/Liter-Zusammenfassungen, CHF/L-Berechnung. Tracking-Tab mit Status (unterwegs/geparkt/unbekannt), GPS-Positionen, Routenverlauf-Timeline, Tages-km, Live-Aktualisierung (simuliert). Dialoge fuer Fahrzeug/Wartung/Tanken hinzufuegen. Detail-Panel mit Kennzeichen-Anzeige, Gesamtkosten, letzte Wartungen/Tankungen.

**Gap-Level:** KLEIN

**Vorhandene Features:**
- Fahrzeug-Verwaltung (PKW/Van/LKW, CRUD)
- Wartungs-Tracking (Service/Reparatur/Pruefung/Reifen)
- Tankprotokoll mit Kosten/Verbrauch-Analyse
- GPS-Tracking mit Routenverlauf-Timeline
- Pruefungs-/Versicherungs-Erinnerungen (Ampel-System)
- Fahrer-Zuweisung
- Detail-Panel mit Kostenauswertung

**Fehlende Features (aus Recherche):**
- [ ] **Finanzamtkonformes Fahrtenbuch** -- Fuer 1%-Regelung vs. Fahrtenbuch-Methode (DE). Privat-/Dienstfahrt-Trennung -- Aufwand: 2-3 Wochen
- [ ] **Kosten-Reports / TCO** -- Total Cost of Ownership pro Fahrzeug (Wartung + Treibstoff + Versicherung + Abschreibung) -- Aufwand: 1 Woche
- [ ] **Dokumente pro Fahrzeug** -- Fahrzeugschein, Versicherungspolice, TUeV-Berichte -- Aufwand: 1 Woche
- [ ] **Schadensmeldung** -- Unfaelle/Schaeden dokumentieren mit Fotos -- Aufwand: 1 Woche
- [ ] **Reifenwechsel-Erinnerung** -- Saisonale Erinnerung (Sommer/Winter) -- Aufwand: 2-3 Tage

**Frontend-Aenderungen noetig:**
- Fahrtenbuch-Tab mit Privat/Dienst-Toggle pro Fahrt
- TCO-Dashboard pro Fahrzeug
- Dokumente-Sektion im Detail-Panel
- Schadensmeldungs-Dialog mit Foto-Upload
- Reifenwechsel-Banner (Oktober/April-Erinnerung)

---

## 16. Produktion

**Datei:** `modules/produktion/ProduktionPage.tsx` (1198)

**Aktueller Stand:** 3 Tabs: Auftraege, Stuecklisten, Qualitaet. Auftrags-Tabelle mit Auftrags-Nr, Produkt, Menge, Status (Geplant/In Produktion/Pausiert/Abgeschlossen/Storniert), Fortschrittsbalken, Start/Faelligkeit. Stuecklisten mit SKU, Version, Materialien-Tabelle (expandierbar). Qualitaets-Tab mit Pruefungs-Tabelle (Datum, Auftrags-Nr, Pruefer, Bestanden/Nicht bestanden, Maengel, Notizen). Auftrags-Detail-Panel mit grossem Fortschrittsbalken, Laufzeit-Visualisierung, verlinkte Stueckliste, Qualitaetspruefungen, Status-Aenderung. Dialoge fuer Auftrag/Stueckliste/Qualitaetspruefung erstellen.

**Gap-Level:** KLEIN

**Vorhandene Features:**
- Fertigungsauftraege mit vollstaendigem Lifecycle
- Stuecklisten (BOM) mit Materialien und Versionen
- Qualitaetspruefungen (Bestanden/Nicht bestanden, Maengel-Zaehlung)
- Fortschrittsbalken und Zeitlinie
- Status-Aenderung (Start/Pause/Resume)
- Verknuepfung Auftrag <-> Stueckliste <-> Qualitaetspruefung

**Fehlende Features (aus Recherche):**
- [ ] **Materialverfuegbarkeits-Pruefung** -- Vor Auftragsstart pruefen ob alle Materialien auf Lager -- Aufwand: 1-2 Wochen (Inventar-Integration)
- [ ] **Arbeitsplaene / Arbeitsgaenge** -- Einzelne Schritte innerhalb eines Fertigungsauftrags -- Aufwand: 2-3 Wochen
- [ ] **Maschinenbelegung** -- Welche Maschine ist wann belegt -- Aufwand: 1-2 Wochen
- [ ] **Ausschuss-Tracking** -- Ausschussrate pro Auftrag/Produkt -- Aufwand: 3-5 Tage

**Frontend-Aenderungen noetig:**
- Materialverfuegbarkeits-Ampel beim Auftragsstart
- Arbeitsgaenge-Sub-Tab im Auftragsdetail
- Maschinenbelegungs-Kalender (Gantt-aehnlich)

---

## 17. Berichte

**Datei:** `modules/berichte/BerichtePage.tsx` (922)

**Aktueller Stand:** 3 Tabs: Dashboard, Erstellen, Geplant. Dashboard mit Modul-Filter, KPI-Karten mit Veraenderungs-% und Gut/Schlecht-Indikatoren, Balkendiagramme (Umsatzverlauf, Tickets nach Prioritaet). Erstellen-Tab mit Report-Name, Modul-Auswahl, Datumsbereich, Format (PDF/Excel), Vorschau-Bereich, Generieren-Button mit Ladeanimation. Gespeicherte Berichte in Sidebar. Geplant-Tab mit geplanten Berichten-Tabelle (Name, Intervall daily/weekly/monthly, Empfaenger, letzter/naechster Lauf, Aktiv-Toggle). Neuer-geplanter-Bericht-Dialog mit Bericht-Auswahl, Intervall-Radio, E-Mail-Empfaenger mit Hinzufuegen/Entfernen.

**Gap-Level:** MITTEL

**Vorhandene Features:**
- KPI-Dashboard mit Modul-Filter
- Berichts-Generator (Name, Modul, Datumsbereich, Format)
- PDF/Excel-Format-Auswahl
- Geplante Berichte mit Intervall und Empfaengern
- Gespeicherte Berichte-Sidebar

**Fehlende Features (aus Recherche):**
- [ ] **Echte Daten-Anbindung** -- Aktuell Mock-Daten. Muss aus Backend-APIs ziehen -- Aufwand: 2-3 Wochen (Backend)
- [ ] **Benutzerdefinierte Dashboards** -- Eigene KPI-Zusammenstellungen erstellen -- Aufwand: 2-3 Wochen
- [ ] **Drill-Down** -- Klick auf KPI zeigt Detail-Daten -- Aufwand: 1-2 Wochen
- [ ] **DATEV-Auswertungen** -- Steuerberater-konforme Reports -- Aufwand: 1 Woche (Teil von DATEV-Export)
- [ ] **Vergleichsberichte** -- Zeitraum-Vergleich (Q1 vs. Q2, Jahr ueber Jahr) -- Aufwand: 1-2 Wochen
- [ ] **Export: tatsaechliche PDF/Excel-Generierung** -- Button existiert, aber keine echte Datei-Generierung -- Aufwand: 1-2 Wochen (Backend)

**Frontend-Aenderungen noetig:**
- API-Hooks fuer echte Daten (React Query)
- Dashboard-Editor (Widget-Auswahl, Drag-and-Drop)
- Drill-Down-Links auf KPI-Karten
- Vergleichs-Toggle (vs. Vorperiode/Vorjahr)

---

## 18. Formulare

**Datei:** `modules/formulare/FormularePage.tsx` (1494)

**Aktueller Stand:** 3 Tabs: Formulare, Eingaenge, Vorlagen. Stats: aktive Formulare, woechentliche Einreichungen, Abschlussrate. Formular-Karten mit Name, Beschreibung, Status, Feld-Anzahl, Einreichungs-Anzahl. Editor-Ansicht: Name/Beschreibung, zwei-Spalten-Layout (Feld-Liste mit Drag-Handle + Live-Vorschau). 9 Feld-Typen: Text, Textarea, Select, Radio, Checkbox, Datum, Zahl, Bewertung (Sterne), Datei-Upload. Feld-Konfigurations-Dialog (Label, Pflichtfeld-Toggle, Platzhalter, Optionen fuer Select/Radio). Einreichungen gruppiert nach Formular, Detail-Panel mit Antworten pro Feld-Typ. Vorlagen mit Feld-Typ-Badges, "Vorlage verwenden"-Button. Teilen-Dialog (Link kopieren + E-Mail-Versand). Neues-Formular-Dialog.

**Gap-Level:** KLEIN

**Vorhandene Features:**
- Formular-Builder mit 9 Feld-Typen
- Live-Vorschau
- Drag-Handle (UI, Reihenfolge aenderbar)
- Einreichungs-Verwaltung
- Vorlagen-System
- Teilen per Link/E-Mail
- Stats (aktive Formulare, Einreichungen, Abschlussrate)

**Fehlende Features (aus Recherche):**
- [ ] **Bedingte Logik** -- Felder ein-/ausblenden basierend auf vorherigen Antworten -- Aufwand: 1-2 Wochen
- [ ] **Mehrseitige Formulare** -- Formulare mit mehreren Seiten/Abschnitten -- Aufwand: 1 Woche
- [ ] **Automatische Aktionen** -- Bei Einreichung: E-Mail senden, Task erstellen, CRM-Kontakt anlegen -- Aufwand: 1-2 Wochen
- [ ] **Oeffentlicher Zugang** -- Formulare ohne Login ausfuellbar (fuer Kunden) -- Aufwand: 1-2 Wochen (Auth-Ausnahme)
- [ ] **Einreichungs-Export** (CSV/Excel) -- Keine Export-Funktion sichtbar -- Aufwand: 3-5 Tage

**Frontend-Aenderungen noetig:**
- Bedingte-Logik-Editor pro Feld (wenn Feld X = Y, zeige Feld Z)
- Seiten-/Abschnitts-Trenner im Builder
- Aktions-Konfiguration im Formular-Einstellungen-Panel
- Export-Button in Eingaenge-Tab

---

## 19. Vermietung

**Datei:** `modules/vermietung/VermietungPage.tsx` (1412)

**Aktueller Stand:** 3 Tabs: Objekte, Reservierungen, Kalender. 4 Objekt-Typen (Geraet/Raum/Fahrzeug/Werkzeug). Objekt-Karten mit Typ, Standort, Status (Verfuegbar/Reserviert/Wartung), Tagessatz/Wochensatz (CHF), Seriennummer, naechste Reservierung. Reservierungen-Tabelle mit Objekt, Mieter (Mitarbeiter/Kunde), Zeitraum, Dauer (Tage), Status (Aktiv/Bevorstehend/Abgeschlossen/Storniert), Abholung/Rueckgabe-Orte, Stornieren-Aktion. Wochen-Kalender-Grid (Objekte x Tage) mit Reservierungs-Bloecken (zusammenhaengend ueber mehrere Tage), Wartungs-Markierung, Hover-to-Add, Legende. KPIs: Verfuegbar, Reserviert, In Wartung, Auslastung %. Detail-Panel mit Preisen, Status, letzte Reservierungen. Objekt-Dialog, Reservierungs-Dialog (mit Mieter-Typ, Abhol-/Rueckgabeort).

**Gap-Level:** KLEIN

**Vorhandene Features:**
- Objekt-Verwaltung mit 4 Typen, CRUD
- Reservierungen mit Zeitraum, Mieter-Typ, Standorte
- Wochen-Kalender-Grid mit visuellen Reservierungsbalken
- Auslastungs-KPI
- Konflikterkennung (Klick auf reservierten Slot zeigt Info)
- Detail-Panel, Stornieren

**Fehlende Features (aus Recherche):**
- [ ] **Automatische Preisberechnung** -- Reservierung erstellen zeigt keinen Gesamtpreis (Tage x Tagessatz) -- Aufwand: 2-3 Tage
- [ ] **Kautionsverwaltung** -- Kaution einziehen/zurueckgeben bei Vermietung an Kunden -- Aufwand: 1 Woche
- [ ] **Schadens-/Zustandsprotokoll** -- Zustand bei Abholung und Rueckgabe dokumentieren -- Aufwand: 1-2 Wochen
- [ ] **Verfuegbarkeits-Kalender (oeffentlich)** -- Kunden koennen Verfuegbarkeit einsehen und reservieren -- Aufwand: 2-3 Wochen
- [ ] **Inventar-Integration** -- Vermietbare Geraete aus Inventar verlinken -- Aufwand: 1 Woche
- [ ] **Multi-Waehrung** -- Nur CHF -- Aufwand: 2-3 Tage

**Frontend-Aenderungen noetig:**
- Gesamtpreis-Berechnung im Reservierungs-Dialog anzeigen
- Kautions-Feld im Objekt-Dialog
- Zustandsprotokoll-Formular bei Abholung/Rueckgabe
- Preis-Zusammenfassung im Reservierungs-Bestaetigungs-Dialog

---

## 20. Vertraege

**Datei:** `modules/vertraege/VertraegePage.tsx` (1234)

**Aktueller Stand:** 3 Tab-Filter: Aktiv, Auslaufend (90 Tage), Archiv. 6 Vertragstypen (Mietvertrag/Liefervertrag/Servicevertrag/Arbeitsvertrag/Lizenz/Versicherung). Vertragstabelle mit Vertragsnr., Titel, Partner, Typ-Badge, Laufzeit-Balken (mit Kuendigungsfrist-Markierung), monatliche Kosten (CHF), Status (Aktiv/Auslaufend/Gekuendigt/Abgelaufen). Stats: aktive Vertraege, monatliche Gesamtkosten, auslaufend (30 Tage), Gesamtvertragswert. Detail-Panel mit Vertragsdetails, Laufzeit-Visualisierung mit Fortschrittsbalken, Kuendigungsfrist-Marker, Wert-Anzeige (monatlich/gesamt), Konditionen (Verlaengerung auto/manual, Kuendigungsfrist), naechste-Aktion-Hinweis, Aenderungshistorie-Timeline. Kuendigungs-Dialog mit Datum, Grund, Bestaetigungs-Checkbox. Suche, Typ-Filter.

**Gap-Level:** KLEIN

**Vorhandene Features:**
- Vollstaendige Vertragsverwaltung (CRUD, 6 Typen)
- Laufzeit-Visualisierung mit Fortschrittsbalken
- Kuendigungsfrist-Tracking mit Deadline-Anzeige
- Automatische vs. manuelle Verlaengerung
- Kuendigungs-Workflow (Datum, Grund, Bestaetigung)
- Aenderungshistorie
- Kosten-Tracking (monatlich, Gesamtwert)
- Auslaufend-Warnungen (30/90 Tage)

**Fehlende Features (aus Recherche):**
- [ ] **E-Signatur (Skribble-Integration)** -- Vertraege digital unterschreiben -- Aufwand: 2-3 Wochen (Synthese, REST API)
- [ ] **Dokument-Verknuepfung** -- Vertrags-PDF aus Dokumente-Modul verlinken (documentRef existiert aber nicht editierbar) -- Aufwand: 3-5 Tage
- [ ] **Erinnerungen/Benachrichtigungen** -- Automatische Erinnerung vor Kuendigungsfrist -- Aufwand: 1 Woche
- [ ] **Vertrags-Vorlagen** -- Standardvertraege als Vorlage speichern -- Aufwand: 1 Woche
- [ ] **Multi-Waehrung** -- Nur CHF -- Aufwand: 2-3 Tage

**Frontend-Aenderungen noetig:**
- Skribble-Widget im Detail-Panel (Unterschreiben-Button)
- Dokument-Picker fuer Vertrags-PDF
- Erinnerungs-Konfiguration im Vertrags-Dialog
- Vorlagen-Tab

---

## 21. Rapporte

**Datei:** `modules/rapporte/RapportePage.tsx` (1471)

**Aktueller Stand:** 3 Tabs: Tagesberichte, Aufmass, Vorlagen. Stats: Berichte diese Woche, Arbeitsstunden gesamt, Material-Kosten (Mock), aktive Projekte. Tagesberichte: Karten mit Datums-Block, Projekt-Badge, Autor, Wetter-Icon + Temperatur, Arbeitszeit (Start-Ende, Netto-Stunden), Mitarbeiter-Anzahl, Foto-Anzahl, Unterschrift-Status (ausstehend/unterschrieben). Detail-Panel mit Wetter-Block, Arbeitszeit-Grid (4 Spalten: Start/Ende/Pause/Netto), Mitarbeiter-Tabelle (Name/Funktion/Stunden), Taetigkeiten mit Kategorie, Material-Tabelle (Artikel/Menge/Einheit), Foto-Platzhalter, Unterschrift-Platzhalter ("kommt bald"), Notizen. Aufmass-Tab mit Positionen-Tabelle (L/B/H/Flaeche/Volumen), Summen, Skizze-Platzhalter ("kommt bald"). Vorlagen mit Standard-Taetigkeiten und -Materialien, "Vorlage verwenden"-Button. Neuer-Tagesbericht-Dialog (Datum, Projekt, Wetter, Temperatur, Arbeitszeit, Mitarbeiter-Liste mit dynamischem Hinzufuegen/Entfernen, Taetigkeiten, Material mit 10 Einheiten, Notizen). Neues-Aufmass-Dialog (Name, Projekt, Positionen mit L/B/H, Live-Berechnung).

**Gap-Level:** KLEIN

**Vorhandene Features:**
- Vollstaendige Tagesberichte (Wetter, Arbeitszeit, Mitarbeiter, Taetigkeiten, Material, Fotos, Notizen)
- Aufmass mit Positionen-Tabelle und automatischer Flaechen-/Volumen-Berechnung
- Vorlagen-System
- Projekt-Filter, Datums-Filter (Woche/Monat/Alle)
- Detail-Panel mit umfassenden Informationen
- Unterschrift-Status-Tracking

**Fehlende Features (aus Recherche):**
- [ ] **Digitale Unterschrift** -- "Kommt bald"-Platzhalter. E-Signatur auf dem Geraet (Touch/Stift) -- Aufwand: 1-2 Wochen
- [ ] **Foto-Upload** -- Foto-Sektion zeigt Platzhalter, kein echter Upload -- Aufwand: 1-2 Wochen (Backend Storage)
- [ ] **Aufmass-Zeichnung/Skizze** -- "Kommt bald"-Platzhalter. Canvas-basierte Skizze -- Aufwand: 3-4 Wochen (komplex)
- [ ] **PDF-Export Tagesbericht** -- Druckfaehiger Rapport als PDF -- Aufwand: 1-2 Wochen
- [ ] **Genehmigungs-Workflow** -- Bauleiter genehmigt Tagesbericht -- Aufwand: 1 Woche
- [ ] **Wetter-API-Integration** -- Automatisch Wetter fuer Standort/Datum abfragen -- Aufwand: 3-5 Tage

**Frontend-Aenderungen noetig:**
- Signatur-Canvas-Komponente (Touch-Input)
- Foto-Upload mit Vorschau (Electron File API)
- Canvas-Skizzen-Editor fuer Aufmass
- PDF-Export-Button mit Layout-Vorlage
- Genehmigungs-Banner und -Workflow

---

## 22. Dashboard

**Datei:** `modules/dashboard/DashboardPage.tsx` (95)

**Aktueller Stand:** Zeitbasierte Begruessung (Guten Morgen/Tag/Abend), Alerts-Sektion, Modul-Grid, anpassbare Widgets mit Bearbeitungsmodus (Bearbeiten/Fertig/Zuruecksetzen).

**Gap-Level:** KLEIN

**Vorhandene Features:**
- Zeitbasierte Begruessung
- Alerts-Sektion
- Module-Grid
- Widget-Container mit Bearbeitungsmodus
- Zuruecksetzen auf Standard

**Fehlende Features (aus Recherche):**
- [ ] **Personalisierte Widgets** -- Branchenspezifische Widget-Vorschlaege basierend auf Business-Profil -- Aufwand: 1 Woche
- [ ] **KPI-Widgets mit Echtdaten** -- Aktuell vermutlich Mock-Daten -- Aufwand: 1-2 Wochen (Backend-Anbindung)
- [ ] **Quick Actions** -- Schnellzugriff-Buttons (Neuer Kontakt, Neue Rechnung, Timer starten) -- Aufwand: 3-5 Tage
- [ ] **Benachrichtigungs-Feed** -- Zentrale Benachrichtigungs-Timeline -- Aufwand: 1-2 Wochen

**Frontend-Aenderungen noetig:**
- Widget-Auswahl basierend auf aktivem Business-Profil
- API-Hooks fuer KPI-Widgets
- Quick-Actions-Leiste unter der Begruessung
- Benachrichtigungs-Timeline-Widget

---

## 23. Chat

**Hinweis:** Chat wurde von Luke (Backend-Dev) gebaut und ist backend-verbunden. Kein separates Modul-File in `modules/` gelesen (vermutlich in eigener Route).

**Gap-Level:** MITTEL (geschaetzt)

**Vorhandene Features (aus Synthese):**
- 1:1 Chat
- Gruppen/Channels

**Fehlende Features (aus Recherche):**
- [ ] **Video/Audio (LiveKit)** -- Geplant aber nicht implementiert -- Aufwand: 3-4 Wochen
- [ ] **Datei-Teilen in Chat** -- Bilder, Dokumente direkt im Chat senden -- Aufwand: 1-2 Wochen
- [ ] **Teams/Slack Bridge** -- Chat-Nachrichten zwischen KMU Hub und Teams/Slack -- Aufwand: 3-4 Wochen (Synthese, optional)
- [ ] **Emoji-Reaktionen** -- Nachrichten mit Emoji reagieren -- Aufwand: 3-5 Tage
- [ ] **Thread-Antworten** -- Separate Konversations-Threads pro Nachricht -- Aufwand: 1-2 Wochen
- [ ] **@Mentions** -- Nutzer erwaehnen mit Benachrichtigung -- Aufwand: 1 Woche

---

## Zusammenfassung

### Gap-Level Verteilung

| Gap-Level | Module | Anzahl |
|-----------|--------|--------|
| **GROSS** | CRM (Backend), E-Mail, Buchhaltung | 3 |
| **MITTEL** | Kontakte, Meetings, Work/Projekte, Zeiterfassung, Helpdesk, Team/HR, Dokumente, Schichtplanung, Inventar, Einkauf, Berichte, Chat | 12 |
| **KLEIN** | Kalender, Fuhrpark, Produktion, Formulare, Vermietung, Vertraege, Rapporte, Dashboard | 8 |

### Top 10 Frontend-Prioritaeten (nach Business Impact)

| # | Aenderung | Betroffene Module | Synthese-Ref | Aufwand |
|---|-----------|-------------------|--------------|---------|
| 1 | **API-Layer fuer IMAP/SMTP** (React Query Hooks fuer E-Mail-Backend) | E-Mail | #1 | 2-3 Wo. (FE) |
| 2 | **MWSt multi-country + formatCurrency** (`finance.ts` umbauen) | Buchhaltung, Einkauf, Inventar, Vermietung, Vertraege | #8 | 2-3 Tage |
| 3 | **TipTap Rich-Text-Editor** (Wiki, Helpdesk-KB, E-Mail Compose) | Dokumente, Helpdesk, E-Mail | #6 | 2-3 Wo. |
| 4 | **CRM CRUD-Formulare** (Create/Edit fuer Kontakte, Firmen, Deals) | CRM | -- | 2-3 Wo. |
| 5 | **Belegkette UI** (Angebot -> Rechnung Button, Status-Flow) | Buchhaltung, Einkauf | #4 | 1-2 Wo. (FE) |
| 6 | **DATEV-Export UI** (Export-Button + Format-Auswahl) | Buchhaltung, Zeiterfassung, Berichte | #2 | 3-5 Tage (FE) |
| 7 | **Canned Responses** (Textbaustein-Auswahl im Reply-Bereich) | Helpdesk | #11 | 3-5 Tage |
| 8 | **Custom Fields Editor** (Feld-Typ-Auswahl, CRUD fuer benutzerdefinierte Felder) | CRM, Helpdesk, Projekte | #3 | 2-3 Wo. |
| 9 | **PDF-Vorschau/Generierung UI** (Vorschau-Panel, Download-Button) | Buchhaltung, Rapporte, Berichte | #9 | 1 Wo. (FE) |
| 10 | **Akadem. Titel + Anrede** (Titel-Dropdown im Kontakt-Formular) | CRM, Kontakte | #10 | 2-3 Tage |

### Querschnitts-Themen (betreffen mehrere Module)

| Thema | Betroffene Module | Status |
|-------|-------------------|--------|
| **Nur CHF-Waehrung** | Buchhaltung, Einkauf, Inventar, Vermietung, Vertraege, Fuhrpark, Rapporte | `finance.ts` hat nur `formatCHF`, `Intl.NumberFormat('de-CH', {currency: 'CHF'})` |
| **Kein Backend** (Zustand-only) | Kontakte, E-Mail, Meetings, Helpdesk, Zeiterfassung, Schichten, Inventar, Einkauf, Fuhrpark, Produktion, Berichte, Formulare, Vermietung, Vertraege, Rapporte | Nur CRM und Work/Projekte nutzen React Query API-Hooks |
| **Kein Rich-Text-Editor** | Dokumente (Wiki), Helpdesk (KB), E-Mail (Compose) | Alle nutzen `<textarea>` statt TipTap |
| **Kein PDF-Export** | Buchhaltung, Rapporte, Berichte, Schichten | Export-Buttons existieren teilweise, generieren aber keine echten Dateien |
| **Kontakt-Dualitaet** | CRM (Backend) vs. Kontakte (Zustand) | Zwei separate Kontakt-Systeme die zusammengefuehrt werden muessen |

### Naechste Schritte (Frontend-Perspektive, Darien)

**Sofort machbar (keine Backend-Abhaengigkeit):**
1. MWSt multi-country in `finance.ts` (2-3 Tage)
2. Akadem. Titel + Anrede in Kontakt-Formularen (2-3 Tage)
3. Canned Responses UI im Helpdesk (3-5 Tage)
4. CRM CRUD-Formulare als UI-Shell (2-3 Wochen)

**Backend-abhaengig (Koordination mit Luke noetig):**
1. IMAP/SMTP Frontend-Integration (nach Backend-Implementierung)
2. TipTap-Editor (kann Frontend-seitig starten, Backend fuer Speicherung noetig)
3. Custom Fields Editor (Backend-Schema muss zuerst stehen)
4. Belegkette (Backend-Logik fuer Konvertierung noetig)

---

*Dieses Dokument basiert auf der Code-Analyse aller Module-Dateien in `desktop/src/renderer/src/modules/` und der Marktrecherche-Synthese `00-SYNTHESE.md`. Alle Aufwandsschaetzungen beziehen sich auf Frontend-Aufwand sofern nicht anders angegeben.*
