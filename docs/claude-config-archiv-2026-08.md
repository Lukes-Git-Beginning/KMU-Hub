# Archiv der Claude-Konfiguration — Setup-Kur 2026-08-24

Alles, was die Setup-Kur (`F:\CLAUDE-SETUP-KUR.md`) gestrichen hat, im **Wortlaut** plus Begründung
plus Fundstelle. Kur-Regel 3: *Nichts wird gelöscht, was nicht ersetzt oder archiviert ist.*

Wer eine Regel vermisst, findet hier den Originaltext **und** den Grund, warum sie ging.
Zurückholen heißt: Block kopieren, an die genannte Fundstelle einsetzen.

Vollständiges Dateibackup vom selben Stand: `~/claude-config-backup/20260824-161433/`
(416 Dateien, 4,1 MB — enthält auch, was hier aus Platzgründen nicht wiederholt wird).

Befund und Begründungsketten: `~/.claude/plans/lies-in-laufwerk-f-jaunty-dahl.md`.

---

## 1 — `~/.claude/CLAUDE.md`

### 1.1 Zeile 8 — Sprachregel (Kommunikation)

**Grund:** steht als `"language": "german"` in `~/.claude/settings.json` und ein drittes Mal im
Harness-Systemprompt. Drei Quellen für dieselbe Regel. Die beiden Zeilen darunter (Code auf Englisch,
keine ASCII-Substitutionen) sind **nicht** abgedeckt und bleiben in der Datei.

```markdown
- Kommunikation: Deutsch (orthographisch korrekt — Umlaute, Eszett, Akzente)
```

### 1.2 Zeilen 12–20 — Model-Routing

**Grund:** Anweisung an den Nutzer, nicht an das Modell — Claude kann sein eigenes Modell nicht
wechseln. Kostet trotzdem in jeder Session in jedem Projekt Kontext. Widerspricht zudem dem eigenen
Setup: `~/.claude/commands/plan.md` setzt `model: opus` im Frontmatter, während Zeile 20 sagt
"Kein Custom-Agent-Setup noetig". Umgezogen nach `docs/claude-model-routing.md`.

```markdown
## Model-Routing

Faustregel: **~80% Sonnet, ~20% Opus**.

- **Opus** (`/model opus`): Architektur-Entscheidungen, komplexes Debugging, Multi-File-Refactoring, Planung neuer Features, strategische Reviews
- **Sonnet** (`/model sonnet`): Code-Volumen, Routine-Refactors, Boilerplate, Sub-Agent-Work, CRUD/UI-Arbeit
- **Haiku**: Atomare Lookups, Classification

Fuer Planungs-+-Ausfuehrungs-Sessions: In der Hauptsession manuell per `/model` zwischen Opus (Plan-Phase) und Sonnet (Execute-Phase) wechseln. Kein Custom-Agent-Setup noetig.
```

### 1.3 Zeilen 22–26 — Effort

**Grund:** `"effortLevel": "xhigh"` steht in `~/.claude/settings.json`. Die Datei beschreibt hier
ihre eigene Konfiguration.

```markdown
## Effort

- **`xhigh`** ist der Sweet Spot fuer Coding — global gesetzt in `settings.json`
- `max` nur fuer echte Frontier-Probleme (Session-only)
- `low`/`medium` fuer einfache Tasks wenn Speed > Tiefe
```

### 1.4 Zeilen 28–31 — Prompt Caching

**Grund:** `ENABLE_PROMPT_CACHING_1H` steht in `~/.claude/settings.json` → `env`. Der
`grep`-Befehl zum Prüfen der Cache-Hits ist ein Nachschlage-Detail, kein Fakt, den Claude braucht.

```markdown
## Prompt Caching

- `ENABLE_PROMPT_CACHING_1H=1` global aktiv — spart Tokens bei langen Sessions mit viel statischem Context (CLAUDE.md, Knowledge-Base)
- Cache-Hits pruefen via `grep ephemeral_1h_input_tokens ~/.claude/projects/*.jsonl`
```

### 1.5 Zeilen 33–37 — Sub-Agents

**Grund:** `CLAUDE_CODE_SUBAGENT_MODEL` steht in `settings.json`. Die Bitte "max 3 gleichzeitig"
ist als Prosa unverbindlich und wurde durch `CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS: "3"` in
`settings.json` ersetzt — eine Umgebungsvariable kann nicht ignoriert werden, eine Bitte schon.

```markdown
## Sub-Agents

- `CLAUDE_CODE_SUBAGENT_MODEL=sonnet` global — alle Sub-Agents (Explore, general-purpose, Plan) laufen auf Sonnet als Kosten-Baseline
- Fuer grosse Explorationen und Research-Batches: parallele Sub-Agents (max 3 gleichzeitig) um das Haupt-Context-Window klein zu halten
- Opus-Sub-Agents nur mit expliziter `model: opus` im Agent-Frontmatter
```

### 1.6 Zeilen 54–61 — Feature-Liste "verifiziert, Stand 2026-04-18"

**Grund:** handgepflegte Feature-Liste, vier Monate alt und teilweise falsch — Zeile 57 führt
`/ultrareview` als eigenes Feature, während es inzwischen ein veralteter Alias für
`/code-review ultra` ist. Der Kostenhinweis (5–20 USD) ist inhaltlich richtig und wurde als eine
Zeile in die neue Datei gerettet. Ersatz für den Rest: `/doctor` und die Release-Notes.

```markdown
## Claude-Code-Features (verifiziert, Stand 2026-04-18)

- **`ultrathink`** Keyword im Prompt — verfuegbar, triggert tieferes Reasoning per System-Reminder. Bedenkenlos nutzen fuer schwere Reasoning-Tasks (keine Zusatzkosten ausser Token-Abrechnung).
- **`/ultrareview`** Cloud-Multi-Agent-Review — verfuegbar, **aber teuer: 5–20 USD pro Ausfuehrung** (Kosten-Disclaimer im Slash-Command). NUR gezielt fuer Security-kritische PRs oder Architektur-Reviews einsetzen.
- **`/model opusplan`** — existiert nicht. Stattdessen manueller Workflow: `/model opus` (Plan-Phase) → `/model sonnet` (Execute-Phase).
- **`task_budget`** Adaptive-Thinking-Feld — API-spezifisch, in CLI nicht relevant.

**Regel:** Neue Community-Features weiterhin erst in Live-Session testen, bevor persistent konfiguriert.
```

---

## 2 — `~/.claude/settings.json`

### 2.1 Zeilen 68–73 — Permissions für den `knowledge`-MCP-Server

**Grund:** Der Server selbst ist gestrichen (siehe 4.1). Permissions verweisen auf Tool-*Namen*;
ohne Server sind sie tot.

```json
      "mcp__knowledge__read_file",
      "mcp__knowledge__read_text_file",
      "mcp__knowledge__list_directory",
      "mcp__knowledge__directory_tree",
      "mcp__knowledge__list_allowed_directories",
      "mcp__knowledge__search_files",
```

### 2.2 Zeilen 92–94 — `enabledPlugins`

**Grund:** Das Plugin dupliziert den Projekt-Skill `.claude/skills/frontend-design/`, der auf Cosmi
angepasst ist. `pluginUsage` in `~/.claude.json` zeigt für das Plugin **usageCount 0** seit
Startup 83 — es wurde nie benutzt, stand aber in jeder Session im Auswahlraum.
Zurückholen: `/plugin` oder diesen Block wieder einsetzen.

```json
  "enabledPlugins": {
    "frontend-design@claude-plugins-official": true
  },
```

### 2.3 Zeilen 101–115 — SessionStart-Hook `claude update`

**Grund:** wirkungslos. Das Binary ist vom 04.08.2026, seither liefen 178 Sessionstarts. In
`~/.claude.json` steht `autoUpdates: false`, was `autoUpdatesChannel: latest` in derselben
`settings.json` schlägt. Ein Hook, der 178-mal nichts tut, ist ein Kommentar. Updates laufen
stattdessen von Hand über `npm i -g @anthropic-ai/claude-code`.

```json
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "claude update",
            "async": true,
            "timeout": 120,
            "statusMessage": "Checking for Claude Code updates..."
          }
        ]
      }
    ]
  }
```

---

## 3 — `KMU Hub/CLAUDE.md`

### 3.1 Zeilen 78–86 — Git-Regeln

**Grund:** stehen wörtlich in `~/.claude/CLAUDE.md` (gilt projektübergreifend) und ein drittes Mal
in MEMORY.md. Der wichtigste Teil ist zusätzlich per Hook erzwungen:
`.claude/hooks/check-commit-message.sh` blockt Nicht-Conventional-Commits, der neue
`.claude/hooks/check-no-attribution.sh` blockt AI-Attribution. Enforcement schlägt Bitte.

```markdown
## Git-Regeln

- **Conventional Commits:** `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`
- **Branch-Strategie:** Ab Sprint 1 (2026-04-18) **direct-to-main ist Default**. Keine Feature-Branches, keine PRs — ausser explizit gefordert
- **CI-Rot-Recovery:** `git revert <sha>`. **NIE** `reset --hard`, **NIE** Force-Push
- **Keine AI-Attribution** (kein `Co-Authored-By`, kein "Generated by")
- **Commit-Messages:** Englisch, imperativ ("Add contact endpoint", nicht "Added...")
- **Push-Rhythmus:** Am Ende jeder Session, um Divergenz zu vermeiden

```

### 3.2 Zeilen 93–114 — Knowledge-Vault-Tabelle

**Grund:** 15 Tabellenzeilen, die den Inhalt von `.knowledge/_index.md` doppeln — dort steht der
Master-Index bereits und wird gepflegt. Ersetzt durch einen Verweis. Der Vault selbst bleibt
unverändert.

```markdown
## Knowledge-Vault (.knowledge/)

Single Source of Truth fuer projektspezifisches Wissen. Notes haben YAML-Frontmatter (`tags`, `updated`), verlinken via `[[note-name]]`. Lesen via MCP-Filesystem-Tools (`mcp__knowledge__read_text_file`).

| Note | Inhalt |
|------|--------|
| `_index.md` | Master-Index, 6-Sprint-Roadmap bis Launch 01.09 |
| `architektur.md` | Services, Routes, Architektur-Regeln (Detail mit Code), Entwicklungs-Kommandos, Feature-Flags, Consent-Wrapper, WASM-OFF |
| `stack.md` | Strategy Decisions, Frontend-Bibliotheken, dompurify, Mobile=PWA |
| `i18n.md` | i18next-Architektur, Schluessel-Konventionen, ICU-Plural-Bug |
| `design.md` | Design-System, Themes, UI/UX-Direktiven, Magic UI, LanguageSwitcher |
| `datenbank.md` | Schema, Migrationskopf 000213 (Prod 209) |
| `api.md` | 28 Endpoint-Domains, Auth-Flow |
| `security.md` | JWT, RBAC, Consent-Enforcement, Prod-Secrets-Assertion, DOMPurify |
| `integrationen.md` | Bexio, Lexware, DATEV, LiveKit, OnlyOffice JWT, Plugin-WASM (off) |
| `deployment.md` | Docker, CI/CD, Hetzner, `make build-prod -tags no_wasm` |
| `testing.md` | Test-Strategie, Sprint-0-Neuzugaenge |
| `pricing.md` | Modul-x-User Preismodell (COSMI + ORBIT) |
| `troubleshooting.md` | Bekannte Probleme, Git-Workflow, Tailwind/CSP, ICU-Plural, Radix-Null |
| `tooling-graphify.md` | Graphify-Eval (vertagt auf Sprint 2/3) |
| `milestones.md` | Meilensteine inkl. Rigorosum Runde 1+2 + Sprint 0 Closure |

```

### 3.3 Zeilen 115–118 — Intel-System

**Grund:** `~/Documents/zentria-intel` existiert auf dieser Maschine nicht — der Abschnitt und die
sechs `intel-*`-Commands beschreiben ein Repo, das hier nicht liegt. Die Commands bleiben
eingecheckt (sie gelten fürs Team), nur ihre Beschreibungen wurden gekürzt. Zum Reaktivieren:
Repo klonen, dann greifen sie wieder.

```markdown
## Intel-System (zentria-intel)

Paralleles Markt-Intelligence-Repo (`~/Documents/zentria-intel/`, github.com/Lukes-Git-Beginning/zentria-intel) — taeglicher CRM-/KMU-Markt-Scan, Friday-Synthese, Discord-Pick-Mechanik, Recall in Cosmi-Sessions. Memory-Pointer in `MEMORY.md > ## Intel-System`.

```

---

## 4 — `.mcp.json`

### 4.1 `knowledge` — Filesystem-Server auf `.knowledge`

**Grund:** `@modelcontextprotocol/server-filesystem` auf ein Verzeichnis **innerhalb** des Projekts,
auf das Read/Glob/Grep ohnehin zugreifen. Er brachte ~11 Tools in den Auswahlraum, die nichts
hinzufügen, plus einen `npx`-Prozess pro Session. Erster Eintrag der Server-Tabelle der Kur
("filesystem → die eingebauten Datei-Werkzeuge"). Der Vault `.knowledge/` selbst ist unberührt und
wird jetzt mit Read/Grep gelesen — schneller als `search_files` über MCP.

```json
"knowledge": { "command": "cmd", "args": ["/c","npx","-y","@modelcontextprotocol/server-filesystem",".knowledge"] }
```

### 4.2 `github` — HTTP-Server ohne Token

**Grund:** `CLAUDE_GITHUB_TOKEN` ist in der Umgebung nicht gesetzt und steht in keiner
Konfigurationsdatei — der Server sendete `Authorization: Bearer ` (leer). Parallel ist `gh` CLI
2.87.3 installiert und authentifiziert nutzbar. Zum Reaktivieren: Token in die Umgebung legen und
diesen Block wieder einsetzen.

```json
"github": { "type": "http", "url": "https://api.githubcopilot.com/mcp/", "headers": { "Authorization": "Bearer ${CLAUDE_GITHUB_TOKEN:-}" } }
```

---

## 5 — `.claude/settings.local.json` (gitignored, nur diese Maschine)

### 5.1 Einmal-Permissions

**Grund:** Reste einmaliger Aktionen, dauerhaft freigeschaltet — der `ralph-claude-code`-Umzug
(vom Nutzer als veraltet bestätigt), drei `unzip`-Varianten für ein Bundle, ein `cp` für
`plan.md`. `PowerShell(robocopy *)` war die breiteste Regel der Datei; robocopy mit `/MOVE` kann
Verzeichnisse leeren. Kostet keinen Kontext — gestrichen, weil eine Allowlist nur so viel wert ist
wie ihre Genauigkeit.

```json
      "PowerShell($proc = Get-Process | Where-Object { $_.Name -match \"git|node|ralph|claude|bash\" } | Select-Object Id, Name, Path; $proc | Format-Table -AutoSize)",
      "PowerShell(Get-CimInstance *)",
      "PowerShell(robocopy \"C:\\\\Users\\\\darie\\\\Documents\\\\KMU Hub\\\\ralph-claude-code\" \"C:\\\\Users\\\\darie\\\\Documents\\\\ralph-claude-code\" /E /MOVE /NFL /NDL /NJH /NJS /NC /NS 2>&1; Write-Output \"ExitCode=$LASTEXITCODE\"; if \\(Test-Path \"C:\\\\Users\\\\darie\\\\Documents\\\\ralph-claude-code\\\\.git\"\\) { Write-Output \"TARGET .git EXISTS\" }; if \\(Test-Path \"C:\\\\Users\\\\darie\\\\Documents\\\\KMU Hub\\\\ralph-claude-code\"\\) { Write-Output \"SOURCE STILL EXISTS\" } else { Write-Output \"SOURCE GONE\" })",
      "PowerShell(robocopy *)",
      "Read(//c/Users/darie/Downloads/**)",
      "Bash(unzip -l \"C:/Users/darie/Downloads/claude-settings-bundle.zip\")",
      "Bash(mkdir -p \"/tmp/luke-bundle\")",
      "Read(//tmp/**)",
      "Bash(unzip -o \"C:/Users/darie/Downloads/claude-settings-bundle.zip\")",
      "Bash(mkdir -p \"C:/Users/darie/AppData/Local/Temp/luke-bundle\")",
      "Read(//c/Users/darie/AppData/Local/Temp/luke-bundle/**)",
      "Bash(mkdir -p \"/c/Users/darie/AppData/Local/Temp/luke-bundle\")",
      "Bash(unzip -o \"/c/Users/darie/Downloads/claude-settings-bundle.zip\")",
      "Read(//c/Users/darie/.claude/**)",
      "Bash(mkdir -p \"/c/Users/darie/.claude/commands\")",
      "Bash(cp \"/c/Users/darie/AppData/Local/Temp/luke-bundle/global/commands/plan.md\" \"/c/Users/darie/.claude/commands/\")",
      "Bash(npm view *)",
      "Bash(npx --version)",
      "Bash(timeout 15 cmd //c \"npx -y @modelcontextprotocol/server-filesystem .knowledge\")",
      "Bash(timeout 30 cmd //c \"npx -y @modelcontextprotocol/server-filesystem .knowledge\")",
      "Bash(chmod +x ~/.claude/statusline.sh)",
```

---

## 6 — `.gitignore`

### 6.1 Drei `gsd-*`-Zeilen

**Grund:** verweisen auf `.claude/hooks/gsd-*` und `.claude/commands/gsd/` — beide existieren
nicht. Das zugehörige `get-shit-done`-Werkzeug (272 KB unter `.claude/get-shit-done/bin/`) wurde
gelöscht; `skillUsage` zeigt `gsd:update` einmal, zuletzt am 26.02.2026.

```gitignore
# Zeile 59:.claude/hooks/gsd-*
# Zeile 61:.claude/commands/gsd/
# Zeile 62:.claude/commands/gsd-*
```

---

## 7 — `MEMORY.md` (Auto-Memory-Index, gitignored)

Mit 18.973 Zeichen war sie der größte Einzelposten im Kontext — mehr als beide CLAUDE.md zusammen —
und über ihrem eigenen Budget (der `memory-size-guard`-Hook meldete das bei jedem Start).

**Die 74 verlinkten Einzelmemos sind vollständig erhalten.** Gekürzt wurde nur der *Index*:
viele Pointer-Zeilen hatten den Inhalt der verlinkten Datei zusammengefasst statt auf sie zu
verweisen. Der Vorgabe nach ist eine Indexzeile `- [Titel](datei.md) — Haken`.
Vollständiger Originalstand: `~/claude-config-backup/20260824-161433/memory/MEMORY.md`.

### 7.1 Zeilen 40–58 — Abschnitt "User Preferences (IMMER befolgen)"

**Grund:** wiederholt wörtlich, was in `~/.claude/CLAUDE.md` steht (Git-Regeln, Planungs-Haltung,
Sprache, thick services / thin handlers). Dritte Quelle für dieselben Regeln.

```markdown
## User Preferences (IMMER befolgen)

### Git
- KEINE AI-Attribution in Commits — kein Co-Authored-By, kein "generated by Claude", nichts
- Conventional Commits: feat:, fix:, docs:, refactor:, test:, chore:
- English, imperative mood ("Add feature", nicht "Added feature")
- Push am Ende jeder Session (avoid divergence)

### Planung
- Mehr Fragen stellen waehrend Planung — nicht annehmen
- Komplex und zukunftsorientiert planen, NICHT short-term und simpel
- Extensibility und langfristige Architektur mitdenken
- Lieber over-planen als under-planen

### Kommunikation & Code
- Sprache: Deutsch fuer Kommunikation, Englisch fuer Code
- Structured logging (slog), kein fmt.Println
- Thick services, thin handlers

```

### 7.2 Zeilen 83–95 — Abschnitt "Laufender Sprint", Langfassung

**Grund:** Zustandsbeschreibung mit Datum — besteht den Halbwertszeit-Test nicht. Der Absatz
"▶ STAND 2026-08-20" allein war ~700 Zeichen und ist in vier Wochen überholt. Ersetzt durch drei
Zeilen mit Verweis auf `.planning/launch-lagebild-2026-08-12.md`, das ohnehin die Single Source of
Truth für Lage und Sequenz ist.

```markdown
## Laufender Sprint

- **#38 (2026-08-12) Launch-Lagebild** — kein Kalenderdatum mehr, dafür Gates/Etappen; Todesursache aller vier Pre-Mortem-Linsen war die Außendarstellung (Vercel/Resend trotz „EU-Souveränität"). Historie #32–#37 → `archive/editor_sessions_32_36.md`.
- **#39 (2026-08-13) Modul-Streichliste steht** (Darien: 11 raus — Meetings, Buchhaltung, Inventar, Einkauf, Fuhrpark, Produktion, Berichte, Formulare, Vermietung, Rapporte, Dialer; 14 bleiben) + Preis-/Kostenanalyse `fdb9d0ca` → [[project_preismodell_1_0]]. Endet mit 12 Entscheidungsfragen, **in der Runde nie besprochen**.
- **▶ STAND 2026-08-20 (Session #40):** Darien war 6 Tage weg, **Luke hat in 3 Tagen 40 Commits gebaut** (16./17./19.08.) und dabei Etappe 0–3 abgearbeitet — Website auf eigenem Server, Cosmi als **Web-App live** auf `app.zentria.tech`, DSGVO-/Backup-Welle. Gemessen: CI/CD grün, Deploy = `71d830a7`, `/health` meldet redis+postgres. Details + was offen ist → [[project_etappen_gates_1_0]]. **Weder** die Website-Redaktionsarbeit **noch** der Editor-Rollout haben begonnen. Darien ist unterwegs und wollte alle offenen Fragen als PDF → **29 Fragen in 5 Teilen (A1–A12, B1–B4, C1–C7, D1–D4, E1–E2)** ausgeliefert, Quelle committet (`0f3646a5`, `.planning/offene-entscheidungen-2026-08-20.html`, PDF via `offene-entscheidungen-render.mjs`, `.planning/*.pdf` ist gitignored). **Er schickt die Antworten als Kennungs-Liste zurück** („A1: B mit 79…") — dann in die Planungsdokumente nachziehen und mit dem starten, was er unter E2 nennt.
- [Etappen & Gates zu 1.0](project_etappen_gates_1_0.md) — Etappe 0–2 erledigt, Etappe 3 gebaut; **Gate 2 offen (Login)**, Vercel-Löschung + echter Prod-Restore offen. SSOT: Lagebild §3/§6
- [Preis- und Kostenmodell 1.0](project_preismodell_1_0.md) — Streichliste, Grundgebühr-Vorschlag 79 €, Break-Even je Besetzung, 12 offene Entscheidungen. SSOT: `.planning/preis-und-kostenanalyse-2026-08-13.md`
- [Editor-Rollout ab 2026-08-11](project_editor_rollout.md) — Editor ist gebaut, jetzt Module ausrollen. Vorlage `docs/EDITOR-MODULE-ROLLOUT.md` (Abschnitt 0 = Zuschnitt + Reihenfolge + Abnahme). Referenzen: Helpdesk, Kontakte. ✅ Streichliste seit 2026-08-13 da — `finanzen` ist raus, Start mit Dokumente/Kalender
- [Auslieferungsmodell (Vorschläge, offen)](project_auslieferungsmodell.md) — Web vs. Desktop + eigener Server pro Kunde + nur gebuchte Module ausliefern. Von der Runde NICHT entschieden. Code-Befunde: `API_BASE_URL` einkompiliert, keine Registry, echtes Weglassen möglich
- [Website-Repo bei Darien](reference_website_repo.md) — `C:\Users\darie\Documents\zentria-website`, seit 2026-08-12 auf `main` (nicht mehr `dev/redesign`)
- **Pre-Launch-Sprint 2026-04-21 bis 2026-06-30** (6 Sprints, Details `docs/ROADMAP.md`)
- [Dialer-Modul Phase 1 + ZFA-Pilot-0](project_dialer.md) — Feature-komplett, Coverage 12% → ≥30% Sprint 2, Phase 2 PSTN vertagt Q3

```

### 7.3 Memo `claude_code_features.md` — nach `memory/archive/` verschoben

**Grund:** dieselbe handgepflegte Feature-Liste wie 1.6, nur als Memo. Inhalt:
"`ultrathink` ✅, `/ultrareview` ✅ (5–20 USD/Call!), `opusplan` ❌". Der Kostenhinweis ist in die
globale CLAUDE.md gerettet; der Rest war zum Zeitpunkt der Kur bereits überholt.
Neuer Ort: `memory/archive/claude_code_features.md` — nicht gelöscht, nur aus dem Index genommen.

### 7.4 Sprint-Historie #38–#40

Woertlich ausgelagert nach `memory/archive/editor_sessions_37_40.md` (Muster wie #32–#36).

---

## 8 — Einzelne Zeilen, die umformuliert statt gestrichen wurden

Diese Zeilen stehen im Ergebnis weiter, aber mit anderem Wortlaut: Datum, Zaehlung oder
Versionsstand wurden entfernt, die Regel selbst blieb. Hier der Originalwortlaut.

### 8.1 `KMU Hub/CLAUDE.md`

```markdown
- **Ziel:** **Produkt 1.0.0** nach Reifegrad-Gates, **kein Kalenderdatum** (das Launch-Datum 2026-09-01 ist seit 2026-08-12 entwertet). Definition und Sequenz: `.planning/launch-lagebild-2026-08-12.md` §3 und §6
- **Version:** 0.1.0
| Backend | Go (API-Gateway + 23 gRPC-Microservices = 24 `backend/cmd/*`-Dirs) |
11. **Tenant-Modell** — Option-B-Full: alle Tabellen `tenant_id UUID NOT NULL` + Row-Level-Security. RLS **produktiv erzwungen** (`COSMI_ENV=production` scharf seit 2026-06-05); App-Services laufen als `kmuhub_app` (NOSUPERUSER NOBYPASSRLS), DDL-Migrations als `kmuhub`. Daten aktuell Single-Tenant, Code Multi-Tenant-fähig. Neue Tabellen MUESSEN `tenant_id` + RLS-Policy haben — oder explizit in die System-Global-Liste (ADR-006, `docs/ARCHITECTURE.md`) eingetragen werden
- **Skills:** `frontend-design` (auto-loaded), `impeccable` (`/audit`, `/critique`, `/polish`, `/animate` etc.). On-demand: `emilkowalski/skill`, `kylezantos/design-motion-principles`.
```

Was jeweils wegfiel und warum:

| Zeile | Entfernt | Grund |
|---|---|---|
| Ziel | „(das Launch-Datum 2026-09-01 ist seit 2026-08-12 entwertet)" | doppelt entwertet — die Aussage „kein Kalenderdatum" traegt sich selbst |
| Version | ganze Zeile `- **Version:** 0.1.0` | Versionsnummer, driftet, steht in `package.json` |
| Backend | „23 gRPC-Microservices = 24 `backend/cmd/*`-Dirs" | Zaehlung driftet mit jedem neuen Service; `ls backend/cmd` ist genauer |
| Regel 11 | „(`COSMI_ENV=production` scharf seit 2026-06-05)" | Datum. Die Pflicht (`tenant_id` + RLS-Policy) blieb woertlich |
| Skills | „(auto-loaded)", `emilkowalski/skill`, `kylezantos/design-motion-principles` | Skill-Namen stimmten nicht mehr: heute `emil-design-eng` und `design-motion-principles` |

### 8.2 `~/.claude/CLAUDE.md` — Meta-Vorspann

**Grund:** beschreibt, was die Datei ist. Besteht den Aufnahmetest nicht — ihr Weglassen
laesst Claude keinen Fehler machen.

```markdown
Persistente Praeferenzen fuer alle Claude-Code-Sessions (projekt-unabhaengig).
Projekt-spezifische Regeln stehen in `<projekt>/CLAUDE.md`.
```

### 8.3 `MEMORY.md` — Index-Hooks

Rund 70 Pointer-Zeilen wurden gekuerzt: sie fassten den Inhalt der verlinkten Memo-Datei
zusammen, statt auf sie zu verweisen. Titel und Dateilink sind bei allen unveraendert
(62 von 63 Links erhalten; der 63. zeigte auf die veraltete Feature-Notiz, siehe 7.3).
Der vollstaendige alte Index steht woertlich in
`~/.claude/projects/C--Users-darie-Documents-KMU-Hub/memory/archive/MEMORY-index-vor-kur-2026-08-24.md`.

