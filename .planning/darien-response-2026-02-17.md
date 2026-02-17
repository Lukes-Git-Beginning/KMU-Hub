# Response to Darien - MASTER-PLAN + Office Strategy Review

**Date:** 2026-02-17
**Context:** Review of MASTER-PLAN.md, Office Strategy (3-tier Collabora), PRODUCT-STRATEGY.md, backend-plan-part1/part2

---

Hey Darien,

Hab beides durchgelesen -- MASTER-PLAN und Office-Strategie. Hier mein Feedback:

## Was ich gut finde

1. **Collabora statt OnlyOffice -- akzeptiert.** MPL 2.0 ist sauberer als AGPL, weniger RAM, und das Beste: unsere WOPI-Endpoints funktionieren mit beiden. Wir haben in Phase 11 bereits CheckFileInfo/GetFile/PutFile gebaut -- die arbeiten standardkonform und brauchen nur minimale Anpassungen fuer Collabora. Switch ist ca. 1 Tag Arbeit.

2. **3-Tier-Modell (Starter/Business/Enterprise) -- sehr gut.** TipTap fuer Business-Tier ist ein cleverer Differentiator. Backend-seitig braucht TipTap keine neuen Endpoints -- wir speichern HTML-Content, das war's.

3. **Deutschland-First (EUR, 19%/7%, de-DE) -- richtig.** Fuer Beta-Launch ist das der sinnvollste Fokus. CH/AT kommt spaeter per Konfiguration.

4. **MASTER-PLAN Struktur -- beeindruckend.** 174 Items, 14 Waves, LOC-Schaetzungen, [BACKEND-DEP]-Marker. Genau so koennen wir Frontend/Backend koordinieren.

5. **"Kommunikation" als separates Modul von "Chat" -- guter UX-Split.** Intern vs. Extern klar getrennt. Das ist bei uns Phase 14 "Unified Inbox" -- gleiche Sache, anderer Name. Lass uns den Namen "Kommunikation" im Frontend und "Unified Inbox" im Backend/Architektur nehmen.

6. **Theme-Cleanup (5 statt 7) -- mach ich.** Cherry-picke die Theme-Files von deinem Branch auf main.

## Wo wir nachbessern muessen

### 1. Backend-Plan ist veraltet

WICHTIG: Dein Sprint-Plan (backend-plan-part2) geht davon aus, dass Phases 8-9 noch offen sind. **Die sind fertig:**

| Phase | Status | Wann |
|-------|--------|------|
| Phase 8 (Video/Meetings) | FERTIG | 2026-02-11 |
| Phase 9 (Security/Compliance) | FERTIG | 2026-02-11 |
| Phase 10 (Email) | 3/7 done (Design-Integration) | Backend pending |
| Phase 11 (Dokumente + WOPI) | FERTIG | 2026-02-17 |

Das heisst: Sprints 7-10 in deinem Plan (8 Wochen) sind schon gebaut. Dein 50-Wochen-Plan reduziert sich auf ca. 20-25 Wochen mit AI-gestuetzter Entwicklung.

**Bitte aktualisiere backend-plan-part2.md** basierend auf unserem tatsaechlichen Stand. Ich schick dir STATE.md separat.

### 2. PRODUCT-STRATEGY.md aktualisieren

Die Datei ist vom 16.02. (vor der Strategy Session). Steht noch "OnlyOffice", "CHF", alte Phase-Reihenfolge. Bitte updaten mit:
- Collabora statt OnlyOffice
- EUR-first, de-DE default
- Payroll Anti-Feature explizit streichen
- Neue Phase-Reihenfolge (11-20)

### 3. WebDAV-Server: spaeter

Du fragst nach WebDAV-Erfahrung -- `golang.org/x/net/webdav` existiert als Go-Package. ABER: Fuer MVP reicht Download -> Edit -> Upload. Vollstaendiges WebDAV mit Locking, Versionierung, Konflikterkennung ist komplex und nicht noetig fuer Beta. Das kommt spaeter wenn Kunden es wirklich brauchen.

### 4. TanStack Query Migration fehlt

Alle 25 Design-Module nutzen Zustand Mocks. Wenn ich Backend-APIs baue, muss jede Page auf TanStack Query migriert werden (so wie wir es bei DokumentePage in 11-05 gemacht haben). Das ist signifikanter Aufwand der im MASTER-PLAN nicht explizit drin steht. Bitte als Cross-Cutting-Concern einplanen.

## Meine naechsten Schritte

1. **Jetzt:** Theme-Cleanup cherry-picken
2. **Naechste Phase:** Phase 10 Backend fertig (IMAP/SMTP, Plans 10-04 bis 10-07)
3. **Danach:** Phase 12 (Finanzen) -- Belegkette, GoBD, DATEV, QR-Rechnung
4. **Dann:** Phase 13 (HR), Phase 14 (Unified Inbox/Kommunikation)

## Was ich von dir brauche

1. PRODUCT-STRATEGY.md updaten (Collabora, EUR, Strategy-Session-Entscheidungen)
2. Backend-Plan-Part2 aktualisieren (Phases 8-11 sind fertig)
3. Klaerung: Ist "Kommunikation" = unser "Unified Inbox" (Phase 14)? Gleiche Architektur?
4. [BACKEND-DEP] Items auf unsere Roadmap-Phasen mappen -- das wird unser Koordinations-Vertrag

Gruss,
Luke
