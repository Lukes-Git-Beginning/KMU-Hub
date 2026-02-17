# Ergaenzungen aus Produkt-Diskussion (2026-02-16)

Diese Punkte kamen NACH den Research-Agents aus der Diskussion mit Darien
und muessen in Frontend-Plan, Backend-Plan und Phase 7 einfliessen.

---

## 1. KMU Hub als Betriebssystem (Lokal-First)

**Vision:** KMU Hub wird einmal eingerichtet und laeuft ueberall im Unternehmen.
Nicht nur SaaS — auch lokale Installation auf Firmenserver/NAS.

**Setups nach Groesse:**
- **Klein (5-20 MA):** Ein Server im Buero oder NAS, Docker-Compose, Zugang von unterwegs per VPN/Tailscale
- **Mittel (20-100 MA):** Dedizierter Server (Hetzner Cloud oder lokal), VPN fuer Remote
- **Gross (100-200 MA):** Cluster-Setup, mehrere Standorte verbunden

**Implikation:**
- Docker-Paket das IT-Dienstleister beim Kunden installiert
- Offline-faehig (Electron Desktop-App)
- Automatische Updates (Docker Image Tags)
- Remote-Zugang sicher (VPN/Tailscale, kein offenes Internet)

---

## 2. Zwei getrennte Module: Chat (intern) + Kommunikation (extern)

**WICHTIG:** Team-Chat und externe Kommunikation sind ZWEI SEPARATE Module.
Nicht in einen Topf werfen — uebersichtlich halten!

### Modul "Chat" (Team-intern) — BESTEHT SCHON (Luke)
- Kollegen-Chat, Channels (#sales, #dev), Gruppen, DMs
- Schnell, informell, Reactions, GIFs
- Wie Slack/Teams fuers Team
- Sidebar-Icon: 💬 Chat

### Modul "Kommunikation" (Extern) — NEU
- ALLE externen Nachrichten an einem Ort
- E-Mails, WhatsApp, Teams-Bridge, Website-Widget, Kundenportal
- Formeller, CRM-verknuepft, nachverfolgbar
- Sidebar-Icon: 📨 Kommunikation

### Verbindung zwischen Chat und Kommunikation:
- "An Kollege weiterleiten" (Kommunikation → Chat)
- "@Firma" im Chat → Link zur Konversation in Kommunikation
- "Ticket erstellen" aus Kundenmail → Helpdesk
- Gleicher CRM-Kontakt, verschiedene Kanaele

### Externe Kanaele (in Kommunikation):
| Kanal | Technologie | Aufwand | Prioritaet |
|-------|------------|---------|-----------|
| E-Mail | IMAP/SMTP (Luke baut es gerade!) | Teil von E-Mail | KRITISCH |
| Teams Bridge | Microsoft Graph API, Bot Framework | 3-4 Wo. | HOCH |
| WhatsApp Business | Meta Cloud API (WhatsApp Business Platform) | 2-3 Wo. | HOCH |
| Website Chat-Widget | WebSocket, embeddable JS-Snippet | 2-3 Wo. | HOCH |
| Kundenportal | Eigener Auth-Flow, eingeschraenkte Ansicht | 3-4 Wo. | MITTEL |
| Slack Bridge | Slack Events API + Web API | 2-3 Wo. | NIEDRIG |

### UI-Konzept (Kommunikation):
- Tab-basiert: [E-Mail] [Teams] [WhatsApp] [Widget] [Portal]
- Konversations-Liste links, Nachrichten-Thread rechts
- Kontext-Panel rechts: CRM-Kontakt, offene Deals, Tickets, Projekte
- Automatische CRM-Kontakt-Zuordnung
- "Aus CRM einfuegen" Button fuer Dokumente/Angebote

### Backend-Anforderungen:
- Unified Message Model (channel, direction, contact_id, content, metadata)
- Channel Adapters (Interface pro Kanal)
- WebSocket fuer Echtzeit
- Queue fuer ausgehende Nachrichten (Retry, Rate Limiting)
- OAuth2 Flows fuer Teams/Slack Verbindungen
- WhatsApp Business API Webhook Handler
- Widget Token-Auth (JWT fuer eingebettetes Widget)

---

## 3. Office-Ersatz (Word/Excel/PowerPoint IN der App)

**Anforderung:** Nicht nur Browser — OnlyOffice direkt in Electron eingebettet.

### Strategie:
- **OnlyOffice Document Server** laeuft als Docker-Container (lokal oder Cloud)
- In Electron: Webview zum lokalen/Cloud OnlyOffice Server
- .docx, .xlsx, .pptx oeffnen und bearbeiten direkt in KMU Hub
- Gleichzeitiges Bearbeiten (Co-Editing) wenn mehrere User offen haben

### Vorlagen-System:
- Briefvorlagen mit Firmenlogo
- Angebotsvorlagen die CRM-Daten ziehen (Kontaktname, Firma, Adresse)
- Rechnungsvorlagen mit QR-Code (CH) / ZUGFeRD (DE)
- "Neues Dokument aus Vorlage" — ein Klick
- Vorlagen-Editor fuer Admins

### Fuer kleine KMUs ohne Docker:
- TipTap-Editor fuer einfache Dokumente (Briefe, Notizen)
- PDF-Export
- OnlyOffice optional als Add-On

### Frontend:
- Neues UI: "Dokument oeffnen" → OnlyOffice Webview statt Download
- Vorlagen-Galerie in Dokumente-Modul
- "Neues Dokument" Dialog mit Vorlagen-Auswahl

### Backend:
- WOPI-Endpoints in Go
- Vorlagen-Storage (S3/MinIO)
- Template-Engine (Platzhalter → CRM-Daten)

---

## 4. Teams-Ersatz (Vollstaendige Kommunikationsplattform)

### Was Teams hat und wir brauchen:
| Teams-Feature | KMU Hub Status | Was fehlt |
|--------------|---------------|-----------|
| Chat 1:1 + Gruppen | Gebaut (Luke) | Threads, Reactions, Datei-Sharing |
| Channels | Gebaut | Thread-Replies |
| Video/Audio | Geplant (LiveKit) | Einbettung in UI |
| Screen-Sharing | Kommt mit LiveKit | — |
| Dateien teilen | Dokumente-Modul | Drag&Drop in Chat |
| Kalender-Integration | Gebaut | Meeting aus Chat starten |
| Status/Praesenz | FEHLT | Online/Abwesend/Busy/DND |
| @mentions | FEHLT | In Chat + ueberall |
| Benachrichtigungen | FEHLT | OS-native Notifications |
| Suche | FEHLT | Globale Suche ueber alles |
| Externe Gaeste | FEHLT | Unified Inbox loest das |
| Apps/Bots | FEHLT | Spaeter (v2+) |

### Unser Vorteil gegenueber Teams:
- Teams = NUR Kommunikation. KMU Hub = Kommunikation + CRM + PM + Finance + ...
- Alles verbunden: Deal im Chat teilen, Meeting aus Projekt starten, Ticket im Chat loesen
- EU-hosted, kein Microsoft noetig
- Externe Kommunikation BESSER als Teams (Unified Inbox)

---

## 5. Zoom-Integration fuer kleine Unternehmen

**Problem:** LiveKit self-hosted kostet Server-Ressourcen. Ein 5-Mann-Betrieb braucht das nicht.

**Loesung:** Gestufte Video-Strategie:
- **Starter-Tier:** Zoom/Google Meet Integration (Link wird in Kalender/Projekte eingebettet)
- **Business-Tier:** LiveKit self-hosted (eigener Videoserver)
- **Enterprise-Tier:** LiveKit mit Recording + Webinar-Modus

### Frontend:
- "Meeting starten" Button → Wenn Zoom konfiguriert: oeffnet Zoom-Link. Wenn LiveKit: oeffnet internes Meeting.
- Settings: "Video-Provider" Auswahl (Zoom / Google Meet / LiveKit / Keiner)
- Zoom OAuth2 Integration fuer automatische Meeting-Erstellung

### Backend:
- Zoom API Integration (Meeting erstellen, Link generieren)
- Google Calendar API fuer Meet-Links
- Abstraction Layer: VideoProvider Interface

---

## 6. Module anpassen: Build vs. Integrate

### Buchhaltung → "Rechnungen & Finanzen" umbenennen
- KEIN FiBu (doppelte Buchfuehrung) — DATEV/Bexio integrieren
- BEHALTEN: Rechnungen, Angebote, Mahnwesen, Belegkette
- NEU: DATEV-Export-Panel, Bexio-Sync-Dashboard
- NEU: QR-Rechnung (CH), ZUGFeRD (DE)

### Team/HR → Lohn entfernen
- Lohnabrechnung NIEMALS bauen
- Lohn-Tab → Integrations-Panel (DATEV Lohn / Abacus HR Link)
- BEHALTEN: Mitarbeiter, Urlaub, Abwesenheiten, Schulungen

### Integration-UIs hinzufuegen:
- DATEV-Export Panel (in Rechnungen & Finanzen)
- Bexio-Sync Dashboard (in Rechnungen & Finanzen)
- OnlyOffice Viewer (in Dokumente)
- E-Signatur Dialog (in Vertraege)
- Newsletter Panel (in CRM/Kontakte)
- Banking Widget (in Rechnungen & Finanzen)

---

## 7. KI-Features (aus ChatGPT-Recherche)

KI ist kein Nice-to-have mehr — KMUs erwarten es:
- **Zusammenfassungen:** Meeting-Notes, Ticket-Verlaeufe, CRM-Aktivitaeten
- **Entwuerfe:** E-Mail-Antworten, Ticket-Responses, Angebotstexte
- **Suche:** Semantische Suche ueber Wiki/Docs/Tickets/CRM
- **Datenklassifizierung:** Auto-Tag (oeffentlich/intern/vertraulich)
- **KI-Governance:** Opt-out pro Modul, kein Training auf Kundendaten, Logging

---

## 8. Datensicherheit als Kern-USP

- 100% EU/CH-Hosting, kein US Cloud Act
- Self-hosted Option (volle Kontrolle)
- Verschluesselung: at rest (AES-256), in transit (TLS 1.3)
- Tenant-Isolation (RLS + separate Keys)
- Backup-Verschluesselung pro Mandant
- Tamper-proof Audit-Logs
- Kein Datentransfer in Drittlaender
- DSGVO/nDSG-konform by Design
