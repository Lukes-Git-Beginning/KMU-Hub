---
tags: [helpdesk, formulare, ticket-intake, csat, customization, briefing]
updated: 2026-07-26
---

# Ticket-Intake & CSAT — Konzept-Briefing (SSOT)

> **Status:** Konzept, VOR dem Bau. Erstellt Session #30 (2026-07-26) nach 2 Ist-Analyse-Agents + 2 Darien-Entscheiden. Bau in FRISCHEM Terminal: erst `git pull`, dann dieses Briefing lesen, offene Fragen (§7) mit Darien klären, dann Phasen (§5).
> **Größenordnung:** eigener Baustein wie RBAC / Customization-Editor. NICHT nebenbei.

## §0 Vision (Darien 2026-07-26, verbindlich)

Ein **anpassbares Ticket-Intake-System** für den Helpdesk mit **drei kombinierbaren Erstellungs-Wegen**, alles **im Editor konfigurierbar** (welche Felder ausgefüllt werden), auf dem **bestehenden Formular-Tool** als Motor. CSAT (Sterne-Bewertung nach Ticket-Schluss) ist **dasselbe Prinzip** — ein Formular per Link.

**Drei Kanäle (mehrere gleichzeitig aktivierbar):**
1. **Extern** — Kunden/Externe erstellen ein Ticket über eine öffentliche Seite / einen Link (wie die Umfrage-Links).
2. **Intern-Selfservice** — Mitarbeiter melden IT-Support-Tickets über ein Helpcenter / einen Ort in den Settings.
3. **Agent** — Helpdesk-Mitarbeiter legen Tickets aus Anrufen/Mails selbst an (der heutige Weg im Modul).

## §1 Darien-Entscheide (gelockt)

- **① Architektur = „Formular-Tool treibt alles":** EIN Ticket-Formular im Form-Builder ist die Feld-Definition. Alle 3 Kanäle nutzen dasselbe Formular; eine Einreichung erzeugt ein Ticket (neue Formular-Aktion), die Werte landen als Ticket-Felder/Zusatzfelder. **Ein Editor** für die Intake-Felder (nicht zwei getrennte Feld-Systeme).
- **② Start = Konzept-Briefing (dieses Dok) + frisches Terminal.** Keine „schnellen Wins" vorab — sie werden Phase 1 des Blocks.
- **③ Wiederverwendbar bauen (Darien 2026-07-26, verbindlich):** Das Intake-System ist KEINE helpdesk-spezifische Insel, sondern eine **shared Engine** — analog zu `shared/document` (Block-System für alle Dokument-/Berichte-Stellen). Muster = „editor-konfigurierbares Multi-Kanal-Formular → modul-spezifischer Datensatz". Helpdesk (Formular→Ticket) ist der ERSTE Konsument; andere Module mit Anfragen/Anträgen/Requests (HR-Anträge, Einkaufs-/Wartungsanfragen, Reklamationen, Onboarding-Requests …) sollen dieselbe Engine angepasst nutzen. **CSAT/Bewertung** = Spezialfall (rating-Formular → Bewertung an einen beliebigen Datensatz, auch außerhalb Helpdesk denkbar — z.B. nach Meeting/Projekt). → Von Anfang an in `shared/` bauen, **modul-agnostische API** (Datensatz-Typ + Feld→Ziel-Mapping als Parameter), Helpdesk nur als erste Instanz verdrahten. Beim Bau immer fragen: „gehört das in die Engine (shared) oder in die Helpdesk-Instanz?"

## §2 Ist-Stand (2 Explore-Agents, 2026-07-26)

### Formular-Tool (`modules/formulare/`) — reif, ~90 % nutzbar
- **Vollwertiger Form-Builder:** 11 Feldtypen (`api/formulare-types.ts:34–77`) — `text, textarea, email, number, select, radio, checkbox, date, file, consent (DSGVO), rating (Sterne/NPS 1–5/1–10)`. Drag-and-Drop-Anordnung, Pflicht/optional, Optionen, Validierung (`minLength/maxLength/min/max/pattern` plz/phone/iban), **Conditional Logic**, **Multi-Page**.
- **Öffentliche Share-Links** (`FormShareLink`, `formulare-types.ts:145–160`): Channels `link | email | embed(iframe) | qr`, Ablauf, Einreichungs-Limit, Token-URL `forms.zentria.tech/r/{token}`. Nur `active`-Formulare teilbar. Views/Submissions-Zähler.
- **Einreichungen:** `POST /formulare/schemas/:id/submissions` mit `answers: Record<fieldId, value>`; Status `new|read|archived`; CSV/XLSX-Export.
- **Webhooks** (`FormWebhook`) bei Einreichung → Callback (Basis für Auto-Ticket-Anlage).
- **Templates** (`isTemplate`), granulare RBAC (`formulare:schemas:*`, `:submissions:*`, `:share:manage`, `:export:run`).
- **`RatingInput`-Komponente** (`FormularePage.tsx:305–375`) fertig, inkl. `light`-Modus für externe Seiten → CSAT-Sterne sofort da.
- **Reifegrad:** sehr hoch. Page ist monolithisch (5.394 Zeilen `FormularePage.tsx`) — beim Bauen ggf. Teil-Extraktion nötig.

### Helpdesk-Intake + CSAT — Lücken
- **Agent-Neu-Dialog** (`HelpdeskPage.tsx:856–922`): sammelt subject/description/category/priority/assignee/contact/Zusatzfelder — aber `createTicketMut.mutate` sendet **nur `{subject, priority, assignee_id}`**. Description, Kategorie, Kontakt, Custom-Fields **gehen verloren**. `CreateTicketInput` (`helpdesk-types.ts:91–97`) ist zu schmal.
- **Requester-Modell:** `requester_id: string` (roh). Seeds mischen echte User-UUIDs (`IDS.users.markus`) und Freitext-Namen (`'Brigitte Schärer'`). **Kein Objekt für Externe** (Name+E-Mail+Rückkanal). `scope=own` kann Externe prinzipbedingt nie matchen.
- **Kanäle:** KEINE Infrastruktur. `contact_channel` (`custom-fields.ts:309`) ist nur ein Label-Feld. Keine öffentliche Route, kein Widget, keine Kanal→Ticket-Brücke. Das Kommunikation-Modul hat ein Kanalsystem, aber keine Helpdesk-Brücke.
- **CSAT:** `CSATWidget.tsx` (Sterne 1–5 + Kommentar) rendert **nur im Agent-Detail** bei resolved/closed. `saveCsat` schreibt in den **Legacy-Store** (`stores/helpdesk.ts:637`), nicht ans Wire-Ticket; der Adapter setzt `csatRating: undefined` beim Lesen. **`CSAT_FEATURE_ENABLED=false`** (Statistik-Kachel+Chart aus, Editor listet sie `locked`). 4 Seed-Ratings existieren im **falschen (Legacy-)Store** → im aktiven Pfad unsichtbar.
- **Custom-Fields am Ticket:** `useModuleCustomFields('helpdesk_ticket')` + `CustomFieldControl` voll funktionsfähig, editor-konfigurierbar (der Modul-Editor, gebaut Session #27–30). 3 Seed-Felder (`sla_tier`, `escalation_reason`, `contact_channel`). ABER: Werte nicht auf dem Wire (`Ticket` hat keine `custom_fields`-Map), nur Session-Overlay.

### Kern-Spannung
Es gibt **zwei getrennte Feld-Systeme**: `formulare.FormField` (11 Typen, formular-spezifisch) und `custom-fields.CustomFieldDefinition` (9 Typen, entity-weit). Keine gemeinsame Registry. Darien-Entscheid ① verlangt, sie zu verbinden: **Der Form-Builder definiert die Intake-Felder, die eingereichten Werte werden zu Ticket-(Custom-)Feldern.**

## §3 Ziel-Architektur (aus Entscheid ①)

```
                 ┌───────────────────────────────────────┐
                 │  Ticket-Intake-Formular (Form-Builder) │  ← EIN Editor für die Felder
                 │  Kern-Felder (subject/desc/priority/   │    (pro Kanal ein Formular/Template,
                 │  category/requester) + freie Zusatzfel.│     alle auf demselben Builder)
                 └───────────────────────────────────────┘
                        │              │               │
            Kanal 1 EXTERN     Kanal 2 INTERN     Kanal 3 AGENT
          (Share-Link/QR,    (Helpcenter im     (Neu-Dialog im
           kein Login)        Login-Bereich)     Helpdesk-Modul)
                        │              │               │
                        └──────► Einreichung ──────────┘
                                     │
                        Formular-Aktion „→ Helpdesk-Ticket" (NEU)
                                     │
                        Ticket (Kern-Felder + Werte als custom_fields
                                + Requester {name,email} + channel-Herkunft)
```

- **Feld-Konvention:** Ein „Ticket-Formular"-Template mit reservierten Feld-Rollen (subject, description, priority, category, requester-email/name) + beliebig vielen freien Zusatzfeldern → die freien Werte mappen auf `helpdesk_ticket`-Custom-Fields.
- **CSAT** = eigenes Formular-Template mit 1× `rating` (scale 5) + optional `textarea` + `consent`; Share-Link per E-Mail nach Ticket-Schluss; Einreichung → `csatRating`/`csatComment` am Ticket.
- **Editor-Anpassbarkeit:** Der Form-Builder IST der Editor für die Intake-Felder. Anbindung an den Modul-Customization-Editor prüfen (§7): eigene Dimension „Ticket-Formulare/Kanäle" ODER Deep-Link in den Form-Builder.

## §4 Kanäle im Detail

- **Kanal 3 (Agent) — zuerst, kleinster Sprung:** Neu-Dialog behalten, aber Felder ans Ticket durchreichen (`CreateTicketInput` erweitern) + Custom-Fields aufs Wire. Optional: Dialog aus dem Ticket-Formular-Template generieren, damit es EINE Feld-Quelle ist.
- **Kanal 2 (Intern-Selfservice):** Ein „Ticket erstellen"/Helpcenter-Einstieg im Login-Bereich (außerhalb des Helpdesk-Moduls — z.B. App-weit oder in Settings). Rendert dasselbe Formular; Requester = eingeloggter Mitarbeiter. Kein öffentlicher Link nötig.
- **Kanal 1 (Extern):** Öffentlicher Share-Link/QR/Embed (kein Login). Braucht die **öffentliche Ausfüll-Route** (Backend/Luke) + Requester-Objekt (Name/E-Mail) + Spam/Rate-Limit + DSGVO-consent.

## §5 Bau-Phasen (Vorschlag, im Terminal schärfen)

- **P0 Daten-Fundament:** `CreateTicketInput` + Wire-`Ticket.custom_fields` + Requester-Objekt (`{id?, name, email, isExternal}`) + `channel`-Herkunft. CSAT ans Wire-Ticket (`csatRating/csatComment`) + Seed-Ratings in den AKTIVEN Store ziehen. MSW-Handler entsprechend.
- **P1 Kanal 3 (Agent):** Neu-Dialog reicht alle Felder durch; Custom-Fields persistieren (Overlay→MSW).
- **P2 Formular→Ticket-Aktion:** neue `FormAction 'helpdesk_ticket'` (heute nur `email/task/crm_contact`); Feld-Rollen-Mapping; „Ticket-Formular"- + „CSAT"-Template.
- **P3 CSAT nutzbar:** rating-Formular + „nach Schluss versenden"-Trigger (FE mock, Mail = Luke) + Einreichung→csatRating + `CSAT_FEATURE_ENABLED=true` + Statistik-Kachel/Chart entsperren (`editorModules.ts` helpdesk.statWidgets `locked` raus).
- **P4 Kanal 2 (Intern-Selfservice):** Helpcenter-Einstieg im Login-Bereich.
- **P5 Kanal 1 (Extern):** öffentliche Route (Luke) + Requester-Objekt + Anti-Spam; FE-Seite/Embed.
- **P6 Editor-Anbindung:** Kanäle + Formular-Zuordnung im Customization-Editor sichtbar/anpassbar (§7-Entscheid).

## §6 Backend-Bedarf für Luke (in `backend-gaps.md` übernehmen)

- 🔴 **Öffentliche Formular-Route** (`/r/:token`, kein Login) — der Haupt-Blocker für Kanal 1 (schon als „Luke's lane" im formulare-Mock markiert).
- 🔴 **`CreateTicketInput` erweitern:** `description, category, requester{name,email,isExternal}, channel, custom_fields`.
- 🔴 **Requester-Modell:** Externe (Nicht-User) als Requester speicherbar (Name/E-Mail), scope-Filter-Verhalten definieren.
- 🔴 **`Ticket.custom_fields`** aufs Wire + persistieren (bislang Session-Overlay).
- 🔴 **Formular→Ticket-Aktion:** Einreichung mit Rolle=Ticket erzeugt serverseitig ein Helpdesk-Ticket (Webhook-Basis vorhanden).
- 🔴 **CSAT-Erhebung** (bereits vorgemerkt §helpdesk-CSAT): Close-Trigger → Survey-Mail mit tokenisiertem rating-Link → Response-Endpoint → `csatRating` am Ticket + Aggregation im `/helpdesk/stats`.
- 🟠 Anti-Spam/Rate-Limit + Datei-Upload (echtes Storage statt String) für externe Formulare.

## §7 Offene Fragen fürs Terminal-Gate (mit Darien klären, VOR Bau)

1. **Feld-Quelle vereinheitlichen — wie tief?** Soll der Agent-Dialog aus DEMSELBEN Ticket-Formular-Template generiert werden (eine Feld-Quelle, sauberste Umsetzung von Entscheid ①), oder bleibt der Agent-Dialog handgebaut und nur die Werte-Struktur ist geteilt? (Empfehlung: aus dem Template generieren, aber P1 pragmatisch mit Durchreichung starten.)
2. **Editor-Ort:** Erscheint die Kanal-/Formular-Konfiguration im **Customization-Modul-Editor** (neue Dimension „Ticket-Formulare") oder direkt im **Form-Builder** mit einer Helpdesk-Verknüpfung? Darien-Vision = „im Editor anpassbar" — welcher Editor?
3. **Ein Formular pro Kanal oder ein Formular für alle?** Darf extern ein anderes (schlankeres) Formular haben als der Agent-Dialog, oder immer identische Felder?
4. **Requester-Identität extern:** reicht Name+E-Mail, oder Verknüpfung zu CRM-Kontakten (Externe = CRM-Kontakt)?
5. **Kanal 2 Ort:** Helpcenter als eigener App-Bereich, in Settings, oder als globaler „Ticket melden"-Button?
6. **CSAT-Auslöser:** automatisch bei Close (Delay) — konfigurierbar pro Tenant im Helpdesk-Setting? Skala Sterne 1–5 fix?

## §8 Bezug zu bereits Gebautem (nicht neu bauen)
- Modul-Customization-Editor inkl. Zusatzfelder-Dimension (`useModuleCustomFields`, `FelderPanel`) + Statistik-Dimension (Session #30) — die CSAT-Kachel ist dort schon als `locked` vorbereitet.
- `CSATWidget`/`CSATAggregate` (Sterne + Aggregation) — wiederverwenden, nur an echte Daten + Requester-Link hängen.
- Ticket-Beschreibungen jetzt in Seeds + Adapter (Session #30) — Basis für P0.
