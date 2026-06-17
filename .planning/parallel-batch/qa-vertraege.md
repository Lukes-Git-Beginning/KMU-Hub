# QA-Protokoll — vertraege Tiefe-Pass (Sub-Terminal, :5174)

> Pro Phase ein Eintrag: was gebaut, Schlüsseldatei(en), **was Darien anschauen soll**, Screenshot-Pfad.
> `[PATTERN]` = zuerst anschauen (betrifft mehrere Phasen / Pattern-Entscheidung).

---

## V-1 — Slide-over DetailPanel → zentriertes DetailModal  `[PATTERN]`

**Gebaut:** Das Vertrag-Detail öffnet jetzt als zentriertes Cosmi-`DetailModal` (wie finanzen/kontakte/dokumente) statt als Slide-over-Panel von rechts. Header sticky (Titel = Vertragstitel, Subtitle = Vertragstyp-Label · Vertragsnummer, Status-Badge), Footer sticky (Bearbeiten / Unterschrift / Kündigen), Body scrollt intern. Alle Sektionen erhalten: Vertragsdetails, Laufzeitbalken, Wert, Konditionen, Erinnerungen, Notizen, Dokumente, Unterschriften, Verknüpfungen, KI-Fristencheck, Änderungshistorie. Tabellenzeile ist jetzt auch **per Tastatur** bedienbar (Tab fokussiert die Zeile, Enter/Space öffnet das Modal; innere Aktions-Buttons fangen ihre Events ab).

**Schlüsseldateien:**
- `desktop/src/renderer/src/modules/vertraege/VertraegePage.tsx` (`DetailPanel`→`DetailModal`, `maxWidth="max-w-3xl"`, Subtitle = Typ-Label · Nr; Tabellenzeile `tabIndex`/`onKeyDown`/`aria-label`/Focus-Ring)
- `i18n/messages/{de,en,fr,it}.json` (+1 Key `vertraege.table.openDetail`)
- QA: `desktop/scripts/qa-vertraege-modal.mjs`

**Was Darien anschauen soll:**
- Screen: `/#/vertraege`, Tab „Aktiv". Auf eine **Tabellenzeile** klicken (z. B. „Büro-Mietvertrag München").
- Erwartung: Modal öffnet **zentriert** (nicht mehr rechts reingeschoben). Beim **Scrollen im Body** bleiben Header (Close-Button) und Footer stehen. Drei-Punkte-Menü / Bearbeiten in der Zeile öffnet NICHT das Detail (stopPropagation ok).
- Tastatur: mit Tab eine Zeile fokussieren (Focus-Ring sichtbar), Enter → Modal öffnet.
- Worauf achten: keine Raw-Keys, kein abgeschnittener Header, Status-Badge korrekt, Subtitle „Mietvertrag · MV-2024-001".

**Screenshots:** `desktop/.qa-screenshots/vertraege-modal/a-modal-open.png`, `…/c-scrolled-sticky-header.png`, `…/d-keyboard-open.png`
**QA-Status:** ALL PASS (zentriert x=336/w=768 in 1440, 9/9 Sektionen, Sticky-Header verifiziert, Keyboard-Open, 0 Raw-Keys, 0 Console-Errors).

---

## V-2 — Dokument-Preview-404 fixen

**Befund:** Der beschriebene 404 besteht **nicht mehr** — die Seed-`fileId`s der Verträge (`file-005/006/007/021`) existieren bereits 1:1 im Dokumente-MSW (`mocks/handlers/documents.ts`) mit passenden Dateinamen; `/documents/files/:id/download` liefert eine echte Blob-PDF-URL. Headed verifiziert: v-1 + v-2 rendern echte PDFs im iframe (kein leerer Viewer).

**Gebaut (Demo-Verbreiterung):** Zwei weiteren **aktiven** Verträgen ein echtes, im Dokumente-MSW vorhandenes PDF angehängt, damit mehr Verträge eine funktionierende Vorschau zeigen:
- v-7 „Thomas Berger Arbeitsvertrag" → `file-014` Arbeitsvertrag_Muster.pdf (semantisch perfekt)
- v-4 „Allianz Betriebsversicherung" → `file-018` Datenschutzerklaerung.pdf
→ Jetzt **4 erreichbare aktive Verträge** mit echter PDF-Vorschau (v-1, v-2, v-7, v-4).

**Nebenbefund (für Darien / V-5):** Mehrere Seed-Verträge haben **vergangene `endDate`s** (relativ zu heute 2026-06-17): v-3 Microsoft 365 + v-11 Lagerraum stehen auf Status `expiring`, fallen aber in **keinen Tab** (Aktiv/Auslaufend/Archiv) → in der Demo unsichtbar. Der „Auslaufend"-Tab ist aktuell **leer** (kein Vertrag 0–90 Tage vor Ablauf). Das betrifft auch V-3 (Reminder feuern nur bei nahem Ablauf) — ich frische dafür in V-3 ein paar Daten auf.

**Schlüsseldateien:** `desktop/src/renderer/src/stores/vertraege.ts` (documents auf v-4 + v-7); QA: `desktop/scripts/qa-vertraege-preview.mjs` (**headed**)

**Was Darien anschauen soll:**
- Screen: `/#/vertraege`, Tab „Aktiv". Vertrag mit Dokument öffnen (z. B. „Büro-Mietvertrag München" oder „Thomas Berger Arbeitsvertrag") → im Modal zur Sektion **Dokumente** scrollen → auf den **Dateinamen** klicken.
- Erwartung: FilePreviewModal öffnet, **PDF rendert im iframe** (kein 404, kein leerer Viewer). In Electron/headed sichtbar; headless Chromium hat keinen PDF-Viewer.

**Screenshots:** `desktop/.qa-screenshots/vertraege-preview/Vertrag_Gruber_Maschinenbau.png`, `…/Arbeitsvertrag_Muster.png`, `…/SLA_Helvetia_Software.png`, `…/Datenschutzerklaerung.png`
**QA-Status:** ALL PASS (4/4 Verträge, iframe blob:-URL, PDF gerendert, 0 Console-Errors).
