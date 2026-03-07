# Roadmap — Produkt-Strategie & Software-Analyse

## Milestone 1: Strategische Grundlage

### Phase 1: Software-Landschaft DACH
**Ziel:** Komplette Uebersicht welche Software DACH-KMUs nutzen
**Status:** DONE (2026-02-17)
**Output:** `research/01-05` (Office, CRM, Finanzen, PM/ERP, DSGVO) + `external/chatgpt-research-summary.md`
- 13 Kategorien recherchiert (Office, E-Mail, CRM, Buchhaltung, HR, PM, Chat, Helpdesk, DMS, Zeit, ERP, Branche, Storage)
- Pro Tool: Funktionen, Preise, Marktanteil, DSGVO-Status

### Phase 2: Funktions-Analyse (Build vs. Integrate)
**Ziel:** Pro Kategorie entscheiden was wir selbst bauen vs. integrieren
**Status:** DONE (2026-02-17)
**Output:** `research/00-SYNTHESE.md` (Build/Integrate Matrix)
- Build: CRM, Chat, Video, Rechnungen, Wiki, Helpdesk, alle Branchenmodule
- Integrate: Buchhaltung (DATEV), Lohn (nie), Newsletter (Brevo), E-Signatur (Skribble), Banking (FinAPI)

### Phase 3: Bestandsaufnahme KMU Hub
**Ziel:** Gap-Analyse — was haben wir, was fehlt?
**Status:** DONE (2026-02-17)
**Output:** `research/06-modul-gap-analyse.md`
- 25+ Module mit Funktionstiefe erfasst
- Top 20 Feature-Luecken priorisiert

### Phase 4: DSGVO/DSG Compliance-Framework
**Ziel:** Rechtliche Grundlage fuer alle Entscheidungen
**Status:** DONE (2026-02-17)
**Output:** `research/07-compliance-framework.md`
- DE (DSGVO, GoBD), CH (nDSG, OR, MWSt)
- Hosting, Verschluesselung, Audit, Loeschkonzept

### Phase 5: Datenbankmodelle
**Ziel:** DB-Schema fuer alle Module definieren/ergaenzen
**Status:** DONE (2026-02-17)
**Output:** `research/08-datenbankmodelle.md`
- 30 neue PostgreSQL-Tabellen definiert
- Multi-Tenancy, Relationen zwischen Modulen

### Phase 6: Infrastruktur-Matrix
**Ziel:** Setup-Empfehlungen nach Groesse und Branche
**Status:** DONE (2026-02-17)
**Output:** `research/09-infrastruktur-matrix.md`
- Klein (5-20 MA), Mittel (20-100), Gross (100-200)
- Self-hosted vs. SaaS, Kosten pro Tier

### Phase 7: Produkt-Roadmap & Handoff
**Ziel:** Alles zusammenfuehren, umsetzbar machen
**Status:** DONE (2026-02-17)
**Output:** `docs/PRODUCT-STRATEGY.md`, `research/PROMPT-FUER-LUKE.md`, `research/11-backend-plan-part1+2.md`, `research/14-frontend-plan.md`
- Priorisierte Umsetzungs-Roadmap
- Handoff-Dokument fuer Luke (Backend)
- Frontend-Implementierungsplan fuer Darien
- Preismodell: 12/19/25 EUR pro User/Mo
