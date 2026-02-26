# Claude Tooling Guide — KMU Hub

> Setup-Anleitung und Strategie fuer Anthropic Claude Tools im KMU Hub Projekt.
> Stand: Februar 2026

---

## Uebersicht: 4 Tools + 2 Integrationen

| # | Tool | Typ | Zweck | Kosten |
|---|------|-----|-------|--------|
| 1 | Security Review Action | GitHub Action | Auto-Security-Scan bei PRs | Im Plan enthalten |
| 2 | Claude Code Action | GitHub Action | AI-Agent fuer PRs, Issues, Automation | Im Plan enthalten |
| 3 | Code Review Plugin | Lokal / CI | Multi-Agent Review mit Confidence-Scoring | Kostenlos |
| 4 | `/security-review` | CLI-Befehl | Lokaler Security-Scan | Im Plan enthalten |
| 5 | GitHub MCP Server | MCP Integration | Direkter GitHub-Zugriff aus Claude Code | Kostenlos |
| 6 | Claude Code Hooks | Projekt-Config | Pre-Commit-Guards und Enforcement | Kostenlos |

**Monatliche Gesamtkosten:** ~$40 (2x Claude Pro/Max — GitHub Actions laufen ueber bestehendes Abo via OAuth)

---

## 1. Claude Code Security Review (GitHub Action)

**Repository:** https://github.com/anthropics/claude-code-security-review

### Was es erkennt

- SQL Injection, Command Injection, XXE, NoSQL Injection
- XSS (Reflected, Stored, DOM-based)
- Broken Authentication, Privilege Escalation, IDOR
- Hardcoded Secrets, Sensitive Data Logging, PII Violations
- Weak Crypto, Insecure RNG, Improper Key Management
- Race Conditions, TOCTOU Issues
- Insecure Defaults, Missing Security Headers, Permissive CORS
- RCE via Deserialization, eval Injection
- Vulnerable Dependencies, Typosquatting

### Funktionsweise

- **Diff-Aware:** Scannt nur geaenderte Dateien (reduziert Noise und Kosten)
- **Semantisch:** Versteht Code-Logik, nicht nur Pattern-Matching
- **PR-Kommentare:** Postet Findings inline als PR-Kommentare
- **Sprachunabhaengig:** Funktioniert mit Go, TypeScript, und allen anderen Sprachen

### Konfigurationsoptionen

| Parameter | Default | Beschreibung |
|-----------|---------|-------------|
| `claude-api-key` | Pflicht | OAuth Token (via `/install-github-app`) oder API Key |
| `comment-pr` | `true` | Findings als PR-Kommentare posten |
| `upload-results` | `true` | Ergebnisse als GitHub Artifact hochladen |
| `exclude-directories` | — | Verzeichnisse ausschliessen (kommasepariert) |
| `claude-model` | `claude-opus-4-1-20250805` | Modell fuer Analyse |
| `claudecode-timeout` | `20 minutes` | Max. Analysezeit |
| `custom-security-scan-instructions` | — | Eigene Security-Richtlinien |
| `false-positive-filtering-instructions` | — | Regeln zur False-Positive-Filterung |

### Einschraenkungen

- **Nicht gehaertet gegen Prompt Injection** — nur fuer interne/trusted PRs verwenden
- Empfehlung: "Require approval for all external contributors" in Repo-Settings aktivieren

### Workflow-Datei

Siehe `.github/workflows/security-review.yml`

---

## 2. Claude Code Action (GitHub Action)

**Repository:** https://github.com/anthropics/claude-code-action
**Marketplace:** https://github.com/marketplace/actions/claude-code-action-official

### 3 Aktivierungsmodi

| Modus | Trigger | Beispiel |
|-------|---------|---------|
| **Interactive** | `@claude` Mention in PR-Kommentar | "Hey @claude, review this change for security issues" |
| **Assignment** | Claude als Assignee auf Issue | Issue zuweisen → Claude untersucht und antwortet |
| **Automation** | Event-basiert (PR open, push, schedule) | Automatischer Review bei jedem PR |

### Was es kann

- **Code Reviews:** Qualitaet, Security, Best Practices, Architektur
- **Code Generation:** Bug Fixes, Refactoring, neue Features, Tests schreiben
- **Repo-Automation:** Issue Triage, Labeling, Docs-Updates, Release Notes
- **Interaktiv:** Fragen zum Code beantworten, Architektur erklaeren, Debugging

### Features

- Multi-Cloud Support (Anthropic Direct, AWS Bedrock, Google Vertex AI)
- Structured Output (JSON-Validation)
- Progress Tracking (visuelle Checkboxen)
- MCP-Erweiterbar (zusaetzliche Tools anschliessbar)

### Setup (einfachster Weg)

```bash
# In Claude Code ausfuehren:
/install-github-app
```

Fuehrt automatisch durch GitHub App Installation und Secret-Setup.

### Workflow-Datei

Siehe `.github/workflows/claude-pr.yml`

---

## 3. Code Review Plugin (Lokal / CI)

**Quelle:** Eingebaut in Claude Code (Plugin-System)

### 4 Parallele Review-Agents

| Agent | Aufgabe |
|-------|---------|
| CLAUDE.md Compliance #1 | Prueft Einhaltung der Projekt-Richtlinien |
| CLAUDE.md Compliance #2 | Redundante Compliance-Pruefung |
| Bug Detector | Erkennt Bugs in geaenderten Dateien |
| History Analyzer | Analysiert Git Blame/History fuer Kontext |

### Confidence-Based Scoring

Jedes Finding bekommt einen Score von 0-100:
- **0-25:** Wahrscheinlich False Positive → wird gefiltert
- **25-50:** Moeglicherweise real → wird gefiltert (unter Threshold)
- **50-75:** Real aber minor → wird gefiltert (unter Threshold)
- **75-100:** Definitiv real und wichtig → wird angezeigt
- **Threshold:** 80 (konfigurierbar)

### Verwendung

```bash
# Lokal ausfuehren (Ausgabe im Terminal):
/code-review

# Als PR-Kommentar posten:
/code-review --comment
```

### Smart Skipping

Ueberspringt automatisch: geschlossene PRs, Draft PRs, triviale Aenderungen, bereits reviewte PRs.

### Unterschied zur Security Review Action

| | Security Review Action | Code Review Plugin |
|---|---|---|
| Fokus | Security Vulnerabilities | Code Quality + Compliance + Bugs |
| Agents | 1 Modell | 4 parallele Agents |
| Scoring | Pattern-Filterung | Explizites 0-100 Scoring |
| Integration | GitHub Action (automatisch) | Lokal oder CI (manuell) |

---

## 4. /security-review (Lokaler Befehl)

### Verwendung

```bash
# In Claude Code ausfuehren:
/security-review
```

### Wann einsetzen

- **Vor jedem PR:** Lokaler Pre-Check bevor Code gepusht wird
- **Nach groesseren Aenderungen:** Insbesondere bei Auth, API, DB-Zugriff
- **Ad-hoc:** Wenn du dir bei einer Implementierung unsicher bist

### Was es prueft

- SQL Injection Risiken
- XSS Vulnerabilities
- Authentication Schwaechen
- Insecure Data Handling
- Dependency Vulnerabilities
- Business Logic Flaws

### Workflow-Empfehlung

```
Code schreiben → /security-review → Fixes → /code-review → PR erstellen
```

---

## 5. GitHub MCP Server

### Was es ermoeglicht

Direkter GitHub-Zugriff aus Claude Code heraus:
- PRs lesen, analysieren, kommentieren
- Issues triagen und labeln
- CI/CD Failures debuggen
- Commits und Branches analysieren
- Dependabot Alerts pruefen

### Setup

```bash
claude mcp add github -e GITHUB_PERSONAL_ACCESS_TOKEN=ghp_XXXXX -- npx -y @modelcontextprotocol/server-github
```

### Verifizieren

```bash
claude mcp list
```

### Benoetigte GitHub Token Scopes

- `repo` (Vollzugriff auf Repositories)
- `read:org` (optional, fuer Org-Infos)

Token erstellen unter: https://github.com/settings/tokens

---

## 6. Claude Code Hooks

### Projekt-Level Hooks (.claude/settings.json)

Hooks erzwingen Best Practices automatisch:

**PreToolUse Hooks:**
- Blockieren gefaehrliche Operationen (exit code 2)
- Laufen VOR jeder Tool-Ausfuehrung

**PostToolUse Hooks:**
- Laufen NACH jeder Tool-Ausfuehrung
- Koennen nicht blockieren, nur warnen

### Konfigurierte Hooks fuer KMU Hub

Siehe `.claude/settings.json` fuer die aktive Konfiguration.

**Conventional Commit Enforcement:**
- Prueft ob Commit-Messages dem Format `feat:|fix:|docs:|refactor:|test:|chore:` folgen
- Blockiert Commits mit falschem Format

**Secrets Protection:**
- Blockiert das Committen von `.env`, `credentials.json`, Dateien mit Secrets
- Warnt bei hardcoded API Keys in Code

**Structured Logging Check:**
- Warnt bei `fmt.Println`, `console.log` in Go/TypeScript Dateien
- Erinnert an `slog` (Go) bzw. strukturiertes Logging (TS)

---

## Kosten-Analyse

### Monatliche Kosten (2 Entwickler)

| Posten | Kosten |
|--------|--------|
| Claude Max — Luke | Im bestehenden Abo |
| Claude Pro — Darien | $20 |
| GitHub Actions (via OAuth) | Im Abo enthalten |
| GitHub Actions Minutes | Free Tier ausreichend |
| GitHub MCP Server | Kostenlos |
| Code Review Plugin | Kostenlos |
| Hooks | Kostenlos |
| **Gesamt** | **Keine Zusatzkosten** (bestehendes Max/Pro Abo) |

### Vergleich

| Setup | Monatlich | Security | Code Review | Automation |
|-------|-----------|----------|-------------|-----------|
| **Unser Claude Setup** | Im Abo | Semantisch (deep) | Multi-Agent + Confidence | Voll (Actions + MCP) |
| CodeRabbit (3 Devs) | $72 | Ja | Pattern-based | Nein |
| GitHub Copilot (3 Devs) | $30-60 | Basic | Oberflaechlich | Nein |
| Snyk + CodeRabbit | $100+ | Pattern-based | Ja | Teilweise |

---

## Empfohlener Workflow

### Taeglicher Entwicklungs-Flow

```
1. Code schreiben (Claude Code)
      ↓
2. /security-review (lokaler Check)
      ↓
3. Fixes anwenden
      ↓
4. /code-review (lokaler Multi-Agent Review)
      ↓
5. git commit (Hooks pruefen: Conventional Commits, Secrets)
      ↓
6. git push + PR erstellen
      ↓
7. Security Review Action (automatisch auf GitHub)
      ↓
8. Claude Code Action reviewt PR (automatisch)
      ↓
9. Teammate reviewt + merged
```

### Bei externen Contributors / Dariens PRs

```
1. Darien erstellt PR von design/brainstorm
      ↓
2. Security Review Action scannt automatisch
      ↓
3. Claude Code Action postet Review
      ↓
4. Luke reviewt Findings + merged
```

---

## Quick Reference

| Aktion | Befehl/Tool |
|--------|-------------|
| Lokaler Security Scan | `/security-review` |
| Lokaler Code Review | `/code-review` |
| Code Review mit PR-Kommentar | `/code-review --comment` |
| GitHub App installieren | `/install-github-app` |
| MCP Server Status pruefen | `claude mcp list` |
| Hook-Config bearbeiten | `.claude/settings.json` |
