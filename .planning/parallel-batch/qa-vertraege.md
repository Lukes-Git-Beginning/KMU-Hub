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
