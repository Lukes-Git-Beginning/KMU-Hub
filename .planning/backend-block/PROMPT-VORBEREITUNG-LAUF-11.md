# Vorbereitung Backend-Nachtlauf 11

> **Auftrag dieser Sitzung:** den Backlog für Lauf 11 **ausschreiben**. Nicht starten, nicht bauen.
> Am Ende steht `BACKLOG.yml` gefüllt, `JOURNAL.md` mit geschriebenem Laufkontext, und das
> Startkommando ist abgenommen.

---

## 1 · Ausgangslage (Stand 2026-08-22, alles gemessen, nicht aus Notizen übernommen)

- `main` = `backend-loop` = **`a859974c`**. Beide Branches identisch, `backend-loop` per
  Fast-Forward nachgezogen (nicht rebased).
- **Migrationskopf 322**, Repo = lokale DB = Produktion, `dirty=false`. Nächste freie Nummer wäre
  323 — trotzdem zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E '^[0-9]{6}' | sort | tail -1`
- **CI grün** auf `a859974c` (Rerun 32571951649, alle fünf Jobs inkl. E2E), **CD grün**
  (32572607117), Produktion antwortet mit `commit: a859974c`.
- **Coverage gesamt 62,7 %** bei Gate 15 % (CI-Artefakt Run 32570176303). Schwächste Kernpakete
  nach ungedeckten Zeilen: `internal/gateway` 54,1 % (10.448), `internal/server` 70,5 % (6.023),
  `internal/biz` 63,7 % (3.190), `internal/work` 61,2 % (1.876).
- **Lokale DB ist Startbedingung**, der Treiber prüft sie im Vorflug. Rolle `kmuhub_app`
  (NOSUPERUSER NOBYPASSRLS), niemals `kmuhub`.
- **Backlog ist leer** (`units: []`) — das ist Absicht, damit kein unvorbereiteter Lauf startet.
  Kandidaten und Befunde stehen in `loop/BACKLOG-NEXT.yml`.

### Was Lauf 10 verändert hat, das die Planung betrifft

**Etappe 3 / G1 ist backend-seitig praktisch abgeräumt.** Fünf Lagebild-Befunde über zusammen
9,1 PT sind erledigt: Art.-15-Auskunft (jetzt 29 Module inkl. Rechnungen und Meetings),
Retention-Engine (läuft in Prod, `dry_run`), `AnonymizeContact`, GoBD-WORM (Migration 315),
`/health` mit Postgres-Checker. **Was von G1 übrig ist, kann der Loop nicht anfassen** — Consent-
Check und Kontakt-Löschung brauchen Frontend, `restore.sh`/`rollback.sh`/Backup-Alerts liegen in
`deploy/`, RUNBOOK und Secret-Rotation sind Ops. Ein Nachtlauf bringt Etappe 3 nicht weiter.

**Zwei G2-Befunde sind ebenfalls schon gefallen:** `internal/idempotency` stand mit „real 0 %
Coverage, Tests gegen einen Mock" im Lagebild und liegt jetzt bei 87 % — und genau dieser erste
echte SQL-Test hat den Advisory-Lock-Leak gefunden. „22 Route-Dateien ohne Test, 8 davon in
Geld-/Compliance-Pfaden" ist auf drei kleine Dateien geschrumpft.

---

## 2 · Zuschnitt — entschieden, nicht mehr zu diskutieren

**Roter Faden: „Geld- und Compliance-Pfade vor dem ersten zahlenden Kunden" (G2).** Entscheidung
Luke, 2026-08-22. Die Blockstruktur folgt Lauf 10, weil sie funktioniert hat: Substanz zuerst,
dann die schwache Fläche, Scans am Schluss.

### Block A — G2-Substanz

- **Lexware-Webhook ohne Doppelzustellungs-Schutz.** ⚠ Der Lagebild-Text ist ungenau, am Code
  geprüft: Eine Doppelzustellung erzeugt **keine** doppelten Kontakte —
  `SyncContactByLexwareID` holt per Resource-ID und upsertet. Der reale Schaden ist ein anderer:
  `HandleEvent` (`webhook_handler.go:167`) feuert `lexware.webhook.received` in den
  Event-Emitter, und darauf hängt der Inbound-Automation-Trigger — **ein Kundenworkflow läuft
  doppelt**. Dazu ein überflüssiger Call gegen ein rate-limitiertes Fremdsystem und ein
  doppelter Sync-Log-Eintrag. ⚠ `LexwareWebhookEvent` (`models.go:186`) trägt **keine
  Event-ID**, nur `eventType`/`resourceId`/`organizationId` — das Dedup-Verfahren (Hash plus
  Zeitfenster? HTTP-Header? Tabelle?) gehört in die Unit entschieden, nicht in die Iteration.
- **EN-16931-Validierung ist Feldprüfung ohne Schematron** (Lagebild 3 PT). `grep -ril schematron
  internal/biz/` → 0 Treffer, weiter offen. Größter Brocken des Blocks; realistisch 3–4 Units.
  Vor dem Ausschreiben klären, ob eine Go-Schematron-Bibliothek ohne neue schwere Dependency
  in Frage kommt — sonst wird es eine Regelmenge in Go, und das ist ein anderer Zuschnitt.
- **`fix-booking-page-orphaned-after-owner-erasure`** — entblockt. Entscheidung: **deaktivieren**
  (`active = false`), und `PreviewErasure` weist die Seite als eigenen Posten aus. Die Unit liegt
  vollständig ausgeschrieben in `loop/archive/lauf-10/BACKLOG.yml`; von dort übernehmen,
  `blocked_reason` durch die Entscheidung ersetzen, `status: todo`.
- **Die Backend-Anteile der sieben offenen Journal-Befunde** aus `BACKLOG-NEXT.yml`. Die zwei
  Frontend-Nachzüge gehören **nicht** hierher (Loop hat kein Playwright-Gate).

### Block B — Geld-Repositories gegen echtes SQL

Das ist der eigentliche Hebel. Rund **2.800 Zeilen SQL auf Geldpfaden ohne eine einzige
Testdatei**:

| Datei | Zeilen |
|---|---:|
| `internal/biz/invoice/postgres_repository.go` | 881 |
| `internal/biz/creditnote/postgres_repository.go` | 517 |
| `internal/biz/invoice/postgres_document_chains.go` | 352 |
| `internal/biz/dunning/postgres_repository.go` | 327 |
| `internal/biz/payment/postgres_repository.go` | 220 |
| `internal/biz/invoice/service_gobd.go` | 201 |
| `internal/biz/dunning/service_gobd.go` | 180 |
| `internal/biz/invoice/postgres_transactions.go` | 135 |

Paket-Coverage dazu: `biz/creditnote` 28,2 % · `biz/quote` 33,3 % · `biz/invoice` 34,8 % ·
`biz/payment` 46,4 % · `biz/dunning` 61,8 %.

**Nicht als Coverage-Übung schneiden, sondern als Bug-Suche** — Coverage ist das Nebenprodukt.
Vorbild sind die beiden `idempotency`-Units aus Lauf 10 (`6507e475`, `254120eb`): echtes SQL,
Nebenläufigkeit, Grenzfälle. Die erste davon hat sofort einen Produktionsfehler gefunden.
`biz/bexio` (54,1 %, 574 ungedeckt) trotz schlechter Zahl **weglassen** — laut Lagebild G3
produktiv aus und als Schweizer Software ohnehin außerhalb der DE-Fokussierung.

### Block C — Muster-Scans (am Schluss, legen zur Laufzeit nach)

- Doppelzustellungs-/Idempotenz-Schutz über **alle** Eingangspfade, nicht nur Lexware.
- OpenAPI-Antwortcode-Drift über die **52 Routen-Dateien, die der Scan in Lauf 10 bewusst
  ausgelassen hat** (Journal Lauf 10, Iteration 46 nennt, was tief geprüft, was nur gegrept und
  was gar nicht angesehen wurde).
- Retention-Mapping für die übrigen Services — registriert sind acht Handler; der C4-Scan aus
  Lauf 10 hat die legitimen Nicht-Zuordnungen bereits begründet, die offene Frage ist
  `advisory_protocols` (Verdacht auf Dokumentationspflicht, **nicht belegt**).
- Geld-Rundung und Steuerlogik: `biz/tax` steht bei 100 %, aber wer ruft es wie auf?

---

## 3 · Umfang — Zielkorridor mit Rechnung

Lauf 10: 00:18–12:43 = **745 min für 92 Iterationen ≈ 8,1 min/Iteration**.

Fenster für Lauf 11: **23:00 → 14:00 = 900 min ≈ 111 Iterationen** (bis 13:00 wären es ~104).

Lauf 10 startete mit 48 Units und endete bei 93 — die neun Scans haben 45 nachgelegt. Bei gleichem
Verhältnis genügten ~57 Start-Units. **Trotzdem 65–75 ausschreiben, davon 8–10 Scans.** Die
Asymmetrie ist Absicht: Ein Überhang bleibt für Lauf 12 liegen und kostet nichts, ein Mangel
beendet den Lauf vorzeitig und verschenkt Stunden. Der Scan-Nachschub ist keine verlässliche
Größe — er hing in Lauf 10 an einzelnen ergiebigen Scans.

⚠ **`-MaxIterations` hochsetzen.** Der Default ist **100**; Lauf 10 endete bei 92/100 und wäre
sonst gedeckelt worden. Für dieses Fenster mindestens **120**.

---

## 4 · Regeln beim Ausschreiben (teuer gelernt, nicht optional)

1. **Jede Schema- und Codebehauptung vorher am Code belegen.** In der Lauf-10-Vorbereitung hat die
   Gegenprüfung einen von sechs Befunden widerlegt — der angeblich fehlende Foreign Key auf
   `contact_tags` existiert seit Migration 000007. Die Unit hätte eine Migration gegen ein
   nicht existierendes Problem gebaut. Prüfen kostet eine Minute.
2. **Units werden aus `BACKLOG-NEXT.yml` nach `BACKLOG.yml` verschoben, nicht kopiert.**
3. **Ein Kopf, der einen Unit-Anhang behauptet, den es nicht gibt, ist kein Backlog.** Genau das
   stand vor Lauf 10 in der Datei — wäre der Lauf so gestartet, hätte der Treiber nach null
   Iterationen beendet.
4. **Treiber-Parsing:** `model:` muss **vor** `status:` stehen, hinter `status: todo` darf kein
   Kommentar stehen.
5. **`coverage_start` gegen den aktuellen CI-Stand nachziehen** (Run 32570176303), nicht die alten
   Werte aus Lauf 10 übernehmen — Lauf 10 hat das Gateway um 8 Punkte bewegt.
6. **In `BACKLOG.yml` steht keine `blocked`-Unit.** Blockiertes gehört mit `blocked_reason` in
   `BACKLOG-PARKED.yml` oder bleibt in `BACKLOG-NEXT.yml`.
7. **Gesperrt** (unverändert): Frontend/Desktop, CSAT und Public-Token-Routen, Dependency-Bumps,
   `deploy/`, Migrations-Umnummerierungen, Preise und Modul-Zuschnitt.

### Was Lauf 10 über das Testen gelernt hat — in den Laufkontext übernehmen

- **Ein DB-Test, der lokal grün ist, weil der Pool nur eine warme Verbindung hatte, beweist
  nichts.** Wer eine Ressource *pro Verbindung* prüft (Advisory Locks, Session-GUCs, temporäre
  Tabellen), hält vorher eine zweite Verbindung fest. Advisory Locks sind innerhalb einer Session
  re-entrant — ein Verifier auf derselben Connection meldet fälschlich „frei".
- **Wer ein bestehendes Muster als Vorlage kopiert, kopiert seine Fehler mit.** So kam der
  Advisory-Lock-Leak aus `idempotency` in den neuen Retention-Scheduler. Vorlage vorher prüfen.
- **`-race` läuft auf dieser Maschine nicht** (kein `gcc` im PATH). Units, deren `done_when`
  `-race` verlangt, können lokal nicht abgeschlossen werden — gehört in die `offen:`-Zeile, CI ist
  der Beweis.
- **`go test` über mehrere DB-Pakete mit vollem Parallelismus** reißt mit
  `53300 remaining connection slots are reserved` ab. `-p 1` verwenden. Kein Code-Fehler.
- Die lokale Dev-Postgres trägt **13.8k Müll-Tenants** aus alten Läufen (Prod hat 1). Jeder Test,
  der über `tenants` iteriert, ist lokal unbrauchbar langsam.

---

## 5 · Startkommando (nach Abnahme des Backlogs)

```powershell
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 `
  -StartNotBefore "23:00" -UntilTime "14:00" -MaxIterations 120
```

Vorflug, bevor das läuft:

- `hooks/test-loop-guard.sh` grün.
- Lokale DB an, Migrationskopf = Repo-Kopf, Anmeldung als `kmuhub_app` möglich.
- `backend-loop` auf `origin/main` **gemergt** (nicht rebased) — steht aktuell schon deckungsgleich.
- Der Treiber setzt die Schlafsperre selbst (seit `e3b1afca`).

**Nach dem Lauf**, nicht vorher: Draft-PR anlegen. Vor dem Merge geht das nicht — `backend-loop`
ist dann identisch mit `main` und `gh pr create` lehnt einen PR ohne Diff ab (die README behauptete
bis `a859974c` das Gegenteil).

```bash
gh workflow disable "Claude PR Review"; gh workflow disable "Security Review"
gh pr create --base main --head backend-loop --draft --title "Backend-Nachtlauf 11"
gh workflow enable "Claude PR Review"; gh workflow enable "Security Review"
```

---

## 6 · Weiterführend

- `loop/BACKLOG-NEXT.yml` — Kandidaten, die sieben offenen Journal-Befunde, gefallene und offene
  Entscheidungen
- `loop/archive/lauf-10/` — Journal und Backlog des letzten Laufs (die Buchungsseiten-Unit liegt
  dort ausgeschrieben)
- `loop/README.md` — Sicherheitsmodell, Guard, Dateien
- `loop/GATE-COMMANDS.md` — verifizierte Gate-Kommandos, Coverage-Definition, RLS-Smoke
- `loop/ITERATION.md` — der konstante Prompt jeder Iteration
- `.planning/launch-lagebild-2026-08-12.md` §4 — G0/G1/G2/G3-Befunde
