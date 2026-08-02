# Übergabe — Merge Nachtlauf 3 + Vorbereitung Nachtlauf 4

> Stand 2026-08-02, ~15:00. Für das neue Fenster. Reihenfolge ist verbindlich: **erst mergen und
> deployen, dann Branch zurücksetzen, dann Backlog committen, dann PR anlegen, dann starten.**

## 0. Ausgangslage in einem Absatz

Nachtlauf 3 ist abgenommen: 61 Units, 101 Commits, Migrationen 256–268, **PR #16** (Draft,
MERGEABLE, `d8d868b2`) mit **grünem CI** (Lint / Test `-race` / E2E / OpenAPI) **und grünem
CI Desktop**. Zwei Treiber-/Testfehler sind unterwegs gefixt. Offen ist nur noch der Merge — und
der löst über den self-hosted Runner automatisch den Prod-Deploy aus.

Alle Prod-Vorabprüfungen sind **bereits erledigt** (2026-08-02, read-only abgefragt):

| Prüfung | Ergebnis |
|---|---|
| Prod-Migrationskopf | **255, `dirty=false`** — exakt `main`, keine Drift |
| `roles`-Bestand | 4 Zeilen, alle System-Presets → **kein Backfill nötig** für 000256 |
| `SYSTEM_SMTP_*` in `.env.production` | vollständig gesetzt |

---

## 1. Merge (der eigentliche Schritt)

```bash
GH="C:/Program Files/GitHub CLI/gh.exe"
"$GH" pr ready 16                 # Draft -> ready
"$GH" pr merge 16 --merge         # KEIN --squash: die Unit-pro-Commit-Historie ist der Wert
```

Danach zieht **CD automatisch** (self-hosted Runner auf `kmuhub-prod`, 0 GitHub-Minuten) und
migriert Prod von 255 auf **268**.

**Beobachten, nicht wegklicken:**

```bash
"$GH" run list --workflow=cd.yml --limit 3
ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195 \
  "sudo docker compose --env-file /opt/kmuhub/.env.production \
   -f /opt/kmuhub/deploy/docker/docker-compose.yml \
   -f /opt/kmuhub/deploy/docker/docker-compose.prod.yml \
   exec -T postgres psql -U kmuhub -d kmuhub -tA -c 'SELECT version, dirty FROM schema_migrations;'"
```

**Erwartung: `268|f`.** Steht dort `dirty=t`, ist eine Migration mittendrin gestorben — dann
**nicht** weiterdeployen, sondern die Stelle suchen (`deploy.sh`-Auto-Rollback rollt nur Code,
nicht die DB → sonst Drift).

### Was mit diesem Deploy fachlich live geht

- **RBAC-Fundament** (000256): 282 Capability-Keys, 7 Preset-Rollen, Scope an jedem Grant.
  Alte grobe Keys bleiben gültig — niemand wird ausgesperrt (`RequirePermissionAny`).
- **Geplante Berichte versenden ab jetzt real.** Der Compose-Passthrough für `SYSTEM_SMTP_*`
  erreicht den `berichte`-Service zum ersten Mal; vorher liefen Schedules als `skipped`. Das ist
  gewollt, aber es ist eine Verhaltensänderung nach außen — kurz drauf schauen, dass keine
  Alt-Schedules sofort einen Schwung Mails auslösen.
- E-Rechnung-Ausgang, Lexware-Sync (vorher No-op), Finance-Erweiterungen, Dokumente-Share-Links.

### Danach Smoke

```bash
ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195 "cd /opt/kmuhub && bash deploy/scripts/smoke.sh"
```

Und der Live-Beleg für das RBAC-Fundament — muss eine **nicht-leere** Capability-Liste liefern:
`GET /api/v1/auth/me/permissions` als Admin.

---

## 2. Branch für Lauf 4 zurücksetzen

Nach dem Merge ist `backend-loop` inhaltlich in `main` aufgegangen. Lokal liegt **ein Commit
darüber**, den ich für Lauf 4 vorbereitet habe (der neue Backlog, siehe §3):

```bash
git checkout main && git pull
git checkout backend-loop
git merge origin/main            # kein rebase - der Guard blockt Rebase, und Force-Push ist verboten
git log --oneline -3             # erwartet: Backlog-Commit oben auf, darunter der Merge
```

---

## 3. Backlog Lauf 4 — was drin ist und was das heißt

Der neue `BACKLOG.yml` liegt als lokaler Commit bereit. **Freigegeben ist RBAC Welle 1b**
(deine Entscheidung von heute); Phase 4 (Branchen-BE), neue `config.RequireX`-Assertionen und das
Scharfschalten neuer `modules.*`-Flags bleiben gesperrt.

| Block | Units | Inhalt |
|---|---|---|
| A — RBAC Welle 1b | 9 | Rollen-CRUD `/admin/roles`, Klon-Semantik, Permissions-PUT, User-Rollen, Guardrails, Audit-Events |
| B — Sicherheit / RLS | 6 | `user_roles`, 4× `*_custom_field_values`, `email_contact_links`, `validation_rules`+`workflow_rules`, `events`-Partition, Allowlist-Audit |
| C — Automatisierung | 3 | `http_request`-Action (SSRF-kritisch, opus), Webhook-Trigger, Cron-Poller |
| D — verifizierte FE-Lücken | 4 | CRM-Timeline, Kalender-Ressourcen-Buchungen, Admin-Billing, Vendor-Access |
| E — verifizierte Bestandsbugs | 2 | `PurchaseOrder.total_amount` (hartkodiert `"0"`), Fuhrpark-Führerscheinkontrolle |
| F — Rest aus Lauf 3 | 5 | Mails Multi-Account + Vorlagen, Video-Download, `work.start_date`, Wiki-Share-Token-Routen |

**≈ 29 Units, realistisch 8–9 Stunden** — nicht die vollen 12. Das ist ein Befund, keine
Nachlässigkeit: ich habe die FE-Client-Pfade **aller** Module gegen die 779 registrierten Routen
diffed, und die dünnen Module (fuhrpark, inventar, vermietung, einkauf, produktion, schichten,
rapporte) haben **keine Routen-Lücken mehr** — die Läufe 1–3 haben sie geschlossen. Auch
`backend-gaps.md` ist an mehreren Stellen überholt (Beispiel: `POST /einkauf/pos/{id}/cancel`
steht dort als fehlend, ist aber seit Lauf 1 gebaut).

Der Backlog auf 40 Units aufzufüllen hätte geheißen, Units aus veralteten Notizen zu erfinden, die
der Loop dann reihenweise als `blocked` zurückgibt. Zwei ehrliche Optionen:

1. **So starten.** Der Loop beendet sich selbst mit `ALLE UNITS ABGEARBEITET` und legt `STOP` an,
   wenn er früher fertig ist. Kein Schaden, nur ein kürzerer Lauf.
2. **Vorher eine Recherche-Session** (1–2 h) auf Feature-Ebene: `backend-gaps.md` gegen den
   aktuellen Code entstauben und daraus 10–12 verifizierte Units nachlegen. Das lohnt sich
   ohnehin bald — die Datei ist die Grundlage für Lauf 5.

---

## 4. PR für Lauf 4 anlegen (sonst kein CI am Ende)

`ci.yml` triggert nur auf `push: [main]` und `pull_request: [main]`. Ohne offenen PR pusht der
Treiber zwar, aber **es läuft kein CI**. Reihenfolge zwingend, weil beide Review-Workflows auf
`opened` feuern und **kein Draft-Gate** haben:

```bash
"$GH" workflow disable "Claude PR Review"
"$GH" workflow disable "Security Review"
"$GH" pr create --base main --head backend-loop --draft --title "Backend-Nachtlauf 4" --body "..."
"$GH" workflow enable "Claude PR Review"
"$GH" workflow enable "Security Review"
```

Das Wieder-Aktivieren darf **sofort** nach dem Anlegen passieren — die Workflows feuern nicht
rückwirkend, und der Treiber-Push in der Nacht löst nur `synchronize` aus. Willst du auch das
vermeiden, lässt du sie bis morgen früh aus (dann aber wirklich wieder anschalten).

---

## 5. Lauf 4 starten — 20:00 bis 08:00

```powershell
# Immer zuerst, unter Aufsicht:
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -DryRun

# Der Lauf:
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 `
  -UntilTime "08:00" -Effort xhigh -BudgetUsd 20
```

`-BudgetUsd` ist der Deckel **pro Iteration**, nicht für den Lauf (Lauf 3: max. $17,63 in einer
Iteration, $538,92 API-Äquivalenz gesamt — auf dem Max-Abo kein Geld, nur Wochen-Cap).

Stoppen: `New-Item -ItemType File .planning\backend-block\loop\STOP` — beendet nach der laufenden
Iteration.

**Der Treiber hält die Maschine jetzt selbst wach** (`ES_SYSTEM_REQUIRED`, seit `e3b1afca`). In
Lauf 3 ging Windows vor diesem Fix in Standby und riss Iteration 15 mit „Response stalled
mid-stream" ab — 47 Minuten weg.

---

## 6. Fallstricke, die heute schon Geld/Zeit gekostet haben

- **Kein `tail -f` auf `run.log`.** Ein verwaister tail-Prozess sperrt die Datei, PowerShell kann
  nicht mehr schreiben, und der Fortschritt verschwindet still — in Lauf 3 fehlten so die
  Iterationen 1–17. Der Treiber **warnt jetzt** einmalig auf der Konsole, statt es zu schlucken.
  Zum Mitlesen stattdessen: `git log --oneline main..backend-loop`, `logs/iter-*.json`, `JOURNAL.md`.
- **Der Push am Ende sieht rot aus und ist es nicht.** git schreibt seinen Fortschritt nach
  stderr. Das hat in Lauf 3 die komplette CI-Phase abgerissen (`ErrorActionPreference = Stop`).
  Gefixt in `4c1041b8` über `Invoke-Native`; empirisch belegt, dass auch `2>$null` nicht schützt.
- **`gh run list` allein beweist nichts.** Job-Steps mitprüfen (`gh run view <id> --json jobs`):
  Jobs mit 0 Steps sind die Actions-Billing-Wall, kein Erfolg.
- **Ein Workflow, der nicht läuft, ist nicht grün.** `ci-desktop.yml` war auf `main` 11 Tage rot,
  unbemerkt, weil der `desktop/**`-paths-Filter ihn nur bei Desktop-Änderungen weckt. Vor
  Merge/Launch den Stand **aller** Workflows prüfen, nicht nur des zuletzt gelaufenen.

---

## 7. Was noch deine Entscheidung braucht (nicht blockierend)

- **Die zwei `blocked` Units** aus Lauf 3 — beide brauchen eine Produkt-, keine Code-Entscheidung:
  - `fe-projects-guest-overview`: Milestones/Status-Updates als echtes Datenmodell bauen
    (2 Tabellen + Schreibpfad), oder die Gast-Route aus dem Scope nehmen?
  - `g-vermietung-inspection-upload`: den toten RPC entfernen, oder ihn für externe Mieter ohne
    CRM-Login echt bauen?
- **`.knowledge/` ist überholt** — `datenbank.md` nennt Migrationskopf 213 (real: 268), `api.md`
  kennt die neuen Endpoint-Domains nicht. Nach dem Merge einmal `/update-knowledge`.
- **`MEMORY.md`** liegt bei 19,2 KB über dem Soft-Budget (18 KB, Hardlimit 25 KB). Weiter kürzen
  geht nur noch auf Kosten von Wissen, das du aktiv nutzt.
