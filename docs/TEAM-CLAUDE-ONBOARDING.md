# Claude Code — Onboarding für Darien & Nico

> **Wichtig:** Alle Settings (`~/.claude/`) bekommt ihr von Luke verschlüsselt per Discord oder USB-Stick. Sobald der Ordner bei euch liegt, seid ihr sofort startklar — die meisten Konfigurationen sind schon gemacht.

### 1. Settings von Luke übernehmen

Luke schickt euch einen **verschlüsselten Ordner** (`.claude.zip` oder ähnlich). Entpackt ihn nach:

- **Windows:** `C:\Users\<euer-username>\.claude\`
- **Mac/Linux:** `~/.claude/`

**WICHTIG:** Passt in den Dateien, die absolute Pfade enthalten, `Luke` auf euren Windows-Usernamen an. Die betroffenen Files sind vor allem:

- `~/.claude/CLAUDE.md` (Pfad-Referenzen)
- `~/.claude/projects/...` (Memory-Ordner)

### 2. Repo klonen

```bash
git clone git@github.com:Lukes-Git-Beginning/KMU-Hub.git
cd KMU-Hub
claude    # Claude startet und liest automatisch CLAUDE.md + MEMORY.md
```

---

## 3. Was ist schon vorkonfiguriert?

Damit ihr wisst, warum Claude bei euch sofort so "klug" wirkt:

| Einstellung | Wert | Was das bedeutet |
|-------------|------|------------------|
| **Sprache** | Deutsch | Claude antwortet auf Deutsch (mit Umlauten!) |
| **Default-Modell** | Opus 4.7 | Das stärkste aktuelle Modell |
| **Sub-Agents** | Sonnet | Hintergrund-Tasks laufen günstiger auf Sonnet |
| **Effort-Level** | xhigh | Claude denkt tiefer nach (mehr Reasoning) |
| **Thinking** | Always-on | Claude zeigt sein "Gedankenflussdiagramm" |
| **Prompt Caching** | 1h TTL | Spart massiv Tokens (= Geld) bei langen Sessions |
| **Voice** | Aktiviert | Ihr könnt per Sprache mit Claude reden |

### Vorkonfigurierte Permissions

Claude darf **ohne Nachfrage**:
- Git-Commands (`add`, `commit`, `push`, `status`, `diff`, `log`, …)
- Go-Commands (`go test`, `go build`, `go run`, `make`)
- npm/npx (`vitest`, `tsc`, `eslint`, `shadcn`, `vite`)
- Docker-Compose
- Knowledge-Base lesen

Alles andere fragt er — einfach mit `1` (Allow once) oder `2` (Always) bestätigen.

---

## 4. Model-Routing — Die 80/20-Regel

**Wichtigste Regel der gesamten Nutzung.** Opus ist teuer, Sonnet ist günstig. Faustregel:

- **80% Sonnet** — für das Grobe: Code schreiben, Refactors, Bugfixes, Boilerplate, UI-Arbeit
- **20% Opus** — für das Schwere: Architektur-Entscheidungen, komplexes Debugging, Multi-File-Refactors, Planung neuer Features

**Workflow in einer Session:**

```
1. Plan-Phase:     /model opus      (Claude denkt über Architektur nach)
2. Execute-Phase:  /model sonnet    (Claude setzt den Plan um)
3. Review:         /model opus      (Claude reviewed das Ergebnis)
```

Einfach `/model opus` oder `/model sonnet` in der Session tippen, Claude wechselt sofort.

> **Tipp von Luke:** Wenn ihr unsicher seid, fragt Claude: *"Welches Modell würdest du für diesen Task nutzen?"* — er schlägt meistens das richtige vor.

---

## 5. Session-Workflow (Best Practice)

### 5.1 Session starten

```bash
cd "C:/Users/<euer-name>/Documents/KMU Hub"
claude
```

Claude liest automatisch:
- `CLAUDE.md` im Projekt-Root → Projekt-Regeln
- `~/.claude/CLAUDE.md` → persönliche Präferenzen
- `~/.claude/projects/.../memory/MEMORY.md` → Projektstand & Entscheidungen

### 5.2 Erstes Prompt — gutes Beispiel

❌ **Schlecht:** *"Bau mir eine Kontaktliste"*

✅ **Gut:** *"Ich will eine neue Kontaktliste im CRM-Modul bauen. Schau dir erst `desktop/src/renderer/src/modules/contacts/` an, dann schlag mir eine Architektur vor. Wir nutzen dort shadcn/ui + Plus Jakarta Sans. Keine Implementierung, nur Plan."*

**Prinzip:**
1. Context geben (wo soll es hin, was gibt es schon)
2. Ziel klar nennen (was soll rauskommen)
3. Einschränkungen benennen (Stack, Regeln)
4. Explizit sagen: Plan oder Implementation?

### 5.3 Planning vor Execution

Bei größeren Tasks immer **erst planen**, dann umsetzen:

```
> /plan bau mir ein Dashboard-Widget für offene Deals
```

Claude geht in Plan-Mode, zeigt euch einen Plan, ihr könnt ihn anpassen — erst dann wird Code geschrieben.

### 5.4 Session beenden

- **Immer committen und pushen** — sonst divergieren eure Branches
- Claude fragt bei `git push`: mit `1` bestätigen
- Am Ende: `/exit` oder einfach Terminal schließen

---

## 6. Skills — Claude's Spezial-Skills

Skills sind wie kleine Experten, die Claude je nach Aufgabe aktiviert. Ihr ruft sie mit `/<skill-name>` auf oder Claude wählt selbst.

### 6.1 Die wichtigsten Skills

**Frontend-Design (automatisch aktiv):**
- `/audit` — Qualitäts-Check der UI (P0-P3 Severity)
- `/critique` — UX-Review gegen Nielsen's Heuristiken
- `/polish` — Final Pass vor dem Ship
- `/animate` — Animationen hinzufügen
- `/bolder` — UI mutiger machen (kein generischer Look)
- `/distill` — Clutter entfernen, Fokus schaffen
- `/harden` — Production-Ready machen (Errors, i18n, Edge Cases)

**Workflow für neue UI-Features:**
```
1. Generiere das Feature
2. /audit           → Qualitäts-Check
3. /critique        → UX-Review
4. Fixes umsetzen
5. /polish          → Final Pass
6. Ship
```

**Review & Quality:**
- `/review` — PR-Review
- `/security-review` — Security-Check
- `/simplify` — Code aufräumen

**Knowledge:**
- `/update-knowledge` — `.knowledge/` aktualisieren nach Feature-Abschluss

### 6.2 `/ultrareview` — VORSICHT!

Es gibt einen Cloud-Multi-Agent-Review namens `/ultrareview`, der **5–20 USD pro Aufruf** kostet. Nur einsetzen für:
- Security-kritische PRs
- Architektur-Reviews vor großen Releases
- Niemals für Routine-Changes

---

## 7. Memory-System — Claude merkt sich Dinge

Claude hat zwei Arten von Gedächtnis:

### 7.1 MEMORY.md (wird automatisch geladen)

Liegt in `~/.claude/projects/C--Users-Luke-Documents-KMU-Hub/memory/MEMORY.md`. Enthält:
- Aktueller Projektstand
- User-Präferenzen (z.B. "keine AI-Attribution in Commits")
- Launch-Datum, laufende Sprints
- Verweise auf Detail-Memories

### 7.2 Detail-Memories (bei Bedarf nachgeladen)

Einzelne `.md`-Files im gleichen Ordner, z.B.:
- `project_launch_decisions.md`
- `project_dialer.md`
- `project_team_ug.md`

### 7.3 Was ihr tun müsst

**Meistens: Nichts.** Claude pflegt das selbst. Wenn ihr ihm etwas Wichtiges sagt, das zukünftige Sessions wissen sollen, sagt einfach:

> *"Merk dir: Unser Launch-Datum ist jetzt 01.07.2026, nicht mehr 01.06."*

Claude legt dann eine Memory-Datei an oder aktualisiert eine bestehende.

**Niemals:** Memory-Files manuell editieren, während Claude läuft. Er merkt es nicht und überschreibt eure Änderungen.

---

## 8. Knowledge-Base (`.knowledge/`)

Zusätzlich zu Memory hat das Projekt eine Knowledge-Base: der Ordner `.knowledge/` im Repo-Root. Das ist ein **Obsidian Vault** mit technischer Dokumentation:

| Datei | Inhalt |
|-------|--------|
| `architektur.md` | Services, Routes, Demo-Mode, i18n |
| `datenbank.md` | Schema, 70 Migrations |
| `api.md` | Alle Endpoints |
| `security.md` | JWT, RBAC, CORS, GDPR |
| `integrationen.md` | Bexio, Lexware, DATEV, LiveKit |
| `troubleshooting.md` | Bekannte Probleme |

**Ihr könnt den Ordner in Obsidian öffnen** und habt dann eine schöne Graph-Ansicht eurer gesamten Projekt-Doku. Claude schreibt hier auch rein, wenn er was Neues lernt — aktualisiert via `/update-knowledge` nach größeren Änderungen.

---

## 9. Die goldenen Projekt-Regeln

Diese stehen in `CLAUDE.md` im Repo-Root — Claude befolgt sie automatisch, aber **ihr solltet sie auch kennen**:

### 9.1 Git-Regeln

- **Conventional Commits:** `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`
- **Englisch, Imperativ:** `"Add contact endpoint"` — nicht `"Added..."`
- **KEINE AI-Attribution:** Weder `Co-Authored-By: Claude` noch `🤖 Generated with...` — nie!
- **Branch-Strategie:** `feature/*` → `develop` → `main`
- **PR-Pflicht:** Kein direkter Push auf `main`

### 9.2 Code-Regeln

- **Thick Services, Thin Handlers** — Business-Logik gehört nicht in HTTP-Handler
- **Structured Logging** (`slog`) — niemals `fmt.Println()`
- **Migrations via Tool** — nie manuell SQL auf der DB
- **Tests für jeden PR** — Coverage-Minimum 15%, Ziel 30%+
- **Keine Secrets im Code** — immer via Environment Variables

### 9.3 UI/UX-Regeln (UI-Arbeit!)

- **Verbotene Fonts:** Inter, Roboto, Arial, Space Grotesk, Helvetica
- **Erlaubte Display-Fonts:** Plus Jakarta Sans (aktuell), Clash Display, Satoshi
- **Kein Card-in-Card Nesting**
- **Kein grauer Text auf farbigem Background** (WCAG AA)
- **Motion:** `cubic-bezier(0.22, 1, 0.36, 1)`, 200-350ms Interactions
- **`prefers-reduced-motion` IMMER respektieren**

---

## 10. Häufige Anfänger-Fehler (vermeiden!)

### ❌ Zu vage Prompts
*"Mach die UI schöner"* → Claude rät. Besser: *"Der Contact-List-Header fühlt sich flach an. Nutze `/bolder` auf `ContactList.tsx` und fokussier auf Typografie-Kontrast."*

### ❌ Opus für alles nutzen
Teuer und unnötig. Standard = Sonnet, Opus nur bei echtem Bedarf.

### ❌ Session nicht mit Context starten
Claude liest zwar automatisch `CLAUDE.md` + `MEMORY.md`, aber er weiß nicht, **was ihr gerade konkret vorhabt**. Erstes Prompt = Kontext-Setting.

### ❌ Memory-Files manuell editieren
Claude macht das selbst. Wenn ihr was ändern wollt, sagt es ihm.

### ❌ Destruktive Commands blind bestätigen
Wenn Claude `rm -rf`, `git reset --hard`, `git push --force` vorschlägt — **erst lesen, dann bestätigen**. Er fragt normalerweise, aber wachsam bleiben.

### ❌ Commits selbst schreiben wollen
Einfach sagen: *"Commit das mit einer passenden Message."* Claude macht conventional commits korrekt.

---

## 11. Cheat-Sheet — Die wichtigsten Commands

| Command | Wirkung |
|---------|---------|
| `/model opus` | Auf Opus wechseln (Planung, Debugging) |
| `/model sonnet` | Auf Sonnet wechseln (Code schreiben) |
| `/plan` | Plan-Mode aktivieren |
| `/audit` | UI-Qualitäts-Check |
| `/critique` | UX-Review |
| `/polish` | Final-Pass UI |
| `/review` | PR-Review |
| `/security-review` | Security-Check |
| `/update-knowledge` | Knowledge-Base aktualisieren |
| `/help` | Alle Commands anzeigen |
| `/exit` | Session beenden |
| `! <command>` | Command in eurer Shell ausführen (z.B. `! gcloud auth login`) |
| `Esc` (zweimal) | Claude unterbrechen |

---

## 12. Troubleshooting

**"Claude versteht mein Projekt nicht."**
→ Ihr seid vermutlich nicht im Projekt-Root. `cd "C:/Users/<ihr>/Documents/KMU Hub"`, dann `claude`.

**"Claude spricht plötzlich Englisch."**
→ In `~/.claude/settings.json` muss `"language": "german"` stehen. Checken!

**"Permissions-Prompt nervt bei jedem Git-Command."**
→ Die Settings-Datei war nicht korrekt übernommen. Nochmal von Luke holen und prüfen ob `~/.claude/settings.json` existiert.

**"Memory wird nicht geladen."**
→ Pfad muss exakt sein: `~/.claude/projects/C--Users-<name>-Documents-KMU-Hub/memory/`. Bei abweichendem Namen funktioniert's nicht — Ordner ggf. umbenennen.

**"Mein Sub-Agent läuft auf Opus — teuer!"**
→ `CLAUDE_CODE_SUBAGENT_MODEL=sonnet` in Environment setzen (sollte in Settings schon drin sein).

---

## 13. Wichtige Links

- **Claude Code Docs:** https://docs.claude.com/en/docs/claude-code
- **Anthropic Console:** https://console.anthropic.com (Token-Usage checken)
- **Issues:** https://github.com/anthropics/claude-code/issues
- **Hilfe in der Session:** `/help`

---

## 14. Fragen?

**Luke** ist erste Anlaufstelle für alles Projekt-spezifische. Bei Claude-Code-Bugs: direkt auf GitHub melden.

Viel Spaß beim Code-Pairing mit Claude — und nicht vergessen: **er ist ein Werkzeug, kein Ersatz für euer Urteil**. Immer lesen, was er schreibt, bevor ihr's committet.

Good luck & happy shipping! 🚀

— Luke + Claude
