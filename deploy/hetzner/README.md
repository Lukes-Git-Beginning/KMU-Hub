# KMU Hub -- Hetzner Production Setup

## Server-Empfehlung

**Hetzner CPX41** (Shared vCPU)
- 8 vCPU (AMD EPYC)
- 16 GB RAM
- 240 GB NVMe SSD
- ~16.90 EUR/Monat
- Standort: Falkenstein (fsn1) oder Nuernberg (nbg1)

Alternativ CPX51 (16 vCPU, 32 GB RAM) wenn mehr Headroom gewuenscht.

## 1. Server erstellen

Via Hetzner Cloud Console oder CLI:

```bash
hcloud server create \
    --name kmuhub-prod \
    --type cpx41 \
    --image ubuntu-24.04 \
    --location fsn1 \
    --ssh-key YOUR_SSH_KEY_NAME
```

## 2. Firewall konfigurieren

```bash
# Firewall erstellen
hcloud firewall create --name kmuhub-fw

# Regeln hinzufuegen
hcloud firewall add-rule kmuhub-fw --direction in --protocol tcp --port 22 --source-ips YOUR_IP/32 --description "SSH"
hcloud firewall add-rule kmuhub-fw --direction in --protocol tcp --port 80 --source-ips 0.0.0.0/0 --description "HTTP"
hcloud firewall add-rule kmuhub-fw --direction in --protocol tcp --port 443 --source-ips 0.0.0.0/0 --description "HTTPS"
hcloud firewall add-rule kmuhub-fw --direction in --protocol tcp --port 7881 --source-ips 0.0.0.0/0 --description "LiveKit WebRTC"
hcloud firewall add-rule kmuhub-fw --direction in --protocol udp --port 50000-60000 --source-ips 0.0.0.0/0 --description "LiveKit ICE/UDP"

# An Server anhaengen
hcloud firewall apply-to-resource kmuhub-fw --type server --server kmuhub-prod
```

## 3. Server absichern

```bash
ssh root@SERVER_IP

# Deploy-User erstellen
adduser deploy
usermod -aG sudo deploy
mkdir -p /home/deploy/.ssh
cp ~/.ssh/authorized_keys /home/deploy/.ssh/
chown -R deploy:deploy /home/deploy/.ssh

# SSH haerten
sed -i 's/^#*PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
systemctl restart sshd

# Firewall + fail2ban
apt update && apt install -y fail2ban ufw
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 7881/tcp
ufw allow 50000:60000/udp
ufw enable

# Automatische Sicherheitsupdates
apt install -y unattended-upgrades
dpkg-reconfigure -plow unattended-upgrades
```

## 4. Docker installieren

```bash
# Als deploy user
sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo usermod -aG docker deploy
# Logout and login again for group change
```

## 5. Deployment

```bash
# Verzeichnisstruktur
sudo mkdir -p /opt/kmuhub/backups
sudo chown -R deploy:deploy /opt/kmuhub

# Repository klonen
cd /opt/kmuhub
git clone https://github.com/Lukes-Git-Beginning/KMU-Hub.git .

# Konfigurationsdateien verlinken
ln -s deploy/docker/docker-compose.yml docker-compose.yml
ln -s deploy/docker/docker-compose.prod.yml docker-compose.prod.yml
ln -s deploy/docker/Caddyfile Caddyfile
ln -s deploy/docker/prometheus.yml prometheus.yml
ln -s deploy/docker/livekit.yaml livekit.yaml
ln -s deploy/docker/egress.yaml egress.yaml

# Symlinks fuer Scripts
ln -s deploy/scripts scripts

# Production Environment erstellen
cp deploy/docker/.env.production.example .env.production
nano .env.production  # ALLE CHANGE_ME Werte aendern!

# Starten
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# Health Check
./scripts/healthcheck.sh
```

## 6. Backup einrichten

```bash
# Naechtliches Backup um 02:00
crontab -e
# Zeile hinzufuegen:
# 0 2 * * * /opt/kmuhub/scripts/backup.sh >> /var/log/kmuhub-backup.log 2>&1
```

## 7. Monitoring

Grafana und Prometheus sind nur auf localhost erreichbar.
Zugriff via SSH Tunnel:

```bash
# Vom lokalen Rechner:
ssh -L 3000:localhost:3000 -L 9091:localhost:9091 deploy@SERVER_IP

# Dann im Browser:
# Grafana: http://localhost:3000
# Prometheus: http://localhost:9091
```

## 8. Updates deployen

```bash
ssh deploy@SERVER_IP
cd /opt/kmuhub
./scripts/deploy.sh
```

## 9. LiveKit Konfiguration

In `livekit.yaml` fuer Produktion `use_external_ip: true` setzen und die korrekte IP/Domain eintragen.

## Troubleshooting

| Problem | Loesung |
|---------|---------|
| Service startet nicht | `docker compose logs SERVICE_NAME` |
| Port bereits belegt | `sudo lsof -i :PORT` |
| Speicher voll | `docker system prune -a` |
| DB Migration fehlschlaegt | `docker compose run --rm migrate` |
| Backup fehlschlaegt | Logs pruefen: `/var/log/kmuhub-backup.log` |
