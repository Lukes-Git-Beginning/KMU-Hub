# Plan: Review von Dariens "Cosmi-Tiefe"-Recherche-Plan

**Erstellt:** 2026-05-07 · **Author:** Luke (Claude-assisted) · **Reviewer:** Darien
**Quelldokument:** Dariens "Komplette Funktions-Recherche & Vergleichsanalyse" (Plan-Datum 2026-04-29, Plan-Owner Darien)
**Status:** Approved 2026-05-07. Ausführung blockiert auf Gate S2 (2026-05-24).
**User-Entscheidungen (Luke, 2026-05-07):** Start nach Gate S2 · 27-Modul-Scope wie Darien · Outputs in `docs/research/` · sachlich-bestimmter Tonfall

> **Hinweis für Darien:** Dieser Plan ist Reviews deines Recherche-Plans. Er übernimmt deine Phasen-Struktur A-E als Skelett, korrigiert aber Reihenfolge, Speicherort, Anti-Halluzinations-Hardening und einige Behauptungen, die bei der Codebase-Verifikation nicht durchgegangen sind. Drei-Pässe-Lese-Anforderung (Pass 1 Faktenprüfung, Pass 2 Methodik-Kritik, Pass 3 Strategische Neuausrichtung). Diskussionsanker-Sektion ist am Ende.

---

## Context

Du hast einen 5-Phasen-Recherche-Plan (Phasen A-E + Phase F als separater Implementation-Plan) vorgelegt, der vor Launch (2026-07-01) jedes der 27 Cosmi-Module gegen 3-5 Marktführer auseinandernimmt, Cross-Modul-Strategie + Settings-Architektur + Workspace-Patterns synthetisiert und in einer Discussion-Doc als Diskussionsgrundlage konsolidiert. Ziel: Lücken zur Markttauglichkeit systematisch sichtbar machen, statt sie intuitiv zu raten.

Der Plan ist in der Form sauber strukturiert und in der Absicht richtig. Beim Drei-Pässe-Read sind aber mehrere harte Probleme aufgefallen, die der Plan vor Ausführung lösen muss — sonst produzieren wir 12-17 Tage Output, der gegen falsche Baselines argumentiert und die Pre-Launch-Zeit verbrennt.

Dieses Dokument:
1. Gibt den Drei-Pässe-Critique mit Code-Evidence wieder
2. Übernimmt Dariens Phasen-Skelett A-E als Roadmap-Idee, kalibriert Scope/Reihenfolge/Speicherort/Anti-Halluzinations-Hardening neu
3. Hängt Cross-Cutting-Concerns als parallele ADR-Spur dran (statt sie in Phase D als Synthese-Output zu erwarten)
4. Liefert harte Stop-Kriterien + Verifikations-Block

Kein Code. Kein Commit. Nach Approval folgt die Implementation in einer separaten Session.

---

## Pass 1 — Faktenprüfung: Dariens Behauptungen vs. Codebase-Realität

Drei parallele Explore-Agents haben die Behauptungen gegen die SSOT (`docs/MODULES_SCOPE_MATRIX.md`, `docs/ROADMAP.md`) und den Code geprüft. Befunde:

### 1.1 Tiefen-Schema "shell/basic/moderate" steht nicht in der SSOT

`docs/MODULES_SCOPE_MATRIX.md` (Stand 2026-04-18) klassifiziert Module nach **Sprint-1/Sprint-2 + Pilot-Segment (Dienstleister/Handwerk/Cross)**, nicht nach Tiefen. Dariens vier Tiefenstufen sind eine eigene Heuristik ohne Quellenbasis. Jede Welle-Reihenfolge die darauf aufbaut, klassifiziert gegen eine nicht-existente Taxonomie.

### 1.2 Konkrete Fehl-Klassifizierungen (Code-Evidence)

| Modul | Darien | Realität (Evidence) |
|---|---|---|
| Vermietung | "shell" (UI nur, kein Backend) | `useVermietung.ts` mit TanStack Query · `backend/internal/vermietung/` 6 Files · Migration 000098 (GIST tstzrange-Overlap-Index) · Welle 2A done 2026-04-28 · Coverage 41.3% |
| Kalender | "basic, FE-Mock" | `useCalendars`/`useEvents` Real-Hooks · Backend caldav-Pkg · Migrations 000032/000033 |
| Mails | "basic, FE-Mock" | 9 FE-Files · `useEmail*` Real-Hooks · `backend/internal/email/` · Migration 000041 |
| Kontakte | "basic, FE-Mock" | 15 FE-Files · `useContacts` Real-Hooks · `backend/internal/crm/` · Migration 000007 |
| Zeiterfassung | "basic, FE-Mock" | `useTimeEntries` Real-Hook · `backend/internal/work/` · Migration 000030 |
| Berichte | "basic, FE-Mock" | `useDashboardKPIs`/`useDefinitions`/`useSchedules` Real-Hooks · `backend/internal/berichte/` 18 Go-Files · Migration 000079 |
| Kommunikation | "moderate" mit eigenem Backend | `backend/internal/kommunikation/` **existiert nicht** · UI-Aggregator über inbox/chat/email-Routes |
| Buchhaltung/Finanzen | ein Modul "moderate" | Zwei FE-Ordner: `modules/finanzen/` (live, 20 Files, real wired) · `modules/buchhaltung/` (DEPRECATED, hat `_DEPRECATED.md`, kein Route-Eintrag) |

**Konsequenz:** Dariens Welle 1 ("Shells zuerst") würde mit **Vermietung als ersten Recherche-Kandidaten** starten — einem Modul, das wir gerade Welle 2A in Sprint 2 vollständig wired haben (109 Files Welle 4A am 2026-04-29). Recherche gegen Konkurrenten würde Features aufzählen, die wir bereits implementiert haben.

### 1.3 "27 Module" ist nicht herleitbar

FE-Verzeichnis hat ~37 Ordner, davon 7-8 Infrastruktur (auth, dashboard, notifications, profil, settings, calendar-Duplikat, buchhaltung-deprecated). Tatsächliche Business-Module: ~29 im FE, **14 in der MODULES_SCOPE_MATRIX als "Modul-Scope"** geführt. Die Zahl 27 stammt aus keiner überprüfbaren Quelle.

### 1.4 "Audit-Baseline 2026-04-29" existiert nicht

Explore-Agent hat das Repo + `~/.claude/plans/` + `.knowledge/` durchsucht: kein File mit "audit" + Datum 2026-04-29. Kein `.planning/cosmi-depth/` Ordner. Die Baseline-Behauptungen aus Dariens Plan-Context (3 Persistenz-Mechanismen, 13 Cross-Modul-Integrationen, Settings-Schicht-Empfehlung) sind nicht durch ein referenzierbares Dokument belegt. Falls der Audit existiert (lokal bei Darien?), bitte an `docs/research/` legen, damit wir gegen ihn argumentieren können.

### 1.5 "13 Cross-Modul-Integrationen" — nachweisbar 6

Was sich im Code finden lässt:

| Integration | Evidence |
|---|---|
| Tasks↔CRM/Channel/Message | `task_entity_links` (Migration 000026) · `LinkEntity`/`UnlinkEntity`/`ListEntityLinks` in `backend/internal/work/task/postgres_repository.go` · `useTaskEntityLinks` + `useLinkEntity` Frontend-Hooks |
| Documents↔Entitäten | `document_entity_links` (Migration 000043) · `CreateEntityLink`/`DeleteEntityLink` in `backend/internal/document/file/postgres_repository.go` |
| Dialer↔CRM | `GRPCCRMBridge` mit `CreateCallActivity`/`GetContactDetails`/`ResolveFilterContacts` |
| Meetings↔Tasks | `meeting_action_items.task_id` FK · `ConvertActionItemsToTasks` |
| CRM↔Finance | `finance_quotes.deal_id UUID REFERENCES deals(id)` · Index `idx_finance_quotes_deal` |
| Email↔CRM | `email_contact_links`-Tabelle · `contactlink`-Package · Email-Suche via JOIN |
| Calendar↔Tasks | **kein direkter FK nachweisbar** — Behauptung steht im Plan-Context, ist aber nicht durch Schema belegt |

### 1.6 Settings-Befunde: weitgehend bestätigt, aber unterschätzt

Darien nennt 3 persisted Stores. Der Audit zeigt **~30 persisted Zustand-Stores ohne User-ID-Scoping** (settings, ui, ai, notifications, navigation, finance, auth, formulare, dialer, locale, work, wiki, vertraege, vermietung, timetracking, team, search, schichten, rapporte, produktion, presence, meetings, kommunikation, inventar, integrations, helpdesk, fuhrpark, einkauf, dashboard, contacts, tour, video, mails, automatisierung). Plus 8+ raw-localStorage-Bypässe (`cosmi:brand:logo`, `cosmi:brand:accent`, `cosmi-quicknote`, `cosmi-employee-wizard-draft`, `cosmi-view-pref-${folderId}`, Billing-Insights, Auth-Cache). Plus locale-Inkonsistenz: `users.locale` server-seitig in M040, aber App liest aus 2× localStorage (settings + ui), nicht synchronisiert.

Dariens Diagnose stimmt — die Größenordnung war zu klein angesetzt. Migrations-Aufwand für 3 saubere Levels: **3-5 Entwicklerwochen, 30+ FE-Files anzupassen**.

---

## Pass 2 — Methodik-Kritik

### 2.1 Tiefe-First-Reihenfolge ist Pilot-2/3-First — verkehrte Pre-Launch-Priorität

Dariens Welle 1 (Shells = Vermietung, Fuhrpark, Schichten, Rapporte) sind alle **Pilot-2/Handwerk-Module** laut MODULES_SCOPE_MATRIX. Pilot 1 startet 2026-07-01 mit dem **Dienstleister-Segment** und braucht: helpdesk, vertraege, formulare, wiki, berichte, buchhaltung/finanzen, video + bestehende CRM/Work/Email/Kontakte/Dokumente/Kalender/Chat. Tiefe-First verbrennt die ersten 5-7 Recherche-Tage mit Modulen, die für den Launch keinen Wert liefern.

### 2.2 27 × ≥50 Funktionen = ~1350 Funktions-Einträge — Halluzinations-Risiko

Quellen-Pflicht pro Funktion mitigiert das nur teilweise: Sub-Agents haben Web-Quellen-Drift-Risiko (Marketing-Text als Spec-Quelle), und 1350 Items sind nicht stichproben-reviewbar. Ohne Codebase-Anker werden Sonnet-Sub-Agents Features dokumentieren, die in Salesforce existieren, ohne zu wissen, dass wir die seit Sprint 1 haben (siehe Pass 1.2). Output wird Wishlist statt Backlog.

### 2.3 `.planning/cosmi-depth/` untracked widerspricht direct-to-main-Default

CLAUDE.md sagt: "Direct-to-main ist Default ab 2026-04-18". Plan-Outputs ausserhalb des Repos sind:
- nicht backuped
- bei Maschinenwechsel verloren
- für Annabel/ZFA nicht zugänglich
- nicht reviewbar in PRs

User-Entscheidung: **Outputs landen in `docs/research/` im Repo**, direct-to-main, mit `.gitignore`-Regel nur für transiente Sub-Files.

### 2.4 Discussion-Doc "max 30 Seiten" widerspricht "Dialog-für-Dialog-Tiefe"

≥50 Funktionen × 27 Module + Settings-Levels + Cross-Modul + Workspace-Patterns + Quellen + Detail-Notizen → realistisch 150-200 Seiten oder oberflächliche 30-Seiten-Mappe. Beides ist nicht implementierungsfähig.

### 2.5 Settings-Architektur als Synthese-Output statt eigenem ADR

Die Settings-Probleme sind **bekannte Code-Realität, nicht Recherche-Frage**. Sie brauchen einen ADR mit Migrations-Plan, nicht einen 30-Seiten-Synthese-Anhang. Genauso entity_relations: das ist Architektur-Entscheidung mit Migrations-Auswirkungen, nicht Recherche-Output.

### 2.6 Phase F als Monolith-Plan ist nicht wartbar

"EIN großer Plan-File für ALLES" für 27 Module mit UI/Backend/Settings/Cross-Modul/i18n/Theme/Tests = 500-2000-Zeilen-File, das nach der ersten Implementation-Session veraltet ist. Cosmi-Workflow nutzt Sprint-bezogene Plan-Files in `~/.claude/plans/` mit klarem Scope. Master-Plan-Ansatz ist Scope-Creep-Garantie.

---

## Pass 3 — Strategische Neuausrichtung (Übernahme von Dariens Skelett, neu kalibriert)

User hat 2026-05-07 entschieden: **27-Modul-Scope wie Darien wird beibehalten**, aber Reihenfolge + Speicherort + Anti-Halluzinations-Hardening werden korrigiert.

### 3.1 Welle-Reihenfolge: Pilot-1-First statt Tiefe-First

Begründung: Recherche-Output muss vor Launch (2026-07-01) Pilot-1-Backlog liefern. Handwerk-Module recherchieren wenn sie an die Reihe kommen.

| Neu | Module | Pilot-Bezug | Notiz |
|---|---|---|---|
| **Welle 1** | helpdesk, vertraege, formulare, wiki, berichte, buchhaltung/finanzen | Pilot-1 Sprint-1-Module | Hier liegen die echten Pilot-1-Lücken |
| **Welle 2** | video (Recording-Consent-UX), kalender, mails, kontakte, zeiterfassung | Pilot-1 bestehende Core-Module | Schärfung statt Neubau — Recherche identifiziert UX-Gaps für Pilot-1 |
| **Welle 3** | crm, aufgaben/work, dokumente, chat, meetings, dialer | Bestehende moderate Module | "20 Feature-Phasen komplett" laut ROADMAP — Recherche fokussiert auf Pilot-Feedback-Items, nicht Feature-Liste |
| **Welle 4** | automatisierung, kommunikation, security/admin | Plattform | Settings-/Cross-Modul-Foundation |
| **Welle 5** | rapporte, schichten, fuhrpark, vermietung, inventar, einkauf, produktion | Pilot-2/3 Handwerk | Optional pre-Launch, akzeptabel post-Launch |

### 3.2 Speicherort: `docs/research/` (User-Entscheidung)

```
docs/research/
├── README.md                            # Plan-Pointer + Status-Tracker
├── PLAN-cosmi-tiefe-review.md          # dieses Dokument
├── methodology/
│   ├── research-template.md             # aus Dariens A.3
│   ├── compare-template.md              # aus Dariens A.4
│   ├── competitor-list.md               # 27 Module × 3-5 Konkurrenten
│   ├── settings-framework.md
│   └── workspace-pattern-guide.md
├── 00-codebase-baseline-2026-05.md     # NEU: Phase A0, Modul-Realität pro Modul
├── modules/
│   ├── 01-helpdesk-research.md          # Welle-1
│   ├── 01-helpdesk-compare.md           # Welle-1
│   ├── 02-vertraege-research.md
│   └── ...
├── synthesis/
│   ├── cross-module-integrations.md
│   ├── settings-architecture.md         # → ADR-007 wenn finalisiert
│   ├── entity-relations.md              # → ADR-008 wenn finalisiert
│   ├── workspace-patterns.md
│   └── shared-ui-blocks.md
└── DISCUSSION.md
```

### 3.3 Codebase-Audit als Phase A0 (NEU)

Vor jeder Web-Recherche: ehrliche Bestandsaufnahme der 27 Module aus dem Code. Pro Modul: LOC, RPCs, Tabellen, Frontend-Hook-Status, Feature-Flag-Status, Coverage. Output: `00-codebase-baseline-2026-05.md`. Diese Baseline ist Pflicht-Input für jeden Sub-Agent in Phase B (Anti-Halluzinations-Hardening).

### 3.4 Cross-Cutting-Concerns als parallele ADR-Spur

Drei ADRs werden **parallel zu Phase A-E** geschrieben, nicht als Synthese-Output erst nach Phase D:

- **ADR-007 Settings-Architektur** (Owner: Luke, Review Darien) — 3 Levels (Tenant-Config-API / User-Preferences-API / UI-Session-localStorage), Migrations-Reihenfolge, geschätzter Aufwand. Input: Phase-A0-Settings-Audit (haben wir bereits aus Explore-Agent-Befund). Zeitrahmen: 1 Tag.
- **ADR-008 entity_relations** (Owner: Luke) — Wann polymorpher Linker (`task_entity_links`-Pattern), wann direkter FK. Welche fehlenden Verbindungen sind Pilot-1-Blocker (z.B. Calendar↔Tasks, Form→Contact, Ticket→CRM aus Dariens 14+ Gaps). Zeitrahmen: 0.5 Tag.
- **ADR-009 Anti-AI-Slop-Konsolidierung** (Owner: Luke) — Konsolidierung CLAUDE.md `[[design]]` + Rigorosum-Runde-1+2-Findings + neue Beobachtungen in `docs/ANTI_PATTERNS.md`. Wird Pflicht-Pre-Brief für jeden Sub-Agent in Phase B/C. Zeitrahmen: 2-3 Stunden.

### 3.5 Anti-Halluzinations-Hardening (Pflicht für jede Phase-B-Welle)

- **Pre-Brief pro Sub-Agent enthält:** Codebase-Baseline-Block aus Phase A0 (Modul-spezifisch), Konkurrenten-Liste mit ≥3 Pflicht-Quellen, Anti-AI-Slop-Direktiven aus ADR-009, Verification-Block-Pflicht.
- **Quellen-Pflicht pro Funktion:** Jede behauptete Konkurrenz-Funktion bekommt eine URL-Quelle. Bei Unsicherheit "unklar, prüfen". Sub-Agent-Output wird abgewiesen, wenn ≥10% der Funktionen keine Quelle haben.
- **Stichproben-Re-Check:** Pro Welle wählt Luke 1 Modul, prüft 5 zufällige Funktionen gegen die Quellen. Bei ≥1 Halluzination: ganze Welle wird wiederholt.
- **Gap-Liste statt Funktions-Liste:** Output-Format pro Modul ist "Was haben wir nicht, das Pilot-1 braucht" — nicht "50 Features die Konkurrent X hat". Limit: 20 Gap-Items pro Modul.

---

## Roadmap (Dariens Phasen-Skelett, neu kalibriert)

**Start:** 2026-05-25 (nach Gate S2 — User-Entscheidung)
**Wallclock-Budget:** ~14 Tage parallelisiert (vergleichbar zu Dariens 12-17, da Scope erhalten bleibt aber Hardening + Codebase-Audit dazukommen)

### Phase A — Setup + Codebase-Audit (3 Tage)

- **A.0 Codebase-Baseline (NEU):** Pro Modul Stand vermessen. Output `docs/research/00-codebase-baseline-2026-05.md`. 1 Tag, Hauptsession.
- **A.1 Verzeichnis-Struktur:** `docs/research/`-Skelett angelegen, `methodology/`-Files committed.
- **A.2 Konkurrenten-Liste schärfen:** Dariens Liste (27 Module × 3-5 Konkurrenten) durchgehen + DACH-Bias prüfen (Lexoffice/Bexio/sevDesk statt nur Anglo-Saxon-Suite). Output `docs/research/methodology/competitor-list.md`. 0.5 Tag.
- **A.3 Templates:** Research-Template (Dariens A.3) und Compare-Template (Dariens A.4) committen. 0.5 Tag.
- **A.4 Anti-AI-Slop-Konsolidierung (= ADR-009 parallel):** `docs/ANTI_PATTERNS.md`. 0.5 Tag.

**Pause-Gate A:** Codebase-Baseline existiert, alle 27 Module mit korrektem Tiefen-Status (aus Code, nicht Heuristik). Templates + Konkurrenten-Liste + Anti-Patterns-Doc committed. **Kein Phase B ohne Gate-A-Approval von Luke.**

### Phase B — Per-Modul-Recherche (5-7 Tage, Sub-Agents Sonnet, max 3 parallel)

Welle-Reihenfolge **Pilot-1-First** (siehe 3.1), nicht Tiefe-First.

| Welle | Module | Sub-Agents | Tage | Pause-Gate |
|---|---|---|---|---|
| B1 | helpdesk, vertraege, formulare | 3 parallel | 1.5 | Welle B1 Review: 5 Stichproben pro Modul gegen Quellen, Gap-Listen ≤20 Items |
| B2 | wiki, berichte, buchhaltung/finanzen | 3 parallel | 1.5 | Welle B2 Review |
| B3 | video, kalender, mails | 3 parallel | 1.0 | Welle B3 Review |
| B4 | kontakte, zeiterfassung, crm, work | 3 parallel + Hauptsession für 4. | 1.5 | Welle B4 Review |
| B5 | dokumente, chat, meetings, dialer | 3 parallel + Hauptsession | 1.0 | Welle B5 Review |
| B6 | automatisierung, kommunikation, security/admin | 3 parallel | 1.0 | Welle B6 Review |
| B7 (optional) | rapporte, schichten, fuhrpark, vermietung, inventar, einkauf, produktion | 3 parallel × 2-3 Sub-Wellen | 1.5 | Optional: nur wenn B1-B6 unter Budget. Sonst Post-Launch. |

**Pflicht-Input pro Sub-Agent:** Codebase-Baseline-Block des Moduls, Konkurrenten-Liste, Research-Template, Anti-Slop-Direktiven, Verification-Block-Pflicht.

**Pause-Gate nach jeder Welle**: Luke reviewt 1 Modul stichprobenartig + verifiziert Quellen + entscheidet "weiter"/"zurück".

### Phase C — Per-Modul-Compare (3-4 Tage, Sub-Agents Sonnet, max 3 parallel)

Pro Modul aus Welle B1-B6: Compare-Agent gegen Codebase-Baseline. Output `docs/research/modules/<n>-<modul>-compare.md` mit Gap-Buckets (P0/P1/P2/Skip + Aufwands-T-Shirt). Welle B7 (Handwerk) wird in Phase C nur dann gespiegelt, wenn B7 in Phase B passiert ist.

**Pause-Gate C:** Pilot-1-Module (helpdesk, vertraege, formulare, wiki, berichte, buchhaltung, video) haben prioriserten Backlog. Backlog ist direkt in `docs/ROADMAP.md §4` integrierbar (P0 → Sprint 3, P1 → Sprint 4).

### Phase D — Cross-Modul-Synthese + ADR-Finalisierung (2 Tage, Hauptsession Opus)

- **D.1 Cross-Modul-Integrationen:** Aggregation aus Phase B/C. **Input für ADR-008 entity_relations** (parallele Spur).
- **D.2 Settings-Architektur:** Aggregation aus Phase B/C. **Input für ADR-007** (parallele Spur, sollte zu diesem Zeitpunkt bereits Draft sein).
- **D.3 Workspace-Patterns:** Welche Pattern wiederholen sich, welche cosmi-Module sind heute "verkehrt herum" gebaut.
- **D.4 Shared UI Building Blocks:** Entity-Chips, Save-to-X-Popover, Convert-to-Y-Dropdown, Activity-Timeline, Kommentar-Composer — was muss einmal gut gebaut werden statt 27× ad-hoc.

**Pause-Gate D:** ADR-007 + ADR-008 + ADR-009 in `docs/adr/` mit Decision + Migrations-Plan. Workspace-Pattern-Empfehlung verbindlich.

### Phase E — Discussion-Doc (1-2 Tage, Hauptsession)

`docs/research/DISCUSSION.md` — max **15 Seiten** (statt Dariens 30, weil Codebase-Baseline-Verweise und ADRs separat existieren). Struktur:

- TL;DR (3-5 Bullets)
- Per-Modul-Übersicht: 27 × Tabellenzeile (Tiefe heute, Pilot-1-Pflicht-Lücken, Diff-Chancen, Aufwand-Bucket, Empfehlung)
- Cross-Modul-Strategie: Top-10 Integrations-Lücken priorisiert, Verweis auf ADR-008
- Settings-Architektur-Vorschlag: Verweis auf ADR-007
- Workspace-Pattern-Vereinheitlichung (3-4 Pattern-Empfehlungen)
- Shared UI Building Blocks (6-10 Komponenten)
- Priorisierung: Bis-Launch / Q3-Post-Launch / 2027
- Risiko-Analyse (16 P0-Blocker parallel + Coverage-Hochziehen + Annabel-Onboarding)
- Offene Fragen für Luke+Darien-Diskussion (5-10 Punkte)

**Pause-Gate E:** Discussion-Doc liegt vor, ist scanbar in 30-60 Min, jeder Abschnitt hat Entscheidungsverantwortlichen. **Hand-off zu Diskussions-Session Luke + Darien.**

### Hand-off: Master-Implementation-Plan (NACH Diskussion)

Erwartete Form: **eigene Sprint-3-/Sprint-4-Plan-Files** statt Monolith-Plan (Dariens "ein File für alles" widerspricht Cosmi-Workflow, siehe Pass 2.6).

---

## Cross-Cutting-Concerns als parallele ADR-Spur

Laufen **parallel zu Phase A-E**, nicht als Synthese-Output erst nach Phase D.

### ADR-007: Settings-Architektur

- **Owner:** Luke, Review Darien
- **Output:** `docs/adr/ADR-007-settings-levels.md`
- **Input:** Settings-Audit-Befund (Pass 1.6: ~30 persisted Stores, 8+ raw-localStorage-Bypässe, locale-Inkonsistenz, dual Mail-Settings)
- **Decision-Vorschlag:** 3 Levels — Tenant-Config-API / User-Preferences-API / UI-Session-localStorage. Tabelle pro existierender Store: Ziel-Level + Migrations-Aufwand.
- **Zeitrahmen:** 1 Tag
- **Start:** Phase A parallel
- **Bezug zu Discussion-Doc:** Phase E §"Settings-Architektur-Vorschlag" verweist auf ADR-007 statt Inhalt zu duplizieren

### ADR-008: entity_relations

- **Owner:** Luke
- **Output:** `docs/adr/ADR-008-entity-relations.md`
- **Input:** Cross-Modul-Audit-Befund (Pass 1.5: 6 nachweisbare Verbindungen, polymorphe Linker `task_entity_links`/`document_entity_links` als erprobtes Pattern)
- **Decision-Vorschlag:** Wann polymorpher Linker, wann direkter FK. Welche fehlenden Verbindungen sind Pilot-1-Blocker. Generalisierung zu `entity_relations`-Service vs. Status-quo halten.
- **Zeitrahmen:** 0.5 Tag
- **Start:** Phase C parallel (braucht Gap-Backlog als Input)

### ADR-009: Anti-AI-Slop-Konsolidierung

- **Owner:** Luke
- **Output:** `docs/ANTI_PATTERNS.md`
- **Input:** CLAUDE.md `[[design]]`-Direktiven + Rigorosum-Runde-1+2-Findings + Sprint-2-Welle-3.5-48-Findings
- **Verwendung:** Pflicht-Pre-Brief für jeden Sub-Agent in Phase B/C
- **Zeitrahmen:** 2-3 Stunden
- **Start:** Phase A parallel (kein Gate-Dependency)

---

## Verifikations-Plan

### Phase-Gates (jedes Gate ist Pflicht-Pause für Luke-Review)

| Gate | Kriterium | Stop-Condition |
|---|---|---|
| A | Codebase-Baseline-Doc + Templates + Konkurrenten-Liste + ADR-009 committed | 3 Stichproben aus Baseline gegen Code geprüft, 0 Diskrepanzen |
| B (pro Welle) | Modul-Research-Files committed + Verification-Block enthalten | 5 Stichproben-Funktionen pro Modul mit Quelle prüfbar, ≤10% Halluzinations-Rate |
| C | Pilot-1-Module haben P0/P1/P2-Backlog | Jedes P0-Item hat Code-Referenz + Aufwands-T-Shirt |
| D | ADR-007 + ADR-008 + ADR-009 in `docs/adr/` final | Decision + Migrations-Plan vorhanden |
| E | DISCUSSION.md ≤15 Seiten | Jeder Abschnitt hat Entscheidungsverantwortlichen, offene Fragen markiert |

### Harte Stop-Kriterien

- **Phase B nach 7 Tagen abbrechen** wenn weniger als Welle B1-B5 vollständig dokumentiert. Nicht verlängern, Welle B7 streichen, Welle B6 abkürzen.
- **Welle B7 (Handwerk) streichen** wenn B1-B6 mehr als 5 Tage gebraucht haben. Handwerk-Module recherchieren wir in Sprint 5 (Q3, Pilot-2-Vorbereitung).
- **Gesamtplan nach 14 Tagen stoppen** unabhängig von Fortschritt. Was bis dahin fertig ist, wird verwendet. Was nicht fertig ist, wird als "Recherche ausstehend" in Sprint-3-Backlog eingetragen.
- **Halluzinations-Rate ≥10% in Welle X** → ganze Welle wiederholen, nicht abnicken.
- **Kein Phase-F-Monolith.** Master-Implementation-Plan wird nicht ein einzelner File für 27 Module. Pro Sprint ein eigener Plan-File nach bestehendem Muster.

### End-to-End-Verifikation

Nach Phase E ist der Plan erfolgreich, wenn:
1. `docs/research/00-codebase-baseline-2026-05.md` existiert und alle 27 Module korrekt einordnet
2. `docs/research/modules/`-Ordner hat 20+ Modul-Research-Files mit Quellen
3. `docs/research/modules/`-Ordner hat 7+ Compare-Files für Pilot-1-Module mit Backlog
4. `docs/adr/ADR-007.md`, `ADR-008.md`, `ADR-009.md`, `docs/ANTI_PATTERNS.md` finalisiert
5. `docs/research/DISCUSSION.md` ≤15 Seiten
6. ROADMAP §4 hat Sprint-3/Sprint-4-Backlog-Items aus den Pilot-1-Compare-Outputs übernommen

---

## Risiko-Analyse

### Risiken wenn wir den modifizierten Plan ausführen

- **R1 — Time-To-Launch-Druck.** 14 Tage Recherche zwischen Gate S2 (2026-05-24) und Launch (2026-07-01) lassen 17 Tage für Implementation. Mitigation: Welle-1-Output nach 1.5 Tagen kann **bereits vor Phase E in Sprint 3 fließen** — nicht warten bis alle Phasen fertig sind. Das ist explizit erlaubt.
- **R2 — Halluzinations-Risiko bei Sub-Agents trotz Hardening.** Quellen-Pflicht und Stichproben-Re-Check mitigieren, aber nicht eliminieren. Mitigation: Codebase-Baseline (Phase A0) als Anker — Sub-Agent kann nicht "wir brauchen X" sagen, wenn Baseline zeigt "X ist seit Sprint 1 wired".
- **R3 — Welle B7 (Handwerk) fällt aus.** Akzeptabel — Pilot-2/3 ist Q3/Q4 2026, Recherche kann parallel zu Sprint 5 laufen.
- **R4 — ADR-007 (Settings) verlangsamt Sprint 3.** Settings-Migration ist 3-5 Wochen Aufwand. Pre-Launch unrealistisch komplett umzusetzen. Mitigation: ADR-007 entscheidet Reihenfolge, Sprint 3 macht **nur Tenant-Config-API + Branding-Bypass-Cleanup** (höchste UX-Priorität für Pilot-1), Rest in Sprint 5/6.

### Risiken wenn wir Dariens Plan unverändert übernehmen

- **R-D1 — Recherche gegen falsche Baselines.** Tiefen-Klassifizierung stimmt in mindestens 5 Fällen nicht mit Code überein (Pass 1.2). Sub-Agent-Output diskutiert Features, die wir haben.
- **R-D2 — Pilot-1-Lücken werden 5-7 Tage später erkannt.** Welle 1 (Shells) liefert null Wert für Launch.
- **R-D3 — `.planning/cosmi-depth/` untracked-Verlust.** Maschinenwechsel oder Repo-Reset löscht 14 Tage Arbeit.
- **R-D4 — 30-Seiten-Discussion-Doc kollidiert mit Dialog-für-Dialog-Tiefe.** Output ist entweder zu oberflächlich für Implementation oder zu tief für Diskussion.
- **R-D5 — Audit-Baseline existiert nicht.** Mehrere Plan-Behauptungen ("13 Integrationen", "3 Persistenz-Mechanismen") basieren auf einem Dokument, das im Repo nicht auffindbar ist.

---

## Critical Files (für Plan-Ausführung relevant)

- Dariens Quellplan (lokal bei Luke: `Komplette Funktions-Recherche & Vergleichsanalyse.txt`)
- `docs/MODULES_SCOPE_MATRIX.md` — SSOT für Modul-Klassifizierung + Pilot-Segmente
- `docs/ROADMAP.md` — SSOT für Sprint-Planung + Launch-Datum
- `CLAUDE.md` — Anti-Pattern-Direktiven, Branch-Strategie, Conventional-Commits
- `docs/ARCHITECTURE.md` — ADR-Verzeichnis-Pattern

## Bestehende Funktionen/Patterns die wiederverwendet werden

- **Polymorpher Linker-Pattern:** `task_entity_links` + `document_entity_links` (Migrations 000026 + 000043) als Vorlage für ADR-008-Generalisierung
- **TanStack-Query-Hooks-Pattern:** `useVermietung`/`useContacts`/`useDeals`-Stil für neue Pilot-1-Modul-Wirings
- **Sub-Agent-Welle-Pattern:** Sprint-2-Welle-2A-Methodik (4 parallele Sonnet-Sub-Agents, ein konsolidierter Direct-to-Main-Commit) als Operations-Vorbild für Phase B/C
- **Feature-Flag-System:** Für jede Pilot-1-Modul-Erweiterung neuer `modules.<name>`-Flag wenn nicht vorhanden — wired über `useFeatureFlags`/`FeatureGate`
- **Pause-Gate-Pattern:** User-Review nach jeder Welle vor nächste Welle starten

---

## Diskussionsanker für Phase-E-Session (Luke ↔ Darien)

Drei Punkte, an denen unsere Sichtweisen aktuell auseinandergehen — bewusst hier zentral gelistet, damit wir sie in der Phase-E-Session offen ansprechen können statt sie im Doc verstreut zu lassen:

1. **Codebase-Audit ist nicht optional.** Wenn wir Phase A0 überspringen, recherchieren wir gegen falsche Annahmen — und das wird im Output sichtbar (Sub-Agents werden "Vermietung braucht Backend" als Empfehlung schreiben). Der Audit-Tag in Phase A ist aus Sicht des Reviewers nicht verhandelbar. Wenn du anderer Meinung bist: Diskussionspunkt.

2. **Welle-Reihenfolge ist Pilot-Priorität.** Tiefe-First klingt logisch, aber wenn wir 5 Tage damit verbrennen, dass Sub-Agents Vermietung-Konkurrenten recherchieren während Pilot-1-Helpdesk nicht angefasst wurde, ist das ein Strategiefehler den wir uns 9 Wochen vor Launch nicht leisten. Pilot-1-First ist die Empfehlung — Begründung in 3.1.

3. **ADRs jetzt, nicht als Synthese-Output später.** Settings und entity_relations sind keine Recherche-Fragen. Wir kennen die Probleme aus dem Code (siehe Pass 1.5 + 1.6). Was wir brauchen ist Entscheidung + Migrations-Plan, nicht 30 Seiten Diskussion warum sie wichtig sind. Wenn du das anders siehst — produktives Streit-Thema für die Phase-E-Diskussion.

---

## Status

**Plan finalisiert 2026-05-07. Ausführung blockiert auf:**
1. Approval von Luke (✅ erteilt)
2. Gate S2 erreicht (2026-05-24, Welle 4B + R2-P0.4 Frontend + R2-P0.7 + Option-B Phase 1 done)

**Erste Aktion nach Gate S2:** Phase A starten — `docs/research/`-Skelett anlegen, Codebase-Baseline-Doc schreiben, ADR-009 (Anti-AI-Slop-Konsolidierung) als ersten ADR rausziehen.
