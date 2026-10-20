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
- `BACKUP_OFFSITE_SSH` — optional, Remote-Shell für `rsync -e`. Für eine Hetzner Storage Box
  **Pflicht**, denn sie spricht SSH nur auf Port 23.

Ausgelagert werden **alle drei Artefakte** — PostgreSQL-Dump, Rollen-Dump und MinIO-Archiv — und
dazu **`.env.production`** als `env_<TS>.production.age`. Jedes wird einzeln mit `age`
verschlüsselt. Die Env-Datei gehört dazu, weil `VAULT_MASTER_SECRET`, `JWT_SECRET` und die
DB-Passwörter nur dort stehen: Ohne sie kommt nach einem Hostverlust zwar die Datenbank zurück,
die im Vault verschlüsselten Zugangsdaten aber nicht. Sie verlässt den Host nur verschlüsselt. Der Rollen-Dump ist dabei kein Beiwerk: ohne ihn verweigert
`restore.sh` den Satz (siehe §Wiederherstellung), und erzwungen mit `--skip-roles` kommt eine
Datenbank zurück, die kein Dienst öffnen kann. Das MinIO-Archiv fehlt im Satz, wenn der lokale
MinIO-Lauf fehlgeschlagen ist — dann steht der Fehler im Protokoll und löst den Alarm aus.

Fehlt `age`, bricht der Transfer ab, statt unverschlüsselt zu senden. Der private Schlüssel gehört
zum zweiten Zugriffsweg (Lagebild §10) und darf nicht nur bei einer Person liegen — sonst ist das
Offsite-Backup im Ernstfall nicht lesbar.

### Einrichtung mit einer Hetzner Storage Box

1. Storage Box buchen (BX11 reicht, EU-Standort), in der Konsole **SSH-Support** aktivieren.
2. Auf dem Rechner des Admins, **nicht** auf dem Server: `age-keygen -o cosmi-backup.agekey`.
   Die Zeile `# public key: age1…` ist der Wert für `BACKUP_AGE_RECIPIENT`. Die Schlüsseldatei
   kommt in den Passwort-Tresor und an den zweiten Zugriffsweg.
3. Auf dem Server als `deploy`: `ssh-keygen -t ed25519 -f ~/.ssh/storagebox -N ""`, dann den
   Public Key auf die Box: `ssh-copy-id -p 23 -s -i ~/.ssh/storagebox.pub uXXXXXX@uXXXXXX.your-storagebox.de`
   (fragt einmal das Box-Passwort ab).
4. `ssh-keyscan -p 23 uXXXXXX.your-storagebox.de >> ~/.ssh/known_hosts` und den Fingerprint
   gegen die Hetzner-Dokumentation prüfen. Danach Zielordner anlegen:
   `ssh -p 23 -i ~/.ssh/storagebox uXXXXXX@uXXXXXX.your-storagebox.de mkdir kmuhub-backups`.
5. Die drei Variablen in `/opt/kmuhub/.env.production` setzen (Vorlage: `PRODUCTION_TEMPLATE`)
   und `age` installieren (`apt-get install age`, im Ansible-Playbook enthalten).
6. Testlauf: `sudo -u deploy bash /opt/kmuhub/deploy/scripts/backup.sh` — im Log müssen vier
   Zeilen `Offsite: … shipped` stehen.

Eine Aufbewahrungsfrist auf der Box gibt es nicht; `rsync` löscht nie. Bei etwa 570 KB pro Nacht
ist das unkritisch (`lean:`-Marker in `backup.sh` nennt die Schwelle).

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

Caddyfile und Compose-Dateien liegen im Git-Repo und brauchen kein Backup. `.env.production`
existiert nur auf dem Host. Lokal wird sie nicht gesichert, mit konfiguriertem Offsite geht sie
aber verschlüsselt mit (siehe §Offsite). Ohne Offsite gilt weiter: Geht der Host verloren, sind
die Secrets weg.

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
