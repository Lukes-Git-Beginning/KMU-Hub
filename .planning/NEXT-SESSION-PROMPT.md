# Session-Ziel: Etappe 0 zu Ende bringen

Stand: **2026-08-12, nach dem Lagebild-Tag.** Es gibt **kein Launch-Datum mehr** — das Ziel ist
**Produkt 1.0.0** nach Reifegrad-Gates. Wenn dir irgendein Dokument ein Datum nennt, ist das Dokument
veraltet, nicht die Lage.

**Lies zuerst:** `.planning/launch-lagebild-2026-08-12.md` — §3 (was 1.0.0 heißt, fünf prüfbare
Kriterien) · §4 (Befunde G0/G1/G2 mit Belegen) · §6 (Etappen 0–4 mit Gates) · §7 (Streichliste:
was ihr bewusst *nicht* anfasst) · §8 (die drei Entscheidungen).

---

## Was am 2026-08-12 passiert ist

**Planungsgrundlage korrigiert** (`ac3ce855`). `CLAUDE.md`, `.planning/status-overview.md` und der
Generator `status-overview.prompt.md` sind auf das Gate-Modell umgestellt; `docs/ROADMAP.md`,
`docs/BACKEND-LAUNCH-PLAN.md` und `docs/pilot0-onboarding/README.md` tragen einen ⚠-Kopf, der
benennt, was entwertet ist und was gilt. Der Generator war der eigentliche Root Cause — er schrieb
das 01.09-Datum und den ZFA-Meilenstein als Pflichtbestandteil des Gantt-Diagramms vor.

**Website entschärft** (`ffb5bb6` im Repo `zentria-website`, ging per Vercel-Git-Integration
sofort live):
- `/ki` offline — geparkt als `src/pages/_ki.astro`, 302 auf `/funktionen`, Header- und
  Footer-Eintrag raus
- **Nachtrag, beim Screenshot-Durchgang gefunden:** Die *Startseite* trug dieselbe Behauptung
  noch einmal, über dem Falz und mit eigenem CTA („Qwen3-14B läuft auf einer GPU, die nur dir
  gehört" — direkt neben „keine OpenAI-API", was sich selbst widerspricht). Sektion, Feature-Liste
  und totes CSS entfernt. **Zwei Lehren:** Das Lagebild hatte nur `/ki` und `/beta` per WebFetch
  geprüft, nicht die Startseite. Und der Grep nach dem überfälligen Datum suchte „Beta startet",
  während dort „Beta-Start" stand — eine Schreibvariante hat den Befund zweimal durchrutschen
  lassen. Bei Claim-Audits nach Wortstämmen greppen, nicht nach Phrasen.
- Startseiten-Badge „Beta-Start Juni 2026" und der Timeline-Eintrag auf `/ueber-uns`
  („Juni 2026 · aktiv · erste zahlende Kunden" — es gibt keine) nennen jetzt einen Zustand
  statt eines Datums
- Erfundene Beta-Belege raus (37 Pioneer, 13 Anmeldungen/Woche, 13 Plätze in 7 Tagen, drei
  Avatar-Initialen, „Beta startet im Juni 2026")
- „AVV inkludiert" und „Kunden-AVV als Download" raus (der Hetzner-AVV ist echt und bleibt)
- `vercel.json` → `"regions": ["fra1"]`. Vorher liefen **alle** Functions in `iad1/Virginia`:
  `/api/waitlist` (Interessenten-Mailadressen), `/book/[slug]` (Endkunden-Buchungen), die
  Cosmi-Proxy-Routen
- Verwaiste `zentria.tech/reset-password` samt Proxy entfernt

**Reset-Link endgültig dicht** (`7e3da706`). Die Produktions-Env war bereits seit dem 11.08.
23:03:54 UTC korrekt — das Lagebild hat wenige Minuten davor gemessen. Heute noch der letzte
abweichende Default: die Ansible-Vorlage leitete die URL aus `pilot_domain` ab und hätte den Fehler
bei jeder Neu-Provisionierung zurückgeholt. Alle vier Defaults sind jetzt deckungsgleich.

**Pre-Commit-Hook** (`251efbbc`). `check-no-secrets.sh` blockte jede Commit-Message, die eine
Env-Datei als Fließtext erwähnte, weil das `git add .*`-Capture gierig bis Zeilenende lief. Scope
korrigiert, Prüfliste unverändert, sieben Testfälle grün.

---

## Was als Nächstes dran ist

### Etappe 0 zu Ende bringen (~2,25 PT) — blockiert nichts anderes

1. **Impressum (G0-2, 0,5 PT).** „Amtsgericht Mainz, Eintragung beantragt" für eine UG, die es nicht
   gibt. Abmahnfähig nach DDG, und die Seite verarbeitet bereits Kontaktdaten unter einer
   Rechtsträger-Fiktion. Solange nichts eingetragen ist, gehört dort eine natürliche Person hin.
   **Braucht Lukes Namen und eine ladungsfähige Anschrift — nicht ohne ihn machen.**
2. **DE-Claims auf Fassung B umschreiben (1 PT).** `index.astro:34` („Keine US-Cloud, keine
   Drittländer"), `:224` („Daten verlassen niemals die EU"), `TrustSection.astro`,
   `OrbitConfigurator.tsx:111`. Der Wortlautvorschlag steht in Lagebild §5. **Hört auf, absolut zu
   formulieren** — „niemals" ist bei jedem Randfall widerlegbar, und es gibt Randfälle.
   ⚠ Setzt Entscheidung 3 voraus.
3. **Website von Vercel auf Hetzner (1 PT).** Der eigentliche Bruch. `fra1` hat die *Rechenleistung*
   nach Frankfurt geholt, aber Vercel Inc. bleibt ein US-Auftragsverarbeiter, und
   `api/waitlist.ts` schickt Interessenten-Mails weiter über **Resend (USA)**. Astro baut statisch,
   Caddy liegt auf dem Server schon da. Danach ist auch die Datenschutzerklärung wieder wahr.

> **Gate 0:** `curl -I zentria.tech` zeigt keinen US-Anbieter. Jede Behauptung auf der Seite hat
> entweder eine Codestelle oder ist weg. Ein Fremder findet in 30 Sekunden keinen Widerspruch.

### Danach Etappe 1 (~4,25 PT) — der Kernpfad

Kontakte-Felder (G0-6, 2,5) · Helpdesk-Assignee auf die echte Team-Liste (G0-7, 0,5 — der Hook
existiert bereits) · E-Rechnungs-Button verdrahten oder ausblenden (G0-8, 1) · Fuhrpark-i18n-Keys
(G0-9, 0,25) · Buchungslink auf die existierende Seite statt auf die nicht existente Subdomain
(G0-10, 0,15). Nico verifiziert **gegen das echte Backend**, nicht gegen Mocks.

---

## Zwei Entscheidungen, die noch bei Luke liegen

- **#2 Desktop-Installer oder Web-Auslieferung?** **Weitgehend geklärt, aber nicht von mir** —
  Darien hat am selben Tag `.planning/auslieferungsmodell-2026-08-12.md` gepusht (`db837d6d`).
  Kernbefunde dort, gegen den Code geprüft: die `.exe` ist ohnehin nur eine Hülle um dieselbe
  React-App über HTTP, der Umbau schrumpft von 15 Renderer-Dateien auf im Wesentlichen **eine**
  (Token-Speicherung), und **der Editor-Einwand des Lagebilds ist widerlegt** —
  `customization-sync.ts` nutzt `BroadcastChannel` und `localStorage`, beides Web-Standards.
  **Lest das, bevor ihr an Etappe 2 rührt.** Darien setzt dort außerdem Prämissen, die den
  Etappenplan verschieben: eigener Hetzner-Server pro Kunde statt Multi-Tenant-SaaS, erster Pilot
  unentgeltlich, ORBIT vorerst raus, nur gebuchte Module ausliefern.
- **#3 Welche Fassung des DE-Versprechens?** Empfehlung B („kein Drittlandtransfer"), ~3 PT,
  ~20 €/Monat. Blockiert Etappe-0-Punkt 2. ⚠ Darien plant Website und Preisgestaltung „in den
  nächsten 1–2 Tagen separat" — **stimmt euch ab, bevor doppelt an der Seite gearbeitet wird.**

Solange #3 offen ist, wird `docs/ROADMAP.md` **nicht** neu geschrieben; eine Roadmap würde die
Annahme festschreiben. Nach der Runde mit Darien ist der richtige Zeitpunkt dafür.

---

## Fallstricke

- **Statusdokumente hinken dem Code hinterher — miss selbst.** Genau daran ist der gestrige Tag
  gehangen, und selbst das Lagebild trug schon einen überholten Befund (die Reset-Env war bereits
  gefixt, als es sie als kaputt meldete). Vor jedem Arbeitspaket verifizieren.
- **Die Website ist ein eigenes Repo:** `C:\Users\Luke\Documents\zentria-website`. Ein Audit nur im
  Produkt-Repo produziert Fehlbefunde — so entstand G0-10.
- **Push nach `main` im Website-Repo = sofortiges Production-Deployment** (Vercel-Git-Integration).
  Erst `npm run build`, dann pushen.
- **Im KMU-Hub-Repo deployt ein Commit nur, wenn er `backend/**` berührt** — `ci.yml` hat einen
  paths-Filter, und `cd.yml` hängt an `workflow_run` von CI. Doku-, `docs/`- und `deploy/`-Commits
  lösen nichts aus. Das ist Absicht, aber es heißt auch: eine Änderung an `deploy/` wird erst beim
  nächsten Backend-Deploy wirksam.
- **`X-Vercel-Id` lesen:** `<edge>::<compute>::<hash>`. Zwei Segmente heißen, die Function lief in
  einer anderen Region als die Edge. `fra1::iad1::` = Frankfurt-Edge, Virginia-Compute.
- **`go test` ohne `DATABASE_URL` ist kein Gate**, die Rolle muss `kmuhub_app` sein.
- **Kein `reset --hard`, kein Force-Push.** CI-Rot-Recovery ist `git revert <sha>`.

## Offene Posten, die nicht in Etappe 0/1 fallen

- **Reset-Flow nie end-to-end geprüft** — Mail anfordern → Link klicken → Passwort setzen → Login,
  gegen einen echten Mailversand. Die Konfiguration ist verifiziert, der Durchlauf nicht.
- **Schichttausch-UI** (`SchichtenPage.tsx:1941`) bietet Partner an, die nicht auf der Schicht
  stehen; der Antrag scheitert erst bei der Genehmigung.
- **`restore.sh` ist strukturell kaputt und nie ausgeführt** (Lagebild §4, G1). Der Restore-Drill
  wurde schon zweimal verschoben — das ist genau das Muster, durch das der Backup-Ausfall zwei
  Monate unbemerkt blieb.
