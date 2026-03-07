# Compliance-Framework: DSGVO/nDSG Deep Dive fuer KMU Hub

**Erstellt:** 2026-02-17
**Basis:** `05-dsgvo-dsg-compliance.md` (Uebersicht), `00-SYNTHESE.md` (Quick-Reference)
**Zweck:** Tiefgehende, modulspezifische Compliance-Anforderungen fuer Implementation
**Confidence:** MEDIUM-HIGH (Trainingsdaten Mai 2025; WebSearch/WebFetch nicht verfuegbar. Gesetze sind stabil, aber aktuelle Durchfuehrungsverordnungen und Urteile nach Mai 2025 sind nicht beruecksichtigt.)
**Hinweis:** Dieses Dokument stellt KEINE Rechtsberatung dar. Vor Implementation MUSS ein auf IT-Recht spezialisierter Rechtsanwalt konsultiert werden.

---

## Inhaltsverzeichnis

- [A) Technische Massnahmen pro Modul](#a-technische-massnahmen-pro-modul)
- [B) Organisatorische Massnahmen](#b-organisatorische-massnahmen)
- [C) Rechtliche Dokumente (Vorlagen-Struktur)](#c-rechtliche-dokumente-vorlagen-struktur)
- [D) Schweiz-spezifisch (nDSG)](#d-schweiz-spezifisch-ndsg)
- [E) Zertifizierungs-Roadmap](#e-zertifizierungs-roadmap)

---

## A) Technische Massnahmen pro Modul

### A.1 CRM-Modul — Consent-Management, Einwilligungsflags, Widerspruchsrecht

#### A.1.1 Rechtsgrundlagen pro Verarbeitungszweck

Jeder CRM-Kontakt muss einer Rechtsgrundlage zugeordnet sein. Das System muss folgende Rechtsgrundlagen abbilden:

| Zweck | Rechtsgrundlage | DSGVO-Artikel | Nachweis erforderlich |
|-------|-----------------|---------------|----------------------|
| Kundenverwaltung | Vertragserfuellung | Art. 6 Abs. 1 lit. b | Vertrag/Bestellung |
| Angebotsversand | Vorvertragliche Massnahme | Art. 6 Abs. 1 lit. b | Anfrage dokumentiert |
| Newsletter | Einwilligung | Art. 6 Abs. 1 lit. a | Double-Opt-in Nachweis |
| Telefon-Marketing | Einwilligung (DE: UWG ss7) | Art. 6 Abs. 1 lit. a | Nachweis Pflicht |
| Profiling/Scoring | Berechtigtes Interesse | Art. 6 Abs. 1 lit. f | Interessenabwaegung dokumentiert |
| Weitergabe an Dritte | Einwilligung | Art. 6 Abs. 1 lit. a | Ausdrueckliche Einwilligung |

#### A.1.2 Consent-Datenmodell (erweitert)

```sql
-- Kontakt-Level: Rechtsgrundlage
CREATE TABLE contact_legal_basis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    contact_id UUID NOT NULL REFERENCES contacts(id),
    legal_basis VARCHAR(30) NOT NULL,          -- 'consent', 'contract', 'legitimate_interest', 'legal_obligation'
    purpose VARCHAR(100) NOT NULL,             -- 'newsletter', 'phone_marketing', 'profiling', 'data_sharing'
    description TEXT,                           -- Freitext-Beschreibung des Zwecks
    granted BOOLEAN NOT NULL DEFAULT FALSE,
    granted_at TIMESTAMPTZ,
    granted_method VARCHAR(50),                -- 'double_opt_in', 'written', 'verbal', 'web_form', 'api'
    granted_evidence TEXT,                      -- URL zum Nachweis, E-Mail-ID, Formular-ID
    granted_ip INET,
    revoked_at TIMESTAMPTZ,
    revoked_method VARCHAR(50),
    revoked_reason TEXT,
    expires_at TIMESTAMPTZ,                    -- Optionales Ablaufdatum
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(tenant_id, contact_id, purpose)
);

CREATE INDEX idx_clb_contact ON contact_legal_basis(contact_id);
CREATE INDEX idx_clb_tenant_purpose ON contact_legal_basis(tenant_id, purpose);
CREATE INDEX idx_clb_expires ON contact_legal_basis(expires_at) WHERE expires_at IS NOT NULL;
```

#### A.1.3 Einwilligungs-Workflow (Double-Opt-in)

```
1. Kontakt gibt E-Mail-Adresse ein (Web-Formular, CRM-Import, manuell)
2. System sendet Bestaetigungs-E-Mail mit einmaligem Token (256-bit, 48h gueltig)
3. Kontakt klickt Bestaetigungslink
4. System loggt:
   - Timestamp der Bestaetigung
   - IP-Adresse
   - User-Agent
   - Token-ID (als Nachweis)
5. consent_records Eintrag wird auf granted=TRUE gesetzt
6. Kontakt erhaelt Bestaetigungs-E-Mail ("Sie haben sich erfolgreich angemeldet")
```

**Technische Anforderungen:**
- Double-Opt-in Token: `crypto/rand`, 32 Bytes, Base64-URL-encoded
- Token-Gueltigkeit: 48 Stunden (konfigurierbar pro Tenant)
- Token MUSS nach Verwendung invalidiert werden (One-Time-Use)
- Rate-Limiting: Max. 3 Bestaetigungs-E-Mails pro E-Mail-Adresse pro 24h

#### A.1.4 Widerspruchsrecht (Art. 21 DSGVO)

**Technische Umsetzung:**

| Anforderung | Implementation |
|-------------|----------------|
| Widerspruch gegen Direktwerbung | SOFORTIGE Einstellung, keine Abwaegung noetig (Art. 21 Abs. 3) |
| Widerspruch gegen berechtigtes Interesse | Abwaegung durch Admin, Dokumentation der Entscheidung |
| Automatisierte Sperrung | Bei Widerspruch gegen Newsletter: sofortige Entfernung aus allen Marketing-Listen |
| Kanal-spezifischer Widerspruch | Kontakt kann Newsletter erlauben aber Telefonmarketing verbieten |
| Unsubscribe-Header | Jede Marketing-E-Mail MUSS `List-Unsubscribe` und `List-Unsubscribe-Post` Header enthalten (RFC 8058) |

**API-Endpunkte:**

```
POST /api/v1/contacts/{id}/consent
  Body: { purpose: "newsletter", granted: true, method: "double_opt_in", evidence: "token_xyz" }

DELETE /api/v1/contacts/{id}/consent/{purpose}
  Body: { reason: "Widerspruch per E-Mail", method: "email" }

GET /api/v1/contacts/{id}/consent
  Response: [{ purpose, granted, granted_at, revoked_at, legal_basis, ... }]

GET /api/v1/consent/export?purpose=newsletter&granted=true
  Response: CSV/JSON aller aktiven Einwilligungen (fuer Nachweis)
```

#### A.1.5 CRM-spezifische Datenschutz-Features

| Feature | Beschreibung | Prioritaet |
|---------|-------------|-----------|
| Kontakt-Sichtbarkeit | `company_shared` vs. `private` (nur Ersteller sieht) | HOCH |
| Datenminimierung-Warnung | Wenn Kontakt >2 Jahre inaktiv: Admin-Warnung "Daten noch noetig?" | MITTEL |
| Import-Consent-Check | Bei CSV-Import: Pflichtfeld "Rechtsgrundlage" pro Kontakt | HOCH |
| Merge-Audit | Bei Duplikat-Zusammenfuehrung: Audit-Log welche Daten wohin | HOCH |
| Custom-Fields Sensitivitaet | Custom Fields koennen als "sensibel" markiert werden (eingeschraenkte Sichtbarkeit) | MITTEL |

---

### A.2 E-Mail-Modul — Verschluesselung, Signatur, Header-Schutz

#### A.2.1 Transport-Verschluesselung

| Verbindung | Minimum | Empfohlen | Implementation |
|------------|---------|-----------|----------------|
| IMAP-Sync | STARTTLS (Port 143) | IMAPS TLS (Port 993) | Go: `emersion/go-imap` mit `tls.Config{MinVersion: tls.VersionTLS12}` |
| SMTP-Send | STARTTLS (Port 587) | SMTPS (Port 465) | Go: `emersion/go-smtp` mit TLS |
| API (Client-Server) | TLS 1.2 | TLS 1.3 | Nginx/Caddy Reverse Proxy |

**WICHTIG:** Opportunistic TLS (STARTTLS) ist anfaellig fuer Downgrade-Attacken. Fuer sensible Kommunikation: IMAPS/SMTPS erzwingen.

#### A.2.2 E-Mail-Signatur und Integritaet

| Technologie | Zweck | Anforderung | Prioritaet |
|-------------|-------|-------------|-----------|
| DKIM | Absender-Authentifizierung | DNS-Record fuer eigene Domain | HOCH (fuer ausgehende Mails) |
| SPF | Autorisierte Mailserver | DNS TXT-Record | HOCH |
| DMARC | Richtlinie bei Verstoessen | DNS-Record, Policy: `reject` oder `quarantine` | HOCH |
| S/MIME | Ende-zu-Ende-Verschluesselung | Zertifikat pro User, PKCS#7 | NIEDRIG (v2+) |
| PGP/GPG | Ende-zu-Ende-Verschluesselung | Key-Pair pro User | NIEDRIG (v2+) |

**Fuer KMU Hub v1:** DKIM + SPF + DMARC fuer eigene Relay-Domain. S/MIME und PGP sind optionale v2-Features fuer Security-bewusste Kunden.

#### A.2.3 Header-Schutz und Metadaten

| Header | Risiko | Massnahme |
|--------|--------|-----------|
| `Received` | Legt interne Server-IPs offen | Interne Hops strippen (nur letzten oeffentlichen behalten) |
| `X-Mailer` | Software-Fingerprinting | Auf generischen Wert setzen oder entfernen |
| `X-Originating-IP` | IP des Absenders | NIEMALS setzen |
| `Message-ID` | Kann internen Hostnamen enthalten | Generieren mit eigener Domain: `<uuid@kmuhub.de>` |
| `References` / `In-Reply-To` | Threading-Info, potentiell sensitiv | Beibehalten (noetig fuer Threading) |

#### A.2.4 E-Mail-Archivierung (GoBD/OR-konform)

```
Archivierungspipeline:
1. Eingehende E-Mail wird von IMAP synchronisiert
2. Vollstaendige E-Mail (Header + Body + Attachments) wird als EML gespeichert
3. EML-Datei wird mit SHA-256 gehasht (Integritaetsnachweis)
4. Hash + Metadaten werden in DB gespeichert (IMMUTABLE)
5. EML wird verschluesselt (AES-256-GCM) in Object Storage (MinIO/S3) abgelegt
6. Original auf IMAP-Server bleibt unveraendert (KMU Hub modifiziert KEINE Mails auf dem Server)

Suchindex:
- Absender, Empfaenger, CC, BCC (anonymisiert bei DSGVO-Loeschung)
- Betreff
- Datum
- Anhang-Namen
- Volltext-Index des Body (fuer Betriebspruefer-Suche, GoBD Z2)

Aufbewahrung:
- Geschaeftsrelevante E-Mails: 6 Jahre (Geschaeftsbriefe) oder 10 Jahre (Rechnungen/Belege)
- System klassifiziert automatisch anhand von Mustern (Rechnung im Betreff, Betrag im Body)
- Admin kann Klassifizierung manuell ueberschreiben
- KEINE automatische Loeschung ohne Admin-Bestaetigung
```

#### A.2.5 Datenschutz-spezifische E-Mail-Features

| Feature | Beschreibung | Rechtsgrundlage |
|---------|-------------|-----------------|
| Tracking-Pixel opt-in | E-Mail-Oeffnungs-Tracking nur mit ausdruecklicher Einwilligung | Art. 5 Abs. 3 ePrivacy-RL / TTDSG ss25 |
| Unsubscribe-Header | `List-Unsubscribe` + `List-Unsubscribe-Post` in jeder Marketing-Mail | RFC 8058, Art. 7 Abs. 3 DSGVO |
| E-Mail-Loeschung bei DSGVO-Anfrage | Absender/Empfaenger anonymisieren, Body loeschen, EML-Archiv mit Sperrfrist | Art. 17 DSGVO |
| Auto-BCC Verhinderung | Kein automatisches BCC an Firmenadressen ohne Wissen des Senders | Fernmeldegeheimnis |

---

### A.3 HR-Modul — Personalakten-Schutz, Zugriffsrechte, Austritts-Workflow

#### A.3.1 Datenkategorien und Schutzniveaus

| Datenkategorie | Schutzniveau | Zugriffsberechtigte | Rechtsgrundlage |
|----------------|-------------|---------------------|-----------------|
| Stammdaten (Name, Adresse, Geburtsdatum) | NORMAL | HR, Admin, Manager (eigene MA) | Art. 6 Abs. 1 lit. b (Arbeitsvertrag) |
| Bankdaten (IBAN, BIC) | HOCH | HR, Admin | Art. 6 Abs. 1 lit. b + c |
| Gehaltsdaten | HOCH | HR, Admin | Art. 6 Abs. 1 lit. b + c |
| Krankmeldungen | BESONDERS GESCHUETZT | Nur HR | Art. 9 DSGVO (Gesundheitsdaten!) |
| Abmahnungen/Verwarnungen | HOCH | HR, Admin | Art. 6 Abs. 1 lit. f |
| Beurteilungen/Zeugnisse | HOCH | HR, Admin, betroffener MA | Art. 6 Abs. 1 lit. b |
| Zeiterfassungsdaten | NORMAL | HR, Admin, Manager (eigene MA), MA (eigene) | Art. 6 Abs. 1 lit. c (ArbZG) |
| Bewerbungsunterlagen | HOCH | HR, einstellender Manager | Art. 6 Abs. 1 lit. b |
| Fotos/Bilder | EINWILLIGUNG | Nach Einwilligung: alle | Art. 6 Abs. 1 lit. a |
| Schwerbehinderung | BESONDERS GESCHUETZT | Nur HR | Art. 9 DSGVO |
| Gewerkschaftszugehoerigkeit | BESONDERS GESCHUETZT | Nur HR | Art. 9 DSGVO |
| Religionszugehoerigkeit | BESONDERS GESCHUETZT | Nur HR (Kirchensteuer DE) | Art. 9 DSGVO |

#### A.3.2 Zugriffskontrolle HR-Modul

```
Hierarchie:
- HR-Rolle: Vollzugriff auf alle Personaldaten des Tenants
- Admin: Zugriff auf Stammdaten + Gehalt + Zeiterfassung, KEIN Zugriff auf Krankmeldungen
- Manager: Nur Stammdaten + Zeiterfassung der eigenen Team-Mitglieder
- Mitarbeiter: Nur eigene Daten (Self-Service)
- Extern: KEIN Zugriff auf HR-Modul

Spezialregel Krankmeldungen:
- Krankmeldungen werden in separater DB-Tabelle gespeichert
- Eigener RLS-Policy: NUR user.role = 'hr' ODER user.id = employee.user_id
- Krankmeldung wird NICHT im allgemeinen Audit-Log protokolliert
  (separates HR-Audit-Log, nur fuer HR sichtbar)
- Keine Anzeige von Diagnosen -- nur "krank von/bis" und "AU-Bescheinigung vorhanden ja/nein"
```

#### A.3.3 Mitarbeiter-Austritts-Workflow (Offboarding)

```
Trigger: HR setzt Mitarbeiterstatus auf "Austretend" mit Austrittsdatum

Sofort (am Austrittsdatum):
1. Deaktivierung des User-Accounts (Login gesperrt)
2. Alle aktiven Sessions invalidieren
3. API-Keys des Mitarbeiters revoken
4. Weiterleitung eingehender E-Mails an definierten Nachfolger (wenn konfiguriert)
5. Entzug aller Berechtigungen
6. Audit-Log: "Mitarbeiter {name} ausgetreten, Account deaktiviert"

Nach 7 Tagen:
7. E-Mail-Zugang vollstaendig entfernen (keine Weiterleitung mehr)
8. Chat-Nachrichten bleiben (Attributierung: "Ehemaliger Mitarbeiter")
9. Dateien: Ownership auf Manager oder HR uebertragen

Nach 30 Tagen:
10. Personalfoto entfernen (ausser Einwilligung zur weiteren Nutzung)
11. Nicht geschaeftsrelevante E-Mails des Mitarbeiters loeschen

Aufbewahrungsfristen (automatisch):
- Lohnunterlagen: 6 Jahre (DE) / 10 Jahre (CH) ab Jahresende des Austritts
- Personalakte (Stammdaten): 3 Jahre ab Austritt (allg. Verjaehrung DE)
- Arbeitszeugnisse: 10 Jahre
- Bewerbungsunterlagen (abgelehnte Bewerber): 6 Monate (AGG ss15 Abs. 4)

Nach Ablauf aller Fristen:
12. Automatische Benachrichtigung an HR: "Aufbewahrungsfrist abgelaufen fuer {name}"
13. HR bestaetigt Loeschung
14. Kaskadierte Anonymisierung (wie DSGVO Art. 17)
```

#### A.3.4 Besondere Kategorien personenbezogener Daten (Art. 9 DSGVO)

```sql
-- Gesundheitsdaten: Separate Tabelle mit eigener RLS-Policy
CREATE TABLE hr_health_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES employees(id),
    record_type VARCHAR(50) NOT NULL,    -- 'sick_leave', 'disability', 'occupational_health'
    start_date DATE NOT NULL,
    end_date DATE,
    has_certificate BOOLEAN DEFAULT FALSE,
    -- KEINE Diagnose-Details! Nur Metadaten.
    notes_encrypted BYTEA,               -- Verschluesselt, nur HR kann entschluesseln
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- RLS: Nur HR-Rolle oder betroffener Mitarbeiter selbst
ALTER TABLE hr_health_records ENABLE ROW LEVEL SECURITY;

CREATE POLICY hr_health_access ON hr_health_records
    USING (
        tenant_id = current_setting('app.current_tenant')::uuid
        AND (
            current_setting('app.current_role') = 'hr'
            OR employee_id = current_setting('app.current_employee')::uuid
        )
    );
```

---

### A.4 Dokumente-Modul — Verschluesselung at Rest, Audit-Trail, Loeschkonzept

#### A.4.1 Verschluesselung at Rest

```
Verschluesselungsarchitektur (3 Schichten):

Schicht 1: Volume-Level (Hetzner/Betriebssystem)
  - LUKS Full-Disk-Encryption auf allen Volumes
  - Schuetzt gegen physischen Diebstahl

Schicht 2: Object-Level (Application)
  - Jede Datei wird VOR Upload in MinIO/S3 verschluesselt
  - Algorithmus: AES-256-GCM (authentifizierte Verschluesselung)
  - Jede Datei erhaelt eigenen DEK (Data Encryption Key)
  - DEK wird mit Tenant-KEK (Key Encryption Key) verschluesselt
  - KEK liegt in separatem Key-Management-System (nicht in der gleichen DB!)

Schicht 3: Feld-Level (Datenbank)
  - Sensible Metadaten (z.B. Dateiname bei vertraulichen Dokumenten) mit pgcrypto verschluesselt
  - Standard-Metadaten (Groesse, MIME-Type, Created-At) im Klartext (fuer Suche)

Key-Hierarchie:
  Master Key (HSM oder Vault)
    └── Tenant KEK (pro Mandant)
         └── File DEK (pro Datei)
              └── Verschluesselte Datei
```

**Enterprise-Option: Customer-Managed Keys (CMK)**
- Kunde generiert eigenen KEK
- KMU Hub kann Daten nicht ohne Kunden-Key entschluesseln
- Vorteil: Maximale Datensouveraenitaet
- Nachteil: Bei Key-Verlust sind Daten unwiederbringlich verloren
- Empfehlung: Nur fuer Self-Hosted oder Enterprise-Tier anbieten

#### A.4.2 Audit-Trail fuer Dokumente

| Aktion | Was wird geloggt | Aufbewahrung |
|--------|-----------------|-------------|
| Upload | User, Dateiname, Groesse, Hash, Zielordner, Zeitpunkt | 10 Jahre (GoBD) |
| Download | User, Dateiname, IP, Zeitpunkt | 3 Jahre |
| Ansicht (Preview) | User, Dateiname, Zeitpunkt | 1 Jahr |
| Umbenennung | User, alter Name, neuer Name | 10 Jahre |
| Verschieben | User, alter Ordner, neuer Ordner | 10 Jahre |
| Berechtigungsaenderung | User, alte Berechtigung, neue Berechtigung | 10 Jahre |
| Loeschung | User, Dateiname, Grund (wenn angegeben) | 10 Jahre |
| Versionierung | User, Versionsnummer, Aenderungsbeschreibung | 10 Jahre |
| Link-Share erstellt | User, Datei, Ablaufdatum, Passwort ja/nein | 3 Jahre |
| Link-Share Zugriff | IP, User-Agent, Zeitpunkt | 1 Jahr |

#### A.4.3 Loeschkonzept (3-Stufen-Modell)

```
Stufe 1: Soft Delete (User loescht Datei)
  - Datei wird in Papierkorb verschoben
  - Metadaten behalten Flag: deleted_at, deleted_by
  - Datei ist innerhalb von 30 Tagen wiederherstellbar
  - Fuer alle Nicht-Admin-User unsichtbar

Stufe 2: Hard Delete (nach 30 Tagen oder Admin-Aktion)
  - Verschluesselte Datei wird aus Object Storage geloescht
  - File DEK wird vernichtet (Crypto-Shredding)
  - Metadaten werden anonymisiert: Dateiname -> "Geloeschte Datei #xxx"
  - Hash und Groesse bleiben im Audit-Log (Nachvollziehbarkeit)

Stufe 3: Crypto-Shredding (bei Tenant-Kuendigung oder DSGVO-Loeschung)
  - Tenant-KEK wird vernichtet
  - ALLE verschluesselten Dateien des Tenants sind sofort unlesbar
  - Physische Dateien koennen asynchron geloescht werden
  - Vorteil: Sofortige "Loeschung" ohne jeden einzelnen Datensatz anzufassen

Ausnahmen:
  - Dateien mit Aufbewahrungspflicht (Rechnungen, Belege) -> Retention-Lock
  - Dateien unter Legal Hold -> keine Loeschung moeglich
  - System warnt Admin: "Diese Datei unterliegt einer Aufbewahrungspflicht bis {datum}"
```

---

### A.5 Helpdesk-Modul — Kundendaten-Schutz, Ticket-Anonymisierung

#### A.5.1 Kundendaten im Helpdesk

| Datentyp | Sichtbarkeit | Schutzmassnahme |
|----------|-------------|-----------------|
| Name/E-Mail des Anfragenden | Agent + Admin | RLS auf Tenant-Ebene |
| Ticket-Inhalt (Freitext) | Agent + Admin | Kann sensible Daten enthalten -- Warnung bei Keywords |
| Anhaenge | Agent + Admin | Verschluesselt gespeichert, Virus-Scan vor Oeffnung |
| Interne Notizen | Nur Agenten/Admin | NIEMALS an Kunde sichtbar |
| SLA-Daten | Agent + Admin + Manager (Reports) | Keine personenbezogenen Daten |

#### A.5.2 Automatische Sensitivitaets-Erkennung

```
Bei Ticket-Erstellung/Update pruefen auf:
- IBAN/Kontonummer (Regex: DE\d{2}\s?\d{4}\s?\d{4}\s?\d{4}\s?\d{4}\s?\d{2})
- Kreditkartennummern (Luhn-Check)
- Sozialversicherungsnummern
- AHV-Nummern (CH: 756.xxxx.xxxx.xx)
- Passwort-Muster ("mein Passwort ist...")

Bei Treffer:
1. Ticket wird als "sensibel" markiert
2. Agent erhaelt Warnung: "Dieses Ticket enthaelt moeglicherweise sensible Daten"
3. Empfehlung: "Bitte fordern Sie den Kunden auf, sensible Daten nicht per Ticket zu senden"
4. Audit-Log: "Sensible Daten im Ticket #xxx erkannt"
```

#### A.5.3 Ticket-Anonymisierung bei DSGVO-Loeschung

```
Wenn Kontakt geloescht wird (Art. 17 DSGVO):

1. Tickets des Kontakts:
   - Kontaktname -> "Anonymisierter Kunde #xxx"
   - E-Mail -> "anon-xxx@deleted.local"
   - Ticket-Inhalt: LOESCHEN (nicht anonymisieren -- Freitext kann ueberall personenbezogene Daten enthalten)
   - Anhaenge: LOESCHEN
   - Interne Notizen: BEHALTEN (keine personenbezogenen Kundendaten)

2. Knowledge-Base-Artikel die aus Tickets erstellt wurden:
   - BEHALTEN (sollten bereits anonymisiert sein)
   - Pruefen ob Kundennamen/Daten enthalten, wenn ja: anonymisieren

3. SLA-Statistiken:
   - BEHALTEN (aggregiert, kein Personenbezug)
```

#### A.5.4 Kundenkommunikationskanal-Schutz

| Kanal | Verschluesselung | Authentifizierung | Datensparsamkeit |
|-------|-----------------|-------------------|------------------|
| E-Mail-zu-Ticket | TLS (Transport) | Absender-Verifizierung via DKIM | Nur Betreff + Body + Anhaenge speichern |
| Web-Formular | TLS + CSRF-Token | reCAPTCHA oder hCaptcha (EU!) | Nur Pflichtfelder abfragen |
| Chat-Widget (geplant) | WSS (WebSocket Secure) | Session-Token | Chat-Verlauf nach Ticket-Schliessung optional loeschen |
| Telefon (geplant) | N/A | Rueckruf zur Verifizierung | Keine Gespraechsaufzeichnung ohne Einwilligung |

---

### A.6 Finance-Modul — GoBD-Konformitaet, Aufbewahrungsfristen, Unveraenderbarkeit

#### A.6.1 GoBD-Anforderungen im Detail

**Die 10 GoBD-Gebote fuer KMU Hub:**

| Nr. | Gebot | Technische Umsetzung |
|-----|-------|---------------------|
| 1 | Lueckenlose Rechnungsnummern | Sequenz pro Nummernkreis, `SELECT nextval('invoice_seq_' || year)` mit Advisory Lock |
| 2 | Keine Aenderung nach Finalisierung | `is_finalized BOOLEAN`, DB-Trigger verhindert UPDATE bei `is_finalized = TRUE` |
| 3 | Storno statt Loeschung | DELETE auf finalisierte Rechnungen ist verboten. Stornobuchung mit Referenz auf Original. |
| 4 | Aenderungsprotokoll | Jede Aenderung vor Finalisierung wird geloggt (old_values, new_values) |
| 5 | Zeitnahe Erfassung | Warnung wenn Rechnung >10 Tage nach Leistungsdatum erstellt wird |
| 6 | Belegfunktion | Jede Buchung muss einem Beleg zugeordnet sein (Rechnung, Quittung, Vertrag) |
| 7 | Nachvollziehbarkeit | Wer hat wann welche Buchung erstellt/geaendert? Lueckenloser Trail. |
| 8 | Maschinelle Auswertbarkeit | Export in strukturiertem Format (DATEV CSV, GoBD-XML) |
| 9 | Verfahrensdokumentation | KMU Hub liefert Vorlage; Kunde muss sie ausfuellen und aktuell halten |
| 10 | Datensicherung | Taegliche verschluesselte Backups, georedundant |

#### A.6.2 Unveraenderbarkeits-Mechanismus

```sql
-- DB-Trigger: Verhindert Aenderung finalisierter Rechnungen
CREATE OR REPLACE FUNCTION prevent_finalized_invoice_update()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.is_finalized = TRUE THEN
        -- Nur status-Aenderungen auf 'cancelled' erlauben (Storno)
        IF NEW.status != 'cancelled' OR
           NEW.invoice_number != OLD.invoice_number OR
           NEW.total_amount != OLD.total_amount OR
           NEW.line_items != OLD.line_items THEN
            RAISE EXCEPTION 'Finalisierte Rechnungen duerfen nicht geaendert werden (GoBD Rn. 58-63)';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_finalized_invoice
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION prevent_finalized_invoice_update();

-- DELETE auf finalisierte Rechnungen komplett verhindern
CREATE OR REPLACE FUNCTION prevent_finalized_invoice_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.is_finalized = TRUE THEN
        RAISE EXCEPTION 'Finalisierte Rechnungen duerfen nicht geloescht werden (GoBD)';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_delete_finalized_invoice
    BEFORE DELETE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION prevent_finalized_invoice_delete();
```

#### A.6.3 Aufbewahrungsfristen-Engine

```sql
CREATE TABLE retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,         -- 'invoice', 'receipt', 'contract', 'email', 'hr_record'
    country VARCHAR(2) NOT NULL,               -- 'DE', 'CH', 'AT'
    retention_years INTEGER NOT NULL,
    legal_basis VARCHAR(200) NOT NULL,         -- 'HGB §257 Abs. 4', 'OR Art. 958f'
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Vorkonfigurierte Policies (Seed-Data)
INSERT INTO retention_policies (tenant_id, entity_type, country, retention_years, legal_basis, description) VALUES
-- Deutschland
('system', 'invoice',          'DE', 10, '§257 Abs. 4 HGB, §147 Abs. 1 Nr. 4 AO', 'Rechnungen und Buchungsbelege'),
('system', 'business_letter',  'DE',  6, '§257 Abs. 4 HGB', 'Handels- und Geschaeftsbriefe'),
('system', 'contract_finance', 'DE', 10, '§147 AO', 'Vertraege mit Buchungsrelevanz'),
('system', 'contract_other',   'DE',  6, '§257 HGB', 'Vertraege ohne Buchungsrelevanz'),
('system', 'payroll',          'DE',  6, '§41 Abs. 1 EStG', 'Lohn- und Gehaltsabrechnungen'),
('system', 'time_record',      'DE',  2, '§16 Abs. 2 ArbZG', 'Arbeitszeitnachweise'),
('system', 'hr_general',       'DE',  3, 'BGB §195', 'Personalakten (allg. Verjaehrung)'),
('system', 'application',      'DE',  0, 'AGG §15 Abs. 4', 'Bewerbungsunterlagen (6 Monate)'),
-- Schweiz
('system', 'invoice',          'CH', 10, 'Art. 958f OR', 'Geschaeftsbuecher und Buchungsbelege'),
('system', 'business_letter',  'CH', 10, 'Art. 958f OR', 'Geschaeftskorrespondenz'),
('system', 'contract_finance', 'CH', 10, 'Art. 958f OR', 'Vertraege'),
('system', 'payroll',          'CH', 10, 'AHVG', 'Sozialversicherungs-Unterlagen'),
('system', 'hr_general',       'CH',  5, 'OR Verjaehrung', 'Personalakten (nach Austritt)'),
-- Oesterreich
('system', 'invoice',          'AT',  7, '§132 BAO', 'Rechnungen und Buchungsbelege'),
('system', 'business_letter',  'AT',  7, '§132 BAO', 'Geschaeftskorrespondenz'),
('system', 'contract_finance', 'AT',  7, '§132 BAO', 'Vertraege');
```

#### A.6.4 Betriebspruefer-Zugang (GoBD Z1/Z2/Z3)

| Zugangsart | Beschreibung | Implementation |
|------------|-------------|----------------|
| Z1 (Unmittelbar) | Pruefer bekommt Read-only Zugang | Eigene Rolle `auditor` mit Read-only auf Finance-Modul |
| Z2 (Mittelbar) | Pruefer fordert Auswertungen an | Admin erstellt Reports nach Vorgabe |
| Z3 (Datentraeger) | Export aller Daten | DATEV-Export + GoBD-XML-Export |

```
Auditor-Rolle:
- Zugriff: Nur Finance-Modul (Rechnungen, Buchungen, Kontenplan)
- Rechte: Read-only, Export
- Kein Zugriff: CRM-Kontaktdetails, HR, Chat, E-Mail
- Zeitlich begrenzt: Zugang wird nach X Tagen automatisch deaktiviert
- Audit-Log: Jeder Zugriff des Pruefers wird geloggt
- 2FA: PFLICHT fuer Auditor-Zugang
```

---

### A.7 Zeiterfassung — Arbeitszeitgesetz, Datensparsamkeit

#### A.7.1 Arbeitszeitgesetz (ArbZG) Anforderungen (Deutschland)

**Rechtsgrundlage:** ss16 Abs. 2 ArbZG + EuGH-Urteil C-55/18 (CCOO, 14.05.2019) + BAG-Beschluss 1 ABR 22/21 (13.09.2022)

| Anforderung | Gesetz | KMU Hub Umsetzung |
|-------------|--------|-------------------|
| Aufzeichnung von Beginn, Ende und Dauer der taeglichen Arbeitszeit | ss16 Abs. 2 ArbZG (seit BAG 2022 fuer ALLE Arbeitgeber) | Timer mit Start/Stopp + manuelle Nacherfassung |
| Aufbewahrung: 2 Jahre | ss16 Abs. 2 ArbZG | Retention Policy: 2 Jahre ab Erfassung |
| Ueberstunden-Dokumentation | ss16 Abs. 2 ArbZG | Automatische Berechnung Soll vs. Ist |
| Pausenzeiten | ss4 ArbZG: >6h = 30min, >9h = 45min | Automatische Warnung bei fehlender Pause |
| Max. 10h/Tag | ss3 ArbZG | Warnung + Admin-Benachrichtigung bei Ueberschreitung |
| Ruhezeit 11h | ss5 ArbZG | Warnung wenn neue Schicht <11h nach letzter endet |
| Sonntagsarbeit | ss9 ArbZG | Markierung + Dokumentation der Ausnahme |

#### A.7.2 Schweizer Arbeitsgesetz (ArG)

| Anforderung | Gesetz | KMU Hub Umsetzung |
|-------------|--------|-------------------|
| Aufzeichnung der geleisteten Arbeit | Art. 46 ArG + ArGV 1 Art. 73 | Timer + manuelle Erfassung |
| Max. 45h/Woche (Industrie, Buero) | Art. 9 ArG | Automatische Warnung |
| Max. 50h/Woche (andere) | Art. 9 ArG | Konfigurierbar pro Mitarbeiter |
| Ueberzeitarbeit max. 170h/Jahr (45h-Woche) | Art. 12 ArG | Jaehrlicher Zaehler mit Warnung |
| Nachtarbeit-Zuschlag 25% | Art. 17b ArG | Automatische Berechnung |

#### A.7.3 Datensparsamkeit bei Zeiterfassung

| Prinzip | Umsetzung |
|---------|-----------|
| Nur erforderliche Daten erfassen | Zeit, Projekt, Taetigkeit. KEIN GPS-Tracking ohne Einwilligung. |
| Keine Verhaltenskontrolle | Tastaturanschlaege, Screenshots, Mausbewegungen werden NICHT erfasst |
| Aggregation fuer Reports | Manager sieht nur Stunden-Summen pro Tag/Woche, NICHT minutengenaue Logs |
| GPS nur mit Einwilligung | Falls Baubranche GPS-Zeiterfassung wuenscht: Opt-in pro Mitarbeiter |
| Pausen nicht detailliert | System erfasst Pause-Dauer, NICHT was MA in der Pause tut |
| Loeschung nach Frist | 2 Jahre (DE), automatische Anonymisierung danach |

```sql
-- Zeiterfassungs-Datenmodell mit Privacy-by-Design
CREATE TABLE time_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL REFERENCES employees(id),
    project_id UUID REFERENCES projects(id),       -- Optional
    task_id UUID REFERENCES tasks(id),             -- Optional
    activity_type VARCHAR(50),                      -- 'work', 'break', 'travel'
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    duration_minutes INTEGER,                       -- Berechnet, fuer schnelle Abfragen
    description TEXT,                               -- Taetigkeit (Freitext, optional)
    -- KEIN GPS-Feld in Standardtabelle!
    billable BOOLEAN DEFAULT FALSE,
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    source VARCHAR(20) DEFAULT 'manual',           -- 'timer', 'manual', 'import'
    retention_until DATE NOT NULL,                  -- Automatisch berechnet: start_time + 2 Jahre
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Optional: GPS-Erweiterung (nur wenn Mitarbeiter eingewilligt hat)
CREATE TABLE time_entry_locations (
    time_entry_id UUID PRIMARY KEY REFERENCES time_entries(id),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    accuracy_meters INTEGER,
    consent_id UUID NOT NULL REFERENCES contact_legal_basis(id),  -- Nachweis der Einwilligung!
    captured_at TIMESTAMPTZ DEFAULT now()
);
```

---

## B) Organisatorische Massnahmen

### B.1 Rollen-Konzept

#### B.1.1 System-Rollen (5 Basis-Rollen + 2 Spezialrollen)

| Rolle | Beschreibung | Scope | Zuweisbar durch |
|-------|-------------|-------|-----------------|
| `admin` | Tenant-Administrator. Volle Kontrolle. | Gesamter Tenant | System/Self |
| `manager` | Abteilungs-/Teamleiter. Erweiterte Rechte fuer eigenes Team. | Team/Abteilung | Admin |
| `member` | Standard-Mitarbeiter. Arbeitsrechte, keine Verwaltung. | Eigene Daten + geteilte | Admin, Manager |
| `hr` | Personalverantwortlicher. Zugriff auf HR-Daten. | HR-Modul + Mitarbeiterdaten | Admin |
| `it_support` | IT-Support. Technische Administration. | System-Einstellungen, kein Datenzugriff | Admin |
| `auditor` | Externer Pruefer (Steuerberater, Wirtschaftspruefer). Read-only Finance. | Finance-Modul (Read-only) | Admin |
| `extern` | Externer Mitarbeiter, Gast. Stark eingeschraenkt. | Nur explizit freigegebene Ressourcen | Admin, Manager |

#### B.1.2 Rollen-Hierarchie und Vererbung

```
admin
  ├── manager (erbt member-Rechte + Team-Verwaltung)
  │     └── member (Basis-Arbeitsrechte)
  ├── hr (eigener Zweig, NICHT unter manager)
  ├── it_support (eigener Zweig)
  ├── auditor (temporaer, Read-only)
  └── extern (minimal)
```

**Wichtig:** HR ist NICHT unter Manager. Ein Manager darf keine HR-Daten sehen (Gehalt, Krankmeldungen), auch nicht fuer sein eigenes Team. HR ist ein separater Berechtigungszweig.

### B.2 Berechtigungsmatrix pro Modul

#### B.2.1 Modul-Zugriff nach Rolle

| Modul | Admin | Manager | Member | HR | IT Support | Auditor | Extern |
|-------|-------|---------|--------|-----|------------|---------|--------|
| **Dashboard** | Voll | Voll | Eigenes | Voll (HR) | System | - | - |
| **CRM Kontakte** | CRUD | CRUD (Team) | CRUD (eigene) | Lesen | - | - | - |
| **CRM Deals** | CRUD | CRUD (Team) | CRUD (eigene) | - | - | - | - |
| **E-Mail** | Voll | Eigene | Eigene | Eigene | - | - | - |
| **Chat** | Voll | Voll | Voll | Voll | Voll | - | Eingeschraenkt |
| **Kalender** | Voll | Team + Eigene | Eigene | Team | - | - | Geteilte |
| **Projekte** | Voll | Team-Projekte | Zugewiesene | - | - | - | Eingeladene |
| **Zeiterfassung** | Voll | Team (Lesen) + Eigene | Eigene | Voll (Reports) | - | - | - |
| **HR** | Voll | Team-Stamm (Lesen) | Eigene (Self-Service) | Voll | - | - | - |
| **Finance** | Voll | Lesen (Team-Budgets) | Eigene Spesen | Lohnbuchhaltung | - | Lesen | - |
| **Helpdesk** | Voll | Team-Tickets | Zugewiesene | - | Zugewiesene | - | Eigene (Kunde) |
| **Dokumente** | Voll | Team + Geteilte | Eigene + Geteilte | HR-Ordner | IT-Ordner | - | Freigegebene |
| **Wiki** | Voll | CRUD | Lesen + Kommentare | HR-Wiki | IT-Wiki | - | Oeffentliche |
| **Schichtplanung** | Voll | CRUD (Team) | Lesen (eigene) | Voll | - | - | - |
| **Inventar** | Voll | CRUD | Lesen | - | - | - | - |
| **Einstellungen** | Voll | - | - | HR-Einstellungen | System-Einstellungen | - | - |

**Legende:** CRUD = Create/Read/Update/Delete, Voll = CRUD + Admin-Funktionen, Lesen = Read-only

#### B.2.2 Feld-Level-Berechtigungen (sensible Felder)

| Feld | Admin | Manager | Member | HR |
|------|-------|---------|--------|----|
| Mitarbeiter-Gehalt | Lesen | - | - | CRUD |
| Mitarbeiter-IBAN | Lesen | - | Eigene (Lesen) | CRUD |
| Mitarbeiter-Sozialversicherungsnr. | - | - | - | CRUD |
| Mitarbeiter-Krankmeldungen | - | - | Eigene (Lesen) | CRUD |
| Kontakt-Zahlungsinformationen | CRUD | Lesen | - | - |
| Deal-Marge/Rabatt | CRUD | CRUD | Lesen | - |

### B.3 Audit-Log Anforderungen

#### B.3.1 Was wird geloggt (Pflicht-Events)

| Kategorie | Events | Severity | Aufbewahrung |
|-----------|--------|----------|-------------|
| **Authentifizierung** | Login (Erfolg), Login (Fehlschlag), Logout, Passwort-Aenderung, 2FA An/Aus, Session-Erstellung, Session-Invalidierung, Passwort-Reset | CRITICAL (Fehlschlag), INFO (Erfolg) | 3 Jahre |
| **Autorisierung** | Rollenaenderung, Berechtigungsaenderung, Zugriffsverweigerung (403), Eskalation | WARNING | 3 Jahre |
| **Benutzerverwaltung** | User erstellen, deaktivieren, loeschen, Rolle aendern, E-Mail aendern | CRITICAL | 3 Jahre |
| **CRM-Daten** | Kontakt/Firma/Deal erstellen, aendern, loeschen, Import, Export, Merge | INFO | 3 Jahre |
| **Finance** | Rechnung erstellen, finalisieren, stornieren, Zahlung verbuchen, Export | CRITICAL | 10 Jahre (GoBD!) |
| **HR** | Mitarbeiter anlegen, Gehalt aendern, Austritt, Krankmeldung (separates Log!) | CRITICAL | 10 Jahre |
| **Dokumente** | Upload, Download, Loeschen, Berechtigungsaenderung, Link-Share | INFO | 10 Jahre |
| **DSGVO/Compliance** | Auskunftsanfrage, Loeschung, Consent-Aenderung, Data-Breach | CRITICAL | 10 Jahre |
| **System** | Backup, Migration, Service-Start/Stop, Konfigurationsaenderung | INFO | 3 Jahre |
| **Tenant-Administration** | Einstellungen aendern, Modul aktivieren/deaktivieren, API-Key erstellen/revoken | WARNING | 3 Jahre |

#### B.3.2 Audit-Log Integritaet

```
Hash-Ketten-Mechanismus (Tamper-Evidence):

1. Jeder Audit-Eintrag erhaelt einen SHA-256 Hash:
   hash = SHA-256(timestamp + user_id + action + entity_type + entity_id + old_values + new_values + previous_hash)

2. previous_hash = Hash des chronologisch vorherigen Eintrags desselben Tenants

3. Erster Eintrag eines Tenants: previous_hash = SHA-256("GENESIS-" + tenant_id)

4. Integritaetspruefung (Cron, taeglich):
   - Alle Eintraege der letzten 24h verifizieren
   - Bei Kettenbruch: CRITICAL-Alert an Admin + System-Admin
   - Monatlich: Vollpruefung aller Eintraege

5. Export fuer Wirtschaftspruefer:
   - CSV/JSON mit Hash-Spalte
   - Verifizierungsscript (Python/Go) mitliefern
   - Pruefer kann Kette unabhaengig validieren
```

#### B.3.3 Audit-Log Zugriff und Schutz

| Regel | Implementation |
|-------|----------------|
| Audit-Logs sind IMMUTABLE | Kein UPDATE/DELETE auf audit_logs Tabelle (nur INSERT) |
| Kein Admin kann Logs loeschen | Auch Admin hat keinen DELETE-Zugriff -- nur System-Admin auf Infrastruktur-Ebene |
| Logs sind tenant-isoliert | RLS auf audit_logs: Tenant sieht nur eigene Logs |
| HR-Logs sind separiert | hr_audit_logs: Nur HR-Rolle sieht Eintraege zu Krankmeldungen/Gehalt |
| Export nur fuer Admin | Audit-Export-Button nur fuer admin und auditor Rolle |
| Log-Rotation | Partitionierung nach Monat, archivierte Partitionen auf guenstigem Cold Storage |

### B.4 Backup-Strategie

#### B.4.1 Backup-Matrix

| Komponente | Frequenz | Methode | Aufbewahrung | Verschluesselung | Standort |
|------------|----------|---------|-------------|-----------------|----------|
| PostgreSQL (komplett) | Taeglich, 02:00 UTC | `pg_dump` (logical) | 30 Tage Rolling | AES-256-GCM | Hetzner Backup Space (anderes RZ) |
| PostgreSQL (WAL) | Kontinuierlich | WAL-Archivierung (PITR) | 7 Tage | AES-256-GCM | Hetzner Backup Space |
| Redis (Snapshots) | Alle 6 Stunden | RDB-Snapshot | 7 Tage | AES-256-GCM | Lokal + Remote |
| Object Storage (Dateien) | Taeglich (inkrementell) | `rclone sync` oder MinIO Replikation | 90 Tage | Bereits at-rest verschluesselt | OVH (DE) -- georedundant |
| Konfigurationen | Bei jeder Aenderung | Git-basiert | Unbegrenzt | Repository-Verschluesselung | Privates Git-Repo |
| Tenant-Export (on-demand) | Auf Anfrage | Vollexport aller Tenant-Daten | Bis Download (max. 7 Tage) | AES-256-GCM + Passwort | Temporaer, automatisch geloescht |

#### B.4.2 Recovery-Ziele

| Metrik | Ziel (SaaS) | Ziel (Self-Hosted) |
|--------|------------|-------------------|
| RPO (Recovery Point Objective) | < 1 Stunde (dank WAL) | Abhaengig von Kundenconfig |
| RTO (Recovery Time Objective) | < 4 Stunden | Abhaengig von Kundeninfrastruktur |
| Backup-Verifizierung | Woechentlich (automatisierter Restore-Test auf Staging) | Empfehlung an Kunden |
| Georedundanz | Hetzner (DE) + OVH (DE, anderes RZ) | Kundenverantwortung |

#### B.4.3 Backup-Verschluesselung

```
Verschluesselungs-Workflow:
1. pg_dump erzeugt Plaintext-Dump
2. Dump wird mit AES-256-GCM verschluesselt (Backup-Key)
3. Backup-Key wird mit Master-Key verschluesselt (Key-Wrapping)
4. Verschluesselter Dump + verschluesselter Key werden zusammen uebertragen
5. Master-Key liegt NICHT auf dem gleichen Server wie die Backups
6. Master-Key wird in separatem Key-Management gespeichert (z.B. Hetzner Secret Manager oder HashiCorp Vault)

Fuer Self-Hosted:
- Kunde erhaelt Backup-Script mit Verschluesselungs-Support
- Kunde verwaltet eigene Keys
- Dokumentation: "Backup & Recovery Guide" als Teil der Verfahrensdokumentation
```

### B.5 Incident-Response-Plan (72h-Frist, Meldekette)

#### B.5.1 Incident-Klassifizierung

| Schwere | Beschreibung | Beispiele | Reaktionszeit |
|---------|-------------|-----------|---------------|
| **SEV-1 (Kritisch)** | Datenpanne mit Zugriff auf personenbezogene Daten | Datenbank-Leak, unbefugter Zugriff auf Kundendaten, Ransomware | SOFORT (<1h) |
| **SEV-2 (Hoch)** | Sicherheitsvorfall ohne bestaetigten Datenzugriff | Brute-Force-Angriff, DDoS, Systemkompromittierung ohne Datenzugriff | <4 Stunden |
| **SEV-3 (Mittel)** | Potentielle Schwachstelle, kein aktiver Angriff | Vulnerability in Abhaengigkeit, fehlkonfigurierte Firewall | <24 Stunden |
| **SEV-4 (Niedrig)** | Informationell, kein direktes Risiko | Fehlgeschlagene Login-Versuche (einzeln), Port-Scans | Naechster Werktag |

#### B.5.2 Meldekette bei Datenpanne (SEV-1)

```
STUNDE 0: Erkennung
├── Automatisch: IDS/Monitoring-Alert -> PagerDuty/OpsGenie
└── Manuell: Mitarbeiter meldet an security@kmuhub.de

STUNDE 0-1: Erste Bewertung
├── Incident-Commander (IC) wird benannt (on-call Engineer)
├── IC bewertet: Sind personenbezogene Daten betroffen?
├── IC bewertet: Welche Tenants sind betroffen?
├── IC dokumentiert erste Erkenntnisse im Incident-Log
└── IC entscheidet: Eskalation ja/nein

STUNDE 1-4: Eindaemmung
├── Betroffene Systeme isolieren
├── Angriffsvektor identifizieren und schliessen
├── Forensische Sicherung (Disk-Images, Logs, Memory-Dumps)
├── Geschaeftsfuehrung informieren
└── Externen DSB informieren (wenn vorhanden)

STUNDE 4-24: Bewertung
├── Umfang der Panne bestimmen:
│   - Welche Daten sind betroffen?
│   - Wie viele Personen?
│   - Welche Tenants?
│   - Welche Datenkategorien (besondere Kategorien = hoeheres Risiko)?
├── Risikobewertung fuer Betroffene:
│   - Gering: z.B. nur E-Mail-Adressen offengelegt
│   - Hoch: z.B. Finanzdaten, Gesundheitsdaten, Passwoerter
│   - Sehr hoch: z.B. Identitaetsdiebstahl moeglich
└── Rechtsanwalt fuer IT-Recht konsultieren

STUNDE 24-48: Benachrichtigung (intern + Kunden)
├── ALLE betroffenen Kunden (Tenants) per E-Mail + In-App informieren:
│   - Was ist passiert?
│   - Welche Daten sind betroffen?
│   - Welche Massnahmen wurden ergriffen?
│   - Was muessen Kunden tun?
│   - Ansprechpartner + Hotline
├── Inhalt nach Art. 33 Abs. 3 DSGVO:
│   - Art der Verletzung
│   - Kategorien und ungefaehre Anzahl betroffener Personen
│   - Kontaktdaten des DSB
│   - Beschreibung wahrscheinlicher Folgen
│   - Beschreibung ergriffener Massnahmen
└── Kunden muessen eigenstaendig an IHRE Aufsichtsbehoerde melden (72h-Frist laeuft ab Kenntnis des KUNDEN!)

STUNDE 48-72: Behoerden-Meldung (durch Kunden)
├── KMU Hub unterstuetzt betroffene Kunden bei der Meldung:
│   - Vorausgefuelltes Meldeformular bereitstellen
│   - Technische Details liefern
│   - Fragen der Aufsichtsbehoerde beantworten
├── Falls KMU Hub SELBST Verantwortlicher ist (eigene MA-Daten):
│   - Meldung an zustaendige Aufsichtsbehoerde innerhalb 72h
│   - BayLDA (falls Firmensitz Bayern), sonst zustaendiges LfDI
└── Schweiz: Meldung an EDOEB "so rasch wie moeglich" (keine exakte Frist, aber SCHNELL)

NACH DEM INCIDENT:
├── Post-Mortem (blameless) innerhalb 5 Werktagen
├── Root-Cause-Analysis dokumentieren
├── Massnahmen zur Verhinderung definieren + umsetzen
├── Breach-Verzeichnis aktualisieren (Art. 33 Abs. 5 DSGVO)
├── Kunden ueber abgeschlossene Massnahmen informieren
└── Ggf. Betroffene direkt informieren (Art. 34 DSGVO: wenn hohes Risiko)
```

#### B.5.3 Breach-Verzeichnis (Art. 33 Abs. 5 DSGVO)

```sql
-- Jede Datenpanne muss dokumentiert werden, auch wenn KEINE Meldepflicht besteht
CREATE TABLE breach_register (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_date TIMESTAMPTZ NOT NULL,
    discovered_date TIMESTAMPTZ NOT NULL,
    description TEXT NOT NULL,
    affected_tenants UUID[],                       -- Betroffene Mandanten
    affected_data_categories TEXT[],                -- z.B. ['kontaktdaten', 'finanzdaten']
    affected_person_count INTEGER,                  -- Ungefaehre Anzahl
    risk_assessment VARCHAR(20) NOT NULL,           -- 'low', 'medium', 'high', 'very_high'
    reported_to_authority BOOLEAN DEFAULT FALSE,    -- Gemeldet an Aufsichtsbehoerde?
    authority_name VARCHAR(100),                     -- z.B. 'BayLDA', 'EDOEB'
    reported_at TIMESTAMPTZ,
    reported_to_affected BOOLEAN DEFAULT FALSE,     -- Betroffene informiert?
    measures_taken TEXT NOT NULL,                    -- Ergriffene Massnahmen
    root_cause TEXT,
    prevention_measures TEXT,                        -- Verhindungsmassnahmen
    post_mortem_url TEXT,                            -- Link zum Post-Mortem
    status VARCHAR(20) DEFAULT 'open',              -- 'open', 'investigating', 'contained', 'resolved'
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
```

#### B.5.4 Vorlagen fuer Incident-Response

**E-Mail-Vorlage an betroffene Kunden:**

```
Betreff: Sicherheitsvorfall bei KMU Hub — Information gemaess Art. 33 Abs. 2 DSGVO

Sehr geehrte/r [Anrede Kunde],

wir informieren Sie hiermit unverzueglich ueber einen Sicherheitsvorfall,
der Ihr KMU Hub-Konto betrifft.

1. Was ist passiert?
   [Beschreibung des Vorfalls]

2. Welche Daten sind betroffen?
   [Kategorien der betroffenen Daten]

3. Welche Massnahmen haben wir ergriffen?
   [Beschreibung der Sofortmassnahmen]

4. Was muessen Sie tun?
   - Als Verantwortlicher gemaess DSGVO obliegt Ihnen die Meldung an Ihre
     zustaendige Aufsichtsbehoerde innerhalb von 72 Stunden nach Kenntnis
     (Art. 33 DSGVO).
   - Wir stellen Ihnen ein vorausgefuelltes Meldeformular bereit: [Link]
   - Bewerten Sie, ob eine Information der betroffenen Personen erforderlich
     ist (Art. 34 DSGVO).

5. Kontakt
   Unser Datenschutzbeauftragter: [Name, E-Mail, Telefon]
   Incident-Hotline: [Telefonnummer]

Mit freundlichen Gruessen
[KMU Hub Geschaeftsfuehrung]
```

---

## C) Rechtliche Dokumente (Vorlagen-Struktur)

### C.1 AVV (Auftragsverarbeitungsvertrag) — Pflichtinhalt nach Art. 28 DSGVO

#### C.1.1 Gliederungsvorschlag

```
AUFTRAGSVERARBEITUNGSVERTRAG
gemaess Art. 28 DSGVO

zwischen
[Kunde — Verantwortlicher]
und
[KMU Hub GmbH — Auftragsverarbeiter]

§ 1 Gegenstand und Dauer der Verarbeitung
  - Bereitstellung der SaaS-Plattform "KMU Hub"
  - Dauer: Laufzeit des Hauptvertrags (SaaS-Vertrag)
  - Automatische Beendigung bei Vertragsende

§ 2 Art und Zweck der Verarbeitung
  - Speicherung und Verarbeitung von Geschaeftsdaten des Auftraggebers
  - Bereitstellung von CRM, Projektmanagement, Kommunikation, HR, Finance
  - Keine eigene Nutzung der Daten durch den Auftragsverarbeiter

§ 3 Art der personenbezogenen Daten
  - Kontaktdaten (Name, E-Mail, Telefon, Adresse)
  - Kommunikationsdaten (E-Mails, Chat-Nachrichten, Kalendereintraege)
  - Finanzdaten (Rechnungen, Zahlungsinformationen)
  - Personalstammdaten (bei Nutzung des HR-Moduls)
  - Zeiterfassungsdaten
  - Nutzungsdaten (Login-Zeiten, Feature-Nutzung)

§ 4 Kategorien betroffener Personen
  - Mitarbeiter des Auftraggebers
  - Kunden und Interessenten des Auftraggebers
  - Lieferanten und Geschaeftspartner des Auftraggebers
  - Bewerber (bei Nutzung des HR-Moduls)

§ 5 Weisungsgebundenheit (Art. 28 Abs. 3 lit. a)
  - Verarbeitung nur nach dokumentierter Weisung
  - Weisungsberechtigte Personen benennen
  - Verfahren bei rechtswidrigen Weisungen

§ 6 Vertraulichkeit (Art. 28 Abs. 3 lit. b)
  - Alle Mitarbeiter mit Datenzugang: Vertraulichkeitsverpflichtung
  - Schulungsnachweis Datenschutz

§ 7 Technische und Organisatorische Massnahmen (Art. 28 Abs. 3 lit. c)
  - Verweis auf TOMs (als Anlage 1)
  - Regelmaessige Ueberpruefung und Aktualisierung

§ 8 Unterauftragsverarbeiter (Art. 28 Abs. 3 lit. d)
  - Aktuelle Liste (Anlage 2)
  - Genehmigungsverfahren: Allgemeine Genehmigung mit Widerspruchsrecht
  - 30 Tage Vorabinformation bei Aenderungen
  - Widerspruchsrecht des Auftraggebers
  - Auftragsverarbeiter stellt sicher, dass Unterauftragsverarbeiter
    gleiche Datenschutzpflichten einhalten

§ 9 Unterstuetzung bei Betroffenenrechten (Art. 28 Abs. 3 lit. e)
  - Technische Unterstuetzung bei Auskunft (Art. 15)
  - Technische Unterstuetzung bei Loeschung (Art. 17)
  - Technische Unterstuetzung bei Datenportabilitaet (Art. 20)
  - Frist: Innerhalb von 5 Werktagen nach Anfrage

§ 10 Unterstuetzung bei Datenschutz-Folgenabschaetzung (Art. 28 Abs. 3 lit. f)
  - Bereitstellung notwendiger Informationen

§ 11 Loeschung und Rueckgabe (Art. 28 Abs. 3 lit. g)
  - Nach Vertragsende: Vollstaendiger Export aller Daten (JSON/CSV)
  - 90 Tage Frist fuer Export nach Vertragsende
  - Danach: Unwiderrufliche Loeschung aller Daten
  - Nachweis der Loeschung (Loeschprotokoll)

§ 12 Nachweispflicht und Audits (Art. 28 Abs. 3 lit. h)
  - Jaehrlicher Compliance-Bericht (SOC 2 oder vergleichbar, sobald vorhanden)
  - Recht auf Vor-Ort-Audit (mit 30 Tage Vorlaufzeit)
  - Kosten fuer Vor-Ort-Audit: [Regelung]
  - Alternative: Unabhaengiger Audit-Bericht akzeptiert

§ 13 Meldung von Datenpannen (Art. 33 Abs. 2)
  - Unverzuegliche Meldung an Auftraggeber (Ziel: <24h)
  - Inhalt der Meldung: Art der Panne, betroffene Daten, Massnahmen
  - Unterstuetzung bei Meldung an Aufsichtsbehoerde

§ 14 Datentransfer in Drittlaender
  - Grundsatz: Alle Daten verbleiben in EU/EWR/CH
  - Kein Transfer in Drittlaender ohne Genehmigung
  - Falls erforderlich: Standardvertragsklauseln (SCCs)

§ 15 Haftung und Schadensersatz
  - Verweis auf Art. 82 DSGVO
  - Haftungsbegrenzung gemaess Hauptvertrag

Anlage 1: Technische und Organisatorische Massnahmen (TOMs)
Anlage 2: Liste der Unterauftragsverarbeiter
Anlage 3: Weisungsberechtigte Personen
```

**Geschaetzte Kosten:** 2.000-4.000 EUR (Erstellung durch Rechtsanwalt). Vorlage kann dann fuer alle Kunden wiederverwendet werden.

### C.2 Datenschutzerklaerung — Mindestinhalt

#### C.2.1 Fuer KMU Hub Website + App

```
DATENSCHUTZERKLAERUNG

1. Verantwortlicher (Art. 13 Abs. 1 lit. a DSGVO)
   - Name, Anschrift, Kontakt der KMU Hub GmbH
   - Kontakt des DSB (wenn vorhanden)

2. Uebersicht der Verarbeitungstaetigkeiten (Art. 13 Abs. 1 lit. c)
   Fuer jede Verarbeitungstaetigkeit:
   a) Zweck der Verarbeitung
   b) Rechtsgrundlage (Art. 6 Abs. 1 DSGVO)
   c) Kategorien personenbezogener Daten
   d) Empfaenger oder Kategorien von Empfaengern
   e) Speicherdauer oder Kriterien fuer Bestimmung
   f) Hinweis auf Betroffenenrechte

3. Verarbeitungstaetigkeiten im Detail:
   3.1 Website-Besuch (Server-Logs)
   3.2 Kontaktaufnahme (E-Mail, Formular)
   3.3 Newsletter
   3.4 Kundenaccount / SaaS-Nutzung
   3.5 Bewerbungen
   3.6 Cookies und Tracking (falls zutreffend)

4. Empfaenger / Unterauftragsverarbeiter (Art. 13 Abs. 1 lit. e)
   - Hosting: Hetzner Online GmbH, Deutschland
   - E-Mail-Versand: [SMTP-Relay-Anbieter]
   - Video: LiveKit (Standort)
   - Zahlungsabwicklung: [Anbieter]

5. Uebermittlung in Drittlaender (Art. 13 Abs. 1 lit. f)
   - "Ihre Daten werden ausschliesslich innerhalb der EU/EWR verarbeitet."
   - Falls Ausnahmen: Rechtsgrundlage benennen (Angemessenheitsbeschluss, SCCs)

6. Speicherdauer (Art. 13 Abs. 2 lit. a)
   - Pro Verarbeitungszweck konkret angeben
   - Nicht: "so lange wie noetig" (zu unbestimmt!)

7. Betroffenenrechte (Art. 13 Abs. 2 lit. b-d)
   - Auskunftsrecht (Art. 15)
   - Berichtigungsrecht (Art. 16)
   - Loeschrecht (Art. 17)
   - Einschraenkung der Verarbeitung (Art. 18)
   - Datenportabilitaet (Art. 20)
   - Widerspruchsrecht (Art. 21)
   - Widerruf der Einwilligung (Art. 7 Abs. 3)
   - Beschwerderecht bei Aufsichtsbehoerde (Art. 77)

8. Pflicht zur Bereitstellung (Art. 13 Abs. 2 lit. e)
   - Welche Daten sind vertraglich oder gesetzlich erforderlich?
   - Folgen der Nichtbereitstellung

9. Automatisierte Entscheidungsfindung (Art. 13 Abs. 2 lit. f)
   - "Findet nicht statt" oder Beschreibung + Logik + Tragweite

10. Aenderungen der Datenschutzerklaerung
    - Datum der letzten Aenderung
    - Hinweis bei wesentlichen Aenderungen
```

**Schweiz-Erweiterung (nDSG Art. 19):**
- Identitaet des Verantwortlichen (auch bei indirekter Datenerhebung)
- Bearbeitungszweck
- Bei Weitergabe: Empfaengerstaat + Garantien
- "Datenschutzerklaerung nach DSGVO und nDSG" als Titel verwenden

### C.3 TOMs — Gliederung nach Art. 32 DSGVO

```
TECHNISCHE UND ORGANISATORISCHE MASSNAHMEN
gemaess Art. 32 DSGVO

Stand: [Datum]
Version: [Versionsnummer]

1. Vertraulichkeit (Art. 32 Abs. 1 lit. b)

   1.1 Zutrittskontrolle (physisch)
       - Rechenzentrum: [Anbieter], ISO 27001 zertifiziert
       - Zutrittskontrollsystem: [Beschreibung]
       - Besuchermanagement: [Beschreibung]

   1.2 Zugangskontrolle (logisch)
       - Authentifizierung: Passwort (min. 12 Zeichen) + 2FA (TOTP)
       - Session-Management: JWT (15min Access, 7d Refresh)
       - Brute-Force-Schutz: Sperre nach 5 Fehlversuchen
       - SSO: [Status]

   1.3 Zugriffskontrolle (Autorisierung)
       - RBAC mit 5 Basis-Rollen + 2 Spezialrollen
       - Mandantentrennung: PostgreSQL Row-Level Security
       - Least-Privilege-Prinzip
       - Quartalsweise Zugriffsueberprufung

   1.4 Trennungskontrolle
       - Multi-Tenancy mit strikter DB-Isolation
       - Keine mandantenuebergreifenden Abfragen
       - Getrennte Umgebungen: Production, Staging, Development

   1.5 Pseudonymisierung und Verschluesselung
       - Transport: TLS 1.3 (alle Verbindungen)
       - At Rest: AES-256-GCM (Dateien), LUKS (Volumes)
       - Datenbank: pgcrypto fuer sensible Felder
       - Passwoerter: Argon2id
       - Backup: AES-256-GCM verschluesselt

2. Integritaet (Art. 32 Abs. 1 lit. b)

   2.1 Weitergabekontrolle
       - TLS 1.3 fuer alle externen Verbindungen
       - mTLS fuer Service-zu-Service-Kommunikation
       - HSTS-Header auf allen Endpunkten
       - Kein unverschluesselter Datenversand

   2.2 Eingabekontrolle
       - Audit-Log mit Hash-Kette (Tamper-Evidence)
       - Versionierung bei Dokumenten und CRM-Daten
       - Aenderungshistorie bei allen geschaeftskritischen Entitaeten
       - Login-Log (IP, Timestamp, User-Agent)

3. Verfuegbarkeit und Belastbarkeit (Art. 32 Abs. 1 lit. b-c)

   3.1 Verfuegbarkeitskontrolle
       - Taegliche verschluesselte Backups
       - Georedundante Backup-Speicherung
       - Health-Checks fuer alle Microservices
       - DDoS-Schutz: [Anbieter/Methode]
       - RTO: < 4h, RPO: < 1h

   3.2 Rasche Wiederherstellbarkeit
       - Dokumentierter Disaster-Recovery-Plan
       - Regelmaessige Recovery-Tests (monatlich)
       - Blue-Green Deployment fuer Zero-Downtime Updates

4. Verfahren zur regelmaessigen Ueberpruefung (Art. 32 Abs. 1 lit. d)

   4.1 Datenschutz-Management
       - Externer DSB benannt: [ja/nein]
       - Verarbeitungsverzeichnis gefuehrt
       - Datenschutz-Folgenabschaetzung bei Bedarf
       - Jaehrliche Schulung aller Mitarbeiter

   4.2 Incident-Response
       - Dokumentierter Incident-Response-Plan
       - Eskalationsmatrix definiert
       - Breach-Verzeichnis gefuehrt
       - 72h-Meldefristen bekannt und dokumentiert

   4.3 Auftrags-/Zugangskontrolle
       - AVV mit allen Unterauftragsverarbeitern
       - Quartalsweise Review aller Zugriffsrechte
       - Onboarding/Offboarding-Prozess dokumentiert

   4.4 Penetrationstests
       - [Frequenz: jaehrlich]
       - [Letzter Test: Datum, Ergebnis]
       - [Naechster geplanter Test: Datum]

Anlage: Unterauftragsverarbeiter-Liste
```

### C.4 Verarbeitungsverzeichnis — Struktur nach Art. 30 DSGVO

```
VERZEICHNIS DER VERARBEITUNGSTAETIGKEITEN
gemaess Art. 30 Abs. 2 DSGVO (KMU Hub als Auftragsverarbeiter)

Pro Verarbeitungstaetigkeit:

| Feld | Beschreibung | Beispiel |
|------|-------------|---------|
| Nr. | Laufende Nummer | VV-001 |
| Bezeichnung | Name der Taetigkeit | CRM-Datenverarbeitung |
| Kategorien von Verarbeitungen | Art der Verarbeitung | Speicherung, Abruf, Aenderung, Loeschung |
| Kategorien betroffener Personen | Personengruppen | Kunden, Mitarbeiter, Lieferanten des Auftraggebers |
| Kategorien personenbezogener Daten | Datenarten | Kontaktdaten, Kommunikation, Finanzdaten |
| Empfaenger | An wen Daten gehen | Auftraggeber (ueber API), Unterauftragsverarbeiter |
| Drittlandstransfers | Uebermittlungen ausserhalb EU/EWR | Keine (nur EU-Hosting) |
| Loeschfristen | Wann werden Daten geloescht | 90 Tage nach Vertragsende |
| TOMs | Verweis auf Sicherheitsmassnahmen | Siehe TOM-Dokument Version X.Y |
| Rechtsgrundlage | DSGVO-Artikel | Art. 28 DSGVO (Auftragsverarbeitung) |

Verarbeitungstaetigkeiten fuer KMU Hub:

VV-001: CRM-Datenverarbeitung (Kontakte, Firmen, Deals)
VV-002: E-Mail-Verarbeitung (IMAP-Sync, SMTP-Versand)
VV-003: Chat-Kommunikation (Nachrichten, Kanaele)
VV-004: Video-Konferenzen (LiveKit)
VV-005: Projektmanagement (Aufgaben, Zeiterfassung)
VV-006: HR-Datenverarbeitung (Personalstammdaten, Zeiterfassung)
VV-007: Finanzdaten (Rechnungen, Zahlungen, Buchungen)
VV-008: Dokumentenverwaltung (Upload, Speicherung, Sharing)
VV-009: Helpdesk (Tickets, Kundenanfragen)
VV-010: Kalender (Termine, Einladungen)
VV-011: Schichtplanung (Arbeitszeitmodelle, Zuordnungen)
VV-012: Audit-Logging (Systemprotokollierung)
VV-013: Backup und Recovery
VV-014: Nutzungsanalyse (anonymisierte Telemetrie, falls aktiv)
```

---

## D) Schweiz-spezifisch (nDSG)

### D.1 Detaillierte Unterschiede DSGVO vs. nDSG

#### D.1.1 Anwendungsbereich

| Aspekt | DSGVO | nDSG | Konsequenz fuer KMU Hub |
|--------|-------|------|------------------------|
| Geschuetzte Personen | Natuerliche Personen | NUR natuerliche Personen (Art. 2 nDSG) | Firmendaten von CH-Kunden = kein Datenschutz unter nDSG |
| Raeumlicher Anwendungsbereich | Marktortprinzip (Art. 3 DSGVO) | Auswirkungsprinzip (Art. 3 nDSG) | KMU Hub unterliegt nDSG wenn CH-Kunden bedient werden |
| Territoriale Vertretung | Art. 27 DSGVO: EU-Vertreter | Art. 14 nDSG: Schweizer Vertretung fuer auslaendische Verantwortliche | KMU Hub (DE-Firma) braucht CH-Vertreter wenn CH-Kunden bedient werden |

**WICHTIG: Art. 14 nDSG — Pflicht zur Benennung einer Vertretung in der Schweiz**

Wenn KMU Hub als deutsches Unternehmen Daten von in der Schweiz ansaessigen Personen bearbeitet, MUSS eine Vertretung in der Schweiz benannt werden (Art. 14 Abs. 1 nDSG), sofern:
- Die Bearbeitung im Zusammenhang mit dem Angebot von Waren/Dienstleistungen steht ODER
- Die Bearbeitung der Verhaltensbeobachtung dient

**Empfehlung:** Schweizer Rechtsanwaltskanzlei oder Treuhandbuero als Vertretung benennen. Kosten: ca. 1.000-3.000 CHF/Jahr.

#### D.1.2 Einwilligung

| Aspekt | DSGVO | nDSG | Konsequenz |
|--------|-------|------|------------|
| Standard-Daten | Einwilligung oder andere Rechtsgrundlage (Art. 6) | Bearbeitung grundsaetzlich erlaubt (Erlaubnis mit Verbotsvorbehalt), ausser bei Verletzung der Persoenlichkeit | CH ist liberaler bei Standard-Daten |
| Besonders schuetzenswerte Daten | Ausdrueckliche Einwilligung (Art. 9) | Ausdrueckliche Einwilligung (Art. 6 Abs. 7 nDSG) | Gleich streng |
| Profiling mit hohem Risiko | Ausdrueckliche Einwilligung (Art. 22 DSGVO) | Ausdrueckliche Einwilligung (Art. 6 Abs. 7 nDSG) | Gleich streng |
| Kopplungsverbot | Art. 7 Abs. 4 DSGVO | Nicht explizit im nDSG | CH weniger streng |
| Einwilligungsfaehigkeit Minderjaehrige | Unter 16 (DE: 16, andere EU: 13-16) | Keine explizite Altersgrenze im nDSG; urteilsfaehige Minderjaehrige koennen einwilligen | CH flexibler |

#### D.1.3 Sanktionen

| Aspekt | DSGVO | nDSG | Konsequenz |
|--------|-------|------|------------|
| Bussgeldhoeche | Bis 20 Mio EUR / 4% Jahresumsatz | Bis 250.000 CHF | nDSG-Bussen deutlich niedriger |
| Adressat der Busse | Unternehmen | NATUERLICHE PERSON (Geschaeftsfuehrung, Verantwortliche) | Persoenliches Haftungsrisiko! |
| Subsidiaere Unternehmensbusse | N/A | Art. 64 nDSG: Bis 50.000 CHF wenn natuerliche Person nicht identifizierbar | Auffang-Tatbestand |
| Verschulden | Abhaengig von Mitgliedsstaat | NUR Vorsatz strafbar (Art. 60 ff. nDSG) | Fahrlaessigkeit nicht strafbar unter nDSG |
| Straftatbestaende | Verwaltungsrechtlich (Bussgeld durch Aufsichtsbehoerde) | Strafrechtlich (Strafbefehl/Urteil durch Strafbehoerde) | nDSG = Strafrecht, nicht Verwaltungsrecht! |

**Straftatbestaende im nDSG (Art. 60-66):**
1. Verletzung der Informationspflicht (Art. 60)
2. Verletzung der Sorgfaltspflicht bei Bekanntgabe ins Ausland (Art. 61)
3. Verletzung der Pflicht zur Ernennung einer Vertretung (Art. 63)
4. Verletzung der beruflichen Schweigepflicht (Art. 62)
5. Missachtung von Verfuegungen des EDOEB (Art. 63)

#### D.1.4 Datenschutz-Folgenabschaetzung (DSFA)

| Aspekt | DSGVO (Art. 35) | nDSG (Art. 22) |
|--------|-----------------|-----------------|
| Pflicht | Bei voraussichtlich hohem Risiko | Bei voraussichtlich hohem Risiko |
| Ausnahme | N/A | Wenn Verhaltenskodex nach Art. 11 nDSG eingehalten wird |
| Konsultation Behoerde | Art. 36 DSGVO: Aufsichtsbehoerde konsultieren | Art. 23 nDSG: EDOEB konsultieren |
| Erleichterung | N/A | Bei ernanntem Datenschutzberater: Konsultation EDOEB entfaellt (Art. 23 Abs. 4 nDSG) |

**Empfehlung:** Datenschutzberater in CH ernennen, um DSFA-Konsultationspflicht beim EDOEB zu vermeiden.

### D.2 Datenresidenz-Anforderungen Schweiz

#### D.2.1 Rechtslage

| Szenario | Erlaubt? | Rechtsgrundlage |
|----------|----------|-----------------|
| CH-Daten in DE/EU hosten | JA | EU hat Angemessenheitsbeschluss (Anhang 1 DSV) |
| CH-Daten in CH hosten | JA | Inlandsverarbeitung |
| CH-Daten in USA hosten | JA (seit 15.09.2024) | Swiss-US Data Privacy Framework |
| CH-Daten in UK hosten | JA | UK auf Laenderliste des Bundesrats |

**Aber: Markterwartung vs. Rechtslage**

Obwohl Hosting in DE fuer Schweizer Kunden rechtlich ERLAUBT ist, erwarten viele Schweizer KMUs Schweizer Datenresidenz. Gruende:
- "Swissness" als emotionaler Faktor
- Manche FINMA-regulierte Unternehmen fordern es
- Misstrauen gegenueber EU-Recht (abnehmend, aber vorhanden)
- Vertriebsargument: "Ihre Daten bleiben in der Schweiz"

#### D.2.2 Technische Umsetzung: Dual-Region

```
Architektur:

Region DE (Hetzner Falkenstein/Nuernberg):
  - Default fuer DE/AT-Kunden
  - PostgreSQL Primary + Replica
  - MinIO Object Storage
  - Redis Cache
  - Alle Microservices

Region CH (Exoscale Zuerich/Genf):
  - Fuer CH-Kunden mit "Swiss Data Residency" Option
  - Eigene PostgreSQL-Instanz (KEINE Replikation nach DE!)
  - Eigener MinIO Storage
  - Eigener Redis Cache
  - Alle Microservices

Routing-Logik:
  - Bei Tenant-Erstellung: Region wird festgelegt (DE oder CH)
  - DNS-basiertes Routing: de.api.kmuhub.de / ch.api.kmuhub.ch
  - Region ist UNVERAENDERLICH nach Erstellung (Migration nur manuell)
  - Kein Cross-Region-Datenfluss (strikte Isolation)

Konsequenz:
  - Zwei getrennte Cluster zu betreiben
  - Doppelte Infrastrukturkosten fuer CH
  - Rechtfertigt Aufpreis: z.B. +10 CHF/User/Monat
```

### D.3 Cross-Border DE <-> CH

#### D.3.1 Szenarien fuer KMU Hub Kunden

| Szenario | Rechtslage | Massnahmen |
|----------|-----------|------------|
| DE-Firma mit CH-Niederlassung | DSGVO gilt fuer DE-Teil, nDSG fuer CH-Teil | Ein Tenant, aber Datenschutzerklaerung muss beide abdecken |
| CH-Firma mit DE-Niederlassung | nDSG gilt fuer CH-Teil, DSGVO fuer DE-Teil (Marktortprinzip) | Gleich wie oben |
| DE-Firma verkauft an CH-Endkunden | DSGVO + nDSG (Auswirkungsprinzip Art. 3 nDSG) | Datenschutzerklaerung in DE und CH veroeffentlichen |
| CH-Firma sendet Daten nach DE (KMU Hub SaaS) | Zulässig (Angemessenheitsbeschluss) | Informationspflicht nach Art. 19 Abs. 4 nDSG |
| DE-Firma sendet Daten nach CH (Self-Hosted in CH) | Zulässig (Angemessenheitsbeschluss) | Informationspflicht nach Art. 13/14 DSGVO |

#### D.3.2 Doppel-Compliance Checkliste

Fuer KMU Hub Kunden die in DE UND CH taetig sind:

```
[ ] Datenschutzerklaerung deckt DSGVO UND nDSG ab
[ ] Verarbeitungsverzeichnis nach Art. 30 DSGVO UND Art. 12 nDSG
[ ] Informationspflichten nach Art. 13/14 DSGVO UND Art. 19 nDSG
[ ] Betroffenenrechte unterstuetzen nach DSGVO UND nDSG
[ ] DSFA nach Art. 35 DSGVO UND Art. 22 nDSG (falls noetig)
[ ] Datenpannenmeldung: 72h (DSGVO) UND "so rasch wie moeglich" (nDSG)
     -> Praxis: An BEIDE Behoerden melden (LfDI + EDOEB)
[ ] AVV nach Art. 28 DSGVO (nDSG kennt kein formelles AVV-Erfordernis,
     aber Art. 9 nDSG verlangt "hinreichende Gewaehrleistung")
[ ] Schweizer Vertretung benannt (Art. 14 nDSG), falls Verantwortlicher in DE
[ ] EU-Vertreter benannt (Art. 27 DSGVO), falls Verantwortlicher in CH
```

---

## E) Zertifizierungs-Roadmap

### E.1 Uebersicht: Welche Zertifizierung wann

| Zertifizierung | Relevanz fuer DACH-KMU-Markt | Wann anstreben | Geschaetzte Kosten | Geschaetzte Dauer |
|----------------|------------------------------|----------------|--------------------|--------------------|
| **ISO 27001:2022** | SEHR HOCH | 12-18 Monate nach Launch | 30.000-70.000 EUR | 6-12 Monate Vorbereitung |
| **BSI C5 Typ 1** | HOCH (oeffentliche Verwaltung) | 18-24 Monate nach Launch | 50.000-110.000 EUR | 6-9 Monate |
| **SOC 2 Type II** | MITTEL (internationale Kunden) | 24-36 Monate nach Launch | 40.000-85.000 EUR | 9-15 Monate |
| **TISAX** | NIEDRIG (nur Automotive) | Nur bei Nachfrage | 30.000-80.000 EUR | 6-9 Monate |
| **ISO 27701** | MITTEL (Privacy-Erweiterung) | Nach ISO 27001 | +15.000-30.000 EUR | 3-6 Monate (aufbauend) |

### E.2 ISO 27001:2022 — Detailplanung

#### E.2.1 Warum ISO 27001 zuerst

- Am weitesten verbreitete Sicherheitszertifizierung weltweit
- Im DACH-Raum der De-facto-Standard fuer IT-Sicherheit
- Groessere KMUs (50+ MA) und oeffentliche Auftraggeber FORDERN es
- Basis fuer weitere Zertifizierungen (BSI C5, TISAX bauen darauf auf)
- Zeigt professionellen Umgang mit Informationssicherheit

#### E.2.2 Die 93 Controls (ISO 27001:2022, Annex A)

ISO 27001:2022 hat die Controls von 114 (2013-Version) auf 93 reduziert und in 4 Kategorien gegliedert:

| Kategorie | Anzahl Controls | Beispiele |
|-----------|----------------|-----------|
| Organisatorisch | 37 | Informationssicherheitspolitik, Rollen, Risikomanagement, Asset-Management |
| Personell | 8 | Screening, Arbeitsvertrag, Awareness, Disziplinarverfahren |
| Physisch | 14 | Sicherheitsbereiche, Equipment, Speichermedien |
| Technologisch | 34 | Zugriffskontrolle, Kryptographie, Logging, Netzwerksicherheit, Secure Development |

**11 neue Controls in 2022:**
1. Threat Intelligence
2. ICT Readiness for Business Continuity
3. Information Security for Cloud Services
4. Physical Security Monitoring
5. Configuration Management
6. Information Deletion
7. Data Masking
8. Data Leakage Prevention
9. Monitoring Activities
10. Web Filtering
11. Secure Coding

#### E.2.3 Timeline und Kosten

```
MONAT 1-2: Gap-Analyse
  - Ist-Zustand der Informationssicherheit erfassen
  - Gegen ISO 27001 Controls abgleichen
  - Gap-Report erstellen
  - Kosten: Intern (3-5 Personentage) oder Berater (~5.000-10.000 EUR)

MONAT 3-4: ISMS aufbauen
  - Informationssicherheitspolitik definieren
  - Risikobehandlungsplan erstellen
  - Statement of Applicability (SoA) schreiben
  - Prozesse dokumentieren
  - Kosten: Intern (erheblicher Aufwand) + Berater (~10.000-20.000 EUR)

MONAT 5-8: Massnahmen implementieren
  - Technische Controls umsetzen (viele bereits durch Produktentwicklung abgedeckt!)
  - Organisatorische Massnahmen einfuehren
  - Schulungen durchfuehren
  - Interne Audits planen
  - Kosten: Primaer interne Arbeit

MONAT 9-10: Internes Audit
  - ISMS-Wirksamkeit pruefen
  - Korrekturmassnahmen
  - Management-Review
  - Kosten: Intern oder externer interner Auditor (~3.000-5.000 EUR)

MONAT 11-12: Zertifizierungsaudit
  - Stage 1: Dokumentationspruefung (1-2 Tage)
  - Stage 2: Implementation-Pruefung (3-5 Tage)
  - Kosten: 8.000-15.000 EUR (abhaengig von Scope und Zertifizierer)

DANACH (laufend):
  - Jaehrliches Ueberwachungsaudit: 5.000-10.000 EUR
  - Re-Zertifizierung alle 3 Jahre: 8.000-15.000 EUR
  - Kontinuierliche Verbesserung (PDCA-Zyklus)

GESAMTKOSTEN (erste Zertifizierung):
  - Mit Berater: 30.000-70.000 EUR
  - Ohne Berater (nur Audit): 15.000-30.000 EUR (aber hoeheres Risiko des Scheiterns)
  - Empfehlung: Mit Berater, da 3-Personen-Team kein ISMS-Know-how hat
```

#### E.2.4 Was KMU Hub bereits abdeckt (durch Produktentwicklung)

| ISO 27001 Control | KMU Hub Status | Luecke |
|-------------------|---------------|--------|
| A.5.1 Policies | Teilweise (CLAUDE.md, aber kein formelles ISMS-Dokument) | ISMS-Politik formalisieren |
| A.5.15 Access Control | Geplant (RBAC, RLS) | Noch zu implementieren |
| A.5.23 Information Security for Cloud | Geplant (EU-only Hosting) | Formell dokumentieren |
| A.8.1 User Endpoint Devices | N/A (SaaS) | Fuer Self-Hosted: Empfehlungen |
| A.8.5 Secure Authentication | Geplant (2FA, JWT) | Implementation ausstehend |
| A.8.9 Configuration Management | Teilweise (docker-compose) | Formelles Config Management |
| A.8.12 Data Leakage Prevention | Nicht geplant | Spaeter (z.B. DLP fuer E-Mail-Modul) |
| A.8.15 Logging | Geplant (SEC-03, SEC-04) | Implementation ausstehend |
| A.8.24 Use of Cryptography | Geplant (TLS, AES-256) | Kryptographie-Policy dokumentieren |
| A.8.25-28 Secure Development | Teilweise (Code-Review, Tests) | SDLC formalisieren |

### E.3 BSI C5 — Wann und warum

#### E.3.1 Wann ist BSI C5 noetig?

- **Pflicht:** Wenn oeffentliche Verwaltung in DE als Kundensegment
- **Empfohlen:** Wenn groessere Unternehmen (250+ MA) oder KRITIS-nahe Branchen
- **Nicht noetig:** Fuer Standard-KMU-Kunden (5-200 MA, privat)

BSI C5 ist seit 2024 zunehmend Voraussetzung fuer Cloud-Dienste in der oeffentlichen Verwaltung. Der Beschluss des IT-Planungsrats von 2024 empfiehlt BSI C5 fuer alle Cloud-Dienste im oeffentlichen Sektor.

#### E.3.2 BSI C5:2020 — Aktuelle Fassung

Die aktuelle Fassung BSI C5:2020 umfasst:
- 17 Themengebiete
- 121 Basiskriterien (MUESSEN erfuellt sein)
- Zusatzkriterien (SOLLTEN erfuellt sein, erhoehte Anforderungen)

**Die 17 Themengebiete:**
1. Organisation der Informationssicherheit (OIS)
2. Sicherheitsrichtlinien und Arbeitsanweisungen (SP)
3. Personal (HR)
4. Asset-Management (AM)
5. Physische Sicherheit (PS)
6. Regelbetrieb (OPS)
7. Identitaets- und Berechtigungsmanagement (IDM)
8. Kryptographie und Schluesselmanagement (CRY)
9. Kommunikationssicherheit (COS)
10. Portabilitaet und Interoperabilitaet (PI)
11. Beschaffung, Entwicklung, Aenderung von Informationssystemen (DEV)
12. Steuerung und Ueberwachung von Dienstleistern und Lieferanten (SSO)
13. Umgang mit Sicherheitsvorfaellen (SIM)
14. Kontinuitaetsmanagement und Notfallmanagement (BCM)
15. Compliance und Datenschutz (COM)
16. Umgang mit Ermittlungsanfragen (INQ)
17. Produktsicherheit (PSS)

#### E.3.3 Zwischenloesung: "BSI C5-konforme Infrastruktur"

**Marketing-Trick (legal und korrekt):**
- Hetzner ist BSI C5 Typ 2 zertifiziert
- KMU Hub kann kommunizieren: "Gehostet auf BSI C5-zertifizierter Infrastruktur (Hetzner Online GmbH)"
- Dies ist NICHT das Gleiche wie "KMU Hub ist BSI C5 zertifiziert" (das waere irreführend)
- Aber es zeigt: Infrastruktur-Layer erfuellt hoechste Standards

### E.4 SOC 2 Type II

#### E.4.1 Die 5 Trust Service Criteria

| Kriterium | Beschreibung | KMU Hub Relevanz |
|-----------|-------------|-----------------|
| Security | Schutz gegen unbefugten Zugriff | HOCH -- Kernkriterium |
| Availability | System-Verfuegbarkeit | HOCH -- SLA-Verpflichtungen |
| Processing Integrity | Korrekte Datenverarbeitung | HOCH -- GoBD, Rechnungen |
| Confidentiality | Schutz vertraulicher Informationen | HOCH -- Multi-Tenancy |
| Privacy | Schutz personenbezogener Daten | HOCH -- DSGVO/nDSG |

#### E.4.2 SOC 2 vs. ISO 27001 fuer DACH

| Aspekt | ISO 27001 | SOC 2 |
|--------|-----------|-------|
| Bekanntheit DACH | SEHR HOCH | MITTEL (steigend) |
| Bekanntheit international | HOCH | SEHR HOCH (USA) |
| Kosten (Erstmalig) | 30-70k EUR | 40-85k EUR |
| Dauer | 6-12 Monate | 9-15 Monate (Type II: 6-12 Mo Beobachtungszeitraum) |
| Wiederholung | Jaehrliches Ueberwachungsaudit | Jaehrlich (Type II) |
| Audit-Standard | Akkreditierte Zertifizierungsstelle | CPA-Firma (AICPA) |
| Ergebnis | Zertifikat (3 Jahre gueltig) | SOC 2 Report (1 Jahr gueltig) |

**Empfehlung:** ISO 27001 ZUERST (DACH-Markt), SOC 2 nur bei internationaler Expansion oder wenn US-Firmen als Kunden angefragt werden.

### E.5 Zertifizierungs-Timeline (Gesamtplan)

```
JAHR 1 (Launch bis +12 Monate):
├── Monat 1-6: Produktive Sicherheitsmassnahmen implementieren
│   ├── RLS, 2FA, Audit-Logging, TLS, Verschluesselung
│   ├── Penetrationstest (#1) durchfuehren
│   └── TOM-Dokument veroeffentlichen
├── Monat 6-12: ISO 27001 Vorbereitung starten
│   ├── Gap-Analyse
│   ├── ISMS aufbauen (Berater engagieren)
│   └── Massnahmen implementieren
└── Kosten: ~15.000-25.000 EUR (Berater + Pentest)

JAHR 2 (+12 bis +24 Monate):
├── Monat 12-18: ISO 27001 Zertifizierung
│   ├── Internes Audit
│   ├── Stage 1 + Stage 2 Audit
│   └── Zertifikat erhalten!
├── Monat 18-24: ISO 27001 leben + BSI C5 evaluieren
│   ├── Erstes Ueberwachungsaudit
│   ├── BSI C5 Gap-Analyse (nur wenn oeffentliche Verwaltung Zielgruppe)
│   └── Penetrationstest (#2)
└── Kosten: ~20.000-40.000 EUR (Zertifizierung + Berater + Pentest)

JAHR 3 (+24 bis +36 Monate):
├── BSI C5 Typ 1 (falls noetig)
├── SOC 2 Type I (falls internationale Kunden)
├── ISO 27701 (Privacy-Erweiterung zu ISO 27001)
├── ISO 27001 Ueberwachungsaudit #2
└── Kosten: Abhaengig von gewaehlten Zertifizierungen

GESAMTKOSTEN UEBER 3 JAHRE (realistisch):
  - Minimum (nur ISO 27001): ~50.000-80.000 EUR
  - Mittel (ISO 27001 + BSI C5 Typ 1): ~100.000-180.000 EUR
  - Maximum (ISO 27001 + BSI C5 + SOC 2): ~150.000-280.000 EUR

HINWEIS: Diese Kosten muessen durch Revenue getragen werden.
  Bei 100 zahlenden Kunden x 400 EUR/Monat = 480.000 EUR/Jahr
  -> Zertifizierungskosten = ~10-15% des ersten Jahres-Revenue
  -> Tragbar ab ~50 zahlende Kunden
```

---

## Anhang: Compliance-Checkliste nach Modul

### Checkliste: Vor Beta-Launch (MUSS)

```
Infrastruktur:
[ ] TLS 1.3 auf allen Endpunkten
[ ] PostgreSQL RLS auf allen Tabellen mit tenant_id
[ ] Verschluesselte Backups (AES-256-GCM)
[ ] Georedundante Backup-Speicherung
[ ] Keine US-Cloud-Services in der Lieferkette

Authentifizierung:
[ ] Passwort-Policy (min. 12 Zeichen, Komplexitaet)
[ ] 2FA (TOTP) fuer Admin-Accounts
[ ] JWT mit kurzer Laufzeit (15min Access)
[ ] Brute-Force-Schutz (5 Versuche, 15min Lockout)
[ ] Passwort-Hashing mit Argon2id

Audit-Logging:
[ ] Login-Events (Erfolg + Fehlschlag)
[ ] Datenmanipulation (CRUD auf allen Entitaeten)
[ ] Admin-Aktionen (Rollen, Berechtigungen, Konfiguration)
[ ] Hash-Kette fuer Tamper-Evidence
[ ] Audit-Logs IMMUTABLE (kein UPDATE/DELETE)

Rechtliche Dokumente:
[ ] AVV-Vorlage (vom Rechtsanwalt erstellt)
[ ] Datenschutzerklaerung (DSGVO + nDSG)
[ ] TOM-Dokument (Art. 32 DSGVO)
[ ] Verarbeitungsverzeichnis (Art. 30 DSGVO)
[ ] Sub-Processor-Liste (oeffentlich)

Organisation:
[ ] Externen DSB engagieren (empfohlen)
[ ] Incident-Response-Plan dokumentiert
[ ] Breach-Verzeichnis angelegt (leer, aber bereit)
[ ] Vertraulichkeitsverpflichtung aller Mitarbeiter
[ ] Datenschutz-Schulung fuer alle Team-Mitglieder
```

### Checkliste: Vor Go-Live (SOLL)

```
DSGVO-Tools:
[ ] Auskunftsrecht (Art. 15): Globale Suche + JSON/CSV-Export
[ ] Loeschrecht (Art. 17): Kaskadierte Anonymisierung
[ ] Datenportabilitaet (Art. 20): Strukturiertes ZIP-Paket
[ ] Consent-Management: Einwilligungsflags pro Kontakt/Zweck

Finance/GoBD:
[ ] Lueckenlose Rechnungsnummern
[ ] Unveraenderbarkeit (DB-Trigger)
[ ] Storno statt Loeschung
[ ] Aenderungsprotokoll
[ ] DATEV-CSV-Export
[ ] Auditor-Rolle (Read-only)

HR:
[ ] Krankmeldungen als Gesundheitsdaten geschuetzt
[ ] Separater HR-Audit-Log
[ ] Offboarding-Workflow implementiert
[ ] Aufbewahrungsfristen automatisch berechnet

E-Mail:
[ ] Archivierung (EML, IMMUTABLE)
[ ] DKIM + SPF + DMARC fuer eigene Relay-Domain
[ ] List-Unsubscribe Header in Marketing-Mails
[ ] E-Mail-Klassifizierung (geschaeftsrelevant ja/nein)

Zeiterfassung:
[ ] ArbZG-Warnungen (Pause, Max-Stunden, Ruhezeit)
[ ] 2-Jahres-Retention-Policy
[ ] Kein GPS ohne Einwilligung

Sicherheit:
[ ] Penetrationstest durchgefuehrt
[ ] Schwachstellen behoben
[ ] Security-Headers (HSTS, CSP, X-Frame-Options)
[ ] Rate-Limiting auf allen API-Endpunkten
[ ] CSRF-Schutz fuer alle mutierenden Endpoints
```

### Checkliste: Schweiz-spezifisch

```
[ ] Schweizer Vertretung benannt (Art. 14 nDSG)
[ ] Datenschutzerklaerung deckt nDSG ab
[ ] Informationspflicht bei Datenerhebung (Art. 19 nDSG)
[ ] Schweizer MWST-Saetze in Finance-Modul (8.1%/2.6%/3.8%)
[ ] QR-Rechnung (Pflicht seit 2022)
[ ] Swiss Data Residency Option evaluiert (Exoscale)
[ ] EDOEB als Aufsichtsbehoerde in Dokumentation erwaehnt
[ ] Cross-Border-Informationspflicht bei DE-CH-Datenfluss
```

---

## Quellen und Referenzen

**DSGVO/BDSG:**
- DSGVO (Verordnung (EU) 2016/679): https://dsgvo-gesetz.de/
- BDSG (Bundesdatenschutzgesetz 2018): https://www.gesetze-im-internet.de/bdsg_2018/
- Art. 28 DSGVO (Auftragsverarbeiter): AVV-Pflichtinhalt
- Art. 32 DSGVO (Sicherheit der Verarbeitung): TOM-Anforderungen
- Art. 33-34 DSGVO (Datenpannenmeldung): 72h-Frist

**nDSG/DSV (Schweiz):**
- nDSG (SR 235.1): https://www.fedlex.admin.ch/eli/cc/2022/491/de
- DSV (SR 235.11): https://www.fedlex.admin.ch/eli/cc/2022/568/de
- EDOEB-Leitfaeden: https://www.edoeb.admin.ch/

**GoBD/Handelsrecht:**
- GoBD (BMF-Schreiben 28.11.2019): GZ IV A 4 - S 0316/19/10003 :001
- HGB ss257: Aufbewahrungsfristen
- AO ss147: Ordnungsvorschriften fuer die Aufbewahrung von Unterlagen
- ArbZG ss16: Auszuege (Arbeitszeitnachweise)
- BAG-Beschluss 1 ABR 22/21 (13.09.2022): Pflicht zur Arbeitszeiterfassung

**Schweizer Arbeitsrecht/Handelsrecht:**
- ArG (Arbeitsgesetz): https://www.fedlex.admin.ch/eli/cc/1966/57_57_57/de
- OR Art. 957-958f: Buchfuehrungspflichten
- MWSTG: Mehrwertsteuergesetz

**Zertifizierungen:**
- ISO 27001:2022: https://www.iso.org/standard/27001
- BSI C5:2020: https://www.bsi.bund.de/C5
- SOC 2 (AICPA): https://us.aicpa.org/soc2

**EuGH-Rechtsprechung:**
- C-311/18 (Schrems II, 16.07.2020): Privacy Shield ungueltig
- C-55/18 (CCOO, 14.05.2019): Arbeitszeiterfassungspflicht

---

*Dieses Dokument stellt KEINE Rechtsberatung dar. Vor jeder konkreten Implementation MUSS ein auf IT-Recht spezialisierter Rechtsanwalt konsultiert werden. Alle Angaben basieren auf Trainingsdaten (Stand Mai 2025). Gesetze, Verordnungen und Rechtsprechung nach Mai 2025 sind nicht beruecksichtigt. Insbesondere koennte der EU-US Data Privacy Framework (DPF) inzwischen Aenderungen erfahren haben.*
