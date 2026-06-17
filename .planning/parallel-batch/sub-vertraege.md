# Sub-Terminal — vertraege Tiefe-Pass (V-1 … V-5)

> **Du bist das Sub-Terminal im Klon `…/KMU-Hub-review`, Dev-Port 5174.** Lies zuerst `.planning/parallel-batch/README.md` (Lane-Regeln, Build-+-Verify-Standard, Gates). Du baust **nur vertraege**. dashboard gehört dem Main-Terminal — fass es nicht an.
>
> **Selbst-enthaltend:** Alle Klärungsfragen sind beantwortet (siehe README „Entscheidungen"). Bau die 5 Punkte ohne Rückfragen ab. Melde Darien nach jedem Punkt „V-x fertig, n/5".

## Ausgangslage (Ist-Abgleich 2026-06-17)
vertraege ist **~75 % fertig**, store-only auf `useVertraegeStore` (Zustand + persist, key `cosmi-verträge`). `VertraegePage.tsx` ist ~2400 Zeilen und enthält Dialog, Detail, Audit-Log, Reminder, Doc-Upload, E-Signatur, CRM-Links, KI-Fristencheck. Der vorbereitete API-Client `api/vertraege-client.ts` + `api/vertraege-types.ts` ist sauber, aber **ungenutzt** (bleibt mock-first, Backend = Luke später).

Stores: `stores/vertraege.ts` (CRUD + History/Audit, `MOCK_CONTRACTS` 12 Stück, `MOCK_TEMPLATES` 6), `stores/vertraegePrefs.ts`, `stores/vertraegeSettings.ts`.
i18n: alles unter `vertraege.*` in `i18n/messages/de.json` (+ en/fr/it).

## Workflow pro Punkt
bauen → i18n ×4 (`{var}`, ICU-Plural) → MSW/Demo-Daten falls nötig → Compile-Gate (`npm run build`, da scoped tsc über den Detail-Graphen crasht) → Playwright-Screenshot-QA gegen **:5174** + **Bilder ansehen** → iterieren → `git pull --rebase` → commit + push → Eintrag in `qa-vertraege.md`.

---

### V-1 — Slide-over DetailPanel → zentriertes DetailModal  `[PATTERN]`
**Ist:** `VertraegePage.tsx` ~L1559–1674: `DetailPanel` ist ein **Slide-in-Panel**. Klick auf Tabellenzeile setzt `selectedContractId`, `DetailPanelContent` rendert alle Sektionen.
**Soll:** Auf `shared/DetailModal` umbauen (zentriertes Cosmi-Modal-Fenster, wie work/kontakte/dokumente). Referenz: work W-1 (Commit `999260ea`), und `shared/DetailModal`-Nutzung in `modules/kontakte` + `modules/dokumente`.
- Header **sticky**: Vertragstyp-Label (z. B. „Vertrag") + Vertragsnummer als Subtitle. Close-Button sticky, nie wegscrollen.
- Body scrollbar, behält **alle** Sektionen: Laufzeitbalken, Finanzen, Reminder/`ReminderSchedule`, Dokumente, Signaturen, Verknüpfungen, `DeadlineCheckPanel`, `AuditLogFeed`. Detail ist sehr reich → großes, sektioniertes Modal.
- **Ganze Tabellenzeile klickbar** (`div role="button"` + Keyboard-Support); innere Buttons (Bearbeiten, Kündigen, Drei-Punkte) `e.stopPropagation()`.
- DnD/sonstige Interaktionen nicht brechen.
**Verify:** Zeile klicken → Modal zentriert auf; alle Sektionen da; Close sticky beim Scrollen; keine Raw-Keys.

### V-2 — Dokument-Preview-404 fixen
**Ist:** Seed-Verträge in `stores/vertraege.ts` haben `ContractDocument.fileId` wie `'file-005'`, die der Dokumente-MSW **nicht** kennt → `FilePreviewModal` presign liefert 404, Vorschau leer.
**Soll:** Seed-`fileId`s auf **echte, im Dokumente-MSW vorhandene** Datei-IDs mappen (prüfe `mocks/handlers/dokumente*.ts` / die Dokumente-Seed-Daten, welche PDF-IDs existieren). Alternativ ein paar echte Vertrags-PDF-Seeds in den Dokumente-Handler legen und darauf zeigen. Mindestens 2–3 Seed-Verträge müssen ein **echtes PDF** in der Vorschau zeigen.
**Verify:** Vertrag mit Dokument öffnen → Vorschau-Button → PDF rendert im iframe (kein 404, kein leerer Viewer). Headed testen (headless hat keinen PDF-Viewer).

### V-3 — Fristen-Notifications verkabeln
**Ist:** `reminderDays: number[]` (30/60/90 Tage) am Contract; Notification-Typ `contract_expiry` existiert in `stores/notifications.ts`, wird aber **nie programmatisch erzeugt**.
**Soll:** Beim Store-Init/Mount prüfen, welche Verträge innerhalb ihrer `reminderDays` vor Ablauf bzw. Kündigungsfrist liegen → `addNotification()` pro fälligem Reminder. **Idempotent** (stabile Notification-ID je Vertrag+Schwelle, kein Duplikat bei Re-Render/Reload). Erscheint im Notification-Center und in der dashboard-`AlertsSection` (die `useVertraegeStore` für expiring contracts liest) → schöne Cross-Modul-Demo.
**Verify:** Notification-Center öffnen → Fristen-Hinweise da; mehrmals neu laden → keine Duplikate.

### V-4 — E-Signatur „Senden": realistischer Demo-Flow
**Ist:** `ESignaturDialog.tsx` ~L225–238: `handleSend()` setzt nur `status: 'sent'`, kein Versand. Canvas-EES funktioniert vollständig. Skribble = deaktiviertes „bald verfügbar"-Banner (L299–309).
**Soll (Darien: realistischer Demo-Flow):** „Zur Unterschrift senden" → `status: 'sent'`, `signedVia: 'dispatch'` + **simulierter Verlauf**: ein klar gekennzeichneter Demo-Schritt „Rücklauf simulieren" (oder kurzer Timer) der den Signer `sent → viewed → signed` durchlaufen lässt und die Timeline/Audit-Events schreibt. **Dezenter „Demo"-Hinweis**, dass der Rücklauf simuliert ist — kein echter Mailversand. Skribble-Banner bleibt unverändert.
**Verify:** Signer hinzufügen → senden → Status „gesendet" mit Demo-Hinweis → Rücklauf simulieren → „signiert" + Timeline-Eintrag. Keine Raw-Keys, ehrliche Kennzeichnung sichtbar.

### V-5 — Demo-Tiefe + Nummernkreis + Audit-Log echter User
- **Nummernkreis:** `vertraegeSettings.numberFormat` ist heute Display-only. Beim Anlegen eines neuen Vertrags die Vertragsnummer **automatisch** aus dem Format + Zähler generieren (statt leer/manuell).
- **Audit-Log:** History-Einträge schreiben heute hardcoded `'Aktueller Benutzer'` → echten Auth-User-Namen aus dem Auth-Context/Store ziehen.
- **Demo-Tiefe-Audit:** restliche vertraege-Buttons/Aktionen durchgehen (Toast-only / console.log / nichts) und ehrlich machen oder verkabeln.
**Verify:** neuen Vertrag anlegen → Nummer auto-vergeben; Änderung machen → Audit-Log zeigt echten Namen; kein toter Button mehr.

---

## Definition of Done (vertraege review-reif)
Alle 5 Punkte verifiziert (Screenshots angesehen), 0 Raw-Keys / 0 Doppelklammern / 0 Console-Errors, jede Phase ein Commit+Push (rebase davor), `qa-vertraege.md` gepflegt. Dann Darien Bescheid: „vertraege 5/5 fertig".
