# Session-Ziel: Etappe 0 abschließen, dann Etappe 2 entscheiden

Stand: **2026-08-16.** Es gibt **kein Launch-Datum** — das Ziel ist **Produkt 1.0.0** nach
Reifegrad-Gates. Nennt dir ein Dokument ein Datum, ist das Dokument veraltet, nicht die Lage.

**Lies zuerst:** `.planning/launch-lagebild-2026-08-12.md` §4 (Befunde G0/G1/G2) · §6 (Etappen mit
Gates) · §8 (Entscheidungen). Dazu Dariens `.planning/preis-und-kostenanalyse-2026-08-13.md` §10
(zwölf offene Fragen) und `.planning/auslieferungsmodell-2026-08-12.md` §6 (sieben weitere).

> **Arbeitsweise für die nächste Session:** mehr mit Subagenten arbeiten. Diese Session lief fast
> vollständig im Hauptkontext und lag am Ende bei über 600k Token. Recherche und Kartierung gehören
> an Explore-Agenten (max. 3 parallel), das Hauptfenster bleibt für Entscheidungen und Diffs.

---

## Was am 2026-08-16 passiert ist

**Etappe 1 im Produkt ist abgeschlossen** (`cb38ea8d`, `f0feea36`, `26e9f800`, `98196b03`,
`4a067f93`, `cf3f18b0`, `951c21af`):

- **G0-6 Kontakte-Felder.** Der Befund war größer als gemeldet: nicht neun Felder, sondern
  **dreizehn** — zu den bekannten kamen `salutation`, `title`, `category`, `status`. Alle sind jetzt
  echte Spalten (Migration **000314**), keine Custom Fields. Gegen ein echtes Postgres 16 verifiziert:
  alle 314 Migrationen laufen, die drei CHECK-Constraints greifen, ein Schreib-Lese-Roundtrip behält
  jedes Feld samt Umlauten, down/up ist reversibel, und die Go-Suiten laufen mit `DATABASE_URL` und
  der Rolle `kmuhub_app` — nicht nur gegen Mocks.
- **Spec-Drift behoben:** `openapi.yaml` nannte `position` „title" und kannte kein `position`. Der
  `as unknown as`-Cast im Desktop-Adapter ist damit weg.
- **G0-7 Helpdesk-Assignee** auf `useEmployees`; der Picker bindet IDs statt Namen.
- **G0-10 Buchungslink** auf `zentria.tech/book/{slug}`, konfigurierbar über
  `RENDERER_VITE_BOOKING_URL`.
- **Neu gefunden und behoben:** Das Infrastruktur-Panel (`modules/admin/InfrastrukturPage.tsx`, 973
  Zeilen) zeigte erfundene Uptime, Backup- und Sicherheitszustände — „Letztes Backup: heute 03:00",
  während die echten Backups zwei Monate tot waren. Aus der Navigation genommen.
- **Dabei ein Root Cause:** `enabled: false` wirkte in der Haupt-Sidebar nicht. Nur drei von vier
  Layout-Varianten prüften das Flag, `ModulesGrid` pflegte eine eigene Liste, und alle 30 Einträge
  standen auf `true`. Der Check sitzt jetzt zentral in `useFilteredNavItems`. **Damit ist die offene
  Frage aus Auslieferungsmodell §7 beantwortet:** Module weglassen ging im Frontend nicht sauber,
  jetzt geht es.
- **CSV-Import:** Der Header-Normalisierer strippte Nicht-ASCII, „Straße" wurde zu `strae` und traf
  kein Mapping — die Straßenspalte jedes deutschen Exports fiel beim Import still unter den Tisch.

**Etappe 0 auf der Website ist bis auf den Vercel-Umzug erledigt** (Repo `zentria-website`,
`cc60c09`, `d2bca8f`, `b60f61f`, `8c32900`):

- `/beta` offline (302 auf `/kontakt`, geparkt als `_beta.astro`). Der CTA auf **vierzehn Seiten**
  versprach „50 % Lifetime-Rabatt für die ersten 100" — ein bindendes Angebot an eine leere Liste.
  Ersetzt durch einen Gesprächs-CTA.
- **Drei Auftragsverarbeiter weniger:** Supabase und Resend fielen mit der Warteliste weg, dazu
  Vercel Analytics und Speed Insights, die auf jeder Seite liefen und **in keiner Zeile der
  Datenschutzerklärung standen**. Plausible bleibt (EU, cookielos, dokumentiert).
- Absolute Claims („keine Drittländer", „kein CLOUD Act", „0 % US-Cloud-Dienste") durch prüfbare
  ersetzt. Fünf tote Komponenten mit veralteten Preisen gelöscht.
- **Modul-Cut umgesetzt:** die elf Module tragen `status: 'planned'`, bleiben sichtbar, sind nicht
  buchbar und zählen in keinem Preis. Modulpreise auf Dariens kalibrierte Fassung — 13 buchbare
  Module, Summe **60 €**.
- Impressum, Datenschutz und AGB nennen keine UG mehr, die es nicht gibt.

---

## Was als Nächstes dran ist

### 1. Zwei Kleinigkeiten aus dieser Session

- **Ladungsfähige Anschrift im Impressum.** `src/pages/impressum.astro` hat `PROVIDER_STREET = ''`
  mit einem TODO. § 5 DDG verlangt Straße und Hausnummer; „55131 Mainz" allein reicht nicht. Die
  falsche Registerangabe ist raus, diese Lücke nicht.
- **Hetzner-Konto ansehen** (Dariens R4): Welches Modell trägt die 31 €, was kostet ein zweiter?
  Die Neupreis-Spanne für gleiche Ausstattung reicht von 16 bis 69 € und entscheidet über ~8 %
  Marge pro Kunde. **R5 ist erledigt** — gemessen 1,53 GiB für alle 36 Container, nicht die
  geschätzten 6–7,4 GB.

### 2. Vercel-Ausstieg (1,5–2 PT) — der letzte Punkt von Gate 0

Nach dem Wegfall der Warteliste ist der Umfang überschaubar: statische Seiten plus vier
Cosmi-Proxy-Routen (`api/cosmi/*`, `api/auth/forgot-password`) plus die SSR-Route `/book/[slug]`.

1. Adapter `@astrojs/vercel` → `@astrojs/node` (`mode: 'standalone'`)
2. Container in `deploy/docker/`; der Server hat 11 GiB frei, ein Astro-Node-Prozess braucht ~150 MB
3. Caddy: Site-Block `zentria.tech` in `/opt/kmuhub/deploy/docker/Caddyfile` — die Datei ist schlank
   (`app.zentria.tech` → gateway, `s3.zentria.tech` → minio), der Block passt daneben. Die fünf
   Security-Header aus `vercel.json` wandern mit
4. **Deploy-Weg — geprüft:** `kmuhub-prod-runner` ist repo-scoped auf `KMU-Hub`, `zentria-website`
   hat 0 Runner, und weil das Konto ein `User` und keine Organisation ist, gibt es keine
   Org-Level-Runner. Empfehlung: zweiten Runner für das Website-Repo auf derselben Maschine
   registrieren — dasselbe systemd-Muster, kein Server-SSH-Key als GitHub-Secret
5. DNS zuletzt: `zentria.tech` → `216.198.79.1` (Vercel), `app.zentria.tech` → `178.104.38.195`.
   A-Record umhängen, Caddy holt das Zertifikat. Kurze Lücke einplanen
6. Vercel-Projekt danach **löschen, nicht pausieren**

> **Gate 0:** `curl -I zentria.tech` zeigt keinen US-Anbieter. Jede Behauptung hat eine Codestelle
> oder ist weg.

### 3. Die neunzehn offenen Entscheidungen

Nichts davon ist entschieden, und mehreres blockiert Folgearbeit:

- **Ein Server pro Kunde oder gemeinsames SaaS?** (Auslieferungsmodell §6 Frage 2) Der
  folgenreichste Punkt. **Die Grundgebühr von 79 € hängt daran** — sie ist bewusst nicht im Code,
  und „Keine Grundgebühr" steht weiter auf der Preisseite, weil es heute stimmt. Sobald entschieden:
  ~2–3 PT (`calculateMonthlyPrice()` in `modules.ts` plus vier Verbraucher).
- **Web oder Desktop?** (Lagebild §8 Entscheidung 2)
- **Reifegrad-Stufe „Vorschau, 50 %"?** (Preisanalyse §10 Frage 6) Das Datenmodell hat dafür schon
  `ModuleStatus` — die dritte Stufe ist ein Einzeiler, wenn ihr sie wollt.

### 4. Etappe 3 (G1, ~10 PT) — vor dem ersten echten Datensatz

`restore.sh` reparieren **und einmal durchführen** · Offsite-Backup verschlüsselt · Consent-Prop
verdrahten · Kontakt-Löschung-UI + vollständige Kaskade · DSAR um Finance/Meetings/Chat ·
`/health` um Postgres · Backup-Alert.

---

## Offene Posten, die nirgends sonst stehen

- **`docs/PRICING.md` ist nicht nachgezogen.** Die Website führt jetzt 13 buchbare Module zu 60 €,
  `PRICING.md` weiter 24 zu 97 € — inklusive „50-90% guenstiger" (`:269`) und 16 Modulzeilen „bis
  X% guenstiger" (`:63-111`), die Darien in §4.4 auf ~27 % korrigiert hat. `.knowledge/pricing.md`
  hängt mit dran. **Das war im Plan und ist nicht mehr geschafft worden.**
- **Das Hero-Mockup zeigt gestrichene Module.** `src/components/mockup/mockup-tokens.ts` ist eine
  zweite, unabhängige Modulliste; `index.astro:99` rendert `ProductMockup` ohne `selectedModules`,
  also erscheinen Meetings, Inventar, Fuhrpark und eine Finanzen-Kachel mit „18 Rechnungen".
  Bewusst zurückgestellt: bei „in Vorbereitung" ist die Diskrepanz klein.
- **~1.950 Zeilen toter Studio-Code** (`MockupShell.tsx`, `ModuleViews.tsx`, `ModuleIcons.tsx`) —
  von keiner Seite importiert, mit ausgearbeiteten Views für die gestrichenen Module.
- **`displayUserName` löst nur Demo-IDs auf.** Im Helpdesk über einen Resolver umgangen, an ~19
  weiteren Stellen (Rapporte, Schichten, Audit-Events) zeigt die UI gegen echtes Backend rohe UUIDs.
- **`lifecycle_stage` und die `lead_*`-Spalten werden nie gelesen** — keine SELECT-Liste im
  Contact-Repository lädt sie. Beim Schreiben gesetzt, beim Lesen verworfen. Nicht geprüft, ob ein
  anderer Pfad sie holt.
- **Kategorielabels passen nicht mehr ganz:** „Finanzen & Einkauf" enthält buchbar nur noch
  Verträge, „Tools & Berichte" nur Wiki. Tragbar, solange die geplanten Module sichtbar bleiben.
- **Reset-Flow nie end-to-end geprüft** — Mail anfordern, Link klicken, Passwort setzen, einloggen.
- **Schichttausch-UI** (`SchichtenPage.tsx:1941`) bietet Partner an, die nicht auf der Schicht stehen.
- **118 tsc-Fehler im Desktop** (Vorbestand), und der CI-Schritt dagegen prüft null Dateien.

## Fallstricke

- **Statusdokumente hinken dem Code hinterher — miss selbst.** Diese Session hat drei Befunde
  korrigiert, die in keinem Dokument standen, und einen, der falsch beschrieben war.
- **Ein zu enger Grep führt in die Irre.** Ich hatte Plausible aus der CSP entfernt, weil ein Grep
  es nicht fand — es war zwei Zeilen tiefer im selben Layout eingebunden. Nach Wortstämmen suchen,
  nicht nach Importmustern.
- **Astro rendert HTML-Kommentare aus.** Eine Begründung, die die alte Falschangabe zitierte, stand
  danach im ausgelieferten Impressum. Solche Notizen gehören ins Frontmatter.
- **`.map(fn)` reicht den Index als zweites Argument durch** — ein neuer optionaler Parameter an
  einer Adapterfunktion wird sonst stillschweigend mit einer Zahl belegt. Der Typecheck hat es
  gefangen.
- **`go build ./...` sprengt den Speicher** (24 Services parallel). Gezielt bauen, `-p 2`.
- **Push nach `main` im Website-Repo = sofortiges Production-Deployment.** Erst `npm run build`.
- **Im KMU-Hub-Repo deployt ein Commit nur, wenn er `backend/**` berührt.**
- **`go test` ohne `DATABASE_URL` ist kein Gate.** Ein Wegwerf-Postgres per Docker kostet zwei
  Minuten: `docker run -d -e POSTGRES_PASSWORD=… pgvector/pgvector:pg16`, dann `migrate up` und die
  Tests mit `kmuhub_app`.
- **Kein `reset --hard`, kein Force-Push.** CI-Rot-Recovery ist `git revert <sha>`.
