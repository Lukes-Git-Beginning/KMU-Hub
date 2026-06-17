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
