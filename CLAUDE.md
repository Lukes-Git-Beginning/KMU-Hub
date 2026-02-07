# CLAUDE.md - KMU Hub CRM

## Projekt-Kontext

- **Mission:** All-in-One CRM fuer DACH-KMUs mit EU-Datensouveraenitaet
- **USP:** Massanfertigung durch 1-Woche-Onsite-Prozessanalyse + Config/WASM-Plugin-System
- **Zielgruppe:** Branchenunabhaengige KMUs (5-200 Mitarbeiter)
- **Team:** 1 Dev + 2 Business, AI-First Development
- **Timeline:** 8-10 Monate bis Beta (AI-gestuetzte Entwicklung)
- **Version:** 0.1.0

---

## Tech-Stack

| Komponente | Technologie |
|-----------|-------------|
| Backend | Go (API Gateway + Microservices) |
| Desktop | Electron + React + TypeScript |
| Mobile | React Native |
| Datenbank | PostgreSQL + Redis |
| Video | LiveKit (self-hostable) |
| Plugins | Config-basiert + WASM (komplexe Erweiterungen) |
| Hosting | EU-only (Hetzner/OVH), SaaS + Self-Hosted Option |

---

## Entwicklungs-Kommandos

### Backend (Go)

```bash
# Entwicklung
cd backend
make run-gateway                    # API Gateway starten (Port 8080)
make run-crm                       # CRM Service starten
make run-chat                      # Chat Service starten
make run-auth                      # Auth Service starten

# Build & Test
make build                         # Alle Services bauen
make test                          # Alle Tests
make test-coverage                 # Mit Coverage-Report
make lint                          # golangci-lint

# Datenbank
make migrate-up                    # Migrations ausfuehren
make migrate-down                  # Letzte Migration zurueckrollen
make migrate-create name=xxx       # Neue Migration erstellen
```

### Desktop (Electron)

```bash
cd desktop
npm install                        # Dependencies installieren
npm run dev                        # Electron Dev-Modus
npm run build                      # Production Build
npm run test                       # Tests ausfuehren
npm run lint                       # ESLint
```

### Docker (Lokale Entwicklung)

```bash
cd deploy/docker
docker-compose up -d               # Alle Services starten
docker-compose logs -f gateway     # Logs eines Services
docker-compose down                # Alles stoppen
```

---

## Architektur-Regeln (KRITISCH)

### 1. Thick Services, Thin Handlers

Business-Logik gehoert in Services, NICHT in HTTP-Handler. Handler sind nur fuer:
- Request parsen / validieren
- Service aufrufen
- Response formatieren

```go
// RICHTIG
func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateContactRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, err)
        return
    }
    contact, err := h.contactService.Create(r.Context(), req)
    if err != nil {
        respondError(w, http.StatusInternalServerError, err)
        return
    }
    respondJSON(w, http.StatusCreated, contact)
}

// FALSCH - Business-Logik im Handler
func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
    // ... Validierung, DB-Zugriff, E-Mail senden direkt hier
}
```

### 2. Centralized Service Registry

Alle Services ueber zentrale Stelle initialisieren und injizieren. Kein `init()` Missbrauch, keine globalen Variablen.

```go
type ServiceRegistry struct {
    ContactService  *contact.Service
    DealService     *deal.Service
    AuthService     *auth.Service
    // ...
}
```

### 3. Structured Logging von Tag 1

Kein `fmt.Println()`, immer `slog` oder `zerolog`:

```go
// RICHTIG
slog.Info("contact created",
    "contact_id", contact.ID,
    "user_id", userID,
)

// FALSCH
fmt.Printf("Created contact %s\n", contact.ID)
log.Println("contact created")
```

### 4. API-First Design

OpenAPI Spec VOR Implementation schreiben. Code wird aus der Spec generiert oder gegen die Spec validiert.

### 5. Database Migrations

IMMER via Migration-Tool (golang-migrate), NIE manuell SQL auf der DB ausfuehren:

```bash
make migrate-create name=add_contacts_table
# -> backend/migrations/000001_add_contacts_table.up.sql
# -> backend/migrations/000001_add_contacts_table.down.sql
```

Index Naming Convention: `idx_{table}_{column}` (z.B. `idx_contacts_email`)

### 6. Test Coverage

- **Gesamt:** 80%+ Minimum
- **Kritische Pfade (Auth, Payments, Data):** 95%+
- **Jeder PR:** Muss Tests enthalten fuer neuen Code
- **Test-Isolation:** Jeder Test raeumt seine Daten auf, keine Abhaengigkeiten zwischen Tests

### 7. Security First

- Auth + Rate Limiting + Input Validation von Anfang an
- CSRF-Schutz fuer alle mutierenden Endpoints
- SQL-Injection: Immer Prepared Statements, nie String-Concatenation
- CORS: Explizite Allowlist, kein Wildcard
- Secrets: Immer ueber Environment Variables, nie im Code

### 8. Graceful Degradation

Services muessen unabhaengig ausfallen koennen. CRM muss funktionieren, auch wenn Chat offline ist.

### 9. Config ueber Environment Variables

```bash
# .env
DATABASE_URL=postgres://user:pass@localhost:5432/kmuhub
REDIS_URL=redis://localhost:6379
JWT_SECRET=...
LIVEKIT_URL=...
LIVEKIT_API_KEY=...
LIVEKIT_API_SECRET=...
```

Nie hardcoded. `.env` Datei NIE committen.

### 10. Idempotente Operationen

Alle API-Calls muessen sicher wiederholbar sein. Idempotency-Keys fuer POST-Requests.

---

## Haeufige Fehler (VERMEIDEN!)

Diese Fehler wurden im Vorgaenger-Projekt (slot_booking_webapp) gemacht und duerfen NICHT wiederholt werden:

1. **Business-Logik in Handlern** statt in Service-Layer schreiben
2. **Direkte DB-Zugriffe** statt ueber Service-Abstraktionsschicht
3. **`fmt.Println` / `console.log`** statt Structured Logging
4. **Dual-Write ohne Transaktionen** — im alten Projekt JSON+PostgreSQL parallel geschrieben, Nightmare. Hier: NUR PostgreSQL, Redis nur als Cache
5. **Templates/Komponenten kopieren** statt wiederverwenden — Komponenten-Library aufbauen
6. **Deployment ohne Backup** — IMMER Backup VOR jeder Aenderung
7. **CSS-Framework-Inkompatibilitaeten** — Tailwind JIT (Runtime) braucht `unsafe-eval`, inkompatibel mit CSP Nonces. Loesung: Tailwind IMMER pre-compilen
8. **Test-Isolation fehlt** — Patch-Pfade muessen dort sein wo importiert wird, nicht wo definiert. Keine verschachtelten Contexts
9. **Migration manuell** statt via Tool — fuehrt zu Drift zwischen Environments
10. **Service-Extension vergessen** — bestehende Services erweitern, KEINE neuen JSON-Dateien/Stores erstellen

---

## Deployment-Regeln

### Reihenfolge (KRITISCH)

1. Backup erstellen
2. Assets bauen (CSS, JS, Electron)
3. Config aktualisieren
4. Code deployen
5. Migrations ausfuehren
6. Services restarten
7. Health Check
8. Rollback-Plan bereithalten

### Self-Hosted (Kunden)

- Docker-Compose Setup fuer einfaches Deployment
- Automatische Backups via Cron
- Update-Mechanismus ueber Docker Image Tags

### SaaS

- Kubernetes auf Hetzner Cloud
- Blue-Green Deployment
- Automatische Skalierung

---

## Git-Regeln

- **Conventional Commits:** `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`
- **Branch-Strategie:** `main` -> `develop` -> `feature/*`, `fix/*`
- **PR-Pflicht:** Kein direkter Push auf `main` oder `develop`
- **Keine AI-Attribution** in Commits
- **Commit-Messages:** Englisch, imperativ ("Add contact endpoint", nicht "Added...")

---

## GSD Workflow

Dieses Projekt nutzt [Get Shit Done](https://github.com/glittercowboy/get-shit-done) fuer strukturierte AI-Entwicklung.

- Planung: `/gsd:discuss-phase` → `/gsd:plan-phase`
- Ausfuehrung: `/gsd:execute-phase`
- Verifikation: `/gsd:verify-work`
- Ad-hoc Fixes: `/gsd:quick`
- Status: `/gsd:progress`

Planungsdateien in `.planning/` werden committed.

---

## Weiterfuehrende Dokumentation

| Thema | Datei |
|-------|-------|
| Architektur-Entscheidungen (ADRs) | `docs/ARCHITECTURE.md` |
| Learnings aus Vorgaenger-Projekt | `docs/LEARNINGS.md` |
| Pricing-Modell | `docs/PRICING.md` |
| Roadmap & Phasen (Archiv) | `docs/ROADMAP.md` |
| GSD Projekt | `PROJECT.md` |
| GSD Requirements | `REQUIREMENTS.md` |
| GSD Roadmap | `ROADMAP.md` |
| GSD State | `STATE.md` |
