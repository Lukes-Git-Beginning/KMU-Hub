# Editor-Pivot — Anforderungen (Darien 2026-07-22, nach Live-Test)

Der Editor ist zum **Edit-in-place**-Werkzeug geworden (Vorschau begehbar, Elemente im Bild anklicken → inline bearbeiten). Nach dem ersten Live-Test hat Darien die volle Spezifikation gegeben. **Pilot = Kontakte.**

## R1 — Editor ist zum Bearbeiten, nicht zum Benutzen (Aktionen dürfen nicht rausführen/mutieren)
Im Editor dürfen In-Modul-Aktionen NICHT das Modul verlassen oder Live-Daten ändern:
- **Navigation raus** (E-Mail → Mail-Modul, Anruf → /chat, rausführende Links unten) → **`useBlocker(true)` in EditorWorkspace** blockt alle Router-Navigationen. State-basierte Nav (Tabs, Detail-Modal) routet nicht → unberührt. Electron-Close = window.close (kein Router) → nie getrappt. ✅ **GEBAUT** (dieser Commit).
- **Mutationen** (Löschen, Favorisieren, Nachricht, Neu anlegen etc.) → beim Instrumentieren des Moduls (R3) die Action-Handler auf No-Op schalten wenn `editing` (EditorSurface signalisiert es schon). ⏳ offen.

## R2 — Ganzes Modul MIT oberer Tab-Leiste rendern + begehbar
Der Sandbox rendert aktuell nur `KontaktePage` (den „Kontakte"-Unterreiter), NICHT die obere Tab-Leiste (Kontakte / Leads / Unternehmen / Deals / Aktivitäten / Auswertungen). **Nötig:** den echten Modul-ROOT rendern (CRM-Shell mit Tab-Leiste), damit man zu Unternehmen/Deals/… wechseln und DORT anpassen kann. → `editorModules.Component` auf den Modul-Root umstellen (CRM-Wrapper finden). ⏳ offen.

## R3 — Edit-in-place ÜBERALL, nicht nur die Kategorie-Leiste
- Obere Tabs (Kontakte/Leads/Unternehmen/Deals) namenstechnisch anpassbar (EditableText).
- **Im Kontakt-Detail** (und allen Unter-Ansichten/Unterkategorien): Feld-Labels + Werte anklickbar → umbenennen/bearbeiten. Das ist die direkte Antwort auf „Eskalationsgrund unsichtbar" — man öffnet den Kontakt und editiert das Feld dort.
- Gilt für ALLE Unterkategorien, nicht nur die Hauptliste. ⏳ offen (systematisches Instrumentieren pro Modul).

## R4 — Tab-Sichtbarkeit an/aus + Erweiterbarkeit (Modul-Komposition, NEUE Dimension)
- Obere Tabs (Deals/Leads/Unternehmen/Kontakte/…) **einzeln ein-/ausschaltbar** im Editor; ausgeschaltete verschwinden aus der Leiste (z.B. keine Leads → kein Leads-Reiter). Soft (Daten bleiben, nur UI-Bereich versteckt).
- **Erweiterbar:** spätere Updates bringen neue Tabs/Bereiche → pro Tenant aktivier-/deaktivierbar („nicht jeder braucht alles"). Alles im Editor.
- **Neue Customization-Dimension** = Tab/Bereichs-Sichtbarkeit pro Tenant (Datenmodell im Overlay: `moduleAreas: { [moduleKey]: { [areaKey]: enabled } }`, ins Draft/Deploy einhängen wie Labels/Value-Sets/Fields). ⏳ offen — braucht Datenmodell + Modul liest es + Editor-Toggle-UI.

## Bau-Reihenfolge (Vorschlag)
1. **R1-Nav** ✅ · **R1-Mutationen** (im Zuge R3)
2. **R2** Modul-Root mit Tabs rendern + begehbar (Tabs sind state-basiert → funktionieren)
3. **R3** Instrumentieren: Tabs + Kontakt-Detail-Felder + Unter-Ansichten edit-in-place; Action-Handler im Editor no-op
4. **R4** Tab-Sichtbarkeit-Dimension (Datenmodell + Modul-Konsum + Editor-Toggles)
5. Chrome aufräumen (Panel → „Neu anlegen" + Kontext-Inspektor)
6. Danach Muster über andere Module ausrollen

Jede Stufe: bauen → i18n ×4 → scoped tsc → eslint → Playwright-Screenshot-QA (Bilder ansehen) → Commit.
