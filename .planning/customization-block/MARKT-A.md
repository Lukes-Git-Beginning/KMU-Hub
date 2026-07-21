# MARKT-A: Salesforce + ServiceNow — No-Code-Customization-Analyse

> **Recherche-Agent:** MARKT-A (Salesforce + ServiceNow Enterprise-Gold-Standards)
> **Datum:** 2026-07-21
> **Zweck:** Marktrecherche für Cosmi No-Code-Self-Service-Customization-Tool
> **Quellen:** Offizielle Doku, Trailhead, Salesforce Ben Surveys, ServiceNow Community

---

## 1. Salesforce

### ① Admin-Grenze: Was ohne Code, was mit Entwickler?

**Ohne Code (Admin-Ebene) — vollständig deklarativ:**

- **Custom Objects & Fields:** Jeder Admin kann im Object Manager neue Objekte (Tabellen) und Felder (Text, Zahl, Datum, Picklist, Lookup, Formel, Checkbox u.v.m.) anlegen. Kein Code nötig. [Quelle: Salesforce LWC Tutorials](https://www.salesforcelwc.com/salesforce-object-manager-explained-fields-page-layouts-and-relationships/)
- **Page Layouts:** Drag-and-Drop-Layout-Editor bestimmt, welche Felder in welcher Reihenfolge auf einem Datensatz erscheinen. Getrennte Layouts pro Profil/Recordtype möglich.
- **Lightning App Builder:** Visueller WYSIWYG-Builder für Seitenstruktur (Tabs, Komponenten-Platzierung). Kein Code. [Quelle: Trailhead](https://trailhead.salesforce.com/content/learn/modules/lightning_app_builder/lightning_app_builder_intro)
- **Flow Builder:** Visueller Canvas mit Drag-and-Drop für automatisierte Prozesse. Record-Triggered Flows (bei Datensatz-Änderung), Screen Flows (mit Benutzer-Interaktion), Scheduled Flows. Elemente: `Create/Update/Delete Records`, `Decision`, `Assignment`, `Send Email`, `Screen`, `Wait`. Enthält If-then-Logik, Collection-Filter, Subflows. [Quelle: default.com Flow Guide](https://www.default.com/post/salesforce-flow-building-visual-workflows-in-salesforce)
- **Validation Rules:** Formel-basierte Validierungen auf Felder, vollständig no-code per Formel-Editor.
- **Global Value Sets (Picklists):** Zentral verwaltete Wertelisten, die mehreren Feldern zugewiesen werden. Eine Änderung propagiert überall. Max. 1.000 Werte pro Set, 500 Sets pro Org. [Quelle: Salesforce Ben Picklists](https://www.salesforceben.com/global-picklists-in-salesforce-explained/)
- **Custom Metadata Types:** Konfigurations-Datensätze, die als Metadaten (nicht Daten) behandelt werden — deploybar, versionierbar, referenzierbar in Flows und Validation Rules. Admin-editierbar, aber technischeres Konzept. [Quelle: Salesforce Ben CMT](https://www.salesforceben.com/custom-metadata-types/)
- **Profiles, Permission Sets, Roles:** Vollständig GUI-basiert.
- **Reports & Dashboards:** No-Code-Builder.

**Grauzone (Admin möglich, aber technisches Verständnis nötig):**

- Komplexe Formeln in Validation Rules und Flows erfordern Formelkenntnisse (Boolean-Logik, Funktionen).
- Dependent Picklists in Flows erfordern Custom Metadata Type als Hilfskonstrukt — konzeptuell komplex.
- Komplexe mehrstufige Flows mit Entscheidungslogik und Daten-Transforms werden schnell unübersichtlich.

**Erfordert Entwickler (Apex-Code):**

- Komplexe mathematische Operationen / algorithmische Logik.
- Callouts zu externen APIs mit komplexer Fehlerbehandlung.
- Batch-Prozesse auf Millionen Datensätze (DML-Governor-Limits umgehen).
- Custom Lightning Components (LWC) für Sonder-UI-Anforderungen.
- Apex Triggers für Szenarien, die Flow-Grenzen überschreiten.

**Fazit Grenzlinie:** Ca. 80% der typischen CRM-Konfiguration ist für erfahrene Admins ohne Code machbar. Die restlichen 20% erfordern Entwickler oder sind Governor-Limit-Workarounds. Die Grenze liegt nicht am Konzept, sondern am wachsenden technischen Verständnis, das Flows für komplexe Logik verlangen.

---

### ② Konfigurations-UI konkret

**Object Manager:**
- Formular-basierter Setup-Bereich in der Salesforce-Administrationsoberfläche.
- Keine wirkliche WYSIWYG-Vorschau beim Anlegen von Feldern — man konfiguriert Eigenschaften, sieht das Ergebnis erst im Page-Layout-Editor.

**Lightning App Builder:**
- Echter WYSIWYG-Drag-and-Drop-Builder.
- Linke Seitenleiste mit Komponenten-Bibliothek, Canvas in der Mitte, Properties-Panel rechts.
- Live-Vorschau während des Bauens.
- Verschiedene Gerätemodi (Desktop, Tablet, Mobil) wählbar.

**Flow Builder:**
- Visueller Canvas mit zwei Layout-Modi: Free-form (manuell positionieren, Pfeile ziehen) und Auto-Layout (automatische Anordnung in Ausführungsreihenfolge).
- Toolbox links: Elemente nach Kategorie gruppiert, per Drag-and-Drop auf Canvas ziehen.
- Properties-Panel rechts: Feld-Konfiguration, Formeln, Bedingungen.
- „Debug"-Modus: Flow im Test-Modus mit Schritt-für-Schritt-Durchlauf.
- Kein echter „Preview in Produkt"-Modus — nur separater Debug-Run.

**Page Layout Editor:**
- Klassischer Drag-and-Drop-Bereich: oben Feld-Palette, unten Layout-Canvas.
- Felder per Drag-and-Drop in Sektionen ziehen.
- Getrennte Layouts pro Profil/Record-Type konfigurierbar.

**Allgemeines Setup:**
- Alles unter dem zentralen „Setup"-Menü (Zahnrad oben rechts).
- Keine einheitliche In-Produkt-Oberfläche — Setup ist ein separater Modus, klar vom normalen CRM getrennt.

---

### ③ Governance

**Sandbox:**
- Salesforce bietet Sandbox-Umgebungen (Developer, Partial Copy, Full Sandbox).
- Änderungen werden in der Sandbox entwickelt und getestet, dann per **Change Sets** oder **Metadata API** in die Produktion deployed.
- Sandbox-Erstellung ist zeitaufwendig und kostenpflichtig (je nach Edition).
- Kein One-Click-„Preview vor Go-Live" innerhalb einer Umgebung — der Workflow ist immer Sandbox → Production.

**Versionierung:**
- Kein natives Git-ähnliches Versionierungssystem.
- Flows haben eine interne Versions-History (frühere Versionen anzeigbar, reaktivierbar).
- Für alles andere: externe Tools (Gearset, Copado, AutoRABIT) für Git-basiertes Metadata-Management empfohlen.
- **Kein nativer Rollback** für Config-Änderungen — Audit Trail ist ein reines Log.

**Audit Trail (Setup Audit Trail):**
- Protokolliert Metadaten-Änderungen der letzten 180 Tage (exportierbar als CSV).
- Zeigt: Wer hat wann was geändert (Timestamp, User, Change-Kategorie).
- **Kritische Lücke:** Bei Page Layouts und Lightning Pages wird nur angezeigt, dass eine Änderung stattfand — nicht was sich geändert hat. [Quelle: Salesforce Ben Audit Trail](https://www.salesforceben.com/setup-audit-trail-keep-track-of-metadata-changes-in-salesforce/)
- Explizit keine Rollback-Funktion: „should not replace a version control solution."

**Wer darf:**
- Profile und Permission Sets steuern Setup-Zugänge granular (z.B. „Modify Metadata" separate Permission).
- Admin-Rollen können feingranular kontrollieren, wer was konfigurieren darf.

---

### ④ Update-Sicherheit

Salesforces Kern-Mechanismus ist die **Metadata-Schichtenarchitektur**:

1. **Kernel-Layer:** Salesforce-eigene Runtime (shared, nicht kundenspezifisch).
2. **Standard-Metadata:** Salesforce-definierte Objekte und Felder.
3. **Kunden-Metadata:** Alles, was der Admin erstellt (Custom Objects, Fields, Flows, Page Layouts etc.) — pro Org isoliert.

**Wie Upgrades funktionieren:**
- Salesforce veröffentlicht 3 Major Releases pro Jahr (Spring, Summer, Winter).
- Kunden-Metadaten liegen in einem separaten Layer — Platform-Upgrades ändern den Kernel und Standard-Metadata, berühren aber Kunden-Metadaten **nicht direkt**.
- Wenn Salesforce ein Standard-Objekt erweitert (z.B. neues Feld), fügt es Metadaten hinzu, ändert aber keine Kunden-seitig existierenden Felder.
- **Risiko besteht** bei Flows, die auf Standard-Felder oder Standard-Verhalten referenzieren, das sich ändert — der Flow kann brechen (bekannt als „Flow Upgrade Problem").
- Formal gibt es keinen „Upgrade-Safe"-Begriff wie bei ServiceNow — aber die Metadata-Trennung sorgt strukturell für Isolierung.

[Quelle: Gearset Metadata API Versions](https://gearset.com/blog/regain-control-of-salesforce-metadata-api-versions/)
[Quelle: Salesforce Customization Best Practices 2026](https://blogs.emorphis.com/salesforce-customization-best-practices/)

---

### ⑤ Multi-Tenancy

Salesforce ist **klassisch multi-tenant** mit gemeinsamer Datenbank und Kernel:

- Alle Kunden-Daten liegen in **denselben physischen Tabellen** — getrennt durch `OrgId` (Tenant-Identifier).
- Jeder Datenbankzugriff wird durch den Kernel automatisch mit `WHERE OrgId = X` gefiltert.
- Custom Objects werden als **Universal Data Dictionary** (UDD) repräsentiert: statt eigener Tabellen pro Objekt gibt es generische Datentabellen mit Metadaten-gesteuerten Spalten.
- Customer Metadata (Field-Definitionen, Page Layouts, Flow-Konfigurationen) ist ebenfalls pro OrgId gespeichert.
- **Ergebnis:** Komplette Isolation auf Daten- und Metadaten-Ebene, obwohl alle Kunden dieselbe Infrastruktur teilen.

[Quelle: Salesforce Multi-Tenant Architecture](https://developer.salesforce.com/ja/wiki/multi_tenant_architecture)
[Quelle: Milestone Technologies Multi-Tenancy](https://milestone.tech/cloud-and-infrastructure/how-salesforce-masters-challenges-posed-by-multi-tenanted-systems/)

---

### ⑥ Vorlagen / Templates

- **AppExchange:** Salesforces Marktplatz bietet tausende vorgefertigte Pakete (Apps, Solutions, Starter Kits).
- **Industry Starter Kits** von Partnern (z.B. Field Service Starter Kit mit vorgefertigten Flows).
- **Branchen-Clouds:** Salesforce bietet eigenständige Produkte (Health Cloud, Financial Services Cloud, Manufacturing Cloud) — keine Templates, sondern komplett konfigurierte Org-Setups auf Basis-Salesforce.
- **Für normale Admins:** Kein natives „Branchen-Template beim Einrichten wählen" — das kommt über AppExchange-Partner-Lösungen oder Implementierungspartner.

[Quelle: AppExchange Solutions](https://appexchange.salesforce.com/)

---

### ⑦ Bekannte Fallen & Kritik

**Survey-Daten (Salesforce Ben Admin Survey 2025/2026):**
- **64,7%** der Admins: Salesforce wird zunehmend komplex.
- **56,3%** nennen **technische Schulden** als größtes Problem.
- **53,1%** sagen, von Admins wird zu viel erwartet.
- **42,86%** der Teams haben keine Entwickler — nur Admins.
- **19,47%** sind Solo-Admins ohne jedes Support-Netz.

[Quelle: Salesforce Ben Complexity Survey](https://www.salesforceben.com/is-salesforce-getting-too-complicated-heres-what-our-developer-survey-reveals/)
[Quelle: 10 Biggest Challenges 2026](https://www.salesforceben.com/the-biggest-challenges-for-salesforce-admins/)

**Konkrete Kritikpunkte:**

1. **Automation-Wildwuchs / Config-Hölle:**
   - Flows, Workflow Rules, Process Builder (veraltet), Apex Triggers — mehrere Automation-Systeme gleichzeitig im Einsatz.
   - Mehrere Flows auf demselben Objekt können **infinite loops** oder **Governor-Limit-Crashes** produzieren.
   - „A simple update to one process might break multiple automations across different objects" — ohne Dokumentation nicht vorhersehbar.
   [Quelle: 5 Hidden Risks](https://www.salesforceben.com/5-hidden-risks-of-too-much-salesforce-automation/)

2. **Governor Limits als versteckter Flaschenhals:**
   - Alle Automations teilen SOQL-Limits, DML-Limits, CPU-Zeit.
   - Komplexe Flows + Trigger + 3rd-Party-Apps können miteinander in Konflikt geraten.
   - Performance-Probleme sind schwer zu debuggen, weil die Ursachen verteilt sind.
   [Quelle: Salesforce Flow Limitations](https://success-craft.com/blog/salesforce-flow-limitations-and-how-to-avoid-them/)

3. **Audit Trail ohne Rollback:**
   - Kein nativer Rollback von Config-Änderungen.
   - Bei Page-Layout-Änderungen nicht mal ersichtlich, was geändert wurde.

4. **Komplexität wächst mit Nutzungsdauer:**
   - Quick Fixes akkumulieren → technische Schulden.
   - Orgs nach 5+ Jahren sind oft unwartbar ohne vollständige Neubewertung.
   - „Most Salesforce orgs do not become complex overnight. They accumulate layers of quick fixes, rushed automations, legacy fields, and abandoned processes over time."

5. **Setup ist separater Modus:**
   - Konfiguration passiert in einem komplett anderen Interface als die tägliche Nutzung.
   - Keine In-Kontext-Bearbeitung.

---

## 2. ServiceNow

### ① Admin-Grenze: Was ohne Code, was mit Entwickler?

**Ohne Code (Citizen Developer / Admin):**

- **Form Designer:** Drag-and-Drop-Formulargestaltung, Felder hinzufügen/entfernen/anordnen, Feldtypen wählen.
- **Flow Designer:** Visuelle End-to-End-Workflows. Keine Programmierung für Genehmigungsprozesse, Benachrichtigungen, bedingte Logik, Datensatz-Updates. Wiederverwendbare Subflows und Actions. [Quelle: ServiceNow Flow Designer](https://www.servicenow.com/products/platform-flow-designer.html)
- **UI Builder:** Konfiguration von App-Workspaces und Dashboards mit No-Code-Oberfläche.
- **Decision Builder:** Regeln-basierte Entscheidungslogik ohne Code.
- **App Engine Studio (AES):** Geführter App-Builder für Citizen Developer — erstellt komplette Scoped Applications aus Templates heraus, mit Drag-and-Drop-Komponenten. [Quelle: ServiceNow AES](https://www.servicenow.com/products/app-engine-studio.html)
- **Process Automation Designer:** Mehrstufige Prozess-Orchestrierung visuell.
- **Business Rules (einfache Varianten):** Bedingungen und Aktionen per GUI konfigurierbar.

**Grauzone (Low-Code, technisches Verständnis nötig):**
- Business Rules mit Scripting-Feldern: Admins mit JavaScript-Grundkenntnissen nötig.
- Client Scripts (Formularverhalten): Oft JavaScript.
- UI Policies können Komplexitäten erfordern.

**Erfordert Entwickler:**
- Custom Script Includes, komplexe Server-seitige Logik.
- REST/SOAP API-Integrationen mit komplexer Fehlerbehandlung.
- Scoped Application mit eigenen Tabellen und komplexer Architektur.
- DOM-Manipulation oder nicht-unterstützte UI-Eingriffe.

**Fazit Grenzlinie:** ServiceNow unterscheidet offiziell drei Ebenen: Citizen Developer (No-Code), Low-Code Developer, Pro Developer. AES ist explizit für Citizen Developers und Low-Code konzipiert — die Grenze liegt klarer als bei Salesforce, weil AES einen guardrail-gesicherten Scoped Sandbox bereitstellt.

[Quelle: Low-Code/No-Code in ServiceNow](https://www.servicenow.com/community/developer-blog/low-code-no-code-development-in-servicenow-accelerate-your-app/ba-p/3421760)

---

### ② Konfigurations-UI konkret

**App Engine Studio:**
- Geführter Wizard-basierter App-Builder.
- Schritt-für-Schritt: App-Name → Tabellen → Formulare → Workflows → UI.
- Drag-and-Drop-Komponenten für Formulare und Layouts.
- Vorschau-Modus während des Bauens.
- Klar strukturiertes Interface: links Navigation, Mitte Canvas, rechts Properties.
- Explizit auf Nicht-Entwickler ausgelegt.

**Flow Designer:**
- Visueller Canvas — ähnlich Salesforce Flow Builder, aber strukturierter.
- Elemente werden in einer Lanes-artigen Darstellung verbunden.
- Trigger → Bedingungen → Aktionen → Genehmigungen als klare Pipeline.
- Drag-and-Drop, vorkonfigurierte Actions aus der Bibliothek.
- Kein Code für Standard-Szenarien nötig.

**Form Designer:**
- Klassisches Drag-and-Drop-Formular-Layout.
- Feldtypen aus Palette links, Formular-Canvas rechts.
- Feldlevel-Bedingungen (sichtbar wenn X) konfigurierbar.

**Allgemeines:**
- Konfiguration passiert in der ServiceNow-Plattform selbst (nicht in einem separaten Admin-Portal).
- AES ist ein eigenes Modul innerhalb der Plattform.

[Quelle: Get started with AES](https://www.servicenow.com/community/s/cgfwn76974/attachments/cgfwn76974/app-engine-events/1/1/Get%20started%20with%20App%20Engine%20Studio.pdf)
[Quelle: Teiva Systems AES](https://teivasystems.com/app-engine-studio/)

---

### ③ Governance

**Sandbox-/Instanz-Strategie:**
- ServiceNow vergibt jedem Kunden **eine eigene dedizierte Instanz** (kein shared Database-Multitenancy wie Salesforce).
- Standard-Workflow: Development Instance → Test Instance → Production.
- Preview vor Production: Änderungen müssen durch Update Sets deployt und in Test-Instanz validiert werden, bevor sie live gehen.
- **Update Set Preview:** Vor dem Commit eines Update Sets in Produktion gibt es einen expliziten „Preview"-Schritt, der Konflikte anzeigt.

**Versionierung mit Update Sets:**
- Update Sets sind Container für Konfigurationsänderungen — ähnlich einem lokalen Git-Commit.
- Können zwischen Instanzen transferiert werden.
- Best Practice: Einzelne Update Sets pro Feature, Lead merged in Master Update Set.
- **Kein echtes Git** — aber strukturierter als Salesforces Change Sets.
- Best Practices (Teil 2): immer Preview vor Commit, Reihenfolge beim Merge beachten.
[Quelle: Update Set Leading Practices](https://www.servicenow.com/community/developer-blog/servicenow-update-set-leading-practices-part-2/ba-p/3257209)

**Audit & Nachvollziehbarkeit:**
- ServiceNow trackt alle Konfigurationsänderungen in der System Log-Infrastruktur.
- Jede Instanz hat eine vollständige Änderungshistorie.
- Rollback: Möglich durch Reverten eines Update Sets oder durch Vergleichs-Tool bei Upgrade-Konflikten.

**Wer darf:**
- Rolle `admin` für volle Setup-Rechte.
- `delegated_developer` für eingeschränkte Entwicklungsrechte.
- AES hat eigene Rollen: App Steward, Developer, Business Analyst, Governance Lead.
- IT kann granular steuern, wer in AES Apps erstellen darf (und welche Scopes).

[Quelle: Citizen Development Governance](https://www.servicenow.com/solutions/creator-workflows/citizen-development-program.html)
[Quelle: NCSU Developer vs Citizen Developer](https://ncsu.service-now.com/kb_view.do?sys_kb_id=d1e7c2f9c31b6994a9b22c6dc001315e)

---

### ④ Update-Sicherheit

ServiceNow löst Update-Sicherheit über ein klar definiertes **„You Touched It, You Own It"**-Prinzip und einen Erweiterungs-statt-Überschreibungs-Ansatz:

**Mechanismus:**
1. Wenn ein Admin/Entwickler ein Out-of-the-Box (OOB)-Record modifiziert (Business Rule, Client Script, Script Include), markiert ServiceNow diesen Record als „customized".
2. Bei einem Platform-Upgrade: Nicht-customisierte Records → werden automatisch mit dem neuen OOB-Stand überschrieben. Customisierte Records → werden in **Skipped Changes** eingetragen und dem Kunden zur Entscheidung vorgelegt.
3. Der Kunde muss aktiv entscheiden: Upgrade-Version übernehmen, eigene Version behalten, oder manuell mergen.

**Merge-Tool:**
- Links: neue OOB-Version, rechts: Kunden-Version.
- Pfeil-basiertes Merge-Interface zum selektiven Übernehmen von Änderungen.

**Upgrade-Safe-Techniken:**
- **Wrapper Classes:** Eigenes Script Include, das das OOB-Script Include via `Object.extendsObject()` erbt. Nur spezifische Methoden werden überschrieben. Das Original bleibt unangetastet — Konflikte bei Upgrades ausgeschlossen.
- **Extension Points nutzen:** Statt OOB-Flows überschreiben, eigene Sub-Flows als Extension Points einhängen.
- **UI Policies statt Client Scripts** bevorzugen (erstere sind deklarativer, weniger bruchgefährdet).
- **Flow Designer statt Legacy Workflows** (Flow Designer ist upgrade-stabiler).
- Keine Hard-coded `sys_id` Referenzen.
- Kein DOM-Hacking.

[Quelle: Upgrade Safe vs Upgrade Hostile](https://www.servicenow.com/community/itsm-forum/what-makes-a-servicenow-customization-upgrade-safe-vs-upgrade/td-p/3453622)
[Quelle: Wrapper Class Upgrade Safe](https://www.servicenow.com/community/developer-advocate-blog/upgrade-safe-modification-with-wrapper-class-in-servicenow/ba-p/3456773)
[Quelle: Kanini Customization vs OOTB](https://kanini.com/blog/servicenow-customization-out-of-the-box-strategy/)

---

### ⑤ Multi-Tenancy

ServiceNow nutzt ein **Multi-Instanz-Modell** — das Gegenteil von Salesforces klassischem Multi-Tenancy:

- Jeder Kunde bekommt eine **eigene dedizierte ServiceNow-Instanz** (eigene Datenbank: MariaDB, eigene Compute-Ressourcen, eigene Job Queues).
- Keine gemeinsam genutzten Datentabellen.
- **Konsequenz:** Absolute Datenisolation, aber auch höhere Infrastrukturkosten pro Kunde.

**Domain Separation (für Kunden mit eigenen Mandanten):**
- Wenn ein ServiceNow-Kunde selbst mehrere Mandanten (z.B. verschiedene Geschäftsbereiche oder Kunden in einem MSP-Szenario) betreiben will, gibt es Domain Separation.
- Jeder Record hat ein `Domain`-Feld.
- Abfragen filtern automatisch nach Domain-Zugehörigkeit.
- Hierarchische Domains (Eltern-Domain sieht Kind-Domain-Daten).

**Scoped Applications:**
- Technisches Isolations-Konstrukt für Apps innerhalb einer Instanz.
- Scoped App hat eigenen Namespace, eigene Tabellen, definierte API-Grenzen.
- Verhindert, dass eine App unkontrolliert auf Daten einer anderen App zugreift.

[Quelle: ServiceNow Domain Separation](https://kanini.com/blog/servicenow-domain-separation/)
[Quelle: Scoped Applications Guide](https://www.nowspectrum.com/blog/scoped-apps-guide)

---

### ⑥ Vorlagen / Templates

- ServiceNow bietet **Industry Solutions** für spezifische Branchen: Finanzdienstleistungen, Gesundheitswesen, Telekommunikation, öffentlicher Sektor, Einzelhandel, Fertigung.
- Diese sind vollständige, vorkonfigurierte App-Pakete — keine einfachen Starter-Templates, sondern komplette Branchenlösungen.
- Im Xanadu-Release (2025) wurde Now Assist in führende Industry Solutions integriert.
- **Für Citizen Developer in AES:** Templates für häufige App-Typen (Anfrage-Apps, Genehmigungsprozesse, Datenerfassung) als Startpunkt.
- ServiceNow Store (ähnlich AppExchange) für Drittanbieter-Lösungen.

[Quelle: ServiceNow Industries](https://www.servicenow.com/industries.html)
[Quelle: ServiceNow Industry Trends](https://corexcorp.com/insights/servicenow-industry-trends)

---

### ⑦ Bekannte Fallen & Kritik

**Upgrade-Risiko durch Über-Customization:**
- Das größte Risiko: Zu viele OOB-Modifikationen → jedes Upgrade wird zu einem Merge-Marathon.
- „Overcustomization can make it difficult to stay updated with the latest releases, and future upgrades tend to become hectic, time-consuming, and may hinder the implementation of other modules."
- Instanzen, die zu weit vom OOB-Zustand abweichen, verlieren effektiv ServiceNows Support für Upgrades.
[Quelle: Kanini Customization Strategy](https://kanini.com/blog/servicenow-customization-out-of-the-box-strategy/)

**Performance-Risiken bei Update Sets:**
- Das `update_synch=true`-Attribut auf einer Tabelle zu setzen (nicht OOB) ist „extremely dangerous": kann zu schwerer Performance-Degradation und Instanz-Ausfall führen.
- Update Sets haben keine inhärente Abhängigkeitsauflösung — falsche Reihenfolge beim Deployen bricht Konfiguration.

**Governance-Lücken in AES:**
- Ohne definierte Review-Gates können Apps außerhalb der Enterprise-Architektur entstehen.
- Sensible Daten können ohne angemessene Kontrollen in Citizen-Developer-Apps landen.
- „Without review gates, apps might not align with enterprise architecture."

**Lizenzkosten AES:**
- App Engine Studio erfordert eine eigene Lizenz — nicht in jeder ServiceNow-Edition enthalten.
- Skalierung von Citizen-Developer-Programmen ist kostenintensiv.
[Quelle: AES License Forum](https://www.servicenow.com/community/app-engine-forum/you-need-a-valid-app-engine-studio-license-to-continue-working/td-p/3362958)

**Komplexität der Instanz-Verwaltung:**
- Drei Instanzen (Dev, Test, Prod) zu pflegen ist ressourcenintensiv.
- Viele KMU-ähnliche Organisationen überfordern das Operations-Modell.

---

## 3. Übertragbare Muster + Anti-Muster für Cosmi

### Übertragbare Muster (bauen!)

**Muster 1 — Schichten-Architektur für Update-Sicherheit (kritischstes Muster):**
ServiceNow macht es mit Wrapper Classes richtig: Kunden-Konfiguration liegt **immer als separater Layer** über dem Produktcode, nie direkt im Produktcode. Für Cosmi bedeutet das:
- Custom Fields, Custom Labels, Layout-Config, Workflow-Config: alles in eigenen `tenant_customizations`-Tabellen (pro Tenant isoliert via `tenant_id`).
- Produktcode liest zur Laufzeit: erst Tenant-Customization, dann Cosmi-Default (Overlay-Prinzip).
- Ein Cosmi-Release ändert nie die Kunden-Customization-Tabellen — nur Default-Werte und Kernel-Code.
- Das Muster existiert bereits ansatzweise: RBAC nutzt `USER_OVERRIDES` als Overlay über Baseline-Permissions. Exakt dasselbe Prinzip auf Config erweitern.

**Muster 2 — Global Value Sets / Zentralisierte Wertelisten:**
Salesforce Global Value Sets zeigen den richtigen Ansatz: Eine Werteliste (z.B. „Kontaktstatus") zentral verwalten, mehrere Felder referenzieren dieselbe Liste. Ändert der Admin die Liste, propagiert es überall.
Für Cosmi: `tenant_value_sets`-Tabelle mit zentralen Wertelisten, die in Custom Fields und Standard-Picklists referenziert werden. Kein Copy-Paste von Listen per Modul.

**Muster 3 — Citizen Developer mit expliziter Governance-Schicht:**
ServiceNow AES zeigt, dass man Citizen Developer ermächtigen UND IT-Oversight behalten kann durch:
- Klare Rollen-Ebenen (was darf wer konfigurieren).
- Review-Gate vor Go-Live (Config-Änderungen können staged sein, bevor sie produktiv werden).
- Audit Trail mit vollständiger Nachvollziehbarkeit wer was wann geändert hat.
Das RBAC-Audit-System (`audit-events.ts`) ist die Blaupause — auf Config-Änderungen anwenden.

---

### Anti-Muster (bewusst NICHT bauen)

**Anti-Muster 1 — Automation-Wildwuchs ohne zentrale Governance:**
Salesforces größtes Schmerzkind: Flows, Triggers, Process Builders — mehrere Systeme für dasselbe Problem, die sich gegenseitig in die Quere kommen. 56,3% der Admins leiden darunter.
Für Cosmi: EIN Workflow-Konzept, klar begrenzt in Mächtigkeit (kein Apex-Äquivalent für Endkunden). Lieber weniger Features, die sicher zusammenspielen, als ein vollständiger Workflow-Builder, den KMU-Admins nicht beherrschen.

**Anti-Muster 2 — Config-UI als separater Admin-Modus:**
Beide Plattformen trennen Setup von täglicher Nutzung (Salesforce Setup-Zahnrad, ServiceNow Admin-Bereich). Das erzeugt mentale Kontextwechsel und macht Änderungen abstrakt.
Für Cosmi: Config so nah wie möglich am Kontext. In-Modul-Einstellungen (bereits vorhanden via `ModuleSettingsShell`), In-Place-Rename von Labels (Terminology-Override direkt am Element), nicht nur unter einem globalen „Admin"-Bereich.

**Anti-Muster 3 — Unbegrenzte Mächtigkeit = unbegrenzte Schulden:**
Je mächtiger das Config-System, desto mehr kumulieren technische Schulden. Salesforce zeigt: nach 5 Jahren ist eine typische Org kaum noch wartbar. Quick Fixes akkumulieren als Legacy-Config.
Für Cosmi: Für KMU bewusst den Scope begrenzen. Felder, Terminologie, Wertelisten, Layout-Priorisierung, einfache Regeln — das reicht für 5–200-MA-Kunden. Komplexe Workflow-Automations (wenn-dann-dann-sonst-Bäume mit verschachtelter Logik) sind WASM-Plugin-Territorium (Feature-Flag OFF bis Phase D), nicht No-Code-Config-Territorium.

---

### ④ Update-Sicherheit: Kritische Erkenntnisse

**ServiceNow-Mechanismus (Referenz für Cosmi):**

```
Platform-Code (Cosmi-Release)
    ↓
Tenant-Customization-Layer (read: Overlay über Default)
    = tenant_id + customization_key + customization_value
    
Cosmi-Default-Layer (fallback wenn kein Override)
    = product_default_key + default_value
```

- Kunden-Anpassungen überleben Cosmi-Updates **automatisch**, weil sie in eigenen Tabellen liegen.
- Ein Cosmi-Update ändert nie `tenant_customizations.*` — es ändert nur Default-Werte und Kernel.
- „Konfliktfall": Cosmi entfernt ein Feature, auf dem eine Kunden-Konfiguration aufbaut → Migration-Script nötig, das Kunden-Config-Einträge umschreibt. Dies ist der einzige Aufwands-Fall.
- ServiceNow nennt das „You Touched It, You Own It" — für Cosmi wäre die Richtung umgekehrt: „Wir versprechen, deine Config niemals anzufassen".

**Salesforce-Mechanismus (ergänzend):**
- Kunden-Metadaten (Custom Objects, Flows, etc.) sind physisch von Salesforce-Kernel-Metadaten getrennt durch OrgId-Isolation.
- Salesforce-Releases ändern Standard-Metadaten, nie Kunden-Metadaten.
- Risiko besteht nur bei indirekten Abhängigkeiten (Flow referenziert Standard-Feld, das Salesforce umbenennt) → Kompatibilitäts-Versprechen via API-Versionierung.

**Fazit für Cosmi:** Der Overlay-Ansatz ist der richtige Weg. Kunden-Config = eigene Tabelle. Produkt-Default = separate Quelle. Runtime: Overlay zusammenführen. Das ist das technisch sauberste Muster aus beiden Enterprise-Plattformen.

---

*Quellen gesammelt:*
- https://www.salesforcelwc.com/salesforce-object-manager-explained-fields-page-layouts-and-relationships/
- https://trailhead.salesforce.com/content/learn/modules/lightning_app_builder/lightning_app_builder_intro
- https://trailhead.salesforce.com/content/learn/modules/lex_customization/lex_customization_page_layouts
- https://www.default.com/post/salesforce-flow-building-visual-workflows-in-salesforce
- https://www.salesforceben.com/global-picklists-in-salesforce-explained/
- https://www.salesforceben.com/custom-metadata-types/
- https://www.salesforceben.com/is-salesforce-getting-too-complicated-heres-what-our-developer-survey-reveals/
- https://www.salesforceben.com/the-biggest-challenges-for-salesforce-admins/
- https://www.salesforceben.com/5-hidden-risks-of-too-much-salesforce-automation/
- https://www.salesforceben.com/setup-audit-trail-keep-track-of-metadata-changes-in-salesforce/
- https://success-craft.com/blog/salesforce-flow-limitations-and-how-to-avoid-them/
- https://gearset.com/blog/regain-control-of-salesforce-metadata-api-versions/
- https://developer.salesforce.com/ja/wiki/multi_tenant_architecture
- https://milestone.tech/cloud-and-infrastructure/how-salesforce-masters-challenges-posed-by-multi-tenanted-systems/
- https://appexchange.salesforce.com/
- https://www.servicenow.com/products/app-engine-studio.html
- https://www.servicenow.com/products/platform-flow-designer.html
- https://teivasystems.com/app-engine-studio/
- https://www.servicenow.com/community/itsm-forum/what-makes-a-servicenow-customization-upgrade-safe-vs-upgrade/td-p/3453622
- https://www.servicenow.com/community/developer-advocate-blog/upgrade-safe-modification-with-wrapper-class-in-servicenow/ba-p/3456773
- https://www.servicenow.com/community/developer-blog/servicenow-update-set-leading-practices-part-2/ba-p/3257209
- https://kanini.com/blog/servicenow-customization-out-of-the-box-strategy/
- https://www.nowspectrum.com/blog/scoped-apps-guide
- https://kanini.com/blog/servicenow-domain-separation/
- https://www.servicenow.com/community/developer-blog/low-code-no-code-development-in-servicenow-accelerate-your-app/ba-p/3421760
- https://www.servicenow.com/solutions/creator-workflows/citizen-development-program.html
- https://www.servicenow.com/industries.html
- https://blogs.emorphis.com/salesforce-customization-best-practices/
