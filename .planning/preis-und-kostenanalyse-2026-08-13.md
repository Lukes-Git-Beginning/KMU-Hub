# Preis- und Kostenanalyse Cosmi 1.0

> Erstellt 2026-08-13 für die Gruppenbesprechung. Grundlage: Modul-Streichliste (Darien,
> 2026-08-13), `launch-lagebild-2026-08-12.md`, `auslieferungsmodell-2026-08-12.md`, der Code
> selbst, und eine Marktrecherche mit Preisständen August 2026.
>
> **Was dieses Dokument ist:** eine Rechengrundlage, kein Beschluss. Jede Zahl ist entweder
> gemessen (dann steht die Quelle dabei), recherchiert (dann steht die Fundstelle in §12) oder
> geschätzt (dann steht **[S]** dabei). Wo eine Annahme das Ergebnis kippen kann, steht sie
> ausdrücklich als Annahme da.
>
> **Die drei Sätze, auf die alles hinausläuft:**
> 1. Die Serverkosten pro Kunde sind mit **~36 €/Monat** so klein, dass sie für die
>    Preisgestaltung fast egal sind. Der echte Kostentreiber ist **Betreuungszeit**.
> 2. Die Streichliste halbiert den verkaufbaren Katalog — die alte Preisliste ergibt **47 € statt
>    97 €** pro Vollausstattung. Ohne Neukalibrierung sinkt der Umsatz pro Kunde um die Hälfte.
> 3. Der alte Claim „50–90 % günstiger als der Tool-Stack" ist mit der 1.0-Palette **nicht mehr
>    haltbar**. Nachgerechnet sind es **~27 %**. Der Verkauf muss über Souveränität, Anpassbarkeit
>    und „ein System statt sechs" laufen — nicht primär über den Preis.

---

## 1 · Der Modul-Zuschnitt 1.0

### 1.1 Was gestrichen ist

Elf Module, nach dem Piloten fertig zu bauen:

**Meetings · Buchhaltung (finanzen) · Inventar · Einkauf · Fuhrpark · Produktion · Berichte ·
Formulare · Vermietung · Rapporte · Dialer (Telefonie)**

Der Zuschnitt deckt sich exakt mit elf der Nav-Einträge in
`desktop/src/renderer/src/components/layout/sidebar/nav-items.ts` — es sind saubere Schnitte, jedes
gestrichene Modul hat einen eigenen Backend-Dienst in `deploy/docker/docker-compose.yml` und lässt
sich damit tatsächlich weglassen (nicht nur abschalten). Siehe `auslieferungsmodell-2026-08-12.md`
§3.4.

**Drei Nebeneffekte, die für die Preisliste zählen:**

| Effekt | Bedeutung |
|---|---|
| Drei G0-Blocker lösen sich auf | G0-7 (Helpdesk-Zuweisung) bleibt, aber G0-8 (E-Rechnung) und G0-9 (Fuhrpark-i18n) fallen mit ihren Modulen weg |
| `livekit` + `livekit-egress` + `biz` + 9 Modul-Dienste fallen vom Server | **12 von 36 Containern weniger** → kleinerer Server pro Kunde möglich |
| Der Preiskatalog halbiert sich | siehe §1.3 — das ist die teuerste Folge |

### 1.2 Was in 1.0 verkauft wird

Aus 30 Nav-Einträgen minus 4 Plattform-Einträge (`admin`, `dashboard`, `notifications`, `settings`)
minus 11 gestrichene = **14 verkaufbare Module** plus ein Plattform-Alleinstellungsmerkmal:

| # | Modul | Backend-Dienst | Reifegrad (Status-Snapshot 12.08.) |
|---|---|---|---|
| 1 | CRM & Kontakte | `crm` (81 RPCs) | voll, ungegatet — **aber G0-6 offen** (9 Felder gehen beim Speichern verloren) |
| 2 | Projekte | `work` | voll |
| 3 | Aufgaben | `work` | voll |
| 4 | Kalender | `work` | voll — **G0-10 offen** (`booking.zentria.tech` existiert nicht) |
| 5 | Dokumente | `document` + OnlyOffice | voll |
| 6 | Chat | `chat` | voll |
| 7 | E-Mail | `email` | voll — Consent-Prüfung ist toter Code (G1) |
| 8 | Verträge | `vertraege` (15 RPCs) | voll, `DEMO_MODE`-gated |
| 9 | Team & HR | `crm`/`work` | voll, `DEMO_MODE`-gated |
| 10 | Zeiterfassung | `work` (56 RPCs) | voll, **ungegatet** |
| 11 | Schichtplanung | `schichten` (20 RPCs) | voll |
| 12 | Helpdesk | `helpdesk` (38 RPCs) | voll — **G0-7 offen** (Zuweisung scheitert am echten Backend) |
| 13 | Wiki | `wiki` (20 RPCs) | voll |
| 14 | Automatisierung | `automation` | voll |
| — | **Konfigurations-Editor** | Plattform | Laut Lagebild §2 „das stärkste Verkaufsargument, das ihr habt" |

> **Offener Punkt:** Der Nav-Eintrag `infrastructure` (`/infrastruktur`) hat kein Modul-Verzeichnis
> unter `modules/` und keinen eigenen Backend-Dienst. Vor der Preisliste klären, ob das ein Modul,
> ein Stub oder ein Rest ist.

### 1.3 Die teuerste Folge: der Katalog halbiert sich

Nach der bisherigen Preisliste (`docs/PRICING.md`) kostete die Vollausstattung eines Users **97 €**.
Was davon in 1.0 übrig bleibt:

| Gruppe | Alt gesamt | Davon gestrichen | Bleibt |
|---|---:|---:|---:|
| Kern (CRM, Aufgaben, Kalender, Dokumente) | 13 € | 0 € | 13 € |
| Kommunikation | 16 € | 9 € (Meetings 4, Telefonie 5) | 7 € |
| Buchhaltung & Einkauf | 21 € | 16 € (Buchhaltung 6, Einkauf 5, Vermietung 5) | 5 € |
| Team & HR | 10 € | 0 € | 10 € |
| Projekte & Betrieb | 30 € | 20 € (Produktion 7, Inventar 5, Fuhrpark 5, Rapporte 3) | 10 € |
| Tools & Berichte | 7 € | 5 € (Berichte 3, Formulare 2) | 2 € |
| **Summe** | **97 €** | **50 €** | **47 €** |

**Die Streichliste nimmt exakt die Hälfte des Katalogwerts heraus.** Wenn die Modulpreise
unverändert bleiben, halbiert sich der Umsatz pro Kunde — bei nahezu gleichbleibenden Server- und
Betreuungskosten. Das ist der Hauptgrund, warum §7 eine Neukalibrierung vorschlägt.

---

## 2 · Kosten pro Kunde — Infrastruktur

### 2.1 Was ein Cosmi-Server wirklich braucht

**Gemessene Referenz:** Der Produktionsserver trägt **36 Container, davon 30 healthy** — Quelle:
`status-overview.md` §1, `docker ps` über SSH. Er kostet **31 €/Monat** und bedient 15–20 Personen
(§2.2).

> **Die Modellbezeichnung ist unklar.** `MEMORY.md` und das Lagebild führen den Server als
> „CPX42" — dieses Modell kostet heute 69,49 €. Der tatsächliche Rechnungsbetrag von 31 € passt
> nicht dazu. Für diese Analyse ist das folgenlos, weil durchgehend mit dem **Rechnungsbetrag**
> gerechnet wird und nicht mit dem Modell. Für die Frage „was kostet Server Nummer zwei" ist es
> dagegen entscheidend — siehe R4.

Nach der Streichliste fallen weg: `berichte`, `formulare`, `inventar`, `einkauf`, `produktion`,
`fuhrpark`, `vermietung`, `rapporte`, `dialer`, `biz`, `livekit`, `livekit-egress` = **12 Container**.

Bleiben ~20–23: Postgres, Redis, MinIO, OnlyOffice, Gateway, Caddy, 12 Go-Dienste, Monitoring.

**Speicherbedarf, geschätzt [S]** — für 20 Nutzer, keine Messung, sondern Erfahrungswerte:

| Komponente | RAM |
|---|---:|
| PostgreSQL 16 | 1,5–2,0 GB |
| OnlyOffice Document Server | 1,5–2,5 GB ← **größter Einzelposten** |
| 13 Go-Microservices à ~60 MB | ~0,8 GB |
| Gateway | 0,2 GB |
| Redis | 0,25 GB |
| MinIO | 0,3 GB |
| Caddy | 0,05 GB |
| Monitoring (Prometheus/Grafana/Alertmanager) | 0,6 GB |
| OS + Docker-Daemon | 0,7 GB |
| **Summe** | **~6,0–7,4 GB** |

> **Vor der ersten Kundeninstallation messen.** `docker stats` über 24 h auf dem Prod-Server, ohne
> die gestrichenen Dienste. Die Zahl entscheidet, ob ein Kundenserver eine Klasse kleiner ausfallen
> kann als die laufende Maschine — bei 20 Kunden ist jede Klasse rund 240 €/Monat wert.

### 2.2 Der Anker: was der laufende Server tatsächlich kostet

**Ist-Wert, aus der eigenen Hetzner-Rechnung (Darien, 2026-08-13):**

> **31 €/Monat für einen Server, der 15–20 Personen trägt.**

Das ist die belastbarste Zahl dieses Abschnitts, und sie ersetzt jede Hochrechnung aus einer
Preisliste. Die gesamte Kalkulation ab §2.3 baut darauf auf.

**Zur Einordnung — die aktuellen Hetzner-Neupreise (August 2026):**

| Modell | vCPU | RAM | NVMe | €/Monat netto |
|---|---:|---:|---:|---:|
| CX23 | 2 | 4 GB | 40 GB | 5,49 |
| CX33 | 4 | 8 GB | 80 GB | 8,49 |
| CX43 | 8 | 16 GB | 160 GB | 15,99 |
| CX53 | 16 | 32 GB | 320 GB | 29,49 |
| CPX42 | 8 | 16 GB | 320 GB | 69,49 |
| CCX33 *(dediziert)* | 8 | 32 GB | 240 GB | 138,49 |

> **Der eigentliche Befund ist die Spreizung, nicht der Einzelpreis.** Für vergleichbare
> Ausstattung (8 vCPU / 16 GB) reicht die Neupreis-Spanne von **15,99 € bis 69,49 €** — Faktor 4,3,
> je nach Prozessorlinie. Die Preisanpassung vom 15.06.2026 hat die AMD-Linie (CPX) und die
> dedizierten Server (CCX) stark verteuert, die CX-Sparlinie nicht.
>
> **Die Frage, die daraus folgt und vor der ersten Kundeninstallation beantwortet sein muss:**
> Der Ist-Wert von 31 € liegt zwischen diesen Stufen — vermutlich ein Bestandstarif oder ein
> Modell, das heute anders bepreist würde. **Ein zweiter Server, neu gebucht, kostet nicht
> automatisch dieselben 31 €.** Im Hetzner-Konto nachsehen: Welches Modell läuft, und was kostet
> ein baugleicher zweiter davon heute? Fällt die Antwort Richtung 69 €, verdoppelt sich die
> Infrastrukturposition in jeder folgenden Rechnung — bei ARPA 420 € kostet das rund 8 % Marge
> pro Kunde. Fällt sie Richtung 16 €, wird die Rechnung besser.

### 2.3 Vollkosten pro Kundenserver

Vier Größenklassen, verankert am Ist-Wert. Alle Preise netto/Monat.

| Posten | **XS** bis 10 User | **S** 15–25 User | **M** bis 60 User | **L** bis 150 User |
|---|---:|---:|---:|---:|
| Server | ~20 € [S] | **31 €** *(Ist-Wert)* | ~60 € [S] | ~140 € [S] |
| Storage Box (Offsite, verschlüsselt) — **heute noch nicht gebucht** | BX11 **3,20** | BX11 **3,20** | BX21 **10,90** | BX21 **10,90** |
| Anteil zentrale Dienste [S] *(Registry, Monitoring-Zentrale, SMTP)* | 2,00 | 2,00 | 2,00 | 2,00 |
| **Infrastruktur gesamt** | **~25 €** | **~36 €** | **~73 €** | **~153 €** |

Zwei Anmerkungen zur Tabelle:

- **Nur die S-Spalte ist gemessen.** XS, M und L sind vom Ist-Wert hochgerechnet — grob nach
  Nutzerzahl, nicht nach einer Modell-Zuordnung. Sobald feststeht, welches Hetzner-Modell die
  31 € trägt, lassen sich die drei anderen Spalten exakt ersetzen.
- **Das Offsite-Backup ist noch nicht gebucht.** Das Lagebild führt unter G1: *„Backups liegen nur
  lokal, unverschlüsselt, teils world-readable, auf derselben Disk wie die DB (85 % voll)."* Die
  3,20 € stehen hier als **künftiger** Posten — sie fehlen heute auf der Rechnung und in der
  Absicherung gleichermaßen.

**Nicht enthalten und wichtig:**

| Risikoposten | Wirkung |
|---|---|
| **OnlyOffice-Lizenz** | Die Community-Edition ist auf **20 gleichzeitige Dokumentverbindungen** begrenzt. Darüber ist die Enterprise Edition nötig: **ab ~1.470 USD** pro Server. Umgelegt sind das **~110 €/Monat pro Kunde** — mehr als der Server. Trifft ab Klasse M. → **Vor dem ersten M-Kunden klären:** Enterprise-Lizenz einpreisen, Collabora CODE prüfen, oder Dokumente-Modul in großen Installationen anders bepreisen. |
| **Traffic** | 20 TB inklusive. Für ein CRM praktisch nie relevant. Kein Posten. |
| **Volumes** | Erst nötig, wenn Dokumente die Serverplatte sprengen. In der S-Klasse für 25 User in der Regel unkritisch. |
| **Preis eines *neuen* Servers** | Der 31-€-Anker gilt für die laufende Maschine. Was Hetzner für einen zweiten, heute gebuchten Server berechnet, ist ungeprüft — siehe §2.2. |

### 2.4 Der eigentliche Kostentreiber: Betreuungszeit

Die Infrastruktur ist billig. Was pro Kunde wirklich kostet, ist Zeit — und die skaliert nicht mit
der Servergröße, sondern mit der Kundenzahl.

| Tätigkeit | Aufwand pro Kunde | Anmerkung |
|---|---|---|
| Ersteinrichtung Server + Deployment | 2–4 h [S] | sinkt stark mit Registry + Einrichtungs-Erzeuger (3–5 PT einmalig, `auslieferungsmodell` §4) |
| Onsite-Prozessanalyse + Konfiguration | 3–5 PT | der USP — und separat berechnet, siehe §7.4 |
| Datenmigration | 0,5–3 PT | separat berechnet |
| **Laufend: Support, Updates, Monitoring, Incidents** | **1–4 h/Monat** [S] | **die entscheidende Unbekannte** |

**Bewertet mit einem internen Satz von 75 €/h** (konservativ für Entwickler-Zeit im DACH-Markt):

| Betreuung | Kosten/Kunde/Monat |
|---|---:|
| 1 h/Monat (eingespielt, alles automatisiert) | 75 € |
| 2 h/Monat (Basisszenario) | **150 €** |
| 4 h/Monat (Frühphase, ohne Runbook) | 300 € |

> **Das ist die Kernaussage der Kostenseite:** Ein Kunde kostet **36 € Infrastruktur und
> 75–300 € Zeit**. Wer die Preisgestaltung an den Serverkosten ausrichtet, rechnet am Faktor 4
> bis 8 vorbei.
> Und: Das Lagebild nennt für die Betriebsreife (`restore.sh`, Backup-Alarm, Runbook, Secret-Rotation)
> ~10 PT, die bei mehreren Kundenservern **pro Maschine** relevant werden. Solange die nicht
> abgearbeitet sind, liegt der Betreuungsaufwand am oberen Rand.

---

## 3 · Unternehmenskosten — laufend

### 3.1 Monatliche Fixkosten (ohne Gehälter)

| Posten | schlank | realistisch | Anmerkung |
|---|---:|---:|---|
| Haupt-/Demo-Server Hetzner | 31 | 31 | **Ist-Wert aus der laufenden Rechnung**, keine Schätzung |
| Website auf eigenem Hetzner | 0 | 5,49 | nach Vercel-Ausstieg (G0-1); kann auf dem Hauptserver mitlaufen |
| Storage Box eigene Backups | 3,20 | 3,20 | BX11, 1 TB |
| **Claude Max 20× (Seat 1)** | **185** | **185** | 200 USD/Monat |
| Claude Max (Seat 2) | 0 | 185 | falls Darien einen eigenen braucht — heute geteilter Account |
| GitHub | 0 | 11 | Free reicht für private Repos; Team 4 USD/User |
| Domain zentria.tech | 3 | 3 | |
| EU-SMTP (statt Resend/USA) | 7 | 15 | Lagebild §5, Fassung B |
| DE-/EU-Captcha (statt Turnstile) | 9 | 9 | Lagebild §5, Fassung B |
| Laufende Buchhaltung | 100 | 250 | Bandbreite lt. Recherche 100–500 € |
| Jahresabschluss (umgelegt) | 42 | 150 | 500 € Festpreis bis 2.500 € klassisch, /12 |
| IHK-Beitrag (umgelegt) | 5 | 25 | Pflichtmitgliedschaft; Kleinbetrag-Befreiung im 1. Jahr prüfen |
| Betriebshaftpflicht | 10 | 25 | 100–300 €/Jahr |
| **Vermögensschaden-/Cyber-Haftpflicht** | 40 | 125 | 500–1.500 €/Jahr — **bei Auftragsverarbeitung dringend zu empfehlen** |
| Geschäftskonto | 0 | 15 | |
| Puffer / Sonstiges | 30 | 60 | |
| **Summe/Monat** | **~465 €** | **~1.095 €** | |

**Für die weitere Rechnung nehme ich 750 €/Monat** — schlank plus zweiter Claude-Seat plus
realistische Buchhaltung, ohne die dickste Versicherungsvariante.

**Auffällig:** Der größte laufende Einzelposten ist **Claude** (185–370 €/Monat = 25–49 % der
Fixkosten) — mehr als Server, Buchhaltung und Versicherung zusammen. Das ist bei einem
AI-First-Entwicklungsmodell mit einem Entwickler zu erwarten und vermutlich richtig investiert;
es gehört aber bewusst so entschieden, nicht nebenbei.

### 3.2 Einmalkosten bis zum ersten zahlenden Kunden

| Posten | von | bis | Anmerkung |
|---|---:|---:|---|
| UG-Gründung: Notar | 300 | 700 | Lagebild §11, recherchiert |
| Handelsregister-Eintragung | 150 | 150 | 4–8 Wochen bis Eintragung |
| Stammkapital-Einlage | 1.000 | 3.000 | kein Aufwand, aber gebundene Liquidität |
| Rechtsberatung AVV/AGB/DSE/TOMs | 1.500 | 4.000 | G0-5 — „AVV inkludiert" wird beworben, existiert nicht |
| Art.-30-Verzeichnis (Vorlage + Prüfung) | 0 | 500 | G2-Posten |
| Markenanmeldung DPMA (3 Klassen) | 290 | 1.800 | 290 € Amtsgebühr, mit Anwalt deutlich mehr |
| Steuerberater Gründungsberatung | 300 | 800 | |
| **Summe ohne Stammkapital** | **~2.540 €** | **~7.950 €** | |

> **Gegen das genannte Budget von 2.000–5.000 €:** Das reicht für **UG + Legal-Minimum**, wenn die
> Markenanmeldung zurückgestellt und die Rechtsberatung auf AVV/AGB/DSE begrenzt wird. Es reicht
> **nicht** für UG + volle Anwaltspakete + Marke. Priorität aus Risikosicht: **AVV zuerst** (jede
> erste Kundenanfrage verlangt ihn), Marke später.

### 3.3 Personalkosten — vier Szenarien

Das ist die Variable, die alles andere dominiert. AG-Kosten ≈ Bruttogehalt × 1,22.

| Szenario | Besetzung | Personal/Monat | Fixkosten gesamt/Monat |
|---|---|---:|---:|
| **A — Nebenberuflich** *(Status quo)* | 0 Gehälter | 0 € | **750 €** |
| **B — Einer hauptberuflich** | 1 × 4.000 € brutto | ~4.880 € | **~5.630 €** |
| **C — Zwei hauptberuflich** | 2 × 4.000 € brutto | ~9.760 € | **~10.510 €** |
| **D — Drei hauptberuflich** | 3 × 4.000 € brutto | ~14.640 € | **~15.390 €** |

Geschäftsführergehälter in der UG sind steuerlich Betriebsausgaben; die Zahlen gelten unabhängig
davon, ob ausgezahlt oder als Gesellschafter-Geschäftsführerbezug gebucht wird.

---

## 4 · Wettbewerbsvergleich — Preise August 2026

### 4.1 All-in-One-Systeme (der direkte Vergleich)

| Anbieter | Preis/User/Monat | Umfang | Hosting | Anpassbar |
|---|---|---|---|---|
| **Zoho One** (all-employee, jährl.) | **35,31 €** *(42,94 € monatl.)* | 45+ Apps | Indien/EU-Rechenzentren, US-Konzernstruktur | mittel |
| **Zoho One** (flexibel) | 85,88 € *(100,19 € monatl.)* | 45+ Apps | dito | mittel |
| **Odoo Standard** | ~24,90 USD Aktion → **~31,10 USD Verlängerung** ≈ 29 € | alle Apps | Belgien/EU wählbar, Self-Hosting möglich | **hoch** (Studio) |
| **Odoo Custom** | ~49 USD Aktion → ~61 USD ≈ 57 € | alle Apps + Studio | dito | **sehr hoch** |
| **weclapp** ERP Starter | **39 €** | CRM-Basis | DE (Frankfurt) | gering |
| **weclapp** ERP Services | 86 € jährl. / 95 € monatl. | + Aufträge, Rechnungen | DE | gering |
| **weclapp** ERP Trade | 163 € jährl. / 179 € monatl. | + Lager | DE | gering |
| **Teamleader Focus** | 25 / 30 / 40 € (min. 2 User) | CRM + Angebote + Rechnungen | BE/EU | gering |
| **Microsoft 365 Business Standard** | **14,00 €** *(ab 01.07.2026, vorher 12,50 €)* | Office + Exchange + Teams | global/US | gering |
| **Microsoft 365 Business Premium** | 22,00 € | + Security/Intune | global/US | gering |

### 4.2 Fachtools (was Cosmi einzeln ersetzt)

| Kategorie | Anbieter | Preis/User/Monat |
|---|---|---|
| CRM | Pipedrive | 14–99 € |
| CRM | HubSpot Sales Starter | 15–20 €, Professional ~100 € |
| CRM | Salesforce | ab 25 € |
| Projekte | monday.com | ab 9 € |
| Projekte | Asana | ab 11 € |
| Chat | Slack Pro | ~8 € |
| Video | Zoom Pro | 13–15 € |
| Zeiterfassung | Clockify/Harvest | ab 5 € |
| Helpdesk | Zendesk | ab 19 € |
| Wissen | Notion / Confluence | 8 € / 5 € |

### 4.3 Wo Cosmi 1.0 landet

Mit dem Vorschlag aus §7 liegt Cosmi bei **~32 €/User/Monat effektiv** (20-User-Kunde, inkl.
Grundgebühr und Support). Bei kleineren Betrieben steigt der Kopfpreis, weil sich die Grundgebühr
auf weniger Köpfe verteilt: 34 € bei 12 Usern, 48 € bei 5 Usern.

```
  14 €              29–34 €           35 €      39 €          57 €
   │                   │               │         │             │
 M365 Std      ┌── Cosmi 1.0 ──┐   Zoho One   weclapp     Odoo Custom
               │    ~32 €      │              Starter
               └───────────────┘
                 Odoo Standard ~29 €
```

**Ehrliche Einordnung — das gehört in die Runde und nicht ins Verkaufsgespräch geschönt:**

| | Cosmi 1.0 | Zoho One | Odoo |
|---|---|---|---|
| Modulzahl | **14** | 45+ | 80+ |
| Buchhaltung enthalten | **nein** (nach 1.0) | ja | ja |
| Lager/Einkauf/Produktion | **nein** (nach 1.0) | ja | ja |
| Video-Meetings | **nein** (nach 1.0) | ja | ja |
| Hosting DE, kein Drittlandtransfer | **ja, prüfbar** | nein | teilweise |
| **Eigener Server pro Kunde** | **ja** | nein | nur self-hosted |
| Nicht gebuchte Module physisch nicht installiert | **ja** | nein | nein |
| Konfigurations-Editor (7 Dimensionen, ohne Code) | **ja** | teilweise | ja (Studio, im Custom-Tarif) |
| Onsite-Prozessanalyse | **ja** | nein | über Partner, teuer |

> **Cosmi 1.0 ist funktional die kleinste Lösung im Feld und preislich nicht die billigste.**
> Das ist tragbar — aber nur, wenn der Verkauf auf den drei fett markierten Zeilen aufbaut und
> nicht auf Funktionsbreite oder Preis.

### 4.4 Was der Kunde real spart — nachrechenbar

20-Personen-Dienstleister, realistische Sitzplatzverteilung (nicht jeder hat jedes Tool):

| Werkzeug heute | Seats | €/Seat | Summe |
|---|---:|---:|---:|
| CRM (Pipedrive Advanced) | 6 | 29 | 174 € |
| Projekte/Aufgaben (Asana Starter) | 15 | 11 | 165 € |
| Chat (Slack Pro) | 20 | 8 | 160 € |
| Zeiterfassung (Clockify) | 20 | 5 | 100 € |
| Wissensdatenbank (Notion) | 10 | 8 | 80 € |
| Helpdesk (Zendesk Team) | 3 | 55 | 165 € |
| Vertragsverwaltung | — | — | 50 € |
| **Summe Fachtools** | | | **894 €** |
| *(Microsoft 365 Business Standard 20 × 14 = 280 € läuft in beiden Fällen weiter)* | | | *(280 €)* |

**Cosmi 1.0 für denselben Betrieb (§7.3 Beispiel A gerechnet): 649 €/Monat.**

| | Betrag |
|---|---:|
| Ersparnis absolut | **245 €/Monat** |
| Ersparnis relativ | **27 %** |
| Ersparnis pro Jahr | **2.940 €** |

**Der alte Claim „50–90 % günstiger" (`docs/PRICING.md` §11.7) ist damit nicht mehr haltbar** und
muss von der neuen Website verschwinden — er war schon mit dem vollen Katalog optimistisch
gerechnet (alle 20 User bekamen jedes Tool), und mit der 1.0-Palette stimmt er nicht.

**Was stattdessen stimmt und stärker ist:**
> „Ein System statt sechs, auf Ihrem eigenen Server in Deutschland, für rund ein Viertel weniger
> als Ihr heutiger Tool-Stack — und was Sie nicht buchen, ist bei Ihnen nicht installiert."

Das ist prüfbar, und Prüfbarkeit ist nach dem Lagebild genau die Währung, die euch gerade fehlt.

---

## 5 · Preisgestaltung — drei Modelle

### 5.1 Warum das bisherige Modell bei „ein Server pro Kunde" nicht mehr trägt

`docs/PRICING.md` sagt ausdrücklich: *„Keine Grundgebühr oder Plattform-Fee."* Das war richtig,
solange alle Kunden auf einer Multi-Tenant-Instanz liefen. Mit Vorschlag 1 aus dem
Auslieferungsmodell (eigener Server pro Kunde) entsteht ein **Fixkostenblock pro Kunde**, den ein
reines Pro-User-Modell nicht abbildet:

| Kunde | Infrastruktur + Betreuung | Erlös (altes Modell, 26 €/User) | Deckungsbeitrag |
|---|---:|---:|---:|
| 5 User | 25 € + ~150 € Zeit = **175 €** | 130 € | **−45 €** ← Verlust |
| 12 User | 36 € + ~150 € = **186 €** | 312 € | +126 € |
| 25 User | 36 € + ~150 € = **186 €** | 650 € | +464 € |

**Ein 5-User-Kunde ist im alten Modell defizitär**, weil er fast dieselben Fixkosten verursacht wie
ein 25-User-Kunde. Das lässt sich auf zwei Wegen heilen: Grundgebühr oder Mindestbestellwert. Ich
schlage beides vor.

### 5.2 Die drei Modelle

| | **A — Status quo** | **B — Grundgebühr + Module** *(Empfehlung)* | **C — Drei Pakete** |
|---|---|---|---|
| Aufbau | reine Summe Modul × User | Grundgebühr je Installation + Modul × User | Starter / Business / Pro, Festpreis je Größenklasse |
| Kleinkunden | defizitär | tragfähig | tragfähig |
| „Zahl nur was du nutzt"-USP | voll erhalten | **erhalten** | verloren |
| Verkaufskomplexität | hoch (14 Entscheidungen) | hoch, aber erklärbar | niedrig |
| Erklärbarkeit der Grundgebühr | — | **stark**: „Ihr eigener Server, Backup, Updates, Monitoring" | entfällt |
| Passt zu „eigener Server pro Kunde" | nein | **ja** | ja |
| Umbauaufwand Website | 0 | gering | mittel |

**Empfehlung: B.** Die Grundgebühr ist bei einem dedizierten Server nicht nur ökonomisch nötig,
sondern **verkäuflich** — sie ist der sichtbare Preis für genau das Alleinstellungsmerkmal, mit dem
ihr werben wollt. „79 € für Ihren eigenen Server in Nürnberg, inklusive Backup, Updates und
Überwachung" ist ein besseres Argument als eine unsichtbare Umlage.

### 5.3 Modell B im Detail

| Baustein | Vorschlag | Begründung |
|---|---|---|
| **Grundgebühr „Eigener Server"** | **79 €/Monat** | deckt 36 € Infrastruktur + 43 € Betriebsanteil. Steigt der Preis für neu gebuchte Server (§2.2), muss die Grundgebühr mitwandern — 79 € trägt bis etwa 45 € Infrastruktur |
| **Modulpreise** | pro User, siehe §7 | „zahl nur was du nutzt" bleibt |
| **Automatisierung** | **29 €/Monat pro Installation** (nicht pro User) | Make/Zapier rechnen auch pro Account, nicht pro Kopf |
| **Mindestbestellwert** | **199 €/Monat** | verhindert defizitäre Kleinstkunden |
| **Volumen-Rabatt** | unverändert (0/5/10/15/20/25 %) | funktioniert, keine Änderung nötig |
| **Branchenpaket-Rabatt** | unverändert 15 % | funktioniert |
| **Support** | Basis 9 € flat · Professional 10 % (min. 29 €) · Premium 15 % (min. 79 €) | **Premium erst zusagen, wenn eine zweite Person erreichbar ist** — siehe Lagebild §10 |
| **Kein Jahresvertrag** | unverändert | bleibt Differenzierungsmerkmal |

> **Warnung zur Support-Stufe Premium (1 h Reaktion, 24/7):** Nach Lagebild §10 gibt es genau einen
> Eskalationspfad, und der ist Lukes Telefon. Premium darf erst auf die Preisliste, wenn ein
> zweiter Mensch an SSH-Key und Vault-Passwort kommt. Bis dahin: Basis und Professional anbieten,
> Professional mit **„Antwort binnen eines Werktags"** statt „4 Stunden".

---

## 6 · Reifegrad-Kennzeichnung

Das Lagebild führt als G2-Befund: *„alle Module werden ohne Reifegrad-Kennzeichnung zum vollen
Preis verkauft"* (1 PT). Mit der Streichliste ist das lösbar, weil die unfertigen Module ohnehin
draußen sind. Trotzdem sollte die Preisliste zwei Stufen kennen:

| Stufe | Bedeutung | Preis | Module in 1.0 |
|---|---|---|---|
| **Verfügbar** | gegen echtes Backend geprüft, Kernfluss hält | 100 % | alle, sobald Etappe 1 (Lagebild §6) durch ist |
| **Vorschau** | nutzbar, aber noch nicht vollständig geprüft | **50 %, jederzeit kündbar** | Kandidaten: Verträge, Automatisierung, Helpdesk *(bis G0-7 gefixt)* |

Das kostet Umsatz und kauft Vertrauen. Nach der Website-Geschichte im Lagebild ist das der bessere
Tausch — und es macht die Formulierung möglich: *„Was bei uns ‚verfügbar' heißt, haben wir gegen
das echte System geprüft, nicht gegen eine Demo."*

---

## 7 · Modulpreise 1.0 — Vorschlag mit Herleitung

### 7.1 Die Preistabelle

| Modul | alt | **neu** | Marktreferenz | Warum die Änderung |
|---|---:|---:|---|---|
| CRM & Kontakte | 6 | **9** | Pipedrive 14–99, HubSpot 15–20, Salesforce ab 25 | Kernwertträger, 81 RPCs, war deutlich unterbepreist |
| Projekte | 5 | **6** | monday ab 9, Asana ab 11 | leicht angehoben |
| Aufgaben | 3 | **3** | in Projekt-Tools enthalten | unverändert |
| Kalender | 2 | **3** | Calendly 12–15 für die Buchungsseite allein | enthält öffentliche Terminbuchung |
| Dokumente | 2 | **6** | Notion 8–12, DMS 10–20 | **stärkste Anhebung**: OnlyOffice self-hosted ist teuer im Betrieb und die beste Souveränitäts-Erzählung |
| Chat | 4 | **4** | Slack Pro ~8 | unverändert |
| E-Mail | 3 | **3** | in M365 enthalten | unverändert |
| Verträge | 5 | **5** | Fachtools 10–25 | unverändert |
| Team & HR | 3 | **4** | Personio ab 6–12 | leicht angehoben |
| Zeiterfassung | 3 | **4** | Clockify 5–9, Papershift 4–8 | Aufzeichnungspflicht nach BAG-Beschluss 2022 erhöht den Wert |
| Schichtplanung | 4 | **4** | Papershift/Shyftplan 5–10 | unverändert |
| Helpdesk | 5 | **6** | Zendesk 19–55, Freshdesk 15–49 | 38 RPCs, deutlich unterbepreist |
| Wiki | 2 | **3** | Confluence ~5, Notion 8 | leicht angehoben |
| **Summe pro User (alle Module)** | **47** | **60** | | |
| Automatisierung | — | **29 €/Installation** | Make/Zapier 20–70 pro Account | neu; nicht pro User |

**Netto-Effekt:** Der Katalog liegt bei 60 statt 47 € — aber immer noch **38 % unter den alten
97 €**. Die Anhebung gleicht die Streichliste nur teilweise aus, und das ist richtig so: Ihr
verkauft weniger, also nehmt ihr weniger ein. Ihr verkauft aber jetzt einen eigenen Server dazu,
und den bezahlt die Grundgebühr.

**Gegenargument, das ich nicht verschweige:** Preise anheben und gleichzeitig Module streichen sieht
von außen schlecht aus. Das trägt hier nur, weil **noch nie jemand einen dieser Preise bezahlt hat**
und die Website ohnehin neu geschrieben wird (G0-1 bis G0-5). Ab dem ersten zahlenden Kunden ist
dieses Fenster zu.

### 7.2 Rollenvorlagen (das, was der Kunde tatsächlich auswählt)

Der Kunde soll nicht 14 Häkchen setzen, sondern drei Rollen vergeben:

| Rolle | Module | €/User |
|---|---|---:|
| **Basis** (Außendienst, Werkstatt, Aushilfe) | Kalender 3, Aufgaben 3, Chat 4, Zeiterfassung 4 | **14 €** |
| **Büro** (Sachbearbeitung, Assistenz) | + CRM 9, Dokumente 6, E-Mail 3 | **32 €** |
| **Leitung** (GF, Team-, Projektleitung) | alle 13 User-Module | **60 €** |

### 7.3 Beispielrechnungen

**A · Dienstleister, 20 Mitarbeiter** *(Referenzfall für §4.4)*

| Posten | Anzahl | Einzel | Summe |
|---|---:|---:|---:|
| Leitung | 3 | 60 € | 180 € |
| Büro | 8 | 32 € | 256 € |
| Basis | 9 | 14 € | 126 € |
| Automatisierung (Installation) | 1 | 29 € | 29 € |
| *Zwischensumme Module* | | | *591 €* |
| Volumen-Rabatt 20 User (5 %) | | | −30 € |
| Grundgebühr eigener Server | | | 79 € |
| Support Basis | | | 9 € |
| **Gesamt/Monat netto** | | | **649 €** |
| *= pro Mitarbeiter* | | | *32 €* |

**B · Handwerksbetrieb, 12 Mitarbeiter** *(realistischer erster Kunde)*

| Posten | Anzahl | Einzel | Summe |
|---|---:|---:|---:|
| Leitung | 2 | 60 € | 120 € |
| Büro | 3 | 32 € | 96 € |
| Basis (+ Schichtplanung 4 €) | 7 | 18 € | 126 € |
| *Zwischensumme Module* | | | *342 €* |
| Volumen-Rabatt 12 User (5 %) | | | −17 € |
| Grundgebühr | | | 79 € |
| Support Basis | | | 9 € |
| **Gesamt/Monat netto** | | | **413 €** |
| *= pro Mitarbeiter* | | | *34 €* |

**C · Kleinbüro, 5 Mitarbeiter** *(Grenzfall)*

| Posten | | Summe |
|---|---|---:|
| 1 × Leitung + 2 × Büro + 2 × Basis | | 152 € |
| Grundgebühr + Support Basis | | 88 € |
| Zwischensumme | | 240 € |
| **Mindestbestellwert greift nicht** (240 > 199) | | **240 €** |
| *= pro Mitarbeiter* | | *48 €* |

> Bei 5 Usern liegt der Preis pro Kopf bei 48 € — über Zoho One. **Kunden unter ~8 Mitarbeitern
> sind mit einem dedizierten Server strukturell zu teuer.** Entweder man nimmt sie bewusst nicht,
> oder es braucht später doch eine geteilte Instanz für die Kleinen. Das gehört in die Runde.

### 7.4 Einmalerlöse

| Leistung | alt | **neu** | Begründung |
|---|---|---|---|
| Onsite-Prozessanalyse (1 Woche) | 5.000–8.000 € | **2.500–5.000 €** | ohne Referenzen und ohne Rechtsträger schwer durchsetzbar; nach den ersten 3 Kunden anheben |
| Datenmigration | 1.000–3.000 € | **750–2.500 €** | je Quellsystem |
| Remote-Konfiguration (statt Onsite) | 500–1.500 € | **500–1.500 €** | unverändert |
| Custom-Plugins (WASM) | 2.000–10.000 € | **von der Preisliste nehmen** | Feature-Flag ist OFF, Phase D — nicht bewerben, was nicht läuft (dieselbe Lehre wie G0-3) |

> **Die Einmalerlöse sind in den ersten zwei Jahren der größere Posten.** Ein Onboarding zu 4.000 €
> entspricht **zehn Monatsabos** eines Durchschnittskunden. Siehe §8.4.

---

## 8 · Gewinnrechnung je Kundenzahl

### 8.1 Annahmen

| Größe | Wert | Herkunft |
|---|---:|---|
| ARPA (Ø Erlös pro Kunde/Monat) | **420 €** | §7.3 liefert 240 € (5 User) · 413 € (12 User) · 649 € (20 User); gewichtet Richtung kleinerer Betriebe, also bewusst konservativ |
| Infrastruktur pro Kunde | **36 €** | §2.3, Klasse S — verankert am Ist-Wert 31 € |
| Betreuung pro Kunde | **2 h/Monat = 150 €** | §2.4, Basisszenario |
| **Deckungsbeitrag I** (nur Infrastruktur) | **384 €** | |
| **Deckungsbeitrag II** (inkl. Betreuung) | **234 €** | die ehrlichere Zahl |
| Fixkosten Szenario A / B / C / D | 750 / 5.630 / 10.510 / 15.390 € | §3.3 |

### 8.2 Break-Even

| Szenario | Break-Even nach DB I | **Break-Even nach DB II** |
|---|---:|---:|
| **A — nebenberuflich** | 2 Kunden | **4 Kunden** |
| **B — einer hauptberuflich** | 15 Kunden | **25 Kunden** |
| **C — zwei hauptberuflich** | 28 Kunden | **45 Kunden** |
| **D — drei hauptberuflich** | 41 Kunden | **66 Kunden** |

### 8.3 Ergebnis je Kundenzahl (DB II, Betreuung eingerechnet)

| Kunden | Umsatz Abo/Mo | Infra | Betreuung (h) | Betreuung (€) | **A** | **B** | **C** | **D** |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 5 | 2.100 € | 180 € | 10 h | 750 € | **+420** | −4.460 | −9.340 | −14.220 |
| 10 | 4.200 € | 360 € | 20 h | 1.500 € | **+1.590** | −3.290 | −8.170 | −13.050 |
| 20 | 8.400 € | 720 € | 40 h | 3.000 € | **+3.930** | **−950** | −5.830 | −10.710 |
| 25 | 10.500 € | 900 € | 50 h | 3.750 € | +5.100 | **+220** | −4.660 | −9.540 |
| 30 | 12.600 € | 1.080 € | 60 h | 4.500 € | +6.270 | **+1.390** | −3.490 | −8.370 |
| 50 | 21.000 € | 1.800 € | 100 h | 7.500 € | +10.950 | +6.070 | **+1.190** | −3.690 |
| 75 | 31.500 € | 2.700 € | 150 h | 11.250 € | +16.800 | +11.920 | **+7.040** | +2.160 |
| 100 | 42.000 € | 3.600 € | 200 h | 15.000 € | +22.650 | +17.770 | +12.890 | **+8.010** |

*Alle Werte €/Monat, netto, vor Steuern. Fett = das jeweils passende Szenario für diese Kundenzahl.*

**Umsatzrendite bei passender Besetzung:**

| Kunden | Szenario | Umsatz/Mo | Ergebnis/Mo | Marge |
|---:|---|---:|---:|---:|
| 10 | A | 4.200 € | 1.590 € | **38 %** |
| 25 | B | 10.500 € | 220 € | **2 %** |
| 50 | C | 21.000 € | 1.190 € | **6 %** |
| 100 | D | 42.000 € | 8.010 € | **19 %** |

### 8.4 Was diese Tabelle wirklich sagt

**1 · Die Betreuungsstunden sind die Wachstumsgrenze, nicht das Geld.** Bei 100 Kunden sind
200 h/Monat Support fällig — **1,25 Vollzeitstellen, die nur Support machen**. Szenario D hat drei
Köpfe, von denen dann einer nichts anderes tut. Wenn die Betreuung auf 4 h/Kunde steigt (heute
realistisch, solange `restore.sh` kaputt ist und kein Runbook existiert), sind es 400 h = 2,5
Vollzeitstellen — und die Rechnung kippt.

> **Die Betreuungszeit pro Kunde ist der Hebel, der über alles entscheidet.** Jede Stunde, die man
> pro Kunde und Monat einspart, ist bei 50 Kunden 3.750 €/Monat wert. Die ~10 PT G1-Betriebsreife
> aus dem Lagebild (Restore, Backup-Alarm, Runbook, Health-Check) amortisieren sich damit **ab
> etwa 15 Kunden innerhalb eines Monats.** Das ist das rentabelste Stück Arbeit im ganzen Plan.

**2 · Szenario B ist das schwierigste, nicht D.** Bei 25 Kunden und einem Vollzeitgehalt liegt die
Marge bei **2 %** — praktisch bei null. Das ist der Engpass: zu viele Kunden für nebenbei, zu wenig
Umsatz für ein Gehalt. Diese Zone durchquert man am besten schnell — oder man überspringt sie,
indem der Übergang zu hauptberuflich erst bei ~30 Kunden erfolgt.

**3 · Die Einmalerlöse tragen die Frühphase.** Bei einem Neukunden pro Monat:

| | Abo | Onboarding | gesamt |
|---|---:|---:|---:|
| Monat 12 (12 Kunden, 1 Onboarding/Monat) | 5.040 € | 4.000 € | **9.040 €** |
| Monat 24 (24 Kunden) | 10.080 € | 4.000 € | **14.080 €** |

**Mit einem Onboarding pro Monat trägt Szenario B ab etwa Monat 8** — statt ab Kunde 25 rein über
Abos. Der Cashflow der ersten zwei Jahre kommt aus der Einrichtung, nicht aus dem Abo. Das
verändert die Vertriebspriorität: **Abschlüsse sind wichtiger als Bestandsausbau.**

**4 · Der Pilot ist unentgeltlich — er kostet trotzdem.** Kunde 1 nach Vorschlag 2 des
Auslieferungsmodells: 36 € Infrastruktur plus 3–5 PT Onboarding plus 2–4 h/Monat. Als
Referenzkunde und Härtetest richtig — aber er gehört als **Investition von ~3.000–5.000 €**
verbucht, nicht als „kostet ja nichts".

**5 · Preissensitivität.** Was ±10 % ARPA bei 30 Kunden bewirken:

| ARPA | Ergebnis Szenario B bei 30 Kunden |
|---:|---:|
| 380 € (−10 %) | **+190 €** |
| 420 € (Basis) | +1.390 € |
| 460 € (+10 %) | **+2.590 €** |

**10 % Preisunterschied verändert das Ergebnis um Faktor 13,6.** In dieser Phase ist der Preis der
empfindlichste Hebel überhaupt — noch vor der Kundenzahl. Ein zu billiger Einstieg ist teurer als
ein verlorener Deal. *(Der Hebel ist durch die korrigierten Serverkosten noch schärfer geworden:
Vor der Korrektur lag er bei Faktor 5,6 — jede Erhöhung der Fixkosten pro Kunde verstärkt ihn.)*

---

## 9 · Risiken und offene Rechnungen

| # | Risiko | Wirkung | Was tun |
|---|---|---|---|
| R1 | **OnlyOffice Community nur bis 20 gleichzeitige Verbindungen** | ab Klasse M ~110 €/Monat/Kunde Lizenz — mehr als der Server | Vor dem ersten M-Kunden: Enterprise-Angebot einholen, Collabora CODE als Alternative prüfen, oder Dokumente-Modul ab 25 Usern anders bepreisen |
| R2 | **Betreuungszeit unbekannt** | die gesamte Rechnung in §8 hängt daran | Beim Piloten **Stunden mitschreiben**, ab Kunde 1. Ohne diese Messung ist jede Prognose Kaffeesatz |
| R3 | **Kleinkunden (<8 User) strukturell unrentabel** | 48 €/Kopf, über Zoho One | Mindestbestellwert 199 €, oder Kleine bewusst ablehnen, oder später doch eine geteilte Instanz |
| R4 | **Preis eines *neu gebuchten* Servers ungeprüft** | Der 31-€-Anker gilt für die laufende Maschine. Neupreise für vergleichbare Ausstattung streuen zwischen 16 und 69 €. Am oberen Rand kostet das ~8 % Marge pro Kunde | **Im Hetzner-Konto nachsehen**, bevor die Preisliste steht: welches Modell trägt die 31 €, was kostet ein zweiter davon heute |
| R5 | **Speicherbedarf ist geschätzt** | falls 16 GB nicht reichen, eine Serverklasse höher | `docker stats` 24 h messen, bevor die Preisliste steht |
| R6 | **Kein Abrechnungssystem** | Lagebild G2: 5 PT, ausdrücklich gestrichen | Bleibt gestrichen. Bis ~10 Kunden Rechnungen per Hand — das sind ~2 h/Monat |
| R7 | **Support-Zusage Premium nicht haltbar** | 24/7 bei einem Eskalationspfad = Vertragsbruch bei erstem Vorfall | Premium erst anbieten, wenn eine zweite Person Zugang hat (Lagebild §10, 1,5 PT) |
| R8 | **Preis-Claim auf der Website** | „50–90 % günstiger" ist mit 1.0 falsch und damit derselbe Fehlertyp wie G0-1 bis G0-4 | Auf die nachrechenbare Fassung aus §4.4 umschreiben — **im selben Aufwasch wie die Website-Etappe 0** |
| R9 | **Preisliste vor Rechtsträger** | Preise nennen, ohne dass es die UG gibt → persönliche Haftung der Handelnden | Preisliste vorbereiten, aber Angebote erst nach Notartermin verschicken |

---

## 10 · Was die Runde entscheiden muss

| # | Frage | Vorschlag | Hängt zusammen mit |
|---|---|---|---|
| 1 | **Grundgebühr einführen (Modell B) oder beim reinen Modul×User bleiben?** | **Modell B, 79 €** | Auslieferungsmodell-Frage 2 (Server pro Kunde) |
| 2 | **Modulpreise neu kalibrieren (§7.1) oder alte Preise halten?** | **neu kalibrieren** — das Fenster schließt mit dem ersten zahlenden Kunden | §1.3 |
| 3 | **Mindestbestellwert 199 €/Monat?** | ja | R3 |
| 4 | **Kunden unter 8 Mitarbeitern annehmen?** | erst mal nein — oder nur mit Aufschlag | R3 |
| 5 | **Automatisierung pro Installation (29 €) oder pro User?** | pro Installation | §7.1 |
| 6 | **Reifegrad-Kennzeichnung „Vorschau, 50 %" einführen?** | ja | §6, Lagebild G2 |
| 7 | **Support Premium (24/7) auf die Preisliste?** | **nein**, bis eine zweite Person Zugang hat | R7, Lagebild §10 |
| 8 | **Onsite-Analyse auf 2.500–5.000 € senken?** | ja, mit Anhebung nach den ersten drei Kunden | §7.4 |
| 9 | **WASM-Plugins von der Preisliste nehmen?** | ja — nicht bewerben, was OFF ist | §7.4, G0-3-Lehre |
| 10 | **Wann von Szenario A nach B wechseln?** | erst ab ~30 Kunden oder ab ~10.000 €/Monat wiederkehrend | §8.4 Punkt 2 |
| 11 | **Was kostet der *zweite* Hetzner-Server?** | **vor der Preisliste im Konto nachsehen** — die Neupreis-Spanne für gleiche Ausstattung reicht von 16 bis 69 € | R4, §2.2 |
| 12 | **Betreuungsstunden ab Pilot 1 mitschreiben?** | **ja, verbindlich** | R2 — ohne das ist §8 nicht validierbar |

---

## 11 · Was für Phase 2 auf der Liste steht

Die elf gestrichenen Module sind kein verlorener Umsatz, sondern eine **Preiserhöhung auf Vorrat**.
Mit den Preisen aus §7-Logik hochgerechnet:

| Modul | alt | Phase-2-Preis [S] | Marktreferenz |
|---|---:|---:|---|
| Buchhaltung/Finanzen | 6 | **9** | sevDesk ab 10, Lexware ab 7 — und 121 RPCs im `biz`-Dienst |
| Meetings | 4 | **5** | Zoom Pro 13–15 |
| Telefonie/Dialer | 5 | **6** | VoIP 8–15 |
| Inventar | 5 | **6** | Fachsoftware 8–20 |
| Einkauf | 5 | **5** | Fachsoftware 10–20 |
| Produktion | 7 | **8** | Fachsoftware 15–30 |
| Fuhrpark | 5 | **5** | Fachsoftware 10–20 |
| Vermietung | 5 | **5** | Fachsoftware 20–40 |
| Rapporte | 3 | **3** | |
| Berichte | 3 | **4** | Power BI/Looker ab 10 |
| Formulare | 2 | **2** | Typeform ab 25 |
| **Summe Nachschlag** | 50 | **58** | |

**Vollausbau nach Phase 2: 118 €/User** statt 97 € heute. Ein Bestandskunde, der von 1.0 auf
Vollausbau wächst, verdoppelt seinen Beitrag — **ohne dass ein einziger Neukunde gewonnen werden
muss.** Das ist das stärkste Argument dafür, die Streichliste jetzt zu akzeptieren: sie verschiebt
Umsatz, sie vernichtet ihn nicht.

Reihenfolge nach Marktwert pro Bauaufwand: **Buchhaltung → Meetings → Inventar/Einkauf →
Berichte → Rest.** Buchhaltung hat mit 121 RPCs bereits die meiste Substanz und den höchsten
Einzelpreis — und laut Lagebild auch das höchste Vertrauensschaden-Risiko, gehört also nicht in die
erste Demo, wohl aber in die erste Erweiterung.

---

## 12 · Quellen und Grenzen

**Aus der eigenen Rechnung:** Serverkosten **31 €/Monat für 15–20 Nutzer** (Darien, 2026-08-13).
Das ist der Anker der gesamten Kostenseite und ersetzt die frühere Fassung dieses Dokuments, die
mit 25 € rechnete — hochgerechnet aus einer Preisliste statt aus der Rechnung.

**Aus dem Repo gemessen:** Modulliste aus `nav-items.ts` (30 Einträge) und
`desktop/src/renderer/src/modules/` (34 Verzeichnisse) · Backend-Dienste aus `backend/cmd/` (24) und
`deploy/docker/docker-compose.yml` (36 Container) · Modul-Feature-Flags aus
`backend/internal/featureflag/registry.go` (17) · Reifegrad und Prod-Containerzahl aus
`.planning/status-overview.md` (Stand 2026-08-12) · Blocker und PT-Schätzungen aus
`.planning/launch-lagebild-2026-08-12.md` · Auslieferungsbefunde aus
`.planning/auslieferungsmodell-2026-08-12.md` · alte Preisliste aus `docs/PRICING.md` und
`.knowledge/pricing.md`.

**Recherchiert (August 2026):**
[Hetzner-Preisübersicht Aug 2026](https://costgoat.com/pricing/hetzner) ·
[Hetzner Preisanpassung 15.06.2026](https://docs.hetzner.com/general/infrastructure-and-availability/price-adjustment/) ·
[Hetzner Preiserhöhungen im Detail](https://webhosting.today/2026/06/18/hetzners-price-increases-reached-209-the-30-headline-applied-to-a-different-tier/) ·
[Hetzner Storage Box BX11](https://www.hetzner.com/storage/storage-box/bx11/) ·
[Storage Box BX21](https://www.hetzner.com/storage/storage-box/bx21/) ·
[weclapp Preise 2026](https://qualimero.com/en/blog/weclapp-prices-costs-2026-complete-guide) ·
[weclapp Kosten und Editionen](https://erp-software.org/weclapp-kosten/) ·
[Zoho One Preise 2026](https://zophoria.de/zoho-wissen/zoho-preise/) ·
[Zoho One Tarifdetails](https://www.zoho.com/de/one/pricing/) ·
[Odoo Pricing 2026](https://www.erpresearch.com/pricing/odoo) ·
[Odoo Preis pro Land](https://oec.sh/odoo-pricing) ·
[Teamleader Focus Preise](https://omr.com/en/reviews/product/teamleader-focus/pricing) ·
[Microsoft 365 Preiserhöhung Juli 2026](https://www.serverstart.de/blog/microsoft-365-preiserhoehung-juli-2026) ·
[Microsoft 365 Business Preise](https://www.microsoft.com/de-de/microsoft-365/business/microsoft-365-plans-and-pricing) ·
[Pipedrive Kosten 2026](https://hanseperformance.de/pipedrive-kosten/) ·
[CRM-Vergleich DACH 2026](https://mind-force.de/knowhow/crm-software-im-vergleich/) ·
[Claude Code Preise 2026](https://www.finout.io/blog/claude-code-pricing-2026) ·
[Claude Subscription Plans 2026](https://intuitionlabs.ai/articles/claude-pricing-plans-api-costs) ·
[ONLYOFFICE Editionen-Vergleich](https://www.onlyoffice.com/compare-editions) ·
[ONLYOFFICE Enterprise Preise](https://www.componentsource.com/product/onlyoffice-docs-enterprise-with-plone-connector/prices) ·
[UG Kosten jährlich](https://onlinebilanz.de/ug-kosten-jaehrlich/) ·
[UG Jahresabschluss Kosten](https://onlinebilanz.de/jahresabschluss-ug-kosten/)

**Geschätzt [S] — mit Konsequenz, falls falsch:**

| Schätzung | Wert | Falls falsch |
|---|---|---|
| **Serverpreis der Klassen XS, M, L** | ~20 / ~60 / ~140 € | vom Ist-Wert 31 € hochgerechnet, nicht auf Modelle abgebildet. Nur die S-Spalte in §2.3 ist gemessen |
| **Preis eines neu gebuchten Servers** | = 31 € angenommen | Neupreis-Spanne 16–69 € für gleiche Ausstattung. Am oberen Rand: −8 % Marge je Kunde, Grundgebühr müsste auf ~95 € |
| Was im 31-€-Betrag enthalten ist | reiner Server angenommen | Sind Backup/IPv4 bereits drin, sinkt die Infrastruktur je Kunde auf ~33 € — die Rechnung wird geringfügig besser |
| RAM-Bedarf einer 1.0-Installation | 6,0–7,4 GB | reicht der Server nicht, eine Klasse höher: +~29 €/Kunde |
| Betreuungsaufwand | 2 h/Kunde/Monat | bei 4 h verschiebt sich jeder Break-Even um Faktor ~1,7 |
| ARPA | 420 €/Monat | ±10 % ARPA = Faktor 13,6 aufs Ergebnis bei 30 Kunden (§8.4) |
| Interner Stundensatz | 75 €/h | reine Rechengröße für Opportunitätskosten, keine Auszahlung |
| Anteil zentraler Dienste pro Kunde | 2 €/Monat | vernachlässigbar |
| Phase-2-Preise (§11) | — | reine Fortschreibung, nicht am Markt validiert |

**Nicht geprüft:** Ob `infrastructure` ein Modul oder ein Rest ist · ob die Sidebar die
Aktivierungstabelle liest oder aus `business-profiles.ts` (offen seit `auslieferungsmodell` §7) ·
tatsächliche Konversionsraten und Vertriebszykluslänge (Pipeline ist leer) · Kirchensteuer- und
Gewerbesteuerhebesatz am Sitz der UG · ob Kunden bereit sind, eine Grundgebühr zu akzeptieren —
das ist eine Marktfrage, die nur ein Gespräch beantwortet.

**Ein Widerspruch, den ich nicht aufgelöst habe:** Der Ist-Wert von 31 €/Monat passt auf kein
aktuelles Hetzner-Listenmodell — CX53 liegt bei 29,49 €, CPX41 (Vorgängergeneration) bei 27,25 €,
CPX42 bei 69,49 €. Das Lagebild vom 12.08. rechnete mit 69,49 € und lag damit ebenfalls daneben.
Wahrscheinlich ein Bestandstarif oder eine Vorgängergeneration, möglicherweise inklusive Backup.
**Aufgelöst wird das nur durch einen Blick ins Hetzner-Konto**, und dieser Blick entscheidet
gleichzeitig, was jeder weitere Kundenserver kostet — siehe R4.
