# Cosmi Dialer — Strategische Roadmap

## Context

Cosmi hat bereits eine produktionsreife LiveKit-Integration (WebRTC Rooms, Recording, Signaling), ein vollstaendiges CRM-Datenmodell mit `ActivityType = "call"` und `Contact.Phone`, ein WASM-Plugin-System, eine erweiterbare Automation Engine und das ServiceRegistry-Pattern fuer neue Microservices. Die Architektur ist ideal vorbereitet, ein Dialer-Modul modular aufzubauen — vom Click-to-Call MVP bis zur vollwertigen Contact-Center-Plattform.

**Ziel:** Von "Click-to-Call MVP" zu einer marktfaehigen, EU-souveraenen Dialer-Plattform, die sich bewusst von Dialfire/Aircall/CloudTalk abhebt.

---

## Phase 1 — MVP: Click-to-Call + Kampagnenlisten

**Scope:** Interne WebRTC-Anrufe, Kampagnenverwaltung, Call-Outcomes, CRM-Integration
**Aufwand:** M (6-10 Wochen mit AI-Pair)
**Status:** ✅ Alle 5 Sub-Phasen abgeschlossen (Stand: 2026-04-09)

### Implementierungsfortschritt

| Sub-Phase | Status | Inhalt |
|-----------|--------|--------|
| 1A — Foundation | ✅ Done | Proto (27 RPCs, 6 Enums), 5 Migrations (063-067), Service-Skeleton, Gateway-Stub, Docker |
| 1B — Backend Core | ✅ Done | service.go (24 Methoden), 4 Repos, Redis Agent-Status, CRM-Bridge, E.164 Phone, gRPC-Server |
| 1C — Gateway + Permissions | ✅ Done | 25 REST-Endpoints (route_dialer.go, 1014 LoC), Permission-Migration (068) |
| 1D — Frontend | ✅ Done | 26 Dateien, 4-Phasen Workspace, Campaigns, Dashboard, Settings, Mock-Handler, i18n (DE/EN/FR/IT) |
| 1E — Integration | ✅ Done | CRM-Timeline live, Callback-Notifications, Filter-Import, Bug Fixes (ContactID, wrap_up, skip), EventEmitter, Unit/Gateway/E2E Tests |

### Features

| Feature | Typ | Aufwand | Abhaengigkeit |
|---------|-----|---------|--------------|
| Click-to-Call Button auf Kontakt-Karten | Commodity | S | LiveKit (done) |
| Kampagnenlisten aus CRM-Filtern erstellen | **Differentiator** | M | CRM Saved-Filter |
| Preview-Modus: Manuelles Durchklicken der Liste | Commodity | M | Kampagnenlisten |
| Call-Outcome Logging (Erreicht / Nicht erreicht / Wiedervorlage / Termin) | Commodity | S | ActivityTypeCall (done) |
| In-Call Notizen + Naechste Aktion | Commodity | S | Activity Custom Fields |
| Terminbuchung waehrend des Gespraechs | **Differentiator** | S | Calendar-Modul (done) |
| Agent-Status: Available / On Call / Wrap-Up / Break | Commodity | S | Presence (erweitern) |
| Kampagnen-Dashboard (Fortschritt, Outcomes, Calls/h) | Commodity | M | DB Aggregation |

### Architektur

**Neuer Service: `dialer` (Port 50061)**
Eigener Microservice statt Erweiterung von `work` — das Dialer-Modul hat eigenen Lifecycle (Kampagnen, Queues, Agent-Pools, Outcomes) und wuerde `work` zum God-Service aufblaehen.

**Neue Datenbank-Tabellen (Migrations 063-067):**

| Tabelle | Zweck |
|---------|-------|
| `dialer_campaigns` | id, name, status (draft/active/paused/completed), settings (JSONB), **tenant_id** |
| `dialer_campaign_contacts` | campaign_id, contact_id, position, status, outcome, notes, callback_at |
| `dialer_call_sessions` | campaign_contact_id -> call_session_id (FK Video), agent_id, outcome, duration, notes |
| `dialer_agent_status` | user_id, status, campaign_id (Redis-backed, DB fuer Audit) |
| `dialer_call_outcomes` | label, color, is_positive, is_callback, **tenant_id** (konfigurierbar pro Tenant) |

**Proto:** `proto/dialer/v1/dialer.proto` mit RPCs:
- Campaign CRUD: `CreateCampaign`, `ListCampaigns`, `UpdateCampaign`, `PauseCampaign`
- Queue: `AddContactsToCampaign`, `GetNextContact`
- Calls: `LogCallOutcome`, `SetAgentStatus`
- Dashboard: `GetCampaignDashboard`

**Gateway:** `route_dialer.go` nach dem `route_video.go`-Pattern.

**Frontend-Modul:** `desktop/src/renderer/src/modules/dialer/`
- `CampaignListPage.tsx` — Kampagnen verwalten
- `DialerWorkspace.tsx` — Call-Screen (Kontakt-Karte + Controls + Outcome-Panel)
- `AgentStatusBar.tsx` — Persistente Statusleiste
- `CampaignDashboard.tsx` — Fortschritt + Outcome-Breakdown
- `CallOutcomeDialog.tsx` — Wrap-Up (Outcome, Notizen, Termin)

**CRM-Bridge:** `LogCallOutcome` -> gRPC Call an CRM Service -> `ActivityTypeCall` Record auf Kontakt-Timeline.

**Neue Automation-Trigger:** `dialer.call.outcome_logged`, `dialer.campaign.completed`, `dialer.contact.callback_scheduled`

### Risiken Phase 1

| Risiko | Mitigation |
|--------|------------|
| WebRTC Audio in Electron | LiveKit SDK bereits battle-tested im Video-Modul |
| Filter-Performance bei grossen Listen | Saved-Filter DSL wiederverwenden, Index auf `(campaign_id, status)` |
| Agent-Status Sync-Lag | Redis Pub/Sub (bereits im Presence-System) |

---

## Phase 2 — Echter PSTN Power Dialer

**Scope:** SIP-Telefonie, Power Dialer, HubSpot-Sync, DSGVO-Recording, Supervisor
**Aufwand:** XL (15-22 Wochen mit AI-Pair)
**Voraussetzung:** Phase 1 abgeschlossen

### Features

| Feature | Typ | Aufwand | Abhaengigkeit |
|---------|-----|---------|--------------|
| LiveKit SIP Trunk Bridge (echte Telefonnummern) | Commodity | L | Phase 1, SIP-Provider |
| Power Dialer (Auto-Advance nach Wrap-Up) | Commodity | M | Phase 1 Preview Mode |
| HubSpot Bidirektionaler Sync (Kontakte, Deals, Activities) | **Differentiator** | L | Bexio-Integration-Pattern |
| Konfigurierbare Qualifizierungsfelder pro Kampagne | **Differentiator** | M | Custom Fields (done) |
| DSGVO Call Recording mit Consent-Flow | Commodity | M | EgressManager + Consent (done) |
| Wiedervorlagen-Scheduling mit Erinnerungen | Commodity | S | Calendar + Notification |
| DNC-Listen (Do Not Call) Management | Commodity | M | Neue DNC-Tabelle |
| Echtzeit Agent-Dashboard (Calls/h, Conversion, Queue) | Commodity | M | Time-Series Aggregation |
| Supervisor-View (Agent-Activity Monitor) | **Differentiator** | M | Agent Status + Redis Pub/Sub |
| Webhook API fuer externe Consumer | Commodity | S | Webhook-Pattern (Bexio) |
| Nummern-Provisioning UI (kaufen/verwalten) | Commodity | M | SIP Provider API |

### Architektur — SIP Integration

```
Agent Browser (WebRTC) -> LiveKit Server -> LiveKit SIP Gateway -> SIP Trunk Provider -> PSTN
```

LiveKit hat native SIP-Unterstuetzung (produktionsreif seit v1.7+). `SIPParticipant` API erzeugt einen Outbound-SIP-Call der einem bestehenden LiveKit Room als Participant beitritt. Kein Asterisk, kein FreeSWITCH noetig.

**Erweiterung `room_manager.go`:**
- `CreateSIPParticipant(ctx, roomName, sipURI, displayName, trunkID)`
- `TransferSIPCall(ctx, roomName, participantID, transferTo)` (vorbereitet fuer Phase 3)

**SIP Trunk Abstraction Interface:**
```go
type SIPTrunkProvider interface {
    InitiateCall(ctx, toNumber, fromNumber, roomName string) (callID string, err error)
    HangupCall(ctx, callID string) error
    ListNumbers(ctx) ([]PhoneNumber, error)
    ProvisionNumber(ctx, countryCode, areaCode string) (*PhoneNumber, error)
}
```
-> `SipgateTrunkProvider` + `TelnyxTrunkProvider` implementieren.

### SIP-Provider-Empfehlung (DACH)

| Provider | Staerken | DSGVO | Empfehlung |
|----------|---------|-------|------------|
| **sipgate team** | DE-basiert (Aachen), transparentes Pricing, gute API | Nativ | **Primaer** — beste DACH-Abdeckung |
| **Telnyx** | Exzellente API, elastic SIP, kompetitiv | SCCs | **Fallback** — fuer internationale Calls |
| **easybell** | Pure-DE (Berlin), guenstigste DACH-Tarife | Nativ | Alternative zu sipgate |

**Kosten:** ~1ct/min (SIP) vs. 50-100 EUR/Agent/Monat (Aircall/Dialfire)

### HubSpot-Integration

Folgt dem Bexio-Pattern (`route_bexio.go`):
- OAuth2 Flow -> Token Storage
- Bidirektionaler Sync: Kontakte, Deals, Activities
- Jedes `LogCallOutcome` -> HubSpot Engagement (Call Type) via API
- Inbound-Webhook fuer HubSpot-Events
- Rate-Limit Handling: Exponential Backoff, Batch Operations, Incremental Sync

### Qualifizierungsfelder pro Kampagne

`qualification_schema` (JSONB) auf `dialer_campaigns` -> dynamisch gerenderte Felder im Wrap-Up-Screen (selbes Pattern wie CRM Custom Fields). Werte in `dialer_call_sessions.qualification_data` (JSONB).

**Differentiator:** Die meisten Dialer haben fixe Outcome-Felder. Cosmi macht sie **pro Kampagne frei konfigurierbar** ohne Code-Deployment.

### Risiken Phase 2

| Risiko | Mitigation |
|--------|------------|
| LiveKit SIP Setup-Komplexitaet | Separater Docker-Container, zu docker-compose.prod.yml hinzufuegen |
| Audio-Qualitaet auf Hetzner | Gutes EU-Peering, sipgate ebenfalls DE; TURN-Config fuer NAT |
| HubSpot Rate Limits (100/10s) | Exponential Backoff + Batch + Incremental Sync |
| DSGVO Consent Multi-Jurisdiktion | Default "Always Announce Recording", Tenant-konfigurierbar |

---

## Phase 3 — Vollwertige Contact-Center-Plattform

**Scope:** Predictive Dialing, Inbound ACD, AI-Scoring, Multi-Tenancy
**Aufwand:** XXL (40-60 Wochen)
**Voraussetzung:** Phase 2 abgeschlossen + stabile SIP-Infrastruktur

### Features

| Feature | Typ | Aufwand |
|---------|-----|---------|
| Predictive Dialing (Erlang-C Modell) | **Differentiator** | XL |
| Voicemail Detection (AMD) | Commodity | L |
| IVR / Auto-Attendant (Sprachmenues) | Commodity | L |
| Inbound ACD mit Skill-Based Routing | **Differentiator** | XL |
| Whisper / Barge-In / Silent Monitor | **Differentiator** | M |
| Multi-Tenant API (Dialer-as-a-Platform) | **Moonshot** | XL |
| Advanced BI Dashboards | Commodity | L |
| Blended Campaigns (Inbound + Outbound) | **Differentiator** | L |
| AI Call Scoring / Sentiment Analysis | **Moonshot** | L |
| Branching Call Script Engine | **Differentiator** | M |
| Number Portability Management | Commodity | M |

### Architektur-Highlights

**Predictive Dialing:**
- Erlang-C Berechnung alle 30s basierend auf Rolling-Window Call-Statistiken
- Output: `dial_ratio` (z.B. 1.3x = 1.3 Calls pro verfuegbarem Agent gleichzeitig)
- **Rechtliche Schranke:** UWG § 7 — max. 3% Abandon Rate, Hard-Cap mit Auto-Fallback auf 1:1
- Ueberschuessige Connects -> Warteschleife mit Haltemusik (LiveKit Audio Inject)

**Inbound ACD:**
```sql
dialer_queues (id, name, strategy [round_robin/least_busy/skill_match], max_wait_seconds)
dialer_queue_agents (queue_id, user_id, skill_tags text[], priority int)
```
Inbound SIP-Call -> LiveKit Webhook -> Queue Lookup via DID -> Routing Strategy -> Agent-Einladung via WebSocket.

**Whisper / Barge-In:**
Supervisor tritt LiveKit Room mit speziellen `VideoGrant` Permissions bei:
- Silent Monitor: Audio-Receive only
- Whisper: Audio an Agent only (SIP-Teilnehmer hoert nicht)
- Barge-In: Volle Audio-Publish-Rechte

**AI Call Scoring (EU-souveraen):**
1. Recording endet -> Automation Trigger `dialer.call.ended`
2. S3 Audio -> Self-hosted Whisper (Hetzner) -> Transkript
3. Transkript -> Claude API (Anthropic DPA) -> Scoring (Sentiment, Talk-Ratio, Script-Adherence)
4. Ergebnisse in `dialer_call_ai_scores` -> Supervisor Dashboard

**Multi-Tenancy:**
- `tenant_id` auf allen Tabellen (bereits ab Phase 1 angelegt)
- API-Key Issuance: `dialer_api_keys`
- Per-Tenant SIP Trunk Assignment
- Usage Metering (Calls, Minuten, Storage)
- Billing: Stripe oder SEPA Direct Debit

---

## Strategische Differenzierungsmatrix

### Commodity (Muss man haben, kein USP)
- Click-to-Call, Preview Dialer, Power Dialer
- Call Outcome Logging, DNC-Listen
- Basic Recording, Basic Dashboard
- Voicemail Detection, IVR

### Echte Differentiator

| Feature | Warum differenzierend |
|---------|----------------------|
| **Kampagnenlisten aus CRM-Filtern** | Natives CRM-Dialer-Integration vs. CSV-Upload bei Wettbewerb |
| **In-Call Terminbuchung** | Ein-Screen-Workflow; kein App-Wechsel |
| **Konfigurierbare Qualifizierungsfelder** | Pro Kampagne ohne Code aenderbar; Wettbewerb hat fixe Felder |
| **EU-Datensouveraenitaet fuer Recordings** | Aufnahmen verlassen nie DACH; kein US-Provider noetig |
| **DSGVO-Consent auf Datenmodell-Ebene** | Nicht nachgeruestet, sondern von Tag 1 designt |
| **Plugin-basierte Call Scripts (WASM)** | Kunden-anpassbar ohne Code-Deployment |
| **Blended Inbound + Outbound** | Erfordert tiefe ACD-Integration; bei SME-Tools selten |
| **Whisper / Barge-In** | Bei Enterprise-Tools (Five9, Genesys) Standard, bei SME selten |

### Moonshot (Hohes Impact wenn richtig umgesetzt)

| Feature | Potential |
|---------|-----------|
| **AI Call Scoring (EU-hosted)** | Whisper + Claude, alles in DE = kein Datentransfer; einzigartig |
| **Multi-Tenant Dialer-as-Platform** | Verwandelt Cosmi vom CRM-Feature zur Telephonie-Plattform |

### Wo Cosmi gezielt besser werden kann

| Bereich | Wettbewerb | Cosmi-Vorteil |
|---------|------------|---------------|
| **UX/Flow** | Dialfire -> separate CRM-Sync noetig | Anruf -> Qualifizierung -> Termin -> Deal-Update in einem Screen |
| **Datensouveraenitaet** | Aircall, CloudTalk = US/SK Daten | EU-only by Design, Hetzner DE |
| **Customization** | Fixe Felder, kein Plugin-System | WASM Plugins + konfigurierbare Fields |
| **Preis** | 50-100 EUR/Agent/Monat + CRM-Lizenz | SIP-Kosten ~1ct/min, kein Vendor Lock-In |
| **Compliance** | DSGVO nachgeruestet | Consent-Management seit Tag 1 im Datenmodell |

---

## Markterweiterung: Dialer als Entry-Produkt

### Zielgruppen-Expansion

| Neue Zielgruppe | Anforderung | Phase |
|-----------------|-------------|-------|
| **DACH SDR-Teams** (5-50 Agents) | Power Dialer + CRM-Integration | Phase 2 |
| **Inkasso-Unternehmen** | DNC + DSGVO Recording + Compliance | Phase 2 |
| **Versicherungs-/Finanz-SDR** | EU-Datensouveraenitaet + reguliert | Phase 2 |
| **Call Center** (50-200 Agents) | Predictive + ACD + Supervisor + Reporting | Phase 3 |
| **Marketing-Agenturen** | Multi-Tenant/Multi-Kampagne Isolation | Phase 3 |

### Produkt-Positioning Evolution

**Phase 1:** *"Cosmi CRM — jetzt mit integrierter Telefonie"*
**Phase 2:** *"Die einzige DACH-native CRM+Dialer Loesung — Anrufe, Daten, Server in Deutschland"*
**Phase 3:** *"EU-souveraene Contact-Center-Plattform fuer regulierte Branchen"*

### Geschaefts-Impact

| Metrik | Ohne Dialer | Mit Dialer (Phase 2+) |
|--------|-------------|----------------------|
| ARR pro Kunde | ~5K EUR (CRM only) | 20-80K EUR (CRM + Dialer) |
| Zielmarkt | SME CRM (gesaettigt) | Contact Center + Sales Teams (wachsend) |
| Lock-In | Mittel (CRM-Wechsel moeglich) | Hoch (Telefonie-Migration = Pain) |
| Wettbewerbs-Moat | EU-Hosting | EU-Hosting + Integrations-Tiefe + Telephonie |

### Zusaetzliche Anforderungen fuer Standalone-Produkt

| Anforderung | Aufwand | Phase |
|-------------|---------|-------|
| Agent-Only Rolle (kein CRM-Zugang) | S | 2 |
| Campaign Manager Rolle | S | 2 |
| Self-Service Nummern-Provisioning | M | 2 |
| Per-Agent Call Quality Monitoring | M | 2 |
| API fuer Kampagnen-Injection (externe CRM) | M | 2 |
| White-Label / Custom Domain | L | 3 |
| Usage-Based Billing API | XL | 3 |
| Onboarding Wizard fuer neue Tenants | M | 3 |

---

## Fruehe Architektur-Entscheidungen (Jetzt treffen, spaeter bauen)

Diese Entscheidungen haben lange Auswirkungen — falsches Design bedeutet teures Refactoring.

### 1. `tenant_id` auf allen Dialer-Tabellen ab Phase 1
Auch bei Single-Tenant: `tenant_id uuid NOT NULL` auf jede Tabelle. Zero Overhead jetzt, aber Unterschied zwischen 2-Wochen vs. 3-Monaten Multi-Tenant-Migration.

### 2. Telefonnummern in E.164 Format
Alle Nummern als `+49301234567` speichern. Normalisierung on write. Ohne das scheitern DNC-Lookups ("030 1234567" != "+49301234567"). Migration fuer bestehende `contacts.phone` Werte.

### 3. Dialer Agent Status != Generic Presence
`dialer_agent_status` (available/on_call/wrap_up/break) **separater** Store von `UserPresence` (online/away/dnd/in_call/offline). Cross-Link: agent `on_call` -> presence `in_call`. Aber separate Stores — sonst koppelt man Presence-Performance an Dialer-Event-Rate.

### 4. Event-Log fuer Call State Machine
Jeder Call durchlaeuft: initiated -> ringing -> answered -> recording -> outcome -> wrap_up -> completed. Statt nur `status`-Column updaten: Transitions als Events in `dialer_call_events` appenden. Gibt: Audit-Log, Supervisor-Timeline, Analytics-Replay, Debugging gratis.

### 5. SIP Trunk Abstraction Interface
Go-Interface `SIPTrunkProvider` von Anfang an — ermoeglicht Provider-Wechsel und Multi-Provider (sipgate DE + Telnyx international) ohne Core-Logik anzufassen.

### 6. Predictive Dialer Legal Guard Rails
UWG § 7: max 3% Abandon Rate. Hard-Cap + Auto-Fallback auf 1:1 Ratio in `PredictiveEngine` struct designen, auch wenn erst Phase 3.

---

## Zusammenfassung

| Phase | Aufwand | Ergebnis | Markt-Impact |
|-------|---------|----------|--------------|
| **Phase 1 MVP** | M (6-10 Wo) | Click-to-Call + Kampagnen + Outcomes | Feature-Differenzierung im CRM-Markt |
| **Phase 2 PSTN** | XL (15-22 Wo) | Echter Power Dialer + HubSpot + Supervisor | Neues Produkt-Segment: CRM+Dialer |
| **Phase 3 Platform** | XXL (40-60 Wo) | Predictive + ACD + AI + Multi-Tenant | Contact-Center-Plattform, 4-16x ARR |

### Kritische Dateien

| Datei | Relevanz |
|-------|----------|
| `backend/internal/work/livekit/room_manager.go` | SIP-Erweiterung (Phase 2) |
| `backend/internal/gateway/registry.go` | Neuer `dialer` Service registrieren |
| `backend/internal/work/presence/models.go` | Cross-Link mit Dialer Agent Status |
| `backend/internal/automation/trigger/types.go` | Neue Dialer-Trigger registrieren |
| `backend/internal/crm/activity/service.go` | CRM-Bridge fuer Call-Outcome -> Activity |
| `backend/internal/gateway/route_bexio.go` | Pattern-Vorlage fuer HubSpot-Integration |

### Verifikation
- **Phase 1:** Kampagne erstellen -> Kontakte hinzufuegen -> Preview-Modus durchklicken -> Call-Outcome loggen -> Termin buchen -> Activity auf Kontakt-Timeline pruefen -> Dashboard-Zahlen validieren
- **Phase 2:** SIP-Call an echte Nummer -> Audio-Qualitaet testen -> Recording + Consent -> HubSpot-Sync pruefen -> DNC-Block testen -> Supervisor-View live
- **Phase 3:** Predictive Ratio unter Last testen -> Abandon Rate < 3% validieren -> Inbound ACD Routing -> AI Scoring Latenz messen -> Multi-Tenant Isolation verifizieren
