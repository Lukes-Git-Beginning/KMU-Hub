# Session-Runbook — „An den Phasen weiterbauen"

> **Trigger:** Sobald Darien sinngemäß sagt **„mach an den Phasen weiter" / „weiter an den Phasen" / „nächste Phasen"** → diese Session folgt **exakt diesem Runbook**.
> **Zweck:** Ein wiederholbarer Zyklus, den jede frische Session vom Terminal-Start an gleich durchläuft, bis Cosmi 1.0 gebaut ist. **Der Fortschritt lebt im `MASTER-PLAN.md` (Haken), nicht im Session-Context** — darum kann man nach jeder Welle ein neues Terminal starten und einfach wieder „weiter" sagen.
> **Pläne:** `MASTER-PLAN.md` (Frontend, Wellen + Phasen) · `BACKEND-PLAN.md` (Luke + FE↔BE-Warte-Mapping) · `backend-gaps.md` (Backend-TODO-Detail).

---

## Der Zyklus (8 Schritte)

### 1 · Laden (Kontext ziehen)
- `git pull` (neuesten Stand holen — beide Terminals pushen nach main).
- Lesen: `MASTER-PLAN.md` (§6 Wellen + §2/§3 Haken) + `BACKEND-PLAN.md` (was ist backend-seitig da/offen).
- `git log --oneline -15` + Migrations-Kopf (`ls backend/migrations | tail`) — realen Stand prüfen, nicht nur Plan glauben.

### 2 · Planen (nächste Welle, Main- + Sub-Lane)
- Aus §6 die **nächste offene Welle** nehmen (aktuell Reihenfolge 1→2→3→4→5, Review=6 zuletzt).
- Die nächsten **~5 Phasen** (5er-Batch-Rhythmus) auswählen.
- In **zwei disjunkte Lanes** aufteilen (keine Hot-File-Überschneidung — verschiedene Module): **Main-Lane** (dieses Terminal) + **Sub-Lane** (zweites Terminal).
- Kurz-Plan formulieren: welche Phasen, welche Lane, welche Backend-Abhängigkeiten (🔌 vorhanden / 🔒 fehlt).

### 3 · Recherchieren
- Pro Modul/Phase: Ist-Stand im Code (Page/Store/Hook/MSW-Handler) + Markt-Recherche wo nötig (`cosmi-modul-marktvergleich.txt`, Web).
- **Backend-Check (PFLICHT, Darien-Regel):** existiert der Endpoint (proto + gRPC + Route + Migration)?
  - **Ja → direkt ans echte Backend hängen** (🔌, kein neuer Mock).
  - **Nein → mock-first bauen** + Eintrag auf Lukes TODO (`backend-gaps.md` + `BACKEND-PLAN.md`) **+ 🔌-„verdrahten"-Zeile im MASTER-PLAN §2/§3**, damit das Verdrahten nach Lukes Bau nicht vergessen wird.

### 4 · Rückfragen bündeln  ◀ EINZIGES Frage-Gate
- Alle offenen Produkt-/Scope-/Design-Fragen **gesammelt** an Darien (AskUserQuestion), inkl. dem Plan zur Bestätigung + dem geplanten Sub-Paket.
- **Nach dem OK läuft der Rest autonom** (Schritte 5–8) bis die Welle/der Batch durch ist.

### 5 · Bauen (Build-+-Verify pro Phase)
- Standard: `.planning/nico-block/WORKFLOW.md`. Pro Phase: bauen → **echtes Backend anhängen wo vorhanden** → i18n ×4 (`{var}`, ICU-Plural, Fragmente `i18n/_wave-fragments/<modul>.json`, **append-only**) → Demo-Handler nur wo Backend fehlt → gescopter Typecheck (nie Full-tsc als Gate).
- Projektweite Standards beachten: Detail = `shared/DetailModal` (zentriert), ganze Zeile klickbar, sticky Back/Close, keine ASCII-Umlaute, keine Emojis in UI, wiederverwendbar in `shared/`.

### 6 · Screenshot-QA (selbst, Bilder WIRKLICH ansehen)
- Dev-Server (`npm run dev`, Main-Klon :5173 / Sub-Klon :5174) + `scripts/qa-<modul>.mjs` (Route `/#/<modul>`, Onboarding-Suppress).
- **Screenshots mit Read-Tool ansehen** (Agent-Asserts sind unzuverlässig): echte Daten, keine Raw-Keys, keine leeren Zustände/Crashes, Umlaute korrekt, Layout. Iterieren bis grün.

### 7 · Speichern
- Pro Phase **ein Commit** (Conventional, Englisch, **keine AI-Attribution**). Atomar: `commit → git pull --rebase → git push`.
- **MASTER-PLAN.md aktualisieren:** erledigte Phasen `[ ]`→`[x]`, Stand/Datum, ggf. 🔌-verdrahten-Zeilen + Backend-TODOs ergänzen.
- Memory-Update wo sinnvoll (neue Architektur-Falle, Modul-Batch fertig).

### 8 · Abschluss / neues Terminal
- Welle/Batch durch → kurzer Stand an Darien (was fertig, was als Nächstes, Backend-Wartepunkte).
- Darien startet ein **neues Terminal** und sagt wieder „weiter" → Schritt 1. So bis alle Bau-Wellen (1–5) durch sind, dann **Welle 6 = Review** (separat, Team, händisch).

---

## Git-Modell (zwei Terminals)
- **Getrennte Klone, beide bauen, direct-to-main.** Terminal A = Klon `KMU Hub` (Dev-Port 5173), Terminal B = `KMU-Hub-review` (5174). (Fehlt der zweite Klon → einmalig `git clone … KMU-Hub-review` + `npm install`.)
- **Disjunkte Lanes** (verschiedene Module) → keine inhaltliche Kollision. **Hot Files** (i18n-JSONs, `App.tsx`, `module-settings-registry.tsx`, `mocks/handlers/index.ts`, Sidebar-Nav) **nur additiv** anfassen, nie umsortieren. Details: `archiv/collision-map.md`.
- **Atomarer Push:** `git add -p` → commit → `git pull --rebase` → `git push`. Bei seltenem Hot-File-Konflikt: beide Blöcke behalten (additiv).
- **Rolle:** Das Terminal, in dem „weiter" gesagt wird = **Main** (plant beide Lanes, baut Main-Lane, schreibt Sub-Paket). Das zweite Terminal = **Sub** (baut die Sub-Lane aus dem Paket).

## Sub-Paket (Format, von Main geschrieben)
- Ablage: `.planning/parallel-batch/sub-<lane>.md` + ein **fertiger Start-Text** zum Reinkopieren ins 2. Terminal.
- **Der Start-Text beginnt IMMER mit dem Ziel-Verzeichnis als kopierbarem Öffnen-Befehl** (der jeweils ANDERE Klon als Main), damit Darien das Terminal direkt dort öffnen kann — Format:
  ```
  ▶ Zweites Terminal hier öffnen:  cd "C:\Users\darie\Documents\KMU Hub"   (oder …\KMU-Hub-review)
     dann:  claude   → diesen Text einfügen.   Dev-Port: 5173 (bzw. 5174, der andere als Main)
  ```
- Inhalt danach: Lane-Module + die 5 Phasen, Backend-Check-Ergebnis (🔌/🔒) je Phase, Referenz-Pfade, Verify-Checkliste, **Pre-Flight** (`git pull` + Migrations-Kopf), „committet+pusht atomar, bleibt in seiner Lane".
- Muster: `.planning/parallel-batch/` (`main-*.md` / `sub-*.md` / `qa-combined.md`) + `archiv/two-terminal-nico-workflow.md`.

## „Durch" — Definition of Done für 1.0
Alle Bau-Wellen 1–5 abgehakt (§2/§3/§6) + alle 🔌-verdrahten-TODOs erledigt (Lukes Backend nachgezogen) → dann **Welle 6 Review** starten. Erst danach ist Cosmi 1.0 vorzeigbar.
