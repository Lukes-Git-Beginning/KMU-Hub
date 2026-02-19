# Nachricht an Darien - Review MASTER-PLAN + Office Strategy + Sync-Request

---

Hey Darien,

Hab alles durchgelesen -- MASTER-PLAN, Office-Strategie, PRODUCT-STRATEGY und die beiden Backend-Plan-Teile. Hier mein komplettes Feedback.

## Was ich gut finde

1. **Collabora statt OnlyOffice -- akzeptiert.** MPL 2.0 ist sauberer als AGPL, weniger RAM, und das Beste: unsere WOPI-Endpoints funktionieren mit beiden. Wir haben in Phase 11 bereits CheckFileInfo/GetFile/PutFile gebaut -- die arbeiten standardkonform und brauchen nur minimale Anpassungen fuer Collabora. Switch ist ca. 1 Tag Arbeit.

2. **3-Tier-Modell (Starter/Business/Enterprise) -- sehr gut.** TipTap fuer Business-Tier ist ein cleverer Differentiator. Backend-seitig braucht TipTap keine neuen Endpoints -- wir speichern HTML-Content, das war's.

3. **Deutschland-First (EUR, 19%/7%, de-DE) -- richtig.** Fuer Beta-Launch ist das der sinnvollste Fokus. CH/AT kommt spaeter per Konfiguration.

4. **MASTER-PLAN Struktur -- beeindruckend.** 174 Items, 14 Waves, LOC-Schaetzungen, [BACKEND-DEP]-Marker. Genau so koennen wir Frontend/Backend koordinieren.

5. **"Kommunikation" als separates Modul von "Chat" -- guter UX-Split.** Intern vs. Extern klar getrennt. Das ist bei uns Phase 14 "Unified Inbox" -- gleiche Sache, anderer Name. Lass uns den Namen "Kommunikation" im Frontend und "Unified Inbox" im Backend/Architektur nehmen.

6. **Theme-Cleanup (5 statt 7) -- bereits erledigt.** Nature und Atelier sind auf main entfernt. Jetzt 5 Themes: Cozy, Dreamy, Raumstation, Clean, Minimal.

## Wo wir nachbessern muessen

### 1. Backend-Plan ist veraltet

WICHTIG: Dein Sprint-Plan (backend-plan-part2) geht davon aus, dass Phases 8-9 noch offen sind. Hier ist der tatsaechliche Stand -- **alles bis Phase 11 ist fertig:**

| Phase | Status | Wann |
|-------|--------|------|
| Phase 8 (Video/Meetings) | FERTIG | 2026-02-11 |
| Phase 9 (Security/Compliance) | FERTIG | 2026-02-11 |
| Phase 10 (Email) | FERTIG (alle 7 Plans) | 2026-02-17 |
| Phase 11 (Dokumente + WOPI) | FERTIG (alle 6 Plans) | 2026-02-17 |

Das heisst: Sprints 7-10 in deinem Plan (8 Wochen) sind schon gebaut. Dein 50-Wochen-Plan reduziert sich auf ca. 20-25 Wochen mit AI-gestuetzter Entwicklung. 66 von 66 Plans (Phase 4-11) sind abgeschlossen.

**Bitte aktualisiere backend-plan-part2.md** basierend auf diesem Stand.

### 2. PRODUCT-STRATEGY.md aktualisieren

Die Datei ist vom 16.02. (vor der Strategy Session). Steht noch "OnlyOffice", "CHF", alte Phase-Reihenfolge. Bitte updaten mit:
- Collabora statt OnlyOffice
- EUR-first, de-DE default
- Payroll Anti-Feature explizit streichen
- Neue Phase-Reihenfolge (11-20)

### 3. WebDAV-Server: spaeter

Du fragst nach WebDAV-Erfahrung -- `golang.org/x/net/webdav` existiert als Go-Package. ABER: Fuer MVP reicht Download -> Edit -> Upload. Vollstaendiges WebDAV mit Locking, Versionierung, Konflikterkennung ist komplex und nicht noetig fuer Beta. Das kommt spaeter wenn Kunden es wirklich brauchen.

### 4. TanStack Query Migration fehlt im MASTER-PLAN

Alle 25 Design-Module nutzen Zustand Mocks. Wenn Backend-APIs stehen, muss jede Page auf TanStack Query migriert werden (so wie wir es bei DokumentePage in Plan 11-05 gemacht haben). Das ist signifikanter Aufwand der im MASTER-PLAN nicht explizit drin steht. Bitte als Cross-Cutting-Concern einplanen.

## WICHTIG: Bitte main pullen und anschauen

Dein Branch `design/brainstorm` ist mittlerweile weit hinter `main`. Auf main ist seit der Design-Integration extrem viel passiert. Bitte hol dir den aktuellen Stand, damit du siehst was wirklich gebaut ist:

```bash
git checkout main
git pull origin main
```

Speziell anschauen solltest du:
- **`backend/`** -- Alle Microservices (auth, crm, chat, work, calendar, video, security, email, documents) sind implementiert
- **`desktop/src/renderer/src/modules/dokumente/`** -- DokumentePage ist bereits von Zustand auf TanStack Query migriert (Referenz fuer alle anderen Module)
- **`desktop/src/renderer/src/modules/mails/`** -- Email-Frontend mit Compose, Inbox, Thread-View
- **`backend/internal/documents/wopi/`** -- WOPI-Endpoints die mit Collabora kompatibel sind
- **`deploy/docker/docker-compose.yml`** -- Alle Services konfiguriert
- **`.planning/`** -- Alle Phase-Plans und Summaries, STATE.md, ROADMAP.md

Das gibt dir ein realistisches Bild davon wo wir stehen und hilft dir beim Frontend-Planning. Viele Sachen die im MASTER-PLAN als TODO stehen haben schon Backend-APIs die du direkt anbinden kannst.

## Meine naechsten Schritte

1. **Jetzt:** Phase 12 starten (Rechnungen & Finanzen) -- Belegkette, GoBD, DATEV, QR-Rechnung
2. **Danach:** Phase 13 (HR), Phase 14 (Unified Inbox/Kommunikation)

## Was ich von dir brauche

1. **main pullen** und den aktuellen Code-Stand anschauen
2. **PRODUCT-STRATEGY.md updaten** (Collabora, EUR, Strategy-Session-Entscheidungen)
3. **Backend-Plan-Part2 aktualisieren** (Phases 8-11 sind komplett fertig)
4. **Klaerung:** Ist "Kommunikation" = unser "Unified Inbox" (Phase 14)? Gleiche Architektur?
5. **[BACKEND-DEP] Items mappen** auf unsere Roadmap-Phasen -- das wird unser Koordinations-Vertrag
6. **Theme-Aenderungen auf deinem Branch nachziehen** -- Nature + Atelier sind weg, desk-themes.ts und desk-asset-urls.ts haben sich geaendert

Gruss,
Luke
