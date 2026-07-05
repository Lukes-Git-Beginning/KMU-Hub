# AUTONOMOUS-RUN — Protokoll für „mach autonom weiter"

> **Trigger (Darien):** In einem frischen Terminal „**mach autonom weiter**" (oder sinngemäß „arbeite autonom weiter am Plan").
> **Ziel:** Der Plan wird da weitergearbeitet, wo wir gerade stehen — auf **solider recherchierter Grundlage**, mit **einer** gebündelten Fragerunde, dann durchgearbeitet bis **~45 % Context-Fenster** pro Block.
> Spezifiziert von Darien 2026-07-05. Ergänzt/verfeinert bei jedem Lauf.

---

## ⚠ Grundregel: ALLE etablierten Konventionen gelten weiter (Darien 2026-07-05)
Autonom heißt **nicht** „lockerer". Alles, was wir fürs Arbeiten festgelegt haben, gilt unverändert — bindend sind `~/.claude/CLAUDE.md`, `<projekt>/CLAUDE.md`, und **jede** Zeile in `MEMORY.md` (Auto-Memory-Index) + die verlinkten `memory/*.md`. Vor dem Bauen den relevanten Teil davon präsent haben. Besonders die operativ-kritischen (nicht abschließend):
- **Umsetzung:** Implementation-Loop bei Screenshot-Feedback (`[[feedback_implementation_loop]]`) · Subagent-Pre-Brief + Mandatory-End-Verify (`[[feedback_subagent_verification]]`) · QA über mehrere Datensätze/Zustände/Breiten (`[[feedback_qa_thoroughness]]`).
- **UX-Muster (modulübergreifend):** Detail = zentriertes `shared/DetailModal`, ganze Zeile klickbar (`[[feedback_detail_modal_standard]]`) · Sticky Back/Close (`[[feedback_sticky_back_buttons]]`) · Sortierung Feld+Richtung `shared/SortMenu`, Zurück-Button überall (`[[feedback_recurring_ux_patterns]]`) · Skeleton-Loading + Abkürzungs-Tooltips (`[[feedback_projektweite_ux_konventionen]]`).
- **Tiefe & Vollständigkeit:** echte Detail-Ansichten + wirksame Exports (kein Toast-Stub) + Placeholder→MSW (`[[feedback_module_depth_standard]]`) · Moduleinstellungen pro Modul (`personal`+`tenant`, `ModuleSettingsShell`) (`[[feedback_module_settings_per_module]]`) · rollierend je Modul review-reif (`[[feedback_rolling_module_completion]]`).
- **Design:** Cosmi-Identität eigen, Apple/Discord nur als Linse, **keine Emojis in UI**, Font-Bans (kein Inter/Roboto/…), Motion nur transform/opacity (`[[feedback_design_philosophy]]`, CLAUDE.md UI/UX).
- **i18n:** ×4 Sprachen, `{var}` nicht `{{var}}`, ICU-Plural (`[[reference_i18next_icu_braces]]`).
- **Git/CI:** Conventional Commits, English, imperativ, **keine AI-Attribution**; lint vor Push (`[[feedback_lint_before_push]]`); keine gestapelten Hintergrund-tsc (`[[feedback_no_piling_background_tsc]]`); `mocks/data/` braucht `git add -f` (`[[reference_gitignore_data_dir]]`).
- **Kommunikation:** Deutsch, scanbar, direkt+begründet, minimal-Diffs bei Code-Erklärung (`[[feedback_communication]]`, `[[feedback_code_explanation]]`, `[[feedback_workstyle]]`).

## Ablauf (jeder autonome Block)

### 0 · Pull + orientieren (immer zuerst)
- `git fetch --all` + `git pull --ff-only` (Luke pusht in dichten Wellen direct-to-main → sonst Divergenz).
- `.planning/RESUME-NEXT.md` **Top-Block** lesen (= Wiedereinstiegspunkt, Stand + nächste Unit).
- Lukes neue Commits sichten (`git log <alt>..origin/main`) — was wurde verdrahtet/gefixt, was ist FE-relevant.

### 1 · Recherche — IMMER, beide Achsen (nicht überspringen)
Das ist Darien-kritisch: **ohne solide Grundlage nicht bauen.**
- **a) Cosmi-Ist:** Aktueller Zustand des nächsten Blocks im eigenen Programm — was ist gebaut, was ist dünn/Mock, welches Backend existiert (Echt-Schaltung möglich?), welche `shared/`-Muster gibt es schon. (Explore-Subagent, wenn breit.)
- **b) Marktrecherche zu den Funktionen:** Web-Recherche zu den Features des Blocks — wie lösen es führende Tools (Notion/Linear/Intercom/HubSpot/Pipedrive/Personio etc.), was ist Standard, was ist Best Practice, was erwarten DACH-KMU. (WebSearch/general-purpose-Subagent.) Ergebnis fließt in Konzept + Fragen ein.
- Subagenten parallel (max 3, Sonnet-Baseline) → Hauptcontext bleibt schlank, ich behalte nur die Schlüsse.

### 2 · Fragen bündeln → EINE Abklärungsrunde
- **Alle** offenen Entscheidungen des Blocks sammeln, nicht kleinteilig nachfragen.
- Die Runde muss den **ganzen Block** abdecken, sodass danach ohne weitere Rückfragen durchgearbeitet werden kann.
- `AskUserQuestion` (2–4 Fragen, Empfehlung je zuerst + „(Empfohlen)"). Wenn nichts offen ist: kurz Plan nennen und loslegen.
- Erst nach Darien-OK bauen.

### 3 · Durcharbeiten bis ~45 % Context
- **Build-+-Verify-Standard pro Phase** (`.planning/nico-block/WORKFLOW.md`): bauen → i18n ×4 (`{var}`, ICU-Plural) → Demo-Handler nur wo Backend fehlt → **gescopter Typecheck** (nur geänderte Dateien) → **Playwright-Screenshot-QA + Bilder wirklich ansehen** → iterieren bis grün.
- **Pro verifiziertem Modul:** eslint (geänderte Dateien) + scoped tsc + QA grün → **ein Commit + Push auf main**. ⚠ Push = **Auto-Deploy auf Hetzner (live)** → vorher hart verifizieren, CI-grün abwarten/prüfen.
- **Echt verbinden statt mocken:** Endpoint existiert → direkt ans echte Backend (🔌). Fehlt → mock-first + Notiz in `.planning/backend-gaps.md` (nicht auf Luke warten).
- Fortschritt über `TaskCreate`/`TaskUpdate` sichtbar halten (Darien mag Status-Anzeige).
- Vor jedem Push nochmal `git pull --rebase` (Luke-Wellen).

### 4 · Abschluss
- `.planning/RESUME-NEXT.md` neuen Top-Block schreiben (Stand, was gebaut, nächste Unit).
- Relevante Planungs-Docs pflegen (`MASTER-PLAN.md` §6, block-spezifische Pläne, `backend-gaps.md`).
- Knapper, scanbarer Bericht an Darien (was gemacht, was gefunden, was für Luke offen, nächste Unit).

## Guardrails / Nicht vergessen
- **Push-Mode:** pro verifiziertem Modul auf main = Auto-Deploy live (Darien 2026-07-05). CI-grün ist Pflicht.
- **Docker:** postgres = Custom-Image (pg_cron) → **bauen**; `--no-deps --no-build` beim Bringup (OOM); nach Luke-Pull crm/gateway/biz/migrate/postgres oft stale → neu bauen. Login `demo@local.test`/`Demo1234!`. Details `[[reference_local_backend_bringup]]` (Memory).
- **Luke-gebunden NICHT bearbeiten** (security-Echt-Schaltung, mails-IMAP, admin-Backend) → in `backend-gaps.md` notieren, self-doable weitermachen.
- **Keine AI-Attribution** in Commits. Conventional Commits, English, imperativ.
- **Dev-Server killen** (Windows): PowerShell `Get-NetTCPConnection -LocalPort 5173 | Stop-Process` + `Get-Process electron | Stop-Process`; nicht `pkill -f vite`.
- **Kein full-tsc als Gate** (~30 Min, Baseline nicht grün) — scoped tsc über geänderte Dateien; scoped-Config-Fehler in Fremddateien = Noise (CI-Desktop full-tsc ist der echte Gate, auf main grün).

## Referenz-Verwandtschaft (Memory)
Dieser Lauf formalisiert: `[[feedback_market_driven_workflow]]` (Recherche-Achsen), `[[feedback_ten_phase_autonomous_batch]]` (Batch-Kadenz), `[[feedback_hetzner_review_workflow]]` (Push=Deploy), `[[feedback_qa_thoroughness]]` (QA-Standard).
