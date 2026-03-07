# Kostenanalyse & Preismodell: KMU Hub

**Datum:** 2026-02-17
**Version:** 1.0
**Zweck:** Detaillierte Kosten-pro-Modul-Analyse, Paketstruktur, Margenkalkulation und Preisempfehlung
**Quellen:** `09-infrastruktur-matrix.md`, `10-integrations-guide.md`, `00-SYNTHESE.md`, `chatgpt-research-summary.md`
**Confidence:** MEDIUM (Preise basieren auf Q1 2025 Trainingsdaten -- vor Einsatz live verifizieren)

---

## Inhaltsverzeichnis

1. [Kosten pro Modul](#1-kosten-pro-modul)
2. [Grundpaket (Base Package)](#2-grundpaket)
3. [Paket-Vorschlaege (Pricing Tiers)](#3-paket-vorschlaege)
4. [Add-Ons (Einzeln zubuchbar)](#4-add-ons)
5. [Wettbewerber-Preisvergleich](#5-wettbewerber-preisvergleich)
6. [Kostenrechnung pro Kundengroesse](#6-kostenrechnung-pro-kundengroesse)
7. [Break-Even Analyse](#7-break-even-analyse)
8. [Branchenspezifische Pakete](#8-branchenspezifische-pakete)
9. [Einrichtungskosten (Onsite-Woche)](#9-einrichtungskosten)
10. [Zusammenfassung: Empfohlenes Preismodell](#10-zusammenfassung)

---

## 1. Kosten pro Modul

### Grundlagen der Kalkulation

**SaaS-Infrastruktur-Referenzwerte (aus `09-infrastruktur-matrix.md`):**

| Kennzahl | Wert |
|----------|------|
| Basis-Infrastruktur fuer 100 Kunden (2.000 User) | ~200 EUR/Mo |
| Pro Kunde (Durchschnitt 20 User) | ~2,00 EUR/Mo |
| Pro User (SaaS, multi-tenant) | ~0,10 EUR/Mo |
| OnlyOffice Developer-Lizenz | ~4.000 EUR/Jahr = ~333 EUR/Mo |
| LiveKit Server (Hetzner CX32) | ~10 EUR/Mo (dediziert) |
| Hetzner Object Storage | ~0,0126 EUR/GB/Mo |
| Hetzner Bandbreite | 20 TB/Server inkludiert, Overage ~1,19 EUR/TB |

**Schluessel-Erkenntnis:** Bei Multi-Tenant-SaaS mit RLS sind die reinen Infrastrukturkosten pro Kunde vernachlaessigbar. Die relevanten variablen Kosten entstehen durch **externe Services**, **Storage** und **Bandbreite**.

---

### 1.1 Kostenlose Module (nur anteiliger Server-Aufwand)

Diese Module benoetigen keine externen Dienste -- nur CPU, RAM und minimale DB-Storage.

| Modul | DB-Wachstum/Monat (15 User) | CPU-Last | Externe Kosten | Kosten/Kunde/Mo |
|-------|------------------------------|----------|----------------|-----------------|
| **CRM / Kontakte** | ~5-20 MB | Niedrig | 0 | ~0,01 EUR |
| **Projekte (Kanban/Gantt)** | ~5-15 MB | Niedrig | 0 | ~0,01 EUR |
| **Kalender** | ~2-5 MB | Niedrig | 0 | ~0,005 EUR |
| **Zeiterfassung** | ~5-10 MB | Niedrig | 0 | ~0,01 EUR |
| **Helpdesk / Tickets** | ~10-30 MB | Niedrig | 0 | ~0,01 EUR |
| **Wiki** (ohne TipTap Pro) | ~5-20 MB (JSONB-Inhalte) | Niedrig | 0 | ~0,01 EUR |
| **Formulare** | ~2-5 MB | Sehr niedrig | 0 | ~0,005 EUR |
| **Vertraege** | ~2-5 MB | Sehr niedrig | 0 | ~0,005 EUR |
| **Rapporte** (ohne Fotos) | ~5-10 MB | Niedrig | 0 | ~0,01 EUR |
| **Berichte / Dashboard** | ~0 MB (Aggregation) | Mittel (Queries) | 0 | ~0,01 EUR |
| **Schichtplanung** | ~5-10 MB | Niedrig | 0 | ~0,01 EUR |
| **Chat** (nur Text) | ~10-50 MB | Mittel (WebSocket) | 0 | ~0,02 EUR |

**Summe Basis-Module:** ~0,12 EUR/Kunde/Monat

**Rechnung:**
- DB-Wachstum: ~60-200 MB/Monat = ~2,4 GB/Jahr
- Hetzner PostgreSQL-Anteil: vernachlaessigbar bei Multi-Tenant
- Redis-Anteil (Sessions, Cache, Pub/Sub): vernachlaessigbar
- **Diese Module sind de facto gratis** -- die Infrastrukturkosten werden ohnehin durch die Basis-Server abgedeckt.

---

### 1.2 Module mit geringen Zusatzkosten

Diese Module erzeugen etwas mehr DB-Storage oder benoetigen gelegentlich mehr CPU, haben aber keine externen Kosten.

| Modul | DB-Wachstum/Monat (15 User) | Besonderheit | Kosten/Kunde/Mo |
|-------|------------------------------|-------------|-----------------|
| **Inventar** | ~10-30 MB (Artikelbilder klein) | Artikelbilder: ~1-3 GB insgesamt | ~0,04 EUR |
| **Einkauf** | ~10-20 MB | Belegdaten, PDFs | ~0,02 EUR |
| **Fuhrpark** | ~5-15 MB | Wartungsdaten, Tankbelege | ~0,02 EUR |
| **Produktion** | ~10-20 MB | Stuecklisten, Fertigungsauftraege | ~0,02 EUR |
| **Vermietung** | ~5-10 MB | Reservierungen | ~0,01 EUR |
| **HR / Team** | ~10-30 MB | Personaldaten, Dokumente | ~0,03 EUR |

**Summe Module mit geringen Zusatzkosten:** ~0,14 EUR/Kunde/Monat

**Rechnung:**
- Zusaetzlicher Storage: ~50-125 MB/Monat
- Object Storage fuer Bilder/Belege: ~500 MB/Monat = ~0,006 EUR/Mo (Hetzner S3)
- **Ebenfalls de facto gratis** im SaaS-Betrieb

---

### 1.3 Module mit signifikanten Zusatzkosten

Hier entstehen reale variable Kosten durch externe Services, Bandbreite oder signifikanten Storage.

#### E-Mail (IMAP/SMTP-Client)

KMU Hub baut einen IMAP-Client -- keinen eigenen Mailserver. Die Kosten haengen davon ab, ob wir Mail-Storage anbieten oder nur als Client fungieren.

**Szenario A: Nur IMAP-Client (empfohlen fuer v1)**
- KMU Hub verbindet sich mit dem bestehenden Mailserver des Kunden (Exchange, Gmail, Hostpoint, etc.)
- **Kosten fuer uns: ~0 EUR** (nur CPU fuer IMAP-Sync)
- IMAP-Sync-Last: ~0,5-1 vCPU pro 50 aktive Mailboxen
- **Pro Kunde: ~0,05 EUR/Mo** (anteiliger Server-Aufwand)

**Szenario B: Eigener Mail-Storage (Cache/Mirror)**
Wenn wir E-Mails lokal zwischenspeichern fuer schnellere Suche:

| Speichervolumen | Kosten (Hetzner S3) | Kosten/Kunde/Mo |
|-----------------|--------------------|-----------------|
| 5 GB pro Mailbox, 15 Mailboxen = 75 GB | 75 x 0,0126 = 0,95 EUR | ~0,95 EUR |
| 50 GB pro Mailbox, 15 Mailboxen = 750 GB | 750 x 0,0126 = 9,45 EUR | ~9,45 EUR |

**Empfehlung:** Szenario A (Client-only). Kein eigener Mail-Storage. E-Mails bleiben beim Provider des Kunden. Wir cachen nur Metadaten + letzte 30 Tage im RAM/Redis.

**Kosten: ~0,05 EUR/Kunde/Mo**

---

#### Meetings / Video (LiveKit Self-Hosted)

**Bandbreiten-Kalkulation (aus `09-infrastruktur-matrix.md`):**

| Parameter | Wert |
|-----------|------|
| Video-Stream (720p) | ~1,5-2 Mbit/s pro Richtung |
| Audio-Stream | ~50-100 Kbit/s pro Richtung |
| Traffic pro Stunde/User (Video 720p bidirektional) | ~700 MB |
| Hetzner inkludiert | 20 TB/Mo pro Server |
| Hetzner Overage | ~1,19 EUR/TB |

**Was kostet ein 10-Personen-Meeting (1 Stunde)?**

```
Server-Bandwidth: 10 Sender x 1,75 Mbit/s = 17,5 Mbit/s Upload
                  10 Empfaenger x 9 Streams x 1,75 Mbit/s = 157,5 Mbit/s Download
                  = ~175 Mbit/s Peak am Server

Traffic:          10 User x 700 MB = 7 GB pro Meeting-Stunde
Kosten:           Bei 20 TB inkludiert = 0 EUR
                  Bei Overage: 7 GB / 1.000 x 1,19 EUR = ~0,008 EUR
```

**Kosten: ~0,008 EUR pro 10-Personen-Meeting-Stunde** (bei inkludiertem Traffic: 0 EUR)

**LiveKit Recording-Storage (optional):**
- 1 Stunde 720p Recording: ~500 MB - 1 GB
- Bei 10 Recordings/Monat pro Kunde: ~5-10 GB
- Kosten: 10 GB x 0,0126 = ~0,13 EUR/Mo

**Monatliche Kalkulation pro Kunde (15 User, typische Nutzung):**

| Nutzungs-Profil | Meetings/Mo | Traffic/Mo | Kosten/Mo |
|-----------------|-------------|------------|-----------|
| Wenig (Handwerk) | 5 Calls a 3 TN, 15 Min | ~2 GB | ~0 EUR |
| Normal (Dienstleister) | 20 Calls a 5 TN, 30 Min | ~35 GB | ~0 EUR |
| Viel (Agentur) | 60 Calls a 8 TN, 45 Min | ~250 GB | ~0 EUR |
| Power (Consulting) | 200 Calls a 10 TN, 60 Min | ~1,4 TB | ~0 EUR* |

*Bei 4 LiveKit-Servern = 80 TB inkludiert -- selbst Power-User verursachen keine Overage-Kosten bei 100 Kunden.

**Server-Kosten (anteilig):**
- LiveKit auf dedizierten CX32-Servern: ~10 EUR/Mo pro Server
- Bei 100 Kunden, 2 Server: 20 EUR = 0,20 EUR/Kunde
- Bei 500 Kunden, 3 Server: 30 EUR = 0,06 EUR/Kunde

**Gesamt-Kosten Video: ~0,06-0,20 EUR/Kunde/Mo** (Server-Anteil, Traffic inkludiert)

---

#### Dokumente + Storage (S3/MinIO)

**Storage-Bedarf nach Branche (pro Jahr, 15 MA, aus `09-infrastruktur-matrix.md`):**

```
Bau:            100-300 GB/Jahr (Fotos, Plaene, Rapporte)
Handwerk:        30-80 GB/Jahr  (Rapporte, Belege)
Dienstleister:   25-60 GB/Jahr  (Vertraege, Angebote)
Handel:          10-30 GB/Jahr  (Artikelbilder, Belege)
Gastro:           5-15 GB/Jahr  (Belege, Lieferscheine)
```

**Kosten pro GB/Monat (Hetzner Object Storage): 0,0126 EUR**

| Storage-Volumen | Kosten/Mo |
|-----------------|-----------|
| 10 GB | 0,13 EUR |
| 50 GB | 0,63 EUR |
| 100 GB | 1,26 EUR |
| 250 GB | 3,15 EUR |
| 500 GB | 6,30 EUR |
| 1 TB | 12,60 EUR |

**Durchschnittskunde nach 1 Jahr:**

| Branche | Kumuliertes Volumen | Kosten/Mo |
|---------|--------------------|-----------|
| Gastro/Handel | ~10-30 GB | 0,13-0,38 EUR |
| Dienstleister | ~25-60 GB | 0,32-0,76 EUR |
| Handwerk | ~30-80 GB | 0,38-1,01 EUR |
| Bau | ~100-300 GB | 1,26-3,78 EUR |

**Durchschnitt (gewichtet, 80% Dienstleister/Handwerk, 20% Bau):** ~0,50-1,50 EUR/Kunde/Mo nach 1 Jahr Nutzung.

**ACHTUNG:** Storage waechst linear. Nach 3 Jahren sind Bau-Kunden bei 300-900 GB = 3,78-11,34 EUR/Mo. Storage-Limits in Paketen sind daher KRITISCH.

---

#### Office-Editing (OnlyOffice)

**Lizenzkosten (aus `10-integrations-guide.md`):**

| Edition | Kosten/Jahr | Fuer wen |
|---------|-------------|----------|
| Community (AGPL) | 0 EUR | Self-Hosted Kunden (max 20 Connections) |
| Developer (OEM/SaaS) | ~3.000-6.000 EUR/Jahr | **KMU Hub SaaS-Betrieb** |
| Enterprise (Self-Hosted) | ~1.200-4.800 EUR/Jahr | Self-Hosted Mittel/Gross |

**SaaS-Kalkulation (Developer-Lizenz = ~4.000 EUR/Jahr = ~333 EUR/Mo):**

| Kunden | Pro-Kunde-Kosten/Mo |
|--------|---------------------|
| 50 | 6,66 EUR |
| 100 | 3,33 EUR |
| 250 | 1,33 EUR |
| 500 | 0,67 EUR |
| 1.000 | 0,33 EUR |

**Server-Kosten zusaetzlich:**
- OnlyOffice braucht dedizierte Server: CX32 (8 GB RAM) = ~9,59 EUR/Mo
- Bei 100 Kunden: +0,10 EUR/Kunde
- Pro gleichzeitiges Dokument: ~100-200 MB RAM
- 2 Server fuer Redundanz: ~19,18 EUR/Mo

**Gesamt OnlyOffice:**

| Kunden | Lizenz + Server | Pro Kunde/Mo |
|--------|----------------|--------------|
| 100 | 333 + 19 = 352 EUR | ~3,52 EUR |
| 500 | 333 + 29 = 362 EUR | ~0,72 EUR |
| 1.000 | 333 + 39 = 372 EUR | ~0,37 EUR |

**Fazit:** OnlyOffice ist bei wenigen Kunden teuer (Fixkosten-dominiert), wird aber mit Skalierung guenstig. Erst ab ~200 Kunden sinkt der Pro-Kopf-Preis unter 2 EUR.

---

#### E-Signatur (Skribble)

**Preise (aus `10-integrations-guide.md`):**

| Signatur-Level | Kosten/Signatur |
|----------------|-----------------|
| EES (Einfach) | ~0,50-1,50 CHF |
| FES (Fortgeschritten) | ~1,50-2,50 CHF |
| QES (Qualifiziert) | ~2,50-4,00 CHF |

**Skribble Business-Account:** Ab 85 CHF/Mo (1 Nutzer, 600 Signaturen/Jahr inkludiert)

**Optionen fuer KMU Hub:**

**Option A: Pass-Through (empfohlen)**
- Kunde hat eigenen Skribble-Account
- KMU Hub integriert per API
- **Kosten fuer uns: 0 EUR** (Skribble verdient direkt am Kunden)

**Option B: Gebundelt (Aufpreis als Add-On)**
- KMU Hub kauft Signaturen im Bulk bei Skribble
- Verkauft an Kunden mit Aufschlag
- Braucht Skribble Enterprise-Vertrag

**Empfehlung:** Option A fuer v1, Option B spaeter als Premium-Feature.

**Kosten bei Option A: 0 EUR/Kunde/Mo**

---

#### Newsletter (Brevo)

**Preise (aus `10-integrations-guide.md`):**

| Plan | Kosten/Mo | E-Mails/Mo |
|------|-----------|-----------|
| Free | 0 EUR | 300/Tag (~9.000/Mo) |
| Starter | Ab 19 EUR | 20.000 |
| Business | Ab 49 EUR | 20.000 (ohne Branding) |

**Kalkulation:**

**Option A: Ein Brevo-Account fuer alle Kunden (SaaS)**
- Alle Kunden-Newsletter laufen ueber KMU Hub's Brevo-Account
- Problem: Deliverability-Risiko (ein Spammer = alle geblockt)
- Kosten abhaengig vom Gesamt-Volumen

| Kunden (Newsletter aktiv) | E-Mails/Mo (geschaetzt) | Brevo-Plan | Kosten/Mo | Pro Kunde |
|---------------------------|------------------------|------------|-----------|-----------|
| 20 | ~40.000 | Starter (40k) | ~29 EUR | ~1,45 EUR |
| 50 | ~100.000 | Business (100k) | ~69 EUR | ~1,38 EUR |
| 100 | ~250.000 | Business (250k) | ~149 EUR | ~1,49 EUR |
| 500 | ~1.000.000 | Enterprise | ~500 EUR | ~1,00 EUR |

**Option B: Kunden bringen eigenen Brevo/CleverReach-Account (empfohlen)**
- Jeder Kunde verbindet seinen eigenen Newsletter-Provider
- **Kosten fuer uns: 0 EUR**
- Bessere Deliverability (isolierte Sender-Reputation)

**Empfehlung:** Option B. Kosten: 0 EUR. Wir stellen nur die Integration bereit.

---

#### Banking (FinAPI)

**Preise (aus `10-integrations-guide.md`):**

| Kostenart | Betrag |
|-----------|--------|
| Setup-Gebuehr | 0-5.000 EUR (einmalig) |
| Monatliche Grundgebuehr | Ab ~200-500 EUR/Mo |
| Pro Bank-Verbindung | ~0,50-2,00 EUR/Mo |
| Pro Transaktion | ~0,01-0,05 EUR |

**Kalkulation (bei Flatrate-Modell ~500 EUR/Mo):**

| Kunden (Banking aktiv) | Grundgebuehr | Pro Kunde/Mo |
|------------------------|--------------|--------------|
| 10 | 500 EUR | 50,00 EUR |
| 50 | 500 EUR | 10,00 EUR |
| 100 | 750 EUR (Volumen) | 7,50 EUR |
| 500 | 1.500 EUR | 3,00 EUR |

**ACHTUNG:** FinAPI ist teuer bei wenigen Kunden. Erst ab ~100 aktiven Banking-Nutzern wirtschaftlich sinnvoll.

**Alternative: Kunden verbinden eigene FinAPI/Bank-API**
- Unwahrscheinlich -- FinAPI ist ein B2B-Dienst, kein End-User-Produkt

**Empfehlung:** Banking als Premium-Add-On positionieren. Erst aktivieren ab 50+ Kunden die es wollen. Preis: 10-15 EUR/Kunde/Mo als Add-On.

---

#### DATEV-Export

- Reiner CSV-Export, keine externe API
- **Kosten: 0 EUR** (nur CPU fuer Generierung, vernachlaessigbar)

#### Bexio-Sync

- API-Zugang kostenlos (Bexio verdient am Kunden-Abo)
- **Kosten: 0 EUR**

#### Swiss QR-Code / ZUGFeRD

- Offene Standards, lokal generiert
- ZUGFeRD braucht evtl. unipdf Lizenz: ~500 EUR/Jahr = ~42 EUR/Mo
- Bei 100 Kunden: ~0,42 EUR/Kunde
- Bei 500 Kunden: ~0,08 EUR/Kunde
- **Kosten: ~0,08-0,42 EUR/Kunde/Mo**

#### TipTap (Rich-Text Editor)

- Open Source (MIT), kein externer Dienst
- **Kosten: 0 EUR**
- Pro-Version erst spaeter (Collaboration): ~29-99 EUR/Mo

---

### 1.4 Kosten-Uebersicht pro Modul (Zusammenfassung)

| Modul | Kosten-Kategorie | Kosten/Kunde/Mo (bei 100 Kunden) | Kosten/Kunde/Mo (bei 500 Kunden) |
|-------|-----------------|----------------------------------|----------------------------------|
| CRM, Kontakte | Gratis | ~0,01 | ~0,01 |
| Projekte | Gratis | ~0,01 | ~0,01 |
| Kalender | Gratis | ~0,005 | ~0,005 |
| Zeiterfassung | Gratis | ~0,01 | ~0,01 |
| Helpdesk | Gratis | ~0,01 | ~0,01 |
| Wiki | Gratis | ~0,01 | ~0,01 |
| Formulare | Gratis | ~0,005 | ~0,005 |
| Vertraege | Gratis | ~0,005 | ~0,005 |
| Rapporte (Text) | Gratis | ~0,01 | ~0,01 |
| Berichte | Gratis | ~0,01 | ~0,01 |
| Schichtplanung | Gratis | ~0,01 | ~0,01 |
| Chat (Text) | Gratis | ~0,02 | ~0,02 |
| Dashboard | Gratis | ~0,01 | ~0,01 |
| Inventar | Gering | ~0,04 | ~0,04 |
| Einkauf | Gering | ~0,02 | ~0,02 |
| Fuhrpark | Gering | ~0,02 | ~0,02 |
| Produktion | Gering | ~0,02 | ~0,02 |
| Vermietung | Gering | ~0,01 | ~0,01 |
| HR / Team | Gering | ~0,03 | ~0,03 |
| **E-Mail (IMAP-Client)** | **Gering** | **~0,05** | **~0,05** |
| **Video/Meetings (LiveKit)** | **Mittel** | **~0,20** | **~0,06** |
| **Dokument-Storage** | **Mittel-Hoch** | **~0,50-1,50** | **~0,50-1,50** |
| **OnlyOffice** | **Hoch (Fixkosten)** | **~3,52** | **~0,72** |
| **E-Signatur (Skribble)** | **Pass-Through** | **0** | **0** |
| **Newsletter (Brevo)** | **Pass-Through** | **0** | **0** |
| **Banking (FinAPI)** | **Sehr hoch** | **~7,50** | **~3,00** |
| **DATEV / Bexio / QR-Code** | **Gratis** | **~0,42** | **~0,08** |

---

## 2. Grundpaket

### Was sollte im guenstigsten Plan enthalten sein?

**Philosophie:** KMU Hub's USP ist "alles in einem". Wenn wir zu viel hinter Paywalls verstecken, verlieren wir den Differentiator gegenueber Pipedrive + Clockodo + Freshdesk. Also: **Grosszuegig im Funktionsumfang, streng bei Limits**.

### Empfohlenes Grundpaket ("Starter")

**Inkludierte Module (alle "Gratis"-Module + "Gering"-Module):**
- CRM / Kontakte / Deals / Pipeline
- Projekte (Kanban, Listen, Gantt)
- Kalender
- Zeiterfassung (Timer, Eintraege, Reports)
- Helpdesk / Tickets
- Wiki (mit TipTap-Editor)
- Formulare
- Vertraege
- Rapporte
- Berichte / Dashboard
- Schichtplanung
- Chat (Text, Channels, DMs)
- Inventar, Einkauf, Fuhrpark, Produktion, Vermietung
- HR / Team (Basis)
- E-Mail (IMAP-Client -- Verbindung zum bestehenden Mailserver)
- DATEV-Export
- Bexio-Sync
- Swiss QR-Rechnung

**Limits:**
- Bis 5 User
- 10 GB Storage
- Kein Video-Meetings
- Kein OnlyOffice
- Keine E-Signatur
- Kein Banking
- Kein Newsletter (Integration)
- Community-Support (E-Mail, 48h Response)

### Was kostet UNS das Grundpaket pro Kunde?

```
Basis-Infrastruktur (anteilig):     ~2,00 EUR/Mo (bei 100 Kunden)
Storage (10 GB inkludiert):         ~0,13 EUR/Mo
Alle Module (CPU/DB):               ~0,30 EUR/Mo
E-Mail IMAP-Sync:                   ~0,05 EUR/Mo
DATEV/Bexio/QR-Anteil:              ~0,42 EUR/Mo
------------------------------------------------------
GESAMT UNSERE KOSTEN:               ~2,90 EUR/Kunde/Mo (bei 100 Kunden)
                                     ~1,50 EUR/Kunde/Mo (bei 500 Kunden)
```

### Wettbewerber-Referenz

| Anbieter | Guenstigstes Produkt | Preis/User/Mo |
|----------|---------------------|---------------|
| Pipedrive | Essential | ~14 EUR |
| HubSpot | Starter | ~15 EUR |
| Monday.com | Basic | ~9 EUR |
| Zoho One | All-in-One | ~37 EUR (Jahresvertrag) |
| Clockodo | Standard | ~6,50 EUR |
| Freshdesk | Growth | ~15 EUR |

**Durchschnitt Einzel-Tool:** ~10-15 EUR/User/Mo
**Durchschnitt All-in-One:** ~37-45 EUR/User/Mo (Zoho One)

### Empfohlener Preis Grundpaket

**9 EUR/User/Mo** (Jahresvertrag) / **12 EUR/User/Mo** (Monatsvertrag)

**Bei 5 Usern = 45-60 EUR/Mo** (vs. unsere Kosten von ~2,90 EUR)
**Bruttomarge: ~93-95%**

**Positionierung:** "Guenstiger als JEDES einzelne Tool -- und du bekommst 20+ Module." Ein CRM allein (Pipedrive Essential) kostet bereits 14 EUR/User. Bei KMU Hub bekommst du CRM + PM + Helpdesk + Zeiterfassung + Chat + 15 weitere Module fuer 9 EUR/User.

---

## 3. Paket-Vorschlaege

### 3.1 Starter (5-15 User, kleine KMUs)

**Zielgruppe:** Handwerker, kleine Agenturen, Berater, Praxen

| Eigenschaft | Details |
|-------------|---------|
| **User** | 5-15 |
| **Module** | Alle Kern-Module (siehe Grundpaket) |
| **Video-Meetings** | 5 Teilnehmer max, 500 Minuten/Mo |
| **Storage** | 25 GB inkludiert |
| **E-Mail** | IMAP-Client (unbegrenzt) |
| **OnlyOffice** | NICHT enthalten |
| **E-Signatur** | NICHT enthalten |
| **Banking** | NICHT enthalten |
| **Newsletter** | NICHT enthalten |
| **Support** | E-Mail (24h Response, Geschaeftszeiten) |
| **DATEV/Bexio/QR** | Inkludiert |

**Unsere Kosten (15 User):**

```
Infrastruktur-Anteil:                ~2,00 EUR
Storage (25 GB):                     ~0,32 EUR
Video (500 Min, ~3 TN avg):         ~0,06 EUR
E-Mail IMAP:                         ~0,05 EUR
DATEV/Bexio/QR:                      ~0,42 EUR
----------------------------------------------
GESAMT:                              ~2,85 EUR/Kunde/Mo (bei 100 Kunden)
                                      ~1,45 EUR/Kunde/Mo (bei 500 Kunden)
```

**Empfohlener Preis:**

| Abrechnungs-Modell | Preis/User/Mo | Preis fuer 15 User/Mo |
|--------------------|---------------|------------------------|
| Jahresvertrag | **12 EUR** | **180 EUR** |
| Monatsvertrag | **15 EUR** | **225 EUR** |

**Marge (Jahresvertrag, 15 User, 100 Kunden):**
```
Umsatz:   180 EUR/Mo
Kosten:   2,85 EUR/Mo
Marge:    177,15 EUR/Mo = 98,4% Bruttomarge
```

---

### 3.2 Business (15-50 User, mittlere KMUs)

**Zielgruppe:** IT-Dienstleister, Bauunternehmen, Handelsunternehmen, mittelstaendisches Handwerk

| Eigenschaft | Details |
|-------------|---------|
| **User** | 15-50 |
| **Module** | Alle Module aus Starter |
| **Video-Meetings** | 25 Teilnehmer max, **unbegrenzt** Minuten |
| **Storage** | 100 GB inkludiert |
| **E-Mail** | IMAP-Client (unbegrenzt) |
| **OnlyOffice** | **Inkludiert** (Office-Dokumente inline bearbeiten) |
| **E-Signatur** | **FES inklusive** (50 Signaturen/Mo, EES unbegrenzt) |
| **Banking** | NICHT enthalten (Add-On) |
| **Newsletter** | **Integration inkludiert** (Kunde bringt eigenen Brevo/CleverReach-Account) |
| **Support** | E-Mail + Chat (8h Response, Geschaeftszeiten) |
| **Branchenprofil** | Inkludiert (anpassbar) |
| **Custom Fields** | Bis 50 pro Modul |

**Unsere Kosten (30 User):**

```
Infrastruktur-Anteil:                ~2,50 EUR
Storage (100 GB):                    ~1,26 EUR
Video (unbegrenzt, geschaetzt 2.000 Min/Mo): ~0,15 EUR
E-Mail IMAP:                         ~0,10 EUR
OnlyOffice (anteilig):               ~3,52 EUR (bei 100 Kunden)
                                      ~0,72 EUR (bei 500 Kunden)
E-Signatur (Pass-Through, 0):        ~0,00 EUR
Newsletter (Pass-Through):           ~0,00 EUR
DATEV/Bexio/QR:                      ~0,42 EUR
----------------------------------------------
GESAMT:                              ~8,00 EUR/Kunde/Mo (bei 100 Kunden)
                                      ~4,65 EUR/Kunde/Mo (bei 500 Kunden)
```

**Empfohlener Preis:**

| Abrechnungs-Modell | Preis/User/Mo | Preis fuer 30 User/Mo |
|--------------------|---------------|------------------------|
| Jahresvertrag | **19 EUR** | **570 EUR** |
| Monatsvertrag | **24 EUR** | **720 EUR** |

**Marge (Jahresvertrag, 30 User, 100 Kunden):**
```
Umsatz:   570 EUR/Mo
Kosten:   8,00 EUR/Mo
Marge:    562 EUR/Mo = 98,6% Bruttomarge
```

**Marge bei 500 Kunden:**
```
Umsatz:   570 EUR/Mo
Kosten:   4,65 EUR/Mo
Marge:    565,35 EUR/Mo = 99,2% Bruttomarge
```

---

### 3.3 Enterprise (50-200 User, groessere KMUs)

**Zielgruppe:** Grosses Bauunternehmen, Produktionsbetrieb, groesseres Handelsunternehmen

| Eigenschaft | Details |
|-------------|---------|
| **User** | 50-200 |
| **Module** | Alle Module |
| **Video-Meetings** | 50 Teilnehmer max, unbegrenzt, **Recording inkludiert** |
| **Storage** | 500 GB inkludiert |
| **E-Mail** | IMAP-Client (unbegrenzt) |
| **OnlyOffice** | Inkludiert |
| **E-Signatur** | **Alle Level** (EES/FES/QES) |
| **Banking** | **Inkludiert** (FinAPI) |
| **Newsletter** | Integration inkludiert |
| **Support** | **Prioritaet** (4h Response, Telefon + Chat + E-Mail) |
| **Branchenprofil** | Inkludiert + **individuell anpassbar** |
| **Custom Fields** | Unbegrenzt |
| **API-Zugang** | Vollstaendig (REST API fuer Drittanbieter) |
| **SSO/SAML** | Inkludiert |
| **Dedizierte DB** | Optional (Aufpreis) |
| **SLA** | 99,5% Uptime |

**Unsere Kosten (100 User):**

```
Infrastruktur-Anteil:                ~5,00 EUR
Storage (500 GB):                    ~6,30 EUR
Video (unbegrenzt + Recording):      ~0,50 EUR
Recording-Storage (~50 GB/Mo):       ~0,63 EUR
E-Mail IMAP:                         ~0,20 EUR
OnlyOffice (anteilig):               ~3,52 EUR (bei 100 Kunden)
Banking (FinAPI anteilig):           ~7,50 EUR (bei 100 aktiven Banking-Kunden)
                                      ~3,00 EUR (bei 500)
E-Signatur (Pass-Through):           ~0,00 EUR
DATEV/Bexio/QR:                      ~0,42 EUR
----------------------------------------------
GESAMT:                              ~24,07 EUR/Kunde/Mo (bei 100 Kunden)
                                      ~16,05 EUR/Kunde/Mo (bei 500 Kunden)
```

**Empfohlener Preis:**

| Abrechnungs-Modell | Preis/User/Mo | Preis fuer 100 User/Mo |
|--------------------|---------------|------------------------|
| Jahresvertrag | **25 EUR** | **2.500 EUR** |
| Monatsvertrag | **32 EUR** | **3.200 EUR** |

**Marge (Jahresvertrag, 100 User, 100 Kunden):**
```
Umsatz:     2.500 EUR/Mo
Kosten:     24,07 EUR/Mo
Marge:      2.475,93 EUR/Mo = 99,0% Bruttomarge
```

---

### 3.4 Self-Hosted

**Zielgruppe:** KMUs mit eigener IT oder IT-Partner, hohe Datenschutz-Anforderungen, Behoerden-Lieferanten

| Modell | Preis | Enthaelt |
|--------|-------|----------|
| **Jahreslizenz (Klein, bis 20 User)** | 2.400 EUR/Jahr (200 EUR/Mo) | Alle Module, Updates, E-Mail-Support |
| **Jahreslizenz (Mittel, bis 100 User)** | 6.000 EUR/Jahr (500 EUR/Mo) | Alle Module, Updates, Prioritaets-Support |
| **Jahreslizenz (Gross, bis 200 User)** | 12.000 EUR/Jahr (1.000 EUR/Mo) | Alle Module, Updates, Telefon-Support, SLA |
| **Einmallizenz** (Optional) | 15.000-40.000 EUR | Dauerlizenz + 1 Jahr Updates, danach optional |

**Unsere Kosten pro Self-Hosted-Kunde:**

```
Entwicklungskosten (anteilig):       0 EUR (bereits in SaaS-Entwicklung enthalten)
Support-Aufwand:                     ~1-4h/Mo (je nach Groesse)
Update-Bereitstellung:               ~0,5h/Mo (Docker Image bauen + testen)
Kein Hosting, kein Storage, kein Bandbreiten-Aufwand (alles beim Kunden)
----------------------------------------------
Tatsaechliche Kosten:                ~50-200 EUR/Mo (rein Arbeitszeit-basiert)
```

**Marge Self-Hosted:**
- Klein: 200 EUR/Mo Umsatz, ~50 EUR/Mo Kosten = 75% Bruttomarge
- Mittel: 500 EUR/Mo Umsatz, ~100 EUR/Mo Kosten = 80% Bruttomarge
- Gross: 1.000 EUR/Mo Umsatz, ~200 EUR/Mo Kosten = 80% Bruttomarge

**Hinweis:** Self-Hosted ist weniger profitabel pro Kunde, aber:
1. Kein Infrastruktur-Risiko (Kunde traegt Hardware-Kosten)
2. Differentiator gegenueber Zoho/HubSpot (die kein Self-Hosting anbieten)
3. Compliance-Argument (volle Datenkontrolle)

---

## 4. Add-Ons (Einzeln zubuchbar)

### 4.1 Video-Meetings Pack

| Tier | Enthalten | Unsere Kosten | Preis/Mo | Marge |
|------|-----------|---------------|----------|-------|
| **Basic** (in Starter) | 500 Min, 5 TN max | ~0,06 EUR | Inkludiert | -- |
| **Pro** | Unbegrenzt, 25 TN, Recording | ~0,50 EUR | 10 EUR | 95% |
| **Enterprise** | Unbegrenzt, 50 TN, Recording, Webinar | ~1,50 EUR | 25 EUR | 94% |

**Typische Kaeufer:** Agenturen, Berater, Dienstleister, IT-Firmen

---

### 4.2 Extra Storage

| Menge | Unsere Kosten | Preis/Mo | Marge |
|-------|---------------|----------|-------|
| +25 GB | 0,32 EUR | 3 EUR | 89% |
| +50 GB | 0,63 EUR | 5 EUR | 87% |
| +100 GB | 1,26 EUR | 9 EUR | 86% |
| +500 GB | 6,30 EUR | 35 EUR | 82% |
| +1 TB | 12,60 EUR | 60 EUR | 79% |

**Typische Kaeufer:** Bau (Fotos!), Handwerk, Unternehmen mit viel Dokumenten-Verkehr

---

### 4.3 OnlyOffice (Office-Editing)

| Tier | Enthalten | Unsere Kosten | Preis/Mo | Marge |
|------|-----------|---------------|----------|-------|
| **Standard** (in Business) | .docx/.xlsx/.pptx bearbeiten | ~3,52 EUR (100 Kunden) | Inkludiert (ab Business) | -- |
| **Einzel-Add-On** (fuer Starter) | .docx/.xlsx/.pptx bearbeiten | ~3,52 EUR | 8 EUR | 56%* |

*Marge steigt mit Kundenzahl: Bei 500 Kunden = 0,72 EUR Kosten -> 91% Marge

**Typische Kaeufer:** Jeder der "kein Microsoft 365 mehr brauchen" will

---

### 4.4 E-Signatur

| Modell | Enthalten | Unsere Kosten | Preis | Marge |
|--------|-----------|---------------|-------|-------|
| **Pay-per-Signatur** (EES) | Pro Signatur | ~0,50-1,50 CHF | 2,00 EUR/Signatur | 50-75% |
| **Pay-per-Signatur** (FES) | Pro Signatur | ~1,50-2,50 CHF | 3,50 EUR/Signatur | 40-60% |
| **Pay-per-Signatur** (QES) | Pro Signatur | ~2,50-4,00 CHF | 6,00 EUR/Signatur | 35-55% |
| **Flatrate** (50 EES+FES/Mo) | Monatlich | ~85 CHF (Skribble Business) | 15 EUR/Mo + Skribble-Konto des Kunden | Pass-Through |

**Empfehlung:** Pass-Through-Modell. Kunde verbindet eigenen Skribble-Account. Wir verdienen nichts daran, aber es erhoehrt den Wert der Plattform und reduziert Churn.

**Typische Kaeufer:** Vertragsintensive Branchen (Bau, Vermietung, Beratung)

---

### 4.5 Newsletter-Integration

| Modell | Enthalten | Unsere Kosten | Preis/Mo | Marge |
|--------|-----------|---------------|----------|-------|
| **Standard** (in Business) | Brevo/CleverReach-Anbindung, Kontakt-Sync | 0 EUR | Inkludiert (ab Business) | -- |
| **Einzel-Add-On** (fuer Starter) | Brevo/CleverReach-Anbindung | 0 EUR | 5 EUR | 100% |

**Hinweis:** Kunde braucht eigenen Brevo/CleverReach-Account (nicht in KMU Hub Preis enthalten).

---

### 4.6 Banking-Integration (FinAPI)

| Modell | Enthalten | Unsere Kosten | Preis/Mo | Marge |
|--------|-----------|---------------|----------|-------|
| **Standard** (in Enterprise) | Bankabgleich, Auto-Matching | ~7,50 EUR (100 Kunden) | Inkludiert (ab Enterprise) | -- |
| **Einzel-Add-On** | Bankabgleich, Auto-Matching | ~7,50 EUR | 15 EUR | 50% |

**ACHTUNG:** Erst anbieten wenn mind. 50 Kunden Banking wollen. Vorher rechnet sich FinAPI nicht.

**Typische Kaeufer:** Jeder mit offener-Posten-Management (Handwerk, Handel, Dienstleister)

---

### 4.7 Premium Support

| Tier | Response-Time | Kanaele | Preis/Mo |
|------|--------------|---------|----------|
| **Standard** (inkl.) | 24h | E-Mail | 0 EUR |
| **Priority** (in Enterprise) | 4h | E-Mail, Chat, Telefon | Inkludiert |
| **Premium Add-On** | 4h | E-Mail, Chat, Telefon, Bildschirmfreigabe | 50 EUR/Mo |
| **Dedicated** | 1h, persoenlicher Ansprechpartner | Alle + Slack-Channel | 200 EUR/Mo |

---

### 4.8 Schweizer Datenresidenz (Exoscale)

| Modell | Enthalten | Unsere Kosten | Preis/Mo |
|--------|-----------|---------------|----------|
| Standard (Hetzner DE) | EU-Hosting | Basis | 0 EUR (inkludiert) |
| **Schweizer Cluster** | Daten in Zuerich/Genf (Exoscale) | ~100-150 EUR Mehrkosten* | +50 EUR/Mo |

*Exoscale ist ~2-3x teurer als Hetzner (aus `09-infrastruktur-matrix.md`):
- Compute: ~40-55 EUR (vs. ~10 EUR Hetzner)
- Managed DB: ~75-100 EUR (vs. ~5-10 EUR Hetzner)
- Object Storage: ~0,022 EUR/GB (vs. ~0,0126 EUR Hetzner)

**Typische Kaeufer:** Schweizer Anwaelte, Aertze, Finanzdienstleister, Behoerden-Lieferanten

---

### 4.9 Dedizierter Server (Compliance)

| Modell | Enthalten | Unsere Kosten | Preis/Mo |
|--------|-----------|---------------|----------|
| **Shared** (Standard) | Multi-Tenant (RLS) | Basis | 0 EUR |
| **Dedizierte DB** | Eigene PostgreSQL-Instanz | ~15-30 EUR | +100 EUR/Mo |
| **Voll dediziert** | Eigene Server-Infrastruktur | ~80-200 EUR | +300 EUR/Mo |

**Typische Kaeufer:** Unternehmen mit ISO 27001, Behoerden-Lieferanten, Branchen mit strengen Datenschutz-Anforderungen

---

## 5. Wettbewerber-Preisvergleich

### Szenario: 15-Personen-Unternehmen (Dienstleister/Agentur)

Benoetigt: CRM, PM, Helpdesk, Zeiterfassung, Chat, Video, Dokumente, E-Mail, Newsletter

| Loesung | Zusammensetzung | Monatliche Kosten (15 User) |
|---------|-----------------|----------------------------|
| **Pipedrive + Clockodo + Freshdesk + Brevo + Dropbox** | CRM (15x14) + Zeit (15x6,50) + Helpdesk (15x15) + Newsletter (19) + Storage (75) | **~627 EUR** |
| **HubSpot Starter + Zendesk + Asana** | HubSpot (15x15) + Zendesk (15x19) + Asana (15x11) | **~675 EUR** |
| **Zoho One** | All-in-One (15x37) | **~555 EUR** |
| **Microsoft 365 E3 + Dynamics + Notion** | M365 (15x36) + Dynamics (15x60) + Notion (15x8) | **~1.560 EUR** |
| **Monday.com + Pipedrive + Freshdesk** | PM (15x12) + CRM (15x14) + Helpdesk (15x15) | **~615 EUR** |
| **Odoo Enterprise** | All-in-One (15x24 + Module-Aufpreise) | **~480-720 EUR** |
| | | |
| **KMU Hub Starter** | Alles in einem | **~180 EUR** |
| **KMU Hub Business** | Alles in einem + Video + OnlyOffice | **~285 EUR** |

### Einsparungs-Argument

| vs. Wettbewerber | KMU Hub Business (15 User) | Ersparnis | Ersparnis % |
|-----------------|---------------------------|-----------|-------------|
| vs. Pipedrive-Stack | 285 vs. 627 EUR | 342 EUR/Mo = **4.104 EUR/Jahr** | 55% |
| vs. HubSpot-Stack | 285 vs. 675 EUR | 390 EUR/Mo = **4.680 EUR/Jahr** | 58% |
| vs. Zoho One | 285 vs. 555 EUR | 270 EUR/Mo = **3.240 EUR/Jahr** | 49% |
| vs. M365+Dynamics | 285 vs. 1.560 EUR | 1.275 EUR/Mo = **15.300 EUR/Jahr** | 82% |
| vs. Odoo Enterprise | 285 vs. ~600 EUR | 315 EUR/Mo = **3.780 EUR/Jahr** | 53% |

**Kern-Message:** "Spart 3.000-15.000 EUR pro Jahr gegenueber dem bisherigen Tool-Mix."

### Detaillierter Feature-Vergleich (15 User, Business-Paket)

| Feature | KMU Hub Business (285 EUR) | Pipedrive+Stack (~627 EUR) | Zoho One (~555 EUR) | M365+Dynamics (~1.560 EUR) |
|---------|---------------------------|---------------------------|--------------------|-----------------------------|
| CRM + Pipeline | Ja | Ja (Pipedrive) | Ja | Ja (Dynamics) |
| Projektmanagement | Ja (Kanban, Gantt) | Nein (extra Tool noetig) | Ja (Zoho Projects) | Teilweise (Planner) |
| Helpdesk | Ja | Ja (Freshdesk, +225 EUR) | Ja (Zoho Desk) | Nein (extra) |
| Zeiterfassung | Ja | Ja (Clockodo, +98 EUR) | Ja (begrenzt) | Nein (extra) |
| Chat | Ja | Nein (Slack noetig, +113 EUR) | Ja (Zoho Cliq) | Ja (Teams) |
| Video-Meetings | Ja | Nein (Zoom noetig) | Ja (Zoho Meeting) | Ja (Teams) |
| Office-Editing | Ja (OnlyOffice) | Nein | Ja (Zoho Writer) | Ja (M365) |
| E-Mail-Client | Ja (IMAP) | Nein | Ja (Zoho Mail) | Ja (Outlook) |
| Inventar | Ja | Nein | Ja (Zoho Inventory) | Nein |
| Schichtplanung | Ja | Nein | Nein | Nein |
| Fuhrpark | Ja | Nein | Nein | Nein |
| Rapporte | Ja | Nein | Nein | Nein |
| EU-Hosting | Ja (Hetzner DE) | Teilweise | Nein (Indien/US) | Nein (US) |
| Self-Hosted | Ja | Nein | Nein | Nein |
| Branchenprofil | Ja (10 Profile) | Nein | Nein | Nein |
| DATEV-Export | Ja | Nein (Clockodo hat es) | Nein | Nein |
| Massanfertigung | Ja (Onsite-Woche) | Nein | Nein | Via Partner (teuer) |

---

## 6. Kostenrechnung pro Kundengroesse

### 6.1 Klein (10 User, Starter-Paket)

**Typisch:** Kleine Agentur, Handwerksbetrieb, Arztpraxis

```
INFRASTRUKTUR
  Server-Anteil (anteilig, 100 Kunden):    2,00 EUR
  Storage (25 GB inkludiert):               0,32 EUR
  Video (500 Min/Mo):                       0,06 EUR
  E-Mail IMAP-Sync:                         0,05 EUR
                                            --------
  Subtotal Infrastruktur:                   2,43 EUR

LIZENZEN (anteilig)
  DATEV/QR/ZUGFeRD (unipdf):               0,42 EUR
  OnlyOffice:                               0,00 EUR (nicht im Starter)
                                            --------
  Subtotal Lizenzen:                        0,42 EUR

EXTERNE DIENSTE
  FinAPI:                                   0,00 EUR (nicht im Starter)
  Skribble:                                 0,00 EUR (nicht im Starter)
  Brevo:                                    0,00 EUR (nicht im Starter)
                                            --------
  Subtotal Externe:                         0,00 EUR

==============================================
GESAMT UNSERE KOSTEN:                       2,85 EUR/Mo

UMSATZ (10 User x 12 EUR/Mo):              120,00 EUR/Mo
BRUTTOMARGE:                                117,15 EUR/Mo = 97,6%
BRUTTOMARGE PRO USER:                       11,72 EUR/Mo
```

---

### 6.2 Mittel (30 User, Business-Paket)

**Typisch:** IT-Dienstleister, mittlerer Handwerksbetrieb, Bauunternehmen

```
INFRASTRUKTUR
  Server-Anteil (anteilig, 100 Kunden):    2,50 EUR
  Storage (100 GB inkludiert):              1,26 EUR
  Video (unbegrenzt, ~2.000 Min):           0,15 EUR
  E-Mail IMAP-Sync:                         0,10 EUR
  Recording-Storage (~10 GB):               0,13 EUR
                                            --------
  Subtotal Infrastruktur:                   4,14 EUR

LIZENZEN (anteilig, 100 Kunden)
  DATEV/QR/ZUGFeRD:                         0,42 EUR
  OnlyOffice Developer:                     3,52 EUR
                                            --------
  Subtotal Lizenzen:                        3,94 EUR

EXTERNE DIENSTE
  FinAPI:                                   0,00 EUR (Add-On)
  Skribble:                                 0,00 EUR (Pass-Through)
  Brevo:                                    0,00 EUR (Pass-Through)
                                            --------
  Subtotal Externe:                         0,00 EUR

==============================================
GESAMT UNSERE KOSTEN:                       8,08 EUR/Mo

UMSATZ (30 User x 19 EUR/Mo):              570,00 EUR/Mo
BRUTTOMARGE:                                561,92 EUR/Mo = 98,6%
BRUTTOMARGE PRO USER:                       18,73 EUR/Mo
```

---

### 6.3 Gross (100 User, Enterprise-Paket)

**Typisch:** Grosses Bauunternehmen, Produktionsbetrieb, Handelsunternehmen

```
INFRASTRUKTUR
  Server-Anteil (anteilig, 100 Kunden):    5,00 EUR
  Storage (500 GB inkludiert):              6,30 EUR
  Video (unbegrenzt + Recording):           0,50 EUR
  Recording-Storage (~50 GB):               0,63 EUR
  E-Mail IMAP-Sync:                         0,20 EUR
                                            --------
  Subtotal Infrastruktur:                   12,63 EUR

LIZENZEN (anteilig, 100 Kunden)
  DATEV/QR/ZUGFeRD:                         0,42 EUR
  OnlyOffice Developer:                     3,52 EUR
                                            --------
  Subtotal Lizenzen:                        3,94 EUR

EXTERNE DIENSTE
  FinAPI (anteilig):                        7,50 EUR
  Skribble:                                 0,00 EUR (Pass-Through)
  Brevo:                                    0,00 EUR (Pass-Through)
                                            --------
  Subtotal Externe:                         7,50 EUR

==============================================
GESAMT UNSERE KOSTEN:                       24,07 EUR/Mo

UMSATZ (100 User x 25 EUR/Mo):             2.500,00 EUR/Mo
BRUTTOMARGE:                                2.475,93 EUR/Mo = 99,0%
BRUTTOMARGE PRO USER:                       24,76 EUR/Mo
```

---

### 6.4 Kosten-Zusammenfassung (alle Groessen)

| Metrik | Klein (10 User) | Mittel (30 User) | Gross (100 User) |
|--------|-----------------|-------------------|-------------------|
| Paket | Starter | Business | Enterprise |
| Preis/User/Mo | 12 EUR | 19 EUR | 25 EUR |
| Umsatz/Mo | 120 EUR | 570 EUR | 2.500 EUR |
| Unsere Kosten/Mo | 2,85 EUR | 8,08 EUR | 24,07 EUR |
| Bruttomarge | 97,6% | 98,6% | 99,0% |
| Bruttomarge/User | 11,72 EUR | 18,73 EUR | 24,76 EUR |

**Zentrale Erkenntnis:** Die Bruttomarge auf Hosting/Infrastruktur ist bei SaaS enorm (>97%). Die echten Kosten sind **nicht die Infrastruktur**, sondern:
- Personal (Entwicklung, Support, Sales)
- Kundenakquise (CAC)
- Compliance (Rechtsanwalt, Penetrationstest, ISO)
- Onsite-Einrichtung (Reise, Zeit)

---

## 7. Break-Even Analyse

### 7.1 Fixkosten (monatlich)

| Posten | Kosten/Mo | Anmerkung |
|--------|-----------|-----------|
| **Infrastruktur (Basis)** | 200 EUR | Hetzner Cluster fuer 100 Kunden |
| **OnlyOffice Lizenz** | 333 EUR | Developer Edition |
| **FinAPI Grundgebuehr** | 500 EUR | Ab dem Zeitpunkt wo Banking aktiv ist |
| **Domain + SSL + DNS** | 10 EUR | Cloudflare, Domains |
| **Monitoring** | 30 EUR | Externes Monitoring (optional) |
| **unipdf Lizenz (anteilig)** | 42 EUR | ZUGFeRD PDF/A-3 |
| **DSB (Datenschutzbeauftragter)** | 400 EUR | Externer DSB |
| **Haftpflicht + Versicherungen** | 200 EUR | Geschaetzt |
| **Buchhaltung/Steuerberater** | 300 EUR | Eigene Buchhaltung |
| **Sonstiges (Tools, SaaS)** | 200 EUR | GitHub, E-Mail, etc. |
| | | |
| **GESAMT Fixkosten (ohne Personal)** | **~2.215 EUR/Mo** | |

### 7.2 Personalkosten

| Rolle | Kosten/Mo (Brutto + Abgaben) | Anmerkung |
|-------|------------------------------|-----------|
| **Luke (Entwicklung)** | ~4.000-6.000 EUR | Haupt-Entwickler (Annahme) |
| **Darien (Design)** | ~3.000-5.000 EUR | Design + Frontend (Annahme) |
| **Business-Person 1** | ~3.000-5.000 EUR | Sales + Onboarding |
| | | |
| **GESAMT Personal** | **~10.000-16.000 EUR/Mo** | |

### 7.3 Einmalige Kosten (Pre-Launch)

| Posten | Kosten | Anmerkung |
|--------|--------|-----------|
| AVV + Datenschutzerklaerung | 3.000-6.000 EUR | Rechtsanwalt |
| Penetrationstest | 3.000-8.000 EUR | Vor Launch |
| Logo / Branding | 1.000-3.000 EUR | Falls noch nicht vorhanden |
| FinAPI Setup | 0-5.000 EUR | Einmalig |
| Marketing-Website | 2.000-5.000 EUR | Landing Pages |
| **GESAMT Einmalig** | **~9.000-27.000 EUR** | |

### 7.4 Break-Even Berechnung

**Monatliche Gesamtkosten = Fixkosten + Personal + Variable Kosten**

```
Fixkosten:                ~2.215 EUR/Mo
Personal:                 ~13.000 EUR/Mo (Mittelwert)
Variable Kosten/Kunde:    ~5 EUR/Mo (gewichteter Durchschnitt)
----------------------------------------------
Gesamt-Fix:               ~15.215 EUR/Mo
Variable/Kunde:           ~5 EUR/Mo
```

**Durchschnittlicher Umsatz pro Kunde (ARPU):**

Angenommen: 60% Starter (10 User), 30% Business (25 User), 10% Enterprise (80 User)

```
ARPU = 0,60 x (10 x 12) + 0,30 x (25 x 19) + 0,10 x (80 x 25)
     = 0,60 x 120 + 0,30 x 475 + 0,10 x 2.000
     = 72 + 142,50 + 200
     = 414,50 EUR/Kunde/Mo
```

**Break-Even Formel:**

```
Break-Even = Fixkosten / (ARPU - Variable Kosten)
           = 15.215 / (414,50 - 5,00)
           = 15.215 / 409,50
           = ~37 Kunden
```

### Break-Even: ~37 zahlende Kunden

**Timeline-Schaetzung:**

| Monat nach Launch | Kunden (konservativ) | Monatlicher Umsatz | Kosten | Gewinn/Verlust |
|-------------------|---------------------|---------------------|--------|----------------|
| Monat 1 | 5 | 2.073 EUR | 15.240 EUR | -13.168 EUR |
| Monat 3 | 15 | 6.218 EUR | 15.290 EUR | -9.073 EUR |
| Monat 6 | 30 | 12.435 EUR | 15.365 EUR | -2.930 EUR |
| **Monat 7-8** | **37** | **15.337 EUR** | **15.400 EUR** | **~0 EUR (Break-Even)** |
| Monat 12 | 60 | 24.870 EUR | 15.515 EUR | +9.355 EUR |
| Monat 18 | 100 | 41.450 EUR | 15.715 EUR | +25.735 EUR |
| Monat 24 | 180 | 74.610 EUR | 16.115 EUR | +58.495 EUR |

### 7.5 Customer Lifetime Value (CLV)

**Annahmen:**
- Durchschnittliche Kundenbindung: 36 Monate (3 Jahre, konservativ fuer KMU-Software)
- ARPU: 414,50 EUR/Mo
- Gross Margin: ~98%
- Variable Kosten: ~5 EUR/Mo

```
CLV = ARPU x Gross Margin x Durchschnittliche Lebensdauer
    = 414,50 x 0,98 x 36
    = ~14.624 EUR pro Kunde

Netto-CLV (nach variablen Kosten):
    = (414,50 - 5,00) x 36
    = ~14.742 EUR pro Kunde
```

**CAC-Ziel (Customer Acquisition Cost):**
- Gesundes SaaS-Verhaeltnis: CLV/CAC >= 3:1
- Maximaler CAC: 14.742 / 3 = **~4.914 EUR pro Kunde**
- Das bedeutet: Wir koennen bis zu ~4.900 EUR pro Kunde in Akquise investieren (Onsite-Woche, Marketing, Sales)

---

## 8. Branchenspezifische Pakete

### 8.1 Dienstleister-Paket (Agentur, Beratung, IT)

**Profil:** `dienstleistung`, `it_tech`
**Typische Groesse:** 5-30 MA
**Abdeckung:** ~85%

| Feature | Details |
|---------|---------|
| CRM + Pipeline + Deals | Ja |
| Projekte (Kanban, Gantt) | Ja |
| Zeiterfassung + Stunden-zu-Rechnung | Ja |
| Chat + Video-Meetings | Ja (unbegrenzt) |
| Helpdesk (fuer IT-Support-Firmen) | Ja |
| OnlyOffice (Angebote/Vertraege) | Ja |
| Newsletter-Integration | Ja |
| DATEV/Bexio | Ja |

**Empfohlenes Paket:** Business (19 EUR/User)
**15 User = 285 EUR/Mo**

**Unsere Kosten:** ~8 EUR/Mo
**Marge:** 97%

**Verkaufs-Argument:** "Ersetzt Pipedrive (210 EUR) + Clockodo (98 EUR) + Asana (165 EUR) + Slack (113 EUR) = 586 EUR. Du sparst 301 EUR/Mo."

---

### 8.2 Handwerk-Paket (Elektriker, Sanitaer, Maler, Schreiner)

**Profil:** `handwerk`
**Typische Groesse:** 5-20 MA
**Abdeckung:** ~80%

| Feature | Details |
|---------|---------|
| CRM + Kontakte | Ja |
| Projekte | Ja |
| Rapporte + Fotodokumentation | Ja |
| Zeiterfassung | Ja |
| Einkauf + Inventar | Ja |
| Fuhrpark + Fahrtenbuch | Ja |
| Schichtplanung | Ja |
| DATEV-Export + QR-Rechnung | Ja |
| Belegkette (Angebot -> Rechnung) | Ja |
| **Extra Storage** | +50 GB (fuer Rapport-Fotos) |

**Empfohlenes Paket:** Starter + Extra Storage
**10 User = 120 EUR + 5 EUR Storage = 125 EUR/Mo**

**Unsere Kosten:** ~3,50 EUR/Mo
**Marge:** 97%

**Verkaufs-Argument:** "Deine Rapporte, Zeiterfassung und Rechnungen in EINER App. Kein Papier mehr. Und dein Steuerberater bekommt die DATEV-Datei per Knopfdruck."

---

### 8.3 Bau-Paket (Bauunternehmen, Tiefbau, Ausbau)

**Profil:** `bau`
**Typische Groesse:** 10-50 MA
**Abdeckung:** ~70%

| Feature | Details |
|---------|---------|
| Projekte (Bauvorhaben) | Ja |
| Rapporte + Aufmass + Fotodokumentation | Ja |
| Schichtplanung | Ja |
| Inventar (Maschinen/Material) | Ja |
| Einkauf | Ja |
| Fuhrpark | Ja |
| Vermietung (Container, Geraete) | Ja |
| Zeiterfassung | Ja |
| DATEV + ZUGFeRD | Ja |
| **Extra Storage** | +250 GB (Fotos, Plaene!) |

**Empfohlenes Paket:** Business + Extra Storage (250 GB)
**30 User = 570 EUR + 35 EUR Storage = 605 EUR/Mo**

**Unsere Kosten:** ~11 EUR/Mo
**Marge:** 98%

**Verkaufs-Argument:** "Bautagebuch, Aufmass, Schichtplanung und Einkauf in EINER App. Alle Fotos direkt am Projekt. Kein WhatsApp-Chaos mehr auf der Baustelle."

---

### 8.4 Handel-Paket (Einzelhandel, Grosshandel)

**Profil:** `einzelhandel`
**Typische Groesse:** 5-50 MA
**Abdeckung:** ~65%

| Feature | Details |
|---------|---------|
| Inventar (Artikelverwaltung) | Ja |
| CRM (Kundenbeziehungen) | Ja |
| Einkauf (Lieferanten) | Ja |
| Schichtplanung | Ja |
| HR/Team | Ja |
| Berichte + Dashboard | Ja |
| DATEV + Bexio | Ja |
| Banking (optional) | Add-On |

**Empfohlenes Paket:** Starter
**15 User = 180 EUR/Mo**

**Unsere Kosten:** ~3 EUR/Mo
**Marge:** 98%

**Verkaufs-Argument:** "Warenwirtschaft + CRM + Schichtplanung zusammen -- nicht 3 verschiedene Apps."

**Einschraenkung:** Ohne E-Commerce/Webshop-Anbindung (Shopify/WooCommerce) fehlt der staerkste Verkaufsgrund. Handel ist daher Phase 4 (nach E-Commerce-Integration).

---

### 8.5 Gastro-Paket (Restaurant, Cafe, Hotel)

**Profil:** `gastronomie`
**Typische Groesse:** 10-50 MA
**Abdeckung:** ~55%

| Feature | Details |
|---------|---------|
| Schichtplanung | Ja |
| Inventar (Lebensmittel, Ablaufdaten) | Ja |
| Einkauf (Lieferanten, Bestellungen) | Ja |
| HR/Team | Ja |
| Zeiterfassung | Ja |
| DATEV | Ja |

**Empfohlenes Paket:** Starter
**20 User = 240 EUR/Mo**

**Unsere Kosten:** ~3 EUR/Mo
**Marge:** 99%

**Verkaufs-Argument:** "Dein komplettes Backoffice: Schichtplanung, Einkauf und Personalverwaltung in einer App."

**Einschraenkung:** Ohne Kassensystem (POS) und Tischreservierung ist Gastro nur Backoffice-Looesung. Positionierung als "die App HINTER der Kasse", nicht als Kassen-Ersatz.

---

### Branchen-Pricing-Uebersicht

| Branche | Typische Groesse | Empfohlenes Paket | Preis/Mo | Unsere Kosten | Marge |
|---------|-----------------|-------------------|----------|---------------|-------|
| Dienstleister | 15 User | Business | 285 EUR | 8 EUR | 97% |
| Handwerk | 10 User | Starter + Storage | 125 EUR | 3,50 EUR | 97% |
| Bau | 30 User | Business + Storage | 605 EUR | 11 EUR | 98% |
| Handel | 15 User | Starter | 180 EUR | 3 EUR | 98% |
| Gastro | 20 User | Starter | 240 EUR | 3 EUR | 99% |

---

## 9. Einrichtungskosten (Onsite-Woche)

### 9.1 Was kostet UNS die Onsite-Woche?

Die 1-Woche-Onsite-Prozessanalyse ist KMU Hub's USP ("Massanfertigung"). Aber sie hat reale Kosten.

**Kosten-Aufstellung (1 Person, 5 Tage):**

| Posten | Kosten (Inland DE/CH) | Kosten (Reise >200 km) |
|--------|----------------------|------------------------|
| **Arbeitszeit** (5 Tage a 8h = 40h) | ~2.000-3.000 EUR* | ~2.000-3.000 EUR |
| **Reise** (Bahn/Auto) | ~50-100 EUR | ~200-400 EUR |
| **Unterkunft** (4 Naechte Hotel) | 0 EUR (Heimkehr) | ~400-800 EUR |
| **Verpflegung** (5 Tage) | ~100 EUR | ~200 EUR |
| **Nachbereitung** (Konfiguration, 2-3 Tage) | ~800-1.200 EUR | ~800-1.200 EUR |
| | | |
| **GESAMT** | **~2.950-4.400 EUR** | **~3.600-5.600 EUR** |

*Arbeitszeit berechnet mit 50-75 EUR/h (interner Satz, nicht Verkaufspreis)

### 9.2 Was berechnen wir dem Kunden?

**Optionen:**

| Modell | Preis | Marge | Empfehlung |
|--------|-------|-------|------------|
| **A: Kostenlos** (bei Jahresvertrag) | 0 EUR | Negativ (-3.000 bis -5.000) | Fuer Enterprise + lange Vertragsbindung (24 Mo) |
| **B: Subventioniert** | 1.500 EUR | ~-50% | Fuer Business + Jahresvertrag -- Einstiegsangebot |
| **C: Kostendeckend** | 3.500 EUR | ~0% | Standard-Preis |
| **D: Gewinnbringend** | 5.000-8.000 EUR | +30-80% | Fuer komplexe Setups (>50 User, mehrere Standorte) |

**Empfehlung:**

```
Starter (5-15 User):     1.500 EUR einmalig (subventioniert)
                          ODER kostenlos bei 24-Monats-Vertrag

Business (15-50 User):   2.500 EUR einmalig
                          ODER kostenlos bei 24-Monats-Vertrag

Enterprise (50-200 User): 5.000 EUR einmalig (2 Personen, 1 Woche)
                           Immer inkludiert bei Enterprise
```

**CLV-Rechnung fuer Kostenlos-Option:**

```
Starter-Kunde (10 User, 24 Mo Vertrag):
  Umsatz:          120 EUR x 24 = 2.880 EUR
  Onsite-Kosten:   -3.000 EUR
  Laufende Kosten: -2,85 x 24 = -68 EUR
  Nettoergebnis:   -188 EUR (NEGATIV!)

  --> Onsite kostenlos nur bei Business/Enterprise sinnvoll

Business-Kunde (25 User, 24 Mo Vertrag):
  Umsatz:          475 EUR x 24 = 11.400 EUR
  Onsite-Kosten:   -4.000 EUR
  Laufende Kosten: -8,08 x 24 = -194 EUR
  Nettoergebnis:   +7.206 EUR = LOHNT SICH

Enterprise-Kunde (80 User, 24 Mo Vertrag):
  Umsatz:          2.000 EUR x 24 = 48.000 EUR
  Onsite-Kosten:   -6.000 EUR (2 Personen)
  Laufende Kosten: -24 x 24 = -576 EUR
  Nettoergebnis:   +41.424 EUR = SEHR LOHNEND
```

### 9.3 Wann brauchen wir die Onsite-Woche?

| Szenario | Onsite? | Begruendung |
|----------|---------|-------------|
| Starter-Kunde (Standard-Setup) | **Nein** | Remote-Onboarding (2-3h Video-Call) reicht |
| Starter-Kunde (komplexe Prozesse) | **Optional** (0,5 Tage Remote) | Wenn Kunde Hilfe bei Konfiguration braucht |
| Business-Kunde | **Empfohlen** (2-3 Tage) | Kuerzere Version, Fokus auf Kernprozesse |
| Enterprise-Kunde | **Pflicht** (5 Tage) | Volle Prozessanalyse, mehrere Abteilungen |
| Self-Hosted-Kunde | **Empfohlen** (2-3 Tage) | Setup + Konfiguration + Schulung |

**Skalierungsplan:**

| Phase | Onsite-Kapazitaet | Wer macht es? |
|-------|-------------------|---------------|
| Beta (0-20 Kunden) | 1-2 Onsites/Monat | Darien oder Business-Person |
| Launch (20-50 Kunden) | 3-4 Onsites/Monat | Business-Person + evtl. Freelancer |
| Growth (50-100 Kunden) | Remote-first mit Video-Onboarding | Standardisiertes Onboarding, Onsite nur Enterprise |
| Scale (100+ Kunden) | Onsite nur Enterprise + Self-Hosted | Partner-Netzwerk aufbauen |

---

## 10. Zusammenfassung: Empfohlenes Preismodell

### 10.1 Preis-Architektur

```
                    STARTER            BUSINESS           ENTERPRISE
Preis/User/Mo:     12 EUR (15*)       19 EUR (24*)       25 EUR (32*)
                   (*Monatsvertrag)

Min. User:         1                  5                   25
Max. User:         15                 50                  200
Storage:           25 GB              100 GB              500 GB
Video:             500 Min, 5 TN      Unbegrenzt, 25 TN  Unbegrenzt, 50 TN
OnlyOffice:        Add-On (+8 EUR)    Inkludiert          Inkludiert
Banking:           --                 Add-On (+15 EUR)    Inkludiert
Newsletter:        --                 Inkludiert          Inkludiert
Support:           E-Mail (24h)       E-Mail+Chat (8h)    Prioritaet (4h)
Custom Fields:     10 pro Modul       50 pro Modul        Unbegrenzt
API-Zugang:        Basis              Voll                Voll + Webhooks
SSO/SAML:          --                 --                  Inkludiert
SLA:               --                 99,0%               99,5%
Onsite-Setup:      1.500 EUR          2.500 EUR           Inkludiert (24 Mo)
```

### 10.2 Preisvergleich-Positionierung

```
Preis-Spektrum (15 User, monatlich):

Billigste Option:    KMU Hub Starter        = 180 EUR
Unser Sweet Spot:    KMU Hub Business       = 285 EUR
                                               |
Zoho One:                                    = 555 EUR
Odoo Enterprise:                             = 480-720 EUR
Pipedrive-Stack:                             = 627 EUR
HubSpot-Stack:                               = 675 EUR
M365 + Dynamics:                             = 1.560 EUR

KMU Hub ist 49-82% guenstiger als der Wettbewerb.
```

### 10.3 Upselling-Strategie

**Phase 1: Land (Starter)**
- Kunde startet mit Basis-Paket (CRM + PM + Zeiterfassung)
- Schneller Nutzen, geringe Einstiegshuerde
- Trigger: "Du nutzt 5 Modules aktiv -- mit Business bekommst du Video-Meetings und OnlyOffice dazu"

**Phase 2: Expand (Starter -> Business)**
- Kunde waechst, braucht mehr User / Features
- OnlyOffice ist der staerkste Treiber ("kein Microsoft 365 mehr noetig")
- Trigger: User-Limit erreicht, oder Kunde fragt nach Office-Editing

**Phase 3: Deepen (Business -> Enterprise / Add-Ons)**
- Kunde will Banking-Integration, mehr Storage, dedizierte Infrastruktur
- Enterprise-Features (SSO, API, SLA) werden bei wachsendem Team relevant
- Trigger: >50 User, Compliance-Anforderungen, Multi-Standort

**Phase 4: Retain (Jaehrliche Vertragsbindung)**
- Rabatt fuer Jahresvertrag (12 vs. 15 EUR Starter, 19 vs. 24 EUR Business)
- Onsite-Einrichtung als Incentive ("kostenlos bei 24-Monats-Business-Vertrag")
- Steigerung der Switching-Costs durch Daten + Prozess-Integration

### 10.4 Self-Hosted als strategische Option

Self-Hosted ist nicht unser Haupt-Umsatztreiber (geringere Marge, mehr Support), aber:

1. **Differentiator:** Weder Zoho, HubSpot noch Pipedrive bieten Self-Hosting
2. **Compliance:** Manche Branchen MUESSEN auf eigener Infrastruktur laufen
3. **Vertrauensbildung:** "Wir haben nichts zu verbergen -- hier ist der Code"
4. **Upselling:** Self-Hosted-Kunden kaufen oft Premium-Support und Updates

**Self-Hosted Preis: 200-1.000 EUR/Mo** (je nach Groesse)
**Oder Einmallizenz: 15.000-40.000 EUR** + jaehrliche Update-Gebuehr

### 10.5 Schweiz-Aufpreis

Fuer den Schweizer Markt (CHF-Pricing):

| Paket | Preis EUR | Preis CHF | Anmerkung |
|-------|-----------|-----------|-----------|
| Starter | 12 EUR/User | 13 CHF/User | ~1:1,08 Wechselkurs + Rundung |
| Business | 19 EUR/User | 21 CHF/User | |
| Enterprise | 25 EUR/User | 28 CHF/User | |
| Schweizer Residenz | +50 EUR/Mo | +55 CHF/Mo | Exoscale-Aufpreis |

**Schweizer Kunden zahlen ~10% mehr** (hoehere Kaufkraft, hoehere Kosten fuer Exoscale-Hosting).

### 10.6 Revenue Forecast (24 Monate)

**Konservatives Szenario (organisches Wachstum):**

| Monat | Kunden | Mix (S/B/E) | MRR | Kosten/Mo | Gewinn/Mo |
|-------|--------|-------------|-----|-----------|-----------|
| 1 | 5 | 4/1/0 | 1.050 EUR | 15.230 EUR | -14.180 EUR |
| 3 | 15 | 10/4/1 | 4.530 EUR | 15.290 EUR | -10.760 EUR |
| 6 | 30 | 18/9/3 | 11.010 EUR | 15.365 EUR | -4.355 EUR |
| 9 | 50 | 30/15/5 | 18.750 EUR | 15.465 EUR | +3.285 EUR |
| 12 | 75 | 45/22/8 | 29.460 EUR | 15.590 EUR | +13.870 EUR |
| 18 | 120 | 72/36/12 | 47.520 EUR | 15.815 EUR | +31.705 EUR |
| 24 | 180 | 108/54/18 | 71.280 EUR | 16.115 EUR | +55.165 EUR |

**ARR nach 24 Monaten: ~855.000 EUR** (bei 180 Kunden)

### 10.7 Empfehlung auf einen Blick

```
1. PREISMODELL:    Per-User, monatlich, 3 Tiers (12/19/25 EUR)
2. USP:            "Alles in einem fuer 12-25 EUR/User -- guenstiger als JEDES Einzel-Tool"
3. LAND-STRATEGIE: Starter (12 EUR) als Einstieg, Upgrade auf Business fuer OnlyOffice
4. MARGE:          >97% Bruttomarge auf Infrastruktur
5. BREAK-EVEN:     ~37 Kunden (~Monat 7-8 nach Launch)
6. CLV:            ~14.700 EUR pro Kunde (36 Monate)
7. MAX CAC:        ~4.900 EUR pro Kunde (inkl. Onsite-Woche)
8. FOKUS-BRANCHEN: Dienstleister (Phase 1), Handwerk (Phase 2), Bau (Phase 3)
9. SELF-HOSTED:    Strategische Option, nicht Haupt-Revenue-Stream
10. SCHWEIZ:       +10% Preisaufschlag, Exoscale-Option als Premium
```

---

## Anhang A: Hetzner Preisliste (Referenz)

| Server-Typ | vCPU | RAM | SSD | Preis/Mo |
|-----------|------|-----|-----|----------|
| CX11 | 1 | 2 GB | 20 GB | ~3,79 EUR |
| CX22 | 2 | 4 GB | 40 GB | ~5,39 EUR |
| CX32 | 4 | 8 GB | 80 GB | ~9,59 EUR |
| CX42 | 8 | 16 GB | 160 GB | ~17,99 EUR |
| **Storage Box BX11** | -- | -- | 1 TB | ~3,81 EUR |
| **Storage Box BX21** | -- | -- | 5 TB | ~10,08 EUR |
| **Storage Box BX31** | -- | -- | 10 TB | ~16,35 EUR |
| **Object Storage** | -- | -- | pro GB | ~0,0126 EUR/GB/Mo |
| **Load Balancer LB11** | -- | -- | -- | ~5,39 EUR |
| **Floating IP** | -- | -- | -- | ~4,51 EUR |
| **Bandwidth** | -- | -- | -- | 20 TB inkl., Overage ~1,19 EUR/TB |

## Anhang B: Externe Dienst-Kosten (Referenz)

| Dienst | Modell | Kosten |
|--------|--------|--------|
| **OnlyOffice Developer** | Jahreslizenz | ~3.000-6.000 EUR/Jahr |
| **FinAPI** | Grundgebuehr + pro Verbindung | ~500 EUR/Mo + ~1 EUR/Bankverbindung |
| **Skribble** | Pro Signatur | EES: ~1 CHF, FES: ~2 CHF, QES: ~3,50 CHF |
| **Brevo** | Pro E-Mail-Volumen | Free bis 9.000/Mo, Starter ab 19 EUR/Mo |
| **CleverReach** | Pro Empfaenger | Free bis 250, ab 15 EUR/Mo |
| **TipTap Pro** | Monatlich | Ab 29 EUR/Mo (nicht noetig fuer v1) |
| **unipdf** (ZUGFeRD PDF/A-3) | Jahreslizenz | ~500 EUR/Jahr |
| **LiveKit** | Self-Hosted | 0 EUR Lizenz, nur Server-Kosten |
| **Nextcloud** | Protokoll-Integration | 0 EUR |
| **DATEV-Export** | Format-Export | 0 EUR |
| **Bexio API** | REST API | 0 EUR |
| **Swiss QR-Code** | Offener Standard | 0 EUR |

---

*Alle Preise in EUR sofern nicht anders angegeben. CHF-Preise zum Kurs ~1,08 umgerechnet. Preise basieren auf Q1 2025 Trainingsdaten und muessen vor Verwendung live verifiziert werden. Dieses Dokument stellt keine verbindliche Preiszusage dar.*
