# wiki Batch 2 (Editor auf berichte-Niveau) — Sub-Terminal-Paket

> Copy-paste-Block unten ins KMU-Hub-review-Terminal (Port 5174). Voller Plan: `.planning/wiki-knowledge-base-VISION.md` (Batch 2 = WP-1…WP-5, ⭐ priorisiert nach Dariens Review).
> Kontext: Darien-Review 2026-06-21 — „das Erstellen sieht kein bisschen so aus wie beim berichte-Modul". Dieser Batch hebt den wiki-Editor gründlich auf berichte-Niveau (wiki-angemessen, KEIN A4-Report).

```
Du bist das Sub-Terminal im Sub-Terminal-5-Phasen-Modus. Arbeitsverzeichnis = dieser KMU-Hub-review-Klon (Dev-Server Port 5174). Das Hauptterminal baut parallel berichte R-6 (Datenquellen) — du fasst NUR wiki-Dateien + zugehörige i18n/shared-Editor an. Sprache: Deutsch (Umlaute, Eszett, Akzente — NIE ASCII-Ersatz).

SCHRITT 0 — aktuellen Stand holen:  git pull --rebase origin main

KONTEXT — wiki ist nach Batch 1 (WT-1…WT-5) funktional solide (rekursive Kategorien, Tags/Titel-Rename, Versions-Diff, Anhänge/Share-Verwaltung, Suche). ABER das ERSTELLEN/der Editor ist ein Standard-TipTap-Kasten und weit unter dem berichte-Authoring. Dieser Batch hebt das auf berichte-Niveau — wiki-angemessen (Notion-artige Seite, KEIN A4-Report mit Charts/KPIs).
Editor-Dateien: modules/wiki/WikiRichEditor.tsx (nutzt shared EditorToolbar + EditorBubbleMenu), WikiEditor.tsx (Wrapper mit Save/Cancel-Footer, max-w-3xl), WikiArticle.tsx (Lese-Modus + Edit-Umschaltung), WikiArticleHeader.tsx. Shared-Editor: components/shared/RichTextEditor/.
BERICHTE-REFERENZ (Niveau, nicht kopieren): modules/berichte/components/documents/BlockEditor.tsx (Block-/+-System, 21 Block-Typen), DocumentReader.tsx (Editorial-Lese-Ansicht, report-serif/Playfair, 65ch), report-print.css. Playfair self-hosted ist schon im Projekt (report-serif).

LÜCKE (das fehlt konkret beim wiki-Erstellen): Slash-/Block-Einfügemenü statt nur Toolbar · reiche Blöcke (Callout/Code-Syntax/Toggle/Trenner/Bild+Caption) · Cover/Icon-Identität · editoriale Premium-Typografie (Playfair-Überschriften, 65ch, ruhige Hierarchie) · Inhaltsverzeichnis · fokussierter „leerer-Canvas"-Editor statt umrahmter Kasten · Titel als großes Editorial-Heading statt Input-Feld · durchdachter Empty-State (Joy).

DEIN BATCH — 5 Phasen, je ein Commit:
- WP-1 Editor-Shell + Look: WikiEditor/WikiRichEditor vom umrahmten Kasten auf fokussierten, ruhigen Canvas. Großzügige Lesebreite, Premium-Typografie wie berichte (Playfair-Überschriften via report-serif self-hosted, 65ch-Fließtext, klare Hierarchie). Titel als großes Editorial-Heading (kein Input-Kasten). Dezente, sticky Toolbar (nicht der volle Kasten-Rahmen). Read-/Edit-Look sauber abgegrenzt.
- WP-2 Slash-Menü + reiche Blöcke: /-Einfügemenü (Icon+Label, tastatur-navigierbar — Muster aus berichte BlockEditor) + neue TipTap-Blöcke: Callout (Info/Warnung/Tipp/Empfehlung, Muster aus berichte CALLOUT_STYLE), Code mit Syntax-Highlight (lowlight), Toggle/Details (aufklappbar), Trenner, Bild mit Caption. Im Read-Modus sauber gerendert + DOMPurify-safe.
- WP-3 Cover + Icon (Identität): Cover-Bild + Icon/Initial pro Artikel (Custom-SVG, KEINE Emojis). Feld in wiki-types + Adapter + MSW ergänzen. Anzeige im Editor-Kopf, Lese-Kopf (WikiArticleHeader) und in der Artikelliste.
- WP-4 Editorial-Lese-Ansicht + TOC: Lese-Modus mit ruhiger Premium-Typografie + Breadcrumbs (Bereich › Kategorie › Artikel) + Lesezeit + Meta-Leiste + Inhaltsverzeichnis (TOC aus H1–H3, Anker-Sprung, Scroll-Spy für aktiven Abschnitt, sticky bei langen Artikeln).
- WP-5 Vorlagen-CRUD + Politur: Eigene Vorlagen (erstellen/bearbeiten/löschen + Vorschau) statt der 3 hardcoded in stores/wiki.ts → MSW-Endpunkte + Hook. Editor-Empty-State als Joy-Moment (Custom-SVG/Wording). Demo-Tiefe-Audit + i18n ×4.

BUILD-+-VERIFY-STANDARD PRO PHASE (verbindlich):
bauen → i18n ×4 (i18n/messages/{de,en,fr,it}.json; {var} NICHT {{var}}; Plural als ICU {count, plural, one {…} other {…}}) → gescopter Typecheck (desktop/tsconfig.wikiBatch.json vom letzten Batch — neue Dateien aufnehmen; desktop/node_modules/.bin/tsc -p tsconfig.wikiBatch.json --noEmit, foreground, echter Exit, NIE | tail) → Playwright-Screenshot-QA gegen http://localhost:5174 (Muster scripts/qa-wiki-*.mjs; Hash-Route /#/wiki) → die PNGs WIRKLICH ansehen (Look mit berichte vergleichen! Raw-Keys/Emojis/Theme/Layout/leere Zustände, mehrere Datensätze + Breiten) → iterieren bis grün → ein Commit.

PROJEKTWEITE STANDARDS (sonst Review-Rückläufer):
Detail = shared/DetailModal. Sticky Back/Close. Leere Zustände = EmptyState mit Aktion. Keine Toast-Stubs/toten Endpoints — echte MSW-Funktion. KEINE Emojis (Personality via Custom-SVG/Motion/Wording). Theme-Tokens (bg-info-light, text-success …). Motion nur transform/opacity. Skeleton statt Spinner. CURRENT_USER aus mocks/data/shared-ids. Neue Dateien unter mocks/data/ brauchen git add -f.

GIT:
Conventional Commits, English imperative, KEINE AI-Attribution. Commit pro Phase mit EXPLIZITEN Datei-Pfaden — NIE git add -A/. Nach jedem Commit: git push origin main; bei Ablehnung git pull --rebase origin main (i18n-Konflikte: beide Key-Bereiche behalten), dann erneut push. Dev-Server (5174) NICHT killen.

ABSCHLUSS: kurze Bilanz — welche Phasen committed (Hashes), QA-Ergebnis je Phase, was offen blieb.
```
