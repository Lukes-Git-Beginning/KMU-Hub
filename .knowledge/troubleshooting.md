---
tags: [troubleshooting, debug]
updated: 2026-03-05
---
# Troubleshooting & Bekannte Probleme

## Architektur-Fehler (NICHT wiederholen)
Aus Vorgaenger-Projekt (slot_booking_webapp) gelernt:

- **Dual-Write vermeiden** — NUR PostgreSQL, Redis = Cache. Nie JSON+DB parallel
- **Business-Logik in Services** — Nicht in Handlern, nicht in DB-Queries
- **Service erweitern** — Bestehende Services erweitern, KEINE neuen Stores/JSON-Files
- **Migrations via Tool** — `make migrate-create`, nie manuelles SQL
- **Komponenten wiederverwenden** — Nicht kopieren, Component-Library nutzen

## Tailwind + CSP
- Tailwind JIT (Runtime) braucht `unsafe-eval` → inkompatibel mit CSP Nonces
- **Loesung:** Tailwind v4 IMMER pre-compilen (Vite Plugin, nicht Runtime)
- Aktuell korrekt konfiguriert in `electron.vite.config.ts`

## Test-Patterns
- Patch-Pfade muessen dort sein wo importiert wird, nicht wo definiert
- Keine verschachtelten Contexts
- Test-Isolation: Jeder Test raeumt seine Daten auf

## Docker Compose
- **Reihenfolge:** Services haben `depends_on` mit Health-Check-Conditions
- **Health-Check-Timeout:** OnlyOffice braucht bis zu 60s Start-Period
- **Volumes:** `pgdata` und `minio_data` persistent, `docker-compose down -v` loescht alles
- **Rebuild nach Code-Aenderung:** `docker-compose build <service> && docker-compose up -d <service>`

## Windows/Dev-Umgebung
- **protoc Pfad:** `C:/Users/Luke/AppData/Local/Microsoft/WinGet/Packages/.../protoc.exe`
- **Go Pfad:** `export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"`
- **GitHub CLI:** `"C:/Program Files/GitHub CLI/gh.exe"`
- **Shell:** Git Bash (Unix-Syntax, nicht Windows CMD)

## golangci-lint
- Version 2 erfordert `version: "2"` in `.golangci.yml`
- `goimports` aus Formatters entfernt (CI-Issues)
- Action: golangci-lint v2.8 (action v7)

## Haeufige Fehler
- `fmt.Println` / `console.log` statt Structured Logging → slog verwenden
- Hardcoded Secrets → Environment Variables
- CORS Wildcard → Explizite Allowlist
- Deployment ohne Backup → IMMER zuerst Backup

## Verwandte Notes
- [[architektur]] — Architektur-Regeln
- [[deployment]] — Docker & CI/CD
- [[stack]] — Dev-Tooling & Pfade
