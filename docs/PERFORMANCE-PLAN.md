# Performance-Optimierungsplan — Cosmi

> **Status:** Geplant (2026-04-08)
> **Geschaetzter Aufwand:** ~8 Arbeitstage
> **Grundlage:** Codebase-Audit (17 Issues: 3x P0, 4x P1, 5x P2, 5x P3) + Best-Practice-Recherche

---

## Ausgangslage

Die Codebase-Analyse hat konkrete Performance-Issues in Frontend, Backend und Build identifiziert. Der Plan ist in 5 Phasen gegliedert, von Quick Wins bis zu groesseren Architektur-Verbesserungen.

### Audit-Ergebnisse (Kurzfassung)

| Bereich | Kritischste Findings |
|---------|---------------------|
| **Backend** | N+1 Queries (61-121 DB-Queries pro Listenaufruf), Connection Pool Exhaustion (250 vs. 100 max), fehlende Indexes |
| **Frontend** | Google Fonts ueber CDN in Electron, 9k Zeilen Demo-Mode Dead Code im Prod-Bundle, kein Chunk-Splitting, keine List-Virtualisierung (ausser Chat) |
| **Build** | Kein Bundle-Analyzer, kein `manualChunks`, `modulePreload: false` |
| **Gateway** | Unbegrenzte Goroutines im Audit Logger, kein Redis-Caching fuer Read-Heavy Endpoints |

---

## Phase 1 — Analyse & Quick Wins (Tag 1)

### 1.1 Bundle-Analyse aufsetzen

- `rollup-plugin-visualizer` als devDependency installieren
- In `electron.vite.config.ts` einbinden (Plugin-Array)
- Einmal laufen lassen → Baseline-Report als HTML generieren
- `build.chunkSizeWarningLimit: 250` setzen (statt default 500kb)

**Dateien:** `desktop/package.json`, `desktop/electron.vite.config.ts`

### 1.2 Google Fonts selbst hosten (P0)

**Problem:** `index.html` laedt Plus Jakarta Sans + JetBrains Mono ueber Google CDN. In Electron Production (`file://` Protokoll) = Netzwerk-Roundtrip bei jedem Kaltstart, offline komplett kaputt.

**Fix:**
1. WOFF2-Dateien herunterladen (Plus Jakarta Sans: 400/500/600/700 + Italic 400/500, JetBrains Mono: 400)
2. Ablage in `src/renderer/public/fonts/`
3. `@font-face` Deklarationen mit `font-display: swap` erstellen
4. CDN-Links + Preconnect aus `index.html` entfernen
5. CSP `font-src` anpassen: `https://fonts.gstatic.com` entfernen

**Dateien:** `desktop/src/renderer/index.html`, neue CSS-Datei oder in `globals.css`

### 1.3 Demo-Mode Dead Code eliminieren (P0)

**Problem:** `mocks/demo-mode.ts` importiert ~9.000 Zeilen Mock-Daten/Handler statisch. Vite tree-shaked `import`-Statements NICHT, auch wenn der Code-Pfad nie erreicht wird. Der Kommentar im Code ("Vite dead-code eliminates everything after the early return") ist **falsch**.

**Fix:** Static `import { handlers }` → Dynamic `const { handlers } = await import('./handlers')` innerhalb des `if (DEMO_MODE)` Blocks.

**Dateien:** `desktop/src/renderer/src/mocks/demo-mode.ts`

### 1.4 Framer Motion → LazyMotion (Bundle -30kb)

**Problem:** Volle `motion`-Komponente = ~34kb gzipped. Mit `LazyMotion` + `m` Component → 4.6kb initial.

**Fix:**
1. `LazyMotion` Provider mit `features={domAnimation}` einrichten
2. `motion.div` → `m.div` in animierten Komponenten
3. Features werden async geladen: `const domAnimation = () => import('motion/dom')`

**Dateien:** `desktop/src/renderer/src/App.tsx` (Provider), alle Komponenten die `motion.*` verwenden

---

## Phase 2 — Frontend-Optimierung (Tag 2-3)

### 2.1 Vite Chunk-Splitting Strategie (P1)

**Problem:** Kein `manualChunks` konfiguriert — ein riesiger Vendor-Chunk mit allen Dependencies.

**Fix:** `electron.vite.config.ts` → `rollupOptions.output.manualChunks`:

```js
manualChunks: {
  'vendor-react': ['react', 'react-dom', 'react-router-dom'],
  'vendor-editor': ['@tiptap/react', '@tiptap/starter-kit', '@tiptap/extension-*', 'lowlight'],
  'vendor-video': ['@livekit/components-react', 'livekit-client'],
  'vendor-query': ['@tanstack/react-query', '@tanstack/react-query-persist-client'],
  'vendor-ui': ['lucide-react', 'motion'],  // radix via shadcn ist tree-shaked
}
```

**Impact:** Stabile Vendor-Chunks die sich zwischen Deploys nicht aendern. LiveKit (~400-600kb) und TipTap (~150kb) werden nur geladen wenn das jeweilige Modul besucht wird.

**Dateien:** `desktop/electron.vite.config.ts`

### 2.2 React Query Persister → Async (P2)

**Problem:** `createSyncStoragePersister` serialisiert den gesamten Query-Cache synchron auf dem Main Thread nach jedem Cache-Write. Bei `GC_TIME = 24h` waechst der localStorage-Blob.

**Fix:** `createSyncStoragePersister` → `createAsyncStoragePersister` mit IndexedDB (`idb-keyval`).

**Dateien:** `desktop/src/renderer/src/App.tsx` (Persister-Setup)

### 2.3 HR Polling entschaerfen (P2)

**Problem:** `hr-hooks.ts:238` — `refetchInterval: 30_000` feuert alle 30 Sekunden, egal ob User auf der HR-Seite ist.

**Fix:** `refetchInterval` nur aktiv wenn HR-Modul sichtbar (`enabled`-Option), oder Intervall auf 5 Minuten erhoehen.

**Dateien:** `desktop/src/renderer/src/api/hooks/hr-hooks.ts`

### 2.4 React Compiler evaluieren

React Compiler 1.0 (seit Oktober 2025) macht automatisches Memoization — 30-60% weniger Re-Renders.

**Vorgehen:**
1. `babel-plugin-react-compiler` installieren
2. Im `annotation`-Mode starten (`compilationMode: 'annotation'`)
3. `"use memo"` Directive auf 3-5 schwere Komponenten (Dashboard, DealPipelineView, ContactsList)
4. Performance messen (React DevTools Profiler)
5. Wenn stabil: auf gesamte App ausweiten
6. Manuelle `useMemo`/`useCallback` entfernen (werden zu Noise)

**Dateien:** `desktop/electron.vite.config.ts` (Babel-Plugin), Komponenten mit `"use memo"` Directive

### 2.5 List Virtualization fuer CRM-Listen

**Problem:** Nur Chat Messages sind virtualisiert (`@tanstack/react-virtual`). Contacts, Deals, Tasks, Emails — alles plain `.map()`. Aktuell durch Pagination (PAGE_SIZE=20) abgemildert, aber kein Schutz bei groesseren Datenmengen.

**Fix:** `@tanstack/react-virtual` (bereits im Projekt fuer Chat) auf diese Komponenten anwenden:

| Komponente | Datei | Prioritaet |
|-----------|-------|-----------|
| ContactsListPage | `modules/crm/contacts/ContactsListPage.tsx` | Hoch |
| DealPipelineView | `modules/crm/deals/DealPipelineView.tsx` | Hoch |
| TasksList | `modules/work/` | Mittel |
| EmailList | `modules/mails/` | Niedrig |

**Dateien:** Jeweilige Modul-Komponenten

---

## Phase 3 — Backend N+1 Queries & DB (Tag 3-5)

### 3.1 N+1 Queries beheben (P0 — kritischstes Backend-Issue)

**Problem Contact List:**
- `crm/contact/service.go` → `getWithRelations()` pro Contact = 3 Extra-Queries
- CompanyName (`SELECT name FROM companies WHERE id = $1`)
- Tags (`SELECT ... FROM tags JOIN contact_tags WHERE contact_id = $1`)
- CustomFields (`SELECT ... FROM contact_custom_field_values WHERE contact_id = $1`)
- **Bei 20 Contacts = 61 Queries pro Seitenaufruf**

**Problem Deal List:**
- `crm/deal/service.go` → `getWithRelations()` = bis zu 6 Extra-Queries pro Deal
- StageName, ContactName, CompanyName, OwnerName, Tags, CustomFields
- **Bei 20 Deals = 121 Queries pro Seitenaufruf**

**Fix — Batch-Loading Pattern:**

```
Contacts:
1. SELECT contacts ... LIMIT 20                                    (1 Query)
2. SELECT id, name FROM companies WHERE id = ANY($1)               (1 Query, alle company_ids)
3. SELECT * FROM contact_tags WHERE contact_id = ANY($1)           (1 Query, alle contact_ids)
4. SELECT * FROM contact_custom_field_values WHERE contact_id = ANY($1)  (1 Query)
→ Total: 4 Queries statt 61

Deals:
1. SELECT deals ... LIMIT 20                                       (1 Query)
2. SELECT id, name FROM pipeline_stages WHERE id = ANY($1)         (1 Query)
3. SELECT id, first_name, last_name FROM contacts WHERE id = ANY($1) (1 Query)
4. SELECT id, name FROM companies WHERE id = ANY($1)               (1 Query)
5. SELECT id, first_name, last_name FROM users WHERE id = ANY($1)  (1 Query)
6. SELECT * FROM deal_tags WHERE deal_id = ANY($1)                 (1 Query)
7. SELECT * FROM deal_custom_field_values WHERE deal_id = ANY($1)  (1 Query)
→ Total: 7 Queries statt 121
```

**Dateien:**
- `backend/internal/crm/contact/service.go` (List + getWithRelations refactoren)
- `backend/internal/crm/contact/postgres_repository.go` (neue Batch-Query Methoden)
- `backend/internal/crm/deal/service.go` (gleiches Pattern)
- `backend/internal/crm/deal/postgres_repository.go` (neue Batch-Query Methoden)

### 3.2 Batch-Inserts fuer Tags & Custom Fields (P1)

**Problem:**
- `contact/postgres_repository.go:343` — ein INSERT pro Tag in einer Schleife
- `deal/postgres_repository.go:238` — gleiches Pattern
- `contact/postgres_repository.go:391` — ein UPSERT pro Custom Field Value

**Fix:**

```sql
-- Statt N einzelne INSERTs:
INSERT INTO contact_tags (contact_id, tag_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT DO NOTHING;

-- Statt N einzelne UPSERTs:
INSERT INTO contact_custom_field_values (contact_id, field_id, value)
SELECT $1, unnest($2::uuid[]), unnest($3::text[])
ON CONFLICT (contact_id, field_id) DO UPDATE SET value = EXCLUDED.value;
```

**Dateien:**
- `backend/internal/crm/contact/postgres_repository.go`
- `backend/internal/crm/deal/postgres_repository.go`

### 3.3 Fehlender Index `contacts.owner_id` (P2)

**Problem:** `ListWithVisibility` filtert auf `owner_id`, aber kein Index vorhanden (Deals hat `idx_deals_owner`).

**Fix:** Neue Migration:

```sql
CREATE INDEX idx_contacts_owner_id ON contacts (owner_id);
```

**Dateien:** `backend/migrations/` (neue Migration-Datei)

### 3.4 Connection Pool Tuning (P1)

**Problem:** 10 Services × `MaxConns=25` = 250 potenzielle Postgres-Connections. PostgreSQL default `max_connections=100`.

**Fix Sofort:**

```go
config.MaxConns = 10              // statt 25 (= 100 total bei 10 Services)
config.MinConns = 2
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute
config.HealthCheckPeriod = time.Minute
```

**Fix Mittelfristig:** PgBouncer als Sidecar-Container in Docker Compose:
- Transaction Mode
- Pool von 20-50 echten Connections
- Alle Services verbinden sich zu PgBouncer statt direkt zu PostgreSQL

**Dateien:** `backend/internal/database/postgres.go`, spaeter `deploy/docker/docker-compose.yml`

### 3.5 PostgreSQL Tuning (Hetzner CPX42, 16GB RAM)

Anpassungen in `postgresql.conf` bzw. Docker-Compose Environment:

```
shared_buffers = 4GB          # 25% von 16GB
effective_cache_size = 12GB   # Planer-Hinweis fuer OS-Cache
work_mem = 64MB               # Fuer Sorts/Joins
```

Partial Indexes evaluieren:

```sql
CREATE INDEX idx_deals_open ON deals (owner_id) WHERE status = 'open';
```

**Dateien:** `deploy/docker/docker-compose.prod.yml` (PostgreSQL Config)

---

## Phase 4 — Electron & Startup (Tag 5-6)

### 4.1 V8 Compile Cache

**Problem:** Jeder Kaltstart kompiliert JavaScript-Sourcen zu V8-Bytecode.

**Fix:** `v8-compile-cache` installieren — cached kompilierten Bytecode auf Disk. Typisch 20-40% schnellerer Kaltstart.

**Dateien:** `desktop/src/main/index.ts` (require am Anfang)

### 4.2 `modulePreload` wieder aktivieren

**Problem:** `modulePreload: false` in Vite Config deaktiviert `<link rel="modulepreload">` Tags. Lazy-Chunks werden erst bei Navigation geladen statt parallel vorgeladen.

**Fix:** `modulePreload: false` entfernen oder auf `true` setzen.

**Dateien:** `desktop/electron.vite.config.ts`

### 4.3 Skeleton Screen

**Problem:** Weisser Screen waehrend React-Initialisierung.

**Fix:** Statisches HTML/CSS Layout-Shell in `index.html` einbauen:
- Sidebar-Silhouette + Topbar + Content-Bereich als reines HTML/CSS
- Verschwindet sobald React hydrated (`#root` ersetzt Skeleton)

**Dateien:** `desktop/src/renderer/index.html`

---

## Phase 5 — Gateway & Caching (Tag 6-8)

### 5.1 Audit Logger optimieren (P2)

**Problem:** `middleware/audit.go:31` — `go a.logEvent(...)` spawnt eine unbegrenzte Goroutine pro mutierendem Request mit gRPC-Call. Unter Last = Goroutine-Explosion.

**Fix:** Buffered Channel (Kapazitaet ~1000) + Worker Pool (10 Worker):

```go
type AuditLogger struct {
    events chan AuditEvent
}

func (a *AuditLogger) Start(workers int) {
    for i := 0; i < workers; i++ {
        go a.worker()
    }
}

func (a *AuditLogger) worker() {
    for event := range a.events {
        a.logEvent(event)
    }
}
```

**Dateien:** `backend/cmd/gateway/middleware/audit.go`

### 5.2 Redis Caching Layer

Stufenweise Einfuehrung von Cache-Aside Pattern:

| Tier | Was | TTL | Aufwand |
|------|-----|-----|---------|
| 1 | Org-Settings, Feature Flags, User-Profile-Lookups | 2-5 Min | Klein |
| 2 | Dashboard-Aggregationen (Pipeline-Totals, Contact-Counts) | 30-60s + Jitter | Mittel |
| 3 | Paginated List-Results fuer haeufige Sorts | 15-30s | Gross |

**Key-Schema:** `org:{org_id}:contacts:page:{n}:sort:{field}`

**Invalidierung:** Event-driven — bei jedem Write ein `DEL` auf betroffene Cache-Keys.

**Anti-Stampede:** TTL-Jitter von ±10-20% um gleichzeitiges Expiry zu vermeiden.

**Dateien:** Neues Package `backend/internal/cache/` mit generischem Cache-Aside Helper

### 5.3 gRPC Connection Management

- Verifizieren dass Gateway gRPC-Connections als Singletons haelt (nicht per-Request)
- Keep-Alive Tuning:

```go
keepalive.ClientParameters{
    Time:                60 * time.Second,
    Timeout:             10 * time.Second,
    PermitWithoutStream: true,
}
```

**Dateien:** `backend/cmd/gateway/main.go` (gRPC Client-Setup)

### 5.4 pprof Profiling einbauen

- Chi `middleware.Profiler()` auf `/debug/pprof` hinter Auth-Middleware mounten
- Ermoeglicht Heap/CPU-Profiling in Development und Staging
- Nicht in Production exposen (Feature-Flag oder Build-Tag)

**Dateien:** `backend/cmd/gateway/main.go`

---

## Zusammenfassung nach Impact

| Prio | Issue | Phase | Aufwand | Impact |
|------|-------|-------|---------|--------|
| P0 | N+1 Queries (61-121 pro Seite) | 3 | 1-2 Tage | **Sehr hoch** |
| P0 | Google Fonts selbst hosten | 1 | 30 Min | **Hoch** (Offline + Startup) |
| P0 | Demo-Mode Dead Code (9k Zeilen) | 1 | 30 Min | **Hoch** (Bundle Size) |
| P1 | Vite Chunk-Splitting | 2 | 2 Std | **Hoch** (Initial Load) |
| P1 | Connection Pool Fix | 3 | 1 Std | **Hoch** (Stabilitaet) |
| P1 | Batch-Inserts Tags/CustomFields | 3 | 3 Std | **Mittel** (Write-Perf) |
| P2 | React Query Async Persister | 2 | 2 Std | **Mittel** (Main Thread) |
| P2 | Fehlender contacts.owner_id Index | 3 | 15 Min | **Mittel** (Query-Perf) |
| P2 | Audit Logger Worker Pool | 5 | 2 Std | **Mittel** (Gateway-Perf) |
| P2 | HR Polling 30s → 5min | 2 | 15 Min | **Niedrig** |
| — | React Compiler | 2 | 1 Tag | **Hoch** (Re-Renders) |
| — | LazyMotion | 1 | 1 Std | **Mittel** (Bundle -30kb) |
| — | List Virtualization | 2 | 1 Tag | **Mittel** (Skalierbarkeit) |
| — | PgBouncer | 3 | 0.5 Tage | **Hoch** (DB-Connections) |
| — | Redis Caching Layer | 5 | 2-3 Tage | **Hoch** (API-Latenz) |
| — | V8 Compile Cache | 4 | 30 Min | **Mittel** (Startup) |
| — | Skeleton Screen | 4 | 2 Std | **Niedrig** (Perceived Perf) |

---

## Quellen & Referenzen

- [React Compiler 1.0 — React Blog](https://react.dev/blog/2025/10/07/react-compiler-1)
- [Electron Performance — Official Docs](https://www.electronjs.org/docs/latest/tutorial/performance)
- [LazyMotion — Motion.dev](https://motion.dev/docs/react-lazy-motion)
- [PgBouncer — Percona Guide](https://www.percona.com/blog/pgbouncer-for-postgresql-how-connection-pooling-solves-enterprise-slowdowns/)
- [gRPC Performance Best Practices](https://grpc.io/docs/guides/performance/)
- [Redis Cache Optimization](https://redis.io/blog/guide-to-cache-optimization-strategies/)
- [Vite Bundle Optimization](https://www.mykolaaleksandrov.dev/posts/2025/11/taming-large-chunks-vite-react/)

---

*Erstellt: 2026-04-08*
