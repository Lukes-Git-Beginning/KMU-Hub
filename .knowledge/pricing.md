---
tags: [pricing, strategie, cosmi, orbit]
updated: 2026-08-17
---
# Preismodell — COSMI (SaaS) + ORBIT (Self-Hosted)

> Quelle: Darien, April 2026; Modulpreise und Modulstatus am 2026-08-17 auf den Stand der Website
> gezogen (`.planning/preis-und-kostenanalyse-2026-08-13.md` §7.1, Website `src/data/modules.ts`).
>
> **Nicht mehr kanonisch für Paketsummen.** Die Branchenpakete und die abgeleiteten „ab EUR"-Beträge
> unten sind mit dem alten 24-Modul-Katalog gerechnet, und jedes Paket enthält Module, die in
> Vorbereitung sind. Details und die offenen Entscheidungen (79-€-Grundgebühr, Mindestbestellwert,
> ORBIT in der Preisliste) stehen in `docs/PRICING.md`. Die Code-Spiegelung
> `desktop/src/renderer/src/lib/pricing.ts` trägt weiterhin die alten Preise — siehe
> `docs/PRICING.md` §13.

## Grundprinzip

**Modul-x-User-Modell** — kein fixes Tier-System. Preis = Summe der Modulpreise pro User, minus Rabatte, plus Support.

```
Monatspreis = Summe(Modulpreis x User mit Modul) x (1 - Volumen-Rabatt) + Support
```

Was es NICHT gibt: Feste Light/Standard/Full-Tiers, Plattform-Fee, Mindestmodule, Jahresvertrag.
Was es gibt: Modul-Zuweisungen pro User, Rollenvorlagen, Volumen-Rabatt, Branchenpakete (15%), Gastnutzer (kostenlos).

---

## COSMI Modulpreise (EUR/User/Monat, zzgl. MwSt.)

**13 buchbar, 11 in Vorbereitung** (Modul-Zuschnitt, `.planning/preis-und-kostenanalyse-2026-08-13.md`
§1.1). Preise entsprechen `src/data/modules.ts` im Website-Repo — der Fassung, die ein Interessent
sieht. Für die 11 steht bewusst kein Preis: sie sind nicht verkäuflich, und ein Preis in einer
Tabelle wird zitiert. Summe der 13 buchbaren Module: **60 EUR** pro User und Monat.

### Kern
| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| CRM & Vertrieb | 9 | buchbar | Pipedrive ab 14, HubSpot ab 15 |
| Aufgaben | 3 | buchbar | Asana ab 11, Monday ab 9 |
| Kalender | 3 | buchbar | In M365 enthalten (8-22) |
| Dokumente | 6 | buchbar | Notion ab 8 |

### Kommunikation
| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Chat | 4 | buchbar | Slack Pro 7-8 |
| E-Mail | 3 | buchbar | In M365 enthalten |
| Meetings | — | in Vorbereitung | Zoom Pro 13-15 |
| Telefonie & Kampagnen | — | in Vorbereitung | VoIP 8-15 |

### Finanzen & Einkauf
| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Finanzen | — | in Vorbereitung | sevDesk ab 10, Lexoffice ab 7 |
| Einkauf | — | in Vorbereitung | Spezialsoftware 10-20 |
| Vertraege | 5 | buchbar | Spezialsoftware 10-25 |
| Vermietung | — | in Vorbereitung | Spezialsoftware 20-40 |

### Team & HR
| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Team | 4 | buchbar | In M365/HR-Tools enthalten |
| Schichten | 4 | buchbar | Spezialsoftware 5-10 |
| Zeiterfassung | 4 | buchbar | Clockify/Harvest ab 5 |

### Projekte & Betrieb
| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Projekte | 6 | buchbar | Monday 9-16, Asana 11 |
| Produktion | — | in Vorbereitung | Spezialsoftware 15-30 |
| Inventar | — | in Vorbereitung | Spezialsoftware 8-20 |
| Fuhrpark | — | in Vorbereitung | Spezialsoftware 10-20 |
| Helpdesk | 6 | buchbar | Zendesk ab 19 |
| Rapporte | — | in Vorbereitung | In Proj.tools ab 24 |

### Tools & Berichte
| Modul | Preis | Status | Markt-Vergleich |
|-------|-------|--------|-----------------|
| Berichte | — | in Vorbereitung | Power BI/Looker ab 10 |
| Formulare | — | in Vorbereitung | Typeform ab 25 |
| Wiki | 3 | buchbar | Notion ab 8, Confluence 5 |

---

## Rabattstruktur

### Volumen-Rabatt (automatisch, auf Modul-Summe)
| User-Staffel | Rabatt |
|--------------|--------|
| 1-9 | 0% |
| 10-24 | 5% |
| 25-49 | 10% |
| 50-99 | 15% |
| 100-249 | 20% |
| 250+ | 25% + individuell |

### Paket-Rabatt
15% auf enthaltene Module bei Branchenpaket-Wahl (mind. 80% Module aktiv).

---

## Branchenpakete (Onboarding-Vorlagen mit 15% Rabatt)

> ⚠ Alte Preise, und jedes Paket enthält Module in Vorbereitung — Handwerk (Finanzen, Rapporte),
> IT & Agentur (Meetings, Berichte), Dienstleister (Finanzen, Berichte), Handel & Logistik
> (Inventar, Finanzen, Einkauf), Produktion (Produktion, Inventar, Fuhrpark). Nicht zitieren; die
> Neuzusammensetzung ist eine Produktentscheidung, siehe `docs/PRICING.md` §5.

| Paket | Module | ab EUR/User/Monat (alt) |
|-------|--------|-------------------|
| Handwerk | CRM, Aufgaben, Kalender, Chat, Zeiterfassung, Buchhaltung, Rapporte, Schichten | ~26 |
| IT & Agentur | CRM, Aufgaben, Kalender, Chat, E-Mail, Meetings, Projekte, Wiki, Berichte | ~29 |
| Dienstleister | CRM, Aufgaben, Kalender, Chat, E-Mail, Buchhaltung, Vertraege, Berichte | ~25 |
| Handel & Logistik | CRM, Aufgaben, Kalender, Chat, Inventar, Buchhaltung, Einkauf | ~26 |
| Produktion | CRM, Aufgaben, Kalender, Chat, Produktion, Inventar, Schichten, Zeiterfassung, Fuhrpark | ~33 |

---

## Support-Stufen

| Stufe | Preis | Reaktionszeit | Verfuegbarkeit |
|-------|-------|---------------|----------------|
| Basis | 9 EUR/Monat Flat | 2 Werktage | Mo-Fr |
| Professional | 10% der Monatssumme, min. 29 EUR | 4h | Mo-Fr |
| Premium | 15% der Monatssumme, min. 79 EUR | 1h | 24/7 |

---

## Abrechnungslogik

- Monatlich, auf Basis aktiver Profile zum 1. des Monats
- User hinzufuegen: sofort aktiv, anteilige Abrechnung
- Modul entfernen: wirkt ab naechster Periode
- Nutzungsanalyse: monatlicher Report ueber ungenutzte Module (Abbestell-Vorschlag)
- Rollenvorlagen: Admin kann Modul-Sets speichern ("Monteur", "Buero", "Leitung")
- Gastnutzer: kostenlos, eingeschraenkter Zugang

---

## ORBIT (Self-Hosted)

Physische Hardware beim Kunden, COSMI laeuft lokal. Zentria uebernimmt remote Wartung, Updates, Monitoring.

### Tiers

| Tier | User | Hardware (Kauf) | Leasing/Mo | Setup | Wartung/Mo |
|------|------|-----------------|------------|-------|------------|
| Pod | 5-20 | Synology DS423+ ~900 EUR | ~30 EUR | 199 EUR (remote) | 39 EUR |
| Station | 20-80 | Synology DS1522+ ~2.200 EUR | ~65 EUR | 499 EUR (vor Ort) | 89 EUR |
| Command | 80-200+ | Synology RS1221+ ab ~5.500 EUR | ab ~150 EUR | 1.490 EUR (vor Ort) | 199 EUR |

### ORBIT-Rabatt
20% auf alle COSMI-Modulpreise (Grund: keine Cloud-Infrastruktur von Zentria).

### Meeting-Optionen
- **Cloud (Standard):** Jitsi auf Hetzner VPS, im Modulpreis enthalten (bis 10 Teiln.)
- **Lokal:** Jitsi auf dediziertem Mini-PC vor Ort (600-2.800 EUR Hardware)
- **Hybrid:** Intern lokal, extern Cloud

### Cloud-Backup
| Paket | Hetzner-Kosten | Zentria-Preis |
|-------|----------------|---------------|
| S (bis 1 TB) | 3,20 EUR | 9 EUR |
| M (bis 5 TB) | 9 EUR | 19 EUR |
| L (bis 20 TB) | 40 EUR | 59 EUR |

### ORBIT-Zielgruppen
- Aerzte, Anwaelte, Steuerberater, Behoerden (Datenschutz)
- Laendliche Standorte / schlechtes Internet
- Betriebe mit eigener IT-Infrastruktur

---

## Beispielrechnung: Handwerksbetrieb (20 User, COSMI Cloud)

| Posten | Betrag |
|--------|--------|
| Module (Chat 20x, Meetings 8x, CRM 5x, Zeit 20x, Buchhaltung 3x, Schichten 20x) | 300 EUR |
| Volumen-Rabatt 20 User (5%) | -15 EUR |
| Support Professional (10% von 285 EUR) | 28,50 EUR |
| **Gesamt/Monat (zzgl. MwSt.)** | **313,50 EUR** |

Vergleich: Pipedrive + Slack + Clockify + Asana + Zoom separat = ~1.060 EUR/Monat fuer 20 User.

## Beispielrechnung: Handwerksbetrieb (30 User, ORBIT Station)

| Posten | Betrag |
|--------|--------|
| Hardware Synology DS1522+ (einmalig) | 2.200 EUR |
| Setup vor Ort (einmalig) | 499 EUR |
| Wartung ORBIT Station | 89 EUR/Mo |
| Cloud Backup M (5 TB) | 19 EUR/Mo |
| COSMI-Lizenz (Handwerk-Paket, -20%) | ~624 EUR/Mo |
| Volumen-Rabatt 30 User (10%) | -62 EUR/Mo |
| Support Professional (10%) | 67 EUR/Mo |
| **Monatlich gesamt (zzgl. MwSt.)** | **~737 EUR/Mo** |

---

## Code-Referenz

Die TypeScript-Datenstrukturen fuer Studio/Billing liegen in den Original-Dokumenten:
- `COSMI Preiskonzept April2026.docx` → MODULES, VOLUME_DISCOUNTS, SUPPORT_TIERS, BRANCH_PACKAGES, calculatePrice()
- `ORBIT Preiskonzept April2026.docx` → ORBIT_TIERS, CLOUD_BACKUP_PLANS, MEETING_SETUPS, DATA_PROFILES, calculateOrbitPrice()

---

## Rollout-Tasks (Website)

### COSMI
- Preisseite: Altes Tier-Modell durch Modul-Logik ersetzen
- Studio: Modulpreise, Paket-Rabatt (15%), Volumen-Rabatt live berechnen
- Kernbotschaft: "Du zahlst nur fuer was dein Team wirklich nutzt"
- Vergleichs-Widget: Tool-Stack-Rechner auf Preisseite

### ORBIT
- Studio: COSMI/ORBIT-Weiche oben einbauen
- ORBIT-Wizard: 8 Schritte (Hardware, Datenprofil, Backup, Meeting, Module, Support, Zusammenfassung)
- ORBIT-Seite: Konkrete Preise statt "auf Anfrage"
- Kein direkter Kaufbutton — immer "Konfiguration anfragen"

## Verwandte Notes
- [[architektur]] — Service-Architektur
- [[integrationen]] — Bexio, DATEV (Billing-relevante Integrationen)
- [[design]] — Studio-UI
