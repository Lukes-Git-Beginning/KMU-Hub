# Backend-Nachtloop — Journal

Append-only. Jede Iteration haengt genau einen Block an. Von unten lesen.

Format:

```markdown
## Iteration <n> — <unit-id> — <done|blocked> — <YYYY-MM-DD HH:MM>
- commit: <sha oder ->
- gebaut: <ein bis drei Zeilen, was real existiert>
- gate: build ok | vet ok | lint ok | test ok | migration ok | rls-smoke ok|n.a.
- verify vorgaenger: <sauber | Befund + angelegte Fix-Unit>
- offen: <was Luke morgens pruefen muss>
```

---

## Iteration 0 — Harness aufgesetzt — 2026-07-26

- commit: -
- gebaut: Loop-Harness (Guard-Hook + Regressionstest, ITERATION.md, BACKLOG.yml mit
  22 Phase-3-Units, run-loop.ps1, GATE-COMMANDS.md).
- gate: Guard-Regressionstest 35/35 gruen. Lokale DB auf Migrationskopf 243
  (= Repo-Kopf). RLS-Smoke gegen `contacts` verifiziert: eigener Tenant 12 Zeilen,
  fremder Tenant 0.
- verify vorgaenger: n.a. (erster Eintrag)
- offen: Trockenlauf ueber zwei Iterationen unter Aufsicht, bevor ein Nachtlauf startet.

## Iteration 1 — p3-einkauf-total-amount — done — 2026-07-26

- commit: e91cdf2a
- gebaut: Repository-Methode `RecomputePOTotal` (SQL-Aggregat SUM(quantity*unit_price)
  in purchase_orders.total_amount, tenant-gescoped) + Aufruf aus AddPOLine/UpdatePOLine/
  DeletePOLine im Service. CreatePO bleibt bei "0" (korrekt, da 0 Zeilen bei Anlage).
  4 neue Unit-Tests (Add/Add-mehrzeilig/Update/Delete) pruefen den total_amount-Wert
  nach der jeweiligen Mutation.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. (keine neue
  Tabelle/Spalte) | rls-smoke n.a. (kein neuer SELECT-Pfad, bestehende Policy greift
  unveraendert)
- verify vorgaenger: n.a. (Iteration 0 war reines Harness-Setup, kein Unit-Commit)
- offen: keins. Naechste Unit ist p3-einkauf-cancel (deps erfuellt).

## Iteration 2 — p3-einkauf-cancel — done — 2026-07-26

- commit: 22883d50
- gebaut: `Service.CancelPO` (analog zu SubmitPO/ReceiveGoods) storniert eine PO nur aus
  den Status draft/submitted/sent (POStatusCancelled existierte bereits im Enum),
  alles andere liefert `ErrPONotCancellable`. Neue RPC `CancelPO` im .proto (Request
  tenant_id+po_id, Response wiederverwendet POResponse{po}) + Regen von .pb.go/_grpc.pb.go.
  gRPC-Server-Handler + Fehler-Mapping (FailedPrecondition -> 409) in einkauf_grpc.go.
  Gateway-Route `POST /pos/{id}/cancel` via bestehende Permission `einkauf:po write`
  (kein neuer Guard, kein Seed noetig) — Handler geht ueber `client.CancelPO`, keine
  direkte Service-Instanz im Gateway. 3 neue Unit-Tests (Erfolg ueber alle drei offenen
  Stati, Ablehnung ueber alle vier nicht-offenen Stati, PO-not-found).
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. (keine neue
  Tabelle/Policy, bestehende purchase_orders-Policy unveraendert) | rls-smoke n.a.
  (kein neuer SELECT-Pfad)
- verify vorgaenger: sauber (e91cdf2a gegen alle 6 Fehlerklassen geprueft — Business-Logik
  korrekt in Service/Repository, tenant-gescopte SQL, kein gRPC-Bypass, kein Stub, kein
  Proto-Touch, kein neuer Guard, keine neue Tabelle).
- offen: FE-Client (`desktop/src/renderer/src/api/einkauf-client.ts`) erwartet bereits
  genau diese Route/Response-Form (mock-first) — kein FE-Aenderungsbedarf. Der
  `🔒 Backend-Gap`-Kommentar im FE-Client und im MSW-Handler (`mocks/handlers/einkauf.ts`)
  kann jetzt entfernt werden, ist aber nicht Teil dieser Iteration (FE-Datei, nicht
  Backend-Scope). DB-Gate (RLS-Smoke) nicht gelaufen, da keine Tabelle/Policy angefasst —
  bewusst n.a., nicht uebersprungen.
