# wiki — Knowledge-Base-Ausbau (Vision & Phasen-Plan)

> Ziel: das wiki-Modul vom soliden Gerüst (W-1…W-5) auf **berichte-Niveau** heben — echte Tiefe
> statt Stubs, Premium-Lesen/Schreiben, durchdachte Workflows, CRM-Integration, dezente Lese-KI.
> **wiki-sinnvoll gedacht** (Darien 2026-06-21): nicht berichte-Features kopieren, sondern das Niveau.
> Use-Case: **intern, Team-Wissen** (Onboarding, Prozesse, How-Tos). Share = teamintern, kein
> öffentliches Hilfe-Center. **KI nur konsumentenseitig** (Lesen/Suchen), NICHT beim Erstellen.

## Designprinzipien (Cosmi)
- **Findability vor Volumen** — Struktur, Tags, Cross-Links, Suche müssen tragen, bevor Inhalt wächst.
- **Vertrauen** — jeder Artikel hat Owner + „zuletzt geprüft am"; veraltetes Wissen wird sichtbar.
- **Editorial, ruhig, daily-use** (Apple-Linse) — Lesen ist die 100×/Tag-Aktion, die muss exzellent sein.
- **Joy an Onboarding/Empty-States/Success** (Discord-Linse), nicht im Schreib-Flow.
- Keine Emojis in der UI, Theme-Tokens, shared/DetailModal, sticky Back/Close, Skeletons.

## Ist-Stand W-1…W-5 (Basis vorhanden)
TipTap-Editor (StarterKit, Tabellen, Tasklisten, Underline, Link, Image, @Mention, [[Wikilink]]),
Kategorien-CRUD, Server-Suche (`/search`), Versionierung+Restore (stateful), Anhänge (Metadaten),
Share-Token, Tags/Pins (mock-first Store), Settings (personal+tenant). i18n 126 Keys ×4.

### Konkrete Lücken (aus Ist-Analyse 2026-06-21)
- **Unterkategorien erscheinen nie** — `WikiSidebar` rendert flach, `WikiTreeNode` ohne Rekursion. Hierarchie faktisch kaputt.
- **Tags nur lesbar**, **Titel nicht umbenennbar**, **kein „Artikel verschieben"** (Kategorie wechseln).
- **Versionen ohne Diff/Notiz** (`changeNote` immer `''`), kein Inhalts-Preview alter Stände.
- **Anhänge ohne Download/Vorschau** (`file_ref` = nur Dateiname), **kein echter Upload-Blob**.
- **Share ohne Verwaltung**: kein Token-Listing, `revokeShareToken` nirgends in UI, MSW nicht stateful, Link `cosmi://` statt sinnvollem internen Verweis.
- **Volltextsuche im Listen-Endpunkt defekt** (`GET /articles?search=` prüft `typeof content==='string'` → trifft TipTap-JSON nie; nur `/search` funktioniert).
- Tote i18n-Keys (`wiki.article.you`, `wiki.actions.pinNotAvailable`); `wiki.header.views` ohne ICU-Plural.

## Markt-Best-Practices (Recherche 2026-06-21)
KI-Suche & Auto-Zusammenfassung = der 2026-Trend · Wissens-Vernetzung (Backlinks/Graph, Nuclino-USP) ·
Vertrauen via Page-Owner + „zuletzt geprüft" · saubere IA (Bereiche nach Funktion/Prozess) · echte Vorlagen.
Quellen: nuclino.com/solutions/wiki-software · document360.com/blog/wiki-software · tettra.com/article/wiki-best-practices-tips · slab.com/blog/internal-wiki

---

## Phasen-Plan — 4 Batches (je ~5 Phasen)

### Batch 1 · Tiefe & Korrektheit — `WT-1…WT-5` (Pflicht, zeitnah → macht wiki „ehrlich fertig")
FE-mockbar vollständig (MSW-Erweiterungen). Bringt wiki schnell auf einen review-reifen Kern.

| # | Phase | Inhalt |
|---|-------|--------|
| WT-1 | Kategorie-Hierarchie | `WikiTreeNode` rekursiv (Kinder via `parent_id`), Expand/Collapse echt, Top-Level-Filter; **Artikel verschieben** (Kategorie wechseln, Kontextmenü/Detail). |
| WT-2 | Tags & Titel | Tags editierbar (hinzufügen/entfernen, Vorschläge aus vorhandenen) + **Tag-Filter** in der Liste; Titel inline umbenennbar (Header). Tags von mock-Store → MSW-Feld. |
| WT-3 | Versionen mit Substanz | Change-Note beim Speichern (optionales „Was geändert?"-Feld) + **Diff/Preview** zweier Stände (added/removed) vor Restore. |
| WT-4 | Anhänge & Freigabe | Anhang-Download + Bild-Vorschau (Demo-Blob/Data-URL); **Share-Verwaltung**: aktive Links listen + widerrufen (MSW stateful), interner Team-Verweis statt `cosmi://`. |
| WT-5 | Suche-Fix + Politur | Listen-Volltextsuche reparieren (Content-Walk), Treffer-Highlight; tote i18n-Keys raus, `views` als ICU-Plural; Demo-Tiefe-Audit. |

### Batch 2 · Premium Lesen & Schreiben — `WP-1…WP-5` (das „berichte-Editorial-Niveau")
FE-mockbar vollständig. Hebt das tägliche Lese-/Schreiberlebnis.

| # | Phase | Inhalt |
|---|-------|--------|
| WP-1 | Editorial-Lese-Ansicht | Ruhige Lesetypografie (Lesebreite aus Prefs), **Breadcrumbs** (Bereich › Kategorie › Artikel), Lesezeit, Meta-Leiste (Owner/aktualisiert). |
| WP-2 | Inhaltsverzeichnis | **TOC** aus H1–H3 mit Anker-Sprung + Scroll-Spy (aktueller Abschnitt), bei langen Artikeln sticky an der Seite. |
| WP-3 | Editor-Tiefe | **Slash-Command-Menü** (`/`) + neue Blöcke: Callout (Info/Warnung/Tipp), Code mit Syntax-Highlight, Toggle/Details (aufklappbar), Trenner. |
| WP-4 | Artikel-Identität | Cover-Bild + Icon/Emoji-Ersatz (Custom-SVG/Initial) pro Artikel; in Liste + Lese-Kopf sichtbar. |
| WP-5 | Vorlagen-CRUD | Eigene Vorlagen statt 3 hardcoded: erstellen/bearbeiten/löschen, Vorschau im Dialog (MSW). |

### Batch 3 · Vertrauen & Vernetzung — `WV-1…WV-5` (Workflow + Knowledge-Graph)
FE-mockbar; Review-Erinnerungen/Notify echt = Backend.

| # | Phase | Inhalt |
|---|-------|--------|
| WV-1 | Lebenszyklus | Status-Kette **Entwurf → Veröffentlicht → Review fällig → Archiviert** (statt nur `published`), Status-Badge + Übergänge im Header (Muster: berichte-Lebenszyklus). |
| WV-2 | Owner & Aktualität | **Page-Owner** („verantwortlich") + **„zuletzt geprüft am"** + Review-Intervall; „veraltet"-Hinweis wenn überfällig; „als geprüft markieren". |
| WV-3 | Kommentare | Diskussion am Artikel (Thread, @Mention, auflösen) — mock-first stateful. |
| WV-4 | Backlinks & Verwandtes | **„Verlinkt von"**-Panel (wer verweist via [[…]] hierher) + „verwandte Artikel" (gleiche Tags/Kategorie). |
| WV-5 | Graph + Politur | Optional **Mini-Graph** (Artikel-Vernetzung) + Demo-Tiefe-Audit + i18n. |

### Batch 4 · Integration & Lese-KI — `WI-1…WI-5` (CRM-Anschluss + dezente KI)
E = FE-mockbar (MSW). F (KI) = FE-Demo mock-first, echtes LLM = Luke/Backend.

| # | Phase | Inhalt |
|---|-------|--------|
| WI-1 | An CRM verknüpfen | Artikel mit **Kontakt/Deal/Aufgabe** verknüpfen (Typeahead, Verweis-Liste am Artikel) — wie berichte R-5, intern. |
| WI-2 | Export/Ablage | Artikel **als PDF** (window.print + Print-CSS wie berichte) / **in Dokumente ablegen**. |
| WI-3 | KI-Zusammenfassung | **„Zusammenfassen"** beim Lesen — TL;DR oben am Artikel (mock-first: vorbereitete/heuristische Kurzfassung, klar als KI gelabelt; echtes LLM später). KEIN KI beim Schreiben. |
| WI-4 | „Frag das Wiki" | Natürlichsprachige **Frage über alle Artikel** → relevante Passagen + Antwort (mock-first: Keyword-/Tag-Matching + Quell-Artikel-Verweise; semantisch später via Backend). |
| WI-5 | Abschluss | Demo-Tiefe-Audit gesamtes Modul, i18n-Vollständigkeit ×4, Settings-Komplettheit, Übergabe-Bilanz. |

---

## KI-Konzept (bewusst zurückhaltend, nur Lese-Seite)
- **Nicht** beim Erstellen (kein Mehrwert, Darien). **Ja** beim Konsumieren: (1) Artikel-TL;DR, (2) „Frag das Wiki".
- Mock-first: FE baut Panel/Button + deterministische Demo-Antwort mit Quell-Verweisen, klar als KI markiert + „Beta/Demo"-Hinweis. Echtes LLM (Claude-Anbindung) + Vektor-Index = Backend (Luke), in backend-gaps.
- Risiko-arm: keine KI-Halluzination im Schreibpfad; Lese-KI ist additiv und abschaltbar.

## Pipeline-Einordnung (Vorschlag — Darien gab Freiheit)
- **Batch 1 (Tiefe) zeitnah** — kleiner, hoher Nutzen, macht wiki ehrlich fertig. Kandidat fürs Sub-Terminal parallel zu berichte R-5/R-6.
- **Batch 2–4 gestaffelt hinter berichte R-5/R-6**, damit beide Module sauber review-reif werden.
- Reihenfolge innerhalb: 1 → 2 → 3 → 4 (Tiefe vor Premium vor Workflow vor Integration/KI). KI (WI-3/4) zuletzt, wenn Kern steht.

## Backend-Bedarf (für backend-gaps, beim Bauen pflegen)
- Tags/Pins/Owner/Status/Review-Datum als echte Felder + Endpunkte (jetzt mock-Store).
- Echter Anhang-Upload (Multipart-Blob + Download-URL).
- Share-Token stateful + interner Auflös-Endpunkt.
- Kommentare-Persistenz + @Mention-Notify.
- Backlinks-Index serverseitig (statt FE-Scan).
- KI: LLM-Anbindung (Zusammenfassung) + Vektor-/Volltext-Index für „Frag das Wiki".
