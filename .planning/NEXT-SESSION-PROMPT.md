# Session-Ziel: Gate 2 abnehmen, dann Etappe 3 (G1)

Stand: **2026-08-17.** Es gibt **kein Launch-Datum** — das Ziel ist **Produkt 1.0.0** nach
Reifegrad-Gates. Nennt dir ein Dokument ein Datum, ist das Dokument veraltet, nicht die Lage.

**Etappe 2 ist gebaut.** Cosmi läuft als Web-App:

```
$ curl -sS -o /dev/null -w "%{http_code}" https://app.zentria.tech/
200          ← die React-App, kein Installer, keine Code-Signatur
$ curl -sS -o /dev/null -w "%{http_code}" https://app.zentria.tech/api/v1/auth/me
401          ← das Gateway, unverändert erreichbar
```

**Lies zuerst:** `.planning/launch-lagebild-2026-08-12.md` §6 (Etappen) · §8 (Entscheidungen).
Dazu `.planning/preis-und-kostenanalyse-2026-08-13.md` §10 und
`.planning/auslieferungsmodell-2026-08-12.md` §6 — dort stehen die weiter offenen Entscheidungen.

> **Arbeitsweise:** Recherche und Kartierung an Explore-Agenten (max. 3 parallel), das Hauptfenster
> für Entscheidungen und Diffs. Hat wieder gut funktioniert.

---

## BLOCKER: Anmeldung — eingegrenzt, Serverseite ist sauber

**Stand 2026-08-17, gemessen am lebenden System.** Die Ursache liegt **nicht** im Backend und
**nicht** im `tokenStore`-Umbau. Ausgeschlossen durch Messung:

| Prüfung | Ergebnis |
|---|---|
| 6 Login-Versuche 12:57:20–12:58:23 UTC | alle **401**, kein 429, kein 5xx |
| Konto `lukeleonhoppe@gmail.com` | aktiv, kein 2FA, bcrypt `$2a$12$` |
| Reset | `updated_at 12:57:05`, Token `used_at` gesetzt → das UPDATE lief |
| Account-Lockout | existiert im Schema überhaupt nicht |
| Rate-Limit auf Login | IP-basiert, 100 rps **pro Sekunde** → irrelevant |
| E-Mail-Lookup | `WHERE lower(email) = $1` mit `normalizeEmail` davor |
| Reset vs. Login | dieselbe Spalte, gleicher bcryptCost, `newPassword` unverändert gehasht |

Es gibt damit keinen Code-Pfad, auf dem ein korrekt eingegebenes Passwort zu 401 führt. Zwei
Mechanismen bleiben, und der erste war ein echter Mangel: **die Login-Maske hatte keine
`autocomplete`-Attribute**, weshalb Chrome die Felder heuristisch zuordnet und ein zu einem
anderen Konto gespeichertes Passwort einsetzen kann — sichtbar nur als Punkte. Behoben.

**Was jetzt zu tun ist:** einen normalen Login-Versuch machen, dann

```
sudo docker logs docker-auth-1 --since 10m | grep -i "login failed"
```

Der Auth-Service loggte Fehlversuche vorher **gar nicht** — deshalb war die Ursache „völlig offen".
Jetzt nennt er den Grund: `unknown_email`, `password_mismatch`, `user_inactive` oder
`lookup_error`. Verifiziert auf dem deployten Stand. Bei `password_mismatch` trennt ein `fetch`
aus der DevTools-Konsole (umgeht Autofill) einen Tippfehler von einer echten Abweichung.

Danach der Rest von Gate 2, weiter ungeprüft: einloggen → **neu laden** → noch angemeldet (daran
hängt der `tokenStore`-Umbau) → abmelden → neu laden → ausgeloggt → Kontakt anlegen, speichern,
neu laden.

### Am 2026-08-17 dazu gebaut und deployt (`44e1773f`)

- **Login-Fehler sind unterscheidbar.** `stores/auth.ts` beantwortete 401, 403, 409, 429, 5xx und
  einen abbrechenden `fetch` mit **einem** hartkodierten englischen Satz, bei Locale de-DE. Jetzt
  ein Mapping auf `response.status`, fünf neue i18n-Keys ×4, plus `auth.serverUnreachable` für den
  vorher ungefangenen Netzwerkfall.
- **Reset-Mail:** Deutsch, `multipart/alternative` mit vollwertigem Plain-Text-Teil, RFC-2047-Betreff,
  `Date`- und `Message-ID`-Header (fehlten beide — eigenes Spam-Signal), und sie **nennt die
  Kontoadresse**. Genau deren Fehlen machte vier eigene Konten verwechselbar.
- **Reset-Seite:** Farbtoken an `globals.css`, „Zur Anmeldung"-Link im Erfolgs- und im
  Ablauf-Zustand (sie sagte vorher „dieses Fenster kann geschlossen werden" — im Web falsch).
  Härtung unverändert, in Produktion per `curl` gegengeprüft.

> **Gate 2:** Jemand außerhalb des Teams startet das Produkt ohne Lukes Hilfe und ohne Warndialog.
> Die URL genügt, es gibt nichts zu installieren — aber solange die Anmeldung nicht geht, ist das
> Gate offen.

## Erledigt: DNS-Korrektur

`.planning/dns-korrektur-2026-08-17.md` ist **vollständig abgearbeitet**. SPF trägt jetzt Brevo,
die Resend-Reste sind weg, die Brevo-DKIM-Records stehen. Belegt durch die angekommene Reset-Mail.

---

## Was am 2026-08-17 passiert ist

### Etappe 2 — Web-Weg gebaut und live

| Was | Wo |
|---|---|
| Root-Cause: `window.electronAPI` war **non-optional** typisiert | `global.d.ts` — deshalb schwieg tsc zu 8 ungeschützten Auth-Aufrufen |
| Plattform-Abstraktion | `lib/platform.ts` — `isElectron()` + `tokenStore`, typisiert als `ElectronAPI['auth']` |
| Web-Build | `vite.web.config.mts`, `npm run build:web` · Chunks geteilt via `build-chunks.mjs` |
| Server-Adresse zur Laufzeit | `lib/constants.ts` leitet aus `window.location.origin` ab |
| Auslieferung | `desktop/Dockerfile.web` → Service `webapp`, Edge-Caddy-Snippet `cosmi_routes` |
| Web-QA ohne Bridge-Stub | `scripts/qa-web-build.mjs` |

Commits: `72777f7e` (Renderer), `99e8ddca` (Build-Target), `7fcaca27` (Auslieferung, gestaged),
`dcd26083` (Asset-404), `554fc807` (Umschaltung), `f292294a` (deploy.sh).

**Damit ist auch Auslieferungsmodell-Frage 5 beantwortet** („Serveradresse zur Laufzeit"): Im Web
kommt die App vom Server des Kunden und kennt ihn dadurch. Ein Image für alle Kunden.

**Vier Befunde, die in keinem Dokument standen:**

1. **Der Compose-Pop-Out verlor den Entwurf — schon in Electron.** Er reichte den Draft durch einen
   Zustand-Store ohne `persist`; ein zweites `BrowserWindow` ist ein eigener Renderer-Prozess mit
   eigenem Heap und sah ihn nie. Wer mitten im Satz „als Fenster öffnen" klickte, verlor den Text.
   Läuft jetzt über `localStorage` mit TTL.
2. **Der SPF-Fund oben.**
3. **`/assets/*` fiel auf `index.html` zurück** — eine fehlende Bundle-Datei antwortete 200 mit HTML
   statt 404. Ein Tab, der einen Deploy überlebt, hätte HTML als JavaScript geparst. **Genau das hat
   der Staging-Port gefangen, bevor es live ging.**
4. **`BUILDABLE_SERVICES` in `deploy.sh` ist eine explizite Liste** — ein neuer Compose-Service wird
   sonst einmal von Hand gebaut und nie wieder.

### Etappe-0-Nachzügler

- **Website-Deploy ist kein Provisorium mehr.** Runner `zentria-web` läuft als systemd-Dienst, ein
  Push nach `main` deployt automatisch (verifiziert), der Container läuft aus dem Runner-Verzeichnis
  statt aus der tar-Kopie.
- **Staging-Host `neu.zentria.tech`** aus dem Caddyfile raus.
- **NS-Delegation ist durch** — Hetzner-Nameserver aktiv, Cloudflare vollständig raus.

---

## Was als Nächstes dran ist

### 1. Gate 2 abnehmen (wartet nur noch auf einen Login-Versuch)

Siehe den Blocker-Abschnitt oben. Die Diagnose ist gemacht, die Instrumentierung ist live — es
fehlt ein Versuch und ein Blick in den Auth-Log.

### 2. Vercel-Projekt löschen — Vorbedingung geprüft

`zentria.tech` und `www` lösen auf 178.104.38.195 auf, Antwort trägt nur `Server: Caddy`, keinen
Vercel-Header. Löschung im Dashboard, muss ein Mensch machen.

### 3. Etappe 3 (G1, ~10 PT) — vor dem ersten echten Datensatz

`restore.sh` reparieren **und einmal durchführen** · Offsite-Backup verschlüsselt · Consent-Prop
verdrahten · Kontakt-Löschung-UI + vollständige Kaskade · DSAR um Finance/Meetings/Chat ·
`/health` um Postgres · Backup-Alert.

---

## Neue Befunde vom 2026-08-17 (nachmittags)

- **CI war seit dem 11.08. rot und niemand sah es.** `golangci-lint` fand drei QF1008-Verstöße in
  `internal/crm/contact/service.go` (aus `98196b03`, Etappe 1). Weil `ci.yml` auf `backend/**`
  filtert und jeder Commit danach nur `desktop/`, `deploy/` oder `.planning/` berührte, lief der
  Linter wochenlang nicht. Der erste Backend-Commit erbt dann eine rote CI, die nicht von ihm
  stammt. Behoben in `4704f464`.
- **`cd.yml` deployt jetzt auch bei Frontend-Commits** (`workflows: ["CI", "CI Desktop"]`). Das
  allein wäre ein Rückschritt gewesen: Das `if` des Jobs sieht nur den auslösenden Workflow, also
  hätte ein grünes „CI Desktop" heute ein Backend mit roter CI deployt. Der neue erste Step fragt
  die API nach dem Geschwister-Lauf und verweigert bei Fehlschlag; er **wartet**, solange der
  andere noch läuft, sonst gäbe es bei jedem Commit über beide Pfade einen roten CD-Lauf. Gegen
  beide echten Fälle von heute verifiziert. `gh` fehlt auf dem Runner — `curl` und `jq` sind da.
- **Caddy überschreibt `Referrer-Policy`.** Die Reset-Seite setzt `no-referrer`, geliefert wird
  `strict-origin-when-cross-origin`, weil das Caddyfile den Header an drei Stellen per
  `header`-Direktive setzt. Praktisch leakt der Token nicht (cross-origin geht nur der Origin
  raus), aber die Absicht des Codes wird kassiert. Fix wäre `header ?Referrer-Policy …` — Caddys
  „nur setzen, wenn nicht vorhanden". Braucht `compose up -d --force-recreate caddy`, deshalb
  nicht nebenbei gemacht.
- **`HEAD /reset-password` antwortet 405.** Die Route ist nur für GET und POST registriert. Für
  Browser irrelevant, für Linkchecker und Monitoring nicht. Achtung bei der Diagnose: `curl -I`
  zeigt dadurch die globalen statt der gehärteten Header — mit `curl -D -` prüfen.
- **`desktop/src/renderer/src/lib/pricing.ts` ist kein Spiegel, obwohl es das behauptet.** Alte
  Modulpreise, kein Modulstatus, und eine **vierte** Variante der Support-Stufen
  (`standard 0 / priority 99 / enterprise 299` gegen Basis 9 / Professional 10 % / Premium 15 %
  überall sonst). Speist nur Demo- und Insight-Daten, ist also kein Geldrisiko — die Demo zeigt
  aber Preise, die niemand mehr verkauft. Details in `docs/PRICING.md` §13.
- **Der Blog-Artikel bewirbt weiter Module in Vorbereitung.** `wahre-kosten-tool-chaos.md`
  beginnt mit „CRM, Chat, Videokonferenzen, Projektmanagement und Finanzen in einer Plattform" und
  verlinkt am Ende `/funktionen/finanzen` — Finanzen und Meetings sind beide nicht buchbar. Der
  Preis- und Ersparnisabsatz ist korrigiert, dieser Teil ist redaktionelle Arbeit und offen.
- **`.claude/settings.local.json` ist im Website-Repo getrackt** (seit `598aa0c`). Enthält nur
  Bash-Allowlists, keine Geheimnisse, Repo ist privat — gehört aber in `.gitignore`.

## Offene Posten, die nirgends sonst stehen

- **`docs/PRICING.md` und `.knowledge/pricing.md` sind nachgezogen** (13 Module zu 60 €, die elf
  anderen ohne Preis, „50-90 %" und die 16 unbelegten Modul-Prozentzahlen entfernt). **Nicht**
  erledigt und dort jetzt als „nicht zitieren" markiert: die Branchenpakete, der
  Wettbewerbsvergleich und das ORBIT-Beispiel. Jedes der fünf Pakete enthält Module in
  Vorbereitung — das Paket „Produktion" besteht überwiegend daraus. Sie neu zusammenzusetzen ist
  eine Produktentscheidung für Darien, keine Dokumentationspflege, und hängt zusätzlich an den
  offenen Punkten aus §10 der Preisanalyse (79-€-Grundgebühr, Mindestbestellwert, ORBIT).
- **Das Intro blockiert die Anmeldung ~6,6 Sekunden** (`CosmiLaunch` `T_TEXT_END`, einmal pro
  Browser-Session, `prefers-reduced-motion` wird respektiert). Im Desktop einmal pro App-Start, im
  Web beim ersten Aufruf. Kein Blocker, aber ein Erstbesucher sieht 6 Sekunden ein „C". Bewusst
  nicht angefasst — das ist eine Marken-Entscheidung.
- **Web-Bundle: 4,1 MB / 1,0 MB gzip im Entry-Chunk.** Durch die geteilte Chunk-Liste von 4,85 MB
  gesenkt, aber `recharts` und die Radix-Pakete stehen noch nicht in `build-chunks.mjs`.
- **`GO-LIVE.md` im Website-Repo ist komplett überholt** — beschreibt Vercel, Resend und eine
  Supabase-Warteliste, alles entfernt. Offene Checkboxen lesen sich als „noch zu tun".
- **Das Hero-Mockup zeigt gestrichene Module.** `src/components/mockup/mockup-tokens.ts` ist eine
  zweite, unabhängige Modulliste; `index.astro:99` rendert `ProductMockup` ohne `selectedModules`.
- **~1.950 Zeilen toter Studio-Code** (`MockupShell.tsx`, `ModuleViews.tsx`, `ModuleIcons.tsx`).
- **`displayUserName` löst nur Demo-IDs auf** — an ~19 Stellen zeigt die UI rohe UUIDs.
- **`lifecycle_stage` und die `lead_*`-Spalten werden nie gelesen.**
- **Schichttausch-UI** (`SchichtenPage.tsx:1941`) bietet Partner an, die nicht auf der Schicht stehen.
- **118 tsc-Fehler im Desktop** (Vorbestand, Zahl gehalten), und der CI-Schritt dagegen prüft null
  Dateien — `tsconfig.json` hat `files: []`. Echtes Gate ist `npx tsc -p tsconfig.web.json`.
- **19 offene Entscheidungen** aus Dariens zwei Dokumenten, darunter „ein Server pro Kunde".

## Fallstricke

- **Ein Boolean wie `isElectron()` macht kein Type-Narrowing.** Für echte Aufrufe die Bridge lokal
  binden (`const x = window.electronAPI?.ns; if (x) x.call()`), sonst meckert tsc weiter zu Recht.
- **Jedes `qa-*.mjs` stubbt `window.electronAPI`** — genau das verbirgt Web-Fehler. `qa-web-build.mjs`
  tut es bewusst nicht.
- **Playwrights „visible" heißt nicht „bedienbar".** Das Login-Feld hat die ganze Zeit eine Box, die
  Intro-Animation liegt nur darüber. Auf `elementFromPoint` prüfen.
- **Screenshots wirklich ansehen.** Die DOM-Prüfung meldete „Login gerendert", das Bild zeigte den
  Splash.
- **Bind-gemountete Configs werden nie neu geladen — und der Reload lügt.** `caddy reload` quittiert
  mit `"config is unchanged"`. Fix: `compose up -d --force-recreate <service>`. Gegenprobe:
  `docker exec <c> grep <marker> <pfad>`.
- **Erst Staging, dann Produktion.** Beim Caddy-Umbau war das ein interner Port `:8081` mit
  identischem Snippet — kein DNS, kein Zertifikat, und er hat den Asset-404-Bug gefunden.
- **`git pull` auf dem Server braucht den Deploy-Key:**
  `sudo GIT_SSH_COMMAND='ssh -i /home/deploy/.ssh/github_deploy' git pull --ff-only`.
- **Neue Gateway-Route außerhalb `/api/`?** Muss ins `cosmi_routes`-Snippet, sonst verschluckt sie
  die SPA und antwortet mit `index.html` statt 404.
- **`package.json` hat kein `"type": "module"`** — eine `.ts`-Vite-Config lädt als CommonJS und kann
  ESM-only-Plugins nicht importieren. Deshalb `.mts` bzw. `.mjs`.
- **Hetzners Konsole blockt Browser-Automatisierung.** DNS-Zonen muss ein Mensch klicken. Aber
  Hetzners Nameserver lassen sich direkt abfragen: `nslookup <name> hydrogen.ns.hetzner.com`.
- **Statusdokumente hinken dem Code hinterher — miss selbst.**
- **`go build ./...` sprengt den Speicher** (24 Services parallel). Gezielt bauen, `-p 2`.
- **Im KMU-Hub-Repo deployt ein Commit nur, wenn er `backend/**` berührt.**
- **`go test` ohne `DATABASE_URL` ist kein Gate.** Wegwerf-Postgres per Docker kostet zwei Minuten.
- **Kein `reset --hard`, kein Force-Push.** CI-Rot-Recovery ist `git revert <sha>`.
