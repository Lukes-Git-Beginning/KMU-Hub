# Infrastruktur-Matrix: KMU Hub

**Datum:** 2026-02-17
**Confidence:** MEDIUM (Preise basieren auf Trainingsdaten Stand Mai 2025, muessen vor Einsatz verifiziert werden)
**Quellen:** Hetzner Cloud/Dedicated Preislisten, Exoscale Preislisten, Synthese-Dokument (00-SYNTHESE.md), Business-Profile (business-profiles.ts)

---

## Inhaltsverzeichnis

- [A) Setup nach Unternehmensgroesse](#a-setup-nach-unternehmensgroesse)
- [B) Setup nach Branche](#b-setup-nach-branche)
- [C) SaaS-Architektur](#c-saas-architektur)
- [D) Self-Hosted-Architektur](#d-self-hosted-architektur)
- [E) Hybrid-Deployments: OnlyOffice / LiveKit / MinIO](#e-hybrid-deployments-onlyoffice--livekit--minio)
- [F) Kosten-Kalkulation](#f-kosten-kalkulation)
- [G) Sicherheits-Architektur](#g-sicherheits-architektur)

---

## A) Setup nach Unternehmensgroesse

### Kontext

KMU Hub bedient Kunden von 5 bis 200 Mitarbeitern. Je nach Groesse unterscheiden sich die Anforderungen an Compute, Storage, Video-Bandbreite und Backup massiv. Die folgenden Empfehlungen gelten fuer **Self-Hosted Einzelkunden** und als Richtwert fuer die SaaS-Dimensionierung pro Mandant.

### Uebersichtstabelle

| Komponente | Klein (5-20 MA) | Mittel (20-100 MA) | Gross (100-200 MA) |
|---|---|---|---|
| **App-Server** | Hetzner CX22 (2 vCPU, 4 GB RAM, 40 GB SSD) | Hetzner CX32 (4 vCPU, 8 GB RAM, 80 GB SSD) | Hetzner CX42 (8 vCPU, 16 GB RAM, 160 GB SSD) |
| **DB-Server** | Shared auf App-Server (PostgreSQL 16) | Hetzner CX22 dediziert (4 GB RAM) | Hetzner CX32 dediziert (8 GB RAM) + Read-Replica |
| **Redis** | Shared auf App-Server (128 MB RAM) | Shared auf App-Server (256 MB RAM) | Hetzner CX11 dediziert (2 GB RAM) |
| **Object Storage (MinIO/S3)** | 50 GB (auf App-Server oder Hetzner Storage Box BX11) | 200 GB (Hetzner Storage Box BX21) | 1 TB (Hetzner Storage Box BX31) |
| **LiveKit (Video)** | Shared auf App-Server (max 5 gleichzeitige Teilnehmer) | Hetzner CX22 dediziert (max 25 gleichzeitige TN) | Hetzner CX32 dediziert (max 50+ gleichzeitige TN) |
| **OnlyOffice** | Nicht inkludiert (optional: +2 GB RAM min.) | Hetzner CX22 (4 GB RAM dediziert) | Hetzner CX32 (8 GB RAM dediziert) |
| **Backup** | Hetzner Storage Box BX11 (1 TB, taegliches Backup) | Hetzner Storage Box BX21 (5 TB, taeglich + stuendliches WAL) | Hetzner Storage Box BX31 (10 TB, taeglich + stuendlich + Georedundanz OVH) |
| **Monitoring** | Docker Health Checks + Uptime Kuma | Prometheus + Grafana (Lightweight Stack) | Prometheus + Grafana + Alertmanager + Loki |
| **Bandbreite** | ~20 TB/Mo inkl. | ~20 TB/Mo inkl. | ~20 TB/Mo inkl. (+ Video-Traffic beachten!) |
| **Geschaetzte Kosten/Mo** | **~15-25 EUR** | **~55-90 EUR** | **~140-230 EUR** |

### Detailbeschreibung pro Groesse

#### Klein (5-20 Mitarbeiter)

**Typisch:** Handwerksbetrieb, kleine Agentur, Arztpraxis, Reinigungsfirma.

- **All-in-One Server:** Ein einzelner Hetzner CX22 reicht fuer alles (Go-Microservices, PostgreSQL, Redis, MinIO). Go-Services sind extrem speichereffizient -- alle Microservices zusammen brauchen unter 500 MB RAM.
- **PostgreSQL:** Laeuft direkt auf dem App-Server. Bei 5-20 Usern mit ~10.000-50.000 Datensaetzen ist RAM kein Engpass. `shared_buffers = 1 GB`, `work_mem = 4 MB`.
- **Redis:** 128 MB genuegen fuer Session-Cache, Rate-Limiting und Pub/Sub bei 20 gleichzeitigen Usern.
- **LiveKit:** Optional auf dem gleichen Server. Maximal 2-3 gleichzeitige Videogespraeche realistisch. Fuer reine Audio-Calls reicht CX22.
- **Backup:** Hetzner Storage Box BX11 (1 TB, ~3,81 EUR/Mo). Taeglich Nacht-Backup per `pg_dump` + Datei-Rsync.
- **Monitoring:** Einfach: Docker Health Checks, systemd Watchdog, optionaler Uptime-Check via externes Tool.
- **Kein OnlyOffice:** Zu ressourcenintensiv fuer diese Groesse. Dokument-Bearbeitung ueber Desktop-App oder externen Editor.

**Hetzner-Kosten:**
| Posten | Server-Typ | EUR/Mo |
|--------|-----------|--------|
| App-Server (alles drauf) | CX22 | ~5,39 |
| Storage Box (Backup) | BX11 (1 TB) | ~3,81 |
| Floating IP | 1 IPv4 | ~4,51 |
| Snapshots (woechtl.) | ~40 GB | ~0,48 |
| **Gesamt** | | **~14-20 EUR** |

#### Mittel (20-100 Mitarbeiter)

**Typisch:** Mittelstaendischer Handwerker, IT-Dienstleister, Bauunternehmen, Handelsunternehmen.

- **App-Server:** CX32 (4 vCPU, 8 GB RAM). Go-Services + Reverse-Proxy (Caddy/Traefik). ~2-3 GB RAM fuer Applikation, Rest fuer OS und Headroom.
- **DB-Server:** Eigener CX22. PostgreSQL bekommt dedizierten RAM (`shared_buffers = 1,5 GB`). Bei 100 Usern und ~500.000 Datensaetzen ist das komfortabel.
- **Redis:** 256 MB auf dem App-Server. Caching von API-Responses, Session-Store, Background-Job-Queue.
- **LiveKit:** Eigener CX22. WebRTC braucht CPU fuer Transcoding bei >10 Teilnehmern. Dedizierter Server vermeidet Interferenz mit App-Performance.
- **OnlyOffice:** Falls gewuenscht, eigener CX22 (Minimum 4 GB RAM). OnlyOffice Document Server ist ein Java/Node-Monster und braucht dedizierten RAM.
- **Backup:** BX21 (5 TB, ~10,08 EUR/Mo). Taeglich `pg_dump`, stuendliches WAL-Archiving, Datei-Rsync. 30 Tage Retention.
- **Monitoring:** Prometheus + Grafana Stack als Docker-Container auf dem App-Server. CPU, RAM, Disk, Response-Times, Error-Rates.

**Hetzner-Kosten:**
| Posten | Server-Typ | EUR/Mo |
|--------|-----------|--------|
| App-Server | CX32 | ~9,59 |
| DB-Server | CX22 | ~5,39 |
| LiveKit-Server | CX22 | ~5,39 |
| OnlyOffice (optional) | CX22 | ~5,39 |
| Storage Box (Backup) | BX21 (5 TB) | ~10,08 |
| Floating IP | 1 IPv4 | ~4,51 |
| Object Storage | 200 GB | ~2,52 |
| **Gesamt (ohne OnlyOffice)** | | **~37-50 EUR** |
| **Gesamt (mit OnlyOffice)** | | **~43-60 EUR** |

#### Gross (100-200 Mitarbeiter)

**Typisch:** Grosses Bauunternehmen, Produktionsbetrieb, regionales Handelsunternehmen.

- **App-Server:** CX42 (8 vCPU, 16 GB RAM). Bei 200 gleichzeitigen Usern und 25+ Microservices braucht es Headroom. Alternativ: 2x CX32 hinter Load Balancer fuer HA.
- **DB-Server:** CX32 (8 GB RAM) als Primary + CX22 als Read-Replica. `shared_buffers = 3 GB`, Connection Pooling via PgBouncer (max_connections auf DB selbst limitiert auf 100, PgBouncer pooled 500+).
- **Redis:** Dedizierter CX11 (2 GB RAM). Bei 200 Usern koennen gleichzeitige Pub/Sub-Connections und Session-Daten den App-Server belasten.
- **LiveKit:** CX32 dediziert. Bandbreite ist der Engpass: 50 Teilnehmer in Video-Calls a 2 Mbit/s = 100 Mbit/s Upload. Hetzner's 20 TB inkludierten Traffic reichen.
- **OnlyOffice:** CX32 (8 GB RAM). Bei 20+ gleichzeitigen Dokumenten-Editoren braucht OnlyOffice mindestens 6-8 GB RAM.
- **Backup:** BX31 (10 TB, ~16,35 EUR/Mo) + Georedundanter Spiegel auf OVH (~15 EUR/Mo zusaetzlich). Stuendliches WAL-Archiving, taeglicher Full-Dump, 90 Tage Retention.
- **Monitoring:** Voller Observability-Stack: Prometheus (Metriken), Loki (Logs), Grafana (Dashboards), Alertmanager (Slack/E-Mail-Alerts). Eigener CX11 oder als Container auf App-Server.

**Hetzner-Kosten:**
| Posten | Server-Typ | EUR/Mo |
|--------|-----------|--------|
| App-Server (oder 2x CX32 HA) | CX42 | ~17,99 |
| DB Primary | CX32 | ~9,59 |
| DB Read-Replica | CX22 | ~5,39 |
| Redis dediziert | CX11 | ~3,79 |
| LiveKit-Server | CX32 | ~9,59 |
| OnlyOffice | CX32 | ~9,59 |
| Storage Box (Backup) | BX31 (10 TB) | ~16,35 |
| OVH Georedundanz | Backup | ~15,00 |
| Object Storage | 1 TB | ~12,60 |
| Floating IP | 2 IPv4 | ~9,02 |
| Load Balancer | LB11 | ~5,39 |
| **Gesamt** | | **~114-175 EUR** |

### Schweiz-Cluster (Exoscale)

Fuer Schweizer Kunden die auf Schweizer Datenresidenz bestehen:

| Komponente | Exoscale Equivalent | Preis-Delta vs. Hetzner |
|---|---|---|
| Compute (Standard) | standard.medium (2 vCPU, 4 GB) | ~2-3x teurer |
| Compute (Gross) | standard.extra-large (8 vCPU, 32 GB) | ~2-3x teurer |
| Managed DB (PostgreSQL) | DBaaS PostgreSQL Hobbyist/Startup | ~3-5x teurer |
| Object Storage (S3) | SOS (S3-kompatibel) | ~0,022 EUR/GB/Mo |
| Standort | Zuerich (ch-dk-2), Genf (ch-gva-2) | -- |

**Empfehlung:** Exoscale nur fuer CH-Kunden die explizit Schweizer Datenresidenz verlangen. Ansonsten Hetzner (DE) -- DSGVO-konform, massiv guenstiger, und DE<->CH Datenaustausch ist durch gegenseitige Angemessenheitsbeschluesse unproblematisch.

**Typische Exoscale-Kosten (Mittel-Setup):**
| Posten | Exoscale-Typ | EUR/Mo |
|--------|-------------|--------|
| App-Server | standard.medium | ~40-55 |
| DB (Managed PostgreSQL) | DBaaS Startup-4 | ~75-100 |
| Object Storage | 200 GB SOS | ~5 |
| Backup | S3-basiert, 1 TB | ~22 |
| **Gesamt** | | **~140-185 EUR** |

---

## B) Setup nach Branche

### B1) Dienstleister / Agentur / Beratung

**Profil:** `dienstleistung` (+ `it_tech` fuer IT-Firmen)
**Typische Groesse:** 5-30 MA
**Abdeckung:** ~85%

**Aktive Module:**
- Kern: CRM, Kalender, Chat, Meetings (LiveKit), Dokumente, Mail, Zeiterfassung, Vertraege, Finance
- Optional: Projekte (Agenturen), Helpdesk (IT-Support), Berichte, Formulare, Rapporte

**Integrationen:**
| Integration | Prioritaet | Begruendung |
|---|---|---|
| IMAP/SMTP (E-Mail) | KRITISCH | Kundenkommunikation laeuft ueber Mail |
| Bexio / DATEV-Export | KRITISCH | Buchhalter-Schnittstelle |
| Brevo / CleverReach | HOCH | Newsletter an Kunden |
| Skribble (E-Signatur) | MITTEL | Vertraege digital signieren |
| OnlyOffice | HOCH | Angebote/Vertraege inline bearbeiten |

**Storage-Bedarf:**
- Gering bis mittel. Hauptsaechlich Dokumente (Vertraege, Angebote, PDFs).
- 5-10 GB/Jahr fuer 10 MA, 20-50 GB/Jahr fuer 30 MA.
- E-Mail-Attachments koennen schnell wachsen: +20-50 GB/Jahr bei aktivem E-Mail-Verkehr.
- Video-Aufzeichnungen (optional): +5-20 GB/Mo bei regelmaessigen Meetings.

**Empfohlenes Setup:**
- Klein (5-15): CX22 All-in-One (~15 EUR/Mo)
- Mittel (15-30): CX32 + CX22 DB (~25 EUR/Mo)

---

### B2) Handwerk

**Profil:** `handwerk`
**Typische Groesse:** 5-20 MA
**Abdeckung:** ~80%

**Aktive Module:**
- Kern: CRM, Projekte, Kalender, Einkauf, Inventar, Fuhrpark, Dokumente, Team, Finance, Rapporte, Zeiterfassung
- Optional: Schichten, Chat, Meetings, Vertraege, Vermietung (Geraete), Formulare

**Integrationen:**
| Integration | Prioritaet | Begruendung |
|---|---|---|
| DATEV-Export | KRITISCH | Steuerberater braucht Daten |
| QR-Rechnung (CH) | KRITISCH (CH) | Rechnungsversand Pflicht |
| PDF-Generierung | KRITISCH | Angebote/Rechnungen an Kunden |
| Belegkette | KRITISCH | Angebot -> Auftrag -> Rechnung |
| ZUGFeRD/XRechnung | MITTEL | Ab 2027 Pflicht (DE B2B) |

**Storage-Bedarf:**
- Mittel. Rapporte mit Fotos sind der groesste Treiber.
- Fotos: 5-15 MB/Stueck. Ein Rapport hat 3-10 Fotos. Bei 5 Rapports/Woche = ~2-5 GB/Mo.
- Dokumente (Angebote, Rechnungen, Lieferscheine): ~5 GB/Jahr.
- Fuhrpark (Wartungsbelege, Schadenfotos): ~2-3 GB/Jahr.
- **Total: ~30-80 GB/Jahr fuer 15 MA.**

**Empfohlenes Setup:**
- Klein (5-15): CX22 All-in-One + BX11 Storage Box (~18 EUR/Mo)
- Kein LiveKit noetig (Handwerker telefonieren, keine Video-Meetings)
- Kein OnlyOffice noetig

---

### B3) Bau

**Profil:** `bau`
**Typische Groesse:** 10-50 MA
**Abdeckung:** ~70%

**Aktive Module:**
- Kern: Projekte, Inventar, Einkauf, Fuhrpark, Team, Schichten, Finance, Kalender, Rapporte, Zeiterfassung
- Optional: CRM, Dokumente, Chat, Berichte, Vermietung (Maschinen/Container), Formulare

**Integrationen:**
| Integration | Prioritaet | Begruendung |
|---|---|---|
| DATEV-Export | KRITISCH | Steuerberater |
| PDF-Generierung | KRITISCH | Aufmasse, Rapporte, Rechnungen |
| Belegkette | HOCH | Material-Bestellungen -> Rechnungen |
| Nextcloud (WebDAV) | MITTEL | Planzeichnungen, grosse Dateien |
| ZUGFeRD/XRechnung | MITTEL | B2B-Rechnungen an Auftraggeber |

**Storage-Bedarf:**
- HOCH. Bauunternehmen produzieren enorm viele Fotos, Plaene und Dokumente.
- Rapporte mit Fotodokumentation: 10-30 Fotos/Rapport, 5-15 MB/Foto. Bei 10 Rapports/Woche = **5-20 GB/Mo**.
- Planzeichnungen (PDF/DWG): 50-200 MB/Stueck. Bei 20 Projekten/Jahr = ~10-20 GB/Jahr.
- Aufmasse, Lieferscheine, Belege: ~10 GB/Jahr.
- **Total: ~100-300 GB/Jahr fuer 30 MA.**

**Empfohlenes Setup:**
- Mittel (10-30): CX32 + CX22 DB + BX21 Storage (~40 EUR/Mo)
- Gross (30-50): CX42 + CX32 DB + BX31 Storage (~80 EUR/Mo)
- Kein LiveKit/OnlyOffice noetig (Baubranche = Telefon + vor Ort)

---

### B4) Handel (Einzelhandel)

**Profil:** `einzelhandel`
**Typische Groesse:** 5-50 MA
**Abdeckung:** ~65%

**Aktive Module:**
- Kern: Inventar, CRM, Einkauf, Schichten, Team, Finance, Zeiterfassung
- Optional: Chat, Berichte, Meetings, Vertraege, Vermietung, Formulare

**Integrationen:**
| Integration | Prioritaet | Begruendung |
|---|---|---|
| DATEV-Export | KRITISCH | Steuerberater |
| Bexio (CH) | HOCH (CH) | Buchhaltung |
| FinAPI (Banking) | MITTEL | Automatischer Bankabgleich |
| E-Commerce API (spaeter) | HOCH (v2+) | Shopify/WooCommerce-Anbindung |
| Barcode/EAN-Lookup | MITTEL | Artikelstammdaten automatisch |

**Storage-Bedarf:**
- Mittel. Hauptsaechlich Artikelbilder und Belege.
- Artikelbilder: 500 KB - 3 MB/Stueck. Bei 1.000 Artikeln = ~1-3 GB.
- Rechnungen/Lieferscheine: ~5 GB/Jahr.
- **Total: ~10-30 GB/Jahr fuer 20 MA.**

**Empfohlenes Setup:**
- Klein (5-15): CX22 All-in-One (~15 EUR/Mo)
- Mittel (15-50): CX32 + CX22 DB (~25 EUR/Mo)

---

### B5) Gastronomie / Hotellerie

**Profil:** `gastronomie`
**Typische Groesse:** 10-50 MA
**Abdeckung:** ~55% (schwaechste Branche)

**Aktive Module:**
- Kern: Inventar (mit Ablaufdaten), Schichten, Einkauf, Team, Finance, Kalender, Zeiterfassung
- Optional: CRM, Chat, Berichte, Vertraege, Formulare

**Integrationen:**
| Integration | Prioritaet | Begruendung |
|---|---|---|
| DATEV-Export | KRITISCH | Steuerberater |
| Bexio (CH) | HOCH (CH) | Buchhaltung |
| Kassensystem-API (spaeter) | HOCH (v2+) | Anbindung an bestehendes POS |
| Reservierungssystem (spaeter) | HOCH (v2+) | Tisch-/Zimmer-Reservierung |

**Storage-Bedarf:**
- Gering. Wenig Dokumente, hauptsaechlich Belege und Lieferscheine.
- **Total: ~5-15 GB/Jahr.**

**Empfohlenes Setup:**
- Klein (10-25): CX22 All-in-One (~15 EUR/Mo)
- Mittel (25-50): CX32 (~10 EUR/Mo) + Schichtplanung ist CPU-leicht

**Anmerkung:** Gastro/Hotel ist ohne Kasse und Tischreservierung nicht vollstaendig bedienbar. KMU Hub wird hier primaer als **Backoffice-Tool** (HR, Schichten, Einkauf, Finanzen) positioniert, nicht als Komplett-Loesung. Integration mit bestehendem POS ist der Weg.

---

### Branchen-Storage-Vergleich (pro Jahr, 15 MA Durchschnitt)

```
Bau:            ████████████████████████████  100-300 GB
Handwerk:       ██████████████                 30-80 GB
Dienstleister:  ████████████                   25-60 GB
Handel:         ██████                         10-30 GB
Gastro:         ███                             5-15 GB
```

---

## C) SaaS-Architektur

### C1) Multi-Tenant-Strategie: Row-Level Security (RLS)

**Entscheidung: Row-Level Security (RLS) statt Schema-per-Tenant.**

| Kriterium | Schema-per-Tenant | Row-Level Security |
|---|---|---|
| Isolation | Stark (physische Trennung) | Mittel (logische Trennung) |
| Migrations | Muessen pro Schema laufen (langsam bei 500+ Tenants) | Einmal pro Tabelle |
| Connection Pooling | Schwierig (pro Schema ein Pool) | Einfach (ein Pool, `SET app.tenant_id`) |
| Skalierung | 100-200 Tenants max. pro DB | 10.000+ Tenants pro DB |
| Backup/Restore einzelner Tenants | Einfach (Schema dump) | Schwierig (WHERE-Filter-Export) |
| Compliance (DSGVO-Loeschung) | Einfach (DROP SCHEMA) | Machbar (DELETE WHERE tenant_id) |
| Performance bei vielen Tenants | Abnehmend (pg_catalog Bloat) | Stabil (Index auf tenant_id) |

**Begruendung RLS:**
- KMU Hub zielt auf 500-5.000+ Kunden. Schema-per-Tenant skaliert schlecht ab ~200 Schemas.
- Migrations sind bei Schema-per-Tenant ein Alptraum: `ALTER TABLE` muss pro Schema laufen.
- Connection-Pooling (PgBouncer) funktioniert trivial mit RLS (`SET app.tenant_id = '...'` pro Connection).
- Fuer Enterprise-Kunden (>100 MA) die staerkere Isolation verlangen: Eigene DB-Instanz als Upsell.

**Implementation:**

```sql
-- Jede Tabelle hat tenant_id
ALTER TABLE contacts ADD COLUMN tenant_id UUID NOT NULL REFERENCES tenants(id);
CREATE INDEX idx_contacts_tenant_id ON contacts(tenant_id);

-- RLS Policy
ALTER TABLE contacts ENABLE ROW LEVEL SECURITY;
CREATE POLICY contacts_tenant_isolation ON contacts
  USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- In Go: Vor jedem Request
db.Exec("SET app.tenant_id = $1", tenantID)
```

**Partitionierung fuer grosse Tabellen:**

```sql
-- Tabellen mit >10 Mio Rows partitionieren
CREATE TABLE activities (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    ...
) PARTITION BY HASH (tenant_id);

-- 16 Partitionen (erweiterbar)
CREATE TABLE activities_p0 PARTITION OF activities FOR VALUES WITH (MODULUS 16, REMAINDER 0);
CREATE TABLE activities_p1 PARTITION OF activities FOR VALUES WITH (MODULUS 16, REMAINDER 1);
-- ...
```

### C2) Kubernetes-Setup auf Hetzner Cloud

**Warum Kubernetes statt Docker Compose:**
- Auto-Scaling (horizontal) fuer Load-Spitzen
- Rolling Updates (Zero-Downtime)
- Self-Healing (Pod-Restart bei Crash)
- Secrets Management (K8s Secrets)
- Ingress Controller fuer Multi-Tenant-Routing

**Cluster-Layout (Start):**

```
Hetzner Cloud Kubernetes (hcloud-k8s oder k3s auf Hetzner VMs)

┌─────────────────────────────────────────────────┐
│  Control Plane (Managed oder 3x CX22)           │
├─────────────────────────────────────────────────┤
│                                                  │
│  Worker Pool "app" (auto-scale 2-8 Nodes)       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐        │
│  │ CX32     │ │ CX32     │ │ CX32     │ ...    │
│  │ API GW   │ │ CRM Svc  │ │ Chat Svc │        │
│  │ Auth Svc │ │ PM Svc   │ │ Mail Svc │        │
│  └──────────┘ └──────────┘ └──────────┘        │
│                                                  │
│  Worker Pool "data" (fixed 2-3 Nodes)           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐        │
│  │ CX42     │ │ CX32     │ │ CX22     │        │
│  │ PG Primary│ │ PG Replica│ │ Redis    │       │
│  └──────────┘ └──────────┘ └──────────┘        │
│                                                  │
│  Worker Pool "media" (auto-scale 1-4 Nodes)     │
│  ┌──────────┐ ┌──────────┐                      │
│  │ CX32     │ │ CX32     │                      │
│  │ LiveKit  │ │ OnlyOff. │                      │
│  └──────────┘ └──────────┘                      │
│                                                  │
│  Ingress: Hetzner LB + Traefik/NGINX Ingress    │
│  Storage: Hetzner Volumes (CSI) + S3 (MinIO)    │
└─────────────────────────────────────────────────┘
```

**Hetzner Kubernetes Option:**
- **hcloud-k8s:** Managed Kubernetes. Control Plane kostenfrei, Worker-Nodes werden als normale Cloud-Server abgerechnet.
- **Alternativ k3s:** Lightweight Kubernetes auf Hetzner VMs. Guenstiger, aber mehr Wartungsaufwand.

**Empfehlung:** Managed hcloud-k8s fuer weniger Ops-Aufwand. Control-Plane-Kosten = 0 EUR (nur Worker-Nodes zahlen).

### C3) Load Balancing

```
                    ┌─────────────────┐
                    │   Cloudflare    │  (DDoS, WAF, CDN)
                    │   oder          │
                    │   Hetzner LB    │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Hetzner LB11   │  (~5,39 EUR/Mo)
                    │  (L4 TCP/UDP)   │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼──┐  ┌───────▼──┐  ┌───────▼──┐
     │ Traefik   │  │ Traefik  │  │ Traefik  │
     │ Ingress   │  │ Ingress  │  │ Ingress  │
     │ (Pod)     │  │ (Pod)    │  │ (Pod)    │
     └────────┬──┘  └──────┬──┘  └──────┬──┘
              │            │            │
              └────────────┼────────────┘
                           │
              Kubernetes Service Mesh
              (Service Discovery + Routing)
```

- **Hetzner Load Balancer LB11:** TCP/HTTP Load Balancing, Health Checks, ~5,39 EUR/Mo.
- **Traefik Ingress Controller:** Automatische TLS-Zertifikate (Let's Encrypt), Routing nach Hostname/Path, Rate Limiting.
- **Sticky Sessions:** Fuer WebSocket-Verbindungen (Chat, LiveKit) via Traefik Cookie-Affinity.

### C4) Auto-Scaling

```yaml
# Horizontal Pod Autoscaler (HPA)
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 2
  maxReplicas: 8
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

**Cluster Autoscaler:** Hetzner Cloud bietet einen Cluster-Autoscaler der automatisch neue Worker-Nodes provisioniert wenn Pods nicht schedulebar sind.

**Scaling-Strategie pro Service:**

| Service | Min Replicas | Max Replicas | Scaling-Trigger |
|---|---|---|---|
| API Gateway | 2 | 8 | CPU > 70% |
| CRM Service | 2 | 6 | CPU > 70% |
| Chat Service | 2 | 4 | WebSocket-Connections > 500 |
| Mail Service | 1 | 4 | Queue-Laenge > 100 |
| Auth Service | 2 | 4 | CPU > 60% (latenz-kritisch) |
| LiveKit | 1 | 4 | Participant Count |
| PostgreSQL | 1 Primary + 1-2 Replicas | Fix | Manuell |
| Redis | 1 | 1 (Sentinel fuer HA) | Fix |

### C5) CDN fuer statische Assets

**Empfehlung: Hetzner Object Storage + Cloudflare CDN (Free Tier)**

- **Statische Assets:** Electron-App wird lokal ausgefuehrt, braucht kein CDN. CDN ist nur fuer:
  - Web-basierte Login-/Onboarding-Seiten
  - Shared Files (externer Link-Share)
  - Marketing-Website
  - API-Dokumentation

- **Hetzner Object Storage:** S3-kompatibel, ~0,0126 EUR/GB/Mo. Fuer Mandanten-Uploads (Avatare, Logos, hochgeladene Dateien).

- **Cloudflare (Free Tier):** DNS, DDoS-Schutz, automatisches HTTPS, Caching. Kein EU-spezifisches Problem da nur Caching (Originaldaten bleiben auf Hetzner).

**Achtung DSGVO:** Cloudflare ist ein US-Unternehmen. Fuer rein europaeische Loesung: **BunnyCDN** (slowenisch, EU-only PoPs verfuegbar) oder Hetzner-eigenes CDN (noch nicht verfuegbar, Stand 2025).

### C6) Blue-Green Deployment

```
Aktiv (Blue)                    Standby (Green)
┌──────────────┐               ┌──────────────┐
│  v2.3.1      │               │  v2.4.0      │
│  API GW      │               │  API GW      │
│  CRM Svc     │               │  CRM Svc     │
│  Chat Svc    │               │  Chat Svc    │
│  ...         │               │  ...         │
└──────┬───────┘               └──────┬───────┘
       │                              │
       │  ◄── Traffic                 │  ◄── Kein Traffic
       │                              │
┌──────▼──────────────────────────────▼───────┐
│           Shared PostgreSQL + Redis          │
│           (Migrations VORHER ausfuehren)     │
└─────────────────────────────────────────────┘
```

**Ablauf:**
1. Green-Environment deployen (neue Version)
2. DB-Migrations ausfuehren (muessen abwaertskompatibel sein!)
3. Health Checks auf Green ausfuehren
4. Smoke Tests (automatisiert)
5. Traffic von Blue auf Green umschalten (Ingress-Update, ~0 Downtime)
6. Blue als Rollback bereithalten (30 Min)
7. Blue herunterfahren

**Kritisch:** DB-Migrations muessen **abwaertskompatibel** sein. Neue Spalten mit Defaults, nie Spalten loeschen im gleichen Release. Loeschung erst im Release danach (wenn Blue nicht mehr laeuft).

**In Kubernetes:**

```yaml
# Deployment mit Label-Selector Switch
apiVersion: v1
kind: Service
metadata:
  name: api-gateway
spec:
  selector:
    app: api-gateway
    version: green    # <-- Umschalten von "blue" auf "green"
  ports:
  - port: 8080
```

---

## D) Self-Hosted-Architektur

### D1) Docker Compose Setup (Einfach)

**Zielgruppe:** KMUs die volle Datenkontrolle wollen, eigene IT haben oder einen IT-Partner beauftragen.

**Minimum Hardware:**

| Groesse | CPU | RAM | Disk | Betriebssystem |
|---|---|---|---|---|
| Klein (5-20 MA) | 2 Cores | 4 GB | 60 GB SSD | Ubuntu 22.04 LTS / Debian 12 |
| Mittel (20-100 MA) | 4 Cores | 8 GB | 120 GB SSD + 500 GB HDD (Storage) | Ubuntu 22.04 LTS |
| Gross (100-200 MA) | 8 Cores | 16 GB | 250 GB SSD + 2 TB HDD (Storage) | Ubuntu 22.04 LTS |

**Docker Compose (Basis-Setup):**

```yaml
# docker-compose.yml
version: '3.8'

services:
  # === Core Services ===
  gateway:
    image: ghcr.io/kmuhub/gateway:${VERSION:-latest}
    restart: unless-stopped
    ports:
      - "443:8443"
      - "80:8080"
    environment:
      - DATABASE_URL=postgres://kmuhub:${DB_PASSWORD}@postgres:5432/kmuhub
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
      - LIVEKIT_URL=ws://livekit:7880
      - LIVEKIT_API_KEY=${LIVEKIT_API_KEY}
      - LIVEKIT_API_SECRET=${LIVEKIT_API_SECRET}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      - ./certs:/etc/kmuhub/certs:ro
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3

  crm:
    image: ghcr.io/kmuhub/crm:${VERSION:-latest}
    restart: unless-stopped
    environment:
      - DATABASE_URL=postgres://kmuhub:${DB_PASSWORD}@postgres:5432/kmuhub
      - REDIS_URL=redis://redis:6379
    depends_on:
      - postgres
      - redis

  chat:
    image: ghcr.io/kmuhub/chat:${VERSION:-latest}
    restart: unless-stopped
    environment:
      - DATABASE_URL=postgres://kmuhub:${DB_PASSWORD}@postgres:5432/kmuhub
      - REDIS_URL=redis://redis:6379
    depends_on:
      - postgres
      - redis

  auth:
    image: ghcr.io/kmuhub/auth:${VERSION:-latest}
    restart: unless-stopped
    environment:
      - DATABASE_URL=postgres://kmuhub:${DB_PASSWORD}@postgres:5432/kmuhub
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - postgres
      - redis

  # === Data Layer ===
  postgres:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: kmuhub
      POSTGRES_USER: kmuhub
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./backups/wal:/var/lib/postgresql/wal_archive
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kmuhub"]
      interval: 10s
      timeout: 5s
      retries: 5
    command: >
      postgres
        -c shared_buffers=1GB
        -c work_mem=4MB
        -c maintenance_work_mem=256MB
        -c effective_cache_size=3GB
        -c wal_level=replica
        -c archive_mode=on
        -c archive_command='cp %p /var/lib/postgresql/wal_archive/%f'
        -c max_connections=100

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redisdata:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # === Object Storage ===
  minio:
    image: minio/minio:latest
    restart: unless-stopped
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - miniodata:/data
    ports:
      - "9001:9001"  # Console (nur intern)

  # === Media (Optional) ===
  livekit:
    image: livekit/livekit-server:latest
    restart: unless-stopped
    ports:
      - "7880:7880"   # HTTP
      - "7881:7881"   # WebSocket
      - "7882:7882/udp"  # WebRTC/UDP
    volumes:
      - ./config/livekit.yaml:/etc/livekit.yaml:ro
    command: --config /etc/livekit.yaml

  # onlyoffice:  # Optional, nur bei Bedarf aktivieren
  #   image: onlyoffice/documentserver:latest
  #   restart: unless-stopped
  #   environment:
  #     - JWT_ENABLED=true
  #     - JWT_SECRET=${ONLYOFFICE_JWT_SECRET}
  #   volumes:
  #     - onlyoffice_data:/var/www/onlyoffice/Data
  #   ports:
  #     - "8443:443"

  # === Monitoring (Optional) ===
  uptime-kuma:
    image: louislam/uptime-kuma:1
    restart: unless-stopped
    volumes:
      - uptime-kuma:/app/data
    ports:
      - "3001:3001"

volumes:
  pgdata:
  redisdata:
  miniodata:
  uptime-kuma:
  # onlyoffice_data:
```

### D2) Backup-Automation

**Backup-Skript (`/opt/kmuhub/backup.sh`):**

```bash
#!/bin/bash
set -euo pipefail

# Konfiguration
BACKUP_DIR="/opt/kmuhub/backups"
RETENTION_DAYS=30
DATE=$(date +%Y%m%d_%H%M%S)
DB_CONTAINER="kmuhub-postgres-1"

# 1. PostgreSQL Full Dump
echo "[$(date)] Starting PostgreSQL backup..."
docker exec ${DB_CONTAINER} pg_dump -U kmuhub -Fc kmuhub \
  > "${BACKUP_DIR}/db/kmuhub_${DATE}.dump"

# 2. MinIO/Files Backup (inkrementell via rsync)
echo "[$(date)] Starting file backup..."
rsync -a --delete \
  /var/lib/docker/volumes/kmuhub_miniodata/_data/ \
  "${BACKUP_DIR}/files/"

# 3. Komprimieren und verschluesseln (AES-256)
echo "[$(date)] Encrypting backup..."
tar czf - "${BACKUP_DIR}/db/kmuhub_${DATE}.dump" \
  | openssl enc -aes-256-cbc -pbkdf2 -pass file:/opt/kmuhub/.backup-key \
  > "${BACKUP_DIR}/encrypted/kmuhub_${DATE}.tar.gz.enc"

# 4. Alte Backups aufraeumen
find "${BACKUP_DIR}/db/" -name "*.dump" -mtime +${RETENTION_DAYS} -delete
find "${BACKUP_DIR}/encrypted/" -name "*.enc" -mtime +${RETENTION_DAYS} -delete

# 5. Optional: Zu Remote-Storage kopieren (Hetzner Storage Box)
# rsync -avz "${BACKUP_DIR}/encrypted/" \
#   u123456@u123456.your-storagebox.de:backups/

echo "[$(date)] Backup completed: kmuhub_${DATE}"
```

**Cron-Job:**
```cron
# Taeglich um 02:00 Uhr
0 2 * * * /opt/kmuhub/backup.sh >> /var/log/kmuhub-backup.log 2>&1

# WAL-Archivierung: wird durch PostgreSQL-Config automatisch erledigt
```

**Restore-Prozedur:**
```bash
# 1. Entschluesseln
openssl enc -d -aes-256-cbc -pbkdf2 -pass file:/opt/kmuhub/.backup-key \
  < kmuhub_20260217_020000.tar.gz.enc | tar xzf -

# 2. Restore
docker exec -i kmuhub-postgres-1 pg_restore -U kmuhub -d kmuhub --clean \
  < kmuhub_20260217_020000.dump
```

### D3) Update-Mechanismus

**Automatisches Update-Skript (`/opt/kmuhub/update.sh`):**

```bash
#!/bin/bash
set -euo pipefail

NEW_VERSION=$1
COMPOSE_DIR="/opt/kmuhub"

echo "[$(date)] Updating KMU Hub to version ${NEW_VERSION}..."

# 1. Backup VOR Update (KRITISCH -- siehe Deployment-Regeln in CLAUDE.md)
/opt/kmuhub/backup.sh

# 2. Images pullen
cd ${COMPOSE_DIR}
VERSION=${NEW_VERSION} docker compose pull

# 3. Migrations ausfuehren (vor Service-Restart)
VERSION=${NEW_VERSION} docker compose run --rm gateway migrate-up

# 4. Rolling Restart (minimale Downtime)
VERSION=${NEW_VERSION} docker compose up -d --remove-orphans

# 5. Health Check
echo "Waiting for health check..."
for i in {1..30}; do
  if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    echo "[$(date)] Update to ${NEW_VERSION} successful!"
    exit 0
  fi
  sleep 2
done

# 6. Rollback bei Fehler
echo "[$(date)] ERROR: Health check failed. Rolling back..."
VERSION=${OLD_VERSION} docker compose up -d
echo "[$(date)] Rolled back to ${OLD_VERSION}"
exit 1
```

**Versionierung:** Semantic Versioning via Docker Image Tags (`ghcr.io/kmuhub/gateway:2.4.0`). `.env` Datei enthaelt `VERSION=2.4.0`.

### D4) Monitoring

**Empfehlung nach Groesse:**

| Groesse | Monitoring-Stack | Aufwand | Kosten |
|---|---|---|---|
| Klein (5-20 MA) | **Uptime Kuma** (Docker) | Minimal (10 Min Setup) | 0 EUR |
| Mittel (20-100 MA) | **Prometheus + Grafana** (Docker Compose) | Mittel (2-4h Setup) | 0 EUR (Self-Hosted) |
| Gross (100-200 MA) | **Prometheus + Grafana + Loki + Alertmanager** | Hoch (1 Tag Setup) | 0 EUR (Self-Hosted) |

**Uptime Kuma (Klein):**
- HTTP-Endpoint-Monitoring (Gateway /health, alle Service-Endpoints)
- TCP-Port-Check (PostgreSQL 5432, Redis 6379)
- E-Mail/Telegram-Alerts bei Ausfall
- Dashboard fuer Uptime-Historie

**Prometheus + Grafana (Mittel/Gross):**

```yaml
# docker-compose.monitoring.yml (optionales Overlay)
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./config/prometheus.yml:/etc/prometheus/prometheus.yml
      - promdata:/prometheus
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    volumes:
      - grafanadata:/var/lib/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}

  loki:
    image: grafana/loki:latest
    volumes:
      - lokidata:/loki
    ports:
      - "3100:3100"

  node-exporter:
    image: prom/node-exporter:latest
    pid: host
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro

volumes:
  promdata:
  grafanadata:
  lokidata:
```

**KMU Hub Go-Services exportieren Metriken:**
- `/metrics` Endpoint (Prometheus-Format) in jedem Service
- Request-Latenz (Histogram), Error-Rate, Active-Connections
- DB-Connection-Pool-Auslastung
- Redis-Hit/Miss-Rate
- Queue-Laenge (Background-Jobs)

**Wichtige Grafana-Dashboards:**
1. **System:** CPU, RAM, Disk, Network (Node Exporter)
2. **Application:** Request-Rate, Latenz P50/P95/P99, Error-Rate
3. **Database:** Connections, Query-Zeit, Tabellengroessen
4. **Business:** Aktive User, API-Calls/Tag, Storage-Nutzung pro Mandant

---

## E) Hybrid-Deployments: OnlyOffice / LiveKit / MinIO

### E1) OnlyOffice Document Server

**Was:** Bearbeitung von .docx/.xlsx/.pptx direkt im Browser, eingebettet in KMU Hub via WOPI-Protokoll.

**Server-Anforderungen:**

| Eigenschaft | Minimum | Empfohlen | Gross (50+ gleichzeitige Editoren) |
|---|---|---|---|
| CPU | 2 Cores | 4 Cores | 8 Cores |
| RAM | 4 GB | 8 GB | 16 GB |
| Disk | 20 GB SSD | 40 GB SSD | 80 GB SSD |
| Netzwerk | 100 Mbit/s | 1 Gbit/s | 1 Gbit/s |

**RAM-Breakdown:**
- OnlyOffice Core (~1,5 GB Basis)
- Pro gleichzeitiges Dokument: ~100-200 MB
- Node.js Frontend: ~300 MB
- PostgreSQL (intern): ~500 MB
- Converter (LibreOffice-basiert): ~500 MB pro Konvertierung

**Docker Deployment:**

```yaml
onlyoffice:
  image: onlyoffice/documentserver:8.2
  restart: unless-stopped
  environment:
    - JWT_ENABLED=true
    - JWT_SECRET=${ONLYOFFICE_JWT_SECRET}
    - JWT_HEADER=AuthorizationJwt
    - WOPI_ENABLED=true
  volumes:
    - onlyoffice_data:/var/www/onlyoffice/Data
    - onlyoffice_logs:/var/log/onlyoffice
    - onlyoffice_fonts:/usr/share/fonts/custom
  ports:
    - "8443:443"
  deploy:
    resources:
      limits:
        memory: 8G
      reservations:
        memory: 4G
```

**Lizenzierung:**

| Edition | Preis | Limits | Fuer KMU Hub |
|---|---|---|---|
| Community (kostenlos) | 0 EUR | Max 20 gleichzeitige Verbindungen, kein Mobile-Editor | Self-Hosted Klein |
| Enterprise (Self-Hosted) | ~1.200-2.400 EUR/Jahr (ab 50 User) | Unbegrenzt, Mobile, SSO | Self-Hosted Mittel/Gross |
| Developer (OEM/SaaS) | ~3.000-6.000 EUR/Jahr | White-Label, eigene Domains | SaaS-Betrieb |

**Empfehlung fuer KMU Hub:**
- **Self-Hosted Kunden (Klein):** Community Edition (kostenlos, 20 Connections reicht fuer 5-20 MA)
- **Self-Hosted Kunden (Mittel/Gross):** Enterprise Lizenz
- **SaaS-Betrieb:** Developer/OEM-Lizenz (erlaubt Multi-Tenant-Nutzung). Preis verhandelbar ab Volumen.

### E2) LiveKit Server

**Was:** WebRTC-basierte Video/Audio-Konferenzen. Self-hostable, Open Source.

**Server-Anforderungen:**

| Szenario | CPU | RAM | Bandbreite | Hetzner-Server |
|---|---|---|---|---|
| 1:1 Calls (max 5 gleichzeitig) | 2 Cores | 2 GB | 50 Mbit/s | CX22 |
| Gruppen-Calls bis 10 TN (max 3 gleichzeitig) | 4 Cores | 4 GB | 200 Mbit/s | CX32 |
| Gruppen-Calls bis 25 TN (max 5 gleichzeitig) | 4 Cores | 8 GB | 500 Mbit/s | CX42 |
| Webinare bis 100 TN (SFU-Modus) | 8 Cores | 16 GB | 1 Gbit/s | CCX23 (dediziert) |

**Bandbreiten-Kalkulation:**
- 1 Video-Stream (720p): ~1,5-2 Mbit/s
- 1 Audio-Stream: ~50-100 Kbit/s
- SFU (Selective Forwarding): Server leitet Streams weiter, kein Transcoding
- 10-Personen-Call: Jeder sendet 1 Stream, empfaengt 9 = ~20 Mbit/s am Server (Upstream)
- 10 solche Calls gleichzeitig: ~200 Mbit/s

**TURN/STUN:**
- **STUN:** Kostenlos, hilft NAT-Traversal. Eigener STUN-Server auf LiveKit-Box.
- **TURN:** Wird benoetigt wenn UDP blockiert ist (~10-15% der Firmennetzwerke). Relay ueber TCP.
- LiveKit hat eingebauten TURN-Support. Kein externer TURN-Server noetig.

**LiveKit Config:**

```yaml
# livekit.yaml
port: 7880
rtc:
  port_range_start: 50000
  port_range_end: 60000
  use_external_ip: true
  tcp_port: 7881
turn:
  enabled: true
  domain: turn.kmuhub.example.com
  tls_port: 5349
  udp_port: 3478
keys:
  # API Key : Secret (aus .env)
  ${LIVEKIT_API_KEY}: ${LIVEKIT_API_SECRET}
room:
  max_participants: 50
  empty_timeout: 300  # 5 Min nach letztem Teilnehmer
logging:
  level: info
```

**Traffic-Kosten:**
- Hetzner inkludiert 20 TB/Mo Traffic bei Cloud-Servern.
- Video-Traffic pro Stunde/User: ~700 MB (720p bidirektional)
- 100 User, je 2h Video/Tag, 22 Arbeitstage: ~3 TB/Mo. Weit unter Hetzner-Limit.
- **Erst ab 500+ Power-Video-Usern wird Traffic ein Kostenfaktor.**

### E3) MinIO Object Storage

**Was:** S3-kompatibles Object Storage. Dateien, Avatare, Attachments, Rapport-Fotos, Backups.

**Warum MinIO statt Hetzner Object Storage:**
- Self-Hosted: Kein externer Dienst noetig
- SaaS: Hetzner Object Storage ist guenstiger. MinIO nur als Self-Hosted-Option.
- Datenhoheit: Alles auf eigener Infrastruktur

**Server-Anforderungen:**

| Groesse | Storage | RAM | CPU | Deployment |
|---|---|---|---|---|
| Klein (bis 100 GB) | 1x SSD oder HDD | 1 GB | 1 Core | Single-Node, Docker |
| Mittel (bis 1 TB) | 2x HDD (Erasure Coding) | 2 GB | 2 Cores | Single-Node |
| Gross (bis 10 TB) | 4x HDD (Erasure Coding) | 4 GB | 4 Cores | Multi-Node |

**Docker Deployment (Single-Node):**

```yaml
minio:
  image: minio/minio:latest
  command: server /data --console-address ":9001"
  environment:
    MINIO_ROOT_USER: ${MINIO_ROOT_USER}
    MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    MINIO_REGION: eu-central-1
  volumes:
    - /mnt/storage/minio:/data
  ports:
    - "9000:9000"   # S3 API
    - "9001:9001"   # Console
  healthcheck:
    test: ["CMD", "mc", "ready", "local"]
    interval: 30s
```

**Bucket-Struktur:**

```
kmuhub-uploads/
  ├── {tenant_id}/
  │   ├── documents/        # Dokumente, Vertraege
  │   ├── avatars/          # Profilbilder
  │   ├── rapporte/         # Rapport-Fotos (Bau/Handwerk)
  │   ├── attachments/      # E-Mail-Attachments
  │   ├── invoices/         # Generierte PDFs
  │   └── temp/             # Temporaere Uploads (24h TTL)
  └── ...

kmuhub-backups/
  ├── daily/
  ├── weekly/
  └── monthly/
```

**SaaS-Alternative:** Hetzner Object Storage (~0,0126 EUR/GB/Mo). S3-kompatibel, kein MinIO noetig. Massiv guenstiger als AWS S3 (~0,023 EUR/GB/Mo).

---

## F) Kosten-Kalkulation

### F1) SaaS Hosting-Kosten pro 100 Kunden

**Annahmen:**
- 100 Kunden, Durchschnitt 20 MA/Kunde = 2.000 User total
- 80% Klein (5-20 MA), 15% Mittel (20-100 MA), 5% Gross (100-200 MA)
- Nicht alle User gleichzeitig aktiv (Peak: ~30% = 600 gleichzeitige User)
- RLS Multi-Tenant (eine DB fuer alle)

| Posten | Spezifikation | EUR/Mo |
|---|---|---|
| **App-Server (K8s Worker Pool)** | 4x CX32 (4 vCPU, 8 GB each) | ~38,36 |
| **DB Primary** | CX42 (8 vCPU, 16 GB) oder CCX13 dediziert | ~17,99 - 49,99 |
| **DB Read-Replica** | CX32 | ~9,59 |
| **Redis** | CX22 (dediziert) | ~5,39 |
| **LiveKit Cluster** | 2x CX32 | ~19,18 |
| **OnlyOffice** | 2x CX32 (fuer Redundanz) | ~19,18 |
| **MinIO / Object Storage** | Hetzner Object Storage, ~2 TB | ~25,20 |
| **Load Balancer** | LB11 | ~5,39 |
| **Backup Storage** | BX41 (20 TB) | ~33,56 |
| **Floating IPs** | 3x IPv4 | ~13,53 |
| **Bandbreite** | Inkludiert (20 TB/Server) | 0,00 |
| **Monitoring** | Auf bestehendem Node | 0,00 |
| **Domains / DNS** | Cloudflare Free | 0,00 |
| | | |
| **Gesamt (Basis)** | | **~185-230 EUR** |
| **Pro Kunde** | | **~1,85-2,30 EUR** |
| **Pro User** | | **~0,09-0,12 EUR** |

**Hochrechnung bei Wachstum:**

| Kunden | User (geschaetzt) | Infra-Kosten/Mo | Pro Kunde | Pro User |
|---|---|---|---|---|
| 100 | 2.000 | ~200 EUR | ~2,00 EUR | ~0,10 EUR |
| 500 | 10.000 | ~600 EUR | ~1,20 EUR | ~0,06 EUR |
| 1.000 | 20.000 | ~1.100 EUR | ~1,10 EUR | ~0,055 EUR |
| 5.000 | 100.000 | ~4.000 EUR | ~0,80 EUR | ~0,04 EUR |

**Economies of Scale:** Infrastruktur-Kosten skalieren sublinear. PostgreSQL mit RLS und guter Indizierung haelt 100.000+ User auf einer Maschine. Horizontal skaliert werden nur stateless Services (Gateway, CRM-Service etc.).

### F2) Self-Hosted Minimum-Kosten

| Szenario | Hardware/Cloud | EUR/Mo |
|---|---|---|
| **Minimalist (Cloud)** | Hetzner CX22 + BX11 | **~10-15 EUR** |
| **Komfortabel (Cloud)** | Hetzner CX32 + CX22 DB + BX21 | **~25-35 EUR** |
| **On-Premise (eigene HW)** | 1x NUC/MiniPC (i5, 16 GB, 500 GB SSD) | **~0 EUR laufend** (Strom: ~5 EUR/Mo, Anschaffung: ~500 EUR einmalig) |
| **On-Premise (gebraucht)** | 1x Dell OptiPlex Micro (i7, 32 GB, 1 TB SSD) | **~0 EUR laufend** (Strom: ~8 EUR/Mo, Anschaffung: ~300-400 EUR gebraucht) |

**Anmerkung On-Premise:** Viele Handwerks-KMUs haben bereits einen Server (NAS, alter PC). KMU Hub laeuft darauf in Docker. Einmalkosten fuer Setup durch IT-Partner: ~500-1.000 EUR.

### F3) OnlyOffice Lizenzkosten

| Edition | Preis | Empfehlung |
|---|---|---|
| **Community** (AGPL) | 0 EUR | Self-Hosted Klein (max 20 Connections) |
| **Enterprise** (Self-Hosted) | ~1.200 EUR/Jahr (50 User) bis ~4.800 EUR/Jahr (250 User) | Self-Hosted Mittel/Gross |
| **Developer** (OEM/SaaS) | ~3.000-6.000 EUR/Jahr | SaaS-Betrieb (Multi-Tenant) |
| **Cloud (SaaS von OnlyOffice)** | Ab ~5 EUR/User/Mo | NICHT empfohlen (Daten verlassen EU-Kontrolle) |

**Fuer KMU Hub SaaS:**
- Developer-Lizenz: ~4.000 EUR/Jahr = ~333 EUR/Mo
- Bei 100 Kunden: +3,33 EUR/Kunde/Mo
- Bei 500 Kunden: +0,67 EUR/Kunde/Mo
- Bei 1.000 Kunden: +0,33 EUR/Kunde/Mo

**Skaliert gut:** OEM-Lizenz ist flat-rate, nicht pro User. Je mehr Kunden, desto guenstiger pro Kopf.

### F4) LiveKit Bandwidth-Kosten

| Szenario | Traffic/Mo | Kosten bei Hetzner | Kosten bei Exoscale |
|---|---|---|---|
| 100 Kunden, wenig Video (2h/Woche avg) | ~2 TB | 0 EUR (inkludiert) | ~30 EUR |
| 100 Kunden, mittel Video (1h/Tag avg) | ~8 TB | 0 EUR (inkludiert) | ~120 EUR |
| 500 Kunden, viel Video (2h/Tag avg) | ~70 TB | ~70 EUR (Overage) | ~1.050 EUR |
| 1.000 Kunden, Power-Video | ~200 TB | ~200-400 EUR | ~3.000 EUR |

**Hetzner Traffic:** 20 TB/Mo inkludiert pro Server. Bei 4 Servern = 80 TB inkludiert. Overage: ~1,19 EUR/TB.
**Exoscale Traffic:** ~0,015 EUR/GB (ausgehend). Deutlich teurer.

**Empfehlung:** Video-intensive Workloads IMMER auf Hetzner. Exoscale nur fuer Daten-Storage (DB, Files) wo die Schweizer Residenz noetig ist.

### F5) Backup-Storage-Kosten

| Groesse | Taeglich | 30-Tage-Retention | Storage | Kosten/Mo |
|---|---|---|---|---|
| Klein (100 Kunden) | ~5 GB Dump + ~20 GB Files | ~150 GB + 20 GB = 170 GB | BX11 (1 TB) | ~3,81 EUR |
| Mittel (500 Kunden) | ~25 GB Dump + ~100 GB Files | ~750 GB + 100 GB = 850 GB | BX11 (1 TB) | ~3,81 EUR |
| Gross (1.000 Kunden) | ~50 GB Dump + ~500 GB Files | ~1,5 TB + 500 GB = 2 TB | BX21 (5 TB) | ~10,08 EUR |
| Sehr Gross (5.000 Kunden) | ~250 GB Dump + ~2 TB Files | ~7,5 TB + 2 TB = 9,5 TB | BX31 (10 TB) | ~16,35 EUR |

**Georedundanz (Empfohlen ab 500 Kunden):**
- Zweiter Backup-Standort bei OVH (DE) oder Exoscale (CH): +~15-30 EUR/Mo
- Automatischer Rsync per Cron (naechtlich)

### F6) Gesamt-Kostenmatrix (SaaS)

| Posten | 100 Kunden | 500 Kunden | 1.000 Kunden |
|---|---|---|---|
| Infrastruktur (Hetzner) | ~200 EUR | ~600 EUR | ~1.100 EUR |
| OnlyOffice Lizenz | ~333 EUR | ~333 EUR | ~333 EUR |
| Backup + Georedundanz | ~20 EUR | ~45 EUR | ~60 EUR |
| Domain + SSL | ~5 EUR | ~5 EUR | ~5 EUR |
| Monitoring (extern, optional) | 0 EUR | ~30 EUR | ~50 EUR |
| **Total Hosting** | **~558 EUR** | **~1.013 EUR** | **~1.548 EUR** |
| **Pro Kunde** | **~5,58 EUR** | **~2,03 EUR** | **~1,55 EUR** |
| **Pro User (20 avg)** | **~0,28 EUR** | **~0,10 EUR** | **~0,08 EUR** |

**Marge bei KMU Hub Pricing (geschaetzt ~20-25 EUR/User/Mo):**
- Bei 100 Kunden (2.000 User): Umsatz ~40.000-50.000 EUR/Mo, Hosting ~558 EUR = **~99% Bruttomarge auf Hosting**
- Bei 1.000 Kunden (20.000 User): Umsatz ~400.000-500.000 EUR/Mo, Hosting ~1.548 EUR = **~99,7% Bruttomarge auf Hosting**

**SaaS-Hosting ist kein relevanter Kostenfaktor.** Die echten Kosten sind Personal, Support, Sales und Compliance.

---

## G) Sicherheits-Architektur

### G1) TLS Everywhere

```
                Internet
                   │
            ┌──────▼──────┐
            │  Cloudflare  │  TLS 1.3 (Edge)
            │  / Hetzner   │
            │    LB        │
            └──────┬───────┘
                   │ TLS 1.3 (Origin)
            ┌──────▼──────┐
            │  Traefik    │  TLS-Termination + mTLS intern
            │  Ingress    │
            └──────┬──────┘
                   │ mTLS (Service-to-Service)
         ┌─────────┼─────────┐
         │         │         │
    ┌────▼───┐ ┌───▼───┐ ┌──▼────┐
    │API GW  │ │CRM Svc│ │Chat   │
    │        │ │       │ │Svc    │
    └────┬───┘ └───┬───┘ └──┬────┘
         │         │        │
         │    TLS (PostgreSQL sslmode=verify-full)
         │    TLS (Redis TLS-Port 6380)
         │         │        │
    ┌────▼─────────▼────────▼────┐
    │  PostgreSQL    │   Redis   │
    │  (ssl=on)      │  (tls)   │
    └────────────────┴──────────┘
```

**Konfiguration:**
- **Extern:** TLS 1.3 only. Kein TLS 1.0/1.1. TLS 1.2 nur als Fallback.
- **PostgreSQL:** `sslmode=verify-full` in Connection String. Self-Signed CA fuer interne Kommunikation.
- **Redis:** TLS-Port 6380 (statt 6379 unverschluesselt). Redis 7+ hat nativen TLS-Support.
- **MinIO:** TLS auf S3-API-Port (9000).
- **LiveKit:** WebRTC ist standardmaessig DTLS-verschluesselt. TURN ueber TLS (Port 5349).

**Zertifikate:**
- **Extern:** Let's Encrypt (automatisch via Traefik/Caddy)
- **Intern:** Eigene CA (Self-Signed), generiert bei Setup. Cert-Manager in Kubernetes.

### G2) Netzwerk-Isolation

**Hetzner Cloud: Private Networks (vSwitch)**

```
┌─────────────────────────────────────────────────┐
│  Private Network: 10.0.0.0/16                   │
│                                                  │
│  Subnet "app": 10.0.1.0/24                      │
│    10.0.1.10 - API Gateway                      │
│    10.0.1.11 - CRM Service                      │
│    10.0.1.12 - Chat Service                     │
│    10.0.1.13 - Auth Service                     │
│                                                  │
│  Subnet "data": 10.0.2.0/24                     │
│    10.0.2.10 - PostgreSQL Primary               │
│    10.0.2.11 - PostgreSQL Replica               │
│    10.0.2.12 - Redis                            │
│                                                  │
│  Subnet "media": 10.0.3.0/24                    │
│    10.0.3.10 - LiveKit                          │
│    10.0.3.11 - OnlyOffice                       │
│    10.0.3.12 - MinIO                            │
│                                                  │
│  Nur Gateway hat oeffentliche IP!               │
│  (+ LiveKit fuer WebRTC UDP)                    │
└─────────────────────────────────────────────────┘
```

**Hetzner Private Networks:** Kostenlos. Kein Traffic zwischen Servern im gleichen Netzwerk zaehlt zum Limit.

**Kubernetes: NetworkPolicies**

```yaml
# Nur Gateway darf auf CRM-Service zugreifen
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: crm-service-policy
spec:
  podSelector:
    matchLabels:
      app: crm-service
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: api-gateway
    ports:
    - port: 8080
```

**Kein VPN noetig:** Hetzner Private Networks sind bereits isoliert (VXLAN-basiert). VPN (WireGuard) nur fuer:
- Zugriff von Entwickler-Rechnern auf interne Services
- Cross-Datacenter-Verbindung (Hetzner DE <-> Exoscale CH)

### G3) Firewall-Regeln

**Hetzner Cloud Firewall (kostenlos):**

```
# App-Server (Gateway)
INBOUND:
  - TCP 443 (HTTPS) von 0.0.0.0/0
  - TCP 80 (HTTP -> Redirect to 443) von 0.0.0.0/0
  - TCP 22 (SSH) von <Admin-IPs> only
OUTBOUND:
  - Alle (fuer API-Calls, Mail etc.)

# DB-Server
INBOUND:
  - TCP 5432 von 10.0.1.0/24 (App-Subnet) only
  - TCP 22 von <Admin-IPs> only
OUTBOUND:
  - TCP 5432 zu 10.0.2.11 (Replica) only
  - Kein Internet-Zugang!

# Redis
INBOUND:
  - TCP 6380 von 10.0.1.0/24 only
OUTBOUND:
  - Keiner

# LiveKit
INBOUND:
  - TCP 7880 (HTTP) von 10.0.1.0/24
  - TCP 7881 (WebSocket) von 0.0.0.0/0
  - UDP 50000-60000 (WebRTC) von 0.0.0.0/0
  - TCP 5349 (TURN/TLS) von 0.0.0.0/0
OUTBOUND:
  - Alle (WebRTC Relay)

# MinIO
INBOUND:
  - TCP 9000 von 10.0.1.0/24 only (S3 API)
  - TCP 9001 von <Admin-IPs> only (Console)
OUTBOUND:
  - Keiner
```

### G4) DDoS-Schutz

**Hetzner inkludierter DDoS-Schutz:**
- Automatische Mitigation bei volumetrischen Angriffen (Layer 3/4)
- Scrubbing Center in Frankfurt
- Kein Aufpreis, aktiviert bei allen Cloud-Servern
- Schutz bis ~100 Gbit/s (Standard), hoeher auf Anfrage

**Zusaetzlicher Schutz (Empfohlen ab 500+ Kunden):**

| Massnahme | Implementation | Kosten |
|---|---|---|
| **Cloudflare Pro** | DNS-Proxy, WAF, Bot-Schutz, Rate Limiting | ~20 EUR/Mo |
| **Rate Limiting (Application-Level)** | Go-Middleware mit Redis Token Bucket | 0 EUR (eingebaut) |
| **Fail2Ban** | SSH Brute-Force-Schutz auf allen Servern | 0 EUR |
| **GeoIP-Blocking** | Nur DACH-Traffic fuer Login erlauben (optional) | 0 EUR (Cloudflare) |

**Rate Limiting im API Gateway (Go):**

```go
// Beispiel: Redis-basiertes Rate Limiting
// 100 Requests/Minute pro API-Key, 1000/Minute pro IP
func RateLimitMiddleware(redis *redis.Client) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := fmt.Sprintf("ratelimit:%s", r.RemoteAddr)
            count, _ := redis.Incr(r.Context(), key).Result()
            if count == 1 {
                redis.Expire(r.Context(), key, time.Minute)
            }
            if count > 100 {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### G5) Secrets Management

**Option A: Environment Variables + Encrypted .env (Start)**

| Vorteil | Nachteil |
|---|---|
| Einfach, Docker-nativ | Keine Rotation, keine Audit-Logs |
| Kein zusaetzlicher Service | Plaintext auf Disk (wenn nicht verschluesselt) |
| 12-Factor-App konform | Kein feingranularer Zugriff |

**Implementation:**
```bash
# .env (NIE committen, .gitignore!)
DB_PASSWORD=<random-64-char>
JWT_SECRET=<random-64-char>
LIVEKIT_API_KEY=<generated>
LIVEKIT_API_SECRET=<generated>
MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=<random-32-char>
ONLYOFFICE_JWT_SECRET=<random-32-char>
```

**Option B: HashiCorp Vault (ab 500+ Kunden)**

| Vorteil | Nachteil |
|---|---|
| Automatische Secret-Rotation | Zusaetzlicher Service (~2 GB RAM) |
| Audit-Logging (wer hat wann welches Secret gelesen) | Komplexeres Setup |
| Dynamic Database Credentials | Erhoehte Ops-Komplexitaet |
| Encryption as a Service | Muss selbst hochverfuegbar sein |

**Vault Deployment:**
```yaml
vault:
  image: hashicorp/vault:1.15
  cap_add:
    - IPC_LOCK
  environment:
    VAULT_ADDR: "http://0.0.0.0:8200"
  volumes:
    - vault_data:/vault/data
    - ./config/vault.hcl:/vault/config/vault.hcl
  command: server -config=/vault/config/vault.hcl
```

**Vault Use Cases fuer KMU Hub:**
1. **DB-Credentials:** Vault generiert kurzlebige PostgreSQL-Credentials (TTL 1h). Bei Kompromittierung automatisch invalidiert.
2. **JWT Signing Keys:** Rotation alle 24h, alte Keys bleiben zum Verifizieren.
3. **API-Keys (Integrationen):** Bexio, Brevo, Skribble API-Keys sicher gespeichert.
4. **Backup-Encryption-Keys:** AES-256-Keys fuer Backup-Verschluesselung.
5. **Tenant-spezifische Secrets:** IMAP-Passwoerter der Kunden (verschluesselt at rest).

**Empfehlung:**
| Phase | Loesung | Begruendung |
|---|---|---|
| Pre-Beta (jetzt) | **Environment Variables** | Einfach, schnell, 3 Mann Team |
| Beta (100 Kunden) | **Environment Variables + SOPS** | Verschluesselte .env Dateien in Git (mozilla/sops) |
| Launch (500+ Kunden) | **HashiCorp Vault** | Secret-Rotation, Audit-Compliance, dynamische DB-Credentials |
| Enterprise (1.000+) | **Vault HA Cluster** | 3-Node-Raft-Cluster fuer Hochverfuegbarkeit |

### G6) Zusammenfassung Sicherheits-Checkliste

| Massnahme | Phase | Status |
|---|---|---|
| TLS 1.3 ueberall (API, DB, Redis, LiveKit) | Pre-Beta | MUSS |
| Hetzner Private Networks (kein oeffentliches DB) | Pre-Beta | MUSS |
| Hetzner Cloud Firewall (alle Server) | Pre-Beta | MUSS |
| Rate Limiting (Redis Token Bucket) | Pre-Beta | MUSS |
| CSRF-Schutz (alle mutierenden Endpoints) | Pre-Beta | MUSS |
| SQL-Injection (Prepared Statements only) | Pre-Beta | MUSS |
| CORS (explizite Allowlist, kein Wildcard) | Pre-Beta | MUSS |
| Row-Level Security (PostgreSQL) | Pre-Beta | MUSS |
| Passwort-Policy (min 12 Zeichen) + 2FA (TOTP) | Pre-Beta | MUSS |
| Backup-Verschluesselung (AES-256) | Pre-Beta | MUSS |
| Fail2Ban (SSH) | Pre-Beta | SOLL |
| Cloudflare Pro (WAF + DDoS Layer 7) | Beta | SOLL |
| GeoIP-Blocking (Login nur DACH) | Beta | OPTIONAL |
| HashiCorp Vault | Launch | SOLL |
| Penetrationstest | Pre-Launch | MUSS (~3.000-8.000 EUR) |
| ISO 27001 Vorbereitung | Post-Launch | SOLL (~30.000-70.000 EUR) |

---

## Anhang: Quick-Reference-Tabellen

### Hetzner Cloud Server-Typen (Referenz, Stand ~2025)

**Shared vCPU (CX-Serie, x86):**

| Typ | vCPU | RAM | SSD | Traffic | EUR/Mo (ca.) |
|---|---|---|---|---|---|
| CX11 | 1 | 2 GB | 20 GB | 20 TB | ~3,79 |
| CX22 | 2 | 4 GB | 40 GB | 20 TB | ~5,39 |
| CX32 | 4 | 8 GB | 80 GB | 20 TB | ~9,59 |
| CX42 | 8 | 16 GB | 160 GB | 20 TB | ~17,99 |
| CX52 | 16 | 32 GB | 320 GB | 20 TB | ~33,99 |

**Shared vCPU (CAX-Serie, ARM64 -- guenstiger):**

| Typ | vCPU | RAM | SSD | Traffic | EUR/Mo (ca.) |
|---|---|---|---|---|---|
| CAX11 | 2 | 4 GB | 40 GB | 20 TB | ~3,79 |
| CAX21 | 4 | 8 GB | 80 GB | 20 TB | ~6,49 |
| CAX31 | 8 | 16 GB | 160 GB | 20 TB | ~11,49 |
| CAX41 | 16 | 32 GB | 320 GB | 20 TB | ~19,49 |

**Dedicated vCPU (CCX-Serie):**

| Typ | vCPU | RAM | SSD | Traffic | EUR/Mo (ca.) |
|---|---|---|---|---|---|
| CCX13 | 2 | 8 GB | 80 GB | 20 TB | ~13,49 |
| CCX23 | 4 | 16 GB | 160 GB | 20 TB | ~24,49 |
| CCX33 | 8 | 32 GB | 240 GB | 20 TB | ~46,49 |
| CCX43 | 16 | 64 GB | 360 GB | 20 TB | ~90,49 |

**Anmerkung ARM (CAX):** Go cross-kompiliert trivial zu ARM64 (`GOARCH=arm64`). PostgreSQL, Redis, MinIO laufen nativ auf ARM. **CAX-Server sind ~20-35% guenstiger bei vergleichbarer Leistung.** Empfehlung: CAX fuer alle Workloads wo ARM-Kompatibilitaet gegeben (= alles ausser OnlyOffice, das braucht x86).

### Hetzner Storage Boxes (Referenz)

| Typ | Storage | Preis/Mo (ca.) | Snapshots | Protokolle |
|---|---|---|---|---|
| BX11 | 1 TB | ~3,81 EUR | 10 | SFTP, SCP, Rsync, Samba/CIFS, WebDAV |
| BX21 | 5 TB | ~10,08 EUR | 10 | dto. |
| BX31 | 10 TB | ~16,35 EUR | 10 | dto. |
| BX41 | 20 TB | ~33,56 EUR | 10 | dto. |

### Hetzner Load Balancer

| Typ | Connections | Bandbreite | Preis/Mo (ca.) |
|---|---|---|---|
| LB11 | 10.000 | 25.000 CPS | ~5,39 EUR |
| LB21 | 25.000 | 50.000 CPS | ~14,99 EUR |
| LB31 | 50.000 | 100.000 CPS | ~29,99 EUR |

### Hetzner Object Storage

| Speicher | Preis/Mo |
|---|---|
| Pro GB | ~0,0126 EUR |
| Pro 100 GB | ~1,26 EUR |
| Pro 1 TB | ~12,60 EUR |
| Egress Traffic | 1 TB frei, danach ~1,19 EUR/TB |

---

*Hinweis: Alle Preise basieren auf Trainingsdaten (Stand Mai 2025). Hetzner, Exoscale und OnlyOffice koennen ihre Preise jederzeit aendern. Vor Verwendung in Vertraegen oder Kalkulationen muessen aktuelle Preise auf den jeweiligen Websites verifiziert werden. Die Architektur-Empfehlungen stellen keine verbindliche Beratung dar und muessen an die spezifischen Anforderungen angepasst werden.*
