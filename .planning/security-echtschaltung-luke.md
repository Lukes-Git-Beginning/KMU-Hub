# security/DSGVO — Echt-Schaltungs-Paket (delegiert an Luke, 2026-06-28)

> **Warum Luke:** Luke kennt das security-Backend (selbst gebaut), das Modul ist security-kritisch,
> und die heiklen Lösch-/Vault-Pfade brauchen Backend-Verständnis + Verantwortung. Darien hat das
> Backend read-only verifiziert (s.u.), die eigentliche Echt-Schaltung ist ein eigener Block.

## Ziel
Das security-FE (10 Seiten, `modules/security/`) von **MSW-Mock** auf das **echte Backend**
umstellen — analog zu kontakte/helpdesk/inbox (Referenz: `.planning/kontakte-mock-exit-DONE.md`).

## Stand FE (S-1…S-5, mock-first gebaut, gemergt `43fecf37`)
- Seiten: `AuditLogPage` · `SessionsPage` · `IPAccessPage` · `RetentionPolicyPage` · `PasswordPolicyPage` · `DSARSearchPage` · `GDPRExportPage` · `GDPRErasurePage` · `TwoFactorSetupWizard` · `SecurityAdminPage` (Hub `/admin/security`, 10 Tabs)
- Client/Hooks: `api/security-client.ts` · `api/hooks/useSecurity.ts` · `useSessions.ts` · `use2FA.ts`
- Mock: `mocks/handlers/security.ts` (läuft aktuell, Demo-Mode)

## Stand Backend (✅ verifiziert echt — Darien 28.06., read-only)
- **~25 Endpoints**, `internal/gateway/route_security.go` → gRPC `internal/server/security_grpc.go` (läuft im **auth**-Binary, `ServiceName()="auth"`).
- DB-Tabellen existieren alle: `audit_log` · `ip_access_rules` · `password_policies` · `retention_policies` · `vault_secrets` · gdpr `data_exports`.
- Live geprüft (alle HTTP 200): `GET /security/audit`, `/password/policy` (echte Row, min_length 12 etc.), `/ip-rules`, `/retention-policies`, `/gdpr/exports`. → **kein Stub.** `audit`/`retention` nur leer (keine Demo-Daten).

## Aufgabe (Echt-Schaltung)
1. **security-client.ts**: Demo-Mode-Branch / MSW → echtes Backend (Muster: `mocks/demo-mode-flag.ts` + `api/casing.ts`).
2. **Wire-Shape-Adapter** (das verdeckt der Mock — siehe inbox/helpdesk-Lessons): `response.JSON` über Proto ⇒ Timestamps als `{seconds,nanos}` → ISO (Helper `api/wire-time.ts` `normalizeWireTimestamps`), Enums als **Int** → String mappen (Muster: `inbox-client.ts` `normalizeMessage`, `helpdesk-client.ts` `unwrapList`). Leere Listen kommen als `{}` (protojson Null-Omission) → `?? []`. **Ausnahme:** `GET /security/dsar/search` liefert das Gateway bereits **flach transformiert** (`HandleDSARSearch` baut `{results:[{...,modules:[{records:[{key:value}]}]}]}`) — kein Adapter nötig.
3. **Demo-Seeds** (`backend/seeds/demo/security-demo.sql`, idempotent, tenant …0001): ein paar `audit_log`-Einträge, 1–2 `ip_access_rules`, 1–2 `retention_policies`, 1 Test-`vault_secrets`, 1 `data_exports` — sonst zeigen alle 10 Tabs Empty-States.
4. **Live-QA über alle 10 Seiten** (Playwright localbackend, Muster `desktop/scripts/qa-*-localbackend.mjs`), mock-verdeckte Bugs fixen (erfahrungsgemäß 2–4).

## ⚠ Sensible / destruktive Pfade — NICHT scharf live testen
- **`GDPRErasurePage` → `POST /security/gdpr/erasure/execute`** löscht **echte User-Daten** (braucht `admin_password`). Nur `POST /erasure/preview` live testen + Code-Pfad prüfen. Erasure selbst gegen einen Wegwerf-Test-User, nie gegen Demo-Admin.
- **`vault_secrets`**: echte Secrets. Nur Test-Secret seeden, nie Prod-Werte.
- **`PasswordPolicyPage` PUT** + **GDPR-Export Approve/Deny**: Write-Pfade mit Bedacht.

## Backend-Restpunkte (falls beim Wiring auffällig)
- `audit`/`retention` ohne Seed leer → nur Daten-, kein Code-Problem.
- 2FA (`use2FA.ts`) + Sessions (`useSessions.ts`) gegen echte Endpoints gegenchecken (separat vom Security-Hub).
- Art.30 RoPA war als FE-Folge-Batch offen (RESUME #3) — separat.

## Referenzen
- Echt-Schaltungs-Muster: `.planning/kontakte-mock-exit-DONE.md` · Wire-Adapter: `api/inbox-client.ts` (`normalizeMessage`), `api/helpdesk-client.ts` (`unwrapList`)
- Backend-Gaps-Kontext: `.planning/backend-gaps.md` (28.06.-Block, security-Befund)
