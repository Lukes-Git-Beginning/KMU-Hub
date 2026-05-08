# Cosmi / KMU Hub — Ansible Provisioning

Pilot-Provisioning-Playbook für den Cosmi/KMU-Hub-Stack auf Hetzner Ubuntu 24.04 Hosts.

## Was dieses Playbook tut

- **foundation role**: apt-Pakete (Docker CE, fail2ban, ufw, jq, certbot, …), deploy-User + sudo, UFW-Firewall (SSH, HTTP/S, LiveKit 7881/tcp + 7882/udp, ICE 50000–60000/udp), fail2ban für sshd, Backup-Cron, Verzeichnis-Layout unter `/opt/kmuhub/`.
- **secrets role**: Generiert 12 kryptografische Secrets via `lookup('password')` und rendert `/opt/kmuhub/.env.production` aus `roles/secrets/templates/env.production.j2`.

**Welle 2 = foundation + secrets. Welle 3 = app-deploy + turn + caddy/dns.**

## Voraussetzungen

### Option A: Lokales Ansible (Linux/macOS)

- ansible-core >= 2.15
- Collections:
  ```bash
  ansible-galaxy collection install community.general community.docker community.crypto ansible.posix
  ```

### Option B: Docker-Wrapper (Windows / kein natives Ansible)

Natives Ansible unter Windows funktioniert nicht (`No module named 'grp'`). Docker Desktop als Workaround:

```bash
MSYS_NO_PATHCONV=1 docker run --rm \
  -v "/c/Users/Luke/Documents/KMU Hub:/work" \
  -w /work/deploy/ansible \
  willhallonline/ansible:latest \
  ansible-playbook -i inventory/hosts.yml site.yml --ask-vault-pass
```

> **Hinweis:** `MSYS_NO_PATHCONV=1` ist zwingend — sonst übersetzt Git Bash `/work` zu `C:/Program Files/Git/work`.

## Inventory vorbereiten

1. `inventory/hosts.yml` öffnen und `<PLATZHALTER_IP>` durch echte Server-IPs ersetzen:
   ```yaml
   pilot-0-zfa:
     ansible_host: 1.2.3.4    # <-- eintragen
   turn-shared:
     ansible_host: 5.6.7.8    # <-- eintragen
   ```

2. SSH-Public-Key für den deploy-User in `inventory/group_vars/all.yml` setzen:
   ```yaml
   deploy_user_pubkeys:
     - "ssh-ed25519 AAAA... luke@zentria"
   ```

3. Optional: `admin_cidr` setzen (schränkt SSH-Zugriff auf eine IP/Range ein):
   ```yaml
   admin_cidr: "1.2.3.4/32"
   ```

## Vault-Setup (Operator-Action — nicht automatisiert)

Secrets die nicht generiert werden (z. B. Bexio/DATEV-Credentials, Slack-Webhook) kommen in einen Vault:

```bash
ansible-vault create inventory/group_vars/pilots/vault.yml
```

Beispiel-Inhalt:
```yaml
vault_slack_webhook_url: "https://hooks.slack.com/services/..."
vault_bexio_client_id: "..."
vault_bexio_client_secret: "..."
```

Die vault-Variablen dann in `group_vars/pilots.yml` referenzieren:
```yaml
slack_webhook_url: "{{ vault_slack_webhook_url }}"
```

**Die `vault.yml` liegt NIE im Repo** — nur `.gitignore`d auf dem Operator-Rechner oder in einem Secrets-Manager.

## Playbook ausführen

```bash
# Syntax-Check (kein echtes Apply):
ansible-playbook -i inventory/hosts.yml --syntax-check site.yml

# Tasks auflisten:
ansible-playbook -i inventory/hosts.yml --list-tasks site.yml

# Trockenlauf (--check):
ansible-playbook -i inventory/hosts.yml --check site.yml --ask-vault-pass

# Echter Apply (Pilot-Provisioning):
ansible-playbook -i inventory/hosts.yml site.yml --ask-vault-pass
```

## Collections — Hinweis bei fehlenden Modulen

Das Docker-Image `willhallonline/ansible:latest` enthält `community.general` oft nicht vorinstalliert. Wenn `ansible-playbook --list-tasks` meldet `Could not find module community.general.ufw`, Collections vor dem Run installieren:

```bash
MSYS_NO_PATHCONV=1 docker run --rm \
  -v "/c/Users/Luke/Documents/KMU Hub:/work" \
  -w /work/deploy/ansible \
  willhallonline/ansible:latest \
  sh -c "ansible-galaxy collection install community.general community.docker community.crypto ansible.posix && ansible-playbook -i inventory/hosts.yml --list-tasks site.yml"
```

## Struktur

```
deploy/ansible/
├── ansible.cfg               # Ansible-Konfiguration
├── site.yml                  # Entry-Point: importiert provision.yml
├── inventory/
│   ├── hosts.yml             # pilot-0-zfa + turn-shared
│   └── group_vars/
│       ├── all.yml           # Globale Variablen
│       ├── pilots.yml        # Pilot-spezifische Variablen
│       └── turn.yml          # TURN-Server-Variablen (Welle 3)
├── playbooks/
│   ├── provision.yml         # foundation + secrets Roles
│   └── update.yml            # Placeholder für app-deploy (Welle 3)
└── roles/
    ├── foundation/           # System-Setup (apt, Docker, UFW, fail2ban, cron)
    └── secrets/              # Secret-Generierung + env.production-Rendering
```

## Roadmap

| Welle | Inhalt |
|-------|--------|
| Welle 2 (jetzt) | foundation + secrets |
| Welle 3 | app-deploy (Git-Pull + docker compose up), turn-Role (coturn), caddy/dns-Role |
| Welle 4 | Smoke-Tests via Ansible, Rollback-Playbook |
