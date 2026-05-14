# Pilot-Setup-Guide

> Status: Skelett. Inhaltliche Befuellung in Sprint 5.

Schritt-fuer-Schritt-Anleitung fuer die initiale Einrichtung eines Pilot-Tenants.

## Voraussetzungen

- Provisionierter Tenant in der Production-Datenbank (Hetzner)
- Tenant-Admin-User mit verifizierter E-Mail
- Modul-Feature-Flags fuer den Tenant gesetzt

## Schritt 1: Tenant-Provisioning

*TODO Sprint 5: Ansible-Run zum Anlegen von tenants-Row, Default-Modulen und Initial-Admin.*

```bash
ansible-playbook deploy/ansible/playbooks/provision-pilot.yml -e tenant_slug=zfa
```

## Schritt 2: Initial-Login + Admin-Setup

*TODO Sprint 5: Erste Anmeldung des Tenant-Admins, MFA aktivieren, Profil vervollstaendigen.*

## Schritt 3: Modul-Aktivierung

*TODO Sprint 5: CRM, Helpdesk, Wiki, Dialer, Berichte, Formulare einzeln durchgehen mit Default-Konfiguration.*

## Schritt 4: User-Anlage + Rollen-Verteilung

*TODO Sprint 5: Bulk-CSV-Import oder manuell. Empfohlene Initial-Rollen-Verteilung fuer Praxis-Setup.*

## Schritt 5: Daten-Import

*TODO Sprint 5: Bexio-Sync aktivieren ODER CSV-Import-Wizard fuer Kontakte/Termine.*

## Schritt 6: Branding + Vorlagen

*TODO Sprint 5/6: Logo, Farbschema, Briefkopf-Templates, Default-Antworten in Helpdesk.*

## Schritt 7: Test-Workflow

*TODO Sprint 5: End-to-End Smoke-Test: Kontakt anlegen, Ticket erstellen, Bericht generieren, Anruf protokollieren.*

## Schritt 8: Go-Live-Checkliste

- [ ] Alle Pilot-User koennen sich anmelden
- [ ] Alle aktivierten Module sind getestet
- [ ] Backup-Job laeuft (siehe [Backup-Strategie](../operations/BACKUP.md))
- [ ] Monitoring-Alerts gehen an Discord-Channel `#cosmi-prod-alerts`
- [ ] Support-Kanal mit Kunde etabliert
- [ ] Pilot-Vertrag unterzeichnet
