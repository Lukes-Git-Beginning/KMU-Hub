# In-Cosmi-Anpassbarkeit (No-Code Self-Service-Konfiguration) — Recherche- & Analyse-Briefing

> **Status: VORBEREITET (Session #24, 2026-07-21) für ein FRISCHES Terminal.** Dieses Dokument setzt eine gründliche Ist-Analyse + Markt-Recherche auf, damit Darien und Claude anschließend gemeinsam den Umsetzungs-Umfang festlegen. **NICHT bauen bevor Recherche-Gate + gebündelte Fragen + Darien-Entscheide durch sind** (exakt das RBAC-Muster: Recherche → gebündelte Fragen → OK → Plan → bauen).
>
> **Startpunkt neues Terminal:** `git pull` → dieses Dokument lesen → Ist-Analyse-Agents + Markt-Recherche-Agents (§2 + §4) parallel starten → Ergebnis in `IST-ANALYSE.md` + `MARKT.md` → gebündelte Fragen aus §6 an Darien → §0 festschreiben → Bau-Plan.

## §0 Vision & Auftrag (Dariens Worte, 2026-07-21)

> „Wir passen die Software individuell an für die Leute. Da müssen wir uns nochmal genauer Gedanken machen: **was können wir alles anpassen, wie passt man es an**. Das Wichtige, damit wir die **Eigenständigkeit** aufrechterhalten können: das muss **in Cosmi** sein, das genauso IT oder Admins steuern können wie das mit den Funktionen und Rechten [= RBAC] gerade. Das Ziel ist, dass auch **größere Unternehmen mit eigener IT** sich darum kümmern und diese Anpassungen später **selbst machen** können — aber das geht ja nicht, indem sie coden. Also wäre das wichtig, gut und umfangreich zu implementieren."

**Kern-Prinzip (verbindlich):** Alles, was wir (Zentria) beim 1-Woche-Onsite-Onboarding an einem Kunden konfigurieren, muss der Kunde später **selbst über eine No-Code-Oberfläche IN Cosmi** ändern können. Kein Code, kein Zentria-Zwang. Das RBAC-System (R-1…R-6, gerade fertig) ist die **Blaupause**: In-Cosmi, No-Code, admin/IT-steuerbar, mit Audit + Guardrails + Preview.

**★ EIN TOOL FÜR BEIDE NUTZUNGEN (Darien 2026-07-21, verbindliche Design-Direktive):** Dieses Konfigurations-Tool ist NICHT nur die Kunden-Selbstbedienung — es ist ZUGLEICH unser eigenes Massanfertigungs-Werkzeug. Wenn wir es ordentlich bauen, richten WIR (Zentria) den Kunden beim Onboarding damit ein und der Kunde führt es danach mit exakt derselben Fläche selbst fort. Konsequenzen für den Bau:
- **Kein separates Zentria-internes Onboarding-Toolkit bauen** — das hier IST es. Ein Aufbau statt zwei.
- **Von Tag 1 mächtig genug für unser eigenes Onboarding** (nicht nur eine abgespeckte Kunden-Version) — der Umfang bemisst sich an dem, was wir heute beim Onsite manuell/per Config machen.
- **Dogfooding validiert sich selbst:** Wir nutzen es täglich beim Einrichten → Lücken fallen sofort auf, das Tool reift an echten Kunden-Setups.
- **Kein Drift Kunde↔Zentria:** gleiche Fläche, gleiche Ergebnisse, keine „Zentria kann mehr als im Tool sichtbar"-Schattenwege (Ausnahme nur echte Deploy/Env/Infra-Ebene, §6 Frage 3).
- Genau das RBAC-Muster: Zentria richtet Rollen beim Onboarding ein (INDUSTRY_ROLE_TEMPLATES), der Kunde ändert sie danach im selben Editor.

## §1 Strategische Einordnung (warum das der wichtigste Baustein nach RBAC ist)

- **Es IST der USP.** CLAUDE.md: „Massanfertigung durch 1-Woche-Onsite-Prozessanalyse + **Config**/WASM-Plugin-System." Die „Config"-Hälfte dieses USP ist genau dieses Thema. WASM-Plugins (Feature-Flag OFF bis Phase D) sind der Code-Weg für Extremfälle; die No-Code-Config ist der Selbstbedienungs-Weg für alle.
- **Datensouveränität + Eigenständigkeit** (Mission: „EU-Datensouveränität", Orbit-Self-Hosted): Ein Kunde, der bei jeder Feldänderung den Anbieter anrufen muss, ist nicht souverän. Selbst-Konfiguration ist die logische Fortsetzung von Self-Hosting.
- **Marktpositionierung gegen Salesforce/ServiceNow:** Deren No-Code-Customization (Object Manager, Flow, Studio) ist der Grund, warum große Firmen bleiben — sie bauen ihre Prozesse selbst nach. Für DACH-KMU + Mittelstand mit eigener IT ist das ein Kaufargument, das lexoffice/sevdesk/weclapp NICHT bieten.
- **Skaliert das Geschäftsmodell:** Je mehr der Kunde selbst kann, desto weniger Zentria-Onboarding-Stunden pro Kunde → mehr Kunden pro Team. Direkt umsatzrelevant.

## §2 Ist-Analyse-Auftrag (was schon konfigurierbar ist — vom neuen Terminal zu verifizieren)

Ziel: Vollständige Landkarte, WAS heute schon an Anpassung existiert, WO (Pfade), und über welche Fläche (In-Cosmi-UI vs. Code/Deploy/Env). Explore-Agents parallel (Pfade sind Anhaltspunkte, verifizieren!):

1. **Modul-Einstellungen** (`ModuleSettingsShell`, `personal` + `tenant`-Bereich pro Modul, `module-settings-registry.tsx`, `hooks/useModuleSettings.ts`): Die bestehende In-Cosmi-Konfigurationsfläche pro Modul. Was ist heute pro Modul einstellbar? Wie wird `tenant` vs. `personal` gescopt (Modul-Leiter/Admin via `settings:tenant:manage`)? Das ist der wahrscheinliche Anker für das ganze Feature.
2. **RBAC (R-1…R-6, FERTIG)** — die Blaupause. Editor-Muster (`RoleEditorPage`, `UserOverrideEditorPage`), In-Cosmi, Audit (`audit-events.ts`), Guardrails, Preview (`startPreview`), „aus Vorlage"-Galerie (`INDUSTRY_ROLE_TEMPLATES`). Wie viel davon ist als Muster für Config-Editoren wiederverwendbar?
3. **Feature-Flags / Modul-Aktivierung:** `featureflag/registry.go` (17 Flags, `COSMI_MODULE_*_ENABLED`, Deploy/Env-seitig!) + Admin-Lizenz-Fläche (`tenantModules`, A-3, `admin/license`). Unterschied: Env-Flags = Deploy-Zeit (nicht selbst-bedienbar), Lizenz-Toggle = In-Cosmi. Wo ist die Grenze, was muss verschoben werden?
4. **Business-Profiles** (`config/business-profiles.ts`: handwerk, gastronomie, einzelhandel, dienstleistung, it_tech, produktion, logistik, gesundheit, bau): steuern Modul-Sichtbarkeits-Vorauswahl. Branchen-Preset-Mechanik — Vorbild für „Config-Vorlagen"?
5. **Formulare-Modul** (`formulare`, Schemas/Submissions, BE-Seeds 000129): Existiert bereits ein Schema-basierter No-Code-Form-Builder? Wie mächtig (Feld-Typen, Validierung, mehrstufig)? Ist das die Keimzelle für Custom-Fields/Custom-Forms?
6. **Automatisierung-Modul** (`automatisierung`, `automations`, BE 000129): Trigger/Bedingungen/Aktionen? Wie weit reicht der No-Code-Workflow-Builder heute? (Salesforce-Flow-Äquivalent.)
7. **Dokument-Engine** (`shared/document`, Block-System, berichte umgestellt — Memory `project_document_engine`): No-Code-Vorlagen-Bau. Phase A done, B geplant. Relevanz für Report-/Dokument-Templates.
8. **Branding/Theme** (`admin/branding`, Design-System `.knowledge/design.md`, Themes): Logo/Farben — wie weit selbst-bedienbar heute?
9. **Terminologie/Labels/i18n** (`i18n/messages/*.json`, 4 Sprachen): Kann ein Kunde heute Begriffe umbenennen (z.B. „Kontakte" → „Patienten")? Vermutlich NEIN (i18n ist Code) — genau das ist ein Kernfall für No-Code-Overrides.
10. **Datenmodell** (`backend/migrations/*`, gRPC-Protos, `api/types.ts`): Können pro Tenant Felder/Entitäten ergänzt werden? Custom-Field-Infrastruktur vorhanden? (In der RBAC-Arbeit tauchte `work_custom_field_definitions` auf — backend-gaps: „Custom-Field-Definitionen (Task) BACKEND ERLEDIGT 2026-06-11, 9 Feldtypen". → EXISTIERT teilweise! Genau prüfen, wie weit.)
11. **Onboarding-System** (geplant, O-0, Memory `project_onboarding_system`): Überschneidung — Onboarding IST teilweise geführte Erst-Konfiguration.
12. **Integrationen** (`admin/integrations`, Bexio/Lexware/DATEV/LiveKit/OnlyOffice): Selbst-anbindbar?

**Ist-Analyse-Deliverable:** `IST-ANALYSE.md` — Tabelle je Anpassungs-Dimension (§3) × { existiert? · wo (Pfad) · In-Cosmi-UI oder Code/Deploy · wer darf (RBAC-Key) · Lücke }.

## §3 Anpassungs-Dimensionen (erste Taxonomie — im Gate mit Darien verfeinern/priorisieren)

Was „individuell anpassen" konkret umfassen KANN (Salesforce/ServiceNow/Odoo-Studio-Achsen, auf Cosmi gemünzt). Priorisierung + Scope-Schnitt ist eine Kern-Gate-Frage:

| # | Dimension | Beispiel | Schwierigkeit |
|---|---|---|---|
| A | **Custom Fields** — Felder zu bestehenden Entitäten hinzufügen | Kontakt bekommt Feld „Kundennummer ERP" | mittel (BE-Teil existiert für Tasks!) |
| B | **Terminologie / Labels umbenennen** | „Kontakte"→„Patienten", „Deals"→„Fälle" | mittel (i18n-Override-Schicht) |
| C | **Feld-/Ansichts-Layouts** — welche Felder, Reihenfolge, Sichtbarkeit, Pflicht | Detail-Ansicht pro Tenant umbauen | mittel-hoch |
| D | **Listen-/Spalten-Konfiguration** — welche Spalten, Default-Sortierung, Filter | Kontaktliste zeigt Custom-Feld als Spalte | mittel |
| E | **Formulare / Datenerfassung** — eigene Formulare, Feldtypen, Validierung | Aufnahmeformular je Branche | teils vorhanden (formulare) |
| F | **Workflows / Automatisierung** — Trigger→Bedingung→Aktion, Genehmigungsketten | „Neuer Deal > 10k → Chef benachrichtigen" | teils vorhanden (automatisierung) |
| G | **Custom Objects / Entitäten** — ganz neue Datentypen | Modul „Fahrzeug-Prüfungen" von Grund auf | HOCH (Datenmodell-Erweiterung) |
| H | **Module & Navigation** — an/aus, umbenennen, Reihenfolge, Icons | Modul ausblenden, Nav umsortieren | niedrig-mittel (teils da) |
| I | **Dashboards / Widgets** — welche Kacheln, Layout | Rollen-spezifische Startseite | teils vorhanden (dashboard) |
| J | **Dokument-/E-Mail-/Report-Vorlagen** | Angebots-Layout, E-Mail-Signatur | teils vorhanden (Dokument-Engine) |
| K | **Branding / Theme** | Logo, Farben | teils vorhanden |
| L | **Rollen & Rechte** = **RBAC (FERTIG)** | — | ✅ Referenz |
| M | **Pipelines / Status-Sets / Kategorien** — modul-spezifische Wertelisten | CRM-Deal-Phasen, Ticket-Prioritäten, Projekt-Status | mittel |
| N | **Benachrichtigungs-/Eskalations-Regeln** | wer wird wann informiert | mittel |

## §4 Markt-Recherche-Auftrag (No-Code-Customization-Plattformen)

Fokus: WIE machen die Marktführer No-Code-Selbstbedienungs-Konfiguration — Mechanik + UI + Governance. Was übertragbar, was Anti-Muster (zu komplex für KMU-IT)?

1. **Salesforce** (Gold-Standard, aber berüchtigt komplex): Object Manager (Custom Objects/Fields), Page Layouts + Lightning App Builder, Flow Builder (Workflows), Validation Rules, Custom Metadata Types, Picklist-Verwaltung. Fokus: Wie trennen sie „admin-configurable" von „developer"? Warum gilt es als überkomplex — was muss Cosmi einfacher machen?
2. **ServiceNow** (App Engine Studio, Form Designer, Flow Designer, UI Builder): Enterprise-No-Code. Governance/Sandbox/Update-Sicherheit.
3. **Odoo Studio** (DER direkte KMU-Vergleich): No-Code-Anpassung von Feldern/Views/Reports/Automatisierung ohne Code. Preis-/Verpackungsmodell, Grenzen, wie es sich zum Odoo-Code-Weg verhält. Kritik einsammeln.
4. **Microsoft Power Platform / Dynamics 365** (Power Apps custom entities/fields, Power Automate): der Microsoft-Selbstbedienungs-Weg.
5. **Zoho Creator + Zoho CRM Customization** (Custom Modules/Fields/Layouts, Blueprint-Workflows): KMU-nah, DACH-relevant.
6. **HubSpot** (Custom Properties, Custom Objects, Workflows): wie einfach für Nicht-Techniker.
7. **Airtable / Monday.com** (Custom Fields/Columns/Views): das „radikal einfach"-Ende — was macht Konfiguration für Nicht-Techniker zugänglich?
8. **Budibase / Retool / Appsmith** (Open-Source Low-Code, self-hostable): relevant für den Orbit-Self-Hosted-Winkel.

**Fokusfragen für ALLE (soweit dokumentiert):**
① Was genau ist ohne Code konfigurierbar, was braucht doch Entwickler? Wo die Grenze?
② Wie sieht die Konfigurations-UI aus (WYSIWYG? Formular? Drag&Drop? Assistent)?
③ **Governance:** Sandbox/Preview vor Live? Versionierung/Rollback? Audit von Config-Änderungen? Wer darf (Rollen)?
④ **Update-Sicherheit:** Überleben Kunden-Anpassungen ein Produkt-Update des Anbieters? Wie technisch gelöst (Overlay vs. Fork)?
⑤ **Multi-Tenancy:** Config pro Mandant isoliert?
⑥ **Vorlagen/Templates:** Branchen-Vorlagen, aus denen man startet (wie unsere RBAC-INDUSTRY_ROLE_TEMPLATES)?
⑦ Bekannte Fallen/Kritik (Overkill, „Config-Hölle", Performance, Sicherheits-Risiken durch Kunden-Formeln).

**Markt-Deliverable:** `MARKT.md` — pro Produkt kompakte Antworten + Quellen-URLs + Abschnitt „Übertragbare Muster + Anti-Muster für Cosmi".

## §5 Governance- & Architektur-Fragen (die schweren Themen — mitdenken, nicht überspringen)

- **Config-Speicher-Architektur:** Wo lebt Kunden-Config? (Tenant-scoped Config-Layer über dem Code-Default — Muster wie RBAC `USER_OVERRIDES` über `ROLE_DEFS`.) JSONB-Config-Tabellen? Wie versioniert?
- **Update-Sicherheit (das kritischste):** Produkt-Update darf Kunden-Config nicht zerstören. Overlay-Prinzip (Code = Default, Config = Overlay obendrauf, nur Abweichungen gespeichert) — dieselbe Semantik wie R-6-Overrides. Migrations-Strategie wenn sich das Code-Schema ändert.
- **No-Code darf nicht No-Safety heißen:** Kunden-Formeln/Validierungen/Workflows dürfen nicht die App crashen oder Security umgehen. Sandboxing, Limits, Validierung.
- **RBAC-Verzahnung:** Neuer Katalog-Key(s) à la `admin:customization:manage` (wer darf konfigurieren). Feld-/Modul-Rechte müssen Custom-Felder/-Module mit-abdecken.
- **Audit + Preview + Rollback:** Jede Config-Änderung ins Audit-Log (R-5-Infrastruktur nutzen), Vorschau vor Live (R-2-`startPreview`-Muster), Rückgängig-Machen.
- **WASM-Plugin-Abgrenzung:** No-Code-Config = 95%-Fälle selbst; WASM (Phase D) = Rest per Code. Klare Grenze definieren, damit sich beide nicht überschneiden.
- **Onboarding-Verzahnung (O-0):** Der geführte Erst-Setup nutzt dieselben Config-Bausteine — Onboarding = geführter Config-Assistent auf demselben Unterbau.
- **Skalierbarkeit für „größere Unternehmen mit eigener IT":** Diese Zielgruppe (Dariens Kern-Argument) will Tiefe UND Sicherheit. Balance geführt (KMU ohne IT) ↔ mächtig (Mittelstand mit IT-Abteilung).

## §6 Erwartbare Darien-Entscheidungsfragen (fürs Gate vorbereiten, nach der Recherche gebündelt stellen)

1. **Scope-Schnitt v1:** Welche Dimensionen aus §3 in die erste Ausbaustufe? (Empfehlung wird aus Ist-Analyse + Aufwand kommen — vermutlich das, was BE-seitig schon Fundament hat: Custom Fields, Terminologie, Listen/Layouts, Status-Sets; Custom Objects/komplexe Workflows später.)
2. **Zielgruppen-Tiefe:** Ein Werkzeug für beide (geführt genug für KMU-ohne-IT + mächtig genug für Mittelstand-IT) oder zwei Stufen (einfacher Modus + Experten-Modus)?
3. **Zentria-vs-Kunde-Grenze:** Alles was Zentria kann auch der Kunde, oder bleibt ein Zentria-only-Kern (wie Deploy/Env-Flags)? Was NUR wir?
4. **Update-Sicherheits-Prinzip:** Overlay-only (Kunde ändert nie Code-Default, nur Abweichung gespeichert — sicherste Variante, meine Erwartung) bestätigen.
5. **Vorlagen:** Branchen-Config-Vorlagen (wie RBAC-Templates) von Anfang an mitdenken?
6. **Naming/Modul-Ort:** Eigenes „Anpassungen"/„Konfiguration"/„Studio"-Modul im Admin-Bereich, oder pro Modul in den Modul-Einstellungen verteilt, oder beides (zentrale Studio-Fläche + Modul-lokale Schnellzugriffe)?

## §7 Ablauf (verbindlich, RBAC-Muster)

1. **Recherche-Gate:** Ist-Analyse-Agents (§2) + Markt-Agents (§4) parallel → `IST-ANALYSE.md` + `MARKT.md`.
2. **Gebündelte Fragen** (§6, mit Empfehlungen aus der Recherche) an Darien.
3. **Darien-Entscheide** → in `KONZEPT.md` §0 festschreiben (SSOT für diesen Block, wie `rbac-block/KONZEPT.md`).
4. **Roadmap-Schnitt** in Ausbaustufen (v1 klein + review-reif, dann erweitern — rollierende Fertigstellung).
5. **Bau** pro Stufe: Fundament (Config-Overlay-Layer + Resolver + Audit) selbst, Editoren via Agents, Gates (i18n ×4, scoped tsc, eslint, Screenshot-QA + Bilder ansehen).
6. **Luke-Paket** in backend-gaps (Config-Persistenz, Overlay-Resolution, Update-Migration).

## §8 Bezug zu bestehenden Plan-Artefakten

- RBAC-Block (Blaupause): `.planning/rbac-block/` (KONZEPT.md, R1…R6-Briefings, R6-RECHERCHE.md).
- backend-gaps §RBAC (Custom-Field-Fund: „work_custom_field_definitions BACKEND ERLEDIGT 2026-06-11, 9 Feldtypen" — genau prüfen, das ist ein existierendes Custom-Field-Fundament!).
- Modul-Feature-Parität (`project_module_feature_parity`, `docs/MODULES_SCOPE_MATRIX.md`) — separater Track, aber Overlap bei modul-spezifischen Wertelisten.
- Onboarding (O-0, `project_onboarding_system`) — geführter Config-Assistent auf demselben Unterbau.
- Dokument-Engine (`project_document_engine`) — Vorlagen-Dimension (J).
- ADRs (`docs/ARCHITECTURE.md`) — für die Config-Speicher-Architektur-Entscheidung ein ADR anlegen.

---
**Nachgelagert (nach diesem Block, von Darien 2026-07-21):** Passwort-Manager + weitere Funktions-Ideen (mit Luke besprochen). Eigenes Paket, NICHT Teil dieses Blocks. Siehe Memory `project_password_manager`.
