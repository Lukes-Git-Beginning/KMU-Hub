# Pricing-Modell Cosmi (v3) — Modul-x-User

> Aktualisiert: April 2026 — Modul-basiertes Pricing, ORBIT Self-Hosted, Branchenpakete
> Quelle: Darien (COSMI Preiskonzept + ORBIT Preiskonzept April 2026)
> Kanonische Knowledge-Base-Note: `.knowledge/pricing.md`

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

Alle Preise in EUR pro User pro Monat, zzgl. MwSt.

### 3.1 Kern-Module

| Modul | Preis | Markt-Vergleich | Vorteil |
|-------|-------|-----------------|---------|
| CRM & Vertrieb | 6 | Pipedrive ab 14, HubSpot ab 15 | bis 60% guenstiger |
| Aufgaben | 3 | Asana ab 11, Monday ab 9 | bis 73% guenstiger |
| Kalender | 2 | M365 (8-22) | Kein Extra-Tool |
| Dokumente | 2 | Notion ab 8 | bis 75% guenstiger |

### 3.2 Kommunikation

| Modul | Preis | Markt-Vergleich | Vorteil |
|-------|-------|-----------------|---------|
| Chat | 4 | Slack Pro 7-8 | bis 50% guenstiger |
| E-Mail | 3 | M365 | Integriert |
| Meetings | 4 | Zoom Pro 13-15 | bis 73% guenstiger |
| Telefonie | 5 | VoIP 8-15 | Integriert in Cosmi |

### 3.3 Buchhaltung & Einkauf

| Modul | Preis | Markt-Vergleich | Vorteil |
|-------|-------|-----------------|---------|
| Buchhaltung | 6 | sevDesk ab 10, Lexoffice ab 7 | Vollintegriert |
| Einkauf | 5 | Spezialsoftware 10-20 | bis 75% guenstiger |
| Vertraege | 5 | Spezialsoftware 10-25 | bis 80% guenstiger |
| Vermietung | 5 | Spezialsoftware 20-40 | bis 88% guenstiger |

### 3.4 Team & HR

| Modul | Preis | Markt-Vergleich | Vorteil |
|-------|-------|-----------------|---------|
| Team | 3 | M365/HR-Tools | Kein Wechsel |
| Schichten | 4 | Spezialsoftware 5-10 | Integriert |
| Zeiterfassung | 3 | Clockify/Harvest ab 5 | bis 40% guenstiger |

### 3.5 Projekte & Betrieb

| Modul | Preis | Markt-Vergleich | Vorteil |
|-------|-------|-----------------|---------|
| Projekte | 5 | Monday 9-16, Asana 11 | bis 69% guenstiger |
| Produktion | 7 | Spezialsoftware 15-30 | bis 77% guenstiger |
| Inventar | 5 | Spezialsoftware 8-20 | bis 75% guenstiger |
| Fuhrpark | 5 | Spezialsoftware 10-20 | bis 75% guenstiger |
| Helpdesk | 5 | Zendesk ab 19 | bis 74% guenstiger |
| Rapporte | 3 | Proj.tools ab 24 | Vollintegriert |

### 3.6 Tools & Berichte

| Modul | Preis | Markt-Vergleich | Vorteil |
|-------|-------|-----------------|---------|
| Berichte | 3 | Power BI/Looker ab 10 | Integriert |
| Formulare | 2 | Typeform ab 25 | bis 92% guenstiger |
| Wiki | 2 | Notion ab 8, Confluence 5 | bis 75% guenstiger |

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

| Paket | Enthaltene Module | ab EUR/User/Mo | Markt-Vergleich |
|-------|-------------------|----------------|-----------------|
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

| Posten | User | EUR/User | Summe/Mo |
|--------|------|----------|----------|
| Chat | 20 | 4 | 80 |
| Meetings | 8 | 4 | 32 |
| CRM & Vertrieb | 5 | 6 | 30 |
| Zeiterfassung | 20 | 3 | 60 |
| Buchhaltung | 3 | 6 | 18 |
| Schichten | 20 | 4 | 80 |
| Zwischensumme | | | 300 |
| Volumen-Rabatt (20 User, 5%) | | | -15 |
| Support Professional (10%) | | | 28,50 |
| **GESAMT (zzgl. MwSt.)** | | | **313,50** |

> Vergleich: Pipedrive + Slack + Clockify + Asana + Zoom = ~1.060 EUR/Mo fuer 20 User, fragmentiert, US-Cloud.

### 9.2 Handwerksbetrieb (30 User, ORBIT Station, Professional Support)

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
| **Cosmi (Handwerk-Paket)** | **ab 26 EUR** | **All-in-One** | **DE (Hetzner)** | **Ja** |

---

## 11. Differenzierung

1. **Modulares "Zahl nur was du nutzt"** — kein Feature-Bloat, keine Ueberzahlung
2. **Branchenpakete** — vorkonfigurierte Loesungen, nicht generisches CRM
3. **Nutzungsanalyse** — aktiver Vorschlag ungenutzte Module abzubestellen (Trust-Signal)
4. **EU-Datensouveraenitaet** — Hetzner DE, Self-Hosted Option (ORBIT)
5. **Onsite-Prozessanalyse** — kein anderer CRM-Anbieter macht das
6. **Kein Vendor-Lock-In** — Self-Hosted + Datenexport
7. **Faire Preise** — 50-90% guenstiger als fragmentierter Tool-Stack

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

Die TypeScript-Datenstrukturen (MODULES, VOLUME_DISCOUNTS, SUPPORT_TIERS, BRANCH_PACKAGES, ORBIT_TIERS, calculatePrice(), calculateOrbitPrice()) sind in den Original-Dokumenten von Darien definiert und werden bei Studio/Billing-Implementierung als kanonische Quelle verwendet.
