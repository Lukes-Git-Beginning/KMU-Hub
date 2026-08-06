# FE↔Backend-Verknüpfungs-Inventur (Stand 2026-06-22)

> Was lässt sich auf `main` schon ans Backend anschließen, was braucht noch Backend — als Vorarbeit für die gebündelte FE→BE-Wiring-Phase ([[project_fe_be_wiring_phase]]).
> Quellen: Code-Inventar Backend (`backend/cmd` 24 Services, `backend/internal/gateway` 56 `route_*.go`, `openapi.yaml` 356 Pfade) + Frontend (`api/hooks`, `*-client.ts`, `stores/*`). Ergänzt das modul-zentrierte `backend-gaps.md` um die Verknüpfungs-Sicht.

## Kernbefund (überraschend gut)

1. **Backend ist auf main quasi vollständig.** Jedes Fachmodul hat einen echten gRPC-Service + Gateway-Routen. **Keine echten Modul-Stubs** — nur 3 chirurgische 501er (CRM-Tag-Mutationen, Advisory-PDF).
2. **Frontend ist fast überall „Kategorie B" = mock-first verdrahtungsbereit.** Die Hooks rufen bereits die echten Endpunkte auf (`apiClient` oder `authenticatedRequest`), nur der MSW-Demo-Layer fängt sie ab. **Anschließen = `RENDERER_VITE_DEMO_MODE=false`** — kein Hook-Umbau.
3. Der eigentliche Aufwand verteilt sich auf **drei kleine, klar abgegrenzte Resthaufen**: Feature-Flags scharfschalten, OpenAPI-Lücke schließen, und die echten „kein-Backend"-Stellen (Kat. C/D) bauen.

## Kategorie 1 — Sofort anschließbar (Demo-Flag aus, ggf. Feature-Flag an)

Diese Module funktionieren beim Umlegen des Demo-Schalters gegen das echte Backend — Hooks unverändert:

| Modul | Backend | Hinweis |
|---|---|---|
| **Auth** | ✅ läuft schon real | Login/Refresh sind bereits live |
| **CRM/Kontakte** | ✅ im OAS | Tag-Mutationen (3 Endpoints) noch 501 → siehe Kat. 3 |
| **Chat/Kommunikation** | ✅ im OAS | Kern komplett; Bookmarks/Unread-Inbox/PATCH = Kat. 3 |
| **Work (Projekte/Tasks/Zeit)** | ✅ im OAS | komplett inkl. Timer |
| **Kalender + Booking** | ✅ im OAS | ⚠ Shape-Adapter (`flattenCalendarList`) — Kat. 2 |
| **Video/Meetings** | ✅ im OAS | braucht LiveKit-Keys in Prod |
| **Finanzen** | ✅ im OAS | DATEV/GoBD/E-Rechnung real |
| **HR/Team/Zeiterfassung** | ✅ im OAS | Payroll-Lohnlauf = Kat. 3 |
| **Notifications** | ✅ im OAS | Quiet-Hours/DND/Mute = Kat. 3 |
| **Dokumente** | ✅ im OAS | MinIO-Public-Endpoint in Prod nötig |
| **Inbox (Posteingang)** | ✅ im OAS | Threading/Status/Tags = Kat. 3 |
| **Automatisierung** | ✅ im OAS | immer aktiv |
| **Dashboard / Global Search** | ✅ im OAS | Team-Scope = Kat. 3 |

**Hinter Feature-Flag (Backend Default-OFF — Env-Var `COSMI_MODULE_*_ENABLED=true` zum Scharfschalten, Code ist fertig):**
Wiki, Helpdesk, Schichten, Fuhrpark, Einkauf, Produktion, Berichte, Formulare, Rapporte, Inventar, Vermietung, Vertraege. Dazu Integrationen Bexio (Flag) + Lexware.

## Kategorie 2 — Anschließbar, aber Feinschliff nötig

- **OpenAPI-Lücke:** ~8 Module + Integrationen (Berichte, Formulare, Rapporte, Inventar, Vermietung, Vertraege, Security, Dialer) haben **keine OAS-Doku**. Ihr FE nutzt `authenticatedRequest` (funktioniert real!), aber nicht den typgenerierten `apiClient`. → Anschluss geht, aber Typsicherheit/Spec-Validierung fehlt. Empfehlung: OAS nachziehen (Luke), dann Hooks optional auf `apiClient` migrieren.
- **Shape-Adapter prüfen:** `useCalendars` (`flattenCalendarList`) + HR-Employees (`adaptEmployee`) gleichen schon MSW-flach vs. gRPC-nested ab — beim echten Swap verifizieren, dass die Adapter greifen.

## Kategorie 3 — Braucht noch echtes Backend (die echten Lücken)

**D — Endpoint existiert nur im Mock (kein Backend/kein OAS):**
| Stelle | Endpoint | Modul |
|---|---|---|
| Chat-Lesezeichen | `GET /messages/bookmarks`, `POST /messages/{id}/bookmark` | kommunikation (KO-4) |
| Chat-Unread-Inbox | `GET /messages/unread-inbox`, `POST /channels/read-all` | kommunikation (KO-7) |
| Chat-Kanal-Edit | `PATCH /channels/{id}` | kommunikation (KO-9) |
| Notifications | quiet-hours / dnd / mutes / pin / dismiss | notifications |
| Kontakt-Tags | `POST/PATCH/DELETE /tags` | crm |

**C — reine Client-Stores (localStorage), brauchen Backend-Feld/-Endpoint:**
- **Inbox:** Threading (`ListThreadMessages`), Status-Feld, Tag-RPC, Canned-Responses-CRUD
- **CRM:** Leads (`/api/v1/leads`), `segment_override`, Advisory-Protocols-Persistenz (revisionssicher!), Lead-Scoring-Regeln
- **HR/Payroll:** komplettes Lohnvorbereitungs-Modul (`payroll_runs` + DATEV-LODAS-Export) — siehe `team-datev-lohn-spec.md`
- **Wiki:** Pin-State, Tenant-Settings
- **Work:** Task-Label-Zuweisung*, Workflow-Settings (*Backend laut `backend-gaps.md` evtl. schon erledigt → nur FE-Wiring offen)
- **Dashboard:** Team-Scope-Layout

> ⚠ Mehrere Kat.-C-Punkte sind laut `backend-gaps.md` backendseitig bereits ERLEDIGT (Reactions, Settings-Fundament, Work-Labels/Custom-Fields, Advisory-Tabelle, Beratungsprotokoll-RPCs). Dort fehlt nur noch das **FE-Wiring** (Store → vorhandener Hook). Vor jeder Kat.-3-Arbeit gegen `backend-gaps.md` ✅-Markierungen abgleichen, sonst doppelte Backend-Arbeit.

## Empfohlenes Vorgehen für die Wiring-Phase (später, nach allen FE-Batches)

1. **Smoke-Swap:** ein Modul ohne offene Lücken (z.B. **Work** oder **CRM-Kern**) mit `DEMO_MODE=false` gegen lokales Backend testen → Auth/Refresh/Shape-Realität verifizieren.
2. **Feature-Flags** der fertigen Module in der Ziel-Umgebung setzen.
3. **OAS nachziehen** (Luke) für die 8 undokumentierten Module → Hooks auf `apiClient` migrieren (Typsicherheit).
4. **Kat. 3 abarbeiten** — aber zuerst gegen `backend-gaps.md`-✅ filtern (vieles ist nur FE-Wiring, kein neues Backend).
5. **Prod-Voraussetzungen:** MinIO-Public-Endpoint (`s3.zentria.tech`), LiveKit-Keys, CORS/CSP für OnlyOffice.

## Grobe Zahlen
- ~27 Fachmodule **verdrahtungsbereit** (Kat. A/B).
- ~12 davon **Backend Default-OFF** (Feature-Flag-Schalter, Code fertig).
- **5 D-Stellen** (Mock-only-Endpoints) + **~10 C-Bereiche** (Client-Stores) = die echte Backend-Restarbeit, teils schon erledigt (nur FE-Wiring).
