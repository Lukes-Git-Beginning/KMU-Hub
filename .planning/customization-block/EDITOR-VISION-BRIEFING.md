# Modul-Editor („Anpassungen" Neu-Ausrichtung) — Vision & Recherche-Briefing

> **Status: VORBEREITET (Session #25, 2026-07-21) für nächstes Terminal.** Darien-Richtungs-Feedback nach v1.0–v1.2: die bisherige Umsetzung (zwei Admin-Listen: Felder + Begriffe) ist zu mager + vom Modul entkoppelt. Neue Vision = **modul-zentrischer, visueller edit-in-place-Editor**. Das ist das Odoo-Studio-Modell aus unserer Markt-Recherche, näher am USP „Massanfertigung".
>
> **★ TRIGGER: Darien sagt „weiter gehts" → RECHERCHE (nicht bauen).** Ablauf §6: Ist-Analyse (§2) + Markt/UX (§3) + Draft/Deploy (§4) parallel recherchieren → gebündelte Fragen + Design-Vorschläge (inkl. „wie soll es aussehen") → **mit Darien besprechen** → Konzept schärfen → dann erst bauen. **NICHT bauen vor Besprechung.**

## §0 Vision & Entscheidungen (Darien 2026-07-21, bestätigt)

**Dariens Worte:** „Ich hätte eher gesagt, dass man da einen kompletten Editor baut. Man hat eine Auswahl der Module und kann diese im Editor öffnen und bearbeiten. Dadurch können wir auch kontrollieren, was im Editor möglich ist, und man kann Änderungen direkt live sehen und sie am Ende übernehmen oder als Blaupause für später speichern."

**3 Entscheidungen (verbindlich):**
1. **Editor-Modell = edit-in-place, aber im EIGENEN Editor-Fenster (Sandbox, isoliert vom Live-System).** NICHT im echten Live-Modul editieren. Das Modul öffnet sich in einem separaten Editor-Fenster als bearbeitbare Kopie (edit-in-place-Gefühl: man klickt Elemente direkt an), die echte App bleibt unberührt bis zum Übernehmen. **Auswahl „welches Modul + was ist anpassbar" liegt im Admin-Hub.** (Darien-Wortlaut: „es soll sich dann nur in dem Editor öffnen und nicht direkt im Modul … vielleicht ein extra Fenster für den Editor, und die Einstellungen welches Modul sind im Admin-Hub.")
2. **Speichern = Entwurf + geplantes/terminiertes Deployment.** „Blaupause" ist als **Entwurf** gedacht (weiterbearbeiten), UND als **geplanter Rollout**: man plant einen Umbau vor und schaltet ihn an einem festen Tag für alle live — weil tenant-weite Änderungen kontrolliert ausgerollt gehören, nicht spontan. **= Change-Management für Config** (Draft → Review → geplantes Deploy an Tag X). (Darien-Wortlaut: „gedacht als Entwurf, falls man noch was ändern möchte, oder wenn man das System/den Ablauf überarbeitet, das erst plant und dann an einem bestimmten Tag deployed, weil es ja Änderungen für alle sind.")
3. **v1-Scope = Fundament-Trio im Editor-Rahmen.** Pro Modul: Felder (Custom Fields), Begriffe (Labels), Wertelisten (Value-Sets) — die schon gebaute Datenschicht, jetzt im modul-zentrischen Editor gebündelt. **Layout & Sichtbarkeit** (Felder verschieben/ausblenden, Pflicht, Spalten) = **direkt folgende Stufe** (grüne Wiese, größer — im visuellen Editor eigentlich Kern, aber Aufwand). Darien-Hinweis: Trio-vs-Layout-Unterschied war unklar → simpel: **Trio = die Bausteine (was drin ist + wie es heißt), Layout = die Anordnung (wie es aussieht + was sichtbar ist).**

## §1 Was bleibt, was wird neu

- **BLEIBT (die Maschinerie dahinter, v1.0–v1.2):** Overlay-Resolver `resolveConfig` default→vendor→tenant + Provenance · Custom-Fields-Datenschicht (`mocks/data/custom-fields.ts`, 5 Entitäten, 9 Typen) · Label-Overlay + **ICU-Live-Fix** (`i18n/i18n.ts` `bindI18nStore:'added removed'` — Overrides greifen jetzt live überall) · Value-Sets-Resolver · Audit (`customization.*`) · RBAC-Key `admin:customization:manage` · MSW-Persistenz. **Das trägt den neuen Editor 1:1.**
- **NEU / UMGEBAUT:** der Einstieg + die UX. Modul-Galerie/-Auswahl (Admin-Hub) → **Editor-Fenster** mit dem Modul als edit-in-place-Sandbox → Live-Preview → **Draft-Schicht** → Übernehmen *oder* geplantes Deploy. Die bestehenden Admin-Tab-Listen (`CustomFieldsTab`, `BegriffeTab`) werden in den Editor **integriert oder ersetzt** (Recherche §2 klärt: wiederverwenden vs. neu).
- **Value-Sets (v1.3) + Hub-Politur (v1.4) aus der alten Roadmap sind SUPERSEDED** durch diese Neu-Ausrichtung — Value-Sets fließen in den Editor ein, kein eigener Listen-Tab mehr.

## §2 Ist-Analyse-Auftrag (Machbarkeit edit-in-place — die zentrale Frage)

**Kernfrage: Kann ein Modul isoliert in einer Editor-Sandbox gerendert + editier-markiert werden, ohne das Live-System zu berühren?** Explore-Agents (Pfade sind Anhaltspunkte, verifizieren):
1. **Modul-Architektur:** Wie sind Module aufgebaut (Routing, Daten-Hooks, Stores, MSW)? Kann eine Modul-Komponente isoliert gerendert werden (eigener Fenster-Kontext, evtl. eigene Provider/QueryClient), ohne globale Stores/Routing zu stören? Beispiel-Module: kontakte (crm), work, helpdesk.
2. **Feld-/Render-Abstraktion:** Rendern Module ihre Felder/Labels hart im JSX, oder gibt es eine gemeinsame Abstraktion (Field-Registry, Detail-Section-Komponenten)? Das entscheidet, wie schwer „Elemente editier-markieren" ist. (v1.1 nutzte `CustomFieldsSection`, `ProfileSections` etc. — wie generisch?)
3. **Sandbox-Isolation + Draft-Schicht:** Wie trennt man Editor-Zustand vom Live? Idee: eine **`draft`-Overlay-Schicht** über vendor/tenant (`resolveConfig` erweitern: default→vendor→tenant→draft, draft nur im Editor-Kontext aktiv). Machbarkeit prüfen.
4. **Wiederverwendung:** Was von v1.0–v1.2 (Overlay-Resolver, custom-fields-Daten, useLabelOverlay+ICU-Fix, Value-Sets, CustomFieldsTab/BegriffeTab, FieldEditorModal) ist im Editor-Rahmen nutzbar? Was wird integriert, was ersetzt?
5. **Fenster-Mechanik:** Wie werden „Fenster" im Cosmi-Frontend gemacht (Modal/DetailModal-Muster, eigenes Overlay, neues Electron-Fenster)? Ein Editor-Fenster = großes Modal/Overlay vs. echtes zweites Fenster — was passt?
6. **Live-Preview:** Nutzt der Editor die vorhandene ICU-Overlay-Live-Mechanik (`applyLabelOverlay` + `addResourceBundle`) für die Vorschau? Wie werden Feld-/Value-Set-Änderungen live in der Sandbox sichtbar?

**Deliverable:** `IST-EDITOR.md` — Machbarkeits-Urteil je Achse (geht / geht mit Aufwand / grüne Wiese), Pfade, empfohlener technischer Ansatz für die Sandbox + Draft-Schicht.

## §3 Markt/UX-Recherche-Auftrag (WIE sieht so ein Editor gut aus)

Fokus diesmal stärker auf **UX/Visual** („wie sowas gut aussieht", Dariens Wort) — nicht nur Mechanik. Cosmi-Ästhetik = Premium/Editorial, kein generischer Dashboard-Look.
1. **Odoo Studio (detailliert, der direkte Vergleich):** edit-in-place-UX genau — wie öffnet sich das Modul, Toolbox links/rechts, Property-Panel, Feld-Drag, Inline-Rename, wie fühlt es sich an. Screenshots/Beschreibungen.
2. **Salesforce Lightning App Builder + ServiceNow UI Builder:** Modul-/Page-Editor-UX (Canvas + Komponenten-Panel + Properties). Was ist gut, was überladen.
3. **Visuelle Editoren allgemein (Webflow / Framer / Retool / Builder.io):** Wie fühlt sich erstklassiges visuelles Editing an — Canvas-Rahmen, Selektions-Outline, Property-Panel, Undo, Preview-Toggle, „Publish"-Flow. Übertragbare Interaktions-Muster.
4. **Design-Muster für Cosmi:** konkrete UI-Bausteine — Editor-Fenster-Rahmen (Chrome, Toolbar, Modul-Titel, Exit), Sandbox-Kennzeichnung („du bearbeitest — nicht live"), Selektions-/Hover-Outlines auf editierbaren Elementen, Property/Edit-Panel, Draft-Banner, Deploy-Dialog mit Terminwahl. **Wie machen wir das eigenständig-Cosmi (nicht Odoo-Klon)?**

**Deliverable:** `MARKT-EDITOR.md` — pro Produkt UX-Kern + Screenshots-Beschreibung + Abschnitt „Übertragbare UI/Interaktions-Muster + Anti-Muster für den Cosmi-Editor" + ein konkreter **Layout-/Interaktions-Vorschlag** (ASCII-Wireframe o.ä.) für das Cosmi-Editor-Fenster.

## §4 Draft- & Scheduled-Deploy-Recherche

1. **Geplantes/terminiertes Config-Deployment:** Wie machen es andere — ServiceNow Update Sets, Salesforce Change Sets, Feature-Flag-Scheduling (LaunchDarkly), staged/scheduled rollout. Mechanik + UX (Termin wählen, Vorschau, Ankündigung, Rollback).
2. **Draft-Modell:** eigene Overlay-Schicht `draft` über tenant; Übergang Draft→Live (sofort ODER geplant an Tag X). Wie technisch sauber (ein Job/Cron der den Draft an Tag X in den tenant-Layer promotet)?
3. **Governance:** wer darf deployen (RBAC), Audit des Deploys, Nutzer-Ankündigung bei tenant-weiter Änderung, Rollback auf vorherigen Stand.

**Deliverable:** `MARKT-EDITOR.md`-Abschnitt oder `DRAFT-DEPLOY.md` — Muster + empfohlenes Draft/Deploy-Modell für Cosmi.

## §5 Erwartbare Darien-Fragen (fürs Besprechen NACH der Recherche, gebündelt)
1. **Fenster-Form:** großes In-App-Overlay/Modal vs. echtes zweites Fenster — was fühlt sich richtig an?
2. **Editier-Granularität v1:** nur Klick→Panel (Feld/Begriff/Werteliste anpassen) oder schon Drag/Verschieben (= Layout, größer)?
3. **Draft/Deploy-Tiefe v1:** MVP „jetzt übernehmen" + „als Entwurf speichern" — kommt die **Terminierung** gleich in v1 oder als Stufe 2?
4. **Modul-Abdeckung v1:** alle Module editierbar, oder erst **1–2 Pilot-Module** (z.B. kontakte + helpdesk) sauber, dann ausrollen?
5. **Bestehende Editoren:** CustomFieldsTab/BegriffeTab in den neuen Editor integrieren oder ersetzen?
6. **„Kontrollieren was möglich ist":** wer/wie definiert pro Modul, was anpassbar ist — Zentria-Vendor-Ebene (was der Kunde darf) vs. technische Grenzen?

## §6 Ablauf (bei „weiter gehts")
1. **Recherche parallel:** Ist-Analyse (§2) + Markt/UX (§3) + Draft/Deploy (§4) → `IST-EDITOR.md` + `MARKT-EDITOR.md` (+ Wireframe-Vorschlag).
2. **Verdichten + gebündelte Fragen (§5) + Design-Vorschlag** (inkl. „so könnte es aussehen") an Darien.
3. **Besprechen** → Entscheide festschreiben (dieses §0 vervollständigen / `EDITOR-KONZEPT.md`).
4. **Roadmap-Schnitt** (Pilot-Modul zuerst) → Bau (Fundament der Sandbox/Draft-Schicht selbst, Editor-UI iterativ, Gates wie gehabt).

## §7 Bezug zum Gebauten
- Datenschicht-SSOT bleibt `KONZEPT.md` (Overlay-Architektur §1 gilt weiter, nur die Editor-UX §4/§5 wird neu). Fortschritt v1.0–v1.2 in `BUILD-PROGRESS.md`.
- Recherche-Historie: `IST-A/B/C.md` + `MARKT-A/B/C.md` (Odoo-Studio-Befund dort ist der Startpunkt für §3).
- Parallel geblockt: RBAC-Team-Review wartet auf Luke-Backend-Deploy (`hetzner-review-checklist.md`).
