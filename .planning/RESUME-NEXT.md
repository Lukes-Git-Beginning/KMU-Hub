# RESUME — nächster Einstieg (Stand 2026-06-21, Session-Ende)

> **Direkt-Wiedereinstieg.** wiki Phase B komplett + gepusht. SCHRITT 0: `git pull --rebase origin main`.

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
