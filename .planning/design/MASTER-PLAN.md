# MASTER-PLAN -- Frontend-Gesamtplan

> Definitive Zusammenfuehrung aus `14-frontend-plan.md` (Implementation Plan) und `06-modul-gap-analyse.md` (Gap-Analyse).
> Autor: Darien | Datum: 2026-02-17 | Branch: `design/brainstorm`

---

## Uebersicht

| Kennzahl | Wert |
|----------|------|
| **Geschaetzter Gesamt-LOC** | ~55.000-62.000 |
| **Waves (Arbeitspakete)** | 14 |
| **Neue Module** | 2 (Kommunikation, Wiki) |
| **Erweiterte Module** | 22 |
| **Neue Shared Components** | 6 |
| **Neue Zustand Stores** | 7 |
| **Neue Dateien** | ~160+ |
| **Frontend-only Items** | ~75% |
| **Backend-abhaengige Items** | ~25% |
| **Timeline (geschaetzt)** | 20-26 Wochen |

### LOC-Aufschluesselung nach Quelle

| Quelle | LOC |
|--------|-----|
| Frontend Implementation Plan (14-frontend-plan.md) | ~18.000-22.000 |
| Gap-Analyse Erweiterungen (06-modul-gap-analyse.md) | ~25.000-35.000 |
| Querschnitts-Aenderungen (Q1-Q5) | ~3.000-5.000 |
| **Gesamt (mit Iteration/Polish)** | **~55.000-62.000** |

---

## Office-Integration: 3-Tier-Strategie

> Entscheidung vom 2026-02-17. Gilt fuer alle Tiers.

### Grundprinzip

**"In Word/Excel/PowerPoint oeffnen" ist in ALLEN Tiers enthalten.** Wer Office lokal installiert hat, kann immer daraus arbeiten. Der Tier-Unterschied betrifft nur das Bearbeiten IM BROWSER ohne lokale Installation.

### Tier-Uebersicht

| Tier | Preis | Lokales Office (Word/Excel/PPT) | Browser-Editor |
|------|-------|--------------------------------|----------------|
| **Starter (12 EUR)** | Guenstig | Ja (immer) | Keiner — nur Dateiverwaltung |
| **Business (19 EUR)** | Mittel | Ja (immer) | TipTap (nur Texte, Wiki, E-Mails) |
| **Enterprise (25 EUR)** | Premium | Ja (immer) | Collabora Online (Word + Excel + PPT im Browser) |

### Tier 1 — Lokales Office (alle Tiers)

**Funktion:** Dateien in KMU Hub speichern, teilen, organisieren. Zum Bearbeiten "In Word oeffnen" klicken → lokales Programm oeffnet sich → bei Speichern wird Datei zurueck synchronisiert.

**Technische Umsetzung (Frontend):**
- "Oeffnen in Word/Excel/PowerPoint" Button im Dokumente-Modul
- Electron: `shell.openPath()` oeffnet lokales Programm
- `fs.watch()` FileWatcher ueberwacht temp-Datei auf Aenderungen
- Bei Aenderung: automatischer Re-Upload in den KMU Hub Speicher
- Dateien: `modules/dokumente/LocalOfficeOpener.tsx` (~200 LOC), Aenderungen an `DokumentePage.tsx` (+50 LOC)
- Tag: `[FE-ONLY]`

**Technische Umsetzung (Backend — Luke):**
- WebDAV-Server damit Office direkt von Server oeffnen/speichern kann (kein Download noetig)
- Versionierung: jede Speicherung = neue Version
- Konflikterkennung bei gleichzeitiger Bearbeitung

**Lizenz:** Keine — wir oeffnen nur Dateien mit dem Programm das der Kunde bereits hat.

### Tier 2 — TipTap Browser-Editor (Business+)

**Funktion:** Einfacher Rich-Text-Editor direkt im Browser fuer Wiki, E-Mails, Notizen, Helpdesk-Artikel. KEIN Spreadsheet, KEINE Praesentationen.

**Was TipTap kann:** Bold/Italic/Headings, Listen, Tabellen (einfach), Bilder, Links, Code-Bloecke, Aufgabenlisten, @Mentions, Echtzeit-Zusammenarbeit (via Hocuspocus, self-hosted, kostenlos).

**Was TipTap NICHT kann:** .docx/.xlsx-Editing, Tabellenkalkulation/Formeln, Praesentationen, Track Changes, komplexe Seitenlayouts.

**Lizenz:** MIT (kostenlos, auch kommerziell). Hocuspocus (Echtzeit-Collab) ebenfalls MIT. Fuer uns: 0 EUR Kosten.

**DOCX Import/Export:** Moeglich ueber TipTap Cloud (ab Start-Plan, kostenlos fuer bis zu 500 Dokumente). Alternativ: eigener Converter mit `mammoth.js` (MIT) fuer .docx-Import und `docx` npm-Package fuer Export.

### Tier 3 — Collabora Online (Enterprise)

**Funktion:** Volle Office-Suite im Browser. Word, Excel UND PowerPoint direkt in KMU Hub bearbeiten, ohne irgendwas lokal zu installieren. Echtzeit-Zusammenarbeit, Kommentare, Versionsverlauf.

**Features:**
- Writer: Volle Textverarbeitung, Styles, Header/Footer, Change Tracking, Kommentare
- Calc: Tabellenkalkulation, Formeln, Pivot-Tabellen, Diagramme, bis 16.000 Spalten
- Impress: Praesentationen, Master-Slides, Animationen, Praesentationsmodus
- Formate: .docx/.xlsx/.pptx/.odt/.ods/.odp/.pdf

**Technische Umsetzung (Frontend):**
- iframe-Einbettung der Collabora-Oberflaeche
- PostMessage API fuer Kommunikation (custom Buttons, Session-Management)
- Dateien: `modules/dokumente/CollaboraViewer.tsx` (~300 LOC), Aenderungen an `DokumentePage.tsx` (+50 LOC)
- Tag: `[FE-ONLY]` (Placeholder), `[BACKEND-DEP]` (WOPI + Docker)

**Technische Umsetzung (Backend — Luke):**
- WOPI-Protokoll: 3 Endpoints (`CheckFileInfo`, `GetFile`, `PutFile`) im Go-Backend
- Collabora Docker-Container: `collabora/code`, Port 9980
- Token-basierte Authentifizierung pro User+Datei
- Ressourcen: ~1 GB RAM + ~50 MB pro gleichzeitigem User

**Lizenz:**
- CODE (Development Edition): Kostenlos, aber begrenzt auf 10 Dokumente / 20 Connections — NUR fuer Entwicklung/Testing
- Business-Lizenz: ~1,82 EUR/User/Monat (bis 99 User)
- Enterprise-Lizenz: Individuell verhandelbar
- ISV/Partner-Programm: Kontakt ueber collaboraonline.com/partner-programme/
- Self-Hosted-Kunden: Brauchen eigene Collabora-Lizenz (wir buendeln das im Preis oder verrechnen separat)
- **Kein AGPL-Problem** (MPL 2.0 Lizenz, kommerziell nutzbar)

### Preisvergleich: KMU Hub vs. Einzelloesungen

| Was der Kunde braucht | Einzeln | KMU Hub Enterprise |
|---|---|---|
| Office (Word/Excel/PPT) | M365: ~13 EUR/User | inkludiert (Collabora) |
| E-Mail | inkludiert in M365 | inkludiert |
| Chat/Video (Teams) | inkludiert in M365 | inkludiert |
| CRM (Pipedrive) | +25 EUR/User | inkludiert |
| PM (Asana) | +11 EUR/User | inkludiert |
| Helpdesk (Zendesk) | +19 EUR/User | inkludiert |
| Zeiterfassung (Clockodo) | +7 EUR/User | inkludiert |
| DMS (Dropbox Business) | +12 EUR/User | inkludiert |
| **GESAMT** | **~87 EUR/User/Mo** | **25 EUR/User/Mo** |

---

## Querschnitts-Aenderungen (betreffen viele Module)

Diese Aenderungen ziehen sich quer durch die gesamte Codebase und muessen frueh adressiert werden.

### Q1: formatCurrency Multi-Waehrung — Deutschland-First

> **Wichtig:** KMU Hub startet mit Deutschland als erstem Markt. EUR ist die Standard-Waehrung, deutsche MWSt-Saetze (19%/7%) sind der Default. CHF/AT kommen spaeter als Konfiguration dazu.

- [ ] **`formatCHF` durch `formatCurrency(amount, currency?)` ersetzen** -- Aktuell nur `Intl.NumberFormat('de-CH', {currency: 'CHF'})`. Default wird `Intl.NumberFormat('de-DE', {currency: 'EUR'})`. Waehrung kommt aus Firmen-/Mandanten-Einstellung, nicht hardcoded.
  - **Default-Waehrung:** EUR (Deutschland)
  - **Default-MWSt:** 19% (Regelsteuersatz), 7% (ermaessigt)
  - **Default-Locale:** `de-DE` (Punkt als Tausender-Trenner in Zahlen, Komma als Dezimal)
  - **Spaeter erweiterbar:** CHF (CH: 8.1%/2.6%/3.8%), AT (20%/10%/13%)
  - **Mandanten-Setting:** `currency: 'EUR' | 'CHF'` + `country: 'DE' | 'AT' | 'CH'` in Settings-Store oder Mandanten-Config
  - Betrifft: Buchhaltung, Einkauf, Inventar, Vermietung, Vertraege, Fuhrpark, Rapporte, Schichten (Zuschlaege), Zeiterfassung
  - Dateien: `stores/finance.ts` (formatCHF → formatCurrency), `stores/settings.ts` oder neuer `stores/tenant.ts` (Mandanten-Config), alle Module die `formatCHF` aufrufen (~10+ Dateien)
  - Aufwand: ~200 LOC (finance.ts + tenant config) + ~250 LOC (Refactoring ueber Module)
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### Q2: Kontakt-Dualitaet loesen

- [ ] **CRM (Backend-verbunden) vs. Kontakte (Zustand) zusammenfuehren** -- Zwei separate Kontakt-Systeme existieren parallel. Das CRM-Modul nutzt React Query API-Hooks, das Kontakte-Modul nutzt lokalen Zustand. Migration auf ein gemeinsames Modell noetig.
  - Dateien: `stores/contacts.ts`, `modules/kontakte/KontaktePage.tsx`, `modules/crm/*`
  - Aufwand: ~800-1200 LOC Refactoring
  - Tag: `[BACKEND-DEP]` (gemeinsames API-Modell noetig)
  - Abhaengigkeit: Luke Phase 8 (CRM Backend)

### Q3: Rich-Text-Editor (TipTap) in alle textarea-Stellen

- [ ] **TipTap-Editor als Shared Component bauen und in alle Module integrieren** -- Aktuell nutzen Wiki, Helpdesk-KB, E-Mail-Compose, und Formulare plain `<textarea>`. TipTap ersetzt diese schrittweise.
  - Dateien: `components/shared/RichTextEditor/*` (neu), dann Integration in 5+ Module
  - Aufwand: ~600 LOC (Component) + ~400 LOC (Integrationen)
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: npm `@tiptap/*` Packages

### Q4: PDF-Export real machen

- [ ] **Alle "PDF Export"-Buttons die aktuell Fake sind mit echtem Export verbinden** -- Betrifft: Buchhaltung (Rechnungen/Angebote), Rapporte (Tagesberichte), Berichte (Reports), Schichten (Wochenplan). Frontend baut PDF-Vorschau-Panel, Backend generiert die Datei.
  - Dateien: Module-spezifische Export-Buttons + neues `PDFPreviewPanel.tsx`
  - Aufwand: ~400 LOC (Frontend-Shell), Backend generiert PDFs
  - Tag: `[BACKEND-DEP]` (PDF-Generierung serverseitig)
  - Abhaengigkeit: Luke (Go PDF-Library)

### Q5: DSGVO-Tools (Consent, Auskunft, Loeschung, Export)

- [ ] **Umfassende DSGVO-Compliance-UI bauen** -- Consent-Management pro Kontakt, Auskunfts-Tool (globale Suche), Loeschung (kaskadierte Anonymisierung), Datenexport (ZIP-Paket), Retention-Policy-Anzeige.
  - Dateien: Neues `modules/settings/dsgvo/*` oder eigenes Modul
  - Aufwand: ~2000-2500 LOC
  - Tag: `[BACKEND-DEP]` (Datenabfrage/-loeschung serverseitig)
  - Abhaengigkeit: Luke Phase 10+

---

## Wave 1: Foundation (Shared Components + Stores)

> Ziel: Alle wiederverwendbaren Bausteine die andere Waves benoetigen.
> Geschaetzter Aufwand: ~3.890 LOC | ~2-3 Wochen

### 1.1 TipTap Rich Text Editor

- [ ] **Shared Rich-Text-Editor auf Basis von TipTap bauen** -- Wird in Wiki, Kommunikation, Helpdesk, Chat, Formulare wiederverwendet. Features: Headings H1-H3, Bold/Italic/Underline, Listen (Bullet/Ordered/Task), Tabellen, Code-Bloecke, Links, Bilder, Horizontal Divider, Placeholder, Read-only-Modus.
  - Dateien: `components/shared/RichTextEditor/RichTextEditor.tsx`, `EditorToolbar.tsx`, `FormatGroup.tsx`, `HeadingGroup.tsx`, `ListGroup.tsx`, `InsertGroup.tsx`, `AlignGroup.tsx`, `EditorContent.tsx`, `EditorBubbleMenu.tsx`, `EditorFooter.tsx`
  - Aufwand: ~600 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: npm `@tiptap/react`, `@tiptap/starter-kit`, `@tiptap/extension-*`, `lowlight`
  - Blockiert: Wave 2 (Wiki, Kommunikation), Wave 5 (Helpdesk Canned Responses, KB), Wave 4 (E-Mail TipTap)

### 1.2 Status/Presence System

- [ ] **Online/Away/Busy/DND/Offline-Statusanzeige als Shared Component** -- Farbige Dots neben Benutzernamen in Chat, Meetings, Team, Kontakte. StatusPicker-Dropdown fuer eigenen Status, Custom-Status-Dialog mit Dauer.
  - Dateien: `components/shared/Presence/StatusDot.tsx`, `StatusPicker.tsx`, `SetStatusDialog.tsx`, `UserPresenceProvider.tsx`
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]` (Mock-Status, WebSocket spaeter)
  - Abhaengigkeit: Keine
  - Blockiert: Chat Presence (Wave 5), Video Meeting (Wave 11)

### 1.3 Global Search (Cmd+K)

- [ ] **Command-Palette / Spotlight-Suche ueber alle Module** -- Oeffnet mit Cmd+K / Ctrl+K. Sucht in Kontakten, Projekten, Tasks, Dokumenten, Wiki-Artikeln, Kalender-Events, E-Mails, Konversationen. Keyboard-Navigation (Pfeiltasten, Enter, Esc). Letzte 5 Suchen. Quick Actions (Neuer Kontakt, Neues Projekt, etc.).
  - Dateien: `components/shared/GlobalSearch/GlobalSearchDialog.tsx`, `SearchInput.tsx`, `SearchResultGroup.tsx`, `SearchResultItem.tsx`, `RecentSearches.tsx`, `QuickActions.tsx`
  - Aufwand: ~550 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine (sucht lokal in bestehenden Stores)

### 1.4 Alle neuen Zustand Stores

- [ ] **7 neue Stores mit Mock-Daten anlegen** -- Folgen bestehendem Pattern: `create<State>()(persist((set, get) => ({...}), { name: 'kmuhub-xxx' }))`. Mock-Daten als const Arrays.
  - Dateien und LOC:
    - `stores/communication.ts` (~450 LOC) -- Kommunikation-Modul
    - `stores/wiki.ts` (~350 LOC) -- Wiki-Modul
    - `stores/presence.ts` (~120 LOC) -- Status/Presence
    - `stores/search.ts` (~100 LOC) -- Global Search
    - `stores/notifications.ts` (~200 LOC) -- Notification Center
    - `stores/integrations.ts` (~250 LOC) -- Integration Settings
    - Erweiterung `stores/contacts.ts` (~100 LOC) -- Custom Fields
  - Aufwand: ~1.570 LOC gesamt
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine
  - Blockiert: Wave 2, 3, 11, 12

### 1.5 Meeting Store Erweiterung

- [ ] **`stores/meetings.ts` um Video-Meeting-State erweitern** -- Aktive Teilnehmer, Audio/Video-Toggle, Screen-Share-State fuer die Video-Meeting-UI.
  - Dateien: `stores/meetings.ts` (bestehend, 421 LOC)
  - Aufwand: ~80 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### 1.6 Q1: formatCurrency Refactor

- [ ] **`formatCHF` zu `formatCurrency(amount, currency)` in `finance.ts` umbauen** -- MWSt-Saetze-Array um DE (19%/7%) und AT (20%/10%/13%) erweitern. Laender-Selector-Logik hinzufuegen. Alle Module die `formatCHF` nutzen auf `formatCurrency` umstellen.
  - Dateien: `stores/finance.ts` + ~10 Modul-Dateien
  - Aufwand: ~350 LOC (150 finance.ts + 200 Refactoring)
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine
  - Blockiert: Wave 3 (Finance), Wave 8 (Einkauf/Inventar Multi-Waehrung)

### 1.7 Routing + Navigation Updates

- [ ] **Neue Routes in `App.tsx`, Nav-Items in `nav-items.ts`, Profile in `business-profiles.ts`** -- Neue Routes: `/kommunikation`, `/wiki`, `/meeting/:roomId`. Lazy Imports. Nav-Label "Buchhaltung" -> "Rechnungen & Finanzen". `kommunikation` und `wiki` in relevante Business-Profile einfuegen.
  - Dateien: `App.tsx`, `config/nav-items.ts`, `config/business-profiles.ts`
  - Aufwand: ~80 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### 1.8 TipTap CSS

- [ ] **ProseMirror-Override-CSS fuer TipTap erstellen** -- Muss CSS-Variablen des Design-Systems respektieren, Light/Dark/Glass/Crystal-kompatibel sein.
  - Dateien: `styles/tiptap.css` (neu)
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.1 (TipTap Component)

### 1.9 Type Definitions

- [ ] **Neue Type-Dateien fuer Kommunikation, Wiki, Presence, Integrations** -- Saubere Type-Exports fuer alle neuen Entitaeten.
  - Dateien: `types/communication.ts`, `types/wiki.ts`, `types/presence.ts`, `types/integrations.ts`
  - Aufwand: ~160 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

---

## Wave 2: Kommunikation + Wiki (2 neue Module)

> Ziel: Die zwei groessten neuen Module bauen.
> Geschaetzter Aufwand: ~4.000 LOC | ~3-4 Wochen
> Abhaengigkeit: Wave 1 (TipTap, Stores)

### 2.1 Kommunikation-Modul (Unified External Inbox)

- [ ] **Neues Modul `/kommunikation`** -- Einzelner Eingang fuer ALLE externen Kommunikationskanaele (E-Mail, Teams, WhatsApp, Widget, Portal). 3-Spalten-Layout: Konversationsliste (links, ~280px), Nachrichten-Thread (Mitte, flex-1), CRM-Kontext-Panel (rechts, ~300px, einklappbar). Tab-basiert nach Kanal. Automatische CRM-Kontakt-Zuordnung. Canned Responses (geteilt mit Helpdesk). "Aus CRM einfuegen"-Button.
  - Dateien:
    - `modules/kommunikation/KommunikationPage.tsx`
    - `modules/kommunikation/ChannelTabs.tsx`
    - `modules/kommunikation/ConversationList.tsx`
    - `modules/kommunikation/ConversationListHeader.tsx`
    - `modules/kommunikation/ConversationListItem.tsx`
    - `modules/kommunikation/ConversationListFilters.tsx`
    - `modules/kommunikation/ConversationThread.tsx`
    - `modules/kommunikation/ConversationThreadHeader.tsx`
    - `modules/kommunikation/MessageTimeline.tsx`
    - `modules/kommunikation/MessageItem.tsx`
    - `modules/kommunikation/ReplyComposer.tsx` (nutzt TipTap)
    - `modules/kommunikation/CannedResponsePicker.tsx`
    - `modules/kommunikation/InsertFromCRMButton.tsx`
    - `modules/kommunikation/InternalNoteComposer.tsx`
    - `modules/kommunikation/ContextPanel.tsx`
    - `modules/kommunikation/ContactCard.tsx`
    - `modules/kommunikation/OpenDeals.tsx`
    - `modules/kommunikation/OpenTickets.tsx`
    - `modules/kommunikation/RelatedProjects.tsx`
    - `modules/kommunikation/ActivityTimeline.tsx`
    - `modules/kommunikation/NewConversationDialog.tsx`
    - `modules/kommunikation/ChannelSettingsDialog.tsx`
  - Mock-Daten: 12-15 Konversationen ueber alle Kanaele, 3-8 Nachrichten pro Konversation
  - Aufwand: ~2.250 LOC (1.800 Components + 450 Store)
  - Tag: `[FE-ONLY]` (Mock-Daten, echte Kanaele spaeter)
  - Abhaengigkeit: Wave 1.1 (TipTap), Wave 1.4 (`communication.ts` Store)

### 2.2 Wiki-Modul (Knowledge Base)

- [ ] **Neues Modul `/wiki`** -- Interne Wissensdatenbank mit Rich-Text-Artikeln. Baum-Navigation links (~240px), Artikel-Ansicht Mitte (flex-1), Versionshistorie als Slide-in-Panel rechts. TipTap fuer Edit- und Read-Modus. Vorlagen-System.
  - Dateien:
    - `modules/wiki/WikiPage.tsx`
    - `modules/wiki/WikiSidebar.tsx`
    - `modules/wiki/WikiTreeNode.tsx`
    - `modules/wiki/WikiSearch.tsx`
    - `modules/wiki/WikiNewButton.tsx`
    - `modules/wiki/WikiArticle.tsx`
    - `modules/wiki/WikiArticleHeader.tsx`
    - `modules/wiki/WikiEditor.tsx` (nutzt TipTap)
    - `modules/wiki/WikiRenderer.tsx`
    - `modules/wiki/WikiArticleFooter.tsx`
    - `modules/wiki/WikiVersionHistory.tsx`
    - `modules/wiki/WikiVersionItem.tsx`
    - `modules/wiki/WikiTemplateDialog.tsx`
    - `modules/wiki/WikiCategoryDialog.tsx`
    - `modules/wiki/WikiShareDialog.tsx`
  - Mock-Daten: 4 Kategorien, 12 Artikel, 3 Vorlagen (Meeting Notes, Post-Mortem, How-To)
  - Aufwand: ~1.750 LOC (1.400 Components + 350 Store)
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.1 (TipTap), Wave 1.4 (`wiki.ts` Store)

---

## Wave 3: CRM + Finance Overhaul (groesste Gaps)

> Ziel: Die zwei Module mit Gap-Level GROSS auf Market-Fit bringen.
> Geschaetzter Aufwand: ~7.500-8.500 LOC | ~4-6 Wochen
> Abhaengigkeit: Wave 1 (Q1 formatCurrency)

### CRM / Kontakte

#### 3.1 CRM CRUD-Formulare

- [ ] **Create/Edit-Formulare fuer Kontakte, Firmen, Deals bauen** -- Aktuell zeigen alle "Kommt bald"-Placeholder. Vollstaendige Formular-Dialoge mit Validierung, Dropdown-Auswahlen, Mehrfach-Tags.
  - Dateien: `modules/crm/contacts/ContactFormDialog.tsx` (neu/erweitert), `modules/crm/companies/CompanyFormDialog.tsx` (neu), `modules/crm/deals/DealFormDialog.tsx` (neu)
  - Aufwand: ~1.200 LOC
  - Tag: `[BACKEND-DEP]` (API-Hooks fuer Persist)
  - Abhaengigkeit: Keine

#### 3.2 Custom Fields Editor UI + Store

- [ ] **Admin-UI fuer benutzerdefinierte Felder** -- Feld-Typen: Text, Zahl, Datum, Dropdown, Checkbox, URL. CRUD fuer Feld-Definitionen, Live-Vorschau, dynamisches Rendering in ContactDetail und ContactForm.
  - Dateien: `modules/kontakte/CustomFieldsConfig.tsx` (neu), `modules/kontakte/CustomFieldRow.tsx` (neu), `modules/kontakte/CustomFieldPreview.tsx` (neu), Aenderungen an `ContactDetailPanel.tsx`, `ContactFormDialog.tsx`
  - Aufwand: ~550 LOC (300 Config + 150 bestehende Components + 100 Store)
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Persistenz via JSONB)
  - Abhaengigkeit: Keine

#### 3.3 Firma (Company) Detail Panel

- [ ] **Firmen als eigene Entitaet mit Detail-Panel** -- Aktuell ist Firma nur ein String im Kontakte-Modul. Im CRM-Modul existiert Firma als Entity. Detail-Panel mit: Company Header (Name, Logo, Branche, Website), Mitarbeiterliste, verlinkte Deals, Activity-Timeline, Company-level Custom Fields.
  - Dateien: `modules/kontakte/FirmaDetailPanel.tsx` (neu), Erweiterungen `stores/contacts.ts`
  - Aufwand: ~620 LOC (500 Component + 120 Store)
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Firma-Entity in DB)
  - Abhaengigkeit: Wave 3.2 (Custom Fields)

#### 3.4 Duplikaterkennung Dialog

- [ ] **Bei Kontakt-Erstellung/Import pruefen ob Duplikat existiert** -- Dialog mit Side-by-Side-Vergleich, Merge-Optionen (welches Feld behalten). Matching auf E-Mail, Telefon, Namens-Aehnlichkeit.
  - Dateien: `modules/kontakte/DuplicateDetectionDialog.tsx` (neu), `modules/kontakte/DuplicateMatchCard.tsx` (neu), `modules/kontakte/MergeFieldSelector.tsx` (neu)
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]` (Mock-Matching), `[BACKEND-DEP]` (echtes Fuzzy-Matching)
  - Abhaengigkeit: Keine

#### 3.5 Kontakt-Timeline

- [ ] **Chronologische Ansicht aller Interaktionen eines Kontakts** -- Mails, Calls, Meetings, Deals, Tickets, Notizen in einer Timeline. Filter nach Typ. Endlos-Scroll.
  - Dateien: `modules/crm/contacts/ContactTimeline.tsx` (neu), `modules/crm/contacts/TimelineItem.tsx` (neu)
  - Aufwand: ~500 LOC
  - Tag: `[BACKEND-DEP]` (Cross-Modul-Daten)
  - Abhaengigkeit: Keine

#### 3.6 Akadem. Titel + Anrede-Logik

- [ ] **Titel-Dropdown (Prof., Dr., etc.) und Anrede (Herr/Frau/Divers) in Kontakt-Formularen** -- "Herr Prof. Dr. Mueller", Sie/Du-Flag, bevorzugte Sprache. DACH-Grunderwartung.
  - Dateien: Aenderungen an `ContactFormDialog.tsx`, `stores/contacts.ts` (Modell erweitern)
  - Aufwand: ~120 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 3.7 Consent-Management (DSGVO)

- [ ] **Einwilligungsflags pro Kontakt pro Zweck mit Timestamp** -- Tab/Sektion in Kontaktdetail: E-Mail-Marketing, Telefon, Post, Profiling, jeweils mit Datum, Quelle, Widerruf-Option.
  - Dateien: `modules/kontakte/ConsentPanel.tsx` (neu), Erweiterung `stores/contacts.ts`
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Persistenz)
  - Abhaengigkeit: Keine

#### 3.8 Newsletter Panel (Brevo/CleverReach)

- [ ] **Integration-Panel fuer Newsletter-Dienste** -- Connection-Status, Subscriber-Listen aus CRM, Send-History, "Sync Contacts"-Button.
  - Dateien: `modules/kontakte/NewsletterPanel.tsx` (neu)
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (API-Connector)
  - Abhaengigkeit: Keine

#### 3.9 CRM Import/Export

- [ ] **CSV- und vCard-Import/Export im CRM-Modul** -- Import-Dialog existiert im Kontakte-Modul, fehlt aber im CRM-Modul. Feld-Mapping, Vorschau, Duplikat-Check.
  - Dateien: `modules/crm/ImportExportDialog.tsx` (neu)
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]` (lokale Verarbeitung), `[BACKEND-DEP]` (Massen-Import)
  - Abhaengigkeit: Wave 3.4 (Duplikaterkennung)

### Buchhaltung / Rechnungen & Finanzen

#### 3.10 Buchhaltung umbenennen

- [ ] **Label in `nav-items.ts` von "Buchhaltung" zu "Rechnungen & Finanzen" aendern** -- Route bleibt `/buchhaltung`. FiBu-Referenzen (Kontenrahmen, Bilanz/GuV) entfernen.
  - Dateien: `config/nav-items.ts`, `modules/buchhaltung/BuchhaltungPage.tsx`
  - Aufwand: ~50 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 3.11 Belegkette-Tab

- [ ] **Visueller Pipeline-Tab: Angebot -> Auftrag -> Lieferschein -> Rechnung -> Mahnung** -- Status-Indikatoren pro Schritt, Klick zeigt verlinkte Dokumente, "In naechsten Schritt umwandeln"-Button. Liste der letzten Belegketten mit Fortschritts-Indikator.
  - Dateien: `modules/buchhaltung/BelegketteTab.tsx` (neu)
  - Aufwand: ~500 LOC
  - Tag: `[FE-ONLY]` (Mock-Pipeline), `[BACKEND-DEP]` (Konvertierungs-Logik)
  - Abhaengigkeit: Keine

#### 3.12 Exporte-Tab (DATEV + Bexio)

- [ ] **Neuer Tab mit DATEV-Export-Panel und Bexio-Sync-Dashboard** -- DATEV: Datumsbereich-Picker, Konten-Mapping-Tabelle, "Export starten"-Button, Export-Historie. Bexio: Connection-Status, letzter Sync, Sync-Button, Konflikt-Liste.
  - Dateien: `modules/buchhaltung/ExporteTab.tsx` (neu)
  - Aufwand: ~450 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (DATEV-Format, Bexio-API)
  - Abhaengigkeit: Keine

#### 3.13 QR-Rechnung Preview (Swiss QR-bill)

- [ ] **Mock Swiss QR-Bill-Darstellung unter der Rechnung** -- QR-Code-Placeholder, Zahlungsschein-Felder (IBAN, Referenz, Betrag). Pflicht in der Schweiz seit 2022.
  - Dateien: `modules/buchhaltung/QRRechnungPreview.tsx` (neu), Aenderungen an `InvoiceDetailPanel.tsx`
  - Aufwand: ~300 LOC (180 Component + 120 Integration)
  - Tag: `[FE-ONLY]` (Mock-QR), `[BACKEND-DEP]` (echte QR-Generierung)
  - Abhaengigkeit: Keine

#### 3.14 ZUGFeRD/XRechnung Indicator

- [ ] **Badge auf Rechnungen das ZUGFeRD-Compliance-Level anzeigt** -- Basic, Comfort, Extended. Tooltip mit Erklaerung. Ab 2025 Empfang Pflicht DE, ab 2027/2028 Versand Pflicht.
  - Dateien: Aenderungen an `InvoiceDetailPanel.tsx`, `stores/finance.ts`
  - Aufwand: ~80 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 3.15 MWSt multi-country (DE-Default)

- [ ] **MWSt-Saetze nach Mandanten-Land** -- Default: DE (19%/7%). Spaeter erweiterbar: AT (20%/10%/13%), CH (8.1%/2.6%/3.8%). Satz wird aus Mandanten-Config (Q1) gelesen, nicht manuell pro Rechnung ausgewaehlt. Optional: Ueberschreiben pro Rechnung fuer Auslands-Kunden.
  - Dateien: Aenderungen an `InvoiceFormDialog.tsx`, `stores/finance.ts`
  - Aufwand: ~150 LOC (80 Form + 70 Store)
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.6 (Q1 formatCurrency)

#### 3.16 Multi-Waehrung im Rechnungsformular

- [ ] **Waehrungs-Auswahl (CHF/EUR) im Rechnungs-Formular** -- Waehrung pro Rechnung, Anzeige in gewaehlter Waehrung.
  - Dateien: Aenderungen an `InvoiceFormDialog.tsx`, `InvoiceDetailPanel.tsx`
  - Aufwand: ~120 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.6 (Q1 formatCurrency)

#### 3.17 GoBD-konforme Rechnungen

- [ ] **Storno statt Loeschung, lueckenlose Nummern, Audit-Log-Anzeige** -- Loeschen-Button wird durch Storno-Button ersetzt. Aenderungsprotokoll im Detail-Panel anzeigen.
  - Dateien: Aenderungen an `BuchhaltungPage.tsx`, `InvoiceDetailPanel.tsx`, `stores/finance.ts`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]` (UI), `[BACKEND-DEP]` (Unveraenderbare Records)
  - Abhaengigkeit: Keine

#### 3.18 PDF-Vorschau Panel

- [ ] **Inline-Vorschau-Panel fuer Rechnungs-PDFs** -- Zeigt generiertes PDF direkt im Detail-Panel. Download-Button. Mock: Styled Placeholder.
  - Dateien: `modules/buchhaltung/PDFPreviewPanel.tsx` (neu)
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (echte PDF-Generierung)
  - Abhaengigkeit: Keine

#### 3.19 Stunden-zu-Rechnung Workflow

- [ ] **Zeiteintraege auswaehlen und daraus Rechnung generieren** -- "Rechnung erstellen"-Button bei selektierten Zeiteintraegen. Stundensatz x Stunden = Rechnungspositionen.
  - Dateien: Aenderungen an `ZeiterfassungPage.tsx`, `BuchhaltungPage.tsx`, neuer Dialog
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]` (Mock-Flow), `[BACKEND-DEP]` (Cross-Modul-Logik)
  - Abhaengigkeit: Keine

#### 3.20 Banking Widget (FinAPI Placeholder)

- [ ] **Placeholder fuer automatischen Bankabgleich** -- Connection-Card fuer FinAPI, Transaktions-Matching-UI (Rechnung <-> Zahlung).
  - Dateien: `modules/buchhaltung/BankingWidget.tsx` (neu)
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (FinAPI-Integration)
  - Abhaengigkeit: Keine

#### 3.21 BuchhaltungPage Restructuring

- [ ] **BuchhaltungPage.tsx um 2 neue Tabs (Belegkette, Exporte) erweitern** -- Tab-Leiste anpassen, Imports hinzufuegen.
  - Dateien: `modules/buchhaltung/BuchhaltungPage.tsx`
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 3.11, 3.12

---

## Wave 4: E-Mail Erweiterungen

> Ziel: E-Mail-Client aufwerten (UI-seitig).
> Geschaetzter Aufwand: ~1.200 LOC | ~1-2 Wochen
> Abhaengigkeit: Wave 1 (TipTap)

### 4.1 TipTap in ComposeModal

- [ ] **Plain-Text-Textarea in ComposeModal durch TipTap ersetzen** -- HTML-Mails mit Formatierung, Bilder, Links. Toolbar angepasst fuer E-Mail-Kontext (kein Code-Block, dafuer Signatur-Insert).
  - Dateien: Aenderungen an `modules/mails/ComposeModal.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.1 (TipTap)

### 4.2 E-Mail-Vorlagen

- [ ] **Vorlagen fuer wiederkehrende Mails (Angebot, Bestaetigung, Follow-Up)** -- Vorlagen-Auswahl im Compose-Dialog, Platzhalter-Variablen ({{name}}, {{firma}}).
  - Dateien: `modules/mails/EmailTemplateDialog.tsx` (neu), Erweiterung ComposeModal
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 4.1 (TipTap in Compose)

### 4.3 E-Mail-Signatur

- [ ] **Konfigurierbare Signatur pro User in Profil-Settings** -- Rich-Text-Editor fuer Signatur, automatisches Anfuegen an ausgehende Mails.
  - Dateien: `modules/profil/SignatureEditor.tsx` (neu), Aenderungen an ComposeModal
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.1 (TipTap)

### 4.4 Kontakt-Chip-Auswahl

- [ ] **An/CC/BCC-Felder mit Autocomplete aus CRM-Daten** -- Tippen zeigt passende Kontakte, Auswahl erzeugt Chip mit Name + E-Mail.
  - Dateien: Aenderungen an `modules/mails/ComposeModal.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine (liest aus contacts Store)

### 4.5 DSGVO Aufbewahrungsfristen-Anzeige

- [ ] **Info-Badge an E-Mails die Aufbewahrungsfristen unterliegen** -- DE: 6 Jahre Geschaeftsbriefe, CH: 10 Jahre. Tooltip mit Details.
  - Dateien: Aenderungen an `modules/mails/MailsPage.tsx`
  - Aufwand: ~100 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

---

## Wave 5: Chat + Helpdesk Extensions

> Ziel: Chat auf Teams-Niveau bringen, Helpdesk auf Zammad-Niveau.
> Geschaetzter Aufwand: ~3.400 LOC | ~3-4 Wochen
> Abhaengigkeit: Wave 1 (TipTap, Presence)

### Chat

#### 5.1 Thread Replies Enhancement

- [ ] **ThreadPanel.tsx erweitern fuer vollstaendige Thread-Konversationen** -- "Reply in Thread"-Button an jeder Nachricht. Thread-Panel slide-in von rechts. Reply-Count-Indikator an Nachrichten.
  - Dateien: Aenderungen an `ThreadPanel.tsx` (+200), `MessageBubble.tsx` (+40)
  - Aufwand: ~240 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 5.2 Reactions (Emoji Bar + Picker)

- [ ] **Emoji-Reaktionen unter Nachrichten** -- Klick zum Hinzufuegen/Entfernen. Reaktions-Picker (kleines Emoji-Grid) bei "+"-Button.
  - Dateien: `ReactionBar.tsx` (neu, ~120), `ReactionPicker.tsx` (neu, ~150), Aenderungen `MessageBubble.tsx` (+30)
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 5.3 @Mentions Autocomplete

- [ ] **`@`-Eingabe in MessageInput oeffnet Autocomplete-Dropdown mit Teammitgliedern** -- Auswahl wird als gestylter Chip im Input dargestellt.
  - Dateien: `MentionAutocomplete.tsx` (neu, ~200), Aenderungen `MessageInput.tsx` (+60)
  - Aufwand: ~260 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 5.4 File Sharing (Drag & Drop)

- [ ] **Dateien per Drag-and-Drop in Chat senden** -- Vorschau-Thumbnails fuer Bilder, Datei-Karten fuer Dokumente.
  - Dateien: `FileDropZone.tsx` (neu, ~100), `FileAttachmentCard.tsx` (neu, ~80), Aenderungen `MessageInput.tsx` (+40), `MessageBubble.tsx` (+50)
  - Aufwand: ~270 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Upload-Endpoint)
  - Abhaengigkeit: Keine

#### 5.5 Presence Integration in Chat

- [ ] **Online/Away/Busy/DND-Dots neben Benutzernamen in Channel-Member-Liste und Nachrichten** -- Nutzt Shared Presence Store.
  - Dateien: Aenderungen ueber Chat-Komponenten
  - Aufwand: ~80 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.2 (Presence System)

### Helpdesk

#### 5.6 Canned Responses Library

- [ ] **Textbaustein-Verwaltung und Schnell-Einfuegen im Reply-Bereich** -- Verwaltungs-Panel (CRUD), TipTap fuer Response-Bearbeitung, Dropdown im Reply-Composer.
  - Dateien: `modules/helpdesk/CannedResponsesPanel.tsx` (neu), `CannedResponseList.tsx` (neu), `CannedResponseEditor.tsx` (neu), `CannedResponsePicker.tsx` (geteilt mit Kommunikation)
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.1 (TipTap)

#### 5.7 Private Notes

- [ ] **Interne Notizen auf Tickets die der Kunde nicht sieht** -- Visuell unterschiedlich (andere Hintergrundfarbe, "Interne Notiz"-Label). Toggle zwischen Reply und interner Notiz.
  - Dateien: Aenderungen an `modules/helpdesk/HelpdeskPage.tsx`
  - Aufwand: ~80 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 5.8 Business Hours Configuration

- [ ] **Admin-Dialog fuer Geschaeftszeiten-Konfiguration** -- Oeffnungszeiten pro Tag, Feiertage, Zeitzone. SLA-Berechnung referenziert diese Zeiten.
  - Dateien: `modules/helpdesk/BusinessHoursDialog.tsx` (neu), Erweiterung Store
  - Aufwand: ~380 LOC (300 Dialog + 80 Store)
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 5.9 SLA Indicators

- [ ] **Farbkodierte Badges auf Tickets (gruen/gelb/rot)** -- Verbleibende Zeit, Ueberfaellig-Warnung, Breach-Alert.
  - Dateien: `modules/helpdesk/SLABadge.tsx` (neu, ~60), Aenderungen `HelpdeskPage.tsx` (+50)
  - Aufwand: ~110 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 5.10 Ticket-Kategorien

- [ ] **Kategorie-Zuordnung bei Ticket-Erstellung und -Bearbeitung** -- Dropdown mit vordefinierten Kategorien, Filter in Ticket-Tabelle.
  - Dateien: Aenderungen an `HelpdeskPage.tsx`, Store
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 5.11 Ticket-Zuweisung/Routing

- [ ] **Agent-Zuweisungs-Dropdown in Ticket-Detail** -- Automatische Zuweisung basierend auf Regeln (Kategorie -> Team/Agent). Routing-Konfiguration.
  - Dateien: Aenderungen an `HelpdeskPage.tsx`, neue Routing-Config-UI
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]` (Mock-Regeln), `[BACKEND-DEP]` (echtes Routing)
  - Abhaengigkeit: Wave 5.10 (Kategorien)

#### 5.12 CSAT (Kundenzufriedenheit)

- [ ] **Bewertungs-Widget nach Ticket-Schliessung** -- 1-5 Sterne + optionaler Kommentar. Aggregierte CSAT-Anzeige in Statistik-Tab.
  - Dateien: `modules/helpdesk/CSATWidget.tsx` (neu), Aenderungen Statistik-Tab
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Kunden-seitige Bewertung)
  - Abhaengigkeit: Keine

#### 5.13 Custom Fields fuer Tickets

- [ ] **Branchenspezifische Felder auf Tickets** -- Nutzt gleiche Custom-Fields-Infrastruktur wie CRM.
  - Dateien: Aenderungen an `HelpdeskPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 3.2 (Custom Fields Editor)

#### 5.14 KB Rich-Text-Editor

- [ ] **TipTap-Editor in Helpdesk-Wissensdatenbank-Artikeln** -- Ersetzt plain textarea.
  - Dateien: Aenderungen an `HelpdeskPage.tsx` (KB-Sektion)
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.1 (TipTap)

#### 5.15 E-Mail-zu-Ticket Konvertierung

- [ ] **UI fuer automatische Ticket-Erstellung aus eingehenden E-Mails** -- Konfiguration: welche E-Mail-Adresse erzeugt Tickets, Kategorie-Zuordnung.
  - Dateien: `modules/helpdesk/EmailToTicketConfig.tsx` (neu)
  - Aufwand: ~250 LOC
  - Tag: `[BACKEND-DEP]` (IMAP-Listener)
  - Abhaengigkeit: Keine

---

## Wave 6: Work/Projekte + Zeiterfassung

> Ziel: Projekt-Management und Zeiterfassung erweitern.
> Geschaetzter Aufwand: ~3.800 LOC | ~3-4 Wochen
> Abhaengigkeit: Keine (unabhaengig von Wave 1-5)

### Work / Projekte

#### 6.1 Interaktive Gantt-Balken

- [ ] **Drag-to-Resize und Drag-to-Move fuer Gantt-Balken** -- Aktuell nur Anzeige, kein Drag. Start-/End-Datum per Drag aendern, Abhaengigkeitslinien.
  - Dateien: Aenderungen an Gantt-Komponenten
  - Aufwand: ~800 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 6.2 Stunden-zu-Rechnung Button

- [ ] **"Rechnung erstellen aus Zeiteintraegen"-Button in Projekt-Detail** -- Oeffnet Dialog mit vorausgefuellten Stunden/Saetzen, generiert Rechnungs-Entwurf.
  - Dateien: Aenderungen an Projekt-Detail-Ansicht, neuer Dialog
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Cross-Modul)
  - Abhaengigkeit: Wave 3.19 (Stunden-zu-Rechnung Workflow)

#### 6.3 Projekt-Budget-Tracking

- [ ] **Budget-Sektion in Projekt-Detail mit Soll/Ist-Vergleich** -- Geplantes Budget vs. tatsaechliche Kosten (Stunden x Stundensatz). Fortschrittsbalken, Warnungen bei Ueberschreitung.
  - Dateien: Aenderungen an Projekt-Detail, neues `BudgetSection.tsx`
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]` (Mock-Daten)
  - Abhaengigkeit: Keine

#### 6.4 Auslastungsberichte

- [ ] **Team-Auslastung pro Projekt/Mitarbeiter visualisieren** -- Balkendiagramm mit Stunden pro Woche/Monat pro Person. Ueberlastungs-Warnungen.
  - Dateien: Neues `AuslastungReport.tsx`
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]` (Mock)
  - Abhaengigkeit: Keine

#### 6.5 Gaeste-Zugang

- [ ] **Read-only Projektstatus-Ansicht fuer Kunden/Partner** -- Separates View ohne Sidebar/Navigation, nur Projektfortschritt, Meilensteine, Status-Updates. Kein Konkurrent hat das gut.
  - Dateien: Neues `GuestProjectView.tsx`, Route-Erweiterung
  - Aufwand: ~500 LOC
  - Tag: `[BACKEND-DEP]` (Auth-System erweitern)
  - Abhaengigkeit: Keine

### Zeiterfassung

#### 6.6 Projekt/Task-Dropdown im Timer

- [ ] **Timer direkt einem Projekt/Task zuordnen** -- Aktuell nur Kategorie-Auswahl. Dropdown fuer Projekte und deren Tasks.
  - Dateien: Aenderungen an `ZeiterfassungPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 6.7 Export-Button (DATEV, CSV, Excel)

- [ ] **Format-Auswahl beim Export von Zeiteintraegen** -- DATEV (CSV, Windows-1252, Semikolon), normales CSV, Excel-kompatibel.
  - Dateien: Aenderungen an `ZeiterfassungPage.tsx`, neuer `ExportDialog.tsx`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]` (CSV lokal), `[BACKEND-DEP]` (DATEV-Format)
  - Abhaengigkeit: Keine

#### 6.8 Ueberstunden-Berechnung

- [ ] **Automatische Ueberstunden-Erkennung basierend auf Soll-Arbeitszeit** -- Anzeige im Wochen-/Monats-Report. Saldo (Plus/Minus-Stunden).
  - Dateien: Aenderungen an `ZeiterfassungPage.tsx`, `stores/timetracking.ts`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 6.9 Genehmigungs-Banner

- [ ] **Vorgesetzter genehmigt Wochenrapport** -- Banner-Komponente mit "Genehmigen"/"Ablehnen"-Buttons. Status-Tracking.
  - Dateien: Aenderungen an `ZeiterfassungPage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Workflow)
  - Abhaengigkeit: Keine

#### 6.10 Abwesenheits-Integration

- [ ] **Urlaub/Krankheit aus Team-Modul in Zeiterfassung reflektieren** -- Abwesenheits-Tage blockieren Timer-Start, werden im Report angezeigt.
  - Dateien: Aenderungen an `ZeiterfassungPage.tsx`, Cross-Store-Reading
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 6.11 GPS-Erfassung (Electron API)

- [ ] **Standort bei Timer-Start/Stop erfassen** -- Fuer Bau/Handwerk. Electron Geolocation API. Standort im Zeiteintrag speichern.
  - Dateien: Aenderungen an `ZeiterfassungPage.tsx`, Electron IPC
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (Mock-Koordinaten)
  - Abhaengigkeit: Keine

---

## Wave 7: Team/HR + Schichtplanung

> Ziel: HR-Modul korrigieren, Schichtplanung auf rechtskonforme Basis bringen.
> Geschaetzter Aufwand: ~4.200 LOC | ~3-4 Wochen
> Abhaengigkeit: Keine

### Team / HR

#### 7.1 Lohn-Tab entfernen, Integrationen-Tab hinzufuegen

- [ ] **Lohn-Tab komplett entfernen** -- Lohnabrechnung wird NIE gebaut. Ersetzen durch "Integrationen"-Tab: DATEV Lohn (Connection-Status, Sync-Count), Abacus HR, "Weitere HR-Systeme"-Placeholder.
  - Dateien: `modules/team/TeamPage.tsx` (-200/+250), `modules/team/HRIntegrationPanel.tsx` (neu, ~280), `stores/team.ts` (-80/+40)
  - Aufwand: ~290 LOC netto
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 7.2 Multi-Country Lohn-Anzeige

- [ ] **Laenderspezifische Abzugs-Anzeige** -- Aktuell nur CH (AHV, BVG, Quellensteuer). DE: Lohnsteuer/SV-Beitraege. AT: SV/Lohnsteuer. Laender-Auswahl bei Mitarbeiter.
  - Dateien: Aenderungen an `TeamPage.tsx`, `stores/team.ts`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]` (Mock-Abzuege)
  - Abhaengigkeit: Wave 7.1

#### 7.3 Digitale Personalakte

- [ ] **Dokumente-Sektion pro Mitarbeiter** -- Vertrag, Zeugnis, Zertifikate. Upload/Download, Kategorien, Ablauf-Tracking.
  - Dateien: `modules/team/PersonnelDocuments.tsx` (neu)
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Storage)
  - Abhaengigkeit: Keine

#### 7.4 Onboarding-Checklisten

- [ ] **Standardisierte Einarbeitungs-Checklisten fuer neue Mitarbeiter** -- Template-basiert, anpassbar, Fortschritts-Tracking.
  - Dateien: `modules/team/OnboardingChecklist.tsx` (neu)
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 7.5 Org-Chart

- [ ] **Organisationsstruktur als Baumstruktur visualisieren** -- Abteilungen, Berichtslinien, Klick-Navigation zu Mitarbeiter-Details.
  - Dateien: `modules/team/OrgChart.tsx` (neu)
  - Aufwand: ~500 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 7.6 Self-Service-Portal

- [ ] **Mitarbeiter koennen eigene Daten einsehen, Antraege stellen, Gehaltsabrechnung herunterladen** -- Eingeschraenkte Ansicht des Team-Moduls.
  - Dateien: `modules/team/SelfServiceView.tsx` (neu)
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Rollen/Berechtigungen)
  - Abhaengigkeit: Keine

### Schichtplanung

#### 7.7 Zuschlaege (Nacht/Wochenende/Feiertag)

- [ ] **Zuschlag-Anzeige auf Schicht-Bloecken** -- "+25% Nacht", "+50% Feiertag". Zuschlag-Regeln konfigurierbar.
  - Dateien: Aenderungen an `SchichtenPage.tsx`, `stores/schichten.ts`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 7.8 Arbeitszeitgesetz-Regeln + Validierung

- [ ] **Maximale Arbeitszeit, Ruhezeiten, Mindest-Pausen automatisch pruefen** -- Warnungen bei Regelverstoessen (rote Umrandung, Warn-Banner).
  - Dateien: Aenderungen an `SchichtenPage.tsx`, neue Validierungs-Logik
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 7.9 Konflikterkennung

- [ ] **Doppelbelegungen und Ueberarbeitung automatisch warnen** -- Konflikterkennung-Banner bei Zuweisung.
  - Dateien: Aenderungen an `SchichtenPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 7.8

#### 7.10 Feiertags-Kalender (CH/DE/AT)

- [ ] **Laenderspezifische Feiertage in Schichtplanung einbeziehen** -- Feiertags-Spalten farblich markiert. Konfiguration: Land + Kanton/Bundesland.
  - Dateien: Aenderungen an `SchichtenPage.tsx`, neue `holidays.ts` Config
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 7.11 Drag-and-Drop im Wochenplan

- [ ] **Schichten per Drag im Wochenraster verschieben** -- Drag-Handler fuer Schicht-Bloecke, visuelles Feedback, Drop-Validierung.
  - Dateien: Aenderungen an `SchichtenPage.tsx`
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 7.12 Self-Service (eigene Verfuegbarkeit)

- [ ] **Mitarbeiter tragen eigene Verfuegbarkeit ein** -- Verfuegbarkeits-Grid (Gruen/Gelb/Rot pro Tag), Tausch-Anfragen direkt stellen.
  - Dateien: Aenderungen an `SchichtenPage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 7.13 PDF-Export Wochenplan

- [ ] **Ausdruckbarer Schichtplan fuer Aushang** -- Export-Button generiert PDF des aktuellen Wochenplans.
  - Dateien: Aenderungen an `SchichtenPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (PDF-Generierung)
  - Abhaengigkeit: Keine

---

## Wave 8: Einkauf + Inventar + Produktion

> Ziel: Branchen-Module fuer Handel/Produktion vervollstaendigen.
> Geschaetzter Aufwand: ~4.800 LOC | ~3-4 Wochen
> Abhaengigkeit: Wave 1 (Q1 formatCurrency)

### Einkauf

#### 8.1 Katalog-Tab fuellen

- [ ] **Katalog-Tab (aktuell Placeholder) mit Artikel-Suche und Schnellbestellung** -- Artikel-Karten mit Preis, Verfuegbarkeit, "In den Warenkorb"-Button.
  - Dateien: Aenderungen an `EinkaufPage.tsx`
  - Aufwand: ~500 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 8.2 Belegkette -> Buchhaltung

- [ ] **"Rechnung zuordnen"-Button im Bestellungs-Detail** -- Verbindung Bestellung -> Lieferschein -> Eingangsrechnung.
  - Dateien: Aenderungen an `EinkaufPage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (Mock-Link), `[BACKEND-DEP]` (Cross-Modul)
  - Abhaengigkeit: Wave 3.11 (Belegkette)

#### 8.3 Lieferanten-Bewertung

- [ ] **Rating/Score basierend auf Lieferqualitaet, Termintreue, Preis** -- Sterne-Rating im Lieferanten-Detail, aggregierte Bewertungs-Historie.
  - Dateien: Aenderungen an `EinkaufPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 8.4 Genehmigungsworkflow

- [ ] **Bestellungen ab Betrag X muessen genehmigt werden** -- Genehmigungs-Banner, Schwellenwert-Konfiguration.
  - Dateien: Aenderungen an `EinkaufPage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Workflow)
  - Abhaengigkeit: Keine

#### 8.5 Multi-Waehrung

- [ ] **Waehrungs-Auswahl im Bestellungs-Dialog** -- CHF/EUR fuer internationale Lieferanten.
  - Dateien: Aenderungen an `EinkaufPage.tsx`
  - Aufwand: ~100 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.6 (Q1 formatCurrency)

#### 8.6 Inventar-Integration

- [ ] **Wareneingang erzeugt automatisch Inventar-Bewegung** -- "Zum Inventar buchen"-Button bei Wareneingang.
  - Dateien: Aenderungen an `EinkaufPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Cross-Modul)
  - Abhaengigkeit: Keine

#### 8.7 Rahmenvertraege

- [ ] **Abrufbestellungen aus Rahmenvertraegen** -- Rahmenvertrag anlegen, einzelne Abrufe erfassen.
  - Dateien: Aenderungen an `EinkaufPage.tsx`, neuer Dialog
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### Inventar

#### 8.8 Barcode-Scanner (Electron IPC)

- [ ] **Barcode per Kamera/Scanner erfassen** -- Electron IPC fuer Barcode-Kamera. Scan-Ergebnis fuellt Artikel-Suche.
  - Dateien: Aenderungen an `InventarPage.tsx`, Electron IPC
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]` (Mock-Scan), `[BACKEND-DEP]` (Electron native Module)
  - Abhaengigkeit: Keine

#### 8.9 Automatische Nachbestellung

- [ ] **Bei Unterschreitung Mindestbestand: Bestellvorschlag generieren** -- "Nachbestellen"-Button bei kritischem Bestand, Link zu Einkauf.
  - Dateien: Aenderungen an `InventarPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 8.10 Chargen-/Seriennummern

- [ ] **Chargen- und Seriennummern-Tracking fuer Rueckverfolgbarkeit** -- Felder im Artikel-Dialog, Chargen-Liste im Detail.
  - Dateien: Aenderungen an `InventarPage.tsx`, `stores/inventar.ts`
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 8.11 Inventur-Workflow

- [ ] **Zaehlliste generieren, Ist-vs-Soll-Vergleich, Differenz-Buchung** -- Inventur-Tab mit Zaehlliste, Eingabe-Grid, Diff-Anzeige.
  - Dateien: Aenderungen an `InventarPage.tsx`, neuer Inventur-Tab
  - Aufwand: ~500 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 8.12 Multi-Waehrung

- [ ] **EUR neben CHF fuer Artikel-Preise** -- Waehrungs-Auswahl im Artikel-Dialog.
  - Dateien: Aenderungen an `InventarPage.tsx`
  - Aufwand: ~80 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.6 (Q1 formatCurrency)

#### 8.13 Belegkette-Anbindung

- [ ] **Wareneingang automatisch aus Einkaufs-Bestellung** -- Verknuepfung zwischen Einkauf und Inventar.
  - Dateien: Aenderungen an `InventarPage.tsx`
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Cross-Modul)
  - Abhaengigkeit: Wave 8.6

### Produktion

#### 8.14 Materialverfuegbarkeits-Pruefung

- [ ] **Vor Auftragsstart pruefen ob alle Materialien auf Lager** -- Ampel-Anzeige (gruen/gelb/rot) beim Auftragsstart.
  - Dateien: Aenderungen an `ProduktionPage.tsx`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Inventar-Abfrage)
  - Abhaengigkeit: Keine

#### 8.15 Arbeitsplaene/Arbeitsgaenge

- [ ] **Einzelne Schritte innerhalb eines Fertigungsauftrags** -- Arbeitsgaenge-Sub-Tab im Auftragsdetail mit Reihenfolge, Dauer, Status.
  - Dateien: Aenderungen an `ProduktionPage.tsx`
  - Aufwand: ~500 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 8.16 Maschinenbelegung

- [ ] **Gantt-aehnliche Ansicht welche Maschine wann belegt ist** -- Zeilen = Maschinen, Spalten = Zeit, Bloecke = Auftraege.
  - Dateien: `modules/produktion/MaschinenbelegungChart.tsx` (neu)
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 8.17 Ausschuss-Tracking

- [ ] **Ausschussrate pro Auftrag/Produkt erfassen und anzeigen** -- Feld im Auftrags-Detail, Aggregation in Statistiken.
  - Dateien: Aenderungen an `ProduktionPage.tsx`
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

---

## Wave 9: Fuhrpark + Rapporte + Vermietung

> Ziel: Branchen-Module fuer Bau/Handwerk/Logistik vervollstaendigen.
> Geschaetzter Aufwand: ~4.500 LOC | ~3-4 Wochen
> Abhaengigkeit: Keine

### Fuhrpark

#### 9.1 Finanzamtkonformes Fahrtenbuch

- [ ] **Privat-/Dienstfahrt-Trennung fuer 1%-Regelung vs. Fahrtenbuch-Methode (DE)** -- Fahrtenbuch-Tab, Fahrten-Erfassung (Start/Ziel/Zweck/km/Privat-Toggle), Zusammenfassung.
  - Dateien: Aenderungen an `FuhrparkPage.tsx`, neuer Fahrtenbuch-Tab
  - Aufwand: ~600 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 9.2 TCO-Dashboard

- [ ] **Total Cost of Ownership pro Fahrzeug** -- Wartung + Treibstoff + Versicherung + Abschreibung. KPI-Karten und Verlaufs-Chart.
  - Dateien: Aenderungen an `FuhrparkPage.tsx`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 9.3 Dokumente pro Fahrzeug

- [ ] **Fahrzeugschein, Versicherungspolice, TUeV-Berichte** -- Dokumente-Sektion im Detail-Panel.
  - Dateien: Aenderungen an `FuhrparkPage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Storage)
  - Abhaengigkeit: Keine

#### 9.4 Schadensmeldung mit Foto-Upload

- [ ] **Unfaelle/Schaeden dokumentieren mit Fotos** -- Schadensmeldungs-Dialog mit Datum, Beschreibung, Foto-Upload-Bereich.
  - Dateien: `modules/fuhrpark/SchadensmeldungDialog.tsx` (neu)
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (File Upload)
  - Abhaengigkeit: Keine

#### 9.5 Reifenwechsel-Erinnerung

- [ ] **Saisonale Erinnerung (Oktober/April) fuer Sommer-/Winterreifen** -- Banner-Hinweis, Reifenwechsel-Termin im Fahrzeug-Detail.
  - Dateien: Aenderungen an `FuhrparkPage.tsx`
  - Aufwand: ~100 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### Rapporte

#### 9.6 Digitale Unterschrift

- [ ] **Touch-/Stift-Signatur auf Canvas** -- Ersetzt "Kommt bald"-Platzhalter. Unterschrift als Bild speichern.
  - Dateien: `modules/rapporte/SignatureCanvas.tsx` (neu), Aenderungen `RapportePage.tsx`
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 9.7 Foto-Upload (Electron File API)

- [ ] **Echte Foto-Upload-Funktion statt Platzhalter** -- Electron File Dialog, Vorschau-Thumbnails, Reihenfolge aenderbar.
  - Dateien: Aenderungen an `RapportePage.tsx`, Electron IPC
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]` (Mock-Dateien), `[BACKEND-DEP]` (Storage)
  - Abhaengigkeit: Keine

#### 9.8 Aufmass-Skizze (Canvas Editor)

- [ ] **Canvas-basierte Skizze fuer Aufmass** -- Ersetzt "Kommt bald"-Platzhalter. Freihand-Zeichnen, Linien, Masse eintragen, Text-Annotations.
  - Dateien: `modules/rapporte/SketchCanvas.tsx` (neu)
  - Aufwand: ~700 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 9.9 PDF-Export Tagesbericht

- [ ] **Druckfaehiger Tagesbericht als PDF** -- Layout-Vorlage mit Wetter, Arbeitszeit, Mitarbeiter, Taetigkeiten, Material, Fotos, Unterschrift.
  - Dateien: Aenderungen an `RapportePage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (PDF-Generierung)
  - Abhaengigkeit: Keine

#### 9.10 Genehmigungs-Workflow

- [ ] **Bauleiter genehmigt Tagesbericht** -- Genehmigungs-Banner, Status-Tracking, Kommentar bei Ablehnung.
  - Dateien: Aenderungen an `RapportePage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Workflow)
  - Abhaengigkeit: Keine

#### 9.11 Wetter-API-Integration

- [ ] **Automatisch Wetter fuer Standort/Datum abfragen** -- Open-Meteo API (kostenlos). Temperatur + Wetter-Icon vorausgefuellt.
  - Dateien: Aenderungen an `RapportePage.tsx`
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]` (API direkt aus Electron)
  - Abhaengigkeit: Keine

### Vermietung

#### 9.12 Automatische Preisberechnung

- [ ] **Reservierung zeigt Gesamtpreis (Tage x Tagessatz)** -- Preis-Zusammenfassung im Reservierungs-Dialog.
  - Dateien: Aenderungen an `VermietungPage.tsx`
  - Aufwand: ~100 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 9.13 Kautionsverwaltung

- [ ] **Kaution einziehen/zurueckgeben bei Vermietung an Kunden** -- Kautions-Feld im Objekt-Dialog, Status-Tracking (eingezogen/zurueckgegeben).
  - Dateien: Aenderungen an `VermietungPage.tsx`, `stores/vermietung.ts`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 9.14 Schadens-/Zustandsprotokoll

- [ ] **Zustand bei Abholung und Rueckgabe dokumentieren** -- Checkliste mit Zustandsmerkmalen, Foto-Upload, Unterschrift.
  - Dateien: `modules/vermietung/ZustandsprotokollDialog.tsx` (neu)
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 9.6 (Signatur-Canvas wiederverwendbar)

#### 9.15 Multi-Waehrung

- [ ] **EUR neben CHF fuer Vermietungspreise** -- Waehrungs-Auswahl im Objekt-Dialog.
  - Dateien: Aenderungen an `VermietungPage.tsx`
  - Aufwand: ~80 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.6 (Q1 formatCurrency)

---

## Wave 10: Formulare + Vertraege + Kalender + Meetings

> Ziel: Module mit Gap-Level KLEIN vervollstaendigen.
> Geschaetzter Aufwand: ~5.200 LOC | ~4-5 Wochen
> Abhaengigkeit: Keine

### Formulare

#### 10.1 Bedingte Logik

- [ ] **Felder ein-/ausblenden basierend auf vorherigen Antworten** -- Logik-Editor pro Feld: "Wenn Feld X = Y, zeige Feld Z".
  - Dateien: Aenderungen an `FormularePage.tsx`
  - Aufwand: ~500 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 10.2 Mehrseitige Formulare

- [ ] **Formulare mit mehreren Seiten/Abschnitten** -- Seiten-Trenner im Builder, Weiter/Zurueck-Navigation fuer Ausfueller.
  - Dateien: Aenderungen an `FormularePage.tsx`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 10.3 Automatische Aktionen

- [ ] **Bei Einreichung: E-Mail senden, Task erstellen, CRM-Kontakt anlegen** -- Aktions-Konfiguration im Formular-Einstellungen-Panel.
  - Dateien: Aenderungen an `FormularePage.tsx`
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Server-Actions)
  - Abhaengigkeit: Keine

#### 10.4 Oeffentlicher Zugang ohne Login

- [ ] **Formulare ohne Login ausfuellbar fuer Kunden** -- Separate Render-Route ohne Auth-Check.
  - Dateien: Neue Route, `PublicFormView.tsx`
  - Aufwand: ~350 LOC
  - Tag: `[BACKEND-DEP]` (Auth-Ausnahme)
  - Abhaengigkeit: Keine

#### 10.5 Einreichungs-Export

- [ ] **CSV/Excel-Export aller Einreichungen eines Formulars** -- Export-Button in Eingaenge-Tab.
  - Dateien: Aenderungen an `FormularePage.tsx`
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### Vertraege

#### 10.6 E-Signatur Dialog (Skribble Mock)

- [ ] **Dialog fuer digitale Unterschrift via Skribble** -- Signer-Liste (per E-Mail hinzufuegen), Signing-Order (sequentiell/parallel), Erinnerungs-Einstellungen, Status-Timeline (gesendet/angesehen/unterschrieben/abgelehnt).
  - Dateien: `modules/vertraege/ESignaturDialog.tsx` (neu), `SignerList.tsx` (neu), `SigningOrderConfig.tsx` (neu), `SignatureStatusTimeline.tsx` (neu), Aenderungen `VertraegePage.tsx`
  - Aufwand: ~600 LOC (450 Dialog + 50 Integration + 100 Store)
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Skribble API)
  - Abhaengigkeit: Keine

#### 10.7 Dokument-Verknuepfung

- [ ] **Vertrags-PDF aus Dokumente-Modul verlinken** -- Dokument-Picker im Vertrags-Detail, `documentRef` editierbar machen.
  - Dateien: Aenderungen an `VertraegePage.tsx`
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 10.8 Erinnerungen/Benachrichtigungen

- [ ] **Automatische Erinnerung vor Kuendigungsfrist** -- Erinnerungs-Konfiguration (30/60/90 Tage) im Vertrags-Dialog.
  - Dateien: Aenderungen an `VertraegePage.tsx`
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Server-Notifications)
  - Abhaengigkeit: Keine

#### 10.9 Vertrags-Vorlagen

- [ ] **Standardvertraege als Vorlage speichern und wiederverwenden** -- Vorlagen-Tab, "Neuer Vertrag aus Vorlage"-Button.
  - Dateien: Aenderungen an `VertraegePage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 10.10 Multi-Waehrung

- [ ] **EUR neben CHF fuer Vertragswerte** -- Waehrungs-Auswahl im Vertrags-Dialog.
  - Dateien: Aenderungen an `VertraegePage.tsx`
  - Aufwand: ~80 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.6 (Q1 formatCurrency)

### Kalender

#### 10.11 Drag-and-Drop Events

- [ ] **Events per Drag in Week/Day-View verschieben/resizen** -- Drag-Handler, visuelles Feedback, Drop-Update.
  - Dateien: Aenderungen an `KalenderPage.tsx`
  - Aufwand: ~500 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 10.12 Video-Meeting Button + Link

- [ ] **"Start Meeting"-Button bei Events mit `isVideoCall: true`** -- Starter-Tier: Zoom-Link. Business+: LiveKit-Room.
  - Dateien: Aenderungen an `KalenderPage.tsx`
  - Aufwand: ~110 LOC (80 Button + 30 Store)
  - Tag: `[FE-ONLY]` (Mock-Link)
  - Abhaengigkeit: Keine

#### 10.13 Push-Erinnerungen

- [ ] **Desktop-Benachrichtigungen via Electron Notification API** -- 15/10/5 Minuten vor Event.
  - Dateien: Aenderungen an `KalenderPage.tsx`, Electron IPC
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 10.14 Externer Buchungslink

- [ ] **Kunden koennen online Termine buchen (Calendly-aehnlich)** -- Oeffentliche Buchungsseite, Verfuegbarkeits-Anzeige, Bestaetigungs-Mail.
  - Dateien: Neues `PublicBookingPage.tsx`, Route-Erweiterung
  - Aufwand: ~500 LOC
  - Tag: `[BACKEND-DEP]` (Oeffentlicher Endpoint, Auth)
  - Abhaengigkeit: Keine

### Meetings

#### 10.15 LiveKit Video-Integration (UI Shell)

- [ ] **Video-Meeting-Raum-UI Shell ohne echte Streams** -- Grid-Layout fuer Teilnehmer, Controls (Mute/Kamera/Teilen/Chat/Beenden). In Mock-Modus: Avatare statt Video.
  - Dateien: Identisch mit Wave 11.1 -- hier als Meetings-Feature referenziert.
  - Aufwand: Siehe Wave 11.1
  - Tag: `[FE-ONLY]` (Shell), `[BACKEND-DEP]` (LiveKit Token)
  - Abhaengigkeit: Keine

#### 10.16 Wiederkehrende Meetings

- [ ] **Wiederholungsmuster im Meeting-Formular** -- Taeglich, woechentlich, monatlich, benutzerdefiniert.
  - Dateien: Aenderungen an `MeetingsPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 10.17 Meeting-Notizen/Protokoll

- [ ] **Notiz-Editor im Meeting-Detail-Panel** -- TipTap fuer strukturierte Meeting-Notizen waehrend/nach dem Meeting.
  - Dateien: Aenderungen an `MeetingsPage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 1.1 (TipTap)

#### 10.18 Kalender-Synchronisation

- [ ] **Meetings bidirektional mit Kalender synchronisieren** -- Gemeinsamer Event-Store oder Cross-Store-Sync.
  - Dateien: Aenderungen an `stores/meetings.ts`, `stores/calendar.ts`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]` (lokaler Sync)
  - Abhaengigkeit: Keine

#### 10.19 Einladungs-E-Mails

- [ ] **Automatische E-Mail an Teilnehmer bei Meeting-Erstellung** -- Mit Kalender-Attachment (.ics).
  - Dateien: Aenderungen an `MeetingsPage.tsx`
  - Aufwand: ~150 LOC
  - Tag: `[BACKEND-DEP]` (E-Mail-Versand)
  - Abhaengigkeit: Keine

---

## Wave 11: Video + Notifications + Dashboard + Berichte + Dokumente

> Ziel: Shared UI Components und Module mit ueberfaelligen Features.
> Geschaetzter Aufwand: ~5.500 LOC | ~4-5 Wochen
> Abhaengigkeit: Wave 1

### 11.1 Video Meeting Room UI Shell

- [ ] **Shared Component fuer LiveKit-Video-Meetings** -- VideoGrid mit Participant-Tiles, MeetingControls (Mute/Kamera/Teilen/Chat/Beenden), MeetingSidebar (Teilnehmer/Chat/Settings), ScreenShareOverlay. In Mock-Modus: Avatare mit Initialen.
  - Dateien: `components/shared/VideoMeeting/VideoMeetingRoom.tsx`, `VideoGrid.tsx`, `VideoTile.tsx`, `MeetingControls.tsx`, `MeetingParticipantList.tsx`, `MeetingSidebar.tsx`, `MeetingChat.tsx`, `ScreenShareOverlay.tsx`, `MeetingEndDialog.tsx`
  - Aufwand: ~800 LOC
  - Tag: `[FE-ONLY]` (Shell), `[BACKEND-DEP]` (LiveKit Token)
  - Abhaengigkeit: Wave 1.5 (Meeting Store Extension)

### 11.2 Notification Center Rebuild

- [ ] **Bell-Icon im Header mit Dropdown + erweiterte Vollseite** -- NotificationBell mit Unread-Count, NotificationDropdown (letzte 10), NotificationItem mit Icon/Text/Zeit/Aktionen, Gruppierung ("Heute"/"Gestern"/"Aelter"), NotificationSettingsPanel (per-Kategorie Prefs). 10 Notification-Typen: Neue Nachricht, E-Mail, Aufgabe zugewiesen, Aufgabe faellig, Meeting in 15 Min, Neues Ticket, Vertrag laeuft aus, Rechnung ueberfaellig, @Mention, System.
  - Dateien: `components/shared/NotificationBell.tsx`, `NotificationDropdown.tsx`, `NotificationItem.tsx`, `NotificationGroupHeader.tsx`, Ueberarbeitung `NotificationCenter.tsx`, `NotificationSettingsPanel.tsx`
  - Aufwand: ~700 LOC (500 Components + 200 Store)
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (WebSocket Push)
  - Abhaengigkeit: Wave 1.4 (`notifications.ts` Store)

### Dashboard

#### 11.3 Personalisierte Widgets

- [ ] **Branchenspezifische Widget-Vorschlaege basierend auf Business-Profil** -- Widget-Auswahl aus Profil-relevanten Modulen.
  - Dateien: Aenderungen an `DashboardPage.tsx`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 11.4 Quick Actions Leiste

- [ ] **Schnellzugriff-Buttons unter der Begruessung** -- "Neuer Kontakt", "Neue Rechnung", "Timer starten", "Neues Projekt".
  - Dateien: Aenderungen an `DashboardPage.tsx`
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 11.5 Benachrichtigungs-Feed Widget

- [ ] **Zentrale Benachrichtigungs-Timeline als Dashboard-Widget** -- Letzte 10 Benachrichtigungen inline im Dashboard.
  - Dateien: Aenderungen an `DashboardPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Wave 11.2 (Notification Store)

### Berichte

#### 11.6 Benutzerdefinierte Dashboards

- [ ] **Eigene KPI-Zusammenstellungen erstellen** -- Dashboard-Editor mit Widget-Auswahl, Drag-and-Drop-Anordnung.
  - Dateien: Aenderungen an `BerichtePage.tsx`
  - Aufwand: ~600 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 11.7 Drill-Down auf KPI-Karten

- [ ] **Klick auf KPI zeigt Detail-Daten** -- Modal oder Panel mit Detail-Tabelle/Chart.
  - Dateien: Aenderungen an `BerichtePage.tsx`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]` (Mock-Daten)
  - Abhaengigkeit: Keine

#### 11.8 Vergleichsberichte

- [ ] **Zeitraum-Vergleich (Q1 vs. Q2, Jahr ueber Jahr)** -- Vergleichs-Toggle, Side-by-Side-Charts, Differenz-Anzeige.
  - Dateien: Aenderungen an `BerichtePage.tsx`
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 11.9 DATEV-Auswertungen

- [ ] **Steuerberater-konforme Reports** -- Teil des DATEV-Export-Features. Formatierte Uebersichten.
  - Dateien: Aenderungen an `BerichtePage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (DATEV-Format)
  - Abhaengigkeit: Wave 3.12 (Exporte-Tab)

### Dokumente

#### 11.10 "Oeffnen in Word/Excel/PPT" (Lokales Office — alle Tiers)

- [ ] **"In Word oeffnen" / "In Excel oeffnen" / "In PowerPoint oeffnen" Button** -- Electron laedt Datei in temp-Ordner, `shell.openPath()` oeffnet lokales Programm, `fs.watch()` FileWatcher synchronisiert Aenderungen zurueck. Erkennt installierte Programme automatisch.
  - Dateien: `modules/dokumente/LocalOfficeOpener.tsx` (neu, ~200), Aenderungen `DokumentePage.tsx` (+50)
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 11.10b Collabora Viewer (Enterprise Tier)

- [ ] **Collabora iframe-Einbettung fuer .docx/.xlsx/.pptx im Browser** -- PostMessage API, Loading-State, Fehler-Handling wenn Collabora nicht verfuegbar.
  - Dateien: `modules/dokumente/CollaboraViewer.tsx` (neu, ~300), Aenderungen `DokumentePage.tsx` (+50)
  - Aufwand: ~350 LOC
  - Tag: `[FE-ONLY]` (Placeholder), `[BACKEND-DEP]` (WOPI Endpoints + Collabora Docker)
  - Abhaengigkeit: Keine

#### 11.11 Template Gallery

- [ ] **"Neu aus Vorlage"-Dialog mit kategorisierten Dokumentvorlagen** -- Kategorien (Vertraege, Briefe, Formulare, ...), Vorschau-Karten.
  - Dateien: `modules/dokumente/TemplateGalleryDialog.tsx` (neu), `TemplateCard.tsx` (neu), `TemplateCategoryFilter.tsx` (neu), Store-Erweiterung
  - Aufwand: ~430 LOC (350 Components + 80 Store)
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 11.12 Share Link Dialog

- [ ] **Shareable-Link fuer Dokumente generieren** -- Optionen: Ablaufdatum, Passwort-Schutz, View-only vs. Download.
  - Dateien: `modules/dokumente/ShareLinkDialog.tsx` (neu)
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]` (Mock-Link), `[BACKEND-DEP]` (Signierte URLs)
  - Abhaengigkeit: Keine

#### 11.13 Versionierung-Anzeige

- [ ] **Versionsverlauf einer Datei im Detail-Panel** -- Timeline mit Version, Autor, Datum, "Wiederherstellen"-Button.
  - Dateien: Aenderungen an `DokumentePage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Server-Versioning)
  - Abhaengigkeit: Keine

#### 11.14 Inline-Vorschau (PDF/Bilder)

- [ ] **PDF- und Bild-Vorschau direkt im Browser** -- Modal oder Panel, nicht nur Download.
  - Dateien: Aenderungen an `DokumentePage.tsx`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

#### 11.15 Nextcloud WebDAV Browser

- [ ] **Dateien aus Nextcloud in KMU Hub anzeigen** -- WebDAV-Ordner als externe Quelle in der Sidebar.
  - Dateien: `modules/dokumente/NextcloudBrowser.tsx` (neu)
  - Aufwand: ~400 LOC
  - Tag: `[BACKEND-DEP]` (WebDAV-Proxy)
  - Abhaengigkeit: Keine

---

## Wave 12: Integration Panels + Settings

> Ziel: Alle Integrations-UIs in Settings bauen.
> Geschaetzter Aufwand: ~1.850 LOC | ~2-3 Wochen
> Abhaengigkeit: Wave 1.4 (`integrations.ts` Store)

### 12.1 IntegrationSettingsTab + IntegrationCard

- [ ] **Grid aller verfuegbaren Integrationen in Settings** -- Karten mit Logo, Name, Status, "Konfigurieren"-Button.
  - Dateien: `modules/settings/integrations/IntegrationSettingsTab.tsx`, `IntegrationCard.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### 12.2 GenericIntegrationPanel

- [ ] **Wiederverwendbare Panel-Vorlage fuer alle Integrationen** -- Connection-Status, Credentials-Input, Sync-Settings, Test-Button, Activity-Log.
  - Dateien: `modules/settings/integrations/GenericIntegrationPanel.tsx`
  - Aufwand: ~150 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### 12.3-12.11 Spezifische Integration Panels

- [ ] **12.3 DATEVConfigPanel** -- DATEV-Export-Konfiguration, Konten-Mapping.
  - Dateien: `modules/settings/integrations/DATEVConfigPanel.tsx`
  - Aufwand: ~120 LOC | Tag: `[FE-ONLY]`
- [ ] **12.4 BexioConfigPanel** -- Bexio OAuth2, Sync-Scope, Konflikt-Handling.
  - Dateien: `modules/settings/integrations/BexioConfigPanel.tsx`
  - Aufwand: ~120 LOC | Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (OAuth2)
- [ ] **12.5 BrevoConfigPanel** -- Brevo API-Key, Listen-Mapping, Sync-Toggle.
  - Dateien: `modules/settings/integrations/BrevoConfigPanel.tsx`
  - Aufwand: ~120 LOC | Tag: `[FE-ONLY]`
- [ ] **12.6 SkribbleConfigPanel** -- Skribble API-Key, Standard-Signatur-Level.
  - Dateien: `modules/settings/integrations/SkribbleConfigPanel.tsx`
  - Aufwand: ~120 LOC | Tag: `[FE-ONLY]`
- [ ] **12.7 CollaboraConfigPanel** -- Collabora Server-URL, WOPI-Discovery, Test-Connection, Lizenz-Status.
  - Dateien: `modules/settings/integrations/CollaboraConfigPanel.tsx`
  - Aufwand: ~150 LOC | Tag: `[FE-ONLY]`
- [ ] **12.8 ZoomConfigPanel** -- Zoom OAuth2, Meeting-Defaults.
  - Dateien: `modules/settings/integrations/ZoomConfigPanel.tsx`
  - Aufwand: ~120 LOC | Tag: `[FE-ONLY]`
- [ ] **12.9 LiveKitConfigPanel** -- LiveKit-Server-URL, API-Key/Secret.
  - Dateien: `modules/settings/integrations/LiveKitConfigPanel.tsx`
  - Aufwand: ~120 LOC | Tag: `[FE-ONLY]`
- [ ] **12.10 TeamsConfigPanel** -- Microsoft Graph API, Bot-Konfiguration.
  - Dateien: `modules/settings/integrations/TeamsConfigPanel.tsx`
  - Aufwand: ~120 LOC | Tag: `[FE-ONLY]`
- [ ] **12.11 WhatsAppConfigPanel** -- WhatsApp Business API, Phone-Number, Webhook-URL.
  - Dateien: `modules/settings/integrations/WhatsAppConfigPanel.tsx`
  - Aufwand: ~120 LOC | Tag: `[FE-ONLY]`

### 12.12 Store: integrations.ts

- [ ] **Integrations-Store (Teil von Wave 1.4, hier referenziert)** -- Bereits in Wave 1.4 enthalten.
  - Aufwand: Enthalten in Wave 1.4 (~250 LOC)

---

## Wave 13: DSGVO + KI + Polish

> Ziel: Compliance-Tools und KI-Features.
> Geschaetzter Aufwand: ~4.500 LOC | ~3-4 Wochen
> Abhaengigkeit: Wave 3 (CRM), Wave 11 (Notifications)

### DSGVO

#### 13.1 Consent-Management UI

- [ ] **Pro Kontakt, pro Zweck, mit Timestamp und Quelle** -- E-Mail-Marketing, Telefon, Post, Profiling. Widerruf-Button mit Datum-Logging.
  - Dateien: Separates Panel oder in Wave 3.7 integriert
  - Aufwand: ~350 LOC (erweitert Wave 3.7)
  - Tag: `[FE-ONLY]` (Mock), `[BACKEND-DEP]` (Persistenz)
  - Abhaengigkeit: Wave 3.7

#### 13.2 DSGVO-Auskunft-Tool

- [ ] **Globale Suche ueber alle Module, JSON/CSV-Export aller Daten einer Person** -- Art. 15 DSGVO. Suche nach Name/E-Mail, sammelt Daten aus CRM, Helpdesk, Mails, Projekte.
  - Dateien: `modules/settings/dsgvo/AuskunftTool.tsx` (neu)
  - Aufwand: ~500 LOC
  - Tag: `[BACKEND-DEP]` (Cross-Modul-Suche)
  - Abhaengigkeit: Keine

#### 13.3 DSGVO-Loeschung UI

- [ ] **Kaskadierte Anonymisierung ueber alle Module** -- Art. 17 DSGVO. Anzeige was geloescht/anonymisiert wird, GoBD-Ausnahmen (Rechnungen 10 Jahre behalten, nur Kontaktdaten anonymisieren). Bestaetigung.
  - Dateien: `modules/settings/dsgvo/LoeschungDialog.tsx` (neu)
  - Aufwand: ~400 LOC
  - Tag: `[BACKEND-DEP]` (Kaskadierung serverseitig)
  - Abhaengigkeit: Keine

#### 13.4 Datenexport/Portabilitaet

- [ ] **Strukturiertes ZIP-Paket aller Daten einer Person** -- Art. 20 DSGVO. Button "Datenpaket erstellen", Download als ZIP.
  - Dateien: `modules/settings/dsgvo/DatenexportPanel.tsx` (neu)
  - Aufwand: ~200 LOC
  - Tag: `[BACKEND-DEP]` (ZIP-Generierung)
  - Abhaengigkeit: Keine

#### 13.5 Retention-Policy Anzeige

- [ ] **Aufbewahrungsfristen DE/CH/AT anzeigen** -- Pro Dokumenttyp: Frist, Ablaufdatum, Status. Rechnungen 10J, Geschaeftsbriefe 6J (DE) / 10J (CH), etc.
  - Dateien: `modules/settings/dsgvo/RetentionPolicyPanel.tsx` (neu)
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### KI-Features

#### 13.6 KI: E-Mail-Entwuerfe

- [ ] **KI-gestuetzte E-Mail-Antwort-Vorschlaege** -- Button "KI-Entwurf" im Compose-Bereich, generiert Antwort basierend auf Kontext.
  - Dateien: Aenderungen an `ComposeModal.tsx`
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]` (OpenAI API aus `.env`), `[BACKEND-DEP]` (Proxy)
  - Abhaengigkeit: Keine

#### 13.7 KI: Meeting-Zusammenfassungen

- [ ] **Automatische Zusammenfassung von Meeting-Notizen** -- Button "Zusammenfassen" im Meeting-Detail.
  - Dateien: Aenderungen an `MeetingsPage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[FE-ONLY]` (OpenAI API), `[BACKEND-DEP]` (Proxy)
  - Abhaengigkeit: Wave 10.17 (Meeting-Notizen)

#### 13.8 KI: Ticket-Response-Vorschlaege

- [ ] **KI schlaegt Antwort basierend auf Ticket-Verlauf und KB-Artikeln vor** -- Button "KI-Vorschlag" im Helpdesk-Reply-Bereich.
  - Dateien: Aenderungen an `HelpdeskPage.tsx`
  - Aufwand: ~250 LOC
  - Tag: `[FE-ONLY]` (OpenAI API), `[BACKEND-DEP]` (Proxy)
  - Abhaengigkeit: Keine

#### 13.9 KI: Semantische Suche

- [ ] **Natuerlichsprachliche Suche ueber Wiki, Docs, Tickets, CRM** -- Erweiterung des Global Search (Cmd+K) mit semantischer Komponente.
  - Dateien: Aenderungen an `GlobalSearchDialog.tsx`
  - Aufwand: ~300 LOC
  - Tag: `[BACKEND-DEP]` (Embedding-Service)
  - Abhaengigkeit: Wave 1.3 (Global Search)

#### 13.10 KI: Auto-Klassifizierung

- [ ] **Automatische Dokument-Klassifizierung (oeffentlich/intern/vertraulich)** -- Badge auf Dokumenten, konfigurierbare Regeln.
  - Dateien: Aenderungen an `DokumentePage.tsx`
  - Aufwand: ~200 LOC
  - Tag: `[BACKEND-DEP]` (KI-Service)
  - Abhaengigkeit: Keine

#### 13.11 KI-Governance-Panel

- [ ] **Opt-out pro Modul, kein Training auf Kundendaten, Logging** -- Settings-Panel mit KI-Konfiguration, Aktivitaets-Log.
  - Dateien: `modules/settings/KIGovernancePanel.tsx` (neu)
  - Aufwand: ~300 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

---

## Wave 14: Visual Polish + Review (D10 + D11)

> Ziel: Feinschliff und Review-Fixes.
> Geschaetzter Aufwand: ~2.000-3.000 LOC | ~2-3 Wochen
> Abhaengigkeit: Alle vorherigen Waves

### 14.1 Animationen (D10)

- [ ] **Page Transitions, Micro-Interactions, Loading States** -- Smooth Uebergaenge zwischen Modulen, Skeleton-Loading, Button-Feedback.
  - Dateien: `styles/animations.css` (neu), Aenderungen in Layout-Komponenten
  - Aufwand: ~600 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### 14.2 Accessibility (D10)

- [ ] **ARIA Labels, Keyboard Navigation, Screen Reader Support** -- Alle interaktiven Elemente muessen per Tastatur bedienbar sein, ARIA-Roles korrekt gesetzt.
  - Dateien: Aenderungen ueber alle Module
  - Aufwand: ~800 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### 14.3 Responsive Fine-Tuning (D10)

- [ ] **Container-Widths von ~800px bis ~1600px testen und anpassen** -- DeskFrame-Window kann verschiedene Groessen haben.
  - Dateien: Aenderungen ueber alle Module
  - Aufwand: ~400 LOC
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Keine

### 14.4 Nico-Review Fixes (D11)

- [ ] **Fixes nach Gesamtdurchgang durch Nico** -- Noch unbekannter Scope, Platzhalter fuer Review-Feedback.
  - Dateien: TBD
  - Aufwand: ~500-1000 LOC (geschaetzt)
  - Tag: `[FE-ONLY]`
  - Abhaengigkeit: Alle Waves abgeschlossen

---

## LOC Summary Table

| Wave | Beschreibung | Geschaetzt LOC | FE-ONLY LOC | BACKEND-DEP LOC |
|------|-------------|----------------|-------------|-----------------|
| Q1-Q5 | Querschnitts-Aenderungen | 3.550 | 1.350 | 2.200 |
| 1 | Foundation (Components + Stores) | 3.890 | 3.890 | 0 |
| 2 | Kommunikation + Wiki | 4.000 | 4.000 | 0 |
| 3 | CRM + Finance Overhaul | 7.840 | 4.740 | 3.100 |
| 4 | E-Mail Erweiterungen | 1.200 | 1.200 | 0 |
| 5 | Chat + Helpdesk Extensions | 3.420 | 2.670 | 750 |
| 6 | Work/Projekte + Zeiterfassung | 3.800 | 2.800 | 1.000 |
| 7 | Team/HR + Schichtplanung | 4.240 | 3.840 | 400 |
| 8 | Einkauf + Inventar + Produktion | 4.830 | 3.880 | 950 |
| 9 | Fuhrpark + Rapporte + Vermietung | 4.530 | 3.580 | 950 |
| 10 | Formulare + Vertraege + Kalender + Meetings | 5.190 | 3.590 | 1.600 |
| 11 | Video + Notifications + Dashboard + Berichte + Dokumente | 5.480 | 3.530 | 1.950 |
| 12 | Integration Panels + Settings | 1.850 | 1.730 | 120 |
| 13 | DSGVO + KI + Polish | 4.500 | 2.200 | 2.300 |
| 14 | Visual Polish + Review | 2.500 | 2.500 | 0 |
| **GESAMT** | | **~55.820** | **~41.500** | **~15.320** |

> Hinweis: LOC-Angaben sind Schaetzungen. Mit Iteration, Bug-Fixes und Anpassungen realistisch **~55.000-62.000 LOC** gesamt.

---

## Dependency Graph

```
Wave 1: Foundation ─────────────────────────────────────────────────────────
  │                                                                         │
  ├─ 1.1 TipTap ────┬──→ Wave 2.1 (Kommunikation ReplyComposer)           │
  │                  ├──→ Wave 2.2 (Wiki Editor)                            │
  │                  ├──→ Wave 4.1 (E-Mail TipTap)                          │
  │                  ├──→ Wave 5.6 (Helpdesk Canned Responses)             │
  │                  ├──→ Wave 5.14 (Helpdesk KB Editor)                    │
  │                  └──→ Wave 10.17 (Meeting-Notizen)                      │
  │                                                                         │
  ├─ 1.2 Presence ──┬──→ Wave 5.5 (Chat Presence)                          │
  │                  └──→ Wave 11.1 (Video Meeting)                         │
  │                                                                         │
  ├─ 1.3 Global Search ──→ Wave 13.9 (KI Semantische Suche)               │
  │                                                                         │
  ├─ 1.4 Stores ────┬──→ Wave 2 (Kommunikation + Wiki Stores)              │
  │                  ├──→ Wave 11.2 (Notification Store)                    │
  │                  └──→ Wave 12 (Integrations Store)                      │
  │                                                                         │
  └─ 1.6 formatCurrency ──→ Wave 3.15-16 (MWSt/Waehrung)                  │
                            ──→ Wave 8.5, 8.12 (Einkauf/Inventar)          │
                            ──→ Wave 9.15 (Vermietung)                      │
                            ──→ Wave 10.10 (Vertraege)                      │

Wave 2: Kommunikation + Wiki (abhaengig von Wave 1)
  │
  └─ (keine Blocker fuer spaetere Waves)

Wave 3: CRM + Finance (abhaengig von Wave 1.6)
  │
  ├─ 3.2 Custom Fields ──→ Wave 3.3 (Firma Detail)
  │                    ──→ Wave 5.13 (Helpdesk Custom Fields)
  ├─ 3.4 Duplikaterkennung ──→ Wave 3.9 (CRM Import)
  ├─ 3.7 Consent ──→ Wave 13.1 (Consent erweitert)
  ├─ 3.11 Belegkette ──→ Wave 8.2 (Einkauf Belegkette)
  └─ 3.19 Stunden-zu-Rechnung ──→ Wave 6.2 (Projekt-Button)

Wave 4-14: Weitgehend parallel moeglich (Ausnahmen oben markiert)
```

### Empfohlene Reihenfolge (parallelisierbar ab Wave 3)

```
Woche 01-02:  Wave 1 (Foundation)
Woche 03-04:  Wave 2 (Kommunikation + Wiki)
Woche 05-07:  Wave 3 (CRM + Finance) — groesster Block
Woche 07-08:  Wave 4 (E-Mail) + Wave 5 (Chat + Helpdesk) — parallel
Woche 08-10:  Wave 6 (Work + Zeit) + Wave 7 (Team + Schichten) — parallel
Woche 10-12:  Wave 8 (Einkauf + Inventar + Produktion)
Woche 12-14:  Wave 9 (Fuhrpark + Rapporte + Vermietung)
Woche 14-17:  Wave 10 (Formulare + Vertraege + Kalender + Meetings)
Woche 17-20:  Wave 11 (Video + Notifications + Dashboard + Berichte + Dokumente)
Woche 20-22:  Wave 12 (Integration Panels) + Wave 13 (DSGVO + KI)
Woche 22-24:  Wave 14 (Visual Polish + Review)
```

---

## Backend-Abhaengigkeiten (Luke)

Alle Items die Lukes Backend-Arbeit benoetigen, gruppiert nach seinen Phasen.

### Luke Phase 8 (CRM + Basis-Infra)

| Item | Wave | Beschreibung |
|------|------|-------------|
| 3.1 | 3 | CRM CRUD-Formulare (API-Persist) |
| 3.2 | 3 | Custom Fields (JSONB-Schema) |
| 3.3 | 3 | Firma als eigene Entity (DB-Migration) |
| 3.4 | 3 | Duplikaterkennung (Fuzzy-Matching) |
| 3.5 | 3 | Kontakt-Timeline (Cross-Modul-Daten) |
| Q2 | Q | Kontakt-Dualitaet loesen (gemeinsames API-Modell) |
| 1.2 | 1 | Presence/Status (WebSocket) |
| 11.2 | 11 | Notification Center (WebSocket Push) |

### Luke Phase 9 (Video + Wiki)

| Item | Wave | Beschreibung |
|------|------|-------------|
| 11.1 | 11 | Video Meeting Room (LiveKit Token) |
| 10.12 | 10 | Kalender Video-Button (Zoom/LiveKit) |
| 10.15 | 10 | LiveKit Integration |
| 2.2 | 2 | Wiki Versioning (Server-side) |

### Luke Phase 10 (E-Mail + Integrationen)

| Item | Wave | Beschreibung |
|------|------|-------------|
| 2.1 | 2 | Kommunikation echte Kanaele (IMAP, Teams, WhatsApp) |
| 3.12 | 3 | DATEV-Export (Format-Generator) |
| 3.12 | 3 | Bexio-Sync (API-Connector) |
| 5.15 | 5 | E-Mail-zu-Ticket (IMAP-Listener) |
| Q4 | Q | PDF-Export real (Go PDF-Library) |

### Luke Phase 11 (Office + E-Signatur)

| Item | Wave | Beschreibung |
|------|------|-------------|
| 11.10 | 11 | "In Word oeffnen" — WebDAV-Server fuer direktes Oeffnen/Speichern ohne Download |
| 11.10b | 11 | Collabora Enterprise — WOPI Endpoints (CheckFileInfo, GetFile, PutFile) + Collabora Docker |
| 10.6 | 10 | E-Signatur Skribble (API-Integration) |
| 3.8 | 3 | Newsletter Brevo (API-Connector) |

### Luke Phase 12+ (DSGVO + KI)

| Item | Wave | Beschreibung |
|------|------|-------------|
| Q5 | Q | DSGVO-Tools (Server-side Datenabfrage/-loeschung) |
| 13.2 | 13 | DSGVO-Auskunft (Cross-Modul-Suche) |
| 13.3 | 13 | DSGVO-Loeschung (Kaskadierung) |
| 13.6-13.10 | 13 | KI-Features (Embedding-Service, Proxy) |

---

## npm Dependencies (einmalig, vor Wave 1)

| Package | Version | Zweck | Groesse |
|---------|---------|-------|---------|
| `@tiptap/react` | ^2.x | TipTap React-Bindings | ~50 KB |
| `@tiptap/starter-kit` | ^2.x | Core Extensions Bundle | ~80 KB |
| `@tiptap/extension-link` | ^2.x | Link-Support | ~5 KB |
| `@tiptap/extension-image` | ^2.x | Bild-Embedding | ~5 KB |
| `@tiptap/extension-table` | ^2.x | Tabellen-Support | ~15 KB |
| `@tiptap/extension-task-list` | ^2.x | Checkbox-Listen | ~5 KB |
| `@tiptap/extension-task-item` | ^2.x | Task-List Items | ~3 KB |
| `@tiptap/extension-placeholder` | ^2.x | Placeholder-Text | ~3 KB |
| `@tiptap/extension-code-block-lowlight` | ^2.x | Syntax-Highlighting | ~10 KB |
| `@tiptap/extension-mention` | ^2.x | @Mentions im Editor | ~5 KB |
| `lowlight` | ^3.x | Syntax-Highlighting-Engine | ~20 KB |

**Zusaetzliche Bundle-Groesse:** ~200 KB (vor Tree-Shaking). TipTap ist modular.

---

## CSS/Styling Regeln (fuer alle neuen Components)

1. **Light + Dark Mode** -- CSS-Variablen (`var(--card)`, `var(--heading)`, `var(--muted)`) aus `globals.css`. Keine hardcoded Farben.
2. **Glass/Crystal Mode** -- Overlay-Komponenten muessen mit `.ui-glass` / `.ui-crystal` auf `<html>` funktionieren. `.glass-elevated` fuer non-Radix Overlays.
3. **Desk Theme Kompatibilitaet** -- Alle Background-Farben via CSS-Variablen, nicht Tailwind-Color-Utilities direkt.
4. **Responsive in-Frame** -- Container-Widths 800px-1600px. Flex-Layouts, keine festen Breiten (ausser Sidebar-Panels).
5. **ScrollArea** -- Radix `<ScrollArea>` fuer alle scrollbaren Bereiche (bereits in `globals.css` gestylt).
6. **TipTap Styling** -- `styles/tiptap.css` mit ProseMirror-Overrides die CSS-Variablen respektieren.

---

*Ende des Master-Plans. Dieses Dokument ist die definitive Referenz fuer alle Frontend-Arbeit im KMU Hub Projekt.*
*Letzte Aktualisierung: 2026-02-17 (Office-Strategie: 3-Tier mit Collabora/TipTap/Lokal)*
