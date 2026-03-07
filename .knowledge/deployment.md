---
tags: [deployment, docker, ci-cd]
updated: 2026-03-05
---
# Deployment & Infrastruktur

## Docker Compose (Lokal + Self-Hosted)
Datei: `deploy/docker/docker-compose.yml`

### Infrastruktur-Container
| Service | Image | Port | Zweck |
|---------|-------|------|-------|
| postgres | postgres:16-alpine | 5432 | Hauptdatenbank |
| redis | redis:7-alpine | 6379 | Cache + Rate Limiting |
| minio | minio:latest | 9000/9001 | S3-kompatibler File-Storage |
| onlyoffice | onlyoffice/documentserver | 8088/8443 | Document-Editing |
| livekit | livekit-server | 7880/7881/7882 | Video/Audio (WebRTC) |
| livekit-egress | livekit-egress | — | Recording-Koordination |

### Backend-Services
| Service | Port | Abhaengigkeiten |
|---------|------|-----------------|
| auth | 50051 | postgres, redis, migrate |
| crm | 50052 | postgres, redis |
| chat | 50053 | postgres, redis, minio |
| notification | 50054 | postgres, redis |
| work | 50055 | postgres, redis, livekit |
| email | 50056 | postgres, redis |
| document | 50057 | postgres, redis, minio, onlyoffice |
| biz | 50058 | postgres, redis |
| automation | 50059 | postgres, redis |
| plugin | 50060 | postgres, redis |
| **gateway** | **8080** | **alle Services** |

### Health Checks
- Alle Services: `wget --spider http://localhost:{port}/health`
- Interval: 5-10s, Timeout: 5s, Retries: 10-15
- Restart: `unless-stopped`

## CI/CD
Datei: `.github/workflows/ci.yml`

- **Trigger:** Push auf main/develop, PRs (nur bei backend/ Aenderungen)
- **Go Version:** 1.25.6
- **Jobs (parallel):**
  1. **Lint** — golangci-lint v2.8
  2. **Test** — `go test ./... -race`, Coverage-Report, 30-Tage-Artifact
  3. **Build** — `make build` (abhaengig von Lint+Test)
  4. **E2E** — Integration Tests (abhaengig von Lint+Test)
- Service-Container: postgres:16-alpine + redis:7-alpine

## Deployment-Reihenfolge (KRITISCH)
1. Backup erstellen
2. Assets bauen (CSS, JS, Electron)
3. Config aktualisieren
4. Code deployen
5. `make migrate-up`
6. Services restarten
7. Health Check
8. Rollback-Plan bereithalten

## Produktion (geplant)
- **Hetzner Cloud** — EU-only, DSGVO-konform
- Kubernetes mit Blue-Green Deployment
- Automatische Skalierung
- **Status:** NOCH NICHT AUFGESETZT (Phase A Blocker)

## Self-Hosted (Kunden)
- Docker Compose Setup
- Automatische Backups via Cron
- Update via Docker Image Tags

## Verwandte Notes
- [[architektur]] — Service-Architektur
- [[security]] — Infrastruktur-Security
- [[integrationen]] — Docker-Container fuer LiveKit, OnlyOffice
