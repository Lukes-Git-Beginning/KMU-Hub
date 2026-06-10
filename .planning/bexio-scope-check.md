# Bexio-Scope-Check gegen Welle 7 (BACKEND-LAUNCH-PLAN.md) — 2026-06-10

> Read-only-Audit (Explore-Subagent) von `backend/internal/biz/bexio/` gegen die Welle-7-Definition:
> „OAuth2-Connect + bidirektionaler Sync-Service (Rechnungen/Kontakte) + Mapping-Settings.
> Verifikation: OAuth-Token-Refresh, Sync-Idempotenz, Mapping-Roundtrip."
> **Fazit: Gerüst ist substanziell weiter als angenommen — keine Green-Field-Welle, sondern eine
> Härtungs-/Fertigstellungs-Welle (~2–3 Tage statt „mittel/neu").** 5 Gaps blockieren 01.09.

## Ist-Stand (Kurzfassung, Belege Datei:Zeile)

| Bereich | Status | Beleg |
|---|---|---|
| OAuth2 Authorize-URL | ✅ | `bexio/oauth_handler.go:65`, Route `route_bexio.go:47` |
| Token-Exchange (code→token) | ✅ | `bexio/auth.go:139`, Callback `route_bexio.go:39` |
| Token-Refresh (auto, <30s-Fenster + 401-Retry) | ✅ | `bexio/auth.go:55,68`, `client.go:109,172` |
| Kontakte-Sync | ✅ bidirektional (Last-Write-Wins) | `contact_sync.go:66,113,116,190` |
| Rechnungen | ⚠ nur outbound Push; Zahlungen inbound via PaymentPoller | `invoice_push.go:39`, `payment_poll.go:38` |
| Offerten | ✅ outbound Push | `quote_push.go:39` |
| Sync-Logs + Entity-Mappings + Field-Mappings | ✅ Migration 000055, RLS seit 000122/125 | `postgres_repository.go:112–268` |
| Gateway-Wiring | ✅ 10 Routen registriert | `route_bexio.go`, `cmd/gateway/main.go:236` |
| gRPC | ✅ 11 RPCs implementiert (kein Stub) | `server/bexio_grpc.go`, `cmd/biz/main.go:279` |
| Credentials | ✅ Refresh-Token in Vault (AES-256-GCM/HKDF) | `auth.go:52`, `security/vault/crypto.go` |
| Tests | ⚠ Service-Mocks + DB-Isolationstests ok (korrekt via `internal/testutil`, keine Build-Tags); Kern-Sync-Logik UNGETESTET | `service_test.go`, `tenant_isolation_phase2/3_test.go` |
| Feature-Flag | ⚠ nur Env-Gate `BEXIO_CLIENT_ID != ""`, kein Registry-Flag | `cmd/biz/main.go:253` |

## Gap-Liste

| # | Gap | Aufwand | Blocker 01.09 |
|---|---|---|---|
| G1 | **OAuth-State ohne CSRF-Schutz**: Callback nutzt `state` direkt als `tenant_id` (kein Nonce/HMAC/Expiry) — Angreifer kann beliebige tenant-UUID einschleusen. `route_bexio.go:97` | S | **JA (Security)** |
| G2 | **ContactService=nil im Wiring**: `cmd/biz/main.go:274` übergibt `nil` → Outbound-Contact-Sync + Inbound-Create = Nil-Pointer-Panic zur Laufzeit. CRM-Service braucht zudem `ListModifiedSince` fürs Interface. | S | **JA (funktional)** |
| G3 | **`resolveContactBexioID` stub-artig**: nimmt erstes verfügbares Contact-Mapping statt CustomerEmail→Contact-Lookup (`invoice_push.go:132`, `quote_push.go:132`) → falscher Bexio-Kontakt an Rechnungen. | M | **JA (Mapping-Roundtrip)** |
| G4 | RevokeTokens lässt Vault-Key orphaned (`auth.go:201`). | S | nein |
| G5 | **vaultSvc=nil wenn `VAULT_MASTER_SECRET` fehlt** (`cmd/biz/main.go:268`) → Panic beim ersten Token-Write statt Start-Validierung. | S | **JA** |
| G6 | Kein HTTP-Endpoint für Sync-Config (Intervalle/Flags nur Defaults beim Callback). | M | nein |
| G7 | `GetBexioConnectionStatus.org_name` wird nie befüllt (`bexio_grpc.go:88`). | S | nein |
| G8 | Scheduler single-tenant (`scheduler.go:52` nimmt nur erste Config). | M | nein (Pilot ok) |
| G9 | Kein Feature-Flag im Registry (FE kann Bexio-Status nicht abfragen). | S | nein |
| G10 | **Kern-Sync-Logik ungetestet** (ContactSyncer/InvoicePusher/PaymentPoller/TokenManager) — Welle-7-Verifikation „Sync-Idempotenz" nicht belegbar. | L | **JA (Verifikation)** |
| G11 | Outbound-Contact-Sync läuft nur bei Delta, nicht beim ersten Full-Sync (`contact_sync.go:116`) — vertretbar, undokumentiert. | S | nein |
| G12 | **Invoice-Pull fehlt** — Rechnungen nicht bidirektional (nur Push + Payment-Poll inbound). | L | **Klärung nötig** |

## Empfohlene Bau-Welle (Folge-Session, Reihenfolge)

1. **G2** ContactService verdrahten (`cmd/biz/main.go` + `ListModifiedSince` am CRM-Contact-Service)
2. **G1** OAuth-State als HMAC-signiertes Kurzzeit-Token (tenant_id+expiry, Secret aus Vault-Master)
3. **G5** Startup-Guard: ohne `VAULT_MASTER_SECRET` Bexio nicht aktivieren
4. **G3** Echter CustomerEmail→Contact-UUID→EntityMapping-Lookup
5. **G10** Unit-Tests: Last-Write-Wins, Push-Idempotenz, already-paid-Skip, Refresh-Rotation
6. **G4** `vault.DeleteSecretByKey` + Aufruf in RevokeTokens
7. Optional ≤01.09: G6, G7, G8, G9

## ⚠ Offene Frage für Darien (Produktentscheidung)

**G12:** Welle 7 sagt „Rechnungen/Kontakte bidirektional". Ist für Rechnungen wirklich ein Pull
Bexio→Cosmi gemeint (in Bexio erstellte Rechnungen importieren)? Oder reicht Push + PaymentPoller
(Zahlungsstatus inbound)? Letzteres entspricht dem Symbiose-Modell („Cosmi macht Vorkette, übergibt
an Bexio") und wäre deutlich kleiner.
