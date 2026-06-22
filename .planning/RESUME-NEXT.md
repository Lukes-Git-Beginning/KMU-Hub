# RESUME — nächster Einstieg (Stand 2026-06-22, Session-Ende)

> **▶▶▶ NÄCHSTER BATCH GESCHNÜRT — direkt in NEUEM Terminal loslegen:**
> - **SCHRITT 0 (beide Terminals):** `git pull --rebase origin main`.
> - **HAUPT-Terminal** (`KMU Hub`, 5173) = **Paket A: kommunikation/chat → review-reif** → `.planning/batch-NEXT2-A-kommunikation.md` (KO-1…KO-10).
> - **SUB-Terminal** (`KMU-Hub-review`, 5174) = **Paket B: formulare → review-reif** → `.planning/batch-NEXT2-B-formulare.md` (FO-1…FO-10).
> - Ablauf je Terminal: **Recherche-Auftrag abarbeiten → gebündelte Gate-Fragen an Darien → erst dann autonom 10 Phasen bauen.** Disjunkt (A=`modules/kommunikation`+`chat.ts`, B=`modules/formulare`+`formulare.ts`); i18n getrennt (`kommunikation.*`/`chat.*` vs `formulare.*`).
> **Paket-Wahl (Begründung):** beide FE-mock-first, disjunkt, KEINE offenen Reviews als Input. kommunikation (3-Panel da, kein Settings-Panel, DMs/Suche/Edit-Löschen offen) + formulare (P1 DnD/DSGVO/Mail + Tiefe). Alternative falls Darien P0 priorisiert: **security/DSGVO** statt formulare (aber backend-lastiger 🔒).
>
> **✅ VORIGER BATCH FERTIG (beide):**
> - **Paket A — mails → review-reif** (MA-1…MA-10, `e2213408`…`86de8f53`). Review-Paket: `.planning/mails-REVIEW-paket.md`. Stateful-MSW, Thread-Konversation+Inline-Bild+Zitat-Toggle, Multi-Account+Unified, Filter+Sort, Vorlagen-CRUD, Labels+Regeln, CRM-Panel, Bulk+Shortcuts, Settings-Panel. Wartet auf Darien-Review → dann Nico.
> - **Paket B — Block-Engine Spezialblöcke (DB-1…DB-10)** gebaut (document/wiki/berichte), bis `b7f68afe`. Wartet ebenfalls auf Review.

> **▶▶ (ARCHIV — Batch-Vorgabe, erledigt für Paket A):**
> - **SCHRITT 0 (beide Terminals):** `git pull --rebase origin main` (Luke pusht nachts — heute schon LiveKit).
> - **HAUPT-Terminal** (`KMU Hub`, 5173) = **Paket A: mails → review-reif** → `.planning/batch-NEXT-A-mails.md` (MA-1…MA-10).
> - **SUB-Terminal** (`KMU-Hub-review`, 5174) = **Paket B: Block-Engine Spezialblöcke + Diagramm-Datenquellen** → `.planning/batch-NEXT-B-blockengine.md` (DB-1…DB-10).
> - Ablauf je Terminal: **Recherche-Auftrag abarbeiten → gebündelte Gate-Fragen an Darien → erst dann autonom bauen.** Pakete sind disjunkt (A=`modules/mails`, B=`components/shared/document`); i18n-Keys getrennt (`mails.*` vs `document.*`/`blocks.*`).
>
> **Begründung Paket-Wahl (2026-06-22 Ist-Check):** Tracker war veraltet — *mails* ist KEIN Neubau (3455 Z., MSW da, nur Settings-Panel fehlt) → Tiefe-Pass bringt ein ganzes Modul für Nico. Block-Engine-Spezialblöcke (`defineBlock`-API steht; fehlend: toggle/code/table/attachment/columns/chart) zahlen auf wiki **und** berichte ein. Beide sind FE-mock-first, brauchen KEINE offenen Darien-Reviews als Input.
> **Offen geblieben (Darien klärt beim Gate):** dein wiki-Phase-B- + berichte-Review steht noch aus → fließt NICHT in diese zwei Pakete (additiv gehalten), kann aber parallel laufen.
> **Nebenstrang (separat, nicht Teil des Batches):** Cosmi-Prod-Desktop gegen Hetzner installiert (CORS-Fix `e25cc411` gepusht); Onboarding offen — Luke legt Darien/Nico als Mitarbeiter an (Anleitung in der Session). Untracked `desktop/scripts/qa-dialer-callflow.mjs` noch zu committen/verwerfen.

> **Direkt-Wiedereinstieg (Vorgängerstand).** wiki Phase B komplett + gepusht.

> **★ NEUER WORKFLOW ab jetzt ([[feedback_ten_phase_autonomous_batch]]):** 2 Terminals × **10 Phasen**. Ablauf: ich schnüre beide Pakete (disjunkt) → jedes Terminal **recherchiert** seine 10 Phasen → stellt Darien **gebündelt die offenen Fragen** (Gate VOR dem Bauen) → baut **autonom 10 Phasen** durch → Darien reviewt **alle 20** → Feedback als neue Phasen in `MASTER-TRACKER.md` → frisches Terminal. **Kandidaten-Scope fürs nächste Paket** (aus `MASTER-TRACKER.md` ziehen): wiki Phase B + berichte **Darien-Review-Findings** als Fix-Phasen · wiki **Batch 2** Spezialblöcke (Toggle/Code/Tabelle/Anhang) · kaskadierende Datenquellen-Auswahl (Diagramm-Blöcke) · nächste ⬜-Module (kommunikation, mails, security/DSGVO, settings P2). dialer macht aktuell der Sub.

## Was diese Session fertig wurde
- **★ wiki Phase B KOMPLETT — review-reif (Darien-Review → dann Nico).** Der wiki-Editor läuft jetzt auf der gemeinsamen Block-Engine (`components/shared/document`), wie berichte. Commits:
  - **PB-1** `4b92105d` — Kern-Switch: wiki-Registry, rows-Seeds (8 Demo-Artikel als Block-Dokumente), Adapter (`extractRows`/`rowsToHtml`/`htmlToRows`), Editor→`DocumentBlockEditor`, Reader→`DocumentReader` web.
  - **CI-Fix** `3aefb538` — 2 Lint-Errors (toter `DocumentReader`-Import aus Phase A + `WikiRichEditor` ref-in-render) raus; WikiRichEditor gelöscht.
  - **PB-2** `666a6798` — frameless Long-Form-Block (Seiten-Look statt boxed Karten) + Bubble-Menü mit H1/H2/H3+Listen + Cover/Icon-Kopf. `RichTextEditor` bekam `frameless`-Prop (additiv).
  - **PB-3 + Feedback** `00e24706` — Block-Heading-TOC (`tocFromRows` + `headingAnchorId`), **Anhänge auch im Edit-Modus**, **Bild-Block per Datei-Auswahl** (statt „Bild-URL").
  - **PB-4a** `a30ba713` — `shared/LinkPreviewPopover`: Klick auf `[[Link]]`/`@Mention` → Vorschau-Karte → Sprung. `RichTextEditor` `extraExtensions`-Prop (WikiLink/WikiMention erhalten beim Edit).
  - **H1-Fix** `26640b59` — kryptisches „H1"-Pill → klarer „ÜBERSCHRIFT [1][2]"-Größen-Umschalter.
  - **PB-4b** `714fa3d2` — Inline-Autocomplete: `[[` → Artikel-Picker, `@` → Personen-Picker (`WikiSuggest`, zero-dep, `onEditorReady`-Prop). Notion-Stil.
  - **PB-5** `3923f200` — tote Extensions (Callout/Details/Figure/lowlight) gelöscht; `adaptVersion` projiziert Block-Versionen→HTML (Diff funktioniert wieder).
  - Jede Phase: gescopter tsc + **`eslint src/ --quiet`** (neues Pre-Push-Gate, [[feedback_lint_before_push]]) + Playwright-Screenshot-QA verifiziert. QA-Skripte: `desktop/scripts/qa-wiki-pb{1..5}.mjs`.
- **notifications (Sub) review-reif** — N-1…N-5 vom Sub-Terminal (Quiet-Hours/DND wirksam, Store-Kohärenz, Nav+Seeds, Pin/Dismiss persistiert, Settings).

## Läuft gerade (Sub-Terminal, Port 5174)
- **dialer D-1…D-5** — Paket `.planning/dialer-SUB.md` (CTI/CRM-Log, Supervisor-Dashboard, Kontakt-DetailModal, i18n-Cleanup, Settings-Panel). Disjunkt zu wiki. Mehrere D-Commits sind beim Rebase schon auf main eingelaufen.

## Offen / als Nächstes
- **Darien-Review wiki Phase B** (frameless Editor, Bild-Picker, Link-Popover, Inline-`[[`/`@`) → Findings als FIX-Phasen → dann Nico. Auch berichte-Review steht noch aus.
- **wiki Spezialblöcke (Batch 2):** Toggle/Aufklapp, Code (lowlight neu), einfache Tabelle, Anhang-Block — als Defs in `wikiBlockRegistry` (Erweiterungspunkt steht).
- **Großer Block (gemeinsam):** kaskadierende Datenquellen-Auswahl für Diagramm-Blöcke → in alle Dokument-Erstellungs-Stellen. Siehe [[project_document_engine]].
- **LinkPreviewPopover** auch in berichte verwenden (Muster steht in `shared/`).

## Build-+-Verify-Standard (pro Phase, beide Terminals)
bauen → i18n ×4 (`{var}`, ICU-Plural) → gescopter tsc (foreground, echter Exit) → **`eslint src/ --quiet`** → Playwright-Screenshot-QA + PNGs WIRKLICH ansehen → ein Commit (explizite Pfade) → push (pull --rebase über parallele Pushes). Latenz: vorbestehender `useTasks.ts`-tsc-Fehler ignorieren.

## Pläne/Pakete
`.planning/wiki-phaseB-VISION.md` (umgesetzt) · `.planning/dialer-SUB.md` · `.planning/notifications-SUB.md` · `.planning/MASTER-TRACKER.md`. Detail-Verlauf: [[project_resume_log]].
