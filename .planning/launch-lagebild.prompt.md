# Prompt — Launch-Lagebild & 21-Tage-Plan (Cosmi/Zentria)

> Präskriptives Gegenstück zu `.planning/status-overview.prompt.md`. Dort: „Wo stehen wir?" ohne
> Wertung. Hier: **„Was hält, was bricht, was tun wir in den verbleibenden Tagen — und was lassen
> wir bewusst liegen?"**
>
> Aufruf: Inhalt dieser Datei als Prompt an Claude Code in diesem Repo, in einer frischen Session.
> Der Lauf ist als **Multi-Agent-Nachtlauf** freigegeben — siehe §3.

---

## 0 · Rolle, Auftrag, Kalibrierung

Du bist nicht mein Assistent, sondern das Gremium, das ich mir für diese Woche kaufen würde: ein
**CTO**, der schon zwei B2B-SaaS-Produkte durch den ersten Echtkunden gebracht hat, ein **CFO**, der
weiß, was ein verschobener Launch in einem bootstrapped Team kostet, und ein **CEO**, der schon
erlebt hat, wie ein zu breites Produktversprechen ein 3-Personen-Team zerreibt. Sprich aus diesen
drei Rollen. **Wo sie sich widersprechen, mach den Widerspruch sichtbar, statt ihn zu glätten** —
der CTO will Härtung, der CEO will Kunden, der CFO will beides billiger. Genau an diesen Reibungen
liegt die Entscheidung.

**Kalibrierung — wichtig, halt dich daran:**
- Sei **eher pessimistisch als optimistisch**. Wenn du zwischen „knapp machbar" und „wird nicht
  fertig" schwankst, sag „wird nicht fertig" und begründe.
- Aber **stampf mich nicht in den Boden**. Ich brauche eine Lage, keine Abrechnung. Benenne
  ausdrücklich, was hier **wirklich trägt** — es gibt substanziell gebaute Dinge, und die zu
  übersehen führt zu genauso falschen Entscheidungen wie Schönfärberei.
- **Keine generischen SaaS-Ratschläge.** Jede Aussage muss an diesem Repo, diesem Team, diesem
  Markt hängen. „Ihr solltet Product-Market-Fit validieren" ist wertlos; „ZFA nutzt heute nur den
  Dialer, also ist der Wedge X" ist eine Aussage.
- **Nenne Konfidenz.** Bei jeder Kern-Aussage: hoch / mittel / niedrig — und woran sie hängt.
- **Rechne mit Zahlen**, wo Zahlen existieren. Wo du schätzen musst, markiere „(geschätzt)" und sag,
  was die Schätzung kippen würde.

---

## 1 · Zuerst: die Fakten, die dieses Repo nicht kennt

**Bevor du irgendetwas analysierst**, frag mich per `AskUserQuestion` — in höchstens zwei Runden à
vier Fragen. Ohne diese Antworten steht die halbe Priorisierung auf Sand. Priorisiere in dieser
Reihenfolge:

1. **Was genau passiert am 01.09.?** Ein warmer Design-Partner (ZFA) auf einer eigenen Instanz mit
   *echten* personenbezogenen Daten? Nur Testdaten? Oder öffentlicher Verkaufsstart mit
   Selbstregistrierung über die Website? — Das ändert die Legal-Anforderungen fundamental und ist
   deshalb Frage eins, nicht Frage fünf.
2. **UG-Status, faktisch:** Notartermin stattgefunden? Handelsregister-Eintragung beantragt oder
   erfolgt (HRB-Nummer)? Geschäftskonto aktiv? Stammkapital eingezahlt? Das Repo widerspricht sich
   hier (01.05. vs. 01.06. vs. „UG i.G." am 11.08.) — **rate nicht**.
3. **Legal-Ressourcen:** Anwalt beauftragt oder in Aussicht, mit welchem Budget und welcher
   Vorlaufzeit? Oder DIY mit Vorlagen? Gibt es bereits AVV-, AGB- oder TOM-Entwürfe irgendwo
   außerhalb des Repos?
4. **Reale Kapazität bis 31.08.:** Wie viele echte Arbeitstage hat Luke (Urlaub, Hauptjob,
   Krankheitsrisiko)? Wie viele Stunden liefert Nico tatsächlich, und ist sie ausschließlich QA oder
   auch Kundenkontakt? Ist Darien in diesen drei Wochen verfügbar?

Wenn Budget für eine zweite Runde bleibt: ZFA-Zusage verbindlich oder mündlich? Wer unterschreibt
dort? · Ist die Pilot-Instanz (CCX33 + TURN) inzwischen bestellt? · Gibt es außer ZFA benannte
Interessenten mit Namen und Zeitpunkt?

Alles, was ich nicht beantworte, führst du im Report als **markierte Annahme** mit der Konsequenz,
falls sie falsch ist.

---

## 2 · Grundregel: verify-first

**Die Statusdokumente in diesem Repo hinken dem Code hinterher — regelmäßig und in beide
Richtungen.** Es sind schon Dinge als „offen" geführt worden, die längst liefen, und als „fertig",
die nie funktioniert haben (das MinIO-Backup lief zwei Monate lang nie und loggte „non-critical").
Miss selbst. Zitiere Dateipfad und Zeile. Wenn Doku und Code auseinandergehen, **benenne die Lücke
als eigenen Befund** — sie ist ein Symptom, nicht nur ein Fehler in der Doku.

Repo-spezifische Fallen, an denen schon Analysen gescheitert sind:
- **„Grün" ≠ korrekt.** Hat hier bereits Stubs, gRPC-Umgehung und nicht regeneriertes Proto verdeckt.
- **`npx tsc --noEmit` im Desktop prüft wegen der Solution-`tsconfig` null Dateien.** Der CI-Schritt
  ist still grün. Nutze `tsc -p tsconfig.web.json --noEmit`.
- **`go test` ohne `DATABASE_URL` ist kein Gate**, und die Rolle muss `kmuhub_app` sein — `kmuhub`
  hat BYPASSRLS und winkt jede RLS-Lücke durch.
- **`//go:build integration` ist weder PR-Gate noch Coverage.** Vor jedem „ungetestet"-Schluss nach
  Build-Tags greppen.
- **`ci.yml` filtert auf `paths: ["backend/**", ...]`.** Reine Frontend- oder Doku-Commits lösen
  weder CI noch CD aus — `main` kann Commits tragen, die Produktion nicht kennt.
- **`TestOpenAPIRouteDrift` ist keine Spec-Validierung.**
- **Security Review meldet auf großen PRs fehl-offen `success` bei 0 Findings** (HTTP 406).
- **Ein RPC, der existiert, ist nicht erreichbar.** Prüf die Kette FE-Hook → Gateway-Route →
  gRPC-Methode → Repository → Migration. Eine `.proto`-Zeile ist kein Feature.
- **PowerShell 5.1:** lange Payloads an native EXEs über STDIN, nie als Argument.

Nützliche Startpunkte (nicht abschließend):

```bash
git log --oneline -40 && git branch -a
ls backend/cmd/ | wc -l
ls backend/migrations/*.up.sql | tail -1
grep -cE "^  /" backend/api/openapi.yaml
curl -s https://app.zentria.tech/health | jq '.status, .commit'
cd desktop && npx tsc -p tsconfig.web.json --noEmit 2>&1 | tail -5
```

---

## 3 · Ausführungsmodus: Multi-Agent-Nachtlauf (freigegeben)

**Ich autorisiere hiermit ausdrücklich einen großen Workflow-Lauf.** Zeit ist da (die Nacht), Kosten
sind für diesen einen Lauf kein Kriterium. Das übersteuert die Standard-Größenrichtlinie für
Workflows — geh in die Breite und in die Tiefe, statt zu sparen. **Aber:** kein Blindflug — nutze
`log()` großzügig, damit ich morgens den Verlauf nachvollziehen kann.

Vorgeschlagene Zerlegung (Struktur ist ein Vorschlag; wenn du eine bessere siehst, nimm sie und
begründe kurz, warum):

**Phase 1 — Messen, parallel, ein Reader je Domäne.** Jeder liefert strukturiert: Befund · Beleg
(Pfad:Zeile oder Kommando-Output) · Einstufung · Restaufwand in Personentagen · Konfidenz.

- **Finance-Block (Wellen 5–7).** Die wichtigste Einzelfrage des Laufs. `docs/BACKEND-LAUNCH-PLAN.md`
  führt E-Rechnung (XRechnung-UBL + ZUGFeRD 2.x, EN-16931), DATEV-EXTF + GoBD-WORM-Archiv und
  Bexio-OAuth2-Sync mit Deadline ≤01.09 — **ohne ✅-Marker**, anders als Wellen 0/8/9. Die Pakete
  `backend/internal/biz/{einvoice,gobdarchive,datev,bexio}` existieren. Frage: End-zu-Ende
  erreichbar (Route → Service → Repo → Migration → FE), oder Gerüst? Erzeugt der Generator eine
  validierbare EN-16931-Rechnung? Ist der EXTF-Export gegen die DATEV-Spec geprüft (Header,
  Windows-1252)? Ist WORM wirklich unveränderbar oder nur so benannt?
- **Backend-Qualität & Trust-Boundary.** Coverage neu messen, nicht abschreiben. `internal/gateway`
  46,0 % als schwächstes Kernpaket, `internal/idempotency` 0,0 %, 29 von 87 `route_*.go` ohne
  Testdatei. Frage nicht „ist die Zahl hoch genug", sondern: **welche der ungetesteten Pfade sind
  Auth-, RBAC-, Tenant- oder Zahlungs-Pfade?** Prüf RLS-Regression, Consent-Enforcement (Email
  braucht `contact_id`-Plumbing), Idempotenz auf POST.
- **Modul-Abdeckung real.** `.planning/status-overview.md` §2 führt 17 operative Module als „voll
  gewired". Stichprobe mindestens sechs davon über die volle Kette, bevorzugt die, die ein
  Pilotkunde am Tag 1 anfasst. Was ist nominell verdrahtet, aber praktisch tot? (Bekannt: die
  Schichttausch-UI bietet Partner an, die gar nicht auf der Schicht stehen — der Antrag scheitert
  erst bei der Genehmigung. Solche Fälle suchst du.)
- **Frontend & Desktop-Auslieferung.** 118 tsc-Fehler (`tsc -p tsconfig.web.json`) nach
  Produktionscode vs. Tests klassifizieren; die Produktionsfälle einzeln bewerten. Dann die Frage,
  die in **keinem** Plandokument steht: **Wie kommt die App auf den Rechner des Kunden?**
  `desktop/electron-builder.yml` + `build:win` existieren — aber Code-Signing (Windows-EV-Zertifikat
  hat Wochen Vorlauf und kostet), Installer-Verteilung, Update-Kanal, Crash-Reporting? Auto-Update
  steht in P3. Recherchier den realistischen Vorlauf für ein Signing-Zertifikat.
- **Betrieb & Wiederanlauf.** Backup **und Restore** — ein Backup, das nie zurückgespielt wurde, ist
  kein Backup; das MinIO-Archiv existiert erst seit dem 11.08. Dazu: drei 503-Vorfälle durch das
  Auto-Rollback-Muster, Instanz-pro-Pilot (Ansible steht, CCX33 + TURN nicht bestellt),
  Secrets-Rotation, Alerting-Abdeckung, Bus-Faktor 1.
- **Legal-Fläche im Produkt.** Nicht die Verträge — was das Produkt können muss: Art.-30-Verzeichnis,
  Auskunft/Export/Löschung real durchführbar (der Audit-Viewer war bis Lauf 9 tot), Consent-Records,
  Aufbewahrung (GoBD 8 Jahre vs. Löschpflicht — der Konflikt ist real), Sub-Prozessoren im Stack
  (Hetzner, Resend, Vercel, LiveKit self-hosted), TOM-Dokument, Website-Consent auf zentria.tech.
  **Websuche erlaubt** für Art.-28-Pflichtinhalte und aktuelle Aufsichtspraxis.
- **GTM & Markt.** 17 Module, 16 der 17 Feature-Flags default OFF — der Flag-Mechanismus ist ein
  fertiger Hebel für einen schmalen Markteinstieg. Preismodell Modul×User (`docs/PRICING.md`),
  Zielsegment Dienstleister/Agenturen, Wettbewerb HubSpot/Pipedrive/weclapp/Odoo. **Websuche
  erlaubt** für aktuelle Preise und Positionierung.
- **Außendarstellung.** `zentria.tech` gegen den realen Delivery-Stand (Repo
  `Lukes-Git-Beginning/zentria-website`, Astro/Vercel). Steht dort etwas, das wir am 01.09. nicht
  halten können — WASM-Plugins, Modul-Liste, Mobile?

**Phase 2 — Adversarial verify.** Jeder Befund der Kategorie „fertig", „blockiert" oder „kritisch"
geht durch 2–3 unabhängige Skeptiker mit der Aufgabe, ihn zu **widerlegen**; im Zweifel gilt er als
widerlegt. Ein „ist fertig" braucht einen belegten Codepfad, kein Dokument.

**Phase 3 — Pre-Mortem-Panel.** Vier Agenten, vier Linsen (CTO / CFO / CEO / Datenschutz), jeder
unabhängig: *„Es ist der 15. Oktober 2026. Der Launch ist schiefgegangen. Schreib das Post-Mortem."*
Was in mehreren Post-Mortems unabhängig auftaucht, ist dein Top-Risiko.

**Phase 4 — Synthese.** Priorisierung, 21-Tage-Plan, Streichliste, Wedge-Empfehlung,
Legal-Reihenfolge. Diese Phase machst du selbst, nicht delegiert.

**Phase 5 — Completeness-Critic.** Ein letzter Agent: Welche Modalität wurde nicht geprüft, welche
Behauptung nicht verifiziert, welche Quelle nicht gelesen? Was er findet, arbeitest du nach oder
führst es explizit als Lücke.

---

## 4 · Die sechs Fragen, die ich beantwortet haben will

### A · Wo stehen wir wirklich, 3 Wochen vor Launch?

Ein ehrlicher Absatz, kein Kennzahlen-Teppich. Wenn du heute entscheiden müsstest: **Hält der
01.09.?** Für welche Launch-Definition hält er, für welche nicht? Wenn er nicht hält — was ist das
kleinste Datum, das hält, und was kostet die Verschiebung (nicht nur Geld: Momentum bei ZFA,
Glaubwürdigkeit im Team, das dritte verschobene Launch-Datum in Folge)?

### B · Backend: was muss noch verdrahtet werden, wie steht die Testabdeckung?

Trenn sauber: **fehlt** (existiert nicht) · **existiert, aber nicht erreichbar** (kein Route/Flag/FE)
· **erreichbar, aber ungeprüft**. Für die Abdeckung interessiert mich nicht der Prozentwert, sondern:
*Welche ungetesteten Pfade würden bei einem echten Kunden zuerst weh tun?* Und: Ist die Abdeckung
überhaupt noch der Engpass, oder ist es inzwischen Korrektheit? (Lauf 8 hat zehn Produktionsbugs
ausgerechnet in den Paketen mit der **höchsten** Coverage gefunden.)

### C · Frontend: was muss noch geschehen?

Produktionsrelevante tsc-Fehler, tote oder halbtote Modul-Pfade, Auslieferung und Signierung,
Erstkontakt-Flows (Login, Onboarding, leere Zustände beim allerersten Tenant — was sieht ein Kunde,
dessen Datenbank leer ist?). Was davon ist Launch-relevant und was ist Kosmetik, die ich nach dem
Launch machen kann?

### D · GTM: mit welchem Kern gehen wir an den Markt?

Wir können nicht mit 17 Modulen gegen HubSpot antreten — dafür sind wir zu klein, und ein breites
Versprechen erzeugt breite Support-Last bei einem Ein-Personen-Entwicklerteam. **Empfiehl mir einen
Wedge**, keine Optionsliste:

- Welche 3–5 Module sind zum Launch **ON**, alles andere bleibt hinter dem Flag?
- Warum genau diese? (Reife × Nutzen-Dichte × Verteidigbarkeit gegen die Großen × Nähe zum
  ZFA-Warm-Einstieg über den Dialer)
- Wie sieht der Land-and-Expand-Pfad aus — womit erweitern wir bei Kunde 2, 5, 10?
- Was ist der Satz, mit dem wir gegen HubSpot/Pipedrive gewinnen, ohne über Featurezahl zu reden?
- Wie skaliert der „1 Woche Onsite-Prozessanalyse"-USP bei genau einem Entwickler? Ab welcher
  Kundenzahl bricht er, und was ist der Ersatz?
- Was kostet der schmale Einstieg im Preismodell (Modul×User) — trägt das den ersten Kunden
  überhaupt, oder verschenken wir Marge, um Referenzen zu kaufen? (CFO-Linse.)

### E · Legal: UG-Anmeldung zuerst oder Rechtsprüfung zuerst?

Meine offene Frage — beantworte sie als **Sequenz mit Abhängigkeiten**, nicht als Liste:

- Was hängt zwingend am eingetragenen Rechtsträger (Impressum mit HRB, AVV als Vertragspartei,
  Geschäftskonto, Rechnungsstellung) — und was **nicht**? Kann eine UG i.G. bereits wirksam
  Verträge schließen, und was ändert das an der Reihenfolge?
- Welche Legal-Artefakte brauche ich **vor dem ersten echten personenbezogenen Datensatz**: AVV
  nach Art. 28 (mit welchen Pflichtinhalten), TOMs, Verzeichnis der Verarbeitungstätigkeiten,
  Löschkonzept, Sub-Prozessoren-Liste, Datenschutzerklärung, AGB, Impressum? Sortier sie nach
  „ohne das kein Kundendatensatz" vs. „ohne das keine Rechnung" vs. „kann im September nachkommen".
- Was ist realistisch in 20 Tagen zu bekommen — Notartermin- und Handelsregister-Laufzeiten,
  Anwaltsvorlauf, Kontoeröffnung? **Recherchier die realen Fristen**, schätz sie nicht.
- Was ändert sich, wenn der 01.09.-„Launch" nur ein kostenloser Design-Partner-Pilot mit einem
  warmen Kontakt ist statt eines kommerziellen Verkaufsstarts? Ist das ein legitimer Weg, Legal
  vom kritischen Pfad zu nehmen — oder wäre das Selbstbetrug?
- Cold Outreach im Direct-Sales: was ist nach UWG §7 im B2B zulässig?

**Kein Anwaltsersatz.** Sag klar, was ich mit einem Anwalt klären muss und was ich selbst kann. Ein
Satz Disclaimer reicht — keine Absicherungs-Prosa.

### F · Der blinde Fleck (den ich nicht gefragt habe)

Ich habe fünf Bereiche abgefragt. Sag mir, **was ich nicht gefragt habe und hätte fragen müssen** —
Support- und Incident-Prozess ab dem ersten echten Kunden, Onboarding-Material, Bus-Faktor 1,
Preis-Durchsetzung, Kundendaten-Migration aus dem Altsystem, das Verhalten des Teams unter der
ersten Beschwerde. Ein Bereich, gut begründet, ist mehr wert als sechs aufgezählte.

---

## 5 · Was ich als Antwort will

Ein Markdown-Dokument, Deutsch (Umlaute korrekt), Code/Identifier englisch. Aufbau:

1. **Lagebild in 5 Sätzen.** Hält der Launch, ja oder nein, und woran es hängt. Diese fünf Sätze
   muss ich Darien und Nico vorlesen können.
2. **Was wirklich trägt** — 3–5 Punkte, kurz. Nicht als Trostpflaster, sondern weil ich auf diesen
   Dingen aufbauen soll statt sie nochmal anzufassen.
3. **Priorisierte Befunde** in vier Klassen, innerhalb jeder Klasse nach Schadenshöhe sortiert:
   - **K0 — hält den Launch auf.** Ohne das kein Go-Live am 01.09., egal in welcher Definition.
   - **K1 — muss vor dem ersten echten personenbezogenen Datensatz.** Nicht unbedingt vor dem
     Datum, aber vor echten Daten.
   - **K2 — in den ersten 30 Tagen danach.**
   - **K3 — bewusst vertagt, mit Datum und Auslöser fürs Nachholen.**

   Je Eintrag: *Befund · Beleg (Pfad:Zeile oder Kommando) · Konsequenz, wenn wir es lassen ·
   Aufwand in Personentagen · Owner (Luke/Nico/Darien/extern) · Konfidenz*.
4. **21-Tage-Plan mit Kapazitätsrechnung.** Rechne zuerst vor, wie viele Personentage überhaupt
   existieren (meine Antworten aus §1; kalkuliere Puffer für Krankheit und mindestens einen
   Deploy-Zwischenfall ein — das Muster ist dreimal aufgetreten). Dann verteile K0 und K1 auf
   Tagesscheiben mit Woche-1/2/3-Meilensteinen und einem Freeze-Datum, ab dem nur noch getestet und
   nicht mehr gebaut wird. **Wenn die Arbeit nicht in die Tage passt, sag das im Klartext und
   schneide** — nicht optimistisch schätzen, damit es aufgeht.
5. **Streichliste: was wäre überstürzt.** Explizit, was wir in diesen drei Wochen **nicht** anfassen,
   obwohl es verlockend ist — und warum das Weglassen die bessere Entscheidung ist. Diese Liste ist
   mir genauso wichtig wie die Todo-Liste.
6. **Die drei Entscheidungen, die nur ich treffen kann** — je mit deiner Empfehlung und einer
   Begründung, nicht mit Optionen. (Meine Vermutung: Launch-Definition, GTM-Wedge,
   Legal-Reihenfolge. Wenn du andere für wichtiger hältst, nimm deine.)
7. **Pre-Mortem-Destillat** — die 3 wahrscheinlichsten Arten, wie das hier schiefgeht, jeweils mit
   dem Frühwarnsignal, an dem ich es zuerst merken würde.
8. **Datenbasis & Annahmen** — was gemessen, was recherchiert, was angenommen; jede offene Annahme
   aus §1 mit ihrer Konsequenz.

Ablage: `.planning/launch-lagebild-<YYYY-MM-DD>.md`. Fass keinen Produktionscode an — dieser Lauf
liest und denkt, er baut nicht. Am Ende zusätzlich die fünf Sätze aus (1) und die K0-Liste direkt in
den Chat, damit ich sie ohne Datei-Öffnen sehe.

---

## 6 · Quellen (lies sie, verlass dich nicht auf sie)

| Quelle | Wofür | Vorsicht |
|---|---|---|
| `.planning/status-overview.md` | dichtester Ist-Stand, selbst gemessen | Datum prüfen — die Vorfassung war nach fünf Tagen in jedem Punkt überholt |
| `docs/ROADMAP.md` | Single Source of Truth, §2 Stand · §4 P0–P3 · §7 Gate S5 | P0/P1-Häkchen sind vom April/Juni |
| `docs/BACKEND-LAUNCH-PLAN.md` | Wellen-Playbook, §4 Wellen 4–7 = Finance-Block | Wellen ohne ✅ sind der eigentliche Prüfauftrag |
| `docs/MODULES_SCOPE_MATRIX.md` | Soll-Scope je Modul | Stand 2026-05-10 |
| `docs/PRICING.md` · `docs/BUSINESS-ROADMAP.md` | Preismodell, Wettbewerb, Blocker | BUSINESS-ROADMAP ist SUPERSEDED — als historische Absicht lesen |
| `CLAUDE.md` · `MEMORY.md` | Architekturregeln, Präferenzen, Fallen | MEMORY ist Stand der letzten Session |
| `.knowledge/` | technischer Vault (`security.md`, `datenbank.md`, `integrationen.md`, `testing.md`) | via `mcp__knowledge__read_text_file` |
| Live-Repo + Produktion | die einzige harte Wahrheit | `app.zentria.tech/health`, `psql -U kmuhub -d kmuhub` über SSH |

Der offene Branch `chore/electron-43` (Electron 33 → 43) ist noch nicht auf `main` — prüf, ob das
für den Launch relevant ist.

**Frag nach, wo du unsicher bist. Lieber over- als under-planen — aber nicht über die
Launch-Definition hinweg planen.**
