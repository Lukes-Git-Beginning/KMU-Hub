# Hetzner-Review-Checkliste (Cosmi-exe, app.zentria.tech)

> **Zweck:** Darien prüft Änderungen hands-on auf der über Hetzner laufenden Cosmi-exe, parallel während Main + Sub weiterbauen.
> **Ablauf:** Alles geht auf `main`. Hier sammeln sich konkrete „klick hier → erwarte das"-Items. Abgehakt = von Darien geprüft + ok.
> **Stand:** 2026-06-24.

## ✅ Voraussetzungen geklärt (Darien, 2026-06-24)
1. **Deploy:** macht **Luke** (irgendwann). Push auf `main` ≠ sofort live → Änderungen erscheinen auf Hetzner erst nach Lukes Deploy.
2. **Mode: LIVE** (echtes Prod-Backend, kein MSW).
   - **Folge A:** Backend-Echt-Schaltungen (dialer/dashboard/HR) sind nach Lukes Deploy prüfbar — zeigen **echte Prod-Daten** (ohne Prod-Seed ggf. leer, nicht kaputt).
   - **Folge B (wichtig):** **FE-mock-Module zeigen im Live-Mode NICHT ihre MSW-Demodaten** — sie treffen das echte Backend. Module mit fehlendem Backend (🔒, z. B. **security/DSGVO**) sind auf Hetzner-Live **erst prüfbar, wenn Luke das Backend gebaut hat**. Bis dahin: reine FE/UX-Reviews dieser Module besser lokal.

---

## A · Jetzt prüfbar (FE/UX, Demo-Mode) — review-reife Module
> Die 15 FE-fertigen Module sind „review-reif". Pro Modul achten auf: tote Buttons, leere Zustände, Raw-i18n-Keys, `{{var}}`, Umlaut-Fehler, Detail-Modals (ganze Zeile klickbar, sticky Close), Sortierung.
- [ ] kontakte · [ ] calendar · [ ] dokumente · [ ] finanzen/Buchhaltung · [ ] work · [ ] team
- [ ] dashboard · [ ] vertraege · [ ] helpdesk · [ ] automatisierung · [ ] profil · [ ] mails · [ ] kommunikation · [ ] berichte · [ ] wiki

## B · Bereit zum Review (gemergt auf main)
- [ ] **security / DSGVO** ✅ gemergt (`43fecf37`, S-1…S-5) — **Demo-Mode prüfbar** (reine FE/MSW-Arbeit). Hub `/admin/security` (10 Sub-Tabs). Achten auf: alle Seiten crashfrei, keine Raw-Keys (DE+EN), DSGVO-Flows durchklickbar — Audit (Filter/Export), DSAR Art.15 (Cross-Modul-Suche + Export), Export Art.15/20 (Genehmigen/Download + Frist), Erasure Art.17 (Preview/Execute + Legal-Hold-Hinweis), Retention (DACH-Fristen + Auto-Löschung-Toggle), Sessions (beenden), Vault, PW-Policy, IP-Access, 2FA. Sub-Bericht: `.planning/parallel-batch/qa-security.md`.
- [ ] **zeiterfassung** (Main, echt-geschaltet) — siehe C.

## C · Backend-Echt-Schaltung (lokal verifiziert — Hetzner erst nach Deploy + Prod-Seed)
> Diese sind gegen das **lokale** Backend + lokale Demo-Seeds live verifiziert (Screenshots lokal). Auf Hetzner brauchen sie (a) Deploy der Backend-Fixes, (b) Prod-Demo-Daten.
- [x] **dialer-Supervisor** — lokal verifiziert (2 BE-Bugs gefixt: recent-calls-SQL + protojson-Null). Hetzner: nach Deploy + Dialer-Seed.
- [x] **dashboard-Layout** — Persistenz-Roundtrip lokal verifiziert (war schon verkabelt).
- [x] **zeiterfassung/HR** — BE-Bug gefixt (NULL `correction_reason` brach die Einträge-Liste). Hetzner: nach Deploy + HR-Seed.
- **Backend-Fixes, die deployt werden sollten (Luke):** `9dfcf89e` dialer recent-calls · `b7242926` hr correction_reason. Beide brechen sonst echte Daten still.

## D · Erledigt-Bestätigung (von Darien)
_(hier hakt Darien ab, was er auf Hetzner geprüft + ok befunden hat)_
