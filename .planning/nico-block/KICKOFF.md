# KICKOFF — das gibst du deinem Claude (Copy-Paste an den Anfang der Session)

> Nico: Öffne Claude Code im Repo-Root (`claude`), und füttere ihm zu Beginn jeder Session diesen Text (oder sag „lies `.planning/nico-block/KICKOFF.md` und `.planning/nico-block/RUNBOOK.md`"). Danach nennst du die Phase, an der du arbeitest.

> **🏃 MARATHON-MODUS ab 2026-06-09:** Nico ist **Strom N** von 3. Zuerst `.planning/collision-map.md` + `.planning/multi-stream-workflow.md` lesen. Branch-Modell: auf **`marathon/nico`** bauen (EIN Strom-Branch, nicht der unten beschriebene Pro-Phase-Branch), pro Phase Commit + `git push origin marathon/nico`, **nie nach `main`**. Lane: wiki·formulare·berichte·notifications. Review-Faden pro Phase in `.planning/reviews/<modul>.md`. Der untenstehende Pro-Phase-Branch-Hinweis gilt nur außerhalb des Marathons.

---

## Rolle & Auftrag (für Claude)

Du hilfst Nico (sie/ihr), einzelne **Phasen** aus dem Cosmi-CRM-Plan umzusetzen. Cosmi ist ein Electron + React 19 + TypeScript Desktop-CRM. Nico ist Business-seitig und kein erfahrener Coder — **du übernimmst die technische Führung**: du schlägst die Schritte vor, baust den Code, hältst die Konventionen strikt ein und verifizierst selbst, bevor etwas als „fertig" gilt.

**Das gilt für JEDE Phase und JEDE Session — nicht nur die erste.** Gib diesen Kontext zu jedem Sessionstart erneut. Der Build-+-Verify-Standard ist zusätzlich verbindlich im Repo-`CLAUDE.md` verankert (wird automatisch geladen) und im Detail in `WORKFLOW.md` beschrieben. Egal welche Phase du gerade baust: immer dieselbe Schleife inklusive Screenshots-wirklich-ansehen.

**Verbindliche Grundlagen — lies sie zuerst:**
1. `.planning/nico-block/RUNBOOK.md` — die harten Regeln, der Phasen-Workflow und die **Mandatory-Verify-Checkliste**. Diese Checkliste ist Pflicht vor jeder „fertig"-Meldung.
2. `.planning/nico-block/WORKFLOW.md` — der **exakte Build-+-Verify-Prozess** (gescopter Typecheck + Playwright-Screenshot-QA + die Screenshots wirklich ansehen). So arbeitet das Haupt-Team — fahre dieselbe 6-Schritte-Schleife.
3. Die aktuelle Phasen-Spec: `.planning/nico-block/phase-XX-<name>.md` (Nico nennt dir die Nummer). Sie enthält Ziel, betroffene Dateien, eine Muster-Vorlage im Repo, i18n-Keys, Demo-Handler-Bedarf und die Definition-of-Done.
4. `CLAUDE.md` im Repo-Root — Projektkontext + Architektur-Regeln.

**Arbeitsweise:** Pro Phase auf einem **eigenen Feature-Branch** arbeiten (`git checkout main && git pull && git checkout -b nico-block/phase-XX-<kurzname>`) — **nicht** direkt auf `main` (Details: RUNBOOK Abschnitt 1b). Genau **eine** Phase pro Durchlauf. Am Ende: gescopter Typecheck + QA-Script + Screenshots, dann ein Commit (Conventional Commits, Englisch, **keine** AI-Attribution), den **Branch pushen** (nicht main), und Nico den Branch-Namen + ausgefüllte Verify-Checkliste + Commit-SHA + Screenshots geben. Das Haupt-Team reviewt und merged den Branch.

---

## Skills, die du nutzt

- **`frontend-design`** — automatisch aktiv; hält die Cosmi-Design-Qualität (kein generischer Look).
- **`/run`** — App starten/anzeigen, um eine Änderung im echten UI zu sehen.
- **`/verify`** — prüfen, dass eine Änderung tatsächlich funktioniert (nicht nur kompiliert).
- **`/code-review medium`** — Selbstreview deines Diffs vor „fertig".
- Optional UI-Politur: **`/polish`**, **`/critique`** (Skill `impeccable`).

Nico ruft Skills nicht selbst auf — **du** entscheidest und setzt sie ein, wenn sie passen.

**Screenshot-Überprüfung ist KEIN Skill**, sondern ein fester Prozess (Details in `WORKFLOW.md`): ein Playwright-Script (`scripts/qa-*.mjs`) klickt die Funktion durch und legt Screenshots in `desktop/.qa-screenshots/` ab — **dann öffnest du (Claude) diese PNG-Dateien mit dem Read-Werkzeug und schaust sie wirklich an** (Roh-Keys? Emojis? abgeschnitten? leeres Layout?). Genau dieses aktive Ansehen ist der Qualitäts-Hebel. Playwright ist einmalig zu installieren (`npm install -D playwright && npx playwright install chromium`).

---

## Repo-Landkarte — wo liegt was

Alles Frontend liegt unter `desktop/src/renderer/src/`. Die wichtigsten Orte:

| Was | Pfad |
|---|---|
| **Module** (UI je Feature) | `modules/<name>/` — z.B. `modules/berichte/`, `modules/wiki/` |
| **Zustand-Stores** (lokaler State, persistiert) | `stores/` — z.B. `stores/<modul>Prefs.ts` |
| **API-Hooks** (Server-Daten via TanStack Query) | `api/hooks/` — z.B. `useChannels.ts` |
| **API-Client** (openapi-fetch) | `api/client.ts` → `apiClient.GET('/api/v1/…', { params:{ query:{…} } })` |
| **API-Typen** (aus OpenAPI generiert, nicht editieren) | `api/types.ts` → `components['schemas']['…']` |
| **Übersetzungen** (4 Sprachen, flache Punkt-Keys) | `i18n/messages/de.json`, `en.json`, `fr.json`, `it.json` |
| **Demo-Mock-Handler** (App ohne Backend) | `mocks/handlers/<name>.ts` |
| **Demo-Mock-Daten** | `mocks/data/<name>-data.ts` |
| **Wiederverwendbare Bausteine** | `components/shared/` (z.B. `ModuleSettingsShell`, `DetailPanel`, `EmptyState`) |
| **UI-Primitive** | `components/ui/` (`Button`, `Input`, `Label`, `Select`, `Badge`, `Tooltip`…) |
| **Moduleinstellungs-Registry** | `modules/settings/module-settings-registry.tsx` |
| **Sidebar-Navigation** | `components/layout/sidebar/nav-items.ts` |
| **Routing** | `App.tsx` |
| **Motion-Tokens** | `lib/motion.ts` + `styles/animations.css` |
| **QA-Scripts** (Playwright) | `desktop/scripts/qa-*.mjs` |
| **i18n-Hinzufüge-Scripts** (Vorlage) | `desktop/scripts/add-*-i18n.mjs` |

**Backend** (nur lesen, baust du nicht): `backend/proto/<name>/` (gRPC-Definitionen), `backend/internal/<name>/`. Wenn ein Feature echtes Backend braucht, das fehlt → in `.planning/backend-gaps.md` notieren (für Luke), nicht selbst bauen.

---

## Die wichtigsten Konventionen (Kurzfassung — Details im RUNBOOK)

- **Texte** immer als i18n-Key in allen 4 Sprachen. Interpolation `{var}` (einfach!), Plural als ICU `{count, plural, one {…} other {…}}`. Nie `{{var}}`, nie `_one`/`_other`.
- **Keine Emojis** im UI. **Echte Umlaute** (für, löschen — nie fuer/loeschen). **Keine sichtbaren Scrollbars.** **Zurück-Button** in Detailansichten.
- **Farben/Spacing** nur über Tailwind-Tokens (`text-foreground`, `bg-card`, `text-muted-foreground`…), nie Hex hart.
- **Wiederverwenden** statt kopieren — erst in `components/shared/` + `components/ui/` schauen.
- **Verifizieren ist Pflicht**: gescopter `tsc` (nur geänderte Dateien) + QA-Script + Screenshots @1440px + keine Raw-Keys + keine Konsolenfehler. Erst dann „fertig".

---

## Ein typischer Phasen-Durchlauf (was du Nico Schritt für Schritt führst)

1. `git pull` → Phasen-Spec lesen → Muster-Modul ansehen.
2. Code bauen entlang der Definition-of-Done.
3. i18n-Keys in alle 4 Sprachen (kleines `add-*-i18n.mjs`-Script wie die vorhandenen).
4. Demo-Handler bauen, falls die Spec es verlangt.
5. Gescopten Typecheck einrichten + laufen lassen (`tsconfig.<phase>check.json`), QA-Script schreiben + laufen lassen.
6. Screenshots prüfen (Raw-Keys? abgeschnitten? Emojis? leere Screens?).
7. Verify-Checkliste aus dem RUNBOOK abhaken.
8. Ein Commit + Push. Nico bekommt SHA + Checkliste + Screenshots für das Review.

> Wenn etwas unklar ist oder Backend fehlt: **nicht raten** — kurz in `.planning/backend-gaps.md` notieren bzw. Nico an Darien/Claude (Haupt-Team) verweisen.
