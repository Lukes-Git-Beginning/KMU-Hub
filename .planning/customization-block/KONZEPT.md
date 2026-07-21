# Self-Service-Customization („Anpassungen") — KONZEPT (SSOT)

> **Status:** Konzept festgeschrieben nach Recherche-Gate (Session #25, 2026-07-21). Recherche liegt in `IST-A/B/C.md` (Codebase-Ist) + `MARKT-A/B/C.md` (Salesforce/ServiceNow/Odoo/Zoho/HubSpot/Power Platform/Airtable/Monday/Budibase/Retool/Appsmith). Vision + Auftrag: `RECHERCHE-BRIEFING.md` §0/§1.
>
> **Ein-Satz-Vision (Darien):** Der Kunde passt Cosmi selbst per No-Code IN Cosmi an (Felder, Begriffe, Wertelisten) — und Zentria richtet mit **exakt derselben Fläche** beim Onboarding ein. Ein Tool, zwei Nutzer. Das RBAC-System (R-1…R-6) ist die Blaupause.

---

## §0 Entscheidungen (verbindlich — Darien, 2026-07-21)

Aus dem Gate nach der Recherche (4 Fragen bestätigt + 2 gesetzt):

1. **v1-Scope = Fundament-Trio** — die erste Ausbaustufe umfasst genau drei Dimensionen: **(A) Custom Fields** (Felder zu bestehenden Entitäten), **(B) Terminologie/Labels** (Umbenennen, z. B. „Kontakte" → „Patienten"), **(M) Wertelisten/Status-Sets** (Deal-Phasen, Ticket-Prioritäten, Projekt-Status). Alle drei Overlay-basiert, review-reif geschnitten. Custom Objects (G), komplexe Workflows (F), Feld-Layouts (C) = spätere Stufen bzw. WASM-Nähe.
2. **Ein Werkzeug, Progressive Disclosure** — kein getrennter Einfach-/Experten-Modus. Eine Fläche in Schichten: Basis (Felder/Labels/Werte) sofort sichtbar, Fortgeschrittenes (Validierung, bedingte Logik) aufklappbar. Marktstandard (Zoho/Airtable), kein Drift, ein Aufbau.
3. **Zwei Overlay-Schichten Vendor → Tenant** — Zentria-Onboarding-Config = Basis-Layer (`vendor`), Kunden-Änderungen = zweiter Layer (`tenant`) obendrauf. Kein Update berührt beide. Die UI zeigt die **Herkunft** („von Zentria eingerichtet" vs. „selbst geändert") — dasselbe geerbt/angepasst-Muster wie R-6.
4. **Zentrale Fläche + Modul-Schnellzugriffe** — ein eigenes Admin-Modul **„Anpassungen"** (Gesamt-Übersicht + mächtige Editoren) PLUS In-Context-Schnellzugriffe in den bestehenden Modul-Einstellungen (`ModuleSettingsShell`). Kombiniert Salesforce-Übersicht + Odoo-In-Place. **Arbeitsname: „Anpassungen"** (final beim Bau bestätigen).
5. **[GESETZT] Update-Sicherheit = Overlay-only** — Config speichert nur **Abweichungen** vom Code-Default (sparse), in getrennten Tabellen, Runtime merged. Ein Produkt-Update ändert Kunden-Config nie. Einstimmige Recherche + identisch zum R-6-Muster (`USER_OVERRIDES` über `ROLE_DEFS`).
6. **[GESETZT] Branchen-Vorlagen von Anfang an mitgedacht** — Template-first (nie leerer Screen). Vorbild: `business-profiles` + `INDUSTRY_ROLE_TEMPLATES`. v1 startet mit 1–2 Beispiel-Vorlagen als Proof, Vollausbau folgt.

---

## §1 Architektur — Overlay-Config-Layer

**Kern-Prinzip (aus Markt-Konvergenz + R-6):** Code liefert den Default. Config ist ein **Overlay**, das nur Abweichungen speichert. Zur Laufzeit wird gemerged:

```
effektiv = code_default  ⊕  vendor_overlay (Zentria)  ⊕  tenant_overlay (Kunde)
                                                          ↑ gewinnt pro Schlüssel
```

- **Zwei Schichten, drei Provenienzen:** `default` (Code) · `vendor` (Zentria-Onboarding) · `tenant` (Kunde). Analog R-6: `inherited` / `override(vendor)` / `override(tenant)`. Provenance wird mitgeführt, damit die UI die Herkunft anzeigt und „auf Zentria-Stand zurücksetzen" bzw. „auf Cosmi-Default zurücksetzen" möglich ist.
- **Speicher (Luke-Paket, BE):** JSONB-Config-Tabellen, tenant-scoped + RLS, pro Config-Domäne. v1-Domänen: `custom_field_definitions` (existiert teilweise!), `tenant_label_overrides` (neu), `tenant_value_sets` (neu). Jede Zeile trägt `layer` (`vendor`|`tenant`), `tenant_id`, `updated_by`, Timestamps. Sparse — nur gesetzte Schlüssel.
- **Resolver (FE zuerst, Mock-first):** ein reiner `resolveConfig(domain)`-Layer wie `applyUserOverrides()` — Code-Default → vendor → tenant. Liefert Wert + Provenance. Keine Kopien pro Modul (Anti-Muster „Global Value Sets").
- **Audit + Preview + Rollback (R-5/R-2-Infra wiederverwenden):** jede Config-Änderung feuert ein Audit-Event (`customization.*`, siehe §4) über `writeAuditEvent`. Vorschau vor Live via `startPreview`-Muster (Zoho hat's, Odoo nicht = USP). Rollback = Overlay-Schlüssel entfernen (fällt auf darunterliegende Schicht zurück).
- **No-Code ≠ No-Safety:** Kunden-Eingaben validieren, Feld-Löschung nur mit Konsequenz-Dialog + Soft-Delete (Airtable-Anti-Muster vermeiden). Keine frei ausführbaren Formeln in v1.

**Wiederverwendbare Bausteine (aus Ist-Analyse):**
| Baustein | Herkunft | Nutzung im Tool |
|---|---|---|
| `ModuleSettingsShell` + `module-settings-registry` + `useModuleSettings` | bestehend, 25 Module, personal/tenant-Scope, JSONB-Persistenz | **zentraler Anker** — Config-Sektionen als reguläre `ModuleSettingsSection`; Modul-Schnellzugriffe |
| RBAC Zwei-Pane-Editor (`UserOverrideEditorPage`), `startPreview`, Staged-Footer, Guardrail-Dialog, TRI-STATE-Rows | R-1…R-6 | Editor-UX-Blaupause für alle Config-Editoren (Overlay-Semantik identisch) |
| `audit-events.ts` (`writeAuditEvent` + Interceptoren) | R-5 | Config-Audit-Trail (`setting.changed` ist reserviert, wird jetzt genutzt) |
| `business-profiles` + `INDUSTRY_ROLE_TEMPLATES` + `orderedSetsForProfile()` | bestehend | Config-Vorlagen-Galerie (Template-first) |
| `work_custom_field_definitions` (9 Typen, RLS, Migration 000146/147) + CRM `custom_field_definitions` (000005, 6 Typen) | BE fertig | **Custom-Fields-Fundament** — v1 vereinheitlicht die FE darauf |
| i18next `addResourceBundle(locale,'translation',overrides,true,true)` | Bibliothek | Runtime-Label-Overlay ohne Rebuild |

---

## §2 v1 — Fundament-Trio im Detail

### A · Custom Fields
- **Ist:** BE fertig für **work-Tasks** (`work_custom_field_definitions`, 9 Typen: text/number/date/boolean/select/multi_select/url/email/phone, RLS-isoliert, RBAC-Seeds 000147) und **CRM** (`custom_field_definitions`, 6 Typen, contact/company/deal/activity). FE-Manager-UIs existieren, sind aber teils an einen Zustand-Store statt an die API gehängt, und die work-UI kennt nur 5 der 9 Typen. `CustomFieldsSection.tsx` ruft die falsche (alte CRM-) API.
- **v1-Auftrag:** **Vereinheitlichen, nicht neu bauen.** Ein generischer, konsistenter Custom-Field-Manager (alle Feldtypen, an die korrekte API, Overlay-/Provenance-fähig), über die vorhandenen BE-Entitäten (work-Tasks + CRM contact/company/deal/activity). Progressive Disclosure: Basis = Feld anlegen (Name/Typ/Pflicht); Fortgeschritten = Validierung/Default/Sichtbarkeit. **Neue Entitäten (Custom Objects G) ausdrücklich NICHT in v1.**
- **Backend:** existiert für den v1-Umfang → geringes Luke-Paket (v. a. FE-Verdrahtung + Typ-Parität + generische Entitäts-Auflösung).

### B · Terminologie / Labels
- **Ist:** i18n = 7.221 Keys × 4 Sprachen, statisch gebundelt, **kein Runtime-Override möglich**. TS-Typen build-zeit-gebunden.
- **v1-Auftrag:** Tenant-Label-Overlay. Neu: Tabelle `tenant_label_overrides (tenant_id, layer, locale, key, value)` + GET-Endpoint nach Login + `addResourceBundle`-Merge sobald Tenant/Sprache bekannt + Admin-UI. **Kuratierte Whitelist (~50–100 Keys):** Modul-Namen, Objekt-Bezeichnungen (Kontakt/Deal/Ticket/Projekt), Status-Labels — NICHT alle 7.221 Keys (Wartungs-/Konsistenz-Falle). Preview vor Live.
- **Backend:** neu (kleine Tabelle + 1 Endpoint). FE: Merge-Hook + Whitelist-Editor.

### M · Wertelisten / Status-Sets
- **Ist:** modul-spezifische Wertelisten (Deal-Phasen, Ticket-Prioritäten, Projekt-Status) heute überwiegend hardcodiert (Ist-Analyse I-A verifiziert Einzelfälle beim Bau).
- **v1-Auftrag:** zentrale **Global Value Sets** — eine Config-Domäne `tenant_value_sets`, in der die wichtigsten Wertelisten definiert werden (Label, Reihenfolge, Farbe, aktiv/inaktiv), Module referenzieren sie über einen Resolver (keine Kopien). Umbenennen/Umsortieren/Ergänzen im Editor, Soft-Delete für benutzte Werte.
- **Backend:** neu (Config-Tabelle + Resolver). FE: Value-Set-Editor + Referenz-Auflösung in den betroffenen Modulen.

---

## §3 Roadmap — Ausbaustufen (rollierend, review-reif pro Stufe)

| Stufe | Inhalt | Warum diese Reihenfolge |
|---|---|---|
| **v1** | Overlay-Infrastruktur (Layer/Resolver/Audit/Preview) + **Custom Fields** (vereinheitlicht) + **Label-Overrides** (Whitelist) + **Value-Sets** + „Anpassungen"-Hub + Modul-Schnellzugriffe + Vendor/Tenant-Herkunft | Höchste Wirkung, meistes BE-Fundament vorhanden, etabliert das Overlay-Muster als Basis für alles Weitere |
| **v2** | Listen-/Spalten-Konfiguration (D) + Feld-/Ansichts-Layouts (C) + Modul-An/Aus pro Tenant (H, braucht Feature-Flag-Migration weg von Env-Vars) + Branding echt persistieren (K) | Baut auf Custom-Fields + Value-Sets auf (Spalten zeigen Custom-Felder); Feature-Flag-Migration ist Luke-Vorlauf |
| **v3** | Workflow-/Automatisierung-Erweiterung (F, begrenzt — kein Salesforce-Wildwuchs) + Dokument-/Report-Vorlagen mit Merge-Fields (J) + Benachrichtigungs-Regeln (N) | Höhere Komplexität, teils WASM-Grenze; nach etablierter Config-Basis |
| **später/WASM** | Custom Objects / neue Entitätstypen (G) | Datenmodell-Erweiterung zur Laufzeit = größter Brocken, WASM-Phase-D-Nähe |

---

## §4 RBAC- & Governance-Verzahnung

- **Neuer Katalog-Key `admin:customization:manage`** — wer die Anpassungen bearbeiten darf (admin/it_admin; delegierbar). Breiter als `settings:tenant:manage`, damit „größere Unternehmen mit eigener IT" es an IT-Rollen geben können.
- **Zentria schreibt den `vendor`-Layer** über den GDAP-Vendor-Zugang aus R-5 (`security:vendor_access`): ist eine Zentria-Vendor-Session aktiv, landen Änderungen im `vendor`-Layer (Provenance), sonst im `tenant`-Layer. **Kein Schattenweg** — gleiche Fläche, nur andere Ziel-Schicht.
- **Audit:** neue Event-Typen `customization.field_*` / `customization.label_*` / `customization.valueset_*` über `writeAuditEvent` (R-5-Infra). `setting.changed` (reserviert, nie gefeuert) wird jetzt aktiviert.
- **Preview:** `startPreview`-Muster — „Vorschau als Rolle/Benutzer" vor Publish (USP ggü. Odoo).
- **Update-Sicherheit:** Overlay-only (§0.5). Bei Feature-Entfernung im Code → Migrations-Script prüft verwaiste Overlay-Keys (einziger echter Test-Fall, ServiceNow-Erkenntnis).
- **ADR nötig:** Config-Speicher-Architektur (Overlay-Layer, Vendor/Tenant-Schichtung, Resolution-Reihenfolge) als ADR in `docs/ARCHITECTURE.md` festhalten.

---

## §5 Bau-Plan v1 (RBAC-Muster: Fundament selbst → Editoren via Agents → Gates)

Alles **Mock-first** (MSW-Overlay-Persistenz), Backend gebündelt als Luke-Paket (feedback: FE→BE-Wiring als eigene Phase).

- **v1.0 Fundament (selbst):** Overlay-Config-Typen + `resolveConfig()`-Resolver (default→vendor→tenant, Provenance) + MSW-Persistenz + Audit-Verzahnung (`customization.*` + `setting.changed`) + RBAC-Key `admin:customization:manage` + i18n-Basis. Kein UI-Editor, nur der Unterbau + ein Smoke-Screen.
- **v1.1 Custom-Fields-Editor (Agent):** generische, vereinheitlichte UI, an korrekte API, alle Feldtypen, Progressive Disclosure, Provenance-Anzeige.
- **v1.2 Label-Override-Editor (Agent):** Whitelist-Keys, `addResourceBundle`-Merge + Live-Preview, Vendor/Tenant-Herkunft.
- **v1.3 Value-Sets-Editor (Agent):** zentrale Wertelisten, Referenz-Auflösung in ≥1 Modul als Proof, Soft-Delete.
- **v1.4 „Anpassungen"-Hub-Shell (selbst/Agent):** Admin-Modul-Fläche (Übersicht + die 3 Editoren) + Modul-Schnellzugriffe in `ModuleSettingsShell` + Template-Galerie-Stub + Vendor/Tenant-Herkunfts-Banner.
- **Gates pro Stufe (verbindlich):** i18n ×4 (`{var}`, ICU-Plural) · scoped tsc (nur geänderte Dateien) · `eslint src/ --quiet` · Playwright-Screenshot-QA **+ Bilder ansehen** · ein Commit + Push pro Stufe.

## §6 Luke-Paket-Vorschau (backend-gaps §Customization)

- Config-Speicher-Tabellen (JSONB, tenant-scoped + RLS, `layer` vendor/tenant): `tenant_label_overrides`, `tenant_value_sets`; Custom-Fields-BE existiert (nur generalisieren/verdrahten).
- Overlay-Resolution serverseitig (default→vendor→tenant) + Provenance.
- Label-Override-GET-Endpoint (nach Login, pro Tenant+Locale).
- Value-Set-Referenz-Auflösung in den betroffenen Modul-APIs.
- Audit-Write für `customization.*` als Middleware (R-5-Muster).
- Update-Migration: verwaiste Overlay-Keys bei Code-Feature-Entfernung.
- **v2-Vorlauf:** `tenant_feature_flags`-Tabelle (Modul-An/Aus weg von Env-Vars), Branding-Persistenz (BE-Write + Asset-Upload + systemweite `--brand-accent`-Konsumption).

## §7 Offene Punkte / vor dem Bau klären
- Finaler Modul-Name („Anpassungen" vs. „Studio" vs. „Konfiguration") — Arbeitsname „Anpassungen".
- Genaue Whitelist der Label-Keys (~50–100) — beim Bau von v1.2 kuratieren.
- Welche Wertelisten in v1 (Deal-Phasen / Ticket-Prioritäten / Projekt-Status als Startset) — beim Bau von v1.3 aus dem Ist verifizieren.

## §8 Nachgelagert
Passwort-Manager + weitere Funktions-Ideen (Darien+Luke) = eigenes Paket NACH diesem Block (`project_password_manager`). Onboarding (O-0) = geführter Config-Assistent auf diesem Unterbau.
