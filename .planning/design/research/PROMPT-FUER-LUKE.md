# Prompt fuer Luke — Produkt-Strategie & Backend-Planung

Kopiere diesen Prompt und gib ihn Luke (oder seinem Claude/ChatGPT) zusammen mit den Referenz-Dokumenten.

---

## DER PROMPT:

```
Wir haben heute eine umfassende Produkt-Strategie und Marktanalyse fuer KMU Hub durchgefuehrt.
KMU Hub soll das "Betriebssystem fuer DACH-KMUs" werden — eine All-in-One-Plattform die
Microsoft 365 + Teams + CRM + PM + Helpdesk + Zeiterfassung + HR ersetzt.

Hier ist was wir erarbeitet haben. Bitte lies ALLE Dokumente durch und erstelle dann:
1. Deinen erweiterten Backend-Implementierungsplan
2. Fragen/Anmerkungen wo du anderer Meinung bist oder Klarstellung brauchst
3. Eine priorisierte Sprint-Planung aus Backend-Sicht

## Kontext

### Vision
- KMU Hub als Betriebssystem: Einmal einrichten, laeuft ueberall im Unternehmen
- Lokal-First: Docker-Paket auf Firmenserver/NAS, Zugang von unterwegs per VPN
- AUCH als SaaS (Hetzner DE / Exoscale CH)
- 100% EU/CH-Hosting, kein US Cloud Act, Self-hosted Option
- Datensicherheit als KERN-USP, nicht Nachgedanke

### Kernentscheidungen

**BAUEN (in unserem Style):**
- CRM (Kontakte, Deals, Pipeline) — erweitern: Custom Fields, Firma als Entity, Duplikaterkennung
- Rechnungen & Finanzen (NICHT Buchhaltung!) — Belegkette, QR-Rechnung, ZUGFeRD, PDF, GoBD
- Helpdesk (erweitern: Canned Responses, Private Notes)
- Projektmanagement (erweitern: Gaeste-Zugang)
- Chat → Unified Inbox (E-Mail + Teams + WhatsApp + Website-Widget in EINER Inbox)
- Video/Meetings (LiveKit self-hosted + Zoom-Fallback fuer kleine KMUs)
- Office-Editing (OnlyOffice Document Server eingebettet, WOPI-Protokoll)
- Wiki (TipTap Rich-Text-Editor)
- Zeiterfassung, Schichtplanung, Fuhrpark, Inventar, Einkauf, Rapporte, Vermietung, Formulare, Vertraege
- Kundenportal (Kunden sehen Projekte/Tickets/Dokumente)
- Status/Praesenz-System (Online/Abwesend/Busy)

**INTEGRIEREN (nie selbst bauen):**
- Buchhaltung/FiBu → DATEV-Export + Bexio-API (GoBD-Testat zu teuer)
- Lohnabrechnung → NIEMALS (26 Kantone CH, 16 Bundeslaender DE)
- Newsletter → Brevo/CleverReach API
- E-Signatur → Skribble (Schweizer Firma, ZertES+eIDAS)
- Banking → FinAPI (4000+ Banken, PSD2)
- Office-Editor → OnlyOffice Document Server (WOPI)

**NIEMALS BAUEN:**
- Eigener Mailserver (Deliverability-Problem)
- Volles ERP (SAP-Territorium)
- Kassensystem/POS
- PSTN-Telefonie
- Recruiting/ATS
- Sprint Planning/Scrum

### Unified Inbox (WICHTIGSTES NEUES FEATURE)
Problem: Unsere Kunden kommunizieren mit IHREN Kunden ueber Teams/WhatsApp/E-Mail.
Wenn wir nur internen Chat haben, fehlt die externe Kommunikation.

Loesung: Alle externen Kanaele kommen in EINE Inbox:
- E-Mail (IMAP/SMTP) — kommt mit E-Mail-Modul
- Teams Bridge (Microsoft Graph API)
- WhatsApp Business (Meta Cloud API)
- Website Chat-Widget (WebSocket, embeddable JS)
- Kundenportal (eigener Auth-Flow)

Backend braucht:
- Unified Message Model (channel, direction, contact_id, content, metadata)
- Channel Adapter Interface pro Kanal
- WebSocket fuer Echtzeit
- Queue fuer ausgehende Nachrichten (Retry, Rate Limiting)
- OAuth2 Flows fuer Teams/Slack
- WhatsApp Business API Webhook Handler
- Widget Token-Auth (JWT)

### Teams-Ersatz
Wir ersetzen Microsoft Teams komplett:
- Chat (1:1, Gruppen, Channels) — haben wir, braucht: Threads, Reactions, @mentions, Datei-Sharing
- Video/Audio — LiveKit + Zoom-Fallback
- Screen-Sharing — LiveKit
- Status/Praesenz — WebSocket + Heartbeat
- Benachrichtigungen — Push Notifications
- Globale Suche — ueber alle Module

### Preismodell (aus Kostenanalyse)
- Starter: 12 EUR/User/Mo — Kernmodule, 25 GB, Basis-Video
- Business: 19 EUR/User/Mo — + OnlyOffice, unbegrenzt Video, Newsletter
- Enterprise: 25 EUR/User/Mo — + Banking, SSO, SLA, 500 GB
- Self-Hosted: Jahreslizenz (Preis TBD)
- Hosting kostet uns ~2 EUR/Kunde/Mo bei 100 Kunden → 97%+ Bruttomarge
- Break-Even bei ~37 Kunden

### Zielgruppen (Reihenfolge)
1. Dienstleister/Agenturen (85% Feature-Abdeckung) — ERSTE ZIELGRUPPE
2. Handwerk (80%)
3. Bau (70%)
4. Handel (65%)
5. Gastro (55%)

## Referenz-Dokumente

Die folgenden Dateien liegen in `.planning/design/research/` (lokal, nicht committed):

| # | Datei | Inhalt | Zeilen |
|---|-------|--------|--------|
| 00 | 00-SYNTHESE.md | Gesamtsynthese mit Build/Integrate Matrix | ~400 |
| 01 | 01-office-email-storage.md | Office, E-Mail, Storage Recherche | ~800 |
| 02 | 02-crm-sales-helpdesk.md | CRM, Sales, Helpdesk Recherche | ~800 |
| 03 | 03-finanzen-buchhaltung-hr.md | Finanzen, Buchhaltung, HR Recherche | ~550 |
| 04 | 04-projektmanagement-erp-branche.md | PM, ERP, Branchenloesungen | ~1200 |
| 05 | 05-dsgvo-dsg-compliance.md | DSGVO/DSG Basis-Recherche | ~600 |
| 06 | 06-modul-gap-analyse.md | Gap-Analyse aller 23 Module | ~850 |
| 07 | 07-compliance-framework.md | Tiefes Compliance-Framework | ~1740 |
| 08 | 08-datenbankmodelle.md | 30 neue PostgreSQL-Tabellen | ~1600 |
| 09 | 09-infrastruktur-matrix.md | Server-Setups nach Groesse/Branche | ~1540 |
| 10 | 10-integrations-guide.md | 12 Integrationen mit Lizenzen/Setup | ~1500 |
| 11 | 11-backend-plan.md | Backend-Implementierungsplan | ~??? |
| 12 | 12-kostenanalyse-preismodell.md | Kosten pro Modul + Preisstufen | ~1500 |
| 13 | 13-vision-ergaenzungen.md | Unified Inbox, Office, Teams-Ersatz | ~200 |
| 14 | 14-frontend-plan.md | Frontend-Implementierungsplan | ~??? |

Plus:
- `external/chatgpt-research-summary.md` — ChatGPT Deep Research Marktanalyse
- `external/Marktanalyse [...].pdf` — Original 20-Seiten PDF

## Was ich von dir brauche

1. **Lies den Backend-Plan (11-backend-plan.md)** und die Datenbankmodelle (08) durch
2. **Erweitere/korrigiere** wo noetig — du kennst den Code besser
3. **Priorisiere aus Backend-Sicht:** Was zuerst, was kann warten?
4. **Unified Inbox Architektur:** Wie baust du die Channel-Adapter? Message Queue? WebSocket?
5. **WOPI-Endpoints fuer OnlyOffice:** Hast du Erfahrung damit?
6. **LiveKit Integration:** Room Management, Token Generation, Recording?
7. **Teams Bridge:** Microsoft Graph API — wie aufwaendig ist das wirklich?
8. **WhatsApp Business API:** Webhook-Handling, Message Templates?
9. **Sicherheit:** RLS, Audit-Logs, Verschluesselung — wie siehst du die Prioritaet?
10. **Sprint-Reihenfolge:** Was ist dein Vorschlag fuer die naechsten 6 Monate?

Wichtig: Buchhaltung/FiBu und Lohnabrechnung bauen wir NICHT. Nur Rechnungen/Angebote/Mahnwesen
mit DATEV-Export und Bexio-Sync. Das Modul heisst "Rechnungen & Finanzen", nicht "Buchhaltung".
```

---

## ANLEITUNG FUER DARIEN

1. Kopiere den Prompt oben
2. Gib Luke Zugang zu den Research-Dateien (z.B. per USB-Stick, Fileshare, oder temporaer committen)
3. Luke gibt den Prompt + Dateien seinem Claude/ChatGPT
4. Luke erweitert den Backend-Plan und gibt Feedback

### Option A: Dateien per Git teilen
```bash
# Temporaer committen auf separatem Branch
git checkout -b planning/handoff
git add .planning/design/research/
git commit -m "temp: research docs for Luke review"
git push origin planning/handoff
# Luke pullt den Branch, liest alles, Branch wird danach geloescht
```

### Option B: Dateien kopieren
Einfach den ganzen `.planning/design/research/` Ordner auf einen USB-Stick
oder per Fileshare an Luke schicken.

### Option C: Masterplan committen (EMPFOHLEN)
Der Masterplan (`docs/BACKEND-HANDOFF.md`) wird auf `design/brainstorm` committed.
Enthaelt alles Wichtige kompakt in einem Dokument. Luke pullt den Branch.
