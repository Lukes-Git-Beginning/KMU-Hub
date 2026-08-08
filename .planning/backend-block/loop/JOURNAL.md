# Backend-Nachtloop — Journal (Lauf 6)

Append-only. Eine Iteration = ein Eintrag. **Immer ans Dateiende anhaengen, nie vor einen
bestehenden Eintrag einsortieren** — der Treiber leitet die Fortschrittsanzeige aus der hoechsten
Iterationsnummer ab, und ein eingeschobener Eintrag hat in Lauf 3 zwei Iterationen lang denselben
Stand gemeldet.

Vorlage:

```markdown
## Iteration <n> — <unit-id> — <done|blocked> — <YYYY-MM-DD HH:MM>
- commit: <sha oder ->
- gebaut: <ein bis drei Zeilen, was real existiert>
- gate: build ok | vet ok | lint ok | test ok | migration ok | rls-smoke ok|n.a.
- verify vorgaenger: <sauber | Befund + angelegte Fix-Unit>
- offen: <was Luke morgens pruefen muss — DB-Gate, Proto-Regen, Route-Registrierung, Annahmen>
```

Bei Coverage-Units (Bloecke C und B) gehoert zusaetzlich in den Eintrag:

```markdown
- mutations-probe: <welche Zeile gebrochen wurde, ob der Test rot wurde, zurueckgedreht ja/nein>
- db-tests: <Zahl der real gelaufenen DB-Tests, Zahl der Skips — bei Block C muss Skips = 0 sein>
```

Journale der Vorlaeufe: `archive/lauf-3/JOURNAL.md`, `archive/lauf-4/JOURNAL.md`,
`archive/lauf-5/JOURNAL.md` (Lauf 5 haengt dort am Ende des Lauf-4-Journals).

---

## Lauf 6 — Ausgangslage (2026-08-07, vor der ersten Iteration)

- Branch `backend-loop`, auf `origin/main` gemergt (Fast-Forward ueber 3 Commits).
- Migrationskopf lokal wie produktiv **297**, `dirty=false`.
- Lokale DB laeuft und ist verifiziert: `docker-postgres-1`, Rolle `kmuhub_app` mit Passwort
  `app_dev`, die RLS-Integrationstests in `internal/crm/contact` laufen real durch (0 Skips).
  `DATABASE_URL` ist damit kein Alibi mehr — wer ohne sie testet, hat kein Gate.
- Backlog: **70 offene Units** in drei Bloecken — A (20, verifizierte Luecken), C (16, Coverage
  auf den kritischen Pfaden biz/crm), B (34, Coverage-Breite server/gateway). Dazu 7 `blocked`
  aus Lauf 5, die bewusst liegen bleiben.
- **Fenster verschoben.** Der Lauf am 07.08. (21:00–08:45) fiel aus, der Rechner war belegt.
  Echtes Fenster: **2026-08-08 14:30 bis 00:45** (`-UntilTime "00:45"`; danach nur noch Push und
  CI-Warten). Das sind rund zehn Stunden. Beim Median aus Lauf 5 — 35 Iterationen in 5 h 40,
  also ~10 min — deckt das etwa 60 der 70 Units. Was von Block B liegen bleibt, startet Lauf 7:
  eingeplant, kein Versaeumnis. Ein Loop, der um 02:00 leerlaeuft, waere der teurere Fehler.
- Coverage-Ausgangswerte, am 2026-08-07 lokal ohne `DATABASE_URL` gemessen (untere Schranke):
  `internal/server` 6,2 % · `internal/gateway` 24,1 %. Mit DB in CI: 8,1 / 27,2.

## Iteration 1 — a-dunning-send-non-fatal — done — 2026-08-08 14:45
- commit: b790bcb8
- gebaut: Neues Sentinel `ErrCompanySettingsMissing` in `internal/biz/dunning/errors.go`;
  `emailNotice` wrappt es beim `settings == nil`-Zweig statt eines anonymen Fehlers, und
  `sendAndNotify` prueft per `errors.Is` — fehlende Firmen-Stammdaten sind jetzt non-fatal
  (Status flippt auf `sent`, `sent_at` gesetzt, Miss als `slog.Error` mit `dunning_id` und
  `tenant_id`), waehrend jeder echte Zustell-, PDF- oder Settings-Ladefehler weiterhin
  fail-closed bleibt und den Datensatz in `draft` haelt.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (dunning 53/53, 0 Skips;
  `./internal/gateway/` ok inkl. TestOpenAPIRouteDrift) | migration n.a. | rls-smoke n.a.
- mutations-probe: `if !errors.Is(sendErr, ErrCompanySettingsMissing)` zu
  `if !errors.Is(sendErr, ErrDunningNotFound)` gedreht (macht den Settings-Fall wieder fatal).
  Ergebnis: genau `TestSend_WithNoticeMailer_MissingCompanySettings_StillSends` wurde rot, die
  drei Nachbartests blieben gruen. Zurueckgedreht, Endgate danach erneut gelaufen und gruen.
  Anmerkung fuer kuenftige Proben: die naive Mutation `if true {` faellt nicht als Test-Rot
  auf, sondern als Compile-Fehler (`"errors" imported and not used`) — dann ist die Probe
  wertlos. Immer eine Mutation waehlen, die kompiliert.
- db-tests: 0 real gelaufene DB-Tests, 0 Skips — `internal/biz/dunning` ist reines
  Mock-Testing, es gibt in dem Paket keinen `SkipIfNoDB`-Pfad. `DATABASE_URL` war gesetzt.
- verify vorgaenger: n.a. — erste Iteration des Laufs, letzter Commit `535ea306` ist
  docs/chore am Loop-Verzeichnis. `git merge origin/main` war "Already up to date".
- offen: Bewusste Abweichung vom `scope`-Wortlaut, gedeckt durch `notes`: entschaerft ist NUR
  `settings == nil`, nicht jeder Fehler des konfigurierten Mailers. Ein SMTP-Refuse laesst die
  Mahnung weiter in `draft` (GoBD: "sent" = zugestellt). Zweiter Punkt fuer Luke: bei
  fehlenden Stammdaten gilt die Mahnung jetzt als `sent`, obwohl keine Mail rausging — die
  Fachlage will das so (der Nil-Mailer-Zweig macht es seit je genauso), aber die UI hat dafuer
  keinen Hinweis. Falls gewuenscht, gehoert eine Warnung in die Send-Response, das waere eine
  eigene Unit inkl. Proto-Feld.
