# Pilot: inventar — Demo-Tiefe auf Standard

> **Warum Pilot:** klein + deckt ALLE 5 Muster-Elemente ab (Modal-Umstellung, Settings-Panel, toter Button, klickbare Sub-Liste, SortMenu, Export). Das hier gebaute Muster ist 1:1 auf die anderen 6 übertragbar.
> **Datei:** `modules/inventar/InventarPage.tsx` (+ neue `modules/inventar/settings/InventarSettingsPanel.tsx`, `stores/inventarPrefs.ts`, `stores/inventarTenant.ts`).
> **Ist-Zustand (Audit):** 4 Tabs — Artikel (Tabelle), Lagerorte (Karten), Bewegungen (Tabelle), Inventur (expandierbare Cards). Barcode-Scanner-Dialog ✅, Bestandsbewegungen echte API (Adjust/Record) ✅, Inventur-Buchen API ✅, Anhänge-Upload ✅. EmptyStates in allen 4 Tabs ✅. Artikel-Detail = `DetailPanel` (Slide-over).

## Aufgaben

- **I-1 · Artikel-Detail `DetailPanel`→`shared/DetailModal`.** Import + Component tauschen (API-kompatibel). Inhalt beibehalten/aufräumen: Meta-Grid (SKU · Kategorie · Bestand/Bestandsbalken · Mindestmenge · Lagerort · Einheit · Preis), Sektion **Bewegungs-Historie** (Preview→ganze Liste), Sektion **Anhänge** (`useItemAttachments`), Footer-Aktionen (Bestand anpassen / buchen). Titel = Artikelname, Badge = Bestand-Status (OK / unter Mindestmenge = rot).
- **I-2 · Lagerort-Karten klickbar → Detail-Modal.** Karte `role="button"`+`onClick`. Neues kleines `DetailModal`: Lagerort-Name/Code, Kapazität/Auslastung, **Liste der Artikel in diesem Lagerort** (klickbar → Artikel-Modal via `onBack`-Kette). Innere Buttons `stopPropagation`.
- **I-3 · „Neue Inventur"-Button echt machen** (heute `toast.success()` ohne Effekt, `InventarPage.tsx:1237`). Dialog „Inventur-Session anlegen" (Name/Lagerort/Datum) → legt Session an. Falls kein Backend-Endpoint: **stateful MSW-Handler** (Muster: notifications-snooze-Handler #9) + `🔒`-Zeile in `backend-gaps.md`.
- **I-4 · Settings-Panel** (`moduleId='inventar'`, ist LEADABLE).
  - `stores/inventarPrefs.ts` (personal, nach `crmPrefs.ts`): Default-Tab, Ansicht/Dichte, „Mindestmengen-Warnung anzeigen".
  - `stores/inventarTenant.ts` (tenant, nach `dialerTenant.ts`): Standard-Einheit (Stück/kg/…), Mindestbestand-Default, Barcode-Format (EAN13/QR/…).
  - `InventarSettingsPanel.tsx` (nach `VideoSettingsPanel.tsx`) → `module-settings-registry.tsx` Eintrag (`id:'inventar'`, `navMatch:['/inventar']`) + beide Stores in `useHydrateModuleSettings.ts` + `moduleSettings.entries.inventar` i18n ×4.
- **I-5 · SortMenu** auf der Artikel-Tabelle (`shared/SortMenu`): Name / Bestand / Kategorie / Lagerort, Feld+Richtung.
- **I-6 · Export** (echter Blob, kein Stub): Artikel-Liste **und** Bewegungen als CSV. Helper `finance-export.ts` `downloadCsv` wiederverwenden oder inline (Muster `CallHistoryDetailModal.tsx`).
- **I-7 · Gate:** i18n ×4 · `tsconfig.inventarcheck.json` (Muster videocheck) scoped tsc · eslint geänderte Dateien · Playwright-QA `scripts/qa-inventar-tiefe.mjs` (Artikel-Zeile→Modal · Lagerort→Modal · Neue-Inventur-Dialog · SortMenu · Export-Klick · Settings-Panel-Assertion) **+ Bilder ansehen** → ein Commit + Push.

## Verifikations-Checkliste (Screenshots ansehen)
- [ ] Artikel-Zeile ganz klickbar → zentriertes Modal mit Bestandsbalken + Bewegungen + Anhängen
- [ ] Lagerort-Karte klickbar → Modal mit Artikel-Liste
- [ ] „Neue Inventur" öffnet echten Dialog (kein Toast-Stub), Session erscheint danach in der Liste
- [ ] Settings-Overlay auf `/inventar` zeigt Inventar-Panel (personal + tenant, Lock für Nicht-Leads)
- [ ] SortMenu sortiert die Artikel sichtbar
- [ ] Export lädt echte CSV herunter
- [ ] i18n: keine Raw-Keys, keine `{{}}`, alle 4 Sprachen · keine pageerrors

## Nach dem Pilot
Muster steht → die anderen 6 nach demselben Schema (README-Tabelle). vermietung/rapporte als Nächstes (klein), dann schichten/fuhrpark/einkauf (mittel), produktion zuletzt (groß, Statuswechsel-Endpoint mit Luke klären).
