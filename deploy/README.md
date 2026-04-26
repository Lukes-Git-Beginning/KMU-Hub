# Deploy Scripts

Scripts under `deploy/scripts/` automate KMU Hub production operations on Hetzner.
They are designed for the Hetzner CPX42 server at `app.zentria.tech` running Ubuntu 24.04,
but the patterns apply to any comparable Linux host.

## Initial Server Setup

### Prerequisites

Install these system packages before running any deploy scripts:

```bash
apt install -y jq            # required by smoke.sh (JSON parsing)
apt install -y gettext-base  # required by render-configs.sh (envsubst)
```

Ensure `docker` and the `docker compose` plugin are installed:

```bash
docker --version
docker compose version
```

### First-time setup

Install the nightly backup cron job as root:

```bash
sudo /opt/kmuhub/deploy/scripts/setup-cron.sh
```

This creates `/opt/kmuhub/logs/` (owned by the `deploy` user) and writes a crontab
entry that runs `backup.sh` daily at 02:00 UTC. Logs are written to
`/opt/kmuhub/logs/backup.log`. The script is idempotent — safe to run multiple times.

> **Note:** Retention is controlled by `backup.sh` itself (`RETENTION_DAYS=30`,
> `MIN_BACKUPS=7`). No separate log-rotation config is needed.

## Scripts Overview

| Script | Purpose |
|--------|---------|
| `deploy.sh` | Production deploy: backup → build → migrations → rolling restart → health + smoke check + auto-rollback on failure |
| `backup.sh` | pg_dump + MinIO tar archive written to `/opt/kmuhub/backups/` |
| `healthcheck.sh` | Checks service health endpoints |
| `smoke.sh` | End-to-end tests against the live URL (requires `jq`) |
| `render-configs.sh` | Renders `livekit-secrets.yaml` from `.env.production` (requires `envsubst`) |
| `setup-cron.sh` | Installs the nightly backup cron job for the `deploy` user |
