# Integrations-Guide: KMU Hub

**Datum:** 2026-02-17
**Zweck:** Detaillierte Anleitung fuer jede geplante Integration -- Lizenz, Kosten, API-Zugang, Setup-Schritte
**Confidence:** MEDIUM-HIGH (basierend auf offizieller Dokumentation, Preise Stand ~Q1 2025 -- VOR Verwendung live verifizieren)
**Referenz:** `00-SYNTHESE.md` Abschnitt 4 (Integrations-Strategie)

---

## Inhaltsverzeichnis

1. [OnlyOffice Document Server](#1-onlyoffice-document-server)
2. [Bexio API](#2-bexio-api)
3. [Skribble (E-Signatur)](#3-skribble-e-signatur)
4. [Brevo (Newsletter)](#4-brevo-ehem-sendinblue)
5. [CleverReach (Newsletter)](#5-cleverreach)
6. [FinAPI (Banking)](#6-finapi-banking)
7. [Nextcloud (WebDAV/CalDAV/CardDAV)](#7-nextcloud-webdavcaldavcarddav)
8. [DATEV-Export](#8-datev-export)
9. [Swiss QR-Code (QR-Rechnung)](#9-swiss-qr-code-qr-rechnung)
10. [ZUGFeRD / XRechnung](#10-zugferd--xrechnung)
11. [LiveKit (Video/Audio)](#11-livekit-videoaudio)
12. [TipTap (Rich-Text Editor)](#12-tiptap-rich-text-editor)

---

## 1. OnlyOffice Document Server

**Typ:** Open Source (Community) / Kommerziell (Enterprise)
**Lizenz:** AGPL v3 (Community), Proprietary (Enterprise)
**Hersteller:** Ascensio System SIA, Riga, Lettland (EU!)
**Website:** https://www.onlyoffice.com

### Lizenzmodell

| Edition | Lizenz | Gleichzeitige Verbindungen | Kosten |
|---------|--------|---------------------------|--------|
| **Community** | AGPL v3 | **20 gleichzeitige Verbindungen** (Hard-Limit) | Kostenlos |
| **Enterprise** | Proprietary | Unbegrenzt (lizenzbasiert) | Ab ~1.200 EUR/Jahr (50 Verbindungen) |
| **Developer** | Proprietary | Fuer SaaS-Einbettung | Individuell (Kontakt noetig) |

**WICHTIG Community Edition:**
- Das Limit von 20 gleichzeitigen Verbindungen bedeutet: Maximal 20 Nutzer koennen GLEICHZEITIG Dokumente bearbeiten.
- Fuer KMUs mit 5-20 MA ist das in der Praxis oft ausreichend (nicht alle bearbeiten gleichzeitig Dokumente).
- Das Limit ist hart codiert -- bei Ueberschreitung werden neue Verbindungen abgewiesen.
- Community Edition darf NICHT in einer kommerziellen SaaS-Plattform eingebettet werden, ohne den eigenen Code unter AGPL zu stellen (AGPL-Copyleft!).

**Enterprise Edition Kosten (circa, Stand 2025):**
- 50 Verbindungen: ~1.200 EUR/Jahr
- 100 Verbindungen: ~2.000 EUR/Jahr
- 300 Verbindungen: ~4.000 EUR/Jahr
- Unbegrenzt: Individuelles Angebot
- Enthaelt: Support, SLA, Mobile Editor, Jira/Confluence-Plugins, WOPI-Support

**Developer Edition (fuer SaaS-Anbieter wie KMU Hub):**
- Erlaubt Einbettung in eigene SaaS-Produkte
- Preis: Individuell, typischerweise ab ~2.000-4.000 EUR/Jahr
- MUSS lizenziert werden, wenn KMU Hub OnlyOffice als SaaS anbietet
- Bei Self-Hosted-Kunden: Kunden koennen Community Edition nutzen (eigene Instanz)

### WOPI-Protokoll Integration

**Was ist WOPI?**
Web Application Open Platform Interface -- ein Microsoft-Protokoll das erlaubt, Office-Dokumente in Web-Apps einzubetten. OnlyOffice unterstuetzt WOPI ab Version 6.4+.

**Alternativ:** OnlyOffice hat auch ein eigenes, einfacheres Protokoll (Document Server API / Callback-Konzept). Fuer neue Integrationen ist die hauseigene API oft einfacher als WOPI.

**Integration-Flow (Document Server API, empfohlen):**

```
1. KMU Hub Backend speichert .docx in S3/MinIO
2. Frontend oeffnet Editor-Iframe mit Config:
   {
     "document": {
       "fileType": "docx",
       "key": "unique-doc-key-v1",
       "title": "Vertrag.docx",
       "url": "https://kmuhub.example.com/api/files/123/download"
     },
     "editorConfig": {
       "callbackUrl": "https://kmuhub.example.com/api/onlyoffice/callback",
       "user": { "id": "user-456", "name": "Max Mueller" },
       "lang": "de"
     }
   }
3. OnlyOffice laedt Datei via URL
4. Nutzer bearbeitet im Browser
5. OnlyOffice schickt Aenderungen via callbackUrl zurueck
6. KMU Hub speichert neue Version
```

**Go-Backend: Benoetigte Endpoints:**
- `GET /api/files/{id}/download` -- Datei zum Lesen bereitstellen
- `POST /api/onlyoffice/callback` -- Callback von OnlyOffice empfangen (Status 2 = fertig bearbeitet, Datei herunterladen und speichern)
- `GET /api/onlyoffice/config/{id}` -- Editor-Config generieren
- JWT-Token-Generierung fuer sichere Kommunikation (HMAC-SHA256)

### Docker-Deployment

**System-Anforderungen:**
- **RAM:** Minimum 4 GB, empfohlen 8 GB (fuer 20+ gleichzeitige Nutzer)
- **CPU:** Minimum 2 Cores, empfohlen 4 Cores
- **Disk:** 2 GB fuer die Installation + Platz fuer Temp-Dateien
- **OS:** Linux (Debian/Ubuntu empfohlen), Docker 19.03+

**Schritt-fuer-Schritt Setup:**

```bash
# 1. Docker Image ziehen
docker pull onlyoffice/documentserver

# 2. Container starten (Community Edition)
docker run -i -t -d -p 8443:443 -p 8080:80 \
  --name onlyoffice-ds \
  -e JWT_SECRET=mein-geheimes-jwt-token \
  -v /app/onlyoffice/data:/var/www/onlyoffice/Data \
  -v /app/onlyoffice/logs:/var/log/onlyoffice \
  -v /app/onlyoffice/lib:/var/lib/onlyoffice \
  onlyoffice/documentserver

# 3. Health-Check
curl http://localhost:8080/healthcheck
# Erwartet: true

# 4. Test-Editor oeffnen
# Browser: http://localhost:8080
# -> Zeigt OnlyOffice Welcome Page

# 5. JWT konfigurieren (WICHTIG fuer Sicherheit!)
# In der Docker-Umgebung:
# JWT_ENABLED=true
# JWT_SECRET=mein-geheimes-jwt-token
# JWT_HEADER=Authorization
```

**docker-compose.yml fuer KMU Hub:**

```yaml
services:
  onlyoffice:
    image: onlyoffice/documentserver:latest
    container_name: kmuhub-onlyoffice
    restart: always
    ports:
      - "8443:443"
    environment:
      - JWT_ENABLED=true
      - JWT_SECRET=${ONLYOFFICE_JWT_SECRET}
      - JWT_HEADER=Authorization
    volumes:
      - onlyoffice_data:/var/www/onlyoffice/Data
      - onlyoffice_logs:/var/log/onlyoffice
      - onlyoffice_lib:/var/lib/onlyoffice
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/healthcheck"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  onlyoffice_data:
  onlyoffice_logs:
  onlyoffice_lib:
```

### Kosten fuer KMU Hub

| Szenario | Lizenz | Kosten/Jahr |
|----------|--------|-------------|
| Self-Hosted Kunden (je eigene Instanz) | Community (AGPL) | 0 EUR (Kunde betreibt selbst) |
| KMU Hub SaaS (multi-tenant) | **Developer Edition noetig** | ~2.000-5.000 EUR/Jahr (verhandeln!) |
| KMU Hub SaaS (gross, 300+ User) | Enterprise/Developer | ~4.000-8.000 EUR/Jahr |

**Sandbox/Test:** Keine spezielle Sandbox. Einfach lokal per Docker starten.

**Go Libraries:**
- Kein offizielles Go-SDK. REST-API direkt aufrufen (HTTP + JSON).
- JWT-Generierung: `github.com/golang-jwt/jwt/v5`
- Callback-Handler: Standard `net/http`

**DSGVO:** Lettische Firma (EU). Kein US-Datenfluss. Docker-Container laeuft auf eigenen Servern. DPA/AVV auf Anfrage (Enterprise).

**Integration-Aufwand:** 2-4 Wochen
- Woche 1: Docker-Setup + Go-Callback-Endpoints
- Woche 2: React-Frontend (Iframe-Einbettung + Editor-Config)
- Woche 3: Collaboration-Features (Co-Editing, Versionierung)
- Woche 4: Testing, Edge Cases (grosse Dateien, gleichzeitige Bearbeitung)

**UI-Anforderungen:**
- Dokument-Viewer/Editor als eingebettetes Iframe in Dokumente-Modul
- "In OnlyOffice bearbeiten"-Button bei .docx/.xlsx/.pptx-Dateien
- Editor oeffnet sich als Full-Width-Panel oder Overlay
- Speicher-Status-Anzeige (gespeichert / speichert...)
- Collaboration-Anzeige (wer bearbeitet gerade)

**Empfehlung:**
1. Lokal per Docker starten und Document Server API (nicht WOPI) evaluieren
2. Community Edition fuer Entwicklung und Self-Hosted-Kunden
3. Developer Edition verhandeln BEVOR SaaS-Launch
4. OnlyOffice Verkauf kontaktieren: sales@onlyoffice.com

---

## 2. Bexio API

**Typ:** Kommerziell (Freemium fuer API-Zugang)
**Lizenz:** Proprietary REST API
**Hersteller:** bexio AG, Rapperswil-Jona, Schweiz
**Website:** https://www.bexio.com | API: https://docs.bexio.com

### API-Zugang: Schritt fuer Schritt

**1. Bexio Developer-Account erstellen:**
```
1. https://developer.bexio.com registrieren
2. E-Mail bestaetigen
3. Im Developer Portal eine "App" anlegen
4. OAuth2 Client ID + Client Secret erhalten
```

**2. OAuth2 Flow:**

Bexio nutzt **Authorization Code Flow** (NICHT Client Credentials!):

```
1. KMU-Kunde klickt "Mit Bexio verbinden" in KMU Hub
2. Redirect zu: https://idp.bexio.com/authorize
   ?client_id=DEIN_CLIENT_ID
   &redirect_uri=https://app.kmuhub.com/integrations/bexio/callback
   &response_type=code
   &scope=openid profile email kb_invoice_edit contact_edit
   &state=random-csrf-token
3. Kunde loggt sich bei Bexio ein, gibt Berechtigung
4. Bexio redirectet zu callback_uri mit ?code=AUTH_CODE
5. Backend tauscht Code gegen Access Token + Refresh Token:
   POST https://idp.bexio.com/token
   {
     grant_type: "authorization_code",
     code: AUTH_CODE,
     redirect_uri: ...,
     client_id: ...,
     client_secret: ...
   }
6. Access Token speichern (verschluesselt!) + Refresh Token fuer Erneuerung
```

**Scopes (wichtig -- nur anfordern was noetig):**
- `contact_show`, `contact_edit` -- Kontakte lesen/schreiben
- `kb_invoice_show`, `kb_invoice_edit` -- Rechnungen
- `kb_offer_show`, `kb_offer_edit` -- Angebote
- `kb_order_show`, `kb_order_edit` -- Auftraege
- `banking_show` -- Bankbewegungen (nur lesen)
- `article_show`, `article_edit` -- Artikel/Produkte

### Rate Limits

- **Standard:** 300 Requests pro Minute pro Access Token
- **Burst:** Kurzzeitig hoehere Rate moeglich
- HTTP 429 bei Ueberschreitung
- `X-RateLimit-Remaining` Header beachten
- **Empfehlung:** Queue + Backoff-Strategie implementieren

### Sandbox / Test-Umgebung

- **Ja!** Bexio stellt eine Sandbox bereit.
- Sandbox-URL: `https://sandbox.bexio.com` (separate Umgebung)
- Test-Accounts mit vordefinierten Daten
- OAuth2-Flow funktioniert identisch, nur andere Base-URL
- ACHTUNG: Sandbox wird regelmaessig zurueckgesetzt

### Partner-Programm

```
1. https://www.bexio.com/de-CH/partner besuchen
2. Formular ausfuellen: Firmenname, Beschreibung der Integration
3. Bexio prueft und schaltet erweiterten API-Zugang frei
4. Partner erhalten:
   - Erhoehte Rate Limits
   - Listing im Bexio Marketplace (optional)
   - Technischen Support
   - Ggf. Provisionen bei Neukunden-Vermittlung
```

**Kosten fuer KMU Hub als Integrator:**
- API-Zugang: **Kostenlos** (kein API-Gebuehren-Modell)
- Partner-Programm: **Kostenlos** (Bexio verdient an den Kunden-Abos)
- KMU Hub Kunden brauchen ein eigenes Bexio-Abo (ab ~29 CHF/Mo)

### Benoetigte Endpoints

| Bereich | Endpoints | Richtung |
|---------|-----------|----------|
| **Kontakte** | `GET/POST/PATCH /2.0/contact` | Bidirektional (KMU Hub <-> Bexio) |
| **Firmen** | `GET/POST/PATCH /2.0/contact` (type=company) | Bidirektional |
| **Rechnungen** | `GET/POST /2.0/kb_invoice` | KMU Hub -> Bexio (Export) |
| **Angebote** | `GET/POST /2.0/kb_offer` | KMU Hub -> Bexio |
| **Zahlungen** | `GET /2.0/banking/account` + `/banking/entry` | Bexio -> KMU Hub (Import) |
| **Artikel** | `GET/POST /2.0/article` | Bidirektional |
| **MWSt-Saetze** | `GET /2.0/tax` | Bexio -> KMU Hub |
| **Waehrungen** | `GET /2.0/currency` | Bexio -> KMU Hub |

### Sync-Strategie

```
1. Initial-Sync: Alle Kontakte + Rechnungen von Bexio holen (Pagination!)
2. Laufend: Webhook-basiert ODER Polling alle 5 Minuten
3. Bexio hat KEINE Webhooks (!) -> Polling noetig
4. Delta-Sync: "modified_since" Parameter nutzen
5. Konflikt-Strategie: "Last Write Wins" mit manueller Konfliktanzeige
```

**Go Libraries:**
- Kein offizielles Go-SDK
- REST-API direkt mit `net/http` oder `github.com/go-resty/resty/v2`
- OAuth2: `golang.org/x/oauth2`
- JSON-Mapping: Standard `encoding/json`

**DSGVO:** Schweizer Firma, Daten in der Schweiz. Angemessenheitsbeschluss EU<->CH vorhanden. AVV/DPA verfuegbar.

**Integration-Aufwand:** 2-4 Wochen
- Woche 1: OAuth2-Flow + Token-Management
- Woche 2: Kontakt-Sync (bidirektional)
- Woche 3: Rechnungs-Export + Zahlungs-Import
- Woche 4: Fehlerbehandlung, Retry-Logik, UI

**UI-Anforderungen:**
- Settings-Page: "Bexio verbinden" Button (OAuth2-Flow startet)
- Sync-Status-Anzeige (letzte Synchronisation, Fehler)
- Kontakt-Detailseite: Bexio-Verlinkung ("In Bexio oeffnen")
- Rechnungs-Export: "An Bexio senden" Button
- Konflikte: Dialog mit "KMU Hub behalten" / "Bexio behalten"

**Empfehlung:**
1. Bexio Developer-Account anlegen: https://developer.bexio.com
2. Sandbox nutzen fuer erste Tests
3. Partner-Programm beantragen (erhoehte Limits + Marketplace-Listing)
4. Kontakt-Sync als erstes implementieren (hoechster Nutzen)

---

## 3. Skribble (E-Signatur)

**Typ:** Kommerziell (Freemium)
**Lizenz:** Proprietary, SaaS
**Hersteller:** Skribble AG, Zuerich, Schweiz
**Website:** https://www.skribble.com | API: https://doc.skribble.com

### Signatur-Standards: EES vs. FES vs. QES

| Standard | Beweiskraft | Identifikation | Anwendung | Preis/Signatur |
|----------|-------------|----------------|-----------|----------------|
| **EES** (Einfache E-Signatur) | Niedrig | E-Mail-Adresse genuegt | Interne Dokumente, Bestellungen, Offerten | ~0,50-1,50 CHF |
| **FES** (Fortgeschrittene E-Signatur) | Mittel | Mobilnummer-Verifikation (SMS) | Vertraege ohne Formvorschrift, NDAs, AGB | ~1,50-2,50 CHF |
| **QES** (Qualifizierte E-Signatur) | Hoechste (= handschriftlich) | Persoenliche Identifikation (Video-Ident oder ID-Check) | Arbeitsvertraege, Mietvertraege, notarielle Dokumente | ~2,50-4,00 CHF |

**Was brauchen KMUs?**
- **95% der Faelle:** EES oder FES genuegen
- **FES empfohlen** als Standard: Gute Beweiskraft, kein Video-Ident noetig
- **QES nur** bei: Arbeitsvertraegen, Mietvertraegen, Buergschaften, behindert. Kuendigungen
- KMU Hub sollte alle drei anbieten, Default = FES

### Pricing (Stand 2025)

| Plan | Kosten | Inkludiert | Fuer wen |
|------|--------|-----------|----------|
| **Free** | 0 CHF/Mo | 2 Signaturen/Mo (EES) | Testen |
| **Fair Flat** | 2,50 CHF/Signatur (Pay-as-you-go) | Keine Grundgebuehr | Wenig-Signierer |
| **Business** | Ab 85 CHF/Mo (1 Nutzer) | 600 EES + FES/Jahr, API-Zugang | KMUs |
| **Enterprise** | Individuell | Unbegrenzt, dedizierter Support, SLA | Grosse Firmen |

**API-Zugang:**
- Ab **Business-Plan** verfuegbar
- Enterprise-Plan fuer White-Label-Integration (Skribble-Branding entfernen)

### API-Zugang: Schritt fuer Schritt

```
1. Account erstellen: https://my.skribble.com/signup
2. Business-Plan buchen (fuer API-Zugang)
3. API-Key generieren: Unter "Einstellungen" -> "API"
4. API-Dokumentation: https://doc.skribble.com
5. Sandbox-Umgebung: https://demo.skribble.com (separate Instanz)
```

### REST API Flow

**Dokument zum Signieren senden:**

```
1. PDF hochladen:
   POST https://api.skribble.com/v2/signature-requests
   Headers: {
     "Authorization": "Bearer API_KEY",
     "Content-Type": "multipart/form-data"
   }
   Body: {
     "title": "Vertrag_2026_001",
     "message": "Bitte unterschreiben Sie den Vertrag",
     "content": [BASE64_PDF],
     "signers": [
       {
         "email": "kunde@example.com",
         "signer_identity": {
           "signer_identity_key": "EMAIL"  // oder "MOBILE" fuer FES
         },
         "visual_signature": {
           "position": { "page": 1, "x": 100, "y": 700 }
         }
       }
     ],
     "quality": "AES"  // SES = EES, AES = FES, QES = QES
   }

2. Skribble sendet E-Mail an Unterzeichner
3. Unterzeichner oeffnet Link, unterschreibt
4. Webhook an KMU Hub:
   POST https://api.kmuhub.com/webhooks/skribble
   {
     "event": "signature_request.signed",
     "signature_request_id": "sr_123",
     "document_url": "https://api.skribble.com/v2/documents/..."
   }

5. KMU Hub laedt signiertes PDF herunter und speichert es
```

### Sandbox

- **Ja, verfuegbar!**
- URL: `https://api-demo.skribble.com` (Demo-Instanz)
- Test-Signaturen sind kostenlos
- Webhooks funktionieren auch in der Sandbox
- Demo-Account bei Skribble anfragen

### Compliance

- **ZertES** (Schweizer Bundesgesetz ueber die elektronische Signatur): Vollstaendig konform
- **eIDAS** (EU-Verordnung): Vollstaendig konform
- **Trust Service Provider:** Swisscom (fuer QES in CH), Europaeische TSPs fuer EU
- **DSGVO:** Schweizer Firma, Daten in der Schweiz (Angemessenheitsbeschluss)
- **AVV/DPA:** Verfuegbar, automatisch bei Vertragsabschluss

**Go Libraries:**
- Kein offizielles Go-SDK
- REST-API direkt aufrufen
- Webhook-Handler: Standard `net/http`
- PDF-Manipulation (Signaturposition): `github.com/pdfcpu/pdfcpu`

**Integration-Aufwand:** 2-3 Wochen
- Woche 1: API-Anbindung + Signature-Request erstellen
- Woche 2: Webhook-Handler + Status-Tracking + signiertes PDF speichern
- Woche 3: UI (Signatur-Dialog, Status-Anzeige, Batch-Signaturen)

**UI-Anforderungen:**
- Vertraege-Modul: "Zur Unterschrift senden" Button
- Signatur-Level-Auswahl (EES/FES/QES) mit Erklaerung
- Status-Tracking: "Gesendet" -> "Geoeffnet" -> "Unterschrieben" -> "Abgeschlossen"
- Signiertes PDF: Vorschau + Download
- Settings: API-Key-Konfiguration + Webhook-URL

**Empfehlung:**
1. Skribble Business-Account erstellen
2. Sandbox nutzen fuer API-Tests
3. FES als Default implementieren (bester Kosten/Nutzen-Verhaeltnis)
4. Integration ins Vertraege-Modul als erstes

---

## 4. Brevo (ehem. Sendinblue)

**Typ:** Freemium
**Lizenz:** Proprietary, SaaS
**Hersteller:** Brevo (ehem. Sendinblue), Paris, Frankreich (EU!)
**Website:** https://www.brevo.com | API: https://developers.brevo.com

### API-Key: Zugang erhalten

```
1. Account erstellen: https://app.brevo.com/account/register
2. E-Mail verifizieren
3. API-Key generieren: "Settings" -> "API Keys" -> "Generate a new API key"
4. v3 API Key = ein einzelner String (beginnt mit "xkeysib-...")
5. Sofort nutzbar, kein Genehmigungsprozess
```

### Pricing (Stand 2025)

| Plan | Kosten/Mo | E-Mails/Mo | Features |
|------|-----------|-----------|----------|
| **Free** | 0 EUR | 300/Tag (~9.000/Mo) | API-Zugang, Brevo-Branding, 1 Sender |
| **Starter** | Ab 19 EUR | 20.000 | Kein taegliches Limit, Basis-Analytics |
| **Business** | Ab 49 EUR | 20.000 | Marketing Automation, A/B Testing, kein Brevo-Branding |
| **Enterprise** | Individuell | Individuell | Dedizierte IP, SSO, Priority Support |

**Transactional E-Mails (separat abgerechnet):**
- Free: 300/Tag inkludiert
- Ab 15 EUR/Mo fuer 20.000 transactionale E-Mails
- Transactional = Bestaetigungen, Passwort-Reset, Benachrichtigungen

### Transactional vs. Marketing API

| Typ | Zweck | API-Endpoint | DSGVO-Relevanz |
|-----|-------|-------------|----------------|
| **Transactional** | Rechnungsversand, Benachrichtigungen, Passwort-Reset | `POST /v3/smtp/email` | Berechtigtes Interesse (Art. 6(1)(f)) -- kein Opt-in noetig |
| **Marketing** | Newsletter, Werbe-Mails, Kampagnen | `POST /v3/emailCampaigns` + Kontaktlisten | Double-Opt-in PFLICHT (DSGVO + UWG) |

**KMU Hub braucht BEIDES:**
- Transactional: Rechnungen per Mail, Benachrichtigungen
- Marketing: Newsletter-Versand fuer KMU-Kunden an deren Endkunden

### SMTP vs. REST API

| Methode | Vorteile | Nachteile | Empfehlung |
|---------|----------|-----------|------------|
| **SMTP Relay** | Einfach, Standard-Protokoll, Go `net/smtp` | Weniger Features, kein Tracking, langsamer | Fuer Transactional (Rechnungsversand) |
| **REST API** | Templating, Tracking, Analytics, Webhooks | HTTP-Dependency, komplexer | Fuer Marketing (Newsletter) |

**Empfehlung:** REST API fuer alles. SMTP nur als Fallback.

### SMTP-Zugangsdaten (falls gewuenscht)

```
SMTP Server: smtp-relay.brevo.com
Port: 587 (STARTTLS)
Username: account-email@example.com
Password: SMTP-Key (nicht der API-Key!)
```

### REST API Beispiel (Transactional E-Mail)

```json
POST https://api.brevo.com/v3/smtp/email
Headers: {
  "api-key": "xkeysib-...",
  "Content-Type": "application/json"
}
Body: {
  "sender": { "name": "KMU Hub", "email": "noreply@kmuhub.com" },
  "to": [{ "email": "kunde@example.com", "name": "Max Mueller" }],
  "subject": "Ihre Rechnung #2026-001",
  "htmlContent": "<html>...</html>",
  "attachment": [{
    "url": "https://api.kmuhub.com/files/invoice-2026-001.pdf",
    "name": "Rechnung-2026-001.pdf"
  }]
}
```

### Double-Opt-In

Brevo unterstuetzt Double-Opt-In nativ:

```
1. Kontakt anlegen mit DOI-Flag:
   POST /v3/contacts/doubleOptinConfirmation
   {
     "email": "neuer-kontakt@example.com",
     "templateId": 123,  // DOI-Bestaetigungs-Template
     "redirectionUrl": "https://kmuhub.com/newsletter/bestaetigt",
     "includeListIds": [5]
   }
2. Brevo sendet automatisch Bestaetigungs-Mail
3. Kontakt klickt Link
4. Kontakt wird der Liste hinzugefuegt
5. Webhook an KMU Hub: "contact.updated" Event
```

### DSGVO-Konformitaet

- **Hosting:** EU (Frankreich + Deutschland, AWS eu-west/eu-central)
- **AVV/DPA:** Automatisch im Vertrag enthalten
- **Datenspeicherung:** EU-only (kein US-Transfer)
- **Sub-Processors:** AWS EU, OVH, Scaleway (alle EU)
- **Datenschutz-Zertifizierung:** ISO 27001
- **Recht auf Loeschung:** API-Endpoint `DELETE /v3/contacts/{identifier}`

**Go Libraries:**
- Offizielles SDK: `github.com/sendinblue/APIv3-go-library` (veraltet, noch "Sendinblue"-Name)
- **Empfehlung:** REST-API direkt mit `net/http` (das SDK ist nicht gut gepflegt)
- Webhook-Handler: Standard `net/http`

**Integration-Aufwand:** 2-3 Wochen
- Woche 1: API-Anbindung, Kontakt-Sync, Transactional E-Mails
- Woche 2: Newsletter-Kampagnen-UI, Template-Management, Listen
- Woche 3: DOI-Flow, Analytics-Dashboard, Webhooks

**UI-Anforderungen:**
- Settings: Brevo API-Key eingeben + verbinden
- Kontakte: "Newsletter"-Flag pro Kontakt (DOI-Status anzeigen)
- Newsletter-Modul: Kampagne erstellen, Empfaengerliste waehlen, Template waehlen, senden
- Analytics: Oeffnungsrate, Klickrate, Bounces, Abmeldungen
- Transactional: Automatisch bei Rechnungsversand (kein UI noetig)

**Empfehlung:**
1. Brevo Free-Account erstellen (sofort nutzbar)
2. Transactional API zuerst (Rechnungsversand)
3. Newsletter-Funktion als zweites (Marketing)
4. Starter-Plan buchen wenn Free-Limit nicht reicht

---

## 5. CleverReach

**Typ:** Freemium
**Lizenz:** Proprietary, SaaS
**Hersteller:** CleverReach GmbH & Co. KG, Rastede, Deutschland
**Website:** https://www.cleverreach.com | API: https://rest.cleverreach.com/explorer/v3

### API-Zugang

```
1. Account erstellen: https://www.cleverreach.com/de/registrierung/
2. API-Token generieren: "Mein Account" -> "Erweitert" -> "REST API"
3. OAuth2 verfuegbar fuer Drittanbieter-Integrationen
4. Client ID + Client Secret im Developer-Bereich
```

### OAuth2 Flow

```
Authorization URL: https://rest.cleverreach.com/oauth/authorize.php
Token URL: https://rest.cleverreach.com/oauth/token.php
Scopes: Keine spezifischen Scopes (Token gibt vollen Zugriff)
```

### Pricing (Stand 2025)

| Plan | Kosten/Mo | Empfaenger | Features |
|------|-----------|-----------|----------|
| **Lite (Free)** | 0 EUR | Bis 250 Empfaenger | CleverReach-Branding, Basis-Editor |
| **Essential** | Ab 15 EUR | Ab 500 Empfaenger | A/B Testing, kein Branding |
| **Flat** | Ab 69 EUR | Unbegrenzt | Automation, Segmentierung |
| **Enterprise** | Individuell | Individuell | Dedizierte IP, SLA |

**Preis-Staffelung (Essential):**
- 500 Empfaenger: 15 EUR/Mo
- 2.500 Empfaenger: 35 EUR/Mo
- 5.000 Empfaenger: 55 EUR/Mo
- 10.000 Empfaenger: 100 EUR/Mo

### DACH-spezifische Features

- **Doppeltes Opt-in:** Nativ eingebaut, rechtssicher
- **Impressum-Pflicht:** Automatisch im Footer
- **Abmeldelink:** Rechtskonforme Abmeldefunktion
- **Deutschsprachiger Support:** Telefon + E-Mail auf Deutsch
- **DSGVO-Compliance:** AVV direkt im Account abschliessbar
- **Hosting:** Ausschliesslich in Deutschland (eigene Server, kein AWS)
- **Datev-kompatible Rechnungen:** Deutsche Rechnungen mit MWSt

### Brevo vs. CleverReach: Wann welches?

| Kriterium | Brevo | CleverReach |
|-----------|-------|-------------|
| **Preis (kleines KMU)** | Guenstiger (Free: 300/Tag) | Teurer (Free: nur 250 Empfaenger) |
| **Transactional E-Mails** | JA (starke Transactional-API) | NEIN (nur Newsletter) |
| **DACH-Vertrauen** | Franzoesisch (EU, aber nicht DACH) | **Deutsch** (hohes Vertrauen) |
| **Hosting** | AWS EU (Frankfurt) | **Eigene Server in DE** |
| **API-Qualitaet** | Besser dokumentiert, moderner | Aelter, aber funktional |
| **Marketing Automation** | Ab Business (49 EUR) | Ab Flat (69 EUR) |
| **Empfehlung** | **Primaere Integration** (Brevo) | **Alternative anbieten** (fuer DACH-sensible Kunden) |

**Empfehlung fuer KMU Hub:**
- Brevo als Standard-Integration (guenstiger, Transactional + Marketing)
- CleverReach als Option fuer Kunden die "Deutsche Firma, Deutsche Server" wollen
- Beide implementieren, Kunde waehlt in Settings

**Go Libraries:**
- Kein offizielles Go-SDK
- REST API v3 direkt aufrufen
- OAuth2: `golang.org/x/oauth2`

**Integration-Aufwand:** 2-3 Wochen (aehnlich wie Brevo)

**UI-Anforderungen:** Identisch zu Brevo (Newsletter-Modul ist generisch, Backend tauscht nur den Provider)

**Empfehlung:**
1. CleverReach Account erstellen
2. Als zweiten Newsletter-Provider implementieren
3. Abstraktes Newsletter-Interface im Backend (`NewsletterProvider` Interface in Go)
4. Kunde waehlt in Settings: Brevo ODER CleverReach

---

## 6. FinAPI (Banking)

**Typ:** Kommerziell (B2B API)
**Lizenz:** Proprietary, SaaS
**Hersteller:** finAPI GmbH, Muenchen, Deutschland (Tochter von Schufa Holding)
**Website:** https://www.finapi.io | API: https://docs.finapi.io

### Partner werden: Schritt fuer Schritt

```
1. Kontaktformular: https://www.finapi.io/kontakt/
2. ODER direkt: partner@finapi.io
3. Use Case beschreiben: "CRM-Plattform mit Bankabgleich-Funktion"
4. FinAPI prueft und erstellt Sandbox-Zugang
5. Vertrag + NDA unterzeichnen
6. Produktiv-Zugang nach erfolgreicher Sandbox-Integration
```

**WICHTIG: Das dauert!** FinAPI ist ein regulierter Zahlungsdienstleister. Erwarten Sie 2-4 Wochen fuer den Onboarding-Prozess.

### PSD2-Lizenz

**Braucht KMU Hub eine eigene PSD2-Lizenz?**

**NEIN.** FinAPI ist ein lizenzierter AISP (Account Information Service Provider) unter PSD2. KMU Hub nutzt FinAPI als Dienstleister und braucht KEINE eigene BaFin-Lizenz. FinAPI uebernimmt die regulatorische Verantwortung.

**Voraussetzung:** KMU Hub muss im FinAPI-Vertrag als "Agent" / "Mandant" eingetragen sein.

### Kosten

| Kostenart | Betrag | Anmerkung |
|-----------|--------|-----------|
| **Setup-Gebuehr** | 0-5.000 EUR (verhandelbar) | Einmalig |
| **Monatliche Grundgebuehr** | Ab ~200-500 EUR/Mo | Abhaengig vom Volumen |
| **Pro Bank-Verbindung** | ~0,50-2,00 EUR/Mo | Pro verbundenem Bankkonto |
| **Pro Transaktion** | ~0,01-0,05 EUR | Pro abgerufener Transaktion |
| **Sandbox** | Kostenlos | Testumgebung |

**Achtung:** Preise sind stark verhandelbar und volumenabhaengig. Bei kleinem Volumen (< 100 Bankverbindungen) rechnet sich FinAPI erst ab ~500 EUR/Mo Grundgebuehr.

**Alternative Kostenmodelle:**
- Flatrate: ~500-2.000 EUR/Mo fuer X Bank-Verbindungen
- Pay-per-Use: Nur zahlen was genutzt wird
- Hybrid: Grundgebuehr + variable Kosten

### Sandbox

```
1. Nach Partner-Onboarding erhaelt man Sandbox-Credentials
2. Sandbox-URL: https://sandbox.finapi.io
3. Simulierte Bankverbindungen (keine echten Banken)
4. Test-User mit vordefinierten Konten und Transaktionen
5. Alle API-Endpoints verfuegbar
6. Kein echtes Geld, kein Risiko
```

### Welche Banken in DACH?

FinAPI deckt ab:
- **Deutschland:** ~3.000+ Banken (praktisch alle: Sparkassen, Volksbanken, Direktbanken, Neobanken)
- **Oesterreich:** ~300+ Banken (Erste Bank, Raiffeisen, BAWAG, etc.)
- **Schweiz:** **EINGESCHRAENKT** -- FinAPI ist primaer auf DE/AT fokussiert. Schweizer Banken werden sukzessive angebunden, aber die Abdeckung ist deutlich geringer (~50-100 Banken).

**Fuer Schweizer Kunden:** Alternativen pruefen (z.B. Contovista/Hypothekarbank Lenzburg, oder bLink von SIX).

### API-Flow (vereinfacht)

```
1. KMU-Nutzer klickt "Bankkonto verbinden"
2. KMU Hub oeffnet FinAPI Web Form (Iframe oder Redirect):
   GET https://webform.finapi.io/webforms?bankId=280
3. Nutzer gibt Online-Banking-Credentials ein (direkt bei FinAPI, NIE bei KMU Hub!)
4. FinAPI verbindet sich mit der Bank via PSD2 (XS2A)
5. FinAPI sendet Callback an KMU Hub: "Verbindung erfolgreich"
6. KMU Hub ruft Transaktionen ab:
   GET https://api.finapi.io/api/v2/transactions
   ?accountIds=123&minDate=2026-01-01
7. KMU Hub zeigt Transaktionen an und ermoeglicht Abgleich mit Rechnungen
```

**Go Libraries:**
- Kein offizielles Go-SDK
- REST API v2 direkt aufrufen
- OAuth2 (Client Credentials Flow fuer Backend-zu-Backend)
- FinAPI WebForm: React-Einbettung (Iframe)

**DSGVO:**
- Deutsche Firma, Server in Deutschland
- BaFin-reguliert (hoechste Sicherheitsstandards)
- AVV automatisch im Vertrag
- Bankdaten werden bei FinAPI gespeichert und verschluesselt
- KMU Hub speichert NUR die Transaktionsdaten (kein Zugriff auf Banking-Credentials!)

**Integration-Aufwand:** 3-4 Wochen
- Woche 1: FinAPI-Account + Sandbox + OAuth2
- Woche 2: WebForm-Einbettung (Bankverbindung herstellen)
- Woche 3: Transaktions-Abruf + Auto-Matching mit Rechnungen
- Woche 4: UI (Bankabgleich-Ansicht, offene Posten zuordnen)

**UI-Anforderungen:**
- Buchhaltung-Modul: "Bankkonto verbinden" Wizard
- Transaktionsliste mit Kategorie-Zuordnung
- Auto-Match: Eingehende Zahlung -> offene Rechnung (Betrags+Referenz-Matching)
- Manuelles Matching: Drag&Drop oder Dropdown
- Dashboard-Widget: Kontostand + offene Posten

**Empfehlung:**
1. FinAPI kontaktieren: partner@finapi.io
2. Sandbox-Zugang beantragen (2-4 Wochen Vorlauf!)
3. WebForm-Integration evaluieren (schnellster Weg)
4. Banking als Premium-Feature positionieren (nicht in Basis-Paket)

---

## 7. Nextcloud (WebDAV/CalDAV/CardDAV)

**Typ:** Open Source
**Lizenz:** AGPL v3
**Hersteller:** Nextcloud GmbH, Stuttgart, Deutschland
**Website:** https://nextcloud.com
**Kosten fuer KMU Hub:** 0 EUR (keine Lizenz noetig)

### Keine Lizenzkosten

Nextcloud ist Open Source. KMU Hub integriert sich als CLIENT -- wir betreiben keinen Nextcloud-Server, sondern verbinden uns mit dem Nextcloud des Kunden. Daher:
- Keine Lizenzkosten
- Kein Nextcloud-Code in KMU Hub (nur Protokoll-Nutzung)
- Kein AGPL-Copyleft-Problem

### WebDAV-Integration (Dateien)

**Protokoll:** WebDAV (RFC 4918) -- standardisiertes HTTP-basiertes Protokoll fuer Dateizugriff.

**Go Libraries:**

| Library | Link | Status |
|---------|------|--------|
| `studio-b12/gowebdav` | github.com/studio-b12/gowebdav | Aktiv gepflegt, gute API |
| `emersion/go-webdav` | github.com/emersion/go-webdav | Von emersion (go-imap Autor), moderner |

**Empfehlung:** `studio-b12/gowebdav` (etablierter, mehr Stars, bessere Doku)

**Beispiel-Code (Go):**

```go
import "github.com/studio-b12/gowebdav"

func connectNextcloud(url, user, pass string) {
    client := gowebdav.NewClient(url, user, pass)

    // Dateien auflisten
    files, _ := client.ReadDir("/Documents/")
    for _, f := range files {
        fmt.Printf("%s (%d bytes)\n", f.Name(), f.Size())
    }

    // Datei herunterladen
    data, _ := client.Read("/Documents/Vertrag.docx")

    // Datei hochladen
    client.Write("/Documents/Rechnung.pdf", pdfBytes, 0644)
}
```

**Verbindung herstellen (Kunden-Perspektive):**

```
1. KMU-Kunde oeffnet Settings -> "Nextcloud verbinden"
2. Eingabe: Nextcloud-URL (z.B. https://cloud.meinefirma.ch)
3. Eingabe: Benutzername + App-Passwort
   (App-Passwort generieren: Nextcloud -> Einstellungen -> Sicherheit -> "Neues App-Passwort")
4. KMU Hub testet Verbindung (WebDAV PROPFIND Request)
5. Ordner-Zuordnung: Welcher Nextcloud-Ordner = KMU Hub Dokumente?
```

### CalDAV-Integration (Kalender)

**Protokoll:** CalDAV (RFC 4791) -- standardisiertes Protokoll fuer Kalender-Synchronisation.

**Go Libraries:**

| Library | Link | Status |
|---------|------|--------|
| `emersion/go-webdav` | github.com/emersion/go-webdav | Beinhaltet CalDAV + CardDAV Client |

**Sync-Strategie:**

```
1. Initial-Sync: Alle Events vom CalDAV-Server holen
   (REPORT Request mit calendar-query)
2. Laufend: ctag/etag-basierter Delta-Sync
   - CalDAV-Server hat einen "ctag" pro Kalender
   - Wenn ctag sich aendert: geaenderte Events per etag identifizieren
   - Nur geaenderte Events herunterladen
3. KMU Hub -> Nextcloud: Neue Termine per PUT erstellen
4. Nextcloud -> KMU Hub: Polling alle 5 Minuten (oder WebSocket wenn moeglich)
5. iCalendar-Format (RFC 5545) als Datenformat
```

### CardDAV-Integration (Kontakte)

**Protokoll:** CardDAV (RFC 6352) -- standardisiertes Protokoll fuer Kontakt-Synchronisation.

**Sync-Strategie (analog zu CalDAV):**

```
1. Initial-Sync: Alle vCards herunterladen
2. Delta-Sync via sync-token oder ctag
3. Mapping: vCard-Felder -> KMU Hub Kontakt-Felder
   - FN -> Anzeigename
   - N -> Vorname, Nachname
   - EMAIL -> E-Mail
   - TEL -> Telefon
   - ORG -> Firma
   - ADR -> Adresse
4. Konflikte: "Last Modified" Timestamp vergleichen
```

**DSGVO:** Nextcloud = EU-Firma (Stuttgart). Daten liegen beim Kunden (Self-Hosted) oder beim Nextcloud-Hoster des Kunden. KMU Hub uebertraegt nur Metadaten. Kein DSGVO-Problem.

**Integration-Aufwand:** 2-3 Wochen
- Woche 1: WebDAV-Datei-Browse + Upload/Download
- Woche 2: CalDAV-Sync (Kalender)
- Woche 3: CardDAV-Sync (Kontakte) + Fehlerbehandlung

**UI-Anforderungen:**
- Settings: "Nextcloud verbinden" Wizard (URL + Credentials)
- Datei-Browser: Nextcloud-Ordner neben lokalen Dateien anzeigen
- Kalender: Nextcloud-Kalender als Layer einblenden (andere Farbe)
- Kontakte: Sync-Richtung waehlen (bidirektional oder nur-lesen)
- Sync-Status: "Letzte Sync: vor 5 Min." + manueller "Jetzt synchronisieren" Button

**Empfehlung:**
1. `studio-b12/gowebdav` fuer WebDAV evaluieren
2. `emersion/go-webdav` fuer CalDAV/CardDAV evaluieren
3. WebDAV-Dateizugriff als erstes (hoechster Nutzen)
4. CalDAV/CardDAV als zweiter Schritt

---

## 8. DATEV-Export

**Typ:** Proprietaeres Export-Format (aber frei dokumentiert)
**Lizenz:** Keine Lizenz noetig (Export-Format, nicht API)
**Hersteller:** DATEV eG, Nuernberg, Deutschland
**Website:** https://www.datev.de | Spec: https://developer.datev.de

### DATEV-Format: Dokumentation

**Wo ist die Spec?**
- Offizielle Doku: https://developer.datev.de/portal/de/dtvf (DATEV-Format)
- Dokument: "DATEV-Format-Beschreibung" (PDF, frei downloadbar nach Registrierung)
- Alternative: Die Spec wird auch von vielen Open-Source-Projekten dokumentiert (z.B. auf GitHub suchen nach "DATEV CSV Format")

**Registrierung bei developer.datev.de:**
```
1. https://developer.datev.de registrieren (kostenlos)
2. E-Mail bestaetigen
3. "DATEV-Format" Dokumentation herunterladen
4. KEIN Partner-Vertrag noetig fuer Export
```

### Braucht man eine DATEV-Partnerschaft?

**NEIN -- fuer den EXPORT nicht!**

| Szenario | Partnerschaft noetig? |
|----------|----------------------|
| CSV-Export im DATEV-Format | NEIN -- frei nutzbar |
| DATEV Unternehmen Online API | JA -- Partner-Vertrag noetig |
| DATEV SmartTransfer API | JA -- Partner-Vertrag noetig |
| "DATEV-Schnittstelle" Label im Marketing | Grauzone -- "DATEV-Format kompatibel" ist sicherer |

**Fuer KMU Hub Phase 1:** Nur CSV-Export. Keine Partnerschaft noetig.
**Fuer KMU Hub Phase 2+:** DATEV Unternehmen Online API anbinden (dann Partnerschaft beantragen).

### Export-Format Details

**Buchungsstapel (Hauptformat):**

```
Encoding:      Windows-1252 (CP1252) -- NICHT UTF-8!
Trennzeichen:  Semikolon (;)
Dezimalzeichen: Komma (,)
Datumsformat:  TTMM (4-stellig, z.B. 1501 fuer 15. Januar)
Dateiendung:   .csv
Zeilenende:    CR+LF (\r\n)
```

**Dateistruktur (3 Zeilen Header + Daten):**

```csv
"EXTF";700;21;"Buchungsstapel";12;20260101;20260131;;"KMU Hub";"";1;20260101;20260131;0;0;0;0;0;;""
Umsatz (ohne Soll/Haben-Kz);Soll/Haben-Kennzeichen;WKZ Umsatz;Kurs;Basis-Umsatz;WKZ Basis-Umsatz;Konto;Gegenkonto (ohne BU-Schluessel);BU-Schluessel;Belegdatum;Belegfeld 1;Belegfeld 2;Skonto;Buchungstext;...
100,00;S;EUR;;100,00;EUR;1200;8400;;1501;RE-2026-001;;"";Webdesign-Dienstleistung;...
```

**Header-Zeile 1 (Formatheader):**

| Feld | Wert | Bedeutung |
|------|------|-----------|
| Kennung | "EXTF" | Externes Format (nicht DATEV-intern) |
| Versionsnummer | 700 | DATEV-Format Version 7.0 |
| Datenkategorie | 21 | Buchungsstapel |
| Formatname | "Buchungsstapel" | Menschenlesbarer Name |
| Formatversion | 12 | Version der Feldliste |
| Erzeugt am | 20260101 | Datum der Erzeugung (YYYYMMDD) |
| Datum von | 20260101 | Buchungszeitraum Start |
| Datum bis | 20260131 | Buchungszeitraum Ende |
| Bezeichnung | "KMU Hub" | Quellsystem |

**Wichtigste Buchungsfelder:**

| # | Feldname | Typ | Beispiel | Pflicht |
|---|---------|------|---------|---------|
| 1 | Umsatz | Decimal(10,2) | 100,00 | Ja |
| 2 | Soll/Haben-Kz | S oder H | S | Ja |
| 3 | WKZ Umsatz | String(3) | EUR | Ja |
| 7 | Konto | Integer | 1200 (Debitor) | Ja |
| 8 | Gegenkonto | Integer | 8400 (Erloese) | Ja |
| 10 | Belegdatum | TTMM | 1501 | Ja |
| 11 | Belegfeld 1 | String(36) | RE-2026-001 | Ja |
| 14 | Buchungstext | String(60) | Webdesign | Empfohlen |

**Kontenrahmen:**
- SKR03: Standard fuer kleine/mittlere Unternehmen (DE)
- SKR04: Alternative (Bilanzorientiert)
- KMU Hub muss den Kontenrahmen NICHT kennen -- der Steuerberater ordnet die Konten zu
- KMU Hub exportiert nur die Buchungen mit Kontonummern die der Kunde konfiguriert hat

### Wie testen ohne DATEV-Zugang?

```
1. CSV-Datei im DATEV-Format erzeugen
2. DATEV Belegtransfer (kostenlose Test-Software, nach Registrierung downloadbar):
   https://www.datev.de/web/de/mydatev/software/datev-belegtransfer/
3. ODER: Steuerberater fragen ob er die Datei probeweise importieren kann
4. ODER: Open-Source-Validator nutzen (z.B. datev-format-validator auf GitHub)
5. Wichtig: Windows-1252 Encoding testen! (haeufigster Fehler)
```

**Go Libraries:**
- Kein spezielles DATEV-Go-Package (Format ist zu einfach dafuer)
- Standard `encoding/csv` (mit Semikolon-Separator)
- Encoding: `golang.org/x/text/encoding/charmap` (fuer Windows-1252)
- Empfehlung: Eigenen DATEV-Exporter schreiben (~200-300 LOC)

**Go-Beispiel:**

```go
import (
    "encoding/csv"
    "golang.org/x/text/encoding/charmap"
    "golang.org/x/text/transform"
)

func exportDATEV(bookings []Booking, w io.Writer) error {
    // Windows-1252 Encoder
    encoder := charmap.Windows1252.NewEncoder()
    writer := transform.NewWriter(w, encoder)

    csvWriter := csv.NewWriter(writer)
    csvWriter.Comma = ';'
    csvWriter.UseCRLF = true

    // Header Zeile 1
    csvWriter.Write([]string{
        `"EXTF"`, "700", "21", `"Buchungsstapel"`, "12",
        "20260101", "20260131", "", `"KMU Hub"`, `""`,
        // ... weitere Header-Felder
    })

    // Header Zeile 2 (Feldnamen)
    csvWriter.Write([]string{
        "Umsatz (ohne Soll/Haben-Kz)", "Soll/Haben-Kennzeichen",
        "WKZ Umsatz", "Kurs", "Basis-Umsatz", // ...
    })

    // Buchungszeilen
    for _, b := range bookings {
        csvWriter.Write([]string{
            formatDecimal(b.Amount), b.DebitCredit,
            "EUR", "", formatDecimal(b.Amount), // ...
        })
    }

    csvWriter.Flush()
    return csvWriter.Error()
}
```

**DSGVO:** Kein Datenfluss an DATEV. Export = lokale Datei die der Kunde an seinen Steuerberater sendet.

**Integration-Aufwand:** 1-2 Wochen
- Woche 1: DATEV-Exporter (Go) + Zeiterfassungs-Export + Rechnungs-Export
- Woche 2: UI (Export-Dialog, Zeitraum-Auswahl, Konten-Zuordnung, Vorschau)

**UI-Anforderungen:**
- Buchhaltung-Modul: "DATEV-Export" Button
- Export-Dialog: Zeitraum waehlen (Monat), Kontenrahmen (SKR03/SKR04)
- Konten-Zuordnung: Einfache Tabelle (KMU Hub Kategorie -> DATEV Konto)
- Vorschau: Erste 10 Zeilen anzeigen vor Download
- Download: CSV-Datei (Windows-1252, .csv)
- Zeiterfassung: "DATEV Lohnexport" (separates Format: Lohn-Buchungsstapel)

**Empfehlung:**
1. developer.datev.de registrieren und Format-Doku herunterladen
2. Buchungsstapel-Export implementieren (einfachstes Format)
3. SKR03 als Default-Kontenrahmen
4. Mit einem Steuerberater testen (realer Import in DATEV)

---

## 9. Swiss QR-Code (QR-Rechnung)

**Typ:** Offener Standard (kostenlos)
**Lizenz:** Frei nutzbar, keine Lizenz
**Herausgeber:** SIX Group / Swiss Payment Standards
**Website:** https://www.paymentstandards.ch
**Spec:** "Swiss Implementation Guidelines QR-Rechnung" (frei downloadbar)

### Spezifikation: Wo dokumentiert?

```
1. Hauptdokument: https://www.paymentstandards.ch/de/shared/communication-grid.html
   -> "Implementation Guidelines QR-Rechnung" (PDF, ~80 Seiten)
2. Technische Spec: "Swiss QR Code" (Style Guide + Datenformat)
3. Validierungstool: https://www.paymentstandards.ch/de/shared/communication-grid.html
   -> "QR-Rechnung Validator" (Online-Tool)
4. Testdaten: Im Spec-Dokument enthalten (Anhang)
```

### QR-Code Datenformat

Der Swiss QR Code enthaelt strukturierte Zahlungsinformationen als Plain Text:

```
SPC                          <- Header (Swiss Payments Code)
0200                         <- Version
1                            <- Coding (UTF-8)
CH4431999123000889012         <- IBAN (Empfaenger)
S                            <- Adresstyp (S=strukturiert, K=kombiniert)
KMU Hub GmbH                <- Name Empfaenger
Musterstrasse 1             <- Strasse
                             <- (leer)
8001                         <- PLZ
Zuerich                     <- Ort
CH                           <- Land
                             <- (7 leere Zeilen fuer Endgueltiger Zahlungspflichtiger)
1949.75                      <- Betrag
CHF                          <- Waehrung
S                            <- Adresstyp Zahler
Max Mueller                  <- Name Zahler
Beispielgasse 3             <- Strasse Zahler
                             <- (leer)
3000                         <- PLZ Zahler
Bern                         <- Ort Zahler
CH                           <- Land Zahler
QRR                          <- Referenztyp (QRR, SCOR, NON)
210000000003139471430009017  <- QR-Referenz (26+1 Stellen)
Rechnung 2026-001           <- Zusaetzliche Info
EPD                          <- Trailer (End Payment Data)
```

### Go Libraries

| Library | Link | Status | Empfehlung |
|---------|------|--------|------------|
| `krepost/swissqr` | github.com/krepost/swissqr | Stabil | Basis-QR-Generierung |
| `sigurn/qrbill` | github.com/sigurn/qrbill | Aktiv | Schweizer QR-Rechnung spezifisch |
| `skip2/go-qrcode` | github.com/skip2/go-qrcode | Sehr populaer | QR-Code-Generierung (generisch) |

**Empfehlung:** `skip2/go-qrcode` fuer QR-Code-Generierung + eigene Datenstruktur fuer den Swiss-QR-Payload (~100-200 LOC).

**Beispiel-Code:**

```go
type SwissQRData struct {
    IBAN         string  // CH-IBAN (21 Stellen) oder QR-IBAN (21 Stellen)
    CreditorName string
    CreditorAddr Address
    Amount       float64 // Optional (kann leer sein)
    Currency     string  // CHF oder EUR
    DebtorName   string  // Optional
    DebtorAddr   Address // Optional
    RefType      string  // QRR, SCOR, NON
    Reference    string  // QR-Referenz oder Creditor Reference
    Message      string  // Max. 140 Zeichen
}

func (q *SwissQRData) Payload() string {
    var sb strings.Builder
    sb.WriteString("SPC\r\n")
    sb.WriteString("0200\r\n")
    sb.WriteString("1\r\n")
    sb.WriteString(q.IBAN + "\r\n")
    sb.WriteString("S\r\n")
    sb.WriteString(q.CreditorName + "\r\n")
    // ... restliche Felder
    sb.WriteString("EPD\r\n")
    return sb.String()
}

func generateQRImage(data SwissQRData) ([]byte, error) {
    payload := data.Payload()
    qr, err := qrcode.New(payload, qrcode.Medium)
    if err != nil { return nil, err }
    qr.DisableBorder = false
    return qr.PNG(256) // 256x256 Pixel
}
```

### Design-Vorgaben (WICHTIG!)

Die Swiss QR-Rechnung hat exakte Gestaltungsvorschriften:

```
Gesamter Zahlteil:     210 mm breit x 105 mm hoch
Position:              Am unteren Rand der A4-Rechnung
Perforierung:          Horizontale Linie oben + vertikale Linie links

Empfangsschein (links):  62 mm breit x 105 mm hoch
Zahlteil (rechts):       148 mm breit x 105 mm hoch

QR-Code:               46 mm x 46 mm (inkl. Schweizer Kreuz in der Mitte!)
Schweizer Kreuz:        7 mm x 7 mm (zentriert im QR-Code)
Schriftgroesse:         Titel 11pt, Werte 10pt, Angaben 8pt
Schriftart:             Liberation Sans, Arial, Frutiger, Helvetica

Abstande:              5 mm Seitenrand, 5 mm zwischen Elementen
```

**Schweizer Kreuz im QR-Code:**
- Das Schweizer Kreuz MUSS im Zentrum des QR-Codes stehen
- Groesse: 7mm x 7mm
- Weisses Kreuz auf schwarzem Hintergrund (invertiert zum normalen Kreuz!)
- Error-Correction-Level des QR-Codes muss "M" sein (damit das Kreuz den Code nicht zerstoert)

### Testdaten / Validierung

```
1. Validierungs-Tool von SIX: https://validation.iso-payments.ch
2. Testdaten im Spec-Dokument (Anhang B):
   - Test-IBAN: CH4431999123000889012
   - Test-QR-Referenz: 210000000003139471430009017
3. QR-Rechnung Referenzpruefsumme: Modulo 10 rekursiv (Algorithmus im Spec)
4. Swiss QR Code Reader App (Mobile) zum Testen der generierten QR-Codes
```

**DSGVO:** Kein externer Dienst. QR-Code wird lokal generiert. Keine Datenuebertragung.

**Integration-Aufwand:** 1-2 Wochen
- Woche 1: QR-Datenstruktur + QR-Code-Generierung + Schweizer Kreuz Overlay
- Woche 2: PDF-Layout (Zahlteil am Seitenende) + Validierung + Tests

**UI-Anforderungen:**
- Rechnungs-PDF: QR-Zahlteil automatisch am Seitenende
- Settings: QR-IBAN oder normale IBAN konfigurieren
- Vorschau: QR-Rechnung vor Versand anzeigen
- Validierung: Pruefziffern-Check bei Eingabe der Referenznummer

**Empfehlung:**
1. Spec herunterladen: https://www.paymentstandards.ch
2. `skip2/go-qrcode` als QR-Library evaluieren
3. Schweizer-Kreuz-Overlay implementieren (Bildbearbeitung in Go: `image/draw`)
4. Mit SIX Validator testen
5. Physisch drucken und mit Banking-App scannen (Endtest!)

---

## 10. ZUGFeRD / XRechnung

**Typ:** Offener Standard (EU-Norm EN 16931)
**Lizenz:** Frei nutzbar
**Herausgeber:** FeRD (Forum elektronische Rechnung Deutschland) / EU
**Spec:** EN 16931 + CII (Cross-Industry Invoice) oder UBL (Universal Business Language)
**Website:** https://www.ferd-net.de (ZUGFeRD) | https://xeinkauf.de/xrechnung/ (XRechnung)

### ZUGFeRD vs. XRechnung

| Merkmal | ZUGFeRD | XRechnung |
|---------|---------|-----------|
| **Format** | PDF/A-3 + eingebettete XML | Reines XML (kein PDF) |
| **Basis** | EN 16931 (CII-Syntax) | EN 16931 (UBL- oder CII-Syntax) |
| **Sichtbarkeit** | Hybrid: PDF fuer Menschen + XML fuer Maschinen | Nur maschinenlesbar |
| **Empfaenger** | B2B allgemein | B2G (Behoerden) in DE |
| **Pflicht (DE)** | Empfang ab 2025, Versand ab 2027/2028 (B2B) | Pflicht fuer oeffentl. Auftraege (seit 2020) |

**Empfehlung fuer KMU Hub:** ZUGFeRD implementieren (deckt B2B ab + XRechnung-Kompatibilitaet via EN 16931).

### Profile

ZUGFeRD definiert verschiedene Detailstufen:

| Profil | Inhalt | Anwendung | Empfehlung |
|--------|--------|-----------|------------|
| **Minimum** | Nur Rechnungseckdaten | Archivierung | Zu wenig fuer Verarbeitung |
| **Basic WL** | + Positionsdetails (ohne Berechnung) | Einfache Rechnungen | Nein |
| **Basic** | + Berechnung auf Positionsebene | Standard-Rechnungen | **JA -- fuer einfache KMUs** |
| **EN 16931 (Comfort)** | Volle EN 16931 Compliance | B2B Standard | **JA -- empfohlen als Default** |
| **Extended** | + branchenspezifische Felder | Spezialfaelle | Nein (ueberfluesssig fuer 95% der KMUs) |

**Empfehlung:** EN 16931 (Comfort) als Default. Deckt alle gesetzlichen Anforderungen ab.

### Go Libraries

| Library | Link | Funktion | Status |
|---------|------|----------|--------|
| `invopop/gobl` | github.com/invopop/gobl | E-Rechnungs-Framework (multi-country) | Aktiv, gut gepflegt |
| `invopop/gobl.zugferd` | github.com/invopop/gobl.zugferd | ZUGFeRD-spezifische Konvertierung | Addon fuer gobl |
| `invopop/gobl.xinvoice` | github.com/invopop/gobl.xinvoice | XRechnung (UBL/CII) | Addon fuer gobl |

**`invopop/gobl` ist die empfohlene Loesung:**
- Modernes Go-Package fuer strukturierte Rechnungsdaten
- Unterstuetzt ZUGFeRD, XRechnung, Factur-X (FR), FatturaPA (IT)
- Validierung gegen EN 16931 eingebaut
- DACH-spezifische Steuerregeln vorhanden
- Aktiv von Invopop (spanisches Startup) gepflegt

**Beispiel-Flow:**

```go
import (
    "github.com/invopop/gobl"
    "github.com/invopop/gobl/bill"
    "github.com/invopop/gobl/org"
    "github.com/invopop/gobl/tax"
    "github.com/invopop/gobl.zugferd"
)

// 1. Rechnung erstellen
inv := &bill.Invoice{
    Type:     bill.InvoiceTypeStandard,
    Code:     "RE-2026-001",
    IssueDate: cal.NewDate(2026, 1, 15),
    Currency:  currency.EUR,
    Supplier: &org.Party{
        Name: "KMU Hub Kunde GmbH",
        TaxID: &tax.Identity{
            Country: "DE",
            Code:    "DE123456789",
        },
    },
    Customer: &org.Party{
        Name: "Empfaenger AG",
    },
    Lines: []*bill.Line{
        {
            Quantity: num.MakeAmount(10, 0),
            Item: &org.Item{
                Name:  "Webdesign-Dienstleistung",
                Price: num.MakeAmount(15000, 2), // 150.00
            },
            Taxes: tax.Set{
                {Category: tax.CategoryVAT, Rate: "standard"}, // 19%
            },
        },
    },
}

// 2. GOBL-Envelope erstellen + validieren
env, _ := gobl.Envelop(inv)

// 3. ZUGFeRD-XML generieren
xmlData, _ := zugferd.Generate(env, zugferd.ProfileEN16931)

// 4. XML in PDF/A-3 einbetten
// -> Hier eine PDF-Library nutzen (z.B. pdfcpu oder unidoc)
```

### PDF/A-3 Einbettung

ZUGFeRD erfordert PDF/A-3 (nicht normales PDF!). Das XML wird als Attachment im PDF eingebettet.

**Go Libraries fuer PDF/A-3:**

| Library | Link | PDF/A-3 Support |
|---------|------|-----------------|
| `unidoc/unipdf` | github.com/unidoc/unipdf | JA (kommerziell, ~500 EUR/Jahr) |
| `pdfcpu/pdfcpu` | github.com/pdfcpu/pdfcpu | PDF/A Validierung, aber keine Erstellung |
| `jung-kurt/gofpdf` | github.com/jung-kurt/gofpdf | NEIN (nur normales PDF) |

**Empfehlung:** `unidoc/unipdf` (kommerziell) oder PDF extern generieren (wkhtmltopdf + Nachbearbeitung zu PDF/A-3).

### Validierung

```
1. Kostenhaus Validator: https://www.portinvoice.com/validator (online, kostenlos)
2. FeRD ZUGFeRD Checker: https://www.ferd-net.de/zugferd/zugferd-checker
3. KoSIT Validator (XRechnung): https://github.com/itplr-kosit/validator
   -> Open-Source Java-Tool, kann in CI/CD integriert werden
4. gobl hat eingebaute Validierung (Schema + Business Rules)
```

**DSGVO:** Kein externer Dienst. XML wird lokal generiert. Keine Datenuebertragung.

**Integration-Aufwand:** 2-3 Wochen
- Woche 1: GOBL-Integration + ZUGFeRD-XML-Generierung + Validierung
- Woche 2: PDF/A-3-Einbettung + XRechnung-Support
- Woche 3: UI (Format-Auswahl, Vorschau, Validierungs-Ergebnis)

**UI-Anforderungen:**
- Rechnungs-Erstellung: "E-Rechnung Format" Dropdown (ZUGFeRD / XRechnung / keins)
- Profil-Auswahl: Basic / EN 16931 (Default) / Extended
- Validierungs-Anzeige: Gruener Haken oder Fehlerliste
- Download: PDF/A-3 mit eingebettetem XML
- B2G-Flag: "Oeffentlicher Auftraggeber" -> automatisch XRechnung

**Empfehlung:**
1. `invopop/gobl` + `invopop/gobl.zugferd` evaluieren
2. EN 16931 (Comfort) Profil als Default
3. ZUGFeRD zuerst, XRechnung als Ergaenzung
4. `unidoc/unipdf` fuer PDF/A-3 evaluieren (Lizenzkosten einplanen)
5. Mit FeRD Checker validieren

---

## 11. LiveKit (Video/Audio)

**Typ:** Open Source (Server) + Kommerziell (Cloud)
**Lizenz:** Apache 2.0 (Server + SDKs)
**Hersteller:** LiveKit Inc., San Francisco, USA (aber Self-Hostable = EU-konform)
**Website:** https://livekit.io | Docs: https://docs.livekit.io

### Self-Hosted: Docker Setup

**System-Anforderungen:**
- **RAM:** Minimum 2 GB, empfohlen 4 GB+ (pro 50 gleichzeitige Teilnehmer)
- **CPU:** Minimum 2 Cores, empfohlen 4+ Cores
- **Bandwidth:** ~3 Mbps pro Teilnehmer (Video), ~100 Kbps (Audio-only)
- **Ports:** TCP 7880 (HTTP/WS), UDP 50000-60000 (WebRTC media), TCP 7881 (TURN/TLS)
- **OS:** Linux (empfohlen), Docker

**docker-compose.yml:**

```yaml
services:
  livekit:
    image: livekit/livekit-server:latest
    container_name: kmuhub-livekit
    restart: always
    ports:
      - "7880:7880"   # HTTP + WebSocket
      - "7881:7881"   # TURN/TLS
      - "50000-60000:50000-60000/udp"  # WebRTC Media
    environment:
      - LIVEKIT_KEYS=APIKey: APISecret  # Generieren!
    volumes:
      - ./livekit-config.yaml:/etc/livekit.yaml
    command: ["--config", "/etc/livekit.yaml"]
```

**livekit-config.yaml:**

```yaml
port: 7880
rtc:
  port_range_start: 50000
  port_range_end: 60000
  use_external_ip: true
  # TURN-Server (eingebaut!)
  turn:
    enabled: true
    domain: livekit.meinefirma.com
    tls_port: 7881
    udp_port: 443  # Fallback fuer restriktive Firewalls
keys:
  # API Key : API Secret (generieren mit: livekit-cli create-keys)
  APIxxxxxxxxxxxxxxx: geheim123456789
redis:
  address: redis:6379  # Optional, fuer Multi-Node
logging:
  level: info
```

**Schritt-fuer-Schritt:**

```
1. API Keys generieren:
   docker run --rm livekit/livekit-server generate-keys
   -> Gibt APIKey + APISecret aus

2. Config-Datei erstellen (siehe oben)

3. Container starten:
   docker-compose up -d

4. Health-Check:
   curl http://localhost:7880

5. Test-Room erstellen (mit livekit-cli):
   livekit-cli create-token --api-key APIxxx --api-secret geheim --room test-room --identity user1 --valid-for 24h

6. Token im Browser testen:
   https://meet.livekit.io/?livekit-url=ws://localhost:7880&token=TOKEN
```

### TURN/STUN Server

**Braucht man externe TURN/STUN Server?**

**NEIN -- LiveKit hat einen eingebauten TURN-Server!**

- STUN: Wird automatisch von LiveKit bereitgestellt
- TURN: Eingebaut in LiveKit Server (ueber Port 7881 / TCP 443)
- Wichtig fuer Firmen-Netzwerke: Viele Firewalls blockieren UDP -- TURN ueber TCP 443 ist der Fallback
- Kein Coturn oder andere externe TURN-Server noetig

### Cloud Pricing (LiveKit Cloud)

| Plan | Kosten | Inkludiert |
|------|--------|-----------|
| **Free** | 0 USD | 50 Teilnehmer-Minuten/Mo |
| **Starter** | Pay-as-you-go | $0.004/Teilnehmer-Minute (Video), $0.0004 (Audio) |
| **Growth** | Individuell | Volumenrabatte |
| **Enterprise** | Individuell | SLA, dedizierte Infrastruktur |

**Kostenbeispiel (Cloud):**
- 1 Meeting mit 5 Teilnehmern, 60 Minuten = 300 Teilnehmer-Minuten
- Kosten: 300 x $0.004 = $1.20
- 100 solcher Meetings/Monat = $120/Mo

**Empfehlung fuer KMU Hub:** Self-Hosted (EU-Konformitaet + keine laufenden Kosten).

### Go SDK

**Offizielles Go SDK:** `github.com/livekit/server-sdk-go`

```go
import (
    lksdk "github.com/livekit/server-sdk-go"
    "github.com/livekit/protocol/auth"
    "github.com/livekit/protocol/livekit"
)

// Room erstellen
roomClient := lksdk.NewRoomServiceClient(
    "ws://livekit.meinefirma.com:7880",
    "APIKey",
    "APISecret",
)

room, _ := roomClient.CreateRoom(ctx, &livekit.CreateRoomRequest{
    Name: "meeting-2026-01-15",
    EmptyTimeout: 300,  // Raum loeschen nach 5 Min ohne Teilnehmer
    MaxParticipants: 20,
})

// Token fuer Teilnehmer generieren
at := auth.NewAccessToken("APIKey", "APISecret")
grant := &auth.VideoGrant{
    RoomJoin: true,
    Room:     "meeting-2026-01-15",
}
at.AddGrant(grant).
    SetIdentity("max.mueller@firma.com").
    SetName("Max Mueller").
    SetValidFor(24 * time.Hour)

token, _ := at.ToJWT()
// -> Token an Frontend senden
```

### React SDK (Frontend)

**Offizielles React SDK:** `@livekit/components-react`

```bash
npm install @livekit/components-react livekit-client
```

```tsx
import { LiveKitRoom, VideoConference } from '@livekit/components-react';
import '@livekit/components-styles';

function MeetingRoom({ token, serverUrl }) {
  return (
    <LiveKitRoom
      token={token}
      serverUrl={serverUrl}
      connect={true}
    >
      <VideoConference />
    </LiveKitRoom>
  );
}
```

### Bandwidth-Anforderungen

| Modus | Pro Teilnehmer (Upload) | Pro Teilnehmer (Download) |
|-------|------------------------|--------------------------|
| Audio-only | ~50-100 Kbps | ~50-100 Kbps pro Stream |
| Video (720p) | ~1.5-2.5 Mbps | ~1.5-2.5 Mbps pro Stream |
| Video (1080p) | ~3-5 Mbps | ~3-5 Mbps pro Stream |
| Screenshare | ~1-3 Mbps | ~1-3 Mbps |

**Server-Bandwidth (Self-Hosted):**
- Meeting mit 5 Teilnehmern, Video 720p: ~50 Mbps Server-Bandwidth
- Meeting mit 10 Teilnehmern, Video 720p: ~200 Mbps Server-Bandwidth
- LiveKit nutzt Simulcast + SFU-Architektur (effizient!)
- Empfehlung: Dedicated Server mit 1 Gbps (Hetzner: ~40 EUR/Mo)

**DSGVO:** Self-Hosted = volle Kontrolle. Kein US-Datenfluss. LiveKit Cloud (US-Server) NICHT fuer EU-Kunden nutzen.

**Integration-Aufwand:** 2-4 Wochen (bereits in Planung laut Luke)
- Woche 1: LiveKit Server Setup + Go-SDK Integration (Token-Generierung, Room-Management)
- Woche 2: React-Frontend (Video-UI, Controls, Teilnehmerliste)
- Woche 3: Chat-Integration (LiveKit Rooms in bestehendes Chat-System einbinden)
- Woche 4: Screen-Sharing, Recording (optional), Testing

**UI-Anforderungen:**
- Meeting-Button im Chat/Kalender: "Videokonferenz starten"
- Video-Grid (Gallery View) mit dynamischem Layout
- Controls: Mikrofon, Kamera, Screenshare, Verlassen
- Teilnehmerliste mit Mute-Status
- Chat waehrend Meeting (LiveKit Data Channels oder bestehender Chat)
- Meeting-Einladung per Link (Token-basiert)

**Empfehlung:**
1. LiveKit Server lokal per Docker starten
2. Go-SDK fuer Token-Generierung + Room-Management
3. `@livekit/components-react` fuer schnelle UI-Entwicklung
4. Self-Hosted auf Hetzner Dedicated Server (1 Gbps, ~40 EUR/Mo)

---

## 12. TipTap (Rich-Text Editor)

**Typ:** Open Source (Core) + Kommerziell (Pro/Cloud)
**Lizenz:** MIT (Core), Proprietary (Pro)
**Hersteller:** Tiptap GmbH, Berlin, Deutschland (EU!)
**Website:** https://tiptap.dev | Docs: https://tiptap.dev/docs

### Open Source vs. Pro

| Feature | Open Source (MIT) | Pro (kostenpflichtig) |
|---------|------------------|----------------------|
| **Rich-Text Editing** | Ja | Ja |
| **Extensions (Basis)** | Ja (30+ Extensions) | Ja |
| **Collaboration (Y.js)** | Ja (eigenen Server betreiben) | **Managed Collaboration Server** |
| **Comments** | NEIN | **JA** |
| **AI Integration** | NEIN | **JA** (AI Autocomplete, etc.) |
| **Table of Contents** | NEIN | **JA** |
| **Drag & Drop (advanced)** | Basis | **Enhanced** |
| **File Handler** | NEIN | **JA** (Drag&Drop Dateien) |
| **UniqueID** | NEIN | **JA** |
| **Content AI** | NEIN | **JA** |

### Brauchen wir Pro?

| Feature | Brauchen wir das? | Begruendung |
|---------|-------------------|-------------|
| **Collaboration** | NEIN (noch nicht) | Y.js Open Source reicht fuer v1. Managed Server spaeter. |
| **Comments** | JA (mittelfristig) | Wiki-Kommentare sind wichtig. Aber v1 kann ohne. |
| **AI Integration** | NEIN | KMU Hub hat eigene AI-Strategie (OpenAI API direkt). |
| **File Handler** | NICE-TO-HAVE | Dateien per Drag&Drop in den Editor. Kann selbst gebaut werden. |

**Empfehlung:** Open Source (MIT) fuer v1. Pro erst wenn Collaboration/Comments DRINGEND werden.

### Pricing Pro

| Plan | Kosten/Mo | Enthaelt |
|------|-----------|----------|
| **Starter** | ~29 EUR/Mo | Pro Extensions, 1 App |
| **Business** | ~99 EUR/Mo | + Collaboration Cloud, multiple Apps |
| **Enterprise** | Individuell | + SLA, Priority Support, On-Premise |

**Collaboration Cloud (Managed Y.js Server):**
- Ab ~49 EUR/Mo fuer bis zu 50 gleichzeitige Dokumente
- Alternative: Eigenen Y.js Server betreiben (kostenlos, aber Aufwand)

### React-Integration

```bash
npm install @tiptap/react @tiptap/starter-kit @tiptap/pm
```

**Basis-Setup:**

```tsx
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'

function WikiEditor({ content, onSave }) {
  const editor = useEditor({
    extensions: [StarterKit],
    content: content,
    onUpdate: ({ editor }) => {
      // Autosave alle 5 Sekunden (debounced)
      debouncedSave(editor.getJSON())
    },
  })

  return (
    <div className="wiki-editor">
      <MenuBar editor={editor} />
      <EditorContent editor={editor} />
    </div>
  )
}
```

### Extensions die wir brauchen

| Extension | Package | Open Source? | Prio |
|-----------|---------|-------------|------|
| **StarterKit** (Basis) | `@tiptap/starter-kit` | Ja | PFLICHT |
| **Heading** (H1-H6) | Im StarterKit | Ja | PFLICHT |
| **Bold/Italic/Strike** | Im StarterKit | Ja | PFLICHT |
| **BulletList/OrderedList** | Im StarterKit | Ja | PFLICHT |
| **Code/CodeBlock** | Im StarterKit | Ja | PFLICHT |
| **Blockquote** | Im StarterKit | Ja | PFLICHT |
| **Link** | `@tiptap/extension-link` | Ja | PFLICHT |
| **Image** | `@tiptap/extension-image` | Ja | PFLICHT |
| **Table** | `@tiptap/extension-table` | Ja | HOCH |
| **TaskList** (Checkboxes) | `@tiptap/extension-task-list` | Ja | HOCH |
| **Placeholder** | `@tiptap/extension-placeholder` | Ja | HOCH |
| **TextAlign** | `@tiptap/extension-text-align` | Ja | MITTEL |
| **Highlight** | `@tiptap/extension-highlight` | Ja | MITTEL |
| **Color** | `@tiptap/extension-color` | Ja | MITTEL |
| **Underline** | `@tiptap/extension-underline` | Ja | MITTEL |
| **Mention** (@user) | `@tiptap/extension-mention` | Ja | MITTEL |
| **Typography** | `@tiptap/extension-typography` | Ja | NIEDRIG |
| **CharacterCount** | `@tiptap/extension-character-count` | Ja | NIEDRIG |
| **Collaboration** (Y.js) | `@tiptap/extension-collaboration` | Ja | SPAETER |
| **CollaborationCursor** | `@tiptap/extension-collaboration-cursor` | Ja | SPAETER |

### Datenformat

TipTap speichert Inhalte als **JSON** (ProseMirror-Schema):

```json
{
  "type": "doc",
  "content": [
    {
      "type": "heading",
      "attrs": { "level": 1 },
      "content": [{ "type": "text", "text": "Wiki-Artikel Titel" }]
    },
    {
      "type": "paragraph",
      "content": [
        { "type": "text", "text": "Normaler Text mit " },
        { "type": "text", "marks": [{ "type": "bold" }], "text": "fettem Text" }
      ]
    }
  ]
}
```

**Speicher-Strategie:**
- JSON in PostgreSQL (`jsonb` Spalte)
- HTML-Export fuer E-Mail/PDF-Generierung: `editor.getHTML()`
- Markdown-Export moeglich mit `tiptap-markdown` Extension

**DSGVO:** Kein externer Dienst (Open Source, laeuft lokal). Bei Pro/Cloud: Deutsche Firma, EU-Server, AVV verfuegbar.

**Integration-Aufwand:** 2-3 Wochen
- Woche 1: TipTap-Setup + Extensions + Toolbar + Wiki-Integration
- Woche 2: Bild-Upload, Tabellen, Mentions, Autosave
- Woche 3: Versionierung (Diff-Ansicht), Export (HTML/PDF/Markdown)

**UI-Anforderungen:**
- Wiki-Modul: TipTap als Haupt-Editor (ersetzt bisherigen Plaintext)
- Toolbar: Formatierungsleiste (Bold, Italic, Headings, Listen, Link, Bild, Tabelle)
- Slash-Commands: "/" zum schnellen Einfuegen von Bloecken
- Bubble-Menu: Formatierung beim Selektieren von Text
- Floating-Menu: "+" Button am Zeilenanfang
- Autosave: Automatisches Speichern alle 5 Sekunden
- Versionierung: Aeltere Versionen anzeigen + wiederherstellen

**Empfehlung:**
1. `@tiptap/starter-kit` installieren + Basis-Editor aufsetzen
2. Extensions schrittweise hinzufuegen (Link, Image, Table zuerst)
3. Open Source genuegt fuer v1
4. Pro erst evaluieren wenn Collaboration/Comments gebraucht werden
5. JSON in PostgreSQL speichern (`jsonb`)

---

## Zusammenfassung: Kosten und Prioritaeten

### Jaehrliche Kosten fuer KMU Hub (SaaS)

| Integration | Einmalig | Jaehrlich | Kommentar |
|-------------|----------|-----------|-----------|
| OnlyOffice (Developer Ed.) | 0 | ~2.000-5.000 EUR | Pflicht fuer SaaS |
| Bexio API | 0 | 0 EUR | Kostenlos (Bexio verdient an Kunden) |
| Skribble | 0 | ~1.020 EUR (Business) | Pro Signatur zusaetzlich |
| Brevo | 0 | ~228-588 EUR (Starter-Business) | Pro KMU Hub Instanz |
| CleverReach | 0 | ~180-420 EUR | Alternative zu Brevo |
| FinAPI | 0-5.000 | ~2.400-6.000 EUR | Teuerste Integration |
| Nextcloud | 0 | 0 EUR | Open Source, Protokoll |
| DATEV | 0 | 0 EUR | Nur Export-Format |
| Swiss QR-Code | 0 | 0 EUR | Offener Standard |
| ZUGFeRD/XRechnung | 0 | 0-500 EUR | gobl = Open Source; unipdf = ~500 EUR/Jahr |
| LiveKit | 0 | ~480 EUR (Hetzner Server) | Self-Hosted |
| TipTap | 0 | 0 EUR (Open Source) | Pro spaeter: ~348-1.188 EUR/Jahr |
| **GESAMT** | **0-5.000 EUR** | **~6.300-14.000 EUR/Jahr** | |

### Implementierungs-Reihenfolge (empfohlen)

| Phase | Integration | Aufwand | Business Impact |
|-------|------------|---------|-----------------|
| **1 (Sofort)** | DATEV-Export | 1-2 Wo. | KRITISCH fuer DE |
| **1 (Sofort)** | Swiss QR-Code | 1-2 Wo. | KRITISCH fuer CH |
| **2 (Kurzfristig)** | TipTap (Wiki-Editor) | 2-3 Wo. | HOCH (Wiki unbrauchbar ohne) |
| **2 (Kurzfristig)** | ZUGFeRD/XRechnung | 2-3 Wo. | HOCH (wird Pflicht) |
| **3 (Mittelfristig)** | Bexio API | 2-4 Wo. | HOCH fuer CH |
| **3 (Mittelfristig)** | LiveKit | 2-4 Wo. | HOCH (Video-Meetings) |
| **3 (Mittelfristig)** | Brevo | 2-3 Wo. | MITTEL-HOCH |
| **4 (Langfristig)** | OnlyOffice | 2-4 Wo. | HOCH (ersetzt Office 365) |
| **4 (Langfristig)** | Skribble | 2-3 Wo. | MITTEL |
| **4 (Langfristig)** | Nextcloud | 2-3 Wo. | MITTEL |
| **5 (Spaeter)** | CleverReach | 2-3 Wo. | MITTEL (Alternative zu Brevo) |
| **5 (Spaeter)** | FinAPI | 3-4 Wo. | MITTEL (teuer, Premium-Feature) |

### Gesamter Implementierungsaufwand

**~28-42 Wochen** (alle 12 Integrationen)

Bei 1 Backend-Entwickler (Luke) und paralleler Arbeit an Kern-Features: **8-12 Monate** realistisch.

**Empfehlung:** Phase 1+2 (DATEV, QR-Code, TipTap, ZUGFeRD) in den naechsten 2-3 Monaten. Phase 3 (Bexio, LiveKit, Brevo) in den darauffolgenden 3 Monaten.

---

### Offene Fragen (VOR Implementation klaeren)

1. **OnlyOffice Lizenz:** Developer Edition Konditionen verhandeln -- VOR SaaS-Launch
2. **FinAPI:** Lohnt sich der Preis? Alternativen fuer CH pruefen (bLink/SIX)
3. **Skribble vs. DocuSign:** Reicht Skribble als einziger E-Signatur-Provider?
4. **TipTap Pro:** Wann brauchen wir Collaboration? Noch vor Beta?
5. **LiveKit Recording:** Sollen Meetings aufgezeichnet werden koennen? (DSGVO-Implikationen!)
6. **Brevo/CleverReach:** Beide implementieren oder nur eines?
7. **DATEV Online API:** Reicht CSV-Export oder brauchen wir die Online-Schnittstelle?
8. **ZUGFeRD PDF/A-3:** `unidoc/unipdf` (kommerziell) oder Open-Source-Alternative?

---

*Hinweis: Preise basieren auf Stand ~Q1 2025. Vor vertraglichen Entscheidungen aktuelle Preise auf den jeweiligen Websites verifizieren. URLs koennen sich aendern.*
