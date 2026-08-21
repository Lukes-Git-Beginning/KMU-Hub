# Backup und Wiederherstellung

> Stand **2026-08-19**, gegen das laufende System gemessen. Jede Aussage hier ist entweder
> belegt oder ausdrücklich als offen gekennzeichnet — die Vorfassung behauptete unter anderem
> eine AES-256-Verschlüsselung at rest, die es nie gab, und einen rclone-Sync, der nie
> existierte. Solche Sätze landen sonst im AVV.

## Was tatsächlich läuft

| | Stand |
|---|---|
| Skript | `deploy/scripts/backup.sh` |
| Auslösung | Cron des Users `deploy`, **täglich 02:00** (plus einmal vor jedem `deploy.sh`) |
| Ablage | `/opt/kmuhub/backups/` auf demselben Host |
| Umfang | PostgreSQL (`pg_dump -Fc`, gzip) · Rollen (`pg_dumpall --roles-only`) · MinIO-Volume (tar über Sidecar) |
| Aufbewahrung | 30 Tage, für alle drei Artefakttypen |
| Protokoll | `/opt/kmuhub/logs/backup.log` |

Verifiziert am 2026-08-19: Läufe lückenlos seit dem 12.08., PostgreSQL und MinIO jede Nacht.

## Was es nicht gibt

Ausdrücklich, damit niemand danach plant:

- **Kein WAL-Archiving**, also keine Punkt-in-Zeit-Wiederherstellung. Der maximale Datenverlust
  ist der Abstand zum letzten nächtlichen Lauf, im schlechtesten Fall knapp 24 Stunden.
- **Keine monatlichen Langzeit-Snapshots.**
- **Keine Verschlüsselung der lokalen Backups.** Verschlüsselt wird nur, was ausgelagert wird
  (siehe unten), und das ist opt-in.
- **Kein automatischer Offsite-Transfer**, solange er nicht konfiguriert ist. Ohne Konfiguration
  liegen alle Backups auf genau dem Host, dessen Ausfall sie absichern sollen.

## Offsite (opt-in)

Aktiv nur, wenn **beide** Variablen gesetzt sind — sonst ist der Block ein No-op:

- `BACKUP_OFFSITE_TARGET` — Ziel für `rsync` (lokaler Pfad oder `user@host:/pfad`)
- `BACKUP_AGE_RECIPIENT` — öffentlicher `age`-Schlüssel (mehrere durch Leerzeichen getrennt)

Ausgelagert werden **alle drei Artefakte** — PostgreSQL-Dump, Rollen-Dump und MinIO-Archiv, jedes
einzeln mit `age` verschlüsselt. Der Rollen-Dump ist dabei kein Beiwerk: ohne ihn verweigert
`restore.sh` den Satz (siehe §Wiederherstellung), und erzwungen mit `--skip-roles` kommt eine
Datenbank zurück, die kein Dienst öffnen kann. Das MinIO-Archiv fehlt im Satz, wenn der lokale
MinIO-Lauf fehlgeschlagen ist — dann steht der Fehler im Protokoll und löst den Alarm aus.

Fehlt `age`, bricht der Transfer ab, statt unverschlüsselt zu senden. Der private Schlüssel gehört
zum zweiten Zugriffsweg (Lagebild §10) und darf nicht nur bei einer Person liegen — sonst ist das
Offsite-Backup im Ernstfall nicht lesbar.

## Alarm (opt-in)

Bei Fehlschlag geht eine Meldung an `BACKUP_ALERT_WEBHOOK`, ersatzweise an `ALERT_WEBHOOK_URL`
(denselben Kanal, den Alertmanager nutzt). Ohne gesetzten Webhook schlägt ein Backup weiterhin
still fehl.

## Wiederherstellung

`deploy/scripts/restore.sh [--yes] [--skip-roles] <pg_backup.dump.gz> [minio_backup.tar.gz]`

Reihenfolge im Skript: Services stoppen → **Rollen** → Daten → Migrationen → Prüfung
(Migrationskopf + Zeilenzahl) → Services starten.

**Die Rollen sind kein Detail.** `pg_dump` sichert sie nicht, sie sind Cluster-Ebene. Gemessen am
2026-08-19 gegen eine frische Instanz mit dem echten Nachtbackup:

| | ohne Rollen-Dump | mit Rollen-Dump |
|---|---|---|
| `pg_restore`-Fehler | **710** | **0** |
| Migrationskopf | 314 | 314 |
| Tabellen ohne `SELECT` für `kmuhub_app` | alle | **0 von 336** |
| Tabellen ohne `INSERT` für `kmuhub_app` | alle | **0 von 336** |

Ohne die Rollen kommt eine Datenbank zurück, die vollständig aussieht und die **kein Dienst öffnen
kann** — alle 24 Services verbinden sich als `kmuhub_app`. Deshalb bricht `restore.sh` ab, wenn der
Rollen-Dump fehlt. Backups von vor dem 2026-08-19 haben keinen; für sie ist `--skip-roles` der
bewusste Ausweg, und die Rollen müssen dann auf dem Ziel bereits existieren.

### Was noch nicht belegt ist

Verifiziert wurde die **Wiederherstellbarkeit der Daten** gegen eine Wegwerf-Instanz. Nicht
verifiziert, weil es nur gegen echte Produktion ginge:

- der Ablauf von `restore.sh` als Ganzes (Services stoppen, `migrate` danach, MinIO-Sidecar)
- die Wiederherstellung der MinIO-Daten
- eine Zeitmessung von leerem Host bis lauffähigem System

Solange das aussteht, gibt es **keine belegte RTO**. Ein Zielwert wäre eine Behauptung, keine
Zusage — deshalb steht hier keiner.

## Konfiguration

`.env.production`, Caddyfile und die Compose-Dateien sind **nicht** Teil des Backups. Die
Compose-Dateien liegen im Git-Repo; `.env.production` existiert nur auf dem Host. Geht der Host
verloren, sind die Secrets weg — das ist die zweite offene Flanke neben dem fehlenden Offsite.

## Prüf-Rhythmus

Ein Restore, den niemand durchführt, ist kein Backup. Nach jeder Änderung an `backup.sh` oder
`restore.sh` gehört ein Durchlauf gegen eine Wegwerf-Instanz dazu — das Rezept steht oben in der
Messtabelle und kostet wenige Minuten:

```
docker run -d --name restore-test -e POSTGRES_DB=kmuhub -e POSTGRES_USER=kmuhub \
  -e POSTGRES_PASSWORD=throwaway docker-postgres:latest \
  postgres -c shared_preload_libraries=pg_cron -c cron.database_name=kmuhub
# Rollen zuerst, dann Daten, dann has_table_privilege über pg_tables prüfen
```

Ohne Port-Mapping und ohne Volume — sonst kollidiert der Container mit der Produktion.

## DSGVO

Art. 32 verlangt die Wiederherstellbarkeit, nicht ihre Behauptung. Vor dem ersten Kunden fehlen
dafür: Offsite-Backup aktiv, ein durchgeführter Restore mit Zeitmessung, und eine Regel, wie
Backups nach einer Löschanfrage behandelt werden (ein anonymisierter Kontakt steht in älteren
Backups weiterhin im Klartext).
