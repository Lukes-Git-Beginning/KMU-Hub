# RESUME — Batch 3 (berichte R-3/R-4 + wiki)

> **Handoff für zwei neue Terminals (Sub-Terminal-5-Phasen-Modus).** Stand siehe unten.
> Hauptterminal (`KMU Hub`-Klon, Dev 5173) baut **berichte R-3 (PDF) + R-4 (Scheduling)**.
> Sub-Terminal (`KMU-Hub-review`-Klon, Dev 5174) baut **wiki**.
> Je 5-Phasen-Batch, dann Review durch Darien. Workflow-Memory: `feedback_sub_terminal_5phase`.

## Was diese Session fertig wurde
- **berichte R-2 + R-2p (B3-1…B3-5):** Lebenszyklus (Status-Buttons Entwurf→Fertig→Freigeben→Archivieren
  + Guards + „Freigegeben am" + Reader-Empty-State) · Reader als **A4-Bögen** auf getöntem Schreibtisch
  mit Akzentlinie + laufender Kopf-/Fußzeile + „Seite X von Y" · **Playfair-Editorial-Typografie**
  (Cover-Titel + H1, self-hosted woff2) + premium Deckblatt + 65ch-Lesebreite · Block-Politur
  (KPI gleiche Höhe, Callout-Icons, Bullets) + Cover bekommt immer eigene Seite · **Print-CSS-Grundlage**
  (`modules/berichte/components/documents/report-print.css`, @page A4 + page-break) + palette-Härtung.
- **berichte Review-Fixes (Darien):** FIX-1 Filter/konkrete Datenauswahl im „Neue Grafik"-Tab
  (Zeitraum + Filter + **echte Werte-Datalist** für String-Felder z.B. Kunde) · FIX-2 Editor-Header
  **sticky** (h-full-Bindung statt flex-1, eigener interner Scroll) · FIX-3 Per-Block-Button in
  Mehrspalten-Zeilen erreichbar (z-20 über Nachbarspalte).
- **formulare Batch B (Sub-Terminal):** FT-2a/2b/3a/3b/5 — alle gemergt.

## SCHRITT 0 (beide Terminals): `git pull --rebase origin main`

---

## Hauptterminal · berichte = R-3 + R-4 (5 Phasen)

Spezifikation: `.planning/berichte-report-authoring-VISION.md` Abschnitte **R-3** (PDF-Export) + **R-4**
(Scheduling). **Viel ist wiederverwendbar** — die `ScheduleList.tsx` hat bereits ein vollständiges
Scheduling-Dialog + Lauf-Historie (`buildRunHistory`, `computeNextRun`) + Toggle/CRUD; `ReportSchedule`
koppelt via `definition_id` (für R-4 = `doc.id` setzen). Print-CSS liegt schon in `report-print.css`.

| # | Phase | Inhalt |
|---|-------|--------|
| B5-1 | R-3a Print/PDF | „Drucken / Als PDF"-Button im **Lese-Modus**-Header → `window.print()`. **`@media print`** in `report-print.css` ergänzen, das die App-Shell ausblendet (Sidebar/Header/Dock — Selektoren aus `components/layout/AppShell.tsx`), sodass NUR die Report-Bögen drucken. Chromium-Druckvorschau muss sauberes A4 zeigen (Deckblatt S.1, Charts/KPIs nicht abgeschnitten, Seitenzahl in Fußzeile). |
| B5-2 | R-4 Scheduling-Modal am Bericht | „Zeitplan einrichten"-Button in der **Editor-Header-Toolbar** (Guard: nur sichtbar/aktiv wenn `status==='released'`, sonst Hinweis „Erst freigeben, dann planen"). Öffnet zentriertes Modal gekoppelt an das **Dokument** (`ReportSchedule.definition_id = doc.id`). **Rhythmus-Picker statt rohem Cron**: täglich / wöchentlich (Wochentag) / monatlich (Tag) / quartalsweise → generiert die cron-Expression. Format (nur erlaubte aus Tenant-Settings) + Toggle aktiv. |
| B5-3 | R-4 Empfänger | Empfänger-Auswahl: **interne Nutzer** (Typeahead aus Nutzerliste, `CURRENT_USER`/shared-ids) + **externe E-Mail-Adressen** (Chips, Validierung — Muster aus `ScheduleList`). |
| B5-4 | R-4 Lauf-Historie + Jetzt senden | Im Modal: letzte 5 Läufe (Datum/Status/Empfänger-Anzahl, `buildRunHistory` wiederverwenden) + **„Jetzt senden"**-Button (manueller Trigger `trigger:'manual'`, stateful MSW-Run) + Vorschau „Nächste Ausführung" (`computeNextRun`). |
| B5-5 | Kopplung + Settings + Polish | „Geplant"-Tab-Liste verlinkt Dokument-Schedules zurück auf den Bericht (Klick öffnet das Dokument). Modul-Einstellungen-Hinweis „Geplante Berichte nur ab Status Freigegeben". Vollständiges i18n-Audit aller neuen `berichte.docs.*`/`berichte.schedule.*`-Keys. |

**Pro Phase:** bauen → i18n ×4 (`{var}`, ICU-Plural) → gescopter Typecheck (`tsconfig.b5check.json` nach Muster
`tsconfig.b2check.json`, neue Dateien aufnehmen; `desktop/node_modules/.bin/tsc -p tsconfig.b5check.json --noEmit`,
foreground) → Playwright-Screenshot-QA gegen 5173 (Muster `scripts/qa-berichte-b3-1.mjs`) → **PNGs wirklich
ansehen** → iterieren bis grün → ein Commit (explizite Pfade) → push (`pull --rebase` über Sub-Terminal-Pushes).
**Backend-Bedarf** (für `backend-gaps.md`): R-3b Server-PDF (Playwright) + R-4 Cron-Executor/Mailer = Luke; FE-Demo vollständig ohne Backend.

---

## Sub-Terminal · wiki (5 Phasen) — copy-paste ins KMU-Hub-review-Terminal

```
Du bist das Sub-Terminal im Sub-Terminal-5-Phasen-Modus. Arbeitsverzeichnis = dieser KMU-Hub-review-Klon (Dev-Server Port 5174). Das Hauptterminal baut parallel berichte R-3/R-4 — du fasst NUR wiki-Dateien + i18n an. Sprache: Deutsch (Umlaute, Eszett).

SCHRITT 0 — aktuellen Stand holen:  git pull --rebase origin main

KONTEXT (Ist-Stand wiki, frisch gescoutet):
- Modul: desktop/src/renderer/src/modules/wiki/ (WikiPage/WikiSidebar/WikiArticle/WikiArticleHeader/WikiEditor/WikiSearch/WikiTreeNode/WikiVersionHistory/WikiVersionItem/WikiTemplateDialog/WikiCategoryDialog/WikiShareDialog).
- Backend-Swap IST DA: api/wiki-types.ts, wiki-client.ts, wiki-adapter.ts, hooks/useWiki.ts (5 Queries + 10 Mutations, TanStack Query), mocks/handlers/wiki.ts (stateful, alle CRUD-Routen). stores/wiki.ts = nur UI-State.
- KRITISCH: WikiEditor.tsx ist ein <textarea>-STUB (Tag-Insert per String-Konkat, speichert {plain:string}). Der fertige Shared-TipTap-Editor liegt unter components/shared/RichTextEditor (StarterKit/Link/Image/Table/TaskList/TextAlign/Underline/Placeholder, compact-Prop, onChange(html)) — der muss rein.
- i18n: ~47 wiki.*-Keys da, keine Roh-Keys. wiki.settings.* fehlt (kein Settings-Panel). Settings-Spec: .planning/nico-block/wiki-settings.md.
- Stubs/Lücken: Share-Dialog ruft useCreateShareToken NICHT auf (nur lokaler State); Anhänge-Hooks da, kein UI; Kategorie-umbenennen-Hook da, kein UI; Versions-Restore-Hook da, kein Button; Suche client-seitig (useSearchArticles ungenutzt); viewCount/isPinned/tags/authorName kommen leer aus dem Adapter.

DEIN BATCH — 5 Phasen, je ein Commit:
- W-1 TipTap-Editor: WikiEditor.tsx auf den Shared-RichTextEditor umstellen (statt textarea-Stub). Speicherformat auf echtes TipTap/HTML umstellen (wiki-adapter.ts: nicht nur {plain} extrahieren). MSW-PUT legt eine neue Version an (Version-Append im stateful Store). Restore-Button in WikiVersionItem.tsx verdrahten (useRestoreVersion existiert). Read-/Edit-Mode sauber abgegrenzt.
- W-2 Anhänge + Kategorie-Management + echte Freigabe: Anhänge-Panel im Artikel-Detail (useAttachments/useUploadAttachment/useDeleteAttachment — alle da). Kategorie umbenennen/löschen-UI in WikiTreeNode (useUpdateCategory/useDeleteCategory). WikiShareDialog ruft wirklich useCreateShareToken auf + zeigt/kopiert den generierten Link (PATCH /categories/:id-Handler in mocks/handlers/wiki.ts ergänzen falls nötig).
- W-3 @Mention + [[Wikilink]]: TipTap @Mention-Extension (Muster aus chat MentionAutocomplete) + Custom-Extension WikiLink für [[Artikel-Titel]] (neue Datei extensions/WikiLinkExtension.ts), Autocomplete-Vorschläge aus useArticles; Klick auf internen Link öffnet den Artikel.
- W-4 Suche + Demo-Tiefe-Felder: echte API-Suche (useSearchArticles mit Debounce statt nur Client-Filter). viewCount-Increment (MSW: bei GET Artikel-Detail erhöhen). isPinned/Tags mock-first via Store (bis Backend liefert) + Adapter ergänzen. authorName via User-Lookup.
- W-5 wiki-Settings-Panel + Abschluss: WikiSettingsPanel.tsx unter modules/wiki/settings/ via ModuleSettingsShell (nach .planning/nico-block/wiki-settings.md — Persönlich: Standard-Ansicht/Editor-Breite/Sidebar-Default via stores/wikiPrefs.ts; Für-alle mock-first: Freigabe-Defaults/Public-Toggle/Kategorie-RBAC-Hinweis). Eintrag in module-settings-registry.tsx. i18n wiki.settings.*. Schlusscheck/Demo-Tiefe.

BUILD-+-VERIFY-STANDARD PRO PHASE (verbindlich):
bauen → i18n ×4 (i18n/messages/{de,en,fr,it}.json; {var} NICHT {{var}}; Plural als ICU {count, plural, one {…} other {…}}) → gescopter Typecheck (desktop/tsconfig.wikiBatch.json nach Muster von tsconfig.formularecheck.json; desktop/node_modules/.bin/tsc -p tsconfig.wikiBatch.json --noEmit, foreground) → Playwright-Screenshot-QA gegen http://localhost:5174 (Muster: ein bestehendes scripts/qa-*.mjs; Hash-Route /#/wiki) → die PNGs WIRKLICH ansehen (Raw-Keys/Emojis/Theme/Layout/leere Zustände, mehrere Datensätze + Breiten) → iterieren bis grün → ein Commit.

PROJEKTWEITE STANDARDS (sonst Review-Rückläufer):
Detail = shared/DetailModal, ganze Zeile klickbar (innere Buttons stopPropagation). Sortierung Feld+Richtung via shared/SortMenu. Sticky Back/Close. Leere Zustände = EmptyState mit Aktion. Keine Toast-Stubs/toten Endpoints — echte MSW-Funktion. KEINE Emojis. Theme-Tokens (bg-info-light, text-success …). Skeleton statt Spinner. CURRENT_USER aus shared-ids.ts. Neue Dateien unter mocks/data/ brauchen git add -f.

GIT:
Conventional Commits, English imperative, KEINE AI-Attribution. Commit pro Phase mit EXPLIZITEN Datei-Pfaden (git add desktop/.../wiki/... i18n/messages/*.json tsconfig.wikiBatch.json) — NIE git add -A/.  Nach jedem Commit: git push origin main; bei Ablehnung git pull --rebase origin main (i18n-Konflikte: beide Key-Bereiche behalten), dann erneut push. Dev-Server (5174) nicht killen.

ABSCHLUSS: kurze Bilanz — welche Phasen committed (Hashes), QA-Ergebnis je Phase, was offen blieb.
```
