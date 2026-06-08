# Kollisions-Karte — Multi-Stream-Bau (3 Ströme parallel, Branch-Modell)

> **Pflichtlektüre für jeden Bau-Strom (Nico · Luke · Dein-PC).** Verlinkt aus jedem KICKOFF.
> Zweck: Drei Claude-Sessions bauen am selben Tag parallel FE-Module. Diese Karte sagt, **wie sie sich nicht ins Gehege kommen** und **welche Dateien beim späteren Zusammenführen kollidieren**.
> Stand: 2026-06-09 (Marathon-Start). **Modell: Branch-pro-Strom** (Darien merged + reviewt, wenn er zurück ist).

---

## 0. Warum Branch-pro-Strom (und nicht alle auf `main`)
Morgen ist Darien weg → **niemand merged/überwacht `main` live**. Drei unbeaufsichtigte Sessions direkt auf `main` = Risiko (eine bricht den Build → blockiert die anderen). Lösung für diesen Marathon: **jeder Strom arbeitet auf seinem eigenen Branch**, `main` bleibt stabil eingefroren. Darien führt die Branches beim Review zusammen. (Bewusste Abweichung vom direct-to-main-Default — gilt nur für den unbeaufsichtigten Parallel-Lauf.)

## 1. Goldene Regeln (für alle Ströme, nicht verhandelbar)
1. **Eigener Klon + eigener Branch.** Jeder Strom: eigener Repo-Klon, ein **Strom-Branch** für den Tag:
   - Nico → `marathon/nico`
   - Dein-PC → `marathon/dein-pc`
   - Luke (FE) → `marathon/luke-fe`
2. **Session-Start einmalig:** `git checkout main && git pull && git checkout -B marathon/<strom>` (oder den bestehenden Strom-Branch auschecken, falls schon angelegt).
3. **Pro Phase = ein Commit** auf dem Strom-Branch (Conventional, Englisch, **keine** AI-Attribution).
4. **Nach jeder Phase: `git push -u origin marathon/<strom>`** (Backup + sichtbar). **Niemals nach `main` pushen. Niemals `main` auschecken/mergen.**
5. **`main` ist heute für dich eingefroren** — du pullst `main` NICHT mitten am Tag (sonst verschiebt sich deine Basis). Nur am allerersten Session-Start einmal.
6. **Du baust NUR die Module deiner Lane** (Lane-Tabelle §4). Nichts außerhalb.
7. **Niemals** `--force`, **niemals** `reset --hard`.

## 2. Hot Files — kollidieren erst beim MERGE (nicht während des Tages)
Weil jeder auf seinem Branch ab demselben `main`-Stand baut, gibt es **während des Tages keine Kollision**. Beim späteren Zusammenführen (Darien) kollidieren genau diese Dateien — alle **additiv** (jeder Strom hat unterschiedliche neue Zeilen). Darien löst sie mit Claude beim Review:

| Hot File | Warum Konflikt beim Merge | Auflösung |
|---|---|---|
| `i18n/messages/{de,en,fr,it}.json` | jeder Strom fügt Keys | **beide Blöcke behalten** (Keys sind modul-namespaced → nie inhaltlich doppelt) |
| `App.tsx` | jeder fügt Route + Lazy-Import | beide behalten |
| `module-settings-registry.tsx` | jeder fügt Settings-Eintrag | beide behalten |
| Sidebar/Nav-Config | neues Modul sichtbar | beide behalten |
| `components/shared/index.ts` · `mocks/handlers/`-Registry | neue Export/Registrierungs-Zeilen | beide behalten |

**Damit der Merge leicht bleibt — Regel für alle Ströme:** an Hot Files **nur additiv** arbeiten (deine Zeilen anhängen), **nie** umsortieren/umformatieren. Keys per Script einfügen, nicht von Hand umstrukturieren.

## 3. Was nie kollidiert
Modul-eigene Ordner (`modules/<deinModul>/`), Stores (`stores/<modul>*`), Hooks (`api/hooks/use<Modul>*`), Mock-Handler (`mocks/handlers/<modul>.ts`), scoped tsconfigs, QA-Scripts — alles neue, modul-spezifische Dateien. Solange jeder in seiner Lane bleibt: null Konflikt.

## 4. Lane-Zuteilung (VORSCHLAG — Darien bestätigt/justiert)
| Strom | Treiber | Branch | Lane (Module, Reihenfolge) |
|---|---|---|---|
| **N — Nico** | Nico | `marathon/nico` | wiki → formulare → berichte → notifications |
| **D — Dein PC** | Nico/Luke (remote) | `marathon/dein-pc` | calendar → dokumente → zeiterfassung |
| **L — Luke (nachm.)** | Luke | `marathon/luke-fe` | vertraege → dashboard → profil |
| **L — Luke (vorm.)** | Luke | — (Backend-Repo) | backend-handover P0 |

**Bewusst NICHT morgen:** dialer · video · mails · security · automatisierung · finanzen-Tiefe · Branchen (BE-lastig / mit Darien).

## 5. Beim Review zusammenführen (Darien, wenn zurück)
Pro Strom-Branch: Review-Fäden in `reviews/<modul>.md` durchgehen → `git checkout main && git merge --no-ff marathon/<strom>` → Hot-File-Konflikte additiv lösen (beide behalten) → `npm run` Smoke + scoped tsc → push. Ein Strom nach dem anderen.
