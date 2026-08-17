# Pricing-Modell Cosmi (v3) — Modul-x-User

> Aktualisiert: 2026-08-17 — Modulpreise und Modul-Zuschnitt auf den Stand der Website gezogen.
> Quellen: `.planning/preis-und-kostenanalyse-2026-08-13.md` (Darien, Neukalibrierung §7.1,
> Ersparnisrechnung §4.4, offene Entscheidungen §10) · Website `src/data/modules.ts` (die
> Preise, die ein Interessent tatsächlich sieht) · Ursprung: Darien, COSMI-/ORBIT-Preiskonzept
> April 2026.
> Kanonische Knowledge-Base-Note: `.knowledge/pricing.md`
>
> **Was an diesem Dokument noch nicht stimmt.** §5 (Branchenpakete), §9 (Beispielrechnungen)
> und §10 (Wettbewerbsvergleich) sind auf dem alten 24-Modul-Katalog gerechnet. Ihre
> abgeleiteten Beträge halten mit der 1.0-Palette nicht mehr — die betroffenen Abschnitte
> tragen einen eigenen Hinweis. Neu zusammengesetzt werden können sie erst, wenn die offenen
> Entscheidungen aus §10 der Preisanalyse gefallen sind (79-€-Grundgebühr, Mindestbestellwert,
> ORBIT in der Preisliste). Bis dahin ist dieses Dokument für **Modulpreise und Modulstatus**
> belastbar und für **Paket- und Beispielsummen nicht**.

---

## 1. Grundprinzip

Cosmi verwendet ein **Modul-x-User-Modell ohne feste Tiers**. Der Preis ergibt sich ausschliesslich aus der Summe der einem User zugewiesenen Module.

```
Monatspreis = Summe(Modulpreis x Anzahl User mit Modul) x (1 - Volumen-Rabatt) + Support-Gebuehr
```

**Was es NICHT gibt:**
- Kein fixes Light/Standard/Full-Tier-System
- Keine Grundgebuehr oder Plattform-Fee
- Keine Mindestanzahl an Modulen
- Kein Laufzeit-Rabatt oder Jahresvertrag

> **Offen, und deshalb hier bewusst nicht eingetragen:** Die Preisanalyse empfiehlt eine
> Grundgebühr von 79 €/Monat (§5.3), weil bei „ein Server pro Kunde" reale Fixkosten pro Kunde
> anfallen. Die Entscheidung ist nicht gefallen (§10, Punkt 1) und hängt am Auslieferungsmodell.
> Solange sie offen ist, gilt oben „keine Grundgebühr" — und genau das sagt auch die Website
> (`src/pages/preise.astro`, FAQ). Ebenfalls offen: Mindestbestellwert 199 €/Monat,
> Automatisierung als 29 € pro Installation statt pro User, Reifegrad-Kennzeichnung.

**Was es gibt:**
- Modul-Zuweisungen pro User (frei konfigurierbar)
- Rollenvorlagen als Abkuerzung ("Monteur", "Buero", "Leitung")
- Volumen-Rabatt nach Gesamtzahl aktiver User
- Support-Grundgebuehr (Basis 9 EUR Flat, Professional 10%, Premium 15%)
- Branchenpakete als Onboarding-Vorlage mit 15% Paket-Rabatt
- Kostenlose Gastnutzer (eingeschraenkter Zugang)

---

## 2. Markt-Analyse (DACH, Stand 2026)

### 2.1 CRM-Markt DACH

| Anbieter | Preis/User/Monat | Zielgruppe | Self-Hosted | Besonderheit |
|----------|-----------------|------------|-------------|--------------|
| Salesforce | 25-300 EUR | Enterprise | Nein | Marktfuehrer |
| HubSpot | 0-130 EUR | SMB-Enterprise | Nein | Kostenlose View-Only Seats |
| Pipedrive | 15-99 EUR | SMB | Nein | Sales-fokussiert |
| Monday CRM | 10-24 EUR | SMB | Nein | Hybrid Work-/CRM |
| Zoho CRM | 14-52 EUR | SMB | Nein | Lite User ab 5 USD |
| weclapp | 39-169 EUR | SMB-Mittelstand | Nein | DACH-fokussiert |
| Twenty CRM | Open Source | Dev-Teams | Ja | OSS-Newcomer |

### 2.2 Pricing-Trends

- **Subscription Fatigue:** 80% der SMBs fuehlen sich ueber-abonniert
- **Modulares Pricing:** Waechst schneller als feste Tiers (OpenView Partners)
- **IT-Budget KMU:** 2-4% des Umsatzes, 33-52 EUR/Monat/Mitarbeiter

---

## 3. COSMI Modulpreise

Alle Preise in EUR pro User pro Monat, zzgl. MwSt. Preise und Status stimmen mit
`src/data/modules.ts` im Website-Repo überein — das ist die Fassung, die ein Interessent sieht.

**13 Module sind buchbar, 11 sind in Vorbereitung** (Modul-Zuschnitt, Preisanalyse §1.1). Für die
11 steht hier bewusst kein Preis: sie sind nicht verkäuflich, und ein Preis in einer Tabelle wird
zitiert. Die Website zeigt an ihrer Stelle „in Vorbereitung". Summe der 13 buchbaren Module:
**60 EUR** pro User und Monat, wenn ein einzelner User alles bekommt.

Die früher hier geführte Spalte „Vorteil" mit Angaben wie „bis 88% guenstiger" ist entfernt. Die
Prozentzahlen waren pro Modul nicht nachgerechnet, und die Gesamtaussage, aus der sie stammten,
ist widerlegt (siehe §11). Der Marktvergleich bleibt als Fremdpreis-Angabe stehen — Stand April
2026, seit dem nicht nachgeprüft.

### 3.1 Kern-Module

| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| CRM & Vertrieb | 9 | buchbar | Pipedrive ab 14, HubSpot ab 15 |
| Aufgaben | 3 | buchbar | Asana ab 11, Monday ab 9 |
| Kalender | 3 | buchbar | M365 (8-22) |
| Dokumente | 6 | buchbar | Notion ab 8 |

### 3.2 Kommunikation

| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Chat | 4 | buchbar | Slack Pro 7-8 |
| E-Mail | 3 | buchbar | M365 |
| Meetings | — | in Vorbereitung | Zoom Pro 13-15 |
| Telefonie & Kampagnen | — | in Vorbereitung | VoIP 8-15 |

### 3.3 Finanzen & Einkauf

| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Finanzen | — | in Vorbereitung | sevDesk ab 10, Lexoffice ab 7 |
| Einkauf | — | in Vorbereitung | Spezialsoftware 10-20 |
| Vertraege | 5 | buchbar | Spezialsoftware 10-25 |
| Vermietung | — | in Vorbereitung | Spezialsoftware 20-40 |

### 3.4 Team & HR

| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Team | 4 | buchbar | M365/HR-Tools |
| Schichten | 4 | buchbar | Spezialsoftware 5-10 |
| Zeiterfassung | 4 | buchbar | Clockify/Harvest ab 5 |

### 3.5 Projekte & Betrieb

| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Projekte | 6 | buchbar | Monday 9-16, Asana 11 |
| Produktion | — | in Vorbereitung | Spezialsoftware 15-30 |
| Inventar | — | in Vorbereitung | Spezialsoftware 8-20 |
| Fuhrpark | — | in Vorbereitung | Spezialsoftware 10-20 |
| Helpdesk | 6 | buchbar | Zendesk ab 19 |
| Rapporte | — | in Vorbereitung | Proj.tools ab 24 |

### 3.6 Tools & Berichte

| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Berichte | — | in Vorbereitung | Power BI/Looker ab 10 |
| Formulare | — | in Vorbereitung | Typeform ab 25 |
| Wiki | 3 | buchbar | Notion ab 8, Confluence 5 |

---

## 4. Rabattstruktur

### 4.1 Volumen-Rabatt (automatisch, auf Modul-Summe)

| User-Staffel | Rabatt | Beispiel (30 EUR Basis) | Effektiv |
|--------------|--------|-------------------------|----------|
| 1-9 User | 0% | 30,00 | 30,00 |
| 10-24 User | 5% | 28,50 | 28,50 |
| 25-49 User | 10% | 27,00 | 27,00 |
| 50-99 User | 15% | 25,50 | 25,50 |
| 100-249 User | 20% | 24,00 | 24,00 |
| 250+ User | 25% + individuell | 22,50+ | ab 22,50 |

### 4.2 Paket-Rabatt (Branchenpakete)

15% auf alle im Paket enthaltenen Module, wenn mindestens 80% der Module aktiv bleiben. Zusatzmodule ausserhalb des Pakets zum regulaeren Listenpreis.

---

## 5. Branchenpakete

Onboarding-Vorlagen, kein separater Kaufgegenstand. Definieren vorausgewaehlte Module + aktivieren Paket-Rabatt.

> ⚠ **Diese Tabelle ist nicht verkaufsfähig.** Jedes der fünf Pakete enthält Module, die in
> Vorbereitung sind: Handwerk (Finanzen, Rapporte) · IT & Agentur (Meetings, Berichte) ·
> Dienstleister (Finanzen, Berichte) · Handel & Logistik (Inventar, Finanzen, Einkauf) ·
> Produktion (Produktion, Inventar, Fuhrpark — davon besteht das Paket überwiegend). Die
> „ab EUR"-Spalte ist zudem mit den alten Modulpreisen gerechnet und dadurch zu niedrig.
>
> Ein Paket neu zusammenzusetzen ist eine Produktentscheidung, keine Dokumentationspflege: Was
> ein Handwerk-Paket ohne Finanzen und Rapporte enthalten soll, muss Darien festlegen. Bis dahin
> bleibt die Tabelle als Ausgangspunkt stehen und ist **nicht** zu zitieren.

| Paket | Enthaltene Module | ab EUR/User/Mo (alt) | Markt-Vergleich |
|-------|-------------------|----------------------|-----------------|
| Handwerk | CRM, Aufgaben, Kalender, Chat, Zeiterfassung, Buchhaltung, Rapporte, Schichten | ~26 | ~45 |
| IT & Agentur | CRM, Aufgaben, Kalender, Chat, E-Mail, Meetings, Projekte, Wiki, Berichte | ~29 | ~50 |
| Dienstleister | CRM, Aufgaben, Kalender, Chat, E-Mail, Buchhaltung, Vertraege, Berichte | ~25 | ~43 |
| Handel & Logistik | CRM, Aufgaben, Kalender, Chat, Inventar, Buchhaltung, Einkauf | ~26 | ~44 |
| Produktion | CRM, Aufgaben, Kalender, Chat, Produktion, Inventar, Schichten, Zeiterfassung, Fuhrpark | ~33 | ~56 |

---

## 6. Support-Stufen

| Stufe | Preis | Reaktionszeit | Verfuegbarkeit | Leistungen |
|-------|-------|---------------|----------------|------------|
| Basis | 9 EUR/Mo Flat | 2 Werktage | Mo-Fr | Doku, In-App-Hilfe, Community, E-Mail-Ticket |
| Professional | 10% der Monatssumme, min. 29 EUR | 4h werktags | Mo-Fr | Prioritaets-Ticket, Live-Chat, Onboarding 60 Min., Screen-Sharing |
| Premium | 15% der Monatssumme, min. 79 EUR | 1h 24/7 | 24/7 | Dedizierter AP, monatl. Check-in, Incident-Hotline |

---

## 7. Abrechnungslogik

| Aspekt | Beschreibung |
|--------|-------------|
| Preisberechnung | Summe aller aktiven Modul-Zuweisungen pro User |
| Zyklus | Monatlich, zum 1. des Monats |
| User hinzufuegen | Sofort aktiv, anteilige Abrechnung |
| Modul entfernen | Wirkt ab naechster Periode |
| Nutzungsanalyse | Monatlich: ungenutzte Module markiert, Abbestell-Vorschlag |
| Rollenvorlagen | Admin speichert Modul-Sets ("Monteur", "Buero", "Leitung") |
| Gastnutzer | Kostenlos, eingeschraenkter Zugang |

---

## 8. ORBIT (Self-Hosted)

Zentria liefert physische Hardware (Synology NAS) und installiert vor Ort. Cosmi laeuft lokal. Zentria uebernimmt remote Wartung, Updates, Monitoring, Backup.

### 8.1 Tiers

| Tier | User | Hardware (Kauf) | Leasing/Mo | Setup | Wartung/Mo |
|------|------|-----------------|------------|-------|------------|
| Pod | 5-20 | Synology DS423+ ~900 EUR | ~30 EUR | 199 EUR (remote) | 39 EUR |
| Station | 20-80 | Synology DS1522+ ~2.200 EUR | ~65 EUR | 499 EUR (vor Ort) | 89 EUR |
| Command | 80-200+ | Synology RS1221+ ab ~5.500 EUR | ab ~150 EUR | 1.490 EUR (vor Ort) | 199 EUR |

### 8.2 ORBIT-Rabatt

20% auf alle Cosmi-Modulpreise (Grund: Zentria stellt keine Cloud-Infrastruktur bereit).

### 8.3 Meeting-Optionen

| Setup | Beschreibung | Fuer wen |
|-------|-------------|----------|
| Cloud (Standard) | Jitsi auf Hetzner VPS, im Modulpreis enthalten | Standard fuer alle |
| Lokal | Jitsi auf Mini-PC vor Ort (600-2.800 EUR) | Hoechster Datenschutz |
| Hybrid | Intern lokal, extern Cloud | Beste Balance |

### 8.4 Cloud-Backup

| Paket | Groesse | Zentria-Preis/Mo |
|-------|---------|------------------|
| S | bis 1 TB | 9 EUR |
| M | bis 5 TB | 19 EUR |
| L | bis 20 TB | 59 EUR |

### 8.5 ORBIT-Zielgruppen

- Aerzte, Anwaelte, Steuerberater, Behoerden (strenge Datenschutzanforderungen)
- Unternehmen mit schlechter Internetanbindung (Produktion, laendlich)
- Betriebe die keine Daten in der Cloud wollen
- Groessere Betriebe mit eigener IT-Infrastruktur

---

## 9. Beispielrechnungen

### 9.1 Handwerksbetrieb (20 User, COSMI Cloud, Professional Support)

Mit der 1.0-Palette und den Preisen aus §3. Die alte Fassung dieser Rechnung enthielt Meetings
und Buchhaltung — beide in Vorbereitung — und ergab dadurch 313,50 EUR.

| Posten | User | EUR/User | Summe/Mo |
|--------|------|----------|----------|
| Chat | 20 | 4 | 80 |
| Aufgaben | 20 | 3 | 60 |
| Zeiterfassung | 20 | 4 | 80 |
| Schichten | 20 | 4 | 80 |
| CRM & Vertrieb | 5 | 9 | 45 |
| Dokumente | 5 | 6 | 30 |
| Zwischensumme | | | 375,00 |
| Volumen-Rabatt (20 User, 5%) | | | -18,75 |
| Support Professional (10% von 356,25) | | | 35,63 |
| **GESAMT (zzgl. MwSt.)** | | | **391,88** |

> **Was dieser Betrieb heute nicht bekommt:** Finanzen und Rapporte. Beide sind in Vorbereitung —
> im Verkaufsgespräch gehört das gesagt, nicht die Summe.
>
> Für die Ersparnis gegenüber einem Tool-Stack gibt es genau eine nachgerechnete Grundlage:
> Preisanalyse §4.4, 20-Personen-Dienstleister, 894 EUR Fachtools gegen 649 EUR Cosmi = **27 %**.
> Die früher hier stehende Zahl „~1.060 EUR/Mo für 20 User" war nicht hergeleitet und ist
> entfernt. Eine eigene Vergleichszahl für dieses Handwerks-Szenario ist noch nicht gerechnet.

### 9.2 Handwerksbetrieb (30 User, ORBIT Station, Professional Support)

> ⚠ Auf dem alten Katalog und den alten Preisen gerechnet (Cosmi-Lizenz ~624 EUR). Zusätzlich
> offen: ob ORBIT überhaupt in der 1.0-Preisliste steht — das Auslieferungsmodell stellt es in
> Frage, die Website bewirbt es noch. Nicht zitieren, bis beides geklärt ist.

| Posten | Betrag |
|--------|--------|
| *Einmalig:* Hardware + Setup | 2.699 EUR |
| Wartung ORBIT Station | 89 EUR/Mo |
| Cloud Backup M (5 TB) | 19 EUR/Mo |
| Cosmi-Lizenz (Handwerk, -20%) | ~624 EUR/Mo |
| Volumen-Rabatt 30 User (10%) | -62 EUR/Mo |
| Support Professional (10%) | 67 EUR/Mo |
| **Monatlich gesamt (zzgl. MwSt.)** | **~737 EUR/Mo** |

> Vergleich: M365 Business Standard (22 EUR/User) fuer 30 User = 660 EUR/Mo, ohne lokale Kontrolle, US-Cloud.

---

## 10. Wettbewerbsvergleich

| Anbieter | Preis/User/Mo | Bereich | EU-Hosting | All-in-One |
|----------|---------------|---------|------------|------------|
| Pipedrive | ab 14 EUR | Nur CRM | Bedingt | Nein |
| HubSpot Starter | 15-20 EUR | CRM + Marketing | Bedingt | Teilweise |
| Salesforce | ab 25 EUR | CRM Enterprise | US-Cloud | Teilweise |
| Monday.com | ab 9 EUR | Projekte | Bedingt | Teilweise |
| Slack + Zoom + Asana etc. | ~54-67 EUR | Fragmentiert | Gemischt | Nein |
| **Cosmi (1.0-Palette)** | **3-9 EUR je Modul** | **13 Module, 11 in Vorbereitung** | **DE (Hetzner)** | **Wachsend** |

> Die Zeile lautete „Cosmi (Handwerk-Paket) ab 26 EUR" — abgeleitet aus einem Paket, das
> Module in Vorbereitung enthält (§5), und mit den alten Preisen gerechnet. Ersetzt durch die
> Angabe, die belegbar ist: die Preisspanne der buchbaren Module. „All-in-One" ist zu
> „Wachsend" korrigiert, solange elf Module fehlen.

---

## 11. Differenzierung

1. **Modulares "Zahl nur was du nutzt"** — kein Feature-Bloat, keine Ueberzahlung
2. **Branchenpakete** — vorkonfigurierte Loesungen, nicht generisches CRM
3. **Nutzungsanalyse** — aktiver Vorschlag ungenutzte Module abzubestellen (Trust-Signal)
4. **EU-Datensouveraenitaet** — Hetzner DE, Self-Hosted Option (ORBIT)
5. **Onsite-Prozessanalyse** — kein anderer CRM-Anbieter macht das
6. **Kein Vendor-Lock-In** — Self-Hosted + Datenexport
7. **Ein System statt sechs** — nachgerechnet rund ein Viertel unter dem heutigen Tool-Stack
   (Preisanalyse §4.4: 894 EUR gegen 649 EUR bei 20 Personen = 27 %), und was nicht gebucht ist,
   ist nicht installiert

> **Der frühere Punkt 7 lautete „50-90% guenstiger als fragmentierter Tool-Stack" und ist
> widerlegt.** Die Preisanalyse rechnet 27 % nach und hält ausdrücklich fest, dass der alte Claim
> „mit der 1.0-Palette nicht mehr haltbar" ist — er war schon mit dem vollen Katalog optimistisch
> gerechnet, weil er jedem der 20 User jedes Tool zuschrieb. Der Verkauf läuft über
> Souveränität, Anpassbarkeit und „ein System statt sechs", nicht primär über den Preis.
>
> Dieselbe Zahl steht noch an einer weiteren Stelle: `wahre-kosten-tool-chaos.md` im
> Website-Repo rechnet 1.280 EUR gegen „unter 500 EUR" und behauptet damit implizit über 60 %.
> Das gehört korrigiert — anderes Repo, hier nur vermerkt.

---

## 12. Einmalkosten (Services)

| Posten | Preis | Anmerkung |
|--------|-------|-----------|
| Onsite-Prozessanalyse (1 Woche) | 5.000-8.000 EUR | USP, inkl. Branchenpaket-Konfiguration |
| Datenmigration (aus Alt-System) | 1.000-3.000 EUR | Abhaengig von Quelldaten |
| Custom WASM-Plugins (pro Plugin) | 2.000-10.000 EUR | Abhaengig von Komplexitaet |
| Branchenpaket-Konfiguration (Remote) | 500-1.500 EUR | Ohne Onsite |

---

## 13. Code-Referenz

`desktop/src/renderer/src/lib/pricing.ts` bezeichnet sich im Kopfkommentar als „single source of
truth" und „Spiegel zu .knowledge/pricing.md / docs/PRICING.md". Nach der Aktualisierung dieses
Dokuments stimmt das nicht mehr — die Drift ist gemessen, nicht vermutet:

| Stelle | Zustand |
|--------|---------|
| `MODULE_PRICES` (pricing.ts:37-67) | trägt die **alten** Preise (crm 6, documents 2, calendar 2, team 3, timetracking 3, projects 5, helpdesk 5, wiki 2) |
| Modulstatus | pricing.ts kennt keine Unterscheidung buchbar / in Vorbereitung, behandelt alle 24 gleich |
| `SUPPORT_TIER_PRICES` (pricing.ts:98-102) | `standard 0 / priority 99 / enterprise 299` — flache Beträge und andere Namen als die Stufen in §6 (Basis 9 flat / Professional 10% min. 29 / Premium 15% min. 79), die Website und Knowledge-Note führen |
| `VOLUME_TIERS` (pricing.ts:82-89) | stimmt mit §4.1 überein |

Genutzt wird die Datei von `mocks/billing-mock.ts` für Demo- und Insight-Daten, nicht für echte
Kundenabrechnung — ein Abrechnungssystem gibt es bewusst noch nicht (Streichliste des
Launch-Lagebilds). Deshalb ist die Drift heute kein Geldrisiko, aber die Demo zeigt Preise, die
niemand mehr verkauft. Nachziehen, sobald die Modulauswahl der Demo geklärt ist; die
Support-Stufen sind der ältere und größere Widerspruch von beiden.
