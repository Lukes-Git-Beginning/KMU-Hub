# Prompt: Doku-, Knowledge-, Memory- & Planungs-Konsolidierung

> Start-Prompt für eine fokussierte Aufräum-Session. Ziel: die Doku-/Wissens-Schicht des Projekts
> nach ~5 Wochen aktiver Entwicklung gegen den Live-Stand auffrischen, Drift beseitigen, Redundanz/Altlast
> entfernen und das Repo auf „Public-Projekt-Qualität" bringen. Diese Prompt ist als Ein-Datei-Einstieg
> gedacht — bei Bedarf `ultracode` voranstellen, um die Audit-Phase als Multi-Agent-Workflow zu fahren.

## Rolle & Ziel

Du bist Doku-/Repo-Maintainer für das KMU-Hub-/Cosmi-CRM (Software „Cosmi", Firma „Zentria"). Bring die
**nicht-Code-Artefakte** in Einklang mit dem tatsächlichen Repo- und Production-Stand: `README.md`,
`docs/`, `.knowledge/`-Vault, der Auto-Memory unter
`~/.claude/projects/C--Users-Luke-Documents-KMU-Hub/memory/` (inkl. `MEMORY.md`) und die Planungs-Ablage
`.planning/`. Entferne Veraltetes, konsolidiere Redundantes, schließe Lücken — **ohne** Funktionalität oder
Code-Verhalten zu ändern (reine Doku-/Wissens-Arbeit).

## Leitprinzip (kritisch)

**Live messen, nicht Doc-zu-Doc kopieren.** Jede Kennzahl, die du auffrischst, MUSS gegen das echte Repo
bzw. die Production gemessen sein — sonst frischt man veraltete Zahlen mit anderen veralteten Zahlen auf.
Unterscheide dabei strikt:
- **Repo-Head** (lokaler/Git-Stand der Migrations) vs. **Production-Migrationskopf** (per SSH/psql gemessen) —
  das sind zwei verschiedene Zahlen, und README/Docs meinen oft den Production-Stand.
- **Geplant** (Scope-Matrix-„~RPCs") vs. **implementiert** (Code).
Markiere, was du nicht belegen kannst, explizit als „(geschätzt)" statt zu raten.

## Bekannte Befunde als Startpunkt (verifizieren, dann fixen — nicht blind übernehmen)

Aus dem letzten Status-Snapshot (`.planning/status-overview.md`, Stand 2026-06-18) sind folgende Drift-Stellen
bekannt. Jede zuerst live gegenprüfen, dann in ALLEN betroffenen Dateien konsistent ziehen:

1. **Migrationskopf:** Live-Repo `000213`; Docs nennen 115 (README) / 116 (Matrix) / 131 (milestones) /
   133 (ADR-0007) — alle veraltet. Production-Head separat per psql messen.
2. **Service-Zahl:** `backend/cmd/*` = 24 (23 µSvc + gateway); README sagt „25 App-Services",
   CLAUDE.md „24 gRPC-Microservices". Begriff (Compose-Services vs. cmd-Dirs) sauber definieren.
3. **Feature-Flags:** Registry `backend/internal/featureflag/registry.go` = 17 (14× `modules.*` + `plugins.wasm`
   + `plugins.config` + `plugins.api`); Docs sagen „16" — `plugins.api` fehlt überall.
4. **R2-P1.12 (Finance-Normalisierung):** ADR-0007 sagt erledigt (Migr. 000132/133), `docs/ROADMAP.md`
   P1-Tabelle sagt „Pending" — Widerspruch auflösen.
5. **`.knowledge/_index.md`** (Stand 2026-05-10) ist ggü. `milestones.md` (2026-06-12) ~5 Wochen veraltet;
   Sprint-4-Ereignisse (alle P0 dicht, `COSMI_ENV=production` scharf, Finance-Normalisierung, RLS produktiv)
   fehlen. Migrations-Zahl „116" in Vault-Tabellen veraltet.
6. **ADR-0007** liegt in `docs/adr/0007-...` statt in `docs/ARCHITECTURE.md` (nur ADR-001…006) — Querverweis
   ergänzen, damit die ADR-Liste vollständig auffindbar ist.
7. **CLAUDE.md-Drift:** Architektur-Regel 11 sagt „aktuell Single-Tenant", obwohl RLS live + `COSMI_ENV=production`
   scharf ist; Feature-Flag-Zähler „16"; Knowledge-Vault-Tabelle „116 Migrations".
8. **MEMORY.md** ist ~47 KB / 233 Zeilen mit Größen-Warnung — überlange Index-Einträge in Topic-Files
   auslagern, Pointer auf ≤200 Zeichen kürzen.

## Arbeitsphasen

### Phase 0 — Messen & Audit (read-only)
Sammle den Ground-Truth, bevor du irgendetwas änderst:
- `git log --oneline -25`, aktueller Branch/HEAD.
- Höchste Migration + Gesamtzahl: `ls backend/migrations/*.up.sql | sort | tail -1` und `| wc -l`.
- Service-Zahl + Namen: `ls -d backend/cmd/*/`.
- Feature-Flags: Keys + Defaults aus `backend/internal/featureflag/registry.go`.
- Coverage-Gate: aus `.github/workflows/ci.yml`. CI-Workflow-Inventar aus `.github/workflows/`.
- **Production-Stand** (read-only): per `ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195` und
  `psql -U kmuhub -d kmuhub -c "SELECT max(version) FROM schema_migrations;"` den Prod-Migrationskopf,
  optional Container-Health. (Nur lesen, nichts deployen.)
- `.knowledge/`-Inventar: alle Notes + ihr `updated:`-Frontmatter.
- Memory-Inventar: alle Dateien im memory-Ordner + jeden `MEMORY.md`-Pointer gegen die Dateiliste abgleichen.
- `.planning/`-Inventar: alle Dateien/Ordner, jeweils mit grober Einordnung „aktiv vs. abgeschlossen".

### Phase 1 — Befund-Report + Entscheidungs-Gate
Lege einen kurzen Befund-Report vor (Tabelle: Datei × veralteter Wert × Live-Wert × Aktion). Für alles, was
**gelöscht, archiviert oder strukturell umgebaut** wird (`.planning`-Files, Memory-Konsolidierung, README-Umbau),
**erst Rückfrage an den User** — keine destruktiven Aktionen ohne Freigabe. Konsensliste vor Phase 2 fixieren.

### Phase 2 — Auffrischen & Konsolidieren (nach Freigabe)

**A · Kennzahlen-Konsistenz-Sweep.** Ziehe Migrationskopf (Repo + Prod getrennt), Service-Zahl, Flag-Zahl (17),
Coverage-Gate, Sprint-Status (heute relativ zu den Sprint-Daten), kombinierte Launch-Note konsistent durch
README, `docs/ROADMAP.md`, `docs/MODULES_SCOPE_MATRIX.md`, `CLAUDE.md`, `.knowledge/*`. **SSOT-Disziplin:**
lege je Kennzahl eine maßgebliche Quelle fest (z. B. ROADMAP = Sprint/Blocker, Registry = Flags) und lasse die
übrigen Stellen darauf verweisen statt die Zahl zu duplizieren.

**B · README (Public-Qualität).** Veraltete Production-Claims korrigieren (Service-Zahl, Migrationskopf,
Sprint-Stand), Setup-Schritte gegen die echten Kommandos/`Makefile`/Compose verifizieren, Tech-Stack-Tabelle
aktualisieren. Prüfen/ergänzen, was ein öffentliches Repo braucht: knappe Projektbeschreibung, Status-Badge(s),
Architektur-Kurzüberblick (ggf. Link auf `.planning/status-overview.md`/PDF), Lizenz-Hinweis, evtl.
`CONTRIBUTING`/Conventional-Commits-Hinweis, korrekte Repo-/Branch-Angaben. Keine Behauptungen, die der Code
nicht hält (z. B. WASM-Plugins als „verfügbar" — die sind OFF).

**C · Knowledge-Vault (`.knowledge/`).** `_index.md` auf den realen Stand heben (Sprint-4-Ereignisse,
Migrations-Zahl, Status-Block). Alle Notes mit verändertem Inhalt im `updated:`-Frontmatter neu datieren.
**Dangling-Link-Check:** jeden `[[wikilink]]` gegen existierende Notes prüfen; tote Links fixen oder die
fehlende Note anlegen. Bereits modifizierte `troubleshooting.md` (uncommittet) reviewen und finalisieren.
Prüfen, ob `update-knowledge`-Skill genutzt werden sollte.

**D · Memory.** Veraltete Fakten korrigieren (jede Memory, die eine Datei/Funktion/Flag nennt, gegen den
Code verifizieren). **MEMORY.md-Diät:** überlange Zeilen kürzen, Detail in die jeweiligen Topic-Files
verschieben, Pointer auf eine Zeile/≤200 Zeichen. **Dangling-Pointer:** jeden `MEMORY.md`-Link gegen die
memory-Dateiliste prüfen. Abgeschlossene `project_*`-Einträge (fertige Sprints/Wellen) konsolidieren —
mehrere Wellen-Memos eines Sprints zu einem Closure-Memo zusammenfassen, Redundanz raus.

**E · `.planning/`-Triage.** Pro Datei entscheiden „behalten/archivieren/löschen" nach Kriterium:
*abgeschlossen UND in Memory/Knowledge dokumentiert → archivierbar; aktiv/laufend → behalten*. Abgeschlossene
Wellen-/Marathon-Plan-Files in einen `.planning/archiv/`-Ordner verschieben (oder löschen nach Freigabe).
Eine kurze `.planning/README.md` (oder `INDEX.md`) anlegen, die sagt, was aktuell aktiv ist und was Archiv ist.
Build-Artefakte (`.planning/pdf-build/` mit 3,2 MB `mermaid.min.js` + PNGs) bewerten — siehe G.

**F · CLAUDE.md.** Regel 11 (Tenant-Modell) auf den realen RLS-/`COSMI_ENV=production`-Stand aktualisieren,
Feature-Flag-Zähler (17), Knowledge-Vault-Tabelle (Migrations-Zahl), Projekt-Kontext-Block auf heutigen Sprint
heben. Konsistenz mit dem aufgefrischten `.knowledge/`-Vault sicherstellen (CLAUDE.md ist nur Pointer-Layer).

**G · Repo-Hygiene.** Untracked-Artefakte triagieren: `desktop/scripts/qa-welle*.mjs`,
`desktop/scripts/build-status-pdf.mjs`, `shot-status.mjs`, `desktop/scripts/screenshots/`,
`.planning/pdf-build/` — je nach Wert committen, in `.gitignore` aufnehmen (Build-Output/Screenshots) oder
löschen. `.gitignore` so ergänzen, dass keine schweren Build-Artefakte (z. B. das 3,2-MB-Mermaid-Bundle, PNGs)
versehentlich eingecheckt werden.

### Phase 3 — Verifikation
- Jede aufgefrischte Zahl steht in jeder Datei identisch und ist gegen Phase-0-Messung belegt.
- Keine toten `[[wikilinks]]` im Vault, keine toten Pointer in `MEMORY.md`.
- README-Setup-Schritte stimmen mit den realen Kommandos überein.
- Markdown rendert sauber (Tabellen/Links). Optional: Status-Snapshot bei Bedarf neu generieren
  (`node desktop/scripts/build-status-pdf.mjs`).

### Phase 4 — Commit & Push
Ein sauberer Direct-to-Main-Commit (Default-Branch-Strategie), Conventional Commits, englische Imperativ-
Messages, **keine AI-Attribution**. Memory-Dateien liegen außerhalb des Repos und werden nicht committet.
Push am Ende der Session.

## Leitplanken
- Nur Doku/Wissen/Planung anfassen — **kein** Code-/Migrations-/Config-Verhalten ändern.
- Production nur lesend berühren (psql SELECT, Health) — **kein** Deploy.
- Vor Löschen/Archivieren/README-Umbau: User-Gate (Phase 1).
- Zahlen belegen, nicht erfinden; Unbelegbares „(geschätzt)" markieren.
- Datumsangaben absolut (nicht „letzte Woche").

## Output / Done-Kriterien
1. Befund-Report (Phase 1) mit Vorher/Nachher-Werten.
2. Aufgefrischte Dateien: README, `docs/*` (soweit driftet), `.knowledge/*`, `CLAUDE.md`, Memory-Ordner.
3. Konsolidierte `.planning/` + Index, triagierte Untracked-Artefakte, ergänzte `.gitignore`.
4. Ein Commit + Push; kurze Zusammenfassung „was aufgefrischt, was archiviert/gelöscht, was offen blieb".
