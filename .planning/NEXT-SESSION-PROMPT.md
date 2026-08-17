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

## BLOCKER: Anmeldung schlägt fehl — Gate 2 ist NICHT bestanden

**Stand 2026-08-17 abends, von Luke am lebenden System geprüft.** Die Web-App lädt, aber:

- **Anmelden mit den eigenen Zugangsdaten funktioniert nicht.** Ursache unbekannt — nicht
  eingegrenzt, ob es an Zugangsdaten, am Backend, am Rate-Limit oder am `tokenStore`-Umbau liegt.
  Das ist der erste Punkt, an dem gearbeitet werden muss; alles andere in Etappe 2 hängt daran.
- **Der Passwort-Reset-Versand funktioniert** — die Mail kam an. Damit ist zugleich belegt, dass
  der SPF-Fix greift: vor der DNS-Korrektur wäre sie an `p=reject` gescheitert.

Der ganze Rest von Gate 2 steht dahinter und ist ungeprüft: einloggen → **neu laden** → noch
angemeldet (daran hängt der `tokenStore`-Umbau) → abmelden → neu laden → ausgeloggt → Kontakt
anlegen, speichern, neu laden.

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

### 1. Automatisches CD für Frontend-Commits (klein, aber offen)

`cd.yml` triggert auf `workflow_run` von **„CI"**, und `ci.yml` filtert auf `backend/**`. Ein reiner
`desktop/`-Commit deployt deshalb **nicht** — `deploy.sh` baut `webapp` inzwischen zwar mit, aber
nur wenn er läuft.

Vorschlag: `cd.yml` auf `workflows: ["CI", "CI Desktop"]` erweitern. Bewusst nicht eigenmächtig
gemacht — ein CD-Trigger mehr kann Doppel-Deploys auslösen, das gehört entschieden, nicht gebaut.

### 1b. Reset-Flow: Mail und Seite sind unfertig (Lukes Eindruck, 17.08.)

Beides funktioniert technisch, wirkt aber nicht wie ein Produkt, das Vertrauen verkauft:

- **Die Reset-Mail ist auf Englisch und reiner Text** — `backend/cmd/auth/mailer.go:41-49`:
  Betreff „Reset your Cosmi password", Signatur „— The Cosmi Team", `Content-Type: text/plain`,
  kein Branding. Bei Locale de-DE und einem Produkt für DACH-KMUs.
- **Die Reset-Seite sieht „vibe coded" aus** — `backend/internal/gateway/reset_password_page.html`,
  176 Zeilen handgeschriebenes HTML, das vom Design-System der App nichts weiß.

Der Fairness halber: Die Seite ist bewusst gehärtet (strikte CSP, `no-store`, `X-Robots-Tag`) und
serverseitig gerendert, damit sie ohne die SPA funktioniert. Beim Überarbeiten darf das nicht
verloren gehen — sie ist der einzige Weg zurück für einen ausgesperrten Nutzer.

### 2. Vercel-Projekt löschen

Der Apex läuft seit dem 16.08. sauber auf eigenem Server. Rückfalloption war der A-Record
`216.198.79.1`; die Git-Integration ist längst pausiert. Kann jetzt weg.

### 3. Etappe 3 (G1, ~10 PT) — vor dem ersten echten Datensatz

`restore.sh` reparieren **und einmal durchführen** · Offsite-Backup verschlüsselt · Consent-Prop
verdrahten · Kontakt-Löschung-UI + vollständige Kaskade · DSAR um Finance/Meetings/Chat ·
`/health` um Postgres · Backup-Alert.

---

## Offene Posten, die nirgends sonst stehen

- **`docs/PRICING.md` ist weiter nicht nachgezogen.** Website: 13 buchbare Module, 60 €.
  `PRICING.md`: 24 zu 97 €, inklusive „50-90% guenstiger" (`:269`) und 16 Modulzeilen „bis X%
  guenstiger" (`:63-111`), die Darien in §4.4 auf ~27 % korrigiert hat. `.knowledge/pricing.md`
  hängt mit dran. **Dritte Session in Folge nicht geschafft.**
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
