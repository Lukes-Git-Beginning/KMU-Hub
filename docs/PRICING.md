# Pricing-Modell KMU Hub (v2)

> Aktualisiert: Februar 2026 — Role-Based Pricing, Einmalkauf-Option, Branchenpakete

---

## 1. Markt-Analyse (DACH, Stand 2026)

### 1.1 CRM-Markt DACH

| Anbieter | Preis/User/Monat | Zielgruppe | Self-Hosted | Besonderheit |
|----------|-----------------|------------|-------------|--------------|
| Salesforce | 25-300 EUR | Enterprise | Nein | Marktfuehrer |
| HubSpot | 0-130 EUR | SMB-Enterprise | Nein | Kostenlose View-Only Seats |
| Pipedrive | 15-99 EUR | SMB | Nein | Sales-fokussiert |
| Monday CRM | 10-24 EUR | SMB | Nein | Hybrid Work-/CRM |
| Zoho CRM | 14-52 EUR | SMB | Nein | Lite User ab 5 USD |
| MS Dynamics 365 | 8-135 USD | Enterprise | Nein | Team Member 8 USD |
| weclapp | 39-169 EUR | SMB-Mittelstand | Nein | DACH-fokussiert, modular |
| Twenty CRM | Open Source | Dev-Teams | Ja | OSS-Newcomer |
| SuiteCRM | Open Source | SMB | Ja | OSS-Standard |

### 1.2 Branchensoftware-Vergleich

| Anbieter | Preis/User/Monat | Branche | Modell |
|----------|-----------------|---------|--------|
| Plancraft | 29,90-48 EUR | Handwerk | SaaS |
| HERO Software | 59 EUR | Handwerk | SaaS |
| openHandwerk | ab 16 EUR | Handwerk | SaaS |
| Labelwin | 1.850 EUR einmalig | Handwerk | Kauflizenz + Wartung |
| combit CRM | ab 430 EUR/User einmalig | Cross-Industry | Kauflizenz |
| 1CRM | 17-55 EUR (Cloud) / 260 EUR/User/Jahr (Perpetual) | Cross-Industry | Hybrid |

### 1.3 Pricing-Trends

- **Subscription Fatigue:** 80% der SMBs fuehlen sich ueber-abonniert (OpenPR 2026)
- **Hybrid-Pricing:** Kombination aus Abo + Einmalkauf steigert Revenue um 20-40%
- **Role-Based Pricing:** Waechst 2x schneller im Revenue als reines Per-Seat (OpenView Partners)
- **IT-Budget KMU:** 2-4% des Umsatzes, 33-52 EUR/Monat/Mitarbeiter fuer Software

---

## 2. User-Tier-Modell (Role-Based Pricing)

### 2.1 Full User (Gesellschafter / Admin / Power User)

- **Zugriff:** Alle Module — CRM, Chat, Video, Calendar, PM, Email, Documents, Finance, HR, Automation, Unified Inbox, Integrationen
- **Admin:** Benutzerverwaltung, Einstellungen, Audit Logs, DSGVO-Tools
- **Extras:** Custom Fields, Workflows, WASM Plugins, API-Zugang
- **Zielgruppe:** Geschaeftsfuehrer, Abteilungsleiter, Projektmanager

### 2.2 Standard User

- **Zugriff:** CRM (Lesen + eigene Kontakte), Chat, Calendar, PM (Aufgaben), Email, Documents (Lesen + Upload)
- **Kein Zugriff:** Finance, HR-Admin, Automation-Erstellung, Plugins, API, Audit Logs
- **Zielgruppe:** Sachbearbeiter, Techniker, Berater, Verkaeufer

### 2.3 Light User (Chat & Collaboration)

- **Zugriff:** Chat (alle Kanaele), Calendar (eigener + Team), Video-Calls, Documents (Lesen), Presence
- **Kein Zugriff:** CRM, PM, Email, Finance, HR, Automation
- **Zielgruppe:** Monteure, Lagerarbeiter, Lehrlinge, Teilzeitkraefte, Freelancer

### 2.4 Guest User (kostenlos)

- **Zugriff:** Guest-Chat via Token-Link (Phase 17.5)
- **Limitiert:** 30 Nachrichten/Minute, kein Login/Account noetig
- **Zielgruppe:** Kunden, Lieferanten, externe Partner

### 2.5 Feature-Vergleichsmatrix

| Feature | Full | Standard | Light | Guest |
|---------|:----:|:--------:|:-----:|:-----:|
| CRM (Vollzugriff) | Ja | Lesen + Eigene | — | — |
| Chat & Messaging | Ja | Ja | Ja | Nur Guest-Chat |
| Video & Voice | Ja | Ja | Ja | — |
| Calendar | Ja | Ja | Eigener + Team | — |
| Project Management | Ja | Aufgaben | — | — |
| Email Integration | Ja | Ja | — | — |
| Documents & Files | Ja | Ja | Lesen | — |
| Finance / Rechnungen | Ja | — | — | — |
| HR & Zeiterfassung | Ja | Eigene Daten | Stempeln | — |
| Automation Engine | Ja | Ausloesen | — | — |
| Unified Inbox | Ja | Ja | — | — |
| Admin & Settings | Ja | — | — | — |
| API Access | Ja | — | — | — |
| WASM Plugins | Ja | — | — | — |

---

## 3. Preismodelle

### 3.1 SaaS (Cloud, EU-Hosting)

| User-Typ | Monatlich | Jaehrlich (pro Monat) | Ersparnis |
|----------|-----------|----------------------|-----------|
| Full User | 49 EUR | 39 EUR | 20% |
| Standard User | 29 EUR | 23 EUR | 21% |
| Light User | 9 EUR | 7 EUR | 22% |
| Guest User | 0 EUR | 0 EUR | — |

Mindestbestellung: 1 Full User.

### 3.2 Self-Hosted Lizenz (jaehrlich)

| Plan | Preis/Jahr | User-Limits | Features |
|------|-----------|-------------|----------|
| Standard | 4.000 EUR | 10 Full + 20 Standard + unbegr. Light | Updates, Community Support |
| Professional | 7.000 EUR | 25 Full + 50 Standard + unbegr. Light | + Priority Support |
| Enterprise | 12.000 EUR | Unlimited alle Typen | + SLA, + 5h Custom Dev Budget |

### 3.3 VIP Einmalkauf ("Kauflizenz") — Nur Self-Hosted

#### Variante A: Direkte Kauflizenz + Support-Pakete

Einmalige Lizenzgebuehr fuer permanente Nutzung (Self-Hosted):

| Paket | Einmalpreis | Inkl. Support | Inkl. Updates | Max Users |
|-------|-----------|---------------|---------------|-----------|
| Starter | 8.000 EUR | 1 Jahr | 1 Jahr | 5 Full + 15 Standard + unbegr. Light |
| Professional | 15.000 EUR | 2 Jahre | 2 Jahre | 15 Full + 40 Standard + unbegr. Light |
| Enterprise | 28.000 EUR | 4 Jahre | 4 Jahre | Unlimited |

**Nach Ablauf der inkludierten Periode:**
- Wartungsvertrag (optional): 20% des Kaufpreises pro Jahr
- Ohne Wartungsvertrag: Software laeuft weiter, aber keine Updates / kein Support
- Wartungsvertrag jederzeit wieder aktivierbar (+ Nachzahlung fuer verpasste Update-Zeitraeume)

**Inkludiert:**
- Permanente Nutzungslizenz (Software gehoert dem Kunden)
- Alle Module die zum Kaufzeitpunkt verfuegbar sind
- Basis-Datenmigration
- 2 Stunden Remote-Onboarding

**Nicht inkludiert:**
- Onsite-Prozessanalyse (separat buchbar)
- Custom WASM-Plugins (separat buchbar)
- Hosting-Infrastruktur (Kunden-Server)
- Neue Module nach Kauf (nur via aktiven Wartungsvertrag)

#### Variante B: JetBrains-Fallback-Modell

Fuer Kunden die mit SaaS oder Self-Hosted-Abo starten, aber langfristig besitzen wollen:

- Kunde zahlt normale SaaS- oder Self-Hosted-Jahresgebuehr
- **Nach 12 durchgehenden Monaten:** Perpetual Fallback auf die aktuelle Version
- **Nach 24 Monaten:** Aktualisierte Fallback-Version
- Perpetual Fallback = dauerhafte Nutzungslizenz dieser Version (Self-Hosted)
- Abo kann danach gekuendigt werden — Software laeuft auf der Fallback-Version weiter
- Abo-Fortsetzung: Weiterhin Updates und Support

**Vorteil:** Kein grosses Upfront-Investment, trotzdem Exit-Sicherheit und kein Vendor-Lock-In.

### 3.4 Zusatz-Optionen

| Option | Preis |
|--------|-------|
| LiveKit Video (SaaS) | 5 EUR/User/Monat |
| Erweiterter Support (SaaS) | 500 EUR/Monat |
| Custom Development | 150 EUR/Stunde |

---

## 4. Branchenspezifische Pakete

Jedes Branchenpaket umfasst: empfohlene User-Zusammensetzung, vorkonfigurierte Module, branchenspezifische Custom Fields und Automationen.

### 4.1 Handwerk (Elektrik, Sanitaer, Schreinerei, etc.)

**Typischer Betrieb:** 1 Meister + 2 Buero + 8 Monteure

| Rolle | User-Typ | Anzahl | Einzelpreis | Summe |
|-------|----------|--------|------------|-------|
| Meister / GF | Full | 1 | 49 EUR | 49 EUR |
| Buero / Verwaltung | Standard | 2 | 29 EUR | 58 EUR |
| Monteure | Light | 8 | 9 EUR | 72 EUR |
| **Gesamt/Monat** | | **11 User** | | **179 EUR** |

> Vergleich Alt-Modell: 11 x 39 EUR = 429 EUR/Monat → **58% guenstiger**

**Vorkonfigurierte Features:**
- CRM: Kundenkartei, Auftraege, Angebots-Pipeline
- Chat: Baustellenkanaele, Direktnachrichten an Monteure
- Calendar: Einsatzplanung, Urlaubskalender
- PM: Aufgaben pro Baustelle/Auftrag
- Finance: Angebote → Rechnungen → Mahnung
- HR: Zeiterfassung (Stempeln per App), Urlaubsantraege
- Guest-Chat: Kundenanfragen-Widget fuer Website

**Empfohlene Automationen:**
- Auftrag abgeschlossen → Rechnung-Entwurf erstellen
- Neuer Kundenanruf → Aufgabe fuer Buero
- Rechnung ueberfaellig → Mahnung nach 14/28/42 Tagen

**Einmalkauf-Empfehlung:** Professional (15.000 EUR) — adressiert Subscription Fatigue im Handwerk direkt.

### 4.2 Dienstleistung & Beratung (Unternehmensberatung, Rechtsanwalt, Steuerberater)

**Typischer Betrieb:** 3 Partner + 5 Berater + 2 Assistenz

| Rolle | User-Typ | Anzahl | Einzelpreis | Summe |
|-------|----------|--------|------------|-------|
| Partner | Full | 3 | 49 EUR | 147 EUR |
| Berater | Standard | 5 | 29 EUR | 145 EUR |
| Assistenz | Standard | 2 | 29 EUR | 58 EUR |
| **Gesamt/Monat** | | **10 User** | | **350 EUR** |

**Vorkonfigurierte Features:**
- CRM: Mandanten-/Kundenverwaltung, Akquise-Pipeline
- PM: Projektbasiertes Arbeiten, Zeiterfassung pro Mandat
- Email: Mandantenbezogene E-Mail-Verknuepfung
- Calendar: Terminplanung, Videokonferenzen
- Finance: Honorarabrechnungen nach Aufwand
- Documents: Vertragsverwaltung, Dokumenten-Sharing

**Empfohlene Automationen:**
- Neuer Deal → Projekt anlegen + Team zuweisen
- Zeiterfassung > Budget → Benachrichtigung an Partner
- Vertrag unterzeichnet → Willkommens-E-Mail + Onboarding-Aufgaben

### 4.3 Handel / Einzelhandel

**Typischer Betrieb:** 1 Inhaber + 1 Einkauf + 3 Verkauf + 2 Lager

| Rolle | User-Typ | Anzahl | Einzelpreis | Summe |
|-------|----------|--------|------------|-------|
| Inhaber | Full | 1 | 49 EUR | 49 EUR |
| Einkauf | Standard | 1 | 29 EUR | 29 EUR |
| Verkauf | Standard | 3 | 29 EUR | 87 EUR |
| Lager | Light | 2 | 9 EUR | 18 EUR |
| **Gesamt/Monat** | | **7 User** | | **183 EUR** |

**Vorkonfigurierte Features:**
- CRM: Lieferanten + Kundenkontakte, Bestellpipeline
- Chat: Teamkommunikation, Lieferanten-Guest-Chat
- Finance: Rechnungen, Lieferscheine, Mahnwesen
- Calendar: Liefertermine, Messen
- Email: Bestellbestaetigungen, Kundenkommunikation

**Empfohlene Automationen:**
- Neue Bestellung → Aufgabe fuer Lager
- Lieferung eingetroffen → Benachrichtigung an Einkauf
- Kundenbestellung → Rechnung-Entwurf

### 4.4 IT / Agentur (Webentwicklung, Marketing, MSP)

**Typischer Betrieb:** 2 GF + 8 Entwickler/Designer + 2 PM + 3 Freelancer

| Rolle | User-Typ | Anzahl | Einzelpreis | Summe |
|-------|----------|--------|------------|-------|
| Geschaeftsfuehrung | Full | 2 | 49 EUR | 98 EUR |
| Entwickler / Designer | Standard | 8 | 29 EUR | 232 EUR |
| Projektmanager | Standard | 2 | 29 EUR | 58 EUR |
| Freelancer (extern) | Light | 3 | 9 EUR | 27 EUR |
| **Gesamt/Monat** | | **15 User** | | **415 EUR** |

**Vorkonfigurierte Features:**
- CRM: Lead-Pipeline, Kundenmanagement
- PM: Kanban, Gantt, Sprints, Zeiterfassung
- Chat: Projekt-Channels, Kunden-Guest-Chat
- Calendar: Projektmeilensteine, Sprint-Planung
- Video: Screen-Sharing, Daily Standups
- Finance: Projektbasierte Abrechnung

**Empfohlene Automationen:**
- Sprint abgeschlossen → Timesheet-Report an GF
- Neuer Lead → Qualifizierungsaufgabe erstellen
- Rechnung ueberfaellig → Benachrichtigung an GF

**Einmalkauf-Empfehlung:** JetBrains-Fallback-Modell (Variante B) — tech-affine Zielgruppe kennt und schaetzt dieses Modell.

---

## 5. Einmalkosten (Services)

| Posten | Preis | Anmerkung |
|--------|-------|-----------|
| Onsite-Prozessanalyse (1 Woche) | 5.000-8.000 EUR | USP — inkl. Branchenpaket-Konfiguration |
| Initiales Setup & Konfiguration | Inkludiert (SaaS) / 1.500 EUR (Self-Hosted) | Self-Hosted erfordert Server-Setup |
| Datenmigration (aus Alt-System) | 1.000-3.000 EUR | Optional, abhaengig von Quelldaten |
| Custom WASM-Plugins (pro Plugin) | 2.000-10.000 EUR | Optional, abhaengig von Komplexitaet |
| Branchenpaket-Konfiguration (Remote) | 500-1.500 EUR | Ohne Onsite, basierend auf Standard-Templates |

---

## 6. Differenzierung

1. **Role-Based Pricing** — Kunden zahlen nur fuer die Funktionen die jeder Mitarbeiter braucht (Monteur ≠ Geschaeftsfuehrer)
2. **VIP Einmalkauf** — einzigartig im DACH-CRM-Markt fuer All-in-One Loesungen
3. **Branchenpakete** — vorkonfigurierte Loesungen, nicht nur generisches CRM
4. **Onsite-Prozessanalyse** — kein anderer CRM-Anbieter macht das
5. **EU-Datensouveraenitaet** — Self-Hosted Option, EU-only Hosting
6. **Faire Preise** — deutlich unter Salesforce/HubSpot Enterprise
7. **Kein Vendor-Lock-In** — Self-Hosted + Datenexport + JetBrains-Fallback
8. **All-in-One** — CRM + Chat + Video in einem Tool (kein Slack + Salesforce + Zoom)

---

## 7. Revenue-Projektion

### 7.1 Konservativ (Jahr 1): ~196.000 EUR

**SaaS-Kunden (20 Betriebe):**
- 8 Handwerksbetriebe: 8 x 179 EUR x 12 = 17.184 EUR
- 5 Dienstleister: 5 x 350 EUR x 12 = 21.000 EUR
- 3 Handelsbetriebe: 3 x 183 EUR x 12 = 6.588 EUR
- 4 IT/Agenturen: 4 x 415 EUR x 12 = 19.920 EUR
- **SaaS Subtotal: 64.692 EUR** (MRR: ~5.391 EUR)

**Einmalkauf:** 3 Kauflizenzen x 15.000 EUR = **45.000 EUR**

**Self-Hosted (Abo):** 2 x Standard (4.000 EUR) = **8.000 EUR**

**Services:**
- 10 Onsite-Analysen x 6.500 EUR = 65.000 EUR
- 5 Datenmigrationen x 2.000 EUR = 10.000 EUR
- 3 Branchenpaket-Configs x 1.000 EUR = 3.000 EUR
- **Services Subtotal: 78.000 EUR**

### 7.2 Optimistisch (Jahr 1): ~335.000 EUR

**SaaS-Kunden (32 Betriebe):**
- 12 Handwerk + 8 Dienstl. + 5 Handel + 7 IT = **105.216 EUR** (MRR: ~8.768 EUR)

**Einmalkauf:** 5 x 15.000 EUR = **75.000 EUR**

**Self-Hosted:** 3 Professional + 1 Enterprise = **33.000 EUR**

**Services:** 15 Onsite + 8 Migrationen + 5 Configs + 20h Custom Dev = **121.500 EUR**

### 7.3 Jahr 2-3 Ausblick

- Wartungsvertraege von Einmalkauf-Kunden (20% p.a.) generieren wiederkehrenden Revenue
- Cross-Sell: Branchenpakete und Upgrades (Light → Standard, Standard → Full)
- User-Wachstum bei bestehenden Kunden
- **Ziel Jahr 2:** 350.000-500.000 EUR
- **Ziel Jahr 3:** 500.000-800.000 EUR

---

## 8. Risikobewertung Einmalkauf-Modell

| Risiko | Wahrscheinlichkeit | Impact | Mitigation |
|--------|-------------------|--------|------------|
| Kannibalisiert SaaS-Revenue | Mittel | Hoch | Break-Even erst nach ~43 Monaten SaaS (15k/350) — sicher |
| Cashflow-Loch nach Einmalzahlung | Mittel | Hoch | Max 30% Gesamtrevenue aus Einmalkauf; max 2-3 Kunden/Quartal |
| Support-Last ohne laufende Einnahmen | Hoch | Mittel | Klare SLA-Abgrenzung; Wartungsvertrag aktiv empfehlen |
| 15k zu teuer fuer kleine Betriebe | Mittel | Mittel | Starter-Paket 8k; Ratenzahlung (3-6 Raten) |
| Neue Module: Kunde fuehlt sich benachteiligt | Mittel | Mittel | Kauflizenz = Module zum Kaufzeitpunkt; neue via Wartungsvertrag |
| Support-Kosten Self-Hosted hoeher als erwartet | Hoch | Mittel | Docker-Compose standardisieren; Installations-Doku automatisieren |

**Schutzmassnahmen:**
1. Einmalkauf NUR fuer Self-Hosted (keine SaaS-Lifetime-Lizenzen)
2. Mindestens 1 Jahr Support/Updates bei Kauf inkludiert
3. Maximale Einmalkauf-Kapazitaet: 2-3 Kunden pro Quartal (Team-Kapazitaet)
4. Quartals-Review der Einmalkauf vs. SaaS Ratio

---

## 9. Pricing-Regeln & Governance

- **Mindestbestellung:** 1 Full User
- **Jaehrliche Preisanpassung:** Maximal 5% (Inflationsausgleich)
- **Downgrade-Policy:** User-Typ-Wechsel zum Monatsende
- **Branchenpaket-Wechsel:** Jederzeit moeglich (Konfiguration wird nicht geloescht)
- **Einmalkauf-Rueckgabe:** 30 Tage Widerrufsrecht (gesetzlich DACH)
- **Wartungsvertrag-Kuendigung:** 3 Monate zum Jahresende
- **Ratenzahlung (Einmalkauf):** 3-6 Monatsraten, 0% Aufschlag
