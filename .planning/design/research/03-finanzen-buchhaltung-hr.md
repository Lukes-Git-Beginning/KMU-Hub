# Marktrecherche: Finanzen, Buchhaltung & HR im DACH-KMU-Segment

> **Autor:** Darien (Design-Recherche fuer KMU Hub)
> **Datum:** 2026-02-16
> **Zielgruppe:** KMUs mit 5-200 Mitarbeitern in Deutschland, Oesterreich, Schweiz
> **Confidence:** MEDIUM — basiert auf Trainingsdaten (Cutoff Mai 2025), keine Live-Verifizierung moeglich (WebSearch/WebFetch nicht verfuegbar). Preise und Versionen koennen sich seit Mai 2025 geaendert haben.

---

## Inhaltsverzeichnis

1. [Buchhaltung](#1-buchhaltung)
2. [Rechnungsstellung](#2-rechnungsstellung)
3. [Lohnabrechnung](#3-lohnabrechnung)
4. [HR-Management](#4-hr-management)
5. [Spesen/Ausgaben](#5-spesenausgaben)
6. [Banking](#6-banking)
7. [Rechtliche Anforderungen — KRITISCH](#7-rechtliche-anforderungen--kritisch)
8. [Funktions-Tiefenanalyse](#8-funktions-tiefenanalyse)
9. [Build vs. Integrate — Strategische Entscheidung](#9-build-vs-integrate--strategische-entscheidung)
10. [Implikationen fuer KMU Hub](#10-implikationen-fuer-kmu-hub)

---

## 1. Buchhaltung

### 1.1 DATEV

| Feld | Detail |
|------|--------|
| **Hersteller** | DATEV eG (Genossenschaft), Nuernberg |
| **Herkunftsland** | Deutschland |
| **Marktanteil (DE)** | ~50-60% aller KMUs arbeiten direkt oder indirekt mit DATEV (ueber Steuerberater). De-facto-Standard. |
| **Marktanteil (CH/AT)** | Minimal. DATEV ist ein rein deutsches Phaenomen. |
| **Preise** | DATEV Unternehmen online: ab ca. 30-40 EUR/Monat (ueber Steuerberater). Vollversion DATEV Kanzlei-Rechnungswesen: nur fuer Steuerberater lizenzierbar, nicht direkt fuer Unternehmen. |
| **Lizenzmodell** | Genossenschaftsmodell — Zugang primaer ueber Steuerberater. KMU nutzt "DATEV Unternehmen online" als Zulieferungs-Tool (Belege hochladen, Bankabgleich), Steuerberater macht die eigentliche Buchung. |
| **DSGVO** | Vollstaendig konform, Rechenzentrum in Nuernberg, nach ISO 27001 zertifiziert |

**Kernfunktionen (DATEV Unternehmen online):**
- Belegerfassung per Scan/Foto mit OCR (DATEV Belegbilderservice)
- Automatischer Bankabgleich (ueber DATEV-Bankanbindung, EBICS/FinTS)
- Kassenbuch digital
- Zahlungsverkehr (SEPA-Ueberweisungen, Lastschriften)
- Dashboard mit offenen Posten, Liquiditaetsvorschau
- DATEV-Export/Import Format (proprietaeres ASCII-Format, Industriestandard in DE)
- Schnittstelle zum Steuerberater: Belege + Kontobewegungen werden direkt in DATEV Kanzlei-Rechnungswesen synchronisiert

**Warum DATEV so dominant ist:**
DATEV ist NICHT primaer eine Software fuer KMUs, sondern das Oekosystem der Steuerberater. ~40.000 Steuerberater-Kanzleien in Deutschland nutzen DATEV. Wenn ein Steuerberater DATEV nutzt (was die Mehrheit tut), MUSS das KMU entweder direkt mit DATEV arbeiten oder eine Software mit DATEV-Export nutzen. Das macht "DATEV-kompatibel" zur Grundanforderung fuer jede Buchhaltungssoftware in Deutschland.

**DATEV-Schnittstelle — was das technisch bedeutet:**
- **DATEV-Format:** CSV-aehnliches Format mit fester Feldstruktur (Buchungssatz-Header, Kontobeschriftungen, etc.)
- **DATEV Connect Online:** REST-API fuer Belegaustausch (OAuth2-basiert)
- **DATEV Rechnungsdatenservice 1.0:** Strukturierte Rechnungsdaten (ZUGFeRD-basiert)
- **Zertifizierung:** DATEV-Partnerschaft erfordert Zertifizierungsprozess (mehrere Monate, technische Pruefung)

---

### 1.2 Lexoffice

| Feld | Detail |
|------|--------|
| **Hersteller** | Haufe-Lexware GmbH & Co. KG (Haufe Group) |
| **Herkunftsland** | Deutschland (Freiburg) |
| **Marktanteil (DE)** | Marktfuehrer bei Cloud-Buchhaltung fuer Kleinunternehmer und kleine KMUs. Geschaetzt 300.000-500.000 Nutzer. |
| **Marktanteil (CH/AT)** | Gering, da stark auf deutsches Steuerrecht fokussiert. |
| **Preise (Stand ~2025)** | S: ~8 EUR/Monat (Rechnungen), M: ~14 EUR/Monat (+Banking), L: ~20 EUR/Monat (+Buchhaltung), XL: ~30 EUR/Monat (+Lohn fuer bis 50 MA) |
| **Lizenzmodell** | Cloud SaaS, monatlich kuendbar |
| **DSGVO** | Konform, Hosting in Deutschland |

**Kernfunktionen:**
- Doppelte Buchfuehrung (automatisiert, KMU muss nicht buchen koennen)
- Automatischer Bankabgleich (ueber FinTS/HBCI + Open Banking)
- USt-Voranmeldung direkt an ELSTER
- SKR03/SKR04 Kontenrahmen (vorkonfiguriert)
- Belegerfassung per OCR (Foto/Scan)
- Jahresabschluss-Vorbereitung (EUeR und Bilanz)
- Angebote, Rechnungen, Mahnwesen
- DATEV-Export (Standard-Format)
- GoBD-zertifiziert (durch unabhaengige Pruefung)
- Offene REST-API (Public API mit OAuth2)
- Integrationen: PayPal, Amazon, Shopify, WooCommerce

**Staerken:** Extrem einfache Bedienung, ideal fuer Nicht-Buchhalter. "Buchhaltung fuer Leute die keine Buchhaltung koennen."
**Schwaechen:** Nur fuer DE sinnvoll. Begrenzte Anpassbarkeit. Ab ~50 Mitarbeitern/komplexeren Strukturen stossen Nutzer an Grenzen.

---

### 1.3 SevDesk

| Feld | Detail |
|------|--------|
| **Hersteller** | sevDesk GmbH |
| **Herkunftsland** | Deutschland (Offenburg) |
| **Marktanteil (DE)** | Zweiter grosser Cloud-Anbieter nach Lexoffice, geschaetzt 150.000-250.000 Nutzer |
| **Preise (Stand ~2025)** | Rechnung: ~9 EUR/Monat, Buchhaltung: ~18 EUR/Monat, Warenwirtschaft: ~40 EUR/Monat |
| **Lizenzmodell** | Cloud SaaS, monatlich/jaehrlich |
| **DSGVO** | Konform, Hosting in Deutschland |

**Kernfunktionen:**
- Doppelte Buchfuehrung
- Automatischer Bankabgleich
- USt-Voranmeldung via ELSTER
- SKR03/SKR04
- Belegerfassung mit KI-OCR (automatische Erkennung von Lieferant, Betrag, MwSt)
- Rechnungen, Angebote, Lieferscheine
- Mahnwesen (3-stufig)
- GoBD-zertifiziert (IDW PS 880)
- DATEV-Export
- REST-API (oeffentlich dokumentiert)
- Warenwirtschaft-Modul (Artikel, Lager)
- Integrationen: Stripe, PayPal, Shopify, WooCommerce, FastBill

**Staerken:** Staerkeres Warenwirtschafts-Modul als Lexoffice. Gute API.
**Schwaechen:** Wie Lexoffice — nur DE. UI etwas weniger poliert als Lexoffice.

---

### 1.4 Bexio

| Feld | Detail |
|------|--------|
| **Hersteller** | bexio AG (gehoert seit 2021 zu Die Mobiliar) |
| **Herkunftsland** | Schweiz (Rapperswil-Jona) |
| **Marktanteil (CH)** | Marktfuehrer bei Cloud-Buchhaltung fuer Schweizer KMUs. ~80.000+ Nutzer (Stand 2024). |
| **Marktanteil (DE/AT)** | Minimal, nicht fuer DE/AT-Recht ausgelegt. |
| **Preise (Stand ~2025)** | Starter: CHF 35/Monat (1 User), Plus: CHF 65/Monat (3 User), Pro: CHF 109/Monat (5 User). Zusaetzliche User: CHF 15-25/Monat. |
| **Lizenzmodell** | Cloud SaaS, jaehrliche Abrechnung (monatlich teurer) |
| **DSGVO/DSG** | DSG-konform (Schweizer Datenschutzgesetz), Hosting in der Schweiz |

**Kernfunktionen:**
- Doppelte Buchfuehrung nach Schweizer OR (Art. 957ff)
- Kontenrahmen KMU (Schweizer Standard-Kontenrahmen)
- MWSt-Abrechnung (Saldo- und Effektiv-Methode)
- Bankabgleich (CAMT.053/054, SIX-Anbindung)
- QR-Rechnungen (Swiss QR-Code, seit 2022 Pflicht)
- Offertenwesen, Rechnungen, Mahnwesen
- Kontaktmanagement (CRM-Light)
- Zeiterfassung
- Lohnabrechnung (Swissdec-zertifiziert)
- Lagerverwaltung
- Projekt-Abrechnung
- REST-API (oeffentlich, gut dokumentiert)
- Banking-Integrationen: PostFinance, UBS, CS, Raiffeisen, ZKB
- LSV+ / Debit Direct Unterstuetzung

**Staerken:** Einzige wirklich vollstaendige Schweizer Cloud-Loesung. Deckt Buchhaltung + Lohn + CRM + Zeiterfassung in einem. Swiss QR-Code nativ. Swissdec-zertifiziert fuer Lohn.
**Schwaechen:** Teurer als DE-Alternativen. UI manchmal etwas "Enterprise"-lastig. Performance bei vielen Buchungen kritisiert.

**KRITISCH fuer KMU Hub:** Bexio ist DER Haupt-Integrations-Kandidat fuer Schweizer Kunden. Die Bexio-API ist gut dokumentiert und stabil. Bereits in LUKE-FEATURE-LIST.md als J1 erfasst.

---

### 1.5 Abacus

| Feld | Detail |
|------|--------|
| **Hersteller** | Abacus Research AG |
| **Herkunftsland** | Schweiz (Wittenbach SG) |
| **Marktanteil (CH)** | Stark bei mittleren und groesseren KMUs (50-500+ MA). ~55.000 Kunden. |
| **Preise** | Nicht oeffentlich. Lizenz + Wartung, typisch CHF 5.000-50.000+ Initialkosten + jaehrliche Wartung. Seit ~2020 auch AbaClik (Cloud, guenstiger). |
| **Lizenzmodell** | Klassisch: On-Premise Lizenz + Wartungsvertrag. Neu: AbaWeb (Cloud). |
| **DSGVO/DSG** | DSG-konform, Schweizer Rechenzentren |

**Kernfunktionen:**
- Vollstaendige Finanzbuchhaltung (Schweizer OR)
- Debitoren/Kreditoren
- Anlagenbuchhaltung
- Lohnbuchhaltung (Swissdec-zertifiziert, Marktfuehrer bei Lohn CH)
- Auftragsbearbeitung
- Projektabrechnung
- Leistungserfassung/Zeiterfassung
- HR-Management (Abacus HR)
- E-Banking (alle Schweizer Banken)
- MWSt-Abrechnung
- Konsolidierung (Konzernbuchhaltung)
- AbaNinja: Rechnungs-App fuer Kleinstunternehmen (gratis)

**Staerken:** Extrem umfangreich, seit 40+ Jahren am Markt, besonders stark bei Lohn. De-facto-Standard fuer Schweizer Lohnabrechnung im Mittelstand.
**Schwaechen:** Teuer, komplex, lange Implementierungszeit. Nicht "mal schnell integrieren".

**KRITISCH fuer KMU Hub:** Abacus-Integration (J2 in LUKE-FEATURE-LIST) ist fuer groessere CH-Kunden wichtig, aber technisch aufwaendig. Abacus hat eine API (AbaConnect), die allerdings weniger modern ist als Bexio.

---

### 1.6 Sage

| Feld | Detail |
|------|--------|
| **Hersteller** | Sage Group plc |
| **Herkunftsland** | UK (Newcastle), DACH-Niederlassungen in Frankfurt, Wien, Zug |
| **Marktanteil (DACH)** | Stark im Mittelstand (50-500 MA). Sage 50 (ehemals PC-Kaufmann) bei kleinen, Sage 100/200/X3 bei mittleren Unternehmen. |
| **Preise** | Sage 50: ab ~30 EUR/Monat. Sage Business Cloud Accounting: ab ~10 EUR/Monat (Basis). Sage HR Suite: ab ~5 EUR/MA/Monat. |
| **Lizenzmodell** | Mix aus Cloud (SaaS) und On-Premise. Sage 50 teilweise noch Desktop. |
| **DSGVO** | Konform (EU-Rechenzentren) |

**Kernfunktionen:**
- Finanzbuchhaltung (DE: GoBD, SKR03/04; AT: RLG; CH: OR)
- Auftragsbearbeitung
- Warenwirtschaft
- Lohnabrechnung (DE: ELSTER, SV-Meldung; CH: Swissdec)
- Anlagenbuchhaltung
- CRM (Sage CRM)
- HR (Sage HR Suite / Sage People)
- DATEV-Schnittstelle

**Staerken:** Breites Produktportfolio, DACH-uebergreifend, lange Historie.
**Schwaechen:** Fragmentiertes Produktportfolio (verschiedene Produkte fuer verschiedene Laender/Groessen). Migration zwischen Sage-Produkten oft schmerzhaft. Cloud-Transformation noch im Gang.

---

### 1.7 Run my Accounts

| Feld | Detail |
|------|--------|
| **Hersteller** | Run my Accounts AG |
| **Herkunftsland** | Schweiz (Stans NW) |
| **Marktanteil (CH)** | Nische, aber wachsend. Primaer fuer Startups und kleine KMUs die Buchhaltung komplett auslagern wollen. |
| **Preise** | Ab CHF 190/Monat (Micro: <100 Belege/Monat). Standard: CHF 390/Monat. Inkl. Buchhalter! |
| **Lizenzmodell** | BPaaS (Business Process as a Service) — Software + menschlicher Buchhalter als Service. |
| **DSGVO/DSG** | DSG-konform, Schweiz |

**Kernfunktionen:**
- Cloud-Buchhaltung (Schweizer OR)
- Automatische Belegverarbeitung
- MWSt-Abrechnung
- Debitoren/Kreditoren
- Bankanbindung (alle Schweizer Banken)
- QR-Rechnungen
- Jahresabschluss
- **USP:** Ein echter Buchhalter prueft und korrigiert. Kunde muss nur Belege hochladen.
- REST-API (dokumentiert)

**Staerken:** "Buchhaltung als Service" — ideal fuer KMUs die keinen eigenen Buchhalter haben/wollen.
**Schwaechen:** Teurer als reine Software. Skaliert preislich weniger gut.

**KRITISCH fuer KMU Hub:** Bereits als J3 in LUKE-FEATURE-LIST. API-Integration sinnvoll fuer CH-Kunden die RmA nutzen.

---

### 1.8 Banana Accounting

| Feld | Detail |
|------|--------|
| **Hersteller** | Banana.ch SA |
| **Herkunftsland** | Schweiz (Lugano) |
| **Marktanteil (CH)** | Beliebt bei Kleinstunternehmen und Vereinen. Eher Tessin/Romandie. |
| **Preise** | Banana Accounting Plus: ab CHF 149/Jahr. Free Version mit Einschraenkungen. |
| **Lizenzmodell** | Desktop-Software + Cloud-Sync. Jahreslizenz. |
| **DSGVO/DSG** | DSG-konform |

**Kernfunktionen:**
- Doppelte Buchfuehrung (CH, DE, AT, IT)
- Kassenbuch
- MWSt-Abrechnung
- Budgetierung
- Vereinsbuchhaltung
- Multi-Waehrung
- Einfache Rechnungen

**Staerken:** Guenstig, mehrsprachig (DE/FR/IT/EN), gut fuer Vereine.
**Schwaechen:** Desktop-First (kein echtes Cloud-Produkt), keine Lohn-Integration, keine moderne API, begrenzte Automatisierung.

**Fuer KMU Hub:** Geringe Relevanz fuer Integration. Kunden die Banana nutzen, sind typischerweise zu klein fuer KMU Hub.

---

## 2. Rechnungsstellung

### 2.1 FastBill

| Feld | Detail |
|------|--------|
| **Hersteller** | FastBill GmbH |
| **Herkunftsland** | Deutschland (Frankfurt) |
| **Marktanteil** | Beliebt bei Freelancern und kleinen Agenturen in DE. |
| **Preise** | Starter: ~9 EUR/Monat, Plus: ~27 EUR/Monat, Premium: ~54 EUR/Monat |
| **Lizenzmodell** | Cloud SaaS |
| **DSGVO** | Konform |

**Kernfunktionen:**
- Rechnungen, Angebote, Gutschriften
- Automatisches Mahnwesen (3-stufig)
- Belegerfassung (OCR)
- Bankanbindung (automatischer Abgleich)
- DATEV-Export
- GoBD-konform
- Zeiterfassung (in hoeherem Tier)
- Online-Banking
- Kundenverwaltung
- API vorhanden

---

### 2.2 Billomat

| Feld | Detail |
|------|--------|
| **Hersteller** | Billomat GmbH (gehoert zu Haufe Group seit 2021) |
| **Herkunftsland** | Deutschland |
| **Marktanteil** | Solide Verbreitung bei kleinen bis mittleren Unternehmen in DE |
| **Preise** | Solo: ~9 EUR/Monat, Business: ~18 EUR/Monat, Enterprise: ~40 EUR/Monat |
| **Lizenzmodell** | Cloud SaaS |
| **DSGVO** | Konform |

**Kernfunktionen:**
- Rechnungen, Angebote, Lieferscheine, Gutschriften
- Mahnwesen (automatisiert, 3 Stufen)
- Belegerfassung
- Online-Banking / Bankabgleich
- Wiederkehrende Rechnungen (Abos)
- Multi-Waehrung
- DATEV-Export
- GoBD-konform
- REST-API (gut dokumentiert)
- Integrationen: Shopify, WooCommerce, PayPal, Stripe

---

### 2.3 SumUp Invoices (ehemals Debitoor)

| Feld | Detail |
|------|--------|
| **Hersteller** | SumUp (hat Debitoor 2022 uebernommen) |
| **Herkunftsland** | UK/DE |
| **Marktanteil** | Primaer fuer Kleinunternehmer, die SumUp-Kartenterminal nutzen |
| **Preise** | SumUp Invoices ist in das SumUp-Oekosystem integriert, Basisversion kostenlos |
| **DSGVO** | Konform |

**Kernfunktionen:**
- Einfache Rechnungserstellung
- Verbindung mit SumUp-Zahlungsterminals
- GoBD-konform (Basis)
- Begrenzter Funktionsumfang im Vergleich zu Lexoffice/SevDesk

**Fuer KMU Hub:** Kaum relevant. Zu basic fuer KMUs mit 5-200 MA.

---

### 2.4 easybill

| Feld | Detail |
|------|--------|
| **Hersteller** | easybill GmbH |
| **Herkunftsland** | Deutschland (Meerbusch) |
| **Marktanteil** | Stark im E-Commerce-Bereich (Automatisierung fuer Amazon, eBay, etc.) |
| **Preise** | Business: ~15 EUR/Monat, Plus: ~20 EUR/Monat, Enterprise (Warenwirtschaft): ~50 EUR/Monat |
| **DSGVO** | Konform |

**Kernfunktionen:**
- Rechnungen, Angebote, Lieferscheine, Gutschriften, Mahnungen
- E-Commerce-Anbindungen (Amazon, eBay, Shopify, WooCommerce, JTL)
- Automatische Rechnungserstellung bei Bestellung
- ZUGFeRD / XRechnung (E-Invoicing, wird ab 2025 in DE Pflicht fuer B2B)
- DATEV-Export
- GoBD-konform (Testat)
- REST-API
- Multi-Waehrung

**Bemerkenswert:** E-Invoicing (ZUGFeRD/XRechnung) wird ab 2025 in Deutschland fuer B2B-Rechnungen schrittweise Pflicht. Easybill ist hier frueh dran.

---

## 3. Lohnabrechnung

### Warum Lohnabrechnung EXTREM komplex ist

**Dies ist das Gebiet mit dem hoechsten rechtlichen Risiko. Falsche Lohnabrechnungen fuehren zu Nachzahlungen, Strafen und Mitarbeiter-Unzufriedenheit.**

#### Deutschland — Komplexitaetstreiber:
1. **Sozialversicherung:** 4 Zweige (Kranken-, Pflege-, Renten-, Arbeitslosenversicherung), jeder mit eigenem Beitragssatz, Beitragsbemessungsgrenzen, Sonderregeln
2. **Lohnsteuer:** Steuerklassen I-VI, Kinderfreibetraege, Kirchensteuer (ja/nein, 8% oder 9% je nach Bundesland), Solidaritaetszuschlag
3. **ELStAM:** Elektronische LohnSteuerAbzugsMerkmale — Arbeitgeber MUSS elektronisch bei der Finanzverwaltung abfragen
4. **Meldungen:** SV-Meldungen an Krankenkassen (DEUES/sv.net), Lohnsteueranmeldung ans Finanzamt (ELSTER), Berufsgenossenschaft, Unfallversicherung
5. **Sonderfaelle:** Minijobs (520-EUR-Grenze), Midijobs (Gleitzone 520,01-2.000 EUR), Kurzarbeit, Mutterschaftsgeld, Elterngeld, betriebliche Altersvorsorge, geldwerte Vorteile (Firmenwagen, Jobticket), Sachbezuege
6. **Jaehrliche Aenderungen:** Beitragssaetze, Bemessungsgrenzen und Grenzwerte aendern sich JEDES JAHR zum 1.1.

#### Schweiz — Komplexitaetstreiber:
1. **Kantonsunterschiede:** 26 Kantone mit unterschiedlichen Steuersaetzen (Quellensteuer!), Familienzulagen-Saetzen, Kinderzulagen
2. **AHV/IV/EO:** Alters- und Hinterlassenenversicherung + Invalidenversicherung + Erwerbsersatzordnung
3. **BVG (Pensionskasse):** Obligatorische berufliche Vorsorge, Koordinationsabzug, Altersgutschriften gestaffelt nach Alter (7% / 10% / 15% / 18%)
4. **UVG (Unfallversicherung):** Berufs- und Nichtberufsunfallversicherung, praemienabhaengig
5. **Quellensteuer:** Fuer Auslaender ohne C-Bewilligung — JEDER Kanton hat eigene Tarife und Tabellen
6. **Swissdec:** Standard fuer elektronische Lohnmeldung an AHV-Ausgleichskassen, Steuerverwaltungen, Unfallversicherer, BVG-Stiftungen — ZERTIFIZIERUNG ERFORDERLICH
7. **Familienzulagen:** Kantonal unterschiedlich (z.B. ZH: CHF 200/Kind, VS: CHF 375/Kind)
8. **13. Monatslohn:** In CH ueblich, muss korrekt verteilt und abgerechnet werden

#### Oesterreich — Komplexitaetstreiber:
1. **Lohnsteuer nach Einkommensteuertarif** (7 Stufen)
2. **Sozialversicherung:** OeGK (Kranken), PV (Pension), UV (Unfall), AV (Arbeitslosen)
3. **Kommunalsteuer:** 3% vom Bruttolohn, kommunal abzufuehren
4. **Dienstgeberbeitrag (DB) und Zuschlag (DZ):** An den FLAF
5. **Sonderzahlungen:** 13./14. Gehalt in AT ueblich und steuerlich beguenstigt (Jahressechstel-Regelung)

---

### 3.1 DATEV Lohn und Gehalt

| Feld | Detail |
|------|--------|
| **Hersteller** | DATEV eG |
| **Markt** | Deutschland — Marktfuehrer |
| **Preise** | Nur ueber Steuerberater. Typisch: 10-25 EUR pro Abrechnung pro Monat (abhaengig vom Steuerberater) |
| **Lizenzmodell** | Steuerberater rechnet fuer KMU ab. KMU liefert nur Daten zu (Arbeitszeiten, Fehlzeiten, Aenderungen). |

**Kernfunktionen:**
- Komplette Lohnabrechnung DE (alle SV-Zweige, alle Steuerklassen)
- Automatische SV-Meldungen (DEUES)
- Automatische Lohnsteueranmeldung (ELSTER)
- ELStAM-Abruf
- Bescheinigungswesen (Lohnsteuerbescheinigung, SV-Bescheinigungen)
- Branchenzuschlaege, Baulohn, Kurzarbeitergeld
- Betriebliche Altersvorsorge
- Integration mit DATEV Personalwirtschaft

**Warum relevant:** Die meisten deutschen KMUs lassen die Lohnabrechnung vom Steuerberater machen. Der Steuerberater nutzt DATEV Lohn und Gehalt. KMU Hub muss also primaer Daten an den Steuerberater LIEFERN (Arbeitszeiten, Urlaub, Krankheit), nicht selbst abrechnen.

---

### 3.2 Personio Payroll

| Feld | Detail |
|------|--------|
| **Hersteller** | Personio SE & Co. KG |
| **Herkunftsland** | Deutschland (Muenchen) |
| **Markt** | DE primaer, expandiert in EU. Stark wachsend. |
| **Preise** | Nicht oeffentlich. Typisch: ab 6-12 EUR/MA/Monat fuer HR, Payroll als Zusatzmodul. |
| **Lizenzmodell** | Cloud SaaS |

**Kernfunktionen (Payroll DE):**
- Vorbereitende Lohnabrechnung (Daten sammeln)
- ODER vollstaendige Lohnabrechnung (seit ~2023, Personio hat eigene Payroll-Engine)
- Automatische SV-Meldungen
- ELSTER-Anbindung
- DATEV-Schnittstelle (fuer Steuerberater-Modell)
- Lohnabrechnungen als PDF fuer Mitarbeiter
- Integriert mit Personio HR (Urlaub, Krankheit, Ueberstunden fliessen direkt ein)

---

### 3.3 Bexio Lohn

| Feld | Detail |
|------|--------|
| **Hersteller** | bexio AG |
| **Markt** | Schweiz |
| **Preise** | Inkludiert in Bexio Pro (CHF 109/Monat), oder als Lohn-Zusatzmodul |

**Kernfunktionen:**
- Schweizer Lohnabrechnung (AHV/IV/EO, UVG, BVG, Quellensteuer)
- Swissdec-zertifiziert (ELM — Einheitliches Lohnmeldeverfahren)
- Alle 26 Kantone
- Automatische Jahresmeldungen
- Lohnausweise (Formular 11)
- 13. Monatslohn
- Familienzulagen
- Spesen-Integration
- PDF-Lohnabrechnungen fuer Mitarbeiter

---

### 3.4 Abacus Lohn

| Feld | Detail |
|------|--------|
| **Hersteller** | Abacus Research AG |
| **Markt** | Schweiz — Marktfuehrer bei Lohn |
| **Preise** | Nicht oeffentlich, typisch CHF 3.000-15.000+ Initial + Wartung |

**Kernfunktionen:**
- Umfassendste Schweizer Lohnloesung am Markt
- Swissdec-zertifiziert (war einer der Ersten)
- Alle 26 Kantone, alle Quellensteuertarife
- Automatische Meldungen an AHV, Steuerverwaltung, UVG, BVG
- Lohnausweis (Formular 11)
- Sozialversicherungsabrechnungen
- Retrograde Berechnungen
- Branchen-spezifische Module (Baulohn, GAV-Pruefung)
- Integration mit Abacus HR, Leistungserfassung, Projekt-Abrechnung

---

### 3.5 Sage HR / Sage Lohn

| Feld | Detail |
|------|--------|
| **Hersteller** | Sage Group |
| **Markt** | DACH-uebergreifend |
| **Preise** | Sage 50 Lohn: ab ~30 EUR/Monat. Sage HR Suite: ab ~5 EUR/MA/Monat |

**Kernfunktionen (DE):**
- Lohnabrechnung DE (alle SV-Zweige, Steuerklassen)
- SV-Meldungen, ELSTER
- ELStAM
- Bescheinigungswesen
- DATEV-Schnittstelle

**Kernfunktionen (CH):**
- Sage Start/200 Lohn: Schweizer Lohnabrechnung, Swissdec-zertifiziert

---

## 4. HR-Management

### 4.1 Personio

| Feld | Detail |
|------|--------|
| **Hersteller** | Personio SE & Co. KG |
| **Herkunftsland** | Deutschland (Muenchen) |
| **Marktanteil** | Marktfuehrer bei HR-Cloud-Software fuer europaeische KMUs. ~10.000+ Kunden. |
| **Preise** | Nicht oeffentlich. Typisch: Core HR ab ~4-6 EUR/MA/Monat, mit Recruiting + Payroll: 8-15 EUR/MA/Monat. Minimum ~50 MA fuer Sales-Kontakt, aber Essential-Plan fuer kleinere. |
| **Lizenzmodell** | Cloud SaaS, Jahresvertrag |
| **DSGVO** | Konform, ISO 27001, Hosting in EU |

**Kernfunktionen:**
- **Stammdatenverwaltung:** Digitale Personalakte, alle MA-Daten zentral
- **Onboarding/Offboarding:** Checklisten, Aufgaben-Workflows
- **Abwesenheitsmanagement:** Urlaub, Krankheit, Sonderurlaub — Antrag/Genehmigung-Workflow, Kalender-Uebersicht, Resturlaub-Tracking
- **Zeiterfassung:** Stempeln per App/Browser, Ueberstunden-Tracking, Arbeitszeitmodelle
- **Recruiting:** Stellenanzeigen, Bewerbermanagement (ATS), Multiposting auf Jobboersen
- **Performance:** Feedbackzyklen, Zielvereinbarungen
- **Gehaltsmanagement:** Gehaltsbaender, Bonus-Tracking
- **Lohnvorbereitung / Payroll:** Siehe oben
- **Reports & Analytics:** HR-KPIs, Headcount, Fluktuation
- **Mitarbeiter-Self-Service:** MA sieht eigene Daten, stellt Antraege, laedt Dokumente hoch
- **Integrationen:** DATEV, Slack, MS Teams, Google Workspace, 200+ Marketplace-Apps
- **API:** REST-API (gut dokumentiert, Webhooks)

**Staerken:** Umfassendstes HR-Tool fuer europaeische KMUs. Sehr gute UX. Starkes Oekosystem.
**Schwaechen:** Preisintransparenz, Mindestgroesse, langfristige Vertraege.

---

### 4.2 HRworks

| Feld | Detail |
|------|--------|
| **Hersteller** | HRworks GmbH |
| **Herkunftsland** | Deutschland (Freiburg) |
| **Marktanteil** | Stark bei deutschen KMUs im Bereich 20-500 MA |
| **Preise** | Ab ~7-10 EUR/MA/Monat (Basis). Module zubuchbar. |
| **DSGVO** | Konform |

**Kernfunktionen:**
- Personalverwaltung (digitale Personalakte)
- Zeiterfassung (mit verschiedenen Arbeitszeitmodellen)
- Abwesenheitsverwaltung
- Reisekostenabrechnung (Staerke! GoBD-konform)
- Bewerbermanagement
- Onboarding
- Gehaltsabrechnung (vorbereitend, DATEV-Export)
- Fuhrparkverwaltung (Fahrtenbuch)
- API vorhanden

**Staerken:** Besonders stark bei Reisekosten und Zeiterfassung. Deutsches Unternehmen, deutsche Compliance.
**Schwaechen:** Weniger international als Personio. UI etwas weniger modern.

---

### 4.3 Kenjo

| Feld | Detail |
|------|--------|
| **Hersteller** | Kenjo GmbH |
| **Herkunftsland** | Deutschland (Berlin) |
| **Marktanteil** | Wachsend, primaer DE/ES |
| **Preise** | Starter: ab ~5 EUR/MA/Monat, Growth: ~7 EUR/MA/Monat |
| **DSGVO** | Konform |

**Kernfunktionen:**
- Personalverwaltung
- Zeiterfassung
- Abwesenheitsmanagement
- Performance-Reviews
- Recruiting
- Onboarding
- Organigramm
- Schichtplanung (Staerke!)
- Gehaltsmanagement
- API vorhanden

**Staerken:** Gutes Preis-Leistungs-Verhaeltnis, speziell fuer Teams mit Schichtarbeit.

---

### 4.4 Jacando

| Feld | Detail |
|------|--------|
| **Hersteller** | jacando AG |
| **Herkunftsland** | Schweiz (Basel) |
| **Marktanteil** | Nische im Schweizer Markt |
| **Preise** | Ab CHF 3/MA/Monat (Basis), Module zubuchbar |
| **DSGVO/DSG** | DSG-konform |

**Kernfunktionen:**
- Personalverwaltung
- Zeiterfassung
- Abwesenheitsmanagement
- Recruiting
- Onboarding/Offboarding
- Organigramm
- Mitarbeiter-Self-Service

**Staerken:** Guenstig, Schweizer Anbieter.
**Schwaechen:** Weniger umfangreich als Personio, kleineres Team.

---

### 4.5 BambooHR

| Feld | Detail |
|------|--------|
| **Hersteller** | BambooHR LLC |
| **Herkunftsland** | USA (Utah) |
| **Marktanteil (DACH)** | Wird von internationalen Teams/Startups in DACH genutzt, aber nicht marktfuehrend |
| **Preise** | Ab ca. 6-8 USD/MA/Monat |
| **DSGVO** | Hat EU-Hosting-Option, aber US-Unternehmen = Privacy-Shield-Problematik |

**Kernfunktionen:**
- Personalverwaltung, Abwesenheiten, Zeiterfassung
- Recruiting (ATS), Onboarding
- Performance Management
- Berichte
- App

**Schwaechen fuer DACH:** Keine DATEV-Integration, keine Swissdec, kein deutsches Steuerrecht, Privacy-Bedenken.

**Fuer KMU Hub:** Nicht relevant als Integrations-Ziel. KMU Hub Kunden mit EU-Datensouveraenitaet wuerden kein US-HR-Tool nutzen.

---

## 5. Spesen/Ausgaben

### 5.1 Circula

| Feld | Detail |
|------|--------|
| **Hersteller** | Circula GmbH |
| **Herkunftsland** | Deutschland (Berlin) |
| **Marktanteil** | Wachsend, spezialisiert auf DACH |
| **Preise** | Ab ~6-8 EUR/MA/Monat |
| **DSGVO** | Konform |

**Kernfunktionen:**
- Spesenabrechnung per App (Foto von Beleg, OCR-Erfassung)
- Reisekostenabrechnung (deutsche Pauschalen, Verpflegungsmehraufwand)
- Firmenkreditkarten-Integration
- Sachbezuege/Benefits-Verwaltung (Essensgeld, Mobilitaetsbudget, Internetpauschale)
- Genehmigungsworkflows
- DATEV-Export (direkt in DATEV Unternehmen online)
- GoBD-konform
- Integration: Personio, DATEV, SAP

**Staerken:** Spezialisiert auf deutsches Recht (Pauschalen, Sachbezuege). Sehr starke DATEV-Integration.

---

### 5.2 Spendesk

| Feld | Detail |
|------|--------|
| **Hersteller** | Spendesk SAS |
| **Herkunftsland** | Frankreich (Paris) |
| **Marktanteil** | Europaweit wachsend, DE als grosser Markt |
| **Preise** | Ab ca. 49 EUR/Monat (Basis), Enterprise: individuell |
| **DSGVO** | Konform |

**Kernfunktionen:**
- Virtuelle und physische Firmenkreditkarten
- Spesenabrechnung (App + OCR)
- Rechnungsmanagement (Lieferantenrechnungen)
- Genehmigungsworkflows
- Budgets pro Abteilung/Projekt
- Echtzeit-Ausgabenueberblick
- Buchhaltungs-Integrationen (DATEV, Xero, QuickBooks, Sage)
- API

**Staerken:** Kombiniert Karten + Spesen + Rechnungen. Gut fuer Ausgabenkontrolle.

---

### 5.3 Moss (ehemals Spendit)

| Feld | Detail |
|------|--------|
| **Hersteller** | Moss GmbH (ehemals Spendit) |
| **Herkunftsland** | Deutschland (Berlin) |
| **Preise** | Firmenkarten kostenlos, Software ab ~7 EUR/Aktiver User/Monat |
| **DSGVO** | Konform |

**Kernfunktionen:**
- Virtuelle und physische Firmenkreditkarten (VISA)
- Echtzeit-Transaktionsuebersicht
- Belegerfassung (OCR)
- Genehmigungsworkflows
- Budgets und Limits pro Karte/Abteilung
- DATEV-Export (direkte Anbindung)
- GoBD-konform
- Rechnungsfreigabe-Workflow

---

### 5.4 Pleo

| Feld | Detail |
|------|--------|
| **Hersteller** | Pleo Technologies |
| **Herkunftsland** | Daenemark (Kopenhagen) |
| **Preise** | Starter: kostenlos (3 User), Pro: ab ~4 EUR/User/Monat |
| **DSGVO** | Konform (EU-Unternehmen) |

**Kernfunktionen:**
- Firmenkreditkarten (VISA)
- Echtzeit-Ausgabenverfolgung
- Beleg-Erfassung per App
- Genehmigungsworkflows
- Kilometer-Tracking
- Abos-Verwaltung
- Integrations: DATEV, Lexoffice, SevDesk, Xero, QuickBooks
- API

---

## 6. Banking

### 6.1 Banken-Schnittstellen im DACH-Raum

#### Deutschland: FinTS/HBCI + PSD2

| Standard | Details |
|----------|---------|
| **FinTS (ehemals HBCI)** | Deutsche Standard-Schnittstelle fuer Online-Banking. Wird von allen deutschen Banken unterstuetzt. Protokoll-basiert (XML/HTTPS). Erlaubt: Kontoauszuege abrufen, Ueberweisungen, Dauerauftraege. Sicherheit: PIN/TAN-Verfahren. |
| **PSD2 / Open Banking** | EU-Richtlinie. Banken MUESSEN APIs bereitstellen (Access to Account). In der Praxis nutzen die meisten DACH-Tools Aggregatoren statt direkte Bank-APIs. |
| **EBICS** | Electronic Banking Internet Communication Standard. Fuer groessere Unternehmen und den Zahlungsverkehr mit Banken. Batch-Auftraege (SEPA-Dateien), Kontoauszuege (CAMT.053). |
| **SEPA** | Single Euro Payments Area. Ueberweisungen (SCT), Lastschriften (SDD). XML-Format (pain.001, pain.008). Standardisiert in ganz Europa. |
| **Banking-Aggregatoren** | FinAPI (Muenchen), Tink (Stockholm), Plaid — stellen normalisierte APIs ueber alle Banken bereit. Lexoffice/SevDesk/etc. nutzen typischerweise solche Dienste. |

#### Schweiz: SIX + EBICS + LSV+

| Standard | Details |
|----------|---------|
| **SIX Financial Information** | Schweizer Finanzinfrastruktur. Stellt Bankclearing, Interbanken-Zahlungsverkehr bereit. |
| **EBICS (CH)** | Verbreiteter Standard fuer Business-Banking in CH. Swiss EBICS-Profil. |
| **CAMT (053/054)** | Kontoauszugs-Format (ISO 20022). Standard in CH seit 2018. |
| **pain.001/pain.002/pain.008** | Zahlungsauftrags-Formate (ISO 20022). CH hat vollstaendig auf ISO 20022 migriert (SIX-Mandat). |
| **QR-Rechnung** | Seit 2022 PFLICHT (ersetzt alten Einzahlungsschein). Swiss QR-Code auf Rechnung, scannbar per Banking-App. |
| **LSV+ (Lastschriftverfahren)** | Schweizer Pendant zum SEPA-Lastschriftverfahren. Fuer wiederkehrende Zahlungen (Miete, Abos). Erfordert Inkassovereinbarung mit Bank. |
| **Debit Direct (DD)** | Neueres Lastschriftverfahren (SIX), ersetzt schrittweise LSV+ im Rahmen von ISO 20022. |
| **eBill** | Elektronische Rechnungsstellung in CH. Rechnung direkt ins E-Banking des Kunden. Ueber SIX/Paynet. |

#### Oesterreich: ELBA/s + PSD2

| Standard | Details |
|----------|---------|
| **ELBA/s** | Electronic Banking-System der oesterreichischen Banken |
| **PSD2** | EU-Standard, wie in DE |
| **SEPA** | Standard fuer Zahlungsverkehr |

### 6.2 Banking-APIs fuer KMU Hub

| Option | Beschreibung | Aufwand |
|--------|-------------|---------|
| **FinAPI** (DE-Fokus) | Aggregator fuer 4.000+ Banken in DACH. REST-API, PSD2-lizenziert. Kontoauszuege, Zahlungsinitiation, Bankabgleich. Ab ca. 500 EUR/Monat. | MITTEL |
| **Tink** (EU-weit) | Von Visa uebernommen. PSD2-Aggregator. Breitere EU-Abdeckung, aber weniger DACH-spezifisch. | MITTEL |
| **Direkte Bank-APIs** | PostFinance API, UBS API, etc. — jede Bank einzeln. | HOCH (pro Bank) |
| **EBICS-Implementierung** | Eigene EBICS-Anbindung. Komplex (Krypto, Schluessel-Management, Protokoll). | SEHR HOCH |
| **Empfehlung** | FinAPI als Aggregator fuer DE/AT, ergaenzt um direkte Schweizer Bank-APIs (PostFinance, etc.) fuer CH. | |

---

## 7. Rechtliche Anforderungen — KRITISCH

### 7.1 GoBD (Deutschland)

**GoBD = Grundsaetze zur ordnungsmaessigen Fuehrung und Aufbewahrung von Buechern, Aufzeichnungen und Unterlagen in elektronischer Form sowie zum Datenzugriff**

Erlassen vom Bundesministerium der Finanzen. Gilt fuer ALLE Unternehmen und Freiberufler in Deutschland.

#### Was GoBD konkret fordert:

| Anforderung | Was das bedeutet | Warum "selber bauen" problematisch ist |
|-------------|-----------------|----------------------------------------|
| **Nachvollziehbarkeit** | Jede Buchung muss fuer einen Dritten (Pruefer) nachvollziehbar sein. Belegprinzip: Keine Buchung ohne Beleg. | Audit-Trail muss lueckenlos sein |
| **Unveraenderbarkeit** | Einmal gebuchte Daten duerfen NICHT geaendert oder geloescht werden. Korrekturen nur durch Stornierung + Neubuchung. | Kein UPDATE/DELETE auf Buchungssaetzen. Append-only. |
| **Vollstaendigkeit** | Alle Geschaeftsvorfaelle muessen lueckenlos erfasst sein. Keine Luecken in Belegnummern. | Laufende Nummerierung ohne Luecken |
| **Zeitgerechte Buchung** | Buchungen muessen zeitnah erfolgen (Grundbuchaufzeichnungen: sofort; periodengerecht: spaetestens Ende Folgemonat). | Timestamps, keine nachtraeglichen Aenderungen |
| **Ordnung** | Systematische Erfassung nach Kontenrahmen (SKR03/SKR04 in DE). | Muss korrekte Konten implementieren |
| **Aufbewahrung** | 10 Jahre fuer Buchungsbelege, 6 Jahre fuer Geschaeftsbriefe. Im URSPRUENGLICH empfangenen Format. | Archivierungssystem mit Integritaetsschutz |
| **Verfahrensdokumentation** | Unternehmen MUSS dokumentieren WIE die Buchhaltung technisch funktioniert — Datenfluss, Kontrollen, Archivierung. | Dokumentation fuer jede eigene Software |
| **Datenzugriff** | Finanzverwaltung hat das Recht auf Datenzugriff (Z1: Nur Lesen, Z2: Maschinelle Auswertung, Z3: Datentraegerueberlassung). Muss GDPdU/GoBD-Export liefern koennen. | Export-Funktion fuer Steuerpruefer |

#### GoBD-Zertifizierung:

Es gibt KEINE offizielle staatliche GoBD-Zertifizierung. Was existiert:

- **IDW PS 880:** Pruefstandard des Instituts der Wirtschaftspruefer. Wirtschaftspruefer pruefen die Software auf GoBD-Konformitaet und stellen ein Testat aus. Das ist der Goldstandard. Lexoffice, SevDesk, FastBill haben dieses Testat.
- **Selbstdeklaration:** Hersteller erklaert GoBD-Konformitaet in einer Verfahrensdokumentation. Rechtlich moeglich, aber weniger vertrauenswuerdig.
- **BSI-Zertifizierung:** Nicht spezifisch fuer GoBD, aber fuer IT-Sicherheit relevant.

**FAZIT:** Es ist NICHT illegal, eigene Buchhaltungssoftware ohne IDW-Testat zu nutzen. ABER: Bei einer Steuerpruefung muss das Unternehmen nachweisen, dass die Software GoBD-konform ist. Mit eigenem Tool hat man die Beweislast. Mit zertifiziertem Tool nicht. Das Risiko faellt auf den KMU Hub-Kunden.

---

### 7.2 Schweizer OR (Obligationenrecht) — Buchfuehrungspflichten

#### Art. 957 OR — Wer ist buchfuehrungspflichtig?
- Einzelunternehmen und Personengesellschaften mit >500.000 CHF Umsatz/Jahr
- Juristische Personen (AG, GmbH, Vereine, Stiftungen) — IMMER

#### Art. 957a OR — Grundsaetze ordnungsmaessiger Buchfuehrung
- Vollstaendige, wahrheitsgetreue und systematische Erfassung aller Geschaeftsvorfaelle
- Belegnachweis fuer jede Buchung
- Klarheit
- Zweckmaessigkeit (angepasst an Groesse/Komplexitaet)
- Nachpruefbarkeit

#### Art. 958 OR — Rechnungslegung
- Jahresrechnung: Bilanz, Erfolgsrechnung, Anhang
- Fuer groessere Unternehmen: zusaetzlich Geldflussrechnung, Lagebericht

#### Art. 958f OR — Aufbewahrung
- **10 Jahre** Aufbewahrungspflicht fuer Geschaeftsbuecher, Buchungsbelege, Geschaeftsbericht
- Aufbewahrung auf Papier ODER elektronisch (wenn Unversehrtheit und Lesbarkeit gewaehrleistet)
- Elektronische Aufbewahrung: Muss Echtheit und Integritaet nachweisen koennen

#### MWSt-Abrechnung Schweiz:
- Quartalsweise oder halbjaehrlich (je nach Abrechnungsmethode)
- **Effektive Methode:** Vorsteuerabzug, aber detaillierte Buchfuehrung noetig
- **Saldosteuersatz-Methode:** Vereinfacht, pauschaler Satz je nach Branche (0.1% bis 6.5%)
- Elektronische Einreichung ueber ESTV Portal (oder Swissdec fuer Lohn)
- **QR-Referenznummer:** Seit Juni 2020 neues QR-Rechnungsformat, ESR-Referenzen abgeloest

---

### 7.3 Steuer-Schnittstellen im Detail

#### ELSTER (Deutschland)
- **Was:** Elektronische Steuererklaerung. Pflicht fuer Umsatzsteuer-Voranmeldung, Lohnsteueranmeldung, Jahres-Steuererklaerung.
- **Technisch:** ELSTER-API (ERiC = ELSTER Rich Client). C-Bibliothek, die in Software eingebunden wird. Oder: Ueber Steuerberater-Software (DATEV, Lexware, etc.).
- **Zertifizierung:** Software muss bei ELSTER registriert/zugelassen werden. ERiC-Integration ist technisch moeglich, aber aufwaendig (Zertifikat-Management, XML-Schemata, regelmaessige Updates).
- **Fazit fuer KMU Hub:** Eigene ELSTER-Integration waere moeglich, aber der Aufwand steht in keinem Verhaeltnis. Besser: Export-Format das der Steuerberater in DATEV importieren kann.

#### FinanzOnline (Oesterreich)
- **Was:** Oesterreichisches Pendant zu ELSTER. Pflicht fuer Umsatzsteuervoranmeldung (UVA), Jahressteuererklaerung.
- **Technisch:** Web-Service-Schnittstelle (SOAP/XML). Registrierung erforderlich.
- **Fuer KMU Hub:** Aehnlich wie ELSTER — Integration moeglich aber aufwaendig.

#### MWSt-Abrechnung Schweiz (ESTV)
- **Was:** Mehrwertsteuer-Abrechnung an die Eidgenoessische Steuerverwaltung.
- **Technisch:** Online-Portal der ESTV. Seit 2024 auch elektronische Einreichung moeglich. Kein standardisiertes API wie ELSTER.
- **Fuer KMU Hub:** Export der MWSt-relevanten Daten im richtigen Format. Abrechnung selbst meist manuell oder ueber Treuhänder.

---

### 7.4 Aufbewahrungspflichten — Zusammenfassung

| Land | Buchungsbelege | Geschaeftsbriefe | Jahresabschluss | Personalakten (nach Austritt) |
|------|---------------|-----------------|-----------------|-------------------------------|
| **Deutschland** | 10 Jahre | 6 Jahre | 10 Jahre | 3 Jahre (Klagehemmung) + 10 Jahre (Lohnunterlagen) |
| **Schweiz** | 10 Jahre | 10 Jahre | 10 Jahre | 5 Jahre (Arbeitsrecht) + 10 Jahre (Lohnausweise) |
| **Oesterreich** | 7 Jahre | 7 Jahre | 7 Jahre | 3 Jahre + 7 Jahre (Lohn) |

**Kritisch:** "Im Originalformat" bedeutet: Ein als PDF empfangener Beleg muss als PDF aufbewahrt werden, nicht als Screenshot oder Ausdruck-Scan. Ein Papierbeleg kann gescannt werden, wenn das Scanverfahren dokumentiert ist (Verfahrensdokumentation!).

---

### 7.5 E-Invoicing — Neue Pflichten ab 2025/2026

**Deutschland (E-Rechnungspflicht B2B):**
- Ab 01.01.2025: Alle Unternehmen muessen E-Rechnungen EMPFANGEN koennen
- Ab 01.01.2027: Unternehmen mit >800.000 EUR Umsatz muessen E-Rechnungen SENDEN
- Ab 01.01.2028: ALLE Unternehmen muessen E-Rechnungen senden
- Formate: ZUGFeRD 2.x oder XRechnung (EN 16931)
- **ZUGFeRD:** Hybridformat — PDF mit eingebettetem XML. Menschenlesbar + maschinenlesbar.
- **XRechnung:** Reines XML-Format (kein PDF).

**Schweiz:**
- Noch keine Pflicht fuer B2B, aber eBill (SIX) waechst stark
- EU-ViDA-Richtlinie (VAT in the Digital Age) koennte indirekt Einfluss haben

**Fuer KMU Hub:** E-Invoicing-Support (ZUGFeRD/XRechnung) ist ein MUSS fuer 2027+. Unsere Rechnungsfunktion (finance.ts) muss das koennen.

---

## 8. Funktions-Tiefenanalyse

### Was wird gebraucht und was kann KMU Hub selbst?

| Funktion | Wird gebraucht? | Warum? | Selbst bauen moeglich? | KMU Hub Status |
|----------|-----------------|--------|------------------------|----------------|
| **Rechnungen erstellen** | JA, Kernfunktion | Jedes KMU stellt Rechnungen | JA — relativ straightforward | VORHANDEN (finance.ts: Invoices + LineItems + PDF) |
| **Angebote erstellen** | JA | Standardprozess im B2B | JA — aehnlich wie Rechnungen | VORHANDEN (type: 'quote' in finance.ts) |
| **Mahnwesen (3-stufig)** | JA | Jedes KMU hat saeumige Zahler | JA — regelbasiert, machbar | VORHANDEN (dunnings in finance.ts) |
| **Zahlungseingaenge verbuchen** | JA | Offene-Posten-Verwaltung | JA — Matching-Logik | VORHANDEN (recordPayment in finance.ts) |
| **DATEV-Export** | JA (DE) | Steuerberater braucht Daten | JA — CSV-Format ist dokumentiert | GEPLANT (ExportDialog.tsx vorhanden) |
| **QR-Rechnung (CH)** | JA (CH) | Seit 2022 Pflicht in CH | JA — Swiss QR-Code Spec ist offen | FEHLT |
| **ZUGFeRD/XRechnung** | JA (DE, ab 2027) | Wird Pflicht fuer B2B | MITTEL — XML-Schema komplex aber dokumentiert | FEHLT |
| **MWSt-Saetze (DE/CH/AT)** | JA | Laenderspezifisch | JA — Konfigurierbar | TEILWEISE (nur CH-Saetze in finance.ts: 7.7%, 8.1%) |
| **Bankabgleich** | NICE-TO-HAVE | Spart viel Zeit | SCHWIERIG — braucht Banking-API | FEHLT |
| **Belegerfassung (OCR)** | NICE-TO-HAVE | Digitalisierung | SCHWIERIG — OCR-Engine noetig | FEHLT |
| **Doppelte Buchfuehrung** | NEIN (nicht selbst) | Zu komplex, zu riskant | NEIN — lieber integrieren | NICHT GEPLANT |
| **USt-Voranmeldung** | NEIN (nicht selbst) | ELSTER-Integration zu aufwaendig | NEIN — Steuerberater | NICHT GEPLANT |
| **Jahresabschluss** | NEIN (nicht selbst) | Wirtschaftspruefer/Steuerberater | NEIN — definitiv nicht | NICHT GEPLANT |
| **Kontierung (SKR03/04)** | NEIN (nicht selbst) | Buchhaltungs-Knowhow noetig | NEIN — integrieren | NICHT GEPLANT |
| **Lohnabrechnung** | NEIN (nicht selbst) | Rechtlich zu riskant | NEIN — IMMER integrieren | NICHT GEPLANT |
| **SV-Meldungen** | NEIN | Extrem komplex | NEIN | NICHT GEPLANT |
| **Urlaub/Abwesenheiten** | JA | Basis-HR | JA — relativ einfach | VORHANDEN (team.ts: HRRequest) |
| **Mitarbeiterdaten** | JA | Stammdaten zentral | JA — CRUD | VORHANDEN (team.ts: TeamMember) |
| **Zeiterfassung** | JA | Kernfunktion fuer KMUs | JA — Timer + Eintraege | VORHANDEN (timetracking.ts) |
| **Organigramm** | JA | Organisationsstruktur | JA — Tree-Darstellung | TEILWEISE (Departments vorhanden) |
| **Spesenabrechnung** | JA | Mitarbeiter-Ausgaben | JA — Antrag/Genehmigung | VORHANDEN (expenses in finance.ts) |
| **Lohn-Vorbereitung** | JA | Daten fuer Steuerberater/Lohntool | JA — Export | TEILWEISE (payroll in team.ts, aber nur Anzeige) |

---

## 9. Build vs. Integrate — Strategische Entscheidung

### 9.1 Buchhaltung: INTEGRIEREN, nicht selbst bauen

**Empfehlung: DATEV-Export (DE) + Bexio-API (CH) + allgemeiner CSV-Export**

**Warum NICHT selbst bauen:**
1. **GoBD-Konformitaet** erfordert IDW PS 880 Testat (ca. 15.000-50.000 EUR Pruefungskosten, jedes Jahr neu)
2. **Doppelte Buchfuehrung** korrekt zu implementieren ist ein eigenes Software-Projekt (Kontenrahmen, Buchungslogik, Abschlussbuchungen, Abgrenzungen, Saldenvortrag)
3. **Rechtliche Haftung:** Wenn KMU Hub's Buchhaltung einen Fehler hat und der Kunde bei der Steuerpruefung Nachzahlungen bekommt, haftet potentiell KMU Hub
4. **Jaehrliche Aenderungen:** Steuersaetze, Bemessungsgrenzen, Formular-Aenderungen — permanenter Wartungsaufwand
5. **Kein Alleinstellungsmerkmal:** Kunden erwarten Integration, nicht Ersatz

**Was KMU Hub STATTDESSEN tun soll:**
- **DATEV-Export-Format** implementieren (CSV mit definierten Feldern — ist dokumentiert, machbar in 2-4 Tagen)
- **Bexio-REST-API** anbinden (OAuth2, gut dokumentiert — 2-4 Wochen fuer sinnvolle Integration)
- **Abacus-AbaConnect** anbinden (fuer groessere CH-Kunden — 2-4 Wochen)
- **Run my Accounts-API** anbinden (REST, fuer CH-Kunden — 1-2 Wochen)
- **Generischer CSV/JSON-Export** fuer andere Buchhaltungstools

### 9.2 Rechnungsstellung: SELBST BAUEN

**Empfehlung: Eigene Rechnungsstellung, ergaenzt um Export-Schnittstellen**

**Warum selbst bauen realistisch ist:**
1. **Bereits vorhanden** (finance.ts hat Invoices, LineItems, Zahlungen, Mahnwesen)
2. **Keine Buchfuehrungspflicht** — Rechnungen erstellen ist NICHT Buchhaltung. Die Buchung der Rechnung passiert in der Buchhaltungssoftware.
3. **PDF-Generierung** ist technisch machbar (wkhtmltopdf, Puppeteer, oder Go-Libraries)
4. **QR-Rechnung (CH)** hat offene Specs (Swiss QR-Code ist gut dokumentiert)
5. **ZUGFeRD/XRechnung** ist aufwaendiger, aber es gibt Libraries (z.B. `go-zugferd`, Factur-X)

**Was noch gebaut werden muss:**
- QR-Rechnung fuer CH (Swiss QR-Code Generator)
- ZUGFeRD PDF/A-3 Generation (fuer DE ab 2027)
- XRechnung XML-Export (fuer DE ab 2027)
- Laenderspezifische MWSt-Saetze (DE: 19%/7%, CH: 8.1%/2.6%/3.8%, AT: 20%/10%/13%)
- Rechnungsnummern-Logik (lueckenlos, konfigurierbar)
- E-Mail-Versand
- PDF-Archivierung (fuer Aufbewahrungspflicht)

### 9.3 Lohnabrechnung: IMMER INTEGRIEREN — niemals selbst bauen

**Empfehlung: KMU Hub liefert Lohn-relevante Daten, die eigentliche Abrechnung macht DATEV/Bexio/Abacus**

**Warum Lohn NIEMALS selbst bauen:**
1. **Rechtliche Haftung:** Falsche Loehne = Nachzahlungen bei SV-Traegern, Finanzamt, Mitarbeiter-Klagen. Kein Startup kann sich das leisten.
2. **Komplexitaet:**
   - DE: ~6 SV-Zweige x 3+ Beitragsgruppen x 6 Steuerklassen x 16 Bundeslaender x jaehrliche Aenderungen = tausende Rechenregeln
   - CH: 26 Kantone x unterschiedliche Quellensteuertarife x AHV/BVG/UVG/KTG = nochmal tausende Regeln
3. **Zertifizierung:** Swissdec-Zertifizierung (CH) dauert Monate und kostet 5-stellig. ELStAM-Anbindung (DE) erfordert ELSTER-Registrierung.
4. **Updates:** Beitragssaetze aendern sich JEDES JAHR zum 1.1. Quellensteuer-Tarife kommen JEDEN Dezember neu. Wer zu spaet updated, berechnet falsche Loehne.
5. **Personio hat 8+ Jahre und 200+ Mio EUR VC gebraucht** um ihre eigene Payroll-Engine zu bauen. Das sagt alles.

**Was KMU Hub stattdessen tun soll:**
- **Lohn-Vorbereitung:** Arbeitszeitdaten, Abwesenheiten, Ueberstunden, Spesen exportieren
- **DATEV-Lohn-Export:** Strukturiertes Format fuer Steuerberater (Stundenzettel, Fehlzeiten)
- **Bexio-Lohn-API:** Zeitdaten an Bexio uebergeben
- **Lohnzettel-Anzeige:** Vom Lohntool generierte PDFs im KMU Hub anzeigen (Import)
- **Payroll-Daten in team.ts** sind nur zur ANZEIGE, nicht zur Berechnung!

### 9.4 HR Basis: SELBST BAUEN (mit Grenzen)

**Empfehlung: Kernfunktionen selbst, spezialisierte Funktionen integrieren**

**Was KMU Hub selbst bauen kann und soll:**

| Funktion | Machbarkeit | Begruendung |
|----------|------------|-------------|
| Mitarbeiterverwaltung (Stammdaten) | EINFACH | Standard-CRUD, bereits vorhanden |
| Urlaub/Abwesenheiten (Antrag/Genehmigung) | EINFACH | Workflow-basiert, bereits vorhanden |
| Zeiterfassung (Timer, Eintraege) | MITTEL | Bereits vorhanden, funktioniert |
| Schichtplanung | MITTEL | Bereits vorhanden (SchichtenPage) |
| Organigramm/Departments | EINFACH | Tree-Struktur |
| Onboarding-Checklisten | EINFACH | Task-Listen, konfigurierbar |
| Einfache Spesenabrechnung | EINFACH | Antrag + Genehmigung + Beleg, bereits vorhanden |
| Digitale Personalakte | MITTEL | Dokumente pro MA, Zugriffsrechte |
| Schulungsverwaltung | EINFACH | CRUD + Zuordnungen, bereits vorhanden (trainings in team.ts) |

**Was KMU Hub NICHT selbst bauen soll:**

| Funktion | Warum nicht |
|----------|------------|
| Recruiting/ATS | Eigenes Produkt (Indeed, Personio, Greenhouse) |
| Performance Reviews (komplex) | Spezialisierte Tools besser (Personio, 15Five) |
| Gehaltsbaender/Compensation | Zu viele Variablen, Benchmark-Daten noetig |
| Compliance-Tracking (DE: Arbeitszeitgesetz) | Rechtlich komplex, aendert sich |

---

## 10. Implikationen fuer KMU Hub

### 10.1 Architektur-Entscheidung: Integration Hub

KMU Hub soll sich als **Integration Hub** positionieren, nicht als Buchhaltungs-Ersatz:

```
[KMU Hub]
    |
    |-- Eigene Staerken: CRM, Projekte, Chat, Meetings, Dokumente, HR-Basis, Rechnungen
    |
    |-- Integrations-Schicht:
    |       |-- DATEV (DE): Export-Format fuer Steuerberater
    |       |-- Bexio (CH): REST-API bidirektional
    |       |-- Abacus (CH): AbaConnect
    |       |-- Run my Accounts (CH): REST-API
    |       |-- Banking: FinAPI (DE/AT) + CH-Banken
    |       |-- Generisch: CSV/JSON-Export
    |
    |-- Nicht anfassen: Doppelte Buchfuehrung, Lohnberechnung, Steuererklaerung
```

### 10.2 Konkrete TODOs fuer die Roadmap

#### Phase-Empfehlungen:

**Frueh (Core):**
- MWSt-Saetze multi-country machen (DE 19%/7%, CH 8.1%/2.6%/3.8%, AT 20%/10%/13%) -> finance.ts
- Rechnungsnummern-Logik (konfigurierbar, lueckenlos)
- DATEV-Export Format implementieren (CSV)
- PDF-Generierung fuer Rechnungen/Angebote

**Mittel (Differenzierung):**
- Bexio-API Integration (OAuth2, Kontakte + Rechnungen sync)
- Swiss QR-Rechnung Generator
- Lohn-Vorbereitung-Export (Zeitdaten, Abwesenheiten -> DATEV-Lohn oder Bexio-Lohn)
- Banking-Aggregator anbinden (FinAPI fuer automatischen Bankabgleich)

**Spaet (Nice-to-have):**
- Abacus AbaConnect Integration
- Run my Accounts API Integration
- ZUGFeRD/XRechnung (wird erst 2027 fuer groessere, 2028 fuer alle Pflicht)
- OCR-Belegerfassung (kann auch ueber Partner-Tool laufen)
- eBill (CH) Integration

### 10.3 Aktuelle Luecken in KMU Hub (finance.ts / team.ts)

| Luecke | Prioritaet | Aufwand |
|--------|-----------|---------|
| MWSt-Saetze nur CH (8.1%, 7.7%) — DE/AT fehlt | HOCH | KLEIN (Config) |
| Keine QR-Rechnung fuer CH | HOCH (CH-Kunden) | MITTEL |
| Kein DATEV-Export-Format | HOCH (DE-Kunden) | MITTEL |
| Kein PDF-Generator (Rechnungen) | HOCH | MITTEL |
| Keine Waehrungs-Unterstuetzung (EUR neben CHF) | HOCH | KLEIN |
| Kein ZUGFeRD/XRechnung | MITTEL (ab 2027) | MITTEL-HOCH |
| Kein Bankabgleich | MITTEL | HOCH (API-Kosten) |
| Kein Belegarchiv (GoBD 10 Jahre) | MITTEL | MITTEL |
| PayrollEntry in team.ts nur Anzeige, keine Berechnung | OK — soll so bleiben! | - |
| Trainings/Schulungen nur Anzeige | OK fuer MVP | - |

### 10.4 Wettbewerbs-Positionierung

KMU Hub konkurriert NICHT mit Lexoffice, Bexio oder Personio direkt. Die Positionierung ist:

**"KMU Hub ist das Betriebssystem fuer dein KMU. Buchhaltung und Lohn machst du weiterhin mit deinem Steuerberater / Bexio / DATEV — aber alles andere (CRM, Projekte, Chat, Meetings, HR-Basics, Rechnungen, Zeiterfassung) laeuft in einer App. Und die spricht mit deinen bestehenden Tools."**

Das ist der USP: NICHT noch ein Buchhaltungstool, sondern die Klammer um alles.

### 10.5 Risiko-Matrix

| Risiko | Wahrscheinlichkeit | Impact | Mitigation |
|--------|-------------------|--------|------------|
| KMU-Kunde fragt "Kann ich meine komplette Buchhaltung in KMU Hub machen?" | HOCH | MITTEL | Klare Kommunikation: "Nein, aber wir integrieren perfekt mit deinem Buchhaltungstool" |
| GoBD-Haftung wenn Rechnungsmodul Fehler hat | MITTEL | HOCH | Rechnungen =/= Buchhaltung. Trotzdem: Lueckenlose Nummern, unveraenderbare Rechnungen, Archivierung |
| Bexio/DATEV aendern API | NIEDRIG | MITTEL | API-Versioning, Monitoring, Fallback auf CSV-Export |
| E-Invoicing Pflicht 2027 nicht rechtzeitig umgesetzt | MITTEL | HOCH | Frueh anfangen (Mitte 2026), ZUGFeRD-Library evaluieren |
| Lohn-Daten falsch exportiert | NIEDRIG | HOCH | Daten-Validierung, klare "Preview before Export"-Flows |

---

## Quellen und Confidence

| Bereich | Confidence | Begruendung |
|---------|-----------|-------------|
| Toolnamen, Hersteller, Herkunft | HIGH | Stabile Fakten, aendern sich nicht |
| Preise | LOW-MEDIUM | Stand Mai 2025, koennen sich geaendert haben. Vor Veroeffentlichung pruefen! |
| Marktanteile | MEDIUM | Geschaetzt aus verschiedenen Quellen, keine exakten Zahlen verfuegbar |
| GoBD-Anforderungen | HIGH | Rechtliche Grundlagen stabil seit BMF-Schreiben 2019 (aktualisiert 2023) |
| Schweizer OR Art. 957ff | HIGH | Gesetzestext, stabil |
| E-Invoicing Pflichten DE | HIGH | Wachstumschancengesetz verabschiedet, Fristen stehen fest |
| Swissdec/ELSTER/FinanzOnline | MEDIUM | Technische Details aus Training, Details koennen sich geaendert haben |
| API-Verfuegbarkeit der Tools | MEDIUM | Stand Mai 2025, APIs sind typischerweise stabil |
| Build-vs-Integrate Empfehlungen | HIGH | Basiert auf Domaen-Wissen und Risiko-Analyse, nicht auf volatilen Daten |

**WARNUNG:** Preise und Versionen sollten vor jeder kundengerichteten Verwendung live verifiziert werden. Diese Recherche dient der internen Entscheidungsfindung, nicht als Verkaufs-Vergleich.

---

## Appendix A: DATEV-Export-Format (Kurzreferenz)

Das DATEV-Buchungsstapel-Format besteht aus:
1. **Header-Zeile:** Formatkennung, Version, Erzeugt-Datum, Berater-Nr, Mandanten-Nr, Wirtschaftsjahr-Beginn, Sachkontenlänge, Datum-Von, Datum-Bis
2. **Buchungszeilen:** Umsatz (Soll/Haben), Gegenkonto, Belegdatum, Buchungstext, Belegnummer, etc.
3. **Encoding:** Windows-1252 (NICHT UTF-8!)
4. **Trennzeichen:** Semikolon
5. **Dezimal:** Komma (deutsches Format)

Minimal-Felder pro Buchungszeile:
- Umsatz (Betrag)
- Soll/Haben-Kennzeichen (S/H)
- Konto (Debitor/Kreditor)
- Gegenkonto (Sachkonto)
- Belegdatum
- Buchungstext

## Appendix B: Swiss QR-Code (Kurzreferenz)

Der Swiss QR-Code auf QR-Rechnungen enthaelt:
1. **Header:** QRType, Version, Coding
2. **Zahlungsempfaenger:** IBAN (CH-IBAN), Name, Adresse
3. **Betrag/Waehrung:** Betrag (optional), Waehrung (CHF/EUR)
4. **Zahlungspflichtiger:** Name, Adresse (optional)
5. **Referenz:** QR-Referenz (26+1 Stellen) ODER Creditor Reference (ISO 11649) ODER keine
6. **Zusatzinfo:** Unstrukturierte/Strukturierte Mitteilung
7. **Alternative Verfahren:** z.B. eBill

Library-Empfehlung: `github.com/nicovince/swissqr` (Go) oder eigene Implementation (Spec ist ~30 Seiten, machbar).

## Appendix C: ZUGFeRD/XRechnung (Kurzreferenz)

- **ZUGFeRD 2.x** = Factur-X (deutsch-franzoesischer Standard) = PDF/A-3 mit eingebettetem XML
- **XRechnung** = Deutsches Profil von EN 16931 = reines XML
- Beide basieren auf **UN/CEFACT Cross Industry Invoice (CII)** oder **UBL 2.1**
- ZUGFeRD Profile: MINIMUM, BASIC WL, BASIC, EN 16931, EXTENDED
- Fuer B2B-Pflicht ab 2025/2027: Mindestens EN 16931 Profil
- Go-Libraries: `invopop/gobl` (multi-format), `go-zugferd` (experimentell)
