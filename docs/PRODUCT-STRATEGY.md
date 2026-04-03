# KMU Hub — Produkt-Strategie & Handoff

**Datum:** 2026-02-19 (aktualisiert nach Strategy Session)
**Von:** Darien (Design/Produkt)
**An:** Luke (Backend)

---

## Executive Summary

KMU Hub wird das **Betriebssystem fuer DACH-KMUs** — eine All-in-One-Plattform die
Microsoft 365, Teams, CRM, PM, Helpdesk, Zeiterfassung und HR ersetzt.
Nicht noch ein Tool, sondern die Klammer um alles.

### Kern-USPs
1. **All-in-One** — 1 Tool, 1 Login, 1 Rechnung
2. **EU-Datensouveraenitaet** — 100% EU/CH-Hosting, Self-hosted Option, kein US Cloud Act
3. **Massanfertigung** — 1-Woche-Onsite-Analyse, Branchenprofile
4. **Lokal-First** — Laeuft auf Firmenserver, Zugang von ueberall

---

## Build vs. Integrate Matrix

### BAUEN (in unserem Style)
- CRM (Kontakte, Deals, Pipeline) + Custom Fields, Firma als Entity, Duplikaterkennung
- **Kommunikation** (NEUES Modul) — Externe Nachrichten: E-Mail + Teams-Bridge + WhatsApp + Website-Widget + Kundenportal
- Chat (Team-intern) — Besteht, erweitern: Threads, Reactions, @mentions, Status/Praesenz
- Video/Meetings — LiveKit self-hosted + Zoom-Fallback fuer kleine KMUs
- Rechnungen & Finanzen (NICHT Buchhaltung!) — Belegkette, QR-Rechnung, ZUGFeRD, PDF, GoBD
- Office-Editing — Collabora Online eingebettet (WOPI, MPL 2.0)
- Wiki — TipTap Rich-Text-Editor
- Helpdesk — Canned Responses, Private Notes, Geschaeftszeiten
- Projekte — Gaeste-Zugang
- Alle Branchenmodule (Schichtplanung, Fuhrpark, Inventar, Einkauf, Rapporte, Vermietung, Formulare, Vertraege, Produktion)

### INTEGRIEREN (nie selbst bauen)
- Buchhaltung/FiBu → **DATEV-Export** + **Bexio-API**
- Lohnabrechnung → **NIEMALS / Anti-Feature** (26 Kantone CH, 16 Bundeslaender DE, staendig wechselnde Tarife — Haftungsrisiko zu hoch, Kunden nutzen DATEV/Bexio/Abacus)
- Newsletter → **Brevo/CleverReach** API
- E-Signatur → **Skribble** (CH, ZertES+eIDAS)
- Banking → **FinAPI** (4000+ Banken, PSD2)
- Office-Editor → **Collabora Online** (WOPI, MPL 2.0 — sauberer als AGPL)

### NIEMALS BAUEN
- Eigener Mailserver, Volles ERP, Kassensystem, PSTN-Telefonie, Recruiting, Sprint Planning

---

## WICHTIG: Chat vs. Kommunikation (2 separate Module!)

### Chat (Team-intern) — Besteht
- Kollegen-Chat, Channels, DMs
- Schnell, informell, Reactions
- Sidebar: "Chat"

### Kommunikation (Extern) — NEU
- ALLE externen Nachrichten: E-Mail, WhatsApp, Teams-Bridge, Widget, Portal
- Formell, CRM-verknuepft, nachverfolgbar
- Kontext-Panel mit CRM-Kontakt, Deals, Tickets
- Sidebar: "Kommunikation"

Verbindung: "An Kollege weiterleiten" (Kommunikation → Chat), "Ticket erstellen" (Kommunikation → Helpdesk)

---

## Top 20 Feature-Luecken (Priorisiert)

| # | Feature | Aufwand | Status |
|---|---------|---------|--------|
| 1 | IMAP/SMTP E-Mail-Backend | 6-8 Wo. | **FERTIG (Phase 10, 2026-02-17)** |
| 2 | DATEV-Export | 1-2 Wo. | Offen |
| 3 | Custom Fields (JSONB) | 3-4 Wo. | Offen |
| 4 | Belegkette (Angebot→Rechnung) | 3-4 Wo. | Offen |
| 5 | QR-Rechnung (CH Pflicht) | 1-2 Wo. | Offen |
| 6 | TipTap Rich-Text-Editor | 2-3 Wo. | Frontend-only |
| 7 | Firma als eigene Entity | 2-3 Wo. | Offen |
| 8 | MWSt multi-country (DE/AT/CH) | 2-3 Tage | Offen |
| 9 | PDF-Generierung | 1-2 Wo. | Offen |
| 10 | Akadem. Titel + Anrede | 2-3 Tage | Offen |
| 11 | Canned Responses + Private Notes | 3-5 Tage | Offen |
| 12 | Duplikaterkennung (CRM) | 1-2 Wo. | Offen |
| 13 | Externer Link-Share (Dateien) | 3-5 Tage | Offen |
| 14 | Collabora WOPI | ~1 Tag | Phase 11 WOPI-Endpoints kompatibel |
| 15 | Bexio-API | 2-4 Wo. | Offen |
| 16 | ZUGFeRD/XRechnung | 2-3 Wo. | Offen |
| 17 | Stunden→Rechnung Workflow | 1-2 Wo. | Offen |
| 18 | Gaeste-Zugang (Projekte) | 2-3 Wo. | Offen |
| 19 | Nextcloud WebDAV | 2-3 Wo. | Offen |
| 20 | GoBD-konforme Rechnungen | 1-2 Wo. | Offen |

---

## Neue Module/Features (nicht in Lukes Roadmap)

| Feature | Beschreibung | Backend-Aufwand |
|---------|-------------|-----------------|
| **Kommunikation (Unified Inbox)** | E-Mail + Teams + WhatsApp + Widget in einem externen Inbox-Modul | 6-8 Wo. |
| **Teams Bridge** | Microsoft Graph API, Bot Framework | 3-4 Wo. |
| **WhatsApp Business** | Meta Cloud API, Webhooks | 2-3 Wo. |
| **Website Chat-Widget** | WebSocket, embeddable JS, JWT Auth | 2-3 Wo. |
| **Kundenportal** | Eigener Auth-Flow, eingeschraenkte Ansicht | 3-4 Wo. |
| **Collabora Online WOPI** | WOPI Endpoints in Go, File Locking (bereits in Phase 11 gebaut) | ~1 Tag Anpassung |
| **Skribble E-Signatur** | REST API, Webhook | 2-3 Wo. |
| **Brevo Newsletter** | REST API, Contact Sync | 2-3 Wo. |
| **FinAPI Banking** | PSD2, Bank Connections | 3-4 Wo. |
| **Zoom Fallback** | OAuth2, Meeting Creation API | 1-2 Wo. |
| **KI-Features** | Zusammenfassungen, Entwuerfe, Suche | 4-6 Wo. |
| **Status/Praesenz** | WebSocket + Redis Heartbeat | 1 Wo. |

---

## Preismodell

> **VERALTET** — Dieses Dokument stammt von Feb 2026. Das aktuelle Preismodell (April 2026) ist ein
> Modul-x-User-System ohne feste Tiers. Siehe `.knowledge/pricing.md` fuer die kanonische Referenz.
>
> Kurzfassung: 23 Module (2-7 EUR/User/Mo), frei kombinierbar, Branchenpakete mit 15% Rabatt,
> Volumen-Rabatt ab 10 User, ORBIT (Self-Hosted) mit 20% Modulrabatt.

---

## Zielgruppen (Reihenfolge)

1. **Dienstleister/Agenturen** (85% Abdeckung) — ERSTE ZIELGRUPPE
2. **Handwerk** (80%) — Rapporte-Modul ist Differentiator
3. **Bau** (70%)
4. **Handel** (65%)
5. **Gastro** (55%)

---

## Infrastruktur

| Groesse | Setup | Kosten |
|---------|-------|--------|
| Klein (5-20 MA) | Docker auf NAS/Server, VPN fuer Remote | ~10-15 EUR/Mo Self-hosted |
| Mittel (20-100 MA) | Hetzner CX32, Redis, MinIO | ~30-50 EUR/Mo |
| Gross (100-200 MA) | Hetzner Cluster, LiveKit, Collabora | ~100-200 EUR/Mo |
| SaaS (100 Kunden) | Kubernetes, Blue-Green | ~200 EUR/Mo total |

---

## Detaillierte Dokumente

Alle Research-Dateien liegen lokal in `.planning/design/research/`.
Darien kann sie dir per USB/Fileshare geben:

| Datei | Inhalt | Zeilen |
|-------|--------|--------|
| 00-SYNTHESE.md | Gesamtsynthese, Build/Integrate Matrix | ~400 |
| 06-modul-gap-analyse.md | Gap-Analyse aller 23 Module | ~850 |
| 07-compliance-framework.md | DSGVO/DSG Compliance Framework | ~1740 |
| 08-datenbankmodelle.md | 30 neue PostgreSQL-Tabellen (SQL) | ~1600 |
| 09-infrastruktur-matrix.md | Server-Setups, Kosten, Sicherheit | ~1540 |
| 10-integrations-guide.md | 12 Integrationen mit Lizenzen/Setup | ~1500 |
| 11-backend-plan-part1.md | Backend: Core, DB, Sicherheit | ~800 |
| 11-backend-plan-part2.md | Backend: APIs, Integrationen, Sprints | ~800 |
| 12-kostenanalyse-preismodell.md | Kosten pro Modul, Preisstufen | ~1500 |
| 13-vision-ergaenzungen.md | Unified Inbox, Office, Teams-Ersatz | ~200 |
| 14-frontend-plan.md | Frontend-Implementierungsplan | ~??? |

---

## Waehrungs- und Markt-Strategie

> Entscheidung aus Strategy Session (2026-02-17)

- **Default-Waehrung:** EUR
- **Default-Locale:** `de-DE`
- **Default-MWSt:** 19% (Regelsatz), 7% (ermaessigt)
- **Spaeter per Config:** CHF (CH: 8.1%/2.6%/3.8%), AT (20%/10%/13%)
- **Beta-Launch:** Deutschland-first, nur EUR

---

## Backend-Stand (aktualisiert 2026-02-19)

| Phase | Inhalt | Status |
|-------|--------|--------|
| Phase 4-7 | CRM, Chat, Work, Calendar | FERTIG |
| Phase 8 | Video/Meetings (LiveKit) | FERTIG (2026-02-11) |
| Phase 9 | Security/Compliance (2FA, Audit) | FERTIG (2026-02-11) |
| Phase 10 | E-Mail (IMAP/SMTP, 7 Plans) | FERTIG (2026-02-17) |
| Phase 11 | Dokumente + WOPI (6 Plans) | FERTIG (2026-02-17) |
| **Phase 12** | **Rechnungen & Finanzen** | **Luke baut jetzt** |
| Phase 13 | HR | Geplant |
| Phase 14 | Unified Inbox / Kommunikation | Geplant |

**66 von 66 Plans (Phase 4-11) abgeschlossen.**

---

## Naechste Schritte

1. Luke baut Phase 12 (Rechnungen & Finanzen: Belegkette, GoBD, DATEV, QR-Rechnung)
2. Darien baut Frontend parallel (TipTap, Modul-Umbauten, neue Module)
3. [BACKEND-DEP] Items aus MASTER-PLAN auf Lukes Roadmap-Phasen mappen
4. TanStack Query Migration als Cross-Cutting-Concern einplanen (alle 25 Module)

**Backend-Reihenfolge (ab Phase 12):**
1. Phase 12: Rechnungen & Finanzen (Belegkette, QR-Rechnung, DATEV, ZUGFeRD, GoBD)
2. Phase 13: HR (Team-Erweiterungen)
3. Phase 14: Unified Inbox / Kommunikation (= Frontend "Kommunikation"-Modul)
4. Custom Fields + Firma Entity (CRM-Erweiterung)
5. Collabora WOPI Switch (~1 Tag, WOPI-Endpoints aus Phase 11 kompatibel)
6. Integrationen (Bexio, Skribble, Brevo, FinAPI)
