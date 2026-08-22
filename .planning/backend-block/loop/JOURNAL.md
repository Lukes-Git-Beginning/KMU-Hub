# Backend-Nachtloop — Journal Lauf 11

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93, inkl. `logs/`).

---

## Laufkontext

**Noch nicht geschrieben — Lauf 11 ist nicht vorbereitet.** Dieser Abschnitt entsteht beim
Ausschreiben des Backlogs und hält fest, was zum Startzeitpunkt gemessen wurde: Ausgangs-Commit,
CI-/CD-Stand, Migrationskopf, Coverage-Ausgangslage, Umfang und Blöcke.

Was aus Lauf 10 hierher gehört, sobald der Kontext geschrieben wird:

- **Ausgangspunkt:** Lauf 10 gemergt als `f87ffdcf`, CI grün auf `778a2e44` (Run 32570176303,
  alle fünf Jobs inkl. E2E). Migrationskopf **322**.
- **Das CI-Signal des Laufs war nicht dekorativ:** es hat einen Advisory-Lock-Leak gefunden, den
  `idempotency.CleanupWithLock` und der neue `gdpr`-Retention-Scheduler teilten (Lock über den
  Pool genommen, über den Pool freigegeben — das Unlock landet auf einer fremden Verbindung).
  Der Scheduler wäre nach dem ersten Tick dauerhaft eingeschlafen, ohne Fehlermeldung. Behoben
  in `778a2e44`. **Lehre für jeden Lauf: ein Test, der lokal grün ist, weil der Pool nur eine
  warme Verbindung hatte, beweist nichts.**
- **`-race` lief in Lauf 10 lokal nie** (kein `gcc` im PATH dieser Maschine). Wo eine Unit
  `-race` verlangt, ist CI der einzige Beweis — das gehört in die `offen:`-Zeile.
- **Lokales Postgres-Verbindungslimit:** `go test` über mehrere DB-Pakete mit vollem
  Parallelismus reißt mit `53300 remaining connection slots are reserved` ab. `-p 1` verwenden
  oder das Paket-Set eingrenzen. Kein Code-Fehler.

---
