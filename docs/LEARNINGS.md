# Learnings aus slot_booking_webapp

Dieses Dokument haelt alle Fehler und Erkenntnisse aus dem Vorgaenger-Projekt fest, damit sie im KMU Hub nicht wiederholt werden.

---

## 1. Dual-Write Komplexitaet (JSON + PostgreSQL)

**Problem:** Daten wurden parallel in JSON-Dateien und PostgreSQL geschrieben. Migration war bei ~61% als das Projekt uebergeben wurde.

**Konsequenzen:**
- Permanente Inkonsistenz-Gefahr zwischen beiden Stores
- Jede neue Funktion musste beide Systeme beruecksichtigen
- Debugging extrem aufwaendig (welcher Store hat die "richtige" Version?)
- Migration zog sich ueber Monate

**Learning fuer KMU Hub:**
- NUR PostgreSQL als primaere Datenquelle
- Redis NUR als Cache, NICHT als Daten-Store
- Keine Dual-Write Patterns — eine Single Source of Truth

---

## 2. CSP Nonce vs. Tailwind JIT Inkompatibilitaet

**Problem:** Tailwind JIT (Runtime via `tailwind.min.js`) benoetigt `unsafe-eval` + `unsafe-inline`, was Content Security Policy (CSP) Nonces unmoeglich macht.

**Konsequenzen:**
- Wochen an Debugging und Fehlversuchen
- CSP-Header konnten nicht korrekt gesetzt werden
- Sicherheitsluecke durch noetige `unsafe-*` Direktiven

**Loesung die funktionierte:**
- Tailwind CSS pre-compilen: `npx tailwindcss -o static/tailwind-compiled.css --minify`
- Kein Runtime-Tailwind mehr

**Learning fuer KMU Hub:**
- Tailwind IMMER pre-compilen (Build-Step)
- Keine Runtime-CSS-Generierung
- CSP Nonces von Tag 1 korrekt implementieren

---

## 3. Service-Extension statt Neuerstellung

**Problem:** Bei neuen Features wurden oft neue JSON-Dateien oder separate Stores erstellt statt bestehende Services zu erweitern.

**Konsequenzen:**
- Fragmentierte Datenlandschaft
- Duplikation von Logik
- Schwer zu wartender Code

**Learning fuer KMU Hub:**
- Service-Registry Pattern von Anfang an
- Neue Features erweitern bestehende Services
- Kein neuer Daten-Store ohne explizite Architektur-Entscheidung (ADR)

---

## 4. Business-Logik in Routes/Handlern

**Problem:** Viel Business-Logik landete direkt in Flask-Route-Funktionen statt in dedizierten Service-Klassen.

**Konsequenzen:**
- Routes wurden hunderte Zeilen lang
- Logik nicht wiederverwendbar (z.B. zwischen API und Background-Jobs)
- Unit-Tests mussten HTTP-Layer mocken statt Services direkt zu testen

**Learning fuer KMU Hub:**
- Strikte Trennung: Handler -> Service -> Repository
- Handler darf nur: Request parsen, Service aufrufen, Response formatieren
- Alles andere gehoert in den Service-Layer

---

## 5. Test-Fixture Patterns

**Problem:** Test-Failures durch falsche Patch-Pfade und Context-Nesting.

**Konkrete Fehler:**
- Patch-Pfade muessen dort sein wo die Funktion IMPORTIERT wird, nicht wo sie definiert ist
  - `@patch('app.routes.hub.data_persistence')` NICHT `@patch('app.core.extensions.data_persistence')`
- Flask `client` Fixture oeffnet bereits einen Context → verschachteltes `with client:` = RuntimeError
- `remove_closer()` hatte Bug weil `load_bucket_data()` die T2_CLOSERS re-synced hat

**Learning fuer KMU Hub:**
- Go Tests: Table-Driven Tests als Standard-Pattern
- Test-Isolation: Jeder Test hat eigene DB-Transaction (Rollback nach Test)
- Keine globalen Variablen die zwischen Tests leaken
- Integration Tests mit eigenem Test-Container (testcontainers-go)

---

## 6. Deployment-Reihenfolge

**Problem:** Deployment-Reihenfolge war kritisch und wurde oft falsch gemacht.

**Korrekte Reihenfolge (slot_booking_webapp):**
1. CSS kompilieren
2. Templates deployen
3. Python-Code deployen
4. Service restarten
5. Health Check

**Falsche Reihenfolge fuehrte zu:**
- Blank Pages (Templates referenzieren nicht-existierendes CSS)
- 500er Errors (Code referenziert nicht-existierende Templates)
- Cache-Probleme

**Learning fuer KMU Hub:**
- Deployment IMMER automatisiert (CI/CD Pipeline)
- Blue-Green Deployment: Neuer Container hochfahren, Health Check, dann Traffic umleiten
- Manuelle Deployments nur im Notfall

---

## 7. Template-Vererbung und Framework-Konsistenz

**Problem:** CLAUDE.md behauptete "Bootstrap 5.3.2", aber alle Templates nutzten bereits Tailwind/DaisyUI. Fuehrte zu Verwirrung und falschen Aenderungen.

**Learning fuer KMU Hub:**
- CLAUDE.md IMMER aktuell halten
- Ein CSS-Framework pro Projekt, keine Mischung
- Component Library von Anfang an aufbauen

---

## 8. Fehlende strukturierte Logs

**Problem:** Mix aus `print()`, `app.logger`, und `logging.getLogger()` machte Log-Analyse schwierig.

**Learning fuer KMU Hub:**
- `slog` (Go stdlib) als einziger Logger
- Structured Fields (key-value), kein String-Formatting
- Correlation IDs fuer Request-Tracing ueber Services hinweg

---

## 9. systemd Hardening

**Gelernt im Vorgaenger:**
- `ReadWritePaths=` explizit setzen — neue Verzeichnisse vergessen = Permission Denied
- `ProtectSystem=full` kann unerwartete Probleme verursachen

**Learning fuer KMU Hub:**
- Docker-Container statt systemd fuer Isolation
- File-System Permissions im Dockerfile definieren
- Non-Root User im Container

---

## 10. Google Calendar / External API Integration

**Problem:** Google Calendar Aufrufe ohne Error-Handling fuehrten zu 500er Errors wenn die API nicht erreichbar war.

**Learning fuer KMU Hub:**
- Jeder externe API-Call: try/catch + Retry mit Exponential Backoff
- Circuit Breaker Pattern fuer externe Services
- `DRY_RUN` Flag fuer alle externen Write-Operationen in Development
- Fallback-Verhalten definieren (Was passiert wenn LiveKit down ist?)
