# Backend-Nachtloop — Journal (Lauf 4)

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
- offen: <was Luke morgens pruefen muss>
```

Uhrzeiten im Journal sind geraten — der Agent hat keine Uhr. Die Wahrheit steht in `logs/run.log`.

Journale der Vorlaeufe: `archive/lauf-1-2/JOURNAL.md` (58 Units, PR #15),
`archive/lauf-3/JOURNAL.md` (61 Units, PR #16).

---

## Ausgangslage Lauf 4 (2026-08-02, vor der ersten Iteration)

Kein Vorgaenger-Commit zu verifizieren — Lauf 3 ist abgenommen, durch CI und gemergt. Die erste
Iteration ueberspringt den Verify-Vorspann daher zu Recht und beginnt direkt mit `p1b-proto`.

Stand, gegen den gebaut wird:

- Migrationskopf Repo **268**. Naechste freie Nummer zur Laufzeit ermitteln, nicht annehmen:
  `ls backend/migrations | grep -E '^[0-9]{6}' | sort | tail -1`
- Registrierte Routen **778** gegen **780** dokumentierte Pfade (`TestOpenAPIRouteDrift` gruen).
- Backend-Gates zuletzt vollstaendig gruen: `golangci-lint ./...` 0 issues,
  `go test ./...` 130 ok / 0 FAIL gegen echte DB, Coverage 28,3 %.
- RBAC-Fundament aus Welle 1a steht und ist empirisch geprueft: als `kmuhub_app` 8 Preset-Rollen,
  1179 Grants, admin = 454 Capabilities. Presets sind lesbar, aber fuer Tenants nicht
  schreib- oder loeschbar (Schreib-Policy ohne die NULL-Klausel; DELETE wertet nur USING aus).
- **Neu freigegeben:** RBAC Welle 1b (Block A). Gesperrt bleiben Phase 4 (Branchen-BE), neue
  `config.RequireX`-Assertionen, das Scharfschalten neuer `modules.*`-Flags sowie Merge/Deploy.

Schwerpunkte dieses Laufs: **Sicherheit/RLS-Reste** (Block B — die Allowlist in ADR-006 kennt vier
Ausnahmen, ungeschuetzt sind deutlich mehr), **Automatisierung fertigbauen** (Block C) und die vier
verifizierten FE-Luecken (Block D).

Ein Hinweis zur Erwartung: der Backlog hat **29 offene Units**, nicht 40. Die FE-Client-Pfade
aller Module wurden gegen die registrierten Routen gediffed — die duennen Module (fuhrpark,
inventar, vermietung, einkauf, produktion, schichten, rapporte) haben keine Routen-Luecken mehr.
Sind alle Units abgearbeitet, ist `ALLE UNITS ABGEARBEITET` ins Journal zu schreiben, `STOP`
anzulegen und der Lauf zu beenden — nicht nach Arbeit zu suchen.

---
