# Sub-Terminal — security / DSGVO FE-Tiefe (S-0 … S-6)

> **Du bist das SUB-Terminal im Klon `…/KMU-Hub-review`, Dev-Port 5174, Demo-Mode (MSW, KEIN Backend nötig).**
> Du baust **nur `security`**. dialer gehört dem Main-Terminal (echt-Schaltung gegen Docker) — fass dialer / `api/dialer-*` / `mocks/handlers/dialer.ts` NICHT an.
> **WICHTIG — Docker:** Du fährst **niemals** Docker hoch. Zwei Klone mit eigenem Stack = WSL2-OOM (16-GB-Maschine). security läuft komplett auf MSW. Das echte DSGVO-Backend ist Lukes Track (🔒).
> **Ablauf:** erst S-0 (Ist-Research + Marktrecherche + Klärungsfragen an Darien) — **STOPP am Gate, nicht weiterbauen, bis Darien antwortet.** Danach S-1…S-6 autonom, Meldung nach jeder Phase (`S-x fertig, n/6`).

---

## Warum security jetzt (Kontext)
**DSGVO ist P0-Launch-Blocker** (Cosmi verkauft „EU-Datensouveränität" als USP). Das security-Modul ist die Vorzeige-Fläche dafür. Die FE-Seiten existieren bereits substanziell, sind aber halb-tot: vermutlich Raw-i18n-Keys + flache Mocks + nicht-funktionale Flows. Ziel dieses Batches: **security review-reif als FE-mock-first** (echtes Backend = Luke, später echt-geschaltet).

## Ist-Stand (Main-Terminal hat grob vor-recherchiert — DU verifizierst tief in S-0)
- **11 Seiten** unter `desktop/src/renderer/src/modules/security/` (~3.600 Zeilen, also kein nacktes Skelett):
  `SecurityAdminPage` · `AuditLogPage` · `SessionsPage` · `IPAccessPage` · `PasswordPolicyPage` · `RetentionPolicyPage` · `GDPRExportPage` (Art. 15/20) · `GDPRErasurePage` (Art. 17) · `DSARSearchPage` (Auskunftsersuchen) · `VaultPage` (Secrets) · `TwoFactorSetupWizard`.
- **Routing:** hängt unter `/admin/security` (AdminHubPage-Tabs) + Legacy-Redirect `/admin/security-legacy`. ⚠ **Nur `modules/security/*` anfassen, NICHT die anderen AdminHub-Tabs (IT/Billing/Integrations).** App.tsx-Routen nur minimal-invasiv, falls überhaupt nötig.
- **i18n KAPUTT (Haupt-Befund):** Seiten rufen `t('security.…')`, `t('gdpr.…')`, `t('audit.…')`, `t('ipAccess.…')`, `t('vault.…')`, `t('password.…')`, `t('session.…')` — aber in `i18n/messages/de.json` sind **0** dieser Namespaces vorhanden → **Raw-Keys werden im UI gerendert.** Das ist der größte mechanische Brocken (i18n ×4 von Grund auf).
- **MSW:** `mocks/handlers/security.ts` **existiert** (Stand verifizieren — wie tief? welche Endpoints? stateful?).

## Out of scope (🔒 Luke — NICHT bauen, nur mock-first + swap-ready lassen)
- Echte Audit-Log-Unveränderlichkeit (append-only Store/Hash-Chain), echte Session-Invalidierung serverseitig, echter Art.-15/20-Export (revisionssicheres PDF/ZIP), echte Erasure/Anonymisierung, echte DSAR-Volltextsuche über alle Tabellen, echte Passwort-Policy-Enforcement, IP-Allowlist-Durchsetzung im Gateway, echter Secrets-Vault (KMS), WebAuthn-Backend. → **Alles FE-mock-first + sichtbar als „Demo" markiert wo sinnvoll.** Backend-Bedarf in `.planning/backend-gaps.md` notieren (Abschnitt „security/DSGVO").

---

## S-0 — Gate: Ist-Research + Marktrecherche + Klärungsfragen  ⛔ STOPP nach diesem Schritt
**Technische Ist-Research (Pflicht, gründlich):**
1. Alle 11 Seiten im echten UI öffnen (Demo-Mode, :5174) + **Screenshots ansehen** → pro Seite festhalten: funktional vs. Skelett, Raw-Keys sichtbar?, leere/tote Buttons?, Mock-Daten realistisch?
2. `mocks/handlers/security.ts` lesen → welche Endpoints, stateful oder flach, welche Felder.
3. i18n-Lücke exakt vermessen: welche Namespaces/Keys fehlen über `{de,en,fr,it}.json` (Liste).
4. Prüfen, ob security im `module-settings-registry.tsx` einen Eintrag hat.

**Marktrecherche (Pflicht — WebSearch, dann mit Cosmi-Ist abgleichen):**
- Wie sehen **DSGVO-/Security-Center** in B2B-SaaS aus? Referenzen: OneTrust, Usercentrics, Vanta, Drata, GitLab/Notion „Security & Privacy"-Settings, Microsoft-Compliance-Center.
- Konkret pro Bereich: Was gehört in ein **Audit-Log** (Spalten, Filter, Export)? Wie präsentiert man **DSAR / Auskunftsersuchen** (Art. 15) und **Löschersuchen** (Art. 17) als geführten Flow (Request → Status → Artefakt)? Was sind erwartbare **Retention-Policy-** und **Passwort-Policy-**Felder (DACH-Norm)? Wie zeigt man **aktive Sessions** + Geräte-Widerruf? Was ist State-of-the-Art bei **2FA-Setup**?
- Ergebnis: 1 knapper Abschnitt „Markt vs. Cosmi-Ist — Lücken & Empfehlungen" je Bereich (in `qa-security.md` festhalten).
- `/intel-recall security` bzw. `/intel-recall dsgvo` laufen lassen, falls Keepers existieren.

**Dann: gebündelte Klärungsfragen an Darien** (eine Nachricht, nummeriert). Erwartbare Fragen:
- Scope-Tiefe je Bereich (alle 11 Seiten gleich tief, oder DSGVO-Kern [Export/Erasure/DSAR] priorisiert)?
- Audit-Log: nur security-Events oder modulübergreifend (alle Mutations)?
- Vault: behalten (Secrets-Manager) oder ist das Feature-Scope-Creep → später?
- 2FA-Wizard: schon woanders im Login/Profil vorhanden → Duplikat vermeiden?
**STOPP. Erst nach Dariens Antworten S-1 starten.**

---

## S-1 — i18n-Fundament ×4 (größter mechanischer Brocken)  `[FOUNDATION]`
**Soll:** Alle fehlenden Namespaces (`security`, `gdpr`, `audit`, `ipAccess`, `vault`, `password`, `session`, `dsar` + ggf. weitere aus S-0) in `i18n/messages/{de,en,fr,it}.json` anlegen — **vollständig, alle 4 Sprachen**. `{var}` single-brace, ICU-Plural `{count, plural, …}`, **nie** `{{var}}`, nie `_one`/`_other`. Keys ins jeweilige Prefix-Cluster einsortieren (nicht ans Datei-Ende).
**Verify:** Über alle 11 Seiten DE+EN **0 Raw-Keys** mehr (Screenshot je Seite ansehen). FR+IT zumindest key-vollständig (kein Fallback-auf-Key).

## S-2 — AuditLog + Sessions funktional (MSW-tief)
**Soll:** AuditLog mit Filtern (Akteur/Aktion/Zeitraum/Modul), unveränderlich-Look (read-only, Export-Knopf CSV via MSW-Blob), Zeilenklick → `shared/DetailModal` mit Volldetails. SessionsPage: aktive Sessions (Gerät/IP/Standort/letzter Zugriff), „Widerrufen"-Aktion wirkt (stateful MSW), „Alle anderen abmelden".
**Verify:** Filter reduzieren korrekt; Detail-Modal vollständig + sticky Close; Widerruf entfernt Session sichtbar; leere Zustände sauber.

## S-3 — DSGVO-Kern: Export (Art. 15/20) + Erasure (Art. 17) + DSAR-Suche
**Soll:** Die 3 GDPR-Seiten als **geführte Flows mock-first**: Anfrage anlegen (Betroffener wählen/suchen) → Status (angefordert/in Arbeit/fertig) → Artefakt herunterladbar (MSW-Blob: ZIP/PDF-Stub). DSARSearch: Suche über Demo-Betroffene → Treffer mit Kategorien. Erasure: Bestätigungs-Flow + Audit-Eintrag. Alles stateful (überlebt Navigation, ggf. Reload via MSW-Seed).
**Verify:** Kompletter Flow je Seite durchklickbar; Download liefert Datei; Status-Wechsel sichtbar; leere Zustände + Bestätigungs-Dialoge da.

## S-4 — Policies: Passwort + IP-Access + Retention
**Soll:** Editierbare Policy-Formulare mit Validierung, stateful MSW-Persist (überlebt Reload). PW-Policy (Länge/Komplexität/Ablauf/History), IP-Access (Allowlist-CIDR hinzufügen/entfernen, Demo-Modus-Banner „nicht enforced"), Retention (pro Datenkategorie Aufbewahrungsfrist).
**Verify:** Speichern persistiert; Validierung greift; CIDR-Add/Remove wirkt; Demo-Hinweise klar.

## S-5 — 2FA-Wizard + Vault
**Soll:** TwoFactorSetupWizard-Flow polieren (QR/Secret → Code-Eingabe → Backup-Codes → Abschluss), Schritte sauber, i18n. VaultPage: Secrets-Liste (Name/Typ/Rotation), Reveal/Copy/Rotate als Mock, Demo-Hinweis. (Falls Darien in S-0 Vault deprioris. → überspringen.)
**Verify:** Wizard-Flow vollständig durchklickbar; Vault-Aktionen wirken (Mock); keine toten Buttons.

## S-6 — Demo-Tiefe-Schlusscheck + Modul-Settings + Screenshot-QA
**Soll:**
- Modul-Settings-Eintrag `id: 'security'` in `module-settings-registry.tsx` (`ModuleSettingsShell`, personal + tenant sinnvoll: z. B. persönlich 2FA-Status, tenant-weit Policy-Defaults-Verweis). ⚠ Main trägt evtl. zeitgleich `id: 'dialer'` ein → finaler Merge behält beide.
- Sweep: keine toten Buttons/Toast-only-Stubs; Sortierung via `shared/SortMenu` wo Listen; ganze Zeile klickbar wo Detail; sticky Back/Close überall.
- Screenshot-QA **alle 11 Seiten** @1440 + @1024, **DE + EN** → 0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors.
**Verify:** Alles grün, Bilder angesehen, in `qa-security.md` dokumentiert.

---

## Branch-Setup (einmalig, ZUERST)
Bau **NICHT** direct-to-main. `git checkout main && git pull`, dann `git checkout -b parallel/security`. Alle S-Punkte committest + pushst du auf **diesen** Branch (`git push -u origin parallel/security`). Kein `git pull --rebase` von main nötig — isoliert. Das Main-Terminal merged `parallel/security` am Ende kontrolliert (i18n + registry: beide Blöcke behalten, danach `npm run build`).

## Workflow pro Phase (Build-+-Verify-Standard, CLAUDE.md)
bauen → i18n ×4 (`{var}`, ICU-Plural) → MSW/Demo-Daten → Compile-Gate (`npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error /tmp/build.log`, **NIE `| tail`**) → Playwright-Screenshot-QA gegen **:5174** + **Bilder WIRKLICH ansehen** → iterieren bis grün → ein Commit + Push auf `parallel/security` → Eintrag in `qa-security.md`.

## Dev-Server (Sub, Demo-Mode, kein Backend)
```bash
cd "C:/Users/darie/Documents/KMU-Hub-review" && git checkout -b parallel/security
cd desktop && npm install   # falls node_modules nicht aktuell
npm run dev -- --port 5174  # Demo-Mode (MSW), security unter /admin/security
```
Dev-Server killen (Windows): PowerShell `Get-NetTCPConnection -LocalPort 5174 | … | Stop-Process`. Nur 1 Dev-Server pro QA-Runde.

## Lane-Trennung (verbindlich)
- Nur `modules/security/*`, `mocks/handlers/security.ts`, security-i18n-Namespaces, `module-settings-registry.tsx` (ein Eintrag). **NICHT** dialer, NICHT andere AdminHub-Tabs, NICHT `shared/*` umbauen (nur konsumieren: `DetailModal`, `SortMenu`).
- `qa-security.md` selbst anlegen (nur deine).

## Definition of Done (security review-reif)
Alle S-1…S-6 verifiziert (Screenshots angesehen), 0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors über alle 11 Seiten DE+EN, DSGVO-Kern-Flows (Export/Erasure/DSAR) durchklickbar mock-first, jede Phase ein Commit+Push auf `parallel/security`, `qa-security.md` gepflegt, Backend-Bedarf in `backend-gaps.md`. Dann Darien: „security 6/6 fertig".
