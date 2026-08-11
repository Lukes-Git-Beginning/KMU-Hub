# Launch-Lagebild — Weg zu Cosmi 1.0.0

> Erstellt 2026-08-12. Grundlage: 35 Agenten in vier Phasen (10 Messungen · 20 adversarische
> Gegenprüfungen · 4 Pre-Mortem-Linsen · 1 Completeness-Critic), 91 Befunde, jeder mit Pfad:Zeile
> oder Kommando-Output belegt. Kein Produktionscode angefasst.
>
> **Prämisse dieses Laufs** (aus drei Frage-Runden, nicht aus dem Repo): Das Datum 2026-09-01 ist
> entwertet, die Vertriebs-Pipeline ist leer, ZFA ist kalt und nicht mehr Fokus, Legal steht bei
> null, Budget 2.000–5.000 € unter Überlebensvorbehalt. Ziel ist ein **Produkt 1.0.0** — zeigbar,
> nutzbar, ohne Fremdscham — unter dem Versprechen „Datenschutz und Daten in Deutschland".

---

## 1 · Lagebild in fünf Sätzen

1. Das Produkt ist deutlich besser als sein Ruf im eigenen Repo — RLS lückenlos über 336 Tabellen,
   186/186 Permission-Seeds vollständig, Finance-Backend echt gebaut, ein Onboarding-Wizard, der
   funktioniert, und ein Konfigurations-Editor, der ein echtes Alleinstellungsmerkmal ist.
2. Kaputt ist nicht die Architektur, sondern die **Außendarstellung**: zentria.tech verspricht
   „Keine US-Cloud, Daten verlassen niemals die EU" und läuft dabei selbst auf Vercel (USA),
   nennt in der eigenen Datenschutzerklärung Vercel und Resend als US-Auftragsverarbeiter, bewirbt
   ein KI-Produkt für 149–899 €/Monat, das zu ~95 % nicht existiert, und führt im Impressum eine
   Handelsregister-Eintragung, die es nie gab.
3. Alle vier Pre-Mortem-Linsen — CTO, CFO, CEO, Datenschutz — haben **unabhängig voneinander
   dieselbe Todesursache** genannt: nicht ein Bug, sondern der erste Interessent, der `curl -I
   zentria.tech` tippt und die eigene Werbeaussage widerlegt sieht.
4. Auf dem Weg zum zeigbaren Produkt stehen dazu vier billige, aber peinliche Brüche im Kernpfad:
   das Kontaktformular verwirft neun Felder beim Speichern, die Helpdesk-Zuweisung scheitert am
   echten Backend, die E-Rechnungs-Schaltfläche meldet Erfolg ohne irgendetwas zu tun, und ein
   Modul zeigt bei jedem Laden rohe Übersetzungsschlüssel.
5. Der 01.09. ist als Datum ohnehin tot — aber selbst das kleinste sinnvolle Ziel („wir können
   jemandem das Produkt zeigen, ohne dass es uns widerlegt") ist in **etwa 8 Personentagen**
   erreichbar, und das ist die eigentliche Nachricht dieses Berichts.

---

## 2 · Was wirklich trägt

Darauf soll aufgebaut werden — nicht noch einmal angefasst:

- **Die Datenschutz-Substanz im Backend ist echt.** RLS ist über alle 336 Tabellen lückenlos (die
  7 Ausnahmen decken sich exakt mit `docs/ARCHITECTURE.md:228-240`, keine hat eine `tenant_id`),
  186 von 186 verwendeten Permissions sind geseedet, 0 verwaiste. Gemessen gegen eine frisch
  migrierte DB unter der Rolle `kmuhub_app` — also ohne `BYPASSRLS`-Schummelei.
- **Kein einziges Telemetrie-Leck im Produkt.** Kein Sentry, kein PostHog, kein Analytics-SDK,
  keine Google Fonts, keine CDNs; strikte CSP in `desktop/src/renderer/index.html:15`. `openai`
  liegt ausschließlich in `devDependencies` und wird nie mitausgeliefert. MinIO, OnlyOffice,
  LiveKit, coturn, Redis: alle self-hosted. **Das Produkt hält das DE-Versprechen bereits — nur
  die Website nicht.**
- **Der Finance-Block ist kein Gerüst.** XRechnung-UBL *und* ZUGFeRD-CII teilen sich denselben
  Betrags-Builder (können also nie abweichende Zahlen ausgeben), abgesichert durch echte
  Roundtrip-Tests. `go test` läuft grün in `einvoice`, `datev`, `gobdarchive`, `bexio`. Der
  DATEV-EXTF-Export ist Ende-zu-Ende verdrahtet und ohne DATEV-Partnerschaft nutzbar.
- **Der Konfigurations-Editor ist das stärkste Verkaufsargument, das ihr habt.** Sieben Dimensionen
  (Begriffe, Wertelisten, Felder, Bereiche, Statistik, Spalten, Kanäle), Cross-Window-State-Sync,
  ein Whitelist-Test gegen stilles Verschwinden von Umbenennungen, 36 dedizierte QA-Skripte — jedes
  an einen dokumentierten realen Fehlschlag gebunden. Und er skaliert *anders* als der beworbene
  „1 Woche Onsite"-USP, der bei einem Entwickler nie skaliert hätte.
- **Der Nachtlauf-Mechanismus liefert nachweisbar in Produktion.** Der `display_name`-Bug
  (`31e22ac4`) und beide Audit-Kette-Bugs (`12be7c3f`, `ef032f24`) sind per
  `git merge-base --is-ancestor` als Vorfahren des deployten Commits `60dcdae1` bestätigt. Was im
  Journal steht, kommt an.

---

## 3 · Was „1.0.0" heißt — prüfbare Definition

Der Begriff war bisher ein Gefühl. Hier ist er als Gate, das man bestehen oder nicht bestehen kann:

| # | Kriterium | Wie geprüft |
|---|---|---|
| 1 | **Nichts, was wir öffentlich sagen, ist durch einen 30-Sekunden-Check widerlegbar.** | `curl -I zentria.tech`; Impressum gegen Rechtsträger-Ist; jede Feature-Behauptung hat eine Codestelle |
| 2 | **Ein Fremder kann das Produkt selbst starten**, ohne dass Luke danebensitzt. | Installer (oder URL) an eine saubere Windows-VM geben, jemand ohne Vorwissen startet ihn |
| 3 | **Die drei Kernflüsse halten gegen das echte Backend**, nicht gegen Mocks. | Kontakt anlegen → speichern → **neu laden** → alle Felder da. Ticket anlegen → zuweisen → schließen. Dokument hochladen → finden → herunterladen |
| 4 | **Kein Bildschirm zeigt Rohschlüssel, kaputte Leerzustände oder Erfolg ohne Wirkung.** | Durchklicken mit **leerem** Tenant, Screenshots ansehen |
| 5 | **Ein Ausfall ist überlebbar.** | Restore einmal wirklich durchgeführt, mit Zeitmessung |

Kriterium 1 und 3 sind heute **nicht** erfüllt, 2 ebenfalls nicht, 4 fast, 5 nicht.

---

## 4 · Priorisierte Befunde

### G0 — steht zwischen euch und einem zeigbaren Produkt

Sortiert nach Schadenshöhe. Alle adversarisch gegengeprüft, keiner widerlegt.

| # | Befund | Beleg | Konsequenz beim Lassen | PT | Owner |
|---|---|---|---|---|---|
| G0-1 | **zentria.tech verspricht „Keine US-Cloud, keine Drittländer" und läuft auf Vercel (USA).** Die eigene Datenschutzerklärung nennt Vercel Inc. und Resend Inc. als US-Auftragsverarbeiter mit Standardvertragsklauseln — also genau die Drittlandsübermittlung, die die Startseite ausschließt. | `curl -sI https://zentria.tech` → `Server: Vercel`, `X-Vercel-Id: fra1::…`; Startseite wörtlich „Daten verlassen niemals die EU"; `/datenschutz` nennt Vercel/Resend USA | Der erste Interessent mit IT-Berater bricht ab. Nicht verhandelbar, sondern Abbruchgrund. § 5 UWG. | 1 | Luke |
| G0-2 | **Das Impressum behauptet „Amtsgericht Mainz, Eintragung beantragt"** für eine UG, die es nicht gibt — ohne Notartermin kann nichts beantragt sein. Genannt werden drei Geschäftsführer. (Der dritte Name ist laut Gegenprüfung vermutlich Nicos bürgerlicher Name — kein Fantasiename, aber die Registerangabe bleibt falsch.) | WebFetch `/impressum`; Team-Stand: kein Notartermin | Falsche Pflichtangaben nach DDG sind abmahnfähig; die Seite verarbeitet bereits Kontaktdaten unter einer Rechtsträger-Fiktion → persönliche Haftung | 0,5 | Luke |
| G0-3 | **Die /ki-Seite verkauft für 149–899 €/Monat ein Produkt, das zu ~95 % nicht existiert:** dedizierte GPUs pro Kunde, Qwen3-14B, Whisper, Qdrant, Docling, Haystack-Chatbot, Jitsi. Im Code: `backend/internal/llm/client.go`, 184 Zeilen, ein OpenAI-kompatibler Client für Meeting-Zusammenfassungen. | WebFetch `/ki`; `grep -riE "Qwen|Whisper|Qdrant|Docling|Jitsi"` → 0 Treffer in `backend/`; `client.go:1-6` „intentionally tiny" | Wer die 149–899 €-Stufe bucht, bekommt kein Produkt. Vom Reputationsschaden bis zum Betrugsvorwurf. | 0,5 | Luke |
| G0-4 | **„Beta startet im Juni 2026"** steht seit über zwei Monaten unkorrigiert. Schwerwiegender: **die Anmeldezahlen sind hartkodierte Design-Platzhalter, die echte Warteliste ist leer.** „37 Pioneer", „13 Anmeldungen diese Woche", „13 Plätze in 7 Tagen vergeben" und drei Avatar-Initialen (MS, JK, AF) stehen als Literale im Quelltext. Live abgefragt: `{"count":0}` bei HTTP 200 — echte Null, kein Fehlerfall. Eingeführt durch Commit `a0a3420` („massive redesign pass"), nie durch echte Daten ersetzt. Ironie: eine korrekt gebaute Zähler-Komponente existiert und wird auf genau dieser Seite nicht verwendet. | `zentria-website/src/pages/beta.astro:34,112-120`; `curl https://zentria.tech/api/waitlist-count` → `{"count":0}`, HTTP 200; `src/components/WaitlistCounter.astro` (ungenutzt auf `/beta`) | Überfälliges Datum ist sichtbare Fremdscham. Die fingierte Verknappung ist gravierender: unwahre Angaben zu begrenzter Verfügbarkeit sind im Anhang zu § 3 Abs. 3 UWG als **per se unlautere** Praktik gelistet — ohne Interessenabwägung | 0,5 | Luke |
| G0-5 | **„AVV inkludiert"** wird beworben — es existiert kein AVV. | WebFetch Startseite; `.planning/legal/` enthält nur einen Gesellschaftervertrags-Entwurf | Erste Kundenanfrage „schickt den AVV" platzt sofort | 0,5 | Luke |
| G0-6 | **Kontakte verwirft beim Speichern neun sichtbare Formularfelder** (Mobil, Abteilung, Straße, PLZ, Ort, Land, Website, LinkedIn, Xing) — ohne Fehlermeldung. Der Code sagt es selbst: „werden serverseitig noch nicht persistiert". | `desktop/…/kontakte/adapters.ts:130-149`; `ContactFormDialog.tsx:76-116`; `openapi.yaml:34216-34266` | Stiller Datenverlust im CRM-Kernmodul — und ausgerechnet im ersten Editor-Pilotmodul | 2,5 | Luke |
| G0-7 | **Helpdesk kann gegen das echte Backend weder ein Ticket anlegen noch zuweisen.** Der Assignee-Picker bietet zwei hartcodierte Demo-Namen; das Backend validiert `assignee_id` als UUID → 400. | `HelpdeskPage.tsx:90,291,459-465,1360-1367`; `route_helpdesk.go:155,189` | Zweites Editor-Pilotmodul bricht im ersten Schritt des Hauptflusses. Fix billig: Team-Hook existiert bereits | 0,5 | Luke |
| G0-8 | **Die E-Rechnungs-Oberfläche ist eine Attrappe mit falschem Erfolgs-Toast.** Vier hartcodierte Rechnungsnummern; „XML herunterladen" ruft `toast.success()` ohne jeden Backend-Call — obwohl der Endpoint real existiert. | `EInvoiceIndicator.tsx:87-121, 369-375`; Route real: `route_biz.go:120` | Klick meldet Erfolg, es passiert nichts. Genau der Fremdscham-Moment, den 1.0.0 ausschließen soll | 1 | Luke |
| G0-9 | **Der Fuhrpark zeigt bei jedem Tabwechsel rohe Übersetzungsschlüssel** — sechs `t()`-Aufrufe ohne `defaultValue` auf Keys, die in `de.json` nicht existieren; `fallbackLng` ist ebenfalls `de`, es gibt also keinen Auffang. | `FuhrparkPage.tsx:2056,2059,2160,2257,2376,2491`; Scan gegen 12.072 Keys | Wörtlich „fuhrpark.loading.vehicles" auf dem Bildschirm | 0,25 | Luke |
| G0-10 | ~~Die Kalender-Terminbuchung hat keine öffentliche Webseite.~~ **KORRIGIERT am 2026-08-12 durch den Website-Audit:** Die Buchungsseite **existiert und funktioniert** — im Website-Repo unter `src/pages/book/[slug].astro`, live als `zentria.tech/book/{slug}`. Der echte Fehler ist enger, aber real: Das Produkt erzeugt den Link auf `booking.zentria.tech`, und **diese Domain existiert nicht** (`curl` → Verbindungsfehler 000, während `zentria.tech/book/…` mit 404 antwortet, die Route also lebt). Jeder Tenant, der den Link kopiert, verschickt einen toten Link. | `BookingPagePreview.tsx:103`; `curl https://booking.zentria.tech/x` → 000; `zentria-website/src/pages/book/[slug].astro` | Kopierte Buchungslinks laufen ins Leere. Lehre: Der Reader suchte nur im Produkt-Repo — **genau die Lücke, die diesen Fehlbefund erzeugt hat** | 0,15 | Luke |
| G0-11 | **`electron-builder --win` ist nie gelaufen.** Kein Release-Workflow unter den 7 Workflow-Dateien, kein Downloadpunkt im Repo. | `package.json:9` vs. `ci-desktop.yml:82`; `ls .github/workflows/` | Ihr könnt heute niemandem — auch keinem internen Tester — ein lauffähiges Produkt aushändigen | 0,5 | Luke |
| G0-12 | **Keine Code-Signatur konfiguriert** → jede Installation zeigt „unbekannter Herausgeber". | `desktop/electron-builder.yml:16-19`; kein `CSC_LINK` in CI | Windows-Warnung beim allerersten Kontakt — bei einem Produkt, das mit Vertrauen wirbt | s. Entscheidung 2 | Luke |
| G0-13 | **`docs/pilot0-onboarding/` beschreibt einen Piloten, den es nicht gibt:** „Kunde: ZFA", „Pilot-Dauer: Juli–Oktober 2026" — ein Zeitraum, der bereits läuft. Status: „Skelett". | `docs/pilot0-onboarding/README.md:5,24,29` | Onboarding-Material für einen Kunden, der nicht kommt. Wer das liest, plant falsch | 0,25 | Luke |

**G0 gesamt: ca. 10 Personentage.** Davon entfallen **3,25 PT auf die Website** — und die tragen den größten Teil des Risikos.

### G1 — vor dem ersten echten personenbezogenen Datensatz

| Befund | Beleg | PT |
|---|---|---|
| **Der Passwort-Reset-Fix vom 11.08. greift nicht — am Server verifiziert.** `.env.production` trägt `PASSWORD_RESET_BASE_URL=https://zentria.tech/reset-password`, also zeigen alle Reset-Mails auf die Astro-Seite statt auf die eigens gebaute, gehärtete Gateway-Seite (`10a1a26e`). Die läuft einwandfrei — und wird von keiner Mail angesteuert. Folge: Der Reset-Token läuft durch eine Vercel-Function in **iad1/Virginia**, auf einer Seite mit `Cache-Control: public` statt `no-store` und ohne `X-Robots-Tag`. Die ungenutzte Seite hat beides plus strikte CSP. **Root Cause ist nicht die Env-Var:** `docker-compose.yml:123` setzt den Default auf `zentria.tech` und überschreibt damit den korrekten Go-Default aus `config.go:204`; `PRODUCTION_TEMPLATE:49` trägt denselben Wert seit `0f49fd7f` (16.06.). Wer nur die Env ändert, holt sich den Fehler bei der nächsten Neuinstallation zurück. Nur `cmd/auth` liest den Wert (`main.go:69`) — Neustart genügt nicht, der Container muss neu erstellt werden (`up -d auth`). | Live-Header-Vergleich beider URLs; `docker-compose.yml:123`; `config.go:204`; `PRODUCTION_TEMPLATE:49` | 0,25 |
| **Der Consent-Check beim E-Mail-Versand ist toter Code.** Er greift nur bei gesetzter `contact_id` — kein einziger Frontend-Pfad setzt sie. `ComposeInline` kann es, aber der einzige Aufrufer übergibt die Prop nie; `ComposeModal` kann es auch und wird nirgends importiert; der CRM-Button ist ein `mailto:`-Link. **Das ist der konkreteste Fall eures Verkaufsversprechens — und er ist wirkungslos.** | `email/send/service.go:160-164`; `MailsPage.tsx:833-839`; `ContactDetailPage.tsx:242` | 1,5 |
| **Kontakt-Löschung (Art. 17) hat keinen einzigen UI-Einstieg.** Der Endpoint existiert, keine Komponente ruft ihn. Das einzige Lösch-UI arbeitet auf Mitarbeitern, nicht auf Kunden. | `route_crm_ext.go:374`; Grep in `desktop/src` → nur i18n + generierter Typ | 1,5 |
| **Und selbst dann löscht `AnonymizeContact` nur 2 von ≥6 Tabellen** — Anrufnotizen mit Klarnamen überleben eine „erfolgreiche" Löschung. | `crm/consent/postgres_repository.go:117-149` | 2 |
| **Art.-15-Auskunft deckt 3 von ≥6 Datentöpfen ab** — Rechnungen, Meetings, Chats fehlen. Der Code sagt es selbst: „deliberate subset". | `security/gdpr/dsar_search.go:56-58` | 3 |
| **Die Aufbewahrungs-Policy ist eine Eingabemaske ohne Wirkung.** CRUD auf `retention_policies` existiert, kein Worker liest sie je. Grep nach `ApplyRetention|EnforceRetention` → 0 Treffer. | `migrations/000233`; `security_grpc.go:819-961` | 3 |
| **`restore.sh` ist strukturell kaputt und nie ausgeführt** — enthält exakt den MinIO-`tar`-Bug, der das Backup zwei Monate lahmlegte, dazu falsche Compose-Pfade und fehlendes `--env-file`. Zwei Commits seit Projektbeginn: Anlegen + `chmod`. | `restore.sh:38,50` vs. `backup.sh:29-39`; `git log` | 1,5 |
| **Backups liegen nur lokal, unverschlüsselt, teils world-readable.** Kein rclone/restic/borg auf dem Server, gleiche Disk wie die DB (85 % voll). | SSH: `which rclone restic borg` leer; `ls -lh /opt/kmuhub/backups/` zeigt `-rw-r--r--` | 1 |
| **`/health` prüft nur Redis** — bei totem Postgres meldet das Gateway weiter „healthy". Der `PostgresChecker` existiert und wird von 22 anderen Services genutzt, nur nicht vom Gateway. | `cmd/gateway/main.go:135-137`; `health/health.go:57-62` | 0,1 |
| **Kein Alert, wenn ein Backup still fehlschlägt** — genau die Lücke, durch die der MinIO-Ausfall zwei Monate unbemerkt blieb, ist unverändert offen. | `prometheus/rules/alerts.yml`: 8 Regeln, keine zum Backup | 0,5 |
| **`rollback.sh` hat keinen Migrations-Schutz.** `deploy.sh` wurde nach dem 31-Minuten-Ausfall repariert, das manuelle Skript nicht — derselbe Fehlermodus, nur von Hand ausgelöst. | `rollback.sh` vs. `deploy.sh:41-65,222-226` | 0,5 |
| **Das GoBD-Archiv ist nur durch Anwendungsdisziplin unveränderbar.** Kein Trigger, keine Rule, kein REVOKE — und `kmuhub_app` hat via Blanket-Grant volles UPDATE/DELETE. „WORM" ist ein Kommentar, kein Mechanismus. | `migrations/000139:8`; `migrations/000121:49,55-56` | 1 |
| **SMTP-Anbieter ist nicht gegen DE/EU abgesichert** — und die naheliegendste Vorlage empfiehlt US-Anbieter, während das Ansible-Template EU nennt. | `PRODUCTION_TEMPLATE:40-41` vs. `env.production.j2:36` | 0,5 |
| **Der Captcha-Default zeigt auf Cloudflare Turnstile (USA)** — heute inaktiv, aber der bequemste Weg ist der falsche. | `config.go:83-86` | 0,25 |
| **`RUNBOOK.md` ist ein Skelett** — alle vier Incident-SOPs sind TODOs; SSH-Key und Vault-Passwort existieren nur bei Luke; `deploy_user_pubkeys: []`. | `RUNBOOK.md:3,23,36,49,74`; `group_vars/all.yml:4` | 2 |
| **Keine Secret-Rotation, keine Leak-Prozedur** für 12–13 Produktionsgeheimnisse. | `roles/secrets/tasks/main.yml:9-24` | 1 |

### G2 — vor dem ersten zahlenden Kunden

DATEV-EXTF schreibt UTF-8 statt Windows-1252 (Umlaute werden beim Steuerberater zu Mojibake,
`exporter.go:73-76`, 0,5 PT) · EN-16931-Validierung ist Feldprüfung ohne Schematron (3 PT) ·
`internal/idempotency` hat real 0 % Coverage, weil die Tests gegen einen Mock laufen statt gegen
die echte SQL-Logik (1 PT) · Lexware-Webhook ohne Doppelzustellungs-Schutz (0,5 PT) · 22 Route-Dateien
ohne Test, davon 8 in Geld-/Compliance-Pfaden (2 PT) · kein Auto-Update-Kanal (1,5 PT) · kein
Crash-Reporting (1 PT) · **kein Abrechnungssystem** — `PRICING.md` ist eine Tabelle ohne Engine,
die Migration sagt selbst „bookkeeping, not enforcement" (5 PT) · ORBIT/Self-Hosted hat keinen
kundenfähigen Installationsweg, das Ansible-Playbook zeigt auf euer privates Repo (4 PT) ·
Art.-30-Verzeichnis existiert nicht (1 PT) · alle Module werden ohne Reifegrad-Kennzeichnung zum
vollen Preis verkauft (1 PT).

### G3 — bewusst vertagt

95 TS-Fehler in Produktionscode — **die beiden einzigen Crash-Kandidaten sind toter, nirgends
gerouteter Code** (`FirmaDetailPanel.tsx`, `IntegrationsSettingsTab.tsx`); löschen statt reparieren,
Rest ist Typ-Drift. *Auslöser: sobald das `tsc`-Gate in CI repariert ist.* · GoBD-Retention rechnet
mit 10 statt 8 Jahren (rechtlich unschädlich, fällt aber beim Steuerberater auf) · Bexio-Sync ist
echt gebaut, aber produktiv aus — und passt als Schweizer Software ohnehin nicht zur DE-Fokussierung
· Feiertags-API ruft `date.nager.at` (keine Personendaten, aber inkonsistent zum absoluten Claim) ·
Emojis: kein systematisches Problem, der einzige UI-Treffer sind Videocall-Reaktionen.

---

## 5 · Das DE-Versprechen: drei belastbare Fassungen

Du wolltest ausdrücklich Feedback, bevor ihr euch verkalkuliert. Hier ist es.

**Die zentrale Einsicht:** Euer *Produkt* hält das Versprechen bereits — kein Telemetrie-SDK, keine
CDNs, alles self-hosted. Gebrochen wird es von der *Website* und von zwei Konfigurationsdefaults.
Ihr habt kein Architektur-, sondern ein Formulierungs- und Beschaffungsproblem. Das ist billig.

| | **A — „Hosting in Deutschland"** | **B — „Kein Drittlandtransfer" (Empfehlung)** | **C — „Kein Byte verlässt Deutschland"** |
|---|---|---|---|
| Zusage | Server, Datenbank und Dateien liegen in DE (Hetzner Nürnberg) | Zusätzlich: kein Anbieter im Verarbeitungspfad außerhalb der EU, keine US-Muttergesellschaft im Datenpfad | Zusätzlich: ausschließlich deutsche Anbieter, Backups in DE, Self-Hosted als Regelfall |
| Heute erfüllt? | fast — Website bricht ihn | nein — Vercel, Resend, Captcha-Default | nein |
| Aufwand | 1 PT (Website-Text) | **~3 PT** | 8–12 PT + laufende Einschränkung |
| Laufende Kosten | 0 € | **~16–20 €/Monat** (EU-SMTP ab 7 €, DE-Captcha ab 9 €, Website auf eigenem Server ~0 €) | höher, weniger Auswahl, ORBIT-Installer nötig |
| Verkaufskraft | schwach — sagt jeder | **stark und prüfbar** | am stärksten, aber schwer zu halten |

**Empfehlung: B.** Drei Gründe:

1. **A ist schwächer als das, was ihr heute schon behauptet.** Auf A zurückzugehen wäre ein
   Rückschritt hinter die eigene Website — und „Server in Deutschland" sagt jeder Wettbewerber.
2. **C könnt ihr nicht ehrlich halten.** Sobald ein Kunde Slack, Teams oder Bexio anschaltet
   (alles kundengesteuerte Opt-ins in eurem Code), fließen Daten hinaus. Ein absolutes Versprechen
   bricht am ersten Randfall — und genau daran krankt ihr gerade.
3. **B ist mit ~20 €/Monat und drei Tagen erreichbar** und lässt sich vom Kunden nachprüfen. Das
   ist die stärkere Verkaufsposition: nicht die kühnste Behauptung, sondern die einzige, die einem
   IT-Berater standhält.

**Wichtiger als die Technik ist die Formulierung.** „Daten verlassen niemals die EU" ist absolut und
damit bei jedem Randfall falsch. Formuliert stattdessen prüfbar, etwa: *„Anwendung, Datenbank und
Dateien laufen auf Servern in Nürnberg. Wir setzen keine Auftragsverarbeiter außerhalb der EU ein.
Integrationen zu Drittsystemen aktivieren Sie selbst — wir sagen Ihnen vorher, wohin die Daten
gehen."* Das ist schwächer im Ton und stärker im Verkaufsgespräch.

Eine Präzisierung, die die Gegenprüfung geliefert hat: **„Zero-Access-Backup" stimmt für ORBIT
(self-hosted, der Kunde hält `VAULT_MASTER_SECRET`), nicht aber für SaaS** — dort haltet ihr das
Secret und könntet technisch entschlüsseln. Trennt den Claim nach Betriebsmodell, statt ihn zu
streichen.

---

## 6 · Sequenz mit Reifegrad-Gates

Kein Kalender. Jede Etappe hat ein Eintritts- und ein Austrittskriterium; Etappe N+1 beginnt, wenn
N belegt abgeschlossen ist.

**Verfügbare Kapazität:** Luke 12–16 PT in drei Wochen · Nico 20–30 h (QA, kein Produktionscode) ·
Darien (Editor-Rollout) · Bella: 0. **Puffer:** Rechnet einen Tag für einen Deploy-Zwischenfall ein
— das Muster ist dreimal aufgetreten.

### Etappe 0 — „Nichts, was wir sagen, ist widerlegbar" · 3,25 PT · Luke

Die billigste Etappe mit dem größten Risikoabbau. **Beginnt sofort, blockiert alles andere nicht.**

1. `/ki` offline nehmen (0,5) — nicht umschreiben, offline. Kommt zurück, wenn es das Produkt gibt.
2. Beta-Datum und die Anmeldezahlen entfernen (0,5).
3. Impressum korrigieren: kein Handelsregister, keine UG — solange nichts eingetragen ist, gehört
   dort eine natürliche Person hin (0,5).
4. „AVV inkludiert" streichen (Teil von 3).
5. Datensouveränitäts-Claims auf die Fassung aus §5 umschreiben (1).
6. Website von Vercel auf euren Hetzner umziehen — Astro baut statisch, Caddy liegt schon da (1).
7. `docs/pilot0-onboarding/README.md` und `.planning/status-overview.md` auf den realen Stand
   ziehen (0,25).

> **Gate 0:** `curl -I zentria.tech` zeigt keinen US-Anbieter. Jede Behauptung auf der Seite hat
> entweder eine Codestelle oder ist weg. Ein Fremder findet in 30 Sekunden keinen Widerspruch.

### Etappe 1 — „Der Kernpfad hält, was die Demo zeigt" · 4,25 PT · Luke + Nico

Kontakte-Felder (2,5) · Helpdesk-Assignee auf echte Team-Liste (0,5) · E-Rechnungs-Button entweder
verdrahten oder ausblenden (1) · Fuhrpark-i18n-Keys (0,25). Nico verifiziert **gegen das echte
Backend**, nicht gegen Mocks.

> **Gate 1:** Nico legt einen Kontakt mit allen Feldern an, speichert, lädt neu — alles da. Legt
> ein Ticket an, weist es zu, schließt es. Kein Klick meldet Erfolg ohne Wirkung. Screenshots
> gemacht **und angesehen**.

### Etappe 2 — „Wir können das Produkt aushändigen" · 0,5–4 PT · Luke

Hängt an Entscheidung 2 (siehe §8). Desktop-Weg: Installer bauen, Release-Workflow, auf sauberer
VM testen (0,5 + Signatur). Web-Weg: Auth-Persistenz umbauen, 3 Popup-Fenster zu Modals,
Screen-Share auf Browser-API (4).

> **Gate 2:** Jemand außerhalb des Teams startet das Produkt ohne Lukes Hilfe und ohne Warndialog.

### Etappe 3 — „Vor dem ersten echten personenbezogenen Datensatz" · ~10 PT

Consent-Prop verdrahten (1,5) · Kontakt-Löschung-UI + vollständige Kaskade (3,5) · DSAR um Finance,
Meetings, Chat erweitern (3) · `restore.sh` reparieren **und einmal durchführen** (1,5) ·
Offsite-Backup verschlüsselt auf Hetzner Storage Box (1, ~5 €/Monat) · `/health` um Postgres (0,1) ·
Backup-Alert (0,5) · `rollback.sh`-Guard (0,5).

> **Gate 3:** Ein Restore wurde einmal wirklich durchgeführt, mit Zeitmessung. Eine
> Auskunftsanfrage und eine Löschanfrage sind vollständig über die Oberfläche bedienbar.

### Etappe 4 — „Bevor Geld fließt"

UG-Gründung (Notar 300–700 €, Handelsregister ~150 €, **4–8 Wochen bis Eintragung**, Konto
zusätzlich 5–14 Tage — das ist sequenziell, nicht parallelisierbar) · AVV/AGB/TOMs/DSE anwaltlich ·
Art.-30-Verzeichnis · Abrechnung (5 PT) · DATEV-Encoding (0,5).

### Rechnet das auf?

Etappe 0–2 = **8–12 PT**. Bei 12–16 verfügbaren Tagen: **ja, mit Puffer.** Etappe 3 (~10 PT) passt
**nicht** mehr in dieselben drei Wochen — und muss es auch nicht, denn sie hängt am ersten echten
Datensatz, nicht am Kalender. Wer beides in drei Wochen plant, plant falsch.

---

## 7 · Streichliste — was ihr jetzt bewusst nicht anfasst

- **Die 95 TS-Fehler nicht flächendeckend beheben.** Die zwei gefährlichen sind toter Code — löschen,
  fertig. Repariert stattdessen das `tsc`-Gate in CI, sonst wächst die Zahl weiter.
- **Kein EV-Zertifikat kaufen, bevor Entscheidung 2 gefallen ist.** Die Recherche hat die Annahme
  widerlegt, das hinge an der UG (SSL.com bietet einen Sole-Proprietor-Pfad) — aber sie hat auch
  ergeben, dass Microsoft seit 2024/2026 **auch EV keine sofortige SmartScreen-Reputation mehr
  gibt**. Jeder neue Datei-Hash startet bei null. 150–750 €/Jahr für ein Problem, das damit nicht
  verschwindet.
- **Bexio nicht anfassen.** Echt gebaut, produktiv aus, Schweizer Software — passt nicht zur
  DE-Fokussierung. Auslöser: ein CH-Kunde steht vor der Tür.
- **Finanzen nicht als nächstes Editor-Modul.** Die Rollout-Doku plant `finanzen` als Nächstes.
  Nehmt stattdessen **Dokumente und Kalender** — beide Kern, ungegated, „Thin"-Tier, und Dokumente
  ist die stärkste Datensouveränitäts-Erzählung, die ihr habt (Verträge, Rechnungen, Kundendateien
  auf eigenem OnlyOffice). Finanzen hat den höchsten Marktwert *und* das höchste
  Vertrauensschaden-Risiko — nicht in die erste Demo.
- **Kein Abrechnungssystem bauen.** Der erste Kunde bekommt eine Rechnung per Hand. 5 PT für einen
  Kunden, den es noch nicht gibt.
- **Kein Schematron-Validator für EN-16931.** Die Feldprüfung fängt die häufigen Fälle; 3 PT für
  ein Risiko, das erst bei einem Empfänger mit striktem KoSIT-Validator scharf wird.
- **ORBIT/Self-Hosted nicht kundenfähig machen.** 4 PT, und es verkauft sich erst, wenn jemand
  danach fragt.
- **Keine weiteren Module in die Breite.** Ihr habt 33 Modul-Verzeichnisse und einen Entwickler.

---

## 8 · Die drei Entscheidungen, die nur du treffen kannst

### Entscheidung 1 — Was passiert heute mit zentria.tech?

**Empfehlung: `/ki` und die Beta-Zahlen heute Abend offline, der Rest innerhalb von drei Tagen.**

Nicht weil eine Abmahnung wahrscheinlich ist — das Risiko ist real, aber begrenzt (§ 13 Abs. 4 Nr. 2
UWG schließt den Kostenerstattungsanspruch bei Datenschutzverstößen für Unternehmen unter 250
Mitarbeitern aus). Sondern weil **alle vier Pre-Mortem-Linsen unabhängig voneinander denselben Tod
beschrieben haben**: nicht ein Bug, sondern ein IT-Berater beim Interessenten, der 20 Minuten
prüft und „unseriös" in eine Mail schreibt. Das ist kein hypothetisches Risiko — es ist der
wahrscheinlichste Weg, wie dieses Vorhaben scheitert, und er kostet 3,25 Tage.

Die Seite bewirbt derzeit ein Produkt, das ihr nicht habt, unter einer Firma, die es nicht gibt,
mit einem Versprechen, das die eigene Datenschutzerklärung widerlegt. Solange das steht, arbeitet
jeder Tag Entwicklung gegen euch statt für euch.

### Entscheidung 2 — Desktop-Installer oder Web-Auslieferung?

**Empfehlung: Web zuerst, Desktop später.**

Der Renderer ist überwiegend Electron-frei — nur 16 von ~1.234 Dateien berühren Electron-APIs, und
der Blast-Radius ist bekannt: Token-Persistenz über `safeStorage`, drei Popup-Fenster, Screen-Share.
Geschätzt 4 PT. Dafür entfällt: Code-Signing (150–750 €/Jahr, das das SmartScreen-Problem nicht
löst), der Update-Kanal (1,5 PT, der zudem DE-gehostet sein müsste), der Release-Workflow, und die
Hürde „IT-Admin blockt eine unbekannte .exe reflexhaft".

Vor allem aber: **Eine URL kann man in eine Mail schreiben.** Bei leerer Pipeline und Direct Sales
ist das der Unterschied zwischen „ich zeig's Ihnen mal" und „schauen Sie selbst". Der Desktop-Client
bleibt euer Premium-Weg für Piloten — aber er darf nicht der einzige sein.

Gegenargument, das ich nicht verschweige: Der Editor öffnet echte `BrowserWindow`s, und die
Sandbox-Mechanik ist auf Electron gebaut. Prüft das zuerst — wenn der Editor im Web nicht geht,
verliert ihr euer stärkstes Verkaufsargument, und die Entscheidung dreht sich.

### Entscheidung 3 — Welche Fassung des DE-Versprechens?

**Empfehlung: Fassung B** (§5), umgesetzt in Etappe 0 und 3. ~20 €/Monat, ~3 PT.

Der Kern der Empfehlung ist nicht technisch: **Hört auf, absolut zu formulieren.** „Niemals" und
„keine" sind bei jedem Randfall widerlegbar, und ihr habt Randfälle (kundengesteuerte
Integrationen, Feiertags-API, künftige Dienstleister). Eine präzise, etwas schwächere Zusage, die
hält, verkauft besser als eine starke, die ein Berater in zwanzig Minuten kippt.

---

## 9 · Pre-Mortem-Destillat — die drei wahrscheinlichsten Arten zu scheitern

**1. Der Widerspruch wird von außen gefunden, bevor ihr ihn schließt.** *(alle vier Linsen,
Wahrscheinlichkeit hoch)* — Ein IT-Dienstleister, eine Kanzlei oder ein Datenschutz-Blogger prüft
die Website gegen sich selbst. Der Deal ist tot, bevor eine Demo stattfand, und im schlimmsten Fall
öffentlich.
› **Frühwarnsignal:** Jemand fragt in einem Erstgespräch nach dem AVV oder nach eurem Hosting — und
ihr merkt, dass ihr die Antwort googeln müsst. Ab da läuft die Uhr.

**2. Der erste ernsthafte Interessent klickt selbst und verliert Daten.** *(CTO, CFO, CEO)* — In
einer Live-Demo legt jemand einen Kontakt an, füllt Adresse und Mobilnummer aus, speichert. Nach dem
Neuladen sind neun Felder weg. Fremdscham vor Publikum, in genau dem Modul, das euer Wedge ist.
› **Frühwarnsignal:** Nico findet beim ersten Durchklick gegen das *echte* Backend (nicht die
Demo-Mocks) mehr als zwei solcher Brüche. Dann sind es nicht vier, sondern zwanzig.

**3. Alles läuft, bis einmal etwas ausfällt — und der Wiederanlauf scheitert am nie getesteten
Skript.** *(CTO, CFO, Datenschutz)* — Backup läuft seit dem 11.08. wirklich. `restore.sh` trägt
denselben Bug, der das Backup zwei Monate lahmlegte, und wurde nie ausgeführt. Ein Datenverlust
beim ersten echten Kunden ist das Ende der Datenschutz-Erzählung, unabhängig von jedem RLS-Beweis.
› **Frühwarnsignal:** Der Restore-Drill wird zum dritten Mal verschoben, weil „gerade wichtigeres
ansteht". Genau so ist auch der Backup-Ausfall zwei Monate unbemerkt geblieben.

---

## 10 · Der blinde Fleck — was du nicht gefragt hast

Du hast nach Stand, Backend, Frontend, GTM und Legal gefragt. Es fehlt: **Was passiert, wenn der
erste Kunde anruft und du im Hauptjob sitzt?**

Der Plan dafür existiert bereits — als Skelett, das trotzdem die einzige Antwort ist, solange
niemand nachlegt: `docs/pilot0-onboarding/README.md:28` nennt als Support-Kanal „Direkter Slack-/
Telefon-Draht zu Luke". `docs/operations/RUNBOOK.md:9` eskaliert Severity 1 per „Discord-DM +
Telefon an Luke" — kein zweiter Name. Vier von sechs Incident-SOPs sind TODOs. Sentry: „TODO Sprint
5: aktivieren oder verwerfen". Es gibt kein Crash-Reporting, also erfahrt ihr von einem Absturz nur,
wenn der Kunde anruft — auf demselben Kanal, der schon der einzige ist.

Das ist mehr als Bus-Faktor 1. Es ist die Kombination aus keinem Ersatz für dich, keiner Zusage
gegenüber dem Kunden, keinem Alarm und keinem Ablauf — bei 12–16 Tagen im Monat. Der erste ernsthafte
Vorfall trifft ein System, dessen einziger Eskalationspfad ein Telefon ist, das gerade woanders
klingelt.

**Konkret vor dem ersten Kunden, ~1,5 PT:** Sagt zu, was ihr halten könnt (z. B. „Antwort binnen
eines Werktags", nicht „direkter Draht") · schreibt die drei häufigsten Incidents als SOP, sodass
Nico sie ausführen kann · legt eine zweite Kopie von SSH-Key und Vault-Passwort so ab, dass Darien
im Notfall herankommt.

**Und eine Meta-Lücke, die vier Linsen unabhängig genannt haben:** Ihr habt eine ausgeprägte
Verify-First-Kultur — `MEMORY.md` ist voll von Lehren über grüne Gates, die nichts prüfen. Aber
**keine einzige dieser Lehren betrifft die Außendarstellung.** Dieselbe Skepsis, die ihr auf jedes
CI-Gate anwendet, wurde nie auf die Website angewendet. Das ist kein Versäumnis einer Person,
sondern ein fehlender Prozessschritt: *Wer liest die Werbetexte gegen den Code, bevor sie live
gehen?* Heute: niemand.

---

## 11 · Datenbasis, Annahmen, Grenzen

**Gemessen** (eigene Kommandos, nicht abgeschrieben): Coverage gegen frisch migrierte DB unter
`kmuhub_app` — gesamt 60,2 %, `gateway` 46,1 %, `auth` 67,9 %, `idempotency` real 0,0 % · 336
Tabellen auf RLS geprüft · 186/186 Permissions · `tsc -p tsconfig.web.json` → 118 Fehler (95
Produktion, 23 Tests) · `go test` in `einvoice`/`datev`/`gobdarchive`/`bexio` grün · Live-SSH auf
Produktion (Backups, Timer, Disk) · `curl` gegen `zentria.tech` und `/health` · neun UI-Screens
gegen eine echte Demo-Instanz angesehen.

**Recherchiert:** AVV-Pflichtinhalte nach Art. 28 Abs. 3 · UG-Fristen (4–8 Wochen bis Eintragung,
Notar 300–700 €, HR ~150 €, Konto 5–14 Tage) · UG i.G. kann Verträge schließen, **aber die
Handelnden haften bis zur Eintragung persönlich** · UWG § 7 (Telefon im B2B risikoärmer als E-Mail)
· Code-Signing-Preise und die Microsoft-Policy-Änderung · Hetzner-Preisanpassung 15.06.2026 (CPX42
real 69,49 €, CCX33 138,49 € — eure Planungsdokumente rechnen mit ~20 bzw. ~50 €).

**Angenommen, mit Konsequenz falls falsch:**
- *„Legal ist bei null"* wurde als gesetzte Prämisse übernommen, nicht verifiziert. Existiert doch
  ein AVV-Entwurf, entschärfen sich G0-5 und Teile von Etappe 4.
- ~~*Die Beta-Anmeldezahlen* sind vom Repo aus nicht prüfbar.~~ **Nachträglich geklärt (2026-08-12):**
  Das Website-Repo liegt lokal unter `C:\Users\Luke\Documents\zentria-website`. Die Zahlen sind
  hartkodierte Platzhalter, die echte Supabase-Warteliste ist leer (`{"count":0}`, HTTP 200).
  Siehe G0-4. Damit ist auch die letzte Hoffnung auf eine unbemerkte Pipeline widerlegt.
- **Neu belegt:** Die Anmeldung schickt die Willkommensmail über **Resend (USA)** —
  `src/pages/api/waitlist.ts` importiert `Resend` direkt. Echte E-Mail-Adressen von Interessenten
  fließen also real über einen US-Dienstleister, nicht nur laut Datenschutzerklärung. Das ist der
  harte Beleg für G0-1, den der Lauf mangels Repo-Zugriff nicht führen konnte.
- *„Bella"* wurde mit 0 Tagen kalkuliert.

**Nicht geprüft (offene Lücken):** Das Website-Repo selbst wurde nie ausgecheckt, nur die
gerenderte Seite · kein KoSIT-Validator gegen die erzeugten Rechnungen · `restore.sh` wurde nicht
wirklich ausgeführt (die Kaputtheit ist aus Code plus dokumentiertem Präzedenzfall belegt) · die
**wirklich leere** Erstkontakt-Instanz wurde nie gesehen, nur die befüllte Demo · Finanzen nur
oberflächlich (ein Bruch wie bei Kontakte ist dort nicht ausgeschlossen) · 6 von 33 Modulen
stichprobenartig geprüft.

**Ein ungelöster Widerspruch:** Die neu gemessene Coverage weicht bei `internal/security` (72,8 %
statt 79,5 %) und `internal/biz` (63,5 % statt 70,6 %) von den geführten Zahlen ab. Vermutlich
unterschiedliche Paket-Abgrenzung, nicht aufgeklärt. Wer die Zahlen extern nennt, sollte sie vorher
mit einer Methode neu messen.

**Und ein Befund über die Befunde:** Drei Repo-Quellen von *gestern* — `status-overview.md`
(„Launch 2026-09-01, noch 21 Tage, Legal ist der einzige Blocker"), `NEXT-SESSION-PROMPT.md`,
`docs/pilot0-onboarding/README.md` — beschreiben eine Lage, die es nicht mehr gibt. Dazu führt
`.planning/backend-gaps.md:264` die Kalender-Terminbuchung seit dem 09.06. als „ERLEDIGT", während
sie ohne öffentliche Seite gar nicht nutzbar ist. Die Statusdokumente hinken nicht nur hinterher —
sie widersprechen sich gegenseitig und dem Code. Solange das so bleibt, ist jede Planung, die auf
ihnen aufbaut, ein Blindflug.
