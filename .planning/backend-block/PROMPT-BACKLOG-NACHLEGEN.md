# Prompt — Backlog Lauf 4 auf Feature-Ebene nachlegen

> Zum Kopieren in ein frisches Fenster. Self-contained: der Prompt setzt keinen Kontext aus der
> Vorsession voraus. Erwarteter Aufwand 1–2 h. Modell: **Opus** (Rechercheurteil, kein Volumen).

---

## Der Prompt

Ich brauche 10–12 zusätzliche, **verifizierte** Units für den Backend-Nachtloop (`BACKLOG.yml`).
Der Backlog hat aktuell 29 offene Units ≈ 8–9 Stunden; das Fenster heute Nacht ist 20:00–08:00,
also fehlen rund 3 Stunden Arbeit.

**Lies zuerst `.planning/backend-block/UEBERGABE-MERGE-UND-LAUF-4.md`** — dort steht der komplette
Stand (Merge-Prozedur, was Lauf 3 geliefert hat, was schon geprüft ist).

### Was NICHT nochmal gemacht werden muss

Das ist am 2026-08-02 bereits erledigt, bitte nicht wiederholen:

- **Routen-Ebene ist abgegrast.** Alle FE-Client-Pfade aus `desktop/src/renderer/src/api/**`
  wurden gegen die 778 registrierten Routen gediffed und jeder Treffer per
  `grep -rn "<pfad>" backend/internal/gateway/` gegengeprüft. Ergebnis: die dünnen Module
  (fuhrpark, inventar, vermietung, einkauf, produktion, schichten, rapporte) haben **keine
  Routen-Lücken mehr**. Die vier echten Funde (CRM-Timeline, Kalender-Ressourcen-Buchungen,
  Admin-Billing, Vendor-Access) stehen bereits als Block D im Backlog.
- **RLS-Scan ist gemacht.** 318 Tabellen, 249 mit RLS; die ungeschützten sind als Block B
  vollständig eingeplant. Nicht neu scannen.
- **RBAC Welle 1b** ist als Block A eingeplant (9 Units) und freigegeben.

### Die Aufgabe

Die Lücken liegen jetzt auf **Feature-Ebene**, nicht auf Routen-Ebene: ein Endpoint existiert,
aber das Feature dahinter ist unvollständig oder falsch. Quelle ist `.planning/backend-gaps.md`
(482 Zeilen, 28 Modul-Abschnitte) — **die Datei ist an mehreren Stellen überholt**, weil die
Läufe 1–3 vieles davon still geschlossen haben.

Deshalb ist die Kernarbeit **Verifikation, nicht Abschreiben**. Beispiel aus dem einkauf-Abschnitt,
beide Punkte stehen dort gleichwertig als offen:

- `POST /einkauf/pos/{id}/cancel` — steht als fehlend, ist aber seit Lauf 1 gebaut
  (`route_einkauf.go:91`). **Keine Unit.**
- `PurchaseOrder.total_amount` — steht als nie berechnet, und das stimmt: `service.go:317` setzt
  hart `"0"`, kein Recompute in Add/Update/DeletePOLine. **Verifizierte Unit** (liegt schon
  als `fix-einkauf-po-total` im Backlog).

Genau diese Trennschärfe brauche ich für 10–12 weitere Punkte.

### Vorgehen

1. Geh `.planning/backend-gaps.md` durch und sammle die konkretesten offenen Behauptungen —
   bevorzugt die mit 🔒 oder mit einer Datei-/Zeilenangabe.
2. **Prüfe jede Behauptung gegen den laufenden Code**, bevor sie eine Unit wird:
   - Tabelle/Spalte da? → lokale DB, Container `docker-postgres-1`:
     `docker exec -i docker-postgres-1 psql -U kmuhub -d kmuhub -tA -c "..."`
     (Migrationskopf ist **268**; als App-Rolle testen: `kmuhub_app`, nicht `kmuhub` — der
     Superuser hat BYPASSRLS)
   - Route da? → `grep -rn "<pfad>" backend/internal/gateway/`
   - Service-Methode echt oder Stub? → die Funktion lesen, nicht den Namen
   - Erwartet das FE etwas? → `desktop/src/renderer/src/api/<modul>-client.ts` + der
     MSW-Handler in `mocks/handlers/` sind der bindende Vertrag
3. Was schon gebaut ist: in `backend-gaps.md` als erledigt markieren (das entstaubt die Datei für
   Lauf 5 gleich mit — ausdrücklich erwünscht).
4. Was echt offen ist: als Unit formulieren.

### Format der Units

Anhängen an `.planning/backend-block/loop/BACKLOG.yml` als neuer Block, **vor** dem Block F
(den aus Lauf 3 übernommenen Units). Struktur exakt wie die bestehenden Units:

```yaml
  - id: <kebab-case>
    phase: 3
    service: <modul>
    model: sonnet          # opus nur für Proto, Security, Migration+RLS
    deps: []
    status: todo
    scope: >
      Was gebaut wird, in ganzen Sätzen. Mit der verifizierten Fundstelle
      (Datei:Zeile oder Tabellenname), nicht mit "laut backend-gaps".
    sources:
      - <pfade, die der Agent zuerst lesen soll>
    notes: >
      Was beim Bauen schiefgehen kann, welches bestehende Muster wiederzuverwenden ist,
      und — falls die Prämisse wackelt — der Auftrag, sie zuerst zu prüfen und sonst
      `blocked` zu setzen.
    done_when:
      - nachprüfbare Kriterien, kein "funktioniert"
```

**Zwei harte Parsing-Regeln des Treibers** (`run-loop.ps1`), sonst zählt er die Units falsch:
`model:` muss **vor** `status:` stehen, und hinter `status: todo` darf **kein** Kommentar stehen.

### Qualitätsmaßstab

- Eine Unit, deren Prämisse ich nicht selbst im Code gesehen habe, kommt nicht rein. Lieber
  8 belastbare Units als 12, von denen der Loop nachts vier als `blocked` zurückgibt.
- Jede neue Tabelle braucht `tenant_id UUID NOT NULL` + `CALL enable_tenant_rls(...)` — im Backlog
  liegt ein Regressionstest (`g-rls-regression-guard`), der das ab dieser Nacht erzwingt.
- Gesperrt bleibt: Phase 4 (Branchen-BE), neue `config.RequireX`-Assertionen, Scharfschalten
  neuer `modules.*`-Flags, Merge/Deploy.
- Schwerpunkt wie bisher: Sicherheit vor Features, echte Lücken vor Kosmetik.

### Abschluss

- `BACKLOG.yml` muss danach valides YAML sein und der Trockenlauf muss die neue Zahl zeigen:
  `powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -DryRun`
  → erwartet „offen: <29 + neue>"
- Ein Commit, Conventional Commits, **keine AI-Attribution**.
- **Nicht pushen**, solange PR #16 noch offen und ungemergt ist — ein Push feuert Claude PR Review
  und Security Review (beide ohne Draft-Gate). Ist der Merge durch, gilt die Reihenfolge aus der
  Übergabe-Datei §4.
- Sag mir am Ende, welche Punkte aus `backend-gaps.md` sich als **bereits erledigt** erwiesen
  haben — das ist für mich genauso wertvoll wie die neuen Units.
