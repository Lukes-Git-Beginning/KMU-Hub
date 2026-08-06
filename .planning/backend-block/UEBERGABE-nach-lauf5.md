# Übergabe — offene Punkte nach Nachtlauf 5 (Stand 2026-08-06)

Kopiervorlage für die nächste Session. Alles unten ist verifiziert, nicht vermutet.

---

## Ausgangslage

`backend-loop` steht auf `f05b1551`, **PR #18 ist als Draft offen**, CI komplett grün
(CI: Lint/Test/E2E/Validate OpenAPI, CI Desktop: Checks/Build — alle mit echten Step-Counts,
keine Billing-Wall). Der Branch ist 58 Commits vor `origin/main`, 0 dahinter.

Der Gegencheck des Nachtlaufs ist durch. Vier Funde wurden bereits behoben und liegen auf dem
Branch: NULL-`notes` in der Personalakten-Projektion, `tenant_id`-Leak auf der öffentlichen
Formular-Antwort, nie gesetzte `ip_address` ebendort, kaputte OpenAPI-Response bei
`/hr/personnel-documents`. Migrationen 288–296, Prod-Kopf ist **287**.

Drei Dinge sind bewusst liegen geblieben und gehören in diese Session — **in dieser Reihenfolge**,
damit nur einmal deployt wird.

---

## A) Vor allem anderen: CSAT-Versand gegen Produktion prüfen (Deploy-Gate)

**Der Befund.** `DefaultCsatConfig()` (`backend/internal/helpdesk/csat_config.go:39-46`) liefert
`Enabled: true`. Ein Tenant, der CSAT nie konfiguriert hat — und das sind nach dem Deploy **alle** —
bekommt Umfragen also standardmässig eingeschaltet. Beim Ticket-Close wird ein Token erzeugt und
eine Mail eingeplant, deren Link aus `CSAT_SURVEY_BASE_URL` + Token besteht; der Default ist
`https://app.zentria.tech/csat`.

**Dort liegt nichts.** `deploy/docker/Caddyfile` proxyt `app.zentria.tech` vollständig auf
`gateway:8080` — es wird kein statisches Frontend ausgeliefert. Cosmi ist eine Electron-App, eine
öffentliche Weboberfläche existiert nicht. Die tatsächliche API-Route ist
`POST /api/v1/public/helpdesk/csat/{token}`, also nichts, was ein Kunde anklicken kann.
Ein zugestellter Umfragelink führt damit auf einen blanken API-404.

**Was das entschärft.** Der Dispatcher startet nur, wenn SMTP konfiguriert ist
(`backend/cmd/helpdesk/main.go:128` — `if systemSender.Configured()`). Ist es das nicht, werden
Tokens zwar erzeugt, aber nichts versendet, und der Service loggt eine Warnung.

**Zu tun, bevor irgendetwas gemergt wird:**

```bash
ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195
grep -E 'SYSTEM_SMTP_(HOST|FROM|USER)' /opt/kmuhub/.env.production
```

- **Nichts gesetzt** → kein Versand, Deploy ist unbedenklich. Trotzdem in `.env.production`
  `CSAT_SURVEY_BASE_URL` bewusst setzen, bevor SMTP später scharf geschaltet wird.
- **Gesetzt** → **nicht deployen**, ohne vorher eine der drei Optionen zu wählen:
  1. `DefaultCsatConfig().Enabled` auf `false` drehen (Opt-in statt Opt-out) — kleinster Eingriff,
     ändert nur eine Zeile plus Test;
  2. den Dispatcher-Start zusätzlich hinter ein Feature-Flag hängen;
  3. eine öffentliche CSAT-Seite bauen — das ist ein eigenes Projekt, nicht diese Session.

Dieselbe Frage betrifft die Wiki- und Formular-Links, aber entschärft: die entstehen nur, wenn
ein Mensch sie bewusst erzeugt. CSAT feuert von allein bei jedem Ticket-Close.

---

## B) Wiki-Share-Tokens widerrufbar machen

**Der Befund.** `wiki_share_tokens` (Migration 000076, `tenant_id` per RLS-Retrofit ergänzt) hat
**kein `revoked_at`** — anders als `report_share_tokens` (000252), `document_share_links` (000266)
und die neuen `form_share_tokens` (000293), die es alle haben. Live-Shape:

```
id, article_id, token (UNIQUE), expires_at (nullable), permissions text[], created_at, tenant_id
RLS: tenant_isolation (tenant_id = current_tenant_id() OR is_system_context()), FORCE
```

`ShareToken.Usable()` (`backend/internal/wiki/share.go:61-66`) prüft nur `expires_at` und ob der
Token `read` gewährt. Die Routen (`backend/internal/gateway/route_wiki.go:87-88`) bieten **List**
und **Create**, keine Revoke — der Kommentar darüber behauptet allerdings „revoking is part of
managing them", was schlicht nicht stimmt.

Vor Lauf 5 war das folgenlos: die Tokens waren tot, es gab keinen Einlöseweg. Der Lauf hat
`POST /api/v1/public/wiki/articles/{token}` gebaut und sie damit zu einem von aussen einlösbaren
Credential gemacht — das man nicht zurückholen kann. `expires_at` ist nullable, heisst: ein Link
ohne gesetztes Ablaufdatum lebt ewig. Der einzige Ausweg heute ist, den Artikel zu
depublizieren, was ihn auch intern unsichtbar macht.

**Umfang.** Eine Einheit, dem Muster von `form_share_tokens` folgen:

1. Migration **000297** — `revoked_at TIMESTAMPTZ NULL` auf `wiki_share_tokens`. Weicher Widerruf,
   nicht DELETE: „dieser Link wurde gekappt" und „diesen Link gab es nie" sind für einen Autor
   verschiedene Tatsachen (Begründung steht ausformuliert in 000293). Bei der Gelegenheit prüfen,
   ob `created_by` mitsoll — heute steht nirgends, wer einen Link erzeugt hat.
2. `Usable()` um `revoked_at IS NULL` erweitern, Log-Zeile in `RedeemShareToken`
   (`share.go:110-120`) um `revoked` ergänzen — sie nennt heute nur `expired`/`grants_read`.
3. `RevokeShareToken`-RPC in `proto/wiki/v1/wiki.proto` + Service + gRPC-Server, danach **regenerieren**
   (`.proto` und `.pb.go` in denselben Commit).
4. Route `DELETE /api/v1/wiki/articles/{id}/share/{tokenId}` hinter `wikiShare`
   (`route_wiki.go:58` — `wiki:articles/write` ODER `wiki:share_token/create`; falls ein eigener
   Revoke-Key gewünscht ist, braucht er eine Seed-Migration, sonst 403 für alle inkl. Admin).
   Den irreführenden Kommentar bei `route_wiki.go:85-86` mitziehen.
5. `openapi.yaml` ergänzen — und **zusätzlich** zum Go-Test
   `npx --yes @apidevtools/swagger-cli validate backend/api/openapi.yaml` fahren:
   `TestOpenAPIRouteDrift` prüft nur Routen gegen Pfade, nie das Schema.
6. Test: ein widerrufener Token wird mit demselben 404 abgewiesen wie ein erfundener, und der
   Widerruf greift cross-tenant nicht daneben.

---

## C) `ListEmployeeDocuments` — toter Rollen-Parameter

`backend/internal/server/hr_grpc.go:906-907` ruft den Service mit hartkodiertem
`callerRole = "admin"`, Kommentar „gateway will enforce actual user role". Das Gateway tut es
nicht: `route_hr.go:168` schützt die Route nur mit `RequirePermission("hr", "read")` und reicht
keine Rolle durch. Der applikative Tier-Filter in `ListByEmployee` ist damit auf diesem Pfad
wirkungslos; dass nichts leakt, hängt allein an der RLS-Policy `hr_document_access` (000127/000128).

Bestandsfund, nicht von Lauf 5 verursacht. Die Entscheidung ist eine Produktfrage, keine
technische: **soll** der Tier-Filter auf diesem Pfad greifen?

- **Ja** → die Rolle des Aufrufers aus dem Kontext durchreichen, so wie der neue tenantweite Pfad
  es tut.
- **Nein, RLS reicht** → den `callerRole`-Parameter und den toten Filter ersatzlos entfernen,
  damit kein zweiter, driftender Filter neben der Policy stehen bleibt. Das ist die schlankere
  Variante und die, die der neue Pfad bereits vorlebt (`ListByTenant` hat bewusst keinen).

---

## D) Merge und Deploy

Erst wenn A geklärt und B/C entschieden sind — dann einmal deployen statt zweimal.

1. Draft-Status aufheben, PR #18 mergen. CI läuft auf `main` erneut, danach zündet **CD
   automatisch** (self-hosted Runner, `cd.yml` auf `workflow_run: CI completed, branch main`).
2. CD beobachten, nicht annehmen. Bekannte Fallstricke:
   - `deploy.sh` Auto-Rollback macht `git checkout <sha>` → **detached HEAD**; `git pull` zieht
     danach nicht mehr nach, migrate läuft in eine Schleife, 503. Recovery:
     `git checkout main && git merge --ff-only`.
   - Das MinIO-Backup scheitert beim Deploy regelmässig (`non-critical`), der DB-Dump läuft.
   - Der Smoke zählt „LiveKit probe skipped" als PASS — beim Lesen daran denken.
3. Migrationskopf verifizieren: **287 → 296** (bzw. 297, falls B mitgeht).
   ```bash
   ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195
   sudo docker compose --env-file /opt/kmuhub/.env.production \
     -f /opt/kmuhub/deploy/docker/docker-compose.yml run --rm migrate version
   ```
4. Erwartung festhalten: **`scans.yml` bleibt auf `main` rot.** Rot macht ihn `npm-audit`
   (ohne `continue-on-error`) wegen **react-router im Frontend**. Die Go-CVEs laufen unter `trivy`
   mit `continue-on-error: true`. Die Bumps aus Lauf 5 machen ihn also **nicht** grün — das ist
   kein Fehlschlag.

---

## Gate-Kommandos

Wortlaut steht in `.planning/backend-block/loop/GATE-COMMANDS.md`. Das Wesentliche:

```bash
export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"
export DATABASE_URL="postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable"
```

Rolle ist **`kmuhub_app`**, nie `kmuhub` (Superuser hat `BYPASSRLS`, RLS-Tests bestünden dann ohne
Aussage). Ohne gesetztes `DATABASE_URL` überspringt `SkipIfNoDB` **alle** DB-Tests und der Lauf
meldet trotzdem `ok`. Keine Pipes um Gate-Kommandos (`| head` macht den Exit-Code immer 0).
`go build ./...` läuft auf dieser Maschine in ein OOM — immer `-p 2` und gezielt auf Pakete.
Docker Desktop muss laufen, `docker-postgres-1` muss oben sein.

## Was nicht in diese Session gehört

- Eine öffentliche Weboberfläche für die sieben Public-Token-Routen. Eigenes Projekt.
- `ticket_number` im Frontend: das Backend liefert das Feld auf jeder Ticket-Response,
  `helpdesk-types.ts` kennt es nicht und `helpdesk-adapters.ts:124-137` baut eine
  Hash-Pseudonummer. Reiner FE-Fix, gehört in eine Frontend-Session.
- `decodeAndValidate` auf `DisallowUnknownFields` umstellen — repo-weite Entscheidung mit
  Breaking-Potenzial für jeden Client, der Zusatzfelder sendet.
- Die drei `blocked`-Units im Backlog (`g-hr-salary-statements`, `g-admin-billing`,
  `fe-projects-guest-overview`) warten auf eine Produktentscheidung, nicht auf Code.
