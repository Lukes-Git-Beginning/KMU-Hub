---
title: Marktrecherche C – Microsoft Power Platform / Dynamics 365, Airtable, Monday.com, Budibase, Retool, Appsmith
area: customization-block
updated: 2026-07-21
tags: [research, customization, no-code, self-hosted, power-platform, airtable, monday, budibase, retool, appsmith]
---

# Marktrecherche C — MS Power Platform · Airtable/Monday · Budibase/Retool/Appsmith

Kontext: Cosmi-No-Code-Self-Service-Customization-Tool für DACH-KMU-Admins + Zentria-Onboarding-Nutzung.  
Orbit = Self-Hosted-Variante → open-source/self-hostable Tools besonders relevant.

---

## 1. Microsoft Power Platform / Dynamics 365

### ① Was ist ohne Code konfigurierbar — Grenze zur Entwickler-Seite?

**Ohne Code (Maker/Citizen Developer):**
- Dataverse: eigene Tabellen, benutzerdefinierte Spalten (Text, Zahl, Datum, Choice, Lookup, etc.) über den Maker-Portal-Wizard erstellen
- Formulare: Drag-and-Drop im model-driven Form Designer — Felder, Tabs, Sections, Show/Hide-Regeln, Spalten-Layout
- Business Rules: No-Code-Validierungs- und Sichtbarkeits-Regeln per visuellen Rule Builder
- Power Apps Canvas Apps: Drag-and-Drop-UI-Builder; Power Automate: klickbasierter Workflow-Editor
- Power Virtual Agents: No-Code-Chatbot-Builder
- Power BI: Drag-and-Drop-Berichte

**Erfordert Pro-Code / Entwickler:**
- Client-side Scripting (JavaScript) für event-getriebene Logik auf Formular-Ebene
- Server-side Plug-ins (C#) für Business-Logik, die im Transaktionskontext läuft
- Power Apps Component Framework (PCF) für Custom Controls
- Custom Connectors für externe APIs ohne fertigen Connector
- Complex Integrations und tiefe Dataverse-Anpassungen (relationale Modellierungen jenseits Standard-Lookups)

**Quelle:** [Pro Code vs No Code vs Low Code — Community Dynamics](https://community.dynamics.com/blogs/post/?postid=32c7974b-ee45-f011-877a-0022481fa6ba), [Power Apps Maker Docs](https://learn.microsoft.com/en-us/power-apps/maker/model-driven-apps/create-and-edit-forms)

---

### ② Konfigurations-UI konkret

- **Maker Portal** (make.powerapps.com): Einstiegspunkt für alle No-Code-Arbeiten — Solutions-Browser, Table Designer, Form Designer
- **Model-Driven Form Designer**: Drag-and-Drop-Canvas mit linker Komponenten-Leiste (Columns, Controls, Components). Felder auf Formular ziehen, Tab- und Section-Struktur verschieben, Eigenschaften-Panel rechts
- **Copilot im Form Designer**: KI schlägt Spalten-Set basierend auf Formular-Name + Beschreibung vor ("column suggestions by Copilot") — senkt Einstiegshürde
- **Business Rule Designer**: visueller Condition-Action-Flow (WENN Feld X = Y, DANN Feld Z anzeigen/ausblenden/befüllen)
- **Power Automate**: klickbasierter Workflow-Trigger → Schritt-für-Schritt-Assistent mit gefüllten Platzhaltern
- **Canvas Apps**: WYSIWYG-Designer mit Formeln (Excel-ähnlich), stärker Entwickler-nah

**Quelle:** [Create and edit forms — Microsoft Learn](https://learn.microsoft.com/en-us/power-apps/maker/model-driven-apps/create-and-edit-forms), [Power Apps 2026 Guide](https://star-knowledge.com/blog/how-to-use-microsoft-power-apps-in-2026-guide/)

---

### ③ Governance: Sandbox, Preview, Versionierung, Rollback, Wer darf?

**Umgebungs-Modell (Environments):**
- Dev-Environment: Unmanaged Solutions — frei editierbar, Experimentierstage
- Test/UAT/Prod: Managed Solutions — Komponenten in Prod nicht direkt editierbar
- "Sandbox"-Environments sind explizite Environment-Typen (7-Tage-Backup, restore-fähig auf Sandbox, dann ggf. zu Prod konvertieren)

**Versionierung via Solutions:**
- Solutions sind Deployement-Container (Zip-Pakete mit Metadaten)
- Managed Solution hat eine Versionsnummer; Upgrade erhöht die Nummer; Rollback via CLI: Solution der alten Version reimportieren
- `power platform cli` ermöglicht Export/Import, CI/CD-Integration
- "Patches" (kleinere Inkremente auf einem Managed-Solution-Stack) — Microsoft rät inzwischen davon ab, stattdessen Full-Upgrades

**Audit:**
- Dataverse Audit-Log: wer hat was wann geändert (Feldwert-Ebene)
- Power Platform Admin Center: Connector-Übersicht, Lizenz-Monitoring, Environment-Gesundheit

**Wer darf:**
- System Admin / Customizer — Maker-Rolle im Dev-Environment
- In Prod: Nur via Managed-Solution-Import (kein direktes Editieren von Managed-Komponenten)
- Security Roles + Teams steuern Laufzeit-Zugriff auf Daten

**Quelle:** [Managed vs Unmanaged Solutions](https://crmminds.com/2025/05/03/managed-vs-unmanaged-solutions-in-power-apps-dynamics-365/), [Restore Power Platform Environment](https://imperiumdynamics.com/blog/power-platform-restore-environment-does-it-restore-customization-and-solutions), [ESPC Deployments 2026](https://www.sharepointeurope.com/managing-power-platform-and-dynamics-365-deployments/)

---

### ④ Update-Sicherheit: Überleben Kunden-Anpassungen Produkt-Updates?

**Kernmechanismus: Solution Layering**

Dataverse kennt zwei Schichten:
1. **Managed Layer** (Stack aller importierten Managed Solutions + Microsoft-System-Solution ganz unten)
2. **Unmanaged Layer** (Customer-Anpassungen, die direkt im Dev-Environment gemacht wurden)

Prioritätsregel: **Unmanaged gewinnt immer über Managed**; beim Managed-Stack gewinnt die oberste Schicht ("top wins"), außer bei Forms/Sitemaps (dort: Merge-Logik).

Wenn Microsoft das System (unterste Schicht) aktualisiert, bleiben alle darüber liegenden Schichten unberührt — weil sie über der System-Schicht liegen, nicht darin kodiert sind.

**Praktisch:** Kundenseitige Dataverse-Anpassungen (Custom Tables, Fields, Forms) überleben Microsoft-Produkt-Updates weitgehend problemlos, solange keine Namespace-Konflikte mit Microsofts eigenen Tabellen entstehen.

**Risiko:** Wenn Microsoft eine Spalte umbenennt oder löscht, auf die eine Kundenanpassung per Hard-Referenz zeigt → Breaking Change. Selten, aber nicht unmöglich.

**Quelle:** [Solution layers ALM — Microsoft Learn](https://learn.microsoft.com/en-us/power-platform/alm/solution-layers-alm), [Solution layering Medium](https://medium.com/@chamara.iresh/how-to-master-solution-layering-and-troubleshoot-customization-conflicts-in-dynamics-365-crm-bf6c071696e1)

---

### ⑤ Multi-Tenancy: Config pro Mandant isoliert?

- Power Platform ist SaaS-multi-tenant auf Microsoft-Ebene, aber jede Organisation hat eigene **Environments** — vollständige Isolation
- Dataverse-Tabellen sind nicht inhärent tenant-aware; Mandanten-Trennung erfolgt durch separate Environments (Standard-Weg) oder Row-Level-Security innerhalb einer Environment (selten, nur für bestimmte Partner-Szenarien)
- Für ISV-Partner (die D365 an Endkunden weiterverteilen): eigenes AppSource-Modell mit Managed Solutions pro Kunde

---

### ⑥ Vorlagen/Templates

- **Solution Templates**: Microsoft und Partner veröffentlichen vorgefertigte Solutions (z.B. "Customer Service Accelerator", "Sales Accelerator") auf AppSource
- **Power Apps templates**: im Maker-Portal direkt startbar — z.B. "Asset Checkout", "Leave Request"
- **Power Automate templates**: Bibliothek vordefinierter Flows (z.B. Genehmigungsprozesse, E-Mail-Automation)
- 2026: verstärkter Einsatz von Copilot als "Template-Generator" — plain-language Beschreibung → generierter App-Entwurf

**Quelle:** [Power Platform May 2026 Update](https://www.microsoft.com/en-us/power-platform/blog/2026/05/14/whats-new-in-power-platform-may-2026-feature-update/)

---

### ⑦ Bekannte Fallen / Kritik

- **Komplexitäts-Schock**: Maker-Portal ist mächtig, aber der Einstieg für echte Nicht-Techniker ist steil; viele Konzepte (Solutions, Environments, Managed vs. Unmanaged) sind für Business-User nicht intuitiv
- **Solution-Layer-Konflikte**: Wenn mehrere Managed Solutions denselben Komponenten überschreiben ("top wins"), entstehen schwer debugbare Verhaltensänderungen
- **Patches sind deprecated**: Der alte Patch-Workflow ist konfus und wurde von Microsoft selbst zurückgezogen — viele Ältere Tutorials sind irreführend
- **Preis-Komplexität**: Power Platform-Lizenzen (per User, per Flow, per App) sind verwirrend und können unerwartet teuer werden
- **Vendor Lock-in**: Dataverse-Daten und Solution-Format sind proprietär; Migration zu einem anderen System sehr aufwändig
- **Update-Risiko bei Unmanaged-Anpassungen in Prod**: Wenn Admins direkt in Prod (unmanaged) customizen, kann ein Microsoft-Update die UI-Darstellung ändern und die Anpassung "überlagern" — Best Practice wird oft nicht befolgt

---

## 2. Airtable

### ① Was ist ohne Code konfigurierbar — Grenze?

**Ohne Code:**
- Bases (Datenbank-Container) erstellen, Tabellen anlegen
- Feld-Typen konfigurieren: Single Line, Long Text, Attachment, Checkbox, Multiple Select, Date, Phone, Email, URL, Number, Currency, Percent, Duration, Rating, Formula, Rollup, Count, Lookup, Linked Record, Created/Modified Time/By, Auto Number, Barcode, Button, AI Field
- Views erstellen: Grid, Gallery, Kanban, Gantt, Calendar, Form
- Interface Designer: vorgefertigte Layouts (List, Gallery, Kanban, Calendar, Timeline, Dashboard, Record Detail, Record Review, Form) konfigurieren; Elemente per Properties-Panel konfigurieren
- Automations: Trigger + Action-Ketten ohne Code (z.B. "Wenn Status = Done → sende E-Mail")
- Omni (AI-Assistent): per Spracheingabe Tabellen, Views, Automations generieren lassen

**Erfordert Entwickler / Einschränkung:**
- Echte Custom Code-Blocks (Scripting Extension) für komplexe Logik
- Custom Apps über Airtable Apps Marketplace oder eigene React-Apps via Airtable SDK
- Interface Designer Advanced Features (bedingte Sichtbarkeit von Feldern) = Business/Enterprise Plan
- Externe API-Integration ohne vorgefertigten Connector → Scripts nötig

**Quelle:** [Airtable Interface Designer Guide 2026](https://workmanagementhub.com/airtable-interfaces-designer-guide-2026/), [Getting Started Interface Designer — Airtable Support](https://support.airtable.com/docs/getting-started-with-airtable-interface-designer)

---

### ② Konfigurations-UI konkret — UI-Kniffe für Nicht-Techniker

Dies ist der wichtigste Abschnitt für Cosmi.

**Grid View (Kerninteraktion):**
- Jede Spalte = ein Feld; Klick auf "+" → Feld-Typ-Auswahl als Scrollliste mit Icons und Kurzbeschreibungen → kein Formular-Overhead, direkt inline
- Spalten-Umbenennen: Doppelklick auf Header
- Feld-Konfiguration: Klick auf Spalten-Header → "Customize field type" → Slide-over mit kontextspezifischen Optionen (je nach Typ andere Einstellungen)
- Reihenfolge: Drag-and-Drop von Spalten-Headern
- **Progressives Konzept**: Nutzer fangen mit einfachen Text-Feldern an; komplexere Typen (Lookup, Rollup, Formula) werden erst sichtbar, wenn man danach sucht

**Views (Perspektiven-Wechsel):**
- Links "Add a view"-Button → View-Typ-Auswahl (visuell mit Thumbnails) → sofort aktiv
- Jede View speichert eigene Filter/Sortierung/Felder-Sichtbarkeit — keine Daten-Kopie
- "View Sharing": einzelne Views als Read-Only-Link teilen ohne Base-Zugang → Stakeholder-Einbindung ohne Lernkurve

**Interface Designer:**
- "Interfaces" als separater Tab im Base → "Create interface" → Layout-Auswahl-Wizard mit Vorschau-Thumbnails
- Keine klassische Drag-and-Drop-Canvas; stattdessen Property-Panel rechts — konfigurationsbasiert statt konstruktionsbasiert
- Omni-KI: "Baue mir ein Dashboard für das Vertriebsteam" → generiert Interface-Entwurf → Nutzer verfeinert
- "Publish"-Button separiert Editing-Modus von Live-Zustand

**Onboarding-Kniffe:**
- Leer-State mit konkreten Beispiel-Daten vorbelegt (statt leere Felder)
- Inline-Tooltips bei Feld-Typen erklären Zweck (z.B. "Rollup: Aggregiere Werte aus verknüpften Tabellen")
- Template-Vorschlag beim Erstellen einer neuen Base

**Quelle:** [Airtable Interface Designer Support](https://support.airtable.com/docs/getting-started-with-airtable-interface-designer), [Airtable Interfaces 2026](https://noloco.io/blog/airtable-interfaces), [Airtable Review 2026](https://www.softr.io/blog/airtable-review)

---

### ③ Governance: Sandbox, Versionierung, Rollback, Wer darf?

**Revisionen / Snapshot-History:**
- Airtable hat Revisions-History (Record-Ebene), nicht auf Schema-Ebene
- Kein nativer "Schema-Rollback" (Feld gelöscht = Daten weg, kein Undo nach Seiten-Reload)
- Enterprise: regelmäßige Snapshot-Exporte empfohlen (CSV/JSON)

**Interface-Permissions:**
- Owner/Creator: kann Interfaces erstellen, editieren, publishen
- Collaborator: kann Interfaces nutzen, abhängig von Laufzeit-Berechtigungen
- End User: nur interagieren, kein Editieren der Interface-Struktur
- Base-Backend-Permissions überschreiben Interface-Permissions ("field and table editing permissions set in the backend base overrule interface settings")

**Audit (Enterprise):**
- Audit Logs über Admin API: wer hat was wann gemacht (User-Aktionen, Feld-Änderungen, Share-Aktionen)
- Admin Panel: User-Export, SCIM-Provisioning, granulare Rollen
- ISO 27001, SOC 2, GDPR-compliant

**Sandbox/Preview:** Kein explizites Sandbox-Environment; Interface "Publish" als einziges Preview-Gate

**Quelle:** [Airtable Enterprise Governance](https://www.airtable.com/platform/governance), [Airtable Audit Logs API](https://airtable.com/developers/web/api/audit-logs-integration-guide), [Airtable Permission Controls](https://www.itsmconnect.com/post/airtable-introduces-advanced-permission-controls-for-enterprise-workflows)

---

### ④ Update-Sicherheit: Überleben Anpassungen Produkt-Updates?

- Airtable ist reines SaaS; Updates werden von Airtable ausgerollt
- Kundenseitige Anpassungen (Felder, Views, Interfaces, Automations) liegen in der Customer-Base-Schicht — von Produkt-Updates vollständig getrennt
- Schnittstellen-API ist versioniert (v0-API, stabile Endpunkte); Breaking Changes werden angekündigt
- **Risiko**: Wenn Airtable einen Interface-Element-Typ deprecated, können bestehende Interfaces das Element-Verhalten verlieren → bisher selten dokumentiert
- Keine Self-Hosted-Option → Update-Zeitpunkt nicht kontrollierbar

---

### ⑤ Multi-Tenancy

- Airtable ist per se Single-Workspace-Modell; Mandanten-Trennung erfolgt über Workspaces und Bases
- Enterprise: Organization-Layer mit übergreifendem Admin-Panel
- Keine native "ein Base, mehrere Mandanten"-Isolation; Multi-Tenancy = separate Bases pro Mandant (manueller Overhead)

---

### ⑥ Templates

- Über 200 offizielle Templates in verschiedenen Kategorien (Sales, Marketing, HR, Projektmanagement)
- "Template Center" direkt beim Erstellen einer neuen Base
- AI Templates (2026): per Text-Beschreibung generierten Workflow als Startpunkt; jedes AI-Template enthält "Get started"-Guide
- Partner-Templates über Airtable Universe (Community)

---

### ⑦ Bekannte Fallen / Kritik

- **Schema-Änderungen sind destruktiv**: Feld löschen → Daten weg, kein strukturierter Rollback. Kritisch für Prod-Setups
- **Formula-Kompetenz nötig**: Sobald Rollups/Lookups/Formulas gebraucht werden, steigt Komplexität sprunghaft — für echte Nicht-Techniker Grenze
- **Interface Designer = schwaches WYSIWYG**: Konfigurations-Panel statt echter Canvas; räumliche Orientierung fehlt
- **Plan-Gating wichtiger Features**: Bedingte Sichtbarkeit, unbegrenzte Interfaces, Enterprise-Audit — nur auf teuren Plänen
- **Kein Self-Hosting**: Orbit-Szenario ausgeschlossen

---

## 3. Monday.com

### ① Was ist ohne Code konfigurierbar — Grenze?

**Ohne Code:**
- Boards erstellen, Gruppen (Abschnitte) anlegen/umbenennen
- Spalten-Typen hinzufügen und konfigurieren: Status (mit farbigen Labels), People, Date, Text, Numbers, Timeline, Tags, File, Link, Rating, Country, Phone, Email, Dropdown, Color Picker, World Clock, Location, Dependency, Formula, Connect Boards, Mirror, Button, Checkbox, Hour, Item ID, Last Updated, Created, Auto Number, Vote
- Views: Table, Kanban, Gantt, Calendar, Chart, Map, Workload, Form
- Automations: Click-Based-Automation-Builder (WENN Trigger DANN Aktion) mit vordefinierten Rezepten
- Dashboards: Widgets aus mehreren Boards kombinieren
- Integrationen: 200+ Apps per Click-Based-Connector (Slack, Gmail, Jira, etc.)

**Erfordert Entwickler:**
- monday Apps Framework (React) für Custom Columns / Views / Widgets
- API-Integrationen jenseits vorgefertigter Konnektoren (REST API)
- monday Vibe (No-Code-App-Builder für einfache Custom Apps — neues 2026-Feature, noch Beta-ähnlich)

---

### ② Konfigurations-UI konkret — UI-Kniffe für Nicht-Techniker

**Spalten-System (Kern-USP für Nicht-Techniker):**
- "+" am Ende jeder Zeile öffnet "Column Center" — Spalten-Bibliothek mit Kategorie-Filterung (Core, Communication, Files, Time, etc.) + KI-Suchfeld ("describe what you need")
- Jede Spalte hat einen **Farb-codierten Typ-Chip** — visuell sofort erkennbar (Status = grün/gelb/rot, Date = blau, Person = Foto-Avatar)
- Status-Spalte: Klick auf Label → Inline-Farbwähler + Umbenennung ohne extra Modal
- Drag-and-Drop: Spalten einfach per Mausklick verschieben; Items in Kanban-View verschieben = Status-Update (direktes mentales Mapping)

**Template-first Onboarding:**
- Neues Board → Template-Galerie (300+ Templates) mit Live-Preview; AI ("monday magic") generiert Board aus Textbeschreibung
- "monday magic": Nutzer beschreibt Arbeitsablauf in natürlicher Sprache → vollständiges Board mit Spalten + Automations + Views generiert
- Managed Templates (Admin-Feature): Admins definieren Pflicht-Templates die Teams nicht verändern können → Governance

**Progressive Disclosure:**
- Simple Boards starten mit nur 3 Spalten (Status, Besitzer, Datum)
- Komplexere Funktionen (Dependency, Mirror, Formula) in der "More"-Sektion des Column-Centers — nicht sofort sichtbar

**Onboarding Dashboard:**
- Zentrales Onboarding-Widget auf der Hauptseite: Action-Checkliste mit Auto-Abhaken, Name-Personalisierung, Tageszeit-Grußformel
- Icons-only Sidebar spart Platz für Board-Shortcuts

**Spalten-Templates:**
- Nutzer können eigene Spalten-Konfigurationen (inkl. Status-Labels, Farben) als Template speichern → via Column Center wiederverwendbar in anderen Boards

**Quelle:** [monday.com column types support](https://support.monday.com/hc/en-us/articles/115005310285-Available-column-types-on-monday-com), [monday.com basics of columns](https://support.monday.com/hc/en-us/articles/115005466609-The-basics-of-columns), [monday.com onboarding](https://www.candu.ai/blog/monday-com-customer-onboarding-experience), [Column templates](https://support.monday.com/hc/en-us/articles/4405213200018-Column-templates)

---

### ③ Governance

**Permissions:**
- Board-Ebene: Main (ganze Organisation), Shareable (externe Gäste), Private (eingeladene Member)
- Spalten-Ebene: bestimmte Spalten können auf "Readonly" gesetzt werden
- Automations: nur Board-Owner/Admin können Automations editieren

**Managed Templates (Enterprise):**
- Admins erstellen Templates, die für alle Teams verbindlich sind (Pflicht-Struktur)
- Änderungen an Managed Templates propagieren nicht automatisch zu bestehenden Boards (Boards sind bei Erstellung geforkt)

**Audit:**
- Enterprise-Plan: Activity Log pro Board; Admin-Panel für Org-weite Aktivitäten
- Keine Schema-Versionierung (Spalte löschen = weg)

**Quelle:** [Permissions on monday.com](https://support.monday.com/hc/en-us/articles/360019222479-Permissions-on-monday-com), [Managed templates](https://support.monday.com/hc/en-us/articles/18229256953234-Managed-templates-on-monday-com)

---

### ④ Update-Sicherheit

- Reines SaaS; Updates nicht kontrollierbar
- Spalten-Konfigurationen (Labels, Farben, Structure) bleiben über Updates erhalten
- API v2 (GraphQL) ist stabil versioniert
- Kein Self-Hosting

---

### ⑤ Multi-Tenancy

- Enterprise: Organisation-Hub mit Sub-Teams; keine native "pro Mandant isolierte Config"
- Trennung via separate Workspaces oder Accounts

---

### ⑥ Templates

- 300+ offizielle Templates, kategorisiert nach Branche und Use Case
- monday magic: KI-generiertes Board aus Freitext
- AI Templates mit integrierten "Get started"-Guides
- Partner-Ökosystem via monday Apps Marketplace

---

### ⑦ Bekannte Fallen / Kritik

- **Preis**: Pro-User-Pricing für erweiterte Features wird für Teams mit vielen Usern teuer
- **Formula-Spalte**: Eingeschränkte Formelfunktionen verglichen mit Airtable/Excel
- **Kein Schema-Rollback**: wie Airtable — destruktive Änderungen sofort live
- **Automations-Limit pro Plan**: Free hat kaum Automations, Team-Plan begrenzt
- **Kein Self-Hosting**: Orbit ausgeschlossen

---

## 4. Budibase (Open-Source Self-Hosted Low-Code)

### ① Was ist ohne Code konfigurierbar — Grenze?

**Ohne Code:**
- CRUD-Apps auf Basis existierender Datenquellen (PostgreSQL, MySQL, REST API, Google Sheets, MongoDB, CouchDB, S3, etc.) — visueller Table/Form-Builder
- UI: Drag-and-Drop-Komponenten-Builder (Tables, Forms, Charts, Cards, Containers)
- Automations: Trigger-Action-Ketten per visuellen Builder
- Role-Based Access Control: Benutzer-Rollen per Klick zuweisen
- "Autogenerated UI": Budibase scannt Datenbank-Schema und generiert CRUD-Interface automatisch

**Erfordert Entwickler:**
- Benutzerdefinierte JavaScript-Logik innerhalb von Automation-Steps oder Component-Bindings
- Custom Plugins/Datasources (Node.js/React) für nicht-native Integrationen
- Custom Styling jenseits der eingebauten Themes
- Kubernetes-Deployment-Setup und -Betrieb

**Quelle:** [Budibase Review 2026](https://aitoolscoop.com/tool/budibase/), [Budibase Custom Plugins Docs](https://docs.budibase.com/docs/custom-plugin)

---

### ② Konfigurations-UI konkret

- **Portals-Konzept**: Mehrere Apps werden in "Portals" gruppiert — interner Multi-App-Workspace
- **UI Builder**: linke Komponenten-Bibliothek → Drag-and-Drop auf die Canvas; Properties-Panel rechts; Live-Preview mittig
- **Data-first**: Datenquelle zuerst verbinden → dann generiert Budibase einen App-Entwurf automatisch
- **Dev/Prod-Switcher**: separate Test- und Produktions-Daten innerhalb einer App (ab v2.x)
- Design-Theme: vordefinierte Farbpaletten; Custom Branding für Self-Hosted

---

### ③ Governance: Sandbox, Preview, Versionierung, Rollback, Wer darf?

**Dev/Prod-Trennung:**
- "Dev/Prod Switcher" erlaubt Arbeit an der App ohne Live-Prod-Auswirkungen (seit v2.x)

**Versionierung:**
- Budibase hat **keine native integrierte Git-Versionierung** für App-Konfigurationen
- App-Definitionen werden intern in CouchDB als JSON gespeichert (nicht als Git-natives Format)
- Backup-Strategie: Export-Funktion (App als `.tar.gz` exportieren) → manuell versionieren
- Ab v2.33.0: SQS (Structured Query Server) als zusätzliche Persistenz-Schicht neben CouchDB

**Audit:**
- Kein eingebauter Enterprise-Audit-Log in der Community-Edition

**Wer darf:**
- RBAC: Admin, App Creator, Basic User, Public — vordefinierte Rollen
- App-Zugang per Rolle steuerbar

**Quelle:** [Budibase Migrations Docs](https://docs.budibase.com/docs/migrations), [Budibase Update Docs](https://docs.budibase.com/docs/updating-budibase), [Budibase vs Appsmith vs ToolJet 2026](https://www.pistack.xyz/posts/budibase-vs-appsmith-vs-tooljet-self-hosted-low-code-guide-2026/)

---

### ④ Update-Sicherheit: Überleben Config bei Self-Hosted-Upgrades?

**Kritischer Befund für Orbit-Relevanz:**

- Budibase-Upgrades erfolgen via `docker compose pull` + `docker compose up -d` oder Budibase CLI
- **v2.33.0 war ein Breaking-Change-Upgrade**: Umstieg auf SQS als Datenbank-Layer erforderte manuelle Infrastruktur-Anpassungen; falsche Kubernetes-Upgrades (nur Image-Tag ändern) brachen Installationen
- Die offizielle Dokumentation enthält **keine expliziten Pre-Upgrade-Backup-Empfehlungen** und keine Breaking-Change-Warnungen in den Update-Docs selbst
- App-Konfigurationen (in CouchDB gespeichert) persistieren in gemounteten Docker-Volumes → überleben Container-Updates, solange Volumes erhalten bleiben
- Es gibt **keinen automatischen Rollback-Mechanismus**: Downgrade = manuelles Ändern des Image-Tags im Compose-File

**Bewertung für Orbit**: Upgrade-Prozess setzt technisches Know-how voraus; End-Kunden-selbst-updaten riskant ohne abstrahierte Update-UI.

**Quelle:** [Budibase Update Docs](https://docs.budibase.com/docs/updating-budibase), [Budibase Migrations](https://docs.budibase.com/docs/migrations), [GitHub Discussion #13024](https://github.com/Budibase/budibase/discussions/13024)

---

### ⑤ Multi-Tenancy

- Self-Hosted: alle Apps laufen in einer Budibase-Instanz; Trennung via separate Apps + RBAC
- Kein natives "Mandant A sieht Config X, Mandant B sieht Config Y" auf einer geteilten Instanz ohne separate App-Objekte
- Für echte Mandanten-Trennung: separate Budibase-Instanzen oder separate Apps mit eigenen Datenquellen-Verbindungen

---

### ⑥ Templates

- Eingebaute App-Templates (z.B. CRM, Inventar, Helpdesk, Rekrutierung) direkt im Builder wählbar
- Template-App wird als Startpunkt geforkt

---

### ⑦ Bekannte Fallen / Kritik

- **Breaking Upgrades ohne klare Kommunikation**: v2.33.0-Migration hat viele Self-Hosted-User überrascht
- **Keine Git-native Config-Versionierung**: Im Gegensatz zu Appsmith (alle Pläne) und Retool (Enterprise) — Budibase bleibt hier schwach
- **CouchDB-Abhängigkeit**: unübliche Datenbank für Kunden, die eigene DB-Kompetenz mitbringen; SQS-Layer macht es noch komplizierter
- **Custom Plugins = Node.js/React**: sobald Default-Komponenten nicht reichen, braucht man Entwickler
- **Community Edition ohne Audit-Log**: für Enterprise/Compliance-Szenarien unzureichend

---

## 5. Retool (Closed Source, Self-Hosted Option)

### ① Was ist ohne Code konfigurierbar — Grenze?

**Ohne Code / Low-Code:**
- UI: Drag-and-Drop-Builder mit 100+ vorgebauten Komponenten (Table, Form, Chart, Button, Input, etc.)
- Queries: visuelle Query-Builder für SQL, REST, GraphQL, etc.
- Einfache Logik: Event-Handler per Klick (OnClick → run query)
- Workflows: Workflow Builder (separates Produkt, visuell)

**Erfordert Code / Entwickler:**
- Retool ist primär auf **Full-Stack Engineers** ausgerichtet; JavaScript-Kenntnisse für alle nicht-trivialen Operationen nötig (Transformationen, Custom Logic, bedingtes Verhalten)
- "Particularly code-intensive" laut unabhängigen Reviews

---

### ② Konfigurations-UI

- WYSIWYG-Canvas mit Snap-to-Grid; Komponenten-Panel links, Properties rechts
- Inline JavaScript-Editor für Transformationen direkt im Properties-Panel
- State Management über JavaScript-Objekte

---

### ③ Governance: Versionierung, Rollback, Audit

**Source Control (Git) — Kern-USP für Self-Hosted:**
- App-Definitionen werden als **strukturierte JSON-Dateien** exportiert (jede Komponente, Query, Event-Handler als JSON-Schlüssel)
- **Retool CLI**: exportiert JSON → Git-Repository; bei Bedarf reimportieren
- **Enterprise: Native GitHub/GitLab/Azure-Integration**: automatischer Commit bei jedem Save; Branch-Workflows (feature branches, PR-Review, Visual Testing vor Merge)
- Branch-Modell: Dev arbeitet auf Feature-Branch; End-User sieht immer `main`; Merge nach Code Review
- **Rollback**: Git-Revert der JSON-Datei → reimportieren in Retool → vorige Version aktiv
- `VERSION_CONTROL_LOCKED=true` Env-Var verhindert, dass die Instanz selbst in Git pusht (read-only)

**Audit:**
- Automatisches Audit-Log jeder User-Aktion (Enterprise)
- Granulare RBAC: App-Ebene, Seiten-Ebene, Custom Permission Groups

**Quelle:** [Retool Source Control Blog](https://retool.com/blog/git-branching-with-source-control), [Retool GitHub Integration Guide](https://retoolers.io/blog-posts/complete-guide-to-retool-github-integration-and-source-control), [Retool Self-Hosted](https://retool.com/self-hosted)

---

### ④ Update-Sicherheit Self-Hosted

- Da App-Configs als JSON in Git versioniert sind: Retool-Produkt-Upgrade unabhängig von App-Config
- Bei einem Breaking-Change im Retool-Produkt: JSON-Format könnte migriert werden müssen; Retool hat Migrations-Skripte für Major-Upgrades
- Rollback via Docker-Image-Tag-Downgrade + Git-Revert der Configs

---

### ⑤ Multi-Tenancy

- **Stark begrenzt für Free/Team**: 25-User-Limit für Self-Hosted Non-Enterprise
- Enterprise: Custom Permission Groups, Multi-Workspace-Support
- Keine native "Mandant A / Mandant B"-Config-Trennung auf einer Instanz; Trennung via Permission-Groups und separate App-Objekte

---

### ⑥ Templates

- Retool-Template-Library (intern): vorgefertigte Apps für häufige Use Cases (Admin Panel, Dashboards, CRUD-Apps)
- Import/Export von App-JSONs als Sharing-Mechanismus

---

### ⑦ Bekannte Fallen / Kritik

- **Git Source Control = Enterprise-only** für Native Integration; Free/Team nur via CLI (mehr Friction)
- **Primär Dev-Tool**: für echte No-Code-User (nicht-technische Admins) ungeeignet — zu code-lastig
- **25-User-Limit Self-Hosted Non-Enterprise**: für Orbit-Multi-Tenant-Szenarien problematisch
- **Geschlossener Quellcode**: Self-Hosted ohne Open-Source-Garantien; Vendor-Abhängigkeit bleibt

---

## 6. Appsmith (Open-Source, Self-Hosted)

### ① Was ist ohne Code konfigurierbar — Grenze?

- Ähnlich Retool: primär Developer-Tool; "heavily tailored towards professional developers"
- Einstiegshürde für No-Code-User ähnlich (JavaScript-Kenntnisse für Logik nötig)
- Einfacher als Retool für einfache Tasks ("less steep learning curve")

---

### ② Konfigurations-UI

- WYSIWYG-Canvas, Drag-and-Drop-Komponenten
- Inline JavaScript für Bindings und Events

---

### ③ Governance: Versionierung, Rollback, Audit

**Git — verfügbar auf ALLEN Plänen (Unterschied zu Retool):**
- Jede App kann mit einem Git-Repository verbunden werden (GitHub, GitLab, Bitbucket, Azure)
- Branches: Dev-Branch für Entwicklung, Production-Branch für Live
- **Auto-Commit**: wenn aktiviert, committed Appsmith Änderungen automatisch bei Version-Upgrades des Produkts auf einen nicht-geschützten Branch
- **Multi-Environment via Git**: `main` = Prod, Feature-Branches = Dev/Staging; eigene Datenquellen-Configs pro Branch
- Package Version Control: separate Versionierung von wiederverwendbaren Modulen (Packages)

**Audit:**
- Enterprise: vollständiges Audit-Log (was wurde gemacht, von wem, wann); auch Config-Änderungen geloggt

**Quelle:** [Appsmith Git Version Control Docs](https://docs.appsmith.com/advanced-concepts/version-control-with-git), [Appsmith Audit Logs](https://github.com/appsmithorg/appsmith-docs/blob/main/website/docs/advanced-concepts/audit-logs.md), [Appsmith Environments with Git](https://docs.appsmith.com/advanced-concepts/version-control-with-git/environments-with-git)

---

### ④ Update-Sicherheit Self-Hosted

- App-Configs in Git = unabhängig vom Produkt-Upgrade
- **Auto-Commit bei Produkt-Upgrade**: wenn Appsmith selbst ein Breaking-Schema-Update auf eine App anwenden muss, committet es automatisch den Migrations-Diff in Git → nachvollziehbar
- Empfehlung: Update alle 2 Wochen; Backup via Kubernetes `values.yaml` in Git

---

### ⑤ Multi-Tenancy

- Workspace-basierte Trennung; RBAC auf Workspace- und App-Ebene
- SCIM-Provisioning für Enterprise (User-Sync aus IdP)
- Keine native Mandanten-Config-Trennung auf einer Instanz ohne separate Workspaces

---

### ⑥ Templates

- Eingebaute Template-Library (Admin Panels, CRUD, Dashboards, etc.)
- Community-Templates via GitHub

---

### ⑦ Bekannte Fallen / Kritik

- **Code-First trotz Low-Code-Label**: für echte Nicht-Techniker unzumutbar
- **Automation = Code-first**: Workflow-Automatisierungen erfordern JavaScript
- **Community-Edition-Einschränkungen**: manche Features (SSO, Audit) nur Enterprise

---

## Übertragbare Muster + Anti-Muster für Cosmi

### Übertragbare Muster

**M1 — Konfigurations-Schichten-Prinzip (Power Platform → Cosmi-Schema)**  
Trenne Zentria-Produkt-Config (Managed, nicht editierbar) von Kunden-Config (Unmanaged, editierbar). Beim Cosmi-Update darf die Kunden-Schicht nie überschrieben werden. Technisch: JSON-Schema mit `_vendor`-Schlüsseln (Zentria) + `_tenant`-Schlüsseln (Kunde); beim Update werden nur `_vendor`-Schlüssel gemerged, `_tenant` bleibt. Analog zum Power-Platform-"Unmanaged always wins"-Prinzip.

**M2 — Inline-Kontextmenü statt Modal-Overhead (Airtable/Monday → Cosmi-Field-Editor)**  
Feldkonfiguration nicht hinter einem separaten Modal verstecken. Airtable: Klick auf Spalten-Header → Slide-over genügt. Monday: Status-Label direkt inline editieren (Klick → Farbe + Name ändern, fertig). Für Cosmi: Felder, Views, Listen-Konfigurationen direkt am Ort des Geschehens editierbar; kein "Einstellungen → Modul → Felder → Edit"-Pfad mit 4 Tiefenebenen.

**M3 — Template-first Einstieg + Progressive Disclosure (Monday/Airtable → Cosmi-Onboarding)**  
Neuer Tenant sieht niemals einen leeren Screen. Branchenspezifische Start-Templates (Handwerk, Kanzlei, IT-Service) mit vorausgefüllten Feldern, Ansichten und Demo-Daten. Komplexe Optionen (Custom Workflows, bedingte Sichtbarkeit, Tenant-weite Richtlinien) erst nach Basis-Setup sichtbar. Monday magic / Airtable Omni als Referenz für KI-gestützte Template-Generierung aus Freitext.

**M4 — Git-native Config-Versionierung für Orbit (Appsmith-Modell)**  
Orbit-Config-Store als JSON mit nativer Git-Integration: jede Konfigurationsänderung committed automatisch in ein Kunden-eigenes Repo (Auto-Commit-Modell von Appsmith). Rollback = `git revert`. Zentria-Updates bringen nur die `_vendor`-Schicht; Kunden-JSON-Teile werden nicht überschrieben. Breaking Changes müssen als explizite Migrations (Appsmith Auto-Commit-Ansatz) veröffentlicht werden.

**M5 — "Publish"-Gate als Schutz vor versehentlichen Live-Änderungen (Airtable/Power Platform)**  
Änderungen im Konfigurations-Modus sind erst live, nachdem der Admin "Veröffentlichen" klickt. In Airtable trennt der "Publish"-Button Editing-Modus von End-User-Sicht. In Power Platform: "Save" vs. "Save and Publish". Für Cosmi: Preview-Modus für den Admin, der zeigt wie die Konfiguration für den Endnutzer aussieht, bevor sie live geht.

---

### Anti-Muster

**A1 — Code-Schleier als Low-Code verkaufen (Retool/Appsmith)**  
Retool und Appsmith sind primär Entwickler-Tools mit Low-Code-Oberfläche. Für einen Nicht-Techniker (KMU-Admin ohne IT-Hintergrund) ist JavaScript in Bindings keine Low-Code-Erfahrung. Anti-Muster: visuelle UI-Oberfläche anbieten, die in jeder zweiten Konfigurationssituation eine JavaScript-Konsole aufmacht. Cosmi-Config muss ohne Code vollständig bedienbar sein; Code-Escape-Hatch nur für Power-User-Szenarien.

**A2 — Destruktive Schema-Änderungen ohne Warnung (Airtable/Monday)**  
In Airtable und Monday: Feld löschen = Daten sofort weg, kein Undo. Für ein KMU-CRM-Konfigurations-Tool ist das inakzeptabel. Cosmi muss jede destruktive Konfigurationsänderung (Feld-Löschung, Typ-Änderung mit Datenverlust, Modul-Deaktivierung) mit einem klaren Konsequenz-Dialog absichern: "X Datensätze verlieren dieses Feld. Fortfahren?" + Soft-Delete (Feld als "archiviert" markieren, nicht sofort gelöscht, 30-Tage-Wiederherstellungsfenster).

---

*Quellen-Zusammenfassung: Microsoft Learn (solution-layers-alm, create-and-edit-forms, types-of-fields), Airtable Support (interface-designer, audit-logs), Appsmith Docs (version-control-with-git, environments-with-git), Retool Blog (git-branching), Budibase Docs (updating-budibase, migrations), monday.com Support (column-types, basics-of-columns, managed-templates), CRMMinds (managed-vs-unmanaged-solutions), unabhängige Reviews (pistack.xyz, aitoolscoop.com, softr.io).*
