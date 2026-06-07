# Kontakte P8 — Beratungsprotokoll: Feldspezifikation (Recherche 2026-06-07)

> Für den „Beratungsprotokoll"-Tab im Kontakt-Detail (Finanzberatungs-Tiefe). Darien: Variante 1 (Claude defaultet nach Recherche, später anpassbar). Rechtlich fundiert für DE.

## Rechtsrahmen (Kurz)
- Klassisches „Beratungsprotokoll" → seit 2018 (MiFID II) bei Anlageberatung durch **Geeignetheitserklärung** ersetzt (§64 WpHG Banken / §18 FinVermV freie Vermittler §34f GewO). Bei **Versicherung** (§61/62 VVG, IDD) heißt es weiter „Beratungsprotokoll".
- **ESG-/Nachhaltigkeitspräferenzen** seit 08/2022 Pflicht (EU-DelegVO 2021/1253).
- **Aufbewahrung:** praktisch **10 Jahre** (deckt WpHG 5-7 J. + FinVermV §18a 10 J. + zivilrechtl. Verjährung).
- **DSGVO:** Art. 6(1)(c) (gesetzliche Pflicht) trägt die Verarbeitung — **keine separate Einwilligung** für Pflichtfelder; Aushändigungspflicht (Kopie an Kunden); Löschung nach Fristablauf.

## Feldspezifikation (8 Abschnitte, P=Pflicht / O=optional)

**1. Gesprächskopf:** Beratungsdatum (P) · Uhrzeit von/bis (P) · Dauer min (P, auto) · Ort [Büro/Telefon/Video/beim Kunden] (P) · Berater (P) · Anlass [Erst/Folge/Anlassberatung + Freitext] (P) · Protokoll-ID (O, auto) · Kundenkategorie [Privat/Professionell] (P).

**2. Kunde & Profil:** Kunde (CRM-Verknüpfung) (P) · Geburtsdatum (O) · Familienstand (O) · Steuerstatus (O).

**3. Kenntnisse & Erfahrungen (§16 FinVermV):** Bekannte Anlagearten [Mehrfach: Aktien/Fonds/ETF/Anleihen/Derivate…] (P) · zurückliegende Transaktionen (Art/Häufigkeit/Zeitraum) (P) · Bildung finanzrelevant (P) · Berufserfahrung (P) · Selbsteinschätzung 1-5 (O).

**4. Finanzielle Situation:** Nettoeinkommen/Monat (P) · regelm. Verbindlichkeiten (P) · liquide Mittel (P) · Kapitalanlagen aktuell (P) · Immobilien (O) · bestehende Versicherungen/Vorsorge (O) · max. Verlusttragfähigkeit abs./% (P).

**5. Anlageziele & Risikoprofil:** Anlagezweck [Altersvorsorge/Liquidität/Wachstum/Sparen/Spekulation] (P) · Anlagehorizont [<1/1-3/3-5/5-10/>10 J.] (P) · Risikobereitschaft (P) · Risikotragfähigkeit objektiv (P) · **Risikoprofil/-klasse** (P) · ESG-Präferenz [Ja/Nein + SFDR/Taxonomie/PAI] (P) · Anlagebetrag einmalig (O) · Sparrate/Monat (O).
   - **Risikoklassen (PRIIP-SRI 1-7 ↔ Praxis):** 1-2 konservativ · 3-4 ausgewogen · 5 wachstumsorientiert · 6 dynamisch · 7 spekulativ.

**6. Besprochene Produkte (Liste):** Produktname (P) · ISIN/WKN (O) · Kategorie (O) · **Risikoklasse SRI 1-7** (P) · Chancen (O) · Risiken (P) · Kosten einmalig/laufend (P, ex-ante) · Empfohlen/Nur besprochen (O).

**7. Empfehlung & Geeignetheitsbegründung:** Empfehlungszusammenfassung (P) · Begründung der Geeignetheit (P) · Bezug zu Kundenzielen (P) · Bezug zur Risikoklasse (P, auto) · Alternativen (O) · nicht empfohlen + Grund (O).

**8. Abschluss & Compliance:** wesentliche Anliegen + Gewichtung (P) · Warnhinweise erteilt [Checkliste] (P) · Kunde hat Dokument erhalten + Datum (P) · Aushändigungsform [Papier/E-Mail/Portal] (P) · Unterschrift Berater (P) · Bestätigung Kunde (O) · Dokumentenverzicht (O, separate Erklärung) · Folgeberatung Datum (O) · interne Notizen (O).

## Bau-Hinweise
- Tab im ContactDetailPanel; pro Kontakt mehrere Protokolle (Historie, datiert).
- Risikoprofil-Klasse + SRI als shared-Enum (auch in CRM-Settings wiederverwendbar).
- Mock-first; backend-gap: `advisory_protocols` (contact_id, alle Felder, immutable nach Aushändigung, 10-J.-Retention) — siehe backend-gaps.
- „Empfohlen von"-Lookup + Mandanten-Segmente A/B/C (regelbasiert nach Umsatzpotenzial) gehören ebenfalls zu P8 (Darien Variante 1).

## Quellen
gesetze-im-internet.de (§16/§18 FinVermV, §83 WpHG) · §61 VVG dejure · BaFin MiFID-II-FAQ · fondsfinanz.de (Geeignetheitserklärung) · code-knacker.de (SRI 1-7) · accura-audit.de (FinVermV-Pflichtinhalte). (Volle URL-Liste im Recherche-Log.)
