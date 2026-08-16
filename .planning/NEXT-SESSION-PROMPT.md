# Session-Ziel: Etappe 2 bauen (Web-Weg)

Stand: **2026-08-16, abends.** Es gibt **kein Launch-Datum** — das Ziel ist **Produkt 1.0.0** nach
Reifegrad-Gates. Nennt dir ein Dokument ein Datum, ist das Dokument veraltet, nicht die Lage.

**Gate 0 ist bestanden.** Zum ersten Mal gemessen statt behauptet:

```
$ curl -I https://zentria.tech
HTTP/1.1 200 OK          ← kein Server-, kein X-Vercel-Id-, kein Via-Header
```

**Lies zuerst:** `.planning/launch-lagebild-2026-08-12.md` §6 (Etappen) · §8 (Entscheidungen).
Dazu `.planning/preis-und-kostenanalyse-2026-08-13.md` §10 und
`.planning/auslieferungsmodell-2026-08-12.md` §6 — dort stehen die weiter offenen Entscheidungen.

> **Arbeitsweise:** Recherche und Kartierung an Explore-Agenten (max. 3 parallel), das Hauptfenster
> für Entscheidungen und Diffs. Hat heute gut funktioniert: drei Agenten haben Website, Deploy-Infra
> und Desktop-Code parallel vermessen, während im Hauptkontext gearbeitet wurde.

---

## Was am 2026-08-16 passiert ist

### Etappe 0 abgeschlossen — die Website läuft auf eurem Server

`zentria.tech` und `www.zentria.tech` zeigen auf `178.104.38.195`, Caddy terminiert TLS, ein
Astro-Node-Container liefert aus. Belegt: 24 Sitemap-Seiten plus Impressum/Datenschutz/AGB alle 200,
`/ki` und `/beta` weiter 302, die SSR-Route und alle vier Cosmi-Proxy-Routen erreichen das Backend,
`www` leitet mit 301 auf den Apex. `app.zentria.tech` und `s3.zentria.tech` blieben unberührt.

Commits: `898820c` (Node-Adapter, Dockerfile, Compose, Deploy-Workflow), `d7580c8` (DSE auf Hetzner),
`fd041c48` (Caddy-Blöcke).

**Drei Befunde, die in keinem Dokument standen:**

1. **Die Website hatte keinen Rückkanal.** `zentria.tech` besaß **keinen MX-Record** — die sieben
   Strato-Postfächer waren bezahlt und unerreichbar, `kontakt@` hatte 11 KB Altbestand. Vermutlich
   beim Wechsel zu Cloudflare verloren gegangen. Gesetzt sind jetzt MX (`smtpin.rzone.de`), SPF
   (`v=spf1 redirect=_spf.strato.com`) und zwei DKIM-CNAMEs; ohne SPF/DKIM hätte `_dmarc p=reject`
   jede ausgehende Mail abgelehnt. **Mit echter Testmail verifiziert.** `hallo@`, `jobs@`, `presse@`
   sind als Aliasse angelegt — das Kontaktformular schickt an `hallo@`.
2. **Cloudflare war der letzte US-Anbieter im Pfad.** Betrieb die Nameserver und stand in keiner
   Zeile der Datenschutzerklärung. Zone ist bei Hetzner angelegt und Record für Record gegen
   Cloudflare verifiziert.
3. **Die Produktionsplatte war zu 85 % voll** — 206 GB toter BuildKit-Cache, keiner davon in
   Benutzung. Bereinigt (243 GB → 51 GB), plus wöchentlicher Cron sonntags 4 Uhr, der Einträge
   älter als 7 Tage entfernt. Mit Kundendaten hatte das nichts zu tun: Docker-Volumes 2,2 GB.

Dazu die Rechtstexte (`9a31e34`): ladungsfähige Anschrift **Mainzer Straße 47, 55124 Mainz** — die
bisherige PLZ 55131 war falsch. Adresse liegt jetzt in `src/config.ts`, Impressum und Datenschutz
ziehen sie von dort. Die DSE beschrieb ein Kontaktformular, das Daten speichert; tatsächlich baut
`kontakt.astro` einen `mailto:`-Link, nichts erreicht je einen Server. Und in `BaseLayout.astro`
stand die interne Begründung zu den entfernten US-Diensten in einem **HTML-Kommentar** — auf allen
26 Seiten im ausgelieferten Quelltext.

### Etappe 2 ist entschieden: Web-Weg

Zwei Dokumente widersprachen sich („≈ eine Datei" vs. „4 PT"). Gemessen liegt die Wahrheit
dazwischen, näher an 4 PT, mit Abzügen — **realistisch 2–3 PT**:

| Punkt | Stand |
|---|---|
| Screen-Share | **schon erledigt**, Web-Fallback in `ScreenSourcePicker.tsx:49-54` |
| IPC-Fläche | klein: 15 Kanäle, 28 Aufrufstellen in 12 Dateien, alles über `window.electronAPI` |
| Router | `createHashRouter` — browserfähig ohne Änderung |
| Native Module | keine; kein Auto-Updater, kein Deep-Link-Handler (totes Gerüst) |
| Auth-Persistenz | `main/ipc/auth.ts` (safeStorage+fs) → `stores/auth.ts` **8 Aufrufstellen ungeschützt**, im Browser ein Crash |
| Popup-Fenster | 3 Stück (`compose`, `employee-wizard`, `editor-window`) → Modals |
| Tray/Menü | entfallen ersatzlos, müssen abgeschottet werden |
| Build | kein Web-Target, `electron.vite.config.ts` kennt nur main/preload/renderer |
| Backend-URL | Build-Zeit-fix (`RENDERER_VITE_API_URL`), nicht laufzeitkonfigurierbar |

Damit ist **Lagebild-Entscheidung 2 beantwortet** und ebenso Auslieferungsmodell §6 Frage 3. Der
Ausschlag: Das Gate von Etappe 2 verlangt „ohne Warndialog", und das heißt beim Desktop-Weg
Code-Signatur — Zertifikat auf Hardware, geprüfte Identität, 200–500 €/Jahr, auf die UG erst nach
Etappe 4. Der Web-Weg braucht das nicht.

---

## Was als Nächstes dran ist

### 1. Drei Nachzügler aus dieser Session (klein, aber nicht vergessen)

- **Vercel-Projekt löschen, nicht pausieren.** Erst wenn der Apex ein bis zwei Tage sauber läuft —
  bis dahin ist der alte A-Record `216.198.79.1` die Rückfalloption. Die Git-Integration ist bereits
  pausiert.
- **Staging-Host abräumen:** Caddy-Block `neu.zentria.tech` in `deploy/docker/Caddyfile` und der
  gleichnamige A-Record in beiden Zonen.
- **Delegation prüfen.** Sie hing am Abend seit zweieinhalb Stunden auf Cloudflare:
  `nslookup -type=NS zentria.tech ns10.trs-dns.org`. Folgenlos, weil beide Zonen identisch sind —
  aber wenn sie morgen noch steht, in der Strato-Domainverwaltung nach einem Transfer-Lock sehen.

### 2. Das Deployment der Website ist noch ein Provisorium

Die Quellen liegen als **Kopie** unter `/opt/zentria-website` (per tar hochgeladen), nicht als
Git-Checkout: Der Deploy-Key des Servers ist repo-scoped auf KMU-Hub und kommt nicht ans
Website-Repo. Ein Textfix auf der Website müsste heute von Hand hoch.

`.github/workflows/deploy.yml` liegt fertig im Website-Repo und wartet auf einen Runner mit den
Labels `[self-hosted, zentria-web]`. Repo-scoped, weil das Konto ein `User` und keine Organisation
ist. `actions/checkout` klont mit dem Workflow-Token — ein Deploy-Key wird dann nicht gebraucht.

### 3. Etappe 2 bauen (2–3 PT)

Reihenfolge nach Risiko: Auth-Persistenz zuerst (der einzige Punkt, der im Browser hart crasht),
dann die drei Popups zu Modals, dann Tray/Menü/Deep-Link-Gerüst abschotten, zuletzt das Web-Build-
Target. Gate: Jemand außerhalb des Teams startet das Produkt ohne Lukes Hilfe und ohne Warndialog.

### 4. Etappe 3 (G1, ~10 PT) — vor dem ersten echten Datensatz

`restore.sh` reparieren **und einmal durchführen** · Offsite-Backup verschlüsselt · Consent-Prop
verdrahten · Kontakt-Löschung-UI + vollständige Kaskade · DSAR um Finance/Meetings/Chat ·
`/health` um Postgres · Backup-Alert.

---

## Offene Posten, die nirgends sonst stehen

- **`docs/PRICING.md` ist weiter nicht nachgezogen.** Website: 13 buchbare Module, 60 €.
  `PRICING.md`: 24 zu 97 €, inklusive „50-90% guenstiger" (`:269`) und 16 Modulzeilen „bis X%
  guenstiger" (`:63-111`), die Darien in §4.4 auf ~27 % korrigiert hat. `.knowledge/pricing.md`
  hängt mit dran. **Zweite Session in Folge nicht geschafft.**
- **Tote DNS-Records** in beiden Zonen: `send.zentria.tech` MX+TXT und `resend._domainkey` (Resend
  ist raus), zwei Brevo-DKIM-CNAMEs (ob Brevo genutzt wird, ist ungeklärt — das Backend nutzt
  generisches SMTP über `SYSTEM_SMTP_HOST` aus `.env.production`). Bewusst beim Umzug **nicht**
  angefasst: Ein Umzug ist der falsche Moment zum Aufräumen. Vorlage mit Markierungen liegt im
  Scratchpad als `zentria.tech.zone`.
- **Das Hero-Mockup zeigt gestrichene Module.** `src/components/mockup/mockup-tokens.ts` ist eine
  zweite, unabhängige Modulliste; `index.astro:99` rendert `ProductMockup` ohne `selectedModules`.
- **~1.950 Zeilen toter Studio-Code** (`MockupShell.tsx`, `ModuleViews.tsx`, `ModuleIcons.tsx`) —
  von keiner Seite importiert.
- **`displayUserName` löst nur Demo-IDs auf** — an ~19 Stellen zeigt die UI gegen echtes Backend
  rohe UUIDs.
- **`lifecycle_stage` und die `lead_*`-Spalten werden nie gelesen** — beim Schreiben gesetzt, beim
  Lesen verworfen.
- **Reset-Flow nie end-to-end geprüft.** Jetzt möglich: Der Mailempfang funktioniert nachweislich.
- **Schichttausch-UI** (`SchichtenPage.tsx:1941`) bietet Partner an, die nicht auf der Schicht stehen.
- **118 tsc-Fehler im Desktop** (Vorbestand), und der CI-Schritt dagegen prüft null Dateien.
- **19 offene Entscheidungen** aus Dariens zwei Dokumenten, darunter „ein Server pro Kunde" — daran
  hängt die 79-€-Grundgebühr, die deshalb bewusst nicht im Code ist.

## Fallstricke

- **Bind-gemountete Configs werden nie neu geladen — und der Reload lügt.** `git pull` ersetzt die
  Datei, der Mount hängt an der alten Inode. `caddy reload` quittierte mit `"config is unchanged"`,
  während der Host längst den neuen Block hatte. Gegenprobe: `docker exec <c> grep <marker> <pfad>`
  gegen den Host. Fix: `compose up -d --force-recreate <service>`.
- **Erst Staging-Host, dann Apex.** Genau dieser Test hat den Mount-Fehler gefangen. Ohne ihn wäre
  es ein TLS-Ausfall mit zweijährigem HSTS geworden.
- **Let's Encrypt validiert gegen das, was gerade im DNS steht.** Der `www`-Erstversuch schlug fehl,
  weil die Challenge noch an Vercel ging. Caddy wiederholt alle 2 min von selbst — nicht eingreifen.
- **Hetzners Konsole blockt Browser-Automatisierung** (Bot-Schutz bleibt bei „Verifying" hängen).
  DNS-Zonen dort muss ein Mensch klicken. Cloudflare und Strato gehen. **Aber:** Hetzners
  Nameserver lassen sich direkt abfragen, auch ohne Delegation — `nslookup <name>
  hydrogen.ns.hetzner.com`. So lässt sich eine Zone vollständig prüfen, bevor sie scharf wird.
- **Strato bindet die Sitzung an eine `sessionID` in der URL**, nicht nur an ein Cookie. Ein zweiter
  Tab ist nicht mitangemeldet.
- **Statusdokumente hinken dem Code hinterher — miss selbst.**
- **Ein zu enger Grep führt in die Irre.** Nach Wortstämmen suchen, nicht nach Importmustern.
- **Astro rendert HTML-Kommentare aus.** Im Layout heißt das: auf jeder Seite. Notizen ins Frontmatter.
- **`go build ./...` sprengt den Speicher** (24 Services parallel). Gezielt bauen, `-p 2`.
- **Push nach `main` im Website-Repo deployt** — solange die Vercel-Integration wieder aktiv ist.
  Erst `npm run build`.
- **Im KMU-Hub-Repo deployt ein Commit nur, wenn er `backend/**` berührt.** Eine reine
  Caddyfile-Änderung löst kein CD aus, der Reload muss von Hand kommen.
- **`go test` ohne `DATABASE_URL` ist kein Gate.** Wegwerf-Postgres per Docker kostet zwei Minuten.
- **Kein `reset --hard`, kein Force-Push.** CI-Rot-Recovery ist `git revert <sha>`.
