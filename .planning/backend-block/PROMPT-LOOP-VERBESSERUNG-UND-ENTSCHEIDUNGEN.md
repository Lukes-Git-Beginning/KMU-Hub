# Nachtlauf-Mechanik verbessern und den Entscheidungsstau auflösen

> **Auftrag dieser Sitzung:** zwei Dinge, in dieser Reihenfolge. Erst **sechs Entscheidungen**
> treffen, die der Loop seit Lauf 11 nicht selbst treffen darf. Dann **fünf belegte Defekte der
> Loop-Mechanik** beheben. Kein Backlog für Lauf 12 ausschreiben — das ist die Sitzung danach,
> und sie wird erheblich besser, wenn diese hier vorher gelaufen ist.
>
> Ausgangspunkt: Lauf 11 ist als `48246830` gemergt und deployt (CI fünf Jobs grün, CD grün,
> `/health` meldet `48246830`, Prod-Migrationskopf 323). `main` = `backend-loop`.

---

## Teil A — Sechs Entscheidungen

Alle sechs sind am Code verifiziert, nicht aus Notizen übernommen. Jede nennt Sachstand, Optionen
und eine Empfehlung. Die Empfehlung ist ein Vorschlag, kein Beschluss.

### A1 · Vier CRM-Endpunkte antworten dauerhaft 501

`fix-crm-tag-endpoints-are-501-stubs` + `feat-crm-activity-deal-tag-rpcs` (BACKLOG.yml, beide blocked)

Tagging von Activities und Deals ist **end-to-end gebaut außer einer Schicht**: Join-Tabellen
`activity_tags`/`deal_tags` (Migration 000006, `tenant_id` seit 000111), Repository-Methoden,
`activity.Service.AddTags/RemoveTags` (`activity/service.go:441/466`),
`deal.Service.AddTags/RemoveTags` (`deal/service.go:458/483`) inkl. Tag-Existenzprüfung, die
Router-Registrierung mit `RequirePermission` (`route_crm.go:165/166/178/179`) und der vollständige
OpenAPI-Vertrag (`openapi.yaml:2731` / `:2964`). Es fehlt **allein der gRPC-Hop** —
`proto/crm/v1/crm.proto` hat Tag-RPCs nur für Contacts. Beide Service-Methoden haben null
Nicht-Test-Aufrufer, sind also toter Code.

- **Weg A:** vier RPCs anlegen, durchreichen. Mechanische Arbeit, der Wert liegt schon da.
- **Weg B:** vier spezifizierte, fachlich fertige Endpunkte löschen.

**Vorher prüfen, das entscheidet es:** ruft das Desktop-Frontend diese vier Endpunkte überhaupt?
`grep -rn "tags" desktop/src/renderer/src/api/hooks/` gegen die Deal- und Activity-Hooks. Ruft es
sie, ist Weg A zwingend — das Frontend läuft dann heute gegen 501. Ruft es sie nicht, ist Weg B die
ehrlichere Antwort. **Empfehlung: Weg A**, falls die Prüfung nicht eindeutig „wird nirgends
gebraucht" ergibt — ein 501 im ausgelieferten Produkt kostet mehr als vier Durchreich-RPCs.

### A2 · HR-Idempotency ist eine Zusage ohne Wirkung

`fix-hr-manual-entry-idempotency-key-not-enforced` (BACKLOG.yml, blocked)

`HandleCreateManualEntry` (`route_hr.go:1621-1625`) verlangt zwingend einen `Idempotency-Key`
(400 ohne) und reicht ihn bis `timetracking.ManualEntryInput.IdempotencyKey` (`repository.go:184`)
durch — **dort endet die Kette**. `Service.CreateManualEntry` (`service.go:485-537`) liest das Feld
nie, es gibt keine `idempotency_key`-Spalte auf `hr_work_time_entries`, keinen Uniqueness-Check.
Ein Client, der nach Netzwerk-Timeout denselben Request wiederholt, bekommt **zwei bezahlte
Arbeitszeiteinträge**. Schlimmer als kein Header, weil der Header Sicherheit suggeriert.

- **Weg A:** generischer `internal/idempotency`-Store auf Middleware-Ebene (Vorlage: die
  Finance-Routen, 87 % Coverage, echter SQL-Test).
- **Weg B:** eigene `idempotency_key`-Spalte plus Unique-Index auf `hr_work_time_entries`.

**Vorher prüfen:** läuft `POST /api/v1/hr/time/entries` durch die Idempotency-Middleware? Wenn ja,
ist Weg A ein Registrierungs-Einzeiler. **Empfehlung: Weg A**, wenn die Route durch die Middleware
geht — sonst Weg B.

### A3 · Lexware-Webhook kann beim zweiten Tenant falsch zuordnen

`harden-lexware-webhook-organization-id-scoping` (BACKLOG.yml, blocked)

Der Scheduler-Pfad ist in Lauf 11 gefixt. Der **Webhook-Pfad** (`lexware/service.go:262-280`)
bleibt bewusst offen: er läuft unter `sysctx`, ruft `GetByPlatform(ctx, "lexware")` ohne
Tenant-Filter, und bei mehr als einem aktiven Lexware-Tenant kann das die Config eines fremden
Tenants liefern. Der Webhook trägt keine Tenant-Info, und die `organization_id`, die er optional
mitbringt (`route_lexware.go:579-592`), wird beim Verbinden **nirgends gespeichert** —
`Service.Connect` legt nur `{"auth_type": "api_key"}` als Metadata an.

**Heute ist genau ein Tenant aktiv, der Schaden ist also nicht real.** Der Code trägt an der Stelle
einen `lean:`-Marker mit Upgrade-Trigger.

**Empfehlung: den teuren Fix nicht jetzt bauen, aber die billige Vorbereitung mitnehmen** —
`organization_id` beim Connect in die Metadata schreiben (additiv, wenige Zeilen, keine Migration).
Ohne diesen Wert ist der Fix später gar nicht möglich, und der zweite Lexware-Kunde kommt ohne
Vorwarnung.

### A4 · Bexio bleibt gesperrt — oder nicht

`fix-bexio-config-lookup-cross-tenant-under-sysctx` (BACKLOG.yml, blocked)

Derselbe Root-Cause wie A3, hier bei Bexio: `PostgresIntegrationConfigRepo.GetByPlatform`
(`bexio/postgres_config_repo.go:27-31`) filtert nicht nach `tenant_id`; `SyncContacts` und
`PollPayments` lösen die Config je Tick unter Systemkontext neu auf. Der Bug ist im Code selbst als
„G8" dokumentiert und nur für `PullInvoicesWithConfig` umgangen. Bexio ist laut Lagebild G3
produktiv aus und als Schweizer Software außerhalb der DE-Fokussierung.

**Empfehlung: gesperrt lassen, aber die Unit nach `BACKLOG-PARKED.yml` verschieben.** Sie blockiert
sonst in jedem Lauf erneut — genau das Muster, das unter B1 Geld gekostet hat.

### A5 · Zwei Automations-Trigger, die nie feuern

`wire-biz-event-emitters-for-finance-triggers` (BACKLOG-NEXT.yml, blocked)

In Lauf 11 verifiziert: Das Automations-Modul bietet in der UI „Rechnung versendet"
(`biz.invoice.sent`) und „Angebot erstellt" (`biz.quote.created`) als wählbare Auslöser an
(`trigger/registry.go:182-218`). Beide hängen an `EmitBizEvent`, das nur feuert, wenn
`SetEventEmitter` gesetzt wurde. **`cmd/biz/main.go` ruft `SetEventEmitter` an keiner Stelle auf**
(0 Grep-Treffer) — invoice und quote laufen mit `nil` (No-op), lexware mit `noopEmitter{}`.

Ein Kunde kann eine Automation bauen, die nie ausgelöst wird, und bekommt keinerlei Hinweis.

- **Weg A:** `SetEventEmitter` in `cmd/biz/main.go` verdrahten. Wenige Zeilen.
- **Weg B:** beide Trigger aus der Registry nehmen, bis sie funktionieren.

**Empfehlung: Weg A.** Beide Trigger sind fachlich sinnvoll und der Emitter existiert. Danach ein
Test, der belegt, dass ein versendeter Beleg wirklich ein Event erzeugt — sonst ist es dieselbe
stille Zusage nur eine Ebene tiefer.

### A6 · Aufbewahrungsfristen für sechs Domänen mit Personenbezug

`decide-retention-policy-for-unmapped-personal-data-domains` (BACKLOG-NEXT.yml, blocked)

Neun von 24 Services haben inzwischen einen Retention-Handler (Einladungen kamen in Lauf 11 dazu).
Für sechs weitere Domänen mit belegtem Personenbezug fehlt die Entscheidung: HR-Personaldaten
(Profile, Urlaub, Dokumente), Fuhrpark (Führerscheine, Fahrtenbuch, Fahrzeugzuordnung) und vier
weitere. **Keine davon ist ein mechanischer Fall** wie die drei bereits gebauten Handler.

HR ist der harte Teil: Lohnunterlagen bis zu zehn Jahre nach § 147 AO / § 257 HGB, reine
Personalakte kürzer nach Verjährung möglicher Ansprüche. Eine pauschale Frist wäre falsch, eine
differenzierte braucht Rechtsberatung.

**Empfehlung: aufteilen.** Die Domänen ohne gesetzliche Frist jetzt entscheiden und als Units
anlegen; HR und alles mit AO/HGB-Bezug an die Legal-Etappe hängen (Etappe 4, wo AVV/DPA ohnehin
liegen). Nicht die ganze Unit blockieren, weil ein Teil davon einen Anwalt braucht.

---

## Teil B — Fünf Defekte der Loop-Mechanik

Alle an Lauf 11 gemessen. Zahlen aus `JOURNAL.md`, `logs/run.log` und `run-loop.ps1`.

### B1 · Die Modellwahl hängt an der falschen Unit — belegt teuer

`Get-NextUnitModel` (`run-loop.ps1:306-316`) nimmt das `model:` der **ersten `status: todo`-Unit
im Backlog**. Der Agent zieht aber nicht zwingend diese Unit — überspringt er sie, weil sie auf
eine Entscheidung wartet, läuft die tatsächlich gebaute Unit trotzdem auf deren Modell.

Genau das ist passiert: `harden-lexware-webhook-organization-id-scoping` (`model: opus`) stand ab
Iteration 86 als `todo` am Backlog-Kopf und wurde laut Journal „in jeder Iteration übersprungen".
Die Iterationen **88, 89, 91, 92, 93, 96, 97, 99, 100** liefen auf **opus** — gebaut wurden dort
`fix-idempotency-409-rollout-non-finance-routes-2/3/4`, `doc-status-code-systemic-400-503-sweep`
und `fix-status-code-drift-baseline-non-systemic`, **allesamt als `model: sonnet` ausgewiesen**.
Neun Iterationen Opus für das Einfügen von YAML-Zeilen in `openapi.yaml`.

**Vorschlag:** Der Agent gibt die tatsächlich gezogene Unit-ID am Iterationsende in einer festen
Zeile aus, und der Treiber wählt das Modell der nächsten Iteration daraus. Oder einfacher:
`status: todo`-Units, die eine Entscheidung brauchen, kommen gar nicht erst ins Backlog (B4) —
dann kann der Kopf nicht klemmen.

### B2 · Drei Doku-Ketten haben den Laufausklang aufgefressen

Von den letzten 29 Iterationen (92–120) gingen **27** an drei sich selbst nachlegende Ketten:
`fix-idempotency-409-rollout-non-finance-routes-2…12`,
`doc-status-code-systemic-400-503-sweep…-10` und
`fix-status-code-drift-baseline-non-systemic…-6`. Jede Iteration schließt **eine**
Registrar-Gruppe — vier bis zehn YAML-Zeilen — und legt die Nachfolge-Unit an. Bei
**7,4 min/Iteration** (885 min für 120) sind das rund 3,3 Stunden für Spec-Zeilen.

Die Restliste in `doc-status-code-systemic-400-503-sweep-11` nennt noch **22 Gruppen mit über 600
Operationen**. Bei diesem Zuschnitt wären das mehr als zwanzig weitere Nachtstunden.

**Vorschlag:** Diese Arbeit ist kein Loop-Fall. Entweder eine einzige Batch-Unit („alle
verbleibenden Registrar-Gruppen in einer Iteration, Gate am Drift-Zähler") oder — besser — ein
Skript, das die Drift-Liste einliest und die `$ref`-Zeilen einfügt, mit `swagger-cli validate` als
Gate. Der Loop soll Bugs finden, nicht YAML tippen.

### B3 · Die Backlog-Dateien werden nie geparst

Der Treiber zählt offene Units per Zeilen-Regex. Nach Lauf 11 waren **beide** Backlog-Dateien mit
`yaml.safe_load` nicht ladbar — 13 Stellen, alle nach demselben Muster
(`- Ein Test belegt: …` über zwei Zeilen). Der Fehler stand ab Iteration 114 in jedem
Journal-Eintrag als Randnotiz und wurde nie behoben; der Lauf lief trotzdem 120 Iterationen durch.
Inzwischen gefixt (`90d48d79`, als `- >-` Block-Scalar), aber der blinde Fleck bleibt.

**Vorschlag:** `yaml.safe_load` gegen beide Dateien in den Vorflug von `run-loop.ps1`, direkt neben
den DB-Check — mit Abbruch, nicht mit Warnung. Kostet eine Sekunde.

### B4 · Entscheidungs-Units stehen im falschen Backlog

Regel 6 der Lauf-11-Vorbereitung sagt wörtlich: *„In `BACKLOG.yml` steht keine `blocked`-Unit."*
Am Laufende standen dort **fünf**. Jede hat mindestens eine Iteration gekostet, die sie liest und
weiterzieht, und eine davon hat zusätzlich B1 ausgelöst.

**Vorschlag:** Der Vorflug prüft es. Findet er im Backlog eine `blocked`-Unit oder eine, deren
`done_when[0]` mit „Luke hat" beginnt, bricht er ab. Entscheidungsbedürftiges gehört nach
`BACKLOG-PARKED.yml` oder `BACKLOG-NEXT.yml` — und damit in eine Sitzung wie diese hier.

### B5 · Der Lauf misst seine eigene Substanz nicht

70 von 120 Journal-Einträgen haben ein nicht-leeres `offen:`; 21 davon nennen „Luke", 10
„Entscheidung". Die `coverage:`-Zeile meldet in der zweiten Laufhälfte fast durchgehend
„unverändert", weil Doku-Units keine Coverage bewegen. Am Ende einer Nacht gibt es **keine Zahl**,
die sagt, wie viel Substanz geliefert wurde — diese Merge-Sitzung musste das aus den
Commit-Präfixen rekonstruieren (49 fix, 39 cov, 10 scan, 10 doc, 7 feat, 4 verify).

**Vorschlag:** Der Treiber schreibt am Laufende eine kurze Bilanz ins Journal: Iterationen, Units
nach Präfix, Commits nach Typ, Coverage-Delta der berührten Pakete, Anzahl `offen:`-Einträge mit
Entscheidungsbedarf. Das ist die Grundlage, um den nächsten Zuschnitt zu steuern statt ihn zu
schätzen.

---

## Reihenfolge und Ergebnis

1. **Teil A abarbeiten**, A1 und A2 mit der genannten Vorabprüfung am Code. Jede Entscheidung
   landet als `decided:`-Feld in der jeweiligen Unit, mit Datum und Begründung — nicht nur im Chat.
2. Entschiedene Units auf `status: todo` setzen und nach `BACKLOG.yml` bewegen; verworfene mit
   Begründung nach `BACKLOG-PARKED.yml`.
3. **Teil B umsetzen**, B3 und B4 zuerst (beide sind kleine Vorflug-Ergänzungen mit sofortiger
   Wirkung), dann B1, dann B5. B2 ist eine Zuschnitt-Entscheidung, keine Code-Änderung.
4. Am Ende: `yaml.safe_load` gegen beide Backlog-Dateien, `hooks/test-loop-guard.sh` grün, ein
   Commit, Push.

**Was diese Sitzung nicht macht:** den Backlog für Lauf 12 ausschreiben. Das ist die nächste
Sitzung, und sie startet dann mit entschiedenen Units und einem Treiber, der nicht mehr am
Backlog-Kopf klemmt.

---

## Belege

- `.planning/backend-block/loop/JOURNAL.md` — 120 Iterationen, Laufkontext im Kopf
- `.planning/backend-block/loop/BACKLOG.yml` — 121 done / 5 blocked / 3 todo
- `.planning/backend-block/loop/BACKLOG-NEXT.yml` — A5 und A6 liegen dort
- `.planning/backend-block/loop/run-loop.ps1:306-316` — die Modellwahl aus B1
- `.planning/backend-block/loop/logs/run.log` — Modell je Iteration
- `.planning/backend-block/PROMPT-VORBEREITUNG-LAUF-11.md` — die Regeln, gegen die Lauf 11 lief
- Merge-Commit `48246830` — die Verifikation vor dem Merge steht vollständig in der Commit-Message
