# Resume-Prompt — Buchungsseite `zentria-demo` (Pilot Zentria) fertigstellen

> ✅ **SUPERSEDED (2026-06-16):** Diese Aufgabe ist erledigt — `zentria.tech/book/zentria-demo`
> ist end-to-end live (Fixes in Cosmi `a57d01f4` + website `4046d37`), der Kalender-Blocker
> wurde behoben und das Duplikat-Konto gelöscht. Die Booking-Admin-UI (vormals „WP-D") ist
> bereits gebaut (`api/booking-client.ts`, `useBookingPages`, `modules/kalender/booking/`).
> Datei nur noch als Historie. Offene Pre-Launch-Punkte siehe Memory `project_zentria_demo_booking`
> (Logo-URL-Daten) — Email-Normalisierung wurde am 2026-06-16 erledigt (Migration 000148).

> Diesen Block an den Anfang der nächsten Session einfügen.

## Rolle & Ziel
Du bist mein Coding-Partner für Cosmi (KMU Hub CRM, Firma Zentria). Sprich **Deutsch** (Umlaute/ß korrekt), Code/Commits Englisch. **Ziel:** eine funktionierende öffentliche Buchungsseite mit Slug `zentria-demo` im **Produktions**-Backend anlegen, damit `https://zentria.tech/book/zentria-demo` end-to-end testbar ist. Wir nehmen Zentria als ersten Piloten. Anlage soll über die **Desktop-Admin-UI** erfolgen (Modul Kalender → Tab „Terminbuchung" → „Neue Buchungsseite").

## Was bisher passiert ist (Vorsession)
Wir haben eine **prod-verbundene Desktop-App** gebaut und dabei **zwei echte, app-weite Bugs** gefunden+gefixt (beide nur gegen das echte Backend sichtbar, im Demo-Mock maskiert). Außerdem eine offene Sache (Kalender wird im UI nicht angezeigt) — das ist der **aktuelle Blocker**.

### Fix #1 (erledigt, UNCOMMITTED): falscher Kalender-API-Pfad
- `desktop/src/renderer/src/api/hooks/useCalendars.ts`: rief durchgängig `/api/v1/calendars…` statt `/api/v1/calendar/…` (Backend mountet unter `/api/v1/calendar`, `route_calendar.go:40`). 15 Pfade korrigiert (`/calendar/calendars`, `/calendar/browse`, `/calendar/preferences`, `/calendar/categories`).
- `desktop/src/renderer/src/mocks/handlers/calendar.ts` (Zeile ~26): Mock-Pfad nachgezogen.
- **Verifiziert:** `GET /api/v1/calendar/calendars` liefert jetzt 200; der persönliche Kalender wurde danach serverseitig auto-angelegt.

### Fix #2 (erledigt, UNCOMMITTED): doppelter Authorization-Header bei Mutationen → 401
- `desktop/src/renderer/src/api/utils/authenticatedFetch.ts`: bei POST/PUT/DELETE wurde der Idempotency-Key via `new Headers(headers)`-Roundtrip gesetzt → Keys lowercased → Objekt enthielt `Authorization` **und** `authorization` → `fetch` merged zu `Bearer x, Bearer x` → Server-Auth lehnt ab (401). Fix: Idempotency-Key direkt setzen (`headers['Idempotency-Key'] = generateIdempotencyKey()`), kein Roundtrip — in **beiden** Funktionen (`authenticatedRequest` + `authenticatedBlobRequest`); Import auf `generateIdempotencyKey` geändert. Empirisch bestätigt (node: Header kam als `"Bearer x, Bearer x"`).
- **Verifiziert:** Booking-POST ging von **401 → 400** (Auth ok, jetzt Validierungsfehler).

## AKTUELLER BLOCKER
`POST /api/v1/calendar/booking-pages` → **400** (war 401). Letzter Prod-Log-Beweis:
```
GET  /api/v1/calendar/calendars   → 200
POST /api/v1/calendar/booking-pages → 400
```
**Hypothese:** `calendar_id` ist leer, weil „Mein Kalender" im Booking-Editor-Dropdown nicht auswählbar ist. **Symptom (Luke):** Im Kalender-Modul erscheinen nur „Feiertage DE" + „Task-Deadlines", **kein eigener Kalender** — obwohl „Mein Kalender" in der DB existiert.

**Heißeste Spur:** Das Frontend filtert die Sidebar nach `c.group` (`'mine'|'shared'|'other'`, `KalenderPage.tsx:1242-1244`, `:492`, `:2338`), aber die Backend-Antwort `ListCalendars` (`CalendarWithMemberInfoProto`, via `response.JSON` = `encoding/json`, snake_case) liefert **kein `group`-Feld**. Die client-injizierten Pseudo-Kalender (holidays/deadlines) haben `group`, die echten nicht → echte Kalender fallen raus. Der Booking-Dropdown (`BookingPageEditor.tsx`) nutzt zwar `calendarsData.calendars` **ohne** group-Filter — also prüfen, ob er den Kalender überhaupt bekommt.

## Erste Schritte nächste Session (in dieser Reihenfolge)
1. **Genauen 400-Grund holen** (welches Feld?): in der App DevTools öffnen (falls aktiv) → Network → fehlgeschlagenen POST → Response-Body (`{error,code,details}`). Alternativ mit gültigem Token reproduzieren. ⚠ Wegwerf-Test-User-Reproduktion wurde vom Auto-Classifier geblockt → Luke vorher explizit autorisieren lassen.
2. **Raw-JSON von `GET /api/v1/calendar/calendars`** für lukeleonhoppe@gmail.com ansehen (DevTools/Token) → enthält die Antwort „Mein Kalender" mit `id`+`name`? Fehlt `group`?
3. **Adapter prüfen:** wo das Frontend `group` an Kalender vergibt (`KalenderPage.tsx` ~Z.355-375 + evtl. `api/hooks/useCalendars.ts`/Adapter). Echte Kalender müssen `group='mine'` bekommen (aus `owner_id === currentUserId`). Fix bauen → `BookingPageEditor`-Dropdown muss „Mein Kalender" zeigen → Save erneut testen.
4. Wenn Save grün: **verifizieren** `GET /api/v1/public/booking-pages/zentria-demo` (aktuell 404) → 200 + Services; `…/availability?date_from=…&date_to=…` → Slots; dann **`https://zentria.tech/book/zentria-demo`** end-to-end.
5. **Committen** (Conventional Commits, direct-to-main): Fix #1 (calendar path), Fix #2 (auth header), + ggf. der group/Adapter-Fix. Das sind app-weite Bugs (Fix #2 betraf ALLE Mutationen).

## Umgebung / Fakten (verifiziert)
- Repo: `C:\Users\Luke\Documents\KMU Hub`
- Prod-SSH: `ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195`
- Container: Postgres `docker-postgres-1`, Gateway `docker-gateway-1`
- psql (Superuser, bypassed RLS): `docker exec docker-postgres-1 psql -U kmuhub -d kmuhub -c "…"`  (⚠ `docker exec -i` NICHT mit `-c` mischen — verschluckt stdin)
- Tenant (single): `00000000-0000-0000-0000-000000000001`
- **Prod-Logs = Ground Truth für Statuscodes:** `ssh … 'docker logs --since 20m docker-gateway-1 2>&1 | grep -E "booking-pages|calendar/calendars" | tail'`. Response-Bodies brauchen Token/DevTools.
- **Auto-Classifier blockt Prod-Writes** (Test-User, direkte Inserts) ohne explizite User-Autorisierung pro Ziel.

### Accounts (Prod)
- `lukeleonhoppe@gmail.com` = `818d8531-907a-484a-9ea9-7827f661038f` → **admin+member** (in dieser Session hochgestuft).
- ⚠ **Duplikat:** `LukeLeonHoppe@gmail.com` (Groß-/Kleinschreibung!) = `a8a540ad-7bfe-407e-babf-c04604683503` → admin+member. Login ist **case-sensitiv** (`postgres_repository.go:36 WHERE email = $1`, keine Normalisierung). Vor Launch konsolidieren (einen behalten/löschen) — mit Luke klären.

### Kalender (Prod)
- „Mein Kalender" = `5165bfe1-188c-46ed-b07f-84aa2568e3bb` (personal, owner `818d…`, tenant `…0001`, is_default=true) — **existiert in DB**, wird aber im UI nicht angezeigt. ← Kern des Blockers.

### App-Build (prod-verbunden)
- `desktop/.env.production` → `RENDERER_VITE_API_URL=https://app.zentria.tech`, **kein** Demo-Mock.
- Bauen: `npm --prefix "C:\Users\Luke\Documents\KMU Hub\desktop" run build:win` → Installer + `desktop\dist\win-unpacked\Cosmi.exe`. **Vor jedem Rebuild Cosmi schließen** (Dateisperre). `npm run dev` = Demo-Mock (nicht für Prod).
- ⚠ Der **aktuell laufende** Build enthält Fix #1 + #2 (GET 200, POST 400). Bei Code-Änderung neu bauen + `dist\win-unpacked\Cosmi.exe` neu starten.

## Soll-Werte der Buchungsseite (für den UI-Editor)
Slug `zentria-demo` · Firma `Zentria` · Service 1: „Kostenloses Erstgespräch" 30 min `0.00` „Unverbindliches Kennenlernen." · Service 2: „Beratung" 60 min `120.00` · Wochentage Mo–Fr · Zeitfenster 09:00–17:00 · Slot 30 · Puffer 0 · Vorlauf 24 h · keine Pausen.
