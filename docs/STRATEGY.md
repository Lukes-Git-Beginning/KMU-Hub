# KMU Hub — Strategie: Digitale Souveraenitaet + MS Office Koexistenz

## 1. Wettbewerber-Analyse (DACH All-in-One KMU Software)

| Anbieter | Preis/User/Mo | Self-Hosted | Staerke | Schwaeche |
|----------|--------------|-------------|---------|-----------|
| Odoo | 31 EUR (alle Apps) | Ja (Open Source) | Riesig modular, 27k+ Kunden | Komplex, braucht Implementierungspartner |
| weclapp | 39-69 EUR | Nein | Made in Germany, DSGVO | Teuer, nur ~1.360 Kunden |
| 2WORK | 24 EUR (flat) | Nein | Sehr guenstig | Nur Dienstleister, wenig Tiefe |
| Myfactory | 30-50 EUR | Nein | Deutsche Server, Handel/Produktion | Klassisch, wenig Innovation |
| Monday CRM | 10-24 EUR | Nein | Einfach, gutes PM | Kein ERP, US-basiert |

### KMU Hub Positionierung

Was keiner der Wettbewerber bietet:
- **Prozessanalyse-Paket** mit persoenlicher Beratung (2-3 Meetings)
- **Self-Hosted + SaaS** als gleichwertige Optionen
- **Branchenprofile** (10 vorkonfigurierte Profile fuer sofortigen Start)
- **WASM-Plugin-System** fuer echte Erweiterbarkeit
- **Integrierter Office-Ersatz** (OnlyOffice) + Chat + Video + Mail in einem Tool

---

## 2. Bestandsaufnahme: Was KMU Hub schon als Office-Alternative hat

| MS Office Tool | KMU Hub Aequivalent | Modul/Datei | Status |
|----------------|---------------------|-------------|--------|
| Outlook (Mail) | Mail-Modul (IMAP/SMTP, beliebiger Provider) | `modules/mails/MailsPage.tsx` | Fertig |
| Outlook (Kalender) | Kalender-Modul (CalDAV Sync) | `modules/calendar/CalendarLayout.tsx` | Fertig |
| Teams (Chat) | Chat-Modul (Channels, DMs, Threads) | `modules/chat/ChatLayout.tsx` | Fertig |
| Teams (Video) | Video/Meetings (LiveKit, self-hostable) | `modules/meetings/` | Fertig |
| Word / Excel / PPT | Dokumente-Modul (OnlyOffice via WOPI) | `modules/dokumente/DokumentePage.tsx` | Fertig |
| OneDrive | Dateiverwaltung (Personal/Team/Projekt Spaces) | `modules/dokumente/` | Fertig |
| SharePoint Wiki | Wiki-System | `modules/dokumente/` (Wiki-Tab) | Fertig |

### OnlyOffice Integration (bereits vorhanden)

- WOPI-Protokoll fuer kollaboratives Bearbeiten
- Unterstuetzte Formate: .docx, .xlsx, .pptx, .odt, .ods, .odp, .csv, .txt
- Echtzeit-Kollaboration mit farbigen Cursorn
- Versionierung bei jedem Speichern
- Self-hostable (EU-Server)
- Herkunft: Lettland (EU), Open Source

### TipTap Rich Text Editor (bereits vorhanden)

- Verwendet in: Meeting-Notizen, Wiki-Artikel, E-Mail-Erstellung
- Features: Bold, Italic, Tabellen, Listen, Links, Bilder, Code-Bloecke
- Pfad: `components/shared/RichTextEditor/`

**Fazit:** ~90% eines MS Office Ersatzes ist bereits gebaut. Es fehlt nur die externe Kommunikationsbruecke.

---

## 3. Strategie: Dual-Mode Sovereignty

### Kernidee

Nicht "ersetze Office" und nicht "integriere nur Office" — sondern beides parallel. Der Kunde startet wo er ist und migriert natuerlich, ohne jemals eingeschraenkt zu sein.

### Schicht 1: Souveraener Kern (90% fertig)

Alles was intern laeuft, laeuft ueber KMU Hub eigene Systeme:

| Bereich | Technologie | Hosting |
|---------|-------------|---------|
| Chat | Eigenes System | Self-hosted / EU SaaS |
| Mail | IMAP/SMTP (beliebiger Provider) | Kundenserver / EU SaaS |
| Video | LiveKit | Self-hosted / EU SaaS |
| Docs | OnlyOffice (WOPI) | Self-hosted / EU SaaS |
| Kalender | CalDAV | Self-hosted / EU SaaS |
| Dateien | Eigener Storage | Self-hosted / EU SaaS |

**Alle Daten bleiben auf EU-Servern oder beim Kunden selbst.**

Mail ersetzt Outlook sofort und komplett — IMAP/SMTP funktioniert mit jedem Provider. Kalendereinladungen (.ics) werden nativ verarbeitet und in den KMU Hub Kalender uebernommen.

### Schicht 2: Smart Bridge (zu bauen)

Fuer externe Kommunikation mit Partnern/Kunden die noch Teams/Outlook nutzen:

| Bridge | Funktion | User-Erlebnis |
|--------|----------|---------------|
| Teams-Bridge | Teams-Nachrichten erscheinen in KMU Hub Chat | Ein Kanal "Externe (Teams)" neben internen |
| Datei-Kompatibilitaet | .docx/.xlsx werden nativ bearbeitet | Kein Konvertieren — OnlyOffice macht das |
| Mail-Bridge | Exchange/Outlook Mails in KMU Hub Mail | Ein Postfach, egal welcher Provider |

Der Mitarbeiter arbeitet NUR in KMU Hub. Externe Nachrichten kommen rein, Antworten gehen raus — aber die interne Verarbeitung und Speicherung bleibt souveraen.

---

## 4. Externe Kommunikation: Gast-Chat / Kundenportal

### Das Problem

Wenn Kunden kein Teams mehr nutzen, brauchen sie einen anderen Kanal fuer Echtzeit-Kommunikation. E-Mail deckt ~75% ab, aber fuer schnelle Rueckfragen fehlt eine Loesung.

### Loesung Phase 1: Gast-Chat (Link-basiert)

- Kunde bekommt einen Link (z.B. `hub.deinefirma.de/chat/gast/abc123`)
- Oeffnet im Browser — keine Installation, kein Account noetig
- Kann chatten, Dateien schicken, optional Videocall starten
- Fuer den KMU Hub User erscheint es als normaler Chat-Kanal
- Aehnlich wie Chatwoot/Intercom, aber B2B-fokussiert und self-hostable

**Beispiel-Workflow:**
1. Handwerker schickt Kunden den Chat-Link per Mail
2. Kunde klickt, schreibt "Wann kommt ihr morgen?"
3. Handwerker antwortet direkt aus KMU Hub
4. Dateien (Angebote, Fotos) koennen hin und her geschickt werden

**Vorteile:**
- Null Huerde fuer den Kunden (nur Link anklicken)
- Daten bleiben auf dem KMU Hub Server
- Kein Drittanbieter noetig
- Funktioniert sofort, ohne dass der Kunde etwas installiert

### Loesung Phase 2 (spaeter): Matrix-Protokoll

Matrix ist das foederierte Messaging-Protokoll das die EU-Kommission gerade testet. Wenn es sich als Standard etabliert:

- KMU Hub spricht Matrix-Protokoll
- Kommunikation mit anderen Matrix-Nutzern direkt moeglich
- Bruecken zu Teams/Slack/WhatsApp ueber Matrix-Bridges
- Ende-zu-Ende verschluesselt

**Aber:** Matrix ist im KMU-Bereich noch nicht verbreitet genug. Kein Handwerkskunde wird sich Element installieren. Deshalb erst Phase 2, wenn der EU-Druck staerker wird.

---

## 5. EU-Trends: Regulierungsdruck als Rueckenwind

### Aktuelle Entwicklungen (Stand Februar 2026)

| Institution | Aktion | Status |
|-------------|--------|--------|
| EU-Kommission | Testet Matrix/Element als Teams-Ersatz | Pilot laeuft (Feb 2026) |
| Frankreich | Matrix-basiertes "Tchap" fuer Regierung | Produktiv |
| Bundeswehr (DE) | "BwMessenger" auf Matrix-Basis | Produktiv |
| Niederlande | Migration weg von Microsoft | In Umsetzung |
| EDPS | Rote Flaggen zu MS 365 GDPR-Compliance | Laufende Pruefung |
| DE Datenschutzbeauftragter | Warnung: MS Teams nicht GDPR-konform nutzbar | Offizielle Stellungnahme |

### Was das fuer KMU Hub bedeutet

1. **KMUs die mit Behoerden arbeiten** werden EU-souveraene Tools brauchen
2. **GDPR-Durchsetzung** wird strenger — MS 365 Nutzung wird riskanter
3. **"Digital Sovereignty" wird Verkaufsargument** — nicht nur Ideologie sondern Business-Notwendigkeit
4. **Frueh positionieren** zahlt sich aus wenn der Druck zunimmt

---

## 6. Verkaufsargumente

### Fuer den Geschaeftsfuehrer

> "Spare 3-4 Abos (Office, Chat, Video, CRM), zahle nur eins. Deine Daten bleiben in der EU. Und deine Kunden merken keinen Unterschied."

**Kostenvergleich pro Mitarbeiter/Monat:**

| Heute (einzelne Tools) | Mit KMU Hub |
|-------------------------|-------------|
| MS 365 Business: 12-22 EUR | Alles inklusive |
| CRM: 15-40 EUR | |
| Slack/Teams: 7-13 EUR | |
| Zoom: 10-14 EUR | |
| **Summe: 44-89 EUR** | **29-49 EUR** |

### Fuer den IT-Verantwortlichen

> "Self-hostable, GDPR-konform, OnlyOffice statt MS Cloud. Die EU-Kommission macht gerade dasselbe. Ein System statt fuenf zu administrieren."

### Fuer den Mitarbeiter

> "Alles an einem Ort — Mail, Chat, Video, Dokumente, Kalender. Kein Wechsel zwischen 5 Apps. Und .docx/.xlsx Dateien oeffnen sich direkt."

---

## 7. Was NICHT gebaut werden muss

| Feature | Warum nicht |
|---------|-------------|
| Eigener Word-Klon | OnlyOffice ist bereits integriert und besser als alles was wir bauen koennten |
| Eigener Excel-Klon | OnlyOffice |
| Eigener PowerPoint-Klon | OnlyOffice |
| Eigener Matrix-Server | Eigenes Chat-System reicht, Matrix spaeter als Protokoll-Layer |
| Eigener E-Mail-Server | IMAP/SMTP funktioniert mit jedem Provider |

**Fokus:** Integration und UX verbessern, nicht das Rad neu erfinden.

---

## 8. Roadmap-Einordnung

### Nicht Teil der aktuellen Waves (11-20)

Die Strategie beeinflusst keine laufende Entwicklung. Waves 11-13 (Module) und 14-20 (UI-Redesign) laufen wie geplant weiter.

### Post-MVP / v1.1 Scope

| Feature | Aufwand | Prioritaet | Abhaengigkeit |
|---------|---------|------------|---------------|
| Gast-Chat / Kundenportal | Gross | Hoch | Backend (Luke) |
| Teams-Webhook-Bridge | Mittel | Mittel | Backend (Luke) |
| Souveraenitaets-Dashboard | Klein | Niedrig | Frontend |
| Import-Assistenten (Teams/OneDrive) | Mittel | Niedrig | Backend |
| Matrix-Protokoll Support | Gross | Spaeter | Backend + Infrastruktur |

### Strategische Entscheidungen die JETZT gelten

1. **Integration statt Konkurrenz** — MS Office bleibt kompatibel, nicht ersetzt
2. **OnlyOffice ist unser Office-Stack** — kein eigener Editor
3. **Gast-Chat vor Matrix** — pragmatisch statt idealistisch
4. **EU-Souveraenitaet als Feature verkaufen** — nicht als Einschraenkung
5. **Self-Hosted bleibt gleichwertige Option** — nicht nur SaaS
