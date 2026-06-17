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

---

## V-3 — Fristen-Notifications verkabeln

**Architektur-Befund (wichtig):** Die App hat **zwei** Notification-Systeme. Die sichtbaren Oberflächen (Topbar-Bell, `/notifications`-Center, Dashboard-`NotificationSummary`) lesen via `useNotifications()` aus **MSW** (`mocks/handlers/notifications.ts`). Der zustand-`useNotificationsStore` speist nur den transienten **Toast** (`NotificationToast`); `isDropdownOpen`/`toggleDropdown` sind ungenutzt. → Reminder müssen in die **MSW-Quelle**, sonst erscheinen sie nur als Toast.

**Gebaut (zwei konsistente Kanäle):**
- **MSW (persistent):** 3 `contract_expiry`-Notifications für die auslaufenden Verträge (v-3 Microsoft/18 T, v-5 Müller/47 T, v-11 Lagerraum/82 T) → erscheinen in Bell + Center + Dashboard-Summary. GET-Handler sortiert jetzt nach `created_at` (neueste zuerst). Idempotent durch deklarative Seeds (kein Duplikat bei Reload).
- **zustand (Live-Toast):** idempotente `ensureNotification(id)`-Methode + Mount-Hook `useContractExpiryNotifications` (`useContractReminders.ts`) — berechnet je Vertrag genau **eine** Notification (engste erreichte reminderDays-Schwelle, stabile ID `contract-expiry-<id>-<threshold>`), feuert als Toast beim Öffnen des Moduls. Kein Duplikat bei Re-Render/Reload (verifiziert über persistierten Store-State).
- **Seed-Daten aufgefrischt** (behebt V-2-Nebenbefund): v-3/v-5/v-11 laufen jetzt relativ zu heute aus (`isoDaysFromNow`, lokale Datums-Komponenten gegen UTC-Off-by-one) → „Auslaufend"-Tab gefüllt (3), Dashboard-AlertsSection (liest `useVertraegeStore`) zeigt sie ebenfalls.

**Mitgenommen (Bug-Fix, vorbestehend):** `notifications.center.unread` war als i18next-`_one`/`_other` definiert, das Projekt nutzt aber **ICU-Plural** → Raw-Key „notifications.center.unread" im Center-Subtitle. Auf ICU `{count, plural, …}` umgestellt (×4) → zeigt jetzt „7 ungelesene Benachrichtigungen".

**Schlüsseldateien:** `mocks/handlers/notifications.ts` (3 Seeds + Sort), `stores/notifications.ts` (`ensureNotification`), `modules/vertraege/useContractReminders.ts` (neu), `VertraegePage.tsx` (Hook-Mount), `stores/vertraege.ts` (`isoDaysFromNow` + v-3/v-5/v-11), i18n ×4. QA: `scripts/qa-vertraege-reminders.mjs`

**Was Darien anschauen soll:**
- Bell/`/notifications` öffnen → „Vertrag läuft bald ab — Microsoft 365 Business … Ablauf in 18 Tagen" (+ Müller 47 T, Lagerraum 82 T), Unread-Badge erhöht, Subtitle „N ungelesene Benachrichtigungen" (kein Raw-Key).
- `/vertraege` öffnen → kurzer Toast unten rechts (Live-Reminder). Tab „Auslaufend" → 3 Verträge. Mehrmals neu laden → keine Duplikate.

**Screenshots:** `desktop/.qa-screenshots/vertraege-reminders/a0-center-direct.png`, `…/c-auslaufend-tab.png`, `…/b-after-reload.png`
**QA-Status:** ALL PASS (Center zeigt 3 Reminder direkt aus MSW; zustand-Store exakt 3 stabile IDs, identisch nach Reload; Auslaufend-Tab 3 Zeilen; ICU-Plural gerendert; 0 Raw-Keys; 0 Console-Errors).

---

## V-4 — E-Signatur „Senden": realistischer Demo-Flow

**Gebaut:** „Zur Unterschrift senden" versendet jetzt nicht mehr nur (status `sent`) und schließt — der Dialog bleibt offen und bietet pro versendetem Unterzeichner „Rücklauf simulieren". Der simulierte Rücklauf läuft `sent → viewed → signed` über einen kurzen Timer (1,5 s) und schreibt echte Audit- + Timeline-Events. Ein **dezenter Demo-Hinweis** („Demo: Der Rücklauf wird simuliert — es wird keine echte E-Mail versendet.") macht klar, dass kein echter Mailversand passiert. Skribble-Banner unverändert.

**Details:**
- Store: `dispatchSigners(contractId, signers)` (sync + pending/viewed→sent + ein `contract_sent`-Audit), `advanceSignerReturn(contractId, idx)` (State-Machine sent→viewed→signed, je Schritt ein Audit-Event; Rücklauf-Events tragen den **Unterzeichner** als Akteur, nicht den aktuellen Nutzer).
- Neue Audit-Action-Codes `contract_sent` (Icon Send) + `contract_viewed` (Icon Eye) inkl. Farben + i18n ×4 → Audit-Log zeigt „Zur Unterschrift versendet" / „Vertrag geöffnet" / „Vertrag unterzeichnet".
- Vor Versand: Vor-Ort-Canvas-Signatur („Jetzt unterschreiben", nur `pending`). Nach Versand: „Rücklauf simulieren" (`sent`/`viewed`), mit Spinner während der Simulation.

**Schlüsseldateien:** `modules/vertraege/ESignaturDialog.tsx`, `stores/vertraege.ts` (2 Methoden), `VertraegePage.tsx` (Action-Codes/Icons), i18n ×4. QA: `scripts/qa-vertraege-esign-demo.mjs`

**Was Darien anschauen soll:**
- `/vertraege` → Vertrag mit Unterzeichnern öffnen (z. B. „Müller Metallbau Rahmenvertrag") → Footer „Unterschrift" → Dialog.
- „Zur Unterschrift senden" → offener Signer wird „Gesendet", **Demo-Hinweis** erscheint, Dialog bleibt offen.
- „Rücklauf simulieren" → Signer läuft Angesehen → Unterschrieben, Statusverlauf + Audit-Log (Detail-Modal) füllen sich. Ehrliche Kennzeichnung (Toast „… (Demo-Rücklauf)").
- Skribble-Banner weiterhin „Bald verfügbar" (unverändert).

**Bekannt → wird in V-5 behoben:** Das `contract_sent`-Audit zeigt aktuell noch „Aktueller Benutzer" (Versand = Aktion des eingeloggten Nutzers) — wird in V-5 auf den echten Auth-User umgestellt.

**Screenshots:** `desktop/.qa-screenshots/vertraege-esign/a-after-send.png`, `…/b-after-simulate.png`, `…/c-audit-log.png`
**QA-Status:** ALL PASS (Senden → Gesendet + Demo-Hinweis + Simulieren-Button; Rücklauf → Unterschrieben + Timeline; Audit-Log zeigt versendet/geöffnet/unterzeichnet; 0 Raw-Keys; 0 Console-Errors).
