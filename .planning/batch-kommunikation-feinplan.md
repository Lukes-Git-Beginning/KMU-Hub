# Batch-Feinplan: team-Lohn + Kommunikation-Merge (2026-06-08)

> **Status: REVIEW — warten auf Dariens Go, dann bauen.** Fünfer-Phasen-Batch nach Standard-Workflow.
> Entscheidungen Darien: Merge-UI = **Umschalter oben (Team | Posteingang)** · Audio/Video = **echte Bridge ins video-Modul**, Bots/Slash = **Mock-Shell + Doku** · Phase-5-Zuschnitt **so lassen**.

## Gesamtplan-Position
- **Phase 1** = Vorbau zu **team P3** (DATEV-Lohn), Cluster 2 Arbeit, ~#50–53 von ~140–162.
- **Phasen 2–5** = **kommunikation/chat P2–P5**, Cluster 1 Vertrieb & Kommunikation, ~#8–11.

## Backend-Lage (geprüft 2026-06-08)
Luke hat **fast alles** gebaut. FE muss primär **verdrahten**, nicht mocken.
- `proto/chat/v1`: Channels CRUD+Archiv, Members Join/Leave/Rolle, Messages Send/Edit/Delete, DMs 1:1, Threads, **SearchChat**, Reactions, Unread/MarkRead, **GetUserMentions**, File-Download/Thumbnail/List.
- `proto/inbox/v1`: List/Get, Read/Unread/Star/Archive, **Snooze/Unsnooze**, **Reply**, **Assign/Claim**, Bulk, **TeamInbox CRUD+Member**, **RoutingRule CRUD+Test**.
- **Fehlt im Backend (→ backend-gaps.md):** Inbox-Status (offen/wartend/gelöst/geschlossen), echtes Inbox-Threading (mehrere Msg/Conversation), Tags-CRUD, Forward, SLA; Chat Gruppen-DMs, Pin/Lesezeichen, Notification-Settings/Channel; **Audio/Video-RPCs, Bot/Webhook/Slash-RPCs** (kompletter Neubau).

---

## Phase 1 — Lohn-Stammdaten am Mitarbeiterprofil (team)
**Ziel:** „Personalstammblatt" digital — alle DATEV-Lohn-relevanten Stammdaten am Mitarbeiterprofil, speist PayrollPrepPanel.
**Quelle:** `.planning/team-lohn-stammdaten-spec.md` (recherchiert, fix).
**Neue Dateien:**
- `lib/payroll-enums.ts` — Steuerklasse(I–VI), Konfession(rk/ev/keine), Beschäftigungsart(VZ/TZ/Minijob/Midijob/Werkstudent/Azubi), SV-Status, Entlohnungsart(Festgehalt/Stundenlohn), Familienstand, Geschlecht(m/w/d). Alle i18n-keyed.
- `stores/payrollMasterData.ts` — persist `Record<employeeId, PayrollMasterData>`, get/set, `isComplete(id)`.
- `modules/team/EmployeePayrollData.tsx` — collapsible Sektion, 5 Sub-Gruppen (Steuer · SV · Beschäftigung · Bezüge · Bank), view+inline-edit, **hr_only**.
**Geändert:** MemberDetailPanel (Sektion einhängen, RBAC) · PayrollPrepPanel `buildRow` nutzt echte Felder + Vollständigkeits-Plausi vor Export.
**Backend (Luke):** `EmployeeProfile` um Lohnfelder erweitern + DSGVO-Sicht. FE-Overlay-Store bis dahin.
**i18n ×4** · **QA:** `scripts/qa-payroll-masterdata.mjs` (hr_only sichtbar, Edit speichert, Badge bei fehlenden Pflichtfeldern).
**DSGVO:** besondere Personaldaten, nur HR/Lohn, Aufbewahrung 6 J. (§41 EStG).

---

## Phase 2 — Modul-Merge-Fundament (kommunikation/chat P2)
**Ziel:** chat + kommunikation → EIN Modul „Kommunikation", Umschalter oben (Team | Posteingang), kein Feature-Verlust.
**Geändert:**
- `modules/kommunikation/KommunikationPage.tsx` → Root mit Segmented-Control „Team | Posteingang" oben links; rendert `<TeamChatView>` (= bisheriges ChatLayout) oder `<InboxView>` (= bisherige Inbox).
- `modules/chat/ChatLayout.tsx` → wird zu `TeamChatView`, eingebettet (ohne eigene Route-Logik).
- `App.tsx`: `/chat/*` → Redirect `/kommunikation?bereich=team`; `/kommunikation` nimmt `?bereich=`.
- `components/layout/sidebar/nav-items.ts`: zwei Einträge → **ein** Eintrag „Kommunikation" (Icon MessageSquareText), alter chat-Eintrag raus. Badge = Summe Unread Team+Posteingang.
- `modules/settings/module-settings-registry.tsx`: neuer Eintrag `{ id:'kommunikation', group:'module', navMatch:['/kommunikation','/chat'], component: KommunikationSettingsPanel }`.
**Neu:** `modules/kommunikation/KommunikationSettingsPanel.tsx` (Skelett, gefüllt in P5).
**i18n:** `kommunikation.bereich.team/posteingang`, Merge der `chat.*`-Keys bleibt (kein Rename nötig).
**QA:** `scripts/qa-komm-merge.mjs` — Umschalter wechselt, beide Bereiche rendern, /chat redirectet, ein Nav-Eintrag, keine Raw-Keys.

---

## Phase 3 — Team-Chat scharfschalten (kommunikation/chat P2)
**Ziel:** vorhandenes chat-Backend verdrahten statt Stubs.
**Geändert/verdrahtet:**
- **Volltextsuche:** Search-Button im ChannelHeader → Such-Panel gegen `SearchChat` (Backend existiert).
- **Member-Panel:** Users-Button → `components/chat/ChannelMemberList` einhängen (existiert, nur kein onClick).
- **Channel Join/Leave:** UI-Einstieg (Hooks existieren).
- **Auto-mark-read:** `useMarkChannelRead` bei Channel-Selektion aufrufen.
- **Inline-Edit/Delete:** `window.prompt`/`confirm` → echtes Inline-Textarea + Kontextmenü (Mutations existieren).
- **Reactions:** die echte `components/chat/ReactionBar`+`ReactionPicker` (frimousse, `useToggleReaction`) in MessageBubble einhängen; Mock-`useState`-Variante entfernen.
- **File-Upload:** `sendMessage` überträgt `pendingFiles` (FileInfo existiert).
- **Mentions-Inbox:** kleine View über `GetUserMentions`.
**Mock:** fehlende MSW-Demo-Handler für search/reactions/files ergänzen (Demo-Mode-Parität).
**Backend (Luke):** Gruppen-DMs, Pin/Lesezeichen → backend-gaps.
**i18n ×4** · **QA:** `scripts/qa-team-chat.mjs` (Suche liefert, Member-Panel, Inline-Edit, Reaction toggelt, Mark-read).

---

## Phase 4 — Posteingang scharfschalten (kommunikation/chat P3)
**Ziel:** Inbox-Backend verdrahten + UI-Einstiegspunkte.
**Geändert/verdrahtet:**
- **Snooze/Claim/Assign:** UI-Buttons in Thread-Header/Liste (Hooks + Backend existieren; `_assignMsg`/Claim bisher ungenutzt).
- **Team-Inbox + Routing-Rules:** aus In-page-Modals → ins **KommunikationSettingsPanel** (Für-alle), Backend CRUD existiert.
- **Bulk-Toolbar:** Mehrfachauswahl → BulkMarkRead/BulkArchive (Hooks existieren, kein UI).
- **Status-Filter:** real machen (Mock-first, da Backend kein Status-Feld) — `stores`-Overlay `Record<msgId,status>`, Filter funktioniert, **verdrahtungs-bereit**.
- **Threading:** Conversation zeigt mehrere Nachrichten (Mock-first Overlay bis Backend), klare Adapter-Schnittstelle.
- **Tags:** Add/Remove Mock-first Overlay + Doku.
- **Edit/Delete/Forward:** Forward real wo möglich, sonst Mock + Doku.
**Mock:** MSW-Demo-Handler für teams/rules/snooze/claim ergänzen (heute fehlen sie → Demo-Mode-Fehler).
**Backend (Luke):** Inbox-Status, echtes Threading, Tags-CRUD, Forward, SLA → backend-gaps.
**i18n ×4** · **QA:** `scripts/qa-inbox.mjs` (Snooze/Assign/Bulk, Status-Filter, Settings-Panel zeigt Teams+Rules).

---

## Phase 5 — Synergie + Moduleinstellungen + Review (P4/P5, FE-mockbar)
**Ziel:** „Inside-Thread"-Synergie (Front/Missive-Pattern) + settings-komplett + die zwei BE-Neubau-Bereiche als verdrahtbare Mock-Shell.
**Synergie:**
- **Interne Notizen** in Kunden-Thread real verdrahten (InternalNoteComposer ist heute No-op) — als eigene Nachrichten-Art, kundenseitig unsichtbar.
- **@Kollegen-Mention** im Kunden-Thread (MentionAutocomplete aus Chat wiederverwenden).
- **Collision-Hinweis** „X bearbeitet gerade" (Mock-first Presence-Overlay).
**Audio/Video:** Call-Button (Channel + Kontakt) → **echte Bridge** ins `video`-Modul (LiveKit existiert), Kontext mitgeben.
**Bots/Slash-Commands:** **Mock-Shell** — Slash-Command-Palette (`/` öffnet Liste: /giphy, /umfrage, /erinnerung), Webhook-Config-Liste im Settings-Panel. Sieht fertig aus, Logik = Doku für Luke.
**Moduleinstellungen** (`KommunikationSettingsPanel`, ModuleSettingsShell):
- **Persönlich:** Standard-Bereich (Team/Posteingang), Benachrichtigungen (pro Channel mute), Dichte, eigener Status, Enter-zum-Senden.
- **Für-alle:** Kanäle (E-Mail/WhatsApp/Widget connect-Shell), Routing-Rules, Team-Inbox, Canned-Responses-Verwaltung, Retention-Hinweis (6 J. Handelsbrief), Webhook-Config.
**Backend (Luke):** Audio/Video-RPCs, Bot/Webhook/Slash-RPCs, Notification-Settings/Channel → backend-gaps.
**i18n ×4** · **QA:** `scripts/qa-komm-synergy.mjs` (interne Notiz unsichtbar-markiert, @mention, Call-Bridge navigiert, Slash-Palette öffnet, Settings-Panel Persönlich+Für-alle).

---

## DACH/DSGVO (aus Markt-Recherche, einzubauen)
- **Retention-Hinweis** 6 J. (Handelsbriefe §257 HGB/§147 AO) im Settings-Panel.
- **Presence datenschutzgerecht:** Status nur selbst setzbar, KEIN Admin-Activity-Report (§87 BetrVG) — als Feature kommunizieren.
- **WhatsApp:** Connect-Flow mit DPA/EU-Server-Hinweis + Double-Opt-In (Shell + Doku).

## backend-gaps.md — Sammelliste für Luke (am Batch-Ende)
Inbox-Status · Inbox-Threading · Tags-CRUD · Forward · SLA · Gruppen-DMs · Pin/Lesezeichen · Notification-Settings/Channel · Audio/Video-RPCs · Bot/Webhook/Slash-RPCs · WhatsApp/Widget-Adapter · EmployeeProfile-Lohnfelder.
