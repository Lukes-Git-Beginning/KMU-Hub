# team — Lohn-Stammdaten am Mitarbeiterprofil: Recherche & Bau-Spec (2026-06-07)

> **Status: NUR SPEC — noch NICHT gebaut** (Darien-Vorgabe). Nächste Session (neues Terminal) startet hiermit.
> **Kontext:** Lohnvorbereitung bleibt im **team**-Modul (entschieden, markt-konform — siehe `team-datev-lohn-spec.md`). Dafür müssen die lohn-relevanten **Stammdaten** ans Mitarbeiterprofil. Aktuell hat `EmployeeProfile` (hr-types.ts) KEINE Lohnfelder.

## Leitprinzip
„Personalstammblatt" digital: alle Daten, die DATEV-Lohn / das Lohnbüro zur Anmeldung (SV) + korrekten Abrechnung braucht. Speist direkt die **Lohnvorbereitung** (PayrollPrepPanel) — Stammdaten-Spalte + Bezüge.

## Feldspezifikation (gruppiert nach DATEV-Themenbereichen; P=Pflicht)
DATEV gliedert Personalstammdaten in: Personaldaten · Steuer · Sozialversicherung · Beschäftigung · Bezüge/Bank. Spiegeln wir.

**1. Persönliche Daten** (teils schon in EmployeeProfile: Name, Anschrift)
- Geburtsdatum (P) · Geburtsort (P) · Geschlecht (P, m/w/d) · Staatsangehörigkeit (P) · Familienstand (P, ledig/verh./gesch./verw.)

**2. Steuer**
- Steuer-ID / IdNr (P, 11-stellig) · Steuerklasse (I–VI, via ELStAM — bei uns manuell erfassbar) · Kinderfreibeträge (Zahl, Komma) · Konfession (rk/ev/keine → Kirchensteuer)

**3. Sozialversicherung**
- SV-Nummer (P, 12-stellig) · Krankenkasse (P, Name + Betriebsnummer) · Elterneigenschaft (ja/nein → PV-Zuschlag) · RV-/SV-Status (pflichtig/freiwillig/privat/Minijob-pauschal)

**4. Beschäftigung** (Eintritt = `startDate` existiert)
- Eintrittsdatum (P, vorhanden) · Wochenarbeitszeit Std (P; aktuell nur `workDaysPerWeek`) · Beschäftigungsart (P: Vollzeit/Teilzeit/Minijob/Midijob/Werkstudent/Azubi) · Tätigkeitsschlüssel (P, 9-stellig) · Befristung/Austrittsdatum (O)

**5. Bezüge & Bankverbindung**
- Entlohnungsart (Festgehalt | Stundenlohn) · Bruttogehalt €/Monat ODER Stundenlohn €/h (P) · Sonderzahlungen (O: 13. Gehalt, Urlaubsgeld) · Abrechnungsgruppe (→ `payrollSettings.groups`) · IBAN (P) · BIC (O) · Kontoinhaber (O, falls abweichend)

## Strukturvorschlag (UI)
- Neue Sektion **„Lohn-Stammdaten / Abrechnungsdaten"** im `MemberDetailPanel`, **collapsible**, mit Sub-Gruppen (Steuer · Sozialversicherung · Beschäftigung · Bezüge & Bank). Anzeige + Inline-Bearbeiten.
- **DSGVO/RBAC:** sensible Daten → **hr_only**-Sichtbarkeit (wie Personalakte). Nicht für Manager/Self ohne Recht.
- Optional als Schritt im `CreateEmployeeWizard` (Erfassung am 1. Arbeitstag).
- **Speist Lohnvorbereitung:** PayrollPrepPanel-`buildRow` nutzt dann echte Felder (Beschäftigungsart→Gruppe, Festgehalt/Stundenlohn, Vollständigkeits-Check „fehlt Steuer-ID/SV-Nr" als Warnung vor Export).

## Bau-Plan (mock-first)
- `stores/payrollMasterData.ts` — Overlay-Store `Record<employeeId, PayrollMasterData>` (persist), get/set. (Backend: Felder gehören eigentlich an `EmployeeProfile` → Luke erweitert hr-types + API; FE-Overlay bis dahin.)
- `lib/payroll-enums.ts` — Steuerklasse, Konfession, Beschäftigungsart, SV-Status, Entlohnungsart (shared, i18n-keys).
- `modules/team/EmployeePayrollData.tsx` — Sektion (view+edit), eingebettet in MemberDetailPanel (hr_only).
- Vollständigkeits-Badge: „Lohn-Stammdaten unvollständig" wenn Pflichtfelder fehlen → in MemberDetail + als Plausi in PayrollPrepPanel.
- i18n ×4. backend-gaps: `EmployeeProfile`-Erweiterung um Lohnfelder (Steuer/SV/Bezüge/Bank) + DSGVO-Sicht.

## DSGVO-Hinweis
Lohn-Stammdaten = besondere Personaldaten. Zugriff nur HR/Lohn-Rolle, Audit-Log empfohlen (wie Personalakte). Aufbewahrung nach Austritt: Lohnunterlagen 6 J. (§41 EStG) / SV-Doku.

## Quellen
DATEV Hilfe-Center (Mitarbeiter anlegen in Lohn und Gehalt, Personalstammdaten Themenbereiche Steuer/Beschäftigung/SV) · lohndialog.de (Personalstammblatt 2026 — vollständige Feldliste) · personio.de/hr-lexikon (Lohnbuchhaltung vorbereiten) · hrlab.de (vorbereitende Lohnabrechnung) · quick-lohn/taxpal Checklisten (Neueinstellung Pflichtangaben: Steuer-ID, SV-Nr, Krankenkasse, IBAN).
