# Prompt — Blocked-Units entscheiden, Loop-Mechanik nachziehen, Lauf 8 planen

> Für eine frische Session im Repo-Root `C:\Users\Luke\Documents\KMU Hub`.
> Erstellt 2026-08-10 nach dem Review und Merge von Lauf 7.

---

Nachtlauf 7 ist gemergt und deployt. Production ist verifiziert: `main` auf `c6d1c972`,
Migrationskopf **309 clean**, Server-Git auf `main` (nicht detached), 30 Container healthy,
`/health` meldet `commit: c6d1c972` mit 23 registrierten Services. `backend-loop` steht auf
demselben Commit wie `main`.

Der vollständige Review liegt in
`~/.claude/plans/nacthlauf-7-ist-durch-concurrent-mccarthy.md` und im Abschlussblock am Ende von
`.planning/backend-block/loop/JOURNAL.md` — **lies beides zuerst**. Alles unten Beschriebene ist
dort belegt.

Deine drei Aufgaben, in dieser Reihenfolge: **(A)** die neun `blocked`-Units aufräumen,
**(B)** die Loop-Mechanik nachziehen, **(C)** Lauf 8 vorbereiten.

**Arbeitsweise:** Nutze für Recherche und Bauarbeit primär Subagenten (max. 3 parallel, Sonnet),
damit das Hauptkontextfenster schlank bleibt — Conclusions, not transcripts. Subagenten können
nicht nachfragen, gib ihnen alles mit: Pfade, Constraints, Akzeptanzkriterien. Die
Produktentscheidungen in Teil A gehören ins Hauptfenster und an Luke, nicht an einen Subagenten.

---

## A · Die neun `blocked`-Units aufräumen

Sie liegen seit bis zu drei Läufen im Backlog und werden bei jedem Lauf neu übersprungen. Das ist
Reibung ohne Ertrag: der Loop liest sie jedes Mal mit, und sie verfälschen den Backlog-Stand.

**Verify-first, in jedem einzelnen Fall.** Die `blocked_reason`-Texte stammen aus Lauf 6 und 7 und
können überholt sein — Lauf 7 hat zwölf Bugs gefixt, die den Code an mehreren dieser Stellen
angefasst haben. Prüf jede Begründung gegen den heutigen Code, **bevor** du sie Luke vorlegst. Eine
Frage zu stellen, die der Code längst beantwortet, kostet Luke unnötig Zeit.

Setz dafür bis zu drei Explore-Subagenten parallel an (je drei Units), mit dem Auftrag: „Gilt diese
`blocked_reason` gegen den aktuellen Code noch? Beleg mit Datei:Zeile, nicht mit Vermutung."

### A1 — Die vier, die gar keine Produktfrage sind

Diese vier gehören **aus dem Nachtloop-Backlog heraus**, nicht in eine weitere Entscheidungsrunde.
Der Loop kann sie strukturell nicht bearbeiten, also verstopfen sie nur die Queue:

| Unit | Warum sie nicht in den Loop gehört | Wohin |
|---|---|---|
| `fe-helpdesk-ticket-number` | Reiner Frontend-Fix | Frontend-Session / `.planning/nico-block/` |
| `g-public-token-landing-pages` | Eigenes Projekt (Hosting, Design, i18n, Consent) — Cosmi ist eine Electron-App, die sieben gebauten Public-Token-Routen haben keine Einlöse-Oberfläche | Eigener Projekt-Backlog |
| `g-decode-disallow-unknown-fields` | Repo-weite Entscheidung mit Breaking-Potenzial für jeden Client | Braucht bewusste Freigabe, dann eigene Session |
| `g-admin-billing` | FE macht gar keinen Backend-Call (`useBilling.ts` = Mock + localStorage); Rechnungshistorie bräuchte ein echtes Payment-Gateway | Produkt-Backlog, nach Payment-Gateway-Entscheidung |

Schlag Luke vor, sie in eine eigene Datei zu verschieben (z. B.
`.planning/backend-block/loop/BACKLOG-PARKED.yml` oder je nach Ziel in den passenden Block) und aus
`BACKLOG.yml` zu entfernen. **Nicht löschen** — die Recherche dahinter ist wertvoll, sie ist nur am
falschen Ort.

### A2 — Die fünf echten Entscheidungen

Für jede: kurz den verifizierten Ist-Zustand, dann die Frage, dann eine **Empfehlung mit
Begründung**. Leg sie Luke per `AskUserQuestion` vor — gebündelt, nicht einzeln, und nicht mehr als
vier Fragen pro Aufruf.

**1. `fix-einkauf-contract-call-no-value-check`** — der einzige, den Lauf 7 selbst neu geblockt hat.
`Service.CreateContractCall` (`internal/einkauf/service_extended.go:610`) prüft nur `v < 0`, lädt
den Rahmenvertrag nur um dessen Existenz zu prüfen (Zeile 620) und vergleicht `Amount` an keiner
Stelle gegen `TotalValue - UsedValue`. `ContractStatus` (`models_extended.go:50-55`) kennt
`draft`/`active`/`expired`. Fragen: (a) Abruf ablehnen, sobald `total_value - used_value`
überschritten wird? (b) Falls ja — auch für `draft`-Verträge, oder nur `active`? (c) Ist bei
`expired` überhaupt noch ein Abruf erlaubt?

**2. `a-inbox-sla`** — `sla_policies` existiert (Migration 000077), gehört aber vollständig dem
Helpdesk-Bounded-Context (hängt an `ticket_queues.sla_policy_id`, nur von `internal/helpdesk`
gelesen). Inbox hat keine Spalte, keinen Code-Pfad, keine Migration, die je darauf verweist.
Frage: teilt Inbox dieselbe Tabelle wie Helpdesk, bekommt sie eine eigene, oder fällt SLA für
Inbox aus dem Scope?

**3. `g-hr-salary-statements`** — kein Payroll-Datenmodell im Backend (nur `HourlyRate` am
Mitarbeiter), das FE erwartet berechnete Brutto- und Nettobeträge. Drei Wege: echtes Payroll-Feld,
PDF-Upload als Dokumentenkategorie, oder Scope-Streichung.

**4. `fe-projects-guest-overview`** — kein Datenmodell für Meilensteine und Status-Updates
(verifiziert: keine Tabelle, kein Struct, kein Schreibpfad). Der FE-Mock ist Platzhalter-Fiktion.
Neubau vs. Datenreduktion vs. Streichung.

**5. `g-csat-public-surface`** — CSAT ist auf Produktion doppelt funktionsunfähig: `SYSTEM_SMTP_*`
wird im Compose nur an `auth`, `biz` und `berichte` durchgereicht, **nicht** an `helpdesk` (der
Container loggt `system SMTP not configured` und startet den Dispatcher nie), und
`CSAT_SURVEY_BASE_URL` zeigt auf `https://app.zentria.tech/csat`, wo nichts liegt — der Caddyfile
proxyt die Domain vollständig auf `gateway:8080`. **Der Passthrough allein wäre schädlich:** dann
gehen Umfragen mit totem Link an `ticket.requester_email` raus, also auch an Mitarbeiter. Frage:
CSAT bis zur Public-Surface stilllegen (Flag hart aus, dokumentiert), oder Public-Surface als
eigenes Projekt aufsetzen? Hängt an Entscheidung `g-public-token-landing-pages` aus A1 — beide
brauchen dieselbe Web-Oberfläche.

**Nach den Antworten:** entschiedene Units als reguläre Lauf-8-Units mit `done_when` und `sources`
neu formulieren, gestrichene mit `status: dropped` und Begründung ins Archiv. Keine Unit bleibt
`blocked` liegen, ohne dass im Backlog steht, worauf sie wartet und wer das entscheidet.

---

## B · Loop-Mechanik nachziehen

Alles unter `.planning/backend-block/loop/`. Von Hand, **nicht** als Backlog-Unit — der Loop darf
seinen eigenen Treiber nicht umbauen. Belege für alle vier Punkte stehen im Journal-Abschlussblock.

### B1 — `coverage:` als Pflichtzeile ins Journal-Template

**Der wichtigste Punkt.** Nur 8 von 71 Iterationen (11 %) haben in Lauf 7 eine Coverage-Zahl
notiert — obwohl Coverage das erklärte Laufziel war. Der Endstand musste nachträglich aus dem
CI-Artefakt rekonstruiert werden. Die `mutations-probe:`-Zeile stand explizit im Backlog-Kopf und
kam 71/71. Der Unterschied ist allein die ausdrückliche Nennung.

In `ITERATION.md` Schritt 6 das Journal-Template um eine Pflichtzeile ergänzen, mit fester
Messmethode:

```markdown
- coverage: <Paket> <vorher> % -> <nachher> % (`go test -coverprofile=... ./internal/<pkg>/`,
  `go tool cover -func | tail -1`) | n.a. (kein Coverage-Ziel in dieser Unit)
```

Und in `GATE-COMMANDS.md` einmal festschreiben, dass laufübergreifend **nur** das
CI-`coverage-report`-Artefakt mit `grep -v "github.com/kmuhub/kmuhub/proto/"` vergleichbar ist.
Anlass: Lauf 6 nannte 26,0 % für `internal/server` (repo-weit gewichtet), Lauf 7 maß 27,6 %
(paket-eigen) — dieselbe Zahl für zwei verschiedene Dinge.

### B2 — Zeitstempel-Vorgabe verschärfen

32 von 72 Journal-Überschriften tragen „(Lauf 7)" oder „(siehe Commit-Zeit)" statt der Uhrzeit, die
`run-loop.ps1` seit Lauf 7 im Laufkontext-Block ausdrücklich mitliefert. Die Template-Zeile
`<YYYY-MM-DD HH:MM>` liest sich als Formatvorschlag statt als Pflichtwert.

Formulierung auf „**exakt** der Wert aus dem Laufkontext-Block, keine Ersatzangabe" ändern.

### B3 — Treiber warnt bei Nummern-/Zeit-Drift

`run-loop.ps1` liest die höchste Journal-Iterationsnummer ohnehin für seine Fortschrittsanzeige.
Nach jeder Iteration zusätzlich prüfen, ob die neu angehängte Überschrift (a) die Nummer `$i` und
(b) die übergebene Uhrzeit enthält — bei Abweichung eine gelbe Logzeile. Nicht abbrechen, nur
sichtbar machen. In Lauf 7 lief der Treiber-Zähler auf 71, das Journal auf 72; die `iter-NNN.json`
sind dadurch nicht mehr 1:1 auf die Journal-Nummern abbildbar.

### B4 — Toter Abschlusspfad

Der Prompt sagt, das Modell solle bei leerem Backlog `ALLE UNITS ABGEARBEITET` schreiben und `STOP`
anlegen. Der Treiber beendet aber vorher über seinen eigenen Open-Count, also wird dieser Pfad nie
erreicht. Entweder den Treiber den Abschlussblock schreiben lassen oder die Prompt-Passage
streichen — aber nicht beides nebeneinander stehen lassen.

### B5 — Archiv-Ritual

Lauf 7 nach `.planning/backend-block/loop/archive/lauf-7/` archivieren (`JOURNAL.md`,
`BACKLOG.yml`), wie `archive/lauf-6/`. Danach `JOURNAL.md` und `BACKLOG.yml` für Lauf 8 frisch
aufsetzen, mit Pointer-Zeile auf die Archive im Kopf.

---

## C · Lauf 8 planen

### C1 — Coverage-Ausgangslage (gemessen, nicht geschätzt)

Aus dem CI-`coverage-report`-Artefakt von Run **31373430274** (finaler Tree von Lauf 7), gefiltert
wie das CI-Gate. **Gesamt 47,7 %** (Gate 15 %).

Pakete mit ≥ 800 Statements, aufsteigend — hier liegt die Arbeit:

| Paket | Coverage | ungedeckte Statements |
|---|---:|---:|
| `inbox` | 32,3 % | 636 |
| **`gateway`** | **34,9 %** | **14.707** |
| `fuhrpark` | 37,9 % | 633 |
| **`server`** | **47,7 %** | **10.632** |
| `security` | 47,9 % | 784 |
| `chat` | 50,9 % | 693 |
| `automation` | 51,9 % | 602 |
| `plugin` | 53,3 % | 535 |
| `formulare` | 53,5 % | 382 |
| `caldav` | 54,2 % | 501 |

Größte ungedeckte Dateien in `internal/gateway`:
`route_hr.go` (845 von 974 ungedeckt, 13,2 %) · `route_fuhrpark.go` (793/944, 16,0 %) ·
`route_video.go` (786/1039, 24,4 %) · `route_email.go` (621/757, 18,0 %) ·
`route_calendar.go` (601/701, 14,3 %) · `route_inventar.go` (590/772, 23,6 %) ·
`route_document.go` (511/697, 26,7 %) · `route_auth.go` (473/710, 33,4 %) ·
`route_rapporte.go` (468/597, 21,6 %) · `route_helpdesk.go` (445/614, 27,5 %)

Größte ungedeckte Dateien in `internal/server`:
`crm_grpc.go` (1413 von 1490 ungedeckt, **5,2 %**) · `biz_grpc.go` (1028/1100, 6,5 %) ·
`calendar_grpc.go` (903/916, **1,4 %**) · `email_grpc.go` (743/793, 6,3 %) ·
`chat_grpc.go` (523/564, 7,3 %) · `dialer_grpc.go` (454/516, 12,0 %) ·
`websocket.go` (453/475, 4,6 %)

`crm_grpc.go`, `biz_grpc.go` und `calendar_grpc.go` sind zusammen 3.344 ungedeckte Statements bei
1–7 % Abdeckung — und sie liegen auf den kritischen Pfaden (CRM, Finanzen, Kalender). Das ist der
offensichtlichste Hebel für Lauf 8. `route_auth.go` (33,4 %) und `internal/security` (47,9 %) sind
sicherheitsrelevant und verdienen Vorrang vor größeren, aber harmloseren Flächen.

### C2 — Was Lauf 8 als Auflage übernehmen muss

**Die Mutations-Probe bleibt Pflicht.** Sie ist der belegte Grund, warum Lauf 7 zwölf reale Bugs
gefunden hat statt nur Zeilen abzudecken — 71/71 eingehalten. Formuliere sie im Backlog-Kopf
mindestens so scharf wie in Lauf 7, plus die neue `coverage:`-Pflichtzeile aus B1.

**Fix-Units bleiben eigenständig.** Das Muster aus Lauf 7 hat sich bewährt: der Coverage-Test
dokumentiert erst den Bug, eine eigene Folge-Unit fixt ihn und stellt denselben Test auf das
korrekte Verhalten um. Nicht inline fixen.

### C3 — Rahmen, den Luke setzen muss

Frag ihn per `AskUserQuestion` ab, bevor du den Backlog baust:

- **Zeitfenster** (`-UntilTime`, ggf. `-PauseFrom`/`-PauseTo`) und Iterations-Deckel
- **Schwerpunkt:** weiter Coverage (`server`-Kernpakete zuerst?), oder diesmal die entschiedenen
  Units aus Teil A vorziehen, oder gemischt
- **Budget pro Iteration** — Lauf 7 lief mit dem bestehenden Deckel ohne einen einzigen
  `error_max_budget_usd`, 287,69 USD gesamt bei Ø 8,0 min

### C4 — Verifikation vor dem Start

- `backend-loop` auf `origin/main` **mergen** (nicht rebasen), Stand prüfen
- Nächste freie Migrationsnummer **zur Laufzeit** ermitteln, nie aus dem Backlog übernehmen:
  `ls backend/migrations | grep -E '^[0-9]{6}' | sort | tail -1` (aktuell 309)
- Lokale DB erreichbar, Rolle `kmuhub_app` mit Passwort `app_dev` — sonst ist `DATABASE_URL` ein
  Alibi und das Gate keins
- `ITERATION.md` inhaltlich auf Lauf 8 stellen (Freigaben, Sperren, Schwerpunkt) — in Lauf 6 stand
  sie noch auf Lauf 5
- Draft-PR für Lauf 8 anlegen, Review-Workflows-Zustand notieren

---

## Was du dabei nicht tun sollst

- **Kein Merge, kein Deploy in dieser Session.** Lauf 7 ist durch; hier wird nur geplant.
- **Keine neue `config.RequireX`-Assertion und kein Scharfschalten von `modules.*`** — `COSMI_ENV=
  production` ist live und CD deployt automatisch.
- **Die Security Review nicht als Gate einplanen.** Sie meldet auf Nachtlauf-großen PRs `success`
  mit 0 Findings, obwohl sie den Diff nie gelesen hat (HTTP 406, belegt für PR #19 und #20).
  Ersatz: den Produktions-Diff separat reviewen —
  `git diff origin/main...HEAD -- 'backend/**' ':(exclude)backend/**/*_test.go'` war bei Lauf 7
  686 Zeilen und damit vollständig lesbar.
- **Keine AI-Attribution** in Commits. Conventional Commits, englisch, imperativ. Deutsch für
  Journal und Notizen.
