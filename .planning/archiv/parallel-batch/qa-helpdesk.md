# QA-Protokoll — helpdesk Demo-tief-Pass (Sub-Terminal, `parallel/helpdesk`)

> Klon `…/KMU-Hub-review`, Dev-Port **5174** (untracked `vite.qa.config.mjs`, Demo-Mode an).
> Build-Gate: `npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -iE error` (nie `| tail`).
> Screenshot-QA: `node scripts/qa-helpdesk.mjs <tag>` → `.qa-screenshots/helpdesk-<tag>/` (gitignored).
> Hinweis: die einzige verbleibende Build-„error"-Zeile ist die vorbestehende, harmlose
> `"use memo"`-Directive-Warnung aus `dashboard/DashboardPage.tsx` (kein helpdesk-Bezug).

---

## H-1 — Store-Actions (Fundament) ✅

**Gebaut:**
- `stores/helpdesk.ts`: Store von reiner State-Factory auf `create()(persist((set, get) => …))` umgestellt.
- Neue State-Felder: `threads: Record<string, ThreadMessage[]>` — **für alle 15 Tickets** sinnvolle Demo-Threads geseedet (vorher nur tk-1/2/3 als in-file `MOCK_THREADS`).
- Actions: `addTicket` (auto Ticket-Nr `HD-YYYY-NNNN` aus höchster +1, SLA/Timestamps, prepend), `updateTicketStatus`, `assignTicket` (+Thread-Notiz), `escalateTicket` (Priorität +1 bis `critical`, Thread-Systemeintrag „Eskaliert von X"), `mergeTicket` (Quelle→Ziel, Thread anhängen, Quelle `closed`), `addReply`, `saveCsat`, Canned-CRUD (`add/update/delete`), `saveBusinessHours`, `saveRoutingRules`.
- `HelpdeskPage.tsx`: in-file `MOCK_THREADS` + `getThread` + lokales `ThreadMessage`-Interface entfernt; `ThreadMessage` aus Store importiert; `TicketDetailPanel` liest Thread via `useHelpdeskStore(s => s.threads[id])` (stabile `EMPTY_THREAD`-Fallback-Ref gegen Selector-Churn).

**Verify:**
- Build-Gate: **EXIT 0** (nur die bekannte DashboardPage-`use memo`-Warnung).
- Screenshot-QA `helpdesk-h1`: Ticket-Liste 15 Zeilen, Detail-Slide-over zeigt Thread aus dem Store, KB- & Statistik-Tab ok. **0 Raw-Keys, 0 `{{ }}`, 0 pageErrors.**
- Sichtbares Wirken der Mutationen kommt mit H-3 (UI-Verkabelung).

**Commit:** `feat(helpdesk): add store actions + per-ticket threads` auf `parallel/helpdesk`.

---

## H-2 — Ticket-Detail: custom Slide-over → `shared/DetailModal` ✅

**Gebaut:**
- `TicketDetailPanel` von `fixed inset-y-0 right-0 w-[440px]` (ohne `role="dialog"`, ohne Fokus-Trap) auf `shared/DetailModal` umgestellt: zentriertes Cosmi-Modal mit Gradient-Stripe, Sticky-Header (Subject als `title`, Ticket-Nr als `subtitle`, Status+Prio als `badge`), intern scrollender Body, **Sticky-Footer** mit dem Reply-Bereich.
- Body behält alle Sektionen: SLA-Breach-Banner, Kategorie/Auto-Routing-Chips, SLA-Timer, Beschreibung, Custom-Fields, Meta-Grid, Status-Dropdown, CSAT-Widget, Thread.
- **Tabellenzeile voll tastatur-zugänglich:** `<tr>` → `role="button"` + `tabIndex={0}` + `aria-label` + `onKeyDown` (Enter/Space, `preventDefault`) + `focus-visible:ring`.
- Doppelter Canned-Picker-Button entfernt (Wrapper-State `showCannedPicker` raus) — `CannedResponsePicker` bringt eigenen Trigger mit, wird jetzt direkt gerendert.
- Hardcodes „Auto-Routing" + „SLA: " im Modal auf `t()` umgestellt (neue Keys `helpdesk.ticket.autoRouting`, `helpdesk.ticket.slaLabel`); unbenutztes `X`-Icon entfernt.

**Verify:**
- Build-Gate: **EXIT 0**.
- Screenshot-QA `helpdesk-h2`: Zeilen-Klick → `dialogOpen: true`, `gradientStripe: 1`; Modal zentriert, Sticky-Header + Close, alle Sektionen sichtbar, Footer-Reply sticky; **Escape schließt** (KB-Tab-Navigation danach erfolgreich). **0 Raw-Keys, 0 `{{ }}`, 0 pageErrors.** Screenshot angesehen.

**Commit:** `feat(helpdesk): migrate ticket detail to DetailModal + a11y row` auf `parallel/helpdesk`.

---

## H-3 — Bestehende Aktionen verkabeln (Neu / Reply / Status / CSAT) ✅

**Gebaut:**
- `handleSaveNewTicket` → `addTicket(...)` (statt nur Toast): neues Ticket erscheint **sofort** oben in der Liste mit Auto-Nr.
- `handleSendReply` → `addReply(id, { author: currentUserName(), body, internal })`: Antwort/interne Notiz hängt sichtbar an den Thread; Autor ist der echte eingeloggte Demo-User (`currentUserName()` aus `stores/auth`).
- `handleStatusChange` → `updateTicketStatus(id, status)`: Badge ändert sich in Tabelle **und** Modal-Header.
- `CSATWidget` an Store gebunden: liest `csatRating/csatComment` vom Ticket, Submit → `saveCsat(...)`; `CSATAggregate` rechnet jetzt aus Store-Tickets (via `useMemo`, kein Selector-Churn). `MOCK_CSAT_RATINGS` entfernt (kein anderer Importeur). `key={ticket.id}` am Widget → frischer State pro Ticket.
- QA-Harness `WIPE` auf **einmalig pro Context** (sessionStorage-Flag) umgestellt, damit der Reload im Persistenz-Test den State behält.

**Verify (Flow-QA `scripts/qa-helpdesk-flow.mjs`, Screenshots angesehen):**
- Neues Ticket: Zeilen 15 → 16, neue Zeile oben mit **`HD-2026-0316`**.
- Reply: Thread „Nachrichten (1)" → „(2)", Bubble sichtbar, Autor „Markus Weber", Toast „Antwort gesendet".
- Status-Control vorhanden & wirkt.
- **Reload (ohne Wipe): Ticket + Reply bleiben erhalten** (`newStillThere: 1`, `replyPersisted: 1`).
- Breiter Scan `helpdesk-h3`: **0 Raw-Keys, 0 `{{ }}`, 0 pageErrors.**
- Bekannter Interim-Artefakt: neue Tickets zeigen SLA „122d übrig" wegen eingefrorener `computeSla`-Uhr → behoben in H-7.

**Commit:** `feat(helpdesk): wire ticket/reply/status/CSAT mutations to store` auf `parallel/helpdesk`.

---

## H-4 — Eskalieren / Zuweisen / Mergen (neue Demo-Aktionen) ✅

**Gebaut:**
- Neue **„Aktionen"-Leiste** im Detail-Modal (vor „Status ändern"):
  - **Zuweisen:** Agent-`<select>` (UserPlus-Icon, Demo-Pool `Marco Hartmann` / `Sandra Bürki`, fremder Bestand bleibt als Option erhalten) → `assignTicket` + Toast; No-op bei gleichem Agent (kein Spam).
  - **Eskalieren:** Button (ArrowUp) → `escalateTicket(id, currentUserName())`, Priorität eine Stufe hoch (bis `critical`, dann disabled + Info-Toast), Thread-System­eintrag „Eskaliert von X".
  - **Mergen:** `<select>` der anderen offenen Tickets (GitMerge-Icon) → `mergeTicket(source, target)`: Quell-Thread ans Ziel angehängt, Quelle auf `closed` mit Notiz „Zusammengeführt mit HD-…", Modal schließt + Toast. Leerer Zustand → disabled mit Hinweis.
- Store-Actions + `tickets` via Selektoren in `TicketDetailPanel`; Merge-Targets via `useMemo`.

**Verify (Flow-QA `scripts/qa-helpdesk-actions.mjs`, Screenshots angesehen):**
- Escalate: Priorität **Niedrig → Mittel**, Thread „(1)" → „(2) 1 intern", Eskalations-Notiz vorhanden.
- Assign: tk-5 (default Sandra) → **Marco Hartmann**; Meta + Toast aktualisiert, Thread-Notiz „zugewiesen an Marco Hartmann" (Count 1).
- Merge: 13 Ziel-Optionen, **Modal schließt**, Quell-Ticket danach `Geschlossen` mit „Zusammengeführt mit"-Notiz.
- Scan `helpdesk-h4`: Modal offen, **0 Raw-Keys, 0 `{{ }}`, 0 pageErrors.**

**Commit:** `feat(helpdesk): add assign/escalate/merge actions to ticket modal` auf `parallel/helpdesk`.

---

## H-5 — Canned Responses CRUD verkabeln ✅

**Gebaut:**
- `CannedResponsesPanel.handleSave` → `addCannedResponse` / `updateCannedResponse` (statt nur Toast); `handleDelete` → `deleteCannedResponse`. `handleInsert` unverändert (war schon korrekt).
- Store-Setter via Selektoren geholt; Payload `{ title, content, category, shortcut }` getrimmt.

**Verify (Flow-QA `scripts/qa-helpdesk-canned.mjs`, Screenshots angesehen):**
- Anlegen: „6 Vorlagen verfügbar" → **7**, neue Vorlage „QA Vorlage Persistenz" erscheint oben (prepend) mit `/qa`-Kürzel, Toast „erstellt".
- Reload: Anzahl **7** bleibt, Vorlage persistiert.
- Löschen: **7 → 6** (Trash-Button auf Karten-Hover). `goneNow:1` zählt nur den transienten Lösch-Toast.
- 0 pageErrors.

**Commit:** `feat(helpdesk): wire canned responses CRUD to store` auf `parallel/helpdesk`.

---

## H-6 — Settings-Panel ✅

**Gebaut:**
- Neuer `stores/helpdeskPrefs.ts` (personal, persist `cosmi-helpdesk-prefs`): `startTab` + `defaultStatusFilter`.
- Neuer `modules/helpdesk/settings/HelpdeskSettingsPanel.tsx` über `ModuleSettingsShell` (`moduleId="helpdesk"`):
  - **personal** „Persönliche Ansicht": Start-Tab- + Standard-Statusfilter-Select (→ Prefs-Store).
  - **tenant** „Geschäftszeiten" + „Routing-Regeln" (Modul-Leiter/Admin-gated).
- `BusinessHoursDialog` + `TicketRoutingConfig` um `embedded`-Modus erweitert (Inhalt ohne Dialog-Chrome + Inline-„Speichern"); beide speichern jetzt **echt** via Store-Actions `saveBusinessHours` / `saveRoutingRules` (vorher nur Toast). `TicketRoutingConfig` liest seine Regeln jetzt aus dem Store statt aus lokalem `INITIAL_RULES` (Dubletten-Typ + `TICKET_CATEGORIES` entfernt, `MOCK_CATEGORIES` wiederverwendet).
- Registry-Eintrag `{ id: 'helpdesk', navMatch: ['/helpdesk'], icon: LifeBuoy, … }` in `module-settings-registry.tsx`.
- `HelpdeskPage`: die zwei Header-Buttons (Geschäftszeiten + Routing) inkl. State + Dialog-Renders + Imports entfernt (auch der hardcodierte „Routing"-String ist damit weg); Tab + Statusfilter aus den Prefs initialisiert.
- i18n: `helpdesk.settings.*` (16 Keys ×4) + `moduleSettings.entries.helpdesk` ×4.

**Verify (QA `scripts/qa-helpdesk-settings.mjs`, Screenshots angesehen):**
- Panel: Overlay öffnet auf /helpdesk **mit Helpdesk preselektiert**; „Helpdesk-Einstellungen" + Start-Tab + Geschäftszeiten + Routing-Regeln rendern; **0 Raw-Keys, 0 `{{ }}`, 0 pageErrors.**
- Pref-Konsum: `startTab='statistik'` → Helpdesk öffnet auf **Statistik-Tab** (`activeTab: "Statistik"`, Chart sichtbar).
- Header nach Cleanup: nur noch „Vorlagen" + „Neues Ticket", Liste + Detail-Modal weiter ok, 0 pageErrors.

**Commit:** `feat(helpdesk): add settings panel (prefs + business hours + routing)` auf `parallel/helpdesk`.

---

## H-7 — SortMenu + SLA-Zeit echt ✅

**Gebaut:**
- **SLA-Fix:** `computeSla` von hardcoded `new Date('2026-02-15T11:00:00')` auf **echtes `new Date()`** umgestellt. Damit alle Seed-Tickets glaubhaft bleiben: `MOCK_TICKETS_RAW` (Originalseed) wird über `SLA_SEED` (per-Ticket-Offsets in Stunden) **relativ zu `Date.now()`** neu basiert → `createdAt`/`updatedAt`/`slaDueAt` + recomputed SLA. Bewusster Mix: 5 überfällig, mehrere bald-fällig (<4h, gelb), Rest komfortabel.
- **SortMenu:** `shared/SortMenu` in die Filter-Zeile integriert (Felder Erstellt / Priorität / Status / SLA-Restzeit, Richtung asc/desc). `sortedTickets`-`useMemo` mit Comparators (Prioritäts-/Status-Rank, SLA via `slaDueAt`-Timestamp). Default: Erstellt absteigend.
- i18n: `helpdesk.sort.*` (4 Keys ×4).

**Verify (Flow-QA `scripts/qa-helpdesk-sort.mjs`, Screenshots angesehen):**
- SLA-Clock: **0 frozen-Artefakte** (kein „122d" mehr), 5 überfällig + 10 übrig, Samples plausibel („3h übrig", „1h übrig", „2d 12h übrig", „4h überfällig").
- Sort SLA aufsteigend → erste Zeile überfällig (HD-2026-0311); absteigend → komfortabelste (HD-2026-0312); **beidseitig reordered**.
- Sort Priorität absteigend → erste Zeile **Kritisch**.
- 0 pageErrors.

**Commit:** `feat(helpdesk): real-clock SLA + ticket list sorting` auf `parallel/helpdesk`.

---

## H-8 — i18n-Strings + Demo-Tiefe-Schlusscheck ✅

**Gebaut:**
- **SLA-Strings i18n:** `computeSla` gibt jetzt **strukturiert** `{ overdue, days, hours }` zurück (keine deutschen Strings mehr im Store). `Ticket.slaRemaining: string` → `slaDays`/`slaHours`. Neuer exportierter Formatter `slaLabel(t, ticket)` als Single-Source-of-Truth; baut das Label via `t('helpdesk.sla.{remaining,overdue}{Hours,Days}')` (4 neue Keys ×4). Alle Verbraucher umgestellt: `SLABadge` (Props strukturiert, Yellow-Logik `days===0 && hours<4`), Modal SLA-Timer + `SLABreachBanner` + Warn-Logik in `HelpdeskPage`, **und der Dashboard-Verbraucher `dashboard/widgets/OpenTickets.tsx`** (legitimer Cross-Modul-Consumer der Helpdesk-Daten — kein team-Lane-Overlap).
- **Hardcodes:** „Auto-Routing" + „SLA:" bereits in H-2 i18n'd, „Routing"-Header-String in H-6 entfernt → keine deutschen UI-Hardcodes mehr im Modul (Ticket-Inhalte/Threads bleiben bewusst deutsch = DACH-Demo-Daten).
- **KB-Speichern stateful:** Store um `kbBodies: Record<string,string>` + `saveKbBody` ergänzt. `KBArticleDetail` liest Override aus dem Store (sonst statischer `KB_BODIES`-Fallback), speichert den editierten HTML-Body, rendert ihn im View-Modus **sanitisiert** (`lib/sanitize`); `key={article.id}` gegen State-Bleed beim Artikelwechsel.

**Verify (QA `scripts/qa-helpdesk-i18n.mjs`, Screenshots angesehen):**
- EN-Switch (`cosmi-locale=en`): **15/15 SLA-Badges englisch** („1d 3h left", „4h overdue"), **0 deutsche Reste**, Modal-Aktionen englisch (Escalate/Merge/Change status); **0 Raw-Keys, 0 `{{ }}`, 0 pageErrors.**
- KB-Save: Marker im Body → **nach Reload erhalten** (`afterSave: 1`, `afterReload: 1`), sanitisiert gerendert.

**Commit:** `feat(helpdesk): i18n SLA strings + stateful KB article save` auf `parallel/helpdesk`.

---

## Definition of Done — helpdesk Demo-tief-Pass ✅ (8/8)

Alle 8 Punkte gebaut, jeweils **Build EXIT 0** (einzige „error"-Zeile = vorbestehende DashboardPage-`use memo`-Warnung) + **Playwright-Screenshots angesehen** + Commit&Push auf `parallel/helpdesk`. 0 Raw-Keys / 0 Doppelklammern / 0 Console-Errors über alle Tabs & Modals. **Out of scope (nicht gebaut, wie beauftragt):** TanStack-Migration, MSW-Handler, CRM-Kontakt-Lookup.

**Merge-Hinweis fürs Main-Terminal:** i18n-Konflikt beim finalen Merge erwartet (beide Lanes hängen an `i18n/messages/*.json`) — helpdesk-Keys liegen im `helpdesk.*`-Cluster + ein `moduleSettings.entries.helpdesk` nach dem team-Eintrag; beide Key-Blöcke behalten, dann `npm run build`. Untracked Helfer (`vite.qa.config.mjs`, `scripts/add-helpdesk-i18n.mjs`) reisen nicht mit dem Branch.
