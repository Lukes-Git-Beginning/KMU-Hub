# Setup-Kur — Abschlussbericht

**Durchgeführt am 2026-08-24** nach `F:\CLAUDE-SETUP-KUR.md`, unter Claude Code **2.1.221**.
Freigabe erteilt für: alles außer der optionalen Backend-Rule · globale Konfiguration mit Diff pro
Datei · neue Hooks hier bauen mit Rückspiel-Vermerk · Playwright-Permissions bleiben stehen.

Befund und Begründungsketten: `~/.claude/plans/lies-in-laufwerk-f-jaunty-dahl.md`
Wortlaut alles Gestrichenen: `docs/claude-config-archiv-2026-08.md`
Vollbackup: `~/claude-config-backup/20260824-161433/` (416 Dateien, 4,1 MB)

---

## Vorher / Nachher

Gemessen wurde, was bei **jeder** Session in den Kontext geladen wird.

| Posten | Zeichen vorher | nachher | Zeilen vorher | nachher |
|---|---:|---:|---:|---:|
| MEMORY.md (Auto-Memory-Index) | 18.973 | **10.345** | 132 | **105** |
| Skill- und Command-Beschreibungen (39) | 11.081 | **7.686** | — | — |
| CLAUDE.md (Projekt) | 8.615 | **7.052** | 129 | **110** |
| CLAUDE.md (global) | 3.167 | **1.193** | 61 | **30** |
| **Summe** | **41.836** | **26.276** | | |

**−15.560 Zeichen, −37 %.** Geschätzt ≈ 10.700 → ≈ 6.700 Tokens je Session.

Weitere Zahlen: 3 → 1 MCP-Server · 6 tote `mcp__knowledge__*`-Permissions entfernt ·
26 → 7 lokale Permissions · 272 KB toter Code gelöscht · 1 gebrochener Skill repariert ·
1 wirkungsloser Hook repariert · 1 Enforcement-Lücke geschlossen.

---

## Was gestrichen wurde

Alles im Wortlaut in `docs/claude-config-archiv-2026-08.md`, Abschnittsnummern in Klammern.

| Was | Wo | Grund |
|---|---|---|
| Model-Routing, Effort, Prompt-Caching, Sub-Agents, Sprachregel | `~/.claude/CLAUDE.md` (1.1–1.5) | steht bereits in `settings.json`; Model-Routing ist zudem eine Anweisung an den Nutzer, nicht ans Modell |
| Feature-Liste „Stand 2026-04-18" | `~/.claude/CLAUDE.md` (1.6) | handgepflegt, vier Monate alt, teils falsch (`/ultrareview` ist inzwischen ein Alias) |
| `claude update`-Hook | `~/.claude/settings.json` (2.3) | 178 Sessionstarts ohne Wirkung — `autoUpdates: false` schlägt ihn |
| `frontend-design`-Plugin | `~/.claude/settings.json` (2.2) | dupliziert den Projekt-Skill, `usageCount: 0` seit Startup 83 |
| Git-Regeln | `KMU Hub/CLAUDE.md` (3.1) | dreifach vorhanden, jetzt per Hook erzwungen |
| Knowledge-Vault-Tabelle (15 Zeilen) | `KMU Hub/CLAUDE.md` (3.2) | doppelt zu `.knowledge/_index.md` |
| Intel-System-Abschnitt | `KMU Hub/CLAUDE.md` (3.3) | `~/Documents/zentria-intel` existiert auf dieser Maschine nicht |
| `knowledge`- und `github`-MCP + 6 Permissions | `.mcp.json` (4.1, 4.2) | Dateiserver auf ein Verzeichnis, das Read/Grep erreichen; GitHub-Server ohne Token |
| 19 Einmal-Permissions | `.claude/settings.local.json` (5.1) | Reste einmaliger Aktionen, darunter `PowerShell(robocopy *)` |
| 3 `gsd-*`-Regeln + 272 KB `get-shit-done` | `.gitignore` (6.1), Platte | Pfade existieren nicht, Werkzeug zuletzt im Februar benutzt |
| „User Preferences"-Block, Sprint-Langfassung | MEMORY.md (7.1, 7.2) | Doppelung zur CLAUDE.md bzw. Zustand mit Datum |

**Beleg geführt:** ein Skript hat jede nicht-leere Zeile der drei Original-Instruktionsdateien
(244 Zeilen) gegen Ergebnis **und** Archiv geprüft. **0 Zeilen fehlen.** Der erste Lauf fand 77
fehlende — das waren 7 umformulierte Zeilen und der alte MEMORY-Index, die daraufhin im Wortlaut
nachgetragen wurden (Archiv Abschnitt 8 und `memory/archive/MEMORY-index-vor-kur-2026-08-24.md`).

---

## Was umgezogen ist — und in welches Fach

| Von | Nach | Fach der Entscheidungsmatrix |
|---|---|---|
| „max 3 Subagents gleichzeitig" (Prosa) | `env.CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS: "3"` | Konfiguration |
| „keine AI-Attribution" (dreimal Prosa) | `.claude/hooks/check-no-attribution.sh` | **Hook** |
| Model-Routing, Effort, Caching | `~/.claude/docs/model-routing.md` | Nachschlagewerk |
| Zustandszeilen (Version, Zählungen, Daten) | gestrichen, Quelle verlinkt | — |
| `session-fetch.sh` (lokal, ungetrackt) | `.claude/hooks/session-fetch.sh`, versioniert | Hook |
| Sprint-Historie #38–#40 | `memory/archive/editor_sessions_37_40.md` | Archiv |

---

## Was neu ist

- **`.claude/hooks/check-no-attribution.sh`** — blockt `Co-Authored-By` mit KI-Namen,
  „Generated with", Session-Links und das Robot-Emoji. Menschliche Co-Autoren gehen durch.
  13 Testfälle, alle grün.
- **`.claude/hooks/tests/`** — zwei Testskripte, die die Hooks scharf auslösen. Sie liegen
  bewusst als Dateien vor: stünden die Muster im Bash-Kommando, blockte der Hook den Testaufruf
  selbst (was er beim ersten Versuch prompt tat).
- **`.claude/agents/browser-operator.md`** — der Agent, den `browser-task` seit dem 20.08.
  referenzierte, ohne dass es ihn gab.
- **`docs/claude-config-archiv-2026-08.md`** — 424 Zeilen, jede Streichung rückholbar.

---

## Zwei Befunde, die erst beim Auslösen auftauchten

Beide bestätigen die Kur-Regel „ein Hook, den du nicht ausgelöst gesehen hast, ist nicht
verifiziert". Beide waren im Befund von Phase 3 **nicht** enthalten — sie kamen erst in Phase 5.

### `check-commit-message.sh` hat seit Juni nie geblockt

Der PreToolUse-Input ist JSON, das Kommando kommt also als `-m \"feat: x\"` an. Der Hook suchte
per Lookbehind nach `-m "` — kein Match, leere Message, und die eingebaute Regel „leere Message =
Heredoc = durchlassen" winkte **jede** doppelt gequotete Commit-Message durch. Single Quotes waren
nie betroffen, weil JSON die nicht escaped; deshalb fiel es zwei Monate nicht auf.

Nachgewiesen durch scharfes Auslösen: `git commit -m "kaputte message ohne praefix"` ging durch.
Nach dem Fix (Unescaping vor der Extraktion) blockt derselbe Aufruf. 12 Testfälle pinnen jetzt
beide Quote-Formen.

**Bekannte Restlücke:** Heredoc-Commits (`git commit -F -`) prüft der Hook weiterhin nicht — das
ist im Skript dokumentiert und bewusst so, weil eine Heredoc-Extraktion mehr Falsch-Positive
erzeugt als sie verhindert. Der neue Attribution-Hook hat diese Lücke **nicht**: er prüft das
ganze Kommando und fängt Heredocs mit.

### `format-on-write.sh` läuft, formatiert aber nichts

Der Hook feuert korrekt und macht dann still nichts, weil **prettier im Projekt nicht installiert
ist** (weder in `desktop/node_modules` noch im Root). Das ist exakt das dokumentierte Verhalten —
„kein prettier im Baum → still no-op" —, aber es heißt: das Formatter-Gate aus dem Kur-Zielbild
existiert hier nur auf dem Papier.

**Nicht behoben**, weil eine npm-Installation über den freigegebenen Umfang hinausgeht.
Ein `npm i -D prettier` in `desktop/` würde den Hook scharf machen — deine Entscheidung.

---

## Was bewusst geblieben ist

Der wertvollste Abschnitt, nach Kur-Regel 2: *ein Unterschied ist keine Richtungsangabe.*

| Was | Warum |
|---|---|
| **`check-no-secrets.sh`** | Der Fallstrick, vor dem die Kur warnt („Ausdruck zu gierig, zieht die Commit-Message mit hinein"), ist im Skript kommentiert und gelöst. Beim Test hat er sogar zu scharf gegriffen — er blockte einen Testaufruf, weil im Fließtext davor `.env` stand. Lieber so herum. |
| **`.gitattributes`** | `*.sh text eol=lf` + `.claude/**/*.md text eol=lf`. Der Windows-CRLF-Fallstrick der Kur war hier längst gelöst; alle sieben Hooks sind verifiziert LF. |
| **Die 22 Design-Skills** | **Keine** Doppelung von `impeccable` — der globale Skill hat nur `SKILL.md`, `reference/` und `scripts/` und orchestriert diese hier. Getrennt geprüft, nicht angenommen. Gekürzt wurden nur Beschreibungen, kein Skill entfernt, kein Rumpf angefasst. |
| **10 `mcp__playwright__*`-Permissions** | Auf deinen Wunsch. Zur Klarstellung: sie kosten keinen Kontext und schaden nicht, greifen aber erst, wenn ein `playwright`-MCP-Server konfiguriert ist — heute ist in keinem Scope einer eingetragen. Deine QA läuft über `desktop/scripts/qa-*.mjs` mit `node`, dafür braucht es sie nicht. |
| **Die 6 `intel-*`-Commands** | Steuern ein Repo, das hier nicht liegt — aber sie sind eingecheckt und gelten fürs Team. Statt zu löschen (das träfe Luke) nur die Beschreibungen gekürzt. Klonst du `zentria-intel`, greifen sie sofort wieder. |
| **UI/UX-Block in der CLAUDE.md** | Nicht in eine `.claude/rules/`-Datei mit `paths:` ausgelagert, obwohl die Kur das für „Alles-in-einer-CLAUDE.md" vorschlägt. Rules laden erst, wenn eine passende Datei angefasst wird — eine Font-Ban-Regel, die im Design-Gespräch ohne Dateizugriff fehlt, ist genau dann weg, wenn sie gebraucht wird. |
| **Architektur-Regeln 1–11, Deployment-Reihenfolge, Build-+-Verify-Standard** | Kur-Regel 4: Fachwissen ist heilig. Nur Datum und Zahl gingen, die Regeln stehen wörtlich. |
| **`context7`-MCP** | Doku-Lookup für Fremdbibliotheken, dupliziert keine eingebaute Fähigkeit. |
| **`docs/LEARNINGS.md`** | Erfüllt Kur-Zielbild 5 im Kern (Problem → Konsequenz → Learning). Kein zweites System daneben gebaut. |

---

## Widersprüche zur Kur-Vorlage

Damit der nächste, der die Kur laufen lässt, weiß, wo nachzuziehen ist.

**1. Diese Maschine ist älter als die Vorlage.** Sie wurde unter 2.1.241 geschrieben, hier läuft
**2.1.221** (Binary vom 04.08.2026, npm-global). Zwei Einträge ihres Altlasten-Katalogs gelten
hier deshalb **nicht**: `/ultraplan` („entfernt mit 2.1.222") ist vorhanden, und `/review`
(„nur noch Alias, seit 2.1.223") ist ein eigenständiger Befehl — die Skill-Liste dieser Session
zeigt beide. Der lokale Changelog-Cache endet passend bei 2.1.221.

**2. Verifiziert und zutreffend** (gegen `~/.claude/cache/changelog.md`):
`CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS` und `..._PER_SESSION` existieren ·
`.claude/rules/` mit `paths:`-Frontmatter existiert · der `InstructionsLoaded`-Hook existiert ·
„Claude ruft `/verify` und `/code-review` nicht mehr selbst auf" steht so im Changelog ·
`context: fork` läuft inzwischen standardmäßig im Hintergrund (`background: false` zum Abwählen).

**3. `extraKnownMarketplaces` → `additionalMarketplaces`** ist im Changelog bis 2.1.221 nicht
auffindbar. Für dieses Setup gegenstandslos — es ist kein Marketplace-Schlüssel gesetzt.

**4. Die Vorlage unterschätzt einen Posten.** Ihre Ist-Tabelle kennt CLAUDE.md, Rules, Skills,
MCP — aber nicht die **Skill-Beschreibungen**, die bei jeder Session im Systemprompt landen.
Hier waren das 11.081 Zeichen, mehr als die Projekt-CLAUDE.md, und der zweitgrößte Posten
überhaupt. Wer nur Dateien misst, übersieht ihn. **Empfehlung für die nächste Fassung der
Vorlage:** eine eigene Zeile in der Ist-Tabelle für die Summe aller `description:`-Frontmatter.

**5. Das `claude-command-center` existiert auf dieser Maschine nicht.** Drei Hooks tragen den Kopf
„KANONISCH (Source of Truth: claude-command-center) … via `sync.ps1` propagieren"; weder das Repo
noch `sync.ps1` liegen auf C:, D: oder E:. Die neuen und geänderten Hooks tragen einen
Rückspiel-Vermerk im Kopf — **das muss jemand tatsächlich tun**, sonst überschreibt der nächste
Sync sie oder sie fehlen in den anderen Projekten.

---

## Was offen bleibt

1. **`/context`, `/doctor` und `/mcp` konnte ich nicht selbst aufrufen.** Die Zahlen oben sind
   aus den Dateien gemessen, nicht aus `/context` gelesen. Tipp sie einmal — dann steht hier
   Gemessenes statt Gerechnetes, und `/mcp` bestätigt, dass `context7` als einziger Server
   sauber verbunden ist.
2. **Claude Code ist ≥20 Versionen alt.** Update von Hand: `npm i -g @anthropic-ai/claude-code`.
   Bewusst nicht während der Kur gemacht — ein Update kann Verhalten ändern, und dann wäre
   unklar, was von der Kur kam und was von der neuen Version.
3. **431 Commits Rückstand.** Die vier Kur-Commits liegen **lokal**. Ein `git push` wird
   abgelehnt, solange nicht gepullt ist. Ein Pull über 431 Commits mit vier eigenen darauf ist
   eine Entscheidung, die dir gehört (Merge oder Rebase) — deshalb habe ich sie nicht getroffen.
   Die Konfiguration selbst kollidiert nicht: `git diff HEAD origin/main -- CLAUDE.md .claude/`
   war vor dem Umbau leer.
4. **prettier fehlt** — siehe oben, das Formatter-Gate ist ohne es wirkungslos.
5. **`teach-impeccable`** ist ein Einmal-Setup-Skill, und die `DESIGN.md`, die er erzeugen soll,
   existiert nicht. Entweder einmal laufen lassen oder aus dem Auswahlraum nehmen.

---

## Wie du es zurückdrehst

- **Einzelne Regel:** Block in `docs/claude-config-archiv-2026-08.md` suchen, an die dort
  genannte Fundstelle zurückkopieren.
- **Eine ganze Datei:** `~/claude-config-backup/20260824-161433/` enthält den kompletten Stand
  von vorher, inklusive der ungetrackten (`settings.local.json`, MEMORY.md).
- **Ein ganzer Schritt:** die vier Commits sind getrennt rollbar —
  `git revert 0dbff8e4` (Skill-Beschreibungen) · `55a5f7b9` (Hooks und Agent) ·
  `09795a25` (Streichungen in CLAUDE.md, MCP, gitignore) · `465cc007` (Archiv).
