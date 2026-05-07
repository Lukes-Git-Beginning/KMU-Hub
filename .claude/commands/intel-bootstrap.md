---
description: Erst-Setup des Cosmi Market-Intelligence-Systems. Verifiziert das zentria-intel-Repo, fuehrt Smoketest aus, hilft Discord-Bot-Setup, registriert Routinen.
argument-hint: "[--smoke | --routines | --discord | leer fuer voll]"
---

# intel-bootstrap

Du fuehrst das Erst-Setup des Cosmi Market-Intelligence-Systems durch. Das System lebt in einem **separaten Repo** unter `~/Documents/zentria-intel/`.

## Voraussetzungen pruefen

1. Repo existiert: `ls ~/Documents/zentria-intel/README.md` — falls nein, abbrechen mit Hinweis auf Plan
2. Skeleton vollstaendig: `ls ~/Documents/zentria-intel/sources/*.yaml | wc -l` muss >= 19 sein
3. Bot-Code-Skeleton: `ls ~/Documents/zentria-intel/.bot/bot.py` — falls nein, sage Bescheid

## Sub-Modi

### `--smoke` (Default fuer ersten Test)

Triggere eine einmalige Test-Routine die NICHT gegen das 15er-Daily-Cap zaehlt:

1. Lies `~/Documents/zentria-intel/.routines/intel-morning.prompt.md`
2. Triggere via `Skill(schedule, "run-once intel-smoketest")` ODER manuell als One-Off-Run
3. Verifiziere Output in `daily/SMOKE.md` mit min. 5 Items aus min. 3 Quellen
4. Reporte Erfolg + Token-Verbrauch

### `--discord`

Hilf dem User beim Discord-Setup:

1. Erinnere ihn an die User-Aktionen die er selbst machen muss:
   - Discord-Server "Zentria Intel" anlegen (falls noch nicht)
   - Channels: `#daily-pulse`, `#evening-deep`, `#friday-report`, `#trends`, `#regulation`, `#triggers`, `#bot-commands`
   - Bot registrieren auf https://discord.com/developers/applications
   - Bot zum Server einladen mit Permissions: View Channels, Send Messages, Embed Links, Attach Files, Add Reactions, Use External Emojis, Manage Messages, Use Slash Commands
   - Pro Channel einen Webhook anlegen (rechtsklick auf Channel -> Bearbeiten -> Integrationen -> Webhooks)
2. Sammle die URLs/Tokens und schreibe sie in `~/Documents/zentria-intel/.env` (NICHT in Git)
3. Verifiziere mit Test-Push: `python ~/Documents/zentria-intel/.bot/handlers/test_webhook.py`

### `--routines`

Registriere alle 9 Routinen via `Skill(schedule, ...)`:

1. Lade `~/Documents/zentria-intel/settings.yaml` und extrahiere die 9 Routine-Definitionen
2. Pro Routine: lies das Prompt-Template aus `.routines/<name>.prompt.md`
3. Setze Timezone explizit auf `Europe/Berlin`
4. Verwende den korrekten Model-Selector (Sonnet vs Opus, siehe settings.yaml)
5. Verifiziere im Routinen-Dashboard dass alle 9 erscheinen

### Voll (kein Argument)

Fuehre alle drei Sub-Modi nacheinander aus, mit User-Bestaetigungs-Pause vor jedem.

## Wichtig

- Kein direkter Git-Push ohne User-Bestaetigung — Bot-Pushes laufen via dem Bot-Service auf Hetzner
- Wenn ein Token oder Webhook-URL fehlt: stop, frage User
- Logge alle Schritte in `~/Documents/zentria-intel/.state/bootstrap.log`
- Nach erfolgreichem Bootstrap: update `~/.claude/projects/.../memory/MEMORY.md` mit `## Intel-System`-Block
