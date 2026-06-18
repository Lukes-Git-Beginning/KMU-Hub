# team — DATEV-Lohn / Lohnvorbereitung: Recherche & Bau-Spec (2026-06-07)

> Darien-Vorgabe: Es braucht **beides** — (1) die **Einstellungen** für die DATEV-(& Co.)-Einbindung in den Modul-Einstellungen UND (2) die **Arbeits-Oberfläche im Modul**, wo man tatsächlich mit der Lohn-Anbindung arbeitet. Recherchiert, wie der Workflow in der Praxis abläuft.

## Leitprinzip (Symbiose, analog Buchhaltung)
Cosmi macht die **vorbereitende Lohnabrechnung** (Stamm- + Bewegungsdaten sammeln, prüfen, als DATEV-Format übergeben) und reicht an **DATEV / Lohnbüro / Steuerberater** weiter, die die eigentliche Abrechnung rechnen. Cosmi ist **kein** Lohnabrechnungs-Ersatz (keine LStB/SV-Berechnung, keine ELStAM-Direktanbindung — das macht DATEV).

## Marktrealität (DE-MVP)
- **DATEV** dominiert; zwei Zielsysteme: **DATEV LODAS** und **DATEV Lohn und Gehalt**. Übergabe per **Datei-Export** (CSV/ASCII) ODER **Lohnimportdatenservice** (DATEVconnect, später, Akkreditierung).
- Rückkanal: **Lohnauswertungsdatenservice** (Lohnabrechnungen/Auswertungen zurück nach Cosmi) — Phase 2.
- Konkurrenz (Personio, HRlab, kenjo, askDANTE, e2n, clockin) macht es alle gleich: vorbereitende Lohnbuchhaltung + DATEV-Schnittstelle als Standard.
- AT/CH später (BMD / Swissdec).

## Zwei Datenklassen (zentral)
- **Stammdaten** (selten geändert): Steuerklasse, Krankenkasse, SV-Nr, Eintritt/Austritt, Festgehalt, Bankverbindung, Standort/Abteilung. → bei Änderung als „Stammdaten-Änderung" melden.
- **Bewegungsdaten** (monatlich): geleistete Stunden, Überstunden, Zuschläge, Abwesenheiten (Urlaub/Krank), Einmal-/wiederkehrende Bezüge (Boni, Sachbezüge), Minijob-Mindestlohn-Doku.

---

## TEIL 1 — EINSTELLUNGEN (Modul-Einstellungen → „Für alle", HR-Leiter)
1. **Verbindung:** Beraternummer + Mandantennummer (DATEV-Ordnungsbegriff, „vom Lohnbüro") · Zielsystem [LODAS | Lohn und Gehalt] · Übergabeart [Datei-Export | Datenservice (später)].
2. **Lohnarten-Zuordnung:** Mapping Cosmi-Kategorie → DATEV-Lohnart-Nr. (Festgehalt, Stundenlohn, Überstunden, Zuschläge, Bonus, Sachbezug …). Vorschlagswerte, editierbar („vom Lohnbüro bestätigen lassen").
3. **Abwesenheits-Schlüssel-Zuordnung:** Urlaub/Krank/Sonstige → DATEV-Abwesenheitsschlüssel; pro Abwesenheitsart Toggle „exportieren ja/nein".
4. **Abrechnungsgruppen:** z.B. Stundenlöhner vs. Festangestellte (unterschiedliche Stichtage/Datensätze).
5. **Zuordnungen:** Standort, Abteilung, Familienstand-Mapping.

## TEIL 2 — ARBEITS-OBERFLÄCHE (In-Page-Tab, z.B. „Lohnvorbereitung")
Monatlicher Zyklus (Lohnlauf):
1. **Abrechnungszeitraum wählen** (laufender Monat / Korrektur Vormonat) + Abrechnungsgruppe.
2. **Änderungsliste / Review** pro Mitarbeiter:
   - Stammdaten-Änderungen (neuer MA, Austritt, Gehaltsänderung, Steuerklasse …)
   - Bewegungsdaten (Stunden/Überstunden aus Zeiterfassung, Abwesenheiten aus team, Einmalbezüge)
   - Quellen: Zeiterfassung + Abwesenheiten + manuelle Einmalbezüge.
3. **Prüfen/Freigeben** (Plausibilitäts-Hinweise, Periode sperren = immutable für die Übergabe).
4. **Export generieren** (LODAS/Lohn&Gehalt-Datei mit Lohnarten + Abwesenheitsschlüssel) bzw. Datenservice-Übergabe.
5. **Status & Historie** (vergangene Lohnläufe, Datum, Anzahl MA, Status).
6. **(Phase 2) Lohnauswertungen importieren** (Abrechnungen zurück vom Lohnbüro).
7. **Lohnabzugs-Vorschau** (Brutto→Netto Schätzung DE/AT/CH) = bestehender HRIntegrationPanel-Teil, als Hilfs-/Plausi-Ansicht behalten.

## Bau-Hinweise / Symbiose mit bestehendem Code
- Bestehend: `HRIntegrationPanel` (Integration-Cards + Lohnabzugs-Vorschau), `ModuleAssignmentTab` (bleibt In-Page — operativ), Mitarbeiter/Anträge schon TanStack.
- **Settings-Konsolidierung:** `TeamSettingsPanel` (ModuleSettingsShell) — Persönlich (Start-Tab, Ansicht) + Für-alle (TeamSettingsTab embedded: Abteilungen/Rollen/Urlaub/Arbeitszeit · **DATEV-Lohn-Verbindung & Mappings** = neue Sektion).
- **Working-Surface:** neuer In-Page-Tab „Lohnvorbereitung" (monatlicher Lohnlauf). Connection-Cards/Config → Settings; operativer Lohnlauf → bleibt/entsteht In-Page.
- Mock-first; backend-gap: Lohnlauf-Persistenz, DATEV-Datei-Generierung (LODAS/L&G), Datenservice-Akkreditierung — `backend-gaps.md`.

## Quellen
hrlab.de (vorbereitende Lohnabrechnung, DATEV-Schnittstelle) · personio.de/support (Monthly Payroll, Lohnimport-/Lohnauswertungsdatenservice, Export-Typen LODAS/L&G) · askdante.com/lohnanbindung (Bewegungsdaten, Lohnarten/Abwesenheitsschlüssel, Buchungsmonat) · DATEV Hilfe-Center (Ordnungsbegriff Berater-/Mandantennummer) · e2n/clockin (Schnittstellen-Konfig: Lohnarten-Vorschläge, Abwesenheits-Konfig).
