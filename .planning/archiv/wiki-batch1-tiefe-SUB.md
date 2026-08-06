# wiki Batch 1 (Tiefe & Korrektheit) — Sub-Terminal-Paket

> Copy-paste-Block unten ins KMU-Hub-review-Terminal (Port 5174). Haupt-Terminal baut parallel berichte R-5.
> Voller Konzept-Plan: `.planning/wiki-knowledge-base-VISION.md` (Batch 1 = WT-1…WT-5).

```
Du bist das Sub-Terminal im Sub-Terminal-5-Phasen-Modus. Arbeitsverzeichnis = dieser KMU-Hub-review-Klon (Dev-Server Port 5174). Das Hauptterminal baut parallel berichte R-5 — du fasst NUR wiki-Dateien + zugehörige MSW/i18n an. Sprache: Deutsch (Umlaute, Eszett, Akzente — NIE ASCII-Ersatz).

SCHRITT 0 — aktuellen Stand holen:  git pull --rebase origin main

KONTEXT — wiki ist nach W-1…W-5 ein solides Gerüst, aber mehrere Dinge sind Stub oder kaputt. Du schließt in diesem Batch die Tiefe-Lücken, damit das Modul „ehrlich fertig" wirkt. Voller Plan: .planning/wiki-knowledge-base-VISION.md.
Modul: desktop/src/renderer/src/modules/wiki/ · Typen: api/wiki-types.ts · Adapter: api/wiki-adapter.ts · Hooks: api/hooks/useWiki.ts · MSW: mocks/handlers/wiki.ts · Stores: stores/wiki.ts (UI + articleMeta für Tags/Pins), stores/wikiPrefs.ts, stores/wikiSettings.ts.

BEKANNTE LÜCKEN (frisch gescoutet — fixe genau diese):
- Unterkategorien erscheinen NIE: WikiSidebar.tsx rendert Kategorien flach (kein parent-Filter); WikiTreeNode.tsx hat keine Rekursion/kein children-Slot. parent_id existiert im Typ + Demo (wcat-004 ist Kind von wcat-001).
- Tags nur lesbar (WikiArticleHeader.tsx), liegen mock-first in useWikiStore.articleMeta. Titel im bestehenden Artikel nicht umbenennbar (updateArticle unterstützt title im Payload).
- Versionen ohne Substanz: wiki-adapter.ts setzt changeNote:'' hart (~Z.127) und lastEditedBy leer (~Z.111); kein Diff/Preview alter Stände.
- Anhänge: WikiAttachments.tsx hat keinen Download/keine Bild-Vorschau (file_ref = nur Dateiname), kein echter Blob.
- Freigabe: WikiShareDialog.tsx nutzt cosmi://wiki/share/{token}; KEIN Token-Listing, revokeShareToken existiert im Hook aber nirgends in der UI; MSW /share ist NICHT stateful (POST gibt frischen Token, DELETE gibt nur 204).
- Suche defekt im Listen-Endpunkt: mocks/handlers/wiki.ts GET /articles?search= prüft typeof a.content==='string' (~Z.381-382) → trifft TipTap-JSON nie, nur Titel. (Der /search-Endpunkt ist korrekt via contentText().)
- Tote i18n-Keys: wiki.article.you, wiki.actions.pinNotAvailable. wiki.header.views ohne ICU-Plural.

DEIN BATCH — 5 Phasen, je ein Commit:
- WT-1 Kategorie-Hierarchie + Verschieben: WikiTreeNode.tsx rekursiv machen (Kinder via parent_id, Expand/Collapse rendert children, Einrückung pro Ebene); WikiSidebar.tsx nur Top-Level (parent_id null) als Wurzeln. PLUS „Artikel verschieben" (Kategorie wechseln) — Auswahl im Artikel-Kontextmenü/Detail (ItemActions in WikiArticleHeader) → PUT category_id via useUpdateArticle.
- WT-2 Tags editierbar + Filter + Titel-Rename: Tags im Artikel hinzufügen/entfernen (Chip-Input mit Vorschlägen aus allen vorhandenen Tags), Tag-Filter in der Artikelliste (klick auf Tag filtert). Tags in MSW als echtes Feld am Article ergänzen (statt nur articleMeta-Store) + Adapter/Typen anpassen. Titel inline umbenennbar im WikiArticleHeader (Edit-Feld, onBlur PUT title).
- WT-3 Versionen mit Substanz: optionales „Was wurde geändert?"-Feld beim Speichern (WikiEditor-Footer) → in MSW appendVersion als change_note speichern + Adapter durchreichen (changeNote befüllen). Diff/Preview: zwei Stände vergleichen (added/removed, simple Text-Zeilen-Diff aus HTML→Text) im WikiVersionHistory/WikiVersionItem, BEVOR man Restore klickt.
- WT-4 Anhänge + Freigabe: WikiAttachments.tsx — Download-Button (Data-URL/Demo-Blob) + Bild-Vorschau (thumbnail bei mime image/*). WikiShareDialog.tsx — Liste aktiver Tokens + Widerruf-Button (useRevokeShareToken verdrahten); MSW SHARE_TOKENS als stateful Array (POST push, GET list, DELETE splice); interner Team-Verweis statt cosmi:// (z.B. Link der den Artikel in-App öffnet).
- WT-5 Suche-Fix + Politur: GET /articles?search= in mocks/handlers/wiki.ts auf echten Content-Walk (contentText()) fixen + Treffer-Highlight in der Ergebnisliste. Tote i18n-Keys entfernen (wiki.article.you, wiki.actions.pinNotAvailable), wiki.header.views auf ICU-Plural ({count, plural, one {…} other {…}}). Demo-Tiefe-Audit: alle Buttons echt, keine toten Endpunkte.

BUILD-+-VERIFY-STANDARD PRO PHASE (verbindlich):
bauen → i18n ×4 (i18n/messages/{de,en,fr,it}.json; {var} NICHT {{var}}; Plural als ICU {count, plural, one {…} other {…}}) → gescopter Typecheck (desktop/tsconfig.wikiBatch.json existiert vom letzten Batch — neue Dateien aufnehmen; desktop/node_modules/.bin/tsc -p tsconfig.wikiBatch.json --noEmit, foreground, echter Exit, NIE | tail) → Playwright-Screenshot-QA gegen http://localhost:5174 (Muster scripts/qa-wiki-*.mjs; Hash-Route /#/wiki) → die PNGs WIRKLICH ansehen (Raw-Keys/Emojis/Theme/Layout/leere Zustände, mehrere Datensätze + Breiten) → iterieren bis grün → ein Commit.

PROJEKTWEITE STANDARDS (sonst Review-Rückläufer):
Detail = shared/DetailModal (zentriertes Fenster, NICHT Slide-over); ganze Zeile klickbar (innere Buttons stopPropagation). Sortierung Feld+Richtung via shared/SortMenu. Sticky Back/Close. Leere Zustände = EmptyState mit Aktion. Keine Toast-Stubs/toten Endpoints — echte MSW-Funktion. KEINE Emojis. Theme-Tokens (bg-info-light, text-success …). Skeleton statt Spinner. CURRENT_USER aus mocks/data/shared-ids. Neue Dateien unter mocks/data/ brauchen git add -f.

GIT:
Conventional Commits, English imperative, KEINE AI-Attribution. Commit pro Phase mit EXPLIZITEN Datei-Pfaden (git add desktop/.../wiki/... mocks/handlers/wiki.ts i18n/messages/*.json tsconfig.wikiBatch.json) — NIE git add -A/. Nach jedem Commit: git push origin main; bei Ablehnung git pull --rebase origin main (i18n-Konflikte: beide Key-Bereiche behalten), dann erneut push. Dev-Server (5174) NICHT killen.

ABSCHLUSS: kurze Bilanz — welche Phasen committed (Hashes), QA-Ergebnis je Phase, was offen blieb.
```
