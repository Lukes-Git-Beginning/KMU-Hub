---
tags: [customization, helpdesk, csat, statistik, research]
updated: 2026-07-26
---

# Statistik-Recherche: Helpdesk CSAT + anpassbare Dashboards

Recherche-Ergebnis für den Modul-Editor (Statistik-Seite Helpdesk). Grundlage: Zendesk, Freshdesk, Zoho Desk, Intercom, HubSpot Service Hub.

---

## 1. CSAT – Marktmuster

### Mechanismus (einheitlicher Standard)

Alle großen Helpdesk-Tools folgen demselben Grundmuster:

1. Ticket wird geschlossen / als "gelöst" markiert
2. System wartet konfigurierbaren Delay (Standard: 10–24h)
3. **E-Mail an Kunden** mit eingebettetem Ein-Klick-Rating direkt im Mail-Body
4. Klick öffnet optional kurzes Kommentarfeld (kein separates Formular nötig)
5. Antwort wird dem Ticket zugeordnet und im Analytics-Dashboard aggregiert

Das Prinzip ist: **so wenig Reibung wie möglich** (1 Klick in der Mail reicht), damit Responserate hoch bleibt.

### Skalen im Vergleich

| Tool | Standard-Skala | Unterliegendes Modell |
|---|---|---|
| **Zendesk** | Anpassbar: 2, 3 oder 5 Optionen (Zahlen, Emojis oder Text), UI zeigt Sterne oder Symbole | Binary Good/Bad intern — alle Varianten werden auf positiv/negativ gemappt |
| **Freshdesk** | 2, 3, 5 oder 7 Punkte — Smileys, Sterne, Text oder Zahl wählbar | Gruppen: negativ / neutral / positiv |
| **Zoho Desk** | Nativ: Binary (Gut/Schlecht) + Kommentarfeld; erweiterte Skalen via Simplesat-Integration (1–5 Sterne, NPS) | Binär nativ |
| **HubSpot Service Hub** | Fest 3 Optionen: Unhappy (0) / Neutral (1) / Happy (2) | Detractor / Passive / Promoter |
| **Intercom** | In-Chat Thumbs up/down (binary) + optionaler Follow-up; nach Bot-Resolution automatisch | Binary, mit optionalem Freitext |

**Fazit:** Der de-facto Standard ist **binary oder 3-stufig (negativ/neutral/positiv)**. Fünf Sterne kommen vor (Freshdesk), werden aber oft auf gut/schlecht reduziert. Die Erhebung läuft **immer per Post-Ticket-Survey**, ausgelöst durch Ticket-Close-Event.

### Was braucht ein CRM, damit CSAT eine echte Datenquelle wird?

Minimalanforderung für echte CSAT-Daten (kein Mock):

1. **CSAT-Modul**: Konfiguration (Ein/Aus, Delay, Kanal, Frage-Text, Skala)
2. **Trigger-Logik**: Ticket-Close-Event → Survey-Job (Hintergrund-Queue, nicht synchron)
3. **Zustellung**: E-Mail-Template mit eingebettetem One-Click-Rating-Link (tokenisiert, einmalig nutzbar)
4. **Response-Handler**: Endpoint, der Click entgegennimmt, Antwort validiert, Ticket zuordnet
5. **Aggregation**: CSAT-Score = (positive Ratings / gesamte beantwortete Surveys) × 100 %
6. **Datenspeicherung**: Pro Ticket: `csat_rating`, `csat_comment`, `csat_answered_at`

Das ist ~3–5 Tage Backend-Arbeit (Luke). Ohne das ist jede CSAT-Kachel reiner Mock.

---

## 2. Anpassbare Statistik-Dashboards – Marktmuster

### Zendesk Explore

- **Ansatz**: Vollständiger Report-Builder mit eigenem Canvas
- **Widgets (heißen dort "Components")**: Static (Reports, Text), Interactive (Filter-Controls), Live Data (Enterprise only)
- **Anordnung**: Drag-and-Drop auf Grid, Resize per Corner-Drag, Auto-Snap
- **Metrik-Katalog**: Keine explizite Katalog-UI — Nutzer wählt Dataset (= Datenquelle), dann Metrik + Attribut per Dropdown im Report-Editor
- **Prebuilt Dashboards**: Ja, Tickets/CSAT/Efficiency-Tabs vorgefertigt; Custom Dashboards ab Professional-Plan
- **Rollenzugriff**: Viewer (nur lesen) vs. Editor (bauen), Admins steuern Zugriff

### Freshdesk Analytics

- **Ansatz**: Widget-Gallery + Drag-and-Drop Canvas (non-technical-friendly)
- **Widget-Typen**: Bar Chart, Line Chart, Donut Chart, Summary/KPI-Kachel, Datentabelle
- **Metriken (Standard-Catalog-Inhalt)**:
  - Ticket Load: Received / Resolved / Reopened (wähle 2 von 3)
  - Time Trends: Avg Resolution Time / Avg First Response Time / Avg Response Time / Avg First Assign Time (wähle 2 von 4)
  - SLA Metrics: First Contact Resolution / Response SLA / Resolution SLA (wähle 2 von 3)
  - Custom Metrics: eigene Formeln mit vordefinierten Operatoren (Pro/Enterprise)
- **Anordnung**: Drag-and-Drop, Gallery für schnellen Start, Blank-Widget für Vollkontrolle
- **Rollenzugriff**: Pro/Enterprise Plan Pflicht; Agent sieht nur eigene Reports, Manager alle

### Zoho Desk Reports

- **Ansatz**: Formular-basierter Report-Builder (kein visuelles Drag-and-Drop nativ)
- **Report-Typen**: Tabellarisch, Summary, Matrix
- **Erweiterte Dashboards**: Via Zoho Analytics Integration (separates Produkt) — dort vollständiges Drag-and-Drop mit KPI-Widgets, Charts, Pivot-Views
- **Standard-Metriken nativ**: First Response Time, Avg Response Time, Avg Resolution Time, SLA Compliance, CSAT Rating, Agent Performance
- **Rollenzugriff**: Admin can create; Visibility: nur ich / alle Agents / bestimmte Agents / Department-spezifisch
- **Prebuilt Dashboards**: Ja, einige Standard-Dashboards (Ticket Overview, Agent Performance, CSAT)

### Gemeinsame Muster across alle Tools

1. **Widget-Katalog** (explizit oder implizit): Nutzer wählt aus einer festen Liste verfügbarer Metriken, keine freie Eingabe
2. **Drag-and-Drop** als Interaktionsmodell (Zendesk, Freshdesk, Zoho Analytics)
3. **Prebuilt Templates** als Einstieg, Custom Dashboards als Erweiterung
4. **Rollen-gesteuerter Zugriff** (Viewer/Editor/Admin)
5. **Keine Warnung bei fehlendem Daten-Feature**: Tools blenden Metriken einfach mit "–" oder "N/A" an, wenn keine Daten vorliegen. Nur Freshdesk filtert im Custom-Widget-Builder, welche Metriken für den gewählten Dataset verfügbar sind.

---

## 3. Standard-Metrik-Katalog Helpdesk (Empfehlung für Cosmi)

Kategorisiert nach Priorität und Datenquelle:

### Kategorie A: Sofort verfügbar (aus Ticket-Daten)

| Metrik | Typ | Formel/Quelle |
|---|---|---|
| Offene Tickets | KPI-Kachel | COUNT tickets WHERE status = open |
| Diese Woche gelöst | KPI-Kachel | COUNT tickets WHERE resolved_at in last 7d |
| Tickets pro Tag | Liniendiagramm | COUNT tickets GROUP BY date |
| Nach Status | Donut-Chart | COUNT tickets GROUP BY status |
| Nach Priorität | Balken-Chart | COUNT tickets GROUP BY priority |
| SLA-Einhaltung (%) | KPI-Kachel | (SLA-konforme Tickets / gesamt) × 100 |
| Durchschn. Antwortzeit | KPI-Kachel | AVG(first_response_at − created_at) |
| Durchschn. Lösungszeit | KPI-Kachel | AVG(resolved_at − created_at) |
| Wiederöffnungsrate | KPI-Kachel | COUNT reopened / COUNT resolved |
| Backlog-Trend | Liniendiagramm | offene Tickets kumuliert über Zeit |

### Kategorie B: Braucht CSAT-Feature (separates Backend-Modul)

| Metrik | Abhängigkeit |
|---|---|
| Kundenzufriedenheit (%) | CSAT-Modul: Ticket-Close-Survey + Response-Handler |
| CSAT-Trend | CSAT-Daten über Zeit |
| Zufriedenheit-Sterne | Nur wenn Freshdesk-Style (multi-point) gewählt, sonst binär |

### Kategorie C: Nice-to-have (Phase D+)

| Metrik | Hinweis |
|---|---|
| Nach Agent | Braucht User-Assignment + Datenschutz-Überlegung (DE/DSGVO) |
| Nach Kanal | Braucht multi-channel-Erfassung (E-Mail, Chat, Portal) |
| First Contact Resolution | Braucht Logik: 1 Antwort gelöst ohne Follow-up |
| NPS | Eigene Survey-Infrastruktur |

---

## 4. Empfehlungen für Cosmi

### (a) Statistik im Editor anpassbar — minimal-sinnvolle Umsetzung

**Empfohlener Ansatz: Widget-Toggle-Katalog (kein freier Canvas)**

Nicht Zendesk Explore (zu komplex für KMU-Admins), sondern ein **kurierter Katalog** mit Toggle-Mechanismus:

1. Admin öffnet Statistik-Seite im Editor
2. Sieht alle verfügbaren Metriken/Widgets als Liste mit Toggle (an/aus)
3. Reihenfolge per Drag-and-Drop auf der Statistik-Seite (optional, Phase 2)
4. Gespeichertes Layout gilt pro Tenant (nicht per User — zu komplex für KMU)

**Warum nicht freier Canvas?** KMU-Admins wollen keine Report-Builder-Lernkurve. Feste Widget-Typen (KPI-Kachel, Liniendiagramm, Donut, Balken) mit vordefiniertem Inhalt reichen. Das entspricht auch dem, was kleinere Freshdesk-Tier-Nutzer (Growth Plan) haben.

**Minimal-Spec:**
- 10–12 Widgets im Katalog (Kategorie A oben)
- Toggle-Steuerung im Editor
- Drag-and-Drop-Reihenfolge im Editor (optional in Phase 2)
- Sichtbarkeit: nur Admin-konfigurierbar, alle Nutzer sehen dasselbe

### (b) Kundenzufriedenheit ohne echte Datenquelle — Empfehlung

**Sofort:** Kachel im Editor als **"Braucht CSAT-Aktivierung"** markieren.

Konkret: Im Widget-Katalog erscheint die CSAT-Kachel mit einem `locked`-Status und einem erklärenden Hinweis: *"Kundenzufriedenheit erfordert das CSAT-Feature (automatische Umfrage nach Ticket-Schließung). Aktivierung unter Einstellungen > Helpdesk > CSAT."*

**Optionen für den Ist-Zustand (Entscheid nötig):**

| Option | Pro | Contra |
|---|---|---|
| **A: Kachel ausgeblendet bis CSAT aktiv** (empfohlen) | Kein Mock, keine Verwirrung | Kachel fehlt in Demo |
| B: Kachel sichtbar, zeigt "–" | Visuell komplett | Wirkt kaputt/leer |
| C: Demo-Mock mit Hinweis-Badge | Piloten sehen vollständige UI | Riskiert, dass Mock als echt wahrgenommen wird |

**Empfehlung Option A** für Produktion. Für Pilot-Demo temporär Option C mit gut sichtbarem "Demo-Daten"-Badge.

**CSAT als Feature:** Luke sollte den Ticket-Close-Survey-Flow als eigenständige Backend-Task nach der FE→BE-Wiring-Phase einplanen. Ist ~3–5 Tage Arbeit, aber ein deutlicher Differentiator gegenüber Konkurrenz-KMU-Tools, die CSAT oft nicht nativ haben.

---

## Quellen

- [Zendesk CSAT UX for email and messaging](https://support.zendesk.com/hc/en-us/articles/4408886173338)
- [Zendesk CSAT good vs bad rating guide 2026 (eesel AI)](https://www.eesel.ai/blog/zendesk-csat-good-vs-bad-rating)
- [Zendesk Adding and arranging dashboard components](https://support.zendesk.com/hc/en-us/articles/4408838017690)
- [Freshdesk CSAT module setup](https://support.freshdesk.com/support/solutions/articles/50000009790)
- [Freshdesk custom reports guide (eesel AI)](https://www.eesel.ai/blog/freshdesk-custom-reports)
- [Zoho Desk custom reports](https://help.zoho.com/portal/en/kb/desk/reports-and-dashboards/reports/articles/creating-custom-reports-in-zoho-desk)
- [HubSpot survey response properties](https://knowledge.hubspot.com/customer-feedback/survey-response-properties)
- [Tidio: 15 Essential Help Desk Metrics & KPIs](https://www.tidio.com/blog/helpdesk-metrics/)
- [Simplesat: Zoho Desk CSAT Surveys](https://www.simplesat.io/zoho-desk-csat-ces-nps-customer-surveys/)
- [Simplesat: Freshdesk CSAT Surveys](https://www.simplesat.io/freshdesk-csat-ces-nps-customer-surveys/)
