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
