# Hetzner-Review-Checkliste (Cosmi-exe, app.zentria.tech)

> **Zweck:** Darien prüft Änderungen hands-on auf der über Hetzner laufenden Cosmi-exe, parallel während Main + Sub weiterbauen.
> **Ablauf:** Alles geht auf `main`. Hier sammeln sich konkrete „klick hier → erwarte das"-Items. Abgehakt = von Darien geprüft + ok.
> **Stand:** 2026-06-24.

## ⚠ Zwei Voraussetzungen (bitte einmal klären)
1. **Deploy-Gap:** Push auf `main` ≠ live auf Hetzner. Laut unseren Notizen ist Auto-Deploy (`cd.yml`) **nicht scharf** (Luke). → Damit Änderungen auf Hetzner erscheinen, muss jemand deployen. **Wer/wie deployt aktuell?**
2. **Demo- oder Live-Mode?** Läuft die Hetzner-exe im **Demo-Mode** (MSW, gefüllte Fake-Daten) oder **Live** (echtes Prod-Backend)?
   - **Demo-Mode** → du prüfst **FE/UX** (Layout, leere Zustände, Raw-Keys, Vollständigkeit). Meine Backend-Echt-Schaltungen (dialer/HR) sind hier **unsichtbar** (greifen nur live).
   - **Live** → die echt-geschalteten Module zeigen **echte (Prod-)Daten**; ohne Prod-Seed sind dialer/HR/Buchhaltung **leer** (nicht kaputt, nur leer), und die Backend-Fixes greifen erst **nach Deploy**.

---

## A · Jetzt prüfbar (FE/UX, Demo-Mode) — review-reife Module
> Die 15 FE-fertigen Module sind „review-reif". Pro Modul achten auf: tote Buttons, leere Zustände, Raw-i18n-Keys, `{{var}}`, Umlaut-Fehler, Detail-Modals (ganze Zeile klickbar, sticky Close), Sortierung.
- [ ] kontakte · [ ] calendar · [ ] dokumente · [ ] finanzen/Buchhaltung · [ ] work · [ ] team
- [ ] dashboard · [ ] vertraege · [ ] helpdesk · [ ] automatisierung · [ ] profil · [ ] mails · [ ] kommunikation · [ ] berichte · [ ] wiki

## B · In Arbeit (prüfen, sobald gemeldet)
- [ ] **security / DSGVO** (Sub baut gerade, Branch `parallel/security`) — wird nach Merge prüfbar. Achten auf: alle Seiten crashfrei, keine Raw-Keys, DSGVO-Flows (Export/Erasure/DSAR) durchklickbar. Sub meldet „security x/5 fertig".
- [ ] **zeiterfassung** (Main, echt-geschaltet) — siehe C.

## C · Backend-Echt-Schaltung (lokal verifiziert — Hetzner erst nach Deploy + Prod-Seed)
> Diese sind gegen das **lokale** Backend + lokale Demo-Seeds live verifiziert (Screenshots lokal). Auf Hetzner brauchen sie (a) Deploy der Backend-Fixes, (b) Prod-Demo-Daten.
- [x] **dialer-Supervisor** — lokal verifiziert (2 BE-Bugs gefixt: recent-calls-SQL + protojson-Null). Hetzner: nach Deploy + Dialer-Seed.
- [x] **dashboard-Layout** — Persistenz-Roundtrip lokal verifiziert (war schon verkabelt).
- [x] **zeiterfassung/HR** — BE-Bug gefixt (NULL `correction_reason` brach die Einträge-Liste). Hetzner: nach Deploy + HR-Seed.
- **Backend-Fixes, die deployt werden sollten (Luke):** `9dfcf89e` dialer recent-calls · `b7242926` hr correction_reason. Beide brechen sonst echte Daten still.

## D · Erledigt-Bestätigung (von Darien)
_(hier hakt Darien ab, was er auf Hetzner geprüft + ok befunden hat)_
