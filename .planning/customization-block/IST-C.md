# IST-Analyse C — Präsentation & Integration
**Scope:** Terminologie/i18n, Branding/Theme, Dokument-/Report-Vorlagen, Onboarding, Integrationen  
**Stand:** 2026-07-21  
**Analysierter Code:** `desktop/src/renderer/src/`

---

## 1. Terminologie / Labels / i18n — Dimension B ★ (KERNFALL)

### Architektur heute (exakt)

Die gesamte i18n-Infrastruktur liegt in `desktop/src/renderer/src/i18n/`:

| Datei | Rolle |
|---|---|
| `i18n/i18n.ts` | Initialisierung: 4 statische JSON-Imports, kein Backend-Loader |
| `i18n/messages/de.json` | Source of Truth, 7.221 Keys |
| `i18n/messages/en.json` | 7.221 Keys (EN) |
| `i18n/messages/fr.json` | 7.221 Keys (FR) |
| `i18n/messages/it.json` | 7.221 Keys (IT) |
| `i18n/i18next.d.ts` | Strict TS-Types gegen `de.json` |
| `i18n/__tests__/plural.test.ts` | ICU-Plural-Regression |

**Initialisierung (i18n.ts, Zeile 26–41):**
```ts
.init({
  lng: locale,
  fallbackLng: 'de',
  keySeparator: false,   // flat dot-notation: "crm.contacts.title"
  nsSeparator: false,    // kein Namespace-Splitting
  resources: {
    de: { translation: messagesDE },
    en: { translation: messagesEN },
    fr: { translation: messagesFR },
    it: { translation: messagesIT },
  },
})
```

Alle 4 Locale-Bundles werden beim App-Start **statisch gebundelt** (Vite-Build-Zeit). Es gibt **keinen i18next-Backend-Plugin**, keinen HTTP-Loader, kein Lazy-Loading. Die i18next.d.ts bindet den TypeScript-Compiler zur Compile-Zeit an `de.json` — jeder ungültige Key ist ein TS-Fehler.

**Key-Format:** `modul.bereich.label` (Beispiele: `crm.contacts.title`, `admin.branding.title`, `settings.integrations.livekit.description`). Flat, kein Nesting, kein Namespace-Separator.

**Sprachpräferenz-Persistenz:** User-Locale wird in `stores/locale.ts` (Zustand + localStorage `cosmi-locale`) gespeichert und per PUT `/api/v1/settings/language/user` server-seitig synchronisiert.

### Kann ein Kunde heute Begriffe umbenennen?

**NEIN — vollständig unmöglich.** Es gibt:
- Keine Runtime-Override-Schicht
- Keine Backend-API für tenant-spezifische Key-Overrides
- Keine Datenbank-Tabelle für Label-Überschreibungen
- Keinen i18next-`postProcessor` oder `missingKeyHandler` der auf Tenant-Daten zurückfallen könnte

Die JSON-Dateien sind Build-Artefakte; ein Kunde hat keinen Zugang dazu.

### Wie könnte eine Label-Override-Schicht andocken?

i18next bietet zwei saubere Einstiegspunkte:

**Option A — i18next Backend-Plugin (empfohlen für v1):**  
Ein `i18next-backend`-artiger Custom-Plugin lädt nach der Initialisierung tenant-spezifische Overrides nach:
```ts
// Pseudocode — nach init(), tenant bereits bekannt:
const overrides = await fetchTenantLabelOverrides(tenantId)
i18n.addResourceBundle('de', 'translation', overrides, true, true)
// drittes Arg = deep merge, viertes = overwrite existing
```
Das `true, true`-Muster überschreibt nur die Keys die der Tenant definiert hat; alle anderen bleiben unverändert. Kein Rebuild nötig. Timing: nach `initI18n()` und nach Auth/Tenant-Resolve.

**Option B — i18next `postProcessor`:**  
Ein PostProcessor-Plugin fängt jeden `t()`-Call ab und prüft gegen eine In-Memory-Map der Tenant-Overrides. Overhead pro Render-Cycle; schlechter als Option A.

**Datenhaltung (Backend):**  
Neue Tabelle `tenant_label_overrides (tenant_id, locale, key, value)` — flach, ein Row pro Key×Locale-Kombination. Abruf per `GET /api/v1/tenant/labels` nach Login. Kandidat für Redis-Caching (ändern sich selten).

**Schwierigkeitsgrad:** Mittel. Die i18next-Infrastruktur ist bereits sauber; das Andock-Pattern (`addResourceBundle`) ist dokumentiert und getestet. Hauptarbeit: Backend-Tabelle + API-Endpoint + UI zum Editieren der Labels. Die 7.221 Keys müssen nicht alle exponierbar sein — für v1 genügt eine kuratierte Whitelist (Modul-Namen, Objekt-Namen wie "Kontakte", "Deals", Status-Labels).

**TS-Typen-Implikation:** Der strict-type-check von `i18next.d.ts` validiert nur statisch bekannte Keys. Runtime-Overrides liegen außerhalb des TS-Typsystems — das ist per Design korrekt (Overrides sind dynamisch).

---

## 2. Branding / Theme — Dimension K

### Was existiert heute

**BrandingAdminHubTab** (`modules/admin/branding/BrandingAdminHubTab.tsx`):
- Workspace-Name (Freitext, Default "Cosmi")
- Logo-Upload (Bild-Datei ≤ 512 KB, via FileReader → base64)
- Icon-Upload (quadratisch, gleiche Limits)
- Akzentfarbe (Picker aus `lib/swatch-colors.ts` — 10 Farben fest kodiert: grau, blau, cyan, grün, amber, orange, rot, lila, pink, indigo)

**Persistenz:** `localStorage` mit Keys `cosmi:brand:name`, `cosmi:brand:logo`, `cosmi:brand:icon`, `cosmi:brand:accent`. Das ist **Demo-Persistence** (Kommentar im Code: "Persisted to localStorage as a mock; the real logo upload (→ S3) is Luke's track"). Backend-Sync fehlt vollständig.

**Accent-CSS-Variable:** `document.documentElement.style.setProperty('--brand-accent', accent)` — wird live angewendet, aber nicht systemweit konsistent konsumiert (nur BrandingAdminHubTab selbst nutzt die Variable explizit).

**RBAC:** `admin:branding:manage` ist als Capability-Key registriert (`config/capabilities.ts:162`, `config/capability-catalog.ts:427`). Der AdminHubPage-Router prüft diesen Key als Gate für den Tab.

**Desk-Theme-System** (`types/desk-theme.ts`, `config/desk-themes.ts`):
- 5 vordefinierte Themes: cozy, dreamy, raumstation, clean, minimal
- 5-Layer-Architektur: Room Scene / Furniture / Decorations / Mount Points / UI Skin
- L5 (UI Skin) = CSS-Variable-Overrides für card-bg, border, shadow, sidebar
- Rein code-seitig definiert; kein In-App-Editor für Themes; kein Tenant-spezifisches Theme-Override

**Statische Branding-Assets** (`config/branding.ts`):  
Cosmi-, Zentria- und Orbit-Logos sind als statische Imports gebundelt — nicht tenant-überschreibbar; werden von Produktkomponenten (Login, Sidebar, About) importiert.

### Lücken

- Logo-Upload schreibt nur in localStorage, nicht auf S3/CDN/Backend
- Akzentfarbe ist auf 10 Swatches begrenzt; kein freier Hex-Picker
- `--brand-accent` wird nicht konsistent im Design-System konsumiert (die meisten Komponenten nutzen `--primary`, nicht `--brand-accent`)
- Desk-Themes sind nicht tenant-konfigurierbar; ein Tenant kann keinen "Firmen-Standard-Theme" erzwingen
- Font ist nicht konfigurierbar (Font-Bann in CLAUDE.md; aktuell hardcoded Plus Jakarta Sans / Satoshi)
- Kein Dark/Light-Mode-Override pro Tenant

---

## 3. Dokument-/E-Mail-/Report-Vorlagen — Dimension J

### Block-Engine (shared/document)

Die Engine liegt in `components/shared/document/` und ist **Phase A fertig**:

| Datei | Rolle |
|---|---|
| `types.ts` | Datenmodell: DocRow → DocColumn → DocBlockBase |
| `block-registry.ts` | `BlockTypeDef`: icon, labelKey, makeDefault, Edit, View |
| `DocumentBlockEditor.tsx` | Drag-and-Drop-Editor (dnd-kit) |
| `DocumentReader.tsx` | Read-only Renderer mit Print-Modus und Accent-Color |
| `blocks/CoreBlocks.tsx` | heading, text, bullet, callout, divider, image |
| `blocks/SpecialBlocks.tsx` | toggle, code, attachment, quote, bookmark |

Das Datenmodell ist module-agnostisch. Neue Block-Typen entstehen durch `defineBlock()` + Eintrag im Registry — kein Engine-Code nötig.

**Berichte (Report-Dokument-Schicht):** `modules/berichte/components/documents/berichte-blocks.tsx` erweitert die Engine um cover, KPI, chart, table, page-break. `ReportDocumentEditor.tsx` baut auf `DocumentBlockEditor`/`DocumentReader` auf.

**Report-Sources-Registry:** `modules/berichte/report-sources/registry.ts` — 11 Quellen (finanzen, kontakte, work, helpdesk, kommunikation, hr, zeiterfassung, vertraege, einkauf, fuhrpark, rapporte). Jede Quelle hat typisierte `FieldDefinition[]` mit `labelKey` und `dataType`. Der Report-Builder (`ReportBuilderShell`) erlaubt No-Code-Zusammenstellung von Charts/Tabellen.

### Was heute möglich ist

Ein **authentifizierter User** (mit Berichte-Modul-Zugang) kann:
- Beliebige Report-Dokumente aus Block-Typen zusammenbauen
- Datenquellen auswählen und Felder konfigurieren (FieldPicker, VizSwitcher)
- Berichte planen (`ScheduleReportModal`) und teilen (`ShareActionsMenu`)
- An Tasks, Kontakte, Dokument-Bibliothek anhängen

### Was NICHT möglich ist

- **Merge-Fields / Template-Platzhalter:** Es gibt keine `{{Kundenname}}`, `{{Rechnungsnummer}}`-Syntax oder Field-Variable-Insertion in Text-Blöcken. TipTap-HTML (`TextBlock`) ist freier Richtext ohne datenbindende Platzhalter.
- **E-Mail-Vorlagen:** Kein E-Mail-Template-System (kein Modul, keine Block-Typen dafür). E-Mail wird über das Kommunikations-/Mail-Modul abgewickelt, aber ohne No-Code-Vorlagen-Verwaltung.
- **Angebots-/Rechnungs-Layout-Vorlagen:** Das Finanzen-Modul generiert Rechnungen, aber das Layout ist code-seitig fixiert (kein Block-System). Kunden können Layout nicht anpassen.
- **Tenant-Vorlagen-Bibliothek:** Keine Möglichkeit, "Unternehmens-Standard-Berichte" als Vorlage zu hinterlegen (alle Berichte sind user-owned, nicht tenant-owned).
- **Datenquellen-Kaskade (Phase B):** Laut Memory-Notiz noch offen — Wiki + kaskadierende Datenquellen-Auswahl gemeinsam.

---

## 4. Onboarding — Überschneidung

### IST-Zustand

Kein Onboarding-Flow, kein Setup-Wizard, kein First-Run-Screen existiert im Code (`modules/onboarding*` — kein Fund, `wizard*` — kein Fund).

O-0 (Onboarding) ist laut Laufender-Sprint-Block das nächste große Deliverable nach dem RBAC-Block. Es ist vollständig geplant aber noch nicht gebaut.

**Relevanz für Customization:**  
Das geplante Onboarding wird vermutlich auf denselben Konfigurations-Primitiven aufbauen, die das Self-Service-Customization-Tool braucht:
- Branding-Setup (Workspace-Name, Logo, Accent)
- Integrations-Anbindung (Buchhaltung, Kommunikation)
- Modul-Aktivierung / RBAC-Rollen-Vergabe
- Label-Overrides (wenn Customization-Tool v1 fertig ist)

Ein Shared-Config-Layer zwischen Onboarding und Self-Service-Tool ist architektonisch sinnvoll — Onboarding = geführter Erstkonfigurations-Fluss über dieselben Einstellungs-APIs.

---

## 5. Integrationen — In-App-Anbindung

### IST-Zustand

**Integration-Registry** (`modules/settings/integrations/integration-registry.ts`):  
12 Integrationen in 5 Kategorien:
- **Buchhaltung:** DATEV Rechnungswesen (`panelType: 'custom'`), Bexio (OAuth2, `panelType: 'custom'`)
- **Kommunikation:** Slack (existing), Teams Webhook (existing), Custom Webhook (existing), Teams Graph (API-Key, Felder: tenantId/clientId/clientSecret), WhatsApp Business (server, Felder: apiUrl/phoneNumber/accessToken)
- **Dokumente:** Skribble (API-Key), Collabora Online (server, serverUrl/wopiDiscoveryUrl)
- **Video:** Zoom (OAuth2, Felder: autoRecord/waitingRoom/defaultDuration), LiveKit (server, serverUrl/apiKey/apiSecret)
- **Marketing:** Brevo (API-Key, Felder: apiKey/listMapping/syncEnabled)

**Generic-Panel-Integrationen** (teams-graph, whatsapp, skribble, collabora, zoom, livekit, brevo): Der Kunde gibt Credentials/Keys selbst ein (UI vorhanden). Felder sind `password`/`text`/`url`/`switch`/`select`-Typen.

**Custom-Panel-Integrationen** (DATEV, Bexio): Eigene Komponenten — Details in separaten Dateien (nicht in scope dieser Analyse, aber DATEV hat `panelType: 'custom'` = vollständige Eigen-UI).

**Persistenz:** `stores/integrations.ts` (Zustand + persist, localStorage `cosmi-integrations`). Demo-Status: `triggerSync()` nutzt `setTimeout(1500)`. Kein Backend-Write für Credentials — das ist Mock-first.

**RBAC:** `admin:integrations:manage` als Capability-Key registriert (`config/capabilities.ts:113`/`159`, `config/capability-catalog.ts:428`). AdminHubPage prüft diesen Key.

**Admin-Hub-Tab:** `modules/admin/tabs/IntegrationsAdminHubTab.tsx` — aktuell **EmptyState** ("Bexio/DATEV/Lexware sind direkt im Buchhaltungsmodul; Slack/Zapier/Webhooks landen hier in einer späteren Phase"). Der Integration-Settings-Tab ist also der eigentliche Ort für tenant-weite Integrations-Konfiguration, während admin/integrations noch ein Platzhalter ist.

### Was heute möglich ist (In-Cosmi-UI)

- Credentials/API-Keys eingeben (UI vorhanden, Demo-Persistenz in localStorage)
- Connect/Disconnect/Sync-Trigger (UI + Store, aber Mock-Backend)
- OAuth2-Flow für Bexio und Zoom (Custom-Panels; echter Backend-Flow: Lukes Track)

### Was NICHT möglich ist

- Kein echter Backend-Credential-Write (nur localStorage)
- Keine Tenant-weite Integration-Policy (z.B. "alle User nutzen dieselbe Slack-Workspace")
- Kein Self-Service-OAuth-Callback ohne Backend-Endpunkt
- Admin-Hub Integrations-Tab ist EmptyState — zentrale tenant-weite Sicht fehlt

---

## Übersichtstabelle

| Dimension | Existiert? | Wo (Pfad) | In-Cosmi-UI oder Code/Deploy | Wer darf (RBAC-Key) | Lücke |
|---|---|---|---|---|---|
| **B — Terminologie/Labels** | Nein | `i18n/messages/*.json` | Code/Build-Zeit | — | Runtime-Override-Schicht komplett fehlend; keine Backend-API, keine DB-Tabelle, kein i18next-Plugin |
| **K — Branding (Logo/Farbe)** | Teilweise (Mock) | `modules/admin/branding/BrandingAdminHubTab.tsx` | In-Cosmi-UI (Admin-Tab) | `admin:branding:manage` | Persistenz nur localStorage; kein S3/Backend; Accent-Variable nicht systemweit konsumiert; nur 10 Swatches |
| **K — Desk-Themes** | Ja (5 Themes) | `config/desk-themes.ts`, `types/desk-theme.ts` | In-Cosmi-UI (persönliche Einstellung) | persönlich (kein tenant-RBAC) | Themes sind nicht tenant-konfigurierbar; kein Firmen-Standard-Theme |
| **K — Statische Branding-Assets** | Ja (fest) | `config/branding.ts` | Code/Deploy | — | Nicht überschreibbar; nur via Rebuild |
| **J — Block-Dokument-Engine** | Ja (Phase A fertig) | `components/shared/document/` | In-Cosmi-UI (Berichte, Wiki) | Modul-spezifisch | Keine Merge-Fields/Platzhalter; keine Tenant-Vorlagen-Bibliothek |
| **J — Report-Builder** | Ja | `modules/berichte/` | In-Cosmi-UI | `berichte:module:view` + Berichte-Capabilities | Keine Angebots-/Rechnungs-Layout-Vorlagen; kein E-Mail-Template-System |
| **J — E-Mail-Vorlagen** | Nein | — | — | — | Komplett fehlend |
| **J — Rechnungs-/Angebots-Layout** | Nein | — | — | — | Layout code-seitig fixiert; kein No-Code-Override |
| **Onboarding-Flow** | Nein (geplant O-0) | — | — | — | Noch nicht gebaut; teilt Primitiven mit Customization-Tool |
| **Integrationen (generic)** | Ja (UI + Demo-Store) | `modules/settings/integrations/` | In-Cosmi-UI (Settings-Tab) | `admin:integrations:manage` | Credentials nur localStorage; kein echter Backend-Write; kein OAuth-Callback |
| **Integrationen (custom: DATEV/Bexio)** | Teilweise | `modules/admin/` (custom panels) | In-Cosmi-UI (Admin-Tab) | `admin:integrations:manage` | OAuth-Flow Backend fehlt (Lukes Track) |
| **Integrationen (Admin-Hub-Tab)** | Nein (EmptyState) | `modules/admin/tabs/IntegrationsAdminHubTab.tsx` | — | `admin:integrations:manage` | Tenant-weite Integrations-Übersicht komplett fehlend |

---

## Quell-Dateien (Referenz)

- `desktop/src/renderer/src/i18n/i18n.ts`
- `desktop/src/renderer/src/i18n/messages/de.json` (7.221 Keys, nicht vollständig gelesen)
- `desktop/src/renderer/src/stores/locale.ts`
- `desktop/src/renderer/src/modules/admin/branding/BrandingAdminHubTab.tsx`
- `desktop/src/renderer/src/config/branding.ts`
- `desktop/src/renderer/src/lib/swatch-colors.ts`
- `desktop/src/renderer/src/types/desk-theme.ts`
- `desktop/src/renderer/src/config/desk-themes.ts`
- `desktop/src/renderer/src/components/shared/document/types.ts`
- `desktop/src/renderer/src/components/shared/document/block-registry.ts`
- `desktop/src/renderer/src/components/shared/document/DocumentBlockEditor.tsx`
- `desktop/src/renderer/src/components/shared/document/DocumentReader.tsx`
- `desktop/src/renderer/src/components/shared/document/blocks/CoreBlocks.tsx`
- `desktop/src/renderer/src/components/shared/document/blocks/SpecialBlocks.tsx`
- `desktop/src/renderer/src/modules/berichte/report-sources/registry.ts`
- `desktop/src/renderer/src/modules/berichte/report-sources/types.ts`
- `desktop/src/renderer/src/modules/berichte/components/documents/berichte-blocks.tsx`
- `desktop/src/renderer/src/modules/berichte/components/documents/ReportDocumentEditor.tsx`
- `desktop/src/renderer/src/modules/settings/integrations/integration-registry.ts`
- `desktop/src/renderer/src/stores/integrations.ts`
- `desktop/src/renderer/src/modules/admin/tabs/IntegrationsAdminHubTab.tsx`
- `desktop/src/renderer/src/config/capabilities.ts`
- `desktop/src/renderer/src/config/capability-catalog.ts`
- `.knowledge/i18n.md`
- `.knowledge/design.md`
