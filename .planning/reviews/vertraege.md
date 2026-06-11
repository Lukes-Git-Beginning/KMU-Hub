# Review-Fäden — vertraege

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `vertraege` · **Strom:** L · **Reviewer (zugeteilt):** offen

---

## Phase 1 — Modul-Einstellungen (VertraegeSettingsPanel)  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route(n): `/vertraege` → Sidebar unten „Modul-Einstellungen" → Eintrag „Verträge" (kontext-vorausgewählt)
- Schritte: Persönlich: Standard-Tab umstellen (z.B. Vorlagen) → Overlay schließen → `/vertraege` neu laden → Modul öffnet auf dem gewählten Tab. Tabellendichte „Kompakt" → Vertragsliste wird enger. Für alle: Vertragsart deaktivieren → „Vertrag anlegen" bietet sie nicht mehr an; Kündigungsfrist/Verlängerung ändern → Dialog vorbelegt.

**Worauf achten (Feinschliff):**
- [ ] Layout/Hierarchie bei voller Breite + schmal (760 geprüft, Screenshot)
- [ ] Keine Raw-i18n-Keys, keine Emojis, keine ASCII-Umlaute (QA: 0 Raw-Keys, alle 4 Sprachen befüllt)
- [ ] Interaktionen echt: Standard-Tab + Dichte + Erinnerungs-Vorauswahl greifen real; Vertragsarten-Toggle + Standardwerte steuern den Neu-Dialog real
- [ ] Erinnerungs-Resolve: persönliche Vorauswahl überschreibt Unternehmens-Standard (Checkbox „Standard des Unternehmens verwenden")
- [ ] Lock-Verhalten „Für alle" für Nicht-Modulleiter

**Screenshots:** `desktop/.qa-screenshots/vertraege-settings/` (panel-top, panel-tenant, panel-pref-set, module-default-tab, panel-760) — QA `desktop/scripts/qa-vertraege-settings.mjs`

**Bekannte offene Punkte / Backend-Bedarf:**
- Tenant-Settings laufen mock-first (`stores/vertraegeSettings.ts`, localStorage) — Persistenz via `tenant_settings` (module_id='vertraege') = Luke-Backend; Settings-Fundament-Endpoints existieren seit Migration 000138.
- Nummernkreis: nur Format + Vorschau; automatische Vergabe = Backend.
- Standard-Laufzeit (Monate): Setting vorhanden, greift noch nicht im Dialog (Enddatum-Autofill bewusst nicht in dieser Phase).
- **Offene Frage für Darien:** vertraege-Domain fehlt komplett in der OpenAPI-Spec (vorbestehend, betrifft nicht diese FE-Phase).
- Beobachtung (vorbestehend, global): Topbar rendert ein „⚙️"-Emoji (innerText-Dump bei 760) — kollidiert mit der No-Emoji-Regel, nicht Teil dieser Phase.

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

## Phase 4 — Audit-Log + Erinnerungs-Detail  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route: `/vertraege` → beliebigen Vertrag in der Tabelle anklicken → DetailPanel öffnet sich
- Sektion "ERINNERUNGEN": zeigt Erinnerungstermine (endDate - X Tage) mit Datum rechts; vergangene Termine sind durchgestrichen + gedimmt
- Sektion "ÄNDERUNGSHISTORIE (N)": chronologischer Feed, neueste zuerst, mit farbigen Icon-Badges je Aktionstyp

**Was gebaut:**
- `AuditLogFeed`-Komponente (inline, nicht importiert): ersetzt den einfachen Timeline-Feed durch einen Activity-Feed mit Lucide-Icons (FilePlus/FilePen/FileX/Pen/BellRing/History) und farbigen Icon-Kreisen; neueste Einträge fett; Leer-Zustand mit Rahmen + zentriertem History-Icon
- `ReminderSchedule`-Komponente (inline): berechnet endDate - X Tage für jeden reminderDays-Eintrag, zeigt Datum rechts, vergangene Termine gedimmt + line-through; Leer-Zustand (keine reminderDays) mit Bell-Icon + Text; Unbefristet-Fallback wenn kein endDate
- **Store-Codierung:** Neue Store-Mutationen schreiben stabile englische Action-Codes (`contract_created`, `contract_updated`, `contract_terminated`). Legacy Mock-Einträge enthalten vorübersetzte deutsche Freitexte — der Renderer erkennt bekannte Codes via `isActionCode()` und übersetzt via i18n; unbekannte Strings werden als Fallback direkt angezeigt
- `ContractHistoryActionCode`-Union-Type im Store + `meta?: string`-Feld für Zusatzpayload (z.B. Kündigungsgrund)
- `addContractFromTemplate` im Store: jetzt mit Code `contract_created` + `meta: template.name`

**Was bewusst nicht gebaut:**
- Kein Reload des `selectedContract` nach Bearbeiten (vorbestehendes Design: DetailPanel zeigt Snapshot — wird erst nach erneutem Anklicken des Vertrags aktualisiert). Das ist eine bekannte UX-Lücke, nicht Phase-4-Scope.
- `contract_signed`/`reminder_triggered` als Codes vorhanden, aber noch kein UI-Trigger (eSignatur-Dialog schreibt noch kein `contract_signed` — kann in einer Folge-Phase ergänzt werden)
- Terminierungsdialog schreibt `contract_terminated` mit `meta: reason` — der Feed zeigt aktuell nur den übersetzten Code, nicht den reason. Erweiterung möglich via `entry.meta`-Appendix im Label.

**Offene Fragen für Darien:**
- vertraege-Domain fehlt weiterhin komplett in der OpenAPI-Spec (vorbestehend, betrifft nicht diese FE-Phase)
- Wann API-Swap der Page auf `useVertraege`-Hooks (statt `useVertraegeStore`)? Dann müssen AuditLogFeed + ReminderSchedule die Props aus den API-Typen beziehen
- `selectedContract` Snapshot-Problem: soll nach Bearbeiten ein `setSelectedContract(updatedContract)` aus dem Store eingefügt werden? (kleiner Fix, braucht Entscheidung)

**Screenshots:** `desktop/.qa-screenshots/vertraege-audit/` (1-detail-with-history, 2-empty-states, 3-after-mutation) — QA `desktop/scripts/qa-vertraege-audit.mjs`

**QA-Ergebnis:** rawKeys: [], pageErrors: [], alle Assertions grün (3/3 Schritte)

**tsc-Gate:** 0 neue Fehler in eigenen Dateien (vorbestehende ~98 typed-i18n-Fehler in fremden Panels unverändert)

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

## Phase 7 — Dokumente ans Dokumente-Modul gekoppelt (Upload, Vorschau, Versionen)  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route: `/vertraege` → Vertrag mit Dokumenten anklicken (z.B. "Büro-Mietvertrag München" → hat 2 Demo-Dokumente) → DetailPanel → Sektion "Dokumente"
- Klick auf Dateiname → FilePreviewModal öffnet sich (PDF als data-URL-iframe im Demo-Modus)
- Hover über Dokument-Zeile → Versions-Button erscheint → Klick → VersionHistoryPanel (Slide-over)
- "Neuer Vertrag" → Dokumente-Sektion: "Aus Dokumenten wählen" öffnet Picker mit Search; "Hochladen" öffnet File-Picker (XHR-Upload mit Progress-Anzeige)
- Vertrag ohne Dokumente → Leerer Zustand mit gestricheltem Rahmen

**Was gebaut:**

- **Store (`stores/vertraege.ts`):** `documentRef?: string` bleibt als deprecated; neues `documents?: ContractDocument[]` mit `{fileId, name, mimeType?, size?, addedAt}`. Neue Store-Actions `addDocument(contractId, doc)` + `removeDocument(contractId, fileId)` schreiben History-Einträge (`document_added`, `document_removed` mit `meta: filename`). Mock-Verträge v-1, v-2, v-3 haben 1–2 Demo-Dokumente mit fileIds aus den MSW-Demos.
- **MSW-Handler (`mocks/handlers/documents.ts`):** Neue Demo-Handler für: Download-URL (`/files/:id/download` → data-URL mit eingebettetem Mini-PDF für iframe-Vorschau), Versionen (`/files/:id/versions` → 2 Demo-Versionen), Versions-Erstellen + Revert, Upload (`/files/upload` → gibt neues File-Objekt zurück), Entity-Links-Stub (`/files/:id/links`, POST/DELETE), Update/Copy/Move/Share/Tag/Untag-Handler für Vollständigkeit; List-Endpoint enriched mit `current_version`/`is_favorite`/`storage_key` für den `DocumentFile`-Typ.
- **ContractDialog:** Dokument-`<select>` entfernt. Neue Sektion mit: (1) Liste der bereits verknüpften Dokumente als Chips mit X-Button (entfernen); (2) Button "Aus Dokumenten wählen" → Inline-Picker-Dropdown mit Suche (useDocumentFiles); (3) Button "Hochladen" → `<input type="file">` → useDocumentUpload-Hook mit Progress-Anzeige; neu geladene/gewählte Dateien erscheinen sofort als Chip. Bei "Speichern" werden `documents` (nicht `documentRef`) in den Store geschrieben.
- **DetailPanelContent:** Sektion "Dokumente (N)" ersetzt den alten `documentRef`-Button. Pro Dokument: Icon nach Typ, Name (klickbar → FilePreviewModal), Größe, Versions-Button (hover-sichtbar, aria-label). Leerer Zustand: gestrichelter Rahmen + Text. Adapter `contractDocToDocumentFile()` baut das DocumentFile-Shape aus ContractDocument für FilePreviewModal (kein Fork des Modals). FilePreviewModal + VersionHistoryPanel aus Dokumente-Modul wiederverwendet — State (`previewDoc`, `versionsDoc`) im VertraegePage-Scope.
- **History-Codes:** `document_added` + `document_removed` in `ContractHistoryActionCode` + `HISTORY_ACTION_CODES` + `getHistoryIcon` (FileText/FileX) + `getHistoryIconColor` (info/secondary).

**Adapter-Entscheidung:**
- `contractDocToDocumentFile()` baut ein minimales `DocumentFile`-Objekt mit `id: doc.fileId` — der `useFileDownloadURL(id)`-Hook in FilePreviewModal löst daraufhin die Download-URL per MSW-Handler auf. Kein Fork/Clone von FilePreviewModal notwendig.
- VersionHistoryPanel nimmt nur `fileId` + `fileName` — direkte Übergabe, kein Adapter nötig.

**Demo-Handler-Ergänzungen:**
- Download-URL-Handler: gibt `data:application/pdf;base64,<mini-pdf>` zurück → iframe-Vorschau funktioniert headless (Inhalt leer, aber Modal-Chrome/Titel korrekt).
- Versions-Handler: gibt 2 konsistente Demo-Versionen zurück (v2=aktuell, v1=ursprünglich).
- Upload-Handler: erzeugt neues File-Objekt in der lokalen `files`-Array und gibt es zurück → useDocumentUpload erhält ein valides `DocumentFile`.

**Offene Fragen für Darien:**
1. **entity_links-Persistenz beim API-Swap:** Die ContractDocument-Liste lebt im Zustand-Store (localStorage). Beim Backend-Swap soll die Verknüpfung über `useLinkFile` (`documentLinkApi.link(fileId, {entity_type: 'contract', entity_id: contractId})`) persistiert werden. Der `removeDocument`-Store-Action entspricht ein `useUnlinkFile(linkId)` — linkId muss dann aus der entity_links-Response ermittelt werden.
2. **Backend UploadDocument/MinIO presigned:** Der Upload-Endpunkt `/api/v1/documents/files/upload` ist im Backend als Stub/Handover-Punkt markiert (`backend-handover-luke.md`). Die XHR-Upload-Logik in `useDocumentUpload` ist bereit für den echten Endpunkt — kein FE-Umbau nötig.
3. **OnlyOffice-Editieren aus dem Vertrag heraus:** Momentan nur Vorschau via FilePreviewModal. Soll ein "Bearbeiten"-Button zum OnlyOfficeEditor aus dem Dokumente-Modul geöffnet werden können? (erfordert WOPI-Token + separaten Editor-Modal)

**Screenshots:** `desktop/.qa-screenshots/vertraege-dokumente/` (a–e) — QA `desktop/scripts/qa-vertraege-dokumente.mjs`

**Bekannte Einschränkungen (Demo-Modus):**
- iframe-PDF-Vorschau ist im Headless-Playwright leer (kein PDF-Renderer); Modal-Chrome und Dateiname korrekt.
- Versions-Button nur bei Hover sichtbar (CSS `opacity-0 group-hover:opacity-100`) — QA erzwingt Sichtbarkeit via JS `style.opacity`.

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

## Phase 8 — E-Signatur EES (Hybrid-Flow mit Canvas)  ·  marathon/luke-fe  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route: `/vertraege` → Zeile „Müller Metallbau Rahmenvertrag" anklicken → DetailPanel rechts → Unterschriften-Sektion sichtbar (2 Signer)
- Signatur-Dialog: Footer-Button „Unterschrift" oder ItemActions-Eintrag „Unterschrift" → Dialog „Digitale Unterschrift" öffnet
- Canvas-Flow: „Jetzt unterschreiben" bei Hans Müller → Canvas-Step → Unterschrift einzeichnen → „Speichern" (Canvas) → „Unterschrift übernehmen" → Badge wechselt auf „Unterschrieben", Thumbnail sichtbar
- Dispatch-Flow: „Unterzeichner hinzufügen" → Name + E-Mail → „Zur Unterschrift senden" → Status aller pending Signer wechselt auf sent, Dialog schließt
- Rapporte: Smoke-Test `/rapporte` — kein Import-Fehler durch Canvas-Umzug

**Was gebaut wurde:**
1. **SignatureCanvas extrahiert** nach `components/signature/SignatureCanvas.tsx` mit konfigurierbaren `hintKey`/`clearKey`-Props. `modules/rapporte/SignatureCanvas.tsx` leitet jetzt nur noch re-export weiter (1-Zeiler). Rapporte-Smoke grün.
2. **Store** (`stores/vertraege.ts`): `ContractSigner` um `signatureDataUrl?: string` + `signedVia?: 'canvas' | 'dispatch'` erweitert. Neue Actions `signSigner` (schreibt `contract_signed`-History-Eintrag) + `updateSigners` (ersetzt Signer-Array ohne spurious `contract_updated`).
3. **ESignaturDialog** komplett neu: Hybrid-Modus mit inline Canvas-Step (kein zweiter Dialog-Layer), pro Signer „Jetzt unterschreiben"-Button, Thumbnail-img nach Signatur. Dispatch-Flow (Abwesende) bleibt funktional.
4. **Text-Entschärfung**: Titel entfernt „— Skribble"; infoBanner ersetzt Skribble-Claim durch ehrlichen EES-Hinweis; Skribble erscheint als disabled „Bald verfügbar"-Zeile. Kein „rechtlich bindend".
5. **DetailPanel SignerSection**: kompakte Signer-Liste (Name, Status-Badge, signedAt, Thumbnail bei Canvas-Signatur), Leer-Zustand mit dashed border.
6. **i18n ×4**: alle neuen Keys punktuell eingefügt (de/en/fr/it), {var}-Syntax, keine {{var}}.

**Canvas-Extraktion begründet (rapporte-Touch):**
Die CLAUDE.md-Regel „nie Komponenten duplizieren" erforderte die Extraktion. rapporte/SignatureCanvas.tsx ist ein minimaler Re-Export — keine Logik- oder i18n-Änderung. Die rapporte.signature.*-Keys bleiben vollständig aktiv (hintKey-Default bleibt `rapporte.signature.hint`). Risiko: minimal (1 Zeile, kein Typ-Change). Playwright-Smoke verifiziert.

**Offene Fragen für Darien:**
1. **Rechtsstufe EES vs. QES (Kosten-Frage):** Wann wird Skribble-Integration gebaut? EES (Canvas) reicht für Rahmenverträge intern, aber für formgebundene Verträge (Arbeitsverträge, Mietverträge > 1 Jahr) ist QES/AES rechtlich geboten. Entscheidung: Pilot-Start mit EES + klarer Kommunikation, Skribble frühestens Phase D.
2. **Dispatch-E-Mails real versenden:** Der `handleSend`-Flow setzt Signer auf `sent` — aber keine E-Mail geht raus. Backend-Endpunkt für E-Signatur-Dispatch fehlt. Scope: POST /api/v1/contracts/{id}/dispatch-signature (Darien-Estimation nötig).
3. **signers-Persistenz beim API-Swap:** Signers inkl. `signatureDataUrl` leben im localStorage-Store. Beim Backend-Swap muss `signers` in der `contracts`-Tabelle gespeichert werden (kein `signature_provider`-Feld im aktuellen Schema — ist Phase-D-Placeholder laut `integrationen.md`). Die Canvas-DataURL ist ein Base64-PNG — Ablage in S3/MinIO mit separatem `contract_signatures`-Endpoint sinnvoll statt BLOB in DB.

**Screenshots:** `desktop/.qa-screenshots/vertraege-esignatur/` (a–f, inkl. c1-canvas-drawn, c2-signed-with-thumbnail) — QA `desktop/scripts/qa-vertraege-esignatur.mjs`

**Bekannte Einschränkungen (Demo-Modus):**
- Lukas Brunner (Alt-Bestand) hat kein `signatureDataUrl` — bewusst kein Fake-DataUrl (spezifikationskonform). Thumbnail erscheint nur für neue Canvas-Signaturen.
- `signedVia='dispatch'` wird beim Dispatch-Flow nicht auf existierende Signer geschrieben (nur neue pending Signer bekommen das Flag) — sauber für Phase D.

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
