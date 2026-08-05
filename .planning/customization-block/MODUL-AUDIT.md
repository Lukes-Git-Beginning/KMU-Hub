# Modul-Audit — Editor-Pivot Rollout (Stand 2026-07-25, Session #28)

> **1. Schritt aus `EDITOR-PIVOT-SPEC.md` §42.** 3 parallele Audit-Agents über alle 33 Module.
> Frage: Welches Modul ist wieviel Instrumentierung wert + wo sind Architektur-Blocker?

## ★ Zentraler Befund: Router- vs. State-Tabs (verändert R2 + Pilot-Wahl)

Der Editor blockt via `useBlocker(true)` **alle** React-Router-Navigationen. Damit gilt:
- **STATE-Tabs** (`useState`/`activeTab` + `onClick`) → funktionieren im Editor sofort, begehbar. ✅
- **ROUTER-Tabs** (`NavLink`/`Outlet`/`useNavigate`) → im Editor **blockiert**, Sub-Tabs unerreichbar. ❌

**Ergebnis der Erhebung:** Von allen Modulen sind **nur 2 router-basiert**: `kontakte` (CRM-Shell `KontakteLayout`, 6 Tabs) und `dialer` (`DialerLayout`). **Alle anderen ~14 lohnenden Module sind state-basiert** → R2 „begehbare Tab-Leiste" ist dort praktisch **geschenkt** (ihr Root-`*Page` enthält die State-Tab-Leiste schon).

**Die Ironie:** Der bisher gewählte Pilot **Kontakte ist genau einer der beiden Härtefälle** — sein `editorModules.Component` zeigt zudem nur auf `KontaktePage` (den „Kontakte"-Unterreiter), nicht auf `KontakteLayout` (Root mit Tab-Leiste). R2 für Kontakte = echte Arbeit (Sandbox-eigener Router ODER State-Tab-Adapter). R2 für finanzen/helpdesk/inventar/… = fast nichts.

## Tier-Einordnung (verfeinert durch Audit)

| Tier | Module | Warum |
|---|---|---|
| **Rich** (state, dichte Wertelisten + Detail-Modal) | finanzen, helpdesk, formulare, inventar, einkauf, vertraege, produktion, vermietung, work | viele umbenennbare Objekt-Nomen + Status/Prio/Typ-Chips; Detail-Modal vorhanden; alle State-Tabs |
| **Rich (blockiert)** | **kontakte** (Pilot) | inhaltlich reich, aber Router-Tabs → R2-Sonderfall |
| **Medium** | kalender, zeiterfassung, rapporte, fuhrpark | State-Tabs, weniger branchenspezifische Wertelisten |
| **Thin** | team, dokumente, berichte, wiki, schichten | wenig feste Labels; Inhalt ist User-Content/Backend-Daten |
| **Kein-Ziel** | dialer | Router-NavLinks → im Editor funktionslos |

## Weitere strukturelle Befunde

- **Custom-Fields im Detail schon gerendert:** nur `helpdesk` (readonly im Detail-Panel) + `work` (via API in `TaskDetailPage`). **Kontakte rendert Custom-Fields NICHT im Detail** (`ContactDetailPanel`), nur in den Settings → R3-Tiefe-2 für Kontakte = zusätzlicher Render-Pfad.
- **`resolveValueSet` wird bisher nur für 2 Sets genutzt** (`deal_stages`, `ticket_priority`). Fast alle Module tragen Status/Prio/Typ als **feste i18n-Enum-Keys** (z.B. `finanzen.status.paid`, `produktion.priority.p1`, `vertraege.type.mietvertrag`). → Pro Modul Scope-Entscheidung: **(a)** nur umbenennen (EditableText auf den Enum-Key, billig) oder **(b)** in echte Wertelisten migrieren (Optionen hinzufügen/entfernen/umfärben, mächtig, mehr Arbeit).
- **Detail-Öffnen:** finanzen/helpdesk/kalender/formulare/inventar/einkauf/vertraege/produktion/vermietung/fuhrpark/rapporte nutzen bereits `shared/DetailModal` (Cosmi-Standard ✅). Kontakte/work/dokumente/wiki nutzen eigene Panels/Slide-over.

## Instrumentierbare Inhalte je Rich-Modul (Bau-Steuerung, i18n-Keys)

- **finanzen** — 13 Tab-Labels (`finanzen.tabs.*`, `buchhaltung.tabs.*`), Spalten (`finanzen.col.*`), Rechnungs-/Angebots-Status (`finanzen.status.*`, `finanzen.quoteStatus.*`). ⚠ „Dashboard"-Tab hartkodiert (kein Key).
- **helpdesk** — Tabelle (`helpdesk.table.*`), Statistik-Labels (`helpdesk.stats.*`), Status/Prio (`helpdesk.status.*`, `helpdesk.priority.*`), Kategorien aus `MOCK_CATEGORIES` (Array, kein Value-Set).
- **formulare** — 3+4 verschachtelte State-Tabs, 11 Feldtyp-Keys (`formulare.fieldType.*`), Form-/Submission-Status. Dichteste Label-Landschaft.
- **inventar** — 4 Tabs, `inventar.movementType/locationType/inventurStatus/status.*` (alle feste Keys).
- **einkauf** — 4 Tabs, `orderStatusLabels`/`contractStatusLabels`.
- **vertraege** — 4 Tabs, `contractTypeConfig` (6 Typen, branchen-USP), `statusConfig`.
- **produktion** — 4 Tabs, `orderStatus/priority(p1-p5)/machineStatus/workStepStatus` — Prioritäten umbenennen sehr wertvoll.
- **vermietung** — 3 Tabs, `objectType.*` (Gerät/Raum/Fahrzeug/Werkzeug — pro Vermieter andere) + 3 Status-Sets.
- **work** — State-View-Switcher (List/Kanban/Gantt), echte Custom-Fields via API, `work.projects/tasks.title` im LABEL_WHITELIST. ⚠ Task-Status kommt aus Backend (kein i18n-Key).
- **kontakte** (Pilot) — Kategorien schon instrumentiert (`kontakte.category.*`), `crm.contacts/companies/deals.title` im Whitelist, `deal_stages` registriert. Router-Tabs = R2-Sonderfall.

## Aufwand (verfeinert ggü. Spec-Schätzung ~25–30)

- **Motor (einmalig): ~4–5** — R2 (state-Module fast frei; CRM/Router = Sonderweg), R1-Mutations-Guard, R4 Datenmodell `moduleAreas`, R4 Toggle-UI + Modul-Konsum, Chrome/EditableText-Ausbau.
- **Pro Modul: ~14** lohnende (9 Rich + kontakte + 4 Medium). Thin (5) optional/billig, dialer raus. Ab Pilot per Sub-Agents parallelisierbar (mechanisch).
- **Backend (Luke, parallel): ~3** · **QA/Polish: ~2.**
- **Summe FE ~20–21.** Erste **~8–10** (Motor + 1 Pilot + Top-5 state-Module) = vorzeigbares Werkzeug.
