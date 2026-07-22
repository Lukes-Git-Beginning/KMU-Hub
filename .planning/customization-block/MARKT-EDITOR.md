---
title: "Markt-Editor: Visual Edit-in-Place UX-Recherche"
updated: 2026-07-22
tags: [customization, research, ux, visual-editor, odoo-studio, salesforce, servicenow, webflow, framer]
---

# Markt-Editor: Wie sieht ein guter Modul-Editor aus?

> Recherche-Paket für den Cosmi-Modul-Editor (§3 des EDITOR-VISION-BRIEFING).
> Fokus: UX/Visual — nicht Mechanik. Wie öffnet es sich, wie fühlt es sich an, was ist gut, was ist überladen?
> Odoo Studio als Startpunkt (bereits in MARKT-B), hier vertieft + neue Produkte.

---

## 1. Odoo Studio — Direkter Vergleich (Vertiefung)

### Wie es sich öffnet

Odoo Studio ist kein separates Fenster — es überlagert die laufende App. Der Admin navigiert in Odoo zu dem Modul (z.B. CRM → Kontakte), klickt dann den **Studio-Toggle-Button** (Stift-Icon, oben in der Toolbar). Die App bleibt vollständig sichtbar; über sie legt sich der Studio-Modus.

**Visuelles Ergebnis:** Die bestehende Formularansicht bleibt auf dem Bildschirm. Links erscheint eine Seitenleiste mit zwei Tabs: **"Add"** (Elemente hinzufügen) und **"Properties"** (Eigenschaften des selektierten Elements). Das Formular selbst wird leicht in den Bearbeitungsmodus versetzt — Felder bekommen dezente Edit-Affordances (Drag-Handle-Icons, Hover-Effekte).

### Toolbox-Struktur (links)

Der **Add-Tab** enthält:
- **New Fields:** Feldtyp-Palette (21 Typen: Text, Integer, Float, Boolean, Selection, Many2One, One2Many, Many2Many, Date, Datetime, Binary, Html, Monetary, Priority, Reference, Integer-Sequence, Char, Properties, Tags, Image, Json). Icons + Feldtyp-Namen. Drag-and-Drop direkt auf die Formularfläche.
- **Existing Fields:** Liste aller Felder des Modells, die noch nicht in der View sind. Klick auf "Existing Fields" klappt die Liste auf → Drag-and-Drop in die gewünschte Position.
- **Structural Elements:** Tabs, Columns, Separator — ebenfalls per Drag-and-Drop auf das Formular.

Der **Properties-Tab** erscheint, sobald ein Element im Formular selektiert wird. Er zeigt view-spezifische Konfiguration: Label, Widget-Wahl (Dropdown mit verfügbaren Widgets für den Feldtyp), Sichtbarkeit (Invisible/Required/Readonly je nach Bedingung), Zugriffskontrolle (nur für bestimmte Gruppen sichtbar).

### Interaktionsmodell: Feld-Drag und Inline-Rename

1. Admin klickt "New Fields" im Add-Tab.
2. Zieht z.B. "Text" auf die Formularfläche → Drop-Zones erscheinen (gestrichelte Trennlinien).
3. Nach dem Drop öffnet sich sofort der Properties-Tab. Das Label-Feld ist fokussiert — der Admin tippt den gewünschten Feldnamen ein.
4. Sobald er außerhalb des Label-Feldes klickt, wird der technische Name automatisch generiert (z.B. `x_customer_tier`).
5. Für **Inline-Rename bereits existierender Felder:** Klick auf das Label direkt im Formular → Properties-Tab wechselt, Label-Feld wird aktiv. Kein separates Modal.

### Hover/Selektionszustände auf dem Formular

- **Hover:** Felder bekommen einen hellblauen/grauen Rahmen + ein Drag-Handle (≡) links.
- **Selektion (Klick):** Feld wird mit einem blauen Rahmen hervorgehoben, Properties-Tab zeigt sofort die Eigenschaften.
- **Strukturelemente (Tabs, Columns):** Klick auf die Gruppenüberschrift selektiert die ganze Gruppe, Properties zeigen Gruppenoptionen.

### Charakter / Gesamtgefühl

Das Gefühl ist **direkt, aber ungeschützt.** Änderungen wirken sofort live auf die Produktionsinstanz — kein Preview, kein Entwurfsmodus, kein Sandbox. Das Modul bleibt sichtbar (echter edit-in-place), aber der mentale Übergang "ich bearbeite die App, nicht nutze sie" ist nicht explizit geschützt. Der visuelle Zustand "ich bin im Editor-Modus" ist subtil — nur die linke Seitenleiste und ein "Close"-Button oben rechts signalisieren den Modus.

**Bekannte Schwäche:** Drag-and-Drop von Spalten/Tabs in Form Views (ab v16) ist zeitweise kaputt (Community-Bug). Das "Edit-in-Place ohne Sandbox" wird von Implementierungspartnern als Risikobereich eingestuft — ein Versehen ist sofort live.

### Was ist gut / was ist überladen

**Gut:**
- Modul bleibt sichtbar — kein Kontextverlust. Man sieht was man baut.
- Add-Tab vs. Properties-Tab: klare Trennung zwischen "Hinzufügen" und "Konfigurieren".
- Inline-Label-Rename direkt im Formular, kein Modal-Overhead.
- Feldtyp-Palette mit 21 Typen gibt Power ohne den Nutzer zu überwältigen (Icons helfen).

**Überladen / problematisch:**
- Kein Sandbox/Preview — Änderungen sind sofort live. Für tenant-weite Config fatal.
- Kein Audit-Log für Studio-Änderungen — wer hat was wann gemacht?
- Kein Undo/Redo für Strukturänderungen (Feld gelöscht → weg).
- Der "Edit-Modus"-Signal ist zu schwach — nur Seitenleiste und Close-Button. Kein visueller Banner "Du bearbeitest jetzt".
- Drag-and-Drop hat bekannte Bugs in neueren Versionen.
- Automation-Editor (Trigger/Action) ist für KMU-Admins zu technisch.

**Quellen:**
- https://www.odoo.com/documentation/19.0/applications/studio.html
- https://www.odoo.com/documentation/18.0/applications/studio/fields.html
- https://odootricks.tips/about/odoo-studio/adding-fields-using-odoo-studio/
- https://www.cybrosys.com/blog/how-to-customize-views-reports-in-odoo-18-studio
- https://www.odoo.com/forum/help-1/odoo-studio-cant-drag-drop-columns-or-tabs-in-form-view-v16e-220626

---

## 2. Salesforce Lightning App Builder

### Wie es sich öffnet

Der Lightning App Builder ist vom normalen CRM-Nutzungsmodus vollständig getrennt. Zugang via Setup-Zahnrad (oben rechts) → "Lightning App Builder" → Seite auswählen oder neu erstellen. Man verlässt das CRM komplett und landet in einem dedizierten Editor-Interface. Kein edit-in-place über der laufenden App.

### Visuelles Layout — Drei-Panel-Architektur

Das Interface folgt dem klassischen Drei-Panel-Muster:
- **Links (Komponenten-Panel):** Kategorisierte Liste aller verfügbaren Lightning-Komponenten (Standard + Custom). Kategorien wie "Standard", "Custom", "AppExchange". Suche. Drag-Affordance.
- **Mitte (Canvas):** Die Seitenvorschau. Für Record-Pages: der tatsächliche Record-Layout mit Komponenten-Slots. Klick auf eine Komponente selektiert sie (keine blaue Outline wie Webflow, eher ein gestrichelter Rahmen + Griffe).
- **Rechts (Properties-Panel):** Kontextspezifisch für die selektierte Komponente. Felder für Datenquellen, Filter, Display-Optionen, Dynamic Visibility Conditions.

**Gerätemodi:** In der oberen Toolbar kann zwischen Desktop, Tablet und Mobil gewechselt werden — der Canvas skaliert entsprechend.

### Interaktionsmodell

1. Komponente aus dem linken Panel auf den Canvas ziehen → Drop-Zones in den Komponenten-Slots leuchten auf.
2. Klick auf eine bereits platzierte Komponente → Properties-Panel rechts zeigt Konfiguration.
3. **Dynamic Visibility Conditions:** Für jede Komponente kann im Properties-Panel eine Sichtbarkeitsregel gesetzt werden (z.B. "nur zeigen wenn Record.Type = 'Partner'"). Das ist ein separates Formular, kein visueller Rule-Builder.
4. **Speichern und Aktivieren:** "Save" speichert den Entwurf. "Activate" macht die Page live für die konfigurierten Zielgruppen (Profiles/Apps). Es gibt einen Aktivierungs-Dialog mit Assignments.

### Charakter / Gesamtgefühl

Der Lightning App Builder fühlt sich **mächtig aber entkoppelt** an. Da man das CRM verlässt, fehlt das direkte Feedback "so sieht es im echten Kontext aus". Die Vorschau ist eine approximierte Darstellung, nicht die echte Liveansicht.

**Stärken:** Klare Drei-Panel-Struktur, Dynamic Visibility ohne Code, Gerätemodi-Wechsel, der Aktivierungs-Dialog zwingt zu bewusster Entscheidung.

**Schwächen:** Separation vom Live-CRM ist ein Kontextverlust. Das Properties-Panel kann bei komplexen Standard-Komponenten sehr lang werden (viele Konfigurationsfelder untereinander ohne visuelle Hierarchie). Keine Live-Preview im CRM-Kontext — man muss zurück ins CRM gehen um zu prüfen.

### Was ist gut / was ist überladen

**Gut:**
- Aktivierungs-Dialog als expliziter "Go-Live"-Schritt mit Zielgruppenauswahl.
- Gerätemodus-Switch direkt in der Toolbar.
- Klar abgegrenzt: Editor-Modus ist nicht der Nutzungsmodus.

**Überladen:**
- Properties-Panel bei komplexen Komponenten: zu viele Felder, keine progressive Disclosure.
- Setup ist ein komplett anderer Kontext als das CRM — hoher kognitiver Aufwand.
- Kein natives Rollback — Änderungen an einer Seite können nur durch manuelles Revert rückgängig gemacht werden.
- Component Visibility Conditions sind nur Admins verständlich, nicht KMU-tauglich ohne Training.

**Quellen:**
- https://trailhead.salesforce.com/content/learn/modules/lightning_app_builder/lightning_app_builder_intro
- https://sdlccorp.com/post/lightning-app-builder-complete-tutorial/
- https://www.salesforce.com/platform/drag-and-drop-app-builder/

---

## 3. ServiceNow UI Builder

### Wie es sich öffnet

Der UI Builder ist ein eigenständiges Tool innerhalb der ServiceNow-Plattform, erreichbar über die Navigation → "UI Builder". Es ist kein edit-in-place über dem laufenden Workspace, sondern ein dedizierter Page-Builder in einem separaten Interface-Tab.

### Visuelles Layout

Wie Lightning App Builder: **Drei-Panel-Architektur.**
- **Links:** Pages/Variants-Navigation + Komponenten-Bibliothek ("Content Tree" und "Components").
- **Mitte:** Canvas — der aufgebaute Workspace als Vorschau. Komponenten sind auf einem Grid platziert.
- **Rechts:** Properties-Panel für die selektierte Komponente (Binding an Datenquellen, Style-Overrides, Conditional Visibility).

**Variant-System:** Für jede Seite können Varianten erstellt werden (z.B. "Agent View" vs. "Manager View" auf derselben URL). Die Varianten-Auswahl ist in der linken Navigation. Das ist ein mächtiges Muster für rollenbasierte Anpassung.

### Interaktionsmodell

1. Komponente aus dem Content-Tree oder der Bibliothek auf den Canvas ziehen.
2. Selektion → Properties-Panel rechts zeigt Bindings und Style-Optionen.
3. **Data Binding:** Die Properties sind oft "Binding"-Felder, die auf ServiceNow-Datenquellen (Tables, Scripted Data Resources) zeigen. Für Nicht-Techniker schwer verständlich.
4. **Save & Publish:** Es gibt einen expliziten "Publish"-Button. Unveröffentlichte Änderungen sind als Entwurf.

### Charakter / Gesamtgefühl

Der UI Builder fühlt sich **professionell aber technisch** an. Er ist primär für ServiceNow-Entwickler und erfahrene Admins ausgelegt, nicht für Citizen Developer. Data Binding an ServiceNow-Tabellen und Scripted Resources setzt Plattformwissen voraus.

**Stärken:** Varianten-System für rollenbasierte Page-Layouts ist einzigartig mächtig. Expliziter Publish-Step schützt vor unbeabsichtigten Änderungen.

**Schwächen:** Hohe Einstiegshürde durch Data-Binding-Konzepte. Kein wirkliches "Live in Context" — der Builder ist eine approximierte Vorschau. Für KMU-Admins ohne ServiceNow-Erfahrung nicht bedienbar.

### Was ist gut / was ist überladen

**Gut:**
- Variant-System für Rollen-basierte Layouts.
- Expliziter Publish-Schritt schützt Live-System.
- Komponenten-Bibliothek ist kategorisiert und durchsuchbar.

**Überladen:**
- Data-Binding-Syntax ist technisch — keine KMU-Tauglichkeit.
- Keine "echte" edit-in-place-Erfahrung — der Builder ist von der Live-Instanz getrennt.
- Properties-Panel kann sehr lang werden (viele Binding-Felder).

**Quellen:**
- https://www.servicenow.com/products/ui-builder.html
- https://www.servicenow.com/community/next-experience-blog/inside-ui-builder-part-one/ba-p/3395615
- https://www.servicenow.com/docs/bundle/zurich-application-development/page/administer/ui-builder/concept/ui-builder-overview.html

---

## 4. Webflow Designer — Referenzklasse für erstklassiges visuelles Editing

Webflow ist kein CRM-Customizer, aber der **Gold-Standard für visuelles Element-Editing** in SaaS. Das Interaktionsmodell ist direkt auf den Cosmi-Editor übertragbar.

### Drei-Panel-Architektur (kanonisch)

- **Linke Seitenleiste (kontextabhängig per Icon-Tabs):**
  - Add Panel: Drag-and-Drop-Elemente (Layouts, Typografie, Media, Components, etc.)
  - Navigator: Baumstruktur aller Elemente auf der Seite (verschachtelter Baum, Eltern-Kind-Hierarchie)
  - Pages, CMS, Assets, Symbols: weitere Kontext-Panels
- **Mitte (Canvas):** Die Seite selbst. Volles WYSIWYG — man sieht genau was der Nutzer sieht.
- **Rechte Seitenleiste (kontextabhängig per Selektion):**
  - Style Panel: Typography, Layout (Flexbox/Grid), Spacing, Backgrounds, Borders, Effects — für das selektierte Element
  - Element Settings: Element-spezifische Optionen (z.B. Link-Ziel für einen Link, Bild-Alt für ein Bild)

### Selektionsmodell (das entscheidende UX-Muster)

**Hover:** Blauer Outline-Rahmen erscheint um das Element. Ein Label in der oberen linken Ecke des Rahmens zeigt den Element-Namen (oder den Class-Namen, wenn vorhanden). Optional: Gear-Icon wenn das Element zusätzliche Settings hat.

**Selektion (Klick):** Blauer Rahmen bleibt, wird aber "ausgefüllter" (solider). Die rechte Seitenleiste wechselt sofort auf die Eigenschaften des selektierten Elements. Das Navigator-Panel scrollt automatisch zum selektierten Element.

**Drag auf dem Canvas:** Blauer Indikator-Strich zeigt an, wo das Element platziert wird (zwischen anderen Elementen). Ein blauer Outline erscheint um das potenzielle Eltern-Element. Präzises Ziel-Feedback vor dem Drop.

**X-Ray Modus:** Sonderansicht (Graustufen) die alle Element-Ränder, Padding und Margin sichtbar macht — für Strukturdiagnose.

**Gerätemodus:** Toolbar oben mit Device-Breakpoint-Auswahl. Canvas skaliert und zeigt den responsiven Zustand.

### Publish-Flow

Webflow trennt den **Design-Modus** (Editor-Toolbar sichtbar, Canvas interaktiv) vom **Live-Zustand** klar. Ein "Publish"-Button in der oberen Toolbar. Noch nicht veröffentlichte Änderungen zeigen einen Hinweis "Changes not yet published". Zwei Ziele: Webflow-Subdomain oder Custom Domain, einzeln wählbar.

Für Content-Editoren (nicht Designer): **Edit Mode** — ein separater, vereinfachter Modus, der nur Content-Felder (Text, Bilder) editierbar macht, ohne den strukturellen Canvas zu öffnen. Editoren sehen blue-outline-Highlights auf editierbaren Content-Feldern.

### Was ist gut / was ist überladen

**Gut:**
- Blue-Outline-Hover + Label-Chip: sofort verständlich, welches Element man anklicken wird.
- Navigator-Sync mit Canvas-Selektion: Kontext nie verloren.
- Style Panel ist tiefgehend aber strukturiert in Sektionen (Layout, Typography, etc.).
- Klare "Publish"-Gate mit visueller "unpublished changes"-Anzeige.
- Edit Mode für Nicht-Designer: reduzierte Oberfläche, trotzdem in-context.

**Überladen:**
- Komplexität für Nicht-Designer trotz klarem Modell sehr hoch — das ist explizit ein Werkzeug für Webprofis.
- Style Panel mit Flexbox/Grid ist für non-designers overload.
- Kein "Geplantes Publish" (Scheduled Publishing) natif.

**Quellen:**
- https://help.webflow.com/hc/en-us/articles/33961319255059-Webflow-canvas-overview
- https://help.webflow.com/hc/en-us/articles/33961320786451-Navigator
- https://webflow.com/glossary/left-tool-bar

---

## 5. Framer — Erstklassiges on-page Editing + Editor Bar

### Interaktionsmodell: zwei Modi

**Design-Modus (Framer-App):** Der klassische Canvas-Editor für Designer. Freeform-Canvas mit Auto-Layout. Linkes Panel (Insert, Pages, Assets), rechtes Panel (Properties, Layout-Controls). Elemente auf dem Canvas per Klick selektierbar. Eine Play-Button in der Toolbar öffnet den **Preview-Modus** (interaktive Vorschau in einem separaten Fenster/Tab, nicht der Editor selbst).

**On-Page Edit-Modus (CMS 2.0, August 2025):** Der bahnbrechende Ansatz für Content-Editoren ohne Designer-Kenntnisse:
1. Man besucht die **veröffentlichte Seite** im Browser.
2. Ein **Editor Bar** erscheint am unteren Rand (für Besucher unsichtbar, nur für Collaborators).
3. Klick auf den Editor Bar → die Seite wechselt in den Edit-Modus.
4. **Blaue Outlines erscheinen um alle editierbaren Elemente** (Text, Bilder, CMS-Felder, Component Properties).
5. Klick auf ein Element → direkte In-Place-Bearbeitung (Tippen für Text, Upload für Bilder).
6. Für versteckte Felder (SEO-Meta, Toggle-Settings): "Show All Fields"-Button öffnet ein Modal mit allen Feldern.
7. Nach Änderungen: "Finish Editing" (für reine Content-Editoren) → Notification an Publisher → Publisher klickt "Publish".

### Editor Bar: Visuelles Design

Der Editor Bar ist **schwebend, am unteren Rand des Viewports**, in die Ecke gedrängt. Für Besucher nicht sichtbar. April-2025-Redesign: leichter, subtiler, rechts positioniert, weniger intrusiv als die Originalversion (die mittig unten saß und als aufdringlich empfunden wurde). Klarer "Open"-Button der in den Edit-Modus führt.

### Sandbox-Signal: Vertrauen durch Edit-Mode-Sichtbarkeit

Der entscheidende UX-Distinktion: Im Edit-Modus sieht der Editor **blaue Outlines auf allen editierbaren Elementen**. Das ist das visuelle Signal "du bist jetzt im Edit-Modus, diese Elemente kannst du bearbeiten". Strukturelle Elemente (Layout-Templates, Overlays, Sektions-Rahmen) sind **gesperrt** und bekommen keine Outline — der Editor kann sie nicht versehentlich zerstören.

**Design-Integrität durch selektive Editierbarkeit:** Nur Content-Felder sind editierbar. Layout bleibt unveränderlich. Das ist ein fundamentales Sicherheitsprinzip — ähnlich dem, was Cosmi für Admin-Config braucht (was ist editierbar, was ist Cosmi-Core?).

### Was ist gut / was ist überladen

**Gut:**
- On-Page-Edit: man ist im echten Kontext, keine approximierte Vorschau.
- Blaue Outlines auf editierbaren Elementen: universell verständliches Signal.
- Editor Bar: unsichtbar für Nicht-Berechtigte, nicht-intrusiv für Berechtigte.
- Publisher/Editor-Rollentrennning: klare Verantwortlichkeit.
- Strukturelle Elemente gesperrt: kein versehentliches Layout-Breaking.

**Überladen:**
- CMS 2.0 ist auf Content-Editing ausgerichtet — strukturelle Änderungen erfordern weiterhin den Design-Modus (für echte Modul-Konfiguration nicht ausreichend).
- On-Page-Edit erfordert eine bereits gepublishte Seite — kein Sandbox-Preview vor dem ersten Publish.

**Quellen:**
- https://www.framer.com/updates/editor-bar
- https://www.framer.com/updates/on-page-editing
- https://www.flatlineagency.com/blog/framer-on-page-editing/

---

## 6. Sanity Visual Editing — Overlay-Architektur als Referenz

Sanity ist ein Headless CMS mit einem einzigartigen Overlay-System das sich direkt auf Cosmi überträgt.

### Overlay-Mechanik (technisch + visuell)

Das System rendert die **echte Live-Website** in einem iFrame innerhalb des Sanity "Presentation Tool". Über dem iFrame-Inhalt werden transparente Overlay-Divs gelegt, die durch `getBoundingClientRect()` exakt auf die darunterliegenden Content-Elemente ausgerichtet sind.

**Visueller Ablauf:**
1. Erstes Öffnen: **kurzes Flash-Animation** (1.5 Sekunden) — alle Overlays blinken auf, um dem Editor zu zeigen "das alles ist editierbar". Dann werden die Overlays wieder transparent.
2. **Hover:** Farbiger Border erscheint um das Element. Eine **Actions Bar** erscheint: "Open in Studio" + Tab mit dem Dokumententitel.
3. **Selektion (Klick):** Dünnerer Border (zeigt Fokus). Das Sanity Studio auf der anderen Seite des Split-Screens navigiert automatisch zum entsprechenden Feld und fokussiert es.
4. **Kein separates Properties-Panel** im Overlay selbst — das Studio daneben übernimmt die Rolle des Property-Editors.

**Split-Screen-Ansatz:** Presentation Tool = links die Website, rechts das Studio. Man sieht Änderungen im Studio sofort auf der Website links. Draft-Content wird live gerendert.

### Draft/Publish-Modell

Sanity trennt Draft-Content von Published-Content auf Dokument-Ebene. Im Presentation Tool sieht der Editor immer den Draft-Stand. Veröffentlichte Seite zeigt Published-Stand. Die Trennung ist transparent.

### Was ist gut / was ist überladen

**Gut:**
- Flash-Animation beim Öffnen: einmaliges, klares "alles was blinkt ist editierbar".
- Hover-State mit Actions Bar ist minimal-intrusiv.
- Split-Screen: sofortiges Feedback ohne Kontextverlust.
- iFrame-basiert: die echte Website, keine Simulation.

**Überladen:**
- Split-Screen erfordert breiten Bildschirm.
- Technische Implementierung (Stega-Encoding, Content Source Maps) ist komplex im eigenen Kontext nicht adaptierbar.

**Quellen:**
- https://www.sanity.io/docs/visual-editing/introduction-to-visual-editing
- https://www.sanity.io/docs/visual-editing/visual-editing-overlays

---

## 7. Retool — Canvas-Editor für Developer-Tools (Referenz für Grid-Editing)

Retool ist ein Entwickler-Tool, kein KMU-Customizer. Sein **Grid-Canvas-Modell** ist aber eine wichtige Referenz für den Umgang mit positionierbaren Elementen.

### Canvas-Mechanik

Retool nutzt ein **12-Spalten-Snap-Grid**. Komponenten werden auf dem Canvas platziert und rasten in Grid-Zellen ein. Beim Drag werden die Grid-Linien sichtbar, Kanten der Komponente rasten an Grid-Punkten ein. Resize über Corner-Handles.

**Linkes Panel:** Komponenten-Bibliothek (100+ Komponenten: Table, Form, Chart, Button, etc.) — kategorisiert und durchsuchbar.
**Rechtes Panel:** Inspector/Properties — Konfiguration der selektierten Komponente. JavaScript-Bindings direkt eingebbar.
**Kein echter Draft-Modus:** Änderungen werden gespeichert, eine "Publish"/"Deploy"-Mechanik gibt es nur via Git-Source-Control (Enterprise).

### Was ist gut / was ist überladen

**Gut:**
- Grid-Snap-System: präzises Positionieren ohne Pixel-Denken.
- 12-Spalten-System ist eine gut verstandene Konvention.

**Überladen:**
- Primär für Full-Stack-Developer, kein KMU-Tool.
- JavaScript-Bindings überall = kein echter No-Code-Zugang.

**Quellen:**
- https://docs.retool.com/apps/concepts/ide
- https://community.retool.com/t/early-access-widgetgrid-faster-editor-interactions/44871

---

## 8. Übertragbare UI/Interaktions-Muster für den Cosmi-Editor

Die folgenden Muster sind direkt adaptierbar und Cosmi-kompatibel (respektieren Premium-SaaS-Ästhetik).

### M-1: Blue-Outline-Hover + Label-Chip (Webflow → Cosmi)

Wenn der Admin mit der Maus über ein editierbares Element in der Modul-Sandbox fährt, erscheint ein dünn-blauer (oder Cosmi-Akzentfarbe) Rahmen um das Element. In der oberen linken Ecke des Rahmens: ein kleiner Label-Chip mit dem Element-Typ ("Feld: Firmenname", "Werteliste: Kontaktstatus", "Label: Primäraktion").

**Warum:** Sofort verständlich, welche Elemente editierbar sind. Kein visuelles Rauschen wenn man nicht hovert.

### M-2: Selektions-Outline mit Action-Pip (Sanity-Referenz → Cosmi)

Nach Klick auf ein Element: Rahmen wird solide + leicht dicker. Ein kleines Action-Pip erscheint (Edit-Icon oder "·" mit Tooltip). Gleichzeitig öffnet sich das rechte Properties-Panel (oder ein Slide-over) für dieses Element. Kein Modal-Overlay, kein Seiten-Wechsel.

**Warum:** Direkter mentaler Zusammenhang zwischen "ich klicke das Element" und "ich sehe die Eigenschaften". Keine Trennung des Kontexts.

### M-3: Flash-Animation beim Editor-Öffnen (Sanity → Cosmi)

Wenn der Editor-Modus für ein Modul zum ersten Mal geöffnet wird (oder bei erneutem Öffnen): alle editierbaren Elemente blinken kurz auf (~1–1.5 Sekunden, reduzierte Motion-Variante: fade in/out). Dann transparent. Danach nur noch bei Hover sichtbar.

**Warum:** Beantwortet sofort die Frage "was kann ich hier überhaupt anfassen?". Einmaliges Signal, kein dauerhaftes Rauschen.

### M-4: Drei-Panel-Grundgerüst (Webflow/Salesforce → Cosmi)

```
[Modul-Toolbar: Modul-Name | View-Switch | Undo/Redo | Preview-Toggle | Draft-Indikator | Übernehmen/Entwerfen]
[Add-Panel / Nav-Tree | Modul-Canvas (Sandbox) | Properties-Panel]
```

- **Links (Add-Panel):** Nur wenn etwas hinzugefügt werden soll. Für den v1-Scope (Felder/Labels/Wertelisten): Panel zeigt die Liste aller Felder des Moduls + "Neues Feld hinzufügen". Kann zugeklappt werden.
- **Mitte (Canvas):** Das Modul als Sandbox-Rendering. Nicht die echte App.
- **Rechts (Properties-Panel):** Öffnet sich bei Selektion eines Elements. Context-abhängig: Feldkonfiguration, Label-Override, Wertelisten-Editor.

### M-5: Expliziter Edit-Mode-Banner (NN/Group-Empfehlung → Cosmi)

Ein persistenter, aber dezenter Banner/Ribbon im oberen Bereich des Editor-Fensters zeigt den aktiven Modus. Nicht intrusiv (kein blocking Modal), aber klar lesbar:

> "Bearbeitungsmodus — Änderungen sind nur in dieser Vorschau sichtbar. Noch nicht live."

Cosmi-Umsetzung: Ein 32px-Banner in einer Akzentfarbe (z.B. warmes Amber/Orange, nicht Rot — Rot signalisiert Fehler) unter der Editor-Toolbar. Text: "Entwurf — Änderungen noch nicht übernommen." Mit einem X zum Ausblenden (wird in der Session gemerkt).

**Warum:** "Mode slips" (der Nutzer vergisst, dass er im Edit-Modus ist) sind das größte UX-Risiko bei edit-in-place-Editoren. Klares, persistentes Signal schützt davor. Odoo Studio hat das nicht — und es ist der häufigste Kritikpunkt.

### M-6: Publish/Deploy-Dialog mit Terminwahl (Salesforce/ServiceNow → Cosmi)

Wenn der Admin auf "Übernehmen" klickt, öffnet sich ein fokussierter Dialog (kein Full-Screen-Modal):

```
┌─────────────────────────────────────────────┐
│  Änderungen übernehmen — [Modul: Kontakte]  │
│                                             │
│  ● Jetzt übernehmen                         │
│    Änderungen sind sofort für alle Nutzer   │
│    aktiv.                                   │
│                                             │
│  ○ Terminiert am [Datum/Uhrzeit-Picker]     │
│    Änderungen werden am gewählten Termin    │
│    automatisch aktiviert.                   │
│                                             │
│  ○ Als Entwurf speichern                    │
│    Änderungen bleiben unveröffentlicht.     │
│                                             │
│  [Abbrechen]          [Übernehmen →]        │
└─────────────────────────────────────────────┘
```

**Warum:** Dariens Vision (Entwurf + geplanter Rollout + sofort) in einem klaren Drei-Wege-Dialog. Salesforce zwingt zum "Activate"-Schritt — das schützt. Cosmi kann das eleganter machen.

### M-7: Strukturelle Elemente sperren (Framer → Cosmi)

Im v1-Editor sind Layout-Strukturen (Tabellenheader, Sektions-Rahmen, Navigation) **visuell ausgegraut + ohne Hover-Outline**. Nur Felder, Labels und Wertelisten bekommen die Edit-Affordances. Ein disabler Tooltip "Dieses Layout-Element ist im Editor nicht veränderbar" bei Hover auf gesperrte Elemente.

**Warum:** Schützt vor versehentlichen Änderungen. Macht den Scope klar ("nur das hier ist anpassbar"). Direkte Übertragung von Framers "nur Content-Felder editierbar, Layout ist gesperrt".

### M-8: Inline-Label-Rename (Odoo Studio → Cosmi)

Doppelklick auf ein Feld-Label in der Canvas → das Label-Feld wird direkt editierbar (kein Modal, kein Panel-Wechsel). Ein subtiler Cursor-Blink + Unterstrich zeigt "du editierst gerade". Enter zum Bestätigen, Escape zum Abbrechen.

**Warum:** Schlankstes mögliches Editing für den häufigsten Use-Case (Label umbenennen). Odoo Studio macht das gut; wir übernehmen es.

### M-9: Progressive Disclosure im Properties-Panel

Für jedes selektierte Element: Properties-Panel zeigt zuerst nur die **häufigsten 3–4 Eigenschaften** (Label, Pflichtfeld, Sichtbarkeit, Typ). Ein "Weitere Optionen ▾" Akkordeon klappt erweiterte Einstellungen auf. Kein scrollbarer Wall of Settings.

**Warum:** Salesforce Lightning App Builder hat das Problem — Properties-Panel bei komplexen Komponenten ist endlos. Progressive Disclosure reduziert kognitive Last.

---

## 9. Anti-Muster — Was wir NICHT übernehmen

### AP-1: Sofortige Live-Wirkung ohne Sandbox (Odoo Studio)

Das Kernproblem von Odoo Studio: keine Sandbox, Änderungen sind sofort live für alle Nutzer. Für tenant-weite Config-Änderungen (alle sehen den umbenannten Button) ist das fahrlässig. **Cosmi braucht zwingend eine Draft-Schicht** — die Editor-Sandbox hat keinen Einfluss auf das Live-System bis explizit übernommen wird.

### AP-2: Kein visuelles "Du bist im Edit-Modus"-Signal (Odoo Studio)

Odoo Studio: nur die linke Seitenleiste + ein "Close"-Button zeigen, dass man im Studio-Modus ist. Kein Banner, keine Farbänderung, kein persistentes Signal. Users verlieren das Bewusstsein für den Modus. **Cosmi braucht einen klaren, persistenten Mode-Banner**.

### AP-3: Properties-Panel als Wall of Settings (Salesforce/ServiceNow)

Wenn das Properties-Panel für eine selektierte Komponente 15+ Felder untereinander zeigt ohne Strukturierung, ist kognitive Last zu hoch. Anti-Muster: alle möglichen Optionen gleichzeitig anzeigen. **Progressive Disclosure ist Pflicht**.

### AP-4: Kontextverlust durch vollständige Trennung (Salesforce Lightning App Builder)

Das CRM verlassen, um in einem separaten Setup-Interface zu arbeiten. Man verliert den echten Datei-Kontext. **Der Cosmi-Editor muss das Modul zeigen** — als Sandbox-Rendering, aber trotzdem das echte Modul, nicht eine abstrakte Konfigurationsmaske.

### AP-5: Überfrachtung durch zu viele Edit-Ebenen gleichzeitig (Odoo Studio Automations)

Odoo Studio macht Felder + Automations + Reports + Menus + Security Rules alle im selben Tool sichtbar. Für den KMU-Admin ist das overwhelm. **Im Cosmi-Editor v1: nur Trio (Felder, Labels, Wertelisten)**. Layout, Automations, Regeln sind Folge-Stufen — nicht gleichzeitig im Editor sichtbar.

### AP-6: Drag-and-Drop für alles ohne Fallback (Odoo Studio)

Drag-and-Drop von Feldern funktioniert nicht immer (bekannte Bugs in v16+). Kein Fallback für nicht-Drag-Nutzer. **Im Cosmi-Editor v1:** Primär Klick-basiertes Editing (Klick → Panel öffnet sich). Drag als optionale Verbesserung, nicht als primäres Interaktionsmuster.

### AP-7: Generisches Dashboard-Look (kein Cosmi-Anti-Muster explizit)

Kein Odoo-Klon-Look (graue Seitenleiste mit langen Feldlisten). Kein Salesforce-Setup-Look (trockenes dreispaltiges Admin-Interface in Corporate-Blau). **Cosmi braucht Editorial-Premium-Optik**: Reduktion auf das Wesentliche, klare Typografie (Plus Jakarta Sans / Satoshi), keine Emoji-Dekoration, Personality via SVG-Illustration (z.B. kleines Modul-Icon im Header) und Mikro-Interaktionen.

---

## 10. Konkreter Layout- und Interaktions-Vorschlag: Der Cosmi-Modul-Editor

### Einstieg: Modul-Galerie im Admin-Hub (nicht Teil des Editors selbst)

Im Admin-Hub → "Anpassungen" → Modul-Galerie: Karten für jedes Modul (Kontakte, Helpdesk, Work, etc.) mit Modul-Icon + Name + "Zuletzt bearbeitet: …". Klick öffnet den Editor-Modus.

### Editor-Fenster: Full-Overlay über der App

Das Editor-Fenster öffnet sich als **vollbildschirmiges Overlay** über Cosmi (kein separates Electron-Fenster, kein Browser-Tab). Animation: leichtes fade-in + leichtes scale-up (Transform only, GPU, 200ms ease-out). Der Rest der App ist dahinter sichtbar aber gedimmt (50% Opacity).

**Warum kein separates Fenster:** Klares Cosmi-Modal-Muster (konsistent mit DetailModal). Kein Window-Management-Overhead für den Nutzer.

### ASCII-Wireframe: Editor-Fenster

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  TOOLBAR (48px, Cosmi-Surface, leichte Bottom-Border)                       │
│  ← Schließen   [Modul-Icon] Kontakte — Felder & Begriffe    [Undo ↺][Redo ↻]│
│                                               [Vorschau ◉]  [Als Entwurf ▾] │
└─────────────────────────────────────────────────────────────────────────────┘
│  DRAFT-BANNER (32px, Amber/Ochre, nur sichtbar wenn Änderungen vorhanden)   │
│  "Entwurf — Diese Änderungen sind noch nicht aktiv."        [×]             │
└─────────────────────────────────────────────────────────────────────────────┘
│                                                                             │
│  LEFT PANEL (260px, zugeklappt wenn kein Add-Modus)                         │
│  ┌──────────────────────────┐                                               │
│  │ FELDER (12)              │      CANVAS (Sandbox-Rendering des Moduls)    │
│  │  ─────────────────────   │                                               │
│  │  ■ Vorname   [Text]      │    ┌──────────────────────────────────────┐  │
│  │  ■ Nachname  [Text]      │    │                                      │  │
│  │  ■ E-Mail    [Email]     │    │   [MODUL-RENDERING — Kontakte]       │  │
│  │  ■ Telefon   [Phone]     │    │                                      │  │
│  │  ■ Status    [Select]    │    │   Vorname ________________           │  │
│  │  ⊕ Feld hinzufügen       │    │                          ← HOVER:   │  │
│  │                          │    │   Nachname ______________  ┌─────┐   │  │
│  │ ─────────────────────    │    │                            │Edit │   │  │
│  │ BEGRIFFE (8)             │    │   E-Mail  ______________   └─────┘   │  │
│  │  ■ "Primäraktion" →      │    │                                      │  │
│  │    "Hinzufügen"          │    │   Status  [Aktiv ▾]                  │  │
│  │  ■ "Kontakt" →           │    │                                      │  │
│  │    "Kunde"               │    │   [SELEKTIERT: blaue Outline]        │  │
│  │  ⊕ Begriff hinzufügen    │    │   Status  [Aktiv ▾]  ◀━━━━━━━━━━━   │  │
│  │                          │    │                                      │  │
│  │ ─────────────────────    │    │                                      │  │
│  │ WERTELISTEN (3)          │    └──────────────────────────────────────┘  │
│  │  ■ Kontaktstatus         │                                               │
│  │  ■ Kontakttyp            │                                               │
│  │  ■ Region                │                                               │
│  │  ⊕ Liste hinzufügen      │                                               │
│  └──────────────────────────┘                                               │
│                                                            RIGHT PANEL      │
│                                               ┌────────────────────────┐    │
│                                               │ Feld: Status           │    │
│                                               │ ──────────────────     │    │
│                                               │ Label         [Status] │    │
│                                               │ Typ           Auswahl  │    │
│                                               │ Pflichtfeld   [ ] Ja   │    │
│                                               │ Sichtbarkeit  Immer ▾  │    │
│                                               │                        │    │
│                                               │ ▾ Weitere Optionen     │    │
│                                               │                        │    │
│                                               │ Werteliste:            │    │
│                                               │ Kontaktstatus ↗        │    │
│                                               └────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
│  FOOTER (40px)                                                              │
│  3 Änderungen nicht gespeichert              [Verwerfen]  [Übernehmen →]   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Interaktions-Flow: Schritt für Schritt

**Schritt 1 — Editor öffnen:**
Admin klickt "Bearbeiten" auf der Modul-Karte in der Admin-Hub-Galerie. Das Editor-Overlay öffnet sich mit einer fade+scale Animation (200ms). Flash-Animation: alle editierbaren Elemente in der Canvas blinken kurz auf (1.5s, subtil). Draft-Banner ist noch nicht sichtbar (keine Änderungen vorhanden).

**Schritt 2 — Element entdecken:**
Admin bewegt die Maus über das Modul-Canvas. Felder bekommen einen dünnen Outline-Rahmen in Cosmi-Akzentfarbe + Label-Chip ("Feld: Status"). Non-editable Elemente (Layout-Rahmen, Navigation-Elements): kein Hover-State, Cursor bleibt default.

**Schritt 3 — Element anklicken:**
Klick auf "Status"-Feld. Outline wird solider. Das Properties-Panel rechts öffnet sich mit einer leichten slide-in Animation (150ms). Properties-Panel zeigt: Label, Typ, Pflichtfeld-Toggle, Sichtbarkeit. "Weitere Optionen"-Akkordeon für erweiterte Einstellungen.

**Schritt 4 — Label ändern:**
Admin klickt auf das Label-Feld im Properties-Panel (oder Doppelklick direkt auf das Label im Canvas). Das Label-Input wird editierbar. Änderung von "Status" auf "Kundenstatus". Das Label im Canvas ändert sich live (via i18next-ICU-Overlay, wie in v1.2 implementiert). Draft-Banner erscheint: "Entwurf — 1 Änderung nicht übernommen."

**Schritt 5 — Werteliste editieren:**
Admin klickt im linken Panel auf "Kontaktstatus" (Werteliste). Properties-Panel wechselt zur Wertelisten-Ansicht: aktuelle Werte als sortierbare Liste, "+Wert hinzufügen"-Button. Inline-Edit jedes Werts per Doppelklick.

**Schritt 6 — Übernehmen:**
Admin klickt "Übernehmen →" im Footer. Deploy-Dialog öffnet sich (fokussiertes Modal, 480px breit):
- "Jetzt übernehmen" (Radio, default)
- "Terminiert am [Datum-Picker]"
- "Als Entwurf speichern"
Klick "Übernehmen" → Dialog schließt → Editor-Overlay zeigt kurzen Success-State ("Änderungen übernommen ✓") → schließt sich oder bleibt offen für weitere Änderungen (TBD mit Darien).

**Schritt 7 — Schließen ohne Übernehmen:**
Klick "← Schließen" oder Escape. Wenn Draft-Banner aktiv: Confirmation-Dialog "Du hast nicht übernommene Änderungen. Als Entwurf speichern oder verwerfen?" — drei Buttons: "Verwerfen", "Als Entwurf speichern", "Weiterarbeiten".

### Cosmi-Design-Differenzierung (kein Odoo-Klon)

- **Typografie im Editor-Chrome:** Plus Jakarta Sans, 14px, moderate weight. Kein Inter, kein Roboto.
- **Farben:** Cosmi-Akzentfarbe für Outlines (kein generisches Salesforce-Blau). Amber/Ochre für den Draft-Banner (nicht Rot = kein Fehler-Signal).
- **Keine Emoji-Dekoration.** Personality via das Modul-SVG-Icon im Toolbar-Header.
- **Motion:** Nur transform/opacity. Editor-Open: `scale(0.97) → scale(1)` + `opacity 0 → 1`, 200ms ease-out. Properties-Panel-Open: `translateX(20px) → translateX(0)` + `opacity 0 → 1`, 150ms ease-out. Kein Bounce, kein Spring-Overdrive.
- **Kantenradien:** Konsistent mit dem Rest von Cosmi (kein "rounded-2xl overload", kein Bootstrap-Look).
- **Kein generischer Dashboard-Charakter:** Der Editor-Toolbar hat keine Icon-Reihe à la Salesforce Setup. Nur das Nötigste: Modul-Titel, Undo/Redo, Preview-Toggle, Entwurf/Übernehmen.

---

## 11. Quellenregister (gesamt)

- https://www.odoo.com/documentation/19.0/applications/studio.html
- https://www.odoo.com/documentation/18.0/applications/studio/fields.html
- https://www.odoo.com/documentation/19.0/applications/studio/fields.html
- https://odootricks.tips/about/odoo-studio/adding-fields-using-odoo-studio/
- https://www.cybrosys.com/blog/how-to-customize-views-reports-in-odoo-18-studio
- https://www.braincuber.com/tutorial/customize-views-reports-odoo-18-studio
- https://www.odoo.com/forum/help-1/odoo-studio-cant-drag-drop-columns-or-tabs-in-form-view-v16e-220626
- https://trailhead.salesforce.com/content/learn/modules/lightning_app_builder/lightning_app_builder_intro
- https://sdlccorp.com/post/lightning-app-builder-complete-tutorial/
- https://www.salesforce.com/platform/drag-and-drop-app-builder/
- https://www.servicenow.com/products/ui-builder.html
- https://www.servicenow.com/community/next-experience-blog/inside-ui-builder-part-one/ba-p/3395615
- https://www.servicenow.com/docs/bundle/zurich-application-development/page/administer/ui-builder/concept/ui-builder-overview.html
- https://help.webflow.com/hc/en-us/articles/33961319255059-Webflow-canvas-overview
- https://help.webflow.com/hc/en-us/articles/33961320786451-Navigator
- https://webflow.com/glossary/left-tool-bar
- https://help.webflow.com/hc/en-us/articles/33961362040723-Style-panel-overview
- https://www.framer.com/updates/editor-bar
- https://www.framer.com/updates/on-page-editing
- https://www.flatlineagency.com/blog/framer-on-page-editing/
- https://www.sanity.io/docs/visual-editing/introduction-to-visual-editing
- https://www.sanity.io/docs/visual-editing/visual-editing-overlays
- https://docs.retool.com/apps/concepts/ide
- https://community.retool.com/t/early-access-widgetgrid-faster-editor-interactions/44871
- https://www.nngroup.com/articles/modes/
- https://www.saasui.design/pattern/editor
