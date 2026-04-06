---
tags: [troubleshooting, debug]
updated: 2026-04-06
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
- **Rebuild nach Code-Änderung:** `docker-compose build <service> && docker-compose up -d <service>`

## Windows/Dev-Umgebung
- **protoc Pfad:** `C:/Users/Luke/AppData/Local/Microsoft/WinGet/Packages/.../protoc.exe`
- **Go Pfad:** `export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"`
- **GitHub CLI:** `"C:/Program Files/GitHub CLI/gh.exe"`
- **Shell:** Git Bash (Unix-Syntax, nicht Windows CMD)

## golangci-lint
- Version 2 erfordert `version: "2"` in `.golangci.yml`
- `goimports` aus Formatters entfernt (CI-Issues)
- Action: golangci-lint v2.8 (action v7)

## Radix Dialog Null-Access Pattern
- Radix Dialog rendert `<DialogContent>` im DOM auch wenn `open={false}`
- Alle Zugriffe auf Dialog-State im Content muessen null-safe sein
- **Pattern:** `showDialog?.property` oder `{showDialog && ...}` im DialogContent
- Betraf: EinkaufPage, FormularePage, ZustandsprotokollDialog (alle gefixt 2026-04-01)

## useMemo Scope-Fehler
- Variablen die INNERHALB von `useMemo()` deklariert werden sind AUSSERHALB nicht verfügbar
- Wenn JSX auf diese Variablen zugreift → `ReferenceError: x is not defined`
- **Fix:** Variable im Return-Objekt des useMemo zurückgeben
- Betraf: CalendarUpcoming (today/dd), MyCalendar (now) (gefixt 2026-04-01)

## Häufige Fehler
- `fmt.Println` / `console.log` statt Structured Logging → slog verwenden
- Hardcoded Secrets → Environment Variables
- CORS Wildcard → Explizite Allowlist
- Deployment ohne Backup → IMMER zuerst Backup

## i18n Migration — Lessons Learned
- **Agent Token-Limits:** Massen-Instrumentierung (200+ Dateien) ueberschreitet Kontext — Waves von 30-50 Dateien, separate Commits
- **JSON-Extraktion trennen:** Erst Schluessel in additions/*.json extrahieren, dann useTranslation/t()-Calls einfuegen — reduziert Merge-Konflikte
- **`keySeparator: false` ist kritisch:** Ohne diese Option wuerde `"crm.contacts.title"` als nested Object geparst — immer explizit setzen
- **Marken-Namen nicht uebersetzen:** "Cosmi", "Zentria" nie in `t()` wrappen
- **ICU-Syntax:** i18next-icu verwendet `{count, plural, one {…} other {…}}` — nicht react-intl's `=1 {…}` Notation

## Verwandte Notes
- [[architektur]] — Architektur-Regeln
- [[i18n]] — i18n-Architektur & Konventionen
- [[deployment]] — Docker & CI/CD
- [[stack]] — Dev-Tooling & Pfade
