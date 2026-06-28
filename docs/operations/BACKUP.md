# Backup-Strategie

> Status: Skelett. Inhaltliche Befuellung in Sprint 5 vor Launch.

## Ziele (vor Launch 2026-09-01)

- **RPO** (Recovery Point Objective): max. 1h Datenverlust
- **RTO** (Recovery Time Objective): max. 4h bis Wiederherstellung
- **Aufbewahrung:** 30 Tage rolling backups + 12 monatliche Snapshots

Diese Werte sind Pilot-0-Defaults. Spaeter SLA-konfigurierbar per Tenant.

## Komponenten

### PostgreSQL Backup

*TODO Sprint 5: pg_dump-basierter taeglicher Full-Backup + WAL-Archiving.*

Script-Hooks:
- `deploy/scripts/backup.sh` — On-Demand-Full-Backup vor jedem Deploy
- *TODO Sprint 5: Cronjob fuer naechtlichen Full + 15min-WAL*

### MinIO/Object-Storage Backup

*TODO Sprint 5: rclone-Sync auf Hetzner Storage-Box (geo-redundant).*

### Konfigurations-Backup

`.env.production` + Caddyfile + docker-compose.yml — manuell in Password-Manager + verschluesseltes Repo-Mirror.

## Aufbewahrung

| Backup-Typ | Frequenz | Aufbewahrung |
|---|---|---|
| Full PostgreSQL | taeglich 03:00 UTC | 30 Tage |
| WAL Postgres | continuous | 7 Tage |
| Monatliche Full Snapshots | 1. des Monats | 12 Monate |
| Pre-Deploy Snapshot | bei jedem `deploy.sh` | 5 letzte |
| Object-Storage | taeglich | 30 Tage |

## Wiederherstellung

### Punkt-in-Zeit-Recovery (PITR)

*TODO Sprint 5: WAL-Replay-Procedure dokumentieren.*

### Vollstaendige Wiederherstellung (Disaster Recovery)

*TODO Sprint 5: Step-by-Step von leerer Hetzner-VM bis lauffaehigem System unter < 4h.*

Annahme: `backup.sh` legt taeglich nach `/var/backups/kmuhub/` ab und rclone-synct nach Hetzner Storage-Box.

### Single-Tenant-Restore

*TODO Sprint 6 (post-Launch): Selektives Restore eines einzelnen Tenants ohne andere zu beruehren — RLS-aware-Restore-Tool.*

## Test-Cadence

- **Monatlich:** Restore eines zufaelligen Backups auf Staging-VM, smoke-Test gegen Restore-Pool
- **Halbjaehrlich:** Vollstaendiger Disaster-Recovery-Drill mit Zeitmessung

## Verschluesselung

Backups sind at-rest verschluesselt (AES-256, Key in separater Hetzner Key-Vault). Transport TLS 1.3.

## Compliance

- **DSGVO Art. 32 (Sicherheit der Verarbeitung):** Backup-Procedure ist Teil des Auftragsverarbeitungsvertrags (AVV) mit Pilot-Kunden.
- **Sperrfristen:** Tenant-Loeschung loescht auch Backups innerhalb 90 Tagen (Pilot-0 Default, Sprint 6 dokumentieren in `security/gdpr`).
