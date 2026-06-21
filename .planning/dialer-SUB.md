# dialer — Sub-Terminal-Paket (zu review-reif bringen)

> Copy-paste-Block unten ins KMU-Hub-review-Terminal (Port 5174). **Disjunkt zum Haupt:**
> Haupt baut wiki Phase B (`modules/wiki` + `shared/document` + `shared/RichTextEditor`).
> Sub fasst NUR `modules/dialer/**`, `mocks/handlers/dialer.ts`, `api/*dialer*`, das
> dialer-Settings-Panel + nav/module-settings an, plus eine kleine Aktivitäts-Schreibung
> in kontakte (CRM-Log). Kein Overlap mit wiki/document/RichTextEditor (gescoutet, bestätigt).

## Ist-Stand (gescoutet, 2026-06-21)
dialer ist FE solide und MSW-stateful: Kampagnen-Liste/Detail, 4-Phasen-Workspace (idle→dialing→on_call→wrap_up), Agent-Dashboard, Outcomes-CRUD-Settings — alles gegen stateful MSW. **Lücken (= dieser Batch):** (P3) **keine CTI/CRM-Integration** — kein Click-to-Dial, keine Aktivitäts-Protokollierung beim Anruf, ContactHeroPanel verlinkt nicht ins Kontakte-Modul. (P5) **kein Supervisor/Reporting** — `getCampaignAgents` liefert 3 Agenten, aber kein Component rendert sie; kein Recent-Calls-Log (i18n-Key `dialer.dashboard.recentCalls` da, ungenutzt). **Demo-Tiefe:** ContactQueueTable-Zeilen nicht klickbar (kein DetailModal); hartcodierte deutsche Strings (CallControlBar „Anrufen"/„Abbrechen", Toasts, Empty-States, OutcomeDonutChart-Label „Anrufe"); 2 i18n-Keys fehlen in FR/IT (`dialer.agentStatus.changeStatus`, `dialer.campaign.requiresContacts`); **kein dialer-Eintrag im ModuleSettingsShell** (nur die In-Modul-Outcomes-Seite). Kein LiveKit nötig — Call-Leg ist simuliert.

```
Du bist das Sub-Terminal im Sub-Terminal-5-Phasen-Modus. Arbeitsverzeichnis = dieser KMU-Hub-review-Klon (Dev-Server Port 5174). Das Hauptterminal baut parallel wiki Phase B (modules/wiki + components/shared/document + components/shared/RichTextEditor) — du fasst NUR dialer an: src/modules/dialer/, src/mocks/handlers/dialer.ts, src/api/*dialer*, das dialer-Settings-Panel + module-settings/nav, plus eine kleine Aktivitäts-Schreibung in kontakte (CRM-Anruf-Log). NICHT wiki/document/RichTextEditor anfassen. Sprache: Deutsch (Umlaute, Eszett, Akzente — NIE ASCII-Ersatz).

SCHRITT 0 — aktuellen Stand holen:  git pull --rebase origin main

KONTEXT (Ist-Stand): dialer ist FE solide und MSW-stateful: Kampagnen-Liste (CampaignListPage) + -Detail (CampaignDetailPage), 4-Phasen-Workspace (DialerWorkspacePage: idle→dialing→on_call→wrap_up), Agent-Dashboard (AgentDashboardPage), Outcomes-CRUD (DialerSettingsPage). Daten via api/hooks/useDialer.ts gegen stateful mocks/handlers/dialer.ts; Call-State in stores/dialer.ts (Zustand). Kein LiveKit — die Telefonie ist simuliert (POST /dialer/calls), das reicht. LÜCKEN: keine CTI/CRM-Integration (kein Click-to-Dial, kein Aktivitäts-Log beim Anruf, ContactHeroPanel ohne Kontakte-Link), kein Supervisor/Reporting (getCampaignAgents ungenutzt, kein Recent-Calls-Log), ContactQueueTable-Zeilen nicht klickbar, hartcodierte deutsche Strings, 2 fehlende FR/IT-Keys, kein dialer-Panel im ModuleSettingsShell.

DEIN BATCH — dialer zu „review-reif", 5 Phasen je ein Commit:
- D-1 CTI/CRM-Anbindung (P3): Beim Abschluss eines Anrufs (logCallOutcome/completeWrapUp) eine Aktivität in den Kontakt schreiben (MSW: kontakte-activities-Handler, Typ „Anruf" mit Outcome+Dauer+Notiz) — sichtbar in der Kontakt-Timeline. ContactHeroPanel: „Im CRM öffnen" → Sprung zum Kontakt-Detail (CampaignContact.contact_id). Stretch: Click-to-Dial aus dem Kontakte-Detail (Telefon-Action → dialer-Workspace mit vorbelegtem Kontakt).
- D-2 Supervisor-Dashboard (P5): neue Ansicht (Tab im DialerLayout oder Abschnitt), die getCampaignAgents rendert — Live-Status aller Agenten (verfügbar/Pause/im Gespräch), Anrufe heute je Agent, plus Recent-Calls-Log (nutzt dialer.dashboard.recentCalls; letzte Anrufe mit Kontakt/Outcome/Dauer aus MSW). Aggregat über Kampagnen.
- D-3 Demo-Tiefe Kontakt-Detail: ContactQueueTable-Zeilen GANZ klickbar (div role=button + stopPropagation auf inneren Buttons) → shared/DetailModal mit allen Kontakt-Infos + Anruf-Historie + letztem Outcome. Sticky Close. Tote Buttons prüfen.
- D-4 i18n + Dead-String-Cleanup: alle hartcodierten deutschen Strings durch t() ersetzen (CallControlBar „Anrufen"/„Abbrechen", DialerWorkspacePage-Toast, CampaignDetailPage/AgentDashboardPage Empty/Error-States, OutcomeDonutChart-Label, OutcomeFormDialog/CampaignFormDialog-Labels). Die 2 fehlenden Keys in FR+IT ergänzen (dialer.agentStatus.changeStatus, dialer.campaign.requiresContacts). i18n ×4 vollständig.
- D-5 Settings-Panel + Schlusscheck: dialer-Eintrag im ModuleSettingsShell (settings/-Unterordner + DialerSettingsPanel) mit personal (z.B. Standard-Wrap-up-Zeit, Auto-Advance zum nächsten Kontakt) + tenant (z.B. max. gleichzeitige Anrufe, Recording-Consent-Default, Standard-Outcome) Bereich — kohärent mit der bestehenden Outcomes-Verwaltung. Demo-Tiefe-Audit + QA.

BUILD-+-VERIFY-STANDARD PRO PHASE (verbindlich):
bauen → i18n ×4 (i18n/messages/{de,en,fr,it}.json; {var} NICHT {{var}}; Plural als ICU {count, plural, one {…} other {…}}) → gescopter Typecheck (eigenes tsconfig nach Muster tsconfig.r6check.json über die geänderten dialer-Dateien; node_modules/.bin/tsc -p … --noEmit, foreground, echter Exit, NIE | tail) → eslint src/ --quiet (foreground, echter Exit — CI fährt genau das, tsc-grün ≠ lint-grün!) → Playwright-Screenshot-QA gegen http://localhost:5174 (#/dialer: Kampagnen, Workspace-Phasen, Supervisor, Settings) → die PNGs WIRKLICH ansehen (Raw-Keys/Emojis/Theme/Layout/leere Zustände, mehrere Zustände) → iterieren bis grün → ein Commit.

PROJEKTWEITE STANDARDS (sonst Review-Rückläufer):
Detail = shared/DetailModal (zentriertes Fenster, NICHT Slide-over). GANZE Zeile klickbar. Sticky Back/Close. Leere Zustände = EmptyState mit Aktion. Keine Toast-Stubs/toten Endpoints — echte MSW-Funktion. KEINE Emojis (Personality via Custom-SVG/Motion/Wording). Theme-Tokens. Motion nur transform/opacity. Skeleton statt Spinner. CURRENT_USER aus mocks/data/shared-ids. Neue Dateien unter mocks/data/ brauchen git add -f.

GIT:
Conventional Commits, English imperative, KEINE AI-Attribution. Commit pro Phase mit EXPLIZITEN Datei-Pfaden — NIE git add -A/. Nach jedem Commit: git push origin main; bei Ablehnung git pull --rebase origin main, dann erneut push. Dev-Server (5174) NICHT killen.

ABSCHLUSS: kurze Bilanz — welche Phasen committed (Hashes), QA-Ergebnis je Phase, was offen blieb.
```
