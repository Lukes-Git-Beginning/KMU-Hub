# Review-Fäden — kontakte

> Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `kontakte` · **Branch:** `kontakte/finish-p7-p8` · **Reviewer (zugeteilt):** offen

---

## ⬜ P7/P8-Finish — FE-Lücken geschlossen (2026-06-09)

**Kontext:** Audit ergab, dass kontakte P7 (Einstellungen) + P8 (Finanzberatung) **UI-seitig bereits fertig + verdrahtet** waren (CrmSettingsPanel mit Pipeline/CustomFields/Tags/Segmente, 8-Abschnitt-Beratungsprotokoll-Editor, „Empfohlen von", Leads-Inbox). Geschlossen wurden nur die echten FE-Restlücken; das große Backend-Paket ist konsolidiert in `backend-gaps.md`.

### 1. Beratungsprotokoll → PDF/Export (rechtlich Pflicht)
**Pfad:** Kontakt öffnen → Tab „Beratungsprotokolle" → „Neues Protokoll" → Editor → Button **„Als PDF / Drucken"** → Vorschau-Overlay → `window.print()`.
- Neue Komponente `AdvisoryProtocolPrint.tsx`: print-gestylte **Geeignetheitserklärung** (Briefkopf aus Settings-Firmenname, alle 8 Abschnitte read-only, Produkt-Tabelle, SRI-Risikoklasse, Unterschriftslinien, rechtlicher Aushändigungshinweis). Isolierter `@media print`-Block (`#advisory-print`), keine PDF-Dependency.
- **Markt-/Rechts-Grundlage (recherchiert):** MiFID II / § 64 WpHG / FinVermV — Geeignetheitserklärung muss dem Privatkunden auf **dauerhaftem Datenträger vor Vertragsschluss** ausgehändigt werden. Quellen im Chat.
- ⚠ **Backend nötig:** revisionssichere unveränderliche Ablage (10 J.) — `window.print` erfüllt die Aufbewahrung NICHT (→ backend-gaps).
- Worauf achten: leere Abschnitte erscheinen bei neuem Entwurf nur als Überschrift (Row blendet leere Werte aus) — bei gefülltem Protokoll erscheinen die Werte.

### 2. Lead-Scoring-Regel-Editor (Markt-Parität)
**Pfad:** Modul-Einstellungen → CRM → Sektion „Lead-Scoring" (tenant-scope).
- `computeLeadScore`/`scoreToTemperature` lesen jetzt aus `useLeadScoringStore` statt hartkodiert (Defaults = altes Verhalten → kein Bruch). Editor: Basispunkte je Quelle, Punkte je Feld, Heiß/Warm-Schwellen, Live-Vorschau, Reset. Markt-Muster HubSpot/Pipedrive (recherchiert).
- **Nicht per Screenshot-QA** abgedeckt (liegt im Settings-Overlay, tiefe Navigation) — tsc grün + Code-Review; spiegelt das funktionierende `SegmentSettings`-Muster.

### 3. Manuelle Segment-Überschreibung A/B/C
**Pfad:** Kontakt öffnen → Segment-Badge (Header) klicken → Popover „Segment manuell setzen" (A/B/C / Automatisch).
- `useSegmentOverrideStore` pro Kontakt; effektives Segment = Override ?? regelbasiert. Badge zeigt „·manuell" bei Override; „Automatisch (X)" zeigt das berechnete Segment.

### 4. NewsletterPanel
- War unverdrahteter Mock-Stub (nur in i18n referenziert) → **entfernt**. Echtes Newsletter-Feature bräuchte Kampagnen-Backend (notiert).

**Verify:** `scripts/qa-kontakte-p7p8.mjs` grün (detailOpen, segPopover, editorOpen, pdfPreview, rawKeys [], pageErrors []) · Screenshots Segment-Override-Popover + Geeignetheitserklärungs-PDF angesehen · gescopter tsc.
**Bug gefangen (durch Screenshot-QA):** doppelter `Popover`-Import in ContactDetailPanel → Vite-Build-Fehler (weiße Seite) → behoben. Beleg, dass „kompiliert"-Annahme ohne echtes Rendern nicht reicht.

⚠ **tsc-Hinweis (Baseline, projektweit — keine neue Logik-Lücke):** Gescopter tsc zeigt TS2345 an `t(dynamischer-key)`-Aufrufen quer durch **pre-existing** kontakte-Dateien (AdvisoryProtocolEditor/SelectField, AdvisoryProtocolsTab, CrmPersonalPrefs, CustomFieldRow/Manager) — mein `AdvisoryProtocolPrint` nutzt dasselbe etablierte `t(o.labelKey)`-Muster. Das ist die bekannte i18next-typed-keys-Strenge (siehe reviews/calendar.md — saubere Lösung wäre ein projektweiter typsicherer `t()`-Wrapper). Zusätzlich 1 **pre-existing** echter Typfehler in `PipelineStagesEditor.tsx:48` (`Property 'order'`, Stage[]-Mismatch) — **nicht von mir berührt**, nur über den Import-Graph (CrmSettingsPanel) sichtbar; sollte separat gefixt werden.

**Branch-Hinweis:** Eigener Branch (keine Marathon-Lane). Hot Files (i18n ×4, CrmSettingsPanel) nur additiv → beim Merge mit Marathon-Branches additiv auflösen.