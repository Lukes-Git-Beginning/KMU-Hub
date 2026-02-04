# KMU Hub

All-in-One CRM fuer DACH-KMUs mit EU-Datensouveraenitaet.

## Vision

CRM-System, das durch 1-Woche-Onsite-Prozessanalyse massgeschneidert wird. Konfigurationsbasierte Anpassung + WASM-Plugin-System fuer komplexe Erweiterungen.

## Tech-Stack

- **Backend:** Go (Microservices + API Gateway)
- **Desktop:** Electron + React + TypeScript
- **Mobile:** React Native
- **Datenbank:** PostgreSQL + Redis
- **Video:** LiveKit
- **Hosting:** EU-only (Hetzner/OVH)

## Quickstart

```bash
# Backend
cd backend && make build && make run-gateway

# Desktop
cd desktop && npm install && npm run dev

# Docker (alles zusammen)
cd deploy/docker && docker-compose up -d
```

## Dokumentation

Siehe `CLAUDE.md` fuer Entwicklungs-Richtlinien und `docs/` fuer detaillierte Dokumentation.
