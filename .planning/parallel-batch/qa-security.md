# QA-Protokoll — Modul `security` (DSGVO / Security-Center)

> Sub-Terminal (`KMU-Hub-review`, Branch `parallel/security`, Demo-Mode/MSW, Dev-Port real **5173**).
> **Stand S-0 (Research) — 2026-06-24.** Build-+-Verify pro Bau-Phase folgt nach Darien-Freigabe.

---

## S-0 · Ist-Research (Code + Live @:5173 + Markt)

### Hinweis vorweg: `sub-security.md` fehlt im Repo
Das verbindliche Phasen-Paket `.planning/parallel-batch/sub-security.md` (Phasen S-1…S-6, Lane-Regeln, Out-of-scope) **existiert nicht** — das Main-Terminal hat es noch nicht gepusht. README im parallel-batch ist noch auf dem alten Batch (berichte/notifications). S-0 ist davon unabhängig (reine Research) → durchgeführt. Scope-Definition S-1…S-6 daher **als Vorschlag** unten, Klärung mit Darien nötig.

### Modul-Topologie (11 Seiten + Hub-Schichten)
| Datei | Zeilen | Rolle |
|---|---|---|
| `SecurityAdminPage.tsx` | 139 | **Legacy-Hub**, Route `/admin/security-legacy`, 9 Tabs |
| `AuditLogPage.tsx` | 365 | Audit-Log |
| `SessionsPage.tsx` | 308 | Aktive Sitzungen |
| `VaultPage.tsx` | 452 | Secret-Vault |
| `PasswordPolicyPage.tsx` | 330 | Passwort-Richtlinie |
| `IPAccessPage.tsx` | 322 | IP-Allow/Block |
| `GDPRExportPage.tsx` | 323 | Datenexport Art. 20 (Admin-Freigabe) |
| `GDPRErasurePage.tsx` | 412 | Löschung Art. 17 |
| `DSARSearchPage.tsx` | 318 | Auskunft Art. 15 (Cross-Modul-Suche) |
| `RetentionPolicyPage.tsx` | 164 | Aufbewahrungsfristen DACH |
| `TwoFactorSetupWizard.tsx` | 463 | 2FA-Setup-Modal (eingebunden in **settings**, nicht im Hub) |

**Zweiter Hub:** `admin/tabs/SecurityAdminHubTab.tsx` — neuer Admin-Hub, Route `/admin/security` (→ `AdminHubPage`), 7 Sub-Tabs: `audit | gdpr | sessions | ip-whitelist | vault | privacy | ai`. Bindet **PasswordPolicy, DSAR, Retention NICHT ein.** DSGVO-Tab rendert Export+Erasure untereinander.

### Routing-Konflikt (klärungsbedürftig)
Es gibt **zwei konkurrierende Security-Hubs**:
- **Neu** `/admin/security` → `AdminHubPage` → `SecurityAdminHubTab` (7 Sub-Tabs). Gate: `['admin','it_support']`. Kommentar: legacy soll „in Phase 4 entfernt" werden.
- **Legacy** `/admin/security-legacy` → `SecurityAdminPage` (9 Tabs, Gruppen Überwachung/Zugriff/Datenschutz).
→ **DSAR + Retention + PasswordPolicy sind aktuell nur über den Legacy-Hub erreichbar.**

### Live-Funktionsstatus (Demo-Mode, real getestet @:5173, Screenshots in `.qa-screenshots/`)
| Seite | Status Demo | Ursache |
|---|---|---|
| Audit-Log | ✅ **funktioniert** (50 Einträge, Filter, Export/Verify-Buttons, i18n ok) | Handler-Contract passt zufällig (`{entries,total}`) |
| Vault | ❌ **CRASH** `secretsList.map is not a function` | Handler liefert `{secrets:[]}`, client erwartet `VaultSecret[]` |
| GDPR-Export | ❌ **CRASH** `allExports.filter is not a function` | Handler liefert `{exports:[]}`, client erwartet `GDPRExportRequest[]` |
| IP-Access | ❌ **CRASH** (gleicher Mismatch) | Handler `{rules:[]}` vs client `IPAccessRule[]` |
| Password-Policy | ❌ **defekt** | Handler `{policy:{}}` vs client `PasswordPolicy` |
| Sessions | ❌ **CRASH** `…toLowerCase` (deviceType undefined) | **kein** MSW-Handler für `/auth/sessions` |
| DSAR | ❌ Suche schlägt fehl | **kein** MSW-Handler für `/security/dsar/search` |
| GDPR-Erasure | ⚠️ **Mock-Skeleton** | lokale `setTimeout`-Simulation, ruft nie echte API |
| Retention | ⚠️ **statisch** | hardcoded `RETENTION_DATA`, kein API, im neuen Hub gar nicht eingebunden |
| 2FA-Wizard | ⚠️ **blockiert** ab Schritt 1 | kein MSW für `/auth/2fa/setup|verify|disable` |

**Kernbefund: Von 11 Seiten funktioniert im Demo-Mode genau EINE (Audit-Log).** Hauptursache = **Daten-Contract-Mismatch** zwischen MSW-Handlern (`{feld:[...]}`) und client-Typen (nacktes Array) + **fehlende Handler**.

### MSW-Coverage: 7 / 33 Endpoints (21 %)
Vorhanden: `GET audit`, `GET vault`, `GET password/policy`, `POST password/validate`, `GET ip-rules`, `GET gdpr/exports` (leer), `GET auth/2fa/policies`.
Fehlend (26): alle Sessions (4), DSAR-Suche, alle 2FA-Mutations (6), audit export/verify, vault reveal/PUT/DELETE, ip POST/DELETE, password PUT, gdpr approve/deny/download/erasure-preview/execute/request.

### i18n
- Namespace `security.*` **vollständig & paritätisch**: 83 Keys × 4 Sprachen (de/en/fr/it), 0 fehlend, 0 extra, **kein `{{var}}`**, kein `_one/_other`. Plus `admin.security.tabs.*` für neuen Hub.
- **Aber hardcoded Strings in Komponenten** (Raw-Strings, keine Keys):
  - Sessions: `'Alle'`, `'Meine'`, `'My Sessions'`, `'All Sessions'`
  - Vault: `'Secrets'`, `'sichtbar'`, `'Version'`
  - PasswordPolicy: `'Strong'/'Moderate'/'Weak'`, `'0 = no expiry'`, Test-Placeholder
  - IPAccess: `'Allow'`, `'Block'`, sr-only-Deutsch, `'Office network'`
  - Retention: `entry.type` + `entry.basis` (Gesetzesrefs) hardcoded deutsch
  - GDPR-Erasure: Modulnamen (`'CRM Kontakte'` …) hardcoded
- Live-Scan: **keine** sichtbaren Raw-Keys im UI (Namespace greift) — die Lücken sind die hardcoded Strings.

### Gefundene Bugs
1. **Handler-Response-Mismatch** (vault/gdpr-export/ip/password) → Crashes — **P0**
2. **Sessions-Crash** `deviceType.toLowerCase()` bei fehlendem Handler — **P0**
3. **2FA URL-Divergenz**: client `POST /auth/2fa/validate` vs handler `/auth/2fa/validate-login`
4. **`disable2FA.mutate(undefined)`** in `SecuritySettingsTab` — Hook erwartet TOTP-Code; Code-Input fehlt
5. **GDPR-Export Format-Select** ohne Effekt auf den API-Call
6. **Passwort-Ändern** in `SecuritySettingsTab` nur `toast.success`, kein API-Call
7. **GDPR-Erasure type-mismatch** `module` (Page) vs `module_name` (types)

### Vorbestehend (NICHT meine Lane, nur notiert)
- Dev-Server-Warnung: `@livekit/track-processors` unresolved (imported by `features/video/BackgroundSelector.tsx`) — video/meetings-Lane.

---

## Markt-Research (DSGVO/Security-Center — OneTrust, Usercentrics, Vanta, Drata, GitLab)

**Pflicht-Minimum für ein KMU-Security-Center (DACH):**
1. **Audit-Log** — Auth/CRUD/Rechte-Events, Filter, CSV-Export, append-only. ✅ bei uns vorhanden.
2. **DSAR-Workflow Art.15/17/20** — Intake (Typ-Dropdown Auskunft/Löschung/Export), **30-Tage-Frist-Countdown**, Status-Board, Cross-Modul-Suche nach E-Mail. ⚠️ teils da (DSAR-Suche + Erasure existieren, aber getrennt & kaputt).
3. **PW-Policy + 2FA** — Komplexität/Ablauf/Reuse, TOTP-Enrollment + Backup-Codes, Enforcement-Toggle + Grace-Period. ⚠️ Code da, Demo kaputt.
4. **Sessions** — aktive Sitzungen + Remote-Logout + „alle anderen beenden". ⚠️ Code da, Demo kaputt.
5. **Retention** — Löschklassen mit DACH-Fristen (§147 AO 8–10 J, §257 HGB, GoBD). ⚠️ statische Anzeige.

**Differenzierer:** Hash-Chain-Tamper-Evidence (Audit), kaskadierende Erasure-Preview über alle Module, Legal-Hold (§147 AO blockt Löschung), WebAuthn/Passkeys, **Verarbeitungsverzeichnis Art.30 (RoPA)** — letzteres fehlt komplett und ist laut Research ein hoch-sichtbarer DSGVO-Verkaufspunkt.

**Priorisierung (Research):** P1 = Audit + DSAR-Workflow + PW/2FA; P2 = Sessions + Retention-Config + Art.30 RoPA; P3 = WebAuthn + Hash-Chain + Legal-Hold.

---

## Phasen-Plan S-1…S-5 (bestätigt von Darien 2026-06-24, Scope „Medium" b)
> Schwerpunkt: **erst reparieren (Demo crashfrei), dann vertiefen.** Alles mock-first; 🔌-verdrahten-TODOs in `backend-gaps.md` (Abschnitt „security/DSGVO"). Contract gegen echtes BE abgeglichen (encoding/json, gewrappte Listen, snake_case).

- **S-1 · Crash-Fix + MSW-Contracts** ✅ **fertig (2026-06-24)** — siehe Log unten.
- **S-2 · MSW-Schreib-Ops + Demo-Tiefe** — POST/PUT/DELETE-Handler stateful (vault reveal/set/delete, ip-rules, password-policy, sessions terminate, audit export-Blob+verify) → Aktionen testbar.
- **S-3 · DSGVO-Kern** — Export/Erasure/DSAR geführte Flows, 30-Tage-Frist-Countdown, Status-Board, Cross-Modul-Suche, user_id→Name auflösen.
- **S-4 · Retention + Erasure-Tiefe** — Retention konfigurierbar (DACH-Vorlagen) + Erasure echte Preview/Receipt (statt setTimeout-Mock); Legal-Hold-Hinweis.
- **S-5 · Routing-Konsolidierung + i18n-Cleanup + Settings + Schluss-QA** — auf neuen Hub `/admin/security` konsolidieren, PW-Policy/DSAR/Retention als Sub-Tabs, `/admin/security-legacy`→Redirect, hardcoded Strings → Keys (×4), module-settings-registry-Eintrag, Screenshot-QA DE+EN.

### 🔭 Eigener späterer Batch (NICHT in S-1…S-5, aber nicht verlieren)
**Verarbeitungsverzeichnis Art. 30 DSGVO (RoPA)** — fehlt komplett. Laut Markt-Research ein hoch-sichtbarer DSGVO-Verkaufspunkt für DACH-KMU: CRUD mit Art.-30-Pflichtfeldern (Verantwortlicher, Zweck, Datenkategorien, Empfänger, Drittlandtransfer, Fristen, TOMs) + Branchen-Vorlagen + PDF-Export. → eigenes Feature einplanen (auch in MASTER-PLAN notiert).

### Phasen-Log · S-1 — ✅ fertig 2026-06-24
**Ursachen:** (1) Daten-Contract-Mismatch — Client erwartete nackte Arrays, echtes BE (encoding/json über protobuf) liefert **gewrappte Listen** (`{secrets}`, `{rules}`, `{export_requests}`, `{sessions}`, `{policies}`). (2) fehlende GET-Handler (Sessions, DSAR). (3) MSW-Demo-Daten mit falschen Feldnamen (Vault `key`→`key_name`, IP `type/cidr`→`rule_type/ip_cidr`, PW `require_numbers`→`require_digit`).
**Gemacht (BE-konform):** `security-client.ts` entpackt 6 GET-Funktionen; `handlers/security.ts` Demo-Daten an `security-types.ts` angeglichen + 3 neue Handler (sessions, sessions/all, dsar) + GDPR-Feld `exports`→`export_requests` mit 3 Demo-Anträgen; `handlers/auth.ts` fehlplatzierte (BE-inkonforme, nacktes-Array-)Session-Handler entfernt → Sessions kanonisch in security.ts; `SessionsPage.tsx` defensiver device_type-Guard.
**i18n-Klarstellung:** Paket-Annahme „i18n kaputt, Raw-Keys überall" **widerlegt** — 164 genutzte Keys ×4 Sprachen vollständig & paritätisch. Offen = nur hardcoded Strings in Komponenten → S-5.
**Verify:** Build ✓ (exit 0). Alle 9 Hub-Tabs + beide Hubs crashfrei, Screenshots angesehen (Vault/Sessions/IP/PW-Policy/GDPR-Export/Retention/DSAR/Erasure korrekt befüllt). 0 Raw-Keys, 0 Page-Errors.
**Offen:** GDPR-Export zeigt rohe `user_id` statt Name → S-3; Schreib-Ops noch nicht stateful → S-2.

### Phasen-Log · S-2 — ✅ fertig 2026-06-24
**Soll:** MSW-Schreib-Ops stateful → Aktionen testbar.
**Gemacht:** `handlers/security.ts` Demo-Daten zu modul-level `let` (in-memory, überlebt Navigation, reset bei Reload) + Schreib-Handler: Vault reveal (`GET /vault/:keyName`), set/upsert (`PUT`, key_version++), delete (`DELETE`); IP-Rules create (`POST`)/delete (`DELETE`); PW-Policy update (`PUT`, merge); Audit export (`GET /audit/export` → CSV/JSON-Blob) + verify (`POST /audit/verify`); Sessions terminate (`DELETE /auth/sessions/:id`) + terminate-all (`DELETE /auth/sessions`, behält is_current). Alle Response-Shapes BE-konform (`{secret}`/`{rule}`/`{policy}`/`{status}`/`{terminated_count}`).
**Verify:** Build ✓ (exit 0). Aktions-QA (`scripts/qa-security-actions.mjs`, Bilder angesehen): IP-Rule add → „3 Allow" + Toast „Erfolgreich" ✓; Vault-Reveal → `demo-secret-…` sichtbar ✓; Session-Terminate → „2 Meine" + Toast „Sitzung wurde beendet" ✓. 0 Page-Errors.
**Hinweis:** Vault-Reveal liefert Demo-Wert; echtes BE `GET /vault/:keyName` = `{secret, decrypted_value}` (Client erwartet `VaultSecretValue {key_name, value}`) → Abweichung in backend-gaps notiert.

---

## Verify-Setup (für Bau-Phasen)
- Dev-Server: `npm run dev` (electron-vite, Demo-Mode) — real auf **:5173** (5174 nicht erzwungen; kein Main-Server parallel). `DEV_BYPASS_AUTH` setzt admin-Profile → Hub-Guard erfüllt, kein Login.
- QA-Script: `desktop/scripts/qa-security.mjs` (beide Hubs, isolierte Tab-Tests mit Reload gegen Error-Boundary-Kleben). Crash-Detektion + Screenshots in `.qa-screenshots/security-*.png`.
- Server killen (Windows): `Get-NetTCPConnection -LocalPort 5173 | Stop-Process` (pkill greift nicht).
