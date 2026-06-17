# Sub-Terminal — helpdesk Demo-tief-Pass (H-1 … H-8)

> **Du bist das Sub-Terminal im Klon `…/KMU-Hub-review`, Dev-Port 5174.** Lies zuerst `.planning/parallel-batch/README.md` (Lane-Regeln, Build-+-Verify-Standard, Gates). Du baust **nur helpdesk**. team gehört dem Main-Terminal — fass es nicht an.
>
> **Selbst-enthaltend:** Scope ist „Demo-tief" (siehe README „Entscheidungen"). **KEIN** TanStack-Migration, **KEIN** MSW-Handler, **KEIN** CRM-Kontakt-Lookup — die bleiben Lukes Backend / späterer Batch. Bau die 8 Punkte ohne Rückfragen ab. Melde Darien nach jedem Punkt „H-x fertig, n/8".

## Ausgangslage (Ist-Abgleich 2026-06-17)
helpdesk ist **UI-vollständig, aber state-tot.** Alle Daten kommen aus `stores/helpdesk.ts` (Zustand + persist, key `cosmi-helpdesk`). Der Store ist eine **reine State-Factory ohne Actions** (`create<HelpdeskStore>()(persist(() => ({ tickets: MOCK_TICKETS, … })))`, L241–254) → **jede Mutation ist heute ein `toast`-Stub und persistiert nichts.** Das ist der Kern-Befund.

`HelpdeskPage.tsx` (~997 Z.) enthält alles inline: Ticket-Liste (Tab `tickets`, L317–444), KB (Tab `wissensdatenbank`, L450–486), Statistik (Tab `statistik`, L491–561), `TicketDetailPanel` (custom fixed Slide-over, L667–924), `KBArticleDetail` (L926–996). Threads sind als **in-file `MOCK_THREADS`** (L113–128) nur für tk-1/tk-2/tk-3 vorhanden.

Hilfskomponenten: `SLABadge.tsx`, `CSATWidget.tsx`, `CannedResponsePicker.tsx`, `CannedResponsesPanel.tsx` (Slide-over CRUD), `BusinessHoursDialog.tsx`, `TicketRoutingConfig.tsx`.

i18n: helpdesk-Strings flach unter `helpdesk.*` in `i18n/messages/de.json` (~Z. 3460–3598) + en/fr/it.

## Branch-Setup (einmalig, ZUERST — Sicherung gegen main-Konflikte)
Bau **NICHT** direct-to-main. Erstelle einmal deinen Isolations-Branch: `git checkout -b parallel/helpdesk`. Alle H-Punkte committest + pushst du auf **diesen** Branch (`git push -u origin parallel/helpdesk`). **Kein** `git pull --rebase` von main nötig — dein Branch ist isoliert. Das Main-Terminal merged `parallel/helpdesk` am Ende kontrolliert.

## Workflow pro Punkt
bauen → i18n ×4 (`{var}`, ICU-Plural) → Store-Daten falls nötig → Compile-Gate (`npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error`, **nie `| tail`**) → Playwright-Screenshot-QA gegen **:5174** + **Bilder ansehen** → iterieren → commit + push auf `parallel/helpdesk` → Eintrag in `qa-helpdesk.md`.

---

### H-1 — Store-Actions (FUNDAMENT, zuerst)  `[KERN]`
**Ist:** `stores/helpdesk.ts` L241–254 — Store hat **keine** `set()`-Actions. Mutationen verpuffen.
**Soll:** Store auf `create<HelpdeskStore>()(persist((set, get) => ({ …state, …actions })))` umbauen und Actions ergänzen:
- `addTicket(input)` → neues Ticket mit auto Ticket-Nr (Format `HD-2026-0316` ff., aus höchster bestehender Nr +1), `createdAt/updatedAt`, SLA via `computeSla`, Status `open`. Vorne in die Liste.
- `updateTicketStatus(id, status)` → Status + `updatedAt` setzen.
- `assignTicket(id, agent)` → `assignedTo` setzen + `updatedAt`.
- `escalateTicket(id)` → Priorität eine Stufe hoch (bis `critical`) + Audit-/Thread-Eintrag.
- `addReply(id, { author, body, internal })` → an Thread anhängen (siehe Threads unten), `updatedAt`.
- `saveCsat(id, rating, comment)` → `csatRating/csatComment` am Ticket.
- `addCannedResponse / updateCannedResponse / deleteCannedResponse`.
- `saveBusinessHours(hours, holidays)` · `saveRoutingRules(rules)`.
- **Threads in den Store ziehen:** `threads: Record<string, ThreadMessage[]>` als State, **für alle 15 Tickets** sinnvolle Demo-Threads seeden (heute nur 3). `MOCK_THREADS` aus `HelpdeskPage.tsx` entfernen, Page liest `useHelpdeskStore(s => s.threads[id])`.
**Verify:** (kommt mit H-3 sichtbar) — hier nur Store + Build grün.

### H-2 — Ticket-Detail: custom Slide-over → `shared/DetailModal`  `[PATTERN]`
**Ist:** `TicketDetailPanel` (`HelpdeskPage.tsx` L667–924) ist ein **custom `fixed inset-y-0 right-0 z-40 w-[440px]`** (L718) ohne `role="dialog"`, ohne Fokus-Trap. Zeilen-Klick (`<tr onClick>`, L393) hat **kein** `role="button"`/`tabIndex`.
**Soll:** Auf `shared/DetailModal` umstellen (zentriertes Cosmi-Modal, wie kontakte/dokumente/work/vertraege). Referenz: `modules/vertraege` (V-1 aus letztem Batch) + `shared/DetailModal`-Nutzung in `modules/kontakte`.
- **Sticky Header:** Ticket-Nr + Subject; Status-Badge + Prioritäts-Badge; Close sticky, nie wegscrollen.
- Body behält **alle** Sektionen: Beschreibung, SLA-Badge + ggf. Breach-Banner, Custom-Fields, Thread (Reply + interne Notizen umschaltbar), Canned-Picker, AI-Suggestion-Button, CSAT-Widget.
- **Ganze Tabellenzeile klickbar:** `<tr>` → `role="button"` + `tabIndex={0}` + `onKeyDown` (Enter/Space). Falls innere Aktions-Buttons in die Zeile kommen → `e.stopPropagation()`.
**Verify:** Zeile klicken → Modal zentriert; alle Sektionen da; Close sticky beim Scrollen; Escape schließt; keine Raw-Keys.

### H-3 — Bestehende Aktionen verkabeln (Neu / Reply / Status / CSAT)
**Ist (alles `toast`-only):** `handleSaveNewTicket` (L226–230), `handleSendReply` (L232–237, auch interne Notiz), `handleStatusChange` (L238–240), `CSATWidget.handleSubmit` (`CSATWidget.tsx` L65–68, nur lokaler State).
**Soll:** Auf die H-1-Actions umstellen. Nach „Neues Ticket" erscheint es **sofort** in der Liste; „Antwort senden" hängt sichtbar an den Thread; Status-Wechsel ändert Badge in Tabelle **und** Modal; CSAT landet am Ticket (überlebt Reload via persist). Toasts bleiben als Bestätigung, aber der State wirkt jetzt.
**Verify:** Jede Aktion sichtbar im UI + nach Reload erhalten.

### H-4 — Eskalieren / Zuweisen / Mergen (neue Demo-Aktionen)
**Ist:** Existieren gar nicht (kein Button, keine Action).
**Soll:** Im Ticket-Detail-Modal (H-2) eine Aktionsleiste: **Zuweisen** (Agent-Dropdown → `assignTicket`), **Eskalieren** (`escalateTicket` → Priorität hoch + Thread-Systemeintrag „eskaliert von X"), **Mergen** (Demo: Ticket in ein anderes offenes Ticket mergen — Thread anhängen, Quell-Ticket auf `closed` mit Hinweis „zusammengeführt mit HD-…"). Agentenliste: die vorhandenen Namen aus `MOCK_ROUTING_RULES`/Demo (`Marco Hartmann`, `Sandra Bürki`) — **kein** echtes Team-Lookup nötig (das wäre CRM/Backend).
**Verify:** Alle drei wirken sichtbar im Store/UI, mit Thread-Spur.

### H-5 — Canned Responses CRUD verkabeln
**Ist:** `CannedResponsesPanel` (`handleSave` L83–95, `handleDelete` L97–99) ändert nur lokalen `useState` → Liste bleibt unverändert.
**Soll:** Auf H-1-Store-Setter (`addCannedResponse`/`updateCannedResponse`/`deleteCannedResponse`). Anlegen/Bearbeiten/Löschen sind sofort in der Liste sichtbar und überleben Reload. `handleInsert` (L101–105) ist schon korrekt — nicht anfassen.
**Verify:** Neue Vorlage anlegen → erscheint; löschen → weg; Reload → Stand erhalten.

### H-6 — Settings-Panel (Standard: jedes Modul settings-komplett)
**Ist:** helpdesk fehlt in `modules/settings/module-settings-registry.tsx` (L62–83). BusinessHours + Routing sind als **Header-Buttons** in `HelpdeskPage.tsx` (L265–281) verbaut — Inkonsistenz ggü. allen anderen Modulen.
**Soll:** `HelpdeskSettingsPanel` neu anlegen (Muster: `modules/settings/panels/TeamSettingsPanel.tsx` / andere Panels), Sections in `ModuleSettingsShell`:
- `personal` (scope `personal`): persönliche Helpdesk-Defaults (Standard-Tab, Standard-Filter/Ansicht).
- `tenant` Section(en): **Geschäftszeiten** (`BusinessHoursDialog`-Inhalt) + **Routing-Regeln** (`TicketRoutingConfig`-Inhalt). Speichern via H-1-Actions (`saveBusinessHours`/`saveRoutingRules`).
- Eintrag in `module-settings-registry.tsx` ergänzen: `{ id: 'helpdesk', navMatch: ['/helpdesk'], … }`.
- Die zwei Header-Buttons in `HelpdeskPage.tsx` entfernen (oder auf „→ Einstellungen öffnen" umbiegen).
**Verify:** Modul-Einstellungen öffnen → helpdesk-Eintrag da, personal + tenant; Geschäftszeiten/Routing dort editier- & speicherbar.

### H-7 — SortMenu + SLA-Zeit echt
- **SortMenu:** Ticket-Liste nutzt heute nur Filter-`<select>`s (L320–374), keine Sortierung. `shared/SortMenu` integrieren (Felder: Erstellt-Datum / Priorität / Status / SLA-Restzeit, Richtung asc/desc).
- **SLA-Fix:** `computeSla` (`stores/helpdesk.ts` L172) nutzt hardcoded `new Date('2026-02-15T11:00:00')` → alle SLAs eingefroren. Auf **echtes `new Date()`** umstellen. **Aber** dann wären alle Feb-2026-Seed-Tickets überfällig → Seed-`slaDueAt`/`createdAt`/`updatedAt` **relativ zu `new Date()`** generieren (Mischung aus offen/bald-fällig/überfällig für eine glaubhafte Demo). `new Date()` ist in App-Code erlaubt (nur Workflow-Scripts verbieten es).
**Verify:** Spalten sortierbar beidseitig; SLA-Badges zeigen plausible Rest-/Überfällig-Zeiten relativ zu heute; Mischung sichtbar.

### H-8 — i18n-Strings + Demo-Tiefe-Schlusscheck
- **Hardcoded Strings → `t()`:** `HelpdeskPage.tsx` „Routing" (L274), „Auto-Routing" (L744–745), „SLA: " (L753). SLA-Einheiten-Strings (`'h übrig'`, `'d'`, `'h überfällig'`) aus `stores/helpdesk.ts` (L176–181) i18n-fähig machen (Einheit als Key, Wert per Interpolation — oder `computeSla` gibt strukturierte `{ hours, days, overdue }` zurück und das Label baut die UI via `t()`).
- **KB-Aktionen:** „War das hilfreich?" (L989) + „Artikel speichern" (L960) sind Toast-Stubs — für Demo-Tiefe mindestens den Speichern-Flow stateful machen (KB-Artikel-Edit in Store) oder ehrlich als „Demo"-Hinweis kennzeichnen.
- **Schlusscheck:** restliche helpdesk-Buttons/Aktionen durchgehen (Toast-only / console.log / nichts) und verkabeln oder ehrlich kennzeichnen. 0 Raw-Keys, 0 `{{var}}`.
**Verify:** Sprache auf EN umschalten → keine deutschen Hardcodes mehr; SLA-Texte übersetzt; keine toten Buttons im Review-Blick.

---

## Definition of Done (helpdesk review-reif)
Alle 8 Punkte verifiziert (Screenshots **angesehen**), 0 Raw-Keys / 0 Doppelklammern / 0 Console-Errors, jede Phase ein Commit+Push auf `parallel/helpdesk`, `qa-helpdesk.md` gepflegt. Dann Darien Bescheid: „helpdesk 8/8 fertig". **Out of scope (NICHT bauen):** TanStack-Migration, MSW-Handler, CRM-Kontakt-Lookup.
