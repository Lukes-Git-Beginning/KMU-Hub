# Docker Deployment

Compose-Stack fuer KMU Hub: 24 Microservices + Gateway + Postgres + Redis + MinIO + LiveKit + OnlyOffice + Caddy + Prometheus + Grafana.

## Required Environment Variables

`docker-compose.yml` erzwingt seit Sprint-2-Hardening (2026-04-26) alle DB-Credentials via `${VAR:?msg}`-Substitution — kein `kmuhub_dev`-Default mehr. Compose bricht beim Start ab, wenn folgende Variablen nicht gesetzt sind.

| Variable | Beispiel (Dev) | Beispiel (Prod) | Wer braucht es? |
|---|---|---|---|
| `DATABASE_URL` | `postgres://kmuhub:kmuhub_dev@postgres:5432/kmuhub?sslmode=disable` | aus `.env.production` (Hetzner) | Alle 24 Backend-Services + `migrate` |
| `POSTGRES_PASSWORD` | `kmuhub_dev` | strong, min 32 Zeichen | `postgres`-Service |
| `LIVEKIT_API_KEY` | `devkey` | aus `.env.production` | `render-configs.sh` (Welle 1.2) |
| `LIVEKIT_API_SECRET` | `devsecret` | aus `.env.production` | `render-configs.sh` (Welle 1.2) |
| `LIVEKIT_WEBHOOK_API_KEY` | `devkey` | aus `.env.production` | `render-configs.sh` (Welle 1.2) |

Die uebrigen Variablen (`JWT_SECRET`, `WOPI_JWT_SECRET`, `VAULT_MASTER_SECRET`, `MINIO_*`) haben weiterhin Dev-Defaults im Compose-File. Fuer Prod werden sie via `.env.production` ueberschrieben — die vollstaendige Prod-Liste steht in [`PRODUCTION_TEMPLATE`](./PRODUCTION_TEMPLATE).

## Dev-Setup

```bash
# Einmalig im Repo-Root: Dev-.env anlegen
cat > deploy/docker/.env <<'EOF'
DATABASE_URL=postgres://kmuhub:kmuhub_dev@postgres:5432/kmuhub?sslmode=disable
POSTGRES_PASSWORD=kmuhub_dev
LIVEKIT_API_KEY=devkey
LIVEKIT_API_SECRET=devsecret
LIVEKIT_WEBHOOK_API_KEY=devkey
EOF

# Stack hochfahren
make dev-up
```

Die `.env`-Datei ist in `.gitignore`/Pre-Commit-Hook-Filter und wird nie committed.

## Prod-Setup

`.env.production` liegt auf dem Server unter `/opt/kmuhub/.env.production` (nicht im Repo). Vorlage zum Kopieren: [`PRODUCTION_TEMPLATE`](./PRODUCTION_TEMPLATE). Symlink `deploy/docker/.env → ../../.env.production` auf dem Server, damit Compose `--env-file` greift.

`render-configs.sh` rendert ausserdem `livekit-secrets.yaml` aus dem Template — siehe [`../scripts/render-configs.sh`](../scripts/render-configs.sh).

## Healthchecks

Alle Backend-Services horchen auf `/health` (Port `9091..9105`). Healthcheck-Pattern: `wget --spider` (HEAD-Request). Backend unterstuetzt seit Sprint-2-Hardening sowohl GET als auch HEAD.

## Files

- `docker-compose.yml` — Base-Stack, Dev-tauglich
- `docker-compose.prod.yml` — Prod-Overlay (Logging, Resource-Limits, OnlyOffice-JWT, Caddy, Monitoring)
- `livekit.yaml` — LiveKit-Base-Config (Dev-Keys, Overlay-Pattern)
- `livekit-turn.yaml` — TURN-Aktivierungs-Overlay (auskommentiert in `prod.yml`, siehe Header dort)
- `livekit-secrets.yaml.tmpl` — Prod-Secrets-Template, gerendert via `render-configs.sh`
- `PRODUCTION_TEMPLATE` — Vollstaendige `.env.production`-Vorlage
- `Caddyfile` — TLS + Reverse-Proxy
- `prometheus.yml` + `grafana/` — Monitoring
