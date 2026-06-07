# Finanzen / Buchhaltung — Symbiose-Strategie (Recherche 2026-06-07)

> **Darien-Vorgabe:** Buchhaltungs-/Finanzmodul **behalten, nicht löschen**. Strategie = Symbiose aus eigenständigen Funktionen + Anbindung an etablierte Software, die KMUs nicht ersetzen wollen (DATEV/Lexware/Bexio). Kein Voll-Ersatz.
> **Hinweis zur Namensverwirrung:** Das Modul, das der Nutzer als „Buchhaltung" in der Sidebar sieht, ist technisch `modules/finanzen/` (Route `/finanzen`, i18n `layout.navItems.finance` = „Buchhaltung"). Der Ordner `modules/buchhaltung/` ist alter, nie gemounteter Code (`_DEPRECATED.md`, eigener Zustand-Mock, keine Route) — **bleibt vorerst unangetastet** (Darien will nichts verlieren). Erst nach Klärung/Migration der dort skizzierten Tabs anfassen.

## Leitprinzip
**Cosmi ist kein Buchhaltungsersatz.** Cosmi macht die **vorbereitende Finanzkette vollständig** (Angebot → AB → Rechnung → Mahnwesen → Zahlungseingang) und **übergibt sauber an den Steuerberater** (DATEV/Bexio). Verständlich für KMU, willkommen bei Steuerberatern (keine Konkurrenz), einfach im Sales.

## Marktrealität DACH
- **DE:** DATEV ~80% Steuerberater-Marktanteil (Genossenschaft, SKR03/04-Standard, ELSTER/GoBD-Integration, Haftung) → **praktisch nicht ersetzbar**. Selbstbucher: lexoffice / sevDesk / Lexware (alle mit DATEV-Export).
- **AT:** BMD (Marktführer StB), RZL. Selbstbucher: everbill, FreeFinance.
- **CH:** **Bexio** dominiert KMU (90.000+ Nutzer, Cloud, Treuhand-Portal); Abacus im Mittelstand. Kein DATEV-Äquivalent.

## Entscheidungstabelle
| Eigenständig (Cosmi baut) | Per Integration/Export | Bewusst NICHT (Steuerberater) |
|---|---|---|
| Angebot, Auftragsbestätigung | **DATEV EXTF-Export** (Buchungsstapel + Belege ZIP) | Kontierung/Buchungssätze (SKR03/04) |
| Rechnung/Faktura (Teil/Schluss) | **Bexio-API** (CH, OAuth) | USt-Voranmeldung (ELSTER) |
| **E-Rechnung** (ZUGFeRD + XRechnung) | BMD-Export (AT, CSV) | Jahresabschluss / Bilanz / GuV |
| Gutschrift/Storno | Lexware/lexoffice-Anbindung | Lohnbuchhaltung (LODAS) |
| Mahnwesen (mehrstufig) | Banking (HBCI/FinTS, finAPI) | Anlagenbuchhaltung / AfA |
| OP-Liste (offene Posten) | DATEV Unternehmen Online / DATEVconnect (später, Marktplatz-Akkr.) | KSt/GewSt-Erklärung |
| Zahlungseingang-Matching (CAMT/MT940-Import) | | ELSTER-Direktanbindung |
| **GoBD-Belegarchiv** (unveränderbar, 8 J.) | | |

## 3 Launch-kritische Integrationen
1. **DATEV EXTF-Export (DE)** — EXTF-Spec ist öffentlich, KEIN Marktplatz-Partner nötig. Berater-Nr. + Mandanten-Nr. + Sachkonto-Mapping in Settings. ~2-3 Wochen BE. **Ohne das ist Cosmi für DE-KMU mit Steuerberater unverkäuflich.**
2. **Bexio-API (CH)** — Developer-API (OAuth2), Rechnungen/Kontakte bidirektional. ~2-4 Wochen BE.
3. **E-Rechnung (ZUGFeRD 2.x + XRechnung, EN-16931)** — **Empfangspflicht DE seit 01.01.2025**, Sendepflicht gestaffelt bis 2027. GoBD-Belegarchiv ebenfalls Pflicht. → Launch-Blocker. (ZUGFeRD ist FE-seitig teils da, siehe Code QRRechnung/Belegkette.)

## Roadmap
- **Launch-Blocker:** GoBD-Belegarchiv · E-Rechnung Generierung (Ausgang) + Empfang/XML-Extraktion (Eingang).
- **Sprint 1:** DATEV EXTF-Export · OP-Liste + Mahnwesen vollständig · Zahlungs-Matching (CAMT.053/MT940-Import).
- **Sprint 2:** Bexio-Anbindung · BMD-Export · Lexware-Webhook · DATEV-Unternehmen-Online-Akkreditierung starten.
- **Sprint 3+:** finAPI/HBCI-Banking · DATEVconnect REST.

## Konsequenz für den Bau-Plan
- finanzen-Phasen NICHT mehr „buchhaltung löschen", sondern: **Symbiose-Features** (Faktura-Kette eigenständig) + **DATEV-Export/Bexio als Settings-Integration** (passt in den „Für alle"-Bereich der Modul-Einstellungen).
- `modules/buchhaltung/` (dead folder) erst anfassen, wenn Darien ok gibt + die dort skizzierten Tabs (Ausgaben-Approve, Transaktions-Journal, Berichte) nach finanzen migriert sind.

## Quellen
DATEV-Presse 2024 · weclapp/orgaMAX/sevDesk/Lexware DATEV-Export-Docs · BMF E-Rechnung FAQ · Bexio/BMD Marktdaten · GoBD-2025-Pflichten. (Volle URL-Liste im Recherche-Log der Session.)
