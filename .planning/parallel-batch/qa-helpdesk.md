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
