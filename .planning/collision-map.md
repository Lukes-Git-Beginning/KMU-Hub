# Kollisions-Karte — Multi-Stream-Bau (3 Ströme parallel)

> **Pflichtlektüre für jeden Bau-Strom (Nico · Luke · Dein-PC).** Verlinkt aus jedem KICKOFF.
> Zweck: Drei Claude-Sessions bauen gleichzeitig FE-Module und pushen alle nach `main`. Diese Karte sagt, **welche Dateien sich überschneiden** und **wie man sie ohne Bruch anfasst**, damit nichts kaputtgeht.
> Stand: 2026-06-08 (Kickoff Marathon).

---

## 1. Goldene Regeln (für alle Ströme, nicht verhandelbar)

1. **Getrennte Klone.** Jeder Strom arbeitet in einem **eigenen** Repo-Klon (eigenes `node_modules`, eigener Dev-Port). Niemals zwei Sessions im selben Ordner.
2. **`git pull --rebase` VOR jedem Push.** Immer. Drei Ströme bewegen `main` ständig.
3. **Atomarer Push:** `git add` → `git commit` → `git pull --rebase origin main` → `git push`. So bleibt `main` linear.
4. **Niemals** `--force`, **niemals** `reset --hard` auf gemeinsame History. Bei Divergenz nur `pull --rebase`.
5. **Ein Strom = ganze Module** (siehe Lane-Tabelle). Du fasst NUR die Dateien deiner Module an + die Hot Files unten (additiv).
6. **Hot Files nur ADDITIV.** Nie umstrukturieren/sortieren/formatieren — nur deine Zeilen hinzufügen. Sonst Konflikt mit den anderen.

---

## 2. Hot Files — die berührt JEDER Strom

Diese Dateien fasst jedes neue Modul/jede Phase an. Hier ist die einzige echte Kollisionsgefahr. Regel pro Datei:

| Hot File | Wofür | Wie anfassen |
|---|---|---|
| `desktop/src/renderer/src/i18n/messages/de.json` · `en.json` · `fr.json` · `it.json` | jede Phase fügt Übersetzungs-Keys | Keys sind **modul-namespaced** (`vertraege.*`, `calendar.*`) → nie inhaltliche Überschneidung. Nur Text-Merge im selben File. **Bei Rebase-Konflikt: beide Blöcke behalten** (additiv), dann weiter. Keys per Script einfügen, nicht von Hand sortieren. |
| `desktop/src/renderer/src/App.tsx` | neue Route + Lazy-Import pro Modul | Nur **eine Zeile Lazy-Import** + **eine Zeile Route** ergänzen, an der bestehenden Stelle. Nichts umsortieren. |
| `desktop/src/renderer/src/modules/settings/module-settings-registry.tsx` | Modul-Settings-Eintrag | Nur **einen Eintrag** ans Array anhängen. |
| Sidebar/Nav-Config (`config/`/`layout/` — Modul-Liste der App-Shell) | neues Modul sichtbar schalten | Nur **einen Eintrag** ergänzen. |
| `desktop/src/renderer/src/components/shared/index.ts` (Barrel) | neue Shared-Komponente exportieren | Nur **eine Export-Zeile** anhängen. Die Komponente selbst ist eine **neue Datei** (kein Konflikt). |
| `desktop/src/renderer/src/mocks/handlers/` (Registry/Index) | neuen Demo-Handler registrieren | Handler ist eine **neue Datei** pro Modul; nur **eine Registrierungs-Zeile** ergänzen. |

**Faustregel:** Wenn ein Rebase-Konflikt auftritt, ist es fast immer in einer dieser Dateien und fast immer **additiv** (beide Seiten haben unterschiedliche neue Zeilen hinzugefügt) → beide behalten, `git rebase --continue`.

---

## 3. Was NICHT kollidiert (beruhigend)

- **Modul-eigene Ordner** (`modules/<deinModul>/…`) — gehören exklusiv deinem Strom.
- **Modul-eigene Stores** (`stores/<modul>*.ts`) — neue Dateien, kein Konflikt.
- **Modul-eigene Hooks** (`api/hooks/use<Modul>*.ts`) — neue Dateien.
- **Modul-eigene Mock-Handler** (`mocks/handlers/<modul>.ts`) — neue Dateien.
- **Scoped tsconfigs** (`tsconfig.<phase>check.json`) — jeder Strom benennt seine eindeutig (z.B. `tsconfig.vertraege-p1.json`), kein Konflikt.
- **QA-Scripts** (`scripts/qa-<modul>-*.mjs`) — eindeutige Namen pro Modul.

---

## 4. Lane-Zuteilung (VORSCHLAG — Darien bestätigt/justiert)

> Solange nicht bestätigt: nicht starten. Sobald bestätigt, ist diese Tabelle verbindlich — kein Strom baut außerhalb seiner Lane.

| Strom | Treiber | Lane (Module, Reihenfolge) | Klon / Port |
|---|---|---|---|
| **N — Nico** | Nico | wiki → formulare → berichte → notifications | Nicos Klon, :5173 |
| **D — Dein PC** | Nico/Luke (remote) | calendar → dokumente → zeiterfassung | Hauptklon, :5173 |
| **L — Luke (nachm.)** | Luke | vertraege → dashboard → profil | Lukes FE-Klon, :5173 |
| **L — Luke (vorm.)** | Luke | backend-handover P0 (kein FE) | Backend-Repo |

**Bewusst NICHT morgen** (BE-lastig / mit Darien): dialer · video · mails · security · automatisierung · finanzen-Tiefe · Branchen.

---

## 5. Wenn zwei Ströme doch dieselbe Nicht-Hot-Datei brauchen
Sollte durch die Modul-Zuteilung kaum vorkommen. Wenn doch (z.B. eine geteilte Komponente in `shared/` muss geändert, nicht nur ergänzt werden):
1. Kurz im gemeinsamen Kanal abstimmen — **einer nach dem anderen**, nicht parallel.
2. Wer zuerst fertig ist, committet + pusht; der andere rebased drauf.
3. Im Zweifel: die Änderung als **neue** Variante/Datei bauen statt die geteilte zu mutieren.
