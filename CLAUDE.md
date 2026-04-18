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
| Mobile | PWA (Phase E, auf Desktop-Basis) |
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

- **Gesamt:** 15%+ Minimum (CI-enforced), Ziel 40% bis Q3 2026
- **Kritische Pfade (Auth, Payments, Data):** 60%+ Ziel
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

### 11. Tenant-Modell

Cosmi ist aktuell **Single-Tenant-only**. Multi-Tenant-Support (tenant_id auf allen Tabellen) ist fuer Phase 3 geplant. Bis dahin:
- Kein SaaS mit mehreren Mandanten auf einer DB-Instanz
- Self-Hosted: Ein Deployment pro Kunde
- Neue Tabellen MUESSEN tenant_id von Anfang an haben

### Knowledge Base (.knowledge/)

- Projektspezifisches Wissen liegt in `.knowledge/` (Obsidian Vault)
- Claude nutzt MCP Filesystem-Tools zum Lesen/Schreiben von Notes
- Neue Erkenntnisse in passende `.knowledge/*.md` Note schreiben
- Jede Note hat YAML Frontmatter: `tags`, `updated`
- Notes untereinander verlinken mit `[[note-name]]`
- User kann den Vault in Obsidian oeffnen (Graph-View, eigene Notes)

---

## UI/UX Design-Directives (KRITISCH)

### Aesthetic Direction

KMU Hub strebt **"Premium SaaS mit Editorial Touch"** an — kein generisches Dashboard-Look. Die UI soll bei jedem Screen ein "Wow" ausloesen. Wir nutzen AI-Design-Skills (frontend-design, impeccable) und halten uns an diese Regeln:

### Font-Bans (NIEMALS verwenden)

- **VERBOTEN:** Inter, Roboto, Arial, Space Grotesk, Helvetica, Open Sans
- **Erlaubt (Display):** Plus Jakarta Sans (aktuell), Clash Display, Satoshi, Cabinet Grotesk, Bricolage Grotesque
- **Erlaubt (Mono):** JetBrains Mono (aktuell), Fira Code
- **Erlaubt (Editorial/Akzente):** Playfair Display, Fraunces, Crimson Pro
- **Prinzip:** Font-Pairing mit hohem Kontrast (Display + Mono, Serif + Geometric Sans). Gewichts-Extreme: 200 vs 800. Groessen-Spruenge 3x+, nicht 1.5x.

### Farb-Hierarchie

- **RICHTIG:** Dominante Primaerfarbe + scharfe Akzente. Klare Hierarchie: 1 Hero-Farbe dominiert, 1-2 Akzente punktuell.
- **FALSCH:** Gleichmaessig verteilte Paletten ohne visuelle Hierarchie. Timide, "sichere" Neutraltoene ueberall.
- CSS-Variablen fuer Konsistenz: `var(--primary)`, `var(--accent-1)`, `var(--accent-2)`
- Status-Farben NUR fuer Status (Success, Warning, Error, Info)

### Motion & Animation

- **RICHTIG:** Ein orchestrierter Page-Load mit gestaffelten Reveals (Stagger-System nutzen). Bewusste, bedeutungsvolle Animationen.
- **FALSCH:** Verstreute Micro-Interactions ohne Zusammenhang. Alles animieren "weil es geht".
- Page-Transitions: `cubic-bezier(0.22, 1, 0.36, 1)` (unser Standard)
- Dauer: 200-350ms fuer Interactions, 400-600ms fuer Page-Transitions
- `prefers-reduced-motion` IMMER respektieren

### Hintergruende & Tiefe

- **RICHTIG:** Atmosphaere und Tiefe erzeugen (Glass-Effekte, subtile Patterns, Schatten-Hierarchie)
- **FALSCH:** Flache Solid-Color-Hintergruende ohne Dimensionalitaet
- Unser 3-Look-System nutzen: Solid, Glass (blur 18px), Crystal (blur 12px)
- Schatten-Hierarchie: `--shadow-card` (default) -> `--shadow-card-hover` (elevated)

### Anti-Patterns (VERBIETEN)

1. **Card-in-Card Nesting** — Maximal 1 Ebene, keine verschachtelten Karten
2. **Grauer Text auf farbigem Hintergrund** — Kontrast-Ratio WCAG AA minimum
3. **Bootstrap-Aera Patterns** — Keine generischen Grid-Karten ohne Persoenlichkeit
4. **Symmetrische, vorhersagbare Layouts** — Visuelle Spannung durch bewusste Asymmetrie
5. **"AI Slop" Aesthetik** — Lila Gradienten, generische Icons, identische Card-Hoehen
6. **Ueber-Animation** — Nicht alles muss bouncen. Weniger ist mehr, aber das Wenige muss sitzen.

### Design-Workflow (Iterativ)

1. **Generate** — UI erstellen mit klarer Aesthetic Direction
2. **`/audit`** — Qualitaets-Check (P0-P3 Severity)
3. **`/critique`** — UX-Review gegen Nielsen's 10 Heuristiken
4. **Fix** — Gefundene Issues beheben
5. **`/polish`** — Final Pass fuer Perfektion
6. **Ship**

### Installierte Design-Skills

| Skill | Commands | Zweck |
|-------|----------|-------|
| **frontend-design** | (auto-loaded) | Aesthetic Direction, Font-Bans, Bold Design |
| **impeccable** | `/audit`, `/critique`, `/polish`, `/animate`, `/normalize`, `/bolder`, `/overdrive` + 14 mehr | Strukturiertes Design-Review & Verbesserung |

### Component Libraries (Copy-Paste)

- **shadcn/ui** (Radix) — Base Components (bereits integriert)
- **Magic UI** (magicui.design) — 150+ animierte Komponenten fuer Wow-Factor
- **Aceternity UI** (ui.aceternity.com) — 3D Cards, Parallax, Spotlight-Effekte

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
- **Branch-Strategie:** Ab Sprint 1 (2026-04-18) **direct-to-main ist Default** — keine Feature-Branches, keine PRs, ausser der User fordert explizit einen PR an. Sprint 0 lief noch mit PRs; ab jetzt direkt auf `main`.
- **CI-Rot-Recovery:** `git revert <sha>` (erzeugt neuen Commit). **NIE** `reset --hard`, **NIE** Force-Push.
- **Keine AI-Attribution** in Commits
- **Commit-Messages:** Englisch, imperativ ("Add contact endpoint", nicht "Added...")

---

## Weiterfuehrende Dokumentation

| Thema | Datei |
|-------|-------|
| Architektur-Entscheidungen (ADRs) | `docs/ARCHITECTURE.md` |
| Learnings aus Vorgaenger-Projekt | `docs/LEARNINGS.md` |
| Pricing-Modell | `docs/PRICING.md` |
| Roadmap & Phasen (Archiv) | `docs/ROADMAP.md` |
