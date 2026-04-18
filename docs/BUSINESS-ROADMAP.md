# Cosmi by Zentria — Business-Roadmap 2026

> ⚠️ **SUPERSEDED** am 2026-04-18 durch `docs/ROADMAP.md` (Post-Rigorosum-Konsolidierung).
>
> Launch-Datum, Team-Status (Moritz "in der Schwebe"), Modul-Scope (alle 14 werden echt) sind in der neuen Kern-Roadmap aktualisiert.
> Finanzprojektionen und Pricing-Modell wandern in `docs/PRICING.md`.
> Dieses File bleibt fuer Audit-Trail erhalten, darf aber nicht mehr als Referenz genutzt werden.

---

> Dieses Dokument gibt einen Überblick über Produkt, Team, Timeline, Finanzen und nächste Schritte für die Markteinführung von Cosmi.
>
> **Stand:** 31. März 2026 | **Version:** 1.0

---

## 1. Executive Summary

**Cosmi** ist eine All-in-One Business-Plattform für KMUs im DACH-Raum (5-200 Mitarbeiter). Die Software vereint CRM, Chat, Video, Projektmanagement, Finanzen, HR und Kalender in einer einzigen Lösung — EU-gehostet, DSGVO by Design, ohne Vendor Lock-In.

**Unser Ansatz:** Statt generischer SaaS-Software bieten wir maßgeschneiderte Konfiguration durch eine 1-Woche-Onsite-Prozessanalyse beim Kunden. Kein anderer CRM-Anbieter im DACH-Markt macht das.

**Firma:** Zentria UG (Eintragung 01.05.2026)
**Website:** zentria.tech | **App:** app.zentria.tech

### Team

| Person | Rolle |
|--------|-------|
| Luke | Development — Backend, Infrastruktur, Full-Stack |
| Darien | UI/UX Design + CFO |
| Nico | QA/Testing + Kundenkontakt |
| Moritz | Marketing & Markteinführung |

### Produkt-Status

- **20 Feature-Phasen** fertig entwickelt (CRM, Chat, Video, Kalender, PM, Finanzen, HR, Workflows, Integrationen, Plugin-System)
- **Beta-Hardening** läuft (8 von ~10 Phasen abgeschlossen)
- **Production-Server** live auf Hetzner (EU, Nürnberg) mit HTTPS, automatischen Backups, Deployment-Pipeline
- **Website** live unter zentria.tech

---

## 2. Timeline & Meilensteine

| Phase | Zeitraum | Fokus | Ergebnis |
|-------|----------|-------|----------|
| **UG-Gründung** | 01.05.2026 | Eintragung, Geschäftskonto, Steuernummer | Rechtsfähigkeit, Rechnungen schreiben |
| **Beta-Launch** | Mai–Juni 2026 | Kostenlose Piloten mit 3 Interessenten, Feedback sammeln | Validierung, Referenzen, Bug-Fixes |
| **Legal** | Mai–Juli 2026 | AVV/DPA, AGB, Impressum, DSGVO-Dokumentation | Rechtssicher für echte Kundendaten |
| **Marketing-Aufbau** | Juni–Aug 2026 | Website-Content, LinkedIn-Präsenz, Case Studies aus Pilot | Erste Sichtbarkeit, Lead-Generierung |
| **Erster Umsatz** | Q3 2026 (Aug–Sep) | Pilot-Kunden konvertieren, erste Neukunden gewinnen | Wiederkehrender Umsatz (MRR) |
| **Skalierung** | Q4 2026+ | Onsite-Analysen, Branchenpakete ausrollen, Vertrieb aufbauen | Wachstum Richtung 20+ Kunden |

### Kritischer Pfad

```
Mai 2026          Juni              Juli              Aug               Sep
|--- UG-Gründung ---|
|-------- Beta-Piloten (3 Unternehmen, kostenlos) --------|
|-------------- Legal (AVV/DPA, AGB) --------------------|
                 |-------- Marketing-Aufbau -------------------------|
                                                  |--- Erster Umsatz ---|
```

---

## 3. Revenue-Projektionen

### Fixkosten (monatlich)

| Posten | Kosten/Monat |
|--------|-------------|
| Hetzner Server (CPX42) | ~20 EUR |
| Domain (zentria.tech) | ~2 EUR |
| Tools & Services | ~30 EUR |
| **Gesamt** | **~50 EUR** |

> Keine Gehälter in der Anfangsphase — alle arbeiten auf Equity-Basis. Break-Even bei erstem zahlenden Kunden.

### Jahr 1 — Konservatives Szenario (~196.000 EUR)

| Einnahmequelle | Kalkulation | Betrag |
|---------------|-------------|--------|
| **SaaS (20 Unternehmen)** | Mix aus Branchen, durchschnittlich ~270 EUR/Mo | ~64.700 EUR |
| **Orbit Self-Hosted (2 Lizenzen)** | 2x Orbit Pod (4.000 EUR/Jahr) | 8.000 EUR |
| **Kauflizenzen (3 Stück)** | 3x Orbit Station (15.000 EUR) | 45.000 EUR |
| **Dienstleistungen** | 10 Onsite-Analysen, Migrationen, Configs | ~78.000 EUR |
| | **Gesamt Jahr 1** | **~196.000 EUR** |

### Jahr 1 — Optimistisches Szenario (~335.000 EUR)

| Einnahmequelle | Kalkulation | Betrag |
|---------------|-------------|--------|
| **SaaS (32 Unternehmen)** | Breiterer Mix | ~105.000 EUR |
| **Orbit Self-Hosted (4 Lizenzen)** | 3x Station + 1x Command | 33.000 EUR |
| **Kauflizenzen (5 Stück)** | 5x Orbit Station | 75.000 EUR |
| **Dienstleistungen** | 15 Onsite + Migrationen + Custom Dev | ~122.000 EUR |
| | **Gesamt Jahr 1** | **~335.000 EUR** |

### Jahr 2–3 Ausblick

- **Jahr 2:** 350.000–500.000 EUR (Wartungsverträge + Bestandskunden-Wachstum)
- **Jahr 3:** 500.000–800.000 EUR (Skalierung, ggf. erste Angestellte)

### Umsatz-Mix-Regel

Maximal 30% des Gesamtumsatzes aus Kauflizenzen — der Rest muss wiederkehrend (SaaS/Orbit-Abo) oder Dienstleistung sein. Das schützt vor Einmaleffekten.

---

## 4. Pricing-Übersicht

### SaaS (Cloud, EU-gehostet)

| Tier | Preis/Monat | Jahresabo | Zielgruppe | Zugang |
|------|------------|-----------|------------|--------|
| **Light** | 9 EUR | 7 EUR | Außendienst, Azubis, Teilzeit | Chat, Kalender, Video, Dokumente (lesen) |
| **Standard** | 29 EUR | 23 EUR | Sachbearbeitung, Berater, Vertrieb | CRM, Chat, Kalender, PM, E-Mail, Dokumente |
| **Full** | 49 EUR | 39 EUR | Geschäftsführung, Projektleiter | Alles inkl. Finanzen, HR, Automation, API |
| **Guest** | Kostenlos | — | Kunden, Lieferanten, Externe | Chat via Token-Link (30 Msg/Min) |

**Vorteil gegenüber Wettbewerb:** Rollenbasiert statt Einheitspreis — ein Handwerksbetrieb mit 1 Meister + 2 Büro + 8 Technikern zahlt **179 EUR/Mo** statt 429 EUR bei Per-Seat-Modellen.

### Orbit Self-Hosted (Jahresabo)

| Tier | Preis/Jahr | User-Limits | Support |
|------|-----------|-------------|---------|
| **Orbit Pod** | 4.000 EUR | 10 Full + 20 Standard + unbegrenzt Light | Community |
| **Orbit Station** | 7.000 EUR | 25 Full + 50 Standard + unbegrenzt Light | Priority |
| **Orbit Command** | 12.000 EUR | Unbegrenzt | SLA + 5h Custom Dev |

### Kauflizenz (Einmalzahlung, Self-Hosted)

| Paket | Einmalpreis | Support inkl. | Updates inkl. |
|-------|-----------|--------------|--------------|
| **Orbit Pod** | 8.000 EUR | 1 Jahr | 1 Jahr |
| **Orbit Station** | 15.000 EUR | 2 Jahre | 2 Jahre |
| **Orbit Command** | 28.000 EUR | 4 Jahre | 4 Jahre |

Nach Ablauf: Optionaler Wartungsvertrag (20% des Kaufpreises/Jahr) oder Software läuft weiter ohne Updates.

### Dienstleistungen

| Service | Preis |
|---------|-------|
| **Onsite-Prozessanalyse (1 Woche)** | 5.000–8.000 EUR |
| Setup & Konfiguration (Self-Hosted) | 1.500 EUR |
| Datenmigration | 1.000–3.000 EUR |
| Custom WASM Plugins | 2.000–10.000 EUR |
| Remote-Konfiguration | 500–1.500 EUR |
| Custom Development | 150 EUR/Stunde |

### Branchenpakete — Beispielrechnungen

| Branche | Typisches Team | Monatlich (SaaS) |
|---------|---------------|-----------------|
| **Handwerk** | 1 Meister + 2 Büro + 8 Techniker | 179 EUR |
| **IT/Agenturen** | 2 Management + 8 Devs + 2 PM + 3 Freelancer | 415 EUR |
| **Beratung** | 3 Partner + 5 Berater + 2 Admin | 350 EUR |
| **Handel** | 1 Inhaber + 1 Einkauf + 3 Vertrieb + 2 Lager | 183 EUR |

---

## 5. USPs für Vertriebsgespräche

1. **Rollenbasiertes Pricing** — Nicht jeder zahlt gleich viel. Spart KMUs 40-60% gegenüber Per-Seat-Modellen
2. **Kaufoption** — Einzigartig im DACH-CRM-Markt. Einmal zahlen, ewig nutzen
3. **Onsite-Prozessanalyse** — Kein anderer CRM-Anbieter kommt 1 Woche vorbei und konfiguriert alles maßgeschneidert
4. **EU-Datensouveränität** — Hetzner Deutschland, DSGVO by Design, kein US Cloud Act
5. **All-in-One** — CRM + Chat + Video + PM + Finanzen + HR statt 6 separate Tools
6. **Kein Vendor Lock-In** — Self-Hosted möglich, Datenexport jederzeit, JetBrains-Fallback-Modell
7. **Branchenpakete** — Vorkonfiguriert für 10+ Branchen, sofort einsatzbereit

---

## 6. Blocking Points & Risiken

### Kritische Blocker

| Blocker | Status | Auswirkung | Lösung |
|---------|--------|-----------|---------|
| **AVV/DPA** (Auftragsverarbeitungsvertrag) | Offen — DIY + Anwalt | Ohne AVV keine echten Kundendaten verarbeiten | Bis Juni/Juli 2026 abschließen |
| **AGB & Impressum** | Offen | Pflicht für B2B-Geschäft in DE | Parallel zum AVV |
| **UG-Eintragung** | Geplant 01.05.2026 | Ohne UG keine Rechnungen, kein Geschäftskonto | Notar-Termin sichern |

### Risiken

| Risiko | Wahrscheinlichkeit | Impact | Mitigation |
|--------|-------------------|--------|------------|
| Pilot-Kunden springen ab | Mittel | Hoch — keine Referenzen, kein Feedback | Engen Kontakt halten, schnell auf Feedback reagieren |
| Legal dauert länger als geplant | Mittel | Hoch — verschiebt Umsatz-Start | Früh anfangen, Anwalt parallel zur Beta |
| Bootstrapped-Budget reicht nicht | Niedrig | Mittel — Fixkosten sind minimal (~50 EUR/Mo) | Lean bleiben, ggf. Fördermittel (EXIST, KfW) |
| Kapazitätsengpass (4-Personen-Team) | Hoch | Mittel — langsamer Rollout | Max 2-3 Kauflizenz-Kunden/Quartal, Prioritäten setzen |
| Wettbewerber (HubSpot, Pipedrive) | Dauerhaft | Mittel | Differenzierung über Onsite + EU + Kaufoption |

---

## 7. Marketing & GTM — Erste Schritte für Moritz

### Phase 1: Grundlagen (Mai–Juni 2026)

- [ ] **Website-Content ausbauen** — Pricing-Seite, Feature-Details, Branchenpakete, FAQ
- [ ] **LinkedIn-Präsenz** — Unternehmensseite + persönliche Profile des Teams
- [ ] **Pitch Deck** erstellen (basierend auf diesem Dokument)
- [ ] **Vergleichsseiten** vorbereiten (Cosmi vs. HubSpot, vs. Pipedrive, vs. Salesforce)

### Phase 2: Sichtbarkeit (Juni–Aug 2026)

- [ ] **Content-Marketing** — Blog-Posts zu EU-Datensouveränität, DSGVO-Probleme mit US-Tools, KMU-Digitalisierung
- [ ] **Case Studies** aus Pilot-Kunden (mit Nico abstimmen)
- [ ] **LinkedIn-Kampagnen** — organisch + ggf. Sponsored Posts (Budget TBD)
- [ ] **Branchenverzeichnisse** und Vergleichsportale (OMR Reviews, Capterra, etc.)

### Phase 3: Lead-Generierung (Aug 2026+)

- [ ] **Landing Pages** für Branchenpakete
- [ ] **Webinare/Demos** — "Cosmi für Handwerker", "Cosmi für Agenturen"
- [ ] **Partnerschaften** — Steuerberater, IT-Dienstleister, Unternehmensberater als Multiplikatoren
- [ ] **Referral-Programm** für bestehende Kunden

### Wer liefert was an Moritz?

| Von | Was | Wann |
|-----|-----|------|
| **Luke** | Produkt-Demos, Feature-Erklärungen, technische USP-Details | Laufend |
| **Darien** | Design-Assets, Screenshots, Brand-Guidelines, Social-Media-Grafiken | Auf Anfrage |
| **Nico** | Kundenfeedback, Pilot-Erfahrungen, Testimonials | Ab Beta-Start |

---

## 8. Wettbewerbslandschaft

| Wettbewerber | Preis/User/Mo | Cosmi-Vorteil |
|-------------|--------------|---------------|
| Salesforce | 25–300 EUR | Deutlich günstiger, rollenbasiert, EU-gehostet |
| HubSpot | 0–130 EUR | Kaufoption, Onsite-Analyse, Self-Hosted |
| Pipedrive | 15–99 EUR | All-in-One statt nur CRM, Branchenpakete |
| Odoo | 31 EUR | Bessere Self-Hosted-Option, EU-souverän |
| weclapp | 39–69 EUR | Plugin-System (WASM), flexibleres Pricing |

**Kernargument:** Die meisten KMUs nutzen 4-6 separate Tools (CRM + Slack + Zoom + Trello + Lexware + ...) für zusammen 50-100 EUR/User/Monat. Cosmi ersetzt das alles für 9-49 EUR/User/Monat.

---

## Anhang: Regulatorische Rückenwinde

- EU testet Matrix/Element als Teams-Ersatz
- Deutsche Datenschutzbehörden warnen: MS Teams nicht DSGVO-konform
- Digitale Souveränität wird Geschäftsanforderung, nicht nur Ideologie
- Durchsetzung wird strenger — Bußgelder steigen

**Das bedeutet für uns:** Jedes Gespräch mit einem KMU-Geschäftsführer kann mit "Wissen Sie, dass Ihr aktuelles Setup möglicherweise nicht DSGVO-konform ist?" starten. Cosmi ist die Lösung.

---

*Dieses Dokument wird laufend aktualisiert. Fragen und Feedback an das Team.*
