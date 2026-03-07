# DSGVO/DSG Compliance-Framework fuer KMU Hub

**Recherchiert:** 2026-02-16
**Confidence:** MEDIUM-HIGH (basiert auf Gesetzestexten Stand Training-Daten Mai 2025; Gesetze aendern sich selten, aber aktuelle Durchfuehrungsverordnungen/Urteile sollten vor Implementation nochmals geprueft werden)
**Hinweis:** WebSearch und WebFetch waren nicht verfuegbar. Alle Angaben basieren auf Training-Daten. Gesetze, Paragraphen und Fristen sind stabil und aendern sich selten, daher ist die Confidence trotzdem hoch. Vor jeder konkreten Implementation MUSS ein Rechtsanwalt fuer IT-Recht konsultiert werden.

---

## Inhaltsverzeichnis

1. [DSGVO (EU/DE) Anforderungen fuer SaaS](#1-dsgvo-eude-anforderungen-fuer-saas)
2. [Schweizer nDSG](#2-schweizer-ndsg-seit-1-september-2023)
3. [Zertifizierungen & Standards](#3-zertifizierungen--standards)
4. [Hosting & Infrastruktur](#4-hosting--infrastruktur)
5. [Branchenspezifische Compliance](#5-branchenspezifische-compliance)
6. [Praktische Umsetzung fuer KMU Hub](#6-praktische-umsetzung-fuer-kmu-hub)
7. [Wettbewerbsvorteil durch Datenschutz](#7-wettbewerbsvorteil-durch-datenschutz)

---

## 1. DSGVO (EU/DE) Anforderungen fuer SaaS

### 1.1 Auftragsverarbeitungsvertrag (AVV) — Art. 28 DSGVO

**Rechtsgrundlage:** Art. 28 Abs. 3 DSGVO

Der AVV ist PFLICHT fuer jede SaaS-Plattform. KMU Hub ist Auftragsverarbeiter (Art. 4 Nr. 8 DSGVO), die KMU-Kunden sind Verantwortliche (Art. 4 Nr. 7 DSGVO).

**Pflichtinhalte des AVV (Art. 28 Abs. 3 DSGVO):**

| Nr. | Pflichtinhalt | KMU Hub Umsetzung |
|-----|---------------|-------------------|
| 1 | **Gegenstand und Dauer** der Verarbeitung | Bereitstellung CRM/ERP SaaS; Dauer = Vertragslaufzeit |
| 2 | **Art und Zweck** der Verarbeitung | Speicherung, Verarbeitung, Anzeige von Geschaeftsdaten |
| 3 | **Art der personenbezogenen Daten** | Kontaktdaten, Kommunikation, Finanzdaten, HR-Daten |
| 4 | **Kategorien betroffener Personen** | Mitarbeiter des Kunden, Kunden des Kunden, Lieferanten |
| 5 | **Pflichten und Rechte des Verantwortlichen** | Weisungsbefugnis, Kontrollrecht |
| 6 | **Weisungsgebundenheit** (Art. 28 Abs. 3 lit. a) | Verarbeitung NUR nach dokumentierter Weisung |
| 7 | **Vertraulichkeitsverpflichtung** (lit. b) | Alle Mitarbeiter mit Zugang muessen Vertraulichkeit zusichern |
| 8 | **TOMs** (lit. c) | Technische und Organisatorische Massnahmen (siehe 1.7) |
| 9 | **Unterauftragsverarbeiter** (lit. d) | Liste aller Sub-Processors mit Genehmigungspflicht |
| 10 | **Unterstuetzung bei Betroffenenrechten** (lit. e) | Technische Moeglichkeit zur Auskunft, Loeschung etc. |
| 11 | **Unterstuetzung bei DSFA** (lit. f) | Datenschutz-Folgenabschaetzung unterstuetzen |
| 12 | **Loeschung/Rueckgabe** nach Ende (lit. g) | Daten loeschen oder zurueckgeben nach Vertragsende |
| 13 | **Nachweispflicht und Audits** (lit. h) | Kunde darf auditieren oder Audit-Berichte anfordern |

**Sub-Processors fuer KMU Hub (muessen im AVV gelistet werden):**

| Sub-Processor | Zweck | Standort |
|---------------|-------|----------|
| Hetzner | Server-Hosting | DE (Falkenstein, Nuernberg) |
| OVH | Backup/Failover (optional) | FR/DE |
| LiveKit Cloud (falls genutzt) | Video-Infrastruktur | Muss EU sein! |
| Hetzner Object Storage / MinIO | Dateispeicherung | DE |
| E-Mail-Provider (SMTP Relay) | E-Mail-Versand | Muss EU sein |

**WICHTIG:** Bei JEDER Aenderung der Sub-Processor-Liste muessen Kunden informiert werden und ein Widerspruchsrecht haben (Art. 28 Abs. 2 DSGVO). Best Practice: 30-Tage Vorabinformation.

**Confidence:** HIGH — Art. 28 DSGVO ist seit 2018 unveraendert.

---

### 1.2 Datenresidenz — Wo MUESSEN Daten gehostet werden?

**Rechtslage:**

Die DSGVO schreibt NICHT vor, dass Daten in der EU gehostet werden muessen. Sie schreibt vor, dass bei Transfer in Drittlaender ein **angemessenes Schutzniveau** bestehen muss (Art. 44-49 DSGVO).

ABER: In der Praxis fuer DACH-KMUs gilt:

| Szenario | Erlaubt? | Bedingung |
|----------|----------|-----------|
| Hosting in DE/EU | Ja | Standard |
| Hosting in CH | Ja | CH hat Angemessenheitsbeschluss (Art. 45 DSGVO) |
| Hosting in UK | Ja | UK hat Angemessenheitsbeschluss (bis 27.06.2025, muss geprueft werden ob verlaengert) |
| Hosting in USA | Problematisch | EU-US Data Privacy Framework (DPF) seit Juli 2023, aber politisch unsicher |
| Hosting in anderen Drittlaendern | Nur mit SCCs + TIA | Standardvertragsklauseln + Transfer Impact Assessment |

**Empfehlung fuer KMU Hub:**
- **SaaS:** Ausschliesslich EU-Rechenzentren (Hetzner DE, OVH DE/FR)
- **Self-Hosted:** Kunde waehlt Standort, KMU Hub empfiehlt EU
- **Marketing-Claim:** "100% EU-Datenresidenz" ist ein STARKER Verkaufsvorteil

**Schrems II Konsequenzen (EuGH C-311/18):**
- Privacy Shield wurde 2020 fuer ungueltig erklaert
- EU-US DPF ist Nachfolger (Angemessenheitsbeschluss Juli 2023)
- ABER: Rechtliche Unsicherheit bleibt — neues EuGH-Verfahren ("Schrems III") ist moeglich
- DACH-KMUs wollen KEIN Risiko → EU-only ist der sichere Weg

**Confidence:** HIGH fuer EU-Hosting-Empfehlung; MEDIUM fuer DPF-Status (politisch volatil).

---

### 1.3 Betroffenenrechte — Technische Umsetzung

Die DSGVO gewaehrt Betroffenen umfangreiche Rechte. KMU Hub muss diese TECHNISCH unterstuetzen.

#### Art. 15 DSGVO — Auskunftsrecht

**Frist:** 1 Monat (verlaengerbar auf 3 Monate bei Komplexitaet, Art. 12 Abs. 3)

**Technische Umsetzung:**
- Globale Suche ueber ALLE Module nach personenbezogenen Daten einer Person
- Abfrage-Endpunkt: `GET /api/v1/gdpr/data-export?email=person@example.com`
- Muss durchsuchen: Kontakte, E-Mails, Chat-Nachrichten, Dateien (Metadaten), Kalendereintraege, Audit-Logs, HR-Daten, Rechnungen
- Output: Maschinenlesbares Format (JSON oder CSV)
- Admin-UI: Button "DSGVO-Auskunft generieren" im Kontakt-Detail

**Bereits geplant als:** SEC-05

#### Art. 17 DSGVO — Recht auf Loeschung ("Recht auf Vergessenwerden")

**Frist:** 1 Monat (unverzueglich, Art. 17 Abs. 1)

**Technische Umsetzung:**
- Kaskadierende Anonymisierung ueber ALLE Module
- NICHT einfach `DELETE FROM contacts WHERE email = ?` — das bricht referentielle Integritaet
- Stattdessen: **Anonymisierung** (Pseudonymisierung der Daten, Beziehungen bleiben intakt)
- Strategie pro Modul:

| Modul | Loeschstrategie |
|-------|-----------------|
| Kontakte | Name → "Geloeschte Person #12345", E-Mail → hash@deleted.local |
| E-Mails | Empfaenger-/Absender-Name anonymisieren, Inhalt loeschen |
| Chat | Nachrichten dieser Person → "[Nachricht geloescht - DSGVO]" |
| Dateien | Dateien der Person loeschen, Metadaten anonymisieren |
| Kalender | Teilnehmer-Name anonymisieren |
| HR | Personaldaten loeschen (ACHTUNG: Aufbewahrungsfristen beachten!) |
| Rechnungen | NICHT loeschen (GoBD!), aber Kontaktdaten anonymisieren |
| Audit-Log | Log-Eintraege anonymisieren (Person-Referenz entfernen) |

**WICHTIG — Ausnahmen vom Loeschrecht (Art. 17 Abs. 3):**
- Aufbewahrungspflichten nach HGB/AO (Rechnungen: 10 Jahre!)
- Geltendmachung von Rechtsanspruechen
- System muss diese Ausnahmen erkennen und den Admin warnen

**Bereits geplant als:** SEC-06

#### Art. 20 DSGVO — Recht auf Datenportabilitaet

**Frist:** 1 Monat

**Technische Umsetzung:**
- Export in "strukturiertem, gaengigem, maschinenlesbarem Format"
- Empfehlung: JSON + CSV-Bundel als ZIP
- Endpunkt: `GET /api/v1/gdpr/portable-export?contact_id=xxx`
- Format:

```json
{
  "export_date": "2026-02-16T10:00:00Z",
  "format_version": "1.0",
  "contact": { ... },
  "emails": [ ... ],
  "calendar_events": [ ... ],
  "files": [ { "name": "...", "download_url": "..." } ],
  "activities": [ ... ]
}
```

**Confidence:** HIGH — Diese Rechte sind im Gesetzestext klar definiert.

---

### 1.4 Einwilligungsmanagement

**Rechtsgrundlage:** Art. 6, Art. 7 DSGVO

Fuer KMU Hub als B2B-SaaS ist die primaere Rechtsgrundlage NICHT Einwilligung, sondern:

| Verarbeitungszweck | Rechtsgrundlage | Paragraph |
|--------------------|-----------------|-----------|
| Vertragsdurchfuehrung (CRM-Nutzung) | Vertragserfuellung | Art. 6 Abs. 1 lit. b |
| Rechnungsstellung | Vertragserfuellung | Art. 6 Abs. 1 lit. b |
| Aufbewahrungspflichten | Rechtliche Verpflichtung | Art. 6 Abs. 1 lit. c |
| Marketing/Newsletter | Einwilligung | Art. 6 Abs. 1 lit. a |
| Mitarbeiterdaten (HR) | Vertragserfuellung (Arbeitsvertrag) | Art. 6 Abs. 1 lit. b + Art. 88 + BDSG ss26 |
| Video-Aufzeichnung | Einwilligung | Art. 6 Abs. 1 lit. a |
| Analytics/Telemetrie | Berechtigtes Interesse | Art. 6 Abs. 1 lit. f |

**Wo Einwilligung NOETIG ist (und KMU Hub dies unterstuetzen muss):**

1. **Video-Recording:** Vor Aufnahmestart muessen ALLE Teilnehmer zustimmen (bereits geplant als VID-07)
2. **E-Mail-Tracking:** Oeffnungsbestaetigung nur mit Opt-in (geplant als MAIL-12 in v2)
3. **Cookie-Banner:** Falls KMU Hub ein Web-Portal hat (nicht fuer Electron-App)
4. **Newsletter des Kunden:** Falls der Kunde KMU Hub fuer Marketing nutzt (E-Mail-Modul)

**Technische Umsetzung:**
- Consent-Flag pro Kontakt pro Zweck speichern
- Timestamp + Herkunft der Einwilligung loggen (Nachweis!)
- Widerrufmoeglichkeit
- DB-Schema:

```sql
CREATE TABLE consent_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL REFERENCES contacts(id),
    tenant_id UUID NOT NULL,
    purpose VARCHAR(100) NOT NULL,        -- z.B. 'newsletter', 'video_recording'
    granted BOOLEAN NOT NULL,
    granted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    source VARCHAR(50) NOT NULL,           -- 'web_form', 'email', 'verbal', 'app'
    ip_address INET,
    evidence TEXT,                          -- Link/Referenz zum Nachweis
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_consent_records_contact ON consent_records(contact_id);
CREATE INDEX idx_consent_records_purpose ON consent_records(tenant_id, purpose);
```

**Confidence:** HIGH

---

### 1.5 Datenpannenmeldung — 72-Stunden-Regel

**Rechtsgrundlage:** Art. 33 + Art. 34 DSGVO

**Pflichten bei einer Datenpanne (Data Breach):**

| Schritt | Frist | An wen | Bedingung |
|---------|-------|--------|-----------|
| 1. Interne Erkennung | Sofort | Internes Team | Immer |
| 2. Meldung an Aufsichtsbehoerde | 72 Stunden ab Kenntnis | LfDI des Bundeslandes / EDOEB (CH) | Wenn Risiko fuer Betroffene |
| 3. Meldung an Betroffene | Unverzueglich | Betroffene Personen | Wenn HOHES Risiko fuer Betroffene |
| 4. Meldung an Kunden (als AV) | Unverzueglich | Alle betroffenen Kunden-Tenants | IMMER (Art. 33 Abs. 2) |

**WICHTIG fuer KMU Hub als Auftragsverarbeiter:**
- KMU Hub muss den KUNDEN (Verantwortlichen) unverzueglich informieren
- Der KUNDE meldet dann an seine Aufsichtsbehoerde
- KMU Hub muss den Kunden bei der Meldung unterstuetzen

**Aufsichtsbehoerden (DE):**

| Bundesland | Behoerde |
|------------|----------|
| Bayern (nicht-oeffentlich) | BayLDA |
| Baden-Wuerttemberg | LfDI BW |
| Berlin | BlnBDI |
| Brandenburg | LDA Brandenburg |
| Hamburg | HmbBfDI |
| Hessen | HBDI |
| NRW | LDI NRW |
| Sachsen | SaechsDSB |
| Bundesebene | BfDI (fuer Telekommunikation, Post) |

**Schweiz:** EDOEB (Eidgenoessischer Datenschutz- und Oeffentlichkeitsbeauftragter)

**Technische Umsetzung — Breach Response Toolkit:**

```
1. Monitoring & Detection
   - Intrusion Detection System (IDS)
   - Anomalie-Erkennung (ungewoehnliche Datenabfragen)
   - Login-Anomalien (Geo-IP, Brute Force)

2. Interne Prozedur
   - Incident-Response-Plan dokumentieren
   - Eskalationsmatrix (wer informiert wen)
   - Breach-Log Template vorhalten

3. Kunden-Benachrichtigung
   - Admin-Notification per E-Mail + In-App
   - Inhalt: Was passiert ist, welche Daten betroffen, Massnahmen

4. Dokumentation
   - Jeder Breach muss dokumentiert werden (auch wenn keine Meldepflicht)
   - Art. 33 Abs. 5: Breach-Verzeichnis fuehren
```

**Confidence:** HIGH

---

### 1.6 Datenschutzbeauftragter (DSB) — Ab wann Pflicht?

**Rechtsgrundlage:** Art. 37 DSGVO + ss38 BDSG (Deutschland-spezifisch!)

**Fuer KMU Hub als Unternehmen:**

| Kriterium | Schwelle | KMU Hub betroffen? |
|-----------|----------|-------------------|
| ss38 Abs. 1 BDSG: Mindestens 20 Personen mit regelmaessiger automatisierter Datenverarbeitung | 20 MA | NEIN (3-Personen-Team) |
| Art. 37 Abs. 1 lit. b: Kerngeschaeft = umfangreiche systematische Ueberwachung | Unwahrscheinlich | NEIN |
| Art. 37 Abs. 1 lit. c: Kerngeschaeft = Verarbeitung besonderer Datenkategorien | Art. 9/10 Daten | Nur wenn Gesundheitsdaten |

**Empfehlung:** Aktuell keine DSB-Pflicht fuer KMU Hub (3 MA). ABER:
- Freiwillige Bestellung zeigt Serioesitaet
- Externer DSB kostet ca. 200-500 EUR/Monat
- Ab Wachstum auf 20+ MA wird es Pflicht
- **Empfehlung: Externen DSB ab Beta-Launch engagieren** (Signal fuer Kunden)

**Fuer die KMU-Kunden:**
- Kunden mit 20+ MA in DE brauchen einen DSB
- KMU Hub kann darauf hinweisen, ist aber nicht verantwortlich
- Feature-Idee: "Datenschutz-Checkliste" im Admin-Bereich

**Schweiz (nDSG):**
- KEIN Pflicht-DSB im nDSG
- Freiwillige Ernennung eines "Datenschutzberaters" (Art. 10 nDSG) moeglich
- Wird empfohlen bei Verarbeitung schuetzenswerter Personendaten

**Confidence:** HIGH (gesetzliche Schwellenwerte sind eindeutig)

---

### 1.7 TOMs — Technische und Organisatorische Massnahmen

**Rechtsgrundlage:** Art. 32 DSGVO

TOMs muessen im AVV aufgefuehrt UND tatsaechlich implementiert sein.

#### Technische Massnahmen

**Zutrittskontrolle (physisch):**
- Rechenzentrum mit Zutrittskontrolle (Hetzner: ISO 27001 zertifiziert)
- Serverraeume nicht oeffentlich zugaenglich
- Bei Self-Hosted: Kundenpflicht

**Zugangskontrolle (logisch):**

| Massnahme | Implementation |
|-----------|----------------|
| Passwort-Policy | Min. 12 Zeichen, Komplexitaet, kein Reuse (SEC-09) |
| 2FA | TOTP (SEC-01), spaeter FIDO2 (SEC-12) |
| Session-Management | JWT mit kurzer Laufzeit (15min Access, 7d Refresh) |
| IP-Allowlist | Fuer Admin-Zugang (SEC-10) |
| Automatische Sperre | Nach 5 Fehlversuchen, 15min Lockout |
| SSO | SAML/OIDC fuer Enterprise (SEC-13, v2) |

**Zugriffskontrolle (Berechtigungen):**

| Massnahme | Implementation |
|-----------|----------------|
| RBAC | 5 Rollen: admin, manager, member, hr, it_support |
| Mandantentrennung | Strikte Tenant-Isolation auf DB-Ebene |
| Least Privilege | Jede Rolle nur Minimalrechte |
| Kontakt-Sichtbarkeit | Company-shared vs. personal (CRM-03) |
| Audit-Trail | Alle Zugriffe loggen (SEC-03) |

**Weitergabekontrolle (Transport):**

| Massnahme | Implementation |
|-----------|----------------|
| TLS 1.3 | Alle API-Calls verschluesselt |
| HSTS | Strict-Transport-Security Header |
| Certificate Pinning | Fuer Electron-App (optional) |
| VPN | Fuer Self-Hosted Kunden empfohlen |
| E-Mail | STARTTLS fuer SMTP, idealerweise S/MIME |

**Eingabekontrolle (Nachvollziehbarkeit):**

| Massnahme | Implementation |
|-----------|----------------|
| Audit-Log | Wer hat wann was geaendert (SEC-03, SEC-04) |
| Versionierung | Dateiversionen (DOC-04) |
| Aenderungshistorie | Bei CRM-Kontakten, Deals etc. |
| Login-Log | IP, Timestamp, User-Agent |

**Verfuegbarkeitskontrolle:**

| Massnahme | Implementation |
|-----------|----------------|
| Backups | Taeglich, verschluesselt, georedundant |
| Monitoring | Health-Checks fuer alle Services |
| Graceful Degradation | Services fallen unabhaengig aus (ADR) |
| DDoS-Schutz | Hetzner DDoS Protection / Cloudflare (EU PoP) |
| Disaster Recovery | RTO < 4h, RPO < 1h |

**Trennungsgebot:**

| Massnahme | Implementation |
|-----------|----------------|
| Multi-Tenancy | Jeder Kunde = eigener Tenant, DB-Isolation |
| Umgebungstrennung | Production, Staging, Development getrennt |
| Daten-Trennung | Keine Kreuz-Tenant-Abfragen moeglich |

#### Organisatorische Massnahmen

| Massnahme | Beschreibung |
|-----------|--------------|
| Vertraulichkeitsverpflichtung | Alle Mitarbeiter unterschreiben NDAs |
| Schulungen | Regelmaessige Datenschutz-Schulungen |
| Clean Desk Policy | Keine Kundendaten auf physischen Medien |
| Incident-Response-Plan | Dokumentierter Prozess bei Datenpannen |
| Regelmaessige Audits | Jaehrliche interne Security-Reviews |
| Zugangsueberprufung | Quartalsweise Review aller Zugriffsrechte |
| Dokumentation | Verarbeitungsverzeichnis (Art. 30 DSGVO) |

**Confidence:** HIGH

---

## 2. Schweizer nDSG (seit 1. September 2023)

### 2.1 Kernunterschiede zur DSGVO

Das neue Schweizer Datenschutzgesetz (nDSG, SR 235.1) mit der Datenschutzverordnung (DSV, SR 235.11) trat am 1. September 2023 in Kraft und ersetzte das alte DSG von 1992.

| Thema | DSGVO (EU) | nDSG (Schweiz) | Relevanz KMU Hub |
|-------|-----------|----------------|------------------|
| **Anwendungsbereich** | Alle personenbez. Daten | Nur natuerliche Personen (nicht jur. Personen!) | Schweizer Firmendaten = kein Datenschutz |
| **Bussen** | Bis 20 Mio EUR / 4% Jahresumsatz (gegen Unternehmen) | Bis 250.000 CHF (gegen NATUERLICHE Personen!) | Geschaeftsfuehrung persoenlich haftbar! |
| **DSB/Datenschutzberater** | Pflicht ab 20 MA (DE) | Keine Pflicht, nur empfohlen | Kein Muss, aber Signal |
| **Einwilligung** | Opt-in (ausdruecklich) | Opt-in nur fuer besonders schuetzenswerte Daten | Einfacher fuer Standarddaten |
| **Profiling** | Art. 22 DSGVO (automatisierte Entscheidung) | Unterscheidung: Profiling vs. Profiling mit hohem Risiko | Strengere Transparenz bei "hohem Risiko" |
| **Datenschutz-Folgenabschaetzung** | Art. 35 DSGVO (bei hohem Risiko) | Art. 22 nDSG (aehnlich) | Gleich |
| **Verarbeitungsverzeichnis** | Art. 30 DSGVO (Pflicht ab 250 MA, de facto immer) | Art. 12 nDSG (Pflicht, Ausnahme < 250 MA mit geringem Risiko) | Pflicht fuer KMU Hub |
| **Meldepflicht Datenpanne** | 72h an Aufsichtsbehoerde | "So rasch wie moeglich" an EDOEB | Keine exakte Frist, aber SCHNELL |
| **Datenportabilitaet** | Art. 20 DSGVO | Art. 28 nDSG (neu seit 1.9.2023) | Muss unterstuetzt werden |
| **Privacy by Design/Default** | Art. 25 DSGVO | Art. 7 nDSG | Gleich |

### 2.2 Besonders schuetzenswerte Personendaten (Art. 5 lit. c nDSG)

Das nDSG kennt eine BREITERE Kategorie als die DSGVO:
- Religioese, weltanschauliche, politische, gewerkschaftliche Ansichten/Taetigkeiten
- Gesundheitsdaten
- Intimsphaere
- Rassenzugehoerigkeit / ethnische Herkunft
- Genetische Daten
- Biometrische Daten
- Daten ueber verwaltungs-/strafrechtliche Verfolgungen/Sanktionen
- **Daten ueber Massnahmen der sozialen Hilfe** (NEU gegenueber DSGVO)

**Fuer KMU Hub relevant:** HR-Modul koennte Gesundheitsdaten enthalten (Krankmeldungen, HR-07).

### 2.3 Datentransfer CH <-> EU Regeln

| Richtung | Regelung | Status |
|----------|----------|--------|
| CH → EU | Art. 16 nDSG + Anhang 1 DSV | EU hat Angemessenheitsbeschluss: ERLAUBT |
| EU → CH | Art. 45 DSGVO | CH hat Angemessenheitsbeschluss: ERLAUBT |
| CH → USA | Art. 16 nDSG | Swiss-US Data Privacy Framework (Swiss-US DPF) seit 15.09.2024 |
| CH → andere Drittlaender | Art. 16-17 nDSG | Nur mit angemessenem Schutz (Laenderliste des Bundesrats) |

**Fuer KMU Hub:**
- Datenfluss DE <-> CH ist UNPROBLEMATISCH (beidseitige Angemessenheitsbeschluessse)
- Hosting in DE fuer Schweizer Kunden = ERLAUBT
- ABER: Manche Schweizer Kunden WOLLEN Schweizer Datenresidenz (Verkaufsargument!)

### 2.4 Schweiz-spezifische Anforderungen fuer KMU Hub

1. **Informationspflicht (Art. 19 nDSG):**
   - Bei JEDER Datenbeschaffung Identitaet des Verantwortlichen mitteilen
   - Bearbeitungszweck nennen
   - Bei Weitergabe: Empfaengerlaender/-kategorien
   - KEINE Ausnahme fuer "offensichtliche" Daten wie in DSGVO

2. **Automatisierte Einzelentscheidung (Art. 21 nDSG):**
   - Betroffene muessen informiert werden
   - Recht auf Ueberpruefung durch natuerliche Person
   - Relevant fuer: Automatisierungen, Scoring, Lead-Bewertung

3. **Datenschutzberater (Art. 10 nDSG):**
   - Freiwillig, aber: Erleichtert Meldeverfahren (Art. 23 Abs. 4 nDSG)
   - Wenn ernannt: Muss dem EDOEB gemeldet werden

4. **Strafbestimmungen (Art. 60-66 nDSG):**
   - ACHTUNG: Bussen bis 250.000 CHF treffen NATUERLICHE PERSONEN
   - Geschaeftsfuehrung, nicht das Unternehmen
   - Straftatbestaende: Verletzung Informationspflicht, Verletzung Sorgfaltspflicht bei Auslandtransfer, Verletzung Berufsgeheimnis
   - Vorsatz erforderlich (Fahrlaessigkeit nicht strafbar)

### 2.5 FINMA-Relevanz bei Finanzdaten

**Wann relevant:** Wenn KMU Hub-Kunden FINMA-regulierte Unternehmen sind (Banken, Versicherungen, Vermoegensverwaltung).

| FINMA-Regelung | Anforderung | KMU Hub Impact |
|----------------|-------------|----------------|
| FINMA RS 2023/1 (Operationelle Risiken) | IT-Risikomanagement | Sicherheitsnachweise noetig |
| FINMA RS 2018/3 (Outsourcing) | Auslagerungsvertrag | AVV allein reicht nicht |
| Bankkundengeheimnis (Art. 47 BankG) | Bankdaten besonders geschuetzt | Verschluesselung Pflicht |
| Art. 10 FINMA-Outsourcing-Rundschreiben | Pruefrechtklausel | FINMA-Auditrecht im Vertrag |

**Empfehlung:** Fuer FINMA-regulierte Kunden separates Pricing-Tier mit:
- Dedizierte Infrastruktur (keine Shared Multi-Tenancy)
- Erweiterte Audit-Rechte
- Erweiterte Verschluesselung (Customer-managed Keys)
- ODER: Self-Hosted empfehlen

**Confidence:** MEDIUM (FINMA-Rundschreiben aendern sich regelmaessig, konkrete Nummern sollten geprueft werden)

---

## 3. Zertifizierungen & Standards

### 3.1 ISO 27001 — Information Security Management System (ISMS)

**Was es abdeckt:**
- Informationssicherheits-Managementsystem (ISMS)
- 93 Controls in Annex A (ISO 27001:2022)
- Risikomanagement, Asset-Management, Zugriffskontrolle, Kryptographie, physische Sicherheit, Operations, Kommunikation, Beschaffung, Incident-Management, Business Continuity, Compliance

**Kosten (Schaetzung):**

| Posten | Kosten | Zeitraum |
|--------|--------|----------|
| Beratung/Vorbereitung | 15.000-40.000 EUR | 6-12 Monate |
| Zertifizierungsaudit | 8.000-15.000 EUR | 1-2 Wochen |
| Jaehrliches Ueberwachungsaudit | 5.000-10.000 EUR | Jaehrlich |
| Re-Zertifizierung (alle 3 Jahre) | 8.000-15.000 EUR | Alle 3 Jahre |
| Interner Aufwand | Erheblich | Laufend |

**Notwendig fuer KMU Hub?**
- Fuer Beta/Launch: NEIN — zu teuer und zeitaufwaendig fuer 3-Personen-Team
- Fuer Enterprise-Kunden (ab 50+ MA): DRINGEND EMPFOHLEN
- Viele groessere KMUs und oeffentliche Auftraggeber FORDERN ISO 27001
- **Empfehlung: Start nach 12 Monaten Betrieb, wenn Revenue es traegt**

**Zwischenloesung bis ISO 27001:**
- SOC 2 Type I (schneller, guenstiger)
- Oder: "ISO 27001-konform" (Massnahmen implementiert, aber nicht zertifiziert) im Marketing
- TOM-Dokument veroeffentilichen (zeigt Transparenz)

**Confidence:** MEDIUM (Kosten sind Schaetzungen, variieren stark nach Anbieter und Scope)

### 3.2 BSI C5 — Cloud Computing Compliance Criteria Catalogue

**Was es ist:**
- Kriterienkatalog des Bundesamtes fuer Sicherheit in der Informationstechnik (BSI)
- Spezifisch fuer Cloud-Anbieter
- 17 Themenbereiche, 121 Basiskriterien

**Wann erforderlich:**
- **Pflicht fuer oeffentliche Verwaltung** in Deutschland (seit 2024 zunehmend gefordert)
- **Empfohlen** fuer Cloud-Anbieter, die oeffentliche Auftraggeber bedienen wollen
- **Nicht Pflicht** fuer private KMUs

**Typen:**
- **Typ 1:** Design und Implementierung zu einem Stichtag (einfacher)
- **Typ 2:** Wirksamkeit ueber einen Zeitraum (6-12 Monate, teurer)

**Kosten:**

| Posten | Typ 1 | Typ 2 |
|--------|-------|-------|
| Vorbereitung | 30.000-60.000 EUR | 40.000-80.000 EUR |
| Pruefung | 20.000-50.000 EUR | 30.000-70.000 EUR |
| Gesamt | 50.000-110.000 EUR | 70.000-150.000 EUR |

**Empfehlung fuer KMU Hub:**
- Fuer v1: NEIN — viel zu teuer fuer ein Startup
- Spaeter nur, wenn oeffentliche Verwaltung als Kundensegment angepeilt wird
- **Alternative:** Hetzner IST bereits BSI C5 zertifiziert — das Marketing nutzen ("Gehostet auf BSI C5-zertifizierter Infrastruktur")

**Confidence:** MEDIUM (Kosten sind Schaetzungen; BSI C5-Anforderungen an Hetzner HIGH confidence)

### 3.3 SOC 2 — Service Organization Control

**Relevanz im DACH-Raum:**
- SOC 2 ist primaer ein US-amerikanischer Standard (AICPA)
- Im DACH-Raum WENIGER bekannt als ISO 27001
- ABER: Zunehmend gefragt von internationalen Kunden und Tech-Unternehmen

**SOC 2 Typen:**
- **Type I:** Systemdesign zu einem Zeitpunkt
- **Type II:** Wirksamkeit ueber 6-12 Monate

**Kosten:**

| Posten | Type I | Type II |
|--------|--------|---------|
| Vorbereitung | 10.000-25.000 EUR | 15.000-35.000 EUR |
| Audit | 15.000-30.000 EUR | 25.000-50.000 EUR |

**Empfehlung fuer KMU Hub:**
- NEIN fuer v1 — im DACH-Raum wenig Wirkung
- ISO 27001 hat deutlich mehr Gewicht in DE/CH
- SOC 2 nur relevant, wenn internationale Expansion geplant

**Confidence:** MEDIUM

### 3.4 TISAX — Trusted Information Security Assessment Exchange

**Was es ist:**
- Automotive-spezifischer Standard (basiert auf ISO 27001 + VDA ISA)
- Verwaltet von der ENX Association
- PFLICHT fuer Zulieferer der Automobilindustrie

**Wann relevant fuer KMU Hub:**
- NUR wenn KMU Hub gezielt Automotive-KMUs bedient
- Wenn ein Kunde z.B. ein Zulieferer von BMW/VW/Daimler ist und KMU Hub als IT-System einsetzt
- Der KUNDE muss TISAX-konform sein, nicht unbedingt KMU Hub
- ABER: Kunden muessen nachweisen, dass auch ihre SaaS-Anbieter sicher sind

**Kosten:** 30.000-80.000 EUR (Assessment + Vorbereitung)

**Empfehlung fuer KMU Hub:**
- NEIN fuer v1 — zu nischig
- Self-Hosted Option loest das Problem: Kunde hostet selbst in TISAX-konformer Umgebung
- Bei Nachfrage: Self-Hosted + Consulting anbieten

**Confidence:** HIGH

### 3.5 Zusammenfassung Zertifizierungen

| Zertifizierung | Prioritaet | Zeitrahmen | Kosten | Empfehlung |
|----------------|-----------|------------|--------|------------|
| ISO 27001 | HOCH | 12-18 Mo nach Launch | 30-70k EUR | Ja, sobald tragbar |
| BSI C5 | NIEDRIG | Nur bei oeff. Verwaltung | 50-150k EUR | Nein fuer v1 |
| SOC 2 | NIEDRIG | Nur bei Int'l Expansion | 25-85k EUR | Nein fuer v1 |
| TISAX | NEIN | Nur Automotive | 30-80k EUR | Self-Hosted stattdessen |

---

## 4. Hosting & Infrastruktur

### 4.1 EU-only Hosting — Anbieter-Vergleich

| Anbieter | Standorte | ISO 27001 | BSI C5 | Preis-Niveau | KMU Hub Eignung |
|----------|-----------|-----------|--------|-------------|-----------------|
| **Hetzner** | DE (Falkenstein, Nuernberg), FI (Helsinki) | Ja | Ja (Rechenzentren) | Sehr guenstig | EMPFOHLEN (bereits gewaehlt) |
| **OVH** | FR (Roubaix, Strasbourg, Gravelines), DE (Frankfurt) | Ja | Nein | Guenstig | Gut fuer Backup/Failover |
| **IONOS (1&1)** | DE (Berlin, Frankfurt, Karlsruhe) | Ja | Ja (teilweise) | Mittel | Alternative zu Hetzner |
| **Exoscale** | CH (Zuerich, Genf), DE (Frankfurt, Muenchen), AT (Wien) | Ja | Nein | Teuer | EMPFOHLEN fuer CH-Kunden |
| **Infomaniak** | CH (Genf) | Ja | Nein | Mittel | Gute CH-Alternative |
| **Proton/ProtonDC** | CH (Genf) | Ja | Nein | Teuer | Premium-Image, klein |

**Empfehlung:**

| Einsatzzweck | Anbieter | Begruendung |
|--------------|----------|-------------|
| SaaS (DE-Kunden) | Hetzner (DE) | Preis-Leistung, BSI C5, ISO 27001 |
| SaaS (CH-Kunden) | Exoscale (CH) | Schweizer Datenresidenz |
| Backup/DR | OVH (DE) | Georedundanz zu Hetzner |
| Self-Hosted | Kundeninfrastruktur | Keine Hosting-Entscheidung noetig |

### 4.2 Schrems II — Duerfen Daten auf US-Server?

**Kurzantwort: Technisch ja (mit DPF), praktisch NEIN fuer KMU Hub.**

**Rechtliche Lage:**
- EU-US Data Privacy Framework (DPF): Angemessenheitsbeschluss seit 10. Juli 2023
- Gilt NUR fuer US-Unternehmen, die sich zertifiziert haben
- Schweiz: Swiss-US DPF seit 15. September 2024
- ABER: Juristisch umstritten, "Schrems III" droht

**Warum KMU Hub KEINE US-Server nutzen sollte:**

1. **Kundenerwarterung:** DACH-KMUs erwarten EU-Hosting
2. **Verkaufsargument:** "100% EU" ist staerker als "DSGVO-konform"
3. **Rechtsunsicherheit:** DPF kann jederzeit gekippt werden
4. **CLOUD Act / FISA 702:** US-Behoerden koennen Zugriff fordern — auch auf EU-Daten bei US-Unternehmen
5. **Sub-Processor-Risiko:** Hetzner, OVH sind EU-Unternehmen ohne US-Jurisdiktion

**Konkret vermeiden:**
- AWS (Amazon, US) — auch EU-Regionen unterliegen US-Jurisdiktion
- Azure (Microsoft, US) — dito
- GCP (Google, US) — dito
- Cloudflare (US) — Fuer CDN/DDoS ggf. akzeptabel (keine Datenspeicherung)

**Confidence:** HIGH (Rechtslagen sind gut dokumentiert; DPF-Stabilitaet = MEDIUM)

### 4.3 Self-Hosted vs. SaaS — Compliance-Unterschiede

| Aspekt | SaaS | Self-Hosted |
|--------|------|-------------|
| **Verantwortlicher** | Kunde | Kunde |
| **Auftragsverarbeiter** | KMU Hub | KEINER (Kunde verarbeitet selbst) |
| **AVV noetig?** | JA (Pflicht) | NEIN (kein AV-Verhaeltnis) |
| **Datenresidenz** | KMU Hub bestimmt | Kunde bestimmt |
| **TOMs** | KMU Hub verantwortlich | Kunde verantwortlich |
| **Datenpanne** | KMU Hub meldet an Kunde | Kunde meldet an Behoerde direkt |
| **Audit-Recht** | Kunde darf auditieren | Nicht noetig |
| **Sub-Processors** | Muessen gelistet werden | Keine |
| **Compliance-Aufwand KMU Hub** | HOCH | NIEDRIG (nur Software-Qualitaet) |
| **Compliance-Aufwand Kunde** | MITTEL | HOCH (muss selbst hosten) |

**Empfehlung:**
- Self-Hosted als PREMIUM-Option fuer Kunden mit hohen Compliance-Anforderungen
- FINMA-regulierte, Gesundheitswesen, oeffentliche Verwaltung → Self-Hosted empfehlen
- Marketing: "Volle Kontrolle ueber Ihre Daten"

### 4.4 Schweizer Datenresidenz fuer Schweizer Kunden

**Rechtlich nicht erforderlich** (EU hat Angemessenheitsbeschluss), ABER:

**Warum trotzdem anbieten:**
1. Schweizer KMUs WOLLEN es (emotionaler Faktor, "Swissness")
2. FINMA-regulierte Unternehmen brauchen es teils
3. Wettbewerbsvorteil gegenueber US-SaaS
4. Differenzierung zu deutschen Anbietern

**Technische Umsetzung:**
- Exoscale Zuerich/Genf als separater Cluster
- Tenant-Routing: Bei Signup CH-Kunden automatisch auf CH-Cluster
- Datenbank-Replikation nur innerhalb CH
- Separate Sub-Processor-Liste fuer CH-Kunden im AVV

**Kosten-Schaetzung:**
- Exoscale ist ca. 30-50% teurer als Hetzner
- Rechtfertigt hoeheren Preis fuer Schweizer Kunden
- "Swiss Data Residency" als kostenpflichtiges Add-on (z.B. +10 CHF/User/Monat)

**Confidence:** MEDIUM (Preisvergleiche sind Schaetzungen)

---

## 5. Branchenspezifische Compliance

### 5.1 GoBD (Deutschland) — Grundsaetze zur ordnungsmaessigen Fuehrung und Aufbewahrung von Buechern, Aufzeichnungen und Unterlagen in elektronischer Form

**Rechtsgrundlage:** BMF-Schreiben vom 28.11.2019 (GZ: IV A 4 - S 0316/19/10003 :001)
**Gueltig seit:** 01.01.2020 (ersetzt GoBS und GDPdU)
**Betrifft:** JEDEN Steuerpflichtigen in Deutschland, der elektronische Aufzeichnungen fuehrt

**Die 8 GoBD-Grundsaetze:**

| Nr. | Grundsatz | Bedeutung | KMU Hub Relevanz |
|-----|-----------|-----------|------------------|
| 1 | **Nachvollziehbarkeit** | Buchungen muessen verstaendlich und nachpruefbar sein | Audit-Trail, Aenderungshistorie |
| 2 | **Nachpruefbarkeit** | Externer Pruefer muss Daten pruefen koennen | Export-Funktion, Pruefer-Zugang |
| 3 | **Wahrheit** | Buchungen muessen Geschaeftsvorfaelle korrekt abbilden | Keine nachtraegliche Manipulation |
| 4 | **Klarheit** | Systematische, uebersichtliche Aufzeichnung | Klare Strukturierung |
| 5 | **Fortlaufende Aufzeichnung** | Zeitnah und lueckenlos buchen | Fortlaufende Rechnungsnummer (FIN-02) |
| 6 | **Vollstaendigkeit** | ALLE Geschaeftsvorfaelle erfasst | Keine Luecken in Nummernkreisen |
| 7 | **Ordnung** | Sachliche und zeitliche Ordnung | Sortierung nach Datum/Kategorie |
| 8 | **Unveraenderbarkeit** | Einmal gebuchtes darf NICHT geaendert/geloescht werden | Stornobuchung statt Loeschung! |

**Konkrete Anforderungen fuer KMU Hub Finance-Modul:**

#### Unveraenderbarkeit (Rn. 58-63 GoBD)
```
- Rechnungen, Buchungen, Belege duerfen nach Festschreibung NICHT mehr geaendert werden
- Aenderungen nur durch Stornobuchung + Neubuchung
- Jede Aenderung muss protokolliert werden (wer, wann, was)
- Loeschen ist VERBOTEN (auch bei Fehlern!)
- Technisch: IMMUTABLE Records + Append-only Audit-Log
```

**DB-Design-Empfehlung:**
```sql
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    invoice_number VARCHAR(50) NOT NULL UNIQUE,  -- Fortlaufend, lueckenlos!
    status VARCHAR(20) NOT NULL DEFAULT 'draft',  -- draft, sent, paid, cancelled
    is_finalized BOOLEAN DEFAULT FALSE,           -- Nach Finalisierung: UNVERAENDERBAR
    finalized_at TIMESTAMPTZ,
    -- ... weitere Felder

    CONSTRAINT no_gaps CHECK (invoice_number ~ '^\d+$')  -- Numerisch, fortlaufend
);

-- Aenderungsprotokoll (Pflicht bei GoBD)
CREATE TABLE invoice_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    action VARCHAR(50) NOT NULL,    -- 'created', 'finalized', 'cancelled', 'payment_recorded'
    changed_by UUID NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT now(),
    old_values JSONB,
    new_values JSONB,
    reason TEXT                      -- Pflicht bei Stornierung
);
```

#### Fortlaufende Nummerierung (Rn. 64-67 GoBD + ss14 Abs. 4 UStG)
- Rechnungsnummern MUESSEN fortlaufend und lueckenlos sein
- Nummernkreise sind erlaubt (z.B. pro Jahr: 2026-0001, 2026-0002)
- Luecken fuehren zu Nachfragen bei Betriebspruefung
- Geleschte Nummern muessen dokumentiert werden (warum fehlt Nr. X?)

#### Verfahrensdokumentation (Rn. 151-155 GoBD)
- JEDES DV-gestuetzte Buchfuehrungssystem braucht eine Verfahrensdokumentation
- Inhalt: Wie werden Daten erfasst, verarbeitet, gespeichert, geschuetzt?
- KMU Hub muss Kunden eine Vorlage liefern

**Verfahrensdokumentation muss enthalten:**
1. Allgemeine Beschreibung des Systems
2. Anwenderdokumentation (Bedienungsanleitung)
3. Technische Systemdokumentation
4. Betriebsdokumentation (Backup, Recovery)
5. Internes Kontrollsystem (IKS)

#### Datenzugriff (Rn. 164-175 GoBD)
Finanzamt hat 3 Zugriffsarten:
1. **Z1 — Unmittelbarer Zugriff:** Pruefer bekommt Read-only Zugang zum System
2. **Z2 — Mittelbarer Zugriff:** Kunde erstellt Auswertungen nach Vorgabe des Pruefers
3. **Z3 — Datentraegerueberlassung:** Export in maschinell auswertbarem Format

**KMU Hub MUSS unterstuetzen:**
- Export in DATEV-kompatiblem CSV-Format (FIN-06)
- GDPdU/GoBD-konformer Export (XML + Daten)
- Read-only Pruefer-Zugang (neues Feature!)

#### Aufbewahrung (Rn. 119-133 GoBD)
Siehe Aufbewahrungsfristen-Tabelle in 5.3.

**Confidence:** HIGH (GoBD-Schreiben ist ein stabiles BMF-Dokument)

---

### 5.2 Schweizer OR — Buchfuehrungspflichten

**Rechtsgrundlage:** Obligationenrecht (OR), Art. 957-958f

#### Art. 957 OR — Wer ist buchfuehrungspflichtig?

| Rechtsform | Buchfuehrungspflicht | Umfang |
|------------|---------------------|--------|
| Einzelunternehmen < 500.000 CHF Umsatz | Nur Einnahmen/Ausgaben + Vermoegen | Vereinfacht |
| Einzelunternehmen >= 500.000 CHF | Volle Buchfuehrung | Doppelte Buchhaltung |
| GmbH, AG, Genossenschaften | IMMER volle Buchfuehrung | Doppelte Buchhaltung |
| Vereine, Stiftungen | Je nach Groesse | Variiert |

#### Art. 957a OR — Grundsaetze ordnungsmaessiger Buchfuehrung
1. Vollstaendige, wahrheitsgetreue, systematische Erfassung
2. Belegpflicht fuer JEDE Buchung
3. Klarheit
4. Zweckmaessigkeit (abgestimmt auf Unternehmensgroesse)
5. Nachpruefbarkeit

#### Art. 958f OR — Aufbewahrung
- **Geschaeftsbuecher und Buchungsbelege:** 10 Jahre
- **Aufbewahrung auf Papier oder elektronisch** erlaubt
- **Elektronisch:** Muss jederzeit lesbar gemacht werden koennen
- **Fristbeginn:** Ende des Geschaeftsjahres

#### Schweizer MWST (Mehrwertsteuergesetz, MWSTG)
- Normalsatz: **8.1%** (seit 01.01.2024)
- Reduzierter Satz: **2.6%** (seit 01.01.2024)
- Sondersatz Beherbergung: **3.8%** (seit 01.01.2024)

**KMU Hub Finance-Modul muss:**
- Schweizer MWST-Saetze kennen (neben DE MwSt)
- Korrekte Rechnungsstellung nach Art. 26 MWSTG
- MWST-Nummer (CHE-xxx.xxx.xxx MWST) auf Rechnungen

**Confidence:** HIGH (OR-Artikel sind stabile Gesetzestexte; MWST-Saetze ab 2024 sollten nochmals verifiziert werden)

---

### 5.3 Aufbewahrungsfristen nach Dokumenttyp

#### Deutschland (HGB ss257 + AO ss147)

| Dokumenttyp | Frist | Rechtsgrundlage | KMU Hub Modul |
|-------------|-------|-----------------|---------------|
| **Handelsbuecher, Inventare, Bilanzen, Jahresabschluesse** | **10 Jahre** | ss257 Abs. 4 HGB | Finance |
| **Buchungsbelege** (Rechnungen, Quittungen, Kontoauszuege) | **10 Jahre** | ss257 Abs. 4 HGB, ss147 Abs. 1 Nr. 4 AO | Finance |
| **Empfangene Handels-/Geschaeftsbriefe** | **6 Jahre** | ss257 Abs. 4 HGB | E-Mail, Dokumente |
| **Kopien versandter Handels-/Geschaeftsbriefe** | **6 Jahre** | ss257 Abs. 4 HGB | E-Mail, Dokumente |
| **Rechnungen (Ein- und Ausgang)** | **10 Jahre** | ss14b UStG | Finance |
| **Vertraege (mit Buchungsrelevanz)** | **10 Jahre** | ss147 AO | Vertraege-Modul |
| **Vertraege (ohne Buchungsrelevanz)** | **6 Jahre** | ss257 HGB | Vertraege-Modul |
| **Personalakten** | **Bis 3 Jahre nach Austritt** (allg. Verjaehrung) + Lohnunterlagen **6 Jahre** | BGB ss195 + Lohnsteuer-RL | HR |
| **Lohn- und Gehaltsabrechnungen** | **6 Jahre** | ss41 Abs. 1 EStG | HR |
| **Arbeitszeitnachweise** | **2 Jahre** | ss16 Abs. 2 ArbZG | HR (Zeiterfassung) |
| **Unfallverhuetungs-Unterlagen** | **5 Jahre** | DGUV Vorschrift 1 | HR |
| **E-Mails (geschaeftsrelevant)** | **6 oder 10 Jahre** (je nach Inhalt) | ss257 HGB | E-Mail |
| **E-Mails (Rechnungen per E-Mail)** | **10 Jahre** | ss14b UStG | E-Mail, Finance |
| **Datenschutz-Dokumentation** | **3 Jahre** (nach Verarbeitungsende) | Art. 5 Abs. 2 DSGVO (Rechenschaftspflicht) | System |
| **Bewerbungsunterlagen (abgelehnt)** | **6 Monate** | AGG ss15 Abs. 4 | HR |

#### Schweiz (OR + MWSTG + ArG)

| Dokumenttyp | Frist | Rechtsgrundlage |
|-------------|-------|-----------------|
| **Geschaeftsbuecher + Buchungsbelege** | **10 Jahre** | Art. 958f OR |
| **Geschaeftskorrespondenz** | **10 Jahre** | Art. 958f OR |
| **MWST-Belege** | **10 Jahre** | Art. 70 Abs. 3 MWSTG |
| **Personalakten** | **5 Jahre nach Austritt** | OR + Verjaehrungsfristen |
| **Arbeitszeugnisse** | **10 Jahre** | OR Art. 330a |
| **Vertraege** | **10 Jahre nach Vertragsende** | Art. 958f OR |
| **Sozialversicherungs-Unterlagen** | **10 Jahre** | AHVG |

**Fristbeginn:** Immer Ende des Geschaeftsjahres, in dem das Dokument erstellt/empfangen wurde.

**Technische Umsetzung in KMU Hub:**

```
1. Retention-Policy Engine:
   - Jeder Datensatz hat ein retention_until Datum
   - Automatische Berechnung basierend auf Dokumenttyp + Erstelldatum
   - WARNUNG wenn Nutzer versucht, geschuetzte Daten zu loeschen
   - Automatische Benachrichtigung wenn Aufbewahrungsfrist ablaeuft

2. Legal Hold:
   - Admin kann "Legal Hold" auf Datensaetze setzen (Rechtsstreit)
   - Legal Hold ueberschreibt ALLE Loeschfristen
   - Audit-Log fuer Legal Hold Aktivierung/Deaktivierung

3. DSGVO vs. Aufbewahrungspflicht Konflikt:
   - Person fordert Loeschung (Art. 17 DSGVO)
   - ABER: Rechnung hat 10-Jahre Aufbewahrungspflicht
   - LOESUNG: Personendaten anonymisieren, Buchungsdaten behalten
   - "Max Mustermann" → "Geloeschte Person #12345", Rechnungsdaten bleiben
```

**Confidence:** HIGH (Aufbewahrungsfristen sind gut dokumentiert und stabil)

---

### 5.4 E-Mail-Archivierung

**Deutschland:**
- Geschaeftsrelevante E-Mails unterliegen denselben Aufbewahrungsfristen wie Geschaeftsbriefe
- **6 Jahre** fuer Handels-/Geschaeftsbriefe
- **10 Jahre** wenn E-Mail als Buchungsbeleg dient (z.B. Rechnung per E-Mail)
- ss257 HGB + ss147 AO

**Anforderungen:**
- E-Mails muessen im ORIGINAL aufbewahrt werden (nicht nur Ausdruck)
- Revisionssicher: Nicht nachtraeglich veraenderbar
- Durchsuchbar fuer Betriebspruefer

**Technische Umsetzung:**
```
1. Bei IMAP-Sync: Alle eingehenden/ausgehenden E-Mails archivieren
2. Archiv ist IMMUTABLE (Write-once)
3. Metadaten indiziert (Volltext-Suche)
4. Retention-Policy automatisch anhand Sender/Empfaenger/Betreff
5. Export-Funktion fuer Betriebspruefer (EML-Format + Index)
```

**Schweiz:**
- Geschaeftskorrespondenz = 10 Jahre (Art. 958f OR)
- E-Mails gelten als Geschaeftskorrespondenz wenn geschaeftsrelevant

**Confidence:** HIGH

---

### 5.5 Gesundheitsdaten — Falls Healthcare-KMUs Zielgruppe

**Rechtsgrundlage:** Art. 9 DSGVO (besondere Kategorien), Art. 5 lit. c nDSG

Gesundheitsdaten gehoeren zu den "besonderen Kategorien personenbezogener Daten" und unterliegen erhoehtem Schutz.

**Wann relevant fuer KMU Hub:**
- Arztpraxen, Physiotherapie, Pflegeheime als Kunden
- HR-Modul: Krankmeldungen (HR-07) enthalten implizit Gesundheitsdaten
- CRM: Wenn Kunde des Kunden ein Patient ist

**Zusaetzliche Anforderungen:**

| Anforderung | Details |
|-------------|---------|
| Rechtsgrundlage | Art. 9 Abs. 2 DSGVO: Ausdrueckliche Einwilligung ODER Arbeitsrecht ODER Gesundheitsversorgung |
| Verschluesselung | At-rest UND in-transit PFLICHT (nicht nur empfohlen) |
| Zugangsbeschraenkung | Streng nach Need-to-Know (nur behandelnder Arzt, HR fuer Krankmeldungen) |
| Datenschutz-Folgenabschaetzung | Art. 35 DSGVO: PFLICHT bei umfangreicher Verarbeitung von Gesundheitsdaten |
| Aufbewahrung Patientendaten | 10 Jahre nach letzter Behandlung (ss630f Abs. 3 BGB DE) |
| Schweigepflicht | ss203 StGB (DE): Strafbar bei Weitergabe von Patientendaten |
| nDSG (CH) | Art. 5 lit. c: "Besonders schuetzenswerte Personendaten" — ausdrueckliche Einwilligung ODER gesetzliche Grundlage |

**Empfehlung fuer KMU Hub:**
- HR-Modul: Krankmeldungen ALS GESUNDHEITSDATEN behandeln
  - Separater Zugriffsbereich (nur HR-Rolle)
  - Verschluesselt speichern
  - Nicht in allgemeinem Audit-Log (nur fuer HR sichtbar)
- Healthcare-KMUs: Self-Hosted DRINGEND empfehlen
- Kein Healthcare-spezifisches Feature in v1 (zu komplex, zu viel Regulierung)

**Confidence:** HIGH (Art. 9 DSGVO und ss203 StGB sind stabile Gesetze)

---

## 6. Praktische Umsetzung fuer KMU Hub

### 6.1 Multi-Tenancy — Datentrennung

**Anforderung:** Strikte Isolation zwischen Kundendaten (Mandanten).

**Architekturentscheidung: Shared Database, Separate Schemas vs. Row-Level-Isolation**

| Ansatz | Vorteile | Nachteile | Empfehlung |
|--------|----------|-----------|------------|
| **Separate Datenbank pro Tenant** | Maximale Isolation, einfache Backups pro Kunde | Teuer, komplex bei 100+ Kunden | Fuer Self-Hosted |
| **Shared DB, Separate Schemas** | Gute Isolation, moderate Kosten | Schema-Migrations muessen N-mal laufen | Gut, aber aufwaendig |
| **Shared DB, Row-Level Isolation** (tenant_id) | Einfach, skalierbar, kosteneffizient | Risiko bei fehlerhaften Queries (Cross-Tenant-Leak) | EMPFOHLEN fuer SaaS |

**Sicherstellung bei Row-Level Isolation:**

```sql
-- JEDE Tabelle hat tenant_id
ALTER TABLE contacts ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE invoices ADD COLUMN tenant_id UUID NOT NULL;
-- ... etc.

-- Row Level Security (PostgreSQL)
ALTER TABLE contacts ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON contacts
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Im Application Code (Go):
// Bei jedem Request:
tx.Exec("SET LOCAL app.current_tenant = $1", tenantID)
```

**PostgreSQL Row Level Security (RLS) ist der GOLDSTANDARD:**
- Automatisch auf DB-Ebene erzwungen
- Selbst bei Bugs im Application Code keine Cross-Tenant-Leaks
- Performance-Impact: Minimal (wird zu WHERE-Klausel optimiert)

**Zusaetzliche Sicherheit:**
- API-Middleware prueft tenant_id bei JEDEM Request
- Unit-Tests: Cross-Tenant-Access MUSS scheitern
- Jaehrlicher Pentest mit Fokus auf Tenant-Isolation

**Confidence:** HIGH (PostgreSQL RLS ist gut dokumentiert und bewaehrt)

### 6.2 Verschluesselung

#### At Rest (gespeicherte Daten)

| Schicht | Methode | Details |
|---------|---------|---------|
| **Festplattenverschluesselung** | LUKS (Linux) | Hetzner bietet verschluesselte Volumes |
| **Datenbank** | PostgreSQL TDE (Transparent Data Encryption) | Ab PostgreSQL 16 experimentell; Alternative: pgcrypto fuer Spalten |
| **Dateien** | AES-256 vor Upload in MinIO/S3 | Application-Level Encryption |
| **Backups** | AES-256-GCM | Verschluesselt BEVOR sie uebertragen werden |
| **Sensible Felder** | pgcrypto + Application-Level | Passwort-Hashes (bcrypt/argon2), API-Keys |

**Spalten-Level Verschluesselung fuer sensible Daten:**

```sql
-- Sensible Felder verschluesselt speichern
-- Schluessel pro Tenant (Customer-Managed Keys fuer Enterprise)
CREATE TABLE encrypted_fields (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    encrypted_value BYTEA NOT NULL,        -- AES-256-GCM verschluesselt
    nonce BYTEA NOT NULL,                   -- Unique per encryption
    created_at TIMESTAMPTZ DEFAULT now()
);
```

#### In Transit (Daten in Uebertragung)

| Verbindung | Protokoll | Minimum |
|------------|-----------|---------|
| Client ↔ API | TLS 1.3 | TLS 1.2 (1.0/1.1 deaktivieren!) |
| Service ↔ Service | mTLS (mutual TLS) | Empfohlen fuer Microservices |
| Service ↔ PostgreSQL | TLS | `sslmode=verify-full` |
| Service ↔ Redis | TLS | Seit Redis 6+ |
| Electron ↔ API | TLS + Certificate Pinning | Optional, erhoehte Sicherheit |
| LiveKit (Video) | DTLS-SRTP | WebRTC Standard |

**Confidence:** HIGH

### 6.3 Audit-Logging — Was muss geloggt werden?

**Rechtsgrundlagen:** Art. 5 Abs. 2 DSGVO (Rechenschaftspflicht), GoBD (Nachvollziehbarkeit), Art. 12 nDSG

#### Pflicht-Events (MUESSEN geloggt werden)

| Kategorie | Events |
|-----------|--------|
| **Authentifizierung** | Login (Erfolg + Fehlschlag), Logout, Passwort-Aenderung, 2FA-Aktivierung/Deaktivierung, Session-Erstellung/Terminierung |
| **Autorisierung** | Rollenaenderung, Berechtigungsaenderung, Zugriffsverweigerung |
| **Datenzugriff** | Export von Kundendaten, Bulk-Downloads, DSGVO-Auskunft, API-Key-Erstellung |
| **Datenmanipulation** | Erstellung/Aenderung/Loeschung von Kontakten, Rechnungen, Vertraegen |
| **Administration** | Benutzer anlegen/deaktivieren, Tenant-Konfiguration aendern, Plugin installieren |
| **Finanzen** | Rechnung erstellen/stornieren, Zahlung verbuchen (GoBD!) |
| **Compliance** | DSGVO-Loeschung, Consent-Aenderung, Data-Breach-Meldung |
| **System** | Backup erstellt, Migration ausgefuehrt, Service-Neustart |

#### Audit-Log Schema

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- WER
    user_id UUID,                          -- NULL fuer System-Events
    user_email VARCHAR(255),               -- Denormalisiert (falls User geloescht)
    user_role VARCHAR(50),
    ip_address INET,
    user_agent TEXT,

    -- WAS
    action VARCHAR(100) NOT NULL,          -- 'contact.created', 'invoice.finalized'
    category VARCHAR(50) NOT NULL,         -- 'auth', 'data', 'admin', 'finance', 'compliance'
    severity VARCHAR(20) NOT NULL,         -- 'info', 'warning', 'critical'

    -- WORAUF
    entity_type VARCHAR(50),              -- 'contact', 'invoice', 'user'
    entity_id UUID,
    entity_name VARCHAR(255),             -- Denormalisiert

    -- DETAILS
    old_values JSONB,                      -- Vorher-Werte (bei Aenderungen)
    new_values JSONB,                      -- Nachher-Werte (bei Aenderungen)
    metadata JSONB,                        -- Zusaetzliche Infos

    -- INTEGRITAET
    previous_hash VARCHAR(64),             -- SHA-256 des vorherigen Eintrags
    entry_hash VARCHAR(64) NOT NULL        -- SHA-256 dieses Eintrags (tamper-evident)
);

-- Indizes fuer performante Suche
CREATE INDEX idx_audit_tenant_time ON audit_logs(tenant_id, timestamp DESC);
CREATE INDEX idx_audit_user ON audit_logs(tenant_id, user_id);
CREATE INDEX idx_audit_entity ON audit_logs(tenant_id, entity_type, entity_id);
CREATE INDEX idx_audit_action ON audit_logs(tenant_id, action);
CREATE INDEX idx_audit_category ON audit_logs(tenant_id, category);

-- Partitionierung nach Monat (Performance bei grossen Datenmengen)
-- CREATE TABLE audit_logs (...) PARTITION BY RANGE (timestamp);
```

**Tamper-Evidence (Manipulationssicherheit):**
- Jeder Eintrag enthaelt SHA-256 Hash des vorherigen Eintrags (Blockchain-Prinzip)
- Nachtraegliche Manipulation bricht die Hash-Kette
- Regelmaessige Integritaetspruefung (Cron-Job)

**Bereits geplant als:** SEC-03 + SEC-04

**Confidence:** HIGH

### 6.4 Recht auf Vergessenwerden — Ueber alle Module hinweg

**Implementierungsstrategie: Kaskadierte Anonymisierung**

```
Ablauf:
1. Admin klickt "DSGVO-Loeschung" im Kontakt-Detail
2. System prueft Aufbewahrungspflichten (GoBD, OR)
3. System zeigt dem Admin:
   - Was SOFORT geloescht/anonymisiert werden kann
   - Was NICHT geloescht werden darf (mit Begruendung + Frist)
4. Admin bestaetigt
5. System fuehrt kaskadierte Anonymisierung aus
6. Audit-Log: "DSGVO-Loeschung ausgefuehrt fuer Kontakt #xxx"
```

**Anonymisierungsmatrix:**

| Modul | Tabelle | Aktion | Ausnahme |
|-------|---------|--------|----------|
| CRM | contacts | Name/E-Mail/Telefon → anonymisiert | -- |
| CRM | companies | NUR Kontakt-Verknuepfung loesen | Firma bleibt |
| CRM | deals | Kontakt-Referenz anonymisieren | Deal-Daten bleiben (Umsatzhistorie) |
| CRM | activities | Beschreibungen mit Personenbezug loeschen | Zeitstempel bleiben |
| E-Mail | emails | Absender/Empfaenger anonymisieren, Body loeschen | Header fuer Archiv behalten |
| Chat | messages | Nachrichten → "[Geloescht - DSGVO]" | -- |
| Chat | channels/DMs | Mitgliedschaft entfernen | Kanal bleibt |
| Kalender | events | Teilnehmer-Name anonymisieren | Event-Datum/Titel bleibt |
| Kalender | attendees | Loeschen | -- |
| Dateien | files | Dateien der Person loeschen | Geteilte Dateien: Ownership wechseln |
| Finance | invoices | Kontaktname anonymisieren, Rechnungsdaten BEHALTEN | **10 Jahre Aufbewahrung!** |
| Finance | transactions | Kontaktname anonymisieren | Buchungsdaten bleiben |
| HR | employees | ALLE Personaldaten loeschen (nach Sperrfrist) | Lohnunterlagen 6 Jahre |
| HR | time_entries | Mitarbeiter-Referenz anonymisieren | Stunden-Daten bleiben |
| Audit | audit_logs | User-Referenz → "Anonymisierter Nutzer #xxx" | Log-Eintraege bleiben |
| Vertraege | contracts | Vertragspartner anonymisieren | Vertragsdaten ggf. behalten (Aufbewahrung) |

**API-Endpunkt:**
```
POST /api/v1/gdpr/erasure
{
    "contact_id": "uuid",
    "reason": "Betroffenenanfrage Art. 17 DSGVO",
    "admin_confirmation": true,
    "override_retention": false    // true = Admin ueberschreibt Aufbewahrungsfrist (mit Warnung!)
}

Response:
{
    "status": "completed",
    "anonymized_records": 47,
    "retained_records": 3,
    "retained_reason": [
        { "module": "finance", "count": 3, "reason": "10-Jahres-Aufbewahrungspflicht ss257 HGB", "retention_until": "2036-12-31" }
    ],
    "audit_log_id": "uuid"
}
```

**Confidence:** HIGH

### 6.5 Datenexport — Format fuer Portabilitaet

**Art. 20 DSGVO / Art. 28 nDSG**

**Anforderung:** "Strukturiertes, gaengiges, maschinenlesbares Format"

**Empfohlenes Format:**

```
gdpr-export-2026-02-16-kontakt-12345.zip
├── manifest.json              # Metadaten, Schema-Version, Exportdatum
├── contact.json               # Kontaktdaten
├── emails/
│   ├── index.json             # E-Mail-Liste
│   └── attachments/           # E-Mail-Anhaenge
├── calendar_events.json       # Kalendereintraege
├── chat_messages.json         # Chat-Nachrichten
├── files/
│   ├── index.json             # Datei-Liste
│   └── downloads/             # Tatsaechliche Dateien
├── activities.json            # CRM-Aktivitaeten
├── invoices.json              # Rechnungen (anonymisiert wenn noetig)
├── time_entries.json          # Zeiterfassung
└── README.txt                 # Erklaerung fuer den Betroffenen
```

**manifest.json:**
```json
{
    "export_format": "kmuhub-gdpr-export",
    "format_version": "1.0.0",
    "exported_at": "2026-02-16T10:00:00Z",
    "exported_by": "admin@company.de",
    "subject": {
        "type": "contact",
        "id": "uuid",
        "name": "[Name des Betroffenen]"
    },
    "modules_included": ["crm", "email", "calendar", "chat", "files", "finance", "timetracking"],
    "total_records": 142,
    "total_files": 7,
    "schema_docs": "https://docs.kmuhub.de/gdpr-export-schema"
}
```

**Zusaetzlich fuer Betriebspruefer (GoBD Z3):**

```
gobd-export-2026.zip
├── index.xml                  # GDPdU/GoBD-konformer Index
├── rechnungen.csv             # Alle Rechnungen
├── buchungen.csv              # Alle Buchungen
├── kontenplan.csv             # Kontenrahmen
├── debitoren.csv              # Debitoren-Liste
├── kreditoren.csv             # Kreditoren-Liste
└── journal.csv                # Buchungsjournal
```

**DATEV-Export (FIN-06):**
- DATEV-Format fuer Buchungsstapel
- ASCII-feste Feldlaengen ODER CSV nach DATEV-Spec
- Felder: Umsatz, Soll/Haben, Konto, Gegenkonto, Belegdatum, Belegnummer, Buchungstext
- Header mit Beraternummer, Mandantennummer, WJ-Beginn

**Confidence:** HIGH

---

## 7. Wettbewerbsvorteil durch Datenschutz

### 7.1 Wie kann sich KMU Hub differenzieren?

**Die Realitaet im DACH-SaaS-Markt:**
- Die meisten CRM/SaaS-Loesungen sind US-basiert oder nutzen US-Infrastruktur
- DACH-KMUs sind zunehmend sensibel bezueglich Datensouveraenitaet
- Datenschutz ist NICHT nur Pflicht, sondern aktives Verkaufsargument

**Differenzierungsstrategie:**

| Feature | Wettbewerber | KMU Hub |
|---------|-------------|---------|
| Datenresidenz | USA (AWS/Azure/GCP) | 100% EU (Hetzner DE) + optional CH (Exoscale) |
| Sub-Processors | Dutzende US-Anbieter | Nur EU-Unternehmen |
| Self-Hosted | Meist nicht moeglich | Docker-Compose, eigene Infrastruktur |
| Datenschutz-Beratung | Nur Software | 1-Woche Onsite inkl. Prozessanalyse |
| GoBD-Konformitaet | Oft nachtraeglich | Von Anfang an designed |
| DSGVO-Tools | Basis (Cookie-Banner) | Vollstaendige Betroffenenrechte-Engine |
| Verschluesselung | TLS (Transport) | TLS + At-Rest + Optional Customer-Managed Keys |
| Audit-Trail | Basis-Logging | Tamper-evident, lueckenlos, GoBD-konform |
| Open Source | Geschlossen | Core als Source-available (Vertrauen) |

### 7.2 Marketing-Claims — Was darf man sagen?

| Claim | Erlaubt? | Bedingung |
|-------|----------|-----------|
| "100% EU-gehostet" | JA | Wenn ALLE Daten in EU bleiben (auch Backups, Logs) |
| "DSGVO-konform" | JA (mit Vorsicht) | Muss stimmen; kein Qualitaetssiegel, nur Gesetzeskonformitaet |
| "Swiss Data Residency" | JA | Wenn CH-Cluster angeboten wird |
| "ISO 27001 zertifiziert" | NUR wenn tatsaechlich zertifiziert | NICHT "nach ISO 27001" wenn nur angelehnt |
| "BSI C5 konforme Infrastruktur" | JA | Wenn Hosting-Provider BSI C5 hat (Hetzner) |
| "Ende-zu-Ende verschluesselt" | NUR wenn tatsaechlich E2E | TLS ist NICHT E2E! Nur wenn Client-seitig verschluesselt |
| "Datensouveraenitaet" | JA | Bei Self-Hosted Option |
| "DACH-SaaS von DACH-Unternehmen" | JA | Lokaler Anbieter, lokale Rechtsprechung |
| "Keine US-Cloud" | JA | Wenn wirklich keine AWS/Azure/GCP |
| "GoBD-konform" | JA (mit Vorsicht) | Wenn Anforderungen implementiert; idealerweise durch Steuerberater bestaetigt |

**Empfohlene Marketing-Messages:**

1. **Fuer Website/Landing Page:**
   > "Ihre Geschaeftsdaten gehoeren Ihnen. KMU Hub wird ausschliesslich auf europaeischen Servern betrieben — ohne US-Cloud, ohne Kompromisse."

2. **Fuer Schweizer Markt:**
   > "Swiss Data Residency: Ihre Daten bleiben in der Schweiz. Gehostet in Zuercher und Genfer Rechenzentren."

3. **Fuer Vertrieb:**
   > "Jeder Mitbewerber speichert Ihre Daten auf Amazon-Servern. Wir nicht."

4. **Fuer Trust-Page:**
   - Datenschutzerklaerung
   - TOMs zum Download
   - Sub-Processor-Liste (transparent)
   - AVV-Vorlage zum Download
   - Penetrationstest-Bericht (zusammengefasst)

### 7.3 Welche Tools VERSAGEN bei DSGVO?

| Tool | Problem | KMU Hub Vorteil |
|------|---------|-----------------|
| **Salesforce** | US-Unternehmen, CLOUD Act, keine Self-Hosted Option, Sub-Processors global | EU-only, Self-Hosted moeglich |
| **HubSpot** | US-Unternehmen, AWS-basiert, keine Datenresidenz-Garantie DE/CH | 100% EU-Hosting |
| **monday.com** | Israel/US, AWS-basiert, keine EU-Datenresidenz-Option | Lokale Infrastruktur |
| **Pipedrive** | Estland/US, AWS EU + US Regions gemischt | Keine AWS-Abhaengigkeit |
| **Notion** | US, AWS, keine Self-Hosted Option, Teams-Daten in US | Self-Hosted Option |
| **Slack** | US, AWS, keine EU-only Garantie | Eigener Chat, EU-only |
| **Zoom** | US, China-Kontroverse, Datenrouting-Probleme | LiveKit, self-hosted |
| **Trello** | Atlassian (US/Australien), AWS global | Lokales PM-Modul |
| **Asana** | US, keine EU-Datenresidenz | PM-Modul EU-only |

**Typische DSGVO-Versagen-Muster:**
1. **Sub-Processor-Kaskade:** Tool nutzt AWS → AWS nutzt Sub-Processors → Undurchsichtig
2. **Kein echtes Loeschrecht:** "Daten werden innerhalb von 90 Tagen geloescht" (nicht sofort!)
3. **Kein Datenexport:** Nur PDF-Export, nicht maschinenlesbar
4. **US-Jurisdiction:** CLOUD Act ermoeglicht US-Behoerden Zugriff auf EU-Daten
5. **Vendor Lock-in:** Daten nur im proprietaeren Format exportierbar
6. **Kein Self-Hosted:** Regulierte Branchen koennen das Tool nicht nutzen

**Confidence:** MEDIUM (Anbieter-spezifische Aussagen basieren auf Training-Daten; aktuelle Datenschutzpolicies der Anbieter sollten geprueft werden)

---

## Zusammenfassung — Compliance-Roadmap fuer KMU Hub

### Phase 1: Vor Beta-Launch (MUSS)

| Massnahme | Aufwand | Prioritaet |
|-----------|---------|-----------|
| AVV-Vorlage erstellen (lassen) | Rechtsanwalt, ~2.000-4.000 EUR | KRITISCH |
| Datenschutzerklaerung | Rechtsanwalt, ~1.000-2.000 EUR | KRITISCH |
| TOMs dokumentieren | Intern, 2-3 Tage | KRITISCH |
| Verarbeitungsverzeichnis (Art. 30) | Intern, 1-2 Tage | KRITISCH |
| Row-Level Security implementieren | Entwicklung, 1 Woche | KRITISCH |
| TLS ueberall | Entwicklung, 2-3 Tage | KRITISCH |
| Audit-Logging (Basis) | Entwicklung, 1 Woche | HOCH |
| Passwort-Policy + 2FA | Bereits geplant (SEC-01, SEC-09) | HOCH |
| Backup-Verschluesselung | Entwicklung, 1-2 Tage | HOCH |
| Externen DSB engagieren | ~300-500 EUR/Monat | EMPFOHLEN |

### Phase 2: Beta bis Launch (SOLL)

| Massnahme | Aufwand | Prioritaet |
|-----------|---------|-----------|
| DSGVO-Auskunft-Tool (SEC-05) | Entwicklung, 1-2 Wochen | HOCH |
| DSGVO-Loeschung (SEC-06) | Entwicklung, 2-3 Wochen | HOCH |
| Datenexport (Portabilitaet) | Entwicklung, 1 Woche | HOCH |
| GoBD-konformes Finance-Modul | Entwicklung, 2-3 Wochen | HOCH |
| Consent-Management | Entwicklung, 1 Woche | MITTEL |
| Retention-Policy Engine | Entwicklung, 1-2 Wochen | MITTEL |
| Incident-Response-Plan | Intern, 1-2 Tage | MITTEL |
| Penetrationstest | Extern, ~3.000-8.000 EUR | EMPFOHLEN |

### Phase 3: Nach Launch (KANN)

| Massnahme | Aufwand | Prioritaet |
|-----------|---------|-----------|
| ISO 27001 Vorbereitung | 6-12 Monate | MITTEL |
| Schweizer Cluster (Exoscale) | Entwicklung, 1-2 Wochen | MITTEL |
| Customer-Managed Encryption Keys | Entwicklung, 2-3 Wochen | NIEDRIG |
| DATEV-Export | Entwicklung, 1 Woche | MITTEL |
| GoBD-Verfahrensdokumentation Vorlage | Intern, 2-3 Tage | MITTEL |
| Trust-Center auf Website | Design + Content, 1 Woche | MITTEL |

### Geschaetzte Gesamtkosten Compliance (bis Launch)

| Posten | Kosten |
|--------|--------|
| Rechtsanwalt (AVV, DSE, Beratung) | 5.000-10.000 EUR |
| Externer DSB (12 Monate) | 3.600-6.000 EUR |
| Penetrationstest | 3.000-8.000 EUR |
| Entwicklungszeit (intern) | ~8-12 Wochen (inkludiert in Roadmap) |
| **Gesamt** | **~12.000-24.000 EUR** (+ Entwicklungszeit) |

---

## Quellen & Referenzen

**Gesetze (DE):**
- DSGVO (Verordnung (EU) 2016/679): https://dsgvo-gesetz.de/
- BDSG (Bundesdatenschutzgesetz): https://www.gesetze-im-internet.de/bdsg_2018/
- HGB ss257 (Aufbewahrungspflichten): https://www.gesetze-im-internet.de/hgb/__257.html
- AO ss147 (Ordnungsvorschriften): https://www.gesetze-im-internet.de/ao_1977/__147.html
- UStG ss14, ss14b (Rechnungen): https://www.gesetze-im-internet.de/ustg_1980/
- GoBD (BMF-Schreiben 28.11.2019): https://www.bundesfinanzministerium.de/
- ArbZG ss16 (Arbeitszeitaufzeichnung): https://www.gesetze-im-internet.de/arbzg/

**Gesetze (CH):**
- nDSG (SR 235.1): https://www.fedlex.admin.ch/eli/cc/2022/491/de
- DSV (SR 235.11): https://www.fedlex.admin.ch/eli/cc/2022/568/de
- OR Art. 957-958f: https://www.fedlex.admin.ch/eli/cc/27/317_321_377/de
- MWSTG: https://www.fedlex.admin.ch/eli/cc/2009/615/de

**Standards:**
- ISO 27001:2022: https://www.iso.org/standard/27001
- BSI C5: https://www.bsi.bund.de/C5
- TISAX: https://www.enx.com/tisax/

**Aufsichtsbehoerden:**
- BfDI (DE): https://www.bfdi.bund.de/
- EDOEB (CH): https://www.edoeb.admin.ch/

---

*Hinweis: Dieses Dokument stellt KEINE Rechtsberatung dar. Vor Umsetzung MUSS ein auf IT-Recht spezialisierter Rechtsanwalt konsultiert werden. Alle Paragraphen und Fristen basieren auf dem Stand der Training-Daten (Mai 2025). Gesetzesaenderungen nach diesem Datum sind nicht beruecksichtigt.*
